package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisBackend implements StorageBackend using Redis
type RedisBackend struct {
	client *redis.Client
	url    string
}

// NewRedisBackend creates a new Redis storage backend
func NewRedisBackend(url string) *RedisBackend {
	return &RedisBackend{url: url}
}

func (r *RedisBackend) Init() error {
	opts, err := redis.ParseURL(r.url)
	if err != nil {
		return fmt.Errorf("failed to parse redis URL: %w", err)
	}

	r.client = redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	log.Info().Str("type", "redis").Msg("Storage initialized")
	return nil
}

func (r *RedisBackend) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *RedisBackend) Type() string {
	return StorageTypeRedis
}

// Redis key format: gagos:{bucket}:{key}
func (r *RedisBackend) redisKey(bucket, key string) string {
	return fmt.Sprintf("gagos:%s:%s", bucket, key)
}

// Redis set key for tracking all keys in a bucket
func (r *RedisBackend) bucketSetKey(bucket string) string {
	return fmt.Sprintf("gagos:%s:__keys__", bucket)
}

func (r *RedisBackend) Set(bucket, key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use pipeline for atomicity
	pipe := r.client.Pipeline()
	pipe.Set(ctx, r.redisKey(bucket, key), value, 0)
	pipe.SAdd(ctx, r.bucketSetKey(bucket), key)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisBackend) Get(bucket, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	val, err := r.client.Get(ctx, r.redisKey(bucket, key)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (r *RedisBackend) Delete(bucket, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.redisKey(bucket, key))
	pipe.SRem(ctx, r.bucketSetKey(bucket), key)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisBackend) List(bucket string) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all keys in bucket
	keys, err := r.client.SMembers(ctx, r.bucketSetKey(bucket)).Result()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, nil
	}

	// Build full key names
	redisKeys := make([]string, len(keys))
	for i, k := range keys {
		redisKeys[i] = r.redisKey(bucket, k)
	}

	// Get all values
	vals, err := r.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, err
	}

	var items [][]byte
	for _, v := range vals {
		if v != nil {
			if str, ok := v.(string); ok {
				items = append(items, []byte(str))
			}
		}
	}
	return items, nil
}

func (r *RedisBackend) ListKeys(bucket string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return r.client.SMembers(ctx, r.bucketSetKey(bucket)).Result()
}

// GetClient returns the underlying Redis client
func (r *RedisBackend) GetClient() *redis.Client {
	return r.client
}
