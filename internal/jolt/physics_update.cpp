/*
 * PhysicsSystem::Update using a Go-owned TempAllocator (malloc).
 *
 * Prebuilt jolt_wrapper's 2-arg Update uses TempAllocatorImpl(10MB) and
 * abort()s on the 17694720-byte contact buffer. This 3-arg symbol is
 * provided by our object file so CGO never hits that path.
 */

#include "wrapper/allocator.h"
#include "wrapper/core.h"
#include "wrapper/physics.h"

#include <Jolt/Jolt.h>
#include <Jolt/Core/JobSystemThreadPool.h>
#include <Jolt/Core/TempAllocator.h>
#include <Jolt/Physics/PhysicsSystem.h>
#include <Jolt/Physics/PhysicsSettings.h>

#include <cstdarg>
#include <cstdio>

using namespace JPH;

namespace {

static void mbJoltTrace(const char *fmt, ...)
{
	va_list args;
	va_start(args, fmt);
	fputs("jolt: ", stderr);
	vfprintf(stderr, fmt, args);
	fputc('\n', stderr);
	va_end(args);
}

} // namespace

extern "C" void JoltInstallSafeTempAllocator()
{
	Trace = mbJoltTrace;
}

extern "C" void JoltPhysicsSystemUpdateWithAllocator(JoltPhysicsSystem system, float deltaTime, JoltTempAllocator allocator)
{
	if (system == nullptr || allocator == nullptr || gJobSystem == nullptr)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	if (ps == nullptr)
	{
		return;
	}
	if (deltaTime < 0.0f)
	{
		deltaTime = 0.0f;
	}
	if (deltaTime > 1.0f / 20.0f)
	{
		deltaTime = 1.0f / 20.0f;
	}
	// Two collision steps at 60 Hz keep stacks and character pushes from tunneling
	// through a single contact solve. Larger frames take more slices.
	int collisionSteps = 2;
	if (deltaTime > 1.0f / 50.0f)
	{
		collisionSteps = 3;
	}
	if (deltaTime > 1.0f / 30.0f)
	{
		collisionSteps = 4;
	}
	ps->Update(deltaTime, collisionSteps, static_cast<TempAllocator *>(allocator), gJobSystem.get());
}

extern "C" void JoltConfigureStablePhysics(JoltPhysicsSystem system)
{
	if (system == nullptr)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	if (ps == nullptr)
	{
		return;
	}
	PhysicsSettings settings = ps->GetPhysicsSettings();
	settings.mNumVelocitySteps = 14;
	settings.mNumPositionSteps = 3;
	settings.mBaumgarte = 0.25f;
	settings.mSpeculativeContactDistance = 0.04f;
	settings.mPenetrationSlop = 0.015f;
	settings.mMinVelocityForRestitution = 0.8f;
	ps->SetPhysicsSettings(settings);
}

extern "C" void JoltOptimizeBroadPhase(JoltPhysicsSystem system)
{
	if (system == nullptr)
	{
		return;
	}
	PhysicsSystem *ps = GetPhysicsSystem(static_cast<PhysicsSystemWrapper *>(system));
	if (ps == nullptr)
	{
		return;
	}
	ps->OptimizeBroadPhase();
}

extern "C" void JoltSetJobThreads(int n)
{
	if (n < 1)
	{
		n = 1;
	}
	if (n > 32)
	{
		n = 32;
	}
	gJobSystem = std::make_unique<JobSystemThreadPool>(cMaxPhysicsJobs, cMaxPhysicsBarriers, n);
}
