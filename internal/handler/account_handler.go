package handler

import (
	"fmt"
	"net/http"

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

// List 获取当前租户下的所有公众号
func (h *AccountHandler) List(c *gin.Context) {
	tenantID := GetTenantID(c)
	var list []model.WxAccount

	query := database.DB.Model(&model.WxAccount{})
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Order("is_default desc, id desc").Find(&list).Error; err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	role, _ := c.Get("role")
	isAdmin := role == "admin" || role == "superadmin"

	type AccountVO struct {
		model.WxAccount
		CallbackURL string `json:"callback_url"`
	}

	result := make([]AccountVO, len(list))
	for i, acc := range list {
		if !isAdmin {
			acc.AppSecret = "******"
			acc.EncodingAESKey = "******"
		}
		cb := BuildPublicURL(c, fmt.Sprintf("/wxmp/%s/handler", acc.AppID))
		result[i] = AccountVO{
			WxAccount:   acc,
			CallbackURL: cb,
		}
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

	var account model.WxAccount
	query := database.DB.Where("id = ?", id)
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.First(&account).Error; err != nil {
		Fail(c, http.StatusNotFound, "公众号不存在或无权操作")
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

	query := database.DB.Where("id = ?", id)
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	res := query.Delete(&model.WxAccount{})
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, "公众号不存在或无权操作")
		return
	}

	Ok(c, gin.H{"id": id, "deleted": true})
}

// SetDefault 设为默认公众号 (防越权)
func (h *AccountHandler) SetDefault(c *gin.Context) {
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var account model.WxAccount
	query := database.DB.Where("id = ?", id)
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.First(&account).Error; err != nil {
		Fail(c, http.StatusNotFound, "公众号不存在或无权操作")
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
	query := database.DB.Where("id = ?", id)
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.First(&account).Error; err != nil {
		Fail(c, http.StatusNotFound, "公众号不存在或无权查看")
		return
	}

	// 1. 若是开放平台第三方代管授权账号
	if account.AccountType == "authorized" && account.ComponentAppID != "" {
		detail, err := service.DefaultWxOpenService.GetAuthorizerInfo(c.Request.Context(), account.ComponentAppID, account.AppID)
		if err != nil {
			Fail(c, http.StatusInternalServerError, "获取微信官方档案失败: "+err.Error())
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
		if info.QRCodeURL != "" {
			updates["qrcode_url"] = info.QRCodeURL
		}
		if info.Signature != "" && account.Description == "" {
			updates["description"] = info.Signature
		}
		if len(updates) > 0 {
			database.DB.Model(&account).Updates(updates)
		}

		Ok(c, gin.H{
			"account":         account,
			"authorizer_info": detail.AuthorizerInfo,
			"func_info":       detail.AuthorizationInfo.FuncInfo,
		})
		return
	}

	// 2. 普通直连公众号，返回已有完整信息
	Ok(c, gin.H{
		"account": account,
		"authorizer_info": gin.H{
			"nick_name": account.Name,
			"user_name": account.OriginalID,
			"signature": account.Description,
		},
	})
}
