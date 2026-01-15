package utils

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		expected float64
		delta    float64 // Acceptable error margin in km
	}{
		{
			name:     "Same point returns zero",
			lat1:     37.7749,
			lon1:     -122.4194,
			lat2:     37.7749,
			lon2:     -122.4194,
			expected: 0,
			delta:    0.001,
		},
		{
			name:     "San Francisco to Los Angeles (~559 km)",
			lat1:     37.7749,
			lon1:     -122.4194,
			lat2:     34.0522,
			lon2:     -118.2437,
			expected: 559,
			delta:    5, // Allow 5km error
		},
		{
			name:     "New York to London (~5570 km)",
			lat1:     40.7128,
			lon1:     -74.0060,
			lat2:     51.5074,
			lon2:     -0.1278,
			expected: 5570,
			delta:    20,
		},
		{
			name:     "Short distance - 1 km apart",
			lat1:     37.7749,
			lon1:     -122.4194,
			lat2:     37.7839,
			lon2:     -122.4094,
			expected: 1.3,
			delta:    0.2,
		},
		{
			name:     "Antipodal points (~20000 km)",
			lat1:     0,
			lon1:     0,
			lat2:     0,
			lon2:     180,
			expected: 20015, // Half Earth circumference
			delta:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HaversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(result-tt.expected) > tt.delta {
				t.Errorf("HaversineDistance() = %v, expected %v (±%v)", result, tt.expected, tt.delta)
			}
		})
	}
}

func TestIsWithinRadius(t *testing.T) {
	// San Francisco coordinates
	sfLat, sfLon := 37.7749, -122.4194

	tests := []struct {
		name     string
		pointLat float64
		pointLon float64
		radius   float64
		expected bool
	}{
		{
			name:     "Same point is within any radius",
			pointLat: sfLat,
			pointLon: sfLon,
			radius:   1,
			expected: true,
		},
		{
			name:     "Oakland is within 20km of SF",
			pointLat: 37.8044,
			pointLon: -122.2712,
			radius:   20,
			expected: true,
		},
		{
			name:     "Los Angeles is not within 100km of SF",
			pointLat: 34.0522,
			pointLon: -118.2437,
			radius:   100,
			expected: false,
		},
		{
			name:     "Zero radius only matches exact point",
			pointLat: 37.7750,
			pointLon: -122.4194,
			radius:   0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWithinRadius(sfLat, sfLon, tt.pointLat, tt.pointLon, tt.radius)
			if result != tt.expected {
				t.Errorf("IsWithinRadius() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidateLocation(t *testing.T) {
	tests := []struct {
		name      string
		lat       float64
		lon       float64
		expectErr bool
	}{
		{"Valid coordinates", 37.7749, -122.4194, false},
		{"Valid edge - North Pole", 90, 0, false},
		{"Valid edge - South Pole", -90, 0, false},
		{"Valid edge - Date Line East", 0, 180, false},
		{"Valid edge - Date Line West", 0, -180, false},
		{"Invalid latitude too high", 91, 0, true},
		{"Invalid latitude too low", -91, 0, true},
		{"Invalid longitude too high", 0, 181, true},
		{"Invalid longitude too low", 0, -181, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocation(tt.lat, tt.lon)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateLocation() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
