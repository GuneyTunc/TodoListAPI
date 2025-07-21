package domain

import "time"

// Todo represents a single todo item within a todo list.
type Todo struct {
	ID          int       `json:"id" example:"101"`
	ListID      int       `json:"list_id" example:"1"` // Foreign key to TodoList
	Title       string    `json:"title" example:"Buy groceries"`
	Description string    `json:"description" example:"Milk, eggs, bread, and fruits."`
	Completed   bool      `json:"completed" example:"false"`
	CreatedAt   time.Time `json:"created_at" example:"2023-01-01T15:04:05Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2023-01-01T15:04:05Z"`
}
