package runtime

import (
	"bytes"
	"strings"
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gui"

	"moonbasic/internal/interp"
	"moonbasic/internal/parse"
	"moonbasic/internal/value"
)

func TestBonePosRecordAndLook(t *testing.T) {
	w := New(t.TempDir())
	root := core.NewNode()
	bone := core.NewNode()
	bone.SetName("hips")
	root.Add(bone)
	id := w.addEntity(&Entity{kind: "rig", node: root}, 0)
	if _, err := w.Call("setbonepos", []value.Value{
		value.Num(float64(id)), value.Str("hips"), value.Num(2), value.Num(0), value.Num(4),
	}); err != nil {
		t.Fatal(err)
	}
	x, err := w.Call("getbonex", []value.Value{value.Num(float64(id)), value.Str("hips")})
	z, err2 := w.Call("getbonez", []value.Value{value.Num(float64(id)), value.Str("hips")})
	if err != nil || err2 != nil || x.Number() != 2 || z.Number() != 4 {
		t.Fatalf("bone %v %v %v %v", x, z, err, err2)
	}
	if _, err := w.Call("bonelookat", []value.Value{
		value.Num(float64(id)), value.Str("hips"), value.Num(0), value.Num(0), value.Num(5),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("animrecord", []value.Value{value.Num(float64(id)), value.Num(1)}); err != nil {
		t.Fatal(err)
	}
	bone.SetPosition(1, 0, 0)
	w.sampleAnim(w.ents[id])
	bone.SetPosition(3, 0, 0)
	w.sampleAnim(w.ents[id])
	n, err := w.Call("animrecord", []value.Value{value.Num(float64(id)), value.Num(0)})
	if err != nil || n.Number() != 3 {
		t.Fatalf("curves %v %v", n, err)
	}
	var xs []float32
	for _, cv := range w.ents[id].anim.fbx[0].curves {
		if cv.axis == "x" {
			xs = cv.vals
		}
	}
	if len(xs) != 2 || xs[0] != 1 || xs[1] != 3 {
		t.Fatalf("recorded x %v", xs)
	}
}

func TestSpriteEventTrailAndAI(t *testing.T) {
	w := New(t.TempDir())
	w.delta = 1
	prog, err := parse.Parse(`
Function OnFrame(f)
  Print f
End Function
`)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	in := interp.New(prog, w)
	in.Out = &buf
	w.Bind(in)
	if err := in.Run(); err != nil {
		t.Fatal(err)
	}
	w.sprites[1] = &ebiSprite{id: 1, animFrom: 0, animCount: 2, animFPS: 1, frame: 0}
	if _, err := w.Call("spriteanimevent", []value.Value{value.Num(1), value.Num(1), value.Str("OnFrame")}); err != nil {
		t.Fatal(err)
	}
	w.tickSpriteAnims()
	if w.sprites[1].frame != 1 || !strings.Contains(buf.String(), "1") {
		t.Fatalf("frame %d out %q", w.sprites[1].frame, buf.String())
	}
	ent := placeEnt(w, 0, 0, 0, 0)
	tr, err := w.Call("createtrail", []value.Value{value.Num(float64(ent)), value.Num(4)})
	if err != nil || tr.Number() == 0 {
		t.Fatal(err)
	}
	w.tickTrails()
	gx, gy, gz := toG3N(0, 0, 2)
	w.ents[ent].node.GetNode().SetPosition(gx, gy, gz)
	w.tickTrails()
	if len(w.trails[int(tr.Number())].pts) != 2 {
		t.Fatalf("trail %+v", w.trails[int(tr.Number())].pts)
	}
	ai, err := w.Call("createai", []value.Value{value.Num(float64(ent))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("aistate", []value.Value{value.Num(ai.Number()), value.Str("patrol")}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("aiaddwaypoint", []value.Value{value.Num(ai.Number()), value.Num(0), value.Num(0), value.Num(4)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("aiupdate", []value.Value{value.Num(ai.Number()), value.Num(4)}); err != nil {
		t.Fatal(err)
	}
	_, _, z, ok := w.entXYZ(ent)
	if !ok || z < 3.5 {
		t.Fatalf("ai z %v", z)
	}
	other := placeEnt(w, 0, 0, 6, 0)
	seen, err := w.Call("aidetecttarget", []value.Value{value.Num(ai.Number()), value.Num(float64(other))})
	if err != nil || seen.Number() != 1 {
		t.Fatalf("detect %v %v", seen, err)
	}
}

func TestMorphRichFlexAndBinaryCurve(t *testing.T) {
	w := New(t.TempDir())
	id := placeEnt(w, 0, 0, 0, 0)
	if _, err := w.Call("setmorph", []value.Value{value.Num(float64(id)), value.Num(1), value.Num(0.25)}); err != nil {
		t.Fatal(err)
	}
	got, err := w.Call("getmorph", []value.Value{value.Num(float64(id)), value.Num(1)})
	if err != nil || got.Number() != 0.25 {
		t.Fatalf("morph %v %v", got, err)
	}
	if _, err := w.Call("setviseme", []value.Value{value.Num(float64(id)), value.Str("ee"), value.Num(1)}); err != nil {
		t.Fatal(err)
	}
	ee, _ := w.Call("getmorph", []value.Value{value.Num(float64(id)), value.Num(1)})
	if ee.Number() != 1 {
		t.Fatalf("viseme %v", ee.Number())
	}
	w.ents[id].anim = &animState{fbx: []fbxClip{{curves: []fbxCurve{{bone: "Hips", axis: "x"}}}}}
	n, err := w.Call("animretarget", []value.Value{value.Num(float64(id)), value.Str("Hips=pelvis")})
	if err != nil || n.Number() != 1 || w.ents[id].anim.fbx[0].curves[0].bone != "pelvis" {
		t.Fatalf("retarget %v %v", n, err)
	}
	spans := parseRich("go {#ff8000}fast{#} now")
	if len(spans) != 3 || !spans[1].colored || spans[1].text != "fast" || spans[2].colored {
		t.Fatalf("rich %+v", spans)
	}
	if _, err := w.Call("guihbox", []value.Value{value.Num(2), value.Num(4)}); err != nil || w.uiHLeft != 2 {
		t.Fatal(err)
	}
	w.hboxAdvance()
	w.hboxAdvance()
	if w.uiHLeft != 0 {
		t.Fatalf("hbox left %d", w.uiHLeft)
	}
	var mesh fbxMesh
	mesh.noteLimb([]fbxProp{{s: "Model::Hips"}, {s: "LimbNode"}})
	mesh.curveOpen = true
	noteFBXNode("KeyTime", []fbxProp{{df: []float64{0, fbxTicksPerSecond}}}, &mesh)
	noteFBXNode("KeyValueFloat", []fbxProp{{df: []float64{1, 5}}}, &mesh)
	mesh.flushCurve()
	if len(mesh.clips) != 1 || mesh.clips[0].curves[0].bone != "Hips" || mesh.clips[0].curves[0].sample(1) != 5 {
		t.Fatalf("binary curve %+v", mesh.clips)
	}
	p := gui.NewPanel(40, 20)
	wid := w.addEntity(&Entity{kind: "panel", node: p}, 0)
	if _, err := w.Call("tweenwidget", []value.Value{value.Num(float64(wid)), value.Num(12), value.Num(8), value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	if p.Position().X != 12 || p.Position().Y != 8 {
		t.Fatalf("tween %+v", p.Position())
	}
	if _, err := w.Call("tweenwidget", []value.Value{value.Num(float64(wid)), value.Num(12), value.Num(28), value.Num(1)}); err != nil {
		t.Fatal(err)
	}
	w.delta = 0.5
	w.tickUITweens()
	if p.Position().Y != 18 {
		t.Fatalf("halfway y %v", p.Position().Y)
	}
}

func TestWidgetAnchor(t *testing.T) {
	w := New(t.TempDir())
	p := gui.NewPanel(100, 40)
	id := w.addEntity(&Entity{kind: "panel", node: p}, 0)
	if _, err := w.Call("setwidgetanchor", []value.Value{value.Num(float64(id)), value.Num(1), value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	if p.Position().X != 700 {
		t.Fatalf("anchor x %v", p.Position().X)
	}
}
