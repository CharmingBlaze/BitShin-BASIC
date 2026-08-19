package runtime

import (
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"

	"bitshinbasic/internal/phys3d"
	"bitshinbasic/internal/value"
)

func TestCreateBodyCylinderAndConvex(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ready = true
	cmds := w.commandTable()
	cyl := w.meshEnt(geometry.NewCylinder(0.4, 1.6, 8, 1, true, true), 0)
	cylE := w.ents[cyl]
	cylE.node.GetNode().SetPosition(0, 1, 0)
	w.refreshWorldMatrices()
	if _, err := cmds["createbodycylinder"]([]value.Value{value.Num(float64(cyl)), value.Num(0.8), value.Num(0.4), value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	if cylE.bodyType != 2 {
		t.Fatalf("static cylinder bodyType %d", cylE.bodyType)
	}
	cube := w.meshEnt(geometry.NewCube(1), 0)
	cubeE := w.ents[cube]
	cubeE.node.GetNode().SetPosition(3, 1, 0)
	w.refreshWorldMatrices()
	got, err := cmds["createbodyconvex"]([]value.Value{value.Num(float64(cube)), value.Num(0)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Num == 0 {
		t.Fatal("convex hull body failed")
	}
}

func TestGrabThrowProjectileBeamBone(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ready = true
	cmds := w.commandTable()
	w.ensurePhys3()
	hand := &Entity{node: core.NewNode(), bodyType: 3, yaw: 0}
	box := &Entity{node: core.NewNode(), bodyType: 1}
	hid := w.addEntity(hand, 0)
	bid := w.addEntity(box, 0)
	hand.node.GetNode().SetPosition(0, 1, 0)
	box.node.GetNode().SetPosition(0.5, 1, 0)
	w.phys3.AddBoxEx(hid, 0, 1, 0, 0.2, 0.2, 0.2, phys3d.MotionTypeKinematic)
	w.phys3.AddBoxEx(bid, 0.5, 1, 0, 0.2, 0.2, 0.2, phys3d.MotionTypeDynamic)
	w.phys3.SetMass(bid, 0.4)
	w.refreshWorldMatrices()
	got, err := cmds["grab"]([]value.Value{value.Num(float64(hid)), value.Num(float64(bid))})
	if err != nil {
		t.Fatal(err)
	}
	if got.Num != float64(bid) {
		t.Fatalf("grab returned %v", got.Num)
	}
	held, _ := cmds["grabbedentity"]([]value.Value{value.Num(float64(hid))})
	if held.Num != float64(bid) {
		t.Fatalf("GrabbedEntity %v", held.Num)
	}
	cmds["dropgrab"]([]value.Value{value.Num(float64(hid))})

	bolt := &Entity{node: core.NewNode(), yaw: 0}
	pid := w.addEntity(bolt, 0)
	bolt.node.GetNode().SetPosition(0, 2, 0)
	cmds["createprojectile"]([]value.Value{value.Num(float64(pid)), value.Num(10), value.Num(0), value.Num(2)})
	startX, startY, startZ := w.blitzPosOf(pid)
	_ = startX
	_ = startY
	w.delta = 1.0 / 30.0
	w.tickProjectiles(1.0 / 30.0)
	_, _, z1 := w.blitzPosOf(pid)
	if z1 <= startZ {
		t.Fatalf("projectile should move +Z, startZ=%v z=%v", startZ, z1)
	}

	dot := &Entity{node: core.NewNode()}
	did := w.addEntity(dot, 0)
	beam, err := cmds["createbeam"]([]value.Value{value.Num(float64(hid)), value.Num(float64(did)), value.Num(0.1)})
	if err != nil || beam.Num == 0 {
		t.Fatalf("createbeam %v %v", beam, err)
	}
	w.tickBeams()

	root := core.NewNode()
	bone := core.NewNode()
	bone.SetName("RightHand")
	bone.SetPosition(0, 1.2, 0)
	root.Add(bone)
	mesh := &Entity{node: root}
	gun := &Entity{node: core.NewNode()}
	mid := w.addEntity(mesh, 0)
	gid := w.addEntity(gun, 0)
	cmds["attachtobone"]([]value.Value{value.Num(float64(gid)), value.Num(float64(mid)), value.Str("RightHand")})
	if p := gun.node.GetNode().Parent(); p == nil || p.GetNode() != bone {
		t.Fatal("gun should parent to RightHand node")
	}
}
