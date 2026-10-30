package runtime

import "math"
import "testing"

func integrateArcGTAO(h1, h2, n, cosN float64) float64 {
	a := h1 * 2
	b := h2 * 2
	s := math.Sin(n)
	return 0.25 * ((-math.Cos(a-n) + cosN + a*s) + (-math.Cos(b-n) + cosN + b*s))
}

func TestAdaptExposureFollowsBrightness(t *testing.T) {
	bright, _ := adaptExposure(1, false, 0.85, 0.5, 1.6, 1.0/60)
	if bright >= 0.8 {
		t.Fatalf("a washed frame should darken, got %v", bright)
	}
	next, _ := adaptExposure(bright, true, 0.12, 0.5, 1.6, 0.5)
	if next <= bright {
		t.Fatalf("a dark frame should open the exposure, %v -> %v", bright, next)
	}
}

func TestGTAOOpenSliceStaysLit(t *testing.T) {
	open := integrateArcGTAO(-math.Pi/2, math.Pi/2, 0, 1)
	if open < 0.98 || open > 1.02 {
		t.Fatalf("flat slice visibility %v", open)
	}
	creased := integrateArcGTAO(-0.4, 0.4, 0, 1)
	if creased >= open-0.2 {
		t.Fatalf("raised horizons should darken, open=%v creased=%v", open, creased)
	}
}
