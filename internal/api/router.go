package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sarabjeet/golang-backend-task/internal/metrics"
	"github.com/sarabjeet/golang-backend-task/internal/queue"
	"github.com/sarabjeet/golang-backend-task/internal/storage"
)

func SetupRouter(db *storage.MongoDB, q *queue.RedisQueue) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware())
	router.Use(MetricsMiddleware())
	router.Use(CORSMiddleware())

	h := NewHandler(db, q)

	router.GET("/health", h.HealthCheck)
	router.POST("/jobs", h.CreateJob)
	router.GET("/jobs/:job_id", h.GetJobStatus)
	router.GET("/jobs/:job_id/result", h.GetJobResult)

	return router
}

func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
	})
}

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

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		metrics.RecordAPIRequestDuration(c.Request.Method, c.FullPath(), duration)
		metrics.RecordAPIRequest(c.Request.Method, c.FullPath(), c.Writer.Status())
	}
}
