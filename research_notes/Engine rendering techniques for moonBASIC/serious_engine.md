# Croteam's Serious Engine: how it scaled across hardware (notes for moonBASIC)

Evidence labels used below:
- **[SOURCE]**: read directly in Croteam's released Serious Engine 1.10 code, commit `b408e88` (2017-01-21). This code is SE1 only.
- **[CROTEAM]**: statement by Croteam staff. Marked "snippet" if I saw it only through a search excerpt because the page refused the fetch.
- **[PRESS]**: third-party measurement or analysis.
- **[COMMUNITY]**: forum post, user guide or wiki. Treat as recollection.

Link base for code: `https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/`

## Is Serious Engine 2 source public?

### Takeaway
I found no public SE2 source. The official release covers Serious Engine v1.10 only, so nothing in the repository may be attributed to Serious Sam 2 (SS2). Everything known about SS2 internals comes from outside observation, mainly one 2006 iXBT article.

### Cited Findings
- [SOURCE] The README describes the repository as "the source code for Serious Engine v.1.10". Its stated resources concern The First Encounter and The Second Encounter — [README](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/README.md)
- [PRESS] Coverage of the 2016 release says this is not the Serious Engine 4 that runs The Talos Principle, and that it cannot render modern AAA graphics (search excerpt) — [GamingOnLinux](https://gamingonlinux.com/2016/03/croteam-releases-serious-engine-version-110-as-free-software/page=2)
- Date in the file headers: "Copyright (c) 2002-2012 Croteam Ltd." — [GLSettings.cpp](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/SeriousSam/GLSettings.cpp#L1)

### Inferences
- The SE1 code shows Croteam's design habits before SS2, and SS2's console-variable naming looks related (`gfx_`, `ren_`, `mdl_`, `d3d_`). That is suggestive only. It does not show that any SE1 mechanism survived into SE2.

### Gaps
- "SE2 source is not public" is an absence claim. I found no release and no Croteam statement either way. I did not find a Croteam statement that explicitly rules out a future release.

## What is known about Serious Engine 2 (Serious Sam 2, 2005)?

### Takeaway
SS2 is thinly documented. The one deep source I found is iXBT's August 2006 graphics analysis. It shows a Direct3D 9 game with a hidden OpenGL path, assembly shaders in SM 1.x and 2.0 paths, five quality presets, and a fillrate-based "Auto detect quality" option. I found no Croteam technical talk or paper on SE2, so none of this has first-party confirmation.

### Cited Findings
All findings in this list come from [iXBT, Aleksey Berillo, 8 Aug 2006](https://www.ixbt.com/video2/tech_ss2.shtml) unless noted. It is an outside analysis using PIX and NVPerfKit, not a Croteam statement. I checked the raw Russian text for the HDR format, auto-detect, `gfx_iAPI` and 512x512 details. The rest comes from a machine summary of the page.

**API paths**
- [PRESS] The game uses Direct3D 9 by default. OpenGL exists but is not offered in the menu. `gfx_iAPI = 1` in the `[Engine]` section of `Sam2.ini` forces it.
- [PRESS] The author says OpenGL looks faster mainly because it lacks dynamic shadows and some post-processing, not because the renderer is better.
- [COMMUNITY] A player reports `gfx_iAPI = 2` is D3D and that switching to 1 fixed crashes — [Steam thread](https://steamcommunity.com/app/204340/discussions/0/613941122732198818). This was a search excerpt.
- [COMMUNITY] A 2005 forum commenter claimed OpenGL ran better because Croteam had longer to optimize it. This is speculation and conflicts with the iXBT explanation — [AnandTech forum](https://forums.anandtech.com/threads/serious-sam-2-open-gl-or-direct3d.774851). This was a search excerpt.

**Shader paths**
- [PRESS] There are two broad paths, SM 1.x and SM 2.0. Vertex shaders stay at vs_1_1. Pixel shaders use 1.1, 1.4 and 2.0, and most are 1.1.
- [PRESS] Shaders are written in assembly, which the author links to the Xbox target. The 2.0 shaders are mostly rewritten 1.x shaders with partial precision.
- [PRESS] The terrain shader takes 1 pass with up to 8 textures in ps_2_0, 2 passes in ps_1_4, and 3 passes in ps_1_1.
- [PRESS] Controls: `d3d_iMaxAllowedPixelProgramVersion`, `d3d_bForcePixelProgramVersion2`, and a menu option "Only shaders 1.1".
- Conflict: a Russian review lists pixel shaders 3.0 and vertex shaders 2.0 — [Mail.ru review](https://hi-tech.mail.ru/review/12583-serious_sam_2/) (search excerpt). iXBT's shader-level inspection is the stronger evidence. Its capture shows ps 2.0 as the maximum.

**Lighting and effects**
- [PRESS] HDR uses an FP16 buffer (`D3DFMT_A16B16G16R16F`), mainly for better bloom, with no eye adaptation. Cards without FP16 render targets fall back to LDR.
- [PRESS] ATI X1000 cards showed blockiness because they could not filter FP16 textures. A patch fixed this.
- [PRESS] Static shadows are baked into lightmaps. Dynamic shadow maps cover animated objects, come from one light (the sun), are unfiltered, and reach at most 512x512.
- [PRESS] Materials have diffuse, normal and specular layers. The engine supports PolyBump-style normal maps. A parallax mapping menu entry produced no visible or performance change in the author's tests.
- [PRESS] Water has refraction, but reflections are static.
- [CROTEAM, unverified] A 2005 developer-diary article describes the choice of "True HDR Rendering" for bright open environments — [Multiplayer.it](https://multiplayer.it/articoli/il-diario-di-sviluppo-di-serious-sam-2.html) (search excerpt, not read in full).

**Presets and auto-detection**
- [PRESS] "Auto detect quality" runs built-in low-level fillrate tests, then sets the Quality level. On the test system it chose Maximum, which the author reports was about 3x a GeForce 6600 GT reference.
- [PRESS] Presets as reported:

| Quality | Resolution | AA | Aniso | Textures |
|---|---|---|---|---|
| Minimum | 640x480 | None | None | Compressed |
| Low | 800x600 | None | None | Compressed |
| Medium | 1024x768 | 2x | 2x | Optimal |
| High | 1152x864 | 4x | 4x | Optimal |
| Maximum | 1280x960 | 4x | 8x | Optimal |

- [PRESS] No preset enables HDR, and none uses 32-bit textures.
- [PRESS] Presets also control texture size, particles, crumbs (small scattered detail), geometry LOD, lightmap quality and size, bloom, filtering, lens flare, parallax, reflections, refractions and "Only shaders 1.1".

**Measured cost of settings (iXBT, GeForce 7800 GTX class system, 1024x768)**

| Setting | FPS impact |
|---|---|
| Dynamic shadows | up to 35% at max settings, about 15% at min |
| Geometry LOD | about 30-35% at max, about 5% at min |
| Crumbs | about 8-10% at max, up to 20% at min |
| HDR | under 10% |
| Hardware skinning off | about 10% slower |
| AA 4x | 3-5% at max |
| Anisotropy, bloom, filtering, particles | 2-4% each |
| Texture and lightmap size | about 2% |
| 32-bit textures | average unchanged, minimums lower, much more VRAM |

- [PRESS] Preset average FPS in the demo: 81, 74, 61, 48 and 40 from Minimum to Maximum. The article does not name the GPU for this run, so read it as a shape rather than a figure.
- [PRESS] Physics alone took 25-30% of frame time, plus AI. VRAM use at max settings with 4x AA was 100-280 MB.

### Inferences
- Geometry LOD and dynamic shadows were the largest levers measured. Post-effects were small. That is one analyst's numbers on a fast 2006 GPU, so the ranking may not transfer.
- The presets scale resolution along with quality, which makes the fps range partly a resolution effect.

### Gaps
- PCGamingWiki's Serious Sam 2 page returned HTTP 403, so its settings details are unused.
- I found no Croteam description of the fillrate test, thresholds, or how scores map to presets.
- I found no first-party SE2 feature list. Minimum system requirements are not in iXBT.
- Normal-mapped characters are from iXBT only.

## What does the open-sourced Serious Engine 1 code show?

### Takeaway
SE1 does not adapt at runtime. It applies per-GPU scripts at startup, then relies on distance LOD with global bias knobs, a 20 Hz simulation with interpolated rendering, budgeted caches, and capability-driven multitexture merging. I found no frame-time-driven adaptive quality and no hardware instancing in the engine code.

### Cited Findings

**Performance profiles (`Sources/SeriousSam/GLSettings.cpp`)** [SOURCE]
- `InitGLSettings` reads `Scripts\GLSettings\GLSettings.lst`, where each line holds a renderer pattern, a description and a script — [L62-L94](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/SeriousSam/GLSettings.cpp#L62-L94)
- `ApplyGLSettings` matches `da_strRenderer` against those patterns. With no match it prints "No matching preferences found! Automatic adjustment disabled!" and returns — [L121-L135](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/SeriousSam/GLSettings.cpp#L121-L135)
- If the GPU description and preference level are unchanged, it keeps existing settings. Custom mode (`sam_iVideoSetup` 3) never runs a script. Otherwise it includes the script and calls `RefreshTextures()` — [L141-L164](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/SeriousSam/GLSettings.cpp#L141-L164)
- The levels are 0 speed, 1 normal, 2 quality, 3 custom — [SeriousSam.cpp L77](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/SeriousSam/SeriousSam.cpp#L77)
- `Scripts/GLSettings/` is not in the Git tree. Only `NetSettings`, `menu` and `Dedicated_startup.ini` are — [Scripts](https://github.com/Croteam-official/Serious-Engine/tree/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Scripts). So the actual per-GPU values are not readable in Git. They may be in the `SE1_10.GRO` resources, which I did not open.
- `CGfxLibrary::Benchmark` measures texture upload, fill rate and geometry throughput, but prints results. The only caller I found is the user console command `Benchmark()` — [Benchmark.cpp L347](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/Benchmark.cpp#L347), [SeriousSam.cpp L471](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/SeriousSam/SeriousSam.cpp#L471)
- Menu options can be hidden by capability, such as `Filter: sys_bHas32bitTextures` on texture quality — [RenderingOptions.cfg](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Scripts/menu/RenderingOptions.cfg)
- The default texture and shadow size options use the same log2 scale, for example `tex_iNormalSize` from 6.5 to 9 and `shd_iStaticSize` from 5 to 8 — [RenderingOptions.cfg](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Scripts/menu/RenderingOptions.cfg)

**Timing and simulation** [SOURCE]
- `CTimer::TickQuantum` is 1/20 s, so game logic runs at 20 Hz — [Timer.cpp L54](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Base/Timer.cpp#L54)
- The timer exposes a lerp factor for rendering between ticks, via `tm_fLerpFactor` — [Timer.cpp L367](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Base/Timer.cpp#L367)
- `LimitFrameRate` caps at `sam_iMaxFPSActive` (500) and `sam_iMaxFPSInactive` (10, when iconic) using `Sleep` — [SeriousSam.cpp L240-L256](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/SeriousSam/SeriousSam.cpp#L240-L256)

**Model LOD** [SOURCE]
- MDL models: a distance-based mip factor is adjusted for stretch, then multiplied and offset by `mdl_fLODMul` and `mdl_fLODAdd` — [RenderModel.cpp L505-L520](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Models/RenderModel.cpp#L505-L520)
- `GetMipModel` picks the first level whose switch factor exceeds the mip factor — [Model.cpp L1979](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Models/Model.cpp#L1979)
- `IsModelVisible` culls a model past its last mip switch factor, controlled by `mdl_iLODDisappear` (0 never, 1 ignore bias, 2 with bias) — [Model.cpp L1958](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Models/Model.cpp#L1958), [GfxLibrary.cpp L204](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/GfxLibrary.cpp#L204)
- SKA skeletal models have LOD on both meshes and skeletons, each with authored max distances scaled by `ska_fLODMul` and `ska_fLODAdd`. A return of -1 means not visible at that distance — [RMRender.cpp L1516-L1566](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Ska/RMRender.cpp#L1516-L1566)
- Model shadow quality is tiered: 0 none, 1 one simple, 2 one complex, 3 all — [RenderModels.cpp L17-L22](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Rendering/RenderModels.cpp#L17-L22)
- Individual features have switches such as `mdl_bRenderDetail`, `mdl_bRenderSpecular`, `mdl_bRenderReflection` and `mdl_bRenderBump` — [GfxLibrary.cpp L195-L198](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/GfxLibrary.cpp#L195-L198)

**Terrain** [SOURCE]
- Terrain is a tile grid with a quadtree, culled by `TestBoxToFrustum` on each node — [TerrainRender.cpp L1193](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Terrain/TerrainRender.cpp#L1193)
- Tile LOD is `distance / tr_fDistFactor`, clamped to `tr_iMaxTileLod`. Vertex rows halve per LOD. A change queues the tile and its four neighbours for regeneration — [TerrainTile.cpp L502-L540](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Terrain/TerrainTile.cpp#L502-L540)
- `tr_fDistFactor` defaults to 32 — [Terrain.cpp L57](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Terrain/Terrain.cpp#L57)
- Vertices are geomorphed with `tt_fLodLerpFactor` under `ter_bLerpVertices`. Dedicated border routines stitch tiles of different LODs — [TerrainRender.cpp L202-L257](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Terrain/TerrainRender.cpp#L202-L257)
- Under `ter_bOptimizeRendering`, tiles in the lowest LOD with no border vertices are delayed and drawn together using one global top-map texture — [TerrainRender.cpp L995-L1007](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Terrain/TerrainRender.cpp#L995-L1007), [L763-L768](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Terrain/TerrainRender.cpp#L763-L768)
- Multi-layer per-tile detail exists only at LOD 0, where layers regenerate — [TerrainTile.cpp L447-L455](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Terrain/TerrainTile.cpp#L447-L455)

**World textures and shadow maps** [SOURCE]
- Lightmaps ("shadow maps") are cached with a budget and a flush delay: `shd_fCacheSize` 8 MB and `shd_tmFlushDelay` 30 s, with `shd_bCacheAll` as an option — [GfxLibrary.cpp L140-L142](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/GfxLibrary.cpp#L140-L142)
- `shd_bAllowFlats` optimizes single-color shadow maps — [GfxLibrary.cpp L143](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/GfxLibrary.cpp#L143)
- `ProbeMode` returns true if a texture or shadow map has not been drawn for more than `gfx_tmProbeDecay` (30 s). Default probe budget `gfx_iProbeSize` is 256 KB — [GfxLibrary.cpp L751-L765](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/GfxLibrary.cpp#L751-L765), [L242-L243](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/GfxLibrary.cpp#L242-L243)
- Detail layers are dropped per polygon when their mip factor runs past the texture's last mipmap, biased by `wld_iDetailRemovingBias` — [DrawPort_RenderScene.cpp L229-L232](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/DrawPort_RenderScene.cpp#L229-L232)

**Batching and the OpenGL path** [SOURCE]
- Polygons are binned into groups by which layers they use, then merged into multitexture passes when `_ctUsableTexUnits>=2` — [DrawPort_RenderScene.cpp L289](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/DrawPort_RenderScene.cpp#L289), [L1270-L1287](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/DrawPort_RenderScene.cpp#L1270-L1287)
- Particles collect into a common vertex array and flush as quads. Environment particles scale with `gfx_fEnvParticlesDensity` and `gfx_fEnvParticlesRange` — [DrawPort_Particles.cpp L507-L520](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/DrawPort_Particles.cpp#L507-L520), [Particles.cpp L2548](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/EntitiesMP/Common/Particles.cpp#L2548)
- OpenGL is loaded through function pointers, and features are gated by extension checks that set flags in `gl_ulFlags` — [Gfx_OpenGL.cpp L132](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/Gfx_OpenGL.cpp#L132), [L414-L443](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/Gfx_OpenGL.cpp#L414-L443)
- `GfxLibrary.cpp` declares the driver knobs: `gap_bOptimizeStateChanges`; `gap_iOptimizeDepthReads` (0 immediately, 1 after frame, 2 every 0.1 s); `gap_iOptimizeClipping`; `ogl_iFinish`; `ogl_iMaxBurstSize`; `ogl_bUseCompiledVertexArrays`; and `gap_iUseTextureUnits` — [GfxLibrary.cpp L150-L179](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/GfxLibrary.cpp#L150-L179)
- The renderer uses legacy state, including matrix modes and client vertex arrays — [Gfx_OpenGL.cpp L511-L516](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/Gfx_OpenGL.cpp#L511-L516). D3D8 support is off by default in this release — [README](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/README.md)
- `CGfxProfile` has named timers for each scene stage, plus counters such as texture uploads and bytes — [GfxProfile.h](https://github.com/Croteam-official/Serious-Engine/blob/b408e88a16fd01aa1cfd0e0a999c86c2c1437c9e/Sources/Engine/Graphics/GfxProfile.h)

**Crowds** [SOURCE]
- A case-insensitive search of `Sources/Engine` for `Instanced`, `ARB_instanced`, `impostor` and `imposter` found nothing. I found no dedicated crowd system in the code I read. I did not review `Sources/EntitiesMP` AI code for enemy throttling.

### Inferences
- SE1's many-enemy performance likely came from the combination of authored mip levels, distance disappearance, tiered shadows, 20 Hz logic and multitexture merging. No file states this as a crowd strategy, so this is my reading.
- The profile list ages badly by design. An unknown GPU string turns adjustment off, leaving prior or default settings.
- SE1 appears to have no runtime adaptive quality. The grep covered `Sources/SeriousSam` for auto-adjust terms and found only the audio script hook `sam_bAutoAdjustAudio`. I did not audit every module.
- `ProbeMode` likely lets rarely drawn items upload at reduced size until they are used. I read the trigger and the `gfx_iProbeSize` budget but did not trace the full upload path.

### Gaps
- Per-GPU preset values, because `Scripts/GLSettings` is absent from Git.
- Whether SKA uses CPU skinning everywhere. A search for hardware or vertex-program terms in `RMRender.cpp` found nothing, which suggests CPU skinning but is not a full trace.

## What did Serious Engine 3/3.5/4 add for huge enemy counts?

### Takeaway
Croteam's public statements give scale claims but no technical method that I could find. The "Legion System" is described as a trick that is used only in some situations. The better-documented Croteam contributions are Vulkan support and frame-pacing analysis, and those are not crowd techniques.

### Cited Findings

**Legion System (Serious Sam 4, 2020)**
- [CROTEAM] Dean Sekulić, graphics engineer, called it a "nice trick" enabling "thousands and thousands, hundreds of thousands" on screen — [PC Gamer](https://www.pcgamer.com/au/serious-sam-4-threatens-to-put-100000-enemies-on-your-screen-at-once) (search excerpt), [GINX](https://www.ginx.tv/en/serious-sam-4-release-window-legion-system)
- [CROTEAM] Per GINX, Sekulić says the original target was "like 10,000 enemies" and the team was "really pleasantly surprised with how fast it can run" — [GINX](https://www.ginx.tv/en/serious-sam-4-release-window-legion-system)
- [CROTEAM, snippet] A TweakTown interview says it renders "100,000 enemies on screen without actually taking a hit on hardware" (full quote seen only in a search excerpt, and the page returned 403) — [TweakTown](https://www.tweaktown.com/articles/9568/serious-sam-interview-croteam-amps-chaos-to-11/index.html). Treat this as marketing wording.
- [CROTEAM, snippet] Davor Hunski reportedly called it "a very situational feature" that "won't be used on the majority of levels" — [SegmentNext](https://segmentnext.com/2020/06/03/Serious-sam-4-interview/) (now HTTP 410; seen only in a search excerpt).
- [COMMUNITY] TV Tropes says it appears briefly in the prologue and final level, and that most units are "just an illusion for the spectacle" — [TV Tropes](https://www.tvtropes.org/pmwiki/pmwiki.php/VideoGame/SeriousSam4). A Steam user reports the same limited use — [Steam thread](https://steamcommunity.com/app/257420/discussions/0/2840039920182894994)
- [PRESS] A review noted Legion spawning differs from regular spawning, without detail — [Gaming Trend](https://gamingtrend.com/reviews/seriously-fun-seriously-silly-serious-sam-4-review/) (search excerpt; attribution from the search summary is not fully certain).

**SE3 and SE3.5 (Serious Sam HD, Serious Sam 3: BFE)**
- [PRESS] Reviews describe hordes, including "500 enemies between you and... the next 500 enemies", without explaining the technology — [Neowin](https://www.neowin.net/reviews/serious-sam-3-review-keeps-on-shooting/) (search excerpt; attribution from the search summary is not fully certain).
- [COMMUNITY] The Serious Sam wiki describes engine versions, but the fetch returned HTTP 402 — [Fandom wiki](https://serioussam.fandom.com/wiki/Serious_Engine). Search excerpts conflict on whether BFE used DirectX 11 or only DirectX 9, so I treat this as unresolved.

**Settings design in later engines**
- [COMMUNITY] A Steam guide lists Talos Principle (Serious Engine 4 era) settings in groups by bottleneck: CPU SPEED (multi-threaded rendering, rendering distance, dynamic shadow distance, debris shadows, max active sounds), GPU SPEED (Max 3D Rendering MPix with rescaling filter, MSAA, FXAA, SSAO, shadows, crumbs density and distance bias, LOD bias, particle density), and GPU MEMORY (texture size and compression, streaming buffer, level caching) — [Steam guide](https://steamcommunity.com/app/257510/discussions/0/364040961449374759)
- [COMMUNITY] That guide lists the API choices as D3D9, D3D11, OpenGL and Vulkan, and describes the MPix option as a fixed cap with upscaling. It mentions no auto-detection or dynamic resolution — [Steam guide](https://steamcommunity.com/app/257510/discussions/0/364040961449374759)
- [PRESS] A BFE performance analysis mentions a "Compressed" GPU Memory option — [DSOGaming](https://www.dsogaming.com/pc-performance-analyses/serious-sam-3-bfe-pc-performance-analysis/) (search excerpt).

**Vulkan and frame pacing**
- [CROTEAM] Sekulić, Feb 2016: the Talos port is a proof of concept. Vulkan requires "several changes in the higher level graphics system", and "some work is left to be done" in low-level and high-level optimizations — [Croteam](https://www.croteam.com/house-interview-vulkan/)
- [CROTEAM] Croteam says it was on the Vulkan Advisory Panel — [Croteam](https://www.croteam.com/house-interview-vulkan/)
- [CROTEAM, joke] Sekulić's "five minutes" remark about adding Vulkan is humor, not an engineering estimate — [GamingOnLinux](https://www.gamingonlinux.com/2016/02/croteam-confirm-the-talos-principle-will-be-their-first-vulkan-title/page=1/)
- [CROTEAM] Developer post: SS4 would not use the Talos engine version but "a much improved version" — [GamingOnLinux, 26 Jan 2016](https://www.gamingonlinux.com/2016/01/talos-principle-serious-sam-3-getting-vulkan-updates-serious-sam-4-will-be-on-a-much-improved-engine/page=1)
- [CROTEAM] Alen Ladavac's GDC 2018 session, "The Elusive Frame Timing: A Case Study for Smoothness Over Speed", covers a lesser-known micro-stutter cause on single-GPU systems and prototype fixes — [GDC Vault](https://gdcvault.com/browse/gdc-18/play/1025031)
- [PRESS, snippet] Phoronix reports the fix as measuring when a frame reaches the display rather than how long it took to render. Croteam tested `VK_GOOGLE_display_timing` in Talos and Serious Sam Fusion — [Phoronix, July 2018](https://www.phoronix.com/news/Frame-Timing-Game-Stuttering), [Phoronix, March 2018](https://www.phoronix.com/news/GDC-Vulkan-Stutter-Frames)
- [PRESS] Croteam showed a "Heartbeat" stutter video alongside a smooth one. Extension support was described as limited — [GamingOnLinux, July 2018](https://www.gamingonlinux.com/2018/07/the-cto-of-croteam-has-written-up-a-post-about-the-elusive-frame-timing). Primary post: [Ladavac, Medium](https://medium.com/@alen.ladavac/the-elusive-frame-timing-168f899aec92) (HTTP 403 to my fetcher, so not read).

### Inferences
- "Trick", "illusion", "situational" and separate spawning fit a cheap crowd layer distinct from full enemies. Croteam's method is unconfirmed. Do not cite Legion as instancing, impostors or GPU simulation.

### Gaps
- No Croteam technical breakdown of Legion found, and no real enemy cap.
- No documented method for how SE3/3.5 handled large counts. I found no talk or paper.
- Sekulić's Reboot Develop 2016 Vulkan lecture was announced, but I found no slides. His forum post with Vulkan details was not retrievable.
- One search summary described a three-stage Vulkan plan (wrapper, then multithreading, then precaching states), attributed to a Croteam developer via GamingOnLinux. Two GamingOnLinux pages I opened did not contain it. **Treat as unverified, do not cite.**
- Ladavac's detailed reasoning needs a human read of the Medium post or GDC talk.

## Which ideas are worth copying today?

### Takeaway
Copy the structure: tiers chosen by measurement, per-system LOD multipliers, a fixed simulation rate with interpolation, settings grouped by bottleneck, explicit fallbacks, and smooth pacing over peak fps. Do not copy SE1's legacy GL code or its GPU-name table as the only detection method. These recommendations are my inferences. I did not audit moonBASIC's existing quality code for this note.

### Cited Findings
Evidence behind each idea is cited in the sections above. Everything below is my judgment:

### Inferences

| Idea | Evidence | Adaptation for moonBASIC (Go, G3N, GL 3.3 forward) |
|---|---|---|
| Benchmark-based tier pick, with a table only for known exceptions | SS2 fillrate test [PRESS]; SE1 table disables itself on unknown GPUs [SOURCE] | Run a short timed fill and geometry test on first launch or GPU change. Keep a small denylist for broken drivers. |
| Rerun detection only when the GPU changes; never overwrite "custom" | `ApplyGLSettings` [SOURCE] | Store the renderer string and last tier in a config file. |
| Global LOD multiplier and offset per system | `mdl_fLODMul`, `ska_fLODMul`, `wld_iDetailRemovingBias` [SOURCE] | One multiplier and bias each for models, skeletons, terrain and particles. Scales without re-authoring assets. |
| Distance disappearance | `mdl_iLODDisappear` [SOURCE] | Cull past the last LOD. Offer a bias for important objects. |
| Prioritize geometry LOD and dynamic shadows | SS2 measurements [PRESS] | Expose shadows and LOD first, then measure on moonBASIC's own scenes rather than assuming the 2006 ranking holds. |
| Shadow tiers | SE1 tiers [SOURCE]; SS2 one sun, 512 max, rest baked [PRESS] | Offer none, blob, one directional light and more. Default the low tier to a single sun with baked or blob shadows. |
| Fixed-rate simulation with interpolated rendering | 20 Hz tick and lerp factor [SOURCE] | Fixed timestep for logic, with interpolation for rendering. 20 Hz is SE1's number, and input-heavy games may want more. |
| Far terrain batching | Lowest-LOD tiles share one top map [SOURCE] | Far terrain in one draw with one baked texture. Replace CPU tile regeneration with static index buffers and shader geomorphing. |
| Geomorph and border stitching | `tt_fLodLerpFactor`, border routines [SOURCE] | Keep the concept to avoid popping. |
| Texture and shadow caches with budgets | `shd_fCacheSize`, flush delay, flats [SOURCE]; streaming buffer in Talos [COMMUNITY] | Use a VRAM budget with LRU eviction, and collapse single-color lightmaps to a constant. |
| Settings grouped by bottleneck | Talos groups [COMMUNITY] | Group options as CPU, GPU and VRAM so a user knows which to lower. |
| Pixel cap instead of a resolution list | Max 3D Rendering MPix [COMMUNITY] | Render at a capped megapixel count and upscale, keeping the HUD native. Works across old and new displays. A dynamic version would be a moonBASIC addition, not something I found in Croteam's documentation. |
| Probe the feature, then fall back | SS2 FP16 fallback and the ATI filtering bug [PRESS] | Test render-target and filtering behavior at runtime and fall back to LDR. GL 3.3 may guarantee these formats, but check real drivers (from memory, verify in the spec). |
| Named frame-stage profiler | `CGfxProfile` [SOURCE] | Named stage timers and upload counters help size tiers. |
| Pacing over raw fps | Ladavac GDC 2018 [CROTEAM] | Treat smoothness as a goal. GL 3.3 has no equivalent I can cite to `VK_GOOGLE_display_timing`, so smoothing frame deltas is an approximation to test, not a Croteam-endorsed fix. |
| Two-tier crowds, used sparingly | Legion described as situational [CROTEAM, snippet] | Full AI for near enemies and a cheap instanced layer for distant spectacle. The method here is my suggestion, not Croteam's. |

Avoid or treat with caution:
- SE1's `ogl_*` and fixed-function-era code, because GL 3.3 core does not have those paths.
- SS2-style parallel SM 1.x and 2.0 paths. moonBASIC's GL 3.3 floor already excludes that hardware, so low tiers should drop features within one shader family.
- Hidden API switches. iXBT attributes OpenGL's speed in SS2 to missing features, so a "faster" path can mislead when feature sets differ.
- Runtime auto-quality as "the Serious Engine way". I found no evidence that SE1 did it, and the sources I read do not describe it in SS2.

### Gaps
- No moonBASIC benchmark data here. Every cost figure comes from 2006-era hardware or from Croteam marketing.
- Whether any adaptive runtime scaling exists in SE3 or SE4 is undocumented in what I could read. PCGamingWiki was blocked, and a check by someone with the games or a browser would close this.
