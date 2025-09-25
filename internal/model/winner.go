package model

import "time"

type Winner struct {
	ID        int64
	Name      string
	Score     int
	CreatedAt time.Time
}
