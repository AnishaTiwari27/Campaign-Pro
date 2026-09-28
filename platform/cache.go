package platform

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client with the two operations analytics-service and
// catalog-service actually need: cache a JSON-able value under a key with a
// TTL, and read it back. A cache miss (key absent, Redis unreachable, or a
// decode failure) always looks the same to the caller — "not cached" — so a
// down Redis degrades to "always recompute," never a hard failure.
type Cache struct {
	rdb *redis.Client
}

func ConnectCache(addr string) *Cache {
	return &Cache{rdb: redis.NewClient(&redis.Options{Addr: addr})}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Cache) Get(ctx context.Context, key string, dest any) bool {
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return false // miss or Redis unavailable — caller recomputes
	}
	return json.Unmarshal(raw, dest) == nil
}

func (c *Cache) Set(ctx context.Context, key string, v any, ttl time.Duration) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.rdb.Set(ctx, key, raw, ttl) // best-effort; ignore errors, same reasoning as EventBus.Publish
}

// FlushPrefix deletes every key starting with prefix — used to invalidate
// all cached aggregate views (every filter combination) in one shot when a
// campaign.created event arrives, rather than tracking which cached
// responses a given write could have affected.
func (c *Cache) FlushPrefix(ctx context.Context, prefix string) {
	iter := c.rdb.Scan(ctx, 0, prefix+"*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		c.rdb.Del(ctx, keys...)
	}
}
