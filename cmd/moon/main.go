package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"moonbasic/internal/glslang"
	"moonbasic/internal/interp"
	"moonbasic/internal/lsp"
	"moonbasic/internal/parse"
	"moonbasic/internal/runtime"
)

const version = "1.0.0"

// moon build sets these with -ldflags -X so a packaged binary runs its game
// when started with no arguments (a double-click).
var (
	bootScript   string // main script beside the binary, e.g. "game.mb"
	bootHeadless string // "1": no window (moon build -headless)
	bootProfile  string // dev, debug, release, or a moon.json profile
)

func main() {
	args := os.Args[1:]
	if bootScript != "" && (len(args) == 0 || args[0] == "-headless" || args[0] == "-debug") {
		os.Exit(runBoot(args))
	}
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(`moonBASIC — Blitz3D-style BASIC (G3N 3D, Ebiten 2D)

Usage:
  moon <program.mb> [-debug] [-break 10,20] [-debugjson] [-headless]
                                 Run a program. -debug pauses on -break lines and DebugBreak,
                                 shows DebugLog output (also MOON_DEBUG=1). At the debug> prompt:
                                 c continue, s step into, n step over, o step out, bt call stack,
                                 locals, globals, p expr, b line, d line, q quit.
                                 -debugjson prints each pause as one @debug JSON line (the IDE uses it)
                                 -headless runs with no window or GL (servers, tests; also MOON_HEADLESS=1)
  moon run <program.mb>          Same
  moon compile <program.mb> [-o app.exe] [-os windows|linux|darwin] [-arch amd64|arm64] [-tags tags]
                                 Compile to a standalone game binary (Flip presents frames)
  moon transpile <program.mb> [-o out.go]
                                 Translate moonBASIC source to native Go code
  moon build <program.mb> [-o dist] [-os windows|linux|darwin] [-arch amd64|arm64] [-tags tags]
             [-profile dev|debug|release|name] [-pak|-loose] [-headless] [-strict]
             [-name Game] [-version 1.2.3] [-company Co] [-icon icon.png|icon.ico]
                                 Portable package folder: game binary + scripts + assets (or a .pak) + natives.
                                 Defaults come from moon.json; see docs/RELEASE.md
  moon lsp                       Language server (stdio JSON-RPC); same as moonsls
  moon version                   Print version
  moon shader <file.glsl> [stage]  Validate GLSL (optional SPIR-V)
  moon smoke [dir|file] [-frames 180] [-timeout 90s] [-run name]
                                 Run each program hidden for N frames; fail on GL errors, crashes, hangs

Examples live in the examples/ folder.
`)
		return
	}
	if args[0] == "version" || args[0] == "-v" || args[0] == "--version" {
		fmt.Println("moonBASIC", version)
		return
	}
	if args[0] == "lsp" {
		if err := lsp.Serve(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "shader" {
		if err := runShader(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "compile" {
		if err := runCompile(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "transpile" {
		if err := runTranspile(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "edit" {
		if err := runEdit(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "smoke" {
		if err := runSmoke(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "build" {
		if err := runBuild(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "run" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "moon run: missing file")
			os.Exit(2)
		}
		args = args[1:]
	}
	opt := splitRunArgs(args)
	if opt.path == "" {
		fmt.Fprintln(os.Stderr, "moon: missing file")
		os.Exit(2)
	}
	if err := runFile(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type runOpts struct {
	path      string
	debug     bool
	debugJSON bool
	headless  bool
	breaks    []int
}

func splitRunArgs(args []string) runOpts {
	var o runOpts
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-debug":
			o.debug = true
		case "-debugjson":
			o.debug, o.debugJSON = true, true
		case "-headless":
			o.headless = true
		case "-break":
			if i+1 < len(args) {
				i++
				o.debug = true
				for _, part := range strings.Split(args[i], ",") {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					n := 0
					fmt.Sscanf(part, "%d", &n)
					if n > 0 {
						o.breaks = append(o.breaks, n)
					}
				}
			}
		default:
			if o.path == "" && !strings.HasPrefix(args[i], "-") {
				o.path = args[i]
			}
		}
	}
	return o
}

func runFile(opt runOpts) error {
	path := opt.path
	prog, err := parse.ParseFile(path)
	if err != nil {
		return err
	}
	prog, err = parse.ExpandIncludes(prog, filepath.Dir(path))
	if err != nil {
		return err
	}
	world := runtime.New(filepath.Dir(path))
	if opt.headless {
		world.SetHeadless(true)
	}
	world.SetScript(path)
	in := interp.New(prog, world)
	// MOON_SCRIPT_NAME lets the IDE report the tab's name instead of its temp run file.
	name := os.Getenv("MOON_SCRIPT_NAME")
	if name == "" {
		name = filepath.Base(path)
	}
	in.SetScript(name)
	if opt.debug {
		in.SetDebug(opt.breaks, os.Stdin, os.Stderr)
		in.SetDebugJSON(opt.debugJSON)
	}
	world.Bind(in)
	if err := in.Run(); err != nil {
		return err
	}
	if world.Headless() {
		return world.LoopHeadless(in)
	}
	if !world.HasGraphics() {
		return in.Err()
	}
	if in.Done() {
		if world.HasPresented() {
			world.RenderOnce()
		}
		return in.Err()
	}
	return world.Loop(in)
}

// runBoot runs the game a packaged binary was built for. A release build has
// no console on Windows, so its error also goes to <exe name>.log.
func runBoot(args []string) int {
	dir := exeDir()
	opt := splitRunArgs(args)
	opt.path = filepath.Join(dir, bootScript)
	if bootHeadless == "1" {
		opt.headless = true
	}
	if bootProfile == "debug" && os.Getenv("MOON_DEBUG") == "" {
		os.Setenv("MOON_DEBUG", "1")
	}
	err := runFile(opt)
	if err == nil {
		return 0
	}
	fmt.Fprintln(os.Stderr, err)
	if bootProfile == "release" {
		self, _ := os.Executable()
		stem := strings.TrimSuffix(filepath.Base(self), filepath.Ext(self))
		_ = os.WriteFile(filepath.Join(dir, stem+".log"), []byte(err.Error()+"\n"), 0o644)
	}
	return 1
}

func runShader(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("moon shader: missing file")
	}
	stage := "vert"
	if len(args) >= 2 {
		stage = args[1]
	}
	r := glslang.CompileFile(args[0], stage)
	fmt.Println(r.Log)
	if len(r.SPIRV) > 0 {
		fmt.Printf("SPIR-V words: %d\n", len(r.SPIRV))
	} else if !glslang.HasNative() {
		fmt.Println("SPIR-V skipped (install glslangValidator for offline SPIR-V; not required)")
	}
	if !r.OK {
		return fmt.Errorf("shader validate failed")
	}
	return nil
}
