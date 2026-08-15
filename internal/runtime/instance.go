package runtime

import (
	"math"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type instXform struct {
	x, y, z                float32
	pitch, yaw, roll       float32
	sx, sy, sz             float32
}

type instancedMesh struct {
	src                *geometry.Geometry
	tris               [][3]math32.Vector3
	count              int
	xforms             []instXform
	batch              int // entity id of merged mesh (CPU fallback / shadows)
	dirty              bool
	gpuOK              bool
	vao, vbo, ebo      uint32
	instVBO            uint32
	indexN, vertN      int32
}

func (w *World) createInstanced(srcID, count int) (int, error) {
	if count < 1 {
		count = 1
	}
	if count > 8000 {
		count = 8000
	}
	var srcGeom *geometry.Geometry
	if e := w.ents[srcID]; e != nil && e.mesh != nil {
		srcGeom = e.mesh.GetGeometry()
	}
	if srcGeom == nil {
		srcGeom = geometry.NewCube(1)
	}
	var ident math32.Matrix4
	ident.Identity()
	tris := geomWorldTris(srcGeom, ident)
	xf := make([]instXform, count)
	for i := range xf {
		xf[i] = instXform{sx: 1, sy: 1, sz: 1}
	}
	pivot := w.addEntity(&Entity{node: graphic.NewMesh(geometry.NewCube(0.01), w.newMat())}, 0)
	if e := w.ents[pivot]; e != nil {
		e.node.GetNode().SetVisible(false)
		e.name = "instances"
	}
	im := &instancedMesh{src: srcGeom, tris: tris, count: count, xforms: xf, dirty: true}
	if w.instances == nil {
		w.instances = map[int]*instancedMesh{}
	}
	w.instances[pivot] = im
	w.rebuildInstanceBatch(pivot)
	return pivot, nil
}

func instanceMatrix(xf instXform) math32.Matrix4 {
	var m math32.Matrix4
	sx, sy, sz := xf.sx, xf.sy, xf.sz
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	if sz == 0 {
		sz = 1
	}
	pr := xf.pitch * math32.Pi / 180
	yr := -xf.yaw * math32.Pi / 180
	rr := -xf.roll * math32.Pi / 180
	cx, sxn := float32(math.Cos(float64(pr))), float32(math.Sin(float64(pr)))
	cy, syn := float32(math.Cos(float64(yr))), float32(math.Sin(float64(yr)))
	cz, szn := float32(math.Cos(float64(rr))), float32(math.Sin(float64(rr)))
	// scale * Rx * Ry * Rz, then translate (Blitz-ish applyRot order)
	var s, rx, ry, rz math32.Matrix4
	s.Scale(&math32.Vector3{sx, sy, sz})
	rx.MakeRotationX(pr)
	ry.MakeRotationY(yr)
	rz.MakeRotationZ(rr)
	m.MultiplyMatrices(&rx, &s)
	m.Multiply(&ry)
	m.Multiply(&rz)
	gx, gy, gz := toG3N(xf.x, xf.y, xf.z)
	m.SetPosition(&math32.Vector3{gx, gy, gz})
	_ = cx
	_ = sxn
	_ = cy
	_ = syn
	_ = cz
	_ = szn
	return m
}

func (w *World) rebuildInstanceBatch(id int) {
	im := w.instances[id]
	if im == nil || len(im.tris) == 0 {
		return
	}
	if w.glmod.gpuInst && w.glmod.caps.instance {
		w.uploadInstanceGPU(im)
		if im.gpuOK {
			if e := w.ents[im.batch]; e != nil && e.node != nil {
				e.node.GetNode().SetVisible(false)
			}
			im.dirty = false
			return
		}
	}
	if e := w.ents[im.batch]; e != nil && e.node != nil {
		e.node.GetNode().SetVisible(true)
	}
	pos := math32.NewArrayF32(0, len(im.tris)*9*im.count)
	nor := math32.NewArrayF32(0, len(im.tris)*9*im.count)
	uvs := math32.NewArrayF32(0, len(im.tris)*6*im.count)
	idx := math32.NewArrayU32(0, len(im.tris)*3*im.count)
	vi := uint32(0)
	for i := 0; i < im.count && i < len(im.xforms); i++ {
		m := instanceMatrix(im.xforms[i])
		for _, t := range im.tris {
			a, b, c := t[0], t[1], t[2]
			a.ApplyMatrix4(&m)
			b.ApplyMatrix4(&m)
			c.ApplyMatrix4(&m)
			e1 := b.Clone().Sub(&a)
			e2 := c.Clone().Sub(&a)
			n := e1.Cross(e2)
			if n.LengthSq() > 1e-10 {
				n.Normalize()
			} else {
				n = &math32.Vector3{0, 1, 0}
			}
			pos.Append(a.X, a.Y, a.Z, b.X, b.Y, b.Z, c.X, c.Y, c.Z)
			nor.Append(n.X, n.Y, n.Z, n.X, n.Y, n.Z, n.X, n.Y, n.Z)
			uvs.Append(0, 0, 1, 0, 0, 1)
			idx.Append(vi, vi+1, vi+2)
			vi += 3
		}
	}
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	if im.batch != 0 {
		w.freeEntityID(im.batch)
		im.batch = 0
	}
	bid := w.meshEnt(g, 0)
	if e := w.ents[bid]; e != nil {
		e.name = "instanced"
	}
	im.batch = bid
	im.dirty = false
}

func (w *World) tickInstances() {
	for id, im := range w.instances {
		if im != nil && im.dirty {
			w.rebuildInstanceBatch(id)
		}
	}
}

func (w *World) instanceCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createinstancedmesh": need(func(a []value.Value) (value.Value, error) {
			id, err := w.createInstanced(argI(a, 0, 0), argI(a, 1, 1))
			return value.Num(float64(id)), err
		}),
		"instancecount": n(func(a []value.Value) (value.Value, error) {
			im := w.instances[argI(a, 0, 0)]
			if im == nil {
				return value.Num(0), nil
			}
			if len(a) >= 2 {
				c := argI(a, 1, im.count)
				if c < 0 {
					c = 0
				}
				if c > 8000 {
					c = 8000
				}
				for len(im.xforms) < c {
					im.xforms = append(im.xforms, instXform{sx: 1, sy: 1, sz: 1})
				}
				im.count = c
				im.dirty = true
			}
			return value.Num(float64(im.count)), nil
		}),
		"setinstancedata":      n(func(a []value.Value) (value.Value, error) { return w.setInstanceXform(a, z) }),
		"setinstancetransform": n(func(a []value.Value) (value.Value, error) { return w.setInstanceXform(a, z) }),
		"batchinstances": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if w.instances[id] != nil {
				w.rebuildInstanceBatch(id)
				return value.Num(float64(w.instances[id].batch)), nil
			}
			return value.Num(0), nil
		}),
		"instanceentity": n(func(a []value.Value) (value.Value, error) {
			im := w.instances[argI(a, 0, 0)]
			if im == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(im.batch)), nil
		}),
	}
}

func (w *World) setInstanceXform(a []value.Value, z func() (value.Value, error)) (value.Value, error) {
	im := w.instances[argI(a, 0, 0)]
	i := argI(a, 1, 0)
	if im == nil || i < 0 || i >= im.count {
		return z()
	}
	xf := &im.xforms[i]
	xf.x = float32(argN(a, 2, 0))
	xf.y = float32(argN(a, 3, 0))
	xf.z = float32(argN(a, 4, 0))
	if len(a) > 5 {
		xf.pitch = float32(argN(a, 5, 0))
		xf.yaw = float32(argN(a, 6, 0))
		xf.roll = float32(argN(a, 7, 0))
	}
	if len(a) > 8 {
		xf.sx = float32(argN(a, 8, 1))
		xf.sy = float32(argN(a, 9, 1))
		xf.sz = float32(argN(a, 10, 1))
	}
	im.dirty = true
	return z()
}
