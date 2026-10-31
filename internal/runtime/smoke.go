package runtime

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Smoke mode runs a script for a set number of frames in a hidden window and
// exits with its GL error count. `moon smoke` runs every example this way.
//
//	MOON_SMOKE_FRAMES=n  stop after n frames: exit 0 clean, 3 on GL errors
//	MOON_NO_GRAB=1       leave the pointer and focus alone (smoke sets it)
//	MOON_HIDDEN=1        create the 3D window hidden (smoke sets it)
//	MOON_SHOT=n:path     save frame n as a PNG (docs and visual checks)

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

// tickSmoke ends a smoke run once its frames are done.
func (w *World) tickSmoke() {
	if at, path, ok := strings.Cut(os.Getenv("MOON_SHOT"), ":"); ok {
		if n, err := strconv.Atoi(at); err == nil && n == w.loopFrames {
			w.shotPath = w.resolve(path)
		}
	}
	if w.smokeLeft <= 0 {
		return
	}
	w.smokeLeft--
	if w.smokeLeft > 0 {
		return
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
