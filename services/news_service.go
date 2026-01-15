package services

import (
	"fmt"
	"time"

	"news-backend/config"
	"news-backend/database"
	"news-backend/models"
	"news-backend/utils"

	"gorm.io/gorm"
)

type NewsService struct {
	db         *gorm.DB
	cfg        *config.Config
	llmService *LLMService
}

// FetchResult contains articles and metadata about the fetch operation
type FetchResult struct {
	Articles       []models.Article
	TotalAvailable int // Total matching articles before limiting
}

// FetchParams contains parameters for fetching articles
type FetchParams struct {
	Intent        string
	Entities      map[string]string
	NamedEntities *models.NamedEntities
	Lat           float64
	Lon           float64
	Radius        float64
}

// NewNewsService creates a new news service instance
func NewNewsService(cfg *config.Config, llmService *LLMService) *NewsService {
	return &NewsService{
		db:         database.GetDB(),
		cfg:        cfg,
		llmService: llmService,
	}
}

// FetchArticles retrieves articles based on intent and entities
func (s *NewsService) FetchArticles(intent string, entities map[string]string, lat, lon, radius float64) ([]models.Article, error) {
	result, err := s.FetchArticlesWithMetadata(FetchParams{
		Intent:   intent,
		Entities: entities,
		Lat:      lat,
		Lon:      lon,
		Radius:   radius,
	})
	if err != nil {
		return nil, err
	}
	return result.Articles, nil
}

// FetchArticlesWithNamedEntities retrieves articles with named entity support
func (s *NewsService) FetchArticlesWithNamedEntities(intent string, entities map[string]string, namedEntities *models.NamedEntities, lat, lon, radius float64) ([]models.Article, error) {
	result, err := s.FetchArticlesWithMetadata(FetchParams{
		Intent:        intent,
		Entities:      entities,
		NamedEntities: namedEntities,
		Lat:           lat,
		Lon:           lon,
		Radius:        radius,
	})
	if err != nil {
		return nil, err
	}
	return result.Articles, nil
}

// FetchArticlesWithMetadata retrieves articles with total count metadata
func (s *NewsService) FetchArticlesWithMetadata(params FetchParams) (*FetchResult, error) {
	articles, sortType, err := s.fetchArticlesByIntent(params)
	if err != nil {
		return nil, err
	}

	// Apply sorting based on intent
	s.applySorting(articles, sortType, params)

	return s.limitArticlesWithTotal(articles), nil
}

// sortType defines how articles should be sorted
type sortType int

const (
	sortByDateDesc sortType = iota
	sortByScoreDesc
	sortByDistance
	sortBySearchRelevance
)

// fetchArticlesByIntent retrieves articles based on intent and returns the appropriate sort type
func (s *NewsService) fetchArticlesByIntent(params FetchParams) ([]models.Article, sortType, error) {
	query := s.db.Model(&models.Article{})

	switch params.Intent {
	case models.IntentCategory:
		articles, err := s.fetchByCategory(query, params.Entities)
		return articles, sortByDateDesc, err

	case models.IntentSource:
		articles, err := s.fetchBySource(query, params.Entities)
		return articles, sortByDateDesc, err

	case models.IntentScore:
		articles, err := s.fetchByScore(query)
		return articles, sortByScoreDesc, err

	case models.IntentNearby:
		radius := params.Radius
		if radius == 0 {
			radius = s.cfg.DefaultRadius
		}
		articles, err := s.fetchNearby(params.Lat, params.Lon, radius, params.Entities)
		return articles, sortByDistance, err

	case models.IntentSearch:
		articles, err := s.fetchBySearch(query, params.Entities)
		return articles, sortBySearchRelevance, err

	default:
		articles, err := s.fetchBySearch(query, params.Entities)
		return articles, sortByDateDesc, err
	}
}

// applySorting applies the appropriate sorting based on sort type
func (s *NewsService) applySorting(articles []models.Article, st sortType, params FetchParams) {
	switch st {
	case sortByDateDesc:
		utils.SortArticles(articles, utils.SortDateDesc)
	case sortByScoreDesc:
		utils.SortArticles(articles, utils.SortScoreDesc)
	case sortByDistance:
		utils.SortByDistanceFrom[models.Article](articles, params.Lat, params.Lon)
	case sortBySearchRelevance:
		// Requirement: rank by combination of relevance_score and text matching score
		utils.SortBySearchRelevance(articles, params.Entities["query"])
	}
}

// EnrichWithSummaries adds LLM-generated summaries to articles
func (s *NewsService) EnrichWithSummaries(articles []models.Article) []models.Article {
	s.llmService.GenerateSummariesBatch(articles)
	return articles
}

// SearchWithIntent performs search with LLM intent parsing
func (s *NewsService) SearchWithIntent(query string) (*FetchResult, models.IntentResponse, error) {
	intentResp := s.llmService.ParseIntent(query)
	entities := map[string]string{"query": query}

	result, err := s.FetchArticlesWithMetadata(FetchParams{
		Intent:        models.IntentSearch,
		Entities:      entities,
		NamedEntities: intentResp.NamedEntities,
	})
	if err != nil {
		return nil, models.IntentResponse{}, err
	}

	result.Articles = s.EnrichWithSummaries(result.Articles)
	return result, intentResp, nil
}

// QueryWithIntent handles generic queries with intent parsing
func (s *NewsService) QueryWithIntent(query string, lat, lon, radius float64) ([]models.Article, models.IntentResponse, error) {
	intentResp := s.llmService.ParseIntent(query)

	articles, err := s.FetchArticles(
		intentResp.Intent,
		intentResp.Entities,
		lat,
		lon,
		radius,
	)
	if err != nil {
		return nil, models.IntentResponse{}, err
	}

	articles = s.EnrichWithSummaries(articles)
	return articles, intentResp, nil
}

// GetArticleByID retrieves a single article by ID
func (s *NewsService) GetArticleByID(id string) (*models.Article, error) {
	var article models.Article
	err := s.db.Where("id = ?", id).First(&article).Error
	if err != nil {
		return nil, fmt.Errorf("article not found: %w", err)
	}
	return &article, nil
}

// GetArticleByIDWithSummary retrieves a single article with LLM summary
func (s *NewsService) GetArticleByIDWithSummary(id string) (*models.Article, error) {
	article, err := s.GetArticleByID(id)
	if err != nil {
		return nil, err
	}
	article.LLMSummary = s.llmService.GenerateSummary(article.ID, article.Description)
	return article, nil
}

// GetArticleStats returns statistics about the article database
func (s *NewsService) GetArticleStats() (map[string]interface{}, error) {
	var totalCount int64
	var categories []string
	var sources []string

	// Total articles
	s.db.Model(&models.Article{}).Count(&totalCount)

	// Unique categories
	s.db.Model(&models.Article{}).Distinct("category").Pluck("category", &categories)

	// Unique sources
	s.db.Model(&models.Article{}).Distinct("source_name").Pluck("source_name", &sources)

	// Date range
	var oldestArticle, newestArticle models.Article
	s.db.Order("publication_date ASC").First(&oldestArticle)
	s.db.Order("publication_date DESC").First(&newestArticle)

	stats := map[string]interface{}{
		"total_articles":    totalCount,
		"unique_categories": len(categories),
		"unique_sources":    len(sources),
		"oldest_article":    oldestArticle.PublicationDate.Format(time.RFC3339),
		"newest_article":    newestArticle.PublicationDate.Format(time.RFC3339),
	}

	return stats, nil
}
