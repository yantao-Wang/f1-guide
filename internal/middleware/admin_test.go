package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIssueAndAuthenticateRoundTrip(t *testing.T) {
	a := NewAdmin("secret-password", "fixed-secret")
	rec := httptest.NewRecorder()
	a.IssueSession(rec)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	if !a.Authenticate(req) {
		t.Fatal("签发的会话应通过认证")
	}
}

func TestAuthenticateRejectsTamperedCookie(t *testing.T) {
	a := NewAdmin("secret-password", "fixed-secret")
	rec := httptest.NewRecorder()
	a.IssueSession(rec)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}

	// 篡改过期时间戳（保持原签名不变）
	cookie := cookies[0]
	cookie.Value = "9999999999" + cookie.Value[len("9999999999"):]
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(cookie)
	if a.Authenticate(req) {
		t.Fatal("篡改 payload 的 Cookie 应拒绝")
	}

	// 篡改签名
	cookie = cookies[0]
	cookie.Value = cookie.Value[:len(cookie.Value)-2] + "ff"
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(cookie)
	if a.Authenticate(req) {
		t.Fatal("篡改签名的 Cookie 应拒绝")
	}

	// 无 Cookie
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	if a.Authenticate(req) {
		t.Fatal("无 Cookie 应拒绝")
	}
}

func TestAuthenticateRejectsExpired(t *testing.T) {
	a := NewAdmin("pw", "fixed-secret")
	rec := httptest.NewRecorder()
	a.IssueSession(rec)

	// 时钟拨到会话过期之后
	a.now = func() time.Time { return time.Now().Add(sessionTTL + time.Hour) }
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	if a.Authenticate(req) {
		t.Fatal("过期会话应拒绝")
	}
}

func TestCheckPasswordConstantTime(t *testing.T) {
	a := NewAdmin("correct-horse", "s")
	if !a.CheckPassword("correct-horse") {
		t.Fatal("正确密码应通过")
	}
	if a.CheckPassword("wrong") {
		t.Fatal("错误密码应拒绝")
	}
	if a.CheckPassword("") {
		t.Fatal("空密码应拒绝")
	}
}

func TestRequireAuth(t *testing.T) {
	a := NewAdmin("pw", "fixed-secret")
	protected := a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 未登录：303 到登录页
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("未登录 status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/login" {
		t.Fatalf("Location = %q, want /admin/login", loc)
	}

	// 已登录：放行
	issue := httptest.NewRecorder()
	a.IssueSession(issue)
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	for _, c := range issue.Result().Cookies() {
		req.AddCookie(c)
	}
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("已登录 status = %d, want 200", rec.Code)
	}
}

func TestClearSession(t *testing.T) {
	a := NewAdmin("pw", "fixed-secret")
	rec := httptest.NewRecorder()
	a.ClearSession(rec)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("登出应下发过期 Cookie, got %+v", cookies)
	}
}

func TestNewAdminRandomSecret(t *testing.T) {
	// 空 secret 生成随机密钥：两次 NewAdmin 签发的会话互不认可
	a1 := NewAdmin("pw", "")
	a2 := NewAdmin("pw", "")
	rec := httptest.NewRecorder()
	a1.IssueSession(rec)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	if a2.Authenticate(req) {
		t.Fatal("不同随机密钥的会话不应互相认可")
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	a := NewAdmin("pw", "s")
	rec := httptest.NewRecorder()
	a.IssueSession(rec)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	c := cookies[0]
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/admin" {
		t.Fatalf("Cookie 属性错误: %+v", c)
	}
}
