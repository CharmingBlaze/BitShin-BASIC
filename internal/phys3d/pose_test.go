package phys3d

import "testing"

func TestQuatIntegrateYaw(t *testing.T) {
	q := [4]float32{0, 0, 0, 1}
	for i := 0; i < 20; i++ {
		q = quatIntegrate(q, [3]float32{0, 1, 0}, 0.05)
	}
	if q[1] < 0.3 || q[3] < 0.3 {
		t.Fatalf("expected yaw off identity, got %v", q)
	}
	x, y, z, w := quatNormalize(0, 0, 0, 0)
	if x != 0 || y != 0 || z != 0 || w != 1 {
		t.Fatalf("zero quat should become identity %v %v %v %v", x, y, z, w)
	}
}

func TestSoftConstraintSlider(t *testing.T) {
	dx, dy, dz := softConstraintDelta(2, 1, 0, 0, 4, 3, 0, 0)
	if dx > 0.01 || dy < 2.9 || dz != 0 {
		t.Fatalf("slider should drop X error, keep Y, got %v %v %v", dx, dy, dz)
	}
	sx, sy, sz := softConstraintDelta(3, 0, 1, 0, 0, 4, 0, 1)
	if sy < 2.9 || sy > 3.1 || sx != 0 || sz != 0 {
		t.Fatalf("spring should close to rest 1, got %v %v %v", sx, sy, sz)
	}
}

func TestConeClampDir(t *testing.T) {
	x, y, z := coneClampDir(1, 0, 0, 0.4, 2, 4, 0)
	ang := hingeAngle(0, 0, 1, 1, 0, 0, x, y, z)
	_ = ang
	nx, _, _, _ := unit3(x, y, z)
	c := nx
	if c < 0.9 {
		t.Fatalf("cone should pull toward +X, got %v %v %v", x, y, z)
	}
}

func TestHingeClampVec(t *testing.T) {
	x, y, z := hingeClampVec(0, 1, 0, 1, 0, 0, -0.2, 0.2, 0, 0, 1)
	if x < 0.7 || z > 0.4 {
		t.Fatalf("hinge limit should keep near +X, got %v %v %v", x, y, z)
	}
}
