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

// UserList 用户列表
// @Summary 获取后台用户列表
// @Tags 系统管理
// @Security BearerAuth
// @Produce json
// @Router /api/system/users [get]
func (h *SystemHandler) UserList(c *gin.Context) {
	var users []model.User
	database.DB.Order("id asc").Find(&users)
	Ok(c, users)
}

// CreateUser 创建系统用户
// @Summary 添加后台用户
// @Tags 系统管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/system/users [post]
func (h *SystemHandler) CreateUser(c *gin.Context) {
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
	if role == "" {
		role = "operator"
	}

	user := model.User{
		Username: req.Username,
		Password: string(hash),
		Nickname: req.Nickname,
		Role:     role,
		Status:   1,
	}
	database.DB.Create(&user)
	Ok(c, user)
}

// UpdateUser 更新用户
// @Summary 修改用户信息或重置密码
// @Tags 系统管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Router /api/system/users/{id} [put]
func (h *SystemHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		Fail(c, http.StatusNotFound, "用户不存在")
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
	if req.Role != "" {
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

// DeleteUser 删除用户
// @Summary 删除指定用户
// @Tags 系统管理
// @Security BearerAuth
// @Produce json
// @Router /api/system/users/{id} [delete]
func (h *SystemHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "1" {
		Fail(c, http.StatusBadRequest, "不能删除超级管理员账号")
		return
	}
	database.DB.Delete(&model.User{}, id)
	Ok(c, gin.H{"id": id, "deleted": true})
}

// PublicIP 获取服务器公网出口 IP
// @Summary 查询服务器公网 IP
// @Tags 系统管理
// @Produce json
// @Router /api/system/public-ip [get]
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
// @Summary 服务各组件健康状态
// @Tags 系统管理
// @Produce json
// @Router /api/system/health [get]
func (h *SystemHandler) HealthCheck(c *gin.Context) {
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
		"status":      "ok",
		"database":    dbStatus,
		"redis":       redisStatus,
		"asynq":       asynqStatus,
		"telemetry":   "active",
		"server_time": time.Now().Format(time.RFC3339),
	})
}
