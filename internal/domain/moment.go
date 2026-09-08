package domain

// 名场面类型（对应 OpenAPI MomentSummary.type 枚举）。
const (
	MomentTypeChampionship = "championship" // 冠军争夺
	MomentTypeOvertake     = "overtake"     // 经典超车
	MomentTypeSafety       = "safety"       // 事故安全
	MomentTypeRain         = "rain"         // 雨战传奇
)

// MomentSummary 名场面列表项（对应 OpenAPI MomentSummary）。
type MomentSummary struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Year      int    `json:"year"`
	GrandPrix string `json:"grandPrix"`
	Type      string `json:"type"`
}

// Moment 名场面详情（对应 OpenAPI Moment）。
type Moment struct {
	MomentSummary
	Background string `json:"background"`
	WhyClassic string `json:"whyClassic"`
	VideoURL   string `json:"videoUrl,omitempty"`
	// RelatedDrivers 关联车手 slug。
	RelatedDrivers []string `json:"relatedDrivers"`
	// RelatedTrack 关联赛道 slug（一个名场面至多对应一条赛道）。
	RelatedTrack string `json:"relatedTrack,omitempty"`
}
