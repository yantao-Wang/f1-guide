package domain

// 赛道类型（对应 OpenAPI TrackSummary.type 枚举）。
const (
	TrackTypePermanent = "permanent" // 专用赛车跑道
	TrackTypeStreet    = "street"    // 临时市街赛道
)

// TrackSummary 赛道列表项（对应 OpenAPI TrackSummary）。
type TrackSummary struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Tagline string `json:"tagline"`
	Type    string `json:"type"`
}

// Track 赛道图鉴详情（对应 OpenAPI Track）。
type Track struct {
	TrackSummary
	FirstGrandPrix int      `json:"firstGrandPrix"`
	LengthKm       float64  `json:"lengthKm"`
	Laps           int      `json:"laps"`
	Highlights     []string `json:"highlights"`
	CircuitMapURL  string   `json:"circuitMapUrl,omitempty"`
	// RelatedMoments 关联名场面 slug。
	RelatedMoments []string `json:"relatedMoments"`
}
