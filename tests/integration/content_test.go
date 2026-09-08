//go:build integration

// Package integration 集成测试：真实 PostgreSQL 上的数据访问验证。
//
// 运行方式（本地）：
//
//	export F1GUIDE_TEST_DATABASE_URL=postgres://f1guide:f1guide@localhost:5432/f1guide?sslmode=disable
//	go test -tags=integration ./tests/integration/...
//
// CI 在 PostgreSQL service container 上运行（.github/workflows/ci.yml integration 任务）。
package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/repository"
	"github.com/yantao-Wang/f1-guide/pkg/db"
)

func TestMain(m *testing.M) {
	if os.Getenv("F1GUIDE_TEST_DATABASE_URL") == "" {
		fmt.Println("跳过集成测试：未设置 F1GUIDE_TEST_DATABASE_URL")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// setup 迁移测试库、清空内容表，返回数据访问层与底层连接（用于 seed）。
func setup(t *testing.T) (*repository.Postgres, *sqlx.DB) {
	t.Helper()
	url := os.Getenv("F1GUIDE_TEST_DATABASE_URL")

	if err := db.MigrateUp(url); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	pg, err := db.Open(url)
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	t.Cleanup(func() { pg.Close() })

	if _, err := pg.Exec(`TRUNCATE drivers, tracks, moments RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("清空表失败: %v", err)
	}
	return repository.New(pg), pg
}

// seedContent 插入一组互相关联的示例内容。
func seedContent(t *testing.T, pg *sqlx.DB) {
	t.Helper()
	if _, err := pg.Exec(`
		INSERT INTO drivers (slug, name, tagline, team_name, team_color, number, championships, country, story, personality, trivia, quote, featured, featured_race_year, featured_race_gp)
		VALUES ('zhou-guanyu', '周冠宇', '示例人设', '凯迪拉克', '#00A1E0', 24, 0, '中国', '示例故事', '{沉稳,坚韧}', '{示例冷知识}', '示例金句', TRUE, 2022, '巴林大奖赛'),
		       ('other-driver', '另一位车手', '示例人设', '法拉利', '#DC0000', 16, 4, '摩纳哥', '示例故事', '{}', '{}', '示例金句', FALSE, NULL, NULL)`); err != nil {
		t.Fatalf("插入 drivers 失败: %v", err)
	}

	if _, err := pg.Exec(`
		INSERT INTO tracks (slug, name, country, tagline, type, first_grand_prix, length_km, laps, highlights, circuit_map_url)
		VALUES ('shanghai', '上海国际赛车场', '中国', '上字形的速度迷宫', 'permanent', 2004, 5.451, 56, '{一号弯,长直道}', '/static/tracks/shanghai.svg')`); err != nil {
		t.Fatalf("插入 track 失败: %v", err)
	}

	if _, err := pg.Exec(`
		INSERT INTO moments (slug, title, year, grand_prix, type, background, why_classic, video_url, track_id)
		VALUES ('2021-abu-dhabi', '2021 阿布扎比 · 最后一圈决出世界冠军', 2021, '阿布扎比大奖赛', 'championship', '示例背景', '示例原因', 'https://www.bilibili.com/video/BV-example',
		        (SELECT id FROM tracks WHERE slug = 'shanghai'))`); err != nil {
		t.Fatalf("插入 moment 失败: %v", err)
	}

	if _, err := pg.Exec(`
		INSERT INTO moment_drivers (moment_id, driver_id)
		SELECT m.id, d.id FROM moments m, drivers d WHERE m.slug = '2021-abu-dhabi' AND d.slug = 'zhou-guanyu'`); err != nil {
		t.Fatalf("插入 moment_drivers 失败: %v", err)
	}
}

func TestListDriversAndFeaturedFilter(t *testing.T) {
	repo, pg := setup(t)
	seedContent(t, pg)
	ctx := context.Background()

	all, err := repo.ListDrivers(ctx, nil)
	if err != nil {
		t.Fatalf("ListDrivers(nil): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("全量 = %d 条, want 2", len(all))
	}

	featured, err := repo.ListDrivers(ctx, boolPtr(true))
	if err != nil {
		t.Fatalf("ListDrivers(featured): %v", err)
	}
	if len(featured) != 1 || featured[0].Slug != "zhou-guanyu" {
		t.Fatalf("精选 = %+v, want 仅 [zhou-guanyu]", featured)
	}
	if featured[0].Team.Color != "#00A1E0" {
		t.Fatalf("Team.Color = %q, want #00A1E0", featured[0].Team.Color)
	}
}

func TestGetDriverWithRelations(t *testing.T) {
	repo, pg := setup(t)
	seedContent(t, pg)

	d, err := repo.GetDriver(context.Background(), "zhou-guanyu")
	if err != nil {
		t.Fatalf("GetDriver: %v", err)
	}

	if d.Number != 24 || d.Championships != 0 || d.Country != "中国" {
		t.Fatalf("基础字段错误: %+v", d)
	}
	// text[] 扫描验证
	if len(d.Personality) != 2 || d.Personality[0] != "沉稳" {
		t.Fatalf("Personality = %v, want [沉稳 坚韧]", d.Personality)
	}
	if len(d.Trivia) != 1 || d.Trivia[0] != "示例冷知识" {
		t.Fatalf("Trivia = %v, want [示例冷知识]", d.Trivia)
	}
	if d.FeaturedRace == nil || d.FeaturedRace.Year != 2022 || d.FeaturedRace.GrandPrix != "巴林大奖赛" {
		t.Fatalf("FeaturedRace = %+v", d.FeaturedRace)
	}
	// 关联名场面
	if len(d.RelatedMoments) != 1 || d.RelatedMoments[0] != "2021-abu-dhabi" {
		t.Fatalf("RelatedMoments = %v, want [2021-abu-dhabi]", d.RelatedMoments)
	}

	// 无推荐比赛的车手：FeaturedRace 应为空
	other, err := repo.GetDriver(context.Background(), "other-driver")
	if err != nil {
		t.Fatalf("GetDriver(other): %v", err)
	}
	if other.FeaturedRace != nil {
		t.Fatalf("FeaturedRace = %+v, want nil", other.FeaturedRace)
	}
	if len(other.RelatedMoments) != 0 {
		t.Fatalf("RelatedMoments = %v, want 空", other.RelatedMoments)
	}
}

func TestGetDriverNotFound(t *testing.T) {
	repo, _ := setup(t)

	_, err := repo.GetDriver(context.Background(), "no-such")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestTrackDetail(t *testing.T) {
	repo, pg := setup(t)
	seedContent(t, pg)

	tr, err := repo.GetTrack(context.Background(), "shanghai")
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if tr.FirstGrandPrix != 2004 || tr.LengthKm != 5.451 || tr.Laps != 56 {
		t.Fatalf("数值字段错误: %+v", tr)
	}
	if tr.Type != domain.TrackTypePermanent {
		t.Fatalf("Type = %q, want permanent", tr.Type)
	}
	if len(tr.Highlights) != 2 || tr.Highlights[1] != "长直道" {
		t.Fatalf("Highlights = %v, want [一号弯 长直道]", tr.Highlights)
	}
	if len(tr.RelatedMoments) != 1 || tr.RelatedMoments[0] != "2021-abu-dhabi" {
		t.Fatalf("RelatedMoments = %v, want [2021-abu-dhabi]", tr.RelatedMoments)
	}

	if _, err := repo.GetTrack(context.Background(), "no-such"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestMomentDetailWithBidirectionalRelations(t *testing.T) {
	repo, pg := setup(t)
	seedContent(t, pg)

	m, err := repo.GetMoment(context.Background(), "2021-abu-dhabi")
	if err != nil {
		t.Fatalf("GetMoment: %v", err)
	}
	if m.Type != domain.MomentTypeChampionship {
		t.Fatalf("Type = %q, want championship", m.Type)
	}
	if len(m.RelatedDrivers) != 1 || m.RelatedDrivers[0] != "zhou-guanyu" {
		t.Fatalf("RelatedDrivers = %v, want [zhou-guanyu]", m.RelatedDrivers)
	}
	if m.RelatedTrack != "shanghai" {
		t.Fatalf("RelatedTrack = %q, want shanghai", m.RelatedTrack)
	}
	if m.VideoURL != "https://www.bilibili.com/video/BV-example" {
		t.Fatalf("VideoURL = %q", m.VideoURL)
	}

	// 列表按年份倒序
	items, err := repo.ListMoments(context.Background())
	if err != nil {
		t.Fatalf("ListMoments: %v", err)
	}
	if len(items) != 1 || items[0].Slug != "2021-abu-dhabi" {
		t.Fatalf("items = %+v", items)
	}
}

func boolPtr(v bool) *bool { return &v }
