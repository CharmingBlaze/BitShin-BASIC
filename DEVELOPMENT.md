# Development

Public name: **BitShin BASIC**. Module path: `bitshinbasic`. CLI: `bs` / `bs.exe`. Go 1.25+ (see `go.mod`).

## Prerequisites

- Go toolchain
- For `go build ./cmd/bs`: a C compiler (MinGW-w64 `gcc` on Windows) and an OpenGL driver. G3N, Ebiten, cimgui-go, and Flecs use CGO.
- Optional: `golangci-lint` for the config in `.golangci.yml`

### Windows

MinGW `gcc` on `PATH`. Dist copies `libc++.dll` and `libunwind.dll` (ImGui) from `third_party/windows/`. Audio is Oto — no OpenAL DLL.

`bs build game.bb -o dist` copies `third_party/windows` DLLs into `dist\`. Zip `dist` and run from that folder. OpenGL, UCRT, and `dbghelp.dll` come from Windows.

### Linux

A C compiler plus OpenGL headers (`gcc`, `libgl1-mesa-dev`, `xorg-dev` on Ubuntu). Audio is Oto — no `libopenal1`.

### macOS

Xcode command-line tools. Audio is Oto — no `brew install openal-soft`.

## Test (no window)

`go test ./internal/...` does not open a display. `internal/runtime` tests need CGO (same as `go build ./cmd/bs`). Language packages do not.

```powershell
go test ./internal/...
go vet  ./internal/lex ./internal/parse ./internal/interp ./internal/value ./internal/ast ./internal/geo ./internal/phys2d ./internal/phys3d ./internal/netenet ./internal/lsp
```

Or:

```powershell
.\scripts\test.ps1
```

```sh
./scripts/test.sh
```

## Build the CLI

```powershell
go build -o bs.exe ./cmd/bs
Copy-Item third_party\windows\*.dll .
.\bs.exe examples\hello_console.bb
.\bs.exe examples\spinning_cube.bb
.\bs.exe examples\platform2d.bb
```

```powershell
.\scripts\build.ps1
```

`scripts/build.ps1` builds `bs.exe` and copies `third_party/windows/*.dll` beside it.

```powershell
.\bs.exe build examples\hello_console.bb -o dist
```

`dist\` then has `bs.exe`, the `.bb`, and `libc++.dll` + `libunwind.dll`. Run from `dist\`.

## Optional build tags

| Tag | Effect |
| --- | --- |
| (default) | 3D physics: native Jolt on Windows, Linux amd64/arm64, and macOS ARM. Other OS/arch: software fallback. Net: UDP. |
| `nojolt` | Software 3D physics fallback (`PhysicsBackend$()` = `fallback`; no Jolt CGO) |
| `enet` | Link [go-enet](https://github.com/codecat/go-enet). On Windows this needs `libenet` (the module ships `enet.lib` for MSVC, not MinGW). |

```powershell
go build -tags enet -o bs.exe ./cmd/bs
```

## Layout

```
cmd/bs              CLI (`bs lsp` = language server)
cmd/bsls            Language server only (stdio, no CGO)
internal/lsp        JSON-RPC LSP (diagnostics, hover, completion, symbols)
internal/lex        lexer
internal/parse      parser
internal/ast        syntax tree
internal/interp     tree-walk + Flip yield
internal/value      dynamic values
internal/runtime    Blitz commands, G3N, Ebiten
internal/phys3d     Jolt / fallback
internal/phys2d     Chipmunk (jakecoffman/cp)
internal/netenet    ENet / UDP
examples            .bb programs
docs                language, commands, architecture
scripts             PowerShell / sh helpers
```

## Adding a command

1. Implement the handler in `internal/runtime` (`commands.go`, `commands_2d.go`, `commands_phys.go`, `commands_net.go`, `commands_anim.go`, `commands_fx.go`, `commands_tiles.go`).
2. Register the lowercase name in that file’s map and merge it from `commandTable()`.
3. Document it in `docs/COMMANDS.md`.
4. Prefer a tiny `examples/*.bb` if the command is user-visible.
