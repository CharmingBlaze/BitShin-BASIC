package runtime

import (
	"hash/fnv"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func (w *World) personCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), _ func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	_ = z
	return map[string]cmd{
		"createperson": n(func(a []value.Value) (value.Value, error) {
			name := argS(a, 0)
			if name == "" {
				name = "Person"
			}
			h, shoulder, cr, cg, cb := personLook(name)
			id := w.meshEnt(personGeom(h, shoulder), 0)
			e := w.ents[id]
			e.name = name
			if e.node != nil {
				e.node.GetNode().SetName(name)
			}
			x := float32(argN(a, 1, 0))
			y := float32(argN(a, 2, 0))
			z := float32(argN(a, 3, 0))
			if e.node != nil {
				gx, gy, gz := toG3N(x, y+h*0.5, z)
				e.node.GetNode().SetPosition(gx, gy, gz)
			}
			if e.mat != nil {
				e.mat.SetColor(rgb(float64(cr), float64(cg), float64(cb)))
			}
			e.kind = "person"
			return value.Num(float64(id)), nil
		}),
	}
}

func personLook(name string) (height, shoulder float32, r, g, b int) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	v := h.Sum32()
	height = 1.45 + float32(v%40)/100
	shoulder = 0.28 + float32((v>>8)%20)/200
	r = 40 + int(v%180)
	g = 40 + int((v>>8)%180)
	b = 40 + int((v>>16)%180)
	return height, shoulder, r, g, b
}

func personGeom(height, shoulder float32) *geometry.Geometry {
	// Torso, head, arms, and two legs as one indexed mesh. Proportions come from the name.
	legH := height * 0.45
	bodyH := height * 0.38
	headR := height * 0.08
	armW := shoulder * 0.28
	armY := legH + bodyH*0.2
	armH := bodyH * 0.72
	boxes := [][6]float32{
		{-shoulder, legH, -0.08, shoulder, legH + bodyH, 0.08},
		{-headR, legH + bodyH, -headR, headR, legH + bodyH + headR*2, headR},
		{-shoulder * 0.7, 0, -0.06, -shoulder * 0.15, legH, 0.06},
		{shoulder * 0.15, 0, -0.06, shoulder * 0.7, legH, 0.06},
		{-shoulder - armW, armY, -0.05, -shoulder, armY + armH, 0.05},
		{shoulder, armY, -0.05, shoulder + armW, armY + armH, 0.05},
	}
	var pos []float32
	var idx []uint32
	var base uint32
	for _, b := range boxes {
		p, ix := boxVerts(b[0], b[1], b[2], b[3], b[4], b[5], base)
		pos = append(pos, p...)
		idx = append(idx, ix...)
		base += 8
	}
	g := geometry.NewGeometry()
	ip := math32.NewArrayU32(0, len(idx))
	ip.Append(idx...)
	vp := math32.NewArrayF32(0, len(pos))
	vp.Append(pos...)
	nor := math32.NewArrayF32(len(pos), len(pos))
	nor = geometry.CalculateNormals(ip, vp, nor)
	uv := math32.NewArrayF32(0, len(pos)/3*2)
	for i := 0; i < len(pos)/3; i++ {
		uv.Append(0, 0)
	}
	g.SetIndices(ip)
	g.AddVBO(gls.NewVBO(vp).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uv).AddAttrib(gls.VertexTexcoord))
	return g
}

func boxVerts(x0, y0, z0, x1, y1, z1 float32, base uint32) ([]float32, []uint32) {
	pos := []float32{
		x0, y0, z0, x1, y0, z0, x1, y1, z0, x0, y1, z0,
		x0, y0, z1, x1, y0, z1, x1, y1, z1, x0, y1, z1,
	}
	faces := []uint32{
		0, 1, 2, 0, 2, 3,
		4, 6, 5, 4, 7, 6,
		0, 4, 5, 0, 5, 1,
		1, 5, 6, 1, 6, 2,
		2, 6, 7, 2, 7, 3,
		3, 7, 4, 3, 4, 0,
	}
	for i := range faces {
		faces[i] += base
	}
	return pos, faces
}
