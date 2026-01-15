# Docker Deployment Guide

This guide explains how to run the News Backend service using Docker.

## 🐳 Prerequisites

- Docker (version 20.10 or higher)
- Docker Compose (version 1.29 or higher)
- Groq API key (or OpenAI API key)

## 🚀 Quick Start

### 1. Set up environment variables

Create a `.env` file from the example:

```bash
cp .env.example .env
```

Edit `.env` and add your API key:

```bash
GROQ_API_KEY=your_groq_api_key_here
```

### 2. Build and run with Docker Compose

```bash
docker-compose up -d
```

This will:
- Build the Docker image
- Start the container
- Expose the service on port 8080
- Create a persistent volume for the database

### 3. Verify the service is running

```bash
curl http://localhost:8080/api/v1/health
```

## 📦 Docker Commands

### Build the image manually

```bash
docker build -t news-backend:latest .
```

### Run the container manually

```bash
docker run -d \
  --name news-backend \
  -p 8080:8080 \
  -e GROQ_API_KEY=your_key_here \
  -v news-db:/data \
  -v $(pwd)/news_data.json:/root/news_data.json:ro \
  news-backend:latest
```

### View logs

```bash
docker-compose logs -f news-backend
```

Or with plain Docker:

```bash
docker logs -f news-backend
```

### Stop the service

```bash
docker-compose down
```

### Stop and remove volumes (⚠️ deletes database)

```bash
docker-compose down -v
```

## 🔧 Configuration

All configuration is done through environment variables in the `docker-compose.yml` file or `.env` file.

### Key Environment Variables

| Variable             | Default                   | Description                               |
| -------------------- | ------------------------- | ----------------------------------------- |
| `PORT`               | `8080`                    | Server port                               |
| `DB_PATH`            | `/data/news.db`           | Database file path                        |
| `LLM_PROVIDER`       | `groq`                    | LLM provider (groq/openai)                |
| `GROQ_API_KEY`       | -                         | Groq API key (required if using Groq)     |
| `OPENAI_API_KEY`     | -                         | OpenAI API key (required if using OpenAI) |
| `INTENT_MODEL`       | `llama-3.3-70b-versatile` | Model for intent parsing                  |
| `SUMMARY_MODEL`      | `llama-3.1-8b-instant`    | Model for summarization                   |
| `DEFAULT_RADIUS`     | `10.0`                    | Default search radius (km)                |
| `MAX_ARTICLES`       | `5`                       | Max articles per response                 |
| `TRENDING_CACHE_TTL` | `300`                     | Cache TTL in seconds                      |

### Using OpenAI Instead of Groq

Edit `docker-compose.yml` and uncomment the OpenAI configuration:

```yaml
environment:
  - LLM_PROVIDER=openai
  - OPENAI_API_KEY=${OPENAI_API_KEY}
```

## 📊 Monitoring

### Health Check

The container includes a health check that runs every 30 seconds:

```bash
docker inspect --format='{{.State.Health.Status}}' news-backend
```

### Container Stats

```bash
docker stats news-backend
```

## 🗄️ Data Persistence

The database is stored in a Docker volume named `news-db`. This ensures your data persists even if the container is stopped or removed.

To backup the database:

```bash
docker run --rm -v news-db:/data -v $(pwd):/backup alpine tar czf /backup/news-db-backup.tar.gz -C /data .
```

To restore from backup:

```bash
docker run --rm -v news-db:/data -v $(pwd):/backup alpine tar xzf /backup/news-db-backup.tar.gz -C /data
```

## 🔄 Updating the Application

### With Docker Compose

```bash
docker-compose down
docker-compose build
docker-compose up -d
```

### Manual Update

```bash
docker stop news-backend
docker rm news-backend
docker build -t news-backend:latest .
docker run -d --name news-backend -p 8080:8080 -e GROQ_API_KEY=your_key_here news-backend:latest
```

## 🐛 Troubleshooting

### View detailed logs

```bash
docker-compose logs --tail=100 news-backend
```

### Access container shell

```bash
docker exec -it news-backend sh
```

### Check if port is available

```bash
lsof -i :8080
```

### Rebuild from scratch

```bash
docker-compose down -v
docker-compose build --no-cache
docker-compose up -d
```

## 🌐 API Endpoints

Once running, the API is available at:

- Health: `http://localhost:8080/api/v1/health`
- Root: `http://localhost:8080/`
- API Docs: See README.md for full endpoint list

## 📝 Notes

- The SQLite database is stored in `/data/news.db` inside the container
- The `news_data.json` file is mounted read-only
- The container runs as root (can be changed for security in production)
- The image uses Alpine Linux for a smaller footprint (~50MB)

## 🚀 Production Deployment

For production deployments, consider:

1. **Use environment-specific compose files**:
   ```bash
   docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
   ```

2. **Set resource limits**:
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '1'
         memory: 512M
   ```

3. **Use secrets for API keys** instead of environment variables

4. **Enable TLS/HTTPS** with a reverse proxy (nginx, traefik)

5. **Implement log aggregation** (ELK stack, Loki, etc.)

6. **Add monitoring** (Prometheus, Grafana)
