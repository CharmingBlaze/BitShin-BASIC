//go:build windows

package jolt

// #include "wrapper/physics.h"
// #include "wrapper/allocator.h"
import "C"

// PhysicsSystem represents a physics simulation world
type PhysicsSystem struct {
	handle C.JoltPhysicsSystem
}

// NewPhysicsSystem creates a new physics world
func NewPhysicsSystem() *PhysicsSystem {
	handle := C.JoltCreatePhysicsSystem()
	return &PhysicsSystem{handle: handle}
}

// Destroy frees the physics system
func (ps *PhysicsSystem) Destroy() {
	C.JoltDestroyPhysicsSystem(ps.handle)
}

// Update advances the simulation by deltaTime seconds using a Go-owned allocator.
func (ps *PhysicsSystem) Update(deltaTime float32, allocator *TempAllocator) {
	if allocator == nil || allocator.handle == nil {
		return
	}
	C.JoltPhysicsSystemUpdateWithAllocator(ps.handle, C.float(deltaTime), allocator.handle)
}

// SetGravity sets Jolt world gravity.
func (ps *PhysicsSystem) SetGravity(g Vec3) {
	C.JoltSetGravity(ps.handle, C.float(g.X), C.float(g.Y), C.float(g.Z))
}
