package handler

import (
	"net/http"
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
		Action          string `json:"action"` // set
		RequestDomain   string `json:"requestdomain"`
		WsDomain        string `json:"wsrequestdomain"`
		UploadDomain    string `json:"uploaddomain"`
		DownloadDomain  string `json:"downloaddomain"`
		WebviewDomain   string `json:"webviewdomain"`
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
