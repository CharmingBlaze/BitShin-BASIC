# Status

What is wired and what is not. Classic command names still work; 2026 `Set*` / `Get*` aliases are registered on top.

**OpenGL baseline: 3.3 core.** G3N shaders are `#version 330 core` (G3N shaman prepends this). Extra GLFW windows hint 3.3. Never require 4.5. Optional 4.x is feature-detected (`docs/COMPAT.md`, `docs/GRAPHICS.md`).

## Pillars

| Pillar | Status | Notes |
| --- | --- | --- |
| Language | Real | If / While / For / Repeat / Select, Const, Dim, Function, Include, Data/Read, Yes/No/Null, degree math, `Type`/`Struct`, `Method`, `Import [As]`, `Namespace` |
| G3N 3D | Real | Window, mesh primitives, lights, cameras, textures, glTF/OBJ/DAE, animation, pick, fog, cubemap skyboxes, weather, 3D particles. Shadows: CSM + PCF/PCSS/EVSM/MSM (`docs/SHADOWS.md`). **GL 3.3** |
| Weather | Real (sky Partial) | Rain/snow/fog/storm particles, wetness, height fog, wind sway, lightning bolts. Atmosphere = Bruneton-style LUT + single scatter (`docs/WEATHER.md`). Not full 4D precompute / 3D clouds |
| Terrain | Real | Heightmap PNG/JPEG, FBM proc, **CPU generator** (`GenerateHeightmap` / erosion / PNG), **Terrain-OpenGL** value-noise + `mbterrain` splat (`CreateTerrainGL`), chunk stream + CPU LOD (tess/compute not required), `TerrainHeight` / `TerrainSlope`, 3.3 cloud raymarch. `docs/TERRAIN.md` |
| Geo | Real | Thin Web Mercator subset (`internal/geo`): `SetGeoOrigin`, `GeoProject` / `GeoUnproject`, XYZ/TMS tiles, `LoadGeoJSON`, DEM bounds → existing `CreateTerrain`. **Not** full flywave/go-geo (PROJ/GEOS/CGO). `docs/GEO.md` |
| Water | Real | Gerstner (vertex + wind + `WaterHeight`) and scenic dual-FBO / DuDv / Fresnel / depth tint / texture foam / dual normals; combo `ocean`. Camera-centered LOD grid. Caustics, wakes, shoreline wetness. **SSR Partial** (lite planar-FBO march). Not FFT. `docs/WATER.md` |
| Streaming | Real | `CreateWorldStream` grid load/unload around origin/follow. Jobs plan; GL upload on Flip |
| Instancing | Real | GPU `glDrawElementsInstanced` on 3.3 (`EnableGPUInstances`); CPU-merged mesh fallback. `SetInstanceData` = `SetInstanceTransform` |
| Modern GL | Partial | **3.3 Real:** UBO, GPU instancing, geometry billboards, caps query. **4.x optional:** compute / SSBO / tess — Real if the driver has them, else return 0 + one skip line. Never crash. `docs/GRAPHICS.md` |
| glslang | Real (validate) | `internal/glslang` + `CompileShader` / `bs shader`. SPIR-V **optional** (`glslangValidator` on PATH). No Vulkan. Native C++ glslang **not shipped** (`-tags glslang` reserved) |
| Gonum | Real | `internal/mathx` for FBM, IBL SH, crowd separation, `JobXform`. G3N scene graph stays `math32` |
| GI probes | Real | `CreateLightProbe` / `SetProbeGrid` drive hemi + ambient. Optional lightmap multiply. No RTX |
| Jobs | Real | Go worker pool. Workers must not touch GL |
| ECS (Flecs) | Real | 4.1.6. `EcsProgress` + Go-side `Position += Velocity * dt`. OOP entity handles still wrap G3N |
| Crowd | Partial | Local separation + Detour paths. **go-detour v0.1.3 has no DetourCrowd** |
| Editor / stats | Real | ImGui overlay commands + `examples/editor.bb` (FPS, draws, chunks, jobs, transform sliders) |
| Ebiten 2D | Real | `Graphics` / `Graphics2D`, images, sprites, Rect/Oval/Line, font, tiles, 2D particles |
| Oto audio | Real | `LoadSound` / `PlaySound` / volume / pitch, `LoadMusic`, `EmitSound`. No OpenAL |
| Physics 2D | Real | Chipmunk (`jakecoffman/cp`) — pin / spring / slide joints, `ApplyImpulse2D`, `Raycast2D` |
| Physics 3D | Real | Jolt on Windows / Linux amd64/arm64 / macOS ARM. Windows: native `SetVelocity` / `ApplyImpulse`, joints (`CreateHingeJoint` / point / slider / spring; body `0` = world), `CreateCharacterController` = CharacterVirtual, vehicles, `SetCCD` = LinearCast. Linux/macOS ARM: those extras stub to 0 / kinematic. Script `Raycast` is a sphere query on body positions. Older `CreateCharacter` is the kinematic helper. `BodySleep` / `BodyWake`. `PhysicsAsync` steps on a job then waits. `PhysicsThreads` stores a number only |
| Net | Real | UDP default. `CreateHost` / `PollNetwork` / `NET_CONNECT`. `go build -tags enet` for ENet |
| JSON | Real | JSON + YAML + MessagePack. **No FlatBuffers** |
| Detour + A* | Real | Mesh-triangle bake + grid A* + slope filter + crowd helper |
| ImGui | Real | `cimgui-go` v1.6.0 on the G3N GLFW + OpenGL 3.3 window |
| G3N widgets | Real | `CreatePanel` / `CreateButton` / `CreateSlider` / `SetOnClick` (scene-graph UI; names do not clash with `Gui*`) |
| GLFW window / input | Real | Primary + extra shared-context windows |
| PBR | Real | Opt-in metallic-roughness on `mbphysical` (GLSL 330). Shadows × direct only. IBL **Partial**: analytic hemi + optional 2D `SetEnvMap` lod — no prefiltered cubemap / BRDF LUT. `docs/PBR.md` |
| Assets | Partial | `LoadMesh` already accepts `.glb` / `.gltf` / `.obj` / `.dae`. PNG/JPEG textures. **No meshopt, no Basis/KTX2** (no extra C++ tree) |
| PostFX | Real | GL 3.3 FBO blit: tonemap, exposure, cheap bloom, FXAA, color grade. Not deferred MRT |
| Shader programs | Real | `CreateShader` / `LoadShader` / `SetShaderUniform` — GLSL 330 + PBR nodes. Not an Unreal graph |
| Input | Real | Keys, `KeyHit`, mouse, `MouseLook`, gamepad + deadzone |
| Serialization | Partial | JSON/YAML/MessagePack Real. `SaveScene` = transforms + tint; reload as cubes. `.bb` `LoadScene` still works |

## The 14 systems (honest)

| System | Status | Leftover Partial |
| --- | --- | --- |
| 1 World streaming | Real | — |
| 2 ECS (Flecs) | Real | G3N entities are a separate world |
| 3 Animation | Partial | Clip play/stop/time Real. `SetAnimBlend` stores a weight only (no dual-pose) |
| 4 Input | Real | — |
| 5 Physics extensions | Real (Windows) | Sleep/wake, 3D joints, CharacterVirtual, vehicles, `SetCCD` LinearCast, Chipmunk 2D joints. Linux/macOS ARM 3D joints / CCD / CharacterVirtual stub |
| 6 Navigation + AI | Partial | Detour bake + grid A* Real. No DetourCrowd C API |
| 7 Material / shader graph | Partial | PBR + Phong + `CreateShader` Real. No Unreal node graph |
| 8 Post-processing | Real | Cheap one-pass bloom/FXAA, not a film stack |
| 9 Sky / atmosphere | Real | Cubemap + preset + analytic dome + 3.3 clouds. Not Bruneton LUT |
| 10 Audio (Oto) | Real | — |
| 11 Editor tools | Real | ImGui + G3N widgets + stats |
| 12 Networking | Real | UDP default; ENet with `-tags enet` |
| 13 Asset pipeline | Partial | glTF/GLB/OBJ/DAE + PNG/JPEG + `bs build`. No FBX / meshopt / KTX2 |
| 14 Serialization | Partial | JSON/YAML/msgpack Real. Scene dump is transforms, not full meshes |

## Not in this product

| Item | Status |
| --- | --- |
| OpenGL 4.5 as a minimum | Never. Context is 3.3. Compute/SSBO/tess are optional |
| FFT ocean | Not on G3N/GL 3.3 |
| True SSR / deferred MRT | Not on G3N |
| Vulkan / bgfx / wgpu | Not wrapped |
| FlatBuffers | Not wrapped |
| FBX | Export to glTF / OBJ and `LoadMesh` |
| GOAP / behavior trees | Not included |
| OpenAL | Replaced by Oto |
| DetourCrowd C API | go-detour does not ship it; we use local avoidance |
| flywave/go-geo (full) | PROJ + GEOS CGO; we keep a pure-Go subset (`docs/GEO.md`) |

## Build

`go build ./cmd/bs` needs CGO (ImGui + Flecs + G3N/GLFW + Jolt on native targets). Language tests do not.

```powershell
go test ./internal/...
go build -o bs.exe ./cmd/bs
.\bs.exe examples\largeworld.bb
.\bs.exe examples\water.bb
.\bs.exe examples\weather.bb
.\bs.exe examples\terrain.bb
.\bs.exe examples\terrain_gl.bb
.\bs.exe examples\heightmap.bb
.\bs.exe examples\geo.bb
.\bs.exe examples\platform64.bb
```

Audio is Oto v3. On Windows, `bs build` copies `libc++.dll` and `libunwind.dll`. Jolt on Windows is static from `third_party/jolt/windows_amd64`. `go build -tags nojolt` → software 3D (`PhysicsBackend$()` = `fallback`).
