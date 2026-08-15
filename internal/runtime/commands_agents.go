package runtime

import (
	"fmt"

	"github.com/quasilyte/pathing"

	"bitshinbasic/internal/value"
)

type gridMap struct {
	w, h  int
	g     *pathing.Grid
	astar *pathing.AStar
	layer pathing.GridLayer
}

type gridPath struct {
	xs, ys []int
}

func (w *World) agentCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"creategrid": n(func(a []value.Value) (value.Value, error) {
			gw, gh := argI(a, 0, 8), argI(a, 1, 8)
			if gw < 1 {
				gw = 1
			}
			if gh < 1 {
				gh = 1
			}
			g := pathing.NewGrid(pathing.GridConfig{
				WorldWidth:  uint(gw),
				WorldHeight: uint(gh),
				CellWidth:   1,
				CellHeight:  1,
			})
			id := w.takeHandle(&w.freeGrids, &w.nextGrid)
			if w.grids == nil {
				w.grids = map[int]*gridMap{}
			}
			w.grids[id] = &gridMap{
				w:     gw,
				h:     gh,
				g:     g,
				astar: pathing.NewAStar(pathing.AStarConfig{NumCols: uint(gw), NumRows: uint(gh)}),
				layer: pathing.MakeGridLayer([8]uint8{0: 1, 1: 0}),
			}
			return value.Num(float64(id)), nil
		}),
		"setgridwalkable": n(func(a []value.Value) (value.Value, error) {
			gm, err := w.gridOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			x, y := argI(a, 1, 0), argI(a, 2, 0)
			on := argI(a, 3, 1) != 0
			if x < 0 || y < 0 || x >= gm.w || y >= gm.h {
				return z()
			}
			tag := uint8(0)
			if !on {
				tag = 1
			}
			gm.g.SetCellTile(pathing.GridCoord{X: x, Y: y}, tag)
			gm.g.SetCellIsBlocked(pathing.GridCoord{X: x, Y: y}, !on)
			return z()
		}),
		"findpath": n(func(a []value.Value) (value.Value, error) {
			gm, err := w.gridOf(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			x1, y1 := argI(a, 1, 0), argI(a, 2, 0)
			x2, y2 := argI(a, 3, 0), argI(a, 4, 0)
			from := pathing.GridCoord{X: x1, Y: y1}
			to := pathing.GridCoord{X: x2, Y: y2}
			res := gm.astar.BuildPath(gm.g, from, to, gm.layer)
			steps := res.Steps
			xs := []int{x1}
			ys := []int{y1}
			cx, cy := x1, y1
			for steps.HasNext() {
				switch steps.Next() {
				case pathing.DirRight:
					cx++
				case pathing.DirLeft:
					cx--
				case pathing.DirDown:
					cy++
				case pathing.DirUp:
					cy--
				}
				xs = append(xs, cx)
				ys = append(ys, cy)
			}
			id := w.takeHandle(&w.freePaths, &w.nextPath)
			if w.paths == nil {
				w.paths = map[int]*gridPath{}
			}
			w.paths[id] = &gridPath{xs: xs, ys: ys}
			return value.Num(float64(id)), nil
		}),
		"pathlength": n(func(a []value.Value) (value.Value, error) {
			p := w.paths[argI(a, 0, 0)]
			if p == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(len(p.xs))), nil
		}),
		"pathx": n(func(a []value.Value) (value.Value, error) {
			p := w.paths[argI(a, 0, 0)]
			i := argI(a, 1, 0)
			if p == nil || i < 0 || i >= len(p.xs) {
				return value.Num(0), nil
			}
			return value.Num(float64(p.xs[i])), nil
		}),
		"pathy": n(func(a []value.Value) (value.Value, error) {
			p := w.paths[argI(a, 0, 0)]
			i := argI(a, 1, 0)
			if p == nil || i < 0 || i >= len(p.ys) {
				return value.Num(0), nil
			}
			return value.Num(float64(p.ys[i])), nil
		}),
	}
}

func (w *World) gridOf(id int) (*gridMap, error) {
	g := w.grids[id]
	if g == nil {
		return nil, fmt.Errorf("invalid grid %d", id)
	}
	return g, nil
}
