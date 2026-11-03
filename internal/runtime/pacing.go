package runtime

import (
	"sort"

	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Frame pacing. GL 3.3 has no way to ask when a frame reached the display, so
// what is left is to stay on budget, keep the queue short, and report the worst
// frames instead of an average. An average of 60 fps can hide a stutter a player
// feels at once, which is why games quote 1% lows.
//
// A fence at each frame's end lets the next frames wait on the GPU, so the driver
// never queues more than paceDepth frames. That bounds input lag and stops a burst
// of queued frames ending in a long stall.

const (
	paceDepth    = 2   // frames the GPU may fall behind
	paceWaitNS   = 5e7 // never wait longer than this on a fence, in nanoseconds
	paceSamples  = 600 // frames of history: ten seconds at 60 fps
	paceMissedAt = 1.5 // a frame is missed past this share of the target
)

type framePacer struct {
	fences [paceDepth]uintptr
	next   int
	ms     [paceSamples]float32
	count  int
	at     int
	missed int
}

// paceFrame runs after a frame's GL work. It records the frame's time and bounds
// how far the GPU can fall behind.
func (w *World) paceFrame() {
	p := &w.pace
	if w.delta > 0 {
		p.add(float32(w.delta*1000), float32(w.dyn.budget()*1000))
	}
	old := p.fences[p.next]
	if old != 0 {
		gl.ClientWaitSync(old, gl.SYNC_FLUSH_COMMANDS_BIT, paceWaitNS)
		gl.DeleteSync(old)
	}
	p.fences[p.next] = gl.FenceSync(gl.SYNC_GPU_COMMANDS_COMPLETE, 0)
	p.next = (p.next + 1) % paceDepth
}

func (p *framePacer) add(ms, budget float32) {
	p.ms[p.at] = ms
	p.at = (p.at + 1) % paceSamples
	if p.count < paceSamples {
		p.count++
	}
	if ms > budget*paceMissedAt {
		p.missed++
	}
}

// percentile is the frame time that this share of recent frames beat: 50 is the
// typical frame, 99 the 1% low.
func (p *framePacer) percentile(pct float64) float32 {
	if p.count == 0 {
		return 0
	}
	s := append([]float32(nil), p.ms[:p.count]...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	i := int(pct / 100 * float64(p.count-1))
	if i < 0 {
		i = 0
	}
	if i >= p.count {
		i = p.count - 1
	}
	return s[i]
}

func (w *World) pacingCommands(n func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// StatsFrameMs(percentile#): the frame time this share of the last ten
		// seconds' frames beat. 50 is typical, 99 is the 1% low.
		"statsframems": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.pace.percentile(argN(a, 0, 99)))), nil
		}),
		// StatsShaderMs(): milliseconds spent building shader programs so far.
		"statsshaderms": n(func(a []value.Value) (value.Value, error) {
			return value.Num(shaderStats.ms), nil
		}),
		// StatsMissedFrames(): frames that ran over 1.5 times the target so far.
		"statsmissedframes": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.pace.missed)), nil
		}),
	}
}
