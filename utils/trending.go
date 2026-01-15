package utils

import (
	"math"
)

// =============================================================================
// Trending Score Utilities
// =============================================================================

// IsGenericQuery checks if a query is generic (like "latest news")
func IsGenericQuery(query string) bool {
	genericQueries := []string{
		"latest news",
		"recent news",
		"today news",
		"current news",
		"news",
		"top news",
		"breaking news",
		"news today",
		"today's news",
	}

	for _, generic := range genericQueries {
		if query == generic {
			return true
		}
	}
	return false
}

// ComputeTrendingScore calculates trending score based on events
func ComputeTrendingScore(eventCount int, totalWeight float64, recencyFactor float64) float64 {
	// Trending score = (event count * average weight * recency factor)
	if eventCount == 0 {
		return 0
	}

	avgWeight := totalWeight / float64(eventCount)
	return float64(eventCount) * avgWeight * recencyFactor
}

// CalculateRecencyFactor calculates a decay factor based on time
// More recent events get higher scores
func CalculateRecencyFactor(hoursAgo float64) float64 {
	// Exponential decay: e^(-t/12)
	// Half-life of 12 hours
	return math.Exp(-hoursAgo / 12.0)
}
