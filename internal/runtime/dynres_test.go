package runtime

import (
	"math"
	"testing"
)

const (
	dynFast = 1.0 / 60
	dynSlow = 1.0 / 30
)

func runDyn(d *dynRes, dt, seconds float64, busy bool) {
	for i := 0; i < int(seconds/dt); i++ {
		d.frame(dt, busy)
	}
}

func nearScale(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func TestDynResDropsToTheFloorOnASlowGPU(t *testing.T) {
	var d dynRes
	d.setFloor(0.5)
	runDyn(&d, dynSlow, 1+3, true)
	if !nearScale(d.scale, 0.5) || !d.atFloor() {
		t.Fatalf("scale %v should reach the floor", d.scale)
	}
}

func TestDynResSharpensAgainButNotPastWhatWasTooMuch(t *testing.T) {
	var d dynRes
	d.setFloor(0.5)
	runDyn(&d, dynFast, 1, true) // warm-up
	runDyn(&d, dynSlow, 1.2, true)
	dropped := d.scale
	if dropped >= 0.85 {
		t.Fatalf("scale %v should have dropped", dropped)
	}
	runDyn(&d, dynFast, 10, true)
	if d.scale <= dropped || d.scale >= 0.9-1e-4 {
		t.Fatalf("scale %v should be sharper than %v, below what ran late", d.scale, dropped)
	}
	runDyn(&d, dynFast, 60, true)
	if !nearScale(d.scale, 1) {
		t.Fatalf("much later full size should be tried again, got %v", d.scale)
	}
}

func TestDynResIgnoresStallsAndWorkThatIsNotDrawing(t *testing.T) {
	var d dynRes
	d.setFloor(0.75)
	runDyn(&d, 1, 5, true) // loading hitches
	if !nearScale(d.scale, 1) {
		t.Fatalf("stalls moved the scale to %v", d.scale)
	}
	runDyn(&d, dynSlow, 5, false) // a script that sleeps
	if !nearScale(d.scale, 1) {
		t.Fatalf("a slow script moved the scale to %v", d.scale)
	}
	runDyn(&d, dynSlow, 5, true)
	if !nearScale(d.scale, 0.75) {
		t.Fatalf("slow drawing should reach the tier floor, got %v", d.scale)
	}
}

func TestShadowCascadesTakeTurns(t *testing.T) {
	counts := [4]int{}
	for f := 0; f < 400; f++ {
		for i := range counts {
			if shadowCascadeDue(i, f) {
				counts[i]++
			}
		}
	}
	if counts != [4]int{400, 200, 100, 100} {
		t.Fatalf("redraws per cascade over 400 frames: %v", counts)
	}
}

func TestDynResStaysAwayWhenTheScriptChoseAScale(t *testing.T) {
	w := New(".")
	w.post.on = true
	if !w.dynActive() {
		t.Fatal("on by default")
	}
	w.retroScr.userScale = true
	if w.dynActive() {
		t.Fatal("SetRenderScale owns the scale")
	}
	w.retroScr.userScale = false
	w.retroScr.lowW = 320
	if w.dynActive() {
		t.Fatal("a retro resolution owns the scale")
	}
}

// The low tier keeps its post pass: it was the slowest tier without one, since
// the scene then drew into G3N's 8x multisampled window, and losing it stalled.
func TestLowTierKeepsThePostPass(t *testing.T) {
	if !tiers[qualityLow].post {
		t.Fatal("low must keep the post pass")
	}
	for i, s := range tiers {
		if !s.post {
			t.Fatalf("tier %d has no post pass", i)
		}
	}
}
