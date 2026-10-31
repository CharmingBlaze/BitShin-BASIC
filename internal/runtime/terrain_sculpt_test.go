package runtime

import (
	"image"
	"math"
	"testing"

	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func testLattice(gw, gd int, cell float32) (*sculptLattice, []float32) {
	h := make([]float32, gw*gd)
	return &sculptLattice{
		dx: cell, dz: cell, bounded: true, gw: gw, gd: gd,
		get: func(i, j int) float32 { return h[j*gw+i] },
		add: func(i, j int, d float32) { h[j*gw+i] += d },
	}, h
}

func near(a, b, eps float32) bool { return float32(math.Abs(float64(a-b))) <= eps }

func TestBrushWeightFalloff(t *testing.T) {
	if w := brushWeight(0, 4, 0.5); w != 1 {
		t.Fatalf("center weight %v", w)
	}
	if w := brushWeight(4, 4, 0.5); w != 0 {
		t.Fatalf("edge weight %v", w)
	}
	if w := brushWeight(5, 4, 0.5); w != 0 {
		t.Fatalf("outside weight %v", w)
	}
	if w := brushWeight(1.9, 4, 0.5); w != 1 {
		t.Fatalf("inner disc should be full, got %v", w)
	}
	if w := brushWeight(3, 4, 0.5); !near(w, 0.5, 1e-5) {
		t.Fatalf("smoothstep midpoint want 0.5 got %v", w)
	}
	if w := brushWeight(3.99, 4, 0); w != 1 {
		t.Fatalf("hard brush should be 1 up to the edge, got %v", w)
	}
	prev := float32(2)
	for d := float32(0); d <= 4; d += 0.25 {
		w := brushWeight(d, 4, 1)
		if w > prev || w < 0 || w > 1 {
			t.Fatalf("soft brush not monotonic at %v: %v after %v", d, w, prev)
		}
		prev = w
	}
	if w := brushWeight(0, 0, 0.5); w != 0 {
		t.Fatalf("zero radius %v", w)
	}
}

func TestSculptRaiseAndLower(t *testing.T) {
	l, h := testLattice(17, 17, 1)
	rc := l.applyBrush(brushRaise, 8, 8, 3, 2, 0, 0.5)
	if rc.empty() {
		t.Fatal("raise touched nothing")
	}
	if h[8*17+8] != 2 {
		t.Fatalf("center %v want 2", h[8*17+8])
	}
	if h[8*17+11] != 0 || h[0] != 0 {
		t.Fatalf("radius edge / far node changed: %v %v", h[8*17+11], h[0])
	}
	if h[8*17+10] <= 0 || h[8*17+10] >= 2 {
		t.Fatalf("falloff node %v", h[8*17+10])
	}
	l.applyBrush(brushRaise, 8, 8, 3, -2, 0, 0.5)
	for i, v := range h {
		if !near(v, 0, 1e-5) {
			t.Fatalf("lower did not undo raise at %d: %v", i, v)
		}
	}
}

func TestSculptBoundsClipping(t *testing.T) {
	l, h := testLattice(9, 9, 1)
	rc := l.applyBrush(brushRaise, 0, 0, 3, 1, 0, 0)
	if rc.i0 != 0 || rc.j0 != 0 || rc.i1 > 2 || rc.j1 > 2 {
		t.Fatalf("corner brush rect %+v", rc)
	}
	if h[0] != 1 {
		t.Fatalf("corner %v", h[0])
	}
	// Entirely outside: no change, empty rect, no panic.
	rc = l.applyBrush(brushRaise, -20, 40, 3, 1, 0, 0)
	if !rc.empty() {
		t.Fatalf("outside brush rect %+v", rc)
	}
	r := l.brushRect(4, 4, 100)
	if r.i0 != 0 || r.j0 != 0 || r.i1 != 8 || r.j1 != 8 {
		t.Fatalf("huge brush not clipped: %+v", r)
	}
	// Smoothing on the border reads clamped neighbours instead of out of range.
	l.applyBrush(brushSmooth, 8, 8, 2, 0, 1, 0.5)
}

func TestFlattenTerrainMath(t *testing.T) {
	l, h := testLattice(17, 17, 1)
	for i := range h {
		h[i] = float32(i%17) * 0.5 // slope along x
	}
	l.applyBrush(brushFlatten, 8, 8, 4, 3, 1, 0)
	for j := 6; j <= 10; j++ {
		for i := 6; i <= 10; i++ {
			if !near(h[j*17+i], 3, 1e-5) {
				t.Fatalf("full-strength flatten node %d,%d = %v", i, j, h[j*17+i])
			}
		}
	}
	l2, h2 := testLattice(5, 5, 1)
	h2[12] = 10
	l2.applyBrush(brushFlatten, 2, 2, 1, 0, 0.5, 0)
	if !near(h2[12], 5, 1e-5) {
		t.Fatalf("half-strength flatten want 5 got %v", h2[12])
	}
}

func TestSmoothTerrainMath(t *testing.T) {
	l, h := testLattice(9, 9, 1)
	h[4*9+4] = 9
	l.applyBrush(brushSmooth, 4, 4, 2.5, 0, 1, 0)
	if !near(h[4*9+4], 1, 1e-5) {
		t.Fatalf("spike should average to 1, got %v", h[4*9+4])
	}
	if !near(h[4*9+5], 1, 1e-5) {
		t.Fatalf("neighbour should read the snapshot spike (1), got %v", h[4*9+5])
	}
	// A plane is a fixed point of the 3x3 average.
	l2, h2 := testLattice(9, 9, 1)
	for j := 0; j < 9; j++ {
		for i := 0; i < 9; i++ {
			h2[j*9+i] = float32(i) + 2*float32(j)
		}
	}
	l2.applyBrush(brushSmooth, 4, 4, 2.5, 0, 1, 0)
	for j := 2; j <= 6; j++ {
		for i := 2; i <= 6; i++ {
			if !near(h2[j*9+i], float32(i)+2*float32(j), 1e-4) {
				t.Fatalf("plane changed at %d,%d: %v", i, j, h2[j*9+i])
			}
		}
	}
}

func TestSculptOverlaySparse(t *testing.T) {
	o := newSculptOverlay(0, 0, 0.5, 0.5)
	if o.sample(3, 3) != 0 {
		t.Fatal("empty overlay")
	}
	o.add(-1, -1, 2)
	o.add(0, -1, 4)
	if o.get(-1, -1) != 2 || o.get(0, -1) != 4 || o.get(-33, -1) != 0 {
		t.Fatalf("negative tile index: %v %v", o.get(-1, -1), o.get(0, -1))
	}
	if v := o.sample(-0.25, -0.5); !near(v, 3, 1e-5) {
		t.Fatalf("bilinear midpoint want 3 got %v", v)
	}
	if v := o.sample(100, 100); v != 0 {
		t.Fatalf("far sample %v", v)
	}
	if floorDiv(-1, 32) != -1 || floorDiv(-32, 32) != -1 || floorDiv(-33, 32) != -2 || floorDiv(31, 32) != 0 {
		t.Fatal("floorDiv")
	}
}

func TestPaintImageChannelsAndBorder(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	r := paintImage(img, 8, 8, 3, 2, 1, 0) // rock = B
	if r.Empty() {
		t.Fatal("paint touched nothing")
	}
	o := img.PixOffset(8, 8)
	if img.Pix[o] != 0 || img.Pix[o+1] != 0 || img.Pix[o+2] != 255 || img.Pix[o+3] != 0 {
		t.Fatalf("rock texel %v", img.Pix[o:o+4])
	}
	// Painting at the corner never writes the clamped outer ring.
	paintImage(img, 0, 0, 4, 1, 1, 0)
	if img.Pix[img.PixOffset(0, 0)+1] != 0 || img.Pix[img.PixOffset(3, 0)+1] != 0 {
		t.Fatal("outer ring painted")
	}
	if img.Pix[img.PixOffset(1, 1)+1] != 255 {
		t.Fatal("inner corner not painted")
	}
	// Weak strokes still move a texel each call.
	before := img.Pix[o+1]
	paintImage(img, 8, 8, 3, 1, 0.001, 0)
	if img.Pix[o+1] == before {
		t.Fatal("weak stroke stalled")
	}
	paintImage(img, 8, 8, 3, -1, 1, 0)
	if img.Pix[o] != 0 || img.Pix[o+1] != 0 || img.Pix[o+2] != 0 || img.Pix[o+3] != 0 {
		t.Fatalf("erase left %v", img.Pix[o:o+4])
	}
}

func TestSculptTerrainCommandsHeightmap(t *testing.T) {
	w := New(".")
	gw := 33
	hf := heightField{gw: gw, gd: gw, worldW: 64, worldD: 64, hscale: 10, ox: -32, oz: -32, h: make([]float32, gw*gw)}
	id := w.addTerrain(hf, 17, 1)
	tt := w.terrains[id]
	k := chunkOf(0, 0, tt.chunkWorld())
	chunk := tt.ents[k]
	if chunk == 0 {
		t.Fatal("origin chunk not seeded")
	}
	maxY := func() float32 {
		m := float32(-1e9)
		w.ents[chunk].mesh.GetGeometry().VBO(gls.VertexPosition).ReadVectors3(gls.VertexPosition, func(v math32.Vector3) bool {
			if v.Y > m {
				m = v.Y
			}
			return false
		})
		return m
	}
	if maxY() > 0.01 {
		t.Fatal("flat terrain mesh not flat")
	}
	call := func(name string, args ...float64) value.Value {
		t.Helper()
		a := make([]value.Value, len(args))
		for i, v := range args {
			a[i] = value.Num(v)
		}
		r, err := w.Call(name, a)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return r
	}
	call("sculptterrain", float64(id), 2, 2, 6, 3)
	if y := w.terrainHeight(2, 2); !near(y, 3, 0.01) {
		t.Fatalf("TerrainHeight after sculpt %v", y)
	}
	w.flushTerrainSculpt()
	if tt.ents[k] != chunk {
		t.Fatal("rebuild should keep the chunk entity")
	}
	if y := maxY(); y < 2.5 {
		t.Fatalf("chunk mesh not rebuilt, max y %v", y)
	}
	call("flattenterrain", float64(id), 2, 2, 8, 1, 1)
	if y := w.terrainHeight(2, 2); !near(y, 1, 0.01) {
		t.Fatalf("flatten %v", y)
	}
	call("setterrainheight", float64(id), 0, 0, 7)
	if y := call("terraingridheight", float64(id), 0, 0).Num; y != 7 {
		t.Fatalf("grid height %v", y)
	}
	if y := w.terrainHeight(-32, -32); !near(y, 7, 0.01) {
		t.Fatalf("grid corner world height %v", y)
	}
	if _, err := w.Call("setterrainheight", []value.Value{value.Num(float64(id)), value.Num(99), value.Num(0), value.Num(1)}); err == nil {
		t.Fatal("out-of-grid SetTerrainHeight should fail")
	}
	call("paintterrain", float64(id), 0, 0, 5, 1)
	if tt.sculpt.paint == nil || tt.blendTex != tt.sculpt.paint.tex || tt.sculpt.paint.dirty.Empty() {
		t.Fatal("paint map not created / dirty")
	}
	call("resetterrainedits", float64(id))
	if y := w.terrainHeight(2, 2); y != 0 {
		t.Fatalf("reset height %v", y)
	}
	if _, err := w.Call("sculptterrain", []value.Value{value.Num(999), value.Num(0), value.Num(0), value.Num(1), value.Num(1)}); err == nil {
		t.Fatal("unknown terrain should error")
	}
	if _, err := w.Call("sculptterrain", []value.Value{value.Num(float64(id)), value.Num(0)}); err == nil {
		t.Fatal("missing args should error")
	}
}

func TestSculptProcTerrainOverlay(t *testing.T) {
	w := New(".")
	hf := heightField{gw: 25, gd: 25, worldW: 40, worldD: 40, hscale: 16, proc: true, style: 1, octaves: 6, freq: 0.04, disp: 16, power: 1.25, seedX: 0.01, seedZ: 0.017}
	id := w.addTerrain(hf, 25, 1)
	base := w.terrainHeight(3, 4)
	if _, err := w.Call("sculptterrain", []value.Value{value.Num(float64(id)), value.Num(3), value.Num(4), value.Num(5), value.Num(-2), value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	cell := w.terrains[id].hf.edit.dx
	// Off-lattice the overlay is bilinear; a hard brush keeps the full delta near the centre.
	if y := w.terrainHeight(3, 4); !near(y-base, -2, 0.05) {
		t.Fatalf("crater depth %v (cell %v)", y-base, cell)
	}
	if y := w.terrainHeight(30, 30); !near(y, w.terrains[id].hf.baseAt(30, 30), 1e-5) {
		t.Fatal("far ground changed")
	}
	w.flushTerrainSculpt()
	if len(w.terrains[id].sculpt.sculptDirty) != 0 {
		t.Fatal("dirty set not drained")
	}
}

func TestPatchHeightMeshMatchesRebuild(t *testing.T) {
	bump := func(x, z float32) float32 {
		d := float32(math.Sqrt(float64((x-10)*(x-10) + (z-12)*(z-12))))
		return 2 * brushWeight(d, 3, 1)
	}
	flat := func(x, z float32) float32 { return 0.1 * x }
	bumped := func(x, z float32) float32 { return flat(x, z) + bump(x, z) }
	for _, c := range []struct {
		step  int
		morph float32
	}{{1, 0}, {2, 1}, {2, 0.375}, {4, 0}} {
		g := buildHeightMesh(flat, 0, 0, 32, 32, 40, c.step, c.morph)
		cell := float32(32) / 40
		grow := cell * float32(2*c.step+1)
		area := sculptArea{7 - cell - grow, 9 - cell - grow, 13 + cell + grow, 15 + cell + grow}
		if !patchHeightMesh(g, bumped, 0, 0, 32, 32, 40, c.step, c.morph, area) {
			t.Fatalf("step %d: patch refused a matching chunk", c.step)
		}
		want := buildHeightMesh(bumped, 0, 0, 32, 32, 40, c.step, c.morph)
		for _, at := range []gls.AttribType{gls.VertexPosition, gls.VertexNormal} {
			got, exp := *g.VBO(at).Buffer(), *want.VBO(at).Buffer()
			if len(got) != len(exp) {
				t.Fatalf("step %d: size %d vs %d", c.step, len(got), len(exp))
			}
			for i := range got {
				if !near(got[i], exp[i], 1e-4) {
					t.Fatalf("step %d attrib %v: [%d] patched %v rebuilt %v", c.step, at, i, got[i], exp[i])
				}
			}
		}
		if bb := g.BoundingBox(); bb.Max.Y < 1.5 {
			t.Fatalf("step %d: bounding box not refreshed (max y %v)", c.step, bb.Max.Y)
		}
	}
	// A chunk built with other parameters is refused (caller rebuilds).
	g := buildHeightMesh(flat, 0, 0, 32, 32, 20, 1, 0)
	if patchHeightMesh(g, bumped, 0, 0, 32, 32, 40, 1, 0, sculptArea{0, 0, 32, 32}) {
		t.Fatal("mismatched layout accepted")
	}
}

func TestSculptCompactHeightmap(t *testing.T) {
	w := New(".")
	gw := 33
	hf := heightField{gw: gw, gd: gw, worldW: 64, worldD: 64, hscale: 10, ox: -32, oz: -32, h: make([]float32, gw*gw)}
	for i := range hf.h {
		hf.h[i] = float32(i%gw) * 0.1
	}
	hf.compact() // 16-bit storage, as maps over 2048² load
	id := w.addTerrain(hf, 17, 1)
	before := w.terrainHeight(2, 2)
	a := []value.Value{value.Num(float64(id)), value.Num(2), value.Num(2), value.Num(6), value.Num(50), value.Num(0)}
	if _, err := w.Call("sculptterrain", a); err != nil {
		t.Fatal(err)
	}
	// Raised far past the stored 16-bit range: the overlay carries it.
	if y := w.terrainHeight(2, 2); !near(y-before, 50, 0.05) {
		t.Fatalf("compact map raise %v", y-before)
	}
	if _, err := w.Call("setterrainheight", []value.Value{value.Num(float64(id)), value.Num(0), value.Num(0), value.Num(-4)}); err != nil {
		t.Fatal(err)
	}
	if y := w.terrainHeight(-32, -32); !near(y, -4, 0.01) {
		t.Fatalf("compact grid corner %v", y)
	}
	w.flushTerrainSculpt()
	if _, err := w.Call("resetterrainedits", []value.Value{value.Num(float64(id))}); err != nil {
		t.Fatal(err)
	}
	if y := w.terrainHeight(2, 2); !near(y, before, 1e-4) {
		t.Fatalf("reset %v want %v", y, before)
	}
}
