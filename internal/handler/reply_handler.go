package handler

import (
	"net/http"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/gin-gonic/gin"
)

type ReplyHandler struct{}

type SaveReplyRequest struct {
	ID        uint   `json:"id"`
	RuleType  string `json:"rule_type" binding:"required"` // subscribe, default, keyword
	RuleName  string `json:"rule_name" binding:"required"`
	Keywords  string `json:"keywords"`                    // JSON string: [{"keyword":"hello","match_type":"full"}]
	ReplyType string `json:"reply_type" binding:"required"` // text, image, news
	Content   string `json:"content"`
	MediaID   string `json:"media_id"`
	Enabled   *bool  `json:"enabled"`
}

// List 获取自动回复规则列表
// @Summary 获取自动回复规则列表
// @Tags 自动回复
// @Security BearerAuth
// @Produce json
// @Router /api/autoreply [get]
func (h *ReplyHandler) List(c *gin.Context) {
	appid := GetAppID(c)
	ruleType := c.Query("rule_type")

	query := database.DB.Where("app_id = ?", appid)
	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}

	var list []model.WxAutoReply
	query.Order("id desc").Find(&list)
	Ok(c, list)
}

// Save 新增或更新自动回复规则
// @Summary 保存自动回复规则
// @Tags 自动回复
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/autoreply [post]
func (h *ReplyHandler) Save(c *gin.Context) {
	appid := GetAppID(c)
	var req SaveReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	var rule model.WxAutoReply
	if req.ID > 0 {
		if err := database.DB.Where("app_id = ? AND id = ?", appid, req.ID).First(&rule).Error; err != nil {
			Fail(c, http.StatusNotFound, "规则不存在")
			return
		}
		rule.RuleName = req.RuleName
		rule.RuleType = req.RuleType
		rule.Keywords = req.Keywords
		rule.ReplyType = req.ReplyType
		rule.Content = req.Content
		rule.MediaID = req.MediaID
		rule.Enabled = enabled
		database.DB.Save(&rule)
	} else {
		// 如果是 subscribe 或 default，先检查是否已存在
		if req.RuleType == "subscribe" || req.RuleType == "default" {
			database.DB.Where("app_id = ? AND rule_type = ?", appid, req.RuleType).Delete(&model.WxAutoReply{})
		}

		rule = model.WxAutoReply{
			AppID:     appid,
			RuleName:  req.RuleName,
			RuleType:  req.RuleType,
			Keywords:  req.Keywords,
			ReplyType: req.ReplyType,
			Content:   req.Content,
			MediaID:   req.MediaID,
			Enabled:   enabled,
		}
		database.DB.Create(&rule)
	}

	Ok(c, rule)
}

// Delete 删除规则
// @Summary 删除自动回复规则
// @Tags 自动回复
// @Security BearerAuth
// @Produce json
// @Router /api/autoreply/{id} [delete]
func (h *ReplyHandler) Delete(c *gin.Context) {
	appid := GetAppID(c)
	id := c.Param("id")
	database.DB.Where("app_id = ? AND id = ?", appid, id).Delete(&model.WxAutoReply{})
	Ok(c, gin.H{"id": id, "deleted": true})
}

// Toggle 启用/禁用规则
// @Summary 切换规则启用状态
// @Tags 自动回复
// @Security BearerAuth
// @Produce json
// @Router /api/autoreply/{id}/toggle [put]
func (h *ReplyHandler) Toggle(c *gin.Context) {
	appid := GetAppID(c)
	id := c.Param("id")

	var rule model.WxAutoReply
	if err := database.DB.Where("app_id = ? AND id = ?", appid, id).First(&rule).Error; err != nil {
		Fail(c, http.StatusNotFound, "规则不存在")
		return
	}

	rule.Enabled = !rule.Enabled
	database.DB.Save(&rule)
	Ok(c, rule)
}
