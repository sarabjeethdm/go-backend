# API Documentation

## Base URL
```
http://localhost:8080
```

## Authentication
Currently, the API does not require authentication. In production, implement JWT or OAuth2.

---

## Endpoints

### 1. Health Check

Check if the API service is running and healthy.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "service": "edi-processing-api"
}
```

**Status Codes:**
- `200 OK` - Service is healthy

**Example:**
```bash
curl http://localhost:8080/health
```

---

### 2. Create Job (Upload EDI File)

Upload an EDI file for processing. The file is validated, stored, and queued for processing.

**Endpoint:** `POST /jobs`

**Request:**
- **Content-Type:** `multipart/form-data`
- **Body Parameters:**
  - `file` (required): The EDI file to upload

**Validation Rules:**
- File must not be empty
- File size must not exceed 10MB
- File content must not be empty

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Job created successfully and queued for processing"
}
```

**Status Codes:**
- `201 Created` - Job created successfully
- `400 Bad Request` - Invalid request (missing file, empty file, or file too large)
- `500 Internal Server Error` - Server error (database or queue failure)

**Error Response:**
```json
{
  "error": "invalid_request",
  "message": "No file provided or invalid file upload"
}
```

**Possible Error Codes:**
- `invalid_request` - No file provided
- `file_too_large` - File exceeds 10MB
- `invalid_file` - File content is empty
- `database_error` - Failed to save job
- `queue_error` - Failed to enqueue job

**Examples:**

```bash
# Using curl
curl -X POST http://localhost:8080/jobs \
  -F "file=@sample_837.edi"

# Using curl with custom file
curl -X POST http://localhost:8080/jobs \
  -F "file=@/path/to/edi_file.txt"
```

```javascript
// Using JavaScript Fetch API
const formData = new FormData();
formData.append('file', fileInput.files[0]);

fetch('http://localhost:8080/jobs', {
  method: 'POST',
  body: formData
})
  .then(response => response.json())
  .then(data => console.log(data));
```

```python
# Using Python requests
import requests

files = {'file': open('sample_837.edi', 'rb')}
response = requests.post('http://localhost:8080/jobs', files=files)
print(response.json())
```

---

### 3. Get Job Status

Retrieve the current status and metadata of a job.

**Endpoint:** `GET /jobs/{job_id}`

**Path Parameters:**
- `job_id` (required): UUID of the job

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "file_name": "sample_837.edi",
  "file_size": 2048,
  "error_msg": "",
  "created_at": "2024-01-15T10:30:00.000Z",
  "updated_at": "2024-01-15T10:30:15.000Z",
  "completed_at": "2024-01-15T10:30:15.000Z"
}
```

**Status Values:**
- `pending` - Job is waiting in the queue
- `processing` - Job is currently being processed
- `completed` - Job completed successfully
- `failed` - Job processing failed

**Status Codes:**
- `200 OK` - Job found and retrieved
- `400 Bad Request` - Invalid job ID format
- `404 Not Found` - Job not found
- `500 Internal Server Error` - Database error

**Error Response:**
```json
{
  "error": "job_not_found",
  "message": "Job with ID 550e8400-e29b-41d4-a716-446655440000 not found"
}
```

**Possible Error Codes:**
- `invalid_job_id` - Invalid UUID format
- `job_not_found` - Job doesn't exist
- `database_error` - Database retrieval failed

**Examples:**

```bash
# Using curl
curl http://localhost:8080/jobs/550e8400-e29b-41d4-a716-446655440000
```

```javascript
// Using JavaScript Fetch API
fetch('http://localhost:8080/jobs/550e8400-e29b-41d4-a716-446655440000')
  .then(response => response.json())
  .then(data => console.log(data));
```

```python
# Using Python requests
import requests

job_id = "550e8400-e29b-41d4-a716-446655440000"
response = requests.get(f'http://localhost:8080/jobs/{job_id}')
print(response.json())
```

---

### 4. Get Job Result

Retrieve the parsed result of a completed job.

**Endpoint:** `GET /jobs/{job_id}/result`

**Path Parameters:**
- `job_id` (required): UUID of the job

**Response (Job Completed):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "claims": [
    {
      "claim_id": "CLM001",
      "patient_control_no": "PC123",
      "claim_amount": "1500.00",
      "patient_last_name": "Smith",
      "patient_first_name": "John",
      "patient_date_of_birth": "19800515",
      "service_lines": [
        {
          "service_date": "20240101",
          "procedure_code": "99213",
          "charge_amount": "150.00",
          "units": "1"
        },
        {
          "service_date": "20240101",
          "procedure_code": "99214",
          "charge_amount": "250.00",
          "units": "1"
        }
      ]
    },
    {
      "claim_id": "CLM002",
      "patient_control_no": "PC124",
      "claim_amount": "2000.00",
      "patient_last_name": "Johnson",
      "patient_first_name": "Jane",
      "patient_date_of_birth": "19750820",
      "service_lines": [
        {
          "service_date": "20240102",
          "procedure_code": "99215",
          "charge_amount": "300.00",
          "units": "1"
        }
      ]
    }
  ],
  "summary": {
    "total_claims": 2,
    "total_claim_amount": 3500.00,
    "total_service_lines": 3,
    "average_claim_amount": 1750.00
  }
}
```

**Response (Job Not Completed):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "message": "Job is processing. Result not available yet."
}
```

**Status Codes:**
- `200 OK` - Result retrieved or job status returned
- `400 Bad Request` - Invalid job ID format
- `404 Not Found` - Job or result not found
- `500 Internal Server Error` - Database error

**Error Response:**
```json
{
  "error": "result_not_found",
  "message": "Result for job 550e8400-e29b-41d4-a716-446655440000 not found"
}
```

**Possible Error Codes:**
- `invalid_job_id` - Invalid UUID format
- `job_not_found` - Job doesn't exist
- `result_not_found` - Result not found (even for completed job)
- `database_error` - Database retrieval failed

**Examples:**

```bash
# Using curl
curl http://localhost:8080/jobs/550e8400-e29b-41d4-a716-446655440000/result
```

```javascript
// Using JavaScript Fetch API
fetch('http://localhost:8080/jobs/550e8400-e29b-41d4-a716-446655440000/result')
  .then(response => response.json())
  .then(data => console.log(data));
```

```python
# Using Python requests
import requests

job_id = "550e8400-e29b-41d4-a716-446655440000"
response = requests.get(f'http://localhost:8080/jobs/{job_id}/result')
print(response.json())
```

---

## Response Format

### Success Response
All successful responses follow the structure defined in the endpoint documentation above.

### Error Response
All error responses follow this consistent structure:

```json
{
  "error": "error_code",
  "message": "Human-readable error message"
}
```

---

## Complete Workflow Example

```bash
# Step 1: Upload an EDI file
response=$(curl -s -X POST http://localhost:8080/jobs \
  -F "file=@sample_837.edi")
job_id=$(echo $response | jq -r '.job_id')
echo "Created job: $job_id"

# Step 2: Check job status (poll until completed)
while true; do
  status=$(curl -s http://localhost:8080/jobs/$job_id | jq -r '.status')
  echo "Job status: $status"
  
  if [ "$status" = "completed" ] || [ "$status" = "failed" ]; then
    break
  fi
  
  sleep 2
done

# Step 3: Get the result
curl -s http://localhost:8080/jobs/$job_id/result | jq .
```

---

## Rate Limiting

Currently, no rate limiting is implemented. For production use:
- Implement rate limiting middleware
- Recommend: 100 requests per minute per IP
- Use Redis for distributed rate limiting

---

## CORS Policy

The API currently allows all origins (`*`). For production:
- Restrict to specific domains
- Configure allowed methods and headers
- Implement preflight request handling

---

## Data Models

### Job Model
```json
{
  "id": "string (UUID)",
  "status": "string (pending|processing|completed|failed)",
  "file_name": "string",
  "file_size": "integer",
  "error_msg": "string (optional)",
  "created_at": "string (ISO 8601)",
  "updated_at": "string (ISO 8601)",
  "completed_at": "string (ISO 8601, optional)"
}
```

### Result Model
```json
{
  "job_id": "string (UUID)",
  "claims": [
    {
      "claim_id": "string",
      "patient_control_no": "string",
      "claim_amount": "string",
      "patient_last_name": "string",
      "patient_first_name": "string",
      "patient_date_of_birth": "string (YYYYMMDD)",
      "service_lines": [
        {
          "service_date": "string (YYYYMMDD)",
          "procedure_code": "string",
          "charge_amount": "string",
          "units": "string"
        }
      ]
    }
  ],
  "summary": {
    "total_claims": "integer",
    "total_claim_amount": "float",
    "total_service_lines": "integer",
    "average_claim_amount": "float"
  }
}
```

---

## Testing

### Postman Collection
Import the following collection for testing:

```json
{
  "info": {
    "name": "EDI Processing API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Health Check",
      "request": {
        "method": "GET",
        "url": "http://localhost:8080/health"
      }
    },
    {
      "name": "Create Job",
      "request": {
        "method": "POST",
        "url": "http://localhost:8080/jobs",
        "body": {
          "mode": "formdata",
          "formdata": [
            {
              "key": "file",
              "type": "file",
              "src": "/path/to/file.edi"
            }
          ]
        }
      }
    },
    {
      "name": "Get Job Status",
      "request": {
        "method": "GET",
        "url": "http://localhost:8080/jobs/{{job_id}}"
      }
    },
    {
      "name": "Get Job Result",
      "request": {
        "method": "GET",
        "url": "http://localhost:8080/jobs/{{job_id}}/result"
      }
    }
  ]
}
```

---

## Support

For issues or questions, please contact the development team or open an issue in the repository.
