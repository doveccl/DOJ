package cache

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisGetSetDelete(t *testing.T) {
	startRedis(t)
	ResetForTest()
	t.Cleanup(ResetForTest)

	type item struct {
		Value string `json:"value"`
	}
	if err := Set(t.Context(), "test:key", item{Value: "ok"}, time.Minute); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	var got item
	found, err := Get(t.Context(), "test:key", &got)
	if err != nil || !found || got.Value != "ok" {
		t.Fatalf("get cache found=%v got=%+v err=%v", found, got, err)
	}
	if err := Delete(t.Context(), "test:key"); err != nil {
		t.Fatalf("delete cache: %v", err)
	}
	found, err = Get(t.Context(), "test:key", &got)
	if err != nil || found {
		t.Fatalf("deleted cache found=%v err=%v", found, err)
	}
}

func TestRedisCacheTTL(t *testing.T) {
	redis := startRedis(t)
	ResetForTest()
	t.Cleanup(ResetForTest)

	if err := Set(t.Context(), "test:ttl", "value", time.Millisecond); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	redis.FastForward(time.Second)
	var got string
	found, err := Get(t.Context(), "test:ttl", &got)
	if err != nil || found {
		t.Fatalf("expired cache found=%v value=%q err=%v", found, got, err)
	}
}

func TestRedisAllow(t *testing.T) {
	startRedis(t)
	ResetForTest()
	t.Cleanup(ResetForTest)

	for i := 0; i < 2; i++ {
		allowed, err := Allow(t.Context(), "test:rate", 2, time.Minute)
		if err != nil || !allowed {
			t.Fatalf("rate allow #%d allowed=%v err=%v", i+1, allowed, err)
		}
	}
	allowed, err := Allow(t.Context(), "test:rate", 2, time.Minute)
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
