package runtime

import (
	"fmt"
	"time"

	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
)

// g3nHost is the G3N window + renderer without G3N's OpenAL app.App().
type g3nHost struct {
	window.IWindow
	rend           *renderer.Renderer
	beforeDestroy  func()
	onFirstPresent func()
	stop           bool
}

func startG3N(title string, width, height int) (host *g3nHost, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Graphics3D: %v", r)
		}
	}()
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	if title == "" {
		title = "BitShin BASIC"
	}
	// Ebiten registers Win32 class GLFW30 at package init. G3N's go-gl
	// GLFW is a second copy and glfw.Init then fails with "Class already exists".
	// G3N window.Init requests OpenGL 3.3 core (never 4.5). Optional 4.x
	// features are queried after the context exists — see detectModernGL.
	releaseForeignGLFWClass()
	if err := window.Init(width, height, title); err != nil {
		return nil, err
	}
	iw := window.Get()
	if gw, ok := iw.(*window.GlfwWindow); ok {
		gw.SetShouldClose(false)
	}
	r := renderer.NewRenderer(iw.Gls())
	if err := r.AddDefaultShaders(); err != nil {
		return nil, fmt.Errorf("Graphics3D: %w", err)
	}
	registerShadowShaders(r)
	registerWaterShaders(r)
	registerTerrainShaders(r)
	registerAtmosphereShaders(r)
	// G3N's C glapi wrapper calls exit(1) on any GL error (1281/1282).
	// Weather / post / shadows use raw GL; leftover state must not End the demo.
	if gs := iw.Gls(); gs != nil {
		gs.SetCheckErrors(false)
	}
	return &g3nHost{IWindow: iw, rend: r}, nil
}

func (a *g3nHost) Run(update func(rend *renderer.Renderer, dt time.Duration)) {
	last := time.Now()
	swapped := false
	for {
		gw, ok := a.IWindow.(*window.GlfwWindow)
		if !ok {
			fmt.Println("Graphics3D: window is not GLFW")
			break
		}
		if a.stop {
			break
		}
		// Do not destroy the window because GLFW set ShouldClose. Creation
		// leftovers are cleared here; after the first present, World.Exit()
		// is the only stop (script End, Esc, or a real window X).
		if !swapped && gw.ShouldClose() {
			gw.SetShouldClose(false)
		}
		now := time.Now()
		update(a.rend, now.Sub(last))
		last = now
		gw.SwapBuffers()
		gw.PollEvents()
		if !swapped {
			swapped = true
			if a.onFirstPresent != nil {
				a.onFirstPresent()
			}
		}
	}
	if a.beforeDestroy != nil {
		a.beforeDestroy()
	}
	a.Destroy()
}

// swapOnce presents one frame. Compiled games call this from Flip instead of Run.
func (a *g3nHost) swapOnce() {
	if a == nil || a.IWindow == nil {
		return
	}
	gw, ok := a.IWindow.(*window.GlfwWindow)
	if !ok {
		return
	}
	gw.SwapBuffers()
	gw.PollEvents()
}

func (a *g3nHost) Exit() {
	a.stop = true
	if gw, ok := a.IWindow.(*window.GlfwWindow); ok {
		gw.SetShouldClose(true)
	}
}
