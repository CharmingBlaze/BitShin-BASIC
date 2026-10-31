# Example assets

Original checker textures and a short coin beep made for moonBASIC. CC0 — not from any commercial game. Regenerated with `go run tools/gen_platform64_assets.go`.

| File | Role |
| --- | --- |
| `grass.png` | 32×32 green checker for the ground |
| `heightmap.png` | Written by `examples/heightmap.mb` (`SaveHeightmap`) |
| `stone.png` | 32×32 tan checker for platforms |
| `coin.png` | 32×32 gold disc |
| `coin.wav` | ~125 ms 988 Hz beep |
| `walker.gltf` | Box figure with idle, walk, run, strafe_left, strafe_right, wave clips (`go run tools/gen_anim_assets.go`) |
| `midtown_walk.geojson` | Short WGS84 walk + point for `examples/geo.mb` |
| `terrain/*.png` | CC0 splat maps (`go run tools/gen_terrain_assets.go`) — not Nintendo, not Terrain-OpenGL photos |
