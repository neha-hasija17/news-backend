package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

/* =======================
   CONFIG
======================= */

const (
	DBName      = "news.db"
	GroqBaseURL = "https://api.groq.com/openai/v1"
)

/* =======================
   MODELS
======================= */

type Article struct {
	ID              string `gorm:"primaryKey"`
	Title           string
	Description     string
	URL             string
	PublicationDate time.Time
	SourceName      string
	Category        string
	RelevanceScore  float64
	Latitude        float64
	Longitude       float64
	LLMSummary      string
}

type UserEvent struct {
	ID        uint `gorm:"primaryKey"`
	ArticleID string
	EventType string
	Latitude  float64
	Longitude float64
	Timestamp time.Time
}

type IntentResponse struct {
	Intent   string            `json:"intent"`
	Entities map[string]string `json:"entities"`
}

/* =======================
   GLOBALS
======================= */

var db *gorm.DB

/* =======================
   DB
======================= */

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open(DBName), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&Article{}, &UserEvent{})
}

/* =======================
   DATA LOADER
======================= */

func loadData(path string) {
	var count int64
	db.Model(&Article{}).Count(&count)
	if count > 0 {
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	type RawArticle struct {
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

	var rawArticles []RawArticle
	json.Unmarshal(raw, &rawArticles)

	for _, r := range rawArticles {
		t, err := time.Parse("2006-01-02T15:04:05", r.PublicationDate)
		if err != nil {
			fmt.Printf("failed to parse date %s: %v\n", r.PublicationDate, err)
			continue
		}

		db.Create(&Article{
			ID:              r.ID,
			Title:           r.Title,
			Description:     r.Description,
			URL:             r.URL,
			PublicationDate: t,
			SourceName:      r.SourceName,
			Category:        strings.Join(r.Category, ","),
			RelevanceScore:  r.RelevanceScore,
			Latitude:        r.Latitude,
			Longitude:       r.Longitude,
		})
	}
}

/* =======================
   LLM
======================= */

func llmClient() *openai.Client {
	key := os.Getenv("GROQ_API_KEY")
	if key == "" {
		panic("GROQ_API_KEY missing")
	}
	cfg := openai.DefaultConfig(key)
	cfg.BaseURL = GroqBaseURL
	return openai.NewClientWithConfig(cfg)
}

func parseIntent(query string) IntentResponse {
	client := llmClient()

	prompt := `
Return ONLY valid JSON.
Extract:
- intent: category | source | search | nearby | score
- entities: category, source_name, query, location
`

	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: prompt},
			{Role: "user", Content: query},
		},
		Temperature: 0,
	})

	if err != nil {
		return IntentResponse{Intent: "search", Entities: map[string]string{"query": query}}
	}

	var out IntentResponse
	json.Unmarshal([]byte(resp.Choices[0].Message.Content), &out)
	if out.Intent == "" {
		out.Intent = "search"
		out.Entities = map[string]string{"query": query}
	}
	return out
}

func summarize(text string) string {
	client := llmClient()
	if len(text) < 20 {
		return "Summary unavailable."
	}

	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "You are a news summarization engine. Summarize the following text factually in one sentence. Do NOT add opinions, refusals, or safety warnings."},
			{Role: "user", Content: text},
		},
	})
	if err != nil {
		return "Summary unavailable."
	}
	return resp.Choices[0].Message.Content
}

/* =======================
   UTILS
======================= */

func distanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

/* =======================
   CORE LOGIC
======================= */

func fetchArticles(intent string, ent map[string]string, lat, lon float64) []Article {
	var arts []Article
	q := db.Model(&Article{})

	switch intent {
	case "category":
		q.Where("category LIKE ?", "%"+ent["category"]+"%").Find(&arts)
	case "source":
		q.Where("source_name LIKE ?", "%"+ent["source_name"]+"%").Find(&arts)
	case "score":
		q.Where("relevance_score > ?", 0.7).Find(&arts)
	case "nearby":
		q.Find(&arts)
		filtered := []Article{}
		for _, a := range arts {
			if distanceKm(lat, lon, a.Latitude, a.Longitude) <= 10 {
				filtered = append(filtered, a)
			}
		}
		arts = filtered
	default:
		search := strings.ToLower(ent["query"])

		// Generic queries → latest news feed
		if search == "latest news" || search == "recent news" || search == "today news" {
			q.Order("publication_date desc").Limit(5).Find(&arts)
			break
		}

		q.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			"%"+search+"%", "%"+search+"%",
		).Find(&arts)

		// Fallback: if no keyword match, return latest
		if len(arts) == 0 {
			q.Order("publication_date desc").Limit(5).Find(&arts)
		}

	}

	if intent == "search" {
		query := ent["query"]

		if isGenericQuery(query) {
			// Discovery mode → recency
			sort.Slice(arts, func(i, j int) bool {
				return arts[i].PublicationDate.After(arts[j].PublicationDate)
			})
		} else {
			// Intentional search → relevance
			sort.Slice(arts, func(i, j int) bool {
				scoreI := computeSearchScore(arts[i], query)
				scoreJ := computeSearchScore(arts[j], query)
				return scoreI > scoreJ
			})
		}
	}
	if len(arts) > 5 {
		arts = arts[:5]
	}

	var wg sync.WaitGroup
	for i := range arts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			arts[i].LLMSummary = summarize(arts[i].Description)
		}(i)
	}
	wg.Wait()

	return arts
}
func isGenericQuery(q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	return q == "latest news" ||
		q == "recent news" ||
		q == "today news" ||
		q == "news"
}

func computeSearchScore(a Article, query string) float64 {
	query = strings.ToLower(query)

	score := 0.0

	// Base relevance from dataset
	score += 0.6 * a.RelevanceScore

	// Title match boost
	if strings.Contains(strings.ToLower(a.Title), query) {
		score += 0.3
	}

	// Description match boost
	if strings.Contains(strings.ToLower(a.Description), query) {
		score += 0.1
	}

	return score
}

/* =======================
   HANDLERS
======================= */

func queryHandler(c *gin.Context) {
	q := c.Query("query")
	lat, lon := 0.0, 0.0
	fmt.Sscanf(c.Query("lat"), "%f", &lat)
	fmt.Sscanf(c.Query("lon"), "%f", &lon)

	intent := parseIntent(q)
	arts := fetchArticles(intent.Intent, intent.Entities, lat, lon)

	c.JSON(http.StatusOK, gin.H{
		"intent":   intent.Intent,
		"entities": intent.Entities,
		"articles": arts,
	})
}

/* =======================
   MAIN
======================= */

func main() {
	initDB()
	loadData("news_data.json")

	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.GET("/news/query", queryHandler)
	}

	r.Run(":8080")
}
