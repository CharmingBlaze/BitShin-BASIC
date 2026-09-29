package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestPlayRunsCommandsOnTheHost(t *testing.T) {
	w := New(".")
	var got float64
	w.Play(func(w *World) {
		v, err := w.Call("createtimer", []value.Value{value.Num(30)})
		if err != nil {
			t.Errorf("createtimer: %v", err)
			return
		}
		got = v.Number()
		if !w.Running() {
			t.Error("Running() cleared before EndGraphics")
		}
		if _, err := w.Call("endgraphics", nil); err != nil {
			t.Errorf("endgraphics: %v", err)
		}
		if w.Running() {
			t.Error("EndGraphics must stop a compiled game loop")
		}
	})
	if got != 1 {
		t.Fatalf("timer id = %v, want 1", got)
	}
	if w.timers[1] == nil || w.timers[1].hz != 30 {
		t.Fatalf("timer not created on the host: %+v", w.timers[1])
	}
}
