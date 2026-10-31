package runtime

import (
	"fmt"
	"image"
	"math"
	"os"
	"sync"

	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Runtime terrain editing: raise / lower / flatten / smooth brushes, single
// lattice writes, and splat painting.
//
// Heights are CPU-side for every terrain kind. A finite heightmap
// (CreateTerrain(file or hm), LoadHeightmap) is edited in place on its grid.
// Procedural terrains (CreateTerrain("default"), CreateProcTerrain,
// CreateTerrainGL) keep their noise and add a sparse delta lattice on top
// (sculptOverlay), so an infinite field can still be dug into.
//
// Brushes change heights at once (TerrainHeight / TerrainSlope / PickTerrain
// see them on the same line). The touched chunk meshes, their near-chunk
// collision and any CreateBodyHeightfield body are rebuilt once per frame in
// tickTerrain, however many brush calls ran. Scatter on a touched chunk is
// replanted after the brush has rested for a few frames.

const (
	sculptTile       = 32  // overlay tile edge in lattice cells
	sculptMaxCells   = 512 // brush span cap per axis (cells)
	sculptScatterRst = 12  // frames without edits before scatter is replanted
	paintDefaultRes  = 512
)

type sculptKey struct{ X, Z int32 }

// sculptOverlay is a sparse height delta on a regular lattice in heightfield
// space. Lattice node (i, j) sits at (ox + i*dx, oz + j*dz). Horizon, vista
// and tree workers sample heights off the GL thread, so reads take the read
// lock and brush writes the write lock.
type sculptOverlay struct {
	mu                     sync.RWMutex
	ox, oz, dx, dz         float32
	tiles                  map[sculptKey]*[sculptTile * sculptTile]float32
	minI, minJ, maxI, maxJ int
}

func newSculptOverlay(ox, oz, dx, dz float32) *sculptOverlay {
	if dx < 0.05 {
		dx = 0.05
	}
	if dz < 0.05 {
		dz = 0.05
	}
	return &sculptOverlay{ox: ox, oz: oz, dx: dx, dz: dz, tiles: map[sculptKey]*[sculptTile * sculptTile]float32{}}
}

func (o *sculptOverlay) empty() bool {
	if o == nil {
		return true
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.tiles) == 0
}

func (o *sculptOverlay) get(i, j int) float32 {
	if o == nil {
		return 0
	}
	o.mu.RLock()
	v := o.getLocked(i, j)
	o.mu.RUnlock()
	return v
}

func (o *sculptOverlay) getLocked(i, j int) float32 {
	if len(o.tiles) == 0 || i < o.minI || i > o.maxI || j < o.minJ || j > o.maxJ {
		return 0
	}
	ti, tj := floorDiv(i, sculptTile), floorDiv(j, sculptTile)
	tile := o.tiles[sculptKey{int32(ti), int32(tj)}]
	if tile == nil {
		return 0
	}
	return tile[(j-tj*sculptTile)*sculptTile+(i-ti*sculptTile)]
}

func (o *sculptOverlay) add(i, j int, d float32) {
	if o == nil || d == 0 {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	ti, tj := floorDiv(i, sculptTile), floorDiv(j, sculptTile)
	k := sculptKey{int32(ti), int32(tj)}
	tile := o.tiles[k]
	if tile == nil {
		tile = new([sculptTile * sculptTile]float32)
		if len(o.tiles) == 0 {
			o.minI, o.minJ, o.maxI, o.maxJ = i, j, i, j
		}
		o.tiles[k] = tile
	}
	tile[(j-tj*sculptTile)*sculptTile+(i-ti*sculptTile)] += d
	if i < o.minI {
		o.minI = i
	}
	if i > o.maxI {
		o.maxI = i
	}
	if j < o.minJ {
		o.minJ = j
	}
	if j > o.maxJ {
		o.maxJ = j
	}
}

// sample bilinearly interpolates the delta at a heightfield-space point.
func (o *sculptOverlay) sample(x, z float32) float32 {
	if o == nil {
		return 0
	}
	u := (x - o.ox) / o.dx
	v := (z - o.oz) / o.dz
	i0 := int(math.Floor(float64(u)))
	j0 := int(math.Floor(float64(v)))
	o.mu.RLock()
	defer o.mu.RUnlock()
	if len(o.tiles) == 0 || i0+1 < o.minI || i0 > o.maxI || j0+1 < o.minJ || j0 > o.maxJ {
		return 0
	}
	tx := u - float32(i0)
	tz := v - float32(j0)
	h00 := o.getLocked(i0, j0)
	h10 := o.getLocked(i0+1, j0)
	h01 := o.getLocked(i0, j0+1)
	h11 := o.getLocked(i0+1, j0+1)
	return (h00*(1-tx)+h10*tx)*(1-tz) + (h01*(1-tx)+h11*tx)*tz
}

// sculptLattice is what a brush edits: a regular grid of height nodes in
// heightfield space. Bounded lattices (finite heightmaps) clip to [0,gw)x[0,gd).
type sculptLattice struct {
	ox, oz, dx, dz float32
	bounded        bool
	gw, gd         int
	get            func(i, j int) float32
	add            func(i, j int, d float32)
}

// sculptRect is an inclusive lattice index rectangle.
type sculptRect struct{ i0, j0, i1, j1 int }

func (r sculptRect) empty() bool { return r.i1 < r.i0 || r.j1 < r.j0 }

// brushWeight is 1 inside radius*(1-falloff), 0 at radius and beyond, and a
// smoothstep in between. falloff 0 is a hard disc, 1 is soft all the way in.
func brushWeight(dist, radius, falloff float32) float32 {
	if radius <= 0 || dist >= radius {
		return 0
	}
	if falloff < 0 {
		falloff = 0
	}
	if falloff > 1 {
		falloff = 1
	}
	inner := radius * (1 - falloff)
	if dist <= inner {
		return 1
	}
	t := (dist - inner) / (radius - inner)
	return 1 - t*t*(3-2*t)
}

// brushRect returns the lattice nodes a circle of radius r at (x, z) can touch,
// clipped to a bounded lattice and capped at sculptMaxCells per axis.
func (l *sculptLattice) brushRect(x, z, r float32) sculptRect {
	if l == nil || l.dx <= 0 || l.dz <= 0 || r <= 0 {
		return sculptRect{0, 0, -1, -1}
	}
	if lim := float32(sculptMaxCells/2) * l.dx; r > lim {
		r = lim
	}
	if lim := float32(sculptMaxCells/2) * l.dz; r > lim {
		r = lim
	}
	i0 := int(math.Ceil(float64((x - r - l.ox) / l.dx)))
	i1 := int(math.Floor(float64((x + r - l.ox) / l.dx)))
	j0 := int(math.Ceil(float64((z - r - l.oz) / l.dz)))
	j1 := int(math.Floor(float64((z + r - l.oz) / l.dz)))
	if l.bounded {
		if i0 < 0 {
			i0 = 0
		}
		if j0 < 0 {
			j0 = 0
		}
		if i1 > l.gw-1 {
			i1 = l.gw - 1
		}
		if j1 > l.gd-1 {
			j1 = l.gd - 1
		}
	}
	return sculptRect{i0, j0, i1, j1}
}

func (l *sculptLattice) nodePos(i, j int) (float32, float32) {
	return l.ox + float32(i)*l.dx, l.oz + float32(j)*l.dz
}

const (
	brushRaise = iota
	brushFlatten
	brushSmooth
)

// applyBrush runs one brush dab and returns the lattice rectangle it changed.
// raise:   h += amount * w
// flatten: h += (amount - h) * clamp(strength * w)    (amount = target height)
// smooth:  h += (avg3x3 - h) * clamp(strength * w)
func (l *sculptLattice) applyBrush(op int, x, z, radius, amount, strength, falloff float32) sculptRect {
	rc := l.brushRect(x, z, radius)
	if rc.empty() {
		return rc
	}
	w := rc.i1 - rc.i0 + 1
	h := rc.j1 - rc.j0 + 1
	// Snapshot with a one-cell border so smoothing does not read its own writes.
	sw, sh := w+2, h+2
	snap := make([]float32, sw*sh)
	clampI := func(i, n int) int {
		if !l.bounded {
			return i
		}
		if i < 0 {
			return 0
		}
		if i > n-1 {
			return n - 1
		}
		return i
	}
	if op == brushSmooth {
		for sj := 0; sj < sh; sj++ {
			for si := 0; si < sw; si++ {
				snap[sj*sw+si] = l.get(clampI(rc.i0+si-1, l.gw), clampI(rc.j0+sj-1, l.gd))
			}
		}
	} else {
		for sj := 1; sj < sh-1; sj++ {
			for si := 1; si < sw-1; si++ {
				snap[sj*sw+si] = l.get(rc.i0+si-1, rc.j0+sj-1)
			}
		}
	}
	out := sculptRect{rc.i1 + 1, rc.j1 + 1, rc.i0 - 1, rc.j0 - 1}
	for j := rc.j0; j <= rc.j1; j++ {
		for i := rc.i0; i <= rc.i1; i++ {
			px, pz := l.nodePos(i, j)
			ddx, ddz := px-x, pz-z
			wt := brushWeight(float32(math.Sqrt(float64(ddx*ddx+ddz*ddz))), radius, falloff)
			if wt <= 0 {
				continue
			}
			si, sj := i-rc.i0+1, j-rc.j0+1
			cur := snap[sj*sw+si]
			var d float32
			switch op {
			case brushRaise:
				d = amount * wt
			case brushFlatten:
				d = (amount - cur) * clamp01f32(strength*wt)
			case brushSmooth:
				sum := float32(0)
				for oj := -1; oj <= 1; oj++ {
					for oi := -1; oi <= 1; oi++ {
						sum += snap[(sj+oj)*sw+si+oi]
					}
				}
				d = (sum/9 - cur) * clamp01f32(strength*wt)
			}
			if d == 0 {
				continue
			}
			l.add(i, j, d)
			if i < out.i0 {
				out.i0 = i
			}
			if i > out.i1 {
				out.i1 = i
			}
			if j < out.j0 {
				out.j0 = j
			}
			if j > out.j1 {
				out.j1 = j
			}
		}
	}
	return out
}

func clamp01f32(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// --- heightfield glue ---------------------------------------------------

// terrainSculptState hangs off terrain (one field, so the hook stays small).
type terrainSculptState struct {
	orig         []float32 // float grid as loaded, for ResetTerrainEdits
	sculptDirty  map[chunkKey]sculptArea
	scatterDirty map[chunkKey]int
	hfBodies     map[int]int // CreateBodyHeightfield body -> sample count
	bodiesDirty  bool
	editTick     int
	paint        *terrainPaint
}

// gridEditable is a finite float grid, edited in place.
func (hf *heightField) gridEditable() bool {
	return hf != nil && !hf.proc && hf.q == nil && hf.gw >= 2 && hf.gd >= 2 && len(hf.h) >= hf.gw*hf.gd
}

// sculptLatticeOf returns the editable node grid for t, creating the overlay
// on first use. The error names the command.
//
//   - float heightmap: its own grid, in place
//   - 16-bit heightmap (over 2048²): an overlay on its grid nodes, so raising
//     past the stored range still works
//   - procedural noise: an overlay with one node per mesh cell
func (w *World) sculptLatticeOf(cmd string, t *terrain) (*sculptLattice, error) {
	hf := &t.hf
	if hf.gridEditable() {
		if t.sculpt.orig == nil {
			t.sculpt.orig = append([]float32(nil), hf.h[:hf.gw*hf.gd]...)
		}
		gw := hf.gw
		return &sculptLattice{
			ox: hf.ox, oz: hf.oz,
			dx: hf.worldW / float32(hf.gw-1), dz: hf.worldD / float32(hf.gd-1),
			bounded: true, gw: hf.gw, gd: hf.gd,
			get: func(i, j int) float32 { return hf.h[j*gw+i] },
			add: func(i, j int, d float32) { hf.h[j*gw+i] += d },
		}, nil
	}
	if !hf.proc && hf.q != nil && hf.gw >= 2 && hf.gd >= 2 {
		if hf.edit == nil {
			hf.edit = newSculptOverlay(hf.ox, hf.oz, hf.worldW/float32(hf.gw-1), hf.worldD/float32(hf.gd-1))
		}
		o := hf.edit
		return &sculptLattice{
			ox: o.ox, oz: o.oz, dx: o.dx, dz: o.dz,
			bounded: true, gw: hf.gw, gd: hf.gd,
			get: func(i, j int) float32 { return hf.at(i, j) + o.get(i, j) },
			add: o.add,
		}, nil
	}
	if !hf.proc {
		return nil, fmt.Errorf("%s: terrain %d has no CPU heights to edit", cmd, t.id)
	}
	if hf.edit == nil {
		cw := t.chunkWorld()
		cell := cw / float32(t.chunkSegs(cw))
		hf.edit = newSculptOverlay(0, 0, cell, cell)
	}
	o := hf.edit
	return &sculptLattice{
		dx: o.dx, dz: o.dz,
		get: func(i, j int) float32 {
			return hf.baseAt(float32(i)*o.dx, float32(j)*o.dz) + o.get(i, j)
		},
		add: o.add,
	}, nil
}

func (w *World) sculptTerrainArg(cmd string, a []value.Value, nargs int, sig string) (*terrain, error) {
	if len(a) < nargs {
		return nil, fmt.Errorf("%s%s needs %d arguments", cmd, sig, nargs)
	}
	id := argI(a, 0, 0)
	if id == 0 {
		id = w.curTerrain
	}
	t := w.terrains[id]
	if t == nil {
		return nil, fmt.Errorf("%s: unknown terrain %d", cmd, argI(a, 0, 0))
	}
	return t, nil
}

// markSculpt queues the chunks covering a changed lattice rect for rebuild.
func (w *World) markSculpt(t *terrain, l *sculptLattice, rc sculptRect) {
	if rc.empty() {
		return
	}
	x0, z0 := l.nodePos(rc.i0, rc.j0)
	x1, z1 := l.nodePos(rc.i1, rc.j1)
	// One lattice cell plus the mesh normal stencil on each side.
	cw := t.chunkWorld()
	pad := max32(l.dx, l.dz) + cw/float32(t.chunkSegs(cw)) + 0.25
	ox, oz := w.worldOrigin()
	w.markSculptArea(t, x0-pad-ox, z0-pad-oz, x1+pad-ox, z1+pad-oz)
}

// sculptArea is a scene-space XZ rectangle waiting to be re-meshed.
type sculptArea struct{ x0, z0, x1, z1 float32 }

func (a sculptArea) union(b sculptArea) sculptArea {
	if b.x0 < a.x0 {
		a.x0 = b.x0
	}
	if b.z0 < a.z0 {
		a.z0 = b.z0
	}
	if b.x1 > a.x1 {
		a.x1 = b.x1
	}
	if b.z1 > a.z1 {
		a.z1 = b.z1
	}
	return a
}

func (w *World) markSculptArea(t *terrain, x0, z0, x1, z1 float32) {
	cw := t.chunkWorld()
	a := chunkOf(x0, z0, cw)
	b := chunkOf(x1, z1, cw)
	if t.sculpt.sculptDirty == nil {
		t.sculpt.sculptDirty = map[chunkKey]sculptArea{}
	}
	ar := sculptArea{x0, z0, x1, z1}
	for kz := a.Z; kz <= b.Z; kz++ {
		for kx := a.X; kx <= b.X; kx++ {
			k := chunkKey{kx, kz}
			if prev, ok := t.sculpt.sculptDirty[k]; ok {
				t.sculpt.sculptDirty[k] = prev.union(ar)
			} else {
				t.sculpt.sculptDirty[k] = ar
			}
		}
	}
	t.sculpt.bodiesDirty = true
}

func (w *World) markSculptAll(t *terrain) {
	if t.sculpt.sculptDirty == nil {
		t.sculpt.sculptDirty = map[chunkKey]sculptArea{}
	}
	const huge = float32(1e30)
	for k := range t.ents {
		t.sculpt.sculptDirty[k] = sculptArea{-huge, -huge, huge, huge}
	}
	t.sculpt.bodiesDirty = true
}

// flushTerrainSculpt rebuilds edited chunks once per frame (called from tickTerrain).
func (w *World) flushTerrainSculpt() {
	for _, t := range w.terrains {
		if t == nil {
			continue
		}
		t.sculpt.editTick++
		if len(t.sculpt.sculptDirty) > 0 {
			far := false
			for k, ar := range t.sculpt.sculptDirty {
				if t.ents[k] == 0 {
					far = true // horizon ring / vista ground, not a detail chunk
					continue
				}
				w.rebuildTerrainChunk(t, k, ar)
				if t.sculpt.scatterDirty == nil {
					t.sculpt.scatterDirty = map[chunkKey]int{}
				}
				t.sculpt.scatterDirty[k] = t.sculpt.editTick
			}
			if far {
				// syncHorizon sees a stale ring and vista and rebuilds them on a worker.
				t.horizonHoles = -1
				t.vistaR = -1
			}
			t.sculpt.sculptDirty = nil
		}
		if t.sculpt.bodiesDirty {
			t.sculpt.bodiesDirty = false
			if w.phys3 != nil {
				for bid, n := range t.sculpt.hfBodies {
					if w.ents[bid] == nil {
						delete(t.sculpt.hfBodies, bid)
						continue
					}
					w.addHeightFieldBody(bid, n)
				}
			}
		}
		if len(t.sculpt.scatterDirty) > 0 && len(t.scatters) > 0 {
			cw := t.chunkWorld()
			for k, tick := range t.sculpt.scatterDirty {
				if t.sculpt.editTick-tick < sculptScatterRst {
					continue
				}
				delete(t.sculpt.scatterDirty, k)
				if t.ents[k] == 0 {
					continue
				}
				step, _ := lodDecode(t.built[k])
				w.placeScatter(t, k, float32(k.X)*cw, float32(k.Z)*cw, cw, step)
			}
		} else if len(t.scatters) == 0 {
			t.sculpt.scatterDirty = nil
		}
	}
}

// lodDecode inverts lodCode. A missing code (0) is the full-detail mesh.
func lodDecode(code int) (step int, morph float32) {
	if code <= 0 {
		return 1, 0
	}
	step = code % 8
	if step < 1 {
		step = 1
	}
	return step, float32(code/8) / morphLevels
}

// rebuildTerrainChunk re-samples one resident chunk inside area. The common
// path patches only the vertices under the brush in the existing VBOs, so the
// entity id, material and parent stay put and a drag costs a few hundred
// samples. If the mesh layout does not match (tessellation changed) the chunk
// is rebuilt through attachTerrainChunk.
func (w *World) rebuildTerrainChunk(t *terrain, k chunkKey, area sculptArea) {
	id := t.ents[k]
	if id == 0 {
		return
	}
	cw := t.chunkWorld()
	code, ok := t.built[k]
	if !ok {
		code = lodCode(1, 0)
	}
	step, morph := lodDecode(code)
	x0 := float32(k.X) * cw
	z0 := float32(k.Z) * cw
	segs := t.chunkSegs(cw)
	sample := func(x, z float32) float32 {
		return w.sampleHeight(&t.hf, x, z)
	}
	// A coarse (step 2/4) chunk pulls verts toward a lattice 2*step cells wide.
	grow := cw / float32(segs) * float32(2*step+1)
	area = sculptArea{area.x0 - grow, area.z0 - grow, area.x1 + grow, area.z1 + grow}
	e := w.ents[id]
	if e != nil && e.mesh != nil && patchHeightMesh(e.mesh.GetGeometry(), sample, x0, z0, x0+cw, z0+cw, segs, step, morph, area) {
		if t.simOn != nil {
			delete(t.simOn, k)
		}
		w.syncChunkSim(t, k, id, x0, z0, cw)
		return
	}
	if t.simOn != nil {
		delete(t.simOn, k)
	}
	w.attachTerrainChunk(t, k, x0, z0, cw, segs, step, morph, code)
}

// patchHeightMesh recomputes Y and the normal of every vertex of a
// buildHeightMesh chunk that lies inside area, with the same sampler, LOD
// lattice pull and normal stencil the builder used. Skirt vertices (the last
// 4*(segs+1)) keep their drop. It returns false when g does not look like a
// chunk built with these parameters.
func patchHeightMesh(g interface {
	VBO(gls.AttribType) *gls.VBO
	OperateOnVertices(func(*math32.Vector3) bool)
}, raw func(x, z float32) float32, x0, z0, x1, z1 float32, segs, step int, morph float32, area sculptArea) bool {
	if g == nil {
		return false
	}
	op, on := g.VBO(gls.VertexPosition), g.VBO(gls.VertexNormal)
	if op == nil || on == nil || op.Stride() != 3 || on.Stride() != 3 {
		return false
	}
	pos, nor := *op.Buffer(), *on.Buffer()
	if len(pos) != len(nor) || len(pos)%3 != 0 {
		return false
	}
	if segs < 2 {
		segs = 2
	}
	if step < 1 {
		step = 1
	}
	if step > segs {
		step = segs
	}
	full := segs
	n := full + 1
	nv := len(pos) / 3
	skirt := nv - 4*n
	if skirt <= 0 {
		return false
	}
	const skirtDrop = float32(8)
	const eps = float32(1e-3)
	// First skirt vertex is (x0, z0); the last is (x1, z1).
	if fx, _, fz := fromG3N(pos[skirt*3], pos[skirt*3+1], pos[skirt*3+2]); !near32(fx, x0, eps) || !near32(fz, z0, eps) {
		return false
	}
	if lx, _, lz := fromG3N(pos[(nv-1)*3], pos[(nv-1)*3+1], pos[(nv-1)*3+2]); !near32(lx, x1, eps) || !near32(lz, z1, eps) {
		return false
	}
	if area.x1 < x0 || area.z1 < z0 || area.x0 > x1 || area.z0 > z1 {
		return true
	}
	span := x1 - x0
	if span < 0.001 {
		span = 0.001
	}
	// Same rule as buildHeightMesh: the chunk border is never morphed.
	border := func(x, z float32) bool {
		return x == x0 || x == x1 || z == z0 || z == z1
	}
	sample := func(x, z float32) float32 {
		if border(x, z) {
			return raw(x, z)
		}
		return coarseY(raw, x, z, x0, z0, x1, z1, full, step, morph)
	}
	normCell := span / float32(full)
	if normCell < 0.2 {
		normCell = 0.2
	}
	for v := 0; v < nv; v++ {
		x, _, z := fromG3N(pos[v*3], pos[v*3+1], pos[v*3+2])
		if x < area.x0 || x > area.x1 || z < area.z0 || z > area.z1 {
			continue
		}
		y := sample(x, z)
		if v >= skirt {
			y -= skirtDrop
		}
		_, gy, _ := toG3N(x, y, z)
		pos[v*3+1] = gy
		ns := sample
		if border(x, z) {
			ns = raw
		}
		nx, ny, nz := sharedHeightNormal(ns, x, z, normCell)
		nor[v*3], nor[v*3+1], nor[v*3+2] = nx, ny, nz
	}
	op.Update()
	on.Update()
	// Invalidate the cached bounding box used by frustum culling.
	g.OperateOnVertices(func(*math32.Vector3) bool { return true })
	return true
}

func near32(a, b, eps float32) bool {
	d := a - b
	return d <= eps && d >= -eps
}

// --- painting ---------------------------------------------------------------

type terrainPaint struct {
	img          *image.RGBA
	tex          int
	x0, z0, w, d float32
	dirty        image.Rectangle
}

// blendRect is the world rectangle the blend map covers.
func (t *terrain) blendRect() (x0, z0, w, d float32) {
	if p := t.sculpt.paint; p != nil && t.blendTex == p.tex {
		return p.x0, p.z0, p.w, p.d
	}
	return t.hf.ox, t.hf.oz, t.hf.worldW, t.hf.worldD
}

func (t *terrain) defaultPaintArea() (x0, z0, w, d float32) {
	if t.hf.gridEditable() {
		return t.hf.ox, t.hf.oz, t.hf.worldW, t.hf.worldD
	}
	r := t.radius
	if r < 1 {
		r = 1
	}
	if r > 4 {
		r = 4
	}
	span := t.chunkWorld() * float32(2*r+1)
	return -span / 2, -span / 2, span, span
}

// ensurePaint makes the CPU blend map and its texture. An existing
// SetTerrainBlendMap image is copied in when its file can be read again.
func (w *World) ensurePaint(t *terrain) *terrainPaint {
	if p := t.sculpt.paint; p != nil {
		if t.blendTex == p.tex {
			return p
		}
		// The script bound another blend map; adopt it into the same canvas.
		w.seedPaint(p, t.blendTex)
		p.dirty = p.img.Bounds()
		t.blendTex = p.tex
		return p
	}
	x0, z0, ww, dd := t.defaultPaintArea()
	p := w.newPaint(t, x0, z0, ww, dd, paintDefaultRes)
	return p
}

func (w *World) newPaint(t *terrain, x0, z0, ww, dd float32, res int) *terrainPaint {
	if res < 16 {
		res = 16
	}
	if res > 2048 {
		res = 2048
	}
	img := image.NewRGBA(image.Rect(0, 0, res, res))
	p := &terrainPaint{img: img, x0: x0, z0: z0, w: ww, d: dd}
	prev := t.blendTex
	if t.sculpt.paint != nil && prev == t.sculpt.paint.tex {
		prev = 0
	}
	w.seedPaint(p, prev)
	tex := texture.NewTexture2DFromRGBA(img)
	tex.SetWrapS(gls.CLAMP_TO_EDGE)
	tex.SetWrapT(gls.CLAMP_TO_EDGE)
	tex.SetMinFilter(gls.LINEAR)
	tex.SetMagFilter(gls.LINEAR)
	if t.sculpt.paint != nil {
		if slot := w.texs[t.sculpt.paint.tex]; slot != nil {
			slot.tex = tex
			p.tex = t.sculpt.paint.tex
		}
	}
	if p.tex == 0 {
		p.tex = w.nextTex
		w.nextTex++
		if w.texs == nil {
			w.texs = map[int]*texSlot{}
		}
		w.texs[p.tex] = &texSlot{tex: tex, path: "terrainpaint"}
	}
	t.sculpt.paint = p
	t.blendTex = p.tex
	return p
}

func (w *World) seedPaint(p *terrainPaint, texID int) {
	for i := range p.img.Pix {
		p.img.Pix[i] = 0
	}
	slot := w.texs[texID]
	if slot == nil || slot.path == "" {
		return
	}
	f, err := os.Open(slot.path)
	if err != nil {
		return
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return
	}
	sb := src.Bounds()
	b := p.img.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			sx := sb.Min.X + x*sb.Dx()/b.Dx()
			sy := sb.Min.Y + y*sb.Dy()/b.Dy()
			r, g, bl, a := src.At(sx, sy).RGBA()
			o := p.img.PixOffset(x, y)
			p.img.Pix[o], p.img.Pix[o+1], p.img.Pix[o+2], p.img.Pix[o+3] = uint8(r>>8), uint8(g>>8), uint8(bl>>8), uint8(a>>8)
		}
	}
}

// paintChannel maps PaintTerrain layers (SetTerrainSplat order) to blend map
// channels: 0 sand R, 1 grass G, 2 rock B, 3 snow A. -1 erases.
func paintChannel(layer int) int {
	if layer < 0 {
		return -1
	}
	if layer > 3 {
		return 3
	}
	return layer
}

// paintImage dabs one brush on img. (u, v) and r are in texels. The outer
// texel ring is never painted: the shader clamps outside the area to it.
func paintImage(img *image.RGBA, u, v, r float32, layer int, strength, falloff float32) image.Rectangle {
	b := img.Bounds()
	x0 := int(math.Floor(float64(u - r)))
	x1 := int(math.Ceil(float64(u + r)))
	y0 := int(math.Floor(float64(v - r)))
	y1 := int(math.Ceil(float64(v + r)))
	if x0 < b.Min.X+1 {
		x0 = b.Min.X + 1
	}
	if y0 < b.Min.Y+1 {
		y0 = b.Min.Y + 1
	}
	if x1 > b.Max.X-2 {
		x1 = b.Max.X - 2
	}
	if y1 > b.Max.Y-2 {
		y1 = b.Max.Y - 2
	}
	if x1 < x0 || y1 < y0 {
		return image.Rectangle{}
	}
	ch := paintChannel(layer)
	var touched image.Rectangle
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := float32(x) + 0.5 - u
			dy := float32(y) + 0.5 - v
			wt := brushWeight(float32(math.Sqrt(float64(dx*dx+dy*dy))), r, falloff)
			k := clamp01f32(strength * wt)
			if k <= 0 {
				continue
			}
			o := img.PixOffset(x, y)
			changed := false
			for c := 0; c < 4; c++ {
				target := float32(0)
				if c == ch {
					target = 255
				}
				cur := float32(img.Pix[o+c])
				d := (target - cur) * k
				if d > 0 && d < 1 {
					d = 1
				} else if d < 0 && d > -1 {
					d = -1
				}
				nv := cur + d
				if (d > 0 && nv > target) || (d < 0 && nv < target) {
					nv = target
				}
				if uint8(nv+0.5) != img.Pix[o+c] {
					img.Pix[o+c] = uint8(nv + 0.5)
					changed = true
				}
			}
			if changed {
				touched = touched.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return touched
}

var terrainPaintGL struct {
	once sync.Once
	ok   bool
}

// uploadTerrainPaint pushes the dirty texels of the paint map. It runs inside
// terrainMat.RenderSetup right after the blend map was bound on its unit.
func (w *World) uploadTerrainPaint(t *terrain) {
	p := t.sculpt.paint
	if p == nil || t.blendTex != p.tex || p.dirty.Empty() {
		return
	}
	slot := w.texs[p.tex]
	if slot == nil || slot.tex == nil {
		return
	}
	terrainPaintGL.once.Do(func() { terrainPaintGL.ok = gl.Init() == nil })
	r := p.dirty.Intersect(p.img.Bounds())
	p.dirty = image.Rectangle{}
	if r.Empty() {
		return
	}
	if !terrainPaintGL.ok {
		slot.tex.SetFromRGBA(p.img) // full upload on the next bind
		return
	}
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(p.img.Bounds().Dx()))
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy()),
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p.img.Pix[p.img.PixOffset(r.Min.X, r.Min.Y)]))
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
}

// --- picking ------------------------------------------------------------------

// terrainRayPick marches a ray (scene space) against t and bisects the first
// crossing. ok is false when the ray never goes under the surface.
func (w *World) terrainRayPick(t *terrain, bx, by, bz, dx, dy, dz, maxT float32) (x, y, z float32, ok bool) {
	l := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if l < 1e-6 {
		return 0, 0, 0, false
	}
	dx, dy, dz = dx/l, dy/l, dz/l
	h := func(s float32) float32 {
		return by + dy*s - w.sampleHeight(&t.hf, bx+dx*s, bz+dz*s)
	}
	cw := t.chunkWorld()
	minStep := cw / float32(t.chunkSegs(cw)) * 0.5
	if minStep < 0.05 {
		minStep = 0.05
	}
	prevS, prevH := float32(0), h(0)
	if prevH < 0 {
		return bx, by - prevH, bz, true
	}
	for s := minStep; s <= maxT; {
		cur := h(s)
		if cur <= 0 {
			lo, hi := prevS, s
			for i := 0; i < 16; i++ {
				mid := (lo + hi) * 0.5
				if h(mid) > 0 {
					lo = mid
				} else {
					hi = mid
				}
			}
			x, z = bx+dx*hi, bz+dz*hi
			return x, w.sampleHeight(&t.hf, x, z), z, true
		}
		prevS, prevH = s, cur
		step := s * 0.01
		if step < minStep {
			step = minStep
		}
		if step > 2 {
			step = 2
		}
		s += step
	}
	_ = prevH
	return 0, 0, 0, false
}

// --- commands -------------------------------------------------------------------

func (w *World) terrainSculptCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	brush := func(name, sig string, nargs, op int) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			t, err := w.sculptTerrainArg(name, a, nargs, sig)
			if err != nil {
				return value.Value{}, err
			}
			l, err := w.sculptLatticeOf(name, t)
			if err != nil {
				return value.Value{}, err
			}
			ox, oz := w.worldOrigin()
			x := float32(argN(a, 1, 0)) + ox
			zz := float32(argN(a, 2, 0)) + oz
			r := float32(argN(a, 3, 4))
			if r <= 0 {
				return value.Value{}, fmt.Errorf("%s: radius must be > 0", name)
			}
			var amount, strength, falloff float32
			switch op {
			case brushRaise:
				amount = float32(argN(a, 4, 0.2))
				falloff = float32(argN(a, 5, 0.5))
			case brushFlatten:
				amount = float32(argN(a, 4, 0))
				strength = float32(argN(a, 5, 1))
				falloff = float32(argN(a, 6, 0.5))
			case brushSmooth:
				strength = float32(argN(a, 4, 0.5))
				falloff = float32(argN(a, 5, 0.5))
			}
			rc := l.applyBrush(op, x, zz, r, amount, strength, falloff)
			w.markSculpt(t, l, rc)
			if rc.empty() {
				return value.Num(0), nil
			}
			return value.Num(float64((rc.i1 - rc.i0 + 1) * (rc.j1 - rc.j0 + 1))), nil
		})
	}
	return map[string]cmd{
		"sculptterrain":  brush("SculptTerrain", "(terrain, x, z, radius, amount [, falloff])", 5, brushRaise),
		"flattenterrain": brush("FlattenTerrain", "(terrain, x, z, radius, height [, strength, falloff])", 5, brushFlatten),
		"smoothterrain":  brush("SmoothTerrain", "(terrain, x, z, radius [, strength, falloff])", 4, brushSmooth),
		"setterrainheight": n(func(a []value.Value) (value.Value, error) {
			t, err := w.sculptTerrainArg("SetTerrainHeight", a, 4, "(terrain, gridX, gridZ, height)")
			if err != nil {
				return value.Value{}, err
			}
			l, err := w.sculptLatticeOf("SetTerrainHeight", t)
			if err != nil {
				return value.Value{}, err
			}
			i, j := argI(a, 1, 0), argI(a, 2, 0)
			if l.bounded && (i < 0 || j < 0 || i >= l.gw || j >= l.gd) {
				return value.Value{}, fmt.Errorf("SetTerrainHeight: grid %d,%d outside %dx%d", i, j, l.gw, l.gd)
			}
			l.add(i, j, float32(argN(a, 3, 0))-l.get(i, j))
			w.markSculpt(t, l, sculptRect{i, j, i, j})
			return z()
		}),
		"terraingridheight": n(func(a []value.Value) (value.Value, error) {
			t, err := w.sculptTerrainArg("TerrainGridHeight", a, 3, "(terrain, gridX, gridZ)")
			if err != nil {
				return value.Value{}, err
			}
			l, err := w.sculptLatticeOf("TerrainGridHeight", t)
			if err != nil {
				return value.Value{}, err
			}
			i, j := argI(a, 1, 0), argI(a, 2, 0)
			if l.bounded {
				if i < 0 || j < 0 || i >= l.gw || j >= l.gd {
					return value.Num(0), nil
				}
			}
			return value.Num(float64(l.get(i, j))), nil
		}),
		"terraincellsize": n(func(a []value.Value) (value.Value, error) {
			t, err := w.sculptTerrainArg("TerrainCellSize", a, 1, "(terrain)")
			if err != nil {
				return value.Value{}, err
			}
			l, err := w.sculptLatticeOf("TerrainCellSize", t)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(l.dx)), nil
		}),
		"paintterrain": n(func(a []value.Value) (value.Value, error) {
			t, err := w.sculptTerrainArg("PaintTerrain", a, 5, "(terrain, x, z, radius, layer [, strength, falloff])")
			if err != nil {
				return value.Value{}, err
			}
			r := float32(argN(a, 3, 4))
			if r <= 0 {
				return value.Value{}, fmt.Errorf("PaintTerrain: radius must be > 0")
			}
			layer := argI(a, 4, 1)
			if layer < -1 || layer > 3 {
				return value.Value{}, fmt.Errorf("PaintTerrain: layer %d (0 sand, 1 grass, 2 rock, 3 snow, -1 erase)", layer)
			}
			p := w.ensurePaint(t)
			b := p.img.Bounds()
			sx := float32(b.Dx()) / max32(p.w, 0.001)
			sz := float32(b.Dy()) / max32(p.d, 0.001)
			u := (float32(argN(a, 1, 0)) - p.x0) * sx
			v := (float32(argN(a, 2, 0)) - p.z0) * sz
			touched := paintImage(p.img, u, v, r*(sx+sz)*0.5, layer, float32(argN(a, 5, 0.35)), float32(argN(a, 6, 0.5)))
			p.dirty = p.dirty.Union(touched)
			return value.Num(float64(touched.Dx() * touched.Dy())), nil
		}),
		"setterrainpaintarea": n(func(a []value.Value) (value.Value, error) {
			t, err := w.sculptTerrainArg("SetTerrainPaintArea", a, 5, "(terrain, x0, z0, width, depth [, resolution])")
			if err != nil {
				return value.Value{}, err
			}
			ww, dd := float32(argN(a, 3, 0)), float32(argN(a, 4, 0))
			if ww <= 0 || dd <= 0 {
				return value.Value{}, fmt.Errorf("SetTerrainPaintArea: width and depth must be > 0")
			}
			w.newPaint(t, float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), ww, dd, argI(a, 5, paintDefaultRes))
			return z()
		}),
		"resetterrainedits": n(func(a []value.Value) (value.Value, error) {
			t, err := w.sculptTerrainArg("ResetTerrainEdits", a, 1, "(terrain)")
			if err != nil {
				return value.Value{}, err
			}
			if t.sculpt.orig != nil && len(t.sculpt.orig) <= len(t.hf.h) {
				copy(t.hf.h, t.sculpt.orig)
			}
			t.hf.edit = nil
			if p := t.sculpt.paint; p != nil {
				for i := range p.img.Pix {
					p.img.Pix[i] = 0
				}
				p.dirty = p.img.Bounds()
			}
			w.markSculptAll(t)
			return z()
		}),
		"pickterrain": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if id == 0 {
				id = w.curTerrain
			}
			t := w.terrains[id]
			if t == nil {
				return value.Value{}, fmt.Errorf("PickTerrain: unknown terrain %d", argI(a, 0, 0))
			}
			if w.cam == nil {
				return value.Num(0), nil
			}
			sx, sy := float64(w.mx), float64(w.my)
			if len(a) >= 3 {
				sx, sy = argN(a, 1, sx), argN(a, 2, sy)
			}
			ww, hh := float64(w.scrW), float64(w.scrH)
			if ww < 1 {
				ww = 800
			}
			if hh < 1 {
				hh = 600
			}
			nx := float32(sx/ww)*2 - 1
			ny := 1 - float32(sy/hh)*2
			p0 := math32.Vector3{X: nx, Y: ny, Z: -1}
			p1 := math32.Vector3{X: nx, Y: ny, Z: 1}
			w.cam.Unproject(&p0)
			w.cam.Unproject(&p1)
			bx, by, bz := fromG3N(p0.X, p0.Y, p0.Z)
			ex, ey, ez := fromG3N(p1.X, p1.Y, p1.Z)
			maxT := float32(math.Sqrt(float64((ex-bx)*(ex-bx) + (ey-by)*(ey-by) + (ez-bz)*(ez-bz))))
			if maxT > 3000 {
				maxT = 3000
			}
			x, y, zz, ok := w.terrainRayPick(t, bx, by, bz, ex-bx, ey-by, ez-bz, maxT)
			if !ok {
				return value.Num(0), nil
			}
			w.pickID = 0
			w.pickX, w.pickY, w.pickZ = x, y, zz
			return value.Num(1), nil
		}),
	}
}
