# Assets

## Meshes (`LoadMesh` / `LoadAnimMesh`)

| Format | Notes |
| --- | --- |
| `.glb` / `.gltf` | Preferred. Clips need `LoadAnimMesh`. Physical materials use `mbphysical`. `AttachToBone child, mesh, "JointName"` parents a prop to a named node in that scene graph (`docs/PHYSICS.md`). |
| `.obj` | + `.mtl` if present |
| `.dae` | Collada |
| `.fbx` | **Not loaded.** Export glTF/OBJ |

No meshopt, no Basis/KTX2, no extra C++ codec tree.

## Textures

PNG / JPEG via `LoadTexture` / `LoadImage`. Heightmaps: PNG/JPEG grayscale (`LoadHeightmap`, `GenerateHeightmap`, `SaveHeightmap`).

## Audio (Oto)

`.wav` / `.ogg`. No OpenAL.

## Packaging

`bs build game.bb -o dist` copies the interpreter, the `.bb`, quoted asset paths, `assets/`, and natives from `third_party/$GOOS`. Zip `dist` and run from that folder. Details: [RELEASE.md](RELEASE.md).

## Scene files

- `LoadScene "setup.bb"` — run setup (no `Flip` / `End`).
- `SaveScene` / `SceneSave` — JSON of primitive `kind`, `LoadMesh` `src`, parent, transform, tint, visibility. Lights and cameras included.
- `LoadScene "dump.json"` — rebuilds tagged primitives, reloads meshes from `src`, restores lights and camera frustum. Pre-1.0 dumps without `kind` still spawn cubes.
