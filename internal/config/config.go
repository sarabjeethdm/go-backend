package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the worker service
type Config struct {
	MongoDB MongoDBConfig
	Redis   RedisConfig
	Worker  WorkerConfig
	Server  ServerConfig
}

// MongoDBConfig holds MongoDB configuration
type MongoDBConfig struct {
	URI      string
	Database string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// WorkerConfig holds worker-specific configuration
type WorkerConfig struct {
	MaxRetries      int
	PollInterval    int // in seconds
	InitialBackoff  int // in seconds
	ShutdownTimeout int // in seconds
}

// ServerConfig holds API server configuration
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGODB_DATABASE", "edi_processor"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Worker: WorkerConfig{
			MaxRetries:      getEnvAsInt("WORKER_MAX_RETRIES", 3),
			PollInterval:    getEnvAsInt("WORKER_POLL_INTERVAL", 1),
			InitialBackoff:  getEnvAsInt("WORKER_INITIAL_BACKOFF", 2),
			ShutdownTimeout: getEnvAsInt("WORKER_SHUTDOWN_TIMEOUT", 30),
		},
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			ReadTimeout:     time.Duration(getEnvAsInt("SERVER_READ_TIMEOUT", 15)) * time.Second,
			WriteTimeout:    time.Duration(getEnvAsInt("SERVER_WRITE_TIMEOUT", 15)) * time.Second,
			ShutdownTimeout: time.Duration(getEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 30)) * time.Second,
		},
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
