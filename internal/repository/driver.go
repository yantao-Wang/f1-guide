package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yantao-Wang/f1-guide/internal/domain"
)

// driverRow 是 drivers 表的完整行映射。
type driverRow struct {
	ID               int64    `db:"id"`
	Slug             string   `db:"slug"`
	Name             string   `db:"name"`
	Tagline          string   `db:"tagline"`
	TeamName         string   `db:"team_name"`
	TeamColor        string   `db:"team_color"`
	Number           int      `db:"number"`
	Championships    int      `db:"championships"`
	Country          string   `db:"country"`
	Story            string   `db:"story"`
	Personality      []string `db:"personality"`
	Trivia           []string `db:"trivia"`
	Quote            string   `db:"quote"`
	Featured         bool     `db:"featured"`
	FeaturedRaceYear *int     `db:"featured_race_year"`
	FeaturedRaceGP   *string  `db:"featured_race_gp"`
	JolpicaID        *string  `db:"jolpica_id"`
	ImageURL         *string  `db:"image_url"`
}

// ListDrivers 返回车手列表。featured 非空时仅返回精选位车手。
// 排序为入库顺序（内容首发优先级）。
func (p *Postgres) ListDrivers(ctx context.Context, featured *bool) ([]domain.DriverSummary, error) {
	rows, err := p.db.QueryxContext(ctx, `
		SELECT slug, name, number, tagline, team_name, team_color, featured, image_url
		FROM drivers
		WHERE ($1::boolean IS NULL OR featured = $1)
		ORDER BY id`, featured)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.DriverSummary, 0)
	for rows.Next() {
		var r struct {
			Slug      string  `db:"slug"`
			Name      string  `db:"name"`
			Number    int     `db:"number"`
			Tagline   string  `db:"tagline"`
			TeamName  string  `db:"team_name"`
			TeamColor string  `db:"team_color"`
			Featured  bool    `db:"featured"`
			ImageURL  *string `db:"image_url"`
		}
		if err := rows.StructScan(&r); err != nil {
			return nil, err
		}
		item := domain.DriverSummary{
			Slug:     r.Slug,
			Name:     r.Name,
			Number:   r.Number,
			Tagline:  r.Tagline,
			Team:     domain.Team{Name: r.TeamName, Color: r.TeamColor},
			Featured: r.Featured,
		}
		if r.ImageURL != nil {
			item.ImageURL = *r.ImageURL
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetDriver 返回车手详情（含关联名场面）。不存在时返回 ErrNotFound。
func (p *Postgres) GetDriver(ctx context.Context, slug string) (*domain.Driver, error) {
	var r driverRow
	err := p.db.GetContext(ctx, &r, `
		SELECT id, slug, name, tagline, team_name, team_color, number,
		       championships, country, story, personality, trivia, quote,
		       featured, featured_race_year, featured_race_gp, jolpica_id, image_url
		FROM drivers
		WHERE slug = $1`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	momentSlugs, err := p.listDriverMomentSlugs(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	d := &domain.Driver{
		Slug:           r.Slug,
		Name:           r.Name,
		Tagline:        r.Tagline,
		Team:           domain.Team{Name: r.TeamName, Color: r.TeamColor},
		Number:         r.Number,
		Championships:  r.Championships,
		Country:        r.Country,
		Story:          r.Story,
		Personality:    r.Personality,
		Trivia:         r.Trivia,
		Quote:          r.Quote,
		Featured:       r.Featured,
		RelatedMoments: momentSlugs,
	}
	if r.FeaturedRaceYear != nil && r.FeaturedRaceGP != nil {
		d.FeaturedRace = &domain.FeaturedRace{Year: *r.FeaturedRaceYear, GrandPrix: *r.FeaturedRaceGP}
	}
	if r.JolpicaID != nil {
		d.JolpicaID = *r.JolpicaID
	}
	if r.ImageURL != nil {
		d.ImageURL = *r.ImageURL
	}
	return d, nil
}

func (p *Postgres) listDriverMomentSlugs(ctx context.Context, driverID int64) ([]string, error) {
	slugs := make([]string, 0)
	err := p.db.SelectContext(ctx, &slugs, `
		SELECT m.slug
		FROM moments m
		JOIN moment_drivers md ON md.moment_id = m.id
		WHERE md.driver_id = $1
		ORDER BY m.year DESC, m.id`, driverID)
	return slugs, err
}
