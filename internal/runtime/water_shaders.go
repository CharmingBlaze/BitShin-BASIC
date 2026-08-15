package runtime

import "github.com/g3n/engine/renderer"

// mbwater: Gerstner vertex displacement + screen-space derivative normals,
// Fresnel body color, Blinn-Phong sun, planar reflection. GLSL 330 core
// (G3N shaman prepends "#version 330 core").

const mbwaterVertex = `#include <attributes>
uniform mat4 ModelViewMatrix;
uniform mat3 NormalMatrix;
uniform mat4 MVP;
uniform mat4 ModelMatrix;
uniform float Time;
uniform float WaveSpeed;
uniform float WaveTime;
uniform int WaveCount;
uniform float WaveStorm;
uniform vec4 WaveDir[4];
uniform vec4 WaveLen[4];
uniform vec4 WaterWind;
uniform sampler2D WaterWake;
uniform int WaterHasWake;
uniform float WakeAmp;
uniform float WakeSpan;
out vec4 Position;
out vec3 Normal;
out vec2 FragTexcoord;
out vec3 WorldPos;
out vec3 WorldNormal;
out vec4 ClipSpace;
void main() {
    vec3 p = VertexPosition;
    vec3 n = vec3(0.0, 1.0, 0.0);
    int nw = WaveCount;
    if (nw < 0) { nw = 0; }
    if (nw > 4) { nw = 4; }
    float t = Time * WaveSpeed;
    if (t == 0.0) { t = WaveTime; }
    for (int i = 0; i < 4; ++i) {
        if (i >= nw) { break; }
        float lambda = max(WaveLen[i].x, 0.2);
        float k = 6.2831853 / lambda;
        float amp = WaveDir[i].w * WaveStorm;
        float steep = clamp(WaveDir[i].z, 0.0, 0.5);
        vec2 d = WaveDir[i].xy;
        float dl = length(d);
        if (dl < 1e-4) { continue; }
        d /= dl;
        vec2 wind = WaterWind.xy;
        float wlen = length(wind);
        float wstr = WaterWind.z;
        if (wlen > 1e-4 && wstr > 0.0) {
            wind /= wlen;
            float mk = clamp(wstr * 0.55, 0.0, 0.85);
            d = normalize(mix(d, wind, mk));
            amp *= 1.0 + wstr * 0.35 * max(dot(d, wind), 0.0);
        }
        float f = k * dot(d, p.xz) - WaveLen[i].y * t;
        float cf = cos(f);
        float sf = sin(f);
        p.x += steep * amp * d.x * cf;
        p.z += steep * amp * d.y * cf;
        p.y += amp * sf;
        n.x -= d.x * k * amp * cf;
        n.z -= d.y * k * amp * cf;
        n.y += 1.0 - steep * k * amp * sf;
    }
    if (WaterHasWake != 0) {
        vec2 wuv = VertexPosition.xz / max(WakeSpan, 8.0) + 0.5;
        if (wuv.x > 0.001 && wuv.x < 0.999 && wuv.y > 0.001 && wuv.y < 0.999) {
            vec4 wk = texture(WaterWake, wuv);
            p.y += (wk.r - 0.5) * WakeAmp * 2.0;
            n.x -= (wk.g - 0.5) * 1.8;
            n.z -= (wk.b - 0.5) * 1.8;
        }
    }
    n = normalize(n);
    vec4 world = ModelMatrix * vec4(p, 1.0);
    WorldPos = world.xyz;
    WorldNormal = normalize(mat3(ModelMatrix) * n);
    Position = ModelViewMatrix * vec4(p, 1.0);
    Normal = normalize(NormalMatrix * n);
    FragTexcoord = VertexTexcoord;
    ClipSpace = MVP * vec4(p, 1.0);
    gl_Position = ClipSpace;
}
`

const mbwaterFragment = `precision highp float;
out vec4 FragColor;
in vec4 Position;
in vec3 Normal;
in vec2 FragTexcoord;
in vec3 WorldPos;
in vec3 WorldNormal;
in vec4 ClipSpace;
#include <lights>
#include <material>
uniform vec3 WaterColor;
uniform float WaterLevel;
uniform sampler2D WaterReflect;
uniform sampler2D WaterRefract;
uniform sampler2D WaterDuDv;
uniform sampler2D WaterNormal;
uniform int WaterReflectOn;
uniform int WaterRefractOn;
uniform int WaterHasDuDv;
uniform int WaterHasNormal;
uniform float WaterMove;
uniform float WaterWaveStrength;
uniform float WaterShine;
uniform float WaterReflectivity;
uniform vec3 WaterSunDir;
uniform vec3 WaterSunColor;
uniform vec3 CamWorldPos;
uniform sampler2D WaterRefractDepth;
uniform int WaterHasDepth;
uniform float CamNear;
uniform float CamFar;
uniform int FogMode;
uniform vec3 FogColor;
uniform float FogNear;
uniform float FogFar;
uniform float FogDensity;
uniform float FogHeight;
uniform float FogHeightFalloff;
uniform float Wetness;
uniform int ShadowEnabled;
uniform float ShadowBias;
uniform int AtlasCols;
uniform int AtlasRows;
uniform mat4 LightVP[3];
uniform sampler2D ShadowMap;
uniform float WaterPeak;
uniform int WaterSSROn;
uniform sampler2D WaterWake;
uniform int WaterHasWake;
uniform float WakeAmp;
uniform float WakeSpan;
float waterShadow(vec3 worldPos, vec3 fragNormal, vec3 lightDir) {
    if (ShadowEnabled == 0) {
        return 1.0;
    }
    float cosTheta = clamp(dot(fragNormal, lightDir), 0.0, 1.0);
    float dynamicBias = ShadowBias * tan(acos(cosTheta));
    dynamicBias = clamp(dynamicBias, ShadowBias, ShadowBias * 5.0);
    vec4 ls = LightVP[0] * vec4(worldPos, 1.0);
    vec3 proj = ls.xyz / max(ls.w, 0.0001);
    proj = proj * 0.5 + 0.5;
    if (proj.z > 1.0 || proj.x <= 0.0 || proj.x >= 1.0 || proj.y <= 0.0 || proj.y >= 1.0) {
        return 1.0;
    }
    int cols = AtlasCols;
    int rows = AtlasRows;
    if (cols < 1) { cols = 1; }
    if (rows < 1) { rows = 1; }
    float pad = 0.012;
    vec2 uv = vec2((clamp(proj.x, pad, 1.0 - pad)) / float(cols), (clamp(proj.y, pad, 1.0 - pad)) / float(rows));
    float d = texture(ShadowMap, uv).r;
    float lit = (proj.z - dynamicBias > d) ? 0.0 : 1.0;
    return mix(0.55, 1.0, lit);
}
void main() {
    vec3 dx_pos = dFdx(WorldPos);
    vec3 dy_pos = dFdy(WorldPos);
    vec3 waveNormal;
    if (dot(dx_pos, dx_pos) < 1e-6 || dot(dy_pos, dy_pos) < 1e-6) {
        waveNormal = normalize(WorldNormal);
    } else {
        waveNormal = normalize(cross(dx_pos, dy_pos));
    }
    if (!gl_FrontFacing) {
        waveNormal = -waveNormal;
    }
    if (any(isnan(waveNormal)) || dot(waveNormal, waveNormal) < 1e-8) {
        waveNormal = normalize(WorldNormal);
        if (!gl_FrontFacing) {
            waveNormal = -waveNormal;
        }
    }

    vec2 ndc = (ClipSpace.xy / max(ClipSpace.w, 0.0001)) * 0.5 + 0.5;
    vec2 reflectUV = vec2(ndc.x, 1.0 - ndc.y);
    vec2 refractUV = vec2(ndc.x, ndc.y);
    vec2 tex = FragTexcoord;
    float contact = 0.0;
    if (WaterHasDepth != 0) {
        float sceneD = texture(WaterRefractDepth, clamp(ndc, 0.001, 0.999)).r;
        float gap = sceneD - gl_FragCoord.z;
        contact = 1.0 - smoothstep(0.0, 0.016, max(gap, 0.0));
    }
    vec2 distort = vec2(0.0);
    if (WaterHasDuDv != 0) {
        vec2 d1 = texture(WaterDuDv, vec2(tex.x + WaterMove, tex.y)).rg * 0.1;
        vec2 distorted = tex + vec2(d1.x, d1.y + WaterMove);
        distort = (texture(WaterDuDv, distorted).rg * 2.0 - 1.0) * WaterWaveStrength;
        distort *= 1.0 - contact;
    }
    reflectUV = clamp(reflectUV + distort, 0.001, 0.999);
    refractUV = clamp(refractUV + distort, 0.001, 0.999);
    if (WaterHasNormal != 0) {
        vec2 nUV1 = tex + vec2(WaterMove, WaterMove * 0.35);
        vec2 nUV2 = tex * 1.5 - vec2(WaterMove * 0.5, WaterMove * 0.25);
        vec3 n1 = texture(WaterNormal, nUV1).rgb * 2.0 - 1.0;
        vec3 n2 = texture(WaterNormal, nUV2).rgb * 2.0 - 1.0;
        vec3 detail = normalize(n1 + n2);
        waveNormal = normalize(waveNormal + detail * 0.15);
    }
    if (WaterHasWake != 0) {
        vec2 localXZ = FragTexcoord / 0.04;
        vec2 wuv = localXZ / max(WakeSpan, 8.0) + 0.5;
        if (wuv.x > 0.001 && wuv.x < 0.999 && wuv.y > 0.001 && wuv.y < 0.999) {
            vec4 wk = texture(WaterWake, wuv);
            vec3 wn = normalize(vec3(-(wk.g - 0.5) * 2.0, 1.0, -(wk.b - 0.5) * 2.0));
            waveNormal = normalize(mix(waveNormal, wn, 0.35 * WakeAmp));
        }
    }

    vec3 viewDir = normalize(CamWorldPos - WorldPos);
    float NdotV = max(dot(waveNormal, viewDir), 0.0);
    float fresnel = pow(1.0 - NdotV, 3.0);
    float waveHeight = WorldPos.y - WaterLevel;
    float depthMix = clamp(fresnel + waveHeight * 0.3, 0.0, 1.0);
    vec3 deepWaterColor = vec3(0.01, 0.08, 0.18);
    vec3 shallowWaterColor = vec3(0.06, 0.35, 0.48);
    vec3 base = mix(deepWaterColor, shallowWaterColor, depthMix);
    if (WaterReflectOn != 0) {
        vec3 grabbed = texture(WaterReflect, reflectUV).rgb;
        vec3 planar = grabbed;
        if (WaterSSROn != 0) {
            vec3 viewR = reflect(normalize(Position.xyz), normalize(Normal));
            vec2 ss = viewR.xy;
            float sl = length(ss);
            if (sl > 1e-4) { ss /= sl; } else { ss = vec2(0.0); }
            vec2 uv = reflectUV;
            vec3 hit = planar;
            float fade = 1.0;
            for (int s = 1; s <= 8; ++s) {
                uv += ss * (0.012 * float(s));
                if (uv.x < 0.004 || uv.x > 0.996 || uv.y < 0.004 || uv.y > 0.996) {
                    fade = 0.0;
                    break;
                }
                hit = texture(WaterReflect, clamp(uv, 0.001, 0.999)).rgb;
            }
            vec2 edge = smoothstep(vec2(0.0), vec2(0.14), uv) * smoothstep(vec2(0.0), vec2(0.14), 1.0 - uv);
            float stable = fade * edge.x * edge.y * smoothstep(-0.05, 0.25, viewR.y);
            if (any(isnan(hit)) || sl < 1e-4) {
                stable = 0.0;
            }
            grabbed = mix(planar, hit, clamp(stable * 0.5, 0.0, 0.5));
        }
        base = mix(base, grabbed, fresnel * 0.45);
    }

    vec3 L = WaterSunDir;
    if (dot(L, L) < 1e-6) {
        L = vec3(0.4, 0.8, 0.3);
    }
    L = normalize(L);
    vec3 H = normalize(L + viewDir);
    float NdotH = max(dot(waveNormal, H), 0.0);
    float NdotL = max(dot(waveNormal, L), 0.0);
    vec3 sunGlint = vec3(1.0, 0.98, 0.9) * pow(NdotH, 512.0) * NdotL * 6.0;
    float maxWavePeak = WaterPeak;
    if (maxWavePeak < 0.2) {
        maxWavePeak = 1.2;
    }
    float foamStart = maxWavePeak * 0.90;
    float foamMask = smoothstep(foamStart, maxWavePeak, waveHeight);
    vec2 foamUV = FragTexcoord * 3.0 + vec2(WaterMove * 0.1);
    float foamNoise = texture(WaterNormal, foamUV).r;
    foamMask *= step(0.3, foamNoise);
    foamMask *= 1.0 - contact;
    vec3 foamColor = vec3(0.82, 0.88, 0.92);
    vec3 waterWithFoam = mix(base, foamColor, foamMask * foamNoise * 0.85);

    float sh = waterShadow(WorldPos, waveNormal, L);
    vec3 finalRGB = (waterWithFoam + sunGlint) * sh;
    float alpha = clamp(0.7 + fresnel * 0.25, 0.6, 0.95);
    FragColor = vec4(finalRGB, alpha);
    if (FogMode != 0) {
        float dist = length(Position.xyz);
        float f = 1.0;
        if (FogMode == 1) {
            f = clamp((FogFar - dist) / max(FogFar - FogNear, 0.0001), 0.0, 1.0);
        } else if (FogMode == 2) {
            f = exp(-FogDensity * dist);
        } else {
            float d = FogDensity * dist;
            f = exp(-d * d);
        }
        if (FogHeightFalloff > 0.0001) {
            float hFac = exp(-max(WorldPos.y - FogHeight, 0.0) * FogHeightFalloff);
            f = 1.0 - (1.0 - f) * clamp(hFac, 0.0, 1.0);
        }
        FragColor.rgb = mix(FogColor, FragColor.rgb, clamp(f, 0.0, 1.0));
    }
}
`

func registerWaterShaders(r *renderer.Renderer) {
	r.AddShader("mbwater_vertex", mbwaterVertex)
	r.AddShader("mbwater_fragment", mbwaterFragment)
	r.AddProgram("mbwater", "mbwater_vertex", "mbwater_fragment")
}
