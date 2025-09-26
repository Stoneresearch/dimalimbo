package game

import (
	"bytes"
	"encoding/json"
	"image/color"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
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
	bgImage   *ebiten.Image
	shader    *ebiten.Shader
	shaderOn  bool
	shaderInt float32
	audio     *aud.Manager
	// parallax
	starsFar  []rectangle
	starsNear []rectangle
	// particles
	particles []particle
	// ambience
	shooters []shootingStar
	// satellites
	satellites []satellite
	// difficulty
	speed      float64
	spawnEvery int
	// settings
	cfg settings.Settings
	// fonts
	titleFace font.Face
	uiFace    font.Face
}

type particle struct {
	x    float64
	y    float64
	vx   float64
	vy   float64
	life int
}

type shootingStar struct {
	x    float64
	y    float64
	vx   float64
	vy   float64
	life int
}

type satellite struct {
	x     float64
	y     float64
	spin  float64
	vel   float64
	size  float64
	glowA uint8
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
	if g.audio != nil {
		g.audio.SetStyle(cfg.MusicStyle)
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
	// medium UI face for leaderboard and HUD (use Regular for legibility)
	if fr, err := opentype.Parse(goregular.TTF); err == nil {
		uiSize := 22.0 * cfg.UIScale
		if uiFace, uerr := opentype.NewFace(fr, &opentype.FaceOptions{Size: uiSize, DPI: 72, Hinting: font.HintingFull}); uerr == nil {
			g.uiFace = uiFace
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
		// auto-fetch background if endpoint provided
		if g.cfg.BackgroundURL == "" && g.cfg.BackgroundEndpoint != "" {
			go func(ep string) {
				req := map[string]any{"prompt": "colorful adventurous synthwave space, cinematic, detailed", "width": 1600, "height": 900}
				b, _ := json.Marshal(req)
				resp, err := http.Post(ep, "application/json", bytes.NewReader(b))
				if err == nil && resp.StatusCode < 300 {
					var r struct {
						URL string `json:"url"`
					}
					_ = json.NewDecoder(resp.Body).Decode(&r)
					resp.Body.Close()
					if r.URL != "" {
						g.cfg.BackgroundURL = r.URL
					}
				}
			}(g.cfg.BackgroundEndpoint)
		}
	}

	// fullscreen toggle
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	// mute toggle
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		if g.audio != nil {
			g.audio.ToggleMute()
		}
	}

	// occasional shooting stars
	if rand.Intn(120) == 0 {
		g.shooters = append(g.shooters, shootingStar{
			x:    float64(screenWidth + 20),
			y:    float64(40 + rand.Intn(160)),
			vx:   -3.2 - rand.Float64()*2.0,
			vy:   0.7 + rand.Float64()*0.6,
			life: 160,
		})
	}
	aliveS := g.shooters[:0]
	for _, s := range g.shooters {
		s.x += s.vx
		s.y += s.vy
		s.life--
		if s.life > 0 && s.x > -40 && s.y < float64(screenHeight-40) {
			aliveS = append(aliveS, s)
		}
	}
	g.shooters = aliveS

	// spawn satellites (parallax foreground)
	if rand.Intn(180) == 0 {
		g.satellites = append(g.satellites, satellite{
			x:     float64(screenWidth + 40),
			y:     float64(40 + rand.Intn(screenHeight/2)),
			spin:  rand.Float64() * math.Pi,
			vel:   0.9 + rand.Float64()*0.6,
			size:  10 + rand.Float64()*10,
			glowA: 160,
		})
	}
	aliveSat := g.satellites[:0]
	for _, s := range g.satellites {
		s.x -= s.vel
		s.spin += 0.02
		if s.x > -40 {
			aliveSat = append(aliveSat, s)
		}
	}
	g.satellites = aliveSat

	switch g.state {
	case stateTitle:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(ebiten.TouchIDs()) > 0 || inpututil.IsGamepadButtonJustPressed(0, ebiten.GamepadButton0) {
			g.resetPlay()
			g.state = statePlaying
			if g.audio != nil && g.cfg.MusicEnabled {
				g.audio.PlayStart()
				g.audio.PlayMusic()
			}
		}
	case statePlaying:
		// Player movement
		// Touch/mouse drag toward target (mobile friendly)
		if ids := ebiten.TouchIDs(); len(ids) > 0 {
			x, y := ebiten.TouchPosition(ids[0])
			tx := float64(x) - (g.player.x + g.player.w*0.5)
			ty := float64(y) - (g.player.y + g.player.h*0.5)
			d := math.Hypot(tx, ty)
			if d > 1 {
				g.player.x += g.playerVel * (tx / d)
				g.player.y += g.playerVel * (ty / d)
			}
		} else if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			cx, cy := ebiten.CursorPosition()
			tx := float64(cx) - (g.player.x + g.player.w*0.5)
			ty := float64(cy) - (g.player.y + g.player.h*0.5)
			d := math.Hypot(tx, ty)
			if d > 1 {
				g.player.x += g.playerVel * (tx / d)
				g.player.y += g.playerVel * (ty / d)
			}
		}
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

		// particles update (neon trail)
		aliveP := g.particles[:0]
		for _, p := range g.particles {
			p.x += p.vx
			p.y += p.vy
			p.vx *= 0.96
			p.vy *= 0.96
			p.life--
			if p.life > 0 {
				aliveP = append(aliveP, p)
			}
		}
		g.particles = aliveP
		// spawn a few new particles at the player's center
		for i := 0; i < 2; i++ {
			px := g.player.x + g.player.w*0.5
			py := g.player.y + g.player.h*0.5
			angle := rand.Float64() * 2 * math.Pi
			speed := 0.8 + rand.Float64()*0.6
			g.particles = append(g.particles, particle{
				x:    px,
				y:    py,
				vx:   math.Cos(angle) * speed * -0.6,
				vy:   math.Sin(angle) * speed * -0.6,
				life: 28 + rand.Intn(16),
			})
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
		// submit on Enter/Space or tap/click release to avoid accidental holds
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(ebiten.TouchIDs()) == 0 {
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
	// dynamic resolution offscreen
	scale := g.cfg.RenderScale
	if scale <= 0 || scale > 1.0 {
		scale = 1.0
	}
	ow := int(float64(screenWidth) * scale)
	oh := int(float64(screenHeight) * scale)
	if ow < 320 {
		ow = 320
	}
	if oh < 240 {
		oh = 240
	}
	if g.offscreen == nil || g.offscreen.Bounds().Dx() != ow || g.offscreen.Bounds().Dy() != oh {
		g.offscreen = ebiten.NewImage(ow, oh)
	}
	// LIMBO-inspired dark atmosphere
	switch g.cfg.BackgroundStyle {
	case "limbo_forest":
		// Dark forest atmosphere with layered silhouettes
		g.offscreen.Fill(color.RGBA{R: 8, G: 8, B: 12, A: 255})
		// Distant hills silhouette
		for i := 0; i < 5; i++ {
			alpha := uint8(15 + i*8)
			height := float64(oh) * (0.6 + float64(i)*0.08)
			ebitenutil.DrawRect(g.offscreen, 0, height, float64(ow), float64(oh)-height, color.RGBA{R: alpha, G: alpha, B: alpha + 5, A: 255})
		}
	case "limbo_industrial":
		// Industrial wasteland
		g.offscreen.Fill(color.RGBA{R: 12, G: 10, B: 8, A: 255})
		// Factory silhouettes
		for i := 0; i < 3; i++ {
			x := float64(i*ow/3 + rand.Intn(50))
			w := 40.0 + rand.Float64()*60
			h := 80.0 + rand.Float64()*120
			ebitenutil.DrawRect(g.offscreen, x, float64(oh)-h, w, h, color.RGBA{R: 20, G: 18, B: 16, A: 255})
		}
	default:
		// Classic LIMBO - pure darkness with subtle gradient
		g.offscreen.Fill(color.RGBA{R: 8, G: 8, B: 12, A: 255})
		// Subtle fog gradient from bottom
		for y := 0; y < oh/3; y++ {
			alpha := uint8(float64(y) / float64(oh/3) * 20)
			ebitenutil.DrawRect(g.offscreen, 0, float64(oh-y), float64(ow), 1, color.RGBA{R: 15 + alpha/2, G: 15 + alpha/2, B: 18 + alpha, A: 255})
		}
	}

	// External background image (AI-generated via URL) if provided
	if g.cfg.BackgroundURL != "" {
		if g.bgImage == nil {
			if resp, err := http.Get(g.cfg.BackgroundURL); err == nil {
				if data, err := io.ReadAll(resp.Body); err == nil {
					img, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(data))
					if img != nil {
						g.bgImage = img
					}
				}
				_ = resp.Body.Close()
			}
		}
		if g.bgImage != nil {
			opBG := &ebiten.DrawImageOptions{}
			sx := float64(ow) / float64(g.bgImage.Bounds().Dx())
			sy := float64(oh) / float64(g.bgImage.Bounds().Dy())
			opBG.GeoM.Scale(sx, sy)
			g.offscreen.DrawImage(g.bgImage, opBG)
		}
	}

	// camera sway
	swayX := math.Sin(float64(g.frames)*0.01) * 2.0
	swayY := math.Cos(float64(g.frames)*0.013) * 1.0

	// parallax background
	stepFar := 1
	stepNear := 1
	if g.cfg.LowPower {
		stepFar, stepNear = 2, 2
	}
	for i := 0; i < len(g.starsFar); i += stepFar {
		s := &g.starsFar[i]
		s.x -= 0.3
		if s.x < 0 {
			s.x = float64(ow)
			s.y = float64(rand.Intn(oh))
		}
		// twinkle
		tw := uint8(180 + 70*math.Sin(float64(g.frames+i)*0.05))
		ebitenutil.DrawRect(g.offscreen, s.x+swayX*0.3, s.y+swayY*0.2, s.w, s.h, color.RGBA{tw, tw, 220, 255})
	}
	for i := 0; i < len(g.starsNear); i += stepNear {
		s := &g.starsNear[i]
		s.x -= 0.8
		if s.x < 0 {
			s.x = float64(ow)
			s.y = float64(rand.Intn(oh))
		}
		tw := uint8(200 + 55*math.Sin(float64(g.frames+i)*0.07))
		ebitenutil.DrawRect(g.offscreen, s.x+swayX*0.6, s.y+swayY*0.4, s.w, s.h, color.RGBA{tw, 220, 255, 255})
	}
	// satellites with glow and rotation
	for _, sat := range g.satellites {
		// glow
		ebitenutil.DrawRect(g.offscreen, sat.x-3, sat.y-3, sat.size+6, sat.size+6, color.RGBA{0, 80, 120, sat.glowA})
		// body
		ebitenutil.DrawRect(g.offscreen, sat.x, sat.y, sat.size, sat.size, color.RGBA{200, 230, 255, 255})
		// solar panels (simple lines)
		lx := sat.x - 8
		rx := sat.x + sat.size + 8
		cy := sat.y + sat.size/2
		ebitenutil.DrawLine(g.offscreen, lx, cy-2, sat.x, cy-2, color.RGBA{120, 200, 255, 255})
		ebitenutil.DrawLine(g.offscreen, sat.x+sat.size, cy-2, rx, cy-2, color.RGBA{120, 200, 255, 255})
	}
	// particle rendering (additive-ish)
	for _, p := range g.particles {
		a := uint8(40 + p.life*5)
		if a > 200 {
			a = 200
		}
		c := color.RGBA{R: 0, G: uint8(180 + rand.Intn(60)), B: 255, A: a}
		ebitenutil.DrawRect(g.offscreen, p.x-1.5, p.y-1.5, 3, 3, c)
	}
	// shooting stars
	for _, s := range g.shooters {
		ebitenutil.DrawLine(g.offscreen, s.x, s.y, s.x-12, s.y-6, color.RGBA{200, 240, 255, 200})
		ebitenutil.DrawRect(g.offscreen, s.x, s.y, 2, 2, color.RGBA{255, 255, 255, 255})
	}
	// 3D-ish ground grid (optional)
	horizonY := float64(oh) * 0.65
	wobble := math.Sin(float64(g.frames) * 0.02)
	if g.cfg.ShowGrid {
		for i := 0; i < 12; i++ {
			t := float64(i) / 11.0
			x := t * float64(ow)
			ebitenutil.DrawLine(g.offscreen, x+swayX, horizonY+swayY, x-80*(t-0.5+wobble*0.02)+swayX, float64(oh), color.RGBA{36, 36, 70, 180})
		}
		for r := 0; r < 10; r++ {
			y := horizonY + float64(r*r)*6 + wobble*2 + swayY
			ebitenutil.DrawLine(g.offscreen, 0, y, float64(ow), y, color.RGBA{36, 36, 70, 160})
		}
	}

	switch g.state {
	case statePlaying:
		// LIMBO-style player - pure black silhouette
		// Subtle glow behind player for visibility
		ebitenutil.DrawRect(g.offscreen, g.player.x-2, g.player.y-2, g.player.w+4, g.player.h+4, color.RGBA{40, 40, 50, 60})
		// Main player silhouette - completely black
		ebitenutil.DrawRect(g.offscreen, g.player.x, g.player.y, g.player.w, g.player.h, color.RGBA{0, 0, 0, 255})

		// LIMBO-style obstacles - dark threatening shapes
		for _, o := range g.obstacles {
			// Subtle danger glow
			ebitenutil.DrawRect(g.offscreen, o.x-1, o.y-1, o.w+2, o.h+2, color.RGBA{60, 20, 20, 80})
			// Main obstacle - very dark gray with slight red tint (danger)
			ebitenutil.DrawRect(g.offscreen, o.x, o.y, o.w, o.h, color.RGBA{25, 15, 15, 255})
		}

		// Atmospheric particles - minimal and dark
		for _, p := range g.particles {
			if p.life > 0 {
				alpha := uint8(p.life * 3) // Much more subtle
				size := 1.0
				ebitenutil.DrawRect(g.offscreen, p.x-size/2, p.y-size/2, size, size, color.RGBA{80, 80, 90, alpha})
			}
		}
	case stateTitle, stateNameEntry, stateLeaderboard:
		// defer UI drawing to after post-processing
	}

	// post-process and upscale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(screenWidth)/float64(ow), float64(screenHeight)/float64(oh))
	if g.shader != nil && g.shaderOn && !g.cfg.LowPower {
		opts := &ebiten.DrawRectShaderOptions{}
		opts.Images[0] = g.offscreen
		opts.Uniforms = map[string]interface{}{
			"time":       float32(g.frames) / 60.0,
			"intensity":  g.shaderInt,
			"resolution": []float32{float32(ow), float32(oh)},
		}
		screen.DrawRectShader(screenWidth, screenHeight, g.shader, opts)
	} else {
		screen.DrawImage(g.offscreen, op)
	}

	// UI pass AFTER post-processing for crisp text and spacing
	switch g.state {
	case stateTitle:
		drawTitleUI(g, screen)
	case statePlaying:
		drawHUDUI(g, screen)
	case stateNameEntry:
		drawNameEntryUI(g, screen)
	case stateLeaderboard:
		drawLeaderboardUI(g, screen)
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

// UI helpers - removed unused drawShadowedText function

func drawTitleUI(g *Game, dst *ebiten.Image) {
	face := g.titleFace
	if face == nil {
		face = basicfont.Face7x13
	}

	title := "DIMBO"

	// Simple, clean LIMBO-style title - properly centered
	titleWidth := len(title) * 8 * int(g.cfg.UIScale) // Rough estimate
	centerX := screenWidth / 2
	titleX := centerX - titleWidth/2
	titleY := screenHeight / 3

	// LIMBO-style title - subtle, not flashy
	// Soft glow behind text
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			if dx != 0 || dy != 0 {
				text.Draw(dst, title, face, titleX+dx, titleY+dy, color.RGBA{40, 40, 50, 30})
			}
		}
	}

	// Main title - clean white text
	text.Draw(dst, title, face, titleX, titleY, color.RGBA{200, 200, 200, 255})

	// Subtitle - inspired by classic LIMBO
	subtitle := "a dark journey"
	subtitleWidth := len(subtitle) * 6
	subtitleX := centerX - subtitleWidth/2
	text.Draw(dst, subtitle, basicfont.Face7x13, subtitleX, titleY+60, color.RGBA{120, 120, 120, 200})

	// Simple prompt - properly centered
	prompt := "Press SPACE to begin"
	promptWidth := len(prompt) * 6
	promptX := centerX - promptWidth/2
	text.Draw(dst, prompt, basicfont.Face7x13, promptX, titleY+120, color.RGBA{160, 160, 160, 180})
}

func drawHUDUI(g *Game, dst *ebiten.Image) {
	face := g.uiFace
	if face == nil {
		face = basicfont.Face7x13
	}

	// LIMBO-style minimal HUD - properly positioned with responsive margins
	margin := int(20 * g.cfg.UIScale)
	if margin < 10 {
		margin = 10
	}

	top := margin + 10

	// Simple score display - clean and readable
	scoreText := "Score: " + itoa(g.score)
	text.Draw(dst, scoreText, face, margin, top, color.RGBA{180, 180, 180, 255})

	// Lives indicator (if we add lives later)
	// Could show as subtle dots in the corner
}

func drawNameEntryUI(g *Game, dst *ebiten.Image) {
	face := g.uiFace
	if face == nil {
		face = basicfont.Face7x13
	}

	centerX := screenWidth / 2
	centerY := screenHeight / 2

	// LIMBO-style game over screen - centered and atmospheric
	gameOverText := "The journey ends..."
	gameOverWidth := len(gameOverText) * 6
	text.Draw(dst, gameOverText, face, centerX-gameOverWidth/2, centerY-60, color.RGBA{150, 150, 150, 255})

	// Score display
	scoreText := "Distance traveled: " + itoa(g.score)
	scoreWidth := len(scoreText) * 6
	text.Draw(dst, scoreText, basicfont.Face7x13, centerX-scoreWidth/2, centerY-20, color.RGBA{120, 120, 120, 200})

	// Name input prompt
	namePrompt := "Your name:"
	namePromptWidth := len(namePrompt) * 6
	text.Draw(dst, namePrompt, basicfont.Face7x13, centerX-namePromptWidth/2, centerY+20, color.RGBA{140, 140, 140, 255})

	// Name input field - properly centered
	nameDisplay := g.nameInput + "_"
	nameWidth := len(nameDisplay) * 6
	text.Draw(dst, nameDisplay, face, centerX-nameWidth/2, centerY+50, color.RGBA{180, 180, 180, 255})

	// Instructions
	instructions := "Press ENTER to continue"
	instrWidth := len(instructions) * 5
	text.Draw(dst, instructions, basicfont.Face7x13, centerX-instrWidth/2, centerY+100, color.RGBA{100, 100, 100, 200})
}

func drawLeaderboardUI(g *Game, dst *ebiten.Image) {
	face := g.uiFace
	if face == nil {
		face = basicfont.Face7x13
	}

	centerX := screenWidth / 2
	startY := 120

	// LIMBO-style leaderboard - properly centered
	title := "Those who traveled far"
	titleWidth := len(title) * 8
	text.Draw(dst, title, face, centerX-titleWidth/2, startY, color.RGBA{160, 160, 160, 255})

	if len(g.leaders) == 0 {
		emptyText := "None have journeyed yet..."
		emptyWidth := len(emptyText) * 6
		text.Draw(dst, emptyText, basicfont.Face7x13, centerX-emptyWidth/2, startY+80, color.RGBA{100, 100, 100, 200})
	} else {
		// List entries - properly centered and spaced
		for i, w := range g.leaders {
			if i >= 10 {
				break
			}

			line := itoa(i+1) + ". " + w.Name + " - " + itoa(w.Score)
			lineWidth := len(line) * 6
			y := startY + 60 + (i * 25)

			text.Draw(dst, line, basicfont.Face7x13, centerX-lineWidth/2, y, color.RGBA{140, 140, 140, 255})
		}
	}

	// Controls - properly positioned at bottom
	controls := "R: reset    SPACE: return"
	controlsWidth := len(controls) * 5
	text.Draw(dst, controls, basicfont.Face7x13, centerX-controlsWidth/2, screenHeight-60, color.RGBA{100, 100, 100, 180})
}
