package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const (
	// csrfCookieName CSRF token Cookie 名。
	csrfCookieName = "f1admin_csrf"
	// csrfFieldName 表单中的 CSRF token 字段名。
	csrfFieldName = "csrf"
	// defaultMaxMemory CSRF 校验时 multipart 解析的内存上限（文件总量由路由 LimitBody 拦截）。
	defaultMaxMemory = 32 << 20
)

// CSRF 防护采用双重提交模式：Cookie 与表单 hidden 字段一致即通过。无状态，适配单实例。
// 同一 token 跨请求复用，不逐请求轮换。

// CSRFToken 返回当前请求的 CSRF token；不存在则生成并下发 Cookie。
// 由渲染表单的 handler 调用，把返回值注入模板 hidden 字段。
func CSRFToken(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(csrfCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	token := newCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     adminPath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return token
}

// RequireCSRF 校验不安全方法的 CSRF token（表单字段与 Cookie 一致才放行）。
func RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			if !validCSRF(r) {
				http.Error(w, "CSRF 校验失败", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// validCSRF 双重提交校验：表单字段与 Cookie 值常量时间比较。
func validCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	// PostForm 是惰性填充的：urlencoded 走 ParseForm；
	// multipart 必须走 ParseMultipartForm（ParseForm 对 multipart 不解析 body）。
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
			return false
		}
	} else if err := r.ParseForm(); err != nil {
		return false
	}
	submitted := r.PostForm.Get(csrfFieldName)
	return hmac.Equal([]byte(submitted), []byte(cookie.Value))
}

// newCSRFToken 生成 32 字节随机 token（hex 编码）。
func newCSRFToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("middleware: generate csrf token: " + err.Error())
	}
	return hex.EncodeToString(buf)
}
