package runtime

import (
	"testing"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/value"
)

func TestUpdateCarThrottleMovesBody(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	ground := &Entity{node: core.NewNode()}
	car := &Entity{node: core.NewNode()}
	gid := w.addEntity(ground, 0)
	cid := w.addEntity(car, 0)
	ground.node.GetNode().SetPosition(0, 0, 10)
	car.node.GetNode().SetPosition(0, 1.2, 10)
	car.node.GetNode().SetScale(1.1, 0.35, 2.0)
	w.phys3.AddBox(gid, 0, 0, 10, 40, 0.25, 40, false)
	c := w.bindCtrl("car", []value.Value{value.Num(float64(cid))}, 0.9, 0.35, 1.8, 1200)
	c.native = w.phys3.CreateWheeledVehicle(c.id, c.hx, c.hy, c.hz)
	_, _, startZ, ok := w.phys3.GetPosition(cid)
	if !ok {
		t.Fatal("car body missing")
	}
	for i := 0; i < 90; i++ {
		w.updateNamed("car", []value.Value{
			value.Num(float64(cid)),
			value.Num(0),
			value.Num(1),
			value.Num(0),
		})
		w.phys3.Step(1.0 / 60.0)
	}
	_, _, z, ok := w.phys3.GetPosition(cid)
	if !ok {
		t.Fatal("car body missing after drive")
	}
	if z-startZ < 1.5 {
		t.Fatalf("throttle should drive the car forward, startZ=%v z=%v native=%d", startZ, z, c.native)
	}
}

func TestUpdateTankThrottleMovesBody(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	ground := &Entity{node: core.NewNode()}
	tank := &Entity{node: core.NewNode()}
	gid := w.addEntity(ground, 0)
	tid := w.addEntity(tank, 0)
	ground.node.GetNode().SetPosition(0, 0, 10)
	tank.node.GetNode().SetPosition(0, 1.1, 10)
	tank.node.GetNode().SetScale(1.3, 0.45, 2.4)
	w.phys3.AddBox(gid, 0, 0, 10, 40, 0.25, 40, false)
	c := w.bindCtrl("tank", []value.Value{value.Num(float64(tid))}, 1.3, 0.45, 2.4, 1800)
	c.native = w.phys3.CreateTrackedVehicle(c.id, c.hx, c.hy, c.hz)
	_, _, startZ, ok := w.phys3.GetPosition(tid)
	if !ok {
		t.Fatal("tank body missing")
	}
	for i := 0; i < 90; i++ {
		w.updateNamed("tank", []value.Value{
			value.Num(float64(tid)),
			value.Num(0),
			value.Num(1),
			value.Num(0),
		})
		w.phys3.Step(1.0 / 60.0)
	}
	_, _, z, ok := w.phys3.GetPosition(tid)
	if !ok {
		t.Fatal("tank body missing after drive")
	}
	if z-startZ < 1.2 {
		t.Fatalf("throttle should drive the tank forward, startZ=%v z=%v native=%d", startZ, z, c.native)
	}
}
