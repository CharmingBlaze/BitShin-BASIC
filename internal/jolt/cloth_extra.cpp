/*
 * Rectangular Jolt soft-body cloth (ezEngine JoltClothSheetComponent recipe).
 */

#include "wrapper/cloth.h"
#include "wrapper/physics.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/Body/Body.h>
#include <Jolt/Physics/Body/BodyInterface.h>
#include <Jolt/Physics/Body/BodyLock.h>
#include <Jolt/Physics/SoftBody/SoftBodyCreationSettings.h>
#include <Jolt/Physics/SoftBody/SoftBodyMotionProperties.h>
#include <Jolt/Physics/SoftBody/SoftBodyContactListener.h>
#include <Jolt/Physics/SoftBody/SoftBodySharedSettings.h>

#include <cmath>

using namespace JPH;

static const BodyID *asID(JoltBodyID bodyID)
{
	return static_cast<const BodyID *>(bodyID);
}

static uint32 clothIdx(int nx, int x, int y)
{
	return static_cast<uint32>(x + y * nx);
}

class OneWaySoftBodyListener final : public SoftBodyContactListener
{
public:
	SoftBodyValidateResult OnSoftBodyContactValidate(const Body &, const Body &,
													 SoftBodyContactSettings &ioSettings) override
	{
		ioSettings.mInvMassScale2 = 0.0f;
		ioSettings.mInvInertiaScale2 = 0.0f;
		return SoftBodyValidateResult::AcceptContact;
	}
};

static OneWaySoftBodyListener g_SoftBodyOneWay;

void JoltEnableSoftBodyOneWayContacts(JoltPhysicsSystem system, int enabled)
{
	if (system == nullptr)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	ps->SetSoftBodyContactListener(enabled ? &g_SoftBodyOneWay : nullptr);
}

JoltBodyID JoltCreateCloth(JoltBodyInterface bodyInterface,
						   float x, float y, float z,
						   float width, float height,
						   int nx, int ny, int pinFlags,
						   float thickness, float damping, float gravityFactor)
{
	if (bodyInterface == nullptr || nx < 2 || ny < 2)
	{
		return nullptr;
	}
	if (nx > 24)
	{
		nx = 24;
	}
	if (ny > 24)
	{
		ny = 24;
	}
	if (width < 0.05f)
	{
		width = 0.05f;
	}
	if (height < 0.05f)
	{
		height = 0.05f;
	}
	if (thickness < 0.0f)
	{
		thickness = 0.05f;
	}
	if (damping < 0.0f)
	{
		damping = 0.5f;
	}

	SoftBodySharedSettings *settings = new SoftBodySharedSettings;
	const float sx = width / static_cast<float>(nx - 1);
	const float sy = height / static_cast<float>(ny - 1);
	const float invMass = 1.0f / 0.2f;

	for (int j = 0; j < ny; ++j)
	{
		for (int i = 0; i < nx; ++i)
		{
			SoftBodySharedSettings::Vertex v;
			v.mPosition = Float3(static_cast<float>(i) * sx, -static_cast<float>(j) * sy, 0.0f);
			v.mInvMass = invMass;
			settings->mVertices.push_back(v);
		}
	}

	auto pinRow = [&](int j) {
		for (int i = 0; i < nx; ++i)
		{
			settings->mVertices[clothIdx(nx, i, j)].mInvMass = 0.0f;
		}
	};
	auto pinCol = [&](int i) {
		for (int j = 0; j < ny; ++j)
		{
			settings->mVertices[clothIdx(nx, i, j)].mInvMass = 0.0f;
		}
	};
	if (pinFlags == 0)
	{
		pinFlags = 1;
	}
	if ((pinFlags & 1) != 0)
	{
		pinRow(0);
	}
	if ((pinFlags & 2) != 0)
	{
		pinRow(ny - 1);
	}
	if ((pinFlags & 4) != 0)
	{
		pinCol(0);
	}
	if ((pinFlags & 8) != 0)
	{
		pinCol(nx - 1);
	}

	for (int j = 0; j < ny; ++j)
	{
		for (int i = 0; i < nx; ++i)
		{
			SoftBodySharedSettings::Edge e;
			e.mCompliance = 0.0008f;
			e.mVertex[0] = clothIdx(nx, i, j);
			if (i < nx - 1)
			{
				e.mVertex[1] = clothIdx(nx, i + 1, j);
				settings->mEdgeConstraints.push_back(e);
			}
			if (j < ny - 1)
			{
				e.mVertex[1] = clothIdx(nx, i, j + 1);
				settings->mEdgeConstraints.push_back(e);
			}
			if (i < nx - 1 && j < ny - 1)
			{
				e.mVertex[1] = clothIdx(nx, i + 1, j + 1);
				settings->mEdgeConstraints.push_back(e);
				e.mVertex[0] = clothIdx(nx, i + 1, j);
				e.mVertex[1] = clothIdx(nx, i, j + 1);
				settings->mEdgeConstraints.push_back(e);
			}
		}
	}
	settings->CalculateEdgeLengths();

	for (int j = 0; j < ny - 1; ++j)
	{
		for (int i = 0; i < nx - 1; ++i)
		{
			SoftBodySharedSettings::Face f;
			f.mVertex[0] = clothIdx(nx, i, j);
			f.mVertex[1] = clothIdx(nx, i, j + 1);
			f.mVertex[2] = clothIdx(nx, i + 1, j + 1);
			settings->AddFace(f);
			f.mVertex[1] = clothIdx(nx, i + 1, j + 1);
			f.mVertex[2] = clothIdx(nx, i + 1, j);
			settings->AddFace(f);
		}
	}
	settings->Optimize();

	SoftBodyCreationSettings cloth(settings, RVec3(x, y, z), Quat::sIdentity(), ObjectLayer(1));
	cloth.mVertexRadius = thickness;
	cloth.mPressure = 0.0f;
	cloth.mLinearDamping = damping;
	cloth.mGravityFactor = gravityFactor <= 0.0f ? 1.0f : gravityFactor;
	cloth.mFacesDoubleSided = true;
	cloth.mMakeRotationIdentity = true;
	cloth.mUpdatePosition = true;
	cloth.mAllowSleeping = false;

	BodyInterface *bi = static_cast<BodyInterface *>(bodyInterface);
	BodyID id = bi->CreateAndAddSoftBody(cloth, EActivation::Activate);
	if (id.IsInvalid())
	{
		return nullptr;
	}
	return new BodyID(id);
}

int JoltGetClothVertexCount(JoltPhysicsSystem system, JoltBodyID bodyID)
{
	if (system == nullptr || bodyID == nullptr)
	{
		return 0;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	BodyLockRead lock(ps->GetBodyLockInterface(), *asID(bodyID));
	if (!lock.Succeeded())
	{
		return 0;
	}
	const Body &body = lock.GetBody();
	if (!body.IsSoftBody())
	{
		return 0;
	}
	const SoftBodyMotionProperties *mp = static_cast<const SoftBodyMotionProperties *>(body.GetMotionProperties());
	return static_cast<int>(mp->GetVertices().size());
}

int JoltGetClothVertices(JoltPhysicsSystem system, JoltBodyID bodyID, float *xyz, int maxFloats)
{
	if (system == nullptr || bodyID == nullptr || xyz == nullptr || maxFloats < 3)
	{
		return 0;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	BodyLockRead lock(ps->GetBodyLockInterface(), *asID(bodyID));
	if (!lock.Succeeded())
	{
		return 0;
	}
	const Body &body = lock.GetBody();
	if (!body.IsSoftBody())
	{
		return 0;
	}
	const SoftBodyMotionProperties *mp = static_cast<const SoftBodyMotionProperties *>(body.GetMotionProperties());
	const Array<SoftBodyVertex> &verts = mp->GetVertices();
	int n = static_cast<int>(verts.size());
	int maxV = maxFloats / 3;
	if (n > maxV)
	{
		n = maxV;
	}
	for (int i = 0; i < n; ++i)
	{
		Vec3 p = verts[static_cast<size_t>(i)].mPosition;
		xyz[i * 3 + 0] = p.GetX();
		xyz[i * 3 + 1] = p.GetY();
		xyz[i * 3 + 2] = p.GetZ();
	}
	return n;
}

void JoltApplyClothWind(JoltPhysicsSystem system, JoltBodyID bodyID,
						float vx, float vy, float vz, unsigned int start, unsigned int step)
{
	if (system == nullptr || bodyID == nullptr)
	{
		return;
	}
	if (step < 1)
	{
		step = 1;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	BodyLockWrite lock(ps->GetBodyLockInterface(), *asID(bodyID));
	if (!lock.Succeeded())
	{
		return;
	}
	Body &body = lock.GetBody();
	if (!body.IsSoftBody())
	{
		return;
	}
	body.SetAllowSleeping(false);
	if (!body.IsActive())
	{
		ps->GetBodyInterfaceNoLock().ActivateBody(*asID(bodyID));
	}
	SoftBodyMotionProperties *mp = static_cast<SoftBodyMotionProperties *>(body.GetMotionProperties());
	Array<SoftBodyVertex> &verts = mp->GetVertices();
	float phase = static_cast<float>(start) * 0.11f;
	const float maxSp = 14.0f;
	const float maxSpSq = maxSp * maxSp;
	for (size_t i = 0; i < verts.size(); ++i)
	{
		if (verts[i].mInvMass <= 0.0f)
		{
			continue;
		}
		float ripple = std::sin(phase + static_cast<float>(i) * 0.37f);
		Vec3 add(vx * 0.28f, vy * 0.28f, vz * 0.28f + 0.22f * ripple);
		if (step > 1 && ((i + start) % step) != 0)
		{
			add *= 0.35f;
		}
		Vec3 v = verts[i].mVelocity + add;
		if (v.LengthSq() > maxSpSq)
		{
			v = v.Normalized() * maxSp;
		}
		verts[i].mVelocity = v;
	}
}
