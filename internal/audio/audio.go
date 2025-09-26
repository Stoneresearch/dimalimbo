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
	music   *audio.Player
	muted   bool
	style   string
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
func (m *Manager) ToggleMute() {
	m.muted = !m.muted
	if m.music != nil {
		if m.muted {
			m.music.Pause()
		} else {
			m.music.Play()
		}
	}
}
func (m *Manager) SetStyle(style string) { m.style = style }

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

// ==== Chiptune-style background music (loop) ====

// squarePCM generates raw PCM16 mono bytes for a square wave with optional ADSR envelope.
func squarePCM(sampleRate int, freq float64, dur time.Duration, vol float64, attackMs, decayMs, sustainLevel, releaseMs float64) []int16 {
	frames := int(float64(sampleRate) * dur.Seconds())
	pcm := make([]int16, frames)
	a := int(attackMs * float64(sampleRate) / 1000)
	d := int(decayMs * float64(sampleRate) / 1000)
	r := int(releaseMs * float64(sampleRate) / 1000)
	for i := 0; i < frames; i++ {
		// Basic square (50% duty)
		phase := math.Sin(2 * math.Pi * freq * float64(i) / float64(sampleRate))
		s := 1.0
		if phase < 0 {
			s = -1.0
		}
		// Envelope
		env := 1.0
		if i < a && a > 0 {
			env = float64(i) / float64(a)
		} else if i < a+d && d > 0 {
			t := float64(i-a) / float64(d)
			env = 1.0 + t*(sustainLevel-1.0)
		} else if i > frames-r && r > 0 {
			t := float64(frames-i) / float64(r)
			env = sustainLevel * t
		} else {
			env = sustainLevel
		}
		v := s * vol * env
		pcm[i] = int16(v * 32767)
	}
	return pcm
}

// mixTracks mixes multiple PCM16 mono tracks, preventing clipping.
func mixTracks(tracks ...[]int16) []byte {
	maxLen := 0
	for _, t := range tracks {
		if len(t) > maxLen {
			maxLen = len(t)
		}
	}
	out := make([]byte, maxLen*2)
	for i := 0; i < maxLen; i++ {
		sum := 0
		for _, t := range tracks {
			if i < len(t) {
				sum += int(t[i])
			}
		}
		// soft clip
		if sum > 32767 {
			sum = 32767
		}
		if sum < -32768 {
			sum = -32768
		}
		binary.LittleEndian.PutUint16(out[i*2:], uint16(int16(sum)))
	}
	return out
}

// composeLoop builds a short melody + bass loop reminiscent of retro consoles.
func composeLoop(sampleRate int) (pcm []byte) {
	tempo := 132.0
	beats := 16 // 4 bars of 4/4
	beatDur := time.Duration(float64(time.Second) * 60.0 / tempo)
	totalDur := beatDur * time.Duration(beats)
	totalSamples := int(float64(sampleRate) * totalDur.Seconds())

	// Lead notes (Hz) in a simple arpeggio pattern
	scale := []float64{261.63, 329.63, 392.00, 523.25} // C E G C
	lead := make([]int16, totalSamples)
	idx := 0
	halfBeatSamples := int(float64(sampleRate) * (beatDur.Seconds() / 2))
	for b := 0; b < beats; b++ {
		f := scale[b%len(scale)]
		note := squarePCM(sampleRate, f, beatDur/2, 0.18, 4, 30, 0.6, 40)
		for i := 0; i < len(note) && idx < len(lead); i++ {
			lead[idx] = note[i]
			idx++
		}
		// small rest
		idx += halfBeatSamples
		if idx > len(lead) {
			idx = len(lead)
		}
	}

	// Bass on downbeats
	bass := make([]int16, len(lead))
	idx = 0
	beatSamples := int(float64(sampleRate) * beatDur.Seconds())
	for b := 0; b < beats; b++ {
		if b%2 == 0 {
			f := 130.81 // C3
			note := squarePCM(sampleRate, f, beatDur, 0.15, 2, 40, 0.5, 60)
			for i := 0; i < len(note) && idx < len(bass); i++ {
				bass[idx] = note[i]
				idx++
			}
		}
		idx += beatSamples
		if idx > len(bass) {
			idx = len(bass)
		}
	}

	pcm = mixTracks(lead, bass)
	return
}

// ==== Synthwave generator ====
// composeSynthwave creates a richer loop with pads, bass, and a noise snare.
func composeSynthwave(sampleRate int) (pcm []byte) {
	tempo := 96.0
	beats := 16
	beatDur := time.Duration(float64(time.Second) * 60.0 / tempo)
	totalDur := beatDur * time.Duration(beats)
	totalSamples := int(float64(sampleRate) * totalDur.Seconds())

	// Pad (detuned saw approximated by averaging two squares)
	pad := make([]int16, totalSamples)
	for i := 0; i < totalSamples; i++ {
		t := float64(i) / float64(sampleRate)
		f1 := 220.0
		f2 := 222.0
		v := !math.Signbit(math.Sin(2 * math.Pi * f1 * t))
		w := !math.Signbit(math.Sin(2 * math.Pi * f2 * t))
		s := 0.0
		if v {
			s += 1
		} else {
			s -= 1
		}
		if w {
			s += 1
		} else {
			s -= 1
		}
		pad[i] = int16(s * 0.09 * 32767)
	}

	// Bass 4-on-the-floor
	bass := make([]int16, totalSamples)
	beatSamples := int(float64(sampleRate) * beatDur.Seconds())
	idx := 0
	for b := 0; b < beats; b++ {
		f := 110.0 // A2
		note := squarePCM(sampleRate, f, beatDur, 0.20, 2, 40, 0.6, 40)
		for i := 0; i < len(note) && idx < len(bass); i++ {
			bass[idx] = note[i]
			idx++
		}
	}

	// Lead arpeggio
	lead := make([]int16, totalSamples)
	scale := []float64{440.0, 554.37, 659.25, 880.0}
	idx = 0
	half := beatSamples / 2
	for b := 0; b < beats; b++ {
		f := scale[b%len(scale)]
		note := squarePCM(sampleRate, f, beatDur/2, 0.17, 3, 30, 0.6, 40)
		for i := 0; i < len(note) && idx < len(lead); i++ {
			lead[idx] = note[i]
			idx++
		}
		idx += half
		if idx > len(lead) {
			idx = len(lead)
		}
	}

	// Noise snare on 2 and 4 & hi-hat 8th notes
	snare := make([]int16, totalSamples)
	hihat := make([]int16, totalSamples)
	rng := uint32(1)
	noise := func() float64 { // simple LCG noise
		rng = rng*1664525 + 1013904223
		return float64(int32(rng>>16)) / 32768.0
	}
	for b := 0; b < beats; b++ {
		if b%4 == 1 || b%4 == 3 { // 2 and 4
			start := b * beatSamples
			length := beatSamples / 3
			for i := 0; i < length && start+i < totalSamples; i++ {
				env := math.Exp(-3.5 * float64(i) / float64(length))
				snare[start+i] = int16(noise() * 0.12 * env * 32767)
			}
		}
		// hi-hat on 8ths
		for h := 0; h < 2; h++ {
			start := b*beatSamples + h*(beatSamples/2)
			length := beatSamples / 8
			for i := 0; i < length && start+i < totalSamples; i++ {
				env := math.Exp(-6.0 * float64(i) / float64(length))
				hihat[start+i] = int16(noise() * 0.06 * env * 32767)
			}
		}
	}

	// Kick on quarter notes using a low sine drop
	kick := make([]int16, totalSamples)
	for b := 0; b < beats; b++ {
		start := b * beatSamples
		length := beatSamples / 6
		for i := 0; i < length && start+i < totalSamples; i++ {
			t := float64(i) / float64(sampleRate)
			f := 90.0 - 60.0*(float64(i)/float64(length))
			s := math.Sin(2 * math.Pi * f * t)
			env := math.Exp(-8.0 * float64(i) / float64(length))
			kick[start+i] = int16(s * 0.22 * env * 32767)
		}
	}

	pcm = mixTracks(pad, bass, lead, snare, hihat, kick)
	return
}

// PlayMusic starts (or restarts) a looping background melody.
func (m *Manager) PlayMusic() {
	if m == nil || m.ctx == nil {
		return
	}
	if m.muted {
		return
	}
	if m.music != nil {
		if m.music.IsPlaying() {
			return
		}
		_ = m.music.Rewind()
		m.music.SetVolume(m.volume * 0.8)
		m.music.Play()
		return
	}
	var pcm []byte
	if m.style == "synthwave" {
		pcm = composeSynthwave(m.ctx.SampleRate())
	} else {
		pcm = composeLoop(m.ctx.SampleRate())
	}
	loop := audio.NewInfiniteLoop(bytes.NewReader(pcm), int64(len(pcm)))
	p, err := m.ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	m.music = p
	m.music.SetVolume(m.volume * 0.4)
	m.music.Play()
}

func (m *Manager) StopMusic() {
	if m == nil || m.music == nil {
		return
	}
	m.music.Pause()
}
