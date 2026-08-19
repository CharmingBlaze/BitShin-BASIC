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
