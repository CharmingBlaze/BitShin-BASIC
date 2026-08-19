# Water

Two techniques share one mesh and the `mbwater` shader (**OpenGL 3.3**, `#version 330 core`). Pick with `SetWaterStyle` or by turning waves / FBOs on yourself.

| Style | What you see | Demo |
| --- | --- | --- |
| **Gerstner** (primary) | Rolling trochoidal ocean: steep crests, directional trains, wind, analytic normals, `WaterHeight` for buoys | `examples/gerstner.bb` |
| **Scenic / LearnOpenGL** | Flat plane + reflection FBO + refraction FBO + DuDv scroll + Fresnel + specular normals + depth tint | `examples/water.bb` |
| **Ocean** (combo) | Gerstner vertex displace **on** scenic shading | `examples/ocean.bb` |

References (behavior reimplemented clean-room in GLSL 330 + Go; no tess/compute):

- GPU Gems I ch.1 (Finch) trochoids — same family as [CaffeineViking/osgw](https://github.com/CaffeineViking/osgw) (`gerstner.glsl`, MIT). The named [danilafe/gerstner-waves](https://github.com/danilafe/gerstner-waves) repo is gone (404).
- [moomoo02 Medium “pretty water”](https://medium.com/@vincehnguyen/simplest-way-to-render-pretty-water-in-opengl-7bce40cbefbe) + ThinMatrix FBO water. LearnOpenGL has **no** water chapter; `src/5.advanced_lighting/6.advanced_lighting` is HDR/gamma, not water.

## How to pick

```basic
SetWaterStyle("gerstner")     ; choppy ocean, still uses planar FBOs
SetWaterStyle("scenic")       ; flat pretty water (DuDv / Fresnel)
SetWaterStyle("ocean")        ; both (default CreateWater)
```

Or: `SetWaterWaves(0)` for a still plane, `SetWaterWaves(4, 0.4)` for Gerstner. `EnableWaterReflection(True)` turns on both FBOs.

## WaterMaterial (G3N)

`CreateWater` attaches a `WaterMaterial` (G3N `Standard` + `mbwater` program). Fields and methods:

| API | Role |
| --- | --- |
| `Time` | Seconds since create; `Update(delta)` does `Time += delta` each Flip |
| `WaveSpeed` | Multiplier on the Gerstner clock (default `1`). `SetWaterWaveSpeed` / `GetWaterWaveSpeed` |
| `Update(delta)` | Advance `Time`; sync CPU buoy clock (`wb.time`) and DuDv `WaterMove` |
| `Bind` / `RenderSetup` | Upload `Time`, `WaveSpeed`, Gerstner trains, FBO/DuDv/normal, softened `ShadowMap` |

Vertex phase is `Time * WaveSpeed`. `SetWaterSpeed` only scrolls DuDv (`WaterMove`); it does not replace the Gerstner clock.

## Gerstner (vertex, GL 3.3)

Sum of up to 4 waves on a **camera-centered LOD grid** (sinh warp: dense near the camera, coarse toward a far radius so the horizon stays filled). `CreateWater` `segs` is inner density (24–160), not a 128 world-size cap. `SetWaterFollow(False)` rebuilds a uniform pond mesh of `w`×`d`.

- Phase clock is `t = Time * WaveSpeed` (per-wave `ω` still comes from `WaveLen.y`)
- Horizontal chop `Q * A * D * cos(k·xz − ωt)` and height `A * sin(...)`
- Analytic normals from the same terms
- `SetWaterWind(dirX, dirZ [, strength])` steers directions and boosts aligned amplitude
- `SetWeather("storm")` scales amplitude (`WaveStorm`)
- CPU `WaterHeight(x, z)` matches the shader (including wind) for `CreateBuoy`

## Scenic (fragment + two cameras)

- Reflection: mirrored camera, hide fully-underwater meshes, color FBO
- Refraction: same camera, hide fully-above-water meshes, color + **depth** texture
- Projective UVs (clip → NDC → 0..1)
- DuDv scroll (`SetWaterSpeed` / `SetWaterWaveStrength`); default maps are generated
- Fresnel: glancing → reflection, looking down → refraction
- Specular from the directional light + water normal map
- Depth coloring from the refraction depth texture (soft deep-blue tint)

Skybox uses G3N’s default shader (no `gl_ClipDistance`), so clip is **visibility hide**, not a hardware clip plane.

Terrain-OpenGL’s water plane (reflection + refraction FBOs, DUDV) is this same `CreateWater` system — not a second ocean.

## Commands

```basic
water = CreateWater(140, 140, 72)
SetWaterStyle("gerstner")
SetWaterColor(12, 62, 88)
SetWaterWaves(4, 0.42)
SetGerstner(0, 0.85, 0.35, 0.42, 0.55, 18, 1.15)
SetWaterWind(0.9, 0.25, 0.7)
EnableWaterReflection(True)
EnableWaterRefraction(True)
SetWaterSpeed(0.04)
SetWaterWaveSpeed(1)
SetWaterWaveStrength(0.05)
SetWaterDuDv(tex)
SetWaterNormalMap(tex)
SetWaterLevel(0)
SetWaterFollow(True)
SetWaterCaustics(water, True)
SetWaterSSR(water, True)
SetWaterAmbientSound(water, "sea.ogg", GetWeatherIntensity())
y# = WaterHeight(x, z)
CreateBuoy(boat)
SetWaterFlow(1.5, 0, 0.2)
SetBuoyancyFactor(crate, 1.2)

```

Aliases: `GetWaterHeight`, `SetWaterReflection`, `SetWaterRefraction`, `SetWaterDuDvMap`, `SetWaterWindDir`, `SetWaterMode`, `EnableWaterCaustics`, `GetWaterCaustics`, `EnableWaterSSR`, `GetWaterSSR`, `SetWaterAmbient`, `GetWeatherIntensity`.

## Already there vs later

| Feature | Status |
| --- | --- |
| Foam on Gerstner peaks | Cheap height foam in the fragment shader |
| Dual scrolling normal maps | `WaterNormal` sampled twice and mixed with Gerstner / dFdx normals |
| Underwater fog | `SetUnderwaterFog` when the camera is below `WaterHeight` |
| Buoyancy / splash / wakes | `CreateBuoy` + CPU impulse buffer (`WaterWake`). Auto `ApplyBuoyancyImpulse` for submerged dynamics (`SetBuoyancyFactor`, `SetWaterFlow`). Flag demo: `examples/cloth.bb` |
| Caustics | Animated projected pattern on `mbterrain` / underwater meshes (`SetWaterCaustics`) |
| Shoreline wetness | Terrain darken + spec near `WaterLevel` using the `Wetness` term |
| SSR | **Partial** — lite screen march in the planar reflection FBO, fades to planar if off-screen / unstable. Not deferred SSR |
| FFT / compute ocean | **Never required** (needs GL 4.3+) |
| Tessellation LOD | **Not required** — sinh LOD grid on 3.3 |

Also: `examples/largeworld.bb`.
