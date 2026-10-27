package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Asynq     AsynqConfig     `mapstructure:"asynq"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Wechat    WechatConfig    `mapstructure:"wechat"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type ServerConfig struct {
	Port         int      `mapstructure:"port"`
	Mode         string   `mapstructure:"mode"`
	AllowOrigins []string `mapstructure:"allow_origins"`
	StaticDir    string   `mapstructure:"static_dir"`
}

type DatabaseConfig struct {
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

	// 默认值
	v.SetDefault("server.port", 9305)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.allow_origins", []string{"*"})
	v.SetDefault("server.static_dir", "./files")

	v.SetDefault("database.host", "127.0.0.1")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "sc")
	v.SetDefault("database.password", "sc")
	v.SetDefault("database.dbname", "woaa")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 50)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", 3600)

	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.password", "sc")
	v.SetDefault("redis.db", 0)

	v.SetDefault("asynq.concurrency", 5)

	v.SetDefault("jwt.secret", "woaa-super-secure-jwt-token-secret-2025")
	v.SetDefault("jwt.expire_hours", 72)
	v.SetDefault("jwt.issuer", "woaa-admin")

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

	return &GlobalConfig, nil
}
