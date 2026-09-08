package handler

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/view"
)

// Pages 是服务端渲染页面的处理器（Go html/template，决策 D4）。
// API 处理器与页面处理器职责分离：JSON 走 Content/Stats，HTML 走 Pages。
type Pages struct {
	svc   *service.Content
	stats *service.Stats
}

// NewPages 创建页面处理器。
func NewPages(svc *service.Content, stats *service.Stats) *Pages {
	return &Pages{svc: svc, stats: stats}
}

// --- 页面数据载荷 ---

type homeData struct {
	DriverCount int
	TrackCount  int
	MomentCount int
	Featured    []domain.DriverSummary
	Standings   []standingsRow
}

// standingsRow 积分榜展示行：Linkable 表示该车手在站内有故事页（可挂链接）。
type standingsRow struct {
	domain.DriverStandingRow
	Linkable bool
}

type scheduleData struct {
	Season    int
	Upcoming  []domain.Race
	Completed []domain.Race
	Cancelled []domain.Race
	Failed    bool
}

type dataPageData struct {
	Season               int
	DriverStandings      []standingsRow
	ConstructorStandings []domain.ConstructorStandingRow
	Failed               bool
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

// Home 首页：Hero + 四大入口 + 精选 + 积分榜速览 + 测验入口（报告 §5.4）。
// 积分榜数据不可用时降级为占位文案，不影响其余模块渲染。
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
			Standings:   h.topStandings(r, drivers, 5),
		},
	})
}

// topStandings 取车手积分榜前 n 行；上游不可用时返回 nil（模板渲染占位）。
func (h *Pages) topStandings(r *http.Request, drivers []domain.DriverSummary, n int) []standingsRow {
	standings, err := h.stats.DriverStandings(r.Context(), 0)
	if err != nil {
		slog.Warn("standings unavailable, render placeholder", "error", err)
		return nil
	}
	slugSet := make(map[string]bool, len(drivers))
	for _, d := range drivers {
		slugSet[d.Slug] = true
	}
	if len(standings.Items) < n {
		n = len(standings.Items)
	}
	rows := make([]standingsRow, 0, n)
	for _, row := range standings.Items[:n] {
		rows = append(rows, standingsRow{DriverStandingRow: row, Linkable: slugSet[row.DriverSlug]})
	}
	return rows
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

// Schedule 赛程页：已完赛（近→远）+ 已取消 + 未开赛（按轮次）。
// 上游不可用时仍返回 200，页面渲染降级提示。
func (h *Pages) Schedule(w http.ResponseWriter, r *http.Request) {
	sched, err := h.stats.Schedule(r.Context(), 0)
	if err != nil {
		slog.Warn("schedule page data unavailable", "error", err)
		renderPage(w, "schedule", view.Page{
			Title:  "赛程",
			Active: "schedule",
			Data:   scheduleData{Failed: true},
		})
		return
	}

	upcoming := make([]domain.Race, 0)
	completed := make([]domain.Race, 0)
	cancelled := make([]domain.Race, 0)
	for _, race := range sched.Races {
		switch race.Status {
		case domain.RaceStatusCompleted:
			completed = append(completed, race)
		case domain.RaceStatusCancelled:
			cancelled = append(cancelled, race)
		default:
			upcoming = append(upcoming, race)
		}
	}
	sort.Slice(completed, func(i, j int) bool { return completed[i].Round > completed[j].Round })

	renderPage(w, "schedule", view.Page{
		Title:  "赛程",
		Active: "schedule",
		Data: scheduleData{
			Season:    sched.Season,
			Upcoming:  upcoming,
			Completed: completed,
			Cancelled: cancelled,
		},
	})
}

// Data 赛事数据页：车手积分榜 + 车队积分榜。上游不可用时渲染降级提示。
func (h *Pages) Data(w http.ResponseWriter, r *http.Request) {
	ds, err := h.stats.DriverStandings(r.Context(), 0)
	if err != nil {
		slog.Warn("data page standings unavailable", "error", err)
		renderPage(w, "data", view.Page{
			Title:  "赛事数据",
			Active: "data",
			Data:   dataPageData{Failed: true},
		})
		return
	}
	cs, err := h.stats.ConstructorStandings(r.Context(), 0)
	if err != nil {
		slog.Warn("data page standings unavailable", "error", err)
		renderPage(w, "data", view.Page{
			Title:  "赛事数据",
			Active: "data",
			Data:   dataPageData{Failed: true},
		})
		return
	}

	rows := make([]standingsRow, 0, len(ds.Items))
	slugSet := map[string]bool{}
	if drivers, err := h.svc.ListDrivers(r.Context(), nil); err == nil {
		for _, d := range drivers {
			slugSet[d.Slug] = true
		}
	} else {
		slog.Warn("driver slug set unavailable, standings 链接降级", "error", err)
	}
	for _, row := range ds.Items {
		rows = append(rows, standingsRow{DriverStandingRow: row, Linkable: slugSet[row.DriverSlug]})
	}

	renderPage(w, "data", view.Page{
		Title:  "赛事数据",
		Active: "data",
		Data: dataPageData{
			Season:               ds.Season,
			DriverStandings:      rows,
			ConstructorStandings: cs.Items,
		},
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
