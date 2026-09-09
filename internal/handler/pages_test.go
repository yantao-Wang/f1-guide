package handler

import (
	"errors"
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
	return NewPages(service.NewContent(store, store, store), service.NewStats(&testutil.FakeStatsClient{}, store))
}

func newTestPagesWithStats(store *testutil.FakeStore, stats *testutil.FakeStatsClient) *Pages {
	return NewPages(service.NewContent(store, store, store), service.NewStats(stats, store))
}

func fullStore() *testutil.FakeStore {
	d := testutil.SampleDriver()
	return &testutil.FakeStore{
		Drivers: []domain.DriverSummary{
			{Slug: d.Slug, Name: d.Name, Tagline: d.Tagline, Team: d.Team, Featured: true},
			{Slug: "max-verstappen", Name: "马克斯·维斯塔潘", Tagline: "示例人设", Team: domain.Team{Name: "红牛", Color: "#1E41FF"}},
		},
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
		"积分榜速览",        // 积分榜模块
		"开始认识车手",       // CTA
		"维斯塔潘",         // 积分榜数据行
		"红牛",           // 积分榜车队列
	} {
		if !strings.Contains(body, want) {
			t.Errorf("首页缺少内容 %q", want)
		}
	}
}

func TestHomePageStandingsFallback(t *testing.T) {
	pages := newTestPagesWithStats(fullStore(), &testutil.FakeStatsClient{Err: errors.New("upstream down")})

	rec := httptest.NewRecorder()
	pages.Home(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "积分榜数据暂不可用") {
		t.Fatalf("上游故障时缺少降级占位: %s", body[:200])
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
	pages.StubPage("方格旗预言", "敬请期待")(rec, httptest.NewRequest(http.MethodGet, "/game", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "方格旗预言") || !strings.Contains(rec.Body.String(), "敬请期待") {
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

func TestSchedulePage(t *testing.T) {
	pages := newTestPages(&testutil.FakeStore{})

	rec := httptest.NewRecorder()
	pages.Schedule(rec, httptest.NewRequest(http.MethodGet, "/schedule", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"澳大利亚大奖赛", // 已完赛（中文映射）
		"🏆 维斯塔潘",  // 冠军车手
		"中国大奖赛",   // 已取消（日期已过无赛果）
		"日本大奖赛",   // 未开赛
		"已完赛", "未开赛", "已取消",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("赛程页缺少内容 %q", want)
		}
	}
}

func TestSchedulePageFailed(t *testing.T) {
	pages := newTestPagesWithStats(&testutil.FakeStore{}, &testutil.FakeStatsClient{Err: errors.New("upstream down")})

	rec := httptest.NewRecorder()
	pages.Schedule(rec, httptest.NewRequest(http.MethodGet, "/schedule", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d（数据不可用时页面仍可用）", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "赛事数据源暂不可用") {
		t.Fatalf("赛程页缺少降级提示: %s", rec.Body.String()[:200])
	}
}

func TestDataPage(t *testing.T) {
	pages := newTestPages(fullStore())

	rec := httptest.NewRecorder()
	pages.Data(rec, httptest.NewRequest(http.MethodGet, "/data", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"车手积分榜", "车队积分榜",
		"维斯塔潘", "红牛",
		"梅赛德斯", "凯迪拉克",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("数据页缺少内容 %q", want)
		}
	}
	// 站内有故事的 slug 挂链接，未知车手不挂
	if !strings.Contains(body, `/stories/drivers/max-verstappen`) {
		t.Error("站内车手应挂详情链接")
	}
}

func TestDataPageFailed(t *testing.T) {
	pages := newTestPagesWithStats(&testutil.FakeStore{}, &testutil.FakeStatsClient{Err: errors.New("upstream down")})

	rec := httptest.NewRecorder()
	pages.Data(rec, httptest.NewRequest(http.MethodGet, "/data", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d（数据不可用时页面仍可用）", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "赛事数据源暂不可用") {
		t.Fatalf("数据页缺少降级提示: %s", rec.Body.String()[:200])
	}
}
