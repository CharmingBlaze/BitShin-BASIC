package runtime

import (
	"fmt"
	"math"
	"sort"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// A large level is thousands of small meshes. G3N pays per mesh, so
// MergeLevel bakes every static mesh under a root into one mesh per material
// per cell. Draws then follow what is on screen, not how many objects the
// file holds. Each cell is frustum culled, and cells past the fog are hidden.

const defaultLevelCell = 64

type levelCell struct {
	ent        int
	cx, cz     float32 // middle, world units
	rad        float32 // half diagonal of the cell's contents
	sources    int     // meshes merged into it
	vertsTotal int
}

type level struct {
	root  int
	cells []*levelCell
	// Meshes MergeLevel hid. They stay in the scene so handles still work.
	hidden []*graphic.Mesh
	stats  levelStats
}

type levelStats struct {
	sourceMeshes, cells, skipped int
}

// mergeBucket gathers one material in one cell.
type mergeBucket struct {
	mat      material.IMaterial
	pos, nor []float32
	uv       []float32
	col      []float32 // per-vertex color when vc
	vc       bool
	idx      []uint32
	sources  int
	min, max math32.Vector3
}

// vertexColorKey is set for an untextured lit material with no maps: its
// color can ride in the vertices, so every such mesh in a cell is one draw
// whatever its color. "" means the material has to stay its own.
func vertexColorKey(im material.IMaterial) string {
	lm, ok := im.(*litMat)
	if !ok || lm.Standard == nil || lm.specMap != nil || lm.emitMap != nil || lm.normMap != nil {
		return ""
	}
	mat := lm.Standard.GetMaterial()
	if mat.Shader() != "bsshadow" || mat.TextureCount() > 0 {
		return ""
	}
	return fmt.Sprintf("vc:%d|%v|%v|%v", mat.Side(), mat.Transparent(), lm.Standard.EmissiveColor(), lm.recvShadow)
}

// materialKey groups meshes that draw the same. A glTF material is shared
// by pointer; script-built objects each own a Standard, so those group by
// what the material looks like.
func materialKey(im material.IMaterial) string {
	switch m := im.(type) {
	case *pbrMat:
		if m.Physical != nil {
			return fmt.Sprintf("pbr:%p", m.Physical)
		}
		return fmt.Sprintf("pbrw:%p", m)
	case *material.Physical:
		return fmt.Sprintf("phys:%p", m)
	}
	if lm, ok := im.(*litMat); ok && (lm.specMap != nil || lm.emitMap != nil || lm.normMap != nil) {
		return fmt.Sprintf("lit:%p", lm)
	}
	if std := unwrapStandard(im); std != nil {
		mat := std.GetMaterial()
		// G3N does not expose a Standard's textures, so a textured one only
		// merges with itself.
		if mat.TextureCount() > 0 {
			return fmt.Sprintf("tex:%p", std)
		}
		return fmt.Sprintf("std:%s|%v|%v|%d|%v",
			mat.Shader(), std.AmbientColor(), std.EmissiveColor(), mat.Side(), mat.Transparent())
	}
	return fmt.Sprintf("mat:%p", im)
}

// readAttrib returns one attribute as a flat slice of comps per vertex.
func readAttrib(g *geometry.Geometry, at gls.AttribType, comps int) []float32 {
	vbo := g.VBO(at)
	if vbo == nil || vbo.Buffer() == nil {
		return nil
	}
	buf := *vbo.Buffer()
	stride := vbo.Stride()
	off := vbo.AttribOffset(at)
	if stride < comps {
		return nil
	}
	out := make([]float32, 0, len(buf)/stride*comps)
	for i := off; i+comps <= len(buf); i += stride {
		out = append(out, buf[i:i+comps]...)
	}
	return out
}

// mergeable is false for an entity a script or the engine still uses on its
// own: collision, a body, animation, sway, or a name it can be found by.
func mergeable(e *Entity) bool {
	return e.etype == 0 && e.bodyType == 0 && e.collKind == 0 && e.anim == nil && !e.windSway &&
		(e.name == "" || e.name == "instanced" || e.name == "scatter")
}

// mergeLevel bakes the static meshes under root into cells of size cell.
// The baked entities are freed unless keep is set: 20k hidden entities still
// cost every per-entity pass a frame.
func (w *World) mergeLevel(root int, cell float32, keep bool) (*level, error) {
	e := w.ents[root]
	if e == nil || e.node == nil {
		return nil, fmt.Errorf("MergeLevel: entity %d not found", root)
	}
	if cell <= 0 {
		cell = defaultLevelCell
	}
	rn := e.node.GetNode()
	rn.UpdateMatrixWorld()
	var rootInv math32.Matrix4
	rootWorld := rn.MatrixWorld()
	if err := rootInv.GetInverse(&rootWorld); err != nil {
		rootInv.Identity()
	}
	lv := &level{root: root}
	owner := map[*graphic.Mesh]int{}
	for id, oe := range w.ents {
		if oe != nil && oe.mesh != nil && id != root {
			owner[oe.mesh] = id
		}
	}
	var baked []int
	buckets := map[string]*mergeBucket{}
	var order []string
	w.walkMeshes(e.node, func(m *graphic.Mesh) {
		n := m.GetNode()
		if !visibleUnder(n, rn) {
			return // hidden templates stay templates
		}
		g := m.GetGeometry()
		mats := m.Materials()
		// One material draws every group (a cube has six), so only the
		// material count matters.
		if g == nil || len(mats) != 1 {
			lv.stats.skipped++
			return
		}
		eid, isEnt := owner[m]
		if isEnt && !mergeable(w.ents[eid]) {
			lv.stats.skipped++
			return
		}
		pos := readAttrib(g, gls.VertexPosition, 3)
		if len(pos) < 9 {
			return
		}
		nor := readAttrib(g, gls.VertexNormal, 3)
		uv := readAttrib(g, gls.VertexTexcoord, 2)
		idx := []uint32(g.Indices())
		vcount := len(pos) / 3
		if len(idx) == 0 {
			idx = make([]uint32, vcount)
			for i := range idx {
				idx[i] = uint32(i)
			}
		}
		// Bake into the root's space, so the merged cells can live under
		// the root and still move with it.
		var mm math32.Matrix4
		mw := n.MatrixWorld()
		mm.MultiplyMatrices(&rootInv, &mw)
		var nm math32.Matrix3
		nm.GetNormalMatrix(&mm)
		var c math32.Vector3
		tpos := make([]float32, len(pos))
		for i := 0; i < vcount; i++ {
			v := math32.Vector3{X: pos[i*3], Y: pos[i*3+1], Z: pos[i*3+2]}
			v.ApplyMatrix4(&mm)
			tpos[i*3], tpos[i*3+1], tpos[i*3+2] = v.X, v.Y, v.Z
			c.Add(&v)
		}
		c.DivideScalar(float32(vcount))
		cx := int(math.Floor(float64(c.X / cell)))
		cz := int(math.Floor(float64(c.Z / cell)))
		im := mats[0].IMaterial()
		mk := vertexColorKey(im)
		vc := mk != ""
		if !vc {
			mk = materialKey(im)
		}
		key := fmt.Sprintf("%d,%d|%s", cx, cz, mk)
		b := buckets[key]
		if b == nil {
			b = &mergeBucket{mat: im, vc: vc, min: math32.Vector3{X: math32.Infinity, Y: math32.Infinity, Z: math32.Infinity},
				max: math32.Vector3{X: -math32.Infinity, Y: -math32.Infinity, Z: -math32.Infinity}}
			buckets[key] = b
			order = append(order, key)
		}
		base := uint32(len(b.pos) / 3)
		b.pos = append(b.pos, tpos...)
		for i := 0; i < vcount; i++ {
			v := math32.Vector3{X: tpos[i*3], Y: tpos[i*3+1], Z: tpos[i*3+2]}
			b.min.Min(&v)
			b.max.Max(&v)
			nv := math32.Vector3{Y: 1}
			if len(nor) >= (i+1)*3 {
				nv = math32.Vector3{X: nor[i*3], Y: nor[i*3+1], Z: nor[i*3+2]}
				nv.ApplyMatrix3(&nm)
				nv.Normalize()
			}
			b.nor = append(b.nor, nv.X, nv.Y, nv.Z)
			if len(uv) >= (i+1)*2 {
				b.uv = append(b.uv, uv[i*2], uv[i*2+1])
			} else {
				b.uv = append(b.uv, 0, 0)
			}
		}
		if b.vc {
			c := unwrapStandard(im).AmbientColor()
			for i := 0; i < vcount; i++ {
				b.col = append(b.col, c.R, c.G, c.B)
			}
		}
		for _, ix := range idx {
			b.idx = append(b.idx, base+ix)
		}
		b.sources++
		lv.stats.sourceMeshes++
		if isEnt && !keep {
			baked = append(baked, eid)
		} else {
			lv.hidden = append(lv.hidden, m)
		}
	})
	sort.Strings(order)
	for _, key := range order {
		b := buckets[key]
		g := geometry.NewGeometry()
		g.SetIndices(math32.ArrayU32(b.idx))
		g.AddVBO(gls.NewVBO(math32.ArrayF32(b.pos)).AddAttrib(gls.VertexPosition))
		g.AddVBO(gls.NewVBO(math32.ArrayF32(b.nor)).AddAttrib(gls.VertexNormal))
		g.AddVBO(gls.NewVBO(math32.ArrayF32(b.uv)).AddAttrib(gls.VertexTexcoord))
		mat := b.mat
		if b.vc {
			g.AddVBO(gls.NewVBO(math32.ArrayF32(b.col)).AddAttrib(gls.VertexColor))
			src := b.mat.(*litMat)
			std := w.newMat()
			std.SetShader("bsshadow")
			std.SetColor(&math32.Color{R: 1, G: 1, B: 1})
			std.SetEmissiveColor(ptrColor(src.Standard.EmissiveColor()))
			std.SetSide(src.Standard.GetMaterial().Side())
			std.SetTransparent(src.Standard.GetMaterial().Transparent())
			lm := newLitMat(w, std)
			lm.recvShadow = src.recvShadow
			lm.vertexColor = true
			mat = lm
		}
		mesh := graphic.NewMesh(g, mat)
		mesh.SetCullable(true)
		id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: unwrapStandard(b.mat), name: "levelcell", kind: "mesh"}, root)
		var mid, size math32.Vector3
		mid.AddVectors(&b.min, &b.max).MultiplyScalar(0.5)
		size.SubVectors(&b.max, &b.min)
		lv.cells = append(lv.cells, &levelCell{
			ent: id, cx: mid.X, cz: mid.Z, rad: size.Length() * 0.5,
			sources: b.sources, vertsTotal: len(b.pos) / 3,
		})
	}
	// Out of the scene graph as well as hidden: G3N would still update
	// 10k transforms a frame for meshes nothing draws.
	for _, m := range lv.hidden {
		m.SetVisible(false)
		if p := m.GetNode().Parent(); p != nil {
			p.GetNode().Remove(m)
		}
	}
	for _, id := range baked {
		w.freeEntityID(id)
	}
	lv.stats.cells = len(lv.cells)
	if w.levels == nil {
		w.levels = map[int]*level{}
	}
	w.levels[root] = lv
	return lv, nil
}

// visibleUnder is false if n or a parent up to stop is hidden.
func visibleUnder(n, stop *core.Node) bool {
	for p := n; p != nil; {
		if !p.Visible() {
			return false
		}
		if p == stop {
			return true
		}
		par := p.Parent()
		if par == nil {
			return true
		}
		p = par.GetNode()
	}
	return true
}

// tickLevels hides cells past the fog. The camera far plane would clip
// them anyway; hiding skips their draw and shadow cost.
func (w *World) tickLevels() {
	if len(w.levels) == 0 || w.cam == nil {
		return
	}
	far := float32(0)
	if w.fogMode != 0 && w.fogFar > 0 {
		far = w.fogFar
	}
	if far <= 0 {
		return
	}
	p := worldPos(w.cam.GetNode())
	for _, lv := range w.levels {
		re := w.ents[lv.root]
		if re == nil || re.node == nil {
			continue
		}
		// The cached matrix; WorldPosition would recompute the subtree.
		rm := re.node.GetNode().MatrixWorld()
		rp := math32.Vector3{X: rm[12], Y: rm[13], Z: rm[14]}
		for _, c := range lv.cells {
			e := w.ents[c.ent]
			if e == nil || e.node == nil {
				continue
			}
			dx := rp.X + c.cx - p.X
			dz := rp.Z + c.cz - p.Z
			lim := far + c.rad
			e.node.GetNode().SetVisible(dx*dx+dz*dz <= lim*lim)
		}
	}
}

func (w *World) levelCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// LoadLevel(path$ [, cell#]) loads a model file and merges its static
		// meshes into culled cells. Returns the level's root entity.
		"loadlevel": need(func(a []value.Value) (value.Value, error) {
			id, err := w.loadMeshFile(argS(a, 0), 0)
			if err != nil {
				return value.Num(0), err
			}
			if _, err := w.mergeLevel(id, float32(argN(a, 1, defaultLevelCell)), false); err != nil {
				return value.Num(0), err
			}
			return value.Num(float64(id)), nil
		}),
		// MergeLevel(root [, cell# [, keep]]) does the same to anything already
		// built. The merged entities are freed; keep = True hides them instead
		// so their handles still answer (at a per-frame cost).
		"mergelevel": n(func(a []value.Value) (value.Value, error) {
			lv, err := w.mergeLevel(argI(a, 0, 0), float32(argN(a, 1, defaultLevelCell)), argI(a, 2, 0) != 0)
			if err != nil {
				return value.Num(0), err
			}
			return value.Num(float64(lv.stats.cells)), nil
		}),
		// LevelCells(root) / LevelSources(root): merged draws and the meshes
		// they replaced.
		"levelcells": n(func(a []value.Value) (value.Value, error) {
			if lv := w.levels[argI(a, 0, 0)]; lv != nil {
				return value.Num(float64(lv.stats.cells)), nil
			}
			return value.Num(0), nil
		}),
		"levelsources": n(func(a []value.Value) (value.Value, error) {
			if lv := w.levels[argI(a, 0, 0)]; lv != nil {
				return value.Num(float64(lv.stats.sourceMeshes)), nil
			}
			return value.Num(0), nil
		}),
	}
}

func ptrColor(c math32.Color) *math32.Color { return &c }
