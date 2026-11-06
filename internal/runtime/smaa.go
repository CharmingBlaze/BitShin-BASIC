package runtime

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// SMAA 1x: Jimenez et al., "SMAA: Enhanced Subpixel Morphological Antialiasing"
// (Eurographics 2012). Three passes over the finished display image: find edges by
// luma, work out from each edge's shape (its length and how its ends turn) how much
// of each pixel the true edge covers, then blend each pixel with the neighbor across
// it by that much. FXAA guesses the same thing from a small window and blurs to be
// safe; SMAA reads the whole edge, so long tree trunks, roof lines and horizons come
// out clean and straight while texture inside surfaces stays sharp.
//
// The shader is the reference SMAA.hlsl, compiled as it is with SMAA_GLSL_3, and the
// two lookup tables are the reference AreaTex and SearchTex, all MIT licensed (see
// smaa/LICENSE.txt). GL's texture rows run bottom to top where D3D's run top to
// bottom, so the passes see the picture upside down; every step works in the same
// mirrored space, which is the same algorithm on a mirrored image.

//go:embed smaa/SMAA.hlsl
var smaaSource string

//go:embed smaa/area.bin
var smaaAreaBytes []byte

//go:embed smaa/search.bin
var smaaSearchBytes []byte

const (
	smaaAreaW, smaaAreaH     = 160, 560
	smaaSearchW, smaaSearchH = 64, 16
)

// smaaHeader goes before the reference source in each pass.
const smaaHeader = `#version 330 core
#define SMAA_GLSL_3 1
#define SMAA_PRESET_HIGH 1
#define SMAA_INCLUDE_VS 1
#define SMAA_INCLUDE_PS 1
uniform vec4 uMetrics;
#define SMAA_RT_METRICS uMetrics
`

const smaaEdgeMain = `
in vec2 vUV;
out vec4 frag;
uniform sampler2D uColor;
void main() {
	vec4 offset[3];
	SMAAEdgeDetectionVS(vUV, offset);
	frag = vec4(SMAALumaEdgeDetectionPS(vUV, offset, uColor), 0.0, 1.0);
}
`

const smaaWeightMain = `
in vec2 vUV;
out vec4 frag;
uniform sampler2D uEdges;
uniform sampler2D uArea;
uniform sampler2D uSearch;
void main() {
	vec2 pixcoord;
	vec4 offset[3];
	SMAABlendingWeightCalculationVS(vUV, pixcoord, offset);
	frag = SMAABlendingWeightCalculationPS(vUV, pixcoord, offset, uEdges, uArea, uSearch, vec4(0.0));
}
`

// The last pass also adds film grain, after the edges were found, as the FXAA pass does.
const smaaBlendMain = `
in vec2 vUV;
out vec4 frag;
uniform sampler2D uColor;
uniform sampler2D uWeights;
uniform float uGrain;
uniform float uTime;
void main() {
	vec4 offset;
	SMAANeighborhoodBlendingVS(vUV, offset);
	vec3 c = SMAANeighborhoodBlendingPS(vUV, offset, uColor, uWeights).rgb;
	if (uGrain > 0.001) {
		vec3 p3 = fract(vec3((vUV * vec2(1920.0, 1080.0) + uTime * 13.0).xyx) * 0.1031);
		p3 += dot(p3, p3.yzx + 33.33);
		c += (fract((p3.x + p3.y) * p3.z) - 0.5) * uGrain;
	}
	frag = vec4(c, 1.0);
}
`

type smaaState struct {
	edgeProg, weightProg, blendProg uint32
	area, search                    uint32
	edges, weights                  fsrTarget
	failed                          bool
}

// smaaSourceASCII is the reference source with its two non-ASCII characters, in a
// comment diagram, made plain: some drivers refuse any byte above 127, comments or not.
func smaaSourceASCII() string {
	return strings.Map(func(r rune) rune {
		if r > 127 {
			return ' '
		}
		return r
	}, smaaSource)
}

func (w *World) ensureSMAA(ww, hh int) bool {
	s := &w.smaa
	if s.failed {
		return false
	}
	if s.edgeProg == 0 {
		src := smaaSourceASCII()
		var err error
		for _, p := range []struct {
			prog *uint32
			main string
		}{{&s.edgeProg, smaaEdgeMain}, {&s.weightProg, smaaWeightMain}, {&s.blendProg, smaaBlendMain}} {
			if *p.prog, err = fsrProgram(smaaHeader + src + p.main); err != nil {
				warnScene("SMAA", fmt.Errorf("%v; using FXAA", err))
				w.disposeSMAA()
				s.failed = true
				return false
			}
		}
		s.area = smaaTable(smaaAreaW, smaaAreaH, gl.RG8, gl.RG, smaaAreaBytes)
		s.search = smaaTable(smaaSearchW, smaaSearchH, gl.R8, gl.RED, smaaSearchBytes)
	}
	return s.edges.ensure(ww, hh) && s.weights.ensure(ww, hh)
}

// smaaTable uploads a lookup table row for row, first row at t = 0, as SMAA reads it.
// The search table must be read without filtering; the area table is read between
// texels on purpose.
func smaaTable(tw, th int, internal int32, format uint32, data []byte) uint32 {
	var t uint32
	gl.GenTextures(1, &t)
	gl.BindTexture(gl.TEXTURE_2D, t)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage2D(gl.TEXTURE_2D, 0, internal, int32(tw), int32(th), 0, format, gl.UNSIGNED_BYTE, gl.Ptr(data))
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	filter := int32(gl.LINEAR)
	if format == gl.RED {
		filter = gl.NEAREST
	}
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return t
}

func (w *World) disposeSMAA() {
	s := &w.smaa
	for _, p := range []*uint32{&s.edgeProg, &s.weightProg, &s.blendProg} {
		if *p != 0 {
			gl.DeleteProgram(*p)
			*p = 0
		}
	}
	for _, t := range []*uint32{&s.area, &s.search} {
		if *t != 0 {
			gl.DeleteTextures(1, t)
			*t = 0
		}
	}
	s.edges.dispose()
	s.weights.dispose()
	s.failed = false
}

// drawEdgeAA smooths the display image in src, of pw x ph, onto the bound target:
// SMAA when asked for and available, FXAA otherwise.
func (w *World) drawEdgeAA(src uint32, pw, ph int) {
	if w.post.smaa && w.ensureSMAA(pw, ph) {
		w.drawSMAA(src, pw, ph)
		return
	}
	w.drawFXAA(src, pw, ph)
}

func (w *World) drawSMAA(src uint32, pw, ph int) {
	s := &w.smaa
	var target int32
	var vp [4]int32
	gl.GetIntegerv(gl.FRAMEBUFFER_BINDING, &target)
	gl.GetIntegerv(gl.VIEWPORT, &vp[0])
	use := func(prog uint32) func(string) int32 {
		gl.UseProgram(prog)
		gl.Uniform4f(gl.GetUniformLocation(prog, gl.Str("uMetrics\x00")), 1/float32(pw), 1/float32(ph), float32(pw), float32(ph))
		return func(name string) int32 { return gl.GetUniformLocation(prog, gl.Str(name+"\x00")) }
	}
	bind := func(unit uint32, tex uint32) {
		gl.ActiveTexture(gl.TEXTURE0 + unit)
		gl.BindTexture(gl.TEXTURE_2D, tex)
	}
	gl.BindVertexArray(w.post.vao)
	gl.Viewport(0, 0, int32(pw), int32(ph))

	// Edges. The pass discards pixels with none, so the target starts clear.
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.edges.fbo)
	gl.ClearColor(0, 0, 0, 0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	loc := use(s.edgeProg)
	bind(0, src)
	gl.Uniform1i(loc("uColor"), 0)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	// Blend weights, from each edge's length and the shape of its ends.
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.weights.fbo)
	loc = use(s.weightProg)
	bind(0, s.edges.tex)
	bind(1, s.area)
	bind(2, s.search)
	gl.Uniform1i(loc("uEdges"), 0)
	gl.Uniform1i(loc("uArea"), 1)
	gl.Uniform1i(loc("uSearch"), 2)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	// Blend into the target the caller bound.
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(target))
	gl.Viewport(vp[0], vp[1], vp[2], vp[3])
	loc = use(s.blendProg)
	bind(0, src)
	bind(1, s.weights.tex)
	gl.Uniform1i(loc("uColor"), 0)
	gl.Uniform1i(loc("uWeights"), 1)
	gl.Uniform1f(loc("uGrain"), w.post.grain)
	gl.Uniform1f(loc("uTime"), float32(w.wx.time))
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	bind(2, 0)
	bind(1, 0)
	bind(0, 0)
	gl.BindVertexArray(0)
}
