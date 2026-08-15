package runtime

import (
	"testing"

	"github.com/g3n/engine/core"
)

func TestSkyFitScaleKeepsCornersInsideFar(t *testing.T) {
	const sqrt3 = 1.7320508
	for _, far := range []float32{400, 900, 4000} {
		s := skyFitScale(0.25, far)
		corner := 0.5 * s * sqrt3
		if corner >= far {
			t.Fatalf("far=%v scale=%v corner=%v — cube would clip", far, s, corner)
		}
		if 0.5*s <= 0.25 {
			t.Fatalf("far=%v scale=%v — near plane would clip the cube faces", far, s)
		}
	}
}

func TestSkyBoxUsesSixFaceMaterials(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	id, err := w.createSkyBoxFromFaces(generateDefaultSkyFaces())
	if err != nil {
		t.Fatal(err)
	}
	s := w.skies[id]
	if s == nil || s.box == nil {
		t.Fatal("sky slot missing")
	}
	if s.box.Cullable() {
		t.Fatal("sky must not be frustum-culled")
	}
	mats := s.box.Materials()
	if len(mats) != 6 {
		t.Fatalf("want 6 face materials, got %d", len(mats))
	}
	if s.box.GetGeometry().GroupCount() != 6 {
		t.Fatalf("want 6 cube groups, got %d", s.box.GetGeometry().GroupCount())
	}
}
