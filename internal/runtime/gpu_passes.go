package runtime

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Per-pass GPU timing, so tuning follows measurements: a timestamp between each
// pass of the frame, read back a few frames later so the CPU never waits on the
// GPU. MOON_GPU_PROFILE=1 prints the table every 300 frames; StatsPassMs(name$)
// reads one pass. Drivers can report wrong timings, so a value is a guide.
const (
	gpuShadow = iota
	gpuReflect
	gpuRefract
	gpuScene
	gpuPost
	gpuPassCount
)

const passRing = 4 // frames in flight; results are read this many frames late

var passNames = [gpuPassCount]string{"shadow", "reflect", "refract", "scene", "post"}

type gpuPasses struct {
	queries [passRing][gpuPassCount + 1]uint32
	used    [passRing][gpuPassCount + 1]bool
	frame   int
	ms      [gpuPassCount]float64 // smoothed
	printed int
	ready   bool
	failed  bool
}

func (g *gpuPasses) ensure() bool {
	if g.ready {
		return true
	}
	if g.failed {
		return false
	}
	var bits int32
	gl.GetQueryiv(gl.TIMESTAMP, gl.QUERY_COUNTER_BITS, &bits)
	if bits == 0 {
		g.failed = true // no usable timer on this driver
		return false
	}
	for i := range g.queries {
		gl.GenQueries(int32(len(g.queries[i])), &g.queries[i][0])
	}
	g.ready = true
	return true
}

// passBegin opens the frame's table, after reading the oldest one.
func (w *World) passBegin() {
	g := &w.passes
	if !g.ensure() {
		return
	}
	g.frame = (g.frame + 1) % passRing
	g.read(g.frame)
	gl.QueryCounter(g.queries[g.frame][0], gl.TIMESTAMP)
	g.used[g.frame][0] = true
}

// passEnd closes one pass. A pass that did not run this frame reads zero.
func (w *World) passEnd(pass int) {
	g := &w.passes
	if !g.ready {
		return
	}
	gl.QueryCounter(g.queries[g.frame][pass+1], gl.TIMESTAMP)
	g.used[g.frame][pass+1] = true
}

// read turns a finished frame's stamps into pass times.
func (g *gpuPasses) read(f int) {
	if !g.used[f][0] {
		return
	}
	// Still in flight: drop this sample, since waiting would stall the frame.
	last := 0
	for p := 0; p <= gpuPassCount; p++ {
		if g.used[f][p] {
			last = p
		}
	}
	var avail uint32
	gl.GetQueryObjectuiv(g.queries[f][last], gl.QUERY_RESULT_AVAILABLE, &avail)
	if avail == 0 {
		g.used[f] = [gpuPassCount + 1]bool{}
		return
	}
	var prev uint64
	gl.GetQueryObjectui64v(g.queries[f][0], gl.QUERY_RESULT, &prev)
	for p := 0; p < gpuPassCount; p++ {
		t := 0.0
		if g.used[f][p+1] {
			var now uint64
			gl.GetQueryObjectui64v(g.queries[f][p+1], gl.QUERY_RESULT, &now)
			if now >= prev {
				t = float64(now-prev) / 1e6
			}
			prev = now
		}
		g.ms[p] = smooth(g.ms[p], t)
		g.used[f][p+1] = false
	}
	g.used[f][0] = false
	if os.Getenv("MOON_GPU_PROFILE") == "" {
		return
	}
	g.printed++
	if g.printed%300 == 0 {
		var b strings.Builder
		b.WriteString("GPU ms:")
		total := 0.0
		for i, n := range passNames {
			fmt.Fprintf(&b, " %s %.2f", n, g.ms[i])
			total += g.ms[i]
		}
		fmt.Fprintf(&b, " total %.2f", total)
		fmt.Println(b.String())
	}
}

func (w *World) passCommands(n func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// StatsPassMs(name$): GPU milliseconds of shadow, reflect, refract,
		// scene, or post, a few frames old. 0 if the driver has no timer.
		"statspassms": n(func(a []value.Value) (value.Value, error) {
			name := strings.ToLower(strings.TrimSpace(argS(a, 0)))
			for i, p := range passNames {
				if p == name {
					return value.Num(w.passes.ms[i]), nil
				}
			}
			return value.Value{}, fmt.Errorf("StatsPassMs: want shadow, reflect, refract, scene, or post, got %q", name)
		}),
	}
}
