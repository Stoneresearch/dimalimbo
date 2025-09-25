package main

import (
	"log"
	"time"

	"github.com/aal/dimalimbo/internal/game"
	"github.com/aal/dimalimbo/internal/settings"
	"github.com/aal/dimalimbo/internal/storage"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	cfg := settings.Load("settings.json")
	store, err := storage.NewStorage(cfg.DBPath, time.Duration(cfg.CacheTTLSeconds)*time.Second)
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}

	g := game.New(store, cfg)
	if cfg.Fullscreen {
		ebiten.SetFullscreen(true)
	}
	if cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		ebiten.SetWindowSize(cfg.WindowWidth, cfg.WindowHeight)
	} else {
		ebiten.SetWindowSize(800, 600)
	}
	ebiten.SetWindowTitle("DIMA LIMBO VOL.1")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("game exited with error: %v", err)
	}
}
