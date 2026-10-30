package runtime

import (
	"testing"

	"moonbasic/internal/value"
)

func TestRestingAndSeparatedContacts(t *testing.T) {
	w := readyWorld()
	ground, err := w.Call("createcube", nil)
	if err != nil {
		t.Fatal(err)
	}
	gid := ground.Int()
	if _, err := w.Call("positionentity", []value.Value{
		value.Num(float64(gid)), value.Num(0), value.Num(0), value.Num(0),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("collide", []value.Value{value.Num(float64(gid)), value.Num(0)}); err != nil {
		t.Skip(err)
	}

	ball, err := w.Call("createsphere", nil)
	if err != nil {
		t.Fatal(err)
	}
	bid := ball.Int()
	if _, err := w.Call("positionentity", []value.Value{
		value.Num(float64(bid)), value.Num(0), value.Num(2.2), value.Num(0),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("collide", []value.Value{value.Num(float64(bid)), value.Num(2)}); err != nil {
		t.Fatal(err)
	}

	var hit value.Value
	for i := 0; i < 180; i++ {
		if _, err := w.Call("updateworld", nil); err != nil {
			t.Fatal(err)
		}
		hit, err = w.Call("entitycollided", []value.Value{value.Num(float64(bid))})
		if err != nil {
			t.Fatal(err)
		}
		if hit.Int() == gid {
			break
		}
	}
	if hit.Int() != gid {
		t.Fatalf("ball never reported the ground, got %v", hit.Int())
	}
	cx, err := w.Call("collisionx", []value.Value{value.Num(float64(bid))})
	if err != nil {
		t.Fatal(err)
	}
	cy, err := w.Call("collisiony", []value.Value{value.Num(float64(bid))})
	if err != nil {
		t.Fatal(err)
	}
	if cy.Num < 0.5 || cy.Num > 1.5 {
		t.Fatalf("contact point %v %v", cx.Num, cy.Num)
	}
	n, err := w.Call("countcollisions", []value.Value{value.Num(float64(bid))})
	if err != nil {
		t.Fatal(err)
	}
	if n.Int() != 1 {
		t.Fatalf("count %v", n.Int())
	}
	other, err := w.Call("collisionentity", []value.Value{value.Num(float64(bid))})
	if err != nil || other.Int() != gid {
		t.Fatalf("collision entity %v %v", other, err)
	}

	for i := 0; i < 30; i++ {
		if _, err := w.Call("updateworld", nil); err != nil {
			t.Fatal(err)
		}
	}
	hit, err = w.Call("entitycollided", []value.Value{value.Num(float64(bid))})
	if err != nil {
		t.Fatal(err)
	}
	if hit.Int() != gid {
		t.Fatalf("resting contact dropped, got %v", hit.Int())
	}

	if _, err := w.Call("positionentity", []value.Value{
		value.Num(float64(bid)), value.Num(0), value.Num(12), value.Num(0),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("resetentity", []value.Value{value.Num(float64(bid))}); err != nil {
		t.Fatal(err)
	}
	// Kinematic move: dynamic bodies are driven by physics, so lift the body
	// through the physics position after the visual move.
	w.phys3.SetPosition(bid, 0, 12, 0)
	w.phys3.SetVelocity(bid, 0, 0, 0)
	if _, err := w.Call("updateworld", nil); err != nil {
		t.Fatal(err)
	}
	hit, err = w.Call("entitycollided", []value.Value{value.Num(float64(bid))})
	if err != nil {
		t.Fatal(err)
	}
	other, err = w.Call("collisionentity", []value.Value{value.Num(float64(bid))})
	if err != nil {
		t.Fatal(err)
	}
	if hit.Int() != 0 || other.Int() != 0 {
		t.Fatalf("separated ball still colliding hit=%v other=%v", hit.Int(), other.Int())
	}
}
