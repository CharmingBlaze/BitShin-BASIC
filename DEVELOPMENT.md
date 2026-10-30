# Development

Public name: **moonBASIC**. Module path: `moonbasic`. CLI binary: `moon.exe`, built from `./cmd/bs`. Language server: `moonsls.exe`, built from `./cmd/bsls`. Go 1.25+ (see `go.mod`).

## Prerequisites

- Go toolchain
- For `go build ./cmd/bs`: a C compiler (MinGW-w64 `gcc` on Windows) and an OpenGL driver. G3N, Ebiten, cimgui-go, and Flecs use CGO.
- Optional: `golangci-lint` for the config in `.golangci.yml`

### Windows

MinGW `gcc` on `PATH`. Dist copies `libc++.dll` and `libunwind.dll` (ImGui) from `third_party/windows/`. Audio is Oto — no OpenAL DLL.

`moon build game.mb -o dist` copies `third_party/windows` DLLs into `dist\`. Zip `dist` and run from that folder. See [docs/RELEASE.md](docs/RELEASE.md). OpenGL, UCRT, and `dbghelp.dll` come from Windows.

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
go build -o moon.exe ./cmd/bs
Copy-Item third_party\windows\*.dll .
.\moon.exe examples\hello_console.mb
.\moon.exe examples\spinning_cube.mb
.\moon.exe examples\platform2d.mb
```

```powershell
.\scripts\build.ps1
```

`scripts/build.ps1` builds `moon.exe` and copies `third_party/windows/*.dll` beside it.

## IDE

The IDE window is [Neutralino](https://neutralino.js.org/). The Go process in `ide/` serves the Svelte app and the editor API. It does not use Wails.

```powershell
.\scripts\build_ide.ps1
.\moonbasic.exe
```

`go build -o moonbasic.exe .` from `ide/` is a normal Go build. The Neutralino binary is `ide/bin/neutralino-win_x64.exe`.

```powershell
.\moon.exe build examples\hello_console.mb -o dist
```

`dist\` then has `moon.exe`, the `.mb`, and `libc++.dll` + `libunwind.dll`. Run from `dist\`.

## Optional build tags

| Tag | Effect |
| --- | --- |
| (default) | 3D physics: native Jolt on Windows, Linux amd64/arm64, and macOS ARM. Other OS/arch: software fallback. Net: UDP. |
| `nojolt` | Software 3D physics fallback (`PhysicsBackend$()` = `fallback`; no Jolt CGO) |
| `enet` | Link [go-enet](https://github.com/codecat/go-enet). On Windows this needs `libenet` (the module ships `enet.lib` for MSVC, not MinGW). |

```powershell
go build -tags enet -o moon.exe ./cmd/bs
```

## Layout

```
cmd/bs              CLI source. Binary is moon.exe (`moon lsp` = language server)
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
internal/netenet    UDP host; ENet with -tags enet. Lobby and LAN beacons are internal/runtime
examples            .mb programs
docs                language, commands, architecture
scripts             PowerShell / sh helpers
```

## Adding a command

1. Implement the handler in `internal/runtime` (`commands.go`, `commands_2d.go`, `commands_phys.go`, `commands_net.go`, `net_session.go`, `commands_anim.go`, `commands_fx.go`, `commands_tiles.go`).
2. Register the lowercase name in that file’s map and merge it from `commandTable()`.
3. Document it in `docs/COMMANDS.md`. Network commands also go in `docs/NETWORK.md`.
4. Prefer a tiny `examples/*.mb` if the command is user-visible.
