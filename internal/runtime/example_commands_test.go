package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"bitshinbasic/internal/ast"
	"bitshinbasic/internal/lex"
	"bitshinbasic/internal/parse"
	"bitshinbasic/internal/syntax"
)

func TestSetMassIsRegistered(t *testing.T) {
	m := New(".").commandTable()
	if m["setmass"] == nil {
		t.Fatal("SetMass must be a registered command")
	}
	if m["mass"] == nil {
		t.Fatal("Mass must alias SetMass")
	}
}

func TestClawExampleCommandsExist(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "claw.bb")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	prog, err := parse.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	table := New(".").commandTable()
	known := map[string]bool{
		"print": true, "rnd": true, "rand": true,
		"true": true, "false": true, "yes": true, "no": true, "null": true,
		"sin": true, "cos": true, "abs": true, "int": true, "str": true,
		"backbuffer": true, "frontbuffer": true, "setbuffer": true,
	}
	for k := range table {
		known[k] = true
	}
	for k := range namedKeys {
		known[k] = true
	}
	for k := range syntax.KeyConstants {
		known[k] = true
	}
	user := map[string]bool{}
	var collectFuncs func([]ast.Stmt)
	collectFuncs = func(stmts []ast.Stmt) {
		for _, s := range stmts {
			switch t := s.(type) {
			case *ast.FuncDecl:
				user[lex.IdentKey(t.Name)] = true
			}
		}
	}
	collectFuncs(prog.Stmts)
	var walkStmts func([]ast.Stmt)
	var walkExpr func(ast.Expr)
	walkExpr = func(e ast.Expr) {
		if e == nil {
			return
		}
		switch n := e.(type) {
		case *ast.CallExpr:
			name := lex.IdentKey(n.Name)
			if !user[name] && !known[name] {
				t.Errorf("claw.bb uses unknown command %s", n.Name)
			}
			for _, a := range n.Args {
				walkExpr(a)
			}
		case *ast.IdentExpr:
			name := lex.IdentKey(n.Name)
			if known[name] || user[name] {
				return
			}
			if len(name) > 4 && name[:4] == "key_" && !known[name] {
				t.Errorf("claw.bb uses unknown key %s", n.Name)
			}
		case *ast.UnaryExpr:
			walkExpr(n.X)
		case *ast.BinaryExpr:
			walkExpr(n.Left)
			walkExpr(n.Right)
		}
	}
	walkStmts = func(stmts []ast.Stmt) {
		for _, s := range stmts {
			switch n := s.(type) {
			case *ast.CallStmt:
				name := lex.IdentKey(n.Name)
				if !user[name] && !known[name] {
					t.Errorf("claw.bb uses unknown command %s", n.Name)
				}
				for _, a := range n.Args {
					walkExpr(a)
				}
			case *ast.AssignStmt:
				walkExpr(n.Value)
			case *ast.IfStmt:
				walkExpr(n.Cond)
				walkStmts(n.Then)
				for _, e := range n.ElseIf {
					walkExpr(e.Cond)
					walkStmts(e.Body)
				}
				walkStmts(n.Else)
			case *ast.WhileStmt:
				walkExpr(n.Cond)
				walkStmts(n.Body)
			case *ast.ForStmt:
				walkExpr(n.Start)
				walkExpr(n.End)
				walkExpr(n.Step)
				walkStmts(n.Body)
			}
		}
	}
	walkStmts(prog.Stmts)
}
