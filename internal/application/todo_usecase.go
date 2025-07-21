package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"todo-api/internal/domain" // Modül yolunuzu projenizin go.mod dosyasına göre güncelleyin
)

// TodoUseCase defines the business logic for Todo operations.
type TodoUseCase struct {
	todoRepo     domain.TodoRepository
	todoListRepo domain.TodoListRepository // Used to check if the parent list exists
	aiService    domain.AIService          // Used for AI suggestions
}

func NewTodoUseCase(
	todoRepo domain.TodoRepository,
	todoListRepo domain.TodoListRepository,
	aiService domain.AIService,
) *TodoUseCase {
	if aiService == nil {
		log.Fatalf("TodoUseCase: aiService is nil when creating TodoUseCase!")
	}
	fmt.Printf("TodoUseCase: aiService injected (type: %T)\n", aiService)
	return &TodoUseCase{
		todoRepo:     todoRepo,
		todoListRepo: todoListRepo,
		aiService:    aiService,
	}
}

// CreateTodo creates a new Todo within a specific TodoList.
func (uc *TodoUseCase) CreateTodo(ctx context.Context, listID int, title, description string) (*domain.Todo, error) {

	fmt.Printf("TodoUseCase: CreateTodo called with title='%s', description='%s'\n", title, description) // BU SATIRI EKLEYİN
	log.Printf("TodoUseCase: description value (length %d): '%s'\n", len(description), description)      // Bu da faydalı olabilir

	if description == "" {
		// Yeni eklenecek satır
		fmt.Printf("TodoUseCase: Description is empty, attempting to generate AI description for title: '%s'\n", title)
		aiDescription, err := uc.aiService.GenerateDescription(ctx, title)
		if err != nil {
			// Yeni eklenecek satır
			fmt.Printf("TodoUseCase: Error generating AI description: %v\n", err)
			// AI üretimi başarısız olursa, şimdilik hatayı döndürüp gözlemleyelim.
			return nil, fmt.Errorf("failed to generate AI description: %w", err)
		}
		description = aiDescription
		// Yeni eklenecek satır
		fmt.Printf("TodoUseCase: AI generated description: '%s'\n", description)
	}

	listExists, err := uc.todoListRepo.TodoListExists(ctx, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to check todo list existence: %w", err)
	}
	if !listExists {
		return nil, fmt.Errorf("to-do list not found")
	}

	todo, err := uc.todoRepo.CreateTodo(ctx, listID, title, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create todo: %w", err)
	}
	return todo, nil
}

// GetTodosByListID retrieves all Todos for a given TodoList ID.
func (uc *TodoUseCase) GetTodosByListID(ctx context.Context, listID int) ([]domain.Todo, error) {
	// Check if the parent TodoList exists
	listExists, err := uc.todoListRepo.TodoListExists(ctx, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to check todo list existence: %w", err)
	}
	if !listExists {
		return nil, fmt.Errorf("to-do list not found")
	}

	todos, err := uc.todoRepo.GetTodosByListID(ctx, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve todos by list ID: %w", err)
	}
	return todos, nil
}

// GetTodoByID retrieves a single Todo by its ID within a specific list.
func (uc *TodoUseCase) GetTodoByID(ctx context.Context, listID, todoID int) (*domain.Todo, error) {
	// Check if the parent TodoList exists
	listExists, err := uc.todoListRepo.TodoListExists(ctx, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to check todo list existence: %w", err)
	}
	if !listExists {
		return nil, fmt.Errorf("to-do list not found")
	}

	todo, err := uc.todoRepo.GetTodoByID(ctx, listID, todoID)
	if err != nil {
		// Specific error for "not found" should be handled by the repository
		if err.Error() == "to-do item not found in this list" {
			return nil, err // Pass through specific not found error
		}
		return nil, fmt.Errorf("failed to retrieve todo by ID: %w", err)
	}
	return todo, nil
}

// UpdateTodo updates an existing Todo.
func (uc *TodoUseCase) UpdateTodo(ctx context.Context, listID, todoID int, title, description string, completed bool) (*domain.Todo, error) {
	// Check if the parent TodoList exists
	listExists, err := uc.todoListRepo.TodoListExists(ctx, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to check todo list existence: %w", err)
	}
	if !listExists {
		return nil, fmt.Errorf("to-do list not found")
	}

	// Get the existing todo to update its fields
	existingTodo, err := uc.todoRepo.GetTodoByID(ctx, listID, todoID)
	if err != nil {
		// This will cover "to-do item not found in this list"
		return nil, fmt.Errorf("failed to find existing todo: %w", err)
	}

	existingTodo.Title = title
	existingTodo.Description = description
	existingTodo.Completed = completed
	existingTodo.UpdatedAt = time.Now() // Update timestamp on modification

	err = uc.todoRepo.UpdateTodo(ctx, existingTodo)
	if err != nil {
		return nil, fmt.Errorf("failed to update todo: %w", err)
	}
	return existingTodo, nil // Return the updated todo object
}

// DeleteTodo deletes a Todo.
func (uc *TodoUseCase) DeleteTodo(ctx context.Context, listID, todoID int) error {
	// Check if the parent TodoList exists
	listExists, err := uc.todoListRepo.TodoListExists(ctx, listID)
	if err != nil {
		return fmt.Errorf("failed to check todo list existence: %w", err)
	}
	if !listExists {
		return fmt.Errorf("to-do list not found")
	}

	err = uc.todoRepo.DeleteTodo(ctx, listID, todoID)
	if err != nil {
		// This will cover "to-do item not found in this list"
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	return nil
}
