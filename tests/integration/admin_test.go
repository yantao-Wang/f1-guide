//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/repository"
)

// sampleDriver 返回一个字段齐全的写操作用车手。
func sampleDriver(slug string) *domain.Driver {
	return &domain.Driver{
		Slug:          slug,
		Name:          "测试车手",
		Tagline:       "测试人设",
		Team:          domain.Team{Name: "测试车队", Color: "#123456"},
		Number:        99,
		Championships: 2,
		Country:       "测试国",
		Story:         "第一段\n\n第二段",
		Personality:   []string{"沉稳", "坚韧"},
		Trivia:        []string{"冷知识一"},
		Quote:         "测试金句",
		Featured:      true,
		FeaturedRace:  &domain.FeaturedRace{Year: 2020, GrandPrix: "测试大奖赛"},
		JolpicaID:     "test_driver",
		ImageURL:      "/uploads/abc.png",
	}
}

func TestCreateDriverRoundTrip(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	d := sampleDriver("test-driver")
	if err := repo.CreateDriver(ctx, d); err != nil {
		t.Fatalf("CreateDriver: %v", err)
	}

	got, err := repo.GetDriver(ctx, "test-driver")
	if err != nil {
		t.Fatalf("GetDriver: %v", err)
	}
	if got.Number != 99 || got.Story != "第一段\n\n第二段" {
		t.Fatalf("基础字段错误: %+v", got)
	}
	if len(got.Personality) != 2 || got.Personality[1] != "坚韧" {
		t.Fatalf("Personality = %v", got.Personality)
	}
	if got.FeaturedRace == nil || got.FeaturedRace.GrandPrix != "测试大奖赛" {
		t.Fatalf("FeaturedRace = %+v", got.FeaturedRace)
	}
	if got.JolpicaID != "test_driver" {
		t.Fatalf("JolpicaID = %q, want test_driver", got.JolpicaID)
	}
	if got.ImageURL != "/uploads/abc.png" {
		t.Fatalf("ImageURL = %q, want /uploads/abc.png", got.ImageURL)
	}
}

func TestCreateDriverConflict(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	if err := repo.CreateDriver(ctx, sampleDriver("dup-slug")); err != nil {
		t.Fatalf("首次创建: %v", err)
	}
	err := repo.CreateDriver(ctx, sampleDriver("dup-slug"))
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}

	// jolpica_id 冲突同样报 ErrConflict
	err = repo.CreateDriver(ctx, sampleDriver("other-slug"))
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("jolpica 冲突 err = %v, want ErrConflict", err)
	}
}

func TestCreateDriverOptionalFieldsNull(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	d := sampleDriver("minimal-driver")
	d.JolpicaID = ""
	d.ImageURL = ""
	d.FeaturedRace = nil
	d.Featured = false
	if err := repo.CreateDriver(ctx, d); err != nil {
		t.Fatalf("CreateDriver: %v", err)
	}

	got, err := repo.GetDriver(ctx, "minimal-driver")
	if err != nil {
		t.Fatalf("GetDriver: %v", err)
	}
	if got.JolpicaID != "" || got.ImageURL != "" || got.FeaturedRace != nil {
		t.Fatalf("可选字段应为空: JolpicaID=%q ImageURL=%q FeaturedRace=%+v", got.JolpicaID, got.ImageURL, got.FeaturedRace)
	}
}

func TestUpdateDriver(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	if err := repo.CreateDriver(ctx, sampleDriver("old-slug")); err != nil {
		t.Fatalf("CreateDriver: %v", err)
	}

	d := sampleDriver("new-slug")
	d.Name = "改名后的车手"
	d.JolpicaID = "renamed_driver"
	if err := repo.UpdateDriver(ctx, "old-slug", d); err != nil {
		t.Fatalf("UpdateDriver: %v", err)
	}

	// 新 slug 可读、旧 slug 404
	got, err := repo.GetDriver(ctx, "new-slug")
	if err != nil {
		t.Fatalf("GetDriver(new): %v", err)
	}
	if got.Name != "改名后的车手" || got.JolpicaID != "renamed_driver" {
		t.Fatalf("更新未生效: %+v", got)
	}
	if _, err := repo.GetDriver(ctx, "old-slug"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("旧 slug err = %v, want ErrNotFound", err)
	}

	// 更新不存在的车手
	if err := repo.UpdateDriver(ctx, "no-such", d); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDeleteDriverCascadeMomentDrivers(t *testing.T) {
	repo, pg := setup(t)
	seedContent(t, pg)
	ctx := context.Background()

	if err := repo.DeleteDriver(ctx, "zhou-guanyu"); err != nil {
		t.Fatalf("DeleteDriver: %v", err)
	}

	// 名场面仍在，但关联被级联清理
	m, err := repo.GetMoment(ctx, "2021-abu-dhabi")
	if err != nil {
		t.Fatalf("GetMoment: %v", err)
	}
	if len(m.RelatedDrivers) != 0 {
		t.Fatalf("RelatedDrivers = %v, want 空（级联清理）", m.RelatedDrivers)
	}

	if err := repo.DeleteDriver(ctx, "no-such"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestTrackCRUDAndSetNull(t *testing.T) {
	repo, pg := setup(t)
	seedContent(t, pg)
	ctx := context.Background()

	// 创建
	newTrack := &domain.Track{
		TrackSummary: domain.TrackSummary{
			Slug: "monza", Name: "蒙扎", Country: "意大利", Tagline: "速度圣殿", Type: domain.TrackTypePermanent,
		},
		FirstGrandPrix: 1950, LengthKm: 5.793, Laps: 53,
		Highlights:    []string{"长直道"},
		CircuitMapURL: "/uploads/monza.png",
	}
	if err := repo.CreateTrack(ctx, newTrack); err != nil {
		t.Fatalf("CreateTrack: %v", err)
	}
	got, err := repo.GetTrack(ctx, "monza")
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if got.CircuitMapURL != "/uploads/monza.png" || got.Laps != 53 {
		t.Fatalf("回读错误: %+v", got)
	}

	// 更新（改名）
	newTrack.Name = "蒙扎国家赛车场"
	if err := repo.UpdateTrack(ctx, "monza", newTrack); err != nil {
		t.Fatalf("UpdateTrack: %v", err)
	}

	// 删除已关联赛道：名场面 track 自动解绑
	if err := repo.DeleteTrack(ctx, "shanghai"); err != nil {
		t.Fatalf("DeleteTrack: %v", err)
	}
	m, err := repo.GetMoment(ctx, "2021-abu-dhabi")
	if err != nil {
		t.Fatalf("GetMoment: %v", err)
	}
	if m.RelatedTrack != "" {
		t.Fatalf("RelatedTrack = %q, want 空（SET NULL）", m.RelatedTrack)
	}
}

func TestMomentCRUDWithRelations(t *testing.T) {
	repo, pg := setup(t)
	seedContent(t, pg)
	ctx := context.Background()

	// 创建：关联赛道 + 两个车手
	nm := &domain.Moment{
		MomentSummary: domain.MomentSummary{
			Slug: "new-moment", Title: "新名场面", Year: 2024, GrandPrix: "测试大奖赛", Type: domain.MomentTypeRain,
		},
		Background:     "第一段\n\n第二段",
		WhyClassic:     "因为它经典",
		VideoURL:       "https://www.bilibili.com/video/BV-new",
		RelatedDrivers: []string{"zhou-guanyu", "other-driver"},
		RelatedTrack:   "shanghai",
	}
	if err := repo.CreateMoment(ctx, nm); err != nil {
		t.Fatalf("CreateMoment: %v", err)
	}
	got, err := repo.GetMoment(ctx, "new-moment")
	if err != nil {
		t.Fatalf("GetMoment: %v", err)
	}
	if len(got.RelatedDrivers) != 2 || got.RelatedTrack != "shanghai" || got.VideoURL != "https://www.bilibili.com/video/BV-new" {
		t.Fatalf("关联回读错误: %+v", got)
	}

	// 更新：换赛道、减车手、改 slug
	nm.Slug = "renamed-moment"
	nm.RelatedDrivers = []string{"other-driver"}
	nm.RelatedTrack = ""
	if err := repo.UpdateMoment(ctx, "new-moment", nm); err != nil {
		t.Fatalf("UpdateMoment: %v", err)
	}
	got, err = repo.GetMoment(ctx, "renamed-moment")
	if err != nil {
		t.Fatalf("GetMoment(renamed): %v", err)
	}
	if len(got.RelatedDrivers) != 1 || got.RelatedDrivers[0] != "other-driver" || got.RelatedTrack != "" {
		t.Fatalf("重建关联错误: %+v", got)
	}

	// 删除
	if err := repo.DeleteMoment(ctx, "renamed-moment"); err != nil {
		t.Fatalf("DeleteMoment: %v", err)
	}
	if _, err := repo.GetMoment(ctx, "renamed-moment"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAdminStats(t *testing.T) {
	repo, pg := setup(t)
	seedContent(t, pg)
	ctx := context.Background()

	stats, err := repo.AdminStats(ctx)
	if err != nil {
		t.Fatalf("AdminStats: %v", err)
	}
	if stats.DriverCount != 2 || stats.TrackCount != 1 || stats.MomentCount != 1 || stats.FeaturedCount != 1 {
		t.Fatalf("计数错误: %+v", stats)
	}

	// 最近更新：seed 后应有 4 条（2 车手 + 1 赛道 + 1 名场面）
	if len(stats.Recent) != 4 {
		t.Fatalf("Recent = %d 条, want 4", len(stats.Recent))
	}

	// 完整性检查：seed 数据未填 jolpica_id 且无照片，应命中对应建议规则
	hasIssue := func(issueText string) bool {
		for _, issue := range stats.Issues {
			if issue.Issue == issueText {
				return true
			}
		}
		return false
	}
	if !hasIssue("未填 Jolpica ID（积分榜无法挂接故事链接）") {
		t.Errorf("应发现未填 Jolpica ID 的建议, Issues = %+v", stats.Issues)
	}
	if !hasIssue("缺少车手照片") {
		t.Errorf("应发现缺少车手照片的建议, Issues = %+v", stats.Issues)
	}

	// 构造占位正文后应命中"缺失"级规则
	d, err := repo.GetDriver(ctx, "zhou-guanyu")
	if err != nil {
		t.Fatalf("GetDriver: %v", err)
	}
	d.Story = "【占位】待写"
	if err := repo.UpdateDriver(ctx, "zhou-guanyu", d); err != nil {
		t.Fatalf("UpdateDriver: %v", err)
	}
	stats, err = repo.AdminStats(ctx)
	if err != nil {
		t.Fatalf("AdminStats: %v", err)
	}
	if !hasIssue("正文缺失或仍是占位内容") {
		t.Errorf("应发现占位正文问题, Issues = %+v", stats.Issues)
	}
}

func TestListDriverMappings(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	d := sampleDriver("mapped-driver")
	d.JolpicaID = "mapped_id"
	if err := repo.CreateDriver(ctx, d); err != nil {
		t.Fatalf("CreateDriver: %v", err)
	}
	// 未填 jolpica_id 的不进入映射
	other := sampleDriver("unmapped-driver")
	other.JolpicaID = ""
	if err := repo.CreateDriver(ctx, other); err != nil {
		t.Fatalf("CreateDriver: %v", err)
	}

	items, err := repo.ListDriverMappings(ctx)
	if err != nil {
		t.Fatalf("ListDriverMappings: %v", err)
	}
	if len(items) != 1 || items[0].JolpicaID != "mapped_id" || items[0].Slug != "mapped-driver" {
		t.Fatalf("items = %+v", items)
	}
}
