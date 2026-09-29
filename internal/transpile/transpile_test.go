package transpile

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bitshinbasic/internal/parse"
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
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".bb") || strings.HasPrefix(e.Name(), ".") {
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
	if strings.Contains(goCode, `"dist"`) || strings.Contains(goCode, `"float"`) || strings.Contains(goCode, `"movewish"`) {
		t.Fatalf("math builtins fell through to runtime calls:\n%s", goCode)
	}
	if !strings.Contains(goCode, "_bb_distance2d(") || !strings.Contains(goCode, "_bb_float(") || !strings.Contains(goCode, "_bb_movewish(") {
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
