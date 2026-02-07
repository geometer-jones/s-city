package models

// DeletedEvent marks a previously accepted event as deleted.
type DeletedEvent struct {
	EventID   string `json:"event_id"`
	DeletedAt int64  `json:"deleted_at"`
	DeletedBy string `json:"deleted_by"`
	Reason    string `json:"reason,omitempty"`
}
