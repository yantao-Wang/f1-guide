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
	// AdminPassword 后台登录密码。为空时管理后台整体禁用（不注册 /admin 路由）。
	AdminPassword string
	// AdminSecret 会话 Cookie 的 HMAC 签名密钥。为空时启动生成随机密钥（重启后需重新登录）。
	AdminSecret string
	// UploadDir 后台文件上传的本地存储目录，默认 ./uploads（Docker 部署需挂卷持久化）。
	UploadDir string
}

// Load 读取环境变量（前缀 F1GUIDE_），缺失时使用本地开发默认值。
func Load() Config {
	v := viper.New()
	v.SetEnvPrefix("F1GUIDE")
	v.AutomaticEnv()

	v.SetDefault("http_addr", ":8080")
	v.SetDefault("log_level", "info")
	v.SetDefault("database_url", "postgres://f1guide:f1guide@localhost:5432/f1guide?sslmode=disable")
	v.SetDefault("admin_password", "")
	v.SetDefault("admin_secret", "")
	v.SetDefault("upload_dir", "./uploads")

	return Config{
		HTTPAddr:      v.GetString("http_addr"),
		LogLevel:      v.GetString("log_level"),
		DatabaseURL:   v.GetString("database_url"),
		AdminPassword: v.GetString("admin_password"),
		AdminSecret:   v.GetString("admin_secret"),
		UploadDir:     v.GetString("upload_dir"),
	}
}
