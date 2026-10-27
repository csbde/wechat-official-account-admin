package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
)

// ==========================================================
// 小程序代码管理与发布 (第三方平台代小程序调用)
// 全部接口统一使用目标小程序的 authorizer_access_token
// 官方文档: /doc/oplatform/openApi/miniprogram-management/code-management/
// ==========================================================

// tokenForMiniApp 获取目标小程序的调用凭证 (自动适配第三方代管与直连凭据)
func (s *WxMiniService) tokenForMiniApp(ctx context.Context, miniAppID string) (string, error) {
	wxClient, err := NewWxClient(miniAppID)
	if err != nil {
		return "", err
	}
	return wxClient.GetAccessToken(ctx)
}

// callMiniJSON 统一发起小程序侧 JSON 接口调用，并校验微信 errcode
func (s *WxMiniService) callMiniJSON(ctx context.Context, token, method, apiPath string, query url.Values, body interface{}, out interface{}) error {
	u := "https://api.weixin.qq.com" + apiPath + "?access_token=" + url.QueryEscape(token)
	if len(query) > 0 {
		u += "&" + query.Encode()
	}

	req := s.http.R().SetContext(ctx)
	if body != nil {
		req = req.SetHeader("Content-Type", "application/json").SetBody(body)
	}

	var resp *resty.Response
	var err error
	switch strings.ToUpper(method) {
	case "GET":
		resp, err = req.Get(u)
	case "DELETE":
		resp, err = req.Delete(u)
	default:
		resp, err = req.Post(u)
	}
	if err != nil {
		return fmt.Errorf("请求微信接口网络错误: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("微信接口无响应内容")
	}
	raw := resp.Body()

	var base struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if e := json.Unmarshal(raw, &base); e != nil {
		return fmt.Errorf("解析微信响应失败: %v (原始响应: %s)", e, snippet(raw))
	}
	if base.Errcode != 0 {
		return fmt.Errorf("[%d] %s", base.Errcode, base.Errmsg)
	}
	if out != nil {
		if e := json.Unmarshal(raw, out); e != nil {
			return fmt.Errorf("解析微信数据失败: %v", e)
		}
	}
	return nil
}

// snippet 截断过长的响应体，避免日志与错误信息爆炸
func snippet(b []byte) string {
	t := strings.TrimSpace(string(b))
	if len(t) > 200 {
		return t[:200] + "..."
	}
	return t
}

// ---------------------------- 提交代码 ----------------------------

// CommitCodeParam 小程序代码提交参数 (wxa/commit)
type CommitCodeParam struct {
	TemplateID  int    `json:"template_id"`  // 代码模板库中的模板 ID
	ExtJSON     string `json:"ext_json"`     // ext.json 配置内容
	UserVersion string `json:"user_version"` // 代码版本号
	UserDesc    string `json:"user_desc"`    // 代码描述
}

// CommitCode 使用代码模板为小程序提交新代码 (wxa/commit)
func (s *WxMiniService) CommitCode(ctx context.Context, miniAppID string, p CommitCodeParam) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}

	// ext_json 为空时按微信要求注入目标小程序 appid，保证模板中的 {{extAppid}} 可被正确替换
	extJSON := strings.TrimSpace(p.ExtJSON)
	if extJSON == "" || !json.Valid([]byte(extJSON)) {
		extJSON = fmt.Sprintf(`{"extAppid":%q,"extEnable":true,"ext":{}}`, miniAppID)
	}

	body := map[string]interface{}{
		"template_id":  p.TemplateID,
		"ext_json":     extJSON,
		"user_version": p.UserVersion,
		"user_desc":    p.UserDesc,
	}
	return s.callMiniJSON(ctx, token, "POST", "/wxa/commit", nil, body, nil)
}

// ---------------------------- 体验版与页面 ----------------------------

// GetCodePageList 获取小程序页面列表 (wxa/get_page)，提交审核填写页面地址时使用
func (s *WxMiniService) GetCodePageList(ctx context.Context, miniAppID string) ([]string, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res struct {
		PageList []string `json:"page_list"`
	}
	if err := s.callMiniJSON(ctx, token, "GET", "/wxa/get_page", nil, nil, &res); err != nil {
		return nil, err
	}
	return res.PageList, nil
}

// GetTrialQrcode 获取小程序体验版二维码 (wxa/get_qrcode)，返回 base64 图片数据
func (s *WxMiniService) GetTrialQrcode(ctx context.Context, miniAppID, path string) (string, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return "", err
	}

	u := "https://api.weixin.qq.com/wxa/get_qrcode?access_token=" + url.QueryEscape(token)
	if strings.TrimSpace(path) != "" {
		u += "&path=" + url.QueryEscape(strings.TrimSpace(path))
	}

	resp, err := s.http.R().SetContext(ctx).Get(u)
	if err != nil {
		return "", fmt.Errorf("请求体验版二维码网络错误: %w", err)
	}

	ct := strings.ToLower(resp.Header().Get("Content-Type"))
	body := resp.Body()
	if strings.Contains(ct, "json") || strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		var e struct {
			Errcode int    `json:"errcode"`
			Errmsg  string `json:"errmsg"`
		}
		_ = json.Unmarshal(body, &e)
		return "", fmt.Errorf("[%d] %s", e.Errcode, e.Errmsg)
	}
	if len(body) < 100 {
		return "", fmt.Errorf("微信返回的二维码内容异常，可能尚未提交体验版代码")
	}

	mime := "image/jpeg"
	if strings.Contains(ct, "png") {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(body), nil
}

// ---------------------------- 提交审核 ----------------------------

// AuditItem 提审类目信息 (item_list 单条)
type AuditItem struct {
	Address     string `json:"address,omitempty"`      // 小程序页面，可通过 get_page 获取
	Tag         string `json:"tag,omitempty"`          // 标签，空格分隔，至多 10 个
	FirstClass  string `json:"first_class,omitempty"`  // 一级类目名称
	SecondClass string `json:"second_class,omitempty"` // 二级类目名称
	ThirdClass  string `json:"third_class,omitempty"`  // 三级类目名称
	FirstID     int    `json:"first_id,omitempty"`     // 一级类目 ID
	SecondID    int    `json:"second_id,omitempty"`    // 二级类目 ID
	ThirdID     int    `json:"third_id,omitempty"`     // 三级类目 ID
	Title       string `json:"title,omitempty"`        // 页面标题，至多 32 字
}

// SubmitAuditParam 提交审核参数 (wxa/submit_audit)
type SubmitAuditParam struct {
	ItemList      []AuditItem `json:"item_list,omitempty"`
	FeedbackInfo  string      `json:"feedback_info,omitempty"`       // 审核反馈说明
	FeedbackStuff string      `json:"feedback_stuff,omitempty"`      // 截图 media_id，| 分隔，至多 5 张
	VersionDesc   string      `json:"version_desc,omitempty"`        // 版本说明与功能解释
	UgcDeclare    interface{} `json:"ugc_declare,omitempty"`         // UGC 安全声明
	PrivacyNotUse bool        `json:"privacy_api_not_use,omitempty"` // 声明不使用未配置隐私接口
	OrderPath     string      `json:"order_path,omitempty"`          // 订单中心 path
}

// SubmitAudit 提交代码审核，返回微信下发的审核单号
func (s *WxMiniService) SubmitAudit(ctx context.Context, miniAppID string, p SubmitAuditParam) (int64, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return 0, err
	}
	var res struct {
		AuditID int64 `json:"auditid"`
	}
	if err := s.callMiniJSON(ctx, token, "POST", "/wxa/submit_audit", nil, p, &res); err != nil {
		return 0, err
	}
	return res.AuditID, nil
}

// AuditStatusResult 审核单状态
type AuditStatusResult struct {
	AuditID         int64  `json:"auditid"`
	Status          int    `json:"status"`
	Reason          string `json:"reason"`
	Screenshot      string `json:"screenshot"`
	UserVersion     string `json:"user_version"`
	UserDesc        string `json:"user_desc"`
	SubmitAuditTime int64  `json:"submit_audit_time"`
}

// GetAuditStatus 查询指定审核单状态 (wxa/get_auditstatus)
func (s *WxMiniService) GetAuditStatus(ctx context.Context, miniAppID string, auditID int64) (*AuditStatusResult, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res AuditStatusResult
	if err := s.callMiniJSON(ctx, token, "POST", "/wxa/get_auditstatus", nil, map[string]interface{}{"auditid": auditID}, &res); err != nil {
		return nil, err
	}
	res.AuditID = auditID
	return &res, nil
}

// GetLatestAuditStatus 查询最近一次提交审核记录 (wxa/get_latest_auditstatus)
func (s *WxMiniService) GetLatestAuditStatus(ctx context.Context, miniAppID string) (*AuditStatusResult, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res AuditStatusResult
	if err := s.callMiniJSON(ctx, token, "GET", "/wxa/get_latest_auditstatus", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// UndoCodeAudit 撤回审核中的代码提审 (wxa/undocodeaudit)
func (s *WxMiniService) UndoCodeAudit(ctx context.Context, miniAppID string) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	return s.callMiniJSON(ctx, token, "GET", "/wxa/undocodeaudit", nil, nil, nil)
}

// CodePrivacyInfo 代码中检测出的隐私接口与配置/权限差异
type CodePrivacyInfo struct {
	WithoutAuthList []string `json:"without_auth_list"` // 代码用到但无权限的隐私接口
	WithoutConfList []string `json:"without_conf_list"` // 代码用到但未在 ext.json 配置的隐私接口
}

// GetCodePrivacyInfo 查询代码隐私接口检测结果 (wxa/security/get_code_privacy_info)
// 提交代码后微信需要一小段时间做静态检测，检测未完成时提审会报 61039
func (s *WxMiniService) GetCodePrivacyInfo(ctx context.Context, miniAppID string) (*CodePrivacyInfo, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res CodePrivacyInfo
	if err := s.callMiniJSON(ctx, token, "GET", "/wxa/security/get_code_privacy_info", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ---------------------------- 发布与灰度 ----------------------------

// ReleaseCode 发布已通过审核的小程序版本 (wxa/release)
func (s *WxMiniService) ReleaseCode(ctx context.Context, miniAppID string) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	return s.callMiniJSON(ctx, token, "POST", "/wxa/release", nil, map[string]interface{}{}, nil)
}

// VisitStatus 服务可访问状态 (0 暂停服务 / 1 正常)
type VisitStatus struct {
	Status int `json:"status"`
}

// GetVisitStatus 查询发布后的可访问状态 (wxa/getvisitstatus)
func (s *WxMiniService) GetVisitStatus(ctx context.Context, miniAppID string) (*VisitStatus, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res VisitStatus
	if err := s.callMiniJSON(ctx, token, "POST", "/wxa/getvisitstatus", nil, map[string]interface{}{}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SetVisitStatus 设置可访问状态 (wxa/change_visitstatus)，action: open / close
func (s *WxMiniService) SetVisitStatus(ctx context.Context, miniAppID, action string) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	return s.callMiniJSON(ctx, token, "POST", "/wxa/change_visitstatus", nil, map[string]interface{}{"action": action}, nil)
}

// RevertVersion 可回退的历史线上版本
type RevertVersion struct {
	AppVersion  int64  `json:"app_version"`
	UserVersion string `json:"user_version"`
	UserDesc    string `json:"user_desc"`
	CommitTime  int64  `json:"commit_time"`
}

// GetHistoryVersions 获取可回退的小程序历史版本 (wxa/revertcoderelease?action=get_history_version)
func (s *WxMiniService) GetHistoryVersions(ctx context.Context, miniAppID string) ([]RevertVersion, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res struct {
		VersionList []RevertVersion `json:"version_list"`
	}
	q := url.Values{"action": {"get_history_version"}}
	if err := s.callMiniJSON(ctx, token, "GET", "/wxa/revertcoderelease", q, nil, &res); err != nil {
		return nil, err
	}
	return res.VersionList, nil
}

// RevertCodeRelease 回退线上版本，appVersion 为空表示回退到上一个版本
func (s *WxMiniService) RevertCodeRelease(ctx context.Context, miniAppID, appVersion string) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	q := url.Values{}
	if strings.TrimSpace(appVersion) != "" {
		q.Set("app_version", strings.TrimSpace(appVersion))
	}
	return s.callMiniJSON(ctx, token, "GET", "/wxa/revertcoderelease", q, nil, nil)
}

// ---------------------------- 基础库最低支持版本 ----------------------------

// VersionUv 单个基础库版本的用户占比
type VersionUv struct {
	Version    string  `json:"version"`
	Percentage float64 `json:"percentage"`
}

// SupportVersionInfo 当前设置的基础库版本与用户占比
type SupportVersionInfo struct {
	NowVersion string `json:"now_version"`
	UvInfo     struct {
		Items []VersionUv `json:"items"`
	} `json:"uv_info"`
}

// GetSupportVersion 获取小程序当前最低基础库版本 (cgi-bin/wxopen/getweappsupportversion)
func (s *WxMiniService) GetSupportVersion(ctx context.Context, miniAppID string) (*SupportVersionInfo, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res SupportVersionInfo
	if err := s.callMiniJSON(ctx, token, "POST", "/cgi-bin/wxopen/getweappsupportversion", nil, map[string]interface{}{}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SetSupportVersion 设置小程序最低基础库版本 (cgi-bin/wxopen/setweappsupportversion)
func (s *WxMiniService) SetSupportVersion(ctx context.Context, miniAppID, version string) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	return s.callMiniJSON(ctx, token, "POST", "/cgi-bin/wxopen/setweappsupportversion", nil, map[string]interface{}{"version": version}, nil)
}

// ---------------------------- 服务类目管理 ----------------------------

// CategoryCert 类目资质
type CategoryCert struct {
	Key   string `json:"key"`
	Value string `json:"value"` // 资质图片 media_id
}

// CategoryItem 小程序服务类目
type CategoryItem struct {
	First       int            `json:"first"`
	Second      int            `json:"second"`
	FirstName   string         `json:"first_name,omitempty"`
	SecondName  string         `json:"second_name,omitempty"`
	AuditStatus int            `json:"audit_status,omitempty"`
	AuditReason string         `json:"audit_reason,omitempty"`
	Certicates  []CategoryCert `json:"certicates,omitempty"`
}

// CategoryListResult 已设置类目列表与配额
type CategoryListResult struct {
	Categories    []CategoryItem `json:"categories"`
	Limit         int            `json:"limit"`
	Quota         int            `json:"quota"`
	CategoryLimit int            `json:"category_limit"`
}

// GetCategories 获取小程序已设置的服务类目 (cgi-bin/wxopen/getcategory)
func (s *WxMiniService) GetCategories(ctx context.Context, miniAppID string) (*CategoryListResult, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res CategoryListResult
	if err := s.callMiniJSON(ctx, token, "GET", "/cgi-bin/wxopen/getcategory", nil, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// AddCategory 为小程序新增服务类目 (cgi-bin/wxopen/addcategory)
func (s *WxMiniService) AddCategory(ctx context.Context, miniAppID string, items []CategoryItem) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	return s.callMiniJSON(ctx, token, "POST", "/cgi-bin/wxopen/addcategory", nil, map[string]interface{}{"categories": items}, nil)
}

// DeleteCategory 删除小程序服务类目 (cgi-bin/wxopen/deletecategory)
func (s *WxMiniService) DeleteCategory(ctx context.Context, miniAppID string, first, second int) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	return s.callMiniJSON(ctx, token, "POST", "/cgi-bin/wxopen/deletecategory", nil, map[string]interface{}{"first": first, "second": second}, nil)
}

// ---------------------------- 体验者成员管理 ----------------------------

// TesterMember 体验者成员
type TesterMember struct {
	Wechatid string `json:"wechatid,omitempty"`
	UserStr  string `json:"userstr,omitempty"`
}

// GetTesterList 获取体验者列表 (wxa/memberauth action=get_experiencer)
func (s *WxMiniService) GetTesterList(ctx context.Context, miniAppID string) ([]TesterMember, error) {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return nil, err
	}
	var res struct {
		Members []TesterMember `json:"members"`
	}
	if err := s.callMiniJSON(ctx, token, "POST", "/wxa/memberauth", nil, map[string]interface{}{"action": "get_experiencer"}, &res); err != nil {
		return nil, err
	}
	return res.Members, nil
}

// BindTester 绑定体验者 (wxa/bind_tester)
func (s *WxMiniService) BindTester(ctx context.Context, miniAppID, wechatID string) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	return s.callMiniJSON(ctx, token, "POST", "/wxa/bind_tester", nil, map[string]interface{}{"wechatid": wechatID}, nil)
}

// UnbindTester 解绑体验者 (wxa/unbind_tester)，wechatid 与 userstr 二选一
func (s *WxMiniService) UnbindTester(ctx context.Context, miniAppID, wechatID, userStr string) error {
	token, err := s.tokenForMiniApp(ctx, miniAppID)
	if err != nil {
		return err
	}
	body := map[string]interface{}{}
	if wechatID != "" {
		body["wechatid"] = wechatID
	}
	if userStr != "" {
		body["userstr"] = userStr
	}
	return s.callMiniJSON(ctx, token, "POST", "/wxa/unbind_tester", nil, body, nil)
}
