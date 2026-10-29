package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	AppType        string `json:"app_type"`     // service, subscription, miniprogram
	VerifyType     *int   `json:"verify_type"`  // -1: 未认证, 0: 微信认证
}

type UpdateAccountRequest struct {
	Name           string `json:"name"`
	AppSecret      string `json:"appsecret"`
	Token          string `json:"token"`
	EncodingAESKey string `json:"encoding_aes_key"`
	OriginalID     string `json:"original_id"`
	Description    string `json:"description"`
	Enabled        *bool  `json:"enabled"`
	AppType        string `json:"app_type"`
	VerifyType     *int   `json:"verify_type"`
}

// List 获取公众号列表
// 默认只返回「当前租户视角」内的账号 (超级管理员可通过 X-Tenant-ID 切换视角)；
// 仅当超级管理员显式携带 ?all=1 时才跨租户总览，并在结果中标注归属租户，避免混淆。
func (h *AccountHandler) List(c *gin.Context) {
	tenantID := GetTenantID(c)
	var list []model.WxAccount

	query := database.DB.Model(&model.WxAccount{})
	// 超级管理员默认可见全平台账号并标注归属租户；显式切换租户视角(X-Tenant-ID)后严格按视角过滤。
	// 普通成员始终只能看到本租户数据；超管可用 ?all=0 只看当前视角。
	scopeAll := SuperAdminHasFullCrossTenantAccess(c) && c.Query("all") != "0"
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

	// 租户名称映射(含已软删租户)：跨租户总览时明确标注每条数据的归属
	tenantMap := make(map[uint]model.Tenant)
	{
		var tenants []model.Tenant
		database.DB.Unscoped().Model(&model.Tenant{}).Select("id", "name", "code", "deleted_at").Find(&tenants)
		for _, t := range tenants {
			tenantMap[t.ID] = t
		}
	}

	type AccountVO struct {
		model.WxAccount
		CallbackURL string `json:"callback_url"`
		TenantName  string `json:"tenant_name,omitempty"`
		TenantCode  string `json:"tenant_code,omitempty"`
		TenantGone  bool   `json:"tenant_missing"`
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
			Manageable:  acc.TenantID == tenantID || SuperAdminHasFullCrossTenantAccess(c),
		}
		if t, ok := tenantMap[acc.TenantID]; ok {
			vo.TenantName = t.Name
			vo.TenantCode = t.Code
			vo.TenantGone = t.DeletedAt.Valid
		} else if acc.TenantID != 0 {
			vo.TenantName = fmt.Sprintf("租户 #%d (记录缺失)", acc.TenantID)
			vo.TenantGone = true
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

	appType := req.AppType
	if appType == "" {
		appType = "service"
	}
	verifyType := -1
	if req.VerifyType != nil {
		verifyType = *req.VerifyType
	}
	serviceType := 2
	if appType == "subscription" {
		serviceType = 0
	} else if appType == "miniprogram" {
		serviceType = 3
	}

	account := model.WxAccount{
		TenantID:       tenantID, // 强制隔离归属
		Name:           req.Name,
		AppID:          req.AppID,
		AppSecret:      req.AppSecret,
		Token:          req.Token,
		EncodingAESKey: req.EncodingAESKey,
		OriginalID:     req.OriginalID,
		Description:    req.Description,
		AppType:        appType,
		ServiceType:    serviceType,
		VerifyType:     verifyType,
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
	id := c.Param("id")

	account, ok := FindAccountForManage(c, id)
	if !ok {
		Fail(c, http.StatusNotFound, AccountNotFoundMessage(c))
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
	if req.AppType != "" {
		account.AppType = req.AppType
		if req.AppType == "subscription" {
			account.ServiceType = 0
		} else if req.AppType == "service" {
			account.ServiceType = 2
		} else if req.AppType == "miniprogram" {
			account.ServiceType = 3
		}
	}
	if req.VerifyType != nil {
		account.VerifyType = *req.VerifyType
	}

	if err := database.DB.Save(account).Error; err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	Ok(c, account)
}

// Delete 删除公众号 (防越权校验)
func (h *AccountHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	account, ok := FindAccountForManage(c, id)
	if !ok {
		Fail(c, http.StatusNotFound, AccountNotFoundMessage(c))
		return
	}
	res := database.DB.Where("id = ? AND tenant_id = ?", account.ID, account.TenantID).Delete(&model.WxAccount{})
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, "删除公众号失败")
		return
	}

	Ok(c, gin.H{"id": id, "deleted": true})
}

// SetDefault 设为默认公众号 (防越权)
func (h *AccountHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")

	account, ok := FindAccountForManage(c, id)
	if !ok {
		Fail(c, http.StatusNotFound, AccountNotFoundMessage(c))
		return
	}

	// 默认标记只在账号所属租户内唯一，避免跨租户互相顶掉默认账号
	database.DB.Model(&model.WxAccount{}).Where("tenant_id = ?", account.TenantID).Update("is_default", false)
	database.DB.Model(&model.WxAccount{}).Where("id = ?", account.ID).Update("is_default", true)
	Ok(c, gin.H{"status": "default_updated"})
}

// GetAuthorizerDetail 获取微信官方权威授权账号全量详情 (api_get_authorizer_info)
func (h *AccountHandler) GetAuthorizerDetail(c *gin.Context) {
	id := c.Param("id")

	account, ok := FindAccountForManage(c, id)
	if !ok {
		Fail(c, http.StatusNotFound, AccountNotFoundMessage(c))
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
				"authorizer_info": localAuthorizerInfo(*account),
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
		// 同步微信权威档案中的服务号/订阅号/小程序类型与认证类型
		updates["service_type"] = info.ServiceTypeInfo.ID
		account.ServiceType = info.ServiceTypeInfo.ID
		updates["verify_type"] = info.VerifyTypeInfo.ID
		account.VerifyType = info.VerifyTypeInfo.ID
		typeName := info.ServiceTypeInfo.Name
		isMp := detail.AuthorizerInfo.MiniProgramInfo != nil ||
			info.ServiceTypeInfo.ID == 4 ||
			strings.Contains(typeName, "小程序") ||
			strings.Contains(typeName, "小游戏")
		if rawInfo, ok := detail.Raw["authorizer_info"].(map[string]interface{}); ok {
			if _, hasMp := rawInfo["MiniProgramInfo"]; hasMp {
				isMp = true
			}
			if _, hasMp2 := rawInfo["mini_program_info"]; hasMp2 {
				isMp = true
			}
		}

		if isMp {
			updates["app_type"] = "miniprogram"
			account.AppType = "miniprogram"
			if account.AccountType != "miniprogram" {
				updates["account_type"] = "miniprogram"
				account.AccountType = "miniprogram"
			}
		} else if info.ServiceTypeInfo.ID == 0 || info.ServiceTypeInfo.ID == 1 {
			updates["app_type"] = "subscription"
			account.AppType = "subscription"
		} else if info.ServiceTypeInfo.ID == 2 {
			updates["app_type"] = "service"
			account.AppType = "service"
		}
		if len(detail.AuthorizationInfo.FuncInfo) > 0 {
			if b, e := json.Marshal(detail.AuthorizationInfo.FuncInfo); e == nil && string(b) != account.FuncInfo {
				updates["func_info"] = string(b)
			}
		}
		if len(updates) > 0 {
			database.DB.Model(account).Updates(updates)
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
			if v, ok := updates["service_type"]; ok {
				account.ServiceType = v.(int)
			}
			if v, ok := updates["verify_type"]; ok {
				account.VerifyType = v.(int)
			}
			if v, ok := updates["app_type"]; ok {
				account.AppType = v.(string)
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
		"authorizer_info": localAuthorizerInfo(*account),
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

// TransferTenant 迁移公众号到另一个租户 (仅超级管理员)。
// 用于纠正"账号建错了租户/遗留在已删除租户"的历史数据：在同一个事务内，
// 把该 AppID 在源租户下的全部业务数据(菜单/素材/自动回复/粉丝/标签/二维码/AI/小程序设置/日志)一并搬家，
// 避免只挪走账号而业务数据留在原租户形成孤立脏数据。
func (h *AccountHandler) TransferTenant(c *gin.Context) {
	if !IsSuperAdmin(c) {
		Fail(c, http.StatusForbidden, "越权操作：仅超级管理员可以迁移公众号归属租户")
		return
	}

	id := c.Param("id")
	var req struct {
		TenantID *uint `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}
	if req.TenantID == nil || *req.TenantID == 0 {
		Fail(c, http.StatusBadRequest, "请指定目标租户 ID")
		return
	}

	account, ok := FindAccountForManage(c, id)
	if !ok {
		Fail(c, http.StatusNotFound, AccountNotFoundMessage(c))
		return
	}

	var target model.Tenant
	if err := database.DB.First(&target, *req.TenantID).Error; err != nil {
		Fail(c, http.StatusBadRequest, "目标租户不存在 (id="+strconv.FormatUint(uint64(*req.TenantID), 10)+")")
		return
	}
	if target.Status != 1 {
		Fail(c, http.StatusBadRequest, "目标租户「"+target.Name+"」已停用，请先启用后再迁移")
		return
	}
	if account.TenantID == target.ID {
		Fail(c, http.StatusBadRequest, "该公众号已经归属租户「"+target.Name+"」，无需迁移")
		return
	}

	// 目标租户内已有相同 AppID 时禁止迁移，避免破坏租户内 AppID 唯一性与凭据串号
	var dup int64
	database.DB.Model(&model.WxAccount{}).Where("app_id = ? AND tenant_id = ?", account.AppID, target.ID).Count(&dup)
	if dup > 0 {
		Fail(c, http.StatusBadRequest, "目标租户「"+target.Name+"」已存在相同 AppID 的公众号，请先删除重复记录后再迁移")
		return
	}

	from := account.TenantID
	appID := account.AppID

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		moves := []struct {
			name  string
			value interface{}
		}{
			{"菜单", &model.WxMenu{}},
			{"素材", &model.WxMaterial{}},
			{"自动回复", &model.WxAutoReply{}},
			{"粉丝", &model.WxFan{}},
			{"标签", &model.WxTag{}},
			{"带参二维码", &model.WxQrcode{}},
			{"AI 助手设置", &model.WxAISetting{}},
			{"小程序业务配置", &model.WxMiniSetting{}},
			{"小程序注册任务", &model.WxMiniRegisterTask{}},
			{"请求日志", &model.RequestLog{}},
		}
		for _, m := range moves {
			if err := tx.Model(m.value).
				Where("app_id = ? AND tenant_id = ?", appID, from).
				Update("tenant_id", target.ID).Error; err != nil {
				return fmt.Errorf("迁移%s失败: %w", m.name, err)
			}
		}
		// 账号本体 (同时修正其租户内默认标记)
		if err := tx.Model(&model.WxAccount{}).Where("id = ?", account.ID).Update("tenant_id", target.ID).Error; err != nil {
			return fmt.Errorf("迁移公众号失败: %w", err)
		}
		var targetTotal int64
		if err := tx.Model(&model.WxAccount{}).Where("tenant_id = ?", target.ID).Count(&targetTotal).Error; err != nil {
			return err
		}
		if targetTotal == 1 {
			if err := tx.Model(&model.WxAccount{}).Where("tenant_id = ?", target.ID).Update("is_default", true).Error; err != nil {
				return err
			}
		} else {
			var keepDefault int64
			if err := tx.Model(&model.WxAccount{}).Where("tenant_id = ? AND is_default = ?", target.ID, true).Count(&keepDefault).Error; err != nil {
				return err
			}
			if keepDefault == 0 {
				if err := tx.Model(&model.WxAccount{}).Where("id = ?", account.ID).Update("is_default", true).Error; err != nil {
					return err
				}
			}
		}
		// 源租户剩余账号中补选一个默认账号
		var srcRemaining []model.WxAccount
		if err := tx.Where("tenant_id = ?", from).Find(&srcRemaining).Error; err != nil {
			return err
		}
		srcHasDefault := false
		for _, a := range srcRemaining {
			if a.IsDefault {
				srcHasDefault = true
				break
			}
		}
		if len(srcRemaining) > 0 && !srcHasDefault {
			if err := tx.Model(&model.WxAccount{}).Where("id = ?", srcRemaining[0].ID).Update("is_default", true).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		Fail(c, http.StatusInternalServerError, "迁移失败已回滚: "+err.Error())
		return
	}

	logger.Info("Account transferred between tenants",
		zap.String("appid", appID),
		zap.Uint("from_tenant_id", from),
		zap.Uint("to_tenant_id", target.ID),
		zap.String("operator", c.GetString("username")),
	)
	Ok(c, gin.H{
		"message":     fmt.Sprintf("公众号 %s 及其全部业务数据已从租户 #%d 迁移至「%s」", appID, from, target.Name),
		"from_tenant": from,
		"to_tenant":   target.ID,
		"to_name":     target.Name,
	})
}
