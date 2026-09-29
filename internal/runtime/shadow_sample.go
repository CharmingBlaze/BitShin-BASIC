package runtime

// Shared GLSL 330 shadow sampling for mbshadow / mbphysical / mbterrain.
// Features 16-tap Poisson Disk PCF with interleaved gradient noise (IGN) rotation,
// slope-scaled bias against light normal, per-light shadow isolation, and distance fade.

const mbshadowSampleGLSL = `
uniform sampler2D ShadowMapDyn;
uniform int ShadowCacheSplit;
uniform float ShadowNormalBias;
uniform vec4 ShadowTexelWorld;
uniform mat4 LightVP[4];
uniform vec4 ShadowSplit;
uniform float EvsmC;
uniform float ShadowSoftness;
uniform vec3 ShadowColor;
uniform float ShadowFadeNear;
uniform float ShadowFadeFar;
uniform vec3 ShadowSunDir;
uniform int MeshReceiveShadow;

const vec2 POISSON_DISK[16] = vec2[](
    vec2(-0.94201624, -0.39906216),
    vec2( 0.94558609, -0.76890725),
    vec2(-0.09418410, -0.92938870),
    vec2( 0.34495938,  0.29387760),
    vec2(-0.91588581,  0.45771432),
    vec2(-0.81544232, -0.87912464),
    vec2(-0.38277543,  0.27676845),
    vec2( 0.97484398,  0.75648377),
    vec2( 0.44323325, -0.97511554),
    vec2( 0.53742981, -0.47373420),
    vec2(-0.26496911, -0.41893023),
    vec2( 0.79197514,  0.19090188),
    vec2(-0.24188840,  0.99706507),
    vec2(-0.81409955,  0.91437590),
    vec2( 0.19984126,  0.78641367),
    vec2( 0.14383161, -0.14100790)
);

float gradientNoise(vec2 screenPos) {
    vec3 magic = vec3(0.06711056, 0.00583715, 52.9829189);
    return fract(magic.z * fract(dot(screenPos, magic.xy)));
}

float ignoise(vec2 p) {
    return gradientNoise(p);
}

vec2 rotateDisk(vec2 sampleVec, float angle) {
    float s = sin(angle);
    float c = cos(angle);
    return vec2(sampleVec.x * c - sampleVec.y * s, sampleVec.x * s + sampleVec.y * c);
}

float sampleDepth(sampler2D map, vec2 uv) {
    return texture(map, uv).r;
}

bool emptyDepth(float d) {
    return d <= 0.001 || d >= 0.999;
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
    float pad = 0.008;
    proj = clamp(proj, pad, 1.0 - pad);
    return vec2((float(col) + proj.x) / float(cols), (float(row) + proj.y) / float(rows));
}

vec2 clampTile(vec2 center, vec2 uv) {
    int cols = AtlasCols;
    int rows = AtlasRows;
    if (cols < 1) { cols = 1; }
    if (rows < 1) { rows = 1; }
    vec2 tile = vec2(1.0 / float(cols), 1.0 / float(rows));
    vec2 local = mod(center, tile);
    vec2 base = center - local;
    vec2 inset = max(tile * 0.02, vec2(1.0) / vec2(textureSize(ShadowMap, 0)));
    return clamp(uv, base + inset, base + tile - inset);
}

float pcf3x3(sampler2D map, vec2 uv, float z, float bias, vec2 texel) {
    float s = 0.0;
    for (int x = -1; x <= 1; x++) {
        for (int y = -1; y <= 1; y++) {
            float d = sampleDepth(map, clampTile(uv, uv + vec2(float(x), float(y)) * texel));
            if (emptyDepth(d)) {
                s += 1.0;
                continue;
            }
            s += (z - bias > d) ? 0.0 : 1.0;
        }
    }
    return s / 9.0;
}

float pcfAt(sampler2D map, vec2 uv, float z, float bias, int k, vec2 texel) {
    float angle = gradientNoise(gl_FragCoord.xy) * 6.2831853;
    float soft = ShadowSoftness;
    if (soft <= 0.001) { soft = 1.0; }
    float radius = max(float(k) * 0.42 * soft, 1.0);
    float s = 0.0;
    for (int i = 0; i < 16; i++) {
        vec2 offset = rotateDisk(POISSON_DISK[i], angle) * texel * radius;
        float d = sampleDepth(map, clampTile(uv, uv + offset));
        if (emptyDepth(d)) {
            s += 1.0;
            continue;
        }
        s += (z - bias > d) ? 0.0 : 1.0;
    }
    return s * 0.0625;
}

float pcssAt(sampler2D map, vec2 uv, float z, float bias, vec2 texel) {
    float soft = ShadowSoftness;
    if (soft <= 0.001) { soft = 1.0; }
    float search = max(ShadowLightSize * 10.0 * texel.x * soft, texel.x * 2.0);
    float blk = 0.0;
    float cnt = 0.0;
    float angle = gradientNoise(gl_FragCoord.xy) * 6.2831853;
    for (int i = 0; i < 16; i++) {
        vec2 offset = rotateDisk(POISSON_DISK[i], angle) * search;
        float d = sampleDepth(map, clampTile(uv, uv + offset));
        if (!emptyDepth(d) && d < z - bias) {
            blk += d;
            cnt += 1.0;
        }
    }
    if (cnt < 1.0) { return 1.0; }
    blk /= cnt;
    float penumbra = ((z - blk) * ShadowLightSize * 16.0) / max(blk, 0.01);
    int k = int(clamp(penumbra * 24.0, 2.0, 8.0));
    return pcfAt(map, uv, z, bias, k, texel);
}

float evsmAt(sampler2D map, vec2 uv, float z, float bias) {
    float c = EvsmC;
    if (c < 1.0 || c > 14.0) { c = 8.0; }
    vec4 m = sampleMoments(map, uv);
    float m1 = m.x;
    float m2 = m.y;
    if (m1 < 0.0001) { return 1.0; }
    if (m1 >= exp(c) * 0.999) { return 1.0; }
    float ez = exp(c * clamp(z, 0.0, 1.0));
    float depthBias = max(bias, 0.0005);
    if (ez <= m1 * exp(c * depthBias)) { return 1.0; }
    float var = max(m2 - m1 * m1, 0.0002);
    float md = ez - m1;
    float p = var / (var + md * md);
    return clamp(p, 0.0, 1.0);
}

float msmAt(sampler2D map, vec2 uv, float z, float bias) {
    vec4 b = sampleMoments(map, uv);
    float b1 = b.x;
    float b2 = b.y;
    if (emptyDepth(b1)) { return 1.0; }
    float mu = b1;
    float var = max(b2 - b1 * b1, 0.00015);
    if (z <= mu + bias) { return 1.0; }
    float md = z - mu;
    float p = var / (var + md * md);
    return clamp(p, 0.0, 1.0);
}

float filterMap(sampler2D map, vec2 uv, float z, float bias, vec2 texel) {
    int k = ShadowPCF;
    if (k < 3) { k = 3; }
    if (ShadowFilter == 1) { return pcssAt(map, uv, z, bias, texel); }
    if (ShadowFilter == 2) { return evsmAt(map, uv, z, bias); }
    if (ShadowFilter == 3) { return msmAt(map, uv, z, bias); }
    if (k <= 3) { return pcf3x3(map, uv, z, bias, texel); }
    return pcfAt(map, uv, z, bias, k, texel);
}

float filterAt(vec2 uv, float z, float bias, vec2 texel) {
    return filterMap(ShadowMap, uv, z, bias, texel);
}

float sampleTile(int tile, vec4 lsp, float bias) {
    vec3 proj = lsp.xyz / max(lsp.w, 0.0001);
    proj = proj * 0.5 + 0.5;
    if (proj.z <= 0.001 || proj.z >= 0.999 || proj.x <= 0.0 || proj.x >= 1.0 || proj.y <= 0.0 || proj.y >= 1.0) {
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

float calcDynamicBias(vec3 n, vec3 ldir, float baseBias) {
    float cosTheta = clamp(dot(n, ldir), 0.0, 1.0);
    float slope = sqrt(max(1.0 - cosTheta * cosTheta, 0.0)) / max(cosTheta, 0.001);
    return baseBias + clamp(slope * baseBias * 1.8, 0.0, baseBias * 4.0);
}

float receiverDepthBias(vec3 proj, vec3 n, vec3 ldir, float baseBias) {
    float dz = max(abs(dFdx(proj.z)), abs(dFdy(proj.z)));
    return calcDynamicBias(n, ldir, baseBias) + clamp(dz * 0.35, 0.0, baseBias);
}

float contactAt(vec2 uv, float z, vec2 texel) {
    float occ = 0.0;
    const float nearGap = 0.0004;
    const float farGap = 0.006;
    for (int i = 0; i < 8; i++) {
        vec2 o = POISSON_DISK[i] * texel * 2.5;
        float d = sampleDepth(ShadowMap, clamp(uv + o, vec2(0.001), vec2(0.999)));
        if (emptyDepth(d)) { continue; }
        float diff = z - d;
        if (diff > nearGap && diff < farGap) {
            occ += 1.0 - (diff - nearGap) / (farGap - nearGap);
        }
    }
    return mix(1.0, 0.42, clamp(occ / 8.0, 0.0, 1.0));
}

float sssAt(vec2 uv, float z, vec2 texel) {
    float q = float(ShadowSSS);
    if (q < 1.0) { return 1.0; }
    if (q > 4.0) { q = 4.0; }
    float occ = 0.0;
    float nearGap = 0.0005;
    float farGap = 0.01 * q;
    float radius = 4.0 + q * 5.0;
    for (int i = 0; i < 12; i++) {
        vec2 o = POISSON_DISK[i] * texel * radius;
        float d = sampleDepth(ShadowMap, clamp(uv + o, vec2(0.001), vec2(0.999)));
        if (emptyDepth(d)) { continue; }
        float diff = z - d;
        if (diff > nearGap && diff < farGap) {
            occ += 1.0 - diff / farGap;
        }
    }
    return mix(1.0, 0.30, clamp(occ / 8.0, 0.0, 1.0));
}

float casAt(int cas, vec3 wp, vec3 n, vec3 ldir, float bias, out float inside) {
    inside = 0.0;
    if (cas < 0 || cas > 3) { return 1.0; }
    float tw = ShadowTexelWorld[cas];
    if (tw < 1e-5) { tw = ShadowTexelWorld.x; }
    if (tw < 1e-5) { tw = 0.05; }
    vec2 texel = 1.0 / vec2(textureSize(ShadowMap, 0));
    float nb = ShadowNormalBias;
    if (nb < 0.15) { nb = 1.0; }
    float ndl = clamp(dot(normalize(n), normalize(ldir)), 0.0, 1.0);
    float grazing = 1.0 - ndl;
    vec3 offsetWP = wp + n * (nb * tw * grazing);
    vec4 lsp = LightVP[cas] * vec4(offsetWP, 1.0);
    vec3 proj = lsp.xyz / max(lsp.w, 0.0001);
    proj = proj * 0.5 + 0.5;
    if (proj.z <= 0.001 || proj.z >= 0.999 || proj.x <= 0.0 || proj.x >= 1.0 || proj.y <= 0.0 || proj.y >= 1.0) {
        return 1.0;
    }
    inside = 1.0;
    vec2 uv = tileUV(cas, proj.xy);
    float dynamicBias = receiverDepthBias(proj, n, ldir, bias);
    float sh = filterAt(uv, proj.z, dynamicBias, texel);
    if (ShadowFilter < 2) {
        if (ShadowContact != 0) { sh *= contactAt(uv, proj.z, texel); }
        if (ShadowSSS != 0) { sh *= sssAt(uv, proj.z, texel); }
    }
    return sh;
}

float cascadeBlend(int cas, int n, float vz, float sh, vec3 wp, vec3 nrm, vec3 ldir) {
    if (cas < 0 || cas >= n) { return sh; }
    float prev = 0.0;
    float farS = ShadowSplit.x;
    if (cas == 1) { prev = ShadowSplit.x; farS = ShadowSplit.y; }
    if (cas == 2) { prev = ShadowSplit.y; farS = ShadowSplit.z; }
    if (cas == 3) { prev = ShadowSplit.z; farS = ShadowSplit.w; }

    // Check the previous cascade just beyond a split as well as the next
    // cascade just before it. Taking the darker valid result prevents a lit
    // seam when one resolution level temporarily misses the same blocker.
    if (cas > 0) {
		float prevNear = 0.0;
		if (cas == 2) { prevNear = ShadowSplit.x; }
		if (cas == 3) { prevNear = ShadowSplit.y; }
		float prevBand = max((prev - prevNear) * 0.10, 1.0);
		if (vz < prev + prevBand) {
			float in0 = 0.0;
			float sh0 = casAt(cas - 1, wp, nrm, ldir, ShadowBias, in0);
			if (in0 > 0.5) { sh = min(sh, sh0); }
		}
	}
    if (cas < n - 1) {
		float band = max((farS - prev) * 0.10, 1.0);
		if (vz > farS - band) {
        float in2 = 0.0;
        float sh2 = casAt(cas + 1, wp, nrm, ldir, ShadowBias, in2);
			if (in2 > 0.5) { sh = min(sh, sh2); }
		}
    }
    return sh;
}

float sunShadowFactor(vec3 wp, vec3 nrm, vec3 ldir) {
    if (ShadowEnabled == 0 || MeshReceiveShadow == 0) { return 1.0; }
    if (dot(ldir, ldir) < 0.001) {
        ldir = ShadowSunDir;
        if (dot(ldir, ldir) < 0.001) { ldir = vec3(0.35, 0.85, 0.4); }
    }
    int n = ShadowCascades;
    if (n < 1) { n = 1; }
    if (n > 4) { n = 4; }
    float vz = -Position.z;
    int cas = 0;
    if (n > 1 && vz > ShadowSplit.x) { cas = 1; }
    if (n > 2 && vz > ShadowSplit.y) { cas = 2; }
    if (n > 3 && vz > ShadowSplit.z) { cas = 3; }
    float inside = 0.0;
    float sh = casAt(cas, wp, nrm, ldir, ShadowBias, inside);
    if (inside < 0.5 && cas + 1 < n) {
		cas += 1;
		sh = casAt(cas, wp, nrm, ldir, ShadowBias, inside);
    }
	if (inside < 0.5 && cas + 1 < n) {
		cas += 1;
		sh = casAt(cas, wp, nrm, ldir, ShadowBias, inside);
    }
	if (inside < 0.5 && cas + 1 < n) {
		cas += 1;
		sh = casAt(cas, wp, nrm, ldir, ShadowBias, inside);
    }
    if (inside < 0.5) { return 1.0; }
    sh = cascadeBlend(cas, n, vz, sh, wp, nrm, ldir);
    if (ShadowFadeFar > ShadowFadeNear && ShadowFadeFar > 0.0) {
        float fade = smoothstep(ShadowFadeNear, ShadowFadeFar, vz);
        sh = mix(sh, 1.0, fade);
    }
    return sh;
}

float pointShadowFactor(int i, vec3 wp, vec3 nrm) {
    if (ShadowEnabled == 0 || i >= ShadowPoints || i >= 2) { return 1.0; }
    vec3 L = wp - PointPos[i];
    float dist = length(L);
    if (dist > PointRange[i] || dist < 0.02) { return 1.0; }
    vec3 ldir = -normalize(L);
    int face = cubeFace(L);
    int tile = PointTile[i] + face;
    float tw = ShadowTexelWorld.x;
    if (tw < 1e-5) { tw = 0.05; }
    vec3 offsetWP = wp + nrm * (ShadowNormalBias * tw * 1.5);
    vec4 lsp = PointVP[i * 6 + face] * vec4(offsetWP, 1.0);
    float dynamicBias = calcDynamicBias(nrm, ldir, ShadowBias * 1.5);
    float att = 1.0 - clamp(dist / max(PointRange[i], 0.01), 0.0, 1.0);
    return mix(1.0, sampleTile(tile, lsp, dynamicBias), att);
}

float spotShadowFactor(int i, vec3 wp, vec3 nrm) {
    if (ShadowEnabled == 0 || i >= ShadowSpots || i >= 2) { return 1.0; }
    vec3 L = wp - SpotPos[i];
    float dist = length(L);
    if (dist > SpotRange[i] || dist < 0.02) { return 1.0; }
    vec3 ld = normalize(L);
    if (dot(ld, normalize(SpotDir[i])) < SpotCos[i]) { return 1.0; }
    vec3 ldir = -ld;
    float tw = ShadowTexelWorld.x;
    if (tw < 1e-5) { tw = 0.05; }
    vec3 offsetWP = wp + nrm * (ShadowNormalBias * tw * 1.5);
    vec4 lsp = SpotVP[i] * vec4(offsetWP, 1.0);
    float dynamicBias = calcDynamicBias(nrm, ldir, ShadowBias);
    float att = 1.0 - clamp(dist / max(SpotRange[i], 0.01), 0.0, 1.0);
    return mix(1.0, sampleTile(SpotTile[i], lsp, dynamicBias), att);
}

float localShadows() {
    float s = 1.0;
    vec3 n = normalize(WorldNormal);
    for (int i = 0; i < 2; i++) {
        if (i < ShadowPoints) { s *= pointShadowFactor(i, WorldPos, n); }
        if (i < ShadowSpots) { s *= spotShadowFactor(i, WorldPos, n); }
    }
    return s;
}

float shadowFactor() {
    vec3 n = normalize(WorldNormal);
    vec3 ldir = ShadowSunDir;
    if (dot(ldir, ldir) < 0.001) { ldir = vec3(0.35, 0.85, 0.4); }
    float sh = sunShadowFactor(WorldPos, n, ldir);
    sh *= localShadows();
    return sh;
}
`
