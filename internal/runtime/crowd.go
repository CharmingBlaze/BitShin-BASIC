package runtime

import (
	"math"

	"moonbasic/internal/mathx"
	"moonbasic/internal/value"
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
		if c == nil || len(c.agents) == 0 {
			continue
		}
		sep := c.sep
		if sep <= 0 {
			sep = 1.1
		}
		type body struct {
			id         int
			x, y, z    float32
			px, pz     float32
			speed, rad float32
		}
		bodies := make([]body, len(c.agents))
		disks := make([]mathx.Disk, len(c.agents))
		for i := range disks {
			disks[i].R = -1
		}
		for i, id := range c.agents {
			e := w.ents[id]
			if e == nil || e.node == nil {
				continue
			}
			p := worldPos(e.node.GetNode())
			x, y, z := fromG3N(p.X, p.Y, p.Z)
			ag := w.agents[id]
			rad := sep * 0.5
			speed := float32(4)
			if ag != nil {
				if ag.radius > 0 {
					rad = ag.radius
				}
				if ag.speed > 0 {
					speed = ag.speed
				}
			}
			prefX, prefZ := float32(0), float32(0)
			if ag != nil && ag.moving && ag.i < len(ag.path) {
				tgt := ag.path[ag.i]
				dx, dz := tgt[0]-x, tgt[2]-z
				dist := float32(math.Hypot(float64(dx), float64(dz)))
				if dist <= 0.12 {
					ag.i++
					if ag.i >= len(ag.path) {
						ag.moving = false
					}
				} else if dist > 0 {
					prefX = dx / dist * speed
					prefZ = dz / dist * speed
				}
			}
			bodies[i] = body{id: id, x: x, y: y, z: z, px: prefX, pz: prefZ, speed: speed, rad: rad}
			vx, vz := float64(prefX), float64(prefZ)
			if ag != nil && ag.hasLast {
				vx = float64(ag.vx)
				vz = float64(ag.vz)
			}
			disks[i] = mathx.Disk{X: float64(x), Z: float64(z), VX: vx, VZ: vz, R: float64(rad)}
		}
		for i, b := range bodies {
			if b.id == 0 {
				continue
			}
			e := w.ents[b.id]
			if e == nil || e.node == nil {
				continue
			}
			vx, vz := mathx.SampleDetour(i, disks, float64(b.px), float64(b.pz), float64(b.speed), 2.5)
			ax, az := float32(vx), float32(vz)
			// Sidestep when ORCA only slowed a head-on pair and a neighbor is still inside the radius.
			xs := make([]float64, len(bodies))
			zs := make([]float64, len(bodies))
			vxs := make([]float64, len(bodies))
			vzs := make([]float64, len(bodies))
			for k, o := range bodies {
				xs[k], zs[k] = float64(o.x), float64(o.z)
				vxs[k], vzs[k] = disks[k].VX, disks[k].VZ
			}
			sx, sz := mathx.Avoid2D(xs, zs, vxs, vzs, i, float64(b.rad), 1.2)
			ax += float32(sx)
			az += float32(sz)
			px := b.x + ax*dt
			py := b.y
			pz := b.z + az*dt
			if w.curTerrain != 0 {
				py = w.terrainHeight(px, pz) + 0.5
			}
			gx, gy, gz := toG3N(px, py, pz)
			e.node.GetNode().SetPosition(gx, gy, gz)
			if ax*ax+az*az > 0.0001 {
				e.node.GetNode().SetRotationY(float32(-math.Atan2(float64(ax), float64(az))))
			}
			if ag := w.agents[b.id]; ag != nil {
				ag.lastX, ag.lastZ, ag.hasLast = px, pz, true
				ag.vx, ag.vz = ax, az
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
			w.agents[ent].inCrowd = true
			c.agents = append(c.agents, ent)
			return value.Num(float64(ent)), nil
		}),
		"crowdsetagentdestination": n(func(a []value.Value) (value.Value, error) {
			ent := argI(a, 0, 0)
			off := 1
			if len(a) >= 5 {
				ent = argI(a, 1, 0)
				off = 2
			}
			ag := w.agents[ent]
			if ag == nil {
				return z()
			}
			x, y, zz := float32(argN(a, off, 0)), float32(argN(a, off+1, 0)), float32(argN(a, off+2, 0))
			pts, err := w.navFindPath(ag, x, y, zz)
			if err != nil || len(pts) == 0 {
				pts = [][3]float32{{x, y, zz}}
			}
			ag.path = pts
			ag.i = 0
			ag.moving = true
			return z()
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
		"getnavmaxslope": n(func(a []value.Value) (value.Value, error) {
			s := w.navMaxSlope
			if s <= 0 {
				s = 45
			}
			return value.Num(float64(s)), nil
		}),
	}
}
