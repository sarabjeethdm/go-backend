package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sarabjeet/golang-backend-task/internal/api"
	"github.com/sarabjeet/golang-backend-task/internal/config"
	"github.com/sarabjeet/golang-backend-task/internal/logger"
	"github.com/sarabjeet/golang-backend-task/internal/metrics"
	"github.com/sarabjeet/golang-backend-task/internal/queue"
	"github.com/sarabjeet/golang-backend-task/internal/storage"
)

func main() {
	// Initialize logger
	logger.Init()
	logger.Info("Starting EDI Processing API Server")

	// Initialize metrics
	metrics.Init()
	logger.Info("Metrics initialized")

	// Load configuration
	cfg := config.Load()
	logger.WithFields(map[string]interface{}{
		"port":        cfg.Server.Port,
		"mongodb_uri": cfg.MongoDB.URI,
		"redis_host":  cfg.Redis.Host,
		"redis_port":  cfg.Redis.Port,
	}).Info("Configuration loaded")

	// Initialize MongoDB
	db, err := storage.NewMongoDB(&cfg.MongoDB)
	if err != nil {
		logger.Fatalf("Failed to initialize MongoDB: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.Close(ctx); err != nil {
			logger.Errorf("Failed to close MongoDB connection: %v", err)
		}
	}()

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := db.CreateIndexes(ctx); err != nil {
		logger.Warnf("Failed to create MongoDB indexes: %v", err)
	}
	cancel()

	// Initialize Redis queue
	redisQueue, err := queue.NewRedisQueue(&cfg.Redis)
	if err != nil {
		logger.Fatalf("Failed to initialize Redis queue: %v", err)
	}
	defer func() {
		if err := redisQueue.Close(); err != nil {
			logger.Errorf("Failed to close Redis connection: %v", err)
		}
	}()

	// Setup router
	router := api.SetupRouter(db, redisQueue)

	// Add Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	logger.Info("Metrics endpoint configured at /metrics")

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.WithFields(map[string]interface{}{
			"port": cfg.Server.Port,
		}).Info("Server started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited gracefully")
}
