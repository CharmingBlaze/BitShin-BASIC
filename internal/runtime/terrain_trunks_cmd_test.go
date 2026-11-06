package runtime

import (
	"testing"

	"moonbasic/internal/value"
)

// With no trees planted, nothing is hit and nothing moves.
func TestTreeCommandsWithoutTrees(t *testing.T) {
	w := New(".")
	w.ready = true
	if v, _ := w.Call("treehit", []value.Value{value.Num(3), value.Num(4), value.Num(1)}); v.Num != 0 {
		t.Fatalf("TreeHit with no trees = %v", v.Num)
	}
	if v, _ := w.Call("treepush", []value.Value{value.Num(3), value.Num(4), value.Num(0.5)}); v.Num != 0 {
		t.Fatalf("TreePush with no trees = %v", v.Num)
	}
	px, _ := w.Call("treepushx", nil)
	pz, _ := w.Call("treepushz", nil)
	if px.Num != 3 || pz.Num != 4 {
		t.Fatalf("TreePush should leave the point where it was, got %v, %v", px.Num, pz.Num)
	}
}
