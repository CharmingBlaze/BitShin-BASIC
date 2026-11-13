# Getting started with moonBASIC

moonBASIC is a BASIC for making 3D and 2D games. You write a short program, press F5, and it runs in a window. When it's ready, one more key turns it into a game anyone can download and play.

This page takes you from the download to a game you can share. It covers the Windows SDK.

## 1. Install

1. Unzip `moonBASIC-1.0.0-windows-amd64.zip` somewhere you keep programs, such as `Documents\moonBASIC` or `C:\moonBASIC`.
2. Open the folder and double-click **moonBASIC.exe**.

There is no installer, and nothing is added to the system. To uninstall, delete the folder (and, if you like, the two folders under [Where moonBASIC keeps things](#where-moonbasic-keeps-things)).

The first time you run it, Windows may say *"Windows protected your PC"*. That's because the program is new and not yet signed. Click **More info**, then **Run anyway**.

**You need:** Windows 10 or 11 (64-bit) and a graphics card or chip that supports OpenGL 3.3. That's almost any PC made since 2012. Keep its graphics driver up to date.

### What's in the folder

| | |
| --- | --- |
| **moonBASIC.exe** | The editor. This is the one you open. |
| **Start Here.html** | This page. |
| `docs\` | The guide to the language and every command. |
| `examples\` | Small programs, one idea each. Open one and press F5. |
| `games\` | Two complete games with their source: *Greenwood* and *The Black Veil*. |
| `extras\` | Support for other editors (VS Code). |
| `bin\` | The compiler and the parts the editor uses. You don't need to open it. |

Keep the folder together: moonBASIC.exe looks for `bin\` beside it.

## 2. Your first program

When moonBASIC opens, it shows the start screen.

1. Click **New project**.
2. Pick a template. **Empty 3D** is a good first one. The others are First-person, Third-person, Platformer 2D, Top-down 2D and Empty 2D.
3. Give it a name and click **Create project**. Projects are saved in `Documents\moonBASIC Projects`.
4. Press **F5**. Your program opens in its own window. Press **Esc** in that window to close it.

Now change something. Here is a whole program: a cube that spins until you press Esc.

```basic
Graphics3D(960, 600, 0, 2)
camera = CreateCamera()
light = CreateLight()
cube = CreateCube()
PositionEntity(cube, 0, 0, 5)

While Not KeyDown(1)        ; key 1 is Esc
    TurnEntity(cube, 0.4, 0.7, 0)
    RenderWorld
    Flip
Wend
```

Every game has this shape: set things up, then loop. Each pass through the loop moves things, draws the world (`RenderWorld`), and shows the frame (`Flip`).

To try something without making a project, press **Ctrl+N** for a new tab, type, and press **F5**. **File → Examples** opens any of the examples.

## 3. The editor

| Key | What it does |
| --- | --- |
| **F5** | Run |
| **Shift+F5** | Stop |
| **Ctrl+F5** | Run with the debugger |
| **F9** | Set or clear a breakpoint on this line |
| **F10** / **F11** / **Shift+F11** | While paused: step over, step into, step out. **F5** continues. |
| **F7** | Build a standalone game (see [Share your game](#5-share-your-game)) |
| **Ctrl+N**, **Ctrl+O**, **Ctrl+S**, **Ctrl+W** | New, open, save, close tab |
| **Ctrl+`** | Show or hide the output panel |

- **As you type**, the editor suggests commands and shows what each one takes. Hover over a command to read about it.
- **The output panel** shows what your program prints, and any error with its line number. Click the **line** link beside an error to jump to it.
- **The sidebar** (the **View** menu) has the **Explorer** for your project's files, the **Outline** of the current file, **Commands** to search every command, and the **Examples**.
- **The debugger:** set a breakpoint with F9, press Ctrl+F5, and the program pauses there. You can see every variable, step one line at a time, and add watches.

## 4. Learn the language

- **[The guide](GUIDE.md)** starts here: values, loops, functions, the frame loop, then a 3D scene, a 2D scene, physics, walking, saving. Each section is a short program you can run.
- **[Every command](COMMANDS.md)**, grouped by what it's for.
- **`examples\`**: open one and press F5. Each shows one engine feature: water, terrain, physics, cars, shaders, sound, UI, networking and more.
- **`games\`**: two finished games to read and change. Open `games\hunter\main.mb` and press F5 to play Greenwood.

When you want depth on one area, the topic guides are in `docs\`: graphics, lighting, shadows, terrain, water, weather, physics, vehicles, animation, audio, particles, UI, scenes and networking.

## 5. Share your game

Press **F7** (or **Run → Build package…**). Pick **Windows**, choose an output folder, and click **Build**.

moonBASIC writes a folder like this:

```
Greenwood\
    Greenwood.exe      the game: double-click to play
    Greenwood.pak      your images, sounds and models, packed into one file
    main.mb, ...       your scripts
    libc++.dll         two small files the game needs
    libunwind.dll
```

**Zip that folder and send it.** Whoever gets it unzips it and double-clicks the .exe. They don't need moonBASIC or anything else installed.

### Name, icon and version

A project's `moon.json` sets how the game is packaged. The parts you'll want:

```json
{
  "name": "Greenwood",
  "build": {
    "icon": "icon.png",
    "version": "1.0.0",
    "company": "Your name",
    "profile": "release"
  }
}
```

- **name** is the .exe's name and the name Windows shows for it.
- **icon** is any PNG (square, 256×256 is best). It becomes the .exe's icon.
- **profile** `"release"` packs the assets into one file and runs the game with no console window. Use `"dev"` while testing: it keeps a console window open for messages, and the assets stay loose so you can swap them.

If a release build closes at once, look for `<game name>.log` beside the .exe: it has the error.

**Building for Linux or macOS:** each needs moonBASIC on that system. A Windows build runs on Windows.

## 6. From the command line (optional)

Everything the editor does, `bin\moon.exe` does from a command prompt.

```
bin\moon examples\spinning_cube.mb                 run a program
bin\moon build games\hunter\main.mb -o C:\Out\Greenwood   package a game
bin\moon version
```

`moon build` takes `-profile dev|release`, `-name`, `-icon` and `-version`, which override `moon.json`. See [Packaging](RELEASE.md) for all of them.

To use another editor, `bin\moonsls.exe` is a language server: completion, hover and errors. `extras\vscode` is a ready VS Code extension (in VS Code: **Extensions → … → Install from Location**).

## Where moonBASIC keeps things

moonBASIC never writes into its own folder, so you can keep it anywhere, even somewhere read-only.

| What | Where |
| --- | --- |
| Your projects | `Documents\moonBASIC Projects` (or wherever you create them) |
| Editor settings and recent projects | `%AppData%\moonBASIC` |
| The copy of an unsaved tab that Run uses | `%Temp%\moonBASIC` |

When you run a saved file, a hidden `.moonbasic_run.mb` appears beside it. That's the copy Run uses, so unsaved edits run too.

## When something goes wrong

**The game window doesn't open, or it's black.** Update your graphics driver. moonBASIC needs OpenGL 3.3. On a laptop with two graphics chips, set moonBASIC.exe and your game to use the faster one (Windows Settings → System → Display → Graphics).

**"moon.exe not found".** The `bin\` folder isn't beside moonBASIC.exe. Unzip the whole download again, and keep its folders together.

**"Windows protected your PC" or an antivirus warning.** The programs aren't signed yet. Click **More info → Run anyway**. Some antivirus tools flag any new .exe; you can allow the moonBASIC folder.

**An error with a line number.** Click the **line** link beside it in the output panel to jump there. Look up a command under **View → Commands**.

**"missing asset" when building.** A file named in your script isn't in the project folder. Keep images and sounds in the project's `assets\` folder, and write paths like `"assets/jump.wav"`.

**A built game says it can't find a file.** Send the whole folder, not just the .exe. The `.pak` and the two DLLs must stay beside it.
