# Open-source libraries for Nanite-like virtualized geometry, and what moonBASIC could reuse

As of 5 November 2026. Engine facts were read from branch `moonbasic-baseline`, commit `121c84d`. Every third-party library below was read, not compiled or run: no number here was measured by me. "Estimate" marks my own arithmetic. Licences were read from the licence file unless the entry says otherwise.

## What can actually be reused: licence, language, maintenance, stage, GPU needs, and is the offline part usable alone?

### Takeaway
Only one thing is directly reusable and worth taking: **meshoptimizer** with its `clusterlod.h` (MIT, C API, offline only, no GPU needs). It removes the need for METIS. Every runtime in the list needs compute, mesh shaders, or 64-bit atomics, so none runs on GL 3.3; their value is as method references. There is no Go binding, so moonBASIC would write a small cgo wrapper.

### Cited Findings

Comparison (dates are last push from the GitHub API on 5 November 2026):

| Project | Licence (how checked) | Language | Maintenance | Stage covered | Runtime GPU needs | Offline part alone? |
| --- | --- | --- | --- | --- | --- | --- |
| [meshoptimizer](https://github.com/zeux/meshoptimizer) | MIT ([LICENSE.md](https://github.com/zeux/meshoptimizer/blob/master/LICENSE.md)) | C++ with C API | Active; 2026-10-14 HEAD; release v1.3 | Offline: meshlets, simplify, partition | None | Yes, that is all it is |
| [clusterlod.h](https://github.com/zeux/meshoptimizer/blob/master/demo/clusterlod.h) | MIT (file header) | C++ header, in `demo/` | Same repo | Offline: full cluster DAG and BVH, selection rule documented | None | Yes |
| [METIS](https://github.com/KarypisLab/METIS) | Apache-2.0 ([LICENSE](https://github.com/KarypisLab/METIS/blob/master/LICENSE)); [GKlib](https://github.com/KarypisLab/GKlib) notice says University of Minnesota, terms not read in full | C | 2026-06 | Offline: graph partitioning | None | Yes; not needed with `meshopt_partitionClusters` |
| [KaHIP](https://github.com/KaHIP/KaHIP), [mt-KaHyPar](https://github.com/kahypar/mt-kahypar) | MIT (GitHub API only) | C++ | 2026-09, 2026-11 | Offline: partitioning | None | Yes; not needed |
| [nv_cluster_lod_builder](https://github.com/nvpro-samples/nv_cluster_lod_builder) | Apache-2.0 (LICENSE) | C++ | **Archived** | Offline: cluster DAG | None | Yes, but abandoned |
| [vk_lod_clusters](https://github.com/nvpro-samples/vk_lod_clusters) | Apache-2.0 (GitHub API) | C++, Vulkan | 2026-10 | Full sample: build, streaming, traversal, raster, ray tracing | `VK_EXT/NV_mesh_shader`, compute | Build uses meshoptimizer; reference only |
| [Bevy meshlet](https://github.com/bevyengine/bevy/tree/main/crates/bevy_pbr/src/meshlet) | MIT or Apache-2.0 (both files present) | Rust, wgpu | Active; experimental | Full: build, compute culling, raster, visibility buffer | `TEXTURE_INT64_ATOMIC`, Vulkan or Metal only | `from_mesh.rs` (1,099 lines), Rust, meshopt plus METIS |
| [Carrot](https://github.com/jglrxavpok/Carrot) | MIT (LICENSE) | C++, Vulkan | 2026-09 HEAD; "Don't expect support" | Hobby engine; asset tool plus Vulkan renderer | Vulkan; mesh shader pipeline code present | `asset_tools/fertilizer`, tied to its engine; bundles METIS |
| [three-virtual-geometry](https://github.com/elad12390/three-virtual-geometry) | MIT (LICENSE, 2026 Elad Ben-Haim) | TypeScript | v0.1.3, 2 stars | Full, for three.js | WebGPU compute | No separate tool seen |
| [nanite-webgpu](https://github.com/Scthe/nanite-webgpu) | MIT (GitHub API only) | TypeScript | 2026-05 | Full, including software raster and impostors | WebGPU compute | Runs meshoptimizer and METIS in WASM in the browser |
| [three-nanite](https://github.com/AIFanatic/three-nanite) | MIT (GitHub API only) | JavaScript | Stale, 2024-09 | Experiment | Not checked | Not checked |
| [Nexus](https://github.com/cnr-isti-vclab/nexus) | C++ **GPL**, JavaScript MIT ([README](https://github.com/cnr-isti-vclab/nexus/blob/master/README.md)); API reports GPL-3.0; no licence file at repo root | C++, JS | 2026-09 | Offline build, CPU traversal, streaming, draw | Old-style GL or WebGL per-patch draws | `nxsbuild` tool yes, but GPL and Qt |
| [AMD DGF SDK](https://github.com/GPUOpen-LibrariesAndSDKs/DGF-SDK) | Unverified: `LICENSE.md` read covers third-party notices; [AMD says](https://gpuopen.com/dgf) "permissive" | C++ | 2026-05 | Offline clustering and compression for ray tracing | D3D12 or Vulkan samples | Compression, no LOD DAG; not useful here |
| [Masked Occlusion Culling](https://github.com/GameTechDev/MaskedOcclusionCulling) (Intel) | Apache-2.0 (`license.txt`) | C++ | **Archived** | Runtime CPU occlusion culling | None; [SSE2, SSE4.1, AVX paths](https://github.com/GameTechDev/MaskedOcclusionCulling/blob/master/README.md) | N/A; the one runtime piece that needs no GPU feature |
| Nanite | Source not open; [Karis et al. 2021](https://advances.realtimerendering.com/s2021/Karis_Nanite_SIGGRAPH_Advances_2021_final.pdf) is method only | | | | | |

Details that matter:
- `clusterlod.h` says it is "intended to either be used as is, or as a reference", and is built on meshoptimizer — [clusterlod.h](https://github.com/zeux/meshoptimizer/blob/master/demo/clusterlod.h). The README calls it a companion project developed with the library — [README](https://github.com/zeux/meshoptimizer/blob/master/README.md).
- Its entry point is `clodBuild(config, mesh, context, callback)`, with `clodBuildHierarchy` for a BVH. `clodCluster::indices` "refer to the original mesh vertex buffer", so all levels share one vertex buffer. Cluster size is asserted to be 4 to 256 triangles; the default target is a 16-cluster partition and a 0.5 simplify ratio — [clusterlod.h](https://github.com/zeux/meshoptimizer/blob/master/demo/clusterlod.h).
- It calls `meshopt_partitionClusters`, not METIS. That function, `meshopt_buildMeshlets*`, `meshopt_simplifyWithAttributes` and `meshopt_simplifySloppy` are declared `MESHOPTIMIZER_API` inside `extern "C"`, not marked experimental at HEAD — [meshoptimizer.h](https://github.com/zeux/meshoptimizer/blob/master/src/meshoptimizer.h).
- Latest release v1.3, "25 Sep" with no year shown on the page, improved `clusterlod.h` and made permissive simplification stable — [releases](https://github.com/zeux/meshoptimizer/releases).
- NVIDIA's own notice says nv_cluster_lod_builder "has been archived" and points to `clusterlod.h` as the maintained equivalent — [README](https://github.com/nvpro-samples/nv_cluster_lod_builder/blob/main/README.md).
- Build cost at scale: Zorah, 1.64B triangles, took 2m 38s on Windows and about 2m 35s on Linux with 16 threads on a Ryzen 7950X, about 45 GB at the measured stage — [Kapoulkine 2025](https://zeux.io/2025/09/30/billions-of-triangles-in-minutes/). The same post's runtime figure is on an NVIDIA 3050 under Vulkan: 16 ms raster.
- Bevy's offline path uses `meshopt` plus `metis-rs` — [from_mesh.rs](https://github.com/bevyengine/bevy/blob/main/crates/bevy_pbr/src/meshlet/from_mesh.rs). Its docs warn of "much greater base overhead" with small amounts of geometry — [mod.rs](https://github.com/bevyengine/bevy/blob/main/crates/bevy_pbr/src/meshlet/mod.rs).
- vk_lod_clusters' compute rasterizer is "not faster than the mesh-shader" in typical scenes, because clusters have triangles larger than a pixel — [README](https://github.com/nvpro-samples/vk_lod_clusters/blob/main/README.md).
- three-virtual-geometry reports 5.7B triangles at 160 to 230 FPS, but this is the author's own run on an M4 Pro through WebGPU — [README](https://github.com/elad12390/three-virtual-geometry/blob/main/README.md). It says nothing about old hardware.
- Go bindings: GitHub repository search for meshoptimizer in Go returned 0 results (5 November 2026). pkg.go.dev lists only [`m31labs.dev/gosx/assetpipe/meshoptim`](https://pkg.go.dev/m31labs.dev/gosx/assetpipe/meshoptim), MIT, which has vertex cache, fetch, overdraw and weld functions and no meshlets or simplification. Its page does not say so outright, but its source files are all Go, so it appears to be its own implementation and not a binding. `proxy.golang.org` lists tags for `github.com/zeux/meshoptimizer`; that only reflects the repo's tags.
- moonBASIC's C++ cgo route already exists: `internal/jolt/cgo_windows.go`, plus Linux and Darwin variants. The project is MIT: `LICENSE`.

### Inferences
- Take meshoptimizer and `clusterlod.h`; compile both through a cgo wrapper like `internal/jolt`. `clusterlod.h` uses `std::vector`, so the wrapper is a `.cpp` file. MIT is compatible with moonBASIC's MIT.
- Skip METIS: no extra dependency is the better trade, and a maintainer-level replacement exists. Karis used METIS, but that does not mean it is required.
- Nexus C++ cannot be linked into an MIT engine without relicensing questions (not legal advice). Its algorithm is published and its MIT JavaScript can be read.
- Load-time build looks affordable, by estimate only: 1.64B triangles in about 158 s across 16 threads is about 0.65M triangles per second per thread, so a 500,000-triangle mesh takes about 1 s on one modern thread, and more on an old CPU. That scene's mix differs from a single mesh, and the figure includes other pipeline stages. Bake and cache by mesh hash for anything large.

### Gaps
- I did not build meshoptimizer here. MinGW compile status and time per million triangles on this machine are unmeasured.
- The DGF SDK's own licence terms were not found in the portion of `LICENSE.md` I read. Carrot's renderer was only grepped. KaHIP, mt-KaHyPar, three-nanite and nanite-webgpu licences rest on GitHub's detection, not the file.
- The year of meshoptimizer's v1.3 release is not shown on the page; 2026 is my inference from repo activity.

## Can three.js r187's MeshletDrawingPipeline and VisibilityBufferDrawingPipeline be ported?

### Takeaway
**Not as code.** The licence allows it, but every part that does the work is TSL compiled to WebGPU compute, and it refuses to run on three's own WebGL backend. On GL 3.3 no shader carries over. On GL 4.3 the culling kernels could be rewritten in GLSL, as a rewrite and not a port. The ideas carry over, and there is a useful surprise: **the r187 meshlet pipeline is not Nanite-style cluster LOD.** It selects one discrete level per object and then culls clusters.

### Cited Findings
- Licence: MIT, "Copyright © 2010-2026 three.js authors" — [LICENSE at r187](https://github.com/mrdoob/three.js/blob/r187/LICENSE). `package.json` at the tag is version 0.187.0. Release r187 was published 2026-10-30; PR #35191 merged 2026-10-29 by sunag, +11,726/-3,162 across 40 files — [PR #35191](https://github.com/mrdoob/three.js/pull/35191).
- Size at r187: `MeshletDrawingPipeline.js` 5,571 lines, 190,386 bytes; `VisibilityBufferDrawingPipeline.js` 954 lines, 37,202 bytes; `utils/MeshletGeometryUtils.js` 417 lines, 15,221 bytes — [pipelines](https://github.com/mrdoob/three.js/tree/r187/examples/jsm/pipelines), [utils](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/utils/MeshletGeometryUtils.js). Both pipeline classes say "This class is experimental and its interface might change".
- Line counts by class, from the source (comment lines excluded by my script, approximate): `MeshletCulling` about 790 lines holding six TSL compute kernels; `createOcclusion` about 90; `MeshletGeometryPool` about 140; `MeshletBatch` about 940; `MeshletDrawingPipeline` about 1,750, mostly three.js scene and render list integration — [source](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/pipelines/MeshletDrawingPipeline.js).
- What it relies on: imports `StorageBufferAttribute`, `IndirectStorageBufferAttribute`, TSL `storage`, `atomicAdd`, `workgroupArray`, `workgroupBarrier`, `instanceIndex`, `vertexIndex`; indirect dispatch and `geometry.setIndirect` — same source, lines 1-33, 2436, 2516.
- Backend gate: `prepareAsync` returns unless `backend.isWebGPUBackend === true`, and support also needs no logarithmic or reversed depth and no clipping planes — lines 3464, 3539 onward. The PR says unsupported objects "fall back to the regular pipeline, as does the WebGL backend" — [PR #35191](https://github.com/mrdoob/three.js/pull/35191).
- **LOD is per object.** "compute shaders select the level of detail per object and cull the objects and clusters against the frustum" (line ~3226); "one thread per object culls the object, selects its level of detail and allocates the clusters of the level" (line ~708). There is no group, partition or DAG in `MeshletGeometryUtils.js`: it simplifies the whole mesh to ratios 1.0, 0.55, 0.25, 0.1, 0.04, 0.015 with `MeshoptSimplifier`, then clusters each level with `MeshoptClusterizer` — [utils lines 35-40, 222-272](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/utils/MeshletGeometryUtils.js). So it depends on meshoptimizer through WASM.
- Clusters are padded to exactly 64 triangles with degenerates, so every cluster is a fixed-size chunk for vertex pulling — same file, "pad to exactly 64 triangles".
- Occlusion: uses depth of clusters visible last frame, two-phase, turning itself off when it does not pay (`OCCLUSION_MIN_CULLED = 0.3`) — source line 65, and [PR #35191](https://github.com/mrdoob/three.js/pull/35191).
- `VisibilityBufferDrawingPipeline`: "small triangles in software by a compute shader, large triangles by the hardware rasterizer"; stores the triangle in 16 bits; uses atomic per-pixel storage buffers; a resolve pass shades with analytic derivatives — [source](https://github.com/mrdoob/three.js/blob/r187/examples/jsm/pipelines/VisibilityBufferDrawingPipeline.js) lines 182-193, 768-786.
- Published results: RTX 4070, 1920×1080, median of 2 runs. Instanced scene: GPU 230.1 → 3.0 ms. City: 5.3 → 2.4 ms. Sponza: GPU 0.60 → 0.73 ms, **slower**, CPU 3.2 → 1.0 ms. The author says gains are largest in "raw scenes that are bound by vertices" and smaller in scenes bound by fragment shading or that have their own LODs and culling — [PR #35191](https://github.com/mrdoob/three.js/pull/35191).
- moonBASIC gates compute and SSBOs on 4.3 or `GL_ARB_compute_shader`, and they return 0 on 3.3 — `docs/GRAPHICS.md:49-55`, `internal/runtime/glmodern.go:160`.

Parts, and what happens to each:

| Part | GL 3.3 | GL 4.3 tier |
| --- | --- | --- |
| LOD table, error accumulation, sloppy fallback (`MeshletGeometryUtils`) | Algorithm carries over; call meshoptimizer's C API directly | Same |
| Normal cone and sphere tests, budgets, self-disabling occlusion policy | Rewrite in Go on the CPU | Rewrite as GLSL 430 compute |
| TSL compute kernels (reset, object cull, cluster cull, args, HZB) | Nothing carries over | Rewrite by hand; no TSL to GLSL compute path exists in moonBASIC |
| Fixed 64-triangle chunks with vertex pulling | Possible with texture buffers (see next section), but means new vertex stages | SSBO pulling, new vertex stages |
| Three.js scene, node material and render bundle plumbing (largest share) | Not applicable to G3N | Not applicable |
| Software rasterizer and visibility buffer | Impossible: needs compute and atomics | Possible in theory, but needs a material resolve pass that moonBASIC's forward renderer does not have. Reject |

### Inferences
- A 4.3 rewrite of just culling and indirect draws is an estimated 1,000 to 2,000 lines of Go and GLSL, plus changing the vertex stage of each batched program (`standard`, `bsshadow`, `mbphysical`). That serves only machines that already have the most headroom. It is the wrong first step for an engine aimed at ten-year-old GPUs.
- sunag's choice is itself evidence: one of the most capable implementers shipped discrete per-object LOD with cluster culling and no DAG. Per-object levels mean a large mesh with its near edge at the camera stays at full detail, though off-screen and back-facing clusters are culled. But they also need no crack handling, and that simplicity is what moonBASIC could actually adopt.
- The Sponza row is the warning for moonBASIC: where a scene is already optimized, the gain can be zero or negative.

### Gaps
- I did not run the three.js examples. The PR's table is the author's single-GPU result, median of two runs. No published result exists for an old or integrated GPU.
- I did not read every line of the 5,571; class-level conclusions come from the class docs, imports, kernels and grep.

## What is the honest best-fit design for OpenGL 3.3 with no compute, and what does it give up?

### Takeaway
Cluster LOD built by `clusterlod.h`, selected and culled in Go on the CPU, drawn from one vertex buffer with `glMultiDrawElementsBaseVertex`, using **large clusters (about 256 triangles or more), on single static meshes and merged world chunks only.** Copies keep instancing with discrete LOD, because GL 3.3 gives a multi-draw no draw identity. This is Batched Multi-Triangulation from 2005 with better tools, not Nanite. It bounds triangle cost; it does not give "unlimited geometry".

### Cited Findings

Precedent for CPU-driven cluster LOD on old APIs:
- Batched Multi-Triangulation (2005) uses "precomputed optimized triangle patches" so that CPU load stays minimal, and rendered about 4M triangles per frame at 35 fps on a GeForce 6800 GT at 800×600 — [Cignoni et al.](https://vcg-legacy.isti.cnr.it/Publications/2005/CGGMPS05/BatchedMT_Vis05.pdf). Karis lists Batched Multi-Triangulation as prior work in the explicit-dependency direction Nanite took — [Karis 2021, slide 44](https://advances.realtimerendering.com/s2021/Karis_Nanite_SIGGRAPH_Advances_2021_final.pdf).
- Nexus still works this way: one `glDrawElements` per patch with 16-bit indices in C++ ([renderer.cpp](https://github.com/cnr-isti-vclab/nexus/blob/master/src/common/renderer.cpp)) and `gl.drawElements` in WebGL ([nexus.js](https://github.com/cnr-isti-vclab/nexus/blob/master/html/js/nexus.js)). Default patch size is 16,286 faces — [Nexus docs](http://vcg.isti.cnr.it/vcgtools/nexus//).
- Frostbite on PS3-era hardware rasterized a coarse z-buffer on the CPU or SPU with low-poly occluders and capped vertices at 10,000 per frame — search summary of [Andersson 2009](https://www.slideshare.net/slideshow/parallel-graphics-in-frostbite-current-future-siggraph-2009-1860503/1860503), not read in full. Collin's [Culling the Battlefield](https://www.slideshare.net/DICEStudio/culling-the-battlefield-data-oriented-design-in-practice) outline covers software occlusion; I could not read its body.

Evidence on cluster size and draw submission, the main risk:
- Minimum triangles per draw before a GPU's front end limits primitive rate: NVIDIA 16, Intel 32 (HD 4000) and 16 (HD 4600), AMD 256 as worst case, 64 on HD 7750 — [g-truc 2014](https://www.g-truc.net/post-0666.html). This used `glDrawArraysInstanced` in a tight loop, not multi-draw, with driver versions unstated.
- Wihlidal: "256 seems to be the sweet spot" for culling clusters; 64 is not optimal because "the CP bottlenecks on too many draws" — [GDC 2016, slide 29](https://www.slideshare.net/slideshow/optimizing-the-graphics-pipeline-with-compute-gdc-2016/59747720) (AMD GCN-centric; read through a page extract, as the original PDF returned 403).
- JMS55 tried one multi-draw sub-draw per cluster and reported "this still performed poorly" because of command processor overhead, then moved to a single `draw_indirect` — [Bevy 0.14 post](https://jms55.github.io/posts/2024-06-09-virtual-geometry-bevy-0-14).
- A forum reply says small multi-draw sub-draws cannot share vertex shader wavefronts and suggests instancing — [Khronos forum](https://community.khronos.org/t/setting-up-an-efficient-renderer-in-opengl/112067). This is a forum claim.
- `glMultiDrawElementsBaseVertex` behaves like a loop of `glDrawElementsBaseVertex`, needing GL 3.1 per the [reference page](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glMultiDrawElementsBaseVertex.xhtml). Go binding is present: `go-gl/gl v3.3-core package.go:11453`. moonBASIC does not use it today: batching calls `gl.DrawElementsInstanced` (`internal/runtime/batch.go:223`).
- **No draw identity:** `gl_DrawIDARB` comes from [ARB_shader_draw_parameters](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_shader_draw_parameters.txt), an extension, not 3.3 core. A shader inside a multi-draw cannot learn which object it belongs to on the baseline.

Evidence on what culling yields, from shipped work with compute:
- Assassin's Creed Unity: 64 triangles per cluster, `MultiDrawIndexedInstancedIndirect`, CPU quad tree first. Result: 1-2 orders of magnitude fewer draws, 20-40% of triangles culled, "<10% of geometry rendering" overall gain, **30-80% of shadow triangles culled** — [Haar and Aaltonen 2015, slides 12-14, 37](https://advances.realtimerendering.com/s2015/aaltonenhaar_siggraph2015_combined_final_footer_220dpi.pdf). The same talk's RedLynx path uses a fixed 64-vertex strip, manual vertex fetch and `DrawInstancedIndirect` (slide 10).
- Wihlidal's example scene culled 78% of triangles per-triangle, of which orientation 46% — slide 83, scene dependent. Per-triangle culling needs compute, so a CPU design only gets the cluster-level part.

Optional GL 3.x steps:
- Instance culling with transform feedback, geometry shader and instancing needs only GL 3.2. The author's informal demo cut 2.7B triangles by over 99.9%, running at 6-7 fps on an HD 2600 XT and over 90 fps on an HD 5770 — [Rákos 2010](https://rastergrid.com/blog/2010/02/instance-culling-using-geometry-shaders/). He warns of early driver problems; the page is a legacy post.
- Hi-Z culling through a geometry shader and transform feedback: under 0.2 ms to build on an HD 5770 at 1024×768. The demo targets GL 4.0 — [Rákos 2010](https://rastergrid.com/blog/2010/10/hierarchical-z-map-based-occlusion-culling/). The reduction chain itself is ordinary framebuffer work, so 3.3 looks possible (inference, untested).
- Conditional render and transform feedback bindings exist in the 3.3 package: `package.go:8888, 8910`.

What Nanite has that this does not:
- Software rasterizer: tiny triangles are "terrible" for a hardware rasterizer, with at most 4 triangles per clock of setup. Nanite's compute rasterizer is 3x faster on average, for clusters with edges under 32 pixels, and needs 64-bit atomics — [Karis 2021, slides 80-88](https://advances.realtimerendering.com/s2021/Karis_Nanite_SIGGRAPH_Advances_2021_final.pdf). Rasterizer efficiency falls from 100% to 6.25% as triangles shrink — Wihlidal slide 51.
- Visibility buffer and deferred material pass (Karis, slide 102 onward). Forward shading keeps quad overdraw on every small triangle.
- Two-pass previous-frame HZB culling per cluster, DAG streaming and compression, and per-triangle culling.
- Karis describes why grouping exists: locked cluster boundaries otherwise "collect dense cruft", which alternating groups per level solves (slides 31-39). `clusterlod.h` handles this.

### Inferences

The design, which is my synthesis and is untested:
1. **Build:** `clodBuild` on a worker or as a cached bake, 256-triangle clusters, one vertex buffer, 32-bit indices. Index data is about 2x the original with a 0.5 ratio (estimate from the geometric series). No new vertex data, which also removes the current `blendAttrs` step.
2. **Scope:** single static meshes, child meshes of loaded models, and the world-space merged cells that `batch_merge.go` already builds. These need one transform, so the missing draw identity does not matter. A uniform per object, then one multi-draw, keeps the existing materials and vertex shaders unchanged. This is the biggest practical advantage over vertex pulling.
3. **Copies:** stay on instancing with discrete levels. Per-copy cuts would need a per-copy cluster list, and no 3.3 route does this cheaply except pulling.
4. **Select and cull in Go, per view:** apply the rule in the header, frustum and cone test, with `clodBuildHierarchy` for large meshes. Merge contiguous index ranges before submitting; this cuts sub-draws a lot because neighbours in a level tend to be adjacent (expected, unmeasured). Estimate: 1M visible triangles at 256 per cluster is about 4,000 clusters, about 8,000 tests with the DAG, about 0.15 ms per view at 20 ns each, about 1 ms across 7 views. At 64 per cluster that is about four times worse.
5. **Shadow passes:** run the same selection with the texel-based error `binDepthLODs` already computes. This is where the Unity data suggests the most return.
6. **Error floor:** keep at least 1 px, as now (`batchLODMaxErr`), and prefer 2 px on low tier, as triangles near pixel size are the case hardware handles worst.
7. **4.x tier, later, optional:** move selection to compute only if the CPU cost proves to matter. Same data, same bake.

Four submission routes to benchmark before committing, since sources disagree and none tested this on a Radeon 780M or Intel HD:

| Route | Needs | Pro | Con |
| --- | --- | --- | --- |
| A. `glMultiDrawElementsBaseVertex`, merged runs | 3.2 | No shader changes | Sub-draw cost may be bad on AMD |
| B. Stream one index buffer per frame, one `glDrawElements` | 3.0 | One draw, front end stays happy | Uploads about 12 MB per view per 1M triangles (estimate); per-view copies |
| C. Fixed chunks, instanced, cluster ID as instance attribute, vertices from a texture buffer | 3.1 | One draw per material, works with copies | New vertex stage in every program; slower fetch; loses index reuse |
| D. Coarse patches of 2,000 to 16,000 triangles, plain draws | 1.x | Proven by BMT and Nexus | Coarser culling; more CPU draw overhead per patch |

Start with A against D. Transform feedback, occlusion queries and Masked Occlusion Culling come last and only if measured need appears. moonBASIC's own test found a far-plane cut gave the city nothing (`docs/GRAPHICS.md:90`).

### Gaps
- I found no published work doing exactly this: CPU cluster-DAG selection with `glMultiDraw*` on GL 3.3. BMT and Nexus are the nearest, with per-patch draws. Everything above about route costs is inference until benchmarked.
- No source gives sub-draw cost for `glMultiDrawElements` on Intel HD, GT 730 or Radeon 780M. Driver behaviour may turn it into a plain loop.
- The Wihlidal PDF returned 403, and his talk is about compute on AMD GCN, not this design. BMT's 4M triangles figure is on 2005 hardware with simple shading and does not predict moonBASIC's shaded cost.

## Does this solve a problem the engine has?

### Takeaway
**Not on the evidence in the repository today.** Every measured scene is limited elsewhere (CPU walks, shadows, fragment work), or is already fixed by the simple discrete LOD. There is a real gap, single high-poly meshes get no LOD at all, but no benchmark exercises it. Extending discrete LOD closes most of that gap at a small fraction of the cost. Cluster LOD should wait until a benchmark shows discrete LOD failing.

### Cited Findings

What LOD exists (`internal/runtime/batch_lod.go`, `docs/GRAPHICS.md:74-76`, `docs/ROADMAP.md:549`):
- Two simplified versions, ratios 0.5 and 0.25 (`batchLODRatios`), for geometries of 256 to 500,000 triangles, on a worker. The chosen level is the simplest with error at most 1 pixel, or 1 texel in directional shadows.
- It runs only on batched groups: shared geometry with equal material. A mesh with no copies "keeps its full detail" (`batch_lod.go:16`). Child meshes of loaded models are not grouped. Point and spot shadow tiles and orthographic views stay at full detail. A mesh over 500,000 triangles gets nothing.
- Merged static meshes have no LOD: `batch_merge.go` has no LOD code, meshes over 20,000 triangles stay alone, and cells are culled as a whole.
- Distance uses the nearest point of a copy's bounding sphere (`batch_lod.go:230`), so a large mesh you stand next to is full detail everywhere.
- Trees have hand-built far models (`docs/GRAPHICS.md:163`) and terrain has its own levels.
- `internal/meshsimplify` is a Go quadric simplifier copied from gosx v0.57.1, MIT (`doc.go`). It locks vertices whose position another vertex shares (`simplify.go:264-267`) and has a boundary weight, but no way to lock a group boundary hard from outside, as a cluster DAG requires. That is another reason to use meshoptimizer.

Measured evidence, Radeon 780M, from `docs/GRAPHICS.md:82-90, 240-249`. The doc says timings "moved a lot from run to run on a shared machine":

| Scene | Result | What limits it |
| --- | --- | --- |
| 4,000 rocks of 8,000 triangles (32M at full detail, my multiplication) | 18.8 ms → 9.8 ms with discrete LOD and the rest | **Triangles did matter here, and two-level discrete LOD already fixed about half** |
| City: 3,600 meshes, 28,800 copies | 36.7 ms, 218 draws; cutting far plane to 120 units gave no speedup | Not visible geometry. Above about 10,000 entities most CPU time is per-frame walks, chiefly material checks |
| 50,000 cubes | 44-57 ms, 1 draw | CPU walks, same note |
| Island orbit | 16.5 ms, 6.3 ms render CPU, "the GPU sets the frame time" | GPU, split below |
| Island forest | About a million triangles, about 230 draws (`:202`) | Not reported as a problem |

- Per-pass GPU on the island at High, before later work: Shadow 4.02, Reflect 0.27, Scene 6.71, Post 1.02, frame 13.6 ms. Later changes reached 9.6 ms through far cascades redrawing in turn, cheaper terrain materials and no multisampled window — `docs/GRAPHICS.md:243-249`. None of those were triangle reductions.
- Terrain occlusion of trees saved 10 to 20% of GPU time in ground views (`:204`), so hidden tree work costs something real, though vertex and fragment shares are not separated.
- Low tier takes 4.4 to 6.3 ms at 1280×720 and 9 to 12 ms at 2560×1440, four times the pixels (`:189-195`).
- The engine report calls redraws the largest remaining item: with 4 cascades, reflection, refraction and main there are 7 views, an upper bound of 13.3M triangles per frame for 1.9M triangles (labelled an estimate there) — `reports/Engine rendering techniques for moonBASIC.md:155`.

### Inferences
- From the Low tier figures, a line through time against pixels gives about 2 to 3 ms that does not scale with pixels and about 1.5 to 2.5 ms per 720p of pixels (estimate, mixing scene ranges). So on the island the pixel side at least matches the non-pixel side, even on the cheapest shaders, and geometry is only part of the non-pixel share. Dynamic resolution helps the pixel side but not vertex work.
- Geometry reduction looks most valuable in **shadow passes and redraws**, not as a replacement for what the main view does. Existing discrete LOD is already wired into directional shadows.
- Target hardware argues the same way: ten-year-old integrated GPUs are usually short of pixel rate and bandwidth first (general reasoning, and the tier table measures only a 780M). The low tier already cuts trees to 45%.
- Order of work I would suggest, cheapest first:
  1. **Measure.** Add a benchmark with one 2M to 5M triangle static mesh and a loaded model scene. Fix `StatsTriangles`, which sums geometry of visible entities (`internal/runtime/profiler.go:149-170`) and therefore cannot report triangles actually drawn after LOD, culling or per view. Add per-view drawn counts. Run with `moon bench` on real old hardware, which has not yet happened (commit `121c84d`).
  2. **Extend discrete LOD:** to uncopied meshes, child meshes of loaded models and merged-out big meshes; add more levels, as 0.25 is a high floor against three's 0.015. Same shaders, no new draw path. Consider swapping in `meshopt_simplifyWithAttributes` for speed and quality, once cgo wrapper exists.
  3. **Cluster culling without a DAG**, as three.js does, for big meshes, on the CPU. The Unity numbers suggest shadows gain most.
  4. **Cluster LOD DAG**, only if step 1 shows large meshes close to the camera still dominating after step 2. Its unique benefit is that one object can be coarse far away and fine nearby.
  5. **Compute tier and software rasterizer:** do not build.
- Who benefits matters too. moonBASIC scripts use primitives, `CopyEntity`, generated forests and terrain; only games loading scanned or sculpted high-poly assets would notice cluster LOD. If that is a target, it is a product decision and not something the current numbers can answer.

### Gaps
- No repository measurement isolates vertex from fragment cost in any pass, and none measures triangles drawn per view after LOD.
- Every engine number is from one Radeon 780M. Nothing is measured on Intel HD or GT 730; the low-tier hardware column is explicitly an estimate (`docs/GRAPHICS.md:189`).
- No test scene has a single high-poly mesh, so the case cluster LOD is best at is unmeasured in either direction. Absence of evidence here is not evidence that it would not help.
- macOS OpenGL is believed to stop at 4.1, which would rule out a compute tier there; I did not verify this for this report.
