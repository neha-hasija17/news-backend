package utils

import (
	"math"
	"testing"
)

func TestComputeTrendingScore(t *testing.T) {
	tests := []struct {
		name          string
		eventCount    int
		totalWeight   float64
		recencyFactor float64
		expected      float64
	}{
		{
			name:          "Zero events returns zero",
			eventCount:    0,
			totalWeight:   10,
			recencyFactor: 1.0,
			expected:      0,
		},
		{
			name:          "Basic calculation",
			eventCount:    10,
			totalWeight:   20,
			recencyFactor: 1.0,
			expected:      20, // 10 * (20/10) * 1.0 = 20
		},
		{
			name:          "With recency decay",
			eventCount:    10,
			totalWeight:   20,
			recencyFactor: 0.5,
			expected:      10, // 10 * (20/10) * 0.5 = 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeTrendingScore(tt.eventCount, tt.totalWeight, tt.recencyFactor)
			if result != tt.expected {
				t.Errorf("ComputeTrendingScore() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateRecencyFactor(t *testing.T) {
	tests := []struct {
		name     string
		hoursAgo float64
		minValue float64
		maxValue float64
	}{
		{
			name:     "Just now (0 hours)",
			hoursAgo: 0,
			minValue: 0.99,
			maxValue: 1.0,
		},
		{
			name:     "12 hours ago (half-life)",
			hoursAgo: 12,
			minValue: 0.35,
			maxValue: 0.40, // e^(-1) ≈ 0.368
		},
		{
			name:     "24 hours ago",
			hoursAgo: 24,
			minValue: 0.13,
			maxValue: 0.15, // e^(-2) ≈ 0.135
		},
		{
			name:     "48 hours ago",
			hoursAgo: 48,
			minValue: 0.01,
			maxValue: 0.03, // e^(-4) ≈ 0.018
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRecencyFactor(tt.hoursAgo)
			if result < tt.minValue || result > tt.maxValue {
				t.Errorf("CalculateRecencyFactor(%v) = %v, expected between %v and %v",
					tt.hoursAgo, result, tt.minValue, tt.maxValue)
			}
		})
	}

	// Test monotonic decrease
	t.Run("Monotonically decreasing", func(t *testing.T) {
		prev := math.MaxFloat64
		for hours := 0.0; hours <= 48; hours += 6 {
			current := CalculateRecencyFactor(hours)
			if current >= prev {
				t.Errorf("Recency factor should decrease over time: f(%v)=%v >= f(prev)=%v",
					hours, current, prev)
			}
			prev = current
		}
	})
}
