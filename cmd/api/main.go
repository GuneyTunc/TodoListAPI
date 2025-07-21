package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"todo-api/internal/application"
	"todo-api/internal/delivery/http"
	"todo-api/internal/infrastructure/ai"
	"todo-api/internal/infrastructure/config"
	"todo-api/internal/infrastructure/persistence/sqlserver" // SQL Server implementasyonu

	_ "github.com/denisenkom/go-mssqldb" // MSSQL sürücüsü
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger" // Swagger UI için
)

func main() {
	// 1. Konfigürasyonu Yükle
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	fmt.Printf("Main: Loaded OpenAI API Key (first 5 chars): '%s'\n", cfg.OpenAIAPIKey[:5]) // İlk 5 karakteri gösteriyoruz

	// 2. Veritabanı Bağlantısı Kur
	db, err := sql.Open("sqlserver", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("Error closing database connection: %v", cerr)
		}
	}()

	// Veritabanı bağlantısını kontrol et
	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			log.Println("Successfully connected to the database!")
			break
		}
		log.Printf("Could not connect to database, retrying in 5 seconds... (%v)", err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database after multiple retries: %v", err)
	}

	// 3. Repository'leri Başlat
	todoListRepo := sqlserver.NewSQLServerTodoListRepository(db)
	todoRepo := sqlserver.NewSQLServerTodoRepository(db)

	aiService := ai.NewOpenAIService(cfg.OpenAIAPIKey)
	if aiService == nil {
		log.Fatalf("Main: aiService is nil after NewOpenAIService call!")
	}
	fmt.Printf("Main: aiService created (type: %T)\n", aiService)

	// 5. Use Case'leri Başlat
	todoListUseCase := application.NewTodoListUseCase(todoListRepo, todoRepo)
	todoUseCase := application.NewTodoUseCase(todoRepo, todoListRepo, aiService)
	aiSuggestionUseCase := application.NewAISuggestionUseCase(aiService)

	// 6. HTTP Handler'ları Başlat
	httpHandler := http.NewHTTPHandler(todoListUseCase, todoUseCase, aiSuggestionUseCase)

	// 7. Echo Web Framework'ü Başlat
	e := echo.New()

	// Middleware'ler
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Geliştirme için tüm originlere izin ver
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// Swagger UI rotası
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// API Rotaları (Handlers.go'daki fonksiyonlara yönlendirmeler)
	// TodoList Rotaları
	e.POST("/todolists", httpHandler.CreateTodoList)
	e.GET("/todolists", httpHandler.GetTodoLists)
	e.GET("/todolists/:id", httpHandler.GetTodoListByID)
	e.PUT("/todolists/:id", httpHandler.UpdateTodoList)
	e.DELETE("/todolists/:id", httpHandler.DeleteTodoList)

	// Todo Rotaları
	e.POST("/todolists/:listID/todos", httpHandler.CreateTodoInList)
	e.GET("/todolists/:listID/todos", httpHandler.GetTodosInList)
	e.GET("/todolists/:listID/todos/:todoID", httpHandler.GetTodoInListByID)
	e.PUT("/todolists/:listID/todos/:todoID", httpHandler.UpdateTodoInList)
	e.DELETE("/todolists/:listID/todos/:todoID", httpHandler.DeleteTodoInList)

	// AI Rotaları
	e.POST("/todos/suggest-description", httpHandler.SuggestTodoDescription)

	// 8. Sunucuyu Başlat
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Varsayılan port
	}
	log.Printf("Server starting on port %s", port)
	if err := e.Start(fmt.Sprintf(":%s", port)); err != nil { // Sadece hata olup olmadığını kontrol et
		log.Fatalf("Server failed to start: %v", err)
	}
}
