package phys3d

import (
	"os"
	"testing"
)

// TestJoltOnThisPlatform proves the native backend loads and simulates. CI
// sets MOON_REQUIRE_JOLT=1 on Windows, Linux amd64/arm64, and macOS arm64, so
// a silent drop to the fallback fails there instead of skipping every Jolt test.
func TestJoltOnThisPlatform(t *testing.T) {
	w := New()
	defer w.Close()
	if w.Backend() != BackendJolt {
		if os.Getenv("MOON_REQUIRE_JOLT") != "" {
			t.Fatalf("MOON_REQUIRE_JOLT is set but the backend is %q", w.Backend())
		}
		t.Skip("needs Jolt")
	}
	w.SetGravity(0, -9.81, 0)
	w.AddGround(1, 0)
	w.AddBoxEx(2, 0, 3, 0, 0.5, 0.5, 0.5, MotionTypeDynamic)
	// A sphere hung from a static anchor by a point joint
	w.AddBoxEx(3, 5, 4, 0, 0.2, 0.2, 0.2, MotionTypeStatic)
	w.AddSphereEx(4, 5, 3, 0, 0.3, MotionTypeDynamic)
	w.CreatePointJoint(3, 4, 5, 3.5, 0)
	for i := 0; i < 180; i++ {
		w.Step(1.0 / 60.0)
	}
	_, y, _, ok := w.GetPosition(2)
	if !ok {
		t.Fatal("missing box")
	}
	if y < 0.6 || y > 0.9 {
		t.Fatalf("box should rest on the ground slab at 0.75, y=%v", y)
	}
	_, hy, _, ok := w.GetPosition(4)
	if !ok {
		t.Fatal("missing hung sphere")
	}
	if hy < 2 {
		t.Fatalf("point joint should hold the sphere up, y=%v", hy)
	}
	id, _, _, _, hit := w.Raycast(0, 10, 0, 0, -20, 0)
	if !hit || id != 2 {
		t.Fatalf("ray down should hit the box, hit=%v id=%d", hit, id)
	}
}
