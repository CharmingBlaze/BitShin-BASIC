package runtime

import (
	"math"
	"testing"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/value"
)

func TestRefreshWorldMatricesUpdatesParentedPose(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	parent := &Entity{node: core.NewNode()}
	child := &Entity{node: core.NewNode()}
	pid := w.addEntity(parent, 0)
	_ = w.addEntity(child, pid)
	parent.node.GetNode().SetPosition(0, 8.4, 0)
	child.node.GetNode().SetPosition(0, -1.05, 0)

	stale := worldPos(child.node.GetNode())
	if math.Abs(float64(stale.Y+1.05)) > 0.05 {
		t.Fatalf("child WorldPosition without ancestor update should stay near local Y, got %v", stale)
	}

	w.refreshWorldMatrices()
	fresh := worldPos(child.node.GetNode())
	if math.Abs(float64(fresh.Y-7.35)) > 0.02 {
		t.Fatalf("after refreshWorldMatrices world Y want 7.35 got %v", fresh)
	}
}

func TestEntityPositionOptionalGlobalCoordinates(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ready = true
	parent := &Entity{node: core.NewNode()}
	child := &Entity{node: core.NewNode()}
	pid := w.addEntity(parent, 0)
	cid := w.addEntity(child, pid)
	parent.node.GetNode().SetPosition(4, 8, 12)
	child.node.GetNode().SetPosition(1, 2, 3)

	localY, err := w.Call("entityy", []value.Value{value.Num(float64(cid))})
	if err != nil {
		t.Fatal(err)
	}
	worldY, err := w.Call("entityy", []value.Value{value.Num(float64(cid)), value.Num(1)})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(localY.Number()-2) > 0.001 || math.Abs(worldY.Number()-10) > 0.001 {
		t.Fatalf("EntityY local/world mismatch: local=%v world=%v", localY.Number(), worldY.Number())
	}
}
