package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

// serve 通过 chi 路由将请求打到指定 handler：pattern 是路由模式，target 是请求 URL（可带查询串）。
func serve(t *testing.T, pattern, target string, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	r.Get(pattern, h)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func newTestHandler(store *testutil.FakeStore) *Content {
	return NewContent(service.NewContent(store, store, store))
}

func TestListDriversOK(t *testing.T) {
	store := &testutil.FakeStore{Drivers: []domain.DriverSummary{{Slug: "a", Name: "A"}}}
	h := newTestHandler(store)

	rec := serve(t, "/api/v1/drivers", "/api/v1/drivers", h.ListDrivers)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp struct {
		Items []domain.DriverSummary `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Slug != "a" {
		t.Fatalf("items = %+v, want 1 条 [a]", resp.Items)
	}
}

func TestListDriversFeaturedTrue(t *testing.T) {
	store := &testutil.FakeStore{Drivers: []domain.DriverSummary{
		{Slug: "a", Featured: true},
		{Slug: "b", Featured: false},
	}}
	h := newTestHandler(store)

	rec := serve(t, "/api/v1/drivers", "/api/v1/drivers?featured=true", h.ListDrivers)

	var resp struct {
		Items []domain.DriverSummary `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Slug != "a" {
		t.Fatalf("items = %+v, want 仅精选 [a]", resp.Items)
	}
}

func TestListDriversInvalidFeatured(t *testing.T) {
	h := newTestHandler(&testutil.FakeStore{})

	rec := serve(t, "/api/v1/drivers", "/api/v1/drivers?featured=maybe", h.ListDrivers)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if resp.Code != "invalid_query" {
		t.Fatalf("code = %q, want invalid_query", resp.Code)
	}
}

func TestGetDriverNotFound(t *testing.T) {
	h := newTestHandler(&testutil.FakeStore{})

	rec := serve(t, "/api/v1/drivers/{slug}", "/api/v1/drivers/no-such-driver", h.GetDriver)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	var resp errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if resp.Code != "not_found" {
		t.Fatalf("code = %q, want not_found", resp.Code)
	}
}

func TestGetDriverOK(t *testing.T) {
	store := &testutil.FakeStore{DriverDetail: testutil.SampleDriver()}
	h := newTestHandler(store)

	r := chi.NewRouter()
	r.Get("/api/v1/drivers/{slug}", h.GetDriver)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/drivers/zhou-guanyu", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp domain.Driver
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if resp.Slug != "zhou-guanyu" || resp.Team.Name != "凯迪拉克" {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.FeaturedRace == nil || resp.FeaturedRace.Year != 2022 {
		t.Fatalf("FeaturedRace = %+v, want 2022", resp.FeaturedRace)
	}
}

func TestGetTrackOK(t *testing.T) {
	store := &testutil.FakeStore{TrackDetail: testutil.SampleTrack()}
	h := newTestHandler(store)

	r := chi.NewRouter()
	r.Get("/api/v1/tracks/{slug}", h.GetTrack)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tracks/shanghai", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp domain.Track
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if resp.LengthKm != 5.451 || len(resp.Highlights) != 2 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestGetMomentNotFound(t *testing.T) {
	h := newTestHandler(&testutil.FakeStore{})

	r := chi.NewRouter()
	r.Get("/api/v1/moments/{slug}", h.GetMoment)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/moments/no-such", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
