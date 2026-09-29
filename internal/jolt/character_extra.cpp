/*
 * CharacterVirtual with an inner kinematic rigid body (raycasts / CCD collide with the player).
 */

#include "wrapper/character.h"
#include "wrapper/physics.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/Character/CharacterVirtual.h>
#include <Jolt/Physics/Body/BodyFilter.h>
#include <Jolt/Physics/Collision/ShapeFilter.h>
#include <Jolt/Physics/Collision/Shape/Shape.h>
#include <Jolt/Core/TempAllocator.h>

using namespace JPH;

static void fillCharacterSettings(CharacterVirtualSettings &out, const JoltCharacterVirtualSettings *settings)
{
	out.mShape = static_cast<Shape *>(settings->shape);
	out.mUp = Vec3(settings->upX, settings->upY, settings->upZ);
	out.mMaxSlopeAngle = settings->maxSlopeAngle;
	out.mMass = settings->mass;
	out.mMaxStrength = settings->maxStrength;
	out.mShapeOffset = Vec3(settings->shapeOffsetX, settings->shapeOffsetY, settings->shapeOffsetZ);
	out.mBackFaceMode = settings->backFaceMode == JoltBackFaceModeIgnore ? EBackFaceMode::IgnoreBackFaces : EBackFaceMode::CollideWithBackFaces;
	out.mPredictiveContactDistance = settings->predictiveContactDistance;
	out.mMaxCollisionIterations = settings->maxCollisionIterations;
	out.mMaxConstraintIterations = settings->maxConstraintIterations;
	out.mMinTimeRemaining = settings->minTimeRemaining;
	out.mCollisionTolerance = settings->collisionTolerance;
	out.mCharacterPadding = settings->characterPadding;
	out.mMaxNumHits = settings->maxNumHits;
	out.mHitReductionCosMaxAngle = settings->hitReductionCosMaxAngle;
	out.mPenetrationRecoverySpeed = settings->penetrationRecoverySpeed;
	out.mEnhancedInternalEdgeRemoval = settings->enhancedInternalEdgeRemoval != 0;
}

JoltCharacterVirtual JoltCreateCharacterVirtualWithInner(JoltPhysicsSystem system,
														const JoltCharacterVirtualSettings *settings,
														float x, float y, float z,
														JoltShape innerShape, int innerLayer)
{
	if (system == nullptr || settings == nullptr || settings->shape == nullptr)
	{
		return nullptr;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	CharacterVirtualSettings vs;
	fillCharacterSettings(vs, settings);
	if (innerShape != nullptr)
	{
		vs.mInnerBodyShape = static_cast<Shape *>(innerShape);
		vs.mInnerBodyLayer = ObjectLayer(innerLayer);
	}
	CharacterVirtual *cv = new CharacterVirtual(&vs, RVec3(x, y, z), Quat::sIdentity(), 0, ps);
	return cv;
}

extern "C" void JoltCharacterGameUpdate(JoltCharacterVirtual character,
										JoltPhysicsSystem system,
										void *allocator,
										float deltaTime,
										float gravityX, float gravityY, float gravityZ)
{
	if (character == nullptr || system == nullptr || allocator == nullptr || deltaTime <= 0.0f)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	if (ps == nullptr)
	{
		return;
	}
	CharacterVirtual *cv = static_cast<CharacterVirtual *>(character);
	CharacterVirtual::ExtendedUpdateSettings settings;
	settings.mStickToFloorStepDown = Vec3(0, -0.55f, 0);
	settings.mWalkStairsStepUp = Vec3(0, 0.45f, 0);
	settings.mWalkStairsMinStepForward = 0.02f;
	settings.mWalkStairsStepForwardTest = 0.28f;
	settings.mWalkStairsStepDownExtra = Vec3(0, -0.05f, 0);
	const ObjectLayer moving = 1;
	BodyFilter bodyFilter;
	ShapeFilter shapeFilter;
	cv->ExtendedUpdate(
		deltaTime,
		Vec3(gravityX, gravityY, gravityZ),
		settings,
		ps->GetDefaultBroadPhaseLayerFilter(moving),
		ps->GetDefaultLayerFilter(moving),
		bodyFilter,
		shapeFilter,
		*static_cast<TempAllocator *>(allocator));
}

extern "C" JoltBodyID JoltCharacterInnerBody(JoltCharacterVirtual character)
{
	if (character == nullptr)
	{
		return nullptr;
	}
	CharacterVirtual *cv = static_cast<CharacterVirtual *>(character);
	BodyID id = cv->GetInnerBodyID();
	if (id.IsInvalid())
	{
		return nullptr;
	}
	return new BodyID(id);
}
