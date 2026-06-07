package entities

import "time"

type Notification struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	ActionRoute string    `json:"action_route"`
	IsRead      bool      `json:"is_read"`
	CreatedAt   time.Time `json:"created_at"`
}
