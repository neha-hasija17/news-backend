package handlers

import (
	"net/http"

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

// GetByCategory retrieves news by category using LLM to parse query
// GET /api/v1/news/category?query=Technology+news
func (h *NewsHandler) GetByCategory(c *gin.Context) {
	h.handleSearchWithIntent(c)
}

// GetBySource retrieves news by source using LLM to parse query
// GET /api/v1/news/source?query=Reuters+news
func (h *NewsHandler) GetBySource(c *gin.Context) {
	h.handleSearchWithIntent(c)
}

// GetByScore retrieves high-relevance articles using LLM to parse query
// GET /api/v1/news/score?query=top+trending+news
func (h *NewsHandler) GetByScore(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		query = "top trending news" // Default query for score-based retrieval
	}

	result, intentResp, err := h.newsService.SearchWithIntent(query)
	if err != nil {
		respondInternalError(c, err.Error())
		return
	}

	h.respondWithEntities(c, result, intentResp, query)
}

// GetNearby retrieves news near a location using LLM to parse query
// GET /api/v1/news/nearby?lat=37.4220&lon=-122.0840&radius=10&query=local+news
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

	if req.Query == "" {
		req.Query = "local news" // Default query for nearby
	}

	articles, intentResp, err := h.newsService.QueryWithIntent(req.Query, req.Lat, req.Lon, req.Radius)
	if err != nil {
		respondInternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"intent":   intentResp.Intent,
		"entities": intentResp.Entities,
		"articles": articlesToResponses(articles),
		"count":    len(articles),
		"location": map[string]interface{}{
			"lat":    req.Lat,
			"lon":    req.Lon,
			"radius": req.Radius,
		},
	})
}

// Search performs text search on articles using LLM to parse query
// GET /api/v1/news/search?query=climate+change
func (h *NewsHandler) Search(c *gin.Context) {
	h.handleSearchWithIntent(c)
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
