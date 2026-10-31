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
noperspective in vec2 FragTexcoordAffine;
noperspective in vec3 WorldPosAffine;
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
uniform vec3 FogSunDir;
uniform vec3 FogSunCol;
uniform vec3 FogCam;
// Haze toward the sun is brighter and warmer, with the sky's own halo
// terms, so fogged land near the sun matches the sky behind it.
vec3 fogTint(vec3 fogColor, vec3 wp) {
    if (dot(FogSunDir, FogSunDir) < 0.5) { return fogColor; }
    float s = max(dot(normalize(wp - FogCam), FogSunDir), 0.0);
    return fogColor + FogSunCol * (pow(s, 8.0) * 0.10 + pow(s, 64.0) * 0.22);
}
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
uniform int TerrainCheap;
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
` + retroFragGLSL + `
// Splat texture lookups use this position: WorldPos, or its screen-linear
// copy as RetroAffine rises (the PS1 warp on the ground textures).
vec3 retroTexPos() {
    if (RetroAffine <= 0.0) { return WorldPos; }
    return mix(WorldPos, WorldPosAffine, clamp(RetroAffine, 0.0, 1.0));
}
`

const mbterrainFragmentTail = shadeLightGLSL + `
float terrainHash(vec2 p) {
    p = fract(p * vec2(123.34, 345.45));
    p += dot(p, p + 34.345);
    return fract(p.x * p.y);
}

// terrainNoise is smooth value noise. The old per-pixel hash at metre
// frequency aliased into wavy stripes across the grass.
float terrainNoise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    float a = terrainHash(i);
    float b = terrainHash(i + vec2(1.0, 0.0));
    float c = terrainHash(i + vec2(0.0, 1.0));
    float d = terrainHash(i + vec2(1.0, 1.0));
    return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
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
    // Low tier: one planar read per layer instead of six.
    if (TerrainCheap != 0) {
        return has == 0 ? fallback : texture(tex, wp.xz * sc).rgb;
    }
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
    vec3 tp = retroTexPos();
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
    grass = mix(grass, triplanarRGB(TerrainGrass, TerrainHasGrass, grass, tp, n, 0.09), 0.86);
    grass = mix(grass, triplanarRGB(TerrainGrass2, TerrainHasGrass2, grass, tp, n, 0.13), 0.42);
    rock = mix(rock, triplanarRGB(TerrainRock, TerrainHasRock, rock, tp, n, 0.055), 0.88);
    // Cliffs read as one flat stain from afar: add horizontal strata and
    // darker crevices so a face has layers, plus a slow tone drift.
    float strata = terrainNoise(vec2(WorldPos.y * 0.55, (WorldPos.x + WorldPos.z) * 0.02));
    float crev = smoothstep(0.62, 0.92, terrainNoise(WorldPos.xz * 0.21 + WorldPos.y * 0.13));
    rock *= (0.84 + 0.30 * strata) * (1.0 - 0.32 * crev);
    rock *= 0.9 + 0.2 * terrainNoise(WorldPos.xz * 0.004 + 3.7);
    sand = mix(sand, triplanarRGB(TerrainSand, TerrainHasSand, sand, tp, n, 0.10), 0.72);
    snow = mix(snow, triplanarRGB(TerrainSnow, TerrainHasSnow, snow, tp, n, 0.07), 0.55);
    // Large dry and lush patches, then a meadow-scale mottle. The fine
    // layer fades with distance before it can shimmer.
    float camD = length(Position.xyz); // view space: distance to the camera
    float big = terrainNoise(WorldPos.xz * 0.006) * 0.65 + terrainNoise(WorldPos.xz * 0.019) * 0.35;
    float mid = terrainNoise(WorldPos.xz * 0.11);
    float fine = mix(terrainNoise(WorldPos.xz * 0.9), 0.5, smoothstep(25.0, 70.0, camD));
    vec3 dry = grass * vec3(1.18, 1.02, 0.62);
    vec3 lush = grass * vec3(0.78, 0.96, 0.74);
    grass = mix(lush, dry, smoothstep(0.32, 0.74, big));
    grass *= 0.88 + 0.16 * mid;
    grass *= 0.95 + 0.10 * fine;
    // Pull toward olive: texture greens alone read as plastic.
    float gl = terrainLuma(grass);
    grass = mix(vec3(gl), grass, 0.78) * vec3(1.04, 1.0, 0.92);
    float sandMask = 1.0 - smoothstep(TerrainWaterY, TerrainWaterY + max(TerrainBlend, 1.2) * 2.0, h);
    float snowMask = 0.0;
    if (TerrainSnowOn != 0) {
        // A snow line wanders with the ground (big drifts and small
        // patches) and reaches lower on flat ground than on slopes. A
        // single height cut a ruler-straight band across every mountain.
        float jit = (terrainNoise(WorldPos.xz * 0.012) - 0.5) * 14.0 + (terrainNoise(WorldPos.xz * 0.09) - 0.5) * 4.0;
        float line = TerrainSnowH + jit - ny * 6.0 + 3.0;
        snowMask = smoothstep(line - 2.5, line + 2.5, h) * smoothstep(0.36, 0.72, ny);
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
    float b0 = terrainNoise(WorldPos.xz * 1.6);
    float bx = terrainNoise((WorldPos.xz + vec2(0.15, 0.0)) * 1.6);
    float bz = terrainNoise((WorldPos.xz + vec2(0.0, 0.15)) * 1.6);
    // Bump fades out with distance: past ~40 m it only adds shimmer.
    float bump = mix(0.42, 0.10, rockMask) * (1.0 - snowMask) * (1.0 - smoothstep(15.0, 45.0, camD));
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
    vec3 n = retroFlatNormal(WorldPos, normalize(WorldNormal));
    vec4 texMixed = vec4(0.22, 0.30, 0.16, 1.0);
    if (TerrainSplatOn != 0) {
        texMixed = terrainSplat(n);
    } else {
#if MAT_TEXTURES > 0
        if (MatTexVisible(0)) {
            texMixed = texture(MatTexture[0], retroUV() * MatTexRepeat(0) + MatTexOffset(0));
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
    float groundLevel = smoothstep(mix(0.55, 0.32, clamp(TerrainGrassCover, 0.0, 1.0)), mix(0.90, 0.67, clamp(TerrainGrassCover, 0.0, 1.0)), clamp(n.y, 0.0, 1.0));
    float gloss = mix(0.20, 0.04, groundLevel);
    if (TerrainSnowOn != 0) {
        float jitG = (terrainNoise(WorldPos.xz * 0.012) - 0.5) * 14.0 + (terrainNoise(WorldPos.xz * 0.09) - 0.5) * 4.0;
        float lineG = TerrainSnowH + jitG - n.y * 6.0 + 3.0;
        float snowM = smoothstep(lineG - 2.5, lineG + 2.5, WorldPos.y) * smoothstep(0.36, 0.72, n.y);
        gloss = mix(gloss, 0.38, snowM);
    }
    float spec = pow(max(dot(reflect(-L, n), V), 0.0), mix(18.0, 48.0, gloss)) * ndl * sh * gloss;
    vec3 shadowTint = ShadowColor * (1.0 - sh) * albedo;
    vec3 local = vec3(0.0);
    vec3 nView = retroFlatNormal(Position.xyz, normalize(Normal));
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
        FragColor.rgb = mix(fogTint(FogColor, WorldPos), FragColor.rgb, retroFog(clamp(f, 0.0, 1.0)));
    }
}
`

var mbterrainFragment = mbterrainFragmentHead + mbshadowSampleGLSL + mbterrainFragmentTail

// mbclouds: one sheet above the camera. Value-noise fbm in world XZ, so
// clouds stay put as you move and drift with CloudDrift. Two short taps
// toward the sun shade the side away from it.

const mbcloudsVertex = `#include <attributes>
uniform mat4 ModelMatrix;
uniform mat4 CloudVP;
out vec3 WorldPos;
void main() {
    WorldPos = (ModelMatrix * vec4(VertexPosition, 1.0)).xyz;
    gl_Position = CloudVP * vec4(WorldPos, 1.0);
    // Past the camera's far plane the sheet would end in a hard line; hold
    // it just inside, like the sky.
    gl_Position.z = min(gl_Position.z, gl_Position.w * 0.99999);
}
`

const mbcloudsFragment = `precision highp float;
in vec3 WorldPos;
uniform vec3 CloudCam;
uniform vec3 CloudSun;
uniform vec3 CloudSunColor;
uniform vec3 CloudHaze;
uniform vec3 CloudColor;
uniform vec3 CloudDrift;
uniform float CloudCover;
uniform float CloudDensity;
uniform float CloudScale;
uniform float CloudTime;
uniform float CloudRadius;
uniform float CloudNight;
uniform float CloudCheap; // low tier: fewer octaves, one sun tap
out vec4 FragColor;

float hash(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
}
float vnoise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    vec2 u = f * f * (3.0 - 2.0 * f);
    float a = hash(i);
    float b = hash(i + vec2(1.0, 0.0));
    float c = hash(i + vec2(0.0, 1.0));
    float d = hash(i + vec2(1.0, 1.0));
    return mix(mix(a, b, u.x), mix(c, d, u.x), u.y);
}
float fbm(vec2 p, int oct) {
    float s = 0.0;
    float amp = 0.5;
    float norm = 0.0;
    for (int i = 0; i < 5; i++) {
        if (i >= oct) break;
        s += amp * vnoise(p);
        norm += amp;
        p = mat2(1.6, 1.2, -1.2, 1.6) * p + vec2(17.0, 9.0);
        amp *= 0.5;
    }
    return s / norm;
}
// Cloud noise at p: big shapes, eaten at the edges by finer noise that
// slowly changes so the clouds boil as they drift.
float cloudNoise(vec2 p, int oct) {
    float base = fbm(p, 2);
    float detail = fbm(p * 3.1 + vec2(CloudTime * 0.012, -CloudTime * 0.008), oct);
    return base * 0.72 + detail * 0.28;
}
// Coverage from noise: 0 clear sky, 1 cloud.
float cloudCov(float n) {
    float edge = 1.0 - CloudCover;
    return smoothstep(edge - 0.06, edge + 0.16, n);
}
// Thickness past the edge. Under full cover the coverage is 1 everywhere;
// this still varies, so an overcast sky keeps its shape.
float cloudThick(float n) {
    return clamp((n - (1.0 - CloudCover)) / 0.32, 0.0, 1.0);
}

void main() {
    vec2 xz = WorldPos.xz - CloudCam.xz;
    float dist = length(xz) / CloudRadius;
    float fade = 1.0 - smoothstep(0.45, 0.95, dist);
    if (fade <= 0.0) discard;
    vec2 p = (WorldPos.xz + CloudDrift.xz) * CloudScale;
    // Fewer octaves far away, where they would only shimmer
    int oct = CloudCheap > 0.5 ? 2 : (dist < 0.25 ? 4 : 3);
    float n = cloudNoise(p, oct);
    float c = cloudCov(n);
    if (c < 0.01) discard;
    float th = cloudThick(n);
    vec3 sun = normalize(CloudSun);
    // Thicker toward the sun: this side is in the cloud's own shadow
    vec2 toSun = normalize(sun.xz + vec2(1e-4)) * 0.35;
    float ahead = cloudThick(cloudNoise(p + toSun * 1.5, CloudCheap > 0.5 ? 1 : 2));
    if (CloudCheap < 0.5) {
        ahead = cloudThick(cloudNoise(p + toSun, 2)) * 0.6 + ahead * 0.4;
    }
    float lit = clamp(1.0 - (ahead - th * 0.5) * 0.9, 0.35, 1.0);
    float dens = clamp(CloudDensity, 0.3, 2.0);
    // Thick middles go grey underneath; storms (density over 1) darker still
    float under = mix(1.0, 0.62, th) * mix(1.0, 0.5, clamp(dens - 1.0, 0.0, 1.0));
    float day = 1.0 - CloudNight;
    vec3 ambient = mix(CloudHaze, vec3(0.62, 0.68, 0.78), 0.4) * (0.55 + 0.25 * day);
    // A low sun still lights the undersides from the side: sunset clouds glow
    vec3 sunC = CloudSunColor * (0.45 + 0.55 * max(sun.y, 0.0));
    vec3 col = CloudColor * (ambient * under + sunC * lit * under * 0.8);
    // Sunset: a low sun lights the clouds from below in its own color
    float low = 1.0 - smoothstep(0.03, 0.42, sun.y);
    col += CloudSunColor * low * day * (0.25 + 0.45 * lit) * (1.0 - 0.4 * th);
    // A bright rim where you look toward the sun through a thin edge
    vec3 view = normalize(WorldPos - CloudCam);
    float rim = pow(max(dot(view, sun), 0.0), 6.0) * (1.0 - th) * 0.9;
    col += CloudSunColor * rim * day;
    // Night: dim moonlit grey
    col = mix(col, CloudHaze * 0.9 + vec3(0.03, 0.035, 0.05), CloudNight * 0.85);
    // Far clouds melt into the haze, then away
    col = mix(col, CloudHaze, smoothstep(0.15, 0.85, dist) * 0.75);
    float alpha = c * fade * clamp(0.55 + 0.45 * dens, 0.4, 0.97);
    FragColor = vec4(col, alpha);
}
`

func registerTerrainShaders(r *renderer.Renderer) {
	r.AddShader("mbterrain_fragment", mbterrainFragment)
	r.AddProgram("mbterrain", "mbshadow_vertex", "mbterrain_fragment")
	r.AddShader("mbclouds_vertex", mbcloudsVertex)
	r.AddShader("mbclouds_fragment", mbcloudsFragment)
	r.AddProgram("mbclouds", "mbclouds_vertex", "mbclouds_fragment")
}
