# three.js advanced rendering techniques, and whether each works in GLSL 330 on OpenGL 3.3 without compute

Source baseline: three.js `dev` at commit `7e26babb` (committed 2026-11-04), one release after r187 (published 2026-10-30, per the GitHub API). All file links are pinned to that commit. PR links are live.

How to read this:
- **Cited Findings** describe what three.js does, from source or PR text.
- **Verdicts are my inference.** three.js does not target GL 3.3. Each verdict maps the source's GPU features onto the GL 3.3 limits in the first section. They sit under Inferences for that reason.
- Verdict labels: **FRAG** = plain fragment passes in GLSL 330. **FRAG+** = fragment passes with a named workaround for a post-330 feature. **BAKE** = runtime is fragment-only, but the bake needs a separate approach. **COMPUTE** = the three.js method needs compute, storage buffers or atomics.
- Costs are only what a source measured. Almost none were measured on ten-year-old hardware. I give no estimates of my own.
- I read SSGI, GTAO, TRAA, Denoise, shadow filters, tone mapping and the Batch accessor in full or nearly so. For the largest files (MeshletDrawingPipeline at 190 KB, SSRNode, TemporalReprojectNode, RecurrentDenoiseNode, VXGIVolume, SMAANode) I read doc comments, inline comments and targeted greps, not every line.

## What GL 3.3 / GLSL 330 rules out, and what three.js's own WebGL path shows

### Takeaway
Three features recur as blockers: compute and storage buffers (GL 4.3), `textureGather` and `bitCount` (GLSL 4.00), and indirect draws (GL 4.0). The last two groups have simple workarounds. three.js's WebGL2 fallback already runs "compute" through transform feedback and textures, which is direct precedent, because WebGL2 is a weaker API than GL 3.3 in the respects that matter here.

### Cited Findings
- `glDispatchCompute` first appears in GL 4.3 — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glDispatchCompute.xhtml)
- `glDrawArraysIndirect` first appears in GL 4.0 — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glDrawArraysIndirect.xhtml); `glMultiDrawElementsIndirect` in GL 4.3 — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glMultiDrawElementsIndirect.xhtml)
- `glDrawTransformFeedback` first appears in GL 4.0, so a draw count cannot come directly from a transform feedback object on 3.3 — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glDrawTransformFeedback.xhtml)
- `imageStore` needs GLSL 4.20 — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/imageStore.xhtml)
- `bitCount` needs GLSL 4.00 — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/bitCount.xhtml); `textureGather` needs GLSL 4.00 — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/textureGather.xhtml)
- `glCopyImageSubData` needs GL 4.3, so a raw texture-to-texture copy needs another route such as an FBO blit — [Khronos refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glCopyImageSubData.xhtml)
- Available on 3.3: `glVertexAttribDivisor` (3.3) — [refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glVertexAttribDivisor.xhtml); `glTexBuffer` (3.1) — [refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glTexBuffer.xhtml); `glMultiDrawElementsBaseVertex` (3.2 per the table; the page's Notes say 3.1, an inconsistency in the page itself) — [refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glMultiDrawElementsBaseVertex.xhtml); `GL_ANY_SAMPLES_PASSED` (3.3) — [refpage](https://registry.khronos.org/OpenGL-Refpages/gl4/html/glBeginQuery.xhtml)
- three.js's WebGL2 backend implements `compute()` as `gl.beginTransformFeedback( gl.POINTS )` plus `gl.drawArraysInstanced( gl.POINTS, 0, 1, count )`, and warns that indirect-count dispatch is unsupported — [WebGLBackend.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/renderers/webgl-fallback/WebGLBackend.js)
- That backend reads storage buffers in the fragment stage by wrapping them as Pixel Buffer Object textures fetched with `texelFetch` — [GLSLNodeBuilder.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/renderers/webgl-fallback/nodes/GLSLNodeBuilder.js)
- A transform-feedback compute "can only write their own element", which limits what can be ported to that model — [PR #34978](https://github.com/mrdoob/three.js/pull/34978)
- The WebGPU-only addons state it in their docs. VXGI: "can only be used with `WebGPURenderer` and a WebGPU backend" — [VXGINode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lighting/vxgi/VXGINode.js)

### Inferences
- Bit operations and `uint` are in GLSL 330, so only `bitCount` itself needs replacing, with a short SWAR popcount. `ARB_gpu_shader5` could be used where present, but cannot be assumed.
- `textureGather` on depth can be replaced with four `texelFetch` calls.
- Transform feedback is core on 3.3, and 3.3 has geometry shaders, which WebGL2 lacks. A geometry shader can drop primitives, so GL 3.3 can append-filter in a way the WebGL2 fallback cannot. The count must then come from a `GL_TRANSFORM_FEEDBACK_PRIMITIVES_WRITTEN` query, which stalls or lags a frame.

### Gaps
- I could not confirm the minimum versions for SSBOs and `gl_DrawID` from a primary page: the Khronos wiki returned 403 and the `gl_DrawID` refpage 404. From memory, SSBOs are GL 4.3 and `gl_DrawID` is GL 4.6 or `ARB_shader_draw_parameters`. Treat both as unverified.
- No driver-coverage data was gathered for extensions on ten-year-old GPUs.

## Screen-space techniques

### Takeaway
Every screen-space node in three.js is a chain of fullscreen fragment passes over depth, normals, colour and velocity, with no compute. SSGI, GTAO, TRAA, TAAU, SMAA, FXAA, DoF, motion blur, bloom, flares and god rays all port to GLSL 330, needing at most a popcount and a gather replacement. The real cost for a forward renderer is that many of them require a normal, velocity or metal-rough MRT or prepass.

### Cited Findings

**SSGI (`SSGINode`)**
- Method: visibility-bitmask horizon sampling, citing SSRT3 and Cdrin's SSILVB slides — [SSGINode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSGINode.js). The underlying paper is Therrien, Levesque and Gilet, replacing two horizon angles with a bit field of hemisphere-slice sectors — [arXiv 2301.11376](https://arxiv.org/abs/2301.11376)
- Per pixel: for each slice, march both directions in screen space with exponential spacing, turn front and back horizon angles into a 32-bit sector mask, count newly occluded sectors with `countOneBits`, and add `beauty * NdotL * lightNdotL * sectors/32`. AO is the total occluded fraction — [SSGINode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSGINode.js)
- Passes and targets: one fullscreen pass into one 2-attachment target, AO in R8 and GI in R11G11B10F. Inputs are beauty, depth and optional normals (reconstructed from depth if absent) — [SSGINode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSGINode.js)
- Samples per pixel = `sliceCount * stepCount * 2`. Presets with temporal filtering: Low 1×12, Medium 2×8, High 3×16. Without: Low 2×6, Medium 3×8, High 4×12 — [SSGINode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSGINode.js)
- Temporal: cycles six rotations and four offsets from the Activision GTAO talk, noise tuned for TRAA convergence. `depthNode.gather()` is used only when `resolutionScale < 1` — [SSGINode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSGINode.js)
- The only stated hardware requirement is the WebGPU feature `rg11b10ufloat-renderable` — [SSGINode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSGINode.js)
- Reported cost: the paper's abstract claims only "low performance overhead", with no numbers on that page — [arXiv 2301.11376](https://arxiv.org/abs/2301.11376)

**SSR (`SSRNode`)**
- Method: reference is Lettier's screen-space reflection tutorial. Linear march clipped to the screen, up to `MAX_STEPS = 64` scaled by `quality`, with an exponential step remap and optional binary refinement — [SSRNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSRNode.js)
- Two modes. Non-stochastic: mirror ray, then a roughness-indexed blur chain (base mip copied, then `boxBlur` per level over 5 levels). Stochastic: GGX-sampled rays with analytic noise, intended to feed temporal reprojection and denoising, with optional multi-bounce from the previous frame — [SSRNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSRNode.js)
- Targets: an SSR half-float target and a mipped half-float blur target. Inputs are colour, depth, normal, metalness and roughness — [SSRNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSRNode.js)
- A rewrite is open and not merged: coverage-based resolve with an ordered fallback (SSR, then PMREM environment), half-resolution tracing and denoising. The author reports 226 fps, about 4.4 ms, at 1920×1080 with 1 ray per tracing pixel — [PR #35228](https://github.com/mrdoob/three.js/pull/35228)
- The older `SSRPass` remains for `WebGLRenderer` — [SSRPass.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/postprocessing/SSRPass.js)

**GTAO (`GTAONode`)**
- Method: Jimenez et al. horizon-based slices. `samples` defaults to 16, split into 3 directions below 30 samples and 5 above; step jitter comes from interleaved gradient noise; a magic-square noise texture rotates the kernel — [GTAONode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/GTAONode.js)
- Passes and targets: one fullscreen pass into one single-channel target — [GTAONode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/GTAONode.js)
- A GLSL version also exists: `GTAOPass` with its shader and Poisson denoise shader — [GTAOPass.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/postprocessing/GTAOPass.js), [GTAOShader.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/shaders/GTAOShader.js), [PoissonDenoiseShader.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/shaders/PoissonDenoiseShader.js)
- `SSAONode` is the cheaper tier: a per-pixel rotated Vogel disk with one depth tap per sample and a built-in separable depth-aware blur — [SSAONode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSAONode.js)
- Measured (authors' own): about 0.5 ms on PS4 at 1080p for a half-resolution occlusion buffer — [Jimenez et al., via search of the Activision paper](https://www.activision.com/cdn/research/Practical_Real_Time_Strategies_for_Accurate_Indirect_Occlusion_NEW%20VERSION_COLOR.pdf). My direct fetch of the PDF failed, so this comes from a search extract.

**AO and GI denoising**
- `DenoiseNode`: one pass, 16 Poisson-ring samples rotated by a 64×64 noise texture, weighted by luma, plane distance and normal similarity. The docs warn it adds "quite some overhead" and should be optional — [DenoiseNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/DenoiseNode.js)
- `TemporalReprojectNode`: 4-tap bilinear history with plane and normal confidence, YCoCg variance clipping, and a reprojection-stretch test from screen-space derivatives. Specular mode adds hit-point parallax — [TemporalReprojectNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/TemporalReprojectNode.js)
- `RecurrentDenoiseNode`: 8-tap Vogel disk with diffuse or specular kernels, fed back as history. Its comment on replacing `acos` with one `exp` shows a focus on per-tap cost — [RecurrentDenoiseNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/RecurrentDenoiseNode.js)

**TRAA, TAAU**
- `TRAANode`: one resolve pass. Halton projection jitter, closest-depth velocity lookup, disocclusion test against previous depth, variance clipping from a 3×3 neighbourhood (Salvi), flicker reduction. Current-frame weight is 0.05 plus subpixel correction up to 0.25 plus motion. MSAA must be off — [TRAANode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/TRAANode.js)
- Targets: history and resolve (half float, swapped each frame) plus previous depth. Input velocity comes from a scene MRT — [TRAANode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/TRAANode.js)
- `TAAUNode` renders input at low resolution and resolves with a 9-tap Blackman-Harris filter, plus a separate lock channel in a 2-attachment history. Cites Karis 2014 and FSR2. The doc calls it an alternative to FSR 2/3 in a single pass — [TAAUNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/TAAUNode.js)
- `TAARenderPass` is the older accumulation approach — [TAARenderPass.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/postprocessing/TAARenderPass.js)

**SMAA, FXAA, FSR1**
- `SMAANode`: SMAA 1x Medium with colour edges. Three passes (edges, weights, blend), with embedded area and search textures. Must run before sRGB conversion — [SMAANode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SMAANode.js)
- `FXAANode`: one pass, requires sRGB input, so tone mapping must come first — [FXAANode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/FXAANode.js)
- Measured (paper): SMAA T2x 1.3 ms and SMAA 4x 2.6 ms on a GTX 470 at 1080p, against 5.4 ms average for MSAA 8x. The authors warn some timings are dated — [SMAA paper, via search extract](https://www.iryoku.com/smaa/downloads/SMAA-Enhanced-Subpixel-Morphological-Antialiasing.pdf)
- `FSR1Node`: two passes, EASU (12 taps) and RCAS (5 taps). A source comment says it avoids `textureGather` because the WebGL 2 backend lacks it, which is direct precedent for the same 330 port — [FSR1Node.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/FSR1Node.js)

**DoF, motion blur, bloom**
- `DepthOfFieldNode`: six passes. CoC at full resolution (2 attachments), blurred CoC, 64-tap Vogel blur, 16-tap near and far blurs (all half resolution), composite at full resolution. Cites Doom 2016 — [DepthOfFieldNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/DepthOfFieldNode.js)
- `motionBlur`: a single function taking N taps (default 16) along per-pixel velocity. No tile or neighbour-max pass — [MotionBlur.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/MotionBlur.js)
- `BloomNode`: luminance high-pass, then 5 mips of separable blur (H and V targets per mip) in half float, then composite. Bilinear tap merging per PR 31528. Selective bloom uses an emissive MRT — [BloomNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/BloomNode.js)

**Lens flare, god rays**
- `LensflareNode`: Chapman pseudo lens flare, built from the bloom result, with ghosts and a default downsample of 4 — [LensflareNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/LensflareNode.js)
- `Lensflare` object: sprite elements with visibility from a 16×16 framebuffer copy — [Lensflare.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/objects/Lensflare.js)
- `GodraysNode`: from three-good-godrays. Half-resolution screen-space march, 60 steps by default, shadow map compared at each step, noise from IGN. Then an optional bilateral blur and `depthAwareBlend`. Point and directional lights only, and it requires real shadows — [GodraysNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/GodraysNode.js)

**Film, outline, retro, stylised**
- `FilmNode`: one function, adding grain from `rand( fract( uv + time ) )` — [FilmNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/FilmNode.js)
- `OutlineNode`: depth of non-selected objects, mask of selected, downsample, edge detection, two blur levels, composite, using 8 targets. It re-renders the selected objects — [OutlineNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/OutlineNode.js)
- `RetroPassNode`: PS1 look through vertex snapping, affine UVs, 0.25 resolution scale and nearest filtering. Combine with Bayer dither and posterize — [RetroPassNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/RetroPassNode.js)
- `OITPassNode`: weighted blended OIT (McGuire and Bavoil) with 2 accumulation targets sharing opaque depth. The WebGL backend loses MSAA in this pass — [OITPassNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/OITPassNode.js)
- `SSSNode`: screen-space contact shadows for one directional light, stored in R8, citing Bend Studio — [SSSNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SSSNode.js)

### Inferences

| Technique | Verdict | Dependency in 330 |
|---|---|---|
| SSGI | FRAG+ | Replace `countOneBits` with SWAR. Replace `gather` with 4 `texelFetch`, or render at full scale. R11F_G11F_B10F MRT needed. Needs normals or depth reconstruction, and temporal filtering for its low presets. |
| SSR | FRAG | Needs normal and metal-rough MRT. HDR environment CDF importance sampling is optional. |
| GTAO, SSAO, Denoise | FRAG | Normals optional via reconstruction. GLSL version already exists in `GTAOPass`. |
| TemporalReproject, RecurrentDenoise | FRAG | Needs velocity and previous-frame depth and normals. |
| TRAA, TAAU | FRAG+ | Previous depth by FBO blit or swap, not `copyImageSubData`. Needs velocity MRT and projection jitter. |
| SMAA, FXAA, FSR1 | FRAG | Port with no feature changes. |
| DoF, motion blur, bloom, flare, god rays, film, outline, retro, OIT, SSS | FRAG | God rays depend on shadow-map comparison samplers, which are core. |

- A forward renderer gets colour and depth for free. Normal, velocity and material MRT outputs must be added to the forward pass or a prepass. That is the main structural cost of SSGI, SSR and TRAA.

### Gaps
- I found no measured cost for three.js's SSGI, GTAO, TRAA, DoF or bloom nodes on any hardware. PR searches gave no timings.
- SMAA 1x has no timing in what I could read. The PDF's full table was not read.
- `FXAANode` carries no version or citation in its header, so I can't say which FXAA revision it follows.
- [PR #35228](https://github.com/mrdoob/three.js/pull/35228) is unmerged and its hardware was not in the excerpt I read.

## Lighting and shadows

### Takeaway
Shadow filtering, CSM, tiled shadows, LTC, volumetrics and probe sampling are fragment-only at runtime. Clustered lighting uses compute in WebGPU, but three.js runs it on WebGL2 with transform feedback, and CPU assignment is an alternative. The new GI work splits cleanly: `LightProbeGridWebGL` needs no compute, while VXGI and ray-traced radiosity are compute-only.

### Cited Findings

**CSM**
- `CSMShadowNode`: N directional lights, 3 cascades by default, with practical, uniform, logarithmic or custom splits and optional fade with an expanded shadow margin — [CSMShadowNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/csm/CSMShadowNode.js)
- `CSM` with `CSMShader` does the same through GLSL shader chunks for `WebGLRenderer` — [CSM.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/csm/CSM.js), [CSMShader.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/csm/CSMShader.js)
- New: `SunLight` and `SunLightShadow`, a fixed 2-cascade fitted CSM at 1024² per cascade with 10% blend, and no target. The docs say it does not suit small scenes — [SunLight.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lights/SunLight.js), [SunLightShadow.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lights/SunLightShadow.js)
- It also has WebGL shader-chunk support via `sampler2DShadow sunShadowMap` — [shadowmap_pars_fragment.glsl.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/renderers/shaders/ShaderChunk/shadowmap_pars_fragment.glsl.js)

**Tiled shadows**
- `TileShadowNode`: splits one shadow camera into a grid of tiles in an array depth texture, rendered through an `ArrayCamera`. A fragment's position in the full shadow camera selects its tile and coordinate. VSM is not supported — [TileShadowNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/shadows/TileShadowNode.js)
- Measured: 3840×2160 fullscreen receiver, PCF, 1024² tiles, shadow map rendered once, MacBook Air M1. Plain shadow map 6.76 ms. After the fix: 2×2 7.06 ms, 4×4 7.06 ms, 8×8 7.11 ms. Before: 8.71, 16.02 and 169.68 ms — [PR #34787](https://github.com/mrdoob/three.js/pull/34787)

**Filtering**
- PCF: 5 Vogel disk samples rotated per pixel by IGN, with hardware depth comparison — [ShadowFilterNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/lighting/ShadowFilterNode.js)
- `PCFSoftShadowMap` is deprecated since r186 in favour of `PCFShadowMap` — [constants.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/constants.js)
- VSM: separate vertical and horizontal blur passes writing mean and standard deviation to an RG half-float target, then Chebyshev's bound at lookup, with light-bleed reduction remapping `(p - 0.3) / 0.65` — [ShadowFilterNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/lighting/ShadowFilterNode.js), [ShadowNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/lighting/ShadowNode.js)
- With VSM, all shadow receivers also cast — [constants.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/constants.js)
- PCSS is not a renderer feature. It exists only as an example that patches shader chunks: 17-sample Poisson blocker search, parallel-plane penumbra estimate, then PCF — [webgl_shadowmap_pcss.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgl_shadowmap_pcss.html)

**Many lights**
- `ClusteredLightsNode`: X×Y tiles times exponential Z slices. Lights are CPU-sorted by view depth with per-slice ranges for Z culling, then a compute pass writes per-cluster light indices. Shadowed lights, `colorNode` lights and projected maps stay on the default path — [ClusteredLightsNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/lighting/ClusteredLightsNode.js)
- It runs on WebGL2 through transform feedback and a PBO-backed index buffer, writing one `ivec4` chunk per invocation — [PR #34978](https://github.com/mrdoob/three.js/pull/34978), [ClusteredLightsNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/lighting/ClusteredLightsNode.js)
- Measured: 1,024 point plus 512 spot lights at 708 fps (1.41 ms) at 1280×720. The rebuild costs about 0.33 ms CPU and about 0.14 ms GPU. Hardware not stated in the excerpt I read — [PR #34978](https://github.com/mrdoob/three.js/pull/34978)
- `TiledLighting` was removed in r185 to focus on `ClusteredLighting` — [PR #33751](https://github.com/mrdoob/three.js/pull/33751)
- `DynamicLighting` batches lights into uniform arrays so count changes don't recompile materials, with no culling — [DynamicLighting.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lighting/DynamicLighting.js)
- An open PR adds a world-space grid for lights marked static, using read-only storage buffers. It is about 6% faster at 720p and about 7% slower at 1440p DPR 2 than clustered on the same scene — [PR #34931](https://github.com/mrdoob/three.js/pull/34931), [PR #34978](https://github.com/mrdoob/three.js/pull/34978)

**Probes and baked GI**
- `LightProbeGridWebGL`: a 3D grid of L2 SH irradiance probes. Bake renders a cube map at each probe (default 8² per face), projects to 9 SH coefficients in a fragment pass into a batch target, then repacks into a single RGBA 3D target holding 7 sub-volumes with padding slices. Extra passes add bounces. No CPU readback — [LightProbeGridWebGL.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lighting/LightProbeGridWebGL.js)
- Runtime: one 3D-atlas lookup per sub-volume (trilinear), with a normal offset of half a probe spacing — [LightProbeGridNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/lighting/LightProbeGridNode.js)
- `LightProbeNode` remains for a single probe — [LightProbeNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/lighting/LightProbeNode.js)

**New GI**
- `VXGINode` with `VXGIVolume`: dense mip chain variant of Crassin 2011. Voxelization uses conservative rasterization along the dominant axis in a compute shader, radiance is injected per voxel from shadow maps, and cones are traced per pixel. Docs position it between SSGI and probes: objects should stay static — [VXGINode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lighting/vxgi/VXGINode.js), [VXGIVolume.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lighting/vxgi/VXGIVolume.js)
- `RayTracedRadiosity` with `LightMapper`: BVH rays via three-mesh-bvh, baked to light maps. Requires a WebGPU backend — [RayTracedRadiosity.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lighting/RayTracedRadiosity.js). `LightMapper` rasterizes meshes in `uv1` space with sub-texel jitter and accumulates at most 256 samples in half float, and needs `WebGPURenderer` — [LightMapper.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/lighting/LightMapper.js). Merged for r188 — [PR #35302](https://github.com/mrdoob/three.js/pull/35302)
- Measured: 3–8 ms per radiosity sample, about 1 ms per denoise at 1024², on an RTX 4090 — [PR #35318](https://github.com/mrdoob/three.js/pull/35318) (open)
- A virtual-point-light PR was closed — [PR #34915](https://github.com/mrdoob/three.js/pull/34915)

**LTC area lights**
- `RectAreaLightNode` plus `LTC_Evaluate`: two LUT textures (float if filterable, otherwise half float) — [RectAreaLightNode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/lighting/RectAreaLightNode.js), [LTC.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/BSDF/LTC.js)
- An open PR would generate the textures at runtime, reported at 3.9 ms on CPU on an Apple M1 — [PR #34611](https://github.com/mrdoob/three.js/pull/34611)

**Volumetric lighting**
- `VolumeNodeMaterial` with `VolumetricLightingModel`: per-fragment raymarch in a bounding mesh. Each step evaluates all lights and shadows at the ray position, multiplies by a scattering node, and integrates with Beer's law. Depth compare against the scene is optional — [VolumetricLightingModel.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/VolumetricLightingModel.js)
- The example uses 12 steps, Bayer16 offset, a 128³ noise texture, its own layer and pass, and a Gaussian blur — [webgpu_volume_lighting.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_volume_lighting.html)

### Inferences

| Technique | Verdict | Dependency in 330 |
|---|---|---|
| CSM, SunLight | FRAG | `sampler2DShadow` or array shadow. One depth pass per cascade, or layered through a geometry shader. |
| TileShadowNode | FRAG | Without multiview, render each tile in its own pass or use a geometry shader with `gl_Layer`. Draw cost scales with tiles, and three.js's `ArrayCamera` path does not apply directly. |
| PCF, VSM, PCSS | FRAG | IGN and Vogel are pure maths. RG16F and RG32F targets are core. |
| Clustered | FRAG+ | Assign on the CPU and upload to a texture or TBO, or copy three.js's transform-feedback model. Fragment shader reads with `texelFetch`. |
| DynamicLighting | FRAG | UBOs or uniform arrays. |
| LightProbeGridWebGL | FRAG | Layered 3D target via `glFramebufferTextureLayer`. Bake is fragment-only. RGBA16F 3D filtering assumed. |
| VXGI | COMPUTE | Atomics and imageStore-style writes. No fragment alternative in three.js. Offline voxelize, or use probes instead. |
| RayTracedRadiosity, LightMapper | COMPUTE/BAKE | The lightmap they output is a plain texture at runtime. Baking offline on the CPU removes the dependency. |
| LTC | FRAG | Two LUT textures. Evaluate is maths. |
| Volumetric lighting | FRAG | Step count times shadow lookups, so cost scales fast. Half-res rendering plus blur is the usual mitigation. |

### Gaps
- No measured costs for CSM, VSM, PCF, probe bake or sampling, or VXGI at runtime. [PR #35164](https://github.com/mrdoob/three.js/pull/35164) lists frame times for `webgpu_lightprobes_sponza` (25.63 to 16.98 ms), but that change concerns ArrayCamera culling, so they are not probe costs.
- The word "irradiance volume" isn't used as a separate class. `LightProbeGrid` is the implementation.
- No PCSS node found in `src/nodes` by filename and `ShadowFilterNode` contents. A custom `filterNode` could add one — `CSMShadowNode` passes it to cascades per [PR #35331](https://github.com/mrdoob/three.js/pull/35331).

## Environment

### Takeaway
Sky and flat water are plain fragment shaders. three.js has no physically based atmosphere addon or FFT ocean among the files I located. Its "volumetric clouds" are a raymarched 3D texture in a box. Compute water has a direct fragment equivalent already in the repo.

### Cited Findings
- `Sky` and `SkyMesh`: Preetham analytic daylight on a dome, with Rayleigh, Mie and turbidity uniforms and a sun disc. `Sky` is GLSL for `WebGLRenderer`, and `SkyMesh` is TSL — [SkyMesh.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/objects/SkyMesh.js), [Sky.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/objects/Sky.js)
- Both gained 2D procedural clouds: fbm of gradient noise on a projected plane, Beer's law opacity, Henyey-Greenstein g=0.7 forward lobe, and hazing through the atmosphere. Controls are cloud scale, speed, coverage, density and elevation — [SkyMesh.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/objects/SkyMesh.js), [PR #32682](https://github.com/mrdoob/three.js/pull/32682), [PR #33942](https://github.com/mrdoob/three.js/pull/33942)
- `webgpu_volume_cloud`: `RaymarchingBox` over a 128³ `Data3DTexture` of Perlin noise, up to 100 steps, with the texture built on the CPU — [webgpu_volume_cloud.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_volume_cloud.html)
- `Water` and `WaterMesh`: planar reflection render target (scale 0.5 by default), four scrolled samples of a normal map at different scales, Phong-style sun specular, Schlick Fresnel with f0 0.02, screen-space distortion of the mirror UVs — [WaterMesh.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/objects/WaterMesh.js), [Water.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/objects/Water.js)
- `Water2` and `Water2Mesh`: `Reflector` plus `Refractor` targets, two normal maps blended on a phase-offset flow cycle after Vlachos, with an optional flow map — [Water2Mesh.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/objects/Water2Mesh.js), [Water2.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/objects/Water2.js)
- Compute water: a 128² height field with three storage buffers and ping-pong compute, plus compute-driven ducks, displacing vertices through `positionNode`. The source has a `TODO: Fix example with WebGL backend` — [webgpu_compute_water.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_compute_water.html)
- Fragment alternative: `webgl_gpgpu_water` runs a height-map simulation with `GPUComputationRenderer`, using RGBA float textures and fragment shaders, and reads back with `readRenderTargetPixels` — [webgl_gpgpu_water.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgl_gpgpu_water.html), [GPUComputationRenderer.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/misc/GPUComputationRenderer.js)
- Both "ocean" examples, `webgpu_ocean` (WaterMesh, SkyMesh, bloom) and `webgl_shaders_ocean` (Water, Sky), use the flat normal-mapped water, not displaced waves — [webgpu_ocean.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_ocean.html), [webgl_shaders_ocean.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgl_shaders_ocean.html)

### Inferences

| Technique | Verdict | Dependency in 330 |
|---|---|---|
| Sky, SkyMesh, clouds on sky | FRAG | Port with no feature changes. Baking to a cube map avoids per-pixel fbm cost. |
| Raymarched cloud box | FRAG | 3D texture, core. Cost scales with steps and covered pixels. |
| Water, Water2 | FRAG | One or two extra scene renders for reflection and refraction. |
| Compute water | COMPUTE, with FRAG alternative | Ping-pong RGBA32F or RG32F textures, as in `webgl_gpgpu_water`. Vertex shader texture fetch, core. |

### Gaps
- No atmosphere addon (Bruneton or Hillaire style) turned up in the tree by filename. I did not search file contents.
- No measured cost for water or sky, apart from `webgpu_sky`'s 0.45 ms with one mirror sphere and no PMREM, in a PR whose hardware I did not see — [PR #35130](https://github.com/mrdoob/three.js/pull/35130).

## Geometry and instancing

### Takeaway
`BatchedMesh`, `InstancedMesh`, `LOD` and occlusion queries all rest on CPU decisions plus core GL features, and they are fully portable. three.js has no impostor class, and no compute grass or culling example outside the meshlet pipeline. Indirect draws are WebGPU-side and the WebGL backend does not support them.

### Cited Findings
- `BatchedMesh`: matrices in a float `DataTexture`, per-instance draw ID in an integer "indirect" texture, optional colours in a texture. Per-object frustum culling and sorting run on the CPU each frame, filling `_multiDrawStarts` and `_multiDrawCounts` — [BatchedMesh.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/objects/BatchedMesh.js)
- The vertex stage reads the ID from `drawIndex` or `instanceIndex`, then fetches the matrix with `textureLoad`. Draw format is `multiDraw` on WebGL and `merged` on WebGPU — [Batch.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/accessors/Batch.js)
- The WebGL backend uses `WEBGL_multi_draw` when available, with a fallback branch if the feature is missing — [WebGLBackend.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/renderers/webgl-fallback/WebGLBackend.js), [WebGLBufferRenderer.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/renderers/webgl-fallback/WebGLBufferRenderer.js)
- `LOD`: CPU distance switching with per-level hysteresis — [LOD.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/objects/LOD.js)
- Third-party LOD layers appear in official examples: `@three.ez/batched-mesh-extensions` and `three-mesh-bvh` for batched LOD with BVH culling (5 LODs, 4 from meshoptimizer), and `@needle-tools/gltf-progressive` — [webgl_batch_lod_bvh.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgl_batch_lod_bvh.html), [webgl_loader_gltf_progressive_lod.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgl_loader_gltf_progressive_lod.html)
- Impostors: the only hit was a PR open since 2021 — [PR #22043](https://github.com/mrdoob/three.js/pull/22043). No impostor class in the tree by filename.
- Indirect: `webgpu_struct_drawindirect` draws 100,000 instances with an indirect buffer written by compute — [webgpu_struct_drawindirect.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_struct_drawindirect.html). The WebGL backend warns that `compute()` cannot take an `IndirectStorageBufferAttribute` — [WebGLBackend.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/renderers/webgl-fallback/WebGLBackend.js)
- Occlusion queries: `object.occlusionTest = true` and `renderer.isOccluded( object )`, with `ANY_SAMPLES_PASSED` on WebGL — [webgpu_occlusion.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_occlusion.html), [Renderer.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/renderers/common/Renderer.js), [WebGLBackend.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/renderers/webgl-fallback/WebGLBackend.js)
- `webgpu_tsl_occlusion_dither` exists in the tree. I did not read it — [webgpu_tsl_occlusion_dither.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_tsl_occlusion_dither.html)
- Terrain: `webgpu_tsl_procedural_terrain` displaces a 500×500 plane in the vertex stage with noise and colours it procedurally — [webgpu_tsl_procedural_terrain.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_tsl_procedural_terrain.html)

### Inferences

| Technique | Verdict | Dependency in 330 |
|---|---|---|
| InstancedMesh | FRAG | `glVertexAttribDivisor`, core at 3.3. |
| BatchedMesh | FRAG+ | `glMultiDrawElementsBaseVertex` exists, but no `gl_DrawID` and no base instance. Options: per-draw uniform, a merged index buffer with an ID vertex attribute, or instanced draws where geometry repeats. |
| LOD | FRAG | CPU only. |
| Occlusion queries | FRAG | Core. Results are asynchronous, so expect a frame of lag. |
| Indirect draws | COMPUTE, no 3.3 path | `glDrawArraysIndirect` needs 4.0. Replace with CPU-filled draw lists and instance counts. |
| Impostors, compute grass | Not in three.js | Instanced billboards or grass with CPU-filled instance buffers would be FRAG. See `webgl_buffergeometry_instancing_billboards`. |

### Gaps
- No filename or PR search hit for compute-driven grass. Community projects were out of scope.
- No measured cost for `BatchedMesh` versus `InstancedMesh` found.

## Nanite-like virtualized geometry

### Takeaway
It's real and located in `examples/jsm/pipelines/`: `MeshletDrawingPipeline` and `VisibilityBufferDrawingPipeline`, merged for r187 in [PR #35191](https://github.com/mrdoob/three.js/pull/35191). But it is **not** true Nanite. It builds meshlets and a per-geometry LOD chain, and selects LOD per object. There is no cluster DAG with crack-free per-cluster LOD, and I saw no streaming. It needs compute, storage buffers, atomics and indirect draws, and three.js's own WebGL backend falls back to regular drawing. Its data layout, and its instanced per-cluster draw model, port to GL 3.3 with CPU selection; the software rasterizer does not.

### Cited Findings

**Where it lives and how it arrived**
- Introduced by `renderer.drawingPipeline`, with the addons in `examples/jsm/pipelines/`. "Unsupported objects and materials fall back to the regular pipeline, as does the WebGL backend" — [PR #35191](https://github.com/mrdoob/three.js/pull/35191)
- Confirmed present at tag r187 and absent at r186 (raw file checks on both tags) — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js)
- Both classes are documented as experimental, with an interface that might change — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js), [VisibilityBufferDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/VisibilityBufferDrawingPipeline.js)
- Lineage: the "nanite-style rasterizer" example, a GPU-driven compute software rasterizer rendering 1+ billion triangles, merged for r185 — [PR #33605](https://github.com/mrdoob/three.js/pull/33605). Then `webgpu_compute_rasterizer_ibl` added visibility-buffer shading through `MeshStandardNodeMaterial`, a depth pyramid and specular AA, on 129,600 DamagedHelmet instances at about 2 billion triangles at full detail — [PR #33783](https://github.com/mrdoob/three.js/pull/33783). That example file exists at r186 and is gone at r187 (raw file checks).
- Examples were renamed `webgpu_meshlets_*` — [PR #35282](https://github.com/mrdoob/three.js/pull/35282)

**Build step (`MeshletGeometryUtils`)**
- Welds identical vertices, then builds a chain of simplified levels with `MeshoptSimplifier`, six targets with ratios 1.0, 0.55, 0.25, 0.1, 0.04 and 0.015. Levels beyond the first use `Permissive` and `Prune`, and a sloppy fallback when topology blocks progress — [MeshletGeometryUtils.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/utils/MeshletGeometryUtils.js)
- Each level is clustered independently with `MeshoptClusterizer.buildMeshlets` (64 triangles, cone weight 0.25), bounds from `computeMeshletBounds`, and clusters padded to exactly 64 triangles with degenerates — [MeshletGeometryUtils.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/utils/MeshletGeometryUtils.js)
- All levels index the same vertices. Vertex data is interleaved in one storage buffer: `[ position.xyz, uv.x ], [ normal.xyz, uv.y ]`, then one `vec4` per other attribute. Runs at load in JavaScript with WASM libraries in the tree — [MeshletGeometryUtils.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/utils/MeshletGeometryUtils.js), [meshopt_clusterizer.module.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/libs/meshopt_clusterizer.module.js)
- Vertices are pulled by global index, so a cluster is not limited by a local vertex count — [MeshletGeometryUtils.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/utils/MeshletGeometryUtils.js)

**Per-frame culling and LOD (`MeshletDrawingPipeline`)**
- Compute stages: one thread per object culls, selects the LOD and allocates clusters with one atomic per workgroup. Clusters are dispatched indirectly, and one thread per cluster culls and appends to a queue. One thread per queue writes indirect draw arguments — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js)
- LOD selection is by projected error, `pixelError = errorWorld / dist * cotHalfFov * screenHeight / 2`, taking the last level within threshold. "One thread per object", so one level per object — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js)
- Cluster culling: frustum (bounding sphere), normal cone and two-phase occlusion. Phase one draws clusters visible last frame into occlusion depth. A max-reduction depth pyramid is built, then all clusters are tested and newly visible ones added. It turns itself off when it culls less than a set fraction of clusters — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js), [PR #35191](https://github.com/mrdoob/three.js/pull/35191)
- The depth pyramid lives in a storage buffer, with the first level covering 2×2 pixels — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js)
- Draw: hardware rasterizer, indirect, "one draw instance per cluster of 64 triangles", pulling vertices — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js)
- The CPU still traverses the scene and uploads changed transforms each frame unless a static `BundleGroup` is used — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js)
- Supported materials are opaque or alpha-tested, front-side, with unsupported nodes on the regular path. Shadow maps use a regular path per docs, though the PR text lists shadow maps as supported. I did not resolve that — [MeshletDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/MeshletDrawingPipeline.js), [PR #35191](https://github.com/mrdoob/three.js/pull/35191)

**Visibility buffer and software rasterizer (`VisibilityBufferDrawingPipeline`)**
- Two 32-bit storage buffers per pixel, updated with `atomicMax`: `depth(16)|triangle(16)` and `depth(15)|instance(17)`, with a fourth-root depth distribution. Depth in the high bits resolves test and payload in one atomic — [VisibilityBufferDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/VisibilityBufferDrawingPipeline.js)
- The compute rasterizer takes one thread per triangle and loops over pixels, so it only suits small triangles. Large ones and those crossing the near plane are queued for the hardware path — [VisibilityBufferDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/VisibilityBufferDrawingPipeline.js)
- Resolve draws each object's bounding box (not fullscreen), reconstructs perspective-correct barycentrics from edge functions, and computes analytic derivatives for textures and tangent frames — [VisibilityBufferDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/VisibilityBufferDrawingPipeline.js)
- Limits: LODs whose triangle index doesn't fit 16 bits use the hardware pipeline. Resolve writes colour only, so MRT passes such as TRAA velocity fall back to hardware — [VisibilityBufferDrawingPipeline.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/pipelines/VisibilityBufferDrawingPipeline.js)

**Measured ([PR #35191](https://github.com/mrdoob/three.js/pull/35191), RTX 4070, 1920×1080, vsync off, median of 2 runs)**

| Example | Standard | Meshlet | LOD error |
|---|---|---|---|
| `webgpu_gpudriven_instanced` | GPU 230.1 ms, CPU 231.0 ms | GPU 3.0 ms, CPU 3.0 ms | 1 px |
| `webgpu_gpudriven_city` | GPU 5.3 ms, CPU 5.2 ms | GPU 2.4 ms, CPU 1.5 ms | 1.5 px |
| `webgpu_lightprobes_sponza` | GPU 0.60 ms, CPU 3.2 ms | GPU 0.73 ms, CPU 1.0 ms | 1 px |

- The author says gains are largest in vertex-bound scenes without manual LODs or culling, and smaller in fragment-bound or already optimized ones. Sponza's GPU time got slightly worse — [PR #35191](https://github.com/mrdoob/three.js/pull/35191)

**How this differs from Nanite**
- Nanite: per the Karis SIGGRAPH 2021 talk, a virtual geometry system — [Karis et al. slides](https://advances.realtimerendering.com/s2021/Karis_Nanite_SIGGRAPH_Advances_2021_final.pdf). Secondary sources describe grouping adjacent clusters, simplifying with locked outer boundaries and re-clustering into a DAG, then a hybrid rasterizer writing a 64-bit visibility buffer — [VR wiki summary (secondary)](https://vrarwiki.com/wiki/Nanite). I did not read the slides in full.
- The reference open implementation of that scheme is meshoptimizer's `clusterlod.h`: partition into groups, lock boundaries shared between groups, simplify, re-cluster, repeat. Runtime draws a cluster when its group's error is above threshold and its refined group's error is at or below it, which gives a consistent, gap-free cut — [clusterlod.h](https://raw.githubusercontent.com/zeux/meshoptimizer/master/demo/clusterlod.h)
- Community three.js libraries with claimed cluster LOD exist, but I did not review their code: [three-virtual-geometry](https://github.com/elad12390/three-virtual-geometry), [three-meshlet](https://github.com/seanhlewis/three-meshlet)

### Inferences

**Dependency verdict for the three.js method:** COMPUTE. It uses `glDispatchCompute`-equivalent work (4.3), storage buffers, atomics, indirect dispatch and indirect draw (4.0). None is available on 3.3.

**What a GL 3.3 version keeps and loses:**

| Part | On GL 3.3 without compute | What's lost |
|---|---|---|
| Meshlet and LOD chain build | Works. Offline tool or native meshoptimizer. | Nothing. |
| DAG cluster LOD via `clusterlod` | Works offline. Runtime selection on the CPU. | CPU cost scales with cluster count; a per-group BVH reduces it. |
| Vertex pulling | TBO plus `texelFetch` in the vertex shader. `gl_VertexID` indexes it. | Possibly slower than storage buffers; no data. |
| Draw | `glDrawArraysInstanced( GL_TRIANGLES, 0, 192, visibleClusters )`, one instance per 64-triangle cluster, with a per-instance attribute carrying the cluster and object IDs from a CPU-written streaming VBO. This mirrors three.js's model with the indirect count replaced by a CPU count. | No post-transform cache reuse (three.js pulls too, so same trade). Degenerates in padded clusters still run the vertex shader. |
| Frustum and cone culling | CPU per cluster. Or transform feedback with a geometry shader as filter, with the count from a query. | Slow on the CPU at very high counts. TF count is stalled or a frame late, because `glDrawTransformFeedback` is 4.0. |
| Occlusion | Fragment-built max-depth mip chain. Read back through a PBO for CPU tests, one or more frames late; or test in the vertex stage against `textureLod` and collapse the cluster. Occlusion queries for coarse objects. | Same-frame two-phase correctness. Expect popping or conservative over-draw when disoccluding. |
| Hardware visibility buffer | RG32UI MRT, `gl_PrimitiveID` plus per-instance ID, normal depth test. Resolve in a fullscreen fragment pass with TBO lookups and analytic derivatives. | Works without atomics, since the depth buffer resolves it. Conflicts with forward shading, so it adds a deferred-style material path. Transparent stays forward. |
| Software rasterizer | **Not portable.** Depends on compute and atomics. | Micro-triangle efficiency. Small triangles still pay hardware quad overdraw, so geometry density must be limited by LOD error. |
| Streaming | three.js shows none. Would be CPU-side `glBufferSubData` either way. | Not applicable. |

- The practical 3.3 subset is: offline clusters, CPU LOD (per object as in three.js, or per cluster with `clusterlod`), CPU frustum and cone culling, delayed Hi-Z, and instanced cluster draws. This keeps three.js's biggest gains for raw vertex-bound scenes in kind, but the PR's 77× figure cannot be transferred: it relies on GPU-side culling, on an RTX 4070. No measurement exists for a CPU-driven version.
- Also worth noting: on already LOD'd, fragment-bound scenes even the full pipeline gained little (Sponza). So the case for a 3.3 port depends on the content.

### Gaps
- I did not read all 190 KB of `MeshletDrawingPipeline.js`. "No DAG" rests on the build code (independent levels, per-object selection) and the docs. "No streaming" rests on not seeing it in comments or greps, so it is less firm.
- The `webgpu_meshlets_instanced` description says "culled and simplified per cluster", which conflicts with the per-object selection in the code. I trust the code — [webgpu_meshlets_instanced.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_meshlets_instanced.html)
- No published measurement of any three.js meshlet path on older hardware, or of a CPU-driven GL 3.3 equivalent.
- The Karis slide details are through secondary sources here.

## Materials

### Takeaway
All `MeshPhysicalMaterial` lobes are closed-form BRDF maths plus LUTs and prefiltered environment lookups, so they are FRAG. Only transmission needs an extra scene texture. There's no built-in wetness or terrain node: those are per-example custom shaders.

### Cited Findings
- Base model: multi-scatter compensation per Fdez-Agüera for IBL and Turquin for direct light, with a DFG LUT — [PhysicalLightingModel.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/PhysicalLightingModel.js), [DFGLUT.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/BSDF/DFGLUT.js)
- r186 energy fixes weight direct diffuse by `1 - F(VdotH)` and apply the same Fdez-Agüera logic to indirect diffuse — [PR #33985](https://github.com/mrdoob/three.js/pull/33985)
- Sheen: Charlie BRDF, with a curve fit for the hemispherical albedo per Estevez and Kulla 2017 and Filament — [BRDF_Sheen.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/BSDF/BRDF_Sheen.js), [PhysicalLightingModel.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/PhysicalLightingModel.js)
- Anisotropy: anisotropic GGX D and Smith V terms — [D_GGX_Anisotropic.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/BSDF/D_GGX_Anisotropic.js), [V_GGX_SmithCorrelated_Anisotropic.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/BSDF/V_GGX_SmithCorrelated_Anisotropic.js)
- Iridescence: Belcour and Barla thin-film Fresnel — [PhysicalLightingModel.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/PhysicalLightingModel.js)
- Transmission: samples a mipped viewport texture of the opaque scene at a refracted coordinate, with roughness scaled by IOR, and Beer-style attenuation — [PhysicalLightingModel.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/PhysicalLightingModel.js)
- Dispersion: `halfSpread = (ior - 1) * dispersion * 0.025`, with separate refraction per channel, so three transmission samples per fragment — [PhysicalLightingModel.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/PhysicalLightingModel.js)
- EON rough diffuse (cited to JCGT volume 14) is new and opt-in, on both renderers — [PR #34379](https://github.com/mrdoob/three.js/pull/34379), [BRDF_EON.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/functions/BSDF/BRDF_EON.js)
- r187 PMREMs moved from the CubeUV atlas to cube render targets, and `CubeUVReflectionMapping` is gone. Materials look slightly different — [r187 release](https://github.com/mrdoob/three.js/releases/tag/r187)
- PMREM generation by fragment takes about 4.6 ms at size 256. An open PR does it with compute at about 1.2 ms. Hardware not seen in my excerpt. This is one of the few direct fragment-versus-compute comparisons in the repo — [PR #35130](https://github.com/mrdoob/three.js/pull/35130)
- Wetness: no node or example by filename. The only mention I found is "procedural wet surfaces" in a demo attached to a PR — [PR #34931](https://github.com/mrdoob/three.js/pull/34931)
- Terrain: no terrain node. Custom `positionNode` and `colorNode` in the example — [webgpu_tsl_procedural_terrain.html](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/webgpu_tsl_procedural_terrain.html)

### Inferences

| Feature | Verdict | Dependency in 330 |
|---|---|---|
| Clearcoat, sheen, anisotropy, iridescence, EON | FRAG | Maths plus DFG LUT. Each adds fragment cost, so compile out with `#define`s. |
| Transmission, dispersion | FRAG | Copy the opaque scene to a mipped texture before transparents. Dispersion triples the lookups. |
| Environment prefilter | FRAG | Fragment path is what three.js has always used. Do it at load or at intervals. |

### Gaps
- No measured per-lobe cost found.

## Tone mapping and colour management

### Takeaway
All operators are per-pixel maths, FRAG in 330, and can be copied directly. The order of operations with AA matters: SMAA wants linear, FXAA wants sRGB.

### Cited Findings
- All from [ToneMappingFunctions.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/nodes/display/ToneMappingFunctions.js):
  - **ACES filmic**: Hill's RRT/ODT fit from `ltc_code`, with exposure divided by 0.6 and input and output matrices.
  - **AgX**: linear sRGB to Rec.2020, inset matrix, log2 encoding from -12.47393 to 4.026069 EV, a 6th-degree contrast polynomial, outset matrix, power 2.2, then back to sRGB.
  - **Neutral**: the Khronos-style operator with `StartCompression = 0.76`, `Desaturation = 0.15`, and a 0.04 offset (smaller when the minimum channel is below 0.08). Doc reference is model-viewer.
  - **Also**: Linear, Reinhard, Cineon.
- Working space is linear sRGB by default, with colour spaces registered through `ColorManagement.define` — [ColorManagement.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/src/math/ColorManagement.js)
- `SMAANode` should be applied before sRGB conversion — [SMAANode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/SMAANode.js). `FXAANode` requires sRGB input after tone mapping — [FXAANode.js](https://github.com/mrdoob/three.js/blob/7e26babbbe69f411112193d04a41baeafb6ebb3b/examples/jsm/tsl/display/FXAANode.js)

### Inferences
- Verdict: FRAG for all. No cost data was found, but these are a few ALU operations per pixel with no texture reads.
- AgX's `log2` and matrix steps make it the heaviest of the three. That's from reading the code; I have no measurement.
- `GL_FRAMEBUFFER_SRGB` could replace manual encode at the end of the chain when writing to an sRGB default framebuffer.

### Gaps
- I did not verify wide-gamut (Display P3) paths in depth. Only the `define` mechanism was seen in my grep.
- No published cost comparison of these three operators was found.
