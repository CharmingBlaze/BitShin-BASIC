# Commands

Names are case-insensitive. Optional arguments in brackets. Handles are integers. Angles are **degrees**. RGB is **0–255**, or **0–1 if every channel is ≤ 1**. Paths resolve next to the `.bb` file, or inside `OpenPak` if set.

`Flip`, `WaitTimer`, `WaitKey`, and `Delay` yield a frame and are not allowed inside `Function`.

## Display / loop

| Command | Meaning |
| --- | --- |
| `Graphics3D w, h [, depth, mode]` | G3N window. `mode` 1 = fullscreen |
| `Graphics2D w, h` / `Graphics w, h` | Ebiten window |
| `AppTitle s$` | Window title |
| `EndGraphics` | Request quit |
| `SetBuffer BackBuffer()` | No-op (classic) |
| `Flip` | Present and yield one frame. Escape / `WindowShouldClose` stay false until the first Flip |
| `Cls` | Clear 2D draw list / 3D HUD text |
| `ClsColor r, g, b` | 2D clear color (0–255 or 0–1) |
| `CameraClsColor r, g, b` | 3D background (0–255 or 0–1) |
| `Color r, g, b` | Draw / text color (0–255 or 0–1) |
| `Text x, y, s$` | HUD text (2D uses `LoadFont` if set) |
| `DeltaTime()` / `FrameTime()` | Seconds since last frame |
| `MilliSecs()` | Milliseconds since start |
| `GraphicsWidth()` / `GraphicsHeight()` | Current window size (GLFW or last `Graphics*`) |
| `ClearWorld` | Free entities/sprites/emitters; keep the window |
| `LoadScene file$` | Run a setup `.bb` (no `Flip`/`End`) on the same variables |
| `OpenPak file$` | Zip archive; later `Load*` look here if the disk file is missing |
| `FileType(path$)` | 0 missing, 1 file, 2 directory |

## 3D scene (G3N)

`CreateCamera([parent])`, `CreateFreeCamera([parent])` — free-look hides/raw-captures the mouse  
`CreateCameraOrtho([parent, size])`, `CameraProjMode cam, mode` — 0 perspective, 1 ortho  
`UpdateFreeLook cam [, speed]` — WASD + mouse look (`speed` units/sec, default 8). `examples/freelook.bb`  
`CreateLight([type, parent])` — 1 directional, 2 point, 3 spot  
`CreateCube([size|parent])`, `CreateBox(w, h, d [, parent])` or `CreateBox(w, h, d, segW, segH, segD [, parent])`  
`CreateSphere([segs, parent])` or `CreateSphere(radius, segs, parent)`  
`CreateCylinder([segs, parent])` or `CreateCylinder(radius, height, segs [, caps, parent])` — **mesh**. Collider is `CreateBodyCylinder(e, halfH, r)` (`halfH` is half of `height`).  
`CreateCone` (same extra args as cylinder), `CreatePlane([parent])` or `CreatePlane(w, h [, parent])` — **mesh only**, not an aircraft. Fly with `CreatePlaneController` ([VEHICLES.md](VEHICLES.md)).  
`CreateTorus([parent])` or `CreateTorus(major, minor [, radial, tubular] [, parent])`  
`CreateCapsule(radius, height [, segs, parent])`, `CreateDisk` / `CreateCircle(radius [, segs, parent])`  
`CreatePyramid([size, parent])`, `CreateWedge(w, h, d [, parent])`, `CreateTube(radius, height [, segs, parent])`, `CreateQuad([size, parent])`  
`CreatePivot`  
`LoadMesh file$ [, parent]` — `.obj`, `.gltf`, `.glb`, `.dae`  
`CopyEntity src [, parent]`, `FreeEntity`, `HideEntity`, `ShowEntity`, `EntityVisible(e)`  
`PositionEntity e, x, y, z`, `MoveEntity`, `TranslateEntity`, `TurnEntity`, `RotateEntity`, `ScaleEntity`, `PointEntity`  
`EntityColor e, r, g, b`, `EntityAlpha e, a`, `EntityAmbient e, r, g, b`, `EntityShininess e, n`, `EntitySpecular e, r, g, b`, `EntityEmission e, r, g, b`  
`Material(r, g, b [, tex, shine, sr, sg, sb])` or `Material($RRGGBB [, tex, shine [, sr, sg, sb]])` — reusable color, texture, and shine. `EntityMaterial e, mat` / `e.Material(mat)` applies it. Passing `sr, sg, sb` sets specular.  
Dot methods: `cam.Position(x,y,z)`, `box.Color(r,g,b)`, chaining `CreateCube().Scale().Position().Color()`. Vec `[x,y,z]` and `$RRGGBB` / `Hex("RRGGBB")` expand on those commands. Setters return the entity handle.  
`EntityX/Y/Z(e [, global])`, `EntityPitch/Yaw/Roll`, `EntityScaleX/Y/Z`, `EntityDistance a, b` — pass `True` for a parented entity's world position.  
`EntityParent child, parent`, `GetParent(e)`, `NameEntity e, s$`, `EntityName$(e)` — `NameEntity` also sets the G3N node name (used by `AttachToBone`).  
How to light a scene, from one sun through falloff, materials, indoor / outdoor, time of day, and cookies: [LIGHTING.md](LIGHTING.md). Shadows: [SHADOWS.md](SHADOWS.md). PBR: [PBR.md](PBR.md).

| Command | Meaning |
| --- | --- |
| `AmbientLight r, g, b` / `SetAmbientColor` | Scene fill. Final color is light × material (Phong) or light × BRDF (PBR). |
| `CreateDirectionalLight` / `CreatePointLight` / `CreateSpotLight` / `CreateAmbientLight` | Same lights as `CreateLight` types 1, 2, 3, and an ambient node. |
| `LightColor` / `SetLightColor light, r, g, b` | Light RGB. |
| `SetLightIntensity light, n` | Multiplier on that color. |
| `SetLightDirection light, pitch, yaw [, roll]` | Aim a sun or a spot. |
| `SetLightCone light, innerDeg, outerDeg` | Spot cone. Squared (glTF) when falloff is `physical`. |
| `LightRange light, radius` | Point/spot reach. Default falloff is a smooth window, zero at `radius`. Shadows use that distance. |
| `SetLightFalloff light, mode$` | `smooth` (default), `classic` (LearnOpenGL polynomial), `physical` (glTF inverse-square). Physical intensity is candela for point/spot (about 80–400 in a room) and lux for a sun. |
| `SetLightAttenuation light, constant, linear, quadratic [, range]` | `1 / (constant + linear·d + quadratic·d²)`. A 32-unit lamp is `1, 0.14, 0.07`. |
| `SetLightSpecular light, r, g, b` | Highlight color. Defaults to the light color. First 8 lights of each kind. |
| `SetLightAmbient light, r, g, b` | Extra ambient from that light, on top of `AmbientLight`. Defaults to none. |
| `SetLightShadow light, on` | Directional casts by default. Point/spot need `True`. |
| `SetLightEnabled light, 0\|1` / `EnableLight` | Off stores the intensity and zeroes it. On restores it. |
| `SetLightTemperature light, kelvin` / `SetLightKelvin` | Black-body tint, 1000–40000 K. Intensity stays. 2000 candle, 6500 daylight. |
| `GetLightType` / `GetLightIntensity` / `GetLightRange` | `0` ambient, `1` directional, `2` point, `3` spot. |
| `GetLightRed` / `GetLightGreen` / `GetLightBlue` | Light color, 0–255, without intensity. |
| `EntityShininess e, n` | `n` ≤ 1 is a 0–1 weight (×128). `n` > 1 is the Phong exponent (`32`, `64`, `128`). |
| `EntityAmbient` / `EntitySpecular` / `EntityEmission` | Material ambient (call after `EntityColor`), highlight, and added glow. |
| `EntityTexture` / `SetDiffuseMap` | Diffuse map. |
| `SetSpecularMap e, tex` / `SetEmissionMap e, tex` | Specular and emission maps. |
| `SetEntityNormalMap e, tex` | Phong normal map (derivative tangents). `SetNormalMap` is the PBR command. |
| `SetLighting "outdoor"\|"indoor"` / `OutdoorLighting` / `IndoorLighting` | Daylight rig or a dark room with inverse-square lamps. `GetLighting()` is `1` or `2`. Create lights first. |
| `SetTimeOfDay hours` / `GetTimeOfDay()` | Outdoor clock. Aims every directional, sky, fog, ambient. Bands in [PLAY.md](PLAY.md). |
| `CreateThreePoint()` | Returns the key. Adds a cool fill and a rim. |
| `SetLightCookie spot, tex` | Gobo on the first two spots. Red channel, inside the cone. |
| `SetTonemap mode$` | `reinhard` (default), `neutral`, `aces`, `none`. Needs `EnablePostFX`. Pair with `SetExposure`. |
| `SetGammaCorrection 0\|1` | Phong sRGB encode when post FX is off. Leave off if `EnablePostFX` is on. |  
`CameraRange cam, near, far`, `CameraZoom cam, z`, `CameraViewport cam, x, y, w, h`  
`CameraFollow cam, target, dist, height [, damp, yaw, pitch]` — Lakitu-style ease toward an orbit point (yaw/pitch in degrees; damp is follow stiffness)  
`CameraFogMode([cam,] mode)` — 0 off, 1 linear, 2 exp, 3 exp². Distance fog on the lit/shadow shader; shadows stay on.  
`CameraFogColor([cam,] r, g, b)`, `CameraFogRange([cam,] near, far)`, `CameraFogDensity([cam,] n)`  
`EnableFog on`  
`SetFog(mode)` or `SetFog(r, g, b [, near, far])` or `SetFog(mode, r, g, b, near, far)`  
`LoadTexture file$`, `EntityTexture e, tex`, `ScaleTexture`  
`Wireframe on`, `UpdateWorld`, `RenderWorld`

Y up, +Z in front of a default camera.

## Shadows (G3N has no built-in CSM)

See `docs/SHADOWS.md`. All of these change rendering (or skip a redundant depth pass, for the cache).

| Command | Meaning |
| --- | --- |
| `EnableShadows [on]` | Depth atlas + `mbshadow` receive. **On after `Graphics3D`** (Blitz3D-style). Pass `False` to disable. |
| `ShadowCascades n` | Directional 1–4 frustum-fitted CSM splits |
| `SetShadowDistance n` | CSM far (~200–300). Fade over the last 30 units |
| `ShadowMapSize n` / `SetShadowResolution(n)` | Resolution per atlas tile (256–8192). Alias of each other |
| `SetLightShadowRes(light, n)` | Stored on the light; if it is the CSM sun, also sets `ShadowMapSize` |
| `SetShadowBias n [, normalBias]` | Depth bias; optional world-normal offset in texels (`ShadowNormalBias`) |
| `SetShadowQuality mode [, pcfTaps]` | `0` PCF, `1` PCSS, `2` EVSM, `3` MSM, or `"low"`/`"medium"`/`"high"`/`"ultra"` presets |
| `SetShadowPCF k` | PCF kernel (minimum 3) |
| `SetShadowFilter "pcf"\|"pcss"\|"evsm"\|"msm"` | Filter (also `0`–`3`) |
| `SetShadowPCSS on` | Filter `pcss` / `pcf` |
| `SetShadowEVSM [on]` | Exponential variance (Chebyshev) |
| `SetShadowMSM [on]` | Four-moment MSM |
| `SetShadowLightSize n` | PCSS penumbra scale |
| `EnableShadowCache` / `SetShadowCache` / `EnableShadowCaching` | Split static (terrain / static bodies) vs dynamic maps; static redraws when sun/dir or static key changes |
| `EnableShadowAtlas` / `SetShadowAtlas` | Packed atlas (cascades + point/spot) |
| `EnableContactShadows` / `SetContactShadows` | Stub |
| `EnableScreenSpaceShadows` / `SetScreenSpaceShadows` | Stub |
| `SetLightShadow light, on` | Directional = CSM sun (on by default for the first directional). Point/spot = extra maps, opt-in |
| `EntityCastShadow` / `EntityReceiveShadow` | Per-mesh cast/receive (aliases `CastShadow`, `ReceiveShadow`) |

## PBR (opt-in metallic-roughness)

See `docs/PBR.md`. OpenGL 3.3 / GLSL 330 (`mbphysical`). Default meshes stay Phong. Handles are entity ids **or** a material from `CreatePBRMaterial()`. RGB is 0–255 or 0–1 (if all channels ≤ 1). Metallic / roughness / AO are 0–1. PBR setters on an entity convert it (Platform 64 never calls these).

```basic
mat = CreatePBRMaterial()
SetMetallic(mat, 1)
SetRoughness(mat, 0.15)
SetAlbedo(mat, 220, 210, 190)
ball = CreateSphere()
SetMaterial(ball, mat)

ground = CreatePlane()
SetMaterialPBR(ground, 1)
SetAlbedo(ground, 48, 54, 64)
SetMetallic(ground, 0)
SetRoughness(ground, 0.85)
SetIBL(True)
```

| Command | Meaning |
| --- | --- |
| `CreatePBRMaterial()` | Library material handle (no window required) |
| `SetMaterialPBR e, on` / `EnablePBR e, on` | Convert entity to PBR (`on` 0 reverts to Phong) |
| `GetMaterialPBR(e)` | 1 if entity or library handle is PBR |
| `SetMaterial e, mat` / `SetPBRMaterial e, mat` | Apply a `CreatePBRMaterial` handle |
| `SetAlbedo e, r, g, b` / `SetBaseColor e, r, g, b` | Albedo RGB |
| `SetAlbedo e, tex` / `SetAlbedoMap e, tex` / `SetBaseColorMap e, tex` | Albedo texture (`LoadTexture`) |
| `GetAlbedoR(e)` `GetAlbedoG(e)` `GetAlbedoB(e)` | Albedo 0–255 |
| `GetBaseColorR(e)` `GetBaseColorG(e)` `GetBaseColorB(e)` | Same |
| `SetMetallic e, n` / `SetMetallicFactor e, n` | 0 dielectric … 1 metal |
| `GetMetallic(e)` / `GetMetallicFactor(e)` | |
| `SetRoughness e, n` / `SetRoughnessFactor e, n` | Perceptual roughness |
| `GetRoughness(e)` / `GetRoughnessFactor(e)` | |
| `SetAO e, n` / `SetOcclusion e, n` / `SetOcclusionFactor e, n` | Ambient occlusion 0–1 |
| `GetAO(e)` / `GetOcclusion(e)` / `GetOcclusionFactor(e)` | |
| `SetEmissive e, r, g, b` | Unlit glow (RGB; values above 255 are allowed) |
| `GetEmissiveR(e)` `GetEmissiveG(e)` `GetEmissiveB(e)` | |
| `SetNormalMap e, tex` | Tangent-space normal |
| `SetMetallicRoughnessMap e, tex` / `SetMetalRoughMap e, tex` | glTF packed MR (G rough, B metal) |
| `SetEmissiveMap e, tex` | |
| `SetAOMap e, tex` / `SetOcclusionMap e, tex` | AO in R |
| `SetEnvMap e, tex` | Per-material 2D lat-long IBL |
| `SetEnvMap tex` | Same map on every PBR material |
| `SetIBL [e,] on` / `EnableIBL on` | Hemisphere, prefiltered sky cubemap, optional 2D env. One arg = world |
| `GetIBL([e])` | |
| `SetIBLIntensity n` / `GetIBLIntensity()` | Scale (default 1) |

`LoadMesh` glTF/GLB Physical materials already use `mbphysical` (shadows, fog, these commands). IBL is an analytic hemisphere, a UE4 EnvBRDF split-sum, and a prefiltered sky cubemap (`textureLod` by roughness). The nearest light probe replaces that cubemap after its six faces have been drawn. `SetEnvMap` is still an optional 2D lat-long.

## Mesh animation (glTF)

Source must be `.gltf` / `.glb` **with clips**.

| Command | Meaning |
| --- | --- |
| `LoadAnimMesh file$ [, parent]` | Load mesh + clips; error if none |
| `Animate e [, mode, speed#, seq]` | 0 stop, 1 loop, 2 ping-pong, 3 once |
| `SetAnimTime e, t#` | Scrub |
| `AnimTime(e)`, `AnimLength(e)` | Seconds |
| `ExtractAnimSeq(e, first, last [, seq])` | New seq index from a time range |
| `SetAnimSeq e, index` or `SetAnimSeq e, name$` | Current clip |
| `AnimSeqName e [, name$]` | Get/set clip by name |
| `StopAnim e` / `StopAnimation e` | Pause (`mode` 0). **Not** the same as `Animate e` |
| `AnimPlaying(e)` | 1 if a clip is advancing |
| `SetAnimBlend e, w# [, seq\|name$]` / `BlendAnimation` | 0–1 mix of the previous clip pose into the current clip (`w=1` is current only). Passing a new seq/name stashes the old clip |
| `AttachToBone child, mesh, bone$` | Parent `child` to a named glTF node (or a named child entity). Alias `AttachBone`. See [PHYSICS.md](PHYSICS.md) |

Clips advance with `DeltaTime` each frame (Flip), not twice if you also `UpdateWorld`.

## Picking

| Command | Meaning |
| --- | --- |
| `LinePick x,y,z, dx,dy,dz [, range]` | Ray; returns entity (0 = none) |
| `RayPick` | Same |
| `CameraPick([cam, x, y])` | Screen ray; defaults to mouse / default camera |
| `PickedEntity()`, `PickedX/Y/Z()` | Last hit (`Raycast`, `GrabPick`, `PlaceAtRay`, projectile impact) |
| `PlaceAtRay src, dest [, maxDist]` | Move `dest` to the hit (or max range). Alias `RayPlace`. Default range 40 |

## 2D (Ebiten)

`LoadImage file$`, `CreateImage w, h`  
`DrawImage img, x, y`  
`DrawImageRect img, x, y, sx, sy, sw, sh [, dw, dh]`  
`CreateSprite(img)`, `PositionSprite`, `MoveSprite`, `RotateSprite s, deg`, `ScaleSprite`  
`SpriteX/Y`, `HideSprite`, `ShowSprite`  
`Rect x, y, w, h [, filled]`, `Oval`, `Line`  
`RectsOverlap`, `SpritesOverlap`, `ImagesOverlap`, `ImagesCollide`  
`LoadFont file.ttf, size` — following `Text` uses it; otherwise the debug font  
`SetFont handle`

### Tiles

`CreateTileMap tw, th, cols, rows`  
`SetTile map, tx, ty, id` — `id` 0 empty; 1+ is a tile  
`DrawTileMap map, x, y [, atlas]` — colored cells if no atlas  
`DrawTile atlas, tile, x, y, tw, th`  
`LoadTileMap file.csv [, tw, th]` — CSV of integer ids

## Skyboxes

`CreateSkyBox([prefix$])` — 6-face cubemap. Empty / `"default"` builds a procedural sky (gradient + sun). Prefix tries `px/nx/py/ny/pz/nz` (also `_px`, `right/left/up/down/front/back`) as `.png` / `.jpg`  
`LoadSkyBox(px$, nx$, py$, ny$, pz$, nz$)`  
`SetSkyBox id` `HideSkyBox` `ShowSkyBox` `FreeSkyBox([id])`  
`SetSkyColor r, g, b` — solid clear color; a visible skybox draws in front of it  
`SetSky` — string = preset (`default`/`sunset`) or cubemap prefix; 3 numbers = `SetSkyColor`; 6 numbers = `SetSkyGradient`  
`CreateAtmosphere()` `SetAtmosphere` `SetAtmosphereRayleigh` `SetAtmosphereMie` `SetAtmosphereTurbidity` `SetSunDirection` — GLSL 330 sky dome: Go-baked 256×64 transmittance LUT + 12-step single scatter (Bruneton-style **Partial**, not the full 4D table)  
`SetClouds cover [, density, speed]` / `CreateVolumetricClouds` — 3.3 layered-noise raymarch dome  

The sky follows the camera, does not write depth, and does not cast shadows.

## Weather

Particles + atmosphere + clouds + height fog + wetness + wind + lightning. Shadows stay on. **GL 3.3.** See `docs/WEATHER.md`.

`SetWeather("clear"|"rain"|"snow"|"fog"|"storm")` or `SetWeather(WEATHER_SNOW)` — constants `WEATHER_CLEAR` `WEATHER_RAIN` `WEATHER_SNOW` `WEATHER_FOG` `WEATHER_STORM` (also `0`–`4`). Default **1.5s** blend (does not pop).  
`SetWeatherTransition(mode$, intensity, seconds)` — e.g. `SetWeatherTransition("storm", 0.85, 10)`  
`SetWeatherIntensity(0-1)` `SetWeatherWind x, y, z` `SetWind dx, dy, dz [, strength]`  
`Weather$()` `WeatherIntensity()` `GetWeatherIntensity()`  
`SetWindSway ent, on [, phase]`  
`StrikeLightning` / `StrikeLightning x,y,z` / six-point bolt — thunder is delayed by distance/343 m/s (Oto; generated rumble if no `thunder.ogg`)  
`SetWeatherWetness(0-1)` `SetSurfaceWetness(0-1)` `WeatherWetness()`  
`SetWeatherDryingSpeed(0.02)` — default `0.01`. Rain/storm accumulate wetness; clear dries.  
`SetCameraRain on` — fullscreen droplet streaks when rain/storm and wetness is high  
`SetFog r, g, b, near, far`  
`SetFogHeight y, falloff` `CameraFogHeight([cam,] y, falloff)` `EnableHeightFog on`

Rain = streaks + wetness + height fog. Snow = flakes + drift. Fog = exp + height + wisps. Storm = heavy rain, ash, wind, auto bolts + flash.

Particle crossfade is a **rate ramp**: outgoing emitters scale emission to 0 over the blend; incoming spawn at a low rate and ramp up (optional slight alpha fade). No transparent pile-up.

`examples/weather.bb`: **1–5** clear / rain / snow / fog / storm, **T** long storm transition, **L** strike, **−/=** intensity, WASD + look.  
`examples/platform64.bb`: same **1–5** keys.

## Particles

One simulator for 3D and 2D: rate, life, size, color, velocity, gravity, wind, cone, drag, burst, duration, loop. Weather uses the same emitters (rain streaks, snow flakes). Call `UpdateWorld` (or `Flip`, which steps the world) or particles do not move.

### 3D

`CreateEmitter([parent])` / `CreateParticleEmitter([parent])` — also an entity, so `PositionEntity` / `ShowEntity` / `HideEntity` work. Parent it to a mesh and the spray follows that mesh.

`SetEmitterShape e, name$` / `EmitterShape` picks the look:

| Name | What you see |
| --- | --- |
| `"soft"` | Default. Camera-facing soft disc. Use this for fire, sparks, smoke. |
| `"cube"` | Lit cubes in the world. They spin and fall. Size is the cube edge. |
| `"streak"` | Thin vertical billboard. Rain uses this. |
| `"flake"` | Soft flake with spin. Snow uses this. |

```basic
em = CreateEmitter(cube)
SetEmitterShape(em, "soft")
SetEmitterRate(em, 40)
SetEmitterMax(em, 80)
SetEmitterLife(em, 1.4)
SetEmitterSpeed(em, 3)
SetEmitterSize(em, 0.22, 0.04)
SetEmitterColor(em, 255, 180, 60, 1, 255, 40, 10, 0)
SetEmitterVelocity(em, 0, 1, 0)
SetEmitterCone(em, 50)
SetEmitterGravity(em, 0, -8, 0)

bits = CreateEmitter(cube)
SetEmitterShape(bits, "cube")
SetEmitterSize(bits, 0.12, 0.04)
```

`EmitterParticle e, tex` `EmitterRate` `EmitterMax` `EmitterLife` `EmitterSpeed`  
`EmitterSize e, start [, end]` or `EmitterSize e, w0, h0, w1, h1` — start size shrinks toward the end size over the particle’s life. On a cube, start/end are the edge length.  
`EmitterColor e, r, g, b [, a, r1, g1, b1, a1]` — RGB 0–255 or 0–1 (if all channels ≤ 1); alpha 0–1 or 0–255. The second color is the color at death.  
`EmitterVelocity e, vx, vy, vz` — spray direction, not a one-shot impulse.  
`EmitterGravity e, x, y, z` `EmitterWind` `EmitterCone e, deg` `EmitterDrag`  
`EmitterArea e, x, y, z` — random box around the origin.  
`EmitterDuration e, secs` `EmitterLoop e, on`  
`PositionEmitter e, x, y [, z]`  
`Emit e [, count]` / `EmitterBurst e, count`  
`FreeEmitter e`  

`SetEmitter*` aliases work (`SetEmitterShape`, `SetEmitterRate`, …). Demo: `examples/particles.bb`.

### 2D (Ebiten)

`CreateEmitter2D([parentSprite])`  
`Particle2DRate` `Particle2DMax` `Particle2DLife` `Particle2DSpeed` `Particle2DSize` `Particle2DColor`  
`Particle2DVelocity` `Particle2DGravity` `Particle2DWind` `Particle2DCone` `Particle2DDrag`  
`Particle2DDuration` `Particle2DLoop` `Particle2DBurst` `Particle2DSprite e, img`  
`PositionEmitter2D` `Emit2D e [, count]` `FreeEmitter2D`  

See `examples/particles.bb` and `examples/particles2d.bb`.

## Input

`KeyDown(code)`, `KeyHit(code)` — `1` / `KEY_ESCAPE` is Escape. Also `KEY_W`, `KEY_SPACE`, `KEY_LEFT`, …  
`While Not KeyDown(1)` works after the first Flip (phantom Esc is ignored until then). Still `Flip` every frame. Demos: `While 1` + `If frames > 8 And KeyHit(KEY_ESCAPE) Then End`.  
`MouseX/Y`, `MouseZ()` / `MouseWheel()` / `GetMouseWheel()`, `MouseXSpeed/YSpeed`, `MouseDown(btn)`, `MouseHit(btn)` — 1 left, 2 right, 3 middle  
`MoveMouse x, y`, `HidePointer`, `ShowPointer`, `FlushKeys`, `FlushMouse`  
`MouseLook [cam] [, sens, pitchMin, pitchMax]` — apply mouse delta to camera pitch/yaw (degrees). Pair with `SetCursorMode 2` / `SetRawMouse`  
`CreateFreeCamera()` + `UpdateFreeLook cam [, speed]` — same look plus WASD. Do not bind Escape; use `WindowShouldClose`. `examples/freelook.bb`  
`SetGamepadDeadzone n` / `GamepadDeadzone()` / `GetGamepadDeadzone()` — axis deadzone (default 0.15)  

See `examples/input.bb`.

## Time

`CreateTimer(hz)`, `WaitTimer t`, `FreeTimer`, `TimerTicks`  
`SetTimer name$, ms`, `TimerReady(name$)`  
`Delay ms`, `WaitKey`

## Math

Every angle is **degrees** (same as Blitz). `ATan2(y, x)` is the usual two-argument arctangent, still in degrees. One return value each — no vector types.

### Constants

| Name | Value |
| --- | --- |
| `True` `Yes` | 1 |
| `False` `No` `Null` | 0 (false / empty) |
| `Pi` | 3.14159… |

`x = Null` stores 0. Use `If x = Null Then` to test. `Mod` and `^` are operators.

### Trig

| Command | Meaning |
| --- | --- |
| `Sin(deg)` `Cos(deg)` `Tan(deg)` | Standard trig |
| `ASin(n)` `ACos(n)` `ATan(n)` | Inverse; result in degrees |
| `ATan2(y, x)` | Inverse tan of `y/x`; degrees |

### Roots, powers, rounding

| Command | Meaning |
| --- | --- |
| `Sqr(n)` | Square root |
| `Pow(a, b)` | `a ^ b` as a function |
| `Abs(n)` `Sgn(n)` | Absolute value; −1 / 0 / 1 |
| `Int(n)` | Truncate toward zero |
| `Floor(n)` `Ceil(n)` `Float(n)` | Floor, ceiling, to float |
| `Min(a, b)` `Max(a, b)` | Smaller / larger |
| `Clamp(x, lo, hi)` | Keep `x` in `[lo, hi]` |

### Interpolation

| Command | Meaning |
| --- | --- |
| `Lerp(a, b, t)` | `a + (b-a)*t` |
| `InvLerp(a, b, x)` | How far `x` is from `a` to `b` (0 at `a`) |
| `SmoothStep(t)` | Hermite 0..1. Or `SmoothStep(edge0, edge1, x)` |
| `EaseIn(t)` / `EaseIn(a, b, t)` | Quad ease-in (`t*t`) |
| `EaseOut(t)` / `EaseOut(a, b, t)` | Quad ease-out |
| `Approach(cur, target, step)` | Move toward `target` by at most `step` (use `speed * DeltaTime()`) |

### Random

`Rnd([max])` — 0..1 if omitted, else 0..`max`  
`Rand(lo, hi)` — inclusive integers (`Rand(n)` is 1..n)  
`SeedRnd n` `RndSeed()`

### Angles

| Command | Meaning |
| --- | --- |
| `WrapAngle(deg)` | Wrap to −180..180 |
| `AngleDelta(from, to)` | Shortest signed turn from `from` to `to` |
| `ApproachAngle(cur, target, step)` | Turn toward `target` by at most `step` degrees |
| `DeltaYaw src, dest` | Entity: shortest yaw (degrees) from `src` toward `dest` |
| `DeltaPitch src, dest` | Same for pitch |
| `VectorYaw(x, y, z)` | Yaw of a direction (`ATan2(x, z)`) |
| `VectorPitch(x, y, z)` | Pitch of a direction |
| `MoveWish(yaw)` | WASD wish direction relative to `yaw`. Returns `wishX, wishZ` |
| `Accelerate(vx, vz, wishX, wishZ, acc, fric, maxSpd, grounded, dt)` | Ground friction, accel, and speed cap. Returns `vx, vz` |
| `TurnToward(yaw, vx, vz, rate [, minSpeed])` | Face horizontal velocity, at most `rate` degrees. Stays put below `minSpeed` (default 0.45) |
| `Land(px, py, pz, vy, pads)` | Snap onto pad structs (`x,y,z,w,d`). Returns `py, vy, grounded` |

```basic
wishX, wishZ = MoveWish(camYaw)
vx, vz = Accelerate(vx, vz, wishX, wishZ, 42, 16, 8.4, grounded, dt)
yaw = TurnToward(yaw, vx, vz, 640 * dt)
py, vy, grounded = Land(px, py, pz, vy, pads)
```

`DeltaYaw` / `DeltaPitch` take entity handles. For two raw angles use `AngleDelta`.

### Distances and directions

World +Z is forward. Yaw 0 faces +Z; `x += Sin(yaw)*dist`, `z += Cos(yaw)*dist`.

| Command | Meaning |
| --- | --- |
| `Dist(x1,y1,x2,y2)` / `Distance2D(...)` | 2D distance |
| `Distance3D(x1,y1,z1,x2,y2,z2)` | 3D distance |
| `PointDistance(...)` | 2D if 4 args, 3D if 6 |
| `Length2D(x, y)` `Length3D(x, y, z)` | Vector length |
| `NormX(x,y)` `NormY(x,y)` | Unit 2D components (0 if zero length) |
| `NormX3(x,y,z)` `NormY3` `NormZ3` | Unit 3D components |
| `DirX(yaw)` `DirZ(yaw)` | `Sin` / `Cos` of yaw (+Z forward) |
| `DirY(pitch)` | `Sin(pitch)` |
| `MovePointX(x, yaw, dist)` | `x + Sin(yaw)*dist` |
| `MovePointZ(z, yaw, dist)` | `z + Cos(yaw)*dist` |
| `MovePointY(y, pitch, dist)` | `y + Sin(pitch)*dist` |
| `PointYaw(x1,z1,x2,z2)` | Yaw from A to B. Or 6 args `x1,y1,z1,x2,y2,z2` |
| `PointPitch(x1,y1,z1,x2,y2,z2)` | Pitch from A to B. Or 3 args as a direction |
| `EntityDistance(a, b)` | Distance between two entities |

### Dot, cross, bounce, rotate

| Command | Meaning |
| --- | --- |
| `Dot2D(ax,ay, bx,by)` `Dot3D(ax,ay,az, bx,by,bz)` | Dot product |
| `CrossX/Y/Z(ax,ay,az, bx,by,bz)` | Cross-product components |
| `ReflectX/Y(vx,vy, nx,ny)` | Bounce `v` off unit-ish normal `n` |
| `BounceX` `BounceY` | Same as `ReflectX` / `ReflectY` |
| `RotatedX(x, y, deg)` `RotatedY(x, y, deg)` | Rotate point around origin (CCW, +Y up) |

### World / local and screen

`src` / `dest` 0 = world.

| Command | Meaning |
| --- | --- |
| `TFormPoint x,y,z, src, dest` | Point from `src` space into `dest` space |
| `TFormVector x,y,z, src, dest` | Direction only (no translation) |
| `TFormedX()` `TFormedY()` `TFormedZ()` | Last TForm result |
| `ProjectedX([cam,] x,y,z)` `ProjectedY` `ProjectedZ` | World point to screen (Z is NDC) |
| `UnprojectX([cam,] sx, sy [, depth])` `UnprojectY` `UnprojectZ` | Screen to world (`depth` 0 near-ish) |
| `CameraPick([cam, x, y])` | Screen ray pick; `PickedX/Y/Z` |

See `examples/math.bb`.

## Audio (Ebiten / Oto)

`LoadSound file$` (`.wav` / `.ogg`), `PlaySound`, `LoopSound`, `StopSound`, `FreeSound`  
`SetSoundVolume s, n` `SetSoundPitch s, n` (1 = normal)  
`LoadMusic file$` `PlayMusic [s]` `StopMusic` `SetMusicVolume n`  
`EmitSound s, x, y, z` or `EmitSound s, entity` — distance volume + stereo pan from the listener (default camera). `SetListener e`  
Playback is Oto v3. No OpenAL.

## Physics

**3D (Jolt, or software `fallback`):** `PhysicsBackend$()` / `GetPhysicsBackend$()` is `"jolt"` or `"fallback"` (honest; `-tags nojolt` and unsupported OS/arch are fallback). Windows Jolt is native. Linux/macOS Jolt keeps the command names, with software velocity, joints, and hull rotation. Full walkthrough: [PHYSICS.md](PHYSICS.md). Vehicles: [VEHICLES.md](VEHICLES.md).

Beginner: `Collide(mesh [, STATIC|KINEMATIC|DYNAMIC [, mass]])` boxes the mesh (sphere if the mesh is a sphere). `SetPhysicsMaterial e, "ice"|"rubber"|"wood"|"metal"|"stone"|"plastic"|"bouncy"|"glass"|"default"` — `SetMaterial` with a name does the same; a numeric id is still a PBR material. `Flip` steps physics once if `UpdateWorld` was not called this frame. `RaycastHit` returns `RayHit` (`entity`, `x`, `y`, `z`, `nx`, `ny`, `nz`, `fraction`, `hit`) without writing `Picked*`. `RaycastAll` returns every hit along the ray. After `Raycast`, `GetRayNormalX/Y/Z` and `GetRayFraction`. Constants: `STATIC` `KINEMATIC` `DYNAMIC` `ON_GROUND` `GROUND_STEEP` `GROUND_UNSUPPORTED` `IN_AIR`.

`SetGravity x,y,z` / `GetGravityX/Y/Z()` — stored and applied to Jolt via `SetGravity`.  
`CreateBody` / `CreateBodySphere` / `CreateBodyBox` / `CreateBodyCapsule` / `CreateBodyCylinder` / `CreateBodyConvex` return the entity/body handle. `ActivateBody e` wakes the body.  
`SetBodyVelocity` / `SetVelocity`, `BodyVelocity` / `X/Y/Z`, `GetBodyVelocityX/Y/Z` — **Jolt:** native linear velocity. `BodyVelocity(e)` is the velocity-vector magnitude (speed), while the X/Y/Z forms return components.  
`ApplyImpulse e, x,y,z` / `ApplyForce e, x,y,z` / `ApplyTorque e, x,y,z` / `ApplyForceAtPosition e, fx,fy,fz, px,py,pz` / `ApplyLocalImpulse e, lx,ly,lz` — Jolt native; fallback integrates.  
`SetGravityScale e, n` — 0 = no gravity. `GetGravityScale(e)`. `SetRestitution` / `GetRestitution` / `SetFriction` / `GetFriction` / `SetLinearDamping` / `GetLinearDamping` / `SetAngularDamping` / `GetAngularDamping`. `GetMass(e)` / `GetCCD(e)` / `GetBodyCCD(e)`. `ApplyBuoyancy e [, waterY, scale]` is Jolt `ApplyBuoyancyImpulse` on the `WaterHeight` plane (or a flat `waterY`). `CreateBodyMesh` / `CreateBodyHeightField` / `CreateSensor` / `ShapeCast` / `OverlapSphere`. `OffsetCenterOfMass e, x,y,z`. Vehicles drop COM automatically. After a loop of `CreateBody*`, call `OptimizePhysics` (alias `OptimizeBroadPhase`) so the quad tree is not left deep.  
`SetBodyAngularVelocity` / `GetBodyAngularVelocityX/Y/Z` — **Jolt:** native. **fallback:** stored.  
`SetBodyMass`  
`SetBodyRotation e, pitch,yaw,roll` / `GetBodyPitch/Yaw/Roll` — quaternion synced onto G3N nodes (native Jolt on Windows; software pose on Linux/macOS and fallback).  
`Raycast(x,y,z, dx,dy,dz)` — Jolt `CastRay` (Windows + Linux/macOS Jolt). Fallback (`-tags nojolt`) uses sphere + AABB. Sets `PickedX/Y/Z`. `LinePick` / `RayPick` use the same physics ray, then a visual-sphere pick if physics missed.  
`CreateHingeJoint(a, b, x,y,z, ax,ay,az)` — aliases `CreateHinge` / `CreateHinge3D`. `CreatePointJoint` / `CreateBallSocketJoint`. `CreateSliderJoint`. `CreateSpringJoint` (distance spring; **no** `CreateDistanceJoint` command). `CreateFixedJoint` / `CreateConeJoint` / `CreateSwingTwistJoint`. `CreateJoint kind, a, b, …` (`JOINT_HINGE`=1 … `JOINT_SWINGTWIST`=7). `Grab holder, target [, freq, damp [, x,y,z]]` / `GrabPick` (hit-point grab) / `DropGrab` / `Throw holder, speed`. `a`/`b` are body handles; **`0` is world-fixed**. No args on hinge → 0. `FreeJoint id`  

`CreateRope(a, b [, length, segments, radius])` creates a sagging, colliding physical rope between body centers. `CreateRopeAnchored(a, b, ax,ay,az, bx,by,bz [, length,segments,radius])` uses local body anchors; an anchor belonging to body `0` is a world coordinate. `SetRopeColor`, `SetRopeMass`, `SetRopeDamping`, `SetRopeStrength rope, stiffness, damping, maxForce`, `SetRopeVisible`, `ResetRope`, `FreeRope`; queries: `RopeLength`, `RopeSegments`, `RopeTension` (0–1).  
`CreateCloth(width, height [, nx, ny, pin])` is a Jolt soft-body sheet (ezEngine recipe). Default pin `1` = top edge. `SetClothWind e, n`. Alias `CreateFlag`. Cloth collides one-way with rigid bodies (they push it; it does not shove them). No cloth-on-cloth, no buoyancy.  
`Grab holder, target [, freq, damp]` / `GrabPick` / `DropGrab` / `Throw holder, speed` / `GrabbedEntity(holder)` — 6DOF spring on Windows Jolt; parented kinematic fallback if the joint is unavailable.  
`CreateProjectile e, speed [, gravity, life, radius, bounce, impulse, ignore]` flies along local +Z (ray hits apply impulse). `CreateBeam a, b [, width]` is a stretched cube between two entities. `PlaceAtRay src, dest [, maxDist]` parks `dest` on the first hit (or at max range). `AttachToBone child, mesh, "BoneName"` parents to a named glTF node (or a named child entity).  
`SetWaterFlow vx, vy, vz` feeds fluid velocity into `ApplyBuoyancyImpulse`. Dynamic bodies that enter the water (not vehicles / buoys) auto-float at factor 1.1; `SetBuoyancyFactor e, n` (`<0` disables).  
`SetBodyCCD e, on` / `SetCCD e, on` — Windows: Jolt `LinearCast`. Linux/macOS and fallback: ray sweep from the last pose. `BodySleep e` / `SleepBody e`, `BodyWake e` / `WakeBody e` / `ActivateBody e` — Jolt Activate/Deactivate  
`CreateCharacterController(e [, height, radius, maxSlope, maxStrength])` — Jolt CharacterVirtual. `height` is the **full** height including the rounded caps (default 1.8, radius 0.4). Windows also attaches a slightly smaller inner kinematic body so rays hit the player; Linux/macOS adds a kinematic capsule on the same handle. `MoveCharacter e, vx, vz` keeps the current Y (jump). `MoveCharacter e, vx, vy, vz` sets all three. Do not add gravity yourself: `UpdateWorld` does, and while the character is supported it copies the floor’s vertical speed unless Y is already a jump. `SetCharacterShape e, "capsule"|"box", h, r`. Ground **0** on / **1** steep / **2** unsupported / **3** air (`GetCharacterGroundState`). `GetCharacterContact(e)`. Older `CreateCharacter(e [, halfH, r])` is the kinematic helper. See `docs/PHYSICS.md`.  
Classic: `EntityType`, `GetEntityType`, `EntityRadius`, `EntityBox`, `Collisions`, `CountCollisions`, `EntityCollided(e [, type|other])`, `ResetEntity`, `CollisionEntity`, `CollisionX/Y/Z` — Windows Jolt `ContactListener` queues in **C++** (mutex, no `//export` from Jolt threads); Go drains in `UpdateWorld`. Linux/macOS and fallback synthesize persist contacts from overlap.

See `examples/physics3d.bb`, `examples/jolt_drop.bb`, `examples/physics_joints.bb`, `examples/physics_body.bb`, `examples/physics_contacts.bb`, `examples/physics_pile.bb`, `examples/character_virt.bb`, `examples/cloth.bb`, `examples/grab_beam.bb`. Helpers: [PHYSICS.md](PHYSICS.md), [MODERN_GAME_HELPERS.md](MODERN_GAME_HELPERS.md).

**2D (Chipmunk, not Box2D):** Box2D CGO was not added; Chipmunk already owns `phys2d`. `Physics2D`, `Gravity2D` / `SetGravity2D`, `CreateCircle2D`, `CreateBox2D`, `CreatePoly2D` / `CreatePolygon2D`, `Body2D`, `SetStatic2D`, `SetMass2D`, `SetVelocity2D` / `Velocity2D` / `Velocity2DX` / `Velocity2DY`, `Position2D` / `Position2DY`, `ApplyImpulse2D`, `ApplyForce2D`, `EntityAngle2D`, `Collides2D`, `CountCollisions2D`, `Raycast2D(x1,y1,x2,y2)` (segment; sets `PickedEntity` / `PickedX/Y`), `SetCCD2D on` (more iterations / tighter slop — not Box2D bullet CCD), `UpdateWorld2D`  
`CreatePin2D(a, b [, ax,ay, bx,by])`, `CreateSpring2D(a, b, rest, stiff, damp [, anchors])`, `CreateSlide2D(a, b, min, max [, anchors])`, `CreateJoint2D(kind, a, b, …)` (1 pin, 2 spring, 3 slide), `FreeJoint2D id`

## Network

See [NETWORK.md](NETWORK.md). `NetHost` / `HostNet port`, `NetConnect` / `ConnectNet host$, port`, `NetClose` — session API (UDP default; `-tags enet` uses ENet). Still works.

High-level host (independent handles):

`host = CreateNetworkHost(port [, maxPeers])` — also `CreateHost`  
`client = CreateNetworkClient(ip$, port)` — also `Connect` / `ConnectNetwork` / `ConnectHost(ip$, port)`  
`ConnectHost(host, ip$, port)` — connect an existing host handle to an address  
`netEvent = PollNetwork(host)` — **returns a `netevent` struct**, not a type int. Fields: `Type`, `PeerID` / `Peer`, `PeerIP` / `IP`, `Data` / `Message` / `Msg`. `PollNetwork()` uses the active host.  
`NET_NONE` = 0, `NET_CONNECT` = 1, `NET_DISCONNECT` = 2, `NET_RECEIVE` / `NET_RECV` = 3  
`SendNetworkMessage peer, data$ [, reliable]` — World host; default reliable  
`SendNetwork host, peer, data$ [, reliable]` or `SendNetwork peer, data$ [, reliable]` — also `SendNet host, peer, msg$ [, reliable]`  
`DisconnectNetwork peer` or `DisconnectNetwork host, peer`  
`CloseNetworkHost host` — also `CloseHost` / `CloseNetwork`  
`GetNetworkEventType()` / `GetNetworkPeerID()` / `GetNetworkPeerIP()` / `GetNetworkData$()` — last `PollNetwork` (also `GetNetEventType` / `GetNetPeerIP`)

`examples/net_host.bb`.

`NetSend peer, msg$ [, reliable]`, `NetSendReliable peer, msg$`, `NetBroadcast msg$ [, reliable]`  
`NetRecv()`, `NetUpdate`, `NetMsg$()` / `NetRecvMsg$()` / `GetNetMsg$()`, `NetPeer()` / `GetNetPeer()`, `NetEvent()` / `GetNetEvent()` — 1 connect, 2 disconnect, 3 receive  
`NetPeerCount()` / `GetNetPeerCount()`, `NetConnected()` / `GetNetConnected()`, `NetBackend$()` / `GetNetBackend$()`, `NetDisconnect peer`  
`NetPing [peer]`, `NetRTT([peer])` / `GetNetRTT([peer])`  
`SetNetPlayerName s$`, `NetPlayerName$([peer])` / `GetNetPlayerName$([peer])` — omit peer for local  
`NetLocalID()` / `GetNetLocalID()`, `NetReady [on]`, `NetAllReady()`  
`NetRoom$()` / `SetNetRoom s$` / `GetNetRoom$()`  
`NetSendJSON json$ [, peer]`, `NetRecvJSON$()`  
`NetReplicate e [, on]`, `NetSnapshot$()`, `NetApplySnapshot json$`, `NetTick()` / `GetNetTick()`  
`NetRegister name$`, `NetCall name$ [, arg$] [, peer]` — RPC; local if no net

Default UDP. `go build -tags enet` uses ENet.

## Language builtins

`Print`  
`Len`, `Left`, `Right`, `Mid`, `Chr`, `Asc`, `Str`, `Instr`, `Lower`, `Upper`, `Trim`  
`CreateList()`, `ListAdd list, v`, `ListGet(list, i)`, `ListSet list, i, v`, `ListCount(list)`, `ListRemove list, i`  
`ArraySize(arr [, dim])` — `Dim` length (`dim` 1 = first axis)  
`Data` / `Read` / `Restore` — classic DATA pointer (language statements, not `World.Call`)  
`BackBuffer()` / `FrontBuffer()` / `SetBuffer` — classic no-ops (`SetBuffer BackBuffer()`)

Math is in the **Math** section above. See [LANGUAGE.md](LANGUAGE.md) for `Select` / `Const` / `Struct`.

## Packaging

```
bs build game.bb -o dist
```

Writes `dist/` with the `bs` binary, the `.bb`, copied `Load*` files, `assets/` if present, and natives from `third_party/$GOOS` (on Windows: `libc++.dll` + `libunwind.dll`). Audio is Oto — OpenAL DLLs are not copied. Zip `dist` and run from that folder.

## Menu / level

Keep graphics mode, swap contents:

```basic
ClearWorld
cam = CreateCamera()
LoadScene "level_setup.bb"
```

The scene file is setup only (create entities). Do not `Flip` or `End` inside it. See `examples/scenes.bb`.

## Window / input (G3N GLFW)

Primary window (`Graphics3D`):

`WindowWidth` `WindowHeight` `SetWindowSize w, h` `WindowX` `WindowY` `SetWindowPos x, y`  
`SetWindowTitle s$` `SetFullscreen on` `IconifyWindow` `MaximizeWindow` `RestoreWindow` `FocusWindow`  
`WindowOpacity()` `SetWindowOpacity n` `SetSwapInterval n` `WindowShouldClose()`  
`Clipboard$()` `SetClipboard s$`  
`SetRawMouse on` `RawMouse()` `SetCursorMode n` — 0 normal, 1 hidden, 2 disabled  
`GamepadPresent([i])` `GamepadName$([i])` `GamepadAxis(i, axis)` `GamepadButton(i, btn)` `JoystickPresent([i])`  
`SetGamepadDeadzone n` / `GamepadDeadzone()` / `GetGamepadDeadzone()`  
`MonitorWidth()` `MonitorHeight()`

### Extra windows (shared G3N context)

`CreateWindow(width, height, title$)` returns a handle. The extra GLFW window shares the main OpenGL context. `Flip` draws the 3D scene for that window’s camera (or the main camera) and swaps both windows. Closing an extra window does **not** end the program. ImGui stays on the main window.

| Command | Meaning |
| --- | --- |
| `CreateWindow(w, h, title$)` | Extra GLFW window. Handle `0` is the main window |
| `FreeWindow(win)` / `DeleteWindow(win)` | Destroy an extra window |
| `SetWindowTitle(win, title$)` | Title (one string arg still sets the main title) |
| `SetWindowSize win, w, h` / `SetWindowPos win, x, y` | Size / position (`w, h` or `x, y` alone still target the main window) |
| `GetWindowWidth(win)` `GetWindowHeight(win)` `GetWindowSize(win)` | Size. `GetWindowSize` returns width |
| `GetWindowX(win)` `GetWindowY(win)` `GetWindowPos(win)` | Position. `GetWindowPos` returns X |
| `ShowWindow(win)` `HideWindow(win)` | Visibility |
| `ActivateWindow(win)` | Focus + current render window |
| `SetRenderWindow(win)` / `WindowGraphics(win)` | Current draw-target handle (`CurrentWindow()`) |
| `SetWindowCamera(win, cam)` `GetWindowCamera(win)` | Camera used for that window’s view |
| `WindowClosed(win)` | 1 if the extra window was closed (X). Main uses `WindowShouldClose()` |
| `WindowKeyDown(win, key)` | Key on that window (`KeyDown` still follows the focused window) |

See `examples/windows.bb`.

## Dear ImGui (cimgui-go v1.6.0)

Drawn on the **same** G3N GLFW + OpenGL window after the 3D scene, automatically on `Flip`. Requires `Graphics3D`.

| Command | Meaning |
| --- | --- |
| `GuiBegin(title$)` | Start a panel. Auto-closed on `Flip` if you skip `GuiEnd` |
| `GuiEnd` | Close the current panel |
| `GuiButton(label$)` | 1 if clicked |
| `GuiText s$` | Label |
| `GuiSlider(label$, lo, hi [, v])` | Returns the current value |
| `GuiCheckbox(label$ [, on])` | 1 / 0 |
| `GuiInputText(label$ [, initial$])` | Returns the string |
| `GuiSameLine` / `GuiSeparator` | Layout |
| `WantCaptureMouse()` / `WantCaptureKeyboard()` | 1 when the panel wants that device |
| `GuiDemo` | Built-in ImGui demo window |

See `examples/gui_demo.bb`.

## G3N scene widgets

Immediate ImGui is `Gui*`. These create **G3N gui** nodes on the 3D window (separate names so they do not clash).

```basic
btn = CreateButton("OK")
SetWidgetPos(btn, 16, 16)
SetOnClick(btn, "OnOK")
```

| Command | Meaning | Example |
| --- | --- | --- |
| `CreatePanel(w, h [, parent])` | Empty panel | `p = CreatePanel(200, 120)` |
| `CreateButton(label$ [, parent])` / `G3NButton` | Button | `b = CreateButton("Play")` |
| `CreateLabel(text$ [, parent])` | Label | `l = CreateLabel("Hi")` |
| `CreateSlider(w, h [, parent])` | Horizontal slider | `s = CreateSlider(160, 24)` |
| `CreateCheckbox(label$ [, parent])` | Checkbox | `c = CreateCheckbox("Mute")` |
| `CreateEdit(width [, parent])` | Text field | `e = CreateEdit(160)` |
| `SetOnClick(e, fn$)` | Call `Function fn(id)` on click | `SetOnClick(b, "OnOK")` |
| `SetOnChange(e, fn$)` | Call `Function fn(id, value)` | `SetOnChange(s, "OnSlide")` |
| `SetWidgetText e, s$` / `WidgetText$(e)` / `GetWidgetText$(e)` | Label / button / edit text | `SetWidgetText(l, "Go")` |
| `SetWidgetValue e, n` / `WidgetValue(e)` / `GetWidgetValue(e)` | Slider 0–1 or checkbox 0/1 | `SetWidgetValue(s, 0.5)` |
| `SetWidgetPos e, x, y` | Screen position | `SetWidgetPos(b, 20, 40)` |

## Flecs

See `docs/ECS.md` and `examples/ecs.bb`. Module version **4.1.6**.

`EcsWorld` `EcsEntity` `EcsComponent` `EcsSet` `EcsGet` `EcsGetX/Y/Z/W` `EcsGetS$` `EcsHas` `EcsAdd` `EcsRemove` `EcsDelete` `EcsAlive` `EcsValid` `EcsLookup` `EcsName` `EcsQuery` `EcsQueryCount` `EcsQueryEntity` `EcsProgress` `EcsCount` `EcsParent` `EcsGetParent` `EcsVersion$`

## Pathfinding

See `docs/NAV.md` and `examples/nav.bb`. Detour + grid A* only. **No DetourCrowd C API** (go-detour v0.1.3).

`CreateNavMesh(mesh)` `AddNavObstacle(entity)` `BakeNavMesh([nav])` `CreateAgent(entity)`  
`SetAgentSpeed e, n` `SetAgentRadius e, n` `SetAgentDestination e, x, y, z`  
`GetAgentPathPointX/Y/Z(e, i)` `AgentCountPath(e)` `AgentStop e` `UpdateNav`  
`SetNavMaxSlope deg` / `GetNavMaxSlope()` — skip faces steeper than `deg` when baking. Unset reads as **45** and does not write. `GetNavMaxSlope` never sets.  
`CreateGrid(w, h)` `SetGridWalkable grid, x, y, on` `FindPath(grid, x1, y1, x2, y2)` `PathLength(p)` `PathX(p, i)` `PathY(p, i)`

### Crowd (local separation)

go-detour v0.1.3 has **no DetourCrowd C API**. This is Detour paths plus a local push-apart, time-to-collision sidestep, and yaw toward travel.

```basic
crowd = CreateCrowd(1.2)
CrowdAddAgent(crowd, e)
CrowdSetDestination(crowd, x, y, z)
CrowdUpdate
```

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateCrowd([sep#])` | Crowd handle. `sep` is personal space (default 1.2) | `cr = CreateCrowd(1.3)` |
| `CrowdAddAgent(crowd, e)` / `AddCrowdAgent` | Track entity; creates a nav agent if needed | `CrowdAddAgent(cr, body)` |
| `CrowdSetDestination(crowd, x, y, z)` / `SetCrowdDestination` | Same Detour path for every member | `CrowdSetDestination(cr, 6, 1, 6)` |
| `CrowdUpdate()` | `UpdateNav` + local push-apart (also on Flip) | `CrowdUpdate()` |
| `SetCrowdRadius crowd, n` / `GetCrowdRadius([crowd])` | Separation radius | `SetCrowdRadius(cr, 1.5)` |
| `SetNavMaxSlope(deg)` / `GetNavMaxSlope()` | Skip steep tris on bake. Getter does not write. Unset = 45 | `slope = GetNavMaxSlope()` |
| `BakeTerrainNav([terrain])` | Detour from loaded terrain chunk triangles | `nav = BakeTerrainNav(land)` |

See `docs/NAV.md`, `examples/crowd.bb`, `examples/ecs_crowd.bb`.

## Jobs

Go worker pool. **Workers must never create GL objects.** Queue GPU work; Flip flushes it.

| Command | Meaning | Example |
| --- | --- | --- |
| `JobSubmit([workN])` | Run `workN` hash iters on a worker. Returns id | `id = JobSubmit(10000)` |
| `JobWait(id)` | Block until that job finishes | `JobWait(id)` |
| `JobWaitAll()` | Wait for every submitted job | `JobWaitAll()` |
| `JobCount()` / `GetJobCount()` | Pending + running | `n = JobCount()` |
| `JobQueue()` / `GetJobQueue()` | Queued, not yet running | `q = JobQueue()` |
| `SetJobWorkers(n)` / `GetJobWorkers()` | Used on the next pool create | `SetJobWorkers(4)` |

## Scene streaming

`SetPlayer` turns the bubble on, and the old stream commands still work on their own. Terrain, water, and props follow that entity unless you call `SetStreamOrigin` or `SetStreamFollow` afterward. Collision is the chunks next to the player. Past 4 km the world shifts and `WorldOriginX` / `WorldOriginZ` record it. `SetWorldBubble(0)` leaves `SetPlayer` as footsteps only. See `docs/STREAM.md`.

Grid of prop chunks. Load/unload by camera/player. Mesh create stays on the GL thread.

| Command | Meaning | Example |
| --- | --- | --- |
| `SetPlayer(ent)` | Footsteps, chase target, and the bubble. Origin and follow still override the center | `SetPlayer(player)` |
| `SetWorldBubble(on [, ent])` | `0` before `SetPlayer` keeps the bubble off | `SetWorldBubble(0)` |
| `SetWorldSimRadius(n)` / `GetWorldSimRadius()` | Heightfield chunks around the player. `-1` is draw-only | `SetWorldSimRadius(1)` |
| `SetWorldShift(on [, metres])` | Move the world when the player passes `metres` (default 4096) | `SetWorldShift(1, 4096)` |
| `SetWorldMorph(on)` | Slide terrain verts toward the next LOD | `SetWorldMorph(1)` |
| `WorldOriginX()` / `WorldOriginZ()` | Accumulated shift | `x# = WorldOriginX()` |
| `CreateWorldStream([chunkSize, radius])` | Prop grid, including the demo slabs and cones | `CreateWorldStream(20, 2)` |
| `SetStreamRadius(n)` / `GetStreamRadius()` | Chunks in each direction | `SetStreamRadius(3)` |
| `SetStreamOrigin(x, y, z)` | Pin the center (y ignored). Wins over follow until the next `SetStreamFollow` | `SetStreamOrigin(EntityX(p), 0, EntityZ(p))` |
| `SetStreamFollow(ent)` | Follow that entity each Flip | `SetStreamFollow(player)` |
| `LoadChunk(cx, cz)` | Force-load one cell | `LoadChunk(0, 1)` |
| `UnloadChunk(cx, cz)` | Free that cell’s entities | `UnloadChunk(0, 1)` |
| `ChunkLoaded(cx, cz)` / `GetChunkLoaded` | 1 if resident | `If ChunkLoaded(0, 0) Then` |
| `StreamChunkCount()` / `GetStreamChunkCount()` | Loaded cells | `Print(StreamChunkCount())` |
| `StreamOriginX()` `StreamOriginZ()` | Current origin | `x = GetStreamOriginX()` |

See `examples/stream.bb`.

## Terrain

See `docs/TERRAIN.md`. OpenGL **3.3** grids: central-difference normals, far chunks with a full border, height-blended splat, stochastic ground tiles. Grass cards, trees, and props are scatter on those chunks (`TerrainFoliage`, `TerrainTrees`, `TerrainProps`). They are not collision and not nav. `SetTerrainDetail` changes how soon a chunk goes coarse. `SetTerrainBlendMap` paints sand / grass / rock / snow.

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateTerrain(file$ \| hm, w, d, hscale)` | PNG/JPEG, `""` / `"default"` = FBM, or a `GenerateHeightmap` handle | `t = CreateTerrain(hm, 80, 80, 14)` |
| `CreateTerrainFromHeightmap(hm [, w, d, hscale])` | Same as numeric `CreateTerrain` | `t = CreateTerrainFromHeightmap(hm, 80, 80, 14)` |
| `LoadHeightmap(file$ [, w, d, hscale])` | Decode grayscale → **terrain** handle | `t = LoadHeightmap("h.png", 64, 64, 10)` |
| `GenerateHeightmap(w, h, seed [, octaves, scale, persist, lacun, …])` | CPU FBM / ridged / diamond-square → heightmap handle | `hm = GenerateHeightmap(128, 128, 42, 6, 70, 0.48, 2.15)` |
| `GenerateHeightmapPreset(name$, w, h, seed)` | `alpine` `rolling-hills` `sharp-peaks` `archipelago` `plateaus` `gentle-dunes` | `hm = GenerateHeightmapPreset("alpine", 128, 128, 3)` |
| `SaveHeightmap(path$ [, hm])` | Greyscale PNG | `SaveHeightmap("assets/heightmap.png", hm)` |
| `ImportHeightmap(file$)` / `LoadHeightmapData` | Image → heightmap handle | `hm = ImportHeightmap("h.png")` |
| `ErodeHeightmap(hm [, iters, talus, transfer])` | Thermal erosion | `ErodeHeightmap(hm, 6)` |
| `HydraulicErodeHeightmap(hm [, drops, life, …])` | Droplet rivers | `HydraulicErodeHeightmap(hm, 280, 20)` |
| `FilterHeightmap(hm, type$, strength#)` | `smooth` `terrace` `ridged` `invert` … | `FilterHeightmap(hm, "terrace", 0.7)` |
| `HeightmapTexture([hm])` | Greyscale texture handle | `tex = HeightmapTexture(hm)` |
| `HeightmapWidth([hm])` / `HeightmapHeight` | Size | `Print(HeightmapWidth(hm))` |
| `CreateProcTerrain(seed, chunkSize, octaves [, hscale, worldScale])` | Infinite-ish FBM chunks | `t = CreateProcTerrain(7, 17, 5, 9, 40)` |
| `CreateTerrainGL([seed, chunk, radius, disp, freq, octaves, scale])` | Terrain-OpenGL noise + splat | `t = CreateTerrainGL()` |
| `ApplyTerrainSplat([t])` | Generate CC0 sand/grass/rock/snow + enable `mbterrain` | `ApplyTerrainSplat(t)` |
| `SetTerrainSplat([t,] sand, grass, rock, snow [, grass2, rockN])` | Bind texture handles | `SetTerrainSplat(t, s, g, r, n)` |
| `SetTerrainOctaves` / `SetTerrainFreq` / `SetTerrainDispFactor` / `SetTerrainPower` | Their GUI sliders (rebuilds chunks) | `SetTerrainOctaves(t, 8)` |
| `SetTerrainGrassCoverage` / `SetTerrainRockColor` / `SetTerrainFogFalloff` | Splat + fog | `SetTerrainGrassCoverage(t, 0.65)` |
| `SetTerrainWaterHeight` / `SetTerrainBlend` / `SetTerrainSnow(on [, y])` | Sand line + optional snow | `SetTerrainSnow(True, 8.5)` |
| `SetTerrainTessMultiplier(m)` | CPU vertex density (not GL tess) | `SetTerrainTessMultiplier(t, 1.4)` |
| `SetTerrainDetail(t, pixels)` | Screen-space size as a CPU step. `8` is the old LOD. Smaller stays fine longer. `0` clears it | `SetTerrainDetail(t, 6)` |
| `SetTerrainBlendMap(t, tex)` | R sand, G grass, B rock, A snow. Empty texels keep slope/height | `SetTerrainBlendMap(t, paint)` |
| `TerrainFoliage(t [, density])` | Crossed grass cards. `1` is a light meadow. `0` clears. Far chunks use one quad | `TerrainFoliage(t)` |
| `TerrainTrees(t, mesh [, density])` | Instanced `mesh` on flat ground. Far copies are a quad. `0` clears that mesh | `TerrainTrees(t, tree)` |
| `TerrainProps(t, mesh [, density])` | Rocks and clutter. Several meshes are several layers. `0` removes that mesh | `TerrainProps(t, rock, 0.4)` |
| `TerrainScatter(t, mesh, kind$, density)` | Same planter. `kind$` is `grass` `tree` `prop` `any` | `TerrainScatter(t, bush, "prop", 0.5)` |
| `CreateProcTexture(kind$)` | `sand` `grass` `grass2` `rock` `snow` `rocknormal` `dudv` `cloud` | `tex = CreateProcTexture("grass")` |
| `TerrainSlope(x, z)` / `GetTerrainSlope` | Up-dot normal (1 = flat) | `n# = TerrainSlope(x, z)` |
| `CreateVolumetricClouds([y, scale])` / `CreateClouds` | 3.3 raymarch dome (no compute) | `c = CreateVolumetricClouds(110, 260)` |
| `SetCloudCoverage` / `SetCloudSpeed` / `SetCloudDensity` | Cloud layer | `SetCloudCoverage(0.55)` |
| `SetSkyPreset(name$)` | `default` `sunset` `sunset1` | `SetSkyPreset("sunset")` |
| `SetSkyGradient(tR,tG,tB, bR,bG,bB)` | Rebuild procedural sky | `SetSkyGradient(134, 187, 214, 230, 230, 242)` |
| `TerrainHeight(x, z)` / `GetTerrainHeight` | Bilinear / FBM / Terrain-GL sample | `y# = TerrainHeight(EntityX(p), EntityZ(p))` |
| `SetTerrainTexture(t, tex)` | Diffuse on loaded chunks | `SetTerrainTexture(t, LoadTexture("grass.png"))` |
| `SetTerrainLightmap(t, tex)` | Second texture multiply | `SetTerrainLightmap(t, lm)` |
| `SetTerrainStreamRadius(t, n)` / `GetTerrainStreamRadius([t])` | Chunk window | `SetTerrainStreamRadius(t, 2)` |
| `SetTerrainLOD(t, on)` / `GetTerrainLOD([t])` | Distant chunks skip verts | `SetTerrainLOD(t, 1)` |
| `TerrainChunkCount()` / `GetTerrainChunkCount()` | Resident terrain tiles | `n = TerrainChunkCount()` |
| `BakeTerrainNav([terrain])` | Detour from loaded chunk triangles. Scatter is not included | `nav = BakeTerrainNav(t)` |

## Outdoor and indoor play

How to use every call, in order: [PLAY.md](PLAY.md). Runnable scene: `examples/outdoor.bb` (`.\bs.exe examples\outdoor.bb`). WASD walks the terrain, E uses the crosshair, `[` `]` moves the hour, F5 / F9 save and load.

One play layer on the same world. Outdoor streaming stays the terrain. Indoor streaming is the current room plus the rooms linked through a doorway. Entities in no room stay visible (the player, the terrain, the sun). There is no portal renderer and no second outdoor renderer.

`SetTimeOfDay(hours)` moves every directional light, the sky colors, the fog, and the outdoor ambient together. `0`–`5.5` and `19.5`–`24` are night, `5.5`–`8` and `17`–`19.5` are sunset, the middle of the day is the default sky. The sun pitch follows the hour and never drops below 8°, so night is a dim light rather than a light under the ground. Calling it again with the same band does not rebuild the skybox.

`CreateRoom` returns a room id. Room `0` is outdoors. `SetRoom(ent, room)` puts an entity in a room (`RoomAdd(room, ent)` is the same pair of arguments swapped). `RoomLink(a, b)` shows both rooms from either one. `EnterRoom(id)` hides every other room. `CurrentRoom()` is the id you are in.

`RoomAmbient(room, r, g, b)` replaces the sky ambient while you are in that room and turns directional intensity down, so a lamp can own the space. Leaving the room restores the outdoor ambient and the sun. `RoomAudio(room, footstepSound, reverb)` plays `footstepSound` about once a metre while the player moves. `reverb` is `0`–`1`. There is no convolution reverb; a value above `0` plays the same clip again, quieter, about 0.12s later. `SetPlayer(ent)` is who those steps follow.

`CreateDoor(mesh [, roomA, roomB, sound])` swings that entity 100° on yaw when `Use` hits it, plays `sound` if you passed one, and links `roomA` and `roomB` so the next room is already visible. The swing runs on Flip.

`Use([ent])` uses the entity, or `CameraPick`’s `PickedEntity` when you omit it. A door toggles. An item from `SetItem(ent, name$)` goes into the inventory and the entity hides. `SetDialogue(ent, text$)` starts a conversation; lines are split on `|` or a newline. `DialogueOn()`, `DialogueLine$()`, and `DialogueAdvance()` walk it. An actor from `CreateActor` is set to chase. Anything else returns `0`.

Inventory is a list of names: `InventoryHas(name$)`, `InventoryCount()`, `InventoryItem(index)`, `InventoryRemove(name$)`.

`CreateActor(ent [, aggro, speed])` is three states and nothing else: `0` idle, `1` chase, `2` attack (within 1.5 units). Default aggro is 8 and default speed is 3.2. Chase walks toward `SetPlayer` and samples `TerrainHeight` when a terrain exists. `ActorState(ent)` reads the state. `Use` on the actor starts a chase.

`SaveGame(path$)` / `LoadGame(path$)` write one JSON file: hour, player position, inventory, which doors are open, visited terrain chunks, and the current dialogue line. This is not `SaveScene`. A relative path is under the program directory. The default name is `savegame.json`.

```basic
SetPlayer(player)
SetTimeOfDay(15)
hall = CreateRoom()
kitchen = CreateRoom()
SetRoom(table, hall)
RoomLink(hall, kitchen)
RoomAmbient(hall, 30, 28, 24)
RoomAudio(hall, stepSnd, 0.4)
CreateDoor(doorMesh, hall, kitchen, doorSnd)
SetItem(keyMesh, "key")
SetDialogue(npc, "Hello|The key is on the table")
CreateActor(wolf, 10, 4)
EnterRoom(hall)
```

| Command | Meaning | Example |
| --- | --- | --- |
| `SetTimeOfDay(hours)` / `GetTimeOfDay()` | Sun, sky, fog, outdoor ambient. Hours wrap at 24 | `SetTimeOfDay(18.5)` |
| `CreateRoom()` | New room id. `0` is outdoors | `hall = CreateRoom()` |
| `SetRoom(ent, room)` / `RoomAdd(room, ent)` | Membership. Room `0` clears it | `SetRoom(crate, hall)` |
| `RoomLink(a, b)` | Both rooms visible from either | `RoomLink(hall, kitchen)` |
| `RoomAmbient(room, r, g, b)` | Indoor ambient while you are inside | `RoomAmbient(hall, 30, 28, 24)` |
| `RoomAudio(room, sound, reverb)` | Footsteps plus a quiet slapback | `RoomAudio(hall, step, 0.35)` |
| `EnterRoom(id)` / `CurrentRoom()` | Show this room and its links | `EnterRoom(hall)` |
| `CreateDoor(mesh [, a, b, sound])` | Swing, optional sound, links the two rooms | `CreateDoor(mesh, hall, kitchen, snd)` |
| `Use([ent])` | Door, item, dialogue, or aggro. Blank uses the camera pick | `Use()` |
| `SetItem(ent, name$)` | `Use` pockets it | `SetItem(key, "key")` |
| `InventoryHas(name$)` / `InventoryCount()` / `InventoryItem(i)` / `InventoryRemove(name$)` | Name list | `If InventoryHas("key")` |
| `SetDialogue(ent, text$)` | Lines split on `\|` or newline | `SetDialogue(npc, "Hello\|Bye")` |
| `DialogueOn()` / `DialogueLine$()` / `DialogueAdvance()` | Current line, then the next, then stop | `Print DialogueLine$()` |
| `SetPlayer(ent)` | Chase target, footstep body, and the world bubble. `SetStreamOrigin` / `SetStreamFollow` still override the center | `SetPlayer(player)` |
| `CreateActor(ent [, aggro, speed])` / `ActorState(ent)` | Idle `0`, chase `1`, attack `2` | `CreateActor(wolf, 9, 3.5)` |
| `SaveGame(path$)` / `LoadGame(path$)` | Player, inventory, doors, hour, visited chunks | `SaveGame("slot1.json")` |

## Geo

WGS84 ↔ world XZ (Web Mercator, +X east, +Z north). Thin subset of flywave/go-geo — see `docs/GEO.md`. No PROJ/GEOS.

| Command | Meaning | Example |
| --- | --- | --- |
| `SetGeoOrigin(lon, lat)` | World `(0,0)` = this WGS84 point; stream origin → 0,0 | `SetGeoOrigin(-73.9857, 40.7484)` |
| `GeoOriginLon()` / `GetGeoOriginLon` | Current origin longitude | `lon# = GeoOriginLon()` |
| `GeoOriginLat()` / `GetGeoOriginLat` | Current origin latitude | `lat# = GeoOriginLat()` |
| `SetGeoScale(s)` / `GeoScale()` | World units per metre (default 1) | `SetGeoScale(0.01)` |
| `GeoProject(lon, lat)` / `GeoProjectX` | Easting X; stores Z | `x# = GeoProject(lon, lat)` |
| `GeoProjectZ([lon, lat])` | Northing Z (or last project) | `z# = GeoProjectZ()` |
| `GeoUnproject(x, z)` / `GeoUnprojectLon` | Longitude; stores lat | `lon# = GeoUnproject(x, z)` |
| `GeoUnprojectLat([x, z])` | Latitude (or last unproject) | `lat# = GeoUnprojectLat()` |
| `GeoTileX(lon, lat [, zoom])` | OSM/Google XYZ tile X | `tx = GeoTileX(lon, lat, 15)` |
| `GeoTileY([lon, lat, zoom])` | XYZ tile Y (UL origin) | `ty = GeoTileY()` |
| `GeoTileZ([lon, lat, zoom])` | Zoom used | `tz = GeoTileZ()` |
| `GeoTMSY([y, zoom])` | TMS Y (LL origin) | `ty = GeoTMSY()` |
| `LoadGeoJSON(path$ [, parent])` | Points → cubes, lines/polygons → tubes | `p = LoadGeoJSON("route.geojson")` |
| `GeoJSONCount()` / `GetGeoJSONCount` | Features spawned by last load | `n = GeoJSONCount()` |
| `LoadGeoDEM(file$, west, south, east, north [, hscale])` | PNG heightmap + lon/lat bounds → `CreateTerrain` | `t = LoadGeoDEM("dem.png", w, s, e, n, 12)` |
| `CreateTerrainFromGeoDEM(hm, west, south, east, north [, hscale])` | Same bounds on a `GenerateHeightmap` handle | `t = CreateTerrainFromGeoDEM(hm, w, s, e, n, 8)` |
| `GeoHeight(lon, lat)` / `GetGeoHeight` | `TerrainHeight` at the projected XZ | `y# = GeoHeight(lon, lat)` |

See `examples/geo.bb`.

## Water

See `docs/WATER.md`. Two techniques, one shader: **Gerstner** vertex ocean + **scenic** dual-FBO / DuDv / Fresnel. The surface is opaque; `SetWaterColor` is the deep color. `SetWaterFollow(True)` keeps ~0.5 m cells under the camera. Lite screen-space reflect is **Partial**. GL 3.3.

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateWater(w, d [, segs])` | Water handle + mesh | `w = CreateWater(140, 140, 72)` |
| `SetWaterStyle(name$)` / `SetWaterMode` | `gerstner` `scenic` `ocean` | `SetWaterStyle("gerstner")` |
| `SetWaterColor([id,] r, g, b)` | RGB 0–255 or 0–1 | `SetWaterColor(12, 62, 88)` |
| `SetWaterDuDv(tex)` / `SetWaterDuDvMap` | Replace default DuDv | `SetWaterDuDv(tex)` |
| `SetWaterNormalMap(tex)` | Replace default water normals | `SetWaterNormalMap(tex)` |
| `EnableWaterReflection([on])` / `SetWaterReflection` | Planar camera + FBO (also enables refraction) | `EnableWaterReflection(True)` |
| `EnableWaterRefraction([on])` / `SetWaterRefraction` | Underwater color + depth FBO | `EnableWaterRefraction(True)` |
| `SetWaterSpeed(s)` / `GetWaterSpeed()` | DuDv scroll rate | `SetWaterSpeed(0.04)` |
| `SetWaterWaveStrength(s)` | DuDv distortion amount | `SetWaterWaveStrength(0.05)` |
| `SetWaterWaves(count [, amp])` | 0 = flat plane, 1–4 Gerstner | `SetWaterWaves(4, 0.42)` |
| `SetGerstner(i, dirX, dirZ, steep, amp, lambda, speed)` | One wave (i = 0–3) | `SetGerstner(0, 0.85, 0.35, 0.42, 0.55, 18, 1.15)` |
| `SetWaterWind(dirX, dirZ [, str])` / `GetWaterWind()` | Steer / boost Gerstner | `SetWaterWind(0.9, 0.25, 0.7)` |
| `WaterHeight(x, z)` / `GetWaterHeight` | CPU Gerstner (matches shader + wind) | `y# = WaterHeight(x, z)` |
| `CreateBuoy(ent)` | Jolt lift + splash + wake impulse | `CreateBuoy(boat)` |
| `SetWaterFollow(on)` | Recenter LOD grid on camera XZ | `SetWaterFollow(True)` |
| `SetWaterLevel(y)` / `GetWaterLevel()` | Still-water plane | `SetWaterLevel(0)` |
| `SetUnderwaterFog(r, g, b [, density])` | When camera is under | `SetUnderwaterFog(8, 35, 50, 0.09)` |
| `SetWaterStreamRadius(n)` | Large ocean: follow + far radius | `SetWaterStreamRadius(2)` |
| `SetWaterCaustics([id,] on)` / `GetWaterCaustics` / `EnableWaterCaustics` | Animated caustics on terrain / underwater | `SetWaterCaustics(water, True)` |
| `SetWaterSSR([id,] on)` / `GetWaterSSR` / `EnableWaterSSR` | Lite planar-FBO screen march (**Partial**) | `SetWaterSSR(water, True)` |
| `SetWaterAmbientSound([id,] path$ [, vol])` / `SetWaterAmbient` | Loop Oto clip; missing file is silent | `SetWaterAmbientSound(water, "sea.ogg", GetWeatherIntensity())` |

## Instancing

G3N has no instanced-draw wrapper. On **OpenGL 3.3** we prefer `glDrawElementsInstanced` (`EnableGPUInstances`). If that path is off or fails, we **merge** identical source triangles into one mesh (CPU fallback). See `docs/GRAPHICS.md`.

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateInstancedMesh(src, count)` / `CreateInstanced` | Handle for `count` copies | `trees = CreateInstancedMesh(cone, 80)` |
| `InstanceCount(id [, n])` / `SetInstanceCount` / `GetInstanceCount` | Get/set count | `InstanceCount(trees, 80)` |
| `SetInstanceTransform(id, i, x, y, z [, pitch, yaw, roll] [, sx, sy, sz])` | Instance `i` | `SetInstanceTransform(trees, 0, 4, 1, 2, 0, 30, 0, 1, 2, 1)` |
| `SetInstanceData(id, i, x, y, z [, pitch, yaw, roll] [, sx, sy, sz])` | Same as `SetInstanceTransform` | `SetInstanceData(trees, 1, 8, 1, 0)` |
| `BatchInstances(id)` | Rebuild GPU buffer or merged mesh now | `BatchInstances(trees)` |
| `InstanceEntity(id)` / `GetInstanceEntity` | The batched mesh entity (CPU path) | `e = InstanceEntity(trees)` |
| `EnableGPUInstances([on])` / `SetGPUInstances` | 1 = instanced draw (3.3). 0 = CPU merge | `EnableGPUInstances(True)` |
| `GPUInstances()` | 1 if GPU instancing is active | `If GPUInstances() Then` |

## Modern OpenGL (3.3 required, 4.x optional)

Runs on **OpenGL 3.3 core**. GLFW / G3N never request 4.5 as a minimum. After `Graphics3D`, the driver version is queried. Missing 4.x features **return 0** and print one `glmodern: … skipped` line — they do **not** `End` the program. Full policy: `docs/COMPAT.md` / `docs/GRAPHICS.md`.

### Query

| Command | Meaning | Example |
| --- | --- | --- |
| `GLVersion$()` / `GetGLVersion$()` | Driver version string (empty-ish before `Graphics3D`) | `Print(GLVersion$())` |
| `GLMajor()` `GLMinor()` | Integer version | `If GLMajor() >= 4 Then` |
| `GLRenderer$()` | GPU name | `Print(GLRenderer$())` |
| `GLHasCompute()` | 1 if 4.3 / `GL_ARB_compute_shader` | `If GLHasCompute() Then` |
| `GLHasSSBO()` | 1 if shader storage buffers | `ok = GLHasSSBO()` |
| `GLHasUBO()` | 1 if uniform buffers (3.1+) | `ok = GLHasUBO()` |
| `GLHasInstancing()` | 1 if instanced draw (3.1+) | `ok = GLHasInstancing()` |
| `GLHasGeometry()` | 1 if geometry shaders (3.2+) | `ok = GLHasGeometry()` |
| `GLHasTessellation()` | 1 if tessellation (4.0 / ARB) | `ok = GLHasTessellation()` |
| `GLFeature(name$)` | 1/0 for `compute` `ssbo` `ubo` `instance` `geom` `tess` | `GLFeature("compute")` |

### Compute + SSBO (optional 4.3)

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateComputeShader([src$])` | Compile compute. Empty src = sine-fill demo. **0** if no compute | `cs = CreateComputeShader("")` |
| `DispatchCompute(id [, gx, gy, gz])` | `glDispatchCompute`. **0** if missing / bad id | `DispatchCompute(cs, 4, 1, 1)` |
| `ComputeLog$(id)` / `GetComputeLog$` | Last compile log | `Print(ComputeLog$(cs))` |
| `CreateStorageBuffer(count)` | `count` floats. GPU SSBO if 4.3, else CPU mirror | `ssbo = CreateStorageBuffer(256)` |
| `SetStorageBuffer(id, index, v0 [, v1…])` | Write floats at `index` | `SetStorageBuffer(ssbo, 0, 1.0)` |
| `GetStorageBuffer(id, index)` | Read one float (GPU readback if bound) | `h# = GetStorageBuffer(ssbo, 0)` |
| `BindStorageBuffer(id, binding)` | `glBindBufferBase` SSBO. **0** on 3.3 | `BindStorageBuffer(ssbo, 0)` |
| `StorageBufferSize(id)` | Float count | `n = StorageBufferSize(ssbo)` |

### UBO (OpenGL 3.3 / 3.1)

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateUniformBuffer(count)` | `count` floats, GL uniform buffer | `ubo = CreateUniformBuffer(16)` |
| `SetUniformBuffer(id, index, v0 [, v1…])` | Write floats | `SetUniformBuffer(ubo, 0, 1, 2, 3, 4)` |
| `GetUniformBuffer(id, index)` | Read one float | `v# = GetUniformBuffer(ubo, 0)` |
| `BindUniformBuffer(id, binding)` | Bind to binding point. **0** if UBO missing | `BindUniformBuffer(ubo, 0)` |

### Geometry shaders (3.2 / 3.3)

Point → camera-facing billboard. **0** if the geom program fails to compile.

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateGeomPoints(count)` | Point list handle | `pts = CreateGeomPoints(32)` |
| `SetGeomPoint(id, i, x, y, z [, size, r, g, b])` | Point `i` (RGB 0–255 or 0–1) | `SetGeomPoint(pts, 0, 0, 2, 5, 0.4, 255, 200, 40)` |
| `GeomPointCount(id)` | How many points | `n = GeomPointCount(pts)` |

### Tessellation (optional 4.0)

Water/terrain stay on a **dense regular mesh** on 3.3. This only compiles a tiny tess program to prove the path.

| Command | Meaning | Example |
| --- | --- | --- |
| `EnableTessellation([on])` / `SetTessellation` | Try compile tess shaders. **0** if driver lacks tess | `EnableTessellation(True)` |
| `Tessellation()` | 1 if tess program is live | `If Tessellation() Then` |

### GLSL validate / SPIR-V (no Vulkan)

`CompileShader` always validates in-process (`internal/glslang`). If `Graphics3D` is up, it also compiles with OpenGL. SPIR-V is emitted only when `glslangValidator` is on `PATH` (optional). Native Khronos libs are **not** shipped. `go build -tags glslang` is reserved for a future C binding.

| Command | Meaning | Example |
| --- | --- | --- |
| `CompileShader(src$, stage$)` | Validate (+ GL compile if context). Stage: `vert` `frag` `comp` `geom` `tesc` `tese` | `s = CompileShader(src$, "vert")` |
| `ShaderLog$(id)` / `GetShaderLog$` | Validate / compile log | `Print(ShaderLog$(s))` |
| `ShaderSPIRVSize(id)` | SPIR-V word count (0 if no validator) | `n = ShaderSPIRVSize(s)` |
| `GlslangNative()` | 1 if `glslangValidator` (or `glslang`) is on PATH | `If GlslangNative() Then` |

CLI: `bs shader file.glsl [stage]` — same validate path, no window.

### Gonum helpers (CPU, no GL)

G3N scene types stay `math32`. These use `gonum.org/v1/gonum` via `internal/mathx`.

| Command | Meaning | Example |
| --- | --- | --- |
| `NoiseFBM(x, z [, octaves, persist, lacunarity])` | Terrain-style FBM | `h# = NoiseFBM(x, z, 5)` |
| `SHEval(nx, ny, nz)` | Evaluate probe SH (red channel) | `r# = SHEval(0, 1, 0)` |
| `JobXform([count])` | Worker-pool batch of 4×4 multiplies | `id = JobXform(256)` |

See `examples/glmodern.bb`.

## Light probes (GI)

Drive the hemisphere in `mbshadow` and, on PBR, the cubemap in `mbphysical`. No RTX. The probe nearest the camera captures one 32×32 face per frame. After six frames that cubemap replaces the sky cubemap. `SetProbeColor` is still the sky and ground tint.

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateLightProbe(x, y, z [, r, g, b])` | One probe | `p = CreateLightProbe(0, 4, 0, 80, 110, 160)` |
| `SetProbeColor(id, skyR, skyG, skyB, grR, grG, grB)` | Sky / ground irradiance | `SetProbeColor(p, 90, 120, 170, 40, 30, 22)` |
| `SetProbeGrid(ox, oy, oz, nx, ny, nz, spacing)` | Fill a grid of probes | `SetProbeGrid(-20, 6, -20, 2, 1, 2, 24)` |
| `SampleProbe()` | Re-apply nearest blend (also each Flip) | `SampleProbe()` |
| `SetLightmap(ent, tex)` | Extra texture multiply | `SetLightmap(floor, lm)` |

## Physics extras (Jolt)

See [PHYSICS.md](PHYSICS.md).

| Command | Meaning | Example |
| --- | --- | --- |
| `BodySleep(id)` / `SleepBody` | Deactivate (Jolt) / skip step (fallback) | `BodySleep(ball)` |
| `BodyWake(id)` / `WakeBody` / `ActivateBody` | Activate again | `ActivateBody(ball)` |
| `SetCCD(id, on)` / `SetBodyCCD` / `GetCCD` / `GetBodyCCD` | Windows LinearCast; else ray sweep. Getters are last Set | `If GetCCD(ball) Then` |
| `CreateHingeJoint` / `CreateHinge` / `CreateHinge3D` | Hinge; `0` = world | `h = CreateHingeJoint(wall, door, x,y,z, 0,1,0)` |
| `SetHingeLimits` / `SetHingeMotor` / `SetHingeFriction` | Swing limits / drive / planar damp. Windows native; else software | `SetHingeLimits(h, -10, 95)` |
| `CreatePointJoint` / `CreateBallSocketJoint` | Shared point | `CreatePointJoint(a, b, x,y,z)` |
| `CreateSliderJoint` / `CreateSpringJoint` / `CreateJoint` | Slider; distance spring; kind 1–7 | `CreateJoint(JOINT_HINGE, a, b, x,y,z, 0,1,0)` |
| `FreeJoint id` | Remove constraint | `FreeJoint(h)` |
| `ApplyTorque` / `ApplyForceAtPosition` / `ApplyLocalImpulse` / `SetGravityScale` | Extra forces | `ApplyLocalImpulse(ship, 0, 0, 12)` |
| `Raycast(x,y,z, dx,dy,dz)` | Jolt `CastRay`; fallback sphere+AABB | `e = Raycast(0, 10, 0, 0, -20, 0)` |
| `ShapeCast(hx,hy,hz, x,y,z, dx,dy,dz)` | Sweep a box | `e = ShapeCast(0.4,0.4,0.4, x,y,z, 0,-20,0)` |
| `OverlapSphere` / `OverlapPoint` | Overlap query | `e = OverlapSphere(x,y,z, 1)` |
| `CreateCloth(width, height [, nx, ny, pin])` | Soft-body sheet. Alias `CreateFlag`. Pin 1=top | `flag = CreateCloth(2, 1.4, 10, 8)` |
| `SetClothWind e, n` | Wind along +X on the sheet | `SetClothWind(flag, 0.8)` |
| `CreateBodyCylinder(e, halfH, r [, motion, mass])` | Y-aligned cylinder collider (`halfH` = shaft half-height). Alias `CreateRigidBodyCylinder` | `CreateBodyCylinder(barrel, 0.7, 0.45)` |
| `CreateBodyCompound(e [, motion, mass])` | Fold child colliders into one actor. Alias `CreateCompoundBody` | `CreateBodyCompound(car)` |
| `CreateHitbox(e, hx,hy,hz)` | Query sensor + green debug. Alias `CreateQueryBox` | `CreateHitbox(hurt, 0.4, 0.8, 0.3)` |
| `Explode x,y,z,r,imp` | Sphere overlap + falloff impulse. Alias `AreaDamage` | `Explode(x,y,z, 5, 20)` |
| `FollowPath e, speed [, loop], pts…` | Polyline rail. Alias `FollowPolyline` | `FollowPath(cam, 8, 0,0,0, 10,2,0)` |
| `EnablePhysicsDebug on` | Collider AABB overlay | `EnablePhysicsDebug(1)` |
| `CreateFixedJoint` / `CreateConeJoint` / `CreateSwingTwistJoint` | Weld / cone / ragdoll-ish limits | `CreateFixedJoint(a, b, x,y,z)` |
| `SetCollisionLayer` / `SetLayerCollides` | 32 layers; default all collide | `SetLayerCollides(0, 1, 0)` |
| `CreateBodyConvex(e [, motion, mass])` | Convex hull from mesh tris (≤96 pts). Aliases `CreateConvexHull` / `CreateConvexBody` / `CreateRigidBodyConvex` | `CreateBodyConvex(wedge, 1)` |
| `Grab holder, target [, freq, damp]` | 6DOF spring grab. Alias `GrabEntity` | `Grab(hand, crate)` |
| `GrabPick holder, maxDist [, freq, damp]` | Ray +Z then grab. Default range 8 | `GrabPick(hand, 10)` |
| `DropGrab holder` | Release. Alias `ReleaseGrab` | `DropGrab(hand)` |
| `Throw holder, speed` | Drop + impulse along +Z. Alias `ThrowEntity` | `Throw(hand, 16)` |
| `GrabbedEntity(holder)` | Held body or 0 | `id = GrabbedEntity(hand)` |
| `CreateProjectile e, speed [, gravity, life, radius, bounce, impulse, ignore]` | +Z flyer; ray hit applies impulse | `CreateProjectile(bolt, 25, -9.81, 3)` |
| `CreateBeam a, b [, width]` | Stretched cube between two entities | `CreateBeam(hand, dot, 0.04)` |
| `PlaceAtRay src, dest [, maxDist]` | Park dest on the ray hit. Alias `RayPlace` | `PlaceAtRay(hand, dot, 18)` |
| `AttachToBone child, mesh, bone$` | Parent to named glTF node. Alias `AttachBone` | `AttachToBone(gun, hero, "RightHand")` |
| Aliases | `CreateRigidBodyCylinder` `CreateConvexHull` `CreateConvexBody` `CreateRigidBodyConvex` `GrabEntity` `ReleaseGrab` `ThrowEntity` `AttachBone` `RayPlace` `CreateFlag` | Same handlers as the names above |
| `CreateSensor` / `SetBodySensor` | Trigger volume | `CreateSensor(trig, 2, 1, 2)` |
| `OffsetCenterOfMass e, x,y,z` | Shift COM | `OffsetCenterOfMass(car, 0, -0.2, 0)` |
| `OptimizePhysics()` / `OptimizeBroadPhase` | Rebuild broadphase after many `CreateBody*` | `OptimizePhysics()` |
| `CreateCharacterController(e [, h, r, slope, str])` | CharacterVirtual | `CreateCharacterController(hero, 1.8, 0.4, 50, 100)` |
| `MoveCharacter` / `SetCharacterShape` / `GetCharacterGroundState` | Walk + ground 0–3 | `MoveCharacter(hero, vx, vz)` |
| `CreateCharacter(e [, halfH, r])` | Older kinematic capsule | `CreateCharacter(hero, 0.9, 0.4)` |
| `CreatePin2D` / `CreateSpring2D` / `CreateSlide2D` | Chipmunk joints | `CreatePin2D(floor, crate)` |
| `PhysicsThreads([n])` / `SetPhysicsThreads` / `GetPhysicsThreads()` | Windows: rebuild Jolt job pool (1–32). Also sizes the Go `PhysicsAsync` / `Job*` pool | `PhysicsThreads(4)` |
| `PhysicsAsync([on])` / `SetPhysicsAsync` / `GetPhysicsAsync()` | Step on a job, wait before apply | `PhysicsAsync(True)` |

## Vehicles

See [VEHICLES.md](VEHICLES.md). `CreateXxxController` + `UpdateXxx` each frame.

**Name clash:** `CreatePlane` is the **ground mesh**. The aircraft is `CreatePlaneController` + `UpdatePlane`. There is no `CreatePlane` → controller alias (`CreateCar` / `CreateJet` / `CreateBoat` *are* aliases).

**Native Jolt** (Windows vehicle constraint; else force fallback): `CreateCarController` / `CreateVehicle` + `UpdateCar` / `UpdateVehicle` / `SetVehicleInput(e, steer, throttle, brake)`. `CreateMotorcycleController` + `UpdateMotorcycle`. `CreateTankController` / `CreateTrackedController` + `UpdateTank` / `UpdateTracked`.

**Force / torque:** `CreatePlaneController` + `UpdatePlane(e, th, pitch, roll, yaw)`. `CreateJetController` + `UpdateJet`. `CreateSpaceshipController` + `UpdateSpaceship`. `CreateBoatController` + `UpdateBoat(e, th, steer)`. `CreateHelicopterController` + `UpdateHelicopter(e, collective, cyclicP, cyclicR, yaw)`. `CreateHovercraftController` + `UpdateHovercraft`. `CreateSubmarineController` + `UpdateSubmarine(e, throttle, steer, dive)`. `CreateDroneController` + `UpdateDrone`.

Also: `CreateGliderController` / `CreateSkiController`. Forces: `ApplyTorque` `ApplyForceAtPosition` `ApplyLocalImpulse` `SetGravityScale` `SetRestitution` `SetLinearDamping` `SetFriction` `ApplyBuoyancy` `WaterHeight`.

Demos: `examples/car.bb` `plane.bb` `jet.bb` `spaceship.bb` `boat.bb` `motorcycle.bb` `helicopter.bb` `hovercraft.bb` `submarine.bb` `tank.bb` `drone.bb` `vehicles_more.bb`.

| Command | Physics | Example |
| --- | --- | --- |
| `CreateCarController` / `CreateVehicle` / `UpdateCar e, steer, th, brake` | Jolt wheeled (Windows) or force fallback | `examples/car.bb` |
| `SetVehicleInput e, steer, th, brake` | Same as `UpdateCar` (alias `SetCarInput`) | |
| `CreateMotorcycleController` / `UpdateMotorcycle` | Jolt motorcycle or fallback | `examples/motorcycle.bb` |
| `CreateTankController` / `CreateTrackedController` / `UpdateTank` | Jolt tracked or fallback | `examples/tank.bb` |
| `CreatePlaneController` / `UpdatePlane e, th, pitch, roll, yaw` | Aero. **Not** `CreatePlane` | `examples/plane.bb` |
| `CreateJetController` / `UpdateJet` | Aero, more thrust | `examples/jet.bb` |
| `CreateSpaceshipController` / `UpdateSpaceship` | Zero-g local impulse | `examples/spaceship.bb` |
| `CreateBoatController` / `UpdateBoat e, th, steer` | Gerstner `WaterHeight` | `examples/boat.bb` |
| `CreateHelicopterController` / `UpdateHelicopter e, col, cycP, cycR, yaw` | Hover + cyclic | `examples/helicopter.bb` |
| `CreateHovercraftController` / `UpdateHovercraft` | Up-force + slip | `examples/hovercraft.bb` |
| `CreateSubmarineController` / `UpdateSubmarine e, th, steer, dive` | `WaterHeight` buoyancy | `examples/submarine.bb` |
| `CreateDroneController` / `UpdateDrone` | Small hover | `examples/drone.bb` |
| `CreateGliderController` / `UpdateGlider` | Aero, no thrust | |
| `CreateSkiController` / `UpdateSki` | Low friction | |

## Editor / stats

ImGui panels use the existing `Gui*` commands. These query live numbers:

| Command | Meaning | Example |
| --- | --- | --- |
| `StatsFPS()` / `GetStatsFPS()` | `1 / DeltaTime` | `fps = Int(StatsFPS())` |
| `StatsDraws()` / `GetStatsDraws()` | Visible mesh count | `d = StatsDraws()` |
| `StatsChunks()` / `GetStatsChunks()` | Stream + terrain tiles | `c = StatsChunks()` |
| `StatsJobs()` / `GetStatsJobs()` | Job pool pending | `j = StatsJobs()` |
| `EntityCount()` / `GetEntityCount()` | G3N entity handles | `n = EntityCount()` |
| `EntityByIndex(i)` | 1-based id | `e = EntityByIndex(1)` |
| `ReloadTexture(tex, file$)` | Swap image (GL thread) | `ReloadTexture(tex, "a.png")` |
| `ReloadMesh(ent, file$)` | Load a new mesh, copy pose | `ReloadMesh(e, "tree.glb")` |

See `examples/editor.bb`. Material sliders: `GuiSlider` + `EntityShininess` / `EntityColor` / `EntitySpecular`.

## Post-processing (GL 3.3 fullscreen blit)

Scene renders to a color FBO, then one blit: tonemap (`reinhard`, `neutral`, `aces`, or `none`), exposure, a half-resolution bloom chain, optional FXAA, contrast/sat/tint. The 9-tap extract remains only if the bloom framebuffers cannot be built. **Not** deferred MRT / Unreal post. See `docs/POSTFX.md`.

| Command | Meaning | Example |
| --- | --- | --- |
| `EnablePostFX [on]` | Turn the stack on | `EnablePostFX(True)` |
| `PostFX()` | 1 if enabled | `If PostFX() Then` |
| `SetExposure n` / `GetExposure()` | Linear exposure (0.05–8) | `SetExposure(1.2)` |
| `SetBloom n` / `GetBloom()` | Bright-extract strength (0 off) | `SetBloom(0.35)` |
| `SetFXAA [on]` | Simple luma FXAA | `SetFXAA(True)` |
| `SetColorGrade contrast, sat [, r, g, b]` | Grade + optional RGB tint 0–255 or 0–1 | `SetColorGrade(1.05, 1.1, 255, 240, 230)` |

See `examples/postfx.bb`.

## Shader programs (graph lite)

PBR + Phong stay as they are. This is **GLSL 330 vert+frag bound to a mesh**, not an Unreal node graph. G3N shaman prepends `#version 330 core` — omit `#version` in files, or it is stripped. Use G3N names: `VertexPosition`, `MVP`, `ModelMatrix`, `NormalMatrix`. `CompileShader` still validates a single stage. See `docs/SHADERS.md`.

| Command | Meaning | Example |
| --- | --- | --- |
| `CreateShader([vert$, frag$])` | Empty = pulsing lit default | `sh = CreateShader()` |
| `LoadShader(vertFile$, fragFile$)` | Read `.vert` / `.frag` / `.glsl` | `sh = LoadShader("a.vert", "a.frag")` |
| `SetShader e, sh` / `SetEntityShader` | Bind program to a mesh | `SetShader(ball, sh)` |
| `EntityShader(e)` | Shader handle or 0 | `id = EntityShader(ball)` |
| `SetShaderUniform [sh,] name$, v0 [, v1, v2, v3]` | Per-shader if `sh` given, else global (`applyUserUniforms`) | `SetShaderUniform(sh, "LightDir", 0.2, 0.9, 0.3)` |
| `SetUniform` | Same | |
| `ShaderOK(sh)` | 1 if validate/register succeeded | `If ShaderOK(sh) Then` |

See `examples/shader.bb`.

## Data

`JSONLoad` `JSONSave` `JSONParse` `JSONGet` `JSONSet` `JSON$`  
`SceneSave` / `SaveScene file$` — JSON of tagged primitives / `LoadMesh` paths, lights, cameras (`fov`/`near`/`far`), parent, transform, tint, visibility  
`SceneLoad` / `LoadSceneJSON file$` — rebuild those kinds (`mesh` reloads `src`; `camera` restores frustum; unknown / old dumps → cubes)  
`LoadScene file$` — `.bb` setup script, **or** `.json` / `.yaml` scene dump  
`YAMLLoad` `YAMLSave` `YAMLParse` `YAMLGet` `YAMLSet`  
`PackSave` `PackLoad` `PackEncode$` `PackDecode`

Scene JSON is not a glTF graph (no materials or clips). Lights and cameras round-trip. See `docs/ASSETS.md`.

## Pools

`CreatePool(name$ [, size])` `PoolGet` `PoolPut` `PoolClear` `CreateBank(size)` / `CreateMemBlock(size)` `FreeBank` `BankSize`

