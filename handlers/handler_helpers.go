package handlers

import (
	"fmt"
	"net/http"

	"news-backend/models"
	"news-backend/services"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// Response Helpers
// =============================================================================

// respondWithError sends a standardized error response
func respondWithError(c *gin.Context, code int, error, message string) {
	c.JSON(code, models.ErrorResponse{
		Error:   error,
		Message: message,
		Code:    code,
	})
}

// respondBadRequest sends a 400 error response
func respondBadRequest(c *gin.Context, message string) {
	respondWithError(c, http.StatusBadRequest, "Invalid request", message)
}

// respondMissingParam sends a 400 error for missing parameters
func respondMissingParam(c *gin.Context, param string) {
	respondWithError(c, http.StatusBadRequest, "Missing parameter", param+" is required")
}

// respondInternalError sends a 500 error response
func respondInternalError(c *gin.Context, message string) {
	respondWithError(c, http.StatusInternalServerError, "Internal error", message)
}

// respondNotFound sends a 404 error response
func respondNotFound(c *gin.Context, message string) {
	respondWithError(c, http.StatusNotFound, "Not found", message)
}

// =============================================================================
// Article Conversion Helpers
// =============================================================================

// articlesToResponses converts a slice of Articles to ArticleResponses
func articlesToResponses(articles []models.Article) []models.ArticleResponse {
	responses := make([]models.ArticleResponse, len(articles))
	for i, article := range articles {
		responses[i] = article.ToResponse()
	}
	return responses
}

// buildNamedEntityFilters creates a filter map from named entities
func buildNamedEntityFilters(entities *models.NamedEntities) map[string]string {
	filters := map[string]string{}
	if entities == nil {
		return filters
	}
	if len(entities.Locations) > 0 {
		filters["locations"] = fmt.Sprintf("%v", entities.Locations)
	}
	if len(entities.People) > 0 {
		filters["people"] = fmt.Sprintf("%v", entities.People)
	}
	if len(entities.Organizations) > 0 {
		filters["organizations"] = fmt.Sprintf("%v", entities.Organizations)
	}
	if len(entities.Events) > 0 {
		filters["events"] = fmt.Sprintf("%v", entities.Events)
	}
	return filters
}

// =============================================================================
// Common Handler Patterns
// =============================================================================

// FetchOptions contains optional parameters for fetching articles
type FetchOptions struct {
	Entities      map[string]string
	NamedEntities *models.NamedEntities
	Lat           float64
	Lon           float64
	Radius        float64
	Query         string
	Filters       map[string]string
}

// fetchAndRespond is a helper that handles the common pattern of:
// 1. Fetch articles with metadata
// 2. Enrich with summaries
// 3. Convert to response
// 4. Send JSON response with metadata
func (h *NewsHandler) fetchAndRespond(c *gin.Context, intent string, opts FetchOptions) {
	result, err := h.newsService.FetchArticlesWithMetadata(services.FetchParams{
		Intent:        intent,
		Entities:      opts.Entities,
		NamedEntities: opts.NamedEntities,
		Lat:           opts.Lat,
		Lon:           opts.Lon,
		Radius:        opts.Radius,
	})
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to fetch articles", err.Error())
		return
	}

	articles := h.newsService.EnrichWithSummaries(result.Articles)
	articleResponses := articlesToResponses(articles)

	c.JSON(http.StatusOK, gin.H{
		"articles": articleResponses,
		"metadata": models.NewResponseMetadata(
			len(articleResponses),
			result.TotalAvailable,
			opts.Query,
			opts.Filters,
		),
	})
}
