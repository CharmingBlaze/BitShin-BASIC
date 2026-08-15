// Package audio plays clips through Oto v3 (github.com/ebitengine/oto/v3,
// the current module for hajimehoshi/oto). Decode is pure Go (.wav / .ogg).
package audio

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"

	oto "github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	SampleRate = 44100
	Channels   = 2
	FrameBytes = 4 // stereo s16le
)

var (
	ctxOnce sync.Once
	otoCtx  *oto.Context
	ctxErr  error
)

func ensure() (*oto.Context, error) {
	ctxOnce.Do(func() {
		var ready chan struct{}
		otoCtx, ready, ctxErr = oto.NewContext(&oto.NewContextOptions{
			SampleRate:   SampleRate,
			ChannelCount: Channels,
			Format:       oto.FormatSignedInt16LE,
		})
		if ctxErr != nil {
			return
		}
		<-ready
	})
	return otoCtx, ctxErr
}

// Clip is decoded stereo PCM at SampleRate.
type Clip struct {
	PCM []byte
}

// Load reads a .wav or .ogg file.
func Load(path string) (*Clip, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("LoadSound: %w", err)
	}
	lower := strings.ToLower(path)
	var stream io.Reader
	switch {
	case strings.HasSuffix(lower, ".ogg"):
		s, err := vorbis.DecodeWithSampleRate(SampleRate, bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("LoadSound: %w", err)
		}
		stream = s
	default:
		s, err := wav.DecodeWithSampleRate(SampleRate, bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("LoadSound: need a .wav or .ogg file (%w)", err)
		}
		stream = s
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("LoadSound: %w", err)
	}
	return &Clip{PCM: pcm}, nil
}

// Voice is one playing instance.
type Voice struct {
	p     *oto.Player
	vol   float64
	pitch float64
}

func (c *Clip) play(loop bool, vol, pitch, pan float64) (*Voice, error) {
	ctx, err := ensure()
	if err != nil {
		return nil, err
	}
	if pitch <= 0 {
		pitch = 1
	}
	if vol < 0 {
		vol = 0
	}
	pcm := c.PCM
	if pan != 0 || vol != 1 {
		pcm = mixPan(pcm, vol, pan)
		vol = 1
	}
	src := &rateReader{pcm: pcm, rate: pitch, loop: loop}
	p := ctx.NewPlayer(src)
	p.SetVolume(vol)
	p.Play()
	return &Voice{p: p, vol: vol, pitch: pitch}, nil
}

func (c *Clip) Play(loop bool) (*Voice, error) {
	return c.play(loop, 1, 1, 0)
}

func (c *Clip) PlayAt(vol, pitch, pan float64, loop bool) (*Voice, error) {
	return c.play(loop, vol, pitch, pan)
}

// GenerateThunder builds a short crack + rumble so weather never Ends on a missing .ogg.
func GenerateThunder(seconds float64) *Clip {
	if seconds < 0.4 {
		seconds = 0.4
	}
	if seconds > 4 {
		seconds = 4
	}
	n := int(seconds * float64(SampleRate))
	pcm := make([]byte, n*FrameBytes)
	var brown float64
	for i := 0; i < n; i++ {
		t := float64(i) / float64(SampleRate)
		white := randN(i)*2 - 1
		brown = brown*0.993 + white*0.07
		if brown > 1 {
			brown = 1
		}
		if brown < -1 {
			brown = -1
		}
		env := math.Exp(-t * 2.4)
		if t < 0.018 {
			env += (1 - t/0.018) * 0.85
		}
		rumble := math.Sin(t*55*2*math.Pi)*0.22 + math.Sin(t*28*2*math.Pi)*0.18
		s := (brown*0.72 + rumble + white*0.08) * env
		v := int16(clamp16(s * 22000))
		o := i * FrameBytes
		pcm[o] = byte(v)
		pcm[o+1] = byte(v >> 8)
		pcm[o+2] = byte(v)
		pcm[o+3] = byte(v >> 8)
	}
	return &Clip{PCM: pcm}
}

func randN(i int) float64 {
	x := uint32(i)*1664525 + 1013904223
	return float64(x&0x7fffffff) / float64(0x7fffffff)
}

func (v *Voice) Play() {
	if v != nil && v.p != nil {
		v.p.Play()
	}
}

func (v *Voice) Pause() {
	if v != nil && v.p != nil {
		v.p.Pause()
	}
}

func (v *Voice) SetVolume(vol float64) {
	if v == nil || v.p == nil {
		return
	}
	if vol < 0 {
		vol = 0
	}
	if vol > 1 {
		vol = 1
	}
	v.vol = vol
	v.p.SetVolume(vol)
}

func (v *Voice) Close() {
	if v != nil && v.p != nil {
		_ = v.p.Close()
		v.p = nil
	}
}

type rateReader struct {
	pcm  []byte
	pos  float64
	rate float64
	loop bool
}

func (r *rateReader) Read(p []byte) (int, error) {
	if len(r.pcm) < FrameBytes {
		return 0, io.EOF
	}
	if r.rate <= 0 {
		r.rate = 1
	}
	n := 0
	for n+FrameBytes <= len(p) {
		i := int(r.pos) * FrameBytes
		if i+FrameBytes > len(r.pcm) {
			if !r.loop {
				if n == 0 {
					return 0, io.EOF
				}
				return n, io.EOF
			}
			r.pos = 0
			i = 0
		}
		copy(p[n:n+FrameBytes], r.pcm[i:i+FrameBytes])
		r.pos += r.rate
		n += FrameBytes
	}
	return n, nil
}

func mixPan(pcm []byte, vol, pan float64) []byte {
	if vol < 0 {
		vol = 0
	}
	if pan < -1 {
		pan = -1
	}
	if pan > 1 {
		pan = 1
	}
	left := vol * math.Min(1, 1-pan)
	right := vol * math.Min(1, 1+pan)
	out := make([]byte, len(pcm))
	for i := 0; i+3 < len(pcm); i += 4 {
		l := float64(int16(pcm[i]) | int16(pcm[i+1])<<8)
		r := float64(int16(pcm[i+2]) | int16(pcm[i+3])<<8)
		ls := int16(clamp16(l * left))
		rs := int16(clamp16(r * right))
		out[i] = byte(ls)
		out[i+1] = byte(ls >> 8)
		out[i+2] = byte(rs)
		out[i+3] = byte(rs >> 8)
	}
	return out
}

func clamp16(v float64) float64 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return v
}

// DistancePan returns gain 0..1 and stereo pan -1..1 from listener to a point.
// yawDeg is Blitz yaw (0 faces +Z). rangeDist is the silence distance.
func DistancePan(lx, ly, lz, yawDeg, x, y, z, rangeDist float64) (vol, pan float64) {
	if rangeDist <= 0 {
		rangeDist = 40
	}
	dx, dy, dz := x-lx, y-ly, z-lz
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	vol = 1 - dist/rangeDist
	if vol < 0 {
		vol = 0
	}
	if vol > 1 {
		vol = 1
	}
	rad := yawDeg * math.Pi / 180
	c, s := math.Cos(rad), math.Sin(rad)
	localX := dx*c + dz*s
	localZ := -dx*s + dz*c
	den := math.Abs(localX) + math.Abs(localZ)
	if den < 1e-6 {
		return vol, 0
	}
	pan = localX / den
	if pan < -1 {
		pan = -1
	}
	if pan > 1 {
		pan = 1
	}
	return vol, pan
}
