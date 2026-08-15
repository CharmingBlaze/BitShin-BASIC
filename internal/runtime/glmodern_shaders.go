package runtime

// GLSL for optional modern-GL wrappers. Ship path is #version 330.
// Compute is 430 and only compiled after a 4.3 / ARB_compute_shader detect.

const mbInstVertex = `#version 330 core
layout(location = 0) in vec3 VertexPosition;
layout(location = 1) in vec3 VertexNormal;
layout(location = 2) in mat4 InstanceMatrix;
uniform mat4 ViewProj;
uniform vec3 Color;
out vec3 vN;
out vec3 vC;
void main() {
    mat4 m = InstanceMatrix;
    vec4 wp = m * vec4(VertexPosition, 1.0);
    vN = mat3(m) * VertexNormal;
    vC = Color;
    gl_Position = ViewProj * wp;
}
`

const mbInstFragment = `#version 330 core
in vec3 vN;
in vec3 vC;
out vec4 FragColor;
void main() {
    vec3 n = normalize(vN);
    float d = max(dot(n, normalize(vec3(0.35, 0.85, 0.4))), 0.15);
    FragColor = vec4(vC * d, 1.0);
}
`

const mbInstDepthVertex = `#version 330 core
layout(location = 0) in vec3 VertexPosition;
layout(location = 2) in mat4 InstanceMatrix;
uniform mat4 ViewProj;
void main() {
    gl_Position = ViewProj * (InstanceMatrix * vec4(VertexPosition, 1.0));
}
`

const mbInstDepthFragment = `#version 330 core
uniform int MomentMode;
uniform float EvsmC;
out vec4 FragColor;
void main() {
    float d = clamp(gl_FragCoord.z, 0.0, 1.0);
    if (MomentMode == 2) {
        float c = EvsmC;
        if (c < 1.0) { c = 40.0; }
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

const mbGeomVertex = `#version 330 core
layout(location = 0) in vec4 Point; // xyz + size
layout(location = 1) in vec3 Color;
out vec3 vColor;
out float vSize;
void main() {
    gl_Position = vec4(Point.xyz, 1.0);
    vSize = Point.w;
    vColor = Color;
}
`

const mbGeomGeometry = `#version 330 core
layout(points) in;
layout(triangle_strip, max_vertices = 4) out;
uniform mat4 ViewProj;
uniform vec3 CamRight;
uniform vec3 CamUp;
in vec3 vColor[];
in float vSize[];
out vec3 gColor;
void main() {
    vec3 c = gl_in[0].gl_Position.xyz;
    float s = max(vSize[0], 0.05);
    vec3 r = CamRight * s;
    vec3 u = CamUp * s;
    gColor = vColor[0];
    gl_Position = ViewProj * vec4(c - r - u, 1.0);
    EmitVertex();
    gl_Position = ViewProj * vec4(c + r - u, 1.0);
    EmitVertex();
    gl_Position = ViewProj * vec4(c - r + u, 1.0);
    EmitVertex();
    gl_Position = ViewProj * vec4(c + r + u, 1.0);
    EmitVertex();
    EndPrimitive();
}
`

const mbGeomFragment = `#version 330 core
in vec3 gColor;
out vec4 FragColor;
void main() { FragColor = vec4(gColor, 1.0); }
`

const mbComputeFill = `#version 430
layout(local_size_x = 64) in;
layout(std430, binding = 0) buffer Heights { float h[]; };
uniform float Time;
uniform uint Count;
void main() {
    uint i = gl_GlobalInvocationID.x;
    if (i >= Count) { return; }
    h[i] = sin(float(i) * 0.17 + Time);
}
`

const mbTessCtrl = `#version 400 core
layout(vertices = 3) out;
in vec3 vPos[];
out vec3 tcPos[];
uniform float TessLevel;
void main() {
    tcPos[gl_InvocationID] = vPos[gl_InvocationID];
    if (gl_InvocationID == 0) {
        float t = max(TessLevel, 1.0);
        gl_TessLevelOuter[0] = t;
        gl_TessLevelOuter[1] = t;
        gl_TessLevelOuter[2] = t;
        gl_TessLevelInner[0] = t;
    }
}
`

const mbTessEval = `#version 400 core
layout(triangles, equal_spacing, ccw) in;
in vec3 tcPos[];
uniform mat4 ViewProj;
void main() {
    vec3 p = gl_TessCoord.x * tcPos[0] + gl_TessCoord.y * tcPos[1] + gl_TessCoord.z * tcPos[2];
    gl_Position = ViewProj * vec4(p, 1.0);
}
`

const mbTessVert = `#version 400 core
layout(location = 0) in vec3 VertexPosition;
out vec3 vPos;
void main() { vPos = VertexPosition; }
`

const mbTessFrag = `#version 400 core
out vec4 FragColor;
void main() { FragColor = vec4(0.2, 0.45, 0.7, 1.0); }
`
