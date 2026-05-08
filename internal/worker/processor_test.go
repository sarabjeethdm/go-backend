package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sarabjeet/golang-backend-task/internal/logger"
	"github.com/sarabjeet/golang-backend-task/internal/models"
)

// MockStorage is a mock implementation of the storage interface
type MockStorage struct {
	jobs                    map[string]*models.Job
	GetJobFunc              func(ctx context.Context, jobID string) (*models.Job, error)
	UpdateJobStatusFunc     func(ctx context.Context, jobID string, status models.JobStatus) error
	UpdateJobWithResultFunc func(ctx context.Context, jobID string, status models.JobStatus, result *models.Result, errorMsg string) error
	IncrementRetryCountFunc func(ctx context.Context, jobID string) error
}

func NewMockStorage() *MockStorage {
	mock := &MockStorage{
		jobs: make(map[string]*models.Job),
	}

	// Set default implementations
	mock.GetJobFunc = func(ctx context.Context, jobID string) (*models.Job, error) {
		if job, ok := mock.jobs[jobID]; ok {
			return job, nil
		}
		return nil, fmt.Errorf("job not found")
	}

	mock.UpdateJobStatusFunc = func(ctx context.Context, jobID string, status models.JobStatus) error {
		if job, ok := mock.jobs[jobID]; ok {
			job.Status = status
			job.UpdatedAt = time.Now()
			return nil
		}
		return fmt.Errorf("job not found")
	}

	mock.UpdateJobWithResultFunc = func(ctx context.Context, jobID string, status models.JobStatus, result *models.Result, errorMsg string) error {
		if job, ok := mock.jobs[jobID]; ok {
			job.Status = status
			job.Result = result
			job.Error = errorMsg
			job.UpdatedAt = time.Now()
			return nil
		}
		return fmt.Errorf("job not found")
	}

	mock.IncrementRetryCountFunc = func(ctx context.Context, jobID string) error {
		if job, ok := mock.jobs[jobID]; ok {
			job.RetryCount++
			job.UpdatedAt = time.Now()
			return nil
		}
		return fmt.Errorf("job not found")
	}

	return mock
}

func (m *MockStorage) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	return m.GetJobFunc(ctx, jobID)
}

func (m *MockStorage) UpdateJobStatus(ctx context.Context, jobID string, status models.JobStatus) error {
	return m.UpdateJobStatusFunc(ctx, jobID, status)
}

func (m *MockStorage) UpdateJobWithResult(ctx context.Context, jobID string, status models.JobStatus, result *models.Result, errorMsg string) error {
	return m.UpdateJobWithResultFunc(ctx, jobID, status, result, errorMsg)
}

func (m *MockStorage) IncrementRetryCount(ctx context.Context, jobID string) error {
	return m.IncrementRetryCountFunc(ctx, jobID)
}

func TestProcessJob_Success(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	processor := NewProcessor(storage, log, 3)

	// Create a test job with valid EDI content
	jobID := "test-job-123"
	validEDI := "CLAIM*CLM001*MEM123*2500\nCLAIM*CLM002*MEM456*3000\nCLAIM*CLM003*MEM789*1500"

	storage.jobs[jobID] = &models.Job{
		FileContent: validEDI,
		Status:      models.StatusPending,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Process the job
	ctx := context.Background()
	err := processor.ProcessJob(ctx, jobID)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify job status was updated to completed
	job := storage.jobs[jobID]
	if job.Status != models.StatusCompleted {
		t.Errorf("Expected status %s, got %s", models.StatusCompleted, job.Status)
	}

	// Verify result is present
	if job.Result == nil {
		t.Fatal("Expected result to be set")
	}

	// Verify result contains correct data
	if job.Result.Summary.TotalClaims != 3 {
		t.Errorf("Expected 3 claims, got %d", job.Result.Summary.TotalClaims)
	}

	expectedAmount := 7000.0
	if job.Result.Summary.TotalAmount != expectedAmount {
		t.Errorf("Expected total amount %f, got %f", expectedAmount, job.Result.Summary.TotalAmount)
	}
}

func TestProcessJob_ParseFailure(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	processor := NewProcessor(storage, log, 3)

	// Create a test job with invalid EDI content
	jobID := "test-job-456"
	invalidEDI := "INVALID*FORMAT*HERE"

	storage.jobs[jobID] = &models.Job{
		FileContent: invalidEDI,
		Status:      models.StatusPending,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Process the job
	ctx := context.Background()
	err := processor.ProcessJob(ctx, jobID)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error for invalid EDI content")
	}

	// Verify retry count was incremented
	job := storage.jobs[jobID]
	if job.RetryCount != 1 {
		t.Errorf("Expected retry count 1, got %d", job.RetryCount)
	}

	// Verify job status was set back to pending for retry
	if job.Status != models.StatusPending {
		t.Errorf("Expected status %s for retry, got %s", models.StatusPending, job.Status)
	}
}

func TestProcessJob_MaxRetriesExceeded(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	maxRetries := 3
	processor := NewProcessor(storage, log, maxRetries)

	// Create a test job with invalid EDI content and max retries already reached
	jobID := "test-job-789"
	invalidEDI := "INVALID*FORMAT*HERE"

	storage.jobs[jobID] = &models.Job{
		FileContent: invalidEDI,
		Status:      models.StatusPending,
		RetryCount:  maxRetries, // Already at max retries
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Process the job
	ctx := context.Background()
	err := processor.ProcessJob(ctx, jobID)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error for max retries exceeded")
	}

	// Verify job status was set to failed
	job := storage.jobs[jobID]
	if job.Status != models.StatusFailed {
		t.Errorf("Expected status %s, got %s", models.StatusFailed, job.Status)
	}

	// Verify error message is set
	if job.Error == "" {
		t.Error("Expected error message to be set")
	}
}

func TestProcessJob_AlreadyProcessing(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	processor := NewProcessor(storage, log, 3)

	// Create a test job that's already processing
	jobID := "test-job-processing"

	storage.jobs[jobID] = &models.Job{
		FileContent: "CLAIM*CLM001*MEM123*2500",
		Status:      models.StatusProcessing,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Process the job
	ctx := context.Background()
	err := processor.ProcessJob(ctx, jobID)

	// Verify no error (job is skipped)
	if err != nil {
		t.Errorf("Expected no error for already processing job, got %v", err)
	}

	// Verify status remains processing
	job := storage.jobs[jobID]
	if job.Status != models.StatusProcessing {
		t.Errorf("Expected status %s, got %s", models.StatusProcessing, job.Status)
	}
}

func TestProcessJob_JobNotFound(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	processor := NewProcessor(storage, log, 3)

	// Try to process a non-existent job
	jobID := "non-existent-job"
	ctx := context.Background()
	err := processor.ProcessJob(ctx, jobID)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error for non-existent job")
	}

	// Verify error message contains "not found"
	if err != nil && err.Error() != "failed to get job: job not found" {
		t.Errorf("Expected 'job not found' error, got %v", err)
	}
}

func TestProcessJob_StorageUpdateError(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	processor := NewProcessor(storage, log, 3)

	// Create a test job
	jobID := "test-job-update-error"
	validEDI := "CLAIM*CLM001*MEM123*2500"

	storage.jobs[jobID] = &models.Job{
		FileContent: validEDI,
		Status:      models.StatusPending,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Mock storage error on status update
	storage.UpdateJobStatusFunc = func(ctx context.Context, jobID string, status models.JobStatus) error {
		return fmt.Errorf("storage update error")
	}

	// Process the job
	ctx := context.Background()
	err := processor.ProcessJob(ctx, jobID)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error for storage update failure")
	}
}

func TestProcessJob_RetryLogic(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	maxRetries := 3
	processor := NewProcessor(storage, log, maxRetries)

	// Test retry logic for different retry counts
	testCases := []struct {
		name           string
		retryCount     int
		shouldRetry    bool
		expectedStatus models.JobStatus
	}{
		{
			name:           "First retry",
			retryCount:     0,
			shouldRetry:    true,
			expectedStatus: models.StatusPending,
		},
		{
			name:           "Second retry",
			retryCount:     1,
			shouldRetry:    true,
			expectedStatus: models.StatusPending,
		},
		{
			name:           "Third retry",
			retryCount:     2,
			shouldRetry:    true,
			expectedStatus: models.StatusPending,
		},
		{
			name:           "Max retries exceeded",
			retryCount:     3,
			shouldRetry:    false,
			expectedStatus: models.StatusFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jobID := fmt.Sprintf("test-job-retry-%d", tc.retryCount)
			invalidEDI := "INVALID*FORMAT"

			storage.jobs[jobID] = &models.Job{
				FileContent: invalidEDI,
				Status:      models.StatusPending,
				RetryCount:  tc.retryCount,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			// Process the job
			ctx := context.Background()
			processor.ProcessJob(ctx, jobID)

			// Verify expected status
			job := storage.jobs[jobID]
			if job.Status != tc.expectedStatus {
				t.Errorf("Expected status %s, got %s", tc.expectedStatus, job.Status)
			}

			// Verify retry count increment for retryable cases
			if tc.shouldRetry && job.RetryCount != tc.retryCount+1 {
				t.Errorf("Expected retry count %d, got %d", tc.retryCount+1, job.RetryCount)
			}
		})
	}
}

func TestGetJob(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	processor := NewProcessor(storage, log, 3)

	// Create a test job
	jobID := "test-job-get"
	expectedJob := &models.Job{
		FileContent: "CLAIM*CLM001*MEM123*2500",
		Status:      models.StatusPending,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	storage.jobs[jobID] = expectedJob

	// Get the job
	ctx := context.Background()
	job, err := processor.GetJob(ctx, jobID)

	// Verify no error
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify job matches
	if job.Status != expectedJob.Status {
		t.Errorf("Expected status %s, got %s", expectedJob.Status, job.Status)
	}
}

func TestCalculateBackoff(t *testing.T) {
	storage := NewMockStorage()
	log := logger.New()
	processor := NewProcessor(storage, log, 3)

	testCases := []struct {
		retryCount      int
		initialBackoff  int
		expectedSeconds int
	}{
		{retryCount: 0, initialBackoff: 2, expectedSeconds: 2},  // 2^0 * 2 = 2
		{retryCount: 1, initialBackoff: 2, expectedSeconds: 4},  // 2^1 * 2 = 4
		{retryCount: 2, initialBackoff: 2, expectedSeconds: 8},  // 2^2 * 2 = 8
		{retryCount: 3, initialBackoff: 2, expectedSeconds: 16}, // 2^3 * 2 = 16
		{retryCount: 1, initialBackoff: 5, expectedSeconds: 10}, // 2^1 * 5 = 10
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("retry_%d_backoff_%d", tc.retryCount, tc.initialBackoff), func(t *testing.T) {
			duration := processor.CalculateBackoff(tc.retryCount, tc.initialBackoff)
			expectedDuration := time.Duration(tc.expectedSeconds) * time.Second

			if duration != expectedDuration {
				t.Errorf("Expected backoff %v, got %v", expectedDuration, duration)
			}
		})
	}
}
