package handler

import (
	"fmt"
	"io"
	"net/http"

	"time"
	"github.com/anchel/wechat-official-account-admin/internal/cache"
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

	type ComponentVO struct {
		model.WxOpenComponent
		AuthEventURL string `json:"auth_event_url"`
		MsgNotifyURL string `json:"msg_notify_url"`
	}

	vo := ComponentVO{
		WxOpenComponent: comp,
		AuthEventURL:    BuildPublicURL(c, "/open/component/notify"),
		MsgNotifyURL:    BuildPublicURL(c, "/open/component/$APPID$/callback"),
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
		VerifyTicket       string `json:"verify_ticket"`
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
		if req.VerifyTicket != "" {
			now := time.Now()
			comp.VerifyTicket = req.VerifyTicket
			comp.TicketUpdatedAt = &now
		}
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
		redirectURI = BuildPublicURL(c, fmt.Sprintf("/api/open/auth-callback?tenant_id=%d", tenantID))
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

// ReceiveAuthNotify 微信开放平台-授权事件接收 URL (支持 AES 密文解密)
func (h *OpenComponentHandler) ReceiveAuthNotify(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "read body failed")
		return
	}

	msgSignature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	// 查找启用的第三方平台
	var comp model.WxOpenComponent
	if err := database.DB.Where("enabled = ?", true).Order("id desc").First(&comp).Error; err == nil {
		crypt, cryptErr := service.NewWxCrypt(comp.Token, comp.EncodingAESKey, comp.ComponentAppID)
		if cryptErr == nil {
			plainXml, decryptErr := crypt.DecryptMsg(msgSignature, timestamp, nonce, body)
			if decryptErr == nil {
				logger.Info("[开放平台授权事件] 成功解密推送事件", zap.String("xml", string(plainXml)))
				_ = service.DefaultWxOpenService.HandleComponentEvent(c.Request.Context(), plainXml)
				c.String(http.StatusOK, "success")
				return
			} else {
				logger.Warn("[开放平台授权事件] 密文解密失败，尝试明文解析", zap.Error(decryptErr))
			}
		}
	}

	// 尝试明文解析
	_ = service.DefaultWxOpenService.HandleComponentEvent(c.Request.Context(), body)
	c.String(http.StatusOK, "success")
}

// ReceiveAppMsgCallback 微信开放平台-代公众号接收用户消息与事件 URL
// 重点：支持全网发布自动化接入测试（返回普通文本消息、返回Api文本消息）
func (h *OpenComponentHandler) ReceiveAppMsgCallback(c *gin.Context) {
	appid := c.Param("appid")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "read body failed")
		return
	}

	msgSignature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	// 获取第三方平台配置以进行加解密
	var comp model.WxOpenComponent
	if err := database.DB.Where("enabled = ?", true).Order("id desc").First(&comp).Error; err != nil {
		c.String(http.StatusOK, "success")
		return
	}

	crypt, err := service.NewWxCrypt(comp.Token, comp.EncodingAESKey, comp.ComponentAppID)
	if err != nil {
		c.String(http.StatusOK, "success")
		return
	}

	// 1. 解密微信推送消息
	plainXml, err := crypt.DecryptMsg(msgSignature, timestamp, nonce, body)
	if err != nil {
		logger.Warn("[代公众号消息] 解密失败", zap.Error(err), zap.String("appid", appid))
		c.String(http.StatusOK, "success")
		return
	}

	logger.Info("[代公众号消息] 解密成功", zap.String("appid", appid), zap.String("plain", string(plainXml)))

	// 2. 优先执行微信开放平台全网发布自动化测试分支！
	isCheck, checkReply, checkErr := service.DefaultWxOpenService.HandleNetworkPublishCheck(
		c.Request.Context(),
		&comp,
		plainXml,
		timestamp,
		nonce,
	)

	if isCheck {
		if checkErr != nil {
			logger.Error("[微信全网发布测试] 处理出错", zap.Error(checkErr))
			c.String(http.StatusOK, "success")
			return
		}
		// 返回测试回复 (已包含加密或 success)
		c.Data(http.StatusOK, "application/xml; charset=utf-8", checkReply)
		return
	}

	// 3. 常规业务消息与事件处理
	replyXml, err := service.HandleWxMessage(c.Request.Context(), appid, plainXml)
	if err != nil || len(replyXml) == 0 {
		c.String(http.StatusOK, "success")
		return
	}

	// 将被动回复加密后返回
	encryptedReply, err := crypt.EncryptMsg(replyXml, timestamp, nonce)
	if err != nil {
		logger.Error("[代公众号消息] 加密回复失败", zap.Error(err))
		c.String(http.StatusOK, "success")
		return
	}

	c.Data(http.StatusOK, "application/xml; charset=utf-8", encryptedReply)
}

// RequestPushTicket 主动请求微信官方立即重新推送 Ticket (无需干等10分钟)
func (h *OpenComponentHandler) RequestPushTicket(c *gin.Context) {
	tenantID := GetTenantID(c)
	var comp model.WxOpenComponent
	if err := database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp).Error; err != nil {
		Fail(c, http.StatusBadRequest, "当前租户未配置第三方平台")
		return
	}

	err := service.DefaultWxOpenService.StartPushTicket(c.Request.Context(), comp.ComponentAppID, comp.ComponentAppSecret)
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	Ok(c, gin.H{
		"message": "已向微信服务器发送立即重推 Ticket 指令，微信将在数秒内推送至您的授权事件接收 URL，请等待片刻后刷新",
	})
}

// ManualSetTicket 允许管理员手动填入/更新 ComponentVerifyTicket
func (h *OpenComponentHandler) ManualSetTicket(c *gin.Context) {
	tenantID := GetTenantID(c)
	var req struct {
		Ticket string `json:"ticket" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "ticket 不能为空")
		return
	}

	var comp model.WxOpenComponent
	if err := database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp).Error; err != nil {
		Fail(c, http.StatusBadRequest, "当前租户未配置第三方平台")
		return
	}

	now := time.Now()
	comp.VerifyTicket = req.Ticket
	comp.TicketUpdatedAt = &now
	database.DB.Save(&comp)

	// 同步 Redis
	if cache.RDB != nil {
		_ = cache.RDB.Set(c.Request.Context(), fmt.Sprintf("wx:open:ticket:%s", comp.ComponentAppID), req.Ticket, 24*time.Hour).Err()
	}

	Ok(c, gin.H{
		"message":           "VerifyTicket 手动设置成功，已立即可用于微信扫码授权与令牌换取",
		"verify_ticket":     req.Ticket,
		"ticket_updated_at": now,
	})
}
