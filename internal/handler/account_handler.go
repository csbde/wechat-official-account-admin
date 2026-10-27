package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
)

type AccountHandler struct{}

type CreateAccountRequest struct {
	Name           string `json:"name" binding:"required"`
	AppID          string `json:"appid" binding:"required"`
	AppSecret      string `json:"appsecret" binding:"required"`
	Token          string `json:"token"`
	EncodingAESKey string `json:"encoding_aes_key"`
	OriginalID     string `json:"original_id"`
	Description    string `json:"description"`
}

type UpdateAccountRequest struct {
	Name           string `json:"name"`
	AppSecret      string `json:"appsecret"`
	Token          string `json:"token"`
	EncodingAESKey string `json:"encoding_aes_key"`
	OriginalID     string `json:"original_id"`
	Description    string `json:"description"`
	Enabled        *bool  `json:"enabled"`
}

// List 获取公众号列表
// 默认只返回「当前租户视角」内的账号 (超级管理员可通过 X-Tenant-ID 切换视角)；
// 仅当超级管理员显式携带 ?all=1 时才跨租户总览，并在结果中标注归属租户，避免混淆。
func (h *AccountHandler) List(c *gin.Context) {
	tenantID := GetTenantID(c)
	var list []model.WxAccount

	query := database.DB.Model(&model.WxAccount{})
	scopeAll := IsSuperAdmin(c) && c.Query("all") == "1"
	if !scopeAll {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		like := "%" + kw + "%"
		query = query.Where("name LIKE ? OR app_id LIKE ?", like, like)
	}

	if err := query.Order("is_default desc, id desc").Find(&list).Error; err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	role, _ := c.Get("role")
	isAdmin := role == "admin" || role == "superadmin"

	tenantMap := make(map[uint]model.Tenant)
	if scopeAll {
		var tenants []model.Tenant
		database.DB.Model(&model.Tenant{}).Select("id", "name", "code").Find(&tenants)
		for _, t := range tenants {
			tenantMap[t.ID] = t
		}
	}

	type AccountVO struct {
		model.WxAccount
		CallbackURL string `json:"callback_url"`
		TenantName  string `json:"tenant_name,omitempty"`
		TenantCode  string `json:"tenant_code,omitempty"`
		Manageable  bool   `json:"manageable"`
	}

	result := make([]AccountVO, len(list))
	for i, acc := range list {
		if !isAdmin {
			acc.AppSecret = "******"
			acc.EncodingAESKey = "******"
		}
		cb := BuildPublicURL(c, fmt.Sprintf("/wxmp/%s/handler", acc.AppID))
		vo := AccountVO{
			WxAccount:   acc,
			CallbackURL: cb,
			Manageable:  acc.TenantID == tenantID,
		}
		if t, ok := tenantMap[acc.TenantID]; ok {
			vo.TenantName = t.Name
			vo.TenantCode = t.Code
		}
		result[i] = vo
	}

	Ok(c, result)
}

// Create 新建微信公众号 (强制绑定当前租户)
func (h *AccountHandler) Create(c *gin.Context) {
	tenantID := GetTenantID(c)
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var count int64
	database.DB.Model(&model.WxAccount{}).Where("app_id = ? AND tenant_id = ?", req.AppID, tenantID).Count(&count)
	if count > 0 {
		Fail(c, http.StatusBadRequest, "该 AppID 在当前租户下已存在")
		return
	}

	// 租户隔离允许不同租户录入相同 AppID，但同一微信号被两个组织分别接入极易引发串号，
	// 因此默认拒绝，仅超级管理员可在确认风险后用 force 强制录入。
	var crossCount int64
	database.DB.Model(&model.WxAccount{}).Where("app_id = ?", req.AppID).Count(&crossCount)
	if crossCount > 0 && c.Query("force") != "1" {
		var other model.WxAccount
		database.DB.Where("app_id = ?", req.AppID).First(&other)
		var otherTenant model.Tenant
		database.DB.First(&otherTenant, other.TenantID)
		Fail(c, http.StatusBadRequest, fmt.Sprintf(
			"AppID %s 已在租户「%s(%s)」中录入。多租户数据相互隔离，同一微信账号跨租户重复接入会导致回复与菜单互相干扰；请切换到该租户视角维护，或确认无误后带 force=1 重试",
			req.AppID, otherTenant.Name, otherTenant.Code))
		return
	}

	var total int64
	database.DB.Model(&model.WxAccount{}).Where("tenant_id = ?", tenantID).Count(&total)

	account := model.WxAccount{
		TenantID:       tenantID, // 强制隔离归属
		Name:           req.Name,
		AppID:          req.AppID,
		AppSecret:      req.AppSecret,
		Token:          req.Token,
		EncodingAESKey: req.EncodingAESKey,
		OriginalID:     req.OriginalID,
		Description:    req.Description,
		Enabled:        true,
		IsDefault:      total == 0,
	}

	if err := database.DB.Create(&account).Error; err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	Ok(c, account)
}

// Update 更新公众号配置 (防越权校验)
func (h *AccountHandler) Update(c *gin.Context) {
	tenantID := GetTenantID(c)
	id := c.Param("id")

	// 严格按当前租户视角隔离：超级管理员如需维护其他租户，请先切换租户视角，避免误操作
	var account model.WxAccount
	if err := database.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&account).Error; err != nil {
		Fail(c, http.StatusNotFound, "公众号不存在或不属于当前租户视角 (超级管理员请先在顶栏切换租户视角)")
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name != "" {
		account.Name = req.Name
	}
	if req.AppSecret != "" && req.AppSecret != "******" {
		account.AppSecret = req.AppSecret
	}
	if req.Token != "" {
		account.Token = req.Token
	}
	if req.EncodingAESKey != "" && req.EncodingAESKey != "******" {
		account.EncodingAESKey = req.EncodingAESKey
	}
	if req.OriginalID != "" {
		account.OriginalID = req.OriginalID
	}
	if req.Description != "" {
		account.Description = req.Description
	}
	if req.Enabled != nil {
		account.Enabled = *req.Enabled
	}

	if err := database.DB.Save(&account).Error; err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	Ok(c, account)
}

// Delete 删除公众号 (防越权校验)
func (h *AccountHandler) Delete(c *gin.Context) {
	tenantID := GetTenantID(c)
	id := c.Param("id")

	res := database.DB.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&model.WxAccount{})
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, "公众号不存在或不属于当前租户视角 (超级管理员请先切换租户视角)")
		return
	}

	Ok(c, gin.H{"id": id, "deleted": true})
}

// SetDefault 设为默认公众号 (防越权)
func (h *AccountHandler) SetDefault(c *gin.Context) {
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var account model.WxAccount
	if err := database.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&account).Error; err != nil {
		Fail(c, http.StatusNotFound, "公众号不存在或不属于当前租户视角 (超级管理员请先切换租户视角)")
		return
	}

	database.DB.Model(&model.WxAccount{}).Where("tenant_id = ?", tenantID).Update("is_default", false)
	database.DB.Model(&model.WxAccount{}).Where("id = ? AND tenant_id = ?", id, tenantID).Update("is_default", true)
	Ok(c, gin.H{"status": "default_updated"})
}

// GetAuthorizerDetail 获取微信官方权威授权账号全量详情 (api_get_authorizer_info)
func (h *AccountHandler) GetAuthorizerDetail(c *gin.Context) {
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var account model.WxAccount
	if err := database.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&account).Error; err != nil {
		Fail(c, http.StatusNotFound, "公众号不存在或不属于当前租户视角 (超级管理员请先切换租户视角)")
		return
	}

	// 授权时落库的权限集，作为兜底展示
	storedFuncInfo := parseStoredFuncInfo(account.FuncInfo)

	// 1. 开放平台第三方代管账号: 实时调用微信官方接口取权威档案
	if account.IsThirdPartyManaged() {
		detail, err := service.DefaultWxOpenService.GetAuthorizerInfo(c.Request.Context(), account.ComponentAppID, account.AppID)
		if err != nil {
			// 官方接口失败时降级返回本地已落库档案，避免前端整页空白
			Ok(c, gin.H{
				"account":         account,
				"authorizer_info": localAuthorizerInfo(account),
				"func_info":       storedFuncInfo,
				"source":          "local",
				"official_error":  err.Error(),
				"hint":            "微信官方档案接口调用失败，以下展示的是本系统已落库的档案与授权时的权限集快照。常见原因: 授权已过期或被取消、第三方平台 component_access_token 异常、网络不通。",
			})
			return
		}

		// 自动增量补充落库 (更新微信官方返回的主体全称、原始ID、头像等)
		info := detail.AuthorizerInfo
		updates := map[string]interface{}{}
		if info.NickName != "" && account.Name != info.NickName {
			updates["name"] = info.NickName
		}
		if info.UserName != "" && account.OriginalID != info.UserName {
			updates["original_id"] = info.UserName
		}
		if info.HeadImg != "" && account.AvatarURL != info.HeadImg {
			updates["avatar_url"] = info.HeadImg
		}
		if info.QRCodeURL != "" && account.QRCodeURL != info.QRCodeURL {
			updates["qrcode_url"] = info.QRCodeURL
		}
		if info.Signature != "" && account.Description == "" {
			updates["description"] = info.Signature
		}
		// 微信官方档案显示为小程序授权 (存在 MiniProgramInfo) 时，纠正账号类型，
		// 使其能被「代码发布管理」的小程序选择器正确检索到
		if detail.AuthorizerInfo.MiniProgramInfo != nil && account.AccountType != "miniprogram" {
			updates["account_type"] = "miniprogram"
			account.AccountType = "miniprogram"
		}
		if len(detail.AuthorizationInfo.FuncInfo) > 0 {
			if b, e := json.Marshal(detail.AuthorizationInfo.FuncInfo); e == nil && string(b) != account.FuncInfo {
				updates["func_info"] = string(b)
			}
		}
		if len(updates) > 0 {
			database.DB.Model(&account).Updates(updates)
			if v, ok := updates["name"]; ok {
				account.Name = v.(string)
			}
			if v, ok := updates["original_id"]; ok {
				account.OriginalID = v.(string)
			}
			if v, ok := updates["avatar_url"]; ok {
				account.AvatarURL = v.(string)
			}
			if v, ok := updates["qrcode_url"]; ok {
				account.QRCodeURL = v.(string)
			}
			if v, ok := updates["description"]; ok {
				account.Description = v.(string)
			}
			if v, ok := updates["func_info"]; ok {
				account.FuncInfo = v.(string)
			}
		}

		liveFuncInfo := make([]service.AuthorizerFuncInfo, len(detail.AuthorizationInfo.FuncInfo))
		copy(liveFuncInfo, detail.AuthorizationInfo.FuncInfo)

		Ok(c, gin.H{
			"account":         account,
			"authorizer_info": detail.AuthorizerInfo,
			"func_info":       liveFuncInfo,
			"source":          "wechat_official",
			"raw":             detail.Raw,
		})
		return
	}

	// 2. 普通直连公众号: 微信未向直连账号开放权威档案接口，返回本系统落库信息
	Ok(c, gin.H{
		"account":         account,
		"authorizer_info": localAuthorizerInfo(account),
		"func_info":       storedFuncInfo,
		"source":          "local",
		"hint":            "该账号是「直连凭据」接入，微信开放平台的 api_get_authorizer_info 只对第三方平台授权托管的账号返回权威档案。若需微信官方主体全称、认证类型、权限集与账号二维码，请使用「微信扫码一键授权接入」重新托管该公众号。",
	})
}

// localAuthorizerInfo 用本系统落库字段拼出可展示的档案信息 (直连账号或官方接口失败时的兜底)
func localAuthorizerInfo(account model.WxAccount) gin.H {
	return gin.H{
		"nick_name":  account.Name,
		"head_img":   account.AvatarURL,
		"user_name":  account.OriginalID,
		"signature":  account.Description,
		"qrcode_url": account.QRCodeURL,
		"alias":      "",
	}
}

// parseStoredFuncInfo 解析授权时落库的 func_info JSON，失败时返回空列表
func parseStoredFuncInfo(raw string) []service.AuthorizerFuncInfo {
	if strings.TrimSpace(raw) == "" {
		return []service.AuthorizerFuncInfo{}
	}
	var list []service.AuthorizerFuncInfo
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []service.AuthorizerFuncInfo{}
	}
	return list
}
