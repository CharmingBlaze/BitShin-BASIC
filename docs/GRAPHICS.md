# Graphics (OpenGL + G3N)

moonBASIC stays on **G3N + go-gl OpenGL**. There is no bgfx, Vulkan, or wgpu path.

## Required baseline: OpenGL 3.3

The ship path is **OpenGL 3.3 core**. Everything needed to run a game must work here:

- G3N window, Phong / `mbshadow` ([docs/LIGHTING.md](LIGHTING.md)), directional CSM + point/spot maps
- Particles, skybox, weather (atmosphere LUT + clouds + lightning), fog
- PBR (`mbphysical`) as GLSL that G3N compiles on 3.3
- Terrain and water on **regular meshes** (Gerstner in the vertex shader)
- Platform 64 and the other 3D examples

If the driver cannot create a 3.3 core context, `Graphics3D` fails the same way G3N already fails. We do **not** require 4.5 (or 4.3) to open the window.

## How the context is created

1. `Graphics3D` calls `startG3N` → G3N `window.Init`.
2. G3N’s GLFW hints (do not change these to 4.5):

```
glfw.WindowHint(ContextVersionMajor, 3)
glfw.WindowHint(ContextVersionMinor, 3)
glfw.WindowHint(OpenGLProfile, OpenGLCoreProfile)
```

On macOS, G3N also sets forward-compatible (required there).

3. Extra windows (`CreateWindow`) use the **same 3.3 core hints** and share the G3N context (`internal/runtime/extra_window.go`).
4. After the context exists, `detectModernGL()` reads `GL_MAJOR_VERSION` / `GL_MINOR_VERSION`, renderer string, and extensions (`glGetStringi`). It never raises the context version.

You will see one line like:

```
glmodern: OpenGL 3.3 (…renderer…) instance=1 geom=1 ubo=1 compute=0 ssbo=0 tess=0
```

## Optional 4.x (feature-detect)

These use raw go-gl **after** G3N state is saved (same idea as `saveShadowGL` in `shadows.go`). Restore program / VAO / buffers / viewport before returning to G3N.

| Feature | Min GL / extension | If missing |
| --- | --- | --- |
| GPU instancing | 3.1 / `GL_ARB_draw_instanced` | CPU-merged mesh (`BatchInstances`) |
| Geometry shaders | 3.2 / `GL_ARB_geometry_shader4` | `CreateGeomPoints` returns **0**, one skip line |
| UBO | 3.1 / `GL_ARB_uniform_buffer_object` | CPU float mirror; `BindUniformBuffer` returns **0** |
| Tessellation | 4.0 / `GL_ARB_tessellation_shader` | `EnableTessellation` returns **0**; water/terrain keep dense mesh |
| Compute | 4.3 / `GL_ARB_compute_shader` | `CreateComputeShader` / `DispatchCompute` return **0** |
| SSBO | 4.3 / `GL_ARB_shader_storage_buffer_object` | `CreateStorageBuffer` keeps a CPU mirror; `BindStorageBuffer` returns **0** |

Commands **never crash** and **never `End`** the program on an old GPU. They return `0` and print one:

```
glmodern: compute skipped (need OpenGL 4.3 or GL_ARB_compute_shader; have 3.3.0 …)
```

The skip is printed once per feature.

## G3N vs wrappers

G3N already uses go-gl internally (`github.com/go-gl/gl`). moonBASIC adds wrappers only for what G3N does not expose (compute, SSBO, UBO bind, instanced draw, geom/tess programs).

Do not fight GLS blindly: `saveModernGL` / restore around raw calls, like shadows.

Depth shadows stay on a G3N `Program` + `gs.UseProgram`. Dummy shadow map when shadows are off (unchanged).

## Automatic batching

Copies need no special commands. Entities that share a geometry and have equal material settings are drawn as one instanced draw, in the scene pass and in every shadow tile, so `CopyEntity` in a loop is cheap. Each copy can still move, turn, scale, hide, and be parented on its own. A copy whose material differs in any way (color, shininess, texture, side, and so on) goes in another group.

These are drawn alone, as before: transparent materials, skinned and morphing meshes, meshes with several materials, swaying trees, custom shaders, terrain, water, and a mesh with no copies. Only `standard`, `bsshadow`, and `mbphysical` materials are grouped. Child meshes inside a loaded model are not entities and are not grouped. Casters with a cut-out texture are grouped in the scene pass but still drawn one by one into the shadow map.

A shared mesh of 256 triangles or more also gets up to two simpler versions, built once on a worker thread. Each view draws a copy with the simplest version whose error covers at most one pixel at its distance, and each directional shadow tile with the simplest whose error fits in one texel. Point and spot shadow tiles and orthographic cameras keep the full mesh. Pixels can change slightly: about 0.1% in the test scenes.

A mesh with no copies that has been still for 30 frames is merged with the others of the same material in its 512-unit area of the level, into one world-space mesh, which makes a level of thousands of different meshes a few hundred draws. Any change to it (moving, a new color, hiding it, gaining a copy) takes it out at once, and it can merge again once it is still. Meshes over 20,000 triangles, mirrored ones, and ones whose vertices can change (`CreateMesh` meshes, skinned meshes, cloth) stay on their own. A merged mesh is frustum-culled as a whole.

In every pass G3N no longer visits batched meshes. Plain shadow and fog uniforms are sent once per program per pass instead of once per mesh; textures are still bound per draw, because programs share texture units. In a water pass, terrain chunks wholly on the discarded side of the clip plane are skipped.

`SetAutoBatch 0` or `MOON_BATCH=0` turns all of this off. `StatsDraws()` counts a group or a merged mesh as one draw; `StatsBatched()` is how many entities groups draw.

| Benchmark, Radeon 780M, 1600 × 900 (island 1280 × 720), shadows on, high tier, dynamic resolution off or not yet present | Before | Now |
| --- | --- | --- |
| 5,000 cubes, one in ten spinning | 58.7 ms, 5,000 draws, dropped itself to the low tier | 6.6 ms, 1 draw |
| 50,000 cubes, one in ten spinning | 63 ms after instancing alone | 44-57 ms, 1 draw |
| City: 3,600 different meshes, 28,800 copies | 43.7 ms, 3,602 draws after instancing alone | 36.7 ms, 218 draws |
| 4,000 rocks of 8,000 triangles, most far away | 18.8 ms after instancing alone | 9.8 ms |
| `examples/island_orbit.mb` camera path, dynamic resolution off | 16.4 ms with all of this off, 9.0 ms of it render CPU | 16.5 ms, 6.3 ms of it render CPU (the GPU sets the frame time) |

Timings moved a lot from run to run on a shared machine. Above about 10,000 entities most CPU time is per-frame walks over every entity, chiefly checking each material for changes. There is no occlusion culling: cutting the city's far plane to 120 units, which removes more than occlusion could, did not make it faster, so it would not help these scenes.

Code: `internal/runtime/batch.go`, `batch_lod.go`, `batch_merge.go`. `MOON_CPUPROFILE=path` with `MOON_SMOKE_FRAMES` writes a Go CPU profile of those frames.

## Instancing

`CreateInstancedMesh` / `SetInstanceTransform` / `SetInstanceData` are the commands (also used by the large-world example).

- **GPU (default on 3.3):** one copy of the mesh + per-instance matrices, `glDrawElementsInstanced`. Color pass after `rend.Render`. Shadow pass uses a small instanced depth program.
- **CPU fallback:** merge all instance triangles into one G3N mesh (original large-world path).

`EnableGPUInstances(0)` forces the merge path.

## Geometry / tessellation

- Geom: points expanded to camera-facing quads (`CreateGeomPoints`).
- Tess: optional compile of a tiny tess program. Terrain/water **do not** require it. If tess shaders fail, we skip; the dense mesh stays.

## Compute demo

`CreateComputeShader("")` compiles a 430 sine-fill. Bind an SSBO and `DispatchCompute`. On 3.3 this returns 0.

## go-glslang / SPIR-V

Package `internal/glslang`:

- Always: pure-Go validate (`#version`, braces, stage tokens).
- If `Graphics3D` is up: also `glCompileShader` (3.3 path).
- SPIR-V: only if `glslangValidator` or `glslang` is on `PATH`. **Vulkan is not required.** Native Khronos C++ is **not** shipped (Windows CGO pain). `-tags glslang` is reserved for a future binding.

```
moon shader myshader.vert vert
CompileShader(src$, "vert")
```

## Gonum vs math32

| Library | Role |
| --- | --- |
| G3N `math32` | Scene graph, cameras, meshes, every G3N API |
| `gonum.org/v1/gonum` via `internal/mathx` | FBM, IBL SH, crowd separation, job-system batch 4×4 |

Do **not** replace `math32.Vector3` / `Matrix4` on nodes. Convert at the boundary (`mathx.FromVec3` / `ToVec3` / `FromMat4` / `ToMat4`).

Used today: probe SH (`SHEval`), crowd push-apart, `NoiseFBM`, `JobXform`.

## One-call look

`SetLook "modern"` after `CreateCamera` gives a finished picture: tone mapping, bloom, FXAA, ambient occlusion, image-based light, sun shadows, and fog that fades into the sky's horizon. It adds a default sky and a sun only if the scene has none, and leaves your fog alone if you set one. Anything set afterwards wins. The low tier skips the post effects. `SetLook "off"` turns them off.

## A world with no art files

Quality tiers can also be named by what they are for: `SetGraphicsQuality("classic")` is low, for older hardware, `"enhanced"` is medium, and `"advanced"` is high.

Terrain, forest and grass can all be generated, so a game can start with no assets:

```
SetLook "modern"
hm = GenerateHeightmapPreset("mountain-island", 256, 256, seed)
HydraulicErodeHeightmap(hm, 3200, 24)
ground = CreateTerrainFromHeightmap(hm, 800, 800, 160)
ApplyTerrainSplat(ground)
BakeTerrain(ground)
SetTerrainClearings(ground, 1)
TerrainTrees(ground, CreateTree("pine", 1), 0.14)
TerrainTrees(ground, CreateTree("oak", 3), 0.06)
SetTerrainGrass(ground, 1)
```

`examples/island_orbit.mb` is this with water and sky, and `examples/valley.mb` is a larger scene with depth: a lake, forest, and mountains fading into haze on a 512 grid, with about 0.6 s of setup and first frame on a warm shader cache, and 13.9 / 7.8 / 4.0 ms frames circling above the shore. Costs on a Radeon 780M at 1280x720 with scaling off, in frame milliseconds for high / medium / low: about 9.4 / 7.1 / 4.6 circling the island, 14.8 / 9.5 / 6.8 standing in the forest.

`BakeTerrain` stores normals with cracks and ledges on steep rock, the real horizon around each point, and where water gathers, in one map. It took 51 ms on the 256 grid above. Rock and grass are still chosen from the ground's smooth shape, since using the detailed normal drew rock in scars across meadows.

Three things keep it cheap. Oak crowns are several lobes, each a dark core inside leaf cards that the shader cuts into leaves from noise, so there is no leaf texture and fewer triangles than solid lumps. Each kind has a hand-built far model, and detail is chosen from the camera even when `SetStreamOrigin` fixes the streaming point. Near trees cast their shadow from the far model, because a soft shadow cannot show the difference and four cascades redraw every caster.

## Lighting

`examples/lighting.mb` uses everything together: a sun with four shadow cascades and PCF filtering, point and spot lights with their own shadows, metal-roughness PBR materials lit by the sky (`SetIBL`, with a prefiltered sky cube, and a plain hemisphere when it is off), ambient occlusion, and contact shadows. On the linear path a light of intensity 1 lights a PBR surface as brightly as a non-PBR one. Lights with `SetLightFalloff "physical"` keep their candela values, so about 15 is a lamp a few metres away and about 40 a spot at the same distance.

## Styles

`SetStyle` picks a game's look, and `examples/styles.mb` shows them on one scene: `realistic`, `lowpoly`, `toon`, `cel`, `painted`, `flat`, and `unlit` change how surfaces shade, and `ps1`, `n64`, `pixel`, `gameboy`, `pico8`, and `dos` change the whole screen. `SetMaterialShading` sets the first group per object. Scenes can mix them. Terrain, water, and scattered trees and grass keep their own shaders, so a style reaches a game's objects and not its landscape.

## Graphics profiles

`SetGraphicsQuality("low" | "medium" | "high" | "ultra" | "auto")`. A game that never calls it gets auto: a starting guess from the GPU's name, then steps down if frames stay over 26 ms, with resolution giving way first. All tiers show the same picture with less in it.

| | Low | Medium | High | Ultra |
| --- | --- | --- | --- | --- |
| Meant for | About ten years old: Intel HD, GT 730 | About seven years old: Iris Xe, GTX 750 Ti, Vega 8 | Current integrated and up | Current discrete card |
| Sun shadows | 1024, 2 cascades | 1024, 3 cascades | 2048, 4 cascades, soft | 2048, 4 cascades, soft |
| Ambient occlusion | Off | 2 directions | 4 directions | 6 directions |
| Bloom | Off | On | On | On |
| Anti-aliasing | FXAA | FXAA | FXAA | FXAA and 4x MSAA |
| Water reflection and refraction | Sky color, baked shore | 512 | 1024 | 2048 / 1024 |
| Trees, grass, and props | 45% | 70% | All | All |
| Largest texture | 1024 | 2048 | No limit | No limit |
| Depth of field, motion blur | Skipped | If asked for | If asked for | If asked for |

Frame time on a Radeon 780M at 1280x720: the only GPU these were measured on. The low column's weaker hardware is an estimate: at 2560x1440, four times the pixels, low takes 9 to 12 ms on the same scenes.

| Scene | Low | Medium | High | Ultra |
| --- | --- | --- | --- | --- |
| Island, from the air | 4.4 ms | 7.8 ms | 10.6 ms | 20 ms |
| Forest, on the ground | 6.3 ms | 7.7 ms | 11 ms | 21 ms |
| Valley | 4.6 ms | 7.3 ms | 13.6 ms | 26 ms |

Low keeps the shoreline without a refraction pass: depth under the water is baked once from the terrain's height field. Materials stay the same on low, because they turned out to be a small cost: the materials scene spends 0.4 ms on them.

## What keeps draw calls and work down

- **Copies:** entities that share a geometry and material draw as one instanced call, with no script changes. 800 separate `CreateCube` calls and 200 `CreateSphere` calls took 8 draws here.
- **Forests:** trees are planted in merged blocks with far versions. About a million triangles take about 230 draws on the island.
- **Culling:** frustum culling for each copy and block, and in each shadow cascade.
- **Terrain occlusion:** trees behind a hill are not drawn, from the height field on the CPU, with no GPU queries and no delay. It saved 10 to 20% of GPU time in most ground-level views here and the picture did not change. `SetTerrainOcclusion(False)` turns it off and `StatsOccluded()` counts what it hid. It covers only trees behind terrain; buildings do not hide anything.
- **Detail:** batched meshes get simpler versions with distance, and terrain has its own levels.
- **Textures:** mipmaps come with every loaded texture, `SetTextureLimit` overrides a tier's size limit, and `LoadTextureAsync` decodes on a worker: the texture works at once as a grey pixel, `TextureReady` and `TexturesLoading` report on it. Meshes still load on the main thread.
- **Shaders:** compiled programs are cached on disk.
- **Auto exposure:** brightness is read back in the background every four frames. Reading it directly made the CPU wait for the GPU every frame, which caused most of the over-budget frames on a walk through the island: 100 to 240 of 1,800 before, 2 to 5 after.

Texture atlases are not built. Their use is to merge draws split by texture, and generated trees carry no textures. They would help a game with many small textures on one kind of mesh.

## Post stack

Each effect has its own switch and works alone: `SetVignette(0.4)` by itself gives a vignette and leaves the rest of the picture as it was, with no tone curve, bloom, or FXAA. What a script turns off stays off through `SetLook` and `SetGraphicsQuality`, although a tier can still take away what it can't afford. `examples/post_stack.mb` has a key for each.

The order: the scene in HDR, with MSAA resolved; depth of field; motion blur; bloom; ambient occlusion; exposure and tone map; grade and LUT; outlines; sharpen; vignette; FXAA; grain; FSR when scaling. Occlusion is applied after the lens effects, so its contact shadows are not blurred with what they fall on. Added cost on a Radeon 780M at 1280x720, on the island: depth of field 0.5 to 1 ms, also at a large blur, and motion blur about 0.3 ms.

Not built, on purpose: general screen-space reflections, which the report found cost about 5 ms on this class of GPU (water has its own reflection); and a depth prepass, which sources disagree on and which has not been tested here.

## Staying smooth

Every game, run or compiled, has two steps of protection, in this order:

1. **Dynamic resolution.** The 3D scene draws smaller when the GPU runs short of time, AMD FSR 1 upsamples and sharpens it onto the window, and HUD text stays at full size. Terrain mips are biased to match, so the smaller picture does not also get blurrier textures. Where the driver has a GPU timer, three frames past 92% of the budget drop 5% per axis (10% past 110%), before any frame is missed. It sharpens after 1.5 s if the next step is predicted under 80%. Without a timer it falls back to missed frames: over a quarter of the frames in half a second running late drops a step, and 4 s on time tries one sharper. Each tier has a floor: ultra 0.85, high 0.75, medium 0.65, low 0.5.
2. **Quality tiers.** Under `SetGraphicsQuality "auto"`, only once resolution is at its floor does a 26 ms average drop one tier. Tiers never step back up by themselves.

The first second, frames over 250 ms, and frames that spent little time drawing (a script that sleeps or computes for long) are not counted. Use `SetDynamicResolution(False)` when timing the renderer, since it holds frame time steady by changing the picture size.

Every tier keeps the post pass. G3N asks for an 8x multisampled window, so drawing the scene straight to the window was about 3x slower than the low tier's post pass, and switching to it stalled for seconds (Radeon 780M). Low only gives up bloom and ambient occlusion from the pass.

## Pacing and first launch

A fence at each frame's end keeps the GPU within two frames of the CPU, which bounds input lag and stops queued frames ending in a long stall. `StatsFrameMs(99)` gives the 1% low of the last ten seconds and `StatsMissedFrames()` counts frames over 1.5 times the target. Report those, not average FPS, since an average can hide a stutter.

One open issue: on the Radeon 780M a run sometimes has a single stall of 500 to 900 ms, at a varying frame. It is not garbage collection (pauses stay under 3 ms) and it does not follow the quality tier, grass, or the shader cache. A stack sample showed it waiting inside a driver call, and removing that call only moved the wait, so its cause is unknown. It may not be moonBASIC's own work at all, and it needs checking on a second machine.

Compiled shaders are saved under the user's cache folder, keyed on GPU and driver, so only a game's first launch on a machine pays to compile them. On the Radeon 780M, shader work on the island went from about 7.2 s to about 15 ms, and the first frames from 7.6 s to 0.3 s. A driver may refuse a saved program, usually after an update; it then compiles from source and saves again. The saved form needs a driver that offers binary formats. Where none does, games compile on every launch as before. `MOON_NO_SHADER_CACHE=1` turns it off, `MOON_SHADER_LOG=1` prints each program's cost, and `StatsShaderMs()` gives the total. Programs moonBASIC builds directly, such as post effects and FSR, are not saved yet.

## Where the frame goes

`StatsPassMs(name$)` and `MOON_GPU_PROFILE=1` report GPU time per pass, from timestamps read a few frames late. `examples/island_orbit.mb`, dynamic resolution off, Radeon 780M at 1280x720, ms:

| Tier | Shadow | Reflect | Refract | Scene | Post | Frame |
| --- | --- | --- | --- | --- | --- | --- |
| High, before the refraction copy | 4.33 | 0.28 | 4.47 | 6.02 | 1.00 | 17.7 |
| High | 4.02 | 0.27 | 0 | 6.71 | 1.02 | 13.6 |
| Medium | 2.40 | 0.12 | 0 | 5.35 | 0.68 | 9.4 |
| Low (before the refraction copy, which it does not use) | 1.81 | 0 | 0 | 2.77 | 0.45 | 5.7 |

Later changes on the same path brought High to 9.6 ms, Medium to 6.9 ms, and Low to 4.8 ms: far shadow cascades redrawing in turn (`SetShadowTurns`), terrain sampling only the materials that show, and no multisampled window (a patch in `third_party/g3n`). A script that calls `EnablePostFX(False)` now gets no anti-aliasing, since the window has no samples. `SetFXAA` is real edge-directed FXAA, after tone mapping; retro looks skip it.

With post effects on, water refraction copies the finished scene when the water is reached instead of drawing the world again. Without a scene target it falls back to the old redraw. Things above the water can then show in a distorted edge of the refraction, a small artifact the redraw did not have.

## Linear lighting

`SetLinearLight True` (part of `SetLook "modern"`) changes how color is handled. Off, which is the default, shaders add lights up on display values and nothing changes for existing games.

On:

- The lit shaders (`bsshadow`, `mbterrain`, `mbphysical`, `mbwater`) decode colors to linear light, add the lights there, and write to the half-float scene target with no clamp.
- Terrain and Phong surfaces take their ambient from the sky picture instead of a fixed gradient.
- Fog still blends as the eye sees it, so `SetFog` distances mean what they did.
- The sky, clouds, and particles are not lit. They decode their output, so they look as they did.
- The post pass applies exposure and the tone curve, then encodes sRGB. Grading, LUTs, and retro effects run after that, on display values.
- Water's Fresnel uses a normal facing the eye. The old path can read the grid as back-facing, which makes it mirror the sky at full strength, so colors tuned for the old path come out much darker here.

Colors tuned for the old path need re-tuning, usually lighter. It needs the post pass, which every tier keeps. Extra windows have no post step, so they stay on the old path. Shaders from `CreateShader` get `uniform int LinearLight` and are not converted for you. Light probes capture into 8-bit targets, so they can band in dark scenes.

## Benchmarking another computer

`moon bench` walks the island along its track on each tier, then once as a game that never picks a tier would, and writes `bench-report.txt`: the GPU and OpenGL version, then each tier's average, median, 95th and 99th percentile, worst frame, hitches, frame count, load time, and GL errors. It names the highest tier whose 95th percentile holds 20 ms, and says where auto started and ended. It takes about three minutes, with the window showing. Ultra runs only if high has plenty of room, and it stops early if a tier is hopeless.

`moon bench -pack bench.zip` writes a zip for a computer with nothing installed: unzip it, double-click `run-bench.bat`, and send back the report.

## Reference images

`moon snapshot` draws each scene in `cmd/moon/testdata/render` with time fixed and the window hidden, then compares frame 90 with the PNG beside it. A pixel differs when a channel moves more than 24 of 255, and a scene fails when over 2% of its pixels differ, so driver rounding passes and a real change does not. Failures leave the new picture and a diff in `moon-snapshots` under the temp folder.

After a change you meant to make, run `moon snapshot -update` and look at the new images before keeping them. The references were drawn on a Radeon 780M. Other GPUs have not been compared against them.

`MOON_FIXED_DT=s` makes every frame last `s` seconds; snapshots use it, and it works with any run.

## Retro geometry

PS1-style effects in the lit shaders, switched for the whole world and all off by default. With everything off the shaders take the old path and the image is unchanged. Post effects such as dithering and palettes are separate (`docs/RETRO.md`).

| Command | Effect | Shaders |
| --- | --- | --- |
| `SetRetroSnap w, h` / `SetRetroSnap n` | Vertex snapping: clip-space xy is rounded to the pixel grid of a `w` × `h` screen, keeping w, so vertices hop from virtual pixel to virtual pixel. One argument takes the rows from the window aspect | `bsshadow`, `mbphysical`, `mbterrain`, `mbwater` vertex |
| `SetRetroAffine amount` | Affine texture mapping: a `noperspective` copy of the UV beside the usual one, blended 0–1 per fragment. 0 is perspective-correct, 1 the full PS1 warp | `bsshadow` (all material textures and the spec, normal, and emissive maps), `mbphysical` (every map), `mbterrain` (splat textures via a `noperspective` world position, and the plain texture path) |
| `SetRetroFlat on` | Flat shading: the face normal `cross(dFdx(p), dFdy(p))` in the space each shader lights in (view space for lights, world space for shadows and ambient), turned to the side of the smooth normal | `bsshadow`, `mbphysical`, `mbterrain` |
| `SetRetroFogSteps n` | Stepped fog: the fog factor is rounded to `n` bands | `bsshadow`, `mbphysical`, `mbterrain`, `mbwater` |

Notes:

- The shadow-depth pass is never snapped, so shadows stay steady while the models wobble.
- Like the real thing, a big polygon that runs behind the camera tilts as its far corners snap, and whatever is drawn on it (shadows, fog, textures) moves with it. A ground made of one huge `CreatePlane` drifts; build floors from smaller tiles or use a terrain. Triangles wholly in front of the camera move by at most half a virtual pixel.
- Affine warp shows on large triangles near the camera. Small, finely tessellated meshes barely change, as on the PS1.
- Flat shading replaces the normal before normal maps are applied, so a normal map still adds detail on each facet.
- GPU particles, `glmodern` draws, user shaders, and the sky are not affected.
- Go: `w.setRetroGeom(snapW, snapH, affine, flat, fogSteps)` and `w.retroGeomState()` set and read all of it at once (presets use them). State lives in `internal/runtime/retro_geom.go`; the uniforms (`RetroSnap`, `RetroAffine`, `RetroFlat`, `RetroFogSteps`) are bound with the fog uniforms on every lit material.

## Commands

Every command in this file is registered and listed in `docs/COMMANDS.md` (Modern OpenGL + Instancing). Demo: `examples/glmodern.mb`.
