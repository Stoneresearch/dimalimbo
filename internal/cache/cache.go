package cache

import (
	"sync"
	"time"

	"github.com/aal/dimalimbo/internal/model"
)

type cachedItem struct {
	winners   []model.Winner
	expiresAt time.Time
}

type TopWinnersCache struct {
	mu    sync.RWMutex
	items map[int]cachedItem // keyed by limit
	ttl   time.Duration
}

func NewTopWinnersCache(ttl time.Duration) *TopWinnersCache {
	return &TopWinnersCache{
		items: make(map[int]cachedItem),
		ttl:   ttl,
	}
}

func (c *TopWinnersCache) Get(limit int) ([]model.Winner, bool) {
	c.mu.RLock()
	item, ok := c.items[limit]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, limit)
		c.mu.Unlock()
		return nil, false
	}
	return item.winners, true
}

func (c *TopWinnersCache) Set(limit int, winners []model.Winner) {
	c.mu.Lock()
	c.items[limit] = cachedItem{winners: winners, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *TopWinnersCache) InvalidateAll() {
	c.mu.Lock()
	c.items = make(map[int]cachedItem)
	c.mu.Unlock()
}
