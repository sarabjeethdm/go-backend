package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/sarabjeet/golang-backend-task/internal/logger"
	"github.com/sarabjeet/golang-backend-task/internal/models"
	"github.com/sarabjeet/golang-backend-task/internal/parser"
	"github.com/sarabjeet/golang-backend-task/internal/storage"
	"github.com/sirupsen/logrus"
)

// Processor handles job processing
type Processor struct {
	storage    *storage.Storage
	log        *logger.Logger
	maxRetries int
}

// NewProcessor creates a new Processor instance
func NewProcessor(storage *storage.Storage, log *logger.Logger, maxRetries int) *Processor {
	return &Processor{
		storage:    storage,
		log:        log,
		maxRetries: maxRetries,
	}
}

// ProcessJob processes a single job
func (p *Processor) ProcessJob(ctx context.Context, jobID string, fileContent string) error {
	p.log.WithFields(logrus.Fields{
		"job_id": jobID,
	}).Info("Starting job processing")

	// Get the job from storage
	job, err := p.storage.GetJob(ctx, jobID)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"job_id": jobID,
			"error":  err.Error(),
		}).Error("Failed to get job from storage")
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Check if job is already processing or completed
	if job.Status != models.StatusPending {
		p.log.WithFields(logrus.Fields{
			"job_id": jobID,
			"status": job.Status,
		}).Warn("Job is not in pending status, skipping")
		return nil
	}

	// Update status to processing
	if err := p.storage.UpdateJobStatus(ctx, jobID, models.StatusProcessing); err != nil {
		p.log.WithFields(logrus.Fields{
			"job_id": jobID,
			"error":  err.Error(),
		}).Error("Failed to update job status to processing")
		return fmt.Errorf("failed to update job status: %w", err)
	}

	p.log.WithFields(logrus.Fields{
		"job_id": jobID,
	}).Info("Job status updated to processing")

	// If file content is not provided (legacy case), fail the job
	if fileContent == "" {
		p.log.WithFields(logrus.Fields{
			"job_id": jobID,
		}).Error("File content not provided")
		if updateErr := p.storage.UpdateJobWithResult(ctx, jobID, models.StatusFailed, nil, "File content not available"); updateErr != nil {
			p.log.WithFields(logrus.Fields{
				"job_id": jobID,
				"error":  updateErr.Error(),
			}).Error("Failed to update job status to failed")
			return fmt.Errorf("failed to update job status: %w", updateErr)
		}
		return fmt.Errorf("file content not available")
	}

	// Parse the EDI file content
	result, err := parser.ParseEDI(fileContent)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"job_id": jobID,
			"error":  err.Error(),
		}).Error("Failed to parse EDI file")

		// Check if we should retry
		if job.RetryCount < p.maxRetries {
			p.log.WithFields(logrus.Fields{
				"job_id":      jobID,
				"retry_count": job.RetryCount + 1,
				"max_retries": p.maxRetries,
			}).Info("Marking job for retry")

			// Update job status back to pending for retry
			if updateErr := p.storage.UpdateJobStatus(ctx, jobID, models.StatusPending); updateErr != nil {
				p.log.WithFields(logrus.Fields{
					"job_id": jobID,
					"error":  updateErr.Error(),
				}).Error("Failed to update job status for retry")
			}

			// Increment retry count
			if incrErr := p.storage.IncrementRetryCount(ctx, jobID); incrErr != nil {
				p.log.WithFields(logrus.Fields{
					"job_id": jobID,
					"error":  incrErr.Error(),
				}).Error("Failed to increment retry count")
			}

			return fmt.Errorf("parsing failed, job marked for retry: %w", err)
		}

		// Max retries exceeded, mark as failed
		p.log.WithFields(logrus.Fields{
			"job_id":      jobID,
			"retry_count": job.RetryCount,
			"max_retries": p.maxRetries,
		}).Error("Max retries exceeded, marking job as failed")

		if updateErr := p.storage.UpdateJobWithResult(ctx, jobID, models.StatusFailed, nil, err.Error()); updateErr != nil {
			p.log.WithFields(logrus.Fields{
				"job_id": jobID,
				"error":  updateErr.Error(),
			}).Error("Failed to update job status to failed")
			return fmt.Errorf("failed to update job status: %w", updateErr)
		}

		return fmt.Errorf("parsing failed after max retries: %w", err)
	}

	p.log.WithFields(logrus.Fields{
		"job_id":       jobID,
		"total_claims": result.Summary.TotalClaims,
		"total_amount": result.Summary.TotalAmount,
	}).Info("Successfully parsed EDI file")

	// Update job with result
	if err := p.storage.UpdateJobWithResult(ctx, jobID, models.StatusCompleted, result, ""); err != nil {
		p.log.WithFields(logrus.Fields{
			"job_id": jobID,
			"error":  err.Error(),
		}).Error("Failed to update job with result")
		return fmt.Errorf("failed to update job with result: %w", err)
	}

	p.log.WithFields(logrus.Fields{
		"job_id": jobID,
	}).Info("Job completed successfully")

	return nil
}

// GetJob retrieves a job from storage
func (p *Processor) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	return p.storage.GetJob(ctx, jobID)
}

// CalculateBackoff calculates exponential backoff duration
func (p *Processor) CalculateBackoff(retryCount int, initialBackoff int) time.Duration {
	// Exponential backoff: 2^retryCount * initialBackoff seconds
	backoffSeconds := (1 << uint(retryCount)) * initialBackoff
	return time.Duration(backoffSeconds) * time.Second
}
