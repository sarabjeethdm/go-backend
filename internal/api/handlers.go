package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sarabjeet/golang-backend-task/internal/logger"
	"github.com/sarabjeet/golang-backend-task/internal/metrics"
	"github.com/sarabjeet/golang-backend-task/internal/models"
	"github.com/sarabjeet/golang-backend-task/internal/queue"
	"github.com/sarabjeet/golang-backend-task/internal/storage"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	db    *storage.MongoDB
	queue *queue.RedisQueue
}

// NewHandler creates a new handler
func NewHandler(db *storage.MongoDB, q *queue.RedisQueue) *Handler {
	return &Handler{
		db:    db,
		queue: q,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// CreateJobResponse represents the create job response
type CreateJobResponse struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

// HealthCheck handles the health check endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	logger.WithFields(map[string]interface{}{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
	}).Info("Health check request")

	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "edi-processing-api",
	})
}

// CreateJob handles the POST /jobs endpoint for uploading EDI files
func (h *Handler) CreateJob(c *gin.Context) {
	logger.WithFields(map[string]interface{}{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
	}).Info("Create job request")

	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get file from request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "No file provided or invalid file upload",
		})
		return
	}
	defer file.Close()

	// Validate file size (10MB max by default)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "file_too_large",
			Message: "File size exceeds maximum allowed size of 10MB",
		})
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":     err.Error(),
			"file_name": header.Filename,
		}).Error("Failed to read file content")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "file_read_error",
			Message: "Failed to read file content",
		})
		return
	}

	fileContent := string(content)

	// Validate file content (basic validation - check if it's not empty)
	if strings.TrimSpace(fileContent) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_file",
			Message: "File content is empty",
		})
		return
	}

	// Generate job ID
	jobID := uuid.New().String()

	// Create job
	job := models.NewJob(jobID, header.Filename)

	// Save job to MongoDB (without file content)
	if err := h.db.SaveJob(c.Request.Context(), job); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
		}).Error("Failed to save job to database")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to save job",
		})
		return
	}

	// Create job message for queue
	jobMsg := &queue.JobMessage{
		JobID:       jobID,
		FileName:    header.Filename,
		FileContent: fileContent,
		CreatedAt:   time.Now(),
	}

	// Enqueue job
	if err := h.queue.Enqueue(c.Request.Context(), jobMsg); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
		}).Error("Failed to enqueue job")

		// Update job status to failed
		job.UpdateStatus(models.StatusFailed, "Failed to enqueue job")
		h.db.UpdateJob(c.Request.Context(), job)

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "queue_error",
			Message: "Failed to enqueue job for processing",
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"job_id":    jobID,
		"file_name": header.Filename,
		"file_size": header.Size,
	}).Info("Job created and enqueued successfully")

	// Record metrics for job creation
	metrics.RecordJobCreated()

	c.JSON(http.StatusCreated, CreateJobResponse{
		JobID:   jobID,
		Message: "Job created successfully and queued for processing",
	})
}

// GetJobStatus handles the GET /jobs/:job_id endpoint
func (h *Handler) GetJobStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	logger.WithFields(map[string]interface{}{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"job_id": jobID,
	}).Info("Get job status request")

	// Validate job ID format
	if _, err := uuid.Parse(jobID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_job_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Get job from database
	job, err := h.db.GetJob(c.Request.Context(), jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "job_not_found",
				Message: fmt.Sprintf("Job with ID %s not found", jobID),
			})
			return
		}

		logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
		}).Error("Failed to get job from database")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve job",
		})
		return
	}

	c.JSON(http.StatusOK, job.ToResponse())
}

// GetJobResult handles the GET /jobs/:job_id/result endpoint
func (h *Handler) GetJobResult(c *gin.Context) {
	jobID := c.Param("job_id")

	logger.WithFields(map[string]interface{}{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"job_id": jobID,
	}).Info("Get job result request")

	// Validate job ID format
	if _, err := uuid.Parse(jobID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_job_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Get job from database to check status
	job, err := h.db.GetJob(c.Request.Context(), jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "job_not_found",
				Message: fmt.Sprintf("Job with ID %s not found", jobID),
			})
			return
		}

		logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
		}).Error("Failed to get job from database")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve job",
		})
		return
	}

	// Check job status
	if job.Status != models.StatusCompleted {
		c.JSON(http.StatusOK, gin.H{
			"job_id":  jobID,
			"status":  job.Status,
			"message": fmt.Sprintf("Job is %s. Result not available yet.", job.Status),
		})
		return
	}

	// Get result from database
	result, err := h.db.GetResult(c.Request.Context(), jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "result_not_found",
				Message: fmt.Sprintf("Result for job %s not found", jobID),
			})
			return
		}

		logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
		}).Error("Failed to get result from database")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve result",
		})
		return
	}

	c.JSON(http.StatusOK, result.ToResponse())
}
