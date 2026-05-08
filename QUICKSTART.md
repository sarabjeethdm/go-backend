# EDI Processing System - Quick Start Guide

## Prerequisites Check

Before starting, ensure you have the following installed:

```bash
# Check Go version (requires 1.21+)
go version

# Check Docker (recommended)
docker --version
docker-compose --version

# Optional: For local development
mongosh --version    # MongoDB client
redis-cli --version  # Redis client
```

## Quick Start (30 Seconds)

### Using Docker Compose (Recommended)

```bash
# Navigate to project directory
cd golang-backend-task

# Start all services (API, Worker, MongoDB, Redis, Prometheus)
docker-compose up -d

# Check if services are running
docker-compose ps

# View logs
docker-compose logs -f
```

**Services Started:**
- API Server: http://localhost:8080
- Worker: Background processing
- MongoDB: localhost:27017
- Redis: localhost:6379
- Prometheus: http://localhost:9090

---

## Testing the Complete System

### 1. Check Health

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "edi-processing-api"
}
```

### 2. Upload an EDI File

```bash
# Using the provided sample file
curl -X POST http://localhost:8080/jobs \
  -F "file=@sample.edi"
```

Expected response:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Job created successfully and queued for processing"
}
```

Save the `job_id` for the next steps!

### 3. Check Job Status

```bash
# Replace JOB_ID with your actual job ID
curl http://localhost:8080/jobs/550e8400-e29b-41d4-a716-446655440000
```

Expected response:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "retry_count": 0,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:05Z"
}
```

**Status values:**
- `pending` - Job queued, waiting for worker
- `processing` - Worker is processing the job
- `completed` - Job successfully completed
- `failed` - Job failed after max retries

### 4. Get Processing Result

```bash
# Replace JOB_ID with your actual job ID
curl http://localhost:8080/jobs/550e8400-e29b-41d4-a716-446655440000/result
```

Expected response:
```json
{
  "status": "completed",
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
      "member_id": "MEM789",
      "amount": 1200
    },
    {
      "claim_id": "CLM004",
      "member_id": "MEM123",
      "amount": 750
    },
    {
      "claim_id": "CLM005",
      "member_id": "MEM456",
      "amount": 350
    }
  ],
  "summary": {
    "total_claims": 5,
    "total_amount": 6600
  }
}
```

---

## Creating Your Own EDI File

### EDI File Format

Create a file with the following format:
```
CLAIM*CLM001*MEM123*2500
CLAIM*CLM002*MEM456*1800
CLAIM*CLM003*MEM789*1200
```

**Format:** `CLAIM*{claim_id}*{member_id}*{amount}`

- `CLAIM` - Record type (must be "CLAIM")
- `claim_id` - Unique claim identifier
- `member_id` - Member identifier  
- `amount` - Claim amount (numeric, can be decimal)

### Example: Create and Upload

```bash
# Create a test EDI file
cat > test.edi << EOF
CLAIM*CLM100*MEM001*5000
CLAIM*CLM101*MEM002*3500
CLAIM*CLM102*MEM001*1250
EOF

# Upload it
curl -X POST http://localhost:8080/jobs -F "file=@test.edi"

# Get the job_id from response and check status
curl http://localhost:8080/jobs/YOUR_JOB_ID

# Get result
curl http://localhost:8080/jobs/YOUR_JOB_ID/result
```

---

## Complete Workflow Example

```bash
#!/bin/bash

# 1. Upload file and capture job ID
RESPONSE=$(curl -s -X POST http://localhost:8080/jobs -F "file=@sample.edi")
JOB_ID=$(echo $RESPONSE | jq -r '.job_id')
echo "Job created: $JOB_ID"

# 2. Wait for processing (2-5 seconds typically)
echo "Waiting for processing..."
sleep 3

# 3. Check status
echo "Checking status..."
curl -s http://localhost:8080/jobs/$JOB_ID | jq

# 4. Get result
echo "Getting result..."
curl -s http://localhost:8080/jobs/$JOB_ID/result | jq
```

---

## Running Locally (Without Docker)

### 1. Start Dependencies

```bash
# Start MongoDB
docker run -d -p 27017:27017 --name mongodb mongo:7.0

# Start Redis
docker run -d -p 6379:6379 --name redis redis:7.2-alpine

# Or use docker-compose for just the dependencies
docker-compose up -d mongodb redis
```

### 2. Run API Server

```bash
# Terminal 1: Run API
go run cmd/api/main.go

# Or use Make
make run
```

### 3. Run Worker

```bash
# Terminal 2: Run Worker
go run cmd/worker/main.go

# Or use Make
make run-worker
```

### 4. Test the System

Use the same curl commands as above to upload files and check results.

---

## Monitoring

### View Logs

```bash
# All services
docker-compose logs -f

# API only
docker-compose logs -f api

# Worker only
docker-compose logs -f worker

# Follow with grep for specific job
docker-compose logs -f | grep "550e8400"
```

### Check Metrics

```bash
# API metrics
curl http://localhost:8080/metrics

# Worker metrics
curl http://localhost:9091/metrics

# Prometheus UI
open http://localhost:9090
```

### Database Inspection

```bash
# Connect to MongoDB
docker-compose exec mongodb mongosh edi_processing

# View all jobs
db.jobs.find().pretty()

# Count by status
db.jobs.aggregate([
  { $group: { _id: "$status", count: { $sum: 1 } } }
])

# Find specific job (replace JOB_ID)
db.jobs.findOne({ job_id: "550e8400-e29b-41d4-a716-446655440000" })

# Find recent jobs
db.jobs.find().sort({ created_at: -1 }).limit(10).pretty()
```

### Queue Inspection

```bash
# Connect to Redis
docker-compose exec redis redis-cli

# Check queue size
LLEN edi:jobs:queue

# View first 10 items in queue
LRANGE edi:jobs:queue 0 9

# View all keys
KEYS *
```

---

## Testing Features

### Test Retry Logic

```bash
# Create an invalid EDI file
cat > invalid.edi << EOF
INVALID*FORMAT*HERE
CLAIM*CLM001*MEM123*NOTANUMBER
BADLINE
EOF

# Upload it
curl -X POST http://localhost:8080/jobs -F "file=@invalid.edi"

# Watch worker logs - you'll see retries with exponential backoff
docker-compose logs -f worker
```

Expected behavior:
1. Attempt 1 fails → retry_count: 1, wait 2 seconds
2. Attempt 2 fails → retry_count: 2, wait 4 seconds
3. Attempt 3 fails → retry_count: 3, wait 8 seconds
4. Attempt 4 fails → status: "failed"

### Test Concurrent Jobs

```bash
# Upload multiple files simultaneously
for i in {1..10}; do
  curl -X POST http://localhost:8080/jobs -F "file=@sample.edi" &
done
wait

# Check queue size
docker-compose exec redis redis-cli LLEN edi:jobs:queue

# Watch worker process them
docker-compose logs -f worker
```

### Test Large File

```bash
# Create a file with 1000 claims
for i in {1..1000}; do
  echo "CLAIM*CLM$(printf %04d $i)*MEM$(($i % 100))*$((1000 + $i))"
done > large.edi

# Upload it
time curl -X POST http://localhost:8080/jobs -F "file=@large.edi"
```

---

## Performance Testing

### Load Test with Apache Bench

```bash
# Install Apache Bench (if not installed)
# macOS: brew install httpd
# Ubuntu: apt-get install apache2-utils

# Create a test file
echo "CLAIM*CLM001*MEM123*2500" > test.edi

# Run load test (100 requests, 10 concurrent)
ab -n 100 -c 10 \
  -p <(cat <<EOF
--boundary
Content-Disposition: form-data; name="file"; filename="test.edi"
Content-Type: application/octet-stream

CLAIM*CLM001*MEM123*2500
--boundary--
EOF
) \
  -T 'multipart/form-data; boundary=boundary' \
  http://localhost:8080/jobs
```

### Monitor System Resources

```bash
# Docker stats
docker stats

# Service resource usage
docker stats api worker mongodb redis
```

---

## Troubleshooting

### Services Not Starting

```bash
# Check if ports are in use
lsof -i :8080  # API
lsof -i :27017 # MongoDB
lsof -i :6379  # Redis

# Check Docker
docker-compose ps

# Restart services
docker-compose restart
```

### API Returns Errors

```bash
# Check API logs
docker-compose logs api

# Check if MongoDB is accessible
docker-compose exec mongodb mongosh --eval "db.adminCommand('ping')"

# Check if Redis is accessible
docker-compose exec redis redis-cli ping
```

### Jobs Not Processing

```bash
# Check worker logs
docker-compose logs worker

# Check if worker is running
docker-compose ps worker

# Check queue
docker-compose exec redis redis-cli LLEN edi:jobs:queue

# Restart worker
docker-compose restart worker
```

### Database Connection Issues

```bash
# Check MongoDB logs
docker-compose logs mongodb

# Test MongoDB connection
docker-compose exec mongodb mongosh edi_processing --eval "db.adminCommand('ping')"

# Check Redis logs
docker-compose logs redis

# Test Redis connection
docker-compose exec redis redis-cli ping
```

---

## Cleanup

### Remove Test Data

```bash
# Clear all jobs from MongoDB
docker-compose exec mongodb mongosh edi_processing \
  --eval "db.jobs.deleteMany({})"

# Clear Redis queue
docker-compose exec redis redis-cli DEL edi:jobs:queue
```

### Stop Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes all data)
docker-compose down -v
```

### Remove Docker Images

```bash
# Remove project images
docker rmi golang-backend-task-api
docker rmi golang-backend-task-worker

# Remove all unused images
docker image prune -a
```

---

## Scaling

### Scale Workers

```bash
# Run 5 worker instances
docker-compose up -d --scale worker=5

# Check worker instances
docker-compose ps worker

# View logs from all workers
docker-compose logs -f worker
```

### Scale API

```bash
# Run 3 API instances (requires load balancer)
docker-compose up -d --scale api=3

# Note: You'll need nginx or similar for load balancing
```

---

## Development Workflow

### Make Commands

```bash
# Build
make build        # Build API
make build-worker # Build Worker

# Run
make run          # Run API locally
make run-worker   # Run Worker locally

# Test
make test         # Run all tests
make test-unit    # Unit tests only
make test-coverage # With coverage

# Docker
make docker-build # Build Docker image
make docker-run   # Run in Docker

# Development
make fmt          # Format code
make lint         # Run linter
make clean        # Clean build artifacts
```

### Hot Reload (Development)

```bash
# Install air for hot reload
go install github.com/cosmtrek/air@latest

# Run with hot reload
air

# Or for worker
air -c .air.worker.toml
```

---

## Next Steps

1. ✅ **Explore API**: Check `API_DOCUMENTATION.md` for detailed API docs
2. ✅ **View Metrics**: Open http://localhost:9090 for Prometheus
3. ✅ **Read Architecture**: See `README.md` for system design
4. ✅ **Run Tests**: Execute `make test` to run test suite
5. ✅ **Scale System**: Try `docker-compose up -d --scale worker=5`

---

## Support & Documentation

- **API Reference**: `API_DOCUMENTATION.md`
- **Complete Guide**: `COMPLETE_GUIDE.md`
- **Project Overview**: `README.md`
- **File Content Changes**: `FILE_CONTENT_CHANGE.md`
- **Setup Instructions**: `SETUP.md`

---

**Happy Processing! 🚀**
