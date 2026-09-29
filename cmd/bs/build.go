package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var assetExt = regexp.MustCompile(`(?i)"([^"]+\.(png|jpg|jpeg|gif|bmp|wav|ogg|mp3|obj|mtl|gltf|glb|dae|ttf|otf|csv|zip|bb))"`)

func runBuild(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("bs build: missing program.bb")
	}
	src := args[0]
	outDir := "dist"
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH
	buildTags := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-o":
			if i+1 < len(args) {
				outDir = args[i+1]
				i++
			}
		case "-os":
			if i+1 < len(args) {
				targetOS = normalizeOS(args[i+1])
				i++
			}
		case "-arch":
			if i+1 < len(args) {
				targetArch = strings.ToLower(args[i+1])
				i++
			}
		case "-tags":
			if i+1 < len(args) {
				buildTags = args[i+1]
				i++
			}
		}
	}
	if targetOS != "windows" && targetOS != "linux" && targetOS != "darwin" && targetOS != "wasm" {
		return fmt.Errorf("bs build: unknown -os %s (windows, linux, darwin, wasm)", targetOS)
	}
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("bs build: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if targetOS == "wasm" {
		return buildWasmBundle(src, outDir)
	}
	binName := "bs"
	if targetOS == "windows" {
		binName = "bs.exe"
	}
	binPath := filepath.Join(outDir, binName)
	gocmd := []string{"build", "-o", binPath}
	if buildTags != "" {
		gocmd = append(gocmd, "-tags", buildTags)
	}
	gocmd = append(gocmd, "./cmd/bs")
	cmd := exec.Command("go", gocmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := append([]string{}, os.Environ()...)
	env = append(env, "CGO_ENABLED=1")
	if targetOS != runtime.GOOS {
		env = append(env, "GOOS="+targetOS)
	}
	if targetArch != runtime.GOARCH {
		env = append(env, "GOARCH="+targetArch)
	}
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		if targetOS != runtime.GOOS || targetArch != runtime.GOARCH {
			return fmt.Errorf("bs build: CGO cross-compile failed for %s/%s — build on that OS (see docs/RELEASE.md): %w", targetOS, targetArch, err)
		}
		return fmt.Errorf("bs build: compile: %w", err)
	}
	base := filepath.Dir(src)
	if err := copyFile(src, filepath.Join(outDir, filepath.Base(src))); err != nil {
		return err
	}
	seen := map[string]bool{}
	if err := collectAssets(src, base, outDir, seen); err != nil {
		return err
	}
	assetsDir := filepath.Join(base, "assets")
	if st, err := os.Stat(assetsDir); err == nil && st.IsDir() {
		if err := copyDir(assetsDir, filepath.Join(outDir, "assets")); err != nil {
			return err
		}
	}
	if err := copyRuntimeNatives(outDir, targetOS); err != nil {
		fmt.Fprintf(os.Stderr, "bs build: natives (%s): %v\n", targetOS, err)
	}
	fmt.Printf("Wrote %s\n", outDir)
	fmt.Println("Zip that folder and send it. Run the binary from that folder (natives stay next to it).")
	return nil
}

func normalizeOS(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "win", "windows":
		return "windows"
	case "linux":
		return "linux"
	case "mac", "macos", "osx", "darwin":
		return "darwin"
	case "wasm", "web", "html5", "js":
		return "wasm"
	default:
		return strings.ToLower(s)
	}
}

func buildWasmBundle(src, outDir string) error {
	return fmt.Errorf("bs: HTML5/WASM is not a runnable game target (%s -> %s). The runtime needs a desktop OpenGL window. Use bs compile or bs build for windows, linux, or darwin", src, outDir)
}

func collectAssets(bb, base, dest string, seen map[string]bool) error {
	raw, err := os.ReadFile(bb)
	if err != nil {
		return err
	}
	for _, m := range assetExt.FindAllStringSubmatch(string(raw), -1) {
		rel := strings.ReplaceAll(m[1], "\\", "/")
		if seen[rel] {
			continue
		}
		seen[rel] = true
		src := rel
		if !filepath.IsAbs(src) {
			src = filepath.Join(base, filepath.FromSlash(rel))
		}
		if strings.HasSuffix(strings.ToLower(rel), ".bb") {
			if _, err := os.Stat(src); err == nil {
				_ = copyFile(src, filepath.Join(dest, filepath.FromSlash(rel)))
				_ = collectAssets(src, filepath.Dir(src), dest, seen)
			}
			continue
		}
		if _, err := os.Stat(src); err != nil {
			continue
		}
		dst := filepath.Join(dest, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func copyRuntimeNatives(dest, goos string) error {
	switch goos {
	case "windows":
		return copyWindowsRuntimeDLLs(dest)
	case "linux", "darwin":
		return copyUnixNatives(dest, goos)
	default:
		return nil
	}
}

func copyUnixNatives(dest, goos string) error {
	root := findGoModRoot(".")
	if root == "" {
		root = findGoModRoot(exeDir())
	}
	if root == "" {
		return nil
	}
	dir := filepath.Join(root, "third_party", goos)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	ext := ".so"
	if goos == "darwin" {
		ext = ".dylib"
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.Contains(strings.ToLower(name), ext) {
			continue
		}
		if err := copyFile(filepath.Join(dir, name), filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	return nil
}

// windowsRuntimeDLLs are MinGW C++ bits for cimgui. Audio is Oto (no OpenAL).
var windowsRuntimeDLLs = []string{
	"libc++.dll",
	"libunwind.dll",
}

func copyWindowsRuntimeDLLs(dest string) error {
	dirs := windowsDLLSearchDirs()
	var missing []string
	for _, name := range windowsRuntimeDLLs {
		src := findNamedFile(dirs, name)
		if src == "" {
			missing = append(missing, name)
			continue
		}
		if err := copyFile(src, filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("not found: %s (looked in third_party/windows and the G3N module cache)", strings.Join(missing, ", "))
	}
	return nil
}

func windowsDLLSearchDirs() []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if seen[abs] {
			return
		}
		if st, err := os.Stat(abs); err == nil && st.IsDir() {
			seen[abs] = true
			dirs = append(dirs, abs)
		}
	}
	for _, start := range []string{".", exeDir()} {
		if root := findGoModRoot(start); root != "" {
			add(filepath.Join(root, "third_party", "windows"))
		}
	}
	if p, err := exec.LookPath("gcc"); err == nil {
		add(filepath.Dir(p))
	}
	return dirs
}

func exeDir() string {
	self, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(self)
}

func findGoModRoot(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func findNamedFile(dirs []string, name string) string {
	for _, dir := range dirs {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}
