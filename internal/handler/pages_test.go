package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

func newTestPages(store *testutil.FakeStore) *Pages {
	return NewPages(service.NewContent(store, store, store))
}

func fullStore() *testutil.FakeStore {
	d := testutil.SampleDriver()
	return &testutil.FakeStore{
		Drivers:      []domain.DriverSummary{{Slug: d.Slug, Name: d.Name, Tagline: d.Tagline, Team: d.Team, Featured: true}},
		DriverDetail: d,
		Tracks:       []domain.TrackSummary{testutil.SampleTrack().TrackSummary},
		TrackDetail:  testutil.SampleTrack(),
		Moments:      []domain.MomentSummary{testutil.SampleMoment().MomentSummary},
		MomentDetail: testutil.SampleMoment(),
	}
}

func TestHomePage(t *testing.T) {
	pages := newTestPages(fullStore())

	rec := httptest.NewRecorder()
	pages.Home(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"从零开始，看懂F1的精彩", // Hero 主标题
		"围场人物志精选",      // 精选模块
		"周冠宇",          // 精选车手卡片
		"积分榜速览",        // 积分榜占位
		"开始认识车手",       // CTA
	} {
		if !strings.Contains(body, want) {
			t.Errorf("首页缺少内容 %q", want)
		}
	}
}

func TestDriversPage(t *testing.T) {
	pages := newTestPages(fullStore())

	rec := httptest.NewRecorder()
	pages.Drivers(rec, httptest.NewRequest(http.MethodGet, "/stories/drivers", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "车手故事") || !strings.Contains(body, "周冠宇") {
		t.Fatalf("车手列表页内容不完整: %s", body[:200])
	}
}

func TestDriverDetailPage(t *testing.T) {
	pages := newTestPages(fullStore())

	r := chi.NewRouter()
	r.Get("/stories/drivers/{slug}", pages.Driver)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stories/drivers/zhou-guanyu", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{"周冠宇", "凯迪拉克", "性格标签", "名言金句", "2021-abu-dhabi"} {
		if !strings.Contains(body, want) {
			t.Errorf("车手详情页缺少内容 %q", want)
		}
	}
}

func TestDriverDetailNotFound(t *testing.T) {
	pages := newTestPages(&testutil.FakeStore{})

	r := chi.NewRouter()
	r.Get("/stories/drivers/{slug}", pages.Driver)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stories/drivers/no-such", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if !strings.Contains(rec.Body.String(), "驶出了赛道") {
		t.Fatalf("404 页面内容缺失: %s", rec.Body.String()[:200])
	}
}

func TestTracksAndMomentsPages(t *testing.T) {
	pages := newTestPages(fullStore())

	paths := []struct {
		path string
		h    http.HandlerFunc
		want string
	}{
		{"/stories/tracks", pages.Tracks, "上海国际赛车场"},
		{"/stories/moments", pages.Moments, "阿布扎比"},
	}
	for _, c := range paths {
		rec := httptest.NewRecorder()
		c.h(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", c.path, rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), c.want) {
			t.Errorf("GET %s 缺少内容 %q", c.path, c.want)
		}
	}
}

func TestMomentDetailTypeLabel(t *testing.T) {
	pages := newTestPages(fullStore())

	r := chi.NewRouter()
	r.Get("/stories/moments/{slug}", pages.Moment)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stories/moments/2021-abu-dhabi", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "冠军争夺") {
		t.Fatalf("类型标签缺失: %s", rec.Body.String()[:200])
	}
}

func TestStubPage(t *testing.T) {
	pages := newTestPages(&testutil.FakeStore{})

	rec := httptest.NewRecorder()
	pages.StubPage("格子棋", "敬请期待")(rec, httptest.NewRequest(http.MethodGet, "/game", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "格子棋") || !strings.Contains(rec.Body.String(), "敬请期待") {
		t.Fatalf("占位页内容缺失: %s", rec.Body.String()[:200])
	}
}

func TestAboutPage(t *testing.T) {
	pages := newTestPages(&testutil.FakeStore{})

	rec := httptest.NewRecorder()
	pages.About(rec, httptest.NewRequest(http.MethodGet, "/about", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "免责声明") {
		t.Fatalf("关于页缺少免责声明")
	}
}
