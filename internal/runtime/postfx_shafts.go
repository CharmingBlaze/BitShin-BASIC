package runtime

import (
	"fmt"

	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Light shafts: sunlight streaming past whatever stands between the eye and the sun. Sky
// near the sun is blurred outward along lines from the sun's place on screen, and the
// trees, peaks and buildings in front of it cut beams into that glow. After Mitchell,
// "Volumetric Light Scattering as a Post-Process", GPU Gems 3. It needs only depth.
//
// The other common way, walking each view ray through the shadow map, was tried first:
// between trees it found too little to show, and seen from the air it laid a haze over
// the sea. This one is also cheap enough for every tier above low. It only shows with
// the sun in or near the view, as it is drawn from the sun's place on screen.

// Three short passes, each blurring the last along the same lines with a step a tenth as
// long, as Crytek, three.js and Unreal do: 3 x 10 taps act like a thousand, so the shafts
// come out smooth without the banding one long pass leaves (Sousa, GDC 2008). The first
// pass's mask is the sky's own brightness above a threshold, as Unreal's is, so a cloud
// near the sun shapes the beams and thin gaps flicker less than with sky-or-not.
const mbShaftFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uDepth;
uniform sampler2D uColor;
uniform sampler2D uSrc;
uniform int uPass;
uniform vec2 uSunUV;
uniform float uAspect;
uniform float uFade;
uniform float uStep;
uniform float uNear;
uniform float uFar;

const int TAPS = 10;

float viewZ(float d) {
	float z = d * 2.0 - 1.0;
	return (2.0 * uNear * uFar) / (uFar + uNear - z * (uFar - uNear));
}

// Open sky, as bright as it is past a threshold, and closer to the sun the more it counts.
float maskAt(vec2 uv) {
	if (uv.x < 0.0 || uv.y < 0.0 || uv.x > 1.0 || uv.y > 1.0) { return 0.0; }
	if (texture(uDepth, uv).r < 0.9995) { return 0.0; }
	float lum = dot(texture(uColor, uv).rgb, vec3(0.2126, 0.7152, 0.0722));
	vec2 d = (uv - uSunUV) * vec2(uAspect, 1.0);
	return clamp(lum - 0.4, 0.0, 1.5) * exp(-dot(d, d) * 14.0);
}

void main() {
	vec2 stepUV = (uSunUV - vUV) * uStep;
	float noise = fract(52.9829189 * fract(dot(gl_FragCoord.xy, vec2(0.06711056, 0.00583715))));
	vec2 uv = vUV + stepUV * (uPass == 0 ? noise : 0.0);
	float sum = 0.0;
	for (int i = 0; i < TAPS; i++) {
		sum += uPass == 0 ? maskAt(uv) : texture(uSrc, uv).r;
		uv += stepUV;
	}
	float v = sum / float(TAPS);
	if (uPass == 2) {
		// Beams laid over something close by look painted on it, so near things take
		// less: half at the eye, all of it by 35 m. Open sky already holds the sun's glow
		// and only washes out with more, so it takes half; the beams are what stands out.
		float raw = texture(uDepth, vUV).r;
		float k = raw >= 0.9995 ? 0.5 : mix(0.45, 1.0, smoothstep(2.0, 35.0, viewZ(raw)));
		v *= uFade * k * 2.4;
	}
	frag = vec4(v, 0.0, 0.0, 1.0);
}
`

type shaftState struct {
	on       bool
	strength float32
	prog     uint32
	broken   bool
	tex, fbo [2]uint32
	w, h     int
}

func (w *World) disposeShafts() {
	s := &w.post.shafts
	for i := range s.tex {
		if s.tex[i] != 0 {
			gl.DeleteTextures(1, &s.tex[i])
			s.tex[i] = 0
		}
		if s.fbo[i] != 0 {
			gl.DeleteFramebuffers(1, &s.fbo[i])
			s.fbo[i] = 0
		}
	}
	s.w, s.h = 0, 0
}

func (w *World) ensureShafts(ww, hh int) bool {
	s := &w.post.shafts
	if s.broken {
		return false
	}
	if s.prog == 0 {
		p, err := postProgram(mbShaftFragment)
		if err != nil {
			fmt.Println("Light shafts:", err)
			s.broken = true
			return false
		}
		s.prog = p
	}
	if w.ensureGTAOProg() != nil { // its blur keeps to one surface
		s.broken = true
		return false
	}
	aw, ah := max(ww/2, 1), max(hh/2, 1)
	if s.tex[0] != 0 && s.w == aw && s.h == ah {
		return true
	}
	w.disposeShafts()
	for i := range s.tex {
		gl.GenTextures(1, &s.tex[i])
		gl.BindTexture(gl.TEXTURE_2D, s.tex[i])
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R16F, int32(aw), int32(ah), 0, gl.RED, gl.HALF_FLOAT, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.GenFramebuffers(1, &s.fbo[i])
		gl.BindFramebuffer(gl.FRAMEBUFFER, s.fbo[i])
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.tex[i], 0)
		if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
			gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
			w.disposeShafts()
			s.broken = true
			return false
		}
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	s.w, s.h = aw, ah
	return true
}

// sunOnScreen is where the sun is in the picture, and how much to show: full when it is
// in view, fading out as it leaves, and nothing behind the camera or below the horizon.
func (w *World) sunOnScreen(sun math32.Vector3) (math32.Vector2, float32, bool) {
	var uv math32.Vector2
	if w.cam == nil || w.post.lens.split || sun.Y < 0.0 {
		return uv, 0, false
	}
	var view, proj, vp math32.Matrix4
	w.cam.ViewMatrix(&view)
	w.cam.ProjMatrix(&proj)
	vp.MultiplyMatrices(&proj, &view)
	eye := worldPos(w.cam.GetNode())
	far := math32.Vector4{X: eye.X + sun.X*1000, Y: eye.Y + sun.Y*1000, Z: eye.Z + sun.Z*1000, W: 1}
	far.ApplyMatrix4(&vp)
	if far.W <= 0 {
		return uv, 0, false
	}
	uv = math32.Vector2{X: far.X/far.W*0.5 + 0.5, Y: far.Y/far.W*0.5 + 0.5}
	// How far outside the frame, in frame widths: none in view, fading out by 0.6 beyond.
	out := math32.Max(math32.Max(-uv.X, uv.X-1), math32.Max(-uv.Y, uv.Y-1))
	fade := 1 - math32.Clamp(out/0.6, 0, 1)
	fade *= math32.Clamp(sun.Y/0.06, 0, 1) // sinking into the horizon
	// And by how far the view is turned from the sun: as it nears side-on, the sun's place
	// on screen runs off toward infinity and the samples spread thin (Mitchell, 13.6).
	fwd := math32.Vector3{X: -view[2], Y: -view[6], Z: -view[10]}
	fade *= smoothstep32(0.2, 0.6, fwd.Dot(&sun))
	// A guard band, so a sun just off screen is not read from far away.
	uv.X = math32.Clamp(uv.X, -0.6, 1.6)
	uv.Y = math32.Clamp(uv.Y, -0.6, 1.6)
	return uv, fade, fade > 0.01
}

// fillShafts draws the half-resolution amount of scattered sun, and reports whether there
// is any.
func (w *World) fillShafts(ww, hh int, picture uint32) bool {
	p := &w.post
	s := &p.shafts
	if !s.on || p.depth == 0 || p.rainOnly || !w.bloomAllowed() || p.vao == 0 {
		return false
	}
	sun := w.shadowLightDir()
	sun.Normalize()
	sunUV, fade, ok := w.sunOnScreen(sun)
	if !ok || !w.ensureShafts(ww, hh) {
		return false
	}
	near, far, _, aspect := w.gtaoCamera()
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.Viewport(0, 0, int32(s.w), int32(s.h))
	gl.BindVertexArray(p.vao)
	gl.UseProgram(s.prog)
	loc := func(name string) int32 { return gl.GetUniformLocation(s.prog, gl.Str(name+"\x00")) }
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, p.depth)
	gl.Uniform1i(loc("uDepth"), 0)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, picture)
	gl.Uniform1i(loc("uColor"), 1)
	gl.Uniform1i(loc("uSrc"), 2)
	gl.Uniform2f(loc("uSunUV"), sunUV.X, sunUV.Y)
	gl.Uniform1f(loc("uAspect"), aspect)
	gl.Uniform1f(loc("uFade"), fade)
	gl.Uniform1f(loc("uNear"), near)
	gl.Uniform1f(loc("uFar"), far)
	// Long steps first, then a tenth, then a hundredth: each pass smooths the one before.
	steps := [3]float32{0.085, 0.0085, 0.00085}
	from, to := uint32(0), 0
	for pass := 0; pass < 3; pass++ {
		gl.BindFramebuffer(gl.FRAMEBUFFER, s.fbo[to])
		gl.ActiveTexture(gl.TEXTURE2)
		gl.BindTexture(gl.TEXTURE_2D, from)
		gl.Uniform1i(loc("uPass"), int32(pass))
		gl.Uniform1f(loc("uStep"), steps[pass])
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
		from, to = s.tex[to], 1-to
	}
	// Three passes end in tex[0]: 0 writes it, 1 reads it into 1, 2 reads that back.
	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	gl.BindVertexArray(0)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.UseProgram(0)
	drainGL("light shafts")
	return true
}

// shaftColor is the sun's light, in linear terms, scaled by the strength asked for.
func (w *World) shaftColor() (float32, float32, float32) {
	c := math32.Color{R: 1, G: 0.95, B: 0.85}
	if e := w.ents[w.shadow.lightID]; e != nil {
		if d, ok := e.node.(*light.Directional); ok {
			c = d.Color()
			k := d.Intensity()
			c.R, c.G, c.B = c.R*k, c.G*k, c.B*k
		}
	}
	s := w.post.shafts.strength
	lin := func(v float32) float32 { return math32.Pow(math32.Max(v, 0), 2.2) * s }
	return lin(c.R), lin(c.G), lin(c.B)
}

func (w *World) shaftCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetLightShafts(on [, strength]): beams of sunlight past whatever stands in front
		// of the sun, when it is in or near the view. Strength defaults to 1. Not on low tier.
		"setlightshafts": n(func(a []value.Value) (value.Value, error) {
			s := &w.post.shafts
			s.on = argI(a, 0, 1) != 0
			s.strength = float32(argN(a, 1, 1))
			if s.on {
				w.wantPost()
			}
			return value.Num(float64(b01(s.on))), nil
		}),
	}
}
