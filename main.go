package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	openai "github.com/sashabaranov/go-openai"
)

// Config holds application configuration.
type Config struct {
	DbServer             string `envconfig:"DB_SERVER" required:"true"`
	DbName               string `envconfig:"DB_NAME"   required:"true"`
	DbIntegratedSecurity bool   `envconfig:"DB_INTEGRATED_SECURITY" default:"false"`
	DbUser               string `envconfig:"DB_USER"`
	DbPassword           string `envconfig:"DB_PASSWORD"`
	AppPort              string `envconfig:"APP_PORT"  default:"8080"`
	OpenAIApiKey         string `envconfig:"OPENAI_API_KEY" required:"true"`
}

type Todo struct {
	ID          int    `json:"id"`
	ListID      int    `json:"listId"` // 'list_id' yerine 'ListID' olarak düzeltildi (JSON tag'i)
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type TodoList struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Todos []Todo `json:"todos,omitempty"`
}

var db *sql.DB
var openaiClient *openai.Client

func main() {
	var cfg Config
	// `envconfig` kütüphanesi, Docker Compose tarafından yüklenen
	// ortam değişkenlerini doğrudan okuyacaktır.
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("Error processing environment variables: %v", err)
	}

	openaiClient = openai.NewClient(cfg.OpenAIApiKey)

	connString := fmt.Sprintf("server=%s;database=%s;", cfg.DbServer, cfg.DbName)

	if cfg.DbIntegratedSecurity {
		connString += "integrated security=true;"
	} else {
		if cfg.DbUser != "" && cfg.DbPassword != "" {
			connString += fmt.Sprintf("user id=%s;password=%s;", cfg.DbUser, cfg.DbPassword)
		} else {
			log.Fatalf("Error: DB_USER and DB_PASSWORD are required when DB_INTEGRATED_SECURITY is false.")
		}
	}

	maxRetries := 10
	retryInterval := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		log.Printf("Attempting to connect to database (Attempt %d/%d)...", i+1, maxRetries)
		db, err = sql.Open("sqlserver", connString)
		if err != nil {
			log.Printf("Error opening database connection: %v. Retrying in %s...", err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		err = db.Ping()
		if err != nil {
			db.Close()
			log.Printf("Database connection test failed: %v. Retrying in %s...", err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		fmt.Println("SQL Server database connection successful!")
		break
	}

	if db == nil || db.Ping() != nil {
		log.Fatalf("Failed to connect to database after %d retries. Exiting.", maxRetries)
	}
	defer db.Close()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/todolists", createTodoList)
	e.GET("/todolists", getTodoLists)
	e.GET("/todolists/:id", getTodoListByID)
	e.PUT("/todolists/:id", updateTodoList)
	e.DELETE("/todolists/:id", deleteTodoList)

	e.POST("/todolists/:listID/todos", createTodoInList)
	e.GET("/todolists/:listID/todos", getTodosInList)
	e.GET("/todolists/:listID/todos/:todoID", getTodoInListByID)
	e.PUT("/todolists/:listID/todos/:todoID", updateTodoInList)
	e.DELETE("/todolists/:listID/todos/:todoID", deleteTodoInList)

	e.POST("/todos/suggest-description", suggestTodoDescription)

	fmt.Printf("Server %s port is listening...\n", cfg.AppPort)
	e.Logger.Fatal(e.Start(":" + cfg.AppPort))
}

func createTodoList(c echo.Context) error {
	todoList := new(TodoList)
	if err := c.Bind(todoList); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if todoList.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Title for todo list is required")
	}

	var lastInsertID int
	// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
	err := db.QueryRow("INSERT INTO TodoLists (Title) OUTPUT INSERTED.ID VALUES (@p1)", todoList.Title).Scan(&lastInsertID)
	if err != nil {
		if sqlErr, ok := err.(*mssql.Error); ok {
			if sqlErr.Number == 2627 || sqlErr.Number == 2601 {
				return echo.NewHTTPError(http.StatusConflict, "A todo list with this title already exists")
			}
		}
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create todo list: %v", err))
	}

	todoList.ID = lastInsertID
	todoList.Todos = []Todo{}

	return c.JSON(http.StatusCreated, todoList)
}

func getTodoLists(c echo.Context) error {
	// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
	rows, err := db.Query("SELECT ID, Title FROM TodoLists")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get todo lists: %v", err))
	}
	defer rows.Close()

	var todoLists []TodoList
	for rows.Next() {
		var tl TodoList
		if err := rows.Scan(&tl.ID, &tl.Title); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to scan todo list: %v", err))
		}
		todoLists = append(todoLists, tl)
	}

	return c.JSON(http.StatusOK, todoLists)
}

func getTodoListByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID format")
	}

	var todoList TodoList
	// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
	err = db.QueryRow("SELECT ID, Title FROM TodoLists WHERE ID = @p1", id).Scan(&todoList.ID, &todoList.Title)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get todo list: %v", err))
	}

	todosInList, err := getTodosForList(todoList.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get todos for list: %v", err))
	}
	todoList.Todos = todosInList

	return c.JSON(http.StatusOK, todoList)
}

func updateTodoList(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID format")
	}

	updatedTodoList := new(TodoList)
	if err := c.Bind(updatedTodoList); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if updatedTodoList.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Title for todo list is required")
	}

	// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
	res, err := db.Exec("UPDATE TodoLists SET Title = @p1 WHERE ID = @p2", updatedTodoList.Title, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update todo list: %v", err))
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get rows affected: %v", err))
	}
	if rowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	updatedTodoList.ID = id
	return c.JSON(http.StatusOK, updatedTodoList)
}

func deleteTodoList(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID format")
	}

	// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
	res, err := db.Exec("DELETE FROM TodoLists WHERE ID = @p1", id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete todo list: %v", err))
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get rows affected: %v", err))
	}
	if rowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	return c.NoContent(http.StatusNoContent)
}

// --- To-Do (Task) Handler Functions ---

func createTodoInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid List ID format")
	}

	newTodo := new(Todo)
	if err := c.Bind(newTodo); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	log.Printf("Received new task request: Title='%s', Description='%s'", newTodo.Title, newTodo.Description)

	if newTodo.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Todo title is required")
	}

	var exists int
	// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
	err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM TodoLists WHERE ID = @p1), 1, 0)", listID).Scan(&exists)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check list existence: %v", err))
	}
	if exists == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	if newTodo.Description == "" {
		log.Println("Description is empty, attempting to generate with AI.")
		suggestedDesc, err := generateDescriptionWithOpenAI(newTodo.Title)
		if err != nil {
			log.Printf("Failed to generate description with OpenAI: %v", err)
		} else {
			newTodo.Description = suggestedDesc
			log.Printf("AI generated description: '%s'", newTodo.Description)
		}
	} else {
		log.Println("Description provided in request, skipping AI generation.")
	}

	completedBit := 0
	if newTodo.Completed {
		completedBit = 1
	}

	var lastInsertID int
	// DİKKAT: Tablo adı 'Todos' ve sütun adı 'ListID' olarak düzeltildi
	err = db.QueryRow(
		"INSERT INTO Todos (ListID, Title, Description, Completed) OUTPUT INSERTED.ID VALUES (@p1, @p2, @p3, @p4)",
		listID, newTodo.Title, newTodo.Description, completedBit,
	).Scan(&lastInsertID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create todo in list: %v", err))
	}

	newTodo.ID = lastInsertID
	newTodo.ListID = listID

	return c.JSON(http.StatusCreated, newTodo)
}

// getTodosInList retrieves all todos for a specific list.
func getTodosInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid List ID format")
	}

	todosInList, err := getTodosForList(listID)
	if err != nil {
		if err.Error() == "To-Do list not found" {
			return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get todos for list: %v", err))
	}

	return c.JSON(http.StatusOK, todosInList)
}

// getTodosForList, a helper function to get all todos for a specific list from database
func getTodosForList(listID int) ([]Todo, error) {
	var listExists int
	// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
	err := db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM TodoLists WHERE ID = @p1), 1, 0)", listID).Scan(&listExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check list existence: %w", err)
	}
	if listExists == 0 {
		return nil, fmt.Errorf("To-Do list not found")
	}

	// DİKKAT: Tablo adı 'Todos' ve sütun adı 'ListID' olarak düzeltildi
	rows, err := db.Query("SELECT ID, ListID, Title, Description, Completed FROM Todos WHERE ListID = @p1", listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get todos from database: %w", err)
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.ListID, &todo.Title, &todo.Description, &todo.Completed); err != nil {
			return nil, fmt.Errorf("failed to scan todo: %w", err)
		}
		todos = append(todos, todo)
	}
	return todos, nil
}

func getTodoInListByID(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid List ID format")
	}
	todoID, err := strconv.Atoi(c.Param("todoID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid Todo ID format")
	}

	var todo Todo
	// DİKKAT: Tablo adı 'Todos' ve sütun adı 'ListID' olarak düzeltildi
	err = db.QueryRow("SELECT ID, ListID, Title, Description, Completed FROM Todos WHERE ListID = @p1 AND ID = @p2", listID, todoID).
		Scan(&todo.ID, &todo.ListID, &todo.Title, &todo.Description, &todo.Completed)
	if err == sql.ErrNoRows {
		var listExists int
		// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
		err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM TodoLists WHERE ID = @p1), 1, 0)", listID).Scan(&listExists)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check list existence: %v", err))
		}
		if listExists == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
		}
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get todo from database: %v", err))
	}

	return c.JSON(http.StatusOK, todo)
}

func updateTodoInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid List ID format")
	}
	todoID, err := strconv.Atoi(c.Param("todoID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid Todo ID format")
	}

	updatedTodo := new(Todo)
	if err := c.Bind(updatedTodo); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if updatedTodo.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Todo title is required")
	}

	var todoExists int
	// DİKKAT: Tablo adı 'Todos' ve sütun adı 'ListID' olarak düzeltildi
	err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM Todos WHERE ListID = @p1 AND ID = @p2), 1, 0)", listID, todoID).Scan(&todoExists)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check todo existence: %v", err))
	}
	if todoExists == 0 {
		var listExists int
		// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
		err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM TodoLists WHERE ID = @p1), 1, 0)", listID).Scan(&listExists)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check list existence: %v", err))
		}
		if listExists == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
		}
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list")
	}

	completedBit := 0
	if updatedTodo.Completed {
		completedBit = 1
	}

	// DİKKAT: Tablo adı 'Todos' ve sütun adı 'ListID' olarak düzeltildi
	res, err := db.Exec(
		"UPDATE Todos SET Title = @p1, Description = @p2, Completed = @p3 WHERE ListID = @p4 AND ID = @p5",
		updatedTodo.Title, updatedTodo.Description, completedBit, listID, todoID,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update todo: %v", err))
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get rows affected: %v", err))
	}
	if rowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list (or no changes made)")
	}

	updatedTodo.ID = todoID
	updatedTodo.ListID = listID
	return c.JSON(http.StatusOK, updatedTodo)
}

func deleteTodoInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid List ID format")
	}
	todoID, err := strconv.Atoi(c.Param("todoID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid Todo ID format")
	}

	var todoExists int
	// DİKKAT: Tablo adı 'Todos' ve sütun adı 'ListID' olarak düzeltildi
	err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM Todos WHERE ListID = @p1 AND ID = @p2), 1, 0)", listID, todoID).Scan(&todoExists)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check todo existence: %v", err))
	}
	if todoExists == 0 {
		var listExists int
		// DİKKAT: Tablo adı 'TodoLists' olarak düzeltildi
		err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM TodoLists WHERE ID = @p1), 1, 0)", listID).Scan(&listExists)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check list existence: %v", err))
		}
		if listExists == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
		}
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list")
	}

	// DİKKAT: Tablo adı 'Todos' ve sütun adı 'ListID' olarak düzeltildi
	res, err := db.Exec("DELETE FROM Todos WHERE ListID = @p1 AND ID = @p2", listID, todoID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete todo: %v", err))
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get rows affected: %v", err))
	}
	if rowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list")
	}

	return c.NoContent(http.StatusNoContent)
}

type SuggestionRequest struct {
	Title string `json:"title"`
}

type SuggestionResponse struct {
	SuggestedDescription string `json:"suggestedDescription"`
}

func suggestTodoDescription(c echo.Context) error {
	req := new(SuggestionRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Todo title is required for suggestion")
	}

	suggestedDesc, err := generateDescriptionWithOpenAI(req.Title)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("OpenAI API error: %v", err))
	}

	return c.JSON(http.StatusOK, SuggestionResponse{SuggestedDescription: suggestedDesc})
}

func generateDescriptionWithOpenAI(title string) (string, error) {
	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant that suggests concise and relevant descriptions for to-do list items. Provide only the description, without any introductory phrases.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Generate a concise description for a to-do item with the title: \"%s\".", title),
				},
			},
			MaxTokens:   50,
			Temperature: 0.7,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no suggestion found from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
