# Debug drawing

A line overlay for looking inside a running 3D scene: collision shapes,
contacts, velocities, joints, recent ray picks, the baked navmesh and agent
paths, light ranges, bounding boxes, vertex normals, entity labels, camera
frustums and sound positions. Scripts can draw their own lines, arrows,
spheres and boxes into the same overlay.

Demo: `examples/debug_draw.mb` (keys 1–9, 0, F, G toggle overlays; T cycles
the depth mode; click to pick).

## Overlays

`DebugDraw what$ [, on]` turns one overlay on (default) or off. Names are
case-insensitive. `DebugDrawEnabled(what$)` reads it back.

| `what$` | Draws | Colours |
| --- | --- | --- |
| `"physics"` | Every collider as its real shape: box, sphere, capsule, cylinder, character capsule. Mesh and convex-hull bodies draw their source mesh's triangles as wireframe (first 6000). Compound bodies and height fields draw their bounding box | Cyan dynamic, blue asleep, orange kinematic, grey static, green sensor/hitbox, yellow character |
| `"contacts"` | The last physics step's contact points (cross) and normals (arrow, 0.5 m) | Red point, yellow normal |
| `"velocity"` | Linear velocity of dynamic and kinematic bodies: an arrow showing a quarter second of travel | Green |
| `"joints"` | Lines from each body to the joint anchor, a cross at the anchor, and a red line when the two bodies disagree on where the anchor is (stretch) | Magenta |
| `"rays"` | `LinePick`, `CameraPick`, `RayPick` and `Raycast` calls: the ray, a cross at the hit, the surface normal (when the physics hit has one). Each stays for `DebugRayTime` seconds (default 1) | Orange hit, grey miss, yellow normal |
| `"nav"` | Baked Detour polygons (border edges bright, shared edges faint), each agent's radius and its remaining path | Cyan mesh, yellow path |
| `"lights"` | Point light range spheres, spot cones (range × cutoff angle), area light rectangles with their facing arrow (length = range, or 1 m), and directional lights as a bundle of arrows a few metres in front of the camera. Disabled lights draw faint | Pale yellow |
| `"bounds"` | Each mesh's local bounding box, transformed (so it turns with the mesh) | White |
| `"normals"` | Vertex normals, at most 4000 per mesh and 20000 in all | Blue |
| `"ids"` | A screen label `#id name` (or the kind) on the 64 nearest entities within 80 m | Yellow on dark |
| `"frustums"` | The view frustum of every camera that is not the active one, far plane cut at 30 m | Pink |
| `"audio"` | `EmitSound` / `EmitSound3D` positions for 1.5 s: a small sphere and the sound's range ring | Teal |
| `"all"` | Every overlay above | |

`EnablePhysicsDebug [on]` is the classic switch: on sets `physics`,
`contacts`, `velocity` and `joints`; off clears them. `SetPhysicsDebug flags`
picks them by bit: 1 shapes, 2 contacts, 4 velocity, 8 joints (15 is all).
`GetPhysicsDebug()` returns the mask.

`DebugDrawNavMesh [nav [, on]]` turns the `nav` overlay on, limited to one
navmesh handle (0 or no argument draws every baked mesh and agent).

## Your own lines

Coordinates are world units, Blitz axes (+Z forward), like `PositionEntity`.
The colour is optional (0–255, or 0–1), default white. `seconds` is optional:
0 (the default) draws the shape for this frame only, so call it every frame
from your loop; a positive value keeps it that long.

| Command | Draws |
| --- | --- |
| `DebugLine x1,y1,z1, x2,y2,z2 [, r,g,b [, seconds]]` | One line |
| `DebugArrow x1,y1,z1, x2,y2,z2 [, r,g,b [, seconds]]` | A line with a head at the second point |
| `DebugSphere x,y,z, radius [, r,g,b [, seconds]]` | Three great circles |
| `DebugBox x,y,z, hx,hy,hz [, r,g,b [, seconds]]` | Axis-aligned box from its centre and half extents |
| `DebugClear` | Drops queued script shapes, rays, contacts and sounds |
| `DebugDrawLines()` | Lines drawn in the last frame (all overlays) |

```basic
DebugDraw "physics", 1
DebugDraw "rays", 1
While Not KeyHit(1)
    hit = CameraPick(cam, MouseX(), MouseY())
    If hit Then DebugSphere(PickedX(), PickedY(), PickedZ(), 0.2, 255, 80, 80)
    DebugArrow(0, 0, 0, 0, 2, 0, 0, 255, 0)
    UpdateWorld
    RenderWorld
    Flip
Wend
```

## Depth

`DebugDrawOnTop mode`: `0` (default) lines are hidden behind geometry like
anything else; `1` draws them over everything; `2` is x-ray: visible parts
normal, hidden parts faint. Each vertex is pulled 0.4 % of its distance
toward the eye, so a wire lying on a surface (a collider on its own mesh)
draws instead of flickering.

## How it draws

All lines of a frame go into one dynamic vertex buffer (position + RGBA)
and one `glDrawArrays(GL_LINES)` call (two in x-ray mode) with a small GLSL
3.30 program, no textures. The overlay is drawn after the scene and the
modern instanced/particle pass, into the scene framebuffer while its depth
buffer is still bound, so lines are depth-tested against the real scene.
That is before post-processing: with post on (the default), the lines are
tonemapped and anti-aliased with the scene, so colours come out a little
softer than the numbers above. Labels are G3N GUI labels like `Text`, drawn
after post. Split-screen viewports draw the overlay once, full screen, from
the main camera.

A frame is capped at 100000 lines; extra lines are dropped. Script shapes
are capped at 50000 queued lines and rays at the 64 most recent.

## Limits

- Physics shapes are drawn from the size the script gave when it made the
  body (`CreateBodyBox`, `CreateBodySphere`, …), at the entity's position
  and rotation. A `SetCharacterShape` change is not reflected.
- Compound bodies show one bounding box, not their parts. Height fields
  show a box.
- Sleep colour needs a backend that reports it: Windows Jolt and the
  software fallback do; the older `-tags joltgo` backend draws sleeping
  bodies as awake.
- Joint anchors are captured in each body's frame the first frame the
  `joints` overlay sees the joint. Turn it on before bodies move for exact
  anchors.
- `LinePick` hits found by the entity-sphere fallback (no physics body) have
  no normal.
- Lines are 1 pixel wide (OpenGL 3.3 core has no wide lines).
