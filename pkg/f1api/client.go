// Package f1api 封装 Jolpica（Ergast 数据镜像）HTTP API。
//
// 数据源：https://api.jolpi.ca/ergast/f1/，免费、无鉴权、限速 500 req/h，
// 正赛后约 1 小时更新。本包只负责原始 JSON 的取数与反序列化，
// 领域映射（车队色、中文名、比赛状态计算）在 internal/service 完成。
package f1api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultBaseURL 是 Jolpica 官方 API 根地址。
const DefaultBaseURL = "https://api.jolpi.ca"

// DefaultTimeout 单次上游请求超时。
const DefaultTimeout = 10 * time.Second

// Client Jolpica HTTP 客户端。并发安全（字段只读）。
type Client struct {
	base *url.URL
	hc   *http.Client
}

// New 创建客户端。baseURL 形如 https://api.jolpi.ca。
func New(baseURL string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("f1api: parse base url: %w", err)
	}
	return &Client{base: base, hc: &http.Client{Timeout: DefaultTimeout}}, nil
}

// seasonPath 把赛季号转为路径段：0/负数表示当前赛季（Jolpica 的 current 别名）。
func seasonPath(season int) string {
	if season <= 0 {
		return "current"
	}
	return strconv.Itoa(season)
}

// --- 原始模型（字段名与 Jolpica JSON 一致，数值均为字符串） ---

// Circuit 赛道信息。
type Circuit struct {
	CircuitName string   `json:"circuitName"`
	Location    Location `json:"Location"`
}

// Location 赛道所在地区。
type Location struct {
	Country string `json:"country"`
}

// Driver 车手引用（赛果/积分榜条目内嵌）。
type Driver struct {
	DriverID   string `json:"driverId"`
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

// Constructor 车队引用。
type Constructor struct {
	Name string `json:"name"`
}

// ScheduleRace 赛程条目（/f1/{season}.json 的 Races[] 元素）。
type ScheduleRace struct {
	Round    string  `json:"round"`
	RaceName string  `json:"raceName"`
	Date     string  `json:"date"`
	Circuit  Circuit `json:"Circuit"`
}

// WinnerEntry 单场赛果中的一条成绩。
type WinnerEntry struct {
	Position string `json:"position"`
	Driver   Driver `json:"Driver"`
}

// WinnerRace 单场冠军（/f1/{season}/results/1.json 的 Races[] 元素，
// 每场完赛比赛一行，Results[0] 即该场冠军）。
type WinnerRace struct {
	Round   string        `json:"round"`
	Results []WinnerEntry `json:"Results"`
}

// Standing 车手积分榜行。
type Standing struct {
	Position     string        `json:"position"`
	Points       string        `json:"points"`
	Wins         string        `json:"wins"`
	Driver       Driver        `json:"Driver"`
	Constructors []Constructor `json:"Constructors"`
}

// ConstructorStanding 车队积分榜行。
type ConstructorStanding struct {
	Position    string      `json:"position"`
	Points      string      `json:"points"`
	Wins        string      `json:"wins"`
	Constructor Constructor `json:"Constructor"`
}

// --- 响应信封 ---

type scheduleResponse struct {
	MRData struct {
		RaceTable struct {
			Races []ScheduleRace `json:"Races"`
		} `json:"RaceTable"`
	} `json:"MRData"`
}

type winnersResponse struct {
	MRData struct {
		RaceTable struct {
			Races []WinnerRace `json:"Races"`
		} `json:"RaceTable"`
	} `json:"MRData"`
}

type driverStandingsResponse struct {
	MRData struct {
		StandingsTable struct {
			StandingsLists []struct {
				DriverStandings []Standing `json:"DriverStandings"`
			} `json:"StandingsLists"`
		} `json:"StandingsTable"`
	} `json:"MRData"`
}

type constructorStandingsResponse struct {
	MRData struct {
		StandingsTable struct {
			StandingsLists []struct {
				ConstructorStandings []ConstructorStanding `json:"ConstructorStandings"`
			} `json:"StandingsLists"`
		} `json:"StandingsTable"`
	} `json:"MRData"`
}

// --- 查询方法 ---

// Schedule 返回指定赛季完整赛程。
func (c *Client) Schedule(ctx context.Context, season int) ([]ScheduleRace, error) {
	var resp scheduleResponse
	if err := c.get(ctx, "/ergast/f1/"+seasonPath(season)+".json", &resp); err != nil {
		return nil, err
	}
	return resp.MRData.RaceTable.Races, nil
}

// Winners 返回指定赛季每场完赛比赛的冠军（results/1 端点，单次请求全量）。
func (c *Client) Winners(ctx context.Context, season int) ([]WinnerRace, error) {
	var resp winnersResponse
	if err := c.get(ctx, "/ergast/f1/"+seasonPath(season)+"/results/1.json", &resp); err != nil {
		return nil, err
	}
	return resp.MRData.RaceTable.Races, nil
}

// DriverStandings 返回指定赛季车手积分榜。
func (c *Client) DriverStandings(ctx context.Context, season int) ([]Standing, error) {
	var resp driverStandingsResponse
	if err := c.get(ctx, "/ergast/f1/"+seasonPath(season)+"/driverStandings.json", &resp); err != nil {
		return nil, err
	}
	if len(resp.MRData.StandingsTable.StandingsLists) == 0 {
		return nil, fmt.Errorf("f1api: driver standings: empty StandingsLists")
	}
	return resp.MRData.StandingsTable.StandingsLists[0].DriverStandings, nil
}

// ConstructorStandings 返回指定赛季车队积分榜。
func (c *Client) ConstructorStandings(ctx context.Context, season int) ([]ConstructorStanding, error) {
	var resp constructorStandingsResponse
	if err := c.get(ctx, "/ergast/f1/"+seasonPath(season)+"/constructorStandings.json", &resp); err != nil {
		return nil, err
	}
	if len(resp.MRData.StandingsTable.StandingsLists) == 0 {
		return nil, fmt.Errorf("f1api: constructor standings: empty StandingsLists")
	}
	return resp.MRData.StandingsTable.StandingsLists[0].ConstructorStandings, nil
}

// get 发起 GET 请求并解码 JSON 信封。
func (c *Client) get(ctx context.Context, path string, out any) error {
	u := c.base.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("f1api: build request: %w", err)
	}
	req.Header.Set("User-Agent", "f1-guide-mvp/0.1 (+https://github.com/yantao-Wang/f1-guide)")

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("f1api: get %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
		return fmt.Errorf("f1api: get %s: status %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("f1api: decode %s: %w", path, err)
	}
	return nil
}
