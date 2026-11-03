# Keeping frame rate smooth in OpenGL games: techniques for moonBASIC (G3N, GL 3.3 core)

Source quality note for the writer: Khronos wiki pages (Shader Compilation, Swap Interval) and the Ladavac Medium post could not be fetched (403 or connection refused). Several claims therefore rest on specs, engine docs and secondary coverage. Where something comes from memory rather than a retrieved source, it is in Gaps and marked as such. Two Intel PDFs downloaded but their text could not be extracted, so they are cited only for what search snippets showed.

## How do dynamic resolution controllers work (GPU timers, thresholds, hysteresis)? What do published ones use?

### Takeaway
Published controllers measure GPU time, not CPU time. They drop resolution fast and raise it slowly, and they keep a headroom margin below the frame budget. Valve's three-rule 70/90/85% controller is the most concrete published design and ports directly to GL 3.3 timer queries, while Unreal adds a history window, a panic path and a minimum change size.

### Cited Findings

**Valve "Adaptive Quality" (Steam Dev Days 2016), numbers for a 90 Hz frame of 11.1 ms**
- Goal is to hold GPU utilization between 70% and 90% — [Scaling Great VR](https://cdn.fastly.steamstatic.com/apps/steamdevdays/slides2016/ScalingGreatVR.pdf)
- Decrease rule: if the last frame finished after the 90% threshold (10.0 ms), drop 2 levels, then wait 2 frames — [Scaling Great VR](https://cdn.fastly.steamstatic.com/apps/steamdevdays/slides2016/ScalingGreatVR.pdf)
- Increase rule: if the last 3 frames finished below the 70% threshold (7.8 ms), raise 1 level, then wait 2 frames — [Scaling Great VR](https://cdn.fastly.steamstatic.com/apps/steamdevdays/slides2016/ScalingGreatVR.pdf)
- Prediction rule: if the last frame is above 85% (9.4 ms) and linear extrapolation from the last two frames lands above 90%, drop 2 levels and wait. The extracted slide text cuts off at "wait 2", which presumably means 2 frames — [Scaling Great VR](https://cdn.fastly.steamstatic.com/apps/steamdevdays/slides2016/ScalingGreatVR.pdf)
- The 10% idle margin is deliberate, because the GPU is shared with other processes including the Windows desktop. Valve lowered its budget recommendation from 11.11 ms to 10.0 ms for this reason — [Scaling Great VR](https://cdn.fastly.steamstatic.com/apps/steamdevdays/slides2016/ScalingGreatVR.pdf)
- Valve lists the tiers as manual tuning, one-time auto-tuning and adaptive scaling, and says adaptive quality lowers the GPU min spec — [Scaling Great VR](https://cdn.fastly.steamstatic.com/apps/steamdevdays/slides2016/ScalingGreatVR.pdf)

**Unreal Engine heuristic**
- It adjusts primary screen percentage from previous frames' GPU timings. When several consecutive GPU frames exceed budget, an "over budget panic" lowers resolution quickly and resets history so cheap past frames do not skew the result — [Epic docs](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Documented defaults are `MinScreenPercentage` 50, `MaxScreenPercentage` 100 (also used to size render targets) and `FrameTimeBudget` 33.3 ms — [Epic docs](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Hysteresis-style knobs, with no defaults stated on the page: `ChangePercentageThreshold` (minimum change before a resize), `MinResolutionChangePeriod` (frames between changes), `HistorySize`, `TargetedGPUHeadRoom` (percent of budget), `MaxConsecutiveOverbudgetGPUFrameCount` (panic trigger) — [Epic docs](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- `CPUBoundScreenPercentage` sets the target when CPU-bound, and the docs suggest a lower value where CPU and GPU share memory bandwidth, which describes integrated GPUs — [Epic docs](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- GPU and render-thread synchronization can add latency before a resolution change takes effect — [Epic docs](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)

**Offline alternative: Redout**
- An AI racer profiled every track and stored a per-track-piece grid of screen percentages, so nothing is analyzed at runtime. Most of the track sat at 80% (59.22%), with 90% at 11.48%, 70% at 18.00%, 60% at 4.04% and 50% at 1.94%. Another 5.05% of cells were CPU hiccups that resolution cannot fix — [34BigThings](https://34bigthings.com/redout-a-technical-analysis-on-the-offline-dynamic-resolution-scaler/)
- The article attributes to Epic a method that avoids buffer reallocation by rendering to a sub-region of the full-size buffer — [34BigThings](https://34bigthings.com/redout-a-technical-analysis-on-the-offline-dynamic-resolution-scaler/)

**FSR 1 as the upscaler**
- Modes per dimension: Ultra Quality 1.3x (77%), Quality 1.5x (67%), Balanced 1.7x (59%), Performance 2.0x (50%) — [AMD GPUOpen](https://gpuopen.com/fidelityfx-superresolution/)
- Arbitrary scaling mode supports any area factor from 1x to 4x, and AMD says it is typically used for dynamic resolution — [AMD GPUOpen](https://gpuopen.com/fidelityfx-superresolution/)
- Published cost upper bounds: 1440p is 0.20 ms on enthusiast, 0.30 ms on performance and 0.50 ms on mainstream GPUs; 4K is 0.40 / 0.60 / 1.0 ms. The listed GPUs are discrete; the 780M is not among them — [AMD GPUOpen](https://gpuopen.com/fidelityfx-superresolution/)

**GL 3.3 timing primitive**
- `ARB_timer_query` is written against GL 3.2. `TIME_ELAPSED` measures from completion of prior commands to completion of the commands before `EndQuery`, in nanoseconds. `QUERY_RESULT_AVAILABLE` returns FALSE if reading the result would delay — [ARB_timer_query](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_timer_query.txt)
- Reissuing a query on the same ID before reading it discards the old result. The spec's own polling sample stalls, and it says collecting results a frame later reduces waiting — [ARB_timer_query](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_timer_query.txt)

### Inferences
- Scaling Valve's percentages to 60 Hz (16.67 ms) gives high 15.0 ms, prediction 14.2 ms and low 11.7 ms. These are my arithmetic; Valve published only the 90 Hz values.
- **Check the 16.5 ms figure.** 16.5 ms is suspiciously close to a 60 Hz vsync interval (16.67 ms). If vsync was on, that number may be vsync wait rather than GPU cost. If it is true GPU time, it is above the 15.0 ms high threshold, so any spike misses the vblank. Measure with `TIME_ELAPSED` per pass before tuning anything.
- Use a ring of 3 to 4 query objects, poll availability and never block. Skip samples tagged with an older resolution, since timer results lag submission by at least a frame. This is why Valve's "wait 2 frames" exists.
- Suggested moonBASIC controller: Valve's three rules on a continuous FSR arbitrary scale, a 50 to 100% range matching Unreal's defaults and FSR Performance, a minimum change of a few percent, and Unreal-style panic with a history reset.
- Allocate targets at maximum size and render into a sub-viewport, so a resolution change never reallocates. Reallocation mid-play would itself cause a hitch.
- Dynamic resolution only reduces fragment work in the scaled passes. Shadow cascades and geometry processing do not shrink with it, and scaled reflection only helps fragment cost (see the flutter_scene note under redraw reduction). A scene dominated by redraw vertex cost, which 1.9M triangles times several passes may be, will not gain much until redraws are cut. If the controller reaches its floor and the frame is still over budget, treat it as CPU-bound or vertex-bound, as Redout's 5.05% cells show.

### Gaps
- No published thresholds found for AAA console controllers (Gears, Doom, etc.). The search surfaced only user-facing settings for Gears titles.
- Unreal's defaults for the hysteresis variables are not on the docs page; they are in `DynamicResolution.cpp`, which I did not read.
- [Intel's Dynamic Resolution Rendering paper](https://www.intel.com/content/dam/develop/external/us/en/documents/dynamicresolutionrendering-183334.pdf) downloaded but its text could not be extracted. A search summary describes it as viewport-based and aimed at older integrated graphics, so it is worth reading by hand for mip-bias guidance.
- From memory, unverified here: timer queries became core in GL 3.3, and a mip LOD bias of about log2(render/display) is normally applied when upscaling.
- No measured FSR 1 cost on a 780M.

## What is frame pacing, why does average FPS hide it, and what can a GL 3.3 app do (vsync, swap timing, glFinish, sleep)?

### Takeaway
Pacing is the evenness of frame delivery, and a good average can hide alternating long and short frames. GL 3.3 has no display-timing API, so an app can only keep the GPU under budget, keep vsync on, limit queued frames with fences or a post-swap sync, and smooth the simulation timestep. Evidence for the GL-specific tactics is mostly forum-grade.

### Cited Findings
- Croteam's Alen Ladavac (GDC 2018, later a [Medium post](https://medium.com/@alen.ladavac/the-elusive-frame-timing-168f899aec92), not retrievable) investigated why games stutter at high frame rates and showed a "Heartbeat" stutter pattern — [GamingOnLinux](https://www.gamingonlinux.com/2018/07/the-cto-of-croteam-has-written-up-a-post-about-the-elusive-frame-timing); [Phoronix](https://www.phoronix.com/news/Frame-Timing-Game-Stuttering)
- His central argument, per secondary coverage: measuring how long a frame takes to render misleads, and the useful measure is when the frame is actually presented. Croteam tried `VK_GOOGLE_display_timing` in The Talos Principle and Serious Sam Fusion and reported less micro-stutter, with limited driver support at the time. That is a Vulkan extension — [Phoronix](https://www.phoronix.com/news/GDC-Vulkan-Stutter-Frames)
- Illustration from a lower-authority blog: frames at 12 ms and 21 ms can average near 16.7 ms and still judder, and a 60 FPS average with occasional 50 ms frames feels stuttery — [Bugnet](https://bugnet.io/blog/how-to-fix-micro-stutter-and-frame-pacing)
- 1% lows expose this: a 60 FPS average with a 58 FPS 1% low is smooth, while the same average with a 15 FPS 1% low stutters badly — [BottleneckPC](https://bottleneckpc.com/blog/1-percent-lows-explained)
- Drivers often queue the swap and return immediately, so the block lands on the next GL call. This comes from forum replies and a commit message, not vendor documentation — [Khronos forum](https://community.khronos.org/t/swapbuffers-and-synchronization/107667); [Alacritty commit](https://git.causa-arcana.com/PolytreeDE/alacritty/commit/38d20d0c391c250953ce3c72a35c8d1156f06000)
- `glFinish` after swap is reported to cut input latency, including one flight-sim report of several frames removed. Opinion is split, with one reply saying never to call it on modern hardware — [Khronos forum](https://community.khronos.org/t/general-opengl-input-lag/60581); [Khronos forum](https://forums.khronos.org/showthread.php/77737)
- Forum consensus prefers a sync object if you want to wait on an earlier frame rather than the current one. Sync objects are the fence route to limiting CPU run-ahead — [Khronos forum](https://community.khronos.org/t/swapbuffers-and-synchronization/107667)
- On NVIDIA, the first GL write after a swap is reported to be what waits for a swapchain image — [NVIDIA forum](https://forums.developer.nvidia.com/t/long-stuttering-in-minimal-opengl-win32-native-application/229934)

### Inferences
- Priority order for moonBASIC:
  1. Keep GPU time under about 90% of the budget, using the controller above. Missed vblanks are the main pacing failure under vsync.
  2. Keep vsync on by default.
  3. Insert a fence at each frame end and wait on the fence from 1 to 2 frames back. This bounds queue depth without a full `glFinish` stall.
  4. Treat `glFinish` as an optional low-latency setting, not a default.
- Time the simulation from a smoothed or vblank-quantized delta rather than the raw CPU delta. A raw delta can feed the "heartbeat" back into motion, which fits Croteam's point about render time vs present time. This is my reading; I could not retrieve the original text.
- Report frame-time percentiles and a frame-time graph in the profiler rather than average FPS. Add a count of missed vblanks.
- Sleep: sleeping before input sampling can trade throughput margin for latency, but I found no published GL numbers. Go's scheduler and Windows timer granularity would need measurement, so treat this as experimental.
- A frame cap under vsync-off needs its own testing; I found no solid source for it.

### Gaps
- The Khronos wiki Swap Interval and Synchronization pages were unreachable, so `WGL_EXT_swap_control_tear` adaptive vsync semantics are unsourced here. From memory, it is a negative swap interval that tears instead of waiting when late, and it is not available everywhere.
- From memory, unverified here: sync objects are core since GL 3.2.
- No GL equivalent to `VK_GOOGLE_display_timing` was found. Present-time feedback in GL 3.3 appears unavailable.

## How do you avoid GL shader compile stutter: program binaries, KHR_parallel_shader_compile, a precompile phase, fewer uber-shader variants? What is available on macOS GL 4.1 and Mesa?

### Takeaway
Layer it: reduce the variant count, compile everything during a loading phase, use parallel compile where exposed, and cache program binaries with a mandatory source fallback. Every layer needs a runtime check, because binary formats can legally number zero and caches invalidate on driver updates.

### Cited Findings

**Program binaries (`ARB_get_program_binary`)**
- The extension requires GL 3.0, so it can exist on a 3.3 context as an extension — [ARB_get_program_binary](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_get_program_binary.txt)
- Set `PROGRAM_BINARY_RETRIEVABLE_HINT` to TRUE before linking; it takes effect on the next successful link or load — [spec](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_get_program_binary.txt)
- `NUM_PROGRAM_BINARY_FORMATS` may legally be 0, and then `GetProgramBinary` generates `INVALID_OPERATION` — [spec](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_get_program_binary.txt)
- `ProgramBinary` failure is silent: it sets `LINK_STATUS` FALSE with no GL error, for example after a hardware, driver or compiler update. The spec says to fall back to GLSL source, relink and re-cache — [spec](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_get_program_binary.txt)
- A driver can specialize on GL state. Retrieving a binary after exercising the expected states can capture optimized variants and avoid later recompiles — [spec](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_get_program_binary.txt)

**Parallel compile (`KHR_parallel_shader_compile`)**
- Adds `MaxShaderCompilerThreadsKHR(count)`, where 0 disables and 0xFFFFFFFF requests the implementation maximum (the default). It is a hint, not a guarantee — [KHR_parallel_shader_compile](https://registry.khronos.org/OpenGL/extensions/KHR/KHR_parallel_shader_compile.txt)
- Adds `COMPLETION_STATUS_KHR` for `GetShaderiv` and `GetProgramiv`. It lets the app poll rather than block on link status — [spec](https://registry.khronos.org/OpenGL/extensions/KHR/KHR_parallel_shader_compile.txt)
- The spec is written against GL 4.5 and does not say which calls may still block — [spec](https://registry.khronos.org/OpenGL/extensions/KHR/KHR_parallel_shader_compile.txt)

**Mesa**
- Mesa has an on-disk shader cache, 1 GB by default, in `$XDG_CACHE_HOME/mesa_shader_cache` or `~/.cache/mesa_shader_cache`, controlled by `MESA_SHADER_CACHE_DISABLE`, `MESA_SHADER_CACHE_MAX_SIZE` and `MESA_SHADER_CACHE_DIR` — [Mesa docs](https://docs.mesa3d.org/envvars.html)
- Mesa program binaries only load on the same Mesa build and same GPU. Support landed first for Intel in 2017, then Gallium drivers — [Phoronix](https://www.phoronix.com/news/Intel-Lands-Program-Bin-Mesa)
- radeonsi in Mesa 19.1 (2019) was the first Mesa GL driver with `KHR_parallel_shader_compile` and `ARB_parallel_shader_compile` — [Phoronix](https://www.phoronix.com/news/RadeonSI-Lands-parallel_shader)

**macOS**
- Apple deprecated OpenGL in macOS 10.14 and froze it at 4.1 — [Geeks3D](https://geeks3d.com/20180605/apple-announces-that-opengl-and-opencl-will-be-deprecated-in-macos-10-14-mojave); [Apple retired docs](https://developer.apple.com/library/mac/documentation/graphicsimaging/conceptual/opengl-macprogguide/opengl_api_versions/opengl_api_versions.html)
- A GL 4.1 cookbook says binary caching arrived with 4.1 but is not widely supported, and notes that Intel's macOS drivers support no binary formats. This is a single book source — [OpenGL 4 Shading Language Cookbook](https://www.oreilly.com/library/view/opengl-4-shading/9781789342253/bae34249-478a-44d2-b0e6-17b3ce90f067.xhtml)
- Apple's guidance is to query the renderer version and extension string before using a feature — [Apple retired docs](https://developer.apple.com/library/mac/documentation/graphicsimaging/conceptual/opengl-macprogguide/opengl_api_versions/opengl_api_versions.html)

**Variants and caches: the Dolphin case study**
- Dolphin's ubershaders replace per-configuration compiles with a flexible interpreter-style shader compiled at startup, removing most in-game stutter. The hybrid mode is recommended, since it cuts the performance cost while keeping games smooth — [Dolphin blog](https://dolphin-emu.org/blog/2017/07/30/ubershaders/)
- Dolphin found that a GPU change, driver update or new Dolphin version invalidates a disk shader cache, so first-run stutter persists with caching alone — [Dolphin blog](https://dolphin-emu.org/blog/2017/07/30/ubershaders/)
- Stutter can come from pipeline state changes even with ubershaders; reported on Vulkan and since fixed — [Dolphin tracker #12646](https://bugs.dolphin-emu.org/issues/12646)

### Inferences
- Recommended order for the 2-second first-frame stall:
  1. Instrument the stall: per-program compile and link time, and how many programs exist. If the renderer compiles lazily at first draw, one 2-second hitch can become scattered hitches once it is spread out, so enumerate all variants up front.
  2. Move compilation to a loading phase behind a screen. Issue all `glCompileShader` and `glLinkProgram` calls, then poll with `COMPLETION_STATUS_KHR` where available so the window stays responsive and can show progress. Without the extension, compile a time-sliced batch per frame.
  3. Cache binaries keyed on hash of source + `GL_VENDOR` + `GL_RENDERER` + `GL_VERSION`. Treat any `LINK_STATUS` FALSE as a cache miss and recompile. Write the binary after a real draw, since the spec's state-dependence note suggests that yields better binaries.
  4. Reduce variants by moving feature toggles from `#define`s into uniform branches where the cost is tolerable. A hybrid like Dolphin's, drawing with a general shader while the specialized one builds in the background, suits a Blitz-style language whose scripts can create materials at runtime.
- Warm draws: after linking, issue a small draw with each program under representative state, so the driver's deferred state-based recompiles happen on the loading screen. The spec's state-dependence wording supports the concept; I found no measurement of the benefit.
- Linux users may already get much of the benefit from Mesa's own disk cache; app-level binary caching adds less there. This is a reasonable reading, not a measured result.
- All of this is GL 3.3-compatible when guarded by extension and format-count checks, with source compile as the universal fallback.

### Gaps
- The Khronos wiki Shader Compilation page returned 403.
- Which parallel-compile extensions current AMD Windows (780M), NVIDIA Windows, or Intel Windows drivers expose was not confirmed. Query at runtime.
- No source confirms `KHR_parallel_shader_compile` on macOS, or the value of `NUM_PROGRAM_BINARY_FORMATS` on Apple Silicon. Apple's [capability tables](https://developer.apple.com/opengl/OpenGL-Capabilities-Tables.pdf) are from the 10.7.5 era. Test on real Macs.
- From memory, unverified here: compiling on a second shared context works on some drivers but is unreliable across vendors. No source retrieved.
- No published data on ms saved by binary caching for a comparable shader set.

## Which are the most effective ways to cut scene redraws on GL 3.3: cascade caching, shadow LOD, low-rate reflections, occlusion culling, MultiDraw and instancing, AZDO limits?

### Takeaway
The biggest published wins come from not rendering at all: cached distant cascades, tighter cascade culling, budgeted half-resolution reflections with layer masks, then occlusion. On 3.3, `glMultiDrawElementsBaseVertex` and instancing remain available, but indirect draws and persistent mapping (AZDO) need GL 4.x or optional extensions.

### Cited Findings

**Shadow caching**
- CRYENGINE: cascades past a set index are rendered once and kept. They recenter when the camera nears the border, and dynamic objects can be excluded from the cache and drawn into normal cascades — [CRYENGINE docs](https://www.cryengine.com/docs/static/engines/cryengine-3/categories/1114113/pages/21267738)
- Cyberpunk 2077 (SIGGRAPH 2021): up to 4 per-frame cascades plus 4 distant levels, each 4 times larger than the previous, with the last covering the whole 256 km² map. Distant rendering runs on a per-frame budget, usually takes several frames, and the map refreshes every few seconds. Collecting meshes cannot be spread across frames, so that step was threaded and batched — [Dybisz and Witanowski](https://history.siggraph.org/wp-content/uploads/2022/06/2021-Talks-Dybisz_Shadows-Optimizations-in-Cyberpunk-2077.pdf)
- Unity HDRP: cached shadow maps switch to every-frame when near the camera, and static casters draw on update while dynamics draw each frame — [Unity Graphics PR](https://github.com/Unity-Technologies/Graphics/pull/1559/files/de8888aec1cef41094d82d40cbd1a2f400521ed9)
- Flax: copies cached static shadows and renders only dynamic objects on top, and update rate can fall with distance — [Flax docs](https://docs.flaxengine.com/manual/graphics/lighting/shadows.html)

**Shadow culling**
- Cyberpunk moved from map-based to interval-based cascade selection, since map-based drew many objects unnecessarily. Interval-based lets each cascade cull against its frustum slice — [Dybisz and Witanowski](https://history.siggraph.org/wp-content/uploads/2022/06/2021-Talks-Dybisz_Shadows-Optimizations-in-Cyberpunk-2077.pdf)
- They cull with a tight convex shape of up to 8 planes rather than a 6-plane box. They also test a light-stretched bounding box against the main view's occlusion buffer, and use the first distant level as an occluder from the light's view — [same talk](https://history.siggraph.org/wp-content/uploads/2022/06/2021-Talks-Dybisz_Shadows-Optimizations-in-Cyberpunk-2077.pdf)
- An offline time-of-day visibility bake, at 12 samples every 2 hours, cut global shadow triangles "by millions". No percentage was published — [same talk](https://history.siggraph.org/wp-content/uploads/2022/06/2021-Talks-Dybisz_Shadows-Optimizations-in-Cyberpunk-2077.pdf)

**Reflections**
- RyzomCore caps realtime water planes per frame and falls back to the environment map elsewhere. Planes rendered last frame get a 1.25x area bonus to prevent flip-flopping. Half-resolution is on by default because bump distortion hides it. This is a project wiki, not a talk — [RyzomCore wiki](https://github.com/ryzom/ryzomcore/wiki/water_planar_reflection)
- flutter_scene defaults to 0.5 scale and notes that fragment cost scales with resolution while draw-call cost scales with scene complexity, so mask what goes into the mirror pass — [flutter_scene docs](https://pub.dev/documentation/flutter_scene/latest/scene/PlanarReflectorComponent-class.html)
- Unity time-slices reflection probes over frames, at the cost of slower response to lighting changes — [Unity docs](https://docs.unity.com/en-us/engine/6000.6/manual/lighting-overview/reflections/ref-probe-performance.md). A forum user reports that dropped reflection frames on planar reflections create visible lag — [Unity forum](https://discussions.unity.com/t/best-usage-for-realtime-reflection-planes/814717)

**Occlusion culling**
- Masked Software Occlusion Culling (HPG 2016): SIMD CPU algorithm, about 3x faster than prior work, culling 98% of what a full-resolution depth buffer culls, with very low memory overhead. Code is at GameTechDev/MaskedOcclusionCulling — [Lund project page](https://fileadmin.cs.lth.se/graphics/research/papers/2016/culling/)
- Intel warns that its reworked library may not exactly match the paper's performance — [Intel](https://intel.com/content/www/us/en/developer/articles/technical/masked-software-occlusion-culling.html)
- A follow-up merges hierarchical buffers to reduce silhouette bleed-through with unsorted geometry and allow concurrent building — [Intel](https://intel.com/content/dam/develop/external/us/en/documents/merging-masked-occlusion-culling-hierarchical-buffers-faster-rendering.pdf)
- GPU Hi-Z (RasterGrid, 2010): after occluders, reduce depth into a max-depth mip chain with one pass per level, choose LOD from the box's screen size with `ceil(log2(max(w,h)/2))`, fetch 4 texels and compare. Build took under 0.2 ms on a Radeon HD5770. Culling used a geometry shader and transform feedback without CPU readback — [RasterGrid](https://www.rastergrid.com/blog/2010/10/hierarchical-z-map-based-occlusion-culling/)
- **Needs porting:** that demo is GL 4.0 with `#version 400 core`. Multisampled depth needs a separate reduction pass per sample, requiring GL 3.2 or `ARB_texture_multisample` — [RasterGrid](https://www.rastergrid.com/blog/2010/10/hierarchical-z-map-based-occlusion-culling/)
- Cyberpunk uses a CPU occlusion buffer for visibility testing — [Dybisz and Witanowski](https://history.siggraph.org/wp-content/uploads/2022/06/2021-Talks-Dybisz_Shadows-Optimizations-in-Cyberpunk-2077.pdf)

**Draw submission**
- `glMultiDrawElementsBaseVertex` is listed for GL 3.1 onward by the reference page, so it is available on 3.3 — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glMultiDrawElementsBaseVertex.xhtml)
- **Needs GL 4.x:** AZDO's listed features include GL 4.3 multi-draw indirect and GL 4.4 buffer storage, and persistent mapped buffers come from `ARB_buffer_storage` — [AZDO slides](https://www.slideshare.net/slideshow/gl-efficiency/32590355); [InformIT](https://informit.com/articles/article.aspx?p=2437085)
- **Needs GL 4.0 or extension:** `ARB_multi_draw_indirect` requires GL 4.0 or `ARB_draw_indirect` — [spec](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_multi_draw_indirect.txt)
- NVIDIA's sample states the MDI goal is reducing CPU load, which may not raise frame rate directly — [NVIDIA](https://docs.nvidia.com/gameworks/content/gameworkslibrary/graphicssamples/opengl_samples/multidrawindirectsample.htm)
- Intel GPA guidance for a high draw-call metric is to batch geometry into fewer calls — [Intel GPA guide](https://www.intel.com/content/www/us/en/docs/gpa/user-guide/2023-4/graphics-api-metrics.html)

### Inferences
- Scale of the problem, assuming 4 cascades (actual count unknown): 4 cascades + reflection + refraction + main = 7 scene views. The worst-case upper bound is about 13.3M vertices per frame for 1.9M triangles, before culling. Per-view counters and timers should confirm the true figure.
- Likely effectiveness order for moonBASIC, by judgment rather than a published ranking:
  1. Per-pass GPU timers and per-view triangle counters.
  2. Tight per-cascade culling with interval-based selection. It has no visual cost and is cheap on CPU.
  3. Cache the far 1 to 2 cascades CRYENGINE-style, rendering static casters on recenter or by a sliced budget like Cyberpunk, and overlaying dynamic casters. Moving-caster and sun-angle changes must force or schedule refreshes.
  4. Water: render reflection at half resolution with a layer mask that drops small props, and clip below the plane. RyzomCore's approach supports a per-frame budget across planes.
  5. Refraction: sampling a copy of the main view's color and depth, rather than redrawing the scene, is common practice and removes a whole pass. I did not retrieve a source for it, and it cannot show objects hidden behind the surface layer from the camera.
  6. Shadow LOD: draw lower LODs into far cascades, and drop small casters. Cyberpunk's character-mesh swapping is a related example, though specific numbers are unpublished.
  7. Occlusion culling, once redraws are down. Value depends strongly on scene density, and outdoor water scenes may see little gain.
- Low-rate water reflection on a single plane is riskier than resolution and masking, given the forum report of visible lag. A camera-moving test is needed before shipping it.
- Hi-Z vs software on 3.3: Hi-Z needs no SIMD library and fits the renderer, but its result returns via transform feedback or readback, and using it without indirect draws is awkward. The Intel library is C++ with one .cpp and one .h file, so it would need cgo. G3N's GLFW binding already seems to imply cgo, though I did not confirm that for moonBASIC's build. A coarse Go-native depth rasterizer of a few occluders is a third option, but it has no published numbers.
- If using 8x MSAA during Hi-Z, the per-sample reduction adds work; a single-sample depth prepass or disabling MSAA avoids it.
- AZDO on 3.3: treat `ARB_draw_indirect`, `ARB_multi_draw_indirect` and `ARB_buffer_storage` as optional runtime-checked paths and never the only path. macOS at 4.1 cannot get 4.3 or 4.4 core features. `glMultiDrawElementsBaseVertex` over a shared VBO and UBO-based instance data is the realistic universal baseline. From memory, unsourced here: multi-draw has no per-draw ID on 3.3 without tricks (`gl_DrawID` needs `ARB_shader_draw_parameters`), so per-object data needs instancing or base-vertex-encoded data.

### Gaps
- The AZDO talk's benchmark numbers were not retrieved; a "15x" claim appeared only second-hand on a [blog](https://viml.nchc.org.tw/?p=524). Do not cite it as fact.
- No direct published benchmark of Hi-Z vs Masked Occlusion Culling was found.
- No published ms savings for CSM caching in the sources read; Cyberpunk reports triangles qualitatively.
- From memory, unverified here: refpage's 3.1 listing differs from my recollection that base-vertex draws entered core at 3.2. Either way it is present on 3.3. Geometry shaders (3.2) and transform feedback (3.0) are also core on 3.3, and `glDrawTransformFeedbackInstanced` needs 4.2.

## What should a game do on Intel HD class integrated GPUs and on macOS's deprecated OpenGL?

### Takeaway
Detect capabilities at runtime, start on low defaults with one-time auto-tuning, then use adaptive scaling. Remove the hardcoded 8x MSAA first. On macOS, assume GL 4.1 at most, no program binaries and possible future removal, and keep the renderer strictly on the 3.3 path.

### Cited Findings
- G3N hardcodes `glfw.WindowHint(glfw.Samples, 8)` at `window/glfw.go:243` in g3n/engine v0.2.0, confirmed in the local module cache: `C:\Users\Snow\go\pkg\mod\github.com\g3n\engine@v0.2.0\window\glfw.go`
- Valve's tiers of manual, one-time auto-tuning and adaptive scaling, and its min-spec benefit — [Scaling Great VR](https://cdn.fastly.steamstatic.com/apps/steamdevdays/slides2016/ScalingGreatVR.pdf)
- Unreal suggests a lower CPU-bound screen percentage where CPU and GPU share memory bandwidth — [Epic docs](https://dev.epicgames.com/documentation/en-us/unreal-engine/dynamic-resolution-in-unreal-engine)
- Intel GPA: batch geometry to lower draw counts, and allocate buffers once at startup if buffer creation counts run high — [Intel GPA guide](https://www.intel.com/content/www/us/en/docs/gpa/user-guide/2023-4/graphics-api-metrics.html)
- Intel's 4th Gen Core developer guide has a dedicated MSAA section (4.3) and covers both DirectX and OpenGL. I could not extract the contents — [Intel guide](https://www.intel.com/content/dam/develop/external/us/en/documents/4th-gen-core-graphics-dev-guide.pdf)
- macOS: deprecated in 10.14, with Metal recommended — [Apple retired docs](https://developer.apple.com/library/mac/documentation/graphicsimaging/conceptual/opengl-macprogguide/opengl_api_versions/opengl_api_versions.html)
- Community reports say GL on Apple Silicon runs over a Metal layer and still works. These are forum posts, not Apple statements — [WineHQ forum](https://forum.winehq.org/viewtopic.php?p=142089); [Apple forums](https://developer.apple.com/forums/thread/694866)
- A community project aims to replace Apple's GL stack with Mesa and Gallium over Metal — [Khronos forum](https://community.khronos.org/t/maybe-macos-is-getting-another-chance-at-modern-opengl/112621)
- Intel's macOS drivers reportedly expose no binary formats — [OpenGL 4 Shading Language Cookbook](https://www.oreilly.com/library/view/opengl-4-shading/9781789342253/bae34249-478a-44d2-b0e6-17b3ce90f067.xhtml)

### Inferences
- **MSAA fix:** because the hint sits inside G3N's window setup, moonBASIC cannot override it from outside without a fork, a `replace` directive to a patched copy, or its own window creation. If moonBASIC renders the scene into its own FBOs for FSR and postfx, a multisampled default framebuffer contributes nothing while costing memory and a resolve. This is especially costly on shared-memory integrated GPUs. Measure before and after on the 780M. Set samples to 0 for the window, and make AA a scene FBO choice or a post-process.
- MSAA at scaled resolution should also be reconsidered. From memory, not confirmed on the page I fetched: FSR 1 expects an anti-aliased input, so some AA must remain. FXAA- or CMAA-class post-AA is the usual cheap choice on weak GPUs, but I did not retrieve a source for this ranking either.
- Suggested low-tier preset, detected from `GL_RENDERER` and a short GPU timer benchmark at first run: no window MSAA, FSR start at 67% with a floor of 50%, 2 cascades with a low-frequency far one, environment-map water with no realtime planar reflection, and capped shadow distance. Save the choice and let the user override it. Values are my judgment, not published.
- Run the one-time benchmark at first launch; adaptive scaling handles the rest. An Intel HD GPU will often sit at the resolution floor, at which point the quality preset is what matters.
- Avoid assuming timer queries are reliable on old Intel or Mac drivers. If queries are missing or results look implausible, fall back to CPU frame delta with vsync off, or to a fixed preset.
- macOS rules: request a 4.1 or 3.3 core forward-compatible context, branch on runtime extension queries, treat binary caching as a no-op when the format count is 0, and precompile on a loading screen every launch. Budget for a future non-GL backend, as Apple could remove GL. No date or removal notice was found.

### Gaps
- Intel's MSAA section and its specific figures could not be read; a person should read section 4.3 of the guide.
- No published frame-time benchmark for G3N or 8x MSAA on 780M or Intel HD class hardware.
- No source on whether macOS GL has timer-query reliability problems or on the behavior of `GL_NUM_PROGRAM_BINARY_FORMATS` on Apple Silicon; test on hardware.
- No Apple timeline for removing OpenGL was found.
