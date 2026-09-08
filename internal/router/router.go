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
// 依赖由 main 组装后传入，路由只做注册。
func New(log *slog.Logger, health *handler.Health, content *handler.Content) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", health.Health)

	// API v1（对应 api/openapi.yaml）
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/drivers", content.ListDrivers)
		r.Get("/drivers/{slug}", content.GetDriver)
		r.Get("/tracks", content.ListTracks)
		r.Get("/tracks/{slug}", content.GetTrack)
		r.Get("/moments", content.ListMoments)
		r.Get("/moments/{slug}", content.GetMoment)
	})

	return r
}
