package runtime

import (
	"testing"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/math32"
)

// mergeWorld is two different meshes, side by side and still long enough to merge.
func mergeWorld(t *testing.T) (*World, int, int) {
	t.Helper()
	w := batchWorld()
	a := w.meshEnt(geometry.NewCube(1), 0)
	b := w.meshEnt(geometry.NewCube(2), 0)
	w.ents[b].mesh.SetPosition(5, 0, 0)
	settleMerges(w)
	return w, a, b
}

func settleMerges(w *World) {
	for i := 0; i < batchMergeFrames+3; i++ {
		w.updateBatches()
	}
}

func TestMergeBakesStillMeshes(t *testing.T) {
	w, a, b := mergeWorld(t)
	if w.mergedDrawCount() != 1 {
		t.Fatalf("two still meshes of one material: want one merged mesh, got %d", w.mergedDrawCount())
	}
	if w.ents[a].mesh.Renderable() || w.ents[b].mesh.Renderable() {
		t.Fatal("merged meshes must not also draw on their own")
	}
	if w.visibleMeshCount() != 1 {
		t.Fatalf("draws: got %d", w.visibleMeshCount())
	}
	var g *mergeGroup
	for _, m := range w.batch.merges {
		g = m
	}
	box := g.geom.BoundingBox()
	// The second cube is moved 5 along X and is 2 across: it reaches 6.
	if box.Max.X < 5.9 || box.Max.X > 6.1 || box.Min.X > -0.4 {
		t.Fatalf("baked in the wrong place: %v", box)
	}
}

func TestMergeReleasesOnMove(t *testing.T) {
	w, a, b := mergeWorld(t)
	w.ents[b].mesh.SetPosition(9, 0, 0)
	w.updateBatches()
	if !w.ents[b].mesh.Renderable() || batchDrawn(w.ents[b]) {
		t.Fatal("a moved mesh must draw on its own at once")
	}
	if !w.ents[a].mesh.Renderable() || w.mergedDrawCount() != 0 {
		t.Fatal("the one left has nothing to merge with")
	}
	settleMerges(w)
	if w.mergedDrawCount() != 1 {
		t.Fatal("still again: it should merge again")
	}
}

func TestMergeReleasesOnColorAndHide(t *testing.T) {
	w, a, b := mergeWorld(t)
	w.ents[a].mat.SetColor(&math32.Color{R: 1})
	w.updateBatches()
	if !w.ents[a].mesh.Renderable() {
		t.Fatal("a recolored mesh must not keep its old baked color")
	}
	w, a, _ = mergeWorld(t)
	w.ents[a].mesh.SetVisible(false)
	w.updateBatches()
	if w.mergedDrawCount() != 0 {
		t.Fatal("a hidden mesh must leave the merged one, which would keep drawing it")
	}
	_ = b
}

func TestMergeSkipsWhatCanChange(t *testing.T) {
	w := batchWorld()
	a := w.meshEnt(geometry.NewCube(1), 0)
	b := w.meshEnt(geometry.NewCube(2), 0)
	w.ents[b].mesh.SetPosition(5, 0, 0)
	w.ents[a].shadowGeomDirty = true
	settleMerges(w)
	if w.mergedDrawCount() != 0 || !w.ents[a].mesh.Renderable() {
		t.Fatal("a mesh with edited vertices must not be baked")
	}
	w = batchWorld()
	a = w.meshEnt(geometry.NewCube(1), 0)
	b = w.meshEnt(geometry.NewCube(2), 0)
	w.ents[a].mesh.SetScale(-1, 1, 1)
	settleMerges(w)
	if w.mergedDrawCount() != 0 {
		t.Fatal("a mirrored mesh must not be baked: its triangles would turn inside out")
	}
	_ = b
}

func TestMergeKeepsAreasApart(t *testing.T) {
	w := batchWorld()
	for _, x := range []float32{0, 5, batchMergeCell * 3, batchMergeCell*3 + 5} {
		id := w.meshEnt(geometry.NewCube(1+x/1000), 0)
		w.ents[id].mesh.SetPosition(x, 0, 0)
	}
	settleMerges(w)
	if w.mergedDrawCount() != 2 {
		t.Fatalf("two areas, want two merged meshes, got %d", w.mergedDrawCount())
	}
}

func TestCopiesLeaveMergesForInstancing(t *testing.T) {
	w := batchWorld()
	geom := geometry.NewCube(1)
	a := w.meshEnt(geom, 0)
	b := w.meshEnt(geometry.NewCube(2), 0)
	settleMerges(w)
	if w.mergedDrawCount() != 1 {
		t.Fatal("setup: should be merged")
	}
	c := w.meshEnt(geom, 0)
	w.updateBatches()
	if w.ents[a].batch.merged != nil || !batchDrawn(w.ents[c]) || len(w.batch.active) != 1 {
		t.Fatal("a mesh that gains a copy is instanced instead")
	}
	if w.mergedDrawCount() != 0 || !w.ents[b].mesh.Renderable() {
		t.Fatal("its old partner goes back to drawing alone")
	}
}

func TestPickLOD(t *testing.T) {
	levels := []lodLevel{{err: 0.1}, {err: 0.5}}
	for _, c := range []struct {
		scale, unitsToPixels float32
		want                 int
	}{
		{1, 100, 0}, // close: 0.1 units is 10 pixels
		{1, 5, 1},   // the first version's error is half a pixel, the second's 2.5
		{1, 1, 2},   // far: both fit
		{20, 1, 0},  // the same distance, scaled up 20 times
	} {
		if got := pickLOD(levels, c.scale, c.unitsToPixels); got != c.want {
			t.Errorf("scale %v, %v px/unit: level %d, want %d", c.scale, c.unitsToPixels, got, c.want)
		}
	}
}

// The simpler versions really are simpler, keep their attributes, and arrive
// only once the build is done.
func TestLODBuildsSimplerVersions(t *testing.T) {
	w := batchWorld()
	geom := geometry.NewSphere(1, 32, 16)
	if w.lodFor(geom) != nil {
		t.Fatal("the build takes time: nothing yet")
	}
	w.adoptLOD(<-w.batch.lodDone)
	l := w.lodFor(geom)
	if l == nil || len(l.levels) == 0 {
		t.Fatal("no simpler versions of a sphere")
	}
	last := geomTriangles(geom)
	for i, lev := range l.levels {
		n := geomTriangles(lev.geom)
		if n >= last || n == 0 {
			t.Fatalf("level %d has %d triangles after %d", i, n, last)
		}
		if len(lev.geom.VBOs()) != len(geom.VBOs()) {
			t.Fatalf("level %d lost vertex attributes", i)
		}
		last = n
	}
	small := geometry.NewCube(1)
	if w.lodFor(small) != nil || w.lodFor(small) != nil {
		t.Fatal("a cube is too small to need versions")
	}
}
