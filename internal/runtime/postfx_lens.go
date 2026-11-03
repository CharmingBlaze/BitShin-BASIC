package runtime

import (
	"fmt"

	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Depth of field and motion blur. They read the HDR scene and its depth and write
// HDR, so bright points blur into real bokeh and bloom sees the result. Depth of
// field comes first: the lens blurs the light before the shutter smears it.

// Single-pass gather along a golden-angle spiral, after Dennis Gustafsson's "Bokeh
// depth of field in a single pass". Samples behind this pixel only count as far as
// this pixel is blurred, so a sharp subject does not bleed onto a blurred background.
const mbDOFFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uTex;
uniform sampler2D uDepth;
uniform vec2 uTexel;
uniform vec2 uNearFar;
uniform float uFocus;
uniform float uRange;
uniform float uMaxPx;

const float GOLDEN = 2.39996323;

float dist(vec2 uv) {
	float n = uNearFar.x, f = uNearFar.y;
	float d = texture(uDepth, uv).r * 2.0 - 1.0;
	return 2.0 * n * f / max(f + n - d * (f - n), 0.0001);
}
float blurPx(float z) { return clamp(abs(z - uFocus) / uRange, 0.0, 1.0) * uMaxPx; }
vec3 finite(vec3 c) { return (any(isnan(c)) || any(isinf(c))) ? vec3(0.0) : c; }

void main() {
	vec3 centre = finite(texture(uTex, vUV).rgb);
	float z = dist(vUV);
	float size = blurPx(z);
	// In focus with nothing blurred near it, as most of a sharp subject is: done.
	float most = size;
	for (int i = 0; i < 8; i++) {
		float a = float(i) * 0.7853982;
		most = max(most, blurPx(dist(vUV + vec2(cos(a), sin(a)) * uTexel * uMaxPx)));
	}
	if (most < 0.5) {
		frag = vec4(centre, 1.0);
		return;
	}
	// The spiral needs r squared over twice the step samples, so a big blur spaces them
	// out: at most about 50, or it costs more than the rest of the frame.
	float STEP = max(1.6, uMaxPx * uMaxPx / 100.0);
	vec3 sum = centre;
	float count = 1.0;
	float r = STEP;
	float ang = 0.0;
	while (r < uMaxPx) {
		vec2 uv = vUV + vec2(cos(ang), sin(ang)) * uTexel * r;
		vec3 c = finite(texture(uTex, uv).rgb);
		float sz = dist(uv);
		float s = blurPx(sz);
		if (sz > z) { s = clamp(s, 0.0, size * 2.0); }
		sum += mix(sum / count, c, smoothstep(r - 0.5, r + 0.5, s));
		count += 1.0;
		ang += GOLDEN;
		r += STEP / r;
	}
	frag = vec4(sum / count, 1.0);
}
`

// Camera motion blur: where each pixel was a frame ago, from its depth and the last
// frame's camera, and a line of samples along that. Things that move on their own
// are not blurred, only the camera's movement past them.
const mbMotionFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uTex;
uniform sampler2D uDepth;
uniform vec2 uTexel;
uniform mat4 uReproject;
uniform float uShutter;
uniform float uMaxPx;

float noise(vec2 p) {
	vec3 q = fract(vec3(p.xyx) * 0.1031);
	q += dot(q, q.yzx + 33.33);
	return fract((q.x + q.y) * q.z);
}
vec3 finite(vec3 c) { return (any(isnan(c)) || any(isinf(c))) ? vec3(0.0) : c; }

void main() {
	vec4 here = vec4(vUV * 2.0 - 1.0, texture(uDepth, vUV).r * 2.0 - 1.0, 1.0);
	vec4 was = uReproject * here;
	vec2 move = (here.xy - was.xy / max(was.w, 0.0001)) * 0.5 * uShutter;
	float px = length(move / uTexel);
	if (px < 0.5) {
		frag = vec4(finite(texture(uTex, vUV).rgb), 1.0);
		return;
	}
	if (px > uMaxPx) { move *= uMaxPx / px; }
	const int N = 9;
	float jitter = noise(gl_FragCoord.xy) - 0.5;
	vec3 sum = vec3(0.0);
	for (int i = 0; i < N; i++) {
		float t = (float(i) + jitter) / float(N - 1) - 0.5;
		sum += finite(texture(uTex, vUV + move * t).rgb);
	}
	frag = vec4(sum / float(N), 1.0);
}
`

type lensState struct {
	focus, focusRange, blurPx float32 // depth of field; focusRange 0 is off
	motion                    float32 // motion blur; 1 is a 1/60 s shutter, 0 is off
	maxMotionPx               float32
	dofProg, motionProg       uint32
	broken                    bool
	tex, fbo                  [2]uint32
	w, h                      int
	lastVP                    math32.Matrix4
	haveLast                  bool
	split                     bool // this frame had split-screen views
}

func (w *World) disposeLens() {
	l := &w.post.lens
	for i := range l.tex {
		if l.tex[i] != 0 {
			gl.DeleteTextures(1, &l.tex[i])
			l.tex[i] = 0
		}
		if l.fbo[i] != 0 {
			gl.DeleteFramebuffers(1, &l.fbo[i])
			l.fbo[i] = 0
		}
	}
	l.w, l.h = 0, 0
}

func (w *World) ensureLens(ww, hh int) bool {
	l := &w.post.lens
	if l.broken {
		return false
	}
	if l.dofProg == 0 {
		var err error
		if l.dofProg, err = postProgram(mbDOFFragment); err == nil {
			l.motionProg, err = postProgram(mbMotionFragment)
		}
		if err != nil {
			fmt.Println("PostFX lens:", err)
			l.broken = true
			return false
		}
	}
	if l.tex[0] != 0 && l.w == ww && l.h == hh {
		return true
	}
	w.disposeLens()
	for i := range l.tex {
		gl.GenTextures(1, &l.tex[i])
		gl.BindTexture(gl.TEXTURE_2D, l.tex[i])
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, int32(ww), int32(hh), 0, gl.RGBA, gl.HALF_FLOAT, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.GenFramebuffers(1, &l.fbo[i])
		gl.BindFramebuffer(gl.FRAMEBUFFER, l.fbo[i])
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, l.tex[i], 0)
		if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
			fmt.Println("PostFX lens: target incomplete")
			gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
			w.disposeLens()
			l.broken = true
			return false
		}
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	l.w, l.h = ww, hh
	return true
}

func postProgram(frag string) (uint32, error) {
	return compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbpostVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, frag},
	)
}

// reprojection is the matrix from this frame's clip space to last frame's, and whether
// to blur at all: not on the first frame, a still camera, or a cut.
func (w *World) reprojection() (math32.Matrix4, bool) {
	l := &w.post.lens
	var view, proj, vp, inv, re math32.Matrix4
	if w.cam == nil || l.split {
		l.haveLast = false
		return re, false
	}
	w.cam.ViewMatrix(&view)
	w.cam.ProjMatrix(&proj)
	vp.MultiplyMatrices(&proj, &view)
	last, had := l.lastVP, l.haveLast
	l.lastVP, l.haveLast = vp, true
	if !had || last == vp || inv.GetInverse(&vp) != nil {
		return re, false
	}
	re.MultiplyMatrices(&last, &inv)
	return re, !reprojectionIsCut(&re)
}

// reprojectionIsCut says a point mid-screen moved so far that the camera must have
// jumped to a new shot; smearing that across the picture would look like a glitch.
func reprojectionIsCut(re *math32.Matrix4) bool {
	p := math32.Vector4{X: 0, Y: 0, Z: 0.98, W: 1}
	p.ApplyMatrix4(re)
	if p.W <= 0 {
		return true
	}
	x, y := p.X/p.W, p.Y/p.W
	return x*x+y*y > 0.5
}

// applyLens runs the lens effects on the scene and returns the texture that holds
// the result, for bloom and the tone map to read.
func (w *World) applyLens(ww, hh int) uint32 {
	p := &w.post
	l := &p.lens
	src := p.color
	wantDOF := l.focusRange > 0 && w.cam != nil
	wantMotion := l.motion > 0
	if !wantMotion {
		l.haveLast = false
	}
	if !w.bloomAllowed() || p.rainOnly || p.depth == 0 || (!wantDOF && !wantMotion) {
		return src
	}
	if !w.ensureLens(ww, hh) {
		return src
	}
	scale := float32(hh) / 720
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.Viewport(0, 0, int32(ww), int32(hh))
	gl.BindVertexArray(p.vao)
	target := 0
	use := func(prog uint32) {
		gl.UseProgram(prog)
		gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uTex\x00")), 0)
		gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uDepth\x00")), 1)
		gl.Uniform2f(gl.GetUniformLocation(prog, gl.Str("uTexel\x00")), 1/float32(ww), 1/float32(hh))
	}
	draw := func() {
		gl.BindFramebuffer(gl.FRAMEBUFFER, l.fbo[target])
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, p.depth)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, src)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
		src = l.tex[target]
		target++
	}
	if wantDOF {
		near, far, _, _ := w.gtaoCamera()
		use(l.dofProg)
		gl.Uniform2f(gl.GetUniformLocation(l.dofProg, gl.Str("uNearFar\x00")), near, far)
		gl.Uniform1f(gl.GetUniformLocation(l.dofProg, gl.Str("uFocus\x00")), l.focus)
		gl.Uniform1f(gl.GetUniformLocation(l.dofProg, gl.Str("uRange\x00")), l.focusRange)
		gl.Uniform1f(gl.GetUniformLocation(l.dofProg, gl.Str("uMaxPx\x00")), l.blurPx*scale)
		draw()
	}
	if wantMotion {
		if re, ok := w.reprojection(); ok {
			use(l.motionProg)
			gl.UniformMatrix4fv(gl.GetUniformLocation(l.motionProg, gl.Str("uReproject\x00")), 1, false, &re[0])
			gl.Uniform1f(gl.GetUniformLocation(l.motionProg, gl.Str("uShutter\x00")), shutterFraction(l.motion, w.delta))
			gl.Uniform1f(gl.GetUniformLocation(l.motionProg, gl.Str("uMaxPx\x00")), l.maxMotionPx*scale)
			draw()
		}
	}
	gl.BindVertexArray(0)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.UseProgram(0)
	drainGL("PostFX lens")
	return src
}

// shutterFraction is how much of one frame's movement the shutter sees. The same blur
// at any frame rate: a frame twice as long moves things twice as far.
func shutterFraction(amount float32, dt float64) float32 {
	if dt < 1.0/240 {
		dt = 1.0 / 240
	}
	return amount / 60 / float32(dt)
}

func (w *World) lensCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetDepthOfField(focus, range [, blur]): sharp at focus meters, fully blurred
		// range meters either side of it, with a blur up to that many pixels at 720p.
		// No arguments, or a range of 0, turns it off.
		"setdepthoffield": n(func(a []value.Value) (value.Value, error) {
			l := &w.post.lens
			l.focus = float32(argN(a, 0, 0))
			l.focusRange = float32(argN(a, 1, 0))
			l.blurPx = float32(argN(a, 2, 10))
			if l.blurPx < 1 {
				l.blurPx = 1
			}
			if l.blurPx > 32 {
				l.blurPx = 32
			}
			if l.focusRange < 0 {
				l.focusRange = 0
			}
			if l.focusRange > 0 {
				w.wantPost()
			}
			return z()
		}),
		// SetMotionBlur(amount [, maxPixels]): blur from the camera's movement. 1 is a
		// 1/60 second shutter, at any frame rate. 0 turns it off.
		"setmotionblur": n(func(a []value.Value) (value.Value, error) {
			l := &w.post.lens
			l.motion = float32(argN(a, 0, 0))
			if l.motion < 0 {
				l.motion = 0
			}
			if l.motion > 4 {
				l.motion = 4
			}
			l.maxMotionPx = float32(argN(a, 1, 24))
			if l.maxMotionPx < 1 {
				l.maxMotionPx = 1
			}
			if l.motion > 0 {
				w.wantPost()
			}
			return z()
		}),
	}
}
