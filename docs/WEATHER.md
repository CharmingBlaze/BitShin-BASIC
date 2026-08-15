# Weather (OpenGL 3.3)

BitShin BASIC weather is a **G3N / GLSL 330** stack: sky + clouds + particles + height fog + wetness + wind + lightning. It does **not** require OpenGL 4.5 (no compute, no 3D textures, no froxel volumes).

Demo: `examples/weather.bb` (keys **1–5**, **L** strike, WASD + mouse). Platform 64 still uses the same `SetWeather` keys.

## What each reference became

| Source | License / notes | What we shipped |
| --- | --- | --- |
| [ebruneton/precomputed_atmospheric_scattering](https://github.com/ebruneton/precomputed_atmospheric_scattering) | BSD-3-Clause | **Partial.** Full 4D inscatter tables are huge and the reference often assumes newer sampling. We bake a **256×64 transmittance LUT in Go** and march **12-step single scatter** in `mbatmo` (GLSL 330). `CreateSkyBox("default")` also paints a CPU Bruneton-ish cubemap so a skybox is never a flat hex. |
| [kevinmoran/Clouds](https://github.com/kevinmoran/Clouds) | typical sample (reimplemented) | Layered 2D-noise raymarch on a dome (`mbclouds`). Same idea as Terrain-OpenGL clouds: **not** compute 3D Worley. `SetClouds` / `CreateVolumetricClouds`. Wind scrolls the noise. |
| LearnOpenGL skybox | educational | Cubemap path already existed (`CreateSkyBox` / `LoadSkyBox`). Extended, not replaced. |
| [Polytonic/ParticleSystem](https://github.com/Polytonic/ParticleSystem) | typical MIT sample | Behavior mapped onto **our existing 3D emitters** (rate, life, wind, gravity). Rain = tall camera-yaw streaks (`mbpart`). Snow/ash/wisps = soft flakes (`mbflake`). |
| [kevinmoran/RainShader](https://github.com/kevinmoran/RainShader) | typical sample | Streak particles + **wetness** on `mbshadow` / `mbphysical` / `mbterrain`: darker upward faces + extra spec when `SetWeather` is rain/storm (or `SetWeatherWetness`). |
| [fogleman/lightning](https://github.com/fogleman/lightning) | algorithm (midpoint displacement) | Procedural 3D bolts (ribbon mesh + fork) + ambient/clear flash. `StrikeLightning` / storm auto. |
| [kevinmoran/Fog](https://github.com/kevinmoran/Fog) | typical sample | Existing linear/exp/exp² **plus height falloff** (`SetFogHeight` / `CameraFogHeight`). Not a froxel volume. |
| [kevinmoran/Wind](https://github.com/kevinmoran/Wind) | typical sample | `SetWind(dir, strength)` drives particles, cloud scroll, and **CPU sway** on `SetWindSway` entities (grass/trees). Not a full tree vertex-skin system. |

Reimplemented in-tree. We do **not** vendor those repos or require their assets.

## Commands

See `docs/COMMANDS.md` (Weather). Short list:

- `SetWeather` (default **1.5s** blend) / `SetWeatherTransition(mode$, intensity, seconds)` / `SetWeatherIntensity` / `Weather$` / `WeatherIntensity`
- `SetWind` / `SetWeatherWind` / `SetWindSway`
- `CreateAtmosphere` / `SetAtmosphere` / `SetSunDirection` / Rayleigh / Mie / turbidity
- `SetClouds` / `CreateVolumetricClouds` / coverage / density / speed
- `StrikeLightning` `[x,y,z]` or six-point bolt — thunder delay = distance/343; quieter/lower pitch when far. Missing `thunder.ogg` prints once and uses a generated Oto rumble (does not End).
- `SetWeatherWetness` / `SetSurfaceWetness` / `WeatherWetness` / `SetWeatherDryingSpeed`
- `SetCameraRain on` — lightweight fullscreen streaks (rain/storm + high wetness)
- `SetFog` / `CameraFog*` / `SetFogHeight` / `EnableHeightFog`

## Transitions

`SetWeatherTransition` sets `targetMode` / `targetIntensity` / `targetWind*` and lerps wind, fog, clouds, ambient, and intensity with a smoothstep. Fog and wind are **not** snapped.

**Particle crossfade = rate ramp** (not a transparent pile-up): outgoing weather emitters scale `rate` to 0 over the duration; incoming emitters spawn immediately at a low rate and ramp up. A slight alpha fade is applied. At `t >= 1`, `mode = targetMode` and the old emitters are freed.

## Wetness

Rain/storm: `wetness += precipRate * dt` (from intensity). Clear: `wetness -= dryingSpeed * dt` (default `0.01`). Clamped 0–1 and pushed as the `Wetness` uniform on terrain / `mbshadow`. `SetSurfaceWetness` / `SetWeatherWetness` lock the value (`wetOwned`).

Rain/snow particles that die or reach `TerrainHeight` / water spawn a ground splash ring or a water wake impulse (`wakeImpulse` / splash).

## Honest 3.3 limits

| Feature | Status |
| --- | --- |
| Bruneton full precompute (4D inscatter + multiple scatter) | **Partial** — LUT + single scatter only |
| True volumetric clouds (3D noise / compute) | **No** — 12-step 2D layers |
| Froxel / raymarched fog | **No** — distance + height exp |
| Rain wetting as a screen-space post | **Partial** — material wetness + optional camera streaks (`SetCameraRain`) |
| Wind on skinned trees | **No** — entity pitch/roll sway |
| OpenGL 4.5 | **Never required** |

Water still scales Gerstner with storm (`docs/WATER.md`). Shadows stay on.
