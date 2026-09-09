package middleware

import "net/http"

// LimitBody 限制请求体大小：超过上限读取即返回 413。
// 文本表单挂 1MB，含文件上传的表单挂 8MB。
func LimitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
