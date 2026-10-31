package runtime

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"moonbasic/internal/value"
)

type tileMap struct {
	tw, th, cols, rows int
	cells              []int // layer 0
	layers             []*tileLayer
	atlas              int
	atlasCols          int
	open               map[int]bool // tile ids that do not block, even when id > 0
	sets               []*tileSet   // Tiled tilesets; empty uses atlas
	anims              map[int]tileAnim
	data               map[int]map[string]value.Value
	objects            []tileObject
	collide            int // layer that blocks sprites
}

type fontSlot struct {
	src  *text.GoTextFaceSource
	size float64
}

func (w *World) tileCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	need2 := func(fn func([]value.Value) (value.Value, error)) cmd {
		return func(a []value.Value) (value.Value, error) {
			if !w.ready {
				return value.Value{}, w.require()
			}
			return fn(a)
		}
	}
	return map[string]cmd{
		"drawimagerect": need2(func(a []value.Value) (value.Value, error) {
			im := w.images[argI(a, 0, 0)]
			if im == nil {
				return z()
			}
			sx, sy, sw, sh := argI(a, 3, 0), argI(a, 4, 0), argI(a, 5, im.w), argI(a, 6, im.h)
			dw, dh := argN(a, 7, float64(sw)), argN(a, 8, float64(sh))
			w.draws = append(w.draws, drawOp{
				kind: 0, img: im.img,
				x: float32(argN(a, 1, 0)), y: float32(argN(a, 2, 0)),
				w: float32(dw), h: float32(dh),
				sx: sx, sy: sy, sw: sw, sh: sh,
			})
			return z()
		}),
		"createtilemap": n(func(a []value.Value) (value.Value, error) {
			tw, th := argI(a, 0, 16), argI(a, 1, 16)
			cols, rows := argI(a, 2, 8), argI(a, 3, 8)
			if tw < 1 {
				tw = 16
			}
			if th < 1 {
				th = 16
			}
			if cols < 1 {
				cols = 1
			}
			if rows < 1 {
				rows = 1
			}
			id := w.nextTile
			w.nextTile++
			tm := &tileMap{tw: tw, th: th, cols: cols, rows: rows, atlasCols: 8}
			tm.addLayer("")
			w.tiles[id] = tm
			return value.Num(float64(id)), nil
		}),
		"settile": n(func(a []value.Value) (value.Value, error) {
			tm := w.tiles[argI(a, 0, 0)]
			if tm == nil {
				return value.Value{}, fmt.Errorf("SetTile: invalid tilemap")
			}
			x, y, id := argI(a, 1, 0), argI(a, 2, 0), argI(a, 3, 0)
			if x < 0 || y < 0 || x >= tm.cols || y >= tm.rows {
				return value.Value{}, fmt.Errorf("SetTile: %d,%d out of map", x, y)
			}
			l, err := tm.layerByArg(a, 4)
			if err != nil {
				return value.Value{}, fmt.Errorf("SetTile: %w", err)
			}
			l.cells[y*tm.cols+x] = id
			return z()
		}),
		"drawtile": need2(func(a []value.Value) (value.Value, error) {
			im := w.images[argI(a, 0, 0)]
			if im == nil {
				return z()
			}
			tile, tw, th := argI(a, 1, 0), argI(a, 4, 16), argI(a, 5, 16)
			if tw < 1 {
				tw = 16
			}
			if th < 1 {
				th = 16
			}
			acols := im.w / tw
			if acols < 1 {
				acols = 1
			}
			sx, sy := (tile%acols)*tw, (tile/acols)*th
			w.draws = append(w.draws, drawOp{
				kind: 0, img: im.img,
				x: float32(argN(a, 2, 0)), y: float32(argN(a, 3, 0)),
				w: float32(tw), h: float32(th),
				sx: sx, sy: sy, sw: tw, sh: th,
			})
			return z()
		}),
		"drawtilemap": need2(func(a []value.Value) (value.Value, error) {
			tm := w.tiles[argI(a, 0, 0)]
			if tm == nil {
				return value.Value{}, fmt.Errorf("DrawTileMap: invalid tilemap")
			}
			atlasID := argI(a, 3, tm.atlas)
			if atlasID != 0 {
				tm.atlas = atlasID
			}
			// One queued draw per layer: each camera draws only the cells it
			// can see when the frame is drawn. DrawTileMap map, x, y [, atlas,
			// layer] draws one layer; without a layer, all of them in order.
			ox, oy := argN(a, 1, 0), argN(a, 2, 0)
			if len(a) > 4 {
				if tm.layer(argI(a, 4, 0)) == nil {
					return value.Value{}, fmt.Errorf("DrawTileMap: layer %d of %d", argI(a, 4, 0), len(tm.layers))
				}
				w.draws = append(w.draws, drawOp{kind: 5, tm: tm, layer: argI(a, 4, 0), x: float32(ox), y: float32(oy), world: true})
				return z()
			}
			for i := range tm.layers {
				w.draws = append(w.draws, drawOp{kind: 5, tm: tm, layer: i, x: float32(ox), y: float32(oy), world: true})
			}
			return z()
		}),
		"loadtilemap": n(func(a []value.Value) (value.Value, error) {
			path, err := w.openPath(argS(a, 0))
			if err != nil {
				return value.Value{}, fmt.Errorf("LoadTileMap: %w", err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return value.Value{}, err
			}
			r := csv.NewReader(strings.NewReader(string(raw)))
			r.FieldsPerRecord = -1
			rows, err := r.ReadAll()
			if err != nil {
				return value.Value{}, fmt.Errorf("LoadTileMap: %w", err)
			}
			if len(rows) == 0 {
				return value.Value{}, fmt.Errorf("LoadTileMap: empty csv")
			}
			cols := 0
			for _, row := range rows {
				if len(row) > cols {
					cols = len(row)
				}
			}
			tw, th := argI(a, 1, 16), argI(a, 2, 16)
			id := w.nextTile
			w.nextTile++
			tm := &tileMap{tw: tw, th: th, cols: cols, rows: len(rows), atlasCols: 8}
			tm.addLayer("")
			for y, row := range rows {
				for x, cell := range row {
					n, _ := strconv.Atoi(strings.TrimSpace(cell))
					tm.cells[y*cols+x] = n
				}
			}
			w.tiles[id] = tm
			return value.Num(float64(id)), nil
		}),
		// GetTile(map, x, y [, layer]); the default is the collision layer.
		"gettile": n(func(a []value.Value) (value.Value, error) {
			tm := w.tiles[argI(a, 0, 0)]
			if tm != nil && len(a) > 3 {
				l := tm.layer(argI(a, 3, 0))
				x, y := argI(a, 1, 0), argI(a, 2, 0)
				if l == nil || x < 0 || y < 0 || x >= tm.cols || y >= tm.rows {
					return value.Num(0), nil
				}
				return value.Num(float64(tileID(l.cells[y*tm.cols+x]))), nil
			}
			return value.Num(float64(tileCell(tm, argI(a, 1, 0), argI(a, 2, 0)))), nil
		}),
		"tileat": n(func(a []value.Value) (value.Value, error) {
			tm := w.tiles[argI(a, 0, 0)]
			if tm == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(tileCell(tm, floorCell(argN(a, 1, 0), tm.tw), floorCell(argN(a, 2, 0), tm.th)))), nil
		}),
		"settilesolid": n(func(a []value.Value) (value.Value, error) {
			tm := w.tiles[argI(a, 0, 0)]
			if tm == nil {
				return z()
			}
			id := argI(a, 1, 0)
			if argI(a, 2, 1) != 0 {
				if tm.open != nil {
					delete(tm.open, id)
				}
				return z()
			}
			if tm.open == nil {
				tm.open = map[int]bool{}
			}
			tm.open[id] = true
			return z()
		}),
		"spritehittile": need2(func(a []value.Value) (value.Value, error) {
			if w.spriteHitsTile(w.sprites[argI(a, 0, 0)], w.tiles[argI(a, 1, 0)]) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"loadfont": n(func(a []value.Value) (value.Value, error) {
			path, err := w.openPath(argS(a, 0))
			if err != nil {
				return value.Value{}, fmt.Errorf("LoadFont: %w", err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return value.Value{}, fmt.Errorf("LoadFont: %w", err)
			}
			src, err := text.NewGoTextFaceSource(bytes.NewReader(raw))
			if err != nil {
				return value.Value{}, fmt.Errorf("LoadFont: %w", err)
			}
			id := w.nextFont
			w.nextFont++
			sz := argN(a, 1, 16)
			if sz < 4 {
				sz = 16
			}
			w.fonts[id] = &fontSlot{src: src, size: sz}
			w.curFont = id
			return value.Num(float64(id)), nil
		}),
		"setfont": n(func(a []value.Value) (value.Value, error) {
			w.curFont = argI(a, 0, 0)
			return z()
		}),
	}
}

func tileCell(tm *tileMap, x, y int) int {
	if tm == nil || x < 0 || y < 0 || x >= tm.cols || y >= tm.rows {
		return 0
	}
	l := tm.layer(tm.collide)
	if l == nil {
		return 0
	}
	return tileID(l.cells[y*tm.cols+x])
}

func tileBlocks(tm *tileMap, id int) bool {
	if tm == nil || id <= 0 {
		return false
	}
	return tm.open == nil || !tm.open[id]
}

func floorCell(p float64, size int) int {
	if size < 1 {
		return 0
	}
	v := p / float64(size)
	i := int(v)
	if v < float64(i) {
		i--
	}
	return i
}

func (w *World) spriteHitsTile(s *ebiSprite, tm *tileMap) bool {
	if s == nil || tm == nil || tm.tw < 1 || tm.th < 1 {
		return false
	}
	ww, hh := spriteSize(s)
	x0, y0 := s.x-ww/2, s.y-hh/2
	tx0, ty0 := floorCell(x0, tm.tw), floorCell(y0, tm.th)
	tx1, ty1 := floorCell(x0+ww-0.001, tm.tw), floorCell(y0+hh-0.001, tm.th)
	for ty := ty0; ty <= ty1; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			if tileBlocks(tm, tileCell(tm, tx, ty)) {
				return true
			}
		}
	}
	return false
}
