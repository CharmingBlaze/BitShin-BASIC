package runtime

import (
	"testing"

	"moonbasic/internal/value"
)

func TestSMAATables(t *testing.T) {
	if len(smaaAreaBytes) != smaaAreaW*smaaAreaH*2 {
		t.Fatalf("area table is %d bytes, want %d", len(smaaAreaBytes), smaaAreaW*smaaAreaH*2)
	}
	if len(smaaSearchBytes) != smaaSearchW*smaaSearchH {
		t.Fatalf("search table is %d bytes, want %d", len(smaaSearchBytes), smaaSearchW*smaaSearchH)
	}
	for _, r := range smaaSourceASCII() {
		if r > 127 {
			t.Fatal("the shader source should reach the driver as plain ASCII")
		}
	}
}

func TestSMAASwitches(t *testing.T) {
	w := New(".")
	w.ready = true
	w.Call("setsmaa", []value.Value{value.Num(1)})
	if !w.post.on || !w.post.fxaa || !w.post.smaa {
		t.Fatal("SetSMAA alone should turn on edge smoothing, by SMAA")
	}
	w.Call("setfxaa", []value.Value{value.Num(1)})
	if !w.post.fxaa || w.post.smaa {
		t.Fatal("SetFXAA should pick FXAA")
	}
	w.Call("setsmaa", []value.Value{value.Num(1)})
	w.Call("setfxaa", []value.Value{value.Num(0)})
	if w.post.fxaa {
		t.Fatal("SetFXAA(False) should turn edge smoothing off")
	}
}

func TestSMAAByTier(t *testing.T) {
	w := New(".")
	w.ready = true
	w.setQuality(qualityHigh)
	if !w.post.smaa {
		t.Fatal("high should use SMAA")
	}
	w.setQuality(qualityMedium)
	if w.post.smaa {
		t.Fatal("medium should use FXAA")
	}
	w.Call("setsmaa", []value.Value{value.Num(1)})
	w.setQuality(qualityLow)
	if !w.post.smaa {
		t.Fatal("a tier should not undo the script's choice of SMAA")
	}
	w = New(".")
	w.ready = true
	w.Call("setfxaa", []value.Value{value.Num(1)})
	w.setQuality(qualityUltra)
	if w.post.smaa {
		t.Fatal("a tier should not undo the script's choice of FXAA")
	}
}
