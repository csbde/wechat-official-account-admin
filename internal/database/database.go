package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/config"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	}

	usePostgres := strings.ToLower(cfg.Type) == "postgres" && cfg.Host != ""

	if usePostgres {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
			cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode,
		)
		db, err := gorm.Open(postgres.Open(dsn), gormCfg)
		if err == nil {
			sqlDB, err := db.DB()
			if err == nil {
				sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
				sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
				sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
				DB = db
				logger.Info("PostgreSQL database connected successfully", zap.String("host", cfg.Host), zap.String("dbname", cfg.DBName))
				return DB, nil
			}
		}
		logger.Warn("Failed to connect PostgreSQL, fallback to SQLite", zap.Error(err))
	}

	// 默认使用 SQLite (litsql / pure-go sqlite)
	sqlitePath := cfg.SqlitePath
	if sqlitePath == "" {
		sqlitePath = "./data/woaa.db"
	}

	dir := filepath.Dir(sqlitePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sqlite directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(sqlitePath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(20)
		sqlDB.SetMaxIdleConns(5)
	}

	DB = db
	logger.Info("SQLite database initialized successfully (default)", zap.String("path", sqlitePath))
	return DB, nil
}
