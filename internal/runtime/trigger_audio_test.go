package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestAudio3DAndTriggers(t *testing.T) {
	w := New(".")
	w.ready = true

	// Test audio sound range
	w.sounds[1] = &sndSlot{minDist: 1, maxDist: 40}
	_, err := w.Call("setsoundrange", []value.Value{value.Num(1), value.Num(3), value.Num(60)})
	if err != nil {
		t.Fatal(err)
	}
	if w.sounds[1].minDist != 3 || w.sounds[1].maxDist != 60 {
		t.Fatalf("unexpected sound range: min=%f max=%f", w.sounds[1].minDist, w.sounds[1].maxDist)
	}

	// Test trigger creation and queries
	e1 := w.addEntity(&Entity{}, 0)
	e2 := w.addEntity(&Entity{}, 0)

	_, err = w.Call("setbodytrigger", []value.Value{value.Num(float64(e1)), value.Num(1)})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate overlap contact
	w.ents[e1].collided = append(w.ents[e1].collided, e2)

	v, err := w.Call("checktrigger", []value.Value{value.Num(float64(e1)), value.Num(float64(e2))})
	if err != nil || v.Num != 1 {
		t.Fatalf("checktrigger expected 1, got %v, err=%v", v, err)
	}

	hit, err := w.Call("gettriggerhit", []value.Value{value.Num(float64(e1))})
	if err != nil || int(hit.Num) != e2 {
		t.Fatalf("gettriggerhit expected %d, got %v, err=%v", e2, hit, err)
	}

	cnt, err := w.Call("triggeroverlaps", []value.Value{value.Num(float64(e1))})
	if err != nil || int(cnt.Num) != 1 {
		t.Fatalf("triggeroverlaps expected 1, got %v, err=%v", cnt, err)
	}
}
