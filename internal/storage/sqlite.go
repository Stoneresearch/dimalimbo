package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"

	"github.com/aal/dimalimbo/internal/cache"
	"github.com/aal/dimalimbo/internal/model"
)

type Storage struct {
	db    *sql.DB
	cache *cache.TopWinnersCache
}

func NewStorage(path string, cacheTTL time.Duration) (*Storage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Storage{db: db, cache: cache.NewTopWinnersCache(cacheTTL)}, nil
}

func initSchema(db *sql.DB) error {
	const schema = `
	CREATE TABLE IF NOT EXISTS winners (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		score INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_winners_score ON winners(score DESC);
	`
	_, err := db.Exec(schema)
	return err
}

func (s *Storage) SaveWinner(name string, score int) error {
	if name == "" {
		return errors.New("name required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, "INSERT INTO winners(name, score) VALUES(?, ?)", name, score)
	if err == nil {
		s.cache.InvalidateAll()
	}
	return err
}

func (s *Storage) TopWinners(limit int) ([]model.Winner, error) {
	if limit <= 0 {
		limit = 10
	}
	if winners, ok := s.cache.Get(limit); ok {
		return winners, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, score, COALESCE(created_at, CURRENT_TIMESTAMP) FROM winners ORDER BY score DESC, id ASC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Winner
	for rows.Next() {
		var w model.Winner
		var ts time.Time
		if err := rows.Scan(&w.ID, &w.Name, &w.Score, &ts); err != nil {
			return nil, err
		}
		w.CreatedAt = ts
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.cache.Set(limit, out)
	return out, nil
}

// Reset removes all winners from the leaderboard.
func (s *Storage) Reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := s.db.ExecContext(ctx, "DELETE FROM winners"); err != nil {
		return err
	}
	s.cache.InvalidateAll()
	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
