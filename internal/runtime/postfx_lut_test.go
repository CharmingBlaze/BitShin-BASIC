package runtime

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"moonbasic/internal/value"
)

func TestPresetLUTIdentity(t *testing.T) {
	data, ok := buildPresetLUT("Identity", 4)
	if !ok || len(data) != 4*4*4*3 {
		t.Fatalf("identity LUT: ok %v len %d", ok, len(data))
	}
	// Red fastest, then green, then blue.
	i := ((3*4+1)*4 + 2) * 3 // b=3 g=1 r=2
	if data[i] != 2.0/3 || data[i+1] != 1.0/3 || data[i+2] != 1 {
		t.Errorf("texel b3 g1 r2 = %v", data[i:i+3])
	}
	for name := range lutPresets {
		d, ok := buildPresetLUT(name, 8)
		if !ok {
			t.Fatalf("preset %s missing", name)
		}
		for _, v := range d {
			if v < 0 || v > 1 {
				t.Fatalf("preset %s value %v out of range", name, v)
			}
		}
	}
	if _, ok := buildPresetLUT("sepia-ish", 8); ok {
		t.Error("unknown preset accepted")
	}
}

func TestParseCubeLUT(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("TITLE \"test\"\n# comment\nLUT_3D_SIZE 2\nDOMAIN_MIN 0 0 0\nDOMAIN_MAX 2 2 2\n")
	for bi := 0; bi < 2; bi++ {
		for gi := 0; gi < 2; gi++ {
			for ri := 0; ri < 2; ri++ {
				b.WriteString(strings.Join([]string{ftoa(ri * 2), ftoa(gi * 2), ftoa(bi * 2)}, " ") + "\n")
			}
		}
	}
	p := filepath.Join(dir, "t.cube")
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	data, size, err := loadLUTFile(p)
	if err != nil || size != 2 {
		t.Fatalf("parse: size %d err %v", size, err)
	}
	// Last texel is white once the 0-2 domain is normalised.
	if got := data[len(data)-3:]; got[0] != 1 || got[1] != 1 || got[2] != 1 {
		t.Errorf("last texel %v", got)
	}
	short := filepath.Join(dir, "short.cube")
	_ = os.WriteFile(short, []byte("LUT_3D_SIZE 2\n0 0 0\n"), 0o644)
	if _, _, err := loadLUTFile(short); err == nil {
		t.Error("short cube accepted")
	}
}

func ftoa(i int) string { return string(rune('0' + i)) }

func TestImageLUTLayouts(t *testing.T) {
	// Strip: 16x4 is size 4. Square: 8x8 is size 4 in 2x2 tiles.
	for _, wh := range [][2]int{{16, 4}, {8, 8}} {
		img := image.NewRGBA(image.Rect(0, 0, wh[0], wh[1]))
		for y := 0; y < wh[1]; y++ {
			for x := 0; x < wh[0]; x++ {
				img.Set(x, y, color.RGBA{uint8(x * 10), uint8(y * 10), 0, 255})
			}
		}
		data, size, err := imageLUT(img)
		if err != nil || size != 4 || len(data) != 64*3 {
			t.Fatalf("%dx%d: size %d len %d err %v", wh[0], wh[1], size, len(data), err)
		}
	}
	if _, _, err := imageLUT(image.NewRGBA(image.Rect(0, 0, 10, 3))); err == nil {
		t.Error("10x3 image accepted as a LUT")
	}
}

func TestLUTAndOutlineCommandsNoGL(t *testing.T) {
	w := New(".")
	if _, err := w.Call("setlut", []value.Value{value.Str("teal_orange"), value.Num(0.6)}); err != nil {
		t.Fatal(err)
	}
	if w.post.lut.size != lutPresetSize || w.post.lut.amount != 0.6 || !w.post.on {
		t.Errorf("SetLUT preset: size %d amount %v post %v", w.post.lut.size, w.post.lut.amount, w.post.on)
	}
	if _, err := w.Call("setlut", []value.Value{value.Str("no_such_lut.cube")}); err == nil {
		t.Error("SetLUT accepted a missing file")
	}
	if _, err := w.Call("clearlut", nil); err != nil || w.post.lut.amount != 0 {
		t.Errorf("ClearLUT: amount %v err %v", w.post.lut.amount, err)
	}
	if _, err := w.Call("setoutline", []value.Value{value.Num(2), value.Num(255), value.Num(0), value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	if w.post.outline.width != 2 || w.post.outline.color.R != 1 || w.post.outline.color.G != 0 {
		t.Errorf("SetOutline: %+v", w.post.outline)
	}
}

func TestPostShaderHasLUTAndOutline(t *testing.T) {
	for _, need := range []string{"uniform sampler3D uLUT;", "applyLUT(rgb)", "outlineEdge(vUV)"} {
		if !strings.Contains(mbpostFragment, need) {
			t.Errorf("post shader missing %q", need)
		}
	}
}
