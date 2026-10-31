package runtime

import (
	"bufio"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// A colour lookup table is a 3D texture the post blit reads after the grade:
// the graded colour is the coordinate, the texel is the new colour. It loads
// from a .cube file (Resolve, Photoshop, most LUT packs), a horizontal PNG
// strip (size*size by size, Unity style), or a square PNG of tiles (512x512
// holds 64 tiles of 64x64), or is built from a named preset.

const (
	unitPostLUT   = 7 // 0 scene, 1 AO, 2-6 bloom chain
	unitPostDepth = 8
	lutPresetSize = 32
)

type lutState struct {
	tex    uint32
	size   int
	data   []float32 // RGB, red fastest; uploaded on the GL thread
	amount float32
	name   string
}

// outlineState draws dark lines where the depth jumps: silhouettes and
// creases, the classic companion to toon shading.
type outlineState struct {
	width float32 // pixels, 0 off
	color math32.Color
	depth float32 // relative depth step that counts as an edge
}

var lutPresets = map[string]func(r, g, b float64) (float64, float64, float64){
	"identity": func(r, g, b float64) (float64, float64, float64) { return r, g, b },
	"warm": func(r, g, b float64) (float64, float64, float64) {
		return r*1.06 + 0.02, g*1.01 + 0.01, b * 0.88
	},
	"cool": func(r, g, b float64) (float64, float64, float64) {
		return r * 0.9, g*0.99 + 0.01, b*1.08 + 0.03
	},
	// Teal shadows, orange skin tones: split by luminance.
	"teal_orange": func(r, g, b float64) (float64, float64, float64) {
		y := lumaRec709(r, g, b)
		t := smooth01(y)
		r = mixf(r*0.85, r*1.1+0.03, t)
		g = mixf(g*1.02+0.02, g*1.0, t)
		b = mixf(b*1.12+0.04, b*0.82, t)
		return sCurve(r, 0.25), sCurve(g, 0.25), sCurve(b, 0.25)
	},
	"bleach": func(r, g, b float64) (float64, float64, float64) {
		y := lumaRec709(r, g, b)
		r, g, b = mixf(y, r, 0.45), mixf(y, g, 0.45), mixf(y, b, 0.45)
		return sCurve(r, 0.45), sCurve(g, 0.45), sCurve(b, 0.45)
	},
	"noir": func(r, g, b float64) (float64, float64, float64) {
		y := sCurve(lumaRec709(r, g, b), 0.5)
		return y, y, y * 0.97
	},
	"vintage": func(r, g, b float64) (float64, float64, float64) {
		y := lumaRec709(r, g, b)
		r, g, b = mixf(y, r, 0.75), mixf(y, g, 0.75), mixf(y, b, 0.75)
		// Lifted blacks, softened whites, a yellow cast.
		return 0.08 + r*0.86, 0.06 + g*0.84, 0.04 + b*0.72
	},
	"vivid": func(r, g, b float64) (float64, float64, float64) {
		y := lumaRec709(r, g, b)
		r, g, b = mixf(y, r, 1.35), mixf(y, g, 1.35), mixf(y, b, 1.35)
		return sCurve(r, 0.2), sCurve(g, 0.2), sCurve(b, 0.2)
	},
	"night": func(r, g, b float64) (float64, float64, float64) {
		y := lumaRec709(r, g, b)
		r, g, b = mixf(y, r, 0.35), mixf(y, g, 0.35), mixf(y, b, 0.35)
		return r * 0.7, g * 0.85, b*1.05 + 0.02
	},
}

func lumaRec709(r, g, b float64) float64 { return 0.2126*r + 0.7152*g + 0.0722*b }

func mixf(a, b, t float64) float64 { return a + (b-a)*t }

func smooth01(x float64) float64 {
	x = math.Max(0, math.Min(1, x))
	return x * x * (3 - 2*x)
}

// sCurve adds contrast around mid grey; k 0 is a straight line.
func sCurve(x, k float64) float64 {
	x = math.Max(0, math.Min(1, x))
	return mixf(x, smooth01(x), k)
}

func buildPresetLUT(name string, size int) ([]float32, bool) {
	fn, ok := lutPresets[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, false
	}
	out := make([]float32, 0, size*size*size*3)
	s := float64(size - 1)
	for bi := 0; bi < size; bi++ {
		for gi := 0; gi < size; gi++ {
			for ri := 0; ri < size; ri++ {
				r, g, b := fn(float64(ri)/s, float64(gi)/s, float64(bi)/s)
				out = append(out, clampLUT(r), clampLUT(g), clampLUT(b))
			}
		}
	}
	return out, true
}

func clampLUT(v float64) float32 {
	if v < 0 || math.IsNaN(v) {
		return 0
	}
	if v > 1 {
		return 1
	}
	return float32(v)
}

// parseCubeLUT reads the Adobe/Resolve .cube text format: a LUT_3D_SIZE
// line, then size^3 "r g b" rows with red changing fastest.
func parseCubeLUT(path string) ([]float32, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	size := 0
	minV := [3]float64{0, 0, 0}
	maxV := [3]float64{1, 1, 1}
	var data []float32
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Fields(line)
		switch strings.ToUpper(fields[0]) {
		case "TITLE":
			continue
		case "LUT_1D_SIZE":
			return nil, 0, fmt.Errorf("%s: 1D LUTs are not supported", filepath.Base(path))
		case "LUT_3D_SIZE":
			if len(fields) < 2 {
				return nil, 0, fmt.Errorf("%s: LUT_3D_SIZE has no value", filepath.Base(path))
			}
			size, err = strconv.Atoi(fields[1])
			if err != nil || size < 2 || size > 256 {
				return nil, 0, fmt.Errorf("%s: bad LUT_3D_SIZE %q", filepath.Base(path), fields[1])
			}
			data = make([]float32, 0, size*size*size*3)
			continue
		case "DOMAIN_MIN", "DOMAIN_MAX":
			if len(fields) < 4 {
				continue
			}
			dst := &minV
			if strings.ToUpper(fields[0]) == "DOMAIN_MAX" {
				dst = &maxV
			}
			for i := 0; i < 3; i++ {
				dst[i], _ = strconv.ParseFloat(fields[i+1], 64)
			}
			continue
		}
		if size == 0 || len(fields) < 3 {
			continue
		}
		for i := 0; i < 3; i++ {
			v, err := strconv.ParseFloat(fields[i], 64)
			if err != nil {
				return nil, 0, fmt.Errorf("%s: bad value %q", filepath.Base(path), fields[i])
			}
			span := maxV[i] - minV[i]
			if span <= 0 {
				span = 1
			}
			data = append(data, clampLUT((v-minV[i])/span))
		}
	}
	if err := sc.Err(); err != nil {
		return nil, 0, err
	}
	if size == 0 {
		return nil, 0, fmt.Errorf("%s: no LUT_3D_SIZE line", filepath.Base(path))
	}
	if len(data) != size*size*size*3 {
		return nil, 0, fmt.Errorf("%s: %d values, want %d", filepath.Base(path), len(data)/3, size*size*size)
	}
	return data, size, nil
}

// imageLUT reads a strip (w = size*size, h = size) or a square of tiles
// (w = h = size*sqrt(size)). Tiles step blue; inside a tile x is red and y
// is green.
func imageLUT(img image.Image) ([]float32, int, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	size, tilesX := 0, 0
	switch {
	case h > 1 && w == h*h:
		size, tilesX = h, h
	case w == h:
		// size^3 = w*h, so size = (w*h)^(1/3)
		s := int(math.Round(math.Cbrt(float64(w * h))))
		if s*s*s == w*h {
			size = s
			tilesX = w / s
		}
	}
	if size < 2 || tilesX < 1 {
		return nil, 0, fmt.Errorf("LUT image is %dx%d: want a strip (size*size x size) or square tiles (e.g. 512x512)", w, h)
	}
	data := make([]float32, 0, size*size*size*3)
	for bi := 0; bi < size; bi++ {
		ox := (bi % tilesX) * size
		oy := (bi / tilesX) * size
		for gi := 0; gi < size; gi++ {
			for ri := 0; ri < size; ri++ {
				r, g, bb, _ := img.At(b.Min.X+ox+ri, b.Min.Y+oy+gi).RGBA()
				data = append(data, float32(r)/65535, float32(g)/65535, float32(bb)/65535)
			}
		}
	}
	return data, size, nil
}

func loadLUTFile(path string) ([]float32, int, error) {
	if strings.EqualFold(filepath.Ext(path), ".cube") {
		return parseCubeLUT(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return imageLUT(img)
}

// uploadLUT makes or refreshes the 3D texture. GL thread only.
func (w *World) uploadLUT() bool {
	l := &w.post.lut
	if l.data != nil {
		if l.tex == 0 {
			gl.GenTextures(1, &l.tex)
		}
		gl.ActiveTexture(gl.TEXTURE0 + unitPostLUT)
		gl.BindTexture(gl.TEXTURE_3D, l.tex)
		gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)
		s := int32(l.size)
		gl.TexImage3D(gl.TEXTURE_3D, 0, gl.RGB16F, s, s, s, 0, gl.RGB, gl.FLOAT, gl.Ptr(l.data))
		gl.BindTexture(gl.TEXTURE_3D, 0)
		gl.ActiveTexture(gl.TEXTURE0)
		l.data = nil
		drainGL("LUT upload")
	}
	return l.tex != 0 && l.amount > 0.001
}

// bindLUTAndOutline sets the post program's LUT and outline uniforms and
// binds their textures; unbindLUTAndOutline clears the units after the draw.
func (w *World) bindLUTAndOutline(prog uint32) {
	lutOn := w.uploadLUT()
	gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uLUTOn\x00")), int32(b01(lutOn)))
	gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uLUT\x00")), unitPostLUT)
	if lutOn {
		gl.ActiveTexture(gl.TEXTURE0 + unitPostLUT)
		gl.BindTexture(gl.TEXTURE_3D, w.post.lut.tex)
		gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uLUTAmount\x00")), w.post.lut.amount)
		gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uLUTSize\x00")), float32(w.post.lut.size))
	}
	o := w.post.outline
	outOn := o.width > 0.01 && w.post.depth != 0
	gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uOutlineOn\x00")), int32(b01(outOn)))
	gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uDepth\x00")), unitPostDepth)
	if outOn {
		gl.ActiveTexture(gl.TEXTURE0 + unitPostDepth)
		gl.BindTexture(gl.TEXTURE_2D, w.post.depth)
		near, far, _, _ := w.gtaoCamera()
		gl.Uniform2f(gl.GetUniformLocation(prog, gl.Str("uNearFar\x00")), near, far)
		gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uOutlineWidth\x00")), o.width)
		gl.Uniform3f(gl.GetUniformLocation(prog, gl.Str("uOutlineColor\x00")), o.color.R, o.color.G, o.color.B)
		th := o.depth
		if th <= 0 {
			th = 0.06
		}
		gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uOutlineDepth\x00")), th)
	}
	gl.ActiveTexture(gl.TEXTURE0)
}

func (w *World) unbindLUTAndOutline() {
	gl.ActiveTexture(gl.TEXTURE0 + unitPostLUT)
	gl.BindTexture(gl.TEXTURE_3D, 0)
	gl.ActiveTexture(gl.TEXTURE0 + unitPostDepth)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.ActiveTexture(gl.TEXTURE0)
}

func lutPresetNames() string {
	return "identity, warm, cool, teal_orange, bleach, noir, vintage, vivid, night"
}

func (w *World) lutCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetLUT "preset" | "file.cube" | "file.png" [, amount]
		"setlut": n(func(a []value.Value) (value.Value, error) {
			name := strings.TrimSpace(argS(a, 0))
			amount := float32(argN(a, 1, 1))
			if name == "" || strings.EqualFold(name, "none") || strings.EqualFold(name, "off") {
				w.post.lut.amount = 0
				w.post.lut.name = ""
				return z()
			}
			data, size := []float32(nil), 0
			if d, ok := buildPresetLUT(name, lutPresetSize); ok {
				data, size = d, lutPresetSize
			} else {
				path, err := w.openPath(name)
				if err != nil {
					return value.Value{}, fmt.Errorf("SetLUT: %q is not a file or a preset (%s)", name, lutPresetNames())
				}
				d, s, err := loadLUTFile(path)
				if err != nil {
					return value.Value{}, fmt.Errorf("SetLUT: %w", err)
				}
				data, size = d, s
			}
			w.post.lut.data, w.post.lut.size, w.post.lut.name = data, size, name
			w.post.lut.amount = clamp01n(amount)
			w.post.on = true
			return z()
		}),
		"setlutamount": n(func(a []value.Value) (value.Value, error) {
			w.post.lut.amount = clamp01n(float32(argN(a, 0, 1)))
			return value.Num(float64(w.post.lut.amount)), nil
		}),
		"getlutamount": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.post.lut.amount)), nil
		}),
		"clearlut": n(func(a []value.Value) (value.Value, error) {
			w.post.lut.amount = 0
			w.post.lut.name = ""
			return z()
		}),
		// SetOutline width [, r, g, b [, depthStep]]; 0 turns it off.
		"setoutline": n(func(a []value.Value) (value.Value, error) {
			wd := float32(argN(a, 0, 1.5))
			if wd < 0 {
				wd = 0
			}
			if wd > 6 {
				wd = 6
			}
			w.post.outline.width = wd
			if len(a) >= 4 {
				w.post.outline.color = math32.Color{
					R: colorArg(argN(a, 1, 0)), G: colorArg(argN(a, 2, 0)), B: colorArg(argN(a, 3, 0)),
				}
			}
			if len(a) >= 5 {
				w.post.outline.depth = float32(argN(a, 4, 0.06))
			}
			if wd > 0 {
				w.post.on = true
			}
			return z()
		}),
	}
}

// colorArg takes 0-255 or 0-1, like SetColorGrade's tint.
func colorArg(v float64) float32 {
	if v > 1 {
		v /= 255
	}
	return clamp01n(float32(v))
}
