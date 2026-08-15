# Graphics (OpenGL + G3N)

BitShin BASIC stays on **G3N + go-gl OpenGL**. There is no bgfx, Vulkan, or wgpu path.

## Required baseline: OpenGL 3.3

The ship path is **OpenGL 3.3 core**. Everything needed to run a game must work here:

- G3N window, Phong / `mbshadow`, directional CSM + point/spot maps
- Particles, skybox, weather (atmosphere LUT + clouds + lightning), fog
- PBR (`mbphysical`) as GLSL that G3N compiles on 3.3
- Terrain and water on **regular meshes** (Gerstner in the vertex shader)
- Platform 64 and the other 3D examples

If the driver cannot create a 3.3 core context, `Graphics3D` fails the same way G3N already fails. We do **not** require 4.5 (or 4.3) to open the window.

## How the context is created

1. `Graphics3D` calls `startG3N` → G3N `window.Init`.
2. G3N’s GLFW hints (do not change these to 4.5):

```
glfw.WindowHint(ContextVersionMajor, 3)
glfw.WindowHint(ContextVersionMinor, 3)
glfw.WindowHint(OpenGLProfile, OpenGLCoreProfile)
```

On macOS, G3N also sets forward-compatible (required there).

3. Extra windows (`CreateWindow`) use the **same 3.3 core hints** and share the G3N context (`internal/runtime/extra_window.go`).
4. After the context exists, `detectModernGL()` reads `GL_MAJOR_VERSION` / `GL_MINOR_VERSION`, renderer string, and extensions (`glGetStringi`). It never raises the context version.

You will see one line like:

```
glmodern: OpenGL 3.3 (…renderer…) instance=1 geom=1 ubo=1 compute=0 ssbo=0 tess=0
```

## Optional 4.x (feature-detect)

These use raw go-gl **after** G3N state is saved (same idea as `saveShadowGL` in `shadows.go`). Restore program / VAO / buffers / viewport before returning to G3N.

| Feature | Min GL / extension | If missing |
| --- | --- | --- |
| GPU instancing | 3.1 / `GL_ARB_draw_instanced` | CPU-merged mesh (`BatchInstances`) |
| Geometry shaders | 3.2 / `GL_ARB_geometry_shader4` | `CreateGeomPoints` returns **0**, one skip line |
| UBO | 3.1 / `GL_ARB_uniform_buffer_object` | CPU float mirror; `BindUniformBuffer` returns **0** |
| Tessellation | 4.0 / `GL_ARB_tessellation_shader` | `EnableTessellation` returns **0**; water/terrain keep dense mesh |
| Compute | 4.3 / `GL_ARB_compute_shader` | `CreateComputeShader` / `DispatchCompute` return **0** |
| SSBO | 4.3 / `GL_ARB_shader_storage_buffer_object` | `CreateStorageBuffer` keeps a CPU mirror; `BindStorageBuffer` returns **0** |

Commands **never crash** and **never `End`** the program on an old GPU. They return `0` and print one:

```
glmodern: compute skipped (need OpenGL 4.3 or GL_ARB_compute_shader; have 3.3.0 …)
```

The skip is printed once per feature.

## G3N vs wrappers

G3N already uses go-gl internally (`github.com/go-gl/gl`). BitShin BASIC adds wrappers only for what G3N does not expose (compute, SSBO, UBO bind, instanced draw, geom/tess programs).

Do not fight GLS blindly: `saveModernGL` / restore around raw calls, like shadows.

Depth shadows stay on a G3N `Program` + `gs.UseProgram`. Dummy shadow map when shadows are off (unchanged).

## Instancing

`CreateInstancedMesh` / `SetInstanceTransform` / `SetInstanceData` are the commands (also used by the large-world example).

- **GPU (default on 3.3):** one copy of the mesh + per-instance matrices, `glDrawElementsInstanced`. Color pass after `rend.Render`. Shadow pass uses a small instanced depth program.
- **CPU fallback:** merge all instance triangles into one G3N mesh (original large-world path).

`EnableGPUInstances(0)` forces the merge path.

## Geometry / tessellation

- Geom: points expanded to camera-facing quads (`CreateGeomPoints`).
- Tess: optional compile of a tiny tess program. Terrain/water **do not** require it. If tess shaders fail, we skip; the dense mesh stays.

## Compute demo

`CreateComputeShader("")` compiles a 430 sine-fill. Bind an SSBO and `DispatchCompute`. On 3.3 this returns 0.

## go-glslang / SPIR-V

Package `internal/glslang`:

- Always: pure-Go validate (`#version`, braces, stage tokens).
- If `Graphics3D` is up: also `glCompileShader` (3.3 path).
- SPIR-V: only if `glslangValidator` or `glslang` is on `PATH`. **Vulkan is not required.** Native Khronos C++ is **not** shipped (Windows CGO pain). `-tags glslang` is reserved for a future binding.

```
bs shader myshader.vert vert
CompileShader(src$, "vert")
```

## Gonum vs math32

| Library | Role |
| --- | --- |
| G3N `math32` | Scene graph, cameras, meshes, every G3N API |
| `gonum.org/v1/gonum` via `internal/mathx` | FBM, IBL SH, crowd separation, job-system batch 4×4 |

Do **not** replace `math32.Vector3` / `Matrix4` on nodes. Convert at the boundary (`mathx.FromVec3` / `ToVec3` / `FromMat4` / `ToMat4`).

Used today: probe SH (`SHEval`), crowd push-apart, `NoiseFBM`, `JobXform`.

## Commands

Every command in this file is registered and listed in `docs/COMMANDS.md` (Modern OpenGL + Instancing). Demo: `examples/glmodern.bb`.
