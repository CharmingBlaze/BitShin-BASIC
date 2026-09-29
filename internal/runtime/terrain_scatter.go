package runtime

import (
	"math"
	"math/rand"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

// scatterLayer is one repeated mesh on a terrain (grass cards, trees, or props).
// Copies follow chunk load and unload. They are not heightfield, collision, or nav.
type scatterLayer struct {
	kind    string
	mesh    int
	density float32
	spacing float32
}

type scatterPt struct {
	x, y, z float32
	yaw     float32
	scale   float32
}

// lodStep is the CPU stand-in for a screen-space triangle target.
// detailPx 0 keeps the old thresholds. Smaller pixels keep the fine mesh farther out.
func (t *terrain) lodStep(dist2 float32) int {
	if t == nil || (t.lod <= 0 && t.detailPx <= 0) {
		return 1
	}
	s := lodScale(t)
	step := 1
	if dist2 > 2*s*s {
		step = 2
	}
	far := float32(8)
	if t.lod > 1 {
		far = 4
	}
	if dist2 > far*s*s {
		step = 4
	}
	return step
}

// lodBuild is the step plus a geomorph weight toward the next coarser grid.
// The weight stays 0 until the last third of the band, then rises to 1.
func (t *terrain) lodBuild(dist2 float32) (int, float32) {
	step := t.lodStep(dist2)
	if t == nil || t.morphOff || step >= 4 || (t.lod <= 0 && t.detailPx <= 0) {
		return step, 0
	}
	s := lodScale(t)
	prev, limit := float32(0), 2*s*s
	if step >= 2 {
		prev = 2 * s * s
		far := float32(8)
		if t.lod > 1 {
			far = 4
		}
		limit = far * s * s
	}
	span := limit - prev
	if span <= 0 {
		return step, 0
	}
	u := (dist2 - prev) / span
	if u < 0.65 {
		return step, 0
	}
	m := (u - 0.65) / 0.35
	if m > 1 {
		m = 1
	}
	return step, m
}

func lodScale(t *terrain) float32 {
	if t == nil || t.detailPx <= 0 {
		return 1
	}
	s := 8 / t.detailPx
	if s < 0.2 {
		s = 0.2
	}
	if s > 4 {
		s = 4
	}
	return s
}

func lodCode(step int, morph float32) int {
	code := step
	if morph >= 0.5 {
		code += 8
	}
	return code
}

func scatterBudget(kind string, density, size float32, step int) (n int, spacing float32) {
	if density <= 0 || size <= 0 {
		return 0, 1
	}
	cell := float32(1.7)
	cap := 48
	spacing = 1.15
	switch kind {
	case "tree":
		cell, cap, spacing = 8, 8, 6
	case "prop":
		cell, cap, spacing = 4.5, 12, 2.8
	case "any":
		cell, cap, spacing = 5, 12, 3
	}
	n = int(density * (size / cell) * (size / cell) * 0.22)
	if step >= 2 {
		n = n / 3
	}
	if n > cap {
		n = cap
	}
	if n < 1 && step < 2 {
		n = 1
	}
	return n, spacing
}

// scatterAccept is the slope and height filter. flat is the up-component of the
// ground normal (1 is level). Grass and trees want that flat ground, above the
// shore and below snow. Props sit on anything that is not a cliff or a snowfield.
func scatterAccept(kind string, flat, h, water, snow float32, snowOn bool) bool {
	switch kind {
	case "grass", "tree":
		if flat < 0.72 {
			return false
		}
		if h < water+0.35 {
			return false
		}
		if snowOn && h > snow-0.4 {
			return false
		}
		return true
	case "prop":
		if flat < 0.38 {
			return false
		}
		if h < water+0.1 {
			return false
		}
		if snowOn && h > snow {
			return false
		}
		return true
	default:
		if flat < 0.30 {
			return false
		}
		if h < water {
			return false
		}
		return true
	}
}

func scatterFlat(t *terrain, x, z float32) float32 {
	if t == nil {
		return 1
	}
	const eps = float32(0.45)
	h := t.hf.atWorld
	hl, hr := h(x-eps, z), h(x+eps, z)
	hd, hu := h(x, z-eps), h(x, z+eps)
	nx := hl - hr
	ny := 2 * eps
	nz := -(hd - hu)
	l := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if l < 1e-5 {
		return 1
	}
	return ny / l
}

func (t *terrain) setScatterLayer(kind string, mesh int, density, spacing float32) {
	if t == nil {
		return
	}
	keep := make([]scatterLayer, 0, len(t.scatters))
	for _, s := range t.scatters {
		if s.kind == kind && s.mesh == mesh {
			continue
		}
		keep = append(keep, s)
	}
	if density > 0 {
		if spacing <= 0 {
			_, spacing = scatterBudget(kind, 1, 32, 1)
		}
		keep = append(keep, scatterLayer{kind: kind, mesh: mesh, density: density, spacing: spacing})
	}
	t.scatters = keep
}

func (w *World) freeScatter(t *terrain, key chunkKey) {
	if t == nil || t.scatterIDs == nil {
		return
	}
	for _, id := range t.scatterIDs[key] {
		delete(w.instances, id)
		w.freeEntityID(id)
	}
	delete(t.scatterIDs, key)
}

func (w *World) refreshScatter(t *terrain) {
	if t == nil {
		return
	}
	cw := t.chunkWorld()
	for k := range t.ents {
		x0 := float32(k.X) * cw
		z0 := float32(k.Z) * cw
		w.placeScatter(t, k, x0, z0, cw, w.scatterStep(t, k))
	}
}

func (w *World) scatterStep(t *terrain, key chunkKey) int {
	ox, oz := float32(0), float32(0)
	if w.cam != nil {
		p := worldPos(w.cam.GetNode())
		ox, _, oz = fromG3N(p.X, p.Y, p.Z)
	}
	c := chunkOf(ox, oz, t.chunkWorld())
	dx := float32(key.X - c.X)
	dz := float32(key.Z - c.Z)
	return t.lodStep(dx*dx + dz*dz)
}

func (w *World) placeScatter(t *terrain, key chunkKey, x0, z0, size float32, step int) {
	if t == nil {
		return
	}
	w.freeScatter(t, key)
	if len(t.scatters) == 0 {
		return
	}
	if t.scatterIDs == nil {
		t.scatterIDs = map[chunkKey][]int{}
	}
	var ids []int
	for i, layer := range t.scatters {
		if (layer.kind == "tree" || layer.kind == "prop") && layer.mesh == 0 {
			continue
		}
		pts := w.scatterPoints(t, layer, key, i, x0, z0, size, step)
		if len(pts) == 0 {
			continue
		}
		far := step >= 2
		if layer.kind == "grass" || far || layer.mesh == 0 {
			id := w.addCardScatter(t, layer.kind, pts, !far && layer.kind == "grass")
			if id != 0 {
				ids = append(ids, id)
			}
			continue
		}
		id, err := w.createInstanced(layer.mesh, len(pts))
		if err != nil || id == 0 {
			continue
		}
		im := w.instances[id]
		if im != nil {
			for pi, p := range pts {
				im.xforms[pi] = instXform{x: p.x, y: p.y, z: p.z, yaw: p.yaw, sx: p.scale, sy: p.scale, sz: p.scale}
			}
			im.dirty = true
			w.rebuildInstanceBatch(id)
		}
		if e := w.ents[id]; e != nil {
			e.name = "scatter"
		}
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		t.scatterIDs[key] = ids
	}
}

func (w *World) scatterPoints(t *terrain, layer scatterLayer, key chunkKey, salt int, x0, z0, size float32, step int) []scatterPt {
	n, spacing := scatterBudget(layer.kind, layer.density, size, step)
	if layer.spacing > 0 {
		spacing = layer.spacing
	}
	if n <= 0 {
		return nil
	}
	seed := int64(key.X)*73856093 ^ int64(key.Z)*19349663 ^ int64(salt+1)*83492791 ^ int64(layer.mesh)*17
	rng := rand.New(rand.NewSource(seed))
	pts := make([]scatterPt, 0, n)
	space2 := spacing * spacing
	tries := n * 10
	if tries > 400 {
		tries = 400
	}
	for i := 0; i < tries && len(pts) < n; i++ {
		x := x0 + rng.Float32()*size
		z := z0 + rng.Float32()*size
		h := w.sampleHeight(&t.hf, x, z)
		if !scatterAccept(layer.kind, scatterFlat(t, x, z), h, t.waterY, t.snowH, t.snowOn) {
			continue
		}
		ok := true
		for _, p := range pts {
			dx, dz := p.x-x, p.z-z
			if dx*dx+dz*dz < space2 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		pts = append(pts, scatterPt{
			x: x, y: h, z: z,
			yaw:   rng.Float32() * 360,
			scale: 0.85 + rng.Float32()*0.3,
		})
	}
	if step >= 2 && w.cam != nil && len(pts) > 0 {
		p := worldPos(w.cam.GetNode())
		cx, _, cz := fromG3N(p.X, p.Y, p.Z)
		for i := range pts {
			dx := cx - pts[i].x
			dz := cz - pts[i].z
			pts[i].yaw = float32(math.Atan2(float64(dx), float64(dz)) * 180 / math.Pi)
		}
	}
	return pts
}

func (w *World) addCardScatter(t *terrain, kind string, pts []scatterPt, crossed bool) int {
	width, height := float32(0.55), float32(0.9)
	col := &math32.Color{0.22, 0.48, 0.16}
	switch kind {
	case "tree":
		width, height = 1.7, 3.4
		col = &math32.Color{0.18, 0.32, 0.14}
	case "prop":
		width, height = 1.1, 1.2
		col = &math32.Color{0.42, 0.40, 0.36}
	}
	g := cardGeometry(pts, crossed, width, height)
	if g == nil {
		return 0
	}
	mat := w.newMat()
	mat.SetColor(col)
	mat.SetSide(material.SideDouble)
	mesh := graphic.NewMesh(g, mat)
	parent := 0
	if t != nil {
		parent = t.root
	}
	id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat, name: "foliage", kind: "scatter"}, parent)
	return id
}

func cardGeometry(pts []scatterPt, crossed bool, width, height float32) *geometry.Geometry {
	if len(pts) == 0 {
		return nil
	}
	pos := math32.NewArrayF32(0, 0)
	nor := math32.NewArrayF32(0, 0)
	uvs := math32.NewArrayF32(0, 0)
	idx := math32.NewArrayU32(0, 0)
	nverts := 0
	addQuad := func(p scatterPt, yawOff, w, h float32) {
		yaw := (p.yaw + yawOff) * math32.Pi / 180
		c := math32.Cos(yaw)
		s := math32.Sin(yaw)
		hw := w * p.scale * 0.5
		hh := h * p.scale
		local := [4][3]float32{{-hw, 0, 0}, {hw, 0, 0}, {hw, hh, 0}, {-hw, hh, 0}}
		base := uint32(nverts)
		nx, ny, nz := toG3N(s, 0, c)
		for i, v := range local {
			rx := v[0]*c + v[2]*s
			rz := -v[0]*s + v[2]*c
			gx, gy, gz := toG3N(p.x+rx, p.y+v[1], p.z+rz)
			pos.Append(gx, gy, gz)
			nor.Append(nx, ny, nz)
			switch i {
			case 0:
				uvs.Append(0, 1)
			case 1:
				uvs.Append(1, 1)
			case 2:
				uvs.Append(1, 0)
			default:
				uvs.Append(0, 0)
			}
			nverts++
		}
		idx.Append(base, base+1, base+2, base, base+2, base+3)
	}
	for _, p := range pts {
		addQuad(p, 0, width, height)
		if crossed {
			addQuad(p, 90, width, height)
		}
	}
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

func (w *World) scatterCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	set := func(kind string, mesh bool, defDensity float64) func(a []value.Value) (value.Value, error) {
		return func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			meshID := 0
			if mesh {
				meshID = argI(a, off, 0)
				off++
			}
			dens := float32(argN(a, off, defDensity))
			t.setScatterLayer(kind, meshID, dens, 0)
			w.refreshScatter(t)
			return z()
		}
	}
	return map[string]cmd{
		"terrainfoliage": n(set("grass", false, 1)),
		"terraintrees":   n(set("tree", true, 1)),
		"terrainprops":   n(set("prop", true, 1)),
		"terrainscatter": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			meshID := argI(a, off, 0)
			kind := argS(a, off+1)
			if kind == "" {
				kind = "any"
			}
			dens := float32(argN(a, off+2, 1))
			t.setScatterLayer(kind, meshID, dens, 0)
			w.refreshScatter(t)
			return z()
		}),
		"setterrainblendmap": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			t.blendTex = argI(a, off, 0)
			return z()
		}),
		"setterraindetail": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			px := float32(argN(a, off, 8))
			if px < 0 {
				px = 0
			}
			t.detailPx = px
			w.dirtyTerrain(t)
			return z()
		}),
	}
}
