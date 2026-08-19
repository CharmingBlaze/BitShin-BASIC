# PBR materials

Opt-in metallic-roughness on G3N OpenGL **3.3** (`#version 330 core`). Default primitives stay Phong (`material.Standard` + `mbshadow`) so Platform 64 is unchanged.

See `docs/COMMANDS.md` for the full command list. Demo: `examples/pbr.bb`.

## Model

`mbphysical` is Cook-Torrance GGX + Fresnel-Schlick + Lambert diffuse (energy-conserving: `(1-F) * albedo / π`).

| Input | Role |
| --- | --- |
| Albedo / base color | Dielectric diffuse, or metal F0 when metallic = 1 |
| Metallic | 0 dielectric, 1 metal |
| Roughness | Perceptual roughness (squared to α for GGX) |
| AO | Multiplies ambient / IBL / hemi only |
| Emissive | Added after lighting (not shadowed) |
| Normal map | Tangent-space; derived TBN if the mesh has no tangents |
| Metallic-roughness map | glTF layout: **G** roughness, **B** metallic |
| AO map | R channel, multiplied by `SetAO` |
| Emissive map | RGB * `SetEmissive` |

Shadows (play-space CSM + point/spot atlas, same uniforms as `mbshadow`) multiply **direct** light only. Ambient, hemi, IBL, and emissive stay lit. Dummy 1×1 `ShadowMap` when shadows are off.

Fog (linear / exp / exp²) is the same uniforms as `mbshadow`.

## vs old shininess

| Phong (`EntityShininess` / `EntitySpecular`) | PBR |
| --- | --- |
| Specular blob size + tint | Roughness + metallic |
| Energy is not conserved | Diffuse drops as metal / Fresnel rises |
| `mbshadow` / `standard` | `mbphysical` |
| Default for `CreateCube` etc. | `CreatePBRMaterial` / `SetMaterialPBR e, 1` |

`EntityColor` / `EntityAlpha` / `EntityTexture` still work on a PBR entity (albedo / opacity / base-color map). RGB is 0–255 or 0–1 (if all channels ≤ 1). Shininess/specular are ignored once PBR is on.

## IBL

Analytic hemisphere (sky / ground) plus **UE4 EnvBRDF split-sum** (`scale * F + bias`) so rough metals lose the old `gloss²` sparkle. Optional `SetEnvMap tex` samples a **2D** lat-long with `textureLod` (GLSL 330). No prefiltered cubemap / GPU DFG LUT texture.

- **Default:** `SetIBL True` is on for PBR.
- Skybox faces are not a cubemap sampler. A visible skybox only tints the analytic sky color.

`SetIBL 0` / `SetIBL e, 0` turns this off. Direct sun + hemi + GGX still run.

## glTF / GLB

`LoadMesh` / `LoadAnimMesh` wrap G3N `material.Physical` in `pbrMat`, set shader `mbphysical`, and fill `Entity.pbr`. Metallic-roughness maps from the file stay attached. Script commands then apply to every PBR material on that entity.

OBJ / DAE stay Phong unless you `SetMaterialPBR`.

## Commands

All of these are registered (plus Get/Set aliases). Parentheses in examples.

`CreatePBRMaterial()`  
`SetMaterialPBR e, 1` / `EnablePBR e, 1` / `GetMaterialPBR(e)`  
`SetMaterial e, mat` / `SetPBRMaterial e, mat`  
`SetAlbedo e, r, g, b` / `SetBaseColor e, r, g, b` / `SetAlbedo e, tex`  
`SetAlbedoMap e, tex` / `SetBaseColorMap e, tex`  
`GetAlbedoR(e)` `GetAlbedoG(e)` `GetAlbedoB(e)`  
`GetBaseColorR(e)` `GetBaseColorG(e)` `GetBaseColorB(e)`  
`SetMetallic e, n` / `SetMetallicFactor e, n` / `GetMetallic(e)` / `GetMetallicFactor(e)`  
`SetRoughness e, n` / `SetRoughnessFactor e, n` / `GetRoughness(e)` / `GetRoughnessFactor(e)`  
`SetAO e, n` / `SetOcclusion e, n` / `SetOcclusionFactor e, n`  
`GetAO(e)` / `GetOcclusion(e)` / `GetOcclusionFactor(e)`  
`SetEmissive e, r, g, b` / `GetEmissiveR(e)` `GetEmissiveG(e)` `GetEmissiveB(e)`  
`SetNormalMap e, tex`  
`SetMetallicRoughnessMap e, tex` / `SetMetalRoughMap e, tex`  
`SetEmissiveMap e, tex`  
`SetAOMap e, tex` / `SetOcclusionMap e, tex`  
`SetEnvMap e, tex` / `SetEnvMap tex`  
`SetIBL True` / `SetIBL e, 1` / `EnableIBL True` / `GetIBL()` / `GetIBL(e)`  
`SetIBLIntensity n` / `GetIBLIntensity()`

## Shader / GL

- Program name: **`mbphysical`**
- Profile: G3N prepends `#version 330 core`. No `#version 450`, no compute, no tessellation, no SSBO.
- Depth pass is unchanged: G3N `*gls.Program` + `gs.UseProgram`.
