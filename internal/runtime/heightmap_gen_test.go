package runtime

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"bitshinbasic/internal/value"
)

func TestAlpineHeightmapHasRelief(t *testing.T) {
	spec := heightmapPreset("alpine")
	spec.Width, spec.Height, spec.Seed = 64, 64, 19
	h := GenerateHeightmapCPU(spec)
	min, max := float32(1), float32(0)
	for _, v := range h {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if max-min < 0.25 {
		t.Fatalf("alpine field too flat: min=%v max=%v", min, max)
	}
}

func TestGenerateHeightmapDeterministic(t *testing.T) {
	spec := HeightmapSpec{Width: 48, Height: 48, Seed: 7, Octaves: 5, Scale: 40, Persistence: 0.5, Lacunarity: 2, White: 1}
	a := GenerateHeightmapCPU(spec)
	b := GenerateHeightmapCPU(spec)
	if len(a) != 48*48 || len(a) != len(b) {
		t.Fatalf("size %d", len(a))
	}
	sum := 0.0
	for i := range a {
		if a[i] < 0 || a[i] > 1 {
			t.Fatalf("out of range %v", a[i])
		}
		if a[i] != b[i] {
			t.Fatalf("not deterministic at %d", i)
		}
		sum += float64(a[i])
	}
	if sum < 1 || sum > float64(len(a))-1 {
		t.Fatalf("flat or empty map sum=%v", sum)
	}
	other := GenerateHeightmapCPU(HeightmapSpec{Width: 48, Height: 48, Seed: 99, Octaves: 5, Scale: 40, White: 1})
	diff := 0
	for i := range a {
		if math.Abs(float64(a[i]-other[i])) > 0.02 {
			diff++
		}
	}
	if diff < 100 {
		t.Fatalf("seed should change the field: diffs=%d", diff)
	}
}

func TestGenerateHeightmapDiamondSquare(t *testing.T) {
	spec := HeightmapSpec{Width: 33, Height: 33, Seed: 3, Mode: "diamond", Persistence: 0.45, White: 1}
	h := GenerateHeightmapCPU(spec)
	if len(h) != 33*33 {
		t.Fatalf("len %d", len(h))
	}
	mn, mx := h[0], h[0]
	for _, v := range h {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	if mx-mn < 0.2 {
		t.Fatalf("diamond-square too flat %v..%v", mn, mx)
	}
}

func TestHeightmapPresetAlpine(t *testing.T) {
	p := heightmapPreset("alpine")
	if p.Octaves != 7 || p.Scale != 100 {
		t.Fatalf("alpine %+v", p)
	}
}

func TestThermalErodeChangesField(t *testing.T) {
	spec := HeightmapSpec{Width: 32, Height: 32, Seed: 2, Octaves: 4, Scale: 16, White: 1}
	h := GenerateHeightmapCPU(spec)
	before := append([]float32(nil), h...)
	ThermalErodeHeightmap(h, 32, 32, 10, 0.015, 0.3)
	changed := 0
	for i := range h {
		if h[i] != before[i] {
			changed++
		}
		if h[i] < 0 || h[i] > 1 {
			t.Fatalf("erode out of range %v", h[i])
		}
	}
	if changed < 10 {
		t.Fatalf("thermal should move sediment: %d", changed)
	}
}

func TestFilterRidgedAndPNG(t *testing.T) {
	h := GenerateHeightmapCPU(HeightmapSpec{Width: 16, Height: 16, Seed: 1, White: 1})
	FilterHeightmap(h, 16, 16, "ridged", 1)
	dir := t.TempDir()
	path := filepath.Join(dir, "hm.png")
	if err := saveHeightmapPNG(path, h, 16, 16); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() < 40 {
		t.Fatalf("png %v %v", st, err)
	}
	hf, err := loadHeightImage(path, 10, 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	if hf.gw != 16 || hf.gd != 16 {
		t.Fatalf("reload %dx%d", hf.gw, hf.gd)
	}
}

func TestHeightmapCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"generateheightmap", "generateheightmappreset", "saveheightmap",
		"importheightmap", "createterrainfromheightmap", "erodeheightmap",
		"hydraulicerodeheightmap", "filterheightmap", "heightmaptexture",
		"heightmapwidth", "getheightmapwidth",
	} {
		if m[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
}

func TestGenerateHeightmapHandle(t *testing.T) {
	w := New(".")
	fn := w.commandTable()["generateheightmap"]
	v, err := fn([]value.Value{
		value.Num(64), value.Num(64), value.Num(11), value.Num(4),
		value.Num(50), value.Num(0.45), value.Num(2.1),
	})
	if err != nil {
		t.Fatal(err)
	}
	id := v.Int()
	hm := w.hmaps[id]
	if hm == nil || hm.w != 64 || len(hm.data) != 64*64 {
		t.Fatalf("handle %d map %+v", id, hm)
	}
}
