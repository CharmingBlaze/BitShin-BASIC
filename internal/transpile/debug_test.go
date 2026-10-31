package transpile

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"moonbasic/internal/parse"
)

func TestTranspileDebugLogAssert(t *testing.T) {
	input := `x = 2
DebugLog "x is " + x, "warn"
Assert x = 2, "x must be 2"
DebugBreak
`
	prog, err := parse.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	goCode, err := Transpile(prog, Options{PackageName: "main", IsExecutable: true, SourceFile: "game.mb"})
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "game.go", goCode, parser.AllErrors); err != nil {
		t.Fatalf("invalid Go: %v\n%s", err, goCode)
	}
	for _, want := range []string{`_bb_debuglog("game.mb:2"`, `_bb_assert("game.mb:3"`, "func _bb_assert("} {
		if !strings.Contains(goCode, want) {
			t.Fatalf("generated code missing %q", want)
		}
	}
	if strings.Contains(goCode, `"debugbreak"`) {
		t.Fatal("DebugBreak should compile to nothing, not a runtime call")
	}
}
