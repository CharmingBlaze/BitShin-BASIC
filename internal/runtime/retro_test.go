package runtime

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"moonbasic/internal/value"
)

func TestRetroTargetAndViewport(t *testing.T) {
	w := New(".")
	if w.retroOn() {
		t.Fatal("retro on by default")
	}
	if pw, ph := w.retroTarget(1280, 720); pw != 1280 || ph != 720 {
		t.Fatalf("off: %dx%d", pw, ph)
	}
	w.retro.lowW = 320 // height follows the window's shape
	if pw, ph := w.retroTarget(1280, 720); pw != 320 || ph != 180 {
		t.Fatalf("320 wide on 16:9: %dx%d", pw, ph)
	}
	w.retro.lowH = 240
	w.retro.pixelPerfect = true
	// 1280x720 holds 320x240 three times: 960x720, centred.
	if x, y, vw, vh := w.retroViewport(1280, 720, 320, 240); x != 160 || y != 0 || vw != 960 || vh != 720 {
		t.Fatalf("pixel-perfect viewport %d,%d %dx%d", x, y, vw, vh)
	}
	if a := w.retroAspect(1280, 720, 320, 240); a < 1.333 || a > 1.334 {
		t.Fatalf("letterboxed aspect %v", a)
	}
	w.retro = retroPost{scale: 0.5}
	if pw, ph := w.retroTarget(1280, 720); pw != 640 || ph != 360 || !w.retroOn() {
		t.Fatalf("render scale: %dx%d", pw, ph)
	}
	if x, y, vw, vh := w.retroViewport(1280, 720, 640, 360); x != 0 || y != 0 || vw != 1280 || vh != 720 {
		t.Fatalf("render scale fills the window: %d,%d %dx%d", x, y, vw, vh)
	}
}

func TestRetroPalettes(t *testing.T) {
	for name, cols := range retroPalettes {
		pal := hexPalette(cols)
		if len(pal) != len(cols)*3 {
			t.Fatalf("%s: %d values", name, len(pal))
		}
		for _, v := range pal {
			if v < 0 || v > 1 {
				t.Fatalf("%s: value %v", name, v)
			}
		}
	}
	if p := hexPalette([]string{"ff8000"}); p[0] != 1 || p[1] < 0.5 || p[1] > 0.51 || p[2] != 0 {
		t.Fatalf("ff8000 -> %v", p)
	}

	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for i, c := range []color.RGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {255, 0, 0, 255}, {0, 0, 255, 255}, {0, 255, 0, 255}, {0, 0, 0, 255}} {
		img.Set(i%3, i/3, c)
	}
	p := filepath.Join(dir, "pal.png")
	f, _ := os.Create(p)
	png.Encode(f, img)
	f.Close()
	pal, err := imagePalette(p)
	if err != nil || len(pal) != 4*3 {
		t.Fatalf("image palette: %d colours, %v", len(pal)/3, err)
	}
}

func TestRetroCommands(t *testing.T) {
	w := New(".")
	cmds := w.commandTable()
	run := func(name string, args ...value.Value) error {
		_, err := cmds[name](args)
		return err
	}
	if err := run("setretroresolution", value.Num(320), value.Num(240), value.Num(1)); err != nil {
		t.Fatal(err)
	}
	if w.retro.lowW != 320 || !w.retro.pixelPerfect || !w.post.on {
		t.Fatalf("resolution %+v post %v", w.retro, w.post.on)
	}
	run("setretroresolution", value.Num(0))
	if w.retroOn() {
		t.Fatal("SetRetroResolution 0 did not turn it off")
	}
	run("setrenderscale", value.Num(0.1))
	if w.retro.scale != 0.25 {
		t.Fatalf("render scale clamps to 0.25: %v", w.retro.scale)
	}
	run("setrenderscale", value.Num(1))
	if w.retro.scale != 0 {
		t.Fatal("render scale 1 should turn it off")
	}
	run("setretrocolors", value.Num(1))
	if w.retro.levels != 2 {
		t.Fatalf("one level makes no image: %v", w.retro.levels)
	}
	if err := run("setretropalette", value.Str("GameBoy")); err != nil || len(w.retro.palette) != 12 {
		t.Fatalf("gameboy palette: %v %d", err, len(w.retro.palette))
	}
	if err := run("setretropalette", value.Str("vaporwave")); err == nil || !strings.Contains(err.Error(), "pico8") {
		t.Fatalf("unknown palette error %v", err)
	}
	run("setretropalette", value.Str("none"))
	if w.retro.palette != nil {
		t.Fatal("palette not cleared")
	}
	if err := run("settexturefilter", value.Num(99), value.Str("nearest")); err == nil {
		t.Error("SetTextureFilter accepted a missing texture")
	}
}

func TestPostShaderHasRetro(t *testing.T) {
	for _, need := range []string{"uniform sampler2D uRetroPal;", "float bayer4(", "rgb = retro(rgb, vUV);"} {
		if !strings.Contains(mbpostFragment, need) {
			t.Errorf("post shader missing %q", need)
		}
	}
}
