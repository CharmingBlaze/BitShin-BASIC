# Portable release

moonBASIC ships a **folder**, not an installer. Zip `dist/` and run the binary **from that folder** so natives stay beside the exe.

Do **not** ship `scratch/`, `imgui.ini`, `.ide_settings.json`, or the git working tree. Those are local-only (see `.gitignore`).

## One game zip

On the **same OS** you want to ship:

```text
moon build game.mb -o dist
```

That writes:

- `bs` / `moon.exe` — interpreter (CGO: G3N, ImGui, Flecs, Jolt)
- `game.mb` plus quoted `Load*` files and `assets/` if present
- Windows: `libc++.dll`, `libunwind.dll`
- Linux/macOS: optional `.so` / `.dylib` from `third_party/linux` or `third_party/darwin`

Optional flags: `-os`, `-arch`, `-tags` (`enet`, `nojolt`). Cross-compiling CGO usually **fails** without a target toolchain; `moon build` no longer copies the host binary as a fake. Build on Windows for Windows, on Linux for Linux, on a Mac for macOS.

`imgui.ini` may appear next to the exe after the first ImGui frame. That is user layout, not a redistributable.

## Engine / IDE

```text
go build -o moon.exe ./cmd/bs
go build -o moonsls.exe ./cmd/bsls
```

IDE: `scripts/build_ide.ps1` (Windows). That script builds the frontend and `moonbasic.exe`. The window is Neutralino (`neutralino-win_x64.exe` beside the config). Preferences are `.ide_settings.json` next to the repo or portable folder and are created on first save.

## Physics parity

| | Windows | Linux amd64/arm64, macOS ARM | Other / `-tags nojolt` |
|---|---|---|---|
| Rigid bodies + `Raycast` | Jolt | Same native Jolt | Software spheres/boxes |
| Cloth | Jolt soft body | Jolt soft body | Verlet |
| Joints, grab, CCD, vehicles | Full | Same native constraints and vehicles | Fallback joints + axis sliders |
| Mesh / convex / heightfield | Native cook | Same native cook | AABB / slab |
| Cylinder | Jolt `CylinderShape` | Jolt `CylinderShape` | Box of the same extents |
| Contacts (`EntityCollided`) | Native listener. One pair once per step; the point clears when they separate | Same native listener | Box and sphere overlap |
| Compound / sensor / overlap | Native compound + CollideShape | Same native compound | AABB / sphere |
| CharacterVirtual | Full + inner body | Full + inner body | Kinematic helper |
| `ApplyImpulse` on rigid bodies | Native | Native | Integrated |
| Rotation / torque / buoyancy | Native | Native rotated shapes | Integrated |
| `PhysicsThreads n` | Rebuilds Jolt job pool (1–32) + sizes the Go async pool | Same job pool | Same |

## Version

`moon version` is **1.0.0**. See [STATUS.md](STATUS.md) for Real vs Partial vs not-in-product.
