package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sarabjeet/golang-backend-task/internal/metrics"
	"github.com/sarabjeet/golang-backend-task/internal/models"
	"github.com/sarabjeet/golang-backend-task/internal/queue"
	"github.com/sarabjeet/golang-backend-task/internal/storage"
)

type Handler struct {
	db    *storage.MongoDB
	queue *queue.RedisQueue
}

func NewHandler(db *storage.MongoDB, q *queue.RedisQueue) *Handler {
	return &Handler{
		db:    db,
		queue: q,
	}
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type CreateJobResponse struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

func (h *Handler) HealthCheck(c *gin.Context) {
	log.Printf("Health check request: method=%s path=%s", c.Request.Method, c.Request.URL.Path)

	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "edi-processing-api",
	})
}

func (h *Handler) CreateJob(c *gin.Context) {
	log.Printf("Create job request: method=%s path=%s", c.Request.Method, c.Request.URL.Path)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("Failed to get file from request: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "No file provided or invalid file upload",
		})
		return
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "file_too_large",
			Message: "File size exceeds maximum allowed size of 10MB",
		})
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Failed to read file content: %v, file_name=%s", err, header.Filename)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "file_read_error",
			Message: "Failed to read file content",
		})
		return
	}

	fileContent := string(content)

	if strings.TrimSpace(fileContent) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_file",
			Message: "File content is empty",
		})
		return
	}

	jobID := uuid.New().String()

	job := models.NewJob(jobID, header.Filename)

	if err := h.db.SaveJob(c.Request.Context(), job); err != nil {
		log.Printf("Failed to save job to database: %v, job_id=%s", err, jobID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to save job",
		})
		return
	}

	jobMsg := &queue.JobMessage{
		JobID:       jobID,
		FileName:    header.Filename,
		FileContent: fileContent,
		CreatedAt:   time.Now(),
	}

	if err := h.queue.Enqueue(c.Request.Context(), jobMsg); err != nil {
		log.Printf("Failed to enqueue job: %v, job_id=%s", err, jobID)

		job.UpdateStatus(models.StatusFailed, "Failed to enqueue job")
		h.db.UpdateJob(c.Request.Context(), job)

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "queue_error",
			Message: "Failed to enqueue job for processing",
		})
		return
	}

	log.Printf("Job created and enqueued successfully: job_id=%s file_name=%s file_size=%d", jobID, header.Filename, header.Size)

	metrics.RecordJobCreated()

	c.JSON(http.StatusCreated, CreateJobResponse{
		JobID:   jobID,
		Message: "Job created successfully and queued for processing",
	})
}

func (h *Handler) GetJobStatus(c *gin.Context) {
	jobID := c.Param("job_id")

	log.Printf("Get job status request: method=%s path=%s job_id=%s", c.Request.Method, c.Request.URL.Path, jobID)

	if _, err := uuid.Parse(jobID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_job_id",
			Message: "Invalid job ID format",
		})
		return
	}

	job, err := h.db.GetJob(c.Request.Context(), jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "job_not_found",
				Message: fmt.Sprintf("Job with ID %s not found", jobID),
			})
			return
		}

		log.Printf("Failed to get job from database: %v, job_id=%s", err, jobID)

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve job",
		})
		return
	}

	c.JSON(http.StatusOK, job.ToResponse())
}

func (h *Handler) GetJobResult(c *gin.Context) {
	jobID := c.Param("job_id")

	log.Printf("Get job result request: method=%s path=%s job_id=%s", c.Request.Method, c.Request.URL.Path, jobID)

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

		log.Printf("Failed to get job from database: %v, job_id=%s", err, jobID)

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve job",
		})
		return
	}

	if job.Status != models.StatusCompleted {
		c.JSON(http.StatusOK, gin.H{
			"job_id":  jobID,
			"status":  job.Status,
			"message": fmt.Sprintf("Job is %s. Result not available yet.", job.Status),
		})
		return
	}

	result, err := h.db.GetResult(c.Request.Context(), jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "result_not_found",
				Message: fmt.Sprintf("Result for job %s not found", jobID),
			})
			return
		}

		log.Printf("Failed to get result from database: %v, job_id=%s", err, jobID)

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve result",
		})
		return
	}

	c.JSON(http.StatusOK, result.ToResponse())
}
