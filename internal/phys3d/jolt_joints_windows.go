//go:build !nojolt && windows

package phys3d

import "bitshinbasic/internal/jolt"

func (w *joltWorld) bodyOf(id int) (*jolt.BodyID, bool) {
	if id == 0 {
		return nil, false
	}
	id = w.resolvePhysID(id)
	b := w.body[id]
	if b == nil {
		return nil, false
	}
	return b, true
}

// bodyOrWorld treats 0 as Jolt's world-fixed body (CreateHingeJoint(0, door, …)).
func (w *joltWorld) bodyOrWorld(id int) (*jolt.BodyID, bool) {
	if id == 0 {
		return nil, true
	}
	return w.bodyOf(id)
}

func (w *joltWorld) storeConstraint(a, b int, c *jolt.Constraint) int {
	if c == nil {
		return 0
	}
	if w.constraints == nil {
		w.constraints = map[int]*jolt.Constraint{}
	}
	if w.linked == nil {
		w.linked = map[int]bool{}
	}
	if w.nextConstraintID < 1 {
		w.nextConstraintID = 1
	}
	cID := w.nextConstraintID
	w.nextConstraintID++
	w.constraints[cID] = c
	if a != 0 {
		w.linked[a] = true
	}
	if b != 0 {
		w.linked[b] = true
	}
	return cID
}

func (w *joltWorld) CreateHingeJoint(entA, entB int, pivotX, pivotY, pivotZ, axisX, axisY, axisZ float32) int {
	b1, ok1 := w.bodyOrWorld(entA)
	b2, ok2 := w.bodyOrWorld(entB)
	if !ok1 || !ok2 || (b1 == nil && b2 == nil) {
		return 0
	}
	pivot := jolt.NewVec3(pivotX, pivotY, pivotZ)
	axis := jolt.NewVec3(axisX, axisY, axisZ)
	constraint := w.ps.CreateHingeConstraint(b1, b2, pivot, axis)
	return w.storeConstraint(entA, entB, constraint)
}

func (w *joltWorld) CreatePointJoint(entA, entB int, pivotX, pivotY, pivotZ float32) int {
	b1, ok1 := w.bodyOrWorld(entA)
	b2, ok2 := w.bodyOrWorld(entB)
	if !ok1 || !ok2 || (b1 == nil && b2 == nil) {
		return 0
	}
	pivot := jolt.NewVec3(pivotX, pivotY, pivotZ)
	constraint := w.ps.CreatePointConstraint(b1, b2, pivot)
	return w.storeConstraint(entA, entB, constraint)
}

func (w *joltWorld) CreateSliderJoint(entA, entB int, pivotX, pivotY, pivotZ, axisX, axisY, axisZ float32) int {
	b1, ok1 := w.bodyOrWorld(entA)
	b2, ok2 := w.bodyOrWorld(entB)
	if !ok1 || !ok2 || (b1 == nil && b2 == nil) {
		return 0
	}
	pivot := jolt.NewVec3(pivotX, pivotY, pivotZ)
	axis := jolt.NewVec3(axisX, axisY, axisZ)
	constraint := w.ps.CreateSliderConstraint(b1, b2, pivot, axis)
	return w.storeConstraint(entA, entB, constraint)
}

func (w *joltWorld) CreateSpringJoint(entA, entB int, pivotX, pivotY, pivotZ, rest, stiff, damp float32) int {
	b1, ok1 := w.bodyOrWorld(entA)
	b2, ok2 := w.bodyOrWorld(entB)
	if !ok1 || !ok2 || (b1 == nil && b2 == nil) {
		return 0
	}
	pivot := jolt.NewVec3(pivotX, pivotY, pivotZ)
	minD, maxD := rest, rest
	if rest <= 0 {
		minD, maxD = -1, -1
	}
	constraint := w.ps.CreateDistanceConstraint(b1, b2, pivot, minD, maxD, stiff, damp)
	return w.storeConstraint(entA, entB, constraint)
}

func (w *joltWorld) SetHingeLimits(id int, minDeg, maxDeg float32) {
	c := w.constraints[id]
	if c == nil {
		return
	}
	c.SetHingeLimits(minDeg, maxDeg)
}

func (w *joltWorld) SetHingeFriction(id int, torque float32) {
	c := w.constraints[id]
	if c == nil {
		return
	}
	c.SetHingeFriction(torque)
}

func (w *joltWorld) SetHingeMotor(id int, targetDeg, maxTorque float32) {
	c := w.constraints[id]
	if c == nil {
		return
	}
	c.SetHingeMotor(targetDeg, maxTorque)
}

func (w *joltWorld) DisableBodyCollision(a, b int) {
	ba, okA := w.bodyOf(a)
	bb, okB := w.bodyOf(b)
	if !okA || !okB {
		return
	}
	w.ps.DisableBodyPairCollision(ba, bb)
}

func (w *joltWorld) RemoveJoint(id int) {
	c := w.constraints[id]
	if c == nil {
		return
	}
	w.ps.RemoveConstraint(c)
	delete(w.constraints, id)
}
