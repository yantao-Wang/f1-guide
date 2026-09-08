// Package testutil 提供测试用 fake 实现（供 service / handler / contract 测试共用）。
package testutil

import (
	"context"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/repository"
)

// FakeStore 同时实现 DriverStore / TrackStore / MomentStore。
// 数据由测试直接填充，可按需注入错误。
type FakeStore struct {
	Drivers      []domain.DriverSummary
	DriverDetail *domain.Driver

	Tracks      []domain.TrackSummary
	TrackDetail *domain.Track

	Moments      []domain.MomentSummary
	MomentDetail *domain.Moment

	// Err 非空时所有方法直接返回该错误。
	Err error
}

// SampleDriver 返回结构完整的示例车手（覆盖契约全部字段）。
func SampleDriver() *domain.Driver {
	return &domain.Driver{
		Slug:           "zhou-guanyu",
		Name:           "周冠宇",
		Tagline:        "让中国国旗第一次出现在 F1 积分区的人",
		Team:           domain.Team{Name: "凯迪拉克", Color: "#00A1E0"},
		Number:         24,
		Championships:  0,
		Country:        "中国",
		Story:          "示例故事",
		Personality:    []string{"沉稳", "坚韧"},
		Trivia:         []string{"示例冷知识"},
		Quote:          "示例金句",
		Featured:       true,
		FeaturedRace:   &domain.FeaturedRace{Year: 2022, GrandPrix: "巴林大奖赛"},
		RelatedMoments: []string{"2021-abu-dhabi"},
	}
}

// SampleTrack 返回结构完整的示例赛道。
func SampleTrack() *domain.Track {
	return &domain.Track{
		TrackSummary: domain.TrackSummary{
			Slug:    "shanghai",
			Name:    "上海国际赛车场",
			Country: "中国",
			Tagline: "上字形的速度迷宫",
			Type:    domain.TrackTypePermanent,
		},
		FirstGrandPrix: 2004,
		LengthKm:       5.451,
		Laps:           56,
		Highlights:     []string{"一号弯", "长直道"},
		CircuitMapURL:  "/static/tracks/shanghai.svg",
		RelatedMoments: []string{"2021-abu-dhabi"},
	}
}

// SampleMoment 返回结构完整的示例名场面。
func SampleMoment() *domain.Moment {
	return &domain.Moment{
		MomentSummary: domain.MomentSummary{
			Slug:      "2021-abu-dhabi",
			Title:     "2021 阿布扎比 · 最后一圈决出世界冠军",
			Year:      2021,
			GrandPrix: "阿布扎比大奖赛",
			Type:      domain.MomentTypeChampionship,
		},
		Background:     "示例背景",
		WhyClassic:     "示例原因",
		VideoURL:       "https://www.bilibili.com/video/BV-example",
		RelatedDrivers: []string{"zhou-guanyu"},
		RelatedTrack:   "shanghai",
	}
}

// ListDrivers 实现 DriverStore。
func (f *FakeStore) ListDrivers(_ context.Context, featured *bool) ([]domain.DriverSummary, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if featured == nil {
		return f.Drivers, nil
	}
	filtered := make([]domain.DriverSummary, 0)
	for _, d := range f.Drivers {
		if d.Featured == *featured {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}

// GetDriver 实现 DriverStore。
func (f *FakeStore) GetDriver(_ context.Context, slug string) (*domain.Driver, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.DriverDetail != nil && f.DriverDetail.Slug == slug {
		return f.DriverDetail, nil
	}
	return nil, repository.ErrNotFound
}

// ListTracks 实现 TrackStore。
func (f *FakeStore) ListTracks(_ context.Context) ([]domain.TrackSummary, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Tracks, nil
}

// GetTrack 实现 TrackStore。
func (f *FakeStore) GetTrack(_ context.Context, slug string) (*domain.Track, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.TrackDetail != nil && f.TrackDetail.Slug == slug {
		return f.TrackDetail, nil
	}
	return nil, repository.ErrNotFound
}

// ListMoments 实现 MomentStore。
func (f *FakeStore) ListMoments(_ context.Context) ([]domain.MomentSummary, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Moments, nil
}

// GetMoment 实现 MomentStore。
func (f *FakeStore) GetMoment(_ context.Context, slug string) (*domain.Moment, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.MomentDetail != nil && f.MomentDetail.Slug == slug {
		return f.MomentDetail, nil
	}
	return nil, repository.ErrNotFound
}
