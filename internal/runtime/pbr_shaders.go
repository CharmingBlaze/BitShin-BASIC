package runtime

// mbphysical is G3N metallic-roughness (Cook-Torrance GGX) plus the mbshadow
// atlas / fog uniforms. Shadows multiply the direct term only.
// GLSL 330 only (G3N prepends "#version 330 core"). No 4.5 / compute / tess / SSBO.

const mbphysicalVertex = `#include <attributes>
uniform mat4 ModelViewMatrix;
uniform mat3 NormalMatrix;
uniform mat4 MVP;
uniform mat4 ModelMatrix;
uniform mat4 LightVP[3];
uniform vec3 ShadowTexelWorld;
#include <morphtarget_vertex_declaration>
#include <bones_vertex_declaration>
out vec3 Position;
out vec3 Normal;
out vec3 CamDir;
out vec2 FragTexcoord;
out vec3 WorldPos;
out vec3 WorldNormal;
out vec4 LightSpacePos[3];
void main() {
    Position = vec3(ModelViewMatrix * vec4(VertexPosition, 1.0));
    Normal = normalize(NormalMatrix * VertexNormal);
    CamDir = normalize(-Position.xyz);
    FragTexcoord = VertexTexcoord;
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
    gl_Position = MVP * finalWorld * vec4(vPosition, 1.0);
}
`

const mbphysicalFragmentHead = `precision highp float;

#ifdef HAS_BASECOLORMAP
uniform sampler2D uBaseColorSampler;
#endif
#ifdef HAS_METALROUGHNESSMAP
uniform sampler2D uMetallicRoughnessSampler;
#endif
#ifdef HAS_NORMALMAP
uniform sampler2D uNormalSampler;
#endif
#ifdef HAS_EMISSIVEMAP
uniform sampler2D uEmissiveSampler;
#endif
#ifdef HAS_OCCLUSIONMAP
uniform sampler2D uOcclusionSampler;
#endif
#ifdef HAS_ENVMAP
uniform sampler2D uEnvSampler;
#endif

uniform vec4 Material[3];
#define uBaseColor          Material[0]
#define uEmissiveColor      Material[1]
#define uMetallicFactor     Material[2].x
#define uRoughnessFactor    Material[2].y

#include <lights>

uniform sampler2D ShadowMap;
uniform int ShadowEnabled;
uniform int ShadowCascades;
uniform int ShadowFilter;
uniform int ShadowPCF;
uniform float ShadowBias;
uniform float ShadowLightSize;
uniform vec3 ShadowSplit;
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
uniform int UseIBL;
uniform float IBLIntensity;
uniform vec3 IBLSky;
uniform vec3 IBLGround;
uniform float uAO;
uniform float uOcclusionStrength;
uniform sampler2D WaterCaustic;
uniform int WaterCausticsOn;
uniform float WaterCausticLevel;
uniform vec3 WaterCausticSun;
uniform float WaterCausticMove;

in vec3 Position;
in vec3 Normal;
in vec3 CamDir;
in vec2 FragTexcoord;
in vec3 WorldPos;
in vec3 WorldNormal;
in vec4 LightSpacePos[3];
out vec4 FragColor;

const float M_PI = 3.141592653589793;
const float c_MinRoughness = 0.04;

struct PBRLightInfo {
    float NdotL;
    float NdotV;
    float NdotH;
    float LdotH;
    float VdotH;
};

struct PBRInfo {
    float perceptualRoughness;
    float metalness;
    vec3 reflectance0;
    vec3 reflectance90;
    float alphaRoughness;
    vec3 diffuseColor;
    vec3 specularColor;
};

vec4 SRGBtoLINEAR(vec4 srgbIn) {
    vec3 bLess = step(vec3(0.04045), srgbIn.xyz);
    vec3 linOut = mix(srgbIn.xyz / vec3(12.92), pow((srgbIn.xyz + vec3(0.055)) / vec3(1.055), vec3(2.4)), bLess);
    return vec4(linOut, srgbIn.w);
}

vec3 getNormal() {
    vec3 pos_dx = dFdx(Position);
    vec3 pos_dy = dFdy(Position);
    vec3 tex_dx = dFdx(vec3(FragTexcoord, 0.0));
    vec3 tex_dy = dFdy(vec3(FragTexcoord, 0.0));
    vec3 t = (tex_dy.t * pos_dx - tex_dx.t * pos_dy) / (tex_dx.s * tex_dy.t - tex_dy.s * tex_dx.t);
    vec3 ng = normalize(Normal);
    t = normalize(t - ng * dot(ng, t));
    vec3 b = normalize(cross(ng, t));
    mat3 tbn = mat3(t, b, ng);
#ifdef HAS_NORMALMAP
    vec3 n = texture(uNormalSampler, FragTexcoord).rgb;
    n = normalize(tbn * ((2.0 * n - 1.0) * vec3(1.0, 1.0, 1.0)));
#else
    vec3 n = normalize(tbn[2].xyz);
#endif
    return n;
}

vec3 diffuseBRDF(PBRInfo pbrInputs) {
    return pbrInputs.diffuseColor / M_PI;
}

vec3 specularReflection(PBRInfo pbrInputs, PBRLightInfo pbrLight) {
    return pbrInputs.reflectance0 + (pbrInputs.reflectance90 - pbrInputs.reflectance0) * pow(clamp(1.0 - pbrLight.VdotH, 0.0, 1.0), 5.0);
}

float geometricOcclusion(PBRInfo pbrInputs, PBRLightInfo pbrLight) {
    float NdotL = pbrLight.NdotL;
    float NdotV = pbrLight.NdotV;
    float r = pbrInputs.alphaRoughness;
    float attenuationL = 2.0 * NdotL / (NdotL + sqrt(r * r + (1.0 - r * r) * (NdotL * NdotL)));
    float attenuationV = 2.0 * NdotV / (NdotV + sqrt(r * r + (1.0 - r * r) * (NdotV * NdotV)));
    return attenuationL * attenuationV;
}

float microfacetDistribution(PBRInfo pbrInputs, PBRLightInfo pbrLight) {
    float roughnessSq = pbrInputs.alphaRoughness * pbrInputs.alphaRoughness;
    float f = (pbrLight.NdotH * roughnessSq - pbrLight.NdotH) * pbrLight.NdotH + 1.0;
    return roughnessSq / (M_PI * f * f);
}

vec3 pbrDirect(PBRInfo pbrInputs, vec3 n, vec3 v, vec3 lightColor, vec3 lightDir) {
    vec3 l = normalize(lightDir);
    vec3 h = normalize(l + v);
    float NdotL = clamp(dot(n, l), 0.001, 1.0);
    float NdotV = abs(dot(n, v)) + 0.001;
    float NdotH = clamp(dot(n, h), 0.0, 1.0);
    float LdotH = clamp(dot(l, h), 0.0, 1.0);
    float VdotH = clamp(dot(v, h), 0.0, 1.0);
    PBRLightInfo pbrLight = PBRLightInfo(NdotL, NdotV, NdotH, LdotH, VdotH);
    vec3 F = specularReflection(pbrInputs, pbrLight);
    float G = geometricOcclusion(pbrInputs, pbrLight);
    float D = microfacetDistribution(pbrInputs, pbrLight);
    vec3 diffuseContrib = (1.0 - F) * diffuseBRDF(pbrInputs);
    vec3 specContrib = F * G * D / (4.0 * NdotL * NdotV);
    return NdotL * lightColor * (diffuseContrib + specContrib);
}

#ifdef HAS_ENVMAP
vec2 dirToEquirect(vec3 d) {
    d = normalize(d);
    float u = atan(d.z, d.x) / (2.0 * M_PI) + 0.5;
    float v = asin(clamp(d.y, -1.0, 1.0)) / M_PI + 0.5;
    return vec2(u, v);
}
#endif

vec3 iblTerm(PBRInfo pbrInputs, vec3 n, vec3 v) {
    float sky = clamp(n.y * 0.5 + 0.5, 0.0, 1.0);
    vec3 irr = mix(IBLGround, IBLSky, sky);
    vec3 r = reflect(-v, n);
    float rsky = clamp(r.y * 0.5 + 0.5, 0.0, 1.0);
    vec3 specEnv = mix(IBLGround, IBLSky, rsky);
#ifdef HAS_ENVMAP
    float lod = pbrInputs.perceptualRoughness * 7.0;
    vec3 envN = SRGBtoLINEAR(textureLod(uEnvSampler, dirToEquirect(n), lod + 3.0)).rgb;
    vec3 envR = SRGBtoLINEAR(textureLod(uEnvSampler, dirToEquirect(r), lod)).rgb;
    irr = mix(irr, envN, 0.85);
    specEnv = mix(specEnv, envR, 0.85);
#endif
    float nv = abs(dot(n, v)) + 0.001;
    vec3 Fs = pbrInputs.reflectance0 + (vec3(1.0) - pbrInputs.reflectance0) * pow(1.0 - nv, 5.0);
    float gloss = (1.0 - pbrInputs.perceptualRoughness);
    vec3 diff = irr * pbrInputs.diffuseColor;
    vec3 spec = specEnv * Fs * gloss * gloss;
    return (diff + spec) * IBLIntensity;
}
`

const mbphysicalFragmentTail = `
void main() {
    float perceptualRoughness = uRoughnessFactor;
    float metallic = uMetallicFactor;
#ifdef HAS_METALROUGHNESSMAP
    vec4 mrSample = texture(uMetallicRoughnessSampler, FragTexcoord);
    perceptualRoughness = mrSample.g * perceptualRoughness;
    metallic = mrSample.b * metallic;
#endif
    perceptualRoughness = clamp(perceptualRoughness, c_MinRoughness, 1.0);
    metallic = clamp(metallic, 0.0, 1.0);
    float alphaRoughness = perceptualRoughness * perceptualRoughness;

#ifdef HAS_BASECOLORMAP
    vec4 baseColor = SRGBtoLINEAR(texture(uBaseColorSampler, FragTexcoord)) * uBaseColor;
#else
    vec4 baseColor = uBaseColor;
#endif

    vec3 f0 = vec3(0.04);
    vec3 diffuseColor = baseColor.rgb * (vec3(1.0) - f0);
    diffuseColor *= 1.0 - metallic;
    vec3 specularColor = mix(f0, baseColor.rgb, metallic);
    float reflectance = max(max(specularColor.r, specularColor.g), specularColor.b);
    float reflectance90 = clamp(reflectance * 25.0, 0.0, 1.0);
    PBRInfo pbrInputs = PBRInfo(
        perceptualRoughness,
        metallic,
        specularColor,
        vec3(1.0) * reflectance90,
        alphaRoughness,
        diffuseColor,
        specularColor
    );

    vec3 n = getNormal();
    vec3 v = normalize(CamDir);
    vec3 ambient = vec3(0.0);
    vec3 direct = vec3(0.0);

#if AMB_LIGHTS>0
    for (int i = 0; i < AMB_LIGHTS; i++) {
        ambient += AmbientLightColor[i] * pbrInputs.diffuseColor;
    }
#endif

    vec3 wN = normalize(WorldNormal);
    float skyH = clamp(wN.y * 0.5 + 0.5, 0.0, 1.0);
    vec3 hemi = mix(vec3(0.16, 0.12, 0.09), vec3(0.22, 0.28, 0.40), skyH) * pbrInputs.diffuseColor;
    ambient += hemi * 0.45;
    if (UseIBL != 0) {
        ambient += iblTerm(pbrInputs, n, v);
    }

#if DIR_LIGHTS>0
    for (int i = 0; i < DIR_LIGHTS; i++) {
        vec3 lightDirection = normalize(DirLightPosition(i));
        direct += pbrDirect(pbrInputs, n, v, DirLightColor(i), lightDirection);
    }
#endif

#if POINT_LIGHTS>0
    for (int i = 0; i < POINT_LIGHTS; i++) {
        vec3 lightDirection = PointLightPosition(i) - vec3(Position);
        float lightDistance = length(lightDirection);
        lightDirection = lightDirection / max(lightDistance, 0.0001);
        float attenuation = 1.0 / (1.0 + PointLightLinearDecay(i) * lightDistance +
            PointLightQuadraticDecay(i) * lightDistance * lightDistance);
        direct += pbrDirect(pbrInputs, n, v, PointLightColor(i) * attenuation, lightDirection);
    }
#endif

#if SPOT_LIGHTS>0
    for (int i = 0; i < SPOT_LIGHTS; i++) {
        vec3 lightDirection = SpotLightPosition(i) - vec3(Position);
        float lightDistance = length(lightDirection);
        lightDirection = lightDirection / max(lightDistance, 0.0001);
        float attenuation = 1.0 / (1.0 + SpotLightLinearDecay(i) * lightDistance +
            SpotLightQuadraticDecay(i) * lightDistance * lightDistance);
        float angle = acos(dot(-lightDirection, SpotLightDirection(i)));
        float cutoff = radians(clamp(SpotLightCutoffAngle(i), 0.0, 90.0));
        if (angle < cutoff) {
            float spotFactor = pow(dot(-lightDirection, SpotLightDirection(i)), SpotLightAngularDecay(i));
            direct += pbrDirect(pbrInputs, n, v, SpotLightColor(i) * attenuation * spotFactor, lightDirection);
        }
    }
#endif

    float ao = clamp(uAO, 0.0, 1.0);
#ifdef HAS_OCCLUSIONMAP
    float mapAO = texture(uOcclusionSampler, FragTexcoord).r;
    ao *= mix(1.0, mapAO, clamp(uOcclusionStrength, 0.0, 1.0));
#endif

#ifdef HAS_EMISSIVEMAP
    vec3 emissive = SRGBtoLINEAR(texture(uEmissiveSampler, FragTexcoord)).rgb * vec3(uEmissiveColor);
#else
    vec3 emissive = vec3(uEmissiveColor);
#endif

    float sh = shadowFactor();
    vec3 color = ambient * ao + direct * sh + emissive;
    color = pow(max(color, vec3(0.0)), vec3(1.0 / 2.2));
    FragColor = vec4(color, baseColor.a);
    if (Wetness > 0.001) {
        float up = clamp(WorldNormal.y, 0.0, 1.0);
        FragColor.rgb *= mix(1.0, 0.58, Wetness * up);
        FragColor.rgb += vec3(0.10, 0.11, 0.13) * Wetness * up;
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

var mbphysicalFragment = mbphysicalFragmentHead + mbshadowSampleGLSL + mbphysicalFragmentTail

