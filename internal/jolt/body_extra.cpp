/*
 * Extra BodyInterface operations compiled into the Go jolt package.
 * Linked against prebuilt libJolt / libjolt_wrapper.
 */

#include "wrapper/body.h"
#include "wrapper/physics.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/Body/Body.h>
#include <Jolt/Physics/Body/BodyInterface.h>
#include <Jolt/Physics/Body/BodyLock.h>
#include <Jolt/Physics/Body/MotionProperties.h>
#include <Jolt/Physics/Body/MotionQuality.h>

using namespace JPH;

static const BodyID *asID(JoltBodyID bodyID)
{
	return static_cast<const BodyID *>(bodyID);
}

unsigned int JoltGetBodyIDValue(JoltBodyID bodyID)
{
	if (bodyID == nullptr)
	{
		return 0;
	}
	return asID(bodyID)->GetIndexAndSequenceNumber();
}

void JoltGetBodyRotation(const JoltBodyInterface bodyInterface,
						const JoltBodyID bodyID,
						float *x, float *y, float *z, float *w)
{
	const BodyInterface *bi = static_cast<const BodyInterface *>(bodyInterface);
	Quat q = bi->GetRotation(*asID(bodyID));
	*x = q.GetX();
	*y = q.GetY();
	*z = q.GetZ();
	*w = q.GetW();
}

void JoltSetBodyRotation(JoltBodyInterface bodyInterface,
						JoltBodyID bodyID,
						float x, float y, float z, float w)
{
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->SetRotation(*asID(bodyID), Quat(x, y, z, w), EActivation::DontActivate);
}

void JoltGetBodyLinearVelocity(const JoltBodyInterface bodyInterface,
							  const JoltBodyID bodyID,
							  float *x, float *y, float *z)
{
	const BodyInterface *bi = static_cast<const BodyInterface *>(bodyInterface);
	Vec3 v = bi->GetLinearVelocity(*asID(bodyID));
	*x = v.GetX();
	*y = v.GetY();
	*z = v.GetZ();
}

void JoltSetBodyLinearVelocity(JoltBodyInterface bodyInterface,
							  JoltBodyID bodyID,
							  float x, float y, float z)
{
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->SetLinearVelocity(*asID(bodyID), Vec3(x, y, z));
}

void JoltGetBodyAngularVelocity(const JoltBodyInterface bodyInterface,
							   const JoltBodyID bodyID,
							   float *x, float *y, float *z)
{
	const BodyInterface *bi = static_cast<const BodyInterface *>(bodyInterface);
	Vec3 v = bi->GetAngularVelocity(*asID(bodyID));
	*x = v.GetX();
	*y = v.GetY();
	*z = v.GetZ();
}

void JoltSetBodyAngularVelocity(JoltBodyInterface bodyInterface,
							   JoltBodyID bodyID,
							   float x, float y, float z)
{
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->SetAngularVelocity(*asID(bodyID), Vec3(x, y, z));
}

void JoltAddForce(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
				 float x, float y, float z)
{
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->AddForce(*asID(bodyID), Vec3(x, y, z), EActivation::Activate);
}

void JoltAddImpulse(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
				   float x, float y, float z)
{
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->AddImpulse(*asID(bodyID), Vec3(x, y, z));
}

void JoltAddTorque(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
				  float x, float y, float z)
{
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->AddTorque(*asID(bodyID), Vec3(x, y, z), EActivation::Activate);
}

void JoltAddAngularImpulse(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
						  float x, float y, float z)
{
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->AddAngularImpulse(*asID(bodyID), Vec3(x, y, z));
}

void JoltSetBodyMotionQuality(JoltBodyInterface bodyInterface, JoltBodyID bodyID, int linearCast)
{
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	EMotionQuality q = linearCast ? EMotionQuality::LinearCast : EMotionQuality::Discrete;
	bi->SetMotionQuality(*asID(bodyID), q);
}

void JoltSetBodyMass(JoltPhysicsSystem system, JoltBodyID bodyID, float mass)
{
	if (system == nullptr || bodyID == nullptr || mass <= 0.0001f)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	BodyLockWrite lock(ps->GetBodyLockInterface(), *asID(bodyID));
	if (!lock.Succeeded())
	{
		return;
	}
	Body &body = lock.GetBody();
	if (!body.IsDynamic())
	{
		return;
	}
	MotionProperties *mp = body.GetMotionProperties();
	if (mp == nullptr)
	{
		return;
	}
	mp->ScaleToMass(mass);
}

void JoltAddForceAtPosition(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
							float fx, float fy, float fz,
							float px, float py, float pz)
{
	if (bodyInterface == nullptr || bodyID == nullptr)
	{
		return;
	}
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->AddForce(*asID(bodyID), Vec3(fx, fy, fz), RVec3(px, py, pz), EActivation::Activate);
}

void JoltAddLocalImpulse(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
						float lx, float ly, float lz)
{
	if (bodyInterface == nullptr || bodyID == nullptr)
	{
		return;
	}
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	Quat rot = bi->GetRotation(*asID(bodyID));
	Vec3 world = rot * Vec3(lx, ly, lz);
	bi->AddImpulse(*asID(bodyID), world);
}

void JoltSetGravityFactor(JoltPhysicsSystem system, JoltBodyID bodyID, float scale)
{
	if (system == nullptr || bodyID == nullptr)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	BodyLockWrite lock(ps->GetBodyLockInterface(), *asID(bodyID));
	if (!lock.Succeeded())
	{
		return;
	}
	Body &body = lock.GetBody();
	if (!body.IsDynamic())
	{
		return;
	}
	body.GetMotionProperties()->SetGravityFactor(scale);
}

void JoltSetRestitution(JoltBodyInterface bodyInterface, JoltBodyID bodyID, float restitution)
{
	if (bodyInterface == nullptr || bodyID == nullptr)
	{
		return;
	}
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->SetRestitution(*asID(bodyID), restitution);
}

void JoltSetLinearDamping(JoltPhysicsSystem system, JoltBodyID bodyID, float damping)
{
	if (system == nullptr || bodyID == nullptr || damping < 0.0f)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	BodyLockWrite lock(ps->GetBodyLockInterface(), *asID(bodyID));
	if (!lock.Succeeded())
	{
		return;
	}
	Body &body = lock.GetBody();
	if (!body.IsDynamic())
	{
		return;
	}
	body.GetMotionProperties()->SetLinearDamping(damping);
}

void JoltSetAngularDamping(JoltPhysicsSystem system, JoltBodyID bodyID, float damping)
{
	if (system == nullptr || bodyID == nullptr || damping < 0.0f)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	BodyLockWrite lock(ps->GetBodyLockInterface(), *asID(bodyID));
	if (!lock.Succeeded())
	{
		return;
	}
	Body &body = lock.GetBody();
	if (!body.IsDynamic())
	{
		return;
	}
	body.GetMotionProperties()->SetAngularDamping(damping);
}

void JoltSetFriction(JoltBodyInterface bodyInterface, JoltBodyID bodyID, float friction)
{
	if (bodyInterface == nullptr || bodyID == nullptr)
	{
		return;
	}
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->SetFriction(*asID(bodyID), friction);
}

void JoltMoveKinematic(JoltBodyInterface bodyInterface, JoltBodyID bodyID,
					float x, float y, float z,
					float qx, float qy, float qz, float qw,
					float deltaTime)
{
	if (bodyInterface == nullptr || bodyID == nullptr)
	{
		return;
	}
	if (deltaTime < 1.0e-6f)
	{
		deltaTime = 1.0e-6f;
	}
	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	bi->MoveKinematic(*asID(bodyID), RVec3(x, y, z), Quat(qx, qy, qz, qw), deltaTime);
}

void JoltSetGravity(JoltPhysicsSystem system, float x, float y, float z)
{
	if (system == nullptr)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	ps->SetGravity(Vec3(x, y, z));
}
