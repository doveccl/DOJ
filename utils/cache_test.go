package utils

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisCacheGetSetDelete(t *testing.T) {
	startRedis(t)
	ResetCacheForTest()
	t.Cleanup(ResetCacheForTest)

	type item struct {
		Value string `json:"value"`
	}
	if err := CacheSet(t.Context(), "test:key", item{Value: "ok"}, time.Minute); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	var got item
	found, err := CacheGet(t.Context(), "test:key", &got)
	if err != nil || !found || got.Value != "ok" {
		t.Fatalf("get cache found=%v got=%+v err=%v", found, got, err)
	}
	if err := CacheDelete(t.Context(), "test:key"); err != nil {
		t.Fatalf("delete cache: %v", err)
	}
	found, err = CacheGet(t.Context(), "test:key", &got)
	if err != nil || found {
		t.Fatalf("deleted cache found=%v err=%v", found, err)
	}
}

func TestRedisCacheTTL(t *testing.T) {
	redis := startRedis(t)
	ResetCacheForTest()
	t.Cleanup(ResetCacheForTest)

	if err := CacheSet(t.Context(), "test:ttl", "value", time.Millisecond); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	redis.FastForward(time.Second)
	var got string
	found, err := CacheGet(t.Context(), "test:ttl", &got)
	if err != nil || found {
		t.Fatalf("expired cache found=%v value=%q err=%v", found, got, err)
	}
}

func TestRedisCacheAllow(t *testing.T) {
	startRedis(t)
	ResetCacheForTest()
	t.Cleanup(ResetCacheForTest)

	for i := 0; i < 2; i++ {
		allowed, err := CacheAllow(t.Context(), "test:rate", 2, time.Minute)
		if err != nil || !allowed {
			t.Fatalf("rate allow #%d allowed=%v err=%v", i+1, allowed, err)
		}
	}
	allowed, err := CacheAllow(t.Context(), "test:rate", 2, time.Minute)
	if err != nil || allowed {
		t.Fatalf("rate should block third request allowed=%v err=%v", allowed, err)
	}
}

func startRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	server := miniredis.RunT(t)
	t.Setenv("REDIS", "redis://"+server.Addr()+"/0")
	return server
}
