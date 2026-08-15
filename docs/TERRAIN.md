# Terrain

Heightfields on **G3N + OpenGL 3.3**. No compute, no tessellation shaders, no OpenGL 4.5.

Chunks are regular grids. Distant tiles drop every other vertex (LOD 1–2). Height samples and mesh builds that need GL stay on the Flip / GL thread; workers only plan work and queue uploads (`jobs.go`).

## Techniques researched (what we ship)

G3N has **no built-in terrain**. Community work ([josephtclegg/g3nterrain](https://github.com/josephtclegg/g3nterrain), [G3N mesh tutorial](https://github.com/jordan4ibanez/G3N-Mesh-Tutorial), [g3n#229](https://github.com/g3n/engine/issues/229)) builds a custom `geometry.NewGeometry()` with position / normal / UV VBOs and `graphic.NewMesh`. That is the path we use.

| Technique | GL 3.3 + G3N? | What we do |
| --- | --- | --- |
| CPU heightmap grid (PNG / FBM) | Yes | `GenerateHeightmap` → `CreateTerrainFromHeightmap` / `CreateTerrain`. Vertices displaced on **Y**, XZ regular. |
| Chunked LOD / geomipmap | Yes | Streamed chunks; far tiles `step=2/4` vertex skip + optional skirts via density. |
| CDLOD (Strugar) | Partial | Quadtree + vertex morph needs a vertex-texture height sample. We keep CPU grids; no CDLOD morph. |
| Geometry clipmaps (Losasso / Hoppe) | Partial | Needs vertex texture fetch + ring updates. GL 3.3 *can* VTF, G3N does not expose a clipmap. |
| Vertex texture fetch height | Possible | `texture()` in a vertex shader is legal in 3.3. Not wired; heights are baked into the mesh. |
| Height / slope splat | Yes | `mbterrain` fragment (sand / grass / rock / snow). |
| Normal maps from height | Yes | Finite-difference mesh normals + optional rock normal map. |
| Marching cubes / voxels | Optional | Not used. G3N can host a custom voxel mesh; we stay on a heightfield. |
| Hardware tessellation TES | **GL 4.x** | Terrain-OpenGL’s TES is not required. `EnableTessellation` only if `GLHasTessellation`. |
| GPU clipmaps / compute clouds | **GL 4.x** | Not shipped. Clouds are a 3.3 raymarch stand-in. |

**Best fit we ship:** CPU FBM / PNG heightfield → chunked XZ `NewGeometry` mesh → height/slope splat → `TerrainHeight` walk. Visible hills come from a real height scale (not `0.005 * freq` on a 40-unit tile).

Demo: `examples/terrain.bb` (alpine heightmap), `examples/terrain_gl.bb` (value-noise port).

## Modes

| Mode | Command | What it does |
| --- | --- | --- |
| Heightmap file | `CreateTerrain(file$, w, d, hscale)` | PNG/JPEG grayscale → finite field. Empty / `"default"` uses FBM |
| Generated map | `CreateTerrain(hm, w, d, hscale)` / `CreateTerrainFromHeightmap(hm, …)` | CPU heightmap handle from `GenerateHeightmap` |
| Load only | `LoadHeightmap(file$ [, w, d, hscale])` | Decode grayscale → **terrain** handle (existing) |
| Procedural | `CreateProcTerrain(seed, chunkSize, octaves [, hscale, worldScale])` | Infinite-ish FBM around the stream origin |
| Terrain-OpenGL | `CreateTerrainGL([seed, chunk, radius, disp, freq, octaves, scale])` | Their value-noise + splat + CPU LOD |

## Heightmap generator

Clean-room port of the documented [ergin3d/heightmap-generator](https://github.com/ergin3d/heightmap-generator) tool (**MIT**; JS was not copied). CPU only — no FlatBuffers, no GL 4.x. Seeded FBM (existing `noise2`), optional diamond-square, black/white levels, thermal + hydraulic erosion, terrace/ridged/invert/clamp/normalize, greyscale PNG.

| Command | Meaning |
| --- | --- |
| `GenerateHeightmap(w, h, seed [, octaves, scale, persist, lacun [, panX, panY, black, white, mode$]])` | Handle. `mode$` = `fbm` (default), `ridged`, `diamond` |
| `GenerateHeightmapPreset(name$, w, h, seed)` | Repo packs: `rolling-hills`, `sharp-peaks`, `archipelago`, `plateaus`, `gentle-dunes`, `alpine` |
| `ErodeHeightmap(hm [, iterations, talus, transfer])` | Thermal (steep slopes shed) |
| `HydraulicErodeHeightmap(hm [, droplets, life, inertia, cap, erode, deposit, evap])` | Droplet rivers |
| `FilterHeightmap(hm, type$, strength#)` | `smooth` `sharpen` `terrace` `ridged` `invert` `clamp` `normalize` `thermal` `hydraulic` |
| `SaveHeightmap(path$ [, hm])` | 8-bit greyscale PNG |
| `ImportHeightmap(file$)` / `LoadHeightmapData` | Image → generator handle (does **not** replace `LoadHeightmap` terrain) |
| `HeightmapTexture([hm])` | Greyscale GL texture |
| `HeightmapWidth` / `HeightmapHeight` | Size |

`CreateTerrain` / `CreateTerrainFromHeightmap` feed the existing chunk mesh + splat path — they do not rebuild it.

Demo: `examples/heightmap.bb` (writes `examples/assets/heightmap.png`).

## Geo / DEM

Lon/lat worlds use `docs/GEO.md` (`SetGeoOrigin`, `GeoProject`). `LoadGeoDEM` and `CreateTerrainFromGeoDEM` only assign projected `ox/oz/worldW/worldD` on an existing heightfield (PNG or `GenerateHeightmap`) and call `addTerrain`. They do not replace Terrain-OpenGL splat or this generator.

`TerrainHeight(x, z)` bilinear-samples the field (proc uses the same FBM). Collision and nav should use this, not an AABB box.

## Streaming + LOD

`SetTerrainStreamRadius(terrain, n)` keeps a `(2n+1)²` window of chunks around the camera / `SetStreamOrigin`. Far chunks use `step=2` when `SetTerrainLOD` is on.

Do not create GL objects on workers. The job submits; `enqueueGL` builds the G3N mesh on Flip.

## License (Terrain-OpenGL)

[mark99106/Terrain-OpenGL](https://github.com/mark99106/Terrain-OpenGL) is **MIT** (© 2019 rickie95, © 2025 Mark). Algorithms were reimplemented (value-noise + splat + sky presets). Their photo JPGs are **not** shipped. Our splat maps are original CC0.

## Terrain-OpenGL port (mark99106 / rickie95, MIT)

Full behavior port of [Terrain-OpenGL](https://github.com/mark99106/Terrain-OpenGL) onto **G3N + GLSL 330**. Their repo is OpenGL 4 (tessellation + compute volumetric clouds). We do **not** require 4.5.

| Their feature | BitShin BASIC |
| --- | --- |
| Value-noise FBM + `pow` displacement in TES | Same hash/quintic/FBM on **CPU** (`terrain_gl_noise.go`); mesh already displaced |
| Distance tessellation LOD | Chunked grid + `step=2/4` vertex skip (`SetTerrainLOD`, `SetTerrainTessMultiplier` raises CPU density) |
| 120×120 infinite tiles | Existing stream radius around camera / `SetStreamOrigin` |
| Sand/grass/rock/snow splat + rock normals | `mbterrain` fragment (height + slope). `ApplyTerrainSplat` / `SetTerrainSplat` |
| Directional light + height fog | G3N lights + CSM from `mbshadow` vertex + engine fog |
| Procedural sky + sun disk | `SetSkyPreset("default"\|"sunset"\|"sunset1")` / `SetSkyGradient` |
| Water reflection/refraction | Existing `CreateWater` + planar FBO (Gerstner, not their DUDV-only plane) |
| Compute volumetric clouds | `CreateVolumetricClouds` — 3.3 raymarch through layered 2D noise (no compute) |

```basic
t = CreateTerrainGL(0, 25, 3, 3.2, 0.035, 8, 40)
ApplyTerrainSplat(t)
SetTerrainGrassCoverage(t, 0.65)
SetTerrainWaterHeight(t, 2.15)
y# = TerrainHeight(x, z)
n# = TerrainSlope(x, z)
```

`CreateProcTexture("sand"|"grass"|"rock"|"snow"|"rocknormal"|"cloud")` builds original CC0 maps (their JPGs are **not** shipped — photo textures may have third-party licenses even though the C++ is MIT).

## Materials / GI

`SetTerrainTexture(terrain, tex)` and `SetTerrainLightmap(terrain, tex)` add textures (multiply/blend in `mbshadow`). Splat terrains use `mbterrain` instead. Hemisphere / probes tint ambient when `CreateLightProbe` / `SetProbeGrid` is used.

## Nav

`BakeTerrainNav([terrain])` bakes **mesh triangles** from loaded chunks (`SetNavMaxSlope` skips steep faces). See `docs/NAV.md`.

## Not on G3N

- Hardware tessellation (optional `EnableTessellation` only if `GLHasTessellation`; default is CPU LOD)
- True compute volumetric clouds (3.3 raymarch stand-in)
- GPU clipmaps / virtual texturing
- meshoptimizer / Basis / KTX2 (not added; scene files stay JSON/YAML/MessagePack)
- FlatBuffers

Demo: `examples/terrain_gl.bb` (full look), `examples/terrain.bb`, `examples/heightmap.bb`, `examples/largeworld.bb`.
