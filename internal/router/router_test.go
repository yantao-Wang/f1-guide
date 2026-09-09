package router

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yantao-Wang/f1-guide/internal/handler"
	mw "github.com/yantao-Wang/f1-guide/internal/middleware"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

// newTestRouter 组装测试路由器（后台默认禁用）。
func newTestRouter() http.Handler {
	return newTestRouterWithAdmin(nil, "")
}

// newTestRouterWithAdmin 组装带后台的测试路由器（admin 非 nil 时启用 /admin；uploadDir 非空时启用 /uploads）。
func newTestRouterWithAdmin(admin *handler.Admin, uploadDir string) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &testutil.FakeStore{}
	svc := service.NewContent(store, store, store)
	statsSvc := service.NewStats(&testutil.FakeStatsClient{}, nil)
	return New(log, handler.NewHealth(log), handler.NewContent(svc), handler.NewStats(statsSvc), handler.NewPages(svc, statsSvc), admin, uploadDir)
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

// newTestAdminRouter 组装启用后台的测试路由器。
func newTestAdminRouter(t *testing.T) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &testutil.FakeStore{}
	svc := service.NewContent(store, store, store)
	statsSvc := service.NewStats(&testutil.FakeStatsClient{}, nil)
	auth := mw.NewAdmin("test-password", "test-secret")
	admin := handler.NewAdmin(service.NewAdmin(&testutil.FakeAdminStore{}), svc, auth, t.TempDir(), log)
	return New(log, handler.NewHealth(log), handler.NewContent(svc), handler.NewStats(statsSvc), handler.NewPages(svc, statsSvc), admin, "")
}

func TestAdminDisabledByDefault(t *testing.T) {
	r := newTestRouter() // admin = nil

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/login", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("后台禁用时 /admin/login status = %d, want 404", rec.Code)
	}
}

func TestUploadsServedWithoutAdmin(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "p.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 后台禁用（admin = nil）但 uploadDir 已设置：/uploads 应照常分发
	r := newTestRouterWithAdmin(nil, dir)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/p.png", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("后台禁用时 /uploads status = %d, want 200", rec.Code)
	}

	// uploadDir 未设置：/uploads 不注册（404）
	r = newTestRouter() // uploadDir = ""
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/p.png", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未设置 uploadDir 时 /uploads status = %d, want 404", rec.Code)
	}
}

func TestAdminRoutesAuthFlow(t *testing.T) {
	r := newTestAdminRouter(t)

	// 登录页公开可访问
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/login", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("登录页 status = %d, want 200", rec.Code)
	}

	// 未登录访问仪表盘：303 到登录页
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("未登录 /admin status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}

	// 登录：POST 携带 CSRF（先取登录页 Cookie）
	loginPage := httptest.NewRecorder()
	r.ServeHTTP(loginPage, httptest.NewRequest(http.MethodGet, "/admin/login", nil))
	body := url.Values{"password": {"test-password"}}
	for _, c := range loginPage.Result().Cookies() {
		if c.Name == "f1admin_csrf" {
			body.Set("csrf", c.Value)
		}
	}
	loginReq := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(body.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range loginPage.Result().Cookies() {
		loginReq.AddCookie(c)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, loginReq)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("登录 status = %d, want 303", rec.Code)
	}
	sessionCookies := rec.Result().Cookies()

	// 携会话 Cookie 访问仪表盘：200
	authed := httptest.NewRequest(http.MethodGet, "/admin", nil)
	for _, c := range sessionCookies {
		authed.AddCookie(c)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, authed)
	if rec.Code != http.StatusOK {
		t.Fatalf("已登录 /admin status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}

	// 后台 GET 页面路径表
	for _, path := range []string{"/admin/drivers", "/admin/drivers/new", "/admin/tracks", "/admin/moments"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		for _, c := range sessionCookies {
			req.AddCookie(c)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", path, rr.Code)
		}
	}
}
