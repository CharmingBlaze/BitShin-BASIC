# Linux natives

Audio is Oto (no OpenAL). Drop optional `.so` files here if you need to vendor extra C++ runtime for a custom cimgui build.

`bs build` copies any `*.so*` in this folder next to `bs`.

Do not vendor `libGL`. The OpenGL driver comes from the OS.
