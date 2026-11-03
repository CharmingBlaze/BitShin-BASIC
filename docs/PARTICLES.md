# Particles

There are two kinds of 3D emitter. Both take the same `Emitter*` setters (see the Particles section of [COMMANDS.md](COMMANDS.md)).

| | `CreateEmitter` (CPU) | `CreateGPUEmitter` (GPU) |
| --- | --- | --- |
| Count | Tens to a few hundred | 100k+ per emitter |
| Each particle | A G3N mesh, moved by Go each frame | A record in a GL buffer, moved by a shader |
| Looks | `soft`, `cube`, `streak`, `flake` | `soft`, `streak`, `flake` (no cubes) |
| Extras | Weather rain and snow use it | Size/speed/life randomness, curl-noise turbulence, ground bounce, additive blend |
| Count read-back | Exact | Estimated from spawn times |
| 2D | `CreateEmitter2D` | 3D only |

Use the CPU emitter for a few dozen sparks or tumbling cubes. Use the GPU emitter for fountains, sparks showers, snow, dust, and magic that need thousands of particles.

## Switching

Change only the create call:

```basic
em = CreateGPUEmitter(100000)      ; was CreateEmitter()
PositionEmitter(em, 0, 2.4, 0)
EmitterRate(em, 30000)
EmitterLife(em, 3, 4)               ; second value: random life up to 4 s
EmitterSpeed(em, 9, 0.12)           ; second value: +-12% speed
EmitterVelocity(em, 0, 1, 0)
EmitterCone(em, 16)
EmitterGravity(em, 0, -9.8, 0)
EmitterSize(em, 0.07, 0.03)
EmitterColor(em, 150, 210, 255, 0.9, 40, 90, 255, 0.1)
EmitterBlend(em, "add")
EmitterGround(em, 1, 0, 0.25, 0.3)  ; bounce off y = 0
```

The id is an entity, so `PositionEntity`, `EntityParent`, `HideEntity`, and `FreeEntity` work on it. A hidden emitter stops spawning; live particles finish. GPU-only setters (`EmitterBlend`, `EmitterTurbulence`, `EmitterGround`, `EmitterWeatherWind`, `EmitterSizeVar`) do nothing on a CPU emitter.

## Spawn shapes and looks

`EmitterShape` takes either a spawn volume or a look; they combine.

- `"point"` (default): the emitter origin, jittered by `EmitterArea x, y, z`.
- `"box", x, y, z`: anywhere in a box of those half extents.
- `"sphere", r`: anywhere inside a sphere.
- `"disc", r`: a flat disc on the XZ plane.
- `"soft"`: round soft sprite. `"streak" [, stretch]`: stretched along the velocity, for sparks and rain. `"flake"`: spinning, drifting flake.

`EmitterCone` spreads the direction from `EmitterVelocity`: 0 is a jet, 180 a hemisphere, 360 every way. `EmitterParticle e, tex` draws a loaded texture instead of the soft disc.

## How it runs

Each emitter has two GL buffers of `max` particles. Every frame one vertex-shader pass reads one buffer and writes the other through transform feedback (OpenGL 3.0, so every GL 3.3 machine has it). The CPU only says which slots of a ring to respawn: rate x frame time, plus any `EmitterBurst`. When the ring is full, the oldest particle is replaced, like the CPU emitter dropping its oldest. Spawns in one frame are spread along the frame and along the emitter's path, so a moving emitter leaves a trail, not clumps.

The step applies gravity, `EmitterWind` (plus weather wind when asked), drag, curl-noise turbulence, the ground plane, and up to four mesh boxes from `EmitterCollide`. Particles are drawn after the scene as instanced camera-facing quads, depth-tested against the scene but not written to depth. They fog like the scene. A floating-origin shift (`SetPlayer` past 4 km) moves live particles with the world.

`EmitterSoft e, on` fades a particle where it meets the scene depth, so a sprite crossing a wall softens instead of cutting a hard edge. It stays off when the post pass has no depth texture. Soft is on by default for a new GPU emitter.

`EmitterSort e, on` reorders the live particles far-to-near before drawing. The read-back stops at 8192 particles. A larger emitter skips the sort for that frame. Sorting is off until you ask. Additive emitters do not need it.

`EmitterCollide e, entity` pushes particles out of that entity's box. Four boxes are kept. A fifth call drops the oldest. `EmitterClearCollide e` clears them. The test is a box, not the mesh triangles, and it runs in the simulation shader.

`EmitterPreset e, name$` fills a GPU or CPU emitter with one recipe: `fire`, `smoke`, `sparks`, `explosion`, `dust`, `magic`, `splash`, `debris`. You can still change any `Emitter*` value afterwards.

`CreateTrail entity, points` is separate from emitters. It keeps the last positions of that entity (2 to 64) and draws them as a short-lived debug line. It is a polyline, not a camera-facing ribbon.

`GPUParticleMs()` reads a GL timer query of the step and the draw (two frames old, so it never stalls). On a Radeon 780M at 1280x720, 100k particles cost about 0.4 ms as a fountain, 1.2 ms as snow, and 1.5–3 ms for big overlapping sparks or dust, where blending, not the step, is the cost.

## Limits

- Compute shaders are not used, even when the GL has them: transform feedback is the only path, so one code path runs everywhere.
- Soft particles compare against the post depth buffer. They do not sort against each other. `EmitterSort` does the sort, and only up to 8192 particles.
- Collision is one ground plane plus four boxes. Particles still pass through triangle meshes and terrain.
- Particles do not cast shadows or take scene lights.
- `EmitterCount` on a GPU emitter is an estimate from spawn times and life, not a read-back.
- `CreateTrail` draws a debug line. It is not a lit ribbon and it does not survive `DebugClear` as its own mesh.

Demo: `examples/gpu_particles.mb` (keys 1–5 switch fountain, sparks, snow, magic, dust; Space bursts).
