package runtime

import (
	"math"

	"github.com/g3n/engine/gls"

	"moonbasic/internal/value"
)

// Retro geometry: PS1-style vertex snapping, affine texture mapping, flat
// shading, and stepped fog in the lit shaders (bsshadow, mbphysical,
// mbterrain; stepped fog also in mbwater). Every effect is off at zero and
// the shaders take the old path untouched, so the default image is the same.
// Post effects (dither, palette, low resolution) live elsewhere.

type retroGeom struct {
	snapW, snapH float32 // virtual resolution for vertex snapping; 0 = off, snapH 0 = window aspect
	affine       float32 // 0 perspective-correct .. 1 fully affine UVs
	flat         bool    // face normals instead of interpolated ones
	fogSteps     int     // quantize the fog factor into N steps; 0 = smooth
}

// setRetroGeom sets all retro geometry state at once (presets use it).
// snapW > 0 with snapH 0 snaps to snapW columns and as many rows as the
// window's aspect gives. Values are clamped to sane ranges.
func (w *World) setRetroGeom(snapW, snapH float32, affine float32, flat bool, fogSteps int) {
	if snapW < 0 || snapW != snapW {
		snapW = 0
	}
	if snapH < 0 || snapH != snapH {
		snapH = 0
	}
	if snapW == 0 {
		snapH = 0
	}
	if affine != affine || affine < 0 {
		affine = 0
	}
	if affine > 1 {
		affine = 1
	}
	if fogSteps < 0 {
		fogSteps = 0
	}
	if fogSteps > 256 {
		fogSteps = 256
	}
	w.retro = retroGeom{snapW: snapW, snapH: snapH, affine: affine, flat: flat, fogSteps: fogSteps}
}

// retroGeomState returns the stored retro geometry state; snapH is 0 when
// the row count follows the window aspect.
func (w *World) retroGeomState() (snapW, snapH, affine float32, flat bool, fogSteps int) {
	r := w.retro
	return r.snapW, r.snapH, r.affine, r.flat, r.fogSteps
}

// retroSnapRes is the grid the vertex shaders snap to, with an automatic
// height resolved from the window size.
func (w *World) retroSnapRes() (float32, float32) {
	sw, sh := w.retro.snapW, w.retro.snapH
	if sw <= 0 {
		return 0, 0
	}
	if sh <= 0 {
		ww, hh := w.scrW, w.scrH
		if w.app != nil {
			ww, hh = w.app.GetSize()
		}
		if ww > 0 && hh > 0 {
			sh = float32(math.Round(float64(sw) * float64(hh) / float64(ww)))
		} else {
			sh = float32(math.Round(float64(sw) * 0.75))
		}
		if sh < 1 {
			sh = 1
		}
	}
	return sw, sh
}

// bindRetroGeom sets the Retro* uniforms on the current program. Programs
// are shared between materials, so it always writes (zeros switch them off).
func (w *World) bindRetroGeom(gs *gls.GLS) {
	if gs == nil {
		return
	}
	sw, sh := w.retroSnapRes()
	u := gls.Uniform{}
	u.Init("RetroSnap")
	if loc := u.Location(gs); loc >= 0 {
		gs.Uniform2f(loc, sw, sh)
	}
	setUni1f(gs, "RetroAffine", w.retro.affine)
	setUni1i(gs, "RetroFlat", bool01(w.retro.flat))
	setUni1i(gs, "RetroFogSteps", w.retro.fogSteps)
}

func (w *World) retroGeomCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetRetroSnap w, h  (0, 0 = off)  |  SetRetroSnap n  (n columns, rows from the window aspect)
		"setretrosnap": n(func(a []value.Value) (value.Value, error) {
			r := w.retro
			sw := float32(argN(a, 0, 0))
			sh := float32(0)
			if len(a) > 1 {
				sh = float32(argN(a, 1, 0))
				if sh <= 0 {
					sw = 0
				}
			}
			w.setRetroGeom(sw, sh, r.affine, r.flat, r.fogSteps)
			return z()
		}),
		// SetRetroAffine amount  (0 perspective-correct .. 1 PS1 warp)
		"setretroaffine": n(func(a []value.Value) (value.Value, error) {
			r := w.retro
			w.setRetroGeom(r.snapW, r.snapH, float32(argN(a, 0, 1)), r.flat, r.fogSteps)
			return z()
		}),
		// SetRetroFlat on
		"setretroflat": n(func(a []value.Value) (value.Value, error) {
			r := w.retro
			w.setRetroGeom(r.snapW, r.snapH, r.affine, argI(a, 0, 1) != 0, r.fogSteps)
			return z()
		}),
		// SetRetroFogSteps n  (0 = smooth)
		"setretrofogsteps": n(func(a []value.Value) (value.Value, error) {
			r := w.retro
			w.setRetroGeom(r.snapW, r.snapH, r.affine, r.flat, argI(a, 0, 0))
			return z()
		}),
	}
}

// retroSnapGLSL goes in a vertex shader after its uniforms. It snaps clip
// xy to the pixel grid of a RetroSnap-sized screen and keeps w, so the
// vertex lands on a whole virtual pixel and jitters as it moves.
const retroSnapGLSL = `
uniform vec2 RetroSnap;
vec4 retroSnapClip(vec4 p) {
    if (RetroSnap.x < 1.0 || RetroSnap.y < 1.0 || p.w <= 0.0) { return p; }
    vec2 hr = RetroSnap * 0.5;
    p.xy = floor(p.xy / p.w * hr + 0.5) / hr * p.w;
    return p;
}
`

// retroFogGLSL is the stepped fog helper on its own (water uses it).
const retroFogGLSL = `
uniform int RetroFogSteps;
// Fog visibility in RetroFogSteps bands.
float retroFog(float f) {
    if (RetroFogSteps <= 0) { return f; }
    float s = float(RetroFogSteps);
    return floor(clamp(f, 0.0, 1.0) * s + 0.5) / s;
}
`

// retroFragGLSL goes in a fragment shader after FragTexcoord and the
// noperspective FragTexcoordAffine inputs.
const retroFragGLSL = retroFogGLSL + `
uniform float RetroAffine;
uniform int RetroFlat;
// The base UV, blended toward the screen-linear (affine) copy.
vec2 retroUV() {
    if (RetroAffine <= 0.0) { return FragTexcoord; }
    return mix(FragTexcoord, FragTexcoordAffine, clamp(RetroAffine, 0.0, 1.0));
}
// The face normal from screen derivatives of pos (any space), turned to the
// same side as the interpolated normal n so two-sided faces stay right.
vec3 retroFlatNormal(vec3 pos, vec3 n) {
    if (RetroFlat == 0) { return n; }
    vec3 fn = cross(dFdx(pos), dFdy(pos));
    if (dot(fn, fn) < 1e-20) { return n; }
    fn = normalize(fn);
    return dot(fn, n) < 0.0 ? -fn : fn;
}
`
