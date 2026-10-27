package handler

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/cache"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/queue"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type SystemHandler struct{}

// UserList 用户列表 (租户隔离防越权)
func (h *SystemHandler) UserList(c *gin.Context) {
	tenantID := GetTenantID(c)
	var users []model.User

	query := database.DB.Model(&model.User{})
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	query.Order("id asc").Find(&users)
	Ok(c, users)
}

// CreateUser 创建系统用户 (强制绑定当前租户)
func (h *SystemHandler) CreateUser(c *gin.Context) {
	tenantID := GetTenantID(c)
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var count int64
	database.DB.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		Fail(c, http.StatusBadRequest, "用户名已存在")
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	role := req.Role
	if role == "" || role == "superadmin" {
		role = "operator"
	}

	user := model.User{
		TenantID: tenantID, // 严格隔离归属当前租户
		Username: req.Username,
		Password: string(hash),
		Nickname: req.Nickname,
		Role:     role,
		Status:   1,
	}
	database.DB.Create(&user)
	Ok(c, user)
}

// UpdateUser 更新用户 (防越权校验)
func (h *SystemHandler) UpdateUser(c *gin.Context) {
	tenantID := GetTenantID(c)
	id := c.Param("id")

	var user model.User
	query := database.DB.Where("id = ?", id)
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.First(&user).Error; err != nil {
		Fail(c, http.StatusNotFound, "用户不存在或无权操作")
		return
	}

	var req struct {
		Nickname string `json:"nickname"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Status   *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Role != "" && req.Role != "superadmin" {
		user.Role = req.Role
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.Password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user.Password = string(hash)
	}

	database.DB.Save(&user)
	Ok(c, user)
}

// DeleteUser 删除用户 (防越权校验)
func (h *SystemHandler) DeleteUser(c *gin.Context) {
	tenantID := GetTenantID(c)
	id := c.Param("id")

	if id == "1" {
		Fail(c, http.StatusBadRequest, "不能删除初始超级管理员")
		return
	}

	query := database.DB.Where("id = ?", id)
	if !IsSuperAdmin(c) {
		query = query.Where("tenant_id = ?", tenantID)
	}

	res := query.Delete(&model.User{})
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, "用户不存在或无权操作")
		return
	}
	Ok(c, gin.H{"id": id, "deleted": true})
}

// PublicIP 获取服务器公网出口 IP
func (h *SystemHandler) PublicIP(c *gin.Context) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		Ok(c, gin.H{"ip": "127.0.0.1"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	Ok(c, gin.H{"ip": strings.TrimSpace(string(body))})
}

// HealthCheck 健康探测
func (h *SystemHandler) HealthCheck(c *gin.Context) {
	dbType := "sqlite"
	if database.DB != nil {
		dbType = database.DB.Dialector.Name()
	}

	dbStatus := "up"
	if database.DB != nil {
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "down"
		}
	} else {
		dbStatus = "down"
	}

	redisStatus := "up"
	if cache.RDB != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
		defer cancel()
		if err := cache.RDB.Ping(ctx).Err(); err != nil {
			redisStatus = "down"
		}
	} else {
		redisStatus = "down"
	}

	asynqStatus := "up"
	if queue.Client == nil {
		asynqStatus = "down"
	}

	Ok(c, gin.H{
		"status":        "ok",
		"database":      dbStatus,
		"database_type": dbType,
		"redis":         redisStatus,
		"asynq":         asynqStatus,
		"telemetry":     "active",
		"server_time":   time.Now().Format(time.RFC3339),
	})
}
