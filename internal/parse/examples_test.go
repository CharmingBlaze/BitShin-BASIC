package parse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamplesParse(t *testing.T) {
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
			if _, err := ParseFile(filepath.Join(dir, name)); err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
		})
	}
	if n == 0 {
		t.Fatal("no examples/*.bb files found")
	}
}
