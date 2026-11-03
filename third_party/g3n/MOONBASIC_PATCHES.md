# G3N v0.2.0, patched

A copy of github.com/g3n/engine v0.2.0 (BSD licence, see LICENSE) with:

- window/glfw.go: the window asks for no multisampling. G3N hardcoded 8x. moonBASIC renders through its post pass, so that buffer was only cleared and resolved each frame: about 0.4 to 1.1 ms on a Radeon 780M.
- renderer/renderer.go: zSort takes depth from the model-view matrix. It applied that matrix to Position(), counting the translation twice.

The audio and tools folders are left out; moonBASIC does not use them.
