package runtime

import (
	"image"
	"image/color"
	"testing"
)

// Without a seam at the wrap, a repeating ground texture shows no level lines
// up a cliff: the two edges of the tile must closely agree.
func TestRockTextureTilesWithoutASeam(t *testing.T) {
	const n = 128
	img := generateProcTerrainRGBA("rock", n)
	var worst int
	gap := func(a, b color.RGBA) int {
		d := int(a.G) - int(b.G)
		if d < 0 {
			d = -d
		}
		return d
	}
	check := func(a, b image.Point) {
		if d := gap(img.RGBAAt(a.X, a.Y), img.RGBAAt(b.X, b.Y)); d > worst {
			worst = d
		}
	}
	for i := 0; i < n; i++ {
		check(image.Pt(i, 0), image.Pt(i, n-1))
		check(image.Pt(0, i), image.Pt(n-1, i))
	}
	if worst > 24 {
		t.Fatalf("tile edges differ by %d of 255", worst)
	}
}
