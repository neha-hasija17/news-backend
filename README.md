# News Backend - Contextual News Data Retrieval System

A backend system for contextual news retrieval with LLM-powered intent parsing, **named entity extraction**, geospatial search, and trending news functionality.

## 🚀 Features

- **Multiple Retrieval Strategies**: Category, source, score-based, nearby, and full-text search
- **LLM-Powered Named Entity Extraction**: Extracts people, organizations, locations, and events from queries
- **LLM-Powered Summarization**: Uses Groq/OpenAI to generate concise article summaries
- **Enhanced Search Relevance**: Entity-based matching for improved article ranking
- **Geospatial Search**: Find news articles near any location using Haversine distance
- **Trending News**: Location-based trending articles with user engagement tracking
- **Smart Caching**: Optimized performance with geospatial cache segmentation
- **RESTful API**: Clean, versioned API endpoints with comprehensive error handling
- **Production Ready**: Proper logging, middleware, and error handling
- **Clean Architecture**: Service layer handles all business logic, handlers focus on HTTP

## 🏗️ Architecture

This project follows a **layered architecture** with clear separation of concerns:

- **Handlers** (HTTP layer): Request validation, response formatting
- **Services** (Business logic): LLM operations, data processing, caching
- **Database** (Data layer): GORM-based SQLite persistence

All LLM operations (intent parsing, summarization) are encapsulated in the service layer, keeping handlers thin and focused on HTTP concerns. See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design documentation.

## 📋 Prerequisites

- Go 1.24 or higher
- SQLite3
- Groq API key (or OpenAI API key)

## 🛠️ Installation

1. **Clone the repository**
```bash
cd /Users/neha.hasija/work_v2/news-backend
```

2. **Install dependencies**
```bash
go mod download
```

3. **Configure environment**
```bash
cp .env.example .env
# Edit .env and add your API keys
```

4. **Set your API key**
```bash
export GROQ_API_KEY=your_groq_api_key_here
# OR for OpenAI
export OPENAI_API_KEY=your_openai_api_key_here
export LLM_PROVIDER=openai
```

## 🏃 Running the Application

### Option 1: Docker (Recommended)

```bash
# 1. Set your API key in .env file
cp .env.example .env
# Edit .env and add your GROQ_API_KEY

# 2. Build and run with Docker Compose
docker-compose up -d

# 3. Check logs
docker-compose logs -f

# 4. Stop the service
docker-compose down
```

See [DOCKER.md](DOCKER.md) for detailed Docker deployment instructions.

### Option 2: Run Locally

```bash
# Run the server
go run main.go

# The server will start on http://localhost:8080
```

The application will automatically:
- Initialize the SQLite database
- Load news data from `news_data.json`
- Seed sample user events for trending functionality

## 📚 API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Health Check
```bash
GET /api/v1/health
```

### News Endpoints

#### 1. Category-Based Search
```bash
GET /api/v1/news/category?category=<category>

# Example:
curl "http://localhost:8080/api/v1/news/category?category=Technology"
```

#### 2. Source-Based Search
```bash
GET /api/v1/news/source?source=<source_name>

# Example:
curl "http://localhost:8080/api/v1/news/source?source=Reuters"
```

#### 3. High-Relevance Articles
```bash
GET /api/v1/news/score

# Example:
curl "http://localhost:8080/api/v1/news/score"
```

#### 4. Nearby Articles (Geospatial)
```bash
GET /api/v1/news/nearby?lat=<latitude>&lon=<longitude>&radius=<km>

# Example:
curl "http://localhost:8080/api/v1/news/nearby?lat=37.4220&lon=-122.0840&radius=10"
```

#### 5. Text Search
```bash
GET /api/v1/news/search?query=<search_term>

# Example:
curl "http://localhost:8080/api/v1/news/search?query=climate+change"

# Example with named entities:
curl "http://localhost:8080/api/v1/news/search?query=Elon+Musk+Twitter+acquisition"
```

**Named Entity Extraction**: The search endpoint uses LLM to extract named entities from queries:
- **People**: Person names (e.g., "Elon Musk", "Joe Biden")
- **Organizations**: Companies, institutions (e.g., "Twitter", "United Nations")
- **Locations**: Cities, countries (e.g., "Palo Alto", "New York")
- **Events**: Specific happenings (e.g., "acquisition", "summit")

Response includes extracted entities:
```json
{
  "query": "Elon Musk Twitter acquisition",
  "named_entities": {
    "people": ["Elon Musk"],
    "organizations": ["Twitter"],
    "events": ["acquisition"]
  },
  "articles": [...],
  "count": 5
}
```

Articles are ranked using entity matching (40% weight) combined with traditional search relevance (60% weight).

#### 6. Get Article by ID
```bash
GET /api/v1/news/article/:id

# Example:
curl "http://localhost:8080/api/v1/news/article/19aaddc0-7508-4659-9c32-2216107f8604"
```

#### 7. Database Statistics
```bash
GET /api/v1/news/stats

# Example:
curl "http://localhost:8080/api/v1/news/stats"
```

### Trending Endpoints

#### 1. Get Trending News
```bash
GET /api/v1/trending?lat=<latitude>&lon=<longitude>&radius=<km>&limit=<n>

# Example:
curl "http://localhost:8080/api/v1/trending?lat=37.4220&lon=-122.0840&radius=50&limit=5"
```

#### 2. Record User Event
```bash
POST /api/v1/trending/event
Content-Type: application/json

{
  "article_id": "article-uuid",
  "user_id": "user-id",
  "event_type": "view",  // "view", "click", or "share"
  "lat": 37.4220,
  "lon": -122.0840
}

# Example:
curl -X POST "http://localhost:8080/api/v1/trending/event" \
  -H "Content-Type: application/json" \
  -d '{"article_id": "19aaddc0-7508-4659-9c32-2216107f8604", "user_id": "user123", "event_type": "view", "lat": 37.4220, "lon": -122.0840}'
```

#### 3. Trending Statistics
```bash
GET /api/v1/trending/stats

# Example:
curl "http://localhost:8080/api/v1/trending/stats"
```

#### 4. Invalidate Cache
```bash
POST /api/v1/trending/cache/invalidate

# Example:
curl -X POST "http://localhost:8080/api/v1/trending/cache/invalidate"
```

## 📊 Response Format

### Standard Article Response
```json
{
  "articles": [
    {
      "title": "Article Title",
      "description": "Article description...",
      "url": "https://example.com/article",
      "publication_date": "2025-03-26T04:46:55Z",
      "source_name": "News Source",
      "category": "Technology",
      "relevance_score": 0.86,
      "llm_summary": "AI-generated summary of the article...",
      "latitude": 37.4220,
      "longitude": -122.0840,
      "distance": 5.2  // Only for nearby queries
    }
  ],
  "count": 5
}
```

### Error Response
```json
{
  "error": "Error type",
  "message": "Detailed error message",
  "code": 400
}
```

## 🔧 Configuration

Environment variables (see `.env.example`):

| Variable               | Description                | Default                  |
| ---------------------- | -------------------------- | ------------------------ |
| `PORT`                 | Server port                | 8080                     |
| `DB_PATH`              | SQLite database path       | news.db                  |
| `LLM_PROVIDER`         | LLM provider (groq/openai) | groq                     |
| `GROQ_API_KEY`         | Groq API key               | Required if using Groq   |
| `OPENAI_API_KEY`       | OpenAI API key             | Required if using OpenAI |
| `INTENT_MODEL`         | Model for intent parsing   | llama-3.3-70b-versatile  |
| `SUMMARY_MODEL`        | Model for summarization    | llama-3.1-8b-instant     |
| `DEFAULT_RADIUS`       | Default search radius (km) | 10.0                     |
| `MAX_ARTICLES`         | Max articles to return     | 5                        |
| `SCORE_THRESHOLD`      | Min relevance score        | 0.7                      |
| `TRENDING_CACHE_TTL`   | Cache TTL (seconds)        | 300                      |
| `TRENDING_RADIUS`      | Trending search radius     | 50.0                     |
| `TRENDING_TIME_WINDOW` | Event window (hours)       | 24                       |

## 🧪 Testing the API

### Using curl

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Category search
curl "http://localhost:8080/api/v1/news/category?category=Business"

# Nearby articles
curl "http://localhost:8080/api/v1/news/nearby?lat=28.6139&lon=77.2090&radius=20"

# Trending news
curl "http://localhost:8080/api/v1/trending?lat=28.6139&lon=77.2090&radius=50&limit=5"

# Get statistics
curl http://localhost:8080/api/v1/news/stats
curl http://localhost:8080/api/v1/trending/stats
```

### Sample Queries

1. **Category**: `/api/v1/news/category?category=Technology`
2. **Source**: `/api/v1/news/source?source=Reuters`
3. **Search**: `/api/v1/news/search?query=climate+change`
4. **Nearby**: `/api/v1/news/nearby?lat=37.4220&lon=-122.0840&radius=10`
5. **High Score**: `/api/v1/news/score`

## 🚀 Production Deployment

### Build for production
```bash
go build -o news-backend main.go
```

### Run in production
```bash
# Set environment variables
export GROQ_API_KEY=your_api_key
export PORT=8080

# Run the binary
./news-backend
```

### Docker (optional)
```dockerfile
FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o news-backend main.go
CMD ["./news-backend"]
```

## 📈 Performance Considerations

- **Caching**: Trending results are cached by location grid with configurable TTL
- **Concurrent Summarization**: LLM summaries are generated concurrently with semaphore limiting
- **Database Indexing**: Key fields (category, source, date, location) are indexed
- **Batch Processing**: Data loading uses batch inserts for efficiency

## 🔒 Security

- CORS middleware enabled for cross-origin requests
- Error handler catches panics and prevents information leakage
- Input validation on all endpoints
- Rate limiting placeholder (implement with Redis in production)

## 📝 Notes 

This implementation demonstrates:

1. **Clean Architecture**: Proper separation of concerns with handlers, services, and data layers
2. **Production Patterns**: Middleware, error handling, logging, configuration management
3. **LLM Integration**: Automatic article summarization for enhanced user experience
4. **Geospatial Processing**: Haversine distance calculation for location-based features
5. **Caching Strategy**: Location-grid-based caching for trending data
6. **RESTful Design**: Versioned API with clear endpoint structure
7. **Scalability**: Concurrent processing, batch operations, efficient database queries
8. **Code Quality**: Well-documented, modular, testable code structure

## 🤝 API Usage Examples

See the examples in the API Documentation section above, or test using the provided curl commands.

