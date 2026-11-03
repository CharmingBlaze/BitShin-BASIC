package runtime

import (
	"sort"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

// Static merging. A level exported from a modeller has thousands of meshes that
// are all different, so instancing cannot help them, and each one costs a draw
// call in the scene and in every shadow tile. A mesh with no copies that has not
// changed for batchMergeFrames is baked in world space, with the others of the
// same material in its area of the level, into one mesh. Any change to it (a
// move, a new color, hiding it) takes it out, and its area is rebuilt without it.

const (
	batchMergeFrames   = 30
	batchMergeCell     = 512
	batchMergeMaxTris  = 20000 // bigger meshes are worth their own draw and culling
	batchMergeMinGroup = 2
)

// mergeKey is a material's key with no geometry, plus an area of the level.
type mergeKey struct {
	mat    batchKey
	cx, cz int
}

type mergeGroup struct {
	key     mergeKey
	members []batchMember // current, by id
	next    []batchMember // gathered this frame
	mesh    *graphic.Mesh
	geom    *geometry.Geometry
	sphere  [4]float32
	double  bool
}

// mergeOutEntity returns a merged entity to being drawn on its own.
func mergeOutEntity(e *Entity) {
	st := e.batch
	if st.merged == nil {
		return
	}
	st.merged = nil
	st.drawn = false
	e.mesh.SetRenderable(true)
}

// canMerge is whether a still mesh may be baked. It rules out anything whose
// vertices can change without its matrix or material changing.
func (w *World) canMerge(id int, e *Entity, key *batchKey, mw *math32.Matrix4) bool {
	if e.skin != nil || e.shadowGeomDirty || w.meshBuilds[id] != nil || e.anim != nil {
		return false
	}
	if key.vertexColor || key.cutTex != nil {
		return false
	}
	node := e.mesh.GetNode()
	if !nodeShown(node) || shadowSkipMesh(e) || len(node.Children()) != 0 {
		return false
	}
	if key.geom.VBO(gls.VertexPosition) == nil || key.geom.VBO(gls.VertexNormal) == nil {
		return false
	}
	if geomTriangles(key.geom) > batchMergeMaxTris {
		return false
	}
	return mw.Determinant() > 0 // a mirror would turn its triangles inside out
}

// tickMergeMember handles one mesh that has no copies, once a frame.
func (w *World) tickMergeMember(m batchMember) {
	e := m.e
	st := e.batch
	mw := e.mesh.GetNode().MatrixWorld()
	if mw != st.mat {
		st.mat = mw
		st.still = 0
		mergeOutEntity(e)
		return
	}
	if st.still < batchMergeFrames {
		st.still++
		return
	}
	if !w.canMerge(m.id, e, &st.key, &mw) {
		mergeOutEntity(e)
		return
	}
	mat := st.key
	mat.geom = nil
	cx, cz := int(math32.Floor(mw[12]/batchMergeCell)), int(math32.Floor(mw[14]/batchMergeCell))
	key := mergeKey{mat, cx, cz}
	b := &w.batch
	g := b.merges[key]
	if g == nil {
		if b.merges == nil {
			b.merges = map[mergeKey]*mergeGroup{}
		}
		g = &mergeGroup{key: key, double: !mat.pbr && mat.side == material.SideDouble}
		b.merges[key] = g
	}
	if st.merged != nil && st.merged != g {
		mergeOutEntity(e)
	}
	g.next = append(g.next, m)
}

// applyMerges rebuilds the areas whose members changed this frame. Sorted, because
// groups are visited in map order and an unchanged list must compare equal.
func (w *World) applyMerges() {
	b := &w.batch
	for key, g := range b.merges {
		sort.Slice(g.next, func(i, j int) bool { return g.next[i].id < g.next[j].id })
		same := len(g.next) == len(g.members)
		if same {
			for i := range g.next {
				if g.next[i] != g.members[i] || g.next[i].e.batch.merged != g {
					same = false
					break
				}
			}
		}
		if same {
			g.next = g.next[:0]
			continue
		}
		for _, m := range g.members {
			// One that left has gone back to G3N; the rest are still baked.
			if m.e.batch != nil && m.e.batch.merged == g {
				mergeOutEntity(m.e)
			}
		}
		g.dropMesh(w)
		g.members, g.next = g.next, g.members[:0]
		if len(g.members) < batchMergeMinGroup {
			g.members = g.members[:0]
			if len(g.next) == 0 {
				delete(b.merges, key)
			}
			continue
		}
		w.buildMerged(g)
	}
}

func (g *mergeGroup) dropMesh(w *World) {
	if g.mesh == nil {
		return
	}
	w.scene.Remove(g.mesh)
	g.geom.Dispose()
	g.mesh, g.geom = nil, nil
}

// buildMerged bakes the members into one world-space mesh.
func (w *World) buildMerged(g *mergeGroup) {
	verts, tris := 0, 0
	for _, m := range g.members {
		src := m.e.batch.key.geom
		verts += src.VBO(gls.VertexPosition).Buffer().Size() / src.VBO(gls.VertexPosition).Stride()
		tris += geomTriangles(src)
	}
	pos := math32.NewArrayF32(0, verts*3)
	nor := math32.NewArrayF32(0, verts*3)
	uvs := math32.NewArrayF32(0, verts*2)
	idx := math32.NewArrayU32(0, tris*3)
	for _, m := range g.members {
		src := m.e.batch.key.geom
		mw := m.e.mesh.GetNode().MatrixWorld()
		appendMergedGeometry(src, &mw, &pos, &nor, &uvs, &idx)
		m.e.batch.merged = g
		m.e.batch.drawn = true
		m.e.mesh.SetRenderable(false)
	}
	geom := geometry.NewGeometry()
	geom.SetIndices(idx)
	geom.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	geom.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	geom.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	rep := g.members[0].e.mesh.Materials()[0].IMaterial()
	g.geom = geom
	g.mesh = graphic.NewMesh(geom, rep)
	sph := geom.BoundingSphere()
	g.sphere = [4]float32{sph.Center.X, sph.Center.Y, sph.Center.Z, sph.Radius}
	w.scene.Add(g.mesh)
}

// appendMergedGeometry adds one mesh, moved to the world, to the merged arrays.
func appendMergedGeometry(src *geometry.Geometry, mw *math32.Matrix4, pos, nor, uvs *math32.ArrayF32, idx *math32.ArrayU32) {
	base := uint32(pos.Size() / 3)
	var nm math32.Matrix3
	nm.GetNormalMatrix(mw)
	src.ReadVertices(func(v math32.Vector3) bool {
		v.ApplyMatrix4(mw)
		pos.Append(v.X, v.Y, v.Z)
		return false
	})
	n := pos.Size()/3 - int(base)
	src.ReadVertexNormals(func(v math32.Vector3) bool {
		v.ApplyMatrix3(&nm)
		v.Normalize()
		nor.Append(v.X, v.Y, v.Z)
		return false
	})
	if uv := src.VBO(gls.VertexTexcoord); uv != nil {
		buf := *uv.Buffer()
		off := uv.AttribOffset(gls.VertexTexcoord)
		for i := 0; i < n; i++ {
			at := i*uv.Stride() + off
			uvs.Append(buf[at], buf[at+1])
		}
	} else {
		for i := 0; i < n; i++ {
			uvs.Append(0, 0)
		}
	}
	if sidx := src.Indices(); len(sidx) > 0 {
		for _, v := range sidx {
			idx.Append(base + v)
		}
	} else {
		for i := 0; i < n; i++ {
			idx.Append(base + uint32(i))
		}
	}
}

// drawMergedDepth draws the merged meshes into the current shadow tile, as one
// instance each with no transform of its own.
func (w *World) drawMergedDepth(gs *gls.GLS, lightVP *math32.Matrix4, prog func(*geometry.Geometry) *gls.Program) int {
	b := &w.batch
	drew := 0
	var identity math32.Matrix4
	identity.Identity()
	for _, g := range b.merges {
		if g.mesh == nil || !sphereInLightBox(lightVP, g.sphere[:]) {
			continue
		}
		if g.double {
			gs.Disable(gls.CULL_FACE)
		} else {
			gs.Enable(gls.CULL_FACE)
			gs.CullFace(gls.BACK)
		}
		p := prog(g.geom)
		gs.UseProgram(p)
		gs.UniformMatrix4fv(p.GetUniformLocation("BatchLightVP"), 1, false, &lightVP[0])
		w.setDepthMomentUniformsFor(gs, p)
		g.geom.RenderSetup(gs)
		b.scratch = append(b.scratch[:0], identity[:]...)
		w.drawBatchInstances(g.geom, 1)
		drew++
	}
	return drew
}

// mergedDrawCount is how many merged meshes exist.
func (w *World) mergedDrawCount() int {
	n := 0
	for _, g := range w.batch.merges {
		if g.mesh != nil {
			n++
		}
	}
	return n
}
