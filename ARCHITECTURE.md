# News Backend - Architecture Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT (REST API)                               │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                               GIN ROUTER                                     │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   Logger    │  │     CORS     │  │ ErrorHandler │  │    Recovery      │  │
│  └─────────────┘  └──────────────┘  └──────────────┘  └──────────────────┘  │
│                            middleware/middleware.go                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    ▼                                   ▼
┌───────────────────────────────────┐   ┌───────────────────────────────────┐
│         NEWS HANDLER              │   │       TRENDING HANDLER            │
│   handlers/news_handler.go        │   │   handlers/trending_handler.go    │
│                                   │   │                                   │
│  • GET /news/category             │   │  • GET /trending                  │
│  • GET /news/source               │   │  • POST /trending/event           │
│  • GET /news/score                │   │  • GET /trending/stats            │
│  • GET /news/nearby               │   │  • POST /trending/cache/invalidate│
│  • GET /news/search               │   │                                   │
│  • GET /news/article/:id          │   │  HTTP Request/Response Handling   │
│                                   │   │  (No LLM dependency)              │
│  HTTP Request/Response Handling   │   │                                   │
│  (No LLM dependency)              │   │                                   │
└───────────────────┬───────────────┘   └───────────────────┬───────────────┘
                    │                                       │
                    │     handlers/handler_helpers.go       │
                    │   (shared error responses, utils)     │
                    │                                       │
                    ▼                                       ▼
┌───────────────────────────────────┐   ┌───────────────────────────────────┐
│          NEWS SERVICE             │   │       TRENDING SERVICE            │
│    services/news_service.go       │   │   services/trending_service.go    │
│    services/query_helpers.go      │   │                                   │
│                                   │   │  • Trending score calculation     │
│  • Intent-based article fetching  │   │  • Location-based caching         │
│  • Text search with entities      │   │  • User event aggregation         │
│  • Distance filtering             │   │  • LLM summary enrichment         │
│  • Sorting strategies             │   │  • Fallback to relevance score    │
│  • LLM intent parsing             │   │                                   │
│  • LLM summary generation         │   │  Depends on: LLMService           │
│                                   │   │                                   │
│  Depends on: LLMService           │   │                                   │
└───────────────────┬───────────────┘   └───────────────────┬───────────────┘
                    │                                       │
                    └─────────────────┬─────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LLM SERVICE                                     │
│                          services/llm_service.go                             │
│                                                                              │
│  ┌─────────────────────────────┐    ┌─────────────────────────────────────┐ │
│  │      Intent Parsing         │    │         Summary Generation          │ │
│  │  (Groq/OpenAI LLM API)      │    │       (Groq/OpenAI LLM API)         │ │
│  │                             │    │                                     │ │
│  │  Query → Intent + Entities  │    │  Article Description → Summary     │ │
│  │  + Named Entity Extraction  │    │  (Cached in sync.Map)              │ │
│  └─────────────────────────────┘    └─────────────────────────────────────┘ │
│                                                                              │
│                          prompts/prompts.go                                  │
│                     (System prompts for LLM)                                 │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                DATABASE                                      │
│                           database/database.go                               │
│                                                                              │
│  ┌─────────────────────────────┐    ┌─────────────────────────────────────┐ │
│  │         Articles            │    │          User Events               │ │
│  │   (SQLite via GORM)         │    │      (SQLite via GORM)             │ │
│  │                             │    │                                     │ │
│  │  • id, title, description   │    │  • article_id, user_id             │ │
│  │  • category, source_name    │    │  • event_type (view/click/share)   │ │
│  │  • relevance_score          │    │  • latitude, longitude             │ │
│  │  • latitude, longitude      │    │  • timestamp                       │ │
│  │  • publication_date         │    │                                     │ │
│  └─────────────────────────────┘    └─────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Example:

Let's trace a real request through the system:

```
User: GET /api/v1/news/search?query=climate+change

    ┌──────────────────────────────────────────────────────────────────┐
    │ 1. NewsHandler.Search()                                          │
    │    - Validates query parameter                                   │
    │    - Delegates to NewsService                                    │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 2. NewsService.SearchWithIntent()                                │
    │    - Calls LLMService.ParseIntent() internally                   │
    │    - Extracts: intent="search", entities, named_entities         │
    │    - Named entities: people, organizations, locations, events    │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 3. NewsService.FetchArticlesWithMetadata()                       │
    │    - Queries SQLite: LIKE '%climate%' OR LIKE '%change%'         │
    │    - Returns matching articles                                   │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 4. utils.SortBySearchRelevance()                                 │
    │    - Calculates text match score (title/description matching)    │
    │    - Combines with relevance_score (60% text + 40% relevance)    │
    │    - Sorts by combined score descending                          │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 5. NewsService.EnrichWithSummaries()                             │
    │    - Calls LLMService.GenerateSummariesBatch()                   │
    │    - Generates 1-sentence summaries for each article             │
    │    - Caches summaries to avoid duplicate LLM calls               │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 6. NewsHandler returns JSON Response                             │
    │    - articles[], metadata, named_entities                        │
    └──────────────────────────────────────────────────────────────────┘
```

---

## Example: How Trending Works

Trending is location-aware - users in San Francisco see different trending news than users in New York:

```
User: GET /api/v1/trending?lat=37.77&lon=-122.41&radius=50

    ┌──────────────────────────────────────────────────────────────────┐
    │ 1. TrendingHandler.GetTrending()                                 │
    │    - Validates lat/lon/radius parameters                         │
    │    - Delegates to TrendingService                                │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 2. TrendingService.GetTrendingNewsWithSummaries()                │
    │    - Check cache (sync.Map with grid-based location keys)        │
    │    - If cached: return immediately                               │
    └──────────────────────────────────────────────────────────────────┘
                                    │ (cache miss)
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 3. calculateTrendingScores()                                     │
    │    - Fetch UserEvents from last 24 hours                         │
    │    - Filter events within radius (IsWithinRadius)                │
    │    - Group events by article_id                                  │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 4. For each article with events:                                 │
    │    - Calculate event weights (view=1, click=2, share=3)          │
    │    - Apply recency decay: e^(-hours/12)                          │
    │    - Compute trending score                                      │
    │    - Boost by relevance_score and proximity                      │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 5. Enrich with LLM summaries (within service)                    │
    │    - Calls LLMService.GenerateSummary() for missing summaries    │
    │    - Sort by trending_score descending                           │
    │    - Cache result with location key                              │
    │    - Return top N articles                                       │
    └──────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌──────────────────────────────────────────────────────────────────┐
    │ 6. TrendingHandler returns JSON Response                         │
    │    - articles[], metadata, location, cached_at                   │
    └──────────────────────────────────────────────────────────────────┘
```

---


```go
switch params.Intent {
case models.IntentCategory: // "tech news" → fetchByCategory
case models.IntentSource:   // "from Reuters" → fetchBySource
case models.IntentSearch:   // "climate change" → fetchBySearch
case models.IntentNearby:   // "near me" → fetchNearby
case models.IntentScore:    // "top stories" → fetchByScore
}
```



## Location-Based Caching (Trending)

**The Problem:** Trending calculations are expensive (DB queries + scoring). Users 1km apart shouldn't trigger separate calculations.

**The Solution:** Grid-based cache keys - users in the same ~5km cell share cached results:

```go
func getCacheKey(lat, lon, radius float64) string {
    latCell := int(lat / 0.05)  // ~5km grid
    lonCell := int(lon / 0.05)
    return fmt.Sprintf("trending_%d_%d_%d", latCell, lonCell, radiusCell)
}
```

**Why it matters:** 100 users in downtown SF hit cache once, not 100 times.

## Configurable Scoring Weights

**The Problem:** Hard-coded scoring makes tuning search relevance a code change. Magic numbers like `0.3` scattered everywhere.

**The Solution:** Named constants that document what each factor contributes:

```go
const (
    WeightTitleMatch       = 0.5  // Title matches are most valuable
    WeightDescriptionMatch = 0.3  // Body text matters, but less
    WeightWordMatch        = 0.2  // Partial word matches get some credit
    WeightTextScore        = 0.6  // Text relevance dominates
    WeightRelevanceScore   = 0.4  // Base relevance as secondary factor
)
```

**Why it matters:** Product says "prioritize title matches more" → change one constant, not hunt through code.

---

## API Endpoints Summary

| Endpoint                            | Method | Description                      |
| ----------------------------------- | ------ | -------------------------------- |
| `/api/v1/health`                    | GET    | Health check                     |
| `/api/v1/news/category`             | GET    | Filter by category               |
| `/api/v1/news/source`               | GET    | Filter by source                 |
| `/api/v1/news/score`                | GET    | High relevance articles          |
| `/api/v1/news/nearby`               | GET    | Location-based + optional search |
| `/api/v1/news/search`               | GET    | Text search with LLM intent      |
| `/api/v1/news/article/:id`          | GET    | Single article                   |
| `/api/v1/news/stats`                | GET    | Database statistics              |
| `/api/v1/trending`                  | GET    | Trending by location             |
| `/api/v1/trending/event`            | POST   | Record user interaction          |
| `/api/v1/trending/stats`            | GET    | Event statistics                 |
| `/api/v1/trending/cache/invalidate` | POST   | Clear trending cache             |

## Technology Stack

| Component     | Technology                    |
| ------------- | ----------------------------- |
| Language      | Go 1.21+ (with Generics)      |
| Web Framework | Gin                           |
| ORM           | GORM                          |
| Database      | SQLite                        |
| LLM           | Groq (llama-3.3-70b) / OpenAI |
| Caching       | In-memory (sync.Map)          |

---

## Trade-offs & Future Improvements

This is a take-home assignment, so I made pragmatic choices. Here's what I'd revisit for production:

### What I'd Change

| Current Choice             | Production Alternative     | Why                                                |
| -------------------------- | -------------------------- | -------------------------------------------------- |
| SQLite                     | PostgreSQL                 | Concurrent writes, better geo queries with PostGIS |
| In-memory cache            | Redis                      | Survives restarts, shared across instances         |
| sync.Map                   | Proper cache with eviction | Memory bounds, LRU eviction                        |
| LLM for every intent parse | Cache common queries       | Cost + latency reduction                           |

### What I Deliberately Kept Simple

- **Single binary deployment**: No microservices overhead for an assignment
- **SQLite**: Zero setup, portable, fast enough for demo data
- **Inline error handling**: No custom error types - keeps code readable

### If I Had More Time

1. **Rate limiting**: Per-IP limits on LLM-heavy endpoints
2. **Structured logging**: JSON logs with request IDs for debugging
3. **Graceful shutdown**: Drain connections before stopping
4. **Integration tests**: End-to-end API testing with test database
