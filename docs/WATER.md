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

Sum of up to 4 waves on a **camera-centered LOD grid**. With `SetWaterFollow(True)` the mesh is rebuilt in bands so the water under the camera stays about **0.5 m** per cell out to 40 m, then 1.2 m, 4 m, and 28 m out to the horizon. That is what stops the ocean reading as huge flat triangles. `CreateWater` `segs` is a density hint, not a hard vertex cap. `SetWaterFollow(False)` rebuilds a uniform pond mesh of `w`×`d`.

- Phase clock is `t = Time * WaveSpeed` (per-wave `ω` still comes from `WaveLen.y`)
- Horizontal chop `Q * A * D * cos(k·xz − ωt)` and height `A * sin(...)`
- Analytic normals from the same terms
- `SetWaterWind(dirX, dirZ [, strength])` steers directions and boosts aligned amplitude
- `SetWeather("storm")` scales amplitude (`WaveStorm`)
- CPU `WaterHeight(x, z)` matches the shader (including wind) for `CreateBuoy`

## Scenic (fragment + two cameras)

The surface is **opaque**. `SetWaterColor` is the deep-water color. Looking across the waves shows the reflection; looking down shows refraction, then the body color. You should not see the beach through the ocean.

- Reflection FBO: the world **above** the plane (sky included). Refraction FBO: the world **below** it.
- Clip is a fragment discard (`WaterClipPlane`), not `gl_ClipDistance`. The skybox shader does not write a clip distance, so a hardware clip would delete the sky.
- Terrain is one mesh that crosses the plane, so it is **not** hidden for the water pass. The clip splits it: sand above stays in the reflection, the seabed stays in the refraction.
- Projective UVs (clip → NDC → 0..1). Reflection V is flipped (`1 - ndc.y`).
- DuDv scroll (`SetWaterSpeed` / `SetWaterWaveStrength`), scaled down at the shore and again with camera distance so the far mirror stays readable. Default maps are generated.
- Fresnel is Schlick (F0 0.02): glancing → reflection, looking down → the body color. The shore mix stays mostly refraction so a cube’s reflection does not ghost onto the sand.
- Refraction only replaces the body color when the seabed is actually close (about a quarter-metre to a few tens of metres). Sky and the far plane stay the deep color.
- Specular is a broad sun path plus a tight sparkle (`WaterShine` and a 512 lobe) on the ripple normal, not a flat face normal. Face normals are only a fallback if the analytic normal is broken. `examples/canal.bb` is the low-camera, dark-green, hazy look.
- Above the water, back faces are discarded so a wave underside is not a dark wedge in the corner.
- Alpha is 1, so a reflection does not fade onto the shore.

`SetWaterColor(6, 48, 72)` with `SetWaterStyle("ocean")` and `SetWaterFollow(True)` is the ocean demo.

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
| Tessellation LOD | **Not required** — banded LOD grid on 3.3 (0.5 m cells near the camera) |

Also: `examples/largeworld.bb`.
