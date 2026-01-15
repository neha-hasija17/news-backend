package utils

import (
	"fmt"
	"math"
)

// HaversineDistance calculates the distance between two points on Earth using the Haversine formula
// Returns distance in kilometers
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const EarthRadiusKm = 6371.0
	
	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180
	
	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return EarthRadiusKm * c
}

// GeoHash creates a simple geohash for location clustering
// Used for caching trending data by location
func GeoHash(lat, lon float64, precision int) string {
	// Simple grid-based hash for caching
	// Divides world into grid cells
	latCell := int(math.Floor(lat*float64(precision)) + 180*float64(precision))
	lonCell := int(math.Floor(lon*float64(precision)) + 90*float64(precision))
	return string(rune(latCell)) + "_" + string(rune(lonCell))
}

// ValidateLocation checks if location coordinates are valid
func ValidateLocation(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("invalid latitude: must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return fmt.Errorf("invalid longitude: must be between -180 and 180")
	}
	return nil
}

// IsWithinRadius checks if a point is within a given radius from reference point
func IsWithinRadius(refLat, refLon, pointLat, pointLon, radius float64) bool {
	return HaversineDistance(refLat, refLon, pointLat, pointLon) <= radius
}
