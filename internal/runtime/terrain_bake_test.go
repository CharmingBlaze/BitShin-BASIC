package runtime

import "testing"

func slopeGrid(w, d int, fn func(x, z int) float32) *bakeGrid {
	g := &bakeGrid{w: w, d: d, cellX: 2, cellZ: 2, h: make([]float32, w*d)}
	for z := 0; z < d; z++ {
		for x := 0; x < w; x++ {
			g.h[z*w+x] = fn(x, z)
		}
	}
	return g
}

func TestBakeNormalsFaceDownhill(t *testing.T) {
	// A gentle slope, under the rock limit, so only soft detail is added. Height
	// rises with X, so the normal leans toward -X.
	g := slopeGrid(64, 64, func(x, z int) float32 { return float32(x) * 2 * 0.3 })
	img := bakeTerrainMap(g, 256, 1)
	if r := img.Pix[img.PixOffset(128, 128)]; r >= 110 {
		t.Fatalf("slope up in +X: normal X encoded %d, want well under 128", r)
	}
	// Rising with map Z is falling with scene Z, so scene normal Z is positive.
	g = slopeGrid(64, 64, func(x, z int) float32 { return float32(z) * 2 * 0.3 })
	img = bakeTerrainMap(g, 256, 1)
	if gz := img.Pix[img.PixOffset(128, 128)+1]; gz <= 146 {
		t.Fatalf("slope up in map +Z: normal Z encoded %d, want well over 128", gz)
	}
}

func TestBakeOcclusionDarkensAPit(t *testing.T) {
	g := slopeGrid(65, 65, func(x, z int) float32 {
		dx, dz := float32(x-32), float32(z-32)
		return (dx*dx + dz*dz) * 0.05
	})
	ao := bakeAO(g)
	mid, rim := ao[32*65+32], ao[0]
	if mid >= 0.7 {
		t.Fatalf("the bottom of a pit should be occluded, got %v", mid)
	}
	if rim <= mid {
		t.Fatalf("the rim %v should see more sky than the pit %v", rim, mid)
	}
}

func TestBakeOcclusionIsOpenOnFlatGround(t *testing.T) {
	g := slopeGrid(32, 32, func(x, z int) float32 { return 5 })
	for i, v := range bakeAO(g) {
		if v != 1 {
			t.Fatalf("cell %d: %v", i, v)
		}
	}
}

func TestBakeFlowCollectsDownhill(t *testing.T) {
	// A V-shaped valley draining toward z = 0: its floor gathers the slopes.
	g := slopeGrid(33, 33, func(x, z int) float32 {
		dx := float32(x - 16)
		if dx < 0 {
			dx = -dx
		}
		return dx*2 + float32(z)*0.5
	})
	flow := bakeFlow(g)
	if floor, side := flow[1*33+16], flow[1*33+4]; floor <= side {
		t.Fatalf("valley floor %v should carry more than the side %v", floor, side)
	}
}

func TestBakeTinyGridIsSafe(t *testing.T) {
	g := slopeGrid(2, 2, func(x, z int) float32 { return float32(x + z) })
	if img := bakeTerrainMap(g, bakeMinSize, 1); img.Bounds().Dx() != bakeMinSize {
		t.Fatalf("size %d", img.Bounds().Dx())
	}
}

func TestBakeSizeStaysAPowerOfTwo(t *testing.T) {
	for gw, want := range map[int]int{16: 256, 128: 512, 256: 1024, 512: 2048, 4096: 2048} {
		if got := bakeSizeFor(gw); got != want {
			t.Fatalf("grid %d: size %d, want %d", gw, got, want)
		}
	}
}

// The full terrain drift turned a meadow gold, so the tint must stay slight.
func TestGrassTintStaysSlight(t *testing.T) {
	var varied bool
	first := grassGroundTint(0, 0)
	for x := float32(-400); x <= 400; x += 37 {
		for z := float32(-400); z <= 400; z += 37 {
			c := grassGroundTint(x, z)
			if c.R <= 0 || c.G <= 0 || c.B <= 0 {
				t.Fatalf("tint at %v,%v is %v", x, z, c)
			}
			if c.R/c.B > 1.25 || c.B/c.R > 1.25 {
				t.Fatalf("tint at %v,%v is %v: strong enough to change the grass's hue", x, z, c)
			}
			varied = varied || c != first
		}
	}
	if !varied {
		t.Fatal("the tint never changes")
	}
}
