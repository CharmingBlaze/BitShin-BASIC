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

func TestBuoyancyMeshSensorQueries(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, -9.81, 0)
	w.AddBox(1, 0, 0.5, 0, 0.5, 0.5, 0.5, true)
	w.SetMass(1, 10)
	ok := w.ApplyBuoyancyImpulse(1, 0, 2, 0, 0, 1, 0, 1.2, 0.5, 0.3, 0, 0, 0, 1.0/60)
	if w.Backend() == BackendJolt && !ok {
		t.Fatal("jolt buoyancy should run")
	}
	w.AddSensorBox(2, 5, 1, 0, 1, 1, 1, MotionTypeKinematic)
	w.SetSensor(2, true)
	id, hit := w.OverlapPoint(5, 1, 0)
	if w.Backend() == BackendFallback {
		if !hit && id == 0 {
			t.Log("fallback overlap is approximate")
		}
	} else if !hit {
		t.Fatal("sensor overlap missed")
	}
	w.AddBox(3, 0, 0, 8, 4, 0.2, 4, false)
	_, _, _, _, castOK := w.ShapeCast(0.3, 0.3, 0.3, 0, 4, 8, 0, -8, 0)
	if !castOK {
		t.Fatal("shapecast missed ground")
	}
	verts := [][3]float32{{0, 0, 0}, {2, 0, 0}, {0, 0, 2}}
	idx := []int32{0, 1, 2}
	w.AddMesh(4, verts, idx, MotionTypeStatic)
	samples := make([]float32, 16*16)
	w.AddHeightField(5, samples, 16, -8, 0, -8, 1, 1, 1)
	w.OffsetCenterOfMass(1, 0, -0.2, 0)
}

func TestOptimizeBroadPhaseManyBodies(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, -20, 0)
	w.AddBox(1, 0, 0, 0, 20, 0.25, 20, false)
	for i := 0; i < 40; i++ {
		w.AddSphere(10+i, float32(i%8)-3.5, 3+float32(i)*0.15, float32(i/8)-2, 0.2, true)
	}
	w.OptimizeBroadPhase()
	id, _, _, _, hit := w.Raycast(0, 20, 0, 0, -40, 0)
	if !hit {
		t.Fatal("raycast after optimize missed")
	}
	if id == 0 {
		t.Fatal("raycast hit nothing")
	}
	for i := 0; i < 30; i++ {
		w.Step(1.0 / 60.0)
	}
	_, y, _, ok := w.GetPosition(10)
	if !ok {
		t.Fatal("missing sphere")
	}
	if y > 4 {
		t.Fatalf("pile should fall after optimize, y=%v", y)
	}
}

func TestClothSheetDrapes(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, -9.81, 0)
	w.AddCloth(1, 0, 4, 0, 2, 2, 8, 8, 1, 0.05, 0.5, 1)
	if w.ClothVertexCount(1) == 0 && w.Backend() == BackendJolt {
		t.Fatal("jolt cloth produced no vertices")
	}
	_, y0, _, ok := w.GetPosition(1)
	if !ok {
		t.Fatal("cloth body missing")
	}
	for i := 0; i < 40; i++ {
		w.Step(1.0 / 60.0)
	}
	_, y1, _, ok := w.GetPosition(1)
	if !ok {
		t.Fatal("cloth body missing after step")
	}
	if y1 > y0-0.08 {
		t.Fatalf("cloth COM should drop, y %v -> %v backend=%s", y0, y1, w.Backend())
	}
}

func clothWorldY(w World, id int, buf []float32) (hi, lo float32, ok bool) {
	n := w.ClothVertices(id, buf)
	if n < 4 {
		return 0, 0, false
	}
	_, comY, _, posOK := w.GetPosition(id)
	if !posOK {
		return 0, 0, false
	}
	// Both Jolt and the Verlet sheet return vertices relative to the body center.
	hi, lo = float32(-1e9), float32(1e9)
	for i := 0; i < n; i++ {
		y := buf[i*3+1] + comY
		if y > hi {
			hi = y
		}
		if y < lo {
			lo = y
		}
	}
	return hi, lo, true
}

func TestClothWindMovesFreeEdge(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, -9.81, 0)
	w.AddCloth(1, 0, 4, 0, 2, 1.6, 8, 6, 1, 0.04, 0.2, 1)
	buf := make([]float32, 8*6*3)
	if w.ClothVertices(1, buf) < 8*6 {
		t.Fatalf("cloth has no vertices, backend=%s", w.Backend())
	}
	for i := 0; i < 90; i++ {
		w.ApplyClothWind(1, 0, 0, 8, uint32(i), 1)
		w.Step(1.0 / 60.0)
	}
	n := w.ClothVertices(1, buf)
	_, comY, comZ, _ := w.GetPosition(1)
	hiY, loY := float32(-1e9), float32(1e9)
	for i := 0; i < n; i++ {
		y := buf[i*3+1] + comY
		if y > hiY {
			hiY = y
		}
		if y < loY {
			loY = y
		}
	}
	mid := (hiY + loY) * 0.5
	pinZ, hemZ := float32(0), float32(0)
	pins, hems := 0, 0
	for i := 0; i < n; i++ {
		y := buf[i*3+1] + comY
		z := buf[i*3+2] + comZ
		if y >= mid {
			pinZ += z
			pins++
		} else {
			hemZ += z
			hems++
		}
	}
	if pins == 0 || hems == 0 {
		t.Fatalf("expected a pinned top and a free hem, pins=%d hems=%d", pins, hems)
	}
	pinZ /= float32(pins)
	hemZ /= float32(hems)
	if hemZ-pinZ < 0.25 {
		t.Fatalf("wind should push the free edge off the pin, pinZ=%v hemZ=%v backend=%s", pinZ, hemZ, w.Backend())
	}
}

func TestClothPinHoldsAndRestsOnFloor(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, -9.81, 0)
	w.AddBox(2, 0, 0.75, 0, 4, 2, 4, false)
	w.AddCloth(1, 0, 6, 0, 2, 8, 5, 8, 1, 0.05, 0.2, 1)
	buf := make([]float32, 5*8*3)
	hi0, lo0, ok := clothWorldY(w, 1, buf)
	if !ok {
		t.Fatalf("cloth has no vertices, backend=%s", w.Backend())
	}
	_, com0, _, _ := w.GetPosition(1)
	if lo0 > 0 {
		t.Fatalf("curtain should start through the floor, lo=%v hi=%v com=%v raw0=%v", lo0, hi0, com0, buf[1])
	}
	for i := 0; i < 120; i++ {
		w.Step(1.0 / 60.0)
	}
	hi, lo, ok := clothWorldY(w, 1, buf)
	if !ok {
		t.Fatal("cloth vertices missing after step")
	}
	_, comY, comZ, _ := w.GetPosition(1)
	n := w.ClothVertices(1, buf)
	inside := 0
	for i := 0; i < n; i++ {
		x := buf[i*3+0]
		y := buf[i*3+1] + comY
		z := buf[i*3+2] + comZ
		if x > -3 && x < 3 && z > -3 && z < 3 && y > -1.2 && y < 2.7 {
			inside++
		}
	}
	if hi < 5.2 {
		t.Fatalf("pinned edge should stay up, hi=%v lo=%v comY=%v backend=%s", hi, lo, comY, w.Backend())
	}
	if inside > 0 {
		t.Fatalf("cloth vertices still inside the floor: inside=%d hi=%v lo=%v", inside, hi, lo)
	}
}

func TestCylinderHullAndGrab(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddCylinder(1, 0, 1, 0, 0.8, 0.3, MotionTypeStatic)
	id, _, _, _, hit := w.Raycast(0, 4, 0, 0, -8, 0)
	if !hit || (id != 1 && w.Backend() == BackendJolt) {
		t.Fatalf("cylinder raycast id=%d hit=%v backend=%s", id, hit, w.Backend())
	}
	pts := [][3]float32{{0, 0, 0}, {1, 0, 0}, {0.5, 1, 0}, {0.5, 0.3, 0.8}}
	w.AddConvexHull(2, pts, 4, 1, 0, MotionTypeStatic)
	id, _, _, _, hit = w.Raycast(4, 5, 0, 0, -8, 0)
	if !hit {
		t.Fatal("convex hull raycast missed")
	}
	w.AddBoxEx(3, 0, 2, 4, 0.2, 0.2, 0.2, MotionTypeKinematic)
	w.AddBoxEx(4, 0.4, 2, 4, 0.2, 0.2, 0.2, MotionTypeDynamic)
	w.SetMass(4, 0.5)
	jid := w.CreateGrabJoint(3, 4, 0.4, 2, 4, 12, 1)
	if jid == 0 {
		return
	}
	dt := float32(1.0 / 60.0)
	for i := 0; i < 40; i++ {
		w.MoveKinematic(3, 0.06*float32(i+1), 2, 4, 0, 0, 0, 1, dt)
		w.Step(dt)
	}
	x, _, _, ok := w.GetPosition(4)
	if !ok {
		t.Fatal("grabbed body missing")
	}
	if x < 0.7 {
		t.Fatalf("grab should pull target, x=%v joint=%d backend=%s", x, jid, w.Backend())
	}
}

func TestCompoundOverlapExplodeAndJoints(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	parts := []CompoundPart{
		{Kind: CompoundBox, A: 0.3, B: 0.3, C: 0.3},
		{Kind: CompoundBox, Ox: 1.2, A: 0.2, B: 0.2, C: 0.2},
	}
	w.AddCompound(1, parts, 0, 1, 0, MotionTypeStatic)
	id, _, _, _, hit := w.Raycast(1.2, 4, 0, 0, -8, 0)
	if !hit {
		t.Fatal("compound offset part raycast missed")
	}
	if w.Backend() == BackendJolt && id != 1 {
		t.Fatalf("compound hit id=%d", id)
	}
	w.AddBoxEx(2, 5, 1, 0, 0.3, 0.3, 0.3, MotionTypeDynamic)
	all := w.OverlapSphereAll(5, 1, 0, 1, 8)
	if len(all) == 0 {
		t.Fatal("OverlapSphereAll missed nearby box")
	}
	jid := w.CreateFixedJoint(1, 2, 2.5, 1, 0)
	if jid == 0 {
		t.Fatal("fixed joint id 0")
	}
	_ = w.CreateConeJoint(1, 2, 2.5, 1, 0, 0, 1, 0, 40)
	_ = w.CreateSwingTwistJoint(1, 2, 2.5, 1, 0, 0, 1, 0, 40, 20)
	w.SetCollisionLayer(2, 1)
	w.SetLayerCollides(0, 1, false)
}

func TestFallbackBoxRaycastAABB(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.AddBox(1, 0, 0, 0, 4, 0.1, 0.1, false)
	id, _, _, _, hit := w.Raycast(6, 0, 0, -12, 0, 0)
	if !hit || id != 1 {
		t.Fatalf("thin box from the side: hit=%v id=%d", hit, id)
	}
	id, _, _, _, hit = w.Raycast(0, 2, 2, 0, 0, -4)
	if hit && id == 1 {
		t.Fatal("ray beside a thin box should miss the AABB")
	}
}

func TestFallbackFixedJointHolds(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddBox(1, 0, 1, 0, 0.3, 0.3, 0.3, false)
	w.AddBox(2, 2, 1, 0, 0.3, 0.3, 0.3, true)
	if w.CreateFixedJoint(1, 2, 1, 1, 0) == 0 {
		t.Fatal("joint")
	}
	for i := 0; i < 20; i++ {
		w.Step(1.0 / 60.0)
	}
	x, _, _, ok := w.GetPosition(2)
	if !ok {
		t.Fatal("missing")
	}
	if x < 1.4 || x > 2.6 {
		t.Fatalf("fixed joint should keep the pair close, x=%v", x)
	}
}

func TestFallbackSliderJointAxis(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddBox(1, 0, 1, 0, 0.3, 0.3, 0.3, false)
	w.AddBox(2, 1, 1, 0, 0.3, 0.3, 0.3, true)
	if w.CreateSliderJoint(1, 2, 0.5, 1, 0, 1, 0, 0) == 0 {
		t.Fatal("slider")
	}
	w.ApplyImpulse(2, 6, 0, 0)
	for i := 0; i < 25; i++ {
		w.Step(1.0 / 60.0)
	}
	x, y, _, ok := w.GetPosition(2)
	if !ok {
		t.Fatal("missing")
	}
	if x < 1.3 {
		t.Fatalf("slider should allow X travel, x=%v", x)
	}
	if y < 0.7 || y > 1.3 {
		t.Fatalf("slider should hold Y, y=%v", y)
	}
}

func TestFallbackShapeCastFatSweep(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.AddBox(1, 0, 0, 0, 0.2, 0.05, 0.2, false)
	_, _, _, _, rayHit := w.Raycast(0, 1, 0, 0, -0.4, 0)
	if rayHit {
		t.Fatal("short ray should miss the thin box")
	}
	id, _, _, _, ok := w.ShapeCast(0.2, 0.7, 0.2, 0, 1, 0, 0, -0.4, 0)
	if !ok || id != 1 {
		t.Fatalf("fat shapecast should hit thin box, ok=%v id=%d", ok, id)
	}
}

func TestFallbackPollContactsOverlap(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddSphere(1, 0, 1, 0, 0.6, true)
	w.AddSphere(2, 0.4, 1, 0, 0.6, true)
	w.Step(1.0 / 60)
	evs := w.PollContacts(8)
	if len(evs) == 0 {
		t.Fatal("overlapping spheres should report a contact")
	}
}

func TestFallbackRestitutionBounce(t *testing.T) {
	bounce := newFallback()
	defer bounce.Close()
	bounce.SetGravity(0, -20, 0)
	bounce.AddSphere(1, 0, 4, 0, 0.5, true)
	peak := float32(0)
	for i := 0; i < 90; i++ {
		bounce.Step(1.0 / 60)
		_, y, _, _ := bounce.GetPosition(1)
		if i > 40 && y > peak {
			peak = y
		}
	}
	if peak < 0.7 {
		t.Fatalf("default restitution should bounce off the floor, peak y=%v", peak)
	}

	dead := newFallback()
	defer dead.Close()
	dead.SetGravity(0, -20, 0)
	dead.AddSphere(1, 0, 4, 0, 0.5, true)
	dead.SetRestitution(1, 0)
	for i := 0; i < 90; i++ {
		dead.Step(1.0 / 60)
	}
	_, y, _, _ := dead.GetPosition(1)
	_, vy, _, _ := dead.GetVelocity(1)
	if y > 0.55 || vy > 0.05 {
		t.Fatalf("SetRestitution(0) should settle on the floor, y=%v vy=%v", y, vy)
	}
}

func TestFallbackConeJointLimitsSwing(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddBoxEx(1, 0, 0, 0, 0.2, 0.2, 0.2, MotionTypeStatic)
	w.AddSphere(2, 2, 0, 0, 0.3, true)
	if w.CreateConeJoint(1, 2, 0, 0, 0, 1, 0, 0, 25) == 0 {
		t.Fatal("cone")
	}
	w.SetPosition(2, 2, 4, 0)
	for i := 0; i < 12; i++ {
		w.Step(1.0 / 60)
	}
	_, y, _, _ := w.GetPosition(2)
	if y > 2.3 || y < 0.4 {
		t.Fatalf("cone should clamp swing off +X, y=%v", y)
	}
}

func TestFallbackHingeLimitAndMotor(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddBoxEx(1, 0, 0, 0, 0.2, 0.2, 0.2, MotionTypeStatic)
	w.AddSphere(2, 1.5, 0, 0, 0.2, true)
	jid := w.CreateHingeJoint(1, 2, 0, 0, 0, 0, 1, 0)
	if jid == 0 {
		t.Fatal("hinge")
	}
	w.SetHingeLimits(jid, -15, 15)
	w.SetPosition(2, 0, 0, 1.5)
	for i := 0; i < 20; i++ {
		w.Step(1.0 / 60)
	}
	_, _, z, _ := w.GetPosition(2)
	if z > 0.6 {
		t.Fatalf("hinge limit should stop a 90° swing, z=%v", z)
	}
	w.SetHingeLimits(jid, -90, 90)
	w.SetHingeMotor(jid, 70, 80)
	for i := 0; i < 80; i++ {
		w.Step(1.0 / 60)
	}
	_, _, z2, _ := w.GetPosition(2)
	if z2 > -0.35 && z2 < 0.35 {
		t.Fatalf("hinge motor should drive toward 70°, z=%v", z2)
	}
}

func TestFallbackCCDStopsTunnel(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddBoxEx(1, 0, 1, 0, 0.15, 2, 2, MotionTypeStatic)
	w.AddSphere(2, 8, 1, 0, 0.25, true)
	if w.SetCCD(2, true) != 1 {
		t.Fatal("SetCCD")
	}
	w.SetVelocity(2, -600, 0, 0)
	w.Step(1.0 / 60)
	x, _, _, _ := w.GetPosition(2)
	if x < 0 {
		t.Fatalf("CCD should stop before tunneling, x=%v", x)
	}
}

func TestFallbackDisableBodyCollision(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddSphere(1, 0, 1, 0, 0.6, true)
	w.AddSphere(2, 0.4, 1, 0, 0.6, true)
	w.DisableBodyCollision(1, 2)
	w.Step(1.0 / 60)
	if len(w.PollContacts(8)) != 0 {
		t.Fatal("disabled pair should not contact")
	}
}

func TestFallbackIntegratesRotation(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddSphere(1, 0, 2, 0, 0.4, true)
	w.SetAngularVelocity(1, 0, 3, 0)
	for i := 0; i < 20; i++ {
		w.Step(1.0 / 60)
	}
	_, y, _, qw, ok := w.GetRotation(1)
	if !ok || (y == 0 && qw == 1) {
		t.Fatalf("fallback rotation should integrate torque, qy=%v qw=%v", y, qw)
	}
}

func TestFallbackHingeFrictionDampsSwing(t *testing.T) {
	damp := newFallback()
	defer damp.Close()
	damp.SetGravity(0, 0, 0)
	damp.AddBoxEx(1, 0, 0, 0, 0.2, 0.2, 0.2, MotionTypeStatic)
	damp.AddSphere(2, 1, 0, 0, 0.2, true)
	jid := damp.CreateHingeJoint(1, 2, 0, 0, 0, 0, 1, 0)
	damp.SetHingeFriction(jid, 20)
	damp.SetVelocity(2, 0, 0, 8)
	for i := 0; i < 30; i++ {
		damp.Step(1.0 / 60)
	}
	_, _, vz, _ := damp.GetVelocity(2)
	if vz > 4 {
		t.Fatalf("hinge friction should cut planar speed, vz=%v", vz)
	}
}

func TestFallbackTwistStopsSpin(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddBoxEx(1, 0, 0, 0, 0.2, 0.2, 0.2, MotionTypeStatic)
	w.AddSphere(2, 1, 0, 0, 0.2, true)
	if w.CreateSwingTwistJoint(1, 2, 0, 0, 0, 1, 0, 0, 40, 8) == 0 {
		t.Fatal("swing-twist")
	}
	w.SetAngularVelocity(2, 12, 0, 0)
	for i := 0; i < 40; i++ {
		w.Step(1.0 / 60)
	}
	ax, _, _, _ := w.GetAngularVelocity(2)
	if ax > 1 {
		t.Fatalf("twist limit should kill spin on the axis, ax=%v", ax)
	}
}

func TestFallbackMaterialGetters(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.AddSphere(1, 0, 2, 0, 0.4, true)
	w.SetRestitution(1, 0.8)
	w.SetFriction(1, 0.2)
	w.SetLinearDamping(1, 1.5)
	w.SetAngularDamping(1, 0.7)
	w.SetGravityScale(1, 0)
	w.SetMass(1, 12)
	if w.SetCCD(1, true) != 1 {
		t.Fatal("ccd")
	}
	if w.GetRestitution(1) != 0.8 || w.GetFriction(1) != 0.2 {
		t.Fatalf("rest/fric %v %v", w.GetRestitution(1), w.GetFriction(1))
	}
	if w.GetLinearDamping(1) != 1.5 || w.GetAngularDamping(1) != 0.7 {
		t.Fatal("damp")
	}
	if w.GetGravityScale(1) != 0 || w.GetMass(1) != 12 || w.GetCCD(1) != 1 {
		t.Fatalf("scale/mass/ccd %v %v %v", w.GetGravityScale(1), w.GetMass(1), w.GetCCD(1))
	}
}

func TestFallbackSensorDoesNotPush(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, 0, 0)
	w.AddSphere(1, 0, 1, 0, 0.5, true)
	w.AddSensorBox(2, 0.2, 1, 0, 0.5, 0.5, 0.5, MotionTypeKinematic)
	x0, _, _, _ := w.GetPosition(1)
	w.Step(1.0 / 60)
	x1, _, _, _ := w.GetPosition(1)
	if x1-x0 > 0.05 || x0-x1 > 0.05 {
		t.Fatalf("sensor should not shove, x0=%v x1=%v", x0, x1)
	}
	if len(w.PollContacts(8)) == 0 {
		t.Fatal("sensor should still report overlap")
	}
}

func TestCapsuleShaftFitsRequestedHeight(t *testing.T) {
	half := capsuleCylinderHalf(1.8, 0.4)
	total := 2 * (half + 0.4)
	if total < 1.75 || total > 1.85 {
		t.Fatalf("capsule total height %v, shaft half %v", total, half)
	}
}

func TestFallbackThinFloorSupportsSphere(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, -10, 0)
	w.AddBoxEx(1, 0, 0, 0, 8, 0.2, 8, MotionTypeStatic)
	w.AddSphere(2, 0, 3, 0, 0.5, true)
	for i := 0; i < 180; i++ {
		w.Step(1.0 / 60)
	}
	_, y, _, _ := w.GetPosition(2)
	if y > 1.2 || y < 0.55 {
		t.Fatalf("sphere should rest on the thin floor, y=%v", y)
	}
}

func TestFallbackCharacterStandsOnPlatform(t *testing.T) {
	w := newFallback()
	defer w.Close()
	w.SetGravity(0, -10, 0)
	w.AddBoxEx(1, 0, 2, 0, 4, 0.2, 4, MotionTypeStatic)
	w.AddCharacterController(2, 0, 6, 0, 1.8, 0.4, 50, 100)
	for i := 0; i < 240; i++ {
		w.Step(1.0 / 60)
	}
	_, y, _, _ := w.GetPosition(2)
	// Platform top is 2.2. Capsule center is half of 1.8 above that.
	if y < 2.9 || y > 3.4 {
		t.Fatalf("character should stand on the platform, y=%v", y)
	}
	if w.CharacterGroundState(2) != 0 {
		t.Fatalf("platform should count as ground, state %d", w.CharacterGroundState(2))
	}
}

func TestCharacterFallsOntoFloor(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, -10, 0)
	w.AddBoxEx(1, 0, 0, 0, 8, 0.25, 8, MotionTypeStatic)
	w.AddCharacterController(2, 0, 4, 0, 1.8, 0.4, 50, 100)
	for i := 0; i < 180; i++ {
		w.Step(1.0 / 60)
	}
	_, y, _, _ := w.GetPosition(2)
	if y > 2.4 || y < 0.9 {
		t.Fatalf("character should stand on the floor, y=%v backend=%s", y, w.Backend())
	}
}

func TestCharacterStandsAndWalks(t *testing.T) {
	w := New()
	defer w.Close()
	w.SetGravity(0, -10, 0)
	w.AddBoxEx(1, 0, 0, 0, 8, 0.25, 8, MotionTypeStatic)
	w.AddCharacterController(2, 0, 4, 0, 1.8, 0.4, 50, 100)
	dt := float32(1.0 / 60)
	for i := 0; i < 180; i++ {
		w.Step(dt)
	}
	_, y, _, _ := w.GetPosition(2)
	// Floor top 0.25, capsule half-height 0.9, center near 1.15.
	if y < 1.0 || y > 1.4 {
		t.Fatalf("capsule center y=%v, want about 1.15 (%s)", y, w.Backend())
	}
	if w.CharacterGroundState(2) != 0 {
		t.Fatalf("standing character ground state %d", w.CharacterGroundState(2))
	}
	x0, _, _, _ := w.GetPosition(2)
	for i := 0; i < 45; i++ {
		w.SetVelocity(2, 3, 0, 0)
		w.Step(dt)
	}
	x1, y1, _, _ := w.GetPosition(2)
	if x1 < x0+0.6 {
		t.Fatalf("walk blocked: x %v -> %v y %v", x0, x1, y1)
	}
	if y1 < 1.0 || y1 > 1.4 {
		t.Fatalf("walk left the floor, y=%v", y1)
	}
}
