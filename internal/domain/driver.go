// Package domain 定义领域模型。纯数据结构，无业务逻辑、无外部依赖。
package domain

// Team 车队信息。
type Team struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// FeaturedRace 车手故事推荐的一场比赛。
type FeaturedRace struct {
	Year      int    `json:"year"`
	GrandPrix string `json:"grandPrix"`
}

// DriverSummary 车手列表项（对应 OpenAPI DriverSummary）。
type DriverSummary struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Tagline  string `json:"tagline"`
	Team     Team   `json:"team"`
	Featured bool   `json:"featured"`
}

// Driver 车手故事详情（对应 OpenAPI Driver）。
type Driver struct {
	Slug          string        `json:"slug"`
	Name          string        `json:"name"`
	Tagline       string        `json:"tagline"`
	Team          Team          `json:"team"`
	Number        int           `json:"number"`
	Championships int           `json:"championships"`
	Country       string        `json:"country"`
	Story         string        `json:"story"`
	Personality   []string      `json:"personality"`
	Trivia        []string      `json:"trivia"`
	Quote         string        `json:"quote"`
	Featured      bool          `json:"featured"`
	FeaturedRace  *FeaturedRace `json:"featuredRace,omitempty"`
	// RelatedMoments 关联名场面 slug。
	RelatedMoments []string `json:"relatedMoments"`
}
