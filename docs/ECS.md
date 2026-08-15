# Flecs

BitShin BASIC wraps **Flecs 4.1.6** (Sander Mertens). The published path `github.com/SanderMertens/flecs-go` is not an upstream Go module; this tree pins that import to `third_party/flecs-go` (C amalgamation + cgo).

One world per program. `EcsProgress` also runs once per `Flip` / `UpdateWorld` (not twice in the same frame).

Component values are four numbers plus an optional string (128 bytes). Pass a component **name** or the handle from `EcsComponent`.

## Commands

| Command | Meaning |
| --- | --- |
| `EcsWorld()` | Create the world (also happens on the first `Ecs*` call) |
| `EcsEntity([name$])` | New entity. Returns the Flecs id |
| `EcsComponent(name$)` | Register or find a component |
| `EcsSet e, comp, x [, y, z, w] [, s$]` | Set values. `comp` is a name or handle. If the third arg is a string, it is stored as text |
| `EcsGet(e, comp)` | First number (`x`) |
| `EcsGetX/Y/Z/W(e, comp)` | Number fields |
| `EcsGetS$(e, comp)` | String field |
| `EcsHas(e, comp)` | 1 if the entity has the component |
| `EcsAdd e, comp` | Add without writing values |
| `EcsRemove e, comp` | Remove a component |
| `EcsDelete e` | Delete the entity |
| `EcsAlive(e)` / `EcsValid(e)` | Liveliness |
| `EcsLookup(name$)` | Entity id or 0 |
| `EcsName e [, name$]` / `EcsName$(e)` | Get or set the name |
| `EcsQuery(expr$)` | Query handle. `expr` is Flecs DSL, e.g. `"Position"` or `"Position, Velocity"` |
| `EcsQueryCount(q)` | Matching entities (refreshes the snapshot) |
| `EcsQueryEntity(q, i)` | Entity at index `i` |
| `EcsProgress([dt#])` | Run systems. `dt` defaults to `DeltaTime` |
| `EcsCount(comp)` | How many entities have `comp` |
| `EcsParent child, parent` | ChildOf pair |
| `EcsGetParent(e)` | Parent entity or 0 |
| `EcsVersion$()` | `"4.1.6"` |

OOP `CreateCube` / `LoadMesh` handles still wrap G3N nodes. Flecs ids are a separate world for data-oriented updates.

`EcsProgress` (and Flip / `UpdateWorld`) runs Flecs systems, then a Go integrator: if both `Position` and `Velocity` exist, `Position += Velocity * dt`. Query snapshots hold up to 16384 entities.

See `examples/ecs.bb` and `examples/ecs_crowd.bb` (2000 entities).
