# OGRE as the benchmark for moonBASIC

Checked on 2026-11-03 against OGRE `master` (latest release v14.6.3) and the moonBASIC working tree. Manual pages were read through a summarising fetch tool; repository facts come from raw files and the GitHub API. Local paths are relative to the moonBASIC repo root.

## What does current OGRE 1.x provide?

### Takeaway
OGRE 1.x (now numbered 14.x) is a large, modular C++ renderer with many back ends, scriptable materials and compositors, and generated shaders. HLMS is not part of it: RTSS fills that role. Since 14.6 it also has Forward+, so that gap now applies to the same engine generation as moonBASIC's baseline.

### Cited Findings

**Release and licence**
- The latest release is v14.6.3, published 2026-11-01, following 14.6.2 and 14.6.1 at about weekly intervals, so the project is active — [GitHub releases API](https://api.github.com/repos/OGRECave/ogre/releases).
- The licence is MIT. OGRE describes itself as a modular renderer for custom engine development — [README](https://github.com/OGRECave/ogre/blob/master/README.md).

**Render systems**
- The repo contains `Direct3D9`, `Direct3D11`, `GL`, `GL3Plus`, `GLES2`, `Metal`, `Tiny` and `Vulkan` — [RenderSystems/](https://github.com/OGRECave/ogre/tree/master/RenderSystems).
- The project's own comparison lists Metal as experimental and Vulkan as beta — [What version to choose](https://www.ogre3d.org/about/what-version-to-choose).
- Vulkan on macOS through MoltenVK arrived in 14.6 — [Docs/14-Notes.md](https://github.com/OGRECave/ogre/blob/master/Docs/14-Notes.md).
- Platforms are Windows (including UWP), Linux, macOS, iOS, Android and HTML5 — [What version to choose](https://www.ogre3d.org/about/what-version-to-choose).
- **GL3Plus has the same baseline as moonBASIC.** It asserts `hasMinGLVersion(3, 3)` with the message "OpenGL 3.3 is not supported". Newer features are gated at runtime: tessellation on 4.0, program binary on 4.1 or `GL_ARB_get_program_binary`, compute and SSBO on 4.3 or the matching ARB extension — [OgreGL3PlusRenderSystem.cpp](https://github.com/OGRECave/ogre/blob/master/RenderSystems/GL3Plus/src/OgreGL3PlusRenderSystem.cpp) (lines 393, 407, 424, 315, 1390).

**Components and plug-ins**
- Components are `Bites`, `Bullet`, `Csharp`, `Java`, `MeshLodGenerator`, `Overlay`, `Paging`, `Property`, `Python`, `RTShaderSystem`, `Terrain` and `Volume` — [Components/](https://github.com/OGRECave/ogre/tree/master/Components).
- Plug-ins are `Assimp`, `BSPSceneManager`, `CgProgramManager`, `DotScene`, `EXRCodec`, `FreeImageCodec`, `GLSLang`, `OctreeSceneManager`, `OctreeZone`, `PCZSceneManager`, `ParticleFX`, `RsImageCodec` and `STBICodec` — [PlugIns/](https://github.com/OGRECave/ogre/tree/master/PlugIns).

**Scene managers**
- The default SceneManager suits small and moderate scenes. `OctreeSceneManager` targets large levels — [The Core Objects](https://ogrecave.github.io/ogre/api/latest/_the-_core-_objects.html).
- BSP and portal-connected-zone managers exist as plug-ins — [PlugIns/](https://github.com/OGRECave/ogre/tree/master/PlugIns).

**Materials**
- Techniques are listed in order of preference. Each is checked for support when the material compiles; if none is supported the material renders blank white — [Material Scripts](https://ogrecave.github.io/ogre/api/latest/_material-_scripts.html).
- A technique has 1 to 16 passes. Passes hold texture units, shader references and RTSS instructions — [Material Scripts](https://ogrecave.github.io/ogre/api/latest/_material-_scripts.html).
- `scheme` gives top-level switching, usually set per viewport. `lod_index` plus `lod_values` select by distance or pixel count. `gpu_vendor_rule` and `gpu_device_rule` can only disable a technique — [Material Scripts](https://ogrecave.github.io/ogre/api/latest/_material-_scripts.html).

**RTSS**
- It assembles vertex and fragment programs from "SubRenderState" modules placed in stages — [RTSS manual](https://ogrecave.github.io/ogre/api/latest/rtss.html).
- Options include Cook-Torrance `metal_roughness`, normal and parallax maps, `shadow_mapping` with pcf4 or pcf16, `integrated_pssm4`, IBL, G-buffer output, triplanar texturing, hardware skinning (linear or dual quaternion), layered blend and instancing — [RTSS manual](https://ogrecave.github.io/ogre/api/latest/rtss.html).
- Generated techniques can be created on demand from the default technique through a `SchemeNotFound` listener. Shaders regenerate when light or fog settings change, and `setLightCount` fixes the count to avoid recompiling — [RTSS manual](https://ogrecave.github.io/ogre/api/latest/rtss.html).
- Clustered light culling (Forward+) is available since 14.6 as `SRS_CLUSTERED_LIGHT_CULLING` — [Docs/14-Notes.md](https://github.com/OGRECave/ogre/blob/master/Docs/14-Notes.md).

**Compositor**
- A compositor holds techniques, which hold texture declarations and `target` sections, which hold passes. Pass types are `clear`, `stencil`, `render_scene`, `render_quad`, `compute` and `render_custom` — [Compositor Scripts](https://ogrecave.github.io/ogre/api/latest/_compositor-_scripts.html).
- `input previous` chains effects. Texture sizes can follow the target (`target_width_scaled`). MRT, `pooled`, and local, chain and global scopes are supported. `render_scene` takes `first_render_queue`, `last_render_queue` and `material_scheme`. `only_initial` runs a target once — [Compositor Scripts](https://ogrecave.github.io/ogre/api/latest/_compositor-_scripts.html).
- A technique is skipped if the hardware lacks enough MRT support or a referenced material has no supported technique — [Compositor Scripts](https://ogrecave.github.io/ogre/api/latest/_compositor-_scripts.html).

**Shadows**
- Stencil additive and modulative; texture additive and modulative; integrated texture shadows — [Shadows](https://ogrecave.github.io/ogre/api/latest/_shadows.html).
- Camera setups are uniform, uniform focused, LiSPSM and plane optimal, and can be subclassed — [Shadows](https://ogrecave.github.io/ogre/api/latest/_shadows.html).
- That page does not mention PSSM. PSSM lives in RTSS as `integrated_pssm4` (single light) — [RTSS manual](https://ogrecave.github.io/ogre/api/latest/rtss.html) — with a `CSMShadows` sample — [Samples/](https://github.com/OGRECave/ogre/tree/master/Samples).
- The manual says texture shadows for point lights work only off-camera, approximated as spotlights — [Shadows](https://ogrecave.github.io/ogre/api/latest/_shadows.html).

**Instancing**
- The manual covers StaticGeometry, an explicit InstanceManager (software or hardware) and auto-instancing. Auto-instancing needs no code changes but does not support animation. Hardware instancing needs D3D9 or GL3 and is not on GLES2 — [Instancing](https://ogrecave.github.io/ogre/api/latest/_instancing.html).

**Terrain, Paging, Volume**
- Terrain includes `TerrainGroup`, `TerrainLodManager`, `TerrainAutoUpdateLod`, `TerrainLayerBlendMap`, `TerrainMaterialGeneratorA` and paging integration — [Components/Terrain/include](https://github.com/OGRECave/ogre/tree/master/Components/Terrain/include).
- The material generator supports per-layer normal and parallax mapping, on by default — [OgreTerrainMaterialGeneratorA.h](https://github.com/OGRECave/ogre/blob/master/Components/Terrain/include/OgreTerrainMaterialGeneratorA.h).
- Paging provides `PagedWorld`, `PagedWorldSection`, and `Grid2DPageStrategy` / `Grid3DPageStrategy` — [Components/Paging/include](https://github.com/OGRECave/ogre/tree/master/Components/Paging/include).
- Volume provides CSG and triplanar texturing — [README](https://github.com/OGRECave/ogre/blob/master/README.md).

**Samples that show the feature spread**
- `BSP`, `Compute`, `DualQuaternion`, `FacialAnimation`, `GaussianSplatting`, `Grass`, `Hair`, `Island`, `LightShafts`, `MeshLod`, `MeshShaders`, `NewInstancing`, `PBR`, `PNTrianglesTessellation`, `RectLight`, `SSAO`, `Smoke`, `Tessellation`, `TextureArray`, `VolumeCSG`, `VolumeTerrain` — [Samples/Simple/include](https://github.com/OGRECave/ogre/tree/master/Samples/Simple/include).
- Also `DeferredShading`, `EndlessWorld`, `Isosurf`, `OceanDemo`, `ParticleGS`, `TerrainTessellation`, `VolumeTex` and `Water` — [Samples/](https://github.com/OGRECave/ogre/tree/master/Samples).
- Mesh shaders on GL3Plus use `GL_NV_mesh_shader` — [Docs/14-Notes.md](https://github.com/OGRECave/ogre/blob/master/Docs/14-Notes.md).

**HLMS**
- Of 3,131 paths in the `master` tree, the only name matching "hlms" is a thumbnail, `Samples/Media/thumbnails/thumb_hlms.png` (a filename search, not a search of file contents) — [git tree API](https://api.github.com/repos/OGRECave/ogre/git/trees/master?recursive=1).

### Inferences
- "HLMS" on the assignment's list belongs to Ogre-Next. Any plan to match HLMS is a choice to copy Ogre-Next's design, not to match OGRE 1.x.
- The GL3Plus capability pattern (3.3 floor, optional 4.x) is the policy in `docs/STATUS.md` already. moonBASIC's GL policy needs no change to be as capable as OGRE on the same GL back end.
- Many sample names (`Hair`, `Smoke`, `GaussianSplatting`) are demos of shaders on top of the engine, not engine features. Counting them as OGRE features would overstate the gap.

### Gaps
- The manual's terrain page could not be fetched (two guessed URLs returned 404), so terrain facts come from headers and are not a full account of LOD or editing.
- I could not find the clustered culling source file under the name I tried, so its GL requirement and light limits are unknown. It may or may not work on plain GL 3.3.
- The manual's instancing page does not name the ShaderBased, VTF, HW Basic, HW VTF and HW VTF LUT techniques. I recall them from the `InstanceManager` API but did not check the header in this pass.
- `Tiny` is listed by directory name only. I recall it as a software rasteriser, unconfirmed here.

## How do OGRE 1.x and Ogre-Next differ, and which is the fair benchmark?

### Takeaway
OGRE 1.x is the fair benchmark: same GL 3.3 floor, same kind of low-end hardware target, and a feature list moonBASIC can measure against. Ogre-Next is a source of two design ideas (hash-keyed shader templates and state blocks), not a target.

### Cited Findings
- The project calls them "sister projects that share a heritage but not an API". 1.x is for maximum platform reach; Next is for modern hardware — [What version to choose](https://www.ogre3d.org/about/what-version-to-choose).
- Next has D3D11, GL 3+, Metal and Vulkan (experimental), with components ParticleFX, Terra and MeshLOD. It has no fixed-function path or stencil shadows, and offers global illumination, SSR and screen-space decals — [What version to choose](https://www.ogre3d.org/about/what-version-to-choose).
- Next's README lists GL 3.3+ among back ends, plus threaded node updates and culling, SIMD AoSoA layouts, Forward Clustered, background texture streaming, and VCT, Instant Radiosity and PCC hybrid GI. It also says it is a renderer without networking, sound or physics — [Ogre-Next README](https://github.com/OGRECave/ogre-next/blob/master/README.md).
- The latest GitHub release of Next is v3.0.0 (2024-10-15), while its online API docs are labelled 4.0 unstable — [releases API](https://api.github.com/repos/OGRECave/ogre-next/releases); [API docs](https://ogrecave.github.io/ogre-next/api/latest/).
- HLMS splits a material into a mutable datablock and immutable shared macroblocks (depth, cull), blendblocks and samplerblocks. Shader templates use `@property`, `@piece` and `@foreach`. Property sets hash to a key: same hash, same shader. A pass hash merges with the renderable hash at lookup, and misses generate and compile — [HLMS manual](https://ogrecave.github.io/ogre-next/api/latest/hlms.html).
- The same page calls old Materials "slow. Very slow." but keeps them for postprocessing and quick shader iteration, where few permutations exist — [HLMS manual](https://ogrecave.github.io/ogre-next/api/latest/hlms.html).
- AZDO work includes a 64-bit render-queue sort hash, a parallel `addRenderable` in FAST mode, a VaoManager that pools GPU memory, triple-buffered dynamic buffers with frame-wide fences, and staging buffers for partial updates — [AZDO manual](https://ogrecave.github.io/ogre-next/api/latest/azdo.html).
- Compute appears in Next tutorials (UAV textures and buffers), and Terra uses compute shaders to generate terrain shadows each frame. A Forward3D sample shows many non-shadow-casting lights — [Samples](https://ogrecave.github.io/ogre-next/api/latest/_samples.html).
- The FAQ warns that compute-shader features "often require a powerful modern GPU" — [Ogre 2.1 FAQ](https://wiki.ogre3d.org/Ogre+2.1+FAQ).

### Inferences
- Next's headline GI (VCT) and compute terrain shadows are out of reach for a 2016 Intel iGPU on GL 3.3, so holding moonBASIC to them would be unfair.
- moonBASIC's auto batching is closer to Next's approach of sorted, grouped draws than to OGRE 1.x's opt-in instance managers.
- A useful phrasing of the user's goal: "as capable as OGRE 1.x on its GL3Plus back end, plus a full game layer".

### Gaps
- I found no primary statement of the exact GL extensions Next needs on 3.3. My recollection that the GL3+ back end expects `GL_ARB_base_instance` and similar is unverified, so the claim "Next runs on all 3.3 hardware" should not be repeated.
- Which AZDO paths need persistent mapping is not stated in the AZDO page I read.

## Which OGRE designs matter most for moonBASIC?

### Takeaway
Copy, in order: a compositor-style post chain with user render targets; a resource lifecycle with background prepare; on-disk shader program cache; named render queues; per-material technique fallback tied to tiers; and schemes. Copy OGRE's selection rules but not its heavy per-pass state model.

### Cited Findings

**1. Compositor chain (highest value)**
- moonBASIC has a "fixed built-in stack via `EnablePostFX`" and "no custom passes", and render-to-texture is "Missing as user command" — `docs/ROADMAP.md` lines 1016–1017.
- A search of registered commands in `internal/runtime/*.go` found no `rendertex`, `rendertarget` or similar name.
- OGRE's primitives are `input previous`, size-relative textures, pooled textures, `render_quad` and `render_scene` with queue ranges — [Compositor Scripts](https://ogrecave.github.io/ogre/api/latest/_compositor-_scripts.html).

**2. Resource groups and background loading**
- Groups load and unload as a unit, and locations include folders and zips. States run declared, unloaded, prepared (data in memory), then loaded (on GPU). A manager can enforce a memory budget using priority and LRU — [Resource Management](https://ogrecave.github.io/ogre/api/latest/_resource-_management.html).
- `ResourceGroupListener` reports loading progress — [Resource Management](https://ogrecave.github.io/ogre/api/latest/_resource-_management.html).
- With the default thread config, workers cannot touch the RenderSystem: "prepare() runs Resource::prepare in the background" and the load happens on the main thread — [OgreResourceBackgroundQueue.h](https://github.com/OGRECave/ogre/blob/master/OgreMain/include/OgreResourceBackgroundQueue.h).
- moonBASIC has the same rule: "Workers must not touch GL" (`docs/STATUS.md`, Jobs row). It already has `openpak` (`internal/runtime/commands_fx.go:495`). The only `*async` command found is `physicsasync` (`internal/runtime/commands_scale.go:124`), so no script-level async asset load was found.

**3. Shader program cache**
- OGRE has a microcode cache: `GpuProgramManager` saves and restores program binaries — [OgreGpuProgramManager.h](https://github.com/OGRECave/ogre/blob/master/OgreMain/include/OgreGpuProgramManager.h).
- moonBASIC is "in-memory G3N programs / no binary disk cache" — `docs/ROADMAP.md:1263`. A search found no `ProgramBinary` in `internal/runtime`.
- HLMS's hash-keyed lookup is the simplest generation scheme — [HLMS manual](https://ogrecave.github.io/ogre-next/api/latest/hlms.html).

**4. Render queues**
- OGRE defines named groups from `RENDER_QUEUE_BACKGROUND = 0` through `RENDER_QUEUE_MAIN = 50`, `SKIES_LATE = 90`, `TRANSPARENTS = 95` and `OVERLAY = 100` — [OgreRenderQueue.h](https://github.com/OGRECave/ogre/blob/master/OgreMain/include/OgreRenderQueue.h).
- moonBASIC scatters magic numbers: 110 (`atmosphere.go:248`), 100 (`skybox.go:105`, `water.go:482`), 90 (`weather_lightning.go:125`), 85 (`terrain_clouds.go:171`), 80 (`particles.go:195`). Entities with non-zero order skip batching (`batch.go:443`). No script command for order was found.

**5. Material techniques with fallback**
- OGRE selects per material at compile time from real capability checks — [Material Scripts](https://ogrecave.github.io/ogre/api/latest/_material-_scripts.html).
- moonBASIC's choice is global. `autoTier` guesses from the GL renderer string, and a frame-time watch corrects downward; a tier sets shadow size, reflections, terrain radius, scatter, AO and MSAA (`internal/runtime/quality.go:24-46, 60-70`).
- Mesh detail is automatic: shared meshes of 256+ triangles get up to two simpler versions, chosen by at most one pixel of error (`docs/GRAPHICS.md:74`). No material LOD command was found.

**6. Schemes**
- A viewport's material scheme swaps all techniques at once, and a compositor pass can override it — [Material Scripts](https://ogrecave.github.io/ogre/api/latest/_material-_scripts.html); [Compositor Scripts](https://ogrecave.github.io/ogre/api/latest/_compositor-_scripts.html).

### Inferences
- **Compositor, minimum version:** `CreateRenderTexture`, `CameraToTexture`, and `AddPostPass shader$, scale#, inputs$`, plus a small text file format that hot reloads like `LoadShader`. Built-in bloom and tonemap stay as named passes. This gives scripts OGRE's chain without its full grammar. Multi-pass custom effects also unlock depth of field, motion blur and light shafts for users.
- **Resources:** `LoadGroup "level2"`, `LoadProgress()` and `GroupLoaded()` map to Blitz style well. Decoding on a worker and uploading a bounded number of items per frame fits the existing Jobs rule and would cut hitching.
- **Cache:** on 2016 Intel iGPUs, first-frame shader compile is likely the most visible stall, so disk caching pays off directly. It must fall back to source when the binary is rejected (driver update) and be gated like OGRE gates it (4.1 or the ARB extension).
- **Queues:** name them in one Go file first (no behaviour change), then expose `SetEntityQueue`. Decide up front what happens to batching, since queue changes currently drop an entity from batching.
- **Techniques:** do not copy multi-pass techniques. Copy the rule: an ordered list of variants, the first supported one wins, keyed by tier and distance. For example, a water or foliage material could offer "full", "no-refraction" and "unlit" variants. That turns `tierSettings` from a global switch into per-material decisions. Also replace the renderer-string check with a capability check where a real feature decides it.
- **Schemes:** most valuable as an internal tool (depth-only, retro, debug views, reflection pass with cheaper shading). Reflection pass with a cheap scheme would directly help the "low" tier.
- **State model:** Next's own criticism of Materials says not to adopt per-pass mutable state. moonBASIC should keep immutable shared state keyed by hash.
- **Do not copy:** stencil volumes, FFP pass-splitting, BSP or portal managers, full `.material` script compatibility.

### Gaps
- `internal/runtime/stream.go` and terrain chunk loading may already prepare data on workers. I checked registered commands, not those internals.
- I did not measure moonBASIC's shader compile time, so the stall claim above is reasoning, not data.
- Whether G3N recompiles when light count changes (the problem `setLightCount` solves in RTSS) was not checked.

## Where is moonBASIC ahead, equal or behind?

### Takeaway
moonBASIC is well ahead as a game engine and on outdoor content. It is about equal on PBR, shadows and instancing. It is behind on renderer extensibility: user render targets, custom post chains, per-material fallback, queues, Forward+, decals, anisotropy and back-end reach.

### Cited Findings
Sources: OGRE cells cite the links given in the first section unless linked here. moonBASIC cells cite `docs/STATUS.md` and `docs/ROADMAP.md`. "Grep" means I searched registered command names in `internal/runtime/*.go`.

| Feature | OGRE 1.x (14.6) | moonBASIC | Gap | Worth doing on GL 3.3 |
| --- | --- | --- | --- | --- |
| Render back ends | D3D9/11, GL, GL3+, GLES2, Metal, Vulkan | GL 3.3 only, by policy | Large | No; only with the planned renderer replacement |
| User render-to-texture | Compositor targets, cubic, 2D array | Internal FBOs only (ROADMAP 1016; grep) | Large | **Yes, first** |
| Custom post chain | Compositor scripts | Fixed stack (ROADMAP 1017) | Large | **Yes, first** |
| Built-in post effects | Compositor sample, SSAO sample; users write the rest | Bloom, tonemap, FXAA, LUT, outlines, grain, AO, auto exposure | moonBASIC ahead out of the box | Keep; add DoF and motion blur once the chain exists |
| Material fallback by hardware | Ordered techniques, vendor/device rules | Global tier from renderer string (quality.go) | Medium | **Yes**, simplified |
| Material schemes | Per viewport, per compositor pass | None found | Medium | Yes, internal first |
| Material LOD | `lod_index`, `lod_values`, 4 strategies | None found | Small | Yes, with fallback lists |
| Mesh LOD | MeshLodGenerator, `MeshLod` sample | Automatic, error-based, two levels, also for shadow tiles (GRAPHICS.md:74) | Equal; moonBASIC is more automatic | Add a script override only |
| Shader generation | RTSS modules; HLMS not in 1.x | G3N defines plus own shaders | Medium | Selectively; hash-keyed variants, not full RTSS |
| Shader binary cache | Microcode cache | None (ROADMAP 1263) | Medium | **Yes**, gated on 4.1 or ARB |
| Shader hot reload | Not checked | `LoadShader` reloads on save; bad compile keeps the old program | Unknown; moonBASIC's is solid | Keep |
| Render queues | Named groups 0–105, compositor ranges | Internal magic numbers (see above) | Medium | **Yes**, cheap |
| Forward+ | `SRS_CLUSTERED_LIGHT_CULLING`, since 14.6 | Missing (ROADMAP 992); "every light in the scene is shaded" (LIGHTING.md:133) | Large for many-light scenes | Likely, via UBO or texture buffer; verify on 2016 iGPU |
| Deferred | Sample and RTSS G-buffer output | "Not on G3N" | Medium | No; MRT from the chain covers most uses |
| Directional shadows | Texture, LiSPSM, PSSM4; pcf4/pcf16 via RTSS | CSM with PCF, PCSS, EVSM, MSM | moonBASIC ahead on filters | Keep |
| Point and spot shadows | Manual: point lights approximated, off-camera only | Atlas; two point and two spot (SHADOWS.md:52, 62–63) | Equal or ahead | Keep |
| Stencil shadows | Yes; CPU-computed volumes, hard edges | None | Small | No |
| Instancing | Static geometry, software and hardware managers, auto | GPU instancing, CPU merge, automatic batching incl. shadow pass | About equal; automatic in moonBASIC | Skinned instancing only if crowds need it |
| Particles | ParticleFX plug-in, `ParticleGS` sample | Transform-feedback GPU emitter; no soft particles or sorting | Probably ahead on count; behind on softness | **Soft particles yes** |
| Terrain | LOD manager, groups, blend maps, per-layer normal and parallax | Heightmaps, splat, chunk LOD, runtime sculpt and paint, erosion | About equal; moonBASIC ahead on tools | Parallax layers optional |
| Paging and streaming | Generic 2D and 3D page strategies | `SetPlayer` bubble, origin shift past 4 km | Different; moonBASIC is more turnkey | Keep |
| Foliage and trees | Not in engine; `Grass` sample, PagedGeometry add-on | `TerrainFoliage`, `TerrainTrees`, impostors, wind | moonBASIC ahead | Keep |
| Water and ocean | `Water` and `OceanDemo` samples only | Gerstner, scenic, FFT tile, caustics, buoyancy | moonBASIC well ahead | Keep |
| Sky and atmosphere | `SkyBox`, `SkyDome`, `SkyPlane` samples | Bruneton-style LUT, weather, clouds | moonBASIC ahead | Keep |
| GI and reflections | IBL through RTSS; dynamic cubes via compositor | Light probes capturing one face per frame, IBL | About equal | Keep |
| Volume and CSG | Component and two samples | None (grep) | Medium | Low priority; game-specific |
| Decals | Not in the 1.x list; screen-space in Next | None (grep) | Medium | **Yes**, simple projected decals |
| Anisotropic filtering | Sampler attribute (HLMS page describes it for Next; 1.x not checked here) | Missing (ROADMAP 1001; grep) | Small but visible | **Yes**, very cheap; check extension |
| Order-independent transparency | Listed for 1.x | Not listed | Small | Later; test cost first |
| Tessellation, compute, mesh shaders | Feature gated; mesh via NV ext | Compute and tess optional; no mesh shaders | Small | Keep optional; never rely on it |
| Dual quaternion skinning | RTSS option | Not listed | Small | Optional |
| Light shafts | Sample | Missing (ROADMAP 1013) | Small | Yes, after the chain |
| 2D and UI | Overlay, ImGui | Ebiten 2D stack, ImGui, G3N widgets | moonBASIC ahead | Keep |
| Asset import | Assimp and DotScene plug-ins, mesh tools | glTF, OBJ, DAE, FBX, STL, PLY; `moon build` paks | About equal; moonBASIC simpler | Keep |
| Physics, audio, nav, net, AI | Bullet component only | Jolt, Chipmunk, Oto, Detour, net, BT and GOAP | moonBASIC far ahead; outside OGRE's scope | Keep |
| Language and tooling | C++, Python, C#, Java bindings | Own language, IDE, debugger, profiler | moonBASIC ahead for its users | Keep |
| Platforms | Adds iOS, Android, HTML5 | Windows, Linux, macOS | Medium | Not a rendering task |

### Inferences
- Roughly eight small-to-medium items close most of the honest gap on GL 3.3: render targets, post chain, material fallback, program cache, queues, soft particles, decals and anisotropy. Forward+ is the one large item.
- Back-end reach cannot be closed on G3N. "As powerful as OGRE" should explicitly exclude it until the renderer is replaced, as `docs/STATUS.md` already says.
- For an ordinary user, moonBASIC already looks stronger out of the box, because OGRE leaves water, sky, foliage and game systems to the developer.

### Gaps
- No benchmarks compare the two engines. Performance claims here are about features only.
- OGRE's hardware occlusion queries and moonBASIC's lack of occlusion culling (grep found only map and 2D occluder commands) were not checked against an OGRE source, so they are left out of the table.
- OGRE's ParticleFX CPU-versus-GPU behaviour was not checked; "probably ahead" is my reasoning.

## What does OGRE not do that the user may assume?

### Takeaway
OGRE is a renderer. It has no audio, navigation, networking or game framework, and physics comes only as a Bullet integration. It has no tree system. Ocean and water exist only as samples, so "no ocean" is true of the engine but not of the repo.

### Cited Findings
- OGRE describes itself as a renderer for custom engine development, with Bullet as an integration — [README](https://github.com/OGRECave/ogre/blob/master/README.md).
- Next says directly it is a rendering engine without networking, sound or physics — [Ogre-Next README](https://github.com/OGRECave/ogre-next/blob/master/README.md).
- The component list has no audio, navigation, networking or AI — [Components/](https://github.com/OGRECave/ogre/tree/master/Components).
- A search of the full file tree found no tree-system, SpeedTree, SkyX or Hydrax paths — [git tree API](https://api.github.com/repos/OGRECave/ogre/git/trees/master?recursive=1).
- `Sample_Ocean` is "An example demonstrating ocean rendering using shaders" — [OceanDemo.cpp](https://github.com/OGRECave/ogre/blob/master/Samples/OceanDemo/src/OceanDemo.cpp). `Water` is "A demo of a simple water effect" — [Water.h](https://github.com/OGRECave/ogre/blob/master/Samples/Water/include/Water.h).
- `Grass` is a sample showing StaticGeometry baking — [Grass.h](https://github.com/OGRECave/ogre/blob/master/Samples/Simple/include/Grass.h).
- Forests come from a separate add-on, PagedGeometry (batched near trees, impostors far, grass loader, zlib licence), under the OGRECave docs — [PagedGeometry](https://ogrecave.github.io/ogre-pagedgeometry/).

### Inferences
- If the user pictures OGRE as producing beautiful outdoor scenes by default, that picture comes from add-ons and showcase projects. moonBASIC's built-in foliage, sky and water already exceed what OGRE ships as engine features.
- "As powerful as OGRE" is really about extensibility: how much a developer can change the pipeline without touching engine source.
- OGRE has no node-based shader graph in the listed components, so moonBASIC's missing node editor (`docs/ROADMAP.md:1269`) is not an OGRE gap.

### Gaps
- Hydrax and SkyX were not found in the search results; my memory is that they are third-party Ogre add-ons, and their status today is unverified.
- I did not read the ocean sample's shaders, so I cannot compare its quality with moonBASIC's water beyond the sample being a demo.

## Would using OGRE directly from Go be sensible?

### Takeaway
No. The licence and binary size are fine, but it would mean rewriting moonBASIC's renderer and all its custom shaders behind a C++ binding nobody maintains. Copy the designs. Reconsider OGRE only as one candidate when the planned renderer replacement happens, and then through a narrow hand-written C layer.

### Cited Findings
- MIT licence, no obstacle — [README](https://github.com/OGRECave/ogre/blob/master/README.md).
- Official bindings are C#, Java and Python, generated with SWIG — [Components/](https://github.com/OGRECave/ogre/tree/master/Components); [Wikipedia: OGRE](https://en.wikipedia.org/wiki/OGRE). Go is not among them.
- SWIG supports Go. cgo has "no convenient way to call C++ code" directly — [SWIG Go docs](https://swig.org/Doc4.4/Go.html).
- `gogre3d` wraps OGRE through the llcoi C layer; last pushed 2012 — [repo](https://github.com/galaktor/gogre3d), [API](https://api.github.com/repos/galaktor/gogre3d).
- `go-ogre3d` also uses llcoi. The holocronweaver fork was last pushed 2014-01-22 (0 stars) and its parent 2014-02-08 (4 stars); both descend from `gogre3d` (23 stars) — [repo](https://github.com/holocronweaver/go-ogre3d), [API](https://api.github.com/repos/holocronweaver/go-ogre3d).
- On C bindings for Next, the maintainer noted SWIG does not generate them and has no plans to add them — [ogre-next issue 428](https://github.com/OGRECave/ogre-next/issues/428).
- Size proxy: `ogre_python` 14.6.3 wheels are 10–11 MB on Windows, 15–16 MB on macOS ARM and 26 MB on Linux (compressed, with bindings) — [PyPI](https://pypi.org/pypi/ogre-python/json).
- moonBASIC's current local `moon.exe` is 83,858,944 bytes (about 80 MB), dated Nov 1 — local file listing. It already depends on cgo for ImGui, Flecs, GLFW and Jolt (`docs/STATUS.md`, Build; `go.mod`).
- Renderer work in moonBASIC is deep in G3N: shaders `mbphysical`, `mbterrain`, `mbwater`, `bsshadow` and own batch, shadow, post and particle code (`internal/runtime/*`; `docs/STATUS.md`). `docs/STATUS.md` says "a later renderer replaces this one" and rules out wrapping Vulkan on G3N.

### Inferences
- **Cost:** two scene graphs cannot share a GL pipeline cleanly, so adopting OGRE means replacing G3N, not adding to it. The custom water, atmosphere, terrain, retro and shadow shaders would need porting into RTSS or OGRE material terms. That is most of what makes moonBASIC look good.
- **Benefit:** the only gains that cannot be reached on GL 3.3 are extra back ends and mobile or web targets. Everything else in the table's "yes" rows is cheaper to build directly.
- **Size:** not the blocker. OGRE would add little compared with an 80 MB binary.
- **Build:** adds a C++ build for three platforms and a binding layer to maintain, on top of an already-CGO build. Cross-OS `moon build` already needs a CGO toolchain, so this worsens an existing pain rather than creating a new one.
- **If reconsidered later:** use a hand-written C shim over the small set of calls moonBASIC needs, not SWIG over the full API, and compare against writing a thin renderer directly on Vulkan or Metal. Run this comparison as a spike and do not schedule it now.

### Gaps
- I did not build OGRE with cgo, so link times, static-link size and plug-in loading on all three platforms are unmeasured.
- The Python wheel is an imperfect proxy for a static C++ link size.
- The `moon.exe` size was not checked for stripped or unstripped state, and its share of G3N is unknown.
