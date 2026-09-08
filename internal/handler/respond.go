package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON 序列化响应。结构体字段的 json tag 必须与 OpenAPI 契约一致。
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write json response", "error", err)
	}
}

// errorBody 对应 OpenAPI Error 组件。
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeError 输出统一错误结构（OpenAPI Error）。
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Code: code, Message: message})
}
