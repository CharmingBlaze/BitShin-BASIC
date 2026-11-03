package runtime

import (
	"testing"

	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func TestEachEffectWorksAlone(t *testing.T) {
	for _, c := range []struct {
		cmd string
		a   float64
	}{
		{"setvignette", 0.5}, {"setfilmgrain", 0.2}, {"setbloom", 0.5}, {"setfxaa", 1},
		{"setmsaa", 4}, {"setexposure", 1.2}, {"setsharpen", 0.5},
		{"setmotionblur", 1}, {"setchromaticaberration", 0.3},
	} {
		w := New(".")
		w.ready = true
		if _, err := w.Call(c.cmd, []value.Value{value.Num(c.a)}); err != nil {
			t.Fatal(c.cmd, err)
		}
		if !w.post.on {
			t.Errorf("%s alone should turn the stack on", c.cmd)
		}
		if w.postTonemap() != 3 {
			t.Errorf("%s alone should not bring a tone curve, got %d", c.cmd, w.postTonemap())
		}
		if c.cmd != "setbloom" && w.post.bloom != 0 {
			t.Errorf("%s alone should not turn on bloom", c.cmd)
		}
		if c.cmd != "setfxaa" && w.post.fxaa {
			t.Errorf("%s alone should not turn on FXAA", c.cmd)
		}
	}
}

func TestTurningAnEffectOffDoesNotTurnTheStackOn(t *testing.T) {
	for _, cmd := range []string{"setvignette", "setfilmgrain", "setbloom", "setfxaa", "setmsaa", "setmotionblur", "setdepthoffield"} {
		w := New(".")
		w.ready = true
		if _, err := w.Call(cmd, []value.Value{value.Num(0)}); err != nil {
			t.Fatal(cmd, err)
		}
		if w.post.on {
			t.Errorf("%s(0) should leave the stack off", cmd)
		}
	}
}

func TestAskingForACurveOrALookIsNotPlain(t *testing.T) {
	w := New(".")
	w.ready = true
	w.Call("setvignette", []value.Value{value.Num(0.5)})
	w.Call("settonemap", []value.Value{value.Str("aces")})
	if w.postTonemap() != 2 {
		t.Fatalf("an asked-for curve should be used, got %d", w.postTonemap())
	}
	w = New(".")
	w.ready = true
	w.Call("setvignette", []value.Value{value.Num(0.5)})
	w.setPostDefaults()
	if w.post.plain || w.post.bloom == 0 || !w.post.fxaa {
		t.Fatal("a whole look should bring its defaults")
	}
}

func TestOffStaysOffThroughLooksAndTiers(t *testing.T) {
	w := New(".")
	w.ready = true
	w.Call("setbloom", []value.Value{value.Num(0)})
	w.Call("setfxaa", []value.Value{value.Num(0)})
	w.setPostDefaults()
	if w.post.bloom != 0 || w.post.fxaa {
		t.Fatal("defaults should not undo a script's off")
	}
	w.Call("setmsaa", []value.Value{value.Num(0)})
	w.setQuality(qualityUltra)
	if w.post.msaa != 0 {
		t.Fatal("ultra should not undo a script's MSAA off")
	}
	if w.post.bloom != 0 || w.post.fxaa {
		t.Fatal("a tier should not undo a script's off")
	}
	w.Call("setbloom", []value.Value{value.Num(0.4)})
	w.setPostDefaults()
	if w.post.bloom != 0.4 {
		t.Fatal("turning it back on should clear the off")
	}
}

func TestDepthOfFieldCommand(t *testing.T) {
	w := New(".")
	w.ready = true
	if _, err := w.Call("setdepthoffield", []value.Value{value.Num(12), value.Num(30), value.Num(500)}); err != nil {
		t.Fatal(err)
	}
	l := w.post.lens
	if l.focus != 12 || l.focusRange != 30 || l.blurPx != 32 || !w.post.on {
		t.Fatalf("got focus %v range %v blur %v", l.focus, l.focusRange, l.blurPx)
	}
	w.Call("setdepthoffield", nil)
	if w.post.lens.focusRange != 0 {
		t.Fatal("no arguments should turn it off")
	}
}

func TestShutterIsTheSameAtAnyFrameRate(t *testing.T) {
	if s := shutterFraction(1, 1.0/60); s < 0.99 || s > 1.01 {
		t.Fatalf("at 60 fps a 1/60 s shutter is the whole frame, got %v", s)
	}
	if s := shutterFraction(1, 1.0/30); s < 0.49 || s > 0.51 {
		t.Fatalf("at 30 fps it is half the frame, got %v", s)
	}
	if s := shutterFraction(1, 0); s > 4.01 {
		t.Fatalf("a zero time step should not blow up, got %v", s)
	}
}

func TestCameraCutIsNotBlurred(t *testing.T) {
	var still math32.Matrix4
	still.Identity()
	if reprojectionIsCut(&still) {
		t.Fatal("a still camera is not a cut")
	}
	var jump math32.Matrix4
	jump.MakeTranslation(3, 0, 0)
	if !reprojectionIsCut(&jump) {
		t.Fatal("a jump across the screen is a cut")
	}
}
