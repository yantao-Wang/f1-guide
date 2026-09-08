package domain

// RaceStatus 比赛状态（对应 OpenAPI Race.status 枚举）。
type RaceStatus string

const (
	RaceStatusUpcoming  RaceStatus = "upcoming"
	RaceStatusCompleted RaceStatus = "completed"
	RaceStatusCancelled RaceStatus = "cancelled"
)

// Race 赛程中的一场比赛（对应 OpenAPI Race）。
// WinnerName 仅供页面展示，不出现在 JSON 响应（契约只约定 winner slug）。
type Race struct {
	Round      int        `json:"round"`
	Date       string     `json:"date"`
	GrandPrix  string     `json:"grandPrix"`
	Circuit    string     `json:"circuit"`
	Country    string     `json:"country"`
	Status     RaceStatus `json:"status"`
	WinnerSlug string     `json:"winner,omitempty"`
	WinnerName string     `json:"-"`
}

// Schedule 当季赛程（对应 OpenAPI Schedule）。
type Schedule struct {
	Season int    `json:"season"`
	Races  []Race `json:"races"`
}

// DriverStandingRow 车手积分榜行（对应 OpenAPI DriverStandingRow）。
type DriverStandingRow struct {
	Position   int    `json:"position"`
	DriverSlug string `json:"driverSlug"`
	DriverName string `json:"driverName"`
	Team       Team   `json:"team"`
	Points     int    `json:"points"`
	Wins       int    `json:"wins"`
}

// ConstructorStandingRow 车队积分榜行（对应 OpenAPI ConstructorStandingRow）。
type ConstructorStandingRow struct {
	Position int  `json:"position"`
	Team     Team `json:"team"`
	Points   int  `json:"points"`
	Wins     int  `json:"wins"`
}

// DriverStandings 车手积分榜（对应 OpenAPI DriverStandings）。
type DriverStandings struct {
	Season int                 `json:"season"`
	Items  []DriverStandingRow `json:"items"`
}

// ConstructorStandings 车队积分榜（对应 OpenAPI ConstructorStandings）。
type ConstructorStandings struct {
	Season int                      `json:"season"`
	Items  []ConstructorStandingRow `json:"items"`
}
