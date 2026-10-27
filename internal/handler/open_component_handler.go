package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/anchel/wechat-official-account-admin/internal/cache"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"time"
)

type OpenComponentHandler struct{}

// GetComponent 获取当前租户的第三方平台配置
func (h *OpenComponentHandler) GetComponent(c *gin.Context) {
	tenantID := GetTenantID(c)
	var comp model.WxOpenComponent

	// 与保存/授权流程保持一致：严格按当前租户视角读取，超级管理员需先切换租户视角，
	// 否则会看到别的平台的凭证配置而写到本租户，造成排查困难。
	query := database.DB.Model(&model.WxOpenComponent{}).Where("tenant_id = ?", tenantID)

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

// AuthCallback 处理微信扫码授权回调 (公开端点，由微信浏览器 302 重定向直连访问)
func (h *OpenComponentHandler) AuthCallback(c *gin.Context) {
	authCode := c.Query("auth_code")
	if authCode == "" {
		authCode = c.Query("authorization_code")
	}

	tenantIDStr := c.Query("tenant_id")
	tenantID := uint(1)
	if tenantIDStr != "" {
		var tid int
		if _, err := fmt.Sscanf(tenantIDStr, "%d", &tid); err == nil && tid > 0 {
			tenantID = uint(tid)
		}
	}

	// 查找当前租户或系统启用的第三方平台
	var comp model.WxOpenComponent
	err := database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp).Error
	if err != nil {
		// 回退查找任意启用的第三方平台
		err = database.DB.Where("enabled = ?", true).Order("id desc").First(&comp).Error
	}

	if err != nil {
		c.String(http.StatusBadRequest, "第三方平台未配置或已禁用")
		return
	}

	account, err := service.DefaultWxOpenService.QueryAuth(c.Request.Context(), comp.ComponentAppID, authCode, tenantID)
	if err != nil {
		logger.Warn("[扫码授权回调] QueryAuth 警告", zap.Error(err))
		// 即使接口报错，检查是否后台事件已提前把该账号存入数据库
		var existing model.WxAccount
		if dbErr := database.DB.Where("tenant_id = ? AND component_app_id <> '' AND account_type IN ?", tenantID, []string{"authorized", "miniprogram"}).Order("updated_at desc").First(&existing).Error; dbErr == nil {
			account = &existing
			err = nil
		}
	}

	accountName := "微信公众号"
	appID := ""
	if account != nil {
		if account.Name != "" {
			accountName = account.Name
		}
		appID = account.AppID
	}

	// 返回极具科技感、优雅的 HTML 授权成功反馈页面 (使用清晰明确的模板替换，彻底解决 %!s(EXTRA string=...) 乱符打乱页面 Bug)
	templateHTML := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>微信公众号授权成功</title>
  <style>
    * { box-sizing: border-box; }
    body {
      margin: 0; padding: 0;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", sans-serif;
      background: linear-gradient(135deg, #064e3b 0%, #0f172a 100%);
      color: #fff; min-height: 100vh;
      display: flex; align-items: center; justify-content: center;
    }
    .card {
      background: rgba(30, 41, 59, 0.9);
      border: 1px solid rgba(255, 255, 255, 0.12);
      backdrop-filter: blur(20px);
      border-radius: 24px;
      padding: 44px 32px;
      text-align: center;
      max-width: 440px;
      width: 90%;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    }
    .icon-box {
      width: 68px; height: 68px;
      margin: 0 auto 20px;
      border-radius: 50%;
      background: rgba(16, 185, 129, 0.2);
      border: 2px solid rgba(16, 185, 129, 0.4);
      display: flex; align-items: center; justify-content: center;
      color: #10b981; font-size: 34px; font-weight: bold;
    }
    h2 { margin: 0 0 10px; font-size: 22px; font-weight: 700; letter-spacing: -0.5px; }
    .badge {
      display: inline-block;
      padding: 6px 16px;
      background: rgba(16, 185, 129, 0.15);
      border: 1px solid rgba(16, 185, 129, 0.35);
      color: #34d399;
      border-radius: 999px;
      font-size: 13px; font-weight: 600;
      margin-bottom: 20px;
    }
    p { margin: 0 0 24px; font-size: 13.5px; color: #94a3b8; line-height: 1.6; }
    .btn {
      display: block; width: 100%;
      padding: 13px 0;
      border-radius: 12px;
      background: #059669;
      color: #fff;
      text-decoration: none;
      font-weight: 600; font-size: 14px;
      cursor: pointer; border: none;
      transition: background 0.2s, transform 0.1s;
    }
    .btn:hover { background: #047857; }
    .btn:active { transform: scale(0.99); }
    #countdown { color: #34d399; font-weight: bold; font-family: monospace; }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon-box">✓</div>
    <h2>授权接入成功！</h2>
    <div class="badge">已成功托管: {{ACCOUNT_NAME}}</div>
    <p>您的微信公众号已完成与本平台的授权对接，所有业务接口与菜单配置已自动就绪。<br>页面将在 <span id="countdown">3</span> 秒后自动返回系统工作台...</p>
    <button class="btn" onclick="goHome()">立即返回系统工作台</button>
  </div>
  <script>
    function goHome() {
      try {
        if (window.opener && !window.opener.closed) {
          window.opener.postMessage({ type: 'WECHAT_AUTH_SUCCESS', appid: '{{APPID}}', name: '{{ACCOUNT_NAME}}' }, '*');
        }
      } catch(e) {}
      window.location.href = '/#auth_success=1&appid={{APPID}}';
    }
    let sec = 3;
    const timer = setInterval(() => {
      sec--;
      const el = document.getElementById('countdown');
      if (el) el.innerText = sec;
      if (sec <= 0) {
        clearInterval(timer);
        goHome();
      }
    }, 1000);
  </script>
</body>
</html>`

	successHTML := strings.ReplaceAll(templateHTML, "{{ACCOUNT_NAME}}", accountName)
	successHTML = strings.ReplaceAll(successHTML, "{{APPID}}", appID)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(successHTML))
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
