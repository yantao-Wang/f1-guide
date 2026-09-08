package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/yantao-Wang/f1-guide/internal/service"
)

// Stats 赛事数据 HTTP 处理器（赛程 / 积分榜，数据来自 Jolpica）。
type Stats struct {
	svc *service.Stats
}

// NewStats 创建赛事数据处理器。
func NewStats(svc *service.Stats) *Stats {
	return &Stats{svc: svc}
}

// Schedule GET /api/v1/schedule
func (h *Stats) Schedule(w http.ResponseWriter, r *http.Request) {
	season, err := parseSeason(r.URL.Query().Get("season"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", "season 参数必须是合法年份")
		return
	}
	sched, err := h.svc.Schedule(r.Context(), season)
	if err != nil {
		writeStatsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sched)
}

// DriverStandings GET /api/v1/standings/drivers
func (h *Stats) DriverStandings(w http.ResponseWriter, r *http.Request) {
	season, err := parseSeason(r.URL.Query().Get("season"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", "season 参数必须是合法年份")
		return
	}
	standings, err := h.svc.DriverStandings(r.Context(), season)
	if err != nil {
		writeStatsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, standings)
}

// ConstructorStandings GET /api/v1/standings/constructors
func (h *Stats) ConstructorStandings(w http.ResponseWriter, r *http.Request) {
	season, err := parseSeason(r.URL.Query().Get("season"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", "season 参数必须是合法年份")
		return
	}
	standings, err := h.svc.ConstructorStandings(r.Context(), season)
	if err != nil {
		writeStatsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, standings)
}

// errSeasonRange 表示年份超出合法范围（F1 始于 1950，且不存在未来赛季数据）。
var errSeasonRange = errors.New("season out of range")

// parseSeason 解析 season 查询参数：缺省为 0（服务层解析为当前赛季），
// 否则必须是 1950 至当前年份的整数。
func parseSeason(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	season, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if season < 1950 || season > time.Now().Year() {
		return 0, errSeasonRange
	}
	return season, nil
}

// writeStatsError 上游数据源不可用 → 502；其他错误 → 500。
func writeStatsError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrUnavailable) {
		writeError(w, http.StatusBadGateway, "upstream_unavailable", "赛事数据源暂不可用，请稍后再试")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal", "服务器开小差了，请稍后再试")
}
