// 后台文件上传：图片类型/大小校验 + 随机文件名 + 本地磁盘存储。
//
// 上传文件经 /uploads/* 公开分发（router 注册 http.FileServer）。
// 扩展名由内容嗅探（DetectContentType）推导，不信任客户端文件名。
package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// maxImageBytes 单张图片上限（与路由 LimitBody(8MB) 一致）。
const maxImageBytes = 8 << 20

// mimeToExt 允许的图片类型 → 存储扩展名。
var mimeToExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// Uploads 返回上传目录的静态文件分发处理器，挂载于 /uploads/*。
func Uploads(dir string) http.Handler {
	return http.StripPrefix("/uploads/", http.FileServer(http.Dir(dir)))
}

// saveUpload 处理表单中的图片文件字段，返回站内 URL 路径（/uploads/...）。
// 未上传新文件时原样返回 current（编辑页保留旧图）；纯文本表单（非 multipart）无文件。
func (h *Admin) saveUpload(r *http.Request, field, current string) (string, error) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		return current, nil
	}
	file, header, err := r.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return current, nil
		}
		return current, err
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxImageBytes {
		return current, fmt.Errorf("图片超过 %dMB 上限", maxImageBytes>>20)
	}

	// 内容嗅探真实类型（前 512 字节足够识别图片），与客户端扩展名无关
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		return current, err
	}
	ext, ok := mimeToExt[http.DetectContentType(head[:n])]
	if !ok {
		return current, errors.New("仅支持 jpg / png / webp 图片")
	}

	// 随机文件名：避免路径注入、覆盖与可枚举性
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return current, err
	}
	name := hex.EncodeToString(buf) + ext

	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		return current, err
	}
	path := filepath.Join(h.uploadDir, name)
	dst, err := os.Create(path)
	if err != nil {
		return current, err
	}
	// 已嗅探的头部字节一并写入
	if _, err := io.Copy(dst, io.MultiReader(bytes.NewReader(head[:n]), file)); err != nil {
		_ = dst.Close()
		_ = os.Remove(path)
		return current, err
	}
	if err := dst.Close(); err != nil {
		return current, err
	}
	return "/uploads/" + name, nil
}

// removeUploadFile 删除站内上传文件（尽力而为；孤儿文件可接受）。
func (h *Admin) removeUploadFile(u string) {
	if !strings.HasPrefix(u, "/uploads/") {
		return
	}
	name := filepath.Base(strings.TrimPrefix(u, "/uploads/"))
	if name == "." || name == string(filepath.Separator) {
		return
	}
	_ = os.Remove(filepath.Join(h.uploadDir, name))
}
