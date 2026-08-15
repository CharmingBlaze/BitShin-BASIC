# World streaming

`CreateWorldStream(chunkSize, radius)` keeps a grid of prop chunks around `SetStreamOrigin` or `SetStreamFollow`.

- CPU jobs **plan** which cells to load/unload.
- Mesh create / GL upload runs on the **Flip** thread (`enqueueGL`). Workers must not touch OpenGL.
- Terrain has its own chunk window (`SetTerrainStreamRadius`). Water can recenter (`SetWaterFollow`).

Commands: `SetStreamRadius`, `LoadChunk`, `UnloadChunk`, `ChunkLoaded`, `StreamChunkCount`. Example: `examples/stream.bb`.
