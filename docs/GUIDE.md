# MoonBASIC

This is the language the `moon` runtime runs, and the shape of a game written in it. Syntax in full is [LANGUAGE.md](LANGUAGE.md). Every command is [COMMANDS.md](COMMANDS.md). What is finished is [STATUS.md](STATUS.md).

A program is a `.mb` file. `moon` lexes it, parses it, and walks the tree. The first `Flip` is the moment the window loop starts. Setup runs on the way there. Each later `Flip` presents one frame and comes back.

```powershell
go build -o moon.exe ./cmd/moon
.\moon.exe examples\hello_console.mb
.\moon.exe examples\spinning_cube.mb
.\moon.exe examples\draw2d.mb
```

`moon compile game.mb -o dist\game.exe` writes a standalone binary. `moon build` packs a folder you can ship. Both are [RELEASE.md](RELEASE.md). The IDE is [STUDIO.md](STUDIO.md).

## A program

Names are case-insensitive. A newline or a colon starts the next statement. `;`, `'`, and `//` comment to the end of the line. `End` stops the program.

```basic
; examples/hello_console.mb
Print "moonBASIC console"

Function Square(n)
    Return n * n
End Function

For i = 1 To 5
    Print i, "squared is", Square(i)
Next

If 2 + 2 = 4 Then Print "math works"
End
```

`Print` writes a line. With a window up it also draws HUD text. `Function` / `End Function` defines a function. `For i = 1 To 5` counts `i` through 1, 2, 3, 4, 5, then `Next` closes it. `Then` keeps the rest of that `If` on one line.

## Values

Every number is a float. `1` and `1.0` are the same value. Strings use double quotes, and `""` inside a string is one quote. `True` and `Yes` are `1`. `False`, `No`, and `Null` are `0`. `Pi` is pi.

`0` and `""` are false. Any other value is true.

A name that has never been written reads as `0`. `Strict` at the top of the file turns that into an error until the name is a `Local`, a `Global`, a `Static`, or a parameter.

```basic
Strict
Local hp% = 3.9
Local title$ = 12
```

`hp%` is `3`. The `%` suffix truncates to an integer. `title$` is the string `"12"`. The `$` suffix stringifies. `#` marks a float and leaves the value alone. `yaw#` and `yaw` are the same variable. Prefer a plain name.

`$FFECc8` and `Hex("FFECc8")` are a packed colour. On a colour command they expand to red, green, and blue. Channels are 0–255. If every channel is 1 or less, they are read as 0–1, so `Color 0.5, 0, 0` is mid-red.

`[0, 2, -8]` is three numbers. `Position`, `Scale`, `Rotate`, and wind commands expand it to `x, y, z`.

`+` concatenates when either side is a string. `And`, `Or`, `Xor`, and `Not` are the boolean operators. Comparison is `=` `<>` `<` `>` `<=` `>=`.

## Decisions and loops

```basic
If hp > 0 Then
    Print "up"
ElseIf hp = 0 Then
    Print "down"
Else
    Print "gone"
EndIf

Select n
    Case 1
        Print "one"
    Case 2, 3
        Print "few"
    Case 4 To 9
        Print "more"
    Default
        Print "other"
End Select

While n < 3
    n = n + 1
Wend

Repeat
    n = n + 1
Until n = 5

Do
    n = n + 1
    If n = 8 Then Exit Do
Loop
```

`Wend` and `End While` both close `While`. `Do While cond` tests at the top. `Do` … `Loop Until cond` tests at the bottom. `Exit For`, `Exit While`, `Exit Repeat`, and `Exit Do` leave that loop. A bare `Exit` leaves the nearest one. `Exit Function` returns `0`.

## Functions, types, and data

Inside a function, a new name is local. `Global n` with no value keeps using the existing global. `Static n = 0` keeps `n` across calls; the first call starts at the written value.

```basic
Function Add(a, b = 10)
    Return a + b
End Function

Function Cut(n As Integer) As Integer
    Return n + 0.9
End Function
```

`As` accepts `Integer`, `Float`, `String`, `Bool`, or a type name. `Integer` truncates. A default may use an earlier parameter. `Return a, b` and `a, b = F()` pack and unpack several values. The function still returns one value; that value holds the list.

`@Add` is a function value. `f = Function()` … `End Function` is an anonymous function. Call either with `f()`.

`Type` and `Struct` are the same. `Player(3)` or `New(Player)` builds one. Assignment shares it. `Copy(hero)` is a separate one. Methods use `Self` (or `This`).

```basic
Type Enemy
    Field hp, x, y
    Method Hurt(n)
        Self.hp = Self.hp - n
    End Method
End Type

e1 = New(Enemy)
e1.hp = 4
For e = Each(Enemy)
    e.Hurt(1)
Next
Delete e1
```

`For e = Each(Enemy)` walks instances made with `New`. `Delete` drops one. `Erase name` clears one array, type, map, or function variable. `Erase All` does that for every global of those kinds. Entity ids stay.

`Dim scores(10)` is indexes `0` through `10`. `Dim pads(8) As Pad` is the same Blitz count. `Dim plat As Platform(4)` is indexes `0` through `3`. `Redim` builds a new array and copies the overlapping slots. `scores(i)` and `scores[i]` are the same index.

`CreateList()` grows with `ListAdd`. `CreateMap()` stores by string key, ignoring case. A missing key is `0`.

```basic
lst = CreateList()
ListAdd lst, "sword"
Print ListGet(lst, 0)

m = CreateMap()
m("hp") = 3
Print MapGet(m, "HP")
```

`Const MaxLives = 3` is a compile-time name. `Data 10, 20, "ok"` / `Read a, b, s$` / `Restore` walk one program-wide pointer. `Read coin.x, coin.y` writes fields; `coin` must already be a struct.

`Include "lib.mb"` splices that file in. `Import "lib.mb" As Math` prefixes its functions and types, so you call `Math.Double(2)`. `Namespace Util` … `End Namespace` does the same prefix in this file.

```basic
Try
    ReadText "missing.txt"
Catch err$
    Print err$
End Try
```

`Catch` receives the error text. A failed `Assert cond, "why"` is that kind of error, and it stops the program when nothing catches it. `DebugLog` and the `debug>` prompt are [LANGUAGE.md](LANGUAGE.md#debugging).

`CreateStateMachine()` returns an id. The first `AddState` is the current state. `UpdateState` calls the function named for that state.

```basic
sm = CreateStateMachine()
AddState sm, "idle", "OnIdle"
Function OnIdle()
    Print StateName(sm)
End Function
UpdateState sm
```

## The frame

One program has one display mode.

| | 3D | 2D |
| --- | --- | --- |
| Open | `Graphics3D w, h` | `Graphics2D w, h` (or `Graphics`) |
| Window | G3N, OpenGL 3.3 | Ebiten |
| Objects | meshes, cameras, lights | sprites, tilemaps, `Rect` |
| Step | `UpdateWorld` | `UpdateWorld` |
| Draw | `RenderWorld` | `Cls`, then draw calls |

`Flip` ends the frame. It yields, so it stays in the main loop, outside `Function`. Escape is `KeyDown(1)`, and it stays false until the first `Flip`. Close the window with Escape after that, or with `WindowShouldClose()`.

Angles are degrees everywhere: `Sin`, `Cos`, entity `Turn`, and `PointEntity`. Y is up. Yaw `0` faces +Z, so forward is `x + Sin(yaw) * dist` and `z + Cos(yaw) * dist`.

Handles are integers. `CreateCube()` returns one. `cube.Position(0, 1, 0)` is `PositionEntity cube, 0, 1, 0`, and the setter returns the handle, so calls chain:

```basic
box = CreateCube().Scale(1, 1, 1).Position([0, 1, 0]).Color($FFECc8)
```

Topic names are the same commands: `Math.Sin`, `String.Upper`, `Input.KeyDown`, `Time.Delta`, `Entity.SetPos`, `File.Read`, `Json.Load`, `Sound.Play`, `Net.Host`. `Entity.SetPos(cube, 0, 1, 0)` and `cube.SetPos(0, 1, 0)` both move that cube.

`DeltaTime()` is seconds since the last frame. Move with `Approach(x, target, speed * DeltaTime())` so the speed stays in units per second. `SetTimeScale(0.5)` slows `UpdateWorld` (physics, tweens, path follows) and leaves `DeltaTime` alone. A scale of `0` still steps at full speed.

## A 3D scene

```basic
Graphics3D 960, 600
camera = CreateCamera()
PositionEntity camera, 0, 2, -6
CreateLight()

cube = CreateCube()
PositionEntity cube, 0, 0, 0

While Not KeyDown(1)
    TurnEntity cube, 0, 40 * DeltaTime(), 0
    UpdateWorld
    RenderWorld
    Flip
Wend
```

`Graphics3D` opens the window. `CreateCamera` looks down +Z from the origin, so the sample moves it back to `(0, 2, -6)`. `CreateLight` is a directional light. `TurnEntity` adds degrees of pitch, yaw, roll this frame. `UpdateWorld` steps physics and animation. `RenderWorld` draws. `Flip` shows it.

`examples/spinning_cube.mb` is this loop. `examples/lights.mb` adds fog and shadows. `examples/playground.mb` is the mesh primitives: cube, sphere, cylinder, cone, plane, capsule, wedge.

A mesh from a file is `LoadMesh "hero.glb"`. Paths resolve next to the `.mb` file, then inside `OpenPak` if you opened one. Formats are [ASSETS.md](ASSETS.md).

## A 2D scene

```basic
Graphics2D 960, 540
x# = 40
vx# = 180

While Not KeyDown(1)
    Cls
    x = x + vx * DeltaTime()
    If x < 0 Or x > GraphicsWidth() Then vx = -vx
    Color 240, 200, 80
    Oval x, 200, 48, 48
    Flip
Wend
```

`Cls` clears to `ClsColor`. `Oval`, `Rect`, `Line`, and `Text` draw immediately. Sprites, sheets, cameras, and tilemaps are the rest of the 2D section in [COMMANDS.md](COMMANDS.md). `examples/draw2d.mb` and `examples/platform2d.mb` are the demos.

## A falling ball

A mesh is a picture. A body is what the physics step moves. `CreateRigidBodyBox` and `CreateRigidBodySphere` attach a body to a mesh you already built. The last number is the motion: `0` stays put, `1` falls. `UpdateWorld` advances the simulation. `Flip` presents the frame, and it steps physics too when this frame has not already called `UpdateWorld`. Calling both still steps once. The samples call both so the draw follows the step.

Space adds an upward impulse. `Raycast` asks what sits on a line and stores the hit in `PickedY`. Run `examples/jolt_drop.mb` for the same scene with a HUD.

```basic
Graphics3D 960, 600
CameraClsColor 18, 20, 28
cam = CreateCamera().Position([0, 6, -14]).Rotate(18, 0, 0)
CreateLight()

ground = CreateCube().Scale(8, 0.2, 8).Position([0, 0, 10]).Color(50, 56, 70)
CreateRigidBodyBox ground, 8, 0.2, 8, 0

ball = CreateSphere().Position([0, 8, 10]).Color(80, 190, 255)
CreateRigidBodySphere ball, 1, 1

While Not KeyDown(1)
    If KeyHit(KEY_SPACE) Then ApplyImpulse ball, 0, 6, 0
    UpdateWorld
    Raycast 0, 12, 10, 0, -20, 0
    RenderWorld
    Text 16, 16, "hit y " + Int(PickedY())
    Flip
Wend
```

`PhysicsBackend$()` is `"jolt"` on Windows, Linux amd64/arm64, and macOS ARM. `-tags nojolt` is the sphere-and-box fallback. Joints, cloth, grab, and vehicles are [PHYSICS.md](PHYSICS.md) and [VEHICLES.md](VEHICLES.md).

## Walking

`CreateFPSController` puts a character capsule on a pivot and parents the view to `cam`. Each frame, `GetAxis` turns a pair of keys into `-1`, `0`, or `1`, and `UpdateFPS` applies that wish, the mouse, and jump. Call it before `UpdateWorld`, which is what actually moves the capsule. The mouse is hidden so looking does not fight the cursor. Escape is ignored for the first frames, which is why the sample counts `frames` before `KeyHit`.

The full arena, pillars, and shadows are `examples/fps_controller.mb`. Third person is `examples/tps_controller.mb`. Overhead aim is `examples/topdown_controller.mb`. A side view is `examples/platformer_controller.mb`.

```basic
Graphics3D 1280, 720
HidePointer
CameraClsColor 40, 50, 70
cam = CreateCamera()
CreateLight()

ground = CreateCube().Scale(40, 0.25, 40).Color(45, 80, 60)
CreateRigidBodyBox ground, 40, 0.25, 40, 0

player = CreatePivot().Position([0, 2, 0])
CreateFPSController player, cam, 1.8, 0.4, 6.5, 11.5, 9.5

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And KeyHit(KEY_ESCAPE) Then Exit
    UpdateFPS player, GetAxis(KEY_A, KEY_D), GetAxis(KEY_S, KEY_W), KeyHit(KEY_SPACE), KeyDown(KEY_LEFT_SHIFT), KeyDown(KEY_LEFT_CONTROL), 0.18
    UpdateWorld
    RenderWorld
    Text 20, 20, "grounded " + IsFPSGrounded(player)
    Flip
Wend
```

`UpdateFPS` uses the real frame time, capped at 50 ms. `SetTimeScale` slows physics and tweens, and it does not slow this controller.

## A named key and a save

`BindAction` gives a key a name you can check later. Binding `"jump"` twice, once to Space and once to a gamepad button, makes `ActionHit("jump")` true for either. `ActionDown` stays true while the key is held. `ActionHit` is true only on the frame it went down.

`SaveSlot` writes `saves/slotN.txt` next to the script. `LoadSlot$` reads it back, or `""` when that file is not there yet. F5 stores the player's place. F9 puts the pivot back. A missing file on F9 leaves the player where they are.

```basic
BindAction "jump", KEY_SPACE
BindAction "quicksave", KEY_F5
BindAction "quickload", KEY_F9

Function Remember(e)
    SaveSlot 1, EntityX(e) + "," + EntityY(e) + "," + EntityZ(e)
End Function
```

Inside the frame, after `UpdateFPS`:

```basic
If ActionHit("jump") Then UpdateFPS player, 0, 0, 1, 0, 0, 0.18
If ActionHit("quicksave") Then Remember(player)
If ActionHit("quickload") Then
    slot$ = LoadSlot$(1)
EndIf
```

`ReadText` and `WriteText` are a whole file. `TableCreate` keeps rows of strings and `TableSave` writes them as JSON. `Try` / `Catch err$` receives the text when a read fails.

## A door that eases open

`TweenPosition` records a start and a goal and moves the mesh over `seconds` during `UpdateWorld`. The ease is `linear`, `in`, `out`, `inout`, `smooth`, or `smoother`. `TweenRotation` and `TweenColor` use the same clock. Press E once. A second press starts another tween from wherever the door is.

```basic
door = CreateCube().Scale(1.2, 2.4, 0.15).Position([2, 1.2, 6]).Color(120, 80, 50)
open = 0

While Not KeyDown(1)
    If KeyHit(KEY_E) Then
        If open = 0 Then
            TweenPosition door, 2, 1.2, 4, 0.45, "out"
            open = 1
        Else
            TweenPosition door, 2, 1.2, 6, 0.45, "in"
            open = 0
        EndIf
    EndIf
    UpdateWorld
    RenderWorld
    Flip
Wend
```

`SetTimeScale(0.35)` makes that door, and any falling body, take longer. `DeltaTime()` stays the real gap between frames, so a hand-rolled `x = x + speed * DeltaTime()` does not slow down with it. A scale of `0` is stored and then stepped as full speed.

## A 2D jump

2D uses Ebiten. `Graphics2D` and `Graphics3D` do not share a window. `Physics2D` turns Chipmunk on, and `Gravity2D 0, 900` is pixels per second squared. The sprite and the body are one handle: `CreateCharacter2D player, 24, 48` follows that sprite. `MoveCharacter2D` wants `-1` to `1` every frame, and `UpdateWorld` writes the body back onto the sprite. `Cls` clears the screen shapes. The sprite itself is drawn by the 2D world. Ledges, a one-way platform, and a sensor are `examples/platformer2d_physics.mb`.

```basic
Graphics2D 800, 480
Physics2D
Gravity2D 0, 900
floor = CreateSprite(CreateImage(800, 40))
PositionSprite floor, 400, 460
CreateBox2D floor, 800, 40, 0, 0
player = CreateSprite(CreateImage(24, 48))
PositionSprite player, 80, 360
SpriteColor player, 80, 190, 255
CreateCharacter2D player, 24, 48

While Not KeyDown(1)
    dx = 0
    If KeyDown(KEY_LEFT) Then dx = -1
    If KeyDown(KEY_RIGHT) Then dx = 1
    MoveCharacter2D player, dx
    If KeyHit(KEY_SPACE) Then JumpCharacter2D player
    UpdateWorld
    Cls
    Flip
Wend
```

## Two programs, one room

`Net.Host(1973)` opens a port. `Net.Join` in the same process connects to it, which is how `examples/net_echo.mb` checks the wire without a second machine. `Net.Say` sends a line. `NetRecv` is true when something is waiting, and `NET_RECEIVE` means that something is chat. A real game calls `NetUpdate` every frame after `Graphics3D`. The lobby, ready flags, and who is in the room are [NETWORK.md](NETWORK.md) and `examples/net_lobby.mb`.

```basic
If Not Net.Host(1973) Then End
If Not Net.Join("127.0.0.1", 1973) Then End
Net.Say "hello"
ticks = 0
While ticks < 50
    ticks = ticks + 1
    If NetRecv() And NetEvent() = NET_RECEIVE Then
        Print NetMsg$()
        Exit
    EndIf
Wend
NetClose
```

## What you see in the other samples

Each of these is the same loop: build the world, then `UpdateWorld`, `RenderWorld`, `Flip`. The page next to the sample is the long version.

| Run this | What you are looking at | Read |
| --- | --- | --- |
| `examples/lights.mb` | A sun, fill, fog, and a shadow | [LIGHTING.md](LIGHTING.md), [SHADOWS.md](SHADOWS.md) |
| `examples/pbr.mb` | Metal and roughness on one material | [PBR.md](PBR.md) |
| `examples/postfx.mb` | Exposure and bloom over the whole frame | [POSTFX.md](POSTFX.md) |
| `examples/shader.mb` | A GLSL 330 program on one mesh | [SHADERS.md](SHADERS.md) |
| `examples/meshbuild.mb` | Vertices and triangles written from the script | [MESHES.md](MESHES.md) |
| `examples/outdoor.mb` | Height, grass, trees, the clock, a room, a save | [PLAY.md](PLAY.md), [TERRAIN.md](TERRAIN.md) |
| `examples/largeworld.mb` | Terrain and props that follow the camera | [STREAM.md](STREAM.md) |
| `examples/water.mb` | A reflective surface. Bodies on it float | [WATER.md](WATER.md) |
| `examples/weather.mb` | Rain, snow, fog, and a storm, blended | [WEATHER.md](WEATHER.md) |
| `examples/particles.mb` | A CPU emitter. `examples/gpu_particles.mb` is the large one | [PARTICLES.md](PARTICLES.md) |
| `examples/anim_blend.mb` | Idle, walk, and run sharing one skeleton | [ANIMATION.md](ANIMATION.md) |
| `examples/platformer2d_physics.mb` | A Chipmunk body, a one-way ledge, a sensor | [COMMANDS.md](COMMANDS.md) 2D physics |
| `examples/car.mb` | Wheels, or a force fallback if Jolt vehicles are tagged off | [VEHICLES.md](VEHICLES.md) |
| `examples/input.mb` | Mouse look, a click, and a gamepad axis | |
| `examples/net_lobby.mb` | One host, a ready room, then play | [NETWORK.md](NETWORK.md) |
| `examples/nav.mb` | A baked path and a follower | [NAV.md](NAV.md) |
| `examples/gui_3d.mb` | An ImGui panel on the 3D window | |
| `examples/scene_load.mb` | A `.scene.json` level and `FindEntity` | [SCENES.md](SCENES.md) |
| `examples/debug_draw.mb` | Collider wires, contacts, and `DebugLine` | [DEBUG.md](DEBUG.md) |
| `examples/retro.mb` | A low-resolution palette and vertex snap | [RETRO.md](RETRO.md) |
| `examples/geo.mb` | A lon/lat origin laid onto a heightmap | [GEO.md](GEO.md) |
| `examples/ecs.mb` | A Flecs world beside the meshes | [ECS.md](ECS.md) |

## What to open next

| You want | Open |
| --- | --- |
| Every keyword and type rule | [LANGUAGE.md](LANGUAGE.md) |
| A command and its arguments | [COMMANDS.md](COMMANDS.md) |
| Whether a feature is finished | [STATUS.md](STATUS.md) |
| Clips, root motion, masks, bones | [ANIMATION.md](ANIMATION.md) |
| Paths, steering, and agents | [NAV.md](NAV.md) |
| Emitters, soft particles, trails | [PARTICLES.md](PARTICLES.md) |
| Panels, widgets, and the profiler | [UI.md](UI.md) |
| Sounds, music, buses, and echo | [AUDIO.md](AUDIO.md) |
| The package that implements it | [ARCHITECTURE.md](ARCHITECTURE.md) |
| A running sample | `examples\` next to the matching doc |
