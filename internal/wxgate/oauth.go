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
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"go.uber.org/zap"
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

	// 服务端向微信换取 oauth2 access_token 与用户信息
	userInfoSafe, err := h.fetchUserInfo(c.Request.Context(), &account, code, session.Scope)
	if err != nil {
		logger.Warn("[WXGate] 换取微信用户信息失败", zap.String("appid", session.AppID), zap.Error(err))

		// 如果明确要求了 snsapi_userinfo (弹窗获取完整用户信息)，但未获取到真实授权资料
		if strings.Contains(session.Scope, "snsapi_userinfo") {
			// 提供友好微信授权重试页面，附带微信官方 forcePopup=true 强制重新弹出绿色授权按钮
			retryAuthURL := fmt.Sprintf("/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&forcePopup=true",
				session.AppID,
				url.QueryEscape(session.OriginalRedirectURI),
				session.Scope,
				session.OriginalState,
			)

			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>请确认微信登录授权</title>
  <style>
    body { font-family: -apple-system, sans-serif; background: #f8fafc; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; padding: 20px; box-sizing: border-box; }
    .card { background: #fff; border-radius: 16px; padding: 32px 24px; max-width: 380px; width: 100%%; text-align: center; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.08); border: 1px solid #e2e8f0; }
    .icon { width: 56px; height: 56px; margin: 0 auto 16px; border-radius: 50%%; background: #ecfdf5; display: flex; align-items: center; justify-content: center; color: #059669; font-size: 28px; }
    h3 { margin: 0 0 8px; font-size: 18px; color: #0f172a; }
    p { margin: 0 0 24px; font-size: 13px; color: #64748b; line-height: 1.5; }
    .btn { display: block; width: 100%%; padding: 12px 0; border-radius: 10px; background: #07c160; color: #fff; text-decoration: none; font-weight: 600; font-size: 15px; border: none; cursor: pointer; }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">💬</div>
    <h3>需要您的授权同意</h3>
    <p>为保障业务系统获取完整的昵称与头像个人资料，请点击下方按钮完成微信授权确认。</p>
    <a href="%s" class="btn">前往点击微信授权按钮</a>
  </div>
</body>
</html>`, retryAuthURL)))
			return
		}
	}

	// 授权确认完成，清理单次会话
	sm.DeleteSession(sessionID)

	// 只有在真正获取到授权信息后，才带上完整的 user_info 拼装并重定向回用户的实际业务跳转地址！
	finalRedirectURL := h.buildFinalRedirect(session.OriginalRedirectURI, code, session.OriginalState, userInfoSafe)

	GetMonitor().RecordCall(session.AppID, "/oauth2/callback", err != nil)
	c.Redirect(http.StatusFound, finalRedirectURL)
}

func (h *OAuthHandler) fetchUserInfo(ctx context.Context, account *model.WxAccount, code, scope string) (*UserInfoSafe, error) {
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

	// 2. 如果要求 snsapi_userinfo 权限，必须向微信官方拉取完整的用户真实资料
	if strings.Contains(scope, "snsapi_userinfo") || strings.Contains(tokenRes.Scope, "snsapi_userinfo") {
		userURL := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN", tokenRes.AccessToken, tokenRes.OpenID)
		userResp, err := h.http.R().SetContext(ctx).Get(userURL)
		if err != nil {
			return nil, fmt.Errorf("拉取微信用户信息网络失败: %w", err)
		}

		var u struct {
			UserInfoSafe
			Errcode int    `json:"errcode"`
			Errmsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(userResp.Body(), &u); err != nil {
			return nil, fmt.Errorf("解析微信用户信息失败: %w", err)
		}
		if u.Errcode != 0 {
			return nil, fmt.Errorf("微信接口返回未授权错误: [%d] %s", u.Errcode, u.Errmsg)
		}

		if u.OpenID != "" {
			safeUser.Nickname = u.Nickname
			safeUser.Sex = u.Sex
			safeUser.City = u.City
			safeUser.Province = u.Province
			safeUser.Country = u.Country
			safeUser.Headimgurl = u.Headimgurl
			safeUser.UnionID = u.UnionID
			safeUser.Privilege = u.Privilege
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
