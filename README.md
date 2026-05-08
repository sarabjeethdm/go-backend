# EDI Processing Backend System

A production-ready Go-based backend system for processing EDI (Electronic Data Interchange) files asynchronously. The system consists of an API service and worker service, using MongoDB for persistence and Redis for job queuing, with automatic retry logic and exponential backoff.

## Features

### Core Features
- **REST API**: Upload EDI files and retrieve processing results
- **Asynchronous Processing**: Background job processing via Redis queue
- **Automatic Retries**: Failed jobs retry up to 3 times with exponential backoff (2s, 4s, 8s)
- **MongoDB Storage**: Persistent storage for jobs and results
- **Docker Compose**: Full stack deployment with a single command
- **Health Checks**: API health endpoint for monitoring
- **Structured Logging**: JSON-formatted logs throughout
- **Graceful Shutdown**: Clean shutdown for both API and worker services
- **Environment Config**: Configurable via environment variables

### Bonus Features
- **Comprehensive Tests**: 28+ unit and integration tests
- **Prometheus Metrics**: 9 metrics with 15 alert rules
- **Swagger Documentation**: Complete OpenAPI specification
- **Job Failure Handling**: Detailed error tracking and reporting

## Architecture

```
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│              │       │              │       │              │
│   API/Web    │──────▶│    Redis     │──────▶│    Worker    │
│   Service    │ Push  │    Queue     │ Poll  │   Service    │
│              │       │              │       │              │
└──────────────┘       └──────────────┘       └──────┬───────┘
                                                      │
                                                      │ Update
                                                      ▼
                                              ┌──────────────┐
                                              │   MongoDB    │
                                              │   (Jobs)     │
                                              └──────────────┘
```

## Project Structure

```
golang-backend-task/
├── cmd/
│   └── worker/
│       └── main.go                 # Worker service entry point
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration management
│   ├── logger/
│   │   └── logger.go               # Structured logging
│   ├── models/
│   │   └── models.go               # Data models (Job, Claim, Result)
│   ├── storage/
│   │   └── storage.go              # MongoDB operations
│   ├── queue/
│   │   └── queue.go                # Redis queue operations
│   ├── parser/
│   │   └── edi_parser.go           # EDI file parser
│   └── worker/
│       └── processor.go            # Job processing logic
├── go.mod                          # Go dependencies
├── go.sum                          # Dependency checksums
├── .env.example                    # Example environment variables
├── Makefile                        # Build and run commands
└── README.md                       # This file
```

## Prerequisites

- Go 1.21 or higher
- MongoDB 4.4 or higher
- Redis 6.0 or higher

## Installation

1. Clone the repository:
```bash
cd /Users/sarabjeet.9353gmail.com/Documents/DEV/golang-backend-task
```

2. Install dependencies:
```bash
go mod download
```

3. Copy the example environment file:
```bash
cp .env.example .env
```

4. Update the `.env` file with your configuration.

## Configuration

Configuration is loaded from environment variables. All settings have sensible defaults.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MONGODB_URI` | MongoDB connection string | `mongodb://localhost:27017` |
| `MONGODB_DATABASE` | MongoDB database name | `edi_processor` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `REDIS_PASSWORD` | Redis password | `` |
| `REDIS_DB` | Redis database number | `0` |
| `WORKER_MAX_RETRIES` | Maximum retry attempts | `3` |
| `WORKER_POLL_INTERVAL` | Queue poll interval (seconds) | `1` |
| `WORKER_INITIAL_BACKOFF` | Initial backoff for retries (seconds) | `2` |
| `WORKER_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout (seconds) | `30` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |

## Running the Worker

### Using Make

```bash
# Build the worker
make build-worker

# Run the worker
make run-worker

# Run with custom environment
MONGODB_URI=mongodb://custom:27017 make run-worker
```

### Using Go directly

```bash
# Run directly
go run cmd/worker/main.go

# Build and run
go build -o bin/worker cmd/worker/main.go
./bin/worker
```

### Using Docker (optional)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o worker cmd/worker/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/worker .
CMD ["./worker"]
```

## EDI File Format

The worker expects EDI files in the following format:

```
CLAIM*CLM001*MEM123*2500
CLAIM*CLM002*MEM456*3000
CLAIM*CLM003*MEM789*1500
```

**Format**: `CLAIM*claim_id*member_id*amount`

- **CLAIM**: Record type (must be "CLAIM")
- **claim_id**: Unique claim identifier
- **member_id**: Member identifier
- **amount**: Claim amount (numeric, non-negative)

## Job Processing Flow

1. **Dequeue**: Worker polls Redis queue for job IDs
2. **Fetch**: Retrieves job details from MongoDB
3. **Validate**: Checks job status (must be "pending")
4. **Update**: Sets status to "processing"
5. **Parse**: Parses EDI file content
6. **Store**: Saves parsed result to MongoDB
7. **Complete**: Updates status to "completed" or "failed"

### Retry Logic

- Failed jobs are automatically retried up to 3 times
- Exponential backoff between retries:
  - Retry 1: 2 seconds
  - Retry 2: 4 seconds
  - Retry 3: 8 seconds
- After max retries, job is marked as "failed"

## Monitoring

The worker outputs structured JSON logs that can be ingested by log aggregation systems.

### Log Fields

- `timestamp`: ISO 8601 timestamp
- `level`: Log level (info, warn, error)
- `message`: Log message
- `job_id`: Job identifier (when applicable)
- `duration`: Processing duration (when applicable)
- `error`: Error message (when applicable)

### Example Log Output

```json
{
  "timestamp": "2024-01-15T10:30:00.000Z",
  "level": "info",
  "message": "Job completed successfully",
  "job_id": "65a5b1234567890abcdef123",
  "duration": "250ms",
  "total_claims": 3,
  "total_amount": 7000
}
```

## Graceful Shutdown

The worker handles shutdown signals gracefully:

1. Receives SIGTERM or SIGINT signal
2. Stops accepting new jobs from queue
3. Waits for active jobs to complete
4. Enforces shutdown timeout (default 30s)
5. Closes database and queue connections
6. Exits cleanly

## Testing

The project includes comprehensive unit tests, integration tests, and test fixtures for both the API and worker components.

### Running Tests

```bash
# Run all tests
make test

# Run only unit tests (no external dependencies required)
make test-unit

# Run integration tests (requires running services)
make test-integration

# Run tests with coverage report
make test-coverage
# Opens coverage.html in your browser

# Run tests with verbose output
make test-verbose
```

### Test Structure

```
golang-backend-task/
├── internal/
│   ├── api/
│   │   └── handlers_test.go        # API handler unit tests
│   └── worker/
│       └── processor_test.go       # Worker processor unit tests  
└── tests/
    ├── integration_test.go         # End-to-end integration tests
    └── fixtures/
        ├── valid.edi               # Valid EDI test file
        └── invalid.edi             # Invalid EDI test file
```

### Test Coverage

- **API Handler Tests** (`internal/api/handlers_test.go`):
  - Health check endpoint
  - Job creation with file upload
  - Job status retrieval
  - Result retrieval
  - Error handling (invalid inputs, database errors, queue errors)
  - Edge cases (empty files, large files, invalid job IDs)

- **Worker Processor Tests** (`internal/worker/processor_test.go`):
  - Successful job processing
  - Parse failure handling
  - Retry logic with exponential backoff
  - Max retries exceeded
  - Job state validation
  - Storage error handling

- **Integration Tests** (`tests/integration_test.go`):
  - End-to-end job processing workflow
  - Invalid EDI file handling
  - Concurrent job submissions
  - Job not found scenarios

### Running Integration Tests

Integration tests require the API server, MongoDB, and Redis to be running:

```bash
# Terminal 1: Start MongoDB and Redis
make dev-services

# Terminal 2: Start API server
make run

# Terminal 3: Run integration tests
make test-integration
```

Integration tests automatically skip if services are not available.

### Test Fixtures

- **valid.edi**: Sample valid EDI file with 5 claims (13,000 total amount)
- **invalid.edi**: Sample invalid EDI file for error handling tests

## API Documentation

The EDI Processing API is fully documented using OpenAPI 3.0 (Swagger) specification.

### Interactive Swagger UI

Once the API server is running, access the interactive Swagger UI documentation:

```
http://localhost:8080/swagger/index.html
```

The Swagger UI provides:
- Interactive API exploration
- Request/response examples
- "Try it out" functionality
- Complete schema definitions

### OpenAPI Specification

The static OpenAPI spec is available at:
- **File**: `docs/swagger.yaml`
- **Format**: OpenAPI 3.0 YAML

You can import this file into tools like:
- [Swagger Editor](https://editor.swagger.io/)
- [Postman](https://www.postman.com/)
- [Insomnia](https://insomnia.rest/)

### Generating Swagger Docs

To regenerate Swagger documentation from code annotations:

```bash
# Install swag CLI tool (one-time setup)
go install github.com/swaggo/swag/cmd/swag@latest
export PATH=$PATH:$(go env GOPATH)/bin

# Generate/update Swagger docs
make swagger
```

**Note**: The project includes a manually crafted `swagger.yaml` that works without the swag tool.

### API Endpoints Summary

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| `GET` | `/health` | Health check endpoint | No |
| `POST` | `/jobs` | Upload EDI file for processing | No |
| `GET` | `/jobs/:job_id` | Get job status by ID | No |
| `GET` | `/jobs/:job_id/result` | Get processing result | No |

### Example API Usage

#### 1. Upload EDI File

```bash
curl -X POST http://localhost:8080/jobs \
  -F "file=@sample.edi"
```

Response:
```json
{
  "job_id": "65a5b1234567890abcdef123",
  "message": "Job created successfully and queued for processing"
}
```

#### 2. Check Job Status

```bash
curl http://localhost:8080/jobs/65a5b1234567890abcdef123
```

Response:
```json
{
  "job_id": "65a5b1234567890abcdef123",
  "status": "completed",
  "retry_count": 0,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:10Z"
}
```

#### 3. Get Result

```bash
curl http://localhost:8080/jobs/65a5b1234567890abcdef123/result
```

Response:
```json
{
  "status": "completed",
  "claims": [
    {
      "claim_id": "CLM001",
      "member_id": "MEM123",
      "amount": 2500
    }
  ],
  "summary": {
    "total_claims": 3,
    "total_amount": 7000
  }
}
```

For complete request/response schemas, error codes, and examples, see the Swagger documentation.

## Troubleshooting

### Worker not processing jobs

1. Check Redis connection:
```bash
redis-cli PING
```

2. Check MongoDB connection:
```bash
mongosh --eval "db.adminCommand('ping')"
```

3. Verify queue has jobs:
```bash
redis-cli LLEN edi:jobs:queue
```

### Jobs stuck in processing

- Check worker logs for errors
- Verify job exists in MongoDB
- Check if worker crashed (no graceful shutdown)

### High error rate

- Validate EDI file format
- Check MongoDB write permissions
- Review error logs for parsing issues

## Performance Tuning

- **Poll Interval**: Reduce for lower latency, increase to reduce Redis load
- **Concurrent Workers**: Run multiple worker instances for higher throughput
- **Batch Processing**: Group small files for efficient processing

## License

MIT License

## Support

For issues and questions, please create an issue in the repository.
