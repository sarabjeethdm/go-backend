package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sarabjeet/golang-backend-task/internal/config"
	"github.com/sarabjeet/golang-backend-task/internal/logger"
	"github.com/sarabjeet/golang-backend-task/internal/metrics"
	"github.com/sarabjeet/golang-backend-task/internal/queue"
	"github.com/sarabjeet/golang-backend-task/internal/storage"
	"github.com/sarabjeet/golang-backend-task/internal/worker"
	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	log := logger.New()
	log.Info("Starting EDI Worker Service")

	// Initialize metrics
	metrics.Init()
	log.Info("Metrics initialized")

	// Load configuration
	cfg := config.Load()
	log.WithFields(logrus.Fields{
		"mongodb_uri":     cfg.MongoDB.URI,
		"redis_host":      cfg.Redis.Host,
		"redis_port":      cfg.Redis.Port,
		"max_retries":     cfg.Worker.MaxRetries,
		"poll_interval":   cfg.Worker.PollInterval,
		"initial_backoff": cfg.Worker.InitialBackoff,
	}).Info("Configuration loaded")

	// Initialize storage
	store, err := storage.New(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize storage")
	}
	log.Info("Storage initialized successfully")

	// Initialize queue
	q, err := queue.New(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize queue")
	}
	log.Info("Queue initialized successfully")

	// Initialize processor
	processor := worker.NewProcessor(store, log, cfg.Worker.MaxRetries)

	// Start metrics server for worker
	go func() {
		metricsPort := os.Getenv("METRICS_PORT")
		if metricsPort == "" {
			metricsPort = "9091"
		}
		http.Handle("/metrics", promhttp.Handler())
		log.WithField("port", metricsPort).Info("Starting metrics server")
		if err := http.ListenAndServe(":"+metricsPort, nil); err != nil {
			log.WithError(err).Error("Failed to start metrics server")
		}
	}()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// WaitGroup to track active jobs
	var wg sync.WaitGroup

	// Worker goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		workerLoop(ctx, log, q, processor, cfg, &wg)
	}()

	// Queue size monitoring goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		monitorQueueSize(ctx, log, q)
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.WithField("signal", sig.String()).Info("Shutdown signal received")

	// Cancel context to stop worker loop
	cancel()

	// Wait for active jobs to complete with timeout
	shutdownComplete := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownComplete)
	}()

	shutdownTimeout := time.Duration(cfg.Worker.ShutdownTimeout) * time.Second
	select {
	case <-shutdownComplete:
		log.Info("All jobs completed successfully")
	case <-time.After(shutdownTimeout):
		log.Warn("Shutdown timeout reached, forcing exit")
	}

	// Close connections
	if err := q.Close(); err != nil {
		log.WithError(err).Error("Error closing queue connection")
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	if err := store.Close(closeCtx); err != nil {
		log.WithError(err).Error("Error closing storage connection")
	}

	log.Info("Worker service shutdown complete")
}

// monitorQueueSize periodically updates the queue size gauge
func monitorQueueSize(ctx context.Context, log *logger.Logger, q *queue.Queue) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			size, err := q.Size(ctx)
			if err != nil {
				log.WithError(err).Error("Failed to get queue size")
				continue
			}
			metrics.SetRedisQueueSize(float64(size))
		}
	}
}

// workerLoop is the main worker loop that polls for jobs
func workerLoop(ctx context.Context, log *logger.Logger, q *queue.Queue, processor *worker.Processor, cfg *config.Config, wg *sync.WaitGroup) {
	log.Info("Worker loop started")
	pollInterval := time.Duration(cfg.Worker.PollInterval) * time.Second

	// Map to track jobs being retried with backoff
	retryJobs := make(map[string]time.Time)
	var retryMutex sync.Mutex

	for {
		select {
		case <-ctx.Done():
			log.Info("Worker loop stopping")
			return
		default:
			// Check for retryable jobs
			retryMutex.Lock()
			for jobID, retryTime := range retryJobs {
				if time.Now().After(retryTime) {
					// Time to retry this job
					delete(retryJobs, jobID)

					wg.Add(1)
					go func(jID string) {
						defer wg.Done()
						processJobWithLogging(ctx, log, processor, jID, "")
					}(jobID)
				}
			}
			retryMutex.Unlock()

			// Try to dequeue a job message
			jobData, err := q.Dequeue(ctx)
			if err != nil {
				log.WithError(err).Error("Failed to dequeue job")
				time.Sleep(pollInterval)
				continue
			}

			// If no job available, wait before polling again
			if jobData == "" {
				time.Sleep(pollInterval)
				continue
			}

			// Try to parse as JobMessage (new format)
			var jobMsg queue.JobMessage
			if err := json.Unmarshal([]byte(jobData), &jobMsg); err == nil && jobMsg.JobID != "" {
				// New format: full job message with file content
				log.WithField("job_id", jobMsg.JobID).Info("Job message dequeued from queue")

				// Get job to check retry count for backoff calculation
				job, err := processor.GetJob(ctx, jobMsg.JobID)
				if err != nil {
					log.WithFields(logrus.Fields{
						"job_id": jobMsg.JobID,
						"error":  err.Error(),
					}).Error("Failed to get job for backoff calculation")
					// Process immediately if we can't get job info
					wg.Add(1)
					go func(msg queue.JobMessage) {
						defer wg.Done()
						processJobWithLogging(ctx, log, processor, msg.JobID, msg.FileContent)
					}(jobMsg)
					continue
				}

				// If job has retries, apply exponential backoff
				if job.RetryCount > 0 {
					backoff := processor.CalculateBackoff(job.RetryCount-1, cfg.Worker.InitialBackoff)
					retryTime := time.Now().Add(backoff)

					log.WithFields(logrus.Fields{
						"job_id":      jobMsg.JobID,
						"retry_count": job.RetryCount,
						"backoff":     backoff.String(),
						"retry_at":    retryTime.Format(time.RFC3339),
					}).Info("Scheduling job for retry with backoff")

					retryMutex.Lock()
					retryJobs[jobMsg.JobID] = retryTime
					retryMutex.Unlock()
					continue
				}

				// Process job asynchronously with file content
				wg.Add(1)
				go func(msg queue.JobMessage) {
					defer wg.Done()
					processJobWithLogging(ctx, log, processor, msg.JobID, msg.FileContent)
				}(jobMsg)
			} else {
				// Old format: just job ID (for backward compatibility)
				log.WithField("job_id", jobData).Info("Job ID dequeued from queue (legacy format)")

				// Get job to check retry count for backoff calculation
				job, err := processor.GetJob(ctx, jobData)
				if err != nil {
					log.WithFields(logrus.Fields{
						"job_id": jobData,
						"error":  err.Error(),
					}).Error("Failed to get job for backoff calculation")
					// Process immediately if we can't get job info
					wg.Add(1)
					go func(jID string) {
						defer wg.Done()
						processJobWithLogging(ctx, log, processor, jID, "")
					}(jobData)
					continue
				}

				// If job has retries, apply exponential backoff
				if job.RetryCount > 0 {
					backoff := processor.CalculateBackoff(job.RetryCount-1, cfg.Worker.InitialBackoff)
					retryTime := time.Now().Add(backoff)

					log.WithFields(logrus.Fields{
						"job_id":      jobData,
						"retry_count": job.RetryCount,
						"backoff":     backoff.String(),
						"retry_at":    retryTime.Format(time.RFC3339),
					}).Info("Scheduling job for retry with backoff")

					retryMutex.Lock()
					retryJobs[jobData] = retryTime
					retryMutex.Unlock()
					continue
				}

				// Process job asynchronously (no file content, will fail)
				wg.Add(1)
				go func(jID string) {
					defer wg.Done()
					processJobWithLogging(ctx, log, processor, jID, "")
				}(jobData)
			}
		}
	}
}

// processJobWithLogging processes a job and logs the result
func processJobWithLogging(ctx context.Context, log *logger.Logger, processor *worker.Processor, jobID string, fileContent string) {
	startTime := time.Now()

	log.WithField("job_id", jobID).Info("Processing job")

	// Record job processing started
	metrics.RecordJobProcessing()
	metrics.IncrementActiveJobs()
	defer metrics.DecrementActiveJobs()

	err := processor.ProcessJob(ctx, jobID, fileContent)

	duration := time.Since(startTime)

	// Record processing duration
	metrics.RecordJobProcessingDuration(duration)

	if err != nil {
		log.WithFields(logrus.Fields{
			"job_id":   jobID,
			"duration": duration.String(),
			"error":    err.Error(),
		}).Error("Job processing failed")
		metrics.RecordJobFailed()
	} else {
		log.WithFields(logrus.Fields{
			"job_id":   jobID,
			"duration": duration.String(),
		}).Info("Job processing completed")
		metrics.RecordJobCompleted()
	}
}
