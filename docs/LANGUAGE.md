# Language

BitShin BASIC is case-insensitive BASIC, close to Blitz3D.

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

Frame-rate independent move: `x = Approach(x, target, speed * DeltaTime())`.

Full list: [COMMANDS.md](COMMANDS.md) (Math). Demo: `examples/math.bb`.

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
    If KeyDown(KEY_ESCAPE) Then End
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

Include "lib.bb"
Import "lib.bb" As Math
End
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

`Include "file.bb"` splices the file into this program.

`Import "file.bb"` is the same. `Import "file.bb" As Math` prefixes types and functions: `Math.Double(2)`.

```basic
Namespace Util
    Function Sum(a, b)
        Return a + b
    End Function
End Namespace
Print Util.Sum(1, 2)
```

See `examples/struct.bb`.

`Const name = value` is a compile-time name. `Data` / `Read` / `Restore` walk a program-wide DATA pointer.

## Dot methods (entity handles)

`recv.Name(` is a method. `recv.Name` without `(` is still a field (`v.x`).

`cam.Position(0, 2.2, -8)` is `SetPosition(cam, 0, 2.2, -8)`. Mutating commands return the handle, so this works:

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

## Editor / language server

`bsls` (also `bs lsp`) is a stdio LSP for `*.bb` files (language id `bitshinbasic`):

```powershell
go build -o bsls.exe ./cmd/bsls
.\bsls.exe
```

Capabilities: full-document sync, parse diagnostics, hover, completion, go-to-definition and document symbols for functions in the same file. Cursor/VS Code: `.vscode/settings.json` associates `*.bb`; extension sources are in `editors/vscode`.

`Graphics` / `Graphics2D` open Ebiten. `Graphics3D` opens G3N (**OpenGL 3.3 core** required; 4.x optional). Call one of them before drawing.

`Flip` ends the current frame (see [ARCHITECTURE.md](ARCHITECTURE.md)). Escape / `WindowShouldClose` are ignored until the first `Flip`, so `While Not KeyDown(1)` enters the loop. Still call `Flip` every frame. Demos often wait `frames > 8` before `KeyHit(KEY_ESCAPE)`.
