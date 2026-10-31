//go:build ignore

// Generates examples/assets/tiled: an original CC0 16 px tileset and a
// Tiled .tmj level for examples/tilemap2d.mb. The level uses layers,
// parallax, a flipped tile, an animated water tile, autotiled paths, tile
// properties, and an object layer. Run from the repo root:
//
//	go run tools/gen_tiled_assets.go
package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
)

const (
	ts   = 16
	cols = 8
	rows = 4
	mapW = 60
	mapH = 34
)

// Tile ids (1-based, as in Tiled with firstgid 1).
const (
	tGrass  = 1
	tGrass2 = 2
	tStone  = 3
	tWater  = 4 // and 5: two animation frames
	tFlower = 6
	tCrate  = 7
	tHill   = 8
	tPath   = 9 // 9..24: 16 autotile variants, north 1, east 2, south 4, west 8
)

func main() {
	dir := filepath.Join("examples", "assets", "tiled")
	must(os.MkdirAll(dir, 0o755))
	img := image.NewRGBA(image.Rect(0, 0, cols*ts, rows*ts))
	rng := rand.New(rand.NewSource(7))

	paint := func(id int, fn func(x, y int) color.RGBA) {
		ox, oy := ((id-1)%cols)*ts, ((id-1)/cols)*ts
		for y := 0; y < ts; y++ {
			for x := 0; x < ts; x++ {
				img.SetRGBA(ox+x, oy+y, fn(x, y))
			}
		}
	}
	noise := func(c color.RGBA, amt int) color.RGBA {
		d := rng.Intn(2*amt+1) - amt
		return color.RGBA{clamp(int(c.R) + d), clamp(int(c.G) + d), clamp(int(c.B) + d), 255}
	}
	grass := color.RGBA{86, 150, 72, 255}
	paint(tGrass, func(x, y int) color.RGBA { return noise(grass, 10) })
	paint(tGrass2, func(x, y int) color.RGBA {
		if (x*7+y*3)%11 == 0 {
			return color.RGBA{120, 186, 92, 255}
		}
		return noise(grass, 10)
	})
	paint(tStone, func(x, y int) color.RGBA {
		if x == 0 || y == 0 || (y == 8 && x < 15) || (x == 8 && y < 8) {
			return color.RGBA{70, 72, 80, 255}
		}
		return noise(color.RGBA{128, 130, 140, 255}, 8)
	})
	for f := 0; f < 2; f++ {
		f := f
		paint(tWater+f, func(x, y int) color.RGBA {
			wave := math.Sin(float64(x)*0.8+float64(f)*math.Pi+float64(y)*0.3) > 0.82
			if wave {
				return color.RGBA{170, 214, 240, 255}
			}
			return noise(color.RGBA{52, 110, 186, 255}, 5)
		})
	}
	paint(tFlower, func(x, y int) color.RGBA {
		dx, dy := x-8, y-7
		switch {
		case dx*dx+dy*dy <= 3:
			return color.RGBA{250, 220, 70, 255}
		case dx*dx+dy*dy <= 12:
			return color.RGBA{230, 90, 140, 255}
		case x == 8 && y > 9:
			return color.RGBA{40, 110, 50, 255}
		}
		return noise(grass, 10)
	})
	paint(tCrate, func(x, y int) color.RGBA {
		if x < 2 || y < 2 || x > 13 || y > 13 || x == y || x == 15-y {
			return color.RGBA{110, 70, 36, 255}
		}
		if x > 14 || y > 14 {
			return noise(grass, 10)
		}
		return noise(color.RGBA{176, 124, 70, 255}, 6)
	})
	paint(tHill, func(x, y int) color.RGBA {
		return color.RGBA{86, 112, 156, 255}
	})
	// Paths: a sand band through the centre, extended toward each neighbour.
	for mask := 0; mask < 16; mask++ {
		mask := mask
		paint(tPath+mask, func(x, y int) color.RGBA {
			in := x >= 4 && x < 12 && y >= 4 && y < 12
			if mask&1 != 0 && x >= 4 && x < 12 && y < 4 {
				in = true
			}
			if mask&2 != 0 && y >= 4 && y < 12 && x >= 12 {
				in = true
			}
			if mask&4 != 0 && x >= 4 && x < 12 && y >= 12 {
				in = true
			}
			if mask&8 != 0 && y >= 4 && y < 12 && x < 4 {
				in = true
			}
			if in {
				return noise(color.RGBA{212, 186, 128, 255}, 8)
			}
			return noise(grass, 10)
		})
	}
	f, err := os.Create(filepath.Join(dir, "tiles.png"))
	must(err)
	must(png.Encode(f, img))
	f.Close()

	// The level.
	ground := make([]uint32, mapW*mapH)
	deco := make([]uint32, mapW*mapH)
	path := make([]uint32, mapW*mapH)
	hills := make([]uint32, mapW*mapH)
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			i := y*mapW + x
			ground[i] = tGrass
			if rng.Intn(5) == 0 {
				ground[i] = tGrass2
			}
			if x == 0 || y == 0 || x == mapW-1 || y == mapH-1 {
				ground[i] = tStone
			}
			// A pond.
			if dx, dy := float64(x-42), float64(y-22)*1.4; dx*dx+dy*dy < 40 {
				ground[i] = tWater
			}
			// Open sky along the top, with far hills on a parallax layer.
			if y < 5 {
				ground[i] = 0
			}
			if y >= 2 && y < 5+int(2*math.Sin(float64(x)*0.35)) {
				hills[i] = tHill
			}
		}
	}
	inPath := map[[2]int]bool{}
	prevY := -1
	for x := 4; x < 54; x++ { // a winding road, one tile wide
		y := 17 + int(math.Round(5*math.Sin(float64(x)*0.18)))
		if prevY >= 0 { // step vertically so the road stays connected
			for yy := min(prevY, y); yy <= max(prevY, y); yy++ {
				inPath[[2]int{x, yy}] = true
			}
		}
		inPath[[2]int{x, y}] = true
		prevY = y
	}
	for y := 5; y < 17; y++ {
		inPath[[2]int{20, y}] = true
	}
	for p := range inPath {
		mask := 0
		for b, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
			if inPath[[2]int{p[0] + d[0], p[1] + d[1]}] {
				mask |= 1 << b
			}
		}
		path[p[1]*mapW+p[0]] = uint32(tPath + mask)
	}
	for k := 0; k < 90; k++ {
		x, y := 1+rng.Intn(mapW-2), 7+rng.Intn(mapH-8)
		i := y*mapW + x
		if ground[i] != tGrass && ground[i] != tGrass2 || path[i] != 0 {
			continue
		}
		deco[i] = tFlower
		if rng.Intn(4) == 0 {
			deco[i] = tCrate
		}
		if rng.Intn(2) == 0 {
			deco[i] |= 1 << 31 // mirrored
		}
	}

	layer := func(name string, data []uint32, extra map[string]any) map[string]any {
		l := map[string]any{"type": "tilelayer", "name": name, "width": mapW, "height": mapH,
			"x": 0, "y": 0, "opacity": 1, "visible": true, "data": data}
		for k, v := range extra {
			l[k] = v
		}
		return l
	}
	level := map[string]any{
		"type": "map", "version": "1.10", "orientation": "orthogonal", "renderorder": "right-down",
		"width": mapW, "height": mapH, "tilewidth": ts, "tileheight": ts, "infinite": false,
		"tilesets": []any{map[string]any{
			"firstgid": 1, "name": "tiles", "image": "tiles.png", "imagewidth": cols * ts, "imageheight": rows * ts,
			"tilewidth": ts, "tileheight": ts, "columns": cols, "tilecount": cols * rows, "margin": 0, "spacing": 0,
			"tiles": []any{
				map[string]any{"id": tWater - 1, "animation": []any{
					map[string]any{"tileid": tWater - 1, "duration": 500},
					map[string]any{"tileid": tWater, "duration": 500},
				}, "properties": []any{map[string]any{"name": "water", "type": "bool", "value": true}}},
				map[string]any{"id": tStone - 1, "properties": []any{map[string]any{"name": "solid", "type": "bool", "value": true}}},
				map[string]any{"id": tCrate - 1, "properties": []any{map[string]any{"name": "solid", "type": "bool", "value": true}}},
			},
		}},
		"layers": []any{
			layer("Hills", hills, map[string]any{"parallaxx": 0.5, "parallaxy": 0.5, "opacity": 0.8}),
			layer("Ground", ground, nil),
			layer("Path", path, nil),
			layer("Deco", deco, nil),
			map[string]any{"type": "objectgroup", "name": "Objects", "visible": true, "opacity": 1, "x": 0, "y": 0,
				"objects": []any{
					map[string]any{"id": 1, "name": "player", "type": "spawn", "x": 6 * ts, "y": 18 * ts, "width": 0, "height": 0, "rotation": 0, "visible": true},
					map[string]any{"id": 2, "name": "sign", "type": "note", "x": 20 * ts, "y": 6 * ts, "width": ts, "height": ts, "rotation": 0, "visible": true,
						"properties": []any{map[string]any{"name": "text", "type": "string", "value": "North road"}}},
				}},
		},
	}
	b, err := json.MarshalIndent(level, "", " ")
	must(err)
	must(os.WriteFile(filepath.Join(dir, "level.tmj"), b, 0o644))
}

func clamp(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
