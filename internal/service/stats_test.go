package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/pkg/f1api"
)

// countingClient 记录调用次数与 season 实参的 StatsClient fake。
type countingClient struct {
	scheduleCalls             []int
	winnersCalls              []int
	driverStandingsCalls      []int
	constructorStandingsCalls []int

	schedule             []f1api.ScheduleRace
	winners              []f1api.WinnerRace
	driverStandings      []f1api.Standing
	constructorStandings []f1api.ConstructorStanding
	err                  error
}

func (c *countingClient) Schedule(_ context.Context, season int) ([]f1api.ScheduleRace, error) {
	c.scheduleCalls = append(c.scheduleCalls, season)
	return c.schedule, c.err
}

func (c *countingClient) Winners(_ context.Context, season int) ([]f1api.WinnerRace, error) {
	c.winnersCalls = append(c.winnersCalls, season)
	return c.winners, c.err
}

func (c *countingClient) DriverStandings(_ context.Context, season int) ([]f1api.Standing, error) {
	c.driverStandingsCalls = append(c.driverStandingsCalls, season)
	return c.driverStandings, c.err
}

func (c *countingClient) ConstructorStandings(_ context.Context, season int) ([]f1api.ConstructorStanding, error) {
	c.constructorStandingsCalls = append(c.constructorStandingsCalls, season)
	return c.constructorStandings, c.err
}

// fixedClock 固定 2026-09-08 的测试时钟。
func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC) }
}

func newTestStats(client StatsClient) *Stats {
	return NewStats(client).withNow(fixedClock())
}

func sampleClient() *countingClient {
	return &countingClient{
		schedule: []f1api.ScheduleRace{
			{
				Round: "1", RaceName: "Australian Grand Prix", Date: "2026-03-08",
				Circuit: f1api.Circuit{CircuitName: "Albert Park Grand Prix Circuit", Location: f1api.Location{Country: "Australia"}},
			},
			{
				Round: "13", RaceName: "Italian Grand Prix", Date: "2026-09-06",
				Circuit: f1api.Circuit{CircuitName: "Autodromo Nazionale di Monza", Location: f1api.Location{Country: "Italy"}},
			},
			{
				Round: "14", RaceName: "Spanish Grand Prix", Date: "2026-09-13",
				Circuit: f1api.Circuit{CircuitName: "Madring", Location: f1api.Location{Country: "Spain"}},
			},
			{
				Round: "16", RaceName: "Bahrain Grand Prix in Malaysia", Date: "2026-10-04",
				Circuit: f1api.Circuit{CircuitName: "Sepang International Circuit", Location: f1api.Location{Country: "Malaysia"}},
			},
		},
		// 冠军只有第 1 站：第 13 站日期已过却无赛果 → cancelled。
		winners: []f1api.WinnerRace{
			{
				Round: "1",
				Results: []f1api.WinnerEntry{
					{Position: "1", Driver: f1api.Driver{DriverID: "max_verstappen", GivenName: "Max", FamilyName: "Verstappen"}},
				},
			},
		},
		driverStandings: []f1api.Standing{
			{
				Position: "2", Points: "201", Wins: "2",
				Driver:       f1api.Driver{DriverID: "max_verstappen", GivenName: "Max", FamilyName: "Verstappen"},
				Constructors: []f1api.Constructor{{Name: "Red Bull"}},
			},
			{
				Position: "1", Points: "267", Wins: "7",
				Driver:       f1api.Driver{DriverID: "antonelli", GivenName: "Andrea Kimi", FamilyName: "Antonelli"},
				Constructors: []f1api.Constructor{{Name: "Mercedes"}},
			},
			{
				Position: "3", Points: "5", Wins: "0",
				Driver:       f1api.Driver{DriverID: "unknown_driver", GivenName: "Nemo", FamilyName: "Unknown"},
				Constructors: []f1api.Constructor{{Name: "Team X"}},
			},
		},
		constructorStandings: []f1api.ConstructorStanding{
			{Position: "1", Points: "468", Wins: "9", Constructor: f1api.Constructor{Name: "Mercedes"}},
			{Position: "11", Points: "0", Wins: "0", Constructor: f1api.Constructor{Name: "Cadillac F1 Team"}},
		},
	}
}

func TestScheduleStatusComputation(t *testing.T) {
	stats := newTestStats(sampleClient())

	sched, err := stats.Schedule(context.Background(), 0)
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if sched.Season != 2026 {
		t.Errorf("season = %d, want 2026（缺省解析为当前年份）", sched.Season)
	}
	want := map[int]domain.RaceStatus{
		1:  domain.RaceStatusCompleted,
		13: domain.RaceStatusCancelled,
		14: domain.RaceStatusUpcoming,
		16: domain.RaceStatusUpcoming,
	}
	for _, race := range sched.Races {
		if got, ok := want[race.Round]; !ok || race.Status != got {
			t.Errorf("round %d status = %q, want %q", race.Round, race.Status, got)
		}
	}
	r1 := sched.Races[0]
	if r1.GrandPrix != "澳大利亚大奖赛" || r1.Circuit != "阿尔伯特公园赛道" || r1.Country != "澳大利亚" {
		t.Errorf("中文映射错误: %+v", r1)
	}
	if r1.WinnerSlug != "max-verstappen" || r1.WinnerName != "维斯塔潘" {
		t.Errorf("冠军映射错误: %+v", r1)
	}
	if r16 := sched.Races[3]; r16.GrandPrix != "巴林大奖赛（马来西亚）" || r16.Circuit != "雪邦国际赛道" {
		t.Errorf("马来西亚站映射错误: %+v", r16)
	}
}

func TestDriverStandingsMapping(t *testing.T) {
	stats := newTestStats(sampleClient())

	ds, err := stats.DriverStandings(context.Background(), 2026)
	if err != nil {
		t.Fatalf("DriverStandings: %v", err)
	}
	if len(ds.Items) != 3 {
		t.Fatalf("len = %d, want 3", len(ds.Items))
	}
	// 防御性排序：上游乱序输入，输出按名次
	if ds.Items[0].Position != 1 || ds.Items[0].DriverName != "安东内利" {
		t.Errorf("第 1 行 = %+v", ds.Items[0])
	}
	row := ds.Items[1]
	if row.DriverSlug != "max-verstappen" || row.DriverName != "维斯塔潘" {
		t.Errorf("slug/中文名映射错误: %+v", row)
	}
	if row.Team.Name != "红牛" || row.Team.Color != "#1E41FF" {
		t.Errorf("车队映射错误: %+v", row.Team)
	}
	// 未知车手/车队回退：英文名 + 兜底色
	fallback := ds.Items[2]
	if fallback.DriverName != "Nemo Unknown" || fallback.DriverSlug != "unknown-driver" {
		t.Errorf("未知车手回退错误: %+v", fallback)
	}
	if fallback.Team.Name != "Team X" || fallback.Team.Color != fallbackColor {
		t.Errorf("未知车队回退错误: %+v", fallback.Team)
	}
}

func TestConstructorStandingsMapping(t *testing.T) {
	stats := newTestStats(sampleClient())

	cs, err := stats.ConstructorStandings(context.Background(), 0)
	if err != nil {
		t.Fatalf("ConstructorStandings: %v", err)
	}
	if len(cs.Items) != 2 {
		t.Fatalf("len = %d, want 2", len(cs.Items))
	}
	if cs.Items[0].Team.Name != "梅赛德斯" || cs.Items[0].Team.Color != "#27F4D2" {
		t.Errorf("第 1 行 = %+v", cs.Items[0])
	}
	if cs.Items[1].Team.Name != "凯迪拉克" || cs.Items[1].Team.Color != "#00A1E0" {
		t.Errorf("凯迪拉克映射错误: %+v", cs.Items[1])
	}
}

func TestStatsCache(t *testing.T) {
	client := sampleClient()
	stats := newTestStats(client)

	if _, err := stats.Schedule(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := stats.Schedule(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	// 第二次命中缓存，不再访问上游
	if len(client.scheduleCalls) != 1 || len(client.winnersCalls) != 1 {
		t.Errorf("calls = %d/%d, want 1/1", len(client.scheduleCalls), len(client.winnersCalls))
	}
	// 缺省赛季解析：客户端收到 2026 而非 0
	if client.scheduleCalls[0] != 2026 {
		t.Errorf("season = %d, want 2026", client.scheduleCalls[0])
	}

	// 积分榜同理
	for i := 0; i < 3; i++ {
		if _, err := stats.DriverStandings(context.Background(), 0); err != nil {
			t.Fatal(err)
		}
	}
	if len(client.driverStandingsCalls) != 1 {
		t.Errorf("driverStandingsCalls = %d, want 1", len(client.driverStandingsCalls))
	}
}

func TestStatsCacheExpiry(t *testing.T) {
	client := sampleClient()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	stats := NewStats(client).withNow(func() time.Time {
		now = now.Add(time.Hour) // 每次调用推进 1 小时
		return now
	})

	if _, err := stats.Schedule(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	// TTL 1 小时：时钟推进 1 小时后过期，重新拉取
	if _, err := stats.Schedule(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if len(client.scheduleCalls) != 2 {
		t.Errorf("scheduleCalls = %d, want 2（TTL 过期后重拉）", len(client.scheduleCalls))
	}
}

func TestStatsUnavailable(t *testing.T) {
	client := sampleClient()
	client.err = errors.New("upstream down")
	stats := newTestStats(client)

	if _, err := stats.Schedule(context.Background(), 0); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Schedule err = %v, want ErrUnavailable", err)
	}
	if _, err := stats.DriverStandings(context.Background(), 0); !errors.Is(err, ErrUnavailable) {
		t.Errorf("DriverStandings err = %v, want ErrUnavailable", err)
	}
	if _, err := stats.ConstructorStandings(context.Background(), 0); !errors.Is(err, ErrUnavailable) {
		t.Errorf("ConstructorStandings err = %v, want ErrUnavailable", err)
	}
	// 失败不写缓存：恢复后立即可用
	client.err = nil
	if _, err := stats.DriverStandings(context.Background(), 0); err != nil {
		t.Errorf("恢复后仍失败: %v", err)
	}
}

func TestStatsBadNumericData(t *testing.T) {
	client := sampleClient()
	client.driverStandings[0].Points = "not-a-number"
	stats := newTestStats(client)

	if _, err := stats.DriverStandings(context.Background(), 0); !errors.Is(err, ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable（坏数据视为上游异常）", err)
	}
}
