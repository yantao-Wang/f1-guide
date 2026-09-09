// 后台处理器：登录/登出/仪表盘 + 渲染助手。
//
// 后台线职责划分：写操作走 service.Admin，读取复用 service.Content；
// 认证与 CSRF 基础设施在 internal/middleware，本文件只调用其导出函数。
package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/yantao-Wang/f1-guide/internal/middleware"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/view"
)

// Admin 后台处理器。
type Admin struct {
	admin     *service.Admin
	svc       *service.Content
	auth      *middleware.Admin
	uploadDir string
	log       *slog.Logger
}

// NewAdmin 组装后台处理器。
func NewAdmin(admin *service.Admin, svc *service.Content, auth *middleware.Admin, uploadDir string, log *slog.Logger) *Admin {
	return &Admin{admin: admin, svc: svc, auth: auth, uploadDir: uploadDir, log: log}
}

// RequireAuth 包装认证中间件（供 router 注册后台路由组使用）。
func (h *Admin) RequireAuth(next http.Handler) http.Handler {
	return h.auth.RequireAuth(next)
}

// adminPage 后台页面公共数据：Flash 成功提示、Error 错误提示、CSRF token、Body 页面专属数据。
type adminPage struct {
	Flash string
	Error string
	CSRF  string
	Body  any
}

// renderAdmin 渲染后台页面（注入 CSRF token）。
func (h *Admin) renderAdmin(w http.ResponseWriter, r *http.Request, name, title string, body any, flash, errMsg string) {
	renderPage(w, name, view.Page{
		Title: title,
		Data: adminPage{
			Flash: flash,
			Error: errMsg,
			CSRF:  middleware.CSRFToken(w, r),
			Body:  body,
		},
	})
}

// renderAdminStatus 带状态码渲染后台页面（校验失败 422、冲突 409 等）。
func (h *Admin) renderAdminStatus(w http.ResponseWriter, r *http.Request, status int, name, title string, body any, flash, errMsg string) {
	w.WriteHeader(status)
	h.renderAdmin(w, r, name, title, body, flash, errMsg)
}

// adminFlash 从列表页 query 参数拼成功提示（PRG 模式）。
func adminFlash(r *http.Request) string {
	q := r.URL.Query()
	switch {
	case q.Get("created") != "":
		return "已创建：" + q.Get("created")
	case q.Get("saved") != "":
		return "已保存：" + q.Get("saved")
	case q.Get("deleted") != "":
		return "已删除：" + q.Get("deleted")
	}
	return ""
}

// LoginPage 登录页。已登录直接进仪表盘。
func (h *Admin) LoginPage(w http.ResponseWriter, r *http.Request) {
	if h.auth.Authenticate(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	h.renderAdmin(w, r, "admin_login", "登录", nil, "", "")
}

// Login 登录提交：限流 → 校验密码 → 签发会话。
func (h *Admin) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单数据过大或格式错误", http.StatusRequestEntityTooLarge)
		return
	}
	if !h.auth.LimiterAllowed(r) {
		h.log.Warn("admin login rate limited", "ip", r.RemoteAddr)
		h.renderAdminStatus(w, r, http.StatusTooManyRequests, "admin_login", "登录", nil, "", "尝试过于频繁，请稍后再试")
		return
	}
	if h.auth.CheckPassword(r.PostForm.Get("password")) {
		h.log.Info("admin login success", "ip", r.RemoteAddr)
		h.auth.IssueSession(w)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	h.auth.RecordFailure(r)
	h.log.Warn("admin login failed", "ip", r.RemoteAddr)
	h.renderAdminStatus(w, r, http.StatusUnauthorized, "admin_login", "登录", nil, "", "密码错误")
}

// Logout 登出：清除会话 Cookie。
func (h *Admin) Logout(w http.ResponseWriter, r *http.Request) {
	h.auth.ClearSession(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// Dashboard 仪表盘：内容计数 + 最近更新 + 完整性检查。
func (h *Admin) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.admin.AdminStats(r.Context())
	if err != nil {
		h.log.Error("admin dashboard stats", "error", err)
		renderPageError(w, err)
		return
	}
	h.renderAdmin(w, r, "admin_dashboard", "仪表盘", stats, "", "")
}

// adminRedirect 构造带 flash 参数的后台列表页重定向（PRG 模式）。
func adminRedirect(w http.ResponseWriter, r *http.Request, kind, action, slug string) {
	http.Redirect(w, r, "/admin/"+kind+"?"+action+"="+url.QueryEscape(slug), http.StatusSeeOther)
}
