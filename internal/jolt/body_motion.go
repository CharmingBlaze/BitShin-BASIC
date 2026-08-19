//go:build windows

package jolt

// #include "wrapper/body.h"
import "C"

// Value is Jolt's packed BodyID (index + sequence), used to map contact events.
func (b *BodyID) Value() uint32 {
	if b == nil {
		return 0
	}
	return uint32(C.JoltGetBodyIDValue(b.handle))
}

// GetRotation returns the current rotation of a body as a quaternion.
func (bi *BodyInterface) GetRotation(bodyID *BodyID) Quat {
	x, y, z, w := C.float(0), C.float(0), C.float(0), C.float(0)
	C.JoltGetBodyRotation(bi.handle, bodyID.handle, &x, &y, &z, &w)
	return Quat{X: float32(x), Y: float32(y), Z: float32(z), W: float32(w)}
}

// SetRotation sets the rotation of a body.
func (bi *BodyInterface) SetRotation(bodyID *BodyID, q Quat) {
	if bodyID == nil {
		return
	}
	C.JoltSetBodyRotation(bi.handle, bodyID.handle, C.float(q.X), C.float(q.Y), C.float(q.Z), C.float(q.W))
}

// GetLinearVelocity returns the center-of-mass linear velocity.
func (bi *BodyInterface) GetLinearVelocity(bodyID *BodyID) Vec3 {
	x, y, z := C.float(0), C.float(0), C.float(0)
	C.JoltGetBodyLinearVelocity(bi.handle, bodyID.handle, &x, &y, &z)
	return Vec3{X: float32(x), Y: float32(y), Z: float32(z)}
}

// SetLinearVelocity sets the center-of-mass linear velocity.
func (bi *BodyInterface) SetLinearVelocity(bodyID *BodyID, v Vec3) {
	if bodyID == nil {
		return
	}
	C.JoltSetBodyLinearVelocity(bi.handle, bodyID.handle, C.float(v.X), C.float(v.Y), C.float(v.Z))
}

// GetAngularVelocity returns the angular velocity in rad/s.
func (bi *BodyInterface) GetAngularVelocity(bodyID *BodyID) Vec3 {
	x, y, z := C.float(0), C.float(0), C.float(0)
	C.JoltGetBodyAngularVelocity(bi.handle, bodyID.handle, &x, &y, &z)
	return Vec3{X: float32(x), Y: float32(y), Z: float32(z)}
}

// SetAngularVelocity sets the angular velocity in rad/s.
func (bi *BodyInterface) SetAngularVelocity(bodyID *BodyID, v Vec3) {
	if bodyID == nil {
		return
	}
	C.JoltSetBodyAngularVelocity(bi.handle, bodyID.handle, C.float(v.X), C.float(v.Y), C.float(v.Z))
}

// AddForce applies a world-space force at the center of mass (this step).
func (bi *BodyInterface) AddForce(bodyID *BodyID, force Vec3) {
	if bodyID == nil {
		return
	}
	C.JoltAddForce(bi.handle, bodyID.handle, C.float(force.X), C.float(force.Y), C.float(force.Z))
}

// AddTorque applies a world-space torque.
func (bi *BodyInterface) AddTorque(bodyID *BodyID, torque Vec3) {
	if bodyID == nil {
		return
	}
	C.JoltAddTorque(bi.handle, bodyID.handle, C.float(torque.X), C.float(torque.Y), C.float(torque.Z))
}

// AddAngularImpulse applies an angular impulse.
func (bi *BodyInterface) AddAngularImpulse(bodyID *BodyID, impulse Vec3) {
	if bodyID == nil {
		return
	}
	C.JoltAddAngularImpulse(bi.handle, bodyID.handle, C.float(impulse.X), C.float(impulse.Y), C.float(impulse.Z))
}

// SetMotionQualityLinearCast enables Jolt CCD (MotionQuality::LinearCast).
func (bi *BodyInterface) SetMotionQualityLinearCast(bodyID *BodyID, on bool) {
	if bodyID == nil {
		return
	}
	flag := C.int(0)
	if on {
		flag = 1
	}
	C.JoltSetBodyMotionQuality(bi.handle, bodyID.handle, flag)
}

// SetMass sets inverse mass on a dynamic body.
func (ps *PhysicsSystem) SetMass(bodyID *BodyID, mass float32) {
	if bodyID == nil {
		return
	}
	C.JoltSetBodyMass(ps.handle, bodyID.handle, C.float(mass))
}

// AddForceAtPosition applies a world-space force at a world-space point.
func (bi *BodyInterface) AddForceAtPosition(bodyID *BodyID, force, position Vec3) {
	if bodyID == nil {
		return
	}
	C.JoltAddForceAtPosition(bi.handle, bodyID.handle,
		C.float(force.X), C.float(force.Y), C.float(force.Z),
		C.float(position.X), C.float(position.Y), C.float(position.Z))
}

// AddLocalImpulse rotates a local-space impulse into world space and applies it.
func (bi *BodyInterface) AddLocalImpulse(bodyID *BodyID, local Vec3) {
	if bodyID == nil {
		return
	}
	C.JoltAddLocalImpulse(bi.handle, bodyID.handle, C.float(local.X), C.float(local.Y), C.float(local.Z))
}

// SetGravityFactor multiplies world gravity for this body (0 = weightless).
func (ps *PhysicsSystem) SetGravityFactor(bodyID *BodyID, scale float32) {
	if bodyID == nil {
		return
	}
	C.JoltSetGravityFactor(ps.handle, bodyID.handle, C.float(scale))
}

// SetRestitution sets bounciness [0,1].
func (bi *BodyInterface) SetRestitution(bodyID *BodyID, restitution float32) {
	if bodyID == nil {
		return
	}
	C.JoltSetRestitution(bi.handle, bodyID.handle, C.float(restitution))
}

// SetLinearDamping sets linear drag on a dynamic body.
func (ps *PhysicsSystem) SetLinearDamping(bodyID *BodyID, damping float32) {
	if bodyID == nil {
		return
	}
	C.JoltSetLinearDamping(ps.handle, bodyID.handle, C.float(damping))
}

// SetAngularDamping sets angular drag on a dynamic body.
func (ps *PhysicsSystem) SetAngularDamping(bodyID *BodyID, damping float32) {
	if bodyID == nil {
		return
	}
	C.JoltSetAngularDamping(ps.handle, bodyID.handle, C.float(damping))
}

// SetFriction sets Coulomb friction.
func (bi *BodyInterface) SetFriction(bodyID *BodyID, friction float32) {
	if bodyID == nil {
		return
	}
	C.JoltSetFriction(bi.handle, bodyID.handle, C.float(friction))
}

// MoveKinematic drives a kinematic body to a pose this step so hinges follow it.
func (bi *BodyInterface) MoveKinematic(bodyID *BodyID, pos Vec3, rot Quat, dt float32) {
	if bodyID == nil {
		return
	}
	C.JoltMoveKinematic(bi.handle, bodyID.handle,
		C.float(pos.X), C.float(pos.Y), C.float(pos.Z),
		C.float(rot.X), C.float(rot.Y), C.float(rot.Z), C.float(rot.W),
		C.float(dt))
}

func (bi *BodyInterface) ApplyBuoyancyImpulse(bodyID *BodyID, surface, normal Vec3, buoyancy, linearDrag, angularDrag float32, fluidVel, gravity Vec3, dt float32) bool {
	if bodyID == nil {
		return false
	}
	ok := C.JoltApplyBuoyancyImpulse(bi.handle, bodyID.handle,
		C.float(surface.X), C.float(surface.Y), C.float(surface.Z),
		C.float(normal.X), C.float(normal.Y), C.float(normal.Z),
		C.float(buoyancy), C.float(linearDrag), C.float(angularDrag),
		C.float(fluidVel.X), C.float(fluidVel.Y), C.float(fluidVel.Z),
		C.float(gravity.X), C.float(gravity.Y), C.float(gravity.Z),
		C.float(dt))
	return ok != 0
}

func (ps *PhysicsSystem) SetBodySensor(bodyID *BodyID, on bool) {
	if bodyID == nil {
		return
	}
	flag := C.int(0)
	if on {
		flag = 1
	}
	C.JoltSetBodySensor(ps.handle, bodyID.handle, flag)
}

func (ps *PhysicsSystem) GetBodyShape(bodyID *BodyID) *Shape {
	if bodyID == nil {
		return nil
	}
	handle := C.JoltGetBodyShape(ps.handle, bodyID.handle)
	if handle == nil {
		return nil
	}
	return &Shape{handle: handle}
}

func (bi *BodyInterface) CreateBodyEx(shape *Shape, position Vec3, motionType MotionType, isSensor, enhancedEdges bool) *BodyID {
	sensor, edges := C.int(0), C.int(0)
	if isSensor {
		sensor = 1
	}
	if enhancedEdges {
		edges = 1
	}
	handle := C.JoltCreateBodyEx(bi.handle, shape.handle,
		C.float(position.X), C.float(position.Y), C.float(position.Z),
		C.JoltMotionType(motionType), sensor, edges)
	if handle == nil {
		return nil
	}
	return &BodyID{handle: handle}
}
