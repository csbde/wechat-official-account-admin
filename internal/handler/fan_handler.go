package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/queue"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
)

type FanHandler struct{}

// List 粉丝列表
// @Summary 粉丝用户分页查询
// @Tags 粉丝与标签
// @Security BearerAuth
// @Produce json
// @Router /api/fans [get]
func (h *FanHandler) List(c *gin.Context) {
	appid := GetAppID(c)
	keyword := c.Query("keyword")
	tagId := c.Query("tag_id")
	page, pageSize := GetPaginationParams(c)

	query := database.DB.Model(&model.WxFan{}).Where("app_id = ?", appid)

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
	query.Order("subscribe_time desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	Page(c, list, total, page, pageSize)
}

// Sync 触发后台粉丝同步
// @Summary 异步同步微信粉丝
// @Tags 粉丝与标签
// @Security BearerAuth
// @Produce json
// @Router /api/fans/sync [post]
func (h *FanHandler) Sync(c *gin.Context) {
	appid := GetAppID(c)
	if err := queue.EnqueueSyncFans(appid); err != nil {
		Fail(c, http.StatusInternalServerError, "分发异步同步任务失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"status": "queued", "message": "微信粉丝同步任务已派发至后台队列"})
}

// UpdateRemark 修改粉丝备注
// @Summary 更新粉丝备注名
// @Tags 粉丝与标签
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/fans/remark [post]
func (h *FanHandler) UpdateRemark(c *gin.Context) {
	appid := GetAppID(c)
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

	database.DB.Model(&model.WxFan{}).
		Where("app_id = ? AND open_id = ?", appid, req.OpenID).
		Update("remark", req.Remark)

	Ok(c, gin.H{"status": "updated"})
}

// GetTags 获取标签列表
// @Summary 获取微信标签列表
// @Tags 粉丝与标签
// @Security BearerAuth
// @Produce json
// @Router /api/fans/tags [get]
func (h *FanHandler) GetTags(c *gin.Context) {
	appid := GetAppID(c)
	var list []model.WxTag
	database.DB.Where("app_id = ?", appid).Order("id asc").Find(&list)
	Ok(c, list)
}

// CreateTag 新建粉丝标签
// @Summary 创建新标签
// @Tags 粉丝与标签
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/fans/tags [post]
func (h *FanHandler) CreateTag(c *gin.Context) {
	appid := GetAppID(c)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var maxTag model.WxTag
	tagID := 100
	if err := database.DB.Where("app_id = ?", appid).Order("tag_id desc").First(&maxTag).Error; err == nil {
		tagID = maxTag.TagID + 1
	}

	tag := model.WxTag{
		AppID: appid,
		TagID: tagID,
		Name:  req.Name,
		Count: 0,
	}
	database.DB.Create(&tag)

	Ok(c, tag)
}

// DeleteTag 删除粉丝标签
// @Summary 删除指定标签
// @Tags 粉丝与标签
// @Security BearerAuth
// @Produce json
// @Router /api/fans/tags/{id} [delete]
func (h *FanHandler) DeleteTag(c *gin.Context) {
	appid := GetAppID(c)
	id := c.Param("id")
	database.DB.Where("app_id = ? AND id = ?", appid, id).Delete(&model.WxTag{})
	Ok(c, gin.H{"id": id, "deleted": true})
}

// BatchTagging 批量给粉丝打/取消标签
// @Summary 批量更新粉丝标签
// @Tags 粉丝与标签
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/fans/tags/batch [post]
func (h *FanHandler) BatchTagging(c *gin.Context) {
	appid := GetAppID(c)
	var req struct {
		OpenIDs []string `json:"openids" binding:"required"`
		TagID   int      `json:"tag_id" binding:"required"`
		Action  string   `json:"action" binding:"required"` // add | remove
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	for _, openid := range req.OpenIDs {
		var fan model.WxFan
		if err := database.DB.Where("app_id = ? AND open_id = ?", appid, openid).First(&fan).Error; err == nil {
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

	// 重新计算标签人数
	var count int64
	tagStr := strconv.Itoa(req.TagID)
	database.DB.Model(&model.WxFan{}).Where("app_id = ? AND tag_ids LIKE ?", appid, "%"+tagStr+"%").Count(&count)
	database.DB.Model(&model.WxTag{}).Where("app_id = ? AND tag_id = ?", appid, req.TagID).Update("count", count)

	Ok(c, gin.H{"status": "batch_completed", "count": count})
}
