package runtime

import (
	"math"
	"testing"

	"github.com/g3n/engine/core"

	"moonbasic/internal/interp"
	"moonbasic/internal/parse"
	"moonbasic/internal/value"
)

func placeEnt(w *World, x, y, z, yaw float32) int {
	n := core.NewNode()
	gx, gy, gz := toG3N(x, y, z)
	n.SetPosition(gx, gy, gz)
	return w.addEntity(&Entity{kind: "ai", node: n, yaw: yaw}, 0)
}

func TestBoneMaskSkipsNamedBone(t *testing.T) {
	L := &animLayer{mask: map[string]float32{"arm": 0, "spine": 0.5}}
	if L.boneWeight("Arm") != 0 || L.boneWeight("spine") != 0.5 || L.boneWeight("hips") != 1 {
		t.Fatalf("weights arm=%v spine=%v hips=%v", L.boneWeight("Arm"), L.boneWeight("spine"), L.boneWeight("hips"))
	}
	open := &animLayer{}
	if open.boneWeight("arm") != 1 {
		t.Fatal("a layer with no mask weighs every bone")
	}
}

func TestRootMotionMovesOnce(t *testing.T) {
	root := core.NewNode()
	bone := core.NewNode()
	bone.SetName("hips")
	root.Add(bone)
	e := &Entity{node: root, anim: &animState{rootOn: true, rootBone: "hips"}}
	bone.SetPosition(1, 2, 3)
	e.anim.applyRootMotion(e)
	bone.SetPosition(1.4, 9, 3.25)
	e.anim.applyRootMotion(e)
	p := root.Position()
	if math.Abs(float64(p.X-0.4)) > 1e-4 || math.Abs(float64(p.Y)) > 1e-4 || math.Abs(float64(p.Z-0.25)) > 1e-4 {
		t.Fatalf("entity moved %+v, want (0.4, 0, 0.25)", p)
	}
	bp := bone.Position()
	if bp.X != 1 || bp.Y != 2 || bp.Z != 3 {
		t.Fatalf("bone was not planted: %+v", bp)
	}
	bone.SetPosition(1.4, 9, 3.25)
	e.anim.applyRootMotion(e)
	p = root.Position()
	if math.Abs(float64(p.X-0.4)) > 1e-4 {
		t.Fatalf("planted pose was applied again: %+v", p)
	}
}

func TestFBXClipSamplesASCIICurve(t *testing.T) {
	src := `
Model: 10, "Model::Hips", "LimbNode" {
}
AnimationCurve: 20, "AnimCurve::" {
 KeyTime: *2 { a: 0,46186158000 }
 KeyValueFloat: *2 { a: 0,4 }
}
`
	clips := parseFBXClips(src)
	if len(clips) != 1 || len(clips[0].curves) != 1 {
		t.Fatalf("clips %+v", clips)
	}
	cv := clips[0].curves[0]
	if cv.bone == "" {
		t.Fatal("curve has no bone")
	}
	got := cv.sample(0.5)
	if math.Abs(float64(got-2)) > 0.01 {
		t.Fatalf("sample = %v, bone %q", got, cv.bone)
	}
}

func TestNavCostAndLinks(t *testing.T) {
	w := New(t.TempDir())
	w.navs[1] = &navMesh{}
	v, err := w.Call("setnavcost", []value.Value{value.Num(1), value.Num(3), value.Num(0.2)})
	if err != nil || v.Number() != 1 {
		t.Fatalf("cost %v %v", v, err)
	}
	_, err = w.Call("addnavlink", []value.Value{
		value.Num(1), value.Num(0), value.Num(0), value.Num(0),
		value.Num(0), value.Num(5), value.Num(0),
	})
	if err != nil || len(w.navs[1].links) != 1 {
		t.Fatal(err)
	}
	path := stitchNavLinks([][3]float32{{0, 0, 0}, {10, 0, 0}}, w.navs[1].links)
	if len(path) != 3 || path[1][1] != 5 {
		t.Fatalf("stitched %+v", path)
	}
}

func TestVisionConeAndSeek(t *testing.T) {
	w := New(t.TempDir())
	w.delta = 1
	from := placeEnt(w, 0, 0, 0, 0)
	ahead := placeEnt(w, 0, 0, 5, 0)
	side := placeEnt(w, 5, 0, 0, 0)
	if !w.canSee(from, ahead) || w.canSee(from, side) {
		t.Fatalf("ahead %v side %v", w.canSee(from, ahead), w.canSee(from, side))
	}
	if !w.canHear(from, ahead, 1) {
		t.Fatal("target inside the hearing radius")
	}
	if _, err := w.Call("steerseek", []value.Value{
		value.Num(float64(from)), value.Num(0), value.Num(0), value.Num(10), value.Num(4),
	}); err != nil {
		t.Fatal(err)
	}
	_, _, z, ok := w.entXYZ(from)
	if !ok || math.Abs(z-4) > 0.05 {
		t.Fatalf("seek z=%v ok=%v", z, ok)
	}
}

func TestEmitterPresetAndAssets(t *testing.T) {
	w := New(t.TempDir())
	w.emitters = map[int]*emitter{1: {emitting: true}}
	if _, err := w.Call("emitterpreset", []value.Value{value.Num(1), value.Str("sparks")}); err != nil {
		t.Fatal(err)
	}
	e := w.emitters[1]
	if e.rate != 40 || e.gravY != -12 {
		t.Fatalf("preset rate=%v grav=%v", e.rate, e.gravY)
	}
	w.trackAsset("mesh cube.obj", 2048)
	v, err := w.Call("statsassetkb", []value.Value{value.Str("mesh cube.obj")})
	if err != nil || math.Abs(v.Number()-2) > 0.01 {
		t.Fatalf("asset kb %v %v", v, err)
	}
	theme, err := w.Call("uitheme", []value.Value{value.Str("Light")})
	if err != nil || theme.String() != "light" || w.uiTheme != "light" {
		t.Fatalf("theme %v %v", theme, err)
	}
	if _, err := w.Call("guianchor", []value.Value{value.Num(0.5), value.Num(1)}); err != nil || !w.uiAnchor {
		t.Fatal(err)
	}
}

func TestFuncProfilerCommand(t *testing.T) {
	w := New(t.TempDir())
	prog, err := parse.Parse(`
Function Work()
  x = 0
  For i = 1 To 200
    x = x + i
  Next
End Function
Work()
`)
	if err != nil {
		t.Fatal(err)
	}
	in := interp.New(prog, w)
	w.Bind(in)
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	calls, err := w.Call("statsfunccalls", []value.Value{value.Str("work")})
	if err != nil || calls.Number() < 1 {
		t.Fatalf("calls %v %v stats %+v", calls, err, in.FuncStats())
	}
	top, err := w.Call("statstopfunc", nil)
	if err != nil || top.String() == "" {
		t.Fatalf("top %v %v", top, err)
	}
}
