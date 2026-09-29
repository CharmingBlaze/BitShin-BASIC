package runtime

import (
	"fmt"

	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"

	"bitshinbasic/internal/value"
)

func (w *World) pbrCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	setColor := func(a []value.Value) (value.Value, error) {
		id := argI(a, 0, 0)
		pm, e, err := w.pbrOf(id, true)
		if err != nil {
			return value.Value{}, err
		}
		if len(a) == 2 {
			tex := w.texByID(argI(a, 1, 0))
			if tex == nil {
				return value.Value{}, fmt.Errorf("SetAlbedo: texture %d not found", argI(a, 1, 0))
			}
			apply := func(m *pbrMat) { m.SetBaseColorMap(tex) }
			if e != nil {
				w.eachEntityPBR(e, apply)
			} else {
				apply(pm)
			}
			return z()
		}
		c := rgb(argN(a, 1, 255), argN(a, 2, 255), argN(a, 3, 255))
		col := math32.Color4{c.R, c.G, c.B, 1}
		if e != nil {
			col.A = e.tint.A
			if col.A <= 0 {
				col.A = 1
			}
			e.tint.R, e.tint.G, e.tint.B = c.R, c.G, c.B
		} else if pm != nil {
			col.A = pm.albedo.A
			if col.A <= 0 {
				col.A = 1
			}
		}
		apply := func(m *pbrMat) {
			m.albedo.R, m.albedo.G, m.albedo.B = col.R, col.G, col.B
			if m.albedo.A <= 0 {
				m.albedo.A = 1
			}
			m.applyFactors()
		}
		if e != nil {
			w.eachEntityPBR(e, apply)
		} else {
			apply(pm)
		}
		return z()
	}
	setMap := func(kind string) func([]value.Value) (value.Value, error) {
		return func(a []value.Value) (value.Value, error) {
			pm, e, err := w.pbrOf(argI(a, 0, 0), true)
			if err != nil {
				return value.Value{}, err
			}
			var tex *texture.Texture2D
			if tid := argI(a, 1, 0); tid != 0 {
				tex = w.texByID(tid)
				if tex == nil {
					return value.Value{}, fmt.Errorf("%s: texture %d not found", kind, tid)
				}
			}
			apply := func(m *pbrMat) {
				switch kind {
				case "SetNormalMap":
					m.SetNormalMap(tex)
				case "SetMetallicRoughnessMap":
					m.SetMetallicRoughnessMap(tex)
				case "SetEmissiveMap":
					m.SetEmissiveMap(tex)
				case "SetAOMap":
					m.SetOcclusionMap(tex)
				case "SetEnvMap":
					w.setPBREnvMap(m, tex)
				case "SetAlbedoMap":
					m.SetBaseColorMap(tex)
				}
			}
			if e != nil {
				w.eachEntityPBR(e, apply)
			} else {
				apply(pm)
			}
			return z()
		}
	}
	getChan := func(which string) func([]value.Value) (value.Value, error) {
		return func(a []value.Value) (value.Value, error) {
			pm, _, err := w.pbrOf(argI(a, 0, 0), false)
			if err != nil {
				return value.Value{}, err
			}
			if pm == nil {
				return value.Num(0), nil
			}
			var v float32
			switch which {
			case "ar":
				v = pm.albedo.R
			case "ag":
				v = pm.albedo.G
			case "ab":
				v = pm.albedo.B
			case "er":
				v = pm.emissive.R
			case "eg":
				v = pm.emissive.G
			case "eb":
				v = pm.emissive.B
			}
			return value.Num(float64(v) * 255), nil
		}
	}

	m := map[string]cmd{
		"createpbrmaterial": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.createPBRMaterial())), nil
		}),
		"setmaterialpbr": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			on := argI(a, 1, 1) != 0
			if on {
				if w.convertEntityPBR(e) == nil {
					return value.Value{}, fmt.Errorf("SetMaterialPBR: entity %d has no mesh", argI(a, 0, 0))
				}
			} else {
				w.convertEntityPhong(e)
			}
			return z()
		}),
		"enablepbr": need(func(a []value.Value) (value.Value, error) {
			return w.Call("setmaterialpbr", a)
		}),
		"getmaterialpbr": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if e := w.ents[id]; e != nil && e.usePBR && e.pbr != nil {
				return value.Num(1), nil
			}
			if w.pbrLib[id] != nil {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"setmaterial": need(func(a []value.Value) (value.Value, error) {
			if len(a) >= 2 && a[1].Kind == value.KindStr {
				if err := w.applyPhysicsMaterial(argI(a, 0, 0), a[1].Str); err != nil {
					return value.Value{}, err
				}
				return z()
			}
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if !w.applyPBRMaterial(e, argI(a, 1, 0)) {
				return value.Value{}, fmt.Errorf("SetMaterial: PBR material %d not found", argI(a, 1, 0))
			}
			return z()
		}),
		"setalbedo":       n(setColor),
		"setbasecolor":    n(setColor),
		"setalbedomap":    n(setMap("SetAlbedoMap")),
		"setbasecolormap": n(setMap("SetAlbedoMap")),
		"getalbedor":      n(getChan("ar")),
		"getbasecolorr":   n(getChan("ar")),
		"getalbedog":      n(getChan("ag")),
		"getbasecolorg":   n(getChan("ag")),
		"getalbedob":      n(getChan("ab")),
		"getbasecolorb":   n(getChan("ab")),
		"setmetallic": n(func(a []value.Value) (value.Value, error) {
			pm, e, err := w.pbrOf(argI(a, 0, 0), true)
			if err != nil {
				return value.Value{}, err
			}
			v := float32(argN(a, 1, 0))
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			apply := func(m *pbrMat) {
				m.metallic = v
				m.applyFactors()
			}
			if e != nil {
				w.eachEntityPBR(e, apply)
			} else {
				apply(pm)
			}
			return z()
		}),
		"getmetallic": n(func(a []value.Value) (value.Value, error) {
			pm, _, err := w.pbrOf(argI(a, 0, 0), false)
			if err != nil {
				return value.Value{}, err
			}
			if pm == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(pm.metallic)), nil
		}),
		"setroughness": n(func(a []value.Value) (value.Value, error) {
			pm, e, err := w.pbrOf(argI(a, 0, 0), true)
			if err != nil {
				return value.Value{}, err
			}
			v := float32(argN(a, 1, 0.5))
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			apply := func(m *pbrMat) {
				m.roughness = v
				m.applyFactors()
			}
			if e != nil {
				w.eachEntityPBR(e, apply)
			} else {
				apply(pm)
			}
			return z()
		}),
		"getroughness": n(func(a []value.Value) (value.Value, error) {
			pm, _, err := w.pbrOf(argI(a, 0, 0), false)
			if err != nil {
				return value.Value{}, err
			}
			if pm == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(pm.roughness)), nil
		}),
		"setao": n(func(a []value.Value) (value.Value, error) {
			pm, e, err := w.pbrOf(argI(a, 0, 0), true)
			if err != nil {
				return value.Value{}, err
			}
			v := float32(argN(a, 1, 1))
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			apply := func(m *pbrMat) { m.ao = v }
			if e != nil {
				w.eachEntityPBR(e, apply)
			} else {
				apply(pm)
			}
			return z()
		}),
		"getao": n(func(a []value.Value) (value.Value, error) {
			pm, _, err := w.pbrOf(argI(a, 0, 0), false)
			if err != nil {
				return value.Value{}, err
			}
			if pm == nil {
				return value.Num(1), nil
			}
			return value.Num(float64(pm.ao)), nil
		}),
		"setemissive": n(func(a []value.Value) (value.Value, error) {
			pm, e, err := w.pbrOf(argI(a, 0, 0), true)
			if err != nil {
				return value.Value{}, err
			}
			c := rgb(argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0))
			apply := func(m *pbrMat) {
				m.emissive = *c
				m.applyFactors()
			}
			if e != nil {
				w.eachEntityPBR(e, apply)
			} else {
				apply(pm)
			}
			return z()
		}),
		"getemissiver":            n(getChan("er")),
		"getemissiveg":            n(getChan("eg")),
		"getemissiveb":            n(getChan("eb")),
		"setnormalmap":            n(setMap("SetNormalMap")),
		"setmetallicroughnessmap": n(setMap("SetMetallicRoughnessMap")),
		"setmetalroughmap":        n(setMap("SetMetallicRoughnessMap")),
		"setemissivemap":          n(setMap("SetEmissiveMap")),
		"setaomap":                n(setMap("SetAOMap")),
		"setocclusionmap":         n(setMap("SetAOMap")),
		"setenvmap": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				tex := w.texByID(argI(a, 0, 0))
				for _, e := range w.ents {
					if e != nil && e.usePBR {
						w.eachEntityPBR(e, func(m *pbrMat) { w.setPBREnvMap(m, tex) })
					}
				}
				for _, m := range w.pbrLib {
					w.setPBREnvMap(m, tex)
				}
				return z()
			}
			return setMap("SetEnvMap")(a)
		}),
		"setibl": n(func(a []value.Value) (value.Value, error) {
			if len(a) < 2 {
				w.iblOn = argI(a, 0, 1) != 0
				if len(a) >= 1 && argN(a, 0, 1) > 1 {
					w.iblOn = true
					w.iblIntensity = float32(argN(a, 0, 1))
				}
				return z()
			}
			pm, e, err := w.pbrOf(argI(a, 0, 0), true)
			if err != nil {
				return value.Value{}, err
			}
			on := 0
			if argI(a, 1, 1) != 0 {
				on = 1
			}
			apply := func(m *pbrMat) { m.ibl = on }
			if e != nil {
				w.eachEntityPBR(e, apply)
			} else {
				apply(pm)
			}
			return z()
		}),
		"getibl": n(func(a []value.Value) (value.Value, error) {
			if len(a) >= 1 {
				pm, _, err := w.pbrOf(argI(a, 0, 0), false)
				if err != nil {
					return value.Value{}, err
				}
				if pm != nil {
					if pm.ibl >= 0 {
						return value.Num(float64(pm.ibl)), nil
					}
				}
			}
			if w.iblOn {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"setiblintensity": n(func(a []value.Value) (value.Value, error) {
			v := float32(argN(a, 0, 1))
			if v < 0 {
				v = 0
			}
			w.iblIntensity = v
			return z()
		}),
		"getiblintensity": n(func(a []value.Value) (value.Value, error) {
			v := w.iblIntensity
			if v <= 0 {
				v = 1
			}
			return value.Num(float64(v)), nil
		}),
	}
	m["setpbrmaterial"] = m["setmaterial"]
	m["setocclusion"] = m["setao"]
	m["getocclusion"] = m["getao"]
	return m
}
