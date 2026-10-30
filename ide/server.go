package main

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

func (a *App) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeAPI(w, false, "POST required", nil)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	var req struct {
		Path        string       `json:"path"`
		Content     string       `json:"content"`
		DefaultName string       `json:"defaultName"`
		RootPath    string       `json:"rootPath"`
		Lines       string       `json:"lines"`
		Line        string       `json:"line"`
		Code        string       `json:"code"`
		FilePath    string       `json:"filePath"`
		Debug       bool         `json:"debug"`
		OutputDir   string       `json:"outputDir"`
		TargetOS    string       `json:"targetOS"`
		Body        string       `json:"body"`
		Edge        string       `json:"edge"`
		Settings    UserSettings `json:"settings"`
	}
	if len(bytesTrim(body)) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeAPI(w, false, err.Error(), nil)
			return
		}
	}
	method := strings.TrimPrefix(r.URL.Path, "/api/")
	var (
		result any
		err    error
	)
	switch method {
	case "OpenFile":
		result, err = a.OpenFile(req.Path)
	case "SaveFile":
		err = a.SaveFile(req.Path, req.Content)
	case "SelectOpenFile":
		result, err = a.SelectOpenFile()
	case "SelectSaveFile":
		result, err = a.SelectSaveFile(req.DefaultName, req.Content)
	case "SelectProjectDirectory":
		result, err = a.SelectProjectDirectory()
	case "GetProjectTree":
		result, err = a.GetProjectTree(req.RootPath)
	case "SetBreakpoints":
		a.SetBreakpoints(req.Lines)
	case "DebugCommand":
		err = a.DebugCommand(req.Line)
	case "RunProgram":
		err = a.RunProgram(req.Code, req.FilePath, req.Debug)
	case "StopProgram":
		err = a.StopProgram()
	case "BuildExecutable":
		result, err = a.BuildExecutable(req.FilePath, req.OutputDir, req.TargetOS)
	case "GetSettings":
		result = a.GetSettings()
	case "SaveSettings":
		err = a.SaveSettings(req.Settings)
	case "GetRepoRoot":
		result = a.GetRepoRoot()
	case "SendLSP":
		err = a.SendLSP(req.Body)
	case "IsLspRunning":
		result = a.IsLspRunning()
	case "WindowMinimise":
		a.WindowMinimise()
	case "WindowToggleMaximise":
		a.WindowToggleMaximise()
	case "WindowStartResize":
		a.WindowStartResize(req.Edge)
	case "WindowStartMove":
		a.WindowStartMove()
	case "Quit":
		a.WindowClose()
	default:
		writeAPI(w, false, "unknown method "+method, nil)
		return
	}
	if err != nil {
		writeAPI(w, false, err.Error(), nil)
		return
	}
	writeAPI(w, true, "", result)
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func writeAPI(w http.ResponseWriter, ok bool, errText string, result any) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     ok,
		"error":  errText,
		"result": result,
	})
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := a.hub.subscribe()
	defer cancel()
	_, _ = io.WriteString(w, ": ok\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case frame, open := <-ch:
			if !open {
				return
			}
			_, _ = w.Write(frame)
			flusher.Flush()
		}
	}
}

func (a *App) handler(dist fs.FS) http.Handler {
	files := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/events" {
			a.handleEvents(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			a.handleAPI(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
}
