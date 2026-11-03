package runtime

import (
	"math"

	"moonbasic/internal/value"
)

// Dynamic resolution: when frames run late the 3D scene draws smaller and FSR 1
// brings it back up, so a slow GPU keeps its frame rate by giving up a little
// sharpness instead of effects. Only at the bottom of a tier's range does the
// tier itself drop. The method follows DBPRO2026's (MIT).
//
// Frame time cannot show spare GPU time while vsync waits. So where the driver has
// a GPU timer this reads it and acts before a frame is missed, with limits scaled
// from Valve's: drop past 92% of the budget, sharpen only if the next step is
// predicted under 80%. Without a timer it works from missed frames. Either way it
// drops quickly, remembers the scale that was too much, and tries higher slowly.
// The remembered ceiling relaxes, so a lighter part of the game can sharpen again.
const (
	dynStep     = 0.05 // scale changes in steps of this, per axis
	dynWindow   = 0.5  // seconds per judgement
	dynLate     = 0.25 // a window with more late frames than this drops
	dynRaise    = 4.0  // on-time seconds before trying one step sharper
	dynRelax    = 15.0 // seconds with no drop before the ceiling rises a step
	dynWarm     = 1.0  // the first second (loading, compiles) is not judged
	dynHitch    = 0.25 // longer frames are a stall, not GPU load
	dynFPS      = 60.0 // default target
	dynLeeway   = 1.15 // a frame is late past this share of the budget
	dynBusyFrac = 0.6  // render or GPU time must reach this share of the budget

	dynGPUHigh  = 0.92 // GPU share of the budget that drops
	dynGPUFar   = 1.1  // past this, drop two steps
	dynGPULow   = 0.8  // the next step must be predicted under this to sharpen
	dynGPUEasy  = 0.6  // under this, the next step ignores the remembered ceiling
	dynGPUOver  = 3    // frames in a row over dynGPUHigh before dropping
	dynGPUHold  = 8    // frames to ignore after a change: timers read old frames
	dynGPURaise = 1.5  // seconds with room to spare before sharpening
)

// dynScaleFloor is the lowest scale per tier, per axis. 0.5 is a quarter of
// the pixels. A low tier gives up more before it has nothing left to drop.
var dynScaleFloor = [...]float64{qualityLow: 0.5, qualityMedium: 0.65, qualityHigh: 0.75, qualityUltra: 0.85}

type dynRes struct {
	off                  bool    // SetDynamicResolution 0
	targetFPS            float64 // 0 is dynFPS
	floor                float64
	scale                float64 // 0 until first used, then the current scale
	ceiling              float64
	elapsed, calm, relax float64
	warm                 float64
	frames, late         int
	started              bool
	gpu                  float64 // GPU seconds a frame, 0 if the driver has no timer
	over, hold           int
}

func (d *dynRes) budget() float64 {
	fps := d.targetFPS
	if fps <= 0 {
		fps = dynFPS
	}
	return 1 / fps
}

func dynSnap(v float64) float64 { return math.Round(v/dynStep) * dynStep }

// setFloor keeps the scale inside a tier's range.
func (d *dynRes) setFloor(floor float64) {
	if !d.started {
		d.started = true
		d.scale, d.ceiling, d.warm = 1, 1+dynStep, dynWarm
	}
	d.floor = floor
	if d.scale < floor {
		d.scale = floor
	}
	if d.ceiling < floor {
		d.ceiling = floor
	}
}

func (d *dynRes) atFloor() bool { return d.scale <= d.floor+1e-4 }

// frame feeds one frame and reports whether the scale changed. busy says the
// frame's time really went on drawing; a script that sleeps or computes for
// long would run late at any resolution.
func (d *dynRes) frame(dt float64, busy bool) bool {
	if dt <= 0 || dt > dynHitch || math.IsNaN(dt) {
		return false
	}
	if d.warm > 0 {
		d.warm -= dt
		return false
	}
	if d.gpu > 0 {
		return d.frameGPU(dt)
	}
	d.elapsed += dt
	d.frames++
	if dt > d.budget()*dynLeeway && busy {
		d.late++
	}
	if d.elapsed < dynWindow {
		return false
	}
	window, late := d.elapsed, float64(d.late)/float64(d.frames)
	d.elapsed, d.frames, d.late = 0, 0, 0
	d.relax += window
	if d.relax >= dynRelax {
		d.relax = 0
		d.ceiling = math.Min(dynSnap(d.ceiling+dynStep), 1+dynStep)
	}
	if late > dynLate {
		d.calm = 0
		if d.atFloor() {
			return false
		}
		d.ceiling, d.relax = d.scale, 0
		steps := 1.0
		if late > 0.75 {
			steps = 2 // mostly late: get back on time sooner
		}
		d.scale = math.Max(dynSnap(d.scale-steps*dynStep), d.floor)
		return true
	}
	if late > 0 {
		d.calm = 0
		return false
	}
	d.calm += window
	if d.calm >= dynRaise && d.scale+dynStep < d.ceiling-1e-4 {
		d.calm = 0
		d.scale = math.Min(dynSnap(d.scale+dynStep), 1)
		return true
	}
	return false
}

// frameGPU judges one frame from GPU time. Pixel work follows the scale squared,
// which is how it predicts the next step.
func (d *dynRes) frameGPU(dt float64) bool {
	d.relax += dt
	if d.relax >= dynRelax {
		d.relax = 0
		d.ceiling = math.Min(dynSnap(d.ceiling+dynStep), 1+dynStep)
	}
	if d.hold > 0 {
		d.hold--
		return false
	}
	use := d.gpu / d.budget()
	if use > dynGPUHigh {
		d.over++
	} else {
		d.over = 0
	}
	if d.over >= dynGPUOver {
		d.over, d.calm = 0, 0
		if d.atFloor() {
			return false
		}
		d.ceiling, d.relax = d.scale, 0
		steps := 1.0
		if use > dynGPUFar {
			steps = 2
		}
		d.scale = math.Max(dynSnap(d.scale-steps*dynStep), d.floor)
		d.hold = dynGPUHold
		return true
	}
	next := d.scale + dynStep
	predicted := use * (next / d.scale) * (next / d.scale)
	// The ceiling guards a step that was too much, unless there is plainly room now.
	if predicted >= dynGPULow || (next >= d.ceiling-1e-4 && predicted >= dynGPUEasy) || next > 1+1e-4 {
		d.calm = 0
		return false
	}
	d.calm += dt
	if d.calm < dynGPURaise {
		return false
	}
	d.calm = 0
	d.scale = math.Min(dynSnap(next), 1)
	d.ceiling = math.Max(d.ceiling, dynSnap(d.scale+dynStep))
	d.hold = dynGPUHold
	return true
}

// dynActive is true while the controller owns the render scale.
func (w *World) dynActive() bool {
	return !w.dyn.off && !w.headless && w.post.on && w.retroScr.lowW == 0 && !w.retroScr.userScale
}

// tickDynRes runs once a frame, before the tier watch.
func (w *World) tickDynRes() {
	if !w.dynActive() {
		return
	}
	floor := dynScaleFloor[qualityHigh]
	if w.quality.set {
		floor = dynScaleFloor[w.quality.tier]
	}
	w.dyn.setFloor(floor)
	limit := w.dyn.budget() * 1000 * dynBusyFrac
	// No GPU timer on this driver: judge by the frame alone.
	busy := w.prof.gpu <= 0 || w.prof.gpu >= limit || w.prof.rend >= limit
	w.dyn.gpu = w.prof.gpu / 1000
	if w.dyn.frame(w.delta, busy) || w.dyn.scale != w.retroScr.scaleOr1() {
		w.retroScr.setScale(w.dyn.scale)
	}
}

func (r *retroPost) scaleOr1() float64 {
	if r.scale <= 0 {
		return 1
	}
	return r.scale
}

func (r *retroPost) setScale(s float64) {
	if s >= 1 {
		r.scale = 0
		return
	}
	r.scale = s
}

func (w *World) dynResCommands(n func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// SetDynamicResolution on [, targetFPS]: draw the 3D scene smaller when
		// frames run late, and sharpen it back up. On unless the script sets
		// SetRenderScale. HUD text stays at full size.
		"setdynamicresolution": n(func(a []value.Value) (value.Value, error) {
			on := argI(a, 0, 1) != 0
			w.dyn.off = !on
			w.retroScr.userScale = false
			if len(a) >= 2 {
				w.dyn.targetFPS = argN(a, 1, dynFPS)
			}
			if !on {
				w.retroScr.scale = 0
				w.dyn.started = false
			}
			return value.Num(float64(b01(on))), nil
		}),
		"getdynamicresolution": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(w.dynActive()))), nil
		}),
	}
}
