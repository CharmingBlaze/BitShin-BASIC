package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed all:frontend/dist
var distEmbed embed.FS

func main() {
	app := NewApp()
	app.startLanguageServer()
	defer app.shutdown()

	dist, err := fs.Sub(distEmbed, "frontend/dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	go func() {
		_ = http.Serve(ln, app.handler(dist))
	}()

	bin, err := findNeutralino()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	configDir, err := findNeutralinoConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	origin := "http://" + ln.Addr().String() + "/"
	cmd := exec.Command(bin, "--path="+configDir, "--url="+origin, "--load-dir-res", "--document-root=/resources/", "--enable-server", "--enable-native-api")
	cmd.Dir = configDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	app.neuCmd = cmd
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Neutralino:", err)
		os.Exit(1)
	}
	if err := cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, "Neutralino:", err)
		os.Exit(1)
	}
}

func findNeutralino() (string, error) {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "neutralino-win_x64.exe"),
			filepath.Join(dir, "bin", "neutralino-win_x64.exe"),
		)
	}
	if root := findRepoRoot(); root != "" {
		candidates = append(candidates, filepath.Join(root, "ide", "bin", "neutralino-win_x64.exe"))
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("neutralino-win_x64.exe not found. Run .\\scripts\\build_ide.ps1")
}

func findNeutralinoConfig() (string, error) {
	var candidates []string
	if root := findRepoRoot(); root != "" {
		candidates = append(candidates, filepath.Join(root, "ide"))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "neutralino.config.json")); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("neutralino.config.json not found next to the IDE")
}
