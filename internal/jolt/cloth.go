//go:build windows

package jolt

// #include "wrapper/cloth.h"
import "C"
import "unsafe"

func (bi *BodyInterface) CreateCloth(pos Vec3, width, height float32, nx, ny, pinFlags int, thickness, damping, gravityFactor float32) *BodyID {
	handle := C.JoltCreateCloth(bi.handle,
		C.float(pos.X), C.float(pos.Y), C.float(pos.Z),
		C.float(width), C.float(height),
		C.int(nx), C.int(ny), C.int(pinFlags),
		C.float(thickness), C.float(damping), C.float(gravityFactor))
	if handle == nil {
		return nil
	}
	return &BodyID{handle: handle}
}

func (ps *PhysicsSystem) ClothVertexCount(bodyID *BodyID) int {
	if bodyID == nil {
		return 0
	}
	return int(C.JoltGetClothVertexCount(ps.handle, bodyID.handle))
}

func (ps *PhysicsSystem) ClothVertices(bodyID *BodyID, dst []float32) int {
	if bodyID == nil || len(dst) < 3 {
		return 0
	}
	n := int(C.JoltGetClothVertices(ps.handle, bodyID.handle, (*C.float)(unsafe.Pointer(&dst[0])), C.int(len(dst))))
	return n
}

func (ps *PhysicsSystem) ApplyClothWind(bodyID *BodyID, vx, vy, vz float32, start, step uint32) {
	if bodyID == nil {
		return
	}
	C.JoltApplyClothWind(ps.handle, bodyID.handle, C.float(vx), C.float(vy), C.float(vz), C.uint(start), C.uint(step))
}
