package runtime

import (
	"fmt"
	"image"
	"image/color"
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

	"bitshinbasic/internal/value"
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
	splat      bool
	sandTex    int
	grassTex   int
	grass2Tex  int
	rockTex    int
	snowTex    int
	rockNrm    int
	grassCover float32
	waterY     float32
	blend      float32
	fogFalloff float32
	tessMul    float32
	rock       math32.Color
	snowOn     bool
	snowH      float32
}

func (hf *heightField) sample(x, z float32) float32 {
	if hf == nil || hf.gw < 2 || hf.gd < 2 || len(hf.h) == 0 {
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
			return 0
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
	if gw < 2 {
		gw = 2
	}
	if gd < 2 {
		gd = 2
	}
	h := make([]float32, gw*gd)
	for z := 0; z < gd; z++ {
		for x := 0; x < gw; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+z).RGBA()
			lum := float32(r+g+bl) / (3 * 65535)
			h[z*gw+x] = lum * hscale
		}
	}
	_ = color.Gray{}
	return heightField{gw: gw, gd: gd, worldW: worldW, worldD: worldD, hscale: hscale, h: h}, nil
}

// sharedHeightNormal is world-space (same stencil at a given XZ) so
// neighboring chunks / LOD steps share the edge normal. Halo samples
// (N+1) let edge verts use the same neighbor heights as the next chunk.
func sharedHeightNormal(sample func(x, z float32) float32, x, z, cell float32) (nx, ny, nz float32) {
	eps := cell
	if eps < 0.35 {
		eps = 0.35
	}
	if eps > 2 {
		eps = 2
	}
	// 3×3 halo around the vertex; accumulate 4 quad face normals.
	h := [3][3]float32{}
	for jz := 0; jz < 3; jz++ {
		for jx := 0; jx < 3; jx++ {
			h[jz][jx] = sample(x+float32(jx-1)*eps, z+float32(jz-1)*eps)
		}
	}
	add := func(ax, ay, az, bx, by, bz, cx, cy, cz float32) {
		e1x, e1y, e1z := bx-ax, by-ay, bz-az
		e2x, e2y, e2z := cx-ax, cy-ay, cz-az
		nx += e1y*e2z - e1z*e2y
		ny += e1z*e2x - e1x*e2z
		nz += e1x*e2y - e1y*e2x
	}
	// G3N positions for the 3×3 (Z flipped).
	var p [3][3][3]float32
	for jz := 0; jz < 3; jz++ {
		for jx := 0; jx < 3; jx++ {
			wx := x + float32(jx-1)*eps
			wz := z + float32(jz-1)*eps
			p[jz][jx][0], p[jz][jx][1], p[jz][jx][2] = toG3N(wx, h[jz][jx], wz)
		}
	}
	// Four quads around the center vertex (1,1), two tris each, matching mesh winding.
	quad := func(i00, j00, i10, j10, i01, j01, i11, j11 int) {
		a := p[j00][i00]
		b := p[j00][i10]
		c := p[j01][i00]
		d := p[j01][i11]
		add(a[0], a[1], a[2], b[0], b[1], b[2], d[0], d[1], d[2])
		add(a[0], a[1], a[2], d[0], d[1], d[2], c[0], c[1], c[2])
	}
	quad(0, 0, 1, 0, 0, 1, 1, 1)
	quad(1, 0, 2, 0, 1, 1, 2, 1)
	quad(0, 1, 1, 1, 0, 2, 1, 2)
	quad(1, 1, 2, 1, 1, 2, 2, 2)
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

func buildHeightMesh(sample func(x, z float32) float32, x0, z0, x1, z1 float32, segs, step int) *geometry.Geometry {
	if segs < 2 {
		segs = 2
	}
	if step < 1 {
		step = 1
	}
	chunkSize := segs / step // quads along a side
	if chunkSize < 1 {
		chunkSize = 1
	}
	n := chunkSize + 1 // vertsPerSide; overlap 1 row with the neighbor's first row
	pos := math32.NewArrayF32(0, n*n*3+n*4*3)
	nor := math32.NewArrayF32(0, n*n*3+n*4*3)
	uvs := math32.NewArrayF32(0, n*n*2+n*4*2)
	idx := math32.NewArrayU32(0, (n-1)*(n-1)*6+n*4*6)
	const normCell = float32(0.85)
	const skirtDrop = float32(8)
	nverts := 0
	vert := func(x, z, yoff float32) {
		y := sample(x, z) - yoff
		gx, gy, gz := toG3N(x, y, z)
		pos.Append(gx, gy, gz)
		nx, ny, nz := sharedHeightNormal(sample, x, z, normCell)
		nor.Append(nx, ny, nz)
		uvs.Append((x-x0)/max32(x1-x0, 1)*4, (z-z0)/max32(z1-z0, 1)*4)
		nverts++
	}
	for iz := 0; iz < n; iz++ {
		for ix := 0; ix < n; ix++ {
			x := gridCoord(x0, x1, ix, chunkSize)
			z := gridCoord(z0, z1, iz, chunkSize)
			vert(x, z, 0)
		}
	}
	for iz := 0; iz < n-1; iz++ {
		for ix := 0; ix < n-1; ix++ {
			a := uint32(iz*n + ix)
			b := a + 1
			c := a + uint32(n)
			d := c + 1
			idx.Append(a, b, d, a, d, c)
		}
	}
	// Vertical skirts so LOD T-junctions don't flash the sky (white horizon slits).
	base := uint32(n * n)
	addSkirt := func(s0, s1, k0, k1 uint32) {
		if nverts < 1 {
			return
		}
		last := uint32(nverts - 1)
		if s0 > last || s1 > last || k0 > last || k1 > last {
			return
		}
		idx.Append(s0, s1, k1, s0, k1, k0)
	}
	// min-z edge
	for ix := 0; ix < n; ix++ {
		x := gridCoord(x0, x1, ix, chunkSize)
		vert(x, z0, skirtDrop)
	}
	for ix := 0; ix < n-1; ix++ {
		s0 := uint32(ix)
		addSkirt(s0, s0+1, base+uint32(ix), base+uint32(ix+1))
	}
	// max-z edge
	k0 := base + uint32(n)
	for ix := 0; ix < n; ix++ {
		x := gridCoord(x0, x1, ix, chunkSize)
		vert(x, z1, skirtDrop)
	}
	for ix := 0; ix < n-1; ix++ {
		s0 := uint32((n-1)*n + ix)
		addSkirt(s0+1, s0, k0+uint32(ix+1), k0+uint32(ix))
	}
	// min-x edge
	k1 := k0 + uint32(n)
	for iz := 0; iz < n; iz++ {
		z := gridCoord(z0, z1, iz, chunkSize)
		vert(x0, z, skirtDrop)
	}
	for iz := 0; iz < n-1; iz++ {
		s0 := uint32(iz * n)
		addSkirt(s0+uint32(n), s0, k1+uint32(iz+1), k1+uint32(iz))
	}
	// max-x edge
	k2 := k1 + uint32(n)
	for iz := 0; iz < n; iz++ {
		z := gridCoord(z0, z1, iz, chunkSize)
		vert(x1, z, skirtDrop)
	}
	for iz := 0; iz < n-1; iz++ {
		s0 := uint32(iz*n + (n - 1))
		addSkirt(s0, s0+uint32(n), k2+uint32(iz), k2+uint32(iz+1))
	}
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
		return t.hf.atWorld(x, z)
	}
	for _, t := range w.terrains {
		if t != nil {
			return t.hf.atWorld(x, z)
		}
	}
	return 0
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
			w.freeEntityID(id)
			delete(t.ents, k)
		}
	}
	for k := range need {
		if t.ents[k] != 0 || t.pending[k] {
			continue
		}
		if t.pending == nil {
			t.pending = map[chunkKey]bool{}
		}
		t.pending[k] = true
		key := k
		size := cw
		step := 1
		dx := float32(key.X-c.X) * float32(key.X-c.X)
		dz := float32(key.Z-c.Z) * float32(key.Z-c.Z)
		if t.lod > 0 && dx+dz > 2 {
			step = 2
		}
		if t.lod > 0 && dx+dz > 8 {
			step = 4
		}
		if t.lod > 1 && dx+dz > 4 {
			step = 4
		}
		hf := t.hf
		segs := t.chunkVerts
		if t.tessMul > 1 {
			segs = int(float32(segs) * t.tessMul)
		}
		if segs < 4 {
			segs = 17
		}
		tid := t.id
		w.ensureJobs().submit(func() {
			// Heights only on the worker. Geometry / VBOs stay on the GL thread.
			x0 := float32(key.X) * size
			z0 := float32(key.Z) * size
			w.ensureJobs().enqueueGL(func() {
				tt := w.terrains[tid]
				if tt == nil {
					return
				}
				g := buildHeightMesh(hf.atWorld, x0, z0, x0+size, z0+size, segs, step)
				id := w.terrainChunkEnt(g, tt)
				tt.ents[key] = id
				delete(tt.pending, key)
			})
		})
	}
	w.cullTerrainChunks(t, cw)
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
			t := w.pickTerrain(a)
			if t != nil {
				off := terrainArgOff(w, a)
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
		"setterrainblend": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				t.blend = float32(argN(a, terrainArgOff(w, a), 1.4))
			}
			return z()
		}),
		"setterrainsnow": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t != nil {
				off := terrainArgOff(w, a)
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
		lod: 1, grassCover: 0.65, waterY: 2.2, blend: 1.4,
		fogFalloff: 0, tessMul: 1, splat: true,
		rock: math32.Color{0.55, 0.48, 0.40}, snowH: 19,
	}
	w.curTerrain = id
	w.ensureSplatTextures(w.terrains[id])
	// Origin ring on the GL thread so the first Flip is not an empty plane.
	w.seedTerrainChunks(w.terrains[id])
	return id
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
	segs := t.chunkVerts
	if t.tessMul > 1 {
		segs = int(float32(segs) * t.tessMul)
	}
	if segs < 4 {
		segs = 17
	}
	for z := -ring; z <= ring; z++ {
		for x := -ring; x <= ring; x++ {
			k := chunkKey{x, z}
			if t.ents[k] != 0 {
				continue
			}
			x0 := float32(x) * cw
			z0 := float32(z) * cw
			g := buildHeightMesh(t.hf.atWorld, x0, z0, x0+cw, z0+cw, segs, 1)
			t.ents[k] = w.terrainChunkEnt(g, t)
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
		delete(t.ents, k)
	}
	t.pending = map[chunkKey]bool{}
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
	for _, e := range m.w.ents {
		if e == nil || e.lgtKind != 1 {
			continue
		}
		if c, ok := e.node.(interface{ Color() math32.Color }); ok {
			col := c.Color()
			cr, cg, cb = col.R, col.G, col.B
			break
		}
		if c, ok := e.node.(interface{ Color() *math32.Color }); ok {
			if col := c.Color(); col != nil {
				cr, cg, cb = col.R, col.G, col.B
			}
			break
		}
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
	bind(t.sandTex, 8, "TerrainSand", "TerrainHasSand")
	bind(t.grassTex, 9, "TerrainGrass", "TerrainHasGrass")
	bind(t.grass2Tex, 10, "TerrainGrass2", "TerrainHasGrass2")
	bind(t.rockTex, 11, "TerrainRock", "TerrainHasRock")
	bind(t.snowTex, 12, "TerrainSnow", "TerrainHasSnow")
	bind(t.rockNrm, 13, "TerrainRockN", "TerrainHasRockN")
}
