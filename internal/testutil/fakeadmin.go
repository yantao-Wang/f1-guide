package testutil

import (
	"context"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/repository"
)

// FakeAdminStore 实现 service.AdminStore。
// 记录调用参数供断言，不模拟完整内存 CRUD（写操作成功与否由注入的错误控制）。
type FakeAdminStore struct {
	// CreatedDrivers / UpdatedDrivers 等按调用顺序记录收到的 domain 对象。
	CreatedDrivers []*domain.Driver
	UpdatedDrivers []*domain.Driver
	DeletedDrivers []string
	CreatedTracks  []*domain.Track
	UpdatedTracks  []*domain.Track
	DeletedTracks  []string
	CreatedMoments []*domain.Moment
	UpdatedMoments []*domain.Moment
	DeletedMoments []string
	Stats          domain.AdminStats
	StatsCalls     int
	Err            error // 非空时所有写方法返回该错误（映射层测试用）
	ErrNotFound    bool  // 真时写方法返回 repository.ErrNotFound
	ErrConflict    bool  // 真时写方法返回 repository.ErrConflict
}

// writeErr 按开关决定本次写操作返回的错误（Err 优先级最高）。
func (f *FakeAdminStore) writeErr() error {
	if f.Err != nil {
		return f.Err
	}
	if f.ErrNotFound {
		return repository.ErrNotFound
	}
	if f.ErrConflict {
		return repository.ErrConflict
	}
	return nil
}

// CreateDriver 实现 AdminStore。
func (f *FakeAdminStore) CreateDriver(_ context.Context, d *domain.Driver) error {
	f.CreatedDrivers = append(f.CreatedDrivers, d)
	return f.writeErr()
}

// UpdateDriver 实现 AdminStore。
func (f *FakeAdminStore) UpdateDriver(_ context.Context, _ string, d *domain.Driver) error {
	f.UpdatedDrivers = append(f.UpdatedDrivers, d)
	return f.writeErr()
}

// DeleteDriver 实现 AdminStore。
func (f *FakeAdminStore) DeleteDriver(_ context.Context, slug string) error {
	f.DeletedDrivers = append(f.DeletedDrivers, slug)
	return f.writeErr()
}

// CreateTrack 实现 AdminStore。
func (f *FakeAdminStore) CreateTrack(_ context.Context, t *domain.Track) error {
	f.CreatedTracks = append(f.CreatedTracks, t)
	return f.writeErr()
}

// UpdateTrack 实现 AdminStore。
func (f *FakeAdminStore) UpdateTrack(_ context.Context, _ string, t *domain.Track) error {
	f.UpdatedTracks = append(f.UpdatedTracks, t)
	return f.writeErr()
}

// DeleteTrack 实现 AdminStore。
func (f *FakeAdminStore) DeleteTrack(_ context.Context, slug string) error {
	f.DeletedTracks = append(f.DeletedTracks, slug)
	return f.writeErr()
}

// CreateMoment 实现 AdminStore。
func (f *FakeAdminStore) CreateMoment(_ context.Context, m *domain.Moment) error {
	f.CreatedMoments = append(f.CreatedMoments, m)
	return f.writeErr()
}

// UpdateMoment 实现 AdminStore。
func (f *FakeAdminStore) UpdateMoment(_ context.Context, _ string, m *domain.Moment) error {
	f.UpdatedMoments = append(f.UpdatedMoments, m)
	return f.writeErr()
}

// DeleteMoment 实现 AdminStore。
func (f *FakeAdminStore) DeleteMoment(_ context.Context, slug string) error {
	f.DeletedMoments = append(f.DeletedMoments, slug)
	return f.writeErr()
}

// AdminStats 实现 AdminStore。
func (f *FakeAdminStore) AdminStats(_ context.Context) (domain.AdminStats, error) {
	f.StatsCalls++
	if f.Err != nil {
		return domain.AdminStats{}, f.Err
	}
	return f.Stats, nil
}
