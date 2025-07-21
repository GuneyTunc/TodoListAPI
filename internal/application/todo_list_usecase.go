package application

import (
	"context"
	"fmt"

	"todo-api/internal/domain" // Modül yolunuzu projenizin go.mod dosyasına göre güncelleyin
)

// TodoListUseCase defines the business logic for TodoList operations.
type TodoListUseCase struct {
	todoListRepo domain.TodoListRepository
	todoRepo     domain.TodoRepository // TodoList silinirken altındaki Todolar için de kullanılabilir
}

// NewTodoListUseCase creates a new TodoListUseCase.
func NewTodoListUseCase(todoListRepo domain.TodoListRepository, todoRepo domain.TodoRepository) *TodoListUseCase {
	return &TodoListUseCase{
		todoListRepo: todoListRepo,
		todoRepo:     todoRepo,
	}
}

// CreateTodoList creates a new TodoList.
func (uc *TodoListUseCase) CreateTodoList(ctx context.Context, title string) (*domain.TodoList, error) {
	// Başlığın zaten var olup olmadığını kontrol edin (isteğe bağlı, repo'da da UNIQUE kısıtlaması var)
	// exists, err := uc.todoListRepo.TodoListExistsByTitle(ctx, title) // Eğer bu fonksiyon repository'de varsa uncomment yapın
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to check existing todo list by title: %w", err)
	// }
	// if exists {
	// 	return nil, fmt.Errorf("a todo list with this title already exists")
	// }

	todoList, err := uc.todoListRepo.CreateTodoList(ctx, title)
	if err != nil {
		return nil, fmt.Errorf("failed to create todo list: %w", err)
	}
	return todoList, nil
}

// GetTodoLists retrieves all TodoLists.
func (uc *TodoListUseCase) GetTodoLists(ctx context.Context) ([]domain.TodoList, error) {
	todoLists, err := uc.todoListRepo.GetTodoLists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve todo lists: %w", err)
	}
	return todoLists, nil
}

// GetTodoListByID retrieves a TodoList by its ID.
func (uc *TodoListUseCase) GetTodoListByID(ctx context.Context, id int) (*domain.TodoList, error) {
	todoList, err := uc.todoListRepo.GetTodoListByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve todo list by ID: %w", err)
	}
	return todoList, nil
}

// UpdateTodoList updates an existing TodoList.
func (uc *TodoListUseCase) UpdateTodoList(ctx context.Context, id int, title string) (*domain.TodoList, error) {
	// Listenin varlığını kontrol et
	exists, err := uc.todoListRepo.TodoListExists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check todo list existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("to-do list not found")
	}

	updatedTodoList, err := uc.todoListRepo.UpdateTodoList(ctx, id, title)
	if err != nil {
		return nil, fmt.Errorf("failed to update todo list: %w", err)
	}
	return updatedTodoList, nil
}

// DeleteTodoList deletes a TodoList by its ID. It should also handle cascading deletion of associated Todos.
func (uc *TodoListUseCase) DeleteTodoList(ctx context.Context, id int) error {
	// Listenin varlığını kontrol et
	exists, err := uc.todoListRepo.TodoListExists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check todo list existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("to-do list not found")
	}

	err = uc.todoListRepo.DeleteTodoList(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete todo list: %w", err)
	}
	// İlişkili Todolar otomatik olarak silinecektir (FOREIGN KEY ON DELETE CASCADE sayesinde)
	return nil
}
