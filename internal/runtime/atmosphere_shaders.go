package runtime

import "github.com/g3n/engine/renderer"

// mbatmo: Bruneton-style sky dome. Transmittance from a 2D LUT (Go bake).
// 12-step single scatter. GLSL 330 — no 3D textures, no compute.

const mbatmoVertex = `#include <attributes>
uniform mat4 MVP;
uniform mat4 ModelMatrix;
out vec3 WorldPos;
void main() {
    vec4 world = ModelMatrix * vec4(VertexPosition, 1.0);
    WorldPos = world.xyz;
    gl_Position = MVP * vec4(VertexPosition, 1.0);
}
`

const mbatmoFragment = `precision highp float;
in vec3 WorldPos;
uniform sampler2D AtmoT;
uniform int AtmoHasT;
uniform vec3 AtmoCam;
uniform vec3 AtmoSun;
uniform float AtmoRayleigh;
uniform float AtmoMie;
uniform float AtmoTurbidity;
uniform float AtmoExposure;
uniform vec3 AtmoGround;
uniform vec3 AtmoTint;
uniform float AtmoFlash;
out vec4 FragColor;

const vec3 betaR = vec3(5.8e-3, 13.5e-3, 33.1e-3);
const float betaM = 0.021;
const float Hr = 8.0;
const float Hm = 1.2;
const float Re = 6360.0;
const float Ra = 6440.0;

vec2 lutUV(float mu, float h) {
    float u = clamp((mu + 0.15) / 2.15, 0.0, 1.0);
    float v = clamp(h / 80.0, 0.0, 1.0);
    return vec2(u, v);
}

vec4 trans(float mu, float h) {
    if (AtmoHasT != 0) {
        return texture(AtmoT, lutUV(mu, h));
    }
    float od = exp(-h / Hr) * 8.0 / max(mu + 0.15, 0.08);
    return vec4(exp(-od * betaR), exp(-od * betaM));
}

float phaseR(float mu) { return 0.75 * (1.0 + mu * mu); }
float phaseM(float mu) {
    float g = 0.76;
    float g2 = g * g;
    return (1.0 - g2) / pow(max(1e-4, 1.0 + g2 - 2.0 * g * mu), 1.5);
}

void main() {
    vec3 rd = normalize(WorldPos - AtmoCam);
    vec3 sun = normalize(AtmoSun);
    if (length(AtmoSun) < 1e-4) { sun = vec3(-0.35, 0.62, 0.70); }
    float muV = rd.y;
    float muS = sun.y;
    float mu = dot(rd, sun);

    if (rd.y < -0.02) {
        float gnd = clamp(-rd.y, 0.0, 1.0);
        vec3 col = mix(AtmoGround, AtmoGround * 0.45, gnd);
        col *= AtmoTint;
        col += vec3(AtmoFlash);
        FragColor = vec4(col, 1.0);
        return;
    }

    vec3 sumR = vec3(0.0);
    vec3 sumM = vec3(0.0);
    float t0 = 0.0;
    float t1 = 80.0 / max(rd.y, 0.04);
    t1 = min(t1, 220.0);
    float ds = t1 / 12.0;
    t0 += fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453) * ds;
    float odR = 0.0;
    float odM = 0.0;
    for (int i = 0; i < 12; i++) {
        float t = t0 + (float(i) + 0.5) * ds;
        float h = max(t * rd.y * 0.35, 0.0);
        float dR = exp(-h / Hr);
        float dM = exp(-h / Hm) * AtmoTurbidity;
        odR += dR * ds;
        odM += dM * ds;
        vec4 ts = trans(muS, h);
        vec3 tCam = exp(-odR * betaR * AtmoRayleigh - odM * vec3(betaM) * AtmoMie);
        vec3 tSun = ts.rgb * ts.a;
        sumR += dR * tCam * tSun * ds;
        sumM += dM * tCam * tSun * ds;
    }
    vec3 col = sumR * betaR * AtmoRayleigh * phaseR(mu) + sumM * betaM * AtmoMie * phaseM(mu);
    col *= 22.0;
    float disk = pow(max(mu, 0.0), 220.0);
    float glow = pow(max(mu, 0.0), 12.0);
    col += vec3(1.0, 0.92, 0.72) * disk * 2.2;
    col += vec3(1.0, 0.78, 0.45) * glow * 0.55;
    float horizon = exp(-max(rd.y, 0.0) * 6.0);
    col = mix(col, col * vec3(1.15, 0.95, 0.75), horizon * 0.35);
    col *= AtmoTint;
    col += vec3(AtmoFlash);
    col = vec3(1.0) - exp(-col * max(AtmoExposure, 0.2));
    FragColor = vec4(col, 1.0);
}
`

func registerAtmosphereShaders(r *renderer.Renderer) {
	r.AddShader("mbatmo_vertex", mbatmoVertex)
	r.AddShader("mbatmo_fragment", mbatmoFragment)
	r.AddProgram("mbatmo", "mbatmo_vertex", "mbatmo_fragment")
}
