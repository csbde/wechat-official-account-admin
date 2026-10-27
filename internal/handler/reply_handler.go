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
	RuleType  string `json:"rule_type" binding:"required"`
	RuleName  string `json:"rule_name" binding:"required"`
	Keywords  string `json:"keywords"`
	ReplyType string `json:"reply_type" binding:"required"`
	Content   string `json:"content"`
	MediaID   string `json:"media_id"`
	Enabled   *bool  `json:"enabled"`
}

// List 获取自动回复规则列表 (租户隔离)
func (h *ReplyHandler) List(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	ruleType := c.Query("rule_type")

	query := database.DB.Where("app_id = ? AND tenant_id = ?", appid, tenantID)
	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}

	var list []model.WxAutoReply
	query.Order("id desc").Find(&list)
	Ok(c, list)
}

// Save 新增或更新自动回复规则 (租户防越权)
func (h *ReplyHandler) Save(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
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
		if err := database.DB.Where("id = ? AND app_id = ? AND tenant_id = ?", req.ID, appid, tenantID).First(&rule).Error; err != nil {
			Fail(c, http.StatusNotFound, "规则不存在或无权操作")
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
		if req.RuleType == "subscribe" || req.RuleType == "default" {
			database.DB.Where("app_id = ? AND tenant_id = ? AND rule_type = ?", appid, tenantID, req.RuleType).Delete(&model.WxAutoReply{})
		}

		rule = model.WxAutoReply{
			TenantID:  tenantID,
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

// Delete 删除规则 (防越权校验)
func (h *ReplyHandler) Delete(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	id := c.Param("id")

	res := database.DB.Where("id = ? AND app_id = ? AND tenant_id = ?", id, appid, tenantID).Delete(&model.WxAutoReply{})
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, "规则不存在或无权操作")
		return
	}
	Ok(c, gin.H{"id": id, "deleted": true})
}

// Toggle 启用/禁用规则 (防越权校验)
func (h *ReplyHandler) Toggle(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var rule model.WxAutoReply
	if err := database.DB.Where("id = ? AND app_id = ? AND tenant_id = ?", id, appid, tenantID).First(&rule).Error; err != nil {
		Fail(c, http.StatusNotFound, "规则不存在或无权操作")
		return
	}

	rule.Enabled = !rule.Enabled
	database.DB.Save(&rule)
	Ok(c, rule)
}
