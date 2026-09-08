package router

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yantao-Wang/f1-guide/internal/handler"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

func newTestRouter() http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &testutil.FakeStore{}
	svc := service.NewContent(store, store, store)
	statsSvc := service.NewStats(&testutil.FakeStatsClient{})
	return New(log, handler.NewHealth(log), handler.NewContent(svc), handler.NewStats(statsSvc), handler.NewPages(svc, statsSvc))
}

func TestHealthRoute(t *testing.T) {
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("body = %q, want %q", body, `{"status":"ok"}`)
	}
}

func TestUnknownRoute(t *testing.T) {
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestContentRoutesRegistered(t *testing.T) {
	r := newTestRouter()

	paths := []string{
		"/api/v1/drivers", "/api/v1/tracks", "/api/v1/moments",
		"/api/v1/schedule", "/api/v1/standings/drivers", "/api/v1/standings/constructors",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("GET %s content-type = %q, want application/json", path, ct)
		}
	}
}

func TestDataPagesRegistered(t *testing.T) {
	r := newTestRouter()

	for _, path := range []string{"/schedule", "/data"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("GET %s content-type = %q, want text/html", path, ct)
		}
	}
}
