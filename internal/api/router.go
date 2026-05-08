package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sarabjeet/golang-backend-task/internal/metrics"
	"github.com/sarabjeet/golang-backend-task/internal/queue"
	"github.com/sarabjeet/golang-backend-task/internal/storage"
)

// SetupRouter configures and returns the Gin router
func SetupRouter(db *storage.MongoDB, q *queue.RedisQueue) *gin.Engine {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware())
	router.Use(MetricsMiddleware())
	router.Use(CORSMiddleware())

	// Create handler
	h := NewHandler(db, q)

	// Routes
	router.GET("/health", h.HealthCheck)
	router.POST("/jobs", h.CreateJob)
	router.GET("/jobs/:job_id", h.GetJobStatus)
	router.GET("/jobs/:job_id/result", h.GetJobResult)

	return router
}

// LoggerMiddleware is a custom logger middleware
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
	})
}

// CORSMiddleware handles CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// MetricsMiddleware records metrics for each API request
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		metrics.RecordAPIRequestDuration(c.Request.Method, c.FullPath(), duration)
		metrics.RecordAPIRequest(c.Request.Method, c.FullPath(), c.Writer.Status())
	}
}
