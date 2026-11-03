package runtime

import "github.com/go-gl/gl/v3.3-core/gl"

// Real FXAA: finds an edge's direction, walks along it to both ends, and blends
// across it by how far this pixel is from the nearer end, as in Lottes' FXAA
// paper. It runs on the finished display image, after tone mapping, as FXAA
// needs: on HDR values bright edges would never register. The old SetFXAA was a
// contrast-gated blur on the HDR input, which softened detail and left bright
// edges jagged. Retro looks keep their hard pixels and skip it.
const mbFXAAFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uTex;
uniform vec2 uTexel;
uniform float uGrain;
uniform float uTime;

const int STEPS = 10;
const float EDGE_MIN = 0.0312;   // darks: below this contrast there is no edge
const float EDGE_REL = 0.125;    // relative to the local maximum
const float SUBPIX = 0.75;       // how much single-pixel specks are softened
float stepSize(int i) { return i < 2 ? 1.0 : (i < 5 ? 1.5 : (i < 8 ? 2.0 : 4.0)); }

float luma(vec3 c) { return dot(c, vec3(0.299, 0.587, 0.114)); }
float lumaAt(vec2 uv) { return luma(textureLod(uTex, uv, 0.0).rgb); }
float hash12(vec2 p) {
	vec3 p3 = fract(vec3(p.xyx) * 0.1031);
	p3 += dot(p3, p3.yzx + 33.33);
	return fract((p3.x + p3.y) * p3.z);
}
// Grain goes on after the edges are found. Before, FXAA took every speck for an edge
// and smoothed the grain away, and the picture with it.
vec4 grained(vec3 c) {
	if (uGrain > 0.001) {
		c += (hash12(vUV * vec2(1920.0, 1080.0) + uTime * 13.0) - 0.5) * uGrain;
	}
	return vec4(c, 1.0);
}

void main() {
	vec2 px = uTexel;
	vec3 centre = textureLod(uTex, vUV, 0.0).rgb;
	float m = luma(centre);
	float n = lumaAt(vUV + vec2(0.0, px.y));
	float s = lumaAt(vUV - vec2(0.0, px.y));
	float e = lumaAt(vUV + vec2(px.x, 0.0));
	float w = lumaAt(vUV - vec2(px.x, 0.0));
	float hi = max(m, max(max(n, s), max(e, w)));
	float lo = min(m, min(min(n, s), min(e, w)));
	float range = hi - lo;
	if (range < max(EDGE_MIN, hi * EDGE_REL)) {
		frag = grained(centre);
		return;
	}
	float ne = lumaAt(vUV + px);
	float nw = lumaAt(vUV + vec2(-px.x, px.y));
	float se = lumaAt(vUV + vec2(px.x, -px.y));
	float sw = lumaAt(vUV - px);

	// How far a lone speck stands out: it blends even with no long edge.
	float avg = (2.0 * (n + s + e + w) + ne + nw + se + sw) / 12.0;
	float sub = clamp(abs(avg - m) / range, 0.0, 1.0);
	sub = smoothstep(0.0, 1.0, sub);
	float subBlend = sub * sub * SUBPIX;

	// Horizontal or vertical edge, from the second differences.
	float horiz = abs(n + s - 2.0 * m) * 2.0 + abs(ne + se - 2.0 * e) + abs(nw + sw - 2.0 * w);
	float vert = abs(e + w - 2.0 * m) * 2.0 + abs(ne + nw - 2.0 * n) + abs(se + sw - 2.0 * s);
	bool isHoriz = horiz >= vert;

	// Which side of the pixel the edge lies on.
	float lumaP = isHoriz ? n : e;
	float lumaN = isHoriz ? s : w;
	float gradP = abs(lumaP - m);
	float gradN = abs(lumaN - m);
	float across = isHoriz ? px.y : px.x;
	float gradient = gradP;
	float other = lumaP;
	if (gradN > gradP) {
		across = -across;
		gradient = gradN;
		other = lumaN;
	}

	// Walk along the edge both ways until the contrast is lost.
	vec2 uvEdge = vUV;
	vec2 along = isHoriz ? vec2(px.x, 0.0) : vec2(0.0, px.y);
	if (isHoriz) { uvEdge.y += across * 0.5; } else { uvEdge.x += across * 0.5; }
	float edgeLuma = (m + other) * 0.5;
	float limit = gradient * 0.25;
	vec2 up = uvEdge;
	vec2 down = uvEdge;
	float dp = 0.0, dn = 0.0;
	bool doneP = false, doneN = false;
	for (int i = 0; i < STEPS && !(doneP && doneN); i++) {
		float st = stepSize(i);
		if (!doneP) {
			up += along * st;
			dp = lumaAt(up) - edgeLuma;
			doneP = abs(dp) >= limit;
		}
		if (!doneN) {
			down -= along * st;
			dn = lumaAt(down) - edgeLuma;
			doneN = abs(dn) >= limit;
		}
	}
	float distP = isHoriz ? up.x - vUV.x : up.y - vUV.y;
	float distN = isHoriz ? vUV.x - down.x : vUV.y - down.y;
	bool nearerP = distP <= distN;
	float nearest = nearerP ? distP : distN;
	float ending = nearerP ? dp : dn;

	// Blend only on the side where the edge's end really steps away from us.
	float edgeBlend = 0.0;
	if ((ending < 0.0) != (m - edgeLuma < 0.0)) {
		edgeBlend = 0.5 - nearest / max(distP + distN, 1e-6);
	}
	float blend = max(edgeBlend, subBlend);
	vec2 uv = vUV;
	if (isHoriz) { uv.y += across * blend; } else { uv.x += across * blend; }
	frag = grained(textureLod(uTex, uv, 0.0).rgb);
}
`

// fxaaWanted says whether this frame's post pass ends with real FXAA.
func (w *World) fxaaWanted() bool {
	r := &w.retroScr
	return w.post.fxaa && r.lowW == 0 && r.levels <= 0 && len(r.palette) == 0 && !w.post.rainOnly && w.ensureFXAA()
}

func (w *World) ensureFXAA() bool {
	f := &w.fsr
	if f.fxaa != 0 {
		return true
	}
	if f.fxaaFailed {
		return false
	}
	p, err := fsrProgram(mbFXAAFragment)
	if err != nil {
		f.fxaaFailed = true
		return false
	}
	f.fxaa = p
	return true
}

// drawFXAA draws the display image in src, of pw x ph, to the bound target.
func (w *World) drawFXAA(src uint32, pw, ph int) {
	f := &w.fsr
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, src)
	gl.UseProgram(f.fxaa)
	gl.Uniform1i(gl.GetUniformLocation(f.fxaa, gl.Str("uTex\x00")), 0)
	gl.Uniform2f(gl.GetUniformLocation(f.fxaa, gl.Str("uTexel\x00")), 1/float32(pw), 1/float32(ph))
	gl.Uniform1f(gl.GetUniformLocation(f.fxaa, gl.Str("uGrain\x00")), w.post.grain)
	gl.Uniform1f(gl.GetUniformLocation(f.fxaa, gl.Str("uTime\x00")), float32(w.wx.time))
	gl.BindVertexArray(w.post.vao)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.BindVertexArray(0)
	gl.BindTexture(gl.TEXTURE_2D, 0)
}
