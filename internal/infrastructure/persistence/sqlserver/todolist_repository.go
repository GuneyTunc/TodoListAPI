package sqlserver

import (
	"context"
	"database/sql"
	"fmt"

	"todo-api/internal/domain"
)

// SQLServerTodoListRepository is an implementation of domain.TodoListRepository for SQL Server.
type SQLServerTodoListRepository struct {
	db *sql.DB
}

// NewSQLServerTodoListRepository creates a new SQLServerTodoListRepository.
func NewSQLServerTodoListRepository(db *sql.DB) *SQLServerTodoListRepository {
	return &SQLServerTodoListRepository{db: db}
}

// CreateTodoList implements domain.TodoListRepository.
func (r *SQLServerTodoListRepository) CreateTodoList(ctx context.Context, title string) (*domain.TodoList, error) {
	todoList := &domain.TodoList{
		Title: title,
	}

	query := `
		INSERT INTO TodoLists (Title)
		OUTPUT INSERTED.ID
		VALUES (@p1);
	`
	err := r.db.QueryRowContext(ctx, query,
		sql.Named("p1", todoList.Title),
	).Scan(&todoList.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create todo list in DB: %w", err)
	}

	return todoList, nil
}

// GetTodoLists implements domain.TodoListRepository.
func (r *SQLServerTodoListRepository) GetTodoLists(ctx context.Context) ([]domain.TodoList, error) {
	query := `
		SELECT ID, Title -- CreatedAt, UpdatedAt kaldırıldı
		FROM TodoLists;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo lists from DB: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Printf("Error closing rows in GetTodoLists: %v\n", cerr)
		}
	}()

	var todoLists []domain.TodoList
	for rows.Next() {
		var todoList domain.TodoList
		err := rows.Scan(
			&todoList.ID,
			&todoList.Title,
			// &todoList.CreatedAt, // Bu satırın kaldırıldığından emin olun
			// &todoList.UpdatedAt, // Bu satırın kaldırıldığından emin olun
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan todo list row: %w", err)
		}
		todoLists = append(todoLists, todoList)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating todo list rows: %w", err)
	}

	return todoLists, nil
}

// GetTodoListByID implements domain.TodoListRepository.
func (r *SQLServerTodoListRepository) GetTodoListByID(ctx context.Context, id int) (*domain.TodoList, error) {
	query := `
		SELECT ID, Title
		FROM TodoLists
		WHERE ID = @p1;
	`
	var todoList domain.TodoList
	err := r.db.QueryRowContext(ctx, query, sql.Named("p1", id)).Scan(
		&todoList.ID,
		&todoList.Title,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("to-do list not found")
		}
		return nil, fmt.Errorf("failed to get todo list by ID from DB: %w", err)
	}
	return &todoList, nil
}

// UpdateTodoList implements domain.TodoListRepository.
func (r *SQLServerTodoListRepository) UpdateTodoList(ctx context.Context, id int, title string) (*domain.TodoList, error) {
	query := `
		UPDATE TodoLists
		SET Title = @p1
		WHERE ID = @p2;
	`
	result, err := r.db.ExecContext(ctx, query,
		sql.Named("p1", title),
		sql.Named("p2", id),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update todo list in DB: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected after todo list update: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("to-do list not found or no changes made")
	}

	updatedTodoList, err := r.GetTodoListByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated todo list: %w", err)
	}
	return updatedTodoList, nil
}

// DeleteTodoList implements domain.TodoListRepository.
func (r *SQLServerTodoListRepository) DeleteTodoList(ctx context.Context, id int) error {
	query := `
		DELETE FROM TodoLists
		WHERE ID = @p1;
	`
	result, err := r.db.ExecContext(ctx, query, sql.Named("p1", id))
	if err != nil {
		return fmt.Errorf("failed to delete todo list from DB: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected after todo list delete: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("to-do list not found")
	}
	return nil
}

// TodoListExists implements domain.TodoListRepository.
func (r *SQLServerTodoListRepository) TodoListExists(ctx context.Context, id int) (bool, error) {
	query := `
		SELECT COUNT(ID) FROM TodoLists
		WHERE ID = @p1;
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, sql.Named("p1", id)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check todo list existence in DB: %w", err)
	}
	return count > 0, nil
}
