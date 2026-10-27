package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
)

type MiniProgramHandler struct{}

// ListTasks 获取快速注册小程序任务列表 (租户隔离)
func (h *MiniProgramHandler) ListTasks(c *gin.Context) {
	tenantID := GetTenantID(c)
	page, pageSize := GetPaginationParams(c)

	var total int64
	query := database.DB.Model(&model.WxMiniRegisterTask{}).Where("tenant_id = ?", tenantID)
	query.Count(&total)

	var list []model.WxMiniRegisterTask
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	Page(c, list, total, page, pageSize)
}

// CreateTask 提交快速注册小程序申请 (免300元认证费)
func (h *MiniProgramHandler) CreateTask(c *gin.Context) {
	tenantID := GetTenantID(c)
	var req struct {
		Name               string `json:"name" binding:"required"`
		Code               string `json:"code" binding:"required"`
		CodeType           int    `json:"code_type"`
		LegalPersonaWechat string `json:"legal_persona_wechat" binding:"required"`
		LegalPersonaName   string `json:"legal_persona_name" binding:"required"`
		ComponentPhone     string `json:"component_phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.CodeType <= 0 {
		req.CodeType = 1
	}

	// 查找当前租户或系统启用的第三方平台
	var comp model.WxOpenComponent
	if err := database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp).Error; err != nil {
		if err := database.DB.Where("enabled = ?", true).Order("id desc").First(&comp).Error; err != nil {
			Fail(c, http.StatusBadRequest, "尚未配置微信开放平台第三方平台凭据，请先在开放平台设置中配置")
			return
		}
	}

	// 提交到微信官方接口
	param := service.FastRegisterParam{
		Name:               req.Name,
		Code:               req.Code,
		CodeType:           req.CodeType,
		LegalPersonaWechat: req.LegalPersonaWechat,
		LegalPersonaName:   req.LegalPersonaName,
		ComponentPhone:     req.ComponentPhone,
	}

	err := service.DefaultWxMiniService.CreateFastRegisterTask(c.Request.Context(), comp.ComponentAppID, param)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "微信接口返回错误: "+err.Error())
		return
	}

	// 存入数据库追踪
	task := model.WxMiniRegisterTask{
		TenantID:           tenantID,
		Name:               req.Name,
		Code:               req.Code,
		CodeType:           req.CodeType,
		LegalPersonaWechat: req.LegalPersonaWechat,
		LegalPersonaName:   req.LegalPersonaName,
		ComponentPhone:     req.ComponentPhone,
		Status:             0,
		StatusDesc:         "已向微信提交任务，微信已下发模板消息至法人微信号，等待法人人脸核身确认",
	}
	database.DB.Create(&task)

	Ok(c, task)
}

// RefreshTask 向微信官方同步查询注册任务状态
func (h *MiniProgramHandler) RefreshTask(c *gin.Context) {
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var task model.WxMiniRegisterTask
	if err := database.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&task).Error; err != nil {
		Fail(c, http.StatusNotFound, "任务不存在或无权操作")
		return
	}

	var comp model.WxOpenComponent
	database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp)
	if comp.ComponentAppID == "" {
		database.DB.Where("enabled = ?", true).Order("id desc").First(&comp)
	}

	res, err := service.DefaultWxMiniService.SearchFastRegisterTask(
		c.Request.Context(),
		comp.ComponentAppID,
		task.Name,
		task.LegalPersonaWechat,
		task.LegalPersonaName,
	)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "查询任务失败: "+err.Error())
		return
	}

	task.Status = res.Status
	task.StatusDesc = res.StatusDesc
	if res.AppID != "" {
		task.AppID = res.AppID
	}
	if res.RejectReason != "" {
		task.RejectReason = res.RejectReason
	}
	database.DB.Save(&task)

	// 若已成功，且系统内尚未添加该小程序账号，自动创建小程序账号
	if task.Status == 1 && task.AppID != "" {
		var account model.WxAccount
		err := database.DB.Where("app_id = ? AND tenant_id = ?", task.AppID, tenantID).First(&account).Error
		if err != nil {
			account = model.WxAccount{
				TenantID:       tenantID,
				Name:           task.Name + " (小程序)",
				AppID:          task.AppID,
				AccountType:    "miniprogram",
				ComponentAppID: comp.ComponentAppID,
				Enabled:        true,
				Description:    "通过第三方平台免认证费快速创建开通",
			}
			database.DB.Create(&account)
		}
	}

	Ok(c, task)
}

// GetDomain 获取小程序合法服务器域名配置
func (h *MiniProgramHandler) GetDomain(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)

	var setting model.WxMiniSetting
	err := database.DB.Where("app_id = ? AND tenant_id = ?", appid, tenantID).First(&setting).Error
	if err != nil {
		// 生成默认推荐域名 (即当前服务所在域名)
		currentHost := c.Request.Host
		if strings.Contains(currentHost, ":") {
			parts := strings.Split(currentHost, ":")
			if parts[1] == "80" || parts[1] == "443" {
				currentHost = parts[0]
			}
		}
		defaultDomain := "https://" + currentHost
		Ok(c, gin.H{
			"appid":           appid,
			"requestdomain":   defaultDomain,
			"wsrequestdomain": strings.Replace(defaultDomain, "https://", "wss://", 1),
			"uploaddomain":    defaultDomain,
			"downloaddomain":  defaultDomain,
			"webviewdomain":   defaultDomain,
		})
		return
	}

	Ok(c, setting)
}

// SetDomain 代小程序配置服务器合法域名 (同步向微信官方生效)
func (h *MiniProgramHandler) SetDomain(c *gin.Context) {
	appid := GetAppID(c)
	tenantID := GetTenantID(c)

	var req struct {
		Action         string `json:"action"` // set
		RequestDomain  string `json:"requestdomain"`
		WsDomain       string `json:"wsrequestdomain"`
		UploadDomain   string `json:"uploaddomain"`
		DownloadDomain string `json:"downloaddomain"`
		WebviewDomain  string `json:"webviewdomain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Action == "" {
		req.Action = "set"
	}

	splitClean := func(s string) []string {
		var res []string
		for _, line := range strings.Split(strings.ReplaceAll(s, ",", "\n"), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				res = append(res, trimmed)
			}
		}
		return res
	}

	reqList := splitClean(req.RequestDomain)
	wsList := splitClean(req.WsDomain)
	uploadList := splitClean(req.UploadDomain)
	downList := splitClean(req.DownloadDomain)

	// 调用微信官方 modify_domain 接口
	err := service.DefaultWxMiniService.ModifyServerDomain(c.Request.Context(), appid, req.Action, reqList, wsList, uploadList, downList)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "微信接口返回错误: "+err.Error())
		return
	}

	// 存入数据库
	var setting model.WxMiniSetting
	err = database.DB.Where("app_id = ? AND tenant_id = ?", appid, tenantID).First(&setting).Error
	if err != nil {
		setting = model.WxMiniSetting{
			TenantID:       tenantID,
			AppID:          appid,
			RequestDomain:  req.RequestDomain,
			WsDomain:       req.WsDomain,
			UploadDomain:   req.UploadDomain,
			DownloadDomain: req.DownloadDomain,
			WebviewDomain:  req.WebviewDomain,
		}
		database.DB.Create(&setting)
	} else {
		setting.RequestDomain = req.RequestDomain
		setting.WsDomain = req.WsDomain
		setting.UploadDomain = req.UploadDomain
		setting.DownloadDomain = req.DownloadDomain
		setting.WebviewDomain = req.WebviewDomain
		database.DB.Save(&setting)
	}

	Ok(c, setting)
}

// Code2Session 小程序免密登录换取 Session 凭证 (WXGate 网关直连)
func (h *MiniProgramHandler) Code2Session(c *gin.Context) {
	var req struct {
		AppID  string `json:"appid" binding:"required"`
		JsCode string `json:"js_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// 查找小程序所属第三方平台
	var account model.WxAccount
	if err := database.DB.Where("app_id = ? AND enabled = ?", req.AppID, true).First(&account).Error; err != nil {
		Fail(c, http.StatusNotFound, "小程序未在系统中注册或已禁用")
		return
	}

	componentAppID := account.ComponentAppID
	if componentAppID == "" {
		var comp model.WxOpenComponent
		database.DB.Where("enabled = ?", true).Order("id desc").First(&comp)
		componentAppID = comp.ComponentAppID
	}

	res, err := service.DefaultWxMiniService.Code2Session(c.Request.Context(), componentAppID, req.AppID, req.JsCode)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "微信登录换取失败: "+err.Error())
		return
	}

	Ok(c, res)
}

// requireThirdPartyOfficial 校验当前账号是否为「第三方平台授权托管」的公众号。
// 公众号关联小程序 (wxamplinkget / wxamplink / wxampunlink) 属于微信开放平台的第三方代调用接口
// (权限集 id=33)，官方文档「适用范围」明确：其他账号类型均不可调用，直连凭据账号调用会返回 48001。
func (h *MiniProgramHandler) requireThirdPartyOfficial(c *gin.Context) string {
	appid := GetAppID(c)
	if appid == "" {
		Fail(c, http.StatusBadRequest, "请在顶部先选择一个微信公众号")
		return ""
	}

	var account model.WxAccount
	if err := database.DB.Where("app_id = ? AND enabled = ?", appid, true).First(&account).Error; err != nil {
		Fail(c, http.StatusBadRequest, "该公众号未在系统中启用或不存在，请先在「公众号与开放平台」中完成接入")
		return ""
	}
	if account.AccountType == "miniprogram" {
		Fail(c, http.StatusBadRequest, "当前选中的是小程序账号，「关联小程序」需要切换到公众号账号后再操作")
		return ""
	}
	if !account.IsThirdPartyManaged() {
		Fail(c, http.StatusBadRequest, fmt.Sprintf(
			"公众号 %s 是「直连凭据」接入，无法查询或管理关联小程序：该能力是微信开放平台的第三方代调用接口 (权限集 id=33)，直连 access_token 调用会被微信拒绝 (48001 api unauthorized)。请在「公众号与开放平台」中改用「微信扫码一键授权接入」托管该公众号 (授权时勾选「小程序关联管理」权限集)，或直接登录微信公众平台后台手动管理关联。", appid))
		return ""
	}
	return appid
}

// GetLinkedMiniPrograms 获取当前微信公众号关联的小程序列表
func (h *MiniProgramHandler) GetLinkedMiniPrograms(c *gin.Context) {
	officialAppID := h.requireThirdPartyOfficial(c)
	if officialAppID == "" {
		return
	}

	items, err := service.DefaultWxMiniService.GetLinkedMiniPrograms(c.Request.Context(), officialAppID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取关联小程序失败: "+err.Error())
		return
	}

	Ok(c, gin.H{
		"official_appid": officialAppID,
		"items":          items,
	})
}

// LinkMiniProgram 为当前微信公众号关联指定小程序
func (h *MiniProgramHandler) LinkMiniProgram(c *gin.Context) {
	officialAppID := h.requireThirdPartyOfficial(c)
	if officialAppID == "" {
		return
	}

	var req struct {
		MiniAppID   string `json:"appid" binding:"required"`
		NotifyUsers int    `json:"notify_users"` // 0:不发, 1:发
		ShowProfile int    `json:"show_profile"` // 0:不在资料页展示, 1:展示
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "小程序 AppID 不能为空")
		return
	}

	err := service.DefaultWxMiniService.LinkMiniProgram(c.Request.Context(), officialAppID, req.MiniAppID, req.NotifyUsers, req.ShowProfile)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "微信接口返回错误: "+err.Error())
		return
	}

	Ok(c, gin.H{
		"message": "关联申请已成功提交至微信！若需小程序管理员确认，请等待对方在微信中确认",
	})
}

// UnlinkMiniProgram 解除公众号与小程序的关联
func (h *MiniProgramHandler) UnlinkMiniProgram(c *gin.Context) {
	officialAppID := h.requireThirdPartyOfficial(c)
	if officialAppID == "" {
		return
	}

	var req struct {
		MiniAppID string `json:"appid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "小程序 AppID 不能为空")
		return
	}

	err := service.DefaultWxMiniService.UnlinkMiniProgram(c.Request.Context(), officialAppID, req.MiniAppID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "解绑失败: "+err.Error())
		return
	}

	Ok(c, gin.H{
		"message": "小程序已成功解绑",
	})
}

// GetTemplateDrafts 获取代码草稿箱列表
func (h *MiniProgramHandler) GetTemplateDrafts(c *gin.Context) {
	tenantID := GetTenantID(c)
	var comp model.WxOpenComponent
	database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp)
	if comp.ComponentAppID == "" {
		database.DB.Where("enabled = ?", true).Order("id desc").First(&comp)
	}

	if comp.ComponentAppID == "" {
		Fail(c, http.StatusBadRequest, "尚未配置微信开放平台第三方平台")
		return
	}

	drafts, err := service.DefaultWxMiniService.GetTemplateDraftList(c.Request.Context(), comp.ComponentAppID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取代码草稿箱失败: "+err.Error())
		return
	}

	Ok(c, drafts)
}

// GetTemplates 获取代码模板库列表
func (h *MiniProgramHandler) GetTemplates(c *gin.Context) {
	tenantID := GetTenantID(c)
	var comp model.WxOpenComponent
	database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp)
	if comp.ComponentAppID == "" {
		database.DB.Where("enabled = ?", true).Order("id desc").First(&comp)
	}

	if comp.ComponentAppID == "" {
		Fail(c, http.StatusBadRequest, "尚未配置微信开放平台第三方平台")
		return
	}

	tplType := 0
	if v := c.Query("template_type"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			tplType = n
		}
	}

	templates, err := service.DefaultWxMiniService.GetTemplateList(c.Request.Context(), comp.ComponentAppID, tplType)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取代码模板库失败: "+err.Error())
		return
	}

	Ok(c, templates)
}

// AddTemplateFromDraft 将草稿箱添加到正式模板库
func (h *MiniProgramHandler) AddTemplateFromDraft(c *gin.Context) {
	tenantID := GetTenantID(c)
	var comp model.WxOpenComponent
	database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp)
	if comp.ComponentAppID == "" {
		database.DB.Where("enabled = ?", true).Order("id desc").First(&comp)
	}

	var req struct {
		DraftID      int `json:"draft_id" binding:"required"`
		TemplateType int `json:"template_type"` // 0: 普通, 1: 标准
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "草稿 ID 不能为空")
		return
	}

	err := service.DefaultWxMiniService.AddDraftToTemplate(c.Request.Context(), comp.ComponentAppID, req.DraftID, req.TemplateType)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "添加模板失败: "+err.Error())
		return
	}

	Ok(c, gin.H{"message": "已成功将该草稿添加至正式小程序模板库！"})
}

// DeleteTemplate 删除指定小程序代码模板
func (h *MiniProgramHandler) DeleteTemplate(c *gin.Context) {
	tenantID := GetTenantID(c)
	idStr := c.Param("id")
	var templateID int
	if _, err := fmt.Sscanf(idStr, "%d", &templateID); err != nil {
		Fail(c, http.StatusBadRequest, "无效的模板 ID")
		return
	}

	var comp model.WxOpenComponent
	database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).First(&comp)
	if comp.ComponentAppID == "" {
		database.DB.Where("enabled = ?", true).Order("id desc").First(&comp)
	}

	err := service.DefaultWxMiniService.DeleteTemplate(c.Request.Context(), comp.ComponentAppID, templateID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "删除模板失败: "+err.Error())
		return
	}

	Ok(c, gin.H{"message": "模板已删除"})
}

// ==========================================================================
// 小程序代码管理与发布 (提交代码 / 体验版 / 提审 / 发布 / 灰度 / 类目 / 体验者)
// 以下接口全部通过 TenantAppID 中间件严格校验目标小程序归属租户，防止跨租户越权
// ==========================================================================

// requireMiniApp 提取并校验目标小程序 AppID
func requireMiniApp(c *gin.Context) string {
	appid := GetAppID(c)
	if appid == "" {
		Fail(c, http.StatusBadRequest, "请先指定要操作的小程序 AppID (X-App-ID 或 appid 参数)")
		return ""
	}
	return appid
}

// ListMiniApps 获取当前租户下的账号列表 (默认只返回小程序，供「代码发布管理」选择目标小程序)
// kind=miniprogram(默认) | official | all
func (h *MiniProgramHandler) ListMiniApps(c *gin.Context) {
	tenantID := GetTenantID(c)
	kind := strings.TrimSpace(c.DefaultQuery("kind", "miniprogram"))

	var list []model.WxAccount
	database.DB.Where("tenant_id = ? AND enabled = ?", tenantID, true).
		Order("account_type asc, id asc").Find(&list)

	out := make([]gin.H, 0, len(list))
	for _, a := range list {
		isMini := a.AccountType == "miniprogram"
		switch kind {
		case "official":
			if isMini {
				continue
			}
		case "all":
		default: // miniprogram
			if !isMini {
				continue
			}
		}
		out = append(out, gin.H{
			"id":             a.ID,
			"name":           a.Name,
			"appid":          a.AppID,
			"original_id":    a.OriginalID,
			"account_type":   a.AccountType,
			"is_miniprogram": isMini,
			"managed":        a.IsThirdPartyManaged(),
			"avatar_url":     a.AvatarURL,
		})
	}
	Ok(c, out)
}

// CommitCode 使用代码模板为小程序提交代码 (wxa/commit)
func (h *MiniProgramHandler) CommitCode(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}

	var req service.CommitCodeParam
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if req.TemplateID <= 0 {
		Fail(c, http.StatusBadRequest, "代码模板 ID 不能为空，请先在「代码模板库」中获取")
		return
	}
	if strings.TrimSpace(req.UserVersion) == "" {
		Fail(c, http.StatusBadRequest, "代码版本号不能为空")
		return
	}
	if strings.TrimSpace(req.UserDesc) == "" {
		req.UserDesc = req.UserVersion
	}

	if err := service.DefaultWxMiniService.CommitCode(c.Request.Context(), appid, req); err != nil {
		Fail(c, http.StatusInternalServerError, "提交代码失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "代码已成功提交至该小程序，可直接扫码体验或提交审核"})
}

// GetCodePages 获取小程序代码的页面列表 (wxa/get_page)
func (h *MiniProgramHandler) GetCodePages(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	pages, err := service.DefaultWxMiniService.GetCodePageList(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取页面列表失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"pages": pages})
}

// GetTrialQrcode 获取体验版二维码 (wxa/get_qrcode)
func (h *MiniProgramHandler) GetTrialQrcode(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	img, err := service.DefaultWxMiniService.GetTrialQrcode(c.Request.Context(), appid, c.Query("path"))
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取体验版二维码失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"image": img})
}

// SubmitAudit 提交代码审核 (wxa/submit_audit)
func (h *MiniProgramHandler) SubmitAudit(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	var req service.SubmitAuditParam
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	auditID, err := service.DefaultWxMiniService.SubmitAudit(c.Request.Context(), appid, req)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "提交审核失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"auditid": auditID, "message": fmt.Sprintf("提审成功，微信审核单号 %d，请等待审核结果", auditID)})
}

// GetAuditStatus 查询指定审核单状态 (wxa/get_auditstatus)
func (h *MiniProgramHandler) GetAuditStatus(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	auditID, err := strconv.ParseInt(c.Query("auditid"), 10, 64)
	if err != nil || auditID <= 0 {
		Fail(c, http.StatusBadRequest, "审核单号 auditid 无效")
		return
	}
	res, err := service.DefaultWxMiniService.GetAuditStatus(c.Request.Context(), appid, auditID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "查询审核状态失败: "+err.Error())
		return
	}
	Ok(c, res)
}

// GetLatestAudit 查询最近一次提审记录 (wxa/get_latest_auditstatus)
func (h *MiniProgramHandler) GetLatestAudit(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	res, err := service.DefaultWxMiniService.GetLatestAuditStatus(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "查询最近审核记录失败: "+err.Error())
		return
	}
	Ok(c, res)
}

// UndoAudit 撤回审核 (wxa/undocodeaudit)
func (h *MiniProgramHandler) UndoAudit(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	if err := service.DefaultWxMiniService.UndoCodeAudit(c.Request.Context(), appid); err != nil {
		Fail(c, http.StatusInternalServerError, "撤回审核失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "已提交撤回审核请求"})
}

// GetCodePrivacyInfo 查询代码隐私接口检测结果 (提审前置检查)
func (h *MiniProgramHandler) GetCodePrivacyInfo(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	res, err := service.DefaultWxMiniService.GetCodePrivacyInfo(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "隐私接口检测查询失败: "+err.Error())
		return
	}
	Ok(c, res)
}

// ReleaseCode 全量发布已审核通过的版本 (wxa/release)
func (h *MiniProgramHandler) ReleaseCode(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	if err := service.DefaultWxMiniService.ReleaseCode(c.Request.Context(), appid); err != nil {
		Fail(c, http.StatusInternalServerError, "发布失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "小程序已成功全量发布上线"})
}

// GetVisitStatus 查询服务可访问状态 (wxa/getvisitstatus)
func (h *MiniProgramHandler) GetVisitStatus(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	res, err := service.DefaultWxMiniService.GetVisitStatus(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "查询访问状态失败: "+err.Error())
		return
	}
	Ok(c, res)
}

// SetVisitStatus 设置服务可访问状态 (wxa/change_visitstatus)
func (h *MiniProgramHandler) SetVisitStatus(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "action 只能为 open 或 close")
		return
	}
	if req.Action != "open" && req.Action != "close" {
		Fail(c, http.StatusBadRequest, "action 只能为 open 或 close")
		return
	}
	if err := service.DefaultWxMiniService.SetVisitStatus(c.Request.Context(), appid, req.Action); err != nil {
		Fail(c, http.StatusInternalServerError, "设置访问状态失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "服务可访问状态已更新"})
}

// GetHistoryVersions 获取可回退的历史线上版本 (wxa/revertcoderelease)
func (h *MiniProgramHandler) GetHistoryVersions(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	list, err := service.DefaultWxMiniService.GetHistoryVersions(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取历史版本失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"versions": list})
}

// RevertRelease 回退线上版本 (wxa/revertcoderelease)
func (h *MiniProgramHandler) RevertRelease(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	var req struct {
		AppVersion string `json:"app_version"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := service.DefaultWxMiniService.RevertCodeRelease(c.Request.Context(), appid, req.AppVersion); err != nil {
		Fail(c, http.StatusInternalServerError, "版本回退失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "线上版本回退指令已提交至微信"})
}

// GetSupportVersion 获取最低基础库版本与用户占比 (getweappsupportversion)
func (h *MiniProgramHandler) GetSupportVersion(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	res, err := service.DefaultWxMiniService.GetSupportVersion(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取基础库版本失败: "+err.Error())
		return
	}
	Ok(c, res)
}

// SetSupportVersion 设置最低基础库版本 (setweappsupportversion)
func (h *MiniProgramHandler) SetSupportVersion(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	var req struct {
		Version string `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "基础库版本号不能为空")
		return
	}
	if err := service.DefaultWxMiniService.SetSupportVersion(c.Request.Context(), appid, req.Version); err != nil {
		Fail(c, http.StatusInternalServerError, "设置基础库版本失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "最低基础库版本已更新"})
}

// GetCategories 获取小程序已设置的服务类目 (wxopen/getcategory)
func (h *MiniProgramHandler) GetCategories(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	res, err := service.DefaultWxMiniService.GetCategories(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取服务类目失败: "+err.Error())
		return
	}
	Ok(c, res)
}

// AddCategory 新增小程序服务类目 (wxopen/addcategory)
func (h *MiniProgramHandler) AddCategory(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	var req struct {
		Categories []service.CategoryItem `json:"categories"`
		First      int                    `json:"first"`
		Second     int                    `json:"second"`
		Certicates []service.CategoryCert `json:"certicates"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	items := req.Categories
	if len(items) == 0 {
		if req.First <= 0 || req.Second <= 0 {
			Fail(c, http.StatusBadRequest, "一级类目 ID 与二级类目 ID 不能为空")
			return
		}
		items = []service.CategoryItem{{First: req.First, Second: req.Second, Certicates: req.Certicates}}
	}
	if err := service.DefaultWxMiniService.AddCategory(c.Request.Context(), appid, items); err != nil {
		Fail(c, http.StatusInternalServerError, "新增类目失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "类目新增申请已提交，需等待微信类目审核通过"})
}

// DeleteCategory 删除小程序服务类目 (wxopen/deletecategory)
func (h *MiniProgramHandler) DeleteCategory(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	var req struct {
		First  int `json:"first" binding:"required"`
		Second int `json:"second" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "一级与二级类目 ID 均不能为空")
		return
	}
	if err := service.DefaultWxMiniService.DeleteCategory(c.Request.Context(), appid, req.First, req.Second); err != nil {
		Fail(c, http.StatusInternalServerError, "删除类目失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "服务类目已删除"})
}

// GetTesters 获取体验者列表 (wxa/memberauth)
func (h *MiniProgramHandler) GetTesters(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	list, err := service.DefaultWxMiniService.GetTesterList(c.Request.Context(), appid)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "获取体验者列表失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"members": list})
}

// BindTester 绑定体验者 (wxa/bind_tester)
func (h *MiniProgramHandler) BindTester(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	var req struct {
		WechatID string `json:"wechatid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "体验者微信号不能为空")
		return
	}
	if err := service.DefaultWxMiniService.BindTester(c.Request.Context(), appid, strings.TrimSpace(req.WechatID)); err != nil {
		Fail(c, http.StatusInternalServerError, "绑定体验者失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "体验者绑定成功"})
}

// UnbindTester 解绑体验者 (wxa/unbind_tester)
func (h *MiniProgramHandler) UnbindTester(c *gin.Context) {
	appid := requireMiniApp(c)
	if appid == "" {
		return
	}
	var req struct {
		WechatID string `json:"wechatid"`
		UserStr  string `json:"userstr"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误")
		return
	}
	if strings.TrimSpace(req.WechatID) == "" && strings.TrimSpace(req.UserStr) == "" {
		Fail(c, http.StatusBadRequest, "微信号与 userstr 至少填写一个")
		return
	}
	if err := service.DefaultWxMiniService.UnbindTester(c.Request.Context(), appid, strings.TrimSpace(req.WechatID), strings.TrimSpace(req.UserStr)); err != nil {
		Fail(c, http.StatusInternalServerError, "解绑体验者失败: "+err.Error())
		return
	}
	Ok(c, gin.H{"message": "体验者已解绑"})
}
