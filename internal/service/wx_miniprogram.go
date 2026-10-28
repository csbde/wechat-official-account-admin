package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/cache"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type WxMiniService struct {
	http *resty.Client
}

var DefaultWxMiniService = &WxMiniService{
	http: resty.New().SetTimeout(15 * time.Second),
}

type FastRegisterParam struct {
	Name               string `json:"name"`                 // 企业名
	Code               string `json:"code"`                 // 企业代码
	CodeType           int    `json:"code_type"`            // 1: 统一社会信用代码
	LegalPersonaWechat string `json:"legal_persona_wechat"` // 法人微信号
	LegalPersonaName   string `json:"legal_persona_name"`   // 法人姓名
	ComponentPhone     string `json:"component_phone"`      // 第三方联系电话
}

// CreateFastRegisterTask 提交快速注册小程序任务 (免 300 元认证费，企业法人人脸核身方式)
func (s *WxMiniService) CreateFastRegisterTask(ctx context.Context, componentAppID string, p FastRegisterParam) error {
	p.Name = strings.TrimSpace(p.Name)
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	p.LegalPersonaWechat = strings.TrimSpace(p.LegalPersonaWechat)
	p.LegalPersonaName = strings.TrimSpace(p.LegalPersonaName)
	p.ComponentPhone = strings.TrimSpace(p.ComponentPhone)

	compToken, err := DefaultWxOpenService.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return fmt.Errorf("获取 component_access_token 失败: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/component/fastregisterweapp?action=create&component_access_token=%s", compToken)
	resp, err := s.http.R().SetContext(ctx).SetBody(p).Post(apiURL)
	if err != nil {
		return fmt.Errorf("调用微信快速注册接口失败: %w", err)
	}

	logger.Info("[快速注册小程序] 微信返回结果",
		zap.String("status", resp.Status()),
		zap.String("body", string(resp.Body())),
		zap.String("enterprise", p.Name),
		zap.String("legal_wechat", p.LegalPersonaWechat),
	)

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if res.Errcode != 0 {
		if res.Errcode == -1 {
			return fmt.Errorf("微信接口返回 [-1] 系统繁忙 (system error) — 请检查开放平台控制台是否已添加并全网发布小程序开发/认证相关权限集，并确认法人姓名、微信号与企业统一信用代码完全无误")
		}
		return FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	logger.Info("[快速注册小程序] 任务已成功提交至微信服务器", zap.String("enterprise", p.Name), zap.String("legal_wechat", p.LegalPersonaWechat))
	return nil
}

// FastRegisterByOfficialAccountResult 复用公众号主体快速注册小程序接口返回
type FastRegisterByOfficialAccountResult struct {
	AppID             string `json:"appid"`              // 新创建小程序的 appid
	AuthorizationCode string `json:"authorization_code"` // 新创建小程序的授权码 (用于换取 refresh_token 落库)
	IsWxVerifySucc    bool   `json:"is_wx_verify_succ"`  // 复用公众号微信认证是否成功
	IsLinkSucc        bool   `json:"is_link_succ"`       // 小程序是否和公众号关联成功
}

// BuildFastRegisterOfficialAuthURL 构建复用公众号主体快速注册小程序的微信扫码授权链接
// 用户点击或公众号管理员扫码该链接即可完成授权确认并回跳第三方平台
func (s *WxMiniService) BuildFastRegisterOfficialAuthURL(componentAppID, mpAppID, redirectURI string, copyWxVerify bool) string {
	copyVal := "1"
	if !copyWxVerify {
		copyVal = "0"
	}
	encodedRedirect := url.QueryEscape(redirectURI)
	return fmt.Sprintf(
		"https://mp.weixin.qq.com/cgi-bin/fastregisterauth?appid=%s&component_appid=%s&copy_wx_verify=%s&redirect_uri=%s",
		mpAppID, componentAppID, copyVal, encodedRedirect,
	)
}

// FastRegisterByOfficialAccount 复用公众号主体快速注册小程序
// 微信接口文档: POST https://api.weixin.qq.com/cgi-bin/account/fastregister?access_token=TOKEN
func (s *WxMiniService) FastRegisterByOfficialAccount(ctx context.Context, mpAppID, ticket string) (*FastRegisterByOfficialAccountResult, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return nil, fmt.Errorf("ticket 不能为空")
	}

	wxClient, err := NewWxClient(mpAppID)
	if err != nil {
		return nil, fmt.Errorf("初始化公众号客户端失败: %w", err)
	}
	authorizerToken, err := wxClient.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取公众号 authorizer_access_token 失败: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/account/fastregister?access_token=%s", authorizerToken)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]string{
		"ticket": ticket,
	}).Post(apiURL)

	if err != nil {
		return nil, fmt.Errorf("调用复用公众号快速注册接口失败: %w", err)
	}

	logger.Info("[复用公众号注册小程序] 微信响应", zap.String("body", string(resp.Body())), zap.String("mp_appid", mpAppID))

	var res struct {
		Errcode           int    `json:"errcode"`
		Errmsg            string `json:"errmsg"`
		AppID             string `json:"appid"`
		AuthorizationCode string `json:"authorization_code"`
		IsWxVerifySucc    bool   `json:"is_wx_verify_succ"`
		IsLinkSucc        bool   `json:"is_link_succ"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, fmt.Errorf("解析微信响应失败: %w", err)
	}

	if res.Errcode != 0 {
		return nil, FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return &FastRegisterByOfficialAccountResult{
		AppID:             res.AppID,
		AuthorizationCode: res.AuthorizationCode,
		IsWxVerifySucc:    res.IsWxVerifySucc,
		IsLinkSucc:        res.IsLinkSucc,
	}, nil
}

type FastRegisterSearchResult struct {
	Status       int    `json:"status"` // 0: 审核中/待人脸, 1: 成功, 2: 驳回
	StatusDesc   string `json:"status_desc"`
	AppID        string `json:"appid,omitempty"`
	RejectReason string `json:"reject_reason,omitempty"`
}

// SearchFastRegisterTask 查询快速注册小程序任务审核状态
func (s *WxMiniService) SearchFastRegisterTask(ctx context.Context, componentAppID string, name, legalPersonaWechat, legalPersonaName string) (*FastRegisterSearchResult, error) {
	compToken, err := DefaultWxOpenService.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return nil, fmt.Errorf("获取 component_access_token 失败: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/component/fastregisterweapp?action=search&component_access_token=%s", compToken)
	bodyParam := map[string]string{
		"name":                 name,
		"legal_persona_wechat": legalPersonaWechat,
		"legal_persona_name":   legalPersonaName,
	}

	resp, err := s.http.R().SetContext(ctx).SetBody(bodyParam).Post(apiURL)
	if err != nil {
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 微信官方状态码说明:
	// 0: 成功
	// 89247: 该任务正在进行中 (等待法人人脸核身)
	// 89248: 该任务已被驳回
	result := &FastRegisterSearchResult{}

	switch res.Errcode {
	case 0:
		result.Status = 1
		result.StatusDesc = "注册成功"
		// 尝试提取微信附带返回的信息
		var ext struct {
			AppID string `json:"appid"`
		}
		_ = json.Unmarshal(resp.Body(), &ext)
		result.AppID = ext.AppID

	case 89247:
		result.Status = 0
		result.StatusDesc = "任务正在进行中，微信已下发人脸核验通知，等待法人微信确认"

	case 89248:
		result.Status = 2
		result.StatusDesc = "微信审核已驳回: " + res.Errmsg
		result.RejectReason = res.Errmsg

	default:
		return nil, FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return result, nil
}

// ModifyServerDomain 代小程序配置服务器合法域名 (request, socket, upload, download)
func (s *WxMiniService) ModifyServerDomain(ctx context.Context, authorizerAppID string, action string, requestDomain, wsDomain, uploadDomain, downloadDomain []string) error {
	wxClient, err := NewWxClient(authorizerAppID)
	if err != nil {
		return err
	}
	token, err := wxClient.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/wxa/modify_domain?access_token=%s", token)
	payload := map[string]interface{}{
		"action":          action, // "set", "get", "add", "delete"
		"requestdomain":   requestDomain,
		"wsrequestdomain": wsDomain,
		"uploaddomain":    uploadDomain,
		"downloaddomain":  downloadDomain,
	}

	resp, err := s.http.R().SetContext(ctx).SetBody(payload).Post(apiURL)
	if err != nil {
		return err
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return err
	}
	if res.Errcode != 0 {
		return FriendlyWxError(fmt.Errorf("modify domain error: [%d] %s", res.Errcode, res.Errmsg))
	}

	return nil
}

// Code2Session 小程序登录凭证校验 (免密换取 openid 和 session_key)
func (s *WxMiniService) Code2Session(ctx context.Context, componentAppID, miniAppID, jsCode string) (map[string]interface{}, error) {
	compToken, err := DefaultWxOpenService.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/component/jscode2session?appid=%s&js_code=%s&grant_type=authorization_code&component_appid=%s&component_access_token=%s",
		miniAppID, jsCode, componentAppID, compToken,
	)

	resp, err := s.http.R().SetContext(ctx).Get(apiURL)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}

	if codeFloat, ok := res["errcode"].(float64); ok && int(codeFloat) != 0 {
		return nil, fmt.Errorf("[%v] %v", res["errcode"], res["errmsg"])
	}

	return res, nil
}

// LinkedMiniProgramItem 公众号关联的小程序详情
type LinkedMiniProgramItem struct {
	Status              int    `json:"status"`   // 1: 已关联, 2: 待小程序管理员确认, 3: 小程序管理员拒绝, 12: 待公众号管理员确认
	Username            string `json:"username"` // 小程序原始 ID (gh_xxx)
	Nickname            string `json:"nickname"` // 小程序昵称
	Selected            int    `json:"selected"` // 是否在公众号资料页展示 0:否, 1:是
	NearbyDisplayStatus int    `json:"nearby_display_status"`
	AppID               string `json:"appid"`
	HeadImgURL          string `json:"headimg_url"`
	Email               string `json:"email"`
	CopyVerifyStatus    int    `json:"copy_verify_status"`
}

// GetLinkedMiniPrograms 获取公众号关联的小程序列表 (cgi-bin/wxopen/wxamplinkget)
func (s *WxMiniService) GetLinkedMiniPrograms(ctx context.Context, officialAppID string) ([]LinkedMiniProgramItem, error) {
	wxClient, err := NewWxClient(officialAppID)
	if err != nil {
		return nil, err
	}
	token, err := wxClient.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/wxopen/wxamplinkget?access_token=%s", token)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]interface{}{}).Post(apiURL)
	if err != nil {
		return nil, fmt.Errorf("请求关联小程序列表网络错误: %w", err)
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
		WxOpens struct {
			Items []LinkedMiniProgramItem `json:"items"`
		} `json:"wxopens"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, fmt.Errorf("解析微信关联小程序响应失败: %w", err)
	}
	if res.Errcode != 0 {
		return nil, FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return res.WxOpens.Items, nil
}

// LinkedMiniProgramsCached 带缓存的关联小程序列表 (默认 5 分钟)。
// 「代码发布管理」等页面会高频读取该关系，缓存后可显著降低对微信接口的调用压力，
// 关联关系变更(Link/Unlink)时会主动失效，保证一致性。
func (s *WxMiniService) LinkedMiniProgramsCached(ctx context.Context, officialAppID string, ttl time.Duration) ([]LinkedMiniProgramItem, error) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	cacheKey := "linked_mini:" + officialAppID
	cm := cache.GetGateCacheManager()
	if v, ok := cm.Get(cacheKey); ok {
		if items, ok := v.([]LinkedMiniProgramItem); ok {
			return items, nil
		}
	}
	items, err := s.GetLinkedMiniPrograms(ctx, officialAppID)
	if err != nil {
		return nil, err
	}
	cm.Set(cacheKey, items, ttl)
	return items, nil
}

// InvalidateLinkedMiniProgramsCache 清除某公众号的关联小程序缓存
func (s *WxMiniService) InvalidateLinkedMiniProgramsCache(officialAppID string) {
	cache.GetGateCacheManager().Delete("linked_mini:" + officialAppID)
}

// LinkMiniProgram 为公众号关联小程序 (cgi-bin/wxopen/wxamplink)
func (s *WxMiniService) LinkMiniProgram(ctx context.Context, officialAppID, miniAppID string, notifyUsers, showProfile int) error {
	wxClient, err := NewWxClient(officialAppID)
	if err != nil {
		return err
	}
	token, err := wxClient.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/wxopen/wxamplink?access_token=%s", token)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"appid":        miniAppID,
		"notify_users": notifyUsers, // 0:不发, 1:发
		"show_profile": showProfile, // 0:不在资料页展示, 1:展示
	}).Post(apiURL)
	if err != nil {
		return err
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return err
	}
	if res.Errcode != 0 {
		return FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return nil
}

// UnlinkMiniProgram 解绑小程序 (cgi-bin/wxopen/wxampunlink)
func (s *WxMiniService) UnlinkMiniProgram(ctx context.Context, officialAppID, miniAppID string) error {
	wxClient, err := NewWxClient(officialAppID)
	if err != nil {
		return err
	}
	token, err := wxClient.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/wxopen/wxampunlink?access_token=%s", token)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"appid": miniAppID,
	}).Post(apiURL)
	if err != nil {
		return err
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return err
	}
	if res.Errcode != 0 {
		return FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return nil
}

// TemplateDraftItem 小程序代码草稿箱条目
type TemplateDraftItem struct {
	DraftID      int    `json:"draft_id"`
	UserVersion  string `json:"user_version"`
	UserDesc     string `json:"user_desc"`
	Developer    string `json:"developer"`
	CreateTime   int64  `json:"create_time"`
	TemplateType int    `json:"template_type"`
}

// TemplateListItem 小程序代码模板库条目
type TemplateListItem struct {
	TemplateID   int    `json:"template_id"`
	UserVersion  string `json:"user_version"`
	UserDesc     string `json:"user_desc"`
	CreateTime   int64  `json:"create_time"`
	TemplateType int    `json:"template_type"`
	Developer    string `json:"developer,omitempty"`
}

// GetTemplateDraftList 获取代码草稿箱列表 (wxa/gettemplatedraftlist)
func (s *WxMiniService) GetTemplateDraftList(ctx context.Context, componentAppID string) ([]TemplateDraftItem, error) {
	compToken, err := DefaultWxOpenService.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/wxa/gettemplatedraftlist?access_token=%s", compToken)
	resp, err := s.http.R().SetContext(ctx).Get(apiURL)
	if err != nil {
		return nil, err
	}

	var res struct {
		DraftList []TemplateDraftItem `json:"draft_list"`
		Errcode   int                 `json:"errcode"`
		Errmsg    string              `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}
	if res.Errcode != 0 {
		return nil, FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return res.DraftList, nil
}

// AddDraftToTemplate 将草稿箱添加到模板库 (wxa/addtotemplate)
func (s *WxMiniService) AddDraftToTemplate(ctx context.Context, componentAppID string, draftID, templateType int) error {
	compToken, err := DefaultWxOpenService.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/wxa/addtotemplate?access_token=%s", compToken)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"draft_id":      draftID,
		"template_type": templateType,
	}).Post(apiURL)
	if err != nil {
		return err
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return err
	}
	if res.Errcode != 0 {
		return FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return nil
}

// GetTemplateList 获取模板库列表 (wxa/gettemplatelist)
func (s *WxMiniService) GetTemplateList(ctx context.Context, componentAppID string, templateType int) ([]TemplateListItem, error) {
	compToken, err := DefaultWxOpenService.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/wxa/gettemplatelist?access_token=%s&template_type=%d", compToken, templateType)
	resp, err := s.http.R().SetContext(ctx).Get(apiURL)
	if err != nil {
		return nil, err
	}

	var res struct {
		TemplateList []TemplateListItem `json:"template_list"`
		Errcode      int                `json:"errcode"`
		Errmsg       string             `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}
	if res.Errcode != 0 {
		return nil, FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return res.TemplateList, nil
}

// DeleteTemplate 删除指定模板 (wxa/deltemplate)
func (s *WxMiniService) DeleteTemplate(ctx context.Context, componentAppID string, templateID int) error {
	compToken, err := DefaultWxOpenService.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/wxa/deltemplate?access_token=%s", compToken)
	resp, err := s.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"template_id": templateID,
	}).Post(apiURL)
	if err != nil {
		return err
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return err
	}
	if res.Errcode != 0 {
		return FriendlyWxError(fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg))
	}

	return nil
}
