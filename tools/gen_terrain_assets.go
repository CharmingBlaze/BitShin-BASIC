// Generates original CC0 splat textures for examples/assets/terrain.
// Not copied from Terrain-OpenGL JPGs.
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func main() {
	dir := filepath.Join("examples", "assets", "terrain")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	kinds := []string{"sand", "grass", "grass2", "rock", "snow", "rocknormal", "cloud"}
	names := []string{"sand.png", "grass.png", "grass2.png", "rock.png", "snow.png", "rock_n.png", "cloud.png"}
	for i, kind := range kinds {
		f, err := os.Create(filepath.Join(dir, names[i]))
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, gen(kind, 192)); err != nil {
			f.Close()
			panic(err)
		}
		f.Close()
	}
}

func gen(kind string, n int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	seed := map[string]uint32{"sand": 3, "grass": 7, "grass2": 11, "rock": 19, "snow": 23, "rocknormal": 29, "cloud": 41}[kind]
	perm := make([]int, 512)
	p := make([]int, 256)
	for i := 0; i < 256; i++ {
		p[i] = i
	}
	s := seed
	if s == 0 {
		s = 1
	}
	for i := 255; i > 0; i-- {
		s = s*1664525 + 1013904223
		j := int(s>>16) % (i + 1)
		p[i], p[j] = p[j], p[i]
	}
	for i := 0; i < 256; i++ {
		perm[i], perm[i+256] = p[i], p[i]
	}
	fade := func(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }
	grad := func(h int, x, z float64) float64 {
		switch h & 3 {
		case 0:
			return x + z
		case 1:
			return -x + z
		case 2:
			return x - z
		default:
			return -x - z
		}
	}
	noise := func(x, z float64) float64 {
		xi, zi := int(math.Floor(x))&255, int(math.Floor(z))&255
		xf, zf := x-math.Floor(x), z-math.Floor(z)
		u, v := fade(xf), fade(zf)
		aa := perm[perm[xi]+zi]
		ab := perm[perm[xi]+zi+1]
		ba := perm[perm[xi+1]+zi]
		bb := perm[perm[xi+1]+zi+1]
		x1 := grad(aa, xf, zf)*(1-u) + grad(ba, xf-1, zf)*u
		x2 := grad(ab, xf, zf-1)*(1-u) + grad(bb, xf-1, zf-1)*u
		return x1*(1-v) + x2*v
	}
	fbm := func(x, z float64) float64 {
		amp, freq, sum, norm := 1.0, 1.0, 0.0, 0.0
		for i := 0; i < 5; i++ {
			sum += noise(x*freq, z*freq) * amp
			norm += amp
			amp *= 0.5
			freq *= 2
		}
		return sum / norm
	}
	u8 := func(v float64) uint8 {
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		return uint8(v * 255)
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			u := float64(x) / float64(n)
			v := float64(y) / float64(n)
			n1 := fbm(u*6, v*6)*0.5 + 0.5
			n2 := fbm(u*14+3.1, v*14-1.7)*0.5 + 0.5
			n3 := fbm(u*28, v*28)*0.5 + 0.5
			var c color.RGBA
			switch kind {
			case "sand":
				g := 0.72 + 0.18*n1 + 0.08*n3
				c = color.RGBA{u8(g * 1.05), u8(g * 0.92), u8(g * 0.62), 255}
			case "grass":
				c = color.RGBA{u8(0.12 + 0.10*n2), u8(0.28 + 0.42*n1 + 0.12*n2), u8(0.10 + 0.08*n3), 255}
			case "grass2":
				c = color.RGBA{u8(0.18 + 0.16*n2), u8(0.22 + 0.38*n1), u8(0.08 + 0.06*n1), 255}
			case "rock":
				g := 0.28 + 0.22*n1 + 0.10*n3
				band := 0.06 * math.Sin(v*40+n2*4)
				c = color.RGBA{u8(g*1.15 + band), u8(g * 0.95), u8(g * 0.72), 255}
			case "snow":
				g := 0.86 + 0.12*n1
				c = color.RGBA{u8(g), u8(g + 0.02), u8(g + 0.04), 255}
			case "rocknormal":
				eps := 1.0 / float64(n)
				hl := fbm((u-eps)*8, v*8)
				hr := fbm((u+eps)*8, v*8)
				hd := fbm(u*8, (v-eps)*8)
				hu := fbm(u*8, (v+eps)*8)
				nx, ny, nz := hl-hr, 0.35, hd-hu
				lenN := math.Sqrt(nx*nx + ny*ny + nz*nz)
				c = color.RGBA{u8(nx/lenN*0.5 + 0.5), u8(nz/lenN*0.5 + 0.5), u8(ny/lenN*0.5 + 0.5), 255}
			default:
				cv := math.Pow(clamp((n1*0.65+n2*0.35-0.28)*1.6), 1.15)
				g := u8(cv)
				c = color.RGBA{g, g, g, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
