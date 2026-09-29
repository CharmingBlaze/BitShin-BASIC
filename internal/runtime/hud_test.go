package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestHudOverlayQueue(t *testing.T) {
	w := New(".")
	w.ready = true
	w.mode2D = false // 3D mode

	// Test SetAlpha / Color with alpha
	_, err := w.Call("color", []value.Value{value.Num(255), value.Num(128), value.Num(64), value.Num(200)})
	if err != nil {
		t.Fatal(err)
	}
	if w.drawRGB != [3]uint8{255, 128, 64} {
		t.Fatalf("unexpected drawRGB: %v", w.drawRGB)
	}
	if w.drawAlpha != 200 {
		t.Fatalf("unexpected drawAlpha: %d", w.drawAlpha)
	}

	_, _ = w.Call("rect", []value.Value{value.Num(10), value.Num(20), value.Num(100), value.Num(50), value.Num(1)})
	_, _ = w.Call("oval", []value.Value{value.Num(200), value.Num(200), value.Num(40), value.Num(40)})
	_, _ = w.Call("line", []value.Value{value.Num(0), value.Num(0), value.Num(800), value.Num(600)})

	if len(w.draws) != 3 {
		t.Fatalf("expected 3 draw ops, got %d", len(w.draws))
	}
	if w.draws[0].kind != 1 || w.draws[0].a != 200 || !w.draws[0].filled {
		t.Fatalf("draws[0] rect mismatch: %+v", w.draws[0])
	}
	if w.draws[1].kind != 2 || w.draws[1].a != 200 {
		t.Fatalf("draws[1] oval mismatch: %+v", w.draws[1])
	}
	if w.draws[2].kind != 3 || w.draws[2].a != 200 {
		t.Fatalf("draws[2] line mismatch: %+v", w.draws[2])
	}

	// Test CreateImage and DrawImage in 3D
	res, err := w.Call("createimage", []value.Value{value.Num(64), value.Num(64)})
	if err != nil {
		t.Fatal(err)
	}
	imgID := int(res.Num)
	if imgID <= 0 {
		t.Fatalf("expected valid image ID, got %d", imgID)
	}
	im := w.images[imgID]
	if im == nil || im.src == nil || im.w != 64 || im.h != 64 {
		t.Fatalf("invalid ebiImage created in 3D: %+v", im)
	}

	_, _ = w.Call("drawimage", []value.Value{value.Num(float64(imgID)), value.Num(50), value.Num(50)})
	if len(w.draws) != 4 {
		t.Fatalf("expected 4 draw ops after DrawImage, got %d", len(w.draws))
	}
	if w.draws[3].kind != 0 || w.draws[3].imgID != imgID {
		t.Fatalf("draws[3] drawimage mismatch: %+v", w.draws[3])
	}

	// Test Cls clears HUD
	_, _ = w.Call("cls", nil)
	if len(w.draws) != 0 {
		t.Fatalf("expected 0 draw ops after cls, got %d", len(w.draws))
	}
}
