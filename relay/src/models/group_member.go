package models

// GroupMember is a projected membership row.
type GroupMember struct {
	GroupID    string `json:"group_id"`
	PubKey     string `json:"pubkey"`
	AddedAt    int64  `json:"added_at"`
	AddedBy    string `json:"added_by"`
	RoleName   string `json:"role_name,omitempty"`
	PromotedAt int64  `json:"promoted_at,omitempty"`
	PromotedBy string `json:"promoted_by,omitempty"`
}
