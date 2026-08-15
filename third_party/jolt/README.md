# Jolt (Windows)

Static MinGW/LLVM libraries for [jolt-go](https://github.com/bbitechnologies/jolt-go) on Windows amd64.

Built from Jolt Physics v5.4.0 with the jolt-go C wrapper. Default 3D physics on Windows uses these (`internal/jolt`). `PhysicsBackend$()` returns `jolt`.

Linux amd64/arm64 and macOS ARM use upstream `jolt-go` prebuilts (no files in this folder).

`go build -tags nojolt` uses the software fallback instead (`PhysicsBackend$()` = `fallback`). That is not Jolt. Other OS/arch (for example macOS Intel) also use the fallback because neither this tree nor upstream ships a prebuilt.
