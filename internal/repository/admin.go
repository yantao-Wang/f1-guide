// 后台写操作数据访问实现（content 三表的 Create/Update/Delete + 仪表盘统计）。
//
// 全部方法由 service 包定义的 AdminStore 接口消费（消费方定义接口原则）。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/yantao-Wang/f1-guide/internal/domain"
)

// ErrConflict 表示唯一约束冲突（slug 或 jolpica_id 重复）。service 层映射为 409。
var ErrConflict = errors.New("repository: conflict")

// isUniqueViolation 判断是否为 PostgreSQL 唯一约束冲突（SQLSTATE 23505）。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// nullableStr 把空字符串转为 NULL 参数（可选字段统一约定：空串不入库）。
func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// execResult 校验单行写操作影响行数：0 行视为资源不存在。
func execResult(res sql.Result, err error) error {
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- 车手 ---

// CreateDriver 新建车手。slug/jolpica_id 冲突返回 ErrConflict。
func (p *Postgres) CreateDriver(ctx context.Context, d *domain.Driver) error {
	var featuredYear any
	var featuredGP any
	if d.FeaturedRace != nil {
		featuredYear = d.FeaturedRace.Year
		featuredGP = d.FeaturedRace.GrandPrix
	}
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO drivers (slug, name, tagline, team_name, team_color, number,
		                     championships, country, story, personality, trivia, quote,
		                     featured, featured_race_year, featured_race_gp, jolpica_id, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		d.Slug, d.Name, d.Tagline, d.Team.Name, d.Team.Color, d.Number,
		d.Championships, d.Country, d.Story, d.Personality, d.Trivia, d.Quote,
		d.Featured, featuredYear, featuredGP, nullableStr(d.JolpicaID), nullableStr(d.ImageURL))
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

// UpdateDriver 按旧 slug 定位更新车手（slug 允许改，改后旧链接失效）。
func (p *Postgres) UpdateDriver(ctx context.Context, slug string, d *domain.Driver) error {
	var featuredYear any
	var featuredGP any
	if d.FeaturedRace != nil {
		featuredYear = d.FeaturedRace.Year
		featuredGP = d.FeaturedRace.GrandPrix
	}
	res, err := p.db.ExecContext(ctx, `
		UPDATE drivers SET slug = $1, name = $2, tagline = $3, team_name = $4, team_color = $5,
		       number = $6, championships = $7, country = $8, story = $9, personality = $10,
		       trivia = $11, quote = $12, featured = $13, featured_race_year = $14,
		       featured_race_gp = $15, jolpica_id = $16, image_url = $17, updated_at = now()
		WHERE slug = $18`,
		d.Slug, d.Name, d.Tagline, d.Team.Name, d.Team.Color, d.Number,
		d.Championships, d.Country, d.Story, d.Personality, d.Trivia, d.Quote,
		d.Featured, featuredYear, featuredGP, nullableStr(d.JolpicaID), nullableStr(d.ImageURL), slug)
	return execResult(res, err)
}

// DeleteDriver 删除车手。关联 moment_drivers 由外键 CASCADE 清理。
func (p *Postgres) DeleteDriver(ctx context.Context, slug string) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM drivers WHERE slug = $1`, slug)
	return execResult(res, err)
}

// --- 赛道 ---

// CreateTrack 新建赛道。slug 冲突返回 ErrConflict。
func (p *Postgres) CreateTrack(ctx context.Context, t *domain.Track) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO tracks (slug, name, country, tagline, type, first_grand_prix,
		                    length_km, laps, highlights, circuit_map_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		t.Slug, t.Name, t.Country, t.Tagline, t.Type, t.FirstGrandPrix,
		t.LengthKm, t.Laps, t.Highlights, nullableStr(t.CircuitMapURL))
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

// UpdateTrack 按旧 slug 定位更新赛道。
func (p *Postgres) UpdateTrack(ctx context.Context, slug string, t *domain.Track) error {
	res, err := p.db.ExecContext(ctx, `
		UPDATE tracks SET slug = $1, name = $2, country = $3, tagline = $4, type = $5,
		       first_grand_prix = $6, length_km = $7, laps = $8, highlights = $9,
		       circuit_map_url = $10, updated_at = now()
		WHERE slug = $11`,
		t.Slug, t.Name, t.Country, t.Tagline, t.Type, t.FirstGrandPrix,
		t.LengthKm, t.Laps, t.Highlights, nullableStr(t.CircuitMapURL), slug)
	return execResult(res, err)
}

// DeleteTrack 删除赛道。关联名场面的 track_id 由外键 ON DELETE SET NULL 自动解绑。
func (p *Postgres) DeleteTrack(ctx context.Context, slug string) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM tracks WHERE slug = $1`, slug)
	return execResult(res, err)
}

// --- 名场面 ---

// CreateMoment 新建名场面：事务内写入主表并重建关联车手。
func (p *Postgres) CreateMoment(ctx context.Context, m *domain.Moment) error {
	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	err = tx.GetContext(ctx, &id, `
		INSERT INTO moments (slug, title, year, grand_prix, type, background, why_classic, video_url, track_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, (SELECT id FROM tracks WHERE slug = NULLIF($9, '')))
		RETURNING id`,
		m.Slug, m.Title, m.Year, m.GrandPrix, m.Type, m.Background, m.WhyClassic,
		nullableStr(m.VideoURL), m.RelatedTrack)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}

	if err := insertMomentDrivers(ctx, tx, id, m.RelatedDrivers); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateMoment 按旧 slug 定位更新名场面（slug 允许改），事务内全删重插关联车手。
func (p *Postgres) UpdateMoment(ctx context.Context, slug string, m *domain.Moment) error {
	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	err = tx.GetContext(ctx, &id, `
		UPDATE moments SET slug = $1, title = $2, year = $3, grand_prix = $4, type = $5,
		       background = $6, why_classic = $7, video_url = $8,
		       track_id = (SELECT id FROM tracks WHERE slug = NULLIF($9, '')), updated_at = now()
		WHERE slug = $10
		RETURNING id`,
		m.Slug, m.Title, m.Year, m.GrandPrix, m.Type, m.Background, m.WhyClassic,
		nullableStr(m.VideoURL), m.RelatedTrack, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM moment_drivers WHERE moment_id = $1`, id); err != nil {
		return err
	}
	if err := insertMomentDrivers(ctx, tx, id, m.RelatedDrivers); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteMoment 删除名场面。关联 moment_drivers 由外键 CASCADE 清理。
func (p *Postgres) DeleteMoment(ctx context.Context, slug string) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM moments WHERE slug = $1`, slug)
	return execResult(res, err)
}

// insertMomentDrivers 按车手 slug 批量插入名场面关联（事务内执行）。
func insertMomentDrivers(ctx context.Context, tx *sqlx.Tx, momentID int64, driverSlugs []string) error {
	if len(driverSlugs) == 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO moment_drivers (moment_id, driver_id)
		SELECT $1, id FROM drivers WHERE slug = ANY($2)`, momentID, driverSlugs)
	return err
}

// --- 仪表盘统计 ---

// AdminStats 返回后台仪表盘数据：内容计数 + 最近更新 + 完整性检查。
func (p *Postgres) AdminStats(ctx context.Context) (domain.AdminStats, error) {
	stats := domain.AdminStats{
		Recent: make([]domain.RecentUpdate, 0),
		Issues: make([]domain.ContentIssue, 0),
	}

	for _, q := range []struct {
		dst   *int
		query string
	}{
		{&stats.DriverCount, `SELECT count(*) FROM drivers`},
		{&stats.TrackCount, `SELECT count(*) FROM tracks`},
		{&stats.MomentCount, `SELECT count(*) FROM moments`},
		{&stats.FeaturedCount, `SELECT count(*) FROM drivers WHERE featured`},
	} {
		if err := p.db.GetContext(ctx, q.dst, q.query); err != nil {
			return domain.AdminStats{}, err
		}
	}

	// recentRow 是最近更新查询的行映射（domain 结构不带 db tag，映射留在 repository 层）。
	var recentRows []struct {
		Kind      string    `db:"kind"`
		Slug      string    `db:"slug"`
		Name      string    `db:"name"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	if err := p.db.SelectContext(ctx, &recentRows, `
		SELECT kind, slug, name, updated_at FROM (
			SELECT 'driver' AS kind, slug, name, updated_at FROM drivers
			UNION ALL SELECT 'track', slug, name, updated_at FROM tracks
			UNION ALL SELECT 'moment', slug, title AS name, updated_at FROM moments
		) r ORDER BY updated_at DESC LIMIT 10`); err != nil {
		return domain.AdminStats{}, err
	}
	for _, r := range recentRows {
		stats.Recent = append(stats.Recent, domain.RecentUpdate{
			Kind: r.Kind, Slug: r.Slug, Name: r.Name, UpdatedAt: r.UpdatedAt,
		})
	}

	// 完整性检查规则：每条一个独立小查询，命中行拼进 Issues。
	rules := []struct {
		kind, severity, issue, query string
	}{
		{"driver", "缺失", "正文缺失或仍是占位内容", `SELECT slug, name FROM drivers WHERE story = '' OR story LIKE '%【占位】%'`},
		{"driver", "缺失", "缺少名言金句", `SELECT slug, name FROM drivers WHERE quote = ''`},
		{"driver", "缺失", "缺少性格标签", `SELECT slug, name FROM drivers WHERE cardinality(personality) = 0`},
		{"driver", "建议", "未填 Jolpica ID（积分榜无法挂接故事链接）", `SELECT slug, name FROM drivers WHERE jolpica_id IS NULL`},
		{"driver", "建议", "缺少车手照片", `SELECT slug, name FROM drivers WHERE image_url IS NULL`},
		{"track", "缺失", "缺少赛道亮点", `SELECT slug, name FROM tracks WHERE cardinality(highlights) = 0`},
		{"track", "建议", "缺少赛道图", `SELECT slug, name FROM tracks WHERE circuit_map_url IS NULL`},
		{"moment", "缺失", "正文缺失或仍是占位内容", `SELECT slug, title AS name FROM moments WHERE background = '' OR why_classic = '' OR background LIKE '%【占位】%' OR why_classic LIKE '%【占位】%'`},
		{"moment", "建议", "缺少视频链接", `SELECT slug, title AS name FROM moments WHERE video_url IS NULL`},
		{"moment", "建议", "未关联赛道", `SELECT slug, title AS name FROM moments WHERE track_id IS NULL`},
		{"moment", "建议", "未关联车手", `SELECT slug, title AS name FROM moments WHERE NOT EXISTS (SELECT 1 FROM moment_drivers md WHERE md.moment_id = moments.id)`},
	}
	for _, rule := range rules {
		if err := p.appendIssues(ctx, &stats.Issues, rule.kind, rule.severity, rule.issue, rule.query); err != nil {
			return domain.AdminStats{}, err
		}
	}
	return stats, nil
}

// appendIssues 执行一条完整性检查查询，把命中行追加到 Issues。
func (p *Postgres) appendIssues(ctx context.Context, issues *[]domain.ContentIssue, kind, severity, issue, query string) error {
	rows, err := p.db.QueryxContext(ctx, query)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var r struct {
			Slug string `db:"slug"`
			Name string `db:"name"`
		}
		if err := rows.StructScan(&r); err != nil {
			return err
		}
		*issues = append(*issues, domain.ContentIssue{
			Kind:     kind,
			Slug:     r.Slug,
			Name:     r.Name,
			Issue:    issue,
			Severity: severity,
		})
	}
	return rows.Err()
}

// --- 映射（stats 服务消费） ---

// ListDriverMappings 返回全部已填 Jolpica ID 的车手映射（Jolpica driverId → 站内 slug/中文名）。
func (p *Postgres) ListDriverMappings(ctx context.Context) ([]domain.DriverMapping, error) {
	var rows []struct {
		JolpicaID string `db:"jolpica_id"`
		Slug      string `db:"slug"`
		Name      string `db:"name"`
	}
	if err := p.db.SelectContext(ctx, &rows, `
		SELECT jolpica_id, slug, name FROM drivers WHERE jolpica_id IS NOT NULL ORDER BY id`); err != nil {
		return nil, err
	}
	items := make([]domain.DriverMapping, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.DriverMapping{JolpicaID: r.JolpicaID, Slug: r.Slug, Name: r.Name})
	}
	return items, nil
}
