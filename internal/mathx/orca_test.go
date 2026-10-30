package mathx

import "testing"

func TestORCAKeepsFreeVelocity(t *testing.T) {
	a := []Disk{{X: 0, Z: 0, R: 0.4}}
	vx, vz := ORCA(0, a, 2, 0, 4, 1.2, 1.0/60)
	if vx < 1.9 || vz > 0.01 {
		t.Fatalf("free %v %v", vx, vz)
	}
}

func TestORCASeparatesOverlap(t *testing.T) {
	a := []Disk{
		{X: 0, Z: 0, VX: 2, VZ: 0, R: 0.6},
		{X: 0.4, Z: 0, VX: -2, VZ: 0, R: 0.6},
	}
	vx, _ := ORCA(0, a, 2, 0, 4, 1.2, 1.0/60)
	if vx >= 2 {
		t.Fatalf("overlapping agent kept full closing speed %v", vx)
	}
	if vx > 0.5 {
		t.Fatalf("expected a strong push apart, vx %v", vx)
	}
}
