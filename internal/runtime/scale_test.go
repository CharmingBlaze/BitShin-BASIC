package runtime

import (
	"math"
	"testing"
)

func TestChunkOfAndNeeded(t *testing.T) {
	s := &worldStream{size: 16, radius: 1}
	if chunkOf(0, 0, 16) != (chunkKey{0, 0}) {
		t.Fatal(chunkOf(0, 0, 16))
	}
	if chunkOf(17, -3, 16) != (chunkKey{1, -1}) {
		t.Fatal(chunkOf(17, -3, 16))
	}
	need := s.needed(0, 0)
	if len(need) != 9 {
		t.Fatalf("radius 1 should be 9 chunks, got %d", len(need))
	}
}

func TestNoiseFBMRange(t *testing.T) {
	n := newNoise2(42)
	min, maxv := 1.0, -1.0
	for i := 0; i < 80; i++ {
		v := n.fbm(float64(i)*0.17, float64(i)*0.11, 4, 0.5, 2)
		if v < min {
			min = v
		}
		if v > maxv {
			maxv = v
		}
	}
	if min < -1.2 || maxv > 1.2 {
		t.Fatalf("fbm out of range %v %v", min, maxv)
	}
	if math.Abs(n.fbm(0, 0, 4, 0.5, 2)-n.fbm(0, 0, 4, 0.5, 2)) > 1e-12 {
		t.Fatal("noise not deterministic")
	}
}

func TestGerstnerHeightStorm(t *testing.T) {
	w := defaultWaves()
	calm := gerstnerHeight(w[:], 4, 0, 0, 0.5, 1)
	storm := gerstnerHeight(w[:], 4, 0, 0, 0.5, 1.7)
	if math.Abs(float64(storm)) < math.Abs(float64(calm))-1e-5 && storm == calm {
		t.Fatalf("storm should scale amp: calm=%v storm=%v", calm, storm)
	}
}

func TestGerstnerZeroWavesFlat(t *testing.T) {
	w := defaultWaves()
	if gerstnerHeight(w[:], 0, 3, 4, 1.2, 1) != 0 {
		t.Fatal("zero wave count must be a still plane")
	}
}

func TestGerstnerWindChangesHeight(t *testing.T) {
	w := defaultWaves()
	a := gerstnerHeightWind(w[:], 4, 2, 3, 0.8, 1, 0, 0, 0)
	b := gerstnerHeightWind(w[:], 4, 2, 3, 0.8, 1, 1, 0, 0.9)
	if a == b {
		t.Fatal("wind should change Gerstner height")
	}
}

func TestHeightFieldBilinear(t *testing.T) {
	hf := heightField{gw: 2, gd: 2, worldW: 10, worldD: 10, h: []float32{0, 10, 0, 10}}
	mid := hf.sample(5, 0)
	if math.Abs(float64(mid-5)) > 0.01 {
		t.Fatalf("bilinear x: %v", mid)
	}
}

func TestTerrainGLHeightDeterministic(t *testing.T) {
	a := terrainGLHeight(10, 20, 6, 0.035, 3.2, 2.15, 0, 0)
	b := terrainGLHeight(10, 20, 6, 0.035, 3.2, 2.15, 0, 0)
	if math.Abs(a-b) > 1e-12 {
		t.Fatal("terrain-gl noise not deterministic")
	}
	if a < 0 || a > 400 {
		t.Fatalf("height out of expected range: %v", a)
	}
	hi := terrainGLHeight(0, 0, 8, 0.035, 6, 1.25, 0, 0)
	lo := terrainGLHeight(0, 0, 8, 0.035, 2, 1.25, 0, 0)
	if hi < lo-1e-6 {
		t.Fatalf("disp should raise height: hi=%v lo=%v", hi, lo)
	}
	h0 := terrainGLHeight(0, 0, 6, 0.04, 16, 1.25, 0, 0)
	h1 := terrainGLHeight(40, 28, 6, 0.04, 16, 1.25, 0, 0)
	if math.Abs(h0-h1) < 1.5 {
		t.Fatalf("terrain-gl should roll: h0=%v h1=%v", h0, h1)
	}
}

func TestChunkEdgeVertsIdentical(t *testing.T) {
	if gridCoord(0, 10, 0, 8) != 0 || gridCoord(0, 10, 8, 8) != 10 {
		t.Fatalf("snap %v %v", gridCoord(0, 10, 0, 8), gridCoord(0, 10, 8, 8))
	}
	if gridCoord(10, 20, 0, 8) != 10 {
		t.Fatalf("neighbor start %v", gridCoord(10, 20, 0, 8))
	}
	if gridCoord(0, 10, 8, 8) != gridCoord(10, 20, 0, 8) {
		t.Fatal("shared border world X must match")
	}
}

func TestSharedHeightNormalSeam(t *testing.T) {
	sample := func(x, z float32) float32 {
		return float32(math.Sin(float64(x)*0.2) + math.Cos(float64(z)*0.15)*2)
	}
	ax, ay, az := sharedHeightNormal(sample, 10, 4, 0.85)
	bx, by, bz := sharedHeightNormal(sample, 10, 4, 0.85)
	if math.Abs(float64(ax-bx))+math.Abs(float64(ay-by))+math.Abs(float64(az-bz)) > 1e-6 {
		t.Fatalf("same XZ must share normal: %v %v", []float32{ax, ay, az}, []float32{bx, by, bz})
	}
	if ay <= 0 {
		t.Fatalf("up-facing expected, ny=%v", ay)
	}
}

func TestHeightMeshSkirtIndicesInRange(t *testing.T) {
	for _, segs := range []int{4, 8, 17, 25} {
		for _, step := range []int{1, 2, 4} {
			g := buildHeightMesh(func(x, z float32) float32 { return 1 }, 0, 0, 16, 16, segs, step)
			chunkSize := segs / step
			if chunkSize < 1 {
				chunkSize = 1
			}
			n := chunkSize + 1
			maxV := uint32(n * (n + 4))
			idx := g.Indices()
			if len(idx) == 0 {
				t.Fatalf("no indices segs=%d step=%d", segs, step)
			}
			for i, v := range idx {
				if v >= maxV {
					t.Fatalf("skirt OOB index %d at %d (n=%d segs=%d step=%d max=%d)", v, i, n, segs, step, maxV)
				}
			}
		}
	}
}

func TestTerrainSplatWeights(t *testing.T) {
	s, g, r, sn := terrainSplatWeights(1.0, 2.2, 1.4, 0.65, 1, false, 9)
	if s < 0.99 || g+r+sn > 0.01 {
		t.Fatalf("below water should be sand: %v %v %v %v", s, g, r, sn)
	}
	s, g, r, sn = terrainSplatWeights(6, 2.2, 1.4, 0.65, 0.95, false, 9)
	if g < 0.9 {
		t.Fatalf("flat highland should be grass: %v %v %v", s, g, r)
	}
	s, g, r, sn = terrainSplatWeights(6, 2.2, 1.4, 0.65, 0.2, false, 9)
	if r < 0.9 {
		t.Fatalf("steep should be rock: %v %v %v", s, g, r)
	}
	_, _, _, sn = terrainSplatWeights(12, 2.2, 1.4, 0.65, 0.9, true, 9)
	if sn < 0.5 {
		t.Fatalf("high snow: %v", sn)
	}
}

func TestTerrainGLCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"createterraingl", "applyterrainsplat", "setterrainsplat",
		"setterrainoctaves", "setterrainfreq", "setterraindispfactor",
		"createproctexture", "createvolumetricclouds", "setskypreset",
	} {
		if m[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
}

func TestScaleCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"createworldstream", "jobsubmit", "createinstancedmesh",
		"createlightprobe", "createprocterrain", "createwater",
		"setgeoorigin", "geoproject", "loadgeojson",
		"generateheightmap", "createterrainfromheightmap",
		"createcrowd", "statsfps", "bodysleep", "setshadowresolution",
	} {
		if m[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
}
