package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"news-backend/models"
	"news-backend/services"

	"github.com/gin-gonic/gin"
)

type TrendingHandler struct {
	trendingService *services.TrendingService
}

// NewTrendingHandler creates a new trending handler
func NewTrendingHandler(trendingService *services.TrendingService) *TrendingHandler {
	return &TrendingHandler{
		trendingService: trendingService,
	}
}

// GetTrending retrieves trending news for a location
// GET /api/v1/trending?lat=37.4220&lon=-122.0840&radius=50&limit=5
func (h *TrendingHandler) GetTrending(c *gin.Context) {
	var req models.TrendingRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		respondBadRequest(c, "Latitude and longitude are required")
		return
	}

	// Get trending articles with summaries
	trendingArticles, cache, err := h.trendingService.GetTrendingNewsWithSummaries(
		req.Latitude,
		req.Longitude,
		req.Radius,
		req.Limit,
	)

	if err != nil {
		respondInternalError(c, err.Error())
		return
	}

	// Convert to response format
	articleResponses := make([]models.ArticleResponse, len(trendingArticles))
	for i, article := range trendingArticles {
		resp := article.Article.ToResponse()
		// Note: TrendingScore and EventCount are not in ArticleResponse
		// If needed, extend ArticleResponse or create TrendingArticleResponse
		articleResponses[i] = resp
	}

	response := models.TrendingResponse{
		Articles: articleResponses,
		Metadata: models.NewResponseMetadata(
			len(articleResponses),
			len(articleResponses), // For trending, total equals returned count
			"",                    // No query for trending
			map[string]string{
				"lat":    fmt.Sprintf("%.4f", req.Latitude),
				"lon":    fmt.Sprintf("%.4f", req.Longitude),
				"radius": fmt.Sprintf("%.1f", cache.RadiusKm),
			},
		),
		Location: cache.Location,
		RadiusKm: cache.RadiusKm,
	}

	if cache != nil {
		response.CachedAt = cache.CachedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	c.JSON(http.StatusOK, response)
}

// RecordEvent records a user interaction event
// POST /api/v1/trending/event
// Body: {"article_id": "...", "user_id": "...", "event_type": "view", "lat": 37.4220, "lon": -122.0840}
func (h *TrendingHandler) RecordEvent(c *gin.Context) {
	var req struct {
		ArticleID string  `json:"article_id" binding:"required"`
		UserID    string  `json:"user_id" binding:"required"`
		EventType string  `json:"event_type" binding:"required"`
		Lat       float64 `json:"lat" binding:"required"`
		Lon       float64 `json:"lon" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, err.Error())
		return
	}

	// Normalize event type
	eventType := strings.ToLower(req.EventType)

	err := h.trendingService.RecordUserEvent(
		req.ArticleID,
		req.UserID,
		eventType,
		req.Lat,
		req.Lon,
	)

	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Event recorded successfully",
	})
}

// GetEventStats returns statistics about user events
// GET /api/v1/trending/stats
func (h *TrendingHandler) GetEventStats(c *gin.Context) {
	stats, err := h.trendingService.GetEventStats()
	if err != nil {
		respondInternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, stats)
}

// InvalidateCache clears the trending cache
// POST /api/v1/trending/cache/invalidate
func (h *TrendingHandler) InvalidateCache(c *gin.Context) {
	h.trendingService.InvalidateCache()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Trending cache invalidated",
	})
}
