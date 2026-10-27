package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// placeholderRe 匹配微信模板占位符，兼容 {{thing1.DATA}}、{{ thing1.DATA }} 等写法
var placeholderRe = regexp.MustCompile(`\{\s*\{\s*([A-Za-z0-9_]+)\s*(?:\.[A-Za-z_]+)?\s*\}\s*\}`)

// ==========================================================
// 消息模板体系：公众号模板消息 + 订阅消息(小程序·服务号)
// 统一使用目标账号(公众号/小程序)的 access_token / authorizer_access_token
// 官方文档: /doc/service/api/notify/template/ 与 /doc/service/api/notify/notify/
// ==========================================================

type WxNotifyService struct{}

var DefaultWxNotifyService = &WxNotifyService{}

// do 统一取凭据并调用微信 JSON 接口
func (s *WxNotifyService) do(ctx context.Context, appid, method, apiPath string, q url.Values, body, out interface{}) error {
	token, err := DefaultWxMiniService.tokenForMiniApp(ctx, appid)
	if err != nil {
		return err
	}
	return DefaultWxMiniService.callMiniJSON(ctx, token, method, apiPath, q, body, out)
}

// ---------------------------- 公众号模板消息 ----------------------------

// OaPrivateTemplate 账号下的私有模板 (已选用的模板)
type OaPrivateTemplate struct {
	TemplateID      string `json:"template_id"`
	Title           string `json:"title"`
	PrimaryIndustry string `json:"primary_industry"`
	DeputyIndustry  string `json:"deputy_industry"`
	Content         string `json:"content"`
	Example         string `json:"example"`
}

// GetIndustry 获取账号已设置的服务行业 (cgi-bin/template/get_industry)
func (s *WxNotifyService) GetIndustry(ctx context.Context, appid string) (map[string]interface{}, error) {
	var res map[string]interface{}
	if err := s.do(ctx, appid, "GET", "/cgi-bin/template/get_industry", nil, nil, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// SetIndustry 设置所属行业，注意：更改行业后原行业下的模板会被删除
func (s *WxNotifyService) SetIndustry(ctx context.Context, appid, industryID1, industryID2 string) error {
	body := map[string]interface{}{"industry_id1": industryID1, "industry_id2": industryID2}
	return s.do(ctx, appid, "POST", "/cgi-bin/template/api_set_industry", nil, body, nil)
}

// GetPrivateTemplates 获取账号已选用的模板列表 (cgi-bin/template/get_all_private_template)
func (s *WxNotifyService) GetPrivateTemplates(ctx context.Context, appid string) ([]OaPrivateTemplate, error) {
	var res struct {
		TemplateList []OaPrivateTemplate `json:"template_list"`
	}
	if err := s.do(ctx, appid, "GET", "/cgi-bin/template/get_all_private_template", nil, nil, &res); err != nil {
		return nil, err
	}
	return res.TemplateList, nil
}

// AddTemplate 从公共模板库选用模板 (cgi-bin/template/api_add_template)，返回新的模板 ID
func (s *WxNotifyService) AddTemplate(ctx context.Context, appid, templateIDShort string, keywordNameList []string) (string, error) {
	body := map[string]interface{}{"template_id_short": templateIDShort}
	if len(keywordNameList) > 0 {
		body["keyword_name_list"] = keywordNameList
	}
	var res struct {
		TemplateID string `json:"template_id"`
	}
	if err := s.do(ctx, appid, "POST", "/cgi-bin/template/api_add_template", nil, body, &res); err != nil {
		return "", err
	}
	return res.TemplateID, nil
}

// DeleteTemplate 删除账号下的指定模板 (cgi-bin/template/del_private_template)
func (s *WxNotifyService) DeleteTemplate(ctx context.Context, appid, templateID string) error {
	return s.do(ctx, appid, "POST", "/cgi-bin/template/del_private_template", nil, map[string]interface{}{"template_id": templateID}, nil)
}

// TemplateMsgPayload 发送模板消息入参
type TemplateMsgPayload struct {
	ToUser      string            `json:"touser"`
	TemplateID  string            `json:"template_id"`
	URL         string            `json:"url,omitempty"`
	MiniAppID   string            `json:"mini_appid,omitempty"`
	PagePath    string            `json:"pagepath,omitempty"`
	ClientMsgID string            `json:"client_msg_id,omitempty"`
	Values      map[string]string `json:"values"`
	Color       string            `json:"color,omitempty"`
}

// buildMsgData 将扁平的关键词-值转换为微信要求的 {key:{value,color}} 结构
func buildMsgData(values map[string]string, color string) map[string]interface{} {
	data := map[string]interface{}{}
	for k, v := range values {
		item := map[string]interface{}{"value": v}
		if strings.TrimSpace(color) != "" {
			item["color"] = strings.TrimSpace(color)
		}
		data[k] = item
	}
	return data
}

// SendTemplateMsg 发送公众号模板消息 (cgi-bin/message/template/send)
func (s *WxNotifyService) SendTemplateMsg(ctx context.Context, appid string, p TemplateMsgPayload) (int64, error) {
	body := map[string]interface{}{
		"touser":      p.ToUser,
		"template_id": p.TemplateID,
		"data":        buildMsgData(p.Values, p.Color),
	}
	if strings.TrimSpace(p.URL) != "" {
		body["url"] = strings.TrimSpace(p.URL)
	}
	if strings.TrimSpace(p.MiniAppID) != "" {
		mp := map[string]interface{}{"appid": strings.TrimSpace(p.MiniAppID)}
		if strings.TrimSpace(p.PagePath) != "" {
			mp["pagepath"] = strings.TrimSpace(p.PagePath)
		}
		body["miniprogram"] = mp
	}
	if strings.TrimSpace(p.ClientMsgID) != "" {
		body["client_msg_id"] = strings.TrimSpace(p.ClientMsgID)
	}

	var res struct {
		MsgID int64 `json:"msgid"`
	}
	if err := s.do(ctx, appid, "POST", "/cgi-bin/message/template/send", nil, body, &res); err != nil {
		return 0, err
	}
	return res.MsgID, nil
}

// OneTimeSubscribePayload 一次性订阅消息下发入参
type OneTimeSubscribePayload struct {
	ToUser           string            `json:"touser"`
	TemplateID       string            `json:"template_id"`
	URL              string            `json:"url,omitempty"`
	Scene            string            `json:"scene,omitempty"`
	Title            string            `json:"title,omitempty"`
	MiniAppID        string            `json:"mini_appid,omitempty"`
	PagePath         string            `json:"pagepath,omitempty"`
	Values           map[string]string `json:"values"`
	Color            string            `json:"color,omitempty"`
	MiniprogramState string            `json:"miniprogram_state,omitempty"`
}

// SendOneTimeSubscribe 推送一次性订阅模板消息 (cgi-bin/message/template/subscribe)
func (s *WxNotifyService) SendOneTimeSubscribe(ctx context.Context, appid string, p OneTimeSubscribePayload) error {
	body := map[string]interface{}{
		"touser":      p.ToUser,
		"template_id": p.TemplateID,
		"data":        buildMsgData(p.Values, p.Color),
	}
	if strings.TrimSpace(p.URL) != "" {
		body["url"] = strings.TrimSpace(p.URL)
	}
	if strings.TrimSpace(p.Scene) != "" {
		body["scene"] = strings.TrimSpace(p.Scene)
	}
	if strings.TrimSpace(p.Title) != "" {
		body["title"] = strings.TrimSpace(p.Title)
	}
	if strings.TrimSpace(p.MiniAppID) != "" {
		mp := map[string]interface{}{"appid": strings.TrimSpace(p.MiniAppID)}
		if strings.TrimSpace(p.PagePath) != "" {
			mp["pagepath"] = strings.TrimSpace(p.PagePath)
		}
		if strings.TrimSpace(p.MiniprogramState) != "" {
			mp["miniprogram_state"] = strings.TrimSpace(p.MiniprogramState)
		}
		body["miniprogram"] = mp
	}
	return s.do(ctx, appid, "POST", "/cgi-bin/message/template/subscribe", nil, body, nil)
}

// QueryBlockedTemplateMsg 查询被拦截的模板消息 (wxa/sec/queryblocktmplmsg)
func (s *WxNotifyService) QueryBlockedTemplateMsg(ctx context.Context, appid, tmplMsgID string, largestID int64, limit int) (map[string]interface{}, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	body := map[string]interface{}{"largest_id": largestID, "limit": limit}
	if strings.TrimSpace(tmplMsgID) != "" {
		body["tmpl_msg_id"] = strings.TrimSpace(tmplMsgID)
	}
	var res map[string]interface{}
	if err := s.do(ctx, appid, "POST", "/wxa/sec/queryblocktmplmsg", nil, body, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// ---------------------------- 订阅消息 (小程序 · 服务号) ----------------------------

// SubscribeCategory 订阅消息类目
type SubscribeCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PubTemplateTitle 公共模板库标题
type PubTemplateTitle struct {
	TID        int    `json:"tid"`
	Title      string `json:"title"`
	Type       int    `json:"type"` // 2: 一次性订阅, 3: 长期订阅
	CategoryID string `json:"categoryId"`
}

// PubTemplateKeyword 公共模板关键词
type PubTemplateKeyword struct {
	KID       int    `json:"kid"`
	Name      string `json:"name"`
	Example   string `json:"example"`
	Rule      string `json:"rule"`
	FieldType int    `json:"field_type,omitempty"`
}

// SubscribeTemplate 已添加到账号下的订阅消息模板
type SubscribeTemplate struct {
	PriTmplID string `json:"priTmplId"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Example   string `json:"example"`
	Type      int    `json:"type"`
}

// GetSubscribeCategories 获取订阅消息类目列表 (wxaapi/newtmpl/getcategory)
func (s *WxNotifyService) GetSubscribeCategories(ctx context.Context, appid string) ([]SubscribeCategory, error) {
	var res struct {
		Data []SubscribeCategory `json:"data"`
	}
	if err := s.do(ctx, appid, "GET", "/wxaapi/newtmpl/getcategory", nil, nil, &res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

// GetPubTemplateTitles 拉取类目下的公共模板标题 (wxaapi/newtmpl/getpubtemplatetitles)
func (s *WxNotifyService) GetPubTemplateTitles(ctx context.Context, appid, ids string, start, limit int) ([]PubTemplateTitle, int, error) {
	if limit <= 0 || limit > 30 {
		limit = 30
	}
	q := url.Values{"ids": {ids}, "start": {fmt.Sprintf("%d", start)}, "limit": {fmt.Sprintf("%d", limit)}}
	var res struct {
		Count int                `json:"count"`
		Data  []PubTemplateTitle `json:"data"`
	}
	if err := s.do(ctx, appid, "GET", "/wxaapi/newtmpl/getpubtemplatetitles", q, nil, &res); err != nil {
		return nil, 0, err
	}
	return res.Data, res.Count, nil
}

// GetPubTemplateKeywords 获取公共模板的关键词列表 (wxaapi/newtmpl/getpubtemplatekeywords)
func (s *WxNotifyService) GetPubTemplateKeywords(ctx context.Context, appid string, tid int) ([]PubTemplateKeyword, error) {
	q := url.Values{"tid": {fmt.Sprintf("%d", tid)}}
	var res struct {
		Data []PubTemplateKeyword `json:"data"`
	}
	if err := s.do(ctx, appid, "GET", "/wxaapi/newtmpl/getpubtemplatekeywords", q, nil, &res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

// AddSubscribeTemplate 把公共模板加入我的模板库 (wxaapi/newtmpl/addtemplate)
func (s *WxNotifyService) AddSubscribeTemplate(ctx context.Context, appid string, tid int, kids []int, sceneDesc string) (string, error) {
	if len(kids) < 2 {
		return "", fmt.Errorf("至少需要选择 2 个关键词")
	}
	if len(kids) > 5 {
		return "", fmt.Errorf("最多只能选择 5 个关键词")
	}
	body := map[string]interface{}{"tid": tid, "kidList": kids, "sceneDesc": sceneDesc}
	var res struct {
		PriTmplID string `json:"priTmplId"`
	}
	if err := s.do(ctx, appid, "POST", "/wxaapi/newtmpl/addtemplate", nil, body, &res); err != nil {
		return "", err
	}
	return res.PriTmplID, nil
}

// GetMySubscribeTemplates 获取我的订阅消息模板列表 (wxaapi/newtmpl/gettemplate)
func (s *WxNotifyService) GetMySubscribeTemplates(ctx context.Context, appid string) ([]SubscribeTemplate, error) {
	var res struct {
		Data []SubscribeTemplate `json:"data"`
	}
	if err := s.do(ctx, appid, "GET", "/wxaapi/newtmpl/gettemplate", nil, nil, &res); err != nil {
		return nil, err
	}
	return res.Data, nil
}

// DeleteSubscribeTemplate 删除我的订阅消息模板 (wxaapi/newtmpl/deltemplate)
func (s *WxNotifyService) DeleteSubscribeTemplate(ctx context.Context, appid, priTmplID string) error {
	return s.do(ctx, appid, "POST", "/wxaapi/newtmpl/deltemplate", nil, map[string]interface{}{"priTmplId": priTmplID}, nil)
}

// SubscribeMsgPayload 发送订阅消息入参
type SubscribeMsgPayload struct {
	ToUser           string            `json:"touser"`
	TemplateID       string            `json:"template_id"`
	Page             string            `json:"page,omitempty"`
	MiniprogramState string            `json:"miniprogram_state,omitempty"`
	Lang             string            `json:"lang,omitempty"`
	Values           map[string]string `json:"values"`
}

// SendSubscribeMsg 发送小程序/服务号订阅消息 (cgi-bin/message/subscribe/send)
func (s *WxNotifyService) SendSubscribeMsg(ctx context.Context, appid string, p SubscribeMsgPayload) error {
	state := strings.TrimSpace(p.MiniprogramState)
	if state == "" {
		state = "formal"
	}
	lang := strings.TrimSpace(p.Lang)
	if lang == "" {
		lang = "zh_CN"
	}
	body := map[string]interface{}{
		"touser":            p.ToUser,
		"template_id":       p.TemplateID,
		"data":              buildMsgData(p.Values, ""),
		"miniprogram_state": state,
		"lang":              lang,
	}
	if strings.TrimSpace(p.Page) != "" {
		body["page"] = strings.TrimSpace(p.Page)
	}
	return s.do(ctx, appid, "POST", "/cgi-bin/message/subscribe/send", nil, body, nil)
}

// SendBizSubscribeMsg 发送服务号一次性订阅消息 (cgi-bin/message/subscribe/bizsend)
func (s *WxNotifyService) SendBizSubscribeMsg(ctx context.Context, appid string, p SubscribeMsgPayload, jumpURL string) error {
	body := map[string]interface{}{
		"touser":      p.ToUser,
		"template_id": p.TemplateID,
		"data":        buildMsgData(p.Values, ""),
	}
	if strings.TrimSpace(jumpURL) != "" {
		body["url"] = strings.TrimSpace(jumpURL)
	}
	if strings.TrimSpace(p.Page) != "" {
		body["page"] = strings.TrimSpace(p.Page)
	}
	return s.do(ctx, appid, "POST", "/cgi-bin/message/subscribe/bizsend", nil, body, nil)
}

// ExtractTemplateKeywords 从模板 content 中提取 {{xxx.DATA}} 关键词占位 (按出现顺序去重)
func ExtractTemplateKeywords(content string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range placeholderRe.FindAllStringSubmatch(content, -1) {
		key := strings.TrimSpace(m[1])
		if key != "" && !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}

// ---------------------------- 服务提醒卡片 (订阅消息·服务卡片) ----------------------------
// 官方文档: /miniprogram/dev/framework/open-ability/subscribe-message-2
// 接口: wxa/set_user_notify (激活与更新) / wxa/get_user_notify (查询状态)
// 第三方代调用权限集 id = 18，可使用 authorizer_access_token

// GetServiceCard 查询用户服务卡片状态 (wxa/get_user_notify)
// code_state: 0 正常 / 1 有风险 / 2 异常 / 10 用户拒收本次 code
func (s *WxNotifyService) GetServiceCard(ctx context.Context, appid, openid, notifyCode string, notifyType int) (map[string]interface{}, error) {
	if strings.TrimSpace(openid) == "" {
		return nil, fmt.Errorf("%w: 请填写用户 openid", ErrInvalidParam)
	}
	if strings.TrimSpace(notifyCode) == "" {
		return nil, fmt.Errorf("%w: 请填写动态更新令牌 notify_code", ErrInvalidParam)
	}
	if notifyType <= 0 {
		return nil, fmt.Errorf("%w: 请填写正确的卡片 id (notify_type)，具体取值与字段见官方「服务提醒」卡片定义", ErrInvalidParam)
	}
	body := map[string]interface{}{
		"openid":      strings.TrimSpace(openid),
		"notify_code": strings.TrimSpace(notifyCode),
		"notify_type": notifyType,
	}
	var res map[string]interface{}
	if err := s.do(ctx, appid, "POST", "/wxa/get_user_notify", nil, body, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// SetServiceCard 激活或更新用户服务卡片 (wxa/set_user_notify)
// contentJSON 为卡片状态 JSON 字符串 (不同卡片字段不同)；checkJSON 仅在使用微信支付订单号作为 notify_code 时需要
func (s *WxNotifyService) SetServiceCard(ctx context.Context, appid, openid, notifyCode string, notifyType int, contentJSON, checkJSON string) error {
	if strings.TrimSpace(openid) == "" {
		return fmt.Errorf("%w: 请填写用户 openid", ErrInvalidParam)
	}
	if strings.TrimSpace(notifyCode) == "" {
		return fmt.Errorf("%w: 请填写动态更新令牌 notify_code (前端 button 获取或微信支付订单号)", ErrInvalidParam)
	}
	if notifyType <= 0 {
		return fmt.Errorf("%w: 请填写正确的卡片 id (notify_type)", ErrInvalidParam)
	}
	content := strings.TrimSpace(contentJSON)
	if content == "" {
		return fmt.Errorf("%w: 请填写卡片内容 content_json，例如 {\"cur_status\":2,\"arrival_time\":1679569348}", ErrInvalidParam)
	}
	if !json.Valid([]byte(content)) {
		return fmt.Errorf("%w: content_json 不是合法的 JSON 字符串", ErrInvalidParam)
	}
	body := map[string]interface{}{
		"openid":       strings.TrimSpace(openid),
		"notify_code":  strings.TrimSpace(notifyCode),
		"notify_type":  notifyType,
		"content_json": content,
	}
	if v := strings.TrimSpace(checkJSON); v != "" {
		if !json.Valid([]byte(v)) {
			return fmt.Errorf("%w: check_json 不是合法的 JSON 字符串", ErrInvalidParam)
		}
		body["check_json"] = v
	}
	return s.do(ctx, appid, "POST", "/wxa/set_user_notify", nil, body, nil)
}
