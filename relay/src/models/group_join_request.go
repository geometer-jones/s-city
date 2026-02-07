package models

// GroupJoinRequest is a pending request for membership.
type GroupJoinRequest struct {
	GroupID   string `json:"group_id"`
	PubKey    string `json:"pubkey"`
	CreatedAt int64  `json:"created_at"`
}
