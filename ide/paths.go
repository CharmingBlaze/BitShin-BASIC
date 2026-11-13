package main

import (
	"os"
	"path/filepath"
)

// The IDE keeps nothing of its own in its install folder, so it runs from a
// read-only place (Program Files) and the folder stays as shipped.

// exeDir is the folder moonBASIC.exe is in.
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// settingsPath is the editor preferences file, in the user's settings folder
// (%AppData%\moonBASIC on Windows).
func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "moonBASIC", "settings.json")
}

// scratchDir holds the copy of an unsaved tab that Run hands to moon.
func scratchDir() string {
	return filepath.Join(os.TempDir(), "moonBASIC")
}

func (a *App) settingsFile() string {
	if a.settings != "" {
		return a.settings
	}
	return filepath.Join(a.repoRoot, ".ide_settings.json")
}
