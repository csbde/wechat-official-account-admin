package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Asynq     AsynqConfig     `mapstructure:"asynq"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	OIDC      OIDCConfig      `mapstructure:"oidc"`
	Wechat    WechatConfig    `mapstructure:"wechat"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type ServerConfig struct {
	Host           string   `mapstructure:"host"`           // 监听主机/IP: 0.0.0.0, 127.0.0.1, 或指定具体网卡IP
	Port           int      `mapstructure:"port"`           // 监听端口: 9305
	ListenAddr     string   `mapstructure:"listen_addr"`    // 完整的监听地址，若设置则优先于 host+port，如 127.0.0.1:9305
	Mode           string   `mapstructure:"mode"`           // debug, release
	AllowOrigins   []string `mapstructure:"allow_origins"`
	StaticDir      string   `mapstructure:"static_dir"`
	TrustedProxies []string `mapstructure:"trusted_proxies"` // 信任的上级反向代理 IP 或网段 (Nginx / Cloudflare 等)
}

func (s *ServerConfig) GetListenAddr() string {
	if s.ListenAddr != "" {
		return s.ListenAddr
	}
	host := strings.TrimSpace(s.Host)
	if host == "localhost" {
		host = "127.0.0.1"
	}
	if host == "" {
		host = "0.0.0.0"
	}
	return net.JoinHostPort(host, strconv.Itoa(s.Port))
}

type DatabaseConfig struct {
	Type            string `mapstructure:"type"` // sqlite (默认), postgres
	SqlitePath      string `mapstructure:"sqlite_path"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AsynqConfig struct {
	Concurrency int `mapstructure:"concurrency"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
	Issuer      string `mapstructure:"issuer"`
}

type OIDCConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	Issuer       string   `mapstructure:"issuer"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURI  string   `mapstructure:"redirect_uri"`
	Scopes       []string `mapstructure:"scopes"`
}

type WechatConfig struct {
	PublicHost string `mapstructure:"public_host"`
	Proxy      string `mapstructure:"proxy"`
}

type TelemetryConfig struct {
	ServiceName string  `mapstructure:"service_name"`
	Environment string  `mapstructure:"environment"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

var GlobalConfig Config

func Load(configPath string) (*Config, error) {
	// 1. 自动读取并加载 .env 文件
	loadDotEnv()

	v := viper.New()
	v.SetConfigType("yaml")
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.SetConfigName("config")
	}

	v.SetEnvPrefix("WOAA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 默认配置
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 9305)
	v.SetDefault("server.listen_addr", "")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.allow_origins", []string{"*"})
	v.SetDefault("server.static_dir", "./files")
	// 默认信任常见局域网与回环，支持 ["0.0.0.0/0"] 信任全部反代
	v.SetDefault("server.trusted_proxies", []string{
		"127.0.0.1",
		"::1",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"0.0.0.0/0",
	})

	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.sqlite_path", "./data/woaa.db")
	v.SetDefault("database.host", "")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "")
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 50)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", 3600)

	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("asynq.concurrency", 5)

	v.SetDefault("jwt.secret", "woaa-super-secure-jwt-token-secret-2025")
	v.SetDefault("jwt.expire_hours", 72)
	v.SetDefault("jwt.issuer", "woaa-admin")

	v.SetDefault("oidc.enabled", false)
	v.SetDefault("oidc.issuer", "")
	v.SetDefault("oidc.client_id", "")
	v.SetDefault("oidc.client_secret", "")
	v.SetDefault("oidc.redirect_uri", "")
	v.SetDefault("oidc.scopes", []string{"openid", "profile", "email"})

	v.SetDefault("telemetry.service_name", "wechat-official-account-admin")
	v.SetDefault("telemetry.environment", "development")
	v.SetDefault("telemetry.sample_rate", 1.0)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	if err := v.Unmarshal(&GlobalConfig); err != nil {
		return nil, err
	}

	// 兼容处理老版本或 .env 中的传统环境变量
	if listenAddr := os.Getenv("LISTEN_ADDR"); listenAddr != "" {
		GlobalConfig.Server.ListenAddr = listenAddr
		parts := strings.Split(listenAddr, ":")
		if len(parts) == 2 {
			GlobalConfig.Server.Host = parts[0]
			if p, err := strconv.Atoi(parts[1]); err == nil && p > 0 {
				GlobalConfig.Server.Port = p
			}
		}
	}
	if host := os.Getenv("HOST"); host != "" {
		GlobalConfig.Server.Host = host
	}
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			GlobalConfig.Server.Port = p
		}
	}
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		GlobalConfig.Redis.Addr = redisAddr
	}
	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		GlobalConfig.Redis.Password = redisPass
	}
	if pubHost := os.Getenv("PUBLIC_HOST"); pubHost != "" {
		GlobalConfig.Wechat.PublicHost = pubHost
	}

	return &GlobalConfig, nil
}

func loadDotEnv() {
	envFiles := []string{
		".env",
		".env.local",
		filepath.Join(".", ".env"),
	}

	for _, file := range envFiles {
		if fi, err := os.Stat(file); err == nil && !fi.IsDir() {
			_ = godotenv.Load(file)
			fmt.Printf("[CONFIG] 自动检测并加载环境变量文件: %s\n", file)
			break
		}
	}
}
