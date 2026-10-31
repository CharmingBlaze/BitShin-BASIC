package phys2d

import (
	"math"

	"github.com/jakecoffman/cp/v2"
)

// Hit is the result of a 2D ray or shape cast. X, Y is the contact point on
// the surface that was hit, NX, NY that surface's normal, and Fraction how far
// along the path (0..1) the shape got before it touched.
type Hit struct {
	OK       bool
	ID       int
	X, Y     float64
	NX, NY   float64
	Fraction float64
}

// LastHit is the result of the most recent RaycastHit / CircleCast / BoxCast.
func (s *Space) LastHit() Hit { return s.ext.last }

func (s *Space) keep(h Hit) Hit {
	s.ext.last = h
	return h
}

// RaycastHit is Raycast with the normal and fraction filled in. Sensors are skipped.
func (s *Space) RaycastHit(x1, y1, x2, y2 float64) Hit {
	return s.CircleCast(x1, y1, x2, y2, 0, 0)
}

// CircleCast sweeps a circle of radius r from (x1,y1) to (x2,y2) and returns
// the first non-sensor shape it touches, skipping body ignore (0 = none).
func (s *Space) CircleCast(x1, y1, x2, y2, r float64, ignore int) Hit {
	if r < 0 {
		r = 0
	}
	a, b := cp.Vector{X: x1, Y: y1}, cp.Vector{X: x2, Y: y2}
	best := Hit{Fraction: 2}
	s.sp.SegmentQuery(a, b, r, cp.SHAPE_FILTER_ALL, func(sh *cp.Shape, p, n cp.Vector, alpha float64, _ any) {
		if sh.Sensor() || (ignore != 0 && idOf(sh.Body()) == ignore) {
			return
		}
		if alpha < best.Fraction {
			best = Hit{OK: true, ID: idOf(sh.Body()), X: p.X, Y: p.Y, NX: n.X, NY: n.Y, Fraction: alpha}
		}
	}, nil)
	if !best.OK {
		best.Fraction = 1
	}
	return s.keep(best)
}

// BoxCast sweeps an axis-aligned w x h box centred on the path. Chipmunk has
// no box sweep, so it steps the box along the path at a quarter of its
// smaller side (so nothing thinner can slip between samples), then bisects
// the first overlapping step.
func (s *Space) BoxCast(x1, y1, x2, y2, w, h float64, ignore int) Hit {
	if w <= 0 || h <= 0 {
		return s.CircleCast(x1, y1, x2, y2, math.Max(w, h)/2, ignore)
	}
	a, b := cp.Vector{X: x1, Y: y1}, cp.Vector{X: x2, Y: y2}
	body := cp.NewKinematicBody()
	box := cp.NewBox(body, w, h, 0)
	overlap := func(t float64) (Hit, bool) {
		body.SetPosition(a.Lerp(b, t))
		var hit Hit
		depth := math.Inf(1)
		s.sp.ShapeQuery(box, func(sh *cp.Shape, set *cp.ContactPointSet) {
			if sh.Sensor() || (ignore != 0 && idOf(sh.Body()) == ignore) || set.Count == 0 {
				return
			}
			d := set.Points[0].Distance
			var px, py float64
			for i := 0; i < set.Count; i++ {
				px += set.Points[i].PointB.X
				py += set.Points[i].PointB.Y
				if set.Points[i].Distance < d {
					d = set.Points[i].Distance
				}
			}
			if d < depth {
				depth = d
				n := set.Normal.Neg()
				c := float64(set.Count)
				hit = Hit{OK: true, ID: idOf(sh.Body()), X: px / c, Y: py / c, NX: n.X, NY: n.Y}
			}
		})
		return hit, hit.OK
	}
	if hit, ok := overlap(0); ok {
		hit.Fraction = 0
		return s.keep(hit)
	}
	dist := a.Distance(b)
	steps := int(math.Ceil(dist / (math.Min(w, h) * 0.25)))
	if steps < 1 {
		steps = 1
	}
	if steps > 512 {
		steps = 512
	}
	lo := 0.0
	for k := 1; k <= steps; k++ {
		hi := float64(k) / float64(steps)
		hit, ok := overlap(hi)
		if !ok {
			lo = hi
			continue
		}
		for i := 0; i < 16; i++ {
			mid := (lo + hi) / 2
			if mh, mok := overlap(mid); mok {
				hi, hit = mid, mh
			} else {
				lo = mid
			}
		}
		hit.Fraction = lo
		return s.keep(hit)
	}
	return s.keep(Hit{Fraction: 1})
}
