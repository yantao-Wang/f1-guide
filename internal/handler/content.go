package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/service"
)

// Content 内容板块 HTTP 处理器。
type Content struct {
	svc *service.Content
}

// NewContent 创建内容处理器。
func NewContent(svc *service.Content) *Content {
	return &Content{svc: svc}
}

// --- 车手 ---

// ListDrivers GET /api/v1/drivers
func (h *Content) ListDrivers(w http.ResponseWriter, r *http.Request) {
	featured, err := parseFeatured(r.URL.Query().Get("featured"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", "featured 参数必须是 true 或 false")
		return
	}

	items, err := h.svc.ListDrivers(r.Context(), featured)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "服务器开小差了，请稍后再试")
		return
	}
	writeJSON(w, http.StatusOK, driverListResponse{Items: items})
}

// GetDriver GET /api/v1/drivers/{slug}
func (h *Content) GetDriver(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.GetDriver(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "该车手不存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "服务器开小差了，请稍后再试")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// --- 赛道 ---

// ListTracks GET /api/v1/tracks
func (h *Content) ListTracks(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListTracks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "服务器开小差了，请稍后再试")
		return
	}
	writeJSON(w, http.StatusOK, trackListResponse{Items: items})
}

// GetTrack GET /api/v1/tracks/{slug}
func (h *Content) GetTrack(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.GetTrack(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "该赛道不存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "服务器开小差了，请稍后再试")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// --- 名场面 ---

// ListMoments GET /api/v1/moments
func (h *Content) ListMoments(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListMoments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "服务器开小差了，请稍后再试")
		return
	}
	writeJSON(w, http.StatusOK, momentListResponse{Items: items})
}

// GetMoment GET /api/v1/moments/{slug}
func (h *Content) GetMoment(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.GetMoment(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "该名场面不存在")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "服务器开小差了，请稍后再试")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// --- 响应与查询解析 ---

// 列表响应包裹层（OpenAPI *List 组件）。
type driverListResponse struct {
	Items []domain.DriverSummary `json:"items"`
}

type trackListResponse struct {
	Items []domain.TrackSummary `json:"items"`
}

type momentListResponse struct {
	Items []domain.MomentSummary `json:"items"`
}

// parseFeatured 解析 featured 查询参数：
// 缺省或 false → 不过滤；true → 仅精选；其他值 → 报错。
func parseFeatured(raw string) (*bool, error) {
	switch raw {
	case "", "false":
		return nil, nil
	case "true":
		return boolPtr(true), nil
	default:
		return nil, fmt.Errorf("invalid featured value %q", raw)
	}
}

func boolPtr(v bool) *bool { return &v }
