package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
)

// requireSyncedTagID 校验「按标签群发」的 tag_id：必须是当前公众号 + 当前租户下、
// 且已真实同步到微信官方的标签。「仅本地」标签 (负的占位 ID) 微信并不认识，
// 过去直接透传只会收到一句难以理解的官方报错。
func requireSyncedTagID(c *gin.Context, appid, raw string) bool {
	tagID, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || tagID == 0 {
		Fail(c, http.StatusBadRequest, "tag_id 不合法，请在下拉框重新选择标签")
		return false
	}
	var tag model.WxTag
	if err := database.DB.Where("tag_id = ? AND app_id = ? AND tenant_id = ?", tagID, appid, GetTenantID(c)).
		First(&tag).Error; err != nil {
		Fail(c, http.StatusBadRequest, "该标签不存在或不属于当前公众号 (tag_id="+raw+")，请先到「粉丝与标签」创建标签")
		return false
	}
	if !tag.Synced || service.IsLocalOnlyFanTagID(tag.TagID) {
		Fail(c, http.StatusBadRequest, "标签「"+tag.Name+"」目前仅在本地记录，请先在「标签管理」点「重新同步到微信」，再按标签群发")
		return false
	}
	return true
}

// failMass 统一错误响应: 参数校验类错误返回 400，微信接口/网络类错误返回 500
func failMass(c *gin.Context, prefix string, err error) {
	if errors.Is(err, service.ErrInvalidParam) {
		Fail(c, http.StatusBadRequest, strings.TrimPrefix(err.Error(), "invalid parameter: "))
		return
	}
	Fail(c, http.StatusInternalServerError, prefix+": "+err.Error())
}

// MassHandler 公众号群发消息与群发图文素材
// 说明: 全部接口位于 tenant 保护组内，需携带 X-App-ID 或 ?appid=，
// 由 TenantAppID 中间件校验账号归属当前租户，防止跨租户群发。
type MassHandler struct{}

// 群发消息请求体 (扁平参数，由 service 层组装微信结构)
type massRequest struct {
	service.MassMessage
	Touser        []string `json:"touser"`
	TouserText    string   `json:"touser_text"` // 兼容前端多行文本粘贴
	IsToAll       bool     `json:"is_to_all"`
	TagID         string   `json:"tag_id"`
	ClientMsgID   string   `json:"clientmsgid"`
	IgnoreReprint int      `json:"send_ignore_reprint"`
}

func (r massRequest) openidList() []string {
	list := append([]string{}, r.Touser...)
	if len(list) == 0 && strings.TrimSpace(r.TouserText) != "" {
		list = strings.FieldsFunc(r.TouserText, func(c rune) bool {
			return c == '\n' || c == '\r' || c == ',' || c == ' ' || c == '\t' || c == ';' || c == '；' || c == '，'
		})
	}
	return list
}

func (h *MassHandler) requireApp(c *gin.Context) string {
	appid := GetAppID(c)
	if appid == "" {
		Fail(c, http.StatusBadRequest, "请先在顶部选择要操作的公众号")
		return ""
	}
	return appid
}

// Send 按 openid 列表群发 (cgi-bin/message/mass/send)
func (h *MassHandler) Send(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req massRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	res, err := service.DefaultWxMassService.SendByOpenids(c.Request.Context(), appid, req.openidList(), req.MassMessage, req.ClientMsgID, req.IgnoreReprint)
	if err != nil {
		failMass(c, "群发任务提交失败", err)
		return
	}
	Ok(c, gin.H{"result": res, "message": "群发任务已提交，稍后可用 msg_id 查询发送状态"})
}

// SendAll 按标签或全体群发 (cgi-bin/message/mass/sendall)
func (h *MassHandler) SendAll(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req massRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if !req.IsToAll {
		if !requireSyncedTagID(c, appid, req.TagID) {
			return
		}
	}
	res, err := service.DefaultWxMassService.SendAll(c.Request.Context(), appid, req.IsToAll, req.TagID, req.MassMessage, req.ClientMsgID, req.IgnoreReprint)
	if err != nil {
		failMass(c, "群发任务提交失败", err)
		return
	}
	Ok(c, gin.H{"result": res, "message": "群发任务已提交，注意微信管理员确认与审核延迟"})
}

// Preview 预览群发消息 (cgi-bin/message/mass/preview)
func (h *MassHandler) Preview(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		service.MassMessage
		Touser   string `json:"touser"`
		ToWxName string `json:"towxname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if err := service.DefaultWxMassService.Preview(c.Request.Context(), appid, req.Touser, req.ToWxName, req.MassMessage); err != nil {
		failMass(c, "预览失败", err)
		return
	}
	Ok(c, gin.H{"message": "预览消息已下发到指定用户"})
}

// Status 查询群发消息发送状态 (cgi-bin/message/mass/get)
func (h *MassHandler) Status(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		MsgID string `json:"msg_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.MsgID) == "" {
		Fail(c, http.StatusBadRequest, "请填写要查询的群发消息 msg_id")
		return
	}
	res, err := service.DefaultWxMassService.GetStatus(c.Request.Context(), appid, req.MsgID)
	if err != nil {
		failMass(c, "查询群发状态失败", err)
		return
	}
	Ok(c, res)
}

// Delete 删除群发消息 (cgi-bin/message/mass/delete)
func (h *MassHandler) Delete(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		MsgID      string `json:"msg_id"`
		URL        string `json:"url"`
		ArticleIdx int    `json:"article_idx"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if err := service.DefaultWxMassService.Delete(c.Request.Context(), appid, req.MsgID, req.URL, req.ArticleIdx); err != nil {
		failMass(c, "删除群发消息失败", err)
		return
	}
	Ok(c, gin.H{"message": "群发消息已删除 (未发送的用户不再收到)"})
}

// SetSpeed 设置群发速度 (cgi-bin/message/mass/speed/set)
func (h *MassHandler) SetSpeed(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		Speed int `json:"speed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写速度档位 speed")
		return
	}
	if err := service.DefaultWxMassService.SetSpeed(c.Request.Context(), appid, req.Speed); err != nil {
		failMass(c, "设置群发速度失败", err)
		return
	}
	Ok(c, gin.H{"message": "群发速度已设置", "speed": req.Speed})
}

// GetSpeed 获取群发速度 (cgi-bin/message/mass/speed/get)
func (h *MassHandler) GetSpeed(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	res, err := service.DefaultWxMassService.GetSpeed(c.Request.Context(), appid)
	if err != nil {
		failMass(c, "获取群发速度失败", err)
		return
	}
	Ok(c, res)
}

// UploadNewsImage 上传图文正文内图片 (cgi-bin/media/uploadimg)，返回图片 URL
func (h *MassHandler) UploadNewsImage(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, "请选择要上传的图片文件 (jpg/png)")
		return
	}
	f, err := file.Open()
	if err != nil {
		failMass(c, "读取上传文件失败", err)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 2*1024*1024))
	if err != nil {
		failMass(c, "读取上传文件失败", err)
		return
	}
	imgURL, err := service.DefaultWxMassService.UploadNewsImage(c.Request.Context(), appid, file.Filename, data)
	if err != nil {
		failMass(c, "上传图片失败", err)
		return
	}
	Ok(c, gin.H{"url": imgURL, "message": "图片已上传，可在图文正文中使用该 URL (不占用素材库配额)"})
}

// UploadThumb 上传永久图片素材，返回群发图文所需的真实 media_id
func (h *MassHandler) UploadThumb(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, "请选择要上传的图片文件 (jpg/png/gif)")
		return
	}
	f, err := file.Open()
	if err != nil {
		failMass(c, "读取上传文件失败", err)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 11*1024*1024))
	if err != nil {
		failMass(c, "读取上传文件失败", err)
		return
	}
	mediaID, imgURL, err := service.DefaultWxMassService.UploadThumbMaterial(c.Request.Context(), appid, file.Filename, data)
	if err != nil {
		failMass(c, "上传素材图片失败", err)
		return
	}
	Ok(c, gin.H{"media_id": mediaID, "url": imgURL, "message": "封面图片已存入微信素材库，可用作图文素材的 thumb_media_id"})
}

// UploadNews 新建图文消息素材 (cgi-bin/media/uploadnews)，返回群发用 media_id
func (h *MassHandler) UploadNews(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		Articles []service.MassNewsArticle `json:"articles"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	mediaID, err := service.DefaultWxMassService.UploadNews(c.Request.Context(), appid, req.Articles)
	if err != nil {
		failMass(c, "新建图文素材失败", err)
		return
	}
	Ok(c, gin.H{"media_id": mediaID, "message": "图文素材已创建，可用该 media_id 进行 mpnews 群发"})
}

// GetServiceCard 查询服务提醒卡片状态 (wxa/get_user_notify)
func (h *MassHandler) GetServiceCard(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		OpenID     string `json:"openid"`
		NotifyCode string `json:"notify_code"`
		NotifyType int    `json:"notify_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	res, err := service.DefaultWxNotifyService.GetServiceCard(c.Request.Context(), appid, req.OpenID, req.NotifyCode, req.NotifyType)
	if err != nil {
		failMass(c, "查询服务卡片失败", err)
		return
	}
	Ok(c, res)
}

// SetServiceCard 激活与更新服务提醒卡片 (wxa/set_user_notify)
func (h *MassHandler) SetServiceCard(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		OpenID      string `json:"openid"`
		NotifyCode  string `json:"notify_code"`
		NotifyType  string `json:"notify_type"`
		ContentJSON string `json:"content_json"`
		CheckJSON   string `json:"check_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	notifyType, err := strconv.Atoi(strings.TrimSpace(req.NotifyType))
	if err != nil {
		Fail(c, http.StatusBadRequest, "卡片 id (notify_type) 必须是数字，如 1001、2001")
		return
	}
	if err := service.DefaultWxNotifyService.SetServiceCard(c.Request.Context(), appid, req.OpenID, req.NotifyCode, notifyType, req.ContentJSON, req.CheckJSON); err != nil {
		failMass(c, "服务卡片激活/更新失败", err)
		return
	}
	Ok(c, gin.H{"message": "服务卡片已激活或更新，微信将按卡片状态向用户推送服务提醒"})
}
