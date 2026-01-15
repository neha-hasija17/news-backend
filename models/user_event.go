package models

import (
	"time"
)

// UserEvent represents a user interaction with an article
type UserEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ArticleID string    `gorm:"index:idx_article_events" json:"article_id"`
	UserID    string    `gorm:"index:idx_user_events" json:"user_id"`
	EventType string    `gorm:"index:idx_event_type" json:"event_type"` // "view", "click", "share"
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `gorm:"index:idx_timestamp" json:"timestamp"`
}

// EventType constants
const (
	EventTypeView  = "view"
	EventTypeClick = "click"
	EventTypeShare = "share"
)

// GetEventWeight returns the weight for trending score calculation
func GetEventWeight(eventType string) float64 {
	switch eventType {
	case EventTypeView:
		return 1.0
	case EventTypeClick:
		return 2.0
	case EventTypeShare:
		return 3.0
	default:
		return 1.0
	}
}

// TrendingArticle represents an article with trending score
type TrendingArticle struct {
	Article
	TrendingScore float64 `json:"trending_score"`
	EventCount    int     `json:"event_count"`
}
