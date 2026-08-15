//go:build !nojolt && ((linux && (amd64 || arm64)) || (darwin && arm64))

package phys3d

// jolt-go C wrapper has no two-body constraints. IDs stay 0 on those platforms.

func (w *joltWorld) CreateHingeJoint(int, int, float32, float32, float32, float32, float32, float32) int {
	return 0
}

func (w *joltWorld) CreatePointJoint(int, int, float32, float32, float32) int { return 0 }

func (w *joltWorld) CreateSliderJoint(int, int, float32, float32, float32, float32, float32, float32) int {
	return 0
}

func (w *joltWorld) CreateSpringJoint(int, int, float32, float32, float32, float32, float32, float32) int {
	return 0
}

func (w *joltWorld) GetRotation(id int) (float32, float32, float32, float32, bool) {
	if _, ok := w.body[id]; !ok {
		return 0, 0, 0, 1, false
	}
	return 0, 0, 0, 1, true
}

func (w *joltWorld) SetRotation(int, float32, float32, float32, float32) {}

func (w *joltWorld) SetCCD(int, bool) int { return 0 }

func (w *joltWorld) RemoveJoint(int) {}

func (w *joltWorld) PollContacts(int) []ContactEvent { return nil }

func (w *joltWorld) EnableContacts() {}

func (w *joltWorld) LookupBody(uint32) int { return 0 }

func (w *joltWorld) ApplyTorque(id int, x, y, z float32) {
	a := w.ang[id]
	w.ang[id] = [3]float32{a[0] + x, a[1] + y, a[2] + z}
}

func (w *joltWorld) ApplyForceAtPosition(id int, fx, fy, fz, _, _, _ float32) {
	w.ApplyForce(id, fx, fy, fz)
}

func (w *joltWorld) ApplyLocalImpulse(id int, lx, ly, lz float32) {
	w.ApplyImpulse(id, lx, ly, lz)
}

func (w *joltWorld) SetGravityScale(int, float32) {}

func (w *joltWorld) SetRestitution(int, float32) {}

func (w *joltWorld) SetLinearDamping(int, float32) {}

func (w *joltWorld) SetAngularDamping(int, float32) {}

func (w *joltWorld) SetFriction(int, float32) {}

func (w *joltWorld) SetHingeLimits(int, float32, float32) {}

func (w *joltWorld) SetHingeFriction(int, float32) {}

func (w *joltWorld) SetHingeMotor(int, float32, float32) {}

func (w *joltWorld) DisableBodyCollision(int, int) {}

func (w *joltWorld) CreateWheeledVehicle(int, float32, float32, float32) int { return 0 }

func (w *joltWorld) CreateMotorcycleVehicle(int, float32, float32, float32) int { return 0 }

func (w *joltWorld) CreateTrackedVehicle(int, float32, float32, float32) int { return 0 }

func (w *joltWorld) SetVehicleInput(int, float32, float32, float32) {}
