//go:build windows

package jolt

// #include "wrapper/contact_listener.h"
import "C"

// ContactEventType is a native Jolt contact callback kind.
type ContactEventType int

const (
	ContactAdded      ContactEventType = C.JOLT_CONTACT_ADDED
	ContactPersisted  ContactEventType = C.JOLT_CONTACT_PERSISTED
	ContactRemoved    ContactEventType = C.JOLT_CONTACT_REMOVED
)

// ContactEvent is a drained contact from the C++ queue (main thread only).
type ContactEvent struct {
	Type  ContactEventType
	BodyA uint32
	BodyB uint32
	X, Y, Z    float32
	NX, NY, NZ float32
}

// EnableContactListener installs the C++ ContactListener (queued, not //export).
func (ps *PhysicsSystem) EnableContactListener(on bool) {
	flag := C.int(0)
	if on {
		flag = 1
	}
	C.JoltSetContactListenerEnabled(ps.handle, flag)
}

// PollContactEvents drains up to max queued contacts. Partial drain if the
// buffer is smaller than the queue.
func (ps *PhysicsSystem) PollContactEvents(max int) []ContactEvent {
	if max <= 0 {
		max = 512
	}
	buf := make([]C.JoltContactEvent, max)
	n := int(C.JoltPollContactEvents(&buf[0], C.int(max)))
	if n <= 0 {
		return nil
	}
	out := make([]ContactEvent, n)
	for i := 0; i < n; i++ {
		c := buf[i]
		out[i] = ContactEvent{
			Type:  ContactEventType(c.eventType),
			BodyA: uint32(c.bodyA),
			BodyB: uint32(c.bodyB),
			X:     float32(c.x),
			Y:     float32(c.y),
			Z:     float32(c.z),
			NX:    float32(c.nx),
			NY:    float32(c.ny),
			NZ:    float32(c.nz),
		}
	}
	return out
}

// ClearContactEvents drops any undrained contact events.
func (ps *PhysicsSystem) ClearContactEvents() {
	C.JoltClearContactEvents()
}
