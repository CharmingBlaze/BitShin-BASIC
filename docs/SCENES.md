# Scenes and prefabs

A scene file (`.scene.json`) is a level: every entity with its transform, look, physics, and light, plus the environment around them. The scene editor reads and writes it, and a game loads it with one line:

```basic
Graphics3D(960, 600)
LoadScene("levels/forest.scene.json")
door = FindEntity("Door")
```

A prefab is a scene file too. An entity of kind `prefab` places one, with per-entity overrides. Prefabs nest.

`SceneSave` still writes the older runtime dump (velocities, controllers, joints) for save games, and `LoadScene` still reads it. This page is about format v2. See [examples/scene_load.mb](../examples/scene_load.mb) and [examples/scenes/demo.scene.json](../examples/scenes/demo.scene.json).

## Commands

| Command | What it does |
| --- | --- |
| `SaveScene path$` | Write the scene as v2 |
| `LoadScene path$ [, additive]` | Load a v2 scene. Without `additive` the scene's entities are freed first and the environment is applied. `.mb` files run as before; `SceneSave` dumps are added as before |
| `SavePrefab name$, e` | Save `e` and its children as a prefab, root at the origin. A name with no extension goes to `prefabs/name.json` |
| `InstantiatePrefab(path$ [, x, y, z, parent])` | Build a prefab (or any scene file) as an instance and return its root. `SpawnPrefab`, `LoadPrefab`, and `SceneInstance` are the same command |
| `EntityPrefab$(e)` | The prefab an instance root came from, or "" |
| `FindEntity(name$)` | First entity with that name (any case) |
| `FindEntityByTag(tag$ [, n])` / `CountEntitiesByTag(tag$)` | Tagged entities, the nth from 1 |
| `EntityTag$(e [, n])` / `EntityTags$(e)` / `EntityHasTag(e, tag$)` | Read tags |
| `SetEntityTag e, tags$` / `AddEntityTag e, tag$` / `RemoveEntityTag e, tag$` | Change tags (comma separated) |
| `EntityUID$(e)` / `EntityByUID(uid$)` | Stable uid, made on first use and kept by save and load |
| `EntityScript$(e)` / `SetEntityScript e, s$` | A free-form string the game reads, saved with the entity |

Paths in `src` (meshes, textures, prefabs) are as written. A prefab `src` is looked up next to the file that names it, then next to the script, then in `prefabs/`; `.prefab.json` and `.json` are tried when there is no extension.

## The file

```json
{
  "version": 2,
  "environment": {
    "ambient": [70, 76, 92], "ambientIntensity": 1,
    "clear": [24, 28, 40],
    "fog": {"mode": "linear", "color": [120, 130, 150], "near": 25, "far": 120, "density": 0.02},
    "sky": "default",
    "ibl": {"on": true, "intensity": 1},
    "sun": [-0.35, 0.62, 0.7],
    "quality": "high",
    "post": {"on": true, "tonemap": "neutral", "exposure": 1.1, "bloom": 0.25, "lut": "warm", "lutAmount": 0.5}
  },
  "entities": [
    {"uid": "cam", "name": "Camera", "kind": "camera", "pos": [0, 6, -14], "rot": [18, 0, 0],
     "camera": {"fov": 60, "near": 0.1, "far": 1000, "active": true, "clear": [24, 28, 40]}},
    {"uid": "sun", "kind": "light", "rot": [50, 35, 0],
     "light": {"type": "directional", "color": [255, 244, 228], "intensity": 1.2, "shadows": true}},
    {"uid": "crate1", "name": "Crate", "tags": ["crate"], "script": "kick=4", "kind": "cube",
     "pos": [-1, 4, -2], "rot": [0, 20, 0], "scale": [0.5, 0.5, 0.5], "visible": true,
     "color": [170, 120, 70, 255],
     "material": {"pbr": true, "metallic": 0, "roughness": 0.7, "textures": {"albedo": "assets/stone.png"}},
     "body": {"type": "dynamic", "shape": "box", "size": [0.5, 0.5, 0.5], "mass": 2, "friction": 0.6}},
    {"uid": "lampR", "kind": "prefab", "src": "lamp.prefab.json", "pos": [5, 1.2, 5],
     "overrides": {"glow": {"light": {"color": [130, 180, 255]}}, "bulb": {"color": [170, 200, 255, 255]}}}
  ]
}
```

Colours are 0–255. Transforms are local to the parent, in Blitz axes (+Z forward) and degrees: `rot` is pitch, yaw, roll. Missing fields keep the default of a fresh entity, so a hand-written file can be short.

### Entity fields

| Field | Meaning |
| --- | --- |
| `uid` | Stable id, unique in the scene. Kept by save and load; made when missing. Instance members have `"<instance uid>/<prefab uid>"` |
| `name` | `EntityName$` |
| `tags` | List of strings (`EntityTag$`, `FindEntityByTag`) |
| `kind` | `cube` `sphere` `cylinder` `cone` `plane` `quad` `disk` `torus` `capsule` `pyramid` `wedge` `tube` `pivot` `mesh` `light` `camera` `arealight` `prefab`. Missing is `cube`, as in old dumps |
| `src` | Mesh file for `mesh`, scene file for `prefab` |
| `parent` | uid of the parent. Rows may come in any order |
| `pos` `rot` `scale` | Local transform |
| `visible` | Shown or hidden |
| `color` | `[r, g, b, a]` tint (`EntityColor`, `EntityAlpha`) |
| `material` | See below (mesh kinds) |
| `body` | See below |
| `light` | `type` (`directional` `point` `spot` `ambient`), `color`, `intensity`, `range` (point, spot), `cone` `[inner, outer]` degrees (spot), `shadows` |
| `area` | Area light: `size` `[w, h]`, `color`, `intensity`, `range` (0 unlimited), `twoSided` |
| `camera` | `fov`, `near`, `far`, `active`, `clear` (the clear colour, with the active camera) |
| `script` | Any string. The engine stores it; the game reads it with `EntityScript$` |
| `overrides` | Prefab only: per-member property overrides |

### Material

| Field | Meaning |
| --- | --- |
| `pbr` | PBR (`SetMaterialPBR`) or the default lit material |
| `metallic` `roughness` `ao` | 0–1 |
| `emissive` | `[r, g, b]`, 0–255; can go above 255 for bloom |
| `shading` | `pbr`, `unlit`, or `toon` (`SetMaterialShading`) |
| `toon` | `{"bands", "rim", "spec"}` (`SetToon`) |
| `ibl` | -1 world default, 0 off, 1 on |
| `fx` | `EntityFX` flags: 1 fullbright, 8 no fog, 16 two-sided |
| `textures` | Paths for `albedo`, `normal`, `metalRough`, `emissive`, `ao` |
| `castShadows` `receiveShadows` | `EntityCastShadow`, `EntityReceiveShadow` |

Only fields that differ from the entity's current material are applied, so a loaded glTF keeps its own materials unless the file changes them.

### Body

| Field | Meaning |
| --- | --- |
| `type` | `none`, `static`, `kinematic`, or `dynamic` (the default) |
| `shape` | `box`, `sphere`, `capsule`, `cylinder`, `mesh` (static triangle mesh), `convex`, `compound` (the children's colliders), or `auto` (box or sphere sized from the mesh, as `Collide`) |
| `size` | Half extents `[x, y, z]`. Capsule and cylinder use `y` as the half height |
| `radius` | Sphere, capsule, and cylinder radius |
| `mass` | kg, dynamic bodies |
| `friction` `restitution` | Surface |
| `layer` | `SetCollisionLayer` |
| `sensor` | Reports contacts without colliding |
| `material` | A physics material name (`SetMaterial e, "ice"`) |

The body is built where the entity ends up after parenting, so a body inside a prefab lands with the instance.

### Environment

`ambient` and `ambientIntensity`, `clear`, `fog` (`mode` is `off`, `linear`, `exp`, or `exp2`), `sky` (a `SetSkyPreset` name) or `skybox` (a `CreateSkyBox` path prefix), `ibl`, `sun` (`SetSunDirection`), `quality` (`low`, `medium`, `high`, `ultra`, `auto`), and `post` (`on`, `tonemap` `reinhard`/`neutral`/`aces`/`none`, `exposure`, `bloom`, `lut`, `lutAmount`). An additive load leaves the environment alone.

## Prefabs

A prefab file is a scene file whose rows are one entity and its children. `SavePrefab` writes the root at the origin; the instance's own `pos`, `rot`, and `scale` place it.

```json
{"version": 2, "prefab": true, "entities": [
  {"uid": "post", "name": "Lamp", "kind": "cylinder", "scale": [0.12, 1.2, 0.12],
   "body": {"type": "static", "shape": "cylinder", "size": [0.12, 1.2, 0.12], "radius": 0.12}},
  {"uid": "bulb", "kind": "sphere", "parent": "post", "pos": [0, 1.05, 0]},
  {"uid": "glow", "kind": "light", "parent": "post", "pos": [0, 1.3, 0], "light": {"type": "point", "range": 9}}
]}
```

Instancing it as `lampR` makes entities with uids `lampR` (the root), `lampR/bulb`, and `lampR/glow`. A file with several root rows gets a pivot as its root.

### Overrides

`overrides` maps a member's uid inside the prefab to the fields that change:

```json
{"uid": "lampR", "kind": "prefab", "src": "lamp.prefab.json", "pos": [5, 1.2, 5],
 "overrides": {
   "bulb": {"color": [170, 200, 255, 255]},
   "glow": {"light": {"color": [130, 180, 255]}},
   ".": {"material": {"metallic": 0.2}},
   "lamp2/bulb": {"scale": [2, 2, 2]},
   "glow2": {"deleted": true}
 }}
```

- `"."` is the root. Its `name`, `tags`, `script`, `visible`, and transform are on the instance row itself.
- `"a/b"` is member `b` of the nested prefab instance `a`.
- `{"deleted": true}` removes that member.
- A field set to `null` clears it (a body, tags, the script).

`SaveScene` writes an instance as its reference, never a copy: the overrides are what differs from the prefab file at load time, so a change to the prefab file reaches every instance next load. Entities added under a member are saved as ordinary rows with that member's uid as `parent`.

### Nested prefabs and variants

A prefab row may appear inside a prefab file, to any depth. A prefab that includes itself, directly or through others, stops with an error at the repeat; the rest of the scene loads. A missing prefab becomes an empty pivot that keeps its `src` and `overrides`, so saving does not lose it.

A variant is a prefab file whose root row is itself a prefab instance with overrides, for example a blue lamp:

```json
{"version": 2, "entities": [
  {"uid": "v", "kind": "prefab", "src": "lamp", "overrides": {"bulb": {"color": [0, 0, 255, 255]}}}
]}
```

`SavePrefab` of an instance root writes exactly this.

## Go API (editor)

The scene editor in `internal/runtime` calls these on `*World`:

| Function | What it does |
| --- | --- |
| `entityJSON(id int) map[string]any` | One entity as v2 JSON (an instance root as its reference and overrides); nil if the format cannot hold it |
| `applyEntityJSON(m map[string]any, keepUID bool) (int, error)` | With `keepUID` and a uid in the scene: update that entity in place to match `m` (same id; missing tags, script, or body are cleared), for undo and redo. A changed kind, mesh `src`, light type, or prefab `src` rebuilds it under the same uid, keeps its other children, and returns the new id. Otherwise it creates a new entity (a fresh uid without `keepUID`) |
| `sceneV2() map[string]any` | The whole scene document |
| `loadSceneV2(root map[string]any, additive bool) error` | Build a document (v1 dumps go through the old loader). Missing assets and prefab errors come back joined; the rest still loads |
| `entityByUID(uid string) int` | Entity id, or 0 |
| `entityUID(id int) string` | The uid, made on first use |

## Limits

- Primitives load at their default size (`CreateCube`'s 2 m cube, a 20 m plane, …); the size a script gave `CreateBox(w, h, d)` or `CreateSphere(segments)` is not stored. Use `scale`.
- Terrain, water, particles, cloth, ropes, sprites, joints, and controllers are not written. `SceneSave` still keeps joints and controllers.
- A dynamic body's rotation is saved from the entity's pitch, yaw, and roll, which the physics step does not update.
- A member's `kind` or `src` cannot be overridden; replace the member instead.
