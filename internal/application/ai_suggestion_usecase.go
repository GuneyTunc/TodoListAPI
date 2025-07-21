package application

import (
	"context"
	"fmt"

	"todo-api/internal/domain" // Modül yolunuzu projenizin go.mod dosyasına göre güncelleyin
)

// AISuggestionUseCase defines the business logic for AI-based suggestions.
type AISuggestionUseCase struct {
	aiService domain.AIService // AIService arayüzüne bağımlılık
}

// NewAISuggestionUseCase creates a new AISuggestionUseCase.
func NewAISuggestionUseCase(aiService domain.AIService) *AISuggestionUseCase {
	return &AISuggestionUseCase{
		aiService: aiService,
	}
}

// SuggestTodoDescription generates an AI-based description suggestion for a given todo title.
func (uc *AISuggestionUseCase) SuggestTodoDescription(ctx context.Context, title string) (string, error) {
	// AI servisini kullanarak açıklama önerisi al
	suggestion, err := uc.aiService.GenerateDescription(ctx, title)
	if err != nil {
		return "", fmt.Errorf("failed to generate AI description suggestion: %w", err)
	}
	return suggestion, nil
}
