package models

// NamedEntities represents structured named entities extracted from query
type NamedEntities struct {
	People        []string `json:"people,omitempty"`        // Person names (e.g., "Elon Musk")
	Organizations []string `json:"organizations,omitempty"` // Companies/orgs (e.g., "Twitter", "Tesla")
	Locations     []string `json:"locations,omitempty"`     // Places (e.g., "Palo Alto", "New York")
	Events        []string `json:"events,omitempty"`        // Events (e.g., "acquisition", "election")
}

// HasEntities returns true if any named entities are present
func (ne *NamedEntities) HasEntities() bool {
	return len(ne.People) > 0 || len(ne.Organizations) > 0 ||
		len(ne.Locations) > 0 || len(ne.Events) > 0
}

// IntentResponse represents the LLM's analysis of user query
type IntentResponse struct {
	Intent        string            `json:"intent"`         // "category", "source", "search", "nearby", "score"
	Entities      map[string]string `json:"entities"`       // Generic extracted entities
	NamedEntities *NamedEntities    `json:"named_entities"` // Structured named entities
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
	Intent        string            `json:"intent"`
	Entities      map[string]string `json:"entities"`
	NamedEntities *NamedEntities    `json:"named_entities,omitempty"`
	Articles      []ArticleResponse `json:"articles"`
	Count         int               `json:"count"`
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
	Articles      []ArticleResponse  `json:"articles"`
	Metadata      *ResponseMetadata  `json:"metadata"`
	Location      string             `json:"location"`
	RadiusKm      float64            `json:"radius_km"`
	CachedAt      string             `json:"cached_at,omitempty"`
}

// ResponseMetadata contains pagination and query information for API responses
type ResponseMetadata struct {
	Count          int               `json:"count"`           // Number of articles returned
	TotalAvailable int               `json:"total_available"` // Total matching articles before limit
	Page           int               `json:"page"`            // Current page number
	PageSize       int               `json:"page_size"`       // Items per page
	Query          string            `json:"query,omitempty"` // Original query string
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
