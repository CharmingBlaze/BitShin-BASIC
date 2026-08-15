package runtime

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/texture"

	"bitshinbasic/internal/value"
)

// Original CC0 splat maps (not Nintendo, not the Terrain-OpenGL JPGs).
// MIT on that repo covers code; their photo textures are not shipped here.

func procTexKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	switch k {
	case "sand", "grass", "grass2", "rock", "snow", "rocknormal", "rockn", "dudv", "cloud", "clouds":
		if k == "rockn" {
			return "rocknormal"
		}
		if k == "clouds" {
			return "cloud"
		}
		return k
	default:
		return "grass"
	}
}

func generateProcTerrainRGBA(kind string, n int) *image.RGBA {
	if n < 16 {
		n = 128
	}
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	seed := 17
	switch procTexKind(kind) {
	case "sand":
		seed = 3
	case "grass":
		seed = 7
	case "grass2":
		seed = 11
	case "rock":
		seed = 19
	case "snow":
		seed = 23
	case "rocknormal":
		seed = 29
	case "dudv":
		seed = 31
	case "cloud":
		seed = 41
	}
	ns := newNoise2(seed)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			u := float64(x) / float64(n)
			v := float64(y) / float64(n)
			img.SetRGBA(x, y, procTexPixel(kind, ns, u, v, x, y, n))
		}
	}
	return img
}

func procTexPixel(kind string, ns *noise2, u, v float64, x, y, n int) color.RGBA {
	n1 := ns.fbm(u*6, v*6, 5, 0.5, 2)*0.5 + 0.5
	n2 := ns.fbm(u*14+3.1, v*14-1.7, 4, 0.55, 2.1)*0.5 + 0.5
	n3 := ns.fbm(u*28, v*28, 3, 0.5, 2)*0.5 + 0.5
	switch procTexKind(kind) {
	case "sand":
		g := 0.72 + 0.18*n1 + 0.08*n3
		return rgb8(g*1.05, g*0.92, g*0.62)
	case "grass":
		g := 0.28 + 0.42*n1 + 0.12*n2
		return rgb8(0.12+0.10*n2, g, 0.10+0.08*n3)
	case "grass2":
		g := 0.22 + 0.38*n1
		return rgb8(0.18+0.16*n2, g, 0.08+0.06*n1)
	case "rock":
		g := 0.28 + 0.22*n1 + 0.10*n3
		band := 0.06 * math.Sin(v*40+n2*4)
		return rgb8(g*1.15+band, g*0.95, g*0.72)
	case "snow":
		g := 0.86 + 0.12*n1
		return rgb8(g, g+0.02, g+0.04)
	case "rocknormal":
		eps := 1.0 / float64(n)
		hl := ns.fbm((u-eps)*8, v*8, 4, 0.5, 2)
		hr := ns.fbm((u+eps)*8, v*8, 4, 0.5, 2)
		hd := ns.fbm(u*8, (v-eps)*8, 4, 0.5, 2)
		hu := ns.fbm(u*8, (v+eps)*8, 4, 0.5, 2)
		nx := float32(hl - hr)
		nz := float32(hd - hu)
		ny := float32(0.35)
		lenN := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
		if lenN < 1e-5 {
			lenN = 1
		}
		return color.RGBA{
			uint8((nx/lenN*0.5 + 0.5) * 255),
			uint8((nz/lenN*0.5 + 0.5) * 255),
			uint8((ny/lenN*0.5 + 0.5) * 255),
			255,
		}
	case "dudv":
		return color.RGBA{uint8(n1 * 255), uint8(n2 * 255), 128, 255}
	case "cloud":
		c := math.Pow(clamp01f((n1*0.65+n2*0.35-0.28)*1.6), 1.15)
		g := uint8(c * 255)
		return color.RGBA{g, g, g, 255}
	default:
		return rgb8(0.3, 0.5, 0.25)
	}
}

func rgb8(r, g, b float64) color.RGBA {
	return color.RGBA{u8f(r), u8f(g), u8f(b), 255}
}

func clamp01f(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (w *World) addProcTexture(kind string) int {
	img := generateProcTerrainRGBA(kind, 192)
	tex := texture.NewTexture2DFromRGBA(img)
	tex.SetWrapS(gls.REPEAT)
	tex.SetWrapT(gls.REPEAT)
	tex.SetRepeat(1, 1)
	id := w.nextTex
	w.nextTex++
	if w.texs == nil {
		w.texs = map[int]*texSlot{}
	}
	w.texs[id] = &texSlot{tex: tex, path: "proc:" + procTexKind(kind)}
	return id
}

func (w *World) ensureSplatTextures(t *terrain) {
	if t == nil {
		return
	}
	if t.sandTex == 0 {
		t.sandTex = w.addProcTexture("sand")
	}
	if t.grassTex == 0 {
		t.grassTex = w.addProcTexture("grass")
	}
	if t.grass2Tex == 0 {
		t.grass2Tex = w.addProcTexture("grass2")
	}
	if t.rockTex == 0 {
		t.rockTex = w.addProcTexture("rock")
	}
	if t.snowTex == 0 {
		t.snowTex = w.addProcTexture("snow")
	}
	if t.rockNrm == 0 {
		t.rockNrm = w.addProcTexture("rocknormal")
	}
	t.splat = true
}

func (w *World) procTexCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createproctexture": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.addProcTexture(argS(a, 0)))), nil
		}),
	}
}
