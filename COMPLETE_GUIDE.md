# Complete EDI Processing System - Deployment & Usage Guide

## 🎯 Overview

This is a **production-ready Go-based backend system** that processes EDI-like files asynchronously with complete Docker Compose support.

### Features Implemented ✅

#### Core Requirements
- ✅ **REST API** with Gin framework
- ✅ **Async Processing** via Redis queue
- ✅ **MongoDB** for persistence
- ✅ **Docker Compose** deployment
- ✅ **Health endpoints**
- ✅ **Structured logging** (JSON format)
- ✅ **Graceful shutdown**
- ✅ **Environment configuration**
- ✅ **Comprehensive error handling**

#### Bonus Features
- ✅ **Retry mechanism** (3 attempts with exponential backoff)
- ✅ **Job failure handling** with detailed error messages
- ✅ **Unit & Integration tests** (28+ test cases)
- ✅ **Prometheus metrics** (9 metrics, 15 alerts)
- ✅ **Swagger API documentation**
- ✅ **CI/CD ready** with Makefile automation

---

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Architecture](#architecture)
3. [API Endpoints](#api-endpoints)
4. [Deployment Options](#deployment-options)
5. [Testing](#testing)
6. [Monitoring](#monitoring)
7. [Troubleshooting](#troubleshooting)

---

## 🚀 Quick Start

### Option 1: Docker Compose (Recommended for Local Development)

```bash
# Clone/navigate to project
cd golang-backend-task

# Start all services
docker-compose up -d

# Check health
curl http://localhost:8080/health

# View logs
docker-compose logs -f api
docker-compose logs -f worker

# Stop services
docker-compose down
```

**Services Started:**
- API Server: http://localhost:8080
- MongoDB: localhost:27017
- Redis: localhost:6379
- Prometheus: http://localhost:9090

### Option 2: Local Development

```bash
# Start dependencies
docker-compose up -d mongodb redis

# Run API
go run cmd/api/main.go

# Run Worker (in another terminal)
go run cmd/worker/main.go
```

---

## 🏗️ Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────────────┐
│    API Service      │ ← POST /jobs (upload file)
│   (Port 8080)       │ ← GET /jobs/{id}
│                     │ ← GET /jobs/{id}/result
└──────┬──────┬───────┘
       │      │
       │      └─────────────┐
       │                    │
       ▼                    ▼
┌────────────┐      ┌──────────────┐
│  MongoDB   │      │    Redis     │
│            │      │   (Queue)    │
└────────────┘      └──────┬───────┘
                           │
                           │ Pop job
                           ▼
                    ┌──────────────┐
                    │    Worker    │
                    │   Service    │
                    │ (3 replicas) │
                    └──────┬───────┘
                           │
                           │ Save result
                           ▼
                    ┌──────────────┐
                    │   MongoDB    │
                    └──────────────┘
```

### Components

1. **API Service**: REST API for file uploads and job management
2. **Worker Service**: Async job processor with retry logic
3. **MongoDB**: Stores jobs and results
4. **Redis**: Job queue for async processing
5. **Prometheus**: Metrics and monitoring (optional)

### Data Flow

1. Client uploads EDI file via `POST /jobs`
2. API validates file, creates job record in MongoDB
3. API pushes job ID to Redis queue
4. Worker pulls job from queue
5. Worker parses EDI file, calculates summary
6. Worker saves result to MongoDB
7. Client retrieves result via `GET /jobs/{id}/result`

---

## 📡 API Endpoints

### 1. Health Check

```bash
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0"
}
```

### 2. Create Job (Upload EDI File)

```bash
POST /jobs
Content-Type: multipart/form-data

file: <EDI file>
```

**Example:**
```bash
curl -X POST http://localhost:8080/jobs \
  -F "file=@sample.edi"
```

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "message": "Job created successfully"
}
```

### 3. Get Job Status

```bash
GET /jobs/{job_id}
```

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:05Z"
}
```

**Status values:** `pending`, `processing`, `completed`, `failed`

### 4. Get Job Result

```bash
GET /jobs/{job_id}/result
```

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "claims": [
    {
      "claim_id": "CLM001",
      "member_id": "MEM123",
      "amount": 2500
    },
    {
      "claim_id": "CLM002",
      "member_id": "MEM456",
      "amount": 1800
    },
    {
      "claim_id": "CLM003",
      "member_id": "MEM123",
      "amount": 300
    }
  ],
  "summary": {
    "total_claims": 3,
    "total_amount": 4600
  }
}
```

### EDI File Format

```
CLAIM*CLM001*MEM123*2500
CLAIM*CLM002*MEM456*1800
CLAIM*CLM003*MEM123*300
```

Format: `CLAIM*{claim_id}*{member_id}*{amount}`

---

## 🐳 Deployment Options

### Docker Compose

**Pros:**
- Simplest setup
- Great for local development
- Fast iteration
- Production-ready for small-scale deployments

**Services:**
```yaml
- api: API server
- worker: Job processor
- mongodb: Database
- redis: Queue
- prometheus: Metrics (optional)
```

**Commands:**
```bash
docker-compose up -d          # Start all services
docker-compose ps             # Check status
docker-compose logs -f api    # View API logs
docker-compose down -v        # Stop and remove volumes
docker-compose restart worker # Restart worker

# Scale workers
docker-compose up -d --scale worker=5
```

### Local Development

**Pros:**
- Fast development cycle
- Easy debugging
- Direct code access

**Setup:**
```bash
# Start dependencies
docker-compose up -d mongodb redis

# Run API
go run cmd/api/main.go

# Run worker (in another terminal)
go run cmd/worker/main.go
```

---

## 🧪 Testing

### Run Tests

```bash
# All tests
make test

# Unit tests only
make test-unit

# Integration tests (requires services running)
make test-integration

# With coverage
make test-coverage
```

### Test Files

1. **Unit Tests:**
   - `internal/api/handlers_test.go` - API handler tests
   - `internal/parser/edi_parser_test.go` - EDI parser tests
   - `internal/worker/processor_test.go` - Worker tests

2. **Integration Tests:**
   - `tests/integration_test.go` - End-to-end tests

3. **Test Fixtures:**
   - `tests/fixtures/valid.edi` - Valid EDI file
   - `tests/fixtures/invalid.edi` - Invalid EDI file

### Manual Testing

```bash
# 1. Upload a file
JOB_ID=$(curl -s -X POST http://localhost:8080/jobs \
  -F "file=@sample.edi" | jq -r '.job_id')
echo "Job ID: $JOB_ID"

# 2. Check status (wait a few seconds)
curl http://localhost:8080/jobs/$JOB_ID | jq

# 3. Get result
curl http://localhost:8080/jobs/$JOB_ID/result | jq
```

---

## 📊 Monitoring

### Metrics Exposed

**API Metrics (port 8080/metrics):**
- `api_requests_total` - Total API requests
- `api_request_duration_seconds` - Request latency
- `jobs_total{status="created|completed|failed"}` - Job counts
- `active_jobs` - Currently processing jobs

**Worker Metrics (port 9091/metrics):**
- `job_processing_duration_seconds` - Processing time
- `redis_queue_size` - Queue depth
- `job_retries_total` - Retry count
- `transactions_processed_total` - Claims processed

### Prometheus

**Access:** http://localhost:9090

**Useful Queries:**
```promql
# Job success rate
rate(jobs_total{status="completed"}[5m]) / rate(jobs_total[5m])

# Average processing time
rate(job_processing_duration_seconds_sum[5m]) / rate(job_processing_duration_seconds_count[5m])

# Queue size
redis_queue_size

# API request rate
rate(api_requests_total[5m])
```

### Grafana Dashboard

Import `monitoring/dashboards/grafana-dashboard.json` for pre-built visualizations.

---

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `API_PORT` | API server port | `8080` |
| `MONGO_URI` | MongoDB connection string | `mongodb://localhost:27017` |
| `MONGO_DATABASE` | MongoDB database name | `edi_processing` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_DB` | Redis database number | `0` |
| `LOG_LEVEL` | Logging level | `info` |
| `WORKER_POLL_INTERVAL` | Worker polling interval | `1s` |
| `MAX_RETRIES` | Max job retries | `3` |

### Configuration Files

- `.env` - Local environment (Docker Compose)
- `docker-compose.yml` - Docker Compose services
- `config/config.go` - Go configuration loader

---

## 🐛 Troubleshooting

### Common Issues

#### 1. API Not Responding

```bash
# Check if API is running
docker-compose ps api

# Check logs
docker-compose logs api

# Check health
curl http://localhost:8080/health
```

#### 2. Jobs Stuck in Pending

```bash
# Check worker status
docker-compose ps worker

# Check worker logs
docker-compose logs worker

# Check Redis queue
docker-compose exec redis redis-cli LLEN job_queue
```

#### 3. MongoDB Connection Issues

```bash
# Check MongoDB
docker-compose ps mongodb

# Test connection
docker-compose exec mongodb mongosh --eval "db.adminCommand('ping')"
```

### Debug Mode

Enable debug logging:
```bash
# Docker Compose
docker-compose exec api sh -c 'export LOG_LEVEL=debug'

# Or edit .env file and restart
LOG_LEVEL=debug
docker-compose restart api worker
```

---

## 📝 Swagger Documentation

Access Swagger UI:
```
http://localhost:8080/swagger/index.html
```

Or view the OpenAPI spec:
```bash
cat docs/swagger.yaml
```

---

## 🚀 Production Checklist

- [ ] Configure persistent volumes for MongoDB
- [ ] Set up reverse proxy (nginx, traefik) for external access
- [ ] Enable TLS/HTTPS
- [ ] Configure authentication/authorization
- [ ] Set up log aggregation (ELK, Loki)
- [ ] Configure alerting (PagerDuty, Slack)
- [ ] Set up backup for MongoDB
- [ ] Configure Redis persistence (AOF/RDB)
- [ ] Scale workers with Docker Compose
- [ ] Set up CI/CD pipeline
- [ ] Add rate limiting
- [ ] Configure CORS properly
- [ ] Set up health checks in load balancer
- [ ] Monitor resource usage (CPU, memory)

---

## 📚 Additional Documentation

- [README.md](README.md) - Project overview
- [API_DOCUMENTATION.md](API_DOCUMENTATION.md) - Detailed API reference
- [QUICKSTART.md](QUICKSTART.md) - Quick start guide
- [MONITORING.md](MONITORING.md) - Monitoring and metrics guide
- [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) - Technical details

---

## 🤝 Support

For issues or questions:
1. Check logs first
2. Review troubleshooting section
3. Check GitHub issues
4. Review Swagger documentation

---

## 📄 License

MIT License

---

**Built with ❤️ using Go, MongoDB, Redis, and Docker**
