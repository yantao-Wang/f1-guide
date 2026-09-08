// Package handler 包含 HTTP 请求处理器。
//
// handler 只做三件事：解析请求、调用 service、写响应。
// 业务规则一律放 service 层，禁止反向依赖。
package handler

import (
	"log/slog"
	"net/http"
)

// Health 提供存活探针。
type Health struct {
	log *slog.Logger
}

// NewHealth 创建健康检查处理器。
func NewHealth(log *slog.Logger) *Health {
	return &Health{log: log}
}

// Health 返回 200 OK。用于部署后自动验证与 uptime 监控。
func (h *Health) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		h.log.Error("write health response", "error", err)
	}
}
