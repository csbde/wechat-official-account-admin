package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/queue"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FanHandler struct{}

// List 粉丝列表 (租户隔离)
func (h *FanHandler) List(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	keyword := c.Query("keyword")
	tagId := c.Query("tag_id")
	page, pageSize := GetPaginationParams(c)

	query := database.DB.Model(&model.WxFan{}).Where("app_id = ? AND tenant_id = ?", appid, tenantID)

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("nickname LIKE ? OR open_id LIKE ? OR remark LIKE ?", like, like, like)
	}

	if tagId != "" {
		query = query.Where("tag_ids LIKE ?", "%"+tagId+"%")
	}

	var total int64
	query.Count(&total)

	var list []model.WxFan
	// 使用全新 Session 避免 Count 污染 Select 字段，彻底解决分页无法翻页的 Bug！
	database.DB.Model(&model.WxFan{}).Where("app_id = ? AND tenant_id = ?", appid, tenantID).
		Scopes(func(d *gorm.DB) *gorm.DB {
			if keyword != "" {
				like := "%" + keyword + "%"
				d = d.Where("nickname LIKE ? OR open_id LIKE ? OR remark LIKE ?", like, like, like)
			}
			if tagId != "" {
				d = d.Where("tag_ids LIKE ?", "%"+tagId+"%")
			}
			return d
		}).
		Order("subscribe_time desc, id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list)

	Page(c, list, total, page, pageSize)
}

// Sync 触发全量粉丝同步 (支持 Asynq 队列与内置后台协程双引擎)
func (h *FanHandler) Sync(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)

	// 检查当前是否已经在运行中
	status := service.DefaultFanSyncManager.GetStatus(appid)
	if status.Status == "running" {
		Ok(c, gin.H{
			"status":  "running",
			"message": "同步任务已经在后台执行中，请查看上方进度条",
		})
		return
	}

	// 1. 尝试派发到 Asynq 分布式队列
	err := queue.EnqueueSyncFans(tenantID, appid)
	if err != nil {
		// 2. 若 Redis/Asynq 未启用，自动无缝降级为内置后台异步协程处理 (单机 SQLite 模式)
		go func() {
			_ = service.DefaultFanSyncManager.DoSyncFansTask(context.Background(), tenantID, appid)
		}()
	}

	Ok(c, gin.H{
		"status":  "started",
		"message": "微信粉丝同步任务已成功启动，正在后台全力拉取中",
	})
}

// GetSyncStatus 获取当前公众号的粉丝同步执行进度
func (h *FanHandler) GetSyncStatus(c *gin.Context) {
	appid := GetAppID(c)
	status := service.DefaultFanSyncManager.GetStatus(appid)
	Ok(c, status)
}

// UpdateRemark 修改粉丝备注 (租户隔离)
func (h *FanHandler) UpdateRemark(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	var req struct {
		OpenID string `json:"openid" binding:"required"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	wxClient, err := service.NewWxClient(appid)
	if err == nil {
		_ = wxClient.UpdateUserRemark(c.Request.Context(), req.OpenID, req.Remark)
	}

	res := database.DB.Model(&model.WxFan{}).
		Where("app_id = ? AND open_id = ? AND tenant_id = ?", appid, req.OpenID, tenantID).
		Update("remark", req.Remark)

	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, "粉丝不存在或无权操作")
		return
	}

	Ok(c, gin.H{"status": "updated"})
}

// GetTags 获取标签列表 (租户隔离)
func (h *FanHandler) GetTags(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	var list []model.WxTag
	database.DB.Where("app_id = ? AND tenant_id = ?", appid, tenantID).Order("id asc").Find(&list)
	Ok(c, list)
}

// CreateTag 新建粉丝标签 (优先向微信官方创建以拿到真实 tag_id，失败时退化为本地标签并明确提示)
func (h *FanHandler) CreateTag(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// 「按标签群发」与微信侧批量打标签都依赖微信分配的真实 tag_id，因此优先向官方创建
	tagID := 0
	synced := false
	if wxClient, err := service.NewWxClient(appid); err == nil {
		if id, werr := wxClient.CreateWxTag(c.Request.Context(), req.Name); werr == nil && id > 0 {
			tagID = id
			synced = true
		}
	}
	if !synced {
		var maxTag model.WxTag
		tagID = 100
		if err := database.DB.Where("app_id = ? AND tenant_id = ?", appid, tenantID).Order("tag_id desc").First(&maxTag).Error; err == nil {
			tagID = maxTag.TagID + 1
		}
	}

	tag := model.WxTag{
		TenantID: tenantID,
		AppID:    appid,
		TagID:    tagID,
		Name:     req.Name,
		Count:    0,
	}
	database.DB.Create(&tag)

	msg := "标签已创建并同步至微信官方"
	if !synced {
		msg = "标签仅在本地创建 (微信官方创建失败)，按标签群发需要真实 tag_id，请检查账号凭据后重试"
	}
	Ok(c, gin.H{"tag": tag, "synced": synced, "message": msg})
}

// DeleteTag 删除粉丝标签 (防越权校验)
func (h *FanHandler) DeleteTag(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var tag model.WxTag
	if err := database.DB.Where("id = ? AND app_id = ? AND tenant_id = ?", id, appid, tenantID).First(&tag).Error; err != nil {
		Fail(c, http.StatusNotFound, "标签不存在或无权操作")
		return
	}

	// 同步向微信官方删除
	wxClient, err := service.NewWxClient(appid)
	if err == nil && tag.TagID > 0 {
		_ = wxClient.DeleteWxTag(c.Request.Context(), tag.TagID)
	}

	database.DB.Delete(&tag)
	Ok(c, gin.H{"id": id, "deleted": true})
}

// BatchTagging 批量给粉丝打/取消标签 (租户隔离)
func (h *FanHandler) BatchTagging(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	var req struct {
		OpenIDs []string `json:"openids" binding:"required"`
		TagID   int      `json:"tag_id" binding:"required"`
		Action  string   `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// 1. 同步向微信官方接口执行打标签或取消标签
	wxClient, err := service.NewWxClient(appid)
	if err == nil {
		if req.Action == "add" {
			_ = wxClient.BatchTaggingMembers(c.Request.Context(), req.TagID, req.OpenIDs)
		} else {
			_ = wxClient.BatchUntaggingMembers(c.Request.Context(), req.TagID, req.OpenIDs)
		}
	}

	// 2. 本地数据库落库同步更新
	for _, openid := range req.OpenIDs {
		var fan model.WxFan
		if err := database.DB.Where("app_id = ? AND open_id = ? AND tenant_id = ?", appid, openid, tenantID).First(&fan).Error; err == nil {
			var tags []int
			_ = json.Unmarshal([]byte(fan.TagIDs), &tags)
			newTags := make([]int, 0)
			has := false
			for _, t := range tags {
				if t == req.TagID {
					has = true
					if req.Action == "add" {
						newTags = append(newTags, t)
					}
				} else {
					newTags = append(newTags, t)
				}
			}
			if req.Action == "add" && !has {
				newTags = append(newTags, req.TagID)
			}
			data, _ := json.Marshal(newTags)
			fan.TagIDs = string(data)
			database.DB.Save(&fan)
		}
	}

	var count int64
	tagStr := strconv.Itoa(req.TagID)
	database.DB.Model(&model.WxFan{}).Where("app_id = ? AND tenant_id = ? AND tag_ids LIKE ?", appid, tenantID, "%"+tagStr+"%").Count(&count)
	database.DB.Model(&model.WxTag{}).Where("app_id = ? AND tenant_id = ? AND tag_id = ?", appid, tenantID, req.TagID).Update("count", count)

	Ok(c, gin.H{"status": "batch_completed", "count": count})
}
