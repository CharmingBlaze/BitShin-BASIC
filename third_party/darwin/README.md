# macOS natives

Audio is Oto (no OpenAL). Drop optional `.dylib` files here if you need to vendor extra C++ runtime for a custom cimgui build.

`bs build` copies any `*.dylib` in this folder next to `bs`.

Do not vendor system OpenGL / Metal. Those come from the OS.
