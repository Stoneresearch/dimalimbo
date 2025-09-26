## DIMA LIMBO VOL.1

High-speed arcade dodger built with Go and Ebitengine (Ebiten v2). Local leaderboard stored in SQLite with an in-memory TTL cache for fast reads. Web build ships as WASM.

### Requirements

- Go 1.23+
- macOS/Windows/Linux

### Desktop Run

```bash
go run ./cmd/dimalimbo
```

Or build:

```bash
go build -o dimalimbo ./cmd/dimalimbo
./dimalimbo
```

### Web (WASM) Build

```bash
# Output to dist/
mkdir -p dist
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" dist/wasm_exec.js
cp web/index.html dist/index.html
GOOS=js GOARCH=wasm go build -o dist/dimalimbo.wasm ./cmd/dimalimbo
```

Serve `dist/` statically (any HTTP server). The loader in `web/index.html` includes fallbacks for MIME/404 issues.

### AI Backgrounds API

Start the secure background API (reads `REPLICATE_API_TOKEN` from env):

```bash
go run ./cmd/bgserver
```

Request a new background (returns JSON with `url`):

```bash
curl -s -X POST http://localhost:8787/api/background \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"colorful adventurous synthwave space, cinematic, detailed","width":1600,"height":900}'
```

In `settings.json` set:

```json
{
  "backgroundEndpoint": "http://localhost:8787/api/background",
  "backgroundUrl": "" ,
  "showGrid": false
}
```

When `backgroundUrl` is empty, you can call the endpoint yourself and paste the returned `url` into settings.

### GitHub Pages (CI/CD)

This repo includes a workflow at `.github/workflows/gh-pages.yml` that:
- Builds the WASM binary with the same steps above
- Uploads `dist/` as an artifact
- Publishes to GitHub Pages on pushes to `main`

Checklist:
- In repo settings, enable Pages with Source: GitHub Actions
- Ensure `web/index.html` exists and references `dimalimbo.wasm` and `wasm_exec.js`
- Push to `main` to trigger deploy

### Controls

- Arrow keys / WASD / Gamepad: Move
- Mouse drag / Touch: Move toward pointer (mobile-friendly)
- Space: Start (title) / Return to title (leaderboard)
- Enter: Submit name after game over

### Settings

Configuration is loaded from `settings.json` if present; otherwise defaults from `internal/settings/settings.go` are used.

Key options:
- `fullscreen` (bool)
- `windowWidth`, `windowHeight` (ints)
- `uiScale` (float) – scales UI sizes
- `postFXEnabled` (bool), `shaderIntensity` (float32)
- `renderScale` (float64) – internal render scale for performance/clarity
- `dbPath` (string) – SQLite file path
- `musicStyle` (string) – `synthwave` or `classic`
- `backgroundStyle` (string) – `neon_space` or `retro_mario`

### Troubleshooting

- Web 404 or wrong MIME type:
  - Ensure `index.html`, `wasm_exec.js`, `dimalimbo.wasm` are in the same folder being served.
  - The loader first HEAD-checks `dimalimbo.wasm` and falls back to ArrayBuffer instantiation.

- Black screen or shader artifacts:
  - Set `postFXEnabled` to false or reduce `shaderIntensity` in `settings.json`.

- Performance on low-end devices:
  - Reduce `renderScale` (e.g., 0.8) or enable `lowPower`.

- Leaderboard empty:
  - On web, scores are saved to `localStorage` after you submit a name. In name entry, press Enter/Space or tap.
  - On desktop, ensure the process can write the `dimalimbo.db` file.

- Database location:
  - A SQLite file `dimalimbo.db` is created next to your binary by default.

### Project Structure

```
cmd/dimalimbo/main.go         # Entrypoint
internal/game/game.go         # Game states, update/draw loop
internal/assets/shaders.go    # Post-processing shader (neon/CRT)
internal/audio/               # Simple SFX manager
internal/storage/sqlite.go    # SQLite access + caching
internal/cache/cache.go       # TTL cache for top winners
internal/model/winner.go      # Winner model
web/index.html                # WASM loader + responsive shell
dist/                         # Built artifacts for web deploy
```

### Deploy to GitHub Pages

This repo ships a workflow that builds to `dist/` and deploys to Pages. On push to `main`, artifacts are uploaded and published.

### Credits

- Built with `github.com/hajimehoshi/ebiten/v2`
- Fonts: Go Bold (title), Go Regular (UI)
f