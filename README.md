# BitShin BASIC

Blitz3D-style BASIC for small 3D and 2D games, implemented in Go. `bs version` is **1.0.0**.

- **3D graphics:** [G3N](https://github.com/g3n/engine) + go-gl. **Runs on OpenGL 3.3.** GL 4.x (compute, SSBO, tessellation) is optional and feature-detected — old GPUs skip those commands instead of refusing to start. See [docs/GRAPHICS.md](docs/GRAPHICS.md).
- **Numerics:** [Gonum](https://www.gonum.org/) for FBM / SH / batch transforms. G3N scene types stay `math32`.
- **2D graphics:** [Ebiten](https://github.com/hajimehoshi/ebiten)
- **3D physics:** [jolt-go](https://github.com/bbitechnologies/jolt-go) on Linux amd64/arm64 and macOS ARM; Windows uses the same API plus `third_party/jolt/windows_amd64`. `-tags nojolt` or other OS/arch: software fallback (`PhysicsBackend$()` = `fallback`)
- **2D physics:** [jakecoffman/cp](https://github.com/jakecoffman/cp) (Chipmunk2D, pure Go)
- **Network:** [go-enet](https://github.com/codecat/go-enet) with `-tags enet`, otherwise UDP
- **Audio:** [Oto](https://github.com/ebitengine/oto) v3 for `.wav` / `.ogg` (no OpenAL)
- **UI:** [cimgui-go](https://github.com/AllenDang/cimgui-go) v1.6.0 on the same G3N GLFW window
- **ECS:** Flecs 4.1.6 (`github.com/SanderMertens/flecs-go` → `third_party/flecs-go`)
- **Pathfinding:** Detour + grid A*

```basic
Graphics3D(960, 600, 0, 2)
camera = CreateCamera()
light = CreateLight()
cube = CreateCube()
SetPosition(cube, 0, 0, 5)

While Not KeyDown(1)
    dt# = DeltaTime() * 60
    TurnEntity(cube, 0.4 * dt, 0.7 * dt, 0)
    RenderWorld
    Flip
Wend
```

`KeyDown(1)` is Escape. 3D axes match Blitz3D (Y up, +Z forward from the origin).

## Install

Need Go 1.25+ and, to **build the windowed CLI**, a C compiler plus **OpenGL 3.3 core** (4.x optional — never required). See [docs/COMPAT.md](docs/COMPAT.md).

```powershell
git clone <this-repo>
cd <repo-root>
go build -o bs.exe ./cmd/bs
go build -o bsls.exe ./cmd/bsls
```

### BitShin BASIC IDE (Wails + Svelte 5)

Modern desktop IDE with Monaco code editor, full IntelliSense autocomplete, instant F5 run & real-time console, 790+ searchable command reference browser, symbol outline, demo gallery, and standalone game packager:

```powershell
.\scripts\build_ide.ps1
.\BitShinIDE.exe
```

### Run spinning_cube (3D / G3N)

```powershell
.\bs.exe examples\spinning_cube.bb
```

A window opens with a turning cube. Esc (`KeyDown(1)`) quits.

### Run draw2d (2D / Ebiten)

```powershell
.\bs.exe examples\draw2d.bb
```

A bouncing oval and a rectangle. Esc quits.

### Other examples

```powershell
.\bs.exe examples\hello_console.bb
.\bs.exe examples\platform64.bb
.\bs.exe examples\windows.bb
.\bs.exe examples\lights.bb
.\bs.exe examples\playground.bb
.\bs.exe examples\shadows.bb
.\bs.exe examples\pbr.bb
.\bs.exe examples\nav.bb
.\bs.exe examples\platform2d.bb
.\bs.exe examples\tiles.bb
.\bs.exe examples\particles.bb
.\bs.exe examples\splitscreen.bb
.\bs.exe examples\scenes.bb
.\bs.exe examples\gui_demo.bb
.\bs.exe examples\hud_demo.bb
.\bs.exe examples\ssao_demo.bb
.\bs.exe examples\trigger_zone_demo.bb
.\bs.exe examples\ecs.bb
.\bs.exe examples\terrain.bb
.\bs.exe examples\terrain_gl.bb
.\bs.exe examples\heightmap.bb
.\bs.exe examples\geo.bb
.\bs.exe examples\water.bb
.\bs.exe examples\stream.bb
.\bs.exe examples\crowd.bb
.\bs.exe examples\editor.bb
.\bs.exe examples\largeworld.bb
```

### Standalone Ahead-of-Time (AOT) Native Compiler

Modern Blitz includes its own native compiler. You can compile any `.bb` script directly into a standalone machine-code binary with zero interpreter overhead:

```powershell
# Compile directly to a standalone native binary (.exe on Windows, ELF/Mach-O on Linux/macOS)
.\bs.exe compile examples\hud_demo.bb -o dist\game.exe

# Transpile BlitzBasic code into clean, formatted native Go source code
.\bs.exe transpile examples\hud_demo.bb -o game.go

# Export standalone HTML5 Web Player bundle (WASM)
.\bs.exe build examples\hud_demo.bb -o dist\web -os wasm
```

You can also package a portable folder using `bs build examples\scenes.bb -o dist`. That writes the binary, assets, and natives from `third_party/windows|linux|darwin` next to it. Run from that folder.

The built game keeps its window until the script ends, you press Escape (after the first frames; a focus ghost does not quit on frame 0), or you close the window. `Flip` presents the frame. Shaders, water, shadows, particles, and physics live in this runtime, so an exe you already compiled does not pick up a newer `bs.exe` until you compile that game again.

**Windows:** MinGW `gcc` on `PATH`. Dist copies `libc++.dll` and `libunwind.dll` (ImGui). Audio is Oto — no OpenAL DLL.  
**Linux / macOS:** a C compiler and an OpenGL driver. Audio is Oto — no OpenAL package.

Language tests do not need CGO.

Full commands: [DEVELOPMENT.md](DEVELOPMENT.md). How to change the tree: [CONTRIBUTING.md](CONTRIBUTING.md).

## 2D vs 3D

| | 3D | 2D |
| --- | --- | --- |
| Start | `Graphics3D w, h` | `Graphics2D w, h` (or `Graphics`) |
| Backend | G3N | Ebiten |
| Objects | `CreateCube`, meshes | `CreateSprite`, `DrawImage`, `Rect` |
| Physics | Jolt / fallback + `CreateBodySphere` | Chipmunk + `CreateCircle2D` |
| Loop | `UpdateWorld` / `RenderWorld` / `Flip` | `UpdateWorld` / `Cls` / `Flip` |

Do not mix `Graphics3D` and `Graphics2D` in one program.

## Examples

| File | Mode |
| --- | --- |
| `examples/hello_console.bb` | Language only |
| `examples/math.bb` | Degrees math library |
| `examples/spinning_cube.bb` | 3D cube |
| `examples/lights.bb` | Lights + fog + shadows |
| `examples/parent.bb` | `SetEntityParent` |
| `examples/playground.bb` | 3D primitives |
| `examples/primitives.bb` | Extra mesh primitives |
| `examples/mario64.bb` | Extra 3D platformer |
| `examples/windows.bb` | Extra GLFW window (shared context) |
| `examples/shadows.bb` | Directional + point/spot + EVSM/MSM |
| `examples/pbr.bb` | Opt-in metallic-roughness (GL 3.3) |
| `examples/rover.bb` | FPS walk |
| `examples/bounce.bb` | Collisions |
| `examples/jolt_drop.bb` | 3D rigid body |
| `examples/cloth.bb` | Soft-body flag + water flow |
| `examples/grab_beam.bb` | Convex hull, cylinder, grab/throw, laser |
| `examples/physics3d.bb` | Raycast, impulse, character |
| `examples/draw2d.bb` | 2D `Rect` / `Oval` (Ebiten) |
| `examples/platform64.bb` | 3D platformer (WASD + jump + coins, skybox, weather, shadows) |
| `examples/platform2d.bb` | Ebiten + Chipmunk |
| `examples/chipmunk2d.bb` | Chipmunk in a 3D view |
| `examples/net_echo.bb` | `HostNet` / `NetRecv` |
| `examples/net_host.bb` | `CreateNetworkHost` / `PollNetwork` |
| `examples/anim.bb` | `LoadAnimation` (needs `hero.glb`) |
| `examples/tiles.bb` | Tilemap |
| `examples/particles.bb` | 3D particles: `SetEmitterShape` `"soft"` or `"cube"` |
| `examples/particles2d.bb` | 2D `CreateEmitter2D` |
| `examples/splitscreen.bb` | `SetCameraViewport` / `CameraPick` |
| `examples/struct.bb` | Struct / Method / Import / Namespace |
| `examples/scenes.bb` | `ClearWorld` / `LoadScene` |
| `examples/gui_demo.bb` | ImGui on the G3N window |
| `examples/hud_demo.bb` | 2D HUD / canvas overlay on 3D (`Rect`, `Oval`, `Line`, `Text`) |
| `examples/ssao_demo.bb` | Screen-Space Ambient Occlusion (SSAO) post-processing |
| `examples/trigger_zone_demo.bb` | Physics trigger sensor zones + 3D audio |
| `examples/ecs.bb` | Flecs (console) |
| `examples/ecs_crowd.bb` | 2000 Flecs Position+Velocity |
| `examples/nav.bb` | Detour (mesh bake) + grid A* |
| `examples/crowd.bb` | Crowd separation + Detour agents |
| `examples/stream.bb` | World chunk stream (WASD) |
| `examples/terrain.bb` | Proc terrain + splat + height snap |
| `examples/outdoor.bb` | Grass, trees, props, time of day, rooms, use, save (`docs/PLAY.md`) |
| `examples/terrain_gl.bb` | Terrain-OpenGL port (splat, water, sky, clouds) |
| `examples/heightmap.bb` | Generate / erode / save PNG / mesh |
| `examples/geo.bb` | GeoJSON path + geo-bounded heightmap (WASD) |
| `examples/water.bb` | Scenic / LearnOpenGL-style water (DuDv, Fresnel, dual FBO) |
| `examples/gerstner.bb` | Gerstner ocean, wind, buoyancy, orbit cam |
| `examples/ocean.bb` | Gerstner + scenic reflect/refract combo |
| `examples/editor.bb` | ImGui scene list + profiler |
| `examples/largeworld.bb` | Terrain + water + instances + stats |
| `examples/glmodern.bb` | GL caps, instancing, UBO, optional compute/SSBO |
| `examples/weather.bb` | Atmosphere + clouds + rain/snow/fog/storm + lightning |
| `examples/postfx.bb` | Tonemap / bloom / FXAA blit |
| `examples/shader.bb` | `CreateShader` + `SetShader` |
| `examples/input.bb` | Keys, mouse look, gamepad |

## Language server (Cursor / VS Code)

Stdio JSON-RPC. No G3N window. Diagnostics come from the parser; hover/completion use keywords, builtins, `docs/COMMANDS.md` (when the workspace root is this repo), and same-file functions.

```powershell
go build -o bsls.exe ./cmd/bsls
.\bsls.exe
# or: .\bs.exe lsp
```

This workspace maps `*.bb` → language id `bitshinbasic` (`.vscode/settings.json`). Point the editor at `bsls.exe`, or install the local extension in `editors/vscode` (**Install Extension from Location…**). Then open a `.bb` file.

## Docs

- [docs/LANGUAGE.md](docs/LANGUAGE.md) — syntax
- [docs/PHYSICS.md](docs/PHYSICS.md) — Jolt bodies, joints, grab, cloth, CharacterVirtual
- [docs/MODERN_GAME_HELPERS.md](docs/MODERN_GAME_HELPERS.md) — FPS/TPS, tweens, grab / beam / projectile
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — packages and Flip
- [docs/STATUS.md](docs/STATUS.md) — Real / Partial / not in product (**1.0.0**)
- [docs/RELEASE.md](docs/RELEASE.md) — portable `bs build` folders
- [docs/COMPAT.md](docs/COMPAT.md) — OpenGL 3.3 required / 4.x optional
- [docs/GRAPHICS.md](docs/GRAPHICS.md) — optional compute / SSBO / tess / GPU instances
- [docs/LIGHTING.md](docs/LIGHTING.md) — Phong lights, falloff, indoor / outdoor, time of day
- [docs/PBR.md](docs/PBR.md) — metallic-roughness (`mbphysical`)
- [docs/TERRAIN.md](docs/TERRAIN.md) — heightmap, proc, LOD, stream (GL 3.3)
- [docs/PLAY.md](docs/PLAY.md) — grass, trees, props, time of day, rooms, use, save
- [docs/GEO.md](docs/GEO.md) — lon/lat origin, tiles, GeoJSON (go-geo subset)
- [docs/WATER.md](docs/WATER.md) — Gerstner + scenic dual-FBO water (no FFT)
- [docs/STREAM.md](docs/STREAM.md) — old chunk stream, and the `SetPlayer` bubble
- [docs/POSTFX.md](docs/POSTFX.md) — fullscreen blit stack
- [docs/SHADERS.md](docs/SHADERS.md) — GLSL 330 programs + PBR
- [docs/ASSETS.md](docs/ASSETS.md) — formats (`LoadMesh`, no FBX)
- [docs/NAV.md](docs/NAV.md) — Detour, crowd, terrain slopes
- [docs/SHADOWS.md](docs/SHADOWS.md) — CSM / cache / filters
- [docs/ECS.md](docs/ECS.md) — Flecs
- [DEVELOPMENT.md](DEVELOPMENT.md) — test and build
- [CONTRIBUTING.md](CONTRIBUTING.md) — where to put changes

## License

[MIT](LICENSE). Geo math is an original subset (`docs/GEO.md`); flywave/go-geo was not vendored (no upstream LICENSE file; PROJ/GEOS CGO).
