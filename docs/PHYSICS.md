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
| `CreateBodyCylinder(e, halfH, r [, motion, mass])` | Y-aligned cylinder collider (Jolt `CylinderShape`). `halfH` is half the **shaft** height, not `CreateCylinder`’s full visual height. Trailing args match `CreateBodyBox`: `0` static, `1` kinematic, else dynamic (that number is mass unless a fifth arg is mass). Alias `CreateRigidBodyCylinder`. Linux Jolt uses a capsule of the same size. |
| `CreateBodyConvex(e [, motion, mass])` | Convex hull from the mesh’s world triangles, stored in local space at the entity pose. At most **96** unique points (1/50 m grid). Needs a mesh (`CreateCone`, `LoadMesh`, …). Same motion/mass trailing args as the box. Aliases `CreateConvexHull` / `CreateConvexBody` / `CreateRigidBodyConvex`. If hull cook fails, the backend falls back to an AABB box. |
| `CreateBodyCompound(e [, motion, mass])` | One actor from the parent mesh plus **child** meshes/colliders (local offsets). Child rigid bodies are removed. Alias `CreateCompoundBody`. Linux Jolt / fallback use a combined AABB box. |
| `CreateHitbox(e, hx,hy,hz [, motion])` | Query-only sensor (same as `CreateSensor`) tagged for green debug draw. Alias `CreateQueryBox`. |
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
| `ShapeCast(hx,hy,hz, x,y,z, dx,dy,dz)` | Sweep a box; hit entity + `PickedX/Y/Z` |
| `OverlapSphere(x,y,z, r)` / `OverlapPoint(x,y,z)` | First overlapping body (sensors included) |
| `Explode x,y,z, r, imp` | Overlap all dynamics in the sphere and apply falloff impulse. Alias `AreaDamage`. |
| `CreateBodyMesh(e)` | Static mesh collider from the entity’s triangles (world space). Enables Jolt enhanced internal-edge removal. |
| `CreateBodyHeightField(e [, n])` | Static height field from the current terrain (`n` samples on a side, power of two, default 64). |
| `CreateSensor(e, hx,hy,hz [, motion])` | Trigger volume (no contact force). Default motion = kinematic. |
| `EnablePhysicsDebug [on]` | Wire AABB overlay: cyan colliders, green hitboxes. |
| `SetCollisionLayer e, layer` | Layer 0–31 (default 0). |
| `SetLayerCollides a, b, on` | Whether two layers generate contacts (default all collide). Pairwise `DisableBodyCollision` still exists. |
| `SetBodySensor e, on` | Toggle sensor on an existing body |
| `OptimizePhysics()` | Rebuild Jolt’s broadphase after spawning many bodies (`OptimizeBroadPhase` / `RebuildBroadPhase`) |

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

See `examples/jolt_drop.bb`, `examples/physics_body.bb`, `examples/physics_pile.bb`, `examples/grab_beam.bb`.

## Forces

World-space unless the name says Local. Jolt uses native `AddImpulse` / `AddForce` / `AddTorque`. Fallback integrates the same numbers.

| Command | Meaning |
| --- | --- |
| `ApplyImpulse e, x, y, z` | Instant Δv at center of mass |
| `ApplyForce e, x, y, z` | Force this step (center) |
| `ApplyTorque e, x, y, z` | Angular shove |
| `ApplyForceAtPosition e, fx,fy,fz, px,py,pz` | Force at a world point (boats, levers) |
| `ApplyLocalImpulse e, lx, ly, lz` | Impulse in the body’s axes (+Z is forward) |
| `ApplyBuoyancy e [, waterY, scale]` | Jolt `ApplyBuoyancyImpulse` against a water plane. Omit `waterY` to use `WaterHeight(x,z)` (slope from nearby samples). `scale` 1 ≈ body density matches water. Fluid velocity comes from `SetWaterFlow`. |
| `SetWaterFlow vx, vy, vz` | Current in `ApplyBuoyancy` / auto water volume. |
| `SetBuoyancyFactor e, n` | Per-body float/sink. `0` = auto 1.1 when submerged; negative = off. |
| `CreateCloth w, h [, nx, ny, pin]` | Jolt XPBD sheet. Pin bits: 1 top, 2 bottom, 4 left, 8 right. `SetClothWind e, n`. |
| `OffsetCenterOfMass e, x, y, z` | Wrap the collider so mass sits at an offset (vehicles drop COM by default). |
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
SetWaterFlow(1.2, 0, 0)
SetBuoyancyFactor(crate, 1.15)
flag = CreateCloth(2, 1.5, 10, 8)

OffsetCenterOfMass(car, 0, -0.2, 0)
CreateBodyMesh(level)
CreateSensor(trigger, 2, 1, 2)
hit = ShapeCast(0.4, 0.4, 0.4, x, y, z, 0, -20, 0)
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
| `CreateFixedJoint(a, b, x,y,z)` | Weld two bodies. Aliases `CreateWeldJoint` / `CreateFixedConstraint`. `JOINT_FIXED` = 5. |
| `CreateConeJoint(a, b, x,y,z, ax,ay,az [, halfConeDeg])` | Ball socket + swing cone. Default half-angle 45°. `JOINT_CONE` = 6. |
| `CreateSwingTwistJoint(a, b, x,y,z, ax,ay,az [, swingDeg, twistDeg])` | Ragdoll-ish cone + twist limits. Defaults 45° / 30°. `JOINT_SWINGTWIST` = 7. |
| `Grab holder, target [, freq, damp [, x,y,z]]` | 6DOF spring grab. Extra `x,y,z` is a world anchor (not COM). Soft **translation** (default freq 8 Hz, damp 1), **rotation free**. Returns the target handle. Both entities should already have bodies. Aliases `GrabEntity`. |
| `GrabPick holder, maxDist [, freq, damp]` | Ray from `holder` along local +Z (`pitch`/`yaw`). Grabs the first hit at the **hit point** (not COM). Default `maxDist` 8. Sets `PickedEntity` / `PickedX/Y/Z`. |
| `DropGrab holder` | Remove that holder’s grab. Alias `ReleaseGrab`. |
| `Throw holder, speed` | `DropGrab` then shove the target along the holder’s +Z (`SetVelocity` + `ApplyImpulse`). Default speed 12. Alias `ThrowEntity`. Returns the thrown entity (0 if none). |
| `GrabbedEntity(holder)` | Current grabbed body (0 if none). |
| `CreateJoint kind, a, b, x,y,z [, extra…]` | One call: hinge / point / slider / spring / fixed / cone / swing-twist |
| `FreeJoint id` | Remove the constraint |

`CreateHinge` / `CreateHingeJoint` with fewer than two args returns **0**.

**Grab platforms:** Windows Jolt is a real `SixDOFConstraint`. Linux/macOS Jolt extras stub the joint to 0; `Grab` then **parents** the target to the holder (kinematic) until `DropGrab`. Fallback (`-tags nojolt`) uses a stiff spring joint. `claw.bb` is still the kinematic-parent claw; `Grab` is the spring/6DOF path.

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

## Physical ropes

Ropes use a stable position-based Verlet chain for sag, bending, and obstacle contact, with forces applied through Jolt at both local attachment points. This avoids unstable light-link/heavy-vehicle mass ratios while still transferring towing force and torque to the attached bodies.

```basic
rope = CreateRopeAnchored(boat, skier, 0,0.2,-2.8, 0,0.92,0.58, 13.5,14,0.03)
SetRopeColor(rope, 226,204,132)
SetRopeMass(rope, 0.09)
SetRopeDamping(rope, 0.24)
SetRopeStrength(rope, 4000, 500, 3500)
Text(12, 12, "Tension " + Int(RopeTension(rope) * 100) + "%")
```

Use `CreateRope(a,b,length,segments,radius)` for center-to-center attachment. `CreateRopeAnchored` accepts local offsets for each body; offsets attached to body `0` are world positions. `SetRopeStrength` tunes the taut-line stiffness, velocity damping, and force cap for unusually light or heavy endpoints. `ResetRope` rebuilds its sag after teleporting an endpoint, and `FreeRope` safely removes its bodies and constraints. See `examples/rope.bb` and `examples/waterski.bb`.

`CreateCloth` is a pinned rectangle (default top edge) using Jolt soft bodies on Windows. See `examples/cloth.bb`.

## Grab, projectiles, beams

Demos: `examples/grab_beam.bb` (Space grab, T throw, G drop, laser via `PlaceAtRay`).

| Command | Meaning |
| --- | --- |
| `CreateProjectile e, speed [, gravity, life, radius, bounce, impulse, ignore]` | Flies along the entity’s local +Z each `UpdateWorld`. Defaults: speed 20, gravity 0, life 4 s, radius 0.1 (stored), bounce 0, impulse 1, ignore 0. If `e` has a body: CCD on, gravity scale 0, velocity set. A physics ray along this step’s motion hits another body → `ApplyImpulse` on the victim (scaled by `impulse`), sets `Picked*`, then stops (or reverses velocity × `bounce` if bounce > 0). `ignore` skips that entity (the gun). When `life` expires the flyer is forgotten; the mesh stays. |
| `CreateBeam a, b [, width]` | Visual only: a cube stretched between two entities each `UpdateWorld`. Default width 0.08. Returns the beam mesh handle. |
| `PlaceAtRay src, dest [, maxDist]` | Move `dest` to the first physics hit along `src`’s +Z, or to `maxDist` (default 40) if nothing is hit. Sets `Picked*`. Alias `RayPlace`. Pair with `CreateBeam(src, dest)` for a laser. |
| `AttachToBone child, mesh, bone$` | Parent `child` to a named node under `mesh` (glTF joint name, case-insensitive). If no node matches, uses a child entity whose `NameEntity` equals `bone$`. If still none, parents to the mesh root. `GetParent(child)` is the mesh entity. Alias `AttachBone`. `NameEntity` also writes the G3N node name, which helps lookup. **Not** skeletal IK; clip play still drives the bones. |

`CreateProjectile` / `PlaceAtRay` / `GrabPick` share `PickedEntity` / `PickedX` / `PickedY` / `PickedZ` with `Raycast`.

## CharacterVirtual

`CreateCharacterController` builds a Jolt **CharacterVirtual** (capsule) on the entity, with a slightly smaller **inner kinematic rigid body** so rays, CCD, and contact listeners can hit the player. `GetPhysicsCharacter$()` is `"jolt"` when that path is live, `"kinematic"` on fallback.

`UpdateWorld` / `Flip` already run Jolt `CharacterVirtual::ExtendedUpdate` (stick-to-floor + walk-stairs). `ExtendedUpdate` / `CharacterExtendedUpdate` are documented no-ops you can call for other engines; they do not double-step.

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

`ExtendedUpdate` / `CharacterExtendedUpdate` exist as commands but do **not** step the sim — `UpdateWorld` / `Flip` already calls `ExtendedUpdate`.

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
