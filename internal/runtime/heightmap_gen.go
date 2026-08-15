package runtime

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/g3n/engine/texture"

	"bitshinbasic/internal/value"
)

// Procedural heightmaps (clean-room). Behavior matches the documented
// ergin3d/heightmap-generator tool (MIT): seeded FBM, pan, black/white
// levels, thermal + hydraulic erosion, terrace/ridged/invert/clamp/normalize,
// greyscale PNG export. No JS was copied. Diamond-square is an extra mode.

type heightMap struct {
	id   int
	w, h int
	data []float32 // [0,1]
	spec HeightmapSpec
}

// HeightmapSpec is the generator input (repo sliders + extras).
type HeightmapSpec struct {
	Width, Height     int
	Seed, Octaves     int
	Persistence       float64
	Lacunarity        float64
	Scale             float64
	PanX, PanY        float64
	Black, White      float64
	Mode              string // fbm (default), ridged, diamond / diamondsquare
}

func (s HeightmapSpec) normalize() HeightmapSpec {
	if s.Width < 2 {
		s.Width = 2
	}
	if s.Height < 2 {
		s.Height = 2
	}
	if s.Width > 1024 {
		s.Width = 1024
	}
	if s.Height > 1024 {
		s.Height = 1024
	}
	if s.Octaves < 1 {
		s.Octaves = 5
	}
	if s.Octaves > 12 {
		s.Octaves = 12
	}
	if s.Persistence <= 0 {
		s.Persistence = 0.5
	}
	if s.Lacunarity <= 0 {
		s.Lacunarity = 2
	}
	if s.Scale <= 0 {
		s.Scale = 80
	}
	if s.White <= 0 {
		s.White = 1
	}
	if s.Black < 0 {
		s.Black = 0
	}
	if s.White > 1 {
		s.White = 1
	}
	s.Mode = strings.ToLower(strings.TrimSpace(s.Mode))
	if s.Mode == "" {
		s.Mode = "fbm"
	}
	return s
}

// heightmapPreset copies the repo's named slider packs (scale is pixels).
func heightmapPreset(name string) HeightmapSpec {
	s := HeightmapSpec{Octaves: 5, Persistence: 0.5, Lacunarity: 2, Scale: 80, White: 1}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "rolling", "rolling-hills", "hills":
		s.Octaves, s.Persistence, s.Lacunarity, s.Scale = 4, 0.35, 2.2, 200
	case "sharp", "sharp-peaks", "peaks":
		s.Octaves, s.Persistence, s.Lacunarity, s.Scale = 8, 0.55, 2.5, 80
	case "archipelago", "islands":
		s.Octaves, s.Persistence, s.Lacunarity, s.Scale = 5, 0.45, 2.0, 120
	case "plateaus", "plateau":
		s.Octaves, s.Persistence, s.Lacunarity, s.Scale = 3, 0.3, 3.0, 150
	case "dunes", "gentle-dunes":
		s.Octaves, s.Persistence, s.Lacunarity, s.Scale = 3, 0.25, 2.0, 300
	case "alpine", "alps":
		s.Octaves, s.Persistence, s.Lacunarity, s.Scale = 7, 0.5, 2.2, 100
	}
	return s
}

// GenerateHeightmapCPU fills a [0,1] field. Pure CPU; no GL.
func GenerateHeightmapCPU(spec HeightmapSpec) []float32 {
	spec = spec.normalize()
	w, h := spec.Width, spec.Height
	data := make([]float32, w*h)
	switch spec.Mode {
	case "diamond", "diamondsquare", "ds":
		fillDiamondSquare(data, w, h, spec.Seed, spec.Persistence)
	default:
		n := newNoise2(spec.Seed)
		ridged := spec.Mode == "ridged" || spec.Mode == "ridge"
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				fx := (float64(x) + spec.PanX) / spec.Scale
				fz := (float64(y) + spec.PanY) / spec.Scale
				v := n.fbmOpen(fx, fz, spec.Octaves, spec.Persistence, spec.Lacunarity)
				if ridged {
					v = 1 - math.Abs(v)
				}
				data[y*w+x] = float32(v)
			}
		}
	}
	normalize01(data)
	if spec.Black > 0 || spec.White < 1 {
		rng := spec.White - spec.Black
		if rng < 1e-6 {
			rng = 1e-6
		}
		for i := range data {
			v := (float64(data[i]) - spec.Black) / rng
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			data[i] = float32(v)
		}
	}
	return data
}

func (n *noise2) fbmOpen(x, z float64, octaves int, persist, lacunarity float64) float64 {
	if octaves < 1 {
		octaves = 1
	}
	if persist <= 0 {
		persist = 0.5
	}
	if lacunarity <= 0 {
		lacunarity = 2
	}
	amp, freq, sum, norm := 1.0, 1.0, 0.0, 0.0
	for i := 0; i < octaves; i++ {
		sum += n.noise(x*freq, z*freq) * amp
		norm += amp
		amp *= persist
		freq *= lacunarity
	}
	if norm <= 0 {
		return 0
	}
	return sum / norm
}

func normalize01(data []float32) {
	if len(data) == 0 {
		return
	}
	mn, mx := data[0], data[0]
	for _, v := range data {
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	rng := mx - mn
	if rng < 1e-8 {
		for i := range data {
			data[i] = 0.5
		}
		return
	}
	for i := range data {
		data[i] = (data[i] - mn) / rng
	}
}

func fillDiamondSquare(dst []float32, w, h, seed int, roughness float64) {
	n := 2
	need := w
	if h > need {
		need = h
	}
	for n+1 < need {
		n *= 2
	}
	size := n + 1
	grid := make([]float32, size*size)
	rng := uint32(seed)
	if rng == 0 {
		rng = 1
	}
	next := func() float64 {
		rng = rng*1664525 + 1013904223
		return float64(rng)/math.MaxUint32*2 - 1
	}
	grid[0] = float32(next())
	grid[size-1] = float32(next())
	grid[(size-1)*size] = float32(next())
	grid[(size-1)*size+size-1] = float32(next())
	if roughness <= 0 {
		roughness = 0.5
	}
	step := n
	amp := 1.0
	for step > 1 {
		half := step / 2
		for y := half; y < size; y += step {
			for x := half; x < size; x += step {
				avg := (grid[(y-half)*size+(x-half)] + grid[(y-half)*size+(x+half)] +
					grid[(y+half)*size+(x-half)] + grid[(y+half)*size+(x+half)]) * 0.25
				grid[y*size+x] = avg + float32(next()*amp)
			}
		}
		for y := 0; y < size; y += half {
			for x := 0; x < size; x += half {
				if (x/half+y/half)%2 == 0 {
					continue
				}
				sum, c := float32(0), 0
				if x-half >= 0 {
					sum += grid[y*size+(x-half)]
					c++
				}
				if x+half < size {
					sum += grid[y*size+(x+half)]
					c++
				}
				if y-half >= 0 {
					sum += grid[(y-half)*size+x]
					c++
				}
				if y+half < size {
					sum += grid[(y+half)*size+x]
					c++
				}
				if c > 0 {
					grid[y*size+x] = sum/float32(c) + float32(next()*amp)
				}
			}
		}
		amp *= roughness
		step = half
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx := float64(x) / float64(w-1) * float64(size-1)
			fy := float64(y) / float64(h-1) * float64(size-1)
			dst[y*w+x] = sampleGrid(grid, size, size, fx, fy)
		}
	}
}

func sampleGrid(data []float32, w, h int, x, y float64) float32 {
	if w < 2 || h < 2 {
		return 0
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x > float64(w-1) {
		x = float64(w - 1)
	}
	if y > float64(h-1) {
		y = float64(h - 1)
	}
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1
	if x1 >= w {
		x1 = w - 1
	}
	if y1 >= h {
		y1 = h - 1
	}
	tx := float32(x - float64(x0))
	ty := float32(y - float64(y0))
	h00 := data[y0*w+x0]
	h10 := data[y0*w+x1]
	h01 := data[y1*w+x0]
	h11 := data[y1*w+x1]
	return (h00*(1-tx)+h10*tx)*(1-ty) + (h01*(1-tx)+h11*tx)*ty
}

// ThermalErodeHeightmap slides material down slopes steeper than talus.
func ThermalErodeHeightmap(data []float32, w, h, iterations int, talus, transfer float32) {
	if w < 3 || h < 3 || len(data) < w*h {
		return
	}
	if iterations < 1 {
		iterations = 1
	}
	if iterations > 80 {
		iterations = 80
	}
	if talus <= 0 {
		talus = 0.02
	}
	if transfer <= 0 {
		transfer = 0.25
	}
	if transfer > 0.5 {
		transfer = 0.5
	}
	tmp := make([]float32, len(data))
	for p := 0; p < iterations; p++ {
		copy(tmp, data)
		for y := 1; y < h-1; y++ {
			for x := 1; x < w-1; x++ {
				ci := y*w + x
				h0 := tmp[ci]
				minH, minI := h0, -1
				var drops [8]float32
				var idxs [8]int
				nLow := 0
				total := float32(0)
				k := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						ni := (y+dy)*w + (x + dx)
						nh := tmp[ni]
						if nh < h0 {
							d := h0 - nh
							drops[nLow] = d
							idxs[nLow] = ni
							total += d
							nLow++
						}
						if nh < minH {
							minH = nh
							minI = ni
						}
						k++
					}
				}
				_ = k
				diff := h0 - minH
				if diff <= talus || minI < 0 {
					continue
				}
				move := diff * transfer
				data[ci] -= move
				if total > 0 {
					for i := 0; i < nLow; i++ {
						data[idxs[i]] += move * (drops[i] / total)
					}
				} else {
					data[minI] += move
				}
			}
		}
	}
	clamp01slice(data)
}

// HydraulicErodeHeightmap is a droplet river carve (CPU).
func HydraulicErodeHeightmap(data []float32, w, h, droplets, life int, inertia, capacity, erode, deposit, evap float64) {
	if w < 4 || h < 4 || len(data) < w*h {
		return
	}
	if droplets < 1 {
		droplets = 200
	}
	if droplets > 8000 {
		droplets = 8000
	}
	if life < 2 {
		life = 24
	}
	if life > 80 {
		life = 80
	}
	if inertia <= 0 {
		inertia = 0.3
	}
	if capacity <= 0 {
		capacity = 4
	}
	if erode <= 0 {
		erode = 0.3
	}
	if deposit <= 0 {
		deposit = 0.3
	}
	if evap <= 0 {
		evap = 0.02
	}
	rng := uint32(42)
	rand01 := func() float64 {
		rng = rng*1664525 + 1013904223
		return float64(rng) / math.MaxUint32
	}
	for d := 0; d < droplets; d++ {
		px := rand01()*float64(w-3) + 1
		py := rand01()*float64(h-3) + 1
		dirX, dirY := 0.0, 0.0
		speed, water, sediment := 1.0, 1.0, 0.0
		for step := 0; step < life; step++ {
			xi, yi := int(math.Floor(px)), int(math.Floor(py))
			if xi < 1 || yi < 1 || xi >= w-2 || yi >= h-2 {
				break
			}
			gx, gy := heightGrad(data, w, h, px, py)
			dirX = dirX*inertia - gx*(1-inertia)
			dirY = dirY*inertia - gy*(1-inertia)
			ln := math.Hypot(dirX, dirY)
			if ln > 1e-4 {
				dirX /= ln
				dirY /= ln
			} else {
				ang := rand01() * 2 * math.Pi
				dirX, dirY = math.Cos(ang), math.Sin(ang)
			}
			nx, ny := px+dirX, py+dirY
			if nx < 1 || ny < 1 || nx >= float64(w-2) || ny >= float64(h-2) {
				break
			}
			oldH := float64(sampleGrid(data, w, h, px, py))
			newH := float64(sampleGrid(data, w, h, nx, ny))
			delta := newH - oldH
			cap := math.Max(-delta, 0.01) * speed * water * capacity
			if sediment > cap || delta > 0 {
				dep := (sediment - cap) * deposit
				if delta > 0 {
					dep = math.Min(sediment, delta)
				}
				sediment -= dep
				spreadDeposit(data, w, h, xi, yi, float32(dep))
			} else {
				amt := math.Min((cap-sediment)*erode, -delta)
				if amt > 0 {
					data[yi*w+xi] = float32(math.Max(0, float64(data[yi*w+xi])-amt))
					sediment += amt
				}
			}
			speed = math.Sqrt(math.Max(0, speed*speed+delta))
			water *= 1 - evap
			px, py = nx, ny
			if water < 0.001 || speed < 0.001 {
				break
			}
		}
	}
	clamp01slice(data)
}

func heightGrad(data []float32, w, h int, x, y float64) (gx, gy float64) {
	eps := 1.0
	r := float64(sampleGrid(data, w, h, x+eps, y))
	l := float64(sampleGrid(data, w, h, x-eps, y))
	u := float64(sampleGrid(data, w, h, x, y+eps))
	d := float64(sampleGrid(data, w, h, x, y-eps))
	return r - l, u - d
}

func spreadDeposit(data []float32, w, h, x, y int, amt float32) {
	q := amt * 0.25
	data[y*w+x] += q
	if x+1 < w {
		data[y*w+x+1] += q
	}
	if y+1 < h {
		data[(y+1)*w+x] += q
	}
	if x+1 < w && y+1 < h {
		data[(y+1)*w+x+1] += q
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clamp01slice(data []float32) {
	for i, v := range data {
		if v < 0 {
			data[i] = 0
		} else if v > 1 {
			data[i] = 1
		}
	}
}

// FilterHeightmap applies a named post filter (repo filter list).
func FilterHeightmap(data []float32, w, h int, kind string, strength float64) {
	if len(data) < w*h || w < 2 || h < 2 {
		return
	}
	if strength < 0 {
		strength = 0
	}
	if strength > 1 {
		strength = 1
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "smooth", "blur":
		smoothHeight(data, w, h, strength)
	case "sharpen":
		blur := append([]float32(nil), data...)
		smoothHeight(blur, w, h, 0.2)
		amt := strength * 2
		for i := range data {
			v := float64(data[i]) + (float64(data[i])-float64(blur[i]))*amt
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			data[i] = float32(v)
		}
	case "terrace":
		levels := 3 + strength*15
		if levels < 2 {
			levels = 2
		}
		for i, v := range data {
			stepped := math.Round(float64(v)*levels) / levels
			data[i] = float32(float64(v) + (stepped-float64(v))*strength)
		}
	case "ridged", "ridge":
		for i, v := range data {
			ridge := 1 - math.Abs(float64(v)*2-1)
			data[i] = float32(float64(v) + (ridge-float64(v))*strength)
		}
	case "invert":
		for i, v := range data {
			inv := 1 - float64(v)
			data[i] = float32(float64(v) + (inv-float64(v))*strength)
		}
	case "clamp":
		low := strength * 0.3
		high := 1 - strength*0.3
		for i, v := range data {
			fv := float64(v)
			if fv < low {
				data[i] = float32(fv + (low-fv)*strength)
			} else if fv > high {
				data[i] = float32(fv + (high-fv)*strength)
			}
		}
	case "normalize":
		cp := append([]float32(nil), data...)
		normalize01(cp)
		for i := range data {
			data[i] = float32(float64(data[i]) + (float64(cp[i])-float64(data[i]))*strength)
		}
	case "erode", "thermal":
		ThermalErodeHeightmap(data, w, h, int(math.Max(1, strength*12)), 0.02, 0.25)
	case "hydraulic", "river":
		HydraulicErodeHeightmap(data, w, h, int(math.Max(80, strength*800)), 20, 0.3, 4, 0.3, 0.3, 0.02)
	}
}

func smoothHeight(data []float32, w, h int, strength float64) {
	passes := int(math.Max(1, math.Round(strength*8)))
	tmp := make([]float32, len(data))
	for p := 0; p < passes; p++ {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				at := func(xx, yy int) float32 {
					if xx < 0 {
						xx = 0
					}
					if yy < 0 {
						yy = 0
					}
					if xx >= w {
						xx = w - 1
					}
					if yy >= h {
						yy = h - 1
					}
					return data[yy*w+xx]
				}
				c := at(x, y)
				n, s, we, e := at(x, y-1), at(x, y+1), at(x-1, y), at(x+1, y)
				nw, ne, sw, se := at(x-1, y-1), at(x+1, y-1), at(x-1, y+1), at(x+1, y+1)
				tmp[y*w+x] = (c*4 + n + s + we + e + nw*0.5 + ne*0.5 + sw*0.5 + se*0.5) / 10
			}
		}
		copy(data, tmp)
	}
}

func heightmapGrayImage(data []float32, w, h int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := data[y*w+x]
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			img.SetGray(x, y, color.Gray{Y: uint8(v*255 + 0.5)})
		}
	}
	return img
}

func heightmapRGBA(data []float32, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := data[y*w+x]
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			g := uint8(v*255 + 0.5)
			img.SetRGBA(x, y, color.RGBA{R: g, G: g, B: g, A: 255})
		}
	}
	return img
}

func saveHeightmapPNG(path string, data []float32, w, h int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, heightmapGrayImage(data, w, h))
}

func heightFieldFromMap(hm *heightMap, worldW, worldD, hscale float32) heightField {
	h := make([]float32, len(hm.data))
	for i, v := range hm.data {
		h[i] = v * hscale
	}
	return heightField{
		gw: hm.w, gd: hm.h, worldW: worldW, worldD: worldD, hscale: hscale,
		ox: -worldW / 2, oz: -worldD / 2, h: h,
	}
}

func (w *World) addHeightMap(spec HeightmapSpec, data []float32) int {
	spec = spec.normalize()
	if w.hmaps == nil {
		w.hmaps = map[int]*heightMap{}
	}
	id := w.takeHandle(&w.freeHMaps, &w.nextHMap)
	w.hmaps[id] = &heightMap{id: id, w: spec.Width, h: spec.Height, data: data, spec: spec}
	w.curHMap = id
	return id
}

func (w *World) pickHeightMap(a []value.Value, i int) *heightMap {
	id := argI(a, i, w.curHMap)
	if hm := w.hmaps[id]; hm != nil {
		return hm
	}
	return w.hmaps[w.curHMap]
}

func (w *World) heightmapCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"generateheightmap": n(func(a []value.Value) (value.Value, error) {
			spec := HeightmapSpec{
				Width: argI(a, 0, 128), Height: argI(a, 1, 128), Seed: argI(a, 2, 1),
				Octaves: argI(a, 3, 5), Scale: argN(a, 4, 80), Persistence: argN(a, 5, 0.5),
				Lacunarity: argN(a, 6, 2), PanX: argN(a, 7, 0), PanY: argN(a, 8, 0),
				Black: argN(a, 9, 0), White: argN(a, 10, 1),
			}
			if len(a) >= 12 {
				spec.Mode = argS(a, 11)
			}
			data := GenerateHeightmapCPU(spec)
			return value.Num(float64(w.addHeightMap(spec, data))), nil
		}),
		"generateheightmappreset": n(func(a []value.Value) (value.Value, error) {
			spec := heightmapPreset(argS(a, 0))
			spec.Width = argI(a, 1, 128)
			spec.Height = argI(a, 2, 128)
			spec.Seed = argI(a, 3, 1)
			data := GenerateHeightmapCPU(spec)
			return value.Num(float64(w.addHeightMap(spec, data))), nil
		}),
		"saveheightmap": n(func(a []value.Value) (value.Value, error) {
			path := argS(a, 0)
			if path == "" {
				return value.Num(0), fmt.Errorf("SaveHeightmap: path required")
			}
			hm := w.pickHeightMap(a, 1)
			if hm == nil {
				return value.Num(0), fmt.Errorf("SaveHeightmap: no heightmap")
			}
			out := w.resolve(path)
			if err := saveHeightmapPNG(out, hm.data, hm.w, hm.h); err != nil {
				return value.Num(0), err
			}
			return value.Num(1), nil
		}),
		"importheightmap": n(func(a []value.Value) (value.Value, error) {
			path, err := w.openPath(argS(a, 0))
			if err != nil {
				return value.Num(0), err
			}
			hf, err := loadHeightImage(path, 1, 1, 1)
			if err != nil {
				return value.Num(0), err
			}
			data := make([]float32, len(hf.h))
			hs := hf.hscale
			if hs == 0 {
				hs = 1
			}
			for i, v := range hf.h {
				data[i] = v / hs
			}
			spec := HeightmapSpec{Width: hf.gw, Height: hf.gd, White: 1}
			return value.Num(float64(w.addHeightMap(spec, data))), nil
		}),
		"erodeheightmap": n(func(a []value.Value) (value.Value, error) {
			hm := w.pickHeightMap(a, 0)
			if hm == nil {
				return value.Num(0), fmt.Errorf("ErodeHeightmap: no heightmap")
			}
			ThermalErodeHeightmap(hm.data, hm.w, hm.h, argI(a, 1, 8), float32(argN(a, 2, 0.02)), float32(argN(a, 3, 0.25)))
			return value.Num(float64(hm.id)), nil
		}),
		"hydraulicerodeheightmap": n(func(a []value.Value) (value.Value, error) {
			hm := w.pickHeightMap(a, 0)
			if hm == nil {
				return value.Num(0), fmt.Errorf("HydraulicErodeHeightmap: no heightmap")
			}
			HydraulicErodeHeightmap(hm.data, hm.w, hm.h, argI(a, 1, 400), argI(a, 2, 24),
				argN(a, 3, 0.3), argN(a, 4, 4), argN(a, 5, 0.3), argN(a, 6, 0.3), argN(a, 7, 0.02))
			return value.Num(float64(hm.id)), nil
		}),
		"filterheightmap": n(func(a []value.Value) (value.Value, error) {
			hm := w.pickHeightMap(a, 0)
			if hm == nil {
				return value.Num(0), fmt.Errorf("FilterHeightmap: no heightmap")
			}
			FilterHeightmap(hm.data, hm.w, hm.h, argS(a, 1), argN(a, 2, 0.6))
			return value.Num(float64(hm.id)), nil
		}),
		"heightmaptexture": need(func(a []value.Value) (value.Value, error) {
			hm := w.pickHeightMap(a, 0)
			if hm == nil {
				return value.Num(0), fmt.Errorf("HeightmapTexture: no heightmap")
			}
			tex := texture.NewTexture2DFromRGBA(heightmapRGBA(hm.data, hm.w, hm.h))
			id := w.nextTex
			w.nextTex++
			if w.texs == nil {
				w.texs = map[int]*texSlot{}
			}
			w.texs[id] = &texSlot{tex: tex, path: ""}
			return value.Num(float64(id)), nil
		}),
		"heightmapwidth": n(func(a []value.Value) (value.Value, error) {
			hm := w.pickHeightMap(a, 0)
			if hm == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(hm.w)), nil
		}),
		"heightmapheight": n(func(a []value.Value) (value.Value, error) {
			hm := w.pickHeightMap(a, 0)
			if hm == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(hm.h)), nil
		}),
		"freeheightmap": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, w.curHMap)
			if w.hmaps[id] == nil {
				return z()
			}
			delete(w.hmaps, id)
			w.recycleHandle(&w.freeHMaps, id)
			if w.curHMap == id {
				w.curHMap = 0
			}
			return z()
		}),
		"createterrainfromheightmap": need(func(a []value.Value) (value.Value, error) {
			hm := w.pickHeightMap(a, 0)
			if hm == nil {
				return value.Num(0), fmt.Errorf("CreateTerrainFromHeightmap: no heightmap")
			}
			ww := float32(argN(a, 1, 64))
			dd := float32(argN(a, 2, 64))
			hs := float32(argN(a, 3, 12))
			return value.Num(float64(w.addTerrain(heightFieldFromMap(hm, ww, dd, hs), 25, 3))), nil
		}),
	}
}
