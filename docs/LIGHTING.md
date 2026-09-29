# Lighting

Phong is the default on every primitive. PBR is opt-in (`docs/PBR.md`). Shadows are opt-in (`docs/SHADOWS.md`). The post stack is opt-in (`docs/POSTFX.md`).

Colors are **0–255**, or **0–1 when every channel is ≤ 1**. Angles are **degrees**. Light handles are integers.

Runnable scene: `examples/lights.bb`.

```powershell
.\bs.exe examples\lights.bb
```

Space switches that scene between outdoor daylight and an indoor room.

## What a pixel is

A lit pixel is:

1. **Ambient** — one scene color (`AmbientLight`) plus an optional extra tint on that light (`SetLightAmbient`).
2. **Diffuse** — light color × material color × how face-on the surface is. A directional light has no distance falloff. A point or spot light fades with `LightRange`.
3. **Specular** — a Blinn-Phong highlight. Size is `EntityShininess`. Color is the material specular times the light specular.
4. **Emission** — added on top. It is not a light and it does not cast.
5. **Shadow** — multiplies the direct term only. Ambient, hemisphere fill, and emission stay lit.

`EntityColor` sets the diffuse color and also the material ambient. If you want a darker ambient than the paint, call `EntityAmbient` **after** `EntityColor`.

## 1. One sun and one cube

```basic
Graphics3D 640, 480
cam = CreateCamera()
PositionEntity cam, 0, 2, -6

sun = CreateLight(1)
SetLightDirection sun, 50, 30, 0
SetLightColor sun, 255, 230, 180

cube = CreateCube()
PositionEntity cube, 0, 0, 5
EntityColor cube, 230, 226, 218
EntityAmbient cube, 36, 34, 32
EntitySpecular cube, 255, 255, 255
EntityShininess cube, 48

AmbientLight 40, 44, 52

While Not KeyDown(1)
    RenderWorld
    Flip
Wend
```

| Call | What you get |
| --- | --- |
| `CreateLight(1)` / `CreateDirectionalLight()` | Sun. No distance falloff. The first one casts shadows once `EnableShadows` is on. |
| `CreateLight(2)` / `CreatePointLight()` | Bulb. Bright at the lamp, gone at `LightRange`. |
| `CreateLight(3)` / `CreateSpotLight()` | Cone. Same falloff as a point, plus an inner and outer angle. |
| `CreateAmbientLight()` | Another ambient node. Prefer `AmbientLight r, g, b` for the scene fill. |

`SetLightDirection light, pitch, yaw` aims a sun or a spot. Positive pitch is above the horizon. `SetLightIntensity light, n` is a multiplier on the color. `SetLightColor` / `LightColor` is the RGB.

`EntityShininess mesh, n` — if `n` is **1 or less**, it is a 0–1 weight times 128 (`0.5` → 64). If `n` is **greater than 1**, it is the Phong exponent (`32` soft, `64` tight, `128` a small dot).

## 2. A lamp that stops at a distance

Point and spot lights start on a **smooth** window: full brightness at the lamp, zero at the radius. That is what old scenes expect. Shadow maps use the same distance.

```basic
lamp = CreatePointLight()
PositionEntity lamp, 2, 2, 4
SetLightColor lamp, 80, 160, 255
LightRange lamp, 12
```

Three falloff modes, pick one per lamp:

| Mode | Call | Shape |
| --- | --- | --- |
| `smooth` | `SetLightFalloff lamp, "smooth"` | Default window. Artist control. |
| `classic` | `SetLightAttenuation lamp, 1, 0.14, 0.07` | LearnOpenGL `1 / (constant + linear·d + quadratic·d²)`. A useful 32-unit lamp is `1, 0.14, 0.07`. `SetLightFalloff lamp, "classic"` uses `1, 0.09, 0.032` if you do not pass your own terms. |
| `physical` | `SetLightFalloff lamp, "physical"` | glTF inverse-square, forced to zero at `LightRange`. Treat `SetLightIntensity` as candela: about **80–400** for a room lamp. A directional light in this mode is lux, and still has no distance falloff. |

`SetLightAttenuation light, constant, linear, quadratic [, range]` writes the classic polynomial. Passing a range also sets how far the shadow map looks.

A spot’s cone:

```basic
spot = CreateSpotLight()
PositionEntity spot, -2, 3, 2
SetLightDirection spot, -50, 24, 0
SetLightCone spot, 16, 34
LightRange spot, 12
```

`16` is the bright inner angle, `34` is the outer edge, both in degrees. In `physical` mode the edge is the glTF squared curve. In `smooth` and `classic` it is a smoothstep between those two angles.

## 3. Materials

```basic
EntityColor box, 40, 90, 190
EntitySpecular box, 180, 210, 255
EntityShininess box, 64
EntityEmission sign, 40, 20, 4
```

| Command | Role |
| --- | --- |
| `EntityColor` | Diffuse paint. Also writes material ambient, so follow it with `EntityAmbient` when those should differ. |
| `EntityAmbient` | How much the scene ambient tints this mesh. |
| `EntitySpecular` | Highlight color on the mesh. |
| `EntityShininess` | Highlight size. See the 0–1 vs exponent rule above. |
| `EntityEmission` / `EntityEmissive` | Color added after lighting. A glow, not a lamp. |
| `EntityTexture` / `SetDiffuseMap` | Diffuse map. |
| `SetSpecularMap mesh, tex` | Multiplies the specular color. |
| `SetEmissionMap mesh, tex` | If emission color is black, the map is the glow. Otherwise it multiplies that color. |
| `SetEntityNormalMap mesh, tex` | Phong normal map. The tangent frame comes from screen derivatives, so the mesh does not need tangents. The green channel is not flipped. |

`SetNormalMap` is the **PBR** command. It turns the mesh onto the metallic-roughness shader. Use `SetEntityNormalMap` when the mesh should stay Phong.

Maps are sampled on their own. Do not `AddTexture` a specular, emission, or normal map onto the diffuse slot.

Per-light color, on top of the material:

```basic
SetLightSpecular sun, 255, 244, 220
SetLightAmbient spot, 24, 12, 4
```

Specular defaults to the light’s own color. Extra ambient defaults to none; `AmbientLight` is still there. These extra colors apply to the first **8** lights of each kind (directional, point, spot). Lights past that still shine, using the light color and no extra ambient.

## 4. Several lights

Every light in the scene is shaded. You do not pick a “main” light except for shadows: the first directional owns the sun shadow map. Point and spot shadows are off until you ask.

```basic
EnableShadows True
ShadowCascades 2
SetShadowFilter "pcf"

SetLightShadow sun, True
SetLightShadow lamp, True
```

`SetLightEnabled light, 0` turns that lamp off and remembers its intensity. `SetLightEnabled light, 1` puts the same intensity back. Hiding the node is not enough on its own; this command is the one that drops the brightness to zero.

Read a light back:

| Command | Returns |
| --- | --- |
| `GetLightType(light)` | `0` ambient, `1` directional, `2` point, `3` spot |
| `GetLightIntensity(light)` | Current multiplier (`0` while disabled) |
| `GetLightRange(light)` | `LightRange`, or the distance implied by the linear term |
| `GetLightRed` / `GetLightGreen` / `GetLightBlue` | Color channels, 0–255, without the intensity multiplier |

## 5. Outdoor and indoor presets

Create the sun and the practical lamps **first**, then apply a preset. The outdoor preset retints a sun you already made. It only creates a sun when the scene has none.

```basic
sun = CreateDirectionalLight()
lamp = CreatePointLight()
LightRange lamp, 8
SetLightColor lamp, 255, 214, 150

OutdoorLighting()
```

`SetLighting "outdoor"` and `OutdoorLighting` are the same call. `SetLighting "indoor"` and `IndoorLighting` are the other. `GetLighting()` is `1` outdoors, `2` indoors, `0` if you have not picked one.

**Outdoor**

- Sun color is a warm white. Intensity becomes `1.45` when the current value is under `4` (a candela-sized number is left alone).
- Ambient is a dim blue-grey. A sky / ground hemisphere fills the shadows (sky blue, earth brown).
- Linear fog from `48` to `220`, clear color a daylight blue.
- Shadow distance `240`.
- Exposure `1.05` and the Khronos neutral tonemap, used once you `EnablePostFX`.

**Indoor**

- Each directional light drops to intensity `0.28` and a warm window color.
- Point and spot lights switch to `physical` falloff. If a lamp’s intensity is under `20`, a point becomes `110` and a spot becomes `160` (candela). Going back outdoors restores the intensity and the falloff you had.
- Ambient is a dark room. Hemisphere is a dim ceiling and floor. Fog turns off. Clear color is nearly black.
- Shadow distance `40`.
- Exposure `1.45` and the same neutral tonemap.

`examples/lights.bb` toggles these with Space. The classic-attenuation bulb in that scene is restored when you return outdoors; the preset does not force every lamp back to the smooth window.

These presets do not turn post FX on. Call `EnablePostFX True` when you want the exposure and the tonemap.

## 6. Time of day

`SetTimeOfDay hours` is the outdoor clock. Hours wrap at 24. `GetTimeOfDay()` reads the hour back.

```basic
sun = CreateDirectionalLight()
SetTimeOfDay(15)

hour# = GetTimeOfDay()
If KeyDown(KEY_LBRACKET) Then hour = hour - 4 * DeltaTime()
If KeyDown(KEY_RBRACKET) Then hour = hour + 4 * DeltaTime()
SetTimeOfDay(hour)
```

| Hours | Look |
| --- | --- |
| `0`–`5.5` and `19.5`–`24` | Night. Dim sun, dark sky, dark fog. |
| `5.5`–`8` and `17`–`19.5` | Sunset. |
| `8`–`17` | Day. |

Every directional light gets a new pitch and yaw. Pitch follows the hour and never drops below `8°`, so night is a low dim light. The skybox, fog color, and outdoor ambient change with the band. Calling it again inside the same band does not rebuild the skybox.

`SetTimeOfDay` and `OutdoorLighting` both write the sun and the ambient. Use the clock when the hour should move. Use the preset when you want one daylight look or one interior look. Rooms (`RoomAmbient`, `EnterRoom`) dim that sun while you are indoors and restore it when you leave; that path is `docs/PLAY.md`.

## 7. Color temperature and a studio rig

`SetLightTemperature light, kelvin` writes a black-body RGB and leaves intensity alone. Values clamp to `1000`–`40000`.

| Kelvin | Typical source |
| --- | --- |
| `2000` | Candle, very warm |
| `3200` | Tungsten practical |
| `5200`–`5600` | Noon sun, key light |
| `6500` | Daylight white |
| `7500` | Cool fill |
| `11000` | Open shade, moonlight |

```basic
SetLightTemperature lamp, 2700
SetLightIntensity lamp, 140
```

`CreateThreePoint()` builds three directional lights and returns the **key**:

| Light | Aim (pitch, yaw) | Intensity | Kelvin | Shadow |
| --- | --- | --- | --- | --- |
| key (returned) | `48`, `35` | `1.35` | `5200` | On, if no sun owns the shadow map yet |
| fill | `20`, `-40` | `0.38` | `7500` | Off |
| rim | `28`, `160` | `0.72` | `6500` | Off |

The nodes are named `key`, `fill`, and `rim`.

## 8. A spot cookie

A cookie (gobo) is a texture projected by a spot. The **red** channel is the mask. Only the first **two** spots in the scene get one.

```basic
gobo = LoadTexture("assets/gobo.png")
SetLightCookie spot, gobo
```

The mask is only drawn inside the outer cone. Spots after the second ignore `SetLightCookie`.

## 9. Showing the picture

The shaders write linear-ish color. Two ways to put that on the screen:

**Post FX on** — this is the one to use with PBR and with the outdoor / indoor presets.

```basic
EnablePostFX True
SetExposure 1.1
SetTonemap "neutral"
```

| `SetTonemap` | Curve |
| --- | --- |
| `"reinhard"` | Default. `c / (c + 1)`. |
| `"neutral"` | Khronos PBR Neutral. Highlights compress and desaturate a little. |
| `"aces"` | Fitted ACES film curve. |
| `"none"` / `"clamp"` | Exposure only, then a hard clamp. |

`SetExposure` is a multiplier before that curve. Bloom, FXAA, and grade are in `docs/POSTFX.md`.

**Post FX off** — Phong only:

```basic
SetGammaCorrection 1
```

That applies a 2.2 gamma encode after the in-shader highlight compress. Leave it off when `EnablePostFX` is on, and leave it off on a PBR scene (that shader already encodes).

## 10. Where the other lighting lives

| Topic | Doc |
| --- | --- |
| Metallic-roughness, `SetNormalMap`, IBL | `docs/PBR.md` |
| Cascades, PCF / PCSS, per-light shadow maps | `docs/SHADOWS.md` |
| Exposure, bloom, FXAA | `docs/POSTFX.md` |
| Sky, fog bands, rooms, `RoomAmbient` | `docs/PLAY.md` |
| Hemisphere probes and lightmaps | `docs/COMMANDS.md` (Light probes) |

Probe grids replace the outdoor / indoor hemisphere while any probe exists. With no probes, `OutdoorLighting` and `IndoorLighting` keep their sky and ground colors.
