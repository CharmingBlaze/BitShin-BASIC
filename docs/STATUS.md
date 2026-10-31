# Status

**1.0.0** of this product: Windows game runtime + documented Linux/macOS Jolt subset + portable `moon build`. Not Vulkan (a later renderer replaces this one; do not wrap Vulkan on G3N) or a deferred renderer. FBX skin is clusters and limb translation, not the Autodesk SDK. Crowd avoidance samples velocities the way DetourCrowd does; the Recast C API is not linked.

What is wired and what is not. Classic command names still work; 2026 `Set*` / `Get*` aliases are registered on top.

**OpenGL baseline: 3.3 core.** G3N shaders are `#version 330 core` (G3N shaman prepends this). Extra GLFW windows hint 3.3. Never require 4.5. Optional 4.x is feature-detected (`docs/COMPAT.md`, `docs/GRAPHICS.md`).

## Pillars

| Pillar | Status | Notes |
| --- | --- | --- |
| Language | Real | This dialect: If / While / For / Repeat / Do, Exit, Static, Enum (`State.IDLE`), `$"{expr}"`, Const, Dim, Function, Include, Data/Read, Yes/No/Null, degree math, `Type`/`Struct` with methods, `Import [As]`, `Namespace`, coroutines, `@func`, `As`, `Math.*` / `String.*` / `Input.*` / `Entity.SetPos`. Raylib `App` / `Draw` / `Render` / `Model` names are not registered |
| G3N 3D | Real | Window, mesh primitives, lights, cameras, textures, glTF/OBJ/DAE, animation, pick, fog, cubemap skyboxes, weather, 3D particles (`SetEmitterShape` soft / cube / streak / flake). Lighting guide: `docs/LIGHTING.md`. Shadows: CSM + PCF/PCSS/EVSM/MSM (`docs/SHADOWS.md`). **GL 3.3** |
| Weather | Real (sky Partial) | Rain/snow/fog/storm particles, wetness, height fog, wind sway, lightning bolts. Atmosphere = Bruneton-style LUT + single scatter (`docs/WEATHER.md`). Not full 4D precompute / 3D clouds |
| Terrain | Real | Heightmap PNG/JPEG, FBM proc, **CPU generator** (`GenerateHeightmap` / erosion / PNG), **Terrain-OpenGL** value-noise + `mbterrain` splat (`CreateTerrainGL`), chunk stream + CPU LOD (`SetTerrainDetail` is the CPU step, not GL tess), `TerrainFoliage` / `TerrainTrees` / `TerrainProps`, `SetTerrainBlendMap`, `TerrainHeight` / `TerrainSlope`, one shader cloud sheet, far 3D trees as impostor cards. `docs/TERRAIN.md`, `docs/PLAY.md` |
| Geo | Real | Thin Web Mercator subset (`internal/geo`): `SetGeoOrigin`, `GeoProject` / `GeoUnproject`, XYZ/TMS tiles, `LoadGeoJSON`, DEM bounds → existing `CreateTerrain`. **Not** full flywave/go-geo (PROJ/GEOS/CGO). `docs/GEO.md` |
| Water | Real | Gerstner (vertex + wind + `WaterHeight`) and scenic dual-FBO / DuDv / Fresnel / depth tint / texture foam / dual normals; combo `ocean`. Optional `EnableFFT` Phillips tile. `WaterHeight` is the CPU inverse FFT. The water mesh samples a 64×64 half-float texture filled by a two-pass GL 3.3 fragment-shader inverse DFT of the same spectrum; `SetFFTStrength` scales it. Camera-centered LOD grid. Caustics, wakes, shoreline wetness. Auto Jolt buoyancy + `SetWaterFlow`. **SSR Partial** (lite planar-FBO march). Not a GPU FFT ocean. `docs/WATER.md` |
| Streaming | Real | `SetPlayer` follows terrain, water, and props. Near chunks collide. Shift past 4 km. `SetWorldBubble(0)` leaves it manual. |
| GPU particles | Real | `CreateGPUEmitter`: transform-feedback step + instanced billboards on GL 3.3, same `Emitter*` setters, turbulence, ground bounce, world shift. No compute path, soft particles, sorting, or mesh collision. `docs/PARTICLES.md` |
| Instancing | Real | GPU `glDrawElementsInstanced` on 3.3 (`EnableGPUInstances`); CPU-merged mesh fallback. `SetInstanceData` = `SetInstanceTransform` |
| Modern GL | Partial | **3.3 Real:** UBO, GPU instancing, geometry billboards, caps query. **4.x optional:** compute / SSBO / tess — Real if the driver has them, else return 0 + one skip line. Never crash. `docs/GRAPHICS.md` |
| glslang | Real (validate) | `internal/glslang` + `CompileShader` / `moon shader`. SPIR-V **optional** (`glslangValidator` on PATH). No Vulkan. Native C++ glslang **not shipped** (`-tags glslang` reserved) |
| Gonum | Real | `internal/mathx` for FBM, IBL SH, crowd separation, `JobXform`. G3N scene graph stays `math32` |
| GI probes | Real | `CreateLightProbe` / `SetProbeGrid` drive hemi + ambient. The nearest probe also captures a small cubemap (one face per frame) for PBR. Optional lightmap multiply. No RTX |
| Jobs | Real | Go worker pool. Workers must not touch GL |
| ECS (Flecs) | Real | 4.1.6. `EcsProgress` + Go-side `Position += Velocity * dt`. OOP entity handles still wrap G3N |
| Crowd | Real | Detour paths + Detour-style adaptive velocity samples + time-to-collision sidestep + yaw. The Recast DetourCrowd C API is not linked |
| Editor / stats | Real | ImGui overlay commands + `examples/editor.mb` (FPS, draws, chunks, jobs, transform sliders) |
| Ebiten 2D | Real | `Graphics` / `Graphics2D`, images, sprites (sheet anim, flip, tint, layer), 2D camera, Rect/Oval/Line, font, tiles with solid queries, 2D particles |
| Oto audio | Real | `LoadSound` / `PlaySound` / volume / pitch, `LoadMusic`, `EmitSound`. No OpenAL |
| Physics 2D | Real | Chipmunk (`jakecoffman/cp`) — pin / spring / slide joints, `ApplyImpulse2D`, `Raycast2D` |
| Physics 3D | Real | Jolt on Windows / Linux amd64/arm64 / macOS ARM. **Raycast** is Jolt `CastRay`. Windows extras: native velocity/impulse, joints, CharacterVirtual **with inner body**, vehicles, CCD LinearCast, mesh/heightfield, sensors, buoyancy, Jolt cloth, hull/cylinder/compound, 6DOF grab, explode, layers, debug overlay. Linux amd64/arm64 and macOS ARM use that same native backend (vehicles, soft-body cloth, rotated shapes, joints), linked to the Jolt 5.4 libraries shipped with `jolt-go`. `-tags joltgo` keeps the older jolt-go subset. Fallback (`-tags nojolt`): sphere + box AABB rays/sweeps, Verlet cloth, overlap contacts, cone/hinge software, ray CCD, **integrated quaternion**. `PhysicsThreads n` rebuilds the Jolt job pool and the Go `PhysicsAsync` pool. |
| Net | Real | One hosted room: join, LAN room list, waiting room, shared facts, small JSON, replicated movement. UDP by default. `go build -tags enet` for reliable delivery. Guide: `docs/NETWORK.md` |
| JSON | Real | JSON + YAML + MessagePack. **No FlatBuffers** |
| Detour + A* | Real | Mesh-triangle bake + grid A* + slope filter + crowd helper |
| ImGui | Real | `cimgui-go` v1.6.0 on the G3N GLFW + OpenGL 3.3 window (panels, menus, tabs, trees, sliders, color, combo). G3N's own widgets stay `CreateButton` / `CreatePanel` |
| G3N widgets | Real | `CreatePanel` / `CreateButton` / `CreateSlider` / `SetOnClick` (scene-graph UI; names do not clash with `Gui*`) |
| GLFW window / input | Real | Primary + extra shared-context windows |
| PBR | Real | Opt-in metallic-roughness on `mbphysical` (GLSL 330). Shadows × direct only. IBL: analytic hemi + UE4 EnvBRDF split-sum + prefiltered sky cubemap (`textureLod`) + optional 2D `SetEnvMap`. `docs/PBR.md` |
| Assets | Real | `LoadMesh`: `.glb` / `.gltf` / `.obj` / `.dae` / `.fbx` (mesh triangles). Textures: PNG/JPEG plus DDS and KTX/KTX2 **S3TC** uploaded compressed. `OptimizeMesh` uses the Go `meshoptim` package (Forsyth vertex cache, overdraw, vertex fetch). `SimplifyMesh` uses quadric error. KTX2 Zstd and ZLIB supercompression inflate to BC1–BC3. **No** BasisLZ transcoder or the meshoptimizer C++ library |
| PostFX | Real | GL 3.3 FBO blit: tonemap, exposure, half-res bloom chain, FXAA, color grade, grain, vignette, chromatic aberration, sharpen. Not deferred MRT |
| Shader programs | Real | `CreateShader` / `LoadShader` / `SetShaderUniform` — GLSL 330 + PBR nodes. Not an Unreal graph |
| Input | Real | Keys, `KeyHit`, mouse, `MouseLook`, gamepad + deadzone |
| Serialization | Real | JSON/YAML/MessagePack. `SaveScene` writes `kind`/`src`/parent/transform/tint/`visible`; lights dump as `kind: light`; cameras dump `fov`/`near`/`far`. `SceneLoad` rebuilds primitives, remeshes, lights, and cameras. Old dumps without `kind` still become cubes. `.mb` `LoadScene` still works |

## The 14 systems (honest)

| System | Status | Leftover Partial |
| --- | --- | --- |
| 1 World streaming | Real | `SetPlayer` bubble. `SetWorldBubble(0)` is the manual path |
| 2 ECS (Flecs) | Real | G3N entities are a separate world |
| 3 Animation | Real | Clip play/stop/time, `AttachToBone`, `SetAnimBlend`, `SetAnimLayer` / `SetAnimAdditive`, two-bone `SetIK`, chain `SetIKChain` (FABRIK). Not a full animation rig from FBX |
| 4 Input | Real | — |
| 5 Physics extensions | Real | Sleep/wake, 3D joints (incl. fixed/cone/swing-twist), 6DOF `Grab` at hit/anchor, compound collider, CharacterVirtual + inner body, vehicles, `SetCCD` LinearCast, Chipmunk 2D joints, collision layers, debug overlay. Linux amd64/arm64 and macOS ARM run the same native constraints, grab, cloth, and vehicles. Fallback integrates software rotation. `-tags joltgo` is the older subset. |
| 6 Navigation + AI | Real | Detour bake + grid A* + Detour-style crowd samples + behavior trees + GOAP (`CreatePlanner`). The DetourCrowd C API is not linked |
| 7 Material / shader graph | Partial | PBR + Phong + `CreateShader` Real. No Unreal node graph |
| 8 Post-processing | Real | Half-res bloom chain, tonemap, FXAA, grain, vignette, chromatic aberration, sharpen. Not deferred |
| 9 Sky / atmosphere | Real | Cubemap + preset + analytic dome + noise cloud sheet + Bruneton-style 2D LUT / single scatter. Not full 4D inscatter |
| 10 Audio (Oto) | Real | — |
| 11 Editor tools | Real | ImGui + G3N widgets + stats |
| 12 Networking | Real | Hosted room plus a raw socket API. UDP by default; ENet with `-tags enet`. LAN discovery only. See `docs/NETWORK.md` |
| 13 Asset pipeline | Real | glTF/GLB/OBJ/DAE/FBX mesh and skin clusters + PNG/JPEG + DDS/KTX S3TC + KTX2 Zstd/ZLIB + `OptimizeMesh` + `SimplifyMesh` + `moon build`. No FBX clips, BasisLZ, or meshoptimizer C++ |
| 14 Serialization | Real | JSON/YAML/msgpack + tagged scene dump + `ExportGLTF` for every mesh under the entity. Not skin, materials, or a full scene graph |

## Not in this product

| Item | Status |
| --- | --- |
| OpenGL 4.5 as a minimum | Never. Context is 3.3. Compute/SSBO/tess are optional |
| FFT ocean | `EnableFFT`: CPU inverse FFT for `WaterHeight`, GPU fragment-shader inverse DFT into the displacement texture. Not a compute-shader ocean |
| True SSR / deferred MRT | Not on G3N |
| Vulkan / bgfx / wgpu | Not wrapped. Stay on OpenGL 3.3 until the renderer is replaced |
| FlatBuffers | Not wrapped |
| Full FBX SDK (cameras, clips) | `LoadMesh` reads triangles, normals, UVs, and skin clusters. Clips are not read. Export a clip to glTF |
| GOAP | `CreatePlanner`, `AddPlanAction`, `Plan`. Not a utility-AI library |
| OpenAL | Replaced by Oto |
| DetourCrowd C API | go-detour does not ship it. Crowd ticks use a Go adaptive sampler with the same shape as DetourCrowd |
| flywave/go-geo (full) | PROJ + GEOS CGO; we keep a pure-Go subset (`docs/GEO.md`) |
| Raylib command registry | `App.Open`, `Draw.Text`, `Render.*`, `Model.*`, and the rest of that checklist. This runtime keeps G3N, Ebiten, and its own command names |

## Build

`go build ./cmd/bs` needs CGO (ImGui + Flecs + G3N/GLFW + Jolt on native targets). Language tests do not.

```powershell
go test ./internal/...
go build -o moon.exe ./cmd/bs
.\moon.exe examples\largeworld.mb
.\moon.exe examples\water.mb
.\moon.exe examples\weather.mb
.\moon.exe examples\terrain.mb
.\moon.exe examples\outdoor.mb
.\moon.exe examples\terrain_gl.mb
.\moon.exe examples\heightmap.mb
.\moon.exe examples\geo.mb
.\moon.exe examples\platform64.mb
```

Audio is Oto v3. Portable folders: [RELEASE.md](RELEASE.md). On Windows, `moon build` copies `libc++.dll` and `libunwind.dll`. Jolt on Windows is static from `third_party/jolt/windows_amd64`. `go build -tags nojolt` → software 3D (`PhysicsBackend$()` = `fallback`). Cross-OS `moon build -os` needs a CGO toolchain for that OS; it no longer copies the host binary on failure.
