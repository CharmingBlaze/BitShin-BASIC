package runtime

import (
	"fmt"
	"strings"

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
	on             bool
	userOn         bool
	rainOnly       bool
	exposure       float32
	tonemap        int // 0 Reinhard, 1 Khronos PBR Neutral, 2 ACES fitted, 3 clamp only
	bloom          float32
	fxaa           bool
	contrast       float32
	sat            float32
	tint           math32.Color
	ssao           bool
	ssaoRadius     float32
	ssaoIntensity  float32
	ssaoBias       float32
	fbo            uint32
	color          uint32
	depth          uint32
	w, h           int
	prog           uint32
	vao, vbo       uint32
	blitOK         bool
	bloomProg      uint32
	bloomTex       [5]uint32
	bloomFBO       [5]uint32
	bloomW, bloomH int
}

func tonemapMode(name string, n int) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "neutral", "pbr", "khronos", "1":
		return 1
	case "aces", "filmic", "2":
		return 2
	case "none", "clamp", "3":
		return 3
	case "reinhard", "0", "":
		if name == "" && n >= 1 && n <= 3 {
			return n
		}
		return 0
	default:
		return 0
	}
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
		"settonemap": n(func(a []value.Value) (value.Value, error) {
			w.post.tonemap = tonemapMode(argS(a, 0), argI(a, 0, 0))
			return value.Num(float64(w.post.tonemap)), nil
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
		"enablessao": need(func(a []value.Value) (value.Value, error) {
			on := argI(a, 0, 1) != 0
			w.post.ssao = on
			if on {
				w.post.on = true
				w.post.userOn = true
				if w.post.ssaoRadius <= 0 {
					w.post.ssaoRadius = 0.8
				}
				if w.post.ssaoIntensity <= 0 {
					w.post.ssaoIntensity = 1.8
				}
				if w.post.ssaoBias <= 0 {
					w.post.ssaoBias = 0.02
				}
			}
			return value.Num(float64(b01(w.post.ssao))), nil
		}),
		"setssao": need(func(a []value.Value) (value.Value, error) {
			on := argI(a, 0, 1) != 0
			w.post.ssao = on
			if on {
				w.post.on = true
				w.post.userOn = true
				if w.post.ssaoRadius <= 0 {
					w.post.ssaoRadius = 0.8
				}
				if w.post.ssaoIntensity <= 0 {
					w.post.ssaoIntensity = 1.8
				}
				if w.post.ssaoBias <= 0 {
					w.post.ssaoBias = 0.02
				}
			}
			return value.Num(float64(b01(w.post.ssao))), nil
		}),
		"getssao": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(w.post.ssao))), nil
		}),
		"setssaoradius": n(func(a []value.Value) (value.Value, error) {
			w.post.ssaoRadius = float32(argN(a, 0, 0.8))
			if w.post.ssaoRadius < 0.01 {
				w.post.ssaoRadius = 0.01
			}
			return value.Num(float64(w.post.ssaoRadius)), nil
		}),
		"getssaoradius": n(func(a []value.Value) (value.Value, error) {
			if w.post.ssaoRadius <= 0 {
				return value.Num(0.8), nil
			}
			return value.Num(float64(w.post.ssaoRadius)), nil
		}),
		"setssaointensity": n(func(a []value.Value) (value.Value, error) {
			w.post.ssaoIntensity = float32(argN(a, 0, 1.8))
			if w.post.ssaoIntensity < 0 {
				w.post.ssaoIntensity = 0
			}
			return value.Num(float64(w.post.ssaoIntensity)), nil
		}),
		"getssaointensity": n(func(a []value.Value) (value.Value, error) {
			if w.post.ssaoIntensity <= 0 {
				return value.Num(1.8), nil
			}
			return value.Num(float64(w.post.ssaoIntensity)), nil
		}),
		"setssaobias": n(func(a []value.Value) (value.Value, error) {
			w.post.ssaoBias = float32(argN(a, 0, 0.02))
			return value.Num(float64(w.post.ssaoBias)), nil
		}),
		"getssaobias": n(func(a []value.Value) (value.Value, error) {
			if w.post.ssaoBias <= 0 {
				return value.Num(0.02), nil
			}
			return value.Num(float64(w.post.ssaoBias)), nil
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
	if (!w.post.on && !w.guiFrame) || ww <= 0 || hh <= 0 {
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
	if w.post.on {
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		w.blitPostFX(gs, ww, hh)
		return
	}
	w.blitSceneCopy(ww, hh)
}

func (w *World) blitSceneCopy(ww, hh int) {
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, w.post.fbo)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
	gl.BlitFramebuffer(0, 0, int32(ww), int32(hh), 0, 0, int32(ww), int32(hh), gl.COLOR_BUFFER_BIT, gl.NEAREST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	drainGL("gui scene blit")
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

	gl.GenTextures(1, &depth)
	gl.BindTexture(gl.TEXTURE_2D, depth)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT24, int32(ww), int32(hh), 0, gl.DEPTH_COMPONENT, gl.UNSIGNED_INT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, color, 0)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, depth, 0)
	st := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	if st != gl.FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("FBO incomplete (%d)", st)
	}
	w.post.fbo, w.post.color, w.post.depth = fbo, color, depth
	w.post.w, w.post.h = ww, hh
	w.disposeBloomChain()
	return w.ensurePostBlit()
}

func (w *World) disposePostFBO() {
	if w.post.color != 0 {
		gl.DeleteTextures(1, &w.post.color)
		w.post.color = 0
	}
	if w.post.depth != 0 {
		gl.DeleteTextures(1, &w.post.depth)
		w.post.depth = 0
	}
	if w.post.fbo != 0 {
		gl.DeleteFramebuffers(1, &w.post.fbo)
		w.post.fbo = 0
	}
	w.disposeBloomChain()
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

func (w *World) disposeBloomChain() {
	for i := 0; i < len(w.post.bloomTex); i++ {
		if w.post.bloomTex[i] != 0 {
			gl.DeleteTextures(1, &w.post.bloomTex[i])
			w.post.bloomTex[i] = 0
		}
		if w.post.bloomFBO[i] != 0 {
			gl.DeleteFramebuffers(1, &w.post.bloomFBO[i])
			w.post.bloomFBO[i] = 0
		}
	}
	w.post.bloomW, w.post.bloomH = 0, 0
}

func (w *World) ensureBloomProg() error {
	if w.post.bloomProg != 0 {
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
		}{gl.FRAGMENT_SHADER, mbBloomDownFragment},
	)
	if err != nil {
		return err
	}
	w.post.bloomProg = p
	return nil
}

func (w *World) ensureBloomChain(ww, hh int) error {
	if ww < 2 || hh < 2 {
		return fmt.Errorf("bloom size")
	}
	if w.post.bloomTex[0] != 0 && w.post.bloomW == ww && w.post.bloomH == hh {
		return nil
	}
	w.disposeBloomChain()
	bw, bh := ww, hh
	for i := 0; i < len(w.post.bloomTex); i++ {
		bw /= 2
		bh /= 2
		if bw < 1 {
			bw = 1
		}
		if bh < 1 {
			bh = 1
		}
		var tex, fbo uint32
		gl.GenTextures(1, &tex)
		gl.BindTexture(gl.TEXTURE_2D, tex)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(bw), int32(bh), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.GenFramebuffers(1, &fbo)
		gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, tex, 0)
		st := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.BindTexture(gl.TEXTURE_2D, 0)
		if st != gl.FRAMEBUFFER_COMPLETE {
			gl.DeleteTextures(1, &tex)
			gl.DeleteFramebuffers(1, &fbo)
			w.disposeBloomChain()
			return fmt.Errorf("bloom FBO incomplete (%d)", st)
		}
		w.post.bloomTex[i] = tex
		w.post.bloomFBO[i] = fbo
	}
	w.post.bloomW, w.post.bloomH = ww, hh
	return nil
}

func (w *World) fillBloomChain(ww, hh int) bool {
	if w.post.bloom <= 0.001 || w.post.rainOnly || w.post.color == 0 || w.post.vao == 0 {
		return false
	}
	if err := w.ensureBloomProg(); err != nil {
		return false
	}
	if err := w.ensureBloomChain(ww, hh); err != nil {
		return false
	}
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.UseProgram(w.post.bloomProg)
	gl.Uniform1i(gl.GetUniformLocation(w.post.bloomProg, gl.Str("uSrc\x00")), 0)
	src := w.post.color
	sw, sh := ww, hh
	for i := 0; i < len(w.post.bloomTex); i++ {
		dw, dh := ww, hh
		for k := 0; k <= i; k++ {
			dw /= 2
			dh /= 2
			if dw < 1 {
				dw = 1
			}
			if dh < 1 {
				dh = 1
			}
		}
		gl.BindFramebuffer(gl.FRAMEBUFFER, w.post.bloomFBO[i])
		gl.Viewport(0, 0, int32(dw), int32(dh))
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, src)
		gl.Uniform2f(gl.GetUniformLocation(w.post.bloomProg, gl.Str("uTexel\x00")), 1/float32(sw), 1/float32(sh))
		thr := float32(0)
		if i == 0 {
			thr = 0.72
		}
		gl.Uniform1f(gl.GetUniformLocation(w.post.bloomProg, gl.Str("uThreshold\x00")), thr)
		gl.BindVertexArray(w.post.vao)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
		src = w.post.bloomTex[i]
		sw, sh = dw, dh
	}
	gl.BindVertexArray(0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.UseProgram(0)
	return true
}

func (w *World) blitPostFX(gs *gls.GLS, ww, hh int) {
	if w.post.prog == 0 || !w.post.blitOK {
		return
	}
	restore := saveModernGL(gs)
	defer restore()
	chain := w.fillBloomChain(ww, hh)
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

	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, w.post.depth)
	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uDepth\x00")), 1)

	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uSSAO\x00")), int32(b01(w.post.ssao)))
	rad := w.post.ssaoRadius
	if rad <= 0 {
		rad = 0.8
	}
	inten := w.post.ssaoIntensity
	if inten <= 0 {
		inten = 1.8
	}
	bias := w.post.ssaoBias
	if bias <= 0 {
		bias = 0.02
	}
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uSSAORadius\x00")), rad)
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uSSAOIntensity\x00")), inten)
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uSSAOBias\x00")), bias)

	exp := w.post.exposure
	if exp <= 0 {
		exp = 1
	}
	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uTonemap\x00")), int32(w.post.tonemap))
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uExposure\x00")), exp)
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uBloom\x00")), w.post.bloom)
	chainOn := int32(0)
	if chain {
		chainOn = 1
		for i := 0; i < len(w.post.bloomTex); i++ {
			gl.ActiveTexture(gl.TEXTURE2 + uint32(i))
			gl.BindTexture(gl.TEXTURE_2D, w.post.bloomTex[i])
			name := fmt.Sprintf("uBloom%d\x00", i)
			gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str(name)), int32(2+i))
		}
	}
	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uBloomChain\x00")), chainOn)
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
	for i := 0; i < len(w.post.bloomTex); i++ {
		gl.ActiveTexture(gl.TEXTURE2 + uint32(i))
		gl.BindTexture(gl.TEXTURE_2D, 0)
	}
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.ActiveTexture(gl.TEXTURE0)
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
	if w.post.fbo != 0 && (w.post.on || w.guiFrame) {
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
