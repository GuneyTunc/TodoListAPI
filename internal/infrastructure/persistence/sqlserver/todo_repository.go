package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"todo-api/internal/domain" // domain.Todo ve domain.TodoRepository arayüzü
)

// SQLServerTodoRepository is an implementation of domain.TodoRepository for SQL Server.
type SQLServerTodoRepository struct {
	db *sql.DB
}

// NewSQLServerTodoRepository creates a new SQLServerTodoRepository.
func NewSQLServerTodoRepository(db *sql.DB) *SQLServerTodoRepository {
	return &SQLServerTodoRepository{db: db}
}

// CreateTodo implements domain.TodoRepository.
func (r *SQLServerTodoRepository) CreateTodo(ctx context.Context, listID int, title, description string) (*domain.Todo, error) {
	todo := &domain.Todo{
		ListID:      listID,
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO Todos (ListID, Title, Description, Completed, CreatedAt, UpdatedAt)
		OUTPUT INSERTED.ID, INSERTED.CreatedAt, INSERTED.UpdatedAt
		VALUES (@p1, @p2, @p3, @p4, @p5, @p6);
	`
	// Parametre isimleri veritabanı sürücüsüne göre değişebilir (@p1, @p2 veya ? )
	// go-mssqldb sürücüsü @pN formatını kullanır.
	err := r.db.QueryRowContext(ctx, query,
		sql.Named("p1", todo.ListID),
		sql.Named("p2", todo.Title),
		sql.Named("p3", todo.Description),
		sql.Named("p4", todo.Completed),
		sql.Named("p5", todo.CreatedAt),
		sql.Named("p6", todo.UpdatedAt),
	).Scan(&todo.ID, &todo.CreatedAt, &todo.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create todo in DB: %w", err)
	}

	return todo, nil
}

// GetTodosByListID implements domain.TodoRepository.
func (r *SQLServerTodoRepository) GetTodosByListID(ctx context.Context, listID int) ([]domain.Todo, error) {
	query := `
		SELECT ID, ListID, Title, Description, Completed, CreatedAt, UpdatedAt
		FROM Todos
		WHERE ListID = @p1;
	`
	rows, err := r.db.QueryContext(ctx, query, sql.Named("p1", listID))
	if err != nil {
		return nil, fmt.Errorf("failed to get todos by list ID from DB: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Printf("Error closing rows in GetTodosByListID: %v\n", cerr)
		}
	}()

	var todos []domain.Todo
	for rows.Next() {
		var todo domain.Todo
		err := rows.Scan(
			&todo.ID,
			&todo.ListID,
			&todo.Title,
			&todo.Description,
			&todo.Completed,
			&todo.CreatedAt,
			&todo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan todo row: %w", err)
		}
		todos = append(todos, todo)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating todo rows: %w", err)
	}

	return todos, nil
}

// GetTodoByID implements domain.TodoRepository.
func (r *SQLServerTodoRepository) GetTodoByID(ctx context.Context, listID, todoID int) (*domain.Todo, error) {
	query := `
		SELECT ID, ListID, Title, Description, Completed, CreatedAt, UpdatedAt
		FROM Todos
		WHERE ID = @p1 AND ListID = @p2;
	`
	var todo domain.Todo
	err := r.db.QueryRowContext(ctx, query,
		sql.Named("p1", todoID),
		sql.Named("p2", listID),
	).Scan(
		&todo.ID,
		&todo.ListID,
		&todo.Title,
		&todo.Description,
		&todo.Completed,
		&todo.CreatedAt,
		&todo.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("to-do item not found in this list")
		}
		return nil, fmt.Errorf("failed to get todo by ID from DB: %w", err)
	}
	return &todo, nil
}

// UpdateTodo implements domain.TodoRepository.
func (r *SQLServerTodoRepository) UpdateTodo(ctx context.Context, todo *domain.Todo) error {
	todo.UpdatedAt = time.Now() // Güncelleme zamanını otomatik olarak ayarla
	query := `
		UPDATE Todos
		SET Title = @p1, Description = @p2, Completed = @p3, UpdatedAt = @p4
		WHERE ID = @p5 AND ListID = @p6;
	`
	result, err := r.db.ExecContext(ctx, query,
		sql.Named("p1", todo.Title),
		sql.Named("p2", todo.Description),
		sql.Named("p3", todo.Completed),
		sql.Named("p4", todo.UpdatedAt),
		sql.Named("p5", todo.ID),
		sql.Named("p6", todo.ListID),
	)
	if err != nil {
		return fmt.Errorf("failed to update todo in DB: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected after todo update: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("to-do item not found or no changes made") // Hata mesajı domain'e uygun olmalı
	}
	return nil
}

// DeleteTodo implements domain.TodoRepository.
func (r *SQLServerTodoRepository) DeleteTodo(ctx context.Context, listID, todoID int) error {
	query := `
		DELETE FROM Todos
		WHERE ID = @p1 AND ListID = @p2;
	`
	result, err := r.db.ExecContext(ctx, query,
		sql.Named("p1", todoID),
		sql.Named("p2", listID),
	)
	if err != nil {
		return fmt.Errorf("failed to delete todo from DB: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected after todo delete: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("to-do item not found in this list") // Use specific error message
	}
	return nil
}

// TodoExists implements domain.TodoRepository.
func (r *SQLServerTodoRepository) TodoExists(ctx context.Context, listID, todoID int) (bool, error) {
	query := `
		SELECT COUNT(ID) FROM Todos
		WHERE ID = @p1 AND ListID = @p2;
	`
	var count int
	err := r.db.QueryRowContext(ctx, query,
		sql.Named("p1", todoID),
		sql.Named("p2", listID),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check todo existence in DB: %w", err)
	}
	return count > 0, nil
}
