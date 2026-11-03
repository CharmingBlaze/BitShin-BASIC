package runtime

import (
	"math"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// aiSense is one entity's vision cone, hearing radius, and flock group.
type aiSense struct {
	see, fov, hear float64
	group          int
	wander         float64
}

// aiUnit is the small CreateAI handle: a state name, waypoints, and an entity.
type aiUnit struct {
	ent   int
	state string
	pts   [][3]float64
	i     int
}

type trail struct {
	ent int
	max int
	pts [][3]float64
}

func (w *World) senseOf(id int) *aiSense {
	if w.senses == nil {
		w.senses = map[int]*aiSense{}
	}
	s := w.senses[id]
	if s == nil {
		s = &aiSense{see: 20, fov: 90, hear: 12}
		w.senses[id] = s
	}
	return s
}

func (w *World) aiCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"setvision": n(func(a []value.Value) (value.Value, error) {
			s := w.senseOf(argI(a, 0, 0))
			s.see = argN(a, 1, s.see)
			s.fov = argN(a, 2, s.fov)
			return z()
		}),
		"sethearing": n(func(a []value.Value) (value.Value, error) {
			w.senseOf(argI(a, 0, 0)).hear = argN(a, 1, 12)
			return z()
		}),
		"cansee": n(func(a []value.Value) (value.Value, error) {
			if w.canSee(argI(a, 0, 0), argI(a, 1, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"canhear": n(func(a []value.Value) (value.Value, error) {
			if w.canHear(argI(a, 0, 0), argI(a, 1, 0), argN(a, 2, 1)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"steerseek": n(func(a []value.Value) (value.Value, error) {
			return w.steer(argI(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0), argN(a, 4, 4), false)
		}),
		"steerflee": n(func(a []value.Value) (value.Value, error) {
			return w.steer(argI(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0), argN(a, 4, 4), true)
		}),
		"steerarrive": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			tx, ty, tz := argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
			speed := argN(a, 4, 4)
			slow := argN(a, 5, 2)
			x, y, z, ok := w.entXYZ(id)
			if !ok {
				return z0()
			}
			dx, dy, dz := tx-x, ty-y, tz-z
			dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
			if dist < 1e-4 {
				return value.Num(0), nil
			}
			if slow > 0 && dist < slow {
				speed *= dist / slow
			}
			return w.stepSteer(id, x, y, z, dx/dist, dy/dist, dz/dist, speed)
		}),
		"steerwander": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			s := w.senseOf(id)
			s.wander += argN(a, 1, 1)
			yaw := s.wander * 37
			return w.steer(id, math.Cos(yaw), 0, math.Sin(yaw), argN(a, 2, 2), false)
		}),
		"steeravoid": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			reach := argN(a, 1, 2)
			x, y, z, ok := w.entXYZ(id)
			if !ok {
				return z0()
			}
			var ax, az float64
			for oid, o := range w.ents {
				if oid == id || o == nil || o.node == nil {
					continue
				}
				ox, oy, oz, ok := w.entXYZ(oid)
				if !ok {
					continue
				}
				dx, dy, dz := x-ox, y-oy, z-oz
				d := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if d < 1e-4 || d > reach {
					continue
				}
				ax += dx / d
				az += dz / d
				_ = oy
			}
			if ax == 0 && az == 0 {
				return value.Num(0), nil
			}
			return w.stepSteer(id, x, y, z, ax, 0, az, argN(a, 2, 3))
		}),
		"setflock": n(func(a []value.Value) (value.Value, error) {
			w.senseOf(argI(a, 0, 0)).group = argI(a, 1, 1)
			return z()
		}),
		"flockupdate": n(func(a []value.Value) (value.Value, error) {
			return w.flockUpdate(argI(a, 0, 0), argN(a, 1, 6), argN(a, 2, 1), argN(a, 3, 0.4), argN(a, 4, 0.3))
		}),
		"createai": n(func(a []value.Value) (value.Value, error) {
			ent := argI(a, 0, 0)
			if w.ents[ent] == nil {
				n := core.NewNode()
				ent = w.addEntity(&Entity{kind: "ai", node: n}, 0)
			}
			if w.ais == nil {
				w.ais = map[int]*aiUnit{}
			}
			w.nextAI++
			w.ais[w.nextAI] = &aiUnit{ent: ent, state: "idle"}
			return value.Num(float64(w.nextAI)), nil
		}),
		"aistate": n(func(a []value.Value) (value.Value, error) {
			u := w.ais[argI(a, 0, 0)]
			if u == nil {
				return value.Str(""), nil
			}
			if len(a) >= 2 && argS(a, 1) != "" {
				u.state = argS(a, 1)
			}
			return value.Str(u.state), nil
		}),
		"aiaddwaypoint": n(func(a []value.Value) (value.Value, error) {
			u := w.ais[argI(a, 0, 0)]
			if u == nil {
				return value.Num(0), nil
			}
			u.pts = append(u.pts, [3]float64{argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)})
			return value.Num(float64(len(u.pts))), nil
		}),
		"aidetecttarget": n(func(a []value.Value) (value.Value, error) {
			u := w.ais[argI(a, 0, 0)]
			if u == nil {
				return value.Num(0), nil
			}
			if w.canSee(u.ent, argI(a, 1, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"aiupdate": n(func(a []value.Value) (value.Value, error) {
			u := w.ais[argI(a, 0, 0)]
			if u == nil || len(u.pts) == 0 {
				return value.Num(0), nil
			}
			if u.i >= len(u.pts) {
				u.i = 0
			}
			p := u.pts[u.i]
			x, y, z, ok := w.entXYZ(u.ent)
			if !ok {
				return value.Num(0), nil
			}
			dx, dy, dz := p[0]-x, p[1]-y, p[2]-z
			if math.Sqrt(dx*dx+dy*dy+dz*dz) < 0.4 {
				u.i = (u.i + 1) % len(u.pts)
				p = u.pts[u.i]
			}
			return w.steer(u.ent, p[0], p[1], p[2], argN(a, 1, 4), false)
		}),
		"createtrail": n(func(a []value.Value) (value.Value, error) {
			ent := argI(a, 0, 0)
			if w.ents[ent] == nil {
				return value.Num(0), nil
			}
			max := argI(a, 1, 16)
			if max < 2 {
				max = 2
			}
			if max > 64 {
				max = 64
			}
			if w.trails == nil {
				w.trails = map[int]*trail{}
			}
			w.nextTrail++
			w.trails[w.nextTrail] = &trail{ent: ent, max: max}
			return value.Num(float64(w.nextTrail)), nil
		}),
		"aidebug": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			x, y, z, ok := w.entXYZ(id)
			if !ok {
				return z0()
			}
			s := w.senseOf(id)
			e := w.ents[id]
			yaw := 0.0
			if e != nil {
				yaw = float64(e.yaw) * math.Pi / 180
			}
			fx, fz := math.Sin(yaw), math.Cos(yaw)
			w.Call("debugline", []value.Value{
				value.Num(x), value.Num(y), value.Num(z),
				value.Num(x + fx*s.see), value.Num(y), value.Num(z + fz*s.see),
				value.Num(80), value.Num(220), value.Num(120), value.Num(0.1),
			})
			half := s.fov * 0.5 * math.Pi / 180
			for _, off := range []float64{half, -half} {
				ex, ez := math.Sin(yaw+off), math.Cos(yaw+off)
				w.Call("debugline", []value.Value{
					value.Num(x), value.Num(y), value.Num(z),
					value.Num(x + ex*s.see), value.Num(y), value.Num(z + ez*s.see),
					value.Num(80), value.Num(180), value.Num(90), value.Num(0.1),
				})
			}
			return value.Num(1), nil
		}),
	}
}

func z0() (value.Value, error) { return value.Num(0), nil }

func (w *World) tickTrails() {
	for _, t := range w.trails {
		if t == nil {
			continue
		}
		x, y, z, ok := w.entXYZ(t.ent)
		if !ok {
			continue
		}
		t.pts = append(t.pts, [3]float64{x, y, z})
		if len(t.pts) > t.max {
			t.pts = t.pts[len(t.pts)-t.max:]
		}
		for i := 1; i < len(t.pts); i++ {
			a, b := t.pts[i-1], t.pts[i]
			w.Call("debugline", []value.Value{
				value.Num(a[0]), value.Num(a[1]), value.Num(a[2]),
				value.Num(b[0]), value.Num(b[1]), value.Num(b[2]),
				value.Num(230), value.Num(180), value.Num(70), value.Num(0.05),
			})
		}
	}
}

func (w *World) entXYZ(id int) (x, y, z float64, ok bool) {
	e := w.ents[id]
	if e == nil || e.node == nil {
		return 0, 0, 0, false
	}
	p := worldPos(e.node.GetNode())
	fx, fy, fz := fromG3N(p.X, p.Y, p.Z)
	return float64(fx), float64(fy), float64(fz), true
}

func (w *World) canSee(from, target int) bool {
	x, y, z, ok := w.entXYZ(from)
	tx, ty, tz, ok2 := w.entXYZ(target)
	if !ok || !ok2 {
		return false
	}
	s := w.senseOf(from)
	dx, dy, dz := tx-x, ty-y, tz-z
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if s.see > 0 && dist > s.see {
		return false
	}
	e := w.ents[from]
	if e != nil && s.fov > 0 && s.fov < 360 {
		yaw := float64(e.yaw) * math.Pi / 180
		fx, fz := math.Sin(yaw), math.Cos(yaw)
		flat := math.Hypot(dx, dz)
		if flat > 1e-4 {
			dot := (fx*dx + fz*dz) / flat
			if dot > 1 {
				dot = 1
			}
			if dot < -1 {
				dot = -1
			}
			ang := math.Acos(dot) * 180 / math.Pi
			if ang > s.fov*0.5 {
				return false
			}
		}
	}
	_ = dy
	return !w.sightBlocked(from, target, x, y, z, tx, ty, tz)
}

// sightBlocked is true when another entity's radius sits on the segment.
func (w *World) sightBlocked(from, target int, x, y, z, tx, ty, tz float64) bool {
	dx, dy, dz := tx-x, ty-y, tz-z
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if dist < 1e-4 {
		return false
	}
	ndx, ndy, ndz := dx/dist, dy/dist, dz/dist
	for id, e := range w.ents {
		if id == from || id == target || e == nil || e.node == nil || e.cam != nil {
			continue
		}
		ex, ey, ez, ok := w.entXYZ(id)
		if !ok {
			continue
		}
		ox, oy, oz := ex-x, ey-y, ez-z
		t := ox*ndx + oy*ndy + oz*ndz
		if t <= 0.05 || t >= dist-0.05 {
			continue
		}
		ddx, ddy, ddz := ox-ndx*t, oy-ndy*t, oz-ndz*t
		r := float64(e.radius)
		if r <= 0 {
			r = 1
		}
		if ddx*ddx+ddy*ddy+ddz*ddz <= r*r {
			return true
		}
	}
	return false
}

func (w *World) canHear(from, target int, loud float64) bool {
	x, _, z, ok := w.entXYZ(from)
	tx, _, tz, ok2 := w.entXYZ(target)
	if !ok || !ok2 {
		return false
	}
	s := w.senseOf(from)
	dx, dz := tx-x, tz-z
	dist := math.Hypot(dx, dz)
	reach := s.hear * loud
	if reach <= 0 {
		reach = s.hear
	}
	return dist <= reach
}

func (w *World) steer(id int, tx, ty, tz, speed float64, flee bool) (value.Value, error) {
	x, y, z, ok := w.entXYZ(id)
	if !ok {
		return value.Num(0), nil
	}
	dx, dy, dz := tx-x, ty-y, tz-z
	if flee {
		dx, dy, dz = -dx, -dy, -dz
	}
	l := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if l < 1e-6 {
		return value.Num(0), nil
	}
	return w.stepSteer(id, x, y, z, dx/l, dy/l, dz/l, speed)
}

func (w *World) stepSteer(id int, x, y, z, dx, dy, dz, speed float64) (value.Value, error) {
	dt := w.delta
	if dt <= 0 {
		dt = 1.0 / 60
	}
	l := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if l < 1e-6 {
		return value.Num(0), nil
	}
	dx, dy, dz = dx/l, dy/l, dz/l
	step := speed * dt
	nx, ny, nz := x+dx*step, y+dy*step, z+dz*step
	e := w.ents[id]
	if e == nil || e.node == nil {
		return value.Num(0), nil
	}
	gx, gy, gz := toG3N(float32(nx), float32(ny), float32(nz))
	e.node.GetNode().SetPosition(gx, gy, gz)
	if dz != 0 || dx != 0 {
		e.yaw = float32(math.Atan2(dx, dz) * 180 / math.Pi)
		e.node.GetNode().SetRotationY(-e.yaw * math32.Pi / 180)
	}
	return value.Num(step), nil
}

func (w *World) flockUpdate(id int, reach, sep, align, cohere float64) (value.Value, error) {
	s := w.senseOf(id)
	x, y, z, ok := w.entXYZ(id)
	if !ok {
		return value.Num(0), nil
	}
	var sx, sz, ax, az, cx, cz float64
	n := 0
	for oid, o := range w.ents {
		if oid == id || o == nil {
			continue
		}
		os := w.senses[oid]
		if os == nil || os.group != s.group || s.group == 0 {
			continue
		}
		ox, _, oz, ok := w.entXYZ(oid)
		if !ok {
			continue
		}
		dx, dz := ox-x, oz-z
		d := math.Hypot(dx, dz)
		if d > reach || d < 1e-4 {
			continue
		}
		sx -= dx / d
		sz -= dz / d
		ax += math.Sin(float64(o.yaw) * math.Pi / 180)
		az += math.Cos(float64(o.yaw) * math.Pi / 180)
		cx += ox
		cz += oz
		n++
	}
	if n == 0 {
		return value.Num(0), nil
	}
	cx = cx/float64(n) - x
	cz = cz/float64(n) - z
	dx := sx*sep + ax*align + cx*cohere
	dz := sz*sep + az*align + cz*cohere
	return w.stepSteer(id, x, y, z, dx, 0, dz, 3)
}
