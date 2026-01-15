package handlers

import (
	"fmt"
	"net/http"

	"news-backend/models"
	"news-backend/services"

	"github.com/gin-gonic/gin"
)

type NewsHandler struct {
	newsService *services.NewsService
}

// NewNewsHandler creates a new news handler
func NewNewsHandler(newsService *services.NewsService) *NewsHandler {
	return &NewsHandler{
		newsService: newsService,
	}
}

// QueryNews handles generic news queries with LLM intent parsing
// GET /api/v1/news/query?query=...&lat=...&lon=...&radius=...
func (h *NewsHandler) QueryNews(c *gin.Context) {
	var req models.NewsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondBadRequest(c, err.Error())
		return
	}

	articles, intentResp, err := h.newsService.QueryWithIntent(
		req.Query,
		req.Latitude,
		req.Longitude,
		req.Radius,
	)
	if err != nil {
		respondInternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, models.NewsQueryResponse{
		Intent:   intentResp.Intent,
		Entities: intentResp.Entities,
		Articles: articlesToResponses(articles),
		Count:    len(articles),
	})
}

// GetByCategory retrieves news by category
// GET /api/v1/news/category?category=Technology
func (h *NewsHandler) GetByCategory(c *gin.Context) {
	category := c.Query("category")
	if category == "" {
		respondMissingParam(c, "Category parameter")
		return
	}

	h.fetchAndRespond(c, models.IntentCategory, FetchOptions{
		Entities: map[string]string{"category": category},
		Filters:  map[string]string{"category": category},
	})
}

// GetBySource retrieves news by source
// GET /api/v1/news/source?source=Reuters
func (h *NewsHandler) GetBySource(c *gin.Context) {
	source := c.Query("source")
	if source == "" {
		respondMissingParam(c, "Source parameter")
		return
	}

	h.fetchAndRespond(c, models.IntentSource, FetchOptions{
		Entities: map[string]string{"source_name": source},
		Filters:  map[string]string{"source": source},
	})
}

// GetByScore retrieves high-relevance articles
// GET /api/v1/news/score
func (h *NewsHandler) GetByScore(c *gin.Context) {
	h.fetchAndRespond(c, models.IntentScore, FetchOptions{
		Filters: map[string]string{"filter": "high_relevance"},
	})
}

// GetNearby retrieves news near a location
// GET /api/v1/news/nearby?lat=37.4220&lon=-122.0840&radius=10
func (h *NewsHandler) GetNearby(c *gin.Context) {
	var req struct {
		Lat    float64 `form:"lat" binding:"required"`
		Lon    float64 `form:"lon" binding:"required"`
		Radius float64 `form:"radius"`
		Query  string  `form:"query"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		respondBadRequest(c, "Latitude and longitude are required")
		return
	}

	opts := FetchOptions{
		Lat:    req.Lat,
		Lon:    req.Lon,
		Radius: req.Radius,
		Query:  req.Query,
		Filters: map[string]string{
			"lat":    fmt.Sprintf("%.4f", req.Lat),
			"lon":    fmt.Sprintf("%.4f", req.Lon),
			"radius": fmt.Sprintf("%.1f", req.Radius),
		},
	}
	if req.Query != "" {
		opts.Entities = map[string]string{"query": req.Query}
	}

	h.fetchAndRespond(c, models.IntentNearby, opts)
}

// Search performs text search on articles
// GET /api/v1/news/search?query=climate+change
func (h *NewsHandler) Search(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		respondMissingParam(c, "Query parameter")
		return
	}

	result, intentResp, err := h.newsService.SearchWithIntent(query)
	if err != nil {
		respondInternalError(c, err.Error())
		return
	}

	// Build filters from named entities
	filters := buildNamedEntityFilters(intentResp.NamedEntities)

	response := gin.H{
		"articles": articlesToResponses(result.Articles),
		"metadata": models.NewResponseMetadata(
			len(result.Articles),
			result.TotalAvailable,
			query,
			filters,
		),
	}

	if intentResp.NamedEntities != nil {
		response["named_entities"] = intentResp.NamedEntities
	}

	c.JSON(http.StatusOK, response)
}

// GetArticleByID retrieves a single article by ID
// GET /api/v1/news/article/:id
func (h *NewsHandler) GetArticleByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondMissingParam(c, "Article ID")
		return
	}

	article, err := h.newsService.GetArticleByIDWithSummary(id)
	if err != nil {
		respondNotFound(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, article.ToResponse())
}

// GetStats returns statistics about the news database
// GET /api/v1/news/stats
func (h *NewsHandler) GetStats(c *gin.Context) {
	stats, err := h.newsService.GetArticleStats()
	if err != nil {
		respondInternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, stats)
}

// HealthCheck is a simple health check endpoint
// GET /api/v1/health
func (h *NewsHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "news-backend",
		"version": "1.0.0",
	})
}
