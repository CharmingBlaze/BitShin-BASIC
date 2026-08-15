/*
 * Jolt VehicleConstraint + wheeled / motorcycle / tracked controllers.
 */

#ifndef JOLT_WRAPPER_VEHICLE_H
#define JOLT_WRAPPER_VEHICLE_H

#include "physics.h"
#include "body.h"
#include "constraint.h"

#ifdef __cplusplus
extern "C" {
#endif

enum {
	JoltVehicleCar = 1,
	JoltVehicleMotorcycle = 2,
	JoltVehicleTank = 3
};

typedef struct JoltVehicleSettings {
	float halfWidth;
	float halfHeight;
	float halfLength;
	float wheelRadius;
	float wheelWidth;
	int objectLayer;
	int kind;
} JoltVehicleSettings;

JoltConstraint JoltCreateVehicle(JoltPhysicsSystem system, JoltBodyID bodyID, const JoltVehicleSettings *settings);
void JoltSetVehicleInput(JoltConstraint constraint, int kind, float steer, float throttle, float brake);

#ifdef __cplusplus
}
#endif

#endif /* JOLT_WRAPPER_VEHICLE_H */
