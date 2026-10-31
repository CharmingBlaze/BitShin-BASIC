package main

import (
	"bufio"
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
)

// App struct
type App struct {
	hub        *eventHub
	neuCmd     *exec.Cmd
	winMu      sync.Mutex
	winMax     bool
	winSaved   *winRect
	cmdMutex   sync.Mutex
	runningCmd *exec.Cmd
	runningPID int
	debugIn    io.WriteCloser
	breaks     string
	repoRoot   string
	lspMu      sync.Mutex
	lspCmd     *exec.Cmd
	lspStdin   io.WriteCloser
	settingsMu sync.Mutex
	editorMu   sync.Mutex
	editors    map[string]*sceneEditor
}

// NewApp creates a new App application struct
func NewApp() *App {
	repoRoot := findRepoRoot()
	return &App{
		hub:      newEventHub(),
		repoRoot: repoRoot,
	}
}

func (a *App) shutdown() {
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
	chosen, err := browseOpenFile()
	if err != nil || chosen == "" {
		return FileResult{}, err
	}
	return a.OpenFile(chosen)
}

// SelectSaveFile opens a save file dialog
func (a *App) SelectSaveFile(defaultName string, content string) (string, error) {
	chosen, err := browseSaveFile(defaultName)
	if err != nil || chosen == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(chosen), ".mb") && !strings.Contains(filepath.Base(chosen), ".") {
		chosen += ".mb"
	}
	err = a.SaveFile(chosen, content)
	return chosen, err
}

// SelectProjectDirectory opens directory dialog
func (a *App) SelectProjectDirectory() (string, error) {
	return browseDirectory()
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

// RunProgram runs code or a file with moon.exe
// SetBreakpoints stores comma-separated source lines for the next debug run.
func (a *App) SetBreakpoints(lines string) {
	a.cmdMutex.Lock()
	a.breaks = lines
	a.cmdMutex.Unlock()
}

// DebugCommand sends a debugger command (c, s, n, o, bt, p expr, b line, d line, q)
// to a paused debug session. Pauses come back on stderr as "@debug {json}" lines.
func (a *App) DebugCommand(line string) error {
	a.cmdMutex.Lock()
	w := a.debugIn
	a.cmdMutex.Unlock()
	if w == nil {
		return fmt.Errorf("program is not paused in the debugger")
	}
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	_, err := io.WriteString(w, line)
	return err
}

func (a *App) RunProgram(code string, filePath string, debug bool) error {
	a.StopProgram()

	moonExe := a.findMoonExecutable()
	if moonExe == "" {
		return fmt.Errorf("moon.exe not found. Build it with go build -o moon.exe ./cmd/moon")
	}

	var runPath string
	var workDir string

	if strings.HasPrefix(filePath, "example://") || strings.HasPrefix(filePath, "temp://") {
		tempDir := filepath.Join(a.repoRoot, "scratch")
		_ = os.MkdirAll(tempDir, 0o755)
		runPath = filepath.Join(tempDir, "_temp_run.mb")
		if err := os.WriteFile(runPath, []byte(code), 0o644); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		workDir = a.repoRoot
	} else if filePath != "" && !strings.HasPrefix(filePath, "temp://") {
		workDir = filepath.Dir(filePath)
		runPath = filepath.Join(workDir, ".moonbasic_run.mb")
		if err := os.WriteFile(runPath, []byte(code), 0o644); err != nil {
			return fmt.Errorf("failed to write temp run file: %w", err)
		}
	} else {
		// Temporary scratch file
		tempDir := filepath.Join(a.repoRoot, "scratch")
		_ = os.MkdirAll(tempDir, 0o755)
		runPath = filepath.Join(tempDir, "_temp_run.mb")
		if err := os.WriteFile(runPath, []byte(code), 0o644); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		workDir = a.repoRoot
	}

	runArgs := []string{runPath}
	if debug {
		runArgs = append(runArgs, "-debug", "-debugjson")
		a.cmdMutex.Lock()
		br := strings.TrimSpace(a.breaks)
		a.cmdMutex.Unlock()
		if br != "" {
			runArgs = append(runArgs, "-break", br)
		}
	}
	cmd := exec.Command(moonExe, runArgs...)
	cmd.Dir = workDir
	// DebugLog shows in the Output panel for every IDE run; errors name the tab, not the temp file.
	cmd.Env = append(os.Environ(), "MOON_DEBUG=1", "MOON_SCRIPT_NAME="+scriptName(filePath, runPath))
	// MOON_IDE turns on asset hot reload; MOON_SCRIPT_SOURCE is the saved tab
	// it watches, since the run itself is a temp copy.
	cmd.Env = append(cmd.Env, "MOON_IDE=1")
	if filePath != "" && !strings.Contains(filePath, "://") {
		cmd.Env = append(cmd.Env, "MOON_SCRIPT_SOURCE="+filePath)
	}
	if debug {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		a.cmdMutex.Lock()
		a.debugIn = stdin
		a.cmdMutex.Unlock()
	}

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
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	a.cmdMutex.Lock()
	a.runningCmd = cmd
	a.runningPID = pid
	a.cmdMutex.Unlock()
	if pid > 0 {
		go raiseProcessWindows(pid)
	}

	startTime := time.Now()
	a.emit("run:start", map[string]any{
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
				a.emit("run:stdout", strings.TrimRight(line, "\r\n"))
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
				a.emit("run:stderr", strings.TrimRight(line, "\r\n"))
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait in background. Only this command may clear the running slot:
	// a later Run can start before this Wait returns, and wiping the
	// new handle is why Stop sometimes has nothing to kill.
	go func() {
		err := cmd.Wait()
		elapsed := time.Since(startTime)

		a.cmdMutex.Lock()
		current := a.runningCmd == cmd
		if current {
			a.runningCmd = nil
			a.runningPID = 0
			if a.debugIn != nil {
				_ = a.debugIn.Close()
				a.debugIn = nil
			}
		}
		a.cmdMutex.Unlock()
		if !current {
			return
		}

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		a.emit("run:exit", map[string]any{
			"code":    exitCode,
			"elapsed": elapsed.Seconds(),
			"time":    time.Now().Format("15:04:05"),
		})
	}()

	return nil
}

// StopProgram kills the running game and every process it started.
func (a *App) StopProgram() error {
	a.cmdMutex.Lock()
	cmd := a.runningCmd
	pid := a.runningPID
	debugIn := a.debugIn
	a.runningCmd = nil
	a.runningPID = 0
	a.debugIn = nil
	a.cmdMutex.Unlock()

	if pid <= 0 && (cmd == nil || cmd.Process == nil) {
		return nil
	}
	if pid <= 0 && cmd != nil && cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	killProcessTree(pid)
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if debugIn != nil {
		_ = debugIn.Close()
	}
	a.emit("run:stopped", "Program stopped by user")
	return nil
}

// BuildExecutable runs bs compile or bs build
func (a *App) BuildExecutable(filePath string, outputDir string, targetOS string) (string, error) {
	moonExe := a.findMoonExecutable()
	if moonExe == "" {
		return "", fmt.Errorf("moon.exe not found")
	}
	if outputDir == "" {
		outputDir = filepath.Join(filepath.Dir(filePath), "dist")
	}

	subcmd := "compile"
	if targetOS == "bundle" {
		subcmd = "build"
		targetOS = runtime.GOOS
	}

	var args []string
	if subcmd == "compile" {
		baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		exeName := baseName
		if targetOS == "windows" || (targetOS == "" && runtime.GOOS == "windows") {
			exeName += ".exe"
		}
		outBin := filepath.Join(outputDir, exeName)
		args = []string{"compile", filePath, "-o", outBin}
		if targetOS != "" {
			args = append(args, "-os", targetOS)
		}
	} else {
		args = []string{"build", filePath, "-o", outputDir}
		if targetOS != "" {
			args = append(args, "-os", targetOS)
		}
	}

	cmd := exec.Command(moonExe, args...)
	cmd.Dir = filepath.Dir(filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("build failed: %s (%w)", string(out), err)
	}
	return string(out), nil
}

func (a *App) findMoonExecutable() string {
	candidates := []string{
		filepath.Join(a.repoRoot, "moon.exe"),
		filepath.Join(a.repoRoot, "moon"),
		filepath.Join(a.repoRoot, "cmd", "moon", "moon.exe"),
		filepath.Join(a.repoRoot, "cmd", "moon", "moon"),
		filepath.Join(".", "moon.exe"),
		filepath.Join(".", "moon"),
		filepath.Join("..", "moon.exe"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() && st.Size() > 0 {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	if p, err := exec.LookPath("moon"); err == nil {
		return p
	}
	if p, err := exec.LookPath("moon.exe"); err == nil {
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
	Theme        string `json:"theme"`
	FontSize     int    `json:"fontSize"`
	TabSize      int    `json:"tabSize"`
	Minimap      bool   `json:"minimap"`
	WordWrap     string `json:"wordWrap"`
	AutoSave     bool   `json:"autoSave"`
	TargetOS     string `json:"targetOS"`
	OutputHeight int    `json:"outputHeight"`
	SidebarWidth int    `json:"sidebarWidth"`
	// RecentProjects is the start screen's list, newest first. The backend owns it.
	RecentProjects []string `json:"recentProjects,omitempty"`
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
		Theme:        "moonbasic-dark",
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

// SaveSettings writes the editor preferences. The recent-project list is kept
// from disk when the caller does not send one.
func (a *App) SaveSettings(s UserSettings) error {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	if s.RecentProjects == nil {
		cur := a.GetSettings()
		s.RecentProjects = cur.RecentProjects
	}
	return a.writeSettings(s)
}

func (a *App) writeSettings(s UserSettings) error {
	settingsFile := filepath.Join(a.repoRoot, ".ide_settings.json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsFile, data, 0o644)
}

// scriptName is the name DebugLog, Assert, and the call stack report for an IDE run.
func scriptName(filePath, runPath string) string {
	name := filePath
	for _, prefix := range []string{"temp://", "example://"} {
		name = strings.TrimPrefix(name, prefix)
	}
	if name == "" {
		return filepath.Base(runPath)
	}
	return filepath.Base(name)
}
