package view

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/yantao-Wang/f1-guide/internal/domain"
)

// renderAll 用最小合法数据渲染每个页面模板，验证模板语法与字段引用。
func TestRenderAllTemplates(t *testing.T) {
	d := &domain.Driver{
		Slug: "zhou-guanyu", Name: "周冠宇", Tagline: "示例人设",
		Team:        domain.Team{Name: "凯迪拉克", Color: "#00A1E0"},
		Personality: []string{"沉稳"}, Trivia: []string{"趣事"}, Quote: "金句",
		FeaturedRace: &domain.FeaturedRace{Year: 2022, GrandPrix: "巴林大奖赛"},
	}
	tr := &domain.Track{
		TrackSummary: domain.TrackSummary{Slug: "shanghai", Name: "上海", Type: "permanent"},
		Highlights:   []string{"一号弯"},
	}
	m := &domain.Moment{
		MomentSummary: domain.MomentSummary{Slug: "m1", Title: "标题", Type: "championship"},
	}

	cases := []struct {
		name string
		page Page
	}{
		{"home", Page{Data: map[string]any{
			"DriverCount": 1, "TrackCount": 1, "MomentCount": 1,
			"Featured": []domain.DriverSummary{{Slug: "a", Name: "A", Team: d.Team}},
		}}},
		{"drivers", Page{Data: map[string]any{
			"Drivers": []domain.DriverSummary{{Slug: "a", Name: "A", Team: d.Team}},
		}}},
		{"driver_detail", Page{Data: map[string]any{"Driver": d}}},
		{"tracks", Page{Data: map[string]any{
			"Tracks": []domain.TrackSummary{tr.TrackSummary},
		}}},
		{"track_detail", Page{Data: map[string]any{"Track": tr}}},
		{"moments", Page{Data: map[string]any{
			"Moments": []struct {
				domain.MomentSummary
				TypeLabel string
			}{{m.MomentSummary, "冠军争夺"}},
		}}},
		{"moment_detail", Page{Data: map[string]any{
			"Moment": m, "TypeLabel": "冠军争夺",
		}}},
		{"about", Page{}},
		{"stub", Page{Data: map[string]any{"Title": "赛程", "Hint": "敬请期待"}}},
		{"error_404", Page{}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.page.Title = "测试"
			var buf bytes.Buffer
			if err := Render(&buf, c.name, c.page); err != nil {
				t.Fatalf("Render(%s): %v", c.name, err)
			}
			if !strings.Contains(buf.String(), "走近围场") {
				t.Fatalf("Render(%s) 未包含公共布局", c.name)
			}
		})
	}
}

func TestParagraphs(t *testing.T) {
	got := paragraphs("第一段\n\n\n第二段\n\n  \n\n第三段")
	want := []string{"第一段", "第二段", "第三段"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paragraphs = %#v, want %#v", got, want)
	}
}

func TestStaticHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	Static().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Fatalf("Content-Type = %q, want text/css", ct)
	}
	if !strings.Contains(rec.Body.String(), "--brand") {
		t.Fatal("style.css 缺少品牌色令牌")
	}
}
