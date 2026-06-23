package utils

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisMu       sync.Mutex
	redisClient   *redis.Client
	redisDisabled bool
)

func Redis(ctx context.Context) *redis.Client {
	redisMu.Lock()
	defer redisMu.Unlock()
	if redisDisabled {
		return nil
	}
	if redisClient != nil {
		return redisClient
	}
	raw := strings.TrimSpace(os.Getenv("REDIS"))
	if raw == "" {
		redisDisabled = true
		return nil
	}
	options, err := redis.ParseURL(raw)
	if err != nil {
		redisDisabled = true
		return nil
	}
	client := redis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		redisDisabled = true
		return nil
	}
	redisClient = client
	return redisClient
}
