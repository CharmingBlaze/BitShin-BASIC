package runtime

import (
	"image"
	"math"
	"sort"

	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/texture"

	"moonbasic/internal/value"
)

// BakeTerrain computes what the terrain shader would otherwise guess every frame
// and stores it in one RGBA8 map: world normal X and Z, ambient occlusion, and
// water flow. One texture read replaces several noise calls per pixel.
//
// Normals come from the map and not the mesh, so a far chunk lights the same as a
// near one (Losasso and Hoppe keep normals at twice geometry resolution for this).
// Steep ground gets detail no grid could hold: cracks, and ledges that follow the
// height, the way rock layers do. Occlusion is the real horizon around each point,
// which replaces a screen-space guess that changed with distance. Flow is D8 water
// accumulation, which shows where a slope has eroded.
//
// It is for a finite map. Sculpting turns the map off, since it no longer matches
// the ground: call BakeTerrain again once the edit is done.

const (
	bakeMaxSize  = 2048
	bakeMinSize  = 64
	bakeAORadius = 16   // cells: the furthest step of the horizon search
	bakeFlowSpan = 11.0 // log2 of the catchment that reads as a full gully
	bakeRockFrom = 0.7  // slope (rise over run) where rock detail starts
	bakeRockFull = 1.4  // and where it is full
)

// bakeGrid is the ground as a grid in metres.
type bakeGrid struct {
	w, d         int
	cellX, cellZ float32
	h            []float32
}

func (g *bakeGrid) at(x, z int) float32 {
	if x < 0 {
		x = 0
	} else if x >= g.w {
		x = g.w - 1
	}
	if z < 0 {
		z = 0
	} else if z >= g.d {
		z = g.d - 1
	}
	return g.h[z*g.w+x]
}

func catmullRom(a, b, c, d, t float32) float32 {
	return b + 0.5*t*(c-a+t*(2*a-5*b+4*c-d+t*(3*(b-c)+d-a)))
}

// smooth is Catmull-Rom between cells, at u, v in cell units. A bilinear height has
// a slope that jumps at every cell, which would print the grid into the normals.
func (g *bakeGrid) smooth(u, v float32) float32 {
	x, z := int(math.Floor(float64(u))), int(math.Floor(float64(v)))
	fx, fz := u-float32(x), v-float32(z)
	var row [4]float32
	for j := -1; j <= 2; j++ {
		row[j+1] = catmullRom(g.at(x-1, z+j), g.at(x, z+j), g.at(x+1, z+j), g.at(x+2, z+j), fx)
	}
	return catmullRom(row[0], row[1], row[2], row[3], fz)
}

func bilerp(f []float32, w, d int, u, v float32) float32 {
	x, z := int(u), int(v)
	if x > w-2 {
		x = w - 2
	}
	if z > d-2 {
		z = d - 2
	}
	fx, fz := u-float32(x), v-float32(z)
	a := f[z*w+x] + (f[z*w+x+1]-f[z*w+x])*fx
	b := f[(z+1)*w+x] + (f[(z+1)*w+x+1]-f[(z+1)*w+x])*fx
	return a + (b-a)*fz
}

// bakeFlow is D8 flow accumulation: visit cells from the highest down, and each
// passes everything it has collected to its lowest neighbour. Returns 0 to 1.
func bakeFlow(g *bakeGrid) []float32 {
	n := g.w * g.d
	order := make([]int32, n)
	for i := range order {
		order[i] = int32(i)
	}
	sort.Slice(order, func(a, b int) bool { return g.h[order[a]] > g.h[order[b]] })
	acc := make([]float32, n)
	for i := range acc {
		acc[i] = 1
	}
	for _, c := range order {
		z, x := int(c)/g.w, int(c)%g.w
		best, low := -1, g.h[c]
		for dz := -1; dz <= 1; dz++ {
			for dx := -1; dx <= 1; dx++ {
				nx, nz := x+dx, z+dz
				if (dx == 0 && dz == 0) || nx < 0 || nz < 0 || nx >= g.w || nz >= g.d {
					continue
				}
				if h := g.h[nz*g.w+nx]; h < low {
					best, low = nz*g.w+nx, h
				}
			}
		}
		if best >= 0 {
			acc[best] += acc[c]
		}
	}
	out := make([]float32, n)
	for i, a := range acc {
		v := float32(math.Log2(float64(a))) / bakeFlowSpan
		if v > 1 {
			v = 1
		}
		out[i] = v
	}
	return out
}

// bakeAO looks along eight directions for the highest thing each point can see.
// The sky left over is the occlusion. Returns 0 (buried) to 1 (open).
func bakeAO(g *bakeGrid) []float32 {
	out := make([]float32, g.w*g.d)
	dirs := [8][2]int{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	parallelRows(g.d, func(z0, z1 int) {
		for z := z0; z < z1; z++ {
			for x := 0; x < g.w; x++ {
				h0 := g.h[z*g.w+x]
				var sum float64
				for _, d := range dirs {
					var top float64
					for s := 1; s <= bakeAORadius; s *= 2 {
						dx, dz := float32(d[0]*s)*g.cellX, float32(d[1]*s)*g.cellZ
						run := math.Hypot(float64(dx), float64(dz))
						if slope := float64(g.at(x+d[0]*s, z+d[1]*s)-h0) / run; slope > top {
							top = slope
						}
					}
					// Sine of the horizon's elevation.
					sum += top / math.Sqrt(1+top*top)
				}
				out[z*g.w+x] = float32(1 - sum/8)
			}
		}
	})
	return out
}

// bakeTerrainMap builds the map: size texels a side over the whole grid.
func bakeTerrainMap(g *bakeGrid, size, seed int) *image.RGBA {
	flow, ao := bakeFlow(g), bakeAO(g)
	worldW, worldD := float32(g.w-1)*g.cellX, float32(g.d-1)*g.cellZ
	stepX, stepZ := worldW/float32(size-1), worldD/float32(size-1)
	crack, layer, warp := newNoise2(seed^0x51), newNoise2(seed^0x52), newNoise2(seed^0x53)

	// Height with detail, first, so normals are differences of one array.
	hs := make([]float32, size*size)
	parallelRows(size, func(j0, j1 int) {
		for j := j0; j < j1; j++ {
			v := float32(j) / float32(size-1) * float32(g.d-1)
			for i := 0; i < size; i++ {
				u := float32(i) / float32(size-1) * float32(g.w-1)
				h := g.smooth(u, v)
				sx := (g.smooth(u+0.5, v) - g.smooth(u-0.5, v)) / g.cellX
				sz := (g.smooth(u, v+0.5) - g.smooth(u, v-0.5)) / g.cellZ
				slope := math32Sqrt(sx*sx + sz*sz)
				rock := smoothstep32(bakeRockFrom, bakeRockFull, slope)
				x, z := float64(u*g.cellX), float64(v*g.cellZ)
				// Soft ground keeps a little unevenness everywhere.
				d := 0.1 * float32(crack.fbm(x*0.16, z*0.16, 3, 0.5, 2))
				if rock > 0 {
					// Cracks: ridged, so they cut in instead of bulging out.
					amp, f, cracks := float32(1.0), 0.09, float32(0)
					for o := 0; o < 4; o++ {
						r := 1 - 2*math.Abs(crack.noise(x*f+float64(o)*7.3, z*f-float64(o)*3.1))
						cracks += float32(r) * amp
						amp *= 0.55
						f *= 2.1
					}
					// Ledges: rock layers follow height, tilted and bent a little.
					w := warp.fbm(x*0.018, z*0.018, 2, 0.5, 2) * 9
					ledge := layer.noise(float64(h)*0.42+w+x*0.03, 11.7)
					d += rock * (cracks + float32(ledge)*1.1)
				}
				hs[j*size+i] = h + d
			}
		}
	})

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	parallelRows(size, func(j0, j1 int) {
		for j := j0; j < j1; j++ {
			jm, jp := j-1, j+1
			if jm < 0 {
				jm = 0
			}
			if jp >= size {
				jp = size - 1
			}
			v := float32(j) / float32(size-1) * float32(g.d-1)
			for i := 0; i < size; i++ {
				im, ip := i-1, i+1
				if im < 0 {
					im = 0
				}
				if ip >= size {
					ip = size - 1
				}
				dhdx := (hs[j*size+ip] - hs[j*size+im]) / (float32(ip-im) * stepX)
				dhdz := (hs[jp*size+i] - hs[jm*size+i]) / (float32(jp-jm) * stepZ)
				// The scene's Z runs opposite to the map's, so its normal's Z keeps
				// the slope's sign where X flips it.
				nx, ny, nz := -dhdx, float32(1), dhdz
				l := math32Sqrt(nx*nx + ny*ny + nz*nz)
				nx, nz = nx/l, nz/l
				u := float32(i) / float32(size-1) * float32(g.w-1)
				o := img.PixOffset(i, j)
				img.Pix[o] = unorm8(nx*0.5 + 0.5)
				img.Pix[o+1] = unorm8(nz*0.5 + 0.5)
				img.Pix[o+2] = unorm8(bilerp(ao, g.w, g.d, u, v))
				img.Pix[o+3] = unorm8(bilerp(flow, g.w, g.d, u, v))
			}
		}
	})
	return img
}

func unorm8(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

func math32Sqrt(v float32) float32 { return float32(math.Sqrt(float64(v))) }

func smoothstep32(a, b, v float32) float32 {
	t := (v - a) / (b - a)
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// bakeSizeFor is four texels per grid cell, as a power of two, within limits.
func bakeSizeFor(gw int) int {
	size := 256
	for size < gw*4 && size < bakeMaxSize {
		size *= 2
	}
	return size
}

// bakeTerrain builds the map from the ground as it stands, sculpting included.
func (w *World) bakeTerrain(t *terrain, size int) {
	hf := &t.hf
	if hf.proc || hf.gw < 2 || hf.gd < 2 || !hf.hasData() {
		return // an endless map has no whole to bake
	}
	if size <= 0 {
		size = bakeSizeFor(hf.gw)
	}
	if size > bakeMaxSize {
		size = bakeMaxSize
	}
	if size < bakeMinSize {
		size = bakeMinSize
	}
	g := &bakeGrid{
		w: hf.gw, d: hf.gd,
		cellX: hf.worldW / float32(hf.gw-1), cellZ: hf.worldD / float32(hf.gd-1),
		h: make([]float32, hf.gw*hf.gd),
	}
	for z := 0; z < g.d; z++ {
		for x := 0; x < g.w; x++ {
			g.h[z*g.w+x] = hf.atWorld(hf.ox+float32(x)*g.cellX, hf.oz+float32(z)*g.cellZ)
		}
	}
	w.dropTerrainBake(t)
	t.bakeImg, t.bakeAOPlain = bakeTerrainMap(g, size, hf.seed), nil
	tex := texture.NewTexture2DFromRGBA(t.bakeImg)
	tex.SetWrapS(gls.CLAMP_TO_EDGE)
	tex.SetWrapT(gls.CLAMP_TO_EDGE)
	id := w.nextTex
	w.nextTex++
	if w.texs == nil {
		w.texs = map[int]*texSlot{}
	}
	w.texs[id] = &texSlot{tex: tex}
	t.bakeTex = id
}

func (w *World) dropTerrainBake(t *terrain) {
	if t.bakeTex == 0 {
		return
	}
	if s := w.texs[t.bakeTex]; s != nil && s.tex != nil {
		s.tex.Dispose()
	}
	delete(w.texs, t.bakeTex)
	t.bakeTex = 0
}

func (w *World) terrainBakeCommands(n func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// BakeTerrain([t] [, size]): normal, occlusion, and flow maps for a finite
		// terrain, for better ground at less cost per pixel. Size defaults to four
		// texels per grid cell. Sculpting turns it off; call it again afterwards.
		"baketerrain": n(func(a []value.Value) (value.Value, error) {
			// One argument can be a terrain or a size, and they cannot be confused:
			// a terrain is a live id and a size is not.
			t, size := w.terrains[w.curTerrain], 0
			if v := argI(a, 0, 0); w.terrains[v] != nil {
				t, size = w.terrains[v], argI(a, 1, 0)
			} else {
				size = v
			}
			if t != nil {
				w.bakeTerrain(t, size)
			}
			return value.Num(float64(b01(t != nil && t.bakeTex != 0))), nil
		}),
	}
}
