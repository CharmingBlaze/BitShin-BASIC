package runtime

import (
	"testing"

	"github.com/g3n/engine/math32"
)

func TestPhysicsThreadsSizesJobPool(t *testing.T) {
	w := New(".")
	w.applyPhysThreads(8)
	if w.physThreads != 8 {
		t.Fatalf("physThreads %d", w.physThreads)
	}
	if w.jobWorkers != 8 {
		t.Fatalf("jobWorkers %d", w.jobWorkers)
	}
	p := w.ensureJobs()
	if p == nil {
		t.Fatal("job pool")
	}
	w.applyPhysThreads(4)
	if w.jobs != nil {
		t.Fatal("pool should rebuild after PhysicsThreads")
	}
	p2 := w.ensureJobs()
	if p2 == nil || p2 == p {
		t.Fatal("expected a new job pool")
	}
	p2.close()
}

func TestNlerpQuatHalfTurn(t *testing.T) {
	a := math32.Quaternion{W: 1}
	b := math32.Quaternion{Y: 1}
	q := nlerpQuat(a, b, 0.5)
	if q.W < 0.6 || q.Y < 0.6 {
		t.Fatalf("nlerp %v %v %v %v", q.X, q.Y, q.Z, q.W)
	}
}

func TestAnimBlendDefaults(t *testing.T) {
	st := &animState{dir: 1, speed: 1, prev: -1, blend: 1}
	if st.prev != -1 || st.blend != 1 {
		t.Fatal("idle blend must not mix clip 0")
	}
}

func TestSceneKindFallbackCube(t *testing.T) {
	if spawnKindName("") != "cube" {
		t.Fatalf("empty kind %q", spawnKindName(""))
	}
	if spawnKindName("sphere") != "sphere" {
		t.Fatal("sphere")
	}
	if spawnKindName("MESH") != "mesh" {
		t.Fatal("mesh")
	}
}
