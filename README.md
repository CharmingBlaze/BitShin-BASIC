# moonBASIC

BASIC for small 3D and 2D games. The window, meshes, and physics are this runtime (G3N, Ebiten, Jolt). `moon version` is **1.0.0**. Raylib command names are not part of this language.

- **3D graphics:** [G3N](https://github.com/g3n/engine) + go-gl. **Runs on OpenGL 3.3.** GL 4.x (compute, SSBO, tessellation) is optional and feature-detected — old GPUs skip those commands instead of refusing to start. See [docs/GRAPHICS.md](docs/GRAPHICS.md).
- **Numerics:** [Gonum](https://www.gonum.org/) for FBM / SH / batch transforms. G3N scene types stay `math32`.
- **2D graphics:** [Ebiten](https://github.com/hajimehoshi/ebiten)
- **3D physics:** native Jolt on Windows, Linux amd64/arm64, and macOS ARM (cars, tanks, bikes, cloth, and rotated colliders). Linux and macOS link the Jolt 5.4 libraries from [jolt-go](https://github.com/bbitechnologies/jolt-go). `-tags nojolt` or another OS/arch: software fallback (`PhysicsBackend$()` = `fallback`)
- **2D physics:** [jakecoffman/cp](https://github.com/jakecoffman/cp) (Chipmunk2D, pure Go)
- **Network:** one player hosts, friends join on the same network. [go-enet](https://github.com/codecat/go-enet) with `-tags enet`, otherwise UDP. See [docs/NETWORK.md](docs/NETWORK.md)
- **Audio:** [Oto](https://github.com/ebitengine/oto) v3 for `.wav` / `.ogg` (no OpenAL)
- **UI:** [cimgui-go](https://github.com/AllenDang/cimgui-go) v1.6.0 on the same G3N GLFW window
- **ECS:** Flecs 4.1.6 (`github.com/SanderMertens/flecs-go` → `third_party/flecs-go`)
- **Pathfinding:** Detour + grid A*

```basic
Graphics3D(960, 600, 0, 2)
camera = CreateCamera()
light = CreateLight()
cube = CreateCube()
Entity.SetPos(cube, 0, 0, 5)

While Not Input.KeyDown(1)
    dt# = Time.Delta() * 60
    TurnEntity(cube, 0.4 * dt, 0.7 * dt, 0)
    RenderWorld
    Flip
Wend
```

`Input.KeyDown(1)` is Escape. 3D axes match Blitz3D (Y up, +Z forward from the origin).

## Install

Need Go 1.25+ and, to **build the windowed CLI**, a C compiler plus **OpenGL 3.3 core** (4.x optional — never required). See [docs/COMPAT.md](docs/COMPAT.md).

```powershell
git clone <this-repo>
cd <repo-root>
go build -o moon.exe ./cmd/moon
go build -o moonsls.exe ./cmd/moonsls
```

### moonBASIC IDE (Neutralino + Svelte 5)

Modern desktop IDE with Monaco code editor, full IntelliSense autocomplete, instant F5 run & real-time console, 790+ searchable command reference browser, symbol outline, demo gallery, and standalone game packager:

```powershell
.\scripts\build_ide.ps1
.\moonbasic.exe
```

### Run spinning_cube (3D / G3N)

```powershell
.\moon.exe examples\spinning_cube.mb
```

A window opens with a turning cube. Esc (`Input.KeyDown(1)`) quits.

### Run draw2d (2D / Ebiten)

```powershell
.\moon.exe examples\draw2d.mb
```

A bouncing oval and a rectangle. Esc quits.

### Other examples

```powershell
.\moon.exe examples\hello_console.mb
.\moon.exe examples\platform64.mb
.\moon.exe examples\windows.mb
.\moon.exe examples\lights.mb
.\moon.exe examples\playground.mb
.\moon.exe examples\shadows.mb
.\moon.exe examples\pbr.mb
.\moon.exe examples\nav.mb
.\moon.exe examples\platform2d.mb
.\moon.exe examples\tiles.mb
.\moon.exe examples\particles.mb
.\moon.exe examples\splitscreen.mb
.\moon.exe examples\scenes.mb
.\moon.exe examples\gui_demo.mb
.\moon.exe examples\gui_panels.mb
.\moon.exe examples\gui_scene.mb
.\moon.exe examples\gui_3d.mb
.\moon.exe examples\hud_demo.mb
.\moon.exe examples\ssao_demo.mb
.\moon.exe examples\trigger_zone_demo.mb
.\moon.exe examples\ecs.mb
.\moon.exe examples\terrain.mb
.\moon.exe examples\terrain_gl.mb
.\moon.exe examples\heightmap.mb
.\moon.exe examples\geo.mb
.\moon.exe examples\water.mb
.\moon.exe examples\stream.mb
.\moon.exe examples\crowd.mb
.\moon.exe examples\editor.mb
.\moon.exe examples\largeworld.mb
```

### Standalone Ahead-of-Time (AOT) Native Compiler

moonBASIC includes its own native compiler. You can compile any `.mb` script directly into a standalone machine-code binary with zero interpreter overhead:

```powershell
# Compile directly to a standalone native binary (.exe on Windows, ELF/Mach-O on Linux/macOS)
.\moon.exe compile examples\hud_demo.mb -o dist\game.exe

# Transpile a script into Go source
.\moon.exe transpile examples\hud_demo.mb -o game.go

# Export standalone HTML5 Web Player bundle (WASM)
.\moon.exe build examples\hud_demo.mb -o dist\web -os wasm
```

You can also package a portable folder using `moon build examples\scenes.mb -o dist`. That writes the binary, assets, and natives from `third_party/windows|linux|darwin` next to it. Run from that folder.

The built game keeps its window until the script ends, you press Escape (after the first frames; a focus ghost does not quit on frame 0), or you close the window. `Flip` presents the frame. Shaders, water, shadows, particles, and physics live in this runtime, so an exe you already compiled does not pick up a newer `moon.exe` until you compile that game again.

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
| `examples/hello_console.mb` | Language only |
| `examples/math.mb` | Degrees math library |
| `examples/spinning_cube.mb` | 3D cube |
| `examples/lights.mb` | Lights + fog + shadows |
| `examples/parent.mb` | `SetEntityParent` |
| `examples/playground.mb` | 3D primitives |
| `examples/primitives.mb` | Extra mesh primitives |
| `examples/mario64.mb` | Extra 3D platformer |
| `examples/windows.mb` | Extra GLFW window (shared context) |
| `examples/shadows.mb` | Directional + point/spot + EVSM/MSM |
| `examples/pbr.mb` | Opt-in metallic-roughness (GL 3.3) |
| `examples/rover.mb` | FPS walk |
| `examples/bounce.mb` | Collisions |
| `examples/jolt_drop.mb` | 3D rigid body |
| `examples/cloth.mb` | Soft-body flag + water flow |
| `examples/grab_beam.mb` | Convex hull, cylinder, grab/throw, laser |
| `examples/physics3d.mb` | Raycast, impulse, character |
| `examples/draw2d.mb` | 2D `Rect` / `Oval` (Ebiten) |
| `examples/platform64.mb` | 3D platformer (WASD + jump + coins, skybox, weather, shadows) |
| `examples/platform2d.mb` | Ebiten + Chipmunk |
| `examples/chipmunk2d.mb` | Chipmunk in a 3D view |
| `examples/net_echo.mb` | `HostGame` / `JoinGame` / `NetSay` |
| `examples/net_lobby.mb` | Waiting room: `NetLobby$`, `NetReady`, `StartGame` |
| `examples/net_host.mb` | `CreateNetworkHost` / `PollNetwork` |
| `examples/net_physics.mb` | Handle API plus a physics kick |
| `examples/anim.mb` | `LoadAnimation` (needs `hero.glb`) |
| `examples/tiles.mb` | Tilemap |
| `examples/particles.mb` | 3D particles: `SetEmitterShape` `"soft"` or `"cube"` |
| `examples/particles2d.mb` | 2D `CreateEmitter2D` |
| `examples/splitscreen.mb` | `SetCameraViewport` / `CameraPick` |
| `examples/struct.mb` | Struct / Method / Import / Namespace |
| `examples/scenes.mb` | `ClearWorld` / `LoadScene` |
| `examples/gui_demo.mb` | ImGui inspector on the G3N window |
| `examples/gui_panels.mb` | ImGui tabs, tree, list, radio, drag, popup |
| `examples/gui_scene.mb` | G3N scene widgets: panel, button, slider, edit |
| `examples/gui_3d.mb` | ImGui panel on a lit 3D scene. Right-drag orbits |
| `examples/hud_demo.mb` | 2D HUD / canvas overlay on 3D (`Rect`, `Oval`, `Line`, `Text`) |
| `examples/ssao_demo.mb` | Screen-Space Ambient Occlusion (SSAO) post-processing |
| `examples/trigger_zone_demo.mb` | Physics trigger sensor zones + 3D audio |
| `examples/ecs.mb` | Flecs (console) |
| `examples/ecs_crowd.mb` | 2000 Flecs Position+Velocity |
| `examples/nav.mb` | Detour (mesh bake) + grid A* |
| `examples/crowd.mb` | Crowd separation + Detour agents |
| `examples/stream.mb` | World chunk stream (WASD) |
| `examples/terrain.mb` | Proc terrain + splat + height snap |
| `examples/outdoor.mb` | Grass, trees, props, time of day, rooms, use, save (`docs/PLAY.md`) |
| `examples/terrain_gl.mb` | Terrain-OpenGL port (splat, water, sky, clouds) |
| `examples/heightmap.mb` | Generate / erode / save PNG / mesh |
| `examples/geo.mb` | GeoJSON path + geo-bounded heightmap (WASD) |
| `examples/water.mb` | Scenic / LearnOpenGL-style water (DuDv, Fresnel, dual FBO) |
| `examples/gerstner.mb` | Gerstner ocean, wind, buoyancy, orbit cam |
| `examples/ocean.mb` | Gerstner + scenic reflect/refract combo |
| `examples/editor.mb` | ImGui scene list + profiler |
| `examples/largeworld.mb` | Terrain + water + instances + stats |
| `examples/glmodern.mb` | GL caps, instancing, UBO, optional compute/SSBO |
| `examples/weather.mb` | Atmosphere + clouds + rain/snow/fog/storm + lightning |
| `examples/postfx.mb` | Tonemap / bloom / FXAA blit |
| `examples/shader.mb` | `CreateShader` + `SetShader` |
| `examples/input.mb` | Keys, mouse look, gamepad |

## Language server (Cursor / VS Code)

Stdio JSON-RPC. No G3N window. Diagnostics come from the parser; hover/completion use keywords, builtins, `docs/COMMANDS.md` (when the workspace root is this repo), and same-file functions.

```powershell
go build -o moonsls.exe ./cmd/moonsls
.\moonsls.exe
# or: .\moon.exe lsp
```

This workspace maps `*.mb` → language id `moonbasic` (`.vscode/settings.json`). Point the editor at `moonsls.exe`, or install the local extension in `editors/vscode` (**Install Extension from Location…**). Then open a `.mb` file.

## Docs

- [docs/LANGUAGE.md](docs/LANGUAGE.md) — syntax
- [docs/NETWORK.md](docs/NETWORK.md) — host a game, find a room, wait, play
- [docs/PHYSICS.md](docs/PHYSICS.md) — Jolt bodies, joints, grab, cloth, CharacterVirtual
- [docs/MODERN_GAME_HELPERS.md](docs/MODERN_GAME_HELPERS.md) — FPS/TPS, tweens, grab / beam / projectile
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — packages and Flip
- [docs/STATUS.md](docs/STATUS.md) — Real / Partial / not in product (**1.0.0**)
- [docs/RELEASE.md](docs/RELEASE.md) — portable `moon build` folders
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
