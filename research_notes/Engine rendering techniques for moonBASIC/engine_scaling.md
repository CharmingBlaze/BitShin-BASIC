# How shipping engines scale rendering across old and new hardware (Godot 4, Unity, Unreal, Source, id Tech), for moonBASIC

Scope and evidence notes for the report writer:
- Target engine: moonBASIC, a Go runtime on G3N (OpenGL 3.3 core, forward), with quality tiers, FSR 1 dynamic resolution, CSM, GTAO, FXAA, bloom, automatic batching and linear lighting. Hardware target is about 2016 onward on Windows, Linux and macOS.
- Evidence quality varies by engine. Godot, Unity and Unreal are mostly current official docs. **Source is thin**: the Valve Developer Community wiki returned HTTP 403 to direct fetches, so those claims rest on search-result extracts of the wiki and are marked. **id Tech is mostly secondary**: a frame-capture analysis and press. The primary papers and slides were not read (see Gaps).
- Engine source code on GitHub was not read directly. All engine behaviour below comes from docs, talks, or analyses.
- Web pages were read through a summarising fetch tool. Exact wording and table contents should be re-checked before being quoted as verbatim. One such summary was wrong in a way that matters; see the OpenGL version conflict in Q5.

## How do Godot 4's Compatibility, Mobile and Forward+ renderers differ, what does Godot do on OpenGL 3.3, and how does it do 3D resolution scaling and FSR?

### Takeaway
Godot ships three separate renderers rather than one scalable one. Its OpenGL 3.3 path (Compatibility) is deliberately reduced: non-clustered forward, multi-pass for shadowed lights, tonemapped sRGB output with no HDR, no compute. It also gets only bilinear 3D scaling, with no FSR 1, so moonBASIC's GL 3.3 path already has more upscaling and post-processing than Godot's equivalent.

### Cited Findings

Renderer structure (docs for Godot "stable"; the fetched page mentions behaviour since 4.4):
- Forward+ and Mobile use the RenderingDevice backend with Vulkan, Direct3D 12 or Metal drivers. Compatibility uses OpenGL and does not use RenderingDevice. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- The docs give a cost model. Compatibility has low base cost and high scaling cost, Mobile medium for both, Forward+ the highest base cost and low scaling cost. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- Lighting differs by renderer. Compatibility is forward single-pass, but lights with shadows use multi-pass with less accurate blending. Mobile is forward single-pass. Forward+ is clustered forward. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- Omni/spot light limits: Compatibility 8 per mesh (can be increased), Mobile 8 per mesh and 256 per view, Forward+ 512 per cluster (can be increased). Directional lights are limited to 8 on all three. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- Reflection probes: Compatibility 2 per mesh, Mobile 8 per mesh, Forward+ unlimited. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- VoxelGI, SSIL, SDFGI, volumetric fog and SSR are Forward+ only. SSAO is on Compatibility and Forward+, not Mobile. FXAA and SMAA are on Mobile and Forward+ only. TAA and FSR2 are Forward+ only. MSAA 3D and SSAA work on all three. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- Compute shaders are unsupported on Compatibility, supported with a penalty on older devices on Mobile, and fully supported on Forward+. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- Depth: Compatibility is 24-bit without reverse Z, Mobile 24-bit with reverse Z, Forward+ 32-bit with reverse Z. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html). The architecture page says 3D depth on the desktop renderer is 24-bit unorm with a 32-bit float fallback, which does not match cleanly. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- Godot's renderer fallback chain (4.4+): on Forward+ or Mobile, if Vulkan is unsupported it tries Direct3D 12, and vice versa. If neither is available it falls back to Compatibility, and the project "may then look different". This can be disabled via Rendering > Rendering Device > Fallback to OpenGL 3. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)

What Godot does on OpenGL 3.3 specifically:
- Compatibility targets OpenGL ES 3.0. Desktop uses OpenGL 3.3 Core Profile, web uses WebGL 2.0, `--rendering-driver opengl3_es` runs native ES on drivers such as Mesa, and ANGLE is optional on Windows and macOS. Its core shaders are written separately in GLSL. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- Lighting is non-clustered forward. The first pass takes any number of unshadowed lights plus at most one shadowed directional. Each later pass takes one omni, one spot and one shadowed directional. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- Shadowed lights blend in sRGB space and unshadowed lights in linear space, so scenes look different. Godot's docs tell users to design around this. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- Shadow performance "scales poorly" with many casters. Guidance is to keep few shadowed lights visible, spread them apart, and have each object touched by one or two. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- Colour output is tonemapped and stored in sRGB with no HDR, which avoids a separate tonemapping pass and prevents banding at the low bit depth. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- **Source conflict on buffer format:** the architecture page says Compatibility uses R10G10B10A2 UNORM ([Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)). The renderers page says RGBA8 ([Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)), and a Godot PR discussion also says RGBA8 ([godot#85120](https://github.com/godotengine/godot/pull/85120)). The R10G10B10A2 line may be misattributed by the summariser. Check `drivers/gles3` in source if the exact format matters.
- Compatibility has no decals and most post-processing is unavailable. LightmapGI rendering is supported, but baking requires RenderingDevice hardware. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html); [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)

Shared design reasoning worth copying:
- Godot chose forward over deferred because of its performance-to-flexibility tradeoff and because deferred needs hacks for MSAA. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- Core shaders apply to every material so must stay simple. `if` branching can raise VGPR usage even when all pixels take the same path, and each `#if` boolean can double the variants a scene needs. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- On the RD renderers, SSAO, SSIL and SSR run at half resolution by default. Instancing groups identical opaque and alpha-tested objects, with each instance still culled individually. Alpha-blended objects are never instanced. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)
- The Mobile renderer can beat Forward+ in simple scenes on desktop. It uses only raster shaders and defaults to R10G10B10A2, which halves bandwidth compared with RGBA16F. — [Godot: Internal rendering architecture](https://docs.godotengine.org/en/stable/engine_details/architecture/internal_rendering_architecture.html)

3D resolution scaling and FSR:
- Modes are bilinear (default, the fallback, all renderers), FSR 1.0 (Forward+ only) and FSR 2.2 (added 4.2, Forward+ only). — [Godot: Resolution scaling](https://docs.godotengine.org/en/stable/tutorials/3d/resolution_scaling.html)
- **Compatibility gets only bilinear scaling and the scale setting**: no FSR 1.0, FSR 2.2, sharpness or mipmap bias. — [Godot: Resolution scaling](https://docs.godotengine.org/en/stable/tutorials/3d/resolution_scaling.html)
- The docs warn FSR 1.0's cost may outweigh its benefit on very slow GPUs. It works best with TAA or MSAA, because FXAA adds blur without temporal information. — [Godot: Resolution scaling](https://docs.godotengine.org/en/stable/tutorials/3d/resolution_scaling.html)
- FSR sharpness range is 0.0 (sharpest) to 2.0 (least), default 0.2. RCAS cannot be used at native scale without EASU. The workaround is a 0.99 scale. — [Godot: Resolution scaling](https://docs.godotengine.org/en/stable/tutorials/3d/resolution_scaling.html)
- Mip bias formula: `log2f(min(scaling_3d_scale, 1.0)) + custom_texture_mipmap_bias`. Offsets of -0.25 for FXAA, -0.5 for TAA, and -0.75 when both are on are applied before the resolution offset. Forward+ and Mobile only. Recommended manual values are -0.5 to 0.0. Negative bias can look grainy and reduce performance. — [Godot: Resolution scaling](https://docs.godotengine.org/en/stable/tutorials/3d/resolution_scaling.html)
- The scale is runtime-settable through `scaling_3d_scale`. If lowering the scale barely helps, the docs point to draw calls or costly effects such as SSAO. The docs also say scaling does not replace lowering settings and recommend exposing both. — [Godot: Resolution scaling](https://docs.godotengine.org/en/stable/tutorials/3d/resolution_scaling.html)

### Inferences
- FSR 1 is not unavailable on Godot's GL path for a technical reason visible in these docs. The docs only state it is unsupported. AMD's EASU and RCAS are described as passes on the current frame without history. So moonBASIC's FSR 1 on GL 3.3 is a capability Godot's equivalent renderer does not offer, not a mistake.
- moonBASIC can copy Godot's mip-bias formula directly. It is a one-line change with a known recipe, and it is a gap in Godot's own Compatibility renderer. With sampler objects (see Q5), it can be applied through `GL_TEXTURE_LOD_BIAS` without touching shaders. Linear lighting plus post-tonemap FSR means moonBASIC must run FSR after tonemapping (see Q5).
- Godot's multi-pass shadowed lights and the resulting sRGB/linear mismatch is a warning. moonBASIC's linear pipeline on an HDR buffer already avoids that mismatch, and should not trade it away for bandwidth on 2016+ hardware.
- Godot's cost model (low base, high scaling) is the likely shape of moonBASIC's forward path too: cheap for small scenes, poor with many lights. A hard per-object light limit, as Godot uses, is the simple control.

### Gaps
- Godot's Compatibility minimum hardware list and driver blocklist were not found. They would need `drivers/gles3` and `main.cpp` in source.
- Exact Compatibility colour format remains unresolved (see conflict above).

## How do Unity (URP quality settings, dynamic resolution, SRP batcher) and Unreal (scalability groups, dynamic resolution, mobile/ES3 path) structure their quality scaling?

### Takeaway
Both separate "named quality bundles" from the runtime controller. Unity bundles settings into per-quality URP Assets. Unreal maps `sg.*` groups 0-3 onto cvars in an ini file, with device profiles layered on top. Both limit dynamic resolution to APIs they trust for GPU timing (Metal, Vulkan, DX12), not OpenGL. Unreal's documented DRS heuristic (history, headroom, change period, panic) is the most directly copyable item.

### Cited Findings

Unity:
- URP render scale and upscale filter are set on the URP Asset under Quality, not in Project Settings. — [Unity: BiRP quality settings location](https://docs.unity3d.com/Manual/urp/birp-onboarding/quality-settings-location.html)
- Unity's onboarding page gives "Low" and "High" presets that both use Render Scale 1, while anti-aliasing is off for Low and 2x MSAA for High. So its stock presets do not lower resolution for Low. — [Unity: quality presets](https://docs.unity3d.com/6000.5/Documentation/Manual/urp/birp-onboarding/quality-presets.html) (from a search extract, page not read in full)
- Upscaling options by pipeline (Unity 6.0): URP has bilinear, nearest, STP and FSR 1. HDRP has Catmull-Rom, CAS, TAAU, STP, FSR 1, FSR 2 and DLSS. FSR 1 uses only the frame buffer. STP cannot be combined with dynamic resolution in URP, or with software dynamic resolution in HDRP. — [Unity 6.0: resolution scale introduction](https://docs.unity.com/en-us/engine/6000.0/manual/cameras/resolution-scale/introduction)
- Dynamic resolution (2022.3 docs) is supported on iOS/macOS/tvOS (Metal only), Android (Vulkan only), and Windows Standalone and UWP (DirectX 12 only). OpenGL is not listed. — [Unity 2022.3: Dynamic resolution](https://docs.unity3d.com/2022.3/Documentation/Manual/DynamicResolution.html)
- Unity allocates render targets at full resolution and uses a portion, so memory stays at full size. Targets carry a `DynamicallyScalable` flag, and `ScalableBufferManager.ResizeBuffers` applies the scale. — [Unity 2022.3: Dynamic resolution](https://docs.unity3d.com/2022.3/Documentation/Manual/DynamicResolution.html)
- `FrameTimingManager` reports `gpuFrameTime` and `cpuFrameTime`, but Unity's example does not adjust scale automatically. The application writes the controller. — [Unity 2022.3: Dynamic resolution](https://docs.unity3d.com/2022.3/Documentation/Manual/DynamicResolution.html)
- The SRP Batcher reduces render-state changes between draw calls, not the draw call count. It keeps material data persistent in GPU memory and uses a dedicated large per-object constant buffer, so only engine per-object data is handled per draw. — [Unity: SRP Batcher](https://docs.unity.com/en-us/engine/current/manual/analysis/graphics-performance-profiling/in-urp/reduce-draw-calls-urp/srpbatcher/srpbatcher)
- Batches are keyed by shader variant, and materials can differ freely within a variant. Unity's advice is to use as few variants as possible. — [Unity: SRP Batcher](https://docs.unity.com/en-us/engine/current/manual/analysis/graphics-performance-profiling/in-urp/reduce-draw-calls-urp/srpbatcher/srpbatcher)
- Shaders must declare engine properties in a single `UnityPerDraw` constant buffer and material properties in a single `UnityPerMaterial` buffer. MaterialPropertyBlock, particles and GPU instancing are incompatible with it. — [Unity 2021.3: SRP Batcher](https://docs.unity3d.com/2021.3/Documentation/Manual/SRPBatcher.html)

Unreal:
- Scalability groups and what they drive: `sg.PostProcessQuality` (motion blur, AO levels, DoF), `sg.ShadowQuality` (`r.ShadowQuality`, `r.Shadow.CSM.MaxCascades`, `r.Shadow.MaxResolution`, `r.Shadow.RadiusThreshold`), `sg.TextureQuality` (`r.Streaming.MipBias`, `r.MaxAnisotropy`, `r.Streaming.PoolSize`), and `sg.EffectsQuality` (`r.SSR`, `r.SceneColorFormat`, `r.DetailMode`, translucency volume). Foliage quality sets `foliage.DensityScale` and `grass.DensityScale`. — [Epic: Scalability reference](https://dev.epicgames.com/documentation/unreal-engine/scalability-reference-for-unreal-engine?lang=en-US)
- Levels are 0 (Low) to 3 (Epic). Defaults live in `BaseScalability.ini`. Projects override them in `DefaultScalability.ini` with sections like `[FoliageQuality@0]`. Also relevant are `r.ScreenPercentage` and `r.ViewDistanceScale`. — [Epic: Scalability reference](https://dev.epicgames.com/documentation/unreal-engine/scalability-reference-for-unreal-engine?lang=en-US)
- `r.DynamicRes.OperationMode`: 0 disabled (default), 1 follows Game User Settings, 2 forced on. Defaults are min 50%, max 100%, budget 33.3 ms. — [Epic: Dynamic resolution](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Heuristic controls: `TargetedGPUHeadRoom` (percent of budget), `ChangePercentageThreshold` (minimum change before resizing), `MinResolutionChangePeriod` (minimum frames between changes, justified by noise, temporal jitter and AA divergence), `HistorySize` (too short is noisy, too long is laggy), and `CPUBoundScreenPercentage` (target when CPU-bound). — [Epic: Dynamic resolution](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Panic switch: `MaxConsecutiveOverbudgetGPUFrameCount` frames over budget lowers resolution immediately and resets the history. The doc's example fires after more than two such frames. — [Epic: Dynamic resolution](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Dynamic resolution is allowlisted for consoles and PC with DX12 and Vulkan. Other platforms are blocked by default because enabling it can cause wrong GPU timings, needless resolution drops, or frame drops. OpenGL is not mentioned. — [Epic: Dynamic resolution](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Render targets are allocated at `MaxScreenPercentage`. Pixel count is proportional to percentage squared. Gaussian DoF and distance field AO have issues with resolution changes. — [Epic: Dynamic resolution](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- The heuristic is replaceable via `IDynamicResolutionState` and `ISceneViewFamilyScreenPercentage`. Default logic is in `DynamicResolution.cpp`. — [Epic: Dynamic resolution](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Mobile path: ES3.1 became the Android default in 4.23 and ES2 was removed in 4.25. Metal 2.0 is the iOS feature level, and Android Vulkan is a high-end option. Mobile deferred was marked experimental at that time (`r.Mobile.ShadingPath=1`). — [Epic tech blog, Jan 2020](https://www.unrealengine.com/tech-blog/updates-to-mobile-rendering-support-in-unreal-engine-4-24-and-4-25) (UE 4.24/4.25-specific, from a search extract); [Epic: Feature levels (4.27)](https://dev.epicgames.com/documentation/unreal-engine/feature-levels-and-rendering-modes?application_version=4.27)
- Mobile dynamic resolution is "API-compatible", but view size differing from render target size is not maintained. — [Epic: Dynamic resolution](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Device profiles are the recommended scalability route on mobile and for PC/Mac scalability buckets: `[iPhone5S DeviceProfile]` sections with `BaseProfileName` inheritance and `+CVars=` lines. Memory buckets (Largest to Tiniest) allow `+CVars_Tiniest=` overrides, with `r.Streaming.PoolSize` as the doc's example. — [Epic: Device profiles](https://dev.epicgames.com/documentation/en-us/unreal-engine/setting-up-device-profiles-in-unreal-engine)

### Inferences
- Both engines' DRS allowlists exclude OpenGL, likely from concerns about timing accuracy. Epic names wrong GPU timings as the risk. moonBASIC must therefore treat GL timer data defensively: a sanity check and a fallback to CPU frame time, or disable DRS when timings look wrong.
- Unreal's allocate-at-max approach also matches Unity's. moonBASIC should allocate scene targets at max scale and render a viewport subset, rather than reallocating when scale moves. The exception would be a change beyond a threshold, which is what `ChangePercentageThreshold` governs in UE.
- Unreal's structure maps neatly onto a small engine: one file, a section per group and level, each line a named setting. moonBASIC's quality tiers could be expressed as data rather than code, using separate groups for shadows, post, textures and effects. Users then get per-group overrides.
- The SRP Batcher's core idea (persistent per-material data, one large per-object buffer, batching by shader variant) is feasible on GL 3.3 with UBOs and `glBindBufferRange` offsets. moonBASIC's automatic batching could sort by program, then material. This is a design inference, not a documented Unity GL path.
- A simple starting controller modelled on Unreal: a frame budget, about 10% headroom, short history, change only every N frames, panic on 2-3 over-budget frames. Keep scale proportional to the square root of the cost ratio, since pixels go with percentage squared. The 10% is a suggestion, not a documented default.

### Gaps
- Unity's official list of graphics APIs for the SRP Batcher: pages read for current and 2021.3 do not list APIs. Whether it works on GL is unverified.
- Unreal Cinematic level 4 is not defined on the fetched scalability page, although "levels 0-4" is common community usage. Unverified here.
- Default values for most `r.DynamicRes.*` cvars are not on the page; read `DynamicResolution.cpp` for them.
- URP quality-level contents per tier were only partly read.

## What did Source (config-by-hardware, dxlevel, cvars) and id Tech (Rage/Doom virtual texturing, dynamic resolution, draw submission) do?

### Takeaway
Source is the simple, table-driven model: a GPU-ID config file picks a feature level, later replaced by memory detection plus shader and effect detail options. id Tech pursued a different goal, holding 60 Hz: Rage used virtual texturing and an auto-balancer, and Doom 2016 used clustered forward, a feedback-driven tile cache, and shadow caching. The most copyable id ideas are shadow caching and half or quarter-resolution effects, not virtual texturing.

### Cited Findings

Source (wiki extracts only; direct fetches were blocked):
- Older Source games use `dxsupport.cfg` to pick a DirectX level per card. For example, GeForce FX 5x00 cards were forced to DX8 level despite supporting DX 9.0a. — [VDC: DirectX Versions / Dxsupport.cfg](https://developer.valvesoftware.com/wiki/Dxsupport.cfg)
- `mat_dxlevel` does not change the actual Direct3D version. It is a feature-level selector; the game uses the D3D9 API with feature level 8.0. It is better set via `+mat_dxlevel` launch parameter or config. — [VDC: Mat_dxlevel](https://developer.valvesoftware.com/wiki/Mat_dxlevel); [VDC: Dxlevel](https://developer.valvesoftware.com/wiki/Dxlevel)
- From the Left 4 Dead branch onward, DX levels were dropped in favour of a shader and effect detail option plus automatic GPU memory size detection. — [VDC: Dxlevel](https://developer.valvesoftware.com/wiki/Dxlevel)
- Vendor and device IDs drive decisions. One example: `-force_vendor_id 0x10DE -force_device_id 0x1088` tricks a game into High shadows on Intel by posing as a GTX 590. — [VDC: Dxlevel](https://developer.valvesoftware.com/wiki/Dxlevel)
- `mat_info`, `mat_configcurrent` and `-autoconfig` are useful diagnostics. — [VDC talk page](https://developer.valvesoftware.com/wiki/Talk:DirectX_Versions)
- A Wine bug comment says Portal's "Recommended settings" came from a predefined list once in `dxsupport.cfg`. This is a secondary source. — [Wine bug comment](https://mail.winehq.org/mailman3/hyperkitty/list/wine-bugs@winehq.org/message/KAS3FNZFD3R5C5B5ZLQG7SA245XHT7GE)

id Tech 5 / Rage:
- id's target was 60 fps on PS3, Xbox 360, and the recommended PC and Mac specification, per Carmack. — [Shacknews](https://www.shacknews.com/article/53713/carmack-doom-4-sports-better) (secondary)
- Rage PC shipped with very limited graphics options and a built-in auto-balancer. The balancer assesses the system and scales visuals, and reviewers said it did not scale high enough. — [Bit-Tech: Rage technical analysis](https://bit-tech.net/reviews/gaming/pc/rage-technical-analysis/2/); [DSOGaming](https://www.dsogaming.com/news/rage-no-advanced-graphics-options-for-the-pc-version/). The first major Rage PC launch problems were reported in the press ([Kotaku](https://kotaku.com/why-was-the-pc-launch-of-rage-such-a-cluster-5847761?tag=rage)); that article's contents were not read.
- Virtual textures map through a page table texture to physical tile storage, with only part resident. — [van Waveren, SIGGRAPH 2012 course slides](https://www.quaddicted.com/webarchive/mrelusive.com_20170131/publications/presentations/2012_siggraph/Virtual_Texturing_in_Software_and_Hardware_final.pdf) (search extract); [van Waveren, Software Virtual Textures, 2012](https://www.quaddicted.com/webarchive/mrelusive.com_20170131/publications/papers/Software-Virtual-Textures.pdf) (search extract; PDF body could not be decoded). Original talk was "id tech 5 challenges", SIGGRAPH 2009. — [SIGGRAPH archive](https://history.siggraph.org/?p=129359)

id Tech 6 / Doom 2016 (third-party frame capture, Vulkan on GTX 980 at Ultra):
- Clustered forward with 3072 clusters (16 x 8 x 24, log depth), up to 256 lights, 256 decals and 256 cubemaps per cluster, built on the CPU. — [Courrèges: Doom 2016 graphics study](http://www.adriancourreges.com/blog/2016/09/09/doom-2016-graphics-study)
- Hybrid: the forward pass also writes normal and specular G-buffers. A depth pre-pass with `EQUAL` test avoids overdraw. — [Courrèges](http://www.adriancourreges.com/blog/2016/09/09/doom-2016-graphics-study); id described it as a hybrid of opaque forward and deferred — [80.lv summary of the SIGGRAPH 2016 talk](https://80.lv/articles/idtech-666-the-secret-of-dooms-render)
- Virtual texturing: about 16k x 8k textures with 128 x 128 tiles, a 160 x 120 feedback buffer, and reactive streaming. — [Courrèges](http://www.adriancourreges.com/blog/2016/09/09/doom-2016-graphics-study)
- Shadows share an 8k x 8k atlas with most maps reused from the previous frame. Only dynamic-affected maps are regenerated. — [Courrèges](http://www.adriancourreges.com/blog/2016/09/09/doom-2016-graphics-study); id also uses caching and a "smart composite" — [80.lv](https://80.lv/articles/idtech-666-the-secret-of-dooms-render)
- SSAO is half-res, bloom is quarter-res, DoF is half-res, and particle lighting uses fixed-size tiles in an atlas independent of output resolution. — [Courrèges](http://www.adriancourreges.com/blog/2016/09/09/doom-2016-graphics-study)
- Draw submission at capture: 1331 draw calls, 132 textures and 50 render targets. UI alone is about 120 draws unbatched. Occlusion culling is Umbra plus GPU queries read in later frames to avoid stalls. — [Courrèges](http://www.adriancourreges.com/blog/2016/09/09/doom-2016-graphics-study)
- About 100 unique shaders were used. Post runs async on consoles and with Vulkan. — [80.lv](https://80.lv/articles/idtech-666-the-secret-of-dooms-render)
- Doom offers OpenGL and Vulkan backends on PC. — [Courrèges](http://www.adriancourreges.com/blog/2016/09/09/doom-2016-graphics-study)
- Consoles used dynamic resolution: PS4 from 1600 x 1080 to 1920 x 1080, and Xbox One from 1350 x 1080 to 1920 x 1080, per Digital Foundry-derived coverage. — [GamingBolt](https://gamingbolt.com/doom-graphics-analysis-ps4-leads-xbox-one-pc-by-far-the-best-version) (secondary)

id Tech 7 / later:
- Doom Eternal remains a forward pipeline with mesh decals added. — [Habr analysis](https://habr.com/ru/post/518724) (third-party, extract only)

### Inferences
- Source's arc is instructive: a card table proved brittle over time and it moved to memory detection plus a few detail options. For moonBASIC, a broad GPU table can be replaced by a small blocklist and vendor heuristic (see Q4).
- Doom's asset-independent scaling lessons fit moonBASIC: GTAO at half-res on all tiers but High, quarter-res bloom, and shadow caching for static cascades or spot lights. Cached static shadow maps require only render-to-texture, so they are possible on GL 3.3.
- Doom's depth pre-pass with `EQUAL` testing is a cheap forward-renderer win, especially since moonBASIC's GTAO needs depth anyway.
- Reading occlusion queries in later frames avoids stalls, and the same discipline applies to the timer queries used for DRS.
- 1331 draws at ultra on a 2016 frame is a rough reference for how many draws a shipping game used. It is one capture, on Vulkan, so it is not a GL 3.3 target.

### Gaps
- No primary source found for Doom's dynamic resolution algorithm, or whether PC used the same one. The SIGGRAPH 2016 "The Devil is in the Details" slides were not read.
- No primary source found on Rage's dynamic resolution or on GPU transcoding details. The 2009 "id tech 5 challenges" notes were not retrieved. Rage's PC API version (OpenGL 3.3?) is unverified; a forum post and a 2007 note disagreed.
- The "Rendering the Hellscape of Doom Eternal" SIGGRAPH 2020 talk was not found, so draw submission in id Tech 7 is unverified. A GDC session on [Doom: The Dark Ages](https://schedule.gdconf.com/session/rip-tear-breaking-down-the-renderingofdoom-the-dark-ages/915274) exists but was not read.
- `dxsupport.cfg` key syntax was not recovered. Source's autoconfig logic details are not verified.

## How does each engine pick default settings for a machine: benchmark, GPU database, or runtime feedback?

### Takeaway
All three approaches are in use, usually combined. Unreal runs a short synthetic benchmark on demand and maps a score to levels via thresholds. Source used a GPU database and later memory detection. Godot and Unity documented no automatic default in the pages read, leaving it to the application. Rage used a built-in auto-balancer. Runtime feedback (dynamic resolution) is a separate layer that handles what remains.

### Cited Findings
- **Unreal, benchmark:** `UGameUserSettings::RunHardwareBenchmark` returns CPU and GPU indices where 100 is a reference "average good" machine. `ApplyHardwareBenchmarkResults()` applies them and results are stored in `GameUserSettings.ini`. It freezes the game during the run. `scalability auto` and `SynthBenchmark` expose it in the console. Version-specific: the source article is UE4-era. — [Looman: Optimal graphics settings](https://tomlooman.com/unreal-engine-optimal-graphics-settings/) (community, not Epic)
- **Unreal, thresholds:** `PerfIndexThresholds_*` keys under `[ScalabilitySettings]` in `(Base)Scalability.ini` map the index to levels per group. Looman recommends tuning late using QA machine data, placing min-spec scores at the Low threshold. — [Looman](https://tomlooman.com/unreal-engine-optimal-graphics-settings/)
- **Unreal, device database:** device profiles by platform device, with selectors `iOSDeviceProfileSelector` and `AndroidDeviceProfileSelector`, plus memory buckets. — [Epic: Device profiles](https://dev.epicgames.com/documentation/en-us/unreal-engine/setting-up-device-profiles-in-unreal-engine)
- **Unreal, runtime:** the DRS heuristic above. — [Epic: Dynamic resolution](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- **Unity:** `SystemInfo` exposes platform and hardware capabilities such as `graphicsMemorySize`, approximate in MB. — [Unity: SystemInfo](https://docs.unity.com/en-us/engine/6000.3/script-reference/unityengine/systeminfo). A community project scores hardware to pick a first-run quality level and says testing is needed. — [Softdrink117/unity-system-info-detection](https://github.com/Softdrink117/unity-system-info-detection). An official benchmark-based default was not found.
- **Source:** GPU table in `dxsupport.cfg`, later memory detection. — [VDC: Dxlevel](https://developer.valvesoftware.com/wiki/Dxlevel) (extract)
- **Rage:** automatic assessment. Method unknown. — [Bit-Tech](https://bit-tech.net/reviews/gaming/pc/rage-technical-analysis/2/)
- **Godot:** no auto-selection found in the docs read. It does automatically fall back from Vulkan or D3D12 to OpenGL 3 by capability, which is a feature check rather than a quality setting. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)

### Inferences
Suggested layered approach for moonBASIC, combining what these engines do:
1. **Capability check at startup** (Godot-style): required extensions and GL version, picking a fallback path. This is not a score.
2. **Small table** (Source-style, but kept minimal): match `GL_VENDOR` and `GL_RENDERER` to cap tiers for known weak families, such as Intel HD of the 2016 era. Keep overrides in a user-editable file. Source's experience suggests a broad table ages poorly, and Linux and macOS strings differ from Windows.
3. **Short GPU benchmark** (Unreal-style): a sub-second timed scene with timer queries on first launch or on a GPU change, mapped to tiers with thresholds calibrated on real machines. Store results in a config keyed on the renderer string.
4. **Runtime feedback**: DRS using the Unreal heuristic. Optionally, sustained low scale could suggest dropping a tier, but should not silently change tiers mid-game, since it can pop and cause shader recompiles.
- Benchmark thresholds need real-hardware calibration. Looman's point is the critical one here, and this can't be derived from research alone.

### Gaps
- Unity: official behaviour for selecting the default quality level per platform at first run was not confirmed.
- Whether Godot has any first-run tier detection in its demos or engine: none found in pages read. A negative based on limited reading, not confirmed.
- GPU VRAM cannot be queried in core GL 3.3 as far as is known. `GL_NVX_gpu_memory_info` and `GL_ATI_meminfo` are the usual vendor extensions. This is from memory and was not verified in this research.

## Which of these are feasible on OpenGL 3.3 without compute shaders?

### Takeaway
Nearly all the low-cost scaling techniques run on GL 3.3 with fragment shaders: tiers, resolution scale, spatial upscalers, timer-query DRS, mip bias, cached shadows, half-res effects, and sorted batching. What is not feasible, or not worth it, is anything needing compute, SSBOs, multi-draw indirect or persistent mapping (GL 4.3 or 4.4). macOS caps at GL 4.1, so these stay out of reach there no matter how new the GPU is.

### Cited Findings
- `glVertexAttribDivisor` (instancing) requires GL 3.3. — [Khronos: glVertexAttribDivisor](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glVertexAttribDivisor.xhtml)
- `glQueryCounter`, for GPU timestamps: Khronos's table shows support from 3.2, while the page's notes say "3.3 or higher". Either way it is available on 3.3. — [Khronos: glQueryCounter](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glQueryCounter.xhtml). **Conflict:** a summary of Wikipedia's OpenGL version table listed timer query and instanced arrays under 4.0 ([Wikipedia: OpenGL](https://en.wikipedia.org/wiki/OpenGL)); Khronos's own refpages contradict that and take precedence. The Wikipedia reading may be a summarisation error.
- Timer query semantics: `GL_TIME_ELAPSED` measures how long commands take to fully complete, in nanoseconds. Use `GL_QUERY_RESULT_AVAILABLE` to avoid stalls. Use 64-bit getters. On tile-based implementations, results may not match expectations because rendering isn't linear, which matters for Apple Silicon. `TIME_ELAPSED` and `TIMESTAMP` may use different clocks. — [Khronos: ARB_timer_query](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_timer_query.txt)
- `glDispatchCompute` requires 4.3, and `glMultiDrawElementsIndirect` requires 4.3. — [Khronos: glDispatchCompute](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glDispatchCompute.xhtml); [Khronos: glMultiDrawElementsIndirect](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glMultiDrawElementsIndirect.xhtml)
- `glBufferStorage` requires 4.4. — [Khronos: glBufferStorage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glBufferStorage.xhtml)
- macOS supports OpenGL 4.1 at most, in forward-compatible core profile only. It is deprecated and poorly maintained, and on modern systems implemented on top of Metal. — [GLFW commit/docs mirror](https://gitlab.cristal.univ-lille.fr/pirvi-public/glfw/-/commit/dfebad786d2bc00b2d63ff71b3fbeb75a1e513d6) (library docs, not Apple's)
- Godot's Compatibility renderer proves the non-compute forward approach ships at scale, with SSAO, MSAA, glow, tonemapping and fullscreen post, and it has no compute. — [Godot: Overview of renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- FSR 1 is spatial, needs only the current anti-aliased frame, and works best in perceptual colour space, so it should run after tonemapping. Noise-adding passes go after upscaling. It accepts any area scale from 1x to 4x, which suits DRS. Presets are Ultra Quality 1.3x, Quality 1.5x, Balanced 1.7x, Performance 2.0x per axis. Published APIs are DX12, DX11 and Vulkan, with OpenGL not mentioned. The AMD upper-bound cost figures are 4K: 0.40 ms or less (enthusiast), 0.60 ms or less (performance), 1.0 ms or less (mainstream); 1440p: 0.20 / 0.30 / 0.50 ms or less. — [AMD GPUOpen: FSR](https://gpuopen.com/fidelityfx-superresolution/)

Feasibility summary (a judgement table; "yes" entries rest on the API facts above plus inference):

| Technique | Source engine | GL 3.3 without compute? | Note |
|---|---|---|---|
| Data-driven quality groups and levels | Unreal, Unity | Yes | No GPU requirement |
| Allocate-at-max, render to sub-viewport | Unity, Unreal | Yes | `glViewport` and UV scaling |
| Timer-query DRS heuristic, with panic and history | Unreal | Yes | Read results frames later; distrust on tile-based GPUs |
| FSR 1 in fragment shaders | Godot Forward+, Unity | Yes (moonBASIC already has it) | AMD does not list GL; run after tonemap |
| Negative mip bias tied to scale | Godot | Yes | Formula above; sampler or shader bias |
| Half-res AO and quarter-res bloom | Doom, Godot RD | Yes | Fixed-function render targets |
| Depth pre-pass with `EQUAL` | Doom | Yes | Standard depth state |
| Cached shadow maps and atlas reuse | Doom | Yes | Render-to-texture plus copy |
| Occlusion queries read frames later | Doom | Yes | Standard GL queries |
| Batching by program with persistent per-material UBOs | Unity SRP Batcher | Yes (design inference) | No persistent mapping; orphaned or ranged UBO updates |
| Instancing | Godot | Yes | GL 3.3 |
| CPU-built clustered light grid | Doom | Yes, likely | Doom builds on CPU; upload via texture or UBO; cost needs testing |
| Compute-built clustered lights, as in Godot | Godot Forward+ | No | Needs 4.3 |
| GPU-driven draws (MDI) | Modern engines | No | 4.3, not on macOS |
| Persistent mapped ring buffers | Modern engines | No | 4.4, not on macOS |
| FSR 2, TAA upscaling | Godot, Unity | Not FSR 2 as shipped; a raster TAA could work | FSR 2's compute dependency was not verified here; Godot's docs state only Forward+ |
| Virtual texturing | id Tech | Possibly, but poor value | Needs a tile cache, page table and feedback; Doom's described path was on Vulkan |

### Inferences
- The largest gain for least work appears to be: mip bias tied to the DRS scale, half-res GTAO, static shadow caching, a depth pre-pass, and a tuned DRS controller.
- moonBASIC should not chase parity with Godot's Compute-based features. Its GL 3.3 target already sits above Godot's Compatibility in post-processing and upscaling (see Q1).
- Because moonBASIC targets macOS, even a future GL 4.x path could not use compute or MDI on Mac. Advanced work would need a Metal-capable backend, which is outside G3N's GL scope.
- Compute-dependent GTAO variants need checking. If moonBASIC's GTAO is fragment-based, this is already resolved, but this research did not look at its implementation.
- On Apple Silicon, tile-based timing means timer-query results can mislead, which is the "wrong GPU timings" risk Epic cites. moonBASIC should combine GPU time with CPU frame pacing and test this case specifically.
- Virtual texturing appears to be a poor fit: it also needs an asset pipeline, tile cache and disk streaming. The moonBASIC tier approach plus mip streaming should cover the same need more cheaply. This is a judgement, not a sourced finding.

### Gaps
- GTAO on non-compute GL: no source reviewed here, and moonBASIC's implementation was not read.
- Virtual texturing on GL 3.3: id's paper could not be read, so whether Rage's path shipped on GL 3.3 is unverified.
- FSR 1 on GL: no AMD statement found. moonBASIC's existing implementation is evidence it works, and that was reported in the brief, not tested here.
- Timer query reliability on Mesa, Intel and macOS GL drivers: no measurements found. Test on real hardware before relying on them.
