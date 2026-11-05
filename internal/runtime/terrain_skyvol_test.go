package runtime

import (
	"testing"

	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func flatGround(x, z float32) float32 { return 0 }

func TestSkyVolumeOpenGroundSeesTheSky(t *testing.T) {
	s := newSkyVolScene(flatGround, -50, -50, 100, 100, 8, nil)
	dirs := skyRayDirs()
	vis, _, by, _ := s.sample(0, 2, 0, &dirs)
	if vis < 0.99 {
		t.Fatalf("open ground sees %v of the sky", vis)
	}
	if by < 0.9 {
		t.Fatalf("the light should come from straight up, got %v", by)
	}
}

func TestSkyVolumeUnderTreesIsDarker(t *testing.T) {
	var trees []skyVolTree
	for x := float32(-30); x <= 30; x += 8 {
		for z := float32(-30); z <= 30; z += 8 {
			trees = append(trees, skyVolTree{x, 14, z, 6})
		}
	}
	s := newSkyVolScene(flatGround, -100, -100, 200, 200, 8, trees)
	dirs := skyRayDirs()
	under, _, _, _ := s.sample(0, 2, 0, &dirs)
	open, _, _, _ := s.sample(90, 2, 90, &dirs)
	if under >= open*0.8 {
		t.Fatalf("under the canopy %v, in the open %v", under, open)
	}
	if under <= 0 {
		t.Fatal("light gets through a canopy")
	}
}

func TestSkyVolumeValleyWallBlocksAndBends(t *testing.T) {
	// A wall to the east.
	wall := func(x, z float32) float32 {
		if x > 12 {
			return 80
		}
		return 0
	}
	s := newSkyVolScene(wall, -100, -100, 200, 200, 8, nil)
	dirs := skyRayDirs()
	vis, bx, _, _ := s.sample(0, 2, 0, &dirs)
	if vis > 0.9 {
		t.Fatalf("a wall should hide part of the sky, got %v", vis)
	}
	if bx >= 0 {
		t.Fatalf("the light should come from away from the wall, got x %v", bx)
	}
}

func TestSkyVolumeBakeFillsBelowGroundFromAbove(t *testing.T) {
	hill := func(x, z float32) float32 { return 20 - math32.Abs(x)*0.2 }
	s := newSkyVolScene(hill, -40, -40, 80, 80, 8, nil)
	r := s.bake(2)
	if len(r.pix) != r.nx*r.ny*r.nz*4 || r.nx < 2 || r.ny < 2 || r.nz < 2 {
		t.Fatalf("got %d bytes for %d by %d by %d", len(r.pix), r.nx, r.ny, r.nz)
	}
	// The lowest layer is under the hilltop: it should copy its value, not stay black.
	o := ((r.nz/2*r.ny+0)*r.nx + r.nx/2) * 4
	if r.pix[o+3] == 0 {
		t.Fatal("a sample under the ground should take the value above it")
	}
	for _, v := range r.ground {
		if v <= 0 || v > 1 {
			t.Fatalf("ground visibility %v", v)
		}
	}
}

func TestBilinearGrid(t *testing.T) {
	g := []float32{0, 1, 1, 0}
	if got := bilinearGrid(g, 2, 2, 0.5, 0.5); got != 0.5 {
		t.Fatalf("got %v", got)
	}
	if got := bilinearGrid(g, 2, 2, -5, -5); got != 0 {
		t.Fatalf("outside should clamp, got %v", got)
	}
}

func TestSkyVolumeCommandWithNoTerrain(t *testing.T) {
	w := New(".")
	w.ready = true
	if _, err := w.Call("bakeskyvolume", nil); err != nil {
		t.Fatal(err)
	}
	if v, _ := w.Call("skyvolumeready", []value.Value{}); v.Num != 0 {
		t.Fatal("nothing has been baked")
	}
}
