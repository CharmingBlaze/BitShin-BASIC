// Package phys3d is 3D rigid bodies. Default is Jolt (jolt-go / Windows static libs).
// Build with -tags nojolt for the software fallback.
package phys3d

// Backend names the active 3D physics engine.
const (
	BackendJolt        = "jolt"
	BackendFallback    = "fallback"
	CharacterJolt      = "jolt"
	CharacterKinematic = "kinematic"

	ContactAdded     = 1
	ContactPersisted = 2
	ContactRemoved   = 3

	JointHinge      = 1
	JointPoint      = 2
	JointSlider     = 3
	JointSpring     = 4
	JointFixed      = 5
	JointCone       = 6
	JointSwingTwist = 7

	CompoundBox      = 0
	CompoundSphere   = 1
	CompoundCapsule  = 2
	CompoundCylinder = 3

	// Motion types match Jolt EMotionType / CreateBodyBox 4th arg.
	MotionTypeStatic    = 0
	MotionTypeKinematic = 1
	MotionTypeDynamic   = 2
)

func motionFromDynamic(dynamic bool) int {
	if dynamic {
		return MotionTypeDynamic
	}
	return MotionTypeStatic
}

// ContactEvent is a drained Jolt (or fallback) collision pair.
type ContactEvent struct {
	Kind       int
	A, B       int
	X, Y, Z    float32
	NX, NY, NZ float32
}

// World is a 3D rigid-body simulation (Jolt where prebuilts exist).
type World interface {
	Backend() string
	Step(dt float32)
	AddBox(id int, x, y, z, hx, hy, hz float32, dynamic bool)
	AddBoxEx(id int, x, y, z, hx, hy, hz float32, motion int)
	AddSphere(id int, x, y, z, r float32, dynamic bool)
	AddSphereEx(id int, x, y, z, r float32, motion int)
	SetPosition(id int, x, y, z float32)
	GetPosition(id int) (x, y, z float32, ok bool)
	// MoveKinematic sweeps a kinematic body to a pose this step so it can push dynamics.
	MoveKinematic(id int, x, y, z, qx, qy, qz, qw, dt float32)
	SetVelocity(id int, x, y, z float32)
	GetVelocity(id int) (x, y, z float32, ok bool)
	ApplyImpulse(id int, x, y, z float32)
	ApplyForce(id int, x, y, z float32)
	SetAngularVelocity(id int, x, y, z float32)
	GetAngularVelocity(id int) (x, y, z float32, ok bool)
	SetMass(id int, mass float32)
	AddCapsule(id int, x, y, z, halfH, r float32, dynamic bool)
	AddGround(id int, y float32)
	Raycast(ox, oy, oz, dx, dy, dz float32) (id int, x, y, z float32, ok bool)
	SetGravity(x, y, z float32)
	GetGravity() (x, y, z float32)
	Remove(id int)
	Close()
	Sleep(id int)
	Wake(id int)
	AddCharacter(id int, x, y, z, halfH, r float32)
	AddCharacterController(id int, x, y, z, height, radius, maxSlopeDeg, maxStrength float32)
	SetCharacterShape(id int, shapeType string, height, radius float32)
	CharacterGround(id int) int
	CharacterGroundState(id int) int
	CharacterGroundNormal(id int) (x, y, z float32)
	CharacterContact(id int) int
	CharacterBackend() string
	GetRotation(id int) (x, y, z, w float32, ok bool)
	SetRotation(id int, x, y, z, w float32)
	SetCCD(id int, on bool) int
	CreateHingeJoint(a, b int, px, py, pz, ax, ay, az float32) int
	CreatePointJoint(a, b int, px, py, pz float32) int
	CreateSliderJoint(a, b int, px, py, pz, ax, ay, az float32) int
	CreateSpringJoint(a, b int, px, py, pz, rest, freq, damp float32) int
	RemoveJoint(id int)
	PollContacts(max int) []ContactEvent
	EnableContacts()
	LookupBody(bodyValue uint32) int
	ApplyTorque(id int, x, y, z float32)
	ApplyForceAtPosition(id int, fx, fy, fz, px, py, pz float32)
	ApplyLocalImpulse(id int, lx, ly, lz float32)
	SetGravityScale(id int, scale float32)
	GetGravityScale(id int) float32
	SetRestitution(id int, r float32)
	GetRestitution(id int) float32
	SetLinearDamping(id int, d float32)
	GetLinearDamping(id int) float32
	SetAngularDamping(id int, d float32)
	GetAngularDamping(id int) float32
	SetFriction(id int, f float32)
	GetFriction(id int) float32
	GetCCD(id int) int
	GetMass(id int) float32
	SetHingeLimits(id int, minDeg, maxDeg float32)
	SetHingeFriction(id int, torque float32)
	SetHingeMotor(id int, targetDeg, maxTorque float32)
	DisableBodyCollision(a, b int)
	CreateWheeledVehicle(id int, halfW, halfH, halfL float32) int
	CreateMotorcycleVehicle(id int, halfW, halfH, halfL float32) int
	CreateTrackedVehicle(id int, halfW, halfH, halfL float32) int
	SetVehicleInput(id int, steer, throttle, brake float32)
	CreatePlaneController(id int) int
	UpdatePlane(id int, throttle, pitch, roll, yaw float32)
	ApplyBuoyancyImpulse(id int, sx, sy, sz, nx, ny, nz, buoyancy, linDrag, angDrag, fvx, fvy, fvz, dt float32) bool
	OffsetCenterOfMass(id int, ox, oy, oz float32)
	AddMesh(id int, verts [][3]float32, indices []int32, motion int)
	AddHeightField(id int, samples []float32, n int, ox, oy, oz, sx, sy, sz float32)
	AddSensorBox(id int, x, y, z, hx, hy, hz float32, motion int)
	SetSensor(id int, on bool)
	ShapeCast(hx, hy, hz, x, y, z, dx, dy, dz float32) (id int, hitX, hitY, hitZ float32, ok bool)
	OverlapSphere(x, y, z, r float32) (id int, ok bool)
	OverlapSphereAll(x, y, z, r float32, max int) []int
	OverlapPoint(x, y, z float32) (id int, ok bool)
	OptimizeBroadPhase()
	AddCloth(id int, x, y, z, width, height float32, nx, ny, pinFlags int, thickness, damping, gravityFactor float32)
	ClothVertexCount(id int) int
	ClothVertices(id int, dst []float32) int
	ApplyClothWind(id int, vx, vy, vz float32, start, step uint32)
	AddCylinder(id int, x, y, z, halfH, r float32, motion int)
	AddConvexHull(id int, points [][3]float32, x, y, z float32, motion int)
	AddCompound(id int, parts []CompoundPart, x, y, z float32, motion int)
	CreateGrabJoint(a, b int, px, py, pz, freq, damp float32) int
	CreateFixedJoint(a, b int, px, py, pz float32) int
	CreateConeJoint(a, b int, px, py, pz, ax, ay, az, halfConeDeg float32) int
	CreateSwingTwistJoint(a, b int, px, py, pz, ax, ay, az, swingDeg, twistDeg float32) int
	SetCollisionLayer(id, layer int)
	SetLayerCollides(a, b int, on bool)
	SetJobThreads(n int)
}

// CompoundPart is one child shape in AddCompound (local offset from the actor).
type CompoundPart struct {
	Kind                int
	Ox, Oy, Oz, A, B, C float32
}

func CompoundAABB(parts []CompoundPart) (hx, hy, hz float32) {
	for _, p := range parts {
		var a, b, c float32
		switch p.Kind {
		case CompoundSphere:
			a, b, c = p.A, p.A, p.A
		case CompoundCapsule, CompoundCylinder:
			a, b, c = p.B, p.A+p.B, p.B
		default:
			a, b, c = p.A, p.B, p.C
		}
		if mx := abs32(p.Ox) + a; mx > hx {
			hx = mx
		}
		if my := abs32(p.Oy) + b; my > hy {
			hy = my
		}
		if mz := abs32(p.Oz) + c; mz > hz {
			hz = mz
		}
	}
	if hx < 0.1 {
		hx = 0.1
	}
	if hy < 0.1 {
		hy = 0.1
	}
	if hz < 0.1 {
		hz = 0.1
	}
	return hx, hy, hz
}

// planeAero is lift/drag state for CreatePlaneController / UpdatePlane.
type planeAero struct {
	thrust, cl, cd, stall, rho, area, mass float32
}
