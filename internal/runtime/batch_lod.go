package runtime

import (
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/meshsimplify"
)

// Distance detail for batched copies. A mesh of batchLODMinTris or more gets up
// to two simpler versions, built once on a worker. Each view then draws a copy
// with the simplest version whose error covers at most a pixel at its distance,
// and each cascade with the simplest whose error fits in one shadow texel.
// A single mesh with no copies keeps its full detail.

const (
	batchLODMinTris = 256
	batchLODMaxTris = 500000
	batchLODMaxErr  = 1.0 // pixels
)

var batchLODRatios = [...]float64{0.5, 0.25}

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
	}
	b.lods[geom] = nil
	if tris := geomTriangles(geom); tris < batchLODMinTris || tris > batchLODMaxTris {
		return nil
	}
	mesh := meshsimplify.Mesh{Positions: positionSlice(geom), Indices: append([]uint32(nil), geom.Indices()...)}
	job := &lodJob{geom: geom, indexN: len(geom.Indices()), pos: posV.Buffer(), vertN: len(*posV.Buffer())}
	done := b.lodDone
	go func() {
		job.results = simplifyForLOD(mesh)
		done <- job
	}()
	return nil
}

// simplifyForLOD runs on a worker: no GL, and nothing shared with the world.
func simplifyForLOD(mesh meshsimplify.Mesh) []meshsimplify.Result {
	var out []meshsimplify.Result
	input := len(mesh.Indices) / 3
	for _, ratio := range batchLODRatios {
		r := meshsimplify.Simplify(mesh, meshsimplify.Options{TargetRatio: ratio})
		// A version that barely saves anything is not worth a draw call.
		if r.OutputTriangles < 1 || len(r.Sources) == 0 || float64(r.OutputTriangles) > float64(input)*ratio*1.5 {
			break
		}
		out = append(out, r)
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
	for _, r := range job.results {
		g := lodGeometry(geom, r)
		if g == nil {
			break
		}
		l.levels = append(l.levels, lodLevel{g, float32(r.MaxCollapseError)})
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
