package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

// pngBytes 最小可识别的 PNG 内容（DetectContentType 只需文件头）。
func pngBytes() []byte {
	return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
}

// multipartRequest 构造携带车手表单 + 图片文件的 multipart 请求。
func multipartRequest(t *testing.T, values url.Values, filename string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for key, vals := range values {
		for _, v := range vals {
			if err := w.WriteField(key, v); err != nil {
				t.Fatalf("WriteField: %v", err)
			}
		}
	}
	part, err := w.CreateFormFile("image", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/drivers/new", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestUploadImageSuccess(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)

	rec := httptest.NewRecorder()
	h.DriverCreate(rec, multipartRequest(t, validDriverValues(), "photo.png", pngBytes()))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(adminStore.CreatedDrivers) != 1 {
		t.Fatalf("CreatedDrivers = %d 条, want 1", len(adminStore.CreatedDrivers))
	}
	imageURL := adminStore.CreatedDrivers[0].ImageURL
	if !strings.HasPrefix(imageURL, "/uploads/") || !strings.HasSuffix(imageURL, ".png") {
		t.Fatalf("ImageURL = %q, want /uploads/<随机名>.png", imageURL)
	}

	// 文件应落盘到上传目录
	path := h.uploadDir + strings.TrimPrefix(imageURL, "/uploads")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("上传文件未落盘: %v", err)
	}
}

func TestUploadRejectsNonImage(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)

	rec := httptest.NewRecorder()
	h.DriverCreate(rec, multipartRequest(t, validDriverValues(), "malware.exe", []byte("not an image")))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "仅支持 jpg / png / webp") {
		t.Fatal("应提示图片类型错误")
	}
	if len(adminStore.CreatedDrivers) != 0 {
		t.Fatal("上传校验失败不应调用写操作")
	}
}

func TestUploadOversizeRejected(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)

	// 超过 8MB 的"图片"：saveUpload 大小校验拒绝（422 重渲染表单）
	big := make([]byte, maxImageBytes+1024)
	copy(big, pngBytes())
	rec := httptest.NewRecorder()
	h.DriverCreate(rec, multipartRequest(t, validDriverValues(), "big.png", big))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "超过 8MB 上限") {
		t.Fatal("应提示大小超限")
	}
	if len(adminStore.CreatedDrivers) != 0 {
		t.Fatal("超大上传不应调用写操作")
	}
}

func TestUploadKeepsExistingOnMissingFile(t *testing.T) {
	h, adminStore, store := newTestAdmin(t)
	existing := testutil.SampleDriver()
	existing.ImageURL = "/uploads/keep.png"
	store.DriverDetail = existing

	// 编辑时不带新文件：ImageURL 应保持原值
	values := validDriverValues()
	values.Set("slug", "zhou-guanyu")
	req := postForm("/admin/drivers/zhou-guanyu/edit", values)
	rec := serveAdmin(t, http.MethodPost, "/admin/drivers/{slug}/edit", "/admin/drivers/zhou-guanyu/edit", req, h.DriverUpdate)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(adminStore.UpdatedDrivers) != 1 {
		t.Fatalf("UpdatedDrivers = %d 条, want 1", len(adminStore.UpdatedDrivers))
	}
	if got := adminStore.UpdatedDrivers[0].ImageURL; got != "/uploads/keep.png" {
		t.Fatalf("ImageURL = %q, want 保持原值 /uploads/keep.png", got)
	}
}

func TestRemoveImageFlag(t *testing.T) {
	h, adminStore, store := newTestAdmin(t)
	existing := testutil.SampleDriver()
	existing.ImageURL = "/uploads/keep.png"
	store.DriverDetail = existing

	values := validDriverValues()
	values.Set("slug", "zhou-guanyu")
	values.Set("remove_image", "on")
	req := postForm("/admin/drivers/zhou-guanyu/edit", values)
	rec := serveAdmin(t, http.MethodPost, "/admin/drivers/{slug}/edit", "/admin/drivers/zhou-guanyu/edit", req, h.DriverUpdate)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := adminStore.UpdatedDrivers[0].ImageURL; got != "" {
		t.Fatalf("ImageURL = %q, want 空（已清除）", got)
	}
}
