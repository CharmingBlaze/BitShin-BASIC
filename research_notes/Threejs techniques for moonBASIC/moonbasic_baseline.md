# moonBASIC renderer baseline, for deciding which three.js techniques would be new

Read from branch `moonbasic-baseline`, commit `121c84d`, with no uncommitted changes under `internal/`, `docs/` or `reports/`. Sources below are repository paths; `internal/runtime/` is shortened to `rt/`, and `ENG` means `reports/Engine rendering techniques for moonBASIC.md`. No code was run: "runs" below means I traced it from command to shader in the source, not that I saw it on screen. All timings are from one Radeon 780M at 1280x720 unless stated otherwise, and timings on that machine "moved a lot from run to run" (`docs/GRAPHICS.md:90`).

## For each area: present, partial, or missing, with file:line evidence?

### Takeaway
The GL 3.3 forward path has real CSM with PCF, half-res GTAO, FXAA, optional MSAA, progressive bloom, DoF, camera motion blur, four tone curves, 3D LUTs, planar-reflection water, instancing, batching and auto LOD. It has no TAA, SMAA, real SSR, dynamic GI, volumetrics, user render targets, user post passes or node graph, and of the material lobes only clearcoat and a heuristic subsurface exist. Several commands promise more than they do: `EnableShadowCache` and `EnableTessellation` are stubs, MSM reads only two moments, `CreateVolumetricClouds` is one flat sheet, `SetScreenSpaceShadows` searches the shadow map, and `SetWaterSSR` is not SSR.

### Cited Findings

#### Shadows

| Item | Status | Evidence |
|---|---|---|
| CSM (1-4 cascades, sun) | Present | Practical splits, lambda 0.70, default distance 250: `rt/shadow_csm.go:13-37`. Cascades live as tiles in one atlas: `rt/shadow_sample.go:68-78`; `docs/SHADOWS.md:65`. Blending between cascades takes the darker result: `rt/shadow_sample.go:285-316` |
| Far cascade time-slicing | Present | Cascade 0 every frame, 1 every 2nd, 2 and 3 every 4th: `rt/shadow_turns.go:14-26`. Moment filters always redraw everything: `rt/shadow_turns.go:28-30` |
| PCF | Present | 3x3: `rt/shadow_sample.go:92-105`. 16-tap Poisson with interleaved gradient noise rotation: `:107-123`. Default is 3x3: `docs/SHADOWS.md:9` |
| PCSS | Partial (runs, but narrow) | Blocker search then PCF: `rt/shadow_sample.go:125-145`. The filter size is clamped: `k` to 2-8 (`:143`), radius `k*0.42*softness` (`:111`), so about 3.4 atlas texels at most at softness 1. Default search is `max(0.04*10*texel, 2*texel)`, which is 2 texels (`:128`; default size 0.04 at `rt/shadows.go`, `renderShadows`). On by tier only for Ultra: `rt/quality.go:48,169-174` |
| EVSM | Partial | Writes only a positive exponent and its square (`rt/shadow_shaders.go:629-633`) to RG32F (`rt/shadow_fbo.go:159-165`), blurred (`rt/shadow_fbo.go:279`, blur shader `rt/shadow_shaders.go:653-668`), read with Chebyshev: `rt/shadow_sample.go:147-162`. No negative-exponent moment. Off by default: `docs/SHADOWS.md:9,36` |
| MSM | Partial (mislabelled) | Writes four moments (`rt/shadow_shaders.go:634-636`) but `msmAt` reads only `b.x`, `b.y` and applies the VSM Chebyshev bound: `rt/shadow_sample.go:164-175`. `docs/SHADOWS.md:37` calls it "Four-moment maps". There is no moment reconstruction, so visually it is VSM |
| Local light shadows | Present | Up to 2 point (6 atlas faces each) and 2 spot, in the same atlas and filter: `rt/shadow_sample.go:355-388`; `rt/shadows.go:24`; `docs/SHADOWS.md:61,66-67`. Not cubemaps |
| Spot cookies | Present | `SetLightCookie`: `rt/commands_light.go:476`; `rt/light_shade.go:41-42`; `rt/pbr_shaders.go:546` |
| Contact shadows | Partial (name) | A search in the shadow map, not screen space: `rt/shadow_sample.go:220-234` |
| "Screen space shadows" (`SetSSS`) | Partial (name) | Same idea with a wider radius, on `ShadowMap`, not scene depth: `rt/shadow_sample.go:236-254`. No screen-space ray march exists |
| `EnableShadowCache` | Stub | Allocates a static and a dynamic target (`rt/shadows.go:839-850`) and binds the second to unit 9 (`:286-292`), but `ShadowMapDyn` and `ShadowCacheSplit` are declared and never read in GLSL (`rt/shadow_sample.go:8-9`; no other GLSL hit). `staticReady` is only ever set false (`rt/shadows.go:589`, `rt/commands_light.go:318,323,328`). `cacheKey` is written and never compared (`rt/shadows.go:597,1211`). `shadowCacheStale` has no caller (`rt/shadows.go:1223`). The real caching is shadow turns |
| Tree shadow proxies | Present | Near trees cast from the far model: `docs/GRAPHICS.md:163`; ground view 19.3 to 14.2 ms: `ENG:89` |

#### AO, AA, SSR

| Item | Status | Evidence |
|---|---|---|
| AO | Present | Half-res horizon GTAO from depth only, with 2/4/6 slices and 5 steps each, then a two-pass depth-aware blur: `rt/postfx_shaders.go:405-521`, `rt/postfx.go:965-1041`, R8 targets at `rt/postfx.go:907`. Composited by multiplying the final HDR colour, weakened on bright pixels: `rt/postfx_shaders.go:238-242`. So it darkens direct light too and has no specular occlusion. No temporal accumulation |
| FXAA | Present | Real edge-walking FXAA after tone mapping, on a display-referred target: `rt/fxaa.go:5-10`; routing at `rt/postfx.go:1204-1208,1299-1311`. The old gated blur remains in the main shader as a fallback when the target is unavailable: `rt/postfx_shaders.go:118-131`, `rt/postfx.go:1260`. Retro looks skip it: `rt/fxaa.go:128-131` |
| MSAA | Present | 2/4/8x RGBA16F and depth renderbuffers, resolved by blit: `rt/postfx.go:584-638`; `SetMSAA` at `:308`. Tier default is Ultra 4x only: `rt/quality.go:48`. The window itself has none: `third_party/g3n/window/glfw.go:243` (`Samples, 0`) |
| TAA | Missing | No match for TAA, history buffer or projection jitter in `rt/` (the only "jitter" in post is sample noise in motion blur: `rt/postfx_lens.go:107`); listed as not done at `ENG:96` |
| SMAA | Missing | No match in `rt/`; recommended but not built at `ENG:36,76,273` |
| FSR 1 | Present | Spatial upscale and sharpen under dynamic resolution: `rt/fsr.go`, `rt/postfx.go:1200-1201`; `docs/GRAPHICS.md:224` |
| General SSR | Missing, by decision | `docs/GRAPHICS.md:218`, `docs/STATUS.md:72` |
| `SetWaterSSR` | Partial (name) | Registered at `rt/water.go:1545`, bound at `rt/water_fx.go:305`, and steps inside the planar reflection texture from `reflectUV`: `rt/water_shaders.go:389-395`. No depth test or scene ray march, as `ENG:118` already said |

#### Light transport

| Item | Status | Evidence |
|---|---|---|
| Sky IBL | Present | Cube built on the CPU in RGBA8 with 8 samples per mip (6 mips), sizes 32/64/128/256 by tier: `rt/envcube.go:14-25,41-106`. Sampled by roughness in PBR, with the analytic split-sum DFG fit: `rt/pbr_shaders.go:366-401`. Diffuse is the cube's lod 5, not SH (`:371`). Rebuilt only when the sky changes: `rt/envcube.go:43` |
| Scene-captured light probes | Partial | One face per frame of the probe nearest the camera, in RGBA8: `rt/envcube.go:190,219-262`, called at `rt/world.go:1342`. All objects share that one probe (`rt/envcube.go:124-127,153-176`). No blending, no parallax correction, 8-bit (`docs/GRAPHICS.md:267`) |
| Probe colours | Partial | Inverse-distance blend of hand-set sky and ground colours into ambient: `rt/probes.go:22-68`. `ProbeSky` is read by the lit shader's non-linear path (`rt/shadow_shaders.go:416-419`) and bypassed under linear light (`:421-423`). `mbterrain` declares it with no read in that file: `rt/terrain_shaders.go:67-68` |
| SH | Partial | SH9 is projected from those two colours but reaches only the `SHEval` command, not a shader: `rt/probes.go:67`, `rt/glmodern_cmd.go:234` |
| Lightmaps | Partial | `SetLightmap` adds a second material texture on the first UV path: `rt/probes.go:134-143`, `rt/shadow_shaders.go:227-238`. Terrain has a similar slot: `rt/terrain.go:1256,1768`. No baker, no second UV set seen in those lines |
| Dynamic GI (SSGI, voxel, irradiance volume) | Missing | No match for ssgi, voxel, or irradiance in `rt/` |
| Area lights | Partial | Rectangles only, at most 4 per frame nearest the camera, no shadows: `rt/area_light.go:19-26,162-218`. Diffuse is a 3x3 point grid (`:72-89`), specular a representative point with widened lobe (`:92-106`; `rt/pbr_shaders.go:555-575`). Not LTC (no `ltc` match). PBR specular uses the geometric normal `wN`, so no normal map (`rt/pbr_shaders.go:563,573`). Terrain and water ignore them (`rt/area_light.go:24`) |
| Volumetric light, god rays, fog volumes | Missing | Only hits for "volumetric" are the cloud command names below. Fog is analytic per surface: linear, exp, exp2 plus height falloff (`rt/pbr_shaders.go:655-672`), sun-tinted (`:110-114`), with an RGB transmittance curve (`rt/linear_light.go:26-33`). `ENG:96`: light shafts not done |

#### Post and colour

| Item | Status | Evidence |
|---|---|---|
| Bloom | Present | 5 RGBA16F levels, 13-tap down, tent up, soft threshold at 0.72: `rt/postfx.go:717-833`, `:806`; `ENG:95`. Off on Low: `rt/quality.go:202-204`. Old 8-tap fallback in main shader: `rt/postfx_shaders.go:247-261` |
| DoF | Present | Single-pass golden-angle spiral on HDR, depth-driven, around 50 samples maximum: `rt/postfx_lens.go:16-73`. One pass, no separate near and far layers. Skipped on Low: `:247` |
| Motion blur | Partial | Camera only, 9 taps along a depth-reprojected vector; the source says objects moving on their own are not blurred: `rt/postfx_lens.go:76-115,205-222` |
| Tone maps | Present | Reinhard, Khronos PBR Neutral, ACES (Narkowicz fit), clamp: `rt/postfx_shaders.go:74-109`, `rt/postfx.go:162-178`. `SetLook "modern"` picks ACES at exposure 0.8: `rt/quality.go:307-308`. No AgX, no Hill ACES (no code match in these functions) |
| Auto exposure | Present | Non-blocking 1x1 PBO read every 4 frames: `rt/postfx.go:1100-1142` |
| Grade | Present | Contrast, saturation, tint only, on display values: `rt/postfx_shaders.go:111-116`. No lift/gamma/gain |
| LUT | Present | 3D RGB16F from `.cube`, PNG strip or tiles, or named preset: `rt/postfx_lut.go:21-31,265-284`. No default LUT per tier (`ENG:92`) |
| Other post | Present | Depth outline, sharpen, vignette, chromatic aberration, grain, retro, rain, 8-bit dither: `rt/postfx_shaders.go:161-174,220-224,274-303` |

#### Sky and water

| Item | Status | Evidence |
|---|---|---|
| Procedural sky | Present | CPU-built six-face images: `rt/skybox.go:335`, default for `SetLook`: `rt/quality.go:315-319` |
| Physical atmosphere | Partial | Optional dome (`CreateAtmosphere`): 256x64 transmittance LUT baked in Go (`rt/atmosphere.go:21,112,230`), 12-step single scatter per pixel, own `1-exp` curve, sun disc: `rt/atmosphere_shaders.go:77-110`. The "Bruneton-style" comment (`:5`) overstates it: no multiple scattering, sky-view LUT or aerial LUT (no match). `ENG:101`: row 17 not started |
| Aerial perspective | Partial | Fog colour from the sky horizon (`rt/quality.go:346-353`) plus the RGB `fogMix` curve (`rt/linear_light.go:26-33`). No scattering integral |
| Clouds | Partial (name) | `CreateVolumetricClouds` is "one wide sheet" with "no texture and no raymarch": `rt/terrain_clouds.go:13-16,160,244`. Weather-map banks and a thin high layer: `ENG:99`. No cloud shadows on terrain (`ENG:99`; no code match) |
| Water | Present | Gerstner, optional single GPU DFT tile at N=64 (`EnableFFT`): `rt/water_fft_gpu.go:10-21`, `rt/water.go:1411`; no cascades. Planar reflection 0/512/1024/2048 by tier: `rt/quality.go:45-48`. Refraction is a scene copy: `rt/water.go:1113-1133,1155-1165`. Beer-Lambert: `rt/water_shaders.go:344-351`. Low bakes shore depth: `rt/water_shore.go:8-15`. Caustics on receivers: `rt/pbr_shaders.go:641-654` |

#### Materials: `mbphysical` (`rt/pbr_shaders.go`), metal-roughness GGX with Smith and Schlick (`:256-273`)

| Lobe | Status | Evidence |
|---|---|---|
| Clearcoat | Present, partly | Direct and IBL: `:317-327,402-410`. `SetClearcoat e, amount [, rough]`: `rt/material_style.go:142-148`. Uniforms only: no coat map or coat normal |
| Subsurface | Partial (heuristic) | Wrap plus a back-light term in the direct path only: `:309-316`. `SetSubsurface`: `rt/material_style.go:150-156`. No thickness map, no screen-space or diffusion pass |
| Sheen | Missing | No match in `rt/` |
| BRDF anisotropy | Missing | Only hits are anisotropic texture filtering: `rt/aniso.go:5-29` |
| Iridescence | Missing | No match in `rt/` |
| Transmission | Missing | No match. Only alpha transparency: `rt/pbr.go:79`. Refraction exists for water alone |
| IOR or specular colour | Missing | F0 is fixed at 0.04 for dielectrics: `rt/pbr_shaders.go:458` |
| Also present | Present | Base colour, MR, normal, emissive, occlusion, equirect env maps (`:51-68`), parallax occlusion (`:190-219`), toon/cel/flat/low-poly/painted (`:129,275-289,604-628`) |

glTF import: the vendored loader handles only `KHR_materials_common`, `KHR_materials_pbrSpecularGlossiness` and `KHR_materials_unlit` among material extensions (`third_party/g3n/loader/gltf/`, file names from a `KHR_materials` search), so a glTF clearcoat would not arrive through an extension.

#### Geometry and scale

| Item | Status | Evidence |
|---|---|---|
| Instancing | Present | `CreateInstancedMesh`, GPU with CPU merge fallback: `rt/instance.go`; `docs/GRAPHICS.md:96-101` |
| Automatic batching | Present | Shared geometry with equal material settings becomes one instanced draw in scene and shadow tiles: `rt/batch.go:22-30`. Exclusions: transparent, skinned, morph, multi-material, swaying trees, custom shaders, terrain, water: `docs/GRAPHICS.md:72` |
| Static merge | Present | `rt/batch_merge.go:12-17`; `docs/GRAPHICS.md:76` |
| LOD | Present, automatic only | Batched meshes of 256+ triangles get two simplified versions on a worker: `rt/batch_lod.go:12-16`. A single uncopied mesh keeps full detail (`:16`). Terrain has levels with a geomorph weight: `rt/terrain_scatter.go:69-76`. Hand-built tree far models: `docs/GRAPHICS.md:163`. No script LOD command found (searched `createlod`, `setlod`, `lodgroup` and similar) |
| Impostors | Partial | Trees only: one side-on picture rasterized on the CPU, cards past the 3D reach: `rt/terrain_impostor.go:12-42,226-229`. No octahedral or general impostor |
| Occlusion | Partial | CPU height-field test, tree blocks behind terrain, main view only: `rt/terrain_occlusion.go:11-17,66`, used at `rt/postfx.go:1437`. No GPU occlusion query (the only `BeginQuery` use found in a search for it is a particle timer: `rt/particles_gpu.go:740`) and no HZB |
| User render targets | Missing | Roadmap unchecked: `docs/ROADMAP.md:486,1016`; no matching command in `rt/`. Internal FBOs only; nearest public feature is `SetWindowCamera` |
| User post passes | Missing | "Partial: fixed built-in stack ... no custom passes": `docs/ROADMAP.md:1017` |
| Shader node or graph | Missing | "Shader-graph lite ... Not an Unreal node graph": `rt/user_shader.go:17-18`. Raw GLSL 330 per mesh. `SetShaderUniform` takes 1-4 floats only: `rt/user_shader.go:234-256`. No sampler binding for scene colour or depth was found in that file |

### Inferences
- Docs are not safe evidence on their own. `docs/POSTFX.md` still calls FXAA a "neighbor luma blend" and bloom a "half-resolution chain", while the code changed. `docs/GRAPHICS.md:90` says "There is no occlusion culling" and `:204` describes it. `rt/quality.go:42-44` and `docs/GRAPHICS.md:229` still describe the 8x window that `third_party/g3n/window/glfw.go:243` removed.
- From a three.js view, the clearly new features are: TAA, SMAA, real SSR, transmission, sheen, BRDF anisotropy, iridescence, IOR, LTC area lights, per-object velocity, user render targets, a custom post chain, node materials, volumetric lighting, raymarched clouds, octahedral impostors, HZB occlusion. Upgrades of something existing rather than new: true MSM or two-sided EVSM, wider PCSS, screen-space contact shadows, probe blending and parallax correction, SH ambient, multi-scatter sky.
- PCSS as coded cannot produce the long soft penumbra its name suggests, because the kernel tops out near 3.4 atlas texels regardless of blocker distance. A three.js-style wider PCSS would be a real visual change, not a duplicate. This comes from reading the constants, not from viewing it.

### Gaps
- I did not trace `EnableShadowAtlas`'s effect; the shaders always use atlas tiles.
- Whether a glTF PBR material lands on `mbphysical` with all its maps was not traced through the loader.
- Default values of `ShadowContact`, `ShadowSSS` and `ShadowColor`, and whether anything turns them on by default, were not checked.
- The `skyAmbient` GLSL and the `mbsky` and `mbclouds` shader bodies were not read in full, so sky and cloud details rest on comments and `ENG` entries.

## What did the report's do-not-build list and progress table decide, why, and what has changed?

### Takeaway
The list stands mostly on reasons that still hold: no render graph, deferred, new backend, virtual texturing, compute froxels or full Bruneton. The code has moved in four places: terrain occlusion was built and already noted, a GPU-name table has grown against the report's advice, `WaterSSROn` was never removed, and several "not started" or "not done" progress entries are stale (soft particles, baked shore, compressed textures, async textures).

### Cited Findings

#### Do-not-build list (`ENG:107-127`)

| Item | Reason given | Still holds? What the code shows |
|---|---|---|
| Render graph | Bevy dropped its own; fixed-order calls suffice (`ENG:109`) | Holds. Passes are plain ordered calls: `rt/postfx.go:1177-1179,1422-1445` |
| Deferred | MSAA hacks; Intel wants few targets; excluded by status doc (`ENG:110`) | Holds. Single colour attachment everywhere. MSAA is now only on Ultra (`rt/quality.go:48`), which narrows the MSAA half of the reason; Intel's half and the alpha and custom-shader paths remain |
| Vulkan, bgfx, WebGPU, G3D | No production-ready Go path (`ENG:111`) | Holds in code: still G3N on GL 3.3 (`docs/GRAPHICS.md:3`). `docs/STATUS.md:3` adds that "a later renderer replaces this one". External state not rechecked (no web research) |
| GPU occlusion | Far-plane cut did not speed the city (`ENG:112`) | Holds for GPU. CPU terrain occlusion was built and saved 10-20% of GPU time in most ground views: `docs/GRAPHICS.md:204`. Measured on trees behind terrain only |
| MDI, persistent mapping as base | GL 4.3/4.4; macOS 4.1 (`ENG:113`) | Holds. `gl4` calls exist only in `rt/glmodern.go` |
| Virtual texturing | Needs a whole pipeline (`ENG:114`) | Holds |
| Full Bruneton, Hosek-Wilkie | Cost and artifacts; no aerial perspective (`ENG:115-116`) | Holds. Dome is 12-step single scatter |
| Compute froxels and compute-only base features | Godot Compatibility lacks them; macOS has no compute (`ENG:117`) | Holds. Compute is only `CreateComputeShader` and `DispatchCompute`: `rt/glmodern.go:474-510`, `rt/glmodern_cmd.go:84` |
| SSR on water, and current `WaterSSROn` | Not real SSR; real SSR about 5 ms on a GeForce 650M; misses off-screen (`ENG:118`) | Partly acted on. No general SSR, but `WaterSSROn` still ships (`rt/water.go:1545`, `rt/water_shaders.go:389`). The report said "Wet floors in Advanced only" as the reopen condition. `docs/GRAPHICS.md:218` restates the cost as "on this class of GPU", though the source is a GeForce 650M and it was never measured on the 780M |
| EVSM or MSM as default | Bleeding with large depth ranges (`ENG:119`) | Holds; default is PCF. They remain options |
| Horizon waves | Compute and offline tools (`ENG:120`) | Holds |
| Runtime Basis, Draco, MaterialX | BC covers desktop (`ENG:121`) | Holds: `rt/compressed_tex.go:236` refuses BasisLZ. But runtime KTX2 with zstd and zlib to BC1-3 was built (`:21`) |
| GPU erosion | Not on 3.3 (`ENG:122`) | Holds |
| Broad GPU-name table | Source moved away; SE1's disables itself (`ENG:123`) | Weakened by the code. `autoTier` now covers many families across about 30 lines: `rt/quality.go:67-100`, up from the `quality.go:62-74` the report cited (`ENG:141`). The comment says only the 780M has been run. The "UHD matches hd graphics" problem the report named still exists: `rt/quality.go:84`. `moon bench` (`docs/GRAPHICS.md:271`) is a separate report, not a first-launch timed test |
| OGRE, HLMS, legacy paths (`ENG:124-126`) | Porting cost, legacy | Holds |
| Silent upward tier change | Pop and recompile (`ENG:127`) | Holds: `rt/quality.go:238-242` |

#### Progress table (`ENG:83-101`), checked against code

| Row | Table says | Code shows |
|---|---|---|
| 29 soft particles, decals | Not started (`ENG:101`) | Soft GPU particles exist and are on by default: `rt/particles_gpu.go:120,963-971`, `rt/particles_gpu_shaders.go:302-307`. No decal code found. Stale for half the row |
| 10 low-tier water | Baked shore depth and foam not done (`ENG:87`) | Done: `rt/water_shore.go:8-15`, `docs/GRAPHICS.md:184,197` |
| 31 background loading | Not started | `LoadTextureAsync` decodes on a worker; meshes do not: `docs/GRAPHICS.md:206`, `rt/asset_async.go`. Partly done |
| 32 KTX2, BC | Not started | Runtime DDS/KTX/KTX2 to BC1-3: `rt/compressed_tex.go:21,45,204`. HDR loaders and glTF extensions not checked |
| 33 high-tier options | DoF and motion done; shafts, TAA, CDLOD, volumetric clouds not | Confirmed. Terrain does have a geomorph weight (`rt/terrain_scatter.go:69`), so "CDLOD not done" should not be read as "no morphing" |
| 16 render queues | Not started | Confirmed: order values remain scattered: `rt/skybox.go:105`, `rt/atmosphere.go:248`, `rt/particles.go:195`, `rt/terrain_clouds.go:183`, `rt/water.go:489`, `rt/weather_lightning.go:125` |
| 17 sky | Not started | Confirmed |
| 21 render targets, post chain | Not started | Confirmed |
| 24 sky SH | Not started | Confirmed: SH reaches no shader; sky cube is CPU-built and rebuilt only on change |
| 27, 28, 34 | Not started | Confirmed by absence; batching proxies draw through G3N (`rt/batch.go:27-30`), no sort key was found |
| 20 tone map per profile, LUTs | Not done | Confirmed: curve is set by `SetLook` or `setQuality`, not tier: `rt/quality.go:138-139,307` |
| Report on Low tier | "stays on the old gamma path" (`ENG:131`) | Changed: `SetLook` now turns on linear light on Low too: `rt/quality.go:300-306` |

The table's own "next by rank" is 16 render queues, 17 sky, 21 render targets (`ENG:101`). The report's file was last changed at `0a96fe7`; two commits follow it (`7923e0c`, `121c84d`).

### Inferences
- The strongest standing reasons are platform ones (macOS at 4.1, weak iGPUs). They rule out compute-first three.js ports (WebGPU node pipelines, compute-driven culling) as base-path items, not as optional paths.
- The SSR decision rests on a 2014-era published number on a different GPU, plus the lack of a normal buffer. The second reason is the firmer one today. The DoF gather costs 0.5-1 ms with up to about 50 taps on this 780M, so a 5 ms SSR figure should not be assumed to transfer. This needs measuring, and I found no such measurement.
- Row 21 is a prerequisite for much of what three.js offers as user-level post, so it shapes the value of everything post-related.

### Gaps
- Whether soft particles predate the report (and the researcher missed them) or were added after: not traced in git history.
- Other do-not-build reasons that depend on outside projects (Bevy, Go bindings) were not rechecked, by instruction.

## Where would a new screen-space or material feature plug in, and how many free texture units does each shader have?

### Takeaway
Screen-space effects plug into the fixed sequence in `blitPostFX`, most easily through `applyLens` for HDR passes and the FXAA slot for display-space passes; each post pass is its own program, so units are not a constraint there. Material lobes go into `pbrDirect` and `iblTerm`, and must also be added to the batch key. PBR has 3 free units, terrain 0, lit 0 by the table (with 4 nominal slots unused in practice), and water 6.

### Cited Findings

#### Frame order
- Scene: shadows, water reflection, clear, `hideOccluded`, `rend.Render`, raw-GL `drawModernPass` (instances, geom points, GPU particles), MSAA resolve — `rt/postfx.go:1422-1448`, `rt/glmodern_draw.go:38-52`.
- Post: `fillGTAO`, `applyLens` (DoF, then motion), `fillBloomChain`, `measureLuma`, main `mbpost` (AO, bloom, tonemap, encode, grade, LUT, outline, sharpen, vignette, grain, retro, rain, dither), FXAA, FSR — `rt/postfx.go:1177-1183,1212-1311`.

#### Plug-in points

| Kind of feature | Where | What to touch |
|---|---|---|
| HDR, before bloom (TAA resolve, SSR composite, fog) | `applyLens`, `rt/postfx_lens.go:238-303` | Uses `use(prog)` and `draw()` closures and returns the result texture. `lensState.tex/fbo` is a `[2]` array (`:123`), so a third pass needs a third target. Programs come from `postProgram` (`:190`). Skipped on Low by `bloomAllowed` (`:247`). Commands via `lensCommands` (`:314`) |
| Half-res with depth-aware blur (SSGI, SSR-like) | Pattern of `fillGTAO`, `ensureGTAOTargets`, `rt/postfx.go:891-1041` | Composite in `mbpost`, `rt/postfx_shaders.go:238-242` |
| Display-referred AA (SMAA) | The FXAA slot, `rt/postfx.go:1204-1208,1299-1311`; pattern `rt/fxaa.go:133-160` | `w.fsr.small` and `w.fsr.mid` are the intermediate targets |
| Sub-pixel jitter (TAA) | Camera projection before `rend.Render`, `rt/postfx.go:1438` | Also needs history targets and per-object motion, neither of which exists. Targets are all disposed when size changes: `rt/postfx.go:505-508,540-542`, and dynamic resolution changes size in 5% steps (`docs/GRAPHICS.md:224`) |
| Transmission or anything reading opaque scene | Scene copy, `rt/water.go:1117-1133`, called from `rt/water_material.go:93` | A blit at the moment the material is reached. Tied to a `waterBody` today; `render order` puts water at 100 (`rt/water.go:489`). The copy has no mips: target is LINEAR with no mip chain (`rt/water.go:822-825`) |
| PBR direct lobe | `pbrDirect`, `rt/pbr_shaders.go:291-329` | Beside clearcoat and subsurface. Area lights use `pbrSpecOnly` separately (`:333-342`) |
| PBR environment lobe | `iblTerm`, `rt/pbr_shaders.go:353-412` | Clearcoat's return at `:402-410` is the model |
| PBR parameters | Field `rt/pbr.go:28-30`; uniforms `rt/pbr.go:515-528`; command `rt/material_style.go:142-156` | **Batch key:** `rt/batch.go:485`. A new parameter not added here would merge different-looking materials |
| A new hand-bound sampler | `rt/texunits.go:8-51,55-83` | `TestTextureUnitsDoNotClash` checks `programUnits` against GLSL and enforces "GL 3.3 promises 16": `rt/terrain_seam_test.go:430` |
| Tier and switches | `tierSettings`, `rt/quality.go:24-49`; `setQuality`, `:116-178`; `noX` latches, `rt/postfx.go:87` | Also add a pass timer: `rt/gpu_passes.go:17-24` |

#### Texture units (GL 3.3 minimum of 16 per fragment stage, the limit the test enforces; `rt/texunits.go`)

| Program | Used | Free in 0-15 |
|---|---|---|
| `mbphysical` (PBR) | 0-5 G3N maps; 8 shadow; 9 `ShadowMapDyn`; 10 env cube; 11 height; 12-13 cookies; 15 caustic | **6, 7, 14** (3). Unit 9 is recoverable (see below) |
| `bsshadow` (lit) | Table reserves 0-6 for G3N; 7 env cube; 8, 9 shadow; 10 spec; 11 emit; 12-13 cookies; 14 normal; 15 caustic | **None** by the table. The GLSL samples only 3 `MatTexture` (`rt/shadow_shaders.go:227-238`; comment `rt/texunits.go:20-22`), so 3-6 are nominally free unless a material carries more than 3 textures. Unit 9 recoverable |
| `mbterrain` | 0-1 G3N; 2-7 splat; 8, 9 shadow; 10 blend; 11 bake; 12-13 cookies; 14 env cube; 15 caustic | **None.** Unit 9 recoverable |
| `mbwater` | 5 wake; 6 refract; 7 reflect; 8 DuDv; 9 normal; 10 refract depth; 11 shadow; 12-13 cookies; 14 FFT | **0-4, 15** (6). Unit 15 has a caustic texture bound during setup (`rt/water_fx.go:218`), not sampled by the water table entry, so a new sampler there would need a rebind |
| `mbatmo` | 5 | 15 |
| `mbgpupart` | 1, 2 | 14 |
| `mbpost` main | 0 scene; 1 AO; 2 bloom; 7 LUT; 8 depth; 9 palette | **3-6, 10-15** (10). The comment "2-6 bloom chain" (`rt/postfx_lut.go:28`) is old; only level 0 is bound (`rt/postfx.go:1255-1257`) |
| DoF, motion | 0, 1 (`rt/postfx_lens.go:262-271`) | 14 |
| GTAO, blur, FXAA, bloom, lum | 1-2 each | 14 or 15 |

- `ShadowMapDyn` on unit 9 is bound but never sampled in GLSL (`rt/shadow_sample.go:8`), so removing the shadow-cache stub frees unit 9 in lit, PBR and terrain.
- The runtime never queries `GL_MAX_TEXTURE_IMAGE_UNITS` (no match for `MAX_*UNITS` in `rt/`), so anything above 15 would be unguarded.
- Vertex samplers (`WaterFFT`, `WaterWake` in the water vertex shader, `rt/water_shaders.go:23,27`) reuse the same unit numbers.

### Inferences
- A three.js-style material with several new texture maps does not fit in `mbphysical` as is. Clearcoat, sheen and transmission maps at the same time would use all of 6, 7, 9, 14. Uniform-only lobes fit freely.
- `mbterrain` cannot take a new texture without dropping or packing one. New terrain effects should be uniform-driven, or go in post.
- A new fullscreen pass has no unit pressure at all. Its limits are time and the data available (colour and depth).
- Adding a generic "scene copy" would reuse the water mechanism and costs one blit per frame when used, per the report's replacement of a 4.47 ms redraw (`docs/GRAPHICS.md:245-246`). Roughness-blurred transmission would also need mips, which are not there.

### Gaps
- The real unit limit on the 780M or other drivers is unknown from this repo. Desktop drivers often offer more than 16, but that is not established here.
- Combined limits across stages and the uniform-count headroom in `bsshadow`, `mbphysical` were not measured (`MAX_*UNIFORM*` is never queried).

## Is there a stored per-pixel normal or velocity buffer, or only depth?

### Takeaway
Only depth. The scene target is one RGBA16F colour texture plus one DEPTH_COMPONENT24 texture; there is no MRT anywhere in the runtime, no normal buffer, and no velocity buffer.

### Cited Findings
- Scene target: RGBA16F colour (`rt/postfx.go:514`), DEPTH_COMPONENT24 texture with NEAREST filter (`:522-524`), attached as `COLOR_ATTACHMENT0` and `DEPTH_ATTACHMENT` (`:530-531`).
- A search of `rt/` for `COLOR_ATTACHMENT[1-9]`, `DrawBuffers` and `gbuffer` found no match. `bindColorTarget` sets `DrawBuffer(COLOR_ATTACHMENT0)`: `rt/postfx.go:448`. `docs/STATUS.md:72`: "True SSR / deferred MRT | Not on G3N". Each lit shader has a single `FragColor` output, e.g. `rt/pbr_shaders.go:155`.
- Normals: GTAO rebuilds a geometric normal from depth differences per pixel, in its own shader, not stored: `rt/postfx_shaders.go:460-467`. It has no normal maps and depends on depth precision.
- AO target stores visibility only, in R8: `rt/postfx.go:907`.
- Velocity: none. Motion blur derives camera motion in the shader from depth and a reprojection matrix: `rt/postfx_lens.go:96-99`. Last frame's view-projection is one matrix (`:125,215-216`), reset when motion blur is off (`:244-246`) and on split-screen (`:208-210`). No previous model matrix exists (search for `prevModel`, `lastWorld`, `prevWorld`, `prevMVP`, no matches).
- Colour alpha is not free: PBR writes `baseColor.a` (`rt/pbr_shaders.go:635`).
- Readers of scene depth today: GTAO and blur (`rt/postfx.go:996,1021`), DoF and motion (`rt/postfx_lens.go:269`), outline (`rt/postfx_lut.go:302-303`), soft particles (`rt/particles_gpu.go:964-967`), and water through its copy (`rt/water.go:1131`, `WaterRefractDepth` on unit 10).
- With MSAA, depth is resolved by a NEAREST blit: `rt/postfx.go:635`. Depth exists only if the post pass is on (`rt/postfx.go:586`); every tier keeps the post pass (`rt/quality.go:45-48`), but `EnablePostFX(False)` removes it. The depth is what was drawn by the end of the frame; a comment on far water in GTAO (`rt/postfx_shaders.go:477-478`) suggests water is in it.
- Projection is plain perspective with depth linearised by the usual formula, not reversed Z: `rt/postfx_lens.go:32-36`.

### Inferences
- These three.js techniques need new data: per-object motion for TAA without smearing, per-object motion blur, normal-map-aware SSR and SSGI, and anything reading roughness per pixel. Depth-only versions (camera-only reprojection TAA, depth-normal SSR, N8AO-style AO) can run today with lower quality.
- Adding a normal or velocity attachment means a second output in every program that writes the scene: `bsshadow`, `mbphysical`, `mbterrain`, `mbwater`, `mbclouds`, `mbsky`, particle programs (`rt/shadows.go:219-229`, `rt/terrain_shaders.go:700-705`, `rt/water_shaders.go:565`), `CreateShader` user programs, the MSAA renderbuffers, and `bindColorTarget`. The alternative is a normal prepass, which redraws geometry; the depth prepass has not been tested here (`docs/GRAPHICS.md:218`) and redraws are named the largest cost (`ENG:155`).
- Velocity needs previous transforms through G3N, batching proxies, merges (which bake to world space), instances and skinning. Batch and merge make per-object history an unusually large change here.

### Gaps
- Cost of adding one more attachment on the 780M and on weak iGPUs is unmeasured. The report quotes Intel advising few targets (`ENG:110`).
- Whether every transparent or special pass writes depth was not audited.

## What are the constraints: units, OpenGL 3.3, optional 4.x paths, costs, tier table?

### Takeaway
GL 3.3 core is the required base and the 4.x paths are optional, opt-in demo commands that no built-in renderer feature uses. On the 780M the High tier runs about 10 ms at 720p with post near 1 ms, but no individual cost exists for AO, bloom, FXAA or PCSS, and no other GPU has been measured at all. The Low tier's target (Intel HD 520 class) is an estimate.

### Cited Findings

#### OpenGL
- G3N and extra windows request 3.3 core, never higher: `docs/GRAPHICS.md:5-31`, `rt/glmodern.go:17-20`. Shaders are `#version 330`: `rt/pbr_shaders.go:5`.
- Optional features are detected after the context opens, return 0 and print one skip line if missing: `docs/GRAPHICS.md:39-58`.

| Feature | Needs | Actually used by the renderer? |
|---|---|---|
| GPU instancing | GL 3.1 | Yes, base path: `docs/GRAPHICS.md:45,98` |
| UBO | GL 3.1 | User commands (`rt/glmodern_cmd.go`); no built-in use checked |
| Geometry shaders | GL 3.2 | `CreateGeomPoints` only: `docs/GRAPHICS.md:46,105` |
| Tessellation | GL 4.0 | Stub. `tessOn` is set and read only in `rt/glmodern_cmd.go:192-199`; "Terrain/water do not require it": `docs/GRAPHICS.md:106` |
| Compute | GL 4.3 | Demo plus user commands: `docs/GRAPHICS.md:110`, `rt/glmodern.go:474-510`. All `gl4.` calls are in `rt/glmodern.go` |
| SSBO | GL 4.3 | User commands; CPU mirror on 3.3: `docs/GRAPHICS.md:50` |
| Anisotropic filtering | Extension | Yes, 2/4/8/16 by tier if present: `rt/aniso.go:14-30`, `rt/glmodern.go:163-164` |
| Program binary cache | Extension-dependent | G3N programs only; direct GL programs such as post are not cached: `docs/GRAPHICS.md:237`, `ENG:90` |
| Timer queries | Used for `StatsPassMs`, "a value is a guide": `rt/gpu_passes.go:13-16` | Yes |

- macOS tops out at GL 4.1 (`ENG:113,167`), and the report advises avoiding geometry shaders and buffer textures in core paths and keeping shaders ES 3.0-friendly where cheap (`ENG:167`).
- No `sampler2DMS` or `textureGather` use was found in `rt/`.

#### Measured costs (Radeon 780M, 1280x720 unless noted; one machine, noisy)

| Scene, tier | Low | Medium | High | Ultra | Source |
|---|---|---|---|---|---|
| Island, from the air | 4.4 ms | 7.8 ms | 10.6 ms | 20 ms | `docs/GRAPHICS.md:193` |
| Forest, ground | 6.3 ms | 7.7 ms | 11 ms | 21 ms | `docs/GRAPHICS.md:194` |
| Valley | 4.6 ms | 7.3 ms | 13.6 ms | 26 ms | `docs/GRAPHICS.md:195` |

| Pass, `island_orbit`, dynamic resolution off | Shadow | Reflect | Refract | Scene | Post | Frame |
|---|---|---|---|---|---|---|
| High | 4.02 | 0.27 | 0 | 6.71 | 1.02 | 13.6 |
| Medium | 2.40 | 0.12 | 0 | 5.35 | 0.68 | 9.4 |
| Low | 1.81 | 0 | 0 | 2.77 | 0.45 | 5.7 |

Source: `docs/GRAPHICS.md:243-248`. Later changes brought frames to High 9.6, Medium 6.9, Low 4.8, without a new pass table (`:250`). Other figures for the same island differ (9.4/7.1/4.6 at `:159`; about 10 at `ENG:85`), which shows the run-to-run spread. The post figure spans everything through `endPostTarget` (`rt/world.go:1374-1376`), so AO, lens, bloom, FXAA and FSR are not separated.

| Effect | Cost | Source |
|---|---|---|
| DoF | 0.5-1 ms, on the island | `docs/GRAPHICS.md:216` |
| Motion blur | about 0.3 ms | `docs/GRAPHICS.md:216` |
| MSAA 2x | about 1.8 ms, 10% of frames past 16.7 ms | `rt/quality.go:34` |
| MSAA 4x | about 4 ms | `rt/quality.go:34` |
| Materials (scene with them) | 0.4 ms | `docs/GRAPHICS.md:197` |
| Terrain occlusion | saves 10-20% GPU in most ground views | `docs/GRAPHICS.md:204` |
| Refraction as a copy vs a redraw | saved 4.47 ms pass ("28% of the frame") | `docs/GRAPHICS.md:245-246`, `rt/water.go:1158-1159` |
| Tree shadow proxies | ground view 19.3 to 14.2 ms | `ENG:89` |
| GPU particles, 100k | about 0.4 ms as a fountain, 1.2 ms in another case | `docs/PARTICLES.md:65` |
| G3N per-object overhead | about 11 microseconds | `rt/batch.go:22` |
| Non-780M | None. Low hardware is an estimate; "2560x1440 low takes 9 to 12 ms" on the same GPU | `docs/GRAPHICS.md:189`, `ENG:86` |

Budget rules: `moon bench` names the highest tier whose 95th percentile holds 20 ms (`docs/GRAPHICS.md:271`); auto drops a tier at a 26 ms average once resolution is at its floor (`rt/quality.go:260-265`), and never goes up.

#### Tier table (`rt/quality.go:45-48`, with `rt/envcube.go:23-24`, `rt/aniso.go:14`, `docs/GRAPHICS.md:224`)

| | Low | Medium | High | Ultra |
|---|---|---|---|---|
| Meant for | Intel HD, GT 730 (HD 520 at 720p) | Iris Xe, GTX 750 Ti, Vega 8 | Current integrated and up | Current discrete |
| Shadow map, cascades | 1024, 2 | 1024, 3 | 2048, 4 | 2048, 4 |
| Filter | PCF | PCF | PCF | PCSS |
| AO slices | Off | 2 | 4 | 6 |
| Bloom | Off | On | On | On |
| AA | FXAA | FXAA | FXAA | FXAA and 4x MSAA |
| Water reflect / refract target | 0 / 0 (baked shore) | 512 / 512 | 1024 / 1024 | 2048 / 1024 |
| Terrain radius, detail px | 1, 14 | 2, 10 | 2, 8 | 3, 6 |
| Scatter, tree reach | 45%, 220 m | 70%, 320 m | 100%, 420 m | 100%, 480 m |
| Largest texture | 1024 | 2048 | No limit | No limit |
| Sky cube, probe face | 32, 32 | 64, 64 | 128, 128 | 256, 128 |
| Anisotropy | 2 | 4 | 8 | 16 |
| Dynamic resolution floor | 0.5 | 0.65 | 0.75 | 0.85 |
| DoF, motion blur | Skipped | If asked | If asked | If asked |

Every tier keeps the post pass, and with it the depth texture: `rt/quality.go:42-48`, `docs/GRAPHICS.md:229`.

### Inferences
- With High at about 10 ms and Post near 1 ms on the 780M, there is room for about 1-3 ms of new effects before the 16.7 ms budget tightens in heavier scenes (valley at 13.6 ms leaves about 3 ms). A 5 ms effect does not fit High on this GPU at 720p. This is arithmetic on noisy numbers.
- Ultra's near-doubling of High has no breakdown. MSAA 4x accounts for about 4 ms by the older figure, leaving 5-6 ms across AO 6 slices, PCSS, larger reflection and terrain radius 3.
- New effects should probably target Medium and up and skip Low, as `applyLens` already does. Low's 4.4-6.3 ms was measured on the 780M, not on the older hardware Low is meant for, so that hardware's headroom is unknown and likely far smaller.

### Gaps
- No per-effect timing for AO, bloom, FXAA, tone map, LUT, or PCSS on the 780M, because `gpu_passes.go` has only five passes.
- No Intel, NVIDIA, Linux or Mac numbers: `ENG:9,81`.
- A 1080p cost table does not exist; these costs would scale at least by pixels for screen-space work, but that is untested.
- GL 3.3 minimums beyond the 16 units the test cites (for example draw buffer count) are not recorded in the repo, and I did not look them up, as no web research was requested.
