package utils

import (
	"sort"
	"strings"
)

// =============================================================================
// Scoring Weight Constants
// =============================================================================

// Text matching weights for search relevance
const (
	WeightTitleMatch       = 0.5  // Weight for exact phrase match in title
	WeightDescriptionMatch = 0.3  // Weight for exact phrase match in description
	WeightWordMatch        = 0.2  // Weight for individual word matches
	WeightTextScore        = 0.6  // Weight for text matching in combined score
	WeightRelevanceScore   = 0.4  // Weight for base relevance in combined score
)

// SortOrder defines the direction of sorting
type SortOrder int

const (
	Ascending SortOrder = iota
	Descending
)

// ArticleSortable is an interface for types that can be sorted
type ArticleSortable interface {
	GetPublicationDateUnix() int64
	GetRelevanceScore() float64
	GetDistance() float64
	GetID() string
}

// DistanceSortable extends ArticleSortable with location data for distance-based sorting
type DistanceSortable interface {
	ArticleSortable
	GetLatitude() float64
	GetLongitude() float64
	SetDistance(float64)
}

// SortField represents the field to sort by
type SortField string

const (
	SortByDate      SortField = "date"
	SortByScore     SortField = "score"
	SortByDistance  SortField = "distance"
)

// SortConfig holds sorting configuration
type SortConfig struct {
	Field SortField
	Order SortOrder
}

// SortArticles sorts a slice of articles based on the provided configuration
// Uses generics to work with any slice that implements ArticleSortable
func SortArticles[T ArticleSortable](articles []T, config SortConfig) {
	sort.Slice(articles, func(i, j int) bool {
		var less bool
		switch config.Field {
		case SortByDate:
			less = articles[i].GetPublicationDateUnix() < articles[j].GetPublicationDateUnix()
		case SortByScore:
			less = articles[i].GetRelevanceScore() < articles[j].GetRelevanceScore()
		case SortByDistance:
			less = articles[i].GetDistance() < articles[j].GetDistance()
		default:
			less = articles[i].GetPublicationDateUnix() < articles[j].GetPublicationDateUnix()
		}

		// Reverse if descending
		if config.Order == Descending {
			return !less
		}
		return less
	})
}

// SortByScoreMap sorts articles using a precomputed score map (for search relevance)
func SortByScoreMap[T ArticleSortable](articles []T, scores map[string]float64, order SortOrder) {
	sort.Slice(articles, func(i, j int) bool {
		less := scores[articles[i].GetID()] < scores[articles[j].GetID()]
		if order == Descending {
			return !less
		}
		return less
	})
}

// Common sort configurations
var (
	SortDateDesc  = SortConfig{Field: SortByDate, Order: Descending}
	SortDateAsc   = SortConfig{Field: SortByDate, Order: Ascending}
	SortScoreDesc = SortConfig{Field: SortByScore, Order: Descending}
	SortScoreAsc  = SortConfig{Field: SortByScore, Order: Ascending}
)

// SortByDistanceFrom calculates distances and sorts by nearest first
// Uses pointer constraint to allow modification of Distance field
func SortByDistanceFrom[T any, PT interface {
	*T
	DistanceSortable
}](items []T, refLat, refLon float64) {
	// Calculate distances using pointer to each element
	for i := range items {
		ptr := PT(&items[i])
		if ptr.GetDistance() == 0 {
			ptr.SetDistance(HaversineDistance(
				refLat, refLon,
				ptr.GetLatitude(),
				ptr.GetLongitude(),
			))
		}
	}
	// Sort by distance ascending (nearest first)
	sort.Slice(items, func(i, j int) bool {
		return PT(&items[i]).GetDistance() < PT(&items[j]).GetDistance()
	})
}

// =============================================================================
// Distance Filtering
// =============================================================================

// FilterByDistance filters items within a radius from a reference point
// and sets the Distance field on each item. Returns filtered slice.
func FilterByDistance[T any, PT interface {
	*T
	DistanceSortable
}](items []T, refLat, refLon, radius float64) []T {
	filtered := make([]T, 0, len(items))
	for i := range items {
		ptr := PT(&items[i])
		dist := HaversineDistance(refLat, refLon, ptr.GetLatitude(), ptr.GetLongitude())
		if dist <= radius {
			ptr.SetDistance(dist)
			filtered = append(filtered, items[i])
		}
	}
	return filtered
}

// FilterByDistanceWithPredicate filters items within a radius with an additional condition
func FilterByDistanceWithPredicate[T any, PT interface {
	*T
	DistanceSortable
}](items []T, refLat, refLon, radius float64, predicate func(PT) bool) []T {
	filtered := make([]T, 0, len(items))
	for i := range items {
		ptr := PT(&items[i])
		dist := HaversineDistance(refLat, refLon, ptr.GetLatitude(), ptr.GetLongitude())
		if dist <= radius && predicate(ptr) {
			ptr.SetDistance(dist)
			filtered = append(filtered, items[i])
		}
	}
	return filtered
}

// CalculateDistance calculates and sets distance for a single item
func CalculateDistance[T any, PT interface {
	*T
	DistanceSortable
}](item *T, refLat, refLon float64) float64 {
	ptr := PT(item)
	dist := HaversineDistance(refLat, refLon, ptr.GetLatitude(), ptr.GetLongitude())
	ptr.SetDistance(dist)
	return dist
}

// =============================================================================
// Search Relevance Sorting
// =============================================================================

// SearchSortable extends ArticleSortable with title and description for text matching
type SearchSortable interface {
	ArticleSortable
	GetTitle() string
	GetDescription() string
}

// SortBySearchRelevance sorts articles by combination of relevance_score and text matching
// As per requirement: "rank by a combination of relevance_score and text matching score"
func SortBySearchRelevance[T SearchSortable](items []T, query string) {
	scores := make(map[string]float64, len(items))
	queryLower := strings.ToLower(query)

	for i := range items {
		textScore := calculateTextMatchScore(items[i], queryLower)
		relevanceScore := items[i].GetRelevanceScore()
		// Combine: text matching weight + relevance score weight
		scores[items[i].GetID()] = textScore*WeightTextScore + relevanceScore*WeightRelevanceScore
	}

	SortByScoreMap(items, scores, Descending)
}

// calculateTextMatchScore calculates how well title/description matches the query
func calculateTextMatchScore[T SearchSortable](item T, queryLower string) float64 {
	title := strings.ToLower(item.GetTitle())
	desc := strings.ToLower(item.GetDescription())

	score := 0.0

	// Exact phrase match in title (highest weight)
	if strings.Contains(title, queryLower) {
		score += WeightTitleMatch
	}

	// Exact phrase match in description
	if strings.Contains(desc, queryLower) {
		score += WeightDescriptionMatch
	}

	// Individual word matches
	words := strings.Fields(queryLower)
	if len(words) > 0 {
		matchedWords := 0
		for _, word := range words {
			if strings.Contains(title, word) || strings.Contains(desc, word) {
				matchedWords++
			}
		}
		// Normalize to WeightWordMatch range based on word match percentage
		score += WeightWordMatch * float64(matchedWords) / float64(len(words))
	}

	return score // Returns 0.0 to 1.0
}
