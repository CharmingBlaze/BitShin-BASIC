# Asset pipelines, procedural generation, and graphics debugging for moonBASIC

Repo facts below come from reading the working tree on 2026-11-03: `go.mod`, `internal/runtime`, and the module cache. They are marked "(repo)". They are not web sources.

## KTX2 and Basis Universal: can Go use them without heavy dependencies? Which GPU formats exist on GL 3.3 across vendors, macOS, and Intel?

### Takeaway
No pure-Go Basis transcoder turned up, so runtime BasisLZ means small-ish C++ cgo, a WASM runtime, or writing a port. The cheapest route for a desktop engine is to transcode at `moon build` time to native BC formats and ship plain KTX2/DDS, with BC1/BC3/BC4/BC5 as the floor, BC7 when the extension exists, and ETC2 and ASTC not used on desktop.

### Cited Findings
**Basis Universal library**
- Apache 2.0. The page describes v2.5, with codecs ETC1S, UASTC LDR 4x4, UASTC HDR 4x4, ASTC HDR 6x6, UASTC HDR 6x6 Intermediate, ASTC LDR 4x4-12x12, XUASTC LDR, and XUBC7 — [basis_universal](https://github.com/BinomialLLC/basis_universal)
- The transcoder is one source file (`transcoder/basisu_transcoder.cpp`) with no third-party dependencies except optional Zstd. Without Zstd it is limited to non-Zstd codecs. XUBC7 always uses Zstd, and XUASTC LDR defaults to the Zstd profile — [basis_universal](https://github.com/BinomialLLC/basis_universal)
- LDR targets include BC1-5, BC7, ETC1, ETC2 RGBA, EAC R11/RG11, and ASTC, plus uncompressed 8888/565/4444. HDR targets include BC6H (unsigned), ASTC HDR, RGB_16F, RGBA_16F, and RGB_9E5 — [basis_universal](https://github.com/BinomialLLC/basis_universal)
- The source's KTX2 path needs Zstd for supercompressed UASTC, and a build flag can disable that support — [basisu_transcoder.cpp mirror](https://skia.googlesource.com/external/github.com/BinomialLLC/basis_universal/+/0a546490dbbbc27b5461a83c0e3c2e66709d9575/transcoder/basisu_transcoder.cpp)
- Precompiled WASI executables (`basisu_mt.wasm`, `basisu_st.wasm`) ship in `bin`. 32-bit WASI limits ASTC/XUASTC compression to about 4 megapixels — [basis_universal](https://github.com/BinomialLLC/basis_universal)
- Khronos publishes optimized WASM transcoders that accept only low-level payloads, so the `.ktx2` or `.basis` container must be parsed by other means first — [Basis-Universal-Transcoders](https://github.com/KhronosGroup/Basis-Universal-Transcoders)
- A pure-Rust `basisu` crate reads `.ktx2`/`.basis` and transcodes at load time. It may be useful as a port reference, but I did not check its license or completeness — [docs.rs basisu](https://docs.rs/crate/basisu/0.1.0/source/README.md)

**Go building blocks**
- wazero is a WebAssembly Core 1.0/2.0 runtime written in Go, with zero dependencies and no CGO. Its interpreter has no GOARCH/GOOS-specific code — [wazero](https://github.com/wazero/wazero)
- `klauspost/compress/zstd` is pure Go, and its compressor and decompressor are marked stable. `DecodeAll` with `WithDecoderConcurrency(0)` runs without allocations after warmup — [klauspost zstd](https://github.com/klauspost/compress/tree/master/zstd)
- (repo) `github.com/klauspost/compress v1.18.0` is already in `go.mod`, so Zstd-supercompressed KTX2 adds no new dependency.

**Which payload for what**
- ETC1S (BasisLZ) has high storage efficiency and prioritizes luma, so it suits albedo and not normal maps. UASTC handles all 8-bit data, including normal and height maps. It uses 16-byte blocks, with Zstd in KTX2 — [KTX Developer Guide](https://github.com/KhronosGroup/3D-Formats-Guidelines/blob/main/KTXDeveloperGuide.md)
- Recommended UASTC order: ASTC 4x4, then BC7, then RGBA8. Lower-quality fallbacks are BC1/BC3 and ETC. For ETC1S on desktop without ETC support: BC7, then BC1/BC3. Red: BC4. Red-Green: BC5 — [KTX Developer Guide](https://github.com/KhronosGroup/3D-Formats-Guidelines/blob/main/KTXDeveloperGuide.md)

**GPU support by vendor**

| Platform | BC1/3/4/5 | BC7 | ETC2/EAC | ASTC |
|---|---|---|---|---|
| NVIDIA desktop | Yes | GeForce 400+ | Not supported | Not supported |
| AMD | Yes | HD 5000+ | Not supported | Not supported |
| Intel (Win/Linux) | Yes | Gen7+ | Gen8+ | Gen9+, but see the conflict below |
| Intel Macs | BC only | BC only | No | No |
| Apple M1+ | Yes | Metal only | Metal required | Metal required |

Table source: [KTX WebGL guide](https://raw.githubusercontent.com/KhronosGroup/3D-Formats-Guidelines/main/KTXDeveloperGuideWebGL.md). It also says "Intel-based Macs support only BC* formats".

- **Conflict on Intel ASTC:** the table above says Gen9 and newer. Phoronix reports that ASTC hardware was removed at Gen12.5 (Xe HP, DG2/Arc), and that Mesa's Iris GL driver emulates it — [Phoronix](https://www.phoronix.com/news/Intel-Gen12.5-No-ASTC); [Phoronix on ANV emulation](https://www.phoronix.com/news/Intel-ANV-ASTC-LDR-Emulation). Treat ASTC as absent on desktop.
- **Nuance on ETC:** the Developer Guide says ETC/EAC is widely exposed by desktop drivers but mostly decompressed in software, with only Intel desktop GPUs newer than Haswell native — [KTX Developer Guide](https://github.com/KhronosGroup/3D-Formats-Guidelines/blob/main/KTXDeveloperGuide.md). The WebGL table's "Not supported" for AMD and NVIDIA refers to native support. So an extension check can succeed and still give CPU decompression.
- Extension checks: S3TC `GL_EXT_texture_compression_s3tc` (sRGB via `GL_EXT_texture_sRGB`), BPTC `GL_ARB_texture_compression_bptc`, RGTC `GL_ARB_texture_compression_rgtc`, ETC2 `GL_ARB_ES3_compatibility`, ASTC `GL_KHR_texture_compression_astc_ldr` — [KTX Developer Guide](https://github.com/KhronosGroup/3D-Formats-Guidelines/blob/main/KTXDeveloperGuide.md)
- On macOS the OpenGL stack stops at 4.1, and a developer on Apple's forum describes wanting BC7 in a GL 4.1 project; Apple's forum answer about BC and ASTC refers to Metal, not OpenGL — [Apple forums 693265](https://developer.apple.com/forums/thread/693265); [Apple forums 679660](https://developer.apple.com/forums/thread/679660)

**glTF rules that constrain this (KHR_texture_basisu)**
- ETC1S must use supercompression scheme 1. UASTC must use scheme 0 or 2. Width and height must be multiples of 4 (powers of 2 are only an implementation note). `KTXorientation` must be `rd` or omitted, and `KTXswizzle` must be `rgba` or omitted. Color data must be BT709 primaries with sRGB transfer; non-color data must be linear — [KHR_texture_basisu](https://raw.githubusercontent.com/KhronosGroup/glTF/main/extensions/2.0/Khronos/KHR_texture_basisu/README.md)
- With a fallback, the extension appears only in `extensionsUsed` and core `source` points at PNG/JPEG. Without one, it also appears in `extensionsRequired` and core `source` is omitted — [KHR_texture_basisu](https://raw.githubusercontent.com/KhronosGroup/glTF/main/extensions/2.0/Khronos/KHR_texture_basisu/README.md)

### Inferences
**Adoption order, cheapest first**
1. Build-time transcode. `moon build` already produces pak files (recent commits, repo). Run the Basis tool as an optional offline step and write native BC payloads, so the runtime gets no new code. Basis's value on the web is one file for many GPUs. Desktop GL does not need that, because BC covers every desktop vendor and Intel Macs.
2. Runtime loading of KTX2 with native BC payloads plus Zstd, using the existing klauspost dependency. This is pure Go and a small change.
3. Runtime Basis, only if user-supplied `.ktx2` from third-party glTF files must load. Two sub-options:
   - cgo around the single C++ file with Zstd disabled. This handles ETC1S and uncompressed UASTC only, and it makes a C++ compiler a build requirement. The project already uses cgo through go-gl and glfw (repo), so only the C++ part is new.
   - wazero plus a WASM transcoder. This keeps pure-Go builds, but transcoding speed under wazero is unmeasured and the Khronos modules may expect a JS host. Prototype before committing.
4. A full Go port is not worth it now. The transcoder covers many codecs, although a port limited to ETC1S to BC1/BC3/BC4/BC5/BC7 would be much smaller.

**Runtime format picker for desktop**
- BC7 if BPTC is exposed, else BC3/BC1.
- BC4/BC5 for one-channel and two-channel data.
- Never choose ETC2 or ASTC just because the extension string appears, since that can mean software decode. The Developer Guide's "ETC first for ETC1S" ordering would hurt on desktop.
- If moonBASIC targets macOS GL, assume BC1/BC3 are present and check BC7 at runtime.
- For normal maps, BC5 beats BC1/BC3 where possible. Do not encode them as ETC1S.
- Keep the sRGB flag from the KTX2 DFD rather than guessing, because a wrong transfer function will show up in the Texture Encoding Test (see the glTF section).

### Gaps
- I found no maintained Go bindings or Go port of Basis Universal. Two searches returned none, which is not proof that none exists.
- wazero's license was not visible in the results (I recall Apache-2.0, unverified), nor was any wazero performance data for Basis.
- `klauspost/compress` license: the page says only "a Go standard license", so check its LICENSE file.
- Apple's actual macOS GL 4.1 compressed format list was not found. One old forum post says only S3TC ([Khronos forum](https://community.khronos.org/t/glcompressedteximage2d/48852)) and it is probably dated. Whether RGTC and BPTC are exposed under Apple's GL needs a test on real hardware.
- Which formats are core in GL 3.3 was not sourced. From memory: RGTC is core in 3.0, BPTC in 4.2, ETC2 in 4.3 (the last matches a [search snippet](https://www.gamedev.net/forums/topic/649558-texture-compression-in-opengl/) that I did not open), and S3TC is never core but universally exposed. Verify this.

## How do you load HDR environment maps (.hdr, OpenEXR) in Go and prefilter them on GL 3.3 for image-based lighting?

### Takeaway
Write a small in-tree Radiance `.hdr` decoder, add `mokiat/goexr` (MIT) for ZIP and uncompressed EXR, and fail clearly on PIZ and DWA. Prefilter with the standard split-sum pipeline in fragment shaders, then cache the results at build time.

### Cited Findings
**Go loaders**
- `mokiat/goexr`: MIT. Supports `NO_COMPRESSION` and `ZIP_COMPRESSION`, `HALF` and `FLOAT` channels, and single-part scanline images only. The repo shows only 2 commits. It does not list PIZ, ZIPS, or DWA, tiled, or multipart — [goexr](https://github.com/mokiat/goexr)
- The tagged v0.1.0 (June 2022) is older and narrower than the main branch, and it returns an RGBA image — [pkg.go.dev goexr](https://pkg.go.dev/github.com/mokiat/goexr/exr). Pin a commit.
- `Opioid/rgbe`: Go, with RLE and uncompressed read, returning `[]float32`. It does not read gamma or exposure headers, has no LICENSE file on the page, and tests are an open task in its README — [rgbe](https://github.com/Opioid/rgbe). Without a license it cannot safely be vendored.
- `mdouchement/hdr`: a broader toolkit with Radiance RGBE/XYZE, PFM, and tone mapping; heavier than a loader — [hdr](https://github.com/mdouchement/hdr)

**Prefilter pipeline (LearnOpenGL)** — all from [LearnOpenGL Specular IBL](https://learnopengl.com/PBR/IBL/Specular-IBL)
- Equirectangular source to cubemap, with 512 per face as the source resolution in the solid-angle calculation.
- Prefiltered cubemap: `GL_RGB16F`, 128x128 per face at mip 0, 5 mip levels, `GL_LINEAR_MIPMAP_LINEAR`, clamp to edge.
- Roughness per mip = `mip / (maxMipLevels - 1)`. Sample at runtime with `roughness * 4.0`.
- GGX importance sampling with a Hammersley sequence, with `a = roughness * roughness`, assuming N = V = R. Weight by NdotL and divide by total weight. The article uses 4096 samples in one snippet and 1024 in the final shader.
- Fireflies: compute each sample's pdf from D, NdotH, and HdotV, derive a source mip from sample solid angle versus texel solid angle, and sample a trilinear-filtered source cubemap with generated mips.
- BRDF LUT: `GL_RG16F`, 512x512, X = NdotV, Y = roughness, 1024 samples, with the IBL k term (`α²/2`).
- Enable `GL_TEXTURE_CUBE_MAP_SEAMLESS` to remove seams at high roughness.

**HDR block compression of the results**
- Basis can target BC6H (unsigned) and RGB_9E5 for HDR — [basis_universal](https://github.com/BinomialLLC/basis_universal). BC6H needs the BPTC extension (see previous section).

### Inferences
- Radiance RGBE is a small format, so an in-tree decoder of a few hundred lines is better than depending on an unlicensed package. This is my estimate of size, not a sourced figure. Include both RLE variants and honor EXPOSURE only if needed.
- Everything in the pipeline can run in fragment shaders rendering into cube-face FBOs, so compute shaders and GL 4.3 are not required.
- Add the bake to `moon build`: store the prefiltered cubemap, irradiance, and LUT as cached assets so startup does not pay for 1024-sample passes. The LUT is scene-independent and can ship once with the engine. `RGB9_E5` or `RGB16F` are safe storage targets. Use BC6H only where BPTC is present.
- Bright suns in HDRIs: mip-based sampling reduces fireflies but does not remove the energy problem, so offer an optional clamp on source radiance.
- (repo) `internal/runtime/envcube.go`, `pbr.go`, and `pbr_shaders.go` exist. I did not read them, so check how much of this already exists before writing a new pipeline.
- Tell users what to do when an EXR is rejected: "re-save as ZIP or uncompressed scanline" in the error message. Otherwise goexr's limits will read as bugs.

### Gaps
- The Filament documentation fetch returned only a redirect, and a search for Karis split-sum and spherical-harmonic irradiance returned irrelevant results. So I have no sourced detail on SH irradiance, Filament's `cmgen`, or the original split-sum paper. Worth reading before implementation: Filament docs and Karis's "Real Shading in Unreal Engine 4".
- I found no pure-Go EXR decoder with PIZ or tiled support. The search did not show one, and I did not find how common PIZ-compressed HDRI downloads are, so the real-world hit rate for goexr is unknown.
- `mdouchement/hdr` license was not checked.
- No benchmarks for prefilter time on integrated GPUs.

## What does glTF need for correctness: extensions worth supporting, the sample assets as a conformance suite, the validator?

### Takeaway
Treat the Testing list in glTF-Sample-Assets as the regression suite, compare renders against the Khronos Sample Renderer, and use the validator to separate bad assets from loader bugs. Add cheap extensions first: `KHR_texture_transform`, `KHR_materials_unlit`, `KHR_materials_emissive_strength`, `KHR_mesh_quantization`, `EXT_meshopt_compression`, and `EXT_mesh_gpu_instancing`.

### Cited Findings
**Extension status**
- Ratified KHR extensions include: `KHR_animation_pointer`, `KHR_draco_mesh_compression`, `KHR_gaussian_splatting`, `KHR_interactivity`, `KHR_lights_punctual`, `KHR_materials_anisotropy`, `_clearcoat`, `_dispersion`, `_emissive_strength`, `_ior`, `_iridescence`, `_sheen`, `_specular`, `_transmission`, `_unlit`, `_variants`, `_volume`, `KHR_mesh_quantization`, `KHR_node_hoverability`, `_selectability`, `_visibility`, `KHR_texture_basisu`, `KHR_texture_transform`, `KHR_xmp_json_ld` — [glTF extensions](https://github.com/KhronosGroup/glTF/tree/main/extensions)
- In progress: `KHR_materials_diffuse_transmission` (release candidate), `KHR_materials_subsurface` (initial draft). `EXT_mesh_gpu_instancing` and `EXT_meshopt_compression` are listed under the EXT prefix. `KHR_meshopt_compression` did not appear on the page at fetch time — [glTF extensions](https://github.com/KhronosGroup/glTF/tree/main/extensions)

**Sample assets**
- `Models/` holds each model as `.gltf` with separate resources, embedded `.gltf`, or `.glb`. Generated lists: Showcase, Complete, Testing, Core Only, PBR tests, Issues. The Testing list is the one meant for viewers, converters, and other software — [glTF-Sample-Assets](https://github.com/KhronosGroup/glTF-Sample-Assets)
- Licenses vary per model and are in each model's README. The earlier glTF-Sample-Models repo was archived, with `2.0` renamed `Models` and some assets removed over licensing — [glTF-Sample-Assets](https://github.com/KhronosGroup/glTF-Sample-Assets)
- Useful tests, with what each checks — [Models-testing.md](https://raw.githubusercontent.com/KhronosGroup/glTF-Sample-Assets/main/Models/Models-testing.md):
  - Alpha Blend Mode Test: alpha modes
  - Texture Settings Test: single- and double-sided rendering and texturing modes
  - Texture Encoding Test: correct transfer functions
  - Texture Linear Interpolation Test: filtering on linear values after sRGB decode
  - Texture Coordinate Test: UV relation to geometry
  - Texture Transform Test and Multi Test: `KHR_texture_transform`
  - Normal-Tangent Mirror Test: supplied tangents
  - Normal-Tangent Test: generated tangents
  - Morph-Primitives Test, Morph Stress Test (up to 8 targets), Simple Morph
  - Simple Skin, Recursive Skeletons (reused meshes, recursive skeletons)
  - Simple Sparse Accessor
  - Vertex Color Test, Box Vertex Colors
  - MultiUV Test
  - Box with interleaved position and normal attributes
  - Orientation Test
  - Negative Scale Test

**Validator**
- Checks JSON syntax, GLB v2 container, asset properties and references, binary buffer data (accessor values, min/max, sparse, animations), images, and supported extensions. Apache-2.0. CLI: `gltf_validator [<options>] <input>`, which accepts a directory recursively, exits non-zero on any error, and writes a `<asset>.report.json`. Also available via npm and a browser tool at github.khronos.org/glTF-Validator. Source builds need the Dart SDK — [glTF-Validator](https://github.com/KhronosGroup/glTF-Validator)

**Reference renderer**
- The Khronos glTF Sample Renderer is the official reference (WebGL), Apache-2.0. It supports many `KHR_materials_*`, Draco, Meshopt, quantization, basisu, WebP, texture transform, punctual lights, and animation pointers. Its regression suite renders in headless Chromium and diffs with pixelmatch against a baseline made on the same machine and GPU — [glTF-Sample-Renderer](https://github.com/KhronosGroup/glTF-Sample-Renderer)

### Inferences
- There is no conformance suite in the formal sense in these sources. The workable substitute has three steps:
  1. Run the validator on the input, so asset errors are not chased as loader bugs.
  2. Load every model in the Testing list and check for no crash and no unknown-extension surprise.
  3. Compare screenshots against golden images you inspected by hand against the Sample Renderer. Same-machine diffs prove only "unchanged", and the reference renderer is what establishes "correct".
- Do not vendor the assets into the repo without checking per-model licenses. Fetch them in a test script instead.
- Reject a file cleanly when `extensionsRequired` lists something unsupported, rather than rendering it wrong. Prefer the core `source` when `KHR_texture_basisu` is unsupported and a fallback exists.
- (repo) A search of `internal/*.go` for `KHR_*`/`EXT_*` strings found only `KHR_lights_punctual` in `light_shade.go` and an S3TC name in `compressed_tex.go`. So the glTF loader appears not to dispatch on extensions by name, or the loader lives elsewhere. Confirm by reading the loader before planning this work.

**Suggested extension tiers (my judgment)**
- Tier 1, parse-level with little renderer change: `KHR_texture_transform`, `KHR_materials_unlit`, `KHR_materials_emissive_strength`, `KHR_lights_punctual`, `KHR_mesh_quantization`, WebP textures (the engine already decodes WebP).
- Tier 2, data or pipeline: `EXT_meshopt_compression` (decoder in Go is a modest port), `EXT_mesh_gpu_instancing` (maps onto `instance.go`), `KHR_texture_basisu` (see first section), `KHR_animation_pointer`, `KHR_materials_variants`.
- Tier 3, shader work, take as needed: `KHR_materials_ior`, `_specular`, `_clearcoat`, then `_sheen`.
- Defer: `_transmission` and `_volume` (they need a scene-color pass), `_iridescence`, `_dispersion`, `KHR_interactivity`, `KHR_gaussian_splatting`. Skip `KHR_draco_mesh_compression` unless users demand it, because I found no Go decoder and meshopt covers the same need.

### Gaps
- `model-index.json` and tag metadata in the sample repo were not visible on the page.
- `EXT_texture_webp` status was not directly confirmed on the extensions page (the Sample Renderer lists WebP support).
- No Go meshopt or Draco decoder was searched for specifically. This is a follow-up item.
- A spec-level list of common loader mistakes (sRGB handling, tangent generation, skinning order) was not researched beyond what the test models imply.

## Which noise and generation techniques give the best results: FastNoise Lite, OpenSimplex2 and licensing, domain warping, Voronoi, Red Blob Games maps, rivers, erosion?

### Takeaway
Vendor FastNoise Lite's single-file Go port (MIT) for noise, cellular, and domain warp. Licensing is clean. For the visual jump, prioritize erosion and river structure over more noise types: one comparison found fBm, warp, and ridged all look plausible but unlike real terrain because they lack erosion.

### Cited Findings
**FastNoise Lite and OpenSimplex2**
- FastNoise Lite is MIT, has a Go port, and offers OpenSimplex2, OpenSimplex2S, Cellular (Voronoi), Perlin, Value, and Value Cubic in 2D and 3D, with fractal options for all noise types and two domain warps (OpenSimplex2-based and basic grid gradient) — [FastNoiseLite](https://github.com/Auburn/FastNoiseLite)
- Go port: `Go/` contains `README.md` and `fastnoise.go`, with no `go.mod` visible. It has a generic constructor (`fastnoise.New[float32]()`, float32 or float64), `noise.NoiseType(fastnoise.OpenSimplex2)`, and `noise.Noise2D(x, y)` — [FastNoiseLite/Go](https://github.com/Auburn/FastNoiseLite/tree/master/Go)
- OpenSimplex2: the root LICENSE is CC0 1.0 — [LICENSE](https://raw.githubusercontent.com/KdotJPG/OpenSimplex2/master/LICENSE). The README says "the majority of files" are CC0 with some directories under UNLICENSE, and states the author's belief that no patent claims cover the code, flagged as not legal advice — [OpenSimplex2](https://github.com/KdotJPG/OpenSimplex2)
- Variants: 2(F) looks closest to Simplex and is about as fast as common Simplex implementations, and its 4D looks dotty so it suits fBm. 2S has larger contribution ranges and better 3D/4D uniformity, and the README recommends it for ridged noise. It was formerly called SuperSimplex — [OpenSimplex2](https://github.com/KdotJPG/OpenSimplex2)

**Domain warping**
- IQ: replace f(p) with f(g(p)). Two-stage: q from fbm at two offsets, then `fbm(p + 4.0*q)`. Three-stage nests a second field r. Offsets such as (5.2, 1.3) only decorrelate samples — [Quilez, warp](https://iquilezles.org/articles/warp). I read this through a search summary, because the direct fetch failed with a 404 and then a redirect.

**Red Blob Games maps**
- Use two noise maps with separate seeds (elevation, moisture) and a Whittaker-style biome lookup table — [Red Blob terrain](https://www.redblobgames.com/maps/terrain-from-noise/)
- Octaves: offset each octave and divide by total amplitude. Redistribution: raise normalized elevation to a power — [Red Blob terrain](https://www.redblobgames.com/maps/terrain-from-noise/)
- Island: `lerp(e, 1-d, mix)` with square bump `d = 1-(1-nx²)(1-ny²)` or Euclidean `d = min(1,(nx²+ny²)/√2)` — [Red Blob terrain](https://www.redblobgames.com/maps/terrain-from-noise/)
- Wrapping: map x onto a circle and sample 3D noise (cylinder) or 4D (torus); higher-dimensional noise has a narrower value range, so it may need rescaling. Ridged: `2 × (0.5 − |0.5 − noise|)`, optionally weighting high octaves by lower ones — [Red Blob terrain](https://www.redblobgames.com/maps/terrain-from-noise/)
- Placement: local maxima of blue noise within radius R, varying R by biome; Poisson disc or jittered grids are more efficient and better distributed — [Red Blob terrain](https://www.redblobgames.com/maps/terrain-from-noise/)
- mapgen4 sits on dual Delaunay/Voronoi meshes and simulates evaporation, wind, and rainfall to drive biomes and rivers. Output is free to use, with credit appreciated — [mapgen4](https://www.redblobgames.com/maps/mapgen4/). Its source license was not stated on that page.

**Erosion and rivers**
- `dandrino/terrain-erosion-3-ways` (MIT) finds plain fBm, warping, and ridge noise plausible but unlike real terrain because they lack erosion. It compares: simulation (about 10 constants, roughly O(N³), parallelizable, tendril ridges and flat valleys), ML (about 8 days of training, little control), and river networks first (O(N² log N), convincing ridges, needs extra work for deposition) — [terrain-erosion-3-ways](https://github.com/dandrino/terrain-erosion-3-ways)
- Sebastian Lague's Unity project is MIT, shows 70k erosion iterations, and lists a firespark.de PDF and a ranmantaru.com post as resources — [Hydraulic-Erosion](https://github.com/SebLague/Hydraulic-Erosion)
- Droplet erosion: droplets start at random points and move downhill, picking up sediment where they speed up and depositing where they slow — [ctkrug/erosion](https://github.com/ctkrug/erosion). Secondary source.
- D8 flow accumulation: give each cell one unit of rain, process high to low, pass water to the lowest neighbor, and mark cells above a threshold as river. Pits trap water, so handle them first — [Cinevva blog](https://app.cinevva.com/blog/2026-05-13-open-world-browser-part-27-island-and-terrain). Secondary source, which attributes the method to Red Blob.
- One Rust library runs fractal generation, then thermal, then hydraulic erosion — [symbios-ground](https://docs.rs/symbios-ground). A map tool reports 100k-200k particles followed by a D8 river step — [CS2_MapCreator](https://github.com/gregspruce/CS2_MapCreator). A GPU compute version reports about one million droplets in roughly 10 seconds — [10Kaiser10/hydraulic-erosion](https://github.com/10Kaiser10/hydraulic-erosion).
- Rune Skovbo Johansen's "Advanced Terrain Erosion Filter" overlays multi-octave gully patterns aligned to steepest descent, building on Fewes, Clay John, and IQ's derivative fBm. It approximates erosion analytically without simulation — [bevy_erosion_filter](https://docs.rs/crate/bevy_erosion_filter/latest/source/README.md). This is a port's description of a 2026 work; I did not read the original.

### Inferences
- Vendor `fastnoise.go` into an internal package rather than importing it, because there appears to be no module path. Keep the MIT notice. Check that its generics work with the project's Go version (it is the newest syntax involved).
- Expose per-seed `NoiseType`, fractal type, and warp in moonBASIC commands. Keep the existing fBm and ridged code if output stability matters, because swapping the base noise changes every seed's terrain.
- Use 2(F) for general fBm and 2S for ridged, per the OpenSimplex2 README.
- A good default pipeline that combines these:
  1. fBm or ridged base, with light domain warp.
  2. Island mask (optional).
  3. Thermal pass.
  4. Droplet hydraulic pass (the engine has erosion; the listed order is thermal first).
  5. Pit fill.
  6. D8 flow accumulation for river cells, carving and a river mask.
  7. Moisture noise combined with distance to river, into a biome table.
  8. Blue-noise placement with radius per biome.
- Step 5-6 is the highest-value addition, since river data also gives gameplay and splat-map inputs. The Go code is small. It is also deterministic and runs on the CPU, which suits a language with no compute shaders on GL 3.3. GPU compute erosion is not available on a GL 3.3 context, so stay on the CPU with worker goroutines (`jobs.go` exists, repo), or fragment-shader ping-pong for grid-based simulation.
- Offer an analytical erosion filter later as a real-time option for streamed terrain, because droplet simulation depends on neighbors and does not work per chunk without seams. Read the original first.
- Voronoi uses: cell regions for biome boundaries, cracked-rock and scale textures via cellular fractal, warped for organic shapes. FastNoise Lite covers these. The mapgen4 mesh approach is a larger design and is not needed for a heightmap engine.

### Gaps
- Exact fractal modes and cellular options in FastNoise Lite were not listed on the fetched page. From memory: FBm, Ridged, PingPong, and domain-warp progressive and independent, with distance and return-type options. Check `fastnoise.go`.
- Simplex patent status: no source found. I recall US 6867776 having expired around 2022. Using OpenSimplex2 avoids depending on this.
- Red Blob's own river write-ups (mapgen2, mapgen4 posts) were not read directly. D8 details are second-hand.
- Lague's parameter values were not on the README, and his source code was not read.
- No side-by-side visual quality study was found, so "best" is judgment plus the dandrino comparison.

## Debugging: how do you use RenderDoc with OpenGL 3.3 (core context requirements), GL debug groups and KHR_debug, GPU timer queries, and vendor profilers?

### Takeaway
RenderDoc works with moonBASIC's 3.3 core context on Windows and Linux, not macOS, and it needs only render-thread discipline and labels. The Go GL binding already exposes KHR_debug functions, but they must be nil-checked at runtime. Add a debug-context flag, debug groups, object labels, and a RenderDoc hook. The engine already has timer queries; use a ring of them. Nsight Graphics is the vendor option, and AMD's RGP will not work.

### Cited Findings
**RenderDoc**
- Capture supports core profile 3.2 through 4.6 only. Compatibility is not generally supported, though capture is attempted and replay works only for a supported subset. Concessions: VAO 0 and luminance/intensity formats are allowed — [RenderDoc OpenGL support](https://renderdoc.org/docs/behind_scenes/opengl_support.html)
- Older docs said 3.2 through 4.5 — [RenderDoc v1.0 docs](https://github.com/baldurk/renderdoc/blob/v1.0/docs/behind_scenes/opengl_support.rst). Use the current page.
- An older wiki says RenderDoc refuses to capture if the app does not use CreateContextAttribs — [wiki mirror](https://github.com/James-Ryan/renderdoc/wiki/OpenGL)
- All frame-rendering GL commands must come from one thread, except possibly SwapBuffers. Other threads are fine during loading or initialization, but interleaved rendering from several threads during capture is not supported — [RenderDoc OpenGL support](https://renderdoc.org/docs/behind_scenes/opengl_support.html)
- macOS is not supported. Vendor-specific extensions are generally unsupported — [RenderDoc OpenGL support](https://renderdoc.org/docs/behind_scenes/opengl_support.html)
- Some analysis is disabled without `GL_ARB_shader_image_load_store`, `GL_ARB_compute_shader`, `ARB_shader_storage_buffer_object`, and `GL_ARB_gpu_shader5`. Shader debugging needs compute and SSBO at minimum, covers vertex, pixel, and compute shaders, and works best with shaders that compile to SPIR-V unmodified and with explicit layouts and bindings. Some drivers may not offer it at all — [RenderDoc OpenGL support](https://renderdoc.org/docs/behind_scenes/opengl_support.html)
- Annotation: `glPushDebugGroup`/`glPopDebugGroup` regions appear in the Event Browser. `glObjectLabel` names resources wherever they are bound, and each resource has one name per capture (the most recently set). The page says KHR_debug "is guaranteed to always be available" (under RenderDoc) — [RenderDoc annotation](https://renderdoc.org/docs/how/how_annotate_capture.html)
- In-application API: do not link. Check whether `renderdoc.dll` is loaded (`GetModuleHandle`) or `librenderdoc.so` (`dlopen` with `RTLD_NOW | RTLD_NOLOAD`), then call `RENDERDOC_GetAPI` from `renderdoc_app.h`. Provides `StartFrameCapture`, `EndFrameCapture`, `TriggerCapture`, and `TriggerMultiFrameCapture(n)`. Passing NULL lets RenderDoc pick the device or window — [RenderDoc in-app API](https://renderdoc.org/docs/in_application_api.html)

**KHR_debug**
- Requires only OpenGL 1.1 and was written to back-port GL 4.3 functionality. Without the debug context flag, the GL may generate no messages, initial `DEBUG_OUTPUT` is FALSE, and enabling it gives implementation-defined output, possibly none. The functions still work without error — [KHR_debug spec](https://registry.khronos.org/OpenGL/extensions/KHR/KHR_debug.txt)
- `DEBUG_OUTPUT_SYNCHRONOUS` makes callbacks arrive on the context's thread before the command returns. Otherwise callbacks may come from other threads or later, so the app must handle thread safety — [KHR_debug spec](https://registry.khronos.org/OpenGL/extensions/KHR/KHR_debug.txt)

**Timer queries**
- `TIME_ELAPSED` measures how long previous commands take to fully complete, in nanoseconds. `QueryCounter(id, TIMESTAMP)` records the time after prior commands are realized and can be used inside a `TIME_ELAPSED` block without affecting it — [ARB_timer_query](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_timer_query.txt)
- Two timer queries cannot run simultaneously, which rules out nested `TIME_ELAPSED`. Use timestamps instead. Reusing an id before reading it discards the earlier result. `QUERY_RESULT_AVAILABLE` returns FALSE if reading would wait, and if one query is available all earlier same-type queries are too. `QUERY_COUNTER_BITS` of zero means no useful data; nonzero is at least 30 bits, about one second — [ARB_timer_query](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_timer_query.txt)

**Vendor profilers**
- NVIDIA Nsight Graphics: the Frame Debugger supports operations defined by the GL 4.5 core profile, and a core context is not required — [Nsight user guide appendix](https://docs.nvidia.com/nsight-graphics/UserGuide/appendix.html). OpenGL range profiling was extended to Turing GPUs in 2019.4 — [Nsight release notes](https://docs.nvidia.com/nsight-graphics/2019.4/ReleaseNotes/index.html). The release notes are old; check current ones.
- AMD Radeon GPU Profiler supports DX12, Vulkan, OpenCL, and HIP, and "will NOT work" with DX11 or OpenGL — [RGP manual](https://gpuopen.com/manuals/rgp_manual/)

**Repo state (repo)**
- Rendering imports `github.com/go-gl/gl/v3.3-core/gl`. `extra_window.go` sets `ContextVersionMajor 3`, `ContextVersionMinor 3`, and `OpenGLCoreProfile`. I read only this window path, not the main window's hints.
- The v3.3-core binding has `DebugMessageCallback`, `PushDebugGroup`, and `ObjectLabel` (plus ARB and KHR variants), loaded through `getProcAddr`. So no binding change is needed, but these pointers are nil when the driver lacks the function.
- `profiler.go` uses `gl.QueryCounter(..., gl.TIMESTAMP)`, and `particles_gpu.go` uses `TIME_ELAPSED`.
- A search of `internal/` found no `PushDebugGroup`, `DebugMessageCallback`, or `ObjectLabel` use.

### Inferences
**Concrete additions, in order**
1. A `MOON_GL_DEBUG=1` environment variable or `--gl-debug` flag sets `glfw.OpenGLDebugContext` before window creation. If `KHR_debug` is in the extension string, enable `DEBUG_OUTPUT` and `DEBUG_OUTPUT_SYNCHRONOUS` and install a callback that logs by severity and filters notifications. In Go, synchronous mode is the safe choice because of the thread rule above. The callback passes through cgo, so keep it off in release builds.
2. A `glDebugGroup(name)` helper around each render pass (shadow, g-buffer or forward, post FX, UI), gated on a non-nil check. Use the same names as the CPU and GPU profiler labels so that captures and profiler output line up.
3. `gl.ObjectLabel` at creation of textures, FBOs, programs, and meshes, using asset paths. This makes captures readable at little cost.
4. A RenderDoc hook. On Windows, detect `renderdoc.dll` with `GetModuleHandle`. Calling the function table needs cgo or `syscall.SyscallN`. Expose `moon run --capture-frame N` or a moonBASIC debug command through `TriggerCapture`. Do this last, because F12 and PrintScreen already work when launched from the RenderDoc UI.
5. Check that the frame path stays on one thread. `jobs.go` should not touch GL during frames, and asset hot reload should upload from the main thread.

**Timer query usage**
- Keep a ring of 3-4 frames per query set.
- Read only when `QUERY_RESULT_AVAILABLE` is true, and never block.
- Use `QueryCounter` pairs rather than `TIME_ELAPSED` wherever passes nest. `particles_gpu.go`'s `TIME_ELAPSED` block cannot be open while another `TIME_ELAPSED` is, so a future pass-level `TIME_ELAPSED` around particles would produce `INVALID_OPERATION`. Standardize on timestamps.
- GPU timings are only meaningful in a release driver state, so flag them as unreliable when a debug context is on.

**Tool recommendation by platform**
- Windows and Linux with any GPU: RenderDoc.
- NVIDIA hardware counters: Nsight Graphics.
- AMD: RenderDoc plus timer queries only, since RGP does not apply.
- macOS: RenderDoc is unavailable. Rely on the engine's own overlay and reproduce GPU bugs on another platform.
- A 3.3-only driver path may disable RenderDoc shader debugging. Expect it to depend on whether the driver also exposes compute and SSBO extensions in the 3.3 context, and use explicit `layout(location)`/binding in shaders to help.

### Gaps
- Exact current RenderDoc wording on `CreateContextAttribs` comes from an old wiki mirror. The current support page does not mention it or forward-compatible contexts. GLFW's core-profile path should satisfy it, which I did not test.
- Whether the macOS GL 4.1 driver exposes KHR_debug was not sourced. I recall it does not, so nil-check rather than assume.
- `ARB_timer_query` promotion to core in 3.3 is not stated in the extension text. The go-gl v3.3-core usage in the repo is consistent with it. Check `QUERY_COUNTER_BITS` at startup anyway.
- Intel GPA and its OpenGL status: no results. AMD's other tools (for example CodeXL, Radeon Developer Tool Suite) for OpenGL were not covered. Linux `apitrace` and Tracy's GL zones were not researched and could be a useful follow-up.
- `renderdoc_app.h` license is not on the fetched page. I recall MIT; verify before vendoring.
- Whether RenderDoc works with go-gl's `getProcAddr` hook path was not sourced. Since it hooks GL at the driver-loader level this is likely fine, but it has not been tested here.
