# The Black Veil

A 3D action-adventure slice in moonBASIC, in the style of the old PlayStation adventures: a night in the village of Black Hollow, with a sword, a gun of finite rounds, and a mill with something under it.

Run it from this folder:

```
moon main.mb
```

## Controls

| Key | Action |
| --- | --- |
| WASD, mouse | Move, turn the camera |
| Left mouse or J | Ashen Blade (sword) |
| Right mouse or Q, held | Target the nearest enemy |
| F | Mourning Star (gun, limited rounds) |
| Space | Roll |
| E | Talk, open |
| Esc | Quit |

## Files

- `main.mb`: the village, the hero, combat, dialogue and the camera
- `chars.mb`: the hero and the creatures
- `look.mb`: lighting, fog and the low-resolution look
- `tools/make_assets.py`: makes every texture and sound in `assets/` (standard-library Python)
