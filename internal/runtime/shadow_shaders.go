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
out vec3 VColor;
uniform int MeshWind;
uniform vec3 WindDir;
uniform float WindAmp;
uniform float WindTime;
noperspective out vec2 FragTexcoordAffine;
noperspective out vec3 WorldPosAffine;
` + retroSnapGLSL + `
void main() {
    VColor = VertexColor;
    Position = ModelViewMatrix * vec4(VertexPosition, 1.0);
    Normal = normalize(NormalMatrix * VertexNormal);
    vec2 texcoord = VertexTexcoord;
#if MAT_TEXTURES > 0
    if (MatTexFlipY(0)) { texcoord.y = 1.0 - texcoord.y; }
#endif
    FragTexcoord = texcoord;
    FragTexcoordAffine = texcoord;
    vec3 vPosition = VertexPosition;
    // Tree batches: texcoord x is how far up the tree (squared), y a phase
    // per tree. A slow lean plus a quicker flutter, so stands do not move as one.
    if (MeshWind != 0) {
        float wgt = VertexTexcoord.x;
        float ph = VertexTexcoord.y * 6.2832 + VertexPosition.x * 0.07 + VertexPosition.z * 0.05;
        float lean = sin(WindTime * 0.9 + ph) * 0.6 + 0.4;
        float flutter = sin(WindTime * 3.1 + ph * 2.3) * 0.18;
        vPosition += WindDir * (lean + flutter) * WindAmp * wgt;
    }
    mat4 finalWorld = mat4(1.0);
    #include <morphtarget_vertex>
    #include <bones_vertex>
    vec4 world = ModelMatrix * finalWorld * vec4(vPosition, 1.0);
    WorldPos = world.xyz;
    WorldPosAffine = world.xyz;
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

const mbshadowFragmentHead = `precision highp float;
in vec4 Position;
in vec3 Normal;
in vec2 FragTexcoord;
noperspective in vec2 FragTexcoordAffine;
in vec3 WorldPos;
in vec3 WorldNormal;
in vec4 LightSpacePos[4];
in vec3 VColor;
uniform int MeshVertexColor;
uniform int MeshCutout;
uniform int MeshWind;
uniform int MeshLeafCard;
uniform float MeshGrassFade;
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
uniform int ProbeEnabled;
uniform vec3 ProbeSky;
uniform vec3 ProbeGround;
uniform vec4 WaterClipPlane;
uniform sampler2D WaterCaustic;
uniform int WaterCausticsOn;
uniform float WaterCausticLevel;
uniform vec3 WaterCausticSun;
uniform float WaterCausticMove;
uniform sampler2D SpecMap;
uniform int SpecMapOn;
uniform sampler2D NormMap;
uniform int NormMapOn;
uniform int GammaOut;
uniform sampler2D EmitMap;
uniform int EmitMapOn;
uniform vec3 DirSpec[8];
uniform vec3 DirAmb[8];
uniform vec3 PointSpec[8];
uniform vec3 PointAmb[8];
uniform vec3 SpotSpec[8];
uniform vec3 SpotAmb[8];
out vec4 FragColor;
` + retroFragGLSL + linearGLSL + skyAmbientGLSL + `
`

const mbshadowFragmentTail = shadeLightGLSL + areaLightGLSL + `
vec2 shadeMapUV() {
    vec2 uv = retroUV();
#if MAT_TEXTURES > 0
    uv = uv * MatTexRepeat(0) + MatTexOffset(0);
#endif
    return uv;
}

vec3 shadeSpecMap(vec3 specColor) {
    if (SpecMapOn != 0) {
        specColor *= texture(SpecMap, shadeMapUV()).rgb;
    }
    return specColor;
}

vec3 shadeEmission(vec3 emissive) {
    if (EmitMapOn == 0) { return emissive; }
    vec3 em = texture(EmitMap, shadeMapUV()).rgb;
    if (dot(emissive, emissive) < 1.0e-8) { return em; }
    return emissive * em;
}

vec3 shadeMappedNormal(vec3 n) {
    if (NormMapOn == 0) { return n; }
    vec2 uv = shadeMapUV();
    vec3 dp1 = dFdx(Position.xyz);
    vec3 dp2 = dFdy(Position.xyz);
    vec2 duv1 = dFdx(uv);
    vec2 duv2 = dFdy(uv);
    vec3 N = normalize(n);
    vec3 T = dp1 * duv2.y - dp2 * duv1.y;
    if (dot(T, T) < 1.0e-8) { return N; }
    T = normalize(T);
    vec3 B = normalize(cross(N, T));
    vec3 tn = texture(NormMap, uv).rgb * 2.0 - 1.0;
    return normalize(mat3(T, B, N) * tn);
}

float leafNoise(vec3 p) {
    vec3 i = floor(p), f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    vec2 k = vec2(0.0, 1.0);
    #define LH(o) fract(sin(dot(i + o, vec3(127.1, 311.7, 74.7))) * 43758.5453)
    return mix(mix(mix(LH(k.xxx), LH(k.yxx), f.x), mix(LH(k.xyx), LH(k.yyx), f.x), f.y),
               mix(mix(LH(k.xxy), LH(k.yxy), f.x), mix(LH(k.xyy), LH(k.yyy), f.x), f.y), f.z);
    #undef LH
}

void main() {
    vec2 baseUV = retroUV();
    if (dot(WaterClipPlane.xyz, WaterClipPlane.xyz) > 1e-8) {
        if (dot(WorldPos, WaterClipPlane.xyz) + WaterClipPlane.w < 0.0) {
            discard;
        }
    }
    if (MeshGrassFade > 0.0) {
        // Grass dissolves toward the edge of its field, so no line shows.
        float gone = smoothstep(MeshGrassFade * 0.6, MeshGrassFade, length(Position.xyz));
        if (gone > fract(sin(dot(floor(WorldPos.xz * 7.0), vec2(12.9898, 78.233))) * 43758.5453)) { discard; }
    }
    float leaves = 1.0;
    if (MeshLeafCard != 0) {
        // A card is a round clump of leaves, its pattern seeded by its shade. Past
        // where the noise would shimmer it settles to a plain disc.
        vec2 q = VColor.rg * 2.0 - 1.0;
        vec3 seed = vec3(VColor.rg, VColor.b * 37.0);
        leaves = leafNoise(seed * vec3(15.0, 15.0, 1.0)) * 0.6 + leafNoise(seed * vec3(38.0, 38.0, 1.0)) * 0.4;
        leaves = mix(leaves, 0.62, smoothstep(100.0, 260.0, length(Position.xyz)));
        if (leaves < 0.3 + 0.42 * dot(q, q)) { discard; }
    }
    vec4 texMixed = vec4(1);
    #if MAT_TEXTURES > 0
        bool firstTex = true;
        if (MatTexVisible(0)) {
            vec4 texColor = texture(MatTexture[0], baseUV * MatTexRepeat(0) + MatTexOffset(0));
            texMixed = texColor;
            firstTex = false;
        }
        #if MAT_TEXTURES > 1
            if (MatTexVisible(1)) {
                vec4 texColor = texture(MatTexture[1], baseUV * MatTexRepeat(1) + MatTexOffset(1));
                if (firstTex) { texMixed = texColor; firstTex = false; } else { texMixed = Blend(texMixed, texColor); }
            }
            #if MAT_TEXTURES > 2
                if (MatTexVisible(2)) {
                    vec4 texColor = texture(MatTexture[2], baseUV * MatTexRepeat(2) + MatTexOffset(2));
                    if (firstTex) { texMixed = texColor; } else { texMixed = Blend(texMixed, texColor); }
                }
            #endif
        #endif
    #endif
    #if MAT_TEXTURES > 0
        // MeshCutout: 0 unset (cut, the old rule), 1 cut, 2 never. A blended
        // material is never cut, so a window or a coin's soft edge keeps
        // its partial alpha instead of getting holes.
        if (MeshCutout != 2 && MatOpacity > 0.999) {
            // Each mip averages alpha down, so a far card thinned to specks.
            // Boost by the mip level to keep its silhouette (Castano 2010).
            vec2 ts = vec2(textureSize(MatTexture[0], 0));
            vec2 du = dFdx(baseUV * ts);
            vec2 dv = dFdy(baseUV * ts);
            float lod = 0.5 * log2(max(max(dot(du, du), dot(dv, dv)), 1e-8));
            float a = texMixed.a * (1.0 + max(lod, 0.0) * 0.25);
            if (MeshCutout == 3) {
                // Hard cut, for far cards: a soft edge blends with whatever
                // was drawn first, often the sky behind the land.
                if (a < 0.45) discard;
                texMixed.a = 1.0;
            } else {
                // A one-pixel ramp around the threshold instead of a hard
                // step: blending is on, so the edge comes out antialiased.
                float w = max(fwidth(a), 1e-4);
                float cov = clamp((a - 0.45) / w + 0.5, 0.0, 1.0);
                if (cov <= 0.0) discard;
                texMixed.a = cov;
            }
        }
    #endif
    vec4 matDiffuse = vec4(MatDiffuseColor, MatOpacity) * texMixed;
    vec4 matAmbient = vec4(MatAmbientColor, MatOpacity) * texMixed;
    // A merged level cell carries each object's color in its vertices.
    if (MeshLeafCard != 0) {
        // Gaps between leaves fall darker, so the clump has depth.
        float k = VColor.b * (0.42 + 0.8 * leaves);
        matDiffuse.rgb *= k;
        matAmbient.rgb *= k;
    } else if (MeshVertexColor != 0) {
        matDiffuse.rgb *= VColor;
        matAmbient.rgb *= VColor;
    }
    matDiffuse.rgb = toLin(matDiffuse.rgb);
    matAmbient.rgb = toLin(matAmbient.rgb);
    vec3 fragNormal = normalize(Normal);
    // A card's normal is the crown's, so it lights the same from either side.
    if (!gl_FrontFacing && MeshLeafCard == 0 && MeshGrassFade == 0.0) { fragNormal = -fragNormal; }
    fragNormal = retroFlatNormal(Position.xyz, fragNormal);
    fragNormal = shadeMappedNormal(fragNormal);
    vec3 camDir = normalize(-Position.xyz);
    vec3 totalDiffuse = vec3(0.0);
    vec3 totalSpec = vec3(0.0);
    vec3 specTint = toLin(shadeSpecMap(MatSpecularColor));
    vec3 ambientColor = toLin(shadeEmission(MatEmissiveColor));

#if AMB_LIGHTS>0
    for (int i = 0; i < AMB_LIGHTS; ++i) {
        ambientColor += toLin(AmbientLightColor[i]) * vec3(matAmbient);
    }
#endif

    vec3 wN = retroFlatNormal(WorldPos, normalize(WorldNormal));
    float sunSh = sunShadowFactor(WorldPos, wN, ShadowSunDir);
    bool leafy = LinearLight != 0 && MeshWind != 0;
    // Solid foliage is mottled with a jittered normal, so needles and lumps do not
    // read as plastic. Trunks (brown) stay plain. It fades before the noise would
    // shimmer, and far trees need none of it.
    if (leafy && MeshLeafCard == 0 && MeshGrassFade == 0.0 && matDiffuse.g >= matDiffuse.r) {
        float leafFade = 1.0 - smoothstep(45.0, 160.0, length(Position.xyz));
        if (leafFade > 0.0) {
            float l1 = leafNoise(WorldPos * 2.7);
            float l2 = leafNoise(WorldPos * 1.15 + 9.0);
            matDiffuse.rgb *= mix(1.0, 0.85 + 0.3 * l1, leafFade);
            fragNormal = normalize(fragNormal + (vec3(l1, l2, l1 * l2 * 2.0) - 0.5) * 0.4 * leafFade);
        }
    }

#if DIR_LIGHTS>0
    for (int i = 0; i < DIR_LIGHTS; ++i) {
        vec3 lightDir = normalize(DirLightPosition(i));
        float ndl = max(dot(fragNormal, lightDir), 0.0);
        float sh = (i == 0) ? sunSh : 1.0;
        if (leafy) {
            // Foliage is a volume, not a shell: light wraps round it, passes
            // through it, and a canopy never shades itself fully.
            ndl = max((dot(fragNormal, lightDir) + 0.45) / 1.45, 0.0);
            float back = pow(max(dot(-lightDir, camDir), 0.0), 4.0);
            totalDiffuse += toLin(DirLightColor(i)) * vec3(matDiffuse) * (back * 0.55 * mix(sh, 1.0, 0.5));
            sh = mix(sh, 1.0, 0.2);
        }
        totalDiffuse += toLin(DirLightColor(i)) * vec3(matDiffuse) * (ndl * sh);
        if (i < 8) { ambientColor += toLin(DirAmb[i]) * vec3(matAmbient); }
        if (ndl > 0.0 && MatShininess > 0.0) {
            vec3 halfV = normalize(lightDir + camDir);
            float ndh = max(dot(fragNormal, halfV), 0.0);
            float fres = pow(1.0 - clamp(dot(fragNormal, camDir), 0.0, 1.0), 5.0);
            vec3 ls = toLin(DirLightColor(i));
            if (i < 8) { ls = toLin(DirSpec[i]); }
            totalSpec += ls * specTint * (shadeSpec(ndh, MatShininess) * mix(1.0, 1.85, fres) * sh);
        }
    }
#endif

#if POINT_LIGHTS>0
    for (int i = 0; i < POINT_LIGHTS; ++i) {
        vec3 lPos = PointLightPosition(i) - Position.xyz;
        float dist = length(lPos);
        if (dist > 0.001) {
            vec3 lightDir = lPos / dist;
            float att = shadeAttenuation(dist, PointLightLinearDecay(i), PointLightQuadraticDecay(i), PointLightConstant(i));
            if (att > 0.001) {
                float ndl = max(dot(fragNormal, lightDir), 0.0);
                float psh = pointShadowFactor(i, WorldPos, wN);
                totalDiffuse += toLin(PointLightColor(i)) * vec3(matDiffuse) * (ndl * att * psh);
                if (i < 8) { ambientColor += toLin(PointAmb[i]) * vec3(matAmbient) * att; }
                if (ndl > 0.0 && MatShininess > 0.0) {
                    vec3 halfV = normalize(lightDir + camDir);
                    float ndh = max(dot(fragNormal, halfV), 0.0);
                    float fres = pow(1.0 - clamp(dot(fragNormal, camDir), 0.0, 1.0), 5.0);
                    vec3 ls = toLin(PointLightColor(i));
                    if (i < 8) { ls = toLin(PointSpec[i]); }
                    totalSpec += ls * specTint * (shadeSpec(ndh, MatShininess) * mix(1.0, 1.85, fres) * att * psh);
                }
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
            float att = shadeAttenuation(dist, SpotLightLinearDecay(i), SpotLightQuadraticDecay(i), SpotLightConstant(i));
            att *= shadeSpot(lightDir, SpotLightDirection(i), SpotLightCutoffAngle(i), SpotLightAngularDecay(i), SpotLightConstant(i));
            att *= shadeCookie(i, -lPos, SpotLightDirection(i), SpotLightCutoffAngle(i));
            if (att > 0.001) {
                float ndl = max(dot(fragNormal, lightDir), 0.0);
                float ssh = spotShadowFactor(i, WorldPos, wN);
                totalDiffuse += toLin(SpotLightColor(i)) * vec3(matDiffuse) * (ndl * att * ssh);
                if (i < 8) { ambientColor += toLin(SpotAmb[i]) * vec3(matAmbient) * att; }
                if (ndl > 0.0 && MatShininess > 0.0) {
                    vec3 halfV = normalize(lightDir + camDir);
                    float ndh = max(dot(fragNormal, halfV), 0.0);
                    float fres = pow(1.0 - clamp(dot(fragNormal, camDir), 0.0, 1.0), 5.0);
                    vec3 ls = toLin(SpotLightColor(i));
                    if (i < 8) { ls = toLin(SpotSpec[i]); }
                    totalSpec += ls * specTint * (shadeSpec(ndh, MatShininess) * mix(1.0, 1.85, fres) * att * ssh);
                }
            }
        }
    }
#endif

    if (AreaCount > 0) {
        vec3 wV = normalize(AreaCam - WorldPos);
        for (int i = 0; i < AreaCount; i++) {
            float win = areaWindow(i, WorldPos);
            if (win <= 0.0) { continue; }
            float ff = areaFormFactor(i, WorldPos, wN);
            totalDiffuse += toLin(AreaColor[i]) * vec3(matDiffuse) * (ff * win);
            if (MatShininess > 0.0) {
                vec3 Ls = normalize(areaSpecPoint(i, WorldPos, reflect(-wV, wN)) - WorldPos);
                float ndl = max(dot(wN, Ls), 0.0);
                float ndh = max(dot(wN, normalize(Ls + wV)), 0.0);
                float nearK = clamp(ff * 6.0, 0.0, 1.0);
                totalSpec += toLin(AreaColor[i]) * specTint * (shadeSpec(ndh, MatShininess) * ndl * nearK * win * areaFacing(i, WorldPos));
            }
        }
    }

    float sky = clamp(wN.y * 0.5 + 0.5, 0.0, 1.0);
    vec3 hemiLo = vec3(0.24, 0.20, 0.16);
    vec3 hemiHi = vec3(0.28, 0.34, 0.42);
    if (ProbeEnabled != 0) {
        hemiLo = ProbeGround;
        hemiHi = ProbeSky;
    }
    vec3 hemi = mix(hemiLo, hemiHi, sky) * vec3(matDiffuse);
    float hemiK = 0.45;
    if (LinearLight != 0) {
        hemi = skyAmbient(wN) * vec3(matDiffuse);
        hemiK = 1.0;
        if (leafy) {
            // Sky from above, leaf-green bounce from below, about 2:1 as in
            // Crysis. More ambient would flatten the shape, so it stays modest.
            vec3 up = skyAmbient(normalize(wN + vec3(0.0, 0.8, 0.0)));
            vec3 bounce = vec3(0.09, 0.12, 0.05);
            hemi = mix(bounce, up, clamp(wN.y * 0.5 + 0.5, 0.0, 1.0)) * vec3(matDiffuse) * 1.35;
        }
    }
    vec3 finalRGB = ambientColor + hemi * hemiK + totalDiffuse + totalSpec;
    FragColor = vec4(LinearLight != 0 ? finalRGB : shadeKnee(finalRGB), matDiffuse.a);
    if (GammaOut != 0 && LinearLight == 0) {
        FragColor.rgb = pow(max(FragColor.rgb, vec3(0.0)), vec3(1.0 / 2.2));
    }
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
        FragColor.rgb = fogMix(toLin(fogTint(FogColor, WorldPos)), FragColor.rgb, retroFog(clamp(f, 0.0, 1.0)));
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
` + linearGLSL + `
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
    FragColor = vec4(toLin(c.rgb), c.a);
}
`

const mbsoftFragment = `precision highp float;
in vec2 FragTexcoord;
#include <material>
out vec4 FragColor;
` + linearGLSL + `
void main() {
    vec4 c = vec4(MatDiffuseColor, MatOpacity);
#if MAT_TEXTURES > 0
    if (MatTexVisible(0)) {
        c *= texture(MatTexture[0], FragTexcoord * MatTexRepeat(0) + MatTexOffset(0));
    }
#endif
    vec2 d = FragTexcoord * 2.0 - 1.0;
    float a = 1.0 - smoothstep(0.05, 1.0, dot(d, d));
    c.a *= a;
    if (c.a < 0.01) { discard; }
    FragColor = vec4(toLin(c.rgb), c.a);
}
`

const mbflakeFragment = `precision highp float;
in vec2 FragTexcoord;
#include <material>
out vec4 FragColor;
` + linearGLSL + `
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
    FragColor = vec4(toLin(c.rgb), c.a);
}
`

var mbshadowFragment = mbshadowFragmentHead + mbshadowSampleGLSL + mbshadowFragmentTail

// G3N enables a geometry's attributes once, on its first draw, by asking the
// program then in use. A mesh first drawn here would lose its normals,
// colors, and texcoords for good, so the depth pass reads all four. Keep is
// always 0; as a uniform the compiler cannot drop what it multiplies.
const depthAttribsGLSL = `layout(location = 0) in vec3 VertexPosition;
layout(location = 1) in vec3 VertexNormal;
layout(location = 2) in vec3 VertexColor;
layout(location = 3) in vec2 VertexTexcoord;
uniform mat4 MVP;
uniform float Keep;
uniform int MeshWind;
uniform vec3 WindDir;
uniform float WindAmp;
uniform float WindTime;
vec3 depthPosition() {
    vec3 p = VertexPosition;
    if (MeshWind != 0) {
        float wgt = VertexTexcoord.x;
        float ph = VertexTexcoord.y * 6.2832 + VertexPosition.x * 0.07 + VertexPosition.z * 0.05;
        float lean = sin(WindTime * 0.9 + ph) * 0.6 + 0.4;
        float flutter = sin(WindTime * 3.1 + ph * 2.3) * 0.18;
        p += WindDir * (lean + flutter) * WindAmp * wgt;
    }
    return p + Keep * (VertexNormal + VertexColor);
}
`

// depthVertexSrc is the lean depth shader: position only. Fetching all
// four attributes for every caster in every cascade cost 4-5 ms a frame.
const depthVertexSrc = `#version 330 core
layout(location = 0) in vec3 VertexPosition;
uniform mat4 MVP;
void main() {
    gl_Position = MVP * vec4(VertexPosition, 1.0);
}
`

// depthFullVertexSrc is used for a geometry's first shadow draw (so G3N
// binds all its attributes) and for swaying trees.
const depthFullVertexSrc = `#version 330 core
` + depthAttribsGLSL + `
void main() {
    gl_Position = MVP * vec4(depthPosition(), 1.0);
}
`

const depthEmptyFragmentSrc = `#version 330 core
void main() {}
`

const depthAlphaVertexSrc = `#version 330 core
` + depthAttribsGLSL + `
out vec2 vUV;
void main() {
    vUV = VertexTexcoord;
    gl_Position = MVP * vec4(depthPosition(), 1.0);
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
