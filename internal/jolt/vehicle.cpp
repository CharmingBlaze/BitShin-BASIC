/*
 * 4-wheel car, 2-wheel motorcycle, and tracked tank via Jolt VehicleConstraint.
 * Constraint is registered as a PhysicsStepListener so it ticks with the world.
 */

#include "wrapper/vehicle.h"
#include "wrapper/physics.h"
#include "wrapper/body.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/Body/Body.h>
#include <Jolt/Physics/Body/BodyLock.h>
#include <Jolt/Physics/Vehicle/VehicleConstraint.h>
#include <Jolt/Physics/Vehicle/WheeledVehicleController.h>
#include <Jolt/Physics/Vehicle/MotorcycleController.h>
#include <Jolt/Physics/Vehicle/TrackedVehicleController.h>
#include <Jolt/Physics/Vehicle/VehicleCollisionTester.h>

using namespace JPH;

static float clampf(float v, float lo, float hi)
{
	if (v < lo)
	{
		return lo;
	}
	if (v > hi)
	{
		return hi;
	}
	return v;
}

static float nonzeroRatio(float v)
{
	if (v > -0.05f && v < 0.05f)
	{
		if (v < 0.0f)
		{
			return -0.05f;
		}
		return 0.05f;
	}
	return v;
}

JoltConstraint JoltCreateVehicle(JoltPhysicsSystem system, JoltBodyID bodyID, const JoltVehicleSettings *settings)
{
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	if (ps == nullptr || bodyID == nullptr)
	{
		return nullptr;
	}
	BodyLockWrite lock(ps->GetBodyLockInterface(), *static_cast<const BodyID *>(bodyID));
	if (!lock.Succeeded())
	{
		return nullptr;
	}
	Body &body = lock.GetBody();
	if (!body.IsDynamic())
	{
		return nullptr;
	}

	float hw = 0.9f;
	float hy = 0.2f;
	float hl = 1.6f;
	float wr = 0.3f;
	float ww = 0.12f;
	int kind = JoltVehicleCar;
	ObjectLayer layer = 1;
	if (settings != nullptr)
	{
		if (settings->halfWidth > 0.05f)
		{
			hw = settings->halfWidth;
		}
		if (settings->halfHeight > 0.02f)
		{
			hy = settings->halfHeight;
		}
		if (settings->halfLength > 0.05f)
		{
			hl = settings->halfLength;
		}
		if (settings->wheelRadius > 0.05f)
		{
			wr = settings->wheelRadius;
		}
		if (settings->wheelWidth > 0.02f)
		{
			ww = settings->wheelWidth;
		}
		if (settings->kind >= JoltVehicleCar && settings->kind <= JoltVehicleTank)
		{
			kind = settings->kind;
		}
		if (settings->objectLayer >= 0)
		{
			layer = ObjectLayer(settings->objectLayer);
		}
	}

	VehicleConstraintSettings vehicle;
	vehicle.mUp = Vec3::sAxisY();
	vehicle.mForward = Vec3::sAxisZ();
	vehicle.mMaxPitchRollAngle = DegreesToRadians(kind == JoltVehicleMotorcycle ? 80.0f : 60.0f);

	if (kind == JoltVehicleMotorcycle)
	{
		WheelSettingsWV *front = new WheelSettingsWV();
		front->mPosition = Vec3(0.0f, -hy, hl);
		front->mRadius = wr;
		front->mWidth = ww * 0.6f;
		front->mSuspensionMinLength = 0.15f;
		front->mSuspensionMaxLength = 0.4f;
		front->mMaxSteerAngle = DegreesToRadians(35.0f);
		front->mMaxHandBrakeTorque = 0.0f;
		WheelSettingsWV *rear = new WheelSettingsWV();
		rear->mPosition = Vec3(0.0f, -hy, -hl);
		rear->mRadius = wr;
		rear->mWidth = ww * 0.7f;
		rear->mSuspensionMinLength = 0.15f;
		rear->mSuspensionMaxLength = 0.4f;
		rear->mMaxSteerAngle = 0.0f;
		vehicle.mWheels = {front, rear};

		MotorcycleControllerSettings *controller = new MotorcycleControllerSettings();
		controller->mDifferentials.resize(1);
		controller->mDifferentials[0].mLeftWheel = 1;
		controller->mDifferentials[0].mRightWheel = -1;
		controller->mEngine.mMaxTorque = 280.0f;
		controller->mMaxLeanAngle = DegreesToRadians(45.0f);
		controller->mLeanSpringConstant = 5000.0f;
		controller->mLeanSpringDamping = 1000.0f;
		vehicle.mController = controller;
	}
	else if (kind == JoltVehicleTank)
	{
		const Vec3 pos[4] = {
			Vec3(hw, -hy, hl),
			Vec3(-hw, -hy, hl),
			Vec3(hw, -hy, -hl),
			Vec3(-hw, -hy, -hl),
		};
		for (int i = 0; i < 4; i++)
		{
			WheelSettingsTV *w = new WheelSettingsTV();
			w->mPosition = pos[i];
			w->mRadius = wr;
			w->mWidth = ww;
			w->mSuspensionMinLength = 0.18f;
			w->mSuspensionMaxLength = 0.42f;
			w->mLongitudinalFriction = 4.0f;
			w->mLateralFriction = 2.0f;
			vehicle.mWheels.push_back(w);
		}
		TrackedVehicleControllerSettings *controller = new TrackedVehicleControllerSettings();
		controller->mTracks[(int)ETrackSide::Left].mDrivenWheel = 2;
		controller->mTracks[(int)ETrackSide::Left].mWheels = {0, 2};
		controller->mTracks[(int)ETrackSide::Right].mDrivenWheel = 3;
		controller->mTracks[(int)ETrackSide::Right].mWheels = {1, 3};
		controller->mEngine.mMaxTorque = 1200.0f;
		vehicle.mController = controller;
	}
	else
	{
		const Vec3 pos[4] = {
			Vec3(hw, -hy, hl),
			Vec3(-hw, -hy, hl),
			Vec3(hw, -hy, -hl),
			Vec3(-hw, -hy, -hl),
		};
		for (int i = 0; i < 4; i++)
		{
			WheelSettingsWV *w = new WheelSettingsWV();
			w->mPosition = pos[i];
			w->mRadius = wr;
			w->mWidth = ww;
			w->mSuspensionMinLength = 0.2f;
			w->mSuspensionMaxLength = 0.5f;
			w->mMaxSteerAngle = (i < 2) ? DegreesToRadians(32.0f) : 0.0f;
			w->mMaxHandBrakeTorque = (i < 2) ? 0.0f : 2500.0f;
			vehicle.mWheels.push_back(w);
		}
		WheeledVehicleControllerSettings *controller = new WheeledVehicleControllerSettings();
		controller->mDifferentials.resize(1);
		controller->mDifferentials[0].mLeftWheel = 2;
		controller->mDifferentials[0].mRightWheel = 3;
		controller->mEngine.mMaxTorque = 1600.0f;
		controller->mEngine.mMinRPM = 1000.0f;
		controller->mEngine.mMaxRPM = 6000.0f;
		vehicle.mController = controller;
	}

	VehicleConstraint *constraint = new VehicleConstraint(body, vehicle);
	constraint->SetVehicleCollisionTester(new VehicleCollisionTesterCastCylinder(layer));
	ps->AddConstraint(constraint);
	ps->AddStepListener(constraint);
	return static_cast<JoltConstraint>(constraint);
}

void JoltSetVehicleInput(JoltConstraint constraint, int kind, float steer, float throttle, float brake)
{
	VehicleConstraint *vc = static_cast<VehicleConstraint *>(constraint);
	if (vc == nullptr)
	{
		return;
	}
	steer = clampf(steer, -1.0f, 1.0f);
	throttle = clampf(throttle, -1.0f, 1.0f);
	brake = clampf(brake, 0.0f, 1.0f);
	if (kind == JoltVehicleTank)
	{
		TrackedVehicleController *tank = static_cast<TrackedVehicleController *>(vc->GetController());
		if (tank == nullptr)
		{
			return;
		}
		float left = nonzeroRatio(1.0f - steer);
		float right = nonzeroRatio(1.0f + steer);
		tank->SetDriverInput(throttle, left, right, brake);
		return;
	}
	WheeledVehicleController *wheels = static_cast<WheeledVehicleController *>(vc->GetController());
	if (wheels == nullptr)
	{
		return;
	}
	wheels->SetDriverInput(throttle, steer, brake, 0.0f);
}
