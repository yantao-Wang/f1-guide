package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

// postForm 构造 urlencoded POST 请求。
func postForm(path string, values url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// serveAdmin 通过 chi 路由把请求打到后台 handler（路径参数经真实路由注入）。
func serveAdmin(t *testing.T, method, pattern, target string, req *http.Request, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	r.Handle(method+" "+pattern, h)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// validDriverValues 一份能通过校验的车手表单值。
func validDriverValues() url.Values {
	return url.Values{
		"slug": {"test-driver"}, "name": {"测试车手"}, "tagline": {"测试人设"},
		"team_name": {"测试车队"}, "team_color": {"#123456"},
		"number": {"24"}, "championships": {"2"}, "country": {"测试国"},
		"jolpica_id":  {"test_driver"},
		"story":       {"第一段\n\n第二段"},
		"personality": {"沉稳\n坚韧"},
		"trivia":      {"冷知识一"},
		"quote":       {"测试金句"},
	}
}

func TestDriverCreateSuccess(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)

	rec := httptest.NewRecorder()
	h.DriverCreate(rec, postForm("/admin/drivers/new", validDriverValues()))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/admin/drivers?created=test-driver") {
		t.Fatalf("Location = %q", loc)
	}
	if len(adminStore.CreatedDrivers) != 1 {
		t.Fatalf("CreatedDrivers = %d 条, want 1", len(adminStore.CreatedDrivers))
	}
	d := adminStore.CreatedDrivers[0]
	if d.Slug != "test-driver" || d.Number != 24 || d.Championships != 2 || d.Team.Color != "#123456" {
		t.Fatalf("基础字段错误: %+v", d)
	}
	if len(d.Personality) != 2 || d.Personality[1] != "坚韧" {
		t.Fatalf("Personality 切分错误: %v", d.Personality)
	}
	if len(d.Trivia) != 1 || d.Trivia[0] != "冷知识一" {
		t.Fatalf("Trivia 切分错误: %v", d.Trivia)
	}
	if d.JolpicaID != "test_driver" {
		t.Fatalf("JolpicaID = %q", d.JolpicaID)
	}
	if d.Featured {
		t.Fatal("未勾选 featured 应为 false")
	}
}

func TestDriverCreateValidationError(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)

	values := validDriverValues()
	values.Set("slug", "BAD Slug!") // 非法字符 + 空格
	values.Set("name", "")          // 缺必填
	rec := httptest.NewRecorder()
	h.DriverCreate(rec, postForm("/admin/drivers/new", values))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "BAD Slug!") {
		t.Fatal("422 应保留已填输入")
	}
	if !strings.Contains(body, "slug 仅允许小写字母") {
		t.Fatal("应提示 slug 格式错误")
	}
	if len(adminStore.CreatedDrivers) != 0 {
		t.Fatal("校验失败不应调用写操作")
	}
}

func TestDriverCreateConflict(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)
	adminStore.ErrConflict = true

	rec := httptest.NewRecorder()
	h.DriverCreate(rec, postForm("/admin/drivers/new", validDriverValues()))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "已被占用") {
		t.Fatal("应提示冲突文案")
	}
}

func TestDriverEdit(t *testing.T) {
	h, _, store := newTestAdmin(t)
	store.DriverDetail = testutil.SampleDriver()
	store.DriverDetail.ImageURL = "/uploads/abc.png"

	req := httptest.NewRequest(http.MethodGet, "/admin/drivers/zhou-guanyu/edit", nil)
	rec := serveAdmin(t, http.MethodGet, "/admin/drivers/{slug}/edit", "/admin/drivers/zhou-guanyu/edit", req, h.DriverEdit)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"周冠宇", "沉稳", "/uploads/abc.png", "zhou-guanyu", "/admin/drivers/zhou-guanyu/edit"} {
		if !strings.Contains(body, want) {
			t.Errorf("编辑页应包含 %q", want)
		}
	}
}

func TestDriverEditNotFound(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/drivers/no-such/edit", nil)
	rec := serveAdmin(t, http.MethodGet, "/admin/drivers/{slug}/edit", "/admin/drivers/no-such/edit", req, h.DriverEdit)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDriverUpdateSuccess(t *testing.T) {
	h, adminStore, store := newTestAdmin(t)
	store.DriverDetail = testutil.SampleDriver()

	values := validDriverValues()
	values.Set("slug", "renamed-driver")
	values.Set("featured", "on")
	req := postForm("/admin/drivers/zhou-guanyu/edit", values)
	rec := serveAdmin(t, http.MethodPost, "/admin/drivers/{slug}/edit", "/admin/drivers/zhou-guanyu/edit", req, h.DriverUpdate)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(adminStore.UpdatedDrivers) != 1 {
		t.Fatalf("UpdatedDrivers = %d 条, want 1", len(adminStore.UpdatedDrivers))
	}
	d := adminStore.UpdatedDrivers[0]
	if d.Slug != "renamed-driver" || !d.Featured {
		t.Fatalf("更新内容错误: %+v", d)
	}
}

func TestDriverDelete(t *testing.T) {
	h, adminStore, store := newTestAdmin(t)
	store.DriverDetail = testutil.SampleDriver()

	req := httptest.NewRequest(http.MethodPost, "/admin/drivers/zhou-guanyu/delete", nil)
	rec := serveAdmin(t, http.MethodPost, "/admin/drivers/{slug}/delete", "/admin/drivers/zhou-guanyu/delete", req, h.DriverDelete)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(adminStore.DeletedDrivers) != 1 || adminStore.DeletedDrivers[0] != "zhou-guanyu" {
		t.Fatalf("DeletedDrivers = %v", adminStore.DeletedDrivers)
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "deleted=zhou-guanyu") {
		t.Fatalf("Location = %q", loc)
	}
}

func TestDriverDeleteNotFound(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/drivers/no-such/delete", nil)
	rec := serveAdmin(t, http.MethodPost, "/admin/drivers/{slug}/delete", "/admin/drivers/no-such/delete", req, h.DriverDelete)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDriverList(t *testing.T) {
	h, _, store := newTestAdmin(t)
	store.Drivers = []domain.DriverSummary{
		{Slug: "zhou-guanyu", Name: "周冠宇", Team: domain.Team{Name: "凯迪拉克"}, Featured: true, ImageURL: "/uploads/a.png"},
		{Slug: "other", Name: "另一位", Team: domain.Team{Name: "X"}},
	}

	rec := httptest.NewRecorder()
	h.DriverList(rec, httptest.NewRequest(http.MethodGet, "/admin/drivers?created=zhou-guanyu", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"周冠宇", "另一位", "已创建：zhou-guanyu", "精选"} {
		if !strings.Contains(body, want) {
			t.Errorf("列表页应包含 %q", want)
		}
	}
}

func TestMomentCreateWithAssociations(t *testing.T) {
	h, adminStore, store := newTestAdmin(t)
	store.Drivers = []domain.DriverSummary{
		{Slug: "zhou-guanyu", Name: "周冠宇"},
		{Slug: "other", Name: "另一位"},
	}
	store.Tracks = []domain.TrackSummary{{Slug: "shanghai", Name: "上海"}}

	values := url.Values{
		"slug": {"new-moment"}, "title": {"新名场面"}, "year": {"2024"},
		"grand_prix": {"测试大奖赛"}, "type": {"rain"},
		"background": {"背景"}, "why_classic": {"原因"},
		"video_url":    {"https://www.bilibili.com/video/BV-x"},
		"track_slug":   {"shanghai"},
		"driver_slugs": {"zhou-guanyu", "other"},
	}
	rec := httptest.NewRecorder()
	h.MomentCreate(rec, postForm("/admin/moments/new", values))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(adminStore.CreatedMoments) != 1 {
		t.Fatalf("CreatedMoments = %d 条, want 1", len(adminStore.CreatedMoments))
	}
	m := adminStore.CreatedMoments[0]
	if m.Type != domain.MomentTypeRain || m.Year != 2024 {
		t.Fatalf("基础字段错误: %+v", m)
	}
	if len(m.RelatedDrivers) != 2 || m.RelatedDrivers[0] != "zhou-guanyu" {
		t.Fatalf("RelatedDrivers = %v", m.RelatedDrivers)
	}
	if m.RelatedTrack != "shanghai" {
		t.Fatalf("RelatedTrack = %q", m.RelatedTrack)
	}
}

func TestMomentCreateValidationError(t *testing.T) {
	h, adminStore, store := newTestAdmin(t)
	store.Drivers = []domain.DriverSummary{{Slug: "d1", Name: "D1"}}

	values := url.Values{"slug": {"m1"}, "title": {"T"}, "year": {"2024"},
		"grand_prix": {"G"}, "type": {"not-a-type"}, "background": {"B"}, "why_classic": {"W"}}
	rec := httptest.NewRecorder()
	h.MomentCreate(rec, postForm("/admin/moments/new", values))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "名场面类型非法") {
		t.Fatal("应提示类型枚举错误")
	}
	if len(adminStore.CreatedMoments) != 0 {
		t.Fatal("校验失败不应调用写操作")
	}
}

func TestMomentUpdateNotFound(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)
	adminStore.ErrNotFound = true

	req := postForm("/admin/moments/no-such/edit", url.Values{
		"slug": {"m1"}, "title": {"T"}, "year": {"2024"}, "grand_prix": {"G"},
		"type": {"rain"}, "background": {"B"}, "why_classic": {"W"},
	})
	rec := serveAdmin(t, http.MethodPost, "/admin/moments/{slug}/edit", "/admin/moments/no-such/edit", req, h.MomentUpdate)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestTrackCreateSuccess(t *testing.T) {
	h, adminStore, _ := newTestAdmin(t)

	values := url.Values{
		"slug": {"monza"}, "name": {"蒙扎"}, "country": {"意大利"}, "tagline": {"速度圣殿"},
		"type": {"permanent"}, "first_grand_prix": {"1950"}, "length_km": {"5.793"},
		"laps": {"53"}, "highlights": {"长直道\n帕拉波利卡"},
	}
	rec := httptest.NewRecorder()
	h.TrackCreate(rec, postForm("/admin/tracks/new", values))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(adminStore.CreatedTracks) != 1 {
		t.Fatalf("CreatedTracks = %d 条, want 1", len(adminStore.CreatedTracks))
	}
	tr := adminStore.CreatedTracks[0]
	if tr.Slug != "monza" || tr.LengthKm != 5.793 || len(tr.Highlights) != 2 {
		t.Fatalf("赛道字段错误: %+v", tr)
	}
}

func TestTrackUpdateValidationError(t *testing.T) {
	h, adminStore, store := newTestAdmin(t)
	store.TrackDetail = testutil.SampleTrack() // slug = shanghai

	values := url.Values{
		"slug": {"shanghai"}, "name": {"蒙扎"}, "country": {"意大利"}, "tagline": {"速度圣殿"},
		"type": {"permanent"}, "first_grand_prix": {"1950"}, "length_km": {"5.793"},
		"laps": {"999"}, // 圈数越界
	}
	req := postForm("/admin/tracks/shanghai/edit", values)
	rec := serveAdmin(t, http.MethodPost, "/admin/tracks/{slug}/edit", "/admin/tracks/shanghai/edit", req, h.TrackUpdate)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "圈数") {
		t.Fatal("应提示圈数校验错误")
	}
	if len(adminStore.UpdatedTracks) != 0 {
		t.Fatal("校验失败不应调用写操作")
	}
}

func TestTrackDeleteNotFound(t *testing.T) {
	h, _, _ := newTestAdmin(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/tracks/no-such/delete", nil)
	rec := serveAdmin(t, http.MethodPost, "/admin/tracks/{slug}/delete", "/admin/tracks/no-such/delete", req, h.TrackDelete)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
