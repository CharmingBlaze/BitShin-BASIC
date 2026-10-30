package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// runSmoke runs every program in a folder (or one file) for a few hundred
// frames in a hidden window and reports GL errors, crashes, and hangs. Each
// runs in its own process from its own folder, so relative assets load.
func runSmoke(args []string) error {
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	frames := fs.Int("frames", 180, "frames to run each program")
	timeout := fs.Duration("timeout", 90*time.Second, "time limit per program")
	match := fs.String("run", "", "only programs whose name contains this")
	if err := fs.Parse(args); err != nil {
		return err
	}
	target := "examples"
	if fs.NArg() > 0 {
		target = fs.Arg(0)
	}
	var files, every []string // every: the whole folder, to find what loads what
	if st, err := os.Stat(target); err != nil {
		return err
	} else if st.IsDir() {
		every, _ = filepath.Glob(filepath.Join(target, "*.mb"))
		for _, f := range every {
			if *match == "" || strings.Contains(filepath.Base(f), *match) {
				files = append(files, f)
			}
		}
		sort.Strings(files)
	} else {
		files = []string{target}
	}
	if len(files) == 0 {
		return fmt.Errorf("smoke: no .mb files in %s", target)
	}
	parts := loadedParts(every)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	var bad []string
	counts := map[string]int{}
	for _, f := range files {
		if by := parts[filepath.Base(f)]; by != "" {
			counts["part"]++
			fmt.Printf("%-8s %-28s loaded by %s\n", "part", filepath.Base(f), by)
			continue
		}
		abs, _ := filepath.Abs(f)
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		cmd := exec.CommandContext(ctx, self, abs)
		cmd.Dir = filepath.Dir(abs)
		cmd.Env = append(os.Environ(), fmt.Sprintf("MOON_SMOKE_FRAMES=%d", *frames), "MOON_NO_GRAB=1", "MOON_HIDDEN=1")
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		start := time.Now()
		err := cmd.Run()
		timedOut := ctx.Err() == context.DeadlineExceeded
		cancel()
		text := out.String()
		status, detail := "pass", ""
		var exit *exec.ExitError
		switch {
		case timedOut:
			status = "TIMEOUT"
			detail = "still running at the limit (waiting for input, or no Flip)"
		case errors.As(err, &exit) && exit.ExitCode() == 3:
			status = "GLERROR"
			detail = grepLines(text, "GL_", 4)
		case err != nil:
			status = "CRASH"
			detail = tailLines(text, 4)
		case !strings.Contains(text, "SMOKE frames="):
			status = "ended" // finished on its own before the budget
		}
		counts[status]++
		fmt.Printf("%-8s %-28s %5.1fs\n", status, filepath.Base(f), time.Since(start).Seconds())
		if detail != "" {
			fmt.Println("         " + strings.ReplaceAll(detail, "\n", "\n         "))
		}
		if status != "pass" && status != "ended" {
			bad = append(bad, filepath.Base(f))
		}
	}
	fmt.Printf("\n%d files: %d pass, %d ended early, %d GL errors, %d crashed, %d timed out, %d loaded by others\n",
		len(files), counts["pass"], counts["ended"], counts["GLERROR"], counts["CRASH"], counts["TIMEOUT"], counts["part"])
	if len(bad) > 0 {
		return fmt.Errorf("smoke: %s", strings.Join(bad, ", "))
	}
	return nil
}

// loadedParts maps a file another program names in quotes (LoadScene,
// Include, Import) to that program: a scene chunk is not run on its own.
func loadedParts(files []string) map[string]string {
	src := map[string]string{}
	for _, f := range files {
		b, _ := os.ReadFile(f)
		src[filepath.Base(f)] = string(b)
	}
	parts := map[string]string{}
	for _, f := range files {
		name := filepath.Base(f)
		for other, text := range src {
			if other != name && strings.Contains(text, `"`+name+`"`) {
				parts[name] = other
				break
			}
		}
	}
	return parts
}

func grepLines(s, sub string, max int) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			out = append(out, strings.TrimSpace(l))
			if len(out) == max {
				break
			}
		}
	}
	return strings.Join(out, "\n")
}

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
