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
	int collisionSteps = 1;
	if (deltaTime > 1.0f / 45.0f)
	{
		collisionSteps = 2;
	}
	ps->Update(deltaTime, collisionSteps, static_cast<TempAllocator *>(allocator), gJobSystem.get());
}
