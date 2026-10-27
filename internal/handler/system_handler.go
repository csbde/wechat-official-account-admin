package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/config"
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

// CreateUser 创建系统用户 (支持邮箱配置用于 OIDC 自动匹配)
func (h *SystemHandler) CreateUser(c *gin.Context) {
	tenantID := GetTenantID(c)
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email"`
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

	if req.Email != "" {
		var emailCount int64
		database.DB.Model(&model.User{}).Where("email = ?", req.Email).Count(&emailCount)
		if emailCount > 0 {
			Fail(c, http.StatusBadRequest, "该邮箱已被其他账号使用")
			return
		}
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	role := req.Role
	if role == "" || role == "superadmin" {
		role = "operator"
	}

	user := model.User{
		TenantID: tenantID,
		Username: req.Username,
		Email:    req.Email,
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
		Email    string `json:"email"`
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
	if req.Email != "" {
		var emailCount int64
		database.DB.Model(&model.User{}).Where("email = ? AND id != ?", req.Email, user.ID).Count(&emailCount)
		if emailCount > 0 {
			Fail(c, http.StatusBadRequest, "该邮箱已被其他账号占用")
			return
		}
		user.Email = req.Email
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

type ComponentHealth struct {
	Name        string `json:"name"`
	Configured  bool   `json:"configured"`
	Connected   bool   `json:"connected"`
	Status      string `json:"status"` // connected, disconnected, not_configured
	Target      string `json:"target"`
	LatencyMs   int64  `json:"latency_ms"`
	Details     string `json:"details,omitempty"`
}

// HealthCheck 全面连通性与健康探测 (检测真实连接情况与延迟)
func (h *SystemHandler) HealthCheck(c *gin.Context) {
	cfg := &config.GlobalConfig

	// 1. 探测数据库真实连通性
	dbHealth := ComponentHealth{
		Name:       "数据库 (Database)",
		Configured: true,
	}
	if database.DB != nil {
		sqlDB, err := database.DB.DB()
		if err != nil {
			dbHealth.Connected = false
			dbHealth.Status = "disconnected"
			dbHealth.Details = err.Error()
		} else {
			start := time.Now()
			pingErr := sqlDB.Ping()
			dbHealth.LatencyMs = time.Since(start).Milliseconds()
			if pingErr != nil {
				dbHealth.Connected = false
				dbHealth.Status = "disconnected"
				dbHealth.Details = pingErr.Error()
			} else {
				dbHealth.Connected = true
				dbHealth.Status = "connected"
				dialName := database.DB.Dialector.Name()
				if dialName == "sqlite" {
					dbHealth.Target = "SQLite (" + cfg.Database.SqlitePath + ")"
					fi, _ := os.Stat(cfg.Database.SqlitePath)
					sizeKB := int64(0)
					if fi != nil {
						sizeKB = fi.Size() / 1024
					}
					dbHealth.Details = fmt.Sprintf("本地文件 WAL 模式，库体积 %d KB", sizeKB)
				} else {
					dbHealth.Target = fmt.Sprintf("PostgreSQL (%s:%d/%s)", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
					stats := sqlDB.Stats()
					dbHealth.Details = fmt.Sprintf("连接池开销: %d 连接中 (空闲 %d)", stats.InUse, stats.Idle)
				}
			}
		}
	} else {
		dbHealth.Connected = false
		dbHealth.Status = "disconnected"
		dbHealth.Details = "数据库未初始化"
	}

	// 2. 探测 Redis 真实连通性
	redisHealth := ComponentHealth{
		Name: "Redis 缓存 (Cache)",
	}
	if cfg.Redis.Addr == "" {
		redisHealth.Configured = false
		redisHealth.Connected = false
		redisHealth.Status = "not_configured"
		redisHealth.Target = "未配置"
		redisHealth.Details = "运行于进程内内存两级缓存模式"
	} else {
		redisHealth.Configured = true
		redisHealth.Target = cfg.Redis.Addr
		if cache.RDB != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 1500*time.Millisecond)
			defer cancel()
			start := time.Now()
			pong, err := cache.RDB.Ping(ctx).Result()
			redisHealth.LatencyMs = time.Since(start).Milliseconds()
			if err != nil {
				redisHealth.Connected = false
				redisHealth.Status = "disconnected"
				redisHealth.Details = "连接失败: " + err.Error()
			} else {
				redisHealth.Connected = true
				redisHealth.Status = "connected"
				redisHealth.Details = fmt.Sprintf("网络响应正常 (PING -> %s)", pong)
			}
		} else {
			redisHealth.Connected = false
			redisHealth.Status = "disconnected"
			redisHealth.Details = "Redis 客户端尚未初始化"
		}
	}

	// 3. 探测 Asynq 队列连通性
	asynqHealth := ComponentHealth{
		Name: "Asynq 任务队列",
	}
	if !redisHealth.Connected {
		asynqHealth.Configured = redisHealth.Configured
		asynqHealth.Connected = false
		asynqHealth.Status = "not_configured"
		asynqHealth.Target = "依赖 Redis"
		asynqHealth.Details = "Redis 未连通，队列降级"
	} else {
		asynqHealth.Configured = true
		asynqHealth.Connected = queue.Client != nil
		asynqHealth.Status = "connected"
		asynqHealth.Target = fmt.Sprintf("并发 Worker: %d", cfg.Asynq.Concurrency)
		asynqHealth.Details = "后台延迟与重试队列就绪"
	}

	// 4. 探测微信官方服务器出网连通性 (api.weixin.qq.com)
	wxHealth := ComponentHealth{
		Name:       "微信官方 API 出网连通",
		Configured: true,
		Target:     "https://api.weixin.qq.com",
	}
	wxStart := time.Now()
	testClient := &http.Client{Timeout: 3 * time.Second}
	wxResp, wxErr := testClient.Get("https://api.weixin.qq.com/cgi-bin/token")
	wxHealth.LatencyMs = time.Since(wxStart).Milliseconds()
	if wxErr != nil {
		wxHealth.Connected = false
		wxHealth.Status = "disconnected"
		wxHealth.Details = "无法连接微信官方服务器，请检查服务器 DNS 或公网防火墙: " + wxErr.Error()
	} else {
		_ = wxResp.Body.Close()
		wxHealth.Connected = true
		wxHealth.Status = "connected"
		wxHealth.Details = "出网握手正常，微信接口网络畅通"
	}

	// 综合评定总体健康状态
	overallStatus := "healthy"
	if !dbHealth.Connected || !wxHealth.Connected {
		overallStatus = "degraded"
	}

	Ok(c, gin.H{
		"overall_status": overallStatus,
		"server_time":    time.Now().Format(time.RFC3339),
		"components": []ComponentHealth{
			dbHealth,
			redisHealth,
			asynqHealth,
			wxHealth,
		},
	})
}
