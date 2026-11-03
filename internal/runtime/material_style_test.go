package runtime

import "testing"

func TestStyleNamesAcceptHowPeopleWriteThem(t *testing.T) {
	for in, want := range map[string]string{
		"Low-Poly": "lowpoly", "low poly": "lowpoly", "Hand-Painted": "painted", "Pixelated 3D": "pixel",
		"Retro PS1": "ps1", "Realistic PBR": "realistic", "Cel Shading": "cel", "Toon Shading": "toon",
		"Flat Shading": "flat", "Unlit": "unlit",
	} {
		got := styleName(in)
		if got != want {
			t.Fatalf("%q: got %q, want %q", in, got, want)
		}
		if _, ok := styles[got]; !ok {
			t.Fatalf("%q maps to %q, which is not a style", in, got)
		}
	}
}

func TestShadingModesAreDistinct(t *testing.T) {
	seen := map[int]string{}
	for _, name := range []string{"pbr", "unlit", "toon", "cel", "flat", "lowpoly", "painted"} {
		m, ok := shadingMode(name, 0)
		if !ok {
			t.Fatalf("%q is not a mode", name)
		}
		if other, dup := seen[m]; dup {
			t.Fatalf("%q and %q are the same mode", name, other)
		}
		seen[m] = name
	}
}

func TestStyleLeavesOwnShadersAlone(t *testing.T) {
	for _, name := range []string{"terrain", "water", "clouds", "scatter", treeCards} {
		e := &Entity{name: name}
		if styleTakes(e) {
			t.Fatalf("SetStyle should not reach %q", name)
		}
	}
}
