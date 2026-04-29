package aggregatecache

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type ttlCacheEntry struct {
	value     any
	expiresAt time.Time
}

type ttlCache struct {
	mu    sync.RWMutex
	items map[string]ttlCacheEntry
}

func newTTLCache() *ttlCache {
	return &ttlCache{items: map[string]ttlCacheEntry{}}
}

func (c *ttlCache) get(key string) (any, bool) {
	now := time.Now()
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if now.After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	return item.value, true
}

func (c *ttlCache) set(key string, value any, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	c.mu.Lock()
	c.items[key] = ttlCacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

var aggregateCache = newTTLCache()

// Value 返回带 TTL 的缓存值；loader 出错时不会写入缓存，避免缓存异常结果。
func Value[T any](key string, ttl time.Duration, load func() (T, error)) (T, error) {
	if raw, ok := aggregateCache.get(key); ok {
		if value, ok := raw.(T); ok {
			return value, nil
		}
	}
	value, err := load()
	if err != nil {
		return value, err
	}
	aggregateCache.set(key, value, ttl)
	return value, nil
}

// Key 将查询维度拼成稳定缓存 key。
func Key(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	return strings.Join(values, "|")
}

// TTL 按查询结束时间动态计算缓存时长：越早的历史数据缓存越久。
func TTL(endUTC string) time.Duration {
	end, err := time.Parse("2006-01-02T15:04:05Z", endUTC)
	if err != nil {
		return 30 * time.Second
	}
	age := time.Since(end)
	switch {
	case age >= 30*24*time.Hour:
		return 6 * time.Hour
	case age >= 24*time.Hour:
		return 1 * time.Hour
	case age >= time.Hour:
		return 5 * time.Minute
	default:
		return 30 * time.Second
	}
}
