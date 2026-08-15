# Physics (3D)

Jolt on Windows / Linux amd64+arm64 / macOS ARM. `PhysicsBackend$()` / `GetPhysicsBackend$()` is `"jolt"` or `"fallback"` (`-tags nojolt` or an unsupported OS/arch).

Call `UpdateWorld` (or `Flip`, which also steps) after you move or shove bodies. Handles are the same integers as `CreateCube` / `CreateCapsule` after you attach a body.

Quit loops: `While Not KeyDown(1)` works after the first-frame Escape fix. Still put `Flip` in the loop. Demos use `While 1` + `Flip` + `If frames > 8 And KeyHit(KEY_ESCAPE) Then End` so a phantom Esc cannot close the window on frame 0.

## Bodies

| Command | Meaning |
| --- | --- |
| `CreateBodyBox(e, hx, hy, hz [, mass])` | Box. `mass` 0 = static. Same handle as the mesh. |
| `CreateBodySphere(e, r [, mass])` | Sphere |
| `CreateBodyCapsule(e, halfH, r [, mass])` | Capsule. Alias `CreateRigidBodyCapsule` / `BodyCapsule` |
| `CreateBody(e [, shape, dyn])` | `shape` 2 = box, else sphere |
| `ActivateBody e` | Wake (`BodyWake` / `WakeBody`) |
| `BodySleep e` / `SleepBody e` | Deactivate |
| `SetBodyMass e, n` | Mass |
| `SetBodyVelocity e, x, y, z` / `SetVelocity` | Linear velocity |
| `GetBodyVelocityX/Y/Z(e)` | Components |
| `SetBodyAngularVelocity e, x, y, z` | Spin |
| `GetBodyAngularVelocityX/Y/Z(e)` | |
| `SetBodyRotation e, pitch, yaw, roll` | Degrees; writes Jolt + the mesh |
| `GetBodyPitch(e)` / `GetBodyYaw` / `GetBodyRoll` | After `UpdateWorld`, from the synced pose |
| `SetGravity x, y, z` / `GetGravityX/Y/Z()` | World gravity (default 0, −9.81, 0) |
| `SetGravityScale e, n` | 0 = no gravity (spaceship). Jolt `SetGravityFactor` |
| `SetRestitution e, n` / `SetFriction e, n` / `SetLinearDamping e, n` | Material / drag |
| `SetBodyCCD e, on` / `SetCCD e, on` | Jolt `MotionQuality::LinearCast` (fast projectiles) |
| `Raycast(x,y,z, dx,dy,dz)` | Hit entity (0 = none) + `PickedX/Y/Z` |

`CreateRigidBodyBox` is an alias of `CreateBodyBox`.

```basic
Graphics3D(960, 540, 0, 2)
cam = CreateCamera()
SetPosition(cam, 0, 6, -14)
CreateLight()

ground = CreateCube()
SetScale(ground, 10, 0.2, 10)
SetPosition(ground, 0, 0, 8)
CreateBodyBox(ground, 10, 0.2, 10, 0)

ball = CreateSphere()
SetPosition(ball, 0, 6, 8)
CreateBodySphere(ball, 1, 1)
SetBodyCCD(ball, True)

frames = 0
While 1
    frames = frames + 1
    If KeyHit(KEY_SPACE) Then ApplyImpulse(ball, 0, 8, 0)
    UpdateWorld
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
```

See `examples/jolt_drop.bb`, `examples/physics_body.bb`.

## Forces

World-space unless the name says Local. Jolt uses native `AddImpulse` / `AddForce` / `AddTorque`. Fallback integrates the same numbers.

| Command | Meaning |
| --- | --- |
| `ApplyImpulse e, x, y, z` | Instant Δv at center of mass |
| `ApplyForce e, x, y, z` | Force this step (center) |
| `ApplyTorque e, x, y, z` | Angular shove |
| `ApplyForceAtPosition e, fx,fy,fz, px,py,pz` | Force at a world point (boats, levers) |
| `ApplyLocalImpulse e, lx, ly, lz` | Impulse in the body’s axes (+Z is forward) |
| `ApplyBuoyancy e [, waterY, scale]` | Up-force while `y` is under `waterY` (or `WaterHeight`) |
| `SetGravityScale e, n` | 0 = no gravity |
| `SetRestitution e, n` | Bounce |
| `SetLinearDamping e, n` | Linear drag |
| `SetFriction e, n` | Friction |
| `WaterHeight(x, z)` | CPU Gerstner (boats / `ApplyBuoyancy`) |

```basic
ApplyImpulse(crate, 0, 12, 4)
ApplyTorque(crate, 0, 400, 0)
ApplyForceAtPosition(boat, 0, 800, 0, px, py, pz)
ApplyLocalImpulse(ship, 0, 0, 12)
SetGravityScale(ship, 0)
SetRestitution(ball, 0.4)
SetLinearDamping(boat, 1.8)
SetFriction(hover, 0.05)
ApplyBuoyancy(crate)
```

## Joints

`a` and `b` are **body handles** (`CreateBodyBox` / `CreateBodySphere` / …). **`0` is world-fixed** — a door hinged to the world: `CreateHingeJoint(0, doorBody, x, y, z, 0, 1, 0)`.

There is **no** `CreateDistanceJoint` command. A springy distance constraint is `CreateSpringJoint` (Jolt distance + frequency/damping).

| Command | Meaning |
| --- | --- |
| `CreateHingeJoint(a, b, x,y,z, ax,ay,az)` | Rotate around axis through the pivot. Aliases: `CreateHinge`, `CreateHinge3D` |
| `CreatePointJoint(a, b, x,y,z)` | Ball socket (shared point). Alias: `CreateBallSocketJoint` |
| `CreateSliderJoint(a, b, x,y,z, ax,ay,az)` | Slide along axis |
| `CreateSpringJoint(a, b, x,y,z, rest, stiff, damp)` | Distance spring. Defaults rest=1, stiff=8, damp=1. `rest <= 0` = free length |
| `CreateJoint kind, a, b, x,y,z [, extra…]` | One call: hinge / point / slider / spring |
| `FreeJoint id` | Remove the constraint |

`CreateHinge` / `CreateHingeJoint` with fewer than two args returns **0**.

### `CreateJoint` kinds

| Constant | Value | Extra args after `x,y,z` |
| --- | --- | --- |
| `JOINT_HINGE` | 1 | `ax, ay, az` (default 0, 1, 0) |
| `JOINT_POINT` | 2 | (none) |
| `JOINT_SLIDER` | 3 | `ax, ay, az` (default 1, 0, 0) |
| `JOINT_SPRING` | 4 | `rest, stiff, damp` |

```basic
; Hinge door — Space shoves it. Esc after Flip (frames>8).
Graphics3D(960, 600, 0, 2)
cam = CreateCamera()
SetPosition(cam, 0, 6, -14)
CreateLight()

ground = CreateCube()
SetScale(ground, 10, 0.2, 10)
SetPosition(ground, 0, 0, 8)
CreateBodyBox(ground, 10, 0.2, 10, 0)

wall = CreateCube()
SetScale(wall, 0.2, 2, 2)
SetPosition(wall, -2, 2, 8)
wallBody = CreateBodyBox(wall, 0.2, 2, 2, 0)

door = CreateCube()
SetScale(door, 0.08, 1.6, 0.9)
SetPosition(door, -0.9, 1.7, 8)
doorBody = CreateBodyBox(door, 0.08, 1.6, 0.9, 2)

doorHinge = CreateHingeJoint(wallBody, doorBody, EntityX(wall), EntityY(wall), EntityZ(wall), 0, 1, 0)
; same as: CreateJoint(JOINT_HINGE, wallBody, doorBody, EntityX(wall), EntityY(wall), EntityZ(wall), 0, 1, 0)

frames = 0
While 1
    frames = frames + 1
    If KeyHit(KEY_SPACE) Then ApplyImpulse(doorBody, 4, 0, 0)
    UpdateWorld()
    RenderWorld()
    Flip()
    If frames > 8 And KeyHit(KEY_ESCAPE) Then End
Wend
```

Point + slider + spring:

```basic
ball = CreatePointJoint(ceiling, lamp, EntityX(lamp), EntityY(ceiling), EntityZ(lamp))
rail = CreateSliderJoint(track, cart, EntityX(cart), EntityY(cart), EntityZ(cart), 1, 0, 0)
spring = CreateSpringJoint(a, b, midX, midY, midZ, 2, 12, 1)
FreeJoint(spring)
```

Demo: `examples/physics_joints.bb`.

## CharacterVirtual

`CreateCharacterController` builds a Jolt **CharacterVirtual** (capsule) on the entity. `GetPhysicsCharacter$()` is `"jolt"` when that path is live, `"kinematic"` on fallback.

`CreateCharacter(e [, halfH, r])` is the older kinematic capsule helper (still registered). Prefer the controller for walk/slide.

| Command | Meaning |
| --- | --- |
| `CreateCharacterController(e [, height, radius, maxSlopeDeg, maxStrength])` | Defaults 1.8, 0.4, 50, 100 |
| `MoveCharacter e, vx, vz` | Walk XZ; keeps current Y velocity (gravity / jump) |
| `MoveCharacter e, vx, vy, vz` | Set all three |
| `SetCharacterShape e, "capsule"\|"box", height, radius` | Swap the virtual shape |
| `GetCharacterGroundState(e)` / `CharacterGroundState(e)` | 0–3 (below) |
| `CharacterOnGround(e)` | 1 if supported |
| `GetCharacterGroundNormal(e [, axis])` | `axis` 0=X, 1=Y (default), 2=Z |
| `CharacterGroundNormalX/Y/Z(e)` | Same, one component |
| `GetCharacterContact(e)` / `CharacterContact(e)` | Other entity touching the capsule (0 = none) |
| `SetCharacterVelocity e, x, y, z` | Raw velocity |

`ExtendedUpdate` / `CharacterExtendedUpdate` exist as commands but do **not** step the sim — `UpdateWorld` / `Flip` does.

### Ground state (0–3)

Same numbers as Jolt `CharacterVirtual::EGroundState`:

| Value | Name | Meaning |
| --- | --- | --- |
| 0 | On ground | Supported; walk |
| 1 | Steep | Touching a slope steeper than `maxSlopeDeg` |
| 2 | Not supported | Touching something that will not hold you |
| 3 | In air | No ground |

```basic
; examples/character_virt.bb — WASD, Esc after a few frames.
Graphics3D(960, 540, 0, 2)
cam = CreateCamera()
SetPosition(cam, 0, 6, -14)
CreateLight()

ground = CreateCube()
SetScale(ground, 12, 0.2, 12)
SetPosition(ground, 0, 0, 8)
CreateRigidBodyBox(ground, 12, 0.2, 12, 0)

hero = CreateCapsule(0.4, 0.9, 8)
SetPosition(hero, 0, 2.2, 8)
CreateCharacterController(hero, 1.8, 0.4, 50, 100)
SetCharacterShape(hero, "capsule", 1.8, 0.4)

frames = 0
While 1
    frames = frames + 1
    vx# = 0
    vz# = 0
    If KeyDown(KEY_A) Then vx = -5
    If KeyDown(KEY_D) Then vx = 5
    If KeyDown(KEY_W) Then vz = 5
    If KeyDown(KEY_S) Then vz = -5
    MoveCharacter(hero, vx, vz)
    UpdateWorld
    RenderWorld
    Text(12, 12, "ground=" + GetCharacterGroundState(hero) + " hit=" + GetCharacterContact(hero))
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
```

## Contacts

Jolt’s `ContactListener` runs on **physics threads**. The wrapper **does not `//export` into Go from those threads**. C++ pushes added/persisted/removed events onto a mutex queue. `UpdateWorld` calls `PollContacts` and fills each entity’s collide list.

| Command | Meaning |
| --- | --- |
| `EntityCollided(e)` | First other entity this step (0 = none) |
| `EntityCollided(e, other)` | `other` if that pair hit |
| `EntityCollided(e, type)` | First hit whose `EntityType` is `type` |
| `CountCollisions(e)` | How many others this step |
| `CollisionEntity(e)` | Last hit id |
| `CollisionX/Y/Z(e)` | Contact point |
| `EntityType e, n` / `GetEntityType(e)` | Filter id for `EntityCollided` |
| `ResetEntity e` | Clear this frame’s list |
| `Collisions srcType, destType` | Extra radius overlap rule (classic) |

Removed contacts are ignored when filling the list (you only see current/persisted hits).

```basic
SetEntityType(hazard, 2)
SetEntityType(ball, 1)
; …
UpdateWorld()
other = EntityCollided(ball, hazard)
If other Then
    Text(16, 36, "hit x=" + Int(CollisionX(ball)))
EndIf
```

Demo: `examples/physics_contacts.bb`.

## First Flip / Escape

GLFW often injects Escape (or a close flag) while the window is created. The runtime **ignores Escape and `WindowShouldClose` until the first `Flip`**. After that, `While Not KeyDown(1)` is a normal quit loop.

Still **call `Flip` every frame**. For demos that must not die on a leftover key, wait a few presented frames:

```basic
frames = 0
While 1
    frames = frames + 1
    UpdateWorld
    RenderWorld
    Flip
    If frames > 8 And KeyHit(KEY_ESCAPE) Then End
Wend
```

Or:

```basic
While Not KeyDown(1)
    UpdateWorld
    RenderWorld
    Flip
Wend
```

## Vehicles

Landed. Full names and demos: [VEHICLES.md](VEHICLES.md).

**`CreatePlane` is still the ground mesh.** Aircraft is `CreatePlaneController` + `UpdatePlane`. No alias.

- **Jolt vehicle:** `CreateCarController` / `CreateVehicle` + `UpdateCar` / `SetVehicleInput(steer, throttle, brake)`. `CreateMotorcycleController`. `CreateTankController` / `CreateTrackedController`.
- **Force / torque:** `CreatePlaneController` `UpdatePlane(th, pitch, roll, yaw)`. `CreateJetController`. `CreateSpaceshipController`. `CreateBoatController` `UpdateBoat`. `CreateHelicopterController` `UpdateHelicopter(collective, cyclicP, cyclicR, yaw)`. `CreateHovercraftController`. `CreateSubmarineController` `UpdateSubmarine(throttle, steer, dive)`. `CreateDroneController`.

Demos: `examples/car.bb` `plane.bb` `jet.bb` `spaceship.bb` `boat.bb` `motorcycle.bb` `helicopter.bb` `hovercraft.bb` `submarine.bb` `tank.bb` `drone.bb` `vehicles_more.bb`.

## 2D (Chipmunk)

`Physics2D`, `CreateCircle2D` / `CreateBox2D`, `ApplyImpulse2D`, `CreatePin2D` / `CreateSpring2D` / `CreateSlide2D`. Not Box2D. See [COMMANDS.md](COMMANDS.md).
