// 后台内容管理服务：三表写操作 + 仪表盘统计。
//
// 接口由本包（消费方）定义，repository 包提供实现（与 content.go 同原则）。
package service

import (
	"context"
	"errors"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/repository"
)

// ErrConflict 表示唯一约束冲突（slug 或 jolpica_id 重复）。handler 映射为 409。
var ErrConflict = errors.New("service: conflict")

// AdminStore 后台写操作数据访问接口。
type AdminStore interface {
	CreateDriver(ctx context.Context, d *domain.Driver) error
	UpdateDriver(ctx context.Context, slug string, d *domain.Driver) error
	DeleteDriver(ctx context.Context, slug string) error

	CreateTrack(ctx context.Context, t *domain.Track) error
	UpdateTrack(ctx context.Context, slug string, t *domain.Track) error
	DeleteTrack(ctx context.Context, slug string) error

	CreateMoment(ctx context.Context, m *domain.Moment) error
	UpdateMoment(ctx context.Context, slug string, m *domain.Moment) error
	DeleteMoment(ctx context.Context, slug string) error

	AdminStats(ctx context.Context) (domain.AdminStats, error)
}

// Admin 后台内容管理服务。只做错误映射与透传（与 Content 同风格）。
type Admin struct {
	store AdminStore
}

// NewAdmin 组装后台管理服务。
func NewAdmin(store AdminStore) *Admin {
	return &Admin{store: store}
}

// CreateDriver 新建车手。
func (s *Admin) CreateDriver(ctx context.Context, d *domain.Driver) error {
	return mapAdminErr(s.store.CreateDriver(ctx, d))
}

// UpdateDriver 按旧 slug 更新车手。
func (s *Admin) UpdateDriver(ctx context.Context, slug string, d *domain.Driver) error {
	return mapAdminErr(s.store.UpdateDriver(ctx, slug, d))
}

// DeleteDriver 删除车手。
func (s *Admin) DeleteDriver(ctx context.Context, slug string) error {
	return mapAdminErr(s.store.DeleteDriver(ctx, slug))
}

// CreateTrack 新建赛道。
func (s *Admin) CreateTrack(ctx context.Context, t *domain.Track) error {
	return mapAdminErr(s.store.CreateTrack(ctx, t))
}

// UpdateTrack 按旧 slug 更新赛道。
func (s *Admin) UpdateTrack(ctx context.Context, slug string, t *domain.Track) error {
	return mapAdminErr(s.store.UpdateTrack(ctx, slug, t))
}

// DeleteTrack 删除赛道。
func (s *Admin) DeleteTrack(ctx context.Context, slug string) error {
	return mapAdminErr(s.store.DeleteTrack(ctx, slug))
}

// CreateMoment 新建名场面。
func (s *Admin) CreateMoment(ctx context.Context, m *domain.Moment) error {
	return mapAdminErr(s.store.CreateMoment(ctx, m))
}

// UpdateMoment 按旧 slug 更新名场面。
func (s *Admin) UpdateMoment(ctx context.Context, slug string, m *domain.Moment) error {
	return mapAdminErr(s.store.UpdateMoment(ctx, slug, m))
}

// DeleteMoment 删除名场面。
func (s *Admin) DeleteMoment(ctx context.Context, slug string) error {
	return mapAdminErr(s.store.DeleteMoment(ctx, slug))
}

// AdminStats 返回后台仪表盘统计。
func (s *Admin) AdminStats(ctx context.Context) (domain.AdminStats, error) {
	return s.store.AdminStats(ctx)
}

// mapAdminErr 把 repository 层错误映射为 service 层语义错误。
func mapAdminErr(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, repository.ErrConflict) {
		return ErrConflict
	}
	return err
}
