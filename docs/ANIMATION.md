# Animation blend trees

Skeletal and node clips come from `LoadAnimMesh` (`.gltf` / `.glb`). The full command list is in [COMMANDS.md](COMMANDS.md#mesh-animation-gltf). This page covers parameter-driven blending and crossfades.

## Pose stack

Each frame an animated entity is posed in this order:

1. **Base pose**: the playing blend tree, or the current clip (mixed with the previous clip by `SetAnimBlend`).
2. **Crossfade**: while `CrossfadeAnim` or `PlayBlendTree e, tree, fade#` runs, the old animation keeps playing and is mixed out with a smoothstep curve.
3. **Layers**: `SetAnimLayer` overrides and `SetAnimAdditive` deltas on top.

Every clip is sampled from the bind pose, so a node one clip does not key is not left at another clip's value.

## 1D locomotion

```basic
hero = LoadAnimMesh("assets/walker.gltf")
loco = CreateBlendTree(hero, 1)
AddBlendClip(loco, "idle", 0)     ; threshold = speed in m/s
AddBlendClip(loco, "walk", 1.5)
AddBlendClip(loco, "run", 4.0)
PlayBlendTree(hero, loco)

; each frame
SetBlendParam(loco, speed)
```

The two thresholds either side of the parameter share the weight linearly. Below the lowest or above the highest threshold, that end clip gets all of it. Thresholds can be added in any order.

## 2D strafe

```basic
strafe = CreateBlendTree(hero, 2)
AddBlendClip(strafe, "idle", 0, 0)
AddBlendClip(strafe, "walk", 0, 1.5)
AddBlendClip(strafe, "strafe_left", -1.5, 0)
AddBlendClip(strafe, "strafe_right", 1.5, 0)
SetBlendParam(strafe, sideSpeed, forwardSpeed)
```

2D trees use gradient band interpolation. The weights always sum to 1, a parameter exactly on a clip's point plays only that clip, and outside the points the nearest edge clips win.

## Phase sync

Trees are synced by default. All clips share one normalized phase (0–1), and the phase advances at the weighted average cycle length. A 1 s walk and a 0.6 s run therefore stay in step at every blend, with no foot sliding or double steps. `SetBlendTreeSpeed tree, speed#, 0` lets each clip run on its own clock instead, for clips that are not cycles. `BlendTreePhase(tree)` returns the shared phase, which you can use for footstep sounds.

## Crossfades

```basic
CrossfadeAnim(hero, "wave", 0.35)   ; into a clip
PlayBlendTree(hero, loco, 0.35)     ; back into the tree
```

The source keeps animating during the fade. Starting a new fade while one is still running freezes the current mixed pose and fades from that, so the pose never pops. `CrossfadeProgress(hero)` returns 0–1, and `Animate` cancels any tree or fade.

## Limits

- Blending is local-transform lerp/nlerp per node, with the same quality as `SetAnimBlend`. It does not do root motion, sync markers, or per-bone masks (use layers for upper-body overlays).
- Each weighted clip is sampled once per frame, so a tree costs about one clip evaluation per clip with non-zero weight.
- Trees belong to one entity. Use one tree per character.
