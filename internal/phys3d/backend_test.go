package phys3d

import "testing"

func TestDefaultBackend(t *testing.T) {
	w := New()
	defer w.Close()
	if w.Backend() != BackendJolt && w.Backend() != BackendFallback {
		t.Fatalf("backend %q", w.Backend())
	}
	t.Log("backend", w.Backend())
}

func TestSoftwareFallbackFirstClass(t *testing.T) {
	w := newFallback()
	defer w.Close()
	if w.Backend() != BackendFallback {
		t.Fatalf("want %q got %q", BackendFallback, w.Backend())
	}
	w.AddSphere(1, 0, 2, 0, 0.5, true)
	w.AddGround(2, 0)
	w.SetGravity(0, -20, 0)
	for i := 0; i < 40; i++ {
		w.Step(1.0 / 60.0)
	}
	_, y, _, ok := w.GetPosition(1)
	if !ok {
		t.Fatal("missing body")
	}
	if y > 1.8 {
		t.Fatalf("fallback sphere should fall, y=%v", y)
	}
	id, _, _, _, hit := w.Raycast(0, 10, 0, 0, -20, 0)
	if !hit {
		t.Fatal("fallback raycast missed")
	}
	if id != 1 && id != 2 {
		t.Fatalf("raycast entity %d", id)
	}
}

func TestFallbackImpulseAndCharacter(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddSphere(1, 0, 2, 0, 0.5, true)
	w.ApplyImpulse(1, 4, 0, 0)
	for i := 0; i < 30; i++ {
		w.Step(1.0 / 60.0)
	}
	x, _, _, ok := w.GetPosition(1)
	if !ok {
		t.Fatal("missing body")
	}
	if x < 1 {
		t.Fatalf("impulse should move sphere, x=%v", x)
	}
	w.AddCharacter(3, 0, 2, 0, 0.9, 0.4)
	w.SetVelocity(3, 2, 0, 0)
	w.SetGravity(0, -20, 0)
	for i := 0; i < 40; i++ {
		w.Step(1.0 / 60.0)
	}
	_, y, _, ok := w.GetPosition(3)
	if !ok {
		t.Fatal("missing character")
	}
	if y > 1.6 {
		t.Fatalf("kinematic character should settle, y=%v", y)
	}
	if w.CharacterBackend() != CharacterKinematic {
		t.Fatalf("character backend %q", w.CharacterBackend())
	}
	if w.CharacterGroundState(3) != 0 {
		t.Fatalf("settled character ground state %d", w.CharacterGroundState(3))
	}
	nx, ny, nz := w.CharacterGroundNormal(3)
	if ny < 0.5 {
		t.Fatalf("ground normal %v %v %v", nx, ny, nz)
	}
	w.SetCharacterShape(3, "capsule", 0.9, 0.4)
	w.AddCharacterController(4, 0, 3, 0, 1.8, 0.4, 45, 75)
	if w.CharacterGroundState(4) != 3 && w.CharacterGroundState(4) != 0 {
		t.Fatalf("new controller state %d", w.CharacterGroundState(4))
	}
}

func TestKinematicBoxPushesDynamicBox(t *testing.T) {
	w := New()
	defer w.Close()
	if w.Backend() != BackendJolt {
		t.Skip("needs Jolt")
	}
	w.SetGravity(0, 0, 0)
	w.AddBoxEx(1, 0, 1, 0, 0.5, 0.5, 0.5, MotionTypeKinematic)
	w.AddBoxEx(2, 1.2, 1, 0, 0.5, 0.5, 0.5, MotionTypeDynamic)
	w.SetMass(2, 0.2)
	w.SetFriction(1, 3)
	w.SetFriction(2, 3)
	dt := float32(1.0 / 60.0)
	for i := 0; i < 45; i++ {
		w.MoveKinematic(1, 0.08*float32(i+1), 1, 0, 0, 0, 0, 1, dt)
		w.Step(dt)
	}
	x, _, _, ok := w.GetPosition(2)
	if !ok {
		t.Fatal("missing dynamic")
	}
	if x < 1.4 {
		t.Fatalf("kinematic should push dynamic, x=%v backend=%s", x, w.Backend())
	}
}
