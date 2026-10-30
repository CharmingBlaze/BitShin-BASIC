package runtime

import (
	"fmt"
	"math"
	"strings"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Fullscreen post stack on OpenGL 3.3: scene → color FBO → blit
// (tonemap + exposure + cheap bloom + optional FXAA + color grade).
// Contact shading is a half-resolution horizon pass on the scene depth.
// Not a deferred MRT / Unreal post graph.

type postFX struct {
	// MSAA: the scene renders here and resolves into fbo (color and depth),
	// so every pass after it reads the same textures as before.
	msaa                           int // wanted samples, 0 off
	msaaFBO, msaaColor, msaaDepth  uint32
	msaaW, msaaH, msaaHave         int
	on             bool
	userOn         bool
	rainOnly       bool
	exposure       float32
	tonemap        int // 0 Reinhard, 1 Khronos PBR Neutral, 2 ACES fitted, 3 clamp only
	bloom          float32
	fxaa           bool
	grain          float32
	vignette       float32
	chroma         float32
	sharpen        float32
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
	gtaoProg       uint32
	gtaoBlurProg   uint32
	gtaoBroken     bool
	aoTex          uint32
	aoBlurTex      uint32
	aoFBO          uint32
	aoBlurFBO      uint32
	aoW, aoH       int
	auto           bool
	autoKey        float32
	autoSpeed      float32
	adapted        float32
	adaptedOK      bool
	lumProg        uint32
	lumTex         uint32
	lumFBO         uint32
}

// enableGrade turns on the Khronos neutral curve and a small bloom.
// Exposure and an explicit tonemap chosen earlier are left as they are,
// except a still-default Reinhard curve becomes neutral.
func (w *World) enableGrade() {
	w.post.on = true
	w.post.userOn = true
	if w.post.tonemap == 0 {
		w.post.tonemap = 1
	}
	if w.post.exposure <= 0 {
		w.post.exposure = 1.05
	}
	if w.post.bloom == 0 {
		w.post.bloom = 0.22
	}
	w.post.auto = true
	if w.post.autoKey <= 0 {
		w.post.autoKey = 0.5
	}
	if w.post.autoSpeed <= 0 {
		w.post.autoSpeed = 1.6
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
	w.post.fxaa = true
}

func clamp01n(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
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
				if w.post.bloom == 0 {
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
		"setautoexposure": n(func(a []value.Value) (value.Value, error) {
			w.post.auto = argI(a, 0, 1) != 0
			if w.post.auto {
				w.post.on = true
				w.post.userOn = true
				if w.post.autoKey <= 0 {
					w.post.autoKey = 0.5
				}
				if w.post.autoSpeed <= 0 {
					w.post.autoSpeed = 1.6
				}
			}
			return value.Num(float64(b01(w.post.auto))), nil
		}),
		"getautoexposure": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(w.post.auto))), nil
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
		"setfilmgrain": n(func(a []value.Value) (value.Value, error) {
			w.post.grain = clamp01n(float32(argN(a, 0, 0)))
			w.post.on = true
			return value.Num(float64(w.post.grain)), nil
		}),
		"setvignette": n(func(a []value.Value) (value.Value, error) {
			w.post.vignette = clamp01n(float32(argN(a, 0, 0)))
			w.post.on = true
			return value.Num(float64(w.post.vignette)), nil
		}),
		"setchromaticaberration": n(func(a []value.Value) (value.Value, error) {
			w.post.chroma = clamp01n(float32(argN(a, 0, 0)))
			w.post.on = true
			return value.Num(float64(w.post.chroma)), nil
		}),
		"setsharpen": n(func(a []value.Value) (value.Value, error) {
			w.post.sharpen = float32(argN(a, 0, 0))
			if w.post.sharpen < 0 {
				w.post.sharpen = 0
			}
			if w.post.sharpen > 2 {
				w.post.sharpen = 2
			}
			w.post.on = true
			return value.Num(float64(w.post.sharpen)), nil
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
		// SetMSAA(samples): multisample the scene (2, 4, or 8; 0 off). It
		// needs post FX on; ultra turns on 4. Smooths thin edges FXAA cannot.
		"setmsaa": n(func(a []value.Value) (value.Value, error) {
			n := argI(a, 0, 4)
			switch {
			case n <= 1:
				n = 0
			case n <= 2:
				n = 2
			case n <= 4:
				n = 4
			default:
				n = 8
			}
			w.post.msaa = n
			return value.Num(float64(n)), nil
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
		glErrCount.Add(1)
		if i == 0 && !drainLogged[where] {
			drainLogged[where] = true
			fmt.Println(where+":", glErrName(err))
		}
	}
}

func glErrName(err uint32) string {
	switch err {
	case gl.INVALID_ENUM:
		return "GL_INVALID_ENUM 1280"
	case gl.INVALID_VALUE:
		return "GL_INVALID_VALUE 1281"
	case gl.INVALID_OPERATION:
		return "GL_INVALID_OPERATION 1282"
	case gl.OUT_OF_MEMORY:
		return "GL_OUT_OF_MEMORY 1285"
	case gl.INVALID_FRAMEBUFFER_OPERATION:
		return "GL_INVALID_FRAMEBUFFER_OPERATION 1286"
	}
	return fmt.Sprintf("GL error %d", err)
}

// bindColorTarget selects the framebuffer the scene colors are written to.
// A depth-only shadow pass sets GL_DRAW_BUFFER to NONE; that state has to
// be put back or the mesh pass clears and then draws nothing.
func bindColorTarget(fbo uint32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	if fbo == 0 {
		gl.DrawBuffer(gl.BACK)
		gl.ReadBuffer(gl.BACK)
	} else {
		gl.DrawBuffer(gl.COLOR_ATTACHMENT0)
		gl.ReadBuffer(gl.COLOR_ATTACHMENT0)
	}
	gl.ColorMask(true, true, true, true)
	gl.DepthMask(true)
	gl.DepthFunc(gl.LEQUAL)
	gl.Disable(gl.SCISSOR_TEST)
	gl.Enable(gl.DEPTH_TEST)
}

func (w *World) beginPostTarget(ww, hh int) bool {
	// ImGui is drawn on the window after the scene. Routing those frames
	// through the post FBO left the default framebuffer cleared and empty.
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
	dw, dh := w.drawableSize(ww, hh)
	gl.BlitFramebuffer(0, 0, int32(ww), int32(hh), 0, 0, int32(dw), int32(dh), gl.COLOR_BUFFER_BIT, gl.NEAREST)
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
	// Half float: the lit shaders roll highlights off toward 2.0, and an
	// 8-bit target clipped them at white before exposure and bloom saw them.
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, int32(ww), int32(hh), 0, gl.RGBA, gl.HALF_FLOAT, nil)
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
	w.disposeGTAO()
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
	w.disposeMSAA()
	w.disposeBloomChain()
	w.disposeGTAO()
}

func (w *World) disposeMSAA() {
	p := &w.post
	if p.msaaColor != 0 {
		gl.DeleteRenderbuffers(1, &p.msaaColor)
		p.msaaColor = 0
	}
	if p.msaaDepth != 0 {
		gl.DeleteRenderbuffers(1, &p.msaaDepth)
		p.msaaDepth = 0
	}
	if p.msaaFBO != 0 {
		gl.DeleteFramebuffers(1, &p.msaaFBO)
		p.msaaFBO = 0
	}
	p.msaaHave = 0
}

// sceneTarget is where the scene is drawn: the MSAA buffer when it is on
// and ready, else the post target. 0 is the window.
func (w *World) sceneTarget(ww, hh int) uint32 {
	p := &w.post
	if !p.on || p.fbo == 0 {
		return 0
	}
	if p.msaa < 2 {
		if p.msaaFBO != 0 {
			w.disposeMSAA()
		}
		return p.fbo
	}
	if p.msaaFBO != 0 && p.msaaW == ww && p.msaaH == hh && p.msaaHave == p.msaa {
		return p.msaaFBO
	}
	w.disposeMSAA()
	var fbo, col, dep uint32
	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.GenRenderbuffers(1, &col)
	gl.BindRenderbuffer(gl.RENDERBUFFER, col)
	gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, int32(p.msaa), gl.RGBA16F, int32(ww), int32(hh))
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.RENDERBUFFER, col)
	gl.GenRenderbuffers(1, &dep)
	gl.BindRenderbuffer(gl.RENDERBUFFER, dep)
	gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, int32(p.msaa), gl.DEPTH_COMPONENT24, int32(ww), int32(hh))
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, dep)
	ok := gl.CheckFramebufferStatus(gl.FRAMEBUFFER) == gl.FRAMEBUFFER_COMPLETE
	gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	p.msaaFBO, p.msaaColor, p.msaaDepth = fbo, col, dep
	p.msaaW, p.msaaH, p.msaaHave = ww, hh, p.msaa
	drainGL("MSAA")
	if !ok {
		// A driver that will not multisample half float: run without.
		fmt.Println("PostFX: MSAA target incomplete; drawing without it")
		w.disposeMSAA()
		p.msaa = 0
		return p.fbo
	}
	return fbo
}

// resolveMSAA copies the multisampled scene (color and depth) into the post
// target the rest of the chain reads.
func (w *World) resolveMSAA(ww, hh int) {
	p := &w.post
	if p.msaaFBO == 0 || p.fbo == 0 {
		return
	}
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, p.msaaFBO)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, p.fbo)
	gl.BlitFramebuffer(0, 0, int32(ww), int32(hh), 0, 0, int32(ww), int32(hh), gl.COLOR_BUFFER_BIT, gl.NEAREST)
	gl.BlitFramebuffer(0, 0, int32(ww), int32(hh), 0, 0, int32(ww), int32(hh), gl.DEPTH_BUFFER_BIT, gl.NEAREST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, p.fbo)
	drainGL("MSAA resolve")
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
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, int32(bw), int32(bh), 0, gl.RGBA, gl.HALF_FLOAT, nil)
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

func (w *World) disposeGTAO() {
	if w.post.aoTex != 0 {
		gl.DeleteTextures(1, &w.post.aoTex)
		w.post.aoTex = 0
	}
	if w.post.aoBlurTex != 0 {
		gl.DeleteTextures(1, &w.post.aoBlurTex)
		w.post.aoBlurTex = 0
	}
	if w.post.aoFBO != 0 {
		gl.DeleteFramebuffers(1, &w.post.aoFBO)
		w.post.aoFBO = 0
	}
	if w.post.aoBlurFBO != 0 {
		gl.DeleteFramebuffers(1, &w.post.aoBlurFBO)
		w.post.aoBlurFBO = 0
	}
	w.post.aoW, w.post.aoH = 0, 0
}

func (w *World) ensureGTAOProg() error {
	if w.post.gtaoProg != 0 && w.post.gtaoBlurProg != 0 {
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
		}{gl.FRAGMENT_SHADER, mbGTAOFragment},
	)
	if err != nil {
		return err
	}
	b, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbpostVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbGTAOBlurFragment},
	)
	if err != nil {
		gl.DeleteProgram(p)
		return err
	}
	w.post.gtaoProg = p
	w.post.gtaoBlurProg = b
	return nil
}

func (w *World) ensureGTAOTargets(ww, hh int) error {
	aw, ah := ww/2, hh/2
	if aw < 1 {
		aw = 1
	}
	if ah < 1 {
		ah = 1
	}
	if w.post.aoTex != 0 && w.post.aoW == aw && w.post.aoH == ah {
		return nil
	}
	w.disposeGTAO()
	makeR := func() (uint32, uint32, error) {
		var tex, fbo uint32
		gl.GenTextures(1, &tex)
		gl.BindTexture(gl.TEXTURE_2D, tex)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R8, int32(aw), int32(ah), 0, gl.RED, gl.UNSIGNED_BYTE, nil)
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
			return 0, 0, fmt.Errorf("gtao fbo")
		}
		return tex, fbo, nil
	}
	tex, fbo, err := makeR()
	if err != nil {
		return err
	}
	blur, blurFBO, err := makeR()
	if err != nil {
		return err
	}
	w.post.aoTex, w.post.aoFBO = tex, fbo
	w.post.aoBlurTex, w.post.aoBlurFBO = blur, blurFBO
	w.post.aoW, w.post.aoH = aw, ah
	return nil
}

func (w *World) gtaoCamera() (near, far, tanHalf, aspect float32) {
	near, far = 0.3, 1000
	fov := float32(60)
	aspect = 1
	if w.post.w > 0 && w.post.h > 0 {
		aspect = float32(w.post.w) / float32(w.post.h)
	}
	if w.cam != nil {
		if n := w.cam.Near(); n > 0.001 {
			near = n
		}
		if f := w.cam.Far(); f > near+1 {
			far = f
		}
		if fv := w.cam.Fov(); fv > 1 && fv < 179 {
			fov = fv
		}
		if a := w.cam.Aspect(); a > 0.01 {
			aspect = a
		}
	}
	tanHalf = float32(math.Tan(float64(fov) * math.Pi / 360))
	if tanHalf < 0.05 {
		tanHalf = 0.05
	}
	return near, far, tanHalf, aspect
}

func (w *World) fillGTAO(ww, hh int) bool {
	if w.post.gtaoBroken || w.post.depth == 0 || ww < 2 || hh < 2 {
		return false
	}
	if err := w.ensureGTAOProg(); err != nil {
		w.post.gtaoBroken = true
		fmt.Println("GTAO:", err)
		return false
	}
	if err := w.ensureGTAOTargets(ww, hh); err != nil {
		return false
	}
	near, far, tanHalf, aspect := w.gtaoCamera()
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
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.UseProgram(w.post.gtaoProg)
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.post.aoFBO)
	gl.Viewport(0, 0, int32(w.post.aoW), int32(w.post.aoH))
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, w.post.depth)
	gl.Uniform1i(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uDepth\x00")), 0)
	gl.Uniform2f(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uTexel\x00")), 1/float32(ww), 1/float32(hh))
	gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uRadius\x00")), rad)
	gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uBias\x00")), bias)
	gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uIntensity\x00")), inten)
	gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uNear\x00")), near)
	gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uFar\x00")), far)
	gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uTanHalf\x00")), tanHalf)
	gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoProg, gl.Str("uAspect\x00")), aspect)
	gl.BindVertexArray(w.post.vao)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	blur := func(src, dstFBO uint32, dirX, dirY float32) {
		gl.UseProgram(w.post.gtaoBlurProg)
		gl.BindFramebuffer(gl.FRAMEBUFFER, dstFBO)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, src)
		gl.Uniform1i(gl.GetUniformLocation(w.post.gtaoBlurProg, gl.Str("uAO\x00")), 0)
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, w.post.depth)
		gl.Uniform1i(gl.GetUniformLocation(w.post.gtaoBlurProg, gl.Str("uDepth\x00")), 1)
		gl.Uniform2f(gl.GetUniformLocation(w.post.gtaoBlurProg, gl.Str("uTexel\x00")), 1/float32(w.post.aoW), 1/float32(w.post.aoH))
		gl.Uniform2f(gl.GetUniformLocation(w.post.gtaoBlurProg, gl.Str("uDirection\x00")), dirX, dirY)
		gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoBlurProg, gl.Str("uNear\x00")), near)
		gl.Uniform1f(gl.GetUniformLocation(w.post.gtaoBlurProg, gl.Str("uFar\x00")), far)
		gl.BindVertexArray(w.post.vao)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	}
	blur(w.post.aoTex, w.post.aoBlurFBO, 1, 0)
	blur(w.post.aoBlurTex, w.post.aoFBO, 0, 1)

	gl.BindVertexArray(0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.UseProgram(0)
	return true
}

func (w *World) measureLuma(ww, hh int) (float32, bool) {
	if w.post.color == 0 || ww < 2 || hh < 2 {
		return 0, false
	}
	if w.post.lumProg == 0 {
		p, err := compileLink(
			struct {
				kind uint32
				src  string
			}{gl.VERTEX_SHADER, mbpostVertex},
			struct {
				kind uint32
				src  string
			}{gl.FRAGMENT_SHADER, mbLumFragment},
		)
		if err != nil {
			fmt.Println("exposure:", err)
			return 0, false
		}
		w.post.lumProg = p
	}
	if w.post.lumTex == 0 {
		var tex, fbo uint32
		gl.GenTextures(1, &tex)
		gl.BindTexture(gl.TEXTURE_2D, tex)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.GenFramebuffers(1, &fbo)
		gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, tex, 0)
		if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
			fmt.Println("exposure: fbo")
			gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
			return 0, false
		}
		w.post.lumTex, w.post.lumFBO = tex, fbo
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.post.lumFBO)
	gl.Viewport(0, 0, 1, 1)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.UseProgram(w.post.lumProg)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, w.post.color)
	gl.Uniform1i(gl.GetUniformLocation(w.post.lumProg, gl.Str("uTex\x00")), 0)
	gl.BindVertexArray(w.post.vao)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	var pix [4]byte
	gl.ReadPixels(0, 0, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pix[0]))
	gl.BindVertexArray(0)
	gl.UseProgram(0)
	drainGL("exposure")
	return float32(pix[0]) / 255, true
}

func adaptExposure(current float32, have bool, lum, key, speed float32, dt float64) (float32, bool) {
	if lum < 0.004 {
		lum = 0.004
	}
	if key <= 0 {
		key = 0.5
	}
	target := key / lum
	if target < 0.35 {
		target = 0.35
	}
	if target > 3.5 {
		target = 3.5
	}
	if !have || speed <= 0 {
		return target, true
	}
	if dt <= 0 {
		dt = 1.0 / 60
	}
	if dt > 0.25 {
		dt = 0.25
	}
	k := 1 - math.Exp(-float64(speed)*dt)
	return current + (target-current)*float32(k), true
}

func (w *World) blitPostFX(gs *gls.GLS, ww, hh int) {
	if w.post.prog == 0 || !w.post.blitOK {
		return
	}
	restore := saveModernGL(gs)
	defer restore()
	chain := w.fillBloomChain(ww, hh)
	gtao := w.post.ssao && w.fillGTAO(ww, hh)
	var lum float32
	lumOK := false
	if w.post.auto {
		lum, lumOK = w.measureLuma(ww, hh)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	if dw, dh := w.drawableSize(ww, hh); dw > 0 && dh > 0 {
		gl.Viewport(0, 0, int32(dw), int32(dh))
	}
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.UseProgram(w.post.prog)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, w.post.color)
	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uTex\x00")), 0)

	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uSSAO\x00")), int32(b01(gtao)))
	gl.ActiveTexture(gl.TEXTURE1)
	if gtao {
		gl.BindTexture(gl.TEXTURE_2D, w.post.aoTex)
	} else {
		gl.BindTexture(gl.TEXTURE_2D, w.post.color)
	}
	gl.Uniform1i(gl.GetUniformLocation(w.post.prog, gl.Str("uAOTex\x00")), 1)

	exp := w.post.exposure
	if exp <= 0 {
		exp = 1
	}
	if w.post.auto && lumOK {
		key := w.post.autoKey
		if key <= 0 {
			key = 0.5
		}
		speed := w.post.autoSpeed
		if speed <= 0 {
			speed = 1.6
		}
		w.post.adapted, w.post.adaptedOK = adaptExposure(w.post.adapted, w.post.adaptedOK, lum, key, speed, w.delta)
		if w.post.adapted > 0 {
			exp *= w.post.adapted
		}
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
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uGrain\x00")), w.post.grain)
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uVignette\x00")), w.post.vignette)
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uCA\x00")), w.post.chroma)
	gl.Uniform1f(gl.GetUniformLocation(w.post.prog, gl.Str("uSharpen\x00")), w.post.sharpen)
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

// rearmSceneProgram forces the next material draw to call glUseProgram.
// ImGui and the shadow depth pass bind other programs, but G3N's shader
// manager still thinks the lit program is current and skips the bind.
// The depth program writes no color, so the window stays the clear color.
func rearmSceneProgram(rend *renderer.Renderer) {
	if rend == nil {
		return
	}
	specs := renderer.ShaderSpecs{
		Name:    "standard",
		Defines: gls.ShaderDefines{"MB_REBIND": "1"},
	}
	if _, err := rend.SetProgram(&specs); err != nil {
		fmt.Println("RenderWorld:", err)
	}
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
	w.prepareWaterFFT()
	if !split {
		targetFBO = w.sceneTarget(ww, hh)
		cam := w.cam
		if cam == nil && len(list) > 0 {
			cam = list[0].cam
		}
		if cam != nil {
			w.renderShadows(rend, cam)
			bindColorTarget(targetFBO)
			drainGL("pre-water")
			w.renderWaterReflection(rend, cam)
			bindColorTarget(targetFBO)
			if ww > 0 && hh > 0 {
				gl.Viewport(0, 0, int32(ww), int32(hh))
			}
			// The shadow pass can leave the window depth buffer full. Clear
			// it again or every mesh fragment fails the depth test.
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
			drainGL("scene")
			rearmSceneProgram(rend)
			if err := rend.Render(w.scene, cam); err != nil {
				fmt.Println("RenderWorld:", err)
			} else {
				w.markShadowPrimed()
			}
			w.drawModernPass(cam)
			if targetFBO != 0 && targetFBO == w.post.msaaFBO {
				w.resolveMSAA(ww, hh)
			}
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
		bindColorTarget(targetFBO)
		drainGL("pre-water")
		w.renderWaterReflection(rend, c.cam)
		bindColorTarget(targetFBO)
		gl.Viewport(int32(c.x), int32(glY), int32(vw), int32(vh))
		gl.Enable(gl.SCISSOR_TEST)
		gl.Scissor(int32(c.x), int32(glY), int32(vw), int32(vh))
		gl.Clear(gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		gl.Disable(gl.SCISSOR_TEST)
		if vh > 0 {
			c.cam.SetAspect(float32(vw) / float32(vh))
		}
		drainGL("scene")
		rearmSceneProgram(rend)
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
