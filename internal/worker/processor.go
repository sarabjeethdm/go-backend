package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/sarabjeet/golang-backend-task/internal/models"
	"github.com/sarabjeet/golang-backend-task/internal/parser"
	"github.com/sarabjeet/golang-backend-task/internal/storage"
)

type Processor struct {
	storage    *storage.Storage
	maxRetries int
}

func NewProcessor(storage *storage.Storage, maxRetries int) *Processor {
	return &Processor{
		storage:    storage,
		maxRetries: maxRetries,
	}
}

func (p *Processor) ProcessJob(ctx context.Context, jobID string, fileContent string) error {
	// Get job from storage
	job, err := p.storage.GetJob(ctx, jobID)
	if err != nil {
		log.Printf("Failed to get job %s: %v", jobID, err)
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Check if job is in pending status
	if job.Status != models.StatusPending {
		log.Printf("Job %s is not pending (status: %s), skipping", jobID, job.Status)
		return nil
	}

	// Update status to processing
	if err := p.storage.UpdateJobStatus(ctx, jobID, models.StatusProcessing); err != nil {
		log.Printf("Failed to update job %s to processing: %v", jobID, err)
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Validate file content
	if fileContent == "" {
		log.Printf("Job %s has no file content", jobID)
		if err := p.storage.UpdateJobWithResult(ctx, jobID, models.StatusFailed, nil, "File content not available"); err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}
		return fmt.Errorf("file content not available")
	}

	// Parse EDI file
	result, err := parser.ParseEDI(fileContent)
	if err != nil {
		log.Printf("Failed to parse EDI file for job %s: %v", jobID, err)

		// Check if we should retry
		if job.RetryCount < p.maxRetries {
			log.Printf("Job %s will be retried (attempt %d/%d)", jobID, job.RetryCount+1, p.maxRetries)

			// Mark job as pending for retry
			if err := p.storage.UpdateJobStatus(ctx, jobID, models.StatusPending); err != nil {
				log.Printf("Failed to update job status for retry: %v", err)
			}

			// Increment retry count
			if err := p.storage.IncrementRetryCount(ctx, jobID); err != nil {
				log.Printf("Failed to increment retry count: %v", err)
			}

			return fmt.Errorf("parsing failed, job marked for retry: %w", err)
		}

		// Max retries exceeded, mark as failed
		log.Printf("Job %s failed after %d attempts", jobID, p.maxRetries)
		if err := p.storage.UpdateJobWithResult(ctx, jobID, models.StatusFailed, nil, err.Error()); err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}
		return fmt.Errorf("parsing failed after max retries: %w", err)
	}

	// Save result and mark as completed
	log.Printf("Job %s parsed successfully: %d claims, total amount: %.2f",
		jobID, result.Summary.TotalClaims, result.Summary.TotalAmount)

	if err := p.storage.UpdateJobWithResult(ctx, jobID, models.StatusCompleted, result, ""); err != nil {
		log.Printf("Failed to save result for job %s: %v", jobID, err)
		return fmt.Errorf("failed to update job with result: %w", err)
	}

	return nil
}
