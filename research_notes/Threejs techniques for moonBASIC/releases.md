# three.js releases, r180 to r187: rendering changes (checked 2026-11-05)

Method and limits, for the report writer:

- Release data came from the unauthenticated GitHub REST API (`/repos/mrdoob/three.js/releases`, `/releases/latest`), the npm registry, and raw files at tag `r187`.
- `github.com/mrdoob/three.js/wiki/Migration-Guide` failed through WebFetch ("socket hang up"). I read the same wiki content from `raw.githubusercontent.com/wiki/mrdoob/three.js/Migration-Guide.md`.
- Most per-release items below are release-note titles only. I read the full PR text only where a finding quotes or paraphrases it.
- The labels [NEW], [IMPROVED], [DEPRECATED] and [REMOVED] are my classification from the wording of titles and the migration guide. They are not project labels. A rename is marked [DEPRECATED: rename]. The guide often does not say whether the old name still works.

Link shorthand: `#NNNNN` always links to `github.com/mrdoob/three.js/pull/NNNNN`. Release notes are at `github.com/mrdoob/three.js/releases/tag/rNNN`.

## What is the latest release number and date?

### Takeaway
The latest release is **r187, published 2026-10-30**, confirmed directly by the GitHub API and by npm (`three@0.187.0`, dist-tag `latest`). GitHub and npm dates differ by up to about 19 days for several earlier releases, so the table gives both.

### Cited Findings
- `/releases/latest` returns tag `r187`, published 2026-10-30T08:03:18Z — [GitHub r187](https://github.com/mrdoob/three.js/releases/tag/r187)
- npm's `latest` dist-tag is `0.187.0`, published 2026-10-30T08:09:24Z — [npm registry](https://registry.npmjs.org/three)

| Release | GitHub published | npm x.0 published | Later npm patches | Notes |
|---|---|---|---|---|
| [r187](https://github.com/mrdoob/three.js/releases/tag/r187) | 2026-10-30 | 2026-10-30 | none | latest |
| [r186](https://github.com/mrdoob/three.js/releases/tag/r186) | 2026-09-24 | 2026-09-08 | 0.186.1 on 09-24 | |
| [r185](https://github.com/mrdoob/three.js/releases/tag/r185) | 2026-07-01 | 2026-06-25 | 0.185.1 on 07-01 | |
| [r184](https://github.com/mrdoob/three.js/releases/tag/r184) | 2026-04-16 | 2026-04-16 | none | |
| [r183](https://github.com/mrdoob/three.js/releases/tag/r183) | 2026-02-20 | 2026-02-18 | 0.183.1 on 02-20, 0.183.2 on 02-28 | |
| [r182](https://github.com/mrdoob/three.js/releases/tag/r182) | 2025-12-10 | 2025-12-10 | none | |
| [r181](https://github.com/mrdoob/three.js/releases/tag/r181) | 2025-11-19 | 2025-10-31 | 0.181.1 on 11-10, 0.181.2 on 11-19 | |
| [r180](https://github.com/mrdoob/three.js/releases/tag/r180) | 2025-09-03 | 2025-09-03 | none | context, just outside 12 months |

### Inferences
- The GitHub timestamp usually matches the last npm patch rather than the first (r186, r185, r183 approximately, r181). The npm x.0 date is probably the better "first shipped" date.
- Cadence over this window was about every 1 to 3 months, not monthly. mrdoob still describes the goal as "a release per month" in a [#33920 comment](https://github.com/mrdoob/three.js/pull/33920).

### Gaps
- None for the release number. I did not check for git tags newer than the latest published release.

## Is there a Nanite-like virtualized geometry feature? (added scope)

### Takeaway
**The report is confirmed, with qualifications.** Official three.js work exists and was merged by a core maintainer (sunag). It is an **experimental addon**, not core, and it is WebGPU-backend only.

- A Nanite-style software rasterizer example landed in **r185** (PR opened 2026-05-19, merged 2026-05-21).
- The usable feature, `MeshletDrawingPipeline` and `VisibilityBufferDrawingPipeline`, landed in **r187** (PR #35191, merged 2026-10-29, released 2026-10-30).
- It shares Nanite's broad ideas but is not a claimed Nanite equivalent. The r187 source says LOD is selected per object, and I found no streaming code.

### Cited Findings
**Where each part lives**
- **Core (`src/`):** only the hook. `renderer.drawingPipeline` and the default `StandardDrawingPipeline` are in `src/renderers/common/`; the old `Renderer._projectObject()` moved there "unchanged" so addons can replace it "without patching the renderer" — [#35191](https://github.com/mrdoob/three.js/pull/35191)
- **Addon (`examples/jsm/`):** `pipelines/MeshletDrawingPipeline.js` (5,571 lines), `pipelines/VisibilityBufferDrawingPipeline.js` (954 lines), `utils/MeshletGeometryUtils.js` (417 lines) — [#35191 file list](https://github.com/mrdoob/three.js/pull/35191/files)
- **Import path and use:** `three/addons/pipelines/MeshletDrawingPipeline.js`, then `renderer.drawingPipeline = new MeshletDrawingPipeline();` — [#35191](https://github.com/mrdoob/three.js/pull/35191)
- **Examples at the r187 tag:** `webgpu_meshlets`, `webgpu_meshlets_city`, `_ibl`, `_instanced`, `_materials`, `_meshes` — [files.json at r187](https://github.com/mrdoob/three.js/blob/r187/examples/files.json). They were renamed from `webgpu_gpudriven_*` — [#35282](https://github.com/mrdoob/three.js/pull/35282)
- **Status:** both classes state "This class is experimental and its interface might change." — [MeshletDrawingPipeline.js at r187](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/pipelines/MeshletDrawingPipeline.js), [VisibilityBufferDrawingPipeline.js at r187](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/pipelines/VisibilityBufferDrawingPipeline.js)

**Timeline**
- r181: [NEW] `dispatchWorkgroupsIndirect` — [#31488](https://github.com/mrdoob/three.js/pull/31488)
- r182: [NEW] `indirectOffset` for indirect drawing — [#32392](https://github.com/mrdoob/three.js/pull/32392)
- r185: [NEW] example "Add nanite-style rasterizer". The PR describes "a experimental, GPU-Driven Software Rasterizer implemented in TSL via compute shaders", inspired by Nanite, rendering "1+ billion triangles", and "still serving as a study". The example was later renamed `webgpu_compute_rasterizer` — [#33605](https://github.com/mrdoob/three.js/pull/33605), [r185 notes](https://github.com/mrdoob/three.js/releases/tag/r185)
- r187: [NEW] the drawing pipelines. The same PR deleted `webgpu_compute_rasterizer.html` and `_ibl.html` — [#35191](https://github.com/mrdoob/three.js/pull/35191)
- r187: [IMPROVED] WebGPURenderer merges consecutive `BatchedMesh` draws sharing a geometry range — [#35259](https://github.com/mrdoob/three.js/pull/35259). This is related, but not part of the meshlet pipeline as far as the title shows.

**How it works (per the PR and source)**
- Geometry is split into clusters of 64 triangles with simplified LODs, using meshoptimizer's clusterizer and simplifier — [#35191](https://github.com/mrdoob/three.js/pull/35191), [MeshletGeometryUtils.js at r187](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/utils/MeshletGeometryUtils.js)
- Culling runs per cluster on the GPU: frustum, normal cone, and two-phase occlusion with a depth pyramid. Occlusion "turns itself off when it doesn't pay off". Draws are indirect. LOD is chosen by screen-space error — [#35191](https://github.com/mrdoob/three.js/pull/35191)
- `VisibilityBufferDrawingPipeline` draws clusters into a visibility buffer, rasterizes small triangles in software in compute (`maxRasterSize` default 8 px), and sends large triangles and near-plane-crossing triangles to the hardware rasterizer. It stores the triangle in 16 bits, and geometries that do not fit use the hardware pipeline. Render calls with MRT, such as TRAA velocity, draw all clusters with the hardware pipeline — [VisibilityBufferDrawingPipeline.js at r187](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/pipelines/VisibilityBufferDrawingPipeline.js)
- The depth test in software raster uses two 32-bit atomics with `atomicMax`, with depth in the high bits — same source. Third-party Nanite ports cite the lack of 64-bit atomics in WebGPU as a blocker — [nanite-webgpu](https://github.com/Scthe/nanite-webgpu) (search snippet only, not fetched).

**Stated limits**
- PR text: unsupported objects and materials "fall back to the regular pipeline, as does the WebGL backend" — [#35191](https://github.com/mrdoob/three.js/pull/35191)
- Class doc: only "opaque and alpha-tested meshes using front side rendering" whose material nodes do not change geometry, coverage or output are supported. Shadow passes, and objects affected by clipping planes, use the regular pipeline. "the CPU still traverses the scene and uploads the transforms every frame" unless content sits in a static `BundleGroup` — [MeshletDrawingPipeline.js at r187](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/pipelines/MeshletDrawingPipeline.js)
- The PR text and class doc disagree on shadows and clipping. The PR lists "shadow maps, clipping" as supported, while the class doc routes them to the regular pipeline (a separate `shadowLodThreshold` parameter also exists) — [#35191](https://github.com/mrdoob/three.js/pull/35191) vs [source](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/pipelines/MeshletDrawingPipeline.js). I did not resolve this.
- The support check in source also requires `logarithmicDepthBuffer` and `reversedDepthBuffer` off, no local or global clipping planes, no `ArrayCamera`, and no cube or layered render target — [source](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/pipelines/MeshletDrawingPipeline.js)

**Author-reported performance (not independently verified)** — vsync off, 1080p, RTX 4070, median of 2 runs — [#35191](https://github.com/mrdoob/three.js/pull/35191)

| Scene | Standard pipeline | Meshlet pipeline |
|---|---|---|
| `gpudriven_instanced` | GPU 230.1 ms / CPU 231.0 ms | GPU 3.0 ms / CPU 3.0 ms |
| `gpudriven_city` | GPU 5.3 ms / CPU 5.2 ms | GPU 2.4 ms / CPU 1.5 ms |
| `lightprobes_sponza` | GPU 0.60 ms / CPU 3.2 ms | GPU 0.73 ms (slower) / CPU 1.0 ms |

- The PR says gains are largest in vertex-bound scenes and smaller in fragment-bound or already-optimized ones — [#35191](https://github.com/mrdoob/three.js/pull/35191)

**Forum**
- One user reports using it after updating to r187: "The trees and other objects in the scene except grass and tree leaves now use the new meshlet pipeline", with no meshlet-specific numbers — [forum, 2026-11-03](https://discourse.threejs.org/t/forest-environment-built-with-three-js-and-webgpu/96126/2)
- A forum search for `MeshletDrawingPipeline` returned nothing — [forum search](https://discourse.threejs.org/search.json?q=MeshletDrawingPipeline)

**Separate third-party work (not official; search snippets only, not evaluated)**
- `three-virtual-geometry` npm package — [README](https://cdn.jsdelivr.net/npm/three-virtual-geometry@0.1.3/README.md)
- Usnul's "Virtually Geometric" demo, stated as not open source — [forum thread](https://discourse.threejs.org/t/virtually-geometric/28420?page=4)
- `Scthe/nanite-webgpu`, a non-three.js reference — [GitHub](https://github.com/Scthe/nanite-webgpu)

### Inferences
- "Like Nanite" is fair for meshlets, GPU cluster culling, a visibility buffer and hybrid software raster. These are the features that make the comparison.
- It differs from Nanite as commonly described. The source describes "a chain of simplified levels of detail", selected "per object", which suggests no per-cluster LOD hierarchy. The words "stream", "DAG" and "METIS" had no hits in a case-insensitive search of the three files, which suggests no geometry streaming. This is from text search, not a full code read, and the project makes no Nanite-parity claim I could find.
- The WebGL 2 path gets no benefit, because the pipeline checks `backend.isWebGPUBackend` and otherwise uses the regular path.
- It has been public for only about a week, so there is almost no outside evidence of how well it holds up.

### Gaps
- I could not confirm live demos on threejs.org. The examples index is rendered by JavaScript, and WebFetch saw no list, so existence is verified only from `files.json` at the tag.
- `drawingPipeline` is not on the [WebGPURenderer docs page](https://threejs.org/docs/pages/WebGPURenderer.html) as fetched. The migration guide has no entry for it either.
- No independent benchmarks or Nanite comparison found.

## What are the most significant rendering changes per release?

### Takeaway
Work is heavily concentrated on WebGPURenderer, TSL and addon nodes. The largest changes are:

- **r181:** JSDoc docs, Inspector, SSGI, sync render APIs
- **r182:** modernized shadows
- **r183:** `RenderPipeline` rename, compatibility mode
- **r184:** TSL in WebGLRenderer, LightProbeGrid
- **r185:** WebXR on WebGPU, ClusteredLighting
- **r186:** VXGI, OIT, Gaussian splats, SunLight
- **r187:** meshlets, PMREM rewrite, LightMapper

### Cited Findings

**r187 (2026-10-30)** — [notes](https://github.com/mrdoob/three.js/releases/tag/r187), [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#186--187)
- WebGPURenderer and pipeline: [NEW] drawing pipelines [#35191](https://github.com/mrdoob/three.js/pull/35191); [NEW] `NodeMaterialInstance`, first named `ProxyNodeMaterial` [#34684](https://github.com/mrdoob/three.js/pull/34684), [#35269](https://github.com/mrdoob/three.js/pull/35269); [NEW] depth-only render targets [#34782](https://github.com/mrdoob/three.js/pull/34782); [NEW] `CompressedCubeTexture` support [#35083](https://github.com/mrdoob/three.js/pull/35083); [IMPROVED] parallel pipeline compile in `compileAsync()` [#34816](https://github.com/mrdoob/three.js/pull/34816); [IMPROVED] `ArrayCamera` culling per sub-camera [#35164](https://github.com/mrdoob/three.js/pull/35164)
- TSL and nodes: [NEW] implicit type promotion [#33695](https://github.com/mrdoob/three.js/pull/33695); [IMPROVED] build performance [#34815](https://github.com/mrdoob/three.js/pull/34815), [#34848](https://github.com/mrdoob/three.js/pull/34848); [IMPROVED] `three/tsl` tree-shakeable [#34533](https://github.com/mrdoob/three.js/pull/34533); [NEW] colour morphing in TSL [#35141](https://github.com/mrdoob/three.js/pull/35141); [NEW] `emissiveNode` on lambert, phong and toon [#35059](https://github.com/mrdoob/three.js/pull/35059). Struct storage buffers are now always arrays [#34807](https://github.com/mrdoob/three.js/pull/34807).
- Materials and PMREM: [NEW] EON rough diffuse BRDF [#34379](https://github.com/mrdoob/three.js/pull/34379); [IMPROVED] PMREM on cube render targets. The PR reports 37 to 82% lower mean RGB error against Blender Cycles and lower frame time on an M1 [#34585](https://github.com/mrdoob/three.js/pull/34585).
- Shadows: [IMPROVED] point-light seam fix [#35267](https://github.com/mrdoob/three.js/pull/35267); [IMPROVED] `TileShadowNode` reversed depth, layers and depth-only target [#34784](https://github.com/mrdoob/three.js/pull/34784), [#34783](https://github.com/mrdoob/three.js/pull/34783); [IMPROVED] PCF radius on non-square maps, both renderers [#35018](https://github.com/mrdoob/three.js/pull/35018)
- GI: [NEW] `LightMapper` progressive diffuse baker working with Light Probe and VXGI [#34991](https://github.com/mrdoob/three.js/pull/34991); [NEW] `unwrapUVs` real-time UV atlas generator [#34944](https://github.com/mrdoob/three.js/pull/34944); [IMPROVED] LightProbeGrid about 4x faster bakes with `ArrayCamera` [#34909](https://github.com/mrdoob/three.js/pull/34909); [IMPROVED] VXGI light-bleed mitigation [#35160](https://github.com/mrdoob/three.js/pull/35160)
- Post-processing: [IMPROVED] SSR refinements, GGX lobe, F0/F90 [#35243](https://github.com/mrdoob/three.js/pull/35243), [#35242](https://github.com/mrdoob/three.js/pull/35242); [NEW] lens dirt [#34715](https://github.com/mrdoob/three.js/pull/34715); [IMPROVED] SSGI resolution scale [#35229](https://github.com/mrdoob/three.js/pull/35229)
- Lights: [IMPROVED] `ClusteredLighting` gains spot lights and WebGL 2 fallback [#34978](https://github.com/mrdoob/three.js/pull/34978)
- Batching and instancing: [IMPROVED] `BatchedMesh` sort-corruption fix [#34828](https://github.com/mrdoob/three.js/pull/34828), shadow node compatibility [#34829](https://github.com/mrdoob/three.js/pull/34829), draw-range fix [#34983](https://github.com/mrdoob/three.js/pull/34983), memory leak fix [#35272](https://github.com/mrdoob/three.js/pull/35272); [IMPROVED] shared builds between instanced meshes [#35007](https://github.com/mrdoob/three.js/pull/35007)
- Tone mapping and colour: [IMPROVED] ACES Filmic fit denominator fix [#35110](https://github.com/mrdoob/three.js/pull/35110); [IMPROVED] colour conversion matrix order fix [#34952](https://github.com/mrdoob/three.js/pull/34952)
- Depth: [IMPROVED] logarithmic depth buffer is ignored when reversed depth is on [#35276](https://github.com/mrdoob/three.js/pull/35276)
- Loaders: [NEW] `JXLLoader` native HDR [#35015](https://github.com/mrdoob/three.js/pull/35015); [NEW] loader priority and HTTP cache controls [#34936](https://github.com/mrdoob/three.js/pull/34936); [IMPROVED] MaterialX additions [#34986](https://github.com/mrdoob/three.js/pull/34986); [IMPROVED] MikkTSpace moved from WASM to JavaScript [#34623](https://github.com/mrdoob/three.js/pull/34623)

**r186 (npm 2026-09-08)** — [notes](https://github.com/mrdoob/three.js/releases/tag/r186), [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#185--186)
- GI: [NEW] `VXGINode` voxel cone tracing for diffuse GI. It needs compute and storage buffers, so it is "not compatible with WebGL 2" [#34402](https://github.com/mrdoob/three.js/pull/34402). [IMPROVED] LightProbeGrid incremental baking [#34486](https://github.com/mrdoob/three.js/pull/34486).
- Shadows: [NEW] `SunLight` cascaded shadow maps, WebGLRenderer [#34221](https://github.com/mrdoob/three.js/pull/34221), WebGPURenderer [#34259](https://github.com/mrdoob/three.js/pull/34259), then moved to addons [#34326](https://github.com/mrdoob/three.js/pull/34326), cascades reduced to 2 ([commit](https://github.com/mrdoob/three.js/commit/52184ab39216913eb2984cbcffaf0efe1b677378))
- Transparency and splats: [NEW] `OITPassNode` weighted blended OIT [#34253](https://github.com/mrdoob/three.js/pull/34253); [NEW] Gaussian splat addon in TSL, working on WebGPU and the WebGL fallback [#33950](https://github.com/mrdoob/three.js/pull/33950), with frustum culling [#34254](https://github.com/mrdoob/three.js/pull/34254) and SPZ v4 [#34222](https://github.com/mrdoob/three.js/pull/34222)
- Post-processing: [NEW] `SSAONode` [#33921](https://github.com/mrdoob/three.js/pull/33921); [NEW] `softParticles()` [#33887](https://github.com/mrdoob/three.js/pull/33887); [IMPROVED] BloomNode faster blur [#33923](https://github.com/mrdoob/three.js/pull/33923); [IMPROVED] GTAO [#33895](https://github.com/mrdoob/three.js/pull/33895), [#33903](https://github.com/mrdoob/three.js/pull/33903)
- WebGPURenderer: [NEW] `DirectRenderPipeline` [#34166](https://github.com/mrdoob/three.js/pull/34166); [NEW] `compileComputeAsync()` [#32551](https://github.com/mrdoob/three.js/pull/32551); [IMPROVED] WebXR MSAA [#34120](https://github.com/mrdoob/three.js/pull/34120); [NEW] dynamic swapping of the lighting system [#34050](https://github.com/mrdoob/three.js/pull/34050); [IMPROVED] skinning beyond the UBO limit [#34200](https://github.com/mrdoob/three.js/pull/34200)
- Materials: [NEW] retroreflectivity [#33949](https://github.com/mrdoob/three.js/pull/33949); [IMPROVED] PMREM spiral blur on both renderers [#32367](https://github.com/mrdoob/three.js/pull/32367); [IMPROVED] energy conservation [#33983](https://github.com/mrdoob/three.js/pull/33983), [#33985](https://github.com/mrdoob/three.js/pull/33985)
- WebGLRenderer: [IMPROVED] node materials in `compile()` [#34232](https://github.com/mrdoob/three.js/pull/34232)
- Batching: [NEW] `batchIndirectIndex` [#34111](https://github.com/mrdoob/three.js/pull/34111); [IMPROVED] `positionPrevious` for batches [#34100](https://github.com/mrdoob/three.js/pull/34100); [IMPROVED] `BatchedMesh` draw offset fix [#34212](https://github.com/mrdoob/three.js/pull/34212)
- LOD-adjacent: [IMPROVED] `SimplifyModifier` rebuilt on meshoptimizer, now async [#34021](https://github.com/mrdoob/three.js/pull/34021)
- Loaders: [NEW] `GaussianSplatPLYLoader` [#34278](https://github.com/mrdoob/three.js/pull/34278); [IMPROVED] MaterialX upgrade [#33485](https://github.com/mrdoob/three.js/pull/33485); [IMPROVED] 3MF beam lattice [#34267](https://github.com/mrdoob/three.js/pull/34267); [IMPROVED] `EXT_mesh_gpu_instancing` attribute fix [#33927](https://github.com/mrdoob/three.js/pull/33927)
- Docs: [NEW] "TSL Guide" [#34085](https://github.com/mrdoob/three.js/pull/34085), [#34457](https://github.com/mrdoob/three.js/pull/34457)

**r185 (npm 2026-06-25)** — [notes](https://github.com/mrdoob/three.js/releases/tag/r185), [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#184--185)
- WebGPURenderer: [NEW] WebXR with WebGPU [#33583](https://github.com/mrdoob/three.js/pull/33583) and XR fallback [#33497](https://github.com/mrdoob/three.js/pull/33497); [NEW] rendering to texture arrays [#33507](https://github.com/mrdoob/three.js/pull/33507); [IMPROVED] full `ExternalTexture` support [#33816](https://github.com/mrdoob/three.js/pull/33816); [IMPROVED] premultiplied alpha changed [#33369](https://github.com/mrdoob/three.js/issues/33369)
- Lights: [NEW] `ClusteredLighting` Forward+, compute-built light lists, defaults 32 px tiles, 24 slices, 1024 lights [#33406](https://github.com/mrdoob/three.js/pull/33406)
- Instancing: [IMPROVED] full `InstancedMesh` support with render bundles [#33839](https://github.com/mrdoob/three.js/pull/33839)
- TSL: [NEW] `storageTexture3D` [#33443](https://github.com/mrdoob/three.js/pull/33443); [NEW] `textureGather` and `textureGatherCompare` [#33475](https://github.com/mrdoob/three.js/pull/33475); [NEW] override context [#33807](https://github.com/mrdoob/three.js/pull/33807); [NEW] `ambientOcclusion` property [#33728](https://github.com/mrdoob/three.js/pull/33728)
- Shadows: [IMPROVED] `textureGatherCompare` in PCF soft filter [#33534](https://github.com/mrdoob/three.js/pull/33534); [IMPROVED] point-shadow log-depth fix [#33584](https://github.com/mrdoob/three.js/pull/33584)
- Post-processing and GI: [IMPROVED] GTAO cosine-weighted, more physically correct, darker and wider [#33654](https://github.com/mrdoob/three.js/pull/33654); [IMPROVED] SSR spatiotemporal denoiser [#33843](https://github.com/mrdoob/three.js/pull/33843); [IMPROVED] SSGI RG11B10 [#33822](https://github.com/mrdoob/three.js/pull/33822); [IMPROVED] LightProbeGrid indirect bounces [#33657](https://github.com/mrdoob/three.js/pull/33657)
- Loaders: [IMPROVED] PLYLoader honours data types [#33622](https://github.com/mrdoob/three.js/pull/33622); [IMPROVED] USDZExporter animation and multi-material [#33589](https://github.com/mrdoob/three.js/pull/33589), [#33598](https://github.com/mrdoob/three.js/pull/33598)

**r184 (2026-04-16)** — [notes](https://github.com/mrdoob/three.js/releases/tag/r184), [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#183--184)
- WebGLRenderer: [NEW] NodeMaterial compatibility layer via `renderer.setNodesHandler( new WebGLNodesHandler() )`, to "ease users into WebGPURenderer" [#32851](https://github.com/mrdoob/three.js/pull/32851); [IMPROVED] working colour space for render targets [#33036](https://github.com/mrdoob/three.js/pull/33036); [NEW] packed normal maps [#33055](https://github.com/mrdoob/three.js/pull/33055); [IMPROVED] normal bias fallback [#33391](https://github.com/mrdoob/three.js/pull/33391)
- GI: [NEW] `LightProbeGrid` position-dependent diffuse GI for WebGLRenderer, with GPU-resident baking [#33125](https://github.com/mrdoob/three.js/pull/33125)
- WebGPURenderer: [IMPROVED] non-blocking `compileAsync()` [#32984](https://github.com/mrdoob/three.js/pull/32984); [NEW] `WEBGL_multi_draw` fallback [#33238](https://github.com/mrdoob/three.js/pull/33238); [NEW] `ReadbackBuffer` and partial readback [#33300](https://github.com/mrdoob/three.js/pull/33300), [#33322](https://github.com/mrdoob/three.js/pull/33322); [NEW] `HTMLTexture` [#31233](https://github.com/mrdoob/three.js/pull/31233)
- Lights: [NEW] `DynamicLighting`, which skips full material rebuilds on light changes but does not cover shadows [#33042](https://github.com/mrdoob/three.js/pull/33042)
- TSL: [IMPROVED] compile performance 3.0x per title [#33120](https://github.com/mrdoob/three.js/pull/33120); [NEW] `global` and `local` scope [#33302](https://github.com/mrdoob/three.js/pull/33302); [NEW] `OnFrameUpdate` [#33356](https://github.com/mrdoob/three.js/pull/33356)
- Post-processing: [NEW] `TAAUNode` [#33368](https://github.com/mrdoob/three.js/pull/33368); [NEW] `FSR1Node` [#33339](https://github.com/mrdoob/three.js/pull/33339)
- Tools: [NEW] DevTools [#30870](https://github.com/mrdoob/three.js/pull/30870); [NEW] Inspector Timeline and Memory tabs [#33078](https://github.com/mrdoob/three.js/pull/33078), [#33097](https://github.com/mrdoob/three.js/pull/33097)
- Loaders: [IMPROVED] EXRLoader multi-part, deep scanline, YCbCr, B44/A [#33218](https://github.com/mrdoob/three.js/pull/33218), [#33212](https://github.com/mrdoob/three.js/pull/33212); [IMPROVED] FBXLoader auto Z-up conversion [#33332](https://github.com/mrdoob/three.js/pull/33332); [IMPROVED] KTX2Loader RGBA16 unorm [#33245](https://github.com/mrdoob/three.js/pull/33245); [IMPROVED] USDLoader primitives and lights [#33082](https://github.com/mrdoob/three.js/pull/33082)

**r183 (npm 2026-02-18)** — [notes](https://github.com/mrdoob/three.js/releases/tag/r183), [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#182--183)
- WebGPURenderer: [IMPROVED] always requests compatibility mode and upgrades to core [#32762](https://github.com/mrdoob/three.js/pull/32762), with antialiasing off in compat mode [#32902](https://github.com/mrdoob/three.js/pull/32902); [NEW] basic reversed depth buffer [#32967](https://github.com/mrdoob/three.js/pull/32967); [NEW] per-attachment MRT blending [#32636](https://github.com/mrdoob/three.js/pull/32636); [NEW] `CubeRenderTarget` export [#32871](https://github.com/mrdoob/three.js/pull/32871)
- Shadows: [IMPROVED] shadow side, `LinearFilter`, `LessEqual` compare and precision aligned with WebGLRenderer; the PR calls the wrong shadow side "the main reasons for the extreme shadow acne" [#32705](https://github.com/mrdoob/three.js/pull/32705); [NEW] `LightShadow.biasNode` [#32811](https://github.com/mrdoob/three.js/pull/32811); [NEW] `shadowMap.transmitted` (coloured shadows) [#32596](https://github.com/mrdoob/three.js/pull/32596), [#32611](https://github.com/mrdoob/three.js/pull/32611); [NEW] `maskShadowNode` [#32598](https://github.com/mrdoob/three.js/pull/32598)
- Post-processing: [DEPRECATED: rename] `PostProcessing` to `RenderPipeline` [#32789](https://github.com/mrdoob/three.js/pull/32789); [NEW] `GodraysNode` [#32888](https://github.com/mrdoob/three.js/pull/32888); [NEW] `retroPass` [#32930](https://github.com/mrdoob/three.js/pull/32930); [IMPROVED] SSR roughness blur [#32834](https://github.com/mrdoob/three.js/pull/32834)
- Materials: [NEW] `scene.environment` IBL on Lambert and Phong [#32791](https://github.com/mrdoob/three.js/pull/32791), [#32795](https://github.com/mrdoob/three.js/pull/32795); [NEW] clearcoat with rect area lights [#32809](https://github.com/mrdoob/three.js/pull/32809); [NEW] `Object3D.pivot` [#32745](https://github.com/mrdoob/three.js/pull/32745)
- Batching: [NEW] `BatchedMesh` per-instance opacity [#32725](https://github.com/mrdoob/three.js/pull/32725) and wireframe [#32948](https://github.com/mrdoob/three.js/pull/32948)
- Loaders: [NEW] `KHR_meshopt_compression` [#32163](https://github.com/mrdoob/three.js/pull/32163); [NEW] USDLoader USDC and animation [#32704](https://github.com/mrdoob/three.js/pull/32704), [#32740](https://github.com/mrdoob/three.js/pull/32740); [NEW] UltraHDR ISO 21496-1 gainmap metadata [#32862](https://github.com/mrdoob/three.js/pull/32862)
- Docs: [NEW] TSL spec `TSL.md` [#32601](https://github.com/mrdoob/three.js/pull/32601)

**r182 (2025-12-10)** — [notes](https://github.com/mrdoob/three.js/releases/tag/r182), [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#181--182)
- Shadows: [IMPROVED] "Modernized shadow mapping" in WebGLRenderer [#32181](https://github.com/mrdoob/three.js/pull/32181); [IMPROVED] WebGPU PCF with Vogel disk sampling and IGN [#32381](https://github.com/mrdoob/three.js/pull/32381); [IMPROVED] point lights use native cube depth texture [#32379](https://github.com/mrdoob/three.js/pull/32379); [IMPROVED] VSM fixes [#32327](https://github.com/mrdoob/three.js/pull/32327)
- WebGLRenderer: [NEW] `outputBufferType` internal HDR and `setEffects()` post-processing without EffectComposer [#32461](https://github.com/mrdoob/three.js/pull/32461)
- Materials: [IMPROVED] GGX VNDF accuracy, matching Blender roughness [#32311](https://github.com/mrdoob/three.js/pull/32311); [IMPROVED] sheen and metalness energy conservation [#32356](https://github.com/mrdoob/three.js/pull/32356), [#32190](https://github.com/mrdoob/three.js/pull/32190)
- Instancing: [NEW] `InstanceNode` with `StorageInstancedBufferAttribute` [#32240](https://github.com/mrdoob/three.js/pull/32240); [NEW] `indirectOffset` [#32392](https://github.com/mrdoob/three.js/pull/32392)
- TSL: [NEW] float pack/unpack intrinsics [#32329](https://github.com/mrdoob/three.js/pull/32329); [NEW] `builtinShadowContext()` and `builtinAOContext()` [#32384](https://github.com/mrdoob/three.js/pull/32384)
- Post-processing: [IMPROVED] TRAA Halton sequence, less smearing [#32319](https://github.com/mrdoob/three.js/pull/32319), [#32322](https://github.com/mrdoob/three.js/pull/32322)
- Docs: [NEW] WebGPURenderer manual guide [#32260](https://github.com/mrdoob/three.js/pull/32260) and post-processing guide [#32279](https://github.com/mrdoob/three.js/pull/32279)
- Loaders: [IMPROVED] VOXLoader scene graph and greedy meshing [#32488](https://github.com/mrdoob/three.js/pull/32488), [#32489](https://github.com/mrdoob/three.js/pull/32489); [IMPROVED] UltraHDRLoader parse performance [#32468](https://github.com/mrdoob/three.js/pull/32468)

**r181 (npm 2025-10-31)** — [notes](https://github.com/mrdoob/three.js/releases/tag/r181), [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#180--181)
- GI and post-processing: [NEW] `SSGINode` [#31839](https://github.com/mrdoob/three.js/pull/31839); [NEW] `SSSNode` screen-space shadows [#32050](https://github.com/mrdoob/three.js/pull/32050); [IMPROVED] GTAO temporal filtering [#32076](https://github.com/mrdoob/three.js/pull/32076); [IMPROVED] TRAA adaptive blend [#32139](https://github.com/mrdoob/three.js/pull/32139)
- WebGPURenderer: [NEW] `Inspector` [#31869](https://github.com/mrdoob/three.js/pull/31869); [NEW] `CanvasTarget` [#31919](https://github.com/mrdoob/three.js/pull/31919); [NEW] `dispatchWorkgroupsIndirect` [#31488](https://github.com/mrdoob/three.js/pull/31488); [IMPROVED] samplers decoupled from textures [#31899](https://github.com/mrdoob/three.js/pull/31899); [IMPROVED] render bundles with MRT [#31906](https://github.com/mrdoob/three.js/pull/31906); [IMPROVED] dynamic shadow map type switching [#32105](https://github.com/mrdoob/three.js/pull/32105)
- Materials: [IMPROVED] DFG LUT replaces analytical approximation [#32054](https://github.com/mrdoob/three.js/pull/32054); [IMPROVED] multi-scattering compensation [#32072](https://github.com/mrdoob/three.js/pull/32072), [#32089](https://github.com/mrdoob/three.js/pull/32089); [IMPROVED] PMREM GGX VNDF importance sampling [#32114](https://github.com/mrdoob/three.js/pull/32114)
- Tone mapping: [IMPROVED] WebGPU tone mapping mode detection fix [#31957](https://github.com/mrdoob/three.js/pull/31957)
- Docs: [NEW] JSDoc-generated API docs, English only — [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#180--181)

**r180 (2025-09-03), context** — [notes](https://github.com/mrdoob/three.js/releases/tag/r180)
- [NEW] WebGPU HDR output [#29573](https://github.com/mrdoob/three.js/pull/29573); [IMPROVED] `DepthOfFieldNode` new implementation [#31547](https://github.com/mrdoob/three.js/pull/31547); [IMPROVED] SSR roughness and quality [#31649](https://github.com/mrdoob/three.js/pull/31649), [#31576](https://github.com/mrdoob/three.js/pull/31576); [NEW] `SubgroupFunctionNode` [#31378](https://github.com/mrdoob/three.js/pull/31378); [IMPROVED] KTX2Loader BC4, BC5, RGB9E5, R11G11B10, PVRTC1 [#31713](https://github.com/mrdoob/three.js/pull/31713), [#31611](https://github.com/mrdoob/three.js/pull/31611), [#31613](https://github.com/mrdoob/three.js/pull/31613), [#31771](https://github.com/mrdoob/three.js/pull/31771)

### Inferences
- Tone mapping saw only fixes and plumbing in this window, plus [#33187](https://github.com/mrdoob/three.js/pull/33187) (RenderPipeline sync with `renderer.toneMapping`). I found no new tone-mapping operator in release-note titles.
- Colour management likewise saw fixes, not new models: [#33036](https://github.com/mrdoob/three.js/pull/33036), [#34952](https://github.com/mrdoob/three.js/pull/34952).
- The classic `LOD` class had no notable title-level change. LOD work went into meshlets and `SimplifyModifier`.

### Gaps
- I read PR text only for selected items, so titled "fixes" may hide larger behaviour changes.
- The release notes have no prose summary, so "most significant" is my judgement from titles and the migration guide.

## Which features were newly added versus ported from WebGLRenderer to WebGPURenderer?

### Takeaway
The large new techniques are mostly WebGPURenderer-first or WebGPU-only. A smaller set went WebGL first and was then ported. There is also a reverse flow, with TSL, node materials and post-processing conveniences moving into WebGLRenderer as a migration aid.

### Cited Findings
**WebGL first, then ported to WebGPU**
- `LightProbeGrid`: WebGLRenderer in r184 [#33125](https://github.com/mrdoob/three.js/pull/33125), WebGPU version in r186 [#33913](https://github.com/mrdoob/three.js/pull/33913), with the WebGL class renamed `LightProbeGridWebGL` — [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#185--186)
- `SunLight`: WebGL [#34221](https://github.com/mrdoob/three.js/pull/34221), WebGPU [#34259](https://github.com/mrdoob/three.js/pull/34259), both in r186
- Multi-scattering and rough IBL: WebGL [#32072](https://github.com/mrdoob/three.js/pull/32072), [#32102](https://github.com/mrdoob/three.js/pull/32102); WebGPU [#32089](https://github.com/mrdoob/three.js/pull/32089), [#32108](https://github.com/mrdoob/three.js/pull/32108), all in r181
- Parity work rather than features: WebXR on WebGPU in r185 [#33583](https://github.com/mrdoob/three.js/pull/33583); WebGPU reversed depth in r183, titled "basic" [#32967](https://github.com/mrdoob/three.js/pull/32967); WebGPU shadow state aligned to WebGLRenderer [#32705](https://github.com/mrdoob/three.js/pull/32705)

**New, WebGPURenderer only or first**
- Meshlet and visibility buffer pipelines [#35191](https://github.com/mrdoob/three.js/pull/35191); VXGI [#34402](https://github.com/mrdoob/three.js/pull/34402)
- The manual says "We have also added new effects like SSGI, SSS or a better DoF exclusively for the new renderer" — [manual](https://threejs.org/manual/pages/webgpurenderer.html)
- OIT [#34253](https://github.com/mrdoob/three.js/pull/34253), whose PR says the renderer "is now in a state where it provides all features" needed; ClusteredLighting [#33406](https://github.com/mrdoob/three.js/pull/33406); DynamicLighting [#33042](https://github.com/mrdoob/three.js/pull/33042); Godrays [#32888](https://github.com/mrdoob/three.js/pull/32888); TAAU [#33368](https://github.com/mrdoob/three.js/pull/33368); LightMapper "for `WebGPURenderer`" [#34991](https://github.com/mrdoob/three.js/pull/34991)
- Gaussian splats are TSL-based and work on WebGPU and the WebGL fallback [#33950](https://github.com/mrdoob/three.js/pull/33950)

**Reverse flow, into WebGLRenderer**
- NodeMaterial and TSL through `WebGLNodesHandler` [#32851](https://github.com/mrdoob/three.js/pull/32851), extended by [#34232](https://github.com/mrdoob/three.js/pull/34232), [#34628](https://github.com/mrdoob/three.js/pull/34628)
- `outputBufferType` and `setEffects()` [#32461](https://github.com/mrdoob/three.js/pull/32461)
- `FrustumArray` support [#34275](https://github.com/mrdoob/three.js/pull/34275); shadow camera layers "Align ... to `WebGPURenderer`" [#34670](https://github.com/mrdoob/three.js/pull/34670)
- `UnrealBloomPass` "Backported blur improvements" ([commit](https://github.com/mrdoob/three.js/commit/057d341afde09e38c59037e39fc81198b241258d))

**Unclear direction**
- `FSR1Node` is titled "Add FSR 1 port for `WebGPURenderer`" [#33339](https://github.com/mrdoob/three.js/pull/33339). I did not read the PR, and it may be a port of FSR rather than of a WebGL feature.

### Inferences
- The "ported" group looks small compared with the "new" group, and most of what WebGLRenderer gained serves migration. This count is from titles and is not exhaustive.

### Gaps
- There is no official list of port versus new. This classification is mine, from titles and selected PRs.

## Is WebGPURenderer the recommended default? What is its WebGL 2 fallback, and what does the fallback lose?

### Takeaway
**The primary sources I read do not make it the default or recommend it universally.** The API docs call it "the new alternative", the manual says it is "still in an experimental state", and `WebGLRenderer` remains "the recommended choice for pure WebGL 2 applications", though project focus is on WebGPURenderer. The fallback is automatic and is a real WebGL 2 backend, but it has no storage buffers or compute shaders, so compute-dependent features are lost or take a reduced path.

### Cited Findings
**Status**
- "This renderer is the new alternative of `WebGLRenderer`." and "falls backs to a WebGL 2 backend" — [API docs](https://threejs.org/docs/pages/WebGPURenderer.html)
- "The renderer itself is still in an experimental state although its maturity level has been greatly improved in the last years." It also warns that users may "encounter missing features or a better performance with `WebGLRenderer`" — [manual](https://threejs.org/manual/pages/webgpurenderer.html)
- `WebGLRenderer` is "still maintained and the recommended choice for pure WebGL 2 applications", with "no plans to add larger new features" since "the project's focus is now on `WebGPURenderer`" — [manual](https://threejs.org/manual/pages/webgpurenderer.html)
- Conflict: a third-party blog says r182 made WebGPURenderer the recommended renderer — [youngju.dev](https://www.youngju.dev/blog/culture/2026-05-16-webgpu-webgl-2026-three-js-r3f-babylon-playcanvas-tres-needle-native-webgpu-wgsl-deep-dive) (search summary only). I found no primary source supporting it. The r182 items that exist are the manual guides [#32260](https://github.com/mrdoob/three.js/pull/32260) and [#32279](https://github.com/mrdoob/three.js/pull/32279).

**Fallback mechanics**
- Automatic, or forced with `forceWebGL: true` — [manual](https://threejs.org/manual/pages/webgpurenderer.html). Init is async; use `setAnimationLoop()` or `await renderer.init()` — same page. Sync methods replaced the async ones in r181 — [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#180--181)
- TSL transpiles to WGSL or GLSL depending on the backend — [manual](https://threejs.org/manual/pages/webgpurenderer.html)
- WebGPU compatibility mode is a separate tier. The renderer requests compat and upgrades to core [#32762](https://github.com/mrdoob/three.js/pull/32762)

**Lost or reduced on the WebGL 2 backend, at r187**
- `GLSLNodeBuilder` declares `storageBuffer: false`, and logs errors for subgroup operations, `subgroupSize`, `subgroupIndex`, 16-bit floats and dual source blending — [GLSLNodeBuilder.js at r187](https://github.com/mrdoob/three.js/blob/r187/src/renderers/webgl-fallback/nodes/GLSLNodeBuilder.js)
- `WebGLBackend.compute()` runs through transform feedback. It warns that `count` "must be a single number", not an array or `IndirectStorageBufferAttribute` — [WebGLBackend.js at r187](https://github.com/mrdoob/three.js/blob/r187/src/renderers/webgl-fallback/WebGLBackend.js)
- Reversed depth needs `EXT_clip_control`, otherwise it falls back to the default depth buffer. Shared depth textures with MSAA need `WEBGL_multisampled_render_to_texture`, otherwise it falls back to single-sample — same file
- Meshlet and visibility buffer pipelines use the regular path [#35191](https://github.com/mrdoob/three.js/pull/35191). VXGI is "not compatible with WebGL 2" [#34402](https://github.com/mrdoob/three.js/pull/34402)
- Available on the fallback: Gaussian splats [#33950](https://github.com/mrdoob/three.js/pull/33950), ClusteredLighting since r187 [#34978](https://github.com/mrdoob/three.js/pull/34978), and an emulated multi-draw path [#33238](https://github.com/mrdoob/three.js/pull/33238)

**Lost compared with WebGLRenderer, on either backend**
- `ShaderMaterial`, `RawShaderMaterial`, `onBeforeCompile()` and `EffectComposer` are "not supported" — [manual](https://threejs.org/manual/pages/webgpurenderer.html)

**Performance reports**
- PMREM rewrite: reported improvement of about 25% in WebGL, about 16% in WebGPU, about 3% in the fallback, Apple M1 [#34585](https://github.com/mrdoob/three.js/pull/34585)
- A user reports on r183 with 4,000 meshes: WebGLRenderer 4.5 ms CPU and 32 ms first frame, WebGPU backend 9.7 ms and 167 ms, `forceWebGL` 10.3 ms and 344 ms. The thread links this to issue #30560. No maintainer replied in the thread, and it predates later changes such as [#34816](https://github.com/mrdoob/three.js/pull/34816) — [forum, 2026-05-29](https://discourse.threejs.org/t/webgpurenderer-2x-slower-cpu-and-5-10x-slower-first-frame-than-webglrenderer-on-many-mesh-scenes-r183-same-on-both-backends/91904)

### Inferences
- The manual line "we are currently investigating ... limited node material support to `WebGLRenderer`" looks stale, because [#32851](https://github.com/mrdoob/three.js/pull/32851) shipped that in r184. So the "experimental" wording may also lag behind the project's actual view.
- In practice, the project's direction clearly favours WebGPURenderer even though its docs do not call it the default.

### Gaps
- The threejs.org manual does not show its version, so I cannot confirm that it matches r187. The path `manual/en/webgpurenderer.html` returned 404 at the `r187` tag and on the live site at `/manual/en/`.
- No official feature matrix for fallback losses was found. The list above is from source text search and PRs, and is likely incomplete.
- Whether WebGPURenderer is planned to become the sole renderer, or by when, is not stated in anything I read.

## What did the project remove or deprecate, and why?

### Takeaway
Reasons are often short or absent. The recurring themes are replacement by a better technique, one maintained implementation instead of two, API correctness, and dropping legacy distribution. The sharpest evidence of what did not work is DualKawaseBloomNode (added and removed in one release), TiledLighting, the PMREM CubeUV atlas and `waitForGPU()`.

### Cited Findings
**With a stated reason**
- [REMOVED] r187 `CubeUVReflectionMapping`: the atlas is replaced by cube targets, with 37 to 82% lower error, reduced near-mirror aliasing, corrected excess brightness at high roughness, and about 33% less GPU texture memory for prefiltered KTX2 environments. Generation was slower for the tested 1K HDRI and faster for the 2K — [#34585](https://github.com/mrdoob/three.js/pull/34585)
- [REMOVED] r186 `DualKawaseBloomNode`, added and removed in the same release ([#33920](https://github.com/mrdoob/three.js/pull/33920), [#33979](https://github.com/mrdoob/three.js/pull/33979)). In [#33920 comments](https://github.com/mrdoob/three.js/pull/33920), mrdoob says it "has some artefacts" and cannot match Gaussian bloom, and Mugen87 objects to two bloom classes on maintenance grounds. BloomNode then gained a faster blur [#33923](https://github.com/mrdoob/three.js/pull/33923).
- [REMOVED] r185 `TiledLighting`: "so we can focus our efforts on improvements to `ClusteredLighting`" — [#33751](https://github.com/mrdoob/three.js/pull/33751)
- [REMOVED] r186 `PCFSoftShadowMap` with WebGPURenderer ([#33987](https://github.com/mrdoob/three.js/pull/33987)); deprecated in r182 for WebGLRenderer. `PCFShadowMap` "is now soft as well" — [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#185--186). In source the old constant warns and falls back.
- [REMOVED] r181 `waitForGPU()`: it was documented as synchronizing CPU and GPU, but "`onSubmittedWorkDone` is not for sychronization, it is for throttling", and a fix from calling it is "just hiding the actual bug" — [issue #32012](https://github.com/mrdoob/three.js/issues/32012), [#32024](https://github.com/mrdoob/three.js/pull/32024)
- [REMOVED] r180 `RGBMLoader`: "UltraHDR seems like much better option" — [#31566](https://github.com/mrdoob/three.js/pull/31566)
- [DEPRECATED] r186 CommonJS build: ESM has been usable in Node since 2019, and `three.cjs` becomes a warning shim that "will be removed" — [#33891](https://github.com/mrdoob/three.js/pull/33891)
- [REMOVED] r186 minified builds: `*.min.js` are gone. jsDelivr minifies on its own, but "unpkg and cdnjs do not, so direct `.min.js` links there will 404" — [#33893](https://github.com/mrdoob/three.js/pull/33893)
- [DEPRECATED] r181 `renderAsync()`, `computeAsync()`, `clearAsync()` and others: use sync calls after init — [guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide#180--181), [#32022](https://github.com/mrdoob/three.js/pull/32022)
- [REMOVED] r187 standalone `webgpu_compute_rasterizer` examples, deleted when the drawing pipelines landed — [#35191](https://github.com/mrdoob/three.js/pull/35191)
- Scope change, r186: `SunLight` moved from core to addons after sunag asked if it should be an addon. mrdoob preferred built-in but tried the addon refactor — [#34259 comments](https://github.com/mrdoob/three.js/pull/34259), [#34326](https://github.com/mrdoob/three.js/pull/34326)

**Reason thin or only a replacement named** (all from the [Migration Guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide))
- r187: [DEPRECATED] `iridescenceThicknessRange` ([#35039](https://github.com/mrdoob/three.js/pull/35039)); [REMOVED] Node support for the `pixelationPass()` `pixelSize` ([#34543](https://github.com/mrdoob/three.js/pull/34543)); [DEPRECATED: rename] `renderer.library.addLight` to `CustomLight.registerNode` ([#34622](https://github.com/mrdoob/three.js/pull/34622)); [REMOVED] GTAOPass combined depth/normal texture ([#34836](https://github.com/mrdoob/three.js/pull/34836)); [REMOVED] code deprecated in r177 ([#34517](https://github.com/mrdoob/three.js/pull/34517))
- r186: [DEPRECATED] GTAO `distanceExponent` and `distanceFallOff`, no effect after the distance model improved ([#33895](https://github.com/mrdoob/three.js/pull/33895)); [REMOVED] `Sky` and `SkyMesh` `up` uniform ([#34354](https://github.com/mrdoob/three.js/pull/34354)); [DEPRECATED: rename] `Source` to `TextureSource` ([#34202](https://github.com/mrdoob/three.js/pull/34202)), `LightProbeGrid` to `LightProbeGridWebGL` ([#33911](https://github.com/mrdoob/three.js/pull/33911))
- r185: [REMOVED] `AnamorphicNode`, use `BloomNode` ([#33600](https://github.com/mrdoob/three.js/pull/33600), which shows bloom mips producing the effect); [REMOVED] `SSAAPassNode` clear properties ([#33478](https://github.com/mrdoob/three.js/pull/33478)); [DEPRECATED] `LWOLoader` ([#33621](https://github.com/mrdoob/three.js/pull/33621)), `DRACOLoader.setDecoderConfig()` as it will always use WASM ([#33602](https://github.com/mrdoob/three.js/pull/33602)), `SVGLoader.createShapes()` ([#33503](https://github.com/mrdoob/three.js/pull/33503)), `Matrix3` transforms ([#33757](https://github.com/mrdoob/three.js/pull/33757)), `inverseTransformDirection()` ([#33351](https://github.com/mrdoob/three.js/pull/33351)); [REMOVED] bundled DRACO encoders and Lottie/TTF libs in favour of CDN ([#33789](https://github.com/mrdoob/three.js/pull/33789), [#33798](https://github.com/mrdoob/three.js/pull/33798))
- r184: [DEPRECATED] `VTKLoader` ([#33363](https://github.com/mrdoob/three.js/pull/33363)); [REMOVED] "deprecated instancing render paths", listed under BatchedMesh ([#33234](https://github.com/mrdoob/three.js/pull/33234)); [REMOVED] return values of `FileLoader.load()` and `ImageBitmapLoader.load()` ([#33380](https://github.com/mrdoob/three.js/pull/33380))
- r183: [DEPRECATED] `Clock`, use `Timer` ([#32798](https://github.com/mrdoob/three.js/pull/32798)); [REMOVED] `MeshPostProcessingMaterial`; [REMOVED] legacy `Sky` gamma correction ([#32677](https://github.com/mrdoob/three.js/pull/32677)); [REMOVED] `WebGLCubeRenderTarget` with WebGPURenderer ([#32871](https://github.com/mrdoob/three.js/pull/32871)); [REMOVED] TSL `scriptable` node ([#32925](https://github.com/mrdoob/three.js/pull/32925)); [DEPRECATED: rename] `PostProcessing` to `RenderPipeline` ([#32789](https://github.com/mrdoob/three.js/pull/32789))
- r182: [DEPRECATED: rename] `colorBufferType` to `outputBufferType` ([#32501](https://github.com/mrdoob/three.js/pull/32501))
- r181: [REMOVED] WebGL 1.0 compatibility code ([#32071](https://github.com/mrdoob/three.js/pull/32071)); [REMOVED] legacy docs and i18n of API docs ([#32117](https://github.com/mrdoob/three.js/pull/32117)); [DEPRECATED] `KTX2Loader.detectSupportAsync()`; [DEPRECATED: rename] `PI2` to `TWO_PI`
- r180: [DEPRECATED: rename] `RGBELoader` to `HDRLoader` ([#31604](https://github.com/mrdoob/three.js/pull/31604)), `resolution` to scalar `resolutionScale` on several nodes

**Policy**
- "deprecation warnings last for 10 releases" — [Migration Guide](https://github.com/mrdoob/three.js/wiki/Migration-Guide)

### Inferences
- `DualKawaseBloomNode` and `TiledLighting` show a preference for one configurable implementation per technique over parallel variants. This is my reading of two cases, not a stated policy.
- `GTAONode`, PMREM and the shadow rework each changed visual output between releases, per the guide. So appearance is not treated as stable across versions.

### Gaps
- The removals of `AnamorphicNode`, `MeshPostProcessingMaterial`, `LWOLoader` and `VTKLoader` have no reason in the text I read. I did not read the `LWOLoader` or `VTKLoader` PRs.
- I did not read the discussion in [#33987](https://github.com/mrdoob/three.js/pull/33987)'s linked commit comment, or the longer reasoning for the original PCF rework beyond the guide's note.
