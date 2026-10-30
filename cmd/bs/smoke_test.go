package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A scene chunk another program loads by name is not run on its own.
func TestSmokeSkipsLoadedParts(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	files := []string{
		write("game.mb", "Graphics3D(640, 480)\nLoadScene(\"level.mb\")\n"),
		write("level.mb", "floor = CreatePlane(20, 20)\n"),
		write("other.mb", "Print(\"level\")\n"),
	}
	parts := loadedParts(files)
	if parts["level.mb"] != "game.mb" {
		t.Fatalf("level.mb should be a part of game.mb, got %q", parts["level.mb"])
	}
	if parts["game.mb"] != "" || parts["other.mb"] != "" {
		t.Fatalf("only level.mb is a part: %v", parts)
	}
}
