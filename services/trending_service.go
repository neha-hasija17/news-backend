package services

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"news-backend/config"
	"news-backend/database"
	"news-backend/models"
	"news-backend/utils"

	"gorm.io/gorm"
)

type TrendingService struct {
	db         *gorm.DB
	cfg        *config.Config
	llmService *LLMService
	cache      sync.Map // Location-based cache
	cacheTimes sync.Map // Track cache timestamps
}

// NewTrendingService creates a new trending service instance
func NewTrendingService(cfg *config.Config, llmService *LLMService) *TrendingService {
	return &TrendingService{
		db:         database.GetDB(),
		cfg:        cfg,
		llmService: llmService,
	}
}

// TrendingCache represents cached trending data
type TrendingCache struct {
	Articles []models.TrendingArticle
	CachedAt time.Time
	Location string
	RadiusKm float64
}

// GetTrendingNews retrieves trending news based on user events and location
func (s *TrendingService) GetTrendingNews(lat, lon, radius float64, limit int) ([]models.TrendingArticle, *TrendingCache, error) {
	if radius == 0 {
		radius = s.cfg.TrendingRadius
	}

	if limit == 0 || limit > s.cfg.MaxArticlesReturn {
		limit = s.cfg.MaxArticlesReturn
	}

	// Generate cache key based on location grid
	cacheKey := s.getCacheKey(lat, lon, radius)

	// Check cache
	if cached, ok := s.getFromCache(cacheKey); ok {
		log.Printf("Returning cached trending data for location (%.4f, %.4f)", lat, lon)
		return cached.Articles, cached, nil
	}

	// Calculate trending scores
	trendingArticles, err := s.calculateTrendingScores(lat, lon, radius)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate trending scores: %w", err)
	}

	// Sort by trending score
	sort.Slice(trendingArticles, func(i, j int) bool {
		return trendingArticles[i].TrendingScore > trendingArticles[j].TrendingScore
	})

	// Limit results
	if len(trendingArticles) > limit {
		trendingArticles = trendingArticles[:limit]
	}

	// Cache results
	cache := &TrendingCache{
		Articles: trendingArticles,
		CachedAt: time.Now(),
		Location: fmt.Sprintf("%.4f,%.4f", lat, lon),
		RadiusKm: radius,
	}
	s.putInCache(cacheKey, cache)

	log.Printf("Calculated and cached %d trending articles for location (%.4f, %.4f)",
		len(trendingArticles), lat, lon)

	return trendingArticles, cache, nil
}

// GetTrendingNewsWithSummaries retrieves trending news with LLM summaries
func (s *TrendingService) GetTrendingNewsWithSummaries(lat, lon, radius float64, limit int) ([]models.TrendingArticle, *TrendingCache, error) {
	trendingArticles, cache, err := s.GetTrendingNews(lat, lon, radius, limit)
	if err != nil {
		return nil, nil, err
	}

	// Convert TrendingArticle to Article for batch processing
	articles := make([]models.Article, len(trendingArticles))
	for i := range trendingArticles {
		articles[i] = models.Article{
			ID:          trendingArticles[i].ID,
			Description: trendingArticles[i].Description,
			LLMSummary:  trendingArticles[i].LLMSummary,
		}
	}

	// Batch generate summaries
	s.llmService.GenerateSummariesBatch(articles)

	// Copy summaries back to trending articles
	for i := range trendingArticles {
		trendingArticles[i].LLMSummary = articles[i].LLMSummary
	}

	return trendingArticles, cache, nil
}

// calculateTrendingScores computes trending scores for articles based on user events
func (s *TrendingService) calculateTrendingScores(lat, lon, radius float64) ([]models.TrendingArticle, error) {
	// Get time window
	timeWindow := time.Now().Add(-time.Duration(s.cfg.TrendingTimeWindow) * time.Hour)

	// Get all events within time window
	var events []models.UserEvent
	err := s.db.Where("timestamp >= ?", timeWindow).Find(&events).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user events: %w", err)
	}

	log.Printf("Found %d user events in last %d hours", len(events), s.cfg.TrendingTimeWindow)

	// Filter events by location and aggregate by article
	articleEvents := make(map[string][]models.UserEvent)
	for _, event := range events {
		if utils.IsWithinRadius(lat, lon, event.Latitude, event.Longitude, radius) {
			articleEvents[event.ArticleID] = append(articleEvents[event.ArticleID], event)
		}
	}

	log.Printf("Found events for %d articles within %.2f km", len(articleEvents), radius)

	if len(articleEvents) == 0 {
		// No events found, return popular articles by relevance score
		return s.getFallbackTrending(lat, lon, radius)
	}

	// Calculate trending score for each article
	trendingArticles := []models.TrendingArticle{}
	now := time.Now()

	for articleID, events := range articleEvents {
		// Fetch article details
		var article models.Article
		if err := s.db.Where("id = ?", articleID).First(&article).Error; err != nil {
			log.Printf("Article %s not found, skipping", articleID)
			continue
		}

		// Calculate distance from query location
		distance := utils.CalculateDistance[models.Article](&article, lat, lon)

		// Calculate trending score
		totalWeight := 0.0
		for _, event := range events {
			// Weight by event type
			weight := models.GetEventWeight(event.EventType)

			// Apply recency decay
			hoursAgo := now.Sub(event.Timestamp).Hours()
			recencyFactor := utils.CalculateRecencyFactor(hoursAgo)

			totalWeight += weight * recencyFactor
		}

		// Compute final trending score
		trendingScore := utils.ComputeTrendingScore(len(events), totalWeight, 1.0)

		// Boost by article relevance and proximity
		trendingScore *= (1.0 + article.RelevanceScore*0.2)
		if distance < 10 {
			trendingScore *= 1.5 // Boost very local news
		}

		trendingArticle := models.TrendingArticle{
			Article:       article,
			TrendingScore: trendingScore,
			EventCount:    len(events),
		}

		trendingArticles = append(trendingArticles, trendingArticle)
	}

	return trendingArticles, nil
}

// getFallbackTrending returns popular articles when no events are found
func (s *TrendingService) getFallbackTrending(lat, lon, radius float64) ([]models.TrendingArticle, error) {
	var articles []models.Article

	// Get all articles
	s.db.Find(&articles)

	// Filter by location and score using generic helper
	scoreThreshold := s.cfg.ScoreThreshold
	filtered := utils.FilterByDistanceWithPredicate[models.Article](
		articles, lat, lon, radius,
		func(a *models.Article) bool {
			return a.RelevanceScore >= scoreThreshold
		},
	)

	// Convert to TrendingArticle
	trendingArticles := make([]models.TrendingArticle, len(filtered))
	for i, article := range filtered {
		trendingArticles[i] = models.TrendingArticle{
			Article:       article,
			TrendingScore: article.RelevanceScore * 10, // Use relevance as fallback score
			EventCount:    0,
		}
	}

	log.Printf("Fallback: returning %d articles with high relevance scores", len(trendingArticles))
	return trendingArticles, nil
}

// getCacheKey generates a cache key based on location
func (s *TrendingService) getCacheKey(lat, lon, radius float64) string {
	// Round to grid cells for better cache hits
	// Grid size ~5km
	precision := 0.05
	latCell := int(lat / precision)
	lonCell := int(lon / precision)
	radiusCell := int(radius / 10) // Group by 10km radius increments

	return fmt.Sprintf("trending_%d_%d_%d", latCell, lonCell, radiusCell)
}

// getFromCache retrieves cached trending data if still valid
func (s *TrendingService) getFromCache(key string) (*TrendingCache, bool) {
	if cached, ok := s.cache.Load(key); ok {
		cache := cached.(*TrendingCache)

		// Check if cache is still valid
		if time.Since(cache.CachedAt).Seconds() < float64(s.cfg.TrendingCacheTTL) {
			return cache, true
		}

		// Cache expired, remove it
		s.cache.Delete(key)
		s.cacheTimes.Delete(key)
	}

	return nil, false
}

// putInCache stores trending data in cache
func (s *TrendingService) putInCache(key string, cache *TrendingCache) {
	s.cache.Store(key, cache)
	s.cacheTimes.Store(key, time.Now())
}

// InvalidateCache clears all cached trending data
func (s *TrendingService) InvalidateCache() {
	s.cache.Range(func(key, value interface{}) bool {
		s.cache.Delete(key)
		return true
	})
	s.cacheTimes.Range(func(key, value interface{}) bool {
		s.cacheTimes.Delete(key)
		return true
	})
	log.Println("Trending cache invalidated")
}

// RecordUserEvent records a user interaction with an article
func (s *TrendingService) RecordUserEvent(articleID, userID, eventType string, lat, lon float64) error {
	// Validate event type
	validTypes := map[string]bool{
		models.EventTypeView:  true,
		models.EventTypeClick: true,
		models.EventTypeShare: true,
	}

	if !validTypes[eventType] {
		return fmt.Errorf("invalid event type: %s", eventType)
	}

	// Create event
	event := models.UserEvent{
		ArticleID: articleID,
		UserID:    userID,
		EventType: eventType,
		Latitude:  lat,
		Longitude: lon,
		Timestamp: time.Now(),
	}

	if err := s.db.Create(&event).Error; err != nil {
		return fmt.Errorf("failed to record user event: %w", err)
	}

	log.Printf("Recorded %s event for article %s by user %s", eventType, articleID, userID)

	// Invalidate nearby caches (simple approach)
	// In production, use more sophisticated cache invalidation
	s.InvalidateCache()

	return nil
}

// GetEventStats returns statistics about user events
func (s *TrendingService) GetEventStats() (map[string]interface{}, error) {
	var totalEvents int64
	var uniqueArticles int64
	var uniqueUsers int64

	s.db.Model(&models.UserEvent{}).Count(&totalEvents)
	s.db.Model(&models.UserEvent{}).Distinct("article_id").Count(&uniqueArticles)
	s.db.Model(&models.UserEvent{}).Distinct("user_id").Count(&uniqueUsers)

	// Event type breakdown
	var viewCount, clickCount, shareCount int64
	s.db.Model(&models.UserEvent{}).Where("event_type = ?", models.EventTypeView).Count(&viewCount)
	s.db.Model(&models.UserEvent{}).Where("event_type = ?", models.EventTypeClick).Count(&clickCount)
	s.db.Model(&models.UserEvent{}).Where("event_type = ?", models.EventTypeShare).Count(&shareCount)

	stats := map[string]interface{}{
		"total_events":      totalEvents,
		"unique_articles":   uniqueArticles,
		"unique_users":      uniqueUsers,
		"views":             viewCount,
		"clicks":            clickCount,
		"shares":            shareCount,
		"cache_size":        s.getCacheSize(),
		"cache_ttl_seconds": s.cfg.TrendingCacheTTL,
	}

	return stats, nil
}

// getCacheSize returns the number of cached entries
func (s *TrendingService) getCacheSize() int {
	count := 0
	s.cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
