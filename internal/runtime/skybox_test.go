package runtime

import (
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
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

func TestSkySampleMatchesThePaintedFace(t *testing.T) {
	top := math32.Color{0.1, 0.2, 0.9}
	bot := math32.Color{0.85, 0.4, 0.15}
	imgs := generateSkyImages(top, bot, 1, 0.15, 0, nil)
	upR, upG, upB, ok := sampleSkyImages(imgs, 0, 1, 0)
	if !ok {
		t.Fatal("sky images should sample")
	}
	dnR, dnG, dnB, _ := sampleSkyImages(imgs, 0, -1, 0)
	if upB <= dnB || upR >= dnR {
		t.Fatalf("up (%v,%v,%v) should be bluer than down (%v,%v,%v)", upR, upG, upB, dnR, dnG, dnB)
	}
	w := New(".")
	w.scene = core.NewNode()
	if _, err := w.publishSky(imgs); err != nil {
		t.Fatal(err)
	}
	if w.skyRev == 0 || w.skyImgs[0] == nil {
		t.Fatal("publishing a sky should feed the reflection cube")
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
