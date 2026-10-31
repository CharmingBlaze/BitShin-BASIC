package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectNormalizeDefaults(t *testing.T) {
	var p Project
	if notes := p.Normalize(); len(notes) != 0 {
		t.Fatalf("empty project should not need notes: %v", notes)
	}
	if p.Main != "main.mb" || p.Window.Width != 1280 || p.Window.Height != 720 || p.Quality != "auto" ||
		p.Scenes != "scenes" || p.Assets != "assets" || p.Build.OutputDir != "dist" || p.Build.Version != "1.0.0" {
		t.Fatalf("defaults: %+v", p)
	}
	bad := Project{Main: "../x.mb", Quality: "insane", Window: ProjectWindow{Width: 5}, Scenes: "C:/abs"}
	if notes := bad.Normalize(); len(notes) != 4 {
		t.Fatalf("want 4 notes, got %v", notes)
	}
	if bad.Main != "main.mb" || bad.Quality != "auto" || bad.Window.Width != 1280 || bad.Scenes != "scenes" {
		t.Fatalf("fixed: %+v", bad)
	}
}

func TestCreateEveryTemplate(t *testing.T) {
	a := &App{repoRoot: t.TempDir()}
	parent := t.TempDir()
	for _, tpl := range projectTemplates {
		info, err := a.CreateProject(parent, "Game "+tpl.ID, tpl.ID)
		if err != nil {
			t.Fatalf("%s: %v", tpl.ID, err)
		}
		if len(info.Warnings) != 0 {
			t.Fatalf("%s warnings: %v", tpl.ID, info.Warnings)
		}
		src, err := os.ReadFile(info.MainPath)
		if err != nil || strings.Contains(string(src), "{{") {
			t.Fatalf("%s main.mb: %v %q", tpl.ID, err, src)
		}
		scene := filepath.Join(info.Scenes, "main.scene.json")
		data, err := os.ReadFile(scene)
		if tpl.Is2D {
			if err == nil {
				t.Fatalf("%s: 2D template wrote a scene", tpl.ID)
			}
		} else {
			var doc struct {
				Version  int              `json:"version"`
				Entities []map[string]any `json:"entities"`
			}
			if err != nil || json.Unmarshal(data, &doc) != nil || doc.Version != 2 || len(doc.Entities) < 3 {
				t.Fatalf("%s scene: %v %s", tpl.ID, err, data)
			}
		}
		if st, err := os.Stat(info.Assets); err != nil || !st.IsDir() {
			t.Fatalf("%s: no assets folder", tpl.ID)
		}
	}
	if _, err := a.CreateProject(parent, "Game empty3d", "empty3d"); err == nil {
		t.Fatal("creating into a non-empty folder should fail")
	}
	recent := a.GetRecentProjects()
	if len(recent) != len(projectTemplates) || !recent[0].Exists || recent[0].Name != "Game topdown2d" {
		t.Fatalf("recent: %+v", recent)
	}
}

func TestSaveSettingsKeepsRecentProjects(t *testing.T) {
	a := &App{repoRoot: t.TempDir()}
	a.addRecentProject(`C:\games\one`)
	s := a.GetSettings()
	s.RecentProjects = nil
	s.FontSize = 18
	if err := a.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	got := a.GetSettings()
	if got.FontSize != 18 || len(got.RecentProjects) != 1 {
		t.Fatalf("settings: %+v", got)
	}
}

func TestPumpEditorEvents(t *testing.T) {
	a := &App{hub: newEventHub()}
	ch, cancel := a.hub.subscribe()
	defer cancel()
	ed := &sceneEditor{file: `C:\p\scenes\main.scene.json`}
	out := "loading\n@editor {\"event\":\"saved\",\"file\":\"C:/p/scenes/main.scene.json\"}\r\n@editor {\"event\":\"play\"}\n@editor not json\n"
	done := make(chan struct{}, 1)
	a.pumpEditor(ed, strings.NewReader(out), "stdout", done)
	var frames []string
	for len(ch) > 0 {
		frames = append(frames, string(<-ch))
	}
	if len(frames) != 4 || ed.events.Load() != 2 {
		t.Fatalf("frames %d events %d: %q", len(frames), ed.events.Load(), frames)
	}
	for i, want := range []string{"event: editor:log", "event: editor:event", "event: editor:event", "event: editor:log"} {
		if !strings.HasPrefix(frames[i], want) {
			t.Fatalf("frame %d = %q, want %s", i, frames[i], want)
		}
	}
	if !strings.Contains(frames[2], `"editorFile"`) || !strings.Contains(frames[2], `"play"`) {
		t.Fatalf("play frame: %s", frames[2])
	}
}

func TestCreateSceneName(t *testing.T) {
	a := &App{repoRoot: t.TempDir()}
	dir := t.TempDir()
	for in, want := range map[string]string{"a": "a.scene.json", "b.json": "b.scene.json", "c.scene.json": "c.scene.json"} {
		got, err := a.CreateScene(filepath.Join(dir, in))
		if err != nil || filepath.Base(got) != want {
			t.Fatalf("%s: %s %v", in, got, err)
		}
	}
	if _, err := a.CreateScene(filepath.Join(dir, "a")); err == nil {
		t.Fatal("existing scene should not be overwritten")
	}
}
