package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

type Manager struct {
	ctx     *audio.Context
	volume  float64
	samples map[string][]byte
	players map[string]*audio.Player
}

func NewManager(sampleRate int, volume float64) *Manager {
	return &Manager{
		ctx:     audio.NewContext(sampleRate),
		volume:  volume,
		samples: make(map[string][]byte),
		players: make(map[string]*audio.Player),
	}
}

func (m *Manager) SetVolume(v float64) { m.volume = v }

// generateSineWAV returns a minimal PCM 16-bit mono WAV.
func generateSineWAV(sampleRate int, freq float64, dur time.Duration, vol float64) []byte {
	frames := int(float64(sampleRate) * dur.Seconds())
	pcm := make([]int16, frames)
	for i := 0; i < frames; i++ {
		v := math.Sin(2*math.Pi*freq*float64(i)/float64(sampleRate)) * vol
		pcm[i] = int16(v * 32767)
	}
	data := make([]byte, len(pcm)*2)
	for i, s := range pcm {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(s))
	}
	// WAV header
	buf := &bytes.Buffer{}
	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36+len(data)))
	buf.WriteString("WAVE")
	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))         // Subchunk1Size
	binary.Write(buf, binary.LittleEndian, uint16(1))          // AudioFormat PCM
	binary.Write(buf, binary.LittleEndian, uint16(1))          // NumChannels
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate)) // SampleRate
	byteRate := uint32(sampleRate * 1 * 16 / 8)
	binary.Write(buf, binary.LittleEndian, byteRate) // ByteRate
	blockAlign := uint16(1 * 16 / 8)
	binary.Write(buf, binary.LittleEndian, blockAlign) // BlockAlign
	binary.Write(buf, binary.LittleEndian, uint16(16)) // BitsPerSample
	// data chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func (m *Manager) ensurePlayer(key string, wavData []byte) (*audio.Player, error) {
	if p, ok := m.players[key]; ok {
		return p, nil
	}
	r, err := wav.DecodeWithSampleRate(m.ctx.SampleRate(), bytes.NewReader(wavData))
	if err != nil {
		return nil, err
	}
	p, err := m.ctx.NewPlayer(r)
	if err != nil {
		return nil, err
	}
	m.players[key] = p
	return p, nil
}

func (m *Manager) PlayStart()  { m.playTone("start", 420, 90*time.Millisecond) }
func (m *Manager) PlayHit()    { m.playTone("hit", 110, 120*time.Millisecond) }
func (m *Manager) PlaySubmit() { m.playTone("submit", 660, 80*time.Millisecond) }

func (m *Manager) playTone(key string, freq float64, dur time.Duration) {
	if m == nil || m.ctx == nil {
		return
	}
	w, ok := m.samples[key]
	if !ok {
		w = generateSineWAV(m.ctx.SampleRate(), freq, dur, m.volume)
		m.samples[key] = w
	}
	p, err := m.ensurePlayer(key, w)
	if err != nil {
		return
	}
	_ = p.Rewind()
	p.SetVolume(m.volume)
	p.Play()
}
