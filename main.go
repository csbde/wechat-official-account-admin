package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anchel/wechat-official-account-admin/config"
	_ "github.com/anchel/wechat-official-account-admin/docs"
	"github.com/anchel/wechat-official-account-admin/internal/auth"
	"github.com/anchel/wechat-official-account-admin/internal/cache"
	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/anchel/wechat-official-account-admin/internal/queue"
	"github.com/anchel/wechat-official-account-admin/internal/router"
	"github.com/anchel/wechat-official-account-admin/internal/telemetry"
	"github.com/anchel/wechat-official-account-admin/internal/validator"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

//go:embed all:wechat-official-account-admin-fe/dist
var frontendFS embed.FS

var configFile string

var rootCmd = &cobra.Command{
	Use:   "woaa",
	Short: "WeChat Official Account Admin",
	Long:  "Modern WeChat Official Account Management Platform built with Go 1.26+, SQLite/PostgreSQL, Multi-tenancy Data Isolation and React",
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to config file")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func startServer() {
	// 1. 加载配置 (Viper)
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化 Zap 日志
	_, err = logger.Init(cfg.Server.Mode)
	if err != nil {
		fmt.Printf("Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Starting WeChat Official Account Admin Server (Go 1.26+)...")

	// 3. 初始化 OpenTelemetry
	shutdownTelemetry, err := telemetry.Init(&cfg.Telemetry)
	if err != nil {
		logger.Warn("Failed to init OpenTelemetry", zap.Error(err))
	} else {
		defer func() {
			_ = shutdownTelemetry(context.Background())
		}()
	}

	// 4. 初始化数据库 (默认 SQLite，若配置 pgsql 则用 pgsql)
	db, err := database.Init(&cfg.Database)
	if err != nil {
		logger.Error("Database connection failed", zap.Error(err))
		os.Exit(1)
	}

	// 自动迁移多租户数据结构
	if err := model.AutoMigrate(db); err != nil {
		logger.Error("Database auto migrate failed", zap.Error(err))
		os.Exit(1)
	}

	// 初始化默认主租户与管理员
	initDefaultTenantAndAdmin(db)

	// 5. 初始化 Redis 缓存 (可选)
	if cfg.Redis.Addr != "" {
		_, err = cache.Init(&cfg.Redis)
		if err != nil {
			logger.Warn("Redis connection failed (caching degraded)", zap.Error(err))
		}
	}

	// 6. 初始化 Asynq 队列 (依赖 Redis，若未开启则优雅跳过)
	if cache.RDB != nil {
		_, server, err := queue.Init(cfg)
		if err == nil && server != nil {
			queue.StartWorker(server, queue.NewServeMux())
		}
	} else {
		logger.Info("Asynq queue disabled (Redis not configured or connected)")
	}

	// 7. 初始化 Casbin RBAC
	_, err = auth.InitCasbin(db)
	if err != nil {
		logger.Error("Failed to init Casbin", zap.Error(err))
		os.Exit(1)
	}

	// 8. 初始化参数校验器
	validator.Init()

	// 9. 构建路由与中间件
	r := router.SetupRouter(&frontendFS)

	listenAddr := cfg.Server.GetListenAddr()
	srv := &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}

	go func() {
		logger.Info(fmt.Sprintf("Server is listening on %s (Database: %s)", listenAddr, db.Dialector.Name()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server start failed", zap.Error(err))
			os.Exit(1)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited properly")
}

func initDefaultTenantAndAdmin(db *gorm.DB) {
	// 初始化默认租户
	var tenant model.Tenant
	if err := db.Where("id = 1").First(&tenant).Error; err != nil {
		tenant = model.Tenant{
			ID:          1,
			Name:        "默认组织",
			Code:        "default",
			Description: "系统默认组织企业",
			Status:      1,
		}
		db.Create(&tenant)
		logger.Info("Default tenant created: ID=1 (default)")
	}

	// 初始化默认管理员用户 (admin / admin123, 属于租户1)
	var count int64
	db.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := model.User{
			TenantID: 1, // 属于默认租户
			Username: "admin",
			Email:    "admin@kkio.com",
			Password: string(hash),
			Nickname: "超级管理员",
			Role:     "superadmin",
			Status:   1,
		}
		db.Create(&admin)
		logger.Info("Default superadmin user created: admin / admin123 (Tenant: 1)")
	}
}
