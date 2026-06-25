package utils

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultRedis = "redis://localhost:6379/0"

var (
	cacheMu     sync.Mutex
	redisClient *redis.Client
)

func CachePing(ctx context.Context) error {
	client, err := redisCache(ctx)
	if err != nil {
		return err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	return client.Ping(pingCtx).Err()
}

func CacheGet(ctx context.Context, key string, value any) (bool, error) {
	client, err := redisCache(ctx)
	if err != nil {
		return false, err
	}
	raw, err := client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(raw, value)
}

func CacheSet(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	client, err := redisCache(ctx)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, raw, ttl).Err()
}

func CacheDelete(ctx context.Context, key string) error {
	client, err := redisCache(ctx)
	if err != nil {
		return err
	}
	return client.Del(ctx, key).Err()
}

func CacheAllow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 || window <= 0 {
		return true, nil
	}
	client, err := redisCache(ctx)
	if err != nil {
		return false, err
	}
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := client.Expire(ctx, key, window).Err(); err != nil {
			return false, err
		}
	}
	return count <= int64(limit), nil
}

func redisCache(ctx context.Context) (*redis.Client, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if redisClient != nil {
		return redisClient, nil
	}
	options, err := redis.ParseURL(redisURL())
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	redisClient = client
	return redisClient, nil
}

func redisURL() string {
	if value := strings.TrimSpace(os.Getenv("REDIS")); value != "" {
		return value
	}
	return defaultRedis
}

func ResetCacheForTest() {
	cacheMu.Lock()
	if redisClient != nil {
		_ = redisClient.Close()
		redisClient = nil
	}
	cacheMu.Unlock()
}
