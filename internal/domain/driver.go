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
	Number   int    `json:"number"`
	Tagline  string `json:"tagline"`
	Team     Team   `json:"team"`
	Featured bool   `json:"featured"`
	// ImageURL 车手照片路径（/uploads 前缀），SSR 车手卡片显示用，不进公开 JSON（契约不变）。
	ImageURL string `json:"-"`
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
	// JolpicaID 上游 Jolpica driverId。后台录入专用，不进公开 API（契约不变）。
	JolpicaID string `json:"-"`
	// ImageURL 车手照片路径（/uploads 前缀），只服务 SSR 页面，不进公开 JSON。
	ImageURL string `json:"-"`
}
