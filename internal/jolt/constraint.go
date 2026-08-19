//go:build windows

package jolt

// #include "wrapper/constraint.h"
import "C"

// Constraint is an opaque Jolt two-body constraint.
type Constraint struct {
	handle C.JoltConstraint
}

func cBody(b *BodyID) C.JoltBodyID {
	if b == nil {
		return nil
	}
	return b.handle
}

// CreateHingeConstraint links two bodies with a hinge at a world pivot/axis.
// A nil body is Jolt Body::sFixedToWorld (CreateHingeJoint(0, door, …)).
func (ps *PhysicsSystem) CreateHingeConstraint(a, b *BodyID, pivot, axis Vec3) *Constraint {
	if a == nil && b == nil {
		return nil
	}
	h := C.JoltCreateHingeConstraint(ps.handle, cBody(a), cBody(b),
		C.float(pivot.X), C.float(pivot.Y), C.float(pivot.Z),
		C.float(axis.X), C.float(axis.Y), C.float(axis.Z))
	if h == nil {
		return nil
	}
	return &Constraint{handle: h}
}

// CreatePointConstraint links two bodies at a world point (ball-socket).
func (ps *PhysicsSystem) CreatePointConstraint(a, b *BodyID, pivot Vec3) *Constraint {
	if a == nil && b == nil {
		return nil
	}
	h := C.JoltCreatePointConstraint(ps.handle, cBody(a), cBody(b),
		C.float(pivot.X), C.float(pivot.Y), C.float(pivot.Z))
	if h == nil {
		return nil
	}
	return &Constraint{handle: h}
}

// CreateSliderConstraint links two bodies along a world axis.
func (ps *PhysicsSystem) CreateSliderConstraint(a, b *BodyID, pivot, axis Vec3) *Constraint {
	if a == nil && b == nil {
		return nil
	}
	h := C.JoltCreateSliderConstraint(ps.handle, cBody(a), cBody(b),
		C.float(pivot.X), C.float(pivot.Y), C.float(pivot.Z),
		C.float(axis.X), C.float(axis.Y), C.float(axis.Z))
	if h == nil {
		return nil
	}
	return &Constraint{handle: h}
}

// CreateDistanceConstraint is a springy distance joint (suspension / platforms).
func (ps *PhysicsSystem) CreateDistanceConstraint(a, b *BodyID, pivot Vec3, minDist, maxDist, frequency, damping float32) *Constraint {
	if a == nil && b == nil {
		return nil
	}
	h := C.JoltCreateDistanceConstraint(ps.handle, cBody(a), cBody(b),
		C.float(pivot.X), C.float(pivot.Y), C.float(pivot.Z),
		C.float(minDist), C.float(maxDist), C.float(frequency), C.float(damping))
	if h == nil {
		return nil
	}
	return &Constraint{handle: h}
}

// RemoveConstraint removes a constraint from the physics system.
func (ps *PhysicsSystem) RemoveConstraint(c *Constraint) {
	if c == nil || c.handle == nil {
		return
	}
	// Jolt owns the constraint while it is registered. RemoveConstraint drops
	// that owning reference and may destroy the object immediately; deleting
	// the same pointer again corrupts the native heap.
	C.JoltRemoveConstraint(ps.handle, c.handle)
	c.handle = nil
}

// SetHingeLimits sets rotation limits in degrees (Jolt clamps min≤0≤max).
func (c *Constraint) SetHingeLimits(minDeg, maxDeg float32) {
	if c == nil || c.handle == nil {
		return
	}
	C.JoltSetHingeLimits(c.handle, C.float(minDeg), C.float(maxDeg))
}

// SetHingeFriction sets max friction torque (N·m) when the motor is off.
func (c *Constraint) SetHingeFriction(torque float32) {
	if c == nil || c.handle == nil {
		return
	}
	C.JoltSetHingeFriction(c.handle, C.float(torque))
}

// SetHingeMotor drives the hinge to targetDeg (0 = rest). maxTorque<=0 turns the motor off.
func (c *Constraint) SetHingeMotor(targetDeg, maxTorque float32) {
	if c == nil || c.handle == nil {
		return
	}
	C.JoltSetHingeMotor(c.handle, C.float(targetDeg), C.float(maxTorque))
}

// CreateGrabConstraint is a 6DOF spring grab (soft translation, free rotation).
func (ps *PhysicsSystem) CreateGrabConstraint(a, b *BodyID, pivot Vec3, frequency, damping float32) *Constraint {
	if a == nil && b == nil {
		return nil
	}
	h := C.JoltCreateGrabConstraint(ps.handle, cBody(a), cBody(b),
		C.float(pivot.X), C.float(pivot.Y), C.float(pivot.Z),
		C.float(frequency), C.float(damping))
	if h == nil {
		return nil
	}
	return &Constraint{handle: h}
}

func (ps *PhysicsSystem) CreateFixedConstraint(a, b *BodyID, pivot Vec3) *Constraint {
	if a == nil && b == nil {
		return nil
	}
	h := C.JoltCreateFixedConstraint(ps.handle, cBody(a), cBody(b),
		C.float(pivot.X), C.float(pivot.Y), C.float(pivot.Z))
	if h == nil {
		return nil
	}
	return &Constraint{handle: h}
}

func (ps *PhysicsSystem) CreateConeConstraint(a, b *BodyID, pivot, axis Vec3, halfConeDeg float32) *Constraint {
	if a == nil && b == nil {
		return nil
	}
	h := C.JoltCreateConeConstraint(ps.handle, cBody(a), cBody(b),
		C.float(pivot.X), C.float(pivot.Y), C.float(pivot.Z),
		C.float(axis.X), C.float(axis.Y), C.float(axis.Z),
		C.float(halfConeDeg))
	if h == nil {
		return nil
	}
	return &Constraint{handle: h}
}

func (ps *PhysicsSystem) CreateSwingTwistConstraint(a, b *BodyID, pivot, axis Vec3, swingDeg, twistDeg float32) *Constraint {
	if a == nil && b == nil {
		return nil
	}
	h := C.JoltCreateSwingTwistConstraint(ps.handle, cBody(a), cBody(b),
		C.float(pivot.X), C.float(pivot.Y), C.float(pivot.Z),
		C.float(axis.X), C.float(axis.Y), C.float(axis.Z),
		C.float(swingDeg), C.float(twistDeg))
	if h == nil {
		return nil
	}
	return &Constraint{handle: h}
}

// DisableBodyPairCollision stops contacts between two bodies (GroupFilterTable).
func (ps *PhysicsSystem) DisableBodyPairCollision(a, b *BodyID) {
	if ps == nil || (a == nil && b == nil) {
		return
	}
	C.JoltDisableBodyPairCollision(ps.handle, cBody(a), cBody(b))
}
