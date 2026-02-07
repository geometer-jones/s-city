package models

// EventTag stores one normalized tag row from an event.
type EventTag struct {
	ID       int64    `json:"id,omitempty"`
	EventID  string   `json:"event_id"`
	TagIndex int      `json:"tag_index"`
	TagName  string   `json:"tag_name"`
	TagValue string   `json:"tag_value"`
	TagArray []string `json:"tag_array"`
}
