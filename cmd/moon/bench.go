package main

import (
	"archive/zip"
	"bufio"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed bench_scene.mb
var benchScene string

const (
	benchWarmup   = 5.0  // seconds of a run left out: loading settles, shaders finish
	benchSteady   = 20.0 // ms at the 95th percentile a tier must hold to be recommended
	benchPlayable = 33.3
	benchHitch    = 33.3
	benchTooSlow  = 100.0 // ms median: no use trying a higher tier
	benchUltraMax = 12.0  // ms median on high before ultra is worth the time
)

// benchFrame is one line of the runtime's frame log (MOON_FRAMELOG).
type benchFrame struct {
	ms     float64
	tier   string
	scale  float64
	errors int
}

type benchLog struct {
	gpu, gl string
	frames  []benchFrame
}

func parseFrameLog(r io.Reader) benchLog {
	var l benchLog
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "gpu "):
			l.gpu = line[4:]
		case strings.HasPrefix(line, "gl "):
			l.gl = line[3:]
		default:
			f := strings.Fields(line)
			if len(f) != 4 {
				continue
			}
			ms, err := strconv.ParseFloat(f[0], 64)
			if err != nil {
				continue
			}
			scale, _ := strconv.ParseFloat(f[2], 64)
			errs, _ := strconv.Atoi(f[3])
			l.frames = append(l.frames, benchFrame{ms, f[1], scale, errs})
		}
	}
	return l
}

type benchSummary struct {
	frames                  int
	avg, median, p95, p99   float64
	worst                   float64
	hitches                 int
	startTier, endTier      string
	endScale                float64
	glErrors                int
}

// summarize leaves out the first warmup seconds. A run too short to have any after
// that is summarized whole, so a very slow machine still gets numbers.
func summarize(frames []benchFrame, warmup float64) benchSummary {
	var s benchSummary
	if len(frames) == 0 {
		return s
	}
	s.startTier = frames[0].tier
	last := frames[len(frames)-1]
	s.endTier, s.endScale, s.glErrors = last.tier, last.scale, last.errors
	start, sum := 0, 0.0
	for i, f := range frames {
		if sum >= warmup*1000 {
			start = i
			break
		}
		sum += f.ms
	}
	if sum < warmup*1000 {
		start = 0
	}
	kept := frames[start:]
	ms := make([]float64, len(kept))
	total := 0.0
	for i, f := range kept {
		ms[i] = f.ms
		total += f.ms
		if f.ms > benchHitch {
			s.hitches++
		}
	}
	sort.Float64s(ms)
	s.frames = len(ms)
	s.avg = total / float64(len(ms))
	s.median = percentile(ms, 0.5)
	s.p95 = percentile(ms, 0.95)
	s.p99 = percentile(ms, 0.99)
	s.worst = ms[len(ms)-1]
	return s
}

// percentile reads a sorted list.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(p*float64(len(sorted)-1) + 0.5)
	return sorted[i]
}

type benchResult struct {
	name    string
	summary benchSummary
	loadSec float64
	failure string
}

// bestBenchTier is the highest tier that holds steady, or failing that the highest
// that is playable, in the order they are given: lowest first.
func bestBenchTier(results []benchResult) (name, how string) {
	for _, limit := range []struct {
		ms  float64
		how string
	}{{benchSteady, "steady"}, {benchPlayable, "playable, not steady"}} {
		for i := len(results) - 1; i >= 0; i-- {
			r := results[i]
			if r.failure == "" && r.summary.frames > 0 && r.name != "auto" && r.summary.p95 <= limit.ms {
				return r.name, limit.how
			}
		}
	}
	return "", ""
}

func runBench(args []string) error {
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	out := fs.String("o", "bench-report.txt", "where to write the report")
	seconds := fs.Int("seconds", 30, "seconds to walk on each tier")
	pack := fs.String("pack", "", "write a zip to run this on another computer, and stop")
	hidden := fs.Bool("hidden", false, "hide the window (for tests: timings are not representative)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *pack != "" {
		return packBench(*pack)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "moon-bench")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	fmt.Printf("Walking the island for %d seconds on each tier, about %d minutes. Leave the\ncomputer alone: other programs change the numbers.\n\n",
		*seconds, (*seconds+15)*5/60+1)
	var results []benchResult
	var info benchLog
	run := func(name string) benchResult {
		fmt.Printf("%-7s ", name)
		r, l := benchOne(self, dir, name, *seconds, *hidden)
		if l.gpu != "" {
			info = l
		}
		if r.failure != "" {
			fmt.Println("FAILED:", firstLine(r.failure))
		} else {
			fmt.Printf("%.1f ms median, %.1f ms at the 95th percentile\n", r.summary.median, r.summary.p95)
		}
		results = append(results, r)
		return r
	}
	for _, tier := range []string{"low", "medium", "high"} {
		r := run(tier)
		if r.failure == "" && r.summary.median > benchTooSlow {
			fmt.Println("Too slow to be worth trying higher tiers.")
			break
		}
		if tier == "high" && r.failure == "" && r.summary.median < benchUltraMax {
			run("ultra")
		}
	}
	run("auto")

	report := benchReport(info, results, *seconds)
	fmt.Println()
	fmt.Print(report)
	if err := os.WriteFile(*out, []byte(report), 0o644); err != nil {
		return err
	}
	abs, _ := filepath.Abs(*out)
	fmt.Printf("\nSaved to %s\n", abs)
	return nil
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	return s
}

// benchOne runs the scene in a process of its own, so no tier inherits another's state.
func benchOne(self, dir, name string, seconds int, hidden bool) (benchResult, benchLog) {
	r := benchResult{name: name}
	setup := "SetGraphicsQuality(\"" + name + "\")\nSetDynamicResolution(False)"
	if name == "auto" {
		setup = "" // what a game gets that never asks
	}
	scene := strings.NewReplacer("@SETUP@", setup, "@NAME@", name, "@SECONDS@", strconv.Itoa(seconds)).Replace(benchScene)
	path := filepath.Join(dir, name+".mb")
	if err := os.WriteFile(path, []byte(scene), 0o644); err != nil {
		r.failure = err.Error()
		return r, benchLog{}
	}
	logPath := filepath.Join(dir, name+".log")
	limit := time.Duration(seconds)*time.Second + 5*time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, self, path)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "MOON_FRAMELOG="+logPath, "MOON_NO_GRAB=1")
	if hidden {
		cmd.Env = append(cmd.Env, "MOON_HIDDEN=1")
	}
	started := time.Now()
	output, runErr := cmd.CombinedOutput()
	var l benchLog
	if f, err := os.Open(logPath); err == nil {
		l = parseFrameLog(f)
		f.Close()
	}
	r.summary = summarize(l.frames, benchWarmup)
	total := 0.0
	for _, f := range l.frames {
		total += f.ms
	}
	r.loadSec = time.Since(started).Seconds() - total/1000
	switch {
	case ctx.Err() != nil:
		r.failure = fmt.Sprintf("no finish after %v", limit)
	case runErr != nil:
		r.failure = strings.TrimSpace(lastChars(string(output), 400) + "\n" + runErr.Error())
	case len(l.frames) == 0:
		r.failure = "drew no frames: " + lastChars(string(output), 400)
	}
	return r, l
}

func lastChars(s string, n int) string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

func benchReport(info benchLog, results []benchResult, seconds int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "moonBASIC benchmark, %s\n\n", time.Now().Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "GPU:     %s\n", orUnknown(info.gpu))
	fmt.Fprintf(&b, "OpenGL:  %s\n", orUnknown(info.gl))
	cpu := os.Getenv("PROCESSOR_IDENTIFIER")
	fmt.Fprintf(&b, "CPU:     %d threads %s\n", runtime.NumCPU(), cpu)
	fmt.Fprintf(&b, "System:  %s %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "Test:    1280x720, %d seconds walking the island, dynamic resolution off except on auto.\n", seconds)
	fmt.Fprintf(&b, "         The first %.0f seconds of each are left out. Hitches are frames over %.1f ms.\n\n", benchWarmup, benchHitch)
	fmt.Fprintf(&b, "%-7s %7s %7s %7s %7s %7s %8s %7s %6s %7s\n", "Tier", "Avg", "Median", "95%", "99%", "Worst", "Hitches", "Frames", "Load", "GL errs")
	for _, r := range results {
		if r.failure != "" {
			fmt.Fprintf(&b, "%-7s FAILED: %s\n", r.name, firstLine(r.failure))
			continue
		}
		s := r.summary
		fmt.Fprintf(&b, "%-7s %7.1f %7.1f %7.1f %7.1f %7.1f %8d %7d %5.0fs %7d\n",
			r.name, s.avg, s.median, s.p95, s.p99, s.worst, s.hitches, s.frames, r.loadSec, s.glErrors)
	}
	b.WriteString("\n")
	if name, how := bestBenchTier(results); name != "" {
		fmt.Fprintf(&b, "Best tier: %s (%s)\n", name, how)
	} else {
		b.WriteString("Best tier: none held 30 fps\n")
	}
	for _, r := range results {
		if r.name == "auto" && r.failure == "" {
			s := r.summary
			fmt.Fprintf(&b, "Auto:      started on %s, ended on %s", s.startTier, s.endTier)
			if s.endScale > 0 {
				fmt.Fprintf(&b, " at %.0f%% resolution", s.endScale*100)
			}
			b.WriteString("\n")
		}
	}
	for _, r := range results {
		if r.failure != "" && strings.Contains(r.failure, "\n") {
			fmt.Fprintf(&b, "\n%s failed:\n%s\n", r.name, r.failure)
		}
	}
	return b.String()
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

const benchBat = "@echo off\r\ncd /d \"%~dp0\"\r\nmoon.exe bench\r\necho.\r\necho Please send bench-report.txt back.\r\npause\r\n"

// packBench writes a zip holding all another computer needs: this program, the DLLs it
// needs on Windows, and a file to double-click.
func packBench(path string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	add := func(name, src string) error {
		w, err := z.Create(name)
		if err != nil {
			return err
		}
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	}
	if runtime.GOOS != "windows" {
		if err := add("moon", self); err != nil {
			return err
		}
		if err := z.Close(); err != nil {
			return err
		}
		fmt.Printf("Wrote %s. On the other computer, unzip it and run: ./moon bench\n", path)
		return nil
	}
	if err := add("moon.exe", self); err != nil {
		return err
	}
	dirs := windowsDLLSearchDirs()
	for _, name := range windowsRuntimeDLLs {
		src := findNamedFile(dirs, name)
		if src == "" {
			return fmt.Errorf("not found: %s", name)
		}
		if err := add(name, src); err != nil {
			return err
		}
	}
	w, err := z.Create("run-bench.bat")
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(benchBat)); err != nil {
		return err
	}
	if err := z.Close(); err != nil {
		return err
	}
	fmt.Printf("Wrote %s. On the other computer: unzip it, double-click run-bench.bat, and send back bench-report.txt.\n", path)
	return nil
}
