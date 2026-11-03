# Animation

Skeletal clips come from `LoadAnimMesh` (`.gltf` / `.glb`). FBX translation curves come in with `LoadMesh`. The command list is in [COMMANDS.md](COMMANDS.md#mesh-animation-gltf). This page is how a character blends, walks, masks a layer, and fires a footstep.

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

## Clip graph

`CreateAnimGraph(entity)` returns an id. `AnimState graph, name$, clipIndex` records a clip on that entity. The first state becomes current. `AnimGo graph, name$` switches to that clip and starts it from the beginning at speed 1. `AnimUpdate$(graph)` is the current state name.

`AddAnimTransition graph, from$, to$, func$` is checked every frame. `from$` of `*` matches any state. When `Function func()` returns nonzero, the graph calls `AnimGo` into `to$`. The script can still call `AnimGo` itself.

```basic
g = CreateAnimGraph(hero)
AnimState g, "idle", 0
AnimState g, "run", 1
AddAnimTransition g, "idle", "run", "ShouldRun"

Function ShouldRun()
    Return speed > 1.5
End Function
```

`SolveTwoBone(ax, ay, tx, ty, len1, len2)` solves a 2D reach and returns the shoulder angle in degrees. `IKBend()` is the elbow angle from the last solve. Bone IK on a skeleton is `SetIK` / `SetIKChain` in [COMMANDS.md](COMMANDS.md#mesh-animation-gltf).

## Root motion and masks

A clip moves the skeleton inside the entity. Root motion takes that motion and applies it to the entity instead, so a walk cycle actually walks.

`SetRootMotion e, on [, bone$] [, useY]` watches one bone. The first frame plants that bone. Later frames move the entity by the change since last frame, then put the bone back on the plant. Without that plant, the same offset would be added again every frame. `useY` 0 (the default) ignores vertical motion, which keeps a walk on the floor. The bone name is a glTF or FBX node name. An empty name uses the first child of the entity.

```basic
SetRootMotion hero, 1, "Hips"
```

Layers normally move every bone. `SetBoneMask e, slot, bone$, weight#` scales one bone on that layer. Weight 0 leaves the bone out. Weight 1 is the layer's own weight. A bone you never name stays at 1. `SetBoneMask e, slot, "*"` clears the mask.

```basic
SetAnimLayer hero, 0, 1, "wave"
SetBoneMask hero, 0, "Hips", 0
SetBoneMask hero, 0, "Spine", 0.2
```

The wave then plays on the arms and only a little on the spine. The hips stay on the base clip.

## Bones

`GetBoneX`, `GetBoneY`, and `GetBoneZ` read a bone's local position in the same axes as `PositionEntity`. `SetBonePos e, bone$, x, y, z` writes that local position. `BoneLookAt e, bone$, x, y, z` aims the bone at a world point, with world up as +Y.

`AttachToBone` parents another entity to a bone. `SetIK` and `SetIKChain` still solve reach. `BoneLookAt` does not replace them. It only turns the bone node.

## Events, recording, retarget

`AnimEvent e, time, func$` calls `Function func(time)` when the clip clock crosses `time`, including across a loop. `ClearAnimEvents e` drops them. A 2D sheet uses `SpriteAnimEvent sprite, frame, func$` instead, and fires when `AnimateSprite` lands on that frame.

`AnimRecord e, 1` starts sampling every named bone under the entity. `AnimRecord e, 0` stops and stores the samples as a clip. Up to 240 samples per bone. That clip plays like an FBX clip and `ExportAnimClip e, path$` can write it.

`AnimRetarget e, "Hips=pelvis,Spine=spine"` renames bones on curves already in memory. It does not rebuild a skeleton. The new names have to exist on the mesh you play the clip on.

## Files

`LoadAnimMesh` reads `.gltf` and `.glb` clips. `LoadMesh` on an ASCII FBX also reads translation curves, and a binary FBX reads them when the file contains `KeyTime` and `KeyValueFloat` under an `AnimationCurve`. Binary curves are paired with limb nodes in the order they appear, X then Y then Z. They are not matched through FBX connection nodes.

`ExportAnimClip e, file$` writes glTF 2.0 JSON. Each bone is a node. Each bone's translation keys sit in one embedded base64 buffer. The file is JSON, not a `.glb`.

## Morphs

`SetMorph e, index, weight` stores a blend-shape weight from 0 to 1. `GetMorph` reads it back. `MorphCount` is how many weights are stored, or how many the mesh reports when that number is larger. If the mesh really is morph geometry, the weight is copied onto it. A plain mesh keeps the number and does not change shape.

`SetViseme e, shape$, weight` is five named slots on that same list: `aa` 0, `ee` 1, `oh` 2, `mm` 3, `fv` 4. There is no face rig and no audio-to-viseme pass. You pick the shape.

## Limits

- Blending is a local-transform mix per node, the same quality as `SetAnimBlend`.
- Each weighted clip is sampled once per frame, so a tree costs about one clip evaluation per clip with a non-zero weight.
- Trees belong to one entity.
- Root motion uses the bone's local translation delta. It does not read a separate root-motion track from glTF.
- Binary FBX curves are translation only, and the bone match is file order, not the FBX connection graph.
- `ExportAnimClip` does not write rotation or scale channels.
