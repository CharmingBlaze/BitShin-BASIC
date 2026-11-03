# Sky, atmosphere, clouds and distance haze on OpenGL 3.3 core (moonBASIC / G3N)

Source caveats that apply throughout:
- Hillaire's EGSR 2020 paper could not be read directly (publisher PDF returned 403, author PDF exceeded the fetch limit). Hillaire facts come from his SIGGRAPH 2020 slides with speaker notes and his MIT reference code.
- The slide performance table on page 22 is an image, so only fragments of it survived text extraction.
- The Inigo Quilez fog page returned 404 to the fetcher, although a search index still lists it. The formula below comes from a forum quote of the article and is re-derived here.
- Local code findings refer to `C:\Users\Snow\Downloads\Modern Blitz\internal\runtime\`, read during this research on the working tree.
- All millisecond budgets marked "estimate" are unmeasured. Benchmark them on target hardware.

## Which sky models fit GL 3.3 and low-end GPUs, and what does each cost?

### Takeaway
Hillaire 2020 is the best fit: moonBASIC already has its transmittance LUT, the remaining LUTs need no high-dimensional tables, and the sky-view LUT should cost fewer shader steps than the current per-pixel march. Preetham and Hosek-Wilkie are cheap but give no aerial perspective, so they cannot solve the blue-mountain and consistency goals. Full Bruneton runs on GL 3.3 but is the heavy option.

### Cited Findings
**Current moonBASIC state (local code)**
- `atmosphere.go` precomputes a 256x64 transmittance LUT in Go, stored as 8-bit `image.RGBA`, and leaves out the 4D inscatter tables on purpose. Its comment assumes those tables typically need GL 4.x sampling; the Bruneton finding below contradicts that.
- `atmosphere_shaders.go` line 86 marches 12 single-scattering steps per sky fragment, so there is no multiple scattering.

**Why model fits fall short**
- Model fits (Perez, CIE, Preetham, Hosek-Wilkie) support only views from the ground, have a limited range of appearance and give no aerial perspective — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).
- Hosek-Wilkie describes itself as fixing most known flaws of Preetham, works as a drop-in replacement and accounts for ground albedo — [Hosek-Wilkie preprint](https://cgg.mff.cuni.cz/projects/SkylightModelling/HosekWilkie_SkylightModel_SIGGRAPH2012_Preprint.pdf) (search excerpt only; the PDF exceeded the fetch limit).
- Vendor docs say Hosek-Wilkie generally looks more realistic, especially at sunrise and sunset — [Maxon Redshift docs](https://help.maxon.net/c4d/en-us/Content/_REDSHIFT_/html/Physical%2BSky.html). Another manual calls Preetham less accurate and significantly brighter, and uses ground albedo only with Hosek-Wilkie — [Cycles4D manual](https://insydium.ltd/cycles4d-manual/html/text_sky.htm).
- Hosek and Wilkie's later fitted model adds attenuation data for atmospheres — [Wilkie et al. 2021](https://cgg.mff.cuni.cz/publications/skymodel-2021/) (seen in search results, not read in depth).

**Bruneton**
- Reference table sizes: transmittance 256x64, scattering R=32, MU=128, MU_S=32, NU=8 (a 256x128x32 3D texture), irradiance 64x16 — [Bruneton constants.h](https://raw.githubusercontent.com/ebruneton/precomputed_atmospheric_scattering/master/atmosphere/constants.h).
- The reference code uses `#version 330` throughout and refers to the OpenGL 3.3 Core Profile. It uses no compute, writing 3D layers through a geometry shader's `gl_Layer` with full-screen quads, and runs one pass per scattering order from 2 upward — [Bruneton model.cc](https://raw.githubusercontent.com/ebruneton/precomputed_atmospheric_scattering/master/atmosphere/model.cc).
- It is BSD licensed — [Bruneton repo](https://github.com/ebruneton/precomputed_atmospheric_scattering).
- Limitations are horizon LUT artifacts, a high-dimensional table that is expensive to update with cost O(n) in scattering order, and no direct soft volumetric shadows — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).

**Hillaire 2020 design**
- The distant sky is low frequency except at the horizon and in volumetric shadows, so low-resolution LUTs work. Quality scales through LUT resolution and sample count, independent of screen resolution — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).
- The multiple-scattering LUT is 2D (altitude, sun zenith) and approximates infinite orders. Its high-quality form integrates 64 directions. A low-end form uses only 2 directions (top and bottom) with the result multiplied by 2 — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).
- Multiple scattering is called critical at sunset to avoid a yellowish atmosphere — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).
- Known caveats: the LUT works well for isotropic phase functions, slowly diverges when anisotropic, and its hue can drift at very high coloured scattering — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).
- Reference constants: sky-view 192x108, up to 30 initial samples, aerial perspective 32 slices at 4 km per slice, transmittance 40 samples, multi-scattering 64 directions (`[numthreads(1,1,64)]`) — [RenderSkyRayMarching.hlsl](https://raw.githubusercontent.com/sebh/UnrealEngineSkyAtmosphere/master/Resources/RenderSkyRayMarching.hlsl).
- The multi-scattering LUT is 32 per side — [Game.h](https://raw.githubusercontent.com/sebh/UnrealEngineSkyAtmosphere/master/Application/Game.h).
- The reference code is MIT licensed, and its Dx11Base dependency suggests D3D11 rather than GL — [UnrealEngineSkyAtmosphere](https://github.com/sebh/UnrealEngineSkyAtmosphere).

**Hillaire 2020 costs**
- Slide fragments show "Render sky & atmosphere @720p = 0.14ms", "Total cost = 0.31ms" and "Lut cost = 1.03ms", with an NVIDIA 1080 label and PC and Mobile rows — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf). Extraction lost the table layout, so which number belongs to which row is unclear.
- A secondary write-up reports iPhone 6s figures: transmittance 256x64 at 0.53 ms, sky-view 96x50 at 0.27 ms, aerial perspective 32x32x16 at 0.11 ms, multi-scattering 32x32 at 0.12 ms, with 8 sky-view steps — [talk notes](https://jojo-lyu.notion.site/s2020-Scalable-Atmospheres-in-Unreal-Engine-6cbf50aae15f4701a4c01b589f63c2c1) (search excerpt, unverified against the paper).
- The paper names the iPhone 6s as its low-end device — [Hillaire 2020, Eurographics](https://diglib.eg.org/handle/10.1111/cgf14050).
- One WebGPU implementation reports about 1.1 to 2.0 ms on an M1 for its whole system, recomputing transmittance only when the sun moves — [Cinevva blog](https://app.cinevva.com/blog/2026-05-07-open-world-browser-part-22-clouds-and-meshlets) (secondary, compute-based, not stated as sky only).

**Open source to study**
- [FlightGear atmos-sky-view.frag](https://git.nayala.duckdns.org/fly/fgdata/src/commit/4d3466216a2d94eedf560186df77728ab242d80c/Shaders/HDR/atmos-sky-view.frag) is a GLSL fragment-shader port with 32 scattering samples and non-linear elevation mapping (search excerpt).
- [pl-sky](https://github.com/hoffstadt/pl-sky) is a graphics prototype of the paper's approach, not a line-by-line port (search excerpt).
- [Bevy atmosphere](https://docs.rs/bevy_pbr/latest/src/bevy_pbr/atmosphere/mod.rs.html) has separate transmittance, multi-scattering, sky-view and aerial-view shaders in WGSL (search excerpt).

### Inferences
**Costs worked out from the numbers above**
- The four iPhone per-pass figures sum to exactly 1.03 ms, so the slide's "Lut cost = 1.03ms" is probably the mobile row, not the NVIDIA 1080. Treat 0.14 and 0.31 ms as the likely 1080 numbers.
- Steps per frame favour the LUT. The current sky at 1080p with sky covering the full screen is about 2.07M pixels x 12 steps, or about 25M steps. A 192x108 LUT at 30 steps is about 0.62M, roughly 40x fewer. A 96x50 LUT at 8 steps is about 38k. Each composited pixel then pays only one bilinear fetch.
- Bruneton memory is 256x128x32, about 1.05M texels: about 8 MB in RGBA16F or 16 MB in RGBA32F, and several full passes for any change to the atmosphere. This suits a fixed atmosphere baked at load, not live tweaking.

**GL 3.3 feasibility of Hillaire (from memory of the GL 3.3 core feature set, not checked against the spec)**
- All LUTs can be fragment-shader passes into FBOs. RGBA16F and R11F_G11F_B10F are renderable, and 3D slices can be written with `glFramebufferTextureLayer` or geometry-shader `gl_Layer`.
- **Compute is optional.** The reference uses it only for the multi-scattering LUT, whose per-texel loop can run as a fragment loop.
- The 32x32 LUT could also be built on the CPU in Go, like the existing transmittance LUT, on the 2-direction path, which avoids any GPU spike.
- The 8-bit transmittance LUT in `atmosphere.go` is worth moving to 16F, since RGBA8 quantises the small transmittance values seen near the horizon at sunset.

**Update strategy**
- Transmittance and multi-scattering change only when atmosphere parameters change (weather, turbidity).
- Sky-view changes with sun direction and camera altitude, so it can run every N frames on low tiers. The sun moves slowly, and the walking camera barely changes altitude relative to the 80 km shell.

**Proposed tiers (estimates)**

| Tier | Transmittance | Multi-scattering | Sky-view | Aerial perspective |
|---|---|---|---|---|
| Low | Existing, CPU built | 32x32, 2 directions, CPU built | 96x50, 8 to 12 steps, every 4 to 8 frames | Analytic fog only |
| Mid | Same | 32x32, 16 directions, on change | 192x108, 16 to 30 steps, every frame | Analytic fog |
| High | Same | 32x32, 64 directions | 192x108 or larger | 32x32x32 volume |

**Overall ranking by visual gain per cost (author's judgement, costs are estimates)**

| Rank | Technique | Gain | Added cost |
|---|---|---|---|
| 1 | RGB analytic aerial perspective coloured from the sky | Blue mountains, sky and world match | One LUT fetch, two exponentials per lit pixel |
| 2 | Multi-scattering LUT plus sky-view LUT | Correct sunsets, and the sky gets cheaper | Likely net saving at 1080p |
| 3 | Sun disc, limb darkening, HG glow, transmittance tint | Sharp sun, correct sunset colour | Negligible |
| 4 | SH L2 sky ambient with ground bounce | Lighting follows time of day | Small fixed uniforms, update off-frame |
| 5 | HDR GPU sky cube, one face per frame | Reflections follow the sky, no CPU stall | Under 0.5 ms |
| 6 | Better 2D clouds: weather map, Beer-powder, HG, light taps, shadows | Much better clouds with no raymarch | About 0.2 to 0.5 ms |
| 7 | Screen-space light shafts, half or quarter res | Dramatic low sun | About 0.3 to 1 ms |
| 8 | 3D aerial perspective LUT | Only needed for shadowed or strongly sun-dependent haze | Small but more plumbing |
| 9 | Volumetric clouds, half or quarter res with reprojection | Best sky, high tier only | About 2 ms or more on a strong GPU, plus TAA-like plumbing |
| Skip | Full Bruneton, Hosek-Wilkie swap, compute froxels | Little beyond the above for this engine | |

### Gaps
- No direct GPU comparison of Preetham, Hosek-Wilkie and LUT skies was found. From memory, both analytic models are a few `exp`/`pow` terms per pixel or vertex plus CPU coefficient setup (about 5 Perez coefficients for Preetham, about 9 per channel for Hosek-Wilkie). Verify before quoting.
- The paper's Section 7 tables, including any PS4 and PC breakdown and its exact transmittance size, were not read.
- No measurements exist on Intel integrated GPUs or other typical low-end GL 3.3 hardware.

## How do you do aerial perspective properly, and keep it consistent with the sky?

### Takeaway
Replace the scalar, gamma-space fog with per-channel Beer-Lambert optical depth integrated analytically along the view ray, using the sky's own Rayleigh and Mie coefficients, and take the in-scatter colour from the sky LUT. Mountains then fade blue and merge into the horizon by construction. A froxel volume is a later upgrade and does not need compute unless it holds local lights.

### Cited Findings
**Current moonBASIC state (local code)**
- `shadow_shaders.go` lines 396 to 411 and the matching `pbr_shaders.go` code compute one scalar factor from linear, exp or exp2 fog.
- They multiply by a height term evaluated only at the shaded point, `exp(-max(WorldPos.y - FogHeight, 0) * FogHeightFalloff)`, ignoring the camera height and the path between. Colour comes from a user `FogColor`.
- `fogTint` in `pbr_shaders.go` adds fixed `pow(s,8)` and `pow(s,64)` sun lobes.
- `fogMix` in `linear_light.go` blends in gamma space.
- None of this reads the atmosphere LUT, so sky and fog match only if tuned by hand.

**Quilez analytic height fog**
- The GL form is `fogAmount = c * exp(-rayOri.y*b) * (1.0-exp(-distance*rayDir.y*b)) / rayDir.y` — [MonoGame forum quoting the article](https://community.monogame.net/t/solved-height-distance-based-fog-again/14371); original at [iquilezles.org](https://iquilezles.org/articles/fog/) (listed by search, 404 when fetched).
- The same thread raises an unconfirmed claim that the first term should use the target's height, plus a division-by-zero worry and an up-axis question. Its original poster reported that the article's Y-up form was fine after retuning constants — [MonoGame forum](https://community.monogame.net/t/solved-height-distance-based-fog-again/14371).
- A library port clamps the result and falls back to plain exponential fog when density or falloff is not positive — [LYGIA fog.glsl](https://git.gay/kaneraZh/lygia/src/branch/main/lighting/raymarch/fog.hlsl) (search excerpt).
- The general idea of integrating density analytically along a segment is explained at [Analytic Fog Density](https://blog.demofox.org/2014/06/22/analytic-fog-density/).

**Hillaire aerial perspective**
- A froxel volume mapped to the camera frustum stores luminance in RGB and greyscale transmittance in alpha, and applies to opaque and translucent surfaces — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).
- The reference uses 32 slices at 4 km each, or 128 km — [RenderSkyRayMarching.hlsl](https://raw.githubusercontent.com/sebh/UnrealEngineSkyAtmosphere/master/Resources/RenderSkyRayMarching.hlsl). The mobile figure reported above is 32x32x16 — [talk notes](https://jojo-lyu.notion.site/s2020-Scalable-Atmospheres-in-Unreal-Engine-6cbf50aae15f4701a4c01b589f63c2c1) (secondary).
- FlightGear stores this LUT as a tiled 2D texture with in-scattering in RGB and transmittance in alpha, with 32 slices and a 128 km range in one version — [atmos-aerial-perspective.frag](https://git.nayala.duckdns.org/fly/fgdata/raw/commit/e945dafca39f76c3e37f70df9299c69ed95dbdec/Shaders/HDR/atmos-aerial-perspective.frag) (search excerpt; the direct fetch failed).
- Volumetric shadows can be stored in the sky-view and aerial LUTs when per-pixel tracing is too expensive — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).

**Froxel volumetric fog**
- It originates with Wronski's 2014 compute-based method — [GDC Vault AC4 talk](https://gdcvault.com/play/1020397/Assassin-s-Creed-IV-Black). Godot's version writes per-cell attributes and integrates with compute, and meshes then just look up the result — [Godot fog volumes article](https://godotengine.org/article/fog-volumes-arrive-in-godot-4).

### Inferences
**The forum's "use target height" claim looks wrong.** Integrating density along the ray gives camera height outside the integral. With density `d(y) = k*exp(-b*(y-y0))`:

```glsl
// x = b*rd.y*t. g(x) -> 1 - x/2 near zero, so level rays do not divide by zero.
float g(float x){ return abs(x) < 1e-3 ? 1.0 - 0.5*x : (1.0 - exp(-x)) / x; }
float depth(float k, float b, float y0, vec3 ro, vec3 rd, float t){
    return k * exp(-b*(ro.y - y0)) * t * g(b*rd.y*t);
}
vec3 od = betaR * depth(1.0, 1.0/Hr, 0.0, ro, rd, t)
        + betaM * depth(1.0, 1.0/Hm, 0.0, ro, rd, t);
vec3 T  = exp(-od * AerialScale);
color   = surface * T + skyInscatter(rd) * (1.0 - T);   // in linear space
```

Clamp `x` from below (around -20) so very long downward rays cannot overflow.

**Why this gives blue mountains**
- Using the sea-level `betaR` in `atmosphere.go` (5.8, 13.5, 33.1 e-3 per km), horizontal Rayleigh transmittance is:

| Distance | Red | Green | Blue |
|---|---|---|---|
| 10 km | 0.94 | 0.87 | 0.72 |
| 30 km | 0.84 | 0.67 | 0.37 |
| 100 km | 0.56 | 0.26 | 0.04 |

- Blue is lost from the surface fastest, and because the in-scatter weight `(1-T)` is largest for blue, the blue sky colour replaces it.
- A game world spanning 2 to 10 km gets almost no effect at physical density, so `AerialScale` of roughly 5 to 20 is needed (starting estimate). Expose it as a command, and keep scale consistent with moonBASIC's world units.

**Keeping sky and world consistent**
- `skyInscatter` should fetch the sky-view LUT with the view direction, elevation clamped to at or just above the horizon. As `T` falls to 0 along a level ray, the surface colour converges exactly to the sky behind it.
- On tiers without the LUT, read a low mip of the sky cube, or precompute one colour per 16 azimuth bins per update.
- The existing user fog command can remain as an artist override multiplied into `T`, so the classic Blitz-style `Fog` API keeps working. Retro mode keeps its stepped scalar path.
- Apply the same function to terrain, PBR, particles, clouds and the sky-horizon blend so nothing is left unfogged. Hillaire notes the atmosphere is also applied to clouds ([slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf)).

**Upgrade order**
1. Analytic RGB fog in lit shaders, about one texture fetch and two exponentials per pixel. This can be per-vertex on dense terrain.
2. A 32x32x16 3D LUT (RGBA16F) rendered by fragment shader one layer per draw, only if sun-angle variation in haze or shadowed haze is required. The fragment path keeps this on GL 3.3 with no compute.
3. Compute froxels only as an optional GL 4.3 path for local fog volumes and lights; it is unnecessary for distant haze.

**Why a 3D LUT is not first.** It needs extra passes, its 4 km slices are coarse for near fog, and moonBASIC's scale of mostly-near scenes gives it little advantage over the analytic form. Keep a dedicated ground-fog term in analytic form for low valley fog, since that has a far shorter scale height than Rayleigh.

### Gaps
- The froxel per-pass millisecond cost at 160x90x64 was not found. Wronski's slides were not read.
- No published comparison of analytic RGB fog against a true aerial LUT at the horizon was found. The convergence argument above is reasoning, not a measured result.

## How do you do clouds on a budget?

### Takeaway
Nubis's headline 2 ms came on PS4 hardware, and only after a 10x reprojection saving from about 20 ms, while Hillaire himself says clouds are not yet reasonable on mobile. So make a much better 2D layer the default on all tiers, reuse its weather map for cloud shadows and ambient, and offer half-res raymarching with 1-in-16 reprojection as a high-tier option. Everything here works in GL 3.3 fragment shaders.

### Cited Findings
**Current moonBASIC state (local code).** `terrain_clouds.go` draws one wide quad with value-noise cumulus, drifting with wind, tinted by sun colour, darkened with thickness, with an edge glow toward the sun and a fade into haze at the horizon. No texture and no raymarch.

**Nubis 2015, from the [talk notes](https://www.advances.realtimerendering.com/s2015/The%20Real-time%20Volumetric%20Cloudscapes%20of%20Horizon%20-%20Zero%20Dawn%20-%20ARTR.pdf)**
- Target is about 2 ms and 20 MB of RAM.
- Noise is a 128^3 RGBA texture (Perlin-Worley plus 3 Worley octaves) for base shape, a 32^3 RGB Worley texture for detail, and a 128^2 RGB curl noise texture for turbulence.
- A weather map holds coverage in red, precipitation in green and cloud type in blue.
- The march has 64 to 128 potential samples with 6 cone light samples per step. Cheap samples at a large step run until a cloud is hit, then step back and use full samples, and return to cheap after several empty samples. Distant light samples switch from full to cheap.
- Lighting is Beer's law, Henyey-Greenstein and a powder term for dark edges facing the light.
- The unoptimised shader cost about 20 ms. Reprojection updates 1 of 16 pixels per 4x4 block per frame using a quarter-res buffer; unreprojectable areas use low-res buffers. This made the shader "10x faster or more" when rendered at half res and upscaled.

**Later Nubis and comparisons**
- A Guerrilla publication page summarises the PS4 outcome as dynamic volumetric skyboxes rendering in under two milliseconds (search excerpt; the exact page was not pinned down, likely [Nubis Cubed](https://www.guerrilla-games.com/read/nubis-cubed) or [Nubis Evolved](https://www.guerrilla-games.com/read/nubis-evolved)). Newer work is in those talks and the [2023 deck](https://advances.realtimerendering.com/s2023/Nubis%20Cubed%20(Advances%202023).pdf) (not read in depth).
- Schneider's update is 1 pixel per 4x4 tile, with 16-frame latency but a full-res look. Bauer (RDR2) traces at quarter res with TAA upsampling, which is instant but blurrier. Hillaire's own conclusion is that only cloud rendering is not reasonably achievable on mobile for now — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).

**Cheap lighting and shadow ideas — all from [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf)**
- Multiple scattering can be approximated as octaves of single scattering (Wrenninge). Two octaves are shown brightening clouds visibly and letting light reach deeper.
- Cloud occlusion matters for sun light shafts and for top-down sky occlusion.
- The Beer Shadow Map stores front depth, mean extinction and maximum optical depth (R11G11B10F): `OpticalDepth = min(B, G*max(0, d-R))`. It avoids the ESM problem of transmittance always converging to 0.
- Lowering albedo near cloud edges gives dark edges without Beer-powder, but overuse looks like dirty cotton.
- Noise tools are at [TileableVolumeNoise](https://github.com/sebh/TileableVolumeNoise).

**Community implementations.** Reimplementers confirm a 1/16 update cost of about 2 ms of a 15 ms budget in Guerrilla's case, and rays start where the eye vector meets the lower cloud plane — [GameDev.net thread](https://www.gamedev.net/forums/topic/680832-horizonzero-dawn-cloud-system/). Treat this as anecdotal.

### Inferences
**Memory.** 128^3 RGBA8 is about 8.4 MB and 32^3 RGB8 about 0.1 MB. Bake noise offline or at load in Go, and upload with mipmaps. A 64^3 base texture (about 1 MB) is a reasonable budget option, at the cost of more visible repetition.

**Tier 0, the improved single sheet (estimate 0.2 to 0.5 ms; no raymarch)**
1. Replace value noise taps with a precomputed tileable 2D Perlin-Worley texture plus a 256^2 or 512^2 RG weather map for coverage and type, which saves per-pixel noise math.
2. Treat density as a fake thickness: `T = exp(-thickness * sigma)`, and take 2 to 4 taps along the sun direction projected onto the sheet to estimate optical depth, giving real self-shadow.
3. Use dual-lobe HG (for example g = 0.8 and g = -0.2, starting values), Beer-powder, and an ambient term from the sky SH. Colour the sun by the transmittance LUT, so clouds go orange at dusk from the same model as the sky.
4. Apply aerial perspective to the sheet, instead of today's separate fade, so horizon clouds dissolve into the same haze.
5. Use 2 layers (cumulus low, cirrus high) with parallax, which is one more quad and a much stronger sense of depth.
6. Project the weather map down onto terrain as moving cloud shadows and feed mean coverage into ambient. This is cheap and probably the largest world-connecting gain.

**Tier 1, cached panorama (estimate under 1 ms amortised).** Render clouds slowly into a low-res octahedral or lat-long HDR target, updating one strip per frame, and sample it from the sky. It only suits camera-far clouds, has no parallax on fast flight, and can double as cloud input to the sky cube.

**Tier 2, half-res raymarch with reprojection (high tier).**
- Budget 32 to 64 base steps and 3 to 6 light steps. Use 128^3 and 32^3 noise, a weather map and blue-noise ray jitter.
- Render at half res updating 1 of 16 in each 4x4 block. Reproject using a cloud-depth estimate, since the clouds do not write scene depth.
- Upsample with depth-aware composite against the scene, or just test against far-plane sky.
- Fragment shaders with `texelFetch` and two history targets are sufficient, so no compute is needed.

**Fallback logic.** Tie this to `quality.go` tiers. Low uses Tier 0 with 1 layer, mid uses 2 layers plus shadows, high uses Tier 2 with Tier 0 kept as the distant ring beyond raymarch range. Disable reprojection during camera cuts and show the lower-res buffer until history refills, following the Nubis fallback.

### Gaps
- No published millisecond figures were found for 2D cloud layers or for reprojected raymarch on a low-end desktop or integrated GPU. All Tier estimates above are unmeasured.
- Bauer's RDR2 talk and Nubis 2017/2022/2023 were not read in detail, and Frostbite 2016 course notes could not be fetched (over the size limit) — [Frostbite notes](https://media.contentapi.ea.com/content/dam/eacom/frostbite/files/s2016_pbs_frostbite_sky_clouds.pdf).
- Prior art specifically for GL 3.3 temporal cloud upscaling was not found. Feasibility is reasoned from its feature set.

## How do you derive ambient light from the sky?

### Takeaway
Use 9-coefficient L2 spherical harmonics for diffuse sky light, projected from the sky LUT and refreshed a few times per second, with an analytic lower-hemisphere term for ground bounce. Keep a separate HDR cube updated one face per frame for reflections. The engine's probe path already works one face at a time, so this is mostly an extension of existing code.

### Cited Findings
**Current moonBASIC state (local code)**
- `envcube.go`, `ensureSkyCube` rebuilds all 6 faces at 6 mips on the CPU through `prefilterEnv`, stored as `gl.RGBA8`, with a size of 32 to 256 by quality tier. It is keyed on `skyRev`, deleting and recreating the texture on each change.
- In the grep performed, `skyRev` is only bumped at `skybox.go:196`. Whether procedural atmosphere changes invalidate the cube was not traced, and should be checked.
- The probe path in the same file already renders one cube face per update (`face := pr.face % 6`).
- `linear_light.go` has `skyAmbientGLSL`, which lights surfaces from the sky picture, and its comment notes the sky map contains no ground.

**Representation**
- Filament accepts 1, 2 or 3 SH bands. 3 bands (9 coefficients) is the recommended form — [Filament IBL mirror](https://www.mintlify.com/google/filament/rendering/ibl) (third-party mirror of Filament docs).
- Unity stores its ambient lighting as SH L2 — [Unity ambientProbe docs](https://docs.unity3d.com/es//ScriptReference/RenderSettings-ambientProbe.html).
- Ramamoorthi and Hanrahan is the standard reference for irradiance from SH, and axis conventions differ between tools (Enlighten swaps y and z) — [Enlighten docs](https://documentation.enlighten.siliconstudio.co.jp/wiki/spaces/SDK310/pages/904999329).
- Bruneton ships a 64x16 ground irradiance table as a separate compact alternative — [Bruneton constants.h](https://raw.githubusercontent.com/ebruneton/precomputed_atmospheric_scattering/master/atmosphere/constants.h).
- The indirect term reduces to AO times sky irradiance for the surface normal — [Frisvad skylight slides](https://people.compute.dtu.dk/jerf/teaching/slides/02564_skylight_slides.pdf).
- On-the-fly SH projection for skylit outdoor scenes is called time consuming, motivating coefficients stored as functions of sky parameters — [Habel et al. 2008](https://www.cg.tuwien.ac.at/research/publications/2008/Habel_08_SSH/Habel_08_SSH-Preprint.pdf).

**Update frequency**
- Unreal's Real Time Capture uses Time Slicing to spread the capture over several frames, and needs a Sky Atmosphere component — [Unreal sky lights docs](https://dev.epicgames.com/documentation/en-us/unreal-engine/sky-lights-in-unreal-engine).
- Godot offers Quality (slow, static skies), Incremental (same sampling spread over frames, for slowly updating skies) and Realtime (fast filter, lower quality, needs 256x256). Sizes of 512 and above are for high-end hardware only — [Godot Sky docs](https://docs.godotengine.org/en/stable/classes/class_sky.html).

**Ground**
- Unreal exposes a Lower Hemisphere Color and a flag that sets lower-hemisphere distant lighting to it — [SkyLightComponent API](https://docs.unrealengine.com/5.1/en-US/PythonAPI/class/SkyLightComponent.html).
- Unreal's atmosphere Ground Albedo matters when multiple scattering is enabled — [Unreal Sky Atmosphere properties](https://dev.epicgames.com/documentation/zh-cn/unreal-engine/sky-atmosphere-component-properties-in-unreal-engine).
- Hillaire offers a secondary trace toward the ground for cloud lighting — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).

### Inferences
**Proposed pipeline (no compute)**
1. Sample the sky-view LUT, not the cube, on the CPU or GPU. On the GPU, render a 9x1 RGBA16F target with a fragment loop of about 64 to 256 stable directions, one coefficient per texel. This takes well under a millisecond (estimate).
2. Read back through a PBO a frame or two later with a fence, to avoid a pipeline stall, or simply store the 9x1 target as a texture and `texelFetch` it in lit shaders so there is no readback at all.
3. Refresh every 0.25 to 1 s and lerp old to new. A 20-minute day moves the sun at 0.3 degrees per second, so faster updates would be invisible. Force an update on a weather change.

**Ground bounce in SH.** For a constant lower-hemisphere radiance `Lg`, projecting gives (my derivation, verify the sign against the engine's Y-up convention):
- `c00 += Lg * sqrt(pi)`
- `c1,y += -Lg * sqrt(3*pi)/2`
- All band-2 terms are zero.

Set `Lg = albedo * (E_sky_up + E_sun * max(sun.y, 0)) / pi`, with albedo as a scene or terrain-average colour and `E_sky_up` evaluated from the sky's SH with the normal straight up. This is one extra uniform calculation and it stops downward-facing surfaces going black or sky-blue. Ground albedo should also feed the multi-scattering LUT, so the sky and bounce use one value.

**Cheap extras**
- Multiply diffuse by AO (SSAO is already part of many pipelines) and by a cloud factor from mean weather-map coverage, so an overcast sky dims and desaturates ambient.
- A cubemap from an analytic sky contains no terrain, so reflections below the horizon should use the same `Lg`.

**Reflections**
- Move the sky cube to RGBA16F or R11F_G11F_B10F. RGBA8 clips any sky radiance over 1 and bands low-light values, so sunsets lose energy.
- Render one face per frame from the sky-view LUT, sample clouds if available, and after 6 frames blur the mips. `glGenerateMipmap` is a coarse roughness approximation that needs no compute; better still is a small fragment prefilter on one mip per frame, following Godot's Incremental idea.
- Crossfade two cubes to hide face-order seams. Reuse the existing probe face loop to do this.
- Remove the CPU `prefilterEnv` path for procedural skies, keeping it only for static image skies, where it is a one-time cost.

**Choice between the two**
- SH is best for diffuse (9 vec3 uniforms, very cheap per pixel).
- Cube mips are best for specular.
- Keep only SH on the low tier, plus one cube updated rarely.

### Gaps
- No source describes the exact pipeline of Hillaire sky LUT to SH with ground bounce. The steps above are reasoned, and Unreal's internal sky-light code was not read.
- The Frostbite 2016 notes likely cover sky cube updates but could not be read.
- Precise SH projection timing on low-end GPUs was not found.

## What are good, cheap sun disc, glow and light shaft methods?

### Takeaway
Keep the disc out of the LUT, draw it analytically with transmittance and limb darkening, and get glow from the Mie lobe plus HDR bloom, which adds almost nothing. For shafts, a half or quarter-res screen-space radial blur masked by sky visibility is the cheapest convincing option, while a fade handles its known failure cases. Shadow-map marching is the higher-quality alternative for strong GPUs.

### Cited Findings
**Sun disc and glow**
- The sun disc is composited after the sky-view LUT is applied — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf). The same LUT can hold the contribution of any number of suns.
- Even the Mie lobe is a soft blob, so it survives low-res LUT storage — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).
- A simple disc uses `smoothstep` with adjustable feather width and intensity, driven by a directional light — [Godot Rayleigh/Mie/Ozone shader](https://godotshaders.com/shader/rayleigh-mie-ozone-based-atmospheric-scatter/).
- A double-lobe HG (forward lobe plus weaker back lobe) gives more natural scattering — [Freeman's Sky Shader](https://godotshaders.com/shader/freemans-sky-shader/).
- HG is a common approximation of the full Mie phase function — [Keller 2008](https://intern.fh-wedel.de/~bo/db/pubs/keller_simAtmoScatteringGPU2008.pdf).
- Higher eccentricity tightens the haze around the sun — [flutter_scene PhysicalSkySource](https://pub.dev/documentation/flutter_scene/0.19.0/scene/PhysicalSkySource-class.html).

**Screen-space shafts — from [GPU Gems 3 ch. 13](https://developer.nvidia.com/gpugems/gpugems3/part-ii-light-and-shadows/chapter-13-volumetric-light-scattering-post-process)**
- Occluders render black and untextured into a source buffer, the image is sampled along a ray toward the light's screen position, then additively blended. The knobs are NUM_SAMPLES, Density, Weight, Decay and Exposure.
- Downsampling reduces bandwidth, and bilinear filtering is sufficient. No preprocessing or scene setup is needed, and it works with animated scenes of any complexity. It pairs well with a depth pre-pass.
- Limitations: background shafts can appear in front of foreground objects with near lights; flicker occurs as occluders cross the screen edge (an extended guard band reduces this); the light's screen position goes to infinity as it nears perpendicular to the view (clamp or fade); and surface texture can cause streaks.
- The chapter gives no benchmark figures.

**Other implementation experience**
- No shafts appear when the light is behind the camera, and occluders off screen do not block light — [Godot screen-space god rays](https://godotshaders.com/shader/screen-space-god-rays-godot-4-3/). An OpenGL implementer confirms this, and calls it far less taxing than 3D light volumes — [Khronos forum](https://community.khronos.org/t/sunbeams-in-crysis/54219).
- An OpenGL walkthrough is at [Fabien Sanglard](https://fabiensanglard.net/lightScattering).

**Volumetric alternatives**
- Hillaire offers per-pixel shadow tracing with jitter and TAA, or stores shadows in the sky-view and aerial LUTs, tuned by resolution and sample count — [Hillaire 2020 slides](https://blog.selfshadow.com/publications/s2020-shading-course/hillaire/s2020_pbs_hillaire_slides.pdf).
- Bruneton's reference code partly implements light shafts (Eqs. 17 and 18, without the shadow volume algorithm) — [Bruneton docs](https://ebruneton.github.io/precomputed_atmospheric_scattering/).

### Inferences
**Sun disc (a few shader instructions)**
- The real sun's angular radius is about 0.27 degrees (from memory); games commonly inflate it. Use `cos(angle)` with `smoothstep` feathered over about one pixel.
- Limb darken by `mix(0.4, 1.0, sqrt(1 - r^2))` (starting value, tune by eye).
- Multiply by the transmittance LUT toward the sun, so it reddens at sunset and matches shadow and direct-light colour. Use the same transmittance to colour the directional light, so sun, sky and world agree.
- Output HDR radiance well above 1, so existing bloom produces the glow. If tone mapping is LDR-only, add an explicit lobe.
- Occlude the disc by cloud density.

**Glow**
- Evaluate HG with g about 0.76 to 0.8 for sky glow, and tighten for a thin aureole lobe at high turbidity. Make the lobe per-pixel in the composite pass if the 192x108 LUT blurs it, since the sharp part near the sun is the part that suffers.
- The current fixed `fogTint` lobes can be replaced by the same phase function, so haze near the sun matches the sky.

**Shafts recipe (GL 3.3 fragment only; estimate 0.3 to 1 ms)**
1. Downsample depth to quarter res and write sun-visible mask = (depth at far plane) x sun disc-plus-glow x (1 - cloud density).
2. Run a radial blur of 16 to 32 samples per pixel with per-pixel dither, then a small blur. Alternatively run 3 passes of 6 samples each, for 216 effective samples, which costs less.
3. Additively composite in linear HDR, tinted by sun colour times transmittance and scaled by haze density and by `pow(max(dot(view, sun), 0), n)`.
4. Fade to zero when the sun is behind the camera or far off screen.

Using the depth-derived mask avoids the extra occluder scene pass and keeps texture streaks out, provided a readable scene depth is available in the post pass (not checked here). The cost is that only sky-visible masks produce shafts, which matches outdoor use.

**Higher tier.** March 8 to 16 dithered shadow-map samples per half-res pixel, composite through bilateral upsampling, and reuse whatever sun shadow map `shadows.go` already produces (its layout was not reviewed). This fixes the sun-behind-camera failure. Choose it only for high and ultra tiers, since it adds per-pixel shadow lookups.

**Clouds and shafts.** The weather map's top-down projection gives cheap cloud-shaft variation by modulating the sun term in the aerial function, without a full Beer Shadow Map.

### Gaps
- No measured GPU cost was found for screen-space shafts or disc techniques. The ms ranges are estimates.
- Limb darkening constants were not sourced, and the coefficient above is a starting value.
- How G3N's post pipeline handles HDR intermediate targets was not reviewed here, and it determines whether bloom can carry the glow. The local `postfx.go` should be checked before relying on it.
