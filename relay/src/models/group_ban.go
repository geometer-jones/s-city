package models

// GroupBan records a user ban for a group.
type GroupBan struct {
	GroupID   string `json:"group_id"`
	PubKey    string `json:"pubkey"`
	Reason    string `json:"reason"`
	BannedAt  int64  `json:"banned_at"`
	BannedBy  string `json:"banned_by"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}
