# Greenwood

A hunting game in moonBASIC, in the style of the old 3D adventures: lock on, draw the bow, and hunt through a generated forest island at morning.

Run it from this folder:

```
moon main.mb
```

## Controls

| Key | Action |
| --- | --- |
| WASD, mouse | Move, turn the camera |
| Shift | Run (animals hear you farther off) |
| C or Left Ctrl | Sneak (quiet, slow) |
| Space | Roll |
| Right mouse or Q, held | Lock on to the nearest animal in front of you |
| Left mouse or F, held | Draw the bow; let go to shoot. Full draw flies faster and hits twice as hard |
| E | Take what you hunted |
| Esc | Quit |

Locked on, arrows lead a running animal and allow for the drop. Without a lock, the camera goes over your shoulder and the arrow flies through the crosshair.

## The hunt

Four hares, three deer and two boar. Deer bolt when they hear you, and the herd goes with them. Boar don't run: they come for you, so roll. Arrows that miss stick in the ground and trees; walk over them to take them back. Sit by the campfire to heal. With the hunt done, the white stag comes out in a glade on the high ground.

## Files

- `main.mb`: the world, the hunter, the bow, the animals and the camera
- `chars.mb`: the hunter and the animals, sculpted from rings of vertices with vertex colour
- `hud.mb`: hearts, the tally, the quiver, the compass and the block font
- `tools/make_assets.py`: makes every sound in `assets/` (standard-library Python)

## Test modes

An empty flag file in this folder changes how it starts, for checking it without playing:

- `autoplay.flag`: walks to a deer, locks on and shoots
- `deaf.flag`: animals don't hear you; autoplay hunts a hare and takes it
- `endtest.flag`: every quota met at the start, so the white stag appears (pair with `deaf.flag`)
- `closeup1.flag` to `closeup4.flag`: a side view of a hare, doe, stag or boar; `closeup6.flag`: the hunter at full draw
