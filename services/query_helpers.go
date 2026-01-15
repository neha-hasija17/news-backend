package services

import (
	"strings"

	"news-backend/models"
	"news-backend/utils"

	"gorm.io/gorm"
)

// =============================================================================
// Fetch Helpers - Database Query Functions
// =============================================================================

// fetchByField is a generic helper for fetching articles by a single field
func (s *NewsService) fetchByField(query *gorm.DB, field, value string) ([]models.Article, error) {
	var articles []models.Article
	err := query.Where(field+" = ?", value).Find(&articles).Error
	return articles, err
}

// fetchByCategory fetches articles by category
func (s *NewsService) fetchByCategory(query *gorm.DB, entities map[string]string) ([]models.Article, error) {
	category := entities["category"]
	if category == "" {
		return s.fetchLatestArticles(query)
	}
	return s.fetchByField(query, "category", category)
}

// fetchBySource fetches articles by source name
func (s *NewsService) fetchBySource(query *gorm.DB, entities map[string]string) ([]models.Article, error) {
	source := entities["source_name"]
	if source == "" {
		return s.fetchLatestArticles(query)
	}
	return s.fetchByField(query, "source_name", source)
}

// fetchByScore fetches high-scoring articles
func (s *NewsService) fetchByScore(query *gorm.DB) ([]models.Article, error) {
	var articles []models.Article
	err := query.Where("relevance_score >= ?", s.cfg.ScoreThreshold).Find(&articles).Error
	return articles, err
}

// fetchNearby fetches articles near a geographic location
func (s *NewsService) fetchNearby(lat, lon, radius float64, entities map[string]string) ([]models.Article, error) {
	var articles []models.Article
	query := s.db.Model(&models.Article{})

	// Apply text search if query provided
	if queryText := entities["query"]; queryText != "" {
		query = s.applyTextSearch(query, queryText)
	}

	// Get all articles and filter by distance
	if err := query.Find(&articles).Error; err != nil {
		return nil, err
	}

	// Filter by distance using generic helper
	filtered := utils.FilterByDistance[models.Article](articles, lat, lon, radius)

	return filtered, nil
}

// fetchBySearch performs text search across title and description
func (s *NewsService) fetchBySearch(query *gorm.DB, entities map[string]string) ([]models.Article, error) {
	searchQuery := entities["query"]
	if searchQuery == "" {
		return s.fetchLatestArticles(query)
	}

	var articles []models.Article
	err := s.applyTextSearch(query, searchQuery).Find(&articles).Error
	return articles, err
}

// =============================================================================
// Query Building Helpers
// =============================================================================

// applyTextSearch adds text search conditions to a query
func (s *NewsService) applyTextSearch(query *gorm.DB, searchText string) *gorm.DB {
	pattern := "%" + strings.ToLower(searchText) + "%"
	return query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", pattern, pattern)
}

// fetchLatestArticles fetches the most recent articles as a fallback
func (s *NewsService) fetchLatestArticles(query *gorm.DB) ([]models.Article, error) {
	var articles []models.Article
	err := query.Order("publication_date DESC").Limit(s.cfg.MaxArticlesReturn).Find(&articles).Error
	return articles, err
}

// =============================================================================
// Result Limiting Helpers
// =============================================================================

// limitArticles limits the number of articles returned
func (s *NewsService) limitArticles(articles []models.Article) []models.Article {
	if len(articles) > s.cfg.MaxArticlesReturn {
		return articles[:s.cfg.MaxArticlesReturn]
	}
	return articles
}

// limitArticlesWithTotal returns a FetchResult with total count and limited articles
func (s *NewsService) limitArticlesWithTotal(articles []models.Article) *FetchResult {
	total := len(articles)
	return &FetchResult{
		Articles:       s.limitArticles(articles),
		TotalAvailable: total,
	}
}
