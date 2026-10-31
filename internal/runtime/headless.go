package runtime

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"moonbasic/internal/interp"
	"moonbasic/internal/value"
)

// Headless mode runs a script with no window and no GL context, for
// dedicated servers and tests. Graphics3D / Graphics2D succeed without
// opening a window, entities, physics, timers, files, and networking still
// run, and commands that only draw, play sound, or show UI do nothing.
//
//	MOON_HEADLESS=1   run headless (moon run -headless and -headless builds set it)
//	MOON_HEADLESS_HZ  frames per second of the headless loop (default 60)

func headlessEnv() bool { return envOn("MOON_HEADLESS") }

// SetHeadless turns headless mode on or off. Call it before the script runs.
func (w *World) SetHeadless(on bool) {
	w.headless = on
	w.cmdMap = nil
}

// Headless reports whether the world runs without a window.
func (w *World) Headless() bool { return w.headless }

// headlessGraphics is Graphics3D / Graphics2D without a window: the world is
// ready for entities and the screen size reads back, but nothing is drawn.
func (w *World) headlessGraphics(width, height int) (value.Value, error) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	w.scrW, w.scrH = width, height
	w.ready = true
	w.mode2D = false
	return value.Num(0), nil
}

// headlessStubTables are the command tables that only draw, play sound, or
// drive windows and UI. In headless mode each of their commands returns 0.
func (w *World) headlessStubTables(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) []map[string]cmd {
	return []map[string]cmd{
		w.lighting2DCommands(n, z),
		w.guiCommands(n, z),
		w.g3nGuiCommands(n, z, need),
		w.glmodernCommands(n, z, need),
		w.postCommands(n, z, need),
		w.shaderCommands(n, z, need),
		w.lutCommands(n, z),
		w.retroCommands(n, z),
		w.retroGeomCommands(n, z),
		w.screenshotCommands(n, z),
		w.profilerCommands(n, z),
		w.debugDrawCommands(n, z),
		w.extraWindowCommands(n, z),
		w.audioCommands(n, z),
		w.gpuParticleCommands(n, z, need),
		w.shadingCommands(n, z),
		w.qualityCommands(n, z),
	}
}

// headlessGLCommands upload GL textures from otherwise headless-safe tables
// (water keeps its entities, levels, and waves for buoyancy).
var headlessGLCommands = []string{"setwaternormalmap", "setwaterdudv", "setwatercaustics", "setwaterambientsound"}

// applyHeadless swaps render-only commands for no-ops and Graphics* for
// windowless versions. It runs on the finished table, before aliases.
func (w *World) applyHeadless(m map[string]cmd, n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) {
	nop := n(func([]value.Value) (value.Value, error) { return z() })
	for _, t := range w.headlessStubTables(n, z, need) {
		for k := range t {
			m[k] = nop
		}
	}
	for _, k := range headlessGLCommands {
		m[k] = nop
	}
	for _, k := range []string{"graphics3d", "graphics2d", "graphics"} {
		m[k] = n(func(a []value.Value) (value.Value, error) {
			return w.headlessGraphics(argI(a, 0, 800), argI(a, 1, 600))
		})
	}
	m["graphicsapp"] = n(func(a []value.Value) (value.Value, error) {
		v, err := w.headlessGraphics(argI(a, 0, 960), argI(a, 1, 640))
		if err == nil {
			w.spawnCamera(0)
		}
		return v, err
	})
	m["waittimer"] = n(func(a []value.Value) (value.Value, error) {
		t := w.timers[argI(a, 0, 0)]
		if t == nil || t.hz <= 0 {
			return z()
		}
		for {
			total := int64(time.Since(t.start).Seconds() * t.hz)
			if pending := total - t.consumed; pending > 0 {
				t.consumed = total
				return value.Num(float64(pending)), nil
			}
			time.Sleep(time.Millisecond)
		}
	})
	m["delay"] = n(func(a []value.Value) (value.Value, error) {
		if ms := argI(a, 0, 0); ms > 0 {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
		return z()
	})
}

var headlessWarned sync.Map

// headlessSkip reports, once per command, a command that failed for want of
// a window. The script goes on with a 0 result.
func headlessSkip(name string, why interface{}) {
	if _, seen := headlessWarned.LoadOrStore(name, true); seen {
		return
	}
	fmt.Fprintf(os.Stderr, "headless: %s skipped (%v)\n", name, why)
}

// LoopHeadless resumes the script after each Flip at a fixed rate until it
// ends or calls End / EndGraphics. MOON_SMOKE_FRAMES ends the process after n
// Flips, as in a windowed smoke run (tickSmoke).
func (w *World) LoopHeadless(in *interp.Interp) error {
	hz := 60
	if v, _ := strconv.Atoi(os.Getenv("MOON_HEADLESS_HZ")); v > 0 {
		hz = v
	}
	frame := time.Second / time.Duration(hz)
	last := time.Now()
	for !w.quit && !in.Done() {
		if d := frame - time.Since(last); d > 0 {
			time.Sleep(d)
		}
		last = time.Now()
		w.tickDelta(frame)
		w.loopFrames++
		if err := in.Run(); err != nil {
			return err
		}
	}
	return in.Err()
}
