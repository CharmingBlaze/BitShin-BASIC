package phys2d

import "testing"

func TestCircleFalls(t *testing.T) {
	s := New()
	s.SetGravity(0, -200)
	s.AddBox(1, 0, -2, 20, 0.5, 0, false)
	s.AddCircle(2, 0, 4, 0.5, 1, true)
	for i := 0; i < 90; i++ {
		s.Step(1.0 / 60.0)
	}
	_, y, ok := s.GetPosition(2)
	if !ok {
		t.Fatal("missing body")
	}
	if y > 2 {
		t.Fatalf("ball should have fallen, y=%v", y)
	}
}

func TestPinJointHoldsDistance(t *testing.T) {
	s := New()
	s.SetGravity(0, -200)
	s.AddBox(1, 0, 0, 2, 2, 0, false)
	s.AddCircle(2, 4, 0, 0.5, 1, true)
	id := s.AddPin(1, 2, 0, 0, 0, 0)
	if id == 0 {
		t.Fatal("pin joint")
	}
	for i := 0; i < 90; i++ {
		s.Step(1.0 / 60.0)
	}
	x, y, ok := s.GetPosition(2)
	if !ok {
		t.Fatal("missing body")
	}
	dist := mathHypot(x, y)
	if dist < 2 || dist > 6 {
		t.Fatalf("pin should keep the circle nearby, dist=%v pos=%v,%v", dist, x, y)
	}
}

func TestSpringAndSlide(t *testing.T) {
	s := New()
	s.AddBox(1, 0, 0, 1, 1, 0, false)
	s.AddCircle(2, 3, 0, 0.4, 1, true)
	if s.AddSpring(1, 2, 3, 80, 8, 0, 0, 0, 0) == 0 {
		t.Fatal("spring")
	}
	s.AddCircle(3, 1, 2, 0.4, 1, true)
	if s.AddSlide(1, 3, 0.5, 4, 0, 0, 0, 0) == 0 {
		t.Fatal("slide")
	}
	for i := 0; i < 20; i++ {
		s.Step(1.0 / 60.0)
	}
}
