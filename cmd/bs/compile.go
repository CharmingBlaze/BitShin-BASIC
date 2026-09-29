package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"bitshinbasic/internal/parse"
	"bitshinbasic/internal/transpile"
)

// runCompile compiles a BlitzBasic program directly into a native machine-code standalone binary.
func runCompile(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("bs compile: missing program.bb")
	}
	src := args[0]
	baseName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH
	buildTags := ""
	outPath := baseName
	if targetOS == "windows" {
		outPath += ".exe"
	}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-o":
			if i+1 < len(args) {
				outPath = args[i+1]
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

	if targetOS == "wasm" {
		outDir := filepath.Dir(outPath)
		if outDir == "." {
			outDir = "dist"
		}
		return buildWasmBundle(src, outDir)
	}

	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	absOut, err := filepath.Abs(outPath)
	if err != nil {
		return err
	}

	root := findGoModRoot(".")
	if root == "" {
		root = findGoModRoot(exeDir())
	}
	if root == "" {
		root = "."
	}

	// 1. Parse & expand includes
	prog, err := parse.ParseFile(absSrc)
	if err != nil {
		return fmt.Errorf("compile error: %w", err)
	}
	prog, err = parse.ExpandIncludes(prog, filepath.Dir(absSrc))
	if err != nil {
		return fmt.Errorf("compile error in includes: %w", err)
	}

	// 2. Transpile AST to native Go
	goCode, err := transpile.Transpile(prog, transpile.Options{
		PackageName:  "main",
		IsExecutable: true,
		SourceFile:   filepath.Base(src),
	})
	if err != nil {
		return fmt.Errorf("transpile error: %w", err)
	}

	// 3. Stage build package in scratch/.compile
	buildDir := filepath.Join(root, "scratch", ".compile")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return err
	}
	mainGo := filepath.Join(buildDir, "main.go")
	if err := os.WriteFile(mainGo, []byte(goCode), 0o644); err != nil {
		return err
	}

	// 4. Ensure destination directory exists
	destDir := filepath.Dir(absOut)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	// 5. Invoke Go compiler backend
	gocmd := []string{"build", "-o", absOut}
	if buildTags != "" {
		gocmd = append(gocmd, "-tags", buildTags)
	}
	gocmd = append(gocmd, "./scratch/.compile")

	cmd := exec.Command("go", gocmd...)
	cmd.Dir = root
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

	fmt.Printf("Compiling %s -> %s (native machine code AOT)...\n", src, outPath)
	if err := cmd.Run(); err != nil {
		if targetOS != runtime.GOOS || targetArch != runtime.GOARCH {
			return fmt.Errorf("bs compile: cross-compile failed for %s/%s: %w", targetOS, targetArch, err)
		}
		return fmt.Errorf("bs compile: failed: %w", err)
	}

	// 6. Copy runtime natives (DLLs, so, dylib) next to executable
	if err := copyRuntimeNatives(destDir, targetOS); err != nil {
		fmt.Fprintf(os.Stderr, "bs compile: natives warning (%s): %v\n", targetOS, err)
	}

	// 7. Copy referenced assets next to executable
	seen := map[string]bool{}
	_ = collectAssets(absSrc, filepath.Dir(absSrc), destDir, seen)
	assetsDir := filepath.Join(filepath.Dir(absSrc), "assets")
	if st, err := os.Stat(assetsDir); err == nil && st.IsDir() {
		_ = copyDir(assetsDir, filepath.Join(destDir, "assets"))
	}

	fmt.Printf("Successfully compiled native standalone executable: %s\n", absOut)
	return nil
}

// runTranspile translates a BlitzBasic script to Go source code without invoking the Go compiler.
func runTranspile(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("bs transpile: missing program.bb")
	}
	src := args[0]
	outPath := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outPath = args[i+1]
			i++
		}
	}
	if outPath == "" {
		outPath = strings.TrimSuffix(src, filepath.Ext(src)) + ".go"
	}

	prog, err := parse.ParseFile(src)
	if err != nil {
		return fmt.Errorf("transpile: parse error: %w", err)
	}
	prog, err = parse.ExpandIncludes(prog, filepath.Dir(src))
	if err != nil {
		return fmt.Errorf("transpile: include error: %w", err)
	}

	goCode, err := transpile.Transpile(prog, transpile.Options{
		PackageName:  "main",
		IsExecutable: true,
		SourceFile:   filepath.Base(src),
	})
	if err != nil {
		return fmt.Errorf("transpile: %w", err)
	}

	if err := os.WriteFile(outPath, []byte(goCode), 0o644); err != nil {
		return err
	}
	fmt.Printf("Transpiled %s -> %s\n", src, outPath)
	return nil
}
