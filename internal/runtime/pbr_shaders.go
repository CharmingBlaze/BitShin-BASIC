package runtime

// mbphysical is G3N metallic-roughness (Cook-Torrance GGX) plus the mbshadow
// atlas / fog uniforms. Shadows multiply the direct term only.
// GLSL 330 only (G3N prepends "#version 330 core"). No 4.5 / compute / tess / SSBO.

const mbphysicalVertex = `#include <attributes>
uniform mat4 ModelViewMatrix;
uniform mat3 NormalMatrix;
uniform mat4 MVP;
uniform mat4 ModelMatrix;
uniform mat4 LightVP[4];
uniform vec4 ShadowTexelWorld;
#include <morphtarget_vertex_declaration>
#include <bones_vertex_declaration>
out vec3 Position;
out vec3 Normal;
out vec3 CamDir;
out vec2 FragTexcoord;
out vec3 WorldPos;
out vec3 WorldNormal;
out vec4 LightSpacePos[4];
noperspective out vec2 FragTexcoordAffine;
` + retroSnapGLSL + `
void main() {
    Position = vec3(ModelViewMatrix * vec4(VertexPosition, 1.0));
    Normal = normalize(NormalMatrix * VertexNormal);
    CamDir = normalize(-Position.xyz);
    FragTexcoord = VertexTexcoord;
    FragTexcoordAffine = VertexTexcoord;
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
    gl_Position = retroSnapClip(MVP * finalWorld * vec4(vPosition, 1.0));
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
uniform int UseIBL;
uniform float IBLIntensity;
uniform vec3 IBLSky;
uniform vec3 IBLGround;
` + skyVolumeGLSL + `
uniform samplerCube uEnvCube;
uniform int HasEnvCube;
uniform vec3 CamWorldPos;
uniform float uAO;
uniform int uPostGrade;
uniform float uOcclusionStrength;
uniform int uShading;      // 0 PBR, 1 unlit, 2 toon, 3 cel, 4 flat, 5 low-poly, 6 painted
uniform float uClearcoat;
uniform float uCoatRough;
uniform float uSubsurface;
uniform vec3 uSubColor;
uniform int uHasHeight;
uniform float uHeightScale;
uniform sampler2D uHeightMap;
uniform float uToonBands;  // diffuse steps
uniform float uToonRim;    // rim light strength
uniform float uToonSpec;   // hard highlight strength
uniform int uNoFog;
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
in vec4 LightSpacePos[4];
noperspective in vec2 FragTexcoordAffine;
out vec4 FragColor;
` + retroFragGLSL + `

` + linearGLSL + `
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

// The UV after parallax, and the world normal the sky lighting reads.
vec2 gUV;
vec3 iblN;

// Parallax occlusion: step along the view through the height map until below it,
// then interpolate the last step. Depth with no geometry. It fades with distance,
// where it could only shimmer and cost.
vec2 parallaxUV(vec2 uv) {
    vec3 pdx = dFdx(Position), pdy = dFdy(Position);
    vec2 dx = dFdx(uv), dy = dFdy(uv);
    vec3 ng = normalize(Normal);
    vec3 t = (dy.t * pdx - dx.t * pdy) / (dx.s * dy.t - dy.s * dx.t);
    t = normalize(t - ng * dot(ng, t));
    vec3 b = cross(ng, t);
    vec3 vd = normalize(CamDir);
    vec3 vts = vec3(dot(vd, t), dot(vd, b), dot(vd, ng));
    float fade = 1.0 - smoothstep(25.0, 60.0, length(Position));
    if (fade <= 0.0) { return uv; }
    float steps = mix(24.0, 8.0, clamp(vts.z, 0.0, 1.0));
    float dh = 1.0 / steps;
    vec2 duv = vts.xy / max(vts.z, 0.15) * uHeightScale * fade * dh;
    vec2 cur = uv;
    float depth = 0.0;
    float h = 1.0 - textureGrad(uHeightMap, cur, dx, dy).r;
    for (int i = 0; i < 24 && depth < h; i++) {
        cur -= duv;
        depth += dh;
        h = 1.0 - textureGrad(uHeightMap, cur, dx, dy).r;
    }
    float after = h - depth;
    float before = (1.0 - textureGrad(uHeightMap, cur + duv, dx, dy).r) - depth + dh;
    float k = after / min(after - before, -1e-4);
    return mix(cur, cur + duv, clamp(k, 0.0, 1.0));
}

// paintNoise is smooth noise for brush strokes.
float paintNoise(vec3 p) {
    vec3 i = floor(p), f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    vec2 k = vec2(0.0, 1.0);
    #define PH(o) fract(sin(dot(i + o, vec3(127.1, 311.7, 74.7))) * 43758.5453)
    return mix(mix(mix(PH(k.xxx), PH(k.yxx), f.x), mix(PH(k.xyx), PH(k.yyx), f.x), f.y),
               mix(mix(PH(k.xxy), PH(k.yxy), f.x), mix(PH(k.xyy), PH(k.yyy), f.x), f.y), f.z);
    #undef PH
}

vec3 getNormal() {
    vec3 pos_dx = dFdx(Position);
    vec3 pos_dy = dFdy(Position);
    vec2 nuv = gUV;
    vec3 tex_dx = dFdx(vec3(nuv, 0.0));
    vec3 tex_dy = dFdy(vec3(nuv, 0.0));
    vec3 t = (tex_dy.t * pos_dx - tex_dx.t * pos_dy) / (tex_dx.s * tex_dy.t - tex_dy.s * tex_dx.t);
    vec3 ng = retroFlatNormal(Position, normalize(Normal));
    t = normalize(t - ng * dot(ng, t));
    vec3 b = normalize(cross(ng, t));
    mat3 tbn = mat3(t, b, ng);
#ifdef HAS_NORMALMAP
    vec3 n = texture(uNormalSampler, nuv).rgb;
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

// toonDirect steps the diffuse into bands with a soft edge between them and
// keeps one hard highlight, so lights still read but surfaces look painted.
vec3 toonDirect(PBRInfo pbrInputs, vec3 n, vec3 v, vec3 lightColor, vec3 l) {
    float ndl = max(dot(n, l), 0.0);
    float bands = max(uToonBands, 1.0);
    float x = ndl * bands;
    // Cel steps with no soft edge beyond antialiasing.
    float edge = fwidth(x) + (uShading == 3 ? 0.0 : 0.02);
    float stepped = (floor(x) + smoothstep(0.5 - edge, 0.5 + edge, fract(x))) / bands;
    vec3 h = normalize(l + v);
    float gloss = mix(256.0, 8.0, pbrInputs.perceptualRoughness);
    float s = pow(max(dot(n, h), 0.0), gloss);
    float spec = smoothstep(0.45, 0.55, s) * uToonSpec * step(0.001, ndl);
    return lightColor * (pbrInputs.diffuseColor * stepped + vec3(spec) * mix(vec3(1.0), pbrInputs.specularColor * 4.0, pbrInputs.metalness));
}

vec3 pbrDirect(PBRInfo pbrInputs, vec3 n, vec3 v, vec3 lightColor, vec3 lightDir) {
    vec3 l = normalize(lightDir);
    if (uShading == 2 || uShading == 3) {
        return toonDirect(pbrInputs, n, v, lightColor, l);
    }
    vec3 h = normalize(l + v);
    float NdotL = clamp(dot(n, l), 0.001, 1.0);
    float NdotV = min(abs(dot(n, v)) + 0.001, 1.0);
    float NdotH = clamp(dot(n, h), 0.0, 1.0);
    float LdotH = clamp(dot(l, h), 0.0, 1.0);
    float VdotH = clamp(dot(v, h), 0.0, 1.0);
    PBRLightInfo pbrLight = PBRLightInfo(NdotL, NdotV, NdotH, LdotH, VdotH);
    vec3 F = specularReflection(pbrInputs, pbrLight);
    float G = geometricOcclusion(pbrInputs, pbrLight);
    float D = microfacetDistribution(pbrInputs, pbrLight);
    vec3 diffuseContrib = (1.0 - F) * diffuseBRDF(pbrInputs);
    vec3 specContrib = min(F * G * D / (4.0 * NdotL * NdotV), vec3(6.0));
    vec3 lit = NdotL * lightColor * (diffuseContrib + specContrib);
    if (uSubsurface > 0.0) {
        // Light wraps past the shadow line and shows through thin parts from behind,
        // tinted by what it passed through.
        float nl = dot(n, l);
        float wrap = max((nl + 0.6) / 1.6, 0.0) - max(nl, 0.0);
        float back = pow(max(dot(v, -normalize(l + n * 0.4)), 0.0), 3.0);
        lit += lightColor * uSubsurface * toLin(uSubColor) * pbrInputs.diffuseColor * (wrap + back * 0.8) / M_PI * 2.0;
    }
    if (uClearcoat > 0.0) {
        // A second, smooth layer on top, with its own highlight. What it reflects
        // does not reach the layer below.
        float a = max(uCoatRough * uCoatRough, 0.002);
        float a2 = a * a;
        float d = NdotH * NdotH * (a2 - 1.0) + 1.0;
        float Dc = a2 / (M_PI * d * d);
        float Fc = (0.04 + 0.96 * pow(clamp(1.0 - VdotH, 0.0, 1.0), 5.0)) * uClearcoat;
        float Vc = 0.25 / max(VdotH * VdotH, 0.05);
        lit = lit * (1.0 - Fc) + lightColor * min(Dc * Vc * Fc, 8.0) * NdotL;
    }
    return lit;
}

// pbrSpecOnly is pbrDirect's highlight alone, for lights whose diffuse is
// worked out another way (area lights).
vec3 pbrSpecOnly(PBRInfo pbrInputs, vec3 n, vec3 v, vec3 lightColor, vec3 l) {
    vec3 h = normalize(l + v);
    float NdotL = clamp(dot(n, l), 0.001, 1.0);
    float NdotV = min(abs(dot(n, v)) + 0.001, 1.0);
    PBRLightInfo pbrLight = PBRLightInfo(NdotL, NdotV, clamp(dot(n, h), 0.0, 1.0), clamp(dot(l, h), 0.0, 1.0), clamp(dot(v, h), 0.0, 1.0));
    vec3 F = specularReflection(pbrInputs, pbrLight);
    float G = geometricOcclusion(pbrInputs, pbrLight);
    float D = microfacetDistribution(pbrInputs, pbrLight);
    return NdotL * lightColor * min(F * G * D / (4.0 * NdotL * NdotV), vec3(6.0));
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
    vec3 irr = toLin(mix(IBLGround, IBLSky, sky));
    vec3 r = reflect(-v, n);
    float rsky = clamp(r.y * 0.5 + 0.5, 0.0, 1.0);
    vec3 specEnv = toLin(mix(IBLGround, IBLSky, rsky));
#ifdef HAS_ENVMAP
    float lod = pbrInputs.perceptualRoughness * 7.0;
    vec3 envN = SRGBtoLINEAR(textureLod(uEnvSampler, dirToEquirect(n), lod + 3.0)).rgb;
    vec3 envR = SRGBtoLINEAR(textureLod(uEnvSampler, dirToEquirect(r), lod)).rgb;
    irr = mix(irr, envN, 0.85);
    specEnv = mix(specEnv, envR, 0.85);
#endif
    if (HasEnvCube != 0) {
        vec3 wn = iblN;
        vec3 wv = normalize(CamWorldPos - WorldPos);
        vec3 wr = reflect(-wv, wn);
        float cubeLod = pbrInputs.perceptualRoughness * 5.0;
        vec3 cubeN = textureLod(uEnvCube, wn, 5.0).rgb;
        vec3 cubeR = textureLod(uEnvCube, wr, cubeLod).rgb;
        if (HasEnvCube == 1) {
            cubeN = toLin(cubeN);
            cubeR = toLin(cubeR);
        }
        if (HasEnvCube != 2) {
            // A sky picture knows nothing of the ground below it. Without
            // this a polished ball mirrors a bright false horizon.
            // The edge blurs with roughness, as the reflection itself does.
            float soft = mix(0.04, 0.9, pbrInputs.perceptualRoughness);
            float below = 1.0 - smoothstep(-0.12 - soft, soft * 0.4, wr.y);
            vec3 floorCol = mix(toLin(IBLGround), cubeN, 0.45);
            cubeR = mix(cubeR, floorCol, below);
        }
        irr = mix(irr, cubeN, 0.85);
        specEnv = mix(specEnv, cubeR, 0.85);
        float seen = skyVisibility(WorldPos, wn);
        irr *= seen;
        specEnv *= mix(seen, 1.0, 0.25);
    }
    // Clamped: facing the camera exactly, nv was 1.001 and pow() of a
    // negative base is NaN in GLSL. That was the black dot in the middle of
    // every polished sphere, and a float post buffer smeared it into a block.
    float nv = min(abs(dot(n, v)) + 0.001, 1.0);
    vec4 c0 = vec4(-1.0, -0.0275, -0.572, 0.022);
    vec4 c1 = vec4(1.0, 0.0425, 1.04, -0.04);
    vec4 ruf = pbrInputs.perceptualRoughness * c0 + c1;
    float a004 = min(ruf.x * ruf.x, exp2(-9.28 * nv)) * ruf.x + ruf.y;
    vec2 dfg = vec2(-1.04, 1.04) * a004 + ruf.zw;
    vec3 diff = irr * pbrInputs.diffuseColor;
    // F0, not a Schlick term: dfg already holds the rise toward grazing, and
    // counting it twice ringed every polished sphere in white.
    vec3 spec = specEnv * (pbrInputs.reflectance0 * dfg.x + dfg.y);
    if (uClearcoat > 0.0) {
        float Fc = (0.04 + 0.96 * pow(1.0 - nv, 5.0)) * uClearcoat;
        vec3 coat = specEnv;
        if (HasEnvCube != 0) {
            coat = textureLod(uEnvCube, reflect(-normalize(CamWorldPos - WorldPos), iblN), uCoatRough * 5.0).rgb;
            if (HasEnvCube == 1) { coat = toLin(coat); }
        }
        return ((diff + spec) * (1.0 - Fc) + coat * Fc) * IBLIntensity;
    }
    return (diff + spec) * IBLIntensity;
}
`

const mbphysicalFragmentTail = shadeLightGLSL + areaLightGLSL + `
uniform vec4 WaterClipPlane;
void main() {
    if (dot(WaterClipPlane.xyz, WaterClipPlane.xyz) > 1e-8) {
        if (dot(WorldPos, WaterClipPlane.xyz) + WaterClipPlane.w < 0.0) {
            discard;
        }
    }
    vec2 baseUV = retroUV();
    if (uHasHeight != 0) {
        baseUV = parallaxUV(baseUV);
    }
    gUV = baseUV;
    float perceptualRoughness = uRoughnessFactor;
    float metallic = uMetallicFactor;
#ifdef HAS_METALROUGHNESSMAP
    vec4 mrSample = texture(uMetallicRoughnessSampler, baseUV);
    perceptualRoughness = mrSample.g * perceptualRoughness;
    metallic = mrSample.b * metallic;
#endif
    perceptualRoughness = clamp(perceptualRoughness, c_MinRoughness, 1.0);
    metallic = clamp(metallic, 0.0, 1.0);
    float alphaRoughness = perceptualRoughness * perceptualRoughness;

#ifdef HAS_BASECOLORMAP
    vec4 baseColor = SRGBtoLINEAR(texture(uBaseColorSampler, baseUV)) * vec4(toLin(uBaseColor.rgb), uBaseColor.a);
#else
    vec4 baseColor = vec4(toLin(uBaseColor.rgb), uBaseColor.a);
#endif

    bool faceted = uShading == 4 || uShading == 5;
    vec3 faceW = vec3(0.0, 1.0, 0.0);
    if (faceted) {
        faceW = normalize(cross(dFdx(WorldPos), dFdy(WorldPos)));
        if (dot(faceW, WorldNormal) < 0.0) { faceW = -faceW; }
    }
    if (uShading == 5) {
        // Low-poly: matte, and each face a slightly different shade of its color.
        float h = fract(sin(dot(floor(faceW * 40.0), vec3(12.9898, 78.233, 37.719))) * 43758.5453);
        baseColor.rgb *= 0.86 + 0.28 * h;
        perceptualRoughness = max(perceptualRoughness, 0.7);
        alphaRoughness = perceptualRoughness * perceptualRoughness;
    }
    vec3 f0 = vec3(0.04);
    if (uShading == 2 || uShading == 3) {
        // Toon keeps the albedo flat; metal only tints the highlight.
        f0 = vec3(0.0);
    }
    vec3 diffuseColor = baseColor.rgb * (vec3(1.0) - f0);
    if (uShading != 2 && uShading != 3) {
        diffuseColor *= 1.0 - metallic;
    }
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
    if (faceted) {
        vec3 faceV = normalize(cross(dFdx(Position), dFdy(Position)));
        n = dot(faceV, n) < 0.0 ? -faceV : faceV;
    }
    if (uShading == 6) {
        // Brushed: the surface leans a little with each stroke, close up. At full
        // strength and at any distance this rippled a whole floor into camouflage.
        vec3 sp = WorldPos * vec3(2.3, 6.1, 2.3);
        float lean = 0.22 * (1.0 - smoothstep(6.0, 22.0, length(Position)));
        n = normalize(n + (vec3(paintNoise(sp), paintNoise(sp + 7.1), paintNoise(sp + 13.7)) - 0.5) * lean);
    }
    iblN = faceted ? faceW : normalize(WorldNormal);
    vec3 v = normalize(CamDir);
    vec3 ambient = vec3(0.0);
    vec3 direct = vec3(0.0);

#if AMB_LIGHTS>0
    for (int i = 0; i < AMB_LIGHTS; i++) {
        ambient += toLin(AmbientLightColor[i]) * pbrInputs.diffuseColor;
    }
#endif

    vec3 wN = faceted ? faceW : retroFlatNormal(WorldPos, normalize(WorldNormal));
    if (UseIBL != 0) {
        ambient += iblTerm(pbrInputs, n, v);
    } else {
        float skyH = clamp(wN.y * 0.5 + 0.5, 0.0, 1.0);
        ambient += toLin(mix(IBLGround, IBLSky, skyH)) * pbrInputs.diffuseColor * 0.55;
    }

    float sunOcc = 1.0;
#if DIR_LIGHTS>0
    for (int i = 0; i < DIR_LIGHTS; i++) {
        vec3 lightDirection = normalize(DirLightPosition(i));
        float sh = 1.0;
        if (i == 0) {
            sh = sunShadowFactor(WorldPos, wN, ShadowSunDir);
            sunOcc = sh;
        }
        // The BRDF divides diffuse by pi, so a light of 1 lit a white wall a third as
        // bright as the same light on a lit (non-PBR) mesh.
        direct += pbrDirect(pbrInputs, n, v, toLin(DirLightColor(i)) * (LinearLight != 0 ? M_PI : 1.0), lightDirection) * sh;
    }
#endif

#if POINT_LIGHTS>0
    for (int i = 0; i < POINT_LIGHTS; i++) {
        vec3 lightDirection = PointLightPosition(i) - vec3(Position);
        float lightDistance = length(lightDirection);
        lightDirection = lightDirection / max(lightDistance, 0.0001);
        float attenuation = shadeAttenuation(lightDistance, PointLightLinearDecay(i), PointLightQuadraticDecay(i), PointLightConstant(i));
        float psh = pointShadowFactor(i, WorldPos, wN); // world space, as it offsets a world point
        // Physical falloff (constant below 0) is in candela, which already means this.
        float pscale = (LinearLight != 0 && PointLightConstant(i) >= -0.5) ? M_PI : 1.0;
        direct += pbrDirect(pbrInputs, n, v, toLin(PointLightColor(i)) * attenuation * pscale, lightDirection) * psh;
    }
#endif

#if SPOT_LIGHTS>0
    for (int i = 0; i < SPOT_LIGHTS; i++) {
        vec3 lightDirection = SpotLightPosition(i) - vec3(Position);
        float lightDistance = length(lightDirection);
        lightDirection = lightDirection / max(lightDistance, 0.0001);
        float attenuation = shadeAttenuation(lightDistance, SpotLightLinearDecay(i), SpotLightQuadraticDecay(i), SpotLightConstant(i));
        attenuation *= shadeSpot(lightDirection, SpotLightDirection(i), SpotLightCutoffAngle(i), SpotLightAngularDecay(i), SpotLightConstant(i));
        attenuation *= shadeCookie(i, -lightDirection * lightDistance, SpotLightDirection(i), SpotLightCutoffAngle(i));
        if (attenuation > 0.001) {
            float ssh = spotShadowFactor(i, WorldPos, wN);
            float sscale = (LinearLight != 0 && SpotLightConstant(i) >= -0.5) ? M_PI : 1.0;
            direct += pbrDirect(pbrInputs, n, v, toLin(SpotLightColor(i)) * attenuation * sscale, lightDirection) * ssh;
        }
    }
#endif

    if (AreaCount > 0) {
        vec3 wV = normalize(AreaCam - WorldPos);
        for (int i = 0; i < AreaCount; i++) {
            float win = areaWindow(i, WorldPos);
            if (win <= 0.0) { continue; }
            direct += toLin(AreaColor[i]) * pbrInputs.diffuseColor * (areaFormFactor(i, WorldPos, wN) * win);
            // Representative point: the light seen as a small sphere at
            // the spot the mirror ray reaches, its lobe widened by its size.
            vec3 sp = areaSpecPoint(i, WorldPos, reflect(-wV, wN));
            vec3 Ls = sp - WorldPos;
            float ds = max(length(Ls), 1e-3);
            Ls /= ds;
            float size = length(AreaRight[i]) + length(AreaUp[i]);
            float a = pbrInputs.alphaRoughness;
            float a2 = clamp(a + size / (2.0 * ds), 0.0, 1.0);
            float norm = (a / a2) * (a / a2);
            float area = 4.0 * length(AreaRight[i]) * length(AreaUp[i]);
            float solid = min(area / (ds * ds), 6.2832);
            direct += pbrSpecOnly(pbrInputs, wN, wV, toLin(AreaColor[i]) * (solid * norm * win * areaFacing(i, WorldPos)), Ls);
        }
    }

    float ao = clamp(uAO, 0.0, 1.0);
#ifdef HAS_OCCLUSIONMAP
    float mapAO = texture(uOcclusionSampler, baseUV).r;
    ao *= mix(1.0, mapAO, clamp(uOcclusionStrength, 0.0, 1.0));
#endif

#ifdef HAS_EMISSIVEMAP
    vec3 emissive = SRGBtoLINEAR(texture(uEmissiveSampler, baseUV)).rgb * toLin(vec3(uEmissiveColor));
#else
    vec3 emissive = toLin(vec3(uEmissiveColor));
#endif

    if (uShading == 2 || uShading == 3) {
        // Flat fill instead of the IBL gradient, and a rim against the sky.
        float sky = step(0.0, wN.y);
        // Bright enough that the darkest band still shows the albedo.
        ambient = toLin(mix(IBLGround, IBLSky, 0.55 + 0.3 * sky)) * pbrInputs.diffuseColor * 1.1;
#if AMB_LIGHTS>0
        for (int i = 0; i < AMB_LIGHTS; i++) {
            ambient += toLin(AmbientLightColor[i]) * pbrInputs.diffuseColor;
        }
#endif
        float rim = 1.0 - max(dot(n, v), 0.0);
        rim = smoothstep(0.62, 0.72, rim) * uToonRim * (0.4 + 0.6 * sunOcc);
        ambient += toLin(IBLSky) * rim;
    }
    vec3 color = ambient * ao + direct + emissive + ShadowColor * ((1.0 - sunOcc) * pbrInputs.diffuseColor * 0.35);
    if (uShading == 3) {
        // Ink along the silhouette, where the surface turns away and is curving. A
        // flat floor seen at a low angle is turned away too, but it has no edge.
        float ink = 1.0 - smoothstep(0.2, 0.3, abs(dot(n, v)));
        ink *= smoothstep(0.004, 0.03, length(fwidth(n)));
        color = mix(color, baseColor.rgb * 0.05, ink);
    }
    if (uShading == 6) {
        // Paint: a few broken tones, cool in the shadows and warm in the light, with
        // brush strokes breaking up where one tone meets the next.
        vec3 bp = WorldPos;
        float brush = paintNoise(bp * vec3(7.0, 22.0, 7.0)) * 0.55 + paintNoise(bp * vec3(20.0, 8.0, 17.0) + 4.0) * 0.45;
        float lum = max(dot(color, vec3(0.2126, 0.7152, 0.0722)), 0.0005);
        float soft = sqrt(lum);
        // Strokes break the edge between two tones, so they need one: where brightness
        // is changing, close up. A flat floor sitting on a tone's boundary would
        // otherwise flip back and forth into camouflage.
        float near = 1.0 - smoothstep(12.0, 45.0, length(Position));
        float changing = smoothstep(0.0004, 0.004, fwidth(soft));
        float tone = floor(soft * 6.0 + (brush - 0.5) * 0.3 * near * changing + 0.5) / 6.0;
        tone = mix(soft, tone, 0.45);
        color *= (tone * tone + 0.002) / lum;
        color *= mix(vec3(0.84, 0.9, 1.06), vec3(1.06, 1.0, 0.9), smoothstep(0.2, 0.65, tone));
        color *= 1.0 + (brush - 0.5) * 0.16 * near;
    }
    if (uShading == 1) {
        color = baseColor.rgb + emissive;
    }
    if (uPostGrade == 0) {
        color = pow(max(color, vec3(0.0)), vec3(1.0 / 2.2));
    }
    FragColor = vec4(color, baseColor.a);
    if (Wetness > 0.001 && uShading != 1) {
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
    if (FogMode != 0 && uNoFog == 0) {
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
        FragColor.rgb = fogMix(toLin(fogTint(FogColor, WorldPos)), FragColor.rgb, retroFog(clamp(f, 0.0, 1.0)));
    }
}
`

var mbphysicalFragment = mbphysicalFragmentHead + mbshadowSampleGLSL + mbphysicalFragmentTail
