package wxgate

import (
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
