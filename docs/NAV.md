# Docs

Topic pages for script authors. Command lists stay in [COMMANDS.md](COMMANDS.md). Language: [LANGUAGE.md](LANGUAGE.md).

| Page | What it covers |
| --- | --- |
| [PHYSICS.md](PHYSICS.md) | Bodies, joints, forces, CharacterVirtual, contacts, first `Flip` / Escape |
| [VEHICLES.md](VEHICLES.md) | Landed controllers. **`CreatePlane` = mesh**; aircraft is `CreatePlaneController` |
| [NETWORK.md](NETWORK.md) | Host a room, find it on the LAN, wait, then play |
| [SHADOWS.md](SHADOWS.md) | CSM, quality 0–3 (grid PCF / PCSS / EVSM / MSM) |
| [WATER.md](WATER.md) | Gerstner + scenic water, `WaterHeight`, buoys |
| [WEATHER.md](WEATHER.md) | Rain / snow / fog / storm |
| [TERRAIN.md](TERRAIN.md) | Heightmaps, splat, chunks |
| [PBR.md](PBR.md) | Metallic-roughness |
| [GRAPHICS.md](GRAPHICS.md) / [COMPAT.md](COMPAT.md) | OpenGL 3.3 |
| [POSTFX.md](POSTFX.md) / [SHADERS.md](SHADERS.md) | Blit stack / GLSL 330 |
| [GEO.md](GEO.md) / [STREAM.md](STREAM.md) / [ECS.md](ECS.md) | Map, chunks, Flecs |
| [ASSETS.md](ASSETS.md) / [ARCHITECTURE.md](ARCHITECTURE.md) | Pack / Flip loop |
| [STATUS.md](STATUS.md) | Honest feature table |
| [ANIMATION.md](ANIMATION.md) | Clips, blends, root motion, masks, bones |
| [UI.md](UI.md) | ImGui, G3N widgets, profiler |
| [AUDIO.md](AUDIO.md) | Clips, streams, buses, echo |
| [PARTICLES.md](PARTICLES.md) | CPU and GPU emitters |
| [RELEASE.md](RELEASE.md) | Portable `moon build`, natives, 1.0 version |

Pathfinding (this file) is below.

# Pathfinding

Detour and grid A* for paths. Behavior trees (`CreateBehavior`) and the GOAP planner (`CreatePlanner`) decide what to do. Steering, vision, and `CreateAI` move an entity once the decision is made.

## 3D — Detour (`github.com/arl/go-detour` v0.1.3)

`CreateNavMesh(mesh)` bakes **walkable triangles** from that entity’s G3N mesh (world-space `ReadFaces`). Collision / loaded meshes use the same path. If the entity has no triangles, the world **AABB top** is the fallback floor. `AddNavObstacle(entity)` subtracts that entity’s mesh triangles, or its AABB box if it has none. `BakeNavMesh` writes a temporary OBJ and runs Recast `solomesh.Build()`.

| Command | Meaning |
| --- | --- |
| `CreateNavMesh(mesh)` | Nav handle |
| `AddNavObstacle(entity)` | Mesh or box obstacle |
| `BakeNavMesh([nav])` | Build the mesh |
| `CreateAgent(entity)` | Agent uses the current nav |
| `SetAgentSpeed` / `SetAgentRadius` | Motion |
| `SetAgentDestination e, x, y, z` | Straight path on the baked mesh |
| `GetAgentPathPointX/Y/Z(e, i)` | Path point |
| `AgentCountPath(e)` | Point count |
| `AgentStop e` | Stop following |
| `UpdateNav` | Step agents. Also runs once per `Flip` / `UpdateWorld` |
| `SetNavMaxSlope(deg)` / `GetNavMaxSlope()` | Skip faces steeper than `deg` when baking (terrain + mesh). Unset getter reports 45 without writing. |
| `BakeTerrainNav([terrain])` | Bake from **loaded chunk triangles**, not AABBs |

## Crowd (ORCA, no DetourCrowd)

`go-detour` v0.1.3 does not ship DetourCrowd. `CreateCrowd` / `CrowdAddAgent` / `CrowdSetDestination` / `CrowdSetAgentDestination` / `CrowdUpdate` / `SetCrowdRadius` / `GetCrowdRadius` run Detour paths, a Detour-style adaptive velocity sample, a time-to-collision sidestep, and yaw toward travel. The Recast C API is not linked. Crowd members are skipped by `UpdateNav` so they are not stepped twice. `SetNavMaxSlope(deg)` drops steep triangles when writing the bake OBJ (G3N Y-up). Hierarchical: coarse grid A* is still available via `CreateGrid`.

## Behavior trees

`CreateBehavior()` returns a tree. `BTSequence`, `BTSelector`, and `BTParallel` are composite nodes. `BTCondition(tree, fn$)` succeeds when `Function fn(entity)` returns nonzero. `BTAction(tree, fn$)` treats `0` as success, a positive number as still running, and a negative number as failure. `BTAdd(tree, parent, child)` links them. `SetBehavior(entity, tree, root)` ticks that root every frame. `ClearBehavior(entity)` stops that entity. The leaf is called with the entity id. Do not `Flip` inside a leaf. `examples/largeworld.mb` uses one tree for eight named people.

See `examples/nav.mb`, `examples/crowd.mb`, `examples/ecs_crowd.mb`.

## Regions and links

`SetNavCost nav, area, cost` changes how expensive an area is to cross. A cost below 1 is raised to 1, because a modifier under 1 breaks Detour's A*. `GetNavCost` reads it back and returns 1 when that area was never set.

`AddNavLink nav, ax, ay, az, bx, by, bz` is an off-mesh connection. When a path passes near one end (within 1.5 units), the other end is inserted. Links are stitched after the path is found, including the straight line returned when the mesh is not baked yet.

## Seeing, hearing, steering

These commands move ordinary entities. They do not require a navmesh.

`SetVision ent, range, degrees` is a yaw cone. Yaw 0 looks down +Z. `CanSee(from, target)` is true when the target is inside that range and cone, and no other entity's radius sits on the segment between them. It does not test triangles. `LinePick` does.

`SetHearing ent, radius` and `CanHear(from, target, loudness)` use horizontal distance. The reach is `radius * loudness`.

```basic
SetVision guard, 18, 70
SetHearing guard, 12
If CanSee(guard, player) Then
    SteerSeek guard, EntityX(player), EntityY(player), EntityZ(player), 4
EndIf
```

`SteerSeek`, `SteerFlee`, `SteerArrive`, `SteerWander`, and `SteerAvoid` step the entity by `speed * DeltaTime`. Arrive slows inside its slow radius. Wander turns a little each call. Avoid pushes away from other entities inside `reach`.

`SetFlock ent, group` puts the entity in a group. `FlockUpdate group, reach, separate, align, cohere` steers every member of that group: apart inside `reach`, toward the average heading, and toward the average position. The three weights default to 1, 0.4, and 0.3. `reach` defaults to 6.

`AIDebug ent` draws the vision heading and both edges of the cone for a tenth of a second.

## A small agent

`CreateAI([entity])` returns an agent id. With no entity, it creates an empty node. `AIState ai [, name$]` gets or sets a name such as `"patrol"`. `AIAddWaypoint ai, x, y, z` appends a point. `AIUpdate ai [, speed]` seeks the current point and advances when it is within 0.4. `AIDetectTarget(ai, entity)` is `CanSee` from the agent's entity.

This is a waypoint follower with a vision check. Behavior trees and the GOAP planner are still the tools for decisions.

## Decisions in the language

`CreateStateMachine`, `AddState`, `GoState`, `UpdateState`, and `StateName` live in the interpreter, so they run with no window. `AddTransition sm, from$, to$, func$` runs before the state's function. A nonzero condition switches state. `from$` of `*` matches any state.

`CreateUtility()` is a separate id on the same table. `UtilityConsider u, name$, score` stores a score. `UtilityChoose(u)` returns the name with the highest score and makes that the current name. Do not `UpdateState` a utility id. The scores are not functions. You pass the number.

```basic
u = CreateUtility()
UtilityConsider u, "idle", 1
UtilityConsider u, "attack", fear * 4
Print UtilityChoose(u)
```

## Grid A* (`github.com/quasilyte/pathing`)

| Command | Meaning |
| --- | --- |
| `CreateGrid(w, h)` | Cell grid |
| `SetGridWalkable grid, x, y, on` | `on` 0 = blocked |
| `FindPath(grid, x1, y1, x2, y2)` | Path handle |
| `PathLength(p)` | Steps including start |
| `PathX(p, i)` / `PathY(p, i)` | Cell |
