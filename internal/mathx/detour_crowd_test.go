package mathx

import "testing"

func TestDetourSampleSidesteps(t *testing.T) {
	a := []Disk{
		{X: 0, Z: 0, VX: 2, VZ: 0, R: 0.5},
		{X: 1.2, Z: 0, VX: -2, VZ: 0, R: 0.5},
	}
	vx, vz := SampleDetour(0, a, 2, 0, 2, 2.5)
	if vx >= 1.95 && vz < 0.05 && vz > -0.05 {
		t.Fatalf("kept the head-on velocity %v %v", vx, vz)
	}
}
