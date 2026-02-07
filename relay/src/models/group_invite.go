package models

// GroupInvite is a projected invite code row.
type GroupInvite struct {
	GroupID       string `json:"group_id"`
	Code          string `json:"code"`
	ExpiresAt     int64  `json:"expires_at,omitempty"`
	MaxUsageCount int    `json:"max_usage_count,omitempty"`
	UsageCount    int    `json:"usage_count,omitempty"`
	CreatedAt     int64  `json:"created_at"`
	CreatedBy     string `json:"created_by"`
}
