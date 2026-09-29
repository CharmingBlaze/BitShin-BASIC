package runtime

import (
	"testing"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/value"
)

func readyWorld() *World {
	w := New(".")
	w.ready = true
	w.scene = core.NewNode()
	return w
}

func TestCollideMatchesMeshAndMaterial(t *testing.T) {
	w := readyWorld()
	cube, err := w.Call("createcube", nil)
	if err != nil {
		t.Fatal(err)
	}
	id := cube.Int()
	if _, err := w.Call("collide", []value.Value{value.Num(float64(id)), value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	e := w.ents[id]
	if e.bodyType != 2 {
		t.Fatalf("STATIC body type %d", e.bodyType)
	}
	if e.boxX < 0.5 || e.boxY < 0.5 || e.boxZ < 0.5 {
		t.Fatalf("box extents %v %v %v", e.boxX, e.boxY, e.boxZ)
	}

	ball, err := w.Call("createsphere", nil)
	if err != nil {
		t.Fatal(err)
	}
	bid := ball.Int()
	if _, err := w.Call("setphysicsmaterial", []value.Value{value.Num(float64(bid)), value.Str("ice")}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("collide", []value.Value{value.Num(float64(bid)), value.Num(2)}); err != nil {
		t.Fatal(err)
	}
	be := w.ents[bid]
	if be.collKind != 2 || be.bodyType != 1 {
		t.Fatalf("sphere collider kind %d type %d", be.collKind, be.bodyType)
	}
	if be.physMat != "ice" {
		t.Fatalf("material stored %q", be.physMat)
	}
	fr, err := w.Call("getfriction", []value.Value{value.Num(float64(bid))})
	if err != nil {
		t.Fatal(err)
	}
	if fr.Num > 0.05 {
		t.Fatalf("ice friction %v", fr.Num)
	}
	name, err := w.Call("getphysicsmaterial", []value.Value{value.Num(float64(bid))})
	if err != nil || name.Str != "ice" {
		t.Fatalf("name %v err %v", name, err)
	}
	if _, err := w.Call("setmaterial", []value.Value{value.Num(float64(bid)), value.Str("rubber")}); err != nil {
		t.Fatal(err)
	}
	if w.ents[bid].physMat != "rubber" {
		t.Fatalf("SetMaterial string did not set physics material, got %q", w.ents[bid].physMat)
	}
}

func TestRaycastHitStruct(t *testing.T) {
	w := readyWorld()
	ground, err := w.Call("createcube", nil)
	if err != nil {
		t.Fatal(err)
	}
	gid := ground.Int()
	if _, err := w.Call("positionentity", []value.Value{value.Num(float64(gid)), value.Num(0), value.Num(0), value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("collide", []value.Value{value.Num(float64(gid)), value.Str("static")}); err != nil {
		t.Fatal(err)
	}
	hit, err := w.Call("raycasthit", []value.Value{
		value.Num(0), value.Num(5), value.Num(0),
		value.Num(0), value.Num(-10), value.Num(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if hit.Fields["hit"].Num != 1 || int(hit.Fields["entity"].Num) != gid {
		t.Fatalf("hit %+v", hit.Fields)
	}
	if hit.Fields["ny"].Num < 0.5 {
		t.Fatalf("normal y %v", hit.Fields["ny"].Num)
	}
	if hit.Fields["fraction"].Num <= 0 || hit.Fields["fraction"].Num >= 1 {
		t.Fatalf("fraction %v", hit.Fields["fraction"].Num)
	}
	all, err := w.Call("raycastall", []value.Value{
		value.Num(0), value.Num(5), value.Num(0),
		value.Num(0), value.Num(-10), value.Num(0),
	})
	if err != nil || len(all.Elems) < 1 {
		t.Fatalf("raycastall %v err %v", all, err)
	}
	if w.pickID != 0 {
		t.Fatal("RaycastHit should not write PickedEntity")
	}
	id, err := w.Call("raycast", []value.Value{
		value.Num(0), value.Num(5), value.Num(0),
		value.Num(0), value.Num(-10), value.Num(0),
	})
	if err != nil || id.Int() != gid {
		t.Fatalf("raycast %v err %v", id, err)
	}
	ny, _ := w.Call("getraynormaly", nil)
	if ny.Num < 0.5 {
		t.Fatalf("picked normal y %v", ny.Num)
	}
}

func TestFlipStepsPhysicsOnce(t *testing.T) {
	w := readyWorld()
	ball, err := w.Call("createsphere", nil)
	if err != nil {
		t.Fatal(err)
	}
	id := ball.Int()
	args := []value.Value{value.Num(float64(id))}
	if _, err := w.Call("positionentity", append(append([]value.Value{}, args...), value.Num(0), value.Num(5), value.Num(0))); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("collide", args); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setbodyvelocity", append(append([]value.Value{}, args...), value.Num(0), value.Num(-20), value.Num(0))); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("updateworld", nil); err != nil {
		t.Fatal(err)
	}
	y1, err := w.Call("entityy", args)
	if err != nil {
		t.Fatal(err)
	}
	if y1.Num > 4.9 {
		t.Fatalf("UpdateWorld did not move the body, y=%v", y1.Num)
	}
	if _, err := w.Call("flip", nil); err != nil {
		t.Fatal(err)
	}
	y2, err := w.Call("entityy", args)
	if err != nil {
		t.Fatal(err)
	}
	if y1.Num-y2.Num > 0.05 {
		t.Fatalf("Flip stepped again: %v -> %v", y1.Num, y2.Num)
	}

	ball2, err := w.Call("createsphere", nil)
	if err != nil {
		t.Fatal(err)
	}
	id2 := ball2.Int()
	a2 := []value.Value{value.Num(float64(id2))}
	if _, err := w.Call("positionentity", append(append([]value.Value{}, a2...), value.Num(3), value.Num(5), value.Num(0))); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("collide", a2); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("setbodyvelocity", append(append([]value.Value{}, a2...), value.Num(0), value.Num(-20), value.Num(0))); err != nil {
		t.Fatal(err)
	}
	y0, _ := w.Call("entityy", a2)
	if _, err := w.Call("flip", nil); err != nil {
		t.Fatal(err)
	}
	y3, _ := w.Call("entityy", a2)
	if y0.Num-y3.Num < 0.05 {
		t.Fatalf("Flip alone should step physics: %v -> %v", y0.Num, y3.Num)
	}
}
