/*
 * Jolt two-body constraints (hinge, point, slider, distance/spring).
 * Vec3 at the C boundary.
 */

#include "wrapper/constraint.h"
#include "wrapper/physics.h"
#include "wrapper/body.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/Body/Body.h>
#include <Jolt/Physics/Body/BodyLock.h>
#include <Jolt/Physics/Constraints/HingeConstraint.h>
#include <Jolt/Physics/Constraints/PointConstraint.h>
#include <Jolt/Physics/Constraints/SliderConstraint.h>
#include <Jolt/Physics/Constraints/DistanceConstraint.h>
#include <Jolt/Physics/Body/BodyInterface.h>
#include <Jolt/Physics/Collision/CollisionGroup.h>
#include <Jolt/Physics/Collision/GroupFilterTable.h>
#include <unordered_map>

using namespace JPH;

static Body *tryBody(PhysicsSystem *ps, JoltBodyID id)
{
	if (ps == nullptr)
	{
		return nullptr;
	}
	if (id == nullptr)
	{
		return &Body::sFixedToWorld;
	}
	return ps->GetBodyLockInterface().TryGetBody(*static_cast<const BodyID *>(id));
}

static Vec3 unitOrY(float x, float y, float z)
{
	Vec3 v(x, y, z);
	if (v.LengthSq() < 1.0e-8f)
	{
		return Vec3::sAxisY();
	}
	return v.Normalized();
}

// One shared GroupFilterTable so a body can disable collision with
// several partners (carriage+left prong, then carriage+right prong)
// without overwriting the previous filter.
static Ref<GroupFilterTable> sPairFilter;
static std::unordered_map<uint32, CollisionGroup::SubGroupID> sBodySub;
static CollisionGroup::SubGroupID sNextSub = 0;
static const uint32 kPairGroupID = 1;
static const uint32 kMaxSub = 256;

static CollisionGroup::SubGroupID subGroupOf(BodyInterface &bi, const BodyID &id)
{
	const uint32 key = id.GetIndexAndSequenceNumber();
	auto it = sBodySub.find(key);
	if (it != sBodySub.end())
	{
		return it->second;
	}
	if (sPairFilter == nullptr)
	{
		sPairFilter = new GroupFilterTable(kMaxSub);
	}
	CollisionGroup::SubGroupID sub = sNextSub;
	if (sNextSub + 1 < kMaxSub)
	{
		sNextSub++;
	}
	sBodySub[key] = sub;
	bi.SetCollisionGroup(id, CollisionGroup(sPairFilter, kPairGroupID, sub));
	return sub;
}

static void disableConstrainedCollision(PhysicsSystem *ps, Body *b1, Body *b2, JoltBodyID idA, JoltBodyID idB)
{
	if (ps == nullptr || b1 == nullptr || b2 == nullptr)
	{
		return;
	}
	if (b1 == &Body::sFixedToWorld || b2 == &Body::sFixedToWorld)
	{
		return;
	}
	if (idA == nullptr || idB == nullptr)
	{
		return;
	}
	BodyInterface &bi = ps->GetBodyInterface();
	const BodyID &ba = *static_cast<const BodyID *>(idA);
	const BodyID &bb = *static_cast<const BodyID *>(idB);
	CollisionGroup::SubGroupID sa = subGroupOf(bi, ba);
	CollisionGroup::SubGroupID sb = subGroupOf(bi, bb);
	if (sPairFilter != nullptr)
	{
		sPairFilter->DisableCollision(sa, sb);
	}
}

JoltConstraint JoltCreateHingeConstraint(JoltPhysicsSystem system,
										JoltBodyID bodyA, JoltBodyID bodyB,
										float pivotX, float pivotY, float pivotZ,
										float axisX, float axisY, float axisZ)
{
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	Body *b1 = tryBody(ps, bodyA);
	Body *b2 = tryBody(ps, bodyB);
	if (b1 == nullptr || b2 == nullptr || b1 == b2)
	{
		return nullptr;
	}
	Vec3 axis = unitOrY(axisX, axisY, axisZ);
	Vec3 normal = axis.GetNormalizedPerpendicular();
	HingeConstraintSettings settings;
	settings.mSpace = EConstraintSpace::WorldSpace;
	settings.mPoint1 = RVec3(pivotX, pivotY, pivotZ);
	settings.mPoint2 = RVec3(pivotX, pivotY, pivotZ);
	settings.mHingeAxis1 = axis;
	settings.mHingeAxis2 = axis;
	settings.mNormalAxis1 = normal;
	settings.mNormalAxis2 = normal;
	settings.mMaxFrictionTorque = 2.0f;
	TwoBodyConstraint *c = settings.Create(*b1, *b2);
	ps->AddConstraint(c);
	disableConstrainedCollision(ps, b1, b2, bodyA, bodyB);
	return static_cast<JoltConstraint>(c);
}

JoltConstraint JoltCreatePointConstraint(JoltPhysicsSystem system,
										JoltBodyID bodyA, JoltBodyID bodyB,
										float pivotX, float pivotY, float pivotZ)
{
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	Body *b1 = tryBody(ps, bodyA);
	Body *b2 = tryBody(ps, bodyB);
	if (b1 == nullptr || b2 == nullptr || b1 == b2)
	{
		return nullptr;
	}
	PointConstraintSettings settings;
	settings.mSpace = EConstraintSpace::WorldSpace;
	settings.mPoint1 = RVec3(pivotX, pivotY, pivotZ);
	settings.mPoint2 = RVec3(pivotX, pivotY, pivotZ);
	TwoBodyConstraint *c = settings.Create(*b1, *b2);
	ps->AddConstraint(c);
	disableConstrainedCollision(ps, b1, b2, bodyA, bodyB);
	return static_cast<JoltConstraint>(c);
}

JoltConstraint JoltCreateSliderConstraint(JoltPhysicsSystem system,
										 JoltBodyID bodyA, JoltBodyID bodyB,
										 float pivotX, float pivotY, float pivotZ,
										 float axisX, float axisY, float axisZ)
{
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	Body *b1 = tryBody(ps, bodyA);
	Body *b2 = tryBody(ps, bodyB);
	if (b1 == nullptr || b2 == nullptr || b1 == b2)
	{
		return nullptr;
	}
	Vec3 axis = unitOrY(axisX, axisY, axisZ);
	SliderConstraintSettings settings;
	settings.mSpace = EConstraintSpace::WorldSpace;
	settings.mAutoDetectPoint = false;
	settings.mPoint1 = RVec3(pivotX, pivotY, pivotZ);
	settings.mPoint2 = RVec3(pivotX, pivotY, pivotZ);
	settings.SetSliderAxis(axis);
	TwoBodyConstraint *c = settings.Create(*b1, *b2);
	ps->AddConstraint(c);
	disableConstrainedCollision(ps, b1, b2, bodyA, bodyB);
	return static_cast<JoltConstraint>(c);
}

JoltConstraint JoltCreateDistanceConstraint(JoltPhysicsSystem system,
										   JoltBodyID bodyA, JoltBodyID bodyB,
										   float pivotX, float pivotY, float pivotZ,
										   float minDist, float maxDist,
										   float frequency, float damping)
{
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	Body *b1 = tryBody(ps, bodyA);
	Body *b2 = tryBody(ps, bodyB);
	if (b1 == nullptr || b2 == nullptr || b1 == b2)
	{
		return nullptr;
	}
	DistanceConstraintSettings settings;
	settings.mSpace = EConstraintSpace::WorldSpace;
	settings.mPoint1 = RVec3(pivotX, pivotY, pivotZ);
	settings.mPoint2 = RVec3(pivotX, pivotY, pivotZ);
	settings.mMinDistance = minDist;
	settings.mMaxDistance = maxDist;
	if (frequency > 0)
	{
		settings.mLimitsSpringSettings.mFrequency = frequency;
		settings.mLimitsSpringSettings.mDamping = damping;
	}
	TwoBodyConstraint *c = settings.Create(*b1, *b2);
	ps->AddConstraint(c);
	disableConstrainedCollision(ps, b1, b2, bodyA, bodyB);
	return static_cast<JoltConstraint>(c);
}

void JoltRemoveConstraint(JoltPhysicsSystem system, JoltConstraint constraint)
{
	if (system == nullptr || constraint == nullptr)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	ps->RemoveConstraint(static_cast<Constraint *>(constraint));
}

void JoltDestroyConstraint(JoltConstraint constraint)
{
	if (constraint == nullptr)
	{
		return;
	}
	delete static_cast<Constraint *>(constraint);
}

static HingeConstraint *asHinge(JoltConstraint constraint)
{
	if (constraint == nullptr)
	{
		return nullptr;
	}
	Constraint *c = static_cast<Constraint *>(constraint);
	if (c->GetSubType() != EConstraintSubType::Hinge)
	{
		return nullptr;
	}
	return static_cast<HingeConstraint *>(c);
}

void JoltSetHingeLimits(JoltConstraint constraint, float minDeg, float maxDeg)
{
	HingeConstraint *h = asHinge(constraint);
	if (h == nullptr)
	{
		return;
	}
	float minR = DegreesToRadians(minDeg);
	float maxR = DegreesToRadians(maxDeg);
	if (minR > 0.0f)
	{
		minR = 0.0f;
	}
	if (maxR < 0.0f)
	{
		maxR = 0.0f;
	}
	h->SetLimits(minR, maxR);
}

void JoltSetHingeFriction(JoltConstraint constraint, float maxFrictionTorque)
{
	HingeConstraint *h = asHinge(constraint);
	if (h == nullptr)
	{
		return;
	}
	if (maxFrictionTorque < 0.0f)
	{
		maxFrictionTorque = 0.0f;
	}
	h->SetMaxFrictionTorque(maxFrictionTorque);
}

void JoltSetHingeMotor(JoltConstraint constraint, float targetDeg, float maxTorque)
{
	HingeConstraint *h = asHinge(constraint);
	if (h == nullptr)
	{
		return;
	}
	if (maxTorque <= 0.0f)
	{
		h->SetMotorState(EMotorState::Off);
		return;
	}
	MotorSettings &motor = h->GetMotorSettings();
	motor.mSpringSettings.mFrequency = 10.0f;
	motor.mSpringSettings.mDamping = 1.0f;
	motor.SetTorqueLimit(maxTorque);
	h->SetTargetAngle(DegreesToRadians(targetDeg));
	h->SetMotorState(EMotorState::Position);
}

void JoltDisableBodyPairCollision(JoltPhysicsSystem system, JoltBodyID bodyA, JoltBodyID bodyB)
{
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	Body *b1 = tryBody(ps, bodyA);
	Body *b2 = tryBody(ps, bodyB);
	disableConstrainedCollision(ps, b1, b2, bodyA, bodyB);
}
