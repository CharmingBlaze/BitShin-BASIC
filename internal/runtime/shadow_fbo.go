package runtime

import (
	"fmt"
	"math"

	"github.com/g3n/engine/gls"
	"github.com/go-gl/gl/v3.3-core/gl"
)

const (
	shadowLayerAll     = 0
	shadowLayerStatic  = 1
	shadowLayerDynamic = 2
)

type shadowTarget struct {
	fbo, depth, color, ping uint32
	w, h                    int32
	moment                  int
	colorFmt                int32
	fallback                bool
}

func (t *shadowTarget) lightTex() uint32 {
	if t == nil {
		return 0
	}
	if t.color != 0 {
		return t.color
	}
	return t.depth
}

func (t *shadowTarget) destroy() {
	if t == nil {
		return
	}
	if t.color != 0 {
		gl.DeleteTextures(1, &t.color)
		t.color = 0
	}
	if t.ping != 0 {
		gl.DeleteTextures(1, &t.ping)
		t.ping = 0
	}
	if t.depth != 0 {
		gl.DeleteTextures(1, &t.depth)
		t.depth = 0
	}
	if t.fbo != 0 {
		gl.DeleteFramebuffers(1, &t.fbo)
		t.fbo = 0
	}
	t.w, t.h = 0, 0
	t.moment = 0
	t.colorFmt = 0
}

func allocColorTex(w, h int32, internal int32, format, typ uint32) uint32 {
	drainGL("shadow color alloc")
	var tex uint32
	gl.GenTextures(1, &tex)
	if tex == 0 {
		return 0
	}
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, internal, w, h, 0, format, typ, nil)
	if gl.GetError() != gl.NO_ERROR {
		gl.DeleteTextures(1, &tex)
		return 0
	}
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	border := []float32{1, 1, 1, 1}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &border[0])
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return tex
}

func (w *World) ensureShadowTarget(t *shadowTarget, cols, rows, moment int) error {
	if t == nil {
		return fmt.Errorf("shadow target nil")
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	wdt := int32(w.shadow.size * cols)
	hgt := int32(w.shadow.size * rows)
	if t.fbo != 0 && t.w == wdt && t.h == hgt && t.moment == moment {
		return nil
	}
	t.destroy()
	var depth uint32
	gl.GenTextures(1, &depth)
	gl.BindTexture(gl.TEXTURE_2D, depth)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT24, wdt, hgt, 0, gl.DEPTH_COMPONENT, gl.UNSIGNED_INT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	border := []float32{1, 1, 1, 1}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &border[0])

	var fbo uint32
	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, depth, 0)

	t.depth = depth
	t.fbo = fbo
	t.w, t.h = wdt, hgt
	t.moment = moment
	t.fallback = false

	if moment == shadowFilterEVSM || moment == shadowFilterMSM {
		color, colorFmt, fallback := w.allocMomentColor(wdt, hgt, moment)
		if color == 0 {
			gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
			t.destroy()
			return fmt.Errorf("shadow moment texture failed")
		}
		t.color = color
		t.colorFmt = colorFmt
		t.fallback = fallback
		ping := allocColorTex(wdt, hgt, colorFmt, gl.RGBA, gl.FLOAT)
		if ping == 0 && colorFmt == gl.RG32F {
			ping = allocColorTex(wdt, hgt, colorFmt, gl.RG, gl.FLOAT)
		}
		if ping == 0 {
			ping = allocColorTex(wdt, hgt, gl.RGBA16F, gl.RGBA, gl.FLOAT)
		}
		t.ping = ping
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, color, 0)
		gl.DrawBuffer(gl.COLOR_ATTACHMENT0)
		gl.ReadBuffer(gl.COLOR_ATTACHMENT0)
	} else {
		gl.DrawBuffer(gl.NONE)
		gl.ReadBuffer(gl.NONE)
	}
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		t.destroy()
		return fmt.Errorf("shadow FBO incomplete")
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	if t.fallback {
		w.shadow.momentFallback = true
	}
	return nil
}

func (w *World) allocMomentColor(wdt, hgt int32, moment int) (tex uint32, internal int32, fallback bool) {
	if moment == shadowFilterEVSM {
		tex = allocColorTex(wdt, hgt, gl.RG32F, gl.RG, gl.FLOAT)
		if tex != 0 {
			w.shadow.evsmC = 40
			return tex, gl.RG32F, false
		}
		tex = allocColorTex(wdt, hgt, gl.RGBA16F, gl.RGBA, gl.FLOAT)
		if tex != 0 {
			w.shadow.evsmC = 5
			return tex, gl.RGBA16F, true
		}
		return 0, 0, true
	}
	tex = allocColorTex(wdt, hgt, gl.RGBA16F, gl.RGBA, gl.FLOAT)
	if tex != 0 {
		return tex, gl.RGBA16F, false
	}
	tex = allocColorTex(wdt, hgt, gl.RGBA32F, gl.RGBA, gl.FLOAT)
	if tex != 0 {
		return tex, gl.RGBA32F, true
	}
	return 0, 0, true
}

func (w *World) bindShadowTarget(gs *gls.GLS, t *shadowTarget) {
	if t == nil || t.fbo == 0 {
		return
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	if gs != nil {
		gs.Viewport(0, 0, t.w, t.h)
	} else {
		gl.Viewport(0, 0, t.w, t.h)
	}
	if t.color != 0 {
		gl.DrawBuffer(gl.COLOR_ATTACHMENT0)
		c := w.shadow.evsmC
		if c <= 0 {
			c = 40
		}
		if t.moment == shadowFilterEVSM {
			e := float32(math.Exp(float64(c)))
			gl.ClearColor(e, e*e, 0, 1)
		} else {
			gl.ClearColor(1, 1, 1, 1)
		}
		if gs != nil {
			gs.Clear(gls.COLOR_BUFFER_BIT | gls.DEPTH_BUFFER_BIT)
		} else {
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		}
		return
	}
	gl.DrawBuffer(gl.NONE)
	if gs != nil {
		gs.Clear(gls.DEPTH_BUFFER_BIT)
	} else {
		gl.Clear(gl.DEPTH_BUFFER_BIT)
	}
}

func (w *World) ensureShadowBlit(gs *gls.GLS) error {
	if w.shadow.blurG3N != nil && w.shadow.blurG3N.Handle() != 0 && w.shadow.blitVAO != 0 {
		return nil
	}
	if gs == nil {
		return fmt.Errorf("no GLS")
	}
	if w.shadow.blurG3N == nil || w.shadow.blurG3N.Handle() == 0 {
		prog := gs.NewProgram()
		prog.AddShader(gls.VERTEX_SHADER, blurVertexSrc)
		prog.AddShader(gls.FRAGMENT_SHADER, blurFragmentSrc)
		if err := prog.Build(); err != nil {
			return fmt.Errorf("shadow blur shader:\n%s", err)
		}
		w.shadow.blurG3N = prog
	}
	if w.shadow.blitVAO == 0 {
		var vao, vbo uint32
		gl.GenVertexArrays(1, &vao)
		gl.GenBuffers(1, &vbo)
		gl.BindVertexArray(vao)
		gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
		verts := []float32{-1, -1, 3, -1, -1, 3}
		gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STATIC_DRAW)
		gl.EnableVertexAttribArray(0)
		gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, nil)
		gl.BindVertexArray(0)
		w.shadow.blitVAO = vao
		w.shadow.blitVBO = vbo
	}
	return nil
}

func (w *World) blurMoments(gs *gls.GLS, t *shadowTarget) {
	if gs == nil || t == nil || t.color == 0 || t.ping == 0 || t.fbo == 0 {
		return
	}
	if err := w.ensureShadowBlit(gs); err != nil {
		fmt.Println("EnableShadows:", err)
		return
	}
	prog := w.shadow.blurG3N
	if prog == nil || prog.Handle() == 0 {
		return
	}
	gs.UseProgram(prog)
	gs.Disable(gls.DEPTH_TEST)
	gs.Disable(gls.CULL_FACE)
	gs.Disable(gls.BLEND)
	gs.Viewport(0, 0, t.w, t.h)
	srcLoc := prog.GetUniformLocation("Src")
	dirLoc := prog.GetUniformLocation("Dir")
	tx := 1 / float32(t.w)
	ty := 1 / float32(t.h)

	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t.ping, 0)
	gl.DrawBuffer(gl.COLOR_ATTACHMENT0)
	gs.ActiveTexture(gls.TEXTURE0)
	gs.BindTexture(gls.TEXTURE_2D, t.color)
	if srcLoc >= 0 {
		gs.Uniform1i(srcLoc, 0)
	}
	if dirLoc >= 0 {
		gs.Uniform2f(dirLoc, tx, 0)
	}
	gl.BindVertexArray(w.shadow.blitVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t.color, 0)
	gs.BindTexture(gls.TEXTURE_2D, t.ping)
	if dirLoc >= 0 {
		gs.Uniform2f(dirLoc, 0, ty)
	}
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
	gl.BindVertexArray(0)
	gs.Enable(gls.DEPTH_TEST)
	gs.Enable(gls.CULL_FACE)
}
