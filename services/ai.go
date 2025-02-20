package services

import (
	"context"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"os"
	"strings"

	"google.golang.org/api/option"
)

type AIService interface {
	ProcessMessage(message string) (string, error)
	HandleMention(message string, username string) (string, error)
}

type aiService struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewAIService() (AIService, error) {
	// Get API key from environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %v", err)
	}

	// Initialize the model (using Gemini-Pro)
	model := client.GenerativeModel("gemini-2.0-pro-exp-02-05")

	return &aiService{
		client: client,
		model:  model,
	}, nil
}

func (s *aiService) ProcessMessage(message string) (string, error) {
	ctx := context.Background()

	// Remove the /ai prefix if present
	message = strings.TrimPrefix(message, "/ai")
	message = strings.TrimSpace(message)

	// Generate response
	resp, err := s.model.GenerateContent(ctx, genai.Text(message))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	// Get the response text
	response := ""
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			response += fmt.Sprint(part)
		}
	}

	return response, nil
}

func (s *aiService) HandleMention(message string, username string) (string, error) {
	// Remove the @AI mention and process the remaining message
	message = strings.ReplaceAll(message, "@AI", "")
	message = strings.TrimSpace(message)

	return s.ProcessMessage(message)
}
