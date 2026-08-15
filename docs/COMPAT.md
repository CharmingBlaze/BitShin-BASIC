# OpenGL compatibility

**Required: OpenGL 3.3 core.** The window always requests 3.3 — never 4.5 as a minimum.  
**Optional: OpenGL 4.x** (compute, SSBO, tessellation). Feature-detect after context create. Missing features return **0** and print one `glmodern: … skipped` line. They do not crash and do not `End` the program.

Full wrappers, G3N interaction, glslang, and Gonum: [GRAPHICS.md](GRAPHICS.md). Commands: [COMMANDS.md](COMMANDS.md) (Modern OpenGL). Status: [STATUS.md](STATUS.md).

## How the context is created

1. `Graphics3D` → `startG3N` → G3N `window.Init`.
2. G3N GLFW hints: `ContextVersionMajor = 3`, `Minor = 3`, core profile (forward-compatible on macOS).
3. Extra windows (`CreateWindow`) use the **same 3.3 hints** and share that context.
4. `detectModernGL()` then reads `GL_MAJOR_VERSION` / renderer / extensions. It never raises the hint.

G3N’s shaman prepends `#version 330 core` to `standard`, `mbshadow`, `mbphysical`, `mbwater`, `mbterrain`, `mbclouds`. Depth / blit shaders are 330 explicitly.

If the driver cannot make a 3.3 core context, `Graphics3D` fails (same as stock G3N). Ebiten 2D does not use this context.

## Required vs optional

| Always (3.3 ship path) | Optional (detect; 0 + skip on old GPUs) |
| --- | --- |
| G3N Phong, PBR (`mbphysical` 330), shadows CSM | `CreateComputeShader` / `DispatchCompute` (GL 4.3 / ARB_compute) |
| Terrain regular mesh + `mbterrain` splat, Gerstner water plane | `BindStorageBuffer` / GPU SSBO (4.3 / ARB_ssbo) |
| Particles, sky, weather, fog, Platform 64 | `EnableTessellation` (GL 4.0 / ARB_tessellation) |
| UBO (`CreateUniformBuffer`) | — UBO itself is 3.1; bind no-ops only if the extension is missing |
| GPU instancing (`EnableGPUInstances`) or CPU merge | — instancing is 3.1; CPU merge if you turn GPU off |
| Geometry billboards (`CreateGeomPoints`) | returns 0 if geom shaders fail to compile |
| `CompileShader` validate (pure Go, no window) | SPIR-V if `glslangValidator` on PATH; GL compile if 3.3 context |

## Old GPU behavior

| Command family | GL 3.3 (typical laptop) | GL 4.3+ |
| --- | --- | --- |
| `GLVersion$` / `GLHas*` | Real numbers / 0–1 | Real |
| `CreateInstancedMesh` | GPU instanced draw | Same |
| `CreateUniformBuffer` | Real UBO | Real |
| `CreateGeomPoints` | Billboards if geom compiles | Same |
| `CreateComputeShader` | **0** + skip line | Handle |
| `DispatchCompute` | **0** | 1 |
| `CreateStorageBuffer` | CPU float mirror (handle ≠ 0) | GPU SSBO |
| `BindStorageBuffer` | **0** + skip | 1 |
| `EnableTessellation` | **0** + skip | 1 if tess shaders compile |
| `CompileShader` | Validate; GL compile if window | Same + compute stage if 4.3 |

`CreateStorageBuffer` still returns a handle on 3.3 so `Set`/`Get` work on the CPU copy. Binding and compute dispatch do not.

## glslang / SPIR-V

No Vulkan runtime. Native Khronos C++ is not shipped. `internal/glslang` always validates. SPIR-V is optional (`glslangValidator` on PATH). `bs shader file.glsl [stage]`. `-tags glslang` is reserved for a future C binding.

## Gonum vs math32

G3N APIs take `math32`. Gonum (`internal/mathx`) is the engine numeric library (FBM, SH, crowd, `JobXform`). Do not replace scene-graph types.
