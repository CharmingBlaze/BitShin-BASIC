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

	JointHinge  = 1
	JointPoint  = 2
	JointSlider = 3
	JointSpring = 4

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
	SetRestitution(id int, r float32)
	SetLinearDamping(id int, d float32)
	SetAngularDamping(id int, d float32)
	SetFriction(id int, f float32)
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
}

// planeAero is lift/drag state for CreatePlaneController / UpdatePlane.
type planeAero struct {
	thrust, cl, cd, stall, rho, area, mass float32
}
