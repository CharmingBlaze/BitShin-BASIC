package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	snapshotDir = "cmd/moon/testdata/render"
	// A pixel differs when a channel moves more than this. Drivers round
	// differently, so references taken on one GPU still pass on another.
	snapshotPixelTol = 24
	// A picture fails when more than this share of its pixels differ.
	snapshotMaxShare = 0.02
)

// runSnapshot draws each scene in the render folder and compares one frame
// with the PNG beside it. Time is fixed and the window hidden, so a repeat
// run draws the same picture. -update rewrites the references.
func runSnapshot(args []string) error {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	update := fs.Bool("update", false, "write new reference images")
	frame := fs.Int("frame", 90, "frame to compare")
	timeout := fs.Duration("timeout", 90*time.Second, "time limit per scene")
	match := fs.String("run", "", "only scenes whose name contains this")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := snapshotDir
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	scenes, _ := filepath.Glob(filepath.Join(dir, "*.mb"))
	sort.Strings(scenes)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	outDir := filepath.Join(os.TempDir(), "moon-snapshots")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	ran := 0
	var bad []string
	for _, scene := range scenes {
		name := strings.TrimSuffix(filepath.Base(scene), ".mb")
		if *match != "" && !strings.Contains(name, *match) {
			continue
		}
		ran++
		actual := filepath.Join(outDir, name+".actual.png")
		if err := drawSnapshot(self, scene, actual, *frame, *timeout); err != nil {
			fmt.Printf("%-8s %-20s %v\n", "CRASH", name, err)
			bad = append(bad, name)
			continue
		}
		ref := strings.TrimSuffix(scene, ".mb") + ".png"
		if *update {
			b, err := os.ReadFile(actual)
			if err == nil {
				err = os.WriteFile(ref, b, 0o644)
			}
			if err != nil {
				return err
			}
			fmt.Printf("%-8s %s\n", "updated", ref)
			continue
		}
		want, err := readPNG(ref)
		if err != nil {
			fmt.Printf("%-8s %-20s no reference; run moon snapshot -update\n", "MISSING", name)
			bad = append(bad, name)
			continue
		}
		got, err := readPNG(actual)
		if err != nil {
			return err
		}
		share, diff, err := compareImages(want, got)
		if err != nil {
			fmt.Printf("%-8s %-20s %v\n", "FAIL", name, err)
			bad = append(bad, name)
			continue
		}
		if share > snapshotMaxShare {
			diffPath := filepath.Join(outDir, name+".diff.png")
			if err := writePNG(diffPath, diff); err != nil {
				return err
			}
			fmt.Printf("%-8s %-20s %.2f%% of pixels differ; see %s\n", "FAIL", name, share*100, diffPath)
			bad = append(bad, name)
			continue
		}
		fmt.Printf("%-8s %-20s %.2f%% of pixels differ\n", "pass", name, share*100)
	}
	if ran == 0 {
		return fmt.Errorf("snapshot: no scenes in %s", dir)
	}
	if len(bad) > 0 {
		return fmt.Errorf("snapshot: %s", strings.Join(bad, ", "))
	}
	return nil
}

// drawSnapshot runs one scene in its own process and leaves frame n at path.
func drawSnapshot(self, scene, path string, frame int, timeout time.Duration) error {
	abs, err := filepath.Abs(scene)
	if err != nil {
		return err
	}
	_ = os.Remove(path)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, self, abs)
	cmd.Dir = filepath.Dir(abs)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MOON_SMOKE_FRAMES=%d", frame+3),
		fmt.Sprintf("MOON_SHOT=%d:%s", frame, path),
		"MOON_FIXED_DT=0.0166667", "MOON_NO_GRAB=1", "MOON_HIDDEN=1")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return errors.New("still running at the limit")
	}
	if err != nil {
		return fmt.Errorf("%v: %s", err, tailLines(out.String(), 4))
	}
	if _, err := os.Stat(path); err != nil {
		return errors.New("ended before the frame was saved")
	}
	return nil
}

// compareImages returns the share of pixels that differ and a picture of them.
func compareImages(want, got image.Image) (float64, *image.RGBA, error) {
	b := want.Bounds()
	if b.Size() != got.Bounds().Size() {
		return 0, nil, fmt.Errorf("size is %v, reference is %v", got.Bounds().Size(), b.Size())
	}
	g := got.Bounds()
	diff := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	differ := 0
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			w8 := color.RGBAModel.Convert(want.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
			g8 := color.RGBAModel.Convert(got.At(g.Min.X+x, g.Min.Y+y)).(color.RGBA)
			if channelGap(w8.R, g8.R) > snapshotPixelTol || channelGap(w8.G, g8.G) > snapshotPixelTol || channelGap(w8.B, g8.B) > snapshotPixelTol {
				differ++
				diff.SetRGBA(x, y, color.RGBA{255, 0, 255, 255})
			} else {
				diff.SetRGBA(x, y, color.RGBA{w8.R / 3, w8.G / 3, w8.B / 3, 255})
			}
		}
	}
	return float64(differ) / float64(b.Dx()*b.Dy()), diff, nil
}

func channelGap(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
