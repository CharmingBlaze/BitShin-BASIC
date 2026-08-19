# Docs

Topic pages for script authors. Command lists stay in [COMMANDS.md](COMMANDS.md). Language: [LANGUAGE.md](LANGUAGE.md).

| Page | What it covers |
| --- | --- |
| [PHYSICS.md](PHYSICS.md) | Bodies, joints, forces, CharacterVirtual, contacts, first `Flip` / Escape |
| [VEHICLES.md](VEHICLES.md) | Landed controllers. **`CreatePlane` = mesh**; aircraft is `CreatePlaneController` |
| [NETWORK.md](NETWORK.md) | `CreateNetworkHost`, `PollNetwork` **struct**, `NET_*`, ENet vs UDP |
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
| [RELEASE.md](RELEASE.md) | Portable `bs build`, natives, 1.0 version |

Pathfinding (this file) is below.

# Pathfinding

Two systems, no planner or behavior-tree libraries.

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
| `SetNavMaxSlope(deg)` / `GetNavMaxSlope()` | Skip faces steeper than `deg` when baking (terrain + mesh) |
| `BakeTerrainNav([terrain])` | Bake from **loaded chunk triangles**, not AABBs |

## Crowd (no DetourCrowd)

`go-detour` v0.1.3 does not ship DetourCrowd. `CreateCrowd` / `CrowdAddAgent` / `CrowdSetDestination` / `CrowdUpdate` / `SetCrowdRadius` / `GetCrowdRadius` run Detour paths plus local separation and a time-to-collision sidestep. `SetNavMaxSlope(deg)` drops steep triangles when writing the bake OBJ (G3N Y-up). Hierarchical: coarse grid A* is still available via `CreateGrid`.

See `examples/nav.bb`, `examples/crowd.bb`, `examples/ecs_crowd.bb`.

## Grid A* (`github.com/quasilyte/pathing`)

| Command | Meaning |
| --- | --- |
| `CreateGrid(w, h)` | Cell grid |
| `SetGridWalkable grid, x, y, on` | `on` 0 = blocked |
| `FindPath(grid, x1, y1, x2, y2)` | Path handle |
| `PathLength(p)` | Steps including start |
| `PathX(p, i)` / `PathY(p, i)` | Cell |
