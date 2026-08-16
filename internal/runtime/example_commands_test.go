package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bitshinbasic/internal/ast"
	"bitshinbasic/internal/lex"
	"bitshinbasic/internal/parse"
	"bitshinbasic/internal/syntax"
)

func TestFindSceneFileFindsExamples(t *testing.T) {
	w := New(".")
	got := w.findSceneFile("level_setup.bb")
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("LoadScene must find examples/level_setup.bb, got %s: %v", got, err)
	}
}

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
	checkExampleCommands(t, filepath.Join("..", "..", "examples", "claw.bb"))
}

func TestAllExampleCommandsExist(t *testing.T) {
	dir := filepath.Join("..", "..", "examples")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".bb") {
			continue
		}
		n++
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			checkExampleCommands(t, filepath.Join(dir, name))
		})
	}
	if n == 0 {
		t.Fatal("no examples/*.bb files found")
	}
}

func exampleKnownCommands() map[string]bool {
	known := map[string]bool{
		"print": true, "rnd": true, "rand": true, "seedrnd": true, "rndseed": true,
		"true": true, "false": true, "yes": true, "no": true, "null": true, "pi": true,
		"sin": true, "cos": true, "tan": true, "asin": true, "acos": true, "atan": true, "atan2": true,
		"sqr": true, "abs": true, "int": true, "floor": true, "ceil": true, "float": true, "sgn": true,
		"min": true, "max": true, "pow": true, "str": true, "hex": true,
		"len": true, "left": true, "right": true, "mid": true, "chr": true, "asc": true,
		"instr": true, "lower": true, "upper": true, "trim": true,
		"millisecs": true,
		"backbuffer": true, "frontbuffer": true, "setbuffer": true,
		"clamp": true, "lerp": true, "invlerp": true, "smoothstep": true,
		"easein": true, "easeout": true, "approach": true, "wrapangle": true,
		"angledelta": true, "approachangle": true,
		"dist": true, "distance2d": true, "distance3d": true, "pointdistance": true,
		"length2d": true, "length3d": true,
		"normx": true, "normy": true, "normx3": true, "normy3": true, "normz3": true,
		"dirx": true, "diry": true, "dirz": true,
		"movepointx": true, "movepointy": true, "movepointz": true,
		"pointyaw": true, "pointpitch": true,
		"dot2d": true, "dot3d": true, "crossx": true, "crossy": true, "crossz": true,
		"reflectx": true, "reflecty": true, "bouncex": true, "bouncey": true,
		"rotatedx": true, "rotatedy": true,
		"createlist": true, "listadd": true, "listget": true, "listset": true,
		"listcount": true, "listremove": true, "arraysize": true,
	}
	table := New(".").commandTable()
	for k := range table {
		known[k] = true
	}
	for k := range namedKeys {
		known[k] = true
	}
	for k := range syntax.KeyConstants {
		known[k] = true
	}
	for k := range syntax.WeatherConstants {
		known[k] = true
	}
	for k := range syntax.NetConstants {
		known[k] = true
	}
	return known
}

type exampleCmdWalker struct {
	t     *testing.T
	file  string
	known map[string]bool
	user  map[string]bool
}

func checkExampleCommands(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	prog, err := parse.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	w := &exampleCmdWalker{
		t:     t,
		file:  filepath.Base(path),
		known: exampleKnownCommands(),
		user:  map[string]bool{},
	}
	w.collect(prog.Stmts)
	w.walkStmts(prog.Stmts)
}

func (w *exampleCmdWalker) collect(stmts []ast.Stmt) {
	for _, s := range stmts {
		switch n := s.(type) {
		case *ast.DimStmt:
			w.user[lex.IdentKey(n.Name)] = true
		case *ast.FuncDecl:
			w.user[lex.IdentKey(n.Name)] = true
			w.collect(n.Body)
		case *ast.TypeDecl:
			w.user[lex.IdentKey(n.Name)] = true
			for _, m := range n.Methods {
				w.user[lex.IdentKey(m.Name)] = true
				w.collect(m.Body)
			}
		case *ast.MethodDecl:
			w.user[lex.IdentKey(n.Name)] = true
			w.collect(n.Body)
		case *ast.NamespaceDecl:
			w.user[lex.IdentKey(n.Name)] = true
			w.collect(n.Stmts)
		case *ast.ImportStmt:
			if n.As != "" {
				w.user[lex.IdentKey(n.As)] = true
			}
		case *ast.IfStmt:
			w.collect(n.Then)
			for _, e := range n.ElseIf {
				w.collect(e.Body)
			}
			w.collect(n.Else)
		case *ast.WhileStmt:
			w.collect(n.Body)
		case *ast.ForStmt:
			w.collect(n.Body)
		case *ast.RepeatStmt:
			w.collect(n.Body)
		case *ast.SelectStmt:
			for _, c := range n.Cases {
				w.collect(c.Body)
			}
			w.collect(n.Default)
		}
	}
}

func (w *exampleCmdWalker) checkCall(name string) {
	key := lex.IdentKey(name)
	if w.user[key] || w.known[key] {
		return
	}
	w.t.Errorf("%s uses unknown command %s", w.file, name)
}

func (w *exampleCmdWalker) walkExpr(e ast.Expr) {
	if e == nil {
		return
	}
	switch n := e.(type) {
	case *ast.CallExpr:
		w.checkCall(n.Name)
		for _, a := range n.Args {
			w.walkExpr(a)
		}
	case *ast.IdentExpr:
		name := lex.IdentKey(n.Name)
		if w.known[name] || w.user[name] {
			return
		}
		if strings.HasPrefix(name, "key_") {
			w.t.Errorf("%s uses unknown key %s", w.file, n.Name)
		}
	case *ast.UnaryExpr:
		w.walkExpr(n.X)
	case *ast.BinaryExpr:
		w.walkExpr(n.Left)
		w.walkExpr(n.Right)
	case *ast.FieldExpr:
		w.walkExpr(n.X)
	case *ast.MethodExpr:
		w.walkExpr(n.X)
		for _, a := range n.Args {
			w.walkExpr(a)
		}
	case *ast.NewExpr:
		for _, a := range n.Args {
			w.walkExpr(a)
		}
	case *ast.IndexExpr:
		for _, a := range n.Index {
			w.walkExpr(a)
		}
	case *ast.VecExpr:
		for _, a := range n.Elems {
			w.walkExpr(a)
		}
	}
}

func (w *exampleCmdWalker) walkStmts(stmts []ast.Stmt) {
	for _, s := range stmts {
		switch n := s.(type) {
		case *ast.CallStmt:
			w.checkCall(n.Name)
			for _, a := range n.Args {
				w.walkExpr(a)
			}
		case *ast.AssignStmt:
			w.walkExpr(n.Value)
			for _, a := range n.Index {
				w.walkExpr(a)
			}
		case *ast.IfStmt:
			w.walkExpr(n.Cond)
			w.walkStmts(n.Then)
			for _, e := range n.ElseIf {
				w.walkExpr(e.Cond)
				w.walkStmts(e.Body)
			}
			w.walkStmts(n.Else)
		case *ast.WhileStmt:
			w.walkExpr(n.Cond)
			w.walkStmts(n.Body)
		case *ast.ForStmt:
			w.walkExpr(n.Start)
			w.walkExpr(n.End)
			w.walkExpr(n.Step)
			w.walkStmts(n.Body)
		case *ast.RepeatStmt:
			w.walkStmts(n.Body)
			w.walkExpr(n.Until)
		case *ast.ReturnStmt:
			w.walkExpr(n.Value)
		case *ast.FuncDecl:
			w.walkStmts(n.Body)
		case *ast.TypeDecl:
			for _, m := range n.Methods {
				w.walkStmts(m.Body)
			}
		case *ast.MethodDecl:
			w.walkStmts(n.Body)
		case *ast.NamespaceDecl:
			w.walkStmts(n.Stmts)
		case *ast.MethodStmt:
			for _, a := range n.Args {
				w.walkExpr(a)
			}
		case *ast.ExprStmt:
			w.walkExpr(n.Value)
		case *ast.SelectStmt:
			w.walkExpr(n.Value)
			for _, c := range n.Cases {
				for _, item := range c.Items {
					w.walkExpr(item.Low)
					w.walkExpr(item.High)
				}
				w.walkStmts(c.Body)
			}
			w.walkStmts(n.Default)
		case *ast.DimStmt:
			for _, a := range n.Sizes {
				w.walkExpr(a)
			}
		case *ast.LocalStmt:
			for _, a := range n.Values {
				w.walkExpr(a)
			}
		case *ast.GlobalStmt:
			for _, a := range n.Values {
				w.walkExpr(a)
			}
		case *ast.ConstStmt:
			for _, a := range n.Values {
				w.walkExpr(a)
			}
		case *ast.EnumStmt:
			for _, a := range n.Values {
				w.walkExpr(a)
			}
		case *ast.DataStmt:
			for _, a := range n.Values {
				w.walkExpr(a)
			}
		}
	}
}
