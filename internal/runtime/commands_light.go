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
		pt := light.NewPoint(col, 2.6)
		tuneLocalLight(pt)
		return w.finishLight(pt, 2, parent)
	case 3:
		sp := light.NewSpot(col, 3.4)
		tuneLocalLight(sp)
		return w.finishLight(sp, 3, parent)
	default:
		id := w.finishLight(light.NewDirectional(col, 1.15), 1, parent)
		w.setLightDir(w.ents[id], 40, 30, 0)
		w.ents[id].castShadow = true
		if w.shadow.lightID == 0 {
			w.shadow.lightID = id
		}
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
			on := argI(a, 0, 1) != 0
			if on {
				w.ensureShadowOn()
			} else {
				w.shadow.on = false
				w.shadow.ready = false
				w.shadow.warm = 0
				w.useShadowMaterials(false)
			}
			fmt.Println("EnableShadows:", w.shadow.on)
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
			name := strings.ToLower(strings.TrimSpace(argS(a, 0)))
			if w.applyShadowPreset(name) {
				if len(a) >= 2 {
					k := argI(a, 1, 3)
					if k < 3 {
						k = 3
					}
					if k > 9 {
						k = 9
					}
					w.shadow.pcf = k
				}
				return z()
			}
			mode := argI(a, 0, 0)
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
				if k < 3 {
					k = 3
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
			if k < 3 {
				k = 3
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
		"setshadowsoftness": n(func(a []value.Value) (value.Value, error) {
			w.shadow.softness = float32(argN(a, 0, 1.0))
			return z()
		}),
		"shadowsoftness": n(func(a []value.Value) (value.Value, error) {
			w.shadow.softness = float32(argN(a, 0, 1.0))
			return z()
		}),
		"setshadowcolor": n(func(a []value.Value) (value.Value, error) {
			w.shadow.color = *rgb(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
			return z()
		}),
		"shadowcolor": n(func(a []value.Value) (value.Value, error) {
			w.shadow.color = *rgb(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
			return z()
		}),
		"setshadowfade": n(func(a []value.Value) (value.Value, error) {
			w.shadow.fadeNear = float32(argN(a, 0, 60))
			w.shadow.fadeFar = float32(argN(a, 1, 100))
			return z()
		}),
		"setlightspecular": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.lgtSpec = *rgb(argN(a, 1, 255), argN(a, 2, 255), argN(a, 3, 255))
			e.lgtSpecOn = true
			return z()
		}),
		"setlightambient": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.lgtAmb = *rgb(argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0))
			e.lgtAmbOn = true
			return z()
		}),
		"setlightattenuation": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			w.setLightAttenuation(e, float32(argN(a, 1, 1)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0)), float32(argN(a, 4, 0)))
			return z()
		}),
		"outdoorlighting": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.applyLightEnvironment("outdoor"))), nil
		}),
		"indoorlighting": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.applyLightEnvironment("indoor"))), nil
		}),
		"setlighttemperature": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			w.setLightTemperature(e, argN(a, 1, 6500))
			return z()
		}),
		"setlightenabled": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			w.setLightEnabled(e, argI(a, 1, 1) != 0)
			return z()
		}),
		"getlightintensity": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			cur, _ := lightIntensityOf(e)
			return value.Num(float64(cur)), nil
		}),
		"getlightrange": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(entityLightRange(e))), nil
		}),
		"getlighttype": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(e.lgtKind)), nil
		}),
		"getlightred": need(func(a []value.Value) (value.Value, error) {
			return lightChan(w, a, 0)
		}),
		"getlightgreen": need(func(a []value.Value) (value.Value, error) {
			return lightChan(w, a, 1)
		}),
		"getlightblue": need(func(a []value.Value) (value.Value, error) {
			return lightChan(w, a, 2)
		}),
		"settimeofday": need(func(a []value.Value) (value.Value, error) {
			w.setTimeOfDay(argN(a, 0, 12))
			return value.Num(w.timeOfDay), nil
		}),
		"gettimeofday": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.timeOfDay), nil
		}),
		"createthreepoint": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.createThreePoint())), nil
		}),
		"setlightcookie": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if t := w.texs[argI(a, 1, 0)]; t != nil {
				e.lgtCookie = t.tex
			}
			return z()
		}),
		"setentitynormalmap": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if t := w.texs[argI(a, 1, 0)]; t != nil {
				w.setPhongNormal(e, t.tex)
			}
			return z()
		}),
		"setgammacorrection": n(func(a []value.Value) (value.Value, error) {
			on := float32(0)
			if argI(a, 0, 1) != 0 {
				on = 1
			}
			if w.shaderUnis == nil {
				w.shaderUnis = map[string]shaderUni{}
			}
			w.shaderUnis["GammaOut"] = shaderUni{n: 1, v: [4]float32{on}}
			return value.Num(float64(on)), nil
		}),
		"setlighting": need(func(a []value.Value) (value.Value, error) {
			mode := strings.ToLower(strings.TrimSpace(argS(a, 0)))
			if mode == "" {
				switch argI(a, 0, 0) {
				case 1:
					mode = "outdoor"
				case 2:
					mode = "indoor"
				}
			}
			return value.Num(float64(w.applyLightEnvironment(mode))), nil
		}),
		"getlighting": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.lightEnv)), nil
		}),
		"setlightfalloff": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			mode := strings.ToLower(strings.TrimSpace(argS(a, 1)))
			if mode == "" {
				switch argI(a, 1, 0) {
				case 1:
					mode = "classic"
				case 2:
					mode = "physical"
				default:
					mode = "smooth"
				}
			}
			w.setLightFalloff(e, mode)
			return z()
		}),
		"entityambient": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.mat != nil {
				e.mat.SetAmbientColor(rgb(argN(a, 1, 255), argN(a, 2, 255), argN(a, 3, 255)))
			}
			return z()
		}),
		"entityemissive": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			c := rgb(argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0))
			if e.mat != nil {
				e.mat.SetEmissiveColor(c)
			}
			if e.pbrWrap != nil {
				e.pbrWrap.emissive = *c
				e.pbrWrap.applyFactors()
			} else if e.pbr != nil {
				e.pbr.SetEmissiveFactor(c)
			}
			return z()
		}),
		"setspecularmap": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if t := w.texs[argI(a, 1, 0)]; t != nil {
				w.setPhongMap(e, t.tex, true)
			}
			return z()
		}),
		"setemissionmap": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if t := w.texs[argI(a, 1, 0)]; t != nil {
				w.setPhongMap(e, t.tex, false)
			}
			return z()
		}),
		"entitycastshadow": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			e.meshNoCast = argI(a, 1, 1) == 0
			return z()
		}),
		"entityreceiveshadow": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			on := argI(a, 1, 1) != 0
			e.meshNoRecv = !on
			w.setMeshReceiveShadow(e, on)
			return z()
		}),
		"setshadowdistance": n(func(a []value.Value) (value.Value, error) {
			d := float32(argN(a, 0, 250))
			if d < 40 {
				d = 40
			}
			w.shadow.distance = d
			if w.shadow.fadeFar <= 0 || w.shadow.fadeFar > d {
				w.shadow.fadeFar = d
				w.shadow.fadeNear = d - shadowFadeTail
			}
			return z()
		}),
	}
}

func lightChan(w *World, a []value.Value, ch int) (value.Value, error) {
	e, err := w.ent(argI(a, 0, 0))
	if err != nil {
		return value.Value{}, err
	}
	col, _ := lightRadiance(e.node)
	v := col.R
	if ch == 1 {
		v = col.G
	}
	if ch == 2 {
		v = col.B
	}
	return value.Num(float64(v) * 255), nil
}
