package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) startLanguageServer() {
	exe, args := a.resolveLanguageServer()
	if exe == "" {
		return
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = a.repoRoot
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return
	}
	a.lspMu.Lock()
	a.lspCmd = cmd
	a.lspStdin = stdin
	a.lspMu.Unlock()
	go a.readLanguageServer(stdout)
	go func() {
		_ = cmd.Wait()
		a.lspMu.Lock()
		if a.lspCmd == cmd {
			a.lspCmd = nil
			a.lspStdin = nil
		}
		a.lspMu.Unlock()
	}()
}

func (a *App) stopLanguageServer() {
	a.lspMu.Lock()
	cmd := a.lspCmd
	stdin := a.lspStdin
	a.lspCmd = nil
	a.lspStdin = nil
	a.lspMu.Unlock()
	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// SendLSP writes one JSON-RPC body to bsls / bs lsp (Content-Length framed).
func (a *App) SendLSP(body string) error {
	a.lspMu.Lock()
	defer a.lspMu.Unlock()
	if a.lspStdin == nil {
		return fmt.Errorf("language server is not running (build bsls.exe with go build -o bsls.exe ./cmd/bsls)")
	}
	_, err := fmt.Fprintf(a.lspStdin, "Content-Length: %d\r\n\r\n%s", len(body), body)
	return err
}

// IsLspRunning reports whether bsls / bs lsp is attached.
func (a *App) IsLspRunning() bool {
	a.lspMu.Lock()
	defer a.lspMu.Unlock()
	return a.lspStdin != nil
}

func (a *App) resolveLanguageServer() (string, []string) {
	candidates := []string{
		filepath.Join(a.repoRoot, "bsls.exe"),
		filepath.Join(a.repoRoot, "bsls"),
		filepath.Join(".", "bsls.exe"),
		filepath.Join("..", "bsls.exe"),
	}
	for _, c := range candidates {
		st, err := os.Stat(c)
		if err != nil || st.IsDir() {
			continue
		}
		abs, absErr := filepath.Abs(c)
		if absErr != nil {
			return c, nil
		}
		return abs, nil
	}
	if p, err := exec.LookPath("bsls"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("bsls.exe"); err == nil {
		return p, nil
	}
	bs := a.findBsExecutable()
	if bs != "" {
		return bs, []string{"lsp"}
	}
	return "", nil
}

func (a *App) readLanguageServer(r io.Reader) {
	br := bufio.NewReader(r)
	for {
		body, err := readLSPFrame(br)
		if err != nil {
			return
		}
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "lsp:message", string(body))
		}
	}
}

func readLSPFrame(r *bufio.Reader) ([]byte, error) {
	n := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "content-length:") {
			v := strings.TrimSpace(line[len("Content-Length:"):])
			parsed, parseErr := strconv.Atoi(v)
			if parseErr != nil {
				return nil, fmt.Errorf("lsp: bad Content-Length")
			}
			n = parsed
		}
	}
	if n <= 0 {
		return nil, fmt.Errorf("lsp: missing Content-Length")
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}
