// Package logger 提供全项目共享的结构化日志（slog）。
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New 返回 JSON 格式的结构化日志，级别由 level 指定。
// level：debug / info / warn / error，非法值回退到 info。
func New(level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
}
