# Quick Setup Guide

This guide will help you get the EDI Processing API up and running quickly.

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (optional, but recommended)
- MongoDB (if not using Docker)
- Redis (if not using Docker)

## Quick Start with Docker Compose (Recommended)

This is the easiest way to get everything running:

```bash
# 1. Navigate to the project directory
cd golang-backend-task

# 2. Start all services (MongoDB, Redis, and API)
docker-compose up -d

# 3. Check if services are running
docker-compose ps

# 4. View logs
docker-compose logs -f api

# 5. Test the health endpoint
curl http://localhost:8080/health
```

The API will be available at `http://localhost:8080`.

To stop the services:
```bash
docker-compose down
```

To stop and remove all data:
```bash
docker-compose down -v
```

## Manual Setup (Without Docker)

### Step 1: Install Dependencies

Make sure you have MongoDB and Redis running locally.

**Start MongoDB:**
```bash
# Using Docker
docker run -d --name mongodb -p 27017:27017 mongo:latest

# Or install locally and start the service
```

**Start Redis:**
```bash
# Using Docker
docker run -d --name redis -p 6379:6379 redis:latest

# Or install locally and start the service
```

### Step 2: Download Go Dependencies

```bash
cd golang-backend-task
go mod download
```

### Step 3: Set Environment Variables (Optional)

Create a `.env` file or export environment variables:

```bash
export PORT=8080
export MONGODB_URI="mongodb://localhost:27017"
export MONGODB_DATABASE="edi_processing"
export REDIS_URI="localhost:6379"
export LOG_LEVEL="info"
```

### Step 4: Run the Application

```bash
# Option 1: Using go run
go run cmd/api/main.go

# Option 2: Build and run
make build
./api-server

# Option 3: Using make
make run
```

The API will start on port 8080 (or the port you specified).

## Verify Installation

### 1. Check Health Endpoint

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "service": "edi-processing-api"
}
```

### 2. Test File Upload

Create a test file:
```bash
echo "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *240115*1030*U*00401*000000001*0*P*:" > test.edi
```

Upload it:
```bash
curl -X POST http://localhost:8080/jobs -F "file=@test.edi"
```

Expected response:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Job created successfully and queued for processing"
}
```

### 3. Check Job Status

```bash
# Replace JOB_ID with the ID from the previous response
curl http://localhost:8080/jobs/JOB_ID
```

## Using Makefile Commands

The project includes a Makefile with helpful commands:

```bash
# Show all available commands
make help

# Download dependencies
make deps

# Build the application
make build

# Run the application
make run

# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean

# Start dev services (MongoDB and Redis)
make dev-services

# Stop dev services
make stop-services

# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Run all checks and build
make all
```

## Configuration

The application uses environment variables for configuration. Here are the key settings:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Server port |
| `MONGODB_URI` | mongodb://localhost:27017 | MongoDB connection string |
| `MONGODB_DATABASE` | edi_processing | Database name |
| `REDIS_URI` | localhost:6379 | Redis server address |
| `LOG_LEVEL` | info | Logging level |

See `.env.example` for all available configuration options.

## Troubleshooting

### Connection Refused Errors

If you get connection errors:

1. **MongoDB not running:**
   ```bash
   docker ps | grep mongo
   # If not running, start it:
   docker start mongodb
   # Or: make dev-services
   ```

2. **Redis not running:**
   ```bash
   docker ps | grep redis
   # If not running, start it:
   docker start redis
   # Or: make dev-services
   ```

### Port Already in Use

If port 8080 is already in use:

```bash
# Change the port
export PORT=8081
go run cmd/api/main.go
```

### Module Errors

If you get Go module errors:

```bash
go mod tidy
go mod download
```

## Next Steps

1. **Read the API Documentation:** See [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for detailed API usage.

2. **Read the README:** See [README.md](./README.md) for architecture and detailed information.

3. **Implement a Worker Service:** To actually process the jobs from the queue, you'll need to implement a worker service that:
   - Dequeues jobs from Redis
   - Processes EDI files
   - Updates job status
   - Saves results to MongoDB

4. **Add Tests:** Implement unit and integration tests for your handlers and services.

5. **Configure for Production:** 
   - Enable authentication
   - Set up HTTPS/TLS
   - Configure MongoDB authentication
   - Set up monitoring and logging
   - Scale workers with Docker Compose

## Development Workflow

```bash
# 1. Start development services
make dev-services

# 2. Run the application
make run

# 3. Make changes to the code

# 4. Run tests
make test

# 5. Format and lint
make fmt
make lint

# 6. Build
make build

# 7. Stop services when done
make stop-services
```

## Getting Help

- Check the [README.md](./README.md) for detailed documentation
- Check the [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for API reference
- Review the code comments for implementation details

## Success!

If you can successfully hit the health endpoint and upload a file, your API is ready to use! 🎉
