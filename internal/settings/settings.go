package settings

import (
	"encoding/json"
	"os"
)

type Settings struct {
	MasterVolume    float64 `json:"masterVolume"`
	ShaderIntensity float32 `json:"shaderIntensity"`
	Palette         int     `json:"palette"`
	// Window/Perf
	Fullscreen    bool    `json:"fullscreen"`
	WindowWidth   int     `json:"windowWidth"`
	WindowHeight  int     `json:"windowHeight"`
	UIScale       float64 `json:"uiScale"`
	TargetFPS     int     `json:"targetFPS"`
	PostFXEnabled bool    `json:"postFXEnabled"`
	// Gameplay/Difficulty
	BaseSpeed           float64 `json:"baseSpeed"`
	SpawnEveryStart     int     `json:"spawnEveryStart"`
	SpawnEveryMin       int     `json:"spawnEveryMin"`
	SpeedAccel          float64 `json:"speedAccel"`
	AccelIntervalFrames int     `json:"accelIntervalFrames"`
	// Input
	EnableGamepad   bool    `json:"enableGamepad"`
	GamepadDeadzone float64 `json:"gamepadDeadzone"`
	InvertY         bool    `json:"invertY"`
	// Leaderboard/Storage
	TopN            int    `json:"topN"`
	CacheTTLSeconds int    `json:"cacheTTLSeconds"`
	DBPath          string `json:"dbPath"`
	// Performance
	RenderScale float64 `json:"renderScale"`
	LowPower    bool    `json:"lowPower"`
}

func Default() Settings {
	return Settings{
		MasterVolume:        0.25,
		ShaderIntensity:     0.7,
		Palette:             0,
		Fullscreen:          false,
		WindowWidth:         1280,
		WindowHeight:        960,
		UIScale:             1.8,
		TargetFPS:           60,
		PostFXEnabled:       true,
		BaseSpeed:           4.0,
		SpawnEveryStart:     60,
		SpawnEveryMin:       24,
		SpeedAccel:          0.4,
		AccelIntervalFrames: 300,
		EnableGamepad:       true,
		GamepadDeadzone:     0.2,
		InvertY:             false,
		TopN:                10,
		CacheTTLSeconds:     30,
		DBPath:              "dimalimbo.db",
		RenderScale:         0.9,
		LowPower:            false,
	}
}

func Load(path string) Settings {
	b, err := os.ReadFile(path)
	if err != nil {
		return Default()
	}
	var s Settings
	if json.Unmarshal(b, &s) != nil {
		return Default()
	}
	return s
}

func Save(path string, s Settings) {
	_ = os.WriteFile(path, must(json.MarshalIndent(s, "", "  ")), 0o644)
}

func must(b []byte, _ error) []byte { return b }
