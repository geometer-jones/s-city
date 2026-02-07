package models

// GroupRole is a named permission set inside a group.
type GroupRole struct {
	GroupID     string   `json:"group_id"`
	RoleName    string   `json:"role_name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	CreatedAt   int64    `json:"created_at"`
	CreatedBy   string   `json:"created_by"`
	UpdatedAt   int64    `json:"updated_at"`
	UpdatedBy   string   `json:"updated_by"`
}
