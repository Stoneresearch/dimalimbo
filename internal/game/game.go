package game

import (
	"image/color"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"

	"github.com/stoneresearch/dimalimbo/internal/assets"
	aud "github.com/stoneresearch/dimalimbo/internal/audio"
	"github.com/stoneresearch/dimalimbo/internal/model"
	"github.com/stoneresearch/dimalimbo/internal/settings"
	"github.com/stoneresearch/dimalimbo/internal/storage"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

type GameState int

const (
	stateTitle GameState = iota
	statePlaying
	stateNameEntry
	stateLeaderboard
)

type rectangle struct {
	x float64
	y float64
	w float64
	h float64
}

func (r rectangle) intersects(o rectangle) bool {
	return r.x < o.x+o.w && r.x+r.w > o.x && r.y < o.y+o.h && r.y+r.h > o.y
}

type Game struct {
	state     GameState
	store     *storage.Storage
	player    rectangle
	playerVel float64
	obstacles []rectangle
	score     int
	frames    int
	nameInput string
	leaders   []model.Winner
	seeded    bool
	// visuals/audio
	offscreen *ebiten.Image
	shader    *ebiten.Shader
	shaderOn  bool
	shaderInt float32
	audio     *aud.Manager
	// parallax
	starsFar  []rectangle
	starsNear []rectangle
	// particles
	particles []particle
	// difficulty
	speed      float64
	spawnEvery int
	// settings
	cfg settings.Settings
	// fonts
	titleFace font.Face
}

type particle struct {
	x    float64
	y    float64
	vx   float64
	vy   float64
	life int
}

func New(store *storage.Storage, cfg settings.Settings) *Game {
	g := &Game{
		state:      stateTitle,
		store:      store,
		player:     rectangle{x: 60, y: screenHeight/2 - 20, w: 30, h: 30},
		playerVel:  4,
		obstacles:  make([]rectangle, 0, 16),
		shaderOn:   cfg.PostFXEnabled,
		shaderInt:  float32(cfg.ShaderIntensity),
		audio:      aud.NewManager(44100, cfg.MasterVolume),
		speed:      cfg.BaseSpeed,
		spawnEvery: cfg.SpawnEveryStart,
		cfg:        cfg,
	}
	// init parallax stars
	for i := 0; i < 64; i++ {
		g.starsFar = append(g.starsFar, rectangle{x: float64(rand.Intn(screenWidth)), y: float64(rand.Intn(screenHeight)), w: 2, h: 2})
	}
	for i := 0; i < 32; i++ {
		g.starsNear = append(g.starsNear, rectangle{x: float64(rand.Intn(screenWidth)), y: float64(rand.Intn(screenHeight)), w: 3, h: 3})
	}
	// compile shader
	if s, err := ebiten.NewShader([]byte(assets.NeonCRTShader)); err == nil {
		g.shader = s
	}
	// big bold title face
	if f, err := opentype.Parse(gobold.TTF); err == nil {
		size := 54.0 * cfg.UIScale
		face, ferr := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		if ferr == nil {
			g.titleFace = face
		}
	}
	return g
}

func (g *Game) spawnObstacle() {
	height := 40 + rand.Intn(140)
	y := rand.Intn(screenHeight - height)
	g.obstacles = append(g.obstacles, rectangle{
		x: screenWidth,
		y: float64(y),
		w: 20,
		h: float64(height),
	})
}

func (g *Game) resetPlay() {
	g.player = rectangle{x: 60, y: screenHeight/2 - 20, w: 30, h: 30}
	g.obstacles = g.obstacles[:0]
	g.score = 0
	g.frames = 0
	g.speed = 4
	g.spawnEvery = 60
}

func (g *Game) Update() error {
	if !g.seeded {
		rand.Seed(time.Now().UnixNano())
		g.seeded = true
		g.leaders, _ = g.store.TopWinners(g.cfg.TopN)
	}

	switch g.state {
	case stateTitle:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsGamepadButtonJustPressed(0, ebiten.GamepadButton0) {
			g.resetPlay()
			g.state = statePlaying
			if g.audio != nil {
				g.audio.PlayStart()
			}
		}
	case statePlaying:
		// Player movement
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.GamepadAxis(0, 1) < -0.2 {
			g.player.y -= g.playerVel
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.GamepadAxis(0, 1) > 0.2 {
			g.player.y += g.playerVel
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.GamepadAxis(0, 0) < -0.2 {
			g.player.x -= g.playerVel
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.GamepadAxis(0, 0) > 0.2 {
			g.player.x += g.playerVel
		}

		// clamp to screen
		if g.player.x < 0 {
			g.player.x = 0
		}
		if g.player.y < 0 {
			g.player.y = 0
		}
		if g.player.x+g.player.w > screenWidth {
			g.player.x = screenWidth - g.player.w
		}
		if g.player.y+g.player.h > screenHeight {
			g.player.y = screenHeight - g.player.h
		}

		// dynamic spawn frequency and speed increase
		if g.frames%g.spawnEvery == 0 {
			g.spawnObstacle()
		}
		if g.frames%g.cfg.AccelIntervalFrames == 0 {
			if g.spawnEvery > g.cfg.SpawnEveryMin {
				g.spawnEvery -= 4
			}
			g.speed += g.cfg.SpeedAccel
		}

		// move obstacles and detect collision
		alive := g.obstacles[:0]
		for _, o := range g.obstacles {
			o.x -= g.speed
			if o.x+o.w > 0 {
				alive = append(alive, o)
			}
			if g.player.intersects(o) {
				g.state = stateNameEntry
				g.nameInput = ""
				if g.audio != nil {
					g.audio.PlayHit()
				}
				return nil
			}
		}
		g.obstacles = alive

		g.frames++
		if g.frames%10 == 0 {
			g.score++
		}
	case stateNameEntry:
		for _, r := range ebiten.InputChars() {
			if r == '\n' || r == '\r' {
				continue
			}
			if len(g.nameInput) < 16 {
				g.nameInput += string(r)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.nameInput) > 0 {
			g.nameInput = g.nameInput[:len(g.nameInput)-1]
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			name := strings.TrimSpace(g.nameInput)
			if name == "" {
				name = "PLAYER"
			}
			_ = g.store.SaveWinner(name, g.score)
			g.leaders, _ = g.store.TopWinners(g.cfg.TopN)
			g.state = stateLeaderboard
			if g.audio != nil {
				g.audio.PlaySubmit()
			}
		}
	case stateLeaderboard:
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			_ = g.store.Reset()
			g.leaders, _ = g.store.TopWinners(g.cfg.TopN)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsGamepadButtonJustPressed(0, ebiten.GamepadButton0) {
			g.state = stateTitle
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// draw into offscreen for post-processing
	if g.offscreen == nil || g.offscreen.Bounds().Dx() != screenWidth || g.offscreen.Bounds().Dy() != screenHeight {
		g.offscreen = ebiten.NewImage(screenWidth, screenHeight)
	}
	g.offscreen.Fill(color.RGBA{R: 6, G: 8, B: 12, A: 255})

	// parallax background
	for i := range g.starsFar {
		s := &g.starsFar[i]
		s.x -= 0.3
		if s.x < 0 {
			s.x = screenWidth
			s.y = float64(rand.Intn(screenHeight))
		}
		ebitenutil.DrawRect(g.offscreen, s.x, s.y, s.w, s.h, color.RGBA{40, 40, 60, 255})
	}
	for i := range g.starsNear {
		s := &g.starsNear[i]
		s.x -= 0.8
		if s.x < 0 {
			s.x = screenWidth
			s.y = float64(rand.Intn(screenHeight))
		}
		ebitenutil.DrawRect(g.offscreen, s.x, s.y, s.w, s.h, color.RGBA{88, 88, 120, 255})
	}
	// 3D-ish ground grid
	horizonY := float64(screenHeight) * 0.65
	for i := 0; i < 12; i++ {
		t := float64(i) / 11.0
		x := t * float64(screenWidth)
		ebitenutil.DrawLine(g.offscreen, x, horizonY, x-80*(t-0.5), float64(screenHeight), color.RGBA{30, 30, 50, 180})
	}
	for r := 0; r < 10; r++ {
		y := horizonY + float64(r*r)*6
		ebitenutil.DrawLine(g.offscreen, 0, y, float64(screenWidth), y, color.RGBA{30, 30, 50, 160})
	}

	switch g.state {
	case stateTitle:
		drawTitle(g, g.cfg.UIScale)
	case statePlaying:
		// player
		// shadow
		ebitenutil.DrawRect(g.offscreen, g.player.x+4, g.player.y+4, g.player.w, g.player.h, color.RGBA{0, 0, 0, 120})
		ebitenutil.DrawRect(g.offscreen, g.player.x, g.player.y, g.player.w, g.player.h, color.RGBA{0, 255, 200, 255})
		// obstacles
		for _, o := range g.obstacles {
			ebitenutil.DrawRect(g.offscreen, o.x+3, o.y+3, o.w, o.h, color.RGBA{0, 0, 0, 120})
			ebitenutil.DrawRect(g.offscreen, o.x, o.y, o.w, o.h, color.RGBA{255, 40, 120, 255})
		}
		drawHUD(g, g.cfg.UIScale)
	case stateNameEntry:
		drawNameEntry(g, g.cfg.UIScale)
	case stateLeaderboard:
		drawLeaderboard(g, g.cfg.UIScale)
	}

	// post-process
	if g.shader != nil && g.shaderOn {
		opts := &ebiten.DrawRectShaderOptions{}
		opts.Images[0] = g.offscreen
		opts.Uniforms = map[string]interface{}{
			"time":       float32(g.frames) / 60.0,
			"intensity":  g.shaderInt,
			"resolution": []float32{float32(screenWidth), float32(screenHeight)},
		}
		screen.DrawRectShader(screenWidth, screenHeight, g.shader, opts)
	} else {
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(g.offscreen, op)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// small int to string without fmt to avoid allocs in hot path
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	b := [20]byte{}
	i := len(b)
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// UI helpers for bold/outlined text (brutalist look)
func drawShadowedText(dst *ebiten.Image, face font.Face, s string, x, y int, fg, shadow color.Color) {
	text.Draw(dst, s, face, x+2, y+2, shadow)
	text.Draw(dst, s, face, x, y, fg)
}

func drawTitle(g *Game, _ float64) {
	face := g.titleFace
	if face == nil {
		face = basicfont.Face7x13
	}
	title := "DIMA LIMBO VOL.1"
	runes := []rune(title)
	spacing := 28
	if face != basicfont.Face7x13 {
		spacing = int(18 * g.cfg.UIScale)
	}
	total := spacing * (len(runes) - 1)
	startX := (screenWidth - total) / 2
	baseY := (screenHeight * 28) / 100
	amp := 24.0 * g.cfg.UIScale
	n := float64(len(runes))
	for i, r := range runes {
		t := float64(i) / math.Max(1, n-1)
		angle := (t - 0.5) * math.Pi
		y := baseY + int(math.Sin(angle)*amp)
		x := startX + i*spacing
		phase := float64(g.frames)/30.0 + t*2*math.Pi
		cr := uint8(180 + 75*math.Sin(phase))
		cg := uint8(180 + 75*math.Sin(phase+2.094))
		cb := uint8(180 + 75*math.Sin(phase+4.188))
		fg := color.RGBA{cr, cg, cb, 255}
		shadow := color.RGBA{20, 20, 40, 200}
		drawShadowedText(g.offscreen, face, string(r), x, y, fg, shadow)
	}
	promptFace := basicfont.Face7x13
	drawShadowedText(g.offscreen, promptFace, "Press SPACE to start", (screenWidth-220)/2, baseY+80, color.RGBA{180, 255, 220, 255}, color.RGBA{40, 40, 40, 255})
}

func drawHUD(g *Game, _ float64) {
	face := basicfont.Face7x13
	drawShadowedText(g.offscreen, face, "Score:", 10, 24, color.White, color.RGBA{40, 40, 40, 255})
	drawShadowedText(g.offscreen, face, itoa(g.score), 70, 24, color.White, color.RGBA{40, 40, 40, 255})
}

func drawNameEntry(g *Game, _ float64) {
	face := basicfont.Face7x13
	drawShadowedText(g.offscreen, face, "Game Over! Enter your name:", 180, 220, color.White, color.RGBA{40, 40, 40, 255})
	drawShadowedText(g.offscreen, face, g.nameInput+"_", 220, 260, color.RGBA{0, 255, 128, 255}, color.RGBA{40, 40, 40, 255})
}

func drawLeaderboard(g *Game, _ float64) {
	face := basicfont.Face7x13
	drawShadowedText(g.offscreen, face, "Leaderboard", 300, 100, color.White, color.RGBA{40, 40, 40, 255})
	if len(g.leaders) == 0 {
		drawShadowedText(g.offscreen, face, "No scores yet.", 320, 140, color.RGBA{200, 200, 220, 255}, color.RGBA{40, 40, 40, 255})
	} else {
		for i, w := range g.leaders {
			line := itoa(i+1) + ". " + w.Name + " - " + itoa(w.Score)
			drawShadowedText(g.offscreen, face, line, 260, 140+(i*24), color.RGBA{220, 220, 220, 255}, color.RGBA{40, 40, 40, 255})
		}
	}
	drawShadowedText(g.offscreen, face, "R: reset leaderboard", 300, 420, color.RGBA{180, 180, 220, 255}, color.RGBA{40, 40, 40, 255})
	drawShadowedText(g.offscreen, face, "SPACE: title", 300, 444, color.RGBA{200, 200, 200, 255}, color.RGBA{40, 40, 40, 255})
}
