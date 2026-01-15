package utils

import (
	"testing"
)

// mockArticle implements ArticleSortable and SearchSortable for testing
type mockArticle struct {
	id          string
	pubDateUnix int64
	score       float64
	distance    float64
	lat         float64
	lon         float64
	title       string
	description string
}

func (m mockArticle) GetID() string                 { return m.id }
func (m mockArticle) GetPublicationDateUnix() int64 { return m.pubDateUnix }
func (m mockArticle) GetRelevanceScore() float64    { return m.score }
func (m mockArticle) GetDistance() float64          { return m.distance }
func (m mockArticle) GetLatitude() float64          { return m.lat }
func (m mockArticle) GetLongitude() float64         { return m.lon }
func (m *mockArticle) SetDistance(d float64)        { m.distance = d }
func (m mockArticle) GetTitle() string              { return m.title }
func (m mockArticle) GetDescription() string        { return m.description }

func TestSortArticles_ByDate(t *testing.T) {
	articles := []mockArticle{
		{id: "1", pubDateUnix: 100},
		{id: "2", pubDateUnix: 300},
		{id: "3", pubDateUnix: 200},
	}

	// Sort descending (newest first)
	SortArticles(articles, SortDateDesc)

	if articles[0].id != "2" || articles[1].id != "3" || articles[2].id != "1" {
		t.Errorf("SortDateDesc failed: got order %s, %s, %s", articles[0].id, articles[1].id, articles[2].id)
	}

	// Sort ascending (oldest first)
	SortArticles(articles, SortDateAsc)

	if articles[0].id != "1" || articles[1].id != "3" || articles[2].id != "2" {
		t.Errorf("SortDateAsc failed: got order %s, %s, %s", articles[0].id, articles[1].id, articles[2].id)
	}
}

func TestSortArticles_ByScore(t *testing.T) {
	articles := []mockArticle{
		{id: "low", score: 0.3},
		{id: "high", score: 0.9},
		{id: "mid", score: 0.6},
	}

	SortArticles(articles, SortScoreDesc)

	if articles[0].id != "high" || articles[1].id != "mid" || articles[2].id != "low" {
		t.Errorf("SortScoreDesc failed: got order %s, %s, %s", articles[0].id, articles[1].id, articles[2].id)
	}
}

func TestSortByDistanceFrom(t *testing.T) {
	// Reference point: San Francisco
	refLat, refLon := 37.7749, -122.4194

	articles := []mockArticle{
		{id: "LA", lat: 34.0522, lon: -118.2437},       // ~559 km
		{id: "Oakland", lat: 37.8044, lon: -122.2712},  // ~13 km
		{id: "Seattle", lat: 47.6062, lon: -122.3321},  // ~1094 km
	}

	SortByDistanceFrom[mockArticle](articles, refLat, refLon)

	// Should be sorted: Oakland (nearest), LA, Seattle (farthest)
	if articles[0].id != "Oakland" {
		t.Errorf("Expected Oakland first, got %s", articles[0].id)
	}
	if articles[1].id != "LA" {
		t.Errorf("Expected LA second, got %s", articles[1].id)
	}
	if articles[2].id != "Seattle" {
		t.Errorf("Expected Seattle third, got %s", articles[2].id)
	}

	// Check that distances were set
	if articles[0].distance == 0 {
		t.Error("Distance was not set on articles")
	}
}

func TestFilterByDistance(t *testing.T) {
	refLat, refLon := 37.7749, -122.4194 // San Francisco

	articles := []mockArticle{
		{id: "Oakland", lat: 37.8044, lon: -122.2712},  // ~13 km
		{id: "LA", lat: 34.0522, lon: -118.2437},       // ~559 km
		{id: "Palo Alto", lat: 37.4419, lon: -122.143}, // ~47 km
	}

	// Filter within 50km
	filtered := FilterByDistance[mockArticle](articles, refLat, refLon, 50)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 articles within 50km, got %d", len(filtered))
	}

	// Check IDs
	ids := map[string]bool{}
	for _, a := range filtered {
		ids[a.id] = true
	}
	if !ids["Oakland"] || !ids["Palo Alto"] {
		t.Error("Expected Oakland and Palo Alto in filtered results")
	}
	if ids["LA"] {
		t.Error("LA should not be in filtered results")
	}
}

func TestSortBySearchRelevance(t *testing.T) {
	articles := []mockArticle{
		{id: "no-match", title: "Weather Report", description: "Sunny day ahead", score: 0.9},
		{id: "title-match", title: "Climate Change Impact", description: "Environmental news", score: 0.5},
		{id: "both-match", title: "Climate Summit", description: "Leaders discuss climate", score: 0.3},
	}

	SortBySearchRelevance(articles, "climate")

	// "both-match" should be first (matches in title AND description)
	// Even though "no-match" has higher base score, text matching matters more
	if articles[0].id != "both-match" {
		t.Errorf("Expected 'both-match' first, got %s", articles[0].id)
	}
}

func TestCalculateTextMatchScore(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		query       string
		minScore    float64
		maxScore    float64
	}{
		{
			name:        "Exact match in title",
			title:       "Climate change is real",
			description: "Some other text",
			query:       "climate change",
			minScore:    0.5, // WeightTitleMatch
			maxScore:    1.0,
		},
		{
			name:        "Match in description only",
			title:       "News today",
			description: "Climate change affects us all",
			query:       "climate change",
			minScore:    0.3, // WeightDescriptionMatch
			maxScore:    0.6,
		},
		{
			name:        "No match",
			title:       "Sports update",
			description: "Football game results",
			query:       "climate",
			minScore:    0.0,
			maxScore:    0.01,
		},
		{
			name:        "Partial word match",
			title:       "Climate news",
			description: "Weather update",
			query:       "climate change",
			minScore:    0.1, // Only "climate" matches (1/2 words = 0.1)
			maxScore:    0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article := mockArticle{title: tt.title, description: tt.description}
			score := calculateTextMatchScore(article, tt.query)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateTextMatchScore() = %v, expected between %v and %v",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}
