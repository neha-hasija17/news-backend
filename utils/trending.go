package utils

import (
	"math"
)

// =============================================================================
// Trending Score Utilities
// =============================================================================

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
