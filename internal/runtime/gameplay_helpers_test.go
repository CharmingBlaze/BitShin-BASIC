package runtime

import (
	"math"
	"testing"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/phys3d"
	"bitshinbasic/internal/value"
)

func TestGameplayMathHelpers(t *testing.T) {
	w := New(".")
	cmds := w.commandTable()

	// Lerp
	res, err := cmds["lerp"]([]value.Value{value.Num(10), value.Num(20), value.Num(0.5)})
	if err != nil || math.Abs(res.Num-15.0) > 1e-4 {
		t.Fatalf("lerp failed: got %v, err %v", res, err)
	}

	// LerpAngle across 360 wrap
	res, err = cmds["lerpangle"]([]value.Value{value.Num(350), value.Num(10), value.Num(0.5)})
	if err != nil || (math.Abs(res.Num-0.0) > 1e-4 && math.Abs(res.Num-360.0) > 1e-4) {
		t.Fatalf("lerpangle wrap failed: got %v, err %v", res, err)
	}

	// SmoothDamp
	res, err = cmds["smoothdamp"]([]value.Value{value.Num(0), value.Num(100), value.Num(0.2), value.Num(500)})
	if err != nil || res.Num <= 0 || res.Num > 100 {
		t.Fatalf("smoothdamp failed: got %v, err %v", res, err)
	}

	// MoveTowards
	res, err = cmds["movetowards"]([]value.Value{value.Num(5), value.Num(20), value.Num(3)})
	if err != nil || math.Abs(res.Num-8.0) > 1e-4 {
		t.Fatalf("movetowards failed: got %v, err %v", res, err)
	}
	res, _ = cmds["movetowards"]([]value.Value{value.Num(19), value.Num(20), value.Num(3)})
	if math.Abs(res.Num-20.0) > 1e-4 {
		t.Fatalf("movetowards clamp to target failed: got %v", res)
	}

	// Clamp
	res, _ = cmds["clamp"]([]value.Value{value.Num(55), value.Num(0), value.Num(10)})
	if math.Abs(res.Num-10.0) > 1e-4 {
		t.Fatalf("clamp max failed: got %v", res)
	}
	res, _ = cmds["clamp"]([]value.Value{value.Num(-12), value.Num(0), value.Num(10)})
	if math.Abs(res.Num-0.0) > 1e-4 {
		t.Fatalf("clamp min failed: got %v", res)
	}

	// WrapAngle
	res, _ = cmds["wrapangle"]([]value.Value{value.Num(370)})
	if math.Abs(res.Num-10.0) > 1e-4 {
		t.Fatalf("wrapangle failed: got %v", res)
	}

	// Distance3D
	res, _ = cmds["distance3d"]([]value.Value{value.Num(0), value.Num(0), value.Num(0), value.Num(3), value.Num(4), value.Num(0)})
	if math.Abs(res.Num-5.0) > 1e-4 {
		t.Fatalf("distance3d failed: got %v", res)
	}

	// Distance2D
	res, _ = cmds["distance2d"]([]value.Value{value.Num(0), value.Num(0), value.Num(6), value.Num(8)})
	if math.Abs(res.Num-10.0) > 1e-4 {
		t.Fatalf("distance2d failed: got %v", res)
	}
}

func TestTimeScale(t *testing.T) {
	w := New(".")
	cmds := w.commandTable()

	cmds["settimescale"]([]value.Value{value.Num(0.5)})
	res, _ := cmds["gettimescale"](nil)
	if math.Abs(res.Num-0.5) > 1e-4 {
		t.Fatalf("gettimescale failed: got %v want 0.5", res)
	}

	cmds["timescale"]([]value.Value{value.Num(2.0)})
	res, _ = cmds["timescale"](nil)
	if math.Abs(res.Num-2.0) > 1e-4 {
		t.Fatalf("timescale setter/getter failed: got %v want 2.0", res)
	}
}

func TestTweens(t *testing.T) {
	w := New(".")
	cmds := w.commandTable()

	// Entity for tween
	id := w.createCubeMesh(nil)

	cmds["tweenposition"]([]value.Value{value.Num(float64(id)), value.Num(10), value.Num(20), value.Num(30), value.Num(1.0), value.Str("out")})
	if len(w.actTweens) != 1 {
		t.Fatalf("tweenposition failed to register tween: %d", len(w.actTweens))
	}

	w.delta = 0.5
	w.updateWorld()

	p := w.ents[id].node.GetNode().Position()
	gx, gy, gz := fromG3N(p.X, p.Y, p.Z)
	if gx <= 0 || gy <= 0 || gz <= 0 {
		t.Fatalf("tweenposition step failed: pos %v,%v,%v", gx, gy, gz)
	}

	// Finish tween
	w.delta = 0.6
	w.updateWorld()
	if len(w.actTweens) != 0 {
		t.Fatalf("completed tween should be removed, got len %d", len(w.actTweens))
	}
}

func TestFollowPathAndExplode(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ready = true
	cmds := w.commandTable()
	id := w.createCubeMesh(nil)
	cmds["followpath"]([]value.Value{
		value.Num(float64(id)), value.Num(10),
		value.Num(0), value.Num(0), value.Num(0),
		value.Num(10), value.Num(0), value.Num(0),
	})
	w.delta = 0.5
	w.updatePathFollows(0.5)
	p := w.ents[id].node.GetNode().Position()
	gx, _, _ := fromG3N(p.X, p.Y, p.Z)
	if gx < 4 {
		t.Fatalf("follow path should advance, x=%v", gx)
	}

	w.ensurePhys3()
	ball := w.createCubeMesh(nil)
	w.ents[ball].bodyType = 1
	w.ents[ball].node.GetNode().SetPosition(2, 1, 0)
	w.refreshWorldMatrices()
	w.phys3.AddBoxEx(ball, 2, 1, 0, 0.3, 0.3, 0.3, phys3d.MotionTypeDynamic)
	n, err := cmds["explode"]([]value.Value{value.Num(2), value.Num(1), value.Num(0), value.Num(4), value.Num(12)})
	if err != nil {
		t.Fatal(err)
	}
	if n.Num < 1 {
		t.Fatalf("explode hit count %v", n.Num)
	}
}

func TestEntityMaterial(t *testing.T) {
	w := New(".")
	w.ready = true
	e := &Entity{node: core.NewNode(), mat: w.newMat()}
	id := w.addEntity(e, 0)
	mat := value.StructOf("material", []string{"r", "g", "b", "tex", "shine", "spec"})
	mat.SetField("r", value.Num(10))
	mat.SetField("g", value.Num(20))
	mat.SetField("b", value.Num(30))
	mat.SetField("shine", value.Num(0.25))
	if _, err := w.Call("entitymaterial", []value.Value{value.Num(float64(id)), mat}); err != nil {
		t.Fatal(err)
	}
	if e.tint.R < 0.03 || e.tint.R > 0.05 {
		t.Fatalf("tint R %v", e.tint.R)
	}
}
