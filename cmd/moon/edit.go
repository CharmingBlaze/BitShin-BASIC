package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"moonbasic/internal/runtime"
)

// runEdit opens the scene editor: moon edit [file.scene.json] [-project dir].
// With no file it opens scenes/main.scene.json in the project.
func runEdit(args []string) error {
	path, project := "", ""
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-project" && i+1 < len(args):
			project = args[i+1]
			i++
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("moon edit: unknown flag %s (usage: moon edit [file.scene.json] [-project dir])", a)
		default:
			path = a
		}
	}
	if project == "" {
		project = "."
		if path != "" {
			project = filepath.Dir(path)
		}
	}
	if path == "" {
		path = filepath.Join(project, "scenes", "main.scene.json")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	proj, err := filepath.Abs(project)
	if err != nil {
		return err
	}
	return runtime.RunEditor(abs, proj)
}
