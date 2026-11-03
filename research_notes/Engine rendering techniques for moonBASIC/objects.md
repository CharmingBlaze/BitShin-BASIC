# Making objects, trees and materials look better in moonBASIC on OpenGL 3.3 core

Reading guide for the report writer:
- "Cited Findings" come from sources. Most pages were read through a summarising fetch tool, so check any formula against the source before copying it. The three.js tone mapping excerpts and the Narkowicz coefficients are the closest to verbatim.
- "Inferences" are my engineering judgement or my own arithmetic. They have not been tested in moonBASIC.
- I did not inspect moonBASIC's source. Statements about its current state come from the brief.
- Source quality varies. Forum posts and blogs (bugnet.io, Polycount, Unreal forums) are practitioner claims, not authority. They are labelled where they carry weight.

## Which techniques give the most visual gain for the cost on OpenGL 3.3?

### Takeaway
The biggest wins are foliage lighting and color, not new rendering features. Fix the shaded-side ambient, normals and AO first, then add per-instance color and wind, then move to better tree meshes and tone mapping. The ranking below is my judgement. No source I found measures visual gain.

### Cited Findings
- Crysis shaded "thousands of different vegetations" with Lambert, a translucency term, hemisphere ambient and baked AO, which shows how simple a production foliage model can be. The chapter gives no frame times — [GPU Gems 3 ch. 16](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-16-vegetation-procedural-animation-and-shading-crysis)
- EVSM4 at 2048x2048 with 4xMSAA, mips, 8xAF and a 9-texel filter cost about 11.5 ms more than a 7x7 PCF kernel, and about 3 ms more at 1024x1024, on the author's PC — [Pettineo, shadow maps](https://therealmjp.github.io/posts/shadow-maps/)

### Inferences
Ranked by gain for cost. The cost column is a rough estimate, not a measurement.

| Rank | Technique | Fixes | Cost | GL 3.3 |
|---|---|---|---|---|
| 1 | Directional ambient: sky color above, darker ground bounce below | Near-black shade, flat shape | About 1 mix per pixel | Yes |
| 2 | Limit AO on foliage, never apply it to direct sun or translucency | Near-black canopy | Near zero | Yes |
| 3 | Canopy normals blended toward a sphere | Faceted, harsh canopies | Mesh or vertex-shader work | Yes |
| 4 | Per-instance hue and value variation, height gradient | Copy-paste forests, flat color | Near zero | Yes, instanced attributes |
| 5 | Translucency or wrap on the shaded side | Dead backlit trees | A few ALU instructions | Yes |
| 6 | Hill ACES or an alternative tone mapper, plus a small grade | Hue skew, crushed toe | Near zero, or one 3D LUT | Yes |
| 7 | Vertex-shader wind | Static scene | Vertex shader only | Yes |
| 8 | Parametric tree generator with a shared skeleton and LODs | Crude cone-and-sphere trees | CPU at load | Yes |
| 9 | Stabilized cascades, tighter fit, shadow proxies | Shimmer, shadow cost | Low | Yes |
| 10 | Geometric specular AA | Sparkle on glossy, curved objects | About 4 lines | Yes, uses dFdx/dFdy |
| 11 | Screen-space contact shadows | Lost fine contact at distance | One post pass | Fragment-shader port needed |
| 12 | Depth-range partitioning, cached far cascades | Long shadow range | Moderate engineering | Yes, no compute |
| 13 | Octahedral impostors | Very far trees | Baker plus shader | Yes |
| 14 | Multi-scattering compensation | Dark rough metals | DFG LUT | Yes, little effect on foliage |

### Gaps
- I found no published visual-gain-for-cost comparison across these techniques.

## How do you stop foliage going near-black in shade without flattening contrast?

### Takeaway
Treat shade as lit by a colored, directional sky rather than as "less sun". Keep a 2:1 up-to-down ambient ratio so shape survives, cap AO on canopies, add transmission, and check the tone mapper's toe, which can halve already-dark values.

### Cited Findings
- Crysis ambient is hemispheric: `lerp(ambient * 0.5, ambient, saturate(N.z * 0.5 + 0.5))`, plus baked AO stored in vertex alpha — [GPU Gems 3 ch. 16](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-16-vegetation-procedural-animation-and-shading-crysis)
- Crysis back-side translucency: `fEdotL = saturate(dot(E, -L))` raised to the 4th power, and back diffuse wrapped as `saturate(dot(-N, L) * 0.6 + 0.4)`. A tunable view-dependency value blends the two, and the result multiplies a back diffuse color. A low-resolution subsurface map (e.g. 128x128) encodes thickness and veins — [GPU Gems 3 ch. 16](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-16-vegetation-procedural-animation-and-shading-crysis)
- Final Crysis composition: `(ambient + diffuse + subsurfaceBack) * diffuseMap * color`, then specular is added, then everything is multiplied by baked AO — [GPU Gems 3 ch. 16](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-16-vegetation-procedural-animation-and-shading-crysis)
- Half-Lambert was designed to stop the rear of an object losing shape and looking flat. Valve calls it completely non-physical — [Valve wiki](https://developer.valvesoftware.com/wiki/Half_Lambert). The form is scale by 1/2, add 1/2, then square — [HN quote of the wiki](https://hn.svelte.dev/item/46458194)
- Half-Life 2 paired Half-Lambert with an ambient cube: six directional samples weighted by squared normal components — [Mitchell, SIGGRAPH 2006](https://advances.realtimerendering.com/s2006/Mitchell-ShadingInValvesSourceEngine.pdf)
- Filament's subsurface wrap for cloth uses `w = 0.5`: `saturate((NoL + w) / (1 + w)^2) * saturate(subsurfaceColor + NoL)`. Filament says this is not physically based — [Filament docs](https://google.github.io/filament/Filament.md.html)
- Filament lists colored penumbrae for its subsurface model, tinted from `subsurfaceColor`, and uses `coloredPenumbra: true` in a grass example. It lists a vegetation base color of sRGB 0.48, 0.51, 0.31 — [Filament Materials](https://google.github.io/filament/Materials.md.html)
- GTAO multi-bounce recovers lost interreflection per channel: `a = 2.0404*albedo - 0.3324`, `b = -4.7951*albedo + 0.6417`, `c = 2.7552*albedo + 0.6903`, result `max(v, ((v*a + b)*v + c)*v)` — [Unity commit](https://plastichub.unity.cn/unity-tech-cn/ScriptableRenderPipeline/commit/f14653df-ebd7-45df-b65d-394ba3319f87); [Jimenez et al.](https://www.iryoku.com/downloads/Practical-Realtime-Strategies-for-Accurate-Indirect-Occlusion.pdf)
- Blender EEVEE reduces albedo to one luminance value first, with a comment that colored multi-bounce "looks bad". This disagrees with using the per-channel form — [Blender commit](https://developer.blender.org/rB779c950098e46a34f89f1705be61b4b24be11568)
- Practitioner reports on the same problem:
  - Stacked foliage cards self-shadow until the interior goes black. Suggested fixes are excluding foliage from heavy AO or bake, or adding a small fill term — [bugnet.io](https://bugnet.io/blog/how-to-fix-foliage-self-shadowing-black-or-flat-when-baked)
  - Letting leaves pass light can address both hard shadow edges and overly dark clump interiors — [Planetside forum](https://planetside.co.uk/forums/index.php/topic,24556.30.html)
  - Too much translucency blurs canopies together — [Planetside forum](https://planetside.co.uk/forums/index.php/topic,31414.msg301576.html)
- Narkowicz's ACES fit is `(2.51x² + 0.03x) / (2.43x² + 0.59x + 0.14)` — [Narkowicz](https://knarkowicz.wordpress.com/2016/01/06/aces-filmic-tone-mapping-curve/)

### Inferences
- **Likely causes, to check in moonBASIC in this order:**
  1. AO multiplying sun or translucency, not only ambient. Crysis does multiply everything by its baked AO, but that AO is authored and gentle. Screen-space GTAO on a dense canopy is much stronger.
  2. A low or neutral ambient term.
  3. Shadow-map self-shadowing inside the canopy, which removes all sun.
  4. The tone mapper's toe.
- **The toe matters (my arithmetic with the Narkowicz fit).** Input 0.01 becomes 0.0038, 0.02 becomes 0.0105, and 0.05 becomes 0.044. In display terms, 0.01 would encode at about 25/255 unmapped but lands at about 12/255. Values above about 0.1 are not reduced. So darkness that looks acceptable before tone mapping can look black after it, and lifting exposure alone also moves the sunlit range.
- **Worked example with assumed inputs.** Take Filament's vegetation green at about 0.22 linear, ambient irradiance factor 0.3 and AO 0.4. That gives 0.027 linear, which tone-maps to about 0.017, or about 35/255. Red and blue are lower, so this reads as near-black.
- **Multi-bounce AO alone will not fix this (my arithmetic).** At v = 0.3, albedo 0.2 gives only about 0.35, while albedo 0.6 gives about 0.53. Dark leaf colors barely benefit. Use a floor as well.
- **Recommended foliage lighting, as a starting point:**
```glsl
// N is the blended canopy normal. canopyAO is baked, screenAO is GTAO.
vec3  amb  = mix(groundBounce, skyColor, N.y * 0.5 + 0.5);       // keep the ratio near 2:1
float ao   = max(canopyAO * mix(1.0, screenAO, 0.35), aoFloor);  // start aoFloor near 0.35
float wrap = saturate((NoL + w) / ((1.0 + w) * (1.0 + w)));      // start w near 0.3
float back = pow(saturate(dot(V, -L)), 4.0) * saturate(dot(-N, L) * 0.6 + 0.4);
vec3  col  = albedo * (amb * ao + sun * shadow * wrap)
           + albedo * transColor * sun * softShadow * back;
```
- **Keeping contrast:**
  - Keep the value contrast between top and underside, and add hue contrast too: blue-tinted sky fill and warm-green bounce.
  - Do not raise flat ambient, which is what causes flattening.
  - Use a height gradient as the main shape cue.
  - Add a local AO term from distance to the canopy center, so inner vertices are darker than outer ones but never black.
- **Half-Lambert versus energy-normalized wrap.** Half-Lambert brightens the whole model. Filament's normalized wrap keeps total energy nearer Lambert. Use normalized wrap for Advanced and Half-Lambert or a ramp for Stylized.
- **Translucent shadow.** Sample translucency with a softened or partial shadow, since a leaf's self-shadow should not cancel transmission.

### Gaps
- Inigo Quilez's outdoor lighting article returned 404 at the URL I tried, so I could not cite his sun, sky and bounce model. That model is the idea behind the recommended term.
- I found no authoritative source on foliage AO floors. The 0.35 starting value is a guess to tune.
- I found no source on how to apply GTAO to foliage specifically.

## How do you build good real-time trees?

### Takeaway
Use a low-poly canopy mesh or crossed cards with sphere-blended normals, two-sided shading, instance-phase wind and a mesh LOD chain. Leave octahedral impostors until the far range matters. Because a solid low-poly canopy needs no alpha, MSAA and alpha to coverage are only needed for card-based trees.

### Cited Findings
**Normals**
- Tree shape normals should point outward from a central location, following a sphere or cone, while grass follows the terrain — [NedMakesGames](https://nedmakesgames.medium.com/creating-a-foliage-shader-in-unity-urp-shader-graph-5854bf8dc4c2)
- ez-tree's `roundedNormals` sets each leaf vertex normal to the leaf direction plus the origin-to-vertex vector, normalized. It then skips the usual backface normal flip — [ez-tree source](https://raw.githubusercontent.com/dgreenheck/ez-tree/main/src/lib/tree.js)
- Polycount users confirm that flipped backface normals can break spherical canopy shading — [Polycount](https://polycount.com/discussion/comment/2757319)
- Transferring normals from a sphere or hull is called essential to 3D foliage by one tutorial, and a hull proxy with a backface fix is what a Blender add-on automates — [Anime foliage pipeline](https://trungduyng.substack.com/p/tutorial-blender-anime-foliage-pipeline); [Foliage Normals](https://superhivemarket.com/products/foliage-normals)
- The technique may not suit realistic styles but works very well on stylized assets, and card meshes can double vertex count, so a card-free LOD can help — [Polycount](https://polycount.com/discussion/comment/2720547)

**Alpha**
- Alpha to coverage derives fragment coverage from alpha and only matters with MSAA enabled. Filament enables it automatically for masked blending, with `maskThreshold` default 0.4 — [Filament Materials](https://google.github.io/filament/Materials.md.html)
- Crysis used double-sided alpha test for leaves and alpha blending for grass. It rejected two-pass test and blend on quality for cost, and used a depth-based post-process to smooth edges because hardware alpha to coverage was unavailable. Grass was shaded per vertex to reduce fill-rate cost — [GPU Gems 3 ch. 16](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-16-vegetation-procedural-animation-and-shading-crysis)

**Wind**
- Main bending displaces xy by normalized height, then renormalizes to the original length to limit deformation — [GPU Gems 3 ch. 16](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-16-vegetation-procedural-animation-and-shading-crysis):
```hlsl
float fBF = vPos.z * fBendScale;
float3 vNewPos = vPos;
vNewPos.xy += vWind.xy * fBF;
vPos.xyz = normalize(vNewPos.xyz) * fLength;
```
- Detail bending sums four smoothed triangle waves at frequencies 1.975, 0.793, 0.375 and 0.193. Vertex color R is edge stiffness, G is per-leaf phase, B is leaf stiffness and A is AO. The cheap wave is `abs(frac(x + 0.5) * 2 - 1)` smoothed by `x*x*(3 - 2*x)` — [GPU Gems 3 ch. 16](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-16-vegetation-procedural-animation-and-shading-crysis)
- GPU Gems 3 chapter 6 also computes wind in the vertex shader in a way that suits instancing — [GPU Gems 3 ch. 6](https://developer.nvidia.com/gpugems/gpugems3/part-i-geometry/chapter-6-gpu-generated-procedural-wind-animations-trees)

**Impostors**
- Brucks's octahedral impostor baker was a UE4 plugin used for Fortnite Battle Royale — [80.lv](https://80.lv/articles/impostor-baker-for-ue4/)
- The Godot port recommends a grid size of 16 and says hemisphere mapping suits standard foliage because it gives more resolution from the side. It bakes base, normal-plus-depth and ORM maps. The Standard shader uses 36 vertices and supports shadows and depth, while the Light shader uses 12 and has no shadows — [Godot Octahedral Impostors](https://github.com/SIsilicon/Godot-Octahedral-Impostors)
- Atlas size trades transition smoothness against performance — [Unity docs](https://docs.unity3d.com/Packages/com.unity.industry.toolkit%403.1/manual/Actions/Autogenerated/BakeImpostor.html)

### Inferences
- **Two-sided rule.** When using sphere normals, do not flip the normal on back faces. When using true leaf normals, do flip it.
- **Tree tiers for moonBASIC:**
  - Near: skeleton mesh plus cards or faceted clumps.
  - Mid: same skeleton with fewer rings and larger, fewer clumps.
  - Far: a very low-poly mesh with the same shader, which may keep draw calls and memory simpler than an atlas.
  - Very far: impostors, only if tree counts demand them. These can be baked at load into an FBO atlas, which fits "scripts create everything".
- **MSAA dependency.** Alpha to coverage needs multisampled HDR targets. Resolving before tone mapping can leave bright edge aliasing. Confirm whether moonBASIC's pipeline has MSAA before promising this. If it does not, use alpha test with post AA.
- **AO and cards.** GTAO sees card depth as solid geometry, which is another reason to reduce its weight on foliage.

### Gaps
- The shaderbits octahedral impostors article returned only navigation text, so the atlas math and three-frame blending are not sourced here.
- Ben Golus's alpha-to-coverage article returned 403, so the coverage-preserving mip and `fwidth` sharpening details are not sourced.
- SpeedTree publications were not retrieved.
- No measured cost was found for any of the leaf shading terms.

## How do stylized games get their look, and is a Classic, Stylized and Advanced profile design sound?

### Takeaway
A profile design is sound if all profiles share one material description and differ only in shading functions and default lighting and grade. Style should also be selectable per material class, because stylized renderers reportedly mix shading models per pixel. Published detail on these specific games is thin.

### Cited Findings
- A secondary write-up of a video analysis says Breath of the Wild defers shading by per-pixel material type across about nine categories (terrain, foliage, water, skin, hair, eyes, clothes or fur, metal, rubber). This lets it cel-shade characters while leaving the world more realistic. Terrain is not cel-shaded, and a decal overwriting Link's material ID caused a known glitch — [Set Side B](https://setsideb.com/?p=3303); [Obsidian notes](https://publish.obsidian.md/xybre/3.+Reference/Creative/Development/Gamedev/Visuals/Shaders+-+Deferred+Shaders). This is not a Nintendo source.
- Ramp shading steps light through a color ramp, and fewer transitions give harder bands — [Godot shader](https://godotshaders.com/shader/ucbcs-stylized-light-using-a-mask/)
- Polycount practitioners aim for a Fortnite-like look with sphere-transferred normals, but do not document Fortnite's actual method — [Polycount](https://polycount.com/discussion/comment/2757361)
- Subsurface color advice from an Unreal forum: use a desaturated or more saturated version of the base color — [Unreal forum](https://forums.unrealengine.com/t/4-7-foliage-material-seems-off/316018)
- Godot exposes three renderers, and features not listed are available in all of them. Compatibility (OpenGL) lacks PCSS for directional lights, SSIL, SDFGI, SSR, TAA, subsurface scattering and compute, and uses RGBA8 color. Forward+ uses RGBA16F. Switching "may require some manual tweaks to your scene, lighting, and environment", and a fallback to Compatibility may look different from what was intended — [Godot renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)
- Filament selects a shading model per material (for example `shadingModel: subsurface` and cloth) — [Filament Materials](https://google.github.io/filament/Materials.md.html)

### Inferences
- **Verdict on profiles: sound, with conditions.** The risks are three times the tuning work and looks drifting apart. Godot's note about manual tweaks shows that this is a real cost even when the material interface is shared. Note that Godot's tiers are hardware tiers, not style tiers, so it only partly supports this design.
- **Conditions:**
  1. Materials describe intent: base color, roughness, metallic, a foliage flag, translucency, emissive. No profile-specific parameters are required in scripts.
  2. A profile selects a light function, ambient model, shadow filter defaults, tone mapper and grade, and nothing else.
  3. Every profile uses linear lighting, so colors keep meaning across profiles.
  4. Profile changes never change gameplay-visible geometry or object placement.
  5. Fallbacks are defined, for example a Stylized foliage flag that maps to a Classic wrap.
  6. A small set of reference scenes is captured in each profile for regression checks, including a shaded tree test.
- **Suggested content:**

| | Classic | Stylized | Advanced |
|---|---|---|---|
| Diffuse | Lambert or Half-Lambert | Ramp or wrap, per material class | Energy-normalized wrap for foliage, PBR otherwise |
| Ambient | Hemisphere | Tinted sky and bounce, height gradient | Hemisphere or cube with GTAO |
| AO | Baked vertex only | Baked with low-weight GTAO | GTAO, multi-bounce, foliage floor |
| Shadows | PCF | PCF with a soft, tinted penumbra | PCSS near, PCF far |
| Tone map | Neutral or none | Neutral or AgX with a grade | Hill ACES or AgX with a look |

- **Polish for solid low-poly trees:**
  - Keep the faceted silhouette, but mix face and sphere normals (start near 0.6 toward the sphere) so facets read softly and the canopy lights as one volume.
  - Add a base-to-top color gradient, darkened inner vertices and per-clump value shifts.
  - Add a small shading rim from `pow(1 - NoV, n)` only if needed.
  - Group shapes into clumps rather than one cone or sphere, since silhouette variety is what removes the "geometric" look.

### Gaps
- I found no primary source (GDC talk, developer article) on Fortnite foliage shading, Nintendo's BotW foliage or tone mapping, Ghibli-style production pipelines, or specific modern low-poly indie games. Treat these looks as inferred from community practice.
- No source evaluates a style-tiered profile system directly.

## How do you add per-instance color variation, vertex gradients and wind to instanced low-poly trees?

### Takeaway
Pack everything into vertex colors and one per-instance attribute. This costs almost nothing and removes the most obvious signs of instancing.

### Cited Findings
- A Godot leaf shader hashes the instance ID into a random value, uses it to sample a color gradient, and offsets sine sway per instance so trees do not move in unison — [Godot Shaders](https://godotshaders.com/shader/leaf-shader-flower/)
- An instanced tree sample gives each of 400 instances a position, scale, rotation and green tint, and substitutes the tint when the mesh vertex color is zero. The page text was fragmented, so treat this as approximate — [Pngine sample](https://hugodaniel.com/pages/pngine/docs/samples/instanced-trees/)
- A low-poly wind shader uses red and blue vertex color influence, and is built for subtle sway rather than strong gusts — [polygon-wind](https://www.github.com/RenanBomtempo/polygon-wind)
- A Unity low-poly wind shader offers a height threshold and trunk color recognition for single-mesh trees — [Unity forum](https://discussions.unity.com/t/low-poly-wind-shader/673989)
- Crysis assigns stiffness per instance and decays wind strength over time when no source affects it — [GPU Gems 3 ch. 16](https://developer.nvidia.com/gpugems/gpugems3/part-iii-rendering/chapter-16-vegetation-procedural-animation-and-shading-crysis)

### Inferences
- **Suggested vertex layout:** R is wind weight (0 on the trunk, rising to the tips), G is per-clump phase, B is canopy height gradient, A is baked AO. Generated trees can compute all four, so no painting is needed.
- **Per-instance:** one vec4 attribute with hue shift, value shift, wind phase and scale, using `glVertexAttribDivisor` (core in 3.3, from memory). Derive the seed from world position if scripts supply nothing.
- **Color:** apply hue and value shifts in linear space with small ranges (about ±5 percent value, with a slight green-to-yellow hue range). Use patch-scale noise in world XZ as well as a per-instance random value, so color drifts across a hillside rather than looking like static.
- **Flat-shaded wind:** if normals come from `cross(dFdx(p), dFdy(p))`, wind updates them automatically. Stored normals barely need updating at small amplitudes.

### Gaps
- No measured results for variation ranges. The numbers above are starting points.

## How can runtime procedural generation make tree meshes with no art assets?

### Takeaway
Use a level-based recursive parametric generator, split into skeleton growth and meshing, as in ez-tree. It suits a scripting API, and one skeleton can produce every LOD. Add space colonization later for species that need crown-shaped growth.

### Cited Findings
- ez-tree separates skeleton growth, which makes all random decisions, from meshing, which uses no RNG. `generate()`, `generateLODs()` and `createGeometry()` can therefore mesh one skeleton at different detail without changing shape — [ez-tree source](https://raw.githubusercontent.com/dgreenheck/ez-tree/main/src/lib/tree.js)
- Growth uses a breadth-first branch queue. Children sit at stratified, jittered heights from `start[level]` to 1.0, with radial slots shuffled. Evergreens scale child length by `1 - childStart`, so lower branches are longer, and their radius tapers fully along the branch — [ez-tree source](https://raw.githubusercontent.com/dgreenheck/ez-tree/main/src/lib/tree.js)
- Each section's orientation is perturbed by gnarliness scaled by `max(1, 1/sqrt(radius))`, so twigs wander more than trunks. Twist rotates about local Y, and a force rotates toward a direction with the angle clamped to `strength / radius` — [ez-tree source](https://raw.githubusercontent.com/dgreenheck/ez-tree/main/src/lib/tree.js)
- Meshing uses `max(3, round(segments * factor))` per ring. Parent and child counts match so joints stay sealed, there are no end caps, and the tip radius is 0.001 — [ez-tree source](https://raw.githubusercontent.com/dgreenheck/ez-tree/main/src/lib/tree.js)
- LODs are defined by `sectionStride`, `segmentFactor`, `leafStride`, `leafScale` (which enlarges kept leaves to preserve coverage) and a billboard override. Leaves can be single or double crossed quads — [ez-tree source](https://raw.githubusercontent.com/dgreenheck/ez-tree/main/src/lib/tree.js)
- ez-tree has over 50 parameters, is free and open source, and exports GLB and PNG — [npm](https://npmjs.com/package/@dgreenheck/ez-tree); [Digital Production](https://digitalproduction.com/2026/01/13/free-browser-based-procedural-tree-generator-released/)
- ez-tree writes no vertex color attribute, and uses 16-bit indices, which cap a buffer at 65,535 vertices — [ez-tree source](https://raw.githubusercontent.com/dgreenheck/ez-tree/main/src/lib/tree.js)
- Weber and Penn's 1995 algorithm is the basis of OpenAlea.WeberPenn — [OpenAlea](https://openaleadocs.readthedocs.io/en/latest/tutorials/visualea/weberpenn.html)
- Runions et al. extend open leaf venation to 3D, producing realistic-looking trees with parameters that match characteristics used in landscaping, which gives control over shape — [Runions et al. 2007](https://algorithmicbotany.org/papers/colonization.egwnp2007.html)

### Inferences
- **Approach:** port the ez-tree structure to Go, with per-level arrays for length, radius, count, angle, gnarliness and taper. Its level-wise parameters resemble Weber-Penn in spirit, but I found no evidence that it implements that algorithm.
- **Additions moonBASIC needs beyond ez-tree:** the vertex-color channels from the previous section, canopy normals computed from a skeleton-derived sphere or hull, and 32-bit indices if near meshes get large.
- **Needle trees:** use evergreen rules (a straight trunk, long lower branches) with needle-clump cards, or stacked faceted cones with jittered rims for low poly.
- **Determinism:** a seed-keyed skeleton lets scripts recreate a tree exactly and lets LODs align.
- **Script surface:** a preset with a few overrides (`TREE.PINE`, seed, height, leaf color), not 50 parameters by default. Generate a few variants per species and instance them, since unique meshes per tree would lose the instancing gains.
- **Space colonization as a second step:** it needs a crown volume and an attraction point loop, and costs more at generation. It suits hero trees and shaped crowns.

### Gaps
- The Runions PDF exceeded the fetch size limit, so attraction-point steps, kill distance and pipe-model width are not sourced here.
- The Codrops write-up of ez-tree returned 403, so I used its source directly.
- I did not research L-systems. They are the classic alternative, but I have no source to cite. From memory, they are harder to art-direct than per-level parameters.

## How do you get the most from CSM at long distance?

### Takeaway
Stabilize cascades, clamp them to visible depth, scale filter kernels so softness is constant in world space, and draw trees into far cascades as simple proxies. Long soft shadows at low sun are a poor match for EVSM and MSM because of light bleeding, so test PCF or PCSS first on far cascades.

### Cited Findings
- Stabilization fits a sphere around the frustum slice so projection size does not change with rotation, and snaps translation to texel increments. This costs effective resolution, and FOV or near and far changes can cause trouble — [Pettineo](https://therealmjp.github.io/posts/shadow-maps/)
- Clamping cascades to the visible min and max Z from a GPU reduction gives tighter partitions, and min Z matters most. Deferred readback avoids a stall but makes cascades trail by a frame, which can cause artifacts when moving toward geometry. Velocity prediction helps only for constant motion — [Pettineo](https://therealmjp.github.io/posts/shadow-maps/)
- A GPU-driven setup was about 0.8 ms slower than CPU with one frame of latency, and about 1 ms faster than CPU with zero latency, on the author's PC — [Pettineo](https://therealmjp.github.io/posts/shadow-maps/)
- SDSM adjusts a fixed number of partitions from the frame's shadow samples, keeping predictable cost and constant memory, with no manual tuning and noticeably better quality than static Z-partitioning — [Intel](https://www.intel.com/content/www/us/en/developer/articles/technical/sample-distribution-shadow-maps.html). It also reduces under-sampling, over-sampling and geometric aliasing — [hgpu summary of the I3D 2011 paper](https://hgpu.org/?p=6085)
- Scaling kernel width per cascade keeps filter size constant in world space, which largely hides seams even without cross-cascade blending — [Pettineo](https://therealmjp.github.io/posts/shadow-maps/)
- VSM-family methods bleed light when the occluder-to-receiver depth range is large. Bleeding reduction over-darkens, and raising EVSM or MSM bias can increase bleeding — [Pettineo](https://therealmjp.github.io/posts/shadow-maps/)
- Bias starting points: manual bias about 0.0015, with offset scale 1.0 to 2.0 for PCF. Normal offset shifts shadows slightly. Receiver-plane bias works well but can misbehave in degenerate cases — [Pettineo](https://therealmjp.github.io/posts/shadow-maps/)
- CRYENGINE renders farther cascades once and keeps them, sampling them without further draw calls. It cites over 16 km range and about 130 MB VRAM by default. This comes from a search snippet, because the fetch returned only the page's navigation — [CRYENGINE docs](https://www.cryengine.com/docs/static/engines/cryengine-3/categories/1114113/pages/21267738)
- A forum post says Crytek aligned cascades to light space because view dependence complicates caching, at the cost of wasted texels — [Unreal forum](https://forums.unrealengine.com/t/4-13-released/72651?page=4)
- Unigine recommends a separate low-detail foliage LOD for shadow casting (from a search snippet) — [Unigine](https://developer.unigine.com/ch/docs/2.20/content/tutorials/vegetation/?rlang=cpp). CRYENGINE lists a Shadow Proxies page — [CRYENGINE docs](https://www.cryengine.com/docs/static/engines/cryengine-3/categories/1114113/pages/21267738). A 2023 paper proposes static foliage shadow imposters in Unreal — [SPIE](https://nanolithography.spiedigitallibrary.org/conference-proceedings-of-spie/12784/127842E/Unreal-engine-nanite-foliage-shadow-imposter/10.1117/12.2692451.full)
- A far cascade with a short range can produce a hard cutoff line, so fading it properly matters — [Unreal forum](https://forums.unrealengine.com/t/the-far-shadow-cascade-for-the-landscape-is-using-the-distancefield-shadow-distance-to-calculate-how-to-fade-out/445122); [UE-71574](https://issues.unrealengine.com/issue/UE-71574)

### Inferences
- **Tree proxies are nearly free here.** moonBASIC's existing cones and spheres are ready-made far-cascade casters. Skipping alpha test also reduces shadow overdraw. Keep the trunk in the proxy, or tree shadows will float.
- **Low sun tips:**
  - Long shadows mean casters far outside the view frustum still matter, so extend the light-space near range or use depth clamping (`GL_DEPTH_CLAMP`, core since 3.2 from memory).
  - Cull casters by a swept volume along the light direction rather than by the camera frustum.
- **Filter choice:** PCF near and far with a world-scaled kernel, PCSS only for the first two cascades, and EVSM or MSM only as an option. Tall trees at low sun have large depth ranges, which is the condition that triggers bleeding.
- **Without compute:** a fragment-shader depth reduction into a small R32F or RG32F target, with PBO readback or delayed readback, can provide the min and max depth on 3.3. Accept one to three frames of latency and pad the range. Do min and max first, since that alone provides most of the tightening per the source's emphasis on min Z. Full SDSM histogram work is likely not worth it.
- **Stabilization conflict:** moving partitions from depth reduction conflicts with stable sphere-fitted cascades. Snap partition distances to coarse steps, or hold them with hysteresis.
- **Caching:** update far cascades on a round-robin schedule (near every frame, the last every 4 to 8 frames), which is simpler than full caching. This suits a mostly static landscape but needs separate handling for moving objects and for a moving sun.
- **Soft look:** a light tint in shadow (not black) plus a kernel that grows with cascade distance matches the "long soft shadow" target at low cost.

### Gaps
- The SDSM slides and course notes were not read, so its histogram and partition math are not sourced.
- I found no official Epic documentation on tree shadow LODs.
- I found no production account of combining SDSM with cached cascades.
- I did not retrieve a source confirming the "compute is core only from GL 4.3" point, because the Khronos wiki fetch failed. Godot's OpenGL Compatibility renderer lacking compute is consistent with it — [Godot renderers](https://docs.godotengine.org/en/stable/tutorials/rendering/renderers.html)

## Which tone mapper and grade give rich color?

### Takeaway
No source declares a winner. Tone mappers are trade-offs, and the neutral ones are designed to leave saturation to a grade. For strong color, pair a well-behaved transform with an explicit grade rather than relying on the Narkowicz fit's oversaturation, and offer it per profile.

### Cited Findings
- **Narkowicz fit.** The author calls it "a very simple luminance only fit, which over saturates brights" and points to Hill's fit for more realism. The curve is pre-exposed, and matching original ACES requires multiplying the input by 0.6. Maximum fit error is 0.0138 — [Narkowicz](https://knarkowicz.wordpress.com/2016/01/06/aces-filmic-tone-mapping-curve/)
- **Hill-derived ACES.** three.js wraps `RRTAndODTFit` between input and output matrices (sRGB to AP1 with RRT_SAT, ODT_SAT back to sRGB). It scales exposure by 1/0.6, a factor its comment calls subjective — [three.js](https://raw.githubusercontent.com/mrdoob/three.js/dev/src/renderers/shaders/ShaderChunk/tonemapping_pars_fragment.glsl.js):
```glsl
vec3 a = v * ( v + 0.0245786 ) - 0.000090537;
vec3 b = v * ( 0.983729 * v + 0.4329510 ) + 0.238081;
return a / b;
```
- **ACES itself.** ACES 1.0 had documented clipping, hue and saturation issues that the ACES 2.0 effort aimed to address — [Display Daily](https://displaydaily.com/a-proposal-for-aces-2-0/). ACES has been described as not neutral, with intentional hue shifting and a specific feel. This phrase appeared in Bevy doc search results, but my two result summaries attached it to different variants — [Bevy docs](https://doc.qu1x.dev/bevy_trackball/bevy/core_pipeline/tonemapping/enum.Tonemapping.html)
- **AgX.** Bevy says "Very neutral. Image is somewhat desaturated when compared to other tonemappers" — [Bevy docs](https://doc.qu1x.dev/bevy_trackball/bevy/core_pipeline/tonemapping/enum.Tonemapping.html). Lower saturation than ACES is expected out of the box — [three.js forum](https://discourse.threejs.org/t/is-agx-tonemapping-implemented-correctly/60609). three.js implements it with two matrices, a log2 range of -12.47393 to 4.026069 EV, and a 6th-degree polynomial contrast fit — [three.js](https://raw.githubusercontent.com/mrdoob/three.js/dev/src/renderers/shaders/ShaderChunk/tonemapping_pars_fragment.glsl.js)
- **Khronos PBR Neutral.** It reproduces base colors exactly for channels between 0.08 and 0.8 under unit white light after an F90 offset, has no hue shifts, stays in [0, 1], and applies no gamut mapping — [Khronos](https://github.com/KhronosGroup/ToneMapping/blob/main/PBR_Neutral/README.md). Key lines — [three.js](https://raw.githubusercontent.com/mrdoob/three.js/dev/src/renderers/shaders/ShaderChunk/tonemapping_pars_fragment.glsl.js):
```glsl
float offset = x < 0.08 ? x - 6.25 * x * x : 0.04;
float newPeak = 1. - d * d / ( peak + d - StartCompression );
float g = 1. - 1. / ( Desaturation * ( peak - newPeak ) + 1. );
return mix( color, vec3( newPeak ), g );
```
- Blender developers say Neutral is not generally better than filmic tone mappers, just better for some uses, and one report describes negative shadows with saturated lights — [Blender devtalk](https://devtalk.blender.org/t/adding-pbr-neutral-tone-mapper/33602?page=2). Bevy states "Despite its name, it is not considered to be neutral" — [Bevy docs](https://doc.qu1x.dev/bevy_trackball/bevy/core_pipeline/tonemapping/enum.Tonemapping.html)
- **Tony McMapface.** It is "intentionally boring", adds no contrast or saturation, stays close to input where compression is not needed, and desaturates brights. It ships as a LUT with no parameters, and its author suggests grading before or after it. Output is linear, so the sRGB OETF is applied afterward — [Tony McMapface](https://github.com/h3r2tic/tony-mc-mapface). Bevy lists it as "Very neutral. Subtle but intentional hue shifting", and loads it as a 3D LUT from KTX2 — [Bevy docs](https://doc.qu1x.dev/bevy_trackball/bevy/core_pipeline/tonemapping/enum.Tonemapping.html); [Bevy source](https://doc.qu1x.dev/bevy_trackball/src/bevy_core_pipeline/tonemapping/mod.rs.html). A secondary book lists it as Bevy's default — [Bevy cheatbook](https://bevy-cheatbook.github.io/graphics/hdr-tonemap.html)
- Sources present these as trade-offs. An interactive comparison is mentioned at tone-mapping.glitch.me, which I did not visit — [three.js forum](https://discourse.threejs.org/t/tone-mapping-overview/75204)

### Inferences
- **Pitfalls of the Narkowicz fit for moonBASIC:**
  1. Saturated greens and warm sun colors skew and oversaturate in brights, which can read as "rich" at first but breaks on highlights.
  2. It has no working-space matrices.
  3. Its pre-exposure means switching to Hill or AgX changes brightness by about 1/0.6, or about 1.67 times, so re-tune exposure when switching.
  4. Its toe darkens low values (see the shade section).
- **Neutral versus Narkowicz in shade (my arithmetic).** For a dark leaf color of (0.01, 0.02, 0.004), Narkowicz gives about (0.0038, 0.0105, 0.0011). Neutral gives about (0.0061, 0.0161, 0.0001). Neutral keeps green brighter here and pushes the darkest channel to almost zero, which raises saturation but can reinforce the negative-shadow concern above.
- **Recommendation to test:**
  - Stylized: Khronos Neutral, because authored palette colors survive and its look is simple, plus a light saturation and split-tone grade.
  - Advanced: Hill ACES for continuity with the current look, or AgX with a punchier look if hue stability in sunlit foliage is preferred.
  - Classic: Neutral or a simple curve.
  - Make the tone mapper a per-profile setting with per-profile exposure.
  - Compare on a sunlit tree, a shaded tree, a sunset and a saturated emissive before choosing.
- **What a grade adds:** it provides the actual "look": lift and tint shadows (cool), warm highlights, a saturation bump in midtones and a slight contrast curve. Apply exposure, then the tone map, then an LDR grade through a 32x32x32 `sampler3D` LUT, or as simple math for Classic. LUT output is easy for scripts to change, and interpolated blending between LUTs supports time of day.
- **Grade pitfalls:** do not clip by adding saturation in linear HDR before the tone map. Apply dithering if banding appears on skies. Grade authored for one tone mapper will not transfer to another.
- **Tony McMapface:** feasible on 3.3 through a 3D texture. Shipping a binary LUT is at odds with a generated-assets approach and it has no tuning, so it is a lower priority for a look that seeks strong color.

### Gaps
- I found no direct hue-shift or clipping analysis of the Narkowicz fit beyond the author's own statement, nor a head-to-head test of all four mappers.
- Bevy's exact ACES wording should be checked, because the fetched extract only showed "Same base implementation that Godot 4.0 uses for Tonemap ACES" for that variant.
- The iolite AgX page gave only a summary, so AgX "punchy" and "golden" look constants are not sourced here. three.js credits that page and Filament PR 7236 for the implementation.
- Tony McMapface LUT dimensions were not stated in what I read.

## What are the cheap wins for object materials?

### Takeaway
Add geometric specular AA first, because it is only a few lines. Then add Toksvig for normal-mapped objects and screen-space contact shadows. Multi-scattering is a cheap correction if a DFG LUT already exists, but it mainly helps rough metals, so it will do little for a foliage landscape.

### Cited Findings
- Kaplanyan et al. (2016) treat highly glossy highlights as the main source of specular aliasing and filter the NDF over a pixel footprint — [NVIDIA paper](https://research.nvidia.com/sites/default/files/pubs/2016-06_Filtering-Distributions-of/NDFFiltering.pdf). Tokuyoshi and Kaplanyan (2021) target instability at grazing angles — [JCGT](https://www.jcgt.org/published/0010/02/02/paper-lowres.pdf)
- Unity HDRP's `FilterRoughness_TOKUYOSHI` starts from `ddx(n)` and `ddy(n)`, and combines the result with a normal-map variance term. It also notes that normal map hardware filtering should be trilinear to be conservative — [Unity commit](https://plastichub.unity.cn/unity-tech-cn/ScriptableRenderPipeline/commit/2871eb29-f4d2-44f9-8032-a95e66beeed9)
- Filament exposes `specularAntiAliasingVariance` with default 0.15 and `specularAntiAliasingThreshold` with default 0.2. Higher variance can add unwanted roughness, and the feature is most effective on glossy materials — [Filament Materials](https://google.github.io/filament/Materials.md.html)
- Clamping roughness also reduces aliasing. Frostbite clamps to 0.045 for analytical lights. Filament clamps to 0.089 to avoid fp16 precision problems — [Filament docs](https://google.github.io/filament/Filament.md.html)
- Toksvig (2005) computes normal variation in a mip chain by fitting a Gaussian NDF — [Han et al.](https://www.cs.columbia.edu/cg/normalmap/normalmap.pdf)
- Multi-scattering: single-scatter Cook-Torrance loses energy at high roughness and metals darken most. The compensation is `vec3 energyCompensation = 1.0 + f0 * (1.0 / dfg.y - 1.0); Fr *= pixel.energyCompensation;`, read from the DFG LUT at negligible cost — [Filament docs](https://google.github.io/filament/Filament.md.html)
- Screen-space shadows greatly improve distant shadow detail and complement shadow maps by capturing small details. Bend says Days Gone (2019) was the first to use them for sun shadows (a studio claim), and published a SIGGRAPH 2023 talk and example code — [Bend blog](https://www.bendstudio.com/blog/inside-bend-screen-space-shadows/); [Bend SIGGRAPH](https://www.bendstudio.com/siggraph)
- GTAO multi-bounce uses the diffuse albedo, or F0 for specular — [LYGIA](https://git.gay/kaneraZh/lygia/src/branch/cpp_bundle/lighting/common/gtaoMultiBounce.glsl)
- Specular occlusion is referenced by Filament through a more correct f90, but I did not reach that section — [Filament docs](https://google.github.io/filament/Filament.md.html)

### Inferences
- **Specular AA.** The widely used form is below. It is from memory and matches the cited pieces (`ddx(n)`, `ddy(n)` and Filament's two defaults), but check it against the Tokuyoshi paper before shipping.
```glsl
vec3 du = dFdx(N), dv = dFdy(N);
float variance = sigma2 * (dot(du, du) + dot(dv, dv));   // sigma2 = 0.15
float kernel   = min(2.0 * variance, kappa);             // kappa  = 0.2
alpha2 = clamp(alpha2 + kernel, 0.0, 1.0);               // alpha2 = roughness^4 in the Filament convention
```
- **Toksvig**, also from memory and not verified: when normals are stored unnormalized in mips, shorter average normals indicate more variance. `variance = (1 - |n|) / |n|` can be added to `alpha2`. It requires a normal map whose mips were not renormalized. Generated normal maps make this easy to control.
- **Contact shadows on 3.3.** Bend's implementation is a published reference, but I could not confirm its form. A short depth-buffer ray march toward the sun in a fragment pass (roughly 8 to 16 steps, with a thickness check) is feasible. Fade by distance, and exclude or weaken foliage, whose noisy depth causes artifacts.
- **AO on objects.** Apply GTAO to ambient only, add multi-bounce using luminance albedo as Blender does, and derive specular occlusion from it to stop bright reflections in crevices. Generated meshes can also bake simple vertex AO cheaply.
- **Multi-scattering.** Do this only if the existing PBR already has a DFG LUT. Otherwise, it is a low priority for this look.

### Gaps
- I could not extract the JCGT paper's shader listing. The PDF was compressed, and rendering was unavailable.
- The Bend blog gave only a high-level overview. Its ray march method, costs and any compute dependence were not read. Treat the fragment port's feasibility as unverified.
- The Filament specular occlusion section lies beyond the part of the document I read.
