package runtime

import (
	"math"

	"bitshinbasic/internal/mathx"
	"bitshinbasic/internal/value"
)

type crowd struct {
	agents []int
	sep    float32
}

func (w *World) tickCrowds() {
	dt := float32(w.delta)
	if dt <= 0 {
		dt = 1.0 / 60
	}
	for _, c := range w.crowds {
		if c == nil {
			continue
		}
		sep := c.sep
		if sep <= 0 {
			sep = 1.1
		}
		type pos struct{ x, y, z float32 }
		ps := make([]pos, len(c.agents))
		xs := make([]float64, len(c.agents))
		zs := make([]float64, len(c.agents))
		vxs := make([]float64, len(c.agents))
		vzs := make([]float64, len(c.agents))
		for i, id := range c.agents {
			if e := w.ents[id]; e != nil && e.node != nil {
				p := worldPos(e.node.GetNode())
				x, y, z := fromG3N(p.X, p.Y, p.Z)
				ps[i] = pos{x, y, z}
				xs[i], zs[i] = float64(x), float64(z)
				if ag := w.agents[id]; ag != nil && ag.hasLast {
					vxs[i] = float64(x-ag.lastX) / float64(dt)
					vzs[i] = float64(z-ag.lastZ) / float64(dt)
				}
			}
		}
		for i, id := range c.agents {
			e := w.ents[id]
			if e == nil || e.node == nil {
				continue
			}
			px, py, pz := ps[i].x, ps[i].y, ps[i].z
			rad := sep
			if ag := w.agents[id]; ag != nil && ag.radius > 0 {
				rad = ag.radius * 2.4
			}
			ax64, az64 := mathx.Separate2D(xs, zs, i, float64(rad))
			bx, bz := mathx.Avoid2D(xs, zs, vxs, vzs, i, float64(rad)*0.5, 1.2)
			ax, az := float32(ax64+bx), float32(az64+bz)
			if ax == 0 && az == 0 {
				if fx, fz := float32(vxs[i]), float32(vzs[i]); fx*fx+fz*fz > 0.0001 {
					e.node.GetNode().SetRotationY(float32(-math.Atan2(float64(fx), float64(fz))))
				}
				if ag := w.agents[id]; ag != nil {
					ag.lastX, ag.lastZ, ag.hasLast = px, pz, true
				}
				continue
			}
			px += ax * dt * 2
			pz += az * dt * 2
			if w.curTerrain != 0 {
				py = w.terrainHeight(px, pz) + 0.5
			}
			gx, gy, gz := toG3N(px, py, pz)
			e.node.GetNode().SetPosition(gx, gy, gz)
			fx, fz := ax, az
			if fx == 0 && fz == 0 {
				fx, fz = float32(vxs[i]), float32(vzs[i])
			}
			if fx*fx+fz*fz > 0.0001 {
				e.node.GetNode().SetRotationY(float32(-math.Atan2(float64(fx), float64(fz))))
			}
			if ag := w.agents[id]; ag != nil {
				ag.lastX, ag.lastZ, ag.hasLast = px, pz, true
			}
		}
	}
}

func (w *World) crowdCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createcrowd": n(func(a []value.Value) (value.Value, error) {
			if w.crowds == nil {
				w.crowds = map[int]*crowd{}
			}
			id := w.takeHandle(&w.freeCrowds, &w.nextCrowd)
			w.crowds[id] = &crowd{sep: float32(argN(a, 0, 1.2))}
			w.curCrowd = id
			return value.Num(float64(id)), nil
		}),
		"crowdaddagent": n(func(a []value.Value) (value.Value, error) {
			c := w.crowds[argI(a, 0, w.curCrowd)]
			ent := argI(a, 1, argI(a, 0, 0))
			if len(a) < 2 {
				c = w.crowds[w.curCrowd]
				ent = argI(a, 0, 0)
			}
			if c == nil {
				return z()
			}
			if _, err := w.ent(ent); err != nil {
				return value.Value{}, err
			}
			if w.agents[ent] == nil {
				if w.agents == nil {
					w.agents = map[int]*navAgent{}
				}
				w.agents[ent] = &navAgent{ent: ent, nav: w.curNav, speed: 4, radius: 0.45}
			}
			c.agents = append(c.agents, ent)
			return value.Num(float64(ent)), nil
		}),
		"crowdsetdestination": n(func(a []value.Value) (value.Value, error) {
			c := w.crowds[argI(a, 0, w.curCrowd)]
			off := 1
			if c == nil {
				c = w.crowds[w.curCrowd]
				off = 0
			}
			if c == nil {
				return z()
			}
			x, y, zz := float32(argN(a, off, 0)), float32(argN(a, off+1, 0)), float32(argN(a, off+2, 0))
			for _, id := range c.agents {
				if ag := w.agents[id]; ag != nil {
					pts, err := w.navFindPath(ag, x, y, zz)
					if err != nil || len(pts) == 0 {
						pts = [][3]float32{{x, y, zz}}
					}
					ag.path = pts
					ag.i = 0
					ag.moving = true
				}
			}
			return z()
		}),
		"crowdupdate": n(func(a []value.Value) (value.Value, error) {
			w.updateNav()
			w.tickCrowds()
			return z()
		}),
		"setcrowdradius": n(func(a []value.Value) (value.Value, error) {
			if c := w.crowds[argI(a, 0, w.curCrowd)]; c != nil {
				c.sep = float32(argN(a, 1, 1.2))
			}
			return z()
		}),
		"getcrowdradius": n(func(a []value.Value) (value.Value, error) {
			c := w.crowds[argI(a, 0, w.curCrowd)]
			if c == nil {
				c = w.crowds[w.curCrowd]
			}
			if c == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(c.sep)), nil
		}),
		"setnavmaxslope": n(func(a []value.Value) (value.Value, error) {
			w.navMaxSlope = float32(argN(a, 0, 45))
			return value.Num(float64(w.navMaxSlope)), nil
		}),
	}
}
