# moonBASIC Roadmap

The full moonBASIC engine specification as a checklist, marked against the code as of **2026-10-31** (`moonbasic-baseline` / `main` at `e380a45`).

moonBASIC is four products:

1. **moonBASIC language:** the BASIC dialect, interpreter, and `moon compile` native compiler.
2. **Moon Engine:** the 2D/3D runtime (G3N on OpenGL 3.3, Ebiten 2D, Jolt 3D physics, Chipmunk 2D physics, Oto audio).
3. **Moon Studio:** the IDE (`moonBASICIDE.exe`, `ide/`).
4. **Moon Builder:** `moon build`, `moon compile`, packaging, and export.

The design goal is the simplicity of Blitz3D with the capabilities of a modern engine, on hardware from 2016 onward. The BASIC language stays the heart of the product: a developer should never need Go, G3N, or Jolt to make a game.

[docs/STATUS.md](STATUS.md) is the short, per-system view of what ships. This file is the long view: every item in the spec, and what is left.

## How to read this

| Mark | Meaning |
| --- | --- |
| `- [x]` | Done: in the code, with the command or file that proves it |
| `- [ ] … **Partial:**` | Some of it exists; the note says what is missing |
| `- [ ] … Missing` | Not started |

Command names in the spec are suggestions. Where moonBASIC has the feature under another name (often the Blitz3D one), the item is marked done and the real name is given. Each item was checked against the registered command tables in `internal/runtime`, the language in `internal/parse` and `internal/interp`, `cmd/moon`, and `ide/`. Documentation alone was not counted.

**Keeping it current:** when a feature lands, tick its item here in the same change and name the command, then run `go run tools/roadmap_counts.go` to recount the table below.

## Where things stand

Coverage counts a partial item as half.

| Area | Done | Partial | Missing | Coverage |
| --- | --- | --- | --- | --- |
| [1. Core engine architecture](#1-core-engine-architecture) | 11 | 15 | 1 | 69% |
| [2. MoonBASIC programming language](#2-moonbasic-programming-language) | 46 | 20 | 14 | 70% |
| [3. 2D graphics engine](#3-2d-graphics-engine) | 74 | 4 | 12 | 84% |
| [4. 3D graphics engine](#4-3d-graphics-engine) | 106 | 16 | 16 | 83% |
| [5. Physics (Jolt)](#5-physics-jolt) | 45 | 8 | 3 | 88% |
| [6. Terrain and world building](#6-terrain-and-world-building) | 36 | 9 | 5 | 81% |
| [7. Water and environment](#7-water-and-environment) | 26 | 4 | 1 | 90% |
| [8. Particles and visual effects](#8-particles-and-visual-effects) | 17 | 10 | 6 | 67% |
| [9. Animation and skeletal systems](#9-animation-and-skeletal-systems) | 7 | 6 | 7 | 50% |
| [10. Audio engine](#10-audio-engine) | 11 | 4 | 22 | 35% |
| [11. UI engine](#11-ui-engine) | 17 | 4 | 17 | 50% |
| [12. Navigation and AI](#12-navigation-and-ai) | 9 | 4 | 10 | 48% |
| [13. Networking and multiplayer](#13-networking-and-multiplayer) | 9 | 4 | 6 | 58% |
| [14. Input and controller system](#14-input-and-controller-system) | 21 | 5 | 6 | 73% |
| [15. Rendering and graphics quality](#15-rendering-and-graphics-quality) | 20 | 7 | 18 | 52% |
| [16. Assets and import pipeline](#16-assets-and-import-pipeline) | 20 | 10 | 27 | 44% |
| [17. Scenes and prefabs](#17-scenes-and-prefabs) | 16 | 7 | 8 | 63% |
| [18. Moon Studio (IDE / editor)](#18-moon-studio-ide--editor) | 21 | 13 | 47 | 34% |
| [19. Shaders and visual programming](#19-shaders-and-visual-programming) | 7 | 3 | 3 | 65% |
| [20. Development and debugging tools](#20-development-and-debugging-tools) | 33 | 8 | 5 | 80% |
| [21. Optimization and performance](#21-optimization-and-performance) | 12 | 8 | 18 | 42% |
| [22. Graphics API and platform](#22-graphics-api-and-platform) | 11 | 3 | 4 | 69% |
| [23. Build system and export](#23-build-system-and-export) | 11 | 8 | 13 | 47% |
| [24. Project management](#24-project-management) | 4 | 3 | 4 | 50% |
| [25. Documentation and community](#25-documentation-and-community) | 3 | 9 | 4 | 47% |
| [26. Plugins and extensions](#26-plugins-and-extensions) | 0 | 3 | 12 | 10% |
| **Total** | **593** | **195** | **289** | **64%** |

**Strongest:** water and environment, the 2D and 3D engines, terrain, physics, and input. **Weakest:** plugins, project management, Moon Studio's visual editing, optimization tooling, and audio.

## The MoonBASIC 1.0 checklist

The 32 features the spec will not compromise on for 1.0. **23 of 32 are done.**

### Language
- [x] Stable BASIC syntax and compiler/runtime — interpreter, `moon compile` AOT, `moon transpile`; no frozen language spec yet
- [x] Variables, functions, types and arrays — `Function`, `Type`/`Struct` with methods, `Dim`, `Enum`, `Const`
- [ ] Complete standard library — **Partial:** math, strings, files, JSON/YAML/msgpack, random, time; thin on compression, binary I/O, system info
- [x] Error reporting and debugging — file:line errors, `Assert`, `DebugLog`; debugger with step over/into/out, call stack, variables, watches
- [x] BASIC command documentation — [COMMANDS.md](COMMANDS.md), IDE hover and command search

### 2D engine
- [x] Sprite rendering and animation — `CreateSprite`, `SetSpriteSheet`, `AnimateSprite`, `FlipSprite`, layers
- [x] Tilemaps — layers, `LoadTiledMap` (.tmx/.tmj), flips, animated tiles, autotiling, per-camera culling
- [x] 2D cameras — camera objects with zoom, rotation, viewports, smooth follow, dead zone, bounds, shake
- [x] 2D collision and physics — Chipmunk bodies, capsules, kinematic, layers and masks, sensors, shape casts, `CreateCharacter2D`
- [x] Basic lighting and particles — `CreateLight2D`, sun, occluder shadows; 2D particles in world space

### 3D engine
- [x] Primitive geometry and model importing — 13 primitives; glTF/GLB, OBJ, DAE, FBX meshes
- [x] Entity and transform system — Blitz3D entity and transform commands, parenting, quaternions
- [x] Cameras and lighting — perspective/ortho, split-screen, pick; directional/point/spot/area lights, CSM, IBL, probes
- [x] Materials and textures — Phong and PBR, unlit, toon, normal/AO/emissive maps, compressed textures
- [x] Static and animated meshes — `LoadMesh`, `LoadAnimMesh`, clips, blend trees, crossfades, IK
- [x] Jolt physics integration — rigid bodies, shapes, joints, characters, vehicles, cloth, casts

### Core systems
- [ ] Scene management — **Partial:** `SaveScene`/`LoadScene`, `ClearWorld`; no named scenes or transitions
- [ ] Resource management — **Partial:** `OpenPak` zip assets; no central cache or reference tracking
- [x] Input and audio — keyboard, mouse, gamepad, action binds; `LoadSound`/`PlaySound`, 3D `EmitSound`, music
- [x] Serialization — JSON/YAML/MessagePack, scene dumps, save slots, `ExportGLTF`
- [x] Performance profiling — `ShowProfiler` overlay and `Stats*` CPU, GPU, memory timings

### Moon Studio
- [ ] Project manager — **Partial:** open a folder; no create/recent projects
- [ ] Scene hierarchy and inspector — **Partial:** read-only scene list in the IDE; editable only in the in-game ImGui overlay
- [ ] 2D and 3D viewports — Missing
- [x] Integrated code editor — Monaco with highlighting, completion, hover docs, diagnostics via `moonsls`
- [ ] Play-in-editor and debugger — **Partial:** full IDE debugger (stack, variables, watches, F10/F11); the game still runs in a separate window
- [x] Asset browser — `AssetBrowser.svelte`

### Deployment
- [x] Windows game exporting — `moon build`, `moon compile`
- [ ] Release and debug builds — **Partial:** interpreter bundle vs native compile; no build profiles
- [x] Asset packaging — `moon build` copies referenced assets and natives
- [x] Runtime independent of editor — `moon.exe` runs games without the IDE
- [ ] Hardware compatibility testing — **Partial:** auto quality tiers; measured on a Radeon 780M only

## Release roadmap

| Release | Scope | Status |
| --- | --- | --- |
| 0.1 Language foundation | Lexer, parser, interpreter, types, window, input, basic 2D/3D, primitives, cameras, transforms, command docs | **Done** |
| 0.2 Game engine | Entities, scenes, materials, lighting, Jolt, 2D physics, audio, model import, animation | **Done** (scenes basic) |
| 0.3 Moon Studio | Project manager, scene editor, inspector, asset browser, code editor, debugger, play-in-editor, undo/redo | **Partial:** code editor and asset browser only |
| 0.4 Advanced engine | Terrain, water, particles, navigation, advanced materials, post-processing, visual shader editor, prefabs | **Mostly done:** no visual shader editor; prefabs minimal |
| 0.5 Production pipeline | Asset optimization, export system, build profiles, profilers, platform compatibility, plugins | **Partial:** mesh optimization and export done; no profiles, profilers, or plugins |
| 1.0 Production release | Stable language and API, complete docs, reliable export, automated tests, compatibility testing, example games, production editor | **Not yet:** 23 of 32 checklist items |

`moon version` already reports 1.0.0 for the runtime. Against this spec, the product is between 0.4 and 0.5: the engine is ahead, the editor and tooling are behind.

## The plan

What to build next, in order. Each milestone closes 1.0 checklist items or strengthens what makes moonBASIC different.

### Milestone A: finish the 2D engine — done 2026-10-31
2D was the weakest part of "one language for 2D and 3D". Examples: `camera2d.mb`, `tilemap2d.mb`, `lighting2d.mb`, `platformer2d_physics.mb`. Left for later: 2D skeletal animation, light masks, normal-mapped sprites, growing (infinite) tilemaps.
- 2D camera: zoom, rotation, bounds, smoothing, dead zone, shake, viewports, camera objects
- Tilemaps: layers, Tiled (TMX/JSON) import, view culling and chunking, animated tiles, flip/rotate, per-tile metadata
- 2D lighting: point and ambient lights with falloff and an occluder shadow pass
- 2D physics: collision layers and masks, sensors, kinematic bodies, a platformer character controller
- Sprites: pivot, blend modes, parenting, arbitrary UV rects

### Milestone B: debugging and profiling — done 2026-11-01
- `DebugLog`, `Assert`, `DebugBreak`; step over/into/out, call stack, variables, and watches in `moon -debug` and an IDE debug sidebar
- `ShowProfiler`: frame-time graph, script/physics/render CPU time, GPU time, draws, triangles, memory, GC
- `DebugDraw`: physics shapes, contacts, velocities, joints, raycasts, nav meshes, light ranges, bounds, normals, entity IDs, frustums, audio; `DebugLine`/`DebugSphere`/`DebugBox` for scripts
- Fixed: HUD `Text` vanished without a shadow-casting sun; planes, quads, and disks carried a hidden pitch that `RotateEntity` and `ScaleEntity` fought. The suspected post-pass sRGB problem was not one: the content is tuned for the current output, and encoding it washed scenes out.
- Left for later: per-function script profiling, a profiler panel in the IDE, per-asset memory diagnostics, multi-file positions in the debugger.

### Milestone C: Moon Studio scene editing
The largest gap, and the core of the "proper integrated editor" goal.
- A 3D (then 2D) viewport inside the IDE, driven by the runtime in an editor mode
- Editable hierarchy and inspector over the scene format; transform gizmos; snapping; undo/redo
- Play-in-editor; project create/open/recent; project settings file

### Milestone D: retro rendering pack
A selling point no mainstream engine leads with.
- Low-resolution render target with nearest upscale, pixel-perfect mode
- Vertex snapping, affine texture toggle, nearest filtering per texture
- Ordered dithering, fixed palettes, colour banding as post passes; retro fog presets

### Milestone E: content and export pipeline
- Script mesh API: `CreateMesh`, `AddVertex`, `AddTriangle`, `UpdateMesh`, vertex colours, second UV set
- `moon build`: pak files, icon and version metadata, build profiles, headless/server builds
- More formats: BMP, TGA, WebP, MP3, FLAC, STL, PLY; streaming music
- Asset hot reload during development

### Milestone F: audio and language polish
- Audio channels: pause/resume, fade, pan, seek, buses, reverb, streaming
- Blitz compatibility: `Goto`/`Gosub`, `Rem`, multiline comments, typed `Local`
- BASIC-level unit tests and benchmarks; documentation comments

### Milestone G: compatibility testing
- Measure the low tier on a 2016-class Intel HD 520 and a 2018-class GTX 1050, the high tier on a current discrete GPU
- Windows in CI; publish a hardware table in [COMPAT.md](COMPAT.md)

### After 1.0
Plugins, visual shader editor, visual scripting, networking beyond hosted rooms (TCP, WebSocket, prediction, dedicated servers), additional graphics backends (Vulkan, Direct3D, Metal), Android and Web targets.

## Full checklist

### 1. Core engine architecture

#### Foundation
- [x] Engine core — lifecycle and initialization — `graphics3d`/`graphics2d`, `renderworld`, `flip`, `endgraphics`
- [ ] Module system — independently loadable subsystems — **Partial:** build tags `nojolt`, `enet` only; no runtime-loadable modules
- [ ] Scene manager — create, load, switch scenes — **Partial:** `scenesave`/`sceneload`, `loadlevel`, `clearworld`; no multiple named scenes
- [x] Entity system — `createentity`, `createpivot`, `freeentity`, `cloneentity`, entity get/set commands
- [x] Component system — attach behaviours and properties — flecs ECS (`ecsadd`, `ecsquery`), `createbehavior`/`setbehavior`
- [ ] Resource manager — centralized asset management — **Partial:** `openpak`/`setpak` zip assets; no central ref-counted cache API
- [ ] Memory manager — allocation and resource tracking — **Partial:** `createpool`/`poolget`/`poolput`, banks; no tracking/reporting
- [ ] Job system — background tasks and parallel work — **Partial:** internal pool (jobs.go); `jobsubmit` runs synthetic work, not BASIC code
- [ ] Event system — communication between engine systems — **Partial:** `callback`, `setonclick`, `netregister`; no general event bus
- [ ] Time system — delta time, fixed time, timers — **Partial:** `deltatime`, `createtimer`, `settimescale`; no fixed-step API
- [ ] Input system — keyboard, mouse, controller, touch — **Partial:** keyboard/mouse/gamepad via GLFW; no touch
- [x] Platform layer — OS and hardware abstraction — GLFW + build-tagged g3napp_windows.go / g3napp_other.go
- [ ] Configuration — engine and project settings — **Partial:** `setgraphicsquality`, `moon build` flags; no project config file
- [ ] Logging — runtime and editor — **Partial:** stdout println, `shaderlog`/`computelog`; no log levels/files
- [x] Error handling — engine errors and recovery — `Try`/`Catch`, `throw`, panic recover in world.go Flip/RenderWorld
- [x] Serialization — save and load engine data — `scenesave`/`sceneload`, `saveslot`, `packsave`, `jsonsave`, `saveprefab`
- [ ] Plugin system — Missing
- [x] Profiler — `ShowProfiler` overlay and `Stats*` timings (CPU, GPU, memory)

#### Architectural requirements
- [x] Go-first implementation — Go module `moonbasic`; cgo only for Jolt/GLFW
- [ ] Minimal native dependencies — **Partial:** cgo Jolt, GLFW/OpenGL, libc++.dll; `nojolt` tag drops Jolt
- [x] Jolt isolated behind a Go API — internal/jolt, internal/phys3d wrappers
- [ ] G3N objects kept out of the public BASIC API — **Partial:** handles are ints, but `g3nbutton` command leaks G3N name
- [x] Editor code separate from runtime code — ide/ is a separate Go module (Wails app)
- [x] 2D and 3D rendering separate — `graphics2d` vs `graphics3d`, commands_2d.go / graphics2d.go
- [ ] Headless execution for tests and servers — **Partial:** `moon smoke` uses hidden GL window; no true headless runtime
- [ ] Low per-frame allocation — **Partial:** pools and interp fast-path cache exist; not measured/enforced
- [ ] Unused subsystems can be disabled — **Partial:** build tags `nojolt`, `enet`; nothing else switchable

### 2. MoonBASIC programming language

#### 2.1 Syntax
- [x] Case-insensitive commands — lex.IdentKey lowercases identifiers
- [x] Optional semicolons — `:` separates statements; `;` starts a comment so trailing ones are harmless
- [x] Newline-based statements — lexer emits Newline tokens as terminators
- [x] If / Else / ElseIf / EndIf — internal/parse/parse.go `if`/`elseif`/`endif`
- [x] For / Next — parse `for`/`next`, plus `For Each`/`For In`
- [x] While / Wend — parse `while`/`wend`
- [x] Repeat / Until — parse `repeat`/`until`
- [x] Select / Case — parse `select`/`case`/`default`
- [x] Function / EndFunction — parse `function`, `End Function`
- [x] Type / EndType — parse `type`/`struct` with `field`
- [ ] Goto / Gosub — Missing
- [ ] Comments with `;` or `Rem` — **Partial:** `;`, `'`, `//` work; no `Rem`
- [ ] Multiline comments — Missing
- [x] Constants — `Const`
- [x] Global and local variables — `Global`, `Local`, `Static`
- [ ] Strongly defined variable types — **Partial:** `As` on params/returns/Dim, `Strict` mode; locals untyped
- [x] Optional type inference — dynamic typing; `As` annotations optional
- [x] Arrays and multidimensional arrays — `Dim a(x,y)`, `Redim`, `Erase`, `arraysize`
- [x] Lists and collections — `createlist`/`listadd`, `createmap`/`mapset`, `tablecreate`
- [x] Enumerations — `Enum`
- [x] Structures — `Struct` keyword (parseType)
- [x] User-defined types — `Type ... Field`, `New`, `Delete`, `Method`
- [ ] Function overloading — Missing
- [x] Optional parameters — default param values (`x = 1`) in parseParamList
- [ ] Named parameters — Missing
- [x] Return values — `Return`, `As` return type
- [x] Modules and imports — `Include`, `Import "file" As name`
- [x] Namespaces — `Namespace ... End Namespace`
- [x] Error handling (Try/Catch or equivalent) — `Try`/`Catch`/`EndTry`, `throw`
- [ ] Generics — Missing
- Also: `Coroutine`/`Yield`, `Data`/`Read`/`Restore`, state machines (`createstatemachine`), `Strict`.

#### 2.2 Data types
- [ ] BYTE (8-bit) — Missing (all numbers float64; `readbyte` file I/O only)
- [ ] SHORT (16-bit) — Missing (`readshort`/`writeshort` file I/O only)
- [ ] INT (32-bit) — **Partial:** `As Int` truncates but stores float64
- [ ] LONG (64-bit) — Missing
- [ ] FLOAT (32-bit) — **Partial:** `As Float` accepted; stored as float64
- [x] DOUBLE (64-bit) — all numbers are float64 (value.KindNum)
- [x] STRING — value.KindStr, `$` suffix accepted
- [ ] BOOL — **Partial:** `As Bool` coerces to 0/1; `True`/`False`; no distinct kind
- [ ] VECTOR2 — **Partial:** generic comma vector literal (value.KindVec); no typed Vec2
- [ ] VECTOR3 — **Partial:** generic KindVec literal; no typed Vec3 or math ops
- [ ] VECTOR4 — **Partial:** generic KindVec literal; no typed Vec4
- [ ] COLOR (RGBA) — Missing (colors passed as r,g,b args)
- [ ] ENTITY handle type — **Partial:** entities are plain integer ids
- [ ] TEXTURE handle type — **Partial:** plain integer ids
- [ ] SOUND handle type — **Partial:** plain integer ids
- [ ] IMAGE handle type — **Partial:** plain integer ids
- [ ] Typed handles / nullable references — **Partial:** `Null` and struct refs; handles untyped ints

#### 2.3 Modern language features
- [x] Interpreter mode — `moon run`, internal/interp tree-walker
- [x] Compiler mode — `moon compile` / `moon transpile` (internal/transpile to Go)
- [ ] Bytecode virtual machine — **Partial:** interp/fast.go opcodes for pure-arithmetic functions only
- [x] Native compilation — `moon compile` transpiles to Go and builds a native binary
- [x] Runtime errors with source line numbers — interp errors formatted `line %d:%d`
- [x] Debugger integration — `moon run -debug -break`, interp/debug.go step/continue/print
- [x] Hot reload — `reloadscripts` reparses and reloads the running script
- [x] Code completion — internal/lsp completion/hover/definition (`moon lsp`)
- [x] Syntax highlighting — editors/vscode/syntaxes/moonbasic.tmLanguage.json
- [ ] Documentation comments — Missing
- [x] Standard library — interp builtins + ~1800 runtime commands
- [ ] Package manager — Missing
- [ ] Module dependency management — **Partial:** `Import`/`Include` file paths; no versions/registry
- [ ] Unit testing for BASIC code — Missing (`moon smoke` only checks programs run)
- [ ] Benchmarking — Missing
- [ ] Asynchronous functions — Missing (coroutines instead)
- [x] Coroutines / cooperative tasks — `Coroutine`, `Yield`
- [x] Event callbacks — `callback`, `setonclick`/`setonchange`, `netregister`/`onrpc`
- [x] File and network APIs — `openfile`/`readline`/`writeline`, `nethost`/`netsend`

#### 2.4 Standard library
- [x] Math — `sin`, `atan2`, `sqr`, `pow`, `clamp`, `lerp`, `smoothstep`, `dot3d`, `crossx`
- [x] Strings — `len`, `left`, `mid`, `instr`, `replace`, `upper`, `trim`, `format`
- [x] Files — `openfile`, `readfile`, `writefile`, `readline`, `fileexists`, `filesize`, `seekfile`
- [ ] Directories — **Partial:** `listdir`, `filetype`; no create/delete/change dir
- [ ] Date and time — **Partial:** `millisecs` only; no calendar date/time (`gettimeofday` is sky time)
- [x] Random numbers — `rnd`, `rand`, `seedrnd`
- [x] Collections — `createlist`, `createmap`, `tablecreate`, `For Each`
- [x] JSON — `jsonload`, `jsonparse`, `jsonget`/`jsonset`, `jsonsave`, `netsendjson`
- [x] Binary read/write — `readbyte`/`readshort`/`readint`/`readfloat` and write equivalents
- [ ] Compression — **Partial:** read-only zip via `openpak`; no compress/decompress commands
- [x] Networking — `nethost`, `netconnect`, `netsend`, `netrecv`
- [ ] System information — **Partial:** `glrenderer`, `glversion`, `monitorwidth`, `getjobworkers`; no OS/CPU/RAM
- [x] Debug printing — `print`, `hudprint`, `-debug` mode
- [ ] Memory utilities — **Partial:** `createbank`/`creatememblock`, `banksize`; no peek/poke
- Also: `yamlload`/`yamlsave`, msgpack `packsave`/`packload`, `clipboard`/`setclipboard`.

### 3. 2D graphics engine

#### 3.1 Sprites
- [x] LoadImage — `loadimage` (commands_2d.go), plus `createimage`
- [x] DrawImage — `drawimage` (screen-space, no camera offset)
- [x] DrawImageRect — `drawimagerect` (commands_tiles.go)
- [x] CreateSprite — `createsprite`
- [x] PositionSprite — `positionsprite`, `movesprite`, alias `setspriteposition`
- [x] RotateSprite — `rotatesprite` (degrees), alias `setspriterotation`
- [x] ScaleSprite — `scalesprite`, alias `setspritescale`
- [x] SpriteColor — `spritecolor` (RGB tint)
- [x] SpriteAlpha — `spritealpha`
- [x] SpriteVisible — `hidesprite` / `showsprite`
- [x] SpriteLayer — `spritelayer` (sorted by layer in graphics2d.go)
- [x] SpriteOrder — `SetSpriteOrder` (within a layer)
- [x] SpriteParent — `SetSpriteParent` (position, rotation, scale), `SpriteWorldX/Y`
- [x] SpriteCollision — `spritesoverlap`, `imagescollide` (AABB only, no rotation/pixel-perfect)
- [x] SpriteAnimation — `setspritesheet`, `animatesprite`, `setspriteframe`
- [x] SpritePivot — `SetSpritePivot`
- [x] SpriteFlip — `flipsprite` (X/Y)
- [x] SpriteBlendMode — `SetSpriteBlend` alpha / add / multiply / screen
- [x] SpriteUV — `SetSpriteRect` (any source rectangle), sheets via `SetSpriteSheet`

#### 3.2 2D camera
- [x] CreateCamera2D — `CreateCamera2D`; camera 0 is the window
- [x] CameraPosition2D — `PositionCamera2D`, `SetCamera2D`, `MoveCamera2D`
- [x] CameraZoom2D — `CameraZoom2D`
- [x] CameraRotation2D — `CameraRotation2D`, `TurnCamera2D`
- [x] CameraFollow (2D) — `CameraTarget2D` (every frame), `CameraFollow2D` (one snap)
- [x] CameraBounds — `CameraBounds2D`
- [x] CameraShake (2D) — `CameraShake2D` (fading)
- [x] CameraSmoothing — `CameraSmooth2D` (frame-rate independent)
- [x] CameraViewport (2D) — `CameraViewport2D`, split screen with several cameras
- [x] CameraDeadZone — `CameraDeadZone2D`
- [x] CameraTarget (2D) — `CameraTarget2D` with offset; `MouseWorldX2D`, `ScreenToWorldX2D`

#### 3.3 Tilemaps
- [x] Tilemap creation — `createtilemap`, `settile`, `gettile`, `drawtilemap`
- [x] Tile atlas — atlas image arg to `drawtilemap`; `drawtile` for single atlas tiles
- [x] Multiple layers — `AddTileLayer`, per-layer visibility, opacity, offset, parallax
- [x] Autotiling — `SetAutoTile`, `PaintAutoTile` (16-tile, 4-neighbour)
- [x] Terrain painting — `PaintAutoTile` from script (mouse painting in `examples/tilemap2d.mb`)
- [x] Collision tiles — `settilesolid`, `spritehittile`
- [x] Animated tiles — `SetTileAnim`; Tiled tile animations load
- [x] Tile selection — `tileat` (world coords to tile id)
- [x] Tile rotation — `SetTileFlip` diagonal bit (90° steps, as in Tiled)
- [x] Tile flipping — `SetTileFlip`; Tiled flip bits load
- [x] Tile metadata — `SetTileData` / `TileData`; Tiled tile properties load
- [ ] Infinite tilemaps — **Partial:** Tiled infinite (chunked) maps load into a fixed grid; no growing at runtime
- [x] Chunk-based rendering — each camera draws only visible cells, one queued draw per layer
- [x] Tilemap importing (e.g. Tiled) — `LoadTiledMap` (.tmx/.tmj, objects, properties), `LoadTileMap` CSV
- [ ] Tilemap editing tools — Missing (script painting only; editor is Milestone C)

#### 3.4 2D lighting
- [x] Point lights — `CreateLight2D`, `SetLight2DCone` spotlights, `ParentLight2D`
- [x] Directional lights — `SetSunLight2D` with directional shadows
- [ ] Normal-mapped sprites — Missing
- [x] Shadow casting — hard shadows per light and from the sun
- [x] Ambient lighting — `SetAmbient2D`
- [ ] Light masks — **Partial:** `SetSpriteLit` 0 opts a sprite out; no per-light layer masks
- [x] Light attenuation — smooth `(1-d²)²` falloff to `SetLight2DRadius`
- [x] 2D glow — `SetSpriteLit` 0 with `SetSpriteBlend "add"`
- [x] Day/night lighting — `SetAmbient2D` + `SetSunLight2D` (`examples/lighting2d.mb`)
- [x] Light occluders — `AddOccluder2D`, `SetSpriteOccluder`, `SetTileOccluders`

#### 3.5 2D animation
- [x] Sprite sheet animation — `setspritesheet` + `animatesprite` (grid frames, FPS)
- [x] Frame-based animation — `animatesprite` frame range, `setspriteframe`
- [ ] Animation timelines — Missing
- [ ] Animation blending — Missing
- [ ] Skeletal 2D animation — Missing
- [ ] Bone hierarchy — Missing
- [ ] Animation events — Missing
- [ ] Animation states — Missing (`animstate` is 3D only)
- [ ] Animation transitions — Missing

#### 3.6 2D particles
- [x] Emitters — `createemitter2d`, `positionemitter2d`, `freeemitter2d` (commands_fx.go)
- [x] Gravity — `particle2dgravity`
- [x] Velocity — `particle2dvelocity`, `particle2dspeed`, `particle2dcone`
- [x] Lifetime — `particle2dlife`, `particle2dduration`, `particle2dloop`
- [ ] Colour gradients — **Partial:** `particle2dcolor` start→end two-stop lerp only
- [ ] Size gradients — **Partial:** `particle2dsize` start→end two-stop lerp only
- [ ] Rotation — Missing (spin simulated but 2D draw ignores it)
- [x] Burst emission — `particle2dburst`, `emit2d`
- [ ] Particle collision — Missing in 2D (ground hits are 3D only)
- [x] Texture particles — `particle2dsprite`
- [ ] GPU particles — Missing for 2D (`creategpuemitter` is 3D only)
- Also: `particle2dwind`, `particle2ddrag`, `particle2dmax`, `particle2drate`.

#### 3.7 2D physics
- [x] Rigid bodies — Chipmunk via internal/phys2d: `physics2d`, `body2d`, `gravity2d`
- [x] Static bodies — `setstatic2d`, or dynamic=0 arg on `createbox2d`/`createcircle2d`
- [x] Dynamic bodies — `createbox2d`/`createcircle2d`, `setmass2d`, `applyforce2d`, `applyimpulse2d`
- [x] Kinematic bodies — `SetKinematic2D`, or dynamic=2 on `CreateBox2D`/`CreateCircle2D`/`CreateCapsule2D`
- [x] Box colliders — `createbox2d`
- [x] Circle colliders — `createcircle2d`
- [x] Polygon colliders — `createpoly2d` (alias `createpolygon2d`)
- [x] Capsule colliders — `CreateCapsule2D`
- [x] Collision layers — `SetCollisionLayer2D`, `SetLayerCollides2D`
- [x] Collision masks — `SetCollisionMask2D`
- [x] Raycasts — `raycast2d`
- [x] Shape casts — `ShapeCast2D`/`BoxCast2D`, `CircleCast2D`, `CastX2D`/`CastNormalX2D`/`CastFraction2D`
- [x] Joints — `createpin2d`, `createspring2d`, `createslide2d`, `createjoint2d`, `freejoint2d`
- [x] Character controllers — `CreateCharacter2D`, `MoveCharacter2D`, `JumpCharacter2D`, `CharacterOnGround2D` (coyote time, jump buffer, max slope, one-way platforms via `SetOneWay2D`)
- [x] Sensors and triggers — `SetSensor2D`, `SensorCount2D`, `SensorBody2D`, `SensorEntered2D`/`SensorExited2D`
- Also: `setccd2d` continuous collision, `velocity2d`/`position2d` getters.

### 4. 3D graphics engine

#### 4.1 Primitive geometry
- [x] CreateCube — `CreateCube` / `CreateBox` (commands.go)
- [x] CreateSphere — `CreateSphere [r, segs, parent]`
- [x] CreateCylinder — `CreateCylinder` (radius, height, segments, caps)
- [x] CreateCone — `CreateCone`
- [x] CreatePlane — `CreatePlane [w, h, parent]`
- [x] CreateCapsule — `CreateCapsule`
- [x] CreateTorus — `CreateTorus major, minor, radial, tubular`
- [x] CreatePyramid — `CreatePyramid`
- [x] CreateWedge — `CreateWedge`
- [x] CreateDisc — `CreateDisk`
- [ ] CreateGrid — Missing (`CreateGrid` is the AI path grid; near: `CreatePlane`)
- [ ] CreateCustomMesh — Missing (no vertex/triangle API; also `CreateTube`, `CreateQuad`)
- [ ] Primitive options: subdivisions, dimensions, orientation, material — **Partial:** size, segments, parent args / no orientation or material args

#### 4.2 Entities
- [x] CreateEntity — `CreateEntity` (alias of `CreatePivot`)
- [x] FreeEntity — `FreeEntity` / `DeleteEntity`
- [x] CopyEntity — `CopyEntity` / `CloneEntity`
- [x] HideEntity / ShowEntity — `HideEntity`, `ShowEntity`
- [x] EntityVisible — `EntityVisible` / `GetEntityVisible`
- [x] EntityParent — `EntityParent`, `GetParent`
- [x] EntityType — `EntityType` (Blitz collision type)
- [x] EntityName — `NameEntity`, `EntityName$`
- [x] EntityID — integer handles; `EntityCount`, `EntityByIndex`
- [x] EntityDistance — `EntityDistance`
- [ ] EntityInView — Missing (near: `CameraProject` + bounds check by hand)
- [x] EntityPick — `LinePick`, `CameraPick`, `RayPick`, `PickedEntity`
- [x] EntityCollision — `EntityCollided`, `Collisions`, `CountCollisions`
- [ ] EntityBounds — **Partial:** `EntityBox`/`EntityRadius` set collision bounds / no mesh AABB getter
- [ ] EntityClass — Missing (primitives tagged internally, no command)
- [ ] EntityTag — Missing (near: `NameEntity`)
- [ ] EntityEnabled — Missing (near: `HideEntity`)

#### 4.3 Transforms
- [x] PositionEntity — `PositionEntity` / `SetPosition`
- [x] RotateEntity — `RotateEntity` / `SetRotation`
- [x] ScaleEntity — `ScaleEntity` / `SetScale`
- [x] MoveEntity — `MoveEntity`
- [x] TurnEntity — `TurnEntity`
- [x] TranslateEntity — `TranslateEntity`
- [x] PointEntity — `PointEntity` / `LookAt`
- [ ] AlignToVector — Missing (near: `VectorYaw`/`VectorPitch` + `RotateEntity`)
- [x] GetEntityPosition — `EntityX/Y/Z` / `GetPositionX..Z`
- [x] GetEntityRotation — `EntityPitch/Yaw/Roll`
- [x] GetEntityScale — `EntityScaleX/Y/Z`
- [ ] GetEntityMatrix — Missing
- [x] EntityWorldPosition — `EntityX(e, 1)` global flag; `EntityWorldX/Z` for shifted origin
- [x] EntityLocalPosition — `EntityX(e)` (local by default)
- [x] Local and world transforms — `TFormPoint`, `TFormVector`, `LocalToWorldX`, `WorldToLocalX`
- [x] Parent-child transforms — `EntityParent`, G3N scene graph
- [ ] Quaternion rotation — **Partial:** used internally (anim nlerp) / no quaternion commands
- [x] Euler rotation — `RotateEntity` pitch/yaw/roll
- [x] Transform interpolation — `TweenPosition`, `TweenRotation`, `MathSmoothDamp`
- [ ] Transform constraints — Missing (near: `AttachToBone`, `FollowPath`)
- [x] Pivot manipulation — `CreatePivot` + parenting

#### 4.4 Cameras
- [x] CreateCamera — `CreateCamera`, `CreateCameraOrtho`, `CreateFreeCamera`
- [x] CameraPosition / CameraRotation — `SetCameraPosition`, `SetCameraRotation`
- [x] CameraZoom — `CameraZoom` / `SetCameraFOV`
- [x] CameraRange — `CameraRange` / `SetCameraRange`
- [x] CameraFog — `CameraFogMode`, `CameraFogRange`, `CameraFogColor`, `CameraFogDensity`
- [x] CameraViewport — `CameraViewport` (commands_classic.go)
- [x] CameraBackground / CameraClear — `CameraClsColor` / `SetCameraClsColor`
- [x] CameraProjection — `CameraProjMode` / `SetCameraProjMode`
- [x] CameraPick — `CameraPick`
- [x] CameraFollow — `CameraFollow` / `SetCameraFollow`
- [x] CameraShake — `CameraShake`, `AddCameraShake`, `ScreenShake`
- [x] CameraTarget — `CameraLookAt` / `PointEntity`
- [x] Perspective cameras — `CreateCamera`
- [x] Orthographic cameras — `CreateCameraOrtho`, `CameraProjMode`
- [x] First-person cameras — `CreateFPSController`, `MouseLook`, `CreateFreeCamera`
- [x] Third-person cameras — `CreateTPSController`, `CameraSpringArm`, `CameraOrbit`
- [ ] Cinematic cameras — **Partial:** `FollowPath`, `Tween*`, `CameraSmoothLook` / no shot sequencer
- [x] Multiple viewports / split-screen — `CameraViewport` per camera (renderSceneCams in postfx.go)
- [ ] Render-to-texture cameras — Missing (near: `SetWindowCamera` into extra windows)
- [ ] Camera stacking — Missing
- Also: `CameraProject`/`CameraUnproject`, `CameraRTS`, `UpdateFreeLook`.

#### 4.5 Lighting
- [x] CreateLight / LightType — `CreateLight kind`, `GetLightType`
- [x] LightColor — `LightColor` / `SetLightColor`, `SetLightKelvin`
- [x] LightRange — `LightRange` / `SetLightRange`, `SetLightAttenuation`
- [x] LightConeAngles — `SetLightCone`
- [x] LightShadows — `SetLightShadow`, `EntityCastShadow`, `EntityReceiveShadow`
- [x] AmbientLight — `AmbientLight` / `SetAmbientLight`, `CreateAmbientLight`
- [x] EnvironmentLight — `SetLightEnvironment`, `SetIBL`, `OutdoorLighting`
- [x] Directional / point / spot lights — `CreateDirectionalLight`, `CreatePointLight`, `CreateSpotLight`
- [x] LightCookie — `SetLightCookie` / `SetSpotlightCookie`
- [ ] LightPriority — Missing
- [x] Area lights — `CreateAreaLight`, `SetAreaLightSize` (max 4, no shadows)
- [x] Forward lighting — G3N forward renderer
- [ ] Clustered lighting — Missing
- [x] Shadow maps — `EnableShadows`, PCF/PCSS/EVSM/MSM filters
- [x] Cascaded shadow maps — `ShadowCascades` (shadow_csm.go)
- [ ] Baked lighting / lightmaps — **Partial:** `SetLightmap`, `SetTerrainLightmap` apply lightmaps / no baker
- [x] Light probes — `CreateLightProbe`, `SetProbeGrid`, `SampleProbe`
- [x] Reflection probes — nearest light probe captures cubemap for PBR (envcube.go)
- [x] Image-based lighting — `SetIBL`, `SetEnvMap`, prefiltered sky cubemap
- [x] Ambient occlusion — `EnableSSAO` (GTAO-style), `SetAOMap`
- Also: contact/screen-space shadows (`EnableContactShadows`), shadow cache/atlas, `CreateThreePoint` rigs.

#### 4.6 Materials
- [ ] CreateMaterial / LoadMaterial — **Partial:** `CreatePBRMaterial` + `SetMaterial` / no material file loader
- [x] EntityMaterial — `EntityMaterial` / `SetMaterial`
- [x] MaterialColor — `EntityColor` / `SetMaterialColor`, `SetAlbedo`
- [x] MaterialMetallic / MaterialRoughness — `SetMetallic`, `SetRoughness`, `SetMetallicRoughnessMap`
- [x] MaterialSpecular — `EntitySpecular`, `EntityShininess`, `SetSpecularMap`
- [x] MaterialEmission — `SetEmissive`, `EntityEmissive`, `SetEmissiveMap`
- [x] MaterialTransparency — `EntityAlpha` / `SetMaterialAlpha`
- [x] MaterialTexture — `EntityTexture`, `SetAlbedoMap`
- [x] MaterialNormalMap — `SetNormalMap`, `EntityNormalMap`
- [ ] MaterialUVScale / MaterialUVOffset — **Partial:** `ScaleTexture` repeat / no UV offset
- [x] MaterialDoubleSided — `EntityFX e, 16`
- [ ] MaterialWireframe — **Partial:** `Wireframe` / `SetWireframe` is global only
- [x] MaterialShader — `EntityShader` / `SetShader`
- [x] PBR — `mbphysical` (pbr_shaders.go), `SetMaterialPBR`
- [x] Unlit — `SetUnlit`, `SetMaterialShading "unlit"`, `EntityFX 1`
- [x] Toon — `SetToon e, bands, rim, spec`
- [x] Transparent — `EntityAlpha`
- [x] Cutout — `EntityCutout`
- [ ] Vertex-coloured — **Partial:** shaders honor `MeshVertexColor` (merged levels) / `EntityFX 2` ignored
- [ ] Retro materials — Missing
- [x] Custom shader materials — `CreateShader`/`LoadShader` + `EntityShader`
- [ ] Material instances — **Partial:** `SetMaterial` shares one material / no per-instance overrides

#### 4.7 Meshes
- [x] Static meshes — `LoadMesh` (glTF/GLB/OBJ/DAE/FBX)
- [ ] Dynamic meshes — Missing (no vertex edit API; terrain sculpt only)
- [ ] Procedural meshes — **Partial:** parametric primitives, `CreateTerrain` / no user vertex API
- [ ] Vertex colours — **Partial:** loaded/merged vertex colours render / no command to set
- [x] Normals — generated by primitives, loaded from glTF
- [ ] Tangents — **Partial:** derivative tangents in shader / no tangent attributes
- [x] UV coordinates — primitives and loaders carry UVs; `ScaleTexture`
- [ ] Multiple UV channels — Missing
- [ ] Multiple material sections — **Partial:** multi-primitive glTF loads / no per-section material command
- [x] Mesh instancing — `CreateInstancedMesh`, `EnableGPUInstances`, `SetInstanceTransform`
- [x] Mesh merging — `BatchInstances` (CPU merge), level merge (level.go)
- [ ] Mesh LOD — **Partial:** `SimplifyMesh`, terrain LOD, tree impostors / no auto mesh LOD switching
- [x] Mesh optimization — `OptimizeMesh`, `SimplifyMesh` (meshopt.go)
- [x] Mesh collision generation — `CreateBodyMesh`, `CreateBodyConvex`
- [x] Mesh import and export — `LoadMesh`; `ExportGLTF`, `ExportMesh`

#### 4.8 3D animation commands
- [x] LoadAnimMesh — `LoadAnimMesh` (glTF)
- [x] Animate — `Animate e, mode, speed`
- [x] SetAnimTime — `SetAnimTime` / `SetAnimationTime`
- [x] SetAnimSpeed — `SetAnimationSpeed` (alias of `Animate`), `SetBlendTreeSpeed`
- [x] ExtractAnimSeq — `ExtractAnimSeq`
- [x] AnimateBlend — `SetAnimBlend` / `BlendAnimation`
- [x] AnimationLength — `AnimLength` / `GetAnimationLength`
- [x] AnimationPlaying — `AnimPlaying`
- [ ] AnimationFinished — **Partial:** `AnimPlaying` returns 0 when stopped / no finished event
- [x] AnimationTransition — `CrossfadeAnim`, `CrossfadeProgress`

### 5. Physics (Jolt)

#### 5.1 3D physics
- [x] Rigid body simulation — Jolt via internal/jolt + internal/phys3d; `CreateBody`
- [x] Static / dynamic / kinematic bodies — `SetMotionType`, `MOTION_STATIC/KINEMATIC/DYNAMIC`
- [x] Box, sphere, capsule, cylinder colliders — `CreateBodyBox/Sphere/Capsule/Cylinder`
- [x] Convex hull colliders — `CreateBodyConvex` / `CreateConvexHull`
- [x] Triangle mesh colliders — `CreateBodyMesh`
- [x] Compound colliders — `CreateBodyCompound`
- [x] Heightfield collision — `CreateBodyHeightfield`
- [x] Collision layers / masks / groups — `SetCollisionLayer`, `SetLayerCollides`
- [ ] Collision callbacks — **Partial:** polled `EntityCollided`/`CountCollisions` / no callback hooks
- [x] Collision filtering — `SetLayerCollides`, `DisableBodyCollision`
- [x] Raycasting — `Raycast`, `RaycastAll`, `GetRayNormalX`, Jolt `CastRay`
- [ ] Sphere casting — Missing (near: `OverlapSphere` overlap, `SetCCD` LinearCast)
- [x] Box casting — `ShapeCast hx,hy,hz, origin, dir` (Jolt CastShape box)
- [ ] Capsule casting — Missing
- [x] Contact events — `Collisions`, `EntityCollided`, contact_listener.cpp
- [x] Trigger volumes — `CreateTrigger`/`CreateSensor`, `CheckTrigger`, `TriggerOverlaps`
- [x] Sleeping bodies — `SleepBody`/`BodySleep`, `WakeBody`, `ActivateBody`
- [x] Continuous collision detection — `SetCCD` / `SetBodyCCD` (LinearCast)
- [x] Fixed joints — `CreateFixedJoint`
- [x] Hinge joints — `CreateHingeJoint`, `SetHingeLimits`, `SetHingeMotor`
- [x] Slider joints — `CreateSliderJoint`
- [x] Distance constraints — `CreateSpringJoint` (Jolt DistanceConstraint, min/max)
- [x] Spring constraints — `CreateSpringJoint` frequency/damping
- [ ] Six-degree-of-freedom constraints — **Partial:** Jolt SixDOF used by `Grab` only / no joint command
- Also: `CreateConeJoint`, `CreateSwingTwistJoint`, Jolt cloth, ropes, `ApplyBuoyancy`, `Explode`.

#### 5.2 Character controllers
- [x] First-person controller — `CreateFPSController`, `UpdateFPS`
- [x] Third-person controller — `CreateTPSController`, `UpdateTPS`
- [x] Platformer controller — `CreatePlatformerController` (multi-jump, coyote time)
- [x] Flying controller — `CreateDroneController`, `CreateSpaceshipController`, `CreateFreeCamera`
- [x] Vehicle controller — `CreateCarController`, `UpdateCar`
- [x] Slope handling — `CreateCharacterController` maxSlopeDeg, steep ground state
- [x] Step climbing — CharacterVirtual `ExtendedUpdate` walk-stairs each step
- [x] Ground detection — `CharacterOnGround`, `GetCharacterGroundState`, `GetCharacterGroundNormal`
- [ ] Air control — **Partial:** `MoveCharacter` works airborne / no air-control tuning
- [x] Jumping — `UpdateFPS`/`UpdatePlatformer` jump args
- [x] Crouching — `UpdateFPS` isCrouch, `SetCharacterShape`
- [x] Moving platforms — CharacterVirtual `GetGroundVelocity` applied (phys3d/jolt_windows.go)
- [x] Character collision events — `GetCharacterContact`
- Also: top-down controller (`CreateTopDownController`).

#### 5.3 Vehicles
- [x] Chassis — `CreateCarController` dynamic box + Jolt VehicleConstraint
- [x] Wheel simulation — Jolt WheeledVehicleController (vehicle.cpp)
- [ ] Suspension — **Partial:** Jolt suspension with preset lengths / not script-tunable
- [x] Steering — `UpdateCar e, steer, throttle, brake`
- [x] Braking — `UpdateCar` brake, handbrake torque presets
- [ ] Engine torque — **Partial:** fixed preset `mMaxTorque` / not script-tunable
- [ ] Gearboxes — **Partial:** Jolt default transmission / no gear commands
- [ ] Differential — **Partial:** preset differentials in vehicle.cpp / not configurable
- [ ] Tire friction — **Partial:** preset wheel friction / no per-wheel command
- [x] Vehicle constraints — Jolt VehicleConstraint (car, motorcycle, tank)
- Also: plane, jet, heli, boat, sub, hovercraft, mech controllers (force-based).

#### 5.4 Physics debugging
- [x] Show collision shapes — `EnablePhysicsDebug`, `DebugDraw "physics"` (box, sphere, capsule, cylinder, mesh wireframe)
- [x] Show contact points — `DebugDraw "contacts"`
- [x] Show collision normals — `DebugDraw "contacts"` (contact normals), `DebugDraw "rays"` (hit normals)
- [x] Show body velocity — `DebugDraw "velocity"`
- [x] Show sleeping bodies — `DebugDraw "physics"` (blue when asleep)
- [x] Show raycasts — `DebugDraw "rays"`, `DebugRayTime`
- [x] Show joints — `DebugDraw "joints"`
- [ ] Physics performance statistics — Missing (only general `StatsFPS`/`StatsDraws`)

#### 5.5 2D physics
- [x] Dedicated 2D physics library (not Jolt) — Chipmunk (`Physics2D`, `CreateBox2D`, `CreatePin2D`, `Raycast2D`)

### 6. Terrain and world building

#### 6.1 Terrain
- [x] CreateTerrain — `CreateTerrain`, `CreateProcTerrain`, `CreateTerrainGL`, `CreateTerrainFromHeightmap` (terrain.go)
- [x] LoadTerrain — `LoadHeightmap(file$ [, w, d, hscale])` returns a terrain handle
- [ ] SaveTerrain — **Partial:** `SaveHeightmap` writes generator maps to PNG; sculpted terrain edits can't be saved
- [x] TerrainHeight — `TerrainHeight`, `TerrainHeightWorld`, `TerrainGridHeight`
- [ ] TerrainSize — **Partial:** size set by create args; `HeightmapWidth/Height`, `TerrainCellSize`; no terrain size getter
- [ ] TerrainScale — **Partial:** w/d/hscale only as `CreateTerrain`/`LoadHeightmap` args; no setter
- [x] TerrainDetail — `SetTerrainDetail` (screen-space triangle target on CPU grid)
- [x] TerrainTexture — `SetTerrainTexture`, `SetTerrainLightmap`
- [x] TerrainLayer — `SetTerrainSplat` (sand/grass/rock/snow layers), `SetTerrainBlendMap`
- [x] TerrainSculpt — `SculptTerrain` (terrain_sculpt.go)
- [x] TerrainSmooth — `SmoothTerrain`
- [x] TerrainFlatten — `FlattenTerrain`
- [x] TerrainRaise / TerrainLower — `SculptTerrain` with positive/negative amount; `SetTerrainHeight` per node
- [x] TerrainPaint — `PaintTerrain(t, x, z, radius, layer)`, `SetTerrainPaintArea`
- [x] TerrainMaterial — `ApplyTerrainSplat`, `SetTerrainSplat` (`mbterrain` height/slope splat)
- [x] TerrainCollision — near-chunk collision + `CreateBodyHeightfield`, rebuilt after brushes
- Also: `PickTerrain` mouse ray, `ResetTerrainEdits`, `TerrainSlope`, `BakeTerrainNav`, `SetTerrainEdge`.

#### 6.2 Terrain editing
- [x] Sculpting brushes — `SculptTerrain`/`FlattenTerrain`/`SmoothTerrain` with radius + falloff; demo `examples/terrain_sculpt.mb`
- [x] Flatten brush — `FlattenTerrain(t, x, z, radius, height [, strength, falloff])`
- [x] Smooth brush — `SmoothTerrain`
- [x] Raise/lower brush — `SculptTerrain` ± amount
- [ ] Erosion brush — **Partial:** whole-map `ErodeHeightmap`/`HydraulicErodeHeightmap`; no local erosion brush
- [x] Texture painting — `PaintTerrain` (glTexSubImage2D partial upload)
- [x] Multi-layer materials — 4-layer splat + height blend (`SetTerrainSplat`, `SetTerrainBlendMap`)
- [x] Heightmap import — `LoadHeightmap`, `ImportHeightmap`, `LoadHeightmapData`, `CreateTerrainFromGeoDEM`
- [ ] Heightmap export — **Partial:** `SaveHeightmap` (PNG) for generator handles only, not edited terrains
- [x] Terrain chunking — chunked meshes, `TerrainChunkCount`
- [x] Terrain LOD — `SetTerrainLOD` (step 2/4 bands, `SetWorldMorph` geomorph)
- [x] Terrain streaming — `SetTerrainStreamRadius`, `SetStreamFollow`, horizon ring + vista
- [ ] Terrain holes — Missing

#### 6.3 Vegetation
- [ ] Grass painting — **Partial:** rule-based `TerrainFoliage`; no paint brush
- [ ] Tree painting — **Partial:** rule-based `TerrainTrees`; no paint brush
- [x] Foliage scattering — `TerrainScatter(t, mesh, kind$, density)`, `TerrainProps`
- [x] Procedural vegetation — `TerrainTrees`/`TerrainFoliage` slope/snow/shore rules, noise-driven stands
- [x] Density controls — density arg on `TerrainFoliage`/`TerrainTrees`/`TerrainScatter`; tier scatter scale
- [x] Random rotation — random yaw per planted copy (terrain_scatter.go)
- [x] Random scaling — 0.85–1.15 scale per copy (terrain_scatter.go)
- [ ] Wind animation — **Partial:** `SetWindSway` CPU sway per entity; no foliage wind shader
- [x] Distance culling — quality-tier tree reach, far grass thinning, frustum-culled batches
- [x] Instanced foliage — scatter planted via instance batches (`plantInstances`)

#### 6.4 Procedural worlds
- [x] Random terrain generation — `GenerateHeightmap`, `GenerateHeightmapPreset` (island, alpine, archipelago…)
- [x] Perlin noise — Perlin-style gradient noise + `NoiseFBM` (noise.go)
- [ ] Simplex noise — Missing (Perlin/value noise only)
- [ ] Voronoi — Missing
- [x] Heightmap generation — `GenerateHeightmap` (fbm/ridged/diamond-square), `FilterHeightmap`
- [ ] Procedural caves — Missing (heightfield only)
- [ ] Procedural roads — Missing (`FlattenTerrain` could grade by hand)
- [x] Procedural vegetation — `TerrainTrees`/`TerrainFoliage`/`TerrainProps`
- [ ] Biomes — **Partial:** height/slope splat bands (sand/grass/rock/snow); no biome maps
- [x] Seed-based generation — seed arg to `GenerateHeightmap`/`GenerateHeightmapPreset`
- [x] World streaming — `CreateWorldStream`, `LoadChunk`/`UnloadChunk`, `SetWorldBubble`, floating origin

### 7. Water and environment

#### Water
- [x] CreateWater — `CreateWater`
- [x] Water plane — `SetWaterLevel`, camera-centred LOD grid (`SetWaterFollow`)
- [x] Water material — `WaterMaterial` + `mbwater`; `SetWaterStyle`, `SetWaterColor`
- [x] Waves — Gerstner `SetWaterWaves`/`SetWaterWave`, FFT tile (water_fft.go)
- [x] Reflection — `EnableWaterReflection`, `SetWaterReflection`, `EnableWaterSSR`
- [x] Refraction — `EnableWaterRefraction`, `SetWaterRefraction`
- [ ] Transparency — **Partial:** opaque surface; refraction shows bottom only in shallows (<0.5 m)
- [x] Depth — refraction depth tint, shallow sand, surf line (water_shaders.go)
- [x] Flow — `SetWaterFlow`, `SetWaterCurrent`, `SetWaterSpeed` (DuDv scroll)
- [x] Foam — height foam on Gerstner crests + shore surf (water shader)
- [x] Underwater rendering — `SetUnderwaterFog`, `SetWaterCaustics`
- [x] Shoreline effects — surf line, shore wetness on `mbterrain`
- [ ] Water collision — **Partial:** `WaterHeight` queries + buoy enter detection; no water volume/swim API
- [x] Buoyancy — `CreateBuoy`, `ApplyBuoyancy`, `SetBuoyancyFactor`
- [x] Simple fluid interaction — `BodyWake`/`WakeBody` wake impulses, rain splash wakes
- Also: `CreateWaterSki` controller, `SetWaterWind`, water ambient sound.

#### Environment
- [x] Skyboxes — `CreateSkybox`, `LoadSkybox`, `SetSkybox`
- [x] Procedural sky — `SetSky`, `CreateAtmosphere`, `SetAtmosphereRayleigh/Mie/Turbidity`
- [x] Sky gradients — `SetSkyGradient`, `SetSkyColor`, `SetSkyPreset`
- [x] Sun and moon — `SetSunDirection`; moon disc + stars in skybox.go at night
- [ ] Day/night cycle — **Partial:** `SetTimeOfDay`/`SetHours` drives sun/sky; no automatic clock
- [x] Atmospheric fog — `EnableFog`, `SetFog`, `CameraFogMode/Range/Density`
- [x] Height fog — `EnableHeightFog`, `SetFogHeight`
- [ ] Volumetric fog — Missing (height fog only; no froxels)
- [x] Weather systems — `SetWeather("clear|rain|snow|fog|storm")`, `SetWeatherTransition`, wetness
- [x] Rain — `SetWeather("rain")`, `SetCameraRain`, splash rings
- [x] Snow — `SetWeather("snow")`, `SetTerrainSnow`
- [x] Wind — `SetWind`, `SetWeatherWind`, `SetWindSway`
- [x] Clouds — `SetClouds`, `CreateVolumetricClouds` (2D noise sheet, not true volumetric)
- [x] Lightning — `StrikeLightning`/`LightningStrike` with delayed thunder
- [x] Environmental particles — weather rain/snow emitters, `EmitterWeatherWind`
- [ ] `CreateSky` / `CreateWeather` / `WeatherType` handle-style API — **Partial:** global `SetSky`/`SetWeather`/`GetWeather`; no weather handles

### 8. Particles and visual effects

#### Commands
- [x] CreateEmitter — `CreateEmitter`, `CreateGPUEmitter`, `CreateEmitter2D`
- [x] EmitParticles — `Emit`, `EmitterBurst`
- [x] ParticleLifetime — `EmitterLife(e, min, max)`
- [x] ParticleVelocity — `EmitterVelocity`, `EmitterSpeed`, `EmitterCone`
- [x] ParticleGravity — `EmitterGravity`
- [x] ParticleSize — `EmitterSize`, `EmitterSizeVar`
- [x] ParticleColor — `EmitterColor` (start/end RGBA)
- [x] ParticleAlpha — `EmitterColor` alpha start/end, `EmitterBlend`
- [ ] ParticleRotation — **Partial:** random spin on flake/cube looks; no rotation setter
- [x] ParticleTexture — `EmitterParticle(e, tex)`
- [x] ParticleBurst — `EmitterBurst`
- [x] ParticleRate — `EmitterRate`
- [ ] ParticleCollision — **Partial:** `EmitterGround` flat plane bounce only (GPU)
- [ ] ParticleAttractor — Missing
- [x] ParticleForce — `EmitterWind`, `EmitterTurbulence` (curl noise), `EmitterDrag`
- [ ] ParticleTrail — **Partial:** `"streak"` shape stretches along velocity; no true trails
- [ ] ParticleRibbon — Missing
- [ ] ParticleSubEmitter — Missing (`ParentEmitter` only attaches to entity)
- Also: `GPUParticleMs` timer query, `EmitterShape` box/sphere/disc, `EmitterCount`.

#### Built-in effects
- [ ] Fire — Missing (no preset; buildable from `Emitter*`)
- [ ] Smoke — Missing (no preset)
- [ ] Sparks — **Partial:** `"spark"` streak look; recipe in `examples/gpu_particles.mb`
- [ ] Explosion — **Partial:** `Explode` is a physics impulse only; no visual effect
- [ ] Dust — **Partial:** recipe in `examples/gpu_particles.mb`, no preset
- [x] Rain — `SetWeather("rain")` streak particles
- [x] Snow — `SetWeather("snow")` flake particles
- [ ] Water splashes — **Partial:** weather rain splash rings/wakes; no general splash effect
- [ ] Magic — **Partial:** recipe in `examples/gpu_particles.mb`, no preset
- [ ] Trails — **Partial:** streak particles only
- [x] Energy beams — `CreateBeam(src, dst, width)` (ez_helpers.go)
- [ ] Debris — **Partial:** CPU `"cube"` tumbling bits; no debris preset
- [ ] Volumetric particles — Missing
- [x] CPU particle system — `CreateEmitter` (particles.go)
- [x] GPU particle system — `CreateGPUEmitter` transform feedback (particles_gpu.go)

### 9. Animation and skeletal systems
- [x] Skeletal animation — `LoadAnimMesh` glTF skins; FBX skin clusters
- [x] Animation clips — `ExtractAnimSeq`, `SetAnimSeq`, `AnimSeqName`
- [x] Animation blending — `SetAnimBlend`, `CrossfadeAnim`
- [x] Blend trees — `CreateBlendTree`, `AddBlendClip`, `SetBlendParam` (1D/2D)
- [ ] State machines — **Partial:** `CreateAnimGraph`, `AnimState`, `AnimGo` / no transitions or conditions
- [x] Animation layers — `SetAnimLayer`, `SetAnimAdditive`
- [ ] Animation masks — Missing (layers affect whole skeleton)
- [ ] Root motion — Missing
- [ ] Bone manipulation — **Partial:** `AttachToBone`, IK / no bone get/set transform commands
- [x] Inverse kinematics — `SetIK` two-bone, `SetIKChain` FABRIK, `SetIKPole`
- [ ] Look-at constraints — **Partial:** `PointEntity` for entities / no bone look-at
- [ ] Morph targets — **Partial:** G3N morph shader chunks included / no weight commands
- [ ] Facial animation — Missing
- [ ] Procedural animation — **Partial:** IK, tweens, `SetWindSway` / no procedural rig system
- [ ] Animation events — Missing
- [ ] Animation retargeting — Missing
- [ ] Animation recording — Missing
- [ ] Timeline editing — Missing
- [ ] Animation import and export — **Partial:** glTF clips import / no FBX clips, no anim export
- [x] `SetAnimation e, "Idle"` / `UpdateAnimations` style API — `SetAnimation` (alias `SetAnimSeq`), updated in `UpdateWorld`

### 10. Audio engine

#### Playback
- [x] LoadSound — `loadsound` (.wav/.ogg)
- [x] PlaySound — `playsound`
- [x] StopSound — `stopsound`
- [ ] PauseSound — Missing (`stopsound` pauses voice but no resume)
- [ ] ResumeSound — Missing
- [x] LoopSound — `loopsound`
- [x] SoundVolume — `setsoundvolume`
- [x] SoundPitch — `setsoundpitch`
- [ ] SoundPan — **Partial:** pan only computed by `emitsound`/`emitsound3d`; no explicit pan
- [ ] SoundSpeed — **Partial:** `setsoundpitch` resamples rate; speed not independent
- [x] SoundPosition — `emitsound3d`, `playsound3d`/`sound3d`
- [x] SoundDistance — `setsoundrange` (min/max distance)
- [x] SoundLoop — `loopsound`
- [ ] SoundFade — Missing
- [ ] SoundChannel — Missing (one voice per sound; no channel handles)
- [ ] SoundPlaybackPosition — Missing

#### 3D audio
- [ ] Positional audio — **Partial:** `emitsound3d` sets vol/pan at start; not updated while playing
- [x] Distance attenuation — audio.DistancePan with `setsoundrange`
- [ ] Doppler effect — Missing
- [x] 3D listeners — `setlistener` (defaults to active camera)
- [ ] Directional audio — Missing (no source cones)
- [ ] Environmental reverb — Missing
- [ ] Occlusion — Missing
- [ ] Obstruction — Missing
- [ ] Audio zones — Missing (`setwaterambientsound` is the only area ambience)
- [ ] Underwater audio — Missing

#### Audio tools
- [ ] Mixer — **Partial:** oto mixes voices; no mixer API or master volume
- [ ] Buses — Missing
- [ ] Effects chains — Missing
- [ ] Reverb — Missing
- [ ] Equalizer — Missing
- [ ] Compressor — Missing
- [ ] Limiter — Missing (only clamp16 sample clipping)
- [ ] Audio visualization — Missing
- [ ] Music streaming — Missing (`loadmusic` decodes whole file to PCM)
- [ ] Sound event editor — Missing
- [ ] Audio import settings — Missing
- Also: procedural clips (engine loop, thunder, footsteps, cloth) in internal/audio; `setmusicvolume`.

### 11. UI engine

#### Runtime UI components
- [x] CreateWindow — ImGui `guibegin`/`guiend`; OS windows via `createwindow`
- [x] CreatePanel — G3N retained `createpanel`; ImGui `guibeginchild`
- [x] CreateButton — G3N retained `createbutton` + `setonclick`; ImGui `guibutton`
- [x] CreateLabel — G3N retained `createlabel`; ImGui `guitext`
- [ ] CreateImage — Missing as UI widget (`createimage` is a 2D image buffer)
- [x] CreateTextBox — G3N `createedit`; ImGui `guiinputtext`
- [ ] CreateTextArea — Missing (single-line inputs only)
- [x] CreateCheckbox — G3N `createcheckbox`; ImGui `guicheckbox`
- [x] CreateRadioButton — ImGui `guiradio` only
- [x] CreateSlider — G3N `createslider`; ImGui `guislider`
- [x] CreateProgressBar — ImGui `guiprogress` only
- [x] CreateDropdown — ImGui `guicombo` ("a|b|c" items) only
- [x] CreateList — ImGui `guiselectable` (inside `guibeginchild`) only
- [x] CreateScrollView — ImGui `guibeginchild` scrolling region
- [x] CreateTabView — ImGui `guibegintabbar`/`guitabitem`
- [x] CreateTooltip — ImGui `guitooltip`
- [x] CreateMenu — ImGui `guibeginmenubar`/`guimenu`/`guimenuitem`, popups `guiopenpopup`
- [ ] CreateInventory — Missing
- [ ] CreateHUD — **Partial:** `hudprint` text overlay only
- Also: `guitree`, `guiheader`, `guicolor`, `guidrag`, `guipickfile`; Gui*/G3N widgets require 3D mode, not Ebiten 2D.

#### UI capabilities
- [ ] Anchoring — Missing (G3N `setwidgetpos` absolute only)
- [ ] Responsive layouts — Missing
- [ ] Resolution scaling — **Partial:** ImGui DisplayFramebufferScale for HiDPI; no UI scale command
- [ ] Flex layouts — Missing
- [ ] Grid layouts — Missing (only `guisameline` row flow)
- [x] Nested containers — ImGui `guibeginchild`, `guitree`, tab bars
- [ ] Themes — Missing (fixed `StyleColorsDark`)
- [ ] Fonts — **Partial:** `loadfont`/`setfont` for 2D text; ImGui uses default font
- [ ] Rich text — Missing
- [ ] Text wrapping — Missing
- [ ] Text alignment — Missing
- [ ] UI animations — Missing
- [x] Mouse interaction — ImGui widgets, G3N `setonclick`/`setonchange`, `wantcapturemouse`
- [ ] Keyboard navigation — **Partial:** ImGui text/Tab input, `wantcapturekeyboard`; nav flags not enabled
- [ ] Gamepad navigation — Missing
- [ ] Localization — Missing
- [ ] Accessibility — Missing
- [ ] Nine-slice scaling — Missing
- [ ] Custom widgets — Missing (only compose ImGui primitives)

### 12. Navigation and AI

#### Navigation
- [x] NavMesh generation — `createnavmesh` (Recast via go-detour), `baketerrainnav`
- [x] NavMesh baking — `bakenavmesh`
- [ ] Navigation regions — Missing (no area types/costs)
- [ ] Navigation obstacles — **Partial:** `addnavobstacle` baked in; requires rebake, no runtime tile cache
- [x] Pathfinding — `setagentdestination`, `getagentpathpointx/y/z`, `agentcountpath`, `updatenav`
- [x] A* — `creategrid`, `setgridwalkable`, `findpath`, `pathx`/`pathy` (commands_agents.go)
- [x] Waypoints — `followpath` (alias `followpolyline`), `tweenposition`
- [ ] Navigation links — Missing (no off-mesh connections)
- [ ] Dynamic obstacle avoidance — **Partial:** `createcrowd`/`crowdupdate` agent separation (crowd.go); no Detour crowd/RVO
- [x] 2D navigation — grid A* `findpath`
- [x] 3D navigation — Detour navmesh agents (commands_nav.go), `setnavmaxslope`

#### AI
- [ ] Finite state machines — Missing (no FSM API)
- [x] Behaviour trees — `createbehavior`, `btsequence`/`btselector`/`btparallel`, `btcondition`/`btaction`, `setbehavior`
- [ ] Utility AI — Missing
- [ ] Steering behaviours — **Partial:** `movetowards`, `smoothdamp`, crowd separation; no seek/flee/wander API
- [ ] Flocking — Missing (crowd separation only)
- [ ] Perception systems — Missing
- [ ] Line-of-sight detection — **Partial:** use `linepick`/`raycast` manually; no LOS command
- [ ] Hearing detection — Missing
- [ ] Vision cones — Missing
- [x] Goal-based navigation (GOAP) — `createplanner`, `addplanaction`, `plan`, `planlength`, `planstep`
- [ ] AI debugging — Missing (no nav/BT debug draw)
- [ ] `CreateAI` / `AIState` / `AIAddWaypoint` / `AIDetectTarget` style API — Missing (closest: `createagent`, `setbehavior`, `followpath`)

### 13. Networking and multiplayer
- [ ] TCP connections — Missing (UDP/ENet only)
- [x] UDP connections — internal/netenet/udp.go default backend
- [ ] WebSocket support — Missing
- [x] Client/server architecture — `nethost`/`hostgame`, `netconnect`/`joingame`
- [ ] Host migration — Missing
- [x] Multiplayer sessions — `netroom`, `netready`/`netallready`, `startgame`, `netplayername`
- [x] Player synchronization — `netreplicate`, `netsnapshot`
- [x] Entity replication — `netreplicate`, `netapplysnapshot`
- [x] Network events — `netevent`/`getnetevent`, `pollnet`, connect/disconnect/receive kinds
- [ ] Network prediction — Missing
- [x] Interpolation — `setnetsmooth` snapshot smoothing (net_session.go)
- [ ] Lag compensation — Missing
- [x] Snapshot synchronization — `netsnapshot`, `netapplysnapshot`, `nettick`
- [ ] Reliable messaging — **Partial:** `netsendreliable` only reliable with `-tags enet`; UDP default isn't
- [ ] Network debugging — **Partial:** `netping`, `netrtt`, `getnetbackend`; no traffic stats/inspector
- [ ] Server-authoritative physics — **Partial:** host replicates transforms; no authoritative physics/rollback
- [ ] Dedicated server builds — Missing (runtime needs a GL window)
- [x] Lobby systems — `netlobby`, `netroom`, `findgames`/`net.find` LAN discovery
- [ ] Basic matchmaking interfaces — **Partial:** LAN `findgames` beacon only; no internet matchmaking
- Also: RPC via `netcall`/`netregister`, `netsendjson`/`netrecvjson`.

### 14. Input and controller system

#### Devices
- [x] Keyboard — `keydown`, `keyhit`, `waitkey`, `flushkeys`
- [x] Mouse — `mousex`, `mousedown`, `mousehit`, `mousez`
- [x] Gamepad — `gamepadaxis`, `gamepadbutton`, `gamepadpresent`
- [x] Joystick — `joystickpresent`, GLFW joystick index via `gamepadaxis`
- [ ] Touchscreen — Missing
- [x] Multiple controllers — pad index argument on `gamepad*` commands
- [ ] Controller hot-plugging — **Partial:** presence polled each call; no connect/disconnect events

#### Keyboard and mouse
- [x] KeyDown — `keydown` (alias `getkeystate`)
- [x] KeyHit — `keyhit`
- [ ] KeyUp — Missing
- [x] MouseDown — `mousedown`
- [x] MouseHit — `mousehit`
- [ ] MouseUp — Missing
- [x] MouseX / MouseY — `mousex`, `mousey`, `mousexspeed`/`mouseyspeed`
- [x] MouseWheel — `mousez` (aliases `mousewheel`, `getmousewheel`)
- [x] MouseMove — `movemouse` (alias `setmouseposition`)
- [x] MouseVisible — `hidepointer`/`showpointer` (aliases `hidecursor`/`showcursor`)
- [x] MouseGrab — `setcursormode` 2 (disabled/captured), `setrawmouse`

#### Gamepad
- [x] JoyX / JoyY / JoyZ — `gamepadaxis(pad, axis)`
- [x] JoyButton — `gamepadbutton(pad, button)`
- [ ] JoyHat — Missing (d-pad readable as buttons only)
- [ ] Controller vibration — Missing
- [x] Controller identification — `gamepadname`
- [x] Analog triggers — `gamepadaxis` trigger axes from GLFW gamepad state
- [x] Deadzone configuration — `setgamepaddeadzone`/`gamepaddeadzone`
- [ ] Input remapping — **Partial:** `bindaction` keyboard keys only; no gamepad remap

#### Input mapping
- [x] Action maps — `bindaction`, `actiondown`, `actionhit`, `unbindaction`
- [ ] Input profiles — **Partial:** `savebinds`/`loadbinds` JSON; no named profile switching
- [x] Custom bindings — `bindaction` / `unbindaction`
- [ ] Multiple devices — **Partial:** actions bind keyboard keys only
- [ ] Context-sensitive input — Missing (no action contexts/layers)
- [ ] Rebindable controls — **Partial:** `bindaction` + `savebinds`; no rebind-capture helper
- Also: `mouselook`, raw mouse (`setrawmouse`), `wantcapturekeyboard`/`wantcapturemouse` for GUI.

### 15. Rendering and graphics quality

#### Rendering modes
- [x] Compatibility tier (older computers) — `SetGraphicsQuality "low"`; auto tier from GL renderer (quality.go)
- [x] Standard tier (mid-range) — `SetGraphicsQuality "medium"`/`"high"`
- [x] Advanced tier (modern) — `SetGraphicsQuality "ultra"` (MSAA 4x, PCSS, AO)

#### Features
- [x] Forward rendering — G3N forward on GL 3.3
- [ ] Forward+ rendering — Missing
- [ ] Deferred rendering — Missing
- [x] PBR — `mbphysical` metallic-roughness (pbr_shaders.go)
- [x] HDR — RGBA16F scene target and bloom buffers (postfx.go)
- [x] Gamma correction — `SetGammaCorrection`
- [x] Tone mapping — `SetTonemap` reinhard/neutral/aces, `SetExposure`, `SetAutoExposure`
- [x] MSAA — `SetMSAA`
- [x] FXAA — `SetFXAA`
- [ ] TAA — Missing
- [ ] Anisotropic filtering — Missing
- [x] Mipmapping — G3N texture mipmaps by default; prefiltered cubemaps (envcube.go)
- [x] Shadow mapping — `EnableShadows`, `SetShadowFilter`
- [x] Cascaded shadows — `ShadowCascades` (shadow_csm.go)
- [x] SSAO — `EnableSSAO`, `SetSSAORadius`
- [ ] Screen-space reflections — **Partial:** water-only lite SSR (`SetWaterSSR`) / no general SSR
- [x] Bloom — `SetBloom` (half-res chain)
- [ ] Depth of field — Missing
- [ ] Motion blur — Missing
- [x] Colour grading — `SetColorGrade`, `SetLUT` (.cube/PNG/presets)
- [x] Vignette — `SetVignette`
- [ ] Lens effects — **Partial:** `SetChromaticAberration`, `SetFilmGrain` / no lens flare or dirt
- [ ] Volumetric effects — **Partial:** height fog, cloud sheet, atmosphere LUT / no light shafts or volumetric fog
- [x] Custom shaders — `CreateShader`, `LoadShader`, `SetShaderUniform`
- [x] Shader variants — G3N `ShaderDefines` per program (e.g. `MB_REBIND`)
- [ ] Render-to-texture — Missing as user command (internal FBOs for water, extra windows)
- [ ] Post-processing stacks — **Partial:** fixed built-in stack via `EnablePostFX` / no custom passes
- Also: `SetOutline` depth-edge ink, `SetSharpen`, `SetAutoExposure`.

#### Retro rendering
- [ ] Low-resolution rendering — Missing
- [ ] Pixel-perfect rendering — Missing (3D)
- [ ] Vertex snapping — Missing
- [ ] Affine texture mapping — Missing
- [ ] Perspective-correct toggle — Missing
- [ ] Low-poly / flat shading — **Partial:** low segment counts on primitives / no flat-shade mode
- [ ] Fixed colour palettes — Missing (near: `SetLUT` with a custom .cube)
- [ ] Dithering — Missing
- [ ] Colour banding — **Partial:** `SetToon` light bands / no posterize pass
- [ ] Ordered dithering — Missing
- [ ] Texture warping — Missing
- [ ] Nearest-neighbour filtering — Missing for 3D textures
- [x] Retro fog — `CameraFogMode`, `CameraFogRange` linear fog
- [ ] Retro lighting — **Partial:** Phong, `SetUnlit`, `SetToon` / no per-vertex Gouraud mode
- [ ] Configurable vertex precision — Missing

### 16. Assets and import pipeline

#### 16.1 3D formats
- [x] glTF — `LoadMesh`/`LoadAnimMesh` .gltf
- [x] GLB — `LoadMesh`/`LoadAnimMesh` .glb
- [x] OBJ — `LoadMesh` .obj + .mtl
- [x] FBX — fbx.go, fbx_skin.go (mesh + skin clusters)
- [ ] STL — Missing
- [ ] PLY — Missing
- [x] Collada — `LoadMesh` .dae
- [x] Static meshes — `LoadMesh`, `LoadLevel`
- [x] Skeletal meshes — `LoadAnimMesh` glTF skins, FBX skin clusters
- [x] Animation clips — `LoadAnimMesh` clips, `ExtractAnimSeq`, `LoadAnimation`
- [x] Materials — glTF PBR → `mbphysical`, OBJ .mtl
- [x] Embedded textures — GLB embedded images via G3N glTF loader
- [x] Collision meshes — `CreateBodyMesh`, `CreateConvexHull`, `CreateBodyConvex`
- [ ] LOD meshes — Missing (`SimplifyMesh` only, no LOD switching)
- Also: `ExportGLTF`/`ExportMesh` (gltf_export.go).

#### 16.2 2D formats
- [x] PNG — `LoadTexture`, `LoadImage`
- [x] JPG / JPEG — `LoadTexture`, `LoadImage`
- [ ] BMP — Missing (no decoder registered)
- [ ] TGA — Missing (listed in AssetBrowser, no decoder)
- [ ] WebP — Missing
- [ ] SVG — Missing
- [ ] GIF — **Partial:** static decode via image/gif; no animated GIF
- [x] Sprite sheets — `SetSpriteSheet`, `SetSpriteFrame`, `AnimateSprite`
- [ ] Texture atlases — **Partial:** `DrawTileMap` atlas arg; no atlas packer/format
- [ ] Aseprite — Missing
- [ ] PSD — Missing
- Also: DDS/KTX/KTX2 S3TC textures (compressed_tex.go).

#### 16.3 Audio formats
- [x] WAV — `LoadSound`/`LoadMusic` (internal/audio)
- [x] OGG — `LoadSound`/`LoadMusic` vorbis
- [ ] MP3 — Missing
- [ ] FLAC — Missing
- [ ] MIDI — Missing

#### 16.4 Pipeline
- [ ] Automatic importing — Missing (raw files loaded at runtime)
- [ ] Asset validation — Missing
- [ ] Texture compression — **Partial:** loads pre-compressed DDS/KTX; no encoder
- [ ] Texture resizing — Missing
- [x] Mipmap generation — G3N auto-mips on upload; DDS mip chains
- [x] Mesh optimization — `OptimizeMesh` (vertex cache/overdraw)
- [ ] Mesh LOD generation — **Partial:** `SimplifyMesh` quadric reduction; no LOD chain
- [ ] Texture atlasing — Missing
- [ ] Dependency tracking — **Partial:** `moon build` scans quoted paths + included .mb recursively
- [ ] Asset metadata — Missing
- [ ] Import presets — Missing
- [ ] Asset versioning — Missing
- [ ] Asset caching — Missing
- [ ] Incremental importing — Missing
- [ ] Hot reloading — **Partial:** `ReloadScripts`, `ReloadTexture`/`ReloadMesh` (manual); no file watcher
- [ ] Asset preview — **Partial:** IDE AssetBrowser.svelte lists by type with load snippet; no thumbnails
- [ ] Duplicate detection — Missing
- [ ] Missing asset detection — **Partial:** load errors at runtime; `moon build` silently skips missing
- [ ] Reference tracking — Missing

#### 16.5 Runtime formats
- [x] Compressed GPU-ready textures — DDS/KTX S3TC stay compressed on GPU (compressed_tex.go)
- [ ] Binary optimized meshes — Missing (GLB only)
- [ ] Packed animation data — Missing
- [ ] Streaming / decoded audio — **Partial:** WAV/OGG fully decoded to PCM; no streaming
- [x] Serialized scene format — scene format v2 `.scene.json` (`SaveScene`/`LoadScene`, scene_v2.go, docs/SCENES.md): stable uids, tags, materials, bodies, lights, cameras, environment, prefab references with overrides; `SceneSave` runtime dumps still load
- [ ] Compiled materials — Missing
- [ ] Cached compiled shaders — Missing
- [ ] Pack files for release builds — **Partial:** `OpenPak`/`SetPak` read zip paks; `moon build` doesn't create one

### 17. Scenes and prefabs

#### Scenes
- [ ] CreateScene — Missing (single world; `ClearWorld`)
- [x] LoadScene — `LoadScene path$ [, additive]` (v2 `.scene.json`, .mb setup, or `SceneSave` JSON/YAML), `SceneLoad`, `LoadSceneJSON`
- [x] SaveScene — `SaveScene` writes scene format v2 (scene_v2.go); `SceneSave` keeps the runtime dump (velocities, controllers, joints)
- [ ] UnloadScene — **Partial:** `ClearWorld` clears everything; a non-additive `LoadScene` replaces the scene's entities; no unload of one loaded file
- [ ] SwitchScene — **Partial:** `ClearWorld` + `LoadScene` (examples/scenes.mb)
- [x] SceneBackground — `CameraClsColor`, `SetSkybox`
- [x] SceneGravity — `SetGravity`, `SetGravity2D`
- [x] SceneEnvironment — the v2 `environment` block: ambient, clear colour, fog, sky preset / skybox, IBL, sun direction, graphics quality, post (tonemap, exposure, bloom, LUT)
- [ ] SceneRoot / SceneParent / SceneChild — **Partial:** `EntityParent`/`GetEntityParent`; no child enumeration
- [x] SceneInstance — `InstantiatePrefab(path$ [, x, y, z, parent])` builds any scene file as an instance (`SceneInstance` alias)

#### Prefabs
- [x] CreatePrefab — `SavePrefab name$, e` saves the entity and its children with every v2 property
- [x] SavePrefab — `SavePrefab` → prefabs/name.json or any `.json` path (v2, root at the origin)
- [ ] LoadPrefab — **Partial:** `LoadPrefab` is an alias of `InstantiatePrefab` (load and instantiate in one step); no prefab handle
- [x] InstantiatePrefab — `InstantiatePrefab(path$ [, x, y, z, parent])`, `SpawnPrefab(name$)`
- [ ] DestroyPrefab — Missing (`FreeEntity` on instance)
- [x] Prefab overrides — `overrides` keyed by member uid; `SaveScene` writes only what differs from the prefab (scene_v2.go)
- [x] Nested prefabs — prefab rows inside prefab files, any depth, with a cycle guard
- [ ] Prefab inheritance — **Partial:** a variant inherits its base prefab and overrides it; no multi-base or class-style inheritance
- [x] Prefab variants — a prefab file whose root is a prefab instance with overrides (`SavePrefab` of an instance writes one)
- [ ] Prefab updates — **Partial:** instances are references, so prefab file edits reach them on the next load; no live refresh

#### Larger games
- [x] Scene streaming — `CreateWorldStream`, `LoadChunk`/`UnloadChunk` (stream.go)
- [x] Additive scene loading — `LoadScene path$, 1` (v2; uids kept unique) and `SceneSave` dumps add to the current world
- [ ] Persistent objects — Missing
- [ ] Scene transitions — Missing
- [ ] Scene references — Missing
- [x] World partitioning — `LoadLevel` 64 m cells, stream chunks, terrain chunks
- [ ] Scene dependency tracking — Missing
- [ ] Scene templates — Missing
- [ ] Scene validation — Missing

#### Entity-component
- [ ] `AddComponent e, "RigidBody"` style components — **Partial:** `EcsAdd`/`EcsSet` on Flecs ids, separate from scene entities
- [x] Simple entity commands without ECS knowledge — `CreateCube`, `PositionEntity`, `CreateBody`, dot methods

### 18. Moon Studio (IDE / editor)

#### 18.1 Main interface
- [x] Project manager — ProjectManager.svelte start screen: New Project from templates, Open Project, Recent Projects (kept in .ide_settings.json); `moon.json` per project (ide/project.go, docs/STUDIO.md)
- [x] Main menu — TitleBar.svelte File/Run/View menus
- [x] Toolbar — ToolBar.svelte
- [ ] Scene hierarchy — **Partial:** ScenePanel.svelte read-only list of SaveScene JSON; `examples/editor.mb` ImGui list
- [ ] Inspector — **Partial:** only in-game ImGui sliders in `examples/editor.mb`
- [x] Asset browser — AssetBrowser.svelte
- [ ] 2D viewport — Missing
- [ ] 3D viewport — Missing
- [ ] Game viewport — **Partial:** F5 runs game in separate window
- [x] Code editor — Monaco in editor/Editor.svelte
- [x] Console — OutputPanel.svelte (Output + Problems tabs)
- [ ] Debugger — **Partial:** gutter breakpoints + continue/step/quit buttons; no watch/stack panes
- [ ] Profiler — **Partial:** in-game `ShowProfiler` overlay; no IDE panel
- [ ] Animation timeline — Missing
- [ ] Material editor — Missing

#### 18.2 Scene editor
- [ ] Select / move / rotate / scale objects — **Partial:** editor.mb select + position sliders only
- [ ] Duplicate / delete objects — Missing
- [ ] Group / parent objects — Missing
- [ ] Align objects — Missing
- [ ] Snap to grid / surface / vertex / angle — Missing
- [ ] Multi-select — Missing
- [ ] Transform gizmos — Missing
- [ ] Local/world coordinates — Missing
- [ ] Pivot controls — Missing
- [ ] Object locking / hiding — Missing
- [ ] Layer visibility — Missing
- [ ] Object search — Missing

#### 18.3 Viewport controls
- [ ] Blender/Godot-style mouse and W/E/R/F/Delete/Ctrl+D/Ctrl+Z/Ctrl+Y bindings — Missing (no editor viewport)
- [ ] Customizable bindings — Missing in IDE (game-side `LoadBinds`/`SaveBinds` exist)

#### 18.4 Viewport display modes
- [ ] Lit / unlit / wireframe / solid — Missing in editor (runtime `Wireframe` only)
- [ ] Material preview / textured — Missing
- [ ] Normals / UV checker — Missing
- [ ] Collision / physics debug view — Missing in editor (runtime `EnablePhysicsDebug`)
- [ ] Lighting only / shadow view — Missing
- [ ] Overdraw visualization — Missing

#### 18.5 2D editor
- [ ] Sprite placement — Missing
- [ ] Tilemap painting / tile selection — Missing (runtime `SetTile`/`LoadTileMap` only)
- [ ] Sprite animation editing — Missing
- [ ] 2D collision editing — Missing
- [ ] 2D light placement — Missing
- [ ] Layers — Missing
- [ ] Camera preview — Missing
- [ ] Pixel grid / pixel snapping — Missing
- [ ] Onion skinning / frame timeline — Missing

#### 18.6 Inspector
- [ ] Transform, material, physics, animation, entity properties — **Partial:** editor.mb position/colour/shininess sliders only
- [ ] Custom component / script properties — Missing
- [ ] Resource references — Missing
- [ ] Component management — Missing
- [ ] Property search — Missing
- [ ] Multi-object editing — Missing

#### 18.7 Code editor
- [x] Syntax highlighting — editor/moonbasic-lang.ts Monarch grammar
- [x] Autocomplete — completions.ts + LSP `textDocument/completion`
- [x] Command documentation (hover) — hover.ts + LSP hover from command catalog
- [ ] Function signatures — **Partial:** syntax shown in completion detail; no signatureHelp
- [x] Error highlighting — LSP `publishDiagnostics` → Monaco markers + Problems tab
- [x] Go to definition — `registerDefinitionProvider` → LSP definition
- [ ] Find references — Missing
- [x] Code folding — `folding: true` + language folding rules
- [x] Multiple tabs — editorState tabs
- [x] Search and replace — Monaco built-in find/replace
- [x] Snippets — completions.ts game-loop snippets
- [ ] Code formatting — Missing
- [x] Breakpoints — F9 / gutter click → `moon -debug -debugjson -break`; toggles while paused send `b` / `d`; paused line highlighted
- [x] Variable inspection — DebugPanel.svelte Locals / Globals from `@debug` JSON, plus a Watch list re-evaluated with `p expr` at every pause
- [x] Call stack — DebugPanel.svelte Call Stack pane (click a frame to jump); F10 / F11 / Shift+F11 step over / into / out
- [ ] Integrated terminal — Missing
- Also: LSP rename + document symbols (SymbolOutline.svelte), ExamplesGallery, BuildModal.

#### 18.8 Material editor
- [ ] Node-based materials, PBR controls, texture slots, UV controls, preview, instances, presets, custom shaders — Missing (runtime PBR setters only)

#### 18.9 Animation editor
- [ ] Timeline, keyframes, tracks, curves, bone/event tracks, blend preview, playback, state machine editor — Missing

#### 18.10 Quality of life
- [ ] Dockable panels / custom layouts — **Partial:** resizable sidebar/output; no docking
- [ ] Multiple monitors — Missing
- [ ] Command palette — **Partial:** Monaco's built-in F1 palette only
- [ ] Context menus — **Partial:** Monaco editor context menu only
- [x] Keyboard shortcuts — F5/Shift+F5/F7/F9/Ctrl+S/O/N/W (App.svelte, Editor.svelte)
- [ ] Undo/redo history — **Partial:** Monaco text undo only
- [ ] Autosave — Missing
- [ ] Crash recovery — Missing
- [ ] Project backups — Missing
- [x] Workspace preferences — `SaveSettings` → .ide_settings.json (SettingsModal.svelte)
- [ ] Theme customization (light and dark) — **Partial:** 4 dark editor themes; no light theme
- [x] Searchable documentation — CommandRef.svelte search over commands.json
- [ ] Plugin manager — Missing

### 19. Shaders and visual programming

#### Shader system
- [x] Vertex shaders — `CreateShader vert$, frag$` (GLSL 330)
- [x] Fragment shaders — `CreateShader`, `LoadShader`
- [ ] Geometry shaders — **Partial:** engine geometry billboards (`CreateGeomPoints`) / no user stage
- [x] Compute shaders — `CreateComputeShader`, `DispatchCompute` (GL 4.3 optional)
- [x] Shader includes — G3N `#include <...>` chunks in user shaders
- [x] Shader uniforms — `SetShaderUniform`, `CreateUniformBuffer`
- [x] Shader variants — G3N `ShaderDefines` per program spec
- [ ] Shader caching — **Partial:** in-memory G3N programs / no binary disk cache
- [ ] Shader hot reload — Missing
- [x] Shader debugging — `ShaderLog`, `ShaderOK`, `CompileShader` (glslang validate)
- [ ] Shader templates — **Partial:** default vert/frag when args empty (user_shader.go)

#### Visual shader editor
- [ ] Node editor (texture, normal, colour, math, vector, lighting, Fresnel, noise, UV, displacement, transparency, emission, custom nodes) — Missing

#### Visual scripting
- [ ] Node-based visual scripting (event, logic, math, physics, input, animation, scene, audio, custom nodes, tracing) — Missing

### 20. Development and debugging tools

#### Debugging
- [x] Print — `Print`, `HudPrint`
- [x] DebugLog — `DebugLog msg$ [, level$]` → `[debug] file.mb:12: msg` under `-debug` / `MOON_DEBUG=1` / IDE (interp/debug.go)
- [x] Assertions — `Assert cond [, msg$]` stops with `file.mb:12: Assert failed: msg` (also after `Flip`; `Try` catches it); `DebugBreak`
- [x] Breakpoints — `moon prog.mb -debug -break 10,20`; IDE gutter
- [x] Step over / into / out — `moon -debug` commands `n` / `s` / `o` by call depth through Functions and Methods; IDE F10 / F11 / Shift+F11
- [x] Variable inspection — `locals` / `globals` / `p expr` (any expression) in `moon -debug`; IDE Variables + Watch panes
- [x] Call stack — `bt` → `#0 Sq game.mb:6` per frame; `@debug` JSON `stack`; IDE Call Stack pane
- [x] Exception reporting — runtime errors as `line N:col`, shown in IDE Problems
- [ ] Runtime object inspection — **Partial:** `EntityCount`/`EntityByIndex`/`EntityName` queries
- [ ] Live entity inspection — **Partial:** `examples/editor.mb` ImGui scene list + inspector
- [ ] Scene debugging — Missing
- [x] Script debugging — `moon -debug` breakpoints, `DebugBreak`, step over/into/out, call stack, variables, watches; IDE Debug sidebar (docs/LANGUAGE.md#debugging)

#### Profiling
- [x] FPS counter — `StatsFPS`
- [x] Frame-time graph — `ShowProfiler` (120 frames, budget colours)
- [x] CPU profiler — `ShowProfiler`, `StatsScriptMs` / `StatsPhysicsMs` / `StatsRenderMs`
- [x] GPU profiler — `StatsGPUMs` (timestamp queries per frame), `GPUParticleMs`
- [x] Memory profiler / RAM usage — `StatsMemoryMB`, heap and sys in `ShowProfiler`
- [ ] VRAM usage — Missing
- [x] Draw-call counter — `StatsDraws`
- [x] Triangle counter — `StatsTriangles`
- [ ] Shader compilation statistics — **Partial:** `ShaderLog`/`GetShaderLog`, `moon shader` validation
- [ ] Physics profiler — **Partial:** `StatsPhysicsMs` step time; no per-body or broadphase breakdown
- [ ] Audio profiler — Missing
- [ ] Script execution profiler — **Partial:** `StatsScriptMs` per frame; no per-function timing
- [x] GC monitoring — `StatsGCPauseMs`, `StatsAllocKB`, GC count in `ShowProfiler`
- [ ] Resource loading statistics — Missing
- Also: `StatsChunks`, `StatsJobs`, `GPUParticleCount`.

#### Visual debugging
- [x] Bounding boxes — `DebugDraw "bounds"`, `DebugBox`
- [x] Wireframes — `Wireframe`/`SetWireframe`
- [ ] Normals / tangents — **Partial:** `DebugDraw "normals"` draws vertex normals / no tangents
- [x] Collision shapes — `DebugDraw "physics"`, `SetPhysicsDebug`
- [x] Raycasts — `DebugDraw "rays"`
- [x] Physics contacts — `DebugDraw "contacts"`
- [x] Camera frustums — `DebugDraw "frustums"`
- [x] Navigation meshes — `DebugDrawNavMesh`
- [x] Light ranges — `DebugDraw "lights"`
- [x] Audio sources — `DebugDraw "audio"`
- [x] Entity IDs — `DebugDraw "ids"`

#### Automated testing
- [x] Unit tests — go test across internal/* (`*_test.go`)
- [ ] Integration tests — **Partial:** `moon smoke` runs examples hidden; not in CI
- [ ] Rendering tests — **Partial:** `moon smoke` fails on GL errors/crashes; no image diffs
- [x] Physics tests — internal/phys3d/*_test.go, phys_easy_test.go
- [x] Scene loading tests — level_test.go, scene_anim_test.go
- [x] Asset import tests — fbx_test.go, level_test.go (glTF)
- [x] Language tests — internal/parse, internal/interp tests; example command checks
- [ ] Performance regression tests — Missing (no benchmarks)
- [x] Automated builds (CI) — .github/workflows/ci.yml (Linux amd64/arm64, macOS; no Windows)

### 21. Optimization and performance

#### CPU
- [x] Multithreaded job scheduling — `JobSubmit`/`JobWait`/`SetJobWorkers` (jobs.go), `PhysicsThreads`
- [ ] Efficient entity updates — **Partial:** Flecs `EcsProgress` path; scene entities are G3N nodes
- [x] Spatial partitioning — Jolt broadphase, `LoadLevel` cells, terrain chunks
- [x] Object pooling — `CreatePool`, `PoolGet`, `PoolPut`
- [ ] Allocation reduction — Missing (no documented effort)
- [ ] Cache-friendly data — Missing
- [ ] Efficient scene traversal — Missing
- [ ] Background asset loading — **Partial:** terrain vista/horizon built on workers; no async `Load*`
- [ ] Asynchronous resource processing — **Partial:** worker-built terrain, `SetPhysicsAsync`
- [ ] Reduced GC pressure — Missing

#### GPU
- [x] Frustum culling — G3N cullable nodes (`SetCullable` in level.go)
- [ ] Occlusion culling — Missing
- [x] Backface culling — G3N single-sided materials by default
- [ ] LOD — **Partial:** terrain LOD and far-scatter cards only; no mesh LOD
- [x] Instanced rendering — `CreateInstancedMesh`, `SetGPUInstances`, `BatchInstances`
- [x] Batching — `LoadLevel`/`MergeLevel` per-material cell merges
- [ ] Texture atlasing — **Partial:** shadow atlas (`EnableShadowAtlas`) only
- [x] Texture compression — DDS/KTX S3TC uploads (compressed_tex.go)
- [ ] Mipmap streaming — Missing
- [x] Shadow optimization (caching) — `SetShadowCache`/`EnableShadowCache`
- [ ] Shader variant management — Missing
- [ ] Resolution scaling — Missing (tiers size water targets only)
- [ ] Render target pooling — Missing

#### Memory
- [ ] Resource sharing — **Partial:** instancing shares one mesh; no general asset cache
- [ ] Reference tracking — Missing
- [ ] Asset streaming — **Partial:** terrain/world chunk streaming; no texture/mesh streaming
- [ ] Memory budgets — Missing
- [ ] Texture memory management — Missing
- [ ] Mesh memory management — Missing
- [ ] Audio streaming — Missing (full PCM decode)
- [x] Scene unloading — `ClearWorld`, `UnloadChunk`, `FreeEntity`
- [ ] Leak detection — Missing
- [ ] Memory diagnostics — **Partial:** `StatsMemoryMB`, `StatsAllocKB`; no per-asset breakdown

#### Hardware targets
- [x] Automatic quality tier from the GPU — `autoTier` from GL renderer + frame-time step-down (quality.go)
- [ ] Tested on 2016–2018 hardware — Missing (low tier targets Intel HD 520, no measurements)
- [ ] Tested on 2019–2021 hardware — Missing
- [x] Tested on 2022–2024 hardware — Radeon 780M measurements in docs/PARTICLES.md, quality.go
- [ ] Tested on 2025–2026 hardware — Missing

### 22. Graphics API and platform

#### Backends
- [x] OpenGL 3.3 — G3N on GL 3.3 core; 4.x features optional
- [ ] Direct3D 11 — Missing
- [ ] Vulkan — Missing (glslang SPIR-V validation only)
- [ ] Direct3D 12 — Missing
- [ ] Metal — Missing

#### Platform services
- [x] Window management — `CreateWindow`, `SetWindowSize`, `SetWindowPos`, `SetWindowTitle`
- [ ] File dialogs — **Partial:** `GuiPickFile` ImGui file list / no native dialogs
- [x] File system access — `OpenFile`, `ReadFile`, `WriteFile`, `ListDir`, `FileExists`
- [x] Keyboard input — `KeyDown`, `KeyHit`, `BindAction`
- [x] Mouse input — `MouseX`, `MouseHit`, `MouseLook`, `RawMouse`
- [x] Gamepad input — `GamepadAxis`, `GamepadButton`, `SetGamepadDeadzone`
- [x] Clipboard — `Clipboard`, `SetClipboard`
- [x] Application lifecycle — `WindowShouldClose`, `IconifyWindow`, `RestoreWindow`, focus callback
- [ ] High-DPI support — **Partial:** framebuffer-size blits / no DPI scale query
- [ ] Multiple monitors — **Partial:** primary `MonitorWidth/Height`, extra windows / no monitor list
- [x] Fullscreen — `SetFullscreen`
- [x] Window resizing — `SetWindowSize`, framebuffer resize handling
- [x] Platform-specific packaging — `moon build` (`-os` cross builds, DLL copy)

### 23. Build system and export

#### Build
- [x] Build project — `moon build prog.mb -o dist`
- [ ] Debug / release / development builds — **Partial:** `moon build` (interpreter) vs `moon compile` (AOT); no profiles
- [x] Standalone build — `moon compile` native executable via transpile
- [ ] Headless build — Missing (`moon smoke` only hides window)
- [ ] Server build — Missing (`-tags nojolt/enet` only)
- [ ] Optimized build — **Partial:** `moon compile` AOT; no strip/optimization flags
- [ ] Incremental compilation — Missing (Go build cache only)
- [x] Asset packaging — `moon build` copies quoted assets + assets/ + natives
- [ ] Dependency checking — **Partial:** follows included .mb for assets; missing files skipped silently
- [ ] Build caching — Missing
- [ ] Build logs — **Partial:** stdout/IDE Output only
- [ ] Build profiles — Missing

#### Targets
- [x] Windows 10/11 — dev platform; `moon build`/`moon compile` -os windows
- [x] Linux — CI ubuntu amd64/arm64 tests + CLI build; `-os linux`
- [x] macOS — CI macos-14 tests + CLI build; `-os darwin`
- [ ] Android — Missing
- [ ] WebAssembly / WebGPU — Missing (`-os wasm` returns error; README overclaims)
- [ ] Steam Deck (via Linux) — **Partial:** Linux amd64 build; untested on Deck

#### Export features
- [x] Executable generation — `moon compile -o app.exe`
- [x] Runtime packaging — `moon build` bundles interpreter + natives (libc++.dll etc.)
- [x] Asset bundling — `moon build` copies referenced assets into dist/
- [ ] Asset compression — **Partial:** runtime reads zip via `OpenPak`; build doesn't produce it
- [ ] Icon configuration — Missing
- [ ] Application metadata — Missing
- [ ] Version information — Missing
- [ ] Startup configuration — Missing (set in script)
- [ ] Resolution configuration — Missing (set in script via `Graphics3D`)
- [x] Graphics quality presets — `SetGraphicsQuality("low|medium|high|ultra|auto")`
- [ ] Debug symbols — Missing (no flag control)
- [ ] Crash logging — **Partial:** panics recovered and printed to stdout; no log file
- [ ] Update-friendly resource packaging — **Partial:** loose files in dist/ folder
- [x] Runtime separate from the editor — moon.exe vs moonBASICIDE.exe

### 24. Project management
- [ ] Project templates (empty 2D, empty 3D, FPS, TPS, platformer, top-down, racing, RPG, multiplayer) — **Partial:** New Project writes Empty 3D, Empty 2D, First-person, Third-person, Platformer 2D, Top-down 2D (ide/templates.go, each smoke-run); no racing, RPG, or multiplayer template
- [x] Project settings — `moon.json` (name, main, window, quality, scenes, assets, build) with defaults and validation in ide/project.go; ProjectSettingsModal.svelte; F5 runs `main`, Build uses `build.outputDir`
- [x] Asset folders — `assets` folder in `moon.json`, created by every template; copied by `moon build`; AssetBrowser
- [x] Scene organization — `scenes` folder in `moon.json`; New Scene writes an empty version-2 `.scene.json`; Explorer scene icon, double-click launches `moon edit <file> -project <root>`; ScenePanel shows the entity tree
- [ ] Project dependencies — **Partial:** `Include`/`Import` of .mb files only
- [ ] Git integration / version control — Missing
- [ ] Backup management — Missing
- [ ] Project migration — Missing
- [ ] Documentation generator — Missing (commands.json/LSP catalog derived from docs)
- [x] Example projects — examples/*.mb (~95) + ExamplesGallery.svelte
- [ ] Learning tutorials — **Partial:** docs/PLAY.md walkthrough only

### 25. Documentation and community
- [x] Complete command reference — docs/COMMANDS.md (~1700 entries in IDE commands.json)
- [x] Language reference — docs/LANGUAGE.md
- [ ] Engine API reference — **Partial:** docs/ARCHITECTURE.md + topic docs; no Go API docs
- [ ] Beginner tutorials — **Partial:** README quickstart, docs/PLAY.md
- [ ] 2D tutorials — **Partial:** examples only (draw2d, platform2d, tiles)
- [ ] 3D tutorials — **Partial:** docs/PLAY.md walkthrough
- [ ] Physics tutorials — **Partial:** docs/PHYSICS.md reference + examples
- [ ] Animation tutorials — **Partial:** docs/ANIMATION.md reference
- [ ] Graphics tutorials — **Partial:** GRAPHICS/PBR/SHADOWS/POSTFX reference docs
- [x] Example projects — examples/ (~95 programs, CI-checked commands)
- [ ] Video tutorials — Missing
- [ ] Downloadable templates — **Partial:** IDE starter templates only
- [ ] Plugin development docs — Missing
- [ ] Extension API docs — Missing
- [ ] Community showcase — Missing
- [ ] Command docs inside the editor (description, parameters, return, example, related) — **Partial:** hover + CommandRef show syntax/description; no return/example/related

### 26. Plugins and extensions
- [ ] Plugin registration — Missing
- [ ] Custom BASIC commands — **Partial:** user `Function` + `Include`/`Import` libraries; no native registration
- [ ] Custom editor panels — Missing
- [ ] Custom asset importers — Missing
- [ ] Custom exporters — Missing
- [ ] Custom inspectors — Missing
- [ ] Custom components — **Partial:** user Flecs components via `EcsComponent`
- [ ] Custom rendering extensions — **Partial:** user GLSL via `LoadShader`/`EntityShader`
- [ ] Custom editor tools — Missing
- [ ] Plugin dependencies — Missing
- [ ] Plugin versioning — Missing
- [ ] Plugin permissions — Missing
- [ ] Plugin enable/disable — Missing
- [ ] Plugin documentation — Missing
- [ ] Editor plugins separate from runtime plugins — Missing

## Future releases

Designed for, not required for 1.0.

| System | Future capabilities | Today |
| --- | --- | --- |
| Rendering | Global illumination, ray tracing | Light probes with cubemap capture, lightmaps |
| Geometry | Mesh sculpting, procedural modelling | Terrain sculpting only |
| Physics | Cloth, soft bodies, advanced vehicles | Jolt cloth and wheeled/tracked vehicles already ship |
| Animation | Motion matching, advanced IK | Two-bone and FABRIK IK, blend trees |
| Terrain | Large-scale world streaming | `SetPlayer` world bubble, 16 km terrains |
| Water | Advanced fluid simulation | Gerstner and FFT ocean, buoyancy, flow |
| AI | Advanced behaviour trees and navigation | Behaviour trees, GOAP, Detour |
| Networking | Replication and multiplayer services | Hosted rooms with replicated movement |
| Scripting | Native compilation | `moon compile` already ships |
| Editor | Collaborative editing | — |
| Graphics | GPU-driven rendering | GPU instancing, transform-feedback particles |
| Audio | Advanced spatial audio | Positional `EmitSound` |
| Tools | Integrated visual game design tools | — |
