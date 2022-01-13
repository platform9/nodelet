package cache_test

import (
	"testing"
	"time"

	"github.com/platform9/pf9-qbert/bouncer/pkg/cache"
)

func expectEntry(t *testing.T, c *cache.LRUExpireCache, key cache.Key, value interface{}) {
	result, ok := c.Get(key)
	if !ok || result != value {
		t.Errorf("Expected cache[%v]: %v, got %v", key, value, result)
	}
}

func expectNotEntry(t *testing.T, c *cache.LRUExpireCache, key cache.Key) {
	if result, ok := c.Get(key); ok {
		t.Errorf("Expected cache[%v] to be empty, got %v", key, result)
	}
}

func TestGet(t *testing.T) {
	c, _ := cache.NewLRUExpireCache(100)
	c.Add("key", "value", 100*time.Millisecond)
	expectEntry(t, c, "key", "value")
}

func TestExpiredGet(t *testing.T) {
	c, _ := cache.NewLRUExpireCache(100)
	c.Add("expired-key", "value", 100*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	expectNotEntry(t, c, "expired-key")
}

func TestOverflow(t *testing.T) {
	c, _ := cache.NewLRUExpireCache(4)
	c.Add("key1", "val1", 1*time.Hour)
	c.Add("key2", "val2", 1*time.Hour)
	c.Add("key3", "val3", 1*time.Hour)
	c.Add("key4", "val4", 1*time.Hour)
	c.Add("key5", "val5", 1*time.Hour)
	expectNotEntry(t, c, "key1")
	expectEntry(t, c, "key2", "val2")
	expectEntry(t, c, "key3", "val3")
	expectEntry(t, c, "key4", "val4")
	expectEntry(t, c, "key5", "val5")
}
