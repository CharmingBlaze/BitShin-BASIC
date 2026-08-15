//go:build windows

package jolt

// #include "wrapper/allocator.h"
import "C"

// TempAllocator is a Jolt temp allocator owned by Go (malloc-backed).
type TempAllocator struct {
	handle C.JoltTempAllocator
}

// NewTempAllocatorMalloc creates a JPH::TempAllocatorMalloc (no fixed cap, no abort).
func NewTempAllocatorMalloc() *TempAllocator {
	return &TempAllocator{handle: C.JoltCreateTempAllocatorMalloc()}
}

// Destroy frees the C++ allocator.
func (a *TempAllocator) Destroy() {
	if a == nil || a.handle == nil {
		return
	}
	C.JoltDestroyTempAllocator(a.handle)
	a.handle = nil
}
