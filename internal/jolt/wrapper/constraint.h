/*
 * Jolt Physics C Wrapper - Two-body constraints
 *
 * Vec3 at the C boundary. Opaque constraint handles for Go.
 */

#ifndef JOLT_WRAPPER_CONSTRAINT_H
#define JOLT_WRAPPER_CONSTRAINT_H

#include "physics.h"
#include "body.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef void* JoltConstraint;

JoltConstraint JoltCreateHingeConstraint(JoltPhysicsSystem system,
                                        JoltBodyID bodyA, JoltBodyID bodyB,
                                        float pivotX, float pivotY, float pivotZ,
                                        float axisX, float axisY, float axisZ);

JoltConstraint JoltCreatePointConstraint(JoltPhysicsSystem system,
                                        JoltBodyID bodyA, JoltBodyID bodyB,
                                        float pivotX, float pivotY, float pivotZ);

JoltConstraint JoltCreateSliderConstraint(JoltPhysicsSystem system,
                                         JoltBodyID bodyA, JoltBodyID bodyB,
                                         float pivotX, float pivotY, float pivotZ,
                                         float axisX, float axisY, float axisZ);

JoltConstraint JoltCreateDistanceConstraint(JoltPhysicsSystem system,
                                           JoltBodyID bodyA, JoltBodyID bodyB,
                                           float pivotX, float pivotY, float pivotZ,
                                           float minDist, float maxDist,
                                           float frequency, float damping);

void JoltRemoveConstraint(JoltPhysicsSystem system, JoltConstraint constraint);
void JoltDestroyConstraint(JoltConstraint constraint);

void JoltSetHingeLimits(JoltConstraint constraint, float minDeg, float maxDeg);
void JoltSetHingeFriction(JoltConstraint constraint, float maxFrictionTorque);
void JoltSetHingeMotor(JoltConstraint constraint, float targetDeg, float maxTorque);
void JoltDisableBodyPairCollision(JoltPhysicsSystem system, JoltBodyID bodyA, JoltBodyID bodyB);

#ifdef __cplusplus
}
#endif

#endif /* JOLT_WRAPPER_CONSTRAINT_H */
