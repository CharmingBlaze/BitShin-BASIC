# World streaming

Two ways to move the world around the player. The old commands are unchanged. `SetPlayer` is the short path that turns the same machinery on.

## Old commands

These do what they did before. You do not need `SetPlayer` for them.

```basic
CreateWorldStream(20, 2)
SetStreamFollow(player)
```

`CreateWorldStream(chunkSize, radius)` keeps a square of prop cells around the center. The default cell is 24 and the default radius is 2, so that is a 5×5 window. Each new cell still builds the demo ground slab and a few cones. `LoadChunk(cx, cz)` forces one cell. `UnloadChunk` frees it. `ChunkLoaded` and `StreamChunkCount` report what is resident.

`SetStreamFollow(ent)` moves that center to the entity every Flip. `SetStreamOrigin(x, y, z)` pins the center instead (y is ignored). An origin call wins over follow until the next `SetStreamFollow`.

`examples/stream.bb` is this path. `examples/largeworld.bb` uses it with a procedural terrain.

Terrain has its own window, and it always did:

```basic
SetTerrainStreamRadius(land, 2)
SetTerrainLOD(land, 1)
SetStreamOrigin(EntityX(cam), 0, EntityZ(cam))
```

`SetTerrainLOD`, `SetTerrainDetail`, `SetTerrainStreamRadius`, `TerrainHeight`, `TerrainFoliage`, `TerrainTrees`, and `TerrainProps` are the same calls. Scatter still loads and unloads with the terrain tile. It is still not collision and not nav.

`SetWaterFollow(1)` still recenters the water mesh. `SetWaterFollow(0)` still leaves a pond where you built it.

`BakeTerrainNav` still bakes the triangles that are loaded. `CreateBodyHeightfield` is still the manual heightfield, on the whole terrain bounds.

CPU jobs plan which cells to load. Mesh create and GL upload stay on the Flip thread. Workers do not touch OpenGL.

## Short path

`SetPlayer(player)` still sets the footstep body and the actor chase target. It also turns the bubble on, unless you opted out.

```basic
t = CreateTerrainFromHeightmap(hm, 220, 220, 28)
ApplyTerrainSplat(t)
TerrainFoliage(t)
SetPlayer(player)
```

That follow uses the same stream center as `SetStreamFollow`. It does not spawn the demo cones. Call `CreateWorldStream` when you want those cones, before or after `SetPlayer`.

On top of the old follow, the bubble adds:

| Piece | What you get | Turn it off |
| --- | --- | --- |
| Terrain window | Chunks around the player. LOD stays the default. | `SetTerrainStreamRadius`, `SetTerrainLOD(t, 0)` |
| Water | The water mesh recenters, if you have not called `SetWaterFollow` yourself | `SetWaterFollow(0)` |
| Near collision | A low-resolution heightfield on chunks next to the player. Far chunks are drawn only. | `SetWorldSimRadius(-1)` |
| Morph | In the last third of a LOD band, verts slide toward the next grid | `SetWorldMorph(0)` |
| Shift | Past 4096 units the world moves so the player is near zero. `EntityX` stays local. `WorldOriginX` / `WorldOriginZ` is the shift. Heights add it. | `SetWorldShift(0)` |

`SetStreamOrigin` and `SetStreamFollow` still override the center after `SetPlayer`. `examples/outdoor.bb` does this: `SetPlayer(cam)` for footsteps, then `SetStreamOrigin` every frame so the terrain window is the camera.

To keep `SetPlayer` as footsteps only:

```basic
SetWorldBubble(0)
SetPlayer(player)
CreateWorldStream(20, 2)
SetStreamFollow(player)
```

`SetWorldBubble(1)` turns the bubble back on. The second argument is the entity to follow. Omit it to use the current player.

## What to call

Beginner, one entity, terrain already created:

```basic
SetPlayer(player)
```

Same scene, you want the old prop grid as well:

```basic
CreateWorldStream(24, 2)
SetPlayer(player)
```

Old scene that should not shift or gain collision:

```basic
SetWorldBubble(0)
SetPlayer(cam)
SetStreamOrigin(EntityX(cam), 0, EntityZ(cam))
```

Hand-built town, one cell at a time:

```basic
CreateWorldStream(32, 1)
SetStreamFollow(player)
LoadChunk(0, 0)
```

`LoadChunk` still builds that cell yourself. The bubble does not replace it.
