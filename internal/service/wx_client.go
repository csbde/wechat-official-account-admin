package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/cache"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/go-resty/resty/v2"
)

type WxClient struct {
	Account   model.WxAccount
	AppID     string
	AppSecret string
	Token     string
	AesKey    string
	http      *resty.Client
}

func NewWxClient(appid string) (*WxClient, error) {
	var account model.WxAccount
	if err := database.DB.Where("app_id = ? AND enabled = ?", appid, true).First(&account).Error; err != nil {
		return nil, fmt.Errorf("wx account not found or disabled: %s", appid)
	}

	client := resty.New().
		SetTimeout(15 * time.Second).
		SetHeader("User-Agent", "woaa-server/2.0")

	return &WxClient{
		Account:   account,
		AppID:     account.AppID,
		AppSecret: account.AppSecret,
		Token:     account.Token,
		AesKey:    account.EncodingAESKey,
		http:      client,
	}, nil
}

// CheckSignature 微信服务器接入签名校验
func (w *WxClient) CheckSignature(signature, timestamp, nonce string) bool {
	strs := []string{w.Token, timestamp, nonce}
	sort.Strings(strs)
	combined := strings.Join(strs, "")
	h := sha1.New()
	h.Write([]byte(combined))
	digest := hex.EncodeToString(h.Sum(nil))
	return digest == signature
}

// GetAccessToken 获取微信 AccessToken (智能适配直连公众号与开放平台第三方代管公众号)
func (w *WxClient) GetAccessToken(ctx context.Context) (string, error) {
	// 如果是微信第三方平台代管授权接入
	if w.Account.AccountType == "authorized" && w.Account.ComponentAppID != "" {
		return DefaultWxOpenService.RefreshAuthorizerToken(
			ctx,
			w.Account.ComponentAppID,
			w.Account.AppID,
			w.Account.AuthorizerRefreshToken,
		)
	}

	// 否则走普通直连模式
	cacheKey := fmt.Sprintf("wx:access_token:%s", w.AppID)
	if cache.RDB != nil {
		val, err := cache.RDB.Get(ctx, cacheKey).Result()
		if err == nil && val != "" {
			return val, nil
		}
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", w.AppID, w.AppSecret)
	resp, err := w.http.R().SetContext(ctx).Get(url)
	if err != nil {
		return "", err
	}

	var res struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return "", err
	}
	if res.Errcode != 0 {
		return "", fmt.Errorf("wechat get token failed: [%d] %s", res.Errcode, res.Errmsg)
	}

	if cache.RDB != nil && res.AccessToken != "" {
		ttl := time.Duration(res.ExpiresIn-300) * time.Second
		if ttl <= 0 {
			ttl = 60 * time.Second
		}
		_ = cache.RDB.Set(ctx, cacheKey, res.AccessToken, ttl).Err()
	}

	return res.AccessToken, nil
}

// GetWxMenu 获取微信服务器已发布的菜单
func (w *WxClient) GetWxMenu(ctx context.Context) (map[string]interface{}, error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/menu/get?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(resp.Body(), &res)
	return res, nil
}

// CreateWxMenu 创建普通自定义菜单
func (w *WxClient) CreateWxMenu(ctx context.Context, menuData interface{}) error {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/menu/create?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(menuData).Post(url)
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
		return fmt.Errorf("create menu error: [%d] %s", res.Errcode, res.Errmsg)
	}
	return nil
}

// DeleteWxMenu 删除默认菜单
func (w *WxClient) DeleteWxMenu(ctx context.Context) error {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/menu/delete?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).Get(url)
	if err != nil {
		return err
	}
	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(resp.Body(), &res)
	if res.Errcode != 0 {
		return fmt.Errorf("delete menu error: [%d] %s", res.Errcode, res.Errmsg)
	}
	return nil
}

// CreateConditionalMenu 创建个性化菜单
func (w *WxClient) CreateConditionalMenu(ctx context.Context, payload interface{}) (string, error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/menu/addconditional?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(payload).Post(url)
	if err != nil {
		return "", err
	}
	var res struct {
		MenuID  string `json:"menuid"`
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return "", err
	}
	if res.Errcode != 0 {
		return "", fmt.Errorf("addconditional menu error: [%d] %s", res.Errcode, res.Errmsg)
	}
	return res.MenuID, nil
}

// DeleteConditionalMenu 删除个性化菜单
func (w *WxClient) DeleteConditionalMenu(ctx context.Context, menuid string) error {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/menu/delconditional?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(map[string]string{"menuid": menuid}).Post(url)
	if err != nil {
		return err
	}
	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(resp.Body(), &res)
	if res.Errcode != 0 {
		return fmt.Errorf("delconditional menu error: [%d] %s", res.Errcode, res.Errmsg)
	}
	return nil
}

// GetMaterialList 获取永久素材列表
func (w *WxClient) GetMaterialList(ctx context.Context, mediaType string, offset, count int) (map[string]interface{}, error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/material/batchget_material?access_token=%s", token)
	body := map[string]interface{}{
		"type":   mediaType,
		"offset": offset,
		"count":  count,
	}
	resp, err := w.http.R().SetContext(ctx).SetBody(body).Post(url)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(resp.Body(), &res)
	return res, nil
}

// CreateQRCode 创建带参二维码 (返回 ticket, 图片URL, 微信扫码链接, 错误)
func (w *WxClient) CreateQRCode(ctx context.Context, qrcodeType string, sceneID int, sceneStr string, expireSeconds int) (ticket, imageURL, scanURL string, err error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return "", "", "", err
	}
	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/qrcode/create?access_token=%s", token)

	body := map[string]interface{}{}
	actionInfo := map[string]interface{}{}

	if qrcodeType == "temp" {
		// 微信官方限制：临时二维码过期时间最长不能超过 30 天 (2592000 秒)
		if expireSeconds <= 0 || expireSeconds > 2592000 {
			expireSeconds = 2592000
		}
		body["expire_seconds"] = expireSeconds
		if sceneStr != "" {
			body["action_name"] = "QR_STR_SCENE"
			actionInfo["scene"] = map[string]interface{}{"scene_str": sceneStr}
		} else {
			body["action_name"] = "QR_SCENE"
			actionInfo["scene"] = map[string]interface{}{"scene_id": sceneID}
		}
	} else {
		if sceneStr != "" {
			body["action_name"] = "QR_LIMIT_STR_SCENE"
			actionInfo["scene"] = map[string]interface{}{"scene_str": sceneStr}
		} else {
			body["action_name"] = "QR_LIMIT_SCENE"
			actionInfo["scene"] = map[string]interface{}{"scene_id": sceneID}
		}
	}
	body["action_info"] = actionInfo

	resp, err := w.http.R().SetContext(ctx).SetBody(body).Post(apiURL)
	if err != nil {
		return "", "", "", err
	}

	var res struct {
		Ticket        string `json:"ticket"`
		ExpireSeconds int    `json:"expire_seconds"`
		URL           string `json:"url"` // 微信扫码内容
		Errcode       int    `json:"errcode"`
		Errmsg        string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return "", "", "", err
	}
	if res.Errcode != 0 {
		return "", "", "", fmt.Errorf("[%d] %s", res.Errcode, res.Errmsg)
	}

	// 微信官方要求：展示二维码图片必须调用 showqrcode?ticket=xxx
	imgURL := fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=%s", url.QueryEscape(res.Ticket))

	return res.Ticket, imgURL, res.URL, nil
}

// GetUserList 获取关注者列表
func (w *WxClient) GetUserList(ctx context.Context, nextOpenID string) (map[string]interface{}, error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/get?access_token=%s&next_openid=%s", token, nextOpenID)
	resp, err := w.http.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	_ = json.Unmarshal(resp.Body(), &res)
	return res, nil
}

// GetTags 获取标签列表
func (w *WxClient) GetTags(ctx context.Context) ([]map[string]interface{}, error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/tags/get?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).Get(url)
	if err != nil {
		return nil, err
	}
	var res struct {
		Tags    []map[string]interface{} `json:"tags"`
		Errcode int                      `json:"errcode"`
		Errmsg  string                   `json:"errmsg"`
	}
	_ = json.Unmarshal(resp.Body(), &res)
	if res.Errcode != 0 {
		return nil, fmt.Errorf("get tags failed: [%d] %s", res.Errcode, res.Errmsg)
	}
	return res.Tags, nil
}

// UpdateUserRemark 设置用户备注名
func (w *WxClient) UpdateUserRemark(ctx context.Context, openid, remark string) error {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/info/updateremark?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(map[string]string{
		"openid": openid,
		"remark": remark,
	}).Post(url)
	if err != nil {
		return err
	}
	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(resp.Body(), &res)
	if res.Errcode != 0 {
		return fmt.Errorf("update remark failed: [%d] %s", res.Errcode, res.Errmsg)
	}
	return nil
}

type WxUserDetail struct {
	Subscribe      int    `json:"subscribe"`
	OpenID         string `json:"openid"`
	Nickname       string `json:"nickname"`
	Sex            int    `json:"sex"`
	Language       string `json:"language"`
	City           string `json:"city"`
	Province       string `json:"province"`
	Country        string `json:"country"`
	HeadImgURL     string `json:"headimgurl"`
	SubscribeTime  int64  `json:"subscribe_time"`
	UnionID        string `json:"unionid"`
	Remark         string `json:"remark"`
	GroupID        int    `json:"groupid"`
	TagIDList      []int  `json:"tagid_list"`
	SubscribeScene string `json:"subscribe_scene"`
}

// BatchGetUserInfo 批量获取粉丝详细资料 (每批最多100个)
func (w *WxClient) BatchGetUserInfo(ctx context.Context, openIDs []string) ([]WxUserDetail, error) {
	if len(openIDs) == 0 {
		return nil, nil
	}
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	userListParam := make([]map[string]string, len(openIDs))
	for i, oid := range openIDs {
		userListParam[i] = map[string]string{
			"openid": oid,
			"lang":   "zh_CN",
		}
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/info/batchget?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"user_list": userListParam,
	}).Post(url)

	if err != nil {
		return nil, err
	}

	var res struct {
		UserInfoList []WxUserDetail `json:"user_info_list"`
		Errcode      int            `json:"errcode"`
		Errmsg       string         `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}
	if res.Errcode != 0 {
		return nil, fmt.Errorf("batchget user info failed: [%d] %s", res.Errcode, res.Errmsg)
	}

	return res.UserInfoList, nil
}

// GetSingleUserInfo 获取单个粉丝详细资料
func (w *WxClient) GetSingleUserInfo(ctx context.Context, openID string) (*WxUserDetail, error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN", token, openID)
	resp, err := w.http.R().SetContext(ctx).Get(apiURL)
	if err != nil {
		return nil, err
	}

	var res struct {
		WxUserDetail
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return nil, err
	}
	if res.Errcode != 0 {
		return nil, fmt.Errorf("get user info failed: [%d] %s", res.Errcode, res.Errmsg)
	}

	return &res.WxUserDetail, nil
}

// CreateWxTag 向微信官方创建新标签并获取微信分配的 tag_id
func (w *WxClient) CreateWxTag(ctx context.Context, name string) (int, error) {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return 0, err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/tags/create?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"tag": map[string]string{"name": name},
	}).Post(apiURL)
	if err != nil {
		return 0, err
	}

	var res struct {
		Tag struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"tag"`
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return 0, err
	}
	if res.Errcode != 0 {
		return 0, fmt.Errorf("create tag failed: [%d] %s", res.Errcode, res.Errmsg)
	}

	return res.Tag.ID, nil
}

// DeleteWxTag 向微信官方删除标签
func (w *WxClient) DeleteWxTag(ctx context.Context, tagID int) error {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/tags/delete?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"tag": map[string]int{"id": tagID},
	}).Post(apiURL)
	if err != nil {
		return err
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(resp.Body(), &res)
	if res.Errcode != 0 {
		return fmt.Errorf("delete tag failed: [%d] %s", res.Errcode, res.Errmsg)
	}

	return nil
}

// BatchTaggingMembers 向微信官方给粉丝批量打标签
func (w *WxClient) BatchTaggingMembers(ctx context.Context, tagID int, openIDs []string) error {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/tags/members/batchtagging?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"openid_list": openIDs,
		"tagid":       tagID,
	}).Post(apiURL)
	if err != nil {
		return err
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(resp.Body(), &res)
	if res.Errcode != 0 {
		return fmt.Errorf("batchtagging failed: [%d] %s", res.Errcode, res.Errmsg)
	}

	return nil
}

// BatchUntaggingMembers 向微信官方批量为粉丝取消标签
func (w *WxClient) BatchUntaggingMembers(ctx context.Context, tagID int, openIDs []string) error {
	token, err := w.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/tags/members/batchuntagging?access_token=%s", token)
	resp, err := w.http.R().SetContext(ctx).SetBody(map[string]interface{}{
		"openid_list": openIDs,
		"tagid":       tagID,
	}).Post(apiURL)
	if err != nil {
		return err
	}

	var res struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(resp.Body(), &res)
	if res.Errcode != 0 {
		return fmt.Errorf("batchuntagging failed: [%d] %s", res.Errcode, res.Errmsg)
	}

	return nil
}
