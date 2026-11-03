package runtime

import "testing"

func TestMountainIslandIsSeaAtTheRimLandInTheMiddle(t *testing.T) {
	s := heightmapPreset("mountain-island")
	s.Width, s.Height, s.Seed = 128, 128, 3
	d := GenerateHeightmapCPU(s)
	const n = 128
	var edge, middle float32
	for i := 0; i < n; i++ {
		edge += d[i]
	}
	edge /= n
	for z := n/2 - 16; z < n/2+16; z++ {
		for x := n/2 - 16; x < n/2+16; x++ {
			middle += d[z*n+x]
		}
	}
	middle /= 32 * 32
	if edge > 0.1 {
		t.Fatalf("rim height %v should be under the sea", edge)
	}
	if middle < edge+0.15 {
		t.Fatalf("middle %v should stand above rim %v", middle, edge)
	}
	again := GenerateHeightmapCPU(s)
	for i := range d {
		if d[i] != again[i] {
			t.Fatal("the same seed must give the same land")
		}
	}
}

func TestValleyHasALakeMeadowsAndMountains(t *testing.T) {
	for _, seed := range []int{3, 11, 42} {
		s := heightmapPreset("valley")
		s.Width, s.Height, s.Seed = 128, 128, seed
		d := GenerateHeightmapCPU(s)
		var lake, meadow, high int
		for i, v := range d {
			if v < 0 || v > 1 || v != v {
				t.Fatalf("seed %d: height %d is %v", seed, i, v)
			}
			switch {
			case v < 0.27:
				lake++
			case v < 0.42:
				meadow++
			case v > 0.6:
				high++
			}
		}
		n := len(d)
		if lake < n/60 || meadow < n/8 || high < n/40 {
			t.Fatalf("seed %d: lake %d, meadow %d, high %d of %d", seed, lake, meadow, high, n)
		}
		if mid, rim := d[64*128+64], d[3*128+3]; mid >= 0.27 || rim <= mid {
			t.Fatalf("seed %d: middle %v should be under water, below the rim %v", seed, mid, rim)
		}
	}
}
