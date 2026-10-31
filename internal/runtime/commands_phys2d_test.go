package runtime

import (
	"strings"
	"testing"

	"moonbasic/internal/value"
)

func TestPhys2DCommands(t *testing.T) {
	w := New(".")
	w.ready = true
	cmds := w.commandTable()
	call := func(name string, args ...float64) value.Value {
		t.Helper()
		vs := make([]value.Value, len(args))
		for i, a := range args {
			vs[i] = value.Num(a)
		}
		v, err := cmds[name](vs)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return v
	}

	call("physics2d")
	call("gravity2d", 0, 900)
	w.sprites[1] = &ebiSprite{id: 1, x: 0, y: 400, sx: 1, sy: 1}
	call("createbox2d", 1, 2000, 40, 0, 0)
	w.sprites[2] = &ebiSprite{id: 2, x: 0, y: 200, sx: 1, sy: 1}
	call("createbox2d", 2, 100, 100, 0, 0)
	call("setsensor2d", 2)
	w.nextID = 10
	ch := int(call("createcharacter2d", 0, 100, 24, 48).Num)
	if ch != 10 {
		t.Fatalf("character handle %d", ch)
	}
	entered := false
	for i := 0; i < 90; i++ {
		w.updateWorld()
		if call("sensorentered2d", 2, float64(ch)).Num == 1 {
			entered = true
			if call("sensorbody2d", 2, 1).Num != float64(ch) || call("sensorcount2d", 2).Num != 1 {
				t.Fatal("sensor body list")
			}
		}
	}
	if !entered || call("characteronground2d", float64(ch)).Num != 1 {
		t.Fatalf("entered=%v ground=%v", entered, call("characteronground2d", float64(ch)).Num)
	}
	if y := call("bodyy2d", float64(ch)).Num; y < 355 || y > 357 {
		t.Fatalf("character y %v", y)
	}
	if id := call("shapecast2d", 300, 0, 300, 500, 20, 20).Num; id != 1 {
		t.Fatalf("shapecast hit %v", id)
	}
	if call("casthit2d").Num != 1 || call("castnormaly2d").Num > -0.99 || call("pickedentity").Num != 1 {
		t.Fatal("cast getters")
	}

	for name, args := range map[string][]float64{
		"sensorcount2d":       {1},
		"setkinematic2d":      {99},
		"setcollisionlayer2d": {1, 40},
		"movecharacter2d":     {1, 1},
	} {
		vs := make([]value.Value, len(args))
		for i, a := range args {
			vs[i] = value.Num(a)
		}
		_, err := cmds[name](vs)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), name) {
			t.Fatalf("%s: want an error naming the command, got %v", name, err)
		}
	}
}
