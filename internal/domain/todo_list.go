package domain

import "time"

// TodoList represents a list of todo items.
type TodoList struct {
	ID        int       `json:"id" example:"1"`
	Title     string    `json:"title" example:"My Shopping List"`
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T15:04:05Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T15:04:05Z"`
}
