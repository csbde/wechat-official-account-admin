package handler

import (
	"fmt"
	"net/http"

	"github.com/anchel/wechat-official-account-admin/config"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
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
// @Summary 获取所有配置的微信公众号
// @Tags 公众号管理
// @Security BearerAuth
// @Produce json
// @Router /api/accounts [get]
func (h *AccountHandler) List(c *gin.Context) {
	var list []model.WxAccount
	if err := database.DB.Order("is_default desc, id desc").Find(&list).Error; err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 非 admin 角色遮蔽 appsecret 等密钥
	role, _ := c.Get("role")
	isAdmin := role == "admin"

	type AccountVO struct {
		model.WxAccount
		CallbackURL string `json:"callback_url"`
	}

	host := config.GlobalConfig.Wechat.PublicHost
	if host == "" {
		host = c.Request.Host
	}

	result := make([]AccountVO, len(list))
	for i, acc := range list {
		if !isAdmin {
			acc.AppSecret = "******"
			acc.EncodingAESKey = "******"
		}
		cb := fmt.Sprintf("http://%s/wxmp/%s/handler", host, acc.AppID)
		result[i] = AccountVO{
			WxAccount:   acc,
			CallbackURL: cb,
		}
	}

	Ok(c, result)
}

// Create 新建微信公众号
// @Summary 添加新公众号
// @Tags 公众号管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/accounts [post]
func (h *AccountHandler) Create(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var count int64
	database.DB.Model(&model.WxAccount{}).Where("app_id = ?", req.AppID).Count(&count)
	if count > 0 {
		Fail(c, http.StatusBadRequest, "该 AppID 已存在")
		return
	}

	var total int64
	database.DB.Model(&model.WxAccount{}).Count(&total)

	account := model.WxAccount{
		Name:           req.Name,
		AppID:          req.AppID,
		AppSecret:      req.AppSecret,
		Token:          req.Token,
		EncodingAESKey: req.EncodingAESKey,
		OriginalID:     req.OriginalID,
		Description:    req.Description,
		Enabled:        true,
		IsDefault:      total == 0, // 第一个自动设为默认
	}

	if err := database.DB.Create(&account).Error; err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	Ok(c, account)
}

// Update 更新微信公众号
// @Summary 更新公众号配置
// @Tags 公众号管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/accounts/{id} [put]
func (h *AccountHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var account model.WxAccount
	if err := database.DB.First(&account, id).Error; err != nil {
		Fail(c, http.StatusNotFound, "公众号不存在")
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

// Delete 删除公众号
// @Summary 删除指定公众号
// @Tags 公众号管理
// @Security BearerAuth
// @Produce json
// @Router /api/accounts/{id} [delete]
func (h *AccountHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&model.WxAccount{}, id).Error; err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	Ok(c, gin.H{"id": id, "deleted": true})
}

// SetDefault 设为默认公众号
// @Summary 设为默认公众号
// @Tags 公众号管理
// @Security BearerAuth
// @Produce json
// @Router /api/accounts/{id}/set-default [post]
func (h *AccountHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")
	database.DB.Model(&model.WxAccount{}).Where("1 = 1").Update("is_default", false)
	database.DB.Model(&model.WxAccount{}).Where("id = ?", id).Update("is_default", true)
	Ok(c, gin.H{"status": "default_updated"})
}
