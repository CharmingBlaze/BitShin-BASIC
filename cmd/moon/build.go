package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"moonbasic/internal/joltstage"
)

// buildOpts is one moon build after flags, moon.json, and the profile merge.
type buildOpts struct {
	src        string
	outDir     string
	targetOS   string
	targetArch string
	tags       string
	profile    string
	prof       profileFlags
	exeName    string // binary name without .exe
	appName    string // ProductName in the Windows version info
	version    string
	company    string
	icon       string   // .png / .jpg / .ico, empty for none
	dirs       []string // folders copied whole (assets, scenes), relative to the script
}

func runBuild(args []string) error {
	o, err := parseBuildArgs(args)
	if err != nil {
		return err
	}
	if o.targetOS != "windows" && o.targetOS != "linux" && o.targetOS != "darwin" && o.targetOS != "wasm" {
		return fmt.Errorf("moon build: unknown -os %s (windows, linux, darwin, wasm)", o.targetOS)
	}
	if _, err := os.Stat(o.src); err != nil {
		return fmt.Errorf("moon build: %w", err)
	}
	scan, err := scanAssets(o.src)
	if err != nil {
		return fmt.Errorf("moon build: %w", err)
	}
	for _, m := range scan.missing {
		fmt.Fprintf(os.Stderr, "moon build: warning: %s:%d: missing asset %q\n", m.file, m.line, m.path)
	}
	if o.prof.strict && len(scan.missing) > 0 {
		return fmt.Errorf("moon build: %d missing asset(s) (-strict)", len(scan.missing))
	}
	if err := os.MkdirAll(o.outDir, 0o755); err != nil {
		return err
	}
	if o.targetOS == "wasm" {
		return buildWasmBundle(o.src, o.outDir)
	}
	if o.targetOS == "linux" || o.targetOS == "darwin" {
		root, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := joltstage.Stage(root); err != nil {
			return fmt.Errorf("moon build: %w", err)
		}
	}
	if o.targetOS == "windows" {
		syso, err := writeWindowsResources(o)
		if err != nil {
			return fmt.Errorf("moon build: %w", err)
		}
		defer os.Remove(syso)
	} else if o.icon != "" {
		fmt.Fprintln(os.Stderr, "moon build: -icon is embedded on Windows only")
	}
	binName := o.exeName
	if o.targetOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(o.outDir, binName)
	cmd := exec.Command("go", goBuildArgs(o, binPath)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := append([]string{}, os.Environ()...)
	env = append(env, "CGO_ENABLED=1")
	if o.targetOS != runtime.GOOS {
		env = append(env, "GOOS="+o.targetOS)
	}
	if o.targetArch != runtime.GOARCH {
		env = append(env, "GOARCH="+o.targetArch)
	}
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		if o.targetOS != runtime.GOOS || o.targetArch != runtime.GOARCH {
			return fmt.Errorf("moon build: CGO cross-compile failed for %s/%s — build on that OS (see docs/RELEASE.md): %w", o.targetOS, o.targetArch, err)
		}
		return fmt.Errorf("moon build: compile: %w", err)
	}
	base := filepath.Dir(o.src)
	for _, rel := range scan.scripts {
		if err := copyFile(filepath.Join(base, filepath.FromSlash(rel)), filepath.Join(o.outDir, filepath.FromSlash(rel))); err != nil {
			return err
		}
	}
	files := append([]string{}, scan.assets...)
	for _, dir := range o.dirs {
		more, err := listDir(base, dir)
		if err != nil {
			return err
		}
		files = append(files, more...)
	}
	files = uniqueStrings(files)
	pakPath := filepath.Join(o.outDir, o.exeName+".pak")
	if o.prof.pak {
		if err := writePak(pakPath, base, files); err != nil {
			return fmt.Errorf("moon build: pak: %w", err)
		}
	} else {
		_ = os.Remove(pakPath)
		for _, rel := range files {
			if err := copyFile(filepath.Join(base, filepath.FromSlash(rel)), filepath.Join(o.outDir, filepath.FromSlash(rel))); err != nil {
				return err
			}
		}
	}
	if err := copyRuntimeNatives(o.outDir, o.targetOS); err != nil {
		fmt.Fprintf(os.Stderr, "moon build: natives (%s): %v\n", o.targetOS, err)
	}
	how := fmt.Sprintf("%d loose asset(s)", len(files))
	if o.prof.pak {
		how = fmt.Sprintf("%d asset(s) in %s", len(files), filepath.Base(pakPath))
	}
	fmt.Printf("Wrote %s (%s profile: %s, %s)\n", o.outDir, o.profile, binName, how)
	fmt.Println("Zip that folder and send it. Run the binary from that folder (natives stay next to it).")
	return nil
}

// goBuildArgs is the go build command line for a profile: build tags, the
// boot script and profile stamped with -X, stripping, and on Windows the GUI
// subsystem (no console window) unless the profile keeps the console.
func goBuildArgs(o buildOpts, binPath string) []string {
	ld := []string{
		quoteLD("main.bootScript=" + filepath.Base(o.src)),
		quoteLD("main.bootProfile=" + o.profile),
	}
	if o.prof.headless {
		ld = append(ld, quoteLD("main.bootHeadless=1"))
	}
	if o.prof.strip {
		ld = append(ld, "-s", "-w")
	}
	if o.targetOS == "windows" && !o.prof.console && !o.prof.headless {
		ld = append(ld, "-H", "windowsgui")
	}
	args := []string{"build", "-o", binPath}
	if o.prof.strip {
		args = append(args, "-trimpath")
	}
	if tags := joinTags(o.prof.tags, o.tags); tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, "-ldflags", strings.Join(ld, " "), "./cmd/moon")
	return args
}

// quoteLD makes one -X setting for -ldflags; the go command splits on spaces
// outside single quotes.
func quoteLD(kv string) string {
	return "-X '" + strings.ReplaceAll(kv, "'", "") + "'"
}

func joinTags(lists ...string) string {
	var out []string
	seen := map[string]bool{}
	for _, l := range lists {
		for _, t := range strings.FieldsFunc(l, func(r rune) bool { return r == ',' || r == ' ' }) {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	return strings.Join(out, ",")
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
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
	return fmt.Errorf("moon: HTML5/WASM is not a runnable game target (%s -> %s). The runtime needs a desktop OpenGL window. Use moon compile or moon build for windows, linux, or darwin", src, outDir)
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
