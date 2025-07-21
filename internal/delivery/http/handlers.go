package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"todo-api/internal/application" // Uygulama katmanındaki use case'ler
	// Domain modelleri
	"github.com/labstack/echo/v4"
)

// Request ve Response modelleri
// Bunlar, API'den gelen ve API'ye gönderilen verilerin yapılarını tanımlar.

// CreateTodoListRequest defines the request body for creating a todo list.
type CreateTodoListRequest struct {
	Title string `json:"title" validate:"required"`
}

// UpdateTodoListRequest defines the request body for updating a todo list.
type UpdateTodoListRequest struct {
	Title string `json:"title" validate:"required"`
}

// CreateTodoRequest defines the request body for creating a todo item.
type CreateTodoRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
}

// UpdateTodoRequest defines the request body for updating a todo item.
type UpdateTodoRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// SuggestTodoDescriptionRequest defines the request body for AI description suggestion.
type SuggestTodoDescriptionRequest struct {
	Title string `json:"title" validate:"required"`
}

// HTTPHandler struct holds references to the application's use cases.
type HTTPHandler struct {
	todoListUseCase     *application.TodoListUseCase
	todoUseCase         *application.TodoUseCase
	aiSuggestionUseCase *application.AISuggestionUseCase
}

// NewHTTPHandler creates a new HTTPHandler instance.
func NewHTTPHandler(
	todoListUC *application.TodoListUseCase,
	todoUC *application.TodoUseCase,
	aiSuggestUC *application.AISuggestionUseCase,
) *HTTPHandler {
	return &HTTPHandler{
		todoListUseCase:     todoListUC,
		todoUseCase:         todoUC,
		aiSuggestionUseCase: aiSuggestUC,
	}
}

// @Summary Create a new TodoList
// @Description Creates a new to-do list with the provided title.
// @Tags TodoLists
// @Accept json
// @Produce json
// @Param request body CreateTodoListRequest true "Todo List creation details"
// @Success 201 {object} domain.TodoList "Successfully created todo list"
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists [post]
func (h *HTTPHandler) CreateTodoList(c echo.Context) error {
	req := new(CreateTodoListRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	todoList, err := h.todoListUseCase.CreateTodoList(ctx, req.Title)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, todoList)
}

// @Summary Get all TodoLists
// @Description Retrieves a list of all existing to-do lists.
// @Tags TodoLists
// @Produce json
// @Success 200 {array} domain.TodoList "Successfully retrieved todo lists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists [get]
func (h *HTTPHandler) GetTodoLists(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	todoLists, err := h.todoListUseCase.GetTodoLists(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, todoLists)
}

// @Summary Get TodoList by ID
// @Description Retrieves a single to-do list by its unique ID.
// @Tags TodoLists
// @Produce json
// @Param id path int true "Todo List ID"
// @Success 200 {object} domain.TodoList "Successfully retrieved todo list"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Todo List not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists/{id} [get]
func (h *HTTPHandler) GetTodoListByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid todo list ID"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	todoList, err := h.todoListUseCase.GetTodoListByID(ctx, id)
	if err != nil {
		if err.Error() == "to-do list not found" { // Use the exact error message from use case/repo
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, todoList)
}

// @Summary Update an existing TodoList
// @Description Updates the title of an existing to-do list.
// @Tags TodoLists
// @Accept json
// @Produce json
// @Param id path int true "Todo List ID"
// @Param request body UpdateTodoListRequest true "Todo List update details"
// @Success 200 {object} domain.TodoList "Successfully updated todo list"
// @Failure 400 {object} map[string]string "Invalid request payload or ID format"
// @Failure 404 {object} map[string]string "Todo List not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists/{id} [put]
func (h *HTTPHandler) UpdateTodoList(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid todo list ID"})
	}

	req := new(UpdateTodoListRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	updatedTodoList, err := h.todoListUseCase.UpdateTodoList(ctx, id, req.Title)
	if err != nil {
		if err.Error() == "to-do list not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, updatedTodoList)
}

// @Summary Delete a TodoList
// @Description Deletes a to-do list by its ID. Associated to-do items will also be deleted.
// @Tags TodoLists
// @Produce json
// @Param id path int true "Todo List ID"
// @Success 204 "Successfully deleted todo list"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Todo List not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists/{id} [delete]
func (h *HTTPHandler) DeleteTodoList(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid todo list ID"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	err = h.todoListUseCase.DeleteTodoList(ctx, id)
	if err != nil {
		if err.Error() == "to-do list not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent) // 204 No Content
}

// --- Todo Rotaları ---

// @Summary Create a new Todo in a TodoList
// @Description Creates a new to-do item within a specific to-do list.
// @Tags Todos
// @Accept json
// @Produce json
// @Param listID path int true "Todo List ID"
// @Param request body CreateTodoRequest true "Todo item creation details"
// @Success 201 {object} domain.Todo "Successfully created todo item"
// @Failure 400 {object} map[string]string "Invalid request payload or list ID format"
// @Failure 404 {object} map[string]string "Todo List not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists/{listID}/todos [post]
func (h *HTTPHandler) CreateTodoInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid list ID"})
	}

	req := new(CreateTodoRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	todo, err := h.todoUseCase.CreateTodo(ctx, listID, req.Title, req.Description)
	if err != nil {
		if err.Error() == "to-do list not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, todo)
}

// @Summary Get all Todos in a TodoList
// @Description Retrieves all to-do items belonging to a specific to-do list.
// @Tags Todos
// @Produce json
// @Param listID path int true "Todo List ID"
// @Success 200 {array} domain.Todo "Successfully retrieved todos"
// @Failure 400 {object} map[string]string "Invalid list ID format"
// @Failure 404 {object} map[string]string "Todo List not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists/{listID}/todos [get]
func (h *HTTPHandler) GetTodosInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid list ID"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	todos, err := h.todoUseCase.GetTodosByListID(ctx, listID)
	if err != nil {
		if err.Error() == "to-do list not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, todos)
}

// @Summary Get a Todo by ID in a TodoList
// @Description Retrieves a specific to-do item by its ID within a given to-do list.
// @Tags Todos
// @Produce json
// @Param listID path int true "Todo List ID"
// @Param todoID path int true "Todo ID"
// @Success 200 {object} domain.Todo "Successfully retrieved todo item"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Todo List or Todo not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists/{listID}/todos/{todoID} [get]
func (h *HTTPHandler) GetTodoInListByID(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid list ID"})
	}
	todoID, err := strconv.Atoi(c.Param("todoID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid todo ID"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	todo, err := h.todoUseCase.GetTodoByID(ctx, listID, todoID)
	if err != nil {
		if err.Error() == "to-do list not found" || err.Error() == "to-do item not found in this list" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, todo)
}

// @Summary Update an existing Todo in a TodoList
// @Description Updates an existing to-do item within a specific to-do list.
// @Tags Todos
// @Accept json
// @Produce json
// @Param listID path int true "Todo List ID"
// @Param todoID path int true "Todo ID"
// @Param request body UpdateTodoRequest true "Todo item update details"
// @Success 200 {object} domain.Todo "Successfully updated todo item"
// @Failure 400 {object} map[string]string "Invalid request payload or ID format"
// @Failure 404 {object} map[string]string "Todo List or Todo not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists/{listID}/todos/{todoID} [put]
func (h *HTTPHandler) UpdateTodoInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid list ID"})
	}
	todoID, err := strconv.Atoi(c.Param("todoID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid todo ID"})
	}

	req := new(UpdateTodoRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	updatedTodo, err := h.todoUseCase.UpdateTodo(ctx, listID, todoID, req.Title, req.Description, req.Completed)
	if err != nil {
		if err.Error() == "to-do list not found" || err.Error() == "to-do item not found in this list" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, updatedTodo)
}

// @Summary Delete a Todo in a TodoList
// @Description Deletes a specific to-do item by its ID within a given to-do list.
// @Tags Todos
// @Produce json
// @Param listID path int true "Todo List ID"
// @Param todoID path int true "Todo ID"
// @Success 204 "Successfully deleted todo item"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Todo List or Todo not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todolists/{listID}/todos/{todoID} [delete]
func (h *HTTPHandler) DeleteTodoInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid list ID"})
	}
	todoID, err := strconv.Atoi(c.Param("todoID"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid todo ID"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	err = h.todoUseCase.DeleteTodo(ctx, listID, todoID)
	if err != nil {
		if err.Error() == "to-do list not found" || err.Error() == "to-do item not found in this list" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent) // 204 No Content
}

// --- AI Rotaları ---

// @Summary Suggest a Todo description using AI
// @Description Generates an AI-powered description suggestion for a given todo title.
// @Tags AI
// @Accept json
// @Produce json
// @Param request body SuggestTodoDescriptionRequest true "Todo title for suggestion"
// @Success 200 {object} map[string]string "Successfully generated suggestion"
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /todos/suggest-description [post]
func (h *HTTPHandler) SuggestTodoDescription(c echo.Context) error {
	req := new(SuggestTodoDescriptionRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second) // AI çağrıları daha uzun sürebilir
	defer cancel()

	suggestion, err := h.aiSuggestionUseCase.SuggestTodoDescription(ctx, req.Title)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"suggestion": suggestion})
}
