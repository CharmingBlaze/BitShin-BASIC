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
const mbpostFragment = `#version 330 core
in vec2 vUV;
out vec4 frag;
uniform sampler2D uTex;
uniform float uExposure;
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

float luma(vec3 c) { return dot(c, vec3(0.299, 0.587, 0.114)); }

vec3 tonemap(vec3 c) {
	c *= max(uExposure, 0.05);
	return c / (c + vec3(1.0));
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
	vec3 bloom = vec3(0.0);
	if (uBloom > 0.001) {
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
	rgb = grade(tonemap(rgb + bloom));
	rgb += vec3(0.78, 0.84, 0.94) * rainStreaks(vUV);
	frag = vec4(rgb, 1.0);
}
`
