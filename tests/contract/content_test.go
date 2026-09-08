// Package contract 契约测试：校验 handler 实际响应符合 api/openapi.yaml。
//
// 契约先行（D16）：实现一旦偏离契约，本测试在 CI 中立即失败。
package contract

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"

	"github.com/yantao-Wang/f1-guide/internal/domain"
	"github.com/yantao-Wang/f1-guide/internal/handler"
	"github.com/yantao-Wang/f1-guide/internal/router"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/internal/testutil"
)

func loadDoc(t *testing.T) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	path, err := filepath.Abs("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		t.Fatalf("加载 openapi.yaml 失败: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi.yaml 结构不合法: %v", err)
	}
	return doc
}

func mustNewRouter(t *testing.T, doc *openapi3.T) routers.Router {
	t.Helper()
	r, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatalf("构建契约路由器失败: %v", err)
	}
	return r
}

// newAppRouter 构建与生产同构的路由（fake 数据源，覆盖契约全字段）。
func newAppRouter() http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &testutil.FakeStore{
		Drivers: []domain.DriverSummary{{
			Slug: "zhou-guanyu", Name: "周冠宇", Tagline: "让中国国旗第一次出现在 F1 积分区的人",
			Team: domain.Team{Name: "凯迪拉克", Color: "#00A1E0"}, Featured: true,
		}},
		DriverDetail: testutil.SampleDriver(),
		Tracks: []domain.TrackSummary{{
			Slug: "shanghai", Name: "上海国际赛车场", Country: "中国",
			Tagline: "上字形的速度迷宫", Type: domain.TrackTypePermanent,
		}},
		TrackDetail: testutil.SampleTrack(),
		Moments: []domain.MomentSummary{{
			Slug: "2021-abu-dhabi", Title: "2021 阿布扎比 · 最后一圈决出世界冠军",
			Year: 2021, GrandPrix: "阿布扎比大奖赛", Type: domain.MomentTypeChampionship,
		}},
		MomentDetail: testutil.SampleMoment(),
	}
	svc := service.NewContent(store, store, store)
	statsSvc := service.NewStats(&testutil.FakeStatsClient{})
	return router.New(log, handler.NewHealth(log), handler.NewContent(svc), handler.NewStats(statsSvc), handler.NewPages(svc, statsSvc))
}

// validateResponse 用 kin-openapi 校验实际响应符合契约。
func validateResponse(t *testing.T, r routers.Router, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	route, pathParams, err := r.FindRoute(req)
	if err != nil {
		t.Fatalf("请求未在契约中定义: %s %s: %v", req.Method, req.URL.Path, err)
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status: rec.Code,
		Header: rec.Header(),
		Body:   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
	}
	if err := openapi3filter.ValidateResponse(context.Background(), input); err != nil {
		t.Fatalf("%s %s 响应违反契约: %v", req.Method, req.URL.Path, err)
	}
}

func TestContentEndpointsMatchContract(t *testing.T) {
	doc := loadDoc(t)
	appRouter := newAppRouter()
	contractRouter := mustNewRouter(t, doc)

	cases := []struct {
		method string
		target string
	}{
		{"GET", "/api/v1/drivers"},
		{"GET", "/api/v1/drivers?featured=true"},
		{"GET", "/api/v1/drivers/zhou-guanyu"},
		{"GET", "/api/v1/tracks"},
		{"GET", "/api/v1/tracks/shanghai"},
		{"GET", "/api/v1/moments"},
		{"GET", "/api/v1/moments/2021-abu-dhabi"},
		{"GET", "/api/v1/schedule"},
		{"GET", "/api/v1/schedule?season=2026"},
		{"GET", "/api/v1/standings/drivers"},
		{"GET", "/api/v1/standings/drivers?season=2026"},
		{"GET", "/api/v1/standings/constructors"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.target, nil)
		rec := httptest.NewRecorder()
		appRouter.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("%s %s: status = %d, body = %s", c.method, c.target, rec.Code, rec.Body.String())
			continue
		}
		validateResponse(t, contractRouter, req, rec)
	}
}

func TestNotFoundResponseMatchesContract(t *testing.T) {
	doc := loadDoc(t)
	appRouter := newAppRouter()
	contractRouter := mustNewRouter(t, doc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/drivers/no-such", nil)
	rec := httptest.NewRecorder()
	appRouter.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	validateResponse(t, contractRouter, req, rec)
}

func TestBadSeasonResponseMatchesContract(t *testing.T) {
	doc := loadDoc(t)
	appRouter := newAppRouter()
	contractRouter := mustNewRouter(t, doc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedule?season=abc", nil)
	rec := httptest.NewRecorder()
	appRouter.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	validateResponse(t, contractRouter, req, rec)
}

func TestUpstreamErrorResponseMatchesContract(t *testing.T) {
	doc := loadDoc(t)
	contractRouter := mustNewRouter(t, doc)

	// 上游数据源故障时的应用路由器
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &testutil.FakeStore{}
	svc := service.NewContent(store, store, store)
	statsSvc := service.NewStats(&testutil.FakeStatsClient{Err: errors.New("upstream down")})
	appRouter := router.New(log, handler.NewHealth(log), handler.NewContent(svc), handler.NewStats(statsSvc), handler.NewPages(svc, statsSvc))

	for _, target := range []string{
		"/api/v1/schedule",
		"/api/v1/standings/drivers",
		"/api/v1/standings/constructors",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		appRouter.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Errorf("GET %s status = %d, want %d", target, rec.Code, http.StatusBadGateway)
			continue
		}
		validateResponse(t, contractRouter, req, rec)
	}
}
