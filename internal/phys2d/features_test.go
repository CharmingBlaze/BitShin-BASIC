package phys2d

import (
	"math"
	"testing"
)

// These tests use the script convention: Y grows down, gravity (0, 900), pixels.

const step = 1.0 / 60.0

func screenSpace() *Space {
	s := New()
	s.SetGravity(0, 900)
	// Floor: top surface at y = 380.
	s.AddBox(1, 0, 400, 2000, 40, 0, false)
	return s
}

func run(s *Space, n int) {
	for i := 0; i < n; i++ {
		s.Step(step)
	}
}

func TestLayerFiltering(t *testing.T) {
	s := screenSpace()
	s.AddCircle(2, -50, 300, 10, 1, true)
	s.AddCircle(3, 50, 300, 10, 1, true)
	if !s.SetCollisionLayer(3, 2) || !s.SetLayerCollides(2, 1, false) {
		t.Fatal("layer setup")
	}
	if s.SetCollisionLayer(3, 33) || s.SetLayerCollides(0, 1, true) {
		t.Fatal("out-of-range layers should be refused")
	}
	run(s, 90)
	_, ya, _ := s.GetPosition(2)
	_, yb, _ := s.GetPosition(3)
	if math.Abs(ya-370) > 2 {
		t.Fatalf("layer-1 ball should rest on the floor, y=%v", ya)
	}
	if yb < 420 {
		t.Fatalf("layer-2 ball should fall through the floor, y=%v", yb)
	}
	// Collides follows the filter too.
	s.SetPosition(3, 50, 395)
	s.collectHits()
	if s.Collides(3, 1) {
		t.Fatal("filtered pair reported as colliding")
	}
	// A mask can narrow a default body: ball 2 stops touching layer 1.
	s.SetLayerCollides(2, 1, true)
	s.SetCollisionMask(2, 0)
	s.SetPosition(2, -50, 300)
	run(s, 90)
	if _, y, _ := s.GetPosition(2); y < 420 {
		t.Fatalf("mask 0 should pass through everything, y=%v", y)
	}
}

func TestSensorEnterExit(t *testing.T) {
	s := New()
	s.SetGravity(0, 900)
	s.AddBox(10, 0, 200, 100, 100, 0, false)
	s.SetSensor(10, true)
	s.AddCircle(2, 0, 0, 5, 1, true)
	var entered, exited, inside bool
	for i := 0; i < 120; i++ {
		s.Step(step)
		for _, id := range s.SensorEntered(10) {
			if id == 2 {
				entered = true
			}
		}
		if s.SensorHas(10, 2) {
			inside = true
			if len(s.SensorBodies(10)) != 1 {
				t.Fatalf("sensor bodies %v", s.SensorBodies(10))
			}
		}
		for _, id := range s.SensorExited(10) {
			if id == 2 {
				if !entered {
					t.Fatal("exit before enter")
				}
				exited = true
			}
		}
	}
	if !entered || !inside || !exited {
		t.Fatalf("entered=%v inside=%v exited=%v", entered, inside, exited)
	}
	// The ball fell straight through: a sensor pushes nothing.
	if _, y, _ := s.GetPosition(2); y < 300 {
		t.Fatalf("ball should pass the sensor, y=%v", y)
	}
}

func TestKinematicPushes(t *testing.T) {
	s := New()
	s.SetGravity(0, 900)
	s.AddBox(1, 0, 0, 40, 40, 0, false)
	if !s.SetKinematic(1, true) || !s.IsKinematic(1) {
		t.Fatal("kinematic")
	}
	s.SetVelocity(1, 100, 0)
	s.AddBox(2, 60, 0, 20, 20, 1, true)
	s.SetGravity(0, 0)
	s.ApplyForce(1, 0, 1e6)
	run(s, 60)
	kx, ky, _ := s.GetPosition(1)
	vx, vy, _ := s.GetVelocity(1)
	if math.Abs(kx-100) > 1 || math.Abs(ky) > 1e-6 || vx != 100 || vy != 0 {
		t.Fatalf("kinematic body should keep its own motion: pos %v,%v vel %v,%v", kx, ky, vx, vy)
	}
	if dx, _, _ := s.GetPosition(2); dx < 125 {
		t.Fatalf("dynamic box should be pushed ahead, x=%v", dx)
	}
	// Back to dynamic keeps the stored mass, so it now falls.
	s.SetGravity(0, 900)
	s.SetKinematic(1, false)
	run(s, 30)
	if _, y, _ := s.GetPosition(1); y < 50 {
		t.Fatalf("dynamic again should fall, y=%v", y)
	}
}

func TestCapsuleRests(t *testing.T) {
	s := screenSpace()
	s.AddCapsule(2, 0, 300, 20, 60, 1, true)
	s.AddCapsule(3, 100, 300, 60, 20, 1, true)
	run(s, 120)
	_, y, _ := s.GetPosition(2)
	_, y2, _ := s.GetPosition(3)
	if math.Abs(y2-370) > 2 {
		t.Fatalf("lying capsule should rest at 370, y=%v", y2)
	}
	// The upright one may topple; it must still sit on the floor, not in it.
	if y < 339 || y > 371 {
		t.Fatalf("upright capsule out of place, y=%v", y)
	}
}

func TestShapeCasts(t *testing.T) {
	s := screenSpace()
	s.AddBox(5, 0, 200, 50, 50, 0, false)
	s.SetSensor(5, true)
	h := s.CircleCast(0, 0, 0, 500, 10, 0)
	if !h.OK || h.ID != 1 {
		t.Fatalf("circle cast should skip the sensor and hit the floor: %+v", h)
	}
	if cy := h.Fraction * 500; math.Abs(cy-370) > 1 {
		t.Fatalf("circle centre at impact %v", cy)
	}
	if math.Abs(h.NY+1) > 1e-6 || math.Abs(h.Y-380) > 1 {
		t.Fatalf("normal/point %+v", h)
	}
	b := s.BoxCast(0, 0, 0, 500, 40, 20, 0)
	if !b.OK || b.ID != 1 {
		t.Fatalf("box cast %+v", b)
	}
	if cy := b.Fraction * 500; math.Abs(cy-370) > 0.5 {
		t.Fatalf("box centre at impact %v", cy)
	}
	if math.Abs(b.NY+1) > 1e-3 || math.Abs(b.Y-380) > 1 {
		t.Fatalf("box normal/point %+v", b)
	}
	if s.LastHit() != b {
		t.Fatal("last hit")
	}
	if m := s.BoxCast(0, 0, 0, 300, 40, 20, 0); m.OK || m.Fraction != 1 {
		t.Fatalf("short cast should miss: %+v", m)
	}
	if ig := s.CircleCast(0, 0, 0, 500, 10, 1); ig.OK {
		t.Fatalf("ignored floor still hit: %+v", ig)
	}
	if r := s.RaycastHit(0, 0, 0, 500); !r.OK || math.Abs(r.Y-380) > 1e-6 {
		t.Fatalf("ray %+v", r)
	}
}

func newChar(s *Space, x, y float64) *Character {
	c := s.AddCharacter(50, x, y, 24, 48)
	return c
}

func TestCharacterGroundAndJump(t *testing.T) {
	s := screenSpace()
	c := newChar(s, 0, 300)
	run(s, 60)
	if !c.OnGround() || c.GroundBody() != 1 {
		t.Fatal("should be on the floor")
	}
	_, y, _ := s.GetPosition(50)
	if math.Abs(y-356) > 0.5 {
		t.Fatalf("feet should sit on the floor, y=%v", y)
	}
	// Standing still: no drift, no jitter.
	for i := 0; i < 30; i++ {
		s.Step(step)
		x, y2, _ := s.GetPosition(50)
		if math.Abs(x) > 0.01 || math.Abs(y2-y) > 0.2 {
			t.Fatalf("jitter at rest: %v,%v", x, y2)
		}
	}
	for i := 0; i < 30; i++ {
		c.Move(1)
		s.Step(step)
	}
	x, _, _ := s.GetPosition(50)
	if math.Abs(x-c.Speed*0.5) > c.Speed*0.05 {
		t.Fatalf("walked %v, want about %v", x, c.Speed*0.5)
	}
	c.Jump()
	minY := y
	for i := 0; i < 90; i++ {
		s.Step(step)
		if _, yy, _ := s.GetPosition(50); yy < minY {
			minY = yy
		}
		if i == 2 && c.OnGround() {
			t.Fatal("still grounded after a jump")
		}
	}
	if h := y - minY; math.Abs(h-c.JumpHeight) > c.JumpHeight*0.1 {
		t.Fatalf("jump height %v, want about %v", h, c.JumpHeight)
	}
	if !c.OnGround() {
		t.Fatal("should land again")
	}
}

func TestCharacterCoyoteAndBuffer(t *testing.T) {
	setup := func() (*Space, *Character) {
		s := New()
		s.SetGravity(0, 900)
		s.AddBox(1, 0, 400, 200, 40, 0, false) // ledge ends at x = 100
		s.AddBox(2, 0, 2000, 4000, 40, 0, false)
		c := newChar(s, 80, 356)
		run(s, 10)
		return s, c
	}
	// Walk off the ledge and count airborne steps.
	walkOff := func(s *Space, c *Character) {
		for c.OnGround() {
			c.Move(1)
			s.Step(step)
		}
	}
	s, c := setup()
	walkOff(s, c)
	s.Step(step) // still inside coyote time (0.1 s)
	c.Jump()
	s.Step(step)
	_, vy, _ := s.GetVelocity(50)
	if vy > -100 {
		t.Fatalf("coyote jump should launch, vy=%v", vy)
	}
	s, c = setup()
	walkOff(s, c)
	run(s, 12) // past coyote time
	c.Jump()
	s.Step(step)
	if _, vy, _ := s.GetVelocity(50); vy < 0 {
		t.Fatalf("late jump should not fire, vy=%v", vy)
	}
	// Jump buffering: press just before landing.
	s = screenSpace()
	c = newChar(s, 0, 300)
	for !c.OnGround() {
		if _, y, _ := s.GetPosition(50); y > 345 {
			c.Jump()
		}
		s.Step(step)
		if _, vy, _ := s.GetVelocity(50); vy < -100 {
			return
		}
	}
	s.Step(step)
	if _, vy, _ := s.GetVelocity(50); vy > -100 {
		t.Fatalf("buffered jump should fire on landing, vy=%v", vy)
	}
}

func TestCharacterSlopeAndWall(t *testing.T) {
	s := screenSpace()
	// A 30 degree ramp rising to the right, from (0,380) to (200,~264.5).
	rise := 200 * math.Tan(30*math.Pi/180)
	s.AddPoly(3, 0, 0, [][2]float64{{0, 380}, {200, 380}, {200, 380 - rise}}, 0, false)
	c := newChar(s, 100, 300)
	run(s, 60)
	if !c.OnGround() {
		t.Fatal("should stand on the ramp")
	}
	x0, y0, _ := s.GetPosition(50)
	run(s, 60)
	x1, y1, _ := s.GetPosition(50)
	if math.Abs(x1-x0) > 0.5 || math.Abs(y1-y0) > 0.5 {
		t.Fatalf("slid on a walkable slope: %v,%v -> %v,%v", x0, y0, x1, y1)
	}
	// Walk up and back down without losing the ground.
	for i := 0; i < 20; i++ {
		c.Move(1)
		s.Step(step)
		if i > 2 && !c.OnGround() {
			t.Fatalf("lost ground walking up at step %d", i)
		}
	}
	for i := 0; i < 40; i++ {
		c.Move(-1)
		s.Step(step)
		if i > 2 && !c.OnGround() {
			t.Fatalf("lost ground walking down at step %d", i)
		}
	}
	// Too steep: max slope 20 means the ramp is not ground and it slides off.
	c.MaxSlope = 20
	s.SetPosition(50, 120, 300)
	s.SetVelocity(50, 0, 0)
	run(s, 30)
	if c.OnGround() && c.GroundBody() == 3 {
		t.Fatal("steep ramp counted as ground")
	}

	// Wall: pushing into it in the air must not hold the character up.
	w := screenSpace()
	w.AddBox(4, 100, 200, 20, 400, 0, false)
	c = w.AddCharacter(50, 70, 100, 24, 48)
	prev := 100.0
	for i := 0; i < 30; i++ {
		c.Move(1)
		w.Step(step)
		if c.OnGround() && c.GroundBody() == 4 {
			t.Fatal("wall counted as ground")
		}
	}
	if _, y, _ := w.GetPosition(50); y < prev+60 {
		t.Fatalf("stuck on the wall, y=%v", y)
	}
}

func TestOneWayPlatformAndMovingPlatform(t *testing.T) {
	s := screenSpace()
	// One-way platform with its top at y = 300, above the character.
	s.AddBox(6, 0, 305, 200, 10, 0, false)
	s.SetOneWay(6, true)
	c := newChar(s, 0, 356)
	run(s, 5)
	c.JumpHeight = 120
	c.Jump()
	run(s, 120)
	_, y, _ := s.GetPosition(50)
	if !c.OnGround() || c.GroundBody() != 6 || math.Abs(y-276) > 0.5 {
		t.Fatalf("should jump up through and land on the one-way platform: ground=%d y=%v", c.GroundBody(), y)
	}

	// Moving kinematic platform carries a standing character.
	m := screenSpace()
	m.AddBox(7, 0, 300, 100, 10, 0, false)
	m.SetKinematic(7, true)
	c = m.AddCharacter(50, 0, 260, 24, 48)
	run(m, 30)
	m.SetVelocity(7, 60, 0)
	run(m, 60)
	px, _, _ := m.GetPosition(7)
	cx, _, _ := m.GetPosition(50)
	if !c.OnGround() || math.Abs(cx-px) > 1 {
		t.Fatalf("character should ride the platform: platform x=%v char x=%v", px, cx)
	}
}

func TestNewCommandsLeaveOldWorldsAlone(t *testing.T) {
	s := screenSpace()
	s.AddCircle(2, 0, 300, 10, 1, true)
	run(s, 60)
	if s.ext.layersOn || s.ext.oneWayOn {
		t.Fatal("extras switched on without being used")
	}
	s.Remove(2)
	if s.Has(2) {
		t.Fatal("remove")
	}
	run(s, 2)
}

func TestCharacterPushesButCannotPassCrate(t *testing.T) {
	s := screenSpace()
	s.AddBox(4, 200, 340, 20, 80, 0, false) // wall, left face at x = 190
	s.AddBox(5, 100, 365, 30, 30, 1, true)  // crate
	c := newChar(s, 0, 356)
	for i := 0; i < 180; i++ {
		c.Move(1)
		s.Step(step)
	}
	cx, _, _ := s.GetPosition(5)
	px, _, _ := s.GetPosition(50)
	if cx < 170 {
		t.Fatalf("crate should be pushed to the wall, x=%v", cx)
	}
	if px > cx-26 {
		t.Fatalf("character went into the crate: char x=%v crate x=%v", px, cx)
	}
}
