# Windows runtime DLLs

`bs build` copies these next to `bs.exe`. Audio is Oto — no OpenAL.

| File | Why |
| --- | --- |
| `libc++.dll` | LLVM-MinGW C++ runtime (cimgui-go) |
| `libunwind.dll` | Required by `libc++.dll` |

Not shipped: OpenGL, GDI, UCRT, `dbghelp.dll` (Windows). OpenAL / libvorbis natives are not used.
