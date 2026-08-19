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

func (w *joltWorld) ApplyBuoyancyImpulse(id int, sx, sy, sz, nx, ny, nz, buoyancy, linDrag, angDrag, fvx, fvy, fvz, dt float32) bool {
	return false
}

func (w *joltWorld) OffsetCenterOfMass(int, float32, float32, float32) {}

func (w *joltWorld) AddMesh(int, [][3]float32, []int32, int) {}

func (w *joltWorld) AddHeightField(int, []float32, int, float32, float32, float32, float32, float32, float32) {
}

func (w *joltWorld) AddSensorBox(id int, x, y, z, hx, hy, hz float32, motion int) {
	w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
}

func (w *joltWorld) SetSensor(int, bool) {}

func (w *joltWorld) ShapeCast(hx, hy, hz, x, y, z, dx, dy, dz float32) (int, float32, float32, float32, bool) {
	return w.Raycast(x, y, z, dx, dy, dz)
}

func (w *joltWorld) OverlapSphere(x, y, z, r float32) (int, bool) {
	id, _, _, _, ok := w.Raycast(x, y, z, 0, -r, 0)
	return id, ok
}

func (w *joltWorld) OverlapPoint(x, y, z float32) (int, bool) {
	id, _, _, _, ok := w.Raycast(x, y+0.05, z, 0, -0.1, 0)
	return id, ok
}

func (w *joltWorld) OptimizeBroadPhase() {}

func (w *joltWorld) AddCloth(int, float32, float32, float32, float32, float32, int, int, int, float32, float32, float32) {
}

func (w *joltWorld) ClothVertexCount(int) int { return 0 }

func (w *joltWorld) ClothVertices(int, []float32) int { return 0 }

func (w *joltWorld) ApplyClothWind(int, float32, float32, float32, uint32, uint32) {}

func (w *joltWorld) CreateGrabJoint(int, int, float32, float32, float32, float32, float32) int {
	return 0
}

func (w *joltWorld) CreateFixedJoint(int, int, float32, float32, float32) int { return 0 }

func (w *joltWorld) CreateConeJoint(int, int, float32, float32, float32, float32, float32, float32, float32) int {
	return 0
}

func (w *joltWorld) CreateSwingTwistJoint(int, int, float32, float32, float32, float32, float32, float32, float32, float32) int {
	return 0
}

func (w *joltWorld) AddCompound(id int, parts []CompoundPart, x, y, z float32, motion int) {
	hx, hy, hz := CompoundAABB(parts)
	w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
}

func (w *joltWorld) OverlapSphereAll(x, y, z, r float32, max int) []int {
	id, ok := w.OverlapSphere(x, y, z, r)
	if !ok || id == 0 {
		return nil
	}
	return []int{id}
}

func (w *joltWorld) SetCollisionLayer(int, int) {}

func (w *joltWorld) SetLayerCollides(int, int, bool) {}
