package domain

import "context"

// TodoListRepository defines the interface for interacting with TodoList data storage.
type TodoListRepository interface {
	CreateTodoList(ctx context.Context, title string) (*TodoList, error)
	GetTodoLists(ctx context.Context) ([]TodoList, error)
	GetTodoListByID(ctx context.Context, id int) (*TodoList, error)
	UpdateTodoList(ctx context.Context, id int, title string) (*TodoList, error)
	DeleteTodoList(ctx context.Context, id int) error
	TodoListExists(ctx context.Context, id int) (bool, error) // Listenin varlığını kontrol eder
	// TodoListExistsByTitle(ctx context.Context, title string) (bool, error) // Eğer başlığa göre varlık kontrolü istenirse eklenebilir
}

// TodoRepository defines the interface for interacting with Todo data storage.
type TodoRepository interface {
	CreateTodo(ctx context.Context, listID int, title, description string) (*Todo, error)
	GetTodosByListID(ctx context.Context, listID int) ([]Todo, error)
	GetTodoByID(ctx context.Context, listID, todoID int) (*Todo, error) // Belirli bir listedeki Todo'yu getirir
	UpdateTodo(ctx context.Context, todo *Todo) error
	DeleteTodo(ctx context.Context, listID, todoID int) error         // Belirli bir listedeki Todo'yu siler
	TodoExists(ctx context.Context, listID, todoID int) (bool, error) // Todo'nun varlığını kontrol eder
}

// AIService defines the interface for interacting with an AI service for domain-level operations.
type AIService interface {
	GenerateDescription(ctx context.Context, title string) (string, error)
	// Diğer AI operasyonları (örn. öneri, kategori belirleme vb.) buraya eklenebilir
}
