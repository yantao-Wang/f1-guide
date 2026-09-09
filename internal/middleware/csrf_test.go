package middleware

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCSRFTokenIssuesCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)

	token := CSRFToken(rec, req)
	if token == "" {
		t.Fatal("token 不应为空")
	}
	if len(token) != 64 {
		t.Fatalf("token 长度 = %d, want 64（32 字节 hex）", len(token))
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != csrfCookieName || cookies[0].Value != token {
		t.Fatalf("应下发同名同值 Cookie, got %+v", cookies)
	}

	// 已有 Cookie 时复用，不再生成新值
	rec2 := httptest.NewRecorder()
	req.AddCookie(cookies[0])
	if got := CSRFToken(rec2, req); got != token {
		t.Fatalf("已有 Cookie 应复用 token, got %q want %q", got, token)
	}
	if len(rec2.Result().Cookies()) != 0 {
		t.Fatal("复用 token 不应重复下发 Cookie")
	}
}

func TestRequireCSRF(t *testing.T) {
	ok := RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	// 先取一个有效 token
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	token := CSRFToken(rec, req)
	req.AddCookie(rec.Result().Cookies()[0])

	post := func() *httptest.ResponseRecorder {
		// POST 表单携带 token
		body := url.Values{csrfFieldName: {token}}.Encode()
		r := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for _, c := range req.Cookies() {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		ok.ServeHTTP(w, r)
		return w
	}

	// 正确 token 通过
	if w := post(); w.Code != http.StatusNoContent {
		t.Fatalf("正确 token status = %d, want 204", w.Code)
	}

	// 错误 token 拒绝
	bad := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader(url.Values{csrfFieldName: {"wrong"}}.Encode()))
	bad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range req.Cookies() {
		bad.AddCookie(c)
	}
	w := httptest.NewRecorder()
	ok.ServeHTTP(w, bad)
	if w.Code != http.StatusForbidden {
		t.Fatalf("错误 token status = %d, want 403", w.Code)
	}

	// 缺 token 拒绝
	missing := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader(url.Values{}.Encode()))
	missing.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range req.Cookies() {
		missing.AddCookie(c)
	}
	w = httptest.NewRecorder()
	ok.ServeHTTP(w, missing)
	if w.Code != http.StatusForbidden {
		t.Fatalf("缺 token status = %d, want 403", w.Code)
	}

	// GET 不校验
	get := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w = httptest.NewRecorder()
	ok.ServeHTTP(w, get)
	if w.Code != http.StatusNoContent {
		t.Fatalf("GET status = %d, want 204", w.Code)
	}
}

func TestRequireCSRFMultipart(t *testing.T) {
	// multipart 表单（文件上传）同样校验 CSRF——ParseForm 对 multipart 不解析 body，
	// validCSRF 必须走 ParseMultipartForm（回归测试）。
	ok := RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	token := CSRFToken(rec, req)
	req.AddCookie(rec.Result().Cookies()[0])

	post := func(token string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField(csrfFieldName, token)
		_ = w.WriteField("password", "x")
		_ = w.Close()
		r := httptest.NewRequest(http.MethodPost, "/admin", &buf)
		r.Header.Set("Content-Type", w.FormDataContentType())
		for _, c := range req.Cookies() {
			r.AddCookie(c)
		}
		rr := httptest.NewRecorder()
		ok.ServeHTTP(rr, r)
		return rr
	}

	if rr := post(token); rr.Code != http.StatusNoContent {
		t.Fatalf("multipart 正确 token status = %d, want 204", rr.Code)
	}
	if rr := post("wrong-token"); rr.Code != http.StatusForbidden {
		t.Fatalf("multipart 错误 token status = %d, want 403", rr.Code)
	}
}

func TestLimitBodyRejectsOversize(t *testing.T) {
	// 真实 handler 姿势：读取 body 出错（MaxBytesReader 超限）时返回 413
	h := LimitBody(16)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader("a-long-body-exceeding-16-bytes"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("超大请求体 status = %d, want 413", w.Code)
	}

	// 小请求体正常放行
	req = httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader("short"))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("正常请求体 status = %d, want 200", w.Code)
	}
}
