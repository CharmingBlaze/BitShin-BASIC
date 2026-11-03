# Renderer architecture and future backend options for moonBASIC

Research date: 2026-11-03. GitHub activity figures come from the public GitHub API on that date. "Local" sources are files in this repository or the module cache copy of G3N v0.2.0 (`C:\Users\Snow\go\pkg\mod\github.com\g3n\engine@v0.2.0`). G3N links point at the v0.2.0 tag, which is the version moonBASIC pins.

## How do Bevy and Three.js structure rendering, and which ideas fit a small forward engine?

### Takeaway
The ideas worth copying are small: a per-frame snapshot of drawable state, separate opaque, transparent and shadow lists, an integer sort key, a program cache keyed on features, and named shader chunks. A render graph is not worth copying. Bevy itself removed its `RenderGraph` in 0.19 in favour of plain ordered systems.

### Cited Findings

**Bevy**
- Bevy's pipelined renderer (introduced in 0.6) has Extract, Prepare, Queue and Render stages. Extract is "the one synchronization point between the Main World and the Render World". Prepare writes extracted data to the GPU. Queue builds render jobs from it. Render produces the GPU commands — [Bevy 0.6 notes](https://bevy.org/news/bevy-0-6/)
- Extraction exists so that the next frame's app logic can run while the current frame renders, and so that dataflow and ownership are easier to reason about. It should stay as quick as possible because it is the one step that cannot run in parallel, so only the visible entities and required components are copied — [Bevy 0.6 notes](https://bevy.org/news/bevy-0-6/)
- A render phase is "a (potentially sorted) list of Entities to be drawn". Draw functions, composed from smaller commands such as set pipeline, bind material and draw mesh, turn those items into GPU commands, per view — [Bevy 0.6 notes](https://bevy.org/news/bevy-0-6/)
- The current schedule (docs for 0.20.1) orders `ExtractCommands`, `PrepareAssets`, `PrepareMeshes`, `CreateViews`, `Specialize`, `PrepareViews`, `Queue`, `PhaseSort`, `Prepare` (with buffer and bind-group subsets), `Render`, `Cleanup`, `PostCleanup`. Shadow maps are extra views, and `Specialize` selects material and mesh pipeline variants — [RenderSystems docs](https://docs.rs/bevy/latest/bevy/render/enum.RenderSystems.html)
- Each view can have several phases, so shadow-casting lights use the same queue-and-phase machinery as cameras — [Bevy render_phase source (mirror)](https://doc.qu1x.dev/bevy_trackball/src/bevy_render/render_phase/mod.rs.html)
- **Render graph removed:** "Bevy's `RenderGraph` architecture has been replaced with ECS schedules." The old graph "was originally designed when Bevy's ECS was less mature" and "required a lot of boilerplate". Passes are now regular systems ordered with `.after(Core3dSystems::MainPass)` — [Bevy 0.19 notes](https://bevy.org/news/bevy-0-19/)
- Phase items split into binned and sorted variants in 0.14. Transparent and transmissive phases must stay sorted, while others can be binned — [Migration guide 0.13 to 0.14](https://bevy.org/learn/migration-guides/0-13-to-0-14/)
- Bevy 0.15 moved to a retained render world instead of clearing it every frame — [Bevy 0.15 notes](https://bevy.org/news/bevy-0-15/)

**Three.js**
- `WebGLRenderLists` holds three arrays: opaque, transmissive and transparent. Items are pooled and overwritten rather than reallocated — [WebGLRenderLists.js](https://raw.githubusercontent.com/mrdoob/three.js/dev/src/renderers/webgl/WebGLRenderLists.js)
- Opaque sort order is groupOrder, renderOrder, material.id, materialVariant, z, id. Transparent sort order is groupOrder, renderOrder, z descending, id. Material outranks depth for opaque draws, so state grouping is preferred over front-to-back order — [WebGLRenderLists.js](https://raw.githubusercontent.com/mrdoob/three.js/dev/src/renderers/webgl/WebGLRenderLists.js)
- The program cache key is a joined string of the shader ID, defines, scalar parameters (light and shadow counts, tone mapping and so on) and two feature bitmasks, plus `customProgramCacheKey`. `acquireProgram` reference-counts entries in a map, and `releaseProgram` destroys a program at zero uses — [WebGLPrograms.js](https://raw.githubusercontent.com/mrdoob/three.js/dev/src/renderers/webgl/WebGLPrograms.js)
- Shader chunks use a recursive `#include <name>` regex resolved against a `ShaderChunk` table. A prefix of `#define USE_MAP`-style flags is prepended, and `NUM_DIR_LIGHTS`-style placeholders are substituted in the body — [WebGLProgram.js](https://raw.githubusercontent.com/mrdoob/three.js/dev/src/renderers/webgl/WebGLProgram.js)
- Link status checks and uniform lookup are deferred to first use. Where `KHR_parallel_shader_compile` exists, `isReady()` polls completion — [WebGLProgram.js](https://raw.githubusercontent.com/mrdoob/three.js/dev/src/renderers/webgl/WebGLProgram.js)
- `onBeforeCompile` is called with the uniforms and shader source just before compile, and `customProgramCacheKey` tells the cache which settings it depends on. Three.js now calls `WebGPURenderer` with node materials and TSL "the recommended approach" to customisation — [Material docs](https://threejs.org/docs/pages/Material.html)

**Sort keys**
- The classic design packs one integer per draw, most significant first: fullscreen layer, viewport, viewport layer, translucency, depth, material. Sort the array, then submit. Opaque draws sort mainly by material, translucent draws by depth. Commands such as a depth clear can be encoded in the key, and keys can be cached with only depth updated — [Ericson, "Order your graphics draw calls around!"](https://realtimecollisiondetection.net/blog/?p=86)
- bgfx exposes the same choice as per-view modes: `Default`, `Sequential`, `DepthAscending`, `DepthDescending` — [bgfx API](https://bkaradzic.github.io/bgfx/bgfx.html)

**What G3N does today (local reading)**
- `Render` walks the scene, culls against the frustum, then sorts opaque and transparent lists by `RenderOrder` then depth with no material or program term. There is one opaque loop and one transparent loop, and GUI panels share them — [renderer.go](https://github.com/g3n/engine/blob/v0.2.0/renderer/renderer.go)
- Per draw, it allocates new shader defines (`gls.NewShaderDefines()`) and re-sends every light's uniforms. Per frame, it allocates a `zLayers` map — [renderer.go](https://github.com/g3n/engine/blob/v0.2.0/renderer/renderer.go)
- `Shaman.SetProgram` finds programs with a linear scan comparing specs. Specs include current light counts (`DirLightsMax`, `PointLightsMax`), which become shader `#define`s — [shaman.go](https://github.com/g3n/engine/blob/v0.2.0/renderer/shaman.go)
- `zSort` applies `ModelViewMatrix` to `Position()`. That matrix already contains the node's own translation — [renderer.go, lines 290-296](https://github.com/g3n/engine/blob/v0.2.0/renderer/renderer.go). Upstream master still has this code — [master renderer.go](https://raw.githubusercontent.com/g3n/engine/master/renderer/renderer.go)
- `SetObjectSorting` and `Graphic.SetRenderOrder(int)` are public, so sort order can be influenced without patching — [renderer.go](https://github.com/g3n/engine/blob/v0.2.0/renderer/renderer.go); [graphic.go](https://github.com/g3n/engine/blob/v0.2.0/graphic/graphic.go)

### Inferences
- **Depth sort error:** transforming `Position()` by a matrix that already includes that translation counts it twice, so the sort depth is wrong for any node away from its parent's origin. The correct input is the origin. This is my reading of the code, not a reported bug, and it should be confirmed with a test.
- **Light-count hitches:** because light counts feed the shader specs, changing the number of lights in a scene should force a new program variant and a compile on first use. If confirmed, this hits a BASIC script that spawns lights at runtime directly. Fixed light slots with a count uniform would remove it.
- **Best fits, roughly in order of value:**
  1. A draw list built once per view from moonBASIC's handle tables (Bevy's extract and queue, without threads).
  2. A 64-bit sort key: layer, translucency, then program, material and coarse depth for opaque, or depth for transparent (Ericson, Three.js).
  3. A hashed program key with a map lookup instead of a linear scan (Three.js).
  4. Fixed-order named passes as plain function calls: shadow, opaque, sky, transparent, post, UI (Bevy 0.19).
- **Handles help here.** BASIC scripts only see integer handles, so the extract boundary already exists. A renderer change would be invisible to scripts as long as commands keep their behaviour.
- **Poor fits:** a render graph, ECS extraction into a second world, pipelined multithreading, and binned phases at Bevy's scale. They solve problems of plugin extensibility and very large scenes. Sorted arrays are enough here.
- **`onBeforeCompile` for scripts:** a named-chunk injection point fits better than exposing string replacement to BASIC. Three.js's own move away from string patching suggests chunk names become a compatibility burden, so only a few stable hook points should be public.

### Gaps
- I did not verify Bevy's binned phase performance figures. The 0.14 notes were too long for the fetched excerpt to reach that section.
- No measurements exist for how much time G3N's per-draw allocations and light uploads cost in moonBASIC scenes. This needs a profile.

## Forward vs Forward+ vs deferred on OpenGL 3.3 without compute, and is a depth prepass worth it on integrated GPUs?

### Takeaway
Stay forward, and keep MSAA as the reason. Clustering is feasible on GL 3.3 by assigning lights on the CPU and uploading the grid to buffer textures or textures, but Godot's Compatibility renderer shows per-object light lists are the cheaper first step for low-end hardware. A depth prepass should be optional and off by default, with front-to-back sorting done first.

### Cited Findings

**Renderer trade-offs**
- Godot chose forward because it "generally provides a better tradeoff for performance versus flexibility", and because deferred "requires using hacks to be able to use MSAA". It cites less realistic art styles benefiting from MSAA — [Godot internal rendering architecture](https://docs.godotengine.org/en/latest/engine_details/architecture/internal_rendering_architecture.html)
- Godot's Forward+ uses a compute shader to bin lights. It "can greatly speed up rendering performance on desktop hardware" but "is substantially less efficient on mobile" — [Godot internal rendering architecture](https://docs.godotengine.org/en/latest/engine_details/architecture/internal_rendering_architecture.html)
- Godot's Compatibility renderer is non-clustered forward on OpenGL 3.3 core or ES 3.0. It lights objects in a single pass, and draws shadowed lights in additional passes — [Godot internal rendering architecture](https://docs.godotengine.org/en/latest/engine_details/architecture/internal_rendering_architecture.html)
- Light limits: Compatibility allows 8 omni/spot per mesh (can be increased), Mobile 8 per mesh and 256 per view, Forward+ 512 per cluster (can be increased). All allow 8 directional — [Godot renderers doc](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- Cost profile: Compatibility has low base cost and high scaling cost, while Forward+ has high base and low scaling cost and is the slowest on old and low-end hardware. Compatibility offers the best performance on all devices when advanced features are not needed — [Godot renderers doc](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- Intel's Gen11 guide says to "use as few render targets as necessary", avoid unnecessary channels or precision, and "minimize the use of stencil or blend when MSAA is enabled". All of these work against a fat G-buffer on integrated GPUs — [Intel Gen11 guide](https://www.intel.com/content/www/us/en/developer/articles/guide/developer-and-optimization-guide-for-intel-processor-graphics-gen11-api.html)
- Clustered shading was introduced for both deferred and forward use, with clustering by 3D position and/or normal. The authors report fewer lighting computations than tiled shading — [Olsson, Billeter and Assarsson, HPG 2012](https://research.chalmers.se/en/publication/161725); [preprint](https://www.cse.chalmers.se/~uffe/clustered_shading_preprint.pdf)
- The follow-up extends clustered forward to transparency and MSAA, building the grid from a pre-pass over geometry — [Olsson et al., SIGGRAPH 2012](https://research.chalmers.se/publication/161726)

**Clustered lights without compute: existing designs**
- **Bevy** selects at startup. GPU clustering needs `COMPUTE_SHADERS` and storage buffers. Otherwise it falls back to CPU clustering, where `extract_clusters_for_cpu_clustering` flattens main-world cluster lists and `prepare_clusters_for_cpu_clustering` writes them to buffers — [bevy_pbr cluster/mod.rs](https://docs.rs/crate/bevy_pbr/latest/source/src/cluster/mod.rs)
- **Bevy's uniform limits** are shaped by the 16 KB limit: 204 lights, 1024 `UVec4` offset items, and 16384 one-byte indices packed as 16 per `UVec4`. Extra lights are dropped — [bevy_pbr cluster/mod.rs](https://docs.rs/crate/bevy_pbr/latest/source/src/cluster/mod.rs)
- **Filament** uses a light UBO of 4 float4 per light with 256 lights maximum, a per-froxel `RG_U16` light list texture holding offset, point count and spot count, and a record UBO packing 16 indices per uint4. Defaults are 4096 froxels in 16 KiB and 16384 records, chosen because 16 KiB is "the minimum guaranteed in GLES 3.x and Vulkan 1.1" — [Froxelizer.h](https://raw.githubusercontent.com/google/filament/main/filament/src/Froxelizer.h)
- **Filament's CPU budget:** its private arena is about 256 KiB, and froxelization is split into 8 jobs for 256 lights — [Froxelizer.h](https://raw.githubusercontent.com/google/filament/main/filament/src/Froxelizer.h)
- **GL limits:** `GL_MAX_UNIFORM_BLOCK_SIZE` must be at least 16384 and `GL_MAX_TEXTURE_BUFFER_SIZE` at least 65536 — [glGet reference](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glGet.xhtml)
- **Buffer textures** are core from GL 3.1, with formats including R8UI, R16UI, R32UI, RG16UI and RGBA32F. Reads past the clamped texel count are undefined — [glTexBuffer reference](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glTexBuffer.xhtml)

**Depth prepass**
- Arm says a prepass must double draw calls and processed vertices, that the cost "usually outweigh[s] the benefits" given Forward Pixel Kill, and advises against using one to remove overdraw. It recommends opaque front to back, then transparent back to front, plus CPU frustum culling — [Arm GPU Best Practices](https://documentation-service.arm.com/static/67a62b17091bfc3e0a947695?token=); [online version](https://support.arm.com/documentation/101897/0304/Optimizing-application-logic/Draw-call-batching-best-practices)
- From Mali-G725 and Immortalis-G925 onward, Fragment Prepass removes overdraw in hardware, and front-to-back sorting can be disabled when no incompatible draws are present — [Arm performance counters guide](https://support.arm.com/documentation/109793/0102/Content-behavior?lang=en)
- Unity's Depth Priming "isn't supported if you use a deferred rendering path or Multisample Anti-aliasing, or at runtime on mobile devices that use tile-based deferred rendering". Unity says the improvement "depends on the number of overlapping pixels and the complexity of the pixel shaders", and that adding the prepass "has an impact on memory and performance" — [Unity URP docs](https://docs.unity.com/en-us/engine/current/manual/render-pipelines/universal-render-pipeline/urp-reference/urp-universal-renderer)
- With depth writes on, any `discard` compiled into a shader defers the depth write and lowers early culling. In MJP's test, front-to-back pixel shader invocations rose from about 440,640 to 582,000. A depth-only alpha-tested prepass followed by an EQUAL test avoids this — [MJP, "To Early-Z, or Not To Early-Z"](https://therealmjp.github.io/posts/to-earlyz-or-not-to-earlyz/)
- Intel HD/UHD 620/630 have a reported driver issue where early-Z is wrongly applied when a fragment shader writes depth — [iTwin.js compatibility note](https://www.itwinjs.org/reference/webgl-compatibility/compatibility/graphicsdriverbugs/fragdepthdoesnotdisableearlyz)

**moonBASIC state (local)**
- Status doc: "Not Vulkan (a later renderer replaces this one; do not wrap Vulkan on G3N) or a deferred renderer" — `docs/STATUS.md`, line 3
- Compute (`#version 430`) and tessellation (`#version 400 core`) shaders exist but are optional, with features detected after context creation — `internal/runtime/glmodern_shaders.go`, `internal/runtime/g3napp.go`

### Inferences
- **Deferred is a poor fit.** It conflicts with the MSAA that the engine exposes through `SetMSAA`, and with the transparent and custom-shader paths that would still need a forward pass. On integrated GPUs, several full-screen render targets cost bandwidth on every frame even in light scenes.
- **Suggested light path:**
  - **Step 1:** select the top N lights per object on the CPU, in a shader with fixed slots. That matches Godot Compatibility, removes light-count recompiles, and is a small change.
  - **Step 2:** add CPU clustering only if games need dozens of lights in view. Light scale is the only benefit, and Godot's cost data says the base cost rises on weak GPUs.
- **Concrete GL 3.3 clustering layout, adapted from Filament and Bevy:**
  - A grid of about 16x9x24 in a `usampler3D` or a buffer texture, storing offset and counts (RG16UI, or RG32UI if needed).
  - An index list in an R16UI buffer texture.
  - Light data in a 16 KB UBO, or an RGBA32F buffer texture to exceed about 200 lights.
  - Assign on the CPU each frame, and read with `texelFetch`.
  - The grid dimensions are my starting suggestion and are not taken from a source.
- **Portability catch:** buffer textures are a GL 3.1 feature, and ES 3.0 is not listed as having them in the sources I read. If ANGLE on Metal (ES 3.0 only) is a future target, a plain 2D or 3D integer texture is the safer storage.
- **Prepass recommendation:**
  - First fix the sort, since a correct front-to-back opaque order is free.
  - Offer a prepass as a quality or material option for heavy PBR scenes with alpha-tested foliage.
  - Consider a partial prepass of large occluders such as terrain and buildings.
  - A forward PBR shader on an immediate-mode integrated GPU is the case where it could pay off, but it needs measuring. It also doubles vertex work, which matters for skinned meshes.
- **MSAA changes the numbers.** Unity disabling priming under MSAA is a signal that the combination is not simply beneficial. A moonBASIC prepass must render into the same multisampled target as the colour pass, or the depth will not match.

### Gaps
- **No primary vendor statement on prepass for Intel or AMD integrated GPUs.** The Intel Gen11 guide is silent on depth prepass, early-Z, HiZ and draw order. Only a benchmark on target hardware can settle it.
- **Apple Silicon:** I found no primary Apple statement on prepass. The Arm guidance applies to Mali only, and extending it to other tile-based GPUs is an inference.
- **Bevy's CPU clustering cost and Filament's frame cost** were not found as numbers. No source gives a Go-specific cost for per-frame CPU light assignment.
- **Godot's Compatibility prepass behaviour** was not stated in the pages I read.
- **Filament's design docs** could not be read (the page only redirected), so its stated reasons for choosing clustered forward are not cited.

## What is the current state of bgfx, wgpu-native, Vulkan, GoGPU and G3D?

### Takeaway
No option is both production-ready and usable from Go without significant work. G3D is a small early library and far behind what moonBASIC already ships. The strongest native candidates are bgfx and wgpu-native, but bgfx has no maintained Go binding and the wgpu-native Go bindings are beta or community forks. Raw Vulkan in Go has only stale bindings.

### Cited Findings

**G3D (`gogpu/g3d`): verified by cloning the repository and reading the source**
- **Size and history:** 40 commits since the first on 2026-04-05, 29 by one author. Latest release is v0.1.11 (2026-10-30), with 28 stars — [g3d repository](https://github.com/gogpu/g3d)
- **Code size:** about 3,800 lines of non-test Go in the root package (`renderer.go` is 886), and two WGSL shaders totalling 214 lines — [g3d repository](https://github.com/gogpu/g3d)
- **Description versus reality:** the repository description says "Scene graph, PBR, GLTF", but the README roadmap marks only Phase 1 complete. Full PBR, shadows, normal maps, textures, glTF, skeletal animation, morph targets, instancing, environment maps, post-processing and skybox are all "Planned" — [g3d README](https://github.com/gogpu/g3d)
- **"PBR" is Blinn-Phong:** the header of `standard.wgsl` reads "Blinn-Phong lit shader", and roughness only maps to a shininess exponent (`mix(256.0, 2.0, material.roughness)`) — [standard.wgsl](https://github.com/gogpu/g3d/blob/main/internal/gpu/shaders/standard.wgsl)
- **Lights:** only ambient and directional kinds exist, with a fixed `array<LightData, 4>` — [light.go](https://github.com/gogpu/g3d/blob/main/light.go); [standard.wgsl](https://github.com/gogpu/g3d/blob/main/internal/gpu/shaders/standard.wgsl)
- **Materials:** `StandardMaterial` has colour, metallic, roughness and emissive factors, and no texture samplers appear in the renderer or shaders. The only textures found are the depth and surface targets — [material_standard.go](https://github.com/gogpu/g3d/blob/main/material_standard.go); [renderer.go](https://github.com/gogpu/g3d/blob/main/renderer.go)
- **MSAA:** none. `NoMultisample()` and `SampleCount: 1` are used — [helpers.go](https://github.com/gogpu/g3d/blob/main/internal/gpu/helpers.go); [renderer.go](https://github.com/gogpu/g3d/blob/main/renderer.go)
- **No loaders:** a source search found no shadow, glTF loader, point light, spot light, skeletal or morph code. "GLTF" appears only in comments about conventions — [g3d repository](https://github.com/gogpu/g3d)
- **Recent fixes show how early it is:** v0.1.11 fixed a sphere rendered inside-out and flipped plane UVs. v0.1.8 fixed duplicate ambient lights and non-indexed geometry — [g3d CHANGELOG](https://github.com/gogpu/g3d/blob/main/CHANGELOG.md)
- **Stated scope:** a rendering library, explicitly not a game engine. It excludes ECS, physics, audio, networking, scripting and an editor — [g3d README](https://github.com/gogpu/g3d)

**GoGPU (`gogpu/gogpu`, `gogpu/wgpu`)**
- `gogpu/gogpu`: created 2025-12-05, latest release v0.54.0 (2026-08-31), 430 stars, 37 open issues and pull requests. The README makes no stability or production-ready claim and asks for help testing macOS, Linux and Windows DX12. Its acknowledgments thank early testers from when "nothing worked and every platform was broken" — [gogpu repository](https://github.com/gogpu/gogpu)
- `gogpu/wgpu`: a pure-Go WebGPU implementation with Vulkan, Metal, DX12, GLES (GL 3.3+, ES 3.0+) and software backends. Latest is v0.34.7 (2026-11-03), so it is still pre-1.0. It has 196 stars and very frequent releases — [gogpu/wgpu](https://github.com/gogpu/wgpu)
- Listed limitations: no Metal timestamp queries, Android preview only, ray tracing experimental, Metal untested on iOS — [gogpu/wgpu](https://github.com/gogpu/wgpu)
- **CGO conflict:** the `gogpu/gogpu` README says "`CGO_ENABLED=0` (Pure Go FFI requires CGO disabled)" and "Always set `CGO_ENABLED=0`" — [gogpu README](https://raw.githubusercontent.com/gogpu/gogpu/main/README.md). The `gogpu/wgpu` README says both `CGO_ENABLED=0` and `CGO_ENABLED=1` work, for example for the race detector — [gogpu/wgpu README](https://raw.githubusercontent.com/gogpu/wgpu/main/README.md)
- The README says known users include born-ml/born and ironwail-go, a Quake engine port — [gogpu repository](https://github.com/gogpu/gogpu)

**wgpu-native and Go bindings**
- wgpu-native: latest v29.0.1.1 (2026-06-23), last push 2026-10-26, 1,419 stars, automated builds for macOS, Windows and Linux. MSRV bumps count as breaking changes — [wgpu-native](https://github.com/gfx-rs/wgpu-native)
- It lists `go-webgpu/webgpu` as its Go wrapper — [wgpu-native](https://github.com/gfx-rs/wgpu-native)
- `go-webgpu/webgpu`: self-described "**Beta**", v0.5.5 (2026-08-02), wgpu-native v29, zero-CGO via goffi, 69 stars — [go-webgpu README](https://raw.githubusercontent.com/go-webgpu/webgpu/main/README.md)
- **Unmaintained:** `cogentcore/webgpu`'s README says to use `oliverbestmann/webgpu` as "the currently-maintained fork", and that the original "will not be updated" — [cogentcore/webgpu README](https://raw.githubusercontent.com/cogentcore/webgpu/main/README.md). `cogentcore/go-webgpu` is archived — [cogentcore/go-webgpu](https://github.com/cogentcore/go-webgpu)
- `oliverbestmann/webgpu` tracked wgpu-native v27.0.2.0 at the time of the indexed README, and its prebuilt libraries exceed `go get`'s 512 MB module limit, so it needs an update script — [oliverbestmann/webgpu](https://github.com/oliverbestmann/webgpu)

**bgfx**
- Actively developed: last push 2026-11-01, 17,604 stars — [bgfx](https://github.com/bkaradzic/bgfx)
- Backends include D3D11, D3D12, Metal, OpenGL 2.1 and 3.1+, GLES 2 and 3.1, Vulkan, WebGL and WebGPU (Dawn only). Platforms include macOS 13+. BSD-2 licensed — [bgfx overview](https://bkaradzic.github.io/bgfx/overview.html)
- The only Go binding its docs list is `james4k/go-bgfx` — [bgfx overview](https://bkaradzic.github.io/bgfx/overview.html)
- **Unmaintained:** `go-bgfx` was archived on 2026-07-22 with its last push in 2014. The README warns "likely only works on OS X at this point" — [go-bgfx](https://github.com/james4k/go-bgfx)
- Shaders use a GLSL-like language with `varying.def.sc`, `$input` and `$output`, and float-only uniforms. They are compiled per backend with the `shaderc` tool. D3D targets can only be compiled on Windows — [bgfx tools](https://bkaradzic.github.io/bgfx/tools.html)
- Multithreaded submission uses per-thread encoders. `getInternalData` and `overrideInternal` provide limited native interop, and the docs warn you are expected to "understand some bgfx internals" — [bgfx API](https://bkaradzic.github.io/bgfx/bgfx.html)

**Vulkan in Go**
- **Stale:** `vulkan-go/vulkan` last pushed 2023-08-05 (808 stars). `goki/vulkan` last pushed 2024-07-20 (10 stars). Neither is archived — [vulkan-go/vulkan](https://github.com/vulkan-go/vulkan); [goki/vulkan](https://github.com/goki/vulkan)

**G3N and related**
- G3N: latest tagged release is v0.2.0 (2021-08-19), but upstream is not dead. Commits on master include a FlyControl camera (2026-01) and a VRM loader (2026-08). 3,121 stars — [g3n/engine](https://github.com/g3n/engine)

### Inferences
- **G3D is not a candidate today.** moonBASIC already has PBR, shadows, skinning, FBX, post-processing, particles and terrain, while G3D lacks textures on materials. Adopting G3D would mean writing the renderer yourself on top of it. Treat it as a possible design reference for WebGPU usage and nothing more.
- **`gogpu/wgpu` alone is the only part of GoGPU worth tracking.** It has a real release pace and claims to accept `CGO_ENABLED=1`. The `gogpu/gogpu` window framework's stated `CGO_ENABLED=0` requirement rules it out for moonBASIC.
- **Zero-CGO is not an advantage here.** moonBASIC already needs CGO through G3N's `gls`, go-gl/glfw and native Jolt and cimgui dependencies, as its README's "C compiler" requirement shows.
- **Any WebGPU or bgfx move is a renderer rewrite.** Between 40 and 50 `#version 330 core` shader sources in `internal/runtime` would need porting, and `user_shader.go` suggests scripts may be able to supply GLSL, which would make the shader language a public compatibility question.
- **Rough ranking if a second backend is ever funded:**
  1. wgpu-native or `gogpu/wgpu` behind moonBASIC's own thin renderer. A single API maps to Metal, Vulkan and DX12, at the cost of pre-1.0 churn on both.
  2. bgfx with a self-maintained C binding. The renderer is proven and it has a sort-key model, but it adds a C++ build and a shader toolchain, and the binding would be yours to own.
  3. Raw Vulkan, which is the highest work and has stale Go bindings.
- **Timing:** `docs/STATUS.md` and `docs/ROADMAP.md` already place other backends out of scope for now. Nothing found here argues for changing that before the G3N draw path is fixed.

### Gaps
- Whether `gogpu/wgpu` can create a surface on an existing GLFW window, or coexist with cgo-linked Jolt and cimgui in one process, is unverified. It needs a spike.
- I did not verify whether `gogpu/naga` can compile GLSL 330 input, which would affect a porting path.
- The "1.25M+ lines of code" ecosystem claim in the G3D README, and its labelling of backends as stable, were not independently checked.
- I found no independent performance comparison between `gogpu/wgpu` and wgpu-native.
- No authoritative list of shipped commercial games for bgfx or wgpu-native was collected. bgfx's overview page makes no maturity claim.
- Current `oliverbestmann/webgpu` activity was not checked through the API.

## OpenGL on macOS is deprecated: what are the practical options?

### Takeaway
Native OpenGL 4.1 still works and is the zero-effort option, so keep it as long as it runs, and accept the lack of compute. ANGLE over Metal is the practical insurance policy, but it means ES 3.0 and a G3N fork. Zink on macOS is experimental and not shippable. MoltenVK is only relevant if the engine moves to Vulkan, and Khronos now recommends KosmicKrisp on Apple Silicon.

### Cited Findings
- Apple's 2018 announcement said apps using OpenGL "will continue to run in macOS 10.14, but these legacy technologies are deprecated", and that graphics apps should adopt Metal. This wording was read through press coverage rather than Apple's page — [AppleInsider](https://appleinsider.com/articles/18/06/04/opengl-opencl-deprecated-in-favor-of-metal-2-in-macos-1014-mojave); [Slashdot](https://apple.slashdot.org/story/18/06/05/1719205/apple-deprecates-opengl-and-opencl-in-macos-1014-mojave)
- Apple's support list tops out at OpenGL 4.1, on an archived page no longer updated — [Apple support](https://support.apple.com/en-us/101525)
- **ANGLE:** on Metal it is complete for ES 2.0 and 3.0, and ES 3.1 is not listed. The Mac OS X row shows Metal "complete" (macOS 10.14+) and Desktop GL "deprecated" — [ANGLE README](https://raw.githubusercontent.com/google/angle/main/README.md)
- **Godot's choice:** the OpenGL driver is paired only with Compatibility and "can run through ANGLE on Windows and macOS". Godot has a native Metal driver for Apple Silicon (Metal 4 where supported since 4.7, Metal 3 otherwise). MoltenVK is "used as a fallback when native Metal support is not available (e.g. on x86 macOS)" — [Godot internal rendering architecture](https://docs.godotengine.org/en/latest/engine_details/architecture/internal_rendering_architecture.html)
- Godot falls back from ANGLE to native GL when ANGLE over Metal is unsupported, using `rendering/gl_compatibility/fallback_to_native` (default on), which was added to stop crashes on older Macs — [Godot PR 83753](https://github.com/godotengine/godot/pull/83753)
- **GLFW:** the `GLFW_ANGLE_PLATFORM_TYPE` init hint accepts `GLFW_ANGLE_PLATFORM_TYPE_METAL`, and must be set before `glfwInit` — [GLFW intro guide](https://www.glfw.org/docs/latest/intro_guide.html)
- **Zink:** "Zink on macOS is experimental with very limited capabilities", running on KosmicKrisp or MoltenVK — [Mesa Zink docs](https://docs.mesa3d.org/drivers/zink.html)
- GL 3.3 through Zink needs `dualSrcBlend`, a vertex attribute divisor extension, and packed 10-bit vertex formats — [Mesa Zink docs](https://docs.mesa3d.org/drivers/zink.html)
- Community reports show Minecraft and Wolfenstein: The New Order running through Zink and KosmicKrisp, using custom Mesa builds and patches — [Minecraft gist](https://gist.github.com/lucamignatti/5312f5e937de2ba44256ecba6de54cc2); [wolf-kosmickrisp](https://github.com/DmitriiTrifonov/wolf-kosmickrisp)
- **KosmicKrisp:** LunarG's Mesa Vulkan driver is Vulkan 1.4 conformant on Apple Silicon. It requires M1 or later and macOS 26 or later because it needs Metal 4. Intel Macs are unsupported. Apps can bundle it with MoltenVK and pick at runtime. The article's own dating conflicts (2026 in the header, 2025 in the body) — [LunarG](https://www.lunarg.com/kosmickrisp-achieves-vulkan-1-4-conformance-on-apple-silicon/)
- **MoltenVK:** "an almost-complete subset of the Vulkan 1.4" API, "not fully compliant". Its README says Khronos has deprecated it, recommending KosmicKrisp on Apple Silicon and MoltenVK for Intel Macs and iOS. Latest release v1.4.2 (2026-07-24) — [MoltenVK README](https://raw.githubusercontent.com/KhronosGroup/MoltenVK/main/README.md)
- **G3N blocks ANGLE as it stands:** on macOS, `gls/glapi.c` opens `/System/Library/Frameworks/OpenGL.framework/OpenGL` directly at line 63 — [glapi.c](https://github.com/g3n/engine/blob/v0.2.0/gls/glapi.c). Desktop builds fix `GLSL_VERSION` to "330 core", while the browser build already uses "300 es" — [version-desktop.go](https://github.com/g3n/engine/blob/v0.2.0/renderer/version-desktop.go); [version-browser.go](https://github.com/g3n/engine/blob/v0.2.0/renderer/version-browser.go)
- **moonBASIC today:** 27 runtime files import `go-gl/gl/v3.3-core/gl`, and macOS builds run in CI (macos-14) — local `internal/runtime`, `docs/ROADMAP.md` line 1422

### Inferences
- **Practical order for moonBASIC:**
  1. Keep native GL on macOS and document that compute features are off, since 4.1 has no compute shaders.
  2. Keep new shaders ES 3.0-friendly where that costs little (no buffer textures or geometry shaders in core paths).
  3. Treat ANGLE as the migration path if Apple actually removes GL.
- **ANGLE is not a drop-in.** It requires a G3N `gls` loader change, an ES-compatible GL binding replacing `v3.3-core` in 27 files, translating `#version 330 core` shaders, and shipping libEGL and libGLESv2 next to the binary. G3N's existing `300 es` browser path means its stock shaders are likely close, which reduces but does not remove the work.
- **Zink is ruled out for shipping today:** experimental status, custom Mesa builds, and KosmicKrisp's macOS 26 floor. It may become an end-user workaround if GL is removed, and it would run existing GL 3.3 shaders unmodified, which is its attraction.
- **Apple Silicon and Intel split:** any Vulkan route needs two drivers (KosmicKrisp and MoltenVK) to cover both, whereas ANGLE covers macOS 10.14 onward with one.

### Gaps
- I could not read Apple's own Mojave release notes (the page returned only a title), so the deprecation wording is via press. I found no Apple statement of a removal date.
- I did not test moonBASIC under ANGLE on a Mac, or confirm which of its shaders exceed ES 3.0.
- Whether `go-gl/gl` offers a suitable GLES 3.0 package was not verified.
- Performance of native GL on Apple Silicon versus ANGLE was not found from a primary source.
- Commercial GL-to-Metal layers were not investigated.

## Is it worth forking G3N or writing a draw loop beside it, and how are go-gl/glfw and window hints handled?

### Takeaway
Yes to both, in small steps: a small vendored patch of G3N through the `replace` mechanism moonBASIC already uses, plus moonBASIC's own draw list for 3D, reusing G3N's math, loaders and GL wrapper. A full fork for its own sake is not needed. The 8x MSAA hint cannot be overridden from outside G3N, because `window.Init` sets it between `glfw.Init` and `CreateWindow` with no hook.

### Cited Findings

**Window hints**
- `window.Init` calls `glfw.Init()`, sets ContextVersion 3.3, core profile and `glfw.Samples, 8`, forward-compat on macOS, then calls `CreateWindow` and `gls.New()` in the same function, with no options — [glfw.go, lines 217-266](https://github.com/g3n/engine/blob/v0.2.0/window/glfw.go)
- Upstream master still has `glfw.Samples, 8` and go-gl/glfw v3.3 — [master glfw.go](https://raw.githubusercontent.com/g3n/engine/master/window/glfw.go); [master go.mod](https://raw.githubusercontent.com/g3n/engine/master/go.mod)
- `win` is an unexported package variable and `window.Init` panics if called twice — [window.go](https://github.com/g3n/engine/blob/v0.2.0/window/window.go); [glfw.go](https://github.com/g3n/engine/blob/v0.2.0/window/glfw.go)
- GLFW hints stay in effect until changed, and are reset by `glfwInit` or `glfwDefaultWindowHints`. `GLFW_SAMPLES` defaults to 0, which disables multisampling — [GLFW window guide](https://www.glfw.org/docs/latest/window_guide.html)
- Init hints only count before `glfwInit` — [GLFW intro guide](https://www.glfw.org/docs/latest/intro_guide.html)
- go-gl/glfw is maintained (last push 2026-08-23) and has `v3.3` and `v3.4` packages — [go-gl/glfw](https://github.com/go-gl/glfw). moonBASIC pins v3.3 at a 2021 commit — local `go.mod`. GLFW's docs are now at 3.5.1 — [GLFW news](https://www.glfw.org/docs/latest/news.html)
- moonBASIC's secondary windows already set their own hints, also including `glfw.Samples, 8`, and `hideWindow` sets hints before `window.Init` — local `internal/runtime/extra_window.go` lines 170-179, `internal/runtime/smoke.go` lines 86-88, `internal/runtime/g3napp.go`

**MSAA interaction**
- moonBASIC has its own multisampled scene target: `SetMSAA` (2, 4, 8 or 0) renders to `msaaFBO`, resolved by `resolveMSAA` — local `internal/runtime/postfx.go` lines 25-29, 259-273, 570-582
- GL blit rules: `GL_INVALID_OPERATION` if `GL_SAMPLE_BUFFERS` is greater than zero on either the read or draw buffer and the source and destination rectangles differ in size — [glBlitFramebuffer](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glBlitFramebuffer.xhtml)
- `blitSceneCopy` blits `ww,hh` to `dw,dh` into framebuffer 0 — local `internal/runtime/postfx.go` lines 440-443
- moonBASIC turns off G3N's GL error checks because "G3N's C glapi wrapper calls exit(1) on any GL error" — local `internal/runtime/g3napp.go` lines 53-57

**How much of G3N is really used (local counts)**
- 137 runtime files import G3N. By package: `math32` 91, `core` 54, `gls` 52, `geometry` 39, `material` 26, `graphic` 26, `texture` 17, `camera` 16, `renderer` 11, `window` 9, `light` 8, `gui` 7, `loader/gltf` 2 — local grep of `internal`
- A parallel raw GL path already exists: 27 files import `go-gl/gl/v3.3-core/gl`, with five separate `gl.Init()` sites. `glmodern_draw.go` (344 lines) issues `DrawElementsInstanced` itself, and `batch.go`, `postfx.go` (1,339 lines) and `shadows.go` (1,605 lines) are raw GL too — local `internal/runtime`
- `rend.Render` is called from several places: `extra_window.go`, `postfx.go` (3 sites) and `water.go` — local grep
- `go.mod` already uses `replace github.com/SanderMertens/flecs-go => ./third_party/flecs-go` for vendoring — local `go.mod`
- G3N's licence is BSD-style — [G3N LICENSE](https://github.com/g3n/engine/blob/v0.2.0/LICENSE)
- `renderer.NewRenderer` takes a `*gls.GLS`, and `gls.New()` is public — [renderer.go](https://github.com/g3n/engine/blob/v0.2.0/renderer/renderer.go); [gls-desktop.go](https://github.com/g3n/engine/blob/v0.2.0/gls/gls-desktop.go)

### Inferences
- **No clean override exists.** Hints set beforehand are overwritten for `Samples` by `window.Init`. Options without patching are: disable multisample rasterisation with `gl.Disable(gl.MULTISAMPLE)` and render through FBOs, which leaves the memory allocated, or copy the window code and own the window. Using `go:linkname` to reach `win` would be fragile.
- **Cost of the wasted 8x:** at 1920x1080, an 8x RGBA8 plus D24S8 default framebuffer is about 66 MB + 66 MB before compression (my arithmetic, 2.07M pixels x 8 samples x 4 bytes each). When the scene goes through `msaaFBO` or a post target anyway, that default buffer's samples add nothing. The driver may also honour a different count, since this is a request, so it should be measured.
- **Possible latent bug:** if the default framebuffer really is multisampled, `blitSceneCopy` with `dw,dh` different from `ww,hh` (HiDPI) meets the `GL_INVALID_OPERATION` rule above. With error checks off, this would fail silently, leaving a blank or stale frame. Worth testing on a scaled display. A one-sample default framebuffer removes the problem.
- **Recommended shape:**
  1. **Vendor G3N** into `third_party/g3n` with `replace`, keeping the diff minimal: a `window.InitWithOptions` or package-level hint variable for Samples (default 0 or 1), the `zSort` correction, removal of per-draw and per-frame allocations, hashed program lookup, and fixed-slot lights. Roughly a few hundred lines by my estimate, not a measured figure. Offer it upstream since upstream is active, but do not depend on acceptance.
  2. **Build moonBASIC's own draw list for world geometry** beside G3N, extending the existing `glmodern` path. Only 11 files import `renderer`, so this replacement is much smaller than the 137 that touch G3N at all. Keep `math32`, `gls`, loaders and `geometry` as they are.
  3. **Do not rewrite everything.** `math32` alone is in 91 files, and replacing it gains nothing for rendering.
- **Non-fork interim step:** since `SetRenderOrder` is public, a moonBASIC-computed key could be fed into `RenderOrder` to get program grouping without patching. It is limited to an `int` and still runs the buggy depth comparison on ties, so it is a stopgap at best.
- **Two GLFW copies and upgrades:** `g3napp.go` already works around Ebiten registering its own GLFW window class. Moving to go-gl/glfw v3.4 needs every importer changed together. G3N, `internal/runtime` and any other direct user must agree, or two C copies get linked. A vendored G3N makes that possible, and it is also a prerequisite for ANGLE hints.
- **Consolidate `gl.Init()`** to one call at startup while doing this. Five call sites suggest order-dependent setup, though I did not trace whether this causes actual problems.

### Gaps
- I could not confirm from the GLFW pages which release added `GLFW_ANGLE_PLATFORM_TYPE`. From memory it is 3.4, so check that the v3.3 binding lacks it before relying on this.
- Actual sample counts returned on target drivers, and the real memory cost of the default framebuffer, need a runtime check with `glGetIntegerv(GL_SAMPLES)`.
- The light-count recompile and double-translation behaviours are from code reading only, and have not been run.
- I did not estimate the number of moonBASIC features that rely on G3N's `Graphic.Render` behaviour for each material type. That count determines the true size of a custom draw list.
