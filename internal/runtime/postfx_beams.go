package runtime

import (
	"fmt"

	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Lamp beams: light from spot and point lights seen in the air, as in fog or dust. For
// each pixel the view ray is walked through the part of the light's reach it crosses
// (for a spot, the part inside the cone), adding the light that air would scatter
// toward the eye: the light's color, fading with distance, more of it looking toward
// the lamp (Henyey-Greenstein). Where the lamp casts shadows, each step also asks its
// shadow map, so a fence, a trunk or a window frame cuts the beam into shafts.
//
// Unlike SetLightShafts, which works from the sun's place on screen, this works in the
// world, so it shows with the lamp off screen too. It runs at half resolution, sixteen
// steps per light, then a blur that keeps to one surface. Four lights at most.

const maxBeamLights = 4

const mbBeamFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform int uPass;          // 0 march, 1 and 2 blur across and down
uniform sampler2D uDepth;
uniform sampler2D uSrc;
uniform sampler2D uAtlas;
uniform int uAtlasCols;
uniform int uAtlasRows;
uniform mat4 uInvVP;
uniform vec3 uEye;
uniform float uNear;
uniform float uFar;
uniform vec2 uTexel;        // of the half-resolution target
uniform int uN;
uniform vec4 uPos[4];       // position, range
uniform vec4 uDir[4];       // way it shines, cosine of the cone's edge; w < -1 for a point
uniform vec3 uCol[4];       // linear color x intensity x strength
uniform vec4 uAtt[4];       // the light's linear, quadratic and constant decay
uniform vec4 uCone[4];      // a spot's cutoff and inner angle in degrees
uniform int uShadow[4];     // -1 none, 0-1 spot slot, 10-11 point slot
uniform int uTile[4];       // atlas tile (a point's first face)
uniform mat4 uSpotVP[2];
uniform mat4 uPointVP[12];

const int STEPS = 16;
const float G = 0.35;
const float AIR = 0.08; // how much light a metre of air turns toward the eye, at strength 1

// The same falloff and cone as the lit surfaces (shadeAttenuation, shadeSpot), so a
// beam is as bright as the ground it lands on.
float atten(float dist, vec4 a) {
	float lin = a.x, quad = a.y, constant = a.z;
	if (constant < -0.5) {
		float range = 1.0 / max(lin, 1.0e-4);
		// Unlike a surface, air right at the bulb would outshine the whole beam, so
		// the nearest tenth of the reach counts as that distance away.
		float d2 = max(dist * dist, range * range * 0.01);
		float window = clamp(1.0 - pow(clamp(dist / range, 0.0, 1.0), 4.0), 0.0, 1.0);
		return window / d2;
	}
	if (quad <= 1.0e-4 && constant <= 1.0e-4) {
		float x = clamp(dist * lin, 0.0, 1.0);
		float edge = (1.0 - x) * (1.0 - x);
		return edge / (1.0 + 3.0 * x * x);
	}
	float c = constant < 1.0e-4 ? 1.0 : constant;
	return 1.0 / (c + lin * dist + quad * dist * dist);
}

float coneAt(float cosTheta, vec4 k, float constant) {
	float outerDeg = clamp(k.x, 1.0, 89.0);
	float innerDeg = outerDeg * 0.62;
	if (k.y > 0.5 && k.y < outerDeg) { innerDeg = k.y; }
	float cosOuter = cos(radians(outerDeg));
	float cosInner = cos(radians(max(innerDeg, 0.5)));
	if (constant < -0.5) {
		float cone = clamp((cosTheta - cosOuter) / max(0.001, cosInner - cosOuter), 0.0, 1.0);
		return cone * cone;
	}
	return smoothstep(cosOuter, max(cosInner, cosOuter + 0.001), cosTheta);
}

float viewZ(float d) {
	float z = d * 2.0 - 1.0;
	return (2.0 * uNear * uFar) / (uFar + uNear - z * (uFar - uNear));
}

vec2 tileUV(int tile, vec2 p) {
	int cols = max(uAtlasCols, 1);
	int rows = max(uAtlasRows, 1);
	int col = tile - (tile / cols) * cols;
	int row = tile / cols;
	p = clamp(p, 0.008, 0.992);
	return vec2((float(col) + p.x) / float(cols), (float(row) + p.y) / float(rows));
}

int cubeFace(vec3 d) {
	vec3 a = abs(d);
	if (a.x >= a.y && a.x >= a.z) { return d.x > 0.0 ? 0 : 1; }
	if (a.y >= a.z) { return d.y > 0.0 ? 2 : 3; }
	return d.z > 0.0 ? 4 : 5;
}

// lit is 1 where the light reaches x and 0 where something stands in its way.
float lit(int i, vec3 x, vec3 L) {
	int s = uShadow[i];
	if (s < 0) { return 1.0; }
	vec4 lsp;
	int tile = uTile[i];
	if (s >= 10) {
		int f = cubeFace(L);
		lsp = uPointVP[(s - 10) * 6 + f] * vec4(x, 1.0);
		tile += f;
	} else {
		lsp = uSpotVP[s] * vec4(x, 1.0);
	}
	vec3 p = lsp.xyz / max(lsp.w, 0.0001) * 0.5 + 0.5;
	if (p.z >= 0.999 || p.x <= 0.0 || p.x >= 1.0 || p.y <= 0.0 || p.y >= 1.0) { return 1.0; }
	float d = texture(uAtlas, tileUV(tile, p.xy)).r;
	return p.z - 0.0008 <= d ? 1.0 : 0.0;
}

// coneSpan narrows [t0, t1] to where the ray is inside the spot's cone, on its lit side.
bool coneSpan(vec3 o, vec3 v, vec3 P, vec3 D, float c, inout float t0, inout float t1) {
	vec3 co = o - P;
	float vd = dot(v, D), cd = dot(co, D), c2 = c * c;
	float a = vd * vd - c2;
	float b = 2.0 * (vd * cd - dot(v, co) * c2);
	float k = cd * cd - dot(co, co) * c2;
	float lo = 1e9, hi = -1e9;
	float disc = b * b - 4.0 * a * k;
	if (abs(a) < 1e-6) {
		return true; // grazing the cone's side: keep the whole reach
	}
	if (disc < 0.0) {
		if (a < 0.0) { return false; }
		lo = t0; hi = t1; // inside the double cone throughout
	} else {
		float sq = sqrt(disc);
		float r1 = (-b - sq) / (2.0 * a), r2 = (-b + sq) / (2.0 * a);
		if (r1 > r2) { float t = r1; r1 = r2; r2 = t; }
		// Inside where f >= 0: between the roots when a < 0, outside them when a > 0.
		vec2 span[2];
		int n = 0;
		if (a < 0.0) { span[0] = vec2(r1, r2); n = 1; }
		else { span[0] = vec2(-1e9, r1); span[1] = vec2(r2, 1e9); n = 2; }
		for (int j = 0; j < 2; j++) {
			if (j >= n) { break; }
			float s0 = max(span[j].x, t0), s1 = min(span[j].y, t1);
			if (s1 <= s0) { continue; }
			// The back half of the double cone is dark.
			if (dot(o + v * (0.5 * (s0 + s1)) - P, D) <= 0.0) { continue; }
			lo = min(lo, s0); hi = max(hi, s1);
		}
	}
	lo = max(lo, t0); hi = min(hi, t1);
	if (hi <= lo) { return false; }
	t0 = lo; t1 = hi;
	return true;
}

vec3 march(vec2 uv) {
	float raw = texture(uDepth, uv).r;
	vec4 w = uInvVP * vec4(uv * 2.0 - 1.0, raw * 2.0 - 1.0, 1.0);
	vec3 hit = w.xyz / w.w;
	vec3 v = hit - uEye;
	float tEnd = length(v);
	v /= max(tEnd, 1e-4);
	if (raw >= 0.9999) { tEnd = 1e6; }
	float noise = fract(52.9829189 * fract(dot(gl_FragCoord.xy, vec2(0.06711056, 0.00583715))));
	vec3 sum = vec3(0.0);
	for (int i = 0; i < 4; i++) {
		if (i >= uN) { break; }
		vec3 P = uPos[i].xyz;
		float R = uPos[i].w;
		vec3 oc = uEye - P;
		float b = dot(oc, v);
		float h = b * b - (dot(oc, oc) - R * R);
		if (h <= 0.0) { continue; }
		h = sqrt(h);
		float t0 = max(-b - h, 0.0), t1 = min(-b + h, tEnd);
		if (t1 <= t0) { continue; }
		bool spot = uDir[i].w > -1.0;
		vec3 D = uDir[i].xyz;
		float c = uDir[i].w;
		if (spot && !coneSpan(uEye, v, P, D, c, t0, t1)) { continue; }
		float dt = (t1 - t0) / float(STEPS);
		vec3 acc = vec3(0.0);
		for (int k = 0; k < STEPS; k++) {
			vec3 x = uEye + v * (t0 + (float(k) + noise) * dt);
			vec3 L = x - P;
			float d = length(L);
			vec3 l = L / max(d, 1e-4);
			float fall = atten(d, uAtt[i]);
			float cone = spot ? coneAt(dot(l, D), uCone[i], uAtt[i].z) : 1.0;
			if (fall * cone < 1e-6) { continue; }
			float ct = dot(l, -v);
			float hg = (1.0 - G * G) / (12.566371 * pow(1.0 + G * G - 2.0 * G * ct, 1.5));
			acc += vec3(fall * cone * hg * lit(i, x, L));
		}
		// Per metre walked, so a long beam is brighter than a short one, as in air.
		sum += uCol[i] * acc * (dt * AIR);
	}
	return sum;
}

void main() {
	if (uPass == 0) {
		frag = vec4(march(vUV), 1.0);
		return;
	}
	// A blur that keeps to one surface, so a beam in front of a wall does not bleed
	// onto the wall behind or around it.
	vec2 dir = uPass == 1 ? vec2(uTexel.x, 0.0) : vec2(0.0, uTexel.y);
	float zc = viewZ(texture(uDepth, vUV).r);
	vec3 acc = vec3(0.0);
	float wsum = 0.0;
	for (int k = -3; k <= 3; k++) {
		vec2 uv = vUV + dir * float(k) * 1.5;
		float z = viewZ(texture(uDepth, uv).r);
		float wk = exp(-float(k * k) / 8.0) * exp(-abs(z - zc) / (0.05 * zc + 0.05));
		acc += texture(uSrc, uv).rgb * wk;
		wsum += wk;
	}
	frag = vec4(acc / max(wsum, 1e-4), 1.0);
}
`

type beamState struct {
	prog     uint32
	broken   bool
	tex, fbo [2]uint32
	w, h     int
}

func (w *World) disposeBeams() {
	s := &w.post.beams
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

func (w *World) ensureBeams(ww, hh int) bool {
	s := &w.post.beams
	if s.broken {
		return false
	}
	if s.prog == 0 {
		p, err := postProgram(mbBeamFragment)
		if err != nil {
			fmt.Println("Light beams:", err)
			s.broken = true
			return false
		}
		s.prog = p
	}
	aw, ah := max(ww/2, 1), max(hh/2, 1)
	if s.tex[0] != 0 && s.w == aw && s.h == ah {
		return true
	}
	w.disposeBeams()
	for i := range s.tex {
		gl.GenTextures(1, &s.tex[i])
		gl.BindTexture(gl.TEXTURE_2D, s.tex[i])
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, int32(aw), int32(ah), 0, gl.RGBA, gl.HALF_FLOAT, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.GenFramebuffers(1, &s.fbo[i])
		gl.BindFramebuffer(gl.FRAMEBUFFER, s.fbo[i])
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.tex[i], 0)
		if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
			gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
			w.disposeBeams()
			s.broken = true
			return false
		}
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	s.w, s.h = aw, ah
	return true
}

// beamLight is one lamp the pass draws this frame.
type beamLight struct {
	pos, dir math32.Vector3
	rng, cos float32 // cos < -1 for a point light
	col      math32.Color
	att      [3]float32 // linear, quadratic, constant decay
	cone     [2]float32 // cutoff and inner angle, degrees
	shadow   int        // -1, a spot slot, or 10 + a point slot
	tile     int
}

// beamLights are the visible lamps with a beam, nearest the eye first, four at most.
func (w *World) beamLights() []beamLight {
	if w.cam == nil {
		return nil
	}
	eye := worldPos(w.cam.GetNode())
	var out []beamLight
	var dist []float32
	sh := &w.shadow
	shadows := sh.on && sh.ready && sh.tex != 0 && sh.filter < shadowFilterEVSM
	for _, m := range w.entityOrder() {
		e := m.e
		if e == nil || e.beam <= 0 || e.node == nil || (e.lgtKind != 2 && e.lgtKind != 3) || !e.node.GetNode().Visible() {
			continue
		}
		lc, ok := e.node.(interface {
			Color() math32.Color
			Intensity() float32
		})
		if !ok {
			continue
		}
		b := beamLight{pos: entityWorldPos(e), rng: entityLightRange(e), cos: -2, shadow: -1}
		c := lc.Color()
		k := lc.Intensity() * e.beam
		lin := func(v float32) float32 { return math32.Pow(math32.Max(v, 0), 2.2) * k }
		b.col = math32.Color{R: lin(c.R), G: lin(c.G), B: lin(c.B)}
		if d, ok := e.node.(interface {
			LinearDecay() float32
			QuadraticDecay() float32
		}); ok {
			b.att = [3]float32{d.LinearDecay(), d.QuadraticDecay(), lightConstant(e.node)}
		}
		if b.att[2] >= -0.5 && w.linearActive() {
			b.col.R, b.col.G, b.col.B = b.col.R*math32.Pi, b.col.G*math32.Pi, b.col.B*math32.Pi
		}
		if sl, ok := e.node.(*light.Spot); ok {
			b.dir = spotShineDir(e.node.GetNode())
			b.cos = entitySpotCos(e)
			b.cone = [2]float32{sl.CutoffAngle(), sl.AngularDecay()}
		}
		if shadows {
			for i := 0; i < sh.spotN; i++ {
				if sh.spotE[i] == e {
					b.shadow, b.tile = i, sh.spotTile[i]
				}
			}
			for i := 0; i < sh.pointN; i++ {
				if sh.pointE[i] == e {
					b.shadow, b.tile = 10+i, sh.pointTile[i]
				}
			}
		}
		d := b.pos.DistanceTo(&eye) - b.rng
		// Keep the nearest four.
		at := len(out)
		for at > 0 && dist[at-1] > d {
			at--
		}
		if at >= maxBeamLights {
			continue
		}
		out = append(out, beamLight{})
		dist = append(dist, 0)
		copy(out[at+1:], out[at:])
		copy(dist[at+1:], dist[at:])
		out[at], dist[at] = b, d
		if len(out) > maxBeamLights {
			out, dist = out[:maxBeamLights], dist[:maxBeamLights]
		}
	}
	return out
}

// fillBeams draws the half-resolution light in the air, and reports whether there is any.
func (w *World) fillBeams(ww, hh int) bool {
	p := &w.post
	s := &p.beams
	if p.depth == 0 || p.rainOnly || !w.bloomAllowed() || p.vao == 0 || p.lens.split {
		return false
	}
	lights := w.beamLights()
	if len(lights) == 0 || !w.ensureBeams(ww, hh) {
		return false
	}
	var view, proj, vp, inv math32.Matrix4
	w.cam.ViewMatrix(&view)
	w.cam.ProjMatrix(&proj)
	vp.MultiplyMatrices(&proj, &view)
	if err := inv.GetInverse(&vp); err != nil {
		return false
	}
	eye := worldPos(w.cam.GetNode())
	near, far, _, _ := w.gtaoCamera()

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
	gl.Uniform1i(loc("uSrc"), 1)
	atlas := uint32(0)
	sh := &w.shadow
	if sh.on && sh.ready && sh.filter < shadowFilterEVSM {
		atlas = sh.tex
	}
	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, atlas)
	gl.Uniform1i(loc("uAtlas"), 2)
	gl.Uniform1i(loc("uAtlasCols"), int32(max(sh.atlasCols, 1)))
	gl.Uniform1i(loc("uAtlasRows"), int32(max(sh.atlasRows, 1)))
	gl.UniformMatrix4fv(loc("uInvVP"), 1, false, &inv[0])
	gl.Uniform3f(loc("uEye"), eye.X, eye.Y, eye.Z)
	gl.Uniform1f(loc("uNear"), near)
	gl.Uniform1f(loc("uFar"), far)
	gl.Uniform2f(loc("uTexel"), 1/float32(s.w), 1/float32(s.h))
	gl.Uniform1i(loc("uN"), int32(len(lights)))
	for i, b := range lights {
		shadow := b.shadow
		if atlas == 0 {
			shadow = -1
		}
		gl.Uniform4f(loc(fmt.Sprintf("uPos[%d]", i)), b.pos.X, b.pos.Y, b.pos.Z, b.rng)
		gl.Uniform4f(loc(fmt.Sprintf("uDir[%d]", i)), b.dir.X, b.dir.Y, b.dir.Z, b.cos)
		gl.Uniform3f(loc(fmt.Sprintf("uCol[%d]", i)), b.col.R, b.col.G, b.col.B)
		gl.Uniform4f(loc(fmt.Sprintf("uAtt[%d]", i)), b.att[0], b.att[1], b.att[2], 0)
		gl.Uniform4f(loc(fmt.Sprintf("uCone[%d]", i)), b.cone[0], b.cone[1], 0, 0)
		gl.Uniform1i(loc(fmt.Sprintf("uShadow[%d]", i)), int32(shadow))
		gl.Uniform1i(loc(fmt.Sprintf("uTile[%d]", i)), int32(b.tile))
	}
	gl.UniformMatrix4fv(loc("uSpotVP"), maxShadowSpots, false, &sh.spotVP[0][0])
	gl.UniformMatrix4fv(loc("uPointVP"), maxPointFaces, false, &sh.pointVP[0][0])

	// March into 0, blur across into 1, blur down back into 0.
	for pass := 0; pass < 3; pass++ {
		to := pass & 1
		gl.BindFramebuffer(gl.FRAMEBUFFER, s.fbo[to])
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, s.tex[1-to])
		if pass == 0 {
			gl.BindTexture(gl.TEXTURE_2D, 0)
		}
		gl.Uniform1i(loc("uPass"), int32(pass))
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	}

	gl.BindVertexArray(0)
	for _, u := range []uint32{gl.TEXTURE2, gl.TEXTURE1, gl.TEXTURE0} {
		gl.ActiveTexture(u)
		gl.BindTexture(gl.TEXTURE_2D, 0)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.UseProgram(0)
	drainGL("light beams")
	return true
}

func (w *World) beamCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// LightBeam(light, strength): the light of a spot or point light seen in the air,
		// as in fog or dust, cut by shadows where the light casts them. 1 is a light haze;
		// 0 turns it off. The nearest four lights with a beam draw. Not on low tier.
		"lightbeam": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.lgtKind != 2 && e.lgtKind != 3 {
				return value.Value{}, fmt.Errorf("LightBeam: entity %d is not a point or spot light", argI(a, 0, 0))
			}
			e.beam = float32(argN(a, 1, 1))
			if e.beam < 0 {
				e.beam = 0
			}
			if e.beam > 0 {
				w.wantPost()
			}
			return z()
		}),
	}
}
