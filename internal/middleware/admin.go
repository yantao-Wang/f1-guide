// Package middleware 提供 HTTP 中间件：后台会话认证、CSRF 防护、登录限流、请求体上限。
package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// sessionCookieName 后台会话 Cookie 名。
	sessionCookieName = "f1admin"
	// sessionTTL 会话有效期。
	sessionTTL = 7 * 24 * time.Hour
	// adminPath 后台路由前缀，Cookie Path 收窄到该前缀。
	adminPath = "/admin"
)

// Admin 后台会话认证器：HMAC-SHA256 签名 Cookie（无状态会话）+ 登录限流。
//
// 密码以环境变量注入（单管理员折中），这里仅做 SHA-256 摘要以实现常量时间比较——
// 摘要不构成密码哈希存储，暴力破解的真正防线是登录限流。
type Admin struct {
	secret   []byte   // Cookie 签名密钥
	password [32]byte // 配置密码的 SHA-256 摘要
	limiter  *LoginLimiter
	now      func() time.Time // 注入时钟（测试用）
}

// NewAdmin 创建认证器。secret 为空时生成随机密钥（仅本次进程有效，重启需重新登录）。
func NewAdmin(password, secret string) *Admin {
	key := []byte(secret)
	if secret == "" {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			// crypto/rand 失败意味着系统熵源损坏，此时无法安全签发会话，直接 panic。
			panic("middleware: generate session secret: " + err.Error())
		}
	}
	return &Admin{
		secret:   key,
		password: sha256.Sum256([]byte(password)),
		limiter:  NewLoginLimiter(5, 15*time.Minute),
		now:      time.Now,
	}
}

// CheckPassword 常量时间比较登录密码。
func (a *Admin) CheckPassword(attempt string) bool {
	digest := sha256.Sum256([]byte(attempt))
	return hmac.Equal(digest[:], a.password[:])
}

// LimiterAllowed 判断当前客户端是否仍在登录限流窗口内。
func (a *Admin) LimiterAllowed(r *http.Request) bool {
	return a.limiter.Allowed(r.RemoteAddr)
}

// RecordFailure 记录一次登录失败（限流计数）。
func (a *Admin) RecordFailure(r *http.Request) {
	a.limiter.RecordFailure(r.RemoteAddr)
}

// IssueSession 登录成功后签发会话 Cookie。
func (a *Admin) IssueSession(w http.ResponseWriter) {
	expires := a.now().Add(sessionTTL)
	payload := strconv.FormatInt(expires.Unix(), 10)
	value := payload + "." + a.sign(payload)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Expires:  expires,
		Path:     adminPath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSession 登出：清除会话 Cookie。
func (a *Admin) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     adminPath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Authenticate 校验请求携带的会话 Cookie（签名 + 未过期）。
func (a *Admin) Authenticate(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	payload, sig, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		return false
	}
	expected := a.sign(payload)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return false
	}
	expires, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return false
	}
	return time.Unix(expires, 0).After(a.now())
}

// sign 计算 payload 的 HMAC-SHA256 十六进制签名。
func (a *Admin) sign(payload string) string {
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// RequireAuth 保护后台路由：未登录 303 重定向到登录页（GET/POST 一致，单管理员可接受）。
func (a *Admin) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Authenticate(r) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
