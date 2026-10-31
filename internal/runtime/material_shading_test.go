package runtime

import (
	"testing"

	"moonbasic/internal/value"
)

func TestShadingModesNoGL(t *testing.T) {
	w := New(".")
	mat, err := w.Call("createpbrmaterial", nil)
	if err != nil {
		t.Fatal(err)
	}
	id := value.Num(float64(mat.Int()))
	shading := func() int {
		v, err := w.Call("getmaterialshading", []value.Value{id})
		if err != nil {
			t.Fatal(err)
		}
		return v.Int()
	}
	for _, c := range []struct {
		arg  value.Value
		want int
	}{
		{value.Str("toon"), shadeToon},
		{value.Str("UNLIT"), shadeUnlit},
		{value.Num(0), shadePBR},
		{value.Num(2), shadeToon},
	} {
		if _, err := w.Call("setmaterialshading", []value.Value{id, c.arg}); err != nil {
			t.Fatal(err)
		}
		if got := shading(); got != c.want {
			t.Errorf("SetMaterialShading %v: shading %d, want %d", c.arg, got, c.want)
		}
	}
	if _, err := w.Call("setmaterialshading", []value.Value{id, value.Str("chrome")}); err == nil {
		t.Error("SetMaterialShading accepted an unknown mode")
	}

	pm := w.pbrLib[mat.Int()]
	if _, err := w.Call("settoon", []value.Value{id, value.Num(40), value.Num(2), value.Num(-1)}); err != nil {
		t.Fatal(err)
	}
	if pm.toonBands != 16 || pm.toonRim != 1 || pm.toonSpec != 0 {
		t.Errorf("SetToon clamps: bands %v rim %v spec %v", pm.toonBands, pm.toonRim, pm.toonSpec)
	}

	if _, err := w.Call("entityfx", []value.Value{id, value.Num(fxFullbright | fxNoFog)}); err != nil {
		t.Fatal(err)
	}
	if pm.shading != shadeUnlit || !pm.noFog {
		t.Errorf("EntityFX 9: shading %d noFog %v", pm.shading, pm.noFog)
	}
	if _, err := w.Call("entityfx", []value.Value{id, value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	if pm.shading != shadePBR || pm.noFog {
		t.Errorf("EntityFX 0 did not reset: shading %d noFog %v", pm.shading, pm.noFog)
	}
	if _, err := w.Call("setunlit", []value.Value{id}); err != nil || pm.shading != shadeUnlit {
		t.Errorf("SetUnlit: shading %d err %v", pm.shading, err)
	}
}

func TestPBRShaderHasShadingModes(t *testing.T) {
	for _, need := range []string{"uniform int uShading;", "toonDirect(", "uShading == 1", "uNoFog == 0"} {
		if !containsStr(mbphysicalFragment, need) {
			t.Errorf("mbphysical fragment missing %q", need)
		}
	}
}
