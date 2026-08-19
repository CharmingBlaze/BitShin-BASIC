/*
 * Jolt Physics C Wrapper - Physics System
 *
 * Handles physics world creation, management, and collision layers.
 */

#ifndef JOLT_WRAPPER_PHYSICS_H
#define JOLT_WRAPPER_PHYSICS_H

#include "allocator.h"

#ifdef __cplusplus
extern "C" {
#endif

// Opaque pointer types
typedef void* JoltPhysicsSystem;

// Create a new physics world
JoltPhysicsSystem JoltCreatePhysicsSystem();

// Destroy a physics world
void JoltDestroyPhysicsSystem(JoltPhysicsSystem system);

// Prebuilt 2-arg Update uses TempAllocatorImpl and abort()s on 17MB spikes.
void JoltPhysicsSystemUpdate(JoltPhysicsSystem system, float deltaTime);

// Go-owned malloc allocator. PhysicsSystem.Update(dt, alloc) calls this.
void JoltPhysicsSystemUpdateWithAllocator(JoltPhysicsSystem system, float deltaTime, JoltTempAllocator allocator);

// World gravity (Jolt default is (0, -9.81, 0))
void JoltSetGravity(JoltPhysicsSystem system, float x, float y, float z);

void JoltOptimizeBroadPhase(JoltPhysicsSystem system);

#ifdef __cplusplus
}

// C++ only: Accessor functions for wrapper internals (used by character.cpp)
namespace JPH {
    class PhysicsSystem;
    class ObjectVsBroadPhaseLayerFilter;
    class ObjectLayerPairFilter;
}

struct PhysicsSystemWrapper;  // Opaque forward declaration

// Accessor functions
JPH::PhysicsSystem* GetPhysicsSystem(PhysicsSystemWrapper* wrapper);
const JPH::ObjectVsBroadPhaseLayerFilter* GetObjectVsBroadPhaseLayerFilter(PhysicsSystemWrapper* wrapper);
const JPH::ObjectLayerPairFilter* GetObjectLayerPairFilter(PhysicsSystemWrapper* wrapper);

#endif

#endif // JOLT_WRAPPER_PHYSICS_H
