package runtime

import (
	"math"
	"runtime"
	"unsafe"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/meshsimplify"
)

// Distance detail for batched copies. A mesh of batchLODMinTris or more gets up
// to two simpler versions, built once on a worker. Each view then draws a copy
// with the simplest version whose error covers at most a pixel at its distance,
// and each cascade with the simplest whose error fits in one shadow texel.
// A large single mesh gets the same treatment, as a group of one. A large part inside a
// loaded model is not an entity and cannot join a group, so it swaps its geometry
// instead, once a frame, for the whole of that frame's passes.

const (
	batchLODMinTris = 256
	batchLODMaxTris = 500000
	batchLODMaxErr  = 1.0 // pixels
	// Below this a lone mesh is not worth a batch of its own.
	singleLODMinTris = 2000
)

// Each level is built from the full mesh, on a worker, so more of them cost load time
// and no frame time. A far mesh would otherwise stop at a quarter of its triangles.
var batchLODRatios = [...]float64{0.5, 0.25, 0.125, 0.0625}

type lodLevel struct {
	geom *geometry.Geometry
	err  float32 // largest error, in the mesh's own units
}

// geomLOD holds a geometry's simpler versions, simplest last.
type geomLOD struct {
	levels []lodLevel
	// What the versions were built from. A later edit of the mesh drops them.
	indexN, vertN int
	pos           *math32.ArrayF32
}

type lodJob struct {
	geom      *geometry.Geometry
	indexN    int
	pos       *math32.ArrayF32
	vertN     int
	results   []meshsimplify.Result
	abandoned bool
}

func geomTriangles(g *geometry.Geometry) int {
	if idx := g.Indices(); len(idx) > 0 {
		return len(idx) / 3
	}
	return g.Items() / 3
}

// lodFor returns a geometry's versions, starting the build the first time. It
// is nil until they are ready, and for meshes too small or too large to need them.
func (w *World) lodFor(geom *geometry.Geometry) *geomLOD {
	b := &w.batch
	posV := geom.VBO(gls.VertexPosition)
	if posV == nil || posV.Buffer() == nil || len(geom.Indices()) < 3 {
		return nil
	}
	if l, ok := b.lods[geom]; ok {
		if l == nil {
			return nil // building, or not worth it
		}
		if l.indexN != len(geom.Indices()) || l.pos != posV.Buffer() || l.vertN != len(*posV.Buffer()) {
			delete(b.lods, geom) // edited since: build again next frame
			return nil
		}
		return l
	}
	if b.lods == nil {
		b.lods = map[*geometry.Geometry]*geomLOD{}
		b.lodDone = make(chan *lodJob, 16)
		// A quarter of the cores: on integrated graphics, a busy CPU also slows the GPU.
		n := runtime.NumCPU() / 4
		if n < 1 {
			n = 1
		}
		b.lodSlots = make(chan struct{}, n)
	}
	b.lods[geom] = nil
	if tris := geomTriangles(geom); tris < batchLODMinTris || tris > batchLODMaxTris {
		return nil
	}
	mesh := meshsimplify.Mesh{Positions: positionSlice(geom), Indices: append([]uint32(nil), geom.Indices()...)}
	job := &lodJob{geom: geom, indexN: len(geom.Indices()), pos: posV.Buffer(), vertN: len(*posV.Buffer())}
	done := b.lodDone
	slots := b.lodSlots
	go func() {
		// A scene of many large meshes queues here, leaving cores for the game.
		slots <- struct{}{}
		job.results = simplifyForLOD(mesh)
		<-slots
		done <- job
	}()
	return nil
}

// weldGeometry returns a copy of a geometry sharing each vertex that is the same in every
// attribute, or nil if it would gain nothing. OBJ files load with three vertices for every
// triangle, whatever their indices say: more to shade, and to the simplifier, a pile of
// separate triangles with nothing it can collapse. The index order is kept, so a material
// group's range still covers the same triangles.
func weldGeometry(g *geometry.Geometry) *geometry.Geometry {
	vbos := g.VBOs()
	if len(vbos) == 0 {
		return nil
	}
	strides := make([]int, len(vbos))
	verts := -1
	for i, v := range vbos {
		if v.Buffer() == nil || v.Stride() <= 0 {
			return nil
		}
		strides[i] = v.Stride()
		// A model with no texture coordinates has an empty buffer for them.
		if c := len(*v.Buffer()) / strides[i]; c > 0 && (verts < 0 || c < verts) {
			verts = c
		}
	}
	source := g.Indices()
	n := len(source)
	if n == 0 {
		n = g.Items()
	}
	if n < 3 || verts < 3 {
		return nil
	}
	seen := make(map[string]uint32, verts/2)
	indices := make([]uint32, n)
	keep := make([]int, 0, verts/2)
	var key []byte
	for at := 0; at < n; at++ {
		i := at
		if len(source) > 0 {
			i = int(source[at])
		}
		if i >= verts {
			return nil
		}
		key = key[:0]
		for b, v := range vbos {
			if len(*v.Buffer()) == 0 {
				continue
			}
			for _, f := range (*v.Buffer())[i*strides[b] : (i+1)*strides[b]] {
				u := math.Float32bits(f)
				key = append(key, byte(u), byte(u>>8), byte(u>>16), byte(u>>24))
			}
		}
		to, ok := seen[string(key)]
		if !ok {
			to = uint32(len(keep))
			seen[string(key)] = to
			keep = append(keep, i)
		}
		indices[at] = to
	}
	if len(keep)*4 > verts*3 {
		return nil // hard edges everywhere: little to share
	}
	out := geometry.NewGeometry()
	for b, v := range vbos {
		src := *v.Buffer()
		buf := make([]float32, 0, len(keep)*strides[b])
		for _, i := range keep {
			if len(src) > 0 {
				buf = append(buf, src[i*strides[b]:(i+1)*strides[b]]...)
			}
		}
		nv := gls.NewVBO(math32.ArrayF32(buf))
		for _, a := range v.Attributes() {
			if a.Type != gls.Undefined {
				nv.AddAttribOffset(a.Type, a.ByteOffset)
			} else {
				nv.AddCustomAttribOffset(a.Name, a.NumElements, a.ByteOffset)
			}
		}
		out.AddVBO(nv)
	}
	out.SetIndices(math32.ArrayU32(indices))
	for i := 0; i < g.GroupCount(); i++ {
		gr := g.GroupAt(i)
		out.AddGroup(gr.Start, gr.Count, gr.Matindex)
	}
	return out
}

// weldLoadedMeshes indexes a loaded model's large meshes.
func (w *World) weldLoadedMeshes(node core.INode) {
	w.walkMeshes(node, func(m *graphic.Mesh) {
		g := m.GetGeometry()
		if g == nil || geomTriangles(g) < singleLODMinTris {
			return
		}
		if _, morph := m.IGeometry().(*geometry.MorphGeometry); morph {
			return
		}
		if welded := weldGeometry(g); welded != nil {
			m.SetIGeometry(welded)
			g.Dispose()
		}
	})
}

// wantsOwnLOD says a mesh with no copies should still be drawn through a batch.
func wantsOwnLOD(g *batchGroup) bool {
	if len(g.members) != 1 || geomTriangles(g.key.geom) < singleLODMinTris {
		return false
	}
	// Planted trees come in blocks that already have their own far versions, and come
	// and go as the camera moves: simplifying them would never end.
	return g.members[0].e == nil || g.members[0].e.name != "scatter"
}

type partLOD struct {
	mesh *graphic.Mesh
	full *geometry.Geometry
	at   *geometry.Geometry
}

// tickPartLODs picks the detail of large parts inside loaded models.
func (w *World) tickPartLODs() {
	b := &w.batch
	if w.cam == nil {
		return
	}
	// What is in a model rarely changes, so look for parts once a second.
	if w.loopFrames%60 == 0 || b.parts == nil {
		w.findPartLODs()
	}
	if len(b.parts) == 0 {
		return
	}
	var proj math32.Matrix4
	w.cam.ProjMatrix(&proj)
	if proj[15] != 0 || proj[5] <= 0 {
		return
	}
	var vp [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &vp[0])
	focal := proj[5] * float32(vp[3]) * 0.5
	eye := worldPos(w.cam.GetNode())
	for _, p := range b.parts {
		target := p.full
		if l := w.lodFor(p.full); l != nil {
			sph := p.full.BoundingSphere()
			mw := p.mesh.GetNode().MatrixWorld()
			c := sph.Center
			c.ApplyMatrix4(&mw)
			scale := math32.Sqrt(math32.Max(mw[0]*mw[0]+mw[1]*mw[1]+mw[2]*mw[2], math32.Max(mw[4]*mw[4]+mw[5]*mw[5]+mw[6]*mw[6], mw[8]*mw[8]+mw[9]*mw[9]+mw[10]*mw[10])))
			if dist := c.DistanceTo(&eye) - sph.Radius*scale; dist > 0 {
				if i := pickLOD(l.levels, scale, focal/dist); i > 0 {
					target = l.levels[i-1].geom
				}
			}
		}
		if target != p.at {
			if target != p.full {
				target.Incref() // the mesh disposes what it holds
			}
			p.mesh.SetIGeometry(target)
			p.at = target
		}
	}
}

// findPartLODs lists meshes under loaded models that can change detail: one plain
// material over a whole, static, indexed geometry.
func (w *World) findPartLODs() {
	b := &w.batch
	old := map[*graphic.Mesh]*partLOD{}
	for _, p := range b.parts {
		old[p.mesh] = p
	}
	b.parts = make([]*partLOD, 0, len(old))
	for _, e := range w.ents {
		if e == nil || e.node == nil || e.skin != nil {
			continue
		}
		if _, plain := e.node.(*graphic.Mesh); plain {
			continue // an entity's own mesh: batching looks after it
		}
		w.walkMeshes(e.node, func(m *graphic.Mesh) {
			if p := old[m]; p != nil {
				b.parts = append(b.parts, p)
				return
			}
			if _, morph := m.IGeometry().(*geometry.MorphGeometry); morph || !partCanSwap(m) {
				return
			}
			if g := m.GetGeometry(); g != nil && len(g.Indices()) >= 3 && geomTriangles(g) >= singleLODMinTris {
				b.parts = append(b.parts, &partLOD{mesh: m, full: g, at: g})
			}
		})
	}
}

// partCanSwap: a material over part of the indices would read past a simpler version.
func partCanSwap(m *graphic.Mesh) bool {
	mats := m.Materials()
	return len(mats) == 1 && *(*int)(unsafe.Add(unsafe.Pointer(&mats[0]), batchLayout.matCount)) == 0
}

// restorePartLODs gives an entity's parts back their full geometry before it is freed,
// so the mesh disposes what it came with.
func (w *World) restorePartLODs(e *Entity) {
	b := &w.batch
	if len(b.parts) == 0 || e == nil || e.node == nil {
		return
	}
	w.walkMeshes(e.node, func(m *graphic.Mesh) {
		for i, p := range b.parts {
			if p.mesh == m {
				m.SetIGeometry(p.full)
				b.parts = append(b.parts[:i], b.parts[i+1:]...)
				return
			}
		}
	})
}

// simplifyForLOD runs on a worker: no GL, and nothing shared with the world.
//
// Each level is made from the one before, so its Sources point there. Starting each from
// the full mesh took about three times the work, and a scene of many large meshes spent
// minutes on it.
func simplifyForLOD(mesh meshsimplify.Mesh) []meshsimplify.Result {
	var out []meshsimplify.Result
	input := len(mesh.Indices) / 3
	for _, ratio := range batchLODRatios {
		r := meshsimplify.Simplify(mesh, meshsimplify.Options{TargetTriangles: int(float64(input) * ratio)})
		// A version that barely saves anything is not worth a draw call.
		if r.OutputTriangles < 1 || len(r.Sources) == 0 || float64(r.OutputTriangles) > float64(input)*ratio*1.5 {
			break
		}
		out = append(out, r)
		mesh = r.Mesh
	}
	return out
}

// collectLODs adopts finished builds. New geometry is made here, on the GL
// thread, because a buffer may only be read safely while nothing can edit it.
func (w *World) collectLODs() {
	b := &w.batch
	for b.lodDone != nil {
		select {
		case job := <-b.lodDone:
			w.adoptLOD(job)
		default:
			return
		}
	}
}

func (w *World) adoptLOD(job *lodJob) {
	b := &w.batch
	geom := job.geom
	posV := geom.VBO(gls.VertexPosition)
	if posV == nil || posV.Buffer() != job.pos || len(*posV.Buffer()) != job.vertN || len(geom.Indices()) != job.indexN {
		delete(b.lods, geom)
		return
	}
	l := &geomLOD{indexN: job.indexN, vertN: job.vertN, pos: job.pos}
	from := geom
	var err float32
	for _, r := range job.results {
		g := lodGeometry(from, r)
		if g == nil {
			break
		}
		// Each pass measures only its own step away from the last, so errors add up.
		err += float32(r.MaxCollapseError)
		l.levels = append(l.levels, lodLevel{g, err})
		from = g
	}
	if len(l.levels) > 0 {
		b.lods[geom] = l
	}
}

// lodGeometry is a new geometry with the simplified triangles and every vertex
// attribute of the original carried over.
func lodGeometry(geom *geometry.Geometry, r meshsimplify.Result) *geometry.Geometry {
	blended := blendAttrs(geom, r.Sources)
	out := geometry.NewGeometry()
	for _, vbo := range geom.VBOs() {
		buf, ok := blended[vbo]
		if !ok {
			return nil
		}
		nv := gls.NewVBO(math32.ArrayF32(buf))
		for _, a := range vbo.Attributes() {
			if a.Type != gls.Undefined {
				nv.AddAttribOffset(a.Type, a.ByteOffset)
			} else {
				nv.AddCustomAttribOffset(a.Name, a.NumElements, a.ByteOffset)
			}
		}
		out.AddVBO(nv)
	}
	out.SetIndices(math32.ArrayU32(append([]uint32(nil), r.Mesh.Indices...)))
	return out
}

// pickLOD returns the simplest level whose error, in pixels, stays within the
// limit: 0 is the full mesh, 1 the first simpler version. unitsToPixels is how
// many pixels one world unit covers at the copy's distance.
func pickLOD(levels []lodLevel, scale, unitsToPixels float32) int {
	for i := len(levels) - 1; i >= 0; i-- {
		if levels[i].err*scale*unitsToPixels <= batchLODMaxErr {
			return i + 1
		}
	}
	return 0
}

// drawLODBins draws each level's copies. The first is the geometry G3N has
// already set up; the caller's next draw sets up its own.
func (w *World) drawLODBins(g *batchGroup, gs *gls.GLS) int {
	b := &w.batch
	drew := 0
	for i := range b.lodBins {
		data := b.lodBins[i]
		if len(data) == 0 {
			continue
		}
		geom := g.key.geom
		if i > 0 {
			geom = g.lod.levels[i-1].geom
		}
		geom.RenderSetup(gs)
		b.scratch = append(b.scratch[:0], data...)
		w.drawBatchInstances(geom, len(data)/batchFloats)
		drew++
	}
	return drew
}

func (b *batchState) resetLODBins() {
	for i := range b.lodBins {
		b.lodBins[i] = b.lodBins[i][:0]
	}
}

// binViewLODs sorts a pass's visible copies by level. It reports false when the
// view has no usable scale (an orthographic camera), and the caller draws them all
// at full detail.
func (p *batchProxy) binViewLODs(view, proj *math32.Matrix4, keep func(i int) bool) bool {
	if proj[15] != 0 || proj[5] <= 0 {
		return false
	}
	g := p.g
	b := &p.w.batch
	var vp [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &vp[0])
	focal := proj[5] * float32(vp[3]) * 0.5
	rBase := g.key.geom.BoundingSphere().Radius
	if rBase <= 0 {
		return false
	}
	b.resetLODBins()
	for i := range g.nodes {
		if !keep(i) {
			continue
		}
		s := g.spheres[i*4 : i*4+4]
		x := view[0]*s[0] + view[4]*s[1] + view[8]*s[2] + view[12]
		y := view[1]*s[0] + view[5]*s[1] + view[9]*s[2] + view[13]
		z := view[2]*s[0] + view[6]*s[1] + view[10]*s[2] + view[14]
		dist := math32.Sqrt(x*x+y*y+z*z) - s[3]
		level := 0
		if dist > 0 {
			level = pickLOD(g.lod.levels, s[3]/rBase, focal/dist)
		}
		b.lodBins[level] = append(b.lodBins[level], g.mats[i*batchFloats:(i+1)*batchFloats]...)
	}
	return true
}

// binDepthLODs is binViewLODs for a shadow tile. A directional tile has the same
// scale everywhere: one texel is 2 / (scale * size) world units across. A point or
// spot tile has perspective, so it stays at full detail.
func (w *World) binDepthLODs(g *batchGroup, lightVP *math32.Matrix4, keep func(i int) bool) bool {
	l := lightVP
	if l[3] != 0 || l[7] != 0 || l[11] != 0 || w.shadow.size <= 0 {
		return false
	}
	rBase := g.key.geom.BoundingSphere().Radius
	sx := math32.Sqrt(l[0]*l[0] + l[4]*l[4] + l[8]*l[8])
	if rBase <= 0 || sx <= 0 {
		return false
	}
	// pixels per unit: one texel is the limit, as a pixel is on screen.
	texels := sx * float32(w.shadow.size) * 0.5
	b := &w.batch
	b.resetLODBins()
	for i := range g.nodes {
		if !keep(i) {
			continue
		}
		level := pickLOD(g.lod.levels, g.spheres[i*4+3]/rBase, texels)
		b.lodBins[level] = append(b.lodBins[level], g.mats[i*batchFloats:(i+1)*batchFloats]...)
	}
	return true
}
