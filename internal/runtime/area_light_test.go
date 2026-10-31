package runtime

import (
	"math"
	"strings"
	"testing"

	"github.com/g3n/engine/math32"
)

func near32(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-4 }

func TestAreaLightPacking(t *testing.T) {
	w := New(".")
	ids := []int{}
	for i := 0; i < 6; i++ {
		id := w.createAreaLight(2, 1, 0)
		w.ents[id].node.GetNode().SetPosition(float32(i)*10, 0, 0)
		ids = append(ids, id)
	}
	for _, id := range ids {
		w.ents[id].node.GetNode().UpdateMatrixWorld()
	}
	w.renderSerial = 1
	u := w.packAreaLights()
	if u.n != maxAreaLights {
		t.Fatalf("packed %d lights, want %d", u.n, maxAreaLights)
	}
	// Camera is nil, so distance is from the origin: nearest first.
	if u.pos[0].X != 0 || u.pos[3].X != 30 {
		t.Errorf("not nearest first: %v .. %v", u.pos[0], u.pos[3])
	}
	// Half extents, and the emitting normal on -Z.
	if !near32(u.right[0].Length(), 1) || !near32(u.up[0].Length(), 0.5) {
		t.Errorf("half extents %v %v", u.right[0].Length(), u.up[0].Length())
	}
	var nrm math32.Vector3
	nrm.CrossVectors(&u.right[0], &u.up[0]).Normalize()
	if !near32(nrm.Z, -1) {
		t.Errorf("emitting normal %v, want -Z", nrm)
	}
	if want := float32(4); u.color[0].R != want {
		t.Errorf("colour*intensity %v, want %v", u.color[0].R, want)
	}

	// A second call in the same frame is cached; a freed light drops out.
	w.freeEntityID(ids[0])
	if again := w.packAreaLights(); again.pos[0].X != 0 {
		t.Error("same-frame pack was recomputed")
	}
	w.renderSerial = 2
	u = w.packAreaLights()
	if u.pos[0].X != 10 || w.areaLights[ids[0]] != nil {
		t.Errorf("freed light still packed: first %v", u.pos[0])
	}
	if _, err := w.areaLightOf(ids[0], "Test"); err == nil {
		t.Error("areaLightOf accepted a freed light")
	}
}

func TestAreaLightGLSLInBothShaders(t *testing.T) {
	for name, src := range map[string]string{"mbphysical": mbphysicalFragment, "bsshadow": mbshadowFragment} {
		for _, need := range []string{"uniform int AreaCount;", "areaFormFactor(i, WorldPos, wN)", "areaSpecPoint("} {
			if !strings.Contains(src, need) {
				t.Errorf("%s missing %q", name, need)
			}
		}
	}
}
