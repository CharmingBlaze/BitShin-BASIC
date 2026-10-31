package runtime

const mbpostVertex = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aUV;
out vec2 vUV;
void main() {
	vUV = aUV;
	gl_Position = vec4(aPos, 0.0, 1.0);
}
`

// One-pass tonemap + cheap bloom (9-tap bright) + optional FXAA + grade.
// Contact shading is a separate half-resolution horizon pass.
const mbpostFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uTex;
uniform float uExposure;
uniform int uTonemap;
uniform float uBloom;
uniform int uFXAA;
uniform float uContrast;
uniform float uSat;
uniform vec3 uTint;
uniform vec2 uTexel;
uniform float uRain;
uniform float uTime;
uniform float uWet;
uniform int uRainOnly;
uniform float uGrain;
uniform float uVignette;
uniform float uCA;
uniform float uSharpen;
uniform vec2 uSrcSize;
uniform float uRetroLevels;
uniform float uRetroDither;
uniform int uRetroPalN;
uniform sampler2D uRetroPal;
uniform int uLUTOn;
uniform sampler3D uLUT;
uniform float uLUTAmount;
uniform float uLUTSize;
uniform int uOutlineOn;
uniform sampler2D uDepth;
uniform vec2 uNearFar;
uniform float uOutlineWidth;
uniform vec3 uOutlineColor;
uniform float uOutlineDepth;

uniform int uSSAO;
uniform sampler2D uAOTex;
uniform int uBloomChain;
uniform sampler2D uBloom0;
uniform sampler2D uBloom1;
uniform sampler2D uBloom2;
uniform sampler2D uBloom3;
uniform sampler2D uBloom4;

float luma(vec3 c) { return dot(c, vec3(0.299, 0.587, 0.114)); }

vec3 tonemapReinhard(vec3 c) {
	return c / (c + vec3(1.0));
}

// Khronos PBR Neutral. Base colors in a mid range stay put; only highlights compress.
vec3 tonemapNeutral(vec3 color) {
	const float startCompression = 0.76;
	const float desaturation = 0.15;
	float x = min(color.r, min(color.g, color.b));
	float offset = x < 0.08 ? x - 6.25 * x * x : 0.04;
	color -= offset;
	float peak = max(color.r, max(color.g, color.b));
	if (peak < startCompression) { return max(color, vec3(0.0)); }
	float d = 1.0 - startCompression;
	float newPeak = 1.0 - d * d / (peak + d - startCompression);
	color *= newPeak / peak;
	float g = 1.0 / (desaturation * (peak - newPeak) + 1.0);
	return max(mix(color, vec3(newPeak), 1.0 - g), vec3(0.0));
}

vec3 tonemapACES(vec3 x) {
	const float a = 2.51;
	const float b = 0.03;
	const float c = 2.43;
	const float d = 0.59;
	const float e = 0.14;
	return clamp((x * (a * x + b)) / (x * (c * x + d) + e), 0.0, 1.0);
}

vec3 tonemap(vec3 c) {
	c *= max(uExposure, 0.05);
	if (uTonemap == 1) { return tonemapNeutral(c); }
	if (uTonemap == 2) { return tonemapACES(c); }
	if (uTonemap == 3) { return clamp(c, 0.0, 1.0); }
	return tonemapReinhard(c);
}

vec3 grade(vec3 c) {
	c = (c - 0.5) * uContrast + 0.5;
	float y = luma(c);
	c = mix(vec3(y), c, uSat);
	return clamp(c * uTint, 0.0, 1.0);
}

vec3 fxaa(vec3 rgb) {
	vec2 px = uTexel;
	vec3 n = texture(uTex, vUV + vec2(0.0, -px.y)).rgb;
	vec3 s = texture(uTex, vUV + vec2(0.0, px.y)).rgb;
	vec3 e = texture(uTex, vUV + vec2(px.x, 0.0)).rgb;
	vec3 ww = texture(uTex, vUV + vec2(-px.x, 0.0)).rgb;
	float lM = luma(rgb);
	float lMin = min(lM, min(min(luma(n), luma(s)), min(luma(e), luma(ww))));
	float lMax = max(lM, max(max(luma(n), luma(s)), max(luma(e), luma(ww))));
	if (lMax - lMin < 0.08) {
		return rgb;
	}
	return (n + s + e + ww + rgb) * 0.2;
}

float rainStreaks(vec2 uv) {
	float n = 0.0;
	for (int i = 0; i < 3; i++) {
		float fi = float(i);
		vec2 cell = vec2(16.0 + fi * 6.0, 7.0 + fi * 2.5);
		vec2 p = uv * cell;
		p.y += uTime * (2.6 + fi * 0.7);
		vec2 id = floor(p);
		vec2 f = fract(p);
		float h = fract(sin(dot(id + fi * 13.1, vec2(127.1, 311.7))) * 43758.5453);
		float drop = smoothstep(0.07, 0.0, abs(f.x - mix(0.2, 0.8, h)))
			* smoothstep(0.62, 0.0, f.y)
			* step(0.55, h);
		n += drop * (0.38 - fi * 0.08);
	}
	return n * uRain * clamp(uWet * 1.15, 0.18, 1.0);
}

float linDepth(vec2 uv) {
	float z = texture(uDepth, uv).r * 2.0 - 1.0;
	float n = uNearFar.x;
	float f = uNearFar.y;
	return 2.0 * n * f / (f + n - z * (f - n));
}

// outlineEdge is 1 where the depth around the pixel jumps by more than
// uOutlineDepth of its own distance, so silhouettes get a line and flat
// floors do not.
float outlineEdge(vec2 uv) {
	vec2 px = uTexel * uOutlineWidth;
	float d = linDepth(uv);
	float dn = linDepth(uv + vec2(0.0, px.y));
	float ds = linDepth(uv - vec2(0.0, px.y));
	float de = linDepth(uv + vec2(px.x, 0.0));
	float dw = linDepth(uv - vec2(px.x, 0.0));
	float nearest = min(d, min(min(dn, ds), min(de, dw)));
	float farthest = max(d, max(max(dn, ds), max(de, dw)));
	float jump = (farthest - nearest) / max(nearest, 1e-3);
	// Lines fade with distance so the horizon does not turn into ink.
	float fade = 1.0 - smoothstep(uNearFar.y * 0.15, uNearFar.y * 0.4, nearest);
	return smoothstep(uOutlineDepth, uOutlineDepth * 2.0, jump) * fade;
}

vec3 applyLUT(vec3 c) {
	// Sample texel centres: scale into [half texel, 1 - half texel].
	vec3 uvw = clamp(c, 0.0, 1.0) * ((uLUTSize - 1.0) / uLUTSize) + 0.5 / uLUTSize;
	return mix(c, texture(uLUT, uvw).rgb, uLUTAmount);
}

// bayer4 is the 4x4 ordered-dither threshold for a source pixel, 0..1.
float bayer4(vec2 p) {
	ivec2 i = ivec2(mod(p, 4.0));
	int m[16] = int[16](0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5);
	return (float(m[i.y * 4 + i.x]) + 0.5) / 16.0;
}

// retro bands colour into levels or snaps it to the palette, with an
// ordered dither in source pixels so it stays crisp after the upscale.
vec3 retro(vec3 c, vec2 uv) {
	float b = bayer4(floor(uv * uSrcSize)) - 0.5;
	c = clamp(c, 0.0, 1.0);
	if (uRetroPalN > 0) {
		c = clamp(c + b * uRetroDither * 0.25, 0.0, 1.0);
		vec3 best = c;
		float bestD = 1e9;
		for (int i = 0; i < 256; i++) {
			if (i >= uRetroPalN) { break; }
			vec3 p = texelFetch(uRetroPal, ivec2(i, 0), 0).rgb;
			vec3 d = (c - p) * vec3(0.30, 0.59, 0.11);
			float dd = dot(d, d) + dot(c - p, c - p) * 0.25;
			if (dd < bestD) { bestD = dd; best = p; }
		}
		return best;
	}
	float l = uRetroLevels - 1.0;
	return clamp(floor(c * l + 0.5 + b * uRetroDither) / l, 0.0, 1.0);
}

float hash12(vec2 p) {
	vec3 p3 = fract(vec3(p.xyx) * 0.1031);
	p3 += dot(p3, p3.yzx + 33.33);
	return fract((p3.x + p3.y) * p3.z);
}

void main() {
	vec2 uv = vUV;
	vec3 rgb;
	if (uCA > 0.0001) {
		vec2 off = (uv - 0.5) * uCA * uTexel * 40.0;
		rgb.r = texture(uTex, uv + off).r;
		rgb.g = texture(uTex, uv).g;
		rgb.b = texture(uTex, uv - off).b;
	} else {
		rgb = texture(uTex, uv).rgb;
	}
	// A NaN from a scene shader would show as a black pixel; drop it.
	if (any(isnan(rgb)) || any(isinf(rgb))) { rgb = vec3(0.0); }
	if (uRainOnly != 0) {
		rgb += vec3(0.78, 0.84, 0.94) * rainStreaks(vUV);
		frag = vec4(rgb, 1.0);
		return;
	}
	if (uFXAA != 0) {
		rgb = fxaa(rgb);
	}
	if (uSSAO != 0) {
		float vis = texture(uAOTex, vUV).r;
		float fill = 1.0 - smoothstep(0.22, 1.05, luma(rgb));
		rgb *= mix(1.0, vis, fill);
	}
	vec3 bloom = vec3(0.0);
	if (uBloom > 0.001) {
		if (uBloomChain != 0) {
			bloom = texture(uBloom0, vUV).rgb * 0.50;
			bloom += texture(uBloom1, vUV).rgb * 0.26;
			bloom += texture(uBloom2, vUV).rgb * 0.14;
			bloom += texture(uBloom3, vUV).rgb * 0.07;
			bloom += texture(uBloom4, vUV).rgb * 0.04;
			bloom *= uBloom * 1.8;
		} else {
			vec2 px = uTexel * 2.2;
			vec3 acc = vec3(0.0);
			acc += texture(uTex, vUV + vec2(-px.x, -px.y)).rgb;
			acc += texture(uTex, vUV + vec2(0.0, -px.y)).rgb;
			acc += texture(uTex, vUV + vec2(px.x, -px.y)).rgb;
			acc += texture(uTex, vUV + vec2(-px.x, 0.0)).rgb;
			acc += texture(uTex, vUV + vec2(px.x, 0.0)).rgb;
			acc += texture(uTex, vUV + vec2(-px.x, px.y)).rgb;
			acc += texture(uTex, vUV + vec2(0.0, px.y)).rgb;
			acc += texture(uTex, vUV + vec2(px.x, px.y)).rgb;
			acc *= 0.125;
			float b = max(luma(acc) - 0.72, 0.0);
			bloom = acc * b * uBloom * 2.4;
		}
	}
	rgb = grade(tonemap(rgb + bloom));
	if (uLUTOn != 0) {
		rgb = applyLUT(rgb);
	}
	if (uOutlineOn != 0) {
		rgb = mix(rgb, uOutlineColor, outlineEdge(vUV));
	}
	if (uSharpen > 0.001) {
		vec2 px = uTexel;
		vec3 blur = texture(uTex, vUV + vec2(px.x, 0.0)).rgb;
		blur += texture(uTex, vUV - vec2(px.x, 0.0)).rgb;
		blur += texture(uTex, vUV + vec2(0.0, px.y)).rgb;
		blur += texture(uTex, vUV - vec2(0.0, px.y)).rgb;
		blur *= 0.25;
		rgb += (rgb - blur) * uSharpen;
	}
	if (uVignette > 0.001) {
		vec2 q = vUV - 0.5;
		float v = dot(q, q);
		rgb *= clamp(1.0 - v * uVignette * 2.2, 0.0, 1.0);
	}
	if (uGrain > 0.001) {
		float n = hash12(vUV * vec2(1920.0, 1080.0) + uTime * 13.0) - 0.5;
		rgb += n * uGrain;
	}
	if (uRetroPalN > 0 || uRetroLevels > 1.0) {
		rgb = retro(rgb, vUV);
	}
	rgb += vec3(0.78, 0.84, 0.94) * rainStreaks(vUV);
	frag = vec4(rgb, 1.0);
}
`

// Center-weighted average of the scene, written as a 1x1 luminance.
const mbLumFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uTex;

float luma(vec3 c) { return dot(c, vec3(0.2126, 0.7152, 0.0722)); }
// finite drops NaN and Inf: in a half-float target one bad pixel would
// otherwise spread through the bloom chain into a block.
vec3 finite(vec3 c) { return (any(isnan(c)) || any(isinf(c))) ? vec3(0.0) : c; }

void main() {
	float acc = 0.0;
	float wsum = 0.0;
	for (int y = 0; y < 16; y++) {
		for (int x = 0; x < 16; x++) {
			vec2 uv = (vec2(float(x), float(y)) + 0.5) / 16.0;
			vec2 p = uv - 0.5;
			float w = exp(-dot(p, p) * 6.0);
			acc += luma(finite(texture(uTex, uv).rgb)) * w;
			wsum += w;
		}
	}
	frag = vec4(acc / max(wsum, 0.0001), 0.0, 0.0, 1.0);
}
`

// Half-resolution bloom downsample. The first pass keeps pixels above uThreshold.
const mbBloomDownFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uSrc;
uniform vec2 uTexel;
uniform float uThreshold;

float luma(vec3 c) { return dot(c, vec3(0.2126, 0.7152, 0.0722)); }
// finite drops NaN and Inf: in a half-float target one bad pixel would
// otherwise spread through the bloom chain into a block.
vec3 finite(vec3 c) { return (any(isnan(c)) || any(isinf(c))) ? vec3(0.0) : c; }

void main() {
	vec3 c = finite(texture(uSrc, vUV + uTexel * vec2(-1.0, -1.0)).rgb);
	c += finite(texture(uSrc, vUV + uTexel * vec2(1.0, -1.0)).rgb);
	c += finite(texture(uSrc, vUV + uTexel * vec2(-1.0, 1.0)).rgb);
	c += finite(texture(uSrc, vUV + uTexel * vec2(1.0, 1.0)).rgb);
	c *= 0.25;
	if (uThreshold > 0.0) {
		float l = luma(c);
		c *= clamp(l - uThreshold, 0.0, 1.0) / max(l, 0.0001);
	}
	frag = vec4(c, 1.0);
}
`

// Half-resolution GTAO. Four horizon slices, analytic arc, visibility in R.
// A flat wall facing the camera stays near 1. Creases and contact drop below 1.
const mbGTAOFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uDepth;
uniform vec2 uTexel;
uniform float uRadius;
uniform float uBias;
uniform float uIntensity;
uniform float uNear;
uniform float uFar;
uniform float uTanHalf;
uniform float uAspect;

const float PI = 3.14159265;
const int SLICES = 4;
const int STEPS = 5;

float viewZ(float d) {
	float z = d * 2.0 - 1.0;
	return (2.0 * uNear * uFar) / (uFar + uNear - z * (uFar - uNear));
}

vec3 viewAt(vec2 uv) {
	float z = max(viewZ(texture(uDepth, uv).r), uNear);
	vec2 ndc = uv * 2.0 - 1.0;
	return vec3(ndc.x * uAspect * uTanHalf * z, ndc.y * uTanHalf * z, z);
}

float horizonCos(vec3 delta, vec3 V, float radius, float bias) {
	float lenSq = dot(delta, delta);
	float biasSq = bias * bias;
	float radSq = radius * radius;
	if (lenSq < biasSq || lenSq > radSq) {
		return -1.0;
	}
	float h = dot(delta, V) * inversesqrt(lenSq);
	float fall = clamp(1.0 - lenSq / radSq, 0.0, 1.0);
	return mix(-1.0, clamp(h, -1.0, 1.0), fall);
}

float integrateArc(float h1, float h2, float n, float cosN) {
	float a = h1 * 2.0;
	float b = h2 * 2.0;
	float s = sin(n);
	return 0.25 * ((-cos(a - n) + cosN + a * s) + (-cos(b - n) + cosN + b * s));
}

void main() {
	float raw = texture(uDepth, vUV).r;
	if (raw >= 0.9995) {
		frag = vec4(1.0);
		return;
	}
	vec3 pos = viewAt(vUV);
	vec3 dx = viewAt(vUV + vec2(uTexel.x, 0.0)) - pos;
	vec3 dy = viewAt(vUV + vec2(0.0, uTexel.y)) - pos;
	vec3 N = normalize(cross(dx, dy));
	vec3 V = normalize(-pos);
	if (dot(N, V) < 0.0) {
		N = -N;
	}

	float noise = fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453);
	float acc = 0.0;
	float wsum = 0.0;
	float z = max(pos.z, uNear);
	vec2 uvRadius = vec2(
		uRadius / max(z * uAspect * uTanHalf * 2.0, 0.0001),
		uRadius / max(z * uTanHalf * 2.0, 0.0001)
	);
	for (int slice = 0; slice < SLICES; slice++) {
		float ang = (float(slice) + noise) * PI / float(SLICES);
		vec2 dir = vec2(cos(ang), sin(ang));
		vec2 stepUV = dir * uvRadius / float(STEPS);
		float h1 = -1.0;
		float h2 = -1.0;
		for (int i = 1; i <= STEPS; i++) {
			float fi = float(i);
			h1 = max(h1, horizonCos(viewAt(vUV + stepUV * fi) - pos, V, uRadius, uBias));
			h2 = max(h2, horizonCos(viewAt(vUV - stepUV * fi) - pos, V, uRadius, uBias));
		}
		vec3 sliceN = normalize(cross(vec3(dir, 0.0), V));
		vec3 projN = N - sliceN * dot(N, sliceN);
		float projLen = length(projN);
		if (projLen < 1e-4) {
			continue;
		}
		float cosN = clamp(dot(projN / projLen, V), -1.0, 1.0);
		vec3 T = cross(V, sliceN);
		float n = -sign(dot(projN, T)) * acos(cosN);
		float a1 = -acos(clamp(h1, -1.0, 1.0));
		float a2 = acos(clamp(h2, -1.0, 1.0));
		a1 = n + max(a1 - n, -PI * 0.5);
		a2 = n + min(a2 - n, PI * 0.5);
		float arc = integrateArc(a1, a2, n, cosN);
		if (arc != arc) {
			arc = 1.0;
		}
		arc = clamp(arc, 0.0, 1.0);
		acc += arc * projLen;
		wsum += projLen;
	}
	float vis = wsum > 0.0 ? acc / wsum : 1.0;
	vis = pow(clamp(vis, 0.0, 1.0), max(uIntensity, 0.0) / 1.8);
	frag = vec4(vis, vis, vis, 1.0);
}
`

// Depth-aware blur. uDirection is (1, 0) or (0, 1) in half-resolution texels.
const mbGTAOBlurFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uAO;
uniform sampler2D uDepth;
uniform vec2 uTexel;
uniform vec2 uDirection;
uniform float uNear;
uniform float uFar;

float viewZ(float d) {
	float z = d * 2.0 - 1.0;
	return (2.0 * uNear * uFar) / (uFar + uNear - z * (uFar - uNear));
}

void main() {
	float z0 = viewZ(texture(uDepth, vUV).r);
	float acc = texture(uAO, vUV).r;
	float wsum = 1.0;
	for (int i = 1; i <= 4; i++) {
		vec2 o = uDirection * uTexel * float(i);
		float spatial = exp(-float(i * i) * 0.18);
		float z1 = viewZ(texture(uDepth, vUV + o).r);
		float z2 = viewZ(texture(uDepth, vUV - o).r);
		float w1 = spatial * exp(-abs(z1 - z0) / max(z0 * 0.04, 0.05));
		float w2 = spatial * exp(-abs(z2 - z0) / max(z0 * 0.04, 0.05));
		acc += texture(uAO, vUV + o).r * w1;
		acc += texture(uAO, vUV - o).r * w2;
		wsum += w1 + w2;
	}
	float v = acc / wsum;
	frag = vec4(v, v, v, 1.0);
}
`
