package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sarabjeet/golang-backend-task/internal/models"
	"github.com/sarabjeet/golang-backend-task/internal/queue"
)

// MockMongoDB is a mock implementation of MongoDB storage
type MockMongoDB struct {
	jobs          map[string]*models.Job
	results       map[string]*models.Result
	SaveJobFunc   func(ctx context.Context, job *models.Job) error
	GetJobFunc    func(ctx context.Context, jobID string) (*models.Job, error)
	GetResultFunc func(ctx context.Context, jobID string) (*models.Result, error)
	UpdateJobFunc func(ctx context.Context, job *models.Job) error
}

func NewMockMongoDB() *MockMongoDB {
	mock := &MockMongoDB{
		jobs:    make(map[string]*models.Job),
		results: make(map[string]*models.Result),
	}

	// Set default implementations
	mock.SaveJobFunc = func(ctx context.Context, job *models.Job) error {
		jobID := uuid.New().String()
		mock.jobs[jobID] = job
		return nil
	}

	mock.GetJobFunc = func(ctx context.Context, jobID string) (*models.Job, error) {
		if job, ok := mock.jobs[jobID]; ok {
			return job, nil
		}
		return nil, fmt.Errorf("job not found")
	}

	mock.GetResultFunc = func(ctx context.Context, jobID string) (*models.Result, error) {
		if result, ok := mock.results[jobID]; ok {
			return result, nil
		}
		return nil, fmt.Errorf("result not found")
	}

	mock.UpdateJobFunc = func(ctx context.Context, job *models.Job) error {
		jobID := job.ID.Hex()
		if _, ok := mock.jobs[jobID]; ok {
			mock.jobs[jobID] = job
			return nil
		}
		return fmt.Errorf("job not found")
	}

	return mock
}

func (m *MockMongoDB) SaveJob(ctx context.Context, job *models.Job) error {
	return m.SaveJobFunc(ctx, job)
}

func (m *MockMongoDB) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	return m.GetJobFunc(ctx, jobID)
}

func (m *MockMongoDB) GetResult(ctx context.Context, jobID string) (*models.Result, error) {
	return m.GetResultFunc(ctx, jobID)
}

func (m *MockMongoDB) UpdateJob(ctx context.Context, job *models.Job) error {
	return m.UpdateJobFunc(ctx, job)
}

// MockRedisQueue is a mock implementation of Redis queue
type MockRedisQueue struct {
	messages    []*queue.JobMessage
	EnqueueFunc func(ctx context.Context, jobMsg *queue.JobMessage) error
}

func NewMockRedisQueue() *MockRedisQueue {
	mock := &MockRedisQueue{
		messages: make([]*queue.JobMessage, 0),
	}

	mock.EnqueueFunc = func(ctx context.Context, jobMsg *queue.JobMessage) error {
		mock.messages = append(mock.messages, jobMsg)
		return nil
	}

	return mock
}

func (m *MockRedisQueue) Enqueue(ctx context.Context, jobMsg *queue.JobMessage) error {
	return m.EnqueueFunc(ctx, jobMsg)
}

func setupTestRouter(db *MockMongoDB, q *MockRedisQueue) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	h := NewHandler(db, q)
	router.GET("/health", h.HealthCheck)
	router.POST("/jobs", h.CreateJob)
	router.GET("/jobs/:job_id", h.GetJobStatus)
	router.GET("/jobs/:job_id/result", h.GetJobResult)

	return router
}

func TestHealthHandler(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()
	router := setupTestRouter(db, q)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}

	if response.Service != "edi-processing-api" {
		t.Errorf("Expected service 'edi-processing-api', got '%s'", response.Service)
	}
}

func TestCreateJobHandler_Success(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()
	router := setupTestRouter(db, q)

	// Create a multipart form with a file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fileContent := "CLAIM*CLM001*MEM123*2500\nCLAIM*CLM002*MEM456*3000"
	part, err := writer.CreateFormFile("file", "test.edi")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte(fileContent))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response CreateJobResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.JobID == "" {
		t.Error("Expected non-empty job ID")
	}

	// Verify job was enqueued
	if len(q.messages) != 1 {
		t.Errorf("Expected 1 message in queue, got %d", len(q.messages))
	}
}

func TestCreateJobHandler_NoFile(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()
	router := setupTestRouter(db, q)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/jobs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "invalid_request" {
		t.Errorf("Expected error 'invalid_request', got '%s'", response.Error)
	}
}

func TestCreateJobHandler_EmptyFile(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()
	router := setupTestRouter(db, q)

	// Create a multipart form with an empty file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "empty.edi")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte(""))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "invalid_file" {
		t.Errorf("Expected error 'invalid_file', got '%s'", response.Error)
	}
}

func TestCreateJobHandler_DatabaseError(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()

	// Mock database error
	db.SaveJobFunc = func(ctx context.Context, job *models.Job) error {
		return fmt.Errorf("database connection error")
	}

	router := setupTestRouter(db, q)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.edi")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte("CLAIM*CLM001*MEM123*2500"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "database_error" {
		t.Errorf("Expected error 'database_error', got '%s'", response.Error)
	}
}

func TestCreateJobHandler_QueueError(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()

	// Mock queue error
	q.EnqueueFunc = func(ctx context.Context, jobMsg *queue.JobMessage) error {
		return fmt.Errorf("redis connection error")
	}

	router := setupTestRouter(db, q)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.edi")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte("CLAIM*CLM001*MEM123*2500"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/jobs", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "queue_error" {
		t.Errorf("Expected error 'queue_error', got '%s'", response.Error)
	}
}

func TestGetJobHandler_Success(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()

	// Create a test job
	jobID := uuid.New().String()
	testJob := &models.Job{
		Status:     models.StatusPending,
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	db.jobs[jobID] = testJob

	router := setupTestRouter(db, q)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/jobs/"+jobID, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetJobHandler_InvalidJobID(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()
	router := setupTestRouter(db, q)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/jobs/invalid-id", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "invalid_job_id" {
		t.Errorf("Expected error 'invalid_job_id', got '%s'", response.Error)
	}
}

func TestGetJobHandler_NotFound(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()
	router := setupTestRouter(db, q)

	jobID := uuid.New().String()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/jobs/"+jobID, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "job_not_found" {
		t.Errorf("Expected error 'job_not_found', got '%s'", response.Error)
	}
}

func TestGetResultHandler_Success(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()

	// Create a completed job with result
	jobID := uuid.New().String()
	testResult := &models.Result{
		Claims: []models.Claim{
			{ClaimID: "CLM001", MemberID: "MEM123", Amount: 2500},
			{ClaimID: "CLM002", MemberID: "MEM456", Amount: 3000},
		},
		Summary: models.Summary{
			TotalClaims: 2,
			TotalAmount: 5500,
		},
	}
	testJob := &models.Job{
		Status:     models.StatusCompleted,
		Result:     testResult,
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	db.jobs[jobID] = testJob
	db.results[jobID] = testResult

	router := setupTestRouter(db, q)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/jobs/"+jobID+"/result", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetResultHandler_JobNotCompleted(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()

	// Create a pending job
	jobID := uuid.New().String()
	testJob := &models.Job{
		Status:     models.StatusPending,
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	db.jobs[jobID] = testJob

	router := setupTestRouter(db, q)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/jobs/"+jobID+"/result", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != string(models.StatusPending) {
		t.Errorf("Expected status 'pending', got '%v'", response["status"])
	}
}

func TestGetResultHandler_InvalidJobID(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()
	router := setupTestRouter(db, q)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/jobs/invalid-id/result", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "invalid_job_id" {
		t.Errorf("Expected error 'invalid_job_id', got '%s'", response.Error)
	}
}

func TestGetResultHandler_JobNotFound(t *testing.T) {
	db := NewMockMongoDB()
	q := NewMockRedisQueue()
	router := setupTestRouter(db, q)

	jobID := uuid.New().String()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/jobs/"+jobID+"/result", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
