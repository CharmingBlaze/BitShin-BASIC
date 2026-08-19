# Portable release

BitShin BASIC ships a **folder**, not an installer. Zip `dist/` and run the binary **from that folder** so natives stay beside the exe.

Do **not** ship `scratch/`, `imgui.ini`, `.ide_settings.json`, or the git working tree. Those are local-only (see `.gitignore`).

## One game zip

On the **same OS** you want to ship:

```text
bs build game.bb -o dist
```

That writes:

- `bs` / `bs.exe` — interpreter (CGO: G3N, ImGui, Flecs, Jolt)
- `game.bb` plus quoted `Load*` files and `assets/` if present
- Windows: `libc++.dll`, `libunwind.dll`
- Linux/macOS: optional `.so` / `.dylib` from `third_party/linux` or `third_party/darwin`

Optional flags: `-os`, `-arch`, `-tags` (`enet`, `nojolt`). Cross-compiling CGO usually **fails** without a target toolchain; `bs build` no longer copies the host binary as a fake. Build on Windows for Windows, on Linux for Linux, on a Mac for macOS.

`imgui.ini` may appear next to the exe after the first ImGui frame. That is user layout, not a redistributable.

## Engine / IDE

```text
go build -o bs.exe ./cmd/bs
go build -o bsls.exe ./cmd/bsls
```

IDE: `scripts/build_ide.ps1` (Windows). Preferences are `.ide_settings.json` next to the repo or portable folder and are created on first save.

## Physics parity

| | Windows | Linux amd64/arm64, macOS ARM | Other / `-tags nojolt` |
|---|---|---|---|
| Rigid bodies + `Raycast` | Jolt | Jolt (`jolt-go`) | Software spheres/boxes |
| Cloth | Jolt soft body | Verlet sheet (same as fallback) | Verlet |
| Joints, grab, CCD, vehicles | Full | Position joints + grab spring; CCD/vehicles stub | Fallback joints |
| Mesh / convex / heightfield | Native cook | jolt-go mesh + convex hull; heightfield → triangle mesh | AABB / slab |
| Compound / sensor / overlap | Native compound + CollideShape | Convex hull of child AABBs; sensor flag at create; CollideShape overlap / stepped box `ShapeCast` | AABB / sphere |
| CharacterVirtual | Full + inner body | CharacterVirtual + kinematic inner capsule | Kinematic helper |
| `ApplyImpulse` on rigid bodies | Native | One-step position kick (jolt-go has no SetLinearVelocity) | Integrated |
| `PhysicsThreads n` | Rebuilds Jolt job pool (1–32) + sizes the Go async pool | Sizes the Go `PhysicsAsync` pool | Same |

## Version

`bs version` is **1.0.0**. See [STATUS.md](STATUS.md) for Real vs Partial vs not-in-product.
