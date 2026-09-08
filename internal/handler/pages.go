package handler

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/view"
)

// Pages 是服务端渲染页面的处理器（Go html/template，决策 D4）。
// API 处理器与页面处理器职责分离：JSON 走 Content，HTML 走 Pages。
type Pages struct {
	svc *service.Content
}

// NewPages 创建页面处理器。
func NewPages(svc *service.Content) *Pages {
	return &Pages{svc: svc}
}

// --- 页面数据载荷 ---

type homeData struct {
	DriverCount int
	TrackCount  int
	MomentCount int
	Featured    []domain.DriverSummary
}

type driversData struct {
	Drivers []domain.DriverSummary
}

type tracksData struct {
	Tracks []domain.TrackSummary
}

type momentCard struct {
	domain.MomentSummary
	TypeLabel string
}

type momentsData struct {
	Moments []momentCard
}

type driverDetailData struct {
	Driver *domain.Driver
}

type trackDetailData struct {
	Track *domain.Track
}

type momentDetailData struct {
	Moment    *domain.Moment
	TypeLabel string
}

type stubData struct {
	Title string
	Hint  string
}

// momentTypeLabel 把类型枚举翻译成用户可读标签（报告 §3.4）。
func momentTypeLabel(t string) string {
	switch t {
	case domain.MomentTypeChampionship:
		return "冠军争夺"
	case domain.MomentTypeOvertake:
		return "经典超车"
	case domain.MomentTypeSafety:
		return "事故安全"
	case domain.MomentTypeRain:
		return "雨战传奇"
	default:
		return t
	}
}

// renderPage 先渲染到缓冲区，成功后再写入响应——模板出错时不产生半截页面。
func renderPage(w http.ResponseWriter, name string, page view.Page) {
	var buf bytes.Buffer
	if err := view.Render(&buf, name, page); err != nil {
		slog.Error("render page", "template", name, "error", err)
		http.Error(w, "页面渲染失败", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

// --- 页面处理器 ---

// Home 首页：Hero + 四大入口 + 精选 + 积分榜占位 + 测验入口（报告 §5.4）。
func (h *Pages) Home(w http.ResponseWriter, r *http.Request) {
	drivers, err := h.svc.ListDrivers(r.Context(), nil)
	if err != nil {
		renderPageError(w, err)
		return
	}
	tracks, err := h.svc.ListTracks(r.Context())
	if err != nil {
		renderPageError(w, err)
		return
	}
	moments, err := h.svc.ListMoments(r.Context())
	if err != nil {
		renderPageError(w, err)
		return
	}

	featured := make([]domain.DriverSummary, 0, 3)
	for _, d := range drivers {
		if d.Featured && len(featured) < 3 {
			featured = append(featured, d)
		}
	}

	renderPage(w, "home", view.Page{
		Title:  "首页",
		Active: "home",
		Data: homeData{
			DriverCount: len(drivers),
			TrackCount:  len(tracks),
			MomentCount: len(moments),
			Featured:    featured,
		},
	})
}

// Drivers 车手故事列表页。
func (h *Pages) Drivers(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListDrivers(r.Context(), nil)
	if err != nil {
		renderPageError(w, err)
		return
	}
	renderPage(w, "drivers", view.Page{
		Title:  "车手故事",
		Active: "stories",
		Data:   driversData{Drivers: items},
	})
}

// Driver 车手详情页。不存在时渲染 404 页面。
func (h *Pages) Driver(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.GetDriver(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}
	renderPage(w, "driver_detail", view.Page{
		Title:  d.Name,
		Active: "stories",
		Data:   driverDetailData{Driver: d},
	})
}

// Tracks 赛道图鉴列表页。
func (h *Pages) Tracks(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListTracks(r.Context())
	if err != nil {
		renderPageError(w, err)
		return
	}
	renderPage(w, "tracks", view.Page{
		Title:  "赛道图鉴",
		Active: "stories",
		Data:   tracksData{Tracks: items},
	})
}

// Track 赛道详情页。
func (h *Pages) Track(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.GetTrack(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}
	renderPage(w, "track_detail", view.Page{
		Title:  t.Name,
		Active: "stories",
		Data:   trackDetailData{Track: t},
	})
}

// Moments 名场面列表页。
func (h *Pages) Moments(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListMoments(r.Context())
	if err != nil {
		renderPageError(w, err)
		return
	}
	cards := make([]momentCard, 0, len(items))
	for _, m := range items {
		cards = append(cards, momentCard{MomentSummary: m, TypeLabel: momentTypeLabel(m.Type)})
	}
	renderPage(w, "moments", view.Page{
		Title:  "名场面",
		Active: "stories",
		Data:   momentsData{Moments: cards},
	})
}

// Moment 名场面详情页。
func (h *Pages) Moment(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.GetMoment(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, service.ErrNotFound) {
		renderNotFound(w)
		return
	}
	if err != nil {
		renderPageError(w, err)
		return
	}
	renderPage(w, "moment_detail", view.Page{
		Title:  m.Title,
		Active: "stories",
		Data:   momentDetailData{Moment: m, TypeLabel: momentTypeLabel(m.Type)},
	})
}

// About 关于页（定位 + 免责声明，版权 b 策略的风险告知）。
func (h *Pages) About(w http.ResponseWriter, _ *http.Request) {
	renderPage(w, "about", view.Page{
		Title:  "关于",
		Active: "about",
	})
}

// StubPage 生成"敬请期待"占位页处理器（赛程/赛事数据/格子棋/新手测验）。
func (h *Pages) StubPage(title, hint string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		renderPage(w, "stub", view.Page{
			Title: title,
			Data:  stubData{Title: title, Hint: hint},
		})
	}
}

// renderNotFound 渲染 404 页面。
func renderNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	renderPage(w, "error_404", view.Page{
		Title: "404",
	})
}

// renderPageError 数据层错误统一降级为 500 页面。
func renderPageError(w http.ResponseWriter, err error) {
	slog.Error("page data error", "error", err)
	http.Error(w, "服务器开小差了，请稍后再试", http.StatusInternalServerError)
}
