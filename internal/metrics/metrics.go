package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// JobsTotal tracks the total number of jobs by status
	JobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edi_jobs_total",
			Help: "Total number of EDI jobs processed",
		},
		[]string{"status"},
	)

	// APIRequestsTotal tracks the total number of API requests
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edi_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status"},
	)

	// JobProcessingDuration tracks the duration of job processing
	JobProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "edi_job_processing_duration_seconds",
			Help:    "Duration of job processing in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0},
		},
	)

	// APIRequestDuration tracks the duration of API requests
	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "edi_api_request_duration_seconds",
			Help:    "Duration of API requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"method", "path"},
	)

	// ActiveJobs tracks the number of currently processing jobs
	ActiveJobs = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "edi_active_jobs",
			Help: "Number of currently active/processing jobs",
		},
	)

	// RedisQueueSize tracks the size of the Redis queue
	RedisQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "edi_redis_queue_size",
			Help: "Current size of the Redis job queue",
		},
	)

	// EDITransactionsProcessed tracks transactions processed from EDI files
	EDITransactionsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edi_transactions_processed_total",
			Help: "Total number of EDI transactions processed",
		},
		[]string{"transaction_type"},
	)

	// JobRetries tracks the number of job retries
	JobRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edi_job_retries_total",
			Help: "Total number of job retry attempts",
		},
		[]string{"retry_count"},
	)
)

// Init initializes the metrics system
// This is a placeholder for future initialization needs
func Init() {
	// Currently, promauto registers metrics automatically
	// This function is here for future extensions like custom registries
}

// RecordJobCreated increments the counter for created jobs
func RecordJobCreated() {
	JobsTotal.WithLabelValues("created").Inc()
}

// RecordJobCompleted increments the counter for completed jobs
func RecordJobCompleted() {
	JobsTotal.WithLabelValues("completed").Inc()
}

// RecordJobFailed increments the counter for failed jobs
func RecordJobFailed() {
	JobsTotal.WithLabelValues("failed").Inc()
}

// RecordJobProcessing increments the counter for processing jobs
func RecordJobProcessing() {
	JobsTotal.WithLabelValues("processing").Inc()
}

// RecordAPIRequest records an API request with method, path, and status
func RecordAPIRequest(method, path string, status int) {
	APIRequestsTotal.WithLabelValues(method, path, http.StatusText(status)).Inc()
}

// RecordJobProcessingDuration records the duration of job processing
func RecordJobProcessingDuration(duration time.Duration) {
	JobProcessingDuration.Observe(duration.Seconds())
}

// RecordAPIRequestDuration records the duration of an API request
func RecordAPIRequestDuration(method, path string, duration time.Duration) {
	APIRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// IncrementActiveJobs increments the active jobs gauge
func IncrementActiveJobs() {
	ActiveJobs.Inc()
}

// DecrementActiveJobs decrements the active jobs gauge
func DecrementActiveJobs() {
	ActiveJobs.Dec()
}

// SetRedisQueueSize sets the current Redis queue size
func SetRedisQueueSize(size float64) {
	RedisQueueSize.Set(size)
}

// RecordEDITransaction records a processed EDI transaction
func RecordEDITransaction(transactionType string) {
	EDITransactionsProcessed.WithLabelValues(transactionType).Inc()
}

// RecordJobRetry records a job retry attempt
func RecordJobRetry(retryCount int) {
	JobRetries.WithLabelValues(fmt.Sprintf("%d", retryCount)).Inc()
}
