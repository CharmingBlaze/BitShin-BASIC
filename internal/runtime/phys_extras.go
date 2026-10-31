package runtime

import (
	"math"

	"moonbasic/internal/phys3d"
	"moonbasic/internal/value"
)

type pathFollow struct {
	pts   [][3]float32
	speed float32
	dist  float32
	loop  bool
}

func compoundKindOf(e *Entity) (kind int, a, b, c float32) {
	if e == nil {
		return phys3d.CompoundBox, 0.5, 0.5, 0.5
	}
	switch e.collKind {
	case 2:
		r := e.radius
		if r < 0.05 {
			r = 0.5
		}
		return phys3d.CompoundSphere, r, 0, 0
	case 3:
		r := e.radius
		if r < 0.05 {
			r = 0.4
		}
		h := e.boxY
		if h < 0.05 {
			h = 0.5
		}
		return phys3d.CompoundCapsule, h, r, 0
	case 4:
		r := e.radius
		if r < 0.05 {
			r = 0.4
		}
		h := e.boxY
		if h < 0.05 {
			h = 0.5
		}
		return phys3d.CompoundCylinder, h, r, 0
	default:
		hx, hy, hz := e.boxX, e.boxY, e.boxZ
		if hx < 0.05 {
			hx = 0.5
		}
		if hy < 0.05 {
			hy = 0.5
		}
		if hz < 0.05 {
			hz = 0.5
		}
		return phys3d.CompoundBox, hx, hy, hz
	}
}

func (w *World) gatherCompoundParts(parent int) ([]phys3d.CompoundPart, []int) {
	pe := w.ents[parent]
	px, py, pz := w.blitzPosOf(parent)
	parts := []phys3d.CompoundPart{}
	drop := []int{}
	if pe != nil && pe.mesh != nil {
		kind, a, b, c := compoundKindOf(pe)
		parts = append(parts, phys3d.CompoundPart{Kind: kind, A: a, B: b, C: c})
	}
	for id, e := range w.ents {
		if e == nil || id == parent || e.parent != parent {
			continue
		}
		if e.mesh == nil && e.bodyType == 0 {
			continue
		}
		cx, cy, cz := w.blitzPosOf(id)
		kind, a, b, c := compoundKindOf(e)
		parts = append(parts, phys3d.CompoundPart{
			Kind: kind,
			Ox:   cx - px, Oy: cy - py, Oz: cz - pz,
			A: a, B: b, C: c,
		})
		if e.bodyType != 0 {
			drop = append(drop, id)
		}
	}
	if len(parts) == 0 && pe != nil {
		kind, a, b, c := compoundKindOf(pe)
		parts = append(parts, phys3d.CompoundPart{Kind: kind, A: a, B: b, C: c})
	}
	return parts, drop
}

func (w *World) explodeAt(x, y, z, r, imp float32) int {
	if r <= 0 {
		r = 1
	}
	w.ensurePhys3()
	hits := w.phys3.OverlapSphereAll(x, y, z, r, 64)
	n := 0
	for _, id := range hits {
		e := w.ents[id]
		if e == nil || e.bodyType != 1 {
			continue
		}
		px, py, pz := w.blitzPosOf(id)
		dx, dy, dz := px-x, py-y, pz-z
		d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if d < 1e-4 {
			dy = 1
			d = 1
		}
		fall := 1 - d/r
		if fall <= 0 {
			continue
		}
		s := imp * fall / d
		w.phys3.ApplyImpulse(id, dx*s, dy*s, dz*s)
		n++
	}
	return n
}

func polylineLength(pts [][3]float32) float32 {
	var total float32
	for i := 1; i < len(pts); i++ {
		dx := pts[i][0] - pts[i-1][0]
		dy := pts[i][1] - pts[i-1][1]
		dz := pts[i][2] - pts[i-1][2]
		total += float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	}
	return total
}

func pointOnPolyline(pts [][3]float32, dist float32) (float32, float32, float32) {
	if len(pts) == 0 {
		return 0, 0, 0
	}
	if len(pts) == 1 || dist <= 0 {
		return pts[0][0], pts[0][1], pts[0][2]
	}
	remain := dist
	for i := 1; i < len(pts); i++ {
		dx := pts[i][0] - pts[i-1][0]
		dy := pts[i][1] - pts[i-1][1]
		dz := pts[i][2] - pts[i-1][2]
		seg := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if seg < 1e-6 {
			continue
		}
		if remain <= seg {
			t := remain / seg
			return pts[i-1][0] + dx*t, pts[i-1][1] + dy*t, pts[i-1][2] + dz*t
		}
		remain -= seg
	}
	last := pts[len(pts)-1]
	return last[0], last[1], last[2]
}

func (w *World) updatePathFollows(dt float32) {
	if len(w.pathFollows) == 0 {
		return
	}
	for id, p := range w.pathFollows {
		e := w.ents[id]
		if e == nil || e.node == nil || p == nil || len(p.pts) < 2 {
			delete(w.pathFollows, id)
			continue
		}
		total := polylineLength(p.pts)
		if total < 1e-4 {
			continue
		}
		p.dist += p.speed * dt
		if p.loop {
			p.dist = float32(math.Mod(float64(p.dist), float64(total)))
			if p.dist < 0 {
				p.dist += total
			}
		} else if p.dist > total {
			p.dist = total
		}
		x, y, z := pointOnPolyline(p.pts, p.dist)
		gx, gy, gz := toG3N(x, y, z)
		e.node.GetNode().SetPosition(gx, gy, gz)
		if w.phys3 != nil && e.bodyType != 0 {
			w.phys3.SetPosition(id, x, y, z)
		}
	}
}

// drawPhysicsDebug is the old per-step hook. The physics view is now part
// of the debug overlay (debug_draw.go), drawn once per frame in render.
func (w *World) drawPhysicsDebug() {}

func parsePathArgs(a []value.Value) (speed float32, loop bool, pts [][3]float32) {
	speed = float32(argN(a, 1, 4))
	start := 2
	rest := len(a) - 2
	if rest >= 7 && rest%3 == 1 {
		loop = argI(a, 2, 0) != 0
		start = 3
	}
	for i := start; i+2 < len(a); i += 3 {
		pts = append(pts, [3]float32{float32(argN(a, i, 0)), float32(argN(a, i+1, 0)), float32(argN(a, i+2, 0))})
	}
	return speed, loop, pts
}
