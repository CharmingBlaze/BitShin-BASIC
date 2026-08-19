package runtime

import (
	"time"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type clothSheet struct {
	id, nx, ny     int
	width, height  float32
	wind           float32
	buf            []float32
	geom           *geometry.Geometry
}

func clampClothSeg(n, def int) int {
	if n < 2 {
		return def
	}
	if n > 24 {
		return 24
	}
	return n
}

func (w *World) buildClothGeom(width, height float32, nx, ny int) *geometry.Geometry {
	sx := width / float32(nx-1)
	sy := height / float32(ny-1)
	npos := nx * ny
	pos := math32.NewArrayF32(0, npos*3)
	nor := math32.NewArrayF32(0, npos*3)
	uvs := math32.NewArrayF32(0, npos*2)
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			pos.Append(float32(i)*sx, -float32(j)*sy, 0)
			nor.Append(0, 0, 1)
			uvs.Append(float32(i)/float32(nx-1), float32(j)/float32(ny-1))
		}
	}
	idx := math32.NewArrayU32(0, (nx-1)*(ny-1)*6)
	for j := 0; j < ny-1; j++ {
		for i := 0; i < nx-1; i++ {
			a := uint32(i + j*nx)
			b := a + 1
			c := uint32(i + (j+1)*nx)
			d := c + 1
			idx.Append(a, c, d, a, d, b)
		}
	}
	g := geometry.NewGeometry()
	posVBO := gls.NewVBO(pos).AddAttrib(gls.VertexPosition)
	posVBO.SetUsage(gls.DYNAMIC_DRAW)
	norVBO := gls.NewVBO(nor).AddAttrib(gls.VertexNormal)
	norVBO.SetUsage(gls.DYNAMIC_DRAW)
	g.AddVBO(posVBO)
	g.AddVBO(norVBO)
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	g.SetIndices(idx)
	return g
}

func (w *World) createCloth(width, height float32, nx, ny, pin int) int {
	w.ensurePhys3()
	nx = clampClothSeg(nx, 12)
	ny = clampClothSeg(ny, 12)
	if width < 0.2 {
		width = 0.2
	}
	if height < 0.2 {
		height = 0.2
	}
	if pin == 0 {
		pin = 1
	}
	g := w.buildClothGeom(width, height, nx, ny)
	mat := w.newMat()
	mat.SetSide(material.SideDouble)
	mesh := graphic.NewMesh(g, newLitMat(w, mat))
	mesh.SetCullable(false)
	id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat, cloth: true, bodyType: 1, boxX: width * 0.5, boxY: height * 0.5, boxZ: 0.05}, 0)
	px, py, pz := float32(0), float32(0), float32(0)
	if e := w.ents[id]; e != nil && e.node != nil {
		p := e.node.GetNode().Position()
		px, py, pz = fromG3N(p.X, p.Y, p.Z)
		e.name = "cloth"
		e.kind = "cloth"
	}
	w.phys3.AddCloth(id, px, py, pz, width, height, nx, ny, pin, 0.01, 0.12, 1)
	if w.cloths == nil {
		w.cloths = map[int]*clothSheet{}
	}
	w.cloths[id] = &clothSheet{
		id: id, nx: nx, ny: ny, width: width, height: height, wind: 0.35,
		buf:  make([]float32, nx*ny*3),
		geom: g,
	}
	if w.shadow.primed == nil {
		w.shadow.primed = map[*geometry.Geometry]bool{}
	}
	w.shadow.primed[g] = true
	return id
}

func (w *World) applyClothWind() {
	if w.phys3 == nil || len(w.cloths) == 0 {
		return
	}
	wx, wy, wz := float32(w.wx.windX)*float32(w.wx.windStr), float32(w.wx.windY)*float32(w.wx.windStr), float32(w.wx.windZ)*float32(w.wx.windStr)
	if wb := w.currentWater(); wb != nil && wx == 0 && wz == 0 {
		wx, wz = wb.windX*(1+wb.windStr), wb.windZ*(1+wb.windStr)
	}
	t := float32(w.loopFrames)
	if !w.started.IsZero() {
		t = float32(time.Since(w.started).Seconds()) * 60
	}
	gust := 1 + 0.55*math32.Sin(t*0.11) + 0.28*math32.Sin(t*0.29)
	wx, wy, wz = wx*gust, wy*gust, wz*gust
	wz += 1.15*math32.Sin(t*0.19) + 0.4*math32.Sin(t*0.41)
	wx += 0.22 * math32.Sin(t*0.13)
	start := uint32(w.loopFrames)
	for id, c := range w.cloths {
		if c == nil || c.wind <= 0 {
			continue
		}
		w.phys3.ApplyClothWind(id, wx*c.wind, wy*c.wind, wz*c.wind, start, 1)
	}
}

func (w *World) tickCloth() {
	if w.phys3 == nil || len(w.cloths) == 0 {
		return
	}
	for id, c := range w.cloths {
		if c == nil || c.geom == nil {
			continue
		}
		n := w.phys3.ClothVertices(id, c.buf)
		if n < 3 {
			continue
		}
		i := 0
		c.geom.OperateOnVertices(func(v *math32.Vector3) bool {
			if i >= n {
				return true
			}
			gx, gy, gz := toG3N(c.buf[i*3], c.buf[i*3+1], c.buf[i*3+2])
			v.X, v.Y, v.Z = gx, gy, gz
			i++
			return false
		})
		w.refreshClothNormals(c)
		if e := w.ents[id]; e != nil {
			e.shadowGeomDirty = true
		}
	}
}

func (w *World) refreshClothNormals(c *clothSheet) {
	if c == nil || c.geom == nil {
		return
	}
	nx, ny := c.nx, c.ny
	pos := make([]math32.Vector3, nx*ny)
	k := 0
	c.geom.ReadVertices(func(v math32.Vector3) bool {
		if k < len(pos) {
			pos[k] = v
			k++
		}
		return false
	})
	i := 0
	c.geom.OperateOnVertexNormals(func(n *math32.Vector3) bool {
		x, y := i%nx, i/nx
		i++
		if x >= nx-1 || y >= ny-1 {
			n.X, n.Y, n.Z = 0, 0, 1
			return false
		}
		p := pos[x+y*nx]
		dx := math32.Vector3{X: pos[x+1+y*nx].X - p.X, Y: pos[x+1+y*nx].Y - p.Y, Z: pos[x+1+y*nx].Z - p.Z}
		dy := math32.Vector3{X: pos[x+(y+1)*nx].X - p.X, Y: pos[x+(y+1)*nx].Y - p.Y, Z: pos[x+(y+1)*nx].Z - p.Z}
		cr := math32.Vector3{
			X: dx.Y*dy.Z - dx.Z*dy.Y,
			Y: dx.Z*dy.X - dx.X*dy.Z,
			Z: dx.X*dy.Y - dx.Y*dy.X,
		}
		if cr.LengthSq() < 1e-10 {
			n.X, n.Y, n.Z = 0, 0, 1
			return false
		}
		cr.Normalize()
		n.X, n.Y, n.Z = cr.X, cr.Y, cr.Z
		return false
	})
}

func (w *World) clothCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createcloth": need(func(a []value.Value) (value.Value, error) {
			id := w.createCloth(float32(argN(a, 0, 2)), float32(argN(a, 1, 2)), argI(a, 2, 12), argI(a, 3, 12), argI(a, 4, 1))
			return value.Num(float64(id)), nil
		}),
		"setclothwind": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if c := w.cloths[id]; c != nil {
				c.wind = float32(argN(a, 1, 0.35))
			}
			return z()
		}),
	}
}
