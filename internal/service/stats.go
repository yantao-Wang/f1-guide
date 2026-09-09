package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/pkg/f1api"
)

// ErrUnavailable 表示上游赛事数据源（Jolpica）暂不可用。handler 将其映射为 502。
var ErrUnavailable = errors.New("service: stats upstream unavailable")

// 缓存 TTL。Jolpica 正赛后约 1 小时更新数据，站内无需实时。
const (
	standingsTTL = 30 * time.Minute
	scheduleTTL  = time.Hour
)

// StatsClient 上游赛事数据接口（消费方定义，pkg/f1api 提供实现）。
type StatsClient interface {
	Schedule(ctx context.Context, season int) ([]f1api.ScheduleRace, error)
	Winners(ctx context.Context, season int) ([]f1api.WinnerRace, error)
	DriverStandings(ctx context.Context, season int) ([]f1api.Standing, error)
	ConstructorStandings(ctx context.Context, season int) ([]f1api.ConstructorStanding, error)
}

// DriverMappingStore 站内车手映射数据源（消费方定义，repository.Postgres 实现）。
// 后台录入新车手时填写 Jolpica ID，积分榜/赛程即可动态挂接站内故事链接。
type DriverMappingStore interface {
	ListDriverMappings(ctx context.Context) ([]domain.DriverMapping, error)
}

// Stats 赛事数据服务：上游原始数据 → 领域模型，附带本地 TTL 缓存。
type Stats struct {
	client StatsClient
	maps   DriverMappingStore // 可为 nil：仅代码表兜底
	now    func() time.Time
	mu     sync.Mutex
	cache  map[string]cacheEntry
}

type cacheEntry struct {
	value   any
	expires time.Time
}

// NewStats 组装赛事数据服务。maps 为站内车手映射源（nil 时仅用代码表兜底）。
func NewStats(client StatsClient, maps DriverMappingStore) *Stats {
	return &Stats{client: client, maps: maps, now: time.Now, cache: make(map[string]cacheEntry)}
}

// withNow 注入时钟（测试用）。
func (s *Stats) withNow(now func() time.Time) *Stats {
	s.now = now
	return s
}

// resolveSeason 把 0（缺省）解析为当前年份。
func (s *Stats) resolveSeason(season int) int {
	if season <= 0 {
		return s.now().Year()
	}
	return season
}

// loadMappings 拉取 DB 车手映射（Jolpica driverId → 站内车手）。
// 失败返回 nil，调用方记录警告后走代码表兜底——积分榜不因 DB 故障瘫痪。
func (s *Stats) loadMappings(ctx context.Context) map[string]domain.DriverMapping {
	if s.maps == nil {
		return nil
	}
	items, err := s.maps.ListDriverMappings(ctx)
	if err != nil {
		slog.Warn("driver mappings unavailable, fallback to code tables", "error", err)
		return nil
	}
	m := make(map[string]domain.DriverMapping, len(items))
	for _, it := range items {
		m[it.JolpicaID] = it
	}
	return m
}

// get 命中缓存直接返回；未命中则调用 fetch。
// 并发未命中时可能重复拉取，代价可接受（上游限速 500 req/h）。
func (s *Stats) get(key string, ttl time.Duration, fetch func() (any, error)) (any, error) {
	s.mu.Lock()
	if e, ok := s.cache[key]; ok && s.now().Before(e.expires) {
		s.mu.Unlock()
		return e.value, nil
	}
	s.mu.Unlock()

	v, err := fetch()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[key] = cacheEntry{value: v, expires: s.now().Add(ttl)}
	s.mu.Unlock()
	return v, nil
}

// Schedule 返回赛季赛程（已完赛场次附冠军摘要）。上游故障时返回 ErrUnavailable。
func (s *Stats) Schedule(ctx context.Context, season int) (domain.Schedule, error) {
	season = s.resolveSeason(season)
	v, err := s.get(fmt.Sprintf("schedule:%d", season), scheduleTTL, func() (any, error) {
		return s.buildSchedule(ctx, season)
	})
	if err != nil {
		return domain.Schedule{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return v.(domain.Schedule), nil
}

// DriverStandings 返回车手积分榜。上游故障时返回 ErrUnavailable。
func (s *Stats) DriverStandings(ctx context.Context, season int) (domain.DriverStandings, error) {
	season = s.resolveSeason(season)
	v, err := s.get(fmt.Sprintf("driverStandings:%d", season), standingsTTL, func() (any, error) {
		return s.buildDriverStandings(ctx, season)
	})
	if err != nil {
		return domain.DriverStandings{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return v.(domain.DriverStandings), nil
}

// ConstructorStandings 返回车队积分榜。上游故障时返回 ErrUnavailable。
func (s *Stats) ConstructorStandings(ctx context.Context, season int) (domain.ConstructorStandings, error) {
	season = s.resolveSeason(season)
	v, err := s.get(fmt.Sprintf("constructorStandings:%d", season), standingsTTL, func() (any, error) {
		return s.buildConstructorStandings(ctx, season)
	})
	if err != nil {
		return domain.ConstructorStandings{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return v.(domain.ConstructorStandings), nil
}

// buildSchedule 拉取赛程 + 冠军清单，按日期与赛果推算每场状态：
// 有冠军 → completed；日期已过且无冠军 → cancelled；其余 → upcoming。
func (s *Stats) buildSchedule(ctx context.Context, season int) (domain.Schedule, error) {
	races, err := s.client.Schedule(ctx, season)
	if err != nil {
		return domain.Schedule{}, err
	}
	winners, err := s.client.Winners(ctx, season)
	if err != nil {
		return domain.Schedule{}, err
	}
	winnerByRound := make(map[int]f1api.Driver, len(winners))
	for _, wr := range winners {
		round, err := atoi(wr.Round)
		if err != nil {
			return domain.Schedule{}, fmt.Errorf("bad winner round %q: %w", wr.Round, err)
		}
		if len(wr.Results) > 0 {
			winnerByRound[round] = wr.Results[0].Driver
		}
	}

	maps := s.loadMappings(ctx)

	today := s.now().Format("2006-01-02")
	out := make([]domain.Race, 0, len(races))
	for _, r := range races {
		round, err := atoi(r.Round)
		if err != nil {
			return domain.Schedule{}, fmt.Errorf("bad race round %q: %w", r.Round, err)
		}
		race := domain.Race{
			Round:     round,
			Date:      r.Date,
			GrandPrix: grandPrixZh(r.RaceName),
			Circuit:   circuitZh(r.Circuit.CircuitName),
			Country:   countryZh(r.Circuit.Location.Country),
		}
		switch winner, ok := winnerByRound[round]; {
		case ok:
			race.Status = domain.RaceStatusCompleted
			race.WinnerSlug = driverSlugWith(maps, winner.DriverID)
			race.WinnerName = driverNameZhWith(maps, winner.DriverID, winner.GivenName, winner.FamilyName)
		case r.Date < today:
			race.Status = domain.RaceStatusCancelled
		default:
			race.Status = domain.RaceStatusUpcoming
		}
		out = append(out, race)
	}
	return domain.Schedule{Season: season, Races: out}, nil
}

// buildDriverStandings 拉取并映射车手积分榜（上游顺序即名次，仍防御性排序）。
func (s *Stats) buildDriverStandings(ctx context.Context, season int) (domain.DriverStandings, error) {
	rows, err := s.client.DriverStandings(ctx, season)
	if err != nil {
		return domain.DriverStandings{}, err
	}
	maps := s.loadMappings(ctx)
	items := make([]domain.DriverStandingRow, 0, len(rows))
	for _, r := range rows {
		pos, err := atoi(r.Position)
		if err != nil {
			return domain.DriverStandings{}, fmt.Errorf("bad position %q: %w", r.Position, err)
		}
		points, err := atoi(r.Points)
		if err != nil {
			return domain.DriverStandings{}, fmt.Errorf("bad points %q: %w", r.Points, err)
		}
		wins, err := atoi(r.Wins)
		if err != nil {
			return domain.DriverStandings{}, fmt.Errorf("bad wins %q: %w", r.Wins, err)
		}
		team := ""
		if len(r.Constructors) > 0 {
			team = r.Constructors[0].Name
		}
		items = append(items, domain.DriverStandingRow{
			Position:   pos,
			DriverSlug: driverSlugWith(maps, r.Driver.DriverID),
			DriverName: driverNameZhWith(maps, r.Driver.DriverID, r.Driver.GivenName, r.Driver.FamilyName),
			Team:       domain.Team{Name: constructorZh(team), Color: constructorColor(team)},
			Points:     points,
			Wins:       wins,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Position < items[j].Position })
	return domain.DriverStandings{Season: season, Items: items}, nil
}

// buildConstructorStandings 拉取并映射车队积分榜。
func (s *Stats) buildConstructorStandings(ctx context.Context, season int) (domain.ConstructorStandings, error) {
	rows, err := s.client.ConstructorStandings(ctx, season)
	if err != nil {
		return domain.ConstructorStandings{}, err
	}
	items := make([]domain.ConstructorStandingRow, 0, len(rows))
	for _, r := range rows {
		pos, err := atoi(r.Position)
		if err != nil {
			return domain.ConstructorStandings{}, fmt.Errorf("bad position %q: %w", r.Position, err)
		}
		points, err := atoi(r.Points)
		if err != nil {
			return domain.ConstructorStandings{}, fmt.Errorf("bad points %q: %w", r.Points, err)
		}
		wins, err := atoi(r.Wins)
		if err != nil {
			return domain.ConstructorStandings{}, fmt.Errorf("bad wins %q: %w", r.Wins, err)
		}
		items = append(items, domain.ConstructorStandingRow{
			Position: pos,
			Team:     domain.Team{Name: constructorZh(r.Constructor.Name), Color: constructorColor(r.Constructor.Name)},
			Points:   points,
			Wins:     wins,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Position < items[j].Position })
	return domain.ConstructorStandings{Season: season, Items: items}, nil
}

// atoi 解析上游字符串数值（严格模式：坏数据直接报错）。
func atoi(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}
