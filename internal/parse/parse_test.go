package parse

import (
	"testing"

	"bitshinbasic/internal/ast"
)

func TestParseClassicLoop(t *testing.T) {
	src := `
Graphics3D 640, 480
camera = CreateCamera()
While Not KeyDown(1)
    TurnEntity cube, 1, 2, 0
    RenderWorld
    Flip
Wend
End
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.Stmts) < 4 {
		t.Fatalf("expected several statements, got %d", len(prog.Stmts))
	}
}

func TestParseStructMethodImport(t *testing.T) {
	src := `
Import "lib.bb" As Math
Struct Vec
    Field x
    Field y
    Method Length()
        Return Sqr(.x * .x + .y * .y)
    End Method
End Struct
Type Point
    x
    y
End Type
Namespace Util
    Function Add(a, b)
        Return a + b
    End Function
End Namespace
v = Vec(1, 2)
v.x = 3
Print v.Length()
Print Math.Add(1, 2)
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var sawType, sawNS, sawImport bool
	for _, s := range prog.Stmts {
		switch t := s.(type) {
		case *ast.TypeDecl:
			if t.Name == "Vec" && len(t.Fields) == 2 && len(t.Methods) == 1 {
				sawType = true
			}
		case *ast.NamespaceDecl:
			if t.Name == "Util" {
				sawNS = true
			}
		case *ast.ImportStmt:
			if t.As == "Math" {
				sawImport = true
			}
		}
	}
	if !sawType || !sawNS || !sawImport {
		t.Fatalf("missing decls type=%v ns=%v import=%v stmts=%d", sawType, sawNS, sawImport, len(prog.Stmts))
	}
}

func TestParseQoLMethodsVecHex(t *testing.T) {
	src := `
Graphics3D 800, 600
cam = CreateCamera()
cam.Position(0, 2.2, -8)
box = CreateCube().Scale(1, 1, 1).Position([0, 1, 0]).Color($FFECc8)
box.Color(Hex("FF0000"))
PositionEntity box, 1, 2, 3
v.x = 1
Print v.x
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var sawMethod, sawChain, sawClassic, sawField bool
	for _, s := range prog.Stmts {
		switch t := s.(type) {
		case *ast.MethodStmt:
			if t.Method == "Position" && t.Paren && t.Object == "cam" {
				sawMethod = true
			}
		case *ast.CallStmt:
			if t.Name == "PositionEntity" && len(t.Args) == 4 {
				sawClassic = true
			}
		case *ast.AssignStmt:
			if t.Name == "v" && len(t.Fields) == 1 && t.Fields[0] == "x" {
				sawField = true
			}
			if t.Name == "box" {
				if _, ok := t.Value.(*ast.MethodExpr); ok {
					sawChain = true
				}
			}
		}
	}
	if !sawMethod || !sawChain || !sawClassic || !sawField {
		t.Fatalf("qol parse method=%v chain=%v classic=%v field=%v", sawMethod, sawChain, sawClassic, sawField)
	}
}

func TestParseVecAndHexExpr(t *testing.T) {
	prog, err := Parse(`x = [0, 2.2, -8] : c = $FFECc8`)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.Stmts) < 2 {
		t.Fatalf("stmts %d", len(prog.Stmts))
	}
	a0 := prog.Stmts[0].(*ast.AssignStmt)
	if _, ok := a0.Value.(*ast.VecExpr); !ok {
		t.Fatalf("want VecExpr got %T", a0.Value)
	}
	a1 := prog.Stmts[1].(*ast.AssignStmt)
	n, ok := a1.Value.(*ast.NumberExpr)
	if !ok || int(n.Value) != 0xFFECc8 {
		t.Fatalf("hex number got %v %T", a1.Value, a1.Value)
	}
}

func TestParseFunction(t *testing.T) {
	src := `
Function Add(a, b)
    Return a + b
End Function
Print Add(2, 3)
`
	if _, err := Parse(src); err != nil {
		t.Fatal(err)
	}
}
