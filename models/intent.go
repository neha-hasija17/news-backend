package models

// Entities represents extracted entities from query
// Contains key-value pairs like: "query", "category", "source", "location", etc.
type Entities map[string]interface{}

// IntentResponse represents the LLM's analysis of user query
type IntentResponse struct {
	Intent   string   `json:"intent"`   // "category", "source", "search", "nearby", "score"
	Entities Entities `json:"entities"` // Extracted entities (people, organizations, locations, events, etc.)
}

// Intent types
const (
	IntentCategory = "category"
	IntentSource   = "source"
	IntentSearch   = "search"
	IntentNearby   = "nearby"
	IntentScore    = "score"
)

// NewsQueryRequest represents an incoming news query
type NewsQueryRequest struct {
	Query     string  `json:"query" form:"query" binding:"required"`
	Latitude  float64 `json:"lat" form:"lat"`
	Longitude float64 `json:"lon" form:"lon"`
	Radius    float64 `json:"radius" form:"radius"` // in km, optional
}

// NewsQueryResponse represents the response for a news query
type NewsQueryResponse struct {
	Intent   string            `json:"intent"`
	Entities Entities          `json:"entities"`
	Articles []ArticleResponse `json:"articles"`
	Count    int               `json:"count"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// TrendingRequest represents a request for trending news
type TrendingRequest struct {
	Latitude  float64 `json:"lat" form:"lat" binding:"required"`
	Longitude float64 `json:"lon" form:"lon" binding:"required"`
	Radius    float64 `json:"radius" form:"radius"` // in km, optional
	Limit     int     `json:"limit" form:"limit"`
}

// TrendingResponse represents trending news response
type TrendingResponse struct {
	Articles []ArticleResponse `json:"articles"`
	Metadata *ResponseMetadata `json:"metadata"`
	Location string            `json:"location"`
	RadiusKm float64           `json:"radius_km"`
	CachedAt string            `json:"cached_at,omitempty"`
}

// ResponseMetadata contains pagination and query information for API responses
type ResponseMetadata struct {
	Count          int               `json:"count"`             // Number of articles returned
	TotalAvailable int               `json:"total_available"`   // Total matching articles before limit
	Page           int               `json:"page"`              // Current page number
	PageSize       int               `json:"page_size"`         // Items per page
	Query          string            `json:"query,omitempty"`   // Original query string
	Filters        map[string]string `json:"filters,omitempty"` // Applied filters (category, source, etc.)
}

// NewResponseMetadata creates a new ResponseMetadata with defaults
func NewResponseMetadata(count, totalAvailable int, query string, filters map[string]string) *ResponseMetadata {
	return &ResponseMetadata{
		Count:          count,
		TotalAvailable: totalAvailable,
		Page:           1,
		PageSize:       count,
		Query:          query,
		Filters:        filters,
	}
}
