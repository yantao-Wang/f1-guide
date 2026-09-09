package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

func TestAdminErrMapping(t *testing.T) {
	ctx := context.Background()

	t.Run("ErrNotFound → ErrNotFound", func(t *testing.T) {
		store := &testutil.FakeAdminStore{ErrNotFound: true}
		svc := NewAdmin(store)

		if err := svc.DeleteDriver(ctx, "no-such"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
		if err := svc.UpdateTrack(ctx, "no-such", testutil.SampleTrack()); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ErrConflict → ErrConflict", func(t *testing.T) {
		store := &testutil.FakeAdminStore{ErrConflict: true}
		svc := NewAdmin(store)

		if err := svc.CreateDriver(ctx, testutil.SampleDriver()); !errors.Is(err, ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
		if err := svc.CreateMoment(ctx, testutil.SampleMoment()); !errors.Is(err, ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("未知错误透传", func(t *testing.T) {
		boom := errors.New("boom")
		store := &testutil.FakeAdminStore{Err: boom}
		svc := NewAdmin(store)

		if err := svc.DeleteTrack(ctx, "x"); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
	})
}

func TestAdminPassThrough(t *testing.T) {
	ctx := context.Background()
	store := &testutil.FakeAdminStore{}
	svc := NewAdmin(store)

	d := testutil.SampleDriver()
	if err := svc.CreateDriver(ctx, d); err != nil {
		t.Fatalf("CreateDriver: %v", err)
	}
	if len(store.CreatedDrivers) != 1 || store.CreatedDrivers[0].Slug != "zhou-guanyu" {
		t.Fatalf("CreatedDrivers = %+v", store.CreatedDrivers)
	}

	tr := testutil.SampleTrack()
	if err := svc.UpdateTrack(ctx, "shanghai", tr); err != nil {
		t.Fatalf("UpdateTrack: %v", err)
	}
	if len(store.UpdatedTracks) != 1 {
		t.Fatalf("UpdatedTracks = %+v", store.UpdatedTracks)
	}

	m := testutil.SampleMoment()
	if err := svc.CreateMoment(ctx, m); err != nil {
		t.Fatalf("CreateMoment: %v", err)
	}
	if err := svc.DeleteMoment(ctx, "2021-abu-dhabi"); err != nil {
		t.Fatalf("DeleteMoment: %v", err)
	}
	if len(store.CreatedMoments) != 1 || len(store.DeletedMoments) != 1 || store.DeletedMoments[0] != "2021-abu-dhabi" {
		t.Fatalf("moment 调用记录 = created:%+v deleted:%+v", store.CreatedMoments, store.DeletedMoments)
	}

	// AdminStats 直接透传
	stats, err := svc.AdminStats(ctx)
	if err != nil {
		t.Fatalf("AdminStats: %v", err)
	}
	if !reflect.DeepEqual(stats, store.Stats) || store.StatsCalls != 1 {
		t.Fatalf("AdminStats 透传失败: stats=%+v calls=%d", stats, store.StatsCalls)
	}
}

func TestAdminStatsDomainTypes(t *testing.T) {
	// 保证 domain.AdminStats 结构可被 fake 与 handler 共享
	stats := domain.AdminStats{
		DriverCount: 2,
		Recent:      []domain.RecentUpdate{{Kind: "driver", Slug: "a", Name: "A"}},
		Issues:      []domain.ContentIssue{{Kind: "track", Slug: "b", Issue: "缺少赛道图", Severity: "建议"}},
	}
	if stats.DriverCount != 2 || len(stats.Recent) != 1 || len(stats.Issues) != 1 {
		t.Fatal("AdminStats 结构错误")
	}
	if stats.Issues[0].Severity != "建议" {
		t.Fatal("ContentIssue 字段错误")
	}
}
