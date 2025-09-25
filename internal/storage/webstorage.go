//go:build js
// +build js

package storage

import (
    "encoding/json"
    "syscall/js"
    "time"

    "github.com/stoneresearch/dimalimbo/internal/cache"
    "github.com/stoneresearch/dimalimbo/internal/model"
)

type Storage struct {
    cache *cache.TopWinnersCache
}

func NewStorage(_ string, cacheTTL time.Duration) (*Storage, error) {
    return &Storage{cache: cache.NewTopWinnersCache(cacheTTL)}, nil
}

func ls() js.Value { return js.Global().Get("localStorage") }

func (s *Storage) SaveWinner(name string, score int) error {
    winners, _ := s.TopWinners(1000)
    w := model.Winner{ID: time.Now().UnixNano(), Name: name, Score: score, CreatedAt: time.Now()}
    winners = append(winners, w)
    b, _ := json.Marshal(winners)
    ls().Call("setItem", "dimalimbo_winners", string(b))
    s.cache.InvalidateAll()
    return nil
}

func (s *Storage) TopWinners(limit int) ([]model.Winner, error) {
    if limit <= 0 { limit = 10 }
    if w, ok := s.cache.Get(limit); ok { return w, nil }
    raw := ls().Call("getItem", "dimalimbo_winners").String()
    winners := []model.Winner{}
    if raw != "" { _ = json.Unmarshal([]byte(raw), &winners) }
    // sort by score desc
    for i := 0; i < len(winners); i++ {
        for j := i+1; j < len(winners); j++ {
            if winners[j].Score > winners[i].Score {
                winners[i], winners[j] = winners[j], winners[i]
            }
        }
    }
    if len(winners) > limit { winners = winners[:limit] }
    s.cache.Set(limit, winners)
    return winners, nil
}

func (s *Storage) Reset() error {
    ls().Call("removeItem", "dimalimbo_winners")
    s.cache.InvalidateAll()
    return nil
}

func (s *Storage) Close() error { return nil }


