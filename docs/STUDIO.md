# Moon Studio projects

Moon Studio is the moonBASIC IDE (`ide/`). A **project** is a folder with a `moon.json` at its root. With a project open, F5 runs the project's main script instead of the current tab, the Explorer shows the project folder, and Build packages the main script into the project's output folder.

## Start screen

The start screen opens with the IDE, and from **File → New project…** or **File → Open project… / Recent**:

- **New project** picks a template, a name, and a location (default `Documents\moonBASIC Projects`), then writes the project and opens it.
- **Open project** picks a folder. If it has no `moon.json`, the screen offers to write a default one (main is `main.mb`, or the first `.mb` file at the root).
- **Recent projects** lists the last ten opened projects, newest first. The list is `recentProjects` in `.ide_settings.json` next to the IDE. Missing folders show as missing; the bin icon removes a row.
- **Continue without a project** keeps the old single-file behaviour: F5 runs the active tab.

**File → Close project** goes back to the workspace tree and the start screen. **File → Project settings…** (or the gear in the Explorer header) edits `moon.json`.

## Templates

Every template writes `moon.json`, `main.mb`, and `assets/` (with a README). The 3D templates also write `scenes/main.scene.json` with a ground, a sun, a crate, and a pillar.

| Template | What `main.mb` does |
| --- | --- |
| Empty 3D | Camera, sun, ground, two props; arrow keys orbit the camera |
| Empty 2D | `Graphics2D` window with a bouncing ball and HUD text |
| First-person | `CreateFPSController` / `UpdateFPS`: WASD, mouse look, jump, sprint, crouch |
| Third-person | `CreateTPSController` / `UpdateTPS` with a spring-arm camera orbiting with the mouse |
| Platformer 2D | Chipmunk `CreateCharacter2D` with a floor, a ledge, and a one-way platform |
| Top-down 2D | WASD over a 2000×2000 world of sprites; camera 0 follows with `CameraTarget2D` |

Each template runs as written: `MOON_HIDDEN=1 MOON_NO_GRAB=1 MOON_SMOKE_FRAMES=90 moon run main.mb` exits 0 with no errors.

The 3D templates build their scene in code. The same objects are in `scenes/main.scene.json`; once `LoadScene` reads version-2 scene files, the code block can be replaced with `LoadScene("scenes/main.scene.json")` (the comment in `main.mb` says where).

## moon.json

Every field is optional. A missing field gets its default; an invalid one is replaced and the IDE prints a `moon.json:` line in the Output panel. Paths are relative to the project root and may not leave it.

```json
{
  "name": "MyGame",
  "main": "main.mb",
  "window": { "width": 1280, "height": 720, "title": "MyGame" },
  "quality": "auto",
  "scenes": "scenes",
  "assets": "assets",
  "build": { "outputDir": "dist", "icon": "", "version": "1.0.0", "company": "" }
}
```

| Field | Default | Meaning |
| --- | --- | --- |
| `name` | folder name | Shown in the title bar, Explorer, and recent list |
| `main` | `main.mb` | The script F5 (and Ctrl+F5, debug) runs. **Run → Run current file** still runs the active tab |
| `window.width` / `height` | 1280 / 720 (960 / 540 for 2D templates) | 160–7680 / 120–4320. Templates write them into `Graphics3D` / `Graphics2D` |
| `window.title` | `name` | Templates write it into `SetWindowTitle` |
| `quality` | `auto` | `auto`, `low`, `medium`, `high`, `ultra`; 3D templates call `SetGraphicsQuality` with it |
| `scenes` | `scenes` | Where **New scene** puts `.scene.json` files |
| `assets` | `assets` | Models, textures, sounds; load them as `"assets/…"` |
| `build.outputDir` | `dist` | The Build dialog's default output folder |
| `build.icon`, `version`, `company` | `""`, `1.0.0`, `""` | Kept for packaging; `moon build` does not stamp them into the binary yet |

The engine does not read `moon.json`. Changing the window or quality fields changes what new templates write, not an existing `main.mb`.

When F5 runs the project, an open tab of the main script with unsaved edits runs as it is in the editor; otherwise the file on disk runs. Breakpoints apply when the active tab is the main script. The program's working folder is the main script's folder.

## Scenes

- `.scene.json` files show with a clapperboard icon in the Explorer.
- **New scene** (File menu, the Explorer header, or the Scene panel) writes an empty version-2 scene, `{"version": 2, "environment": {}, "entities": []}`, opens it, and launches the scene editor.
- **Double-click** a scene in the Explorer, or press **Edit Scene** in the Scene panel, to run `moon edit <file> -project <root>`. One editor window per file: asking again raises the open one.
- The editor's output streams to the Output panel as `[editor] …`. `@editor {"event":"saved","file":"…"}` reloads the Scene panel and any unedited tab of that file; `@editor {"event":"play"}` runs the project like F5.
- If `moon edit` exits with an error before sending any `@editor` line (for example a `moon.exe` built before the scene editor existed), the Output panel says the editor did not start and how to rebuild `moon.exe`.
- The Scene panel lists the active JSON tab, or the last scene opened, as a tree: each entity's name and kind under its `parent`, with its position (`position: [x, y, z]`, or `x`/`y`/`z` in older `SaveScene` dumps).
