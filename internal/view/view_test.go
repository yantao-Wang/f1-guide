package view

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

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
		{"admin_login", Page{Data: map[string]any{"CSRF": "token", "Error": "密码错误"}}},
		{"admin_dashboard", Page{Data: map[string]any{
			"CSRF": "token",
			"Body": struct {
				DriverCount, TrackCount, MomentCount, FeaturedCount int
				Recent                                              []struct {
					Kind, Slug, Name string
					UpdatedAt        time.Time
				}
				Issues []struct {
					Kind, Slug, Name, Issue, Severity string
				}
			}{
				DriverCount: 1, TrackCount: 1, MomentCount: 1, FeaturedCount: 0,
				Recent: []struct {
					Kind, Slug, Name string
					UpdatedAt        time.Time
				}{{Kind: "driver", Slug: "a", Name: "A"}},
				Issues: []struct {
					Kind, Slug, Name, Issue, Severity string
				}{{Kind: "track", Slug: "b", Name: "B", Issue: "缺少赛道图", Severity: "建议"}},
			},
		}}},
		{"admin_drivers", Page{Data: map[string]any{
			"CSRF":  "token",
			"Flash": "已创建：a",
			"Body":  map[string]any{"Drivers": []domain.DriverSummary{{Slug: "a", Name: "A", Team: d.Team, Featured: true, ImageURL: "/uploads/x.png"}}},
		}}},
		{"admin_driver_form", Page{Data: map[string]any{
			"CSRF": "token",
			"Body": map[string]any{
				"Form": map[string]any{
					"Slug": "a", "Name": "A", "Errors": map[string]string{"slug": "已被占用"},
				},
				"IsEdit":       true,
				"OldSlug":      "a",
				"CurrentImage": "/uploads/x.png",
			},
		}}},
		{"admin_tracks", Page{Data: map[string]any{
			"CSRF": "token",
			"Body": map[string]any{"Tracks": []domain.TrackSummary{{Slug: "a", Name: "A", CircuitMapURL: "/uploads/t.png"}}},
		}}},
		{"admin_track_form", Page{Data: map[string]any{
			"CSRF": "token",
			"Body": map[string]any{
				"Form":         map[string]any{"Slug": "a", "Errors": map[string]string{}},
				"IsEdit":       false,
				"CurrentImage": "",
			},
		}}},
		{"admin_moments", Page{Data: map[string]any{
			"CSRF": "token",
			"Body": map[string]any{
				"Moments": []struct {
					domain.MomentSummary
					TypeLabel string
				}{{m.MomentSummary, "雨战传奇"}},
			},
		}}},
		{"admin_moment_form", Page{Data: map[string]any{
			"CSRF": "token",
			"Body": map[string]any{
				"Form": map[string]any{
					"Slug": "a", "Errors": map[string]string{},
					"DriverSlugs": []string{"a"},
				},
				"IsEdit":  true,
				"OldSlug": "a",
				"Tracks":  []domain.TrackSummary{{Slug: "sh", Name: "上海"}},
				"Drivers": []domain.DriverSummary{{Slug: "a", Name: "A"}},
			},
		}}},
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

func TestDriverHeroImageStates(t *testing.T) {
	// 有照片：显示 <img>；无照片：显示降级占位
	render := func(d *domain.Driver) string {
		var buf bytes.Buffer
		if err := Render(&buf, "driver_detail", Page{Title: "测试", Data: map[string]any{"Driver": d}}); err != nil {
			t.Fatalf("Render: %v", err)
		}
		return buf.String()
	}

	with := render(&domain.Driver{
		Slug: "a", Name: "A", Team: domain.Team{Name: "T", Color: "#123456"},
		ImageURL: "/uploads/photo.png",
	})
	if !strings.Contains(with, `<img class="driver-hero-photo" src="/uploads/photo.png"`) {
		t.Error("有照片时应渲染 <img> 卡片")
	}

	without := render(&domain.Driver{
		Slug: "a", Name: "A", Team: domain.Team{Name: "T", Color: "#123456"},
	})
	if !strings.Contains(without, "driver-hero-photo-fallback") {
		t.Error("无照片时应渲染降级占位")
	}
	if strings.Contains(without, "<img") {
		t.Error("无照片时不应渲染 <img>")
	}
}

func TestAdminTemplateFuncs(t *testing.T) {
	if got := adminEditURL("driver", "zhou-guanyu"); got != "/admin/drivers/zhou-guanyu/edit" {
		t.Errorf("adminEditURL = %q", got)
	}
	if got := adminEditURL("unknown", "x"); got != "" {
		t.Errorf("adminEditURL(unknown) = %q, want 空", got)
	}
	if got := adminFrontURL("moment", "m1"); got != "/stories/moments/m1" {
		t.Errorf("adminFrontURL = %q", got)
	}
	if !containsString([]string{"a", "b"}, "b") || containsString(nil, "a") {
		t.Error("containsString 行为错误")
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
