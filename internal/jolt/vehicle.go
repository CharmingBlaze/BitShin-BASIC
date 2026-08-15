//go:build windows

package jolt

// #include "wrapper/vehicle.h"
import "C"

const (
	VehicleCar        = int(C.JoltVehicleCar)
	VehicleMotorcycle = int(C.JoltVehicleMotorcycle)
	VehicleTank       = int(C.JoltVehicleTank)
)

// VehicleSettings is the C-boundary 4-wheel / 2-wheel / tracked description.
type VehicleSettings struct {
	HalfWidth   float32
	HalfHeight  float32
	HalfLength  float32
	WheelRadius float32
	WheelWidth  float32
	ObjectLayer int
	Kind        int
}

// VehicleConstraint is a Jolt VehicleConstraint registered as a StepListener.
type VehicleConstraint struct {
	handle C.JoltConstraint
	kind   int
}

// CreateVehicle builds a VehicleConstraint + wheeled/motorcycle/tracked controller.
func (ps *PhysicsSystem) CreateVehicle(bodyID *BodyID, settings VehicleSettings) *VehicleConstraint {
	if bodyID == nil {
		return nil
	}
	kind := settings.Kind
	if kind == 0 {
		kind = VehicleCar
	}
	cs := C.JoltVehicleSettings{
		halfWidth:   C.float(settings.HalfWidth),
		halfHeight:  C.float(settings.HalfHeight),
		halfLength:  C.float(settings.HalfLength),
		wheelRadius: C.float(settings.WheelRadius),
		wheelWidth:  C.float(settings.WheelWidth),
		objectLayer: C.int(settings.ObjectLayer),
		kind:        C.int(kind),
	}
	if settings.ObjectLayer == 0 {
		cs.objectLayer = 1
	}
	h := C.JoltCreateVehicle(ps.handle, bodyID.handle, &cs)
	if h == nil {
		return nil
	}
	return &VehicleConstraint{handle: h, kind: kind}
}

// SetInput applies driver input (steer -1..1, throttle -1..1, brake 0..1).
func (v *VehicleConstraint) SetInput(steer, throttle, brake float32) {
	if v == nil || v.handle == nil {
		return
	}
	C.JoltSetVehicleInput(v.handle, C.int(v.kind), C.float(steer), C.float(throttle), C.float(brake))
}
