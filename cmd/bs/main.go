package main

import (
	"fmt"
	"os"
	"path/filepath"

	"bitshinbasic/internal/glslang"
	"bitshinbasic/internal/interp"
	"bitshinbasic/internal/lsp"
	"bitshinbasic/internal/parse"
	"bitshinbasic/internal/runtime"
)

const version = "1.0.0"

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(`BitShin BASIC — Blitz3D-style BASIC (G3N 3D, Ebiten 2D)

Usage:
  bs <program.bb>              Run a program
  bs run <program.bb>          Same
  bs build <program.bb> [-o dist] [-os windows|linux|darwin] [-arch amd64|arm64] [-tags tags]
                               Portable folder: binary + .bb + assets + natives.
                               Build on the target OS (CGO). Do not zip the git tree.
  bs lsp                       Language server (stdio JSON-RPC); same as bsls
  bs version                   Print version
  bs shader <file.glsl> [stage]  Validate GLSL (optional SPIR-V)

Examples live in the examples/ folder.
`)
		return
	}
	if args[0] == "version" || args[0] == "-v" || args[0] == "--version" {
		fmt.Println("BitShin BASIC", version)
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
	if args[0] == "build" {
		if err := runBuild(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if args[0] == "run" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "bs run: missing file")
			os.Exit(2)
		}
		args = args[1:]
	}
	if err := runFile(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runFile(path string) error {
	prog, err := parse.ParseFile(path)
	if err != nil {
		return err
	}
	prog, err = parse.ExpandIncludes(prog, filepath.Dir(path))
	if err != nil {
		return err
	}
	world := runtime.New(filepath.Dir(path))
	in := interp.New(prog, world)
	world.Bind(in)
	if err := in.Run(); err != nil {
		return err
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

func runShader(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("bs shader: missing file")
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
