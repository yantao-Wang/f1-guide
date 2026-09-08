// Package config 负责加载运行时配置。
//
// 全部配置来自环境变量（12-factor），敏感信息不进入代码仓库。
// 未设置的环境变量使用本地开发默认值。
package config

import (
	"github.com/spf13/viper"
)

// Config 汇总 API 服务的运行时配置。
type Config struct {
	// HTTPAddr 监听地址，默认 :8080。
	HTTPAddr string
	// LogLevel 日志级别：debug / info / warn / error。
	LogLevel string
	// DatabaseURL PostgreSQL 连接串。
	DatabaseURL string
}

// Load 读取环境变量（前缀 F1GUIDE_），缺失时使用本地开发默认值。
func Load() Config {
	v := viper.New()
	v.SetEnvPrefix("F1GUIDE")
	v.AutomaticEnv()

	v.SetDefault("http_addr", ":8080")
	v.SetDefault("log_level", "info")
	v.SetDefault("database_url", "postgres://f1guide:f1guide@localhost:5432/f1guide?sslmode=disable")

	cfg := Config{
		HTTPAddr:    v.GetString("http_addr"),
		LogLevel:    v.GetString("log_level"),
		DatabaseURL: v.GetString("database_url"),
	}
	if cfg.HTTPAddr == "" {
		panic("invalid config: http_addr is empty")
	}
	return cfg
}
