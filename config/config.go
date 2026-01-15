package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	// Server Configuration
	ServerPort string
	
	// Database Configuration
	DatabasePath string
	
	// LLM Configuration
	LLMProvider    string // "openai" or "groq"
	OpenAIKey      string
	GroqKey        string
	LLMBaseURL     string
	IntentModel    string
	SummaryModel   string
	
	// Business Logic Configuration
	DefaultRadius      float64
	MaxArticlesReturn  int
	ScoreThreshold     float64
	
	// Trending Configuration
	TrendingCacheTTL   int // seconds
	TrendingRadius     float64
	TrendingTimeWindow int // hours
}

var AppConfig *Config

func LoadConfig() *Config {
	AppConfig = &Config{
		ServerPort:         getEnv("PORT", "8080"),
		DatabasePath:       getEnv("DB_PATH", "news.db"),
		LLMProvider:        getEnv("LLM_PROVIDER", "groq"),
		OpenAIKey:          os.Getenv("OPENAI_API_KEY"),
		GroqKey:            os.Getenv("GROQ_API_KEY"),
		LLMBaseURL:         getEnv("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
		IntentModel:        getEnv("INTENT_MODEL", "llama-3.3-70b-versatile"),
		SummaryModel:       getEnv("SUMMARY_MODEL", "llama-3.1-8b-instant"),
		DefaultRadius:      getEnvFloat("DEFAULT_RADIUS", 10.0),
		MaxArticlesReturn:  getEnvInt("MAX_ARTICLES", 5),
		ScoreThreshold:     getEnvFloat("SCORE_THRESHOLD", 0.7),
		TrendingCacheTTL:   getEnvInt("TRENDING_CACHE_TTL", 300),
		TrendingRadius:     getEnvFloat("TRENDING_RADIUS", 50.0),
		TrendingTimeWindow: getEnvInt("TRENDING_TIME_WINDOW", 24),
	}
	
	// Validate required configuration
	if AppConfig.LLMProvider == "openai" && AppConfig.OpenAIKey == "" {
		log.Fatal("OPENAI_API_KEY is required when LLM_PROVIDER is 'openai'")
	}
	if AppConfig.LLMProvider == "groq" && AppConfig.GroqKey == "" {
		log.Fatal("GROQ_API_KEY is required when LLM_PROVIDER is 'groq'")
	}
	
	return AppConfig
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
