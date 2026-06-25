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

type memoryCacheItem struct {
	Raw       []byte
	ExpiresAt time.Time
}

type memoryRateItem struct {
	Count     int
	ExpiresAt time.Time
}

var (
	cacheMu     sync.Mutex
	cacheItems  = map[string]memoryCacheItem{}
	rateItems   = map[string]memoryRateItem{}
	redisClient *redis.Client
)

func CacheUsesRedis() bool {
	return strings.TrimSpace(os.Getenv("REDIS")) != ""
}

func CacheGet(ctx context.Context, key string, value any) (bool, error) {
	if CacheUsesRedis() {
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

	cacheMu.Lock()
	item, ok := cacheItems[key]
	if !ok || cacheExpired(item, time.Now()) {
		delete(cacheItems, key)
		cacheMu.Unlock()
		return false, nil
	}
	raw := append([]byte(nil), item.Raw...)
	cacheMu.Unlock()
	return true, json.Unmarshal(raw, value)
}

func CacheSet(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if CacheUsesRedis() {
		client, err := redisCache(ctx)
		if err != nil {
			return err
		}
		return client.Set(ctx, key, raw, ttl).Err()
	}

	item := memoryCacheItem{Raw: raw}
	if ttl > 0 {
		item.ExpiresAt = time.Now().Add(ttl)
	}
	cacheMu.Lock()
	cacheItems[key] = item
	cacheMu.Unlock()
	return nil
}

func CacheDelete(ctx context.Context, key string) error {
	if CacheUsesRedis() {
		client, err := redisCache(ctx)
		if err != nil {
			return err
		}
		return client.Del(ctx, key).Err()
	}

	cacheMu.Lock()
	delete(cacheItems, key)
	cacheMu.Unlock()
	return nil
}

func CacheAllow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 || window <= 0 {
		return true, nil
	}
	if CacheUsesRedis() {
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

	now := time.Now()
	cacheMu.Lock()
	item := rateItems[key]
	if item.ExpiresAt.IsZero() || !item.ExpiresAt.After(now) {
		item = memoryRateItem{ExpiresAt: now.Add(window)}
	}
	item.Count++
	rateItems[key] = item
	allowed := item.Count <= limit
	cacheMu.Unlock()
	return allowed, nil
}

func redisCache(ctx context.Context) (*redis.Client, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if redisClient != nil {
		return redisClient, nil
	}
	options, err := redis.ParseURL(strings.TrimSpace(os.Getenv("REDIS")))
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

func cacheExpired(item memoryCacheItem, now time.Time) bool {
	return !item.ExpiresAt.IsZero() && !item.ExpiresAt.After(now)
}

func ResetCacheForTest() {
	cacheMu.Lock()
	if redisClient != nil {
		_ = redisClient.Close()
		redisClient = nil
	}
	cacheItems = map[string]memoryCacheItem{}
	rateItems = map[string]memoryRateItem{}
	cacheMu.Unlock()
}
