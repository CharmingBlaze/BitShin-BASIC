//go:build !nojolt && windows

package phys3d

import "bitshinbasic/internal/jolt"

func (w *joltWorld) ApplyTorque(id int, x, y, z float32) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.bi.AddTorque(b, jolt.NewVec3(x, y, z))
		w.bi.ActivateBody(b)
	}
}

func (w *joltWorld) ApplyForceAtPosition(id int, fx, fy, fz, px, py, pz float32) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.bi.AddForceAtPosition(b, jolt.NewVec3(fx, fy, fz), jolt.NewVec3(px, py, pz))
		w.bi.ActivateBody(b)
	}
}

func (w *joltWorld) ApplyLocalImpulse(id int, lx, ly, lz float32) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.bi.AddLocalImpulse(b, jolt.NewVec3(lx, ly, lz))
		w.bi.ActivateBody(b)
		return
	}
	qx, qy, qz, qw, ok := w.GetRotation(id)
	if !ok {
		w.ApplyImpulse(id, lx, ly, lz)
		return
	}
	world := jolt.Quat{X: qx, Y: qy, Z: qz, W: qw}.Rotate(jolt.NewVec3(lx, ly, lz))
	w.ApplyImpulse(id, world.X, world.Y, world.Z)
}

func (w *joltWorld) SetGravityScale(id int, scale float32) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.ps.SetGravityFactor(b, scale)
	}
}

func (w *joltWorld) SetRestitution(id int, r float32) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.bi.SetRestitution(b, r)
	}
}

func (w *joltWorld) SetLinearDamping(id int, d float32) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.ps.SetLinearDamping(b, d)
	}
}

func (w *joltWorld) SetAngularDamping(id int, d float32) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.ps.SetAngularDamping(b, d)
	}
}

func (w *joltWorld) SetFriction(id int, f float32) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.bi.SetFriction(b, f)
	}
}

func (w *joltWorld) createNativeVehicle(id int, halfW, halfH, halfL float32, kind int) int {
	id = w.resolvePhysID(id)
	b, ok := w.bodyOf(id)
	if !ok || b == nil {
		return 0
	}
	if w.vehicles == nil {
		w.vehicles = map[int]*jolt.VehicleConstraint{}
	}
	if w.linked == nil {
		w.linked = map[int]bool{}
	}
	settings := jolt.VehicleSettings{
		HalfWidth:   halfW,
		HalfHeight:  halfH,
		HalfLength:  halfL,
		WheelRadius: 0.3,
		WheelWidth:  0.12,
		ObjectLayer: 1,
		Kind:        kind,
	}
	if kind == jolt.VehicleMotorcycle {
		settings.WheelRadius = 0.28
		settings.WheelWidth = 0.08
	}
	v := w.ps.CreateVehicle(b, settings)
	if v == nil {
		return 0
	}
	w.vehicles[id] = v
	w.linked[id] = true
	w.bi.ActivateBody(b)
	return id
}

func (w *joltWorld) CreateWheeledVehicle(id int, halfW, halfH, halfL float32) int {
	return w.createNativeVehicle(id, halfW, halfH, halfL, jolt.VehicleCar)
}

func (w *joltWorld) CreateMotorcycleVehicle(id int, halfW, halfH, halfL float32) int {
	return w.createNativeVehicle(id, halfW, halfH, halfL, jolt.VehicleMotorcycle)
}

func (w *joltWorld) CreateTrackedVehicle(id int, halfW, halfH, halfL float32) int {
	return w.createNativeVehicle(id, halfW, halfH, halfL, jolt.VehicleTank)
}

func (w *joltWorld) SetVehicleInput(id int, steer, throttle, brake float32) {
	id = w.resolvePhysID(id)
	if v := w.vehicles[id]; v != nil {
		v.SetInput(steer, throttle, brake)
		if b, ok := w.body[id]; ok {
			w.bi.ActivateBody(b)
		}
	}
}
