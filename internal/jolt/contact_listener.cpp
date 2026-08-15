/*
 * Jolt ContactListener — queue in C++ (mutex), drain from Go.
 * Never call into Go from these callbacks (Jolt worker threads).
 */

#include "wrapper/contact_listener.h"
#include "wrapper/physics.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/Body/Body.h>
#include <Jolt/Physics/Collision/ContactListener.h>
#include <Jolt/Physics/Collision/CollideShape.h>
#include <Jolt/Physics/Collision/Shape/SubShapeIDPair.h>

#include <deque>
#include <mutex>

using namespace JPH;

class EngineContactListener final : public ContactListener
{
public:
	ValidateResult OnContactValidate(const Body &inBody1, const Body &inBody2,
									 RVec3Arg inBaseOffset, const CollideShapeResult &inCollisionResult) override
	{
		(void)inBody1;
		(void)inBody2;
		(void)inBaseOffset;
		(void)inCollisionResult;
		return ValidateResult::AcceptAllContactsForThisBodyPair;
	}

	void OnContactAdded(const Body &inBody1, const Body &inBody2,
						const ContactManifold &inManifold, ContactSettings &ioSettings) override
	{
		(void)ioSettings;
		queueContact(JOLT_CONTACT_ADDED, inBody1, inBody2, inManifold);
	}

	void OnContactPersisted(const Body &inBody1, const Body &inBody2,
							const ContactManifold &inManifold, ContactSettings &ioSettings) override
	{
		(void)ioSettings;
		queueContact(JOLT_CONTACT_PERSISTED, inBody1, inBody2, inManifold);
	}

	void OnContactRemoved(const SubShapeIDPair &inSubShapePair) override
	{
		JoltContactEvent ev{};
		ev.eventType = JOLT_CONTACT_REMOVED;
		ev.bodyA = inSubShapePair.GetBody1ID().GetIndexAndSequenceNumber();
		ev.bodyB = inSubShapePair.GetBody2ID().GetIndexAndSequenceNumber();
		pushEvent(ev);
	}

	void pushEvent(JoltContactEvent ev)
	{
		std::lock_guard<std::mutex> lock(mMu);
		if (mQueue.size() >= kMaxQueued)
		{
			mQueue.pop_front();
		}
		mQueue.push_back(ev);
	}

	int poll(JoltContactEvent *outEvents, int maxEvents)
	{
		std::lock_guard<std::mutex> lock(mMu);
		int n = 0;
		while (n < maxEvents && !mQueue.empty())
		{
			outEvents[n] = mQueue.front();
			mQueue.pop_front();
			n++;
		}
		return n;
	}

	void clear()
	{
		std::lock_guard<std::mutex> lock(mMu);
		mQueue.clear();
	}

private:
	void queueContact(int kind, const Body &inBody1, const Body &inBody2, const ContactManifold &inManifold)
	{
		RVec3 p = inManifold.GetWorldSpaceContactPointOn1(0);
		Vec3 n = inManifold.mWorldSpaceNormal;
		JoltContactEvent ev{};
		ev.eventType = kind;
		ev.bodyA = inBody1.GetID().GetIndexAndSequenceNumber();
		ev.bodyB = inBody2.GetID().GetIndexAndSequenceNumber();
		ev.x = static_cast<float>(p.GetX());
		ev.y = static_cast<float>(p.GetY());
		ev.z = static_cast<float>(p.GetZ());
		ev.nx = n.GetX();
		ev.ny = n.GetY();
		ev.nz = n.GetZ();
		pushEvent(ev);
	}

	static constexpr size_t kMaxQueued = 4096;
	std::mutex mMu;
	std::deque<JoltContactEvent> mQueue;
};

static EngineContactListener g_ContactListener;

void JoltSetContactListenerEnabled(JoltPhysicsSystem system, int enabled)
{
	if (system == nullptr)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	if (enabled)
	{
		ps->SetContactListener(&g_ContactListener);
	}
	else
	{
		ps->SetContactListener(nullptr);
	}
}

int JoltPollContactEvents(JoltContactEvent *outEvents, int maxEvents)
{
	if (outEvents == nullptr || maxEvents <= 0)
	{
		return 0;
	}
	return g_ContactListener.poll(outEvents, maxEvents);
}

void JoltClearContactEvents(void)
{
	g_ContactListener.clear();
}
