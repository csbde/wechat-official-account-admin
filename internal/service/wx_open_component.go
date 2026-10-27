package service

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
		"time"

	"github.com/anchel/wechat-official-account-admin/internal/cache"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type WxOpenService struct {
	http *resty.Client
}

var DefaultWxOpenService = &WxOpenService{
	http: resty.New().SetTimeout(15 * time.Second),
}

// OpenEventMessage 微信开放平台推送的 XML 报文
type OpenEventMessage struct {
	XMLName                    xml.Name `xml:"xml"`
	AppId                      string   `xml:"AppId"`
	CreateTime                 int64    `xml:"CreateTime"`
	InfoType                   string   `xml:"InfoType"`
	ComponentVerifyTicket      string   `xml:"ComponentVerifyTicket"`
	AuthorizerAppid            string   `xml:"AuthorizerAppid"`
	AuthorizationCode          string   `xml:"AuthorizationCode"`
	AuthorizationCodeExpiredTime int64  `xml:"AuthorizationCodeExpiredTime"`
	PreAuthCode                string   `xml:"PreAuthCode"`
}

// HandleComponentVerifyTicket 处理微信开放平台定时推送的 VerifyTicket 与授权变动事件
func (s *WxOpenService) HandleComponentEvent(ctx context.Context, rawXml []byte) error {
	var msg OpenEventMessage
	if err := xml.Unmarshal(rawXml, &msg); err != nil {
		return fmt.Errorf("unmarshal open xml error: %w", err)
	}

	logger.Info("Received WeChat Open Platform event",
		zap.String("app_id", msg.AppId),
		zap.String("info_type", msg.InfoType),
		zap.String("authorizer_appid", msg.AuthorizerAppid),
	)

	switch msg.InfoType {
	case "component_verify_ticket":
		// 更新 ticket
		now := time.Now()
		database.DB.Model(&model.WxOpenComponent{}).
			Where("component_app_id = ?", msg.AppId).
			Updates(map[string]interface{}{
				"verify_ticket":     msg.ComponentVerifyTicket,
				"ticket_updated_at": &now,
			})
		if cache.RDB != nil {
			_ = cache.RDB.Set(ctx, fmt.Sprintf("wx:open:ticket:%s", msg.AppId), msg.ComponentVerifyTicket, 24*time.Hour).Err()
		}

	case "authorized", "updateauthorized":
		// 授权成功或更新，若有 auth_code 可自动拉取最新授权凭据
		if msg.AuthorizationCode != "" {
			go func() {
				_, _ = s.QueryAuth(context.Background(), msg.AppId, msg.AuthorizationCode, 1)
			}()
		}

	case "unauthorized":
		// 取消授权，标记该公众号已失效
		database.DB.Model(&model.WxAccount{}).
			Where("app_id = ? AND component_app_id = ?", msg.AuthorizerAppid, msg.AppId).
			Updates(map[string]interface{}{
				"enabled": false,
			})
	}

	return nil
}

// GetComponentAccessToken 获取第三方平台的 ComponentAccessToken
func (s *WxOpenService) GetComponentAccessToken(ctx context.Context, componentAppID string) (string, error) {
	cacheKey := fmt.Sprintf("wx:open:token:%s", componentAppID)
	if cache.RDB != nil {
		val, err := cache.RDB.Get(ctx, cacheKey).Result()
		if err == nil && val != "" {
			return val, nil
		}
	}

	var comp model.WxOpenComponent
	if err := database.DB.Where("component_app_id = ? AND enabled = ?", componentAppID, true).First(&comp).Error; err != nil {
		return "", fmt.Errorf("component not configured: %s", componentAppID)
	}

	ticket := comp.VerifyTicket
	if ticket == "" && cache.RDB != nil {
		ticket, _ = cache.RDB.Get(ctx, fmt.Sprintf("wx:open:ticket:%s", componentAppID)).Result()
	}
	if ticket == "" {
		return "", fmt.Errorf("verify ticket not received yet for component: %s", componentAppID)
	}

	url := "https://api.weixin.qq.com/cgi-bin/component/api_component_token"
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]string{
		"component_appid":         comp.ComponentAppID,
		"component_appsecret":     comp.ComponentAppSecret,
		"component_verify_ticket": ticket,
	}).Post(url)

	if err != nil {
		return "", err
	}

	var res struct {
		ComponentAccessToken string `json:"component_access_token"`
		ExpiresIn            int    `json:"expires_in"`
		Errcode              int    `json:"errcode"`
		Errmsg               string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return "", err
	}
	if res.Errcode != 0 {
		return "", fmt.Errorf("get component token error: [%d] %s", res.Errcode, res.Errmsg)
	}

	if cache.RDB != nil && res.ComponentAccessToken != "" {
		ttl := time.Duration(res.ExpiresIn-300) * time.Second
		if ttl <= 0 {
			ttl = 60 * time.Second
		}
		_ = cache.RDB.Set(ctx, cacheKey, res.ComponentAccessToken, ttl).Err()
	}

	return res.ComponentAccessToken, nil
}

// GetPreAuthCode 获取预授权码
func (s *WxOpenService) GetPreAuthCode(ctx context.Context, componentAppID string) (string, error) {
	token, err := s.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/component/api_create_preauthcode?component_access_token=%s", token)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]string{
		"component_appid": componentAppID,
	}).Post(apiURL)
	if err != nil {
		return "", err
	}

	var res struct {
		PreAuthCode string `json:"pre_auth_code"`
		ExpiresIn   int    `json:"expires_in"`
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return "", err
	}
	if res.Errcode != 0 {
		return "", fmt.Errorf("get pre auth code error: [%d] %s", res.Errcode, res.Errmsg)
	}

	return res.PreAuthCode, nil
}

// BuildAuthURL 构建引导公众号管理员微信扫码授权的 URL
func (s *WxOpenService) BuildAuthURL(ctx context.Context, componentAppID, redirectURI string) (string, error) {
	preAuthCode, err := s.GetPreAuthCode(ctx, componentAppID)
	if err != nil {
		return "", err
	}

	authURL := fmt.Sprintf(
		"https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=%s&pre_auth_code=%s&redirect_uri=%s&auth_type=3",
		componentAppID,
		preAuthCode,
		url.QueryEscape(redirectURI),
	)
	return authURL, nil
}

// QueryAuth 使用扫码回调得到的 authorization_code 换取公众号授权凭据并落库
func (s *WxOpenService) QueryAuth(ctx context.Context, componentAppID, authCode string, tenantID uint) (*model.WxAccount, error) {
	token, err := s.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/component/api_query_auth?component_access_token=%s", token)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]string{
		"component_appid":    componentAppID,
		"authorization_code": authCode,
	}).Post(apiURL)

	if err != nil {
		return nil, err
	}

	var res struct {
		AuthorizationInfo struct {
			AuthorizerAppid        string `json:"authorizer_appid"`
			AuthorizerAccessToken  string `json:"authorizer_access_token"`
			ExpiresIn              int    `json:"expires_in"`
			AuthorizerRefreshToken string `json:"authorizer_refresh_token"`
			FuncInfo               []any  `json:"func_info"`
		} `json:"authorization_info"`
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}

	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}
	if res.Errcode != 0 {
		return nil, fmt.Errorf("query auth error: [%d] %s", res.Errcode, res.Errmsg)
	}

	info := res.AuthorizationInfo
	funcInfoBytes, _ := json.Marshal(info.FuncInfo)

	// 缓存 authorizer_access_token
	if cache.RDB != nil && info.AuthorizerAccessToken != "" {
		cacheKey := fmt.Sprintf("wx:authorizer:token:%s", info.AuthorizerAppid)
		ttl := time.Duration(info.ExpiresIn-300) * time.Second
		if ttl <= 0 {
			ttl = 60 * time.Second
		}
		_ = cache.RDB.Set(ctx, cacheKey, info.AuthorizerAccessToken, ttl).Err()
	}

	// 查找并同步公众号基本信息 (头像、昵称等)
	accountName := fmt.Sprintf("授权公众号_%s", info.AuthorizerAppid[:min(len(info.AuthorizerAppid), 6)])
	avatarUrl := ""
	authorizerInfo, err := s.GetAuthorizerInfo(ctx, componentAppID, info.AuthorizerAppid)
	if err == nil && authorizerInfo != nil {
		if nick, ok := authorizerInfo["nick_name"].(string); ok && nick != "" {
			accountName = nick
		}
		if head, ok := authorizerInfo["head_img"].(string); ok && head != "" {
			avatarUrl = head
		}
	}

	// 自动在当前租户下创建或更新公众号记录
	var account model.WxAccount
	err = database.DB.Where("app_id = ? AND tenant_id = ?", info.AuthorizerAppid, tenantID).First(&account).Error
	if err != nil {
		account = model.WxAccount{
			TenantID:               tenantID,
			Name:                   accountName,
			AppID:                  info.AuthorizerAppid,
			AccountType:            "authorized",
			ComponentAppID:         componentAppID,
			AuthorizerRefreshToken: info.AuthorizerRefreshToken,
			FuncInfo:               string(funcInfoBytes),
			AvatarURL:              avatarUrl,
			Enabled:                true,
			Description:            "由微信开放平台第三方授权接入托管",
		}
		database.DB.Create(&account)
	} else {
		account.Name = accountName
		account.AccountType = "authorized"
		account.ComponentAppID = componentAppID
		account.AuthorizerRefreshToken = info.AuthorizerRefreshToken
		account.FuncInfo = string(funcInfoBytes)
		if avatarUrl != "" {
			account.AvatarURL = avatarUrl
		}
		account.Enabled = true
		database.DB.Save(&account)
	}

	return &account, nil
}

// RefreshAuthorizerToken 刷新公众号的 authorizer_access_token
func (s *WxOpenService) RefreshAuthorizerToken(ctx context.Context, componentAppID, authorizerAppID, refreshToken string) (string, error) {
	cacheKey := fmt.Sprintf("wx:authorizer:token:%s", authorizerAppID)
	if cache.RDB != nil {
		val, err := cache.RDB.Get(ctx, cacheKey).Result()
		if err == nil && val != "" {
			return val, nil
		}
	}

	componentToken, err := s.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/component/api_authorizer_token?component_access_token=%s", componentToken)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]string{
		"component_appid":          componentAppID,
		"authorizer_appid":         authorizerAppID,
		"authorizer_refresh_token": refreshToken,
	}).Post(apiURL)

	if err != nil {
		return "", err
	}

	var res struct {
		AuthorizerAccessToken  string `json:"authorizer_access_token"`
		ExpiresIn              int    `json:"expires_in"`
		AuthorizerRefreshToken string `json:"authorizer_refresh_token"`
		Errcode                int    `json:"errcode"`
		Errmsg                 string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return "", err
	}
	if res.Errcode != 0 {
		return "", fmt.Errorf("refresh authorizer token error: [%d] %s", res.Errcode, res.Errmsg)
	}

	// 更新新的 refresh token
	if res.AuthorizerRefreshToken != "" {
		database.DB.Model(&model.WxAccount{}).
			Where("app_id = ?", authorizerAppID).
			Update("authorizer_refresh_token", res.AuthorizerRefreshToken)
	}

	if cache.RDB != nil && res.AuthorizerAccessToken != "" {
		ttl := time.Duration(res.ExpiresIn-300) * time.Second
		if ttl <= 0 {
			ttl = 60 * time.Second
		}
		_ = cache.RDB.Set(ctx, cacheKey, res.AuthorizerAccessToken, ttl).Err()
	}

	return res.AuthorizerAccessToken, nil
}

// GetAuthorizerInfo 获取授权公众号的基本账号信息
func (s *WxOpenService) GetAuthorizerInfo(ctx context.Context, componentAppID, authorizerAppID string) (map[string]interface{}, error) {
	token, err := s.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/component/api_get_authorizer_info?component_access_token=%s", token)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]string{
		"component_appid":  componentAppID,
		"authorizer_appid": authorizerAppID,
	}).Post(apiURL)

	if err != nil {
		return nil, err
	}

	var res struct {
		AuthorizerInfo map[string]interface{} `json:"authorizer_info"`
	}
	_ = json.Unmarshal(resp.Body(), &res)
	return res.AuthorizerInfo, nil
}
