package transpile

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"moonbasic/internal/parse"
)

func TestTranspileBasicScript(t *testing.T) {
	input := `
Function AddNumbers(a, b)
    Return a + b
End Function

Local total = 0
For i = 1 To 10
    total = AddNumbers(total, i)
Next

If total > 50 Then
    Print "Total is large: " + total
Else
    Print "Total is small"
EndIf

Graphics3D 800, 600, 0, 2
camera = CreateCamera()
cube = CreateCube()
`

	prog, err := parse.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	goCode, err := Transpile(prog, Options{
		PackageName:  "main",
		IsExecutable: true,
		SourceFile:   "test.bb",
	})
	if err != nil {
		t.Fatalf("Transpile error: %v", err)
	}

	if len(goCode) == 0 {
		t.Fatalf("Transpile output empty")
	}

	// Validate that the output is valid Go syntax
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "test.go", goCode, parser.AllErrors)
	if err != nil {
		t.Fatalf("Generated Go code is invalid syntax: %v\n---\n%s", err, goCode)
	}
}

func TestTranspileExpressionsAndLoops(t *testing.T) {
	input := `
x# = 3.14
y# = Sin(90) + Cos(0) * Sqr(16)
name$ = "Modern " + "Blitz"

While x < 10
    x = x + 1.5
Wend

Repeat
    x = x - 1
Until x <= 0
`

	prog, err := parse.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	goCode, err := Transpile(prog, Options{
		PackageName:  "main",
		IsExecutable: true,
		SourceFile:   "loops.bb",
	})
	if err != nil {
		t.Fatalf("Transpile error: %v", err)
	}

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "loops.go", goCode, parser.AllErrors)
	if err != nil {
		t.Fatalf("Generated Go code is invalid syntax: %v\n---\n%s", err, goCode)
	}
}

func TestTranspileGamePlumbing(t *testing.T) {
	input := `
score = 1
Function Bump()
    score = score + 1
End Function

While Not KeyDown(KEY_ESCAPE)
    Bump()
    Flip
Wend
`
	prog, err := parse.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "game.bb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "game.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{"w.Play(", "w.Running()", "fn_bump(w)", "value.Num(1)", "v_score"} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("generated game code missing %q\n%s", want, goCode)
		}
	}
	if strings.Contains(goCode, "loc_v_score = v_score") {
		t.Fatal("global score was treated as a function local")
	}
}

func TestTranspileTypesSelectData(t *testing.T) {
	input := `
Type Particle
    Field x#, y#, z#
    Field life#
End Type

p.Particle = New Particle
p\x = 10.5
p\life = 100

score = 85
Select score
    Case 100
        Print "Perfect"
    Case 80 To 99
        Print "Great"
    Default
        Print "Good"
End Select

Data 10, 20, 30
Restore
Read a, b, c
`

	prog, err := parse.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	goCode, err := Transpile(prog, Options{
		PackageName:  "main",
		IsExecutable: true,
		SourceFile:   "types_demo.bb",
	})
	if err != nil {
		t.Fatalf("Transpile error: %v", err)
	}

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "types_demo.go", goCode, parser.AllErrors)
	if err != nil {
		t.Fatalf("Generated Go code is invalid syntax: %v\n---\n%s", err, goCode)
	}
}

func TestTranspileMethodsAndChains(t *testing.T) {
	input := `
Struct Vec
    Field x
    Field y
    Method Length()
        Return Sqr(Self.x * Self.x + Self.y * Self.y)
    End Method
End Struct

Namespace Util
    Function Sum(a, b)
        Return a + b
    End Function
End Namespace

v = Vec(3, 4)
Print v.Length()
Print Util.Sum(2, 3)

box = CreateCube().Scale(1, 1, 1).Position([0, 1, 0]).Color(70, 160, 255)
cam = CreateCamera()
cam.Point(box)
`
	prog, err := parse.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "chain.bb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "chain.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{"_bb_dot(", "\"Scale\"", "\"Position\"", "\"Color\"", "value.Vec(", "_bb_methods[", "fn_vec_length", "fn_util_sum", "\"Point\""} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("missing %q\n%s", want, goCode)
		}
	}
}

func TestTranspileStringCommandSuffix(t *testing.T) {
	prog, err := parse.Parse("Print EcsVersion$()\nPrint EcsName$(1)\n")
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "ecs.bb"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(goCode, "ecsversion$") || strings.Contains(goCode, "ecsname$") {
		t.Fatalf("string suffix left on command name:\n%s", goCode)
	}
	if !strings.Contains(goCode, `"ecsversion"`) || !strings.Contains(goCode, `"ecsname"`) {
		t.Fatalf("missing stripped command names:\n%s", goCode)
	}
}

func TestTranspileExamples(t *testing.T) {
	dir := filepath.Join("..", "..", "examples")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".mb") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		n++
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			prog, err := parse.ParseFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			prog, err = parse.ExpandIncludes(prog, dir)
			if err != nil {
				t.Fatal(err)
			}
			goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: name})
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			if _, err := parser.ParseFile(fset, name+".go", goCode, parser.AllErrors); err != nil {
				t.Fatalf("invalid Go: %v\n%s", err, goCode)
			}
		})
	}
	if n == 0 {
		t.Fatal("no examples")
	}
}

func TestTranspileLogicalAndShortCircuit(t *testing.T) {
	prog, err := parse.Parse("If 2 And 4 Then Print \"yes\"\nIf frames > 8 And KeyHit(1) Then End\nIf KeyHit(1) Or frames > 8 Then End\n")
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "logic.bb"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(goCode, "& int64") || strings.Contains(goCode, "| int64") {
		t.Fatalf("and/or still bitwise:\n%s", goCode)
	}
	if !strings.Contains(goCode, "_bb_and(") || !strings.Contains(goCode, "_bb_or(") || !strings.Contains(goCode, "_bb_live") {
		t.Fatalf("missing logical ops or post-flip quit rewrite:\n%s", goCode)
	}
}

func TestTranspileStringBuiltins(t *testing.T) {
	prog, err := parse.Parse("Print Left(\"abcd\", 2)\nPrint Mid(\"abcd\", 2, 2)\nSetBuffer(BackBuffer())\n")
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "str.bb"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(goCode, "_bb_left(") || !strings.Contains(goCode, "_bb_mid(") {
		t.Fatalf("string builtins were not inlined:\n%s", goCode)
	}
	if strings.Contains(goCode, `"setbuffer"`) || strings.Contains(goCode, `"backbuffer"`) {
		t.Fatalf("buffer commands should be no-ops, not runtime calls:\n%s", goCode)
	}
}

func TestTranspileMathBuiltins(t *testing.T) {
	prog, err := parse.Parse("Print Dist(0, 0, 3, 4)\nPrint Float(2)\nPrint Clamp(15, 0, 10)\nPrint MoveWish(90)\nFunction Dist(a, b, c, d)\nReturn a\nEnd Function\n")
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "math.bb"})
	if err != nil {
		t.Fatal(err)
	}
	// `"dist":` is fine — that is the user-function registry, not a runtime call.
	if strings.Contains(goCode, `_bb_call(w, "dist"`) || strings.Contains(goCode, `_bb_call(w, "float"`) || strings.Contains(goCode, `_bb_call(w, "movewish"`) {
		t.Fatalf("math builtins fell through to runtime calls:\n%s", goCode)
	}
	if !strings.Contains(goCode, "_bb_float(") || !strings.Contains(goCode, "_bb_clamp(") || !strings.Contains(goCode, "_bb_movewish(") {
		t.Fatalf("missing math helpers:\n%s", goCode)
	}
	if !strings.Contains(goCode, "fn_dist(") {
		t.Fatalf("user Dist should win over the builtin:\n%s", goCode)
	}
}

func TestPhysicsConstantsInline(t *testing.T) {
	prog, err := parse.Parse("x = DYNAMIC\ny = IN_AIR\n")
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "phys.bb"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(goCode, "value.Num(2)") || !strings.Contains(goCode, "value.Num(3)") {
		t.Fatalf("constants were not inlined:\n%s", goCode)
	}
}

func TestTranspileMatchesInterpreterForms(t *testing.T) {
	src := `
Type Enemy
  Field hp
End Type
Function Boom()
  Print 1
End Function
Coroutine Tick
  Yield
End Coroutine
e = New(Enemy)
For x = Each(Enemy)
  Print x.hp
Next
Delete e
Erase All
Dim a(2)
a(0) = 1
ReDim a(4)
cam = 1
cam.Hide
cam.Hide()
Dim foe(1)
foe(0) = cam
foe(0).Hide()
id = Coroutine.Start(@Boom)
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "forms.mb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "forms.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{
		`_bb_dot(w, v_cam, "Hide", false)`,
		`_bb_dot(w, v_cam, "Hide", true)`,
		`_bb_index(v_foe,`,
		`_bb_liveInst["enemy"]`,
		"_bb_untrack(",
		"_bb_erase_all()",
		"_bb_track(",
		"copy(",
		"_bb_yield()",
		"_bb_spawn(",
		`_bb_coro_start(w, value.Func("Boom"))`,
	} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("missing %q\n%s", want, goCode)
		}
	}
}

func TestTranspileCallsAndLoops(t *testing.T) {
	src := `
Function Outer()
  While 1
    For i = 1 To 2
      Exit While
    Next
  Wend
End Function
f = Function(n)
  Return n + 1
End Function
Print f(2)
Dim a(2)
a(0) = 9
Print a(0)
lst = CreateList()
ListAdd lst, 4
For v In lst
  Print v
Next
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "calls.mb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "calls.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{
		"break _bbL",
		`value.Func("__anon1")`,
		"fn___anon1",
		"_bb_dispatch(",
		"_bb_iter(",
		"_bb_set_index(",
		"_bb_funcs",
	} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("missing %q\n%s", want, goCode)
		}
	}
}

func TestTranspileFieldsAndSignatures(t *testing.T) {
	src := `
Type Spot
  Field x
End Type
Type Line
  Field a
End Type
Function Cut(n As Integer) As Integer
  Return n + 0.9
End Function
ln = New(Line)
ln.a = New(Spot)
ln.a.x = 3
Function Add(a, b = 10)
  Return a + b
End Function
Print Add(2)
Function Keep()
  Global x
  x = x + 1
End Function
n = 1
Function Bump()
  n = n + 1
  Return n
End Function
Type Pad
  Field y
End Type
Dim pads(2) As Pad
pads(1).y = 4
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "sig.mb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "sig.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{
		`_bb_coerce("Integer"`,
		`_bb_as("Integer"`,
		`_bb_set_fields(v_ln,`,
		`"a", "x"`,
		"else {",
		"value.Num(10)",
		"panic(err)",
		"_bb_shadow(&loc_v_n_on, loc_v_n, v_n)",
		"_bb_set_at(",
		"_bb_promote(",
	} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("missing %q\n%s", want, goCode)
		}
	}
	if strings.Contains(goCode, "v_x = value.Num(0)") {
		t.Fatalf("Global x inside Keep reset the variable\n%s", goCode)
	}
}

func TestTranspileStrictAndForStep(t *testing.T) {
	src := `
Strict
x = 1
Function F()
  x = 2
End Function
s = 1
For i = 1 To 3 Step s
  s = 2
Next
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "strict.mb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "strict.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{
		"_bb_strict = true",
		`_bb_undef("x")`,
		"_step =",
		"_end =",
	} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("missing %q\n%s", want, goCode)
		}
	}
	if strings.Contains(goCode, "for ; (_step") {
		t.Fatalf("For loop still freezes the trip\n%s", goCode)
	}
}

func TestTranspileExitDoAndRand(t *testing.T) {
	src := `
x = 0
Do
  x = x + 1
  If x = 2 Then Exit Do
Loop
Print Rand(5, 1)
Print Rnd(0)
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "exitdo.mb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "exitdo.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{
		"break _bbL",
		"_bb_rand(",
		"true)",
		"_bb_rnd(",
	} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("missing %q\n%s", want, goCode)
		}
	}
	if strings.Contains(goCode, "1 + _bb_rng.IntN") {
		t.Fatalf("Rand still calls IntN directly\n%s", goCode)
	}
}

func TestTranspileDataReadsLate(t *testing.T) {
	src := `
x = 10
Data x, 7
Function Hidden()
  Data 4
End Function
Read a, b, c
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "data.mb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "data.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{
		"_bb_data_src = []func() value.Value{",
		"return v_x",
		"value.Num(7)",
		"value.Num(4)",
		"_bb_data_ready",
	} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("missing %q\n%s", want, goCode)
		}
	}
	if strings.Contains(goCode, "_bb_data = append(_bb_data, v_x)") {
		t.Fatalf("Data still captured v_x before assignment\n%s", goCode)
	}
}

func TestTranspileDottedStdlib(t *testing.T) {
	src := `
Print Math.Abs(-3)
Print Math.Sin(90)
Print String.Upper("ab")
Print List.Count(0)
If Input.KeyDown(1) Then Print 1
`
	prog, err := parse.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "dot.mb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "dot.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{
		"math.Abs",
		"_bb_deg",
		"strings.ToUpper",
		"_bb_listcount",
		`_bb_call(w, "input.keydown"`,
	} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("missing %q\n%s", want, goCode)
		}
	}
}
