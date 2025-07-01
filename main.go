package main

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Todo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

var (
	todos    = make(map[int]Todo)
	nextID   = 1
	todosMux sync.Mutex // Concurrency kontrolü için mutex
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())  //Log tutucu
	e.Use(middleware.Recover()) //Crash onleyici

	e.GET("/todos", getTodos)
	e.GET("/todos/:id", getTodoByID)
	e.POST("/todos", createTodo)
	e.PUT("/todos/:id", updateTodo)
	e.DELETE("/todos/:id", deleteTodo)

	e.Logger.Fatal(e.Start(":8080"))
}

func getTodos(c echo.Context) error {
	todosMux.Lock()
	defer todosMux.Unlock()

	todoList := make([]Todo, 0, len(todos)) //Todo structını içeren, ilk başta 0 elemanı olan ve todos mapının uzunluğu kadar elemanı olan bir slice oluşturur
	for _, todo := range todos {
		todoList = append(todoList, todo)
	}
	return c.JSON(http.StatusOK, todoList)
}

func getTodoByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id")) //id alınır
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID format")
	}

	todosMux.Lock() //Id doğru formattaysa mutex kilitlenir
	defer todosMux.Unlock()

	todo, found := todos[id]
	if !found { //Eger id bulunamazsa item not found hatası döner
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found")
	}
	return c.JSON(http.StatusOK, todo)
}

func createTodo(c echo.Context) error {
	todo := new(Todo)
	if err := c.Bind(todo); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if todo.Title == "" { //Title boşsa title required hatası döner
		return echo.NewHTTPError(http.StatusBadRequest, "Title is required")
	}

	todosMux.Lock()
	defer todosMux.Unlock()

	todo.ID = nextID //Hata yoksa Id atanır
	nextID++
	todos[todo.ID] = *todo

	return c.JSON(http.StatusCreated, todo)
}

func updateTodo(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID format")
	}

	updatedTodo := new(Todo)
	if err := c.Bind(updatedTodo); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if updatedTodo.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Title is required")
	}

	todosMux.Lock()
	defer todosMux.Unlock()

	_, found := todos[id] //Id bulunamazsa item not found hatası döner
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found")
	}

	updatedTodo.ID = id      //Id bulunursa yeni id atanır
	todos[id] = *updatedTodo //todo güncellenir

	return c.JSON(http.StatusOK, updatedTodo)
}

func deleteTodo(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID format")
	}

	todosMux.Lock()
	defer todosMux.Unlock()

	_, found := todos[id]
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found")
	}

	delete(todos, id)
	return c.NoContent(http.StatusNoContent)
}
