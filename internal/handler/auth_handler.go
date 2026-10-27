package handler

import (
	"net/http"

	"github.com/anchel/wechat-official-account-admin/internal/auth"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct{}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token  string        `json:"token"`
	User   *model.User   `json:"user"`
	Tenant *model.Tenant `json:"tenant,omitempty"`
}

// Login 用户密码登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}

	var user model.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		Fail(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	if user.Status == 0 {
		Fail(c, http.StatusForbidden, "账号已被禁用")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		Fail(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 查询租户
	var tenant model.Tenant
	if user.TenantID > 0 {
		database.DB.First(&tenant, user.TenantID)
	}

	token, err := auth.GenerateToken(user.ID, user.TenantID, user.Username, user.Role, user.SSOProvider)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "生成令牌失败")
		return
	}

	Ok(c, LoginResponse{
		Token:  token,
		User:   &user,
		Tenant: &tenant,
	})
}

// UserInfo 获取当前登录用户信息及所属租户
func (h *AuthHandler) UserInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		Fail(c, http.StatusNotFound, "用户不存在")
		return
	}

	var tenant model.Tenant
	if user.TenantID > 0 {
		database.DB.First(&tenant, user.TenantID)
	}

	Ok(c, gin.H{
		"user":   user,
		"tenant": tenant,
	})
}

// Logout 退出登录
func (h *AuthHandler) Logout(c *gin.Context) {
	Ok(c, gin.H{"status": "logged_out"})
}

// SSOCallback 单点登录授权回调
func (h *AuthHandler) SSOCallback(c *gin.Context) {
	var req struct {
		Provider string `json:"provider"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "SSO 参数错误")
		return
	}

	var user model.User
	err := database.DB.Where("sso_provider = ? AND sso_uid = ?", req.Provider, req.Code).First(&user).Error
	if err != nil {
		user = model.User{
			TenantID:    1,
			Username:    "sso_" + req.Code,
			Nickname:    "SSO User",
			Role:        "operator",
			SSOProvider: req.Provider,
			SSOUID:      req.Code,
			Status:      1,
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte("sso@2025"), bcrypt.DefaultCost)
		user.Password = string(hash)
		database.DB.Create(&user)
	}

	var tenant model.Tenant
	database.DB.First(&tenant, user.TenantID)

	token, _ := auth.GenerateToken(user.ID, user.TenantID, user.Username, user.Role, user.SSOProvider)
	Ok(c, LoginResponse{Token: token, User: &user, Tenant: &tenant})
}
