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
| [RELEASE.md](RELEASE.md) | Portable `moon build`, natives, 1.0 version |

Pathfinding (this file) is below.

# Pathfinding

Detour and grid A* for paths. Behavior trees are a third system (`CreateBehavior`). There is no GOAP library.

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

## Grid A* (`github.com/quasilyte/pathing`)

| Command | Meaning |
| --- | --- |
| `CreateGrid(w, h)` | Cell grid |
| `SetGridWalkable grid, x, y, on` | `on` 0 = blocked |
| `FindPath(grid, x1, y1, x2, y2)` | Path handle |
| `PathLength(p)` | Steps including start |
| `PathX(p, i)` / `PathY(p, i)` | Cell |
