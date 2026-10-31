# Assets

## Meshes (`LoadMesh` / `LoadAnimMesh`)

| Format | Notes |
| --- | --- |
| `.glb` / `.gltf` | Preferred. Clips need `LoadAnimMesh`. Physical materials use `mbphysical`. `AttachToBone child, mesh, "JointName"` parents a prop to a named node in that scene graph (`docs/PHYSICS.md`). |
| `.obj` | + `.mtl` if present |
| `.dae` | Collada |
| `.fbx` | Mesh triangles, normals, vertex UVs, and skin clusters (indexes, weights, limb translation). Z-up files become Y-up. A unit scale of 50 or more (centimetres) is converted to metres. Moving a limb node drags its weighted vertices. Cameras and animation clips are not read |
| `.stl` | ASCII or binary. Facet normals (computed when zero); corners that share a position and normal are welded, so edges stay sharp. No UVs or colours |
| `.ply` | ASCII or binary little-endian. Positions, normals (`nx ny nz`), UVs (`u v` / `s t`), and vertex colours (`red green blue`, uchar or float). Triangle, quad, and polygon faces are fanned. Vertex colours show under the lit shader (the default when shadows are on). Big-endian and point clouds are not read |

**Large levels.** `LoadLevel(file$ [, cell#])` loads a file and merges its static meshes into one mesh per material per 64 m cell. A 1500-node glTF city becomes 48 draws, and the `city.mb` example's 20,000 script-built meshes become 676. G3N pays per mesh, not per triangle, so this is what keeps a big level fast. Cells are frustum culled and hidden past the fog. `MergeLevel(root)` does the same to anything already built; doors, pickups, and anything with a body, collider, animation, or name stay separate.

`OptimizeMesh entity` reorders that mesh for the vertex cache, then for overdraw, and packs vertices in draw order. `SimplifyMesh entity [, ratio]` reduces the triangle count with quadric error. Both use the Go packages copied into `internal/meshoptim` and `internal/meshsimplify` from [m31labs.dev/gosx](https://pkg.go.dev/m31labs.dev/gosx/assetpipe/meshoptim). They do not link the meshoptimizer C++ library.

## Textures

PNG, JPEG, BMP, TGA, WebP, and (static) GIF via `LoadTexture` / `LoadImage`, and anywhere else an image is read (heightmaps, LUT strips, skybox faces, palettes). TGA covers uncompressed and RLE truecolor (15/16/24/32-bit), greyscale, and colour-mapped files, either origin; a 32-bit TGA whose alpha is all zero loads opaque. WebP is lossy or lossless. Heightmaps: PNG/JPEG grayscale (`LoadHeightmap`, `GenerateHeightmap`, `SaveHeightmap`).

`LoadTexture` also accepts `.dds` (DXT1, DXT3, DXT5, or 32-bit uncompressed) and `.ktx` / `.ktx2` when the image is S3TC/BC1–BC3. KTX2 Zstandard and ZLIB supercompression are inflated first. BasisLZ is not transcoded. The blocks stay compressed on the GPU (`EXT_texture_compression_s3tc` on OpenGL 3.3).

## Audio (Oto)

`.wav` / `.ogg` / `.mp3` / `.flac`, resampled to 44.1 kHz stereo. No OpenAL.

`LoadSound` decodes the whole file to PCM so it can start instantly and overlap. `LoadMusic` streams: it checks the file decodes, then each `PlayMusic` decodes from disk (or the file extracted from an `OpenPak` archive) a block at a time while it plays, so a long track never sits in memory as PCM. Music loops by reopening the file at its end; `SetMusicVolume`, `StopMusic`, and `SetSoundPitch` work as before.

## Packaging

`moon build game.mb -o dist` copies the interpreter, the `.mb`, quoted asset paths, `assets/`, and natives from `third_party/$GOOS`. Zip `dist` and run from that folder. Details: [RELEASE.md](RELEASE.md).

## Scene files

- `LoadScene "setup.mb"` — run setup (no `Flip` / `End`).
- `SaveScene "level.scene.json"` / `LoadScene "level.scene.json" [, additive]` — scene format v2: stable uids, names, tags, materials, bodies, lights, cameras, environment, and prefab references with overrides. [SCENES.md](SCENES.md) is the reference.
- `SceneSave` — the older runtime dump (primitive `kind`, `LoadMesh` `src`, parent, transform, tint, visibility, lights, cameras, controller and body velocities, joints). Good for save games.
- `LoadScene "dump.json"` — rebuilds a `SceneSave` dump: tagged primitives, meshes from `src`, lights, camera frustum. Pre-1.0 dumps without `kind` still spawn cubes.

## Hot reload

While a game runs from the IDE (`MOON_IDE=1`) or with `moon run game.mb -watch`, the engine checks the files it loaded about twice a second, between frames, and reloads the ones that changed. Built and compiled games leave it off. `SetHotReload on` turns it on or off from the script, and `HotReloadEnabled()` reports it.

| Asset | What reloads |
| --- | --- |
| `LoadTexture` | The image is decoded into the same texture, so every entity, material, and sprite using the handle shows it |
| `LoadMesh` | The entity keeps its handle, transform, visibility, parent, child entities, colour, alpha, texture, and `SetShader` program. Other material settings come from the file again. Physics shapes and `LoadAnimMesh` entities are not rebuilt |
| `LoadImage` | Redrawn in place; in 2D, a resized image is swapped into its sprites and emitters |
| `LoadShader` / `CreateShader` with files | Both files are read and validated again; bound entities switch to the new program |
| The script and its `Include` files | Functions are replaced, as `ReloadScripts` does. The main loop that is already running keeps going. Under the IDE the saved tab is watched, not the temp run copy |

Each reload prints one line, such as `HotReload: texture gfx/crate.png`. A file that fails to load prints the error and the old asset stays, so the game keeps running; fix the file and save again. Files read from a pak (`OpenPak`) are not watched.

`OnAssetReloaded "fn"` calls `Function fn(path$)` after each reload, and `AssetChanged(path$)` returns True once after that file reloaded, for state the script derives from an asset. `ReloadTexture(tex [, file$])`, `ReloadMesh(ent)`, and `ReloadScripts` do the same by hand.
