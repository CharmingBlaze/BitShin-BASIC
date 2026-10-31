package runtime

// GLSL for GPU emitters (particles_gpu.go). Both programs are #version 330:
// the simulation runs as a vertex shader into transform feedback, and the
// draw expands each particle into a camera-facing quad with instancing.
// Everything is in G3N scene space (z is the script's -z).

// gpuPartVaryings are captured by transform feedback, interleaved, in the
// order of the particle record: 3 vec4 = gpuPartFloats floats.
var gpuPartVaryings = []string{"oPosAge", "oVelLife", "oMisc"}

const mbGPUPartSimVertex = `#version 330 core
layout(location = 0) in vec4 inPosAge;  // xyz position, w age (s)
layout(location = 1) in vec4 inVelLife; // xyz velocity, w lifespan (s); 0 = dead
layout(location = 2) in vec4 inMisc;    // x random, y spin, z size jitter, w spin speed
out vec4 oPosAge;
out vec4 oVelLife;
out vec4 oMisc;

uniform float Dt;
uniform float SimTime;
uniform int MaxCount;
uniform int SpawnStart;
uniform int SpawnCount;
uniform float SpawnSpread;
uniform uint Seed;
uniform vec3 Origin;
uniform vec3 PrevOrigin;
uniform int Shape;       // 0 point/box jitter, 2 sphere, 3 disc
uniform vec3 ShapeSize;  // box half extents, or radius in x
uniform vec3 Dir;
uniform float Speed;
uniform float SpeedVar;
uniform float ConeHalf;  // radians
uniform vec2 Life;       // min, max
uniform vec3 Accel;      // gravity + wind
uniform float Drag;
uniform vec4 Turb;       // strength, scale, speed, on
uniform vec4 Ground;     // on, y, bounce, friction
uniform vec3 Shift;      // world shift since the last step
uniform int Style;       // 0 soft, 1 streak, 2 flake

uint pcg(uint v) {
    uint s = v * 747796405u + 2891336453u;
    uint w = ((s >> ((s >> 28u) + 4u)) ^ s) * 277803737u;
    return (w >> 22u) ^ w;
}
float rnd(inout uint s) {
    s = pcg(s);
    return float(s) * (1.0 / 4294967296.0);
}
// One rnd per statement: drivers may copy an inout in before every call
// of one expression runs, which hands all three the same number.
vec3 rnd3(inout uint s) {
    float a = rnd(s);
    float b = rnd(s);
    float c = rnd(s);
    return vec3(a, b, c);
}

float hash3(vec3 p) {
    p = fract(p * 0.3183099 + vec3(0.71, 0.113, 0.419));
    p *= 17.0;
    return fract(p.x * p.y * p.z * (p.x + p.y + p.z));
}
float vnoise(vec3 x) {
    vec3 i = floor(x);
    vec3 f = fract(x);
    f = f * f * (3.0 - 2.0 * f);
    return mix(mix(mix(hash3(i + vec3(0, 0, 0)), hash3(i + vec3(1, 0, 0)), f.x),
                   mix(hash3(i + vec3(0, 1, 0)), hash3(i + vec3(1, 1, 0)), f.x), f.y),
               mix(mix(hash3(i + vec3(0, 0, 1)), hash3(i + vec3(1, 0, 1)), f.x),
                   mix(hash3(i + vec3(0, 1, 1)), hash3(i + vec3(1, 1, 1)), f.x), f.y), f.z);
}
vec3 potential(vec3 p) {
    return vec3(vnoise(p), vnoise(p + vec3(31.4, -17.2, 5.7)), vnoise(p + vec3(-9.1, 23.3, 41.7)));
}
// Curl of a noise potential: divergence-free, so it swirls without bunching.
vec3 curlNoise(vec3 p) {
    const float e = 0.12;
    vec3 dx = vec3(e, 0.0, 0.0);
    vec3 dy = vec3(0.0, e, 0.0);
    vec3 dz = vec3(0.0, 0.0, e);
    vec3 x0 = potential(p - dx), x1 = potential(p + dx);
    vec3 y0 = potential(p - dy), y1 = potential(p + dy);
    vec3 z0 = potential(p - dz), z1 = potential(p + dz);
    vec3 c = vec3((y1.z - y0.z) - (z1.y - z0.y),
                  (z1.x - z0.x) - (x1.z - x0.z),
                  (x1.y - x0.y) - (y1.x - y0.x));
    return c / (2.0 * e);
}

vec3 coneDir(vec3 d, inout uint s) {
    float cosA = cos(rnd(s) * ConeHalf);
    float phi = rnd(s) * 6.2831853;
    vec3 a = abs(d.y) > 0.9 ? vec3(1.0, 0.0, 0.0) : vec3(0.0, 1.0, 0.0);
    vec3 t = normalize(cross(a, d));
    vec3 b = cross(d, t);
    float sinA = sqrt(max(0.0, 1.0 - cosA * cosA));
    return d * cosA + (t * cos(phi) + b * sin(phi)) * sinA;
}

vec3 spawnOffset(inout uint s) {
    if (Shape == 2) {
        vec3 v = rnd3(s) * 2.0 - 1.0;
        float l = length(v);
        v = l > 1e-4 ? v / l : vec3(0.0, 1.0, 0.0);
        return v * ShapeSize.x * pow(rnd(s), 1.0 / 3.0);
    }
    if (Shape == 3) {
        float a = rnd(s) * 6.2831853;
        float r = ShapeSize.x * sqrt(rnd(s));
        return vec3(cos(a) * r, 0.0, sin(a) * r);
    }
    return (rnd3(s) * 2.0 - 1.0) * ShapeSize;
}

void main() {
    vec4 pa = inPosAge;
    vec4 vl = inVelLife;
    vec4 mi = inMisc;
    int idx = gl_VertexID;
    int rel = idx - SpawnStart;
    if (rel < 0) {
        rel += MaxCount;
    }
    if (rel < SpawnCount) {
        uint s = Seed ^ (uint(idx) * 2654435761u);
        float frac = (float(rel) + rnd(s)) / float(SpawnCount);
        float pre = SpawnSpread * (1.0 - frac);
        vec3 p = mix(PrevOrigin, Origin, frac) + spawnOffset(s);
        float sp = Speed * (1.0 + SpeedVar * (rnd(s) * 2.0 - 1.0));
        vec3 v = coneDir(normalize(Dir), s) * sp;
        float life = mix(Life.x, Life.y, rnd(s));
        v += Accel * pre;
        p += v * pre;
        pa = vec4(p, pre);
        vl = vec4(v, max(life, 0.001));
        vec3 m3 = rnd3(s);
        float m4 = rnd(s);
        mi = vec4(m3.x, m3.y * 6.2831853, m3.z, (m4 * 2.0 - 1.0) * 2.8);
    } else if (vl.w > 0.0 && pa.w < vl.w) {
        vec3 p = pa.xyz - Shift;
        vec3 v = vl.xyz + Accel * Dt;
        if (Turb.w > 0.0) {
            v += curlNoise(p * Turb.y + vec3(0.0, SimTime * Turb.z, SimTime * Turb.z * 0.37)) * (Turb.x * Dt);
        }
        v *= max(0.0, 1.0 - Drag * Dt);
        p += v * Dt;
        if (Style == 2) {
            p.x += sin(pa.w * 1.7 + mi.y) * 0.62 * Dt;
            p.z += cos(pa.w * 1.25) * 0.48 * Dt;
            mi.y += mi.w * Dt;
        }
        if (Ground.x > 0.5 && p.y < Ground.y) {
            p.y = Ground.y;
            if (v.y < 0.0) {
                v.y = -v.y * Ground.z;
            }
            v.xz *= max(0.0, 1.0 - Ground.w);
        }
        pa = vec4(p, pa.w + Dt);
        vl.xyz = v;
    }
    oPosAge = pa;
    oVelLife = vl;
    oMisc = mi;
    gl_Position = vec4(0.0, 0.0, 0.0, 1.0);
}
`

// Rasterizer discard drops every fragment; some drivers still want one.
const mbGPUPartSimFragment = `#version 330 core
out vec4 FragColor;
void main() {
    FragColor = vec4(0.0);
}
`

const mbGPUPartVertex = `#version 330 core
layout(location = 0) in vec4 iPosAge;
layout(location = 1) in vec4 iVelLife;
layout(location = 2) in vec4 iMisc;
uniform mat4 ViewProj;
uniform vec3 CamRight;
uniform vec3 CamUp;
uniform vec3 CamPos;
uniform vec2 Size;       // start, end
uniform float SizeVar;
uniform float Tall;
uniform float StreakLen;
uniform vec4 Color0;
uniform vec4 Color1;
uniform int Style;
out vec2 vUV;
out vec4 vCol;
out float vDist;
void main() {
    float life = iVelLife.w;
    float age = iPosAge.w;
    if (life <= 0.0 || age >= life) {
        vUV = vec2(0.0);
        vCol = vec4(0.0);
        vDist = 0.0;
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0); // outside clip: no fragments
        return;
    }
    vec2 c = vec2(float(gl_VertexID & 1), float((gl_VertexID >> 1) & 1)) * 2.0 - 1.0;
    float t = clamp(age / life, 0.0, 1.0);
    float sz = max(mix(Size.x, Size.y, t) * (1.0 + SizeVar * (iMisc.z * 2.0 - 1.0)), 0.0);
    vec3 p = iPosAge.xyz;
    vec3 r = CamRight;
    vec3 u = CamUp;
    float hx = sz * 0.5;
    float hy = sz * 0.5 * Tall;
    if (Style == 1) {
        vec3 v = iVelLife.xyz;
        vec3 toCam = normalize(CamPos - p);
        vec3 ax = v - dot(v, toCam) * toCam;
        float l = length(ax);
        if (l > 1e-4) {
            u = ax / l;
            r = normalize(cross(u, toCam));
            hy = hy + length(v) * StreakLen * 0.5;
        }
    } else if (Style == 2) {
        float cs = cos(iMisc.y);
        float sn = sin(iMisc.y);
        vec3 r2 = r * cs + u * sn;
        u = u * cs - r * sn;
        r = r2;
    }
    vec3 wp = p + r * (c.x * hx) + u * (c.y * hy);
    gl_Position = ViewProj * vec4(wp, 1.0);
    vUV = c * 0.5 + 0.5;
    vCol = mix(Color0, Color1, t);
    vDist = length(wp - CamPos);
}
`

const mbGPUPartFragment = `#version 330 core
in vec2 vUV;
in vec4 vCol;
in float vDist;
uniform sampler2D PartTex;
uniform int UseTex;
uniform int Additive;
uniform int Style;
uniform int FogMode;
uniform vec3 FogColor;
uniform float FogNear;
uniform float FogFar;
uniform float FogDensity;
out vec4 FragColor;
void main() {
    vec4 c = vCol;
    if (UseTex == 1) {
        c *= texture(PartTex, vUV);
    } else {
        vec2 d = vUV * 2.0 - 1.0;
        if (Style == 1) {
            d.y *= 0.55; // a streak keeps its length
        }
        float r2 = dot(d, d);
        if (r2 >= 1.0) {
            discard;
        }
        float a = 1.0 - r2;
        c.a *= a * a;
    }
    if (c.a < 0.002) {
        discard;
    }
    float f = 1.0;
    if (FogMode == 1) {
        float span = max(FogFar - FogNear, 0.0001);
        f = clamp((FogFar - vDist) / span, 0.0, 1.0);
    } else if (FogMode == 2) {
        f = exp(-FogDensity * vDist);
    } else if (FogMode != 0) {
        float fd = FogDensity * vDist;
        f = exp(-fd * fd);
    }
    f = clamp(f, 0.0, 1.0);
    // Premultiplied out, blended ONE / ONE_MINUS_SRC_ALPHA: alpha 0 adds.
    if (Additive == 1) {
        FragColor = vec4(c.rgb * c.a * f, 0.0);
    } else {
        FragColor = vec4(mix(FogColor, c.rgb, f) * c.a, c.a);
    }
}
`
