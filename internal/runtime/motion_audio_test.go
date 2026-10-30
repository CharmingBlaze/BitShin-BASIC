package runtime

import "testing"

func TestStepDueWaitsForAStride(t *testing.T) {
	accum, play := stepDue(0, 2, 0.5)
	if play || accum < 0.9 || accum > 1.1 {
		t.Fatalf("half stride accum=%v play=%v", accum, play)
	}
	accum, play = stepDue(accum, 2, 0.4)
	if !play || accum < 0.2 || accum > 0.4 {
		t.Fatalf("stride accum=%v play=%v", accum, play)
	}
	accum, play = stepDue(1, 0.2, 0.5)
	if play || accum != 1 {
		t.Fatalf("standing still should not step, accum=%v play=%v", accum, play)
	}
}
