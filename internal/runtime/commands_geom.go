package runtime

import (
	"math"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

func clampSegs(n, lo, def int) int {
	if n < lo {
		return def
	}
	return n
}

func (w *World) knownEnt(id int) bool {
	return id != 0 && w.ents[id] != nil
}

func (w *World) createCubeMesh(a []value.Value) int {
	size := 2.0
	parent := 0
	switch len(a) {
	case 0:
	case 1:
		if w.knownEnt(argI(a, 0, 0)) {
			parent = argI(a, 0, 0)
		} else {
			size = argN(a, 0, 2)
		}
	default:
		size = argN(a, 0, 2)
		parent = argI(a, 1, 0)
	}
	if size <= 0 {
		size = 2
	}
	return w.meshEnt(geometry.NewCube(float32(size)), parent)
}

func (w *World) createBoxMesh(a []value.Value) int {
	ww, hh, dd := argN(a, 0, 2), argN(a, 1, 2), argN(a, 2, 2)
	if ww <= 0 {
		ww = 2
	}
	if hh <= 0 {
		hh = 2
	}
	if dd <= 0 {
		dd = 2
	}
	parent := 0
	sw, sh, sd := 1, 1, 1
	switch {
	case len(a) >= 6:
		sw = clampSegs(argI(a, 3, 1), 1, 1)
		sh = clampSegs(argI(a, 4, 1), 1, 1)
		sd = clampSegs(argI(a, 5, 1), 1, 1)
		parent = argI(a, 6, 0)
	case len(a) == 4:
		parent = argI(a, 3, 0)
	}
	return w.meshEnt(geometry.NewSegmentedBox(float32(ww), float32(hh), float32(dd), sw, sh, sd), parent)
}

func parseSphereArgs(a []value.Value, known func(int) bool) (radius float64, segs, parent int) {
	radius, segs, parent = 1, 8, 0
	switch {
	case len(a) >= 3:
		radius = argN(a, 0, 1)
		segs = argI(a, 1, 8)
		parent = argI(a, 2, 0)
	case len(a) == 2:
		segs = argI(a, 0, 8)
		parent = argI(a, 1, 0)
	case len(a) == 1:
		// Classic CreateSphere(segments). A lone id must never be a parent:
		// Platform 64 coins are CreateSphere(8) after entity 8 already exists.
		segs = argI(a, 0, 8)
	}
	if radius <= 0 {
		radius = 1
	}
	segs = clampSegs(segs, 4, 8)
	return
}

func parseCylinderArgs(a []value.Value, known func(int) bool) (radius, height float64, segs int, caps bool, parent int) {
	radius, height, segs, caps, parent = 1, 2, 8, true, 0
	switch {
	case len(a) >= 5:
		radius, height = argN(a, 0, 1), argN(a, 1, 2)
		segs = argI(a, 2, 8)
		caps = argI(a, 3, 1) != 0
		parent = argI(a, 4, 0)
	case len(a) == 4:
		radius, height = argN(a, 0, 1), argN(a, 1, 2)
		segs = argI(a, 2, 8)
		if known != nil && known(argI(a, 3, 0)) {
			parent = argI(a, 3, 0)
		} else {
			caps = argI(a, 3, 1) != 0
		}
	case len(a) == 3:
		radius, height, segs = argN(a, 0, 1), argN(a, 1, 2), argI(a, 2, 8)
	case len(a) == 2:
		segs = argI(a, 0, 8)
		parent = argI(a, 1, 0)
	case len(a) == 1:
		// Classic CreateCylinder(segments) / CreateCone(segments).
		segs = argI(a, 0, 8)
	}
	if radius <= 0 {
		radius = 1
	}
	if height <= 0 {
		height = 2
	}
	segs = clampSegs(segs, 3, 8)
	return
}

func (w *World) createSphereMesh(a []value.Value) int {
	r, segs, parent := parseSphereArgs(a, w.knownEnt)
	return w.meshEnt(geometry.NewSphere(r, segs*2, segs), parent)
}

func (w *World) createCylinderMesh(a []value.Value) int {
	r, h, segs, caps, parent := parseCylinderArgs(a, w.knownEnt)
	return w.meshEnt(geometry.NewCylinder(r, h, segs, 1, caps, caps), parent)
}

func (w *World) createConeMesh(a []value.Value) int {
	r, h, segs, _, parent := parseCylinderArgs(a, w.knownEnt)
	return w.meshEnt(geometry.NewCone(r, h, segs, 1, true), parent)
}

func (w *World) createPlaneMesh(a []value.Value) int {
	pw, ph := 20.0, 20.0
	parent := 0
	switch {
	case len(a) >= 3:
		pw, ph, parent = argN(a, 0, 20), argN(a, 1, 20), argI(a, 2, 0)
	case len(a) == 2:
		pw, ph = argN(a, 0, 20), argN(a, 1, 20)
	case len(a) == 1:
		if w.knownEnt(argI(a, 0, 0)) {
			parent = argI(a, 0, 0)
		} else {
			pw = argN(a, 0, 20)
			ph = pw
		}
	}
	if pw <= 0 {
		pw = 20
	}
	if ph <= 0 {
		ph = 20
	}
	id := w.meshEnt(geometry.NewPlane(float32(pw), float32(ph)), parent)
	if e := w.ents[id]; e != nil {
		e.node.GetNode().SetRotation(-math32.Pi/2, 0, 0)
		e.pitch = -90
	}
	return id
}

func (w *World) createTorusMesh(a []value.Value) int {
	major, minor := 1.0, 0.35
	radial, tubular, parent := 12, 24, 0
	switch {
	case len(a) >= 5:
		major, minor = argN(a, 0, 1), argN(a, 1, 0.35)
		radial, tubular = clampSegs(argI(a, 2, 12), 3, 12), clampSegs(argI(a, 3, 24), 3, 24)
		parent = argI(a, 4, 0)
	case len(a) == 4:
		major, minor = argN(a, 0, 1), argN(a, 1, 0.35)
		radial, tubular = clampSegs(argI(a, 2, 12), 3, 12), clampSegs(argI(a, 3, 24), 3, 24)
	case len(a) == 3:
		major, minor = argN(a, 0, 1), argN(a, 1, 0.35)
		if w.knownEnt(argI(a, 2, 0)) {
			parent = argI(a, 2, 0)
		} else {
			radial = clampSegs(argI(a, 2, 12), 3, 12)
		}
	case len(a) == 2:
		if w.knownEnt(argI(a, 1, 0)) && argN(a, 0, 0) > 0 && argN(a, 0, 0) < 8 {
			major = argN(a, 0, 1)
			parent = argI(a, 1, 0)
		} else if w.knownEnt(argI(a, 0, 0)) {
			parent = argI(a, 0, 0)
		} else {
			major, minor = argN(a, 0, 1), argN(a, 1, 0.35)
		}
	case len(a) == 1:
		if w.knownEnt(argI(a, 0, 0)) {
			parent = argI(a, 0, 0)
		} else {
			major = argN(a, 0, 1)
		}
	}
	if major <= 0 {
		major = 1
	}
	if minor <= 0 {
		minor = 0.35
	}
	return w.meshEnt(geometry.NewTorus(major, minor, radial, tubular, math.Pi*2), parent)
}

func (w *World) createCapsuleMesh(a []value.Value) int {
	r, h := 0.4, 1.2
	segs, parent := 10, 0
	switch {
	case len(a) >= 4:
		r, h, segs, parent = argN(a, 0, 0.4), argN(a, 1, 1.2), argI(a, 2, 10), argI(a, 3, 0)
	case len(a) == 3:
		r, h, segs = argN(a, 0, 0.4), argN(a, 1, 1.2), argI(a, 2, 10)
	case len(a) == 2:
		r, h = argN(a, 0, 0.4), argN(a, 1, 1.2)
	case len(a) == 1:
		if w.knownEnt(argI(a, 0, 0)) {
			parent = argI(a, 0, 0)
		} else {
			r = argN(a, 0, 0.4)
		}
	}
	if r <= 0 {
		r = 0.4
	}
	if h < 0 {
		h = 0
	}
	segs = clampSegs(segs, 6, 10)
	return w.meshEnt(newCapsuleGeom(r, h, segs), parent)
}

func (w *World) createDiskMesh(a []value.Value) int {
	r := argN(a, 0, 1)
	segs, parent := 24, 0
	switch {
	case len(a) >= 3:
		segs, parent = argI(a, 1, 24), argI(a, 2, 0)
	case len(a) == 2:
		if w.knownEnt(argI(a, 1, 0)) {
			parent = argI(a, 1, 0)
		} else {
			segs = argI(a, 1, 24)
		}
	case len(a) == 1:
		r = argN(a, 0, 1)
	case len(a) == 0:
		r = 1
	}
	if r <= 0 {
		r = 1
	}
	segs = clampSegs(segs, 3, 24)
	id := w.meshEnt(geometry.NewDisk(r, segs), parent)
	if e := w.ents[id]; e != nil {
		e.node.GetNode().SetRotation(-math32.Pi/2, 0, 0)
		e.pitch = -90
	}
	return id
}

func (w *World) createPyramidMesh(a []value.Value) int {
	size := 2.0
	parent := 0
	switch {
	case len(a) >= 2:
		size, parent = argN(a, 0, 2), argI(a, 1, 0)
	case len(a) == 1:
		size = argN(a, 0, 2)
	}
	if size <= 0 {
		size = 2
	}
	return w.meshEnt(geometry.NewCone(size*0.5, size, 4, 1, true), parent)
}

func (w *World) createWedgeMesh(a []value.Value) int {
	ww, hh, dd := argN(a, 0, 2), argN(a, 1, 1), argN(a, 2, 2)
	parent := argI(a, 3, 0)
	if ww <= 0 {
		ww = 2
	}
	if hh <= 0 {
		hh = 1
	}
	if dd <= 0 {
		dd = 2
	}
	return w.meshEnt(newWedgeGeom(float32(ww), float32(hh), float32(dd)), parent)
}

func (w *World) createTubeMesh(a []value.Value) int {
	r, h := argN(a, 0, 1), argN(a, 1, 2)
	segs, parent := 16, 0
	switch {
	case len(a) >= 4:
		segs, parent = argI(a, 2, 16), argI(a, 3, 0)
	case len(a) == 3:
		if w.knownEnt(argI(a, 2, 0)) {
			parent = argI(a, 2, 0)
		} else {
			segs = argI(a, 2, 16)
		}
	case len(a) == 1:
		if w.knownEnt(argI(a, 0, 0)) {
			r, h, parent = 1, 2, argI(a, 0, 0)
		}
	case len(a) == 0:
		r, h = 1, 2
	}
	if r <= 0 {
		r = 1
	}
	if h <= 0 {
		h = 2
	}
	segs = clampSegs(segs, 3, 16)
	path := []math32.Vector3{{0, float32(-h / 2), 0}, {0, float32(h / 2), 0}}
	return w.meshEnt(geometry.NewTube(path, float32(r), segs, false), parent)
}

func (w *World) createQuadMesh(a []value.Value) int {
	size := 2.0
	parent := 0
	switch {
	case len(a) >= 2:
		size, parent = argN(a, 0, 2), argI(a, 1, 0)
	case len(a) == 1:
		if w.knownEnt(argI(a, 0, 0)) {
			parent = argI(a, 0, 0)
		} else {
			size = argN(a, 0, 2)
		}
	}
	if size <= 0 {
		size = 2
	}
	id := w.meshEnt(geometry.NewPlane(float32(size), float32(size)), parent)
	if e := w.ents[id]; e != nil {
		e.node.GetNode().SetRotation(-math32.Pi/2, 0, 0)
		e.pitch = -90
	}
	return id
}

func newCapsuleGeom(radius, cylHeight float64, segs int) *geometry.Geometry {
	if segs < 6 {
		segs = 8
	}
	rings := segs / 2
	if rings < 4 {
		rings = 4
	}
	g := geometry.NewGeometry()
	pos := math32.NewArrayF32(0, 16)
	nor := math32.NewArrayF32(0, 16)
	uvs := math32.NewArrayF32(0, 16)
	idx := math32.NewArrayU32(0, 16)
	r := float32(radius)
	half := float32(cylHeight / 2)
	latCount := rings*2 + 1
	for iy := 0; iy <= latCount; iy++ {
		var y, ny float32
		var v float32
		if iy <= rings {
			t := float32(iy) / float32(rings)
			ang := -math32.Pi/2 + t*math32.Pi/2
			ny = math32.Sin(ang)
			xr := math32.Cos(ang)
			y = -half + ny*r
			_ = xr
			v = t * 0.5
			for ix := 0; ix <= segs; ix++ {
				phi := float32(ix) / float32(segs) * math32.Pi * 2
				cx, cz := math32.Cos(phi)*xr, math32.Sin(phi)*xr
				pos.Append(cx*r, y, cz*r)
				nor.Append(cx, ny, cz)
				uvs.Append(float32(ix)/float32(segs), v)
			}
		} else {
			t := float32(iy-rings) / float32(rings)
			ang := t * math32.Pi / 2
			ny = math32.Sin(ang)
			xr := math32.Cos(ang)
			y = half + ny*r
			v = 0.5 + t*0.5
			for ix := 0; ix <= segs; ix++ {
				phi := float32(ix) / float32(segs) * math32.Pi * 2
				cx, cz := math32.Cos(phi)*xr, math32.Sin(phi)*xr
				pos.Append(cx*r, y, cz*r)
				nor.Append(cx, ny, cz)
				uvs.Append(float32(ix)/float32(segs), v)
			}
		}
	}
	stride := segs + 1
	for iy := 0; iy < latCount; iy++ {
		for ix := 0; ix < segs; ix++ {
			a := uint32(iy*stride + ix)
			b := a + 1
			c := a + uint32(stride)
			d := c + 1
			idx.Append(a, c, b, b, c, d)
		}
	}
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

func newWedgeGeom(w, h, d float32) *geometry.Geometry {
	hw, hh, hd := w/2, h/2, d/2
	g := geometry.NewGeometry()
	// ramp: high at -Z, low at +Z
	verts := []float32{
		-hw, -hh, -hd, hw, -hh, -hd, hw, hh, -hd, -hw, hh, -hd, // back (tall)
		-hw, -hh, hd, hw, -hh, hd, // front (low)
	}
	// triangles: back, bottom, left, right, ramp
	idx := math32.NewArrayU32(0, 24)
	idx.Append(
		0, 1, 2, 0, 2, 3, // back
		0, 4, 5, 0, 5, 1, // bottom
		0, 3, 4, // left
		1, 5, 2, // right
		3, 2, 5, 3, 5, 4, // ramp
	)
	pos := math32.NewArrayF32(0, 18)
	pos.Append(verts...)
	nor := math32.NewArrayF32(len(verts), len(verts))
	uvs := math32.NewArrayF32(len(verts)/3*2, len(verts)/3*2)
	nor = geometry.CalculateNormals(idx, pos, nor)
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}
