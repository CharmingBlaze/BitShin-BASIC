package runtime

import (
	"strings"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

type heightField struct {
	gw, gd                 int
	worldW, worldD, hscale float32
	ox, oz                 float32
	h                      []float32
	seed                   int
	octaves                int
	proc                   bool
	style                  int // 0 FBM, 1 Terrain-OpenGL value-noise
	freq, disp, power      float32
	seedX, seedZ           float32
	edge                   int // finite map past its border: terrainEdgeZero, Clamp, Mirror
	// A large map keeps 16-bit heights instead of h: qmin + q*qstep.
	// 4096² is 32 MB this way, 64 MB as floats.
	q            []uint16
	qmin, qstep  float32
}

// compactSamples is the size above which a height field goes 16-bit.
const compactSamples = 2048 * 2048

func (hf *heightField) hasData() bool { return len(hf.h) > 0 || len(hf.q) > 0 }

// compact moves h into 16-bit storage. Steps are range/65535: 180 m of
// relief is under 3 mm a step.
func (hf *heightField) compact() {
	if len(hf.h) == 0 {
		return
	}
	lo, hi := hf.h[0], hf.h[0]
	for _, v := range hf.h {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	step := (hi - lo) / 65535
	if step <= 0 {
		step = 1
	}
	q := make([]uint16, len(hf.h))
	for i, v := range hf.h {
		q[i] = uint16((v-lo)/step + 0.5)
	}
	hf.q, hf.qmin, hf.qstep, hf.h = q, lo, step, nil
}

// heights is every sample as a float, for tools that edit the whole map.
func (hf *heightField) heights() []float32 {
	if hf.q == nil {
		return hf.h
	}
	out := make([]float32, len(hf.q))
	for i, v := range hf.q {
		out[i] = hf.qmin + float32(v)*hf.qstep
	}
	return out
}

const (
	terrainEdgeZero   = iota // height 0: an island in a sea
	terrainEdgeClamp         // the border height carries on
	terrainEdgeMirror        // the hills reflect, so the land keeps going
)

// edgeCoord maps a grid coordinate past a finite map back onto it.
func edgeCoord(u float32, n int, mode int) float32 {
	top := float32(n - 1)
	if mode == terrainEdgeClamp {
		if u < 0 {
			return 0
		}
		if u > top {
			return top
		}
		return u
	}
	period := 2 * top
	u = float32(math.Mod(float64(u), float64(period)))
	if u < 0 {
		u += period
	}
	if u > top {
		u = period - u
	}
	return u
}

type terrain struct {
	id         int
	hf         heightField
	chunkVerts int
	radius     int
	ents       map[chunkKey]int
	pending    map[chunkKey]bool
	tex        int
	lod        int
	lightmap   int
	root       int
	// Splat layers (Terrain-OpenGL port). Generator does not own these.
	splat        bool
	sandTex      int
	grassTex     int
	grass2Tex    int
	rockTex      int
	snowTex      int
	rockNrm      int
	grassCover   float32
	waterY       float32
	blend        float32
	fogFalloff   float32
	tessMul      float32
	rock         math32.Color
	snowOn       bool
	snowH        float32
	blendTex     int
	detailPx     float32
	scatters     []scatterLayer
	scatterIDs   map[chunkKey][]int
	scatterAt    map[chunkKey]int // scatterStep each chunk's copies were placed for
	treeMesh     map[chunkKey][]int // 3D tree batches by treeBlock, out past the detail window
	treePending  map[chunkKey]bool  // blocks a worker is building
	treeGen      int                // bumps when trees are released
	treeFar      map[chunkKey]bool  // block was planted with the simplified parts
	built        map[chunkKey]int
	simOn        map[chunkKey]bool
	morphOff     bool
	horizonID    int
	horizonAt    chunkKey
	horizonN     int
	horizonHoles int  // loaded detail chunks the ring was cut around
	horizonBusy  bool // a worker is building the next ring
	vistaID      int      // coarse ground past the ring, out to the fog
	vistaAt      chunkKey // camera chunk the vista was cut for
	vistaR       float32
	shiftSerial  int // bumps on a world shift; a build from before it is dropped
	horizonTrees []int
	cards        []treeCard
	// Tree card setup. cardW/cardH 0 keeps the picture's default size.
	// cardFace defaults on: one card turns toward the camera.
	cardW, cardH float32
	cardCross    bool
	cardFace     bool
}

func (hf *heightField) sample(x, z float32) float32 {
	if hf == nil || hf.gw < 2 || hf.gd < 2 || !hf.hasData() {
		return 0
	}
	u := (x - hf.ox) / hf.worldW * float32(hf.gw-1)
	v := (z - hf.oz) / hf.worldD * float32(hf.gd-1)
	if hf.proc {
		// infinite: worldW is the noise period scale
		u = (x - hf.ox) / hf.worldW * float32(hf.gw-1)
		v = (z - hf.oz) / hf.worldD * float32(hf.gd-1)
	}
	if u < 0 || v < 0 || u > float32(hf.gw-1) || v > float32(hf.gd-1) {
		if !hf.proc {
			if hf.edge == terrainEdgeZero {
				return 0
			}
			u = edgeCoord(u, hf.gw, hf.edge)
			v = edgeCoord(v, hf.gd, hf.edge)
		}
	}
	x0 := int(math.Floor(float64(u)))
	z0 := int(math.Floor(float64(v)))
	tx := u - float32(x0)
	tz := v - float32(z0)
	h00 := hf.at(x0, z0)
	h10 := hf.at(x0+1, z0)
	h01 := hf.at(x0, z0+1)
	h11 := hf.at(x0+1, z0+1)
	return (h00*(1-tx)+h10*tx)*(1-tz) + (h01*(1-tx)+h11*tx)*tz
}

func (hf *heightField) at(x, z int) float32 {
	if hf.proc {
		n := newNoise2(hf.seed)
		fx := float64(x) / float64(hf.gw-1)
		fz := float64(z) / float64(hf.gd-1)
		// wrap into tiled field for finite CreateTerrain; proc chunks use world coords
		return float32(n.fbm(fx*4, fz*4, hf.octaves, 0.5, 2)) * hf.hscale
	}
	if x < 0 {
		x = 0
	}
	if z < 0 {
		z = 0
	}
	if x >= hf.gw {
		x = hf.gw - 1
	}
	if z >= hf.gd {
		z = hf.gd - 1
	}
	if hf.q != nil {
		return hf.qmin + float32(hf.q[z*hf.gw+x])*hf.qstep
	}
	return hf.h[z*hf.gw+x]
}

func (hf *heightField) atWorld(x, z float32) float32 {
	if hf.style == 1 {
		return float32(terrainGLHeight(float64(x), float64(z), hf.octaves, float64(hf.freq), float64(hf.disp), float64(hf.power), float64(hf.seedX), float64(hf.seedZ)))
	}
	if hf.proc {
		n := newNoise2(hf.seed)
		fx := float64(x) * 0.03
		fz := float64(z) * 0.03
		return float32(n.fbm(fx, fz, hf.octaves, 0.5, 2)) * hf.hscale
	}
	return hf.sample(x, z)
}

func loadHeightImage(path string, worldW, worldD, hscale float32) (heightField, error) {
	f, err := os.Open(path)
	if err != nil {
		return heightField{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return heightField{}, err
	}
	b := img.Bounds()
	gw, gd := b.Dx(), b.Dy()
	if gw < 2 || gd < 2 {
		return heightField{}, fmt.Errorf("heightmap %s is %dx%d; it needs at least 2x2", path, gw, gd)
	}
	h := make([]float32, gw*gd)
	// Gray and 16-bit gray maps read their pixels directly. img.At per
	// pixel took seconds on a 4096² map.
	switch m := img.(type) {
	case *image.Gray16:
		for z := 0; z < gd; z++ {
			row := m.Pix[z*m.Stride:]
			for x := 0; x < gw; x++ {
				v := uint16(row[x*2])<<8 | uint16(row[x*2+1])
				h[z*gw+x] = float32(v) / 65535 * hscale
			}
		}
	case *image.Gray:
		for z := 0; z < gd; z++ {
			row := m.Pix[z*m.Stride:]
			for x := 0; x < gw; x++ {
				h[z*gw+x] = float32(row[x]) / 255 * hscale
			}
		}
	default:
		for z := 0; z < gd; z++ {
			for x := 0; x < gw; x++ {
				r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+z).RGBA()
				h[z*gw+x] = float32(r+g+bl) / (3 * 65535) * hscale
			}
		}
	}
	hf := heightField{gw: gw, gd: gd, worldW: worldW, worldD: worldD, hscale: hscale, h: h}
	if gw*gd > compactSamples {
		hf.compact()
	}
	return hf, nil
}

// sharedHeightNormal is the central-difference normal used by heightmap
// terrains (Fayaz CDM / CosmicLearn): N = normalize(hL-hR, 2·cell, hD-hU).
// The same stencil at a given XZ means neighboring chunks and LOD steps
// share the edge normal. Z is flipped to match toG3N vertex positions.
func sharedHeightNormal(sample func(x, z float32) float32, x, z, cell float32) (nx, ny, nz float32) {
	eps := cell
	if eps < 0.2 {
		eps = 0.2
	}
	if eps > 8 {
		eps = 8
	}
	hl := sample(x-eps, z)
	hr := sample(x+eps, z)
	hd := sample(x, z-eps)
	hu := sample(x, z+eps)
	nx = hl - hr
	ny = 2 * eps
	nz = -(hd - hu)
	lenN := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if lenN < 1e-5 {
		return 0, 1, 0
	}
	return nx / lenN, ny / lenN, nz / lenN
}

func gridCoord(a, b float32, i, nQuads int) float32 {
	if nQuads < 1 {
		nQuads = 1
	}
	if i <= 0 {
		return a
	}
	if i >= nQuads {
		return b
	}
	return a + (b-a)*float32(i)/float32(nQuads)
}

// terrainMeshSegs raises a coarse chunk so a cell stays near 0.4 world
// units. Far LOD still thins that count with step inside buildHeightMesh.
// A raised count is a multiple of 4, so step 2 and 4 stay whole and the
// horizon ring can pick a cell that lands on the chunk border.
func terrainMeshSegs(span float32, segs int) int {
	if segs < 2 {
		segs = 2
	}
	const maxCell = float32(0.4)
	if span > maxCell {
		need := int(math.Ceil(float64(span / maxCell)))
		if need > segs {
			segs = (need + 3) / 4 * 4
		}
	}
	if segs > 72 {
		segs = 72
	}
	return segs
}

func buildHeightMesh(sample func(x, z float32) float32, x0, z0, x1, z1 float32, segs, step int, morph float32) *geometry.Geometry {
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
	span := x1 - x0
	if span < 0.001 {
		span = 0.001
	}
	raw := sample
	// The border is never morphed. It is the edge a neighbor, at another
	// step or morph, and the horizon ring all share.
	border := func(x, z float32) bool {
		return x == x0 || x == x1 || z == z0 || z == z1
	}
	sample = func(x, z float32) float32 {
		if border(x, z) {
			return raw(x, z)
		}
		return coarseY(raw, x, z, x0, z0, x1, z1, full, step, morph)
	}
	normCell := span / float32(full)
	if normCell < 0.2 {
		normCell = 0.2
	}
	const skirtDrop = float32(8)
	n := full + 1
	ids := make([]int, n*n)
	for i := range ids {
		ids[i] = -1
	}
	keep := func(ix, iz int) bool {
		if ix == 0 || iz == 0 || ix == full || iz == full {
			return true
		}
		return ix%step == 0 && iz%step == 0
	}
	pos := math32.NewArrayF32(0, 0)
	nor := math32.NewArrayF32(0, 0)
	uvs := math32.NewArrayF32(0, 0)
	idx := math32.NewArrayU32(0, 0)
	nverts := 0
	vert := func(x, z, yoff float32) int {
		y := sample(x, z) - yoff
		gx, gy, gz := toG3N(x, y, z)
		pos.Append(gx, gy, gz)
		ns := sample
		if border(x, z) {
			ns = raw
		}
		nx, ny, nz := sharedHeightNormal(ns, x, z, normCell)
		nor.Append(nx, ny, nz)
		uvs.Append((x-x0)/max32(span, 1)*4, (z-z0)/max32(z1-z0, 1)*4)
		id := nverts
		nverts++
		return id
	}
	for iz := 0; iz < n; iz++ {
		for ix := 0; ix < n; ix++ {
			if !keep(ix, iz) {
				continue
			}
			ids[iz*n+ix] = vert(gridCoord(x0, x1, ix, full), gridCoord(z0, z1, iz, full), 0)
		}
	}
	at := func(ix, iz int) uint32 {
		id := ids[iz*n+ix]
		if id < 0 {
			return 0
		}
		return uint32(id)
	}
	// Coarse interior quads. Border edges stay at full resolution so a
	// step-2/4 chunk shares vertices with a step-1 neighbor (the same job
	// tessellation outer-levels do on GL 4).
	for iz := 0; iz < full; {
		nz := iz + step
		if nz > full {
			nz = full
		}
		for ix := 0; ix < full; {
			nx := ix + step
			if nx > full {
				nx = full
			}
			segsOf := func(a, b, c, d int) int {
				if (a == 0 && c == 0) || (a == full && c == full) || (b == 0 && d == 0) || (b == full && d == full) {
					if c-a < 0 {
						return a - c
					}
					if d-b < 0 {
						return b - d
					}
					if c != a {
						return c - a
					}
					return d - b
				}
				return 1
			}
			sb := segsOf(ix, iz, nx, iz)
			sr := segsOf(nx, iz, nx, nz)
			st := segsOf(nx, nz, ix, nz)
			sl := segsOf(ix, nz, ix, iz)
			if sb == 1 && sr == 1 && st == 1 && sl == 1 {
				a := at(ix, iz)
				b := at(nx, iz)
				c := at(ix, nz)
				d := at(nx, nz)
				idx.Append(a, b, d, a, d, c)
			} else {
				cx := gridCoord(x0, x1, ix, full) + (gridCoord(x0, x1, nx, full)-gridCoord(x0, x1, ix, full))*0.5
				cz := gridCoord(z0, z1, iz, full) + (gridCoord(z0, z1, nz, full)-gridCoord(z0, z1, iz, full))*0.5
				center := uint32(vert(cx, cz, 0))
				pts := make([][2]int, 0, 16)
				push := func(x, z int) {
					if len(pts) > 0 && pts[len(pts)-1][0] == x && pts[len(pts)-1][1] == z {
						return
					}
					pts = append(pts, [2]int{x, z})
				}
				walk := func(x0, z0, x1, z1, nseg int) {
					if nseg < 1 {
						nseg = 1
					}
					for k := 0; k <= nseg; k++ {
						push(x0+(x1-x0)*k/nseg, z0+(z1-z0)*k/nseg)
					}
				}
				walk(ix, iz, nx, iz, sb)
				walk(nx, iz, nx, nz, sr)
				walk(nx, nz, ix, nz, st)
				walk(ix, nz, ix, iz, sl)
				if len(pts) > 1 && pts[0] == pts[len(pts)-1] {
					pts = pts[:len(pts)-1]
				}
				for i := 0; i < len(pts); i++ {
					p0 := pts[i]
					p1 := pts[(i+1)%len(pts)]
					idx.Append(at(p0[0], p0[1]), at(p1[0], p1[1]), center)
				}
			}
			ix = nx
		}
		iz = nz
	}
	// Skirts drop the full-resolution border so a missed sample cannot show sky.
	surface := nverts
	addSkirtEdge := func(points [][2]int, flip bool) {
		if len(points) < 2 {
			return
		}
		base := nverts
		for _, p := range points {
			x := gridCoord(x0, x1, p[0], full)
			z := gridCoord(z0, z1, p[1], full)
			vert(x, z, skirtDrop)
		}
		for i := 0; i < len(points)-1; i++ {
			s0 := at(points[i][0], points[i][1])
			s1 := at(points[i+1][0], points[i+1][1])
			k0 := uint32(base + i)
			k1 := uint32(base + i + 1)
			if k0 >= uint32(nverts) || k1 >= uint32(nverts) {
				continue
			}
			if flip {
				idx.Append(s1, s0, k0, s1, k0, k1)
			} else {
				idx.Append(s0, s1, k1, s0, k1, k0)
			}
		}
	}
	minZ := make([][2]int, 0, n)
	maxZ := make([][2]int, 0, n)
	minX := make([][2]int, 0, n)
	maxX := make([][2]int, 0, n)
	for ix := 0; ix < n; ix++ {
		minZ = append(minZ, [2]int{ix, 0})
		maxZ = append(maxZ, [2]int{ix, full})
	}
	for iz := 0; iz < n; iz++ {
		minX = append(minX, [2]int{0, iz})
		maxX = append(maxX, [2]int{full, iz})
	}
	_ = surface
	addSkirtEdge(minZ, false)
	addSkirtEdge(maxZ, true)
	addSkirtEdge(minX, true)
	addSkirtEdge(maxX, false)
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

func (w *World) ensureTerrain(id int) *terrain {
	if w.terrains == nil {
		w.terrains = map[int]*terrain{}
	}
	return w.terrains[id]
}

func (w *World) terrainHeight(x, z float32) float32 {
	if t := w.terrains[w.curTerrain]; t != nil {
		return w.sampleHeight(&t.hf, x, z)
	}
	for _, t := range w.terrains {
		if t != nil {
			return w.sampleHeight(&t.hf, x, z)
		}
	}
	return 0
}

func (w *World) sampleHeight(hf *heightField, x, z float32) float32 {
	if hf == nil {
		return 0
	}
	ox, oz := w.worldOrigin()
	return hf.atWorld(x+ox, z+oz)
}

func (w *World) tickTerrain() {
	t := w.terrains[w.curTerrain]
	if t == nil || t.radius < 0 {
		return
	}
	if t.ents == nil {
		t.ents = map[chunkKey]int{}
	}
	if t.pending == nil {
		t.pending = map[chunkKey]bool{}
	}
	ox, oz := float32(0), float32(0)
	if w.stream != nil {
		ox, oz = w.stream.ox, w.stream.oz
	} else if w.cam != nil {
		p := worldPos(w.cam.GetNode())
		ox, _, oz = fromG3N(p.X, p.Y, p.Z)
	}
	cw := t.chunkWorld()
	need := map[chunkKey]bool{}
	c := chunkOf(ox, oz, cw)
	for z := c.Z - t.radius; z <= c.Z+t.radius; z++ {
		for x := c.X - t.radius; x <= c.X+t.radius; x++ {
			need[chunkKey{x, z}] = true
		}
	}
	for k, id := range t.ents {
		if !need[k] {
			w.releaseTerrainChunk(t, k, id)
		}
	}
	for k := range need {
		dx := float32(k.X-c.X) * float32(k.X-c.X)
		dz := float32(k.Z-c.Z) * float32(k.Z-c.Z)
		// Distance from the stream origin, not from its chunk, so the weight
		// moves while you walk inside one chunk.
		fx := float32(k.X) + 0.5 - ox/cw
		fz := float32(k.Z) + 0.5 - oz/cw
		step, morph := t.lodBuild(fx*fx + fz*fz)
		morph = morphLevel(morph)
		code := lodCode(step, morph)
		x0 := float32(k.X) * cw
		z0 := float32(k.Z) * cw
		if id := t.ents[k]; id != 0 && t.built != nil && t.built[k] == code {
			w.syncChunkSim(t, k, id, x0, z0, cw)
			continue
		}
		if t.pending[k] {
			continue
		}
		segs := t.chunkSegs(cw)
		// The chunk under the camera, and the ones it is about to enter, are
		// built before the frame draws. A far chunk can arrive a frame later.
		if t.ents[k] == 0 && dx+dz <= 2 {
			w.attachTerrainChunk(t, k, x0, z0, cw, segs, step, morph, code)
			continue
		}
		// Keep the mesh that is already drawing. The replacement is attached
		// when it is ready, so a LOD change does not open a hole.
		t.pending[k] = true
		key := k
		size := cw
		tid := t.id
		lod := code
		serial := t.shiftSerial
		w.ensureJobs().submit(func() {
			// Heights only on the worker. Geometry / VBOs stay on the GL thread.
			w.ensureJobs().enqueueGL(func() {
				tt := w.terrains[tid]
				if tt == nil || tt.shiftSerial != serial {
					return // built in the frame before a world shift
				}
				ax, az := w.terrainAnchor()
				if !tt.chunkInRange(key, ax, az) {
					delete(tt.pending, key)
					return
				}
				w.attachTerrainChunk(tt, key, x0, z0, size, segs, step, morph, lod)
				delete(tt.pending, key)
			})
		})
	}
	w.syncHorizon(t, cw, c)
	w.syncMeshTrees(t, cw, c)
	w.faceTreeCards(t)
	w.cullTerrainChunks(t, cw)
}

// detailLoaded reports a chunk inside the detail window whose mesh is up.
func (t *terrain) detailLoaded(cam, k chunkKey) bool {
	dx, dz := k.X-cam.X, k.Z-cam.Z
	if dx < -t.radius || dx > t.radius || dz < -t.radius || dz > t.radius {
		return false
	}
	return t.ents[k] != 0
}

// inHorizonRing is the coarse ground past the detailed chunks. Games keep
// this ring out to the fog so the hill does not end in open sky.
func inHorizonRing(cam, k chunkKey, radius, n int) bool {
	dx, dz := k.X-cam.X, k.Z-cam.Z
	if dx < 0 {
		dx = -dx
	}
	if dz < 0 {
		dz = -dz
	}
	d := dx
	if dz > d {
		d = dz
	}
	return d > radius && d <= n
}

func (w *World) syncHorizon(t *terrain, cw float32, cam chunkKey) {
	if t == nil || cw < 1 || t.radius < 0 {
		return
	}
	n := w.horizonChunks(t, cw)
	holes := 0
	for z := cam.Z - t.radius; z <= cam.Z+t.radius; z++ {
		for x := cam.X - t.radius; x <= cam.X+t.radius; x++ {
			if t.ents[chunkKey{x, z}] != 0 {
				holes++
			}
		}
	}
	vr := w.vistaReach(t, cw, n)
	vistaStale := vr != t.vistaR || (vr > 0 && t.vistaAt != cam)
	if t.horizonID != 0 && t.horizonAt == cam && t.horizonN == n && t.horizonHoles == holes && !vistaStale {
		return
	}
	if t.horizonBusy {
		return // the next tick after it lands checks again
	}
	loaded := t.loadedDetail(cam)
	swap := func(tt *terrain, m, v *meshArrays) {
		if g := m.geometry(); g != nil {
			id := w.terrainChunkEnt(g, tt)
			if e := w.ents[id]; e != nil {
				e.meshNoCast = true // far ground: past the shadow cascades anyway
			}
			if tt.horizonID != 0 {
				w.retireChunkEntity(tt.horizonID)
			}
			tt.horizonID = id
			tt.horizonAt = cam
			tt.horizonN = n
			tt.horizonHoles = holes
			w.placeHorizonTrees(tt, cw, cam, n)
		}
		if vistaStale {
			// Same frame as the ring, so the hole always matches it.
			if tt.vistaID != 0 {
				w.retireChunkEntity(tt.vistaID)
				tt.vistaID = 0
			}
			if g := v.geometry(); g != nil {
				tt.vistaID = w.terrainChunkEnt(g, tt)
				if e := w.ents[tt.vistaID]; e != nil {
					e.meshNoCast = true // far ground: past the shadow cascades anyway
				}
			}
			tt.vistaAt, tt.vistaR = cam, vr
		}
	}
	build := func() (m, v *meshArrays) {
		m = w.horizonArrays(t, cw, cam, n, loaded)
		if vistaStale && vr > 0 {
			v = w.vistaArrays(t, cw, cam, n, vr)
		}
		return
	}
	// The first ring is built now so frame one has ground to the fog. Later
	// rings take ~25 ms on a 4 km map; a worker builds them while the old
	// ring keeps drawing.
	if t.horizonID == 0 {
		m, v := build()
		swap(t, m, v)
		return
	}
	t.horizonBusy = true
	tid := t.id
	serial := t.shiftSerial
	jobs := w.ensureJobs()
	jobs.submit(func() {
		m, v := build()
		jobs.enqueueGL(func() {
			tt := w.terrains[tid]
			if tt == nil {
				return
			}
			tt.horizonBusy = false
			if tt.shiftSerial != serial {
				return // the old frame's ring; the next tick builds this one
			}
			swap(tt, m, v)
		})
	})
}

// vistaReach is how far the coarse vista goes, 0 when off. It runs past the
// ring only on high and ultra, and only when the fog lets the eye see there.
func (w *World) vistaReach(t *terrain, cw float32, n int) float32 {
	if w.quality.set && w.quality.tier < qualityHigh {
		return 0
	}
	far := float32(0)
	if w.fogMode != 0 {
		far = w.fogFar
	} else if w.cam != nil {
		far = w.cam.Far() * 0.9
	}
	if far > 4000 {
		far = 4000
	}
	ring := float32(n) * cw
	if far < ring+cw {
		return 0
	}
	// Whole chunks, so a small change in fog does not rebuild it.
	return float32(math.Ceil(float64(far/cw))) * cw
}

// vistaArrays is the ground from the ring's edge out to reach, in cells of
// about reach/200 that divide the chunk. The ring's square (n chunks around
// cam) is left open, and the vista's inner edge hangs a skirt into it so
// the T-joins between the two grids cannot open a crack. A finite map
// with a zero edge is clipped to its own bounds: past it is flat sea floor.
func (w *World) vistaArrays(t *terrain, cw float32, cam chunkKey, n int, reach float32) *meshArrays {
	m := int(math.Ceil(float64(reach / cw)))
	want := reach / 200
	if want < 12 {
		want = 12
	}
	per := int(cw/want + 0.5)
	if per < 1 {
		per = 1
	}
	cell := cw / float32(per)
	clip := !t.hf.proc && t.hf.style == 0 && t.hf.edge == terrainEdgeZero && t.hf.worldW > 0
	mx0, mz0 := t.hf.ox, t.hf.oz
	mx1, mz1 := t.hf.ox+t.hf.worldW, t.hf.oz+t.hf.worldD
	if wox, woz := w.worldOrigin(); clip {
		mx0, mx1, mz0, mz1 = mx0-wox, mx1-wox, mz0-woz, mz1-woz
	}
	pos := math32.NewArrayF32(0, 0)
	nor := math32.NewArrayF32(0, 0)
	uvs := math32.NewArrayF32(0, 0)
	idx := math32.NewArrayU32(0, 0)
	type key struct{ x, z int }
	ids := map[key]uint32{}
	var nverts uint32
	height := func(x, z float32) float32 { return w.sampleHeight(&t.hf, x, z) }
	put := func(x, z, drop float32) uint32 {
		y := height(x, z) - drop
		gx, gy, gz := toG3N(x, y, z)
		pos.Append(gx, gy, gz)
		nx, ny, nz := sharedHeightNormal(height, x, z, cell)
		nor.Append(nx, ny, nz)
		uvs.Append(x/max32(cw, 1)*4, z/max32(cw, 1)*4)
		nverts++
		return nverts - 1
	}
	corner := func(ix, iz int) (uint32, float32, float32) {
		x := float32(cam.X-m)*cw + float32(ix)*cell
		z := float32(cam.Z-m)*cw + float32(iz)*cell
		if id, ok := ids[key{ix, iz}]; ok {
			return id, x, z
		}
		id := put(x, z, 0)
		ids[key{ix, iz}] = id
		return id, x, z
	}
	inHole := func(ix, iz int) bool {
		cx := cam.X - m + floorDiv(ix, per)
		cz := cam.Z - m + floorDiv(iz, per)
		// The ring meshes chunks cam-n .. cam+n-1; the hole must be exactly
		// that, or a row of chunks on the + sides is never drawn.
		return cx >= cam.X-n && cx < cam.X+n && cz >= cam.Z-n && cz < cam.Z+n
	}
	drop := cell*0.6 + 2
	skirt := func(ax, az, bx, bz int) {
		a, x0, z0 := corner(ax, az)
		b, x1, z1 := corner(bx, bz)
		a2 := put(x0, z0, drop)
		b2 := put(x1, z1, drop)
		idx.Append(a, b, b2, a, b2, a2) // the terrain material is double-sided
	}
	cells := 2 * m * per
	for iz := 0; iz < cells; iz++ {
		for ix := 0; ix < cells; ix++ {
			if inHole(ix, iz) {
				continue
			}
			x := float32(cam.X-m)*cw + float32(ix)*cell
			z := float32(cam.Z-m)*cw + float32(iz)*cell
			if clip && (x+cell < mx0 || x > mx1 || z+cell < mz0 || z > mz1) {
				continue
			}
			a, _, _ := corner(ix, iz)
			b, _, _ := corner(ix+1, iz)
			c, _, _ := corner(ix, iz+1)
			d, _, _ := corner(ix+1, iz+1)
			idx.Append(a, b, d, a, d, c)
			if inHole(ix-1, iz) {
				skirt(ix, iz, ix, iz+1)
			}
			if inHole(ix+1, iz) {
				skirt(ix+1, iz, ix+1, iz+1)
			}
			if inHole(ix, iz-1) {
				skirt(ix, iz, ix+1, iz)
			}
			if inHole(ix, iz+1) {
				skirt(ix, iz+1, ix+1, iz+1)
			}
		}
	}
	if nverts < 3 || idx.Size() < 3 {
		return nil
	}
	return &meshArrays{pos: &pos, nor: &nor, uvs: &uvs, idx: &idx}
}

// horizonCells is how many detail cells one horizon cell spans. It divides
// the chunk, so the ring's inner edge lands on the detailed tiles' own
// border vertices instead of crossing the hill on another grid.
func horizonCells(cw float32, segs, chunks int) int {
	if segs < 1 {
		return 1
	}
	cell := cw / float32(segs)
	want := cw / 10
	if want < 1.2 {
		want = 1.2
	}
	if want > 2.8 {
		want = 2.8
	}
	best, bestErr := segs, float32(math.MaxFloat32)
	for d := 1; d <= segs; d++ {
		if segs%d != 0 || chunks*(segs/d) > 240 {
			continue
		}
		e := cell*float32(d) - want
		if e < 0 {
			e = -e
		}
		if e < bestErr {
			best, bestErr = d, e
		}
	}
	return best
}

// horizonChunks is how many chunks the coarse ring reaches from the camera.
func (w *World) horizonChunks(t *terrain, cw float32) int {
	reach := float32(360)
	if w.fogMode != 0 && w.fogFar > reach {
		reach = w.fogFar
	}
	if reach > 480 {
		reach = 480
	}
	n := int(math.Ceil(float64(reach / cw)))
	if n < t.radius+1 {
		n = t.radius + 1
	}
	if n > 28 {
		n = 28
	}
	return n
}

// groundBand is the horizon ring's cell size and reach, for meshes that
// lie on the ground (water) and should be no coarser than it.
func (w *World) groundBand() (cell, reach float32, ok bool) {
	t := w.terrains[w.curTerrain]
	if t == nil || t.radius < 0 {
		return 0, 0, false
	}
	cw := t.chunkWorld()
	if cw < 1 {
		return 0, 0, false
	}
	n := w.horizonChunks(t, cw)
	segs := t.chunkSegs(cw)
	d := horizonCells(cw, segs, 2*n)
	return cw / float32(segs/d), float32(n) * cw, true
}

func (w *World) buildHorizonMesh(t *terrain, cw float32, cam chunkKey, n int) *geometry.Geometry {
	return w.horizonArrays(t, cw, cam, n, t.loadedDetail(cam)).geometry()
}

// meshArrays is a mesh's CPU side. A worker fills it; the GL thread wraps it.
type meshArrays struct {
	pos, nor, uvs *math32.ArrayF32
	idx           *math32.ArrayU32
}

func (m *meshArrays) geometry() *geometry.Geometry {
	if m == nil {
		return nil
	}
	g := geometry.NewGeometry()
	g.SetIndices(*m.idx)
	g.AddVBO(gls.NewVBO(*m.pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(*m.nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(*m.uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

// loadedDetail snapshots the detail chunks that are drawing, so a worker
// can cut the ring around them without reading t.ents.
func (t *terrain) loadedDetail(cam chunkKey) map[chunkKey]bool {
	out := map[chunkKey]bool{}
	for z := cam.Z - t.radius; z <= cam.Z+t.radius; z++ {
		for x := cam.X - t.radius; x <= cam.X+t.radius; x++ {
			if k := (chunkKey{x, z}); t.ents[k] != 0 {
				out[k] = true
			}
		}
	}
	return out
}

// horizonArrays is safe on a worker: it reads the height field and the
// loaded snapshot only.
func (w *World) horizonArrays(t *terrain, cw float32, cam chunkKey, n int, loaded map[chunkKey]bool) *meshArrays {
	// A chunk-sized quad reads as a seam and a flat hill. Keep cells near
	// the size of the ground you can still see through the fog.
	segs := t.chunkSegs(cw)
	d := horizonCells(cw, segs, 2*n)
	per := segs / d
	normCell := cw / float32(segs)
	if normCell < 0.2 {
		normCell = 0.2
	}
	quads := 2 * n * per
	// Cut only where a detail chunk is already drawing. A chunk still on its
	// way in keeps coarse ground instead of a hole for a frame.
	hole := func(ix, iz int) bool {
		return loaded[chunkKey{cam.X - n + ix/per, cam.Z - n + iz/per}]
	}
	type hz struct{ x, z int }
	ids := map[hz]uint32{}
	pos := math32.NewArrayF32(0, 0)
	nor := math32.NewArrayF32(0, 0)
	uvs := math32.NewArrayF32(0, 0)
	idx := math32.NewArrayU32(0, 0)
	var nverts uint32
	coord := func(c0, i int) float32 {
		// Same arithmetic as a chunk's border (attachTerrainChunk +
		// buildHeightMesh), so the shared verts match to the bit.
		k := c0 + i/per
		a := float32(k) * cw
		return gridCoord(a, a+cw, (i%per)*d, segs)
	}
	corner := func(ix, iz int) uint32 {
		if id, ok := ids[hz{ix, iz}]; ok {
			return id
		}
		x := coord(cam.X-n, ix)
		z := coord(cam.Z-n, iz)
		y := w.sampleHeight(&t.hf, x, z)
		gx, gy, gz := toG3N(x, y, z)
		pos.Append(gx, gy, gz)
		nx, ny, nz := sharedHeightNormal(func(sx, sz float32) float32 {
			return w.sampleHeight(&t.hf, sx, sz)
		}, x, z, normCell)
		nor.Append(nx, ny, nz)
		// Chunks repeat a material texture 4x per chunk.
		uvs.Append(x/max32(cw, 1)*4, z/max32(cw, 1)*4)
		id := nverts
		nverts++
		ids[hz{ix, iz}] = id
		return id
	}
	for iz := 0; iz < quads; iz++ {
		for ix := 0; ix < quads; ix++ {
			if hole(ix, iz) {
				continue
			}
			a := corner(ix, iz)
			b := corner(ix+1, iz)
			c := corner(ix, iz+1)
			d := corner(ix+1, iz+1)
			idx.Append(a, b, d, a, d, c)
		}
	}
	if nverts < 3 || idx.Size() < 3 {
		return nil
	}
	return &meshArrays{pos: &pos, nor: &nor, uvs: &uvs, idx: &idx}
}

func (w *World) attachTerrainChunk(t *terrain, key chunkKey, x0, z0, size float32, segs, step int, morph float32, lod int) {
	if t == nil {
		return
	}
	g := buildHeightMesh(func(x, z float32) float32 {
		return w.sampleHeight(&t.hf, x, z)
	}, x0, z0, x0+size, z0+size, segs, step, morph)
	id := w.terrainChunkEnt(g, t)
	if prev := t.ents[key]; prev != 0 && prev != id {
		w.retireChunkEntity(prev)
	}
	t.ents[key] = id
	if t.built == nil {
		t.built = map[chunkKey]int{}
	}
	t.built[key] = lod
	// A LOD or morph rebuild only changes the ground. Trees and props keep
	// their copies, so the forest does not pop while you walk.
	if at, ok := t.scatterAt[key]; !ok || at != t.scatterStep(step) {
		w.placeScatter(t, key, x0, z0, size, step)
	}
	w.syncChunkSim(t, key, id, x0, z0, size)
}

func (w *World) cullTerrainChunks(t *terrain, cw float32) {
	if t == nil || w.cam == nil {
		return
	}
	var view, proj, vp math32.Matrix4
	w.cam.ViewMatrix(&view)
	w.cam.ProjMatrix(&proj)
	vp.MultiplyMatrices(&proj, &view)
	ymin, ymax := float32(-4), t.hf.hscale
	if ymax < 8 {
		ymax = 32
	}
	ymax += 8
	for k, id := range t.ents {
		e := w.ents[id]
		if e == nil || e.node == nil {
			continue
		}
		x0 := float32(k.X) * cw
		z0 := float32(k.Z) * cw
		e.node.GetNode().SetVisible(aabbInClip(&vp, x0, ymin, z0, x0+cw, ymax, z0+cw))
	}
}

func aabbInClip(vp *math32.Matrix4, x0, y0, z0, x1, y1, z1 float32) bool {
	if vp == nil {
		return true
	}
	xs := [2]float32{x0, x1}
	ys := [2]float32{y0, y1}
	zs := [2]float32{z0, z1}
	and := uint32(0x3f)
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			for k := 0; k < 2; k++ {
				gx, gy, gz := toG3N(xs[i], ys[j], zs[k])
				cx := vp[0]*gx + vp[4]*gy + vp[8]*gz + vp[12]
				cy := vp[1]*gx + vp[5]*gy + vp[9]*gz + vp[13]
				cz := vp[2]*gx + vp[6]*gy + vp[10]*gz + vp[14]
				cw := vp[3]*gx + vp[7]*gy + vp[11]*gz + vp[15]
				bits := uint32(0)
				if cx < -cw {
					bits |= 1
				}
				if cx > cw {
					bits |= 2
				}
				if cy < -cw {
					bits |= 4
				}
				if cy > cw {
					bits |= 8
				}
				if cz < -cw {
					bits |= 16
				}
				if cz > cw {
					bits |= 32
				}
				and &= bits
			}
		}
	}
	return and == 0
}

func (w *World) terrainCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createterrain": need(func(a []value.Value) (value.Value, error) {
			ww := float32(argN(a, 1, 64))
			dd := float32(argN(a, 2, 64))
			hs := float32(argN(a, 3, 12))
			if len(a) > 0 && a[0].Kind == value.KindNum {
				if hm := w.hmaps[argI(a, 0, 0)]; hm != nil {
					return value.Num(float64(w.addTerrain(heightFieldFromMap(hm, ww, dd, hs), 17, 2))), nil
				}
			}
			file := argS(a, 0)
			var hf heightField
			if file != "" && file != "default" {
				path, err := w.openPath(file)
				if err != nil {
					return value.Value{}, err
				}
				hf, err = loadHeightImage(path, ww, dd, hs)
				if err != nil {
					return value.Value{}, err
				}
			} else {
				hf = heightField{gw: 65, gd: 65, worldW: ww, worldD: dd, hscale: hs, proc: true, seed: 1, octaves: 5, ox: -ww / 2, oz: -dd / 2}
			}
			return value.Num(float64(w.addTerrain(hf, 17, 2))), nil
		}),
		"loadheightmap": n(func(a []value.Value) (value.Value, error) {
			path, err := w.openPath(argS(a, 0))
			if err != nil {
				return value.Num(0), err
			}
			hf, err := loadHeightImage(path, float32(argN(a, 1, 64)), float32(argN(a, 2, 64)), float32(argN(a, 3, 12)))
			if err != nil {
				return value.Num(0), err
			}
			return value.Num(float64(w.addTerrain(hf, 17, 1))), nil
		}),
		"createprocterrain": need(func(a []value.Value) (value.Value, error) {
			seed := argI(a, 0, 1)
			chunk := argI(a, 1, 17)
			oct := argI(a, 2, 5)
			hs := float32(argN(a, 3, 10))
			scale := float32(argN(a, 4, 48))
			hf := heightField{gw: chunk, gd: chunk, worldW: scale, worldD: scale, hscale: hs, proc: true, seed: seed, octaves: oct}
			return value.Num(float64(w.addTerrain(hf, chunk, 3))), nil
		}),
		"terrainheight": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.terrainHeight(float32(argN(a, 0, 0)), float32(argN(a, 1, 0))))), nil
		}),
		"setterraintexture": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return z()
			}
			t.tex = argI(a, 1, 0)
			for _, id := range t.ents {
				if e := w.ents[id]; e != nil && e.mat != nil {
					if tex := w.texs[t.tex]; tex != nil {
						e.mat.AddTexture(tex.tex)
					}
				}
			}
			return z()
		}),
		"setterrainlightmap": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return z()
			}
			t.lightmap = argI(a, 1, 0)
			return z()
		}),
		"setterrainstreamradius": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return z()
			}
			if len(a) >= 2 {
				t.radius = argI(a, 1, 2)
			} else {
				t.radius = argI(a, 0, 2)
			}
			return value.Num(float64(t.radius)), nil
		}),
		"setterrainlod": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t != nil {
				t.lod = argI(a, 1, argI(a, 0, 1))
			}
			return z()
		}),
		"getterrainstreamradius": n(func(a []value.Value) (value.Value, error) {
			t := w.terrains[argI(a, 0, w.curTerrain)]
			if t == nil {
				t = w.terrains[w.curTerrain]
			}
			if t == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(t.radius)), nil
		}),
		"getterrainlod": n(func(a []value.Value) (value.Value, error) {
			t := w.terrains[argI(a, 0, w.curTerrain)]
			if t == nil {
				t = w.terrains[w.curTerrain]
			}
			if t == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(t.lod)), nil
		}),
		"terrainchunkcount": n(func(a []value.Value) (value.Value, error) {
			t := w.terrains[w.curTerrain]
			if t == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(len(t.ents))), nil
		}),
		"createterraingl": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.addTerrainGL(a))), nil
		}),
		"applyterrainsplat": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return z()
			}
			w.ensureSplatTextures(t)
			w.dirtyTerrain(t)
			w.seedTerrainChunks(t)
			return z()
		}),
		"setterrainsplat": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return z()
			}
			off := 0
			if w.terrains[argI(a, 0, 0)] != nil && len(a) >= 5 {
				off = 1
			}
			t.sandTex = argI(a, off, 0)
			t.grassTex = argI(a, off+1, 0)
			t.rockTex = argI(a, off+2, 0)
			t.snowTex = argI(a, off+3, 0)
			if len(a) > off+4 {
				t.grass2Tex = argI(a, off+4, 0)
			}
			if len(a) > off+5 {
				t.rockNrm = argI(a, off+5, 0)
			}
			t.splat = true
			w.rebindTerrainMats(t)
			return z()
		}),
		"setterrainoctaves": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.hf.octaves = argI(a, terrainArgOff(w, a), 8)
				w.dirtyTerrain(t)
			}
			return z()
		}),
		"setterrainfreq": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.hf.freq = float32(argN(a, terrainArgOff(w, a), 0.035))
				w.dirtyTerrain(t)
			}
			return z()
		}),
		"setterrainfrequency": n(func(a []value.Value) (value.Value, error) {
			return w.Call("setterrainfreq", a)
		}),
		"setterraindispfactor": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.hf.disp = float32(argN(a, terrainArgOff(w, a), 3.2))
				w.dirtyTerrain(t)
			}
			return z()
		}),
		"setterrainpower": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.hf.power = float32(argN(a, terrainArgOff(w, a), 2))
				w.dirtyTerrain(t)
			}
			return z()
		}),
		"setterraingrasscoverage": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.grassCover = float32(argN(a, terrainArgOff(w, a), 0.65))
			}
			return z()
		}),
		"setterrainrockcolor": n(func(a []value.Value) (value.Value, error) {
			// (r, g, b) is three arguments; only a fourth makes the first a
			// terrain. A red of 1 used to read as terrain 1.
			t, off := w.terrainArgsN(a, 3)
			if t != nil {
				t.rock = *rgb(argN(a, off, 180), argN(a, off+1, 158), argN(a, off+2, 112))
			}
			return z()
		}),
		"setterrainfogfalloff": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.fogFalloff = float32(argN(a, terrainArgOff(w, a), 0.012))
			}
			return z()
		}),
		"setterrainwaterheight": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.waterY = float32(argN(a, terrainArgOff(w, a), 2.2))
			}
			return z()
		}),
		"setterrainedge": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			mode := terrainEdgeZero
			switch strings.ToLower(strings.TrimSpace(argS(a, terrainArgOff(w, a)))) {
			case "clamp":
				mode = terrainEdgeClamp
			case "mirror", "reflect":
				mode = terrainEdgeMirror
			}
			if t.hf.edge != mode {
				t.hf.edge = mode
				w.dirtyTerrain(t)
				if wb := w.currentWater(); wb != nil && wb.follow {
					w.rebuildWaterMesh(wb)
				}
			}
			return z()
		}),
		"setterrainblend": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.blend = float32(argN(a, terrainArgOff(w, a), 1.4))
			}
			return z()
		}),
		"setterrainsnow": n(func(a []value.Value) (value.Value, error) {
			// Two arguments are always (on, y). True is 1, the first terrain's
			// handle, so guessing from the value read (True, 140) as
			// (terrain, on) and dropped the height.
			t := w.terrains[w.curTerrain]
			off := 0
			if len(a) >= 3 {
				t = w.pickTerrain(a)
				off = terrainArgOff(w, a)
			}
			if t != nil {
				t.snowOn = argI(a, off, 1) != 0
				if len(a) > off+1 {
					t.snowH = float32(argN(a, off+1, 9))
				}
			}
			return z()
		}),
		"setterraintessmultiplier": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.tessMul = float32(argN(a, terrainArgOff(w, a), 1))
				if t.tessMul < 0.25 {
					t.tessMul = 0.25
				}
				w.dirtyTerrain(t)
			}
			return z()
		}),
		"getterrainoctaves": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(t.hf.octaves)), nil
		}),
		"getterrainfreq": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(t.hf.freq)), nil
		}),
		"getterraindispfactor": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(t.hf.disp)), nil
		}),
		"getterraingrasscoverage": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(t.grassCover)), nil
		}),
		"terrainslope": n(func(a []value.Value) (value.Value, error) {
			x, z := float32(argN(a, 0, 0)), float32(argN(a, 1, 0))
			t := w.terrains[w.curTerrain]
			if t == nil {
				return value.Num(0), nil
			}
			var nx, ny, nz float64
			if t.hf.style == 1 {
				nx, ny, nz = terrainGLNormal(float64(x), float64(z), 0.4, t.hf.octaves, float64(t.hf.freq), float64(t.hf.disp), float64(t.hf.power), float64(t.hf.seedX), float64(t.hf.seedZ))
			} else {
				eps := 0.4
				hl := float64(t.hf.atWorld(x-float32(eps), z))
				hr := float64(t.hf.atWorld(x+float32(eps), z))
				hd := float64(t.hf.atWorld(x, z-float32(eps)))
				hu := float64(t.hf.atWorld(x, z+float32(eps)))
				nx, ny, nz = 0, 1, 0
				tx, ty, tz := 2*eps, hr-hl, 0.0
				bx, by, bz := 0.0, hu-hd, 2*eps
				nx = ty*bz - tz*by
				ny = tz*bx - tx*bz
				nz = tx*by - ty*bx
				lenN := math.Sqrt(nx*nx + ny*ny + nz*nz)
				if lenN > 1e-8 {
					nx, ny, nz = nx/lenN, ny/lenN, nz/lenN
				}
			}
			_ = nx
			_ = nz
			return value.Num(ny), nil
		}),
		"baketerrainnav": n(func(a []value.Value) (value.Value, error) {
			t := w.ensureTerrain(argI(a, 0, w.curTerrain))
			if t == nil {
				return value.Num(0), fmt.Errorf("BakeTerrainNav: no terrain")
			}
			if w.navMaxSlope <= 0 {
				w.navMaxSlope = 45
			}
			// bake from first loaded chunk mesh (triangles, not AABB)
			var meshID int
			for _, id := range t.ents {
				meshID = id
				break
			}
			if meshID == 0 {
				return value.Num(0), fmt.Errorf("BakeTerrainNav: no loaded chunks yet")
			}
			nid := w.takeHandle(&w.freeNavs, &w.nextNav)
			if w.navs == nil {
				w.navs = map[int]*navMesh{}
			}
			nm := &navMesh{meshID: meshID}
			for _, id := range t.ents {
				if id != meshID {
					nm.walkExtra = append(nm.walkExtra, id)
				}
			}
			w.navs[nid] = nm
			w.curNav = nid
			if err := w.bakeNav(nm); err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(nid)), nil
		}),
	}
}

func (w *World) addTerrain(hf heightField, chunkVerts, radius int) int {
	id := w.takeHandle(&w.freeTerrains, &w.nextTerrain)
	if w.terrains == nil {
		w.terrains = map[int]*terrain{}
	}
	root := w.addEntity(&Entity{node: core.NewNode()}, 0)
	if e := w.ents[root]; e != nil {
		e.name = "terrain_root"
	}
	if chunkVerts < 4 {
		chunkVerts = 17
	}
	w.terrains[id] = &terrain{
		id: id, hf: hf, chunkVerts: chunkVerts, radius: radius,
		ents: map[chunkKey]int{}, pending: map[chunkKey]bool{}, root: root,
		lod: 1, grassCover: 0.65, waterY: 2.2, blend: 1.4, cardFace: true,
		fogFalloff: 0, tessMul: 1, splat: true,
		rock: math32.Color{0.55, 0.48, 0.40}, snowH: 19,
	}
	if w.bubble != nil && w.bubble.morphOff {
		w.terrains[id].morphOff = true
	}
	w.curTerrain = id
	w.applyTerrainQuality(w.terrains[id], false)
	w.ensureSplatTextures(w.terrains[id])
	// Origin ring on the GL thread so the first Flip is not an empty plane.
	w.seedTerrainChunks(w.terrains[id])
	return id
}

// chunkSegs is the quad count across one detailed chunk.
func (t *terrain) chunkSegs(cw float32) int {
	segs := t.chunkVerts
	if t.tessMul > 1 {
		segs = int(float32(segs) * t.tessMul)
	}
	if segs < 4 {
		segs = 17
	}
	return terrainMeshSegs(cw, segs)
}

func (t *terrain) chunkWorld() float32 {
	if t == nil {
		return 32
	}
	cw := t.hf.worldW
	if t.chunkVerts > 1 && t.hf.gw > 1 {
		cw = t.hf.worldW / float32(t.hf.gw-1) * float32(t.chunkVerts-1)
	}
	if cw < 8 {
		cw = 32
	}
	return cw
}

func (w *World) seedTerrainChunks(t *terrain) {
	if t == nil {
		return
	}
	cw := t.chunkWorld()
	ring := t.radius
	if ring > 2 {
		ring = 2
	}
	if ring < 1 {
		ring = 1
	}
	segs := t.chunkSegs(cw)
	for z := -ring; z <= ring; z++ {
		for x := -ring; x <= ring; x++ {
			k := chunkKey{x, z}
			if t.ents[k] != 0 {
				continue
			}
			x0 := float32(x) * cw
			z0 := float32(z) * cw
			g := buildHeightMesh(func(x, z float32) float32 {
				return w.sampleHeight(&t.hf, x, z)
			}, x0, z0, x0+cw, z0+cw, segs, 1, 0)
			t.ents[k] = w.terrainChunkEnt(g, t)
			if t.built == nil {
				t.built = map[chunkKey]int{}
			}
			t.built[k] = lodCode(1, 0)
			w.placeScatter(t, k, x0, z0, cw, 1)
		}
	}
}

func (w *World) addTerrainGL(a []value.Value) int {
	seed := argI(a, 0, 0)
	chunk := argI(a, 1, 25)
	radius := argI(a, 2, 3)
	disp := float32(argN(a, 3, 3.2))
	freq := float32(argN(a, 4, 0.035))
	oct := argI(a, 5, 8)
	scale := float32(argN(a, 6, 40))
	if chunk < 8 {
		chunk = 25
	}
	if disp < 8 {
		disp = 16
	}
	if freq > 0 && freq < 0.008 {
		freq = 0.04
	}
	hf := heightField{
		gw: chunk, gd: chunk, worldW: scale, worldD: scale,
		hscale: disp, proc: true, style: 1, seed: seed, octaves: oct,
		freq: freq, disp: disp, power: 1.25,
		seedX: float32(seed) * 0.01, seedZ: float32(seed) * 0.017,
	}
	id := w.addTerrain(hf, chunk, radius)
	t := w.terrains[id]
	t.splat = true
	t.snowOn = true
	t.snowH = 8.5
	t.waterY = 2.15
	t.grassCover = 0.65
	w.ensureSplatTextures(t)
	w.dirtyTerrain(t)
	w.seedTerrainChunks(t)
	return id
}

// terrainArgsN reads an optional leading terrain for a command whose own
// arguments number base: the first argument is a terrain only when there
// are more than base. Otherwise it is the current terrain.
func (w *World) terrainArgsN(a []value.Value, base int) (*terrain, int) {
	if len(a) > base {
		if t := w.terrains[argI(a, 0, 0)]; t != nil {
			return t, 1
		}
	}
	return w.terrains[w.curTerrain], 0
}

func terrainArgOff(w *World, a []value.Value) int {
	if w != nil && len(a) >= 2 && w.terrains[argI(a, 0, 0)] != nil {
		return 1
	}
	return 0
}

func (w *World) pickTerrain(a []value.Value) *terrain {
	if t := w.terrains[argI(a, 0, 0)]; t != nil {
		return t
	}
	return w.ensureTerrain(w.curTerrain)
}

func (w *World) dirtyTerrain(t *terrain) {
	if t == nil {
		return
	}
	for k, id := range t.ents {
		w.freeEntityID(id)
		w.freeScatter(t, k)
		delete(t.ents, k)
	}
	t.pending = map[chunkKey]bool{}
	t.vistaR = -1 // heights changed; the vista is rebuilt with the next ring
}

func (w *World) terrainChunkEnt(g *geometry.Geometry, tt *terrain) int {
	if tt != nil && tt.splat {
		mat := w.newMat()
		mat.SetShader("mbterrain")
		mat.SetColor(&math32.Color{1, 1, 1})
		mat.SetShininess(4)
		mat.SetSpecularColor(&math32.Color{0.04, 0.04, 0.04})
		mat.SetSide(material.SideDouble)
		mesh := graphic.NewMesh(g, &terrainMat{Standard: mat, w: w, t: tt})
		id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat}, tt.root)
		if e := w.ents[id]; e != nil {
			e.name = "terrain"
		}
		return id
	}
	id := w.meshEnt(g, tt.root)
	if e := w.ents[id]; e != nil {
		e.name = "terrain"
		if tex := w.texs[tt.tex]; tex != nil && e.mat != nil {
			e.mat.AddTexture(tex.tex)
		}
		if lm := w.texs[tt.lightmap]; lm != nil && e.mat != nil {
			e.mat.AddTexture(lm.tex)
		}
		e.mat.SetColor(&math32.Color{0.28, 0.42, 0.22})
	}
	return id
}

func (w *World) rebindTerrainMats(t *terrain) {
	if t == nil {
		return
	}
	for _, id := range t.ents {
		if e := w.ents[id]; e != nil && e.mat != nil && t.splat {
			e.mat.SetShader("mbterrain")
		}
	}
}

type terrainMat struct {
	*material.Standard
	w *World
	t *terrain
}

func (m *terrainMat) GetMaterial() *material.Material { return m.Standard.GetMaterial() }
func (m *terrainMat) Dispose()                        { m.Standard.Dispose() }

func (m *terrainMat) RenderSetup(gs *gls.GLS) {
	m.Standard.RenderSetup(gs)
	if m.w == nil || m.t == nil || gs == nil {
		return
	}
	m.w.applyUserUniforms(gs)
	m.w.bindShadowUniforms(gs)
	m.w.bindFogUniforms(gs)
	t := m.t
	setUni1i(gs, "TerrainSplatOn", bool01(t.splat))
	setUni1i(gs, "TerrainCheap", bool01(m.w.quality.set && m.w.quality.tier == qualityLow))
	setUni1f(gs, "TerrainWaterY", t.waterY)
	setUni1f(gs, "TerrainBlend", t.blend)
	setUni1f(gs, "TerrainGrassCover", t.grassCover)
	setUni1f(gs, "TerrainSnowH", t.snowH)
	setUni1i(gs, "TerrainSnowOn", bool01(t.snowOn))
	setUni3f(gs, "TerrainRockColor", t.rock.R, t.rock.G, t.rock.B)
	setUni1f(gs, "TerrainFogFalloff", t.fogFalloff)
	sx, sy, sz := m.w.waterSunDir()
	setUni3f(gs, "TerrainSunDir", sx, sy, sz)
	cr, cg, cb := float32(1), float32(0.90), float32(0.74)
	if ls := m.w.scanLights(); ls.sunSet {
		cr, cg, cb = ls.sun.R, ls.sun.G, ls.sun.B
	}
	setUni3f(gs, "TerrainSunColor", cr, cg, cb)
	if m.w.cam != nil {
		cp := worldPos(m.w.cam.GetNode())
		setUni3f(gs, "CamWorldPos", cp.X, cp.Y, cp.Z)
	}
	bind := func(id, unit int, sampler, flag string) {
		has := 0
		if slot := m.w.texs[id]; slot != nil && slot.tex != nil {
			slot.tex.RenderSetup(gs, unit, unit)
			setUni1i(gs, sampler, unit)
			has = 1
		}
		setUni1i(gs, flag, has)
	}
	// Units 1-7. 8 and 9 are the shadow maps, 12 and 13 spot cookies, 15
	// caustics: sand and grass used to sit on the shadow units, so the
	// terrain read grass as depth and never showed a shadow.
	bind(t.sandTex, 1, "TerrainSand", "TerrainHasSand")
	bind(t.grassTex, 2, "TerrainGrass", "TerrainHasGrass")
	bind(t.grass2Tex, 3, "TerrainGrass2", "TerrainHasGrass2")
	bind(t.rockTex, 4, "TerrainRock", "TerrainHasRock")
	bind(t.snowTex, 5, "TerrainSnow", "TerrainHasSnow")
	bind(t.rockNrm, 6, "TerrainRockN", "TerrainHasRockN")
	bind(t.blendTex, 7, "TerrainBlendMap", "TerrainHasBlend")
	setUni1f(gs, "TerrainWorldW", t.hf.worldW)
	setUni1f(gs, "TerrainWorldD", t.hf.worldD)
	setUni1f(gs, "TerrainOriginX", t.hf.ox)
	setUni1f(gs, "TerrainOriginZ", t.hf.oz)
}
