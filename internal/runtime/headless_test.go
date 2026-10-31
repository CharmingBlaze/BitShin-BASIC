package runtime

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"moonbasic/internal/interp"
	"moonbasic/internal/parse"
)

func TestHeadlessRunsLogicWithoutWindow(t *testing.T) {
	t.Setenv("MOON_HEADLESS_HZ", "1000")
	prog, err := parse.Parse(`Graphics3D 640, 480
cam = CreateCamera()
light = CreateLight()
cube = CreateCube()
water = CreateWater()
SetWaterCaustics 1
ShowProfiler 1
SetRetroPreset "ps1"
EnableShadows 1
Color 255, 0, 0
Text 10, 10, "hud"
frames = 0
While frames < 5
	MoveEntity cube, 0, 0, 1
	UpdateWorld
	RenderWorld
	Flip
	frames = frames + 1
Wend
w = GraphicsWidth()
End
`)
	if err != nil {
		t.Fatal(err)
	}
	w := New(t.TempDir())
	w.SetHeadless(true)
	in := interp.New(prog, w)
	w.Bind(in)
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	if err := w.LoopHeadless(in); err != nil {
		t.Fatal(err)
	}
	if !in.Done() || w.loopFrames < 5 {
		t.Fatalf("script did not finish: done=%v frames=%d", in.Done(), w.loopFrames)
	}
	if w.app != nil {
		t.Fatal("headless opened a window")
	}
	moved := false
	for _, e := range w.ents {
		// Five MoveEntity steps of 1 along the cube's forward axis.
		if z := e.node.GetNode().Position().Z; e.kind == "cube" && (z > 4.5 || z < -4.5) {
			moved = true
		}
	}
	if !moved {
		t.Fatal("MoveEntity did not run headless")
	}
}

func TestAutoPakFindsDataPak(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "data.pak"))
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	ew, _ := zw.Create("assets/level.csv")
	ew.Write([]byte("packed"))
	ew, _ = zw.Create("assets/both.csv")
	ew.Write([]byte("packed"))
	zw.Close()
	f.Close()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "both.csv"), []byte("loose"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	if w.pak == nil {
		t.Fatal("data.pak beside the script was not opened")
	}
	defer w.pak.close()
	p, err := w.openPath("assets/level.csv")
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "packed" {
		t.Fatalf("pak file read %q", b)
	}
	p, err = w.openPath("assets/both.csv")
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "loose" {
		t.Fatalf("a loose file must override the pak, read %q", b)
	}
}
