package runtime

import (
	"testing"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/value"
)

func TestRopeBuildsSaggingConstrainedChain(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	a := &Entity{node: core.NewNode()}
	b := &Entity{node: core.NewNode()}
	aID := w.addEntity(a, 0)
	bID := w.addEntity(b, 0)
	w.phys3.AddBox(aID, 0, 3, 0, 0.3, 0.3, 0.3, false)
	w.phys3.AddBox(bID, 0, 3, 10, 0.3, 0.3, 0.3, false)

	ropeID := w.createRope(aID, bID, ropeVec{}, ropeVec{}, 12, 8, 0.03)
	r := w.ropes[ropeID]
	if ropeID == 0 || r == nil {
		t.Fatal("CreateRope should return a live rope")
	}
	if len(r.pos) != 7 || len(r.links) != 8 {
		t.Fatalf("8 rope segments should create 7 nodes and 8 links, got nodes=%d links=%d", len(r.pos), len(r.links))
	}
	midY := r.pos[len(r.pos)/2].y
	if midY >= 2.8 {
		t.Fatalf("slack rope should begin with visible gravity sag, middle y=%v", midY)
	}

	for i := 0; i < 120; i++ {
		w.phys3.Step(1.0 / 60.0)
		w.simulateRopes(1.0 / 60.0)
	}
	pts, ok := w.ropePoints(r)
	if !ok {
		t.Fatal("rope points disappeared after stepping")
	}
	maxLink := r.length/float32(r.segments)*1.35 + 0.05
	for i := 1; i < len(pts); i++ {
		if d := ropeDistance(pts[i-1], pts[i]); d > maxLink {
			t.Fatalf("constraint allowed rope link %d to stretch: %v > %v", i-1, d, maxLink)
		}
	}

	r.stiffness, r.lineDamp, r.maxForce = 1000, 100, 1000
	w.phys3.SetPosition(bID, 0, 3, 13)
	w.applyRopeForces()
	if tension := w.ropeTension(r); tension < 0.7 {
		t.Fatalf("nearly taut rope should report high tension, got %v", tension)
	}
	w.freeRope(ropeID)
	if w.ropes[ropeID] != nil {
		t.Fatal("FreeRope should remove the rope")
	}
}

func TestRopeSupportsHeavyDynamicEndpoint(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	weight := &Entity{node: core.NewNode()}
	weightID := w.addEntity(weight, 0)
	w.phys3.AddBox(weightID, 0, 3.5, 8, 0.7, 0.7, 0.7, true)
	w.phys3.SetMass(weightID, 18)
	ropeID := w.createRope(0, weightID, ropeVec{0, 9, 8}, ropeVec{0, 0.7, 0}, 7.2, 12, 0.04)
	r := w.ropes[ropeID]
	if r == nil {
		t.Fatal("anchored rope was not created")
	}
	r.stiffness, r.lineDamp, r.maxForce = 5000, 700, 12000
	for i := 0; i < 240; i++ {
		w.applyRopeForces()
		w.phys3.Step(1.0 / 60.0)
		w.simulateRopes(1.0 / 60.0)
	}
	_, y, _, _ := w.phys3.GetPosition(weightID)
	if y < 0.65 || y > 2.2 {
		t.Fatalf("rope should support the heavy endpoint near its maximum length, y=%v", y)
	}
	w.freeRope(ropeID)
}

func TestPositionEntityTeleportsRootPhysicsBody(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ready = true
	w.ensurePhys3()
	e := &Entity{node: core.NewNode()}
	id := w.addEntity(e, 0)
	w.phys3.AddBox(id, 0, 1, 0, 0.5, 0.5, 0.5, true)
	_, err := w.Call("positionentity", []value.Value{
		value.Num(float64(id)), value.Num(4), value.Num(5), value.Num(6),
	})
	if err != nil {
		t.Fatal(err)
	}
	x, y, z, _ := w.phys3.GetPosition(id)
	if x != 4 || y != 5 || z != 6 {
		t.Fatalf("PositionEntity should teleport a root physics body, got (%v,%v,%v)", x, y, z)
	}
}
