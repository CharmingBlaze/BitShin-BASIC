/*
 * Offset COM decorator and height-field shapes (not in the prebuilt C wrapper).
 */

#include "wrapper/shape.h"

#include <Jolt/Jolt.h>
#include <Jolt/Physics/Collision/Shape/OffsetCenterOfMassShape.h>
#include <Jolt/Physics/Collision/Shape/HeightFieldShape.h>
#include <Jolt/Physics/Collision/Shape/CylinderShape.h>
#include <Jolt/Physics/Collision/Shape/BoxShape.h>
#include <Jolt/Physics/Collision/Shape/SphereShape.h>
#include <Jolt/Physics/Collision/Shape/CapsuleShape.h>
#include <Jolt/Physics/Collision/Shape/StaticCompoundShape.h>
#include <Jolt/Physics/Collision/Shape/RotatedTranslatedShape.h>

using namespace JPH;

JoltShape JoltOffsetCenterOfMass(JoltShape inner, float ox, float oy, float oz)
{
	if (inner == nullptr)
	{
		return nullptr;
	}
	OffsetCenterOfMassShapeSettings settings(Vec3(ox, oy, oz), static_cast<Shape *>(inner));
	Shape::ShapeResult result = settings.Create();
	if (result.HasError())
	{
		return nullptr;
	}
	Shape *shape = result.Get().GetPtr();
	shape->AddRef();
	return shape;
}

JoltShape JoltCreateHeightField(const float *samples, int sampleCount,
							   float ox, float oy, float oz,
							   float sx, float sy, float sz)
{
	if (samples == nullptr || sampleCount < 4)
	{
		return nullptr;
	}
	HeightFieldShapeSettings settings(samples, Vec3(ox, oy, oz), Vec3(sx, sy, sz), uint32(sampleCount));
	Shape::ShapeResult result = settings.Create();
	if (result.HasError())
	{
		return nullptr;
	}
	Shape *shape = result.Get().GetPtr();
	shape->AddRef();
	return shape;
}

JoltShape JoltCreateCylinder(float halfHeight, float radius)
{
	if (halfHeight < 1.0e-4f)
	{
		halfHeight = 1.0e-4f;
	}
	if (radius < 1.0e-4f)
	{
		radius = 1.0e-4f;
	}
	CylinderShapeSettings settings(halfHeight, radius);
	Shape::ShapeResult result = settings.Create();
	if (result.HasError())
	{
		return nullptr;
	}
	Shape *shape = result.Get().GetPtr();
	shape->AddRef();
	return shape;
}

static Shape *makePartShape(int kind, float a, float b, float c)
{
	if (a < 1.0e-4f)
	{
		a = 1.0e-4f;
	}
	Shape::ShapeResult result;
	switch (kind)
	{
	case 1:
		result = SphereShapeSettings(a).Create();
		break;
	case 2:
		if (b < 1.0e-4f)
		{
			b = 1.0e-4f;
		}
		result = CapsuleShapeSettings(a, b).Create();
		break;
	case 3:
		if (b < 1.0e-4f)
		{
			b = 1.0e-4f;
		}
		result = CylinderShapeSettings(a, b).Create();
		break;
	default:
		if (b < 1.0e-4f)
		{
			b = 1.0e-4f;
		}
		if (c < 1.0e-4f)
		{
			c = 1.0e-4f;
		}
		result = BoxShapeSettings(Vec3(a, b, c)).Create();
		break;
	}
	if (result.HasError())
	{
		return nullptr;
	}
	Shape *shape = result.Get().GetPtr();
	shape->AddRef();
	return shape;
}

JoltShape JoltCreateCompound(const JoltCompoundPart *parts, int numParts)
{
	if (parts == nullptr || numParts <= 0)
	{
		return nullptr;
	}
	StaticCompoundShapeSettings settings;
	for (int i = 0; i < numParts; i++)
	{
		Shape *inner = makePartShape(parts[i].kind, parts[i].a, parts[i].b, parts[i].c);
		if (inner == nullptr)
		{
			continue;
		}
		settings.AddShape(Vec3(parts[i].ox, parts[i].oy, parts[i].oz), Quat::sIdentity(), inner);
		inner->Release();
	}
	if (settings.mSubShapes.empty())
	{
		return nullptr;
	}
	Shape::ShapeResult result = settings.Create();
	if (result.HasError())
	{
		return nullptr;
	}
	Shape *shape = result.Get().GetPtr();
	shape->AddRef();
	return shape;
}
