package runtime

import (
	"fmt"
	"image/color"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"bitshinbasic/internal/value"
)

// hopJob is one runtime command sent from a compiled script goroutine to the
// thread that owns the window.
type hopJob struct {
	name string
	args []value.Value
	resp chan hopResult
}

type hopResult struct {
	v   value.Value
	err error
}

// Play runs a compiled program. The script runs beside the window thread so
// Flip, Delay, WaitKey, and WaitTimer present real frames. Graphics calls hop
// onto this thread because OpenGL and Ebiten must run there.
func (w *World) Play(body func(*World)) {
	if w == nil || body == nil {
		return
	}
	w.playing = true
	w.hopCh = make(chan hopJob)
	w.hostDead = make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("runtime:", r)
			}
		}()
		body(w)
	}()
	defer close(w.hostDead)
	w.serve(done)
}

// Running reports whether a compiled game loop should keep going.
// Window close and EndGraphics clear it.
func (w *World) Running() bool {
	if w == nil {
		return false
	}
	return !w.quitSeen.Load()
}

func (w *World) requestQuit() {
	w.quit = true
	w.quitSeen.Store(true)
}

func (w *World) hopCall(name string, args []value.Value) (value.Value, error) {
	resp := make(chan hopResult, 1)
	job := hopJob{name: name, args: args, resp: resp}
	select {
	case w.hopCh <- job:
	case <-w.hostDead:
		return value.Value{}, fmt.Errorf("graphics host stopped")
	}
	select {
	case r := <-resp:
		return r.v, r.err
	case <-w.hostDead:
		return value.Value{}, fmt.Errorf("graphics host stopped")
	}
}

func (w *World) serve(done <-chan struct{}) {
	runtime.LockOSThread()
	defer w.shutdownGraphics()
	for {
		select {
		case <-done:
			return
		case job := <-w.hopCh:
			if w.mode2D && w.ready && w.app == nil && isFrameCmd(job.name) {
				// Graphics2D already returned; frame commands belong to the 2D loop.
				// This only happens if a frame command was queued early. Start the loop
				// and let it consume this job first.
				w.serve2D(done, &job)
				return
			}
			w.onHost = true
			v, err := w.callDirect(job.name, job.args)
			w.onHost = false
			reply(job, v, err)
			if w.mode2D && w.ready && w.app == nil {
				w.serve2D(done, nil)
				return
			}
		}
	}
}

func reply(job hopJob, v value.Value, err error) {
	if job.resp == nil {
		return
	}
	job.resp <- hopResult{v: v, err: err}
}

func isFrameCmd(name string) bool {
	switch name {
	case "flip", "delay", "waitkey", "waittimer":
		return true
	}
	return false
}

func (w *World) shutdownGraphics() {
	if w.app == nil {
		return
	}
	if w.app.beforeDestroy != nil {
		w.app.beforeDestroy()
	}
	w.app.Destroy()
	w.app = nil
}

// present3DRest renders and swaps one 3D frame. tickDelta and markFlip run first
// so DeltaTime matches the interpreter: the script reads the previous frame's delta.
func (w *World) present3DRest() {
	if w.app == nil || w.quit {
		return
	}
	w.noteMouse()
	w.loopFrames++
	if !w.presentOK {
		w.drainPhantomQuit()
	}
	w.tickFX()
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("RenderWorld: panic", r)
			}
		}()
		w.render(w.app.rend)
	}()
	// Poll before the close check. Checking first sees ShouldClose false,
	// arms closeReady, then the first PollEvents sets the creation close
	// flag and the next Flip quits the built game.
	w.app.swapOnce()
	if w.windowWantsClose() {
		w.requestQuit()
		return
	}
	if !w.presentOK {
		w.latchHeldKeys()
		w.drainPhantomQuit()
		if w.heldAtFlip[KeyEscape] || w.keys[KeyEscape] {
			w.escapeLatch = true
			w.clearHeldKey(KeyEscape)
			w.heldAtFlip[KeyEscape] = true
		}
		w.presentOK = true
	}
	w.clearFrameText()
	w.draws = w.draws[:0]
}

func (w *World) pumpDelay(ms int) (value.Value, error) {
	if ms <= 0 {
		return value.Num(0), nil
	}
	deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
	for time.Now().Before(deadline) && w.Running() {
		w.tickDelta(0)
		w.markFlip()
		w.present3DRest()
	}
	return value.Num(0), nil
}

func (w *World) pumpWaitKey() (value.Value, error) {
	for w.Running() {
		w.tickDelta(0)
		w.markFlip()
		w.present3DRest()
		if code, ok := w.takeKeyHit(); ok {
			return value.Num(float64(code)), nil
		}
	}
	return value.Num(0), nil
}

func (w *World) pumpWaitTimer(id int) (value.Value, error) {
	for w.Running() {
		if v, ok := w.takeTimer(id); ok {
			return value.Num(v), nil
		}
		w.tickDelta(0)
		w.markFlip()
		w.present3DRest()
	}
	return value.Num(0), nil
}

func (w *World) takeKeyHit() (int, bool) {
	for code, hit := range w.hits {
		if !hit || w.heldAtFlip[code] {
			continue
		}
		w.hits[code] = false
		return code, true
	}
	return 0, false
}

func (w *World) takeTimer(id int) (float64, bool) {
	t := w.timers[id]
	if t == nil {
		return 0, true
	}
	total := int64(time.Since(t.start).Seconds() * t.hz)
	pending := total - t.consumed
	if pending <= 0 {
		return 0, false
	}
	t.consumed = total
	return float64(pending), true
}

// serve2D runs the Ebiten window on this thread. The script stays blocked
// inside Flip / Delay / WaitKey / WaitTimer until the frame has been drawn.
func (w *World) serve2D(done <-chan struct{}, first *hopJob) {
	ebiten.SetWindowSize(w.scrW, w.scrH)
	if w.title != "" {
		ebiten.SetWindowTitle(w.title)
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	g := &play2D{w: w, done: done, first: true, queued: first}
	_ = ebiten.RunGame(g)
	w.requestQuit()
}

type play2D struct {
	w         *World
	done      <-chan struct{}
	first     bool
	queued    *hopJob
	pending   *hopJob
	hold      *hopJob
	holdKind  string
	holdUntil time.Time
	holdID    int
}

func (g *play2D) Update() error {
	w := g.w
	select {
	case <-g.done:
		g.unblock()
		return ebiten.Termination
	default:
	}
	if g.first {
		ebiten.SetWindowClosingHandled(true)
	}
	w.pollEbitenInput()
	// Same phantom-close rule as the 3D window: ignore close until a frame
	// has been presented and a poll has reported the window still open.
	if ebiten.IsWindowBeingClosed() {
		if w.closeReady {
			w.requestQuit()
		} else if w.presentOK && w.loopFrames >= 12 {
			w.closeReady = true
		}
	} else if w.presentOK {
		w.closeReady = true
	}
	if w.presented && !w.presentOK {
		w.latchHeldKeys()
		w.drainPhantomQuit()
		if w.heldAtFlip[KeyEscape] || w.keys[KeyEscape] {
			w.escapeLatch = true
			w.clearHeldKey(KeyEscape)
			w.heldAtFlip[KeyEscape] = true
		}
		w.presentOK = true
	}
	w.noteMouse()
	w.loopFrames++
	tps := ebiten.ActualTPS()
	if tps > 1 {
		w.tickDelta(time.Duration(float64(time.Second) / tps))
	} else {
		w.tickDelta(0)
	}
	if !g.first {
		w.draws = w.draws[:0]
		w.clearFrameText()
	}
	g.first = false
	if g.hold != nil {
		if done, v := g.pollHold(); done {
			reply(*g.hold, v, nil)
			g.hold = nil
			g.holdKind = ""
		} else {
			w.tickFX()
			return nil
		}
	}
	for {
		select {
		case <-g.done:
			g.unblock()
			return ebiten.Termination
		default:
		}
		var job hopJob
		if g.queued != nil {
			job = *g.queued
			g.queued = nil
		} else {
			select {
			case <-g.done:
				g.unblock()
				return ebiten.Termination
			case job = <-w.hopCh:
			}
		}
		if yield, v := g.accept(job); yield {
			if g.holdKind == "flip" {
				w.tickFX()
				return nil
			}
			if g.hold != nil {
				w.tickFX()
				return nil
			}
			reply(job, v, nil)
			continue
		}
	}
}

func (g *play2D) accept(job hopJob) (yield bool, v value.Value) {
	w := g.w
	switch job.name {
	case "flip":
		w.markFlip()
		j := job
		g.pending = &j
		g.holdKind = "flip"
		return true, value.Num(0)
	case "delay":
		ms := 0
		if len(job.args) > 0 {
			ms = int(job.args[0].Number())
		}
		if ms <= 0 || !w.Running() {
			return true, value.Num(0)
		}
		j := job
		g.hold = &j
		g.holdKind = "delay"
		g.holdUntil = time.Now().Add(time.Duration(ms) * time.Millisecond)
		return true, value.Num(0)
	case "waitkey":
		if code, ok := w.takeKeyHit(); ok {
			return true, value.Num(float64(code))
		}
		if !w.Running() {
			return true, value.Num(0)
		}
		j := job
		g.hold = &j
		g.holdKind = "waitkey"
		return true, value.Num(0)
	case "waittimer":
		id := 0
		if len(job.args) > 0 {
			id = int(job.args[0].Number())
		}
		if n, ok := w.takeTimer(id); ok || !w.Running() {
			return true, value.Num(n)
		}
		j := job
		g.hold = &j
		g.holdKind = "waittimer"
		g.holdID = id
		return true, value.Num(0)
	default:
		w.onHost = true
		v, err := w.callDirect(job.name, job.args)
		w.onHost = false
		reply(job, v, err)
		return false, v
	}
}

func (g *play2D) pollHold() (bool, value.Value) {
	w := g.w
	switch g.holdKind {
	case "delay":
		if w.Running() && time.Now().Before(g.holdUntil) {
			return false, value.Num(0)
		}
		return true, value.Num(0)
	case "waitkey":
		if code, ok := w.takeKeyHit(); ok {
			return true, value.Num(float64(code))
		}
		if !w.Running() {
			return true, value.Num(0)
		}
		return false, value.Num(0)
	case "waittimer":
		if n, ok := w.takeTimer(g.holdID); ok || !w.Running() {
			return true, value.Num(n)
		}
		return false, value.Num(0)
	default:
		return true, value.Num(0)
	}
}

func (g *play2D) unblock() {
	if g.pending != nil {
		reply(*g.pending, value.Num(0), nil)
		g.pending = nil
	}
	if g.hold != nil {
		reply(*g.hold, value.Num(0), nil)
		g.hold = nil
	}
}

func (g *play2D) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{g.w.clsRGB[0], g.w.clsRGB[1], g.w.clsRGB[2], 255})
	g.w.draw2D(screen)
	if g.pending != nil {
		reply(*g.pending, value.Num(0), nil)
		g.pending = nil
		g.holdKind = ""
	}
}

func (g *play2D) Layout(ow, oh int) (int, int) {
	if g.w.scrW <= 0 {
		return 800, 600
	}
	return g.w.scrW, g.w.scrH
}
