# Stylized and low-poly rendering plus post-processing for moonBASIC (OpenGL 3.3 forward)

Evidence labels used in these notes:
- **Primary**: read directly from the author's paper, notes, spec, docs or devlog.
- **Secondary**: a tutorial, port or forum post. Forum claims are flagged as such.
- **Repo**: read directly from this repository's code.
- Everything under "Inferences" is engineering judgement, not a sourced fact. All cost and gain rankings are judgement too, because I found no benchmark that compares these techniques on one machine.

## How do ramp lighting, gradient shading, cel shading, dithered shading, hand-painted looks and stylized water and foliage work?

### Takeaway
Nearly all of these looks replace one or two terms of the lighting equation with an authored curve and put the quality into authored data: normals, thresholds, gradients and colour scripts. For low-poly environments, the best return comes from gradient sky and height/distance fog, ramp lighting, shape normals on foliage, and depth-based water. All of these work on GL 3.3 forward. Shipped-game primary sources were only partly reachable, so see Gaps.

### Cited Findings

**Ramp, toon and cel lighting**
- Filament's official toon example is a custom per-light `surfaceShading` function. It quantizes `NdotL` into five shades with a minimum ambient of 0.1, multiplies by `visibility` and `attenuation`, and returns `diffuseColor * light * toon`. The function is also called for shadowed fragments, so constant ambient can be handled there (Primary) — [Filament Materials](https://google.github.io/filament/main/materials.html)
- Guilty Gear Xrd (Motomura, GDC 2015): cel shading is binary, lit or unlit, so a small normal change can produce a large dark patch. A convincing 2D look therefore needs precise control over how shades are distributed (seen through a search excerpt of the deck; direct fetch failed) — [Motomura slides](https://www.ggxrd.com/Motomura_Junya_GuiltyGearXrd.pdf)
- Xrd's artists hand-edited vertex normals because automatic normals often do not match artistic intent (deck excerpt, as above). Polycount commenters say this was done in Softimage. They also believe vertex colours drive a per-vertex shading threshold, and that occlusion maps mark where light falls. These are explicitly forum interpretations, not confirmed by the talk — [Polycount thread](https://polycount.com/discussion/comment/2666816)
- Sable: the only source found is a Steam forum guess at a smoothstep two-tone threshold, one directional light, and post-process edges from depth and normals. It is not a developer statement, so treat it as low confidence — [Steam thread](https://steamcommunity.com/app/757310/discussions/0/3079880882845833169/)
- Hi-Fi Rush (GDC session, listing excerpt only): a deferred toon renderer whose agenda covers comic shaders, toon lights, shadow-map strategies and face shadows. It is a useful lead, but its details were not reachable — [GDC schedule](https://schedule.gdconf.com/session/3d-toon-rendering-in-hi-fi-rush/899802)
- Genshin Impact (Unity Japan deck of miHoYo's talk, search excerpt): the engine uses two customized pipelines, one for PC and console and one for mobile, and describes its tone as PBR-based stylized — [Docswell deck](https://docswell.com/s/UnityJapan/KWRPQ5-210617-unity-dojo20211mihoyozhenzhongyi)

**Gradient shading and environment colour (Firewatch)**
- Firewatch's sky is layered, including a three-colour gradient (top, half, bottom) that controls the main sky colour, plus a sun disc and halos. Most of the colour comes from lighting, fog and colour correction, and most rocks have a grey diffuse. An interpolation manager blends sun direction, sun disc, two fog systems and a colour-correction post effect (Primary, studio blog; search excerpts) — [Campo Santo post](https://blog.camposanto.com/post/112703721804/this-blog-post-is-an-in-detail-explanation-of-part), [Campo Santo Q&A](https://blog.camposanto.com/post/102893584324/i-asked-twitter-if-anyone-had-questions-about-the)
- Firewatch contrasts a flat diffuse-only shader with one adding gloss and normal maps, because the team wanted a stylized world that still felt like a tangible space — [Campo Santo blog](https://blog.camposanto.com/tagged/posted%20by%20jane/page/2)

**Dithered shading (Return of the Obra Dinn)**
- Bayer patterns reproduce tones more accurately and blue noise looks more organic. At 640x360 stretched fullscreen, the pixels were big enough that the eye could not blend them, so the author concluded dithering should be used sparingly. The final build unified the 3D camera with the 2D dither pattern and raised the resolution to 800x450 (Primary) — [PlayStation Blog, Lucas Pope](https://blog.playstation.com/archive/2019/10/17/lucas-pope-on-return-of-the-obra-dinns-art-style)
- A screen-mapped dither is disorienting during camera rotation, because the eye can track everything. Pope stabilized rotation but could not find a way to stabilize translation, and accepted that because so much changes onscreen during translation that the dither is less noticeable (Primary) — [Obra Dinn devlog #800](https://dukope.com/devlogs/obra-dinn/tig-34/)
- The stabilizer maps the dither pattern onto a sphere. Blue noise also survived screen scaling and video compression better than Bayer — [Zucconi showcase](https://www.alanzucconi.com/2018/10/24/shader-showcase-saturday-11/). The blue-noise matrix was generated from void-and-cluster research (search excerpt) — [Obra Dinn devlog 2014](https://dukope.com/devlogs/obra-dinn/tig-02/)
- Outlines made the 3D space easier to read in Obra Dinn. Pope notes they would suit games poorly where obscuring objects matters or draw distances are long — [PlayStation Blog](https://blog.playstation.com/archive/2019/10/17/lucas-pope-on-return-of-the-obra-dinns-art-style)

**Stylized water**
- Roystan's toon water recipe:
  - Depth difference (scene eye depth minus water depth), divided by a maximum and clamped, blends shallow to deep colour.
  - Scrolling Perlin noise is thresholded into binary waves, with a very narrow `smoothstep` to reduce aliasing.
  - A two-channel distortion texture offsets the noise UVs.
  - Foam comes from lowering the noise cutoff near the shore.
  - An optional scene-normal comparison keeps foam width consistent on steep and flat shores.
  - State: transparent queue, alpha blend, depth write off. The article also renders a separate normals pass, because Unity's combined depth-normals mode lacks depth precision — [Roystan Toon Water](https://roystan.net/articles/toon-water/)
- Portability pitfall on OpenGL: depth must be remapped from [0,1] to [-1,1] before reconstructing position, or foam can come out pure white everywhere (forum fix, search excerpt) — [three.js forum](https://discourse.threejs.org/t/toon-water-shader-with-depth-based-fog-and-intersection-foam/35978)
- Wind Waker's look differs from depth-based shorelines, per a Unity forum poster. No official Nintendo breakdown was found — [Unity forum](https://discussions.unity.com/t/the-legend-of-zelda-wind-waker-styled-water/567867)

**Foliage**
- Grass shape normals should follow the terrain. Tree normals should point out from a central location, following a sphere or cone. Using shape normals for diffuse hides that the mesh is made of flat planes. The same tutorial covers translucency and alpha clipping (search excerpt; direct fetch returned 403) — [NedMakesGames tutorial](https://nedmakesgames.medium.com/creating-a-foliage-shader-in-unity-urp-shader-graph-5854bf8dc4c2)
- A commercial vegetation package drives wind from vertex colours (red for wiggle, alpha for main wind) and uses ramp textures for light falloff. This is vendor marketing, but it shows the common data layout — [Fab listing](https://www.fab.com/listings/1a592c1a-49c8-4a95-bf2e-674b95b239f0)
- Cheap billboard trees offset quad vertices from UVs. The illusion breaks if the vertices are too far apart in model space, and leaves thin out with distance, which a lower alpha threshold counters (community post) — [Godot Shaders](https://godotshaders.com/shader/simple-cheap-stylized-tree-shader/)

### Inferences

**Ranking for moonBASIC low-poly environments, best gain per cost first:**

| Rank | Technique | Runtime cost | Why it ranks here |
|---|---|---|---|
| 1 | Three-colour gradient sky plus distance/height fog tinted from the same gradient | Few ALU per pixel | Ties every object into one colour script. Firewatch gets most of its colour this way |
| 2 | Per-material 1D ramp replacing `NdotL`, plus hemisphere ambient | One texture read | Works with existing lights and shadows. Must multiply by shadow visibility or shadows ignore the ramp |
| 3 | Flat-shaded option via `dFdx`/`dFdy` normals, or authored shape normals | Near zero | Flat facets are the low-poly signature, and shape normals fix foliage |
| 4 | Toon water from depth difference, thresholded noise, foam | One depth read plus 2-3 texture reads, on water pixels only | Very high visual return. Needs a depth copy before the transparent pass |
| 5 | Foliage: shape normals, wrap or translucent lighting, vertex-colour wind, alpha-to-coverage or dithered clip | Vertex math plus small ALU | Wind makes a static scene feel alive |
| 6 | Baked vertex AO, or per-vertex colour gradients on meshes | Zero at runtime, some tool work | Reads as hand painted without textures |
| 7 | World-stable dither for fade and shading, with blue noise instead of Bayer where motion matters | One small texture read | Cheap but polarizing. Keep it optional |
| 8 | Xrd-grade per-vertex threshold and normal editing | Small at runtime, heavy authoring | Mainly for hero characters, and it depends on tools |

**Implementation notes on GL 3.3**
- Ramp as data: sample `ramp(0.5 + 0.5 * NdotL)` with clamped filtering, so the ramp gives both wrap and bands. Use nearest filtering or a narrow `smoothstep` for hard bands. Hard thresholds alias badly unless the width follows `fwidth(NdotL)`, the same idea as Roystan's narrow smoothstep.
- Multi-light toon: apply the ramp per light, as Filament does. Point lights should ramp attenuation separately, or bands form ugly rings.
- Linear or gamma: authored ramps look right when applied in the space they were painted in. Ramp textures should be declared sRGB, and the ramp result used as a linear multiplier in linear mode. Otherwise the same ramp looks different in Classic and Advanced.
- Hand-painted looks are mostly an authoring discipline: light and shadow baked into albedo, low or no specular, and unlit or lightly lit shading. The engine's job is to avoid fighting the paint, which means reducing tonemap contrast and bloom. The neutral tonemap or a mix-to-unlit parameter suits this better than ACES, which shifts hue and contrast.
- Dithered shading: dither on the lighting term before quantizing, not on final colour, when combined with a retro palette. Keep the screen-space pattern at integer scale, since Pope found that stretching defeats blending.
- Sphere-mapped dither (rotation stability) could be approximated by indexing the pattern with view-direction angles rather than `gl_FragCoord`. The exact mapping is only partly documented, so prototype before committing.
- The moonBASIC water FBO already exists (per [docs/POSTFX.md](../../docs/POSTFX.md), Repo), so depth-based foam should be a small addition. Scene normals for the optional steep-shore foam can be reconstructed from depth derivatives rather than adding a normal pass.

### Gaps
- The Motomura PDF refused connection, so its details come from a search excerpt and forum interpretation. The per-vertex threshold and occlusion-channel claims remain unconfirmed.
- No primary source was found for Breath of the Wild, Wind Waker, Tunic, or Sable rendering. Do not cite these as technical examples without new evidence.
- No primary source was found for hand-painted texture workflows (for example World of Warcraft or Torchlight style). The hand-painted notes above are judgement.
- Firewatch's GDC talk pages are member-gated. The specific "fog gradient lookup" could not be confirmed. Only the three-colour sky gradient and the two blended fog systems are sourced.
- Genshin ramp and SDF face shadow specifics were not found in a primary source.

## What is a sound way to offer Classic, Stylized and Advanced profiles over one material interface?

### Takeaway
Keep one material record and make a profile only a default bundle: a lighting function per material plus a post preset per camera. Style must be selectable per material, so mixed scenes work naturally in forward. Style-specific fields sit in an optional block that other profiles ignore. No source prescribes this exact profile design. It is an inference from how Filament and glTF separate data from the shading model.

### Cited Findings
- Filament selects behaviour by shading model (lit, subsurface, cloth, unlit, specular-glossiness) over a shared parameter vocabulary. Unlit exposes only `baseColor`, `emissive` and `postLightingColor`, and has a `shadowMultiplier` option for shadow-receiving invisible planes — [Filament Materials](https://google.github.io/filament/main/materials.html)
- `customSurfaceShading` swaps the per-light function while keeping the rest of the lit pipeline, and it is supported only with the lit model. This is an existing precedent for toon shading as a per-light swap rather than a separate renderer — [Filament Materials](https://google.github.io/filament/main/materials.html)
- `variantFilter` skips variants the app will never need (such as `dynamicLighting`, `shadowReceiver`, `skinning`, `ssr`, `motionVectors`), but filtering a variant the runtime needs can crash it. `constants` are specialized at package load time — [Filament Materials](https://google.github.io/filament/main/materials.html)
- `KHR_materials_unlit` and `KHR_materials_variants` are ratified glTF extensions, so assets can carry an unlit flag and multiple material bindings — [glTF extensions registry](https://github.com/KhronosGroup/glTF/blob/main/extensions/README.md)
- Hi-Fi Rush shows a shipped game going fully toon with toon lights and its own shadow-map strategies, on a deferred renderer (listing excerpt) — [GDC schedule](https://schedule.gdconf.com/session/3d-toon-rendering-in-hi-fi-rush/899802)
- moonBASIC's post stack is one fullscreen pass with uniforms for each effect (Repo) — [internal/runtime/postfx_shaders.go](../../internal/runtime/postfx_shaders.go)

### Inferences

**Recommended split**

| Layer | Scope | Contents |
|---|---|---|
| Material data | Per material, same in every profile | `baseColor`, `metallic`, `roughness`, `normal`, `occlusion`, `emissive`, `alphaMode`, `alphaCutoff`, `doubleSided` |
| `shading` selector | Per material | `unlit`, `ramp`, `toon`, `retro`, `pbr` |
| Stylize block | Per material, ignored by `pbr` and `unlit` | `ramp` texture, `shadowTint`, `rimColor`, `rimPower`, `specularSteps`, `outlineMask`, `flatShade`, `ditherFade` |
| Post preset | Per camera | Bloom, AA, grade, outline, tonemap, grain |
| Profile | Project or scene default | Only sets defaults on the three layers above. Any field can be overridden |

**What each profile defaults to**

| | Classic | Stylized | Advanced |
|---|---|---|---|
| Shading default | `unlit` or `retro`, Lambert with vertex colour | `ramp` with gradient ambient | `pbr` (metallic-roughness with IBL) |
| Colour pipeline | Gamma, clamp or none | Linear, neutral tonemap | Linear, ACES or neutral |
| AA | MSAA or none, keep hard pixels | SMAA 1x, or MSAA plus post AA | SMAA 1x as default, TAA optional |
| Effects | Palette, vertex snap | Outlines, bloom, LUT | GTAO, bloom, DoF, motion blur, SSR |

**Rules that keep style mixing sound**
1. Same inputs give sensible output everywhere. A PBR asset under `ramp` should use `baseColor` and ignore or reduce specular. A ramp asset under `pbr` should fall back to `roughness` 1 and `metallic` 0. Importing a glTF always works.
2. Parameter semantics do not change across profiles. Do not make `roughness` mean "number of bands" in one profile.
3. Per-material style is free in forward, because each draw picks its own shader. Effects that act on the whole frame are the real conflict. These need per-pixel masks to differ by object: outline on or off, no tonemap on UI-like objects, no bloom on palette objects. Options on GL 3.3 are stencil bits (cheap, limited to a few flags) or an RGBA8 MRT attachment.
4. Retro palette as a post effect applies to the whole frame. To let a retro object sit in a PBR scene, offer an in-material palette or quantize mode as well.
5. Shader permutations: SPIR-V specialization constants are not available on 3.3 core, so use `#define`-generated programs compiled lazily and cached by a key built from (shading, skin, shadow, alpha, light tier). Keep the toon, ramp and PBR differences inside one `surfaceShading` include, as in Filament, so lights and shadows have one code path. Unlike Filament, an unexpected variant should compile on demand rather than crash.
6. Use a uniform buffer for the stylize block, so the parameter layout is shared. UBOs are in GL 3.3 core.

### Gaps
- I found no published write-up of a shipped engine offering user-selectable retro, stylized and PBR profiles over a single material model. The recommendation is a synthesis.
- Permutation counts and compile hitching on moonBASIC's target drivers were not measured.

## Post-processing on GL 3.3: FXAA vs SMAA vs TAA, bloom methods, depth of field, motion blur, SSR on forward, and correct effect order

### Takeaway
Order matters more than any new effect. moonBASIC's current `fxaa()` should be fixed first: it is a contrast-gated 5-tap blur that runs on the HDR input before tonemapping, which is both not FXAA and where the FXAA paper says not to run. Then replace it with real SMAA 1x (MIT-style licence, has a GLSL 3 target) or FXAA 3.11 at the end of the LDR chain. Upgrade bloom to a Jimenez-style 13-tap and tent chain. Treat TAA, DoF, motion blur and SSR as optional later tiers.

### Cited Findings

**Anti-aliasing: published costs**

| Method | Published cost | Hardware and resolution | Source |
|---|---|---|---|
| FXAA preset 2 | Under 1 ms | GTX 480, 1920x1200 | [FXAA whitepaper](https://developer.download.nvidia.com/assets/gamedev/files/sdk/11/FXAA_WhitePaper.pdf) |
| SMAA 1x | 1.02 ms | GTX 470, 1080p | [SMAA paper](https://www.iryoku.com/smaa/downloads/SMAA-Enhanced-Subpixel-Morphological-Antialiasing.pdf) |
| SMAA T2x | 1.32 ms | GTX 470, 1080p | same |
| SMAA S2x | 2.04 ms | GTX 470, 1080p | same |
| SMAA 4x | 2.34 ms | GTX 470, 1080p | same |
| MSAA 8x | 5.4 ms average, 7.7 ms bandwidth-heavy; up to 126 MB in forward | GTX 470 | same |

These are authors' own numbers on 2010-2011 GPUs and from different test setups, so compare them only loosely. The paper's own figures for 4x conflict: its Figure 1 caption says 2.6 ms and its results section says 2.34 ms. Current GPUs should be much faster, but I found no modern measurements.

- **FXAA placement and input.** It is engineered for the end of post-processing, after conversion to LDR and sRGB. Running it on HDR before tonemapping turns off AA on edges that span in-range and out-of-range values: 0.0 and 16.0 average to 8.0, which may tonemap to 1.0, with no visible blend. It needs perceptually encoded input — [FXAA whitepaper](https://developer.download.nvidia.com/assets/gamedev/files/sdk/11/FXAA_WhitePaper.pdf)
- **FXAA algorithm.** Compare local luma contrast against a threshold, classify horizontal or vertical edges, search along the edge for its ends, compute a sub-pixel shift perpendicular to the edge, then resample. A sub-pixel test also reduces single-pixel aliasing — [FXAA whitepaper](https://developer.download.nvidia.com/assets/gamedev/files/sdk/11/FXAA_WhitePaper.pdf)
- **SMAA.** Adds local contrast analysis, sharp geometric feature and diagonal handling, and accelerated pattern classification. It can be combined with MSAA, SSAA and temporal reprojection — [SMAA page](https://www.iryoku.com/smaa/)
- **SMAA integration.** `SMAA_GLSL_3` is a supported target. It needs two RGBA targets (edges and blend weights), precomputed `areaTex` and `searchTex`, three passes, and linear clamp samplers. Input for luma and colour edge detection must be gamma-corrected. Presets run from LOW (threshold 0.15, 4 search steps, about 60% quality) to ULTRA (0.05, 32 steps, about 99%). Start with luma detection. Predication with depth is optional, and the header notes depth suits indoor or short-range scenes. A stencil mask speeds up the second pass — [SMAA.hlsl](https://raw.githubusercontent.com/iryoku/smaa/master/SMAA.hlsl)
- **SMAA licence.** MIT-style — [SMAA LICENSE](https://raw.githubusercontent.com/iryoku/smaa/master/LICENSE.txt)
- **SMAA T2x artifacts.** It weights history by a velocity difference (constant K = 30 in the paper). This handles disocclusion robustly, but disoccluded regions get no temporal AA, which the morphological or multisample components then cover — [SMAA paper](https://www.iryoku.com/smaa/downloads/SMAA-Enhanced-Subpixel-Morphological-Antialiasing.pdf)
- **TAA.** Karis's SIGGRAPH 2014 talk is the standard reference, cited by Blender and three.js for luma-weighted blending and reducing the history weight near the clip box — [Blender commit](https://developer.blender.org/rB8ac5b1fdb39), [three.js TRAA PR](https://github.com/mrdoob/three.js/pull/32133/files)
- **TAA tuning.** Neighbourhood clipping is called the main ingredient of modern temporal methods — [Salvi GDC 2016](https://developer.download.nvidia.com/gameworks/events/GDC2016/msalvi_temporal_supersampling.pdf). The three.js implementation clamps to a 3x3 box and uses mean ± 1.25 standard deviations, noting that tighter clipping means less ghosting but more flicker — [three.js TRAA PR](https://github.com/mrdoob/three.js/pull/32133/files)
- **TAA costs.** Consumer guides quote FPS percentages (2-7%) that conflict and are not controlled. I would not use them — [GamerHardware](https://gamerhardware.org/what-is-anti-aliasing-gaming/)

**Bloom**
- Jimenez's Advanced Warfare method uses a pyramidal filter hierarchy for temporal stability. Fireflies are handled by weighting each 2x2 block by `1/(1+luma)`. Upsampling is a 3x3 tent with a radius in UV space. No threshold is used: bloom blends at about 0.04, so only very bright pixels bloom noticeably. Bloom runs before exposure in that pipeline. No timing numbers appear on the page — [Jimenez post](https://www.iryoku.com/next-generation-post-processing-in-call-of-duty-advanced-warfare/)
- LearnOpenGL gives a full `#version 330 core` implementation:
  - 13-tap downsample with weights 0.125 (centre), 0.03125 (corners), 0.0625 (edge midpoints), 0.125 (inner four).
  - 9-tap tent upsample, additive with `glBlendFunc(GL_ONE, GL_ONE)`, `filterRadius` about 0.005.
  - 5 or 6 mips in `GL_R11F_G11F_B10F`, clamp to edge.
  - Mixed into HDR by `mix(hdr, bloom, 0.04)`, with useful values of 0.03 to 0.15.
  - Karis average only on the first downsample, plus `max(x, 0.0001)` to stop black boxes spreading — [LearnOpenGL Phys.-Based Bloom](https://learnopengl.com/Guest-Articles/2022/Phys.-Based-Bloom)
- Dual filter (Bjørge, ARM, SIGGRAPH 2015; note this is not Jimenez):
  - 5-tap downsample (centre x4 plus four half-pixel diagonals, divided by 8).
  - 8-tap upsample (four axis at 2x half-pixel weight 1, four diagonals at weight 2, divided by 12).
  - In the 97x97-equivalent test on a Mali-T760 MP8, it was the fastest, used 7% of total bandwidth compared to the linear-sampling filter and less than half of Kawase's. PSNR was slightly below Kawase. Naive mipmap blur showed blocky, unstable artifacts — [Bjørge notes](https://community.arm.com/cfs-file/__key/communityserver-blogs-components-weblogfiles/00-00-00-20-66/siggraph2015_2D00_mmg_2D00_marius_2D00_notes.pdf)
  - The notes' listing has syntax slips (missing parentheses and semicolon), so correct them when porting.
  - Android's RenderEngine and KWin both adopted dual Kawase — [Android commit](https://gerrit.omnirom.org/plugins/gitiles/android_frameworks_native/+/263152043dee4d2bce8df5e442aac88da4304127%5E%21), [KWin](https://invent.kde.org/plasma/kwin/-/blob/0179f741bbd4cc01da5866fdf7b6a302869a099b/effects/blur/blur.h)

**Depth of field**
- Jimenez uses scatter-as-you-gather with foreground and background classification from CoC, tiles storing maximum CoC, and an alpha mask for silhouettes. Transparents are a known problem: a separate translucency DoF pass was too expensive, so transparents were rendered before or after post — [Jimenez post](https://www.iryoku.com/next-generation-post-processing-in-call-of-duty-advanced-warfare/)

**Motion blur**
- McGuire-style reconstruction: tile-max velocity, then neighbour-max, then a gather. NVIDIA's GLSL sample suggests radius K about 20 and S about 15 taps, using only `texelFetch` and `textureSize` — [NVIDIA sample](https://docs.nvidia.com/gameworks/content/gameworkslibrary/graphicssamples/opengl_samples/motionblurgl4gles3advancedsample.htm). Note the sample is labelled GL4/GLES3, so verify it against 3.3 when porting.
- Guertin, McGuire and Nowrouzezahrai's filter reports about 3 ms per frame — [Guertin 2014](https://mail.casual-effects.com/research/Guertin2014MotionBlur/Guertin2014MotionBlur-small.pdf)
- Jimenez removed camera-rotation blur from his blur to reduce problems between neighbours with orthogonal velocities — [Jimenez post](https://www.iryoku.com/next-generation-post-processing-in-call-of-duty-advanced-warfare/)

**SSR on forward**
- McGuire and Mara's 2D-DDA screen-space ray tracing: about 25 iterations per pixel at 1080p took under 1 ms on a GeForce Titan and about 5 ms on a GeForce 650M. It is BSD 2-clause. It needs depth (linear camera Z preferred) and a thickness value. Off-screen and occluded rays miss, so a cube map fallback is needed. Infinite thickness causes streaks. Temporal filtering of the result was unsatisfactory in the authors' experiments. The code contains non-standard `int2` and needs adapting — [Casual Effects](https://casual-effects.blogspot.com/2014/08/screen-space-ray-tracing.html)
- SSR does not strictly need a G-buffer. flutter_scene reconstructs view-space positions and normals from a depth pre-pass and notes this fits forward rendering — [flutter_scene docs](https://pub.dev/documentation/flutter_scene/0.19.0/scene/ScreenSpaceReflectionsSettings-class.html). Older Unity docs instead restrict SSR to deferred because it relies on the normals G-buffer — [Unity 5.6 manual](https://docs.unity.cn/560/Documentation/Manual/PostProcessing-ScreenSpaceReflection.html)
- Hi-Z pitfalls from the GPU Pro 5 chapter (forum): the sample code was reported incorrect, and ray marching did not work correctly on non-power-of-two resolutions — [GameDev.net](https://www.gamedev.net/forums/topic/658702-help-with-gpu-pro-5-hi-z-screen-space-reflections/page-4?page=6)

**Order**
- Unity HDRP's documented order: NaN killer, TAA/SMAA, DoF, motion blur, Panini, bloom (pyramid), LUT baking, lens flares, lens distortion, chromatic aberration, bloom apply, vignette, colour grade apply, FXAA, film grain, 8-bit dither — [HDRP execution order](https://docs.unity3d.com/Packages/com.unity.render-pipelines.high-definition@16.0/manual/rendering-excecution-order.html)
- The same page does not list tonemapping or upscaling. SMAA placed before colour grade conflicts with the SMAA header's guidance that input should be gamma-corrected (above), so HDRP's placement should not be assumed for moonBASIC without checking how HDRP encodes its input.
- A third-party mirror of Filament's docs lists SSAO, lighting, SSR, TAA, DoF, bloom, grading and tonemapping, vignette, dither, then FXAA last. The mirror is now gone (HTTP 410) and was not verified against Filament's repo — [Filament mirror](https://www.mintlify.com/google/filament/rendering/post-processing)
- Community opinion disagrees on AA first or last. These are forum posts, not authorities — [GameDev.net thread](https://www.gamedev.net/forums/topic/595971-order-of-post-processing-passes-in-pipeline/)
- FSR 1 goes after anti-aliasing and tonemapping, on [0,1] perceptual input, and before noise-adding effects such as film grain and chromatic aberration, and before UI — [AMD FSR 1.2 guide](https://gpuopen.com/manuals/fidelityfx_sdk/techniques/super-resolution-spatial/). Non-antialiased input likely gives poor results, because hard edges get exacerbated — [AMD naming guidelines](https://gpuopen.com/fidelityfx-naming-guidelines/)

**Current moonBASIC state (Repo, read from code)**
- `fxaa()` in [postfx_shaders.go](../../internal/runtime/postfx_shaders.go) averages the pixel and four neighbours when contrast exceeds 0.08. It has no edge direction, edge search, or sub-pixel logic, so it is a gated blur rather than FXAA.
- It runs at line 239, before SSAO, bloom and `tonemap()` at line 272, on the HDR RGBA16F scene.
- Order in the single pass: chroma aberration, FXAA, AO, bloom add, tonemap, encode, grade, LUT, outline, sharpen, vignette, grain, retro, rain.
- Outlines are drawn after AA and so stay aliased, softened only by a `smoothstep` on depth jump.
- Sharpen samples unprocessed scene texels, tonemapped only in linear mode. In gamma mode it compares graded output with raw scene colour, so the sharpen may shift brightness and tint rather than only edges.
- [fsr.go](../../internal/runtime/fsr.go) runs EASU and RCAS after this whole pass, so grain, retro palette and rain get upscaled and sharpened when dynamic resolution is active, contrary to AMD's guidance.
- The bloom chain already has five levels ([postfx.go](../../internal/runtime/postfx.go)), composited with fixed weights in the blit. [docs/POSTFX.md](../../docs/POSTFX.md) describes a bright extract followed by four smaller passes. So "half-resolution bloom" undersells it, but it still uses a threshold-style extract rather than a progressive upsample.

### Inferences

**Ranking by visual gain for cost**

| Rank | Change | Cost | Gain | Notes |
|---|---|---|---|---|
| 1 | Fix order: real AA after tonemap and LUT; move grain, retro, rain after FSR | Near zero runtime, small refactor | High | Needs splitting the single blit into two passes. Fixes AA on bright edges, including the sky and emissive edges typical in stylized scenes |
| 2 | Replace `fxaa()` with SMAA 1x MEDIUM or HIGH, keeping FXAA 3.11 as the cheap tier | About 1 ms on old hardware, 2 small textures | High | Low-poly scenes have long straight edges, which is exactly where morphological AA excels and a gated blur does worst |
| 3 | Move outline before AA | Near zero | Medium to high | Then AA smooths the line too |
| 4 | Normal- or ID-assisted outlines | One extra attachment or pass | High for toon | Depth alone misses creases and contact lines on flat-shaded low-poly |
| 5 | 13-tap and tent progressive bloom, no threshold, Karis on the first mip | Slightly above current | Medium to high | More stable with less flicker. Dual filter is the option for low-end tiers |
| 6 | MSAA in Classic and Stylized, resolved after tonemap where possible | Memory and bandwidth | Medium | Forward can use it, and moonBASIC already does. Resolve in HDR causes the same bright-edge problem as HDR FXAA. Per-sample tonemap is not practical in this stack, so tonemap-weighted resolve is the realistic fix |
| 7 | Half-resolution DoF, simple CoC blur first | About 1-3 ms (judgement) | Medium, situational | Tilt-shift DoF on a low-poly diorama is a very strong look for its cost |
| 8 | TAA | Velocity pass plus history | High for sub-pixel and specular aliasing, negative for crisp styles | Advanced profile only. Off for palette, dither and hard-edge toon |
| 9 | Per-object motion blur | Velocity plus tile passes | Low for stylized | Later. Camera-only blur from depth reprojection is cheap and can come first |
| 10 | SSR | Up to about 5 ms on low-end GPUs per the McGuire numbers | Low overall | Only worth it for water and wet floors in Advanced. Planar reflection or cube map is better for a stylized water plane, and moonBASIC already has a water FBO |

**Recommended order on a single-sample HDR forward path**

Each line is a pass or group, top to bottom.

1. Depth pre-pass or depth resolve (optional normals and velocity).
2. GTAO, ideally used inside the forward lighting pass on ambient only, otherwise as a post multiply.
3. Opaque forward with sky.
4. Copy depth and colour for water and refraction.
5. Transparent and water.
6. NaN and inf clean, as moonBASIC already does.
7. Optional SSR composite, then TAA in HDR with a tonemap-weighted history, if enabled.
8. DoF, then motion blur. The HDRP order is DoF first, so the streak is not defocused.
9. Bloom downsample and upsample, mixed in HDR.
10. Exposure, tonemap, encode to perceptual.
11. Grade, then LUT.
12. Outlines, so they are not bloomed, tone-shifted or graded. Or place them before bloom if a glowing ink line is wanted, as a per-style option.
13. Post AA: SMAA or FXAA. Skip if TAA is on, or keep a light SMAA for edges.
14. FSR 1 EASU and RCAS, or a plain sharpen when not upscaling.
15. Vignette and chromatic aberration (AMD says aberration after FSR).
16. Retro palette and dither, in display-resolution pixels or a chosen virtual pixel size.
17. Film grain and 8-bit dither, then UI.

**Notes on the order**
- Vignette multiplied before or after FSR is low risk, but it fits with the other final effects.
- Retro modes should ideally render at their own low internal resolution and skip FSR altogether. EASU would smooth away intended pixels.
- 8-bit output dither is a cheap and easy gain for gradient skies, which band badly. It matters more for stylized gradient-heavy scenes than it looks, so consider moving it up the roadmap.

**GL 3.3 feasibility**
- All of these fit fragment-only passes and FBOs: no compute shaders, image load/store or SSBOs are needed.
- `GL_R11F_G11F_B10F` and `RGBA16F` are renderable in 3.3. moonBASIC currently uses RGBA16F per docs/POSTFX.md, and `R11F_G11F_B10F` for bloom would save memory and has no alpha.
- Hi-Z needs a mip chain built with a fragment pass. Follow the 2D-DDA approach first.
- TAA and SMAA T2x need velocity. On forward that means an MRT attachment or a separate velocity pass for moving objects, plus depth-reprojected camera velocity for static ones. A depth-only reprojection covers the static low-poly environment case at low cost.
- MSAA plus velocity or normal MRT is awkward because averaging those on resolve is wrong. A single-sample pre-pass avoids this at the price of an extra geometry pass, which is cheap for low-poly scenes.

**Artifacts that matter specifically for style mixing**
- TAA jitter breaks vertex-snapped PS1 looks and hard-threshold cels, which shimmer or smear, and it softens palette output unless the palette is applied afterwards. A palette applied after TAA is safe.
- FXAA and SMAA read dither patterns as edges and may blur them. Do retro dither after post AA.
- Hard ramp bands with luma-based AA are treated as edges, which is mostly beneficial.
- Bloom before palette quantization produces clean bands. Afterwards it reintroduces colours outside the palette. moonBASIC's existing order (bloom first, retro later) is right here.

### Gaps
- No controlled, same-hardware benchmark of FXAA, SMAA and TAA was found. Modern GL 3.3 timings would need measuring in the moonBASIC bench on the user's GPU.
- No primary source could be retrieved for Karis's slides directly (cited through implementers), or for Abadie's scatter-as-gather DoF.
- Jimenez's actual slide deck timings were not read. The page has no numbers, and the deck itself was not fetched.
- Filament's real post-process order was not verified against its source (`PostProcessManager`). Only a since-removed mirror was seen.
- Whether HDRP's early SMAA runs on tonemapped or log-encoded input was not determined.
- I did not run or profile moonBASIC. The Repo findings come from reading code. Line numbers reflect the current uncommitted working tree.

## What is a good material interface: Filament's material model, glTF metallic-roughness and extensions, and MaterialX ideas for a future material editor?

### Takeaway
Use glTF 2.0 metallic-roughness as the core field set, because assets already arrive that way, and use Filament's names and semantics where they extend it. Add a small stylize block, and leave all layered effects as optional and costed. For the editor, adopt the MaterialX idea (typed nodes compiled to generated GLSL) but not the library itself as a runtime dependency, since its GLSL targets are reported as 4.0 or higher.

### Cited Findings

**Filament**
- Standard parameters: `baseColor`, `metallic` (should be 0 or 1 in practice), `roughness`, `reflectance` (avoid below 0.35), `sheenColor` and `sheenRoughness`, `clearCoat` and `clearCoatRoughness` (noted as costly), `anisotropy` and `anisotropyDirection`, `ambientOcclusion`, `normal`, `emissive` (RGB in nits, alpha as exposure weight) — [Filament Materials](https://google.github.io/filament/main/materials.html)
- Layer order, outside to inside: clear coat, then iridescence, sheen, second specular, specular, diffuse — [Filament Materials](https://google.github.io/filament/main/materials.html)
- Blending modes: `opaque`, `transparent`, `fade`, `add`, `masked` (with `maskThreshold` default 0.4), `multiply`, `screen`, `custom`. `postLightingBlending` controls how `postLightingColor` combines with the lit result — [Filament Materials](https://google.github.io/filament/main/materials.html)
- Definition structure: a `material` block (with `requires`, `parameters`, `constants`, `variantFilter`), a required `fragment` block and an optional `vertex` block. Vertex domains are `object`, `world`, `view` and `device`. `matc` compiles definitions into packages per platform and API — [Filament Materials](https://google.github.io/filament/main/materials.html)

**glTF**
- Ratified material extensions: `anisotropy`, `clearcoat`, `dispersion`, `emissive_strength`, `ior`, `iridescence`, `sheen`, `specular`, `transmission`, `unlit`, `variants`, `volume`. Ratified texture extensions: `KHR_texture_basisu`, `KHR_texture_transform` — [glTF extensions registry](https://github.com/KhronosGroup/glTF/blob/main/extensions/README.md)
- In progress: `KHR_materials_diffuse_transmission` (release candidate), `KHR_materials_subsurface` and `KHR_texture_procedurals` (initial draft), `EXT_texture_procedurals_mx_1_39` (review draft). `KHR_materials_pbrSpecularGlossiness` is archived — [glTF extensions registry](https://github.com/KhronosGroup/glTF/blob/main/extensions/README.md)

**MaterialX**
- The standard library has a BxDF graph library (including `gltf_pbr.mtlx`, `standard_surface.mtlx`, `usd_preview_surface.mtlx`) and pbrlib language-specific implementations, with GLSL, OSL, MDL and MSL targets — [MaterialX libraries](https://github.com/AcademySoftwareFoundation/MaterialX/blob/v1.38.4/libraries/bxdf/gltf_pbr.mtlx), [Blender bundle](https://projects.blender.org/blender/lib-linux_x64/src/tag/v4.1.0/materialx/libraries)
- GLSL target support is noted as version 4.0 or higher in a Blender-bundled copy (search excerpt) — [Blender bundle](https://projects.blender.org/blender/lib-linux_x64/src/tag/v4.1.0/materialx/libraries)
- MaterialX 1.39.0 adds an OpenPBR Surface definition with example materials — [MaterialX 1.39.0 release](https://github.com/AcademySoftwareFoundation/MaterialX/releases/tag/v1.39.0). OpenPBR is hosted by ASWF as a MaterialX subproject — [Autodesk docs](https://help.autodesk.com/cloudhelp/ENU/MayaCRE-Tech-Docs/Nodes/openPBRSurface.html)
- A 2022 PR maps Standard Surface to glTF PBR through a node graph. Coat maps to clearcoat, transmission maps only in part, and iridescence was not supported by the `gltf_pbr` nodedef at that time — [MaterialX PR 1064](https://github.com/AcademySoftwareFoundation/MaterialX/pull/1064)
- A draft glTF path to procedural textures through MaterialX exists (`EXT_texture_procedurals_mx_1_39`, above), and glTF-to-MaterialX work is collected in [glTF_MaterialX](https://www.github.com/kwokcb/glTF_MaterialX)

### Inferences

**Suggested moonBASIC material fields, by tier**

| Tier | Fields | Source of semantics | Cost |
|---|---|---|---|
| Core (always) | `baseColor` + texture, `metallic`, `roughness`, MR texture, `normal`, `occlusion`, `emissive` + `emissiveStrength`, `alphaMode` (opaque, mask, blend), `alphaCutoff`, `doubleSided`, UV transform | glTF core, `KHR_materials_emissive_strength`, `KHR_texture_transform` | Base shader |
| Common extras | `blend` modes (add, multiply), `vertexColor` use, second UV for lightmap | Filament blend names | Small |
| Stylize | `shading` selector, `ramp`, `shadowTint`, `rimColor`, `rimPower`, `specularSteps`, `flatShade`, `outlineMask`, `ditherFade` | moonBASIC-specific | Small |
| Advanced optional | `clearCoat`, `sheen`, `anisotropy`, `ior`, `specular` | glTF `KHR_materials_*`, Filament layering | Each is a variant. Only compile when used |
| Later | `transmission`, `volume`, `iridescence`, `dispersion` | glTF | Needs scene-colour copy or more. Low priority for low-poly |

**Interface principles**
1. Naming: use glTF and Filament names exactly, so importers are a direct map and documentation can be shared. A BASIC command surface can stay small: `SetMaterial mat, "roughness", 0.4`, or typed setters.
2. Reflectance: use `reflectance` only in Advanced, default 0.5 (F0 0.04), which also matches the glTF default IOR of 1.5.
3. Emissive: Filament's nits with exposure weighting is overkill for stylized scenes. glTF's unitless `emissive * emissiveStrength` matches how bloom is used here.
4. Import: read `KHR_materials_unlit` to set `shading=unlit`. Round-trip the stylize block as a custom `MOON_materials_stylize` extension, so glTF stays valid for other tools.
5. Editor: a graph that compiles to a generated `material()` GLSL function which fills the same `MaterialInputs` struct, with the lighting model chosen separately. This is the same split as Filament's `material()` and `surfaceShading()`. It means a graph authored once works under all three profiles, and generated code targets `#version 330 core`.
6. MaterialX ideas worth copying: typed nodes with signatures separate from per-language implementations, and a graph-defined surface such as `gltf_pbr`. Start with a small node set (texture, UV, math, mix, ramp, noise, time, world position, fresnel, depth fade). Export to `.mtlx` can come later as a one-way bridge.
7. Do not base the core interface on OpenPBR or Standard Surface. They are film-oriented uber-shaders with many parameters, which is more than a beginner BASIC language needs. Map them on import only if there is demand.

### Gaps
- The glTF 2.0 core spec text was not fetched in this session. Field names in the Core row are from general knowledge of that spec and should be verified against the [glTF 2.0 spec](https://registry.khronos.org/glTF/specs/2.0/glTF-2.0.html) before implementation, as should the default IOR and F0 values.
- Filament's post-process material domain was not found in the sections read, so I cannot describe it.
- MaterialX GLSL generation for a 3.3 target was not tested. The 4.0 claim comes from one search excerpt of an older bundled copy, and a current `GlslShaderGenerator` version check is needed. An exact count of nodes needed for a useful beginner editor was not found.
