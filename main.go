package main

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Todo struct { //Görevlerin yapısı
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type TodoList struct { //Listelerin yapısı
	ID               int          `json:"id"`
	Title            string       `json:"title"`
	Todos            map[int]Todo `json:"todos"` // Her liste için ayrı bir todo seti
	nextTodoIDInList int          // Bu liste için benzersiz ID'ler oluşturmak için
}

var (
	// Bireysel görevler için kullanılan global değişkenler artık gerekli değil ve kaldırıldı.
	// todos      = make(map[int]Todo)
	// nextTodoID = 1
	// todosMux   sync.Mutex

	todoLists      = make(map[int]TodoList)
	nextTodoListID = 1
	todoListsMux   sync.Mutex // Todo listeleri için Mutex
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/todolists", createTodoList)       //Yeni liste oluşturur
	e.GET("/todolists", getTodoLists)          //Tüm listeleri getirir
	e.GET("/todolists/:id", getTodoListByID)   //Listeyi id ile getirir
	e.PUT("/todolists/:id", updateTodoList)    //Listeyi günceller
	e.DELETE("/todolists/:id", deleteTodoList) //Listeyi siler

	e.POST("/todolists/:listID/todos", createTodoInList)           // İstenen listeye görev ekleme
	e.GET("/todolists/:listID/todos", getTodosInList)              // İstenen listeye ait tüm görevleri getirir
	e.GET("/todolists/:listID/todos/:todoID", getTodoInListByID)   // İstenen listeye ait görevleri id ile getirir
	e.PUT("/todolists/:listID/todos/:todoID", updateTodoInList)    // İstenen listeye ait istenen görevleri günceller
	e.DELETE("/todolists/:listID/todos/:todoID", deleteTodoInList) // İstenen listeye ait istenen görevleri siler

	e.Logger.Fatal(e.Start(":8080"))
}

func createTodoList(c echo.Context) error {
	todoList := new(TodoList)
	if err := c.Bind(todoList); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if todoList.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Title for todo list is required")
	}

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	todoList.ID = nextTodoListID
	todoList.Todos = make(map[int]Todo)
	todoList.nextTodoIDInList = 1
	nextTodoListID++
	todoLists[todoList.ID] = *todoList

	return c.JSON(http.StatusCreated, todoList)
}

func getTodoLists(c echo.Context) error {
	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	list := make([]TodoList, 0, len(todoLists))
	for _, tl := range todoLists {
		list = append(list, tl)
	}
	return c.JSON(http.StatusOK, list)
}

func getTodoListByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID format")
	}

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	todoList, found := todoLists[id]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}
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

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	existingTodoList, found := todoLists[id]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	existingTodoList.Title = updatedTodoList.Title

	todoLists[id] = existingTodoList

	return c.JSON(http.StatusOK, existingTodoList)
}

func deleteTodoList(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID format")
	}

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	_, found := todoLists[id]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	delete(todoLists, id)
	return c.NoContent(http.StatusNoContent)
}

func createTodoInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid List ID format")
	}

	newTodo := new(Todo)
	if err := c.Bind(newTodo); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if newTodo.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Todo title is required")
	}

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	list, found := todoLists[listID]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	newTodo.ID = list.nextTodoIDInList
	list.nextTodoIDInList++

	list.Todos[newTodo.ID] = *newTodo

	todoLists[listID] = list

	return c.JSON(http.StatusCreated, newTodo)
}

func getTodosInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid List ID format")
	}

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	list, found := todoLists[listID]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	todosInList := make([]Todo, 0, len(list.Todos))
	for _, todo := range list.Todos {
		todosInList = append(todosInList, todo)
	}

	return c.JSON(http.StatusOK, todosInList)
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

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	list, found := todoLists[listID]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	todo, found := list.Todos[todoID]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list")
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

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	list, found := todoLists[listID]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	_, found = list.Todos[todoID]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list")
	}

	updatedTodo.ID = todoID
	list.Todos[todoID] = *updatedTodo

	todoLists[listID] = list

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

	todoListsMux.Lock()
	defer todoListsMux.Unlock()

	list, found := todoLists[listID]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	_, found = list.Todos[todoID]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list")
	}

	delete(list.Todos, todoID)

	todoLists[listID] = list

	return c.NoContent(http.StatusNoContent)
}
