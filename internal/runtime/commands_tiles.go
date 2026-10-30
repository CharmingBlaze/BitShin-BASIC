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
	cells              []int
	atlas              int
	atlasCols          int
	open               map[int]bool // tile ids that do not block, even when id > 0
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
			w.tiles[id] = &tileMap{tw: tw, th: th, cols: cols, rows: rows, cells: make([]int, cols*rows), atlasCols: 8}
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
			tm.cells[y*tm.cols+x] = id
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
			ox, oy := argN(a, 1, 0), argN(a, 2, 0)
			im := w.images[tm.atlas]
			for y := 0; y < tm.rows; y++ {
				for x := 0; x < tm.cols; x++ {
					id := tm.cells[y*tm.cols+x]
					if id <= 0 {
						continue
					}
					px := float32(ox + float64(x*tm.tw) - w.cam2X)
					py := float32(oy + float64(y*tm.th) - w.cam2Y)
					if im != nil {
						acols := tm.atlasCols
						if acols < 1 {
							acols = im.w / tm.tw
							if acols < 1 {
								acols = 1
							}
						}
						tile := id - 1
						sx, sy := (tile%acols)*tm.tw, (tile/acols)*tm.th
						w.draws = append(w.draws, drawOp{
							kind: 0, img: im.img, x: px, y: py,
							w: float32(tm.tw), h: float32(tm.th),
							sx: sx, sy: sy, sw: tm.tw, sh: tm.th,
						})
					} else {
						w.draws = append(w.draws, drawOp{
							kind: 1, x: px, y: py, w: float32(tm.tw - 1), h: float32(tm.th - 1),
							r: uint8(40 + (id*40)%200), g: uint8(80 + (id*20)%160), b: uint8(120),
							filled: true,
						})
					}
				}
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
			tm := &tileMap{tw: tw, th: th, cols: cols, rows: len(rows), cells: make([]int, cols*len(rows)), atlasCols: 8}
			for y, row := range rows {
				for x, cell := range row {
					n, _ := strconv.Atoi(strings.TrimSpace(cell))
					tm.cells[y*cols+x] = n
				}
			}
			w.tiles[id] = tm
			return value.Num(float64(id)), nil
		}),
		"gettile": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(tileCell(w.tiles[argI(a, 0, 0)], argI(a, 1, 0), argI(a, 2, 0)))), nil
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
	return tm.cells[y*tm.cols+x]
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
