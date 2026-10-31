# Terrain

Heightfields on **G3N + OpenGL 3.3**. The mesh is a CPU grid. Tessellation shaders are optional later (`GLHasTessellation`); they do not run the terrain today. No compute, no OpenGL 4.5 requirement.

Chunks are regular grids. The builder raises the quad count so a cell stays near **0.4** world units (capped at 72 quads on a side). A small island chunk no longer shades as a handful of big facets. Rock lighting uses the smoothed vertex normal. Distant tiles use a coarse interior (`step=2/4`) and a **full-resolution border** so they meet the near tile. Height samples and mesh builds that need GL stay on the Flip / GL thread; workers only plan work and queue uploads (`jobs.go`).

## Techniques researched (what we ship)

G3N has **no built-in terrain**. Community work ([josephtclegg/g3nterrain](https://github.com/josephtclegg/g3nterrain), [G3N mesh tutorial](https://github.com/jordan4ibanez/G3N-Mesh-Tutorial), [g3n#229](https://github.com/g3n/engine/issues/229)) builds a custom `geometry.NewGeometry()` with position / normal / UV VBOs and `graphic.NewMesh`. That is the path we use.

| Technique | GL 3.3 + G3N? | What we do |
| --- | --- | --- |
| CPU heightmap grid (PNG / FBM) | Yes | `GenerateHeightmap` → `CreateTerrainFromHeightmap` / `CreateTerrain`. Vertices displaced on **Y**, XZ regular. |
| Chunked LOD / geomipmap | Yes | Streamed chunks; far tiles `step=2/4` inside, **full-resolution borders** so they meet the near tile (same idea as tessellation outer levels). Central-difference normals. Skirts remain as a fallback. |
| CDLOD (Strugar) | Partial | Quadtree + vertex morph needs a vertex-texture height sample. We keep CPU grids; no CDLOD morph. |
| Geometry clipmaps (Losasso / Hoppe) | Partial | Needs vertex texture fetch + ring updates. GL 3.3 *can* VTF, G3N does not expose a clipmap. |
| Vertex texture fetch height | Possible | `texture()` in a vertex shader is legal in 3.3. Not wired; heights are baked into the mesh. |
| Height / slope splat | Yes | `mbterrain`: sand / grass / dirt / rock / snow, then a **height blend** (texture luminance stands in for a height map). |
| Stochastic ground tiles | Yes | Four hashed UV offsets on the ground projection, blended by luminance (Heitz / Deliot sampling, height blend instead of a grey average). Side faces stay one sample. |
| Normal maps from height | Yes | Central-difference mesh normals `normalize(hL−hR, 2·cell, hD−hU)` plus an optional rock normal map. |
| Marching cubes / voxels | Optional | Not used. G3N can host a custom voxel mesh; we stay on a heightfield. |
| Hardware tessellation TES | **GL 4.x** | Not used on the terrain mesh. `SetTerrainDetail` changes the CPU step instead. `EnableTessellation` only compiles a standalone program. |
| Billboard grass | Yes | `TerrainFoliage`: crossed cards up close, one camera-facing quad on far chunks. Not in the heightfield. |
| Painted RGBA blend map | Yes | `SetTerrainBlendMap`. Empty texels keep the slope/height splat. |
| GPU clipmaps / CBT / virtual texturing | No | 2024 concurrent binary trees and virtual textures are a second renderer. Not planned. Clouds stay a 3.3 raymarch. |

**Best fit we ship:** CPU FBM / PNG heightfield → chunked XZ `NewGeometry` mesh → height/slope splat → `TerrainHeight` walk. Visible hills come from a real height scale (not `0.005 * freq` on a 40-unit tile).

Demo: `examples/terrain.mb` (alpine heightmap), `examples/terrain_gl.mb` (value-noise port).

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
| `GenerateHeightmapPreset(name$, w, h, seed)` | Repo packs: `rolling-hills`, `sharp-peaks`, `archipelago`, `plateaus`, `gentle-dunes`, `alpine`, `island` (one landmass whose rim sinks below 0.1, so it sits in the sea; its hills scale with the map width) |
| `ErodeHeightmap(hm [, iterations, talus, transfer])` | Thermal (steep slopes shed) |
| `HydraulicErodeHeightmap(hm [, droplets, life, inertia, cap, erode, deposit, evap])` | Droplet rivers |
| `FilterHeightmap(hm, type$, strength#)` | `smooth` `sharpen` `terrace` `ridged` `invert` `clamp` `normalize` `thermal` `hydraulic` `island` |
| `SaveHeightmap(path$ [, hm])` | 8-bit greyscale PNG |
| `ImportHeightmap(file$)` / `LoadHeightmapData` | Image → generator handle (does **not** replace `LoadHeightmap` terrain) |
| `HeightmapTexture([hm])` | Greyscale GL texture |
| `HeightmapWidth` / `HeightmapHeight` | Size |

`CreateTerrain` / `CreateTerrainFromHeightmap` feed the existing chunk mesh + splat path — they do not rebuild it. A map is finite. Past its border the ground is 0 by default. `SetTerrainEdge(t, "clamp")` carries the border height on, and `SetTerrainEdge(t, "mirror")` reflects the hills so walking off the map is more land, not a cliff into water.

Demo: `examples/heightmap.mb` (writes `examples/assets/heightmap.png`).

## Geo / DEM

Lon/lat worlds use `docs/GEO.md` (`SetGeoOrigin`, `GeoProject`). `LoadGeoDEM` and `CreateTerrainFromGeoDEM` only assign projected `ox/oz/worldW/worldD` on an existing heightfield (PNG or `GenerateHeightmap`) and call `addTerrain`. They do not replace Terrain-OpenGL splat or this generator.

`TerrainHeight(x, z)` bilinear-samples the field (proc uses the same FBM). Collision and nav should use this, not an AABB box.

## Streaming + LOD

`SetTerrainStreamRadius(terrain, n)` keeps a `(2n+1)²` window of detailed chunks around the camera, `SetStreamFollow`, or `SetStreamOrigin`. A coarser ring continues from that window out to the fog (capped at 480), the same split games use so the horizon is ground instead of sky. Its cells are a whole number of detail cells that divide the chunk, so the ring's inner edge sits on the tiles' own border vertices. Those three still work on their own. `SetPlayer` points the same window at the player until an origin or follow call overrides it. With `SetTerrainLOD` on (the default), a far chunk is coarse in the middle (`step=2`, then `step=4`) and full resolution on the border. In the last third of each band the verts slide toward the next lattice (`SetWorldMorph`) so the swap does not pop. The weight follows the stream origin's distance to the chunk and moves in eighths, one rebuild each. Trees and props keep their copies through those rebuilds. Only grass thins on far chunks. `SetWorldMorph(0)` leaves the old step swap. Skirts still drop that border so a missed sample cannot show sky. `SetTerrainTessMultiplier` raises CPU density. It is not a tessellation shader.

Do not create GL objects on workers. The job submits; `enqueueGL` builds the G3N mesh on Flip.

## License (Terrain-OpenGL)

[mark99106/Terrain-OpenGL](https://github.com/mark99106/Terrain-OpenGL) is **MIT** (© 2019 rickie95, © 2025 Mark). Algorithms were reimplemented (value-noise + splat + sky presets). Their photo JPGs are **not** shipped. Our splat maps are original CC0.

## Terrain-OpenGL port (mark99106 / rickie95, MIT)

Full behavior port of [Terrain-OpenGL](https://github.com/mark99106/Terrain-OpenGL) onto **G3N + GLSL 330**. Their repo is OpenGL 4 (tessellation + compute volumetric clouds). We do **not** require 4.5.

| Their feature | moonBASIC |
| --- | --- |
| Value-noise FBM + `pow` displacement in TES | Same hash/quintic/FBM on **CPU** (`terrain_gl_noise.go`); mesh already displaced |
| Distance tessellation LOD | Chunked grid. Interior `step=2/4`, border matches the near chunk. `SetTerrainTessMultiplier` raises CPU density |
| 120×120 infinite tiles | Existing stream radius around camera / `SetStreamOrigin` |
| Sand/grass/rock/snow splat + rock normals | `mbterrain`: height + slope weights, then height blend. `ApplyTerrainSplat` / `SetTerrainSplat` |
| Directional light + height fog | G3N lights + CSM from `mbshadow` vertex + engine fog |
| Procedural sky + sun disk | `SetSkyPreset("default"\|"sunset"\|"sunset1")` / `SetSkyGradient` |
| Water reflection/refraction | Existing `CreateWater` + planar FBO (Gerstner, not their DUDV-only plane) |
| Compute volumetric clouds | `CreateVolumetricClouds` — one noise-shaded sheet above the camera (no compute, no 3D texture) |

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

`SetTerrainTexture(terrain, tex)` and `SetTerrainLightmap(terrain, tex)` add textures (multiply/blend in `mbshadow`). Splat terrains use `mbterrain`: slope and height choose sand, grass, dirt, rock, and snow, then a **height blend** (luminance of each texture, the same idea as modern landscape materials) so the tall grains of one material poke through the next instead of a grey mix. The ground projection is **stochastic** (four hashed offsets, height-blended) so a tiled map does not repeat across a field. `SetTerrainGrassCoverage` moves the grass/rock slope line. Mesh normals are central differences at the chunk’s cell size, and far chunks keep a full-resolution border. Hemisphere / probes tint ambient when `CreateLightProbe` / `SetProbeGrid` is used.

## Nav

`BakeTerrainNav([terrain])` bakes **mesh triangles** from loaded chunks (`SetNavMaxSlope` skips steep faces). See `docs/NAV.md`.

## Beginners and experts

Same terrain. `SetPlayer` makes the world follow that entity: chunk window, water, near-chunk collision, and the 4 km shift. Beginner calls plant grass, trees, and props. Expert calls name the rule, paint the mix, set how big a triangle may be, or turn a piece of the bubble off (`SetWorldBubble`, `SetWorldSimRadius`, `SetWorldShift`, `SetWorldMorph`). See `docs/STREAM.md`.

### Beginner

```basic
hm = GenerateHeightmapPreset("alpine", 160, 160, 19)
t = CreateTerrainFromHeightmap(hm, 220, 220, 28)
ApplyTerrainSplat(t)
SetTerrainGrassCoverage(t, 0.68)
SetTerrainWaterHeight(t, 5.5)
SetTerrainSnow(True, 19)
y# = TerrainHeight(x, z)
SetPlayer(player)
```

`ApplyTerrainSplat` builds the CC0 sand, grass, rock, and snow maps and turns on `mbterrain`. Coverage, water line, and snow are the knobs. Walk with `TerrainHeight`. How to plant grass, trees, and props, then go indoors: [PLAY.md](PLAY.md). Demos: `examples/terrain.mb`, `examples/outdoor.mb`.

### Ground cover, trees, and props

Grass on the ground and grass you walk through are different. Trees and rocks are meshes sitting on `TerrainHeight`, not extra triangles in the heightfield. Scatter follows the terrain chunks: a tile unloads its copies, and builds them again when the tile comes back. They are not collision and they are not in `BakeTerrainNav`. `CreateWorldStream` is still the tool for a hand-built town.

| Layer | What it is | Command |
| --- | --- | --- |
| Ground grass | The green splat | `ApplyTerrainSplat`, `SetTerrainGrassCoverage` |
| Grass cards | Crossed quads on flat ground | `TerrainFoliage(t [, density])` |
| Trees | Instanced mesh, minimum spacing | `TerrainTrees(t, mesh [, density])` |
| Rocks and props | Same planter, looser slope rule | `TerrainProps(t, mesh [, density])` |

**Grass cards.** `TerrainFoliage(terrain [, density])` scatters crossed cards where the ground is flat, below the snow line, and above the shore. `TerrainFoliage(t, 0)` clears them. Density `1` is a light meadow. Near cards are two crossed quads. On a far chunk (`step` 2 or 4) each card is one vertical quad turned toward the camera when that chunk is built.

**Trees.** `TerrainTrees(terrain, mesh [, density])` plants copies on the same flat-ground test, with a minimum spacing so trunks do not stack. If that mesh has a texture, each copy is a card of the picture and the transparent pixels are cut out. `SetTreeCards(terrain, width, height [, crossed, face])` sets the card size in world units, `crossed` plants a second card at 90 degrees, and `face` turns the card toward the camera. `SetTreeCardCross` and `SetTreeCardFace` change one of those on their own. Face defaults on, crossed defaults off. Y comes from `TerrainHeight`. Scale varies a little around 1. `TerrainTrees(t, mesh, 0)` clears that mesh. Steep rock and snow stay bare. Trees grow in stands: low-frequency noise keeps every tree in a forest and a fifth of them in a clearing.

**Large maps.** `GenerateHeightmapPreset("island", 4096, 4096, seed)` over 16 km generates in about a second on every core and is stored 16-bit. On high and ultra a coarse vista continues past the 480 m ring out to the fog (up to 4 km), built on a worker with the ring and swapped in the same frame, so distant mountains show. The floating origin keeps terrain roots and directional lights in place on a shift; the pieces move instead, so chunks built after a shift land where they belong.

**3D trees.** A template with no texture is planted as real meshes. Its direct children come along with their own colors, scale, and offset, so a hidden trunk with crown children is one tree:

```basic
pine = CreateCylinder(0.2, 3, 5, 0).Color(92, 64, 42).Hide()
crown = CreateCone(1.9, 4.2, 6, 1, pine).Position(0, 3, 0).Color(34, 92, 46)
TerrainTrees(ground, pine, 0.3)
```

Copies are smooth-shaded from the template's own normals, each one a little lighter, darker, warmer, or cooler, and they sway with `SetWind` (a light breeze if none is set); their shadows sway with them. 3D trees are kept in 3×3-chunk blocks out to the quality tier's reach (220 m low to 480 m ultra), one batch per part per block, so a large map stays near a hundred draws. A block keeps its trees while its chunks change LOD, and new blocks are planted nearest first, one per frame. Props use the same planter, so a prop template's scale and children are kept too.

**Props.** `TerrainProps(terrain, mesh [, density])` uses the same planter for rocks, bushes, and crates. The default rule is looser than trees: anything that is not a cliff or a snowfield, including the shore. One terrain can have several prop meshes. Each call adds a layer. Density `0` removes that mesh. Far props are one vertical quad.

Beginner scene:

```basic
t = CreateTerrainFromHeightmap(hm, 220, 220, 28)
ApplyTerrainSplat(t)
TerrainFoliage(t)
tree = LoadMesh("tree.glb")
TerrainTrees(t, tree)
rock = LoadMesh("rock.glb")
TerrainProps(t, rock, 0.4)
```

**Expert.** `TerrainScatter(terrain, mesh, kind$, density)` is the same planter with the rule named: `"grass"`, `"tree"`, `"prop"`, or `"any"`. Density `0` removes that mesh and kind. Copies follow the terrain stream.

Trees stay instances (or a far quad). They are not baked into the heightfield, and there is no separate forest renderer.

### Expert

`SetTerrainBlendMap(terrain, tex)` paints the mix over the whole terrain. Channels match the CosmicLearn blend map:

| Channel | Layer |
| --- | --- |
| R | Sand |
| G | Grass |
| B | Rock |
| A | Snow |

Where a channel has weight, it replaces the slope and height mask. Where the map is empty, the shipped height-and-slope splat remains, so a half-painted image still covers the ground. `SetTerrainSplat` still supplies the textures those channels sample.

`SetTerrainDetail(terrain, pixels)` is the screen-space triangle target (MadEqua / NVIDIA terrain tessellation), mapped onto the CPU grid. `pixels` is how large one triangle may be. Smaller than 8 keeps the fine mesh farther out. Larger than 8 coarsens sooner. `0` restores the old distance steps. G3N draws indexed triangles, so this does not submit a tessellation patch. The near border stays full resolution either way.

`EnableTessellation` only tries to compile a standalone tessellation program. It does not subdivide terrain.

### Sources

Shipped behavior follows these, on the 3.3 path:

- [LearnOpenGL height map](https://learnopengl.com/Guest-Articles/2021/Tessellation/Height-map) — CPU grid from a grayscale image
- [Fayaz OpenGL terrain](https://fayaz-09.github.io/terrain.html) — central-difference normals instead of flat face normals
- [CosmicLearn terrain rendering](https://www.cosmiclearn.com/opengl/terrain-rendering.php) — the same normal, and the RGBA blend-map layout (`SetTerrainBlendMap`)
- [Rishi Mungia terrain](https://rishimungia.com/projects/opengl-terrain) — height/slope materials and billboard grass (`TerrainFoliage`)
- [Terrain-OpenGL](https://github.com/mark99106/Terrain-OpenGL) — value noise, sand/grass/rock/snow (MIT; their photos are not shipped)
- [MadEqua opengl-terrain-demo](https://github.com/MadEqua/opengl-terrain-demo) — screen-space triangle size, applied as CPU `SetTerrainDetail`
- Heitz and Deliot, stochastic tiling — ground-projection anti-tile, with a height blend instead of a linear mix (shipped in `mbterrain`)

Not planned: concurrent binary trees (Dupuy 2020, Intel Siggraph 2024) and virtual texturing. Those replace the mesh and the texture cache.

## Sculpting and painting

Scripts can reshape and paint the ground while the game runs: a level editor, craters, a road being graded. Every terrain kind has CPU heights, so every kind can be sculpted.

- **Heightmap terrains** (`CreateTerrain(file$ | hm, …)`, `CreateTerrainFromHeightmap`, `LoadHeightmap`) are edited on their own grid. `gridX` / `gridZ` are pixels of the source map. Brushes clip to the map edge. A map over 2048² keeps 16-bit heights; edits go into a delta lattice on its grid nodes instead, so a brush can still raise past the stored range.
- **Procedural terrains** (`CreateTerrain("default")`, `CreateProcTerrain`, `CreateTerrainGL`) keep their noise and add a sparse delta lattice on top, one node per mesh cell (`TerrainCellSize`). They stay infinite; only the dug-into tiles hold memory.

| Command | What it does |
| --- | --- |
| `SculptTerrain(t, x, z, radius, amount [, falloff])` | Raise by `amount` at the centre (negative lowers). Call it every frame with `amount * Time.Delta()` while a button is held |
| `FlattenTerrain(t, x, z, radius, height [, strength, falloff])` | Pull toward `height`. `strength` 1 sets it outright |
| `SmoothTerrain(t, x, z, radius [, strength, falloff])` | Pull toward the 3×3 average |
| `SetTerrainHeight(t, gridX, gridZ, height)` / `TerrainGridHeight(t, gridX, gridZ)` | One lattice node |
| `PaintTerrain(t, x, z, radius, layer [, strength, falloff])` | `layer` 0 sand, 1 grass, 2 rock, 3 snow, `-1` erases back to the slope/height splat |
| `SetTerrainPaintArea(t, x0, z0, width, depth [, res])` | World rectangle and texel size of the paint map (default: the heightmap rectangle, or the streamed window around the origin for procedural terrains; 512²) |
| `ResetTerrainEdits(t)` | Back to the loaded heights and an empty paint map |
| `PickTerrain(t [, screenX, screenY])` | Ray from the mouse (or a pixel) to the ground. `1` on a hit, point in `PickedX/Y/Z` |

`x`, `z` and `radius` are world units. `falloff` is the soft fraction of the radius (`0` hard disc, `1` soft to the centre, default `0.5`). `t` may be `0` for the current terrain. An unknown handle or a missing argument is an error that names the command.

What updates, and when:

- `TerrainHeight`, `TerrainSlope`, `PickTerrain` and scatter planting read the new heights on the same line.
- Chunk meshes rebuild once per frame in `RenderWorld`, however many brush calls ran. Only the vertices under the brush are re-sampled in the existing buffers, so the chunk keeps its entity handle. A heightmap brush costs well under a millisecond; on 8-octave `CreateTerrainGL` noise a radius-8 dab is a few milliseconds.
- Near-chunk collision (`SetPlayer` bubble) and bodies from `CreateBodyHeightfield` are rebuilt from the new heights the same frame.
- Grass cards, trees and props on a touched chunk are replanted once the brush has rested for a few frames, so a drag does not churn instances.
- Painting writes a CPU RGBA copy of the blend map and uploads only the touched texels (`glTexSubImage2D`). The first `PaintTerrain` adopts a `SetTerrainBlendMap` image when its file can be read again; otherwise it starts empty (the slope/height splat shows through).
- Ground past the detail window (the horizon ring and vista) is rebuilt on a worker when a brush lands there, so it catches up a frame or two later. The delta lattice is read under a lock, so those workers never see a half-built tile.

Limits: `BakeTerrainNav` is a snapshot, so bake again after sculpting. 3D tree blocks out past the detail window keep their old height until they are planted again. With `clamp` / `mirror` map edges the reflected land outside the map follows the edit only when its chunk next rebuilds. Brushes cover at most 512 lattice cells across. Painting outside the paint area does nothing, and the outer texel ring is never painted, because the shader clamps outside the area to it.

Demo: `examples/terrain_sculpt.mb` (the first 1.5 s carve and paint a ring by themselves). Mouse picks the ground, left raises and right lowers; `1`–`4` switch sculpt / flatten / smooth / paint, `Tab` cycles the paint layer, `[` `]` size, `-` `=` strength, `R` resets.

```basic
If PickTerrain(ground) And Input.MouseDown(1)
    SculptTerrain(ground, PickedX(), PickedZ(), 6, 3 * Time.Delta())
EndIf
```

## Not on G3N

- Hardware tessellation of the terrain mesh. `SetTerrainDetail` changes the CPU step. `EnableTessellation` only compiles a program.
- A camera-facing grass shader that turns every card every frame. Far cards face the camera when the chunk is built.
- True compute volumetric clouds (3.3 raymarch stand-in)
- GPU clipmaps, concurrent binary trees, virtual texturing
- meshoptimizer C++ and BasisLZ (not added). KTX2 Zstd and ZLIB supercompression inflate to S3TC. `OptimizeMesh` and `SimplifyMesh` use the Go `meshoptim` and `meshsimplify` packages. DDS and KTX S3TC load as compressed GPU textures. Scene files stay JSON/YAML/MessagePack
- FlatBuffers

Demo: `examples/outdoor.mb` (foliage, trees, props, time of day, rooms), `examples/terrain_gl.mb` (full look), `examples/terrain.mb`, `examples/heightmap.mb`, `examples/largeworld.mb`. Walkthrough: [PLAY.md](PLAY.md).
