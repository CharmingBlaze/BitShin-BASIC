package phys3d

import "testing"

func TestHeightFieldTrisGrid(t *testing.T) {
	n := 4
	samples := make([]float32, n*n)
	verts, idx := heightFieldTris(samples, n, 0, 0, 0, 1, 1, 1)
	if len(verts) != n*n {
		t.Fatalf("verts %d", len(verts))
	}
	if len(idx) != (n-1)*(n-1)*6 {
		t.Fatalf("idx %d", len(idx))
	}
	v2, i2 := heightFieldTris(nil, 2, 0, 0, 0, 1, 1, 1)
	if v2 != nil || i2 != nil {
		t.Fatal("short samples should yield no mesh")
	}
}

func TestCompoundHullPoints(t *testing.T) {
	pts := compoundHullPoints([]CompoundPart{
		{Kind: CompoundBox, Ox: 1, A: 0.5, B: 0.25, C: 0.5},
		{Kind: CompoundSphere, Oz: -1, A: 0.4},
	})
	if len(pts) != 16 {
		t.Fatalf("got %d corners", len(pts))
	}
	found := false
	for _, p := range pts {
		if p[0] > 1.4 {
			found = true
		}
	}
	if !found {
		t.Fatal("box offset corners missing")
	}
}
