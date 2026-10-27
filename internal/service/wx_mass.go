package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"path/filepath"
	"strings"
)

// ==========================================================
// 公众号群发消息 (cgi-bin/message/mass/*) + 群发图文素材
// 统一使用目标公众号的 access_token / authorizer_access_token
// 第三方平台代调用需勾选「群发消息」权限集 (id=7)
// 官方文档: /doc/service/api/message/  (masssend / sendall / get / delete / preview / speed)
// 频次限制: 订阅号每天 1 次、认证服务号每月 4 次 (超出返回 44004)
// ==========================================================

// ErrInvalidParam 参数校验错误，HTTP 层据此返回 400 (区别于微信接口调用失败的 500)
var ErrInvalidParam = errors.New("invalid parameter")

type WxMassService struct{}

var DefaultWxMassService = &WxMassService{}

// do 复用消息模板体系的凭据与统一调用逻辑 (自动适配第三方代管与直连凭据)
func (s *WxMassService) do(ctx context.Context, appid, apiPath string, body, out interface{}) error {
	return DefaultWxNotifyService.do(ctx, appid, "POST", apiPath, nil, body, out)
}

// MassMessage 群发消息内容 (扁平入参，由后端组装为微信要求的结构)
type MassMessage struct {
	MsgType            string   `json:"msgtype"`     // text/mpnews/image/voice/mpvideo/wxcard
	Text               string   `json:"text"`        // msgtype=text 时必填
	MediaID            string   `json:"media_id"`    // mpnews/voice/mpvideo 素材 ID
	MediaIDs           []string `json:"media_ids"`   // image 贴图消息的素材 ID 列表
	CardID             string   `json:"card_id"`     // wxcard 卡券 ID
	Title              string   `json:"title"`       // image/mpvideo 标题
	Recommend          string   `json:"recommend"`   // image 推荐语
	Description        string   `json:"description"` // mpvideo 描述
	NeedOpenComment    int      `json:"need_open_comment"`
	OnlyFansCanComment int      `json:"only_fans_can_comment"`
}

func (m MassMessage) normalizedType() (string, error) {
	t := strings.ToLower(strings.TrimSpace(m.MsgType))
	if t == "" {
		return "", fmt.Errorf("%w: 请选择群发的消息类型", ErrInvalidParam)
	}
	switch t {
	case "text", "mpnews", "news", "image", "voice", "mpvideo", "wxcard":
	default:
		return "", fmt.Errorf("%w: 不支持的消息类型: %s (可选 text/mpnews/image/voice/mpvideo/wxcard)", ErrInvalidParam, m.MsgType)
	}
	if t == "news" {
		t = "mpnews"
	}
	return t, nil
}

// massBody 组装正式群发的消息体 (send / sendall)，贴图消息使用 images.media_ids
func (m MassMessage) massBody() (map[string]interface{}, error) {
	t, err := m.normalizedType()
	if err != nil {
		return nil, err
	}
	switch t {
	case "text":
		if strings.TrimSpace(m.Text) == "" {
			return nil, fmt.Errorf("%w: 文本消息内容不能为空", ErrInvalidParam)
		}
		return map[string]interface{}{"text": map[string]interface{}{"content": strings.TrimSpace(m.Text)}}, nil
	case "mpnews":
		if strings.TrimSpace(m.MediaID) == "" {
			return nil, fmt.Errorf("%w: 图文消息必须提供 media_id (可先新建图文素材或使用临时素材)", ErrInvalidParam)
		}
		return map[string]interface{}{"mpnews": map[string]interface{}{"media_id": strings.TrimSpace(m.MediaID)}}, nil
	case "voice":
		if strings.TrimSpace(m.MediaID) == "" {
			return nil, fmt.Errorf("%w: 语音消息必须提供 media_id", ErrInvalidParam)
		}
		return map[string]interface{}{"voice": map[string]interface{}{"media_id": strings.TrimSpace(m.MediaID)}}, nil
	case "mpvideo":
		if strings.TrimSpace(m.MediaID) == "" {
			return nil, fmt.Errorf("%w: 视频消息必须提供 media_id", ErrInvalidParam)
		}
		v := map[string]interface{}{"media_id": strings.TrimSpace(m.MediaID)}
		if strings.TrimSpace(m.Title) != "" {
			v["title"] = strings.TrimSpace(m.Title)
		}
		if strings.TrimSpace(m.Description) != "" {
			v["description"] = strings.TrimSpace(m.Description)
		}
		return map[string]interface{}{"mpvideo": v}, nil
	case "image":
		ids := make([]string, 0, len(m.MediaIDs))
		for _, id := range m.MediaIDs {
			if s := strings.TrimSpace(id); s != "" {
				ids = append(ids, s)
			}
		}
		if len(ids) == 0 && strings.TrimSpace(m.MediaID) != "" {
			ids = []string{strings.TrimSpace(m.MediaID)}
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("%w: 图片(贴图)消息至少需要提供一个 media_id", ErrInvalidParam)
		}
		v := map[string]interface{}{"media_ids": ids}
		if strings.TrimSpace(m.Title) != "" {
			v["title"] = strings.TrimSpace(m.Title)
		}
		if strings.TrimSpace(m.Recommend) != "" {
			v["recommend"] = strings.TrimSpace(m.Recommend)
		}
		v["need_open_comment"] = m.NeedOpenComment
		v["only_fans_can_comment"] = m.OnlyFansCanComment
		return map[string]interface{}{"images": v}, nil
	default: // wxcard
		if strings.TrimSpace(m.CardID) == "" {
			return nil, fmt.Errorf("%w: 卡券消息必须提供 card_id", ErrInvalidParam)
		}
		return map[string]interface{}{"wxcard": map[string]interface{}{"card_id": strings.TrimSpace(m.CardID)}}, nil
	}
}

// previewBody 组装预览消息体，预览接口使用 image.media_id 单图形式
func (m MassMessage) previewBody() (map[string]interface{}, error) {
	t, err := m.normalizedType()
	if err != nil {
		return nil, err
	}
	if t != "image" {
		return m.massBody()
	}
	id := strings.TrimSpace(m.MediaID)
	if id == "" && len(m.MediaIDs) > 0 {
		id = strings.TrimSpace(m.MediaIDs[0])
	}
	if id == "" {
		return nil, fmt.Errorf("预览图片消息必须提供 media_id")
	}
	return map[string]interface{}{"image": map[string]interface{}{"media_id": id}}, nil
}

// appendSendOptions 追加 clientmsgid 与原创转载策略
func (m MassMessage) applyOptions(body map[string]interface{}, clientMsgID string, ignoreReprint int) error {
	if cid := strings.TrimSpace(clientMsgID); cid != "" {
		if len(cid) > 32 {
			return fmt.Errorf("%w: clientmsgid 长度不能超过 32 字节", ErrInvalidParam)
		}
		body["clientmsgid"] = cid
	}
	if t, _ := m.normalizedType(); t == "mpnews" {
		if ignoreReprint == 1 {
			body["send_ignore_reprint"] = 1
		} else {
			body["send_ignore_reprint"] = 0
		}
	}
	return nil
}

// SendByOpenids 按 openid 列表群发 (cgi-bin/message/mass/send)，openid 数量 2~10000
func (s *WxMassService) SendByOpenids(ctx context.Context, appid string, openids []string, m MassMessage, clientMsgID string, ignoreReprint int) (map[string]interface{}, error) {
	ids := make([]string, 0, len(openids))
	for _, o := range openids {
		if v := strings.TrimSpace(o); v != "" {
			ids = append(ids, v)
		}
	}
	if len(ids) < 2 {
		return nil, fmt.Errorf("%w: openid 列表至少需要 2 个 (可在「粉丝与标签管理」中勾选粉丝)", ErrInvalidParam)
	}
	if len(ids) > 10000 {
		return nil, fmt.Errorf("%w: openid 列表最多 10000 个，当前 %d 个", ErrInvalidParam, len(ids))
	}
	content, err := m.massBody()
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{"touser": ids, "msgtype": mustType(m)}
	for k, v := range content {
		body[k] = v
	}
	if err := m.applyOptions(body, clientMsgID, ignoreReprint); err != nil {
		return nil, err
	}

	var res map[string]interface{}
	if err := s.do(ctx, appid, "/cgi-bin/message/mass/send", body, &res); err != nil {
		return nil, err
	}
	res["audience"] = fmt.Sprintf("指定 openid %d 个", len(ids))
	return res, nil
}

// SendAll 按标签或全体群发 (cgi-bin/message/mass/sendall)
func (s *WxMassService) SendAll(ctx context.Context, appid string, isToAll bool, tagID string, m MassMessage, clientMsgID string, ignoreReprint int) (map[string]interface{}, error) {
	filter := map[string]interface{}{"is_to_all": isToAll}
	if isToAll {
		tagID = ""
	} else {
		tag := strings.TrimSpace(tagID)
		if tag == "" {
			return nil, fmt.Errorf("%w: 按标签群发时必须选择标签 (或勾选「发送给全部用户」)", ErrInvalidParam)
		}
		tagID = tag
		filter["tag_id"] = tag
	}
	content, err := m.massBody()
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{"filter": filter, "msgtype": mustType(m)}
	for k, v := range content {
		body[k] = v
	}
	if err := m.applyOptions(body, clientMsgID, ignoreReprint); err != nil {
		return nil, err
	}

	var res map[string]interface{}
	if err := s.do(ctx, appid, "/cgi-bin/message/mass/sendall", body, &res); err != nil {
		return nil, err
	}
	if isToAll {
		res["audience"] = "全部用户"
	} else {
		res["audience"] = "标签 " + tagID
	}
	return res, nil
}

// Preview 预览群发消息 (cgi-bin/message/mass/preview)，touser 与 towxname 二选一
func (s *WxMassService) Preview(ctx context.Context, appid, toUser, toWxName string, m MassMessage) error {
	if strings.TrimSpace(toUser) == "" && strings.TrimSpace(toWxName) == "" {
		return fmt.Errorf("%w: 请填写接收用户的 openid 或接收人的微信号", ErrInvalidParam)
	}
	content, err := m.previewBody()
	if err != nil {
		return err
	}
	body := map[string]interface{}{"msgtype": mustType(m)}
	if v := strings.TrimSpace(toUser); v != "" {
		body["touser"] = v
	} else {
		body["towxname"] = strings.TrimSpace(toWxName)
	}
	for k, v := range content {
		body[k] = v
	}
	return s.do(ctx, appid, "/cgi-bin/message/mass/preview", body, nil)
}

// GetStatus 查询群发消息发送状态 (cgi-bin/message/mass/get)
// msg_status: SEND_SUCCESS 发送成功 / SENDING 发送中 / SEND_FAIL 发送失败 / DELETE 已删除
func (s *WxMassService) GetStatus(ctx context.Context, appid, msgID string) (map[string]interface{}, error) {
	if strings.TrimSpace(msgID) == "" {
		return nil, fmt.Errorf("%w: 请填写群发消息返回的 msg_id", ErrInvalidParam)
	}
	var res map[string]interface{}
	body := map[string]interface{}{"msg_id": strings.TrimSpace(msgID)}
	if err := s.do(ctx, appid, "/cgi-bin/message/mass/get", body, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// Delete 删除群发消息 (cgi-bin/message/mass/delete)，msg_id 与 url 必须有一个
func (s *WxMassService) Delete(ctx context.Context, appid, msgID, contentURL string, articleIdx int) error {
	if strings.TrimSpace(msgID) == "" && strings.TrimSpace(contentURL) == "" {
		return fmt.Errorf("%w: 请提供要删除的消息 msg_id 或图文 url", ErrInvalidParam)
	}
	body := map[string]interface{}{"article_idx": articleIdx}
	if v := strings.TrimSpace(msgID); v != "" {
		body["msg_id"] = v
	} else {
		body["url"] = strings.TrimSpace(contentURL)
		body["article_idx"] = articleIdx
	}
	return s.do(ctx, appid, "/cgi-bin/message/mass/delete", body, nil)
}

// SetSpeed 设置群发速度 (cgi-bin/message/mass/speed/set)，0=80w/分 1=60w 2=45w 3=30w 4=10w
func (s *WxMassService) SetSpeed(ctx context.Context, appid string, speed int) error {
	if speed < 0 || speed > 4 {
		return fmt.Errorf("%w: 群发速度档位必须在 0~4 之间", ErrInvalidParam)
	}
	return s.do(ctx, appid, "/cgi-bin/message/mass/speed/set", map[string]interface{}{"speed": speed}, nil)
}

// GetSpeed 获取群发速度 (cgi-bin/message/mass/speed/get)
func (s *WxMassService) GetSpeed(ctx context.Context, appid string) (map[string]interface{}, error) {
	var res map[string]interface{}
	if err := s.do(ctx, appid, "/cgi-bin/message/mass/speed/get", map[string]interface{}{}, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// ---------------------------- 群发图文素材 ----------------------------

// UploadThumbMaterial 上传永久图片素材 (cgi-bin/material/add_material?type=image)
// 群发图文 (media/uploadnews) 的 thumb_media_id 必须是微信素材库中的真实 media_id，
// 本系统素材库中的 mock_ 前缀 ID 为本地占位，不能用于群发。
func (s *WxMassService) UploadThumbMaterial(ctx context.Context, appid, filename string, data []byte) (string, string, error) {
	if len(data) == 0 {
		return "", "", fmt.Errorf("%w: 图片内容为空", ErrInvalidParam)
	}
	if len(data) > 10*1024*1024 {
		return "", "", fmt.Errorf("%w: 永久图片素材大小不能超过 10MB，当前 %.2fMB", ErrInvalidParam, float64(len(data))/1024/1024)
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		return "", "", fmt.Errorf("%w: 图片素材仅支持 jpg/png/gif 格式", ErrInvalidParam)
	}

	token, err := DefaultWxMiniService.tokenForMiniApp(ctx, appid)
	if err != nil {
		return "", "", err
	}
	q := url.Values{"type": {"image"}}
	u := "https://api.weixin.qq.com/cgi-bin/material/add_material?access_token=" + url.QueryEscape(token) + "&" + q.Encode()
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "image/jpeg"
	}
	resp, err := DefaultWxMiniService.http.R().SetContext(ctx).
		SetMultipartField("media", filename, ct, bytes.NewReader(data)).
		Post(u)
	if err != nil {
		return "", "", fmt.Errorf("请求微信接口网络错误: %w", err)
	}
	if resp == nil {
		return "", "", fmt.Errorf("微信接口无响应内容")
	}
	raw := resp.Body()
	var base struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
		MediaID string `json:"media_id"`
		URL     string `json:"url"`
	}
	if e := json.Unmarshal(raw, &base); e != nil {
		return "", "", fmt.Errorf("解析微信响应失败: %v (原始响应: %s)", e, snippet(raw))
	}
	if base.Errcode != 0 {
		return "", "", fmt.Errorf("[%d] %s", base.Errcode, base.Errmsg)
	}
	if base.MediaID == "" {
		return "", "", fmt.Errorf("微信未返回 media_id: %s", snippet(raw))
	}
	return base.MediaID, base.URL, nil
}

// MassNewsArticle 图文消息单条图文 (media/uploadnews 的 articles 元素)
type MassNewsArticle struct {
	Title              string `json:"title"`
	Author             string `json:"author"`
	ThumbMediaID       string `json:"thumb_media_id"`
	Content            string `json:"content"`
	ContentSourceURL   string `json:"content_source_url"`
	Digest             string `json:"digest"`
	ShowCoverPic       int    `json:"show_cover_pic"`
	NeedOpenComment    int    `json:"need_open_comment"`
	OnlyFansCanComment int    `json:"only_fans_can_comment"`
}

// UploadNewsImage 上传图文正文内图片 (cgi-bin/media/uploadimg)，返回图片 URL (不占素材库配额)
func (s *WxMassService) UploadNewsImage(ctx context.Context, appid, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("%w: 图片内容为空", ErrInvalidParam)
	}
	if len(data) > 1024*1024 {
		return "", fmt.Errorf("%w: 图片大小必须小于 1MB，当前 %.2fMB", ErrInvalidParam, float64(len(data))/1024/1024)
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return "", fmt.Errorf("%w: 该接口仅支持 jpg/png 格式图片", ErrInvalidParam)
	}

	token, err := DefaultWxMiniService.tokenForMiniApp(ctx, appid)
	if err != nil {
		return "", err
	}
	u := "https://api.weixin.qq.com/cgi-bin/media/uploadimg?access_token=" + url.QueryEscape(token)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "image/jpeg"
	}
	resp, err := DefaultWxMiniService.http.R().SetContext(ctx).
		SetMultipartField("media", filename, ct, bytes.NewReader(data)).
		Post(u)
	if err != nil {
		return "", fmt.Errorf("请求微信接口网络错误: %w", err)
	}
	if resp == nil {
		return "", fmt.Errorf("微信接口无响应内容")
	}
	raw := resp.Body()
	var base struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
		URL     string `json:"url"`
	}
	if e := json.Unmarshal(raw, &base); e != nil {
		return "", fmt.Errorf("解析微信响应失败: %v (原始响应: %s)", e, snippet(raw))
	}
	if base.Errcode != 0 {
		return "", fmt.Errorf("[%d] %s", base.Errcode, base.Errmsg)
	}
	if base.URL == "" {
		return "", fmt.Errorf("微信未返回图片 URL: %s", snippet(raw))
	}
	return base.URL, nil
}

// UploadNews 新建图文消息素材 (cgi-bin/media/uploadnews)，返回可用于群发的 media_id
func (s *WxMassService) UploadNews(ctx context.Context, appid string, articles []MassNewsArticle) (string, error) {
	if len(articles) == 0 {
		return "", fmt.Errorf("%w: 请至少填写一条图文", ErrInvalidParam)
	}
	if len(articles) > 8 {
		return "", fmt.Errorf("%w: 一个图文消息最多 8 条图文，当前 %d 条", ErrInvalidParam, len(articles))
	}
	list := make([]map[string]interface{}, 0, len(articles))
	for i, a := range articles {
		if strings.TrimSpace(a.Title) == "" {
			return "", fmt.Errorf("%w: 第 %d 条图文缺少标题", ErrInvalidParam, i+1)
		}
		if strings.TrimSpace(a.Content) == "" {
			return "", fmt.Errorf("%w: 第 %d 条图文缺少正文内容", ErrInvalidParam, i+1)
		}
		if strings.TrimSpace(a.ThumbMediaID) == "" {
			return "", fmt.Errorf("%w: 第 %d 条图文缺少封面 (thumb_media_id)，可在「素材多媒体」中上传永久图片素材获取", ErrInvalidParam, i+1)
		}
		item := map[string]interface{}{
			"title":                 strings.TrimSpace(a.Title),
			"thumb_media_id":        strings.TrimSpace(a.ThumbMediaID),
			"content":               a.Content,
			"show_cover_pic":        a.ShowCoverPic,
			"need_open_comment":     a.NeedOpenComment,
			"only_fans_can_comment": a.OnlyFansCanComment,
		}
		if v := strings.TrimSpace(a.Author); v != "" {
			item["author"] = v
		}
		if v := strings.TrimSpace(a.Digest); v != "" {
			item["digest"] = v
		}
		if v := strings.TrimSpace(a.ContentSourceURL); v != "" {
			item["content_source_url"] = v
		}
		list = append(list, item)
	}

	var res struct {
		Type      string `json:"type"`
		MediaID   string `json:"media_id"`
		CreatedAt string `json:"created_at"`
	}
	body := map[string]interface{}{"articles": list}
	if err := s.do(ctx, appid, "/cgi-bin/media/uploadnews", body, &res); err != nil {
		return "", err
	}
	if res.MediaID == "" {
		return "", fmt.Errorf("微信未返回图文 media_id")
	}
	return res.MediaID, nil
}

// mustType 群发内部使用的类型转换，调用前已经过 normalizedType 校验
func mustType(m MassMessage) string {
	t, err := m.normalizedType()
	if err != nil {
		return strings.ToLower(strings.TrimSpace(m.MsgType))
	}
	return t
}
