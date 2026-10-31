# Assets

## Meshes (`LoadMesh` / `LoadAnimMesh`)

| Format | Notes |
| --- | --- |
| `.glb` / `.gltf` | Preferred. Clips need `LoadAnimMesh`. Physical materials use `mbphysical`. `AttachToBone child, mesh, "JointName"` parents a prop to a named node in that scene graph (`docs/PHYSICS.md`). |
| `.obj` | + `.mtl` if present |
| `.dae` | Collada |
| `.fbx` | Mesh triangles, normals, vertex UVs, and skin clusters (indexes, weights, limb translation). Z-up files become Y-up. A unit scale of 50 or more (centimetres) is converted to metres. Moving a limb node drags its weighted vertices. Cameras and animation clips are not read |

**Large levels.** `LoadLevel(file$ [, cell#])` loads a file and merges its static meshes into one mesh per material per 64 m cell. A 1500-node glTF city becomes 48 draws, and the `city.mb` example's 20,000 script-built meshes become 676. G3N pays per mesh, not per triangle, so this is what keeps a big level fast. Cells are frustum culled and hidden past the fog. `MergeLevel(root)` does the same to anything already built; doors, pickups, and anything with a body, collider, animation, or name stay separate.

`OptimizeMesh entity` reorders that mesh for the vertex cache, then for overdraw, and packs vertices in draw order. `SimplifyMesh entity [, ratio]` reduces the triangle count with quadric error. Both use the Go packages copied into `internal/meshoptim` and `internal/meshsimplify` from [m31labs.dev/gosx](https://pkg.go.dev/m31labs.dev/gosx/assetpipe/meshoptim). They do not link the meshoptimizer C++ library.

## Textures

PNG / JPEG via `LoadTexture` / `LoadImage`. Heightmaps: PNG/JPEG grayscale (`LoadHeightmap`, `GenerateHeightmap`, `SaveHeightmap`).

`LoadTexture` also accepts `.dds` (DXT1, DXT3, DXT5, or 32-bit uncompressed) and `.ktx` / `.ktx2` when the image is S3TC/BC1–BC3. KTX2 Zstandard and ZLIB supercompression are inflated first. BasisLZ is not transcoded. The blocks stay compressed on the GPU (`EXT_texture_compression_s3tc` on OpenGL 3.3).

## Audio (Oto)

`.wav` / `.ogg`. No OpenAL.

## Packaging

`moon build game.mb -o dist` copies the interpreter, the `.mb` files, quoted asset paths, `assets/` and `scenes/`, and natives from `third_party/$GOOS`. It warns about quoted paths that do not exist (`-strict` fails the build). `-pak` or `-profile release` puts the assets in one `<name>.pak` zip that the game opens at startup; loose files still win over it. Zip `dist` and run from that folder. Details: [RELEASE.md](RELEASE.md).

## Scene files

- `LoadScene "setup.mb"` — run setup (no `Flip` / `End`).
- `SaveScene "level.scene.json"` / `LoadScene "level.scene.json" [, additive]` — scene format v2: stable uids, names, tags, materials, bodies, lights, cameras, environment, and prefab references with overrides. [SCENES.md](SCENES.md) is the reference.
- `SceneSave` — the older runtime dump (primitive `kind`, `LoadMesh` `src`, parent, transform, tint, visibility, lights, cameras, controller and body velocities, joints). Good for save games.
- `LoadScene "dump.json"` — rebuilds a `SceneSave` dump: tagged primitives, meshes from `src`, lights, camera frustum. Pre-1.0 dumps without `kind` still spawn cubes.
