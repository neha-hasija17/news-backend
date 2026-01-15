package models

import (
	"encoding/json"
	"strings"
	"time"
)

// Article represents a news article in the database
// This is the core domain model with GORM tags for database operations
type Article struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	Title           string    `gorm:"index:idx_title" json:"title"`
	Description     string    `json:"description"`
	URL             string    `json:"url"`
	PublicationDate time.Time `gorm:"index:idx_pub_date" json:"publication_date"`
	SourceName      string    `gorm:"index:idx_source" json:"source_name"`
	Category        string    `gorm:"index:idx_category" json:"category"`
	RelevanceScore  float64   `gorm:"index:idx_relevance" json:"relevance_score"`
	Latitude        float64   `gorm:"index:idx_location" json:"latitude"`
	Longitude       float64   `gorm:"index:idx_location" json:"longitude"`
	LLMSummary      string    `json:"llm_summary,omitempty"`
	Distance        float64   `gorm:"-" json:"distance,omitempty"` // Computed, not stored
}



// ArticleResponse represents the API response structure
// Excludes internal ID, uses same shape for external consumers
type ArticleResponse struct {
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	URL             string    `json:"url"`
	PublicationDate time.Time `json:"publication_date"`
	SourceName      string    `json:"source_name"`
	Category        string    `json:"category"`
	RelevanceScore  float64   `json:"relevance_score"`
	LLMSummary      string    `json:"llm_summary"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	Distance        float64   `json:"distance,omitempty"`
}

// ToResponse converts an Article to ArticleResponse
func (a *Article) ToResponse() ArticleResponse {
	return ArticleResponse{
		Title:           a.Title,
		Description:     a.Description,
		URL:             a.URL,
		PublicationDate: a.PublicationDate,
		SourceName:      a.SourceName,
		Category:        a.Category,
		RelevanceScore:  a.RelevanceScore,
		LLMSummary:      a.LLMSummary,
		Latitude:        a.Latitude,
		Longitude:       a.Longitude,
		Distance:        a.Distance,
	}
}

// ArticleSortable interface implementation for generic sorting

// GetPublicationDateUnix returns publication date as Unix timestamp for sorting
func (a Article) GetPublicationDateUnix() int64 {
	return a.PublicationDate.Unix()
}

// GetRelevanceScore returns the relevance score for sorting
func (a Article) GetRelevanceScore() float64 {
	return a.RelevanceScore
}

// GetDistance returns the computed distance for sorting
func (a Article) GetDistance() float64 {
	return a.Distance
}

// GetID returns the article ID for score map lookups
func (a Article) GetID() string {
	return a.ID
}

// Locatable interface implementation for distance calculations

// GetLatitude returns the article's latitude
func (a Article) GetLatitude() float64 {
	return a.Latitude
}

// GetLongitude returns the article's longitude
func (a Article) GetLongitude() float64 {
	return a.Longitude
}

// SetDistance sets the computed distance (requires pointer receiver to modify)
func (a *Article) SetDistance(d float64) {
	a.Distance = d
}

// SearchSortable interface implementation

// GetTitle returns the article title for search scoring
func (a Article) GetTitle() string {
	return a.Title
}

// GetDescription returns the article description for search scoring
func (a Article) GetDescription() string {
	return a.Description
}

// UnmarshalJSON custom unmarshaler to handle JSON format differences
func (a *Article) UnmarshalJSON(data []byte) error {
	// Temporary struct matching JSON format
	var raw struct {
		ID              string   `json:"id"`
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		URL             string   `json:"url"`
		PublicationDate string   `json:"publication_date"`
		SourceName      string   `json:"source_name"`
		Category        []string `json:"category"`
		RelevanceScore  float64  `json:"relevance_score"`
		Latitude        float64  `json:"latitude"`
		Longitude       float64  `json:"longitude"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Parse publication date
	pubDate, err := time.Parse("2006-01-02T15:04:05", raw.PublicationDate)
	if err != nil {
		return err
	}

	// Assign to Article fields
	a.ID = raw.ID
	a.Title = raw.Title
	a.Description = raw.Description
	a.URL = raw.URL
	a.PublicationDate = pubDate
	a.SourceName = raw.SourceName
	a.Category = strings.Join(raw.Category, ",")
	a.RelevanceScore = raw.RelevanceScore
	a.Latitude = raw.Latitude
	a.Longitude = raw.Longitude

	return nil
}