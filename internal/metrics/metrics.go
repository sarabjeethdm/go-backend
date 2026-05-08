package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edi_jobs_total",
			Help: "Total number of EDI jobs processed",
		},
		[]string{"status"},
	)

	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edi_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status"},
	)

	JobProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "edi_job_processing_duration_seconds",
			Help:    "Duration of job processing in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0},
		},
	)

	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "edi_api_request_duration_seconds",
			Help:    "Duration of API requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"method", "path"},
	)

	ActiveJobs = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "edi_active_jobs",
			Help: "Number of currently active/processing jobs",
		},
	)

	RedisQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "edi_redis_queue_size",
			Help: "Current size of the Redis job queue",
		},
	)

	EDITransactionsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edi_transactions_processed_total",
			Help: "Total number of EDI transactions processed",
		},
		[]string{"transaction_type"},
	)

	JobRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edi_job_retries_total",
			Help: "Total number of job retry attempts",
		},
		[]string{"retry_count"},
	)
)

func Init() {
}

func RecordJobCreated() {
	JobsTotal.WithLabelValues("created").Inc()
}

func RecordJobCompleted() {
	JobsTotal.WithLabelValues("completed").Inc()
}

func RecordJobFailed() {
	JobsTotal.WithLabelValues("failed").Inc()
}

func RecordJobProcessing() {
	JobsTotal.WithLabelValues("processing").Inc()
}

func RecordAPIRequest(method, path string, status int) {
	APIRequestsTotal.WithLabelValues(method, path, http.StatusText(status)).Inc()
}

func RecordJobProcessingDuration(duration time.Duration) {
	JobProcessingDuration.Observe(duration.Seconds())
}

func RecordAPIRequestDuration(method, path string, duration time.Duration) {
	APIRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

func IncrementActiveJobs() {
	ActiveJobs.Inc()
}

func DecrementActiveJobs() {
	ActiveJobs.Dec()
}

func SetRedisQueueSize(size float64) {
	RedisQueueSize.Set(size)
}

func RecordEDITransaction(transactionType string) {
	EDITransactionsProcessed.WithLabelValues(transactionType).Inc()
}

func RecordJobRetry(retryCount int) {
	JobRetries.WithLabelValues(fmt.Sprintf("%d", retryCount)).Inc()
}
