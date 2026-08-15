package runtime

// Shared GLSL 330 shadow sampling for mbshadow / mbphysical / mbterrain.
// EVSM/MSM: one bilinear fetch of pre-filtered moments (no 5x5 in lighting).
// PCF is an unrotated grid (no Vogel/IGN rotation — no TAA to hide sandy noise).

const mbshadowSampleGLSL = `
uniform sampler2D ShadowMapDyn;
uniform int ShadowCacheSplit;
uniform float ShadowNormalBias;
uniform vec3 ShadowTexelWorld;
uniform mat4 LightVP[3];
uniform float EvsmC;

float gradientNoise(vec2 screenPos) {
    vec3 magic = vec3(0.06711056, 0.00583715, 52.9829189);
    return fract(magic.z * fract(dot(screenPos, magic.xy)));
}

float ignoise(vec2 p) {
    return gradientNoise(p);
}

float sampleDepth(sampler2D map, vec2 uv) {
    return texture(map, uv).r;
}

vec4 sampleMoments(sampler2D map, vec2 uv) {
    return texture(map, uv);
}

vec2 tileUV(int tile, vec2 proj) {
    int cols = AtlasCols;
    int rows = AtlasRows;
    if (cols < 1) { cols = 1; }
    if (rows < 1) { rows = 1; }
    int col = tile - (tile / cols) * cols;
    int row = tile / cols;
    float pad = 0.012;
    proj = clamp(proj, pad, 1.0 - pad);
    return vec2((float(col) + proj.x) / float(cols), (float(row) + proj.y) / float(rows));
}

float pcfAt(sampler2D map, vec2 uv, float z, float bias, int k, vec2 texel) {
    float s = 0.0;
    int r = clamp(k / 2, 0, 4);
    float n = 0.0;
    for (int x = -r; x <= r; x++) {
        for (int y = -r; y <= r; y++) {
            float d = sampleDepth(map, uv + vec2(float(x), float(y)) * texel);
            s += (z - bias > d) ? 0.0 : 1.0;
            n += 1.0;
        }
    }
    if (n < 1.0) { return 1.0; }
    return s / n;
}

float pcssAt(sampler2D map, vec2 uv, float z, float bias, vec2 texel) {
    float search = max(ShadowLightSize * 12.0 * texel.x, texel.x * 3.0);
    float blk = 0.0;
    float cnt = 0.0;
    for (int x = -4; x <= 4; x++) {
        for (int y = -4; y <= 4; y++) {
            float d = sampleDepth(map, uv + vec2(float(x), float(y)) * search);
            if (d < z - bias) {
                blk += d;
                cnt += 1.0;
            }
        }
    }
    if (cnt < 1.0) { return 1.0; }
    blk /= cnt;
    float penumbra = ((z - blk) * ShadowLightSize) / max(blk, 0.0008);
    int k = int(clamp(penumbra * 48.0, 3.0, 9.0));
    if ((k - (k / 2) * 2) == 0) { k = k + 1; }
    return pcfAt(map, uv, z, bias, k, texel);
}

float evsmAt(sampler2D map, vec2 uv, float z, float bias) {
    float c = EvsmC;
    if (c < 1.0) { c = 40.0; }
    vec4 m = sampleMoments(map, uv);
    float m1 = m.x;
    float m2 = m.y;
    float ez = exp(c * clamp(z, 0.0, 1.0));
    float var = max(m2 - m1 * m1, 0.0002);
    float md = ez - m1;
    float p = var / (var + md * md);
    if (ez <= m1 + bias * c) { return 1.0; }
    return clamp(p, 0.15, 1.0);
}

float msmAt(sampler2D map, vec2 uv, float z, float bias) {
    vec4 b = sampleMoments(map, uv);
    float b1 = b.x;
    float b2 = b.y;
    float b3 = b.z;
    float b4 = b.w;
    float z3 = z * z * z;
    float mu = b1;
    float var = max(b2 - b1 * b1, 0.00015);
    float sk = (b3 - 3.0 * b1 * b2 + 2.0 * b1 * b1 * b1) / max(pow(var, 1.5), 0.00001);
    float ku = (b4 - 4.0 * b1 * b3 + 6.0 * b1 * b1 * b2 - 3.0 * b1 * b1 * b1 * b1) / max(var * var, 0.00001);
    float p = var / (var + (z - mu) * (z - mu));
    p *= clamp(1.2 - 0.08 * abs(sk) - 0.04 * clamp(ku - 3.0, 0.0, 8.0), 0.35, 1.15);
    if (z <= mu + bias) { return 1.0; }
    return clamp(p + 0.08 * clamp(z3, 0.0, 1.0), 0.12, 1.0);
}

float filterMap(sampler2D map, vec2 uv, float z, float bias, vec2 texel) {
    int k = ShadowPCF;
    if (k < 1) { k = 1; }
    if (ShadowFilter == 1) { return pcssAt(map, uv, z, bias, texel); }
    if (ShadowFilter == 2) { return evsmAt(map, uv, z, bias); }
    if (ShadowFilter == 3) { return msmAt(map, uv, z, bias); }
    return pcfAt(map, uv, z, bias, k, texel);
}

float filterAt(vec2 uv, float z, float bias, vec2 texel) {
    float a = filterMap(ShadowMap, uv, z, bias, texel);
    if (ShadowCacheSplit != 0) {
        float b = filterMap(ShadowMapDyn, uv, z, bias, texel);
        return min(a, b);
    }
    return a;
}

float sampleTile(int tile, vec4 lsp, float bias) {
    vec3 proj = lsp.xyz / max(lsp.w, 0.0001);
    proj = proj * 0.5 + 0.5;
    if (proj.z > 1.0 || proj.x <= 0.0 || proj.x >= 1.0 || proj.y <= 0.0 || proj.y >= 1.0) {
        return 1.0;
    }
    vec2 uv = tileUV(tile, proj.xy);
    vec2 texel = 1.0 / vec2(textureSize(ShadowMap, 0));
    return filterAt(uv, proj.z, bias, texel);
}

int cubeFace(vec3 d) {
    vec3 a = abs(d);
    if (a.x >= a.y && a.x >= a.z) { return d.x > 0.0 ? 0 : 1; }
    if (a.y >= a.z) { return d.y > 0.0 ? 2 : 3; }
    return d.z > 0.0 ? 4 : 5;
}

vec3 shadowOffsetPos() {
    vec3 n = normalize(WorldNormal);
    float tw = ShadowTexelWorld.x;
    if (tw < 1e-5) { tw = 0.05; }
    return WorldPos + n * ShadowNormalBias * tw;
}

float localShadows() {
    float s = 1.0;
    vec3 wp = shadowOffsetPos();
    for (int i = 0; i < 2; i++) {
        if (i >= ShadowPoints) { break; }
        vec3 L = wp - PointPos[i];
        float dist = length(L);
        if (dist > PointRange[i] || dist < 0.02) { continue; }
        int face = cubeFace(L);
        int tile = PointTile[i] + face;
        vec4 lsp = PointVP[i * 6 + face] * vec4(wp, 1.0);
        float att = 1.0 - clamp(dist / max(PointRange[i], 0.01), 0.0, 1.0);
        s *= mix(1.0, sampleTile(tile, lsp, ShadowBias * 1.5), att);
    }
    for (int i = 0; i < 2; i++) {
        if (i >= ShadowSpots) { break; }
        vec3 L = wp - SpotPos[i];
        float dist = length(L);
        if (dist > SpotRange[i] || dist < 0.02) { continue; }
        vec3 ld = normalize(L);
        if (dot(ld, normalize(SpotDir[i])) < SpotCos[i]) { continue; }
        vec4 lsp = SpotVP[i] * vec4(wp, 1.0);
        float att = 1.0 - clamp(dist / max(SpotRange[i], 0.01), 0.0, 1.0);
        s *= mix(1.0, sampleTile(SpotTile[i], lsp, ShadowBias), att);
    }
    return s;
}

float contactAt(vec2 uv, float z, vec2 texel) {
    float dark = 1.0;
    vec2 dir = vec2(0.35, 0.65);
    for (int i = 1; i <= 6; i++) {
        float t = float(i);
        vec2 suv = uv - dir * texel * t * 2.5;
        float d = sampleDepth(ShadowMap, suv);
        float gap = z - d;
        if (gap > 0.0004 && gap < 0.035) {
            dark = min(dark, mix(0.28, 1.0, gap / 0.035));
        }
    }
    return dark;
}

float sssAt(vec2 uv, float z, vec2 texel) {
    float dark = 1.0;
    vec2 dir = vec2(0.25, 0.8);
    int steps = 8;
    if (ShadowSSS > 1) { steps = 16; }
    for (int i = 1; i <= 16; i++) {
        if (i > steps) { break; }
        float t = float(i);
        vec2 suv = uv - dir * texel * t * 3.5;
        float d = sampleDepth(ShadowMap, suv);
        float gap = z - d;
        if (gap > 0.001 && gap < 0.12) {
            float fade = 1.0 - t / float(steps);
            dark = min(dark, mix(0.4, 1.0, gap / 0.12) * (1.0 - 0.45 * fade) + 0.45 * fade);
        }
    }
    return dark;
}

float casAt(int cas, float bias, out float inside) {
    inside = 0.0;
    if (cas < 0 || cas > 2) { return 1.0; }
    vec3 n = normalize(WorldNormal);
    float tw = ShadowTexelWorld[cas];
    if (tw < 1e-5) { tw = ShadowTexelWorld.x; }
    if (tw < 1e-5) { tw = 0.05; }
    vec2 texel = 1.0 / vec2(textureSize(ShadowMap, 0));
    vec3 wp = WorldPos + n * (texel.x * 2.5);
    wp += n * ShadowNormalBias * tw;
    vec4 lsp = LightVP[cas] * vec4(wp, 1.0);
    vec3 proj = lsp.xyz / max(lsp.w, 0.0001);
    proj = proj * 0.5 + 0.5;
    if (proj.z > 1.0 || proj.x <= 0.0 || proj.x >= 1.0 || proj.y <= 0.0 || proj.y >= 1.0) {
        return 1.0;
    }
    inside = 1.0;
    vec2 uv = tileUV(cas, proj.xy);
    float nUp = clamp(dot(n, vec3(0.0, 1.0, 0.0)), 0.0, 1.0);
    float dynamicBias = max(bias * (1.0 - nUp), bias * 0.4);
    float sh = filterAt(uv, proj.z, dynamicBias, texel);
    if (ShadowFilter < 2) {
        if (ShadowContact != 0) { sh *= contactAt(uv, proj.z, texel); }
        if (ShadowSSS != 0) { sh *= sssAt(uv, proj.z, texel); }
    }
    return sh;
}

float cascadeBlend(int cas, int n, float vz, float sh) {
    if (n > 1 && cas == 0) {
        float band = max(ShadowSplit.x * 0.35, 6.0);
        float t = smoothstep(ShadowSplit.x - band, ShadowSplit.x, vz);
        if (t > 0.0) {
            float in2 = 0.0;
            float sh2 = casAt(1, ShadowBias, in2);
            if (in2 > 0.5) { sh = mix(sh, sh2, t); }
        }
    } else if (n > 2 && cas == 1) {
        float band = max((ShadowSplit.y - ShadowSplit.x) * 0.35, 8.0);
        float t = smoothstep(ShadowSplit.y - band, ShadowSplit.y, vz);
        if (t > 0.0) {
            float in2 = 0.0;
            float sh2 = casAt(2, ShadowBias, in2);
            if (in2 > 0.5) { sh = mix(sh, sh2, t); }
        }
    }
    return sh;
}

float shadowFactor() {
    if (ShadowEnabled == 0) { return 1.0; }
    int n = ShadowCascades;
    if (n < 1) { n = 1; }
    if (n > 3) { n = 3; }
    float vz = -Position.z;
    int cas = 0;
    if (n > 1 && vz > ShadowSplit.x) { cas = 1; }
    if (n > 2 && vz > ShadowSplit.y) { cas = 2; }
    float inside = 0.0;
    float sh = casAt(cas, ShadowBias, inside);
    if (inside < 0.5 && cas + 1 < n) {
        sh = casAt(cas + 1, ShadowBias, inside);
    }
    if (inside < 0.5 && cas + 2 < n) {
        sh = casAt(cas + 2, ShadowBias, inside);
    }
    sh = cascadeBlend(cas, n, vz, sh);
    sh *= localShadows();
    return sh;
}
`
