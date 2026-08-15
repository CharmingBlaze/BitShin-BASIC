/*
 * Go-owned JPH::TempAllocatorMalloc. Delete as TempAllocator* so the
 * virtual destructor runs.
 */

#include "wrapper/allocator.h"

#include <Jolt/Jolt.h>
#include <Jolt/Core/TempAllocator.h>

extern "C" JoltTempAllocator JoltCreateTempAllocatorMalloc()
{
	return static_cast<JoltTempAllocator>(new JPH::TempAllocatorMalloc());
}

extern "C" void JoltDestroyTempAllocator(JoltTempAllocator allocator)
{
	delete static_cast<JPH::TempAllocator *>(allocator);
}
