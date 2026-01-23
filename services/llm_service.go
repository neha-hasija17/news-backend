package services

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"

	"news-backend/config"
	"news-backend/models"
	"news-backend/prompts"

	openai "github.com/sashabaranov/go-openai"
)

type LLMService struct {
	client       *openai.Client
	cfg          *config.Config
	summaryCache sync.Map // Cache for article summaries
}

// NewLLMService creates a new LLM service instance
func NewLLMService(cfg *config.Config) *LLMService {
	var client *openai.Client

	switch cfg.LLMProvider {
	case "openai":
		clientConfig := openai.DefaultConfig(cfg.OpenAIKey)
		client = openai.NewClientWithConfig(clientConfig)
	case "groq":
		clientConfig := openai.DefaultConfig(cfg.GroqKey)
		clientConfig.BaseURL = cfg.LLMBaseURL
		client = openai.NewClientWithConfig(clientConfig)
	default:
		log.Fatalf("Invalid LLM provider: %s", cfg.LLMProvider)
	}

	return &LLMService{
		client: client,
		cfg:    cfg,
	}
}

// ParseIntent analyzes user query and extracts intent and entities using LLM
func (s *LLMService) ParseIntent(query string) models.IntentResponse {
	ctx := context.Background()

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.cfg.IntentModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: prompts.IntentParsingPrompt},
			{Role: "user", Content: query},
		},
		Temperature: 0.0,
		MaxTokens:   200,
	})

	if err != nil {
		log.Printf("LLM intent parsing error: %v", err)
		// Fallback to search intent
		return models.IntentResponse{
			Intent:   models.IntentSearch,
			Entities: models.Entities{"query": query},
		}
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Clean up markdown code blocks if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var intentResp models.IntentResponse
	if err := json.Unmarshal([]byte(content), &intentResp); err != nil {
		log.Printf("Failed to parse LLM response: %v, content: %s", err, content)
		// Fallback
		return models.IntentResponse{
			Intent:   models.IntentSearch,
			Entities: models.Entities{"query": query},
		}
	}

	// Validate intent
	validIntents := map[string]bool{
		models.IntentCategory: true,
		models.IntentSource:   true,
		models.IntentSearch:   true,
		models.IntentNearby:   true,
		models.IntentScore:    true,
	}

	if !validIntents[intentResp.Intent] {
		log.Printf("Invalid intent from LLM: %s, defaulting to search", intentResp.Intent)
		intentResp.Intent = models.IntentSearch
	}

	// Ensure entities map exists
	if intentResp.Entities == nil {
		intentResp.Entities = make(models.Entities)
	}

	// Add query to entities if not present
	if _, ok := intentResp.Entities["query"]; !ok {
		intentResp.Entities["query"] = query
	}

	return intentResp
}

// GenerateSummary creates a concise summary of article content using LLM
func (s *LLMService) GenerateSummary(articleID, text string) string {
	// Check cache first
	if cached, ok := s.summaryCache.Load(articleID); ok {
		return cached.(string)
	}

	// Validate input
	if len(text) < 20 {
		return "Summary unavailable - insufficient content."
	}

	// Truncate very long text to save tokens
	if len(text) > 1000 {
		text = text[:1000]
	}

	ctx := context.Background()

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.cfg.SummaryModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: prompts.SummaryPrompt},
			{Role: "user", Content: text},
		},
		Temperature: 0.3,
		MaxTokens:   100,
	})

	if err != nil {
		log.Printf("LLM summarization error for article %s: %v", articleID, err)
		return "Summary unavailable."
	}

	summary := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Cache the summary
	s.summaryCache.Store(articleID, summary)

	return summary
}

// GenerateSummariesBatch generates summaries for multiple articles concurrently
func (s *LLMService) GenerateSummariesBatch(articles []models.Article) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent LLM calls

	for i := range articles {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			articles[idx].LLMSummary = s.GenerateSummary(
				articles[idx].ID,
				articles[idx].Description,
			)
		}(i)
	}

	wg.Wait()
}
