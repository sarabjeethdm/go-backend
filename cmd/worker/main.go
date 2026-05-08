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
	log := logger.New()
	log.Info("Starting EDI Worker Service")

	metrics.Init()
	log.Info("Metrics initialized")

	cfg := config.Load()
	log.WithFields(logrus.Fields{
		"mongodb_uri":     cfg.MongoDB.URI,
		"redis_host":      cfg.Redis.Host,
		"redis_port":      cfg.Redis.Port,
		"max_retries":     cfg.Worker.MaxRetries,
		"poll_interval":   cfg.Worker.PollInterval,
		"initial_backoff": cfg.Worker.InitialBackoff,
	}).Info("Configuration loaded")

	store, err := storage.New(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize storage")
	}
	log.Info("Storage initialized successfully")

	q, err := queue.New(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize queue")
	}
	log.Info("Queue initialized successfully")

	processor := worker.NewProcessor(store, log, cfg.Worker.MaxRetries)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		workerLoop(ctx, log, q, processor, cfg, &wg)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		monitorQueueSize(ctx, log, q)
	}()

	sig := <-sigChan
	log.WithField("signal", sig.String()).Info("Shutdown signal received")

	cancel()

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

func workerLoop(ctx context.Context, log *logger.Logger, q *queue.Queue, processor *worker.Processor, cfg *config.Config, wg *sync.WaitGroup) {
	log.Info("Worker loop started")
	pollInterval := time.Duration(cfg.Worker.PollInterval) * time.Second

	retryJobs := make(map[string]time.Time)
	var retryMutex sync.Mutex

	for {
		select {
		case <-ctx.Done():
			log.Info("Worker loop stopping")
			return
		default:
			retryMutex.Lock()
			for jobID, retryTime := range retryJobs {
				if time.Now().After(retryTime) {
					delete(retryJobs, jobID)

					wg.Add(1)
					go func(jID string) {
						defer wg.Done()
						processJobWithLogging(ctx, log, processor, jID, "")
					}(jobID)
				}
			}
			retryMutex.Unlock()

			jobData, err := q.Dequeue(ctx)
			if err != nil {
				log.WithError(err).Error("Failed to dequeue job")
				time.Sleep(pollInterval)
				continue
			}

			if jobData == "" {
				time.Sleep(pollInterval)
				continue
			}

			var jobMsg queue.JobMessage
			if err := json.Unmarshal([]byte(jobData), &jobMsg); err == nil && jobMsg.JobID != "" {
				log.WithField("job_id", jobMsg.JobID).Info("Job message dequeued from queue")

				job, err := processor.GetJob(ctx, jobMsg.JobID)
				if err != nil {
					log.WithFields(logrus.Fields{
						"job_id": jobMsg.JobID,
						"error":  err.Error(),
					}).Error("Failed to get job for backoff calculation")
					wg.Add(1)
					go func(msg queue.JobMessage) {
						defer wg.Done()
						processJobWithLogging(ctx, log, processor, msg.JobID, msg.FileContent)
					}(jobMsg)
					continue
				}

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

				wg.Add(1)
				go func(msg queue.JobMessage) {
					defer wg.Done()
					processJobWithLogging(ctx, log, processor, msg.JobID, msg.FileContent)
				}(jobMsg)
			} else {
				log.WithField("job_id", jobData).Info("Job ID dequeued from queue (legacy format)")

				job, err := processor.GetJob(ctx, jobData)
				if err != nil {
					log.WithFields(logrus.Fields{
						"job_id": jobData,
						"error":  err.Error(),
					}).Error("Failed to get job for backoff calculation")
					wg.Add(1)
					go func(jID string) {
						defer wg.Done()
						processJobWithLogging(ctx, log, processor, jID, "")
					}(jobData)
					continue
				}

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

				wg.Add(1)
				go func(jID string) {
					defer wg.Done()
					processJobWithLogging(ctx, log, processor, jID, "")
				}(jobData)
			}
		}
	}
}

func processJobWithLogging(ctx context.Context, log *logger.Logger, processor *worker.Processor, jobID string, fileContent string) {
	startTime := time.Now()

	log.WithField("job_id", jobID).Info("Processing job")

	metrics.RecordJobProcessing()
	metrics.IncrementActiveJobs()
	defer metrics.DecrementActiveJobs()

	err := processor.ProcessJob(ctx, jobID, fileContent)

	duration := time.Since(startTime)

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
