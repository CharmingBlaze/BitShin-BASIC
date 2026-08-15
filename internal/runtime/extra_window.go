package runtime

import (
	"fmt"
	"runtime"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"bitshinbasic/internal/value"
)

// extraWin is a GLFW window that shares the main G3N OpenGL context.
// The 3D scene is rendered to a texture on the main context (VAOs stay valid),
// then a fullscreen blit runs on the extra context (textures/programs are shared).
type extraWin struct {
	id      int
	win     *glfw.Window
	title   string
	camID   int
	closed  bool
	hidden  bool
	focused bool
	keys    map[int]bool
	fbo     uint32
	color   uint32
	depth   uint32
	rttW    int
	rttH    int
	blitVAO uint32
	blitVBO uint32
	blitOK  bool
}

const (
	winBlitVert = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aUV;
out vec2 vUV;
void main() {
	vUV = aUV;
	gl_Position = vec4(aPos, 0.0, 1.0);
}
`
	winBlitFrag = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uTex;
void main() {
	frag = texture(uTex, vUV);
}
`
)

func (w *World) extraWindowCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	need3 := func(fn func([]value.Value) (value.Value, error)) cmd {
		return func(a []value.Value) (value.Value, error) {
			if err := w.require3D(); err != nil {
				return value.Value{}, err
			}
			return fn(a)
		}
	}
	return map[string]cmd{
		"createwindow": need3(func(a []value.Value) (value.Value, error) {
			id, err := w.createExtraWindow(argI(a, 0, 480), argI(a, 1, 360), argS(a, 2))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(id)), nil
		}),
		"freewindow": need3(func(a []value.Value) (value.Value, error) {
			w.freeExtraWindow(argI(a, 0, 0))
			return z()
		}),
		"deletewindow": need3(func(a []value.Value) (value.Value, error) {
			w.freeExtraWindow(argI(a, 0, 0))
			return z()
		}),
		"showwindow": need3(func(a []value.Value) (value.Value, error) {
			w.showExtraWindow(argI(a, 0, 0), true)
			return z()
		}),
		"hidewindow": need3(func(a []value.Value) (value.Value, error) {
			w.showExtraWindow(argI(a, 0, 0), false)
			return z()
		}),
		"activatewindow": need3(func(a []value.Value) (value.Value, error) {
			w.activateWindow(argI(a, 0, 0))
			return z()
		}),
		"setrenderwindow": need3(func(a []value.Value) (value.Value, error) {
			w.setRenderWindow(argI(a, 0, 0))
			return z()
		}),
		"windowgraphics": need3(func(a []value.Value) (value.Value, error) {
			w.setRenderWindow(argI(a, 0, 0))
			return z()
		}),
		"setwindowcamera": need3(func(a []value.Value) (value.Value, error) {
			if err := w.setWindowCamera(argI(a, 0, 0), argI(a, 1, 0)); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		"getwindowcamera": need3(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.windowCamera(argI(a, 0, 0)))), nil
		}),
		"windowclosed": n(func(a []value.Value) (value.Value, error) {
			if w.windowClosed(argI(a, 0, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"windowkeydown": n(func(a []value.Value) (value.Value, error) {
			if w.windowKeyDown(argI(a, 0, 0), argI(a, 1, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"getwindowwidth": n(func(a []value.Value) (value.Value, error) {
			ww, _ := w.windowSizeOf(argI(a, 0, 0))
			return value.Num(float64(ww)), nil
		}),
		"getwindowheight": n(func(a []value.Value) (value.Value, error) {
			_, hh := w.windowSizeOf(argI(a, 0, 0))
			return value.Num(float64(hh)), nil
		}),
		"getwindowsize": n(func(a []value.Value) (value.Value, error) {
			ww, _ := w.windowSizeOf(argI(a, 0, 0))
			return value.Num(float64(ww)), nil
		}),
		"getwindowx": n(func(a []value.Value) (value.Value, error) {
			x, _ := w.windowPosOf(argI(a, 0, 0))
			return value.Num(float64(x)), nil
		}),
		"getwindowy": n(func(a []value.Value) (value.Value, error) {
			_, y := w.windowPosOf(argI(a, 0, 0))
			return value.Num(float64(y)), nil
		}),
		"getwindowpos": n(func(a []value.Value) (value.Value, error) {
			x, _ := w.windowPosOf(argI(a, 0, 0))
			return value.Num(float64(x)), nil
		}),
		"currentwindow": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.renderWin)), nil
		}),
	}
}

func (w *World) createExtraWindow(width, height int, title string) (int, error) {
	share := w.mainGLFW()
	if share == nil {
		return 0, fmt.Errorf("CreateWindow: Graphics3D window is missing")
	}
	if width <= 0 {
		width = 480
	}
	if height <= 0 {
		height = 360
	}
	if title == "" {
		title = "Window"
	}
	// Same 3.3 core hint as G3N. Do not raise this to 4.5.
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Samples, 8)
	if runtime.GOOS == "darwin" {
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	}
	glfw.WindowHint(glfw.Visible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Focused, glfw.False)
	nw, err := glfw.CreateWindow(width, height, title, nil, share)
	if err != nil {
		return 0, fmt.Errorf("CreateWindow: %w", err)
	}
	nw.SetShouldClose(false)
	mx, my := share.GetPos()
	mw, _ := share.GetSize()
	nw.SetPos(mx+mw+16, my)

	id := w.takeHandle(&w.freeWins, &w.nextWin)
	ew := &extraWin{id: id, win: nw, title: title, keys: map[int]bool{}}
	w.wins[id] = ew
	w.bindExtraWindow(ew)
	share.MakeContextCurrent()
	return id, nil
}

func (w *World) bindExtraWindow(ew *extraWin) {
	ew.win.SetCloseCallback(func(_ *glfw.Window) {
		ew.closed = true
		ew.hidden = true
		ew.win.SetShouldClose(false)
		ew.win.Hide()
	})
	ew.win.SetFocusCallback(func(_ *glfw.Window, focused bool) {
		ew.focused = focused
		if focused {
			w.focusWin = ew.id
		} else if w.focusWin == ew.id {
			w.focusWin = 0
			for code, down := range ew.keys {
				if down {
					w.keys[code] = false
				}
			}
		}
	})
	ew.win.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		gk := window.Key(key)
		code := w.blitzKey(gk)
		if key == glfw.KeyEscape {
			code = KeyEscape
		}
		down := action != glfw.Release
		if code != 0 {
			ew.keys[code] = down
			if ew.focused || w.focusWin == ew.id {
				w.setKey(gk, down)
			}
		}
	})
	ew.win.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		if ew.focused || w.focusWin == ew.id {
			w.mx, w.my = float32(x), float32(y)
			w.fireHook(w.onMouseFn, value.Num(x), value.Num(y))
		}
	})
	ew.win.SetMouseButtonCallback(func(win *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if !(ew.focused || w.focusWin == ew.id) {
			return
		}
		b := int(button) + 1
		down := action == glfw.Press
		if b >= 0 && b < len(w.mouse) {
			w.mouse[b] = down
			if down {
				w.mouseHits[b] = true
			}
		}
		xpos, ypos := win.GetCursorPos()
		w.mx, w.my = float32(xpos), float32(ypos)
	})
	ew.win.SetScrollCallback(func(_ *glfw.Window, _, yoff float64) {
		if ew.focused || w.focusWin == ew.id {
			w.mz += float32(yoff)
		}
	})
}

func (w *World) freeExtraWindow(id int) {
	if id == 0 {
		return
	}
	ew := w.wins[id]
	if ew == nil {
		return
	}
	w.releaseExtraGL(ew)
	if ew.win != nil {
		ew.win.Destroy()
		ew.win = nil
	}
	delete(w.wins, id)
	w.recycleHandle(&w.freeWins, id)
	if w.renderWin == id {
		w.renderWin = 0
	}
	if w.focusWin == id {
		w.focusWin = 0
	}
	if main := w.mainGLFW(); main != nil {
		main.MakeContextCurrent()
	}
}

func (w *World) destroyExtraWindows() {
	ids := make([]int, 0, len(w.wins))
	for id := range w.wins {
		ids = append(ids, id)
	}
	for _, id := range ids {
		w.freeExtraWindow(id)
	}
	if w.winBlitProg != 0 {
		if main := w.mainGLFW(); main != nil {
			main.MakeContextCurrent()
		}
		gl.DeleteProgram(w.winBlitProg)
		w.winBlitProg = 0
	}
}

func (w *World) releaseExtraGL(ew *extraWin) {
	main := w.mainGLFW()
	if ew.blitOK && ew.win != nil {
		ew.win.MakeContextCurrent()
		if ew.blitVAO != 0 {
			gl.DeleteVertexArrays(1, &ew.blitVAO)
			ew.blitVAO = 0
		}
		if ew.blitVBO != 0 {
			gl.DeleteBuffers(1, &ew.blitVBO)
			ew.blitVBO = 0
		}
		ew.blitOK = false
	}
	if main != nil {
		main.MakeContextCurrent()
	}
	if ew.fbo != 0 {
		gl.DeleteFramebuffers(1, &ew.fbo)
		ew.fbo = 0
	}
	if ew.color != 0 {
		gl.DeleteTextures(1, &ew.color)
		ew.color = 0
	}
	if ew.depth != 0 {
		gl.DeleteRenderbuffers(1, &ew.depth)
		ew.depth = 0
	}
	ew.rttW, ew.rttH = 0, 0
}

func (w *World) extraOf(id int) *extraWin {
	if id <= 0 {
		return nil
	}
	return w.wins[id]
}

func (w *World) showExtraWindow(id int, show bool) {
	if id == 0 {
		if gw := w.glfwWin(); gw != nil {
			if show {
				gw.Restore()
				gw.Focus()
			} else {
				gw.Iconify()
			}
		}
		return
	}
	ew := w.extraOf(id)
	if ew == nil || ew.win == nil {
		return
	}
	if show {
		ew.hidden = false
		ew.closed = false
		ew.win.SetShouldClose(false)
		ew.win.Show()
		return
	}
	ew.hidden = true
	ew.win.Hide()
}

func (w *World) activateWindow(id int) {
	w.setRenderWindow(id)
	if id == 0 {
		if gw := w.glfwWin(); gw != nil {
			gw.Focus()
		}
		w.focusWin = 0
		return
	}
	ew := w.extraOf(id)
	if ew == nil || ew.win == nil {
		return
	}
	if ew.hidden || ew.closed {
		w.showExtraWindow(id, true)
	}
	ew.win.Focus()
	w.focusWin = id
}

func (w *World) setRenderWindow(id int) {
	if id != 0 && w.extraOf(id) == nil {
		return
	}
	w.renderWin = id
}

func (w *World) setWindowCamera(winID, camID int) error {
	ew := w.extraOf(winID)
	if ew == nil {
		return fmt.Errorf("SetWindowCamera: invalid window %d", winID)
	}
	if camID == 0 {
		ew.camID = 0
		return nil
	}
	e, err := w.ent(camID)
	if err != nil {
		return err
	}
	if e.cam == nil {
		return fmt.Errorf("SetWindowCamera: %d is not a camera", camID)
	}
	ew.camID = camID
	return nil
}

func (w *World) windowCamera(id int) int {
	if ew := w.extraOf(id); ew != nil {
		return ew.camID
	}
	return 0
}

func (w *World) windowClosed(id int) bool {
	if id == 0 {
		return w.windowWantsClose()
	}
	ew := w.extraOf(id)
	if ew == nil {
		return true
	}
	if ew.win != nil && ew.win.ShouldClose() {
		ew.closed = true
		ew.hidden = true
		ew.win.SetShouldClose(false)
		ew.win.Hide()
	}
	return ew.closed
}

func (w *World) windowKeyDown(winID, code int) bool {
	if winID == 0 {
		return w.keyDown(code)
	}
	if (code == KeyEscape || code == KeySpace) && !w.presented {
		return false
	}
	if w.heldAtFlip[code] {
		return false
	}
	ew := w.extraOf(winID)
	if ew == nil || ew.win == nil || ew.closed {
		return false
	}
	if gk, ok := dikToKey[code]; ok {
		return ew.win.GetKey(glfw.Key(gk)) == glfw.Press
	}
	if code == KeyEscape {
		return ew.win.GetKey(glfw.KeyEscape) == glfw.Press
	}
	return ew.keys[code]
}

func (w *World) windowSizeOf(id int) (int, int) {
	if id == 0 {
		if gw := w.glfwWin(); gw != nil {
			return gw.GetSize()
		}
		return w.scrW, w.scrH
	}
	ew := w.extraOf(id)
	if ew == nil || ew.win == nil {
		return 0, 0
	}
	return ew.win.GetSize()
}

func (w *World) windowPosOf(id int) (int, int) {
	if id == 0 {
		if gw := w.glfwWin(); gw != nil {
			return gw.GetPos()
		}
		return 0, 0
	}
	ew := w.extraOf(id)
	if ew == nil || ew.win == nil {
		return 0, 0
	}
	return ew.win.GetPos()
}

func (w *World) applyWindowTitle(id int, title string) bool {
	if id == 0 {
		w.title = title
		if gw := w.glfwWin(); gw != nil {
			gw.SetTitle(title)
		}
		return true
	}
	ew := w.extraOf(id)
	if ew == nil || ew.win == nil {
		return false
	}
	ew.title = title
	ew.win.SetTitle(title)
	return true
}

func (w *World) applyWindowSize(id, ww, hh int) bool {
	if ww <= 0 || hh <= 0 {
		return false
	}
	if id == 0 {
		if gw := w.glfwWin(); gw != nil {
			gw.SetSize(ww, hh)
		}
		w.scrW, w.scrH = ww, hh
		return true
	}
	ew := w.extraOf(id)
	if ew == nil || ew.win == nil {
		return false
	}
	ew.win.SetSize(ww, hh)
	return true
}

func (w *World) applyWindowPos(id, x, y int) bool {
	if id == 0 {
		if gw := w.glfwWin(); gw != nil {
			gw.SetPos(x, y)
		}
		return true
	}
	ew := w.extraOf(id)
	if ew == nil || ew.win == nil {
		return false
	}
	ew.win.SetPos(x, y)
	return true
}

func (w *World) mainGLFW() *glfw.Window {
	gw := w.glfwWin()
	if gw == nil {
		return nil
	}
	return gw.Window
}

func (w *World) extraCam(ew *extraWin) *camera.Camera {
	if ew.camID != 0 {
		if e := w.ents[ew.camID]; e != nil && e.cam != nil {
			return e.cam
		}
	}
	return w.cam
}

func (w *World) presentExtraWindows(rend *renderer.Renderer) {
	if len(w.wins) == 0 || w.app == nil || w.scene == nil {
		return
	}
	main := w.mainGLFW()
	if main == nil {
		return
	}
	if err := gl.Init(); err != nil {
		fmt.Println("CreateWindow: gl.Init", err)
		return
	}
	if err := w.ensureWinBlitProg(); err != nil {
		fmt.Println("CreateWindow:", err)
		return
	}
	cr, cg, cb := w.clear.R, w.clear.G, w.clear.B
	if w.fogMode != 0 && !w.skyVisible() {
		cr, cg, cb = w.fogRGB.R, w.fogRGB.G, w.fogRGB.B
	}
	if fr, fg, fb, ok := w.weatherClearMix(); ok {
		cr, cg, cb = cr+fr, cg+fg, cb+fb
	}

	for _, ew := range w.wins {
		if ew == nil || ew.win == nil || ew.hidden {
			continue
		}
		if ew.win.ShouldClose() {
			ew.closed = true
			ew.hidden = true
			ew.win.SetShouldClose(false)
			ew.win.Hide()
			continue
		}
		fbw, fbh := ew.win.GetFramebufferSize()
		if fbw <= 0 || fbh <= 0 {
			continue
		}
		if err := w.ensureExtraRTT(ew, fbw, fbh); err != nil {
			fmt.Println("CreateWindow:", err)
			continue
		}
		main.MakeContextCurrent()
		gl.BindFramebuffer(gl.FRAMEBUFFER, ew.fbo)
		gl.Viewport(0, 0, int32(fbw), int32(fbh))
		w.app.Gls().ClearColor(cr, cg, cb, 1)
		w.app.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		if cam := w.extraCam(ew); cam != nil && rend != nil {
			cam.SetAspect(float32(fbw) / float32(fbh))
			if err := rend.Render(w.scene, cam); err != nil {
				fmt.Println("RenderWorld:", err)
			}
		}
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

		ew.win.MakeContextCurrent()
		if err := w.ensureExtraBlit(ew); err != nil {
			fmt.Println("CreateWindow:", err)
			main.MakeContextCurrent()
			continue
		}
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.Viewport(0, 0, int32(fbw), int32(fbh))
		gl.Disable(gl.DEPTH_TEST)
		gl.Disable(gl.BLEND)
		gl.UseProgram(w.winBlitProg)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, ew.color)
		gl.Uniform1i(gl.GetUniformLocation(w.winBlitProg, gl.Str("uTex\x00")), 0)
		gl.BindVertexArray(ew.blitVAO)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
		gl.BindVertexArray(0)
		gl.BindTexture(gl.TEXTURE_2D, 0)
		gl.UseProgram(0)
		ew.win.SwapBuffers()
	}

	main.MakeContextCurrent()
	if ww, hh := w.app.GetSize(); ww > 0 && hh > 0 {
		gl.Viewport(0, 0, int32(ww), int32(hh))
	}
}

func (w *World) ensureWinBlitProg() error {
	if w.winBlitProg != 0 {
		return nil
	}
	vert, err := compileGLShader(gl.VERTEX_SHADER, winBlitVert)
	if err != nil {
		return fmt.Errorf("window blit: %w", err)
	}
	frag, err := compileGLShader(gl.FRAGMENT_SHADER, winBlitFrag)
	if err != nil {
		gl.DeleteShader(vert)
		return fmt.Errorf("window blit: %w", err)
	}
	prog := gl.CreateProgram()
	gl.AttachShader(prog, vert)
	gl.AttachShader(prog, frag)
	gl.LinkProgram(prog)
	var ok int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &ok)
	gl.DeleteShader(vert)
	gl.DeleteShader(frag)
	if ok == gl.FALSE {
		log := glInfoLog(prog, true)
		gl.DeleteProgram(prog)
		return fmt.Errorf("window blit link: %s", log)
	}
	w.winBlitProg = prog
	return nil
}

func (w *World) ensureExtraRTT(ew *extraWin, width, height int) error {
	if ew.fbo != 0 && ew.rttW == width && ew.rttH == height {
		return nil
	}
	if ew.fbo != 0 {
		gl.DeleteFramebuffers(1, &ew.fbo)
		ew.fbo = 0
	}
	if ew.color != 0 {
		gl.DeleteTextures(1, &ew.color)
		ew.color = 0
	}
	if ew.depth != 0 {
		gl.DeleteRenderbuffers(1, &ew.depth)
		ew.depth = 0
	}
	gl.GenTextures(1, &ew.color)
	gl.BindTexture(gl.TEXTURE_2D, ew.color)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.GenRenderbuffers(1, &ew.depth)
	gl.BindRenderbuffer(gl.RENDERBUFFER, ew.depth)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, int32(width), int32(height))
	gl.GenFramebuffers(1, &ew.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, ew.fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, ew.color, 0)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.RENDERBUFFER, ew.depth)
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		return fmt.Errorf("window FBO incomplete")
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	ew.rttW, ew.rttH = width, height
	return nil
}

func (w *World) ensureExtraBlit(ew *extraWin) error {
	if ew.blitOK {
		return nil
	}
	quad := []float32{
		-1, -1, 0, 0,
		1, -1, 1, 0,
		-1, 1, 0, 1,
		1, 1, 1, 1,
	}
	gl.GenVertexArrays(1, &ew.blitVAO)
	gl.GenBuffers(1, &ew.blitVBO)
	gl.BindVertexArray(ew.blitVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, ew.blitVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(2*4))
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	ew.blitOK = true
	return nil
}
