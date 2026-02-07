package models

// Group is the projected group metadata state.
type Group struct {
	GroupID      string `json:"group_id"`
	Name         string `json:"name,omitempty"`
	About        string `json:"about,omitempty"`
	Picture      string `json:"picture,omitempty"`
	Geohash      string `json:"geohash,omitempty"`
	IsPrivate    bool   `json:"is_private"`
	IsRestricted bool   `json:"is_restricted"`
	IsVetted     bool   `json:"is_vetted"`
	IsHidden     bool   `json:"is_hidden"`
	IsClosed     bool   `json:"is_closed"`
	CreatedAt    int64  `json:"created_at"`
	CreatedBy    string `json:"created_by"`
	UpdatedAt    int64  `json:"updated_at"`
	UpdatedBy    string `json:"updated_by"`
}
