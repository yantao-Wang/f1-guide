// Package service 承载业务逻辑。依赖方向：handler → service → repository → domain。
//
// 数据访问接口由本包（消费方）定义，repository 包提供实现，
// 便于测试时注入 fake 实现。
package service

import (
	"context"
	"errors"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/repository"
)

// ErrNotFound 表示业务资源不存在。handler 将其映射为 404。
var ErrNotFound = errors.New("service: not found")

// DriverStore 车手数据访问接口。
type DriverStore interface {
	ListDrivers(ctx context.Context, featured *bool) ([]domain.DriverSummary, error)
	GetDriver(ctx context.Context, slug string) (*domain.Driver, error)
}

// TrackStore 赛道数据访问接口。
type TrackStore interface {
	ListTracks(ctx context.Context) ([]domain.TrackSummary, error)
	GetTrack(ctx context.Context, slug string) (*domain.Track, error)
}

// MomentStore 名场面数据访问接口。
type MomentStore interface {
	ListMoments(ctx context.Context) ([]domain.MomentSummary, error)
	GetMoment(ctx context.Context, slug string) (*domain.Moment, error)
}

// Content 内容板块（车手故事 / 赛道图鉴 / 名场面）业务逻辑。
type Content struct {
	drivers DriverStore
	tracks  TrackStore
	moments MomentStore
}

// NewContent 组装内容服务。
func NewContent(drivers DriverStore, tracks TrackStore, moments MomentStore) *Content {
	return &Content{drivers: drivers, tracks: tracks, moments: moments}
}

// ListDrivers 返回车手列表。featured 非空时仅返回精选位。
func (s *Content) ListDrivers(ctx context.Context, featured *bool) ([]domain.DriverSummary, error) {
	return s.drivers.ListDrivers(ctx, featured)
}

// GetDriver 返回车手详情，不存在时返回 ErrNotFound。
func (s *Content) GetDriver(ctx context.Context, slug string) (*domain.Driver, error) {
	d, err := s.drivers.GetDriver(ctx, slug)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return d, err
}

// ListTracks 返回赛道列表。
func (s *Content) ListTracks(ctx context.Context) ([]domain.TrackSummary, error) {
	return s.tracks.ListTracks(ctx)
}

// GetTrack 返回赛道详情，不存在时返回 ErrNotFound。
func (s *Content) GetTrack(ctx context.Context, slug string) (*domain.Track, error) {
	t, err := s.tracks.GetTrack(ctx, slug)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return t, err
}

// ListMoments 返回名场面列表。
func (s *Content) ListMoments(ctx context.Context) ([]domain.MomentSummary, error) {
	return s.moments.ListMoments(ctx)
}

// GetMoment 返回名场面详情，不存在时返回 ErrNotFound。
func (s *Content) GetMoment(ctx context.Context, slug string) (*domain.Moment, error) {
	m, err := s.moments.GetMoment(ctx, slug)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return m, err
}
