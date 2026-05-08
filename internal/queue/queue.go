package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sarabjeet/golang-backend-task/internal/config"
)

const (
	jobQueueKey = "edi:jobs:queue"
)

// Queue handles Redis queue operations
type Queue struct {
	client *redis.Client
}

// RedisQueue is an alias for Queue to match handler expectations
type RedisQueue = Queue

// JobMessage represents a job message in the queue
type JobMessage struct {
	JobID       string    `json:"job_id"`
	FileName    string    `json:"file_name"`
	FileContent string    `json:"file_content"`
	CreatedAt   time.Time `json:"created_at"`
}

// New creates a new Queue instance
func New(cfg *config.Config) (*Queue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Queue{
		client: client,
	}, nil
}

// NewRedisQueue creates a new RedisQueue instance
func NewRedisQueue(cfg *config.RedisConfig) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Queue{
		client: client,
	}, nil
}

// Close closes the Redis connection
func (q *Queue) Close() error {
	return q.client.Close()
}

// Enqueue adds a job message to the queue
func (q *Queue) Enqueue(ctx context.Context, jobMsg *JobMessage) error {
	// Serialize job message to JSON
	data, err := json.Marshal(jobMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal job message: %w", err)
	}

	err = q.client.RPush(ctx, jobQueueKey, data).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}
	return nil
}

// EnqueueJobID adds a job ID to the queue (for backward compatibility)
func (q *Queue) EnqueueJobID(ctx context.Context, jobID string) error {
	err := q.client.RPush(ctx, jobQueueKey, jobID).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}
	return nil
}

// Dequeue retrieves a job ID from the queue
// Returns empty string if queue is empty
func (q *Queue) Dequeue(ctx context.Context) (string, error) {
	result, err := q.client.LPop(ctx, jobQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // Queue is empty
		}
		return "", fmt.Errorf("failed to dequeue job: %w", err)
	}
	return result, nil
}

// DequeueBlocking retrieves a job ID from the queue with blocking
// Waits for the specified timeout duration if queue is empty
func (q *Queue) DequeueBlocking(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := q.client.BLPop(ctx, timeout, jobQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // Timeout or queue empty
		}
		return "", fmt.Errorf("failed to dequeue job: %w", err)
	}
	if len(result) < 2 {
		return "", nil
	}
	return result[1], nil // result[0] is the key, result[1] is the value
}

// QueueLength returns the number of items in the queue
func (q *Queue) QueueLength(ctx context.Context) (int64, error) {
	length, err := q.client.LLen(ctx, jobQueueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}

// Size returns the number of items in the queue (alias for QueueLength)
func (q *Queue) Size(ctx context.Context) (int64, error) {
	return q.QueueLength(ctx)
}
