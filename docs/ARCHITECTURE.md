# Architecture

```
.bb source
    → lex → parse → ast.Program
    → interp (tree-walk)
         Flip  → yield
         other → runtime.World.Call
                    ├─ G3N app loop          (Graphics3D)
                    ├─ Ebiten RunGame        (Graphics2D)
                    ├─ phys3d (Jolt/fallback)
                    ├─ phys2d (Chipmunk)
                    ├─ netenet (ENet/UDP)
                    ├─ animation (glTF clips)
                    ├─ ImGui (cimgui-go on the G3N GLFW window)
                    ├─ G3N gui widgets (CreateButton / CreatePanel)
                    ├─ Flecs (third_party/flecs-go)
                    ├─ jobs (Go pool; GL uploads on Flip only)
                    ├─ stream / terrain chunks (FBM + Terrain-OpenGL splat) / water Gerstner
                    ├─ pak (zip) / emitters / tilemaps / nav / data
                    ├─ glmodern (go-gl wrappers: compute/SSBO/UBO/instance/geom/tess)
                    ├─ postfx (GL 3.3 color FBO + fullscreen blit)
                    ├─ user shaders (GLSL 330 programs on meshes)
                    └─ mathx (Gonum) / glslang (GLSL validate, optional SPIR-V)
```

## Flip

`cmd/bs` runs the program until the first `Flip` (or `End`). Setup (`Graphics3D` / `Graphics2D`, cameras, sprites) happens on that first pass.

Then:

- **3D:** local `g3nHost.Run` (GLFW + renderer, no G3N OpenAL `app.App`) — each frame resumes the interpreter until the next `Flip`, then: flush job GL uploads, tick stream/terrain/water/instances/probes, render shadows, optional **planar water FBO**, G3N scene (optionally into a **PostFX** color FBO), fullscreen blit, ImGui. Extra windows (`CreateWindow`) share that **OpenGL 3.3** context: the scene is drawn to a texture on the main context, then blit + swap on the extra GLFW window. Extra windows hint `ContextVersion 3,3` — never 4.5.
- **2D:** `ebiten.RunGame` — `Update` resumes until `Flip`; `Draw` paints the clear color, immediate draw list, then sprites.

A normal `While Not KeyDown(1) … Flip Wend` **is** the game loop. Escape and `WindowShouldClose` are false until the first `Flip` (window create used to inject a phantom Esc). Still call `Flip` every frame. `Flip` is not allowed inside `Function`.

## Packages

| Package | Role |
| --- | --- |
| `internal/lex` | Tokens, comments, `<>` |
| `internal/parse` | Recursive descent, `Include` |
| `internal/ast` | Statements and expressions |
| `internal/interp` | Frames for `While`/`For`/`Function`; host builtins |
| `internal/value` | Number / string / array |
| `internal/runtime` | Command tables, input, G3N, Ebiten, `glmodern.go` |
| `internal/mathx` | Gonum numerics + math32 convert (not a G3N replace) |
| `internal/geo` | Pure-Go Web Mercator / tiles / GeoJSON (not full flywave/go-geo) |
| `internal/glslang` | GLSL validate; SPIR-V if `glslangValidator` on PATH |
| `internal/phys3d` | `World` interface; Jolt tagged build, fallback otherwise |
| `internal/phys2d` | `jakecoffman/cp/v2` space |
| `internal/netenet` | `Host` interface; UDP default, `enet` tag |
| `cmd/bs` | Parse file, expand includes, run; `bs build`; `bs lsp` |
| `cmd/bsls` / `internal/lsp` | Stdio language server for `.bb` (no window) |

## 2D vs 3D

Exactly one display mode per process:

- `Graphics3D` sets `mode2D = false` and creates the G3N window immediately.
- `Graphics2D` / `Graphics` sets `mode2D = true`; the Ebiten window starts in `Loop`.

`HasGraphics()` is true after either call. Console programs (`Print` only) never call them and exit after the first `Run`.

## Pinned libraries

| Role | Module | Why |
| --- | --- | --- |
| 3D | `github.com/g3n/engine` v0.2.0 | Upstream is more recently pushed than personal forks |
| 2D | `github.com/hajimehoshi/ebiten/v2` | Standard Go 2D engine |
| 3D physics | `github.com/bbitechnologies/jolt-go` v0.8.4 | Native Jolt: Windows (`internal/jolt` + `third_party/jolt/windows_amd64`), Linux amd64/arm64, macOS ARM. Software fallback otherwise and with `-tags nojolt` |
| 2D physics | `github.com/jakecoffman/cp/v2` v2.4.0 | Maintained Chipmunk port, no CGO (`go-zero/go-chipmunk` is abandoned) |
| Net | `github.com/codecat/go-enet` | Optional via `-tags enet` |
| ImGui | `github.com/AllenDang/cimgui-go` v1.6.0 | Dear ImGui widgets + OpenGL3 renderer on the existing G3N GLFW window (IO from G3N events; no second window, no GLFW 3.4 backend) |
| Flecs | `github.com/SanderMertens/flecs-go` → `./third_party/flecs-go` | Pins Flecs **4.1.6** C amalgamation (upstream Go module is unpublished) |
| Nav | `github.com/arl/go-detour` v0.1.3 | Recast/Detour bake from G3N mesh triangles (AABB fallback) |
| Grid A* | `github.com/quasilyte/pathing` | 2D cell paths |
| Data | `gopkg.in/yaml.v3`, `github.com/vmihailenco/msgpack/v5` | Plus `encoding/json` |
| Pool | `github.com/gobwas/pool` v0.2.1 | Banks and scratch buffers |
| Audio | `github.com/ebitengine/oto/v3` | Oto playback; wav/ogg decode is pure Go (no OpenAL) |
| OpenGL | `github.com/go-gl/gl` (G3N dep) | 3.3 core context; 4.x loaded only after feature-detect |
| Numerics | `gonum.org/v1/gonum` | Heavy CPU math; G3N stays `math32` |

## OpenGL policy

**Required: OpenGL 3.3 core.** G3N `window.Init` and extra GLFW windows hint 3.3 — never 4.5 as the minimum. After create, `detectModernGL` queries the real version.

**Optional:** compute, SSBO, tessellation (and any 4.3/4.5 API). Feature-detect; commands return 0 and print one skip line. See `docs/GRAPHICS.md`.

Raw go-gl calls save/restore GLS state (same pattern as `shadows.go`). Do not migrate to Vulkan/bgfx/wgpu.

## Jobs vs GL thread

`internal/runtime/jobs.go` is a small Go pool (`JobSubmit` / `JobWait` / `JobWaitAll`). Workers compute CPU data only (chunk plans, hashes). They **must not** create G3N geometries, textures, or FBOs. Results go on `enqueueGL` and run during `tickFX` / Flip on the thread that owns the G3N context.

Terrain and world-stream follow that path. Water Gerstner is a vertex shader on a regular grid (no compute FFT). Scenic water adds planar reflection + refraction cameras into color/depth FBOs (GL 3.3).

## Adding a backend hook

Implement the small interface (`phys3d.World`, `netenet.Host`) and keep the Blitz command names stable. Do not teach the interpreter about G3N or Ebiten types.
