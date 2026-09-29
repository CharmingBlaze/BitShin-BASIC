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

// One-pass tonemap + cheap bloom (9-tap bright) + optional FXAA + grade + SSAO.
const mbpostFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uTex;
uniform sampler2D uDepth;
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

uniform int uSSAO;
uniform float uSSAORadius;
uniform float uSSAOIntensity;
uniform float uSSAOBias;
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

float linearizeDepth(float d) {
	float zNear = 0.1;
	float zFar = 150.0;
	return (2.0 * zNear * zFar) / (zFar + zNear - (d * 2.0 - 1.0) * (zFar - zNear));
}

const vec2 ssaoSamples[12] = vec2[](
	vec2( 0.537,  0.843),
	vec2(-0.843,  0.537),
	vec2(-0.537, -0.843),
	vec2( 0.843, -0.537),
	vec2( 0.234,  0.412),
	vec2(-0.412,  0.234),
	vec2(-0.234, -0.412),
	vec2( 0.412, -0.234),
	vec2( 0.112,  0.920),
	vec2(-0.920,  0.112),
	vec2(-0.112, -0.920),
	vec2( 0.920, -0.112)
);

float computeSSAO(vec2 uv) {
	float centerRaw = texture(uDepth, uv).r;
	if (centerRaw >= 0.9999) {
		return 0.0;
	}
	float centerZ = linearizeDepth(centerRaw);
	float rad = uSSAORadius / max(centerZ, 0.35);
	rad = clamp(rad, 1.5, 36.0);

	vec2 pCoord = uv / uTexel;
	float angle = fract(sin(dot(pCoord, vec2(12.9898, 78.233))) * 43758.5453) * 6.2831853;
	float ca = cos(angle);
	float sa = sin(angle);
	mat2 rot = mat2(ca, -sa, sa, ca);

	float occlusion = 0.0;
	for (int i = 0; i < 12; i++) {
		vec2 offset = (rot * ssaoSamples[i]) * rad * uTexel;
		float sampleRaw = texture(uDepth, uv + offset).r;
		float sampleZ = linearizeDepth(sampleRaw);
		float diff = centerZ - sampleZ;
		if (diff > uSSAOBias && diff < uSSAORadius * 3.5) {
			float r = (uSSAORadius * 3.5 - diff) / (uSSAORadius * 3.5);
			occlusion += r * r;
		}
	}
	occlusion /= 12.0;
	return clamp(occlusion * uSSAOIntensity, 0.0, 0.85);
}

void main() {
	vec3 rgb = texture(uTex, vUV).rgb;
	if (uRainOnly != 0) {
		rgb += vec3(0.78, 0.84, 0.94) * rainStreaks(vUV);
		frag = vec4(rgb, 1.0);
		return;
	}
	if (uFXAA != 0) {
		rgb = fxaa(rgb);
	}
	if (uSSAO != 0) {
		float ao = computeSSAO(vUV);
		rgb *= (1.0 - ao);
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
	rgb += vec3(0.78, 0.84, 0.94) * rainStreaks(vUV);
	frag = vec4(rgb, 1.0);
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

void main() {
	vec3 c = texture(uSrc, vUV + uTexel * vec2(-1.0, -1.0)).rgb;
	c += texture(uSrc, vUV + uTexel * vec2(1.0, -1.0)).rgb;
	c += texture(uSrc, vUV + uTexel * vec2(-1.0, 1.0)).rgb;
	c += texture(uSrc, vUV + uTexel * vec2(1.0, 1.0)).rgb;
	c *= 0.25;
	if (uThreshold > 0.0) {
		float l = luma(c);
		c *= clamp(l - uThreshold, 0.0, 1.0) / max(l, 0.0001);
	}
	frag = vec4(c, 1.0);
}
`
