#ifndef JOLT_WRAPPER_CLOTH_H
#define JOLT_WRAPPER_CLOTH_H

#include "body.h"

#ifdef __cplusplus
extern "C" {
#endif

/* pinFlags: bit0 top, bit1 bottom, bit2 left, bit3 right (hanging XY sheet). */
JoltBodyID JoltCreateCloth(JoltBodyInterface bodyInterface,
                           float x, float y, float z,
                           float width, float height,
                           int nx, int ny, int pinFlags,
                           float thickness, float damping, float gravityFactor);

int JoltGetClothVertexCount(JoltPhysicsSystem system, JoltBodyID bodyID);
int JoltGetClothVertices(JoltPhysicsSystem system, JoltBodyID bodyID,
                         float *xyz, int maxFloats);
void JoltApplyClothWind(JoltPhysicsSystem system, JoltBodyID bodyID,
                        float vx, float vy, float vz, unsigned int start, unsigned int step);
void JoltEnableSoftBodyOneWayContacts(JoltPhysicsSystem system, int enabled);

#ifdef __cplusplus
}
#endif

#endif
