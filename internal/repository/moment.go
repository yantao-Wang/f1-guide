package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yantao-Wang/f1-guide/internal/domain"
)

// ListMoments 返回名场面列表，按年份倒序（新场面在前）。
func (p *Postgres) ListMoments(ctx context.Context) ([]domain.MomentSummary, error) {
	rows, err := p.db.QueryxContext(ctx, `
		SELECT slug, title, year, grand_prix, type
		FROM moments
		ORDER BY year DESC, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.MomentSummary, 0)
	for rows.Next() {
		var r struct {
			Slug      string `db:"slug"`
			Title     string `db:"title"`
			Year      int    `db:"year"`
			GrandPrix string `db:"grand_prix"`
			Type      string `db:"type"`
		}
		if err := rows.StructScan(&r); err != nil {
			return nil, err
		}
		items = append(items, domain.MomentSummary{
			Slug:      r.Slug,
			Title:     r.Title,
			Year:      r.Year,
			GrandPrix: r.GrandPrix,
			Type:      r.Type,
		})
	}
	return items, rows.Err()
}

// GetMoment 返回名场面详情（含关联车手与赛道）。不存在时返回 ErrNotFound。
func (p *Postgres) GetMoment(ctx context.Context, slug string) (*domain.Moment, error) {
	var r struct {
		ID         int64   `db:"id"`
		Slug       string  `db:"slug"`
		Title      string  `db:"title"`
		Year       int     `db:"year"`
		GrandPrix  string  `db:"grand_prix"`
		Type       string  `db:"type"`
		Background string  `db:"background"`
		WhyClassic string  `db:"why_classic"`
		VideoURL   *string `db:"video_url"`
		TrackID    *int64  `db:"track_id"`
	}
	err := p.db.GetContext(ctx, &r, `
		SELECT id, slug, title, year, grand_prix, type,
		       background, why_classic, video_url, track_id
		FROM moments
		WHERE slug = $1`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	driverSlugs := make([]string, 0)
	err = p.db.SelectContext(ctx, &driverSlugs, `
		SELECT d.slug
		FROM drivers d
		JOIN moment_drivers md ON md.driver_id = d.id
		WHERE md.moment_id = $1
		ORDER BY d.id`, r.ID)
	if err != nil {
		return nil, err
	}

	m := &domain.Moment{
		MomentSummary: domain.MomentSummary{
			Slug:      r.Slug,
			Title:     r.Title,
			Year:      r.Year,
			GrandPrix: r.GrandPrix,
			Type:      r.Type,
		},
		Background:     r.Background,
		WhyClassic:     r.WhyClassic,
		RelatedDrivers: driverSlugs,
	}
	if r.VideoURL != nil {
		m.VideoURL = *r.VideoURL
	}
	if r.TrackID != nil {
		var trackSlug string
		err = p.db.GetContext(ctx, &trackSlug, `SELECT slug FROM tracks WHERE id = $1`, *r.TrackID)
		if err != nil {
			return nil, err
		}
		m.RelatedTrack = trackSlug
	}
	return m, nil
}
