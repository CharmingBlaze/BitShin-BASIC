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
        float f = k * dot(d, VertexPosition.xz) - WaveLen[i].y * t;
        float cf = cos(f);
        float sf = sin(f);
        p.x += steep * amp * d.x * cf;
        p.z += steep * amp * d.y * cf;
        p.y += amp * sf;
        n.x -= d.x * k * amp * cf;
        n.z -= d.y * k * amp * cf;
        n.y -= steep * k * amp * sf;
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
uniform mat4 LightVP[4];
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
` + shadeLightGLSL + `
void main() {
    vec3 analytic = normalize(WorldNormal);
    vec3 dx_pos = dFdx(WorldPos);
    vec3 dy_pos = dFdy(WorldPos);
    vec3 faceN = analytic;
    if (dot(dx_pos, dx_pos) > 1e-6 && dot(dy_pos, dy_pos) > 1e-6) {
        faceN = normalize(cross(dx_pos, dy_pos));
    }
    if (!gl_FrontFacing) {
        analytic = -analytic;
        faceN = -faceN;
    }
    if (any(isnan(faceN)) || dot(faceN, faceN) < 1e-8) {
        faceN = analytic;
    }
    vec3 waveNormal = analytic;
    if (any(isnan(waveNormal)) || dot(waveNormal, waveNormal) < 1e-6) {
        waveNormal = faceN;
    }
    if (!gl_FrontFacing && CamWorldPos.y > WaterLevel + 0.35) {
        discard;
    }

    vec2 ndc = (ClipSpace.xy / max(ClipSpace.w, 0.0001)) * 0.5 + 0.5;
    vec2 reflectUV = vec2(ndc.x, 1.0 - ndc.y);
    vec2 refractUV = vec2(ndc.x, ndc.y);
    vec2 tex = FragTexcoord;
    float meters = 12.0;
    if (WaterHasDepth != 0) {
        float n = max(CamNear, 0.05);
        float f = max(CamFar, n + 1.0);
        float zWater = (2.0 * n * f) / max(f + n - (2.0 * gl_FragCoord.z - 1.0) * (f - n), 0.001);
        float sceneD = texture(WaterRefractDepth, clamp(ndc, 0.001, 0.999)).r;
        float zScene = (2.0 * n * f) / max(f + n - (2.0 * sceneD - 1.0) * (f - n), 0.001);
        meters = max(zScene - zWater, 0.0);
    }
    float contact = 1.0 - smoothstep(0.15, 1.2, meters);
    vec2 distort = vec2(0.0);
    if (WaterHasDuDv != 0) {
        vec2 d1 = texture(WaterDuDv, vec2(tex.x + WaterMove, tex.y)).rg * 0.1;
        vec2 distorted = tex + vec2(d1.x, d1.y + WaterMove);
        distort = (texture(WaterDuDv, distorted).rg * 2.0 - 1.0) * WaterWaveStrength;
        distort *= clamp(meters / 20.0, 0.0, 1.0);
        distort *= mix(0.18, 1.0, exp(-length(CamWorldPos - WorldPos) * 0.028));
    }
    reflectUV = clamp(reflectUV + distort, 0.001, 0.999);
    refractUV = clamp(refractUV + distort, 0.001, 0.999);
    if (WaterHasNormal != 0) {
        vec2 nUV1 = tex * 6.0 + vec2(WaterMove * 0.7, WaterMove * 0.22);
        vec2 nUV2 = tex * 1.5 + vec2(-WaterMove * 0.4, WaterMove * 0.18);
        vec3 n1 = texture(WaterNormal, nUV1).rgb * 2.0 - 1.0;
        vec3 n2 = texture(WaterNormal, nUV2).rgb * 2.0 - 1.0;
        vec3 detail = normalize(n1 + n2);
        vec3 rippled = vec3(waveNormal.x + detail.x * 0.95, waveNormal.y, waveNormal.z + detail.z * 0.95);
        waveNormal = normalize(rippled + detail * 0.15);
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

    float camDist = length(CamWorldPos - WorldPos);
    float nearRipple = exp(-camDist * 0.02);
    reflectUV = clamp(reflectUV + waveNormal.xz * (0.05 * nearRipple), 0.001, 0.999);
    refractUV = clamp(refractUV + waveNormal.xz * (0.02 * nearRipple), 0.001, 0.999);

    vec3 viewDir = normalize(CamWorldPos - WorldPos);
    float NdotV = max(dot(waveNormal, viewDir), 0.0);
    float fresnel = pow(clamp(1.0 - NdotV, 0.0, 1.0), 5.0);
    float schlick = 0.02 + 0.98 * fresnel;
    float waveHeight = WorldPos.y - WaterLevel;
    float depthMix = smoothstep(0.4, 6.0, meters);
    vec3 deepWaterColor = WaterColor;
    if (dot(deepWaterColor, deepWaterColor) < 0.004) {
        deepWaterColor = vec3(0.02, 0.16, 0.28);
    }
    vec3 shallowWaterColor = deepWaterColor * vec3(1.2, 1.45, 1.08);
    vec3 refrColor = mix(shallowWaterColor, deepWaterColor, depthMix);
    if (WaterRefractOn != 0) {
        vec3 grabbedR = texture(WaterRefract, refractUV).rgb;
        float bottom = smoothstep(0.25, 3.0, meters) * (1.0 - smoothstep(28.0, 70.0, meters));
        refrColor = mix(refrColor, grabbedR, bottom * 0.8);
    }
    vec3 reflectCol = mix(deepWaterColor, vec3(0.65, 0.78, 0.9), 0.35);
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
        reflectCol = grabbed;
    }
    float refl = WaterReflectivity;
    if (refl < 0.15) { refl = 0.45; }
    float shoreFade = smoothstep(0.4, 2.8, meters);
    float facing = mix(schlick * 0.22, schlick, shoreFade) * clamp(refl, 0.35, 1.0);
    vec3 base = mix(refrColor, reflectCol, facing);

    vec3 L = WaterSunDir;
    if (dot(L, L) < 1e-6) {
        L = vec3(0.4, 0.8, 0.3);
    }
    L = normalize(L);
    vec3 sunCol = WaterSunColor;
#if DIR_LIGHTS>0
    sunCol = DirLightColor(0);
#endif
    vec3 H = normalize(L + viewDir);
    float NdotH = max(dot(waveNormal, H), 0.0);
    float NdotL = max(dot(waveNormal, L), 0.0);
    float shine = WaterShine;
    if (shine < 8.0) { shine = 48.0; }
    float tight = pow(NdotH, 512.0);
    float broad = pow(NdotH, shine);
    vec3 sunGlint = sunCol * (broad * 3.4 + tight * 6.0) * NdotL;
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
    vec3 waterWithFoam = mix(base, foamColor, clamp(foamMask * foamNoise, 0.0, 0.4));

    float sh = waterShadow(WorldPos, waveNormal, L);
    vec3 lit = waterWithFoam;
    vec3 spec = sunGlint * sh * (0.55 + 1.6 * schlick);
    vec3 local = vec3(0.0);
    vec3 waterN = normalize(Normal);
    vec3 waterV = normalize(-Position.xyz);
#if POINT_LIGHTS>0
    for (int i = 0; i < POINT_LIGHTS; ++i) {
        vec3 lPos = PointLightPosition(i) - Position.xyz;
        float dist = length(lPos);
        if (dist > 0.001) {
            vec3 ld = lPos / dist;
            float att = shadeAttenuation(dist, PointLightLinearDecay(i), PointLightQuadraticDecay(i), PointLightConstant(i));
            if (att > 0.001) {
                float ndl = max(dot(waterN, ld), 0.0);
                vec3 h2 = normalize(ld + waterV);
                float ndh = max(dot(waterN, h2), 0.0);
                local += PointLightColor(i) * att * (ndl * waterWithFoam * 0.65 + pow(ndh, 180.0) * ndl * 1.6);
            }
        }
    }
#endif
#if SPOT_LIGHTS>0
    for (int i = 0; i < SPOT_LIGHTS; ++i) {
        vec3 lPos = SpotLightPosition(i) - Position.xyz;
        float dist = length(lPos);
        if (dist > 0.001) {
            vec3 ld = lPos / dist;
            float att = shadeAttenuation(dist, SpotLightLinearDecay(i), SpotLightQuadraticDecay(i), SpotLightConstant(i));
            att *= shadeSpot(ld, SpotLightDirection(i), SpotLightCutoffAngle(i), SpotLightAngularDecay(i), SpotLightConstant(i));
            att *= shadeCookie(i, -lPos, SpotLightDirection(i), SpotLightCutoffAngle(i));
            if (att > 0.001) {
                float ndl = max(dot(waterN, ld), 0.0);
                vec3 h2 = normalize(ld + waterV);
                float ndh = max(dot(waterN, h2), 0.0);
                local += SpotLightColor(i) * att * (ndl * waterWithFoam * 0.65 + pow(ndh, 180.0) * ndl * 1.6);
            }
        }
    }
#endif
    vec3 finalRGB = lit + spec + local;
    float alpha = 1.0;
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
