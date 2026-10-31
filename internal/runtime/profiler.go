package runtime

import (
	"fmt"
	"image/color"
	"math"
	goruntime "runtime"
	"time"

	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"moonbasic/internal/value"
)

// The profiler times each frame's parts on the CPU (script, physics,
// render), the GPU with timestamp queries (they cannot clash with the
// particle system's elapsed-time query), and samples memory twice a second.
// ShowProfiler draws it in the corner; the Stats* commands read it.
const profFrames = 120

type profiler struct {
	show                  bool
	frames                [profFrames]float64 // frame times, ms
	idx, filled           int
	last                  time.Time
	script, physics, rend float64 // ms, smoothed
	gpu                   float64
	scriptStart           time.Time
	physStart             time.Time
	renderStart           time.Time
	queries               [2][2]uint32 // ping-pong pairs of begin/end timestamps
	qFrame                int
	qOK                   bool
	tris                  int
	trisAt                int
	memAt                 time.Time
	heapMB, sysMB         float64
	gcCount               uint32
	gcPause               float64 // ms, last pause
	allocKB               float64 // per frame, smoothed
	lastAlloc             uint64
	lastAllocFrames       int
	labels                []*gui.Label
	textAt                time.Time
	lines                 []string
}

// smooth eases a timing toward its newest value so the readout is legible.
func smooth(old, v float64) float64 {
	if old == 0 {
		return v
	}
	return old + (v-old)*0.1
}

func (p *profiler) frameStart() {
	now := time.Now()
	if !p.last.IsZero() {
		p.frames[p.idx] = float64(now.Sub(p.last)) / float64(time.Millisecond)
		p.idx = (p.idx + 1) % profFrames
		if p.filled < profFrames {
			p.filled++
		}
	}
	p.last = now
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func (p *profiler) scriptBegin() { p.scriptStart = time.Now() }
func (p *profiler) scriptEnd()   { p.script = smooth(p.script, ms(time.Since(p.scriptStart))) }
func (p *profiler) physBegin()   { p.physStart = time.Now() }
func (p *profiler) physEnd()     { p.physics = smooth(p.physics, ms(time.Since(p.physStart))) }

// stats over the recorded frames: average and worst.
func (p *profiler) frameStats() (avg, worst float64) {
	if p.filled == 0 {
		return 0, 0
	}
	for i := 0; i < p.filled; i++ {
		avg += p.frames[i]
		worst = math.Max(worst, p.frames[i])
	}
	return avg / float64(p.filled), worst
}

// sampleMemory reads the Go heap twice a second; ReadMemStats briefly
// stops the world, so not every frame.
func (p *profiler) sampleMemory(frames int) {
	if time.Since(p.memAt) < 500*time.Millisecond {
		return
	}
	p.memAt = time.Now()
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)
	p.heapMB = float64(m.HeapAlloc) / (1 << 20)
	p.sysMB = float64(m.Sys) / (1 << 20)
	p.gcCount = m.NumGC
	p.gcPause = float64(m.PauseNs[(m.NumGC+255)%256]) / 1e6
	if p.lastAlloc != 0 && frames > p.lastAllocFrames {
		p.allocKB = float64(m.TotalAlloc-p.lastAlloc) / 1024 / float64(frames-p.lastAllocFrames)
	}
	p.lastAlloc, p.lastAllocFrames = m.TotalAlloc, frames
}

// GPU timing: a timestamp before and after the frame's GL work. Results
// are read a frame later so the CPU never waits on the GPU.
func (p *profiler) gpuBegin() {
	if p.queries[0][0] == 0 {
		gl.GenQueries(2, &p.queries[0][0])
		gl.GenQueries(2, &p.queries[1][0])
	}
	q := p.queries[p.qFrame%2]
	gl.QueryCounter(q[0], gl.TIMESTAMP)
}

func (p *profiler) gpuEnd() {
	q := p.queries[p.qFrame%2]
	gl.QueryCounter(q[1], gl.TIMESTAMP)
	prev := p.queries[(p.qFrame+1)%2]
	if p.qOK {
		var ready int32
		gl.GetQueryObjectiv(prev[1], gl.QUERY_RESULT_AVAILABLE, &ready)
		if ready != 0 {
			var t0, t1 uint64
			gl.GetQueryObjectui64v(prev[0], gl.QUERY_RESULT, &t0)
			gl.GetQueryObjectui64v(prev[1], gl.QUERY_RESULT, &t1)
			if t1 > t0 {
				p.gpu = smooth(p.gpu, float64(t1-t0)/1e6)
			}
		}
	}
	p.qOK = true
	p.qFrame++
}

// triangles in visible meshes, counted once per frame while shown.
func (w *World) triangleCount() int {
	p := &w.prof
	if p.trisAt == w.renderSerial && w.renderSerial != 0 {
		return p.tris
	}
	p.trisAt = w.renderSerial
	n := 0
	for _, e := range w.ents {
		if e == nil || e.node == nil || !e.node.GetNode().Visible() {
			continue
		}
		w.walkMeshes(e.node, func(m *graphic.Mesh) {
			if g := m.GetGeometry(); g != nil {
				if idx := g.Indices(); len(idx) > 0 {
					n += len(idx) / 3
				} else {
					n += g.Items() / 3
				}
			}
		})
	}
	p.tris = n
	return n
}

func (w *World) profileLines() []string {
	p := &w.prof
	if time.Since(p.textAt) < 250*time.Millisecond && p.lines != nil {
		return p.lines
	}
	p.textAt = time.Now()
	avg, worst := p.frameStats()
	fps := 0.0
	if avg > 0 {
		fps = 1000 / avg
	}
	gpu := "n/a"
	if p.qOK && p.gpu > 0 {
		gpu = fmt.Sprintf("%.1f", p.gpu)
	}
	p.lines = []string{
		fmt.Sprintf("FPS %.0f   frame %.1f ms (worst %.1f)", fps, avg, worst),
		fmt.Sprintf("script %.1f  physics %.1f  render %.1f  GPU %s ms", p.script, p.physics, p.rend, gpu),
	}
	if w.mode2D {
		p.lines = append(p.lines, fmt.Sprintf("sprites %d", len(w.sprites)))
	} else {
		p.lines = append(p.lines, fmt.Sprintf("draws %d  tris %s  entities %d", w.visibleMeshCount(), kilo(w.triangleCount()), len(w.ents)))
	}
	p.lines = append(p.lines, fmt.Sprintf("heap %.0f MB  sys %.0f MB  GC %d (%.2f ms)  %.0f KB/frame", p.heapMB, p.sysMB, p.gcCount, p.gcPause, p.allocKB))
	return p.lines
}

func kilo(n int) string {
	switch {
	case n >= 1000000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

// profBar is one graph column: height and colour from the frame time
// against the 60 fps (16.7 ms) and 30 fps (33.3 ms) budgets.
func profBar(frameMS float64) (h float64, r, g, b uint8) {
	h = math.Min(frameMS/50, 1) * 60
	switch {
	case frameMS <= 16.8:
		return h, 90, 210, 110
	case frameMS <= 33.4:
		return h, 235, 200, 70
	}
	return h, 235, 80, 70
}

const (
	profY      = 8
	profLineH  = 16
	profGraphW = profFrames * 3
)

// profX puts the overlay in the top-right corner, clear of the HUD text
// scripts usually put top-left.
func (w *World) profX() int {
	sw, _ := w.screenSize2D()
	return max(8, sw-profGraphW-12)
}

// overlay3D queues the profiler for the HUD pass: labels in the scene (they
// are lifted above post like Text) and the graph as HUD rects.
func (w *World) profilerOverlay3D() {
	p := &w.prof
	if !p.show || w.scene == nil {
		for _, l := range p.labels {
			if l.Parent() != nil {
				l.Parent().GetNode().Remove(l)
			}
		}
		p.labels = nil
		return
	}
	lines := w.profileLines()
	profX := w.profX()
	for len(p.labels) < len(lines) {
		l := gui.NewLabel(" ")
		l.SetColor(&math32.Color{R: 0.92, G: 0.95, B: 1})
		l.SetBgColor4(&math32.Color4{R: 0.05, G: 0.06, B: 0.09, A: 0.75})
		p.labels = append(p.labels, l)
	}
	for i, l := range p.labels {
		if i >= len(lines) {
			l.SetVisible(false)
			continue
		}
		if l.Text() != lines[i] {
			l.SetText(lines[i])
		}
		l.SetPosition(float32(profX), float32(profY+i*profLineH))
		l.SetVisible(true)
		if l.Parent() == nil {
			w.scene.Add(l)
		}
	}
	gy := float32(profY + len(lines)*profLineH + 6)
	w.draws = append(w.draws, drawOp{kind: 1, x: float32(profX), y: gy, w: profGraphW, h: 62, r: 13, g: 15, b: 23, a: 190, filled: true})
	for i := 0; i < p.filled; i++ {
		f := p.frames[(p.idx-p.filled+i+profFrames)%profFrames]
		h, r, g, b := profBar(f)
		w.draws = append(w.draws, drawOp{kind: 1, x: float32(profX + i*3), y: gy + 61 - float32(h), w: 2, h: float32(h), r: r, g: g, b: b, a: 255, filled: true})
	}
	// The 60 fps line.
	w.draws = append(w.draws, drawOp{kind: 1, x: float32(profX), y: gy + 61 - float32(16.7/50*60), w: profGraphW, h: 1, r: 120, g: 130, b: 150, a: 200, filled: true})
}

func (w *World) profilerOverlay2D(screen *ebiten.Image) {
	p := &w.prof
	if !p.show {
		return
	}
	lines := w.profileLines()
	profX := w.profX()
	vector.FillRect(screen, float32(profX-4), profY-4, profGraphW+8, float32(len(lines)*profLineH+76), color.RGBA{13, 15, 23, 200}, false)
	for i, s := range lines {
		ebitenutil.DebugPrintAt(screen, s, profX, profY+i*profLineH)
	}
	gy := float32(profY + len(lines)*profLineH + 6)
	for i := 0; i < p.filled; i++ {
		f := p.frames[(p.idx-p.filled+i+profFrames)%profFrames]
		h, r, g, b := profBar(f)
		vector.FillRect(screen, float32(profX+i*3), gy+61-float32(h), 2, float32(h), color.RGBA{r, g, b, 255}, false)
	}
	vector.FillRect(screen, float32(profX), gy+61-float32(16.7/50*60), profGraphW, 1, color.RGBA{120, 130, 150, 200}, false)
}

func (w *World) profilerCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	p := &w.prof
	num := func(fn func() float64) cmd {
		return n(func(a []value.Value) (value.Value, error) { return value.Num(fn()), nil })
	}
	return map[string]cmd{
		// ShowProfiler [on]: the corner overlay.
		"showprofiler": n(func(a []value.Value) (value.Value, error) {
			p.show = argI(a, 0, 1) != 0
			return z()
		}),
		"statsframems": num(func() float64 { avg, _ := p.frameStats(); return avg }),
		"statsworstms": num(func() float64 { _, worst := p.frameStats(); return worst }),
		"statsscriptms": num(func() float64 { return p.script }),
		"statsphysicsms": num(func() float64 { return p.physics }),
		"statsrenderms": num(func() float64 { return p.rend }),
		// StatsGPUMs(): 0 until the GPU has timed a frame (or in 2D).
		"statsgpums":     num(func() float64 { return p.gpu }),
		"statstriangles": num(func() float64 { return float64(w.triangleCount()) }),
		"statsmemorymb": num(func() float64 {
			p.memAt = time.Time{}
			p.sampleMemory(w.loopFrames)
			return p.heapMB
		}),
		"statsgcpausems": num(func() float64 { return p.gcPause }),
		"statsallockb":   num(func() float64 { return p.allocKB }),
	}
}
