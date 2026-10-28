package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/queue"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FanHandler struct{}

// wxTagBatchLimit 微信批量打/取消标签接口单次最多处理的 openid 数量 (官方限制 50)
const wxTagBatchLimit = 50

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

	if tagID, err := strconv.Atoi(strings.TrimSpace(tagId)); err == nil && tagID != 0 {
		query = query.Where(tagIDsMatchExpr(), tagIDsLikeArg(tagID))
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
			if tagID, err := strconv.Atoi(strings.TrimSpace(tagId)); err == nil && tagID != 0 {
				d = d.Where(tagIDsMatchExpr(), tagIDsLikeArg(tagID))
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
		if werr := wxClient.UpdateUserRemark(c.Request.Context(), req.OpenID, req.Remark); werr != nil {
			// 备注以微信官方为准：微信失败时不再静默写本地，避免界面显示成功而微信侧没变
			Fail(c, http.StatusBadGateway, "微信官方备注设置失败: "+service.FriendlyWxError(werr).Error())
			return
		}
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

// CreateTag 新建粉丝标签 (优先向微信官方创建以拿到真实 tag_id；失败时先在本地记录并明确标记未同步)
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
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(c, http.StatusBadRequest, "标签名称不能为空")
		return
	}

	tag := model.WxTag{TenantID: tenantID, AppID: appid, Name: name, Count: 0, Synced: true}

	// 「按标签群发」与微信侧批量打标签都依赖微信分配的真实 tag_id，因此优先向官方创建
	wechatErr := ""
	wxClient, cerr := service.NewWxClient(appid)
	if cerr != nil {
		wechatErr = cerr.Error()
	} else {
		id, werr := wxClient.CreateWxTag(c.Request.Context(), name)
		if werr != nil {
			wechatErr = service.FriendlyWxError(werr).Error()
		} else if id > 0 {
			tag.TagID = id
		} else {
			wechatErr = "微信官方未返回 tag_id"
		}
	}

	if tag.TagID > 0 {
		if err := database.DB.Create(&tag).Error; err != nil {
			Fail(c, http.StatusInternalServerError, "保存标签失败: "+err.Error())
			return
		}
		Ok(c, gin.H{"tag": tag, "synced": true, "message": fmt.Sprintf("标签已创建并同步至微信官方 (tag_id=%d)", tag.TagID)})
		return
	}

	// 微信侧创建失败：先落本地，用「负的本地主键」作占位 ID (绝不会与微信真实 tag_id 冲突)，并标记未同步
	tag.TagID = 0
	tag.Synced = false
	if err := database.DB.Create(&tag).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "保存标签失败: "+err.Error())
		return
	}
	placeholder := service.LocalOnlyFanTagID(tag.ID)
	if err := database.DB.Model(&model.WxTag{}).Where("id = ?", tag.ID).
		Updates(map[string]interface{}{"tag_id": placeholder, "synced": false}).Error; err != nil {
		Fail(c, http.StatusInternalServerError, "保存标签失败: "+err.Error())
		return
	}
	tag.TagID = placeholder

	msg := "标签已先在本地记录，但尚未同步到微信官方，暂时无法用于打标签与按标签群发"
	if wechatErr != "" {
		msg += "；微信侧原因: " + wechatErr
	}
	msg += "。可在「标签管理」中点「重新同步到微信」补齐真实 tag_id。"
	Ok(c, gin.H{"tag": tag, "synced": false, "wechat_error": wechatErr, "message": msg})
}

// PushTag 把「仅本地」标签同步到微信官方，换取真实 tag_id，并把粉丝身上的占位 ID 就地换成新 ID
func (h *FanHandler) PushTag(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var tag model.WxTag
	if err := database.DB.Where("id = ? AND app_id = ? AND tenant_id = ?", id, appid, tenantID).First(&tag).Error; err != nil {
		Fail(c, http.StatusNotFound, "标签不存在或无权操作")
		return
	}
	// force=1 时即使记录里已有正数 tag_id 也重新向微信创建：用于纠正历史遗留的"假 tag_id"
	force := c.Query("force") == "1"
	if !force && tag.Synced && tag.TagID > 0 {
		Ok(c, gin.H{"tag": tag, "synced": true, "message": fmt.Sprintf("该标签已与微信官方一致 (tag_id=%d)，无需重新同步", tag.TagID)})
		return
	}

	wxClient, err := service.NewWxClient(appid)
	if err != nil {
		Fail(c, http.StatusBadRequest, "账号不可用: "+err.Error())
		return
	}
	realID, werr := wxClient.CreateWxTag(c.Request.Context(), tag.Name)
	if werr != nil {
		Fail(c, http.StatusBadGateway, "同步到微信官方失败: "+service.FriendlyWxError(werr).Error())
		return
	}
	if realID <= 0 {
		Fail(c, http.StatusBadGateway, "微信官方未返回有效 tag_id，请稍后重试")
		return
	}

	oldID := tag.TagID
	var moved int64
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.WxTag{}).Where("id = ?", tag.ID).
			Updates(map[string]interface{}{"tag_id": realID, "synced": true}).Error; err != nil {
			return err
		}
		moved = service.RemapFanTagID(tx, appid, tenantID, oldID, realID)
		var cnt int64
		tx.Model(&model.WxFan{}).Where("app_id = ? AND tenant_id = ?", appid, tenantID).
			Where(tagIDsMatchExpr(), tagIDsLikeArg(realID)).Count(&cnt)
		return tx.Model(&model.WxTag{}).Where("id = ?", tag.ID).Update("count", cnt).Error
	}); err != nil {
		Fail(c, http.StatusInternalServerError, "同步标签失败: "+err.Error())
		return
	}

	tag.TagID = realID
	tag.Synced = true

	Ok(c, gin.H{
		"tag":           tag,
		"synced":        true,
		"migrated_fans": moved,
		"message":       fmt.Sprintf("标签已同步到微信官方 (tag_id %d → %d)，已迁移 %d 位粉丝的标签归属", oldID, realID, moved),
	})
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

	// 仅同步过微信官方 (真实正数 tag_id) 的标签才需要向微信删除；本地占位标签直接清理本地数据
	wechatErr := ""
	if tag.TagID > 0 {
		if wxClient, err := service.NewWxClient(appid); err == nil {
			if werr := wxClient.DeleteWxTag(c.Request.Context(), tag.TagID); werr != nil {
				wechatErr = service.FriendlyWxError(werr).Error()
			}
		}
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", tag.ID).Delete(&model.WxTag{}).Error; err != nil {
			return err
		}
		// 清掉粉丝身上残留的标签 ID，否则列表会一直显示 "Tag xxx" 这种无主标签
		service.RemapFanTagID(tx, appid, tenantID, tag.TagID, 0)
		return nil
	}); err != nil {
		Fail(c, http.StatusInternalServerError, "删除标签失败: "+err.Error())
		return
	}

	msg := "标签已删除"
	if tag.TagID > 0 && wechatErr == "" {
		msg = "标签已同步删除 (含微信官方)"
	} else if wechatErr != "" {
		msg = "本地标签已删除，但微信官方删除失败: " + wechatErr
	}
	Ok(c, gin.H{"id": tag.ID, "deleted": true, "wechat_error": wechatErr, "message": msg})
}

// BatchTagging 批量给粉丝打/取消标签 (租户隔离)
// 语义：微信官方同步失败时不再静默吞错，本地先写上并在响应里明确标记 synced=false + 失败原因。
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
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "add" && action != "remove" {
		Fail(c, http.StatusBadRequest, "action 只能是 add 或 remove")
		return
	}
	openIDs := uniqueStrings(req.OpenIDs)
	if len(openIDs) == 0 {
		Fail(c, http.StatusBadRequest, "请至少选择一位粉丝")
		return
	}

	// 1. 先确认标签确实存在且属于当前账号与租户 (过去直接用未校验的 tag_id，打错也提示成功)
	var tag model.WxTag
	if err := database.DB.Where("tag_id = ? AND app_id = ? AND tenant_id = ?", req.TagID, appid, tenantID).First(&tag).Error; err != nil {
		Fail(c, http.StatusBadRequest, fmt.Sprintf("标签不存在或不属于当前账号 (tag_id=%d)，请先在「标签管理」里确认或重新创建", req.TagID))
		return
	}

	// 2. 同步向微信官方执行打/取消标签；微信侧失败只记录原因，不再中断本地落库。
	//    微信批量接口是整批语义：一批 50 人里只要有一个非法 openid，整批全部失败，
	//    所以整批被拒后把「当前批 + 后续批」交给队列逐条重试，点名出真正失败的粉丝。
	wechatErr := ""
	retryTotal, retryQueued := 0, false
	if tag.Synced && tag.TagID > 0 {
		wxClient, cerr := service.NewWxClient(appid)
		if cerr != nil {
			wechatErr = cerr.Error()
			retryTotal, retryQueued = enqueueFanTagRetry(tenantID, appid, tag.TagID, action, openIDs)
		} else {
			chunks := chunkStrings(openIDs, wxTagBatchLimit)
			for i, chunk := range chunks {
				var werr error
				if action == "add" {
					werr = wxClient.BatchTaggingMembers(c.Request.Context(), tag.TagID, chunk)
				} else {
					werr = wxClient.BatchUntaggingMembers(c.Request.Context(), tag.TagID, chunk)
				}
				if werr != nil {
					wechatErr = service.FriendlyWxError(werr).Error() +
						"（常见原因：该标签已被删除或不属于当前账号、openid 不属于当前公众号、账号缺少用户标签接口权限）"
					var pending []string
					for _, rest := range chunks[i:] {
						pending = append(pending, rest...)
					}
					retryTotal, retryQueued = enqueueFanTagRetry(tenantID, appid, tag.TagID, action, pending)
					break
				}
			}
		}
	} else {
		wechatErr = fmt.Sprintf("该标签仅在本地记录 (占位 tag_id=%d)，尚未同步到微信官方", tag.TagID)
	}

	// 3. 本地一律落库 (需求：先写上并标记未同步)，逐条 Update 单列，避免整行 Save 覆盖并发字段
	localUpdated, localMissing := 0, 0
	for _, openid := range openIDs {
		var fan model.WxFan
		if err := database.DB.Where("app_id = ? AND open_id = ? AND tenant_id = ?", appid, openid, tenantID).First(&fan).Error; err != nil {
			localMissing++
			continue
		}
		ids := parseFanTagIDs(fan.TagIDs)
		next := make([]int, 0, len(ids)+1)
		has := false
		for _, v := range ids {
			if v == req.TagID {
				has = true
				if action == "add" {
					next = append(next, v)
				}
				continue
			}
			next = append(next, v)
		}
		if action == "add" && !has {
			next = append(next, req.TagID)
		}
		if err := database.DB.Model(&model.WxFan{}).Where("id = ?", fan.ID).
			Update("tag_ids", marshalFanTagIDs(next)).Error; err == nil {
			localUpdated++
		}
	}

	// 4. 精确重算该标签人数
	var count int64
	database.DB.Model(&model.WxFan{}).Where("app_id = ? AND tenant_id = ?", appid, tenantID).
		Where(tagIDsMatchExpr(), tagIDsLikeArg(req.TagID)).Count(&count)
	database.DB.Model(&model.WxTag{}).Where("id = ?", tag.ID).Update("count", count)

	synced := wechatErr == ""
	verb := "打标签"
	if action != "add" {
		verb = "取消标签"
	}
	msg := fmt.Sprintf("微信官方%s成功，本地同步更新 %d 位粉丝", verb, localUpdated)
	if !synced {
		msg = fmt.Sprintf("已先在本地记录%s (%d 位)，但微信官方未同步：%s", verb, localUpdated, wechatErr)
	}
	if retryTotal > 0 {
		if retryQueued {
			msg += fmt.Sprintf("；已派发 %d 位逐条重试 (队列执行，页面上方可见进度与失败名单)", retryTotal)
		} else {
			msg += fmt.Sprintf("；%d 位未能进入重试队列，请稍后再试", retryTotal)
		}
	}
	if localMissing > 0 {
		msg += fmt.Sprintf("；另有 %d 位粉丝在本地无档案，请先在「粉丝与标签」执行异步同步粉丝", localMissing)
	}

	Ok(c, gin.H{
		"status":        "batch_completed",
		"count":         count,
		"local_updated": localUpdated,
		"local_missing": localMissing,
		"synced":        synced,
		"wechat_error":  wechatErr,
		"retry_total":   retryTotal,
		"retry_queued":  retryQueued,
		"message":       msg,
	})
}

// GetTagRetryStatus 标签逐条重试任务进度 (Asynq 队列或内置后台协程执行)
func (h *FanHandler) GetTagRetryStatus(c *gin.Context) {
	appid := GetAppID(c)
	Ok(c, service.DefaultFanTagRetryManager.GetStatus(appid))
}

// enqueueFanTagRetry 派发标签逐条重试任务；Redis/Asynq 未启用时无缝降级为内置后台协程
// (与「异步同步粉丝」完全同一套双引擎策略)。返回 (待重试人数, 是否已派发)。
func enqueueFanTagRetry(tenantID uint, appid string, tagID int, action string, openIDs []string) (int, bool) {
	if len(openIDs) == 0 {
		return 0, false
	}
	if err := queue.EnqueueBatchTaggingRetry(queue.BatchTaggingPayload{
		TenantID: tenantID,
		AppID:    appid,
		TagID:    tagID,
		Action:   action,
		OpenIDs:  openIDs,
	}); err != nil {
		go func() {
			_ = service.DefaultFanTagRetryManager.RetryTaggingOneByOne(context.Background(), tenantID, appid, tagID, action, openIDs)
		}()
	}
	return len(openIDs), true
}

// ---------------------------- 粉丝标签内部工具 ----------------------------
// tag_ids 的存储格式、精确匹配 SQL 片段、仅本地占位 ID 等统一由 service 层维护
// (见 internal/service/wx_fan_tag.go)，这里只做别名，避免 handler 与粉丝同步两处实现漂移。

var (
	parseFanTagIDs   = service.ParseFanTagIDs
	marshalFanTagIDs = service.MarshalFanTagIDs
	tagIDsMatchExpr  = service.FanTagIDsMatchExpr
	tagIDsLikeArg    = service.FanTagIDsLikeArg
)

// chunkStrings 按 size 切分列表 (微信批量打标签单次最多 50 个 openid)
func chunkStrings(items []string, size int) [][]string {
	if size <= 0 {
		size = len(items)
	}
	out := make([][]string, 0, (len(items)+size-1)/size)
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		out = append(out, items[i:end])
	}
	return out
}

// uniqueStrings 去重并保序
func uniqueStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" || seen[it] {
			continue
		}
		seen[it] = true
		out = append(out, it)
	}
	return out
}
