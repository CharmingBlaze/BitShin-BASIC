/*
 * Heap-backed Jolt temp allocator (TempAllocatorMalloc).
 * Owned by Go; passed into PhysicsSystem::Update so constraint buffers
 * never abort() against a fixed TempAllocatorImpl.
 */

#ifndef JOLT_WRAPPER_ALLOCATOR_H
#define JOLT_WRAPPER_ALLOCATOR_H

#ifdef __cplusplus
extern "C" {
#endif

typedef void *JoltTempAllocator;

JoltTempAllocator JoltCreateTempAllocatorMalloc();
void JoltDestroyTempAllocator(JoltTempAllocator allocator);

#ifdef __cplusplus
}
#endif

#endif
