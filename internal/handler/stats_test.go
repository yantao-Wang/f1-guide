package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

func newTestStatsHandler(client *testutil.FakeStatsClient) *Stats {
	return NewStats(service.NewStats(client, nil))
}

func TestScheduleEndpoint(t *testing.T) {
	client := &testutil.FakeStatsClient{}
	h := newTestStatsHandler(client)

	rec := httptest.NewRecorder()
	h.Schedule(rec, httptest.NewRequest(http.MethodGet, "/api/v1/schedule", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var sched domain.Schedule
	if err := json.Unmarshal(rec.Body.Bytes(), &sched); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if sched.Season != time.Now().Year() {
		t.Errorf("season = %d, want %d（缺省当前赛季）", sched.Season, time.Now().Year())
	}
	if len(sched.Races) != 3 {
		t.Fatalf("races = %d, want 3", len(sched.Races))
	}
	r1 := sched.Races[0]
	if r1.Status != domain.RaceStatusCompleted || r1.WinnerSlug != "max-verstappen" {
		t.Errorf("第 1 场 = %+v", r1)
	}
	if r1.WinnerName != "" {
		t.Errorf("WinnerName 泄漏进 JSON 响应: %+v", r1)
	}
	if sched.Races[1].Status != domain.RaceStatusCancelled {
		t.Errorf("第 2 场 status = %q, want cancelled", sched.Races[1].Status)
	}
	if sched.Races[2].Status != domain.RaceStatusUpcoming {
		t.Errorf("第 3 场 status = %q, want upcoming", sched.Races[2].Status)
	}
	// 缺省赛季在服务层解析后传给上游
	if len(client.ScheduleCalls) == 0 || client.ScheduleCalls[0] != time.Now().Year() {
		t.Errorf("ScheduleCalls = %v, want [%d]", client.ScheduleCalls, time.Now().Year())
	}
}

func TestStandingsEndpoints(t *testing.T) {
	client := &testutil.FakeStatsClient{}
	h := newTestStatsHandler(client)

	rec := httptest.NewRecorder()
	h.DriverStandings(rec, httptest.NewRequest(http.MethodGet, "/api/v1/standings/drivers?season=2026", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("drivers status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var ds domain.DriverStandings
	if err := json.Unmarshal(rec.Body.Bytes(), &ds); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if len(ds.Items) != 6 || ds.Items[0].DriverName != "维斯塔潘" || ds.Items[0].Team.Name != "红牛" {
		t.Errorf("车手积分榜 = %+v", ds.Items)
	}

	rec2 := httptest.NewRecorder()
	h.ConstructorStandings(rec2, httptest.NewRequest(http.MethodGet, "/api/v1/standings/constructors", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("constructors status = %d, body = %s", rec2.Code, rec2.Body.String())
	}
	var cs domain.ConstructorStandings
	if err := json.Unmarshal(rec2.Body.Bytes(), &cs); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if len(cs.Items) != 3 || cs.Items[0].Team.Name != "梅赛德斯" {
		t.Errorf("车队积分榜 = %+v", cs.Items)
	}
}

func TestSeasonParamValidation(t *testing.T) {
	client := &testutil.FakeStatsClient{}
	h := newTestStatsHandler(client)

	cases := []struct {
		target string
		want   int
	}{
		{"/api/v1/schedule", http.StatusOK},
		{"/api/v1/schedule?season=2026", http.StatusOK},
		{"/api/v1/schedule?season=1950", http.StatusOK},
		{"/api/v1/schedule?season=abc", http.StatusBadRequest},
		{"/api/v1/schedule?season=1949", http.StatusBadRequest},
		{"/api/v1/schedule?season=" + strings.Repeat("9", 5), http.StatusBadRequest}, // 超出 int 范围
	}
	future := time.Now().Year() + 1
	cases = append(cases, struct {
		target string
		want   int
	}{"/api/v1/schedule?season=" + strconv.Itoa(future), http.StatusBadRequest})

	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.Schedule(rec, httptest.NewRequest(http.MethodGet, c.target, nil))
		if rec.Code != c.want {
			t.Errorf("GET %s status = %d, want %d", c.target, rec.Code, c.want)
		}
	}
}

func TestStatsUpstreamUnavailable(t *testing.T) {
	h := newTestStatsHandler(&testutil.FakeStatsClient{Err: errors.New("upstream down")})

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/api/v1/schedule":               h.Schedule,
		"/api/v1/standings/drivers":      h.DriverStandings,
		"/api/v1/standings/constructors": h.ConstructorStandings,
	}
	for target, fn := range handlers {
		rec := httptest.NewRecorder()
		fn(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusBadGateway {
			t.Errorf("GET %s status = %d, want 502", target, rec.Code)
			continue
		}
		var body errorBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("GET %s 错误响应不是合法 JSON: %v", target, err)
		}
		if body.Code != "upstream_unavailable" {
			t.Errorf("GET %s code = %q, want upstream_unavailable", target, body.Code)
		}
	}
}
