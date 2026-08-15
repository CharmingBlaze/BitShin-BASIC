# Contributing

## Before a pull request

1. Read [DEVELOPMENT.md](DEVELOPMENT.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
2. Keep the change small and in the right package (see below).
3. Update docs in the same change:
   - new/changed BASIC syntax → `docs/LANGUAGE.md`
   - new/changed command → `docs/COMMANDS.md`
   - new package or Flip/backend change → `docs/ARCHITECTURE.md`
   - new example → `README.md` table
4. Run the test and build commands in DEVELOPMENT.md.

## Where code goes

| Change | Package / file |
| --- | --- |
| Tokens, comments, operators | `internal/lex` |
| Grammar, `If`/`While`/`Function` | `internal/parse`, `internal/ast` |
| Evaluation, `Flip` yield | `internal/interp` |
| Numbers/strings/arrays | `internal/value` |
| 3D G3N commands | `internal/runtime/commands.go`, `world.go` |
| 2D Ebiten commands | `internal/runtime/commands_2d.go`, `graphics2d.go` |
| Jolt / 3D bodies | `internal/phys3d`, `commands_phys.go` |
| Chipmunk 2D | `internal/phys2d`, `commands_phys.go` |
| Networking | `internal/netenet`, `commands_net.go` |
| Animation / particles / scenes | `commands_anim.go`, `commands_fx.go` |
| Tiles / fonts | `commands_tiles.go` |
| Zip pak | `pak.go` |
| `bs build` | `cmd/bs/build.go` |
| CLI | `cmd/bs` |
| Samples | `examples/*.bb` |

New builtins are registered by name (lowercase) in the relevant `*Commands` map. The interpreter is case-insensitive.

## Naming

- Go: exported API only at package boundaries; command handlers stay on `*World`.
- BASIC commands match Blitz3D when a classic name exists (`CreateCube`, `KeyDown`).
- New commands use a clear suffix: `Graphics2D`, `CreateSprite`, `CreateBodySphere`, `NetHost`.

## Style

- `gofmt` on all `.go` files.
- `go vet` on language packages before a PR.
- Do not commit secrets, `.exe` binaries, or editor junk (see `.gitignore`).
- Do not add paid services or API-key config to the repo.

## Commit messages

One or two sentences on why, not a file list. Example: `Add Graphics2D Flip path so 2D programs share the same game loop.`
