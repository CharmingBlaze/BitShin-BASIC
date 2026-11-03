package runtime

import (
	"bytes"
	"strings"
	"testing"

	"moonbasic/internal/interp"
	"moonbasic/internal/parse"
	"moonbasic/internal/value"
)

func TestSightBlockedByRadius(t *testing.T) {
	w := New(t.TempDir())
	from := placeEnt(w, 0, 0, 0, 0)
	ahead := placeEnt(w, 0, 0, 5, 0)
	if !w.canSee(from, ahead) {
		t.Fatal("open segment")
	}
	placeEnt(w, 0, 0, 2.5, 0)
	if w.canSee(from, ahead) {
		t.Fatal("entity on the segment should block sight")
	}
}

func TestAnimMarkAndGraphTransition(t *testing.T) {
	if !markHit(0, 0.5, 0.25, false) || markHit(0, 0.2, 0.25, false) {
		t.Fatal("mark crossing")
	}
	w := New(t.TempDir())
	prog, err := parse.Parse(`
Function Ping(t)
  Print 1
End Function
Function Ready()
  Return 1
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
	id := placeEnt(w, 0, 0, 0, 0)
	if _, err := w.Call("animevent", []value.Value{value.Num(float64(id)), value.Num(0.25), value.Str("Ping")}); err != nil {
		t.Fatal(err)
	}
	w.fireAnimMarks(w.ents[id].anim, 0, 0.5, false)
	if !strings.Contains(buf.String(), "1") {
		t.Fatalf("event output %q", buf.String())
	}
	st := w.extState()
	st.graphs[1] = &animGraph{states: map[string]int{"idle": 0, "go": 1}, cur: "idle"}
	if _, err := w.Call("addanimtransition", []value.Value{
		value.Num(1), value.Str("idle"), value.Str("go"), value.Str("Ready"),
	}); err != nil {
		t.Fatal(err)
	}
	w.stepAnimGraphs()
	if st.graphs[1].cur != "go" {
		t.Fatalf("state %q", st.graphs[1].cur)
	}
}

func TestExportClipIsGLTF(t *testing.T) {
	b, err := (fbxClip{
		name: "walk",
		curves: []fbxCurve{{
			bone: "Hips", axis: "x",
			times: []float32{0, 1}, vals: []float32{0, 2},
		}},
	}).glTFDocument()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"version": "2.0"`, "base64,", `"path": "translation"`, `"type": "VEC3"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
}

func TestNineSliceAndVRAM(t *testing.T) {
	w := New(t.TempDir())
	w.ready = true
	w.images[1] = &ebiImage{w: 30, h: 30}
	if _, err := w.Call("drawnineslice", []value.Value{
		value.Num(1), value.Num(0), value.Num(0), value.Num(90), value.Num(60), value.Num(8),
	}); err != nil {
		t.Fatal(err)
	}
	if len(w.draws) != 9 {
		t.Fatalf("slices %d", len(w.draws))
	}
	if w.draws[0].sw != 8 || w.draws[4].sw != 14 {
		t.Fatalf("src widths corner %d center %d", w.draws[0].sw, w.draws[4].sw)
	}
	w.noteVRAM(256, 256, 0)
	v, err := w.Call("statsvrammb", nil)
	if err != nil || v.Number() < 0.2 || v.Number() > 0.3 {
		t.Fatalf("vram %v %v", v, err)
	}
}
