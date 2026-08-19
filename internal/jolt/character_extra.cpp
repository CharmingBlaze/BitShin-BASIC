/*
 * CharacterVirtual with an inner kinematic rigid body (raycasts / CCD collide with the player).
 */

#include "wrapper/character.h"
#include "wrapper/physics.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/Character/CharacterVirtual.h>
#include <Jolt/Physics/Collision/Shape/Shape.h>

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
