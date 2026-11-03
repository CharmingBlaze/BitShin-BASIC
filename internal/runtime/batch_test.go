package runtime

import (
	"strings"
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func batchWorld() *World {
	w := New(".")
	w.scene = core.NewNode()
	w.ready = true
	w.batch.envRead = true
	return w
}

func TestBatchLayoutMatchesG3N(t *testing.T) {
	if !batchLayout.ok {
		t.Fatal("G3N's material fields moved: every mesh would be drawn alone")
	}
	m := material.NewStandard(&math32.Color{0.25, 0.5, 0.75})
	m.SetShininess(33)
	m.SetOpacity(0.5)
	lm := newLitMat(nil, m)
	w := batchWorld()
	id := w.meshEnt(geometry.NewCube(1), 0)
	w.ents[id].mesh.SetMaterial(lm)
	k, ok := batchKeyOf(w.ents[id])
	if !ok {
		t.Fatal("a plain lit mesh should be eligible")
	}
	if k.uni[0] != 0.25 || k.uni[1] != 0.5 || k.uni[2] != 0.75 {
		t.Fatalf("ambient read as %v", k.uni[:3])
	}
	if k.uni[12] != 33 || k.uni[13] != 0.5 {
		t.Fatalf("shininess and opacity read as %v, %v", k.uni[12], k.uni[13])
	}
}

func TestBatchGroupsEqualCopies(t *testing.T) {
	w := batchWorld()
	geom := geometry.NewCube(1)
	a, b, c := w.meshEnt(geom, 0), w.meshEnt(geom, 0), w.meshEnt(geom, 0)
	other := w.meshEnt(geometry.NewCube(2), 0)
	w.updateBatches()
	if len(w.batch.active) != 1 || len(w.batch.active[0].members) != 3 {
		t.Fatalf("want one group of three, got %d groups", len(w.batch.active))
	}
	for _, id := range []int{a, b, c} {
		if w.ents[id].mesh.Renderable() {
			t.Fatalf("batched entity %d is still drawn on its own", id)
		}
	}
	if !w.ents[other].mesh.Renderable() || batchDrawn(w.ents[other]) {
		t.Fatal("a mesh with no copies stays on its own")
	}
	if w.visibleMeshCount() != 2 {
		t.Fatalf("draws: want the group and the single mesh, got %d", w.visibleMeshCount())
	}
}

// Two entities that differ in any setting must not share a draw.
func TestBatchSplitsDifferentMaterials(t *testing.T) {
	w := batchWorld()
	geom := geometry.NewCube(1)
	a, b, c, d := w.meshEnt(geom, 0), w.meshEnt(geom, 0), w.meshEnt(geom, 0), w.meshEnt(geom, 0)
	w.ents[c].mat.SetColor(&math32.Color{1, 0, 0})
	w.ents[d].mat.SetColor(&math32.Color{1, 0, 0})
	w.updateBatches()
	if len(w.batch.active) != 2 {
		t.Fatalf("two colors, want two groups, got %d", len(w.batch.active))
	}
	w.ents[d].mat.SetShininess(90)
	w.updateBatches()
	if len(w.batch.active) != 1 || !w.ents[c].mesh.Renderable() || !w.ents[d].mesh.Renderable() {
		t.Fatalf("shininess must split them: %d groups", len(w.batch.active))
	}
	if !batchDrawn(w.ents[a]) || !batchDrawn(w.ents[b]) {
		t.Fatal("the unchanged pair stays batched")
	}
}

// A transparent copy leaves its group and is drawn by G3N again.
func TestBatchReleasesWhatStopsQualifying(t *testing.T) {
	w := batchWorld()
	geom := geometry.NewCube(1)
	a, b, c := w.meshEnt(geom, 0), w.meshEnt(geom, 0), w.meshEnt(geom, 0)
	w.updateBatches()
	w.ents[c].mat.SetTransparent(true)
	w.updateBatches()
	if !w.ents[c].mesh.Renderable() || batchDrawn(w.ents[c]) {
		t.Fatal("a transparent entity must be drawn on its own")
	}
	if len(w.batch.active) != 1 || len(w.batch.active[0].members) != 2 {
		t.Fatalf("the rest stay grouped: %d groups", len(w.batch.active))
	}
	delete(w.ents, b)
	w.updateBatches()
	if !w.ents[a].mesh.Renderable() || len(w.batch.active) != 0 {
		t.Fatal("one copy left: back to a plain mesh")
	}
	if n := len(w.scene.Children()); n > 3 {
		t.Fatalf("the proxy should have left the scene: %d children", n)
	}
}

func TestBatchOffRestoresEveryMesh(t *testing.T) {
	w := batchWorld()
	geom := geometry.NewCube(1)
	a, b := w.meshEnt(geom, 0), w.meshEnt(geom, 0)
	w.updateBatches()
	if _, err := w.commandTable()["setautobatch"]([]value.Value{value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	w.updateBatches()
	if !w.ents[a].mesh.Renderable() || !w.ents[b].mesh.Renderable() || len(w.batch.active) != 0 {
		t.Fatal("SetAutoBatch 0 should give every mesh back")
	}
	if w.batchedCount() != 0 {
		t.Fatalf("batched count %d", w.batchedCount())
	}
}

func TestBatchKeepsHiddenCopiesInTheGroup(t *testing.T) {
	w := batchWorld()
	geom := geometry.NewCube(1)
	a, b := w.meshEnt(geom, 0), w.meshEnt(geom, 0)
	w.ents[a].mesh.SetVisible(false)
	w.updateBatches()
	g := w.batch.active[0]
	n := g.cullToScratch(&w.batch, func(i int) bool { return nodeShown(g.nodes[i]) })
	if n != 1 {
		t.Fatalf("one copy is hidden, %d drawn", n)
	}
	_ = b
}

func TestFrustumPlanes(t *testing.T) {
	var view, proj, vp math32.Matrix4
	view.Identity() // at the origin, looking down -Z
	proj.MakePerspective(90, 1, 1, 100)
	vp.MultiplyMatrices(&proj, &view)
	planes := frustumPlanes(&vp)
	for _, c := range []struct {
		name string
		s    [4]float32
		want bool
	}{
		{"in front", [4]float32{0, 0, -10, 1}, true},
		{"behind", [4]float32{0, 0, 10, 1}, false},
		{"past far", [4]float32{0, 0, -200, 1}, false},
		{"far left", [4]float32{-100, 0, -10, 1}, false},
		{"touching the left edge", [4]float32{-10.5, 0, -10, 1}, true},
	} {
		if got := sphereInPlanes(&planes, c.s[:]); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestBatchVertexSourceHasBothForms(t *testing.T) {
	for name, src := range map[string]string{"mbshadow": mbshadowVertex, "mbphysical": mbphysicalVertex} {
		out := batchVertexSource(src)
		for _, want := range []string{"#ifdef MB_INSTANCED", "location = 12", "MVP = BatchProj * ModelViewMatrix", "uniform mat3 NormalMatrix;\n#endif"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q", name, want)
			}
		}
		if n := strings.Count(out, "uniform mat4 MVP;"); n != 1 {
			t.Errorf("%s: MVP declared %d times outside the variant", name, n)
		}
	}
}
