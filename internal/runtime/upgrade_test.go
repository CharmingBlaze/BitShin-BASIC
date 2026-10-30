package runtime

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func TestFilmGradeCommands(t *testing.T) {
	w := New(".")
	if _, err := w.Call("setfilmgrain", []value.Value{value.Num(0.2)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setvignette", []value.Value{value.Num(0.4)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setchromaticaberration", []value.Value{value.Num(0.15)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setsharpen", []value.Value{value.Num(0.6)}); err != nil {
		t.Fatal(err)
	}
	if w.post.grain != 0.2 || w.post.vignette != 0.4 || w.post.chroma != 0.15 || w.post.sharpen != 0.6 {
		t.Fatalf("film %v %v %v %v", w.post.grain, w.post.vignette, w.post.chroma, w.post.sharpen)
	}
	if !w.post.on {
		t.Fatal("film grade should turn the post stack on")
	}
}

func TestIKChainReaches(t *testing.T) {
	w := New(".")
	a := core.NewNode()
	a.SetName("a")
	b := core.NewNode()
	b.SetName("b")
	b.SetPosition(0, 1, 0)
	c := core.NewNode()
	c.SetName("c")
	c.SetPosition(0, 1, 0)
	d := core.NewNode()
	d.SetName("d")
	d.SetPosition(0, 1, 0)
	c.Add(d)
	b.Add(c)
	a.Add(b)
	id := w.addEntity(&Entity{node: a}, 0)
	_, err := w.Call("setikchain", []value.Value{
		value.Num(float64(id)), value.Str("a,b,c,d"), value.Num(2), value.Num(1), value.Num(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	w.refreshWorldMatrices()
	var p math32.Vector3
	d.WorldPosition(&p)
	if math.Hypot(float64(p.X-2), float64(p.Y-1)) > 0.35 {
		t.Fatalf("tip %v %v", p.X, p.Y)
	}
}

func TestGOAPPlansEat(t *testing.T) {
	w := New(".")
	id, err := w.Call("createplanner", nil)
	if err != nil {
		t.Fatal(err)
	}
	p := value.Num(id.Number())
	_, err = w.Call("addplanaction", []value.Value{p, value.Str("eat"), value.Num(1), value.Str("hungry"), value.Str("fed")})
	if err != nil {
		t.Fatal(err)
	}
	name, err := w.Call("plan", []value.Value{p, value.Str("hungry"), value.Str("fed")})
	if err != nil {
		t.Fatal(err)
	}
	if name.String() != "eat" {
		t.Fatalf("plan %q", name.String())
	}
}

func TestExportGLTFWritesMesh(t *testing.T) {
	dir := t.TempDir()
	w := New(dir)
	id, err := w.Call("createperson", []value.Value{value.Str("Mara"), value.Num(0), value.Num(0), value.Num(0)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Call("exportgltf", []value.Value{value.Str("mara.gltf"), id})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "mara.gltf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"meshes"`) || !strings.Contains(string(b), "base64") {
		t.Fatal("gltf missing mesh buffer")
	}
}

func TestFFTWaterMoves(t *testing.T) {
	w := New(".")
	wb := &waterBody{y: 1, nWaves: 0, fftOn: true, time: 0.2}
	w.waters[1] = wb
	w.curWater = 1
	h0 := w.waterHeight(3, 4)
	wb.time = 2.5
	if wb.fft != nil {
		wb.fft.time = -1
	}
	h1 := w.waterHeight(3, 4)
	if h0 == h1 {
		t.Fatalf("fft height stuck at %v", h0)
	}
}

func TestSimplifyMeshKeepsFewerTriangles(t *testing.T) {
	g := geometry.NewGeometry()
	const n = 12
	pos := math32.NewArrayF32(0, n*n*3)
	for z := 0; z < n; z++ {
		for x := 0; x < n; x++ {
			pos.Append(float32(x), 0, float32(z))
		}
	}
	idx := math32.NewArrayU32(0, (n-1)*(n-1)*6)
	for z := 0; z < n-1; z++ {
		for x := 0; x < n-1; x++ {
			a := uint32(z*n + x)
			idx.Append(a, a+1, a+uint32(n), a+1, a+1+uint32(n), a+uint32(n))
		}
	}
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	before := len(g.Indices()) / 3
	got := simplifyGeometry(g, 0.4)
	if got <= 0 || got >= before {
		t.Fatalf("simplify %d -> %d", before, got)
	}
	if len(*g.VBO(gls.VertexPosition).Buffer()) < 9 {
		t.Fatal("simplify dropped the positions")
	}
}

func TestPeopleDiffer(t *testing.T) {
	ha, _, ra, ga, ba := personLook("Mara")
	hb, _, rb, gb, bb := personLook("Bram")
	if ha == hb && ra == rb && ga == gb && ba == bb {
		t.Fatal("names produced the same person")
	}
}

// SetFFTStrength scales the chop WaterHeight reports, and 0 flattens it.
func TestFFTStrengthScalesHeight(t *testing.T) {
	w := New(".")
	wb := &waterBody{y: 1, nWaves: 0, fftOn: true, time: 0.7}
	w.waters[1] = wb
	w.curWater = 1
	base := w.waterHeight(3, 4) - 1
	if base == 0 {
		t.Fatal("no fft chop at this point")
	}
	wb.fftAmp = 3
	if got := w.waterHeight(3, 4) - 1; math32.Abs(got-3*base) > 1e-4 {
		t.Fatalf("strength 3: %v, want %v", got, 3*base)
	}
	wb.fftAmp = -1
	if got := w.waterHeight(3, 4); got != 1 {
		t.Fatalf("strength 0 should be flat, got %v", got)
	}
	if wb.fft.n != waterFFTN {
		t.Fatalf("tile is %d, want %d", wb.fft.n, waterFFTN)
	}
}
