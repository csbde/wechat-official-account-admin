package wxgate

import (
	"strings"
	qrcode "github.com/skip2/go-qrcode"
		"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
		"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type ProxyHandler struct {
	http *resty.Client
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		http: resty.New().SetTimeout(15 * time.Second),
	}
}

// HandleTokenRequest 代理全局 access_token 请求 (/cgi-bin/token) 带智能缓存
func (h *ProxyHandler) HandleTokenRequest(c *gin.Context) {
	grantType := c.Query("grant_type")
	appID := c.Query("appid")
	secret := c.Query("secret")

	// 兼容 POST 表单与 JSON
	if appID == "" {
		var req struct {
			GrantType string `json:"grant_type" form:"grant_type"`
			AppID     string `json:"appid" form:"appid"`
			Secret    string `json:"secret" form:"secret"`
		}
		_ = c.ShouldBind(&req)
		grantType = req.GrantType
		appID = req.AppID
		secret = req.Secret
	}

	if grantType != "client_credential" || appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errcode": 40002, "errmsg": "invalid grant_type or appid"})
		return
	}

	// 校验公众号凭证（如果未传 secret 则从数据库查找）
	var account model.WxAccount
	if err := database.DB.Where("app_id = ? AND enabled = ?", appID, true).First(&account).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"errcode": 40013, "errmsg": "appid not registered or disabled in wxgate"})
		return
	}

	if secret != "" && account.AccountType == "direct" && account.AppSecret != secret {
		c.JSON(http.StatusUnauthorized, gin.H{"errcode": 40001, "errmsg": "invalid credential, appsecret mismatch"})
		return
	}

	cm := GetCacheManager()
	token, err := cm.GetOrFetchClientAccessToken(c.Request.Context(), appID, func() (string, int, error) {
		wxClient, err := service.NewWxClient(appID)
		if err != nil {
			return "", 0, err
		}
		tok, err := wxClient.GetAccessToken(c.Request.Context())
		return tok, 7200, err
	})

	if err != nil {
		GetMonitor().RecordCall(appID, "/cgi-bin/token", true)
		c.JSON(http.StatusInternalServerError, gin.H{"errcode": -1, "errmsg": err.Error()})
		return
	}

	GetMonitor().RecordCall(appID, "/cgi-bin/token", false)

	// 按照微信官方标准数据格式输出
	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"expires_in":   7200,
	})
}

// HandleStableTokenRequest 微信稳定版 Token 接口代理 (/cgi-bin/stable_token)
func (h *ProxyHandler) HandleStableTokenRequest(c *gin.Context) {
	var req struct {
		GrantType    string `json:"grant_type"`
		AppID        string `json:"appid"`
		Secret       string `json:"secret"`
		ForceRefresh bool   `json:"force_refresh"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errcode": 40002, "errmsg": "invalid params"})
		return
	}

	// 若强制刷新，清除缓存
	if req.ForceRefresh {
		GetCacheManager().Delete("client_token:" + req.AppID)
	}

	c.Request.URL.RawQuery = fmt.Sprintf("grant_type=%s&appid=%s&secret=%s", req.GrantType, req.AppID, req.Secret)
	h.HandleTokenRequest(c)
}

// HandleGenericProxy 通用反向代理至微信服务器 (如 /sns/* 等)
func (h *ProxyHandler) HandleGenericProxy(c *gin.Context) {
	target, _ := url.Parse("https://api.weixin.qq.com")
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 记录监控
	endpoint := c.Request.URL.Path
	appID := c.Query("appid")
	GetMonitor().RecordCall(appID, endpoint, false)

	// 传递请求
	c.Request.Host = target.Host
	proxy.ServeHTTP(c.Writer, c.Request)
}

// GetGateStats 网关状态统计与监控端点
func (h *ProxyHandler) GetGateStats(c *gin.Context) {
	cacheStatus := GetCacheManager().GetStatus()
	sessionStats := GetSessionManager().GetStats()
	callStats := GetMonitor().GetStats()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"cache":    cacheStatus,
			"sessions": sessionStats,
			"monitor":  callStats,
		},
	})
}

// ClearAppCache 一键清除指定应用缓存
func (h *ProxyHandler) ClearAppCache(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		appID = c.Query("appid")
	}

	GetCacheManager().ClearAppAllCache(appID)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "应用缓存已彻底清空: " + appID,
	})
}

// GetOAuthSessionStatus 轮询检查扫码测试会话状态 (返回微信用户信息与扫码结果)
func (h *ProxyHandler) GetOAuthSessionStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errcode": 40001, "errmsg": "session_id required"})
		return
	}

	sm := GetSessionManager()
	session, exists := sm.GetSession(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"errcode": 404, "errmsg": "session expired or not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": session,
	})
}

// GenerateQRCodeImage 快速将任意 URL 渲染成高清 PNG 二维码输出
func (h *ProxyHandler) GenerateQRCodeImage(c *gin.Context) {
	content := c.Query("text")
	if content == "" {
		c.String(http.StatusBadRequest, "text required")
		return
	}

	pngBytes, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		c.String(http.StatusInternalServerError, "generate qrcode error: "+err.Error())
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-cache")
	c.Writer.Write(pngBytes)
}

// DiagnoseHealth 网关与当前公众号接口健康自检诊断
func (h *ProxyHandler) DiagnoseHealth(c *gin.Context) {
	appID := c.Query("appid")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "appid required"})
		return
	}

	var account model.WxAccount
	if err := database.DB.Where("app_id = ?", appID).First(&account).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "公众号不存在"})
		return
	}

	// 1. 获取外网出口 IP (向公共权威 IP 服务探测)
	outboundIP := "未知"
	client := resty.New().SetTimeout(3 * time.Second)
	ipResp, err := client.R().Get("https://api.ipify.org?format=text")
	if err == nil && ipResp.StatusCode() == 200 {
		outboundIP = strings.TrimSpace(ipResp.String())
	}

	// 2. 检查 Access Token 获取与连通性
	tokenStatus := "ok"
	tokenErr := ""
	wxClient, cErr := service.NewWxClient(appID)
	if cErr != nil {
		tokenStatus = "error"
		tokenErr = cErr.Error()
	} else {
		_, tErr := wxClient.GetAccessToken(c.Request.Context())
		if tErr != nil {
			tokenStatus = "error"
			tokenErr = tErr.Error()
		}
	}

	// 3. 账号类型与功能匹配自检
	typeFeatures := map[string]interface{}{
		"is_service":     account.AppType == "service" || account.ServiceType == 2,
		"is_subscribed":  account.AppType == "subscription" || account.ServiceType == 0 || account.ServiceType == 1,
		"is_miniprogram": account.AppType == "miniprogram" || account.ServiceType == 3,
		"is_verified":    account.VerifyType == 0,
		"can_oauth_userinfo": (account.AppType == "service" || account.ServiceType == 2) && account.VerifyType == 0,
		"can_template_msg":   account.AppType == "service" || account.ServiceType == 2,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"appid":         appID,
			"account_name":  account.Name,
			"app_type":      account.AppType,
			"service_type":  account.ServiceType,
			"verify_type":   account.VerifyType,
			"outbound_ip":   outboundIP,
			"token_status":  tokenStatus,
			"token_error":   tokenErr,
			"features":      typeFeatures,
		},
	})
}
