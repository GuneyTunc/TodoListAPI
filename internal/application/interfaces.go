package application

import (
	"context"
)

// TodoListRepository provides an interface for interacting with TodoList data storage.
type TodoListRepository interface {
	CreateTodoList(ctx context.Context, title string) (*TodoList, error)
	GetTodoLists(ctx context.Context) ([]*TodoList, error)
	GetTodoListByID(ctx context.Context, id int) (*TodoList, error)
	UpdateTodoList(ctx context.Context, id int, title string) (*TodoList, error)
	DeleteTodoList(ctx context.Context, id int) error
}

// TodoRepository provides an interface for interacting with Todo data storage.
type TodoRepository interface {
	CreateTodo(ctx context.Context, listID int, title, description string, completed bool) (*Todo, error)
	GetTodosByListID(ctx context.Context, listID int) ([]Todo, error)
	GetTodoByIDAndListID(ctx context.Context, todoID, listID int) (*Todo, error)
	UpdateTodo(ctx context.Context, listID, todoID int, title, description string, completed bool) (*Todo, error)
	DeleteTodo(ctx context.Context, listID, todoID int) error
	TodoExistsInList(ctx context.Context, listID, todoID int) (bool, error)
}

// AIService provides an interface for interacting with an AI-based description suggestion service.
type AIService interface {
	SuggestDescription(ctx context.Context, title string) (string, error)
}
