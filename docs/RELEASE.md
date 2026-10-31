# Portable release

moonBASIC ships a **folder**, not an installer. Zip `dist/` and run the binary **from that folder** so natives stay beside the exe.

Do **not** ship `scratch/`, `imgui.ini`, `.ide_settings.json`, or the git working tree. Those are local-only (see `.gitignore`).

## One game zip

On the **same OS** you want to ship:

```text
moon build game.mb -o dist
```

Run it from the moonBASIC source root (it runs `go build ./cmd/moon`). That writes:

- `moon` / `moon.exe`, or `<name>.exe` with `-name` or a `moon.json` name: the interpreter (CGO: G3N, ImGui, Flecs, Jolt) with the game's script stamped in, so a double-click runs it. `game.exe other.mb` still runs another script
- `game.mb` and every `.mb` it names (`Include`), always as loose files
- the assets: quoted paths in the scripts, plus the `assets/` and `scenes/` folders if present. Loose files, or one `<name>.pak` zip (`-pak`, the release profile)
- Windows: `libc++.dll`, `libunwind.dll`
- Linux/macOS: optional `.so` / `.dylib` from `third_party/linux` or `third_party/darwin`

Cross-compiling CGO usually **fails** without a target toolchain; `moon build` no longer copies the host binary as a fake. Build on Windows for Windows, on Linux for Linux, on a Mac for macOS.

### Flags

| Flag | Meaning |
| --- | --- |
| `-o dir` | Output folder (default `build.outputDir` from `moon.json`, else `dist`) |
| `-os`, `-arch` | Target: `windows`, `linux`, `darwin`; `amd64`, `arm64` |
| `-tags tags` | Go build tags, e.g. `enet`, `nojolt` |
| `-profile name` | `dev` (default), `debug`, `release`, or a profile from `moon.json` |
| `-pak` / `-loose` | Assets in `<name>.pak`, or as loose files, whatever the profile says |
| `-headless` | A server build: no window or GL (also `-server`) |
| `-strict` | A missing asset fails the build instead of printing a warning |
| `-name Game` | Binary name and the product name in the Windows file properties |
| `-version 1.2.3` | File and product version (Windows) |
| `-company Co` | Company name (Windows) |
| `-icon icon.png` | Application icon (Windows). `.ico` is used as is; `.png`, `.jpg`, or `.gif` is resized to 256, 64, 48, 32, and 16 pixels |

On Windows the icon and version info go into a resource object (`cmd/moon/zz_moonbuild_windows_<arch>.syso`) that exists only during the build. Explorer shows them under **Properties → Details**. Other targets ignore `-icon`.

### Profiles

| Profile | Assets | Binary | Console window (Windows) | Other |
| --- | --- | --- | --- | --- |
| `dev` | Loose: edit a file in `dist` and run again | Symbols kept | Yes | |
| `debug` | Loose | Symbols kept | Yes | `DebugLog` prints, as with `MOON_DEBUG=1` |
| `release` | `<name>.pak` | Stripped (`-s -w -trimpath`) | No (`-H windowsgui`) | An error at startup also goes to `<name>.log` beside the binary |

`-headless` keeps the console in every profile. Flags win over the profile; `Assert` and the `-debug` debugger work in every profile.

### Pak files

A packed build writes `<name>.pak`, a zip of the assets with their paths (`assets/coin.png`). At startup the runtime opens `<exe name>.pak`, else `data.pak`, from beside the binary, else from the script's folder, with no `OpenPak` call. A loose file with the same path wins over the pak, so you can drop a fixed file next to a release build, and a `data.pak` in a project folder works during development too. Scripts are not packed.

### Missing assets

`moon build` reads every quoted path with an asset extension in the scripts it packages. A path that does not exist prints `moon build: warning: game.mb:12: missing asset "sfx/jump.wav"`; with `-strict` (or `"strict": true` in the profile) the build stops. Paths after a command that writes a file (`SaveScreenshot "shot.png"`, `SaveSlot`, `WriteFile`, `Export*`) and comment lines are not counted. Paths outside the game folder (absolute, or starting with `../`) count as missing: they would not ship.

### moon.json

`moon build` reads the `moon.json` in the script's folder or up to three folders above it ([STUDIO.md](STUDIO.md#moonjson)). It uses `name` (binary and product name), `assets` and `scenes` (folders to package), and from `build`: `outputDir`, `icon`, `version`, `company`, `profile` (the default profile), and `profiles`:

```json
{
  "name": "Star Game",
  "build": {
    "icon": "assets/icon.png",
    "version": "1.2.0",
    "company": "Moon Co",
    "profile": "release",
    "profiles": {
      "release": { "strict": true },
      "server": { "base": "release", "headless": true, "tags": "enet" }
    }
  }
}
```

A profile sets any of `pak`, `strip`, `console`, `debug`, `headless`, `strict` (true or false) and `tags`. One named like a built-in profile changes it; a new one starts from `base` (default `dev`). `moon build main.mb -profile server` then builds a stripped, packed, console server.

## Headless and server builds

`moon run -headless game.mb` (or `MOON_HEADLESS=1`) runs a script with no window and no GL context. A `-headless` build does the same with no flag.

- `Graphics3D`, `Graphics2D`, and `Graphics` open no window; `GraphicsWidth` / `GraphicsHeight` still return the size you asked for
- Entities, transforms, collisions, physics, timers, files, data, networking, and the ECS run as usual; water keeps its levels and waves for buoyancy
- Drawing, post-processing, shaders, GUI, extra windows, screenshots, the profiler overlay, and audio commands do nothing and return 0. A command that needs a window anyway is skipped with one `headless: <command> skipped` line on stderr
- No keys or mouse buttons are ever down
- `Flip` paces the loop at 60 frames a second (`MOON_HEADLESS_HZ` changes it); `WaitTimer` and `Delay` sleep. `End` or `EndGraphics` stops it
- `MOON_SMOKE_FRAMES=n` stops after n frames with `SMOKE frames=n headless`

The examples in `examples/` run headless for 60 frames without a crash.

`imgui.ini` may appear next to the exe after the first ImGui frame. That is user layout, not a redistributable.

## Engine / IDE

```text
go build -o moon.exe ./cmd/moon
go build -o moonsls.exe ./cmd/moonsls
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
