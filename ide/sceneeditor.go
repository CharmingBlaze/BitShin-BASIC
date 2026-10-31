package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// sceneEditor is one running `moon edit <file>` window.
type sceneEditor struct {
	file   string
	cmd    *exec.Cmd
	pid    int
	events atomic.Int32 // @editor lines seen; 0 at exit means it never started
}

// SceneEditorResult tells the IDE whether a new editor started or an open one was raised.
type SceneEditorResult struct {
	File    string `json:"file"`
	Reused  bool   `json:"reused"`
	Command string `json:"command"`
}

// OpenSceneEditor runs `moon edit <file> -project <root>`. One editor per
// file: a second request raises the window that is already open. Its stdout
// "@editor {json}" lines become editor:event; other output is editor:log;
// editor:exit reports how it ended.
func (a *App) OpenSceneEditor(file, projectRoot string) (SceneEditorResult, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return SceneEditorResult{}, fmt.Errorf("no scene file given")
	}
	file, _ = filepath.Abs(file)
	if _, err := os.Stat(file); err != nil {
		return SceneEditorResult{}, fmt.Errorf("scene file %s: %w", filepath.Base(file), err)
	}
	key := strings.ToLower(file)

	a.editorMu.Lock()
	if a.editors == nil {
		a.editors = map[string]*sceneEditor{}
	}
	if ed := a.editors[key]; ed != nil {
		a.editorMu.Unlock()
		if ed.pid > 0 {
			go raiseProcessWindows(ed.pid)
		}
		return SceneEditorResult{File: file, Reused: true}, nil
	}
	a.editorMu.Unlock()

	moonExe := a.findMoonExecutable()
	if moonExe == "" {
		return SceneEditorResult{}, fmt.Errorf("moon.exe not found. Build it with go build -o moon.exe ./cmd/moon")
	}
	args := []string{"edit", file}
	if root := strings.TrimSpace(projectRoot); root != "" {
		args = append(args, "-project", root)
	}
	cmd := exec.Command(moonExe, args...)
	if root := strings.TrimSpace(projectRoot); root != "" {
		cmd.Dir = root
	} else {
		cmd.Dir = filepath.Dir(file)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return SceneEditorResult{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return SceneEditorResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return SceneEditorResult{}, fmt.Errorf("failed to start the scene editor: %w", err)
	}
	ed := &sceneEditor{file: file, cmd: cmd}
	if cmd.Process != nil {
		ed.pid = cmd.Process.Pid
	}
	a.editorMu.Lock()
	a.editors[key] = ed
	a.editorMu.Unlock()
	if ed.pid > 0 {
		go raiseProcessWindows(ed.pid)
	}
	started := time.Now()
	command := "moon " + strings.Join(args, " ")
	a.emit("editor:start", map[string]any{"file": file, "command": command, "time": started.Format("15:04:05")})

	done := make(chan struct{}, 2)
	go a.pumpEditor(ed, stdout, "stdout", done)
	go a.pumpEditor(ed, stderr, "stderr", done)
	go func() {
		<-done
		<-done
		err := cmd.Wait()
		a.editorMu.Lock()
		if a.editors[key] == ed {
			delete(a.editors, key)
		}
		a.editorMu.Unlock()
		code := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				code = exitErr.ExitCode()
			} else {
				code = 1
			}
		}
		msg := ""
		if code != 0 && ed.events.Load() == 0 {
			msg = fmt.Sprintf("The scene editor did not start (moon edit exited with code %d). "+
				"This moon.exe may not have the `moon edit` command yet: rebuild it with go build -o moon.exe ./cmd/moon.", code)
		} else if code != 0 {
			msg = fmt.Sprintf("The scene editor exited with code %d.", code)
		}
		a.emit("editor:exit", map[string]any{
			"file":    file,
			"code":    code,
			"elapsed": time.Since(started).Seconds(),
			"message": msg,
		})
	}()
	return SceneEditorResult{File: file, Command: command}, nil
}

func (a *App) pumpEditor(ed *sceneEditor, r io.Reader, stream string, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if body, ok := strings.CutPrefix(strings.TrimSpace(line), "@editor "); ok && stream == "stdout" {
			var ev map[string]any
			if json.Unmarshal([]byte(body), &ev) == nil {
				ed.events.Add(1)
				if _, ok := ev["file"]; !ok {
					ev["file"] = ed.file
				}
				ev["editorFile"] = ed.file
				a.emit("editor:event", ev)
				continue
			}
		}
		a.emit("editor:log", map[string]any{"file": ed.file, "stream": stream, "line": line})
	}
}

// OpenSceneEditors lists the scene files that have an editor window open.
func (a *App) OpenSceneEditors() []string {
	a.editorMu.Lock()
	defer a.editorMu.Unlock()
	out := make([]string, 0, len(a.editors))
	for _, ed := range a.editors {
		out = append(out, ed.file)
	}
	return out
}
