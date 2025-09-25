# DIMA LIMBO VOL.1 (Go / Ebitengine)

A 2D arcade-style game built with Go and Ebitengine (Ebiten v2). Scores are saved to a local SQLite database (pure-Go driver) with an in-memory TTL cache for fast leaderboard reads.

## Requirements

- Go 1.21+
- macOS, Windows, or Linux

## Run

```bash
go run ./cmd/dimalimbo
```

Or build a binary:

```bash
go build -o dimalimbo ./cmd/dimalimbo && ./dimalimbo
```

## Controls

- Arrow keys: Move
- Space: Start (from title) / Return to title (from leaderboard)
- Enter your name after a game over, press Enter to submit

## Data

- A SQLite file `dimalimbo.db` is created beside the binary.
- The leaderboard view shows the top 10 scores.

## Project Structure

```
cmd/dimalimbo/main.go         # entrypoint
internal/game/game.go         # game states, update/draw loop
internal/storage/sqlite.go    # SQLite access + caching integration
internal/cache/cache.go       # simple TTL cache for top winners
internal/model/winner.go      # shared model for winners
```


