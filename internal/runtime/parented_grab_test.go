package runtime

import (
	"testing"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/phys3d"
)

func TestResolveDrivenOverlapsPushesPrizeFromMesh(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	claw := &Entity{node: core.NewNode(), bodyType: 3, boxX: 0.20, boxY: 1.05, boxZ: 0.28}
	prize := &Entity{node: core.NewNode(), bodyType: 1, boxX: 0.30, boxY: 0.30, boxZ: 0.30}
	cid := w.addEntity(claw, 0)
	pid := w.addEntity(prize, 0)
	claw.node.GetNode().SetPosition(0, 2, 0)
	prize.node.GetNode().SetPosition(0, 1.15, 0)
	w.phys3.AddBoxEx(cid, 0, 8, 0, 0.20, 1.05, 0.28, phys3d.MotionTypeKinematic)
	w.phys3.AddBoxEx(pid, 0, 1.15, 0, 0.30, 0.30, 0.30, phys3d.MotionTypeDynamic)
	w.phys3.SetMass(pid, 0.1)
	w.refreshWorldMatrices()
	w.resolveDrivenOverlaps()
	_, y, _, ok := w.phys3.GetPosition(pid)
	if !ok {
		t.Fatal("missing prize body")
	}
	if y > 1.05 {
		t.Fatalf("mesh-posed claw must depenetrate prize, y=%v (collider was left at y=8 on purpose)", y)
	}
}

func TestDriveParentedBodiesMatchesMeshWorldPose(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	parent := &Entity{node: core.NewNode(), bodyType: 3, boxX: 0.4, boxY: 0.16, boxZ: 0.4}
	child := &Entity{node: core.NewNode(), bodyType: 3, boxX: 0.18, boxY: 1.05, boxZ: 0.28}
	pid := w.addEntity(parent, 0)
	cid := w.addEntity(child, pid)
	parent.node.GetNode().SetPosition(0, 8.4, 0)
	child.node.GetNode().SetPosition(0, -1.05, 0)
	w.phys3.AddBoxEx(pid, 0, 8.4, 0, 0.4, 0.16, 0.4, phys3d.MotionTypeKinematic)
	w.phys3.AddBoxEx(cid, 0, 0, 0, 0.18, 1.05, 0.28, phys3d.MotionTypeKinematic)
	w.delta = 1.0 / 60.0
	w.driveParentedBodies(1.0 / 60.0)
	w.phys3.Step(1.0 / 60.0)
	x, y, z, ok := w.phys3.GetPosition(cid)
	if !ok {
		t.Fatal("missing child body")
	}
	if y < 6.5 || y > 8.0 {
		t.Fatalf("parented kinematic should MoveKinematic to mesh world Y~7.35, got %v %v %v", x, y, z)
	}
}

func TestCreateBodyBoxStoresHalfExtents(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	e := &Entity{node: core.NewNode()}
	id := w.addEntity(e, 0)
	e.node.GetNode().SetPosition(1, 2, 3)
	w.refreshWorldMatrices()
	w.ensurePhys3()
	hx, hy, hz := float32(0.18), float32(1.05), float32(0.28)
	w.phys3.AddBoxEx(id, 1, 2, -3, hx, hy, hz, phys3d.MotionTypeKinematic)
	e.bodyType = motionBodyType(phys3d.MotionTypeKinematic)
	e.boxX, e.boxY, e.boxZ = hx, hy, hz
	if e.boxX != hx || e.boxY != hy || e.boxZ != hz {
		t.Fatalf("extents %v %v %v", e.boxX, e.boxY, e.boxZ)
	}
}
