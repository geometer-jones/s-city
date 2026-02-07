package models

// GroupEvent links a stored event to a group projection.
type GroupEvent struct {
	GroupID   string `json:"group_id"`
	EventID   string `json:"event_id"`
	CreatedAt int64  `json:"created_at"`
}
