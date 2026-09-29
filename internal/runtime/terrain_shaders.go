package runtime

import "github.com/g3n/engine/renderer"

// mbterrain: Terrain-OpenGL splat (sand/grass/rock/snow by height + slope)
// on GLSL 330 / OpenGL 3.3. No tessellation control/eval, no compute.
// Vertex is mbshadow_vertex (WorldPos + CSM). Heights are already in the mesh.

const mbterrainFragmentHead = `precision highp float;
in vec4 Position;
in vec3 Normal;
in vec2 FragTexcoord;
in vec3 WorldPos;
in vec3 WorldNormal;
in vec4 LightSpacePos[4];
#include <lights>
#include <material>
#include <phong_model>
uniform sampler2D ShadowMap;
uniform int ShadowEnabled;
uniform int ShadowCascades;
uniform int ShadowFilter;
uniform int ShadowPCF;
uniform float ShadowBias;
uniform float ShadowLightSize;
uniform int ShadowContact;
uniform int ShadowSSS;
uniform int AtlasCols;
uniform int AtlasRows;
uniform int ShadowPoints;
uniform vec3 PointPos[2];
uniform float PointRange[2];
uniform int PointTile[2];
uniform mat4 PointVP[12];
uniform int ShadowSpots;
uniform vec3 SpotPos[2];
uniform vec3 SpotDir[2];
uniform float SpotRange[2];
uniform float SpotCos[2];
uniform int SpotTile[2];
uniform mat4 SpotVP[2];
uniform int FogMode;
uniform vec3 FogColor;
uniform float FogNear;
uniform float FogFar;
uniform float FogDensity;
uniform float FogHeight;
uniform float FogHeightFalloff;
uniform float Wetness;
uniform sampler2D WaterCaustic;
uniform int WaterCausticsOn;
uniform float WaterCausticLevel;
uniform vec3 WaterCausticSun;
uniform float WaterCausticMove;
uniform int ProbeEnabled;
uniform vec3 ProbeSky;
uniform vec3 ProbeGround;
uniform sampler2D TerrainSand;
uniform sampler2D TerrainGrass;
uniform sampler2D TerrainGrass2;
uniform sampler2D TerrainRock;
uniform sampler2D TerrainSnow;
uniform sampler2D TerrainRockN;
uniform sampler2D TerrainBlendMap;
uniform int TerrainHasBlend;
uniform float TerrainWorldW;
uniform float TerrainWorldD;
uniform float TerrainOriginX;
uniform float TerrainOriginZ;
uniform int TerrainSplatOn;
uniform int TerrainHasSand;
uniform int TerrainHasGrass;
uniform int TerrainHasGrass2;
uniform int TerrainHasRock;
uniform int TerrainHasSnow;
uniform int TerrainHasRockN;
uniform int TerrainSnowOn;
uniform float TerrainWaterY;
uniform float TerrainBlend;
uniform float TerrainGrassCover;
uniform float TerrainSnowH;
uniform vec3 TerrainRockColor;
uniform float TerrainFogFalloff;
uniform vec3 TerrainSunDir;
uniform vec3 TerrainSunColor;
uniform vec3 CamWorldPos;
uniform vec4 WaterClipPlane;
out vec4 FragColor;
`

const mbterrainFragmentTail = shadeLightGLSL + `
float terrainHash(vec2 p) {
    p = fract(p * vec2(123.34, 345.45));
    p += dot(p, p + 34.345);
    return fract(p.x * p.y);
}

float terrainLuma(vec3 c) {
    return dot(c, vec3(0.299, 0.587, 0.114));
}

// Stochastic tile break (Heitz / Deliot sampling, Booth height blend):
// four virtual offsets, blended by luminance so the mix does not go grey.
vec3 stochasticRGB(sampler2D tex, vec2 uv) {
    vec2 iuv = floor(uv);
    vec2 fuv = fract(uv);
    fuv = fuv * fuv * (3.0 - 2.0 * fuv);
    vec2 o00 = vec2(terrainHash(iuv), terrainHash(iuv + 13.7));
    vec2 o10 = vec2(terrainHash(iuv + vec2(1.0, 0.0)), terrainHash(iuv + vec2(1.0, 0.0) + 13.7));
    vec2 o01 = vec2(terrainHash(iuv + vec2(0.0, 1.0)), terrainHash(iuv + vec2(0.0, 1.0) + 13.7));
    vec2 o11 = vec2(terrainHash(iuv + vec2(1.0, 1.0)), terrainHash(iuv + vec2(1.0, 1.0) + 13.7));
    vec3 c00 = texture(tex, uv + o00).rgb;
    vec3 c10 = texture(tex, uv + o10).rgb;
    vec3 c01 = texture(tex, uv + o01).rgb;
    vec3 c11 = texture(tex, uv + o11).rgb;
    float h00 = terrainLuma(c00) + (1.0 - fuv.x) * (1.0 - fuv.y);
    float h10 = terrainLuma(c10) + fuv.x * (1.0 - fuv.y);
    float h01 = terrainLuma(c01) + (1.0 - fuv.x) * fuv.y;
    float h11 = terrainLuma(c11) + fuv.x * fuv.y;
    float m = max(max(h00, h10), max(h01, h11)) - 0.18;
    float b00 = max(h00 - m, 0.0);
    float b10 = max(h10 - m, 0.0);
    float b01 = max(h01 - m, 0.0);
    float b11 = max(h11 - m, 0.0);
    float s = max(b00 + b10 + b01 + b11, 0.0001);
    return (c00 * b00 + c10 * b10 + c01 * b01 + c11 * b11) / s;
}

vec3 triplanarRGB(sampler2D tex, int has, vec3 fallback, vec3 wp, vec3 n, float sc) {
    vec3 an = abs(n);
    an = pow(max(an, vec3(0.0)), vec3(3.0));
    float s = an.x + an.y + an.z;
    if (s < 1e-5) { return fallback; }
    an /= s;
    if (has == 0) { return fallback; }
    vec3 cx = texture(tex, wp.zy * sc).rgb;
    vec3 cy = stochasticRGB(tex, wp.xz * sc);
    vec3 cz = texture(tex, wp.xy * sc).rgb;
    return cx * an.x + cy * an.y + cz * an.z;
}

vec3 heightBlend(vec3 c0, vec3 c1, vec3 c2, vec3 c3, vec3 c4, float w0, float w1, float w2, float w3, float w4) {
    float h0 = terrainLuma(c0) + w0;
    float h1 = terrainLuma(c1) + w1;
    float h2 = terrainLuma(c2) + w2;
    float h3 = terrainLuma(c3) + w3;
    float h4 = terrainLuma(c4) + w4;
    float m = max(max(max(h0, h1), max(h2, h3)), h4) - 0.22;
    float b0 = max(h0 - m, 0.0);
    float b1 = max(h1 - m, 0.0);
    float b2 = max(h2 - m, 0.0);
    float b3 = max(h3 - m, 0.0);
    float b4 = max(h4 - m, 0.0);
    float s = max(b0 + b1 + b2 + b3 + b4, 0.0001);
    return (c0 * b0 + c1 * b1 + c2 * b2 + c3 * b3 + c4 * b4) / s;
}

vec4 terrainSplat(inout vec3 n) {
    float h = WorldPos.y;
    float ny = clamp(n.y, 0.0, 1.0);
    float cover = clamp(TerrainGrassCover, 0.0, 1.0);
    float grassLo = mix(0.55, 0.32, cover);
    // High rockMask is flat ground (grass). Low is a cliff.
    float rockMask = smoothstep(grassLo, grassLo + 0.35, ny);
    float hFac = smoothstep(TerrainWaterY + 1.5, max(TerrainSnowH * 0.7, TerrainWaterY + 8.0), h);
    vec3 valleyGrass = vec3(0.16, 0.30, 0.12);
    vec3 hillGrass = vec3(0.36, 0.38, 0.18);
    vec3 grass = mix(valleyGrass, hillGrass, hFac);
    vec3 rock = TerrainRockColor;
    if (rock.x + rock.y + rock.z < 0.05) { rock = vec3(0.42, 0.36, 0.30); }
    vec3 dirt = vec3(0.34, 0.24, 0.14);
    vec3 sand = vec3(0.64, 0.55, 0.34);
    vec3 snow = vec3(0.90, 0.92, 0.94);
    grass = mix(grass, triplanarRGB(TerrainGrass, TerrainHasGrass, grass, WorldPos, n, 0.09), 0.86);
    grass = mix(grass, triplanarRGB(TerrainGrass2, TerrainHasGrass2, grass, WorldPos, n, 0.13), 0.42);
    rock = mix(rock, triplanarRGB(TerrainRock, TerrainHasRock, rock, WorldPos, n, 0.055), 0.88);
    sand = mix(sand, triplanarRGB(TerrainSand, TerrainHasSand, sand, WorldPos, n, 0.10), 0.72);
    snow = mix(snow, triplanarRGB(TerrainSnow, TerrainHasSnow, snow, WorldPos, n, 0.07), 0.55);
    float patch = terrainHash(floor(WorldPos.xz * 0.07));
    float fine = terrainHash(WorldPos.xz * 1.9);
    grass = mix(grass, grass * vec3(0.72, 1.05, 0.58), patch * 0.28);
    grass *= 0.94 + 0.06 * fine;
    float sandMask = 1.0 - smoothstep(TerrainWaterY, TerrainWaterY + max(TerrainBlend, 1.2) * 2.0, h);
    float snowMask = 0.0;
    if (TerrainSnowOn != 0) {
        snowMask = smoothstep(TerrainSnowH - 1.2, TerrainSnowH + 2.0, h) * smoothstep(0.40, 0.78, ny);
    }
    if (TerrainHasBlend != 0) {
        float sx = WorldPos.x;
        float sz = -WorldPos.z;
        vec2 buv = vec2((sx - TerrainOriginX) / max(TerrainWorldW, 1.0), (sz - TerrainOriginZ) / max(TerrainWorldD, 1.0));
        buv = clamp(buv, 0.0, 1.0);
        vec4 bw = texture(TerrainBlendMap, buv);
        float sum = bw.r + bw.g + bw.b + bw.a;
        if (sum > 0.04) {
            float k = clamp(sum, 0.0, 1.0);
            sandMask = mix(sandMask, bw.r, k);
            snowMask = mix(snowMask, bw.a, k);
            rockMask = mix(rockMask, bw.g / max(bw.g + bw.b, 0.001), k);
        }
    }
    float open = (1.0 - sandMask) * (1.0 - snowMask);
    float dirtW = (1.0 - rockMask) * smoothstep(grassLo, grassLo + 0.18, ny) * (1.0 - hFac) * 0.55 * open;
    vec3 albedo = heightBlend(rock, grass, dirt, sand, snow,
        (1.0 - rockMask) * open,
        rockMask * open,
        dirtW,
        sandMask,
        snowMask);
    float b0 = terrainHash(WorldPos.xz * 2.2);
    float bx = terrainHash((WorldPos.xz + vec2(0.11, 0.0)) * 2.2);
    float bz = terrainHash((WorldPos.xz + vec2(0.0, 0.11)) * 2.2);
    float bump = mix(0.42, 0.10, rockMask) * (1.0 - snowMask);
    n = normalize(n + vec3(b0 - bx, 0.0, b0 - bz) * bump);
    if (TerrainHasRockN != 0 && rockMask < 0.85) {
        vec3 up = vec3(0.0, 0.0, 1.0);
        if (abs(n.y) > 0.92) { up = vec3(1.0, 0.0, 0.0); }
        vec3 t = normalize(cross(up, n));
        vec3 b = normalize(cross(n, t));
        mat3 TBN = mat3(t, b, n);
        vec3 rn = texture(TerrainRockN, WorldPos.xz * 0.07).rgb * 2.0 - 1.0;
        n = normalize(mix(n, TBN * rn, (1.0 - rockMask) * 0.55));
    }
    return vec4(albedo, 1.0);
}

void main() {
    if (dot(WaterClipPlane.xyz, WaterClipPlane.xyz) > 1e-8) {
        if (dot(WorldPos, WaterClipPlane.xyz) + WaterClipPlane.w < 0.0) {
            discard;
        }
    }
    vec3 n = normalize(WorldNormal);
    vec4 texMixed = vec4(0.22, 0.30, 0.16, 1.0);
    if (TerrainSplatOn != 0) {
        texMixed = terrainSplat(n);
    } else {
#if MAT_TEXTURES > 0
        if (MatTexVisible(0)) {
            texMixed = texture(MatTexture[0], FragTexcoord * MatTexRepeat(0) + MatTexOffset(0));
        }
#endif
    }
    vec3 albedo = texMixed.rgb * MatDiffuseColor;
    vec3 nDx = dFdx(n);
    vec3 nDy = dFdy(n);
    float cavity = clamp(dot(normalize(n + nDx + nDy), n), 0.0, 1.0);
    albedo *= mix(0.74, 1.0, cavity);
    vec3 L = TerrainSunDir;
    if (dot(L, L) < 1e-6) { L = vec3(0.35, 0.82, 0.28); }
    L = normalize(L);
    float ndl = max(dot(n, L), 0.0);
    float sunSh = sunShadowFactor(WorldPos, n, L);
    float locSh = localShadows();
    float sh = sunSh * locSh;
    vec3 sunColor = TerrainSunColor;
    if (dot(sunColor, sunColor) < 1e-6) {
        sunColor = vec3(1.0, 0.92, 0.78);
    }
    vec3 sun = sunColor * ndl * sh;
    vec3 hemi = mix(vec3(0.30, 0.22, 0.14), vec3(0.40, 0.50, 0.62), n.y * 0.5 + 0.5);
    vec3 V = normalize(CamWorldPos - WorldPos);
    float flat = smoothstep(mix(0.55, 0.32, clamp(TerrainGrassCover, 0.0, 1.0)), mix(0.90, 0.67, clamp(TerrainGrassCover, 0.0, 1.0)), clamp(n.y, 0.0, 1.0));
    float gloss = mix(0.20, 0.04, flat);
    if (TerrainSnowOn != 0) {
        float snowM = smoothstep(TerrainSnowH - 1.2, TerrainSnowH + 2.0, WorldPos.y) * smoothstep(0.40, 0.78, n.y);
        gloss = mix(gloss, 0.38, snowM);
    }
    float spec = pow(max(dot(reflect(-L, n), V), 0.0), mix(18.0, 48.0, gloss)) * ndl * sh * gloss;
    vec3 shadowTint = ShadowColor * (1.0 - sh) * albedo;
    vec3 local = vec3(0.0);
    vec3 nView = normalize(Normal);
    vec3 viewL = normalize(-Position.xyz);
#if POINT_LIGHTS>0
    for (int i = 0; i < POINT_LIGHTS; ++i) {
        vec3 lPos = PointLightPosition(i) - Position.xyz;
        float dist = length(lPos);
        if (dist > 0.001) {
            vec3 ld = lPos / dist;
            float att = shadeAttenuation(dist, PointLightLinearDecay(i), PointLightQuadraticDecay(i), PointLightConstant(i));
            if (att > 0.001) {
                float lndl = max(dot(nView, ld), 0.0);
                float psh = pointShadowFactor(i, WorldPos, n);
                float lspec = shadeSpec(max(dot(nView, normalize(ld + viewL)), 0.0), 32.0) * lndl * att * psh * 0.12;
                local += PointLightColor(i) * (albedo * lndl * att * psh + lspec);
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
                float lndl = max(dot(nView, ld), 0.0);
                float ssh = spotShadowFactor(i, WorldPos, n);
                float lspec = shadeSpec(max(dot(nView, normalize(ld + viewL)), 0.0), 32.0) * lndl * att * ssh * 0.12;
                local += SpotLightColor(i) * (albedo * lndl * att * ssh + lspec);
            }
        }
    }
#endif
    vec3 col = albedo * (sun * 1.15 + hemi * 0.62) + shadowTint + spec * sunColor + local;
    FragColor = vec4(shadeKnee(col), texMixed.a * MatOpacity);
    float shoreWet = 1.0 - smoothstep(0.06, 1.9, abs(WorldPos.y - WaterCausticLevel));
    float wet = clamp(max(Wetness, shoreWet * 0.82), 0.0, 1.0);
    if (wet > 0.001) {
        float up = clamp(n.y, 0.0, 1.0);
        FragColor.rgb *= mix(1.0, 0.72, wet * up);
        FragColor.rgb += spec * wet * up;
    }
    if (WaterCausticsOn != 0 && WorldPos.y < WaterCausticLevel - 0.02) {
        vec3 Lc = WaterCausticSun;
        if (dot(Lc, Lc) < 1e-6) { Lc = vec3(0.4, 0.8, 0.3); }
        Lc = normalize(Lc);
        float depth = WaterCausticLevel - WorldPos.y;
        float ly = max(abs(Lc.y), 0.12);
        vec2 proj = WorldPos.xz + Lc.xz * (depth / ly);
        vec2 uv1 = proj * 0.07 + vec2(WaterCausticMove * 1.6, WaterCausticMove * 0.9);
        vec2 uv2 = proj * 0.13 + vec2(-WaterCausticMove * 0.7, WaterCausticMove * 1.3);
        float c = textureLod(WaterCaustic, uv1, 0.0).r * textureLod(WaterCaustic, uv2, 0.0).g;
        float fade = exp(-depth * 0.045) * smoothstep(0.02, 0.45, depth);
        float ndl = max(dot(n, Lc), 0.0);
        FragColor.rgb += vec3(0.45, 0.72, 0.85) * c * ndl * fade * 0.85;
    }
    if (FogMode != 0) {
        float dist = length(Position.xyz);
        float f = 1.0;
        if (FogMode == 1) {
            float span = max(FogFar - FogNear, 0.0001);
            f = clamp((FogFar - dist) / span, 0.0, 1.0);
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

var mbterrainFragment = mbterrainFragmentHead + mbshadowSampleGLSL + mbterrainFragmentTail

// mbclouds: 3.3 fragment raymarch through layered 2D noise (no compute / 3D tex).
// Visual stand-in for Terrain-OpenGL volumetric_clouds.comp.

const mbcloudsVertex = `#include <attributes>
uniform mat4 ModelViewMatrix;
uniform mat4 MVP;
uniform mat4 ModelMatrix;
out vec3 WorldPos;
out vec2 FragTexcoord;
void main() {
    vec4 world = ModelMatrix * vec4(VertexPosition, 1.0);
    WorldPos = world.xyz;
    FragTexcoord = VertexTexcoord;
    gl_Position = MVP * vec4(VertexPosition, 1.0);
}
`

const mbcloudsFragment = `precision highp float;
in vec3 WorldPos;
in vec2 FragTexcoord;
uniform sampler2D CloudNoise;
uniform int CloudHasNoise;
uniform vec3 CloudCam;
uniform vec3 CloudSun;
uniform float CloudTime;
uniform float CloudCover;
uniform float CloudDensity;
uniform vec3 CloudColor;
uniform vec3 CloudSkyBot;
uniform vec3 CloudWind;
out vec4 FragColor;

float hash13(vec3 p) {
    return fract(sin(dot(p, vec3(127.1, 311.7, 74.7))) * 43758.5453);
}

float cloudNoise(vec3 p) {
    if (CloudHasNoise != 0) {
        vec2 woff = CloudWind.xy * CloudTime * 0.04;
        float a = texture(CloudNoise, p.xz * 0.004 + vec2(CloudTime * 0.012, CloudTime * 0.007) + woff).r;
        float b = texture(CloudNoise, p.xz * 0.011 + vec2(-CloudTime * 0.008, CloudTime * 0.01) + woff * 0.6).g;
        return mix(a, b, 0.45);
    }
    return hash13(floor(p * 0.04));
}

void main() {
    vec3 ro = CloudCam;
    vec3 rd = normalize(WorldPos - CloudCam);
    if (rd.y < 0.02) { discard; }
    float t0 = (95.0 - ro.y) / rd.y;
    float t1 = (145.0 - ro.y) / rd.y;
    if (t1 < t0) { float tmp = t0; t0 = t1; t1 = tmp; }
    t0 = max(t0, 0.0);
    if (t1 < 0.0) { discard; }
    float step = (t1 - t0) / 12.0;
    if (step < 1.0) { step = 1.0; }
    float acc = 0.0;
    vec3 pos = ro + rd * t0;
    for (int i = 0; i < 12; i++) {
        float n = cloudNoise(pos);
        float h = clamp((pos.y - 95.0) / 50.0, 0.0, 1.0);
        float d = smoothstep(1.0 - CloudCover, 1.0, n) * (1.0 - h * 0.65);
        acc += d * CloudDensity * 0.22;
        pos += rd * step;
    }
    acc = clamp(acc, 0.0, 1.0);
    if (acc < 0.02) { discard; }
    float sun = pow(max(dot(rd, normalize(CloudSun)), 0.0), 8.0);
    vec3 col = mix(CloudColor * 0.55, CloudColor, acc);
    col += sun * vec3(1.0, 0.85, 0.55) * 0.35 * acc;
    col = mix(CloudSkyBot, col, 0.85);
    FragColor = vec4(col, acc * 0.78);
}
`

func registerTerrainShaders(r *renderer.Renderer) {
	r.AddShader("mbterrain_fragment", mbterrainFragment)
	r.AddProgram("mbterrain", "mbshadow_vertex", "mbterrain_fragment")
	r.AddShader("mbclouds_vertex", mbcloudsVertex)
	r.AddShader("mbclouds_fragment", mbcloudsFragment)
	r.AddProgram("mbclouds", "mbclouds_vertex", "mbclouds_fragment")
}
