package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

// CreateFastRegisterTask 提交快速注册小程序任务 (免 300 元认证费)
func (s *WxMiniService) CreateFastRegisterTask(ctx context.Context, componentAppID string, p FastRegisterParam) error {
	compToken, err := DefaultWxOpenService.GetComponentAccessToken(ctx, componentAppID)
	if err != nil {
		return fmt.Errorf("获取 component_access_token 失败: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/component/fastregisterweapp?action=create&component_access_token=%s", compToken)
	resp, err := s.http.R().SetContext(ctx).SetBody(p).Post(apiURL)
	if err != nil {
		return fmt.Errorf("调用微信快速注册接口失败: %w", err)
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if res.Errcode != 0 {
		return fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg)
	}

	logger.Info("[快速注册小程序] 任务已成功提交至微信服务器", zap.String("enterprise", p.Name), zap.String("legal_wechat", p.LegalPersonaWechat))
	return nil
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
		return nil, fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg)
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
		return fmt.Errorf("modify domain error: [%d] %s", res.Errcode, res.Errmsg)
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
