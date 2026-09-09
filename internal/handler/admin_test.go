package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/middleware"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

// newTestAdmin 组装测试用后台处理器（上传目录为临时目录）。
func newTestAdmin(t *testing.T) (*Admin, *testutil.FakeAdminStore, *testutil.FakeStore) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &testutil.FakeStore{}
	adminStore := &testutil.FakeAdminStore{}
	svc := service.NewContent(store, store, store)
	auth := middleware.NewAdmin("test-password", "test-secret")
	h := NewAdmin(service.NewAdmin(adminStore), svc, auth, t.TempDir(), log)
	return h, adminStore, store
}

// csrfToken 从登录页 GET 响应中提取 CSRF token。
func csrfToken(t *testing.T, h *Admin) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.LoginPage(rec, httptest.NewRequest(http.MethodGet, "/admin/login", nil))
	for _, c := range rec.Result().Cookies() {
		if c.Name == "f1admin_csrf" {
			return c.Value
		}
	}
	t.Fatal("登录页未下发 CSRF Cookie")
	return ""
}

// loginRequest 构造登录 POST 请求（携带 CSRF token 与密码）。
func loginRequest(t *testing.T, h *Admin, password string) *http.Request {
	t.Helper()
	body := url.Values{"csrf": {csrfToken(t, h)}, "password": {password}}.Encode()
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestAdminLoginPage(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	rec := httptest.NewRecorder()
	h.LoginPage(rec, httptest.NewRequest(http.MethodGet, "/admin/login", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "name=\"csrf\"") {
		t.Fatal("登录页应包含 CSRF 隐藏字段")
	}
	if !strings.Contains(rec.Body.String(), "管理后台") {
		t.Fatal("登录页应使用后台骨架")
	}
}

func TestAdminLoginPageRedirectWhenAuthed(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	// 先登录拿会话 Cookie
	rec := httptest.NewRecorder()
	h.Login(rec, loginRequest(t, h, "test-password"))
	cookies := rec.Result().Cookies()

	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec = httptest.NewRecorder()
	h.LoginPage(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin" {
		t.Fatalf("已登录访问登录页 status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAdminLoginSuccess(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	rec := httptest.NewRecorder()
	h.Login(rec, loginRequest(t, h, "test-password"))

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin" {
		t.Fatalf("status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "f1admin" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("登录成功应签发会话 Cookie")
	}
}

func TestAdminLoginWrongPassword(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	rec := httptest.NewRecorder()
	h.Login(rec, loginRequest(t, h, "wrong"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "密码错误") {
		t.Fatal("应提示密码错误")
	}
}

func TestAdminLoginRateLimited(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	// 连续 5 次错误密码：前 5 次 401，第 6 次 429
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		h.Login(rec, loginRequest(t, h, "wrong"))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("第 %d 次 status = %d, want 401", i+1, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.Login(rec, loginRequest(t, h, "wrong"))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("限流后 status = %d, want 429", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "尝试过于频繁") {
		t.Fatal("应提示限流文案")
	}
}

func TestAdminLogout(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	rec := httptest.NewRecorder()
	h.Logout(rec, httptest.NewRequest(http.MethodPost, "/admin/logout", nil))

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "f1admin" && c.MaxAge >= 0 {
			t.Fatal("登出应清除会话 Cookie")
		}
	}
}

func TestAdminDashboard(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)
	adminStore.Stats = domain.AdminStats{
		DriverCount:   2,
		TrackCount:    1,
		MomentCount:   3,
		FeaturedCount: 1,
		Recent: []domain.RecentUpdate{
			{Kind: "driver", Slug: "zhou-guanyu", Name: "周冠宇", UpdatedAt: time.Now()},
		},
		Issues: []domain.ContentIssue{
			{Kind: "track", Slug: "shanghai", Name: "上海", Issue: "缺少赛道图", Severity: "建议"},
			{Kind: "driver", Slug: "x", Name: "X", Issue: "正文缺失或仍是占位内容", Severity: "缺失"},
		},
	}

	rec := httptest.NewRecorder()
	h.Dashboard(rec, httptest.NewRequest(http.MethodGet, "/admin", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"车手故事", "赛道图鉴", "名场面", "首页精选",
		"周冠宇", "缺少赛道图", "正文缺失或仍是占位内容",
		"/admin/drivers/zhou-guanyu/edit",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("仪表盘应包含 %q", want)
		}
	}
}

func TestAdminDashboardStatsError(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)
	adminStore.Err = io.ErrUnexpectedEOF

	rec := httptest.NewRecorder()
	h.Dashboard(rec, httptest.NewRequest(http.MethodGet, "/admin", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
