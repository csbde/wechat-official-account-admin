package handler

import (
	"net/http"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
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

// GetAISetting 获取公众号 AI 配置 (兼容 ChatGPT 规范)
func (h *ReplyHandler) GetAISetting(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)

	var setting model.WxAISetting
	err := database.DB.Where("app_id = ? AND tenant_id = ?", appid, tenantID).First(&setting).Error
	if err != nil {
		// 返回默认初始结构
		Ok(c, gin.H{
			"appid":         appid,
			"enabled":       false,
			"provider":      "openai",
			"base_url":      "https://api.openai.com/v1",
			"api_key":       "",
			"model":         "gpt-4o-mini",
			"system_prompt": "你是该公众号的官方客服智能助手，请用简洁、亲切的中文回答用户咨询。",
			"temperature":   0.7,
			"max_tokens":    800,
			"match_mode":    "fallback",
			"timeout_sec":   4,
		})
		return
	}

	// 遮蔽 APIKey 部分字符
	displayKey := setting.APIKey
	if len(displayKey) > 8 {
		displayKey = displayKey[:4] + "******" + displayKey[len(displayKey)-4:]
	}

	Ok(c, gin.H{
		"id":            setting.ID,
		"appid":         setting.AppID,
		"enabled":       setting.Enabled,
		"provider":      setting.Provider,
		"base_url":      setting.BaseURL,
		"api_key":       setting.APIKey,
		"display_key":   displayKey,
		"model":         setting.Model,
		"system_prompt": setting.SystemPrompt,
		"temperature":   setting.Temperature,
		"max_tokens":    setting.MaxTokens,
		"match_mode":    setting.MatchMode,
		"timeout_sec":   setting.TimeoutSec,
	})
}

// SaveAISetting 保存或更新 AI 智能助手配置
func (h *ReplyHandler) SaveAISetting(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)

	var req struct {
		Enabled      bool    `json:"enabled"`
		Provider     string  `json:"provider"`
		BaseURL      string  `json:"base_url"`
		APIKey       string  `json:"api_key"`
		Model        string  `json:"model"`
		SystemPrompt string  `json:"system_prompt"`
		Temperature  float64 `json:"temperature"`
		MaxTokens    int     `json:"max_tokens"`
		MatchMode    string  `json:"match_mode"`
		TimeoutSec   int     `json:"timeout_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var setting model.WxAISetting
	err := database.DB.Where("app_id = ? AND tenant_id = ?", appid, tenantID).First(&setting).Error
	if err != nil {
		setting = model.WxAISetting{
			TenantID:     tenantID,
			AppID:        appid,
			Enabled:      req.Enabled,
			Provider:     req.Provider,
			BaseURL:      req.BaseURL,
			APIKey:       req.APIKey,
			Model:        req.Model,
			SystemPrompt: req.SystemPrompt,
			Temperature:  req.Temperature,
			MaxTokens:    req.MaxTokens,
			MatchMode:    req.MatchMode,
			TimeoutSec:   req.TimeoutSec,
		}
		database.DB.Create(&setting)
	} else {
		setting.Enabled = req.Enabled
		setting.Provider = req.Provider
		setting.BaseURL = req.BaseURL
		if req.APIKey != "" && req.APIKey != "******" {
			setting.APIKey = req.APIKey
		}
		setting.Model = req.Model
		setting.SystemPrompt = req.SystemPrompt
		setting.Temperature = req.Temperature
		setting.MaxTokens = req.MaxTokens
		setting.MatchMode = req.MatchMode
		if req.TimeoutSec > 0 {
			setting.TimeoutSec = req.TimeoutSec
		}
		database.DB.Save(&setting)
	}

	Ok(c, setting)
}

// TestAIConnection 测试 AI 连通性
func (h *ReplyHandler) TestAIConnection(c *gin.Context) {
	var req struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Model   string `json:"model"`
		Prompt  string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	reply, err := service.DefaultAIService.TestAIConnection(
		c.Request.Context(),
		req.BaseURL,
		req.APIKey,
		req.Model,
		req.Prompt,
	)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "AI 接口请求失败: "+err.Error())
		return
	}

	Ok(c, gin.H{
		"reply": reply,
	})
}
