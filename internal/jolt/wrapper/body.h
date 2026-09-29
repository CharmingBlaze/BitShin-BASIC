/*
 * Jolt Physics C Wrapper - Body Operations
 *
 * Handles rigid body creation and manipulation.
 */

#ifndef JOLT_WRAPPER_BODY_H
#define JOLT_WRAPPER_BODY_H

#include "physics.h"

#ifdef __cplusplus
extern "C" {
#endif

// Opaque pointer types
typedef void* JoltBodyInterface;
typedef void* JoltBodyID;
typedef void* JoltShape;

// Motion type enum (matches Jolt's EMotionType)
typedef enum {
    JoltMotionTypeStatic = 0,    // Immovable, zero velocity
    JoltMotionTypeKinematic = 1, // Movable by user, zero velocity response to forces
    JoltMotionTypeDynamic = 2    // Affected by forces
} JoltMotionType;

// Get the body interface for creating/manipulating bodies
JoltBodyInterface JoltPhysicsSystemGetBodyInterface(JoltPhysicsSystem system);

// Get the position of a body
void JoltGetBodyPosition(const JoltBodyInterface bodyInterface,
                        const JoltBodyID bodyID,
                        float* x, float* y, float* z);

// Set the position of a body
void JoltSetBodyPosition(JoltBodyInterface bodyInterface,
                        JoltBodyID bodyID,
                        float x, float y, float z);

// Create a body with specific motion type and sensor flag
JoltBodyID JoltCreateBody(JoltBodyInterface bodyInterface,
                          JoltShape shape,
                          float x, float y, float z,
                          JoltMotionType motionType,
                          int isSensor);

// Activate a body (makes it participate in simulation)
void JoltActivateBody(JoltBodyInterface bodyInterface, JoltBodyID bodyID);

// Deactivate a body (removes from active simulation)
void JoltDeactivateBody(JoltBodyInterface bodyInterface, JoltBodyID bodyID);

// Set the shape of a body
void JoltSetBodyShape(JoltBodyInterface bodyInterface,
                     JoltBodyID bodyID,
                     JoltShape shape,
                     int updateMassProperties);

// Destroy a body ID wrapper only (does not remove the body from the world).
void JoltDestroyBodyID(JoltBodyID bodyID);

// Remove the body from the world, destroy it, and free the ID wrapper.
void JoltRemoveAndDestroyBody(JoltBodyInterface bodyInterface, JoltBodyID bodyID);

// Packed BodyID index+sequence (stable across heap copies of the same ID)
unsigned int JoltGetBodyIDValue(JoltBodyID bodyID);

// Rotation (Jolt Quat, xyzw)
void JoltGetBodyRotation(const JoltBodyInterface bodyInterface,
                        const JoltBodyID bodyID,
                        float* x, float* y, float* z, float* w);
void JoltSetBodyRotation(JoltBodyInterface bodyInterface,
                        JoltBodyID bodyID,
                        float x, float y, float z, float w);

// Linear / angular velocity
void JoltGetBodyLinearVelocity(const JoltBodyInterface bodyInterface,
                              const JoltBodyID bodyID,
                              float* x, float* y, float* z);
void JoltSetBodyLinearVelocity(JoltBodyInterface bodyInterface,
                              JoltBodyID bodyID,
                              float x, float y, float z);
void JoltGetBodyAngularVelocity(const JoltBodyInterface bodyInterface,
                               const JoltBodyID bodyID,
                               float* x, float* y, float* z);
void JoltSetBodyAngularVelocity(JoltBodyInterface bodyInterface,
                               JoltBodyID bodyID,
                               float x, float y, float z);

// Forces and impulses at center of mass
void JoltAddForce(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
                 float x, float y, float z);
void JoltAddImpulse(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
                   float x, float y, float z);
void JoltAddTorque(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
                  float x, float y, float z);
void JoltAddAngularImpulse(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
                          float x, float y, float z);

// MotionQuality::LinearCast when enabled (CCD)
void JoltSetBodyMotionQuality(JoltBodyInterface bodyInterface, JoltBodyID bodyID, int linearCast);

// Inverse mass (dynamic bodies only)
void JoltSetBodyMass(JoltPhysicsSystem system, JoltBodyID bodyID, float mass);
float JoltGetInverseMass(JoltPhysicsSystem system, JoltBodyID bodyID);

void JoltAddForceAtPosition(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
                           float fx, float fy, float fz,
                           float px, float py, float pz);
void JoltAddLocalImpulse(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
                        float lx, float ly, float lz);
void JoltSetGravityFactor(JoltPhysicsSystem system, JoltBodyID bodyID, float scale);
void JoltSetRestitution(JoltBodyInterface bodyInterface, JoltBodyID bodyID, float restitution);
void JoltSetLinearDamping(JoltPhysicsSystem system, JoltBodyID bodyID, float damping);
void JoltSetAngularDamping(JoltPhysicsSystem system, JoltBodyID bodyID, float damping);
void JoltSetFriction(JoltBodyInterface bodyInterface, JoltBodyID bodyID, float friction);

void JoltMoveKinematic(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
                      float x, float y, float z,
                      float qx, float qy, float qz, float qw,
                      float deltaTime);

int JoltApplyBuoyancyImpulse(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
                            float surfaceX, float surfaceY, float surfaceZ,
                            float normalX, float normalY, float normalZ,
                            float buoyancy, float linearDrag, float angularDrag,
                            float fluidVX, float fluidVY, float fluidVZ,
                            float gravityX, float gravityY, float gravityZ,
                            float deltaTime);

void JoltSetBodySensor(JoltPhysicsSystem system, JoltBodyID bodyID, int isSensor);
JoltShape JoltGetBodyShape(JoltPhysicsSystem system, JoltBodyID bodyID);
JoltBodyID JoltCreateBodyEx(JoltBodyInterface bodyInterface, JoltShape shape,
                           float x, float y, float z,
                           JoltMotionType motionType, int isSensor, int enhancedEdges);

#ifdef __cplusplus
}
#endif

#endif // JOLT_WRAPPER_BODY_H
