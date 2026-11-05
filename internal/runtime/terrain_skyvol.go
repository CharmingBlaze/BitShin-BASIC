package runtime

import (
	"math"
	"runtime"
	"sync"

	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// How much of the sky each point in the world can see, through the ground and the trees.
// Ambient light comes from the sky, so without this the middle of a forest is lit as well
// as a meadow. It stores no color, only visibility and the direction it mostly comes from,
// so unlike baked light it stays right as the sun moves and the sky changes.
//
// It is worked out on the CPU, by tracing rays through the height field and a grid of tree
// crowns, on a worker. Nothing is rendered, so it costs no frame. It knows only of terrain
// and planted trees: a building does not shade what stands under it.

const (
	skyVolDefaultCell = 8.0 // meters between samples
	skyVolDefaultAmt  = 0.7
	skyVolMaxSide     = 160
	skyVolRays        = 24
	skyVolReach       = 180.0 // meters a ray looks for ground
	skyVolCrownSigma  = 0.1   // how quickly a tree crown stops light, per meter
	skyVolCrownCell   = 4.0   // meters across a cell of the crown grid
)

// skyVolTree is one tree's crown as light sees it: a soft ball.
type skyVolTree struct{ x, y, z, r float32 }

// skyVolScene is what the rays run through. No GL, and nothing shared with the world.
type skyVolScene struct {
	height       func(x, z float32) float32
	x0, z0, w, d float32
	y0, y1       float32 // lowest ground and the top of everything
	crowns       []float32
	cx, cy, cz   int
	crownY0      float32
	nx, ny, nz   int
	cell         float32
}

type skyVolume struct {
	tex       uint32
	min, size math32.Vector3
	strength  float32
	pending   bool
	done      chan *skyVolResult
	doneFor   *terrain
}

type skyVolResult struct {
	pix        []uint8
	nx, ny, nz int
	min, size  math32.Vector3
	ground     []float32 // visibility at ground level, nx by nz
}

func newSkyVolScene(height func(x, z float32) float32, x0, z0, w, d, cell float32, trees []skyVolTree) *skyVolScene {
	s := &skyVolScene{height: height, x0: x0, z0: z0, w: w, d: d, cell: cell}
	s.nx = clampInt(int(w/cell)+1, 2, skyVolMaxSide)
	s.nz = clampInt(int(d/cell)+1, 2, skyVolMaxSide)
	lo, hi := float32(math.MaxFloat32), float32(-math.MaxFloat32)
	for j := 0; j <= 64; j++ {
		for i := 0; i <= 64; i++ {
			h := height(x0+w*float32(i)/64, z0+d*float32(j)/64)
			lo, hi = math32.Min(lo, h), math32.Max(hi, h)
		}
	}
	for _, t := range trees {
		hi = math32.Max(hi, t.y+t.r)
	}
	s.y0, s.y1 = lo-cell, hi+cell
	s.ny = clampInt(int((s.y1-s.y0)/cell)+1, 2, skyVolMaxSide)
	s.buildCrowns(trees)
	return s
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (s *skyVolScene) buildCrowns(trees []skyVolTree) {
	s.cx = int(s.w/skyVolCrownCell) + 1
	s.cz = int(s.d/skyVolCrownCell) + 1
	s.crownY0 = s.y0
	s.cy = int((s.y1-s.y0)/skyVolCrownCell) + 1
	s.crowns = make([]float32, s.cx*s.cy*s.cz)
	for _, t := range trees {
		r := int(t.r/skyVolCrownCell) + 1
		cx := int((t.x - s.x0) / skyVolCrownCell)
		cy := int((t.y - s.crownY0) / skyVolCrownCell)
		cz := int((t.z - s.z0) / skyVolCrownCell)
		for k := cz - r; k <= cz+r; k++ {
			for j := cy - r; j <= cy+r; j++ {
				for i := cx - r; i <= cx+r; i++ {
					if i < 0 || j < 0 || k < 0 || i >= s.cx || j >= s.cy || k >= s.cz {
						continue
					}
					dx := s.x0 + (float32(i)+0.5)*skyVolCrownCell - t.x
					dy := s.crownY0 + (float32(j)+0.5)*skyVolCrownCell - t.y
					dz := s.z0 + (float32(k)+0.5)*skyVolCrownCell - t.z
					if dx*dx+dy*dy+dz*dz <= t.r*t.r {
						o := (k*s.cy+j)*s.cx + i
						s.crowns[o] = math32.Min(s.crowns[o]+skyVolCrownSigma, skyVolCrownSigma*2)
					}
				}
			}
		}
	}
}

// skyRayDirs are directions over the upper half of the sky, weighted as a surface facing
// up receives them, spaced by the golden angle.
func skyRayDirs() [skyVolRays][3]float32 {
	var out [skyVolRays][3]float32
	for i := range out {
		u := (float64(i) + 0.5) / skyVolRays
		sinT := math.Sqrt(u)
		cosT := math.Sqrt(1 - u)
		phi := float64(i) * 2.39996323
		out[i] = [3]float32{float32(sinT * math.Cos(phi)), float32(cosT), float32(sinT * math.Sin(phi))}
	}
	return out
}

// through is how much light gets to (x, y, z) along a direction, 0 to 1.
func (s *skyVolScene) through(x, y, z float32, d [3]float32) float32 {
	tau := float32(0)
	t := s.cell * 0.5
	for t < skyVolReach {
		px, py, pz := x+d[0]*t, y+d[1]*t, z+d[2]*t
		if py > s.y1 {
			break
		}
		if py < s.height(px, pz) {
			return 0
		}
		i, j, k := int((px-s.x0)/skyVolCrownCell), int((py-s.crownY0)/skyVolCrownCell), int((pz-s.z0)/skyVolCrownCell)
		// Steps grow with distance: what is far away shades a little of the sky.
		step := s.cell*0.5 + t*0.12
		if i >= 0 && j >= 0 && k >= 0 && i < s.cx && j < s.cy && k < s.cz {
			tau += s.crowns[(k*s.cy+j)*s.cx+i] * step
			if tau > 5 {
				return 0
			}
		}
		t += step
	}
	return float32(math.Exp(float64(-tau)))
}

// sample is visibility and the mean direction of what is visible.
func (s *skyVolScene) sample(x, y, z float32, dirs *[skyVolRays][3]float32) (vis, bx, by, bz float32) {
	for i := range dirs {
		l := s.through(x, y, z, dirs[i])
		vis += l
		bx += dirs[i][0] * l
		by += dirs[i][1] * l
		bz += dirs[i][2] * l
	}
	vis /= skyVolRays
	if m := math32.Sqrt(bx*bx + by*by + bz*bz); m > 1e-4 {
		bx, by, bz = bx/m, by/m, bz/m
	} else {
		bx, by, bz = 0, 1, 0
	}
	return vis, bx, by, bz
}

// bake fills the volume, on as many cores as may be spared. A sample under the ground takes
// the value of the first one above it, so filtering near the surface does not read black.
func (s *skyVolScene) bake(workers int) *skyVolResult {
	r := &skyVolResult{nx: s.nx, ny: s.ny, nz: s.nz}
	r.min = math32.Vector3{X: s.x0, Y: s.y0, Z: s.z0}
	r.size = math32.Vector3{X: s.w, Y: s.y1 - s.y0, Z: s.d}
	r.pix = make([]uint8, s.nx*s.ny*s.nz*4)
	r.ground = make([]float32, s.nx*s.nz)
	dirs := skyRayDirs()
	at := func(i, n int, lo, size float32) float32 { return lo + size*(float32(i)+0.5)/float32(n) }
	var wg sync.WaitGroup
	rows := make(chan int, s.nz)
	for k := 0; k < s.nz; k++ {
		rows <- k
	}
	close(rows)
	if workers < 1 {
		workers = 1
	}
	for n := 0; n < workers; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range rows {
				z := at(k, s.nz, s.z0, s.d)
				for i := 0; i < s.nx; i++ {
					x := at(i, s.nx, s.x0, s.w)
					ground := s.height(x, z)
					first := true
					for j := 0; j < s.ny; j++ {
						y := at(j, s.ny, s.y0, s.y1-s.y0)
						if y < ground {
							continue
						}
						vis, bx, by, bz := s.sample(x, y, z, &dirs)
						put := func(jj int) {
							o := ((k*s.ny+jj)*s.nx + i) * 4
							r.pix[o] = uint8((bx*0.5 + 0.5) * 255)
							r.pix[o+1] = uint8((by*0.5 + 0.5) * 255)
							r.pix[o+2] = uint8((bz*0.5 + 0.5) * 255)
							r.pix[o+3] = uint8(vis*255 + 0.5)
						}
						put(j)
						if first {
							first = false
							for jj := 0; jj < j; jj++ {
								put(jj)
							}
							// Just above the ground, not on it: on it, half the rays start buried.
							gv, _, _, _ := s.sample(x, ground+1.5, z, &dirs)
							r.ground[k*s.nx+i] = gv
						}
					}
				}
			}
		}()
	}
	wg.Wait()
	return r
}

// startSkyVolume gathers the trees here, on the GL thread, and bakes on a worker.
func (w *World) startSkyVolume(t *terrain, cell, strength float32) {
	hf := &t.hf
	if hf.proc || hf.worldW <= 0 {
		return
	}
	var trees []skyVolTree
	cw := t.chunkWorld()
	for _, li := range w.meshTreeLayers(t) {
		layer := t.scatters[li]
		parts, minY := w.treeMeshParts(layer.mesh)
		top := partsTop(parts, minY)
		x0, x1 := int(math32.Floor(hf.ox/cw)), int(math32.Ceil((hf.ox+hf.worldW)/cw))
		z0, z1 := int(math32.Floor(hf.oz/cw)), int(math32.Ceil((hf.oz+hf.worldD)/cw))
		for cz := z0; cz < z1; cz++ {
			for cx := x0; cx < x1; cx++ {
				k := chunkKey{X: cx, Z: cz}
				for _, p := range w.scatterPoints(t, layer, k, li, float32(cx)*cw, float32(cz)*cw, cw, 1) {
					h := top * p.scale
					trees = append(trees, skyVolTree{p.x, p.y + h*0.62, p.z, h * 0.28})
				}
			}
		}
	}
	height := func(x, z float32) float32 { return w.sampleHeight(hf, x, z) }
	s := newSkyVolScene(height, hf.ox, hf.oz, hf.worldW, hf.worldD, cell, trees)
	sv := &w.skyVol
	sv.strength = strength
	sv.pending = true
	workers := runtime.NumCPU() / 4
	if fixedDelta() > 0 {
		// A repeatable run must not change when the worker happens to finish.
		w.adoptSkyVolume(t, s.bake(workers))
		return
	}
	done := make(chan *skyVolResult, 1)
	sv.done = done
	sv.doneFor = t
	go func() { done <- s.bake(workers) }()
}

func (w *World) tickSkyVolume() {
	sv := &w.skyVol
	if sv.done == nil {
		return
	}
	select {
	case r := <-sv.done:
		sv.done = nil
		w.adoptSkyVolume(sv.doneFor, r)
	default:
	}
}

func (w *World) adoptSkyVolume(t *terrain, r *skyVolResult) {
	sv := &w.skyVol
	sv.pending = false
	if w.headless {
		return
	}
	if sv.tex == 0 {
		gl.GenTextures(1, &sv.tex)
	}
	gl.ActiveTexture(gl.TEXTURE0 + unitSkyVolume)
	gl.BindTexture(gl.TEXTURE_3D, sv.tex)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage3D(gl.TEXTURE_3D, 0, gl.RGBA8, int32(r.nx), int32(r.ny), int32(r.nz), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(r.pix))
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_3D, 0)
	gl.ActiveTexture(gl.TEXTURE0)
	sv.min, sv.size = r.min, r.size
	w.foldSkyIntoBake(t, r)
}

// foldSkyIntoBake shades the terrain's own occlusion map with the sky volume at ground
// level. Terrain has no texture unit to spare for the volume, and this needs none.
func (w *World) foldSkyIntoBake(t *terrain, r *skyVolResult) {
	if t == nil || t.bakeImg == nil || t.bakeTex == 0 {
		return
	}
	slot := w.texs[t.bakeTex]
	if slot == nil || slot.tex == nil {
		return
	}
	if t.bakeAOPlain == nil {
		t.bakeAOPlain = make([]uint8, t.bakeImg.Bounds().Dx()*t.bakeImg.Bounds().Dy())
		for i := range t.bakeAOPlain {
			t.bakeAOPlain[i] = t.bakeImg.Pix[i*4+2]
		}
	}
	b := t.bakeImg.Bounds()
	size := b.Dx()
	for y := 0; y < b.Dy(); y++ {
		v := (float32(y) + 0.5) / float32(b.Dy()) * float32(r.nz)
		for x := 0; x < size; x++ {
			u := (float32(x) + 0.5) / float32(size) * float32(r.nx)
			vis := bilinearGrid(r.ground, r.nx, r.nz, u-0.5, v-0.5)
			vis = 1 - (1-vis)*w.skyVol.strength
			o := (y*size + x)
			t.bakeImg.Pix[o*4+2] = uint8(float32(t.bakeAOPlain[o]) * vis)
		}
	}
	slot.tex.SetFromRGBA(t.bakeImg)
}

func bilinearGrid(g []float32, nx, nz int, u, v float32) float32 {
	u = math32.Clamp(u, 0, float32(nx-1))
	v = math32.Clamp(v, 0, float32(nz-1))
	i, j := int(u), int(v)
	fu, fv := u-float32(i), v-float32(j)
	i2, j2 := i+1, j+1
	if i2 >= nx {
		i2 = i
	}
	if j2 >= nz {
		j2 = j
	}
	a := g[j*nx+i]*(1-fu) + g[j*nx+i2]*fu
	b := g[j2*nx+i]*(1-fu) + g[j2*nx+i2]*fu
	return a*(1-fv) + b*fv
}

// bindSkyVolume tells a lit or PBR program about the volume.
func (w *World) bindSkyVolume(gs *gls.GLS) {
	sv := &w.skyVol
	if sv.tex == 0 || !w.linearActive() {
		setUni1i(gs, "SkyVolOn", 0)
		return
	}
	gl.ActiveTexture(gl.TEXTURE0 + unitSkyVolume)
	gl.BindTexture(gl.TEXTURE_3D, sv.tex)
	gl.ActiveTexture(gl.TEXTURE0)
	setUni1i(gs, "uSkyVol", unitSkyVolume)
	setUni1i(gs, "SkyVolOn", 1)
	setUni3f(gs, "SkyVolMin", sv.min.X, sv.min.Y, sv.min.Z)
	setUni3f(gs, "SkyVolSize", sv.size.X, sv.size.Y, sv.size.Z)
	setUni1f(gs, "SkyVolStrength", sv.strength)
}

// skyVolumeGLSL is how much of the sky a surface sees, facing the way it faces. The bent
// direction is in the volume's coordinates, where z runs opposite to the renderer's.
const skyVolumeGLSL = `
uniform sampler3D uSkyVol;
uniform int SkyVolOn;
uniform vec3 SkyVolMin;
uniform vec3 SkyVolSize;
uniform float SkyVolStrength;
float skyVisibility(vec3 wp, vec3 n) {
    if (SkyVolOn == 0) { return 1.0; }
    vec3 p = vec3(wp.x, wp.y, -wp.z);
    vec4 t = texture(uSkyVol, (p - SkyVolMin) / SkyVolSize);
    vec3 bent = t.rgb * 2.0 - 1.0;
    bent.z = -bent.z;
    float toward = clamp(dot(n, bent) * 0.5 + 0.5, 0.0, 1.0);
    return mix(1.0, clamp(0.2 + t.a * (0.45 + 0.8 * toward), 0.0, 1.0), SkyVolStrength);
}
`

func (w *World) skyVolumeCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// BakeSkyVolume([t,] [cell, strength]): shade ambient light by how much sky each
		// point can see through the ground and planted trees. Call it after planting.
		// Works in the background, and stays right as the sun moves.
		"bakeskyvolume": n(func(a []value.Value) (value.Value, error) {
			t := w.terrains[w.curTerrain]
			at := 0
			if len(a) > 0 {
				if tt := w.terrains[argI(a, 0, 0)]; tt != nil {
					t, at = tt, 1
				}
			}
			if t == nil {
				return value.Num(0), nil
			}
			cell := float32(argN(a, at, skyVolDefaultCell))
			if cell < 2 {
				cell = 2
			}
			w.startSkyVolume(t, cell, float32(argN(a, at+1, skyVolDefaultAmt)))
			return z()
		}),
		// SkyVolumeReady(): 1 once the volume is in use, 0 while it bakes or if never asked.
		"skyvolumeready": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(b01(w.skyVol.tex != 0 && !w.skyVol.pending))), nil
		}),
	}
}
