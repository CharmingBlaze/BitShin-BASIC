# G3N v0.2.0, patched

A copy of github.com/g3n/engine v0.2.0 (BSD licence, see LICENSE) with:

- window/glfw.go: the window asks for no multisampling. G3N hardcoded 8x. moonBASIC renders through its post pass, so that buffer was only cleared and resolved each frame: about 0.4 to 1.1 ms on a Radeon 780M.
- renderer/renderer.go: zSort takes depth from the model-view matrix. It applied that matrix to Position(), counting the translation twice.

- renderer/shaman.go: `#define` lines are sorted. Map order made every launch produce different source for the same shader, which defeats any cache.
- gls: program binaries. `Program.Build` asks `gls.ProgramCache` for a saved program before compiling, and `gls.ProgramBuilt` reports what each one cost. glapi.c loads the three entry points, which a 3.3 context may not have; `BinarySupported` is false then.

- graphic/graphic.go: `SetIGeometry`, so a mesh inside a loaded model can change detail with distance.

The audio and tools folders are left out; moonBASIC does not use them.
