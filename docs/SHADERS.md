# Shaders

Two layers:

1. **Engine materials** — Phong (`mbshadow`), PBR (`mbphysical`), water, terrain, clouds. See `docs/PBR.md`.
2. **Shader programs (graph lite)** — `CreateShader` / `LoadShader` / `SetShader` / `SetShaderUniform`.

This is **not** an Unreal material graph. You write GLSL 330 vert+frag. G3N shaman prepends `#version 330 core` and expands `#include <attributes>`.

Typical vertex inputs: `VertexPosition`, `VertexNormal`, `VertexTexcoord`. Typical uniforms: `MVP`, `ModelMatrix`, `NormalMatrix`, `Color`. `Time` is set each draw on user materials.

`CompileShader(src$, stage$)` only **validates** a single stage (and optionally compiles it). It does not bind to a mesh.

`SetShaderUniform name$, …` without a shader handle writes **global** uniforms applied on lit/PBR/terrain. With a handle, uniforms stay on that program.

Example: `examples/shader.bb`.
