# Shadows

G3N 0.2 has no shadow, CSM, or shadow-map API. BitShin BASIC owns a **dedicated shadow pass** (depth atlas, then the main camera pass) on OpenGL **3.3**. Lighting still uses G3N scene/renderer/gls. Dummy 1×1 `ShadowMap` when shadows are off.

## Default path

Shadows are **on by default** (World construction and `Graphics3D`), like Blitz3D. Meshes cast and receive by default, and the first directional light automatically becomes the shadow-casting sun. Use `EnableShadows False`, `EntityCastShadow entity, False`, or `EntityReceiveShadow entity, False` to opt out. The first two valid depth frames stay fully lit so a follow-cam ease cannot stamp a cascade-sized umbra.

**PCF on `DEPTH_COMPONENT`**, 4-cascade directional CSM fitted to the **camera frustum** (not a play-plane disc). Default quality is **High**: 4 × 2048, 3×3 PCF (Poisson 16-tap when `SetShadowPCF` ≥ 4). EVSM/MSM remain optional filters; they must not be the Platform 64 default (they can self-shadow a whole plane).

`examples/platform64.bb` uses `SetShadowQuality "high"`. `examples/claw.bb` uses PCF (`SetShadowQuality 0, 4`). `SetShadowEVSM` / `SetShadowFilter "evsm"` still work.

| Command | What you see |
| --- | --- |
| `EnableShadows [on]` | Depth pass + `mbshadow` / `mbphysical` / `mbterrain` receive. Shadow multiplies **direct sun (and local lights)** only — never ambient, IBL, or emissive. |
| `ShadowCascades n` | Directional CSM, 1–4. Practical splits (λ = 0.70 log+uniform) from camera near to shadow distance (~250). |
| `SetShadowDistance n` | CSM far plane (about 200–300). Distance fade uses the last 30 units. |
| `SetShadowQuality mode [, pcfTaps]` | Filter `0` PCF, `1` PCSS, `2` EVSM, `3` MSM (names `"pcf"` / `"pcss"` / `"evsm"` / `"msm"`). **Or** presets `"low"` 2×1024, `"medium"` 3×1536, `"high"` 4×2048, `"ultra"` 4×4096 (PCF). |
| `SetShadowFilter "pcf"\|"pcss"\|"evsm"\|"msm"` | Same four filters as quality 0–3. |
| `SetShadowPCSS True` / `SetShadowEVSM` / `SetShadowMSM` | Same filters as above. |
| `SetShadowPCF k` | PCF taps (minimum 3). `k<=3` is 3×3; higher is Poisson. |
| `SetShadowBias n [, normalBias]` | Receiver bias plus slope-aware GLSL bias. Depth pass also uses polygon offset. Optional `ShadowNormalBias` in texels. |
| `EntityCastShadow id, on` / `CastShadow` | Per-mesh cast. Sky, water, and particles never cast. Default on for meshes. |
| `EntityReceiveShadow id, on` / `ReceiveShadow` | Per-mesh receive. Default on. |

Out-of-cascade / failed samples are **lit (1.0)**. Cascades blend across the last **10%** of each split. Texel snap stabilizes shimmer. Light-space Z is padded 30–100 units so tall casters are not clipped.

## Filter modes (0–3)

| Mode | Name | What you see |
| --- | --- | --- |
| 0 | PCF | **Default.** 3×3 or Poisson on the depth atlas. |
| 1 | PCSS | Blocker search, then PCF with a wider radius. `SetShadowLightSize` scales it. |
| 2 | EVSM | Exponential variance on a blurred moment map. Optional; can crush large receivers if moments are wrong. |
| 3 | MSM | Four-moment maps. Same moment-pass + blur idea. |

Ambient / hemisphere / IBL are **not** multiplied by the shadow term. Raise `SetAmbientLight` if contacts look too black.

```basic
EnableShadows True
SetShadowQuality "high"
SetShadowBias 0.0018, 1
SetAmbientLight 40, 50, 65
```

| `ShadowMapSize n` / `SetShadowResolution(n)` | Depth tile resolution (256–8192). `SetLightShadowRes(light, n)` stores per-light; CSM sun also updates this. |
| `SetLightShadow light, True` | Directional light aims the CSM (same vector as G3N position / sky sun). Point and spot lights each get their own map (cubemap faces / perspective). |
| `SetShadowLightSize n` | PCSS light size (penumbra scale). Default `0.04`. |
| `EnableShadowCache` / `SetShadowCache` / `EnableShadowCaching` | Two maps: **static** vs **dynamic**. |
| `EnableShadowAtlas` / `SetShadowAtlas` | Pack directional cascades + point faces + spot maps into a square-ish atlas. |
| `EnableContactShadows` / `SetContactShadows` | Stub (returns lit). |
| `EnableScreenSpaceShadows` / `SetScreenSpaceShadows` | Stub (returns lit). |

## Implementation notes

- Directional CSM: one sun, up to **4** cascades fitted to each camera-frustum slice. Atlas tiles (not a `TEXTURE_2D_ARRAY` yet).
- Point lights: six perspective 90° faces in the atlas. Up to two shadowed point lights.
- Spot lights: one perspective map from the cone. Up to two shadowed spot lights.
- Depth pass is empty-fragment for PCF/PCSS; moment shaders still run for EVSM/MSM. Alpha-test depth shader for textured cutouts (`EntityTexture`).
- Hidden meshes, sky, particles, and `EntityCastShadow False` do not cast.
- Depth pass uses a G3N `*gls.Program` + `gs.UseProgram`. Light basis uses `CrossVectors` (G3N `Cross` mutates the receiver).
