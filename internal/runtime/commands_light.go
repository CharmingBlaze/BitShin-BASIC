package runtime

import (
	"fmt"
	"strings"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

func (w *World) makeLight(kind, parent int) int {
	col := &math32.Color{1, 1, 1}
	switch kind {
	case 0:
		return w.finishLight(light.NewAmbient(col, 0.8), 0, parent)
	case 2:
		return w.finishLight(light.NewPoint(col, 8), 2, parent)
	case 3:
		return w.finishLight(light.NewSpot(col, 8), 3, parent)
	default:
		id := w.finishLight(light.NewDirectional(col, 1.15), 1, parent)
		w.setLightDir(w.ents[id], 40, 30, 0)
		return id
	}
}

func (w *World) finishLight(node core.INode, kind, parent int) int {
	id := w.addEntity(&Entity{node: node, lgtKind: kind}, parent)
	if c, ok := node.(interface{ SetColor(color *math32.Color) }); ok {
		w.ents[id].lgt = c
	}
	if i, ok := node.(interface{ SetIntensity(float32) }); ok {
		w.ents[id].lgtI = i
	}
	return id
}

// dirLightOffset is the G3N world position of a directional light.
// G3N treats that position as the vector from a fragment toward the sun.
func dirLightOffset(pitch, yaw float64) (float32, float32, float32) {
	pr := pitch * math32.Pi / 180
	yr := yaw * math32.Pi / 180
	x := math32.Sin(float32(yr)) * math32.Cos(float32(pr))
	y := math32.Sin(float32(pr))
	z := math32.Cos(float32(yr)) * math32.Cos(float32(pr))
	return toG3N(x, y, z)
}

func (w *World) setLightDir(e *Entity, pitch, yaw, roll float64) {
	e.pitch, e.yaw, e.roll = float32(pitch), float32(yaw), float32(roll)
	w.applyRot(e)
}

func (w *World) lightCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createambientlight": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.makeLight(0, argI(a, 0, 0)))), nil
		}),
		"createdirectionallight": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.makeLight(1, argI(a, 0, 0)))), nil
		}),
		"createpointlight": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.makeLight(2, argI(a, 0, 0)))), nil
		}),
		"createspotlight": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.makeLight(3, argI(a, 0, 0)))), nil
		}),
		"setlightdirection": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			w.setLightDir(e, argN(a, 1, 45), argN(a, 2, 30), argN(a, 3, 0))
			return z()
		}),
		"setlightcone": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			inner := float32(argN(a, 1, 15))
			outer := float32(argN(a, 2, 45))
			if s, ok := e.node.(*light.Spot); ok {
				s.SetAngularDecay(inner)
				s.SetCutoffAngle(outer)
			}
			return z()
		}),
		"setlightintensity": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.lgtI != nil {
				e.lgtI.SetIntensity(float32(argN(a, 1, 1)))
			}
			return z()
		}),
		"setlightshadow": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			e, err := w.ent(id)
			if err != nil {
				return value.Value{}, err
			}
			e.castShadow = argI(a, 1, 1) != 0
			if e.castShadow && (e.lgtKind == 1 || e.lgtKind == 0) {
				w.shadow.lightID = id
			}
			return z()
		}),
		"enableshadows": need(func(a []value.Value) (value.Value, error) {
			w.shadow.on = argI(a, 0, 1) != 0
			fmt.Println("EnableShadows:", w.shadow.on)
			if w.shadow.size <= 0 {
				w.shadow.size = 2048
			}
			if w.shadow.cascades < 1 {
				w.shadow.cascades = 2
			}
			if w.shadow.pcf < 1 {
				w.shadow.pcf = 3
			}
			if w.shadow.bias <= 0 {
				w.shadow.bias = 0.0025
			}
			w.useShadowMaterials(w.shadow.on)
			return z()
		}),
		"setshadowresolution": n(func(a []value.Value) (value.Value, error) {
			sz := argI(a, 0, 2048)
			if sz < 256 {
				sz = 256
			}
			if sz > 8192 {
				sz = 8192
			}
			w.shadow.size = sz
			return z()
		}),
		"setlightshadowres": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			sz := argI(a, 1, 1024)
			if sz < 256 {
				sz = 256
			}
			if sz > 8192 {
				sz = 8192
			}
			e.shadowRes = sz
			if e.castShadow && (e.lgtKind == 1 || w.shadow.lightID == argI(a, 0, 0)) {
				w.shadow.size = sz
			}
			return z()
		}),
		"shadowmapsize": n(func(a []value.Value) (value.Value, error) {
			sz := argI(a, 0, 2048)
			if sz < 256 {
				sz = 256
			}
			if sz > 8192 {
				sz = 8192
			}
			w.shadow.size = sz
			return z()
		}),
		"shadowcascades": n(func(a []value.Value) (value.Value, error) {
			nCas := argI(a, 0, 2)
			if nCas < 1 {
				nCas = 1
			}
			if nCas > maxShadowCascades {
				nCas = maxShadowCascades
			}
			w.shadow.cascades = nCas
			return z()
		}),
		"setshadowbias": n(func(a []value.Value) (value.Value, error) {
			w.shadow.bias = float32(argN(a, 0, 0.0025))
			if len(a) >= 2 {
				w.shadow.normalBias = float32(argN(a, 1, 1))
			}
			return z()
		}),
		"setshadowquality": n(func(a []value.Value) (value.Value, error) {
			mode := argI(a, 0, 0)
			name := strings.ToLower(strings.TrimSpace(argS(a, 0)))
			switch name {
			case "pcf":
				mode = shadowFilterPCF
			case "pcss":
				mode = shadowFilterPCSS
			case "evsm":
				mode = shadowFilterEVSM
			case "msm":
				mode = shadowFilterMSM
			}
			if mode < 0 {
				mode = 0
			}
			if mode > 3 {
				mode = 3
			}
			w.shadow.filter = mode
			if len(a) >= 2 {
				k := argI(a, 1, 3)
				if k < 1 {
					k = 1
				}
				if k > 9 {
					k = 9
				}
				w.shadow.pcf = k
			}
			if mode == shadowFilterEVSM || mode == shadowFilterMSM {
				w.ensureShadowOn()
			}
			return z()
		}),
		"setshadowpcf": n(func(a []value.Value) (value.Value, error) {
			k := argI(a, 0, 3)
			if k < 1 {
				k = 1
			}
			if k > 9 {
				k = 9
			}
			w.shadow.pcf = k
			return z()
		}),
		"setshadowfilter": n(func(a []value.Value) (value.Value, error) {
			name := strings.ToLower(strings.TrimSpace(argS(a, 0)))
			if name == "" && len(a) > 0 {
				switch argI(a, 0, 0) {
				case 1:
					name = "pcss"
				case 2:
					name = "evsm"
				case 3:
					name = "msm"
				default:
					name = "pcf"
				}
			}
			switch name {
			case "pcss", "1", "true":
				w.shadow.filter = shadowFilterPCSS
			case "evsm", "2":
				w.shadow.filter = shadowFilterEVSM
				w.ensureShadowOn()
			case "msm", "3":
				w.shadow.filter = shadowFilterMSM
				w.ensureShadowOn()
			case "pcf", "0", "false", "":
				w.shadow.filter = shadowFilterPCF
			default:
				w.shadow.filter = shadowFilterPCF
			}
			return z()
		}),
		"setshadowpcss": n(func(a []value.Value) (value.Value, error) {
			if argI(a, 0, 1) != 0 {
				w.shadow.filter = 1
			} else {
				w.shadow.filter = 0
			}
			return z()
		}),
		"setshadowlightsize": n(func(a []value.Value) (value.Value, error) {
			sz := float32(argN(a, 0, 0.04))
			if sz < 0.001 {
				sz = 0.001
			}
			w.shadow.lightSize = sz
			return z()
		}),
		"setshadowevsm": n(func(a []value.Value) (value.Value, error) {
			if argI(a, 0, 1) != 0 {
				w.shadow.filter = shadowFilterEVSM
				w.ensureShadowOn()
			} else if w.shadow.filter == shadowFilterEVSM {
				w.shadow.filter = shadowFilterPCF
			}
			return z()
		}),
		"setshadowmsm": n(func(a []value.Value) (value.Value, error) {
			if argI(a, 0, 1) != 0 {
				w.shadow.filter = shadowFilterMSM
				w.ensureShadowOn()
			} else if w.shadow.filter == shadowFilterMSM {
				w.shadow.filter = shadowFilterPCF
			}
			return z()
		}),
		"enableshadowcache": n(func(a []value.Value) (value.Value, error) {
			w.shadow.cache = argI(a, 0, 1) != 0
			w.shadow.staticReady = false
			return z()
		}),
		"setshadowcache": n(func(a []value.Value) (value.Value, error) {
			w.shadow.cache = argI(a, 0, 1) != 0
			w.shadow.staticReady = false
			return z()
		}),
		"enableshadowcaching": n(func(a []value.Value) (value.Value, error) {
			w.shadow.cache = argI(a, 0, 1) != 0
			w.shadow.staticReady = false
			return z()
		}),
		"enableshadowatlas": n(func(a []value.Value) (value.Value, error) {
			w.shadow.atlas = argI(a, 0, 1) != 0
			w.ensureShadowOn()
			return z()
		}),
		"setshadowatlas": n(func(a []value.Value) (value.Value, error) {
			w.shadow.atlas = argI(a, 0, 1) != 0
			w.ensureShadowOn()
			return z()
		}),
		"enablecontactshadows": n(func(a []value.Value) (value.Value, error) {
			w.shadow.contact = argI(a, 0, 1) != 0
			w.ensureShadowOn()
			return z()
		}),
		"setcontactshadows": n(func(a []value.Value) (value.Value, error) {
			w.shadow.contact = argI(a, 0, 1) != 0
			w.ensureShadowOn()
			return z()
		}),
		"enablescreenspaceshadows": n(func(a []value.Value) (value.Value, error) {
			w.shadow.sss = argI(a, 0, 1)
			w.ensureShadowOn()
			return z()
		}),
		"setscreenspaceshadows": n(func(a []value.Value) (value.Value, error) {
			w.shadow.sss = argI(a, 0, 1)
			w.ensureShadowOn()
			return z()
		}),
		"setsss": n(func(a []value.Value) (value.Value, error) {
			w.shadow.sss = argI(a, 0, 1)
			w.ensureShadowOn()
			return z()
		}),
	}
}
