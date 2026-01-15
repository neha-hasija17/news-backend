package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"news-backend/config"
	"news-backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB(cfg *config.Config) error {
	var err error
	
	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}
	
	DB, err = gorm.Open(sqlite.Open(cfg.DatabasePath), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Auto migrate schemas
	err = DB.AutoMigrate(
		&models.Article{},
		&models.UserEvent{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	
	log.Println("Database initialized successfully")
	return nil
}

// LoadNewsData loads news articles from JSON file into database
func LoadNewsData(filePath string) error {
	// Check if data already exists
	var count int64
	DB.Model(&models.Article{}).Count(&count)
	if count > 0 {
		log.Printf("Database already contains %d articles, skipping data load", count)
		return nil
	}
	
	log.Println("Loading news data from file:", filePath)
	
	// Read JSON file
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read data file: %w", err)
	}
	
	// Parse JSON directly into Article slice (uses custom UnmarshalJSON)
	var articles []models.Article
	if err := json.Unmarshal(raw, &articles); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	log.Printf("Parsed %d articles from file", len(articles))
	
	// Insert articles in batches
	batchSize := 100
	successCount := 0
	errorCount := 0
	
	for i := 0; i < len(articles); i += batchSize {
		end := i + batchSize
		if end > len(articles) {
			end = len(articles)
		}
		
		batch := articles[i:end]
		if err := DB.Create(&batch).Error; err != nil {
			log.Printf("Failed to insert batch: %v", err)
			errorCount += len(batch)
		} else {
			successCount += len(batch)
		}
	}
	
	log.Printf("Data load complete: %d successful, %d errors", successCount, errorCount)
	return nil
}

// SeedUserEvents generates sample user events for testing trending functionality
func SeedUserEvents() error {
	// Check if events already exist
	var count int64
	DB.Model(&models.UserEvent{}).Count(&count)
	if count > 0 {
		log.Printf("Database already contains %d user events, skipping seed", count)
		return nil
	}
	
	log.Println("Seeding sample user events...")
	
	// Get some articles to create events for
	var articles []models.Article
	DB.Limit(50).Find(&articles)
	
	if len(articles) == 0 {
		return fmt.Errorf("no articles found to create events")
	}
	
	events := []models.UserEvent{}
	now := time.Now()
	
	// Create diverse events for different articles
	for i, article := range articles {
		// Recent articles get more engagement
		baseEvents := 10
		if i < 10 {
			baseEvents = 50 // Top 10 articles are very popular
		} else if i < 20 {
			baseEvents = 25 // Next 10 are moderately popular
		}
		
		for j := 0; j < baseEvents; j++ {
			// Distribute events over last 24 hours
			hoursAgo := float64(j%24) + (float64(j%10) / 10.0)
			timestamp := now.Add(-time.Duration(hoursAgo) * time.Hour)
			
			// Vary event types
			eventType := models.EventTypeView
			if j%3 == 0 {
				eventType = models.EventTypeClick
			}
			if j%7 == 0 {
				eventType = models.EventTypeShare
			}
			
			event := models.UserEvent{
				ArticleID: article.ID,
				UserID:    fmt.Sprintf("user_%d", j%20), // Simulate 20 users
				EventType: eventType,
				Latitude:  article.Latitude + (float64(j%5) - 2) * 0.1, // Vary location slightly
				Longitude: article.Longitude + (float64(j%5) - 2) * 0.1,
				Timestamp: timestamp,
			}
			events = append(events, event)
		}
	}
	
	// Insert events in batches
	batchSize := 500
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}
		
		if err := DB.Create(events[i:end]).Error; err != nil {
			log.Printf("Failed to insert event batch: %v", err)
		}
	}
	
	log.Printf("Seeded %d sample user events", len(events))
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
