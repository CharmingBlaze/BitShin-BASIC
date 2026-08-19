package runtime

import (
	"fmt"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/go-gl/gl/v3.3-core/gl"

	"bitshinbasic/internal/value"
)

// Fullscreen post stack on OpenGL 3.3: scene → color FBO → blit
// (tonemap + exposure + cheap bloom + optional FXAA + color grade).
// Not a deferred MRT / Unreal post graph.

type postFX struct {
	on       bool
	userOn   bool
	rainOnly bool
	exposure float32
	bloom    float32
	fxaa     bool
	contrast float32
	sat      float32
	tint     math32.Color
	fbo      uint32
	color    uint32
	depth    uint32
	w, h     int
	prog     uint32
	vao, vbo uint32
	blitOK   bool
}

func (w *World) postCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"enablepostfx": need(func(a []value.Value) (value.Value, error) {
			on := argI(a, 0, 1) != 0
			w.post.on = on
			w.post.userOn = true
			if on {
				if w.post.exposure <= 0 {
					w.post.exposure = 1
				}
				if w.post.contrast <= 0 {
					w.post.contrast = 1
				}
				if w.post.sat <= 0 {
					w.post.sat = 1
				}
				if w.post.tint.R+w.post.tint.G+w.post.tint.B == 0 {
					w.post.tint = math32.Color{1, 1, 1}
				}
				if w.post.bloom == 0 && len(a) < 1 {
					w.post.bloom = 0.22
				}
				w.post.fxaa = true
			}
			return value.Num(float64(b01(w.post.on))), nil
		}),
		"postfx": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(w.post.on))), nil
		}),
		"setexposure": n(func(a []value.Value) (value.Value, error) {
			w.post.exposure = float32(argN(a, 0, 1))
			if w.post.exposure < 0.05 {
				w.post.exposure = 0.05
			}
			if w.post.exposure > 8 {
				w.post.exposure = 8
			}
			return value.Num(float64(w.post.exposure)), nil
		}),
		"getexposure": n(func(a []value.Value) (value.Value, error) {
			if w.post.exposure <= 0 {
				return value.Num(1), nil
			}
			return value.Num(float64(w.post.exposure)), nil
		}),
		"setbloom": n(func(a []value.Value) (value.Value, error) {
			w.post.bloom = float32(argN(a, 0, 0.22))
			if w.post.bloom < 0 {
				w.post.bloom = 0
			}
			if w.post.bloom > 2 {
				w.post.bloom = 2
			}
			return value.Num(float64(w.post.bloom)), nil
		}),
		"getbloom": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.post.bloom)), nil
		}),
		"setfxaa": n(func(a []value.Value) (value.Value, error) {
			w.post.fxaa = argI(a, 0, 1) != 0
			return value.Num(float64(b01(w.post.fxaa))), nil
		}),
		"setcolorgrade": n(func(a []value.Value) (value.Value, error) {
			w.post.contrast = float32(argN(a, 0, 1))
			w.post.sat = float32(argN(a, 1, 1))
			if len(a) >= 5 {
				c := rgb(argN(a, 2, 255), argN(a, 3, 255), argN(a, 4, 255))
				w.post.tint = *c
			}
			return z()
		}),
	}
}

var drainLogged = map[string]bool{}

func drainGL(where string) {
	for i := 0; i < 8; i++ {
		err := gl.GetError()
		if err == gl.NO_ERROR {
			return
		}
		if i == 0 && !drainLogged[where] {
			drainLogged[where] = true
			fmt.Println(where+":", err)
		}
	}
}

func (w *World) beginPostTarget(ww, hh int) bool {
	if !w.post.on || ww <= 0 || hh <= 0 {
		return false
	}
	if w.app != nil && w.app.Gls() != nil {
		// already have a context; skip repeated gl.Init
	} else if err := gl.Init(); err != nil {
		fmt.Println("PostFX:", err)
		return false
	}
	if err := w.ensurePostFX(ww, hh); err != nil {
		fmt.Println("PostFX:", err)
		drainGL("PostFX")
		return false
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.post.fbo)
	drainGL("PostFX bind")
	return w.post.fbo != 0
}

func (w *World) endPostTarget(gs *gls.GLS, ww, hh int) {
	if w.post.fbo == 0 {
		return
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	w.blitPostFX(gs, ww, hh)
}

func (w *World) ensurePostFX(ww, hh int) error {
	if err := w.ensurePostProg(); err != nil {
		return err
	}
	if w.post.fbo != 0 && w.post.w == ww && w.post.h == hh {
		return nil
	}
	w.disposePostFBO()
	var fbo, color, depth uint32
	gl.GenTextures(1, &color)
	gl.BindTexture(gl.TEXTURE_2D, color)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(ww), int32(hh), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.GenRenderbuffers(1, &depth)
	gl.BindRenderbuffer(gl.RENDERBUFFER, depth)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, int32(ww), int32(hh))
	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, color, 0)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.RENDERBUFFER, depth)
	st := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	if st != gl.FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("FBO incomplete (%d)", st)
	}
	w.post.fbo, w.post.color, w.post.depth = fbo, color, depth
	w.post.w, w.post.h = ww, hh
	return w.ensurePostBlit()
}

func (w *World) disposePostFBO() {
	if w.post.color != 0 {
		gl.DeleteTextures(1, &w.post.color)
		w.post.color = 0
	}
	if w.post.depth != 0 {
		gl.DeleteRenderbuffers(1, &w.post.depth)
		w.post.depth = 0
	}
	if w.post.fbo != 0 {
		gl.DeleteFramebuffers(1, &w.post.fbo)
		w.post.fbo = 0
	}
}

func (w *World) ensurePostProg() error {
	if w.post.prog != 0 {
		return nil
	}
	p, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbpostVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbpostFragment},
	)
	if err != nil {
		return fmt.Errorf("program: %w", err)
	}
	w.post.prog = p
	return nil
}

func (w *World) ensurePostBlit() error {
	if w.post.blitOK {
		return nil
	}
	quad := []float32{
		-1, -1, 0, 0,
		1, -1, 1, 0,
		-1, 1, 0, 1,
		1, 1, 1, 1,
	}
	gl.GenVertexArrays(1, &w.post.vao)
	gl.GenBuffers(1, &w.post.vbo)
	gl.BindVertexArray(w.post.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.post.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(2*4))
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	w.post.blitOK = true
	return nil
}

func (w *World) blitPostFX(gs *gls.GLS, ww, hh int) {
	if w.post.prog == 0 || !w.post.blitOK {
		return
	}
	restore := saveModernGL(gs)
	defer restore()
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	if ww > 0 && hh > 0 {
		gl.Viewport(0, 0, int32(ww), int32(hh))
	}
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.UseProgram(w.post.prog)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, w.post.color)
	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uTex\x00")), 0)
	exp := w.post.exposure
	if exp <= 0 {
		exp = 1
	}
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uExposure\x00")), exp)
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uBloom\x00")), w.post.bloom)
	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uFXAA\x00")), int32(b01(w.post.fxaa)))
	con := w.post.contrast
	if con <= 0 {
		con = 1
	}
	sat := w.post.sat
	if sat <= 0 {
		sat = 1
	}
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uContrast\x00")), con)
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uSat\x00")), sat)
	t := w.post.tint
	if t.R+t.G+t.B == 0 {
		t = math32.Color{1, 1, 1}
	}
	gl.Uniform3f(gl.GetUniformLocation(w.post.prog, gl.Str("uTint\x00")), t.R, t.G, t.B)
	gl.Uniform2f(gl.GetUniformLocation(w.post.prog, gl.Str("uTexel\x00")), 1/float32(ww), 1/float32(hh))
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uRain\x00")), w.cameraRainAmount())
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uTime\x00")), float32(w.wx.time))
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uWet\x00")), w.wetness)
	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uRainOnly\x00")), int32(b01(w.post.rainOnly)))
	gl.BindVertexArray(w.post.vao)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.BindVertexArray(0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.UseProgram(0)
	gl.Enable(gl.DEPTH_TEST)
	drainGL("PostFX blit")
}

func (w *World) renderSceneCams(rend *renderer.Renderer, ww, hh int) {
	type camVP struct {
		cam          *camera.Camera
		x, y, vw, vh int
	}
	var list []camVP
	for _, e := range w.ents {
		if e.cam == nil {
			continue
		}
		list = append(list, camVP{e.cam, e.vpX, e.vpY, e.vpW, e.vpH})
	}
	split := false
	for _, c := range list {
		if c.vw > 0 && c.vh > 0 {
			split = true
			break
		}
	}
	targetFBO := uint32(0)
	if w.post.on && w.post.fbo != 0 {
		targetFBO = w.post.fbo
	}
	if !split {
		cam := w.cam
		if cam == nil && len(list) > 0 {
			cam = list[0].cam
		}
		if cam != nil {
			w.renderShadows(rend, cam)
			gl.BindFramebuffer(gl.FRAMEBUFFER, targetFBO)
			drainGL("pre-water")
			w.renderWaterReflection(rend, cam)
			gl.BindFramebuffer(gl.FRAMEBUFFER, targetFBO)
			if ww > 0 && hh > 0 {
				gl.Viewport(0, 0, int32(ww), int32(hh))
			}
			drainGL("scene")
			if err := rend.Render(w.scene, cam); err != nil {
				fmt.Println("RenderWorld:", err)
			} else {
				w.markShadowPrimed()
			}
			w.drawModernPass(cam)
		}
		return
	}
	for _, c := range list {
		vw, vh := c.vw, c.vh
		if vw <= 0 || vh <= 0 {
			vw, vh = ww, hh
		}
		glY := hh - c.y - vh
		if glY < 0 {
			glY = 0
		}
		w.renderShadows(rend, c.cam)
		gl.BindFramebuffer(gl.FRAMEBUFFER, targetFBO)
		drainGL("pre-water")
		w.renderWaterReflection(rend, c.cam)
		gl.BindFramebuffer(gl.FRAMEBUFFER, targetFBO)
		gl.Viewport(int32(c.x), int32(glY), int32(vw), int32(vh))
		if vh > 0 {
			c.cam.SetAspect(float32(vw) / float32(vh))
		}
		drainGL("scene")
		if err := rend.Render(w.scene, c.cam); err != nil {
			fmt.Println("RenderWorld:", err)
		} else {
			w.markShadowPrimed()
		}
		w.drawModernPass(c.cam)
	}
	if ww > 0 && hh > 0 {
		gl.Viewport(0, 0, int32(ww), int32(hh))
	}
}
