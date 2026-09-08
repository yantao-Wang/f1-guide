// Package router 注册全部 HTTP 路由与中间件。
package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yantao-Wang/f1-guide/internal/handler"
)

// New 构建应用路由器。所有新路由在此注册，handler 层不感知路由结构。
func New(log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	health := handler.NewHealth(log)
	r.Get("/health", health.Health)

	return r
}
