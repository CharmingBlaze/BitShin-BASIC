/*
 * NarrowPhase CastShape / CollidePoint for BASIC ShapeCast and OverlapPoint.
 */

#include "wrapper/query.h"
#include "wrapper/physics.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/Body/BodyID.h>
#include <Jolt/Physics/Collision/NarrowPhaseQuery.h>
#include <Jolt/Physics/Collision/ShapeCast.h>
#include <Jolt/Physics/Collision/CollisionCollectorImpl.h>
#include <Jolt/Physics/Collision/CollidePointResult.h>
#include <Jolt/Physics/Collision/Shape/Shape.h>

using namespace JPH;

int JoltCastShape(JoltPhysicsSystem system, JoltShape shape,
				 float x, float y, float z,
				 float dx, float dy, float dz,
				 JoltRaycastHit *outHit)
{
	if (system == nullptr || shape == nullptr || outHit == nullptr)
	{
		return 0;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	RMat44 start = RMat44::sTranslation(RVec3(x, y, z));
	RShapeCast shapeCast = RShapeCast::sFromWorldTransform(static_cast<Shape *>(shape), Vec3::sOne(), start, Vec3(dx, dy, dz));
	ShapeCastSettings settings;
	ClosestHitCollisionCollector<CastShapeCollector> collector;
	ps->GetNarrowPhaseQuery().CastShape(shapeCast, settings, shapeCast.mCenterOfMassStart.GetTranslation(), collector);
	if (!collector.HadHit())
	{
		return 0;
	}
	const ShapeCastResult &hit = collector.mHit;
	RVec3 p = shapeCast.GetPointOnRay(hit.mFraction);
	outHit->bodyID = new BodyID(hit.mBodyID2);
	outHit->hitPointX = float(p.GetX());
	outHit->hitPointY = float(p.GetY());
	outHit->hitPointZ = float(p.GetZ());
	Vec3 n = -hit.mPenetrationAxis.NormalizedOr(Vec3::sAxisY());
	outHit->normalX = n.GetX();
	outHit->normalY = n.GetY();
	outHit->normalZ = n.GetZ();
	outHit->fraction = hit.mFraction;
	return 1;
}

int JoltCollidePoint(JoltPhysicsSystem system,
					float x, float y, float z,
					JoltRaycastHit *outHit)
{
	if (system == nullptr || outHit == nullptr)
	{
		return 0;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	ClosestHitCollisionCollector<CollidePointCollector> collector;
	ps->GetNarrowPhaseQuery().CollidePoint(RVec3(x, y, z), collector);
	if (!collector.HadHit())
	{
		return 0;
	}
	outHit->bodyID = new BodyID(collector.mHit.mBodyID);
	outHit->hitPointX = x;
	outHit->hitPointY = y;
	outHit->hitPointZ = z;
	outHit->normalX = 0;
	outHit->normalY = 1;
	outHit->normalZ = 0;
	outHit->fraction = 0;
	return 1;
}
