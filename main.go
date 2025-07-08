package main

import (
	"database/sql"
	"fmt"
	"log" // log paketi eklendi
	"net/http"
	"os" // os paketi eklendi
	"strconv"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/joho/godotenv" // godotenv kütüphanesi eklendi
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Todo struct {
	ID          int    `json:"id"`
	ListID      int    `json:"listId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type TodoList struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Todos []Todo `json:"todos,omitempty"`
}

// Global veritabanı bağlantı nesnesi
var db *sql.DB

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// .env dosyasından bağlantı bilgilerini al
	dbServer := os.Getenv("DB_SERVER")
	dbName := os.Getenv("DB_NAME")
	dbIntegratedSecurity := os.Getenv("DB_INTEGRATED_SECURITY")
	appPort := os.Getenv("APP_PORT")

	connString := fmt.Sprintf("server=%s;database=%s;", dbServer, dbName)

	if dbIntegratedSecurity == "true" {
		connString += "integrated security=true"
	} /*else {
		// Kullanıcı adı ve şifre varsa ekle
		if dbUser != "" && dbPassword != "" {
			connString += fmt.Sprintf("user id=%s;password=%s;", dbUser, dbPassword)
		}
	}*/

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		panic(fmt.Sprintf("Veritabanına bağlanılamadı: %v", err))
	}
	defer db.Close() // Uygulama kapanırken veritabanı bağlantısını kapat

	// Veritabanı bağlantısının açık olup olmadığını test et
	err = db.Ping()
	if err != nil {
		panic(fmt.Sprintf("Veritabanı bağlantısı testi başarısız: %v", err))
	}
	fmt.Println("SQL Server veritabanı bağlantısı başarılı!")

	// Tabloları oluştur (eğer yoksa)
	createTables()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// TodoList rotaları
	e.POST("/todolists", createTodoList)
	e.GET("/todolists", getTodoLists)
	e.GET("/todolists/:id", getTodoListByID)
	e.PUT("/todolists/:id", updateTodoList)
	e.DELETE("/todolists/:id", deleteTodoList)

	// Belirli bir TodoList içindeki Todo rotaları
	e.POST("/todolists/:listID/todos", createTodoInList)
	e.GET("/todolists/:listID/todos", getTodosInList)
	e.GET("/todolists/:listID/todos/:todoID", getTodoInListByID)
	e.PUT("/todolists/:listID/todos/:todoID", updateTodoInList)
	e.DELETE("/todolists/:listID/todos/:todoID", deleteTodoInList)

	fmt.Printf("Sunucu %s portunda dinliyor...\n", appPort)
	e.Logger.Fatal(e.Start(":" + appPort))
}

// createTables veritabanı tablolarını oluşturur (eğer yoksa).
func createTables() {
	// SQL Server için ID alanını IDENTITY(1,1) olarak ayarla
	// BOOLEAN yerine BIT kullan
	createTodoListsTableSQL := `
	IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='todolists' and xtype='U')
	CREATE TABLE todolists (
		id INT IDENTITY(1,1) PRIMARY KEY,
		title NVARCHAR(255) NOT NULL UNIQUE
	);`

	createTodosTableSQL := `
	IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='todos' and xtype='U')
	CREATE TABLE todos (
		id INT IDENTITY(1,1) PRIMARY KEY,
		list_id INT NOT NULL,
		title NVARCHAR(255) NOT NULL,
		description NVARCHAR(MAX),
		completed BIT NOT NULL DEFAULT 0,
		CONSTRAINT FK_TodoList FOREIGN KEY (list_id) REFERENCES todolists (id) ON DELETE CASCADE
	);`

	// Transaction başlatılır
	tx, err := db.Begin()
	if err != nil {
		panic(fmt.Sprintf("Transaction başlatılamadı: %v", err))
	}
	defer tx.Rollback() // Hata durumunda rollback

	_, err = tx.Exec(createTodoListsTableSQL)
	if err != nil {
		panic(fmt.Sprintf("todolists tablosu oluşturulurken hata: %v", err))
	}

	_, err = tx.Exec(createTodosTableSQL)
	if err != nil {
		panic(fmt.Sprintf("todos tablosu oluşturulurken hata: %v", err))
	}

	err = tx.Commit() // İşlem başarılıysa commit et
	if err != nil {
		panic(fmt.Sprintf("Transaction commit edilirken hata: %v", err))
	}
	fmt.Println("Veritabanı tabloları başarıyla oluşturuldu veya zaten mevcut.")
}

func createTodoList(c echo.Context) error {
	todoList := new(TodoList)
	if err := c.Bind(todoList); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if todoList.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Title for todo list is required")
	}

	// SQL Server'da IDENTITY değeri almak için INSERT INTO ... OUTPUT INSERTED.ID kullanılır
	var lastInsertID int
	err := db.QueryRow("INSERT INTO todolists (title) OUTPUT INSERTED.id VALUES (@p1)", todoList.Title).Scan(&lastInsertID)
	if err != nil {
		if sqlErr, ok := err.(*mssql.Error); ok {
			if sqlErr.Number == 2627 || sqlErr.Number == 2601 { // SQL Server duplicate key error codes
				return echo.NewHTTPError(http.StatusConflict, "A todo list with this title already exists")
			}
		}
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create todo list: %v", err))
	}

	todoList.ID = lastInsertID
	todoList.Todos = []Todo{} // Yeni liste oluşturulduğunda görevleri boş döndür

	return c.JSON(http.StatusCreated, todoList)
}

func getTodoLists(c echo.Context) error {
	rows, err := db.Query("SELECT id, title FROM todolists")
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
	err = db.QueryRow("SELECT id, title FROM todolists WHERE id = @p1", id).Scan(&todoList.ID, &todoList.Title)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get todo list: %v", err))
	}

	// Listeye ait görevleri de çek
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

	// Veritabanında listeyi güncelle
	res, err := db.Exec("UPDATE todolists SET title = @p1 WHERE id = @p2", updatedTodoList.Title, id)
	if err != nil {
		// SQL Server'daki UNIQUE kısıtlama hatası mesajı veya kodu kontrolü gerekebilir.
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

	res, err := db.Exec("DELETE FROM todolists WHERE id = @p1", id)
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

// -- Todo (Görev) Handler Fonksiyonları --

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

	// Önce listenin varlığını kontrol et
	var exists int // SQL Server'da EXISTS 1 veya 0 döndürür, BIT tipi int olarak taranabilir
	err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM todolists WHERE id = @p1), 1, 0)", listID).Scan(&exists)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check list existence: %v", err))
	}
	if exists == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
	}

	// Veritabanına yeni görev ekle
	// SQL Server'da BOOLEAN yerine BIT kullanıldığı için, Go'daki bool'u int'e dönüştürmemiz gerekebilir.
	// go-mssqldb sürücüsü genellikle bool'u BIT'e otomatik dönüştürür, ancak emin olmak için:
	completedBit := 0
	if newTodo.Completed {
		completedBit = 1
	}

	var lastInsertID int
	err = db.QueryRow(
		"INSERT INTO todos (list_id, title, description, completed) OUTPUT INSERTED.id VALUES (@p1, @p2, @p3, @p4)",
		listID, newTodo.Title, newTodo.Description, completedBit,
	).Scan(&lastInsertID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create todo in list: %v", err))
	}

	newTodo.ID = lastInsertID
	newTodo.ListID = listID // Atanan list ID'sini de nesneye ekle

	return c.JSON(http.StatusCreated, newTodo)
}

func getTodosInList(c echo.Context) error {
	listID, err := strconv.Atoi(c.Param("listID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid List ID format")
	}

	todosInList, err := getTodosForList(listID)
	if err != nil {
		// getTodosForList zaten NotFound hatasını döndürebilir, burada kontrol edelim
		if err.Error() == "To-Do list not found" {
			return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get todos for list: %v", err))
	}

	return c.JSON(http.StatusOK, todosInList)
}

// getTodosForList, belirli bir listeye ait tüm görevleri veritabanından çeken yardımcı bir fonksiyondur.
func getTodosForList(listID int) ([]Todo, error) {
	// Önce listenin varlığını kontrol et
	var listExists int
	err := db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM todolists WHERE id = @p1), 1, 0)", listID).Scan(&listExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check list existence: %w", err)
	}
	if listExists == 0 {
		return nil, fmt.Errorf("To-Do list not found")
	}

	// SQL Server'da BIT kolonunu Go'da bool'a tarama
	rows, err := db.Query("SELECT id, list_id, title, description, completed FROM todos WHERE list_id = @p1", listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get todos from database: %w", err)
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		// completedBit'i kaldırın, doğrudan todo.Completed'a tarayın
		if err := rows.Scan(&todo.ID, &todo.ListID, &todo.Title, &todo.Description, &todo.Completed); err != nil { // Değişiklik burada
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
	// completedBit'i kaldırın, doğrudan todo.Completed'a tarayın
	err = db.QueryRow("SELECT id, list_id, title, description, completed FROM todos WHERE list_id = @p1 AND id = @p2", listID, todoID).
		Scan(&todo.ID, &todo.ListID, &todo.Title, &todo.Description, &todo.Completed) // Değişiklik burada
	if err == sql.ErrNoRows {
		var listExists int
		err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM todolists WHERE id = @p1), 1, 0)", listID).Scan(&listExists)
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

	// Önce listenin ve görevin varlığını kontrol et
	var todoExists int
	err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM todos WHERE list_id = @p1 AND id = @p2), 1, 0)", listID, todoID).Scan(&todoExists)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check todo existence: %v", err))
	}
	if todoExists == 0 {
		var listExists int
		err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM todolists WHERE id = @p1), 1, 0)", listID).Scan(&listExists)
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

	// Veritabanında görevi güncelle
	res, err := db.Exec(
		"UPDATE todos SET title = @p1, description = @p2, completed = @p3 WHERE list_id = @p4 AND id = @p5",
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
	err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM todos WHERE list_id = @p1 AND id = @p2), 1, 0)", listID, todoID).Scan(&todoExists)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check todo existence: %v", err))
	}
	if todoExists == 0 {
		var listExists int
		err = db.QueryRow("SELECT IIF(EXISTS(SELECT 1 FROM todolists WHERE id = @p1), 1, 0)", listID).Scan(&listExists)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check list existence: %v", err))
		}
		if listExists == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "To-Do list not found")
		}
		return echo.NewHTTPError(http.StatusNotFound, "To-Do item not found in this list")
	}

	res, err := db.Exec("DELETE FROM todos WHERE list_id = @p1 AND id = @p2", listID, todoID)
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
