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
