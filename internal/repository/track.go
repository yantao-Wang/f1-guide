package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/yantao-Wang/f1-guide/internal/domain"
)

// ListTracks 返回赛道列表，入库顺序（内容首发优先级）。
func (p *Postgres) ListTracks(ctx context.Context) ([]domain.TrackSummary, error) {
	rows, err := p.db.QueryxContext(ctx, `
		SELECT slug, name, country, tagline, type
		FROM tracks
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.TrackSummary, 0)
	for rows.Next() {
		var r struct {
			Slug    string `db:"slug"`
			Name    string `db:"name"`
			Country string `db:"country"`
			Tagline string `db:"tagline"`
			Type    string `db:"type"`
		}
		if err := rows.StructScan(&r); err != nil {
			return nil, err
		}
		items = append(items, domain.TrackSummary{
			Slug:    r.Slug,
			Name:    r.Name,
			Country: r.Country,
			Tagline: r.Tagline,
			Type:    r.Type,
		})
	}
	return items, rows.Err()
}

// GetTrack 返回赛道详情（含关联名场面）。不存在时返回 ErrNotFound。
func (p *Postgres) GetTrack(ctx context.Context, slug string) (*domain.Track, error) {
	var r struct {
		ID             int64    `db:"id"`
		Slug           string   `db:"slug"`
		Name           string   `db:"name"`
		Country        string   `db:"country"`
		Tagline        string   `db:"tagline"`
		Type           string   `db:"type"`
		FirstGrandPrix int      `db:"first_grand_prix"`
		LengthKm       float64  `db:"length_km"`
		Laps           int      `db:"laps"`
		Highlights     []string `db:"highlights"`
		CircuitMapURL  *string  `db:"circuit_map_url"`
	}
	err := p.db.GetContext(ctx, &r, `
		SELECT id, slug, name, country, tagline, type,
		       first_grand_prix, length_km, laps, highlights, circuit_map_url
		FROM tracks
		WHERE slug = $1`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	momentSlugs := make([]string, 0)
	err = p.db.SelectContext(ctx, &momentSlugs, `
		SELECT slug FROM moments WHERE track_id = $1 ORDER BY year DESC, id`, r.ID)
	if err != nil {
		return nil, err
	}

	t := &domain.Track{
		TrackSummary: domain.TrackSummary{
			Slug:    r.Slug,
			Name:    r.Name,
			Country: r.Country,
			Tagline: r.Tagline,
			Type:    r.Type,
		},
		FirstGrandPrix: r.FirstGrandPrix,
		LengthKm:       r.LengthKm,
		Laps:           r.Laps,
		Highlights:     r.Highlights,
		RelatedMoments: momentSlugs,
	}
	if r.CircuitMapURL != nil {
		t.CircuitMapURL = *r.CircuitMapURL
	}
	return t, nil
}
