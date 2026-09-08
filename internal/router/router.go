// Package router 注册全部 HTTP 路由与中间件。
package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yantao-Wang/f1-guide/internal/handler"
	"github.com/yantao-Wang/f1-guide/internal/view"
)

// New 构建应用路由器。所有新路由在此注册，handler 层不感知路由结构。
// 依赖由 main 组装后传入，路由只做注册。
func New(log *slog.Logger, health *handler.Health, content *handler.Content, pages *handler.Pages) http.Handler {
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

	// 页面路由（报告 §6.2 URL 结构）
	r.Get("/", pages.Home)
	r.Get("/stories/drivers", pages.Drivers)
	r.Get("/stories/drivers/{slug}", pages.Driver)
	r.Get("/stories/tracks", pages.Tracks)
	r.Get("/stories/tracks/{slug}", pages.Track)
	r.Get("/stories/moments", pages.Moments)
	r.Get("/stories/moments/{slug}", pages.Moment)
	r.Get("/schedule", pages.StubPage("赛程", "2026 赛季赛程与赛果 —— 数据接入中，预计 10 月下旬点亮"))
	r.Get("/data", pages.StubPage("赛事数据", "积分榜与赛果 —— 数据接入中"))
	r.Get("/game", pages.StubPage("格子棋", "每站预测前十名，积分竞猜 —— 2027 赛季揭幕前正式上线"))
	r.Get("/quiz", pages.StubPage("新手测验", "5 道趣味选择题，测测你是哪个 F1 车手 —— 敬请期待"))
	r.Get("/about", pages.About)

	// 静态资源（CSS 等，embed 分发）
	r.Handle("/static/*", view.Static())

	return r
}
