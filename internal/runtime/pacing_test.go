package runtime

import "testing"

func TestPacerPercentilesShowTheWorstFrames(t *testing.T) {
	var p framePacer
	for i := 0; i < 99; i++ {
		p.add(16, 16.7)
	}
	p.add(50, 16.7)
	if got := p.percentile(50); got != 16 {
		t.Fatalf("typical frame %v", got)
	}
	if got := p.percentile(100); got != 50 {
		t.Fatalf("worst frame %v", got)
	}
	if p.missed != 1 {
		t.Fatalf("missed %d", p.missed)
	}
}

func TestPacerForgetsOldFrames(t *testing.T) {
	var p framePacer
	p.add(100, 16.7)
	for i := 0; i < paceSamples; i++ {
		p.add(16, 16.7)
	}
	if got := p.percentile(100); got != 16 {
		t.Fatalf("old frame still counted: %v", got)
	}
	if p.missed != 1 {
		t.Fatalf("the missed count is for the whole run, got %d", p.missed)
	}
}

func TestPacerEmptyIsZero(t *testing.T) {
	var p framePacer
	if got := p.percentile(99); got != 0 {
		t.Fatalf("got %v", got)
	}
}

func runDynGPU(d *dynRes, gpu, seconds float64) {
	d.gpu = gpu
	for i := 0; i < int(seconds*60); i++ {
		d.frame(dynFast, true)
	}
}

func TestDynResDropsOnGPUTimeBeforeAFrameIsMissed(t *testing.T) {
	var d dynRes
	d.setFloor(0.5)
	runDynGPU(&d, 0.008, 1.1) // warm-up
	// 16 ms is on budget, so no frame is late, but there is no headroom left.
	runDynGPU(&d, 0.016, 0.2)
	if d.scale >= 1 {
		t.Fatalf("scale %v should have dropped before missing a frame", d.scale)
	}
}

func TestDynResKeepsAFrameThatFits(t *testing.T) {
	var d dynRes
	d.setFloor(0.5)
	runDynGPU(&d, 0.012, 20) // 72% of budget
	if !nearScale(d.scale, 1) {
		t.Fatalf("scale %v should stay full size", d.scale)
	}
}

func TestDynResSharpensOnlyWhenTheNextStepWouldFit(t *testing.T) {
	var d dynRes
	d.setFloor(0.5)
	runDynGPU(&d, 0.008, 1.1)
	runDynGPU(&d, 0.018, 2)
	low := d.scale
	if low >= 0.9 {
		t.Fatalf("scale %v should have fallen", low)
	}
	// Tight: one step sharper would pass the budget's 80%, so it stays put.
	runDynGPU(&d, 0.0135, 8)
	if !nearScale(d.scale, low) {
		t.Fatalf("scale %v should hold at %v", d.scale, low)
	}
	// Room again: it sharpens, and in time past the ceiling it once hit.
	runDynGPU(&d, 0.005, 60)
	if !nearScale(d.scale, 1) {
		t.Fatalf("scale %v should be back to full size", d.scale)
	}
}

func TestDynResFloorHolds(t *testing.T) {
	var d dynRes
	d.setFloor(0.75)
	runDynGPU(&d, 0.008, 1.1)
	runDynGPU(&d, 0.05, 10)
	if !nearScale(d.scale, 0.75) || !d.atFloor() {
		t.Fatalf("scale %v should rest on the floor", d.scale)
	}
}
