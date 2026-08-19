package runtime

import (
	"math"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/phys3d"
	"bitshinbasic/internal/value"
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

func appendBoxEdges(dst *math32.ArrayF32, c math32.Vector3, hx, hy, hz float32) {
	corners := [8]math32.Vector3{
		{c.X - hx, c.Y - hy, c.Z - hz}, {c.X + hx, c.Y - hy, c.Z - hz}, {c.X + hx, c.Y + hy, c.Z - hz}, {c.X - hx, c.Y + hy, c.Z - hz},
		{c.X - hx, c.Y - hy, c.Z + hz}, {c.X + hx, c.Y - hy, c.Z + hz}, {c.X + hx, c.Y + hy, c.Z + hz}, {c.X - hx, c.Y + hy, c.Z + hz},
	}
	edges := [12][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 0}, {4, 5}, {5, 6}, {6, 7}, {7, 4}, {0, 4}, {1, 5}, {2, 6}, {3, 7}}
	for _, e := range edges {
		a, b := corners[e[0]], corners[e[1]]
		dst.Append(a.X, a.Y, a.Z, b.X, b.Y, b.Z)
	}
}

func (w *World) replaceDebugMesh(cur **graphic.Mesh, verts math32.ArrayF32, r, g, b float32) {
	if *cur != nil {
		if p := (*cur).GetNode().Parent(); p != nil {
			p.GetNode().Remove(*cur)
		}
		*cur = nil
	}
	if len(verts) < 6 || w.scene == nil {
		return
	}
	geom := geometry.NewGeometry()
	geom.AddVBO(gls.NewVBO(verts).AddAttrib(gls.VertexPosition))
	geom.AddGroup(gls.LINES, 0, len(verts)/3)
	mat := material.NewStandard(&math32.Color{R: r, G: g, B: b})
	mat.SetWireframe(true)
	mesh := graphic.NewMesh(geom, mat)
	w.scene.Add(mesh)
	*cur = mesh
}

func (w *World) drawPhysicsDebug() {
	if !w.physDebug {
		w.replaceDebugMesh(&w.physDebugMesh, nil, 0, 0, 0)
		w.replaceDebugMesh(&w.physHitDebugMesh, nil, 0, 0, 0)
		return
	}
	col := math32.NewArrayF32(0, 0)
	hit := math32.NewArrayF32(0, 0)
	w.refreshWorldMatrices()
	for _, e := range w.ents {
		if e == nil || e.node == nil || (e.bodyType == 0 && !e.hitbox) {
			continue
		}
		p := worldPos(e.node.GetNode())
		hx, hy, hz := e.boxX, e.boxY, e.boxZ
		if hx < 0.05 {
			hx = e.radius
		}
		if hy < 0.05 {
			hy = e.radius
		}
		if hz < 0.05 {
			hz = e.radius
		}
		if hx < 0.05 {
			hx = 0.5
		}
		if hy < 0.05 {
			hy = 0.5
		}
		if hz < 0.05 {
			hz = 0.5
		}
		if e.hitbox || e.collKind == 5 {
			appendBoxEdges(&hit, p, hx, hy, hz)
		} else {
			appendBoxEdges(&col, p, hx, hy, hz)
		}
	}
	w.replaceDebugMesh(&w.physDebugMesh, col, 0.2, 0.85, 1)
	w.replaceDebugMesh(&w.physHitDebugMesh, hit, 0.2, 1, 0.25)
}

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
