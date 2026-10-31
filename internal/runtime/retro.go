package runtime

import (
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/g3n/engine/gls"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
)

// Retro screen effects, all in the post pass:
//   - a low internal resolution, upscaled with sharp (nearest) pixels, and
//     optionally pixel-perfect: the largest whole-number scale, letterboxed;
//   - a plain render scale for speed (smooth upscale);
//   - colour banding (levels per channel), ordered (Bayer) dithering, and a
//     fixed palette every pixel snaps to;
//   - nearest texture filtering for the whole scene, through one sampler
//     object on the material texture units.
// HUD text and ImGui draw after the upscale, at full resolution.

const (
	unitPostPalette = 9 // 0 scene, 1 AO, 2-6 bloom, 7 LUT, 8 depth
	retroTexUnits   = 8 // material units 0-7 get the nearest sampler; shadows start at 8
	maxPalette      = 256
)

type retroPost struct {
	lowW, lowH   int     // internal resolution; 0 off
	pixelPerfect bool    // whole-number scale, letterboxed
	scale        float64 // render scale 0.25..1 (smooth); 0 or 1 off
	levels       float32 // colour levels per channel; 0 off
	dither       float32 // ordered dither strength 0..1
	palette      []float32
	paletteName  string
	palTex       uint32
	palDirty     bool
	nearestTex   bool
	sampler      uint32
}

var retroPalettes = map[string][]string{
	"gameboy": {"0f380f", "306230", "8bac0f", "9bbc0f"},
	"pico8": {"000000", "1d2b53", "7e2553", "008751", "ab5236", "5f574f", "c2c3c7", "fff1e8",
		"ff004d", "ffa300", "ffec27", "00e436", "29adff", "83769c", "ff77a8", "ffccaa"},
	"cga": {"000000", "55ffff", "ff55ff", "ffffff"},
	"ega": {"000000", "0000aa", "00aa00", "00aaaa", "aa0000", "aa00aa", "aa5500", "aaaaaa",
		"555555", "5555ff", "55ff55", "55ffff", "ff5555", "ff55ff", "ffff55", "ffffff"},
	"c64": {"000000", "ffffff", "880000", "aaffee", "cc44cc", "00cc55", "0000aa", "eeee77",
		"dd8855", "664400", "ff7777", "333333", "777777", "aaff66", "0088ff", "bbbbbb"},
	"grey4": {"000000", "555555", "aaaaaa", "ffffff"},
}

func hexPalette(cols []string) []float32 {
	out := make([]float32, 0, len(cols)*3)
	for _, h := range cols {
		v, _ := strconv.ParseUint(h, 16, 32)
		out = append(out, float32(v>>16&255)/255, float32(v>>8&255)/255, float32(v&255)/255)
	}
	return out
}

// imagePalette takes each distinct colour of an image, in reading order.
func imagePalette(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	seen := map[uint32]bool{}
	var out []float32
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			k := (r>>8)<<16 | (g>>8)<<8 | bl>>8
			if seen[k] {
				continue
			}
			if len(seen) == maxPalette {
				return nil, fmt.Errorf("more than %d colours", maxPalette)
			}
			seen[k] = true
			out = append(out, float32(r>>8)/255, float32(g>>8)/255, float32(bl>>8)/255)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no colours")
	}
	return out, nil
}

// retroTarget is the size the scene renders at for a window of ww x hh.
func (w *World) retroTarget(ww, hh int) (int, int) {
	r := &w.retroScr
	switch {
	case r.lowW > 0:
		h := r.lowH
		if h <= 0 {
			h = max(1, int(math.Round(float64(r.lowW)*float64(hh)/float64(max(ww, 1)))))
		}
		return r.lowW, h
	case r.scale > 0 && r.scale < 1:
		return max(1, int(float64(ww)*r.scale)), max(1, int(float64(hh)*r.scale))
	}
	return ww, hh
}

// retroOn is true when the scene renders at a size other than the window.
func (w *World) retroOn() bool { return w.retroScr.lowW > 0 || w.retroScr.scale > 0 && w.retroScr.scale < 1 }

// retroViewport is where the upscaled scene goes in a dw x dh window:
// the whole window, or a centred whole-number multiple when pixel-perfect.
func (w *World) retroViewport(dw, dh, pw, ph int) (x, y, vw, vh int) {
	if !w.retroScr.pixelPerfect || w.retroScr.lowW <= 0 || pw <= 0 || ph <= 0 {
		return 0, 0, dw, dh
	}
	s := max(1, min(dw/pw, dh/ph))
	vw, vh = pw*s, ph*s
	return (dw - vw) / 2, (dh - vh) / 2, vw, vh
}

// retroAspect is the camera aspect for the scene target: the target's own
// shape when letterboxed, else the window's (the image is stretched to it).
func (w *World) retroAspect(ww, hh, pw, ph int) float32 {
	if w.retroScr.pixelPerfect && w.retroScr.lowW > 0 {
		return float32(pw) / float32(max(ph, 1))
	}
	return float32(ww) / float32(max(hh, 1))
}

// bindRetroSampler puts a nearest-filtering sampler on the material texture
// units for the scene pass; unbindRetroSampler takes it off before the HUD.
func (w *World) bindRetroSampler() {
	r := &w.retroScr
	if !r.nearestTex {
		return
	}
	if r.sampler == 0 {
		gl.GenSamplers(1, &r.sampler)
		gl.SamplerParameteri(r.sampler, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.SamplerParameteri(r.sampler, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.SamplerParameteri(r.sampler, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.SamplerParameteri(r.sampler, gl.TEXTURE_WRAP_T, gl.REPEAT)
	}
	for u := uint32(0); u < retroTexUnits; u++ {
		gl.BindSampler(u, r.sampler)
	}
}

func (w *World) unbindRetroSampler() {
	if !w.retroScr.nearestTex {
		return
	}
	for u := uint32(0); u < retroTexUnits; u++ {
		gl.BindSampler(u, 0)
	}
}

// bindRetroPost sets the post program's retro uniforms and palette.
func (w *World) bindRetroPost(prog uint32, pw, ph int) {
	r := &w.retroScr
	gl.Uniform2f(gl.GetUniformLocation(prog, gl.Str("uSrcSize\x00")), float32(pw), float32(ph))
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uRetroLevels\x00")), r.levels)
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uRetroDither\x00")), r.dither)
	n := len(r.palette) / 3
	gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uRetroPalN\x00")), int32(n))
	gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uRetroPal\x00")), unitPostPalette)
	if n == 0 {
		return
	}
	if r.palTex == 0 {
		gl.GenTextures(1, &r.palTex)
		r.palDirty = true
	}
	gl.ActiveTexture(gl.TEXTURE0 + unitPostPalette)
	gl.BindTexture(gl.TEXTURE_2D, r.palTex)
	if r.palDirty {
		buf := make([]float32, maxPalette*3)
		copy(buf, r.palette)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB16F, maxPalette, 1, 0, gl.RGB, gl.FLOAT, gl.Ptr(buf))
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		r.palDirty = false
	}
	gl.ActiveTexture(gl.TEXTURE0)
}

func (w *World) unbindRetroPost() {
	if len(w.retroScr.palette) == 0 {
		return
	}
	gl.ActiveTexture(gl.TEXTURE0 + unitPostPalette)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.ActiveTexture(gl.TEXTURE0)
}

// setPostFilter makes the scene texture sample nearest (sharp pixels) when
// a retro resolution is on, else linear.
func (w *World) setPostFilter() {
	f := int32(gl.LINEAR)
	if w.retroScr.lowW > 0 {
		f = gl.NEAREST
	}
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, f)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, f)
}

// retroPreset is one look: screen settings plus geometry settings.
type retroPreset struct {
	lowW, lowH   int
	pixelPerfect bool
	levels       float32
	dither       float32
	palette      string
	nearestTex   bool
	snapW        float32 // vertex snap columns; rows follow the window
	affine       float32
	flat         bool
	fogSteps     int
}

var retroPresets = map[string]retroPreset{
	"off":     {},
	"ps1":     {lowW: 320, levels: 32, dither: 0.5, nearestTex: true, snapW: 320, affine: 1, fogSteps: 8},
	"n64":     {lowW: 320, levels: 32, dither: 0.25},
	"gameboy": {lowW: 160, lowH: 144, pixelPerfect: true, palette: "gameboy", dither: 0.8, nearestTex: true, flat: true},
	"pico8":   {lowW: 128, lowH: 128, pixelPerfect: true, palette: "pico8", dither: 0.5, nearestTex: true},
	"dos":     {lowW: 320, lowH: 200, levels: 6, dither: 0.35, nearestTex: true},
	"cga":     {lowW: 320, lowH: 200, palette: "cga", dither: 1, nearestTex: true},
	"lowpoly": {flat: true},
}

func (w *World) applyRetroPreset(name string) error {
	p, ok := retroPresets[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return fmt.Errorf("SetRetroPreset: want off, ps1, n64, gameboy, pico8, dos, cga, or lowpoly, got %q", name)
	}
	r := &w.retroScr
	r.lowW, r.lowH, r.pixelPerfect = p.lowW, p.lowH, p.pixelPerfect
	r.levels, r.dither, r.nearestTex = p.levels, p.dither, p.nearestTex
	r.palette, r.paletteName = nil, ""
	if p.palette != "" {
		r.palette, r.paletteName, r.palDirty = hexPalette(retroPalettes[p.palette]), p.palette, true
	}
	w.setRetroGeom(p.snapW, 0, p.affine, p.flat, p.fogSteps)
	if w.retroOn() || r.levels > 0 || r.palette != nil {
		w.post.on = true
	}
	if r.lowW == 0 && w.cam != nil {
		sw, sh := w.screenSize2D()
		w.cam.SetAspect(float32(sw) / float32(max(sh, 1)))
	}
	return nil
}

func (w *World) retroCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	r := &w.retroScr
	return map[string]cmd{
		// SetRetroPreset name$: a whole look in one call — ps1, n64, gameboy,
		// pico8, dos, cga, lowpoly — or "off". Change single settings after.
		"setretropreset": n(func(a []value.Value) (value.Value, error) {
			if err := w.applyRetroPreset(argS(a, 0)); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		// SetRetroResolution w [, h, pixelPerfect]: render the scene at
		// w x h and upscale with sharp pixels. h 0 follows the window's shape.
		// SetRetroResolution 0 turns it off.
		"setretroresolution": n(func(a []value.Value) (value.Value, error) {
			r.lowW, r.lowH = max(0, argI(a, 0, 0)), max(0, argI(a, 1, 0))
			r.pixelPerfect = argI(a, 2, 0) != 0
			if r.lowW > 0 {
				w.post.on = true
			} else if w.cam != nil {
				sw, sh := w.screenSize2D()
				w.cam.SetAspect(float32(sw) / float32(max(sh, 1)))
			}
			return z()
		}),
		// SetRenderScale f: render at f (0.25 to 1) of the window and scale up
		// smoothly. A speed knob, not a look; 1 turns it off.
		"setrenderscale": n(func(a []value.Value) (value.Value, error) {
			s := argN(a, 0, 1)
			if s >= 1 || s <= 0 {
				r.scale = 0
				return z()
			}
			r.scale = math.Max(0.25, s)
			w.post.on = true
			return z()
		}),
		"getrenderscale": n(func(a []value.Value) (value.Value, error) {
			if r.scale <= 0 {
				return value.Num(1), nil
			}
			return value.Num(r.scale), nil
		}),
		// SetRetroColors levels: levels per channel (32 is 15-bit colour,
		// 4 is very banded); 0 turns it off.
		"setretrocolors": n(func(a []value.Value) (value.Value, error) {
			l := argI(a, 0, 0)
			if l == 1 {
				l = 2
			}
			r.levels = float32(max(0, min(l, 256)))
			if r.levels > 0 {
				w.post.on = true
			}
			return z()
		}),
		// SetRetroDither amount: 0..1 ordered (Bayer 4x4) dither between
		// colour levels or palette colours.
		"setretrodither": n(func(a []value.Value) (value.Value, error) {
			r.dither = clamp01n(float32(argN(a, 0, 1)))
			return z()
		}),
		// SetRetroPalette name$ | file.png: every pixel snaps to the nearest
		// palette colour. Names: gameboy, pico8, cga, ega, c64, grey4.
		// "" or "none" turns it off.
		"setretropalette": n(func(a []value.Value) (value.Value, error) {
			name := strings.TrimSpace(argS(a, 0))
			if name == "" || strings.EqualFold(name, "none") || strings.EqualFold(name, "off") {
				r.palette, r.paletteName = nil, ""
				return z()
			}
			var pal []float32
			if cols, ok := retroPalettes[strings.ToLower(name)]; ok {
				pal = hexPalette(cols)
			} else {
				path, err := w.openPath(name)
				if err != nil {
					return value.Value{}, fmt.Errorf("SetRetroPalette: %q is not a file or one of gameboy, pico8, cga, ega, c64, grey4", name)
				}
				if pal, err = imagePalette(path); err != nil {
					return value.Value{}, fmt.Errorf("SetRetroPalette: %s: %w", name, err)
				}
			}
			r.palette, r.paletteName, r.palDirty = pal, name, true
			w.post.on = true
			return z()
		}),
		// SetRetroTextures on: nearest-neighbour filtering on every scene
		// texture (crisp texels, no smoothing).
		"setretrotextures": n(func(a []value.Value) (value.Value, error) {
			r.nearestTex = argI(a, 0, 1) != 0
			return z()
		}),
		// SetTextureFilter tex, mode$: "nearest" or "linear" for one texture.
		"settexturefilter": n(func(a []value.Value) (value.Value, error) {
			ts := w.texs[argI(a, 0, 0)]
			if ts == nil || ts.tex == nil {
				return value.Value{}, fmt.Errorf("SetTextureFilter: %d is not a texture", argI(a, 0, 0))
			}
			switch strings.ToLower(strings.TrimSpace(argS(a, 1))) {
			case "nearest", "point", "pixel":
				ts.tex.SetMagFilter(gls.NEAREST)
				ts.tex.SetMinFilter(gls.NEAREST)
			case "linear", "smooth", "":
				ts.tex.SetMagFilter(gls.LINEAR)
				ts.tex.SetMinFilter(gls.LINEAR_MIPMAP_LINEAR)
			default:
				return value.Value{}, fmt.Errorf("SetTextureFilter: want nearest or linear, got %q", argS(a, 1))
			}
			return z()
		}),
	}
}
