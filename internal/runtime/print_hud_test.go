package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestHudPrintPersistsUntilCls(t *testing.T) {
	w := New(".")
	if _, err := w.Call("hudprint", []value.Value{value.Str("Game loop started. Use ESC to quit.")}); err != nil {
		t.Fatal(err)
	}
	if len(w.hudLines) != 1 {
		t.Fatalf("Print must queue on-screen lines, got %d", len(w.hudLines))
	}
	w.clearFrameText()
	if len(w.hudLines) != 1 {
		t.Fatal("Print must survive Flip/Text clear; only Cls removes it")
	}
	if _, err := w.Call("cls", nil); err != nil {
		t.Fatal(err)
	}
	if len(w.hudLines) != 0 || len(w.hudLabs) != 0 {
		t.Fatal("Cls must clear the Print cursor and lines")
	}
}

func TestHudPrintDoesNotTouchTextSlice(t *testing.T) {
	w := New(".")
	w.hudPrint("hello")
	if len(w.texts) != 0 {
		t.Fatal("Print must not share Text() labels (claw.bb)")
	}
}
