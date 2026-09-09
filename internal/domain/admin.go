package domain

import "time"

// AdminStats 后台仪表盘数据：内容概况 + 最近更新 + 完整性检查。
type AdminStats struct {
	DriverCount   int
	TrackCount    int
	MomentCount   int
	FeaturedCount int
	Recent        []RecentUpdate
	Issues        []ContentIssue
}

// RecentUpdate 最近一条内容更新（三表混合）。
type RecentUpdate struct {
	Kind      string // driver / track / moment
	Slug      string
	Name      string
	UpdatedAt time.Time
}

// ContentIssue 内容完整性检查发现的问题。
type ContentIssue struct {
	Kind     string // driver / track / moment
	Slug     string
	Name     string
	Issue    string // 中文问题描述
	Severity string // "缺失" / "建议"
}

// DriverMapping Jolpica driverId → 站内车手（供积分榜/赛程映射）。
type DriverMapping struct {
	JolpicaID string
	Slug      string
	Name      string
}
