package runtime

// mbshadow is G3N "standard" plus a shadow atlas (CSM + point/spot) and filters.
// G3N shaman prepends "#version 330 core".

const mbshadowVertex = `#include <attributes>
uniform mat4 ModelViewMatrix;
uniform mat3 NormalMatrix;
uniform mat4 MVP;
uniform mat4 ModelMatrix;
uniform mat4 LightVP[4];
uniform vec4 ShadowTexelWorld;
#include <material>
#include <morphtarget_vertex_declaration>
#include <bones_vertex_declaration>
out vec4 Position;
out vec3 Normal;
out vec2 FragTexcoord;
out vec3 WorldPos;
out vec3 WorldNormal;
out vec4 LightSpacePos[4];
void main() {
    Position = ModelViewMatrix * vec4(VertexPosition, 1.0);
    Normal = normalize(NormalMatrix * VertexNormal);
    vec2 texcoord = VertexTexcoord;
#if MAT_TEXTURES > 0
    if (MatTexFlipY(0)) { texcoord.y = 1.0 - texcoord.y; }
#endif
    FragTexcoord = texcoord;
    vec3 vPosition = VertexPosition;
    mat4 finalWorld = mat4(1.0);
    #include <morphtarget_vertex>
    #include <bones_vertex>
    vec4 world = ModelMatrix * finalWorld * vec4(vPosition, 1.0);
    WorldPos = world.xyz;
    vec3 wn = mat3(ModelMatrix) * mat3(finalWorld) * VertexNormal;
    if (dot(wn, wn) < 1e-10) { wn = vec3(0.0, 1.0, 0.0); }
    WorldNormal = normalize(wn);
    vec3 wpShadow = WorldPos + WorldNormal * (ShadowTexelWorld.x * 2.5);
    LightSpacePos[0] = LightVP[0] * vec4(wpShadow, 1.0);
    LightSpacePos[1] = LightVP[1] * vec4(wpShadow, 1.0);
    LightSpacePos[2] = LightVP[2] * vec4(wpShadow, 1.0);
    LightSpacePos[3] = LightVP[3] * vec4(wpShadow, 1.0);
    gl_Position = MVP * finalWorld * vec4(vPosition, 1.0);
}
`

const mbshadowFragmentHead = `precision highp float;
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
uniform int ProbeEnabled;
uniform vec3 ProbeSky;
uniform vec3 ProbeGround;
uniform sampler2D WaterCaustic;
uniform int WaterCausticsOn;
uniform float WaterCausticLevel;
uniform vec3 WaterCausticSun;
uniform float WaterCausticMove;
out vec4 FragColor;
`

const mbshadowFragmentTail = `
void main() {
    vec4 texMixed = vec4(1);
    #if MAT_TEXTURES > 0
        bool firstTex = true;
        if (MatTexVisible(0)) {
            vec4 texColor = texture(MatTexture[0], FragTexcoord * MatTexRepeat(0) + MatTexOffset(0));
            texMixed = texColor;
            firstTex = false;
        }
        #if MAT_TEXTURES > 1
            if (MatTexVisible(1)) {
                vec4 texColor = texture(MatTexture[1], FragTexcoord * MatTexRepeat(1) + MatTexOffset(1));
                if (firstTex) { texMixed = texColor; firstTex = false; } else { texMixed = Blend(texMixed, texColor); }
            }
            #if MAT_TEXTURES > 2
                if (MatTexVisible(2)) {
                    vec4 texColor = texture(MatTexture[2], FragTexcoord * MatTexRepeat(2) + MatTexOffset(2));
                    if (firstTex) { texMixed = texColor; } else { texMixed = Blend(texMixed, texColor); }
                }
            #endif
        #endif
    #endif
    vec4 matDiffuse = vec4(MatDiffuseColor, MatOpacity) * texMixed;
    vec4 matAmbient = vec4(MatAmbientColor, MatOpacity) * texMixed;
    vec3 fragNormal = normalize(Normal);
    if (!gl_FrontFacing) { fragNormal = -fragNormal; }
    vec3 camDir = normalize(-Position.xyz);
    vec3 totalDiffuse = vec3(0.0);
    vec3 totalSpec = vec3(0.0);
    vec3 ambientColor = MatEmissiveColor;

#if AMB_LIGHTS>0
    for (int i = 0; i < AMB_LIGHTS; ++i) {
        ambientColor += AmbientLightColor[i] * vec3(matAmbient);
    }
#endif

    vec3 wN = normalize(WorldNormal);
    float sunSh = sunShadowFactor(WorldPos, wN, ShadowSunDir);

#if DIR_LIGHTS>0
    for (int i = 0; i < DIR_LIGHTS; ++i) {
        vec3 lightDir = normalize(DirLightPosition(i));
        float ndl = max(dot(fragNormal, lightDir), 0.0);
        float sh = (i == 0) ? sunSh : 1.0;
        totalDiffuse += DirLightColor(i) * vec3(matDiffuse) * (ndl * sh);
        if (ndl > 0.0 && MatShininess > 0.0) {
            vec3 halfV = normalize(lightDir + camDir);
            float ndh = max(dot(fragNormal, halfV), 0.0);
            totalSpec += DirLightColor(i) * MatSpecularColor * (pow(ndh, MatShininess) * sh);
        }
    }
#endif

#if POINT_LIGHTS>0
    for (int i = 0; i < POINT_LIGHTS; ++i) {
        vec3 lPos = PointLightPosition(i) - Position.xyz;
        float dist = length(lPos);
        if (dist > 0.001) {
            vec3 lightDir = lPos / dist;
            float att = 1.0 / (1.0 + PointLightLinearDecay(i) * dist + PointLightQuadraticDecay(i) * dist * dist);
            float ndl = max(dot(fragNormal, lightDir), 0.0);
            float psh = pointShadowFactor(i, WorldPos, wN);
            totalDiffuse += PointLightColor(i) * vec3(matDiffuse) * (ndl * att * psh);
            if (ndl > 0.0 && MatShininess > 0.0) {
                vec3 halfV = normalize(lightDir + camDir);
                float ndh = max(dot(fragNormal, halfV), 0.0);
                totalSpec += PointLightColor(i) * MatSpecularColor * (pow(ndh, MatShininess) * att * psh);
            }
        }
    }
#endif

#if SPOT_LIGHTS>0
    for (int i = 0; i < SPOT_LIGHTS; ++i) {
        vec3 lPos = SpotLightPosition(i) - Position.xyz;
        float dist = length(lPos);
        if (dist > 0.001) {
            vec3 lightDir = lPos / dist;
            float angle = dot(-lightDir, normalize(SpotLightDirection(i)));
            if (angle >= SpotLightCutoffAngle(i)) {
                float att = 1.0 / (1.0 + SpotLightLinearDecay(i) * dist + SpotLightQuadraticDecay(i) * dist * dist);
                float sc = SpotLightCutoffAngle(i);
                float spotFactor = clamp((angle - sc) / max(1.0 - sc, 0.001), 0.0, 1.0);
                att *= spotFactor;
                float ndl = max(dot(fragNormal, lightDir), 0.0);
                float ssh = spotShadowFactor(i, WorldPos, wN);
                totalDiffuse += SpotLightColor(i) * vec3(matDiffuse) * (ndl * att * ssh);
                if (ndl > 0.0 && MatShininess > 0.0) {
                    vec3 halfV = normalize(lightDir + camDir);
                    float ndh = max(dot(fragNormal, halfV), 0.0);
                    totalSpec += SpotLightColor(i) * MatSpecularColor * (pow(ndh, MatShininess) * att * ssh);
                }
            }
        }
    }
#endif

    float sky = clamp(wN.y * 0.5 + 0.5, 0.0, 1.0);
    vec3 hemiLo = vec3(0.24, 0.20, 0.16);
    vec3 hemiHi = vec3(0.28, 0.34, 0.42);
    if (ProbeEnabled != 0) {
        hemiLo = ProbeGround;
        hemiHi = ProbeSky;
    }
    vec3 hemi = mix(hemiLo, hemiHi, sky) * vec3(matDiffuse);
    vec3 finalRGB = ambientColor + hemi * 0.45 + totalDiffuse + totalSpec;
    FragColor = min(vec4(finalRGB, matDiffuse.a), vec4(1.0));
    if (Wetness > 0.001) {
        float up = clamp(WorldNormal.y, 0.0, 1.0);
        FragColor.rgb *= mix(1.0, 0.58, Wetness * up);
        FragColor.rgb += (totalSpec * 1.8 + vec3(0.10, 0.11, 0.12)) * Wetness * up;
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
        float ndl = max(dot(normalize(WorldNormal), Lc), 0.0);
        FragColor.rgb += vec3(0.45, 0.72, 0.85) * c * ndl * fade * 0.7;
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

const mbpartVertex = `#include <attributes>
#include <material>
uniform mat4 MVP;
out vec2 FragTexcoord;
void main() {
    vec2 texcoord = VertexTexcoord;
#if MAT_TEXTURES > 0
    if (MatTexFlipY(0)) { texcoord.y = 1.0 - texcoord.y; }
#endif
    FragTexcoord = texcoord;
    gl_Position = MVP * vec4(VertexPosition, 1.0);
}
`

const mbpartFragment = `precision highp float;
in vec2 FragTexcoord;
#include <material>
out vec4 FragColor;
void main() {
    vec4 c = vec4(MatDiffuseColor, MatOpacity);
#if MAT_TEXTURES > 0
    if (MatTexVisible(0)) {
        c *= texture(MatTexture[0], FragTexcoord * MatTexRepeat(0) + MatTexOffset(0));
    }
#endif
    vec2 d = FragTexcoord * 2.0 - 1.0;
    float streak = (1.0 - smoothstep(0.12, 1.0, abs(d.x))) * (1.0 - smoothstep(0.75, 1.0, abs(d.y)));
    c.a *= streak;
    if (c.a < 0.01) { discard; }
    FragColor = c;
}
`

const mbflakeFragment = `precision highp float;
in vec2 FragTexcoord;
#include <material>
out vec4 FragColor;
void main() {
    vec4 c = vec4(MatDiffuseColor, MatOpacity);
#if MAT_TEXTURES > 0
    if (MatTexVisible(0)) {
        c *= texture(MatTexture[0], FragTexcoord * MatTexRepeat(0) + MatTexOffset(0));
    }
#endif
    vec2 d = FragTexcoord * 2.0 - 1.0;
    float r = length(d);
    float flake = (1.0 - smoothstep(0.22, 1.0, r)) * (0.55 + 0.45 * (1.0 - smoothstep(0.0, 0.35, r)));
    c.a *= flake;
    if (c.a < 0.01) { discard; }
    FragColor = c;
}
`

var mbshadowFragment = mbshadowFragmentHead + mbshadowSampleGLSL + mbshadowFragmentTail

const depthVertexSrc = `#version 330 core
layout(location = 0) in vec3 VertexPosition;
uniform mat4 MVP;
void main() {
    gl_Position = MVP * vec4(VertexPosition, 1.0);
}
`

const depthEmptyFragmentSrc = `#version 330 core
void main() {}
`

const depthAlphaVertexSrc = `#version 330 core
layout(location = 0) in vec3 VertexPosition;
layout(location = 3) in vec2 VertexTexcoord;
uniform mat4 MVP;
out vec2 vUV;
void main() {
    vUV = VertexTexcoord;
    gl_Position = MVP * vec4(VertexPosition, 1.0);
}
`

const depthAlphaFragmentSrc = `#version 330 core
in vec2 vUV;
uniform sampler2D Cutout;
uniform float CutoutAlpha;
void main() {
    if (texture(Cutout, vUV).a < CutoutAlpha) { discard; }
}
`

const depthFragmentSrc = `#version 330 core
uniform int MomentMode;
uniform float EvsmC;
out vec4 FragColor;
void main() {
    float d = clamp(gl_FragCoord.z, 0.0, 1.0);
    if (MomentMode == 2) {
        float c = EvsmC;
        if (c < 1.0 || c > 14.0) { c = 8.0; }
        float e = exp(c * d);
        FragColor = vec4(e, e * e, 0.0, 1.0);
    } else if (MomentMode == 3) {
        float d2 = d * d;
        FragColor = vec4(d, d2, d2 * d, d2 * d2);
    } else {
        FragColor = vec4(d, 0.0, 0.0, 1.0);
    }
}
`

const blurVertexSrc = `#version 330 core
layout(location = 0) in vec2 VertexPosition;
out vec2 vUV;
void main() {
    vUV = VertexPosition * 0.5 + 0.5;
    gl_Position = vec4(VertexPosition, 0.0, 1.0);
}
`

const blurFragmentSrc = `#version 330 core
in vec2 vUV;
uniform sampler2D Src;
uniform vec2 Dir;
out vec4 FragColor;
void main() {
    vec2 t = Dir;
    vec4 c = texture(Src, vUV) * 0.227027;
    c += texture(Src, vUV + t) * 0.1945946;
    c += texture(Src, vUV - t) * 0.1945946;
    c += texture(Src, vUV + 2.0 * t) * 0.1216216;
    c += texture(Src, vUV - 2.0 * t) * 0.1216216;
    c += texture(Src, vUV + 3.0 * t) * 0.054054;
    c += texture(Src, vUV - 3.0 * t) * 0.054054;
    FragColor = c;
}
`
