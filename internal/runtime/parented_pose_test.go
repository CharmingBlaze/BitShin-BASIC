package runtime

import (
	"math"
	"testing"

	"github.com/g3n/engine/core"
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
