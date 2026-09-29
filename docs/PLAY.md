# Outdoor and indoor play

One world. The terrain streams the outdoors. Rooms stream the indoors. `examples/outdoor.bb` is the whole loop in one file. Run it with:

```powershell
.\bs.exe examples\outdoor.bb
```

| Key | What it does |
| --- | --- |
| WASD | Walk. The camera stays on `TerrainHeight`. |
| Mouse or arrows | Look. |
| E | Use whatever is in the center of the screen. |
| `[` `]` | Move the hour. The sun, sky, fog, and outdoor ambient follow. |
| F5 | `SaveGame("outdoor.json")` |
| F9 | `LoadGame("outdoor.json")` |
| Esc | Quit, after the window has drawn a few frames. |

Command tables: [COMMANDS.md](COMMANDS.md) (Terrain, and Outdoor and indoor play). Terrain mesh and splat: [TERRAIN.md](TERRAIN.md).

## 1. Ground, grass, trees, rocks

Build a heightfield, turn on the splat, then plant. The cone and the cube are templates. `HideEntity` keeps the template out of the scene. The copies are made when a terrain chunk loads, and freed when that chunk unloads.

```basic
hm = GenerateHeightmapPreset("rolling-hills", 96, 96, 7)
ground = CreateTerrainFromHeightmap(hm, 140, 140, 18)
ApplyTerrainSplat(ground)
SetTerrainGrassCoverage(ground, 0.7)
SetTerrainWaterHeight(ground, 2)
SetTerrainSnow(True, 16)
SetTerrainLOD(ground, 1)
SetTerrainDetail(ground, 8)

tree = CreateCone(0.55, 2.4, 8)
HideEntity(tree)
TerrainTrees(ground, tree, 0.7)

rock = CreateCube()
HideEntity(rock)
TerrainProps(ground, rock, 0.45)

TerrainFoliage(ground, 1)
```

What each density means:

| Call | Density `1` | Density `0` | Where it plants |
| --- | --- | --- | --- |
| `TerrainFoliage(t [, density])` | A light meadow of crossed cards | Clears the cards | Flat ground, above the shore, below the snow |
| `TerrainTrees(t, mesh [, density])` | Spaced copies of `mesh` | Clears that mesh | Same flat-ground test as grass |
| `TerrainProps(t, mesh [, density])` | A sparser layer of that mesh | Clears that mesh | Shore, dirt, and moderate slopes. Not cliffs, not snow |

Call `TerrainProps` again with a different mesh to add a second layer (bushes, then crates). Near trees and props are instances. On a far chunk they become one vertical quad turned toward the camera at the moment the chunk is built. Near grass cards are two crossed quads. Far grass is one quad.

These copies are not in the heightfield, not in collision, and not in `BakeTerrainNav`. A tree does not stop a character unless you give that character your own collision. `CreateWorldStream` is still how you stream a hand-built town, and it still fills new cells with the demo slabs and cones. `LoadChunk` still forces one cell.

`SetPlayer(cam)` is still the body that actors chase and that footsteps follow. It also turns the world bubble on: the terrain window, water (if you have not set follow yourself), near-chunk collision, and a shift past 4 km. `SetStreamOrigin` every frame, as `examples/outdoor.bb` does, still pins the window on the camera and overrides that follow. `SetWorldBubble(0)` before `SetPlayer` keeps this call as footsteps only. Details: [STREAM.md](STREAM.md).

Snap the camera to the ground yourself:

```basic
SetPlayer(cam)
SetStreamOrigin(EntityX(cam), 0, EntityZ(cam))
h# = TerrainHeight(EntityX(cam), EntityZ(cam))
SetPosition(cam, EntityX(cam), h + 1.75, EntityZ(cam))
```

## 2. Time of day

Lamp color, falloff, materials, cookies, and the outdoor / indoor presets are [LIGHTING.md](LIGHTING.md). This section is the clock that moves the sun and the sky together.

```basic
sun = CreateDirectionalLight()
SetTimeOfDay(15)
```

`hours` runs `0`–`24` and wraps. Every directional light gets a new pitch and yaw. The sky colors, fog, and outdoor ambient change together.

| Hours | Look |
| --- | --- |
| `0`–`5.5` and `19.5`–`24` | Night. Dim sun, dark sky, dark fog. |
| `5.5`–`8` and `17`–`19.5` | Sunset. |
| `8`–`17` | Day. |

The sun pitch follows the hour and stays at least 8°, so night is a low dim light. `GetTimeOfDay()` reads the hour back. Calling it again inside the same band does not rebuild the skybox.

In the example, hold `[` or `]` and the hour moves:

```basic
hour# = GetTimeOfDay()
If KeyDown(KEY_LBRACKET) Then hour = hour - 4 * dt
If KeyDown(KEY_RBRACKET) Then hour = hour + 4 * dt
SetTimeOfDay(hour)
```

## 3. A painted ground mix

`SetTerrainBlendMap(terrain, tex)` overrides the slope and height splat where the image has paint. Empty texels leave the splat alone.

| Channel | Layer |
| --- | --- |
| R | Sand |
| G | Grass |
| B | Rock |
| A | Snow |

```basic
paint = LoadTexture("assets/blend.png")
SetTerrainBlendMap(ground, paint)
```

`SetTerrainSplat` still supplies the textures those channels sample. `SetTerrainDetail(terrain, pixels)` is how large one triangle may be. `8` matches the old distance steps. A smaller number keeps the fine mesh farther out. `0` restores the old steps. This is a CPU grid. It does not turn on hardware tessellation.

The expert planter is the same three calls with the rule named:

```basic
TerrainScatter(ground, bush, "prop", 0.5)
TerrainScatter(ground, bush, "prop", 0)
```

`kind$` is `"grass"`, `"tree"`, `"prop"`, or `"any"`. The last line removes that mesh.

## 4. Rooms

A room is a visibility set, not a second renderer. Room `0` is outdoors. An entity with no room (the player, the terrain, the sun, the cabin walls in the example) stays visible everywhere. An entity you `SetRoom` is visible only in that room, or in a room linked to it.

```basic
hall = CreateRoom()
kitchen = CreateRoom()
RoomLink(hall, kitchen)
SetRoom(table, hall)
SetRoom(pot, kitchen)
EnterRoom(hall)
```

`RoomAdd(hall, table)` is the same assignment as `SetRoom(table, hall)`. `EnterRoom(0)` goes back outside and hides the furniture. `CurrentRoom()` is the id you are in.

From `hall` you also see `kitchen`, because they are linked. You do not see a third room until you link it or enter it.

`RoomAmbient(hall, 42, 36, 28)` replaces the sky ambient while you are in that room and turns the sun down, so a local light can own the space. `EnterRoom(0)` restores the outdoor ambient and the sun. Colors are 0–255, or 0–1 if every channel is at most 1.

```basic
lamp = CreatePointLight()
SetPosition(lamp, hx, hy + 2.2, hz)
SetRoom(lamp, hall)
```

## 5. Doors

`CreateDoor` uses an entity you already built. `Use` swings it 100° on yaw. The swing runs each Flip. Passing two room ids also links those rooms, so the next room is visible while the door is in front of you.

```basic
door = CreateCube()
SetScale(door, 0.08, 1.1, 0.55)
SetPosition(door, hx + 2.1, hy + 1.1, hz)
CreateDoor(door, hall, kitchen, doorSnd)
```

`doorSnd` is a `LoadSound` handle. Pass `0`, or leave it off, for a silent door. The door itself has no room, so the mesh stays visible outdoors and indoors.

Entering is a line you write. The engine does not teleport the player through the doorway:

```basic
CameraPick(cam, 640, 360)
hit = PickedEntity()
Use(hit)
If hit = door Then
    If CurrentRoom() = 0 Then
        EnterRoom(hall)
    Else
        EnterRoom(0)
    EndIf
EndIf
```

## 6. Use, items, and dialogue

`Use()` with no argument uses `PickedEntity()` from the last `CameraPick`, `Raycast`, or `GrabPick`. `Use(ent)` names the entity. It returns `1` when something happened.

| The entity has | `Use` does |
| --- | --- |
| `CreateDoor` | Toggles the swing and plays the door sound |
| `SetItem(ent, name$)` | Adds `name$` to the inventory and hides the entity |
| `SetDialogue(ent, text$)` | Starts the line list |
| `CreateActor` | Sets the actor to chase |
| none of those | Returns `0` |

```basic
key = CreateCube()
SetItem(key, "key")

npc = CreateCube()
SetRoom(npc, hall)
SetDialogue(npc, "The key is outside.|Bring it back.")
```

Lines split on `|` or a newline. While a conversation is open, E should advance it instead of picking again:

```basic
If KeyHit(KEY_E) Then
    If DialogueOn() Then
        DialogueAdvance()
    Else
        CameraPick(cam, 640, 360)
        Use(PickedEntity())
    EndIf
EndIf

If DialogueOn() Then Text(12, 52, DialogueLine$())
If InventoryHas("key") Then Text(12, 72, "You have the key")
```

`InventoryCount()` is how many names you hold. `InventoryItem(i)` is name `i` starting at `0`. `InventoryRemove("key")` drops one copy of that name from the list. It does not respawn the mesh.

## 7. Footsteps

`SetPlayer` is who the steps measure, and it also starts the world bubble described in [STREAM.md](STREAM.md). `SetWorldBubble(0)` keeps it as the footstep body only. `RoomAudio` is the clip for the room you are in, including room `0` if you want outdoor steps.

```basic
stepSnd = LoadSound("assets/step.wav")
RoomAudio(hall, stepSnd, 0.4)
```

The clip plays about once a metre. `reverb` is `0`–`1`. There is no convolution reverb. Above `0`, the same clip plays again, quieter, about 0.12 seconds later.

## 8. Actors

Three states. `ActorState(ent)` returns the number.

| State | Meaning |
| --- | --- |
| `0` | Idle. The player is farther than `aggro`. |
| `1` | Chase. Walks toward `SetPlayer` and samples `TerrainHeight`. |
| `2` | Attack. Within 1.5 units. The actor stops there. |

```basic
wolf = CreateCube()
SetPosition(wolf, 28, TerrainHeight(28, 24) + 0.4, 24)
CreateActor(wolf, 11, 4)
```

`11` is the aggro distance. `4` is the speed. Defaults are `8` and `3.2` if you omit them. `Use` on the wolf starts a chase even when you are still outside aggro. This is not a behavior tree and not inverse kinematics. Attack does not deal damage. You read `ActorState` and apply damage yourself.

## 9. Save and load

```basic
If KeyHit(KEY_F5) Then SaveGame("outdoor.json")
If KeyHit(KEY_F9) Then LoadGame("outdoor.json")
```

The file is JSON next to the program when the path is relative. It stores the hour, the player position, the inventory names, which doors are open, visited terrain-chunk keys, the current room, and the dialogue line. It does not store the whole scene. `SaveScene` is the full dump. `LoadGame` of a missing file returns `0` and an error.

Give every object a room only when it should hide. The player, the camera, the terrain, and the sun stay untagged so a load that calls `EnterRoom` does not hide the world.