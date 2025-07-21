package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv" // .env dosyasını okumak için
)

// Config holds all application configuration.
type Config struct {
	DatabaseURL  string
	OpenAIAPIKey string
	Port         string
}

// LoadConfig loads configuration from environment variables.
// It tries to load from a .env file first for local development.
func LoadConfig() (*Config, error) {
	// Geliştirme ortamında .env dosyasından yükle
	// Üretim ortamında ortam değişkenleri doğrudan kullanılacaktır
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found or error loading .env. Assuming environment variables are set.")
	}

	cfg := &Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		Port:         os.Getenv("PORT"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}
	// OpenAI API anahtarı AI özellikleri kullanılmıyorsa zorunlu olmayabilir,
	// ancak projemizde kullanıldığı için kontrol ediyoruz.
	if cfg.OpenAIAPIKey == "" {
		// Bu hatayı loglamak veya fatal olmak yerine, AI özelliği olmadan devam etme kararı alınabilir.
		// Şimdilik hata verelim.
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	// Port boşsa varsayılanı ayarla
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}
