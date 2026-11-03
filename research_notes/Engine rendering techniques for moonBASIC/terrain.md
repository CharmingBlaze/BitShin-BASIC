# Terrain rendering and generation for moonBASIC (OpenGL 3.3 core, no compute)

Evidence labels. "Repo" findings come from reading the working tree at `internal/runtime/`. Nothing was run or rendered for these notes, so the stripe ranking is a code-derived prediction with tests to confirm it. Web findings marked "(search summary)" come from a search result, not a full read of the page. The iquilezles.org pages returned 404 to direct fetches, Golus's Medium article returned 403, and the GDC Vault talks are paywalled.

## Why do mountains show hard horizontal stripes close up, how do the causes differ, and what is best practice for normals and triplanar rock?

### Takeaway
The likeliest main cause is the default procedural rock texture, not the normals. It has a sine band in `v` and does not tile, and the side projections map `v` directly to world Y. That predicts a hard level line every 18.18 m with soft bands about 2.2 m apart inside each tile. Fix the texture and remove the shader strata first, then bake a normal map from the height field and move rock to decorrelated triplanar or biplanar sampling with correctly framed normals.

### Cited Findings

**Repo: what the code does today**
- The procedural rock texture adds `0.07 * sin(v*52 + n2*6)` to every channel — `internal/runtime/terrain_tex.go:91`.
- `52` is not a whole number of cycles per tile (52 / 2π = 8.28), and `noise2.fbm` has no wrap at the tile edge (`u*6`, `v*6` and so on, with a 256-entry lattice) — `internal/runtime/terrain_tex.go:70-72`, `internal/runtime/noise.go:54-70`. So the texture does not tile, and it wraps with `REPEAT` — `internal/runtime/terrain_tex.go:144`.
- The procedural texture is 192×192 — `internal/runtime/terrain_tex.go:142`.
- Rock is sampled at scale 0.055. Side projections use `wp.zy` and `wp.xy` with plain `texture()`, so `v` is world Y on both. Only the top projection goes through `stochasticRGB` — `internal/runtime/terrain_shaders.go:173-175,233`.
- The shader adds its own strata: `terrainNoise(vec2(WorldPos.y*0.55, (x+z)*0.02))`, at amplitude 0.10 in linear light and 0.30 otherwise. A code comment already warns these bands can become hard stripes in linear light — `internal/runtime/terrain_shaders.go:236-241`.
- Albedo goes through `toLin` after splatting — `internal/runtime/terrain_shaders.go:336`.
- The rock normal map is sampled planar from `WorldPos.xz * 0.07`, even on cliffs. Its tangent is `cross((0,0,1), n)` unless `|n.y| > 0.92`, which degenerates on faces whose normal is near ±Z — `internal/runtime/terrain_shaders.go:307-314`.
- Bump noise perturbs only the X and Z normal components, driven by `WorldPos.xz` — `internal/runtime/terrain_shaders.go:292-305`.
- A cavity term darkens albedo using `dFdx(n)` and `dFdy(n)` — `internal/runtime/terrain_shaders.go:337-340`.
- Vertex normals are central differences through a global sampler, so neighbouring same-LOD chunks share a stencil. Epsilon is the mesh cell clamped to 0.2–8 and takes no account of the height map's own cell size — `internal/runtime/terrain.go:298-322,385-388,414`.
- Height sampling is bilinear — `internal/runtime/terrain.go:191-199`.
- Interior vertices take normals from the morphed height function and border vertices from the raw one — `internal/runtime/terrain.go:410-414`.
- All regular quads use the same diagonal, `a,b,d / a,d,c` — `internal/runtime/terrain.go:473`.
- Height storage: `image.Gray16` is kept at 16 bits. `image.Gray` and any RGB PNG path yield 8-bit steps — `internal/runtime/terrain.go:268-290`. Maps over 2048² are compacted to 16 bits, about 3 mm a step for 180 m of relief — `internal/runtime/terrain.go:42-48`.

**Web: quantization, normals, triplanar**
- An 8-bit height map has 256 levels, which can cause banding or stepped terrain — [Meshy glossary](https://www.meshy.ai/3d-glossary/height-map) (search summary).
- Stair stepping is commonly traced to an 8-bit stage in the pipeline, and moving that stage to 16 bits fixes it — [Polycount](https://polycount.com/discussion/comment/2444929) (search summary).
- Blurring after converting 8-bit data to 16-bit does not remove the steps — [Terragen forum](https://planetside.co.uk/forums/index.php?msg=212539) (search summary). A jMonkeyEngine user reports the opposite in practice: steps persisted unless the map was smoothed after loading — [jME forum](https://hub.jmonkeyengine.org/t/jagged-terrain/19305) (search summary). Both are anecdotes. Re-exporting from the source at higher depth is the safer conclusion.
- Losasso and Hoppe store an 8-bit-per-channel normal map at twice the geometry resolution, because one normal per vertex is too blurry and a map is more efficient than per-vertex normals. The map is computed from the geometry on update — [Geometry Clipmaps paper](https://www.hhoppe.com/geomclipmap.pdf).
- GPU Gems 3 triplanar: weights come from `abs(normal)`, tightened by subtracting about 0.2 and scaling by about 7, clamped, then normalized so they sum to 1. Each projection needs its own tangent basis, built by swizzling and sign flips — [GPU Gems 3, ch. 1](https://developer.nvidia.com/gpugems/gpugems3/part-i-geometry/chapter-1-generating-complex-procedural-terrains-using-gpu).
- Using mesh tangents for triplanar normal maps is a common shortcut that Golus calls flawed; per-projection frames are the correct approach — [Golus](https://bgolus.medium.com/normal-mapping-for-a-triplanar-shader-10bf39dca05a) (search summary).
- Whiteout adds the XY components and multiplies Z, then normalizes, at 7 SM3 ALU instructions. RNM is 8 with normalization and keeps more detail — [Self Shadow, Blending in Detail](https://blog.selfshadow.com/publications/blending-in-detail).
- RNM was not designed for object or world normals; users swizzle the geometry normal into the projection's frame, blend, and swizzle back — [Self Shadow discussion](https://discourse.selfshadow.com/t/blending-in-detail/21?page=3) (search summary).
- Biplanar mapping uses two fetches rather than three, for some extra arithmetic — [Quilez, biplanar](https://iquilezles.org/articles/biplanar) (search summary). Artifacts appear near normals where all three axes are equal, such as (±1, ±1, ±1) — [0b5vr note](https://scrapbox.io/0b5vr/Biplanar%20Mapping) (search summary).

### Inferences

**Numbers derived from the code**
- Rock tile height: 1 / 0.055 = 18.18 m. Texel: 18.18 / 192 = 9.5 cm.
- Texture bands: 18.18 / 8.28 = 2.2 m each. The sine phase jumps 1.73 rad at the wrap (52 mod 2π), and the fbm values also jump.
- Shader strata: value-noise cells 1 / 0.55 = 1.82 m tall, and level across about 50 m along the x+z diagonal.
- Contrast: the texture's mean grey is about 0.45, so ±0.07 is about ±15% before decoding and about −31% to +37% after an approximate 2.2 gamma decode, before the 0.88 mix. The linear path reduced the shader strata but not this sine, so linear light makes the texture bands stand out more.

**Ranked causes for this codebase**

| Rank | Cause | Signature | Decisive test |
|---|---|---|---|
| 1 | Rock texture wrap seam (non-tiling procedural texture) | Hard one-texel line, level at every multiple of 18.18 m world Y, on all cliff faces. Matching vertical seams every 18.18 m | Change 0.055 to 0.11: spacing halves. Load a known-tiling rock texture: the lines vanish |
| 2 | Sine strata in the rock texture | Soft bands about 2.2 m apart, repeating every 18.18 m, stronger in linear light | Drop the `strata` term from `terrain_tex.go:91` |
| 3 | Shader strata | Irregular bands in 1.82 m cells, unrelated to texture scale | Set `strataK = 0`, or see that the bands ignore a change to 0.055 |
| 4 | 8-bit height source (only if the map is an 8-bit gray or RGB PNG) | True contour lines on every face at `hscale/255` spacing (0.71 m for 180 m). Visible in wireframe and silhouette, and survives a white albedo | Log the decoded image type. Spacing scales with height scale, not texture scale |
| 5 | Normal epsilon below the height-map cell | Lines every height-map cell along grid axes. They look horizontal only on faces pointing along +X, −X, +Z or −Z, and cross contours on diagonal faces | Render normals as colour. Set `eps = max(hfCell, normCell)` |
| 6 | Tangent degeneracy in the rock normal path | Sharp lighting breaks on cliffs facing near ±Z, with planar smearing down the fall line | Disable `TerrainHasRockN` |
| 7 | Constant triangle diagonal | Parallel diagonal sawteeth on one slope direction, following the mesh, not level | Flip the diagonal: the pattern mirrors |
| 8 | Shadow acne (not on the coordinator's list, worth ruling out) | Bands that crawl as the sun moves and change with the shadow bias | Disable the shadow term |

Same-LOD per-chunk normal mismatch is unlikely because the stencil is shared. A rectangular outline at LOD borders is a separate issue from the morphed-versus-raw normal stencil, and it would form boxes rather than level bands.

**Quick triage views, in order**
1. White albedo with lighting on. Stripes that vanish are texture or strata (causes 1–3). Stripes that stay are normals, geometry or shadows.
2. Unlit albedo. Stripes that stay are in the colour path.
3. Normals as colour. Lines here are normal or height causes.
4. Move the terrain up 5 m in world Y. Causes 1–3 stay at the same world height and slide across the rock, while causes 4–5 move with the land.

**Best practice for normals**
- Bake one world-space normal texture per finite map, at 1–2× the height resolution (Hoppe uses 2× of the rendered geometry), from float heights with epsilon of one height-map cell or a Sobel kernel. Do this before any 16-bit compaction.
- Store X and Z in `GL_RG8` or `GL_RG16` and rebuild Y in the fragment shader. RG8 has about 0.45° of angular step, and Hoppe's 8-bit map shows it is adequate. Prefer RG16 for very gentle slopes with specular.
- Use mipmaps and renormalize. Filtered normal mips also give distant LODs a stable average, which interpolated per-vertex point samples do not.
- Use the map for lighting and vertex normals only as a fallback. This can drop 12 bytes per vertex and makes lighting independent of LOD, which removes the morphed-versus-raw mismatch.
- Update sculpted regions with `glTexSubImage2D`. Infinite procedural terrain needs per-chunk normal tiles, or vertex normals with `eps >= hfCell`.
- In the same bake, compute slope, curvature, AO and erosion flow to replace the screen-derivative `cavity` term, which depends on view distance and shows triangle edges in `dFdx(n)`.

**Best practice for triplanar rock on GL 3.3**
1. Make procedural textures truly tile: wrap the noise lattice at an integer period per octave, using integer frequencies. Remove the sine, or use a whole number of cycles and warp it, then lower it well below ±0.07.
2. Base blend weights on the geometry normal only, not on a detail-bumped one.
3. Decorrelate the projections so rows stop lining up around a mountain. Offset X and Z projections by different constants, and shift `v` by low-frequency noise of horizontal position (a 2–6 m amplitude over 50–200 m). Any strata should tilt and warp by the same field, as a real dipping layer does.
4. Sample side planes through tile hiding as well, or at least mix a macro-scale layer at a non-integer scale ratio such as 0.055 and 0.0137.
5. Sample a matched normal for each projection, with a Whiteout blend against the swizzled geometry normal and sign handling for negative faces. Cost is about 7 ALU per plane per Self Shadow.

```glsl
// GL 3.3, world-space triplanar normal. Sketch, not tested in moonBASIC.
vec3 w = max(abs(g) - 0.2, 0.0); w = pow(w, vec3(4.0)); w /= (w.x + w.y + w.z);
vec3 tx = texture(N, uvX).xyz * 2.0 - 1.0;  // uvX = wp.zy*s + offsetX
vec3 ty = texture(N, uvY).xyz * 2.0 - 1.0;  // uvY = wp.xz*s
vec3 tz = texture(N, uvZ).xyz * 2.0 - 1.0;  // uvZ = wp.xy*s + offsetZ
tx = vec3(tx.xy + g.zy, abs(tx.z) * g.x);   // Whiteout in swizzled space
ty = vec3(ty.xy + g.xz, abs(ty.z) * g.y);
tz = vec3(tz.xy + g.xy, abs(tz.z) * g.z);
vec3 n = normalize(tx.zyx * w.x + ty.xzy * w.y + tz.xyz * w.z);
```

Check the swizzles against the repo's Z flip and the texture's green-channel convention, since `toG3N` flips Z.

### Gaps
- No stripes were observed directly, and a user-supplied rock texture would change ranks 1 and 2. No screenshot or runtime capture was available.
- Golus's full text was not readable (403), so the snippet above follows GPU Gems 3 and the Whiteout formula from Self Shadow, not his code.
- I found no source comparing central differences with Sobel filtering for terrain, so that choice is engineering judgement.

## Which geometry LOD fits GL 3.3: geomipmapping, CDLOD or geoclipmaps?

### Takeaway
All three fit GL 3.3. Keep the current chunks in the short term, since seams are not the reported problem. If LOD gets rewritten, CDLOD suits moonBASIC best: uneven mountains, a finite height map, per-vertex morphing, and no stitching strips.

### Cited Findings
- CDLOD uses a quadtree of regular grids, drawn from source heightmap data, with an LOD function based on precise 3D distance. It needs no stitching meshes and works on Shader Model 3.0 or later — [CDLOD repo](https://github.com/fstrugar/CDLOD).
- CDLOD morphs each vertex by its own distance, unlike [Ulrich 02]'s per-chunk morph. Eight triangles collapse into two, leaving degenerates, so the mesh exactly matches the next level before the swap, with no seams or T-junctions — [CDLOD paper](https://raw.githubusercontent.com/fstrugar/CDLOD/master/cdlod_paper_latest.pdf).
- The morph itself is short vertex shader code, ending in `vertex.xy - fracPart * g_quadScale.xy * morphK` — [CDLOD paper](https://raw.githubusercontent.com/fstrugar/CDLOD/master/cdlod_paper_latest.pdf).
- Strugar's critique of clipmaps: LOD follows the observer's 2D position and only partly accounts for height, which distributes detail poorly on uneven terrain. Stitching strips add cost and can cause shadow-map and overdraw artifacts — [CDLOD paper](https://raw.githubusercontent.com/fstrugar/CDLOD/master/cdlod_paper_latest.pdf). This is the CDLOD author's own comparison.
- CDLOD limits: a node handles one transition between two LOD layers, so settings must suit the dataset, and bad settings cause seams. It adds a small quadtree memory cost, with otherwise similar triangle output and batch count to GPU clipmaps. Vertex texture sampling is expensive, but the paper says it is usually not the main limit — [CDLOD paper](https://raw.githubusercontent.com/fstrugar/CDLOD/master/cdlod_paper_latest.pdf).
- A commenter disputes "continuous" and calls CDLOD interpolated discrete LOD, while others reply that a segment fully morphs before swapping. Tuning effort is also disputed — [GameDev.net thread](https://gamedev.net/forums/topic/620084-a-good-heightmap-lod-technique-no-vtf-please) (search summary).
- Original geometry clipmaps keep nested grids centred on the viewer, in video-memory vertex buffers. Transition regions of width n/10 morph geometry, and zero-area triangles prevent T-junctions. The system ran at 60 frames/sec on 2004 hardware — [Losasso and Hoppe](https://www.hhoppe.com/geomclipmap.pdf).
- GPU clipmaps need vertex texture fetch. Removing the fetch raised a GeForce 6800 GT test from 130 to 185 fps. Draw calls are about 6L+5, or 71 at L=11, and instancing could reach about 35. Frustum culling draws about 4 of 12 blocks per level. Levels are dropped when their extent falls below 2.5× the viewer's height. Triangles are about 5 pixels at n=255 and 1024×768 — [GPU Gems 2, ch. 2](https://developer.nvidia.com/gpugems/gpugems2/part-i-geometric-complexity/chapter-2-terrain-rendering-using-gpu-based-geometry).
- An implementer calls clipmap trim pieces a necessary evil that was hard to line up — [Tymoon, Geometry Clipmaps](https://reader.tymoon.eu/article/365) (search summary; attribution to this page among the results was not confirmed). Another stores pre-averaged heights per level in a secondary texture instead of adding stitching geometry — [nostatic blog](https://blog.nostatic.org/2010/07/terrain-level-of-detail-dealing-with.html) (search summary). Terrain3D is an MIT-licensed Godot clipmap terrain — [Terrain3D](https://github.com/outobugi/Terrain3D/wiki) (search summary).
- Far Cry 5 uses a GPU compute pipeline for LOD, culling, stitching and rendering — [GDC Vault](https://gdcvault.com/play/1025261/Terrain-Rendering-in-Far-Cry). This is a compute design and outside the base GL 3.3 target.
- Repo: chunks are CPU-built, with the morph passed per chunk. Borders are never morphed and stay at full resolution, using fans around a center vertex — `internal/runtime/terrain.go:358-384,436-477`.

### Inferences

| Method | Seams | Popping | CPU | GPU | Fit |
|---|---|---|---|---|---|
| Current: per-chunk morph with full-res borders (geomipmap family) | Solved by shared borders, with skirts too (`skirtDrop`) | Per-chunk, not per-vertex | Highest: Go rebuilds meshes | No vertex fetch, but more vertices at borders and many unique meshes | Works today |
| CDLOD | None if ranges are set well | Very low | Quadtree traversal only | One instanced grid and vertex fetch | Best quality for mountains |
| Clipmaps | Trim and degenerate strips | Low | Ring updates, or little with a static map | About 5 pixels per triangle, centred on the viewer | Best for huge or streaming maps and low flight. Weaker on tall relief |

- CDLOD also keeps shadow passes simple, because the same grid and heightmap draw there.
- Reuse the existing `instance.go` path if it allows a per-instance offset and scale. Otherwise a uniform per node costs one draw call per node.
- Per-vertex morph needs a height texture. Use `GL_R16` for 16-bit data, with manual bilinear in the vertex shader if needed, and skirts remain as a cheap safety.
- Tessellation is optional and GL 4 only. Note the repo's comment already treats the border rule as the GL 3 substitute for outer tessellation levels.

### Gaps
- De Boer's geomipmapping paper was not read, so the column for that family comes from repo code and secondary descriptions.
- No source benchmarks all three methods on the same hardware.
- GL 3.3's vertex image-unit guarantee is from memory and needs a check against the GL 3.3 spec.

## How do you shade convincing terrain?

### Takeaway
Most of the visible gain is in the bake and the layer budget, not in more noise. The target is macro colour, baked normals, slope, curvature and flow masks, then two or three layers per pixel with height blending and tile hiding only where tiling can be seen.

### Cited Findings
- Height blending treats a layer's value as a height: the highest wins, with blending when heights are close — [jME forum](https://hub.jmonkeyengine.org/t/improved-terrain-splatting-and-parallax-mapping/27844) (search summary).
- A vendor doc says a lookup costs about 7× a normal one, but attributes that figure to both height-based blending and tile removal in different sections, so it is unreliable. It is also offline-renderer guidance — [3Delight docs](https://documentation.3delightcloud.com/x/QIevMg) (search summary; the exact page for each statement was not confirmed).
- Quilez's texture repetition methods: per-tile random offset and orientation with smooth Voronoi Gaussian blends, and a two-fetch method using about 8 virtual copies by constant offset — [Quilez, texture repetition](https://iquilezles.org/articles/texturerepetition/) (search summary).
- Triplanar fixes stretching on steep terrain but creates blending problems near 45° where splat layers overlap — [Tuts+](https://code.tutsplus.com/use-tri-planar-texture-mapping-for-better-terrain--gamedev-13821a) (search summary).
- Far Cry 5's procedural pipeline covers biomes, terrain texturing, freshwater networks and cliff rocks — [SideFX talk page](https://www.sidefx.com/learn/talks/procedural-world-generation-far-cry-5/) (search summary). Far Cry 5 also supports procedural cliffs and displacement geometry — [GDC Vault](https://gdcvault.com/play/1025261/Terrain-Rendering-in-Far-Cry).
- Ghost of Tsushima's specular horizon occlusion accounts for the tilt between the normal-mapped and vertex normals — [Patry, Advances 2021](https://www.advances.realtimerendering.com/s2021/jpatry_advances2021/index.html). This helps baked and detail normals on slopes.
- Repo: `triplanarRGB` costs six fetches per layer (1 + 4 + 1) unless `TerrainCheap` is set. Five texture layers are always evaluated — `internal/runtime/terrain_shaders.go:162-177,231-244`.
- Repo: `stochasticRGB` uses plain `texture()` with per-cell random offsets — `internal/runtime/terrain_shaders.go:145-148`.
- Repo: height blending uses luminance as height — `internal/runtime/terrain_shaders.go:179-193`.
- Repo: the grass tiles are 11.1 m and 7.7 m (scales 0.09, 0.13) on 192-texel maps. Its fine `speck` uses frequencies of 70 and 140 cycles per tile on `grass`, and 54 and 108 on `grass2` — `internal/runtime/terrain_tex.go:80,86,142`, `internal/runtime/terrain_shaders.go:231-232`.

### Inferences

**Ranked changes (visual gain for cost)**
1. Fix tiling and the stripe sources: near-zero runtime cost.
2. Baked normal plus masks: one fetch for large gain, and it can replace several noise calls.
3. Macro colour map from the height, slope, flow and biome bake, multiplied over the splat: one fetch. It breaks repetition at all distances.
4. Grass detail layer: repeat every 1–2 m with one grayscale or albedo-and-height fetch, faded by distance. At 5.8 cm/texel the present map is blurry close up, and 140 cycles over 192 texels is 1.37 texels per cycle, past the 2-texel Nyquist limit, so the finest grass octave aliases at bake time and mips average it away. That likely explains "grass lacks fine texture". Raising the texture to 1024 also helps, at a memory cost of about 5.3 MB uncompressed with mips per layer.
5. Weighted early-out: skip a layer's fetches when its weight is about zero. Dynamic branching can reduce 30+ fetches to about 6–12 for most pixels. Use `textureGrad` or `textureLod` in branches, since implicit derivatives inside them are undefined or unreliable.
6. Replace `stochasticRGB`'s four-fetch form on cheap tiers with Quilez's two-fetch method.
7. Fix the stochastic mip spike: in pixel quads straddling a cell, random offsets create large implicit derivatives, which selects a blurry mip and can draw one-pixel grid lines. `textureGrad(tex, uv+o, dFdx(uv), dFdy(uv))` is core in GL 3.3 and resolves it. The same issue applies to any offset-based technique.
8. Biplanar on cliffs: two side fetches instead of three, with `textureGrad`.
9. Curvature and flow masks: use them to put scree, dirt and darker staining in gullies and exposed rock on ridges. Export the erosion droplet density as the flow mask. This adds realism for no extra fetches if packed in the bake.
10. Better height for blending: store real height in rock and grass alpha instead of luminance. Luminance ties blend shape to colour.

Parallax and displacement are low priority on the target GPU. A displaced near-field mesh is optional.

### Gaps
- I could not read the original Heitz and Neyret or Deliot and Heitz texts, or Booth's height-blend article. The repo credits them, so the formulas should be checked against those sources before being cited.
- I found no published comparison of fetch counts or ms for these tile-hiding methods on one GPU.
- No source was read for how Horizon texture-blends terrain, despite searching.

## How do you generate better mountains, and how do you avoid striped strata?

### Takeaway
"Simple" mountains probably come from the noise basis and the octave budget before erosion is at fault. Add per-octave ridging with derivative damping and rotated octaves, run a coarse-to-fine erosion pass on CPU at bake time, and export its masks. Strata should be geology-shaped: warped, dipping and low contrast.

### Cited Findings
- Quilez injects analytic noise derivatives into fBm, dividing each octave by `1.0 + dot(d,d)`, which damps detail on steep accumulated slopes and yields erosion-like flats beside rough areas. Analytic derivatives are faster and more accurate than central differences. He also rotates the domain between octaves and notes that cubic fades cause derivative artefacts, so quintic is smoother — [Quilez, value noise derivatives](https://iquilezles.org/articles/morenoise/) (search summary).
- Musgrave's ridged multifractal appears in *Texturing and Modeling* — [GameDev.net white paper](https://gamedev.net/tutorials/programming/graphics/white-paper-procedural-terrain-generation-with-r2452) (search summary).
- Musgrave, Kolb and Mace 1989 is an early work on thermal weathering and hydraulic erosion — [SIGGRAPH archive](https://history.siggraph.org/?p=119354). Thermal weathering models material shedding according to local gradient — [Beneš and Forsbach 2002](https://www.cs.purdue.edu/cgvlab/www/resources/papers/Benes-2002-Visual_simulation_of_hydraulic_erosion.pdf).
- Beneš and Forsbach split hydraulic erosion into separate steps and name water transport as the weakest part of erosion algorithms — [Beneš and Forsbach 2002](https://www.cs.purdue.edu/cgvlab/www/resources/papers/Benes-2002-Visual_simulation_of_hydraulic_erosion.pdf).
- Mei, Decaudin and Hu 2007 is GPU hydraulic erosion — cited in [arXiv 2210.14496](https://www.arxiv.org/pdf/2210.14496) (search summary).
- Sebastian Lague's droplet erosion is MIT-licensed — [itch.io page](https://sebastian.itch.io/hydraulic-erosion) (search summary). An aggregator suggests radius 2–4, 30–64 steps and 30k–150k droplets for 512² — [getbutterfly](https://getbutterfly.com/procedural-eroded-terrain-in-three-js-theory-techniques-field-notes/). Treat those as unverified starting values.
- Hydrology-first generation builds terrain from river networks — [Génevaux et al.](https://history.siggraph.org/?p=108417). A survey of digital terrain modelling is also available — [Galin et al. resource page](https://perso.liris.cnrs.fr/eguerin/demrepo/) (search summary).
- Repo: `fillMountainIsland` takes one `abs` of a 6-octave fBm result for ridges, then squares it, with domain warping and a mask. The final pass is `pow(v, 1.35)` — `internal/runtime/heightmap_gen.go:131-155`.
- Repo: Perlin noise uses only four diagonal gradients (`±x±z`) — `internal/runtime/noise.go:41-52`. `fbm` uses no rotation and caps at 8 octaves — `internal/runtime/noise.go:76-78,85-91`.
- Repo: thermal and droplet hydraulic erosion functions exist — `internal/runtime/heightmap_gen.go:425,499`.

### Inferences
- `abs(sum of octaves)` creates one crease system. Per-octave `abs` with weight feedback creates branching ridges that look much more like mountains.
- The island samples `u*4`, so 6 octaves top out at 128 cycles across the map (with the 0.8 factor on ridges, about 102). On a 4096² map that leaves most resolution unused, and 9–11 octaves would use it. Raise the cap and stop near the sample Nyquist limit.
- The four diagonal-only gradients and unrotated octaves likely add a diagonal or axis grain. Move to 8–12 gradients or a rotated domain per octave, using a small matrix such as Quilez's.
- Recommended CPU pipeline, in order:
  1. Warped ridged base with derivative damping, on a coarse grid such as 1024².
  2. Droplet erosion in stages at 512², 1024² and 2048², upsampling between each, so large valleys form before small gullies.
  3. Thermal pass with talus around 30–40° for loose slopes. Keep exposed rock steeper by using a lower-erosion mask at high flow.
  4. Export flow, deposition, slope and curvature masks.
- GPU erosion without compute is possible via ping-pong float FBOs on GL 3.3, as in the Mei grid style, but `GL_R32F` render targets and readback add driver risk. CPU in Go across cores at bake is the safe default, and the repo already has `parallelRows`.
- Strata rules:
  - Do not use world Y alone. Use `y + dip·(x·a + z·b) + warp(xz)` with warp amplitude greater than one band height.
  - Use irregular band widths from a 1D noise lookup or small gradient texture.
  - Mask strata to steep, low-flow rock and fade by distance.
  - Stay at ±5% or less in linear space, and feed it into the normal or a height value for relief instead of painting darker colour.
  - Geometric terracing should follow the same warped field if used at all.

### Gaps
- Olsen 2004, Cordonnier et al. 2016 (stream power), and Nick McDonald's particle erosion were not retrieved, so no numbers are cited from them. Check these before picking a stream-power path.
- The Beneš PDF gave no talus angle in the extracted text, so the 30–40° above is a starting value and needs tuning.
- Published per-megapixel timings for CPU erosion in Go were not found. Benchmark the repo's functions.

## How is grass done: instancing, cards, and terrain colour matching at distance?

### Takeaway
For GL 3.3 and Intel-class targets, use a tiered approach: instanced clumps or cards in the near field, with a density and width fade, then a terrain-only detail layer whose colour matches the grass. The most important trick is sampling the terrain's colour map at each blade root so a blade blends into the ground as it fades.

### Cited Findings
- Ghost of Tsushima generates individual blades on the GPU, each with its own procedural appearance and animation, covering acres within memory and performance limits — [GDC 2021 session](https://gdcvault.com/play/1027033/).
- Secondhand notes on that talk say that at distance blades double in width as count halves — [Xybre's notes](https://publish.obsidian.md/xybre/3.+Reference/Creative/Development/Gamedev/Visuals/Shaders+-+Grass+and+Foliage) (search summary). A reimplementation stretches blades horizontally in view space when they face the camera edge-on, following the talk — [GodotGrass](https://github.com/2Retr0/GodotGrass) (search summary).
- A Unity reimplementation uses compute plus indirect instancing, with colour from a length gradient, clump colour and albedo texture — [Rademan](https://cainrademan.artstation.com/projects/QndNax) (search summary).
- Unity terrain details support GPU instancing, and grass can be quads or meshes. A billboard option is available, and with sparse grass the clump rotations can look strange. Placement varies by Perlin noise, and `Align to Ground` controls normal alignment — [Unity Manual](https://docs.unity3d.com/2022.3/Documentation/Manual/terrain-Grass.html) (search summary).
- A Godot clipmap grass shader uses dithered fade, computes distance per vertex and reads terrain normals from a heightmap normal texture — [Godot Shaders](https://godotshaders.com/shader/wandering-clipmap-stylized-grass/) (search summary).
- A vertex-buffer grass workaround translates a static billboard buffer on a height map without vertex texture fetch — [MonoGame community](https://community.monogame.net/t/vertex-texture-fetch-in-an-opengl-project/10481) (search summary).
- Instancing mainly targets vertex upload and CPU processing, not fillrate — [Khronos forum](https://community.khronos.org/t/arb-meeting-notes-from-jun-sept-dec/38669/49) (search summary).
- Repo: scatter, tree and impostor code already exists in `internal/runtime/terrain_scatter.go`, `terrain_trees3d.go` and `terrain_impostor.go`.

### Inferences
- GL 3.3 version of the Tsushima idea, without compute:
  - A static tile VBO with per-blade seed. Use `gl_InstanceID` for per-tile offset, or a small uniform or texture buffer of tile positions.
  - The vertex shader fetches height, normal and terrain colour from baked textures, and does procedural curve and wind on a 3–7 vertex blade.
  - Cull tiles on CPU with a simple quadtree, and sort them roughly front to back.
  - LOD is an index-count choice (7, 5 or 3 vertex blades), and a per-blade seed threshold drops blades smoothly.
- Terrain matching:
  - Root colour is `macroColor(xz)`, and tips brighten from it.
  - Blend the blade normal toward the terrain normal with distance, which stops blades from reading as bright noise.
  - Fade out by shrinking height to zero, not by alpha, when MSAA alpha-to-coverage is unavailable.
  - Put a matching grass-coloured detail layer on the ground beneath, so the fade has nowhere visible to go.
- Cards remain the Intel-safe tier. Alpha test costs overdraw and disables some early-Z benefits. Geometry blades have far less overdraw per area but more vertices. Mesh clumps of about 8–15 blades are a good midpoint.
- Grass should come after the terrain fixes, since the detail layer gets much of the perceived gain at a fraction of the cost.

### Gaps
- The Tsushima talk's tile size, vertex counts, LOD scheme and albedo-to-colour method were not verified, since the talk is paywalled and no reliable summary covered them. The video exists at [YouTube](https://www.youtube.com/watch?v=Ibe1JBF5i5Y) and should be watched before quoting specifics.
- I found no source for how Unity or Godot match blade colour to terrain at distance in core features.
- Horizon's GPU placement material was not found by search.

## What does each cost on Intel HD class GPUs?

### Takeaway
I found no credible published Intel HD numbers for these techniques. Fetch and overdraw counts are the available guide, and the repo's own shader is the biggest measurable cost: about 32 fragment texture fetches on the non-cheap tier, before shadows.

### Cited Findings
- Repo: 5 layers × 6 fetches, plus rock normal and blend map, is about 32 fetches per pixel. `TerrainCheap` drops to 5 plus extras — `internal/runtime/terrain_shaders.go:162-177,231-244,275,313`.
- Repo: fragment noise also runs roughly 15+ `terrainNoise` calls per pixel, each with four hashes — `internal/runtime/terrain_shaders.go:202,219,236-237,242,248-250,264,292-294,301-303`.
- One forum report describes 50 ms for 768 triangles with instancing on Intel HD 2000/3000 under OpenGL 3.3, with no driver or code details — [GameDev.net](https://www.gamedev.net/forums/topic/661100-gldrawinstanced-unsupported-intel/) (search summary). This is a single anecdote, and it suggests a driver path problem, not a hardware limit.
- Another user reports Intel HD hardware limited to OpenGL 2.1 — [Khronos forum](https://community.khronos.org/t/system-compatibility-with-opengl-3-3/75267) (search summary). Generation is unspecified and it conflicts with the report above, so GL 3.3 availability must be checked per generation.
- An older post criticizes Intel's Windows OpenGL drivers — [Ozone3D](https://www.ozone3d.net/blogs/lab/tag/opengl/) (search summary). It is dated.
- Vertex fetch is not free: 130 → 185 fps without it on 2004–05 NVIDIA hardware, per [GPU Gems 2, ch. 2](https://developer.nvidia.com/gpugems/gpugems2/part-i-geometric-complexity/chapter-2-terrain-rendering-using-gpu-based-geometry). This does not transfer directly to Intel.

### Inferences

Relative cost guesses for a low-end integrated GPU, 1080p. These are engineering estimates and need measuring.

| Technique | Added per-pixel work | Relative cost | Gain | Tier |
|---|---|---|---|---|
| Fix tiling and strata | Removes work | None or negative | Very high | All |
| Baked normal (RG8) | +1 fetch, minus noise normals | Low | Very high | All |
| Macro and mask bake | +1–2 fetches | Low | High | All |
| Layer weight early-out, max 3 layers | −15 to −25 fetches | Large saving | None direct | All |
| Detail layer | +1–2 fetches | Low | High for grass close up | All |
| Height blend | ALU only with alpha height | Low | High | All |
| Two-fetch tile hiding | +1 per layer | Low-medium | Medium | Medium up |
| Biplanar rock | 2 fetches + ALU | Low-medium | High on cliffs | Medium up |
| Triplanar with normals | 6 fetches on rock | Medium | Slightly above biplanar | High |
| Four-fetch stochastic per layer | +3 per layer | High | Medium | High only |
| CDLOD | Vertex fetch, few CPU rebuilds | Low-medium | Medium | Medium up, if drivers behave |
| Blade grass | Vertex-heavy, overdraw | High | High near field | High |
| Card grass | Overdraw-heavy | Medium-high | Medium | Low-medium, short range |

**Suggested budgets on Intel HD**
- 8–12 fragment fetches, with no fragment fBm beyond 1–2 noise calls.
- Render at 720p–900p scale if needed. Fragment work dominates on integrated GPUs, so cutting resolution is the strongest single lever.
- About 100–300k terrain triangles, using the existing LOD.
- Grass range about 30–50 m, with clumps and no MSAA alpha-to-coverage reliance.
- Pack layers into a `GL_TEXTURE_2D_ARRAY` (core in 3.3), which reduces sampler bindings and allows one shared sampler state.
- Bake on CPU at load, then treat masks as textures rather than per-pixel noise.
- Measure with `GL_TIME_ELAPSED` timer queries (core in 3.3), since the repo has a profiler and anything from this table is a guess until measured.

### Gaps
- No benchmark of terrain shaders, CDLOD, clipmaps or grass on Intel HD, UHD or Iris was found. All ms figures would be invented, so none are given. Testing on a real HD 4000, UHD 620 or similar is the only reliable way to get these.
- Exact GL 3.3 support per Intel generation was not verified from an Intel source.
- Tessellation and compute are not in the budget above. Both are GL 4 features and remain optional for higher tiers only.
