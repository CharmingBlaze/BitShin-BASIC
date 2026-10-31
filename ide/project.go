package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ProjectFileName is the project file at a project root. The IDE reads it to
// know what F5 runs, where scenes and assets live, and how Build packages it.
const ProjectFileName = "moon.json"

// Project is moon.json. Every field is optional on disk; Normalize fills
// defaults so a hand-written {"name":"x"} is a valid project.
type Project struct {
	// Name shows in the title bar and the recent list.
	Name string `json:"name"`
	// Main is the script F5 runs, relative to the project root.
	Main string `json:"main"`
	// Window is what the templates pass to Graphics3D / Graphics2D and SetWindowTitle.
	Window ProjectWindow `json:"window"`
	// Quality is the SetGraphicsQuality tier: auto, low, medium, high, ultra.
	Quality string `json:"quality"`
	// Scenes is the folder for .scene.json files, relative to the root.
	Scenes string `json:"scenes"`
	// Assets is the folder for models, textures, and sounds, relative to the root.
	Assets string `json:"assets"`
	// Build holds the Build dialog's defaults.
	Build ProjectBuild `json:"build"`
}

// ProjectWindow is the game window.
type ProjectWindow struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Title  string `json:"title"`
}

// ProjectBuild is used by the Build dialog and by moon build, which stamps
// icon, version, and company into the Windows binary. Profile and Profiles
// are moon build's; the IDE keeps them as they are (cmd/moon/build_profile.go).
type ProjectBuild struct {
	OutputDir string          `json:"outputDir"`
	Icon      string          `json:"icon"`
	Version   string          `json:"version"`
	Company   string          `json:"company"`
	Profile   string          `json:"profile,omitempty"`
	Profiles  json.RawMessage `json:"profiles,omitempty"`
}

// ProjectInfo is a loaded project plus the absolute paths the IDE needs.
type ProjectInfo struct {
	Root     string   `json:"root"`
	File     string   `json:"file"`
	MainPath string   `json:"mainPath"`
	Scenes   string   `json:"scenesPath"`
	Assets   string   `json:"assetsPath"`
	Output   string   `json:"outputPath"`
	Project  Project  `json:"project"`
	Warnings []string `json:"warnings,omitempty"`
}

var qualityTiers = map[string]bool{"auto": true, "low": true, "medium": true, "high": true, "ultra": true}

// DefaultProject is what a missing moon.json field becomes.
func DefaultProject(name string) Project {
	p := Project{Name: name}
	p.Normalize()
	return p
}

// Normalize fills defaults for missing fields and fixes invalid ones. It
// returns a note for each value it had to change (not for empty ones).
func (p *Project) Normalize() []string {
	var notes []string
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		p.Name = "Untitled"
	}
	p.Main = cleanRel(p.Main)
	if p.Main == "" {
		p.Main = "main.mb"
	} else if !isInside(p.Main) {
		notes = append(notes, fmt.Sprintf("main %q is outside the project; using main.mb", p.Main))
		p.Main = "main.mb"
	}
	if p.Window.Width == 0 {
		p.Window.Width = 1280
	}
	if p.Window.Height == 0 {
		p.Window.Height = 720
	}
	if p.Window.Width < 160 || p.Window.Width > 7680 {
		notes = append(notes, fmt.Sprintf("window width %d is out of range 160-7680; using 1280", p.Window.Width))
		p.Window.Width = 1280
	}
	if p.Window.Height < 120 || p.Window.Height > 4320 {
		notes = append(notes, fmt.Sprintf("window height %d is out of range 120-4320; using 720", p.Window.Height))
		p.Window.Height = 720
	}
	if strings.TrimSpace(p.Window.Title) == "" {
		p.Window.Title = p.Name
	}
	p.Quality = strings.ToLower(strings.TrimSpace(p.Quality))
	if p.Quality == "" {
		p.Quality = "auto"
	} else if !qualityTiers[p.Quality] {
		notes = append(notes, fmt.Sprintf("quality %q is not auto, low, medium, high, or ultra; using auto", p.Quality))
		p.Quality = "auto"
	}
	p.Scenes = folderOr(p.Scenes, "scenes", "scenes", &notes)
	p.Assets = folderOr(p.Assets, "assets", "assets", &notes)
	p.Build.OutputDir = folderOr(p.Build.OutputDir, "dist", "build.outputDir", &notes)
	p.Build.Icon = cleanRel(p.Build.Icon)
	if strings.TrimSpace(p.Build.Version) == "" {
		p.Build.Version = "1.0.0"
	}
	p.Build.Company = strings.TrimSpace(p.Build.Company)
	return notes
}

func folderOr(v, def, field string, notes *[]string) string {
	v = cleanRel(v)
	if v == "" {
		return def
	}
	if !isInside(v) {
		*notes = append(*notes, fmt.Sprintf("%s %q is outside the project; using %s", field, v, def))
		return def
	}
	return v
}

// cleanRel turns a moon.json path into a clean slash path.
func cleanRel(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	if p == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(p))
}

func isInside(rel string) bool {
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || (len(rel) > 1 && rel[1] == ':') {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, "../")
}

// projectRoot accepts a folder or a moon.json path.
func projectRoot(path string) string {
	path = strings.TrimSpace(path)
	if strings.EqualFold(filepath.Base(path), ProjectFileName) {
		return filepath.Dir(path)
	}
	return path
}

// LoadProject reads moon.json from a project folder (or the file itself).
func (a *App) LoadProject(path string) (ProjectInfo, error) {
	root := projectRoot(path)
	if root == "" {
		return ProjectInfo{}, errors.New("no project folder given")
	}
	root, _ = filepath.Abs(root)
	file := filepath.Join(root, ProjectFileName)
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectInfo{}, fmt.Errorf("no %s in %s", ProjectFileName, root)
		}
		return ProjectInfo{}, err
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return ProjectInfo{}, fmt.Errorf("%s: %w", file, err)
	}
	if strings.TrimSpace(p.Name) == "" {
		p.Name = filepath.Base(root)
	}
	notes := p.Normalize()
	info := projectInfo(root, p)
	info.Warnings = notes
	if _, err := os.Stat(info.MainPath); err != nil {
		info.Warnings = append(info.Warnings, fmt.Sprintf("main script %s does not exist yet", p.Main))
	}
	a.addRecentProject(root)
	return info, nil
}

// SaveProject validates p and writes moon.json at root.
func (a *App) SaveProject(root string, p Project) (ProjectInfo, error) {
	root = projectRoot(root)
	if root == "" {
		return ProjectInfo{}, errors.New("no project folder given")
	}
	root, _ = filepath.Abs(root)
	notes := p.Normalize()
	if err := writeProjectFile(root, p); err != nil {
		return ProjectInfo{}, err
	}
	for _, dir := range []string{p.Scenes, p.Assets} {
		_ = os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755)
	}
	info := projectInfo(root, p)
	info.Warnings = notes
	a.addRecentProject(root)
	return info, nil
}

// InitProject writes a default moon.json into an existing folder, picking the
// first .mb file at the root as main when there is no main.mb.
func (a *App) InitProject(root string) (ProjectInfo, error) {
	root = projectRoot(root)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return ProjectInfo{}, fmt.Errorf("%s is not a folder", root)
	}
	if _, err := os.Stat(filepath.Join(root, ProjectFileName)); err == nil {
		return a.LoadProject(root)
	}
	p := DefaultProject(filepath.Base(root))
	if _, err := os.Stat(filepath.Join(root, "main.mb")); err != nil {
		if entries, err := os.ReadDir(root); err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mb") && !strings.HasPrefix(e.Name(), ".") {
					p.Main = e.Name()
					break
				}
			}
		}
	}
	return a.SaveProject(root, p)
}

func writeProjectFile(root string, p Project) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, ProjectFileName), append(data, '\n'), 0o644)
}

func projectInfo(root string, p Project) ProjectInfo {
	abs := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	return ProjectInfo{
		Root:     root,
		File:     filepath.Join(root, ProjectFileName),
		MainPath: abs(p.Main),
		Scenes:   abs(p.Scenes),
		Assets:   abs(p.Assets),
		Output:   abs(p.Build.OutputDir),
		Project:  p,
	}
}

var unsafeName = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

// CreateProject makes parent/name from a template and returns the loaded project.
func (a *App) CreateProject(parent, name, template string) (ProjectInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ProjectInfo{}, errors.New("enter a project name")
	}
	if unsafeName.MatchString(name) || name == "." || name == ".." || strings.HasSuffix(name, ".") {
		return ProjectInfo{}, fmt.Errorf("%q cannot be a folder name", name)
	}
	tpl, ok := findTemplate(template)
	if !ok {
		return ProjectInfo{}, fmt.Errorf("unknown template %q", template)
	}
	parent = strings.TrimSpace(parent)
	if parent == "" {
		parent = a.DefaultProjectsDir()
	}
	root, _ := filepath.Abs(filepath.Join(parent, name))
	if entries, err := os.ReadDir(root); err == nil && len(entries) > 0 {
		return ProjectInfo{}, fmt.Errorf("%s already exists and is not empty", root)
	}
	p := DefaultProject(name)
	if tpl.Is2D {
		p.Window.Width, p.Window.Height = 960, 540
	}
	if err := writeTemplate(root, p, tpl); err != nil {
		return ProjectInfo{}, err
	}
	return a.LoadProject(root)
}

// DefaultProjectsDir is where New Project puts projects unless the user picks a folder.
func (a *App) DefaultProjectsDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		docs := filepath.Join(home, "Documents")
		if st, err := os.Stat(docs); err == nil && st.IsDir() {
			return filepath.Join(docs, "moonBASIC Projects")
		}
		return filepath.Join(home, "moonBASIC Projects")
	}
	return filepath.Join(a.repoRoot, "projects")
}

// RecentProject is one row of the start screen's recent list.
type RecentProject struct {
	Root   string `json:"root"`
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
}

const maxRecentProjects = 10

// GetRecentProjects returns the recent list with each project's current name.
func (a *App) GetRecentProjects() []RecentProject {
	s := a.GetSettings()
	out := make([]RecentProject, 0, len(s.RecentProjects))
	for _, root := range s.RecentProjects {
		r := RecentProject{Root: root, Name: filepath.Base(root)}
		if data, err := os.ReadFile(filepath.Join(root, ProjectFileName)); err == nil {
			r.Exists = true
			var p Project
			if json.Unmarshal(data, &p) == nil && strings.TrimSpace(p.Name) != "" {
				r.Name = strings.TrimSpace(p.Name)
			}
		}
		out = append(out, r)
	}
	return out
}

// RemoveRecentProject drops root from the recent list.
func (a *App) RemoveRecentProject(root string) error {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	s := a.GetSettings()
	s.RecentProjects = withoutPath(s.RecentProjects, root)
	return a.writeSettings(s)
}

func (a *App) addRecentProject(root string) {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	s := a.GetSettings()
	list := append([]string{root}, withoutPath(s.RecentProjects, root)...)
	if len(list) > maxRecentProjects {
		list = list[:maxRecentProjects]
	}
	s.RecentProjects = list
	_ = a.writeSettings(s)
}

func withoutPath(list []string, path string) []string {
	out := make([]string, 0, len(list))
	for _, p := range list {
		if !samePath(p, path) {
			out = append(out, p)
		}
	}
	return out
}

func samePath(a, b string) bool {
	ca, cb := filepath.Clean(a), filepath.Clean(b)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(ca, cb)
	}
	return ca == cb
}

// emptySceneJSON is a new version-2 scene file.
const emptySceneJSON = `{
  "version": 2,
  "environment": {},
  "entities": []
}
`

// CreateScene writes an empty version-2 scene. A name without ".scene.json" gets it.
func (a *App) CreateScene(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("no scene path given")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".scene.json") {
		if strings.EqualFold(filepath.Ext(path), ".json") {
			path = path[:len(path)-len(".json")]
		}
		path += ".scene.json"
	}
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists", filepath.Base(path))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte(emptySceneJSON), 0o644)
}
