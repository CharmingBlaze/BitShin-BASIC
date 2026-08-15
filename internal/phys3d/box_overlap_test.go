package phys3d

import "testing"

func TestBoxOverlapMTVSeparatesAxisAligned(t *testing.T) {
	nx, ny, nz, depth, hit := BoxOverlapMTV(
		0, 1, 0, 0.18, 1.05, 0.28, 0, 0, 0, 1,
		0.1, 1, 0, 0.30, 0.30, 0.30, 0, 0, 0, 1,
	)
	if !hit {
		t.Fatal("prong-sized box should overlap a 0.6m prize at the same center")
	}
	if depth < 0.2 {
		t.Fatalf("expected deep overlap, depth=%v n=%v %v %v", depth, nx, ny, nz)
	}
	_ = nx
}

func TestBoxOverlapMTVApart(t *testing.T) {
	_, _, _, _, hit := BoxOverlapMTV(
		0, 8, 0, 0.42, 0.16, 0.42, 0, 0, 0, 1,
		0, 1, 0, 0.30, 0.30, 0.30, 0, 0, 0, 1,
	)
	if hit {
		t.Fatal("carriage at y=8 should not overlap a floor prize")
	}
}

func TestKinematicRestingOverlapVsSleepingDynamic(t *testing.T) {
	w := New()
	defer w.Close()
	if w.Backend() != BackendJolt {
		t.Skip("needs Jolt")
	}
	w.SetGravity(0, 0, 0)
	w.AddBoxEx(1, 0, 1, 0, 0.5, 0.5, 0.5, MotionTypeKinematic)
	w.AddBoxEx(2, 0.2, 1, 0, 0.5, 0.5, 0.5, MotionTypeDynamic)
	w.SetMass(2, 0.1)
	w.Sleep(2)
	dt := float32(1.0 / 60.0)
	for i := 0; i < 20; i++ {
		w.MoveKinematic(1, 0, 1, 0, 0, 0, 0, 1, dt)
		w.Step(dt)
	}
	x, _, _, ok := w.GetPosition(2)
	if !ok {
		t.Fatal("missing dynamic")
	}
	if x > 0.35 {
		t.Log("Jolt did depenetrate a sleeping overlap; x=", x)
		return
	}
	t.Log("proved: kinematic sitting in a sleeping box does not push (x=", x, ")")
}
