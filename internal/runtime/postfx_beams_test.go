package runtime

import (
	"testing"

	"moonbasic/internal/value"
)

func TestLightBeamCommand(t *testing.T) {
	w := New(".")
	w.ready = true
	spot, err := w.Call("createspotlight", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("lightbeam", []value.Value{spot, value.Num(2)}); err != nil {
		t.Fatal(err)
	}
	e, _ := w.ent(int(spot.Num))
	if e.beam != 2 || !w.post.on {
		t.Fatalf("LightBeam should set the strength and turn the post stack on, got %v %v", e.beam, w.post.on)
	}
	w.Call("lightbeam", []value.Value{spot, value.Num(-1)})
	if e.beam != 0 {
		t.Fatal("a negative strength should be off")
	}
	cube, _ := w.Call("createcube", nil)
	if _, err := w.Call("lightbeam", []value.Value{cube, value.Num(1)}); err == nil {
		t.Fatal("LightBeam on a mesh should be an error")
	}
}

func TestBeamLightsNearestFour(t *testing.T) {
	w := New(".")
	w.ready = true
	cam, _ := w.Call("createcamera", nil)
	w.Call("positionentity", []value.Value{cam, value.Num(0), value.Num(0), value.Num(0)})
	var far value.Value
	for i := 0; i < 6; i++ {
		p, err := w.Call("createpointlight", nil)
		if err != nil {
			t.Fatal(err)
		}
		w.Call("positionentity", []value.Value{p, value.Num(float64(i * 20)), value.Num(0), value.Num(0)})
		w.Call("lightrange", []value.Value{p, value.Num(5)})
		w.Call("lightbeam", []value.Value{p, value.Num(1)})
		far = p
	}
	if w.cam == nil {
		t.Skip("no camera in this build")
	}
	got := w.beamLights()
	if len(got) != maxBeamLights {
		t.Fatalf("want the nearest %d lights, got %d", maxBeamLights, len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].pos.Length() < got[i-1].pos.Length() {
			t.Fatal("beam lights should come nearest first")
		}
	}
	fe, _ := w.ent(int(far.Num))
	fp := entityWorldPos(fe)
	for _, b := range got {
		if b.pos == fp {
			t.Fatal("the farthest light should be left out")
		}
		if b.cos > -1 || b.shadow != -1 {
			t.Fatal("a point light without shadows has no cone and no shadow tile")
		}
	}
}
