package cache

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
)

// LRUExpireCache is an LRU cache where each key has a TTL
// It is inspired by https://github.com/kubernetes/kubernetes/blob/release-1.5/pkg/util/cache/lruexpirecache.go
type LRUExpireCache struct {
	cache *lru.Cache
}

type Key interface{}

type entry struct {
	value      interface{}
	expireTime time.Time
}

// NewLRUExpireCache returns an LRUExpireCache of a fixed size
func NewLRUExpireCache(size int) (*LRUExpireCache, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &LRUExpireCache{cache}, nil
}

// Add stores the value under the key, and sets the key to expire after a duration
func (c *LRUExpireCache) Add(key Key, value interface{}, ttl time.Duration) {
	expireTime := time.Now().Add(ttl)
	_ = c.cache.Add(key, &entry{value, expireTime})
	time.AfterFunc(ttl, func() { c.cache.Remove(key) })
}

// Get returns the value found under the key and true. If the key is not found,
// or if the key has just expired, Get returns nil and false.
func (c *LRUExpireCache) Get(key Key) (interface{}, bool) {
	e, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	if time.Now().After(e.(*entry).expireTime) {
		c.cache.Remove(key)
		return nil, false
	}
	return e.(*entry).value, true
}
