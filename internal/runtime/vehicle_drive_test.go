package runtime

import (
	"math"
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"

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

func TestBoatBuoyancyStability(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	boat := &Entity{node: core.NewNode()}
	bid := w.addEntity(boat, 0)
	boat.node.GetNode().SetPosition(0, 1.4, 6)
	boat.node.GetNode().SetScale(1.8, 0.55, 3.4)
	c := w.bindCtrl("boat", []value.Value{value.Num(float64(bid))}, 1.6, 0.35, 3.2, 800)
	c.thrustMax = 9000

	wb := &waterBody{
		waves: [4]gerstnerWave{
			{dirX: 0.9, dirZ: 0.3, steep: 0.28, amp: 0.45, lambda: 18, speed: 1.1},
		},
		nWaves: 1,
	}
	if w.waters == nil {
		w.waters = map[int]*waterBody{}
	}
	w.waters[1] = wb

	for i := 0; i < 300; i++ {
		wb.time += 1.0 / 60.0
		w.applyBoatForces(c, 0, 0)
		w.phys3.Step(1.0 / 60.0)
		w.syncPhys3Pose()
	}
	e := w.ents[bid]
	p := e.node.GetNode().Position()
	if math32.Abs(p.Y) > 2.0 {
		t.Fatalf("Boat should float near y=0, got y=%v", p.Y)
	}
	if math32.Abs(e.pitch) > 25.0 || math32.Abs(e.roll) > 25.0 {
		t.Fatalf("Boat pitched/rolled excessively: pitch=%v roll=%v", e.pitch, e.roll)
	}
}

func TestWaterSkiStaysNearSurface(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	ski := &Entity{node: core.NewNode()}
	id := w.addEntity(ski, 0)
	ski.node.GetNode().SetPosition(0, 0.4, 0)
	ski.node.GetNode().SetScale(0.22, 0.12, 1.05)
	c := w.bindCtrl("waterski", []value.Value{value.Num(float64(id))}, 0.22, 0.12, 1.05, 85)
	c.thrustMax = 4200
	wb := &waterBody{
		waves: [4]gerstnerWave{
			{dirX: 0.9, dirZ: 0.3, steep: 0.2, amp: 0.2, lambda: 14, speed: 1},
		},
		nWaves: 1,
	}
	if w.waters == nil {
		w.waters = map[int]*waterBody{}
	}
	w.waters[1] = wb
	for i := 0; i < 180; i++ {
		wb.time += 1.0 / 60.0
		w.applyWaterSkiForces(c, 1, 0.2, 0)
		w.phys3.Step(1.0 / 60.0)
		w.syncPhys3Pose()
	}
	_, y, z, _ := w.phys3.GetPosition(id)
	if y < -3 || y > 4 {
		t.Fatalf("waterski should plane near the surface, y=%v", y)
	}
	if z < 0.5 {
		t.Fatalf("tow throttle should pull the ski forward, z=%v", z)
	}
}

func TestJetSkiPlanesWithoutViolentJolts(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	ski := &Entity{node: core.NewNode()}
	id := w.addEntity(ski, 0)
	ski.node.GetNode().SetPosition(0, 1.1, 0)
	c := w.bindCtrl("jetski", []value.Value{
		value.Num(float64(id)), value.Num(0.6), value.Num(0.35), value.Num(1.55),
	}, 0.6, 0.35, 1.55, 380)
	c.thrustMax = 15500
	wb := &waterBody{
		waves: [4]gerstnerWave{
			{dirX: 0.94, dirZ: 0.25, steep: 0.3, amp: 0.48, lambda: 24, speed: 0.92},
			{dirX: -0.2, dirZ: 0.98, steep: 0.24, amp: 0.24, lambda: 12, speed: 1.32},
		},
		nWaves: 2,
	}
	w.waters = map[int]*waterBody{1: wb}

	maxVerticalSpeed := float32(0)
	for i := 0; i < 360; i++ {
		wb.time += 1.0 / 60.0
		steer := float32(0)
		if i > 210 {
			steer = 0.45
		}
		w.applyJetSkiForces(c, 1, steer)
		w.phys3.Step(1.0 / 60.0)
		_, vertical, _, _ := w.phys3.GetVelocity(id)
		if math32.Abs(vertical) > maxVerticalSpeed {
			maxVerticalSpeed = math32.Abs(vertical)
		}
	}
	x, y, z, _ := w.phys3.GetPosition(id)
	if z < 12 {
		t.Fatalf("jetski should plane forward, position=(%v,%v,%v)", x, y, z)
	}
	if y < -2.5 || y > 3.5 {
		t.Fatalf("jetski should remain near the water surface, y=%v", y)
	}
	if maxVerticalSpeed > 11 {
		t.Fatalf("distributed hull should prevent violent vertical jolts, max vy=%v", maxVerticalSpeed)
	}
}

func TestBodyVelocityReturnsMagnitude(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ready = true
	w.ensurePhys3()
	body := &Entity{node: core.NewNode()}
	id := w.addEntity(body, 0)
	w.phys3.AddBox(id, 0, 2, 0, 0.5, 0.5, 0.5, true)
	w.phys3.SetVelocity(id, 3, 4, 12)
	got, err := w.Call("bodyvelocity", []value.Value{value.Num(float64(id))})
	if err != nil {
		t.Fatalf("BodyVelocity failed: %v", err)
	}
	if math32.Abs(float32(got.Number())-13) > 0.01 {
		t.Fatalf("BodyVelocity should return vector magnitude 13, got %v", got.Number())
	}
}

func TestControllerAcceptsExplicitMass(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	body := &Entity{node: core.NewNode()}
	id := w.addEntity(body, 0)
	c := w.bindCtrl("boat", []value.Value{
		value.Num(float64(id)), value.Num(1.8), value.Num(0.5), value.Num(3.4), value.Num(1250),
	}, 1.6, 0.35, 3.2, 800)
	if c.mass != 1250 {
		t.Fatalf("controller should use explicit mass, got %v", c.mass)
	}
}

func TestWaterSkiTowRopeIsDampedAndCapped(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	boat := &Entity{node: core.NewNode()}
	skier := &Entity{node: core.NewNode()}
	boatID := w.addEntity(boat, 0)
	skiID := w.addEntity(skier, 0)
	boat.node.GetNode().SetPosition(0, 0.7, 18)
	skier.node.GetNode().SetPosition(0, 0.35, 0)
	boatCtrl := w.bindCtrl("boat", []value.Value{value.Num(float64(boatID))}, 1.6, 0.35, 3.2, 800)
	boatCtrl.thrustMax = 9000
	skiCtrl := w.bindCtrl("waterski", []value.Value{value.Num(float64(skiID))}, 0.22, 0.12, 1.05, 85)
	skiCtrl.thrustMax = 4200
	wb := &waterBody{
		waves:  [4]gerstnerWave{{dirX: 0.9, dirZ: 0.3, steep: 0.2, amp: 0.24, lambda: 14, speed: 1}},
		nWaves: 1,
	}
	w.waters = map[int]*waterBody{1: wb}
	maxSpeed := float32(0)
	for i := 0; i < 360; i++ {
		wb.time += 1.0 / 60.0
		w.applyBoatForces(boatCtrl, 0.72, 0.12)
		w.applyWaterSkiForces(skiCtrl, 0.8, 0.35, boatID)
		w.phys3.Step(1.0 / 60.0)
		vx, vy, vz, _ := w.phys3.GetVelocity(skiID)
		speed := float32(math.Sqrt(float64(vx*vx + vy*vy + vz*vz)))
		if speed > maxSpeed {
			maxSpeed = speed
		}
	}
	sx, sy, sz, _ := w.phys3.GetPosition(skiID)
	bx, _, bz, _ := w.phys3.GetPosition(boatID)
	dx, dz := bx-sx, bz-sz
	distance := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	if sy < -2.5 || sy > 3.5 {
		t.Fatalf("towed skier should remain near the surface, y=%v", sy)
	}
	if distance > 28 {
		t.Fatalf("damped rope should keep skier within tow range, distance=%v", distance)
	}
	if maxSpeed > 45 {
		t.Fatalf("capped rope should prevent launch-speed jolts, max speed=%v", maxSpeed)
	}
}

func TestHelicopterHover(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	heli := &Entity{node: core.NewNode()}
	hid := w.addEntity(heli, 0)
	heli.node.GetNode().SetPosition(0, 5, 0)
	heli.node.GetNode().SetScale(1.4, 0.5, 2.4)
	c := w.bindCtrl("heli", []value.Value{value.Num(float64(hid))}, 1.4, 0.5, 2.4, 700)
	c.thrustMax = 14000

	for i := 0; i < 180; i++ {
		_, err := w.Call("updatehelicopter", []value.Value{
			value.Num(float64(hid)),
			value.Num(0), // neutral hover collective
			value.Num(0),
			value.Num(0),
			value.Num(0),
		})
		if err != nil {
			t.Fatalf("updatehelicopter error: %v", err)
		}
		w.phys3.Step(1.0 / 60.0)
		w.syncPhys3Pose()
		if i%30 == 0 {
			vx, vy, vz, _ := w.phys3.GetVelocity(hid)
			px, py, pz, _ := w.phys3.GetPosition(hid)
			t.Logf("Frame %d: pos=(%v,%v,%v) vel=(%v,%v,%v)", i, px, py, pz, vx, vy, vz)
		}
	}
	e := w.ents[hid]
	p := e.node.GetNode().Position()
	if p.Y < 3.0 || p.Y > 7.0 {
		t.Fatalf("Helicopter should stay hovering near y=5, got y=%v", p.Y)
	}
}

func TestDroneHover(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	drone := &Entity{node: core.NewNode()}
	did := w.addEntity(drone, 0)
	drone.node.GetNode().SetPosition(0, 3, 0)
	drone.node.GetNode().SetScale(0.45, 0.12, 0.45)
	c := w.bindCtrl("drone", []value.Value{value.Num(float64(did))}, 0.45, 0.12, 0.45, 8)
	c.thrustMax = 220

	for i := 0; i < 120; i++ {
		_, err := w.Call("updatedrone", []value.Value{
			value.Num(float64(did)),
			value.Num(0), // neutral throttle
			value.Num(0),
			value.Num(0),
			value.Num(0),
		})
		if err != nil {
			t.Fatalf("updatedrone error: %v", err)
		}
		w.phys3.Step(1.0 / 60.0)
		w.syncPhys3Pose()
	}
	e := w.ents[did]
	p := e.node.GetNode().Position()
	if p.Y < 1.5 || p.Y > 4.5 {
		t.Fatalf("Drone should stay hovering near y=3, got y=%v", p.Y)
	}
}

func TestHovercraftCushion(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	hover := &Entity{node: core.NewNode()}
	hid := w.addEntity(hover, 0)
	hover.node.GetNode().SetPosition(0, 0.5, 0)
	hover.node.GetNode().SetScale(1.8, 0.25, 2.4)
	c := w.bindCtrl("hover", []value.Value{value.Num(float64(hid))}, 1.8, 0.25, 2.4, 500)
	c.thrustMax = 7000

	for i := 0; i < 120; i++ {
		_, err := w.Call("updatehovercraft", []value.Value{
			value.Num(float64(hid)),
			value.Num(1.0), // throttle forward
			value.Num(0),
		})
		if err != nil {
			t.Fatalf("updatehovercraft error: %v", err)
		}
		w.phys3.Step(1.0 / 60.0)
		w.syncPhys3Pose()
	}
	e := w.ents[hid]
	p := e.node.GetNode().Position()
	if p.Y < 0.0 || p.Y > 2.0 {
		t.Fatalf("Hovercraft should stay riding cushion near y=0.5, got y=%v", p.Y)
	}
	_, _, z, _ := w.phys3.GetPosition(hid)
	if z < 1.0 {
		t.Fatalf("Hovercraft should accelerate forward in phys coords, got z=%v", z)
	}
}

func TestSubmarineDive(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	sub := &Entity{node: core.NewNode()}
	sid := w.addEntity(sub, 0)
	sub.node.GetNode().SetPosition(0, -2, 0)
	sub.node.GetNode().SetScale(1.2, 0.7, 4)
	c := w.bindCtrl("sub", []value.Value{value.Num(float64(sid))}, 1.2, 0.7, 4, 1400)
	c.thrustMax = 11000

	for i := 0; i < 120; i++ {
		_, err := w.Call("updatesubmarine", []value.Value{
			value.Num(float64(sid)),
			value.Num(1.0), // forward throttle
			value.Num(0),
			value.Num(0), // neutral depth
		})
		if err != nil {
			t.Fatalf("updatesubmarine error: %v", err)
		}
		w.phys3.Step(1.0 / 60.0)
		w.syncPhys3Pose()
	}
	_, _, z, _ := w.phys3.GetPosition(sid)
	if z < 1.0 {
		t.Fatalf("Submarine should propel forward in phys coords, got z=%v", z)
	}
}

func TestSpaceshipRCS(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.ensurePhys3()
	ship := &Entity{node: core.NewNode()}
	sid := w.addEntity(ship, 0)
	ship.node.GetNode().SetPosition(0, 0, 0)
	ship.node.GetNode().SetScale(1.4, 0.6, 2.2)
	c := w.bindCtrl("ship", []value.Value{value.Num(float64(sid))}, 1.4, 0.6, 2.2, 400)
	c.thrustMax = 8000

	for i := 0; i < 60; i++ {
		_, err := w.Call("updatespaceship", []value.Value{
			value.Num(float64(sid)),
			value.Num(1.0), // main thruster
			value.Num(0),
			value.Num(0),
			value.Num(0),
		})
		if err != nil {
			t.Fatalf("updatespaceship error: %v", err)
		}
		w.phys3.Step(1.0 / 60.0)
		w.syncPhys3Pose()
	}
	_, _, z, _ := w.phys3.GetPosition(sid)
	if z < 1.0 {
		t.Fatalf("Spaceship should accelerate forward in phys coords, got z=%v", z)
	}
}
