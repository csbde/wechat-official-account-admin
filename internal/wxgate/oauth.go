package wxgate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/config"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type OAuthHandler struct {
	http *resty.Client
}

func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{
		http: resty.New().SetTimeout(12 * time.Second),
	}
}

// UserInfoSafe 安全脱敏的用户信息
type UserInfoSafe struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	City       string   `json:"city"`
	Province   string   `json:"province"`
	Country    string   `json:"country"`
	Headimgurl string   `json:"headimgurl"`
	Privilege  []string `json:"privilege,omitempty"`
	UnionID    string   `json:"unionid,omitempty"`
}

// HandleAuthorize 处理业务方微信授权请求 (/connect/oauth2/authorize 与 /wechat/auth/authorize)
func (h *OAuthHandler) HandleAuthorize(c *gin.Context) {
	appID := c.Query("appid")
	originalRedirectURI := c.Query("redirect_uri")
	responseType := c.DefaultQuery("response_type", "code")
	scope := c.DefaultQuery("scope", "snsapi_userinfo")
	state := c.Query("state")

	if appID == "" || originalRedirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errcode": 40013, "errmsg": "missing appid or redirect_uri"})
		return
	}

	// 校验公众号是否存在
	var account model.WxAccount
	if err := database.DB.Where("app_id = ? AND enabled = ?", appID, true).First(&account).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"errcode": 40013, "errmsg": "app not found or disabled: " + appID})
		return
	}

	// 创建代理会话，保存业务原始 redirect_uri 与 state
	sm := GetSessionManager()
	sessionID, err := sm.CreateSession(appID, originalRedirectURI, state, scope, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errcode": -1, "errmsg": "create session failed"})
		return
	}

	// 计算网关自己的回调地址
	gatewayCallback := h.buildGatewayCallbackURL(c, sessionID)

	// 构建前往微信官方授权页面的 URL (支持普通公众号与第三方平台代管授权)
	wechatAuthURL := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=%s&scope=%s&state=%s",
		appID,
		url.QueryEscape(gatewayCallback),
		responseType,
		scope,
		sessionID,
	)

	// 若该公众号是由微信开放平台第三方代管授权接入，必须追加 component_appid 参数
	// 否则微信会报错 10003 (redirect_uri域名与后台配置不一致)！
	if account.AccountType == "authorized" && account.ComponentAppID != "" {
		wechatAuthURL += "&component_appid=" + account.ComponentAppID
	}

	wechatAuthURL += "#wechat_redirect" 

	GetMonitor().RecordCall(appID, "/oauth2/authorize", false)

	// 重定向至微信官方授权页面
	c.Redirect(http.StatusFound, wechatAuthURL)
}

// HandleCallback 处理微信授权成功后的回调 (/connect/oauth2/callback 与 /connect/oauth2/callback/:session_id)
func (h *OAuthHandler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	sessionID := c.Param("session_id")
	if sessionID == "" {
		sessionID = c.Query("state")
	}

	if code == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errcode": 40029, "errmsg": "invalid code or session"})
		return
	}

	sm := GetSessionManager()
	session, exists := sm.GetSession(sessionID)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"errcode": 40029, "errmsg": "session expired or not found"})
		return
	}

	// 获取公众号凭据
	var account model.WxAccount
	if err := database.DB.Where("app_id = ? AND enabled = ?", session.AppID, true).First(&account).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"errcode": 40013, "errmsg": "app not found"})
		return
	}

	// 服务端向微信换取 oauth2 access_token 与 openid
	userInfoSafe, err := h.fetchUserInfo(c.Request.Context(), &account, code)
	// 无论获取用户信息是否成功，完成后均清理单次会话
	sm.DeleteSession(sessionID)

	// 拼接业务方原本的 redirect_uri 并附带 code、state 与脱敏后的 user_info
	finalRedirectURL := h.buildFinalRedirect(session.OriginalRedirectURI, code, session.OriginalState, userInfoSafe)

	GetMonitor().RecordCall(session.AppID, "/oauth2/callback", err != nil)

	// 重定向回用户的业务页面
	c.Redirect(http.StatusFound, finalRedirectURL)
}

func (h *OAuthHandler) fetchUserInfo(ctx context.Context, account *model.WxAccount, code string) (*UserInfoSafe, error) {
	var resp *resty.Response
	var err error

	// 1. 区分直连公众号与微信开放平台第三方代管公众号换取 OAuth2 令牌
	if account.AccountType == "authorized" && account.ComponentAppID != "" {
		// 第三方平台代管模式: 必须调用 /sns/oauth2/component/access_token 接口
		compToken, cErr := service.DefaultWxOpenService.GetComponentAccessToken(ctx, account.ComponentAppID)
		if cErr != nil {
			return nil, fmt.Errorf("获取 component_access_token 失败: %w", cErr)
		}
		tokenURL := fmt.Sprintf(
			"https://api.weixin.qq.com/sns/oauth2/component/access_token?appid=%s&code=%s&grant_type=authorization_code&component_appid=%s&component_access_token=%s",
			account.AppID, code, account.ComponentAppID, compToken,
		)
		resp, err = h.http.R().SetContext(ctx).Get(tokenURL)
	} else {
		// 普通直连模式: 调用原生 /sns/oauth2/access_token 接口
		tokenURL := fmt.Sprintf(
			"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
			account.AppID, account.AppSecret, code,
		)
		resp, err = h.http.R().SetContext(ctx).Get(tokenURL)
	}
	if err != nil {
		return nil, err
	}

	var tokenRes struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		OpenID       string `json:"openid"`
		Scope        string `json:"scope"`
		UnionID      string `json:"unionid"`
		Errcode      int    `json:"errcode"`
		Errmsg       string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &tokenRes); err != nil || tokenRes.Errcode != 0 {
		return nil, fmt.Errorf("exchange token error: %v, [%d] %s", err, tokenRes.Errcode, tokenRes.Errmsg)
	}

	safeUser := &UserInfoSafe{
		OpenID:  tokenRes.OpenID,
		UnionID: tokenRes.UnionID,
	}

	// 2. 如果包含 snsapi_userinfo 权限，自动拉取用户信息
	if strings.Contains(tokenRes.Scope, "snsapi_userinfo") {
		userURL := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN", tokenRes.AccessToken, tokenRes.OpenID)
		userResp, err := h.http.R().SetContext(ctx).Get(userURL)
		if err == nil {
			var u UserInfoSafe
			if err := json.Unmarshal(userResp.Body(), &u); err == nil && u.OpenID != "" {
				safeUser = &u
			}
		}
	}

	return safeUser, nil
}

func (h *OAuthHandler) buildGatewayCallbackURL(c *gin.Context, sessionID string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = strings.ToLower(proto)
	}

	rawHost := config.GlobalConfig.Wechat.PublicHost
	if rawHost == "" {
		if fwdHost := c.GetHeader("X-Forwarded-Host"); fwdHost != "" {
			rawHost = fwdHost
		} else {
			rawHost = c.Request.Host
		}
	}

	// 如果配置自带了协议头 (如 https://wechat.company.com)
	if strings.HasPrefix(rawHost, "http://") || strings.HasPrefix(rawHost, "https://") {
		u, err := url.Parse(rawHost)
		if err == nil {
			scheme = u.Scheme
			rawHost = u.Host
		}
	}

	// 微信官方对 10003 校验极其严苛：若为标准 80/443 端口，清洗掉端口
	hostOnly := rawHost
	if strings.Contains(hostOnly, ":") {
		parts := strings.Split(hostOnly, ":")
		if parts[1] == "80" || parts[1] == "443" {
			hostOnly = parts[0]
		}
	}

	return fmt.Sprintf("%s://%s/connect/oauth2/callback/%s", scheme, hostOnly, sessionID)
}

func (h *OAuthHandler) buildFinalRedirect(originalRedirect, code, state string, user *UserInfoSafe) string {
	u, err := url.Parse(originalRedirect)
	if err != nil {
		sep := "?"
		if strings.Contains(originalRedirect, "?") {
			sep = "&"
		}
		return fmt.Sprintf("%s%scode=%s&state=%s", originalRedirect, sep, code, state)
	}

	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}

	if user != nil {
		bytes, _ := json.Marshal(user)
		q.Set("user_info", string(bytes))
	}

	u.RawQuery = q.Encode()
	return u.String()
}
