package main

import (
	"log"
	"os"

	"news-backend/config"
	"news-backend/database"
	"news-backend/handlers"
	"news-backend/middleware"
	"news-backend/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	log.Println("Configuration loaded successfully")

	// Initialize database
	if err := database.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database initialized")

	// Load news data from JSON file
	dataFile := "news_data.json"
	if _, err := os.Stat(dataFile); err == nil {
		if err := database.LoadNewsData(dataFile); err != nil {
			log.Printf("Warning: Failed to load news data: %v", err)
		}
	} else {
		log.Printf("Warning: News data file not found: %s", dataFile)
	}

	// Seed user events for trending functionality
	if err := database.SeedUserEvents(); err != nil {
		log.Printf("Warning: Failed to seed user events: %v", err)
	}

	// Initialize services
	llmService := services.NewLLMService(cfg)
	newsService := services.NewNewsService(cfg, llmService)
	trendingService := services.NewTrendingService(cfg, llmService)
	log.Println("Services initialized")

	// Initialize handlers
	newsHandler := handlers.NewNewsHandler(newsService)
	trendingHandler := handlers.NewTrendingHandler(trendingService)

	// Setup Gin router
	if cfg.ServerPort == "8080" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Global middleware
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())
	router.Use(gin.Recovery())

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", newsHandler.HealthCheck)

		// News endpoints
		news := v1.Group("/news")
		{
			// API endpoints as per assignment requirements
			news.GET("/category", newsHandler.GetByCategory)
			news.GET("/source", newsHandler.GetBySource)
			news.GET("/score", newsHandler.GetByScore)
			news.GET("/nearby", newsHandler.GetNearby)
			news.GET("/search", newsHandler.Search)

			// Statistics
			news.GET("/stats", newsHandler.GetStats)
		}

		// Trending endpoints
		trending := v1.Group("/trending")
		{
			// Get trending news
			trending.GET("", trendingHandler.GetTrending)

			// Record user event
			trending.POST("/event", trendingHandler.RecordEvent)

			// Statistics
			trending.GET("/stats", trendingHandler.GetEventStats)

			// Cache management
			trending.POST("/cache/invalidate", trendingHandler.InvalidateCache)
		}
	}

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "Contextual News Data Retrieval System",
			"version": "1.0.0",
			"status":  "running",
			"endpoints": gin.H{
				"health":   "/api/v1/health",
				"category": "/api/v1/news/category?query=<query>",
				"source":   "/api/v1/news/source?query=<query>",
				"score":    "/api/v1/news/score?query=<query>",
				"nearby":   "/api/v1/news/nearby?lat=<lat>&lon=<lon>&radius=<km>&query=<query>",
				"search":   "/api/v1/news/search?query=<query>",
				"trending": "/api/v1/trending?lat=<lat>&lon=<lon>&radius=<km>&limit=<n>",
			},
		})
	})

	// Start server
	serverAddr := ":" + cfg.ServerPort
	log.Printf("Starting server on %s", serverAddr)
	log.Printf("API Documentation: http://localhost%s/", serverAddr)

	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
