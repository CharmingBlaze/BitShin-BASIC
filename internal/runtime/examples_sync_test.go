package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The IDE ships each example's source in examples.json. It drifted from the
// files (cloth, pbr, and five more were stale), so the IDE ran old code.
func TestIDEExamplesMatchTheirFiles(t *testing.T) {
	root := filepath.Join("..", "..")
	b, err := os.ReadFile(filepath.Join(root, "ide", "frontend", "src", "lib", "data", "examples.json"))
	if err != nil {
		t.Skip("no IDE example list:", err)
	}
	var items []struct {
		Code     string `json:"code"`
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(b, &items); err != nil {
		t.Fatal(err)
	}
	norm := func(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
	for _, it := range items {
		src, err := os.ReadFile(filepath.Join(root, "examples", it.Filename))
		if err != nil {
			t.Errorf("%s: listed in the IDE but %v", it.Filename, err)
			continue
		}
		if norm(string(src)) != norm(it.Code) {
			t.Errorf("%s: the IDE's copy differs from examples/%s", it.Filename, it.Filename)
		}
	}
}
