package f1api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const scheduleJSON = `{"MRData":{"RaceTable":{"season":"2026","Races":[
  {"round":"1","raceName":"Australian Grand Prix","date":"2026-03-08",
   "Circuit":{"circuitName":"Albert Park Grand Prix Circuit","Location":{"country":"Australia"}}}
]}}}`

const winnersJSON = `{"MRData":{"RaceTable":{"season":"2026","Races":[
  {"round":"1","Results":[{"position":"1","Driver":{"driverId":"russell","givenName":"George","familyName":"Russell"}}]}
]}}}`

const driverStandingsJSON = `{"MRData":{"StandingsTable":{"season":"2026","StandingsLists":[
  {"season":"2026","round":"18",
   "DriverStandings":[{"position":"1","points":"267","wins":"7",
     "Driver":{"driverId":"antonelli","givenName":"Andrea Kimi","familyName":"Antonelli"},
     "Constructors":[{"constructorId":"mercedes","name":"Mercedes"}]}]}
]}}}`

const constructorStandingsJSON = `{"MRData":{"StandingsTable":{"season":"2026","StandingsLists":[
  {"season":"2026","round":"18",
   "ConstructorStandings":[{"position":"1","points":"468","wins":"9",
     "Constructor":{"constructorId":"mercedes","name":"Mercedes"}}]}
]}}}`

// newTestClient 起一个回放固定 JSON 的 httptest 服务并记录请求路径。
func newTestClient(t *testing.T) (*Client, *[]string) {
	t.Helper()
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/results/1.json"):
			_, _ = w.Write([]byte(winnersJSON))
		case strings.HasSuffix(r.URL.Path, "/driverStandings.json"):
			_, _ = w.Write([]byte(driverStandingsJSON))
		case strings.HasSuffix(r.URL.Path, "/constructorStandings.json"):
			_, _ = w.Write([]byte(constructorStandingsJSON))
		case strings.HasSuffix(r.URL.Path, ".json"):
			_, _ = w.Write([]byte(scheduleJSON))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client, &paths
}

func TestSeasonPaths(t *testing.T) {
	ctx := context.Background()
	client, paths := newTestClient(t)

	if _, err := client.Schedule(ctx, 0); err != nil {
		t.Fatalf("Schedule(current): %v", err)
	}
	if _, err := client.Schedule(ctx, 2025); err != nil {
		t.Fatalf("Schedule(2025): %v", err)
	}
	if _, err := client.DriverStandings(ctx, 0); err != nil {
		t.Fatalf("DriverStandings: %v", err)
	}
	if _, err := client.ConstructorStandings(ctx, 0); err != nil {
		t.Fatalf("ConstructorStandings: %v", err)
	}
	if _, err := client.Winners(ctx, 0); err != nil {
		t.Fatalf("Winners: %v", err)
	}

	want := []string{
		"/ergast/f1/current.json",
		"/ergast/f1/2025.json",
		"/ergast/f1/current/driverStandings.json",
		"/ergast/f1/current/constructorStandings.json",
		"/ergast/f1/current/results/1.json",
	}
	if len(*paths) != len(want) {
		t.Fatalf("paths = %v, want %v", *paths, want)
	}
	for i, p := range want {
		if (*paths)[i] != p {
			t.Errorf("paths[%d] = %q, want %q", i, (*paths)[i], p)
		}
	}
}

func TestScheduleDecode(t *testing.T) {
	client, _ := newTestClient(t)
	races, err := client.Schedule(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if len(races) != 1 {
		t.Fatalf("len = %d, want 1", len(races))
	}
	r := races[0]
	if r.Round != "1" || r.RaceName != "Australian Grand Prix" || r.Date != "2026-03-08" {
		t.Errorf("race = %+v", r)
	}
	if r.Circuit.CircuitName != "Albert Park Grand Prix Circuit" || r.Circuit.Location.Country != "Australia" {
		t.Errorf("circuit = %+v", r.Circuit)
	}
}

func TestWinnersDecode(t *testing.T) {
	client, _ := newTestClient(t)
	winners, err := client.Winners(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Winners: %v", err)
	}
	if len(winners) != 1 || len(winners[0].Results) != 1 {
		t.Fatalf("winners = %+v", winners)
	}
	if got := winners[0].Results[0].Driver.DriverID; got != "russell" {
		t.Errorf("winner driverId = %q, want russell", got)
	}
}

func TestDriverStandingsDecode(t *testing.T) {
	client, _ := newTestClient(t)
	rows, err := client.DriverStandings(context.Background(), 2026)
	if err != nil {
		t.Fatalf("DriverStandings: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.Position != "1" || r.Points != "267" || r.Wins != "7" {
		t.Errorf("row = %+v", r)
	}
	if r.Driver.DriverID != "antonelli" || r.Constructors[0].Name != "Mercedes" {
		t.Errorf("driver/team = %+v / %+v", r.Driver, r.Constructors)
	}
}

func TestConstructorStandingsDecode(t *testing.T) {
	client, _ := newTestClient(t)
	rows, err := client.ConstructorStandings(context.Background(), 2026)
	if err != nil {
		t.Fatalf("ConstructorStandings: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len = %d, want 1", len(rows))
	}
	if rows[0].Constructor.Name != "Mercedes" || rows[0].Points != "468" {
		t.Errorf("row = %+v", rows[0])
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	client, err := New(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Schedule(context.Background(), 0)
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Errorf("err = %v, want status 500", err)
	}
}

func TestInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)
	client, err := New(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.Schedule(context.Background(), 0); err == nil {
		t.Error("want decode error, got nil")
	}
}

func TestNewBadBaseURL(t *testing.T) {
	if _, err := New("://bad"); err == nil {
		t.Error("want parse error, got nil")
	}
}
