package runtime

import (
	"math"

	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type ropeVec struct{ x, y, z float32 }

type ropeSystem struct {
	id, a, b             int
	anchorA, anchorB     ropeVec // local to each body; world coordinates when body is 0
	length, radius       float32
	mass, damping        float32
	stiffness, lineDamp  float32
	maxForce, tension    float32
	segments          int
	links             []int
	pos, prev         []ropeVec
	rest               []float32
	visible            bool
	color              [3]float32
}

func ropeDistance(a, b ropeVec) float32 {
	dx, dy, dz := b.x-a.x, b.y-a.y, b.z-a.z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

func ropeMid(a, b ropeVec) ropeVec {
	return ropeVec{(a.x + b.x) * 0.5, (a.y + b.y) * 0.5, (a.z + b.z) * 0.5}
}

func (w *World) ropeAnchorWorld(body int, local ropeVec) (ropeVec, bool) {
	if body == 0 {
		return local, true
	}
	x, y, z, ok := w.phys3.GetPosition(body)
	if !ok {
		return ropeVec{}, false
	}
	fx, fy, fz, ux, uy, uz, rx, ry, rz := w.physAxes(body)
	return ropeVec{
		x: x + rx*local.x + ux*local.y + fx*local.z,
		y: y + ry*local.x + uy*local.y + fy*local.z,
		z: z + rz*local.x + uz*local.y + fz*local.z,
	}, true
}

// ropeCurve returns endpoints plus dynamic-node positions. The downward sine
// creates enough initial arc length for a slack rope instead of constructing a
// straight, already-taut chain.
func ropeCurve(a, b ropeVec, length float32, segments int) []ropeVec {
	if segments < 2 {
		segments = 2
	}
	direct := ropeDistance(a, b)
	if length < direct*1.001 {
		length = direct * 1.001
	}
	makeCurve := func(sag float32) []ropeVec {
		pts := make([]ropeVec, segments+1)
		for i := 0; i <= segments; i++ {
			t := float32(i) / float32(segments)
			pts[i] = ropeVec{
				x: a.x + (b.x-a.x)*t,
				y: a.y + (b.y-a.y)*t - sag*float32(math.Sin(float64(math.Pi*t))),
				z: a.z + (b.z-a.z)*t,
			}
		}
		return pts
	}
	curveLength := func(pts []ropeVec) float32 {
		total := float32(0)
		for i := 1; i < len(pts); i++ {
			total += ropeDistance(pts[i-1], pts[i])
		}
		return total
	}
	lo, hi := float32(0), length
	for curveLength(makeCurve(hi)) < length {
		hi *= 2
	}
	for i := 0; i < 18; i++ {
		mid := (lo + hi) * 0.5
		if curveLength(makeCurve(mid)) < length {
			lo = mid
		} else {
			hi = mid
		}
	}
	return makeCurve((lo + hi) * 0.5)
}

func (w *World) nextRopeHandle() int {
	w.nextRope++
	if w.nextRope < 1 {
		w.nextRope = 1
	}
	return w.nextRope
}

func (w *World) createRope(a, b int, anchorA, anchorB ropeVec, length float32, segments int, radius float32) int {
	w.ensurePhys3()
	pa, okA := w.ropeAnchorWorld(a, anchorA)
	pb, okB := w.ropeAnchorWorld(b, anchorB)
	if !okA || !okB || (a == 0 && b == 0) {
		return 0
	}
	if segments < 2 {
		segments = 12
	}
	if segments > 32 {
		segments = 32
	}
	direct := ropeDistance(pa, pb)
	if length <= 0 {
		length = direct * 1.06
	}
	if length < direct*1.001 {
		length = direct * 1.001
	}
	if radius <= 0 {
		radius = 0.035
	}
	r := &ropeSystem{
		id: w.nextRopeHandle(), a: a, b: b, anchorA: anchorA, anchorB: anchorB,
		length: length, radius: radius, mass: 0.12, damping: 0.18,
		stiffness: 5000, lineDamp: 600, maxForce: 10000,
		segments: segments, visible: true, color: [3]float32{220, 196, 126},
	}
	if w.ropes == nil {
		w.ropes = map[int]*ropeSystem{}
	}
	w.ropes[r.id] = r
	pts := ropeCurve(pa, pb, length, segments)
	for i := 1; i < len(pts)-1; i++ {
		r.pos = append(r.pos, pts[i])
		r.prev = append(r.prev, pts[i])
	}
	for i := 1; i < len(pts); i++ {
		r.rest = append(r.rest, ropeDistance(pts[i-1], pts[i]))
	}
	for i := 0; i < segments; i++ {
		linkID := w.createCubeMesh(nil)
		if e := w.ents[linkID]; e != nil {
			w.setEntityRGB(e, r.color[0], r.color[1], r.color[2])
			e.meshNoCast = true
		}
		r.links = append(r.links, linkID)
	}
	w.tickRope(r)
	return r.id
}

func (w *World) resetRope(r *ropeSystem) {
	pa, okA := w.ropeAnchorWorld(r.a, r.anchorA)
	pb, okB := w.ropeAnchorWorld(r.b, r.anchorB)
	if !okA || !okB {
		return
	}
	pts := ropeCurve(pa, pb, r.length, r.segments)
	for i := range r.pos {
		r.pos[i] = pts[i+1]
		r.prev[i] = pts[i+1]
	}
	r.rest = r.rest[:0]
	for i := 1; i < len(pts); i++ {
		r.rest = append(r.rest, ropeDistance(pts[i-1], pts[i]))
	}
	r.tension = 0
	w.tickRope(r)
}

func (w *World) ropePoints(r *ropeSystem) ([]ropeVec, bool) {
	pa, okA := w.ropeAnchorWorld(r.a, r.anchorA)
	pb, okB := w.ropeAnchorWorld(r.b, r.anchorB)
	if !okA || !okB {
		return nil, false
	}
	pts := make([]ropeVec, 0, len(r.pos)+2)
	pts = append(pts, pa)
	pts = append(pts, r.pos...)
	pts = append(pts, pb)
	return pts, true
}

func (w *World) tickRope(r *ropeSystem) {
	pts, ok := w.ropePoints(r)
	if !ok || len(pts)-1 != len(r.links) {
		return
	}
	for i, linkID := range r.links {
		e := w.ents[linkID]
		if e == nil || e.node == nil {
			continue
		}
		n := e.node.GetNode()
		n.SetVisible(r.visible)
		if !r.visible {
			continue
		}
		a, b := pts[i], pts[i+1]
		mid := ropeMid(a, b)
		gx, gy, gz := toG3N(mid.x, mid.y, mid.z)
		n.SetPosition(gx, gy, gz)
		tx, ty, tz := toG3N(b.x, b.y, b.z)
		target := math32.Vector3{X: tx, Y: ty, Z: tz}
		up := math32.Vector3{X: 0, Y: 1, Z: 0}
		distance := ropeDistance(a, b)
		if distance > 0.0001 && float32(math.Abs(float64((b.y-a.y)/distance))) > 0.97 {
			up = math32.Vector3{X: 1, Y: 0, Z: 0}
		}
		n.LookAt(&target, &up)
		n.SetScale(r.radius, r.radius, distance*0.5)
	}
}

func (w *World) tickRopes() {
	for _, r := range w.ropes {
		if r != nil {
			w.tickRope(r)
		}
	}
}

func (w *World) collideRopePoint(r *ropeSystem, p *ropeVec) {
	if p == nil {
		return
	}
	if len(w.terrains) > 0 {
		ground := w.terrainHeight(p.x, p.z) + r.radius
		if p.y < ground {
			p.y = ground
		}
	}
	for id, e := range w.ents {
		if e == nil || e.bodyType == 0 || id == r.a || id == r.b {
			continue
		}
		x, y, z, ok := w.phys3.GetPosition(id)
		if !ok {
			continue
		}
		hx, hy, hz := e.boxX+r.radius, e.boxY+r.radius, e.boxZ+r.radius
		dx, dy, dz := p.x-x, p.y-y, p.z-z
		ax, ay, az := float32(math.Abs(float64(dx))), float32(math.Abs(float64(dy))), float32(math.Abs(float64(dz)))
		if ax >= hx || ay >= hy || az >= hz {
			continue
		}
		px, py, pz := hx-ax, hy-ay, hz-az
		switch {
		case py <= px && py <= pz:
			if dy < 0 {
				p.y = y - hy
			} else {
				p.y = y + hy
			}
		case px <= pz:
			if dx < 0 {
				p.x = x - hx
			} else {
				p.x = x + hx
			}
		default:
			if dz < 0 {
				p.z = z - hz
			} else {
				p.z = z + hz
			}
		}
	}
}

func (w *World) simulateRopes(dt float32) {
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	if dt > 0.05 {
		dt = 0.05
	}
	for _, r := range w.ropes {
		if r == nil || len(r.pos) == 0 || len(r.rest) != r.segments {
			continue
		}
		pa, okA := w.ropeAnchorWorld(r.a, r.anchorA)
		pb, okB := w.ropeAnchorWorld(r.b, r.anchorB)
		if !okA || !okB {
			continue
		}
		keep := float32(1) - r.damping*dt*2.5
		if keep < 0.65 {
			keep = 0.65
		}
		if keep > 1 {
			keep = 1
		}
		for i := range r.pos {
			cur := r.pos[i]
			vel := ropeVec{(cur.x - r.prev[i].x) * keep, (cur.y - r.prev[i].y) * keep, (cur.z - r.prev[i].z) * keep}
			r.prev[i] = cur
			r.pos[i].x += vel.x
			r.pos[i].y += vel.y - 9.81*dt*dt
			r.pos[i].z += vel.z
		}

		for iteration := 0; iteration < 12; iteration++ {
			for i := 0; i < r.segments; i++ {
				a := pa
				b := pb
				if i > 0 {
					a = r.pos[i-1]
				}
				if i < r.segments-1 {
					b = r.pos[i]
				}
				dx, dy, dz := b.x-a.x, b.y-a.y, b.z-a.z
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
				if dist < 0.00001 {
					continue
				}
				corr := (dist - r.rest[i]) / dist
				switch {
				case i == 0:
					r.pos[0].x -= dx * corr
					r.pos[0].y -= dy * corr
					r.pos[0].z -= dz * corr
				case i == r.segments-1:
					last := len(r.pos) - 1
					r.pos[last].x += dx * corr
					r.pos[last].y += dy * corr
					r.pos[last].z += dz * corr
				default:
					half := corr * 0.5
					r.pos[i-1].x += dx * half
					r.pos[i-1].y += dy * half
					r.pos[i-1].z += dz * half
					r.pos[i].x -= dx * half
					r.pos[i].y -= dy * half
					r.pos[i].z -= dz * half
				}
			}
			if iteration >= 7 {
				for i := range r.pos {
					w.collideRopePoint(r, &r.pos[i])
				}
			}
		}
	}
}

func (w *World) applyRopeForces() {
	if w.phys3 == nil {
		return
	}
	for _, r := range w.ropes {
		if r == nil || r.length <= 0 {
			continue
		}
		pa, okA := w.ropeAnchorWorld(r.a, r.anchorA)
		pb, okB := w.ropeAnchorWorld(r.b, r.anchorB)
		if !okA || !okB {
			r.tension = 0
			continue
		}
		ropeWeight := r.mass * float32(len(r.pos)) * 9.81 * 0.5
		if r.a != 0 && ropeWeight > 0 {
			w.phys3.ApplyForceAtPosition(r.a, 0, -ropeWeight, 0, pa.x, pa.y, pa.z)
		}
		if r.b != 0 && ropeWeight > 0 {
			w.phys3.ApplyForceAtPosition(r.b, 0, -ropeWeight, 0, pb.x, pb.y, pb.z)
		}
		dx, dy, dz := pb.x-pa.x, pb.y-pa.y, pb.z-pa.z
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if dist <= r.length || dist < 0.0001 {
			r.tension = 0
			continue
		}
		nx, ny, nz := dx/dist, dy/dist, dz/dist
		avx, avy, avz := float32(0), float32(0), float32(0)
		bvx, bvy, bvz := float32(0), float32(0), float32(0)
		if r.a != 0 {
			avx, avy, avz, _ = w.phys3.GetVelocity(r.a)
		}
		if r.b != 0 {
			bvx, bvy, bvz, _ = w.phys3.GetVelocity(r.b)
		}
		stretchRate := (bvx-avx)*nx + (bvy-avy)*ny + (bvz-avz)*nz
		force := (dist-r.length)*r.stiffness + stretchRate*r.lineDamp
		if force < 0 {
			force = 0
		}
		if r.maxForce > 0 && force > r.maxForce {
			force = r.maxForce
		}
		if r.maxForce > 0 {
			r.tension = force / r.maxForce
		} else {
			r.tension = 0
		}
		if r.a != 0 {
			w.phys3.ApplyForceAtPosition(r.a, nx*force, ny*force, nz*force, pa.x, pa.y, pa.z)
			w.phys3.Wake(r.a)
		}
		if r.b != 0 {
			w.phys3.ApplyForceAtPosition(r.b, -nx*force, -ny*force, -nz*force, pb.x, pb.y, pb.z)
			w.phys3.Wake(r.b)
		}
	}
}

func (w *World) freeRope(id int) {
	r := w.ropes[id]
	if r == nil {
		return
	}
	delete(w.ropes, id)
	for _, link := range r.links {
		w.freeEntityID(link)
	}
}

func (w *World) ropeTension(r *ropeSystem) float32 {
	if r == nil || r.tension < 0 {
		return 0
	}
	if r.tension > 1 {
		return 1
	}
	return r.tension
}

func (w *World) ropeBetween(a, b int) *ropeSystem {
	for _, r := range w.ropes {
		if r != nil && ((r.a == a && r.b == b) || (r.a == b && r.b == a)) {
			return r
		}
	}
	return nil
}

func (w *World) ropeCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createrope": need(func(a []value.Value) (value.Value, error) {
			id := w.createRope(argI(a, 0, 0), argI(a, 1, 0), ropeVec{}, ropeVec{},
				float32(argN(a, 2, 0)), argI(a, 3, 12), float32(argN(a, 4, 0.035)))
			return value.Num(float64(id)), nil
		}),
		"createropeanchored": need(func(a []value.Value) (value.Value, error) {
			id := w.createRope(argI(a, 0, 0), argI(a, 1, 0),
				ropeVec{float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0))},
				ropeVec{float32(argN(a, 5, 0)), float32(argN(a, 6, 0)), float32(argN(a, 7, 0))},
				float32(argN(a, 8, 0)), argI(a, 9, 12), float32(argN(a, 10, 0.035)))
			return value.Num(float64(id)), nil
		}),
		"setropecolor": need(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				r.color = [3]float32{float32(argN(a, 1, 220)), float32(argN(a, 2, 196)), float32(argN(a, 3, 126))}
				for _, link := range r.links {
					w.setEntityRGB(w.ents[link], r.color[0], r.color[1], r.color[2])
				}
			}
			return z()
		}),
		"setropemass": need(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				mass := float32(argN(a, 1, float64(r.mass)))
				if mass > 0 {
					r.mass = mass
				}
			}
			return z()
		}),
		"setropedamping": need(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				r.damping = float32(argN(a, 1, float64(r.damping)))
				if r.damping < 0 {
					r.damping = 0
				}
			}
			return z()
		}),
		"setropestrength": need(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				r.stiffness = float32(argN(a, 1, float64(r.stiffness)))
				r.lineDamp = float32(argN(a, 2, float64(r.lineDamp)))
				r.maxForce = float32(argN(a, 3, float64(r.maxForce)))
				if r.stiffness < 0 {
					r.stiffness = 0
				}
				if r.lineDamp < 0 {
					r.lineDamp = 0
				}
				if r.maxForce < 0 {
					r.maxForce = 0
				}
			}
			return z()
		}),
		"setropevisible": need(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				r.visible = argI(a, 1, 1) != 0
				w.tickRope(r)
			}
			return z()
		}),
		"resetrope": need(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				w.resetRope(r)
			}
			return z()
		}),
		"freerope": need(func(a []value.Value) (value.Value, error) {
			w.freeRope(argI(a, 0, 0))
			return z()
		}),
		"ropelength": n(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				return value.Num(float64(r.length)), nil
			}
			return value.Num(0), nil
		}),
		"ropetension": n(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				return value.Num(float64(w.ropeTension(r))), nil
			}
			return value.Num(0), nil
		}),
		"ropesegments": n(func(a []value.Value) (value.Value, error) {
			if r := w.ropes[argI(a, 0, 0)]; r != nil {
				return value.Num(float64(r.segments)), nil
			}
			return value.Num(0), nil
		}),
	}
}
