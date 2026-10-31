package runtime

import (
	"fmt"
	"strings"

	"github.com/g3n/engine/material"

	"moonbasic/internal/value"
)

// Unlit and toon are shading modes of the PBR material, so they keep its
// textures, alpha, shadows-received, and fog. EntityFX maps the Blitz flags
// onto the same switches.

const (
	fxFullbright = 1
	fxNoFog      = 8
	fxTwoSided   = 16
)

func shadingMode(name string, n int) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "pbr", "lit", "standard", "0":
		return shadePBR, true
	case "unlit", "fullbright", "flat", "1":
		return shadeUnlit, true
	case "toon", "cel", "cartoon", "2":
		return shadeToon, true
	case "":
		if n >= shadePBR && n <= shadeToon {
			return n, true
		}
	}
	return 0, false
}

// eachPBR runs fn on every PBR material of an entity (converting it first)
// or on a material from CreatePBRMaterial.
func (w *World) eachPBR(id int, where string, fn func(*pbrMat)) error {
	pm, e, err := w.pbrOf(id, true)
	if err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}
	if e != nil {
		w.eachEntityPBR(e, fn)
	} else {
		fn(pm)
	}
	return nil
}

func (m *pbrMat) setToon(bands, rim, spec float32) {
	m.shading = shadeToon
	if bands < 1 {
		bands = 1
	}
	if bands > 16 {
		bands = 16
	}
	m.toonBands = bands
	m.toonRim = clamp01n(rim)
	m.toonSpec = clamp01n(spec)
}

func (w *World) shadingCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetMaterialShading e, "pbr" | "unlit" | "toon"
		"setmaterialshading": n(func(a []value.Value) (value.Value, error) {
			mode, ok := shadingMode(argS(a, 1), argI(a, 1, 0))
			if !ok {
				return value.Value{}, fmt.Errorf("SetMaterialShading: want pbr, unlit, or toon, got %q", argS(a, 1))
			}
			return z2(w.eachPBR(argI(a, 0, 0), "SetMaterialShading", func(m *pbrMat) {
				if mode == shadeToon && m.toonBands <= 0 {
					m.setToon(3, 0.5, 0.6)
					return
				}
				m.shading = mode
			}))
		}),
		"getmaterialshading": n(func(a []value.Value) (value.Value, error) {
			pm, _, err := w.pbrOf(argI(a, 0, 0), false)
			if err != nil || pm == nil {
				return value.Num(0), err
			}
			return value.Num(float64(pm.shading)), nil
		}),
		// SetUnlit e [, on]
		"setunlit": n(func(a []value.Value) (value.Value, error) {
			on := argI(a, 1, 1) != 0
			return z2(w.eachPBR(argI(a, 0, 0), "SetUnlit", func(m *pbrMat) {
				if on {
					m.shading = shadeUnlit
				} else if m.shading == shadeUnlit {
					m.shading = shadePBR
				}
			}))
		}),
		// SetToon e [, bands, rim, spec]
		"settoon": n(func(a []value.Value) (value.Value, error) {
			bands := float32(argN(a, 1, 3))
			rim := float32(argN(a, 2, 0.5))
			spec := float32(argN(a, 3, 0.6))
			return z2(w.eachPBR(argI(a, 0, 0), "SetToon", func(m *pbrMat) {
				m.setToon(bands, rim, spec)
			}))
		}),
		// EntityFX e, flags: 1 fullbright, 8 no fog, 16 two-sided. Other
		// Blitz bits (2 vertex colour, 4 flat shade, 32 force alpha) are ignored.
		"entityfx": n(func(a []value.Value) (value.Value, error) {
			fx := argI(a, 1, 0)
			return z2(w.eachPBR(argI(a, 0, 0), "EntityFX", func(m *pbrMat) {
				if fx&fxFullbright != 0 {
					m.shading = shadeUnlit
				} else if m.shading == shadeUnlit {
					m.shading = shadePBR
				}
				m.noFog = fx&fxNoFog != 0
				side := material.SideFront
				if fx&fxTwoSided != 0 {
					side = material.SideDouble
				}
				m.GetMaterial().SetSide(side)
			}))
		}),
	}
}

// z2 turns a setter's error into the command result.
func z2(err error) (value.Value, error) {
	if err != nil {
		return value.Value{}, err
	}
	return value.Num(0), nil
}
