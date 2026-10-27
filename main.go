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
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
)

//go:embed all:wechat-official-account-admin-fe/dist
var frontendFS embed.FS

var configFile string

var rootCmd = &cobra.Command{
	Use:   "woaa",
	Short: "WeChat Official Account Admin",
	Long:  "Modern WeChat Official Account Management Platform built with Go, Gin, GORM, PostgreSQL, Redis, Asynq, Casbin and OpenTelemetry",
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
	logger.Info("Starting WeChat Official Account Admin Server...")

	// 3. 初始化 OpenTelemetry
	shutdownTelemetry, err := telemetry.Init(&cfg.Telemetry)
	if err != nil {
		logger.Warn("Failed to init OpenTelemetry", zap.Error(err))
	} else {
		defer func() {
			_ = shutdownTelemetry(context.Background())
		}()
	}

	// 4. 初始化数据库 (GORM + PostgreSQL)
	db, err := database.Init(&cfg.Database)
	if err != nil {
		logger.Error("Database connection failed", zap.Error(err))
		os.Exit(1)
	}

	// 自动迁移
	if err := model.AutoMigrate(db); err != nil {
		logger.Error("Database auto migrate failed", zap.Error(err))
		os.Exit(1)
	}

	// 初始化默认管理员用户 (admin / admin123)
	initDefaultAdmin(db)

	// 5. 初始化 Redis 缓存
	_, err = cache.Init(&cfg.Redis)
	if err != nil {
		logger.Warn("Redis connection failed (caching will be degraded)", zap.Error(err))
	}

	// 6. 初始化 Asynq 队列
	_, server, err := queue.Init(cfg)
	if err == nil && server != nil {
		queue.StartWorker(server, queue.NewServeMux())
	} else {
		logger.Warn("Asynq queue disabled", zap.Error(err))
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

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		logger.Info(fmt.Sprintf("Server is listening on :%d", cfg.Server.Port))
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

func initDefaultAdmin(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := model.User{
			Username: "admin",
			Password: string(hash),
			Nickname: "超级管理员",
			Role:     "admin",
			Status:   1,
		}
		db.Create(&admin)
		logger.Info("Default administrator user created: admin / admin123")
	}
}
