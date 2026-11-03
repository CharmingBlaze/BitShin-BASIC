package runtime

import (
	"fmt"
	"strings"

	"moonbasic/internal/value"
)

// Clearcoat, subsurface scattering, height maps, and SetStyle. A game picks its
// look with one call and the engine stays the same: per-material shading for what
// lives on a surface (toon, cel, flat, painted), and the retro screen for what
// lives on the whole picture (PS1, pixelated).

// Style is a look for a whole game. Shading applies to every plain mesh, including
// ones made later. Screen is a SetRetroPreset name.
type styleSpec struct {
	shading int
	screen  string
	pixelW  int
	modern  bool
}

var styles = map[string]styleSpec{
	"realistic": {shading: shadePBR, screen: "off", modern: true},
	"pbr":       {shading: shadePBR, screen: "off", modern: true},
	"toon":      {shading: shadeToon, screen: "off"},
	"cel":       {shading: shadeCel, screen: "off"},
	"flat":      {shading: shadeFlat, screen: "off"},
	"lowpoly":   {shading: shadeLowPoly, screen: "off", modern: true},
	"painted":   {shading: shadePainted, screen: "off", modern: true},
	"unlit":     {shading: shadeUnlit, screen: "off"},
	"ps1":       {shading: shadePBR, screen: "ps1"},
	"n64":       {shading: shadePBR, screen: "n64"},
	"pixel":     {shading: shadePBR, screen: "off", pixelW: 320},
	"gameboy":   {shading: shadePBR, screen: "gameboy"},
	"pico8":     {shading: shadePBR, screen: "pico8"},
	"dos":       {shading: shadePBR, screen: "dos"},
}

// styleName normalises what a script may write: "Low-Poly", "low poly", "lowpoly".
func styleName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("-", "", "_", "", " ", "").Replace(s)
	switch s {
	case "pixelated", "pixelated3d", "pixel3d":
		return "pixel"
	case "handpainted":
		return "painted"
	case "realisticpbr", "default":
		return "realistic"
	case "retro", "retrops1", "playstation":
		return "ps1"
	case "flatshading":
		return "flat"
	case "celshading":
		return "cel"
	case "toonshading", "cartoon":
		return "toon"
	}
	return s
}

// styleTakes is whether SetStyle's shading reaches this entity. Terrain, water, sky,
// clouds and scatter draw with shaders of their own.
func styleTakes(e *Entity) bool {
	if e == nil || e.mesh == nil || e.sky || e.lgtKind != 0 {
		return false
	}
	switch e.name {
	case "terrain", "terrain_root", "water", "clouds", "scatter", treeFarOnly, treeCards:
		return false
	}
	return e.mat == nil || e.mat.Shader() == "bsshadow" || e.mat.Shader() == "standard" || e.usePBR
}

func (w *World) setStyle(name string) error {
	key := styleName(name)
	st, ok := styles[key]
	if !ok {
		return fmt.Errorf("SetStyle: want realistic, lowpoly, toon, cel, painted, ps1, pixel, flat, or unlit, got %q", name)
	}
	if st.modern {
		if err := w.applyLook("modern"); err != nil {
			return err
		}
	}
	if err := w.applyRetroPreset(st.screen); err != nil {
		return err
	}
	if st.pixelW > 0 {
		r := &w.retroScr
		r.lowW, r.lowH, r.nearestTex = st.pixelW, 0, true
		w.post.on = true
	}
	w.styleOn, w.styleShading, w.styleAt = true, st.shading, -1
	w.applyStyle()
	return nil
}

// applyStyle reaches meshes made since the last call. It only walks when one was.
func (w *World) applyStyle() {
	if !w.styleOn || w.styleAt == w.entsAdded {
		return
	}
	w.styleAt = w.entsAdded
	for id, e := range w.ents {
		if !styleTakes(e) || e.styled == w.styleShading+1 {
			continue
		}
		e.styled = w.styleShading + 1
		if w.styleShading == shadePBR && !e.usePBR {
			continue // already what a plain mesh draws as
		}
		mode := w.styleShading
		_ = w.eachPBR(id, "SetStyle", func(m *pbrMat) { m.setShading(mode) })
	}
}

// setShading picks a mode with sensible settings for the modes that have any.
func (m *pbrMat) setShading(mode int) {
	switch {
	case mode == shadeToon && m.toonBands <= 0:
		m.setToon(3, 0.5, 0.6)
	case mode == shadeCel:
		m.setToon(2, 0, 0.8)
	}
	m.shading = mode
}

func (w *World) materialCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetStyle name$: a whole game's look. realistic, lowpoly, toon, cel,
		// painted, flat, unlit, ps1, n64, pixel, gameboy, pico8, or dos. Reaches
		// plain meshes made before or after it; change any one afterwards.
		"setstyle": n(func(a []value.Value) (value.Value, error) {
			if err := w.setStyle(argS(a, 0)); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		// SetClearcoat e, amount# [, roughness#]: a clear lacquer layer over the
		// material, with its own sharp highlight: car paint, varnish, wet plastic.
		"setclearcoat": n(func(a []value.Value) (value.Value, error) {
			amount, rough := clamp01n(float32(argN(a, 1, 1))), clamp01n(float32(argN(a, 2, 0.08)))
			return z2(w.eachPBR(argI(a, 0, 0), "SetClearcoat", func(m *pbrMat) {
				m.clearcoat, m.coatRough = amount, rough
			}))
		}),
		// SetSubsurface e, amount# [, r, g, b]: light passing through: skin, wax,
		// leaves. The color is what it picks up inside (0-255, default a warm red).
		"setsubsurface": n(func(a []value.Value) (value.Value, error) {
			amount := clamp01n(float32(argN(a, 1, 1)))
			c := rgb(argN(a, 2, 255), argN(a, 3, 90), argN(a, 4, 60))
			return z2(w.eachPBR(argI(a, 0, 0), "SetSubsurface", func(m *pbrMat) {
				m.subsurface, m.subColor = amount, *c
			}))
		}),
		// SetHeightMap e, tex [, scale#]: surface depth with no extra geometry, by
		// parallax occlusion. White is high; scale is the depth in UV units (default
		// 0.04). Outlines stay the mesh's own. Tex 0 turns it off.
		"setheightmap": n(func(a []value.Value) (value.Value, error) {
			tid := argI(a, 1, 0)
			tex := w.texByID(tid)
			if tid != 0 && tex == nil {
				return value.Value{}, fmt.Errorf("SetHeightMap: texture %d not found", tid)
			}
			scale := float32(argN(a, 2, 0.04))
			return z2(w.eachPBR(argI(a, 0, 0), "SetHeightMap", func(m *pbrMat) {
				m.heightTex, m.heightScale = tex, scale
			}))
		}),
		// SetOpacity e, n#: 0 is clear, 1 solid.
		"setopacity": n(func(a []value.Value) (value.Value, error) {
			op := clamp01n(float32(argN(a, 1, 1)))
			return z2(w.eachPBR(argI(a, 0, 0), "SetOpacity", func(m *pbrMat) {
				m.albedo.A = op
				m.applyFactors()
			}))
		}),
	}
}
