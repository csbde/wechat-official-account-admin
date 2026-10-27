package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/anchel/wechat-official-account-admin/config"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type OpenComponentHandler struct{}

// GetComponent 获取当前租户的第三方平台配置
func (h *OpenComponentHandler) GetComponent(c *gin.Context) {
	tenantID := GetTenantID(c)
	var comp model.WxOpenComponent

	query := database.DB.Model(&model.WxOpenComponent{})
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.First(&comp).Error; err != nil {
		Ok(c, nil)
		return
	}

	host := config.GlobalConfig.Wechat.PublicHost
	if host == "" {
		host = c.Request.Host
	}

	type ComponentVO struct {
		model.WxOpenComponent
		AuthEventURL string `json:"auth_event_url"`
		MsgNotifyURL string `json:"msg_notify_url"`
	}

	vo := ComponentVO{
		WxOpenComponent: comp,
		AuthEventURL:    fmt.Sprintf("http://%s/open/component/notify", host),
		MsgNotifyURL:    fmt.Sprintf("http://%s/open/component/$APPID$/callback", host),
	}

	Ok(c, vo)
}

// SaveComponent 保存或更新第三方平台凭证
func (h *OpenComponentHandler) SaveComponent(c *gin.Context) {
	tenantID := GetTenantID(c)
	var req struct {
		Name               string `json:"name" binding:"required"`
		ComponentAppID     string `json:"component_appid" binding:"required"`
		ComponentAppSecret string `json:"component_appsecret" binding:"required"`
		Token              string `json:"token" binding:"required"`
		EncodingAESKey     string `json:"encoding_aes_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var comp model.WxOpenComponent
	err := database.DB.Where("component_app_id = ? AND tenant_id = ?", req.ComponentAppID, tenantID).First(&comp).Error
	if err != nil {
		comp = model.WxOpenComponent{
			TenantID:           tenantID,
			Name:               req.Name,
			ComponentAppID:     req.ComponentAppID,
			ComponentAppSecret: req.ComponentAppSecret,
			Token:              req.Token,
			EncodingAESKey:     req.EncodingAESKey,
			Enabled:            true,
		}
		database.DB.Create(&comp)
	} else {
		comp.Name = req.Name
		comp.ComponentAppSecret = req.ComponentAppSecret
		comp.Token = req.Token
		comp.EncodingAESKey = req.EncodingAESKey
		comp.Enabled = true
		database.DB.Save(&comp)
	}

	Ok(c, comp)
}

// GetAuthURL 生成管理员扫码授权链接
func (h *OpenComponentHandler) GetAuthURL(c *gin.Context) {
	tenantID := GetTenantID(c)
	var comp model.WxOpenComponent
	if err := database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp).Error; err != nil {
		Fail(c, http.StatusBadRequest, "当前租户尚未配置微信开放平台第三方平台")
		return
	}

	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		host := config.GlobalConfig.Wechat.PublicHost
		if host == "" {
			host = c.Request.Host
		}
		redirectURI = fmt.Sprintf("http://%s/api/open/auth-callback?tenant_id=%d", host, tenantID)
	}

	authURL, err := service.DefaultWxOpenService.BuildAuthURL(c.Request.Context(), comp.ComponentAppID, redirectURI)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取授权链接失败: "+err.Error())
		return
	}

	Ok(c, gin.H{
		"auth_url":        authURL,
		"component_appid": comp.ComponentAppID,
	})
}

// AuthCallback 处理扫码授权回调换取凭证
func (h *OpenComponentHandler) AuthCallback(c *gin.Context) {
	authCode := c.Query("auth_code")
	if authCode == "" {
		authCode = c.Query("authorization_code")
	}
	tenantID := GetTenantID(c)

	var comp model.WxOpenComponent
	if err := database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp).Error; err != nil {
		Fail(c, http.StatusBadRequest, "租户开放平台未配置")
		return
	}

	account, err := service.DefaultWxOpenService.QueryAuth(c.Request.Context(), comp.ComponentAppID, authCode, tenantID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "换取授权凭据失败: "+err.Error())
		return
	}

	Ok(c, account)
}

// ReceiveAuthNotify 微信开放平台-授权事件接收 URL
func (h *OpenComponentHandler) ReceiveAuthNotify(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "read body failed")
		return
	}

	logger.Info("Received Open Platform Auth Notify", zap.String("raw", string(body)))
	_ = service.DefaultWxOpenService.HandleComponentEvent(c.Request.Context(), body)

	c.String(http.StatusOK, "success")
}

// ReceiveAppMsgCallback 微信开放平台-代公众号接收用户消息与事件 URL
func (h *OpenComponentHandler) ReceiveAppMsgCallback(c *gin.Context) {
	appid := c.Param("appid")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "read body failed")
		return
	}

	replyXml, err := service.HandleWxMessage(c.Request.Context(), appid, body)
	if err != nil || len(replyXml) == 0 {
		c.String(http.StatusOK, "success")
		return
	}

	c.Data(http.StatusOK, "application/xml; charset=utf-8", replyXml)
}
