package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
)

// NotifyHandler 消息模板体系 (公众号模板消息 + 小程序/服务号订阅消息)
type NotifyHandler struct{}

// 说明: 本组接口全部位于 tenant 保护组内，必须携带 X-App-ID 或 ?appid=，
// 由 TenantAppID 中间件强制校验该账号归属当前租户，防止跨租户代发消息。

func (h *NotifyHandler) requireApp(c *gin.Context) string {
	appid := GetAppID(c)
	if appid == "" {
		Fail(c, http.StatusBadRequest, "请先在顶部选择要操作的公众号或小程序")
		return ""
	}
	return appid
}

// GetIndustry 获取账号已设置的服务行业
func (h *NotifyHandler) GetIndustry(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	res, err := service.DefaultWxNotifyService.GetIndustry(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取行业信息失败: "+err.Error())
		return
	}
	Ok(c, res)
}

// SetIndustry 设置所属行业 (更改行业将删除原行业下的模板)
func (h *NotifyHandler) SetIndustry(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		IndustryID1 string `json:"industry_id1" binding:"required"`
		IndustryID2 string `json:"industry_id2" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "一级行业代码与二级行业代码均不能为空")
		return
	}
	if err := service.DefaultWxNotifyService.SetIndustry(c.Request.Context(), appid, req.IndustryID1, req.IndustryID2); err != nil {
		Fail(c, http.StatusInternalServerError, "设置行业失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "所属行业设置成功，注意原行业下的模板可能已被微信删除"})
}

// ListTemplates 获取账号已选用的模板库列表 (附关键词解析)
func (h *NotifyHandler) ListTemplates(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	list, err := service.DefaultWxNotifyService.GetPrivateTemplates(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取模板库失败: "+err.Error())
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, t := range list {
		out = append(out, gin.H{
			"template_id":      t.TemplateID,
			"title":            t.Title,
			"primary_industry": t.PrimaryIndustry,
			"deputy_industry":  t.DeputyIndustry,
			"content":          t.Content,
			"example":          t.Example,
			"keywords":         service.ExtractTemplateKeywords(t.Content),
		})
	}
	Ok(c, out)
}

// AddTemplate 从公共模板库选用模板
func (h *NotifyHandler) AddTemplate(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		TemplateIDShort string   `json:"template_id_short" binding:"required"`
		KeywordNameList []string `json:"keyword_name_list"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "模板编号 template_id_short 不能为空")
		return
	}
	tmplID, err := service.DefaultWxNotifyService.AddTemplate(c.Request.Context(), appid, strings.TrimSpace(req.TemplateIDShort), req.KeywordNameList)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "选用模板失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"template_id": tmplID, "message": "模板已添加至账号模板库"})
}

// DeleteTemplate 删除账号下的指定模板
func (h *NotifyHandler) DeleteTemplate(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		TemplateID string `json:"template_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "模板 ID 不能为空")
		return
	}
	if err := service.DefaultWxNotifyService.DeleteTemplate(c.Request.Context(), appid, req.TemplateID); err != nil {
		Fail(c, http.StatusInternalServerError, "删除模板失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "模板已删除"})
}

// SendTemplateMsg 发送公众号模板消息
func (h *NotifyHandler) SendTemplateMsg(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req service.TemplateMsgPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if strings.TrimSpace(req.ToUser) == "" || strings.TrimSpace(req.TemplateID) == "" {
		Fail(c, http.StatusBadRequest, "接收者 openid 与模板 ID 均不能为空")
		return
	}
	if len(req.Values) == 0 {
		Fail(c, http.StatusBadRequest, "模板内容 data 不能为空")
		return
	}
	msgID, err := service.DefaultWxNotifyService.SendTemplateMsg(c.Request.Context(), appid, req)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "发送失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"msgid": msgID, "message": "模板消息已进入微信下发队列"})
}

// SendOneTimeSubscribe 下发一次性订阅消息
func (h *NotifyHandler) SendOneTimeSubscribe(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req service.OneTimeSubscribePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if strings.TrimSpace(req.ToUser) == "" || strings.TrimSpace(req.TemplateID) == "" {
		Fail(c, http.StatusBadRequest, "接收者 openid 与模板 ID 均不能为空")
		return
	}
	if err := service.DefaultWxNotifyService.SendOneTimeSubscribe(c.Request.Context(), appid, req); err != nil {
		Fail(c, http.StatusInternalServerError, "下发失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "一次性订阅消息已下发"})
}

// QueryBlocked 查询被拦截的模板消息
func (h *NotifyHandler) QueryBlocked(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		TmplMsgID string `json:"tmpl_msg_id"`
		LargestID int64  `json:"largest_id"`
		Limit     int    `json:"limit"`
	}
	_ = c.ShouldBindJSON(&req)
	res, err := service.DefaultWxNotifyService.QueryBlockedTemplateMsg(c.Request.Context(), appid, req.TmplMsgID, req.LargestID, req.Limit)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "查询被拦截模板消息失败: "+err.Error())
		return
	}
	Ok(c, res)
}

// ---------------------------- 订阅消息 ----------------------------

// SubCategories 获取订阅消息类目列表
func (h *NotifyHandler) SubCategories(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	list, err := service.DefaultWxNotifyService.GetSubscribeCategories(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取订阅消息类目失败: "+err.Error())
		return
	}
	Ok(c, list)
}

// SubPubTitles 拉取类目下的公共模板标题
func (h *NotifyHandler) SubPubTitles(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	ids := strings.TrimSpace(c.Query("ids"))
	if ids == "" {
		Fail(c, http.StatusBadRequest, "请先选择订阅消息类目 (ids)")
		return
	}
	start, _ := strconv.Atoi(c.DefaultQuery("start", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	list, count, err := service.DefaultWxNotifyService.GetPubTemplateTitles(c.Request.Context(), appid, ids, start, limit)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取公共模板库失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"total": count, "list": list})
}

// SubPubKeywords 获取公共模板的关键词
func (h *NotifyHandler) SubPubKeywords(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	tid, err := strconv.Atoi(c.Query("tid"))
	if err != nil || tid <= 0 {
		Fail(c, http.StatusBadRequest, "模板标题 tid 无效")
		return
	}
	list, err := service.DefaultWxNotifyService.GetPubTemplateKeywords(c.Request.Context(), appid, tid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取模板关键词失败: "+err.Error())
		return
	}
	Ok(c, list)
}

// SubAddTemplate 将公共模板加入我的订阅模板库
func (h *NotifyHandler) SubAddTemplate(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		TID       int    `json:"tid" binding:"required"`
		KidList   []int  `json:"kid_list" binding:"required"`
		SceneDesc string `json:"scene_desc"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "模板 tid 与关键词 kid_list 不能为空")
		return
	}
	priTmplID, err := service.DefaultWxNotifyService.AddSubscribeTemplate(c.Request.Context(), appid, req.TID, req.KidList, req.SceneDesc)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "添加订阅模板失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"priTmplId": priTmplID, "message": "已加入我的订阅模板库"})
}

// SubMyTemplates 获取我的订阅消息模板列表
func (h *NotifyHandler) SubMyTemplates(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	list, err := service.DefaultWxNotifyService.GetMySubscribeTemplates(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取我的订阅模板失败: "+err.Error())
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, t := range list {
		out = append(out, gin.H{
			"priTmplId": t.PriTmplID,
			"title":     t.Title,
			"content":   t.Content,
			"example":   t.Example,
			"type":      t.Type,
			"keywords":  service.ExtractTemplateKeywords(t.Content),
		})
	}
	Ok(c, out)
}

// SubDeleteTemplate 删除我的订阅消息模板
func (h *NotifyHandler) SubDeleteTemplate(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		PriTmplID string `json:"priTmplId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "模板 ID 不能为空")
		return
	}
	if err := service.DefaultWxNotifyService.DeleteSubscribeTemplate(c.Request.Context(), appid, req.PriTmplID); err != nil {
		Fail(c, http.StatusInternalServerError, "删除订阅模板失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "订阅模板已删除"})
}

// SubSend 发送订阅消息 (小程序 / 服务号长期订阅)
func (h *NotifyHandler) SubSend(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req service.SubscribeMsgPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if strings.TrimSpace(req.ToUser) == "" || strings.TrimSpace(req.TemplateID) == "" {
		Fail(c, http.StatusBadRequest, "接收者 openid 与模板 ID 均不能为空")
		return
	}
	if len(req.Values) == 0 {
		Fail(c, http.StatusBadRequest, "订阅消息 data 不能为空")
		return
	}
	if err := service.DefaultWxNotifyService.SendSubscribeMsg(c.Request.Context(), appid, req); err != nil {
		Fail(c, http.StatusInternalServerError, "发送失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "订阅消息已下发"})
}

// SubBizSend 发送服务号一次性订阅消息
func (h *NotifyHandler) SubBizSend(c *gin.Context) {
	appid := h.requireApp(c)
	if appid == "" {
		return
	}
	var req struct {
		service.SubscribeMsgPayload
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if strings.TrimSpace(req.ToUser) == "" || strings.TrimSpace(req.TemplateID) == "" {
		Fail(c, http.StatusBadRequest, "接收者 openid 与模板 ID 均不能为空")
		return
	}
	if err := service.DefaultWxNotifyService.SendBizSubscribeMsg(c.Request.Context(), appid, req.SubscribeMsgPayload, req.URL); err != nil {
		Fail(c, http.StatusInternalServerError, "发送失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "一次性订阅消息已下发"})
}
