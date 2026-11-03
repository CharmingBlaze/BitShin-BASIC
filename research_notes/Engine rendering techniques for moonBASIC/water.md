# Real-time water for moonBASIC (OpenGL 3.3 core, Go on G3N)

Reading guide. Claims about moonBASIC come from reading the working tree and cite file and line. Claims about techniques cite a URL. Almost no public source gives frame-time numbers, so every "cost" in the ranking is my estimate and sits under Inferences. Sources marked (snippet) were seen only as search summaries, not read in full.

## Objective: what to do first, ranked by visual gain for cost

### Takeaway
The biggest wins are not new wave techniques. They are removing the refraction redraw, fixing target sizes, giving the Low tier a sky to reflect, replacing the O(N^3) DFT with a real FFT, and using exponential absorption. All of these work on GL 3.3 fragment shaders.

### Cited Findings: current moonBASIC state
- Each water pass calls `rend.Render` on the whole scene root. The only per-entity culling is hiding objects on the wrong side of the plane, plus terrain chunks wholly on the wrong side — `internal/runtime/water.go:1004-1038`, `:1067`.
- Reflection and refraction targets are square: Medium 512/512, High 1024/1024, Ultra 2048/1024. Low has neither — `internal/runtime/quality.go:42-45`, `water.go:1109-1116`.
- `autoTier` sends Intel "HD Graphics" to Low — `quality.go:68-69`. On Low, `reflectCol` is one fixed colour and there is no depth, so no surf line — `water_shaders.go:330`, `:400`.
- The wave tile is 64x64 over 32 m, a separable inverse DFT that loops all 64 samples in each fragment of each of 2 passes. Targets are RG32F then RGBA16F, with a CPU fallback — `water_fft_gpu.go:13`, `:47`, `:67`, `:120-121`, `water_fft.go:22`, `:115`.
- It stores height only, with no horizontal displacement channels, and per-pixel slope fades out between 25 and 140 m — `water_shaders.go:31`, `:240`.
- Absorption is `smoothstep(0.4, 6, meters)` between two colours, not an exponential — `water_shaders.go:306`, `:318`.
- `meters` is the difference of linearised eye depths, so it measures depth along the view axis and not vertical depth — `water_shaders.go:253-263`.
- Crest foam is an instantaneous function of absolute wave height, with no history — `water_shaders.go:392-397`.
- The `WaterSSROn` path marches 8 steps inside the planar reflection texture with no depth test and keeps only the last sample. It is opt-in — `water_shaders.go:334-356`, `water.go:1489`.
- Fresnel is Schlick with F0 = 0.02 (correct for water), but is then scaled by `clamp(refl, 0.35, 1)`, and by 0.22 at the shore — `water_shaders.go:303-304`, `:362`.

### Inferences: ranked list
Cost is my estimate of added GPU time at 1080p on an integrated GPU. Negative means a saving.

| # | Change | Visual gain | Cost | Notes |
|---|--------|-------------|------|-------|
| 1 | Refraction from a copy of the main pass, not a redraw | None to small | Large saving (one whole scene) | See reflections section; needs a G3N pass split |
| 2 | Cap and aspect-match the reflection target | None visible under distortion | Large saving on Ultra | 2048 square exceeds a 1080p screen for a wobbling image |
| 3 | Simplified reflection scene (distance cap, no scatter, no small props) | Small loss | Large saving | Add a per-entity "in reflection" flag |
| 4 | Sky cubemap or analytic sky for Low and for far pixels | Large on Low | Very low | One lookup; Low currently reflects a constant colour |
| 5 | Beer-Lambert absorption by per-channel exponential | Large | About zero | Drop-in for the smoothstep |
| 6 | Vertical depth instead of view-axis depth | Medium | About zero to low | Stops shore colour changing with view angle |
| 7 | Remove the Fresnel scale, keep an art tint | Medium | Zero | Lets the horizon reach full reflection |
| 8 | Radix-2 FFT passes replacing the DFT | Enables 128-256 tiles | Saving at 64 | 49k vs 524k loop iterations per frame |
| 9 | Crest scattering from wave height and sun | Large | Very low | Sea of Thieves recipe |
| 10 | Foam with feedback buffer | Large | Low | One small R8 ping-pong |
| 11 | Baked shore distance field | Large on coasts, enables Low | Load-time, one sample | Terrain heightmap already exists |
| 12 | 2-3 incommensurate cascades | Large (breaks tile) | Low to medium | Replaces most tiling issues |
| 13 | Normal variance into sun lobe at distance | Medium, fixes shimmer | Very low | Bruneton idea, cheap approximation |
| 14 | Half-rate reflections | Risk of lag | Saving | Do after 1-3 only if needed |
| 15 | Flow maps | Rivers only | 5 samples | Skip for open sea |
| 16 | Wave particles, shallow-water sim | Interaction | Medium to high | Later, optional |
| 17 | True screen-space planar reflections | Same as planar | Needs compute for the standard form | Optional GL 4.3+ tier |

### Gaps
- No frame-time profile of the current passes. Measure before ordering 1-3. The renderer already records 780M timings in `quality.go:34` comments, so that machine is a usable baseline.

## What makes water look good?

### Takeaway
Fresnel sets the balance of sky against body colour. Beer-Lambert absorption gives depth its colour. A crest-scatter term gives the glowing wave look. Foam and a correct sun lobe sell scale. Reflection source matters less than people expect at distance, because waves scramble it.

### Cited Findings
- Tessendorf's shading model has a Fresnel reflectivity term weighting light reflected from the sky and a transmissivity term weighting light from below the surface — [Tessendorf course notes](https://jtessen.people.clemson.edu/reports/papers_files/coursenotes2002.pdf).
- GPU Gems 2 uses bias 0.2 and power 5 in its approximation, and uses different bump strengths for refraction (0.075) and reflection (0.02) — [GPU Gems 2 ch. 19](https://developer.nvidia.com/gpugems/gpugems2/part-ii-shading-lighting-and-shadows/chapter-19-generic-refraction-simulation).
- Beer-Lambert reduces intensity exponentially with distance travelled and an attenuation coefficient — [Darles et al. survey](https://arxiv.org/pdf/1109.6494).
- A browser water build applies per-channel extinction, so red fades within metres and blue persists, and masks the sky so the far-plane backdrop is not fogged — [Cinevva](https://app.cinevva.com/blog/2026-05-12-open-world-browser-part-26-water).
- For one water type the coefficients are constant vectors that can be tuned by hand, and values above 1 give very foggy water — [GameDev.net thread](https://gamedev.net/forums/topic/527145-rendering-natural-waters-coefficients-conversion/) (snippet).
- Sea of Thieves blends deep colour and subsurface colour by view angle, sun direction and a wave peak mask. The mask comes from FFT chop offsets, because peaks have a shorter light path through water — [Ang et al. 2018](https://history.siggraph.org/wp-content/uploads/2022/09/2018-Talks-Ang_The-Technical-Art-of-Sea-of-Thieves.pdf).
- The same game uses an area specular highlight (closest-point-on-sphere) so a low sun gives a large reflection, and a Snell's window from below — [Ang et al. 2018](https://history.siggraph.org/wp-content/uploads/2022/09/2018-Talks-Ang_The-Technical-Art-of-Sea-of-Thieves.pdf).
- Bruneton et al. simplify geometry per viewing distance and encode lost detail into the normal and BRDF, with the slope distribution treated as Gaussian — [paper page](https://evasion.inrialpes.fr/Publications/2010/BNH10), [slides](https://cgg.mff.cuni.cz/%7Ejaroslav/teaching/2012-npgr031/prezentace/Realtime%20Realistic%20Ocean%20Lighting.pdf).
- GPU Gems 1 corrects a cubemap lookup by intersecting the eye ray with a sphere around the pool, at about five arithmetic operations, so near shore objects reflect at a plausible distance — [GPU Gems 1 ch. 1](https://developer.nvidia.com/gpugems/gpugems/part-i-natural-effects/chapter-1-effective-water-simulation-physical-models).
- A magnitude-weighted blend of planar reflection and procedural sky wrongly erased dark reflections and was removed — [Cinevva](https://app.cinevva.com/blog/2026-05-12-open-world-browser-part-26-water).

### Inferences
- Absorption replacement for `water_shaders.go:318`, with `d` the path length through water: `vec3 T = exp(-sigma * d); refr = mix(inscatter, sceneColor, T);`. Red largest in `sigma`. `sigma` and `inscatter` become the artist-facing colour controls, and can be derived from the existing `WaterColor` and `WaterShallow`.
- Use the true path length for T. For seabed colour, vertical depth plus view path both matter; the common cheap form is `d = verticalDepth / max(NdotV, 0.1)`.
- Crest scatter sketch: `peak = saturate(waveHeight / WaterPeak); s = peak * pow(saturate(dot(viewDir, -L)), 4) * (1 - NdotV)`, added in `inscatter`. It works now, because `waveHeight` already exists at `:305`. This is my sketch of the Sea of Thieves description, not their code.
- Keep Fresnel pure and remove the multiply at `:362`. `CreateWater` starts at 0.88 (`water.go:1163`), but a preset at 0.42 (`water.go:537`) caps horizon reflection at 42%, which reads as a flat band and hides sky colour. To avoid breaking tuned games, gate it on `LinearLight`, as the normal fix did at `:227`.
- The bug-fix guard at `:227` runs before detail normals are added. After perturbation, `NdotV` can again go below 0 and clamp at `:302` to a grazing value. That gives sparkle on far wave backs. Re-orient after perturbation, e.g. `n = normalize(n + viewDir * max(0, 0.05 - dot(n, viewDir)))`, or compute Fresnel on a blend of the macro and detail normals.
- `waveNormal` detail at `:284` takes `detail.x, detail.z` as horizontal slope. That matches the built-in map, which stores Y-up in G (`water.go:894-913`). A standard tangent-space map passed to `SetWaterNormalMap` has Z-up in B, which would add a constant tilt to every normal. Worth a check or a documented format.
- Reflection source by distance: planar near, sky far. See the reflections section.

### Gaps
- No trustworthy RGB per-metre coefficient table found. The [Jerlov water type work](https://diglib.eg.org/bitstream/handle/10.2312/stag20221264/119-121.pdf) (snippet) is the best lead for physically grounded values; tune by eye until then.

## How do you get detail without tiling?

### Takeaway
Run several tiles at unrelated scales, drive them with a true FFT at 128-256 on fragment shaders, and hide leftovers with distance-faded normal scrolls. Sea of Thieves shows FFT works down to integrated GPUs. Public detail on Assassin's Creed is thin.

### Cited Findings
**FFT**
- Tessendorf: grids from 16 to 2048, with 128 to 512 sufficient for many situations. Waterworld and Titanic used 2048 at about 3 cm spacing — [course notes](https://jtessen.people.clemson.edu/reports/papers_files/coursenotes2002.pdf).
- Tiling adds artificial periodicity. It stays hidden while the patch is large against the field of view, but may show as repeated structures across the view — [course notes](https://jtessen.people.clemson.edu/reports/papers_files/coursenotes2002.pdf).
- Horizontal displacement x + lambda*D sharpens peaks and broadens valleys, and can fold. The Jacobian J = Jxx*Jyy - Jxy*Jyx goes below zero in the overlap, which he suggests as a signal for foam and spray — [course notes](https://jtessen.people.clemson.edu/reports/papers_files/coursenotes2002.pdf).
- Tessendorf quantises the dispersion relation so a sequence loops after a chosen repeat time; a longer repeat is closer to the true curve — [course notes](https://jtessen.people.clemson.edu/reports/papers_files/coursenotes2002.pdf).
- Fragment-only FFT is established: glsl-rfft needs at least two float or half-float framebuffers to ping-pong — [glsl-rfft](https://www.npmjs.com/package/glsl-rfft). A Three.js ocean generates displacement and normal maps in WebGL — [jbouny/fft-ocean](https://github.com/jbouny/fft-ocean). A Panda3D ocean ping-pongs horizontal and vertical butterfly stages across log2(N) steps, but uses compute for the initial spectrum — [Panda3D forum](https://discourse.panda3d.org/t/bringing-an-ocean-to-life-in-panda3d/31503).

**How shipped games do it**
- Sea of Thieves: FFT per Tessendorf, on hardware from laptop integrated GPUs up — [Ang et al. 2018](https://history.siggraph.org/wp-content/uploads/2022/09/2018-Talks-Ang_The-Technical-Art-of-Sea-of-Thieves.pdf). Grid sizes are not in the paper.
- Crest: concentric power-of-2 geometry rings and viewer-centred cascaded textures, all one resolution (the example is 256x256, largest LOD about 4 km²). Each wave component goes to the LOD matching its wavelength, then a combine pass sums them — [Crest technical docs](https://crest.readthedocs.io/en/4.9/user/technical-information.html). Later versions moved shape generation to FFT — [Crest system notes](https://crest.readthedocs.io/en/4.23.0/user/system-notes.html) (snippet). Crest targets texture-rendered shape data, not GL 3.3 specifically.
- Crest samples normal and foam maps at two scales tied to the current and next LOD and blends by lodAlpha, and combines these with normals from displacement — [Crest technical docs](https://crest.readthedocs.io/en/4.9/user/technical-information.html).
- Crest's own performance advice: reduce wave components, disable shader features, and switch to a deep-water material offshore to skip shallow-water work — [Crest performance](https://crest.readthedocs.io/en/4.9/user/performance.html).
- A Russian ship game ocean blends three FFT textures for large, medium and small waves, with the large one setting the shape and driving physics. It outputs 3-axis offsets, not height, for sharper waves — [Habr](https://habr.com/en/articles/451604).
- Horizon Forbidden West does not use FFT for its signature feature. Breaking waves are one baked animated cross-section, swept along artist-defined wavefronts by a compute-generated vertex buffer. Rivers use 128-layer flipbook deformations from Houdini. Malan names wave particles as an inspiration, and names too few triangles as the main limit — [Malan 2022](https://advances.realtimerendering.com/s2022/SIGGRAPH2022-Advances-Water-Malan.pdf).
- Killzone 3 streamed mesh data and COD WWII used a looping Houdini sim for its sea — [Malan 2022](https://advances.realtimerendering.com/s2022/SIGGRAPH2022-Advances-Water-Malan.pdf).
- Ubisoft Singapore's GDC 2023 talk covers naval tech from Assassin's Creed III through Skull and Bones, but the abstract names no method — [GDC Vault](https://gdcvault.com/play/1029230/Making-Waves-for-Skull-and).

**Normal maps and older methods**
- GPU Gems 1 sums about 15 texture waves into a 256x256 normal map per frame, in 2-4 passes, with wavelengths of about 4 to 32 texels. It attenuates geometric waves as mesh edge length nears the wavelength (1 at 4x edge length, 0 at 2x) — [GPU Gems 1 ch. 1](https://developer.nvidia.com/gpugems/gpugems/part-i-natural-effects/chapter-1-effective-water-simulation-physical-models).
- Gerstner steepness has an upper limit; too-large Qi loops above crests. Qi = Q/(wi*Ai*numWaves) — [GPU Gems 1 ch. 1](https://developer.nvidia.com/gpugems/gpugems/part-i-natural-effects/chapter-1-effective-water-simulation-physical-models).
- GPU Gems 2 averages four scrolled bump layers — [GPU Gems 2 ch. 19](https://developer.nvidia.com/gpugems/gpugems2/part-ii-shading-lighting-and-shadows/chapter-19-generic-refraction-simulation).

**Flow maps**
- Flow uses RG direction plus two phases offset by half a cycle with triangle-wave weights, so resets never show. Noise per pixel in alpha breaks up the global pulse. Total is 5 samples per pixel: 1 flow, 2 albedo, 2 derivative-height — [Catlike Coding](https://catlikecoding.com/unity/tutorials/flow/texture-distortion/).

**Stochastic tiling**
- A 2025 thesis adds stochastic tiling with spectral variance preservation to reduce FFT periodicity — [Charles University](https://dspace.cuni.cz/handle/20.500.11956/199815) (snippet).
- Mikkelsen's hex-tiling builds on Heitz and Neyret as a practical GPU form, with a colour-texture quality trade-off — [JCGT](https://jcgt.org/published/0011/03/05/paper.pdf) (snippet). The method only suits stochastic-looking textures — [Unity blog](https://unity.com/blog/engine-platform/procedural-stochastic-texturing-in-unity) (snippet).

**Wave particles**
- Yuksel et al.: simple, fast, unconditionally stable waves and object interactions. Graphics hardware converts particles to a height field, warped horizontally for local flow, scaling to several hundred objects — [SIGGRAPH history](https://history.siggraph.org/?p=115408) (snippet).

### Inferences
- Replace the DFT. My arithmetic, loop iterations per frame per tile:

| N | Current DFT, 2*N^3 | Radix-2 FFT, 2*log2(N)*N^2 |
|---|---------------|--------------------|
| 64 | 524,288 | 49,152 |
| 128 | 4,194,304 | 229,376 |
| 256 | 33,554,432 | 1,048,576 |

- FFT adds FBO switches: 12 passes at N=64 and 16 at 256, vs 2 today. On weak GPUs that fragment count is fine but the pass overhead is real, so at N=64 the DFT's real cost gap is below the table's 10x. Test it.
- GL 3.3 pass layout: one spectrum pass (h0 and dispersion) to RG32F; log2(N) horizontal stages; log2(N) vertical stages; a final pass packing height and slopes. Precompute a butterfly lookup texture, or compute twiddles from `gl_FragCoord`. Use NEAREST on intermediates, as `water_fft_gpu.go:120` already does.
- Pack more channels. A complex FFT carries two real outputs, so one RG target gives height + Dx, and one more gives Dz + a slope. Slopes straight from the spectrum avoid the finite differences at `water_shaders.go:237-238`, which low-pass the highest frequencies. A 32 m tile at 64 has 0.5 m texels, so no wavelength below 1 m exists.
- Cascades: tiles at e.g. 250 m, 37 m and 5.3 m (sizes without simple ratios, the choice is mine and unverified), each N=128-256. Each cascade covers only its own wavelength band, by masking the spectrum, to avoid doubled energy. Update the far cascade every 2nd or 3rd frame; update round robin.
- Vertices sample only the large cascade. Fragments combine all, with near cascades fading with distance, replacing the current 25-140 m fade at `:240`.
- Compare the cheaper path: a Tessendorf looping tile baked at load via the existing CPU tile (`water_fft.go:69`) into a `GL_TEXTURE_2D_ARRAY` with mips. 256x256 RG8 at 64 frames is 8.4 MB before mips and costs a single array lookup with lerp between frames. Works on every tier and removes GPU sim entirely, which is good for Low. Periods are visible only if the loop is short, so use a repeat around 30 s or more and spread frames over time.
- Skip Crest's ring structure for now, but copy its rule of matching wavelengths to LOD (`water_shader_test.go:104-115` shows a LOD span already). Copy the edge-length fade from GPU Gems 1 to the vertex-stage Gerstner sum.
- Horizon's approach (compute, heavy tessellation, offline Houdini) is out of reach, but its core idea survives: one authored cross-section texture applied in a vertex shader along shore distance. See the shoreline section.
- Flow maps suit rivers and estuaries; do not spend 5 samples on open sea.
- Stochastic tiling on a normal channel is plausible for far detail, but blended normals flatten; I saw no source applying it to ocean normals besides the thesis.
- A hex or stochastic lookup does not address per-frame animation coherence; test before committing.

### Gaps
- Could not read the [SIGGRAPH 2017 Crest course](http://advances.realtimerendering.com/s2017/index.html) slides or the 2019 follow-up, so details beyond the docs are missing.
- Found no public technical source on Assassin's Creed Black Flag, Odyssey or Mirage ocean internals; the [AC3 GDC session](https://gdcvault.com/play/1017710/Rendering-Assassin-s-Creed) and Skull and Bones talk are behind GDC Vault. Flow-map use in those games is unconfirmed.
- The Far Cry 5 talk was only available as an agenda listing half-precision math — [80.lv](https://80.lv/articles/the-gdc-talk-water-rendering-in-far-cry-5).
- Could not reach the Khronos wiki (connection refused) to confirm required renderable formats. RG32F and RGBA16F are already used with a failure path in `water_fft_gpu.go:95-115`, so keep that check.

## How do you make reflections cheap?

### Takeaway
Cut pixels first, then scene content, then rate. Half resolution in a window-aspect target is safe, because distortion hides it. Refraction does not need a scene redraw at all. Sky-only reflection is correct far away and for the Low tier.

### Cited Findings
- Ryzom defaults to half-resolution planar reflection, since it halves overdraw cost and bump wobble hides the loss. It sizes the target once and renders into a sub-region matching the water's screen coverage, avoiding reallocation stutter — [Ryzom wiki](https://github.com/ryzom/ryzomcore/wiki/water_planar_reflection).
- It rounds target size down to a power of two, because rounding up risks large memory spikes on big screens — [Ryzom wiki](https://github.com/ryzom/ryzomcore/wiki/water_planar_reflection).
- It biases the clip plane 0.25 m under the surface, culls underwater terrain patches, skips the water surface and vegetable blend layer, and skips shadow map generation in the reflection pass — [Ryzom wiki](https://github.com/ryzom/ryzomcore/wiki/water_planar_reflection).
- Hardware too slow to render the scene twice falls back to envmap water, at no extra cost. A budget of planes shrinks tiles rather than dropping planes — [Ryzom wiki](https://github.com/ryzom/ryzomcore/wiki/water_planar_reflection).
- BeamNG ties planar reflection to a high shader quality and uses a fallback cubemap otherwise — [BeamNG docs](https://documentation.beamng.com/modding/lighting/reflections/) (snippet).
- GPU Gems 2 needs no separate refraction pass. The scene goes to a back-buffer texture, water is marked in alpha, and distortion that lands outside water falls back to the undistorted pixel — [GPU Gems 2 ch. 19](https://developer.nvidia.com/gpugems/gpugems2/part-ii-shading-lighting-and-shadows/chapter-19-generic-refraction-simulation).
- One ocean moved from a full separate camera render to Screen Space Planar Reflections and reported a good performance boost, with no numbers — [Habr](https://habr.com/en/articles/451604).
- SSR marching was capped at 64 steps with 6 binary refinements in a browser water build, and it cannot reflect anything off-screen or occluded; the author rendered the mirrored camera instead when that mattered — [Cinevva](https://app.cinevva.com/blog/2026-05-12-open-world-browser-part-26-water).
- Half-resolution tracing with full-resolution resolve is a common SSR optimisation — [Chalmers slides](https://www.cse.chalmers.se/edu/year/2018/course/TDA361/Advanced%20Computer%20Graphics/Screen-space%20reflections.pdf) (snippet).
- A cubemap can be overkill for flat water, because most faces go unused — [Polycount](https://polycount.com/discussion/comment/2712620) (snippet).

### Inferences
Ordered by payoff. Line numbers are in `internal/runtime/water.go` unless stated.

1. **Drop the refraction redraw (`:1129-1135`).** Render opaque scene, `glBlitFramebuffer` colour and depth into textures (or sample a resolved copy), then draw water. Leak guard: sample scene depth at the distorted UV, and if it is nearer than the water fragment use the undistorted UV. G3N's `rend.Render` draws a whole tree in one call, so this needs a split: render with the water node hidden, copy, then render only water without clearing. Not verified that G3N's renderer allows this without clearing; check before starting. Fallback that avoids the split: last frame's scene texture, at the cost of a one-frame lag near moving objects.
2. **Fix target shape and size (`:1109-1116`, `quality.go:42-45`).** Square targets ignore aspect: at 1080p, High gets 1024 of 1920 across, and Ultra's 2048 square is about 4.2 MP, twice the 2.07 MP screen. Use a fraction of the viewport: Medium 1/4, High 1/2, Ultra 3/4 or less, capped. Half of 1080p on each axis is 0.52 MP, about half the 1.05 MP of the current High target.
3. **Build a reflection render list (`:1004-1038`).** Add a distance cap per entity (e.g. reflect only within 300 m, with the sky beyond), a size-over-distance cull, no grass or scatter, no decals or particles. Skip shadow map refresh, and use lowest terrain LOD. Check whether instanced scatter lives outside `w.ents`; that loop may not catch it. Add a BASIC command so scripts can flag big landmarks as always-reflected.
4. **Skip the pass when it cannot matter.** Skip if no water in the frustum, or if water pixels on screen are few (occlusion query or a rough bounding estimate). Skip when the camera is underwater. Clear to sky colour instead of the tinted fallback at `:1120`.
5. **Sky at distance.** Sky comes from the existing `skybox.go` / `envcube.go` machinery. Fragment shader: `mix(planar, skyLookup, smoothstep(far0, far1, camDist))`. Weight by distance, never by reflected colour, per the Cinevva failure. Only a low cubemap mip is needed under waves. Dynamic sky can update one face per frame at 64-128 px. Sampling the cube with a true reflect vector also fixes the "reflecting a texture" look at grazing angles.
6. **Half rate.** Alternate reflection refresh on even frames, and the refraction copy on odd if it is still a redraw. A stale target swims when the camera moves, because lookups use current screen UVs. To fix, store last refresh's mirror view-projection and project `WorldPos` with it. Simple gate: refresh every frame when the camera moved or turned fast, halve when still. I found no source testing this for water; treat as an experiment.
7. **Remove or disable `WaterSSROn` (`water_shaders.go:334-356`).** It costs 8 fetches, has no depth test and blends a sample offset from the true reflection. It is not SSR. Real SSR needs main-pass depth, and only sees what is on screen.
8. **SSPR as optional.** Its usual form needs compute and image writes; offer it only when a GL 4.3+ context exists, never as the base path. It would still miss off-screen objects. moonBASIC's planar plus simplified list is likely sufficient. Compute becoming core in 4.3 is from my background knowledge; the Khronos page would not load.

### Gaps
- No public millisecond comparisons of planar vs SSPR vs cubemap on comparable scenes.
- Half-rate planar water has no source I could find.
- G3N's clear behaviour across a split render was not checked.

## How do you do shoreline foam and wave breaking?

### Takeaway
Bake a distance-to-shore and depth texture from the terrain at load for coast structure and wave timing, and use a depth-buffer comparison for dynamic contact. Add a small feedback buffer so foam lingers. Sea of Thieves does the runtime half of this, and Horizon shows that authored wavefronts beat simulation where budgets are tight.

### Cited Findings
- Depth fade: copy scene depth to a texture and fade foam in as water-to-scene distance shrinks — [Unity forum](https://discussions.unity.com/t/water-foam-shoreline-shader/593150) (snippet). Same in [ToonWaterShader](https://github.com/favoyang/ToonWaterShader-UPM) (snippet).
- Baked options include vertex colours or a baked depth heightmap, fine for static water but harder with dynamic oceans — [Unity forum](https://discussions.unity.com/t/water-shoreline-effects/523636) (snippet).
- Unigine ships a shoreline distance field generated from a baked shoreline texture with radius, blur and downsample settings — [Unigine docs](https://developer.unigine.com/en/docs/2.12/api/library/fields/class.fieldshoreline?rlang=cs) (snippet).
- Unreal developers sample distance fields for foam within a few metres of geometry, and only where the landscape has a field — [Unreal forum](https://forums.unrealengine.com/t/tidal-water-foam/14682) (snippet).
- Depth-based foam has artefacts at object silhouettes and hard edges — [Unreal answers](https://answers.unrealengine.com/questions/764425/index.html) (snippet).
- Sea of Thieves: wave-peak foam from the Tessendorf method, plus depth-compared foam around intersecting objects inside a camera-centred window. The buffer gets progressive blur with feedback to disperse, then blends with authored textures. Calm water only foams around objects; storms produce more — [Ang et al. 2018](https://history.siggraph.org/wp-content/uploads/2022/09/2018-Talks-Ang_The-Technical-Art-of-Sea-of-Thieves.pdf).
- Habr ocean: crest foam fades by blending with previous frames, giving a trailing smear, with a mask per FFT layer combined with bubble and streak textures — [Habr](https://habr.com/en/articles/451604).
- Crest stores foam in the LOD chain and warns that shape data moves with waves, which suits foam but not depth; shallows appeared to move with the waves — [Crest technical docs](https://crest.readthedocs.io/en/4.9/user/technical-information.html) (citing [issue 96](https://github.com/wave-harmonic/crest/issues/96)).
- Waves steeper than a slope of about 1/6 eventually break — [Tessendorf](https://jtessen.people.clemson.edu/reports/papers_files/coursenotes2002.pdf).
- GPU Gems 1 uses water depth to control opacity, reflection strength and wave amplitude, fading waves to zero just above the water plane so they lap without climbing the shore — [GPU Gems 1 ch. 1](https://developer.nvidia.com/gpugems/gpugems/part-i-natural-effects/chapter-1-effective-water-simulation-physical-models).
- Horizon: an animated XYZ cross-section stored in a single 2D texture, sampled along wavefront quads (U across, V as animation time). A variation texture scales strength so waves have gaps. Foam, relative height and strength pass through vertex colour, which pops where it is coarser than triangles — [Malan 2022](https://advances.realtimerendering.com/s2022/SIGGRAPH2022-Advances-Water-Malan.pdf).

### Inferences
- **Baked shore map.** At water creation, rasterise terrain height to a texture over the water extent. Channels: R is depth, G is signed distance to the waterline (jump flood on CPU, or brute force at low size), BA is its gradient. 512² in RG8 + gradient is small. Rebuild when terrain edits. Sample with undisplaced XZ, not `WorldPos`, or the shoreline swims like Crest's issue 96. Works on Low since no refraction pass is required, and fixes the view-angle issue at `water_shaders.go:253-263`.
- **Moving surf bands.** `wave = sin(dist * k + t * w + noise)`, multiplied by a narrow ring in G, so bands march up the gradient. Vary phase along the shore with low-frequency noise so they do not ring the coast evenly, as Horizon's variation texture does. This replaces the depth-based sine at `water_shaders.go:403`, including its aliasing fix at `:406`, which becomes less necessary because `dist` is smooth.
- **Geometry response.** Shoal Gerstner in the vertex stage: amplitude falls near zero depth, steepness grows, wavelength shortens, direction bends toward `-gradient`. Cap steepness per the Qi limit. Add one authored 1D-animated cross-section lookup for a cheap breaking lip, if tessellation near shore allows; true overhang needs dense mesh, per Horizon.
- **Foam memory.** One R8 or RG8 ping-pong, 512² or 1024², mapped in the FFT tile or a camera-centred window snapped to texels. Each frame: `foam = max(source, prev * decay)`, plus a small blur. Sources: Jacobian below a threshold, or a slope/curvature proxy while no D channels exist; shore ring; depth contact and wake. Advect with low-amplitude noise. Then shade: `coverage = smoothstep(1 - foam, 1 - foam + width, foamTexture)`, which erodes foam into clumps rather than fading its alpha.
- Replace crest foam at `:392-397`, which depends on absolute height, so every tall wave gets it and calm-sea scale needs per-water tuning. Add a weather parameter so calm water only foams on contact, per Sea of Thieves.
- Use the texture's blue-noise or dedicated foam tile, not `WaterNormal.r` at `:395`, which correlates foam with slope direction.
- Keep depth compare only for dynamic objects (`contact` at `:265`), with 4 taps as now. Move object foam to a screen or world-space mask.

### Gaps
- No primary published recipe for the combined bake and depth hybrid; this is my synthesis of the sources above.
- Horizon's foam shading is named as an unsolved area in the talk, not detailed.

## What are low-end fallbacks for Intel HD class GPUs?

### Takeaway
Give the Low tier a real look with no extra passes: sky reflection through Fresnel, baked depth absorption, two normal scrolls, a sun lobe and baked shoreline. No FBOs. Today Low has constant reflection and no shore foam, so this is the highest visual gain per unit of work in the file.

### Cited Findings
- Sea of Thieves is designed to run from laptop integrated GPUs upward, using depth-tested foam and FFT water with stylisation — [Ang et al. 2018](https://history.siggraph.org/wp-content/uploads/2022/09/2018-Talks-Ang_The-Technical-Art-of-Sea-of-Thieves.pdf). Which quality settings integrated GPUs get is not stated.
- Ryzom falls back to envmap water on slow hardware, since it is the original single-pass pipeline — [Ryzom wiki](https://github.com/ryzom/ryzomcore/wiki/water_planar_reflection).
- BeamNG falls back to a cubemap when planar reflections are unavailable — [BeamNG docs](https://documentation.beamng.com/modding/lighting/reflections/) (snippet).
- GPU Gems 1 is the template: a few vertex Gerstner waves, a texture-wave normal map, a bump-mapped cubemap reflection and depth-driven opacity — [GPU Gems 1 ch. 1](https://developer.nvidia.com/gpugems/gpugems/part-i-natural-effects/chapter-1-effective-water-simulation-physical-models).
- Crest recommends turning off shader features, reducing wave components, and using a deep-water material offshore — [Crest performance](https://crest.readthedocs.io/en/4.9/user/performance.html).
- Half precision was on the Far Cry 5 talk agenda, with no detail available — [80.lv](https://80.lv/articles/the-gdc-talk-water-rendering-in-far-cry-5). 
- moonBASIC notes on a Radeon 780M that G3N's 8x multisampled window is about 3x slower, so Low keeps the post pass — `quality.go:39-41`.

### Inferences
Tier plan. Everything except GPU FFT and foam feedback draws in one pass, in the existing water draw.

| Feature | Low (Intel HD) | Medium | High | Ultra |
|---|---|---|---|---|
| Reflection | Sky only, analytic gradient or 64-128 px cube | Planar 1/4 res, simplified, sky far | Planar 1/2 res | Planar up to 3/4 res |
| Refraction | None; alpha from baked depth | Copy of main pass | Copy | Copy |
| Waves, vertex | 3-4 Gerstner, or fewer far | Gerstner + large cascade | Same, more | Same, more |
| Normals, fragment | 2 scrolls, or baked loop array | Baked loop or 1 FFT at 64-128 | 2 FFT at 128-256 | 3 FFT at 256 |
| Foam | Baked shore only | Shore + crest | + feedback | + feedback at higher res |
| Absorption | Baked depth + Beer-Lambert | Scene depth | Scene depth | Scene depth |

- Replace `reflectCol` at `water_shaders.go:330` with a three-colour gradient by reflection vector Y, or sample the existing sky cube at a low mip. Use the tier-appropriate Fresnel unchanged.
- Without refraction, draw with blending: `alpha = mix(1 - T, 1, fresnel)`, with T from baked depth. The shore thins out through alpha instead of a grabbed image. Without a bake, fall back to the GPU Gems vertex-depth approach from terrain height.
- Compute the DuDv path only when a reflection target exists; on Low it is wasted, as are the point and spot light loops at `:432`, `:450`. Compile a Low shader variant with `#define`s so these and `WaterFFT` paths drop out, instead of `if (uniform)` branches.
- Reduce sampling: 2 normal fetches, 1 baked map, 1 foam. Evaluate Fresnel per vertex when triangles are small and far.
- Avoid float render targets on this tier. No FBOs means no dependence on driver float support.
- Half-resolution water composite is possible but gives edge artefacts; try only if the shader alone proves too slow.
- Keep the frame-time auto-tier correction (`quality.go:60-61`) as a safety net.

### Gaps
- No public Intel HD water benchmark found. The tier table is a design, untested on real hardware. Test on an actual HD 4000 or UHD 620 class machine before fixing numbers.
- Driver reliability of RG32F render targets on old Intel GPUs under GL 3.3 is unverified. The existing CPU tile fallback (`water_fft_gpu.go:115`) covers failure at creation, but not slow or incorrect output.
