package runtime

import (
	"testing"
	"time"

	"github.com/g3n/engine/core"
)

func TestCreateClothDrapes(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	id := w.createCloth(2, 2, 8, 8, 1)
	if id == 0 || w.cloths[id] == nil {
		t.Fatal("CreateCloth should return a sheet")
	}
	if w.phys3.ClothVertexCount(id) == 0 {
		t.Fatal("cloth body missing vertices")
	}
	_, y0, _, ok := w.phys3.GetPosition(id)
	if !ok {
		t.Fatal("cloth pose missing")
	}
	w.started = time.Now()
	for i := 0; i < 45; i++ {
		w.loopFrames = i
		w.applyClothWind()
		w.phys3.Step(1.0 / 60.0)
		w.tickCloth()
	}
	_, y1, _, ok := w.phys3.GetPosition(id)
	if !ok {
		t.Fatal("cloth pose missing after step")
	}
	if y1 > y0-0.08 {
		t.Fatalf("cloth should sag, y %v -> %v", y0, y1)
	}
}

func TestClothWindKeepsMoving(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.started = time.Now().Add(-time.Second)
	id := w.createCloth(2, 1.4, 10, 8, 1)
	if c := w.cloths[id]; c != nil {
		c.wind = 1
	}
	snap := func() []float32 {
		buf := make([]float32, 10*8*3)
		n := w.phys3.ClothVertices(id, buf)
		return append([]float32{}, buf[:n*3]...)
	}
	for i := 0; i < 25; i++ {
		w.loopFrames = i
		w.applyClothWind()
		w.phys3.Step(1.0 / 60.0)
	}
	a := snap()
	for i := 25; i < 50; i++ {
		w.loopFrames = i
		w.applyClothWind()
		w.phys3.Step(1.0 / 60.0)
	}
	b := snap()
	if len(a) < 9 || len(a) != len(b) {
		t.Fatalf("cloth verts %d -> %d", len(a), len(b))
	}
	max := float32(0)
	for i := 0; i < len(a); i++ {
		d := a[i] - b[i]
		if d < 0 {
			d = -d
		}
		if d > max {
			max = d
		}
	}
	if max < 0.02 {
		t.Fatalf("cloth should keep moving under wind, max delta %v", max)
	}
}
