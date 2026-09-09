// Package router 注册全部 HTTP 路由与中间件。
package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yantao-Wang/f1-guide/internal/handler"
	mw "github.com/yantao-Wang/f1-guide/internal/middleware"
	"github.com/yantao-Wang/f1-guide/internal/view"
)

// New 构建应用路由器。所有新路由在此注册，handler 层不感知路由结构。
// 依赖由 main 组装后传入，路由只做注册。
// admin 为 nil（后台未启用）时不注册任何 /admin 路由，攻击面默认关闭。
func New(log *slog.Logger, health *handler.Health, content *handler.Content, stats *handler.Stats, pages *handler.Pages, admin *handler.Admin, uploadDir string) http.Handler {
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
		r.Get("/schedule", stats.Schedule)
		r.Get("/standings/drivers", stats.DriverStandings)
		r.Get("/standings/constructors", stats.ConstructorStandings)
	})

	// 页面路由（报告 §6.2 URL 结构）
	r.Get("/", pages.Home)
	r.Get("/stories/drivers", pages.Drivers)
	r.Get("/stories/drivers/{slug}", pages.Driver)
	r.Get("/stories/tracks", pages.Tracks)
	r.Get("/stories/tracks/{slug}", pages.Track)
	r.Get("/stories/moments", pages.Moments)
	r.Get("/stories/moments/{slug}", pages.Moment)
	r.Get("/schedule", pages.Schedule)
	r.Get("/data", pages.Data)
	r.Get("/game", pages.StubPage("方格旗预言", "每站预测前十名，积分竞猜 —— 2027 赛季揭幕前正式上线"))
	r.Get("/quiz", pages.StubPage("新手测验", "5 道趣味选择题，测测你是哪个 F1 车手 —— 敬请期待"))
	r.Get("/about", pages.About)

	// 静态资源（CSS 等，embed 分发）
	r.Handle("/static/*", view.Static())

	// 上传文件分发：公开页面的内容图片（车手照/赛道图），独立于后台开关；
	// 后台关闭时已发布的内容仍能正常显示图片
	if uploadDir != "" {
		r.Handle("/uploads/*", handler.Uploads(uploadDir))
	}

	// 管理后台：admin 未启用时不注册（404），攻击面默认关闭
	if admin != nil {
		registerAdminRoutes(r, admin)
	}

	return r
}

// registerAdminRoutes 注册后台路由：登录公开，其余需认证；写操作加 CSRF 与请求体上限。
func registerAdminRoutes(r chi.Router, admin *handler.Admin) {
	r.Get("/admin/login", admin.LoginPage)
	r.With(mw.RequireCSRF).Post("/admin/login", admin.Login)

	r.Group(func(r chi.Router) {
		r.Use(admin.RequireAuth)

		r.Get("/admin", admin.Dashboard)
		r.Get("/admin/drivers", admin.DriverList)
		r.Get("/admin/drivers/new", admin.DriverNew)
		r.Get("/admin/drivers/{slug}/edit", admin.DriverEdit)
		r.Get("/admin/tracks", admin.TrackList)
		r.Get("/admin/tracks/new", admin.TrackNew)
		r.Get("/admin/tracks/{slug}/edit", admin.TrackEdit)
		r.Get("/admin/moments", admin.MomentList)
		r.Get("/admin/moments/new", admin.MomentNew)
		r.Get("/admin/moments/{slug}/edit", admin.MomentEdit)

		// 写操作：CSRF + 请求体上限（含文件上传的表单 8MB，纯文本 1MB）
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireCSRF)
			r.Use(mw.LimitBody(8 << 20))
			r.Post("/admin/logout", admin.Logout)
			r.Post("/admin/drivers/new", admin.DriverCreate)
			r.Post("/admin/drivers/{slug}/edit", admin.DriverUpdate)
			r.Post("/admin/drivers/{slug}/delete", admin.DriverDelete)
			r.Post("/admin/tracks/new", admin.TrackCreate)
			r.Post("/admin/tracks/{slug}/edit", admin.TrackUpdate)
			r.Post("/admin/tracks/{slug}/delete", admin.TrackDelete)
			r.Post("/admin/moments/new", admin.MomentCreate)
			r.Post("/admin/moments/{slug}/edit", admin.MomentUpdate)
			r.Post("/admin/moments/{slug}/delete", admin.MomentDelete)
		})
	})
}
