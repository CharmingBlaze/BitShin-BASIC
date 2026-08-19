package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	cmdMutex   sync.Mutex
	runningCmd *exec.Cmd
	repoRoot   string
	lspMu      sync.Mutex
	lspCmd     *exec.Cmd
	lspStdin   io.WriteCloser
}

// NewApp creates a new App application struct
func NewApp() *App {
	repoRoot := findRepoRoot()
	return &App{
		repoRoot: repoRoot,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startLanguageServer()
}

func (a *App) onDomReady(ctx context.Context) {
	a.ctx = ctx
	a.fixFramelessBounds()
	wailsruntime.WindowShow(ctx)
}

// fixFramelessBounds forces Windows to recalculate the frameless client area.
// Wails frameless windows keep a hidden caption inset on first paint, so the
// custom title bar starts off-screen until the user resizes.
func (a *App) fixFramelessBounds() {
	ctx := a.ctx
	if ctx == nil {
		return
	}
	w, h := wailsruntime.WindowGetSize(ctx)
	if w < 800 {
		w = 980
	}
	if h < 500 {
		h = 620
	}
	wailsruntime.WindowSetSize(ctx, w, h)
	wailsruntime.WindowCenter(ctx)
}

// shutdown is called when the app terminates
func (a *App) shutdown(ctx context.Context) {
	a.StopProgram()
	a.stopLanguageServer()
}

func findRepoRoot() string {
	// Try current working dir, then exe dir, then check upward for go.mod or examples
	dir, err := os.Getwd()
	if err == nil {
		if _, err := os.Stat(filepath.Join(dir, "examples")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "..", "examples")); err == nil {
			return filepath.Clean(filepath.Join(dir, ".."))
		}
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 5; i++ {
			if _, err := os.Stat(filepath.Join(dir, "examples")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return dir
}

// FileResult holds loaded file details
type FileResult struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	IsBinary bool   `json:"isBinary"`
}

// FileNode represents a tree item in the explorer
type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"isDir"`
	Children []*FileNode `json:"children,omitempty"`
	Size     int64       `json:"size,omitempty"`
}

// OpenFile reads file from disk
func (a *App) OpenFile(filePath string) (FileResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{
		Path:    filePath,
		Name:    filepath.Base(filePath),
		Content: string(data),
	}, nil
}

// SaveFile writes content to disk
func (a *App) SaveFile(filePath string, content string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0o644)
}

// SelectOpenFile opens a file dialog
func (a *App) SelectOpenFile() (FileResult, error) {
	chosen, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Open BitShin BASIC File",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "BitShin BASIC (*.bb, *.basic)", Pattern: "*.bb;*.basic;*.txt"},
			{DisplayName: "Shaders (*.glsl, *.vert, *.frag)", Pattern: "*.glsl;*.vert;*.frag"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || chosen == "" {
		return FileResult{}, err
	}
	return a.OpenFile(chosen)
}

// SelectSaveFile opens a save file dialog
func (a *App) SelectSaveFile(defaultName string, content string) (string, error) {
	chosen, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:            "Save BitShin BASIC File",
		DefaultFilename:  defaultName,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "BitShin BASIC (*.bb)", Pattern: "*.bb"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || chosen == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(chosen), ".bb") && !strings.Contains(filepath.Base(chosen), ".") {
		chosen += ".bb"
	}
	err = a.SaveFile(chosen, content)
	return chosen, err
}

// SelectProjectDirectory opens directory dialog
func (a *App) SelectProjectDirectory() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Project Directory",
	})
}

// GetProjectTree returns file tree
func (a *App) GetProjectTree(rootPath string) (*FileNode, error) {
	if rootPath == "" {
		rootPath = a.repoRoot
	}
	info, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}
	rootNode := &FileNode{
		Name:  filepath.Base(rootPath),
		Path:  rootPath,
		IsDir: info.IsDir(),
	}
	if !info.IsDir() {
		return rootNode, nil
	}

	var walk func(node *FileNode, depth int) error
	walk = func(node *FileNode, depth int) error {
		if depth > 5 {
			return nil
		}
		entries, err := os.ReadDir(node.Path)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "dist" || name == "third_party" {
				continue
			}
			childPath := filepath.Join(node.Path, name)
			childInfo, _ := e.Info()
			var size int64
			if childInfo != nil {
				size = childInfo.Size()
			}
			childNode := &FileNode{
				Name:  name,
				Path:  childPath,
				IsDir: e.IsDir(),
				Size:  size,
			}
			if e.IsDir() {
				_ = walk(childNode, depth+1)
			}
			node.Children = append(node.Children, childNode)
		}
		return nil
	}

	_ = walk(rootNode, 0)
	return rootNode, nil
}

// RunProgram runs code or a file with bs.exe
func (a *App) RunProgram(code string, filePath string, debug bool) error {
	a.StopProgram()

	bsExe := a.findBsExecutable()
	if bsExe == "" {
		return fmt.Errorf("bs.exe not found. Please build bs.exe first using 'go build -o bs.exe ./cmd/bs'")
	}

	var runPath string
	var workDir string

	if strings.HasPrefix(filePath, "example://") || strings.HasPrefix(filePath, "temp://") {
		tempDir := filepath.Join(a.repoRoot, "scratch")
		_ = os.MkdirAll(tempDir, 0o755)
		runPath = filepath.Join(tempDir, "_temp_run.bb")
		if err := os.WriteFile(runPath, []byte(code), 0o644); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		workDir = a.repoRoot
	} else if filePath != "" && !strings.HasPrefix(filePath, "temp://") {
		workDir = filepath.Dir(filePath)
		runPath = filepath.Join(workDir, ".bitshin_run.bb")
		if err := os.WriteFile(runPath, []byte(code), 0o644); err != nil {
			return fmt.Errorf("failed to write temp run file: %w", err)
		}
	} else {
		// Temporary scratch file
		tempDir := filepath.Join(a.repoRoot, "scratch")
		_ = os.MkdirAll(tempDir, 0o755)
		runPath = filepath.Join(tempDir, "_temp_run.bb")
		if err := os.WriteFile(runPath, []byte(code), 0o644); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		workDir = a.repoRoot
	}

	cmd := exec.Command(bsExe, runPath)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start program: %w", err)
	}

	a.cmdMutex.Lock()
	a.runningCmd = cmd
	a.cmdMutex.Unlock()

	startTime := time.Now()
	wailsruntime.EventsEmit(a.ctx, "run:start", map[string]any{
		"file": filepath.Base(runPath),
		"path": runPath,
		"time": startTime.Format("15:04:05"),
	})

	// Stream stdout
	go func() {
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				wailsruntime.EventsEmit(a.ctx, "run:stdout", strings.TrimRight(line, "\r\n"))
			}
			if err != nil {
				break
			}
		}
	}()

	// Stream stderr
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				wailsruntime.EventsEmit(a.ctx, "run:stderr", strings.TrimRight(line, "\r\n"))
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait in background
	go func() {
		err := cmd.Wait()
		elapsed := time.Since(startTime)

		a.cmdMutex.Lock()
		a.runningCmd = nil
		a.cmdMutex.Unlock()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		wailsruntime.EventsEmit(a.ctx, "run:exit", map[string]any{
			"code":    exitCode,
			"elapsed": elapsed.Seconds(),
			"time":    time.Now().Format("15:04:05"),
		})
	}()

	return nil
}

// StopProgram kills running instance
func (a *App) StopProgram() error {
	a.cmdMutex.Lock()
	defer a.cmdMutex.Unlock()

	if a.runningCmd != nil && a.runningCmd.Process != nil {
		_ = a.runningCmd.Process.Kill()
		a.runningCmd = nil
		wailsruntime.EventsEmit(a.ctx, "run:stopped", "Program stopped by user")
	}
	return nil
}

// BuildExecutable runs bs build
func (a *App) BuildExecutable(filePath string, outputDir string, targetOS string) (string, error) {
	bsExe := a.findBsExecutable()
	if bsExe == "" {
		return "", fmt.Errorf("bs.exe not found")
	}
	if outputDir == "" {
		outputDir = filepath.Join(filepath.Dir(filePath), "dist")
	}
	args := []string{"build", filePath, "-o", outputDir}
	if targetOS != "" {
		args = append(args, "-os", targetOS)
	}

	cmd := exec.Command(bsExe, args...)
	cmd.Dir = filepath.Dir(filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("build failed: %s (%w)", string(out), err)
	}
	return string(out), nil
}

func (a *App) findBsExecutable() string {
	candidates := []string{
		filepath.Join(a.repoRoot, "bs.exe"),
		filepath.Join(a.repoRoot, "bs"),
		filepath.Join(a.repoRoot, "cmd", "bs", "bs.exe"),
		filepath.Join(a.repoRoot, "cmd", "bs", "bs"),
		filepath.Join(".", "bs.exe"),
		filepath.Join(".", "bs"),
		filepath.Join("..", "bs.exe"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() && st.Size() > 0 {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	if p, err := exec.LookPath("bs"); err == nil {
		return p
	}
	if p, err := exec.LookPath("bs.exe"); err == nil {
		return p
	}
	return ""
}

// GetRepoRoot returns repository root directory
func (a *App) GetRepoRoot() string {
	return a.repoRoot
}

// UserSettings represents IDE preferences
type UserSettings struct {
	Theme         string `json:"theme"`
	FontSize      int    `json:"fontSize"`
	TabSize       int    `json:"tabSize"`
	Minimap       bool   `json:"minimap"`
	WordWrap      string `json:"wordWrap"`
	AutoSave      bool   `json:"autoSave"`
	TargetOS      string `json:"targetOS"`
	OutputHeight  int    `json:"outputHeight"`
	SidebarWidth  int    `json:"sidebarWidth"`
}

func (a *App) GetSettings() UserSettings {
	settingsFile := filepath.Join(a.repoRoot, ".ide_settings.json")
	var s UserSettings
	data, err := os.ReadFile(settingsFile)
	if err == nil {
		_ = json.Unmarshal(data, &s)
		return s
	}
	return UserSettings{
		Theme:        "bitshin-dark",
		FontSize:     14,
		TabSize:      4,
		Minimap:      true,
		WordWrap:     "off",
		AutoSave:     true,
		TargetOS:     runtime.GOOS,
		OutputHeight: 220,
		SidebarWidth: 320,
	}
}

func (a *App) SaveSettings(s UserSettings) error {
	settingsFile := filepath.Join(a.repoRoot, ".ide_settings.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsFile, data, 0o644)
}
