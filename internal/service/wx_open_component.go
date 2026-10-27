package service

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
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
	XMLName                      xml.Name `xml:"xml"`
	AppId                        string   `xml:"AppId"`
	CreateTime                   int64    `xml:"CreateTime"`
	InfoType                     string   `xml:"InfoType"`
	ComponentVerifyTicket        string   `xml:"ComponentVerifyTicket"`
	AuthorizerAppid              string   `xml:"AuthorizerAppid"`
	AuthorizationCode            string   `xml:"AuthorizationCode"`
	AuthorizationCodeExpiredTime int64    `xml:"AuthorizationCodeExpiredTime"`
	PreAuthCode                  string   `xml:"PreAuthCode"`
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
	originalID := ""
	qrCodeURL := ""
	signature := ""
	authorizerInfo, err := s.GetAuthorizerInfo(ctx, componentAppID, info.AuthorizerAppid)
	if err == nil && authorizerInfo != nil {
		if authorizerInfo.AuthorizerInfo.NickName != "" {
			accountName = authorizerInfo.AuthorizerInfo.NickName
		}
		if authorizerInfo.AuthorizerInfo.HeadImg != "" {
			avatarUrl = authorizerInfo.AuthorizerInfo.HeadImg
		}
		// 一并落库微信权威档案中的原始ID、二维码与功能介绍，后续离线也能直接展示
		originalID = authorizerInfo.AuthorizerInfo.UserName
		qrCodeURL = authorizerInfo.AuthorizerInfo.QRCodeURL
		signature = authorizerInfo.AuthorizerInfo.Signature
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
			OriginalID:             originalID,
			QRCodeURL:              qrCodeURL,
			Enabled:                true,
			Description:            signature,
		}
		if account.Description == "" {
			account.Description = "由微信开放平台第三方授权接入托管"
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
		if originalID != "" {
			account.OriginalID = originalID
		}
		if qrCodeURL != "" {
			account.QRCodeURL = qrCodeURL
		}
		if signature != "" && account.Description == "" {
			account.Description = signature
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

	if res.AuthorizerAccessToken != "" {
		safeTTL := time.Duration(res.ExpiresIn-300) * time.Second
		if safeTTL <= 0 {
			safeTTL = 60 * time.Second
		}
		cache.GetGateCacheManager().Set(fmt.Sprintf("client_token:%s", authorizerAppID), &cache.TokenCacheItem{
			AccessToken: res.AuthorizerAccessToken,
			ExpiresIn:   res.ExpiresIn,
			ExpireTime:  time.Now().Add(safeTTL),
			AppID:       authorizerAppID,
			CreatedAt:   time.Now(),
		}, safeTTL)

		if cache.RDB != nil {
			_ = cache.RDB.Set(ctx, cacheKey, res.AuthorizerAccessToken, safeTTL).Err()
		}
	}

	return res.AuthorizerAccessToken, nil
}

// WxAuthTypeInfo 微信账号类型 / 认证类型 (官方同时返回 id 与可读名称 name)
type WxAuthTypeInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// WxConfirmInfo 权限集的确认状态 (部分权限集需要公众号管理员确认)
type WxConfirmInfo struct {
	NeedConfirm    int `json:"need_confirm"`
	AlreadyConfirm int `json:"already_confirm"`
	CanConfirm     int `json:"can_confirm"`
}

// AuthorizerFuncInfo 授权给第三方平台的单个权限集
type AuthorizerFuncInfo struct {
	FuncscopeCategory struct {
		ID int `json:"id"`
	} `json:"funcscope_category"`
	ConfirmInfo *WxConfirmInfo `json:"confirm_info,omitempty"`
}

// AuthorizerBaseInfo 授权方基本信息 (api_get_authorizer_info 的 authorizer_info)
// 注意: 微信会在同一响应里追加视频号/小店等新字段，类型以官方文档为准
type AuthorizerBaseInfo struct {
	NickName        string                 `json:"nick_name"`
	HeadImg         string                 `json:"head_img"`
	ServiceTypeInfo WxAuthTypeInfo         `json:"service_type_info"`
	VerifyTypeInfo  WxAuthTypeInfo         `json:"verify_type_info"`
	UserName        string                 `json:"user_name"` // 原始 ID
	Alias           string                 `json:"alias"`     // 微信号 (可能为空)
	QRCodeURL       string                 `json:"qrcode_url"`
	PrincipalName   string                 `json:"principal_name"` // 主体全称
	Signature       string                 `json:"signature"`      // 功能介绍
	RegisterType    int                    `json:"register_type"`
	AccountStatus   int                    `json:"account_status"`
	BusinessInfo    map[string]interface{} `json:"business_info"`
	BasicConfig     map[string]interface{} `json:"basic_config"`
	MiniProgramInfo map[string]interface{} `json:"MiniProgramInfo,omitempty"`
	ChannelsInfo    map[string]interface{} `json:"channels_info,omitempty"`
	StoreInfo       map[string]interface{} `json:"store_info,omitempty"`
	TalentInfo      map[string]interface{} `json:"talent_info,omitempty"`
	SupplierInfo    map[string]interface{} `json:"supplier_info,omitempty"`
}

// AuthorizerFullInfo 微信官方授权账号全量详细档案
type AuthorizerFullInfo struct {
	AuthorizerInfo    AuthorizerBaseInfo     `json:"authorizer_info"`
	AuthorizationInfo AuthorizationInfo      `json:"authorization_info"`
	Raw               map[string]interface{} `json:"-"` // 微信原始响应，用于前端兜底展示
}

// AuthorizationInfo 授权信息 (authorization_info)
type AuthorizationInfo struct {
	AuthorizerAppid        string               `json:"authorizer_appid"`
	AuthorizerRefreshToken string               `json:"authorizer_refresh_token,omitempty"`
	FuncInfo               []AuthorizerFuncInfo `json:"func_info"`
}

// GetAuthorizerInfo 获取授权公众号或小程序的完整官方档案
func (s *WxOpenService) GetAuthorizerInfo(ctx context.Context, componentAppID, authorizerAppID string) (*AuthorizerFullInfo, error) {
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

	// 先按 map 解析: 保证 errcode 判断与「原始响应」兜底展示不受字段类型变化影响
	var raw map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &raw); err != nil {
		return nil, fmt.Errorf("解析授权信息响应失败: %v", err)
	}
	if code, ok := raw["errcode"].(float64); ok && int(code) != 0 {
		msg, _ := raw["errmsg"].(string)
		return nil, fmt.Errorf("[%d] %s", int(code), msg)
	}

	// 再按结构体容错解析: 微信新增或调整个别字段类型时，不影响其余档案字段的展示
	var res AuthorizerFullInfo
	_ = json.Unmarshal(resp.Body(), &res)
	res.Raw = raw

	return &res, nil
}

// HandleNetworkPublishCheck 处理微信开放平台全网发布自动化测试用例
// 返回 (是否命中测试用例, 应答密文XML或空, 错误)
func (s *WxOpenService) HandleNetworkPublishCheck(ctx context.Context, comp *model.WxOpenComponent, plainXml []byte, timestamp, nonce string) (bool, []byte, error) {
	var baseMsg WxBaseMessage
	if err := xml.Unmarshal(plainXml, &baseMsg); err != nil {
		return false, nil, err
	}

	crypt, err := NewWxCrypt(comp.Token, comp.EncodingAESKey, comp.ComponentAppID)
	if err != nil {
		return false, nil, err
	}

	content := strings.TrimSpace(baseMsg.Content)

	// =========================================================================
	// 测试用例 1: 模拟粉丝发送文本消息 (返回普通文本消息)
	// 文本内容固定为: TESTCOMPONENT_MSG_TYPE_TEXT
	// 必须被动回复内容为: TESTCOMPONENT_MSG_TYPE_TEXT_callback
	// =========================================================================
	if baseMsg.MsgType == "text" && content == "TESTCOMPONENT_MSG_TYPE_TEXT" {
		logger.Info("[微信全网发布测试] 命中测试用例1: 返回普通文本消息")
		replyText := "TESTCOMPONENT_MSG_TYPE_TEXT_callback"
		replyObj := WxTextReply{
			ToUserName:   baseMsg.FromUserName,
			FromUserName: baseMsg.ToUserName,
			CreateTime:   time.Now().Unix(),
			MsgType:      "text",
			Content:      replyText,
		}
		rawReply, _ := xml.Marshal(replyObj)
		encryptedReply, err := crypt.EncryptMsg(rawReply, timestamp, nonce)
		return true, encryptedReply, err
	}

	// =========================================================================
	// 测试用例 2: 模拟粉丝发送文本消息 (返回Api文本消息)
	// 文本内容为: QUERY_AUTH_CODE:xxxx
	// 必须直接返回空串或 success，同时异步使用 xxxx 换取 token 并调用客服接口发送 xxxx_from_api
	// =========================================================================
	if baseMsg.MsgType == "text" && strings.HasPrefix(content, "QUERY_AUTH_CODE:") {
		authCode := strings.TrimPrefix(content, "QUERY_AUTH_CODE:")
		logger.Info("[微信全网发布测试] 命中测试用例2: 返回Api文本消息", zap.String("auth_code", authCode))

		// 异步调用客服 API 发送消息
		go s.sendCustomMessageForNetworkCheck(comp.ComponentAppID, authCode, baseMsg.FromUserName)

		// 微信规定必须直接回复 success 或空
		return true, []byte("success"), nil
	}

	// =========================================================================
	// 测试用例 3: 模拟粉丝发送事件推送消息
	// 事件类型为: LOCATION 或其他，返回: 事件类型 + "from_callback"
	// =========================================================================
	if baseMsg.MsgType == "event" && baseMsg.Event != "" {
		replyText := baseMsg.Event + "from_callback"
		logger.Info("[微信全网发布测试] 命中测试用例3: 事件消息回复", zap.String("reply", replyText))
		replyObj := WxTextReply{
			ToUserName:   baseMsg.FromUserName,
			FromUserName: baseMsg.ToUserName,
			CreateTime:   time.Now().Unix(),
			MsgType:      "text",
			Content:      replyText,
		}
		rawReply, _ := xml.Marshal(replyObj)
		encryptedReply, err := crypt.EncryptMsg(rawReply, timestamp, nonce)
		return true, encryptedReply, err
	}

	return false, nil, nil
}

// 异步换取 token 并调用微信客服消息接口
func (s *WxOpenService) sendCustomMessageForNetworkCheck(componentAppID, authCode, toUser string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. 获取 component_access_token
	compToken, err := s.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		logger.Error("[微信全网发布测试] 获取 component_access_token 失败", zap.Error(err))
		return
	}

	// 2. 用 authCode 换取该测试公众号的 authorizer_access_token
	queryURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/component/api_query_auth?component_access_token=%s", compToken)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]string{
		"component_appid":    componentAppID,
		"authorization_code": authCode,
	}).Post(queryURL)
	if err != nil {
		logger.Error("[微信全网发布测试] api_query_auth 请求失败", zap.Error(err))
		return
	}

	var authRes struct {
		AuthorizationInfo struct {
			AuthorizerAppid       string `json:"authorizer_appid"`
			AuthorizerAccessToken string `json:"authorizer_access_token"`
		} `json:"authorization_info"`
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &authRes); err != nil || authRes.Errcode != 0 {
		logger.Error("[微信全网发布测试] api_query_auth 解析失败", zap.String("body", resp.String()))
		return
	}

	authorizerAccessToken := authRes.AuthorizationInfo.AuthorizerAccessToken
	logger.Info("[微信全网发布测试] 成功换取测试公众号令牌",
		zap.String("appid", authRes.AuthorizationInfo.AuthorizerAppid),
		zap.String("token", authorizerAccessToken),
	)

	// 3. 调用微信客服接口发送 content + "_from_api"
	customSendURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s", authorizerAccessToken)
	msgPayload := map[string]interface{}{
		"touser":  toUser,
		"msgtype": "text",
		"text": map[string]string{
			"content": authCode + "_from_api",
		},
	}

	cResp, cErr := s.http.R().SetContext(ctx).SetBody(msgPayload).Post(customSendURL)
	if cErr != nil {
		logger.Error("[微信全网发布测试] 客服消息发送网络错误", zap.Error(cErr))
		return
	}

	logger.Info("[微信全网发布测试] 客服消息已成功发送至微信！",
		zap.String("to_user", toUser),
		zap.String("response", cResp.String()),
	)
}

// StartPushTicket 调用微信官方接口，触发微信服务器立即向授权事件接收URL推送最新的 component_verify_ticket (无需干等10分钟)
func (s *WxOpenService) StartPushTicket(ctx context.Context, componentAppID, componentSecret string) error {
	apiURL := "https://api.weixin.qq.com/cgi-bin/component/api_start_push_ticket"
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]string{
		"component_appid":  componentAppID,
		"component_secret": componentSecret,
	}).Post(apiURL)

	if err != nil {
		return fmt.Errorf("调用微信 api_start_push_ticket 失败: %w", err)
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return fmt.Errorf("解析微信响应失败: %w", err)
	}
	if res.Errcode != 0 {
		return fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg)
	}

	logger.Info("[微信开放平台] 已成功请求微信服务器立即重新推送 Ticket", zap.String("appid", componentAppID))
	return nil
}
