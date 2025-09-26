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
	// background by style
	switch g.cfg.BackgroundStyle {
	case "neon_space":
		g.offscreen.Fill(color.RGBA{R: 14, G: 16, B: 24, A: 255})
	case "retro_mario":
		// bright sky with gradient bands
		g.offscreen.Fill(color.RGBA{R: 110, G: 170, B: 255, A: 255})
		// simple ground strip
		ebitenutil.DrawRect(g.offscreen, 0, float64(oh)-60, float64(ow), 60, color.RGBA{R: 80, G: 180, B: 100, A: 255})
		// decorative hills
		for i := 0; i < 6; i++ {
			x := float64(i*ow/6 + 20)
			ebitenutil.DrawRect(g.offscreen, x, float64(oh)-90, 60, 30, color.RGBA{R: 70, G: 160, B: 90, A: 255})
		}
	default:
		g.offscreen.Fill(color.RGBA{R: 14, G: 16, B: 24, A: 255})
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
		// player
		// shadow
		ebitenutil.DrawRect(g.offscreen, g.player.x+4, g.player.y+4, g.player.w, g.player.h, color.RGBA{0, 0, 0, 120})
		// neon player with border
		ebitenutil.DrawRect(g.offscreen, g.player.x-1, g.player.y-1, g.player.w+2, g.player.h+2, color.RGBA{0, 60, 80, 200})
		ebitenutil.DrawRect(g.offscreen, g.player.x, g.player.y, g.player.w, g.player.h, color.RGBA{0, 255, 220, 255})
		// obstacles
		for _, o := range g.obstacles {
			ebitenutil.DrawRect(g.offscreen, o.x+3, o.y+3, o.w, o.h, color.RGBA{0, 0, 0, 120})
			// neon outline
			ebitenutil.DrawRect(g.offscreen, o.x-1, o.y-1, o.w+2, o.h+2, color.RGBA{60, 0, 40, 200})
			ebitenutil.DrawRect(g.offscreen, o.x, o.y, o.w, o.h, color.RGBA{255, 40, 140, 255})
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

// UI helpers for bold/outlined text (brutalist look)
func drawShadowedText(dst *ebiten.Image, face font.Face, s string, x, y int, fg, shadow color.Color) {
	text.Draw(dst, s, face, x+2, y+2, shadow)
	text.Draw(dst, s, face, x, y, fg)
}

func drawTitleUI(g *Game, dst *ebiten.Image) {
	face := g.titleFace
	if face == nil {
		face = basicfont.Face7x13
	}
	title := "DIMA LIMBO VOL.1"
	runes := []rune(title)
	// wider letter spacing
	spacing := int(28 * g.cfg.UIScale)
	if spacing < 22 {
		spacing = 22
	}
	total := spacing * (len(runes) - 1)
	startX := (screenWidth - total) / 2
	baseY := (screenHeight * 26) / 100
	// curve radius and depth
	radius := 260.0 * g.cfg.UIScale
	depth := 6
	for i, r := range runes {
		t := float64(i) / float64(len(runes)-1)
		angle := (t - 0.5) * 1.6 // stronger curve
		x := startX + i*spacing
		y := baseY + int(math.Sin(angle)*radius*0.1)
		// layered extrusion for 3D
		for dz := depth; dz >= 0; dz-- {
			off := float64(dz)
			col := color.RGBA{uint8(30 + dz*4), uint8(50 + dz*6), uint8(80 + dz*8), 255}
			drawShadowedText(dst, face, string(r), x+int(off), y+int(off), col, color.RGBA{0, 0, 0, 0})
		}
		phase := float64(g.frames)/24.0 + t*2*math.Pi
		fg := color.RGBA{uint8(200 + 40*math.Sin(phase)), uint8(220), 255, 255}
		drawShadowedText(dst, face, string(r), x, y, fg, color.RGBA{20, 20, 40, 200})
	}
	promptFace := basicfont.Face7x13
	drawShadowedText(dst, promptFace, "Press SPACE to start", (screenWidth-220)/2, baseY+80, color.RGBA{180, 255, 220, 255}, color.RGBA{40, 40, 40, 255})
}

func drawHUDUI(g *Game, dst *ebiten.Image) {
	face := g.uiFace
	if face == nil {
		face = basicfont.Face7x13
	}
	// responsive margins for small screens and measured spacing
	left := 10
	top := 28
	if g.cfg.UIScale < 1.2 {
		left = 6
		top = 22
	}
	label := "Score:"
	lb := text.BoundString(face, label)
	labelW := lb.Dx()
	if labelW < 0 {
		labelW = 0
	}
	drawShadowedText(dst, face, label, left, top, color.White, color.RGBA{40, 40, 40, 255})
	drawShadowedText(dst, face, itoa(g.score), left+labelW+8, top, color.White, color.RGBA{40, 40, 40, 255})
}

func drawNameEntryUI(g *Game, dst *ebiten.Image) {
	face := g.uiFace
	if face == nil {
		face = basicfont.Face7x13
	}
	baseX := 160
	if g.cfg.UIScale < 1.2 {
		baseX = 120
	}
	drawShadowedText(dst, face, "Game Over! Enter your name:", baseX, 220, color.White, color.RGBA{40, 40, 40, 255})
	drawShadowedText(dst, face, g.nameInput+"_", baseX+60, 264, color.RGBA{0, 255, 128, 255}, color.RGBA{40, 40, 40, 255})
	drawShadowedText(dst, basicfont.Face7x13, "Press ENTER/SPACE or TAP to submit", baseX, 300, color.RGBA{200, 220, 200, 255}, color.RGBA{40, 40, 40, 255})
}

func drawLeaderboardUI(g *Game, dst *ebiten.Image) {
	face := g.uiFace
	if face == nil {
		face = basicfont.Face7x13
	}
	titleX := 300
	if g.cfg.UIScale < 1.2 {
		titleX = 260
	}
	drawShadowedText(dst, face, "Leaderboard", titleX, 100, color.White, color.RGBA{40, 40, 40, 255})
	if len(g.leaders) == 0 {
		drawShadowedText(dst, face, "No scores yet.", titleX+20, 160, color.RGBA{200, 200, 220, 255}, color.RGBA{40, 40, 40, 255})
		drawShadowedText(dst, basicfont.Face7x13, "Tip: After a run, type your name and press ENTER/SPACE or TAP", 120, 200, color.RGBA{190, 210, 190, 255}, color.RGBA{40, 40, 40, 255})
	} else {
		for i, w := range g.leaders {
			line := itoa(i+1) + ". " + w.Name + " - " + itoa(w.Score)
			drawShadowedText(dst, face, line, titleX-80, 160+(i*28), color.RGBA{220, 220, 220, 255}, color.RGBA{40, 40, 40, 255})
		}
	}
	drawShadowedText(dst, face, "R: reset leaderboard", titleX-20, 440, color.RGBA{180, 180, 220, 255}, color.RGBA{40, 40, 40, 255})
	drawShadowedText(dst, face, "SPACE: title", titleX-20, 468, color.RGBA{200, 200, 200, 255}, color.RGBA{40, 40, 40, 255})
}
