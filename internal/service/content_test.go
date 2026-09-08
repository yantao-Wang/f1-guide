package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

func newTestContent(store *testutil.FakeStore) *Content {
	return NewContent(store, store, store)
}

func TestListDriversFeaturedFilter(t *testing.T) {
	store := &testutil.FakeStore{Drivers: []domain.DriverSummary{
		{Slug: "a", Featured: true},
		{Slug: "b", Featured: false},
	}}
	svc := newTestContent(store)

	// featured=true 只返回精选
	got, err := svc.ListDrivers(context.Background(), boolPtr(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "a" {
		t.Fatalf("featured=true 结果 = %+v, want 仅 [a]", got)
	}

	// featured=nil 返回全部
	got, err = svc.ListDrivers(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("featured=nil 结果 = %d 条, want 2", len(got))
	}
}

func TestGetDriverNotFound(t *testing.T) {
	svc := newTestContent(&testutil.FakeStore{})

	_, err := svc.GetDriver(context.Background(), "no-such-driver")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetDriverFound(t *testing.T) {
	store := &testutil.FakeStore{DriverDetail: testutil.SampleDriver()}
	svc := newTestContent(store)

	d, err := svc.GetDriver(context.Background(), "zhou-guanyu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Name != "周冠宇" {
		t.Fatalf("name = %q, want 周冠宇", d.Name)
	}
	if len(d.RelatedMoments) != 1 || d.RelatedMoments[0] != "2021-abu-dhabi" {
		t.Fatalf("RelatedMoments = %v, want [2021-abu-dhabi]", d.RelatedMoments)
	}
}

func TestGetTrackNotFound(t *testing.T) {
	svc := newTestContent(&testutil.FakeStore{})

	_, err := svc.GetTrack(context.Background(), "no-such-track")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetMomentNotFound(t *testing.T) {
	svc := newTestContent(&testutil.FakeStore{})

	_, err := svc.GetMoment(context.Background(), "no-such-moment")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestRepositoryErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	store := &testutil.FakeStore{Err: boom}
	svc := newTestContent(store)

	if _, err := svc.ListTracks(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want 原样透传 boom", err)
	}
	// repository.ErrNotFound 以外的错误不得被吞掉或误映射
	if _, err := svc.GetDriver(context.Background(), "any"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want 原样透传 boom", err)
	}
}

func boolPtr(v bool) *bool { return &v }
