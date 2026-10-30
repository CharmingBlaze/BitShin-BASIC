# Language

moonBASIC here is this runtime's dialect. It keeps the everyday BASIC forms and this engine's commands (`Graphics3D`, `CreateCube`, `HostGame`). It does not implement the Raylib registry (`App.Open`, `Draw.Text`, `Render.*`, `Model.*`).

Type suffixes are optional. The forms are `Local` / `Global` / `Static`, `Type` records, `If` / `Select` / `For` / `While` / `Repeat` / `Do`, `Function` with `Return`, `As` signatures, `@Name` and anonymous functions, `Erase` / `Delete`, `For v = Each(Type)`, and coroutines.

Plain functions also answer under a topic name: `Math.Abs`, `Math.Lerp`, `String.Upper`, `List.Add`, `Map.Get`. `Input.KeyDown`, `Time.Delta`, `Entity.SetPos`, `File.Read`, `Json.Load`, `Sound.Play`, and `Net.Host` are the same commands as `KeyDown`, `DeltaTime`, `PositionEntity`, `ReadFile`, `JSONLoad`, `PlaySound`, and `HostGame`. A handle still uses `cube.SetPos(x, y, z)`.

`$` `#` `%` on a name still mean the same variable, so older Blitz-style scripts keep running.

## Comments

`;` `'` and `//` run to end of line.

## Values

- Numbers: `1`, `3.14`, hex `$FFECc8` (packed RGB integer)
- Strings: `"hello"` (`""` inside a string is one quote)
- Vec3: `[0, 2.2, -8]` — expands to `x, y, z` on Position / Scale / Rotate / wind
- `Hex("FFECc8")` / `$FFECc8` unpack to `r, g, b` on Color commands
- Color channels are **0–255**, or **0–1 if every channel is ≤ 1** (`Color(0.5, 0, 0)` is mid-red)
- Suffixes `$` `#` `%` on names are the same variable (`yaw#` and `yaw`). `#` does not change the value. `%` truncates to an integer. `$` forces a string. Prefer plain names.
- Undefined names are `0`
- `0` and `""` are false; anything else is true
- `True` / `Yes` are 1 (true). `False` / `No` / `Null` are 0 (false, empty)
- `x = Null` stores 0. Compare with `If x = Null Then`. `If Not Null` is the same as `If Not 0` (true)
- `Pi` is a builtin

## Operators

`+ - * / ^ Mod`  
`And Or Xor Not`  
`= <> < > <= >=` (`=<` `=>` `><` also lex)

`+` concatenates if either side is a string.

## Math (degrees)

Angles are **degrees** everywhere, including `Sin` / `Cos` / `Tan` / `ASin` / `ACos` / `ATan` / `ATan2`, `WrapAngle`, `AngleDelta`, `ApproachAngle`, `PointYaw` / `PointPitch`, and entity `DeltaYaw` / `DeltaPitch`. There is no radian mode.

Each function returns one number (Blitz style). Use `NormX`/`NormY` or `RotatedX`/`RotatedY` instead of a vector type.

Frame-rate independent move: `x = Approach(x, target, speed * Time.Delta())`.

Full list: [COMMANDS.md](COMMANDS.md) (Math). Demo: `examples/math.mb`.

## Statements

```basic
x = 1 : y = 2
If x > 0 Then Print "ok"
If x > 0 Then
    Print "block"
ElseIf x = 0 Then
    Print "zero"
Else
    Print "neg"
EndIf

While True
    If Input.KeyDown(KEY_ESCAPE) Then End
    Flip
Wend

For i = 1 To 10 Step 2
    Print i
Next

Repeat
    n = n + 1
Until n = 5

Select n
    Case 1
        Print "one"
    Case 2, 3
        Print "few"
    Default
        Print "other"
End Select

Const MaxLives = 3

Data 10, 20, "ok"
Read a, b, s$
Restore

Dim scores(10)
scores(0) = 3
Print ArraySize(scores)

lst = CreateList()
ListAdd lst, 7
Print ListGet(lst, 0), ListCount(lst)

Function Double(n)
    Return n * 2
End Function

Include "lib.mb"
Import "lib.mb" As Math
End
```

## Do, Exit, Static, Enum, strings

```basic
Enum State
    IDLE
    WALK
End Enum
Print State.IDLE
Print STATE_IDLE

Print $"n={1+1}"
Print $"hp={3.5:.1f}"

x = 0
Do
    x = x + 1
    If x = 3 Then Exit Do
Loop

Function Hits()
    Static n = 0
    n = n + 1
    Return n
End Function
```

`Do` … `Loop` repeats. `Do While cond` … `Loop` tests at the top. `Do` … `Loop Until cond` and `Do` … `Loop While cond` test at the bottom. `Exit Do`, `Exit For`, `Exit While`, and `Exit Repeat` leave that kind of loop. A bare `Exit` leaves the nearest loop. `Exit Function` returns 0.

`Static n = 0` inside a function keeps `n` across calls. The first call starts at the written value. Later calls see the value left last time.

`Enum State` defines `IDLE`, `STATE_IDLE`, and a record named `State`, so `State.IDLE` is `0`. The next name is one higher unless you write `WALK = 4`.

`$"text {expr}"` builds one string. `{expr:.2f}` formats a number (`3.50`). `{name:s}` inserts text.

## Functions, instances, and coroutines

`As` on a parameter or a function result accepts `Integer`, `Float`, `String`, `Bool`, or a `Type` name. `Integer` truncates. A user type must be that type, or `0`.

`@Name` is a function value. `Function()` … `End Function` with no name is an anonymous function. Call either with `f()`.

`New(Enemy)` creates an `Enemy` and remembers it. `For e = Each(Enemy)` walks those instances. `Delete e1` drops that instance and sets the variable to `0`. `Erase a` sets one array, type, map, or function variable to `0`. `Erase All` does that for every global of those kinds. It does not free entity ids.

`Dim scores(10)` is indexes `0` through `10` (Blitz). `Dim plat As Platform(4)` follows the moonBASIC count: indexes `0` through `3`. A type body can list fields as `x, y, z` or as `Field x`. `EndType`, `EndEnum`, `EndIf`, `EndFunction`, and `EndSelect` are one word or two.

`String.Interp("Hi {0}", name)` fills numbered holes. `Format(3.5, "%.1f")` and `String.Format` do the same job as `{expr:.1f}`. `Free.All` is `Erase All`. 

A `Coroutine` block starts when the program starts. `Yield` pauses it until the next `Flip`, or until the program ends. `Coroutine.Wait(seconds)` pauses for that long. `Coroutine.Start(@Fn)` starts a normal function as a coroutine and returns its id (`1` for the first coroutine, including blocks that started on their own).

```basic
Function Cut(n As Integer) As Integer
    Return n + 0.9
End Function

f = Function()
    Return 7
End Function
Print f()
Print Coroutine.Start(@Cut)

Type Enemy
    Field hp
End Type
e1 = New(Enemy)
For e = Each(Enemy)
    Print e.hp
Next
Delete e1
Erase All

Coroutine Tick
    Yield
    Print "after"
End Coroutine
```

## Struct / Type

`Struct` and `Type` are the same. Fields, a constructor, and `.field`. Methods use `Self` (or `This`, or a leading `.x`). Assignment shares the object; `Copy` makes another one.

```basic
Struct Vec
    Field x
    Field y
    Method Length()
        Return Sqr(Self.x * Self.x + Self.y * Self.y)
    End Method
End Struct

v = Vec(3, 4)
Print v.x, v.Length()
w = v
w.x = 0
Print v.x
```

`v = New Vec` is an empty instance. `End Type` / `EndStruct` / `End Method` also work.

`Method Vec.Add(n)` after the type is allowed.

Assignment shares the struct. Changing `w.x` also changes `v`. `Copy(v)` is a separate object.

Several fields can sit on one line: `Field x, y, z`.

## Arrays of types

`Dim pads(8) As Pad` makes a Blitz array (indexes `0` through `8`). Empty slots stay `0` until you write a field or store a value. `pads(i).y = 1` creates a `Pad` in that slot and sets the field. `For p In pads` and `For Each p In pads` visit slots that hold a value and skip the empty ones. Field writes on the loop variable change the array element.

```basic
Type Pad
    Field x, y, z, w, d
End Type
Dim pads(8) As Pad
pads(1).x = 10
pads(1).y = 2
For p In pads
    Print p.x
Next
```

`CreateList()` values work with `For item In list` as well. Numbers in a `For` loop are still `For i = 1 To n`.

## Defaults, locals, and several results

Parameters can have defaults, and a default may use an earlier parameter. Inside a `Function`, a new name is local. `Global n` (with no value) keeps assigning the existing global and does not reset it. `Strict` rejects a bare assignment in a function unless the name is a parameter, a `Local`, or a `Global`.

`Return a, b` and `a, b = 1, 2` pack and unpack a list of values. One function still returns one value; that value may be a vector of several numbers.

```basic
Function Add(a, b = 10)
    Return a + b
End Function

n = 1
Function Bump()
    Global n
    n = n + 1
End Function
```

`Read coin.x, coin.y` stores DATA into fields. `coin` must already be a struct.

## Import / Namespace

`Include "file.mb"` splices the file into this program.

`Import "file.mb"` is the same. `Import "file.mb" As Math` prefixes types and functions: `Math.Double(2)`.

```basic
Namespace Util
    Function Sum(a, b)
        Return a + b
    End Function
End Namespace
Print Util.Sum(1, 2)
```

See `examples/struct.mb`.

`Const name = value` is a compile-time name. `Data` / `Read` / `Restore` walk a program-wide DATA pointer.

## Dot methods (entity handles)

`recv.Name(` is a method. `recv.Name` without `(` is still a field (`v.x`).

`cam.Position(0, 2.2, -8)` is `Entity.SetPos(cam, 0, 2.2, -8)`. Mutating commands return the handle, so this works:

```basic
box = CreateCube().Scale(1, 1, 1).Position([0, 1, 0]).Color($FFECc8)
```

Classic `PositionEntity box, 0, 1, 0` is unchanged.

| Method | Command |
| --- | --- |
| `Position` | `SetPosition` / `PositionEntity` |
| `Scale` | `SetScale` / `ScaleEntity` |
| `Rotate` / `Rotation` | `SetRotation` / `RotateEntity` |
| `Color` | `EntityColor` |
| `Alpha` | `EntityAlpha` |
| `Move` / `Translate` / `Turn` | `MoveEntity` / `TranslateEntity` / `TurnEntity` |
| `Point` / `LookAt` | `PointEntity` |
| `Hide` / `Show` | `HideEntity` / `ShowEntity` |
| `Name` / `Parent` / `Texture` | `NameEntity` / `EntityParent` / `EntityTexture` |
| `Shininess` / `Specular` | `EntityShininess` / `EntitySpecular` |
| `Follow` | `CameraFollow` |

Weather: `WEATHER_CLEAR` `WEATHER_RAIN` `WEATHER_SNOW` `WEATHER_FOG` `WEATHER_STORM` (also `0`–`4`). `SetWeather(WEATHER_SNOW)` (1.5s blend). `SetWeatherTransition("storm", 0.85, 10)`.

`CreateList` / `ListAdd` / `ListGet` / `ListSet` / `ListCount` / `ListRemove` are interpreter lists (not `Dim` arrays).

`Replace`, `String`, and `Instr` with a start index are the rest of the classic string set. `ReadFile` / `WriteFile` / `OpenFile` plus `ReadLine` / `WriteLine` and the little-endian `Read*` / `Write*` calls are sequential files. A failed open is `0`. See [COMMANDS.md](COMMANDS.md).

## Editor / language server

`moonsls` (also `moon lsp`) is a stdio LSP for `*.mb` files (language id `moonbasic`). The IDE looks for `moonsls.exe` beside the project, and runs games with `moon.exe`.

```powershell
go build -o moon.exe ./cmd/bs
go build -o moonsls.exe ./cmd/bsls
.\moonsls.exe
```

Capabilities: full-document sync, parse diagnostics, hover, completion, go-to-definition and document symbols for functions in the same file. Cursor/VS Code: `.vscode/settings.json` associates `*.mb`; extension sources are in `editors/vscode`.

`Graphics` / `Graphics2D` open Ebiten. `Graphics3D` opens G3N (**OpenGL 3.3 core** required; 4.x optional). Call one of them before drawing.

`Flip` ends the current frame (see [ARCHITECTURE.md](ARCHITECTURE.md)). Escape / `WindowShouldClose` are ignored until the first `Flip`, so `While Not Input.KeyDown(1)` enters the loop. Still call `Flip` every frame. Demos often wait `frames > 8` before `Input.KeyHit(KEY_ESCAPE)`.
