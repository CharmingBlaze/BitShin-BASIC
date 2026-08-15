# Shadows

G3N 0.2 has no shadow, CSM, or shadow-map API. BitShin BASIC draws depth maps with OpenGL **3.3** framebuffers (`go-gl/gl`, `#version 330 core`) and a custom `mbshadow` program registered next to G3N’s `standard` shader. Dummy 1×1 `ShadowMap` when shadows are off. No GL 4.5.

## Pixels that change

These commands write real darkened fragments on lit Standard materials (`CreateCube`, platforms, the player, and so on). `examples/weather.bb`, `examples/platform64.bb`, and `examples/physics_joints.bb` use directional CSM + **EVSM** (`SetShadowQuality(2, 4)`). `examples/shadows.bb` also turns on point/spot maps and the extra techniques.

| Command | What you see |
| --- | --- |
| `EnableShadows [on]` | Shadow maps. Ground and meshes receive shadows from lights that `SetLightShadow`. |
| `ShadowCascades n` | Directional only. `n` is 1–3. Split distances mix log + uniform (λ = 0.65). Cascade orthos are **play-space locked** (camera XZ + flattened look-ahead, texel snap), not a tight pitch-tracking frustum slab. |
| `SetShadowFilter "pcf"\|"pcss"\|"evsm"\|"msm"` | Same four filters as quality 0–3. |
| `SetShadowQuality mode [, pcfTaps]` | `0` grid PCF, `1` PCSS, `2` EVSM (smooth; prefer this), `3` MSM (names `"pcf"` / `"pcss"` / `"evsm"` / `"msm"` also work). Optional PCF radius. |
| `SetShadowPCSS True` / `SetShadowEVSM` / `SetShadowMSM` | Same filters as above. |
| `SetShadowPCF k` | PCF kernel radius (odd 1–9). Grid half-extent is `clamp(k/2, 0, 4)`. |
| `SetShadowBias n [, normalBias]` | Depth bias. Optional `ShadowNormalBias` offsets `WorldPos` along `WorldNormal` by `normalBias * cascadeTexel` before the light-space project. |

## Quality modes (0–3)

| Mode | Name | What you see |
| --- | --- | --- |
| 0 | PCF (grid) | Unrotated `-r..r` kernel (`r = clamp(k/2, 0, 4)`). Clean edges; can look blocky. |
| 1 | PCSS | Blocker search, then the same grid PCF with a wider radius (penumbra). `SetShadowLightSize` scales it. |
| 2 | EVSM | **Use this for smooth shadows.** Exponential variance (Chebyshev) on a blurred moment map. One bilinear tap in lighting. |
| 3 | MSM | Four-moment shadow maps. Same moment-pass + blur idea. |

**Vogel disk was reverted:** a 16-tap Vogel/IGN rotation looks sandy without a TAA pass. The engine does not run TAA, so PCF is the unrotated grid again. Prefer **quality 2 (EVSM)** for soft penumbras.

Ambient light is **not** multiplied by the shadow term — only the Phong (or PBR) light. Raise `SetAmbientLight` / `CreateAmbientLight` if contacts look too black; lower it if you want a harder night.

```basic
EnableShadows True
SetShadowQuality 2, 4          ; EVSM — smooth without TAA
; SetShadowQuality 0, 5         ; grid PCF
; SetShadowQuality 1            ; PCSS
SetShadowBias 0.0025, 1
SetAmbientLight 40, 50, 65
```
| `ShadowMapSize n` / `SetShadowResolution(n)` | Depth tile resolution (256–8192). `SetLightShadowRes(light, n)` stores per-light; CSM sun also updates this. |
| `SetLightShadow light, True` | Directional light aims the CSM. Point and spot lights each get their own map (cubemap faces / perspective). |
| `SetShadowLightSize n` | PCSS light size (penumbra scale). Default `0.04`. |
| `EnableShadowCache` / `SetShadowCache` / `EnableShadowCaching` | Two maps: **static** (terrain, `bodyType` static) redraw when the sun/dir or static key changes; **dynamic** entities every frame. Lighting composites with `min`. `MarkShadowDirty()` / `shadowGeomDirty` still force a static redraw. |
| `EnableShadowAtlas` / `SetShadowAtlas` | Pack directional cascades + point faces + spot maps into a square-ish atlas (not only a horizontal cascade strip). Point/spot maps always use atlas tiles. |
| `EnableContactShadows` / `SetContactShadows` | Short light-space march along the directional map (tight contact darkening). |
| `EnableScreenSpaceShadows` / `SetScreenSpaceShadows` | Longer per-fragment march on the directional map (no separate G-buffer). |

`CreateLight()` (type 1) also works if you `SetLightShadow` it or leave it as the only directional light.

Implementation notes (honest):

- Directional CSM is unchanged: one sun, 1–3 cascades.
- Point lights: six perspective 90° faces in the atlas (major-axis face at sample time). Up to two shadowed point lights.
- Spot lights: one perspective map from the cone. Up to two shadowed spot lights.
- EVSM / MSM write moments in the shadow pass (`RG32F` EVSM or `RGBA16F` MSM; EVSM falls back to `RGBA16F` + a smaller exponent if `RG32F` is missing). A separable Gaussian blur runs after the moment pass. Lighting takes **one bilinear** sample. PCF / PCSS still filter depth in the lighting shader.
- Cascade overlap uses a wide `smoothstep` mix: cas 0 band `max(ShadowSplit.x * 0.35, 6.0)`, cas 1 band `max((ShadowSplit.y - ShadowSplit.x) * 0.35, 8.0)`.
- Contact and screen-space shadows march the directional depth atlas in light UV (faithful without a deferred G-buffer).
- Ambient is not shadowed; only the Phong term is scaled.
- Hidden meshes do not cast.
- Depth pass uses a G3N `*gls.Program` + `gs.UseProgram`. VAOs are primed on the color pass first.
- Shadow cache is a **split**, not a whole-atlas skip. Static tiles hash light VPs + static caster `MatrixWorld()`. Dynamic casters (and GPU instances) redraw every frame. Playing `Animate` clips, 3D emitters, and `MarkShadowDirty()` / `Entity.shadowGeomDirty` still dirty the static map when those casters are static.

## Typical setup

```basic
EnableShadows True
ShadowCascades 2
ShadowMapSize 2048
SetShadowBias 0.0025
SetShadowPCF 3
SetShadowFilter "pcss"
sun = CreateDirectionalLight()
SetLightDirection sun, 55, 40, 0
SetLightColor sun, 255, 235, 200
SetLightShadow sun, True
SetAmbientLight 40, 50, 65
```
