# Vehicles

Controllers are **landed**. Each `CreateXxxController(entity [, hx, hy, hz])` attaches a dynamic box body (if the entity has none) and stores a controller. Call the matching `UpdateXxx` **every frame**, then `UpdateWorld` / `Flip`.

Optional `hx, hy, hz` are half-extents. If omitted, the mesh box / default hull for that kind is used.

Quit: `While 1` + `Flip` + `If frames > 8` then `KeyHit(KEY_ESCAPE)`. See [PHYSICS.md](PHYSICS.md).

## `CreatePlane` is the mesh — not the aircraft

| Command | What it is |
| --- | --- |
| `CreatePlane([parent])` or `CreatePlane(w, h [, parent])` | **Ground / quad mesh** (same family as `CreateCube`). Always has been. |
| `CreatePlaneController(entity)` | **Aircraft** controller (aero forces). |
| `UpdatePlane(entity, throttle, pitch, roll, yaw)` | Drive that aircraft. |

There is **no** alias from `CreatePlane` → `CreatePlaneController`. `CreateJet` / `CreateBoat` / `CreateCar` *are* aliases of their controllers. `CreatePlane` is **not**.

```basic
ground = CreatePlane()                 ; flat mesh
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

craft = CreateCube()
SetPosition(craft, 0, 8, 20)
CreatePlaneController(craft)           ; NOT CreatePlane(craft)
UpdatePlane(craft, 0.55, 0, 0, 0)
```

Demo of the aircraft: `examples/plane.bb`.

## Two physics paths

**Native Jolt VehicleConstraint** (Windows Jolt; otherwise the same `Update*` still runs a force fallback):

| Create | Update / input | Demo |
| --- | --- | --- |
| `CreateCarController(e)` / `CreateVehicle(e)` | `UpdateCar e, steer, throttle, brake` — also `UpdateVehicle`, `SetVehicleInput e, steer, throttle, brake` (`SetCarInput`) | `examples/car.bb` |
| `CreateMotorcycleController(e)` (`CreateBike`) | `UpdateMotorcycle e, steer, throttle, brake` | `examples/motorcycle.bb` |
| `CreateTankController(e)` / `CreateTrackedController(e)` (`CreateTank`) | `UpdateTank` / `UpdateTracked e, steer, throttle, brake` | `examples/tank.bb` |

`SetVehicleInput` writes Jolt wheel/track input when a native vehicle exists. If native is 0, it applies the wheeled force fallback for car / moto / tank.

**Force / torque / buoyancy** (no Jolt vehicle constraint):

| Create | Update | Demo |
| --- | --- | --- |
| `CreatePlaneController(e)` | `UpdatePlane e, throttle, pitch, roll, yaw` | `examples/plane.bb` |
| `CreateJetController(e)` (`CreateJet`) | `UpdateJet e, throttle, pitch, roll, yaw` | `examples/jet.bb` |
| `CreateSpaceshipController(e)` (`CreateSpaceship`) | `UpdateSpaceship e, throttle, pitch, roll, yaw` | `examples/spaceship.bb` |
| `CreateBoatController(e)` (`CreateBoat`) | `UpdateBoat e, throttle, steer` | `examples/boat.bb` |
| `CreateHelicopterController(e)` (`CreateHeli`) | `UpdateHelicopter e, collective, cyclicP, cyclicR, yaw` | `examples/helicopter.bb` |
| `CreateHovercraftController(e)` (`CreateHover`) | `UpdateHovercraft e, throttle, steer` | `examples/hovercraft.bb` |
| `CreateSubmarineController(e)` (`CreateSub`) | `UpdateSubmarine e, throttle, steer, dive` | `examples/submarine.bb` |
| `CreateDroneController(e)` (`CreateDrone`) | `UpdateDrone e, throttle, pitch, roll, yaw` | `examples/drone.bb` |

Also registered (same pattern): `CreateGliderController` / `UpdateGlider` (aero, no thrust), `CreateSkiController` / `UpdateSki` (low friction). All six extra kinds together: `examples/vehicles_more.bb`.

## What the sim actually does

| Kind | Internals |
| --- | --- |
| Car / moto / tank | Jolt wheeled / motorcycle / tracked when `Create*Vehicle` succeeds; else `ApplyForce` along +Z, yaw `ApplyTorque`, brake drag |
| Plane / jet / glider | Aero: lift + drag + thrust `ApplyForce`, stick `ApplyTorque`. Plane stall ~12, jet ~28 |
| Spaceship | `SetGravityScale 0`, `ApplyLocalImpulse` +Z, `ApplyTorque` |
| Boat | Gerstner `WaterHeight` at four hull corners → `ApplyForceAtPosition` lift + forward thrust |
| Hovercraft | Constant up-force, low `SetFriction`, XZ thrust |
| Submarine | Buoyancy vs `WaterHeight` + dive force |
| Helicopter | Hover = mass × \|g\| + **collective**; cyclicP / cyclicR / yaw are torques |
| Drone | Same hover idea, small mass, angular damping |

## Forces you can call yourself

These are real commands (vehicles use them internally):

| Command | Meaning |
| --- | --- |
| `ApplyTorque e, x, y, z` | Angular shove |
| `ApplyForceAtPosition e, fx,fy,fz, px,py,pz` | Force at a world point |
| `ApplyLocalImpulse e, lx, ly, lz` | Impulse in body axes (+Z forward) |
| `SetGravityScale e, n` | 0 = no gravity |
| `SetRestitution e, n` | Bounce |
| `SetLinearDamping e, n` | Linear drag |
| `SetFriction e, n` | Friction |
| `ApplyBuoyancy e [, waterY, scale]` | Up-force while under `waterY` or `WaterHeight(x, z)` |
| `WaterHeight(x, z)` | CPU Gerstner height (matches the water mesh) |

```basic
SetGravityScale(ship, 0)
SetLinearDamping(boat, 1.8)
SetFriction(hover, 0.05)
ApplyLocalImpulse(ship, 0, 0, 12)
ApplyTorque(ship, 0, 800, 0)
y# = WaterHeight(EntityX(boat), EntityZ(boat))
ApplyBuoyancy(crate, y#, 1)
```

## Car (native wheels)

```basic
Graphics3D(960, 540, 0, 2)
cam = CreateCamera()
SetPosition(cam, 0, 8, -16)
CreateLight()

ground = CreateCube()
SetScale(ground, 40, 0.25, 40)
SetPosition(ground, 0, 0, 10)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

car = CreateCube()
SetScale(car, 1.1, 0.35, 2.0)
SetPosition(car, 0, 1.2, 10)
CreateCarController(car)
; same as CreateVehicle(car)

frames = 0
While 1
    frames = frames + 1
    steer# = 0
    throttle# = 0
    brake# = 0
    If KeyDown(KEY_A) Then steer = -1
    If KeyDown(KEY_D) Then steer = 1
    If KeyDown(KEY_W) Then throttle = 1
    If KeyDown(KEY_S) Then throttle = -0.4
    If KeyDown(KEY_SPACE) Then brake = 1
    UpdateCar(car, steer, throttle, brake)
    ; or: SetVehicleInput(car, steer, throttle, brake)
    UpdateWorld
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
```

## Helicopter (collective + cyclic)

```basic
CreateHelicopterController(heli)
; …
UpdateHelicopter(heli, collective, cyclicP, cyclicR, yaw)
```

`collective` is extra lift on top of hover-against-gravity. `cyclicP` / `cyclicR` are pitch / roll stick, not degrees.

## Boat / sub (Gerstner)

Need `CreateWater` so `WaterHeight` matches the waves.

```basic
water = CreateWater(220, 220, 48)
SetWaterLevel(0)
CreateBoatController(boat)
UpdateBoat(boat, throttle, steer)

CreateSubmarineController(sub)
UpdateSubmarine(sub, throttle, steer, dive)
```

## Honest limits

- No wheel meshes or gear UI — a cube + a constraint / forces.
- Native Jolt vehicles are wired on **Windows**. Other Jolt platforms may get the force fallback.
- Boat / sub use **CPU** `WaterHeight`, not a separate fluid sim.
- `CreateBuoy(ent)` (splash / wake) is a different water helper — [WATER.md](WATER.md).

## Demos

`examples/car.bb` `plane.bb` `jet.bb` `spaceship.bb` `boat.bb` `motorcycle.bb` `helicopter.bb` `hovercraft.bb` `submarine.bb` `tank.bb` `drone.bb` `vehicles_more.bb`
