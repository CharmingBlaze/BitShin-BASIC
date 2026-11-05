package runtime

import (
	"bufio"
	"fmt"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Smoke mode runs a script for a set number of frames in a hidden window and
// exits with its GL error count. `moon smoke` runs every example this way.
//
//	MOON_SMOKE_FRAMES=n  stop after n frames: exit 0 clean, 3 on GL errors
//	MOON_NO_GRAB=1       leave the pointer and focus alone (smoke sets it)
//	MOON_HIDDEN=1        create the 3D window hidden (smoke sets it)
//	MOON_SHOT=n:path     save frame n as a PNG (docs and visual checks)
//	MOON_FIXED_DT=s      every frame lasts s seconds (repeatable snapshots)
//	MOON_CPUPROFILE=path write a Go CPU profile of a smoke run's frames
//	MOON_FRAMELOG=path   write every frame's time, for `moon bench`

// glErrCount counts GL errors drainGL has seen, logged or not.
var glErrCount atomic.Int64

func envOn(name string) bool {
	v := os.Getenv(name)
	return v != "" && v != "0"
}

// smokeFrames is the frame budget, 0 when smoke mode is off.
func smokeFrames() int {
	n, _ := strconv.Atoi(os.Getenv("MOON_SMOKE_FRAMES"))
	if n < 0 {
		return 0
	}
	return n
}

var cpuProfileOnce sync.Once

// startCPUProfile begins MOON_CPUPROFILE at the first frame, so loading a
// scene is left out. Smoke mode writes it out when the frames are done.
func startCPUProfile() {
	path := os.Getenv("MOON_CPUPROFILE")
	if path == "" {
		return
	}
	cpuProfileOnce.Do(func() {
		f, err := os.Create(path)
		if err != nil {
			fmt.Println("MOON_CPUPROFILE:", err)
			return
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Println("MOON_CPUPROFILE:", err)
		}
	})
}

// fixedDelta is the frame length MOON_FIXED_DT asks for, 0 when unset.
func fixedDelta() float64 {
	s, err := strconv.ParseFloat(os.Getenv("MOON_FIXED_DT"), 64)
	if err != nil || s <= 0 {
		return 0
	}
	return s
}

// noGrab is true when a script must not hide, warp, or capture the pointer,
// or take focus: smoke runs and benchmarks share the desktop with a person.
func noGrab() bool { return envOn("MOON_NO_GRAB") || smokeFrames() > 0 }

// hideWindow sets GLFW to create the next window hidden and unfocused. GLFW
// is initialized here first: G3N's own glfw.Init is then a no-op that keeps
// these hints, where a fresh init would reset them.
func hideWindow() {
	if !envOn("MOON_HIDDEN") && smokeFrames() == 0 {
		return
	}
	if err := glfw.Init(); err != nil {
		return
	}
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.FocusOnShow, glfw.False)
	glfw.WindowHint(glfw.Focused, glfw.False)
}

// logFrame appends a line for the frame just shown: its milliseconds, tier, render
// scale, and the GL errors so far. Flushed often, as a script's End exits at once.
func (w *World) logFrame() {
	path := os.Getenv("MOON_FRAMELOG")
	if path == "" {
		return
	}
	now := time.Now()
	if w.frameLog == nil {
		f, err := os.Create(path)
		if err != nil {
			fmt.Println("MOON_FRAMELOG:", err)
			os.Unsetenv("MOON_FRAMELOG")
			return
		}
		w.frameLog = bufio.NewWriter(f)
		w.detectModernGL()
		fmt.Fprintf(w.frameLog, "gpu %s\ngl %s\n", w.glmod.caps.renderer, w.glmod.caps.version)
		w.frameLast = now
		return
	}
	tier := "none"
	if w.quality.set {
		tier = qualityNames[w.quality.tier]
	}
	scale := 1.0
	if w.dynActive() && w.dyn.scale > 0 {
		scale = w.dyn.scale
	}
	fmt.Fprintf(w.frameLog, "%.3f %s %.3f %d\n", now.Sub(w.frameLast).Seconds()*1000, tier, scale, glErrCount.Load())
	w.frameLast = now
	w.frameLogged++
	if w.frameLogged%30 == 0 {
		w.frameLog.Flush()
	}
}

// tickSmoke ends a smoke run once its frames are done.
func (w *World) tickSmoke() {
	if at, path, ok := strings.Cut(os.Getenv("MOON_SHOT"), ":"); ok {
		if n, err := strconv.Atoi(at); err == nil && n == w.loopFrames {
			w.shotPath = w.resolve(path)
		}
	}
	startCPUProfile()
	w.logFrame()
	if w.smokeLeft <= 0 {
		return
	}
	w.smokeLeft--
	if w.smokeLeft > 0 {
		return
	}
	pprof.StopCPUProfile()
	if w.headless {
		fmt.Printf("SMOKE frames=%d headless\n", smokeFrames())
		os.Exit(0)
	}
	if w.app != nil {
		drainGL("smoke")
	}
	n := glErrCount.Load()
	shown := 0
	if gw := w.glfwWin(); gw != nil {
		shown = gw.GetAttrib(glfw.Visible)
	}
	fmt.Printf("SMOKE frames=%d glerrors=%d shown=%d\n", smokeFrames(), n, shown)
	if n > 0 {
		os.Exit(3)
	}
	os.Exit(0)
}
