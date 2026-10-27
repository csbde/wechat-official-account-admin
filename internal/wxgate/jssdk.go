package wxgate

import (
		"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type JSSDKHandler struct {
	http *resty.Client
}

func NewJSSDKHandler() *JSSDKHandler {
	return &JSSDKHandler{
		http: resty.New().SetTimeout(10 * time.Second),
	}
}

// HandleGetConfig 获取 JSSDK 网页签名配置 (/api/jssdk/config 与 /jssdk/config)
func (h *JSSDKHandler) HandleGetConfig(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		var req struct {
			URL   string `json:"url"`
			AppID string `json:"appid"`
		}
		_ = c.ShouldBindJSON(&req)
		rawURL = req.URL
		if c.Query("appid") == "" && req.AppID != "" {
			c.Request.URL.RawQuery += "&appid=" + req.AppID
		}
	}

	if rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数 url 不能为空"})
		return
	}

	// 截取 # 前面的部分
	cleanURL := strings.Split(rawURL, "#")[0]
	decodedURL, err := url.QueryUnescape(cleanURL)
	if err == nil && strings.HasPrefix(decodedURL, "http") {
		cleanURL = decodedURL
	}

	appID := c.Query("appid")
	var account model.WxAccount
	if appID != "" {
		if err := database.DB.Where("app_id = ? AND enabled = ?", appID, true).First(&account).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "未找到指定有效公众号"})
			return
		}
	} else {
		// 取默认公众号
		if err := database.DB.Where("is_default = ? AND enabled = ?", true, true).First(&account).Error; err != nil {
			if err := database.DB.Where("enabled = ?", true).First(&account).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "系统中未配置任何启用的公众号"})
				return
			}
		}
	}

	cm := GetCacheManager()
	ctx := c.Request.Context()

	// 获取/缓存全局 access_token
	clientToken, err := cm.GetOrFetchClientAccessToken(ctx, account.AppID, func() (string, int, error) {
		wxClient, err := service.NewWxClient(account.AppID)
		if err != nil {
			return "", 0, err
		}
		tok, err := wxClient.GetAccessToken(ctx)
		return tok, 7200, err
	})
	if err != nil {
		GetMonitor().RecordCall(account.AppID, "/jssdk/config", true)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 access_token 失败: " + err.Error()})
		return
	}

	// 获取/缓存 jsapi_ticket
	ticket, err := cm.GetOrFetchJsapiTicket(ctx, account.AppID, func() (string, int, error) {
		apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi", clientToken)
		resp, err := h.http.R().SetContext(ctx).Get(apiURL)
		if err != nil {
			return "", 0, err
		}
		var res struct {
			Ticket    string `json:"ticket"`
			ExpiresIn int    `json:"expires_in"`
			Errcode   int    `json:"errcode"`
			Errmsg    string `json:"errmsg"`
		}
		if err := json.Unmarshal(resp.Body(), &res); err != nil {
			return "", 0, err
		}
		if res.Errcode != 0 {
			return "", 0, fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg)
		}
		return res.Ticket, res.ExpiresIn, nil
	})

	if err != nil {
		GetMonitor().RecordCall(account.AppID, "/jssdk/config", true)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 jsapi_ticket 失败: " + err.Error()})
		return
	}

	// 生成签名
	nonceStr := fmt.Sprintf("%x", rand.Uint64())
	timestamp := time.Now().Unix()
	signStr := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", ticket, nonceStr, timestamp, cleanURL)

	hasher := sha1.New()
	hasher.Write([]byte(signStr))
	signature := hex.EncodeToString(hasher.Sum(nil))

	GetMonitor().RecordCall(account.AppID, "/jssdk/config", false)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"appId":     account.AppID,
			"timestamp": timestamp,
			"nonceStr":  nonceStr,
			"signature": signature,
			"url":       cleanURL,
			"jsApiList": []string{
				"updateAppMessageShareData",
				"updateTimelineShareData",
				"onMenuShareAppMessage",
				"onMenuShareTimeline",
				"chooseImage",
				"previewImage",
				"getLocation",
				"openLocation",
				"scanQRCode",
			},
		},
	})
}

// HandleGetTicket 原生 /cgi-bin/ticket/getticket 代理 (带自动缓存)
func (h *JSSDKHandler) HandleGetTicket(c *gin.Context) {
	token := c.Query("access_token")
	ticketType := c.DefaultQuery("type", "jsapi")

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errcode": 40001, "errmsg": "invalid access_token"})
		return
	}

	// 缓存代理直接转发或利用本地缓存
	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=%s", token, ticketType)
	resp, err := h.http.R().SetContext(c.Request.Context()).Get(apiURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errcode": -1, "errmsg": err.Error()})
		return
	}

	c.Data(resp.StatusCode(), resp.Header().Get("Content-Type"), resp.Body())
}

// HandleJSSDKScript 微信官方 jweixin.js 静态文件加速代理
func (h *JSSDKHandler) HandleJSSDKScript(c *gin.Context) {
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=86400")

	// 内存缓存微信 JS 脚本内容
	cm := GetCacheManager()
	if val, ok := cm.Get("static:jweixin_js"); ok {
		if content, ok := val.([]byte); ok {
			c.Data(http.StatusOK, "application/javascript; charset=utf-8", content)
			return
		}
	}

	// 向微信官方 CDN 获取
	resp, err := h.http.R().Get("https://res.wx.qq.com/open/js/jweixin-1.6.0.js")
	if err == nil && resp.StatusCode() == 200 {
		cm.Set("static:jweixin_js", resp.Body(), 24*time.Hour)
		c.Data(http.StatusOK, "application/javascript; charset=utf-8", resp.Body())
		return
	}

	c.String(http.StatusOK, `// fallback wechat jssdk
window.wx = window.wx || { config: function(){}, ready: function(f){ f && f(); } };`)
}
