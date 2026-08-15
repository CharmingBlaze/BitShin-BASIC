/*
 * Jolt Physics C Wrapper - Contact listener
 *
 * Callbacks run on Jolt worker threads. Events are queued in C++ under a
 * mutex and drained from Go during UpdateWorld. Do not call into Go here.
 */

#ifndef JOLT_WRAPPER_CONTACT_LISTENER_H
#define JOLT_WRAPPER_CONTACT_LISTENER_H

#include "physics.h"

#ifdef __cplusplus
extern "C" {
#endif

enum {
    JOLT_CONTACT_ADDED = 1,
    JOLT_CONTACT_PERSISTED = 2,
    JOLT_CONTACT_REMOVED = 3
};

typedef struct JoltContactEvent {
    int eventType;
    unsigned int bodyA;
    unsigned int bodyB;
    float x, y, z;
    float nx, ny, nz;
} JoltContactEvent;

void JoltSetContactListenerEnabled(JoltPhysicsSystem system, int enabled);
int JoltPollContactEvents(JoltContactEvent* outEvents, int maxEvents);
void JoltClearContactEvents(void);

#ifdef __cplusplus
}
#endif

#endif /* JOLT_WRAPPER_CONTACT_LISTENER_H */
