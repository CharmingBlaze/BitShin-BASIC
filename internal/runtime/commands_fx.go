package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/g3n/engine/camera"

	"bitshinbasic/internal/parse"
	"bitshinbasic/internal/syntax"
	"bitshinbasic/internal/value"
)

func (w *World) fxCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	emCmd := func(fn func(*emitter, []value.Value) (value.Value, error)) cmd {
		return n(func(a []value.Value) (value.Value, error) {
			e := w.emitterOf(argI(a, 0, 0))
			if e == nil {
				return z()
			}
			return fn(e, a)
		})
	}
	return map[string]cmd{
		"createemitter": n(func(a []value.Value) (value.Value, error) {
			if err := w.require(); err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(w.newEmitter(argI(a, 0, 0), w.mode2D))), nil
		}),
		"createparticleemitter": n(func(a []value.Value) (value.Value, error) {
			if err := w.require(); err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(w.newEmitter(argI(a, 0, 0), w.mode2D))), nil
		}),
		"createemitter2d": n(func(a []value.Value) (value.Value, error) {
			if err := w.require(); err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(w.newEmitter(argI(a, 0, 0), true))), nil
		}),
		"emitterrate": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.rate = argN(a, 1, 20)
			return z()
		}),
		"emittermax": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.max = argI(a, 1, 64)
			if e.max < 1 {
				e.max = 1
			}
			return z()
		}),
		"emitterlife": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.life = argN(a, 1, 1)
			return z()
		}),
		"emitterspeed": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.speed = argN(a, 1, 4)
			return z()
		}),
		"emittersize": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.size0 = argN(a, 1, 0.2)
			e.size1 = argN(a, 2, e.size0*0.4)
			if len(a) >= 5 {
				e.size0 = argN(a, 1, 0.2)
				e.tall = argN(a, 2, 1) / mathMax(e.size0, 0.001)
				e.size1 = argN(a, 3, e.size0)
				if argN(a, 4, e.size0) > 0 && e.size1 > 0 {
					e.tall = argN(a, 4, e.size1) / e.size1
				}
			}
			return z()
		}),
		"emittercolor": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			vals := make([]float64, 0, 8)
			for i := 1; i < len(a) && i <= 8; i++ {
				vals = append(vals, argN(a, i, 0))
			}
			e.setColor(vals)
			return z()
		}),
		"emittervelocity": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.vx, e.vy, e.vz = argN(a, 1, 0), argN(a, 2, 1), argN(a, 3, 0)
			return z()
		}),
		"emittergravity": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			if len(a) >= 4 {
				e.gravX, e.gravY, e.gravZ = argN(a, 1, 0), argN(a, 2, -6), argN(a, 3, 0)
			} else {
				e.gravY = argN(a, 1, -6)
			}
			return z()
		}),
		"emitterwind": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.windX, e.windY, e.windZ = argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
			return z()
		}),
		"emittercone": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.cone = argN(a, 1, 30)
			return z()
		}),
		"emitterdrag": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.drag = argN(a, 1, 0)
			return z()
		}),
		"emitterduration": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.duration = argN(a, 1, 0)
			e.elapsed = 0
			return z()
		}),
		"emitterloop": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.looping = argI(a, 1, 1) != 0
			return z()
		}),
		"emitterarea": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.areaX, e.areaY, e.areaZ = argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
			return z()
		}),
		"emitterparticle": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			tex := argI(a, 1, 0)
			e.tex = tex
			if im := w.images[tex]; im != nil {
				e.img = im.img
			}
			return z()
		}),
		"positionemitter": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.x, e.y, e.z = argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
			if e.ent != 0 {
				if ent := w.ents[e.ent]; ent != nil && ent.node != nil {
					gx, gy, gz := toG3N(float32(e.x), float32(e.y), float32(e.z))
					ent.node.GetNode().SetPosition(gx, gy, gz)
					e.x, e.y, e.z = 0, 0, 0
				}
			}
			return z()
		}),
		"emit": n(func(a []value.Value) (value.Value, error) {
			e := w.emitterOf(argI(a, 0, 0))
			if e == nil {
				return value.Value{}, fmt.Errorf("Emit: invalid emitter")
			}
			nburst := argI(a, 1, 1)
			if nburst < 1 {
				nburst = 1
			}
			for i := 0; i < nburst; i++ {
				w.spawnParticle(e)
			}
			return z()
		}),
		"emitterburst": n(func(a []value.Value) (value.Value, error) {
			e := w.emitterOf(argI(a, 0, 0))
			if e == nil {
				return z()
			}
			nburst := argI(a, 1, 16)
			if nburst < 1 {
				nburst = 1
			}
			for i := 0; i < nburst; i++ {
				w.spawnParticle(e)
			}
			return z()
		}),
		"freeemitter": n(func(a []value.Value) (value.Value, error) {
			w.freeEmitter(argI(a, 0, 0))
			return z()
		}),
		"emitterspread": aliasCmd(w, "emittercone"),
		"parentemitter": emCmd(func(e *emitter, a []value.Value) (value.Value, error) {
			e.parent = argI(a, 1, 0)
			return z()
		}),
		"particle2drate":     aliasCmd(w, "emitterrate"),
		"particle2dmax":      aliasCmd(w, "emittermax"),
		"particle2dlife":     aliasCmd(w, "emitterlife"),
		"particle2dspeed":    aliasCmd(w, "emitterspeed"),
		"particle2dsize":     aliasCmd(w, "emittersize"),
		"particle2dcolor":    aliasCmd(w, "emittercolor"),
		"particle2dvelocity": aliasCmd(w, "emittervelocity"),
		"particle2dgravity":  aliasCmd(w, "emittergravity"),
		"particle2dwind":     aliasCmd(w, "emitterwind"),
		"particle2dcone":     aliasCmd(w, "emittercone"),
		"particle2ddrag":     aliasCmd(w, "emitterdrag"),
		"particle2dduration": aliasCmd(w, "emitterduration"),
		"particle2dloop":     aliasCmd(w, "emitterloop"),
		"particle2dburst":    aliasCmd(w, "emitterburst"),
		"particle2dsprite":   aliasCmd(w, "emitterparticle"),
		"positionemitter2d":  aliasCmd(w, "positionemitter"),
		"freeemitter2d":      aliasCmd(w, "freeemitter"),
		"emit2d":             aliasCmd(w, "emit"),
		"createskybox": need(func(a []value.Value) (value.Value, error) {
			id, err := w.makeSkyBox(argS(a, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(id)), nil
		}),
		"loadskybox": need(func(a []value.Value) (value.Value, error) {
			var paths [6]string
			for i := 0; i < 6; i++ {
				paths[i] = argS(a, i)
			}
			faces, err := w.loadSkyFaces(paths)
			if err != nil {
				return value.Value{}, err
			}
			id, err := w.createSkyBoxFromFaces(faces)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(id)), nil
		}),
		"setskybox": need(func(a []value.Value) (value.Value, error) {
			w.setSkyBox(argI(a, 0, 0))
			return z()
		}),
		"freeskybox": need(func(a []value.Value) (value.Value, error) {
			w.freeSkyBox(argI(a, 0, 0))
			return z()
		}),
		"hideskybox": need(func(a []value.Value) (value.Value, error) {
			w.hideSkyBox(false)
			return z()
		}),
		"showskybox": need(func(a []value.Value) (value.Value, error) {
			w.hideSkyBox(true)
			return z()
		}),
		"setskycolor": need(func(a []value.Value) (value.Value, error) {
			c := rgb(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
			w.clear = *c
			return z()
		}),
		"setskypreset": need(func(a []value.Value) (value.Value, error) {
			w.applySkyPreset(argS(a, 0))
			return z()
		}),
		"setsky": need(func(a []value.Value) (value.Value, error) {
			if len(a) >= 1 && a[0].Kind == value.KindStr {
				name := argS(a, 0)
				low := strings.ToLower(name)
				if low == "sunset" || low == "sunset1" || low == "default" || low == "day" {
					w.applySkyPreset(name)
					return z()
				}
				id, err := w.makeSkyBox(name)
				if err != nil {
					return value.Value{}, err
				}
				return value.Num(float64(id)), nil
			}
			if len(a) >= 6 {
				return w.Call("setskygradient", a)
			}
			if len(a) >= 3 {
				return w.Call("setskycolor", a)
			}
			w.applySkyPreset("default")
			return z()
		}),
		"setskygradient": need(func(a []value.Value) (value.Value, error) {
			w.skyTop = *rgb(argN(a, 0, 134), argN(a, 1, 187), argN(a, 2, 214))
			w.skyBot = *rgb(argN(a, 3, 230), argN(a, 4, 230), argN(a, 5, 242))
			id, err := w.createSkyBoxFromFaces(generateSkyFaces(w.skyTop, w.skyBot, -0.5, 0.55, 0.75))
			if err != nil {
				return z()
			}
			w.setSkyBox(id)
			return z()
		}),
		"setweather": need(func(a []value.Value) (value.Value, error) {
			mode := "clear"
			if len(a) > 0 {
				mode = syntax.WeatherMode(a[0])
			}
			w.setWeather(mode)
			return z()
		}),
		"setweathertransition": need(func(a []value.Value) (value.Value, error) {
			mode := "clear"
			if len(a) > 0 {
				mode = syntax.WeatherMode(a[0])
			}
			w.ensureWeather()
			inten := argN(a, 1, w.wx.intensity)
			secs := argN(a, 2, 1.5)
			w.setWeatherTransition(mode, inten, secs)
			return z()
		}),
		"setweatherintensity": need(func(a []value.Value) (value.Value, error) {
			w.setWeatherIntensity(argN(a, 0, 1))
			return z()
		}),
		"setweatherdryingspeed": need(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			n := argN(a, 0, 0.01)
			if n < 0 {
				n = 0
			}
			w.wx.dryingSpeed = n
			w.wx.dryingInit = true
			return z()
		}),
		"setsurfacewetness": need(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			w.wx.wetOwned = true
			n := clamp01(argN(a, 0, 0))
			w.wetness = float32(n)
			return z()
		}),
		"setcamerarain": need(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			w.wx.camRain = argN(a, 0, 1) > 0.5
			return z()
		}),
		"setweatherwind": need(func(a []value.Value) (value.Value, error) {
			w.setWind(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), -1)
			return z()
		}),
		"setwind": need(func(a []value.Value) (value.Value, error) {
			if len(a) >= 4 {
				w.setWind(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 1))
			} else if len(a) == 1 {
				w.ensureWeather()
				w.wx.windStr = argN(a, 0, 1)
			} else {
				w.setWind(argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0), -1)
			}
			return z()
		}),
		"createatmosphere": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.ensureAtmosphere())), nil
		}),
		"setatmosphere": need(func(a []value.Value) (value.Value, error) {
			w.setAtmosphereParams(a)
			return z()
		}),
		"setatmosphererayleigh": need(func(a []value.Value) (value.Value, error) {
			w.ensureAtmosphere()
			if w.atmo != nil {
				w.atmo.rayleigh = float32(argN(a, 0, 1))
			}
			return z()
		}),
		"setatmospheremie": need(func(a []value.Value) (value.Value, error) {
			w.ensureAtmosphere()
			if w.atmo != nil {
				w.atmo.mie = float32(argN(a, 0, 1))
			}
			return z()
		}),
		"setatmosphereturbidity": need(func(a []value.Value) (value.Value, error) {
			w.ensureAtmosphere()
			if w.atmo != nil {
				w.atmo.turbidity = float32(argN(a, 0, 1.2))
			}
			return z()
		}),
		"setsundirection": need(func(a []value.Value) (value.Value, error) {
			w.ensureAtmosphere()
			if w.atmo != nil {
				w.atmo.sunX = float32(argN(a, 0, -0.35))
				w.atmo.sunY = float32(argN(a, 1, 0.62))
				w.atmo.sunZ = float32(argN(a, 2, 0.70))
			}
			return z()
		}),
		"strikelightning": need(func(a []value.Value) (value.Value, error) {
			if len(a) >= 6 {
				w.strikeLightning(argN(a, 0, 0), argN(a, 1, 28), argN(a, 2, 0), argN(a, 3, 0), argN(a, 4, 0.5), argN(a, 5, 0))
			} else if len(a) >= 3 {
				w.strikeLightning(argN(a, 0, 0), argN(a, 1, 28), argN(a, 2, 0), argN(a, 0, 0), 0.4, argN(a, 2, 0))
			} else {
				w.autoStormBolt()
			}
			return z()
		}),
		"setweatherwetness": need(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			w.wx.wetOwned = true
			n := argN(a, 0, 0)
			if n < 0 {
				n = 0
			}
			if n > 1 {
				n = 1
			}
			w.wetness = float32(n)
			return z()
		}),
		"weatherwetness": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.wetness)), nil
		}),
		"setfogheight": need(func(a []value.Value) (value.Value, error) {
			w.fogHeight = float32(argN(a, 0, 0))
			w.fogHFall = float32(argN(a, 1, 0.06))
			if w.fogHFall < 0 {
				w.fogHFall = 0
			}
			return z()
		}),
		"camerafogheight": need(func(a []value.Value) (value.Value, error) {
			off := 0
			if len(a) >= 3 {
				off = 1
			}
			w.fogHeight = float32(argN(a, off, 0))
			w.fogHFall = float32(argN(a, off+1, 0.06))
			return z()
		}),
		"enableheightfog": need(func(a []value.Value) (value.Value, error) {
			if argN(a, 0, 1) > 0.5 {
				if w.fogHFall <= 0 {
					w.fogHFall = 0.06
				}
			} else {
				w.fogHFall = 0
			}
			return z()
		}),
		"setwindsway": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return z()
			}
			e.windSway = argN(a, 1, 1) > 0.5
			e.windPhase = float32(argN(a, 2, float64(argI(a, 0, 0))*0.37))
			return z()
		}),
		"getwindx": n(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			return value.Num(w.wx.windX), nil
		}),
		"getwindy": n(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			return value.Num(w.wx.windY), nil
		}),
		"getwindz": n(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			return value.Num(w.wx.windZ), nil
		}),
		"getwindstrength": n(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			return value.Num(w.wx.windStr), nil
		}),
		"weather": n(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			return value.Str(w.effectiveWeatherMode()), nil
		}),
		"weatherintensity": n(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			return value.Num(w.wx.intensity), nil
		}),
		"setfog": need(func(a []value.Value) (value.Value, error) {
			if len(a) <= 1 || (len(a) == 2 && argN(a, 0, 0) <= 3) {
				w.fogMode = argI(a, 0, 1)
				if len(a) >= 2 && argN(a, 0, 0) <= 3 {
					w.fogMode = argI(a, 0, 1)
				}
				w.wx.fogOwned = false
				w.applyLitShaders()
				return z()
			}
			if len(a) >= 6 {
				w.fogMode = argI(a, 0, 1)
				w.setFog(argN(a, 1, 180), argN(a, 2, 180), argN(a, 3, 190), argN(a, 4, 8), argN(a, 5, 60))
				return z()
			}
			w.setFog(argN(a, 0, 180), argN(a, 1, 180), argN(a, 2, 190), argN(a, 3, 8), argN(a, 4, 60))
			return z()
		}),
		"clearworld": n(func(a []value.Value) (value.Value, error) {
			w.clearWorld()
			return z()
		}),
		"loadscene": n(func(a []value.Value) (value.Value, error) {
			if err := w.loadSceneFile(argS(a, 0)); err != nil {
				return value.Value{}, err
			}
			return z()
		}),
		"openpak": n(func(a []value.Value) (value.Value, error) {
			path, err := w.openPath(argS(a, 0))
			if err != nil {
				path = w.resolve(argS(a, 0))
			}
			p, err := openPakFile(path)
			if err != nil {
				return value.Value{}, err
			}
			if w.pak != nil {
				w.pak.close()
			}
			w.pak = p
			return z()
		}),
		"filetype": n(func(a []value.Value) (value.Value, error) {
			p := w.resolve(argS(a, 0))
			st, err := os.Stat(p)
			if err != nil {
				if w.pak != nil {
					if _, err := w.pak.extract(argS(a, 0)); err == nil {
						return value.Num(1), nil
					}
				}
				return value.Num(0), nil
			}
			if st.IsDir() {
				return value.Num(2), nil
			}
			return value.Num(1), nil
		}),
		"createcameraortho": need(func(a []value.Value) (value.Value, error) {
			return w.createCameraOrtho(argI(a, 0, 0), argN(a, 1, 20))
		}),
		"cameraprojmode": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.cam == nil {
				return value.Value{}, fmt.Errorf("CameraProjMode: not a camera")
			}
			if argI(a, 1, 0) == 1 {
				e.cam.SetProjection(1)
			} else {
				e.cam.SetProjection(0)
			}
			return z()
		}),
	}
}

func aliasCmd(w *World, name string) cmd {
	return func(a []value.Value) (value.Value, error) {
		if w.cmdMap == nil {
			w.cmdMap = w.commandTable()
		}
		fn := w.cmdMap[name]
		if fn == nil {
			return value.Value{}, fmt.Errorf("unknown command %s", name)
		}
		return fn(a)
	}
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (w *World) createCameraOrtho(parent int, size float64) (value.Value, error) {
	aspect := float32(16) / 9
	if w.app != nil {
		ww, hh := w.app.GetSize()
		if hh > 0 {
			aspect = float32(ww) / float32(hh)
		}
	}
	if size <= 0 {
		size = 20
	}
	cam := camera.NewOrthographic(aspect, 0.1, 1000, float32(size), camera.Vertical)
	id := w.addEntity(&Entity{node: cam, cam: cam}, parent)
	if w.cam == nil {
		w.cam = cam
	}
	return value.Num(float64(id)), nil
}

func (w *World) clearWorld() {
	w.clearWeatherEmitters()
	w.clearLightning()
	w.clearAtmosphere()
	w.clearSkies()
	for id, e := range w.ents {
		if e.node != nil {
			n := e.node.GetNode()
			if p := n.Parent(); p != nil {
				p.GetNode().Remove(e.node)
			}
			n.SetVisible(false)
		}
		delete(w.ents, id)
	}
	w.cam = nil
	w.sprites = map[int]*ebiSprite{}
	w.images = map[int]*ebiImage{}
	w.draws = w.draws[:0]
	w.texts = w.texts[:0]
	w.clearHudPrint()
	for _, em := range w.emitters {
		w.releaseEmitterParts(em)
	}
	w.emitters = map[int]*emitter{}
	w.partPool = nil
	w.tiles = map[int]*tileMap{}
	w.rules = nil
	w.phys3 = nil
	w.phys2 = nil
	w.nextID = 1
	w.nextImg = 1
	w.nextEmit = 1
	w.nextTile = 1
	w.nextSky = 1
	w.skyID = 0
}

func (w *World) loadSceneFile(rel string) error {
	ext := strings.ToLower(filepath.Ext(rel))
	if ext == ".json" || ext == ".yaml" || ext == ".yml" {
		var root any
		var err error
		if ext == ".json" {
			root, err = w.readJSONFile(rel)
		} else {
			root, err = w.readYAMLFile(rel)
		}
		if err != nil {
			return fmt.Errorf("LoadScene: %w", err)
		}
		if w.ready && !w.mode2D {
			w.applySceneJSON(root)
		}
		return nil
	}
	if w.runner == nil {
		return fmt.Errorf("LoadScene: interpreter not bound")
	}
	path, err := w.openPath(rel)
	if err != nil {
		path = w.findSceneFile(rel)
	}
	prog, err := parse.ParseFile(path)
	if err != nil {
		return fmt.Errorf("LoadScene: %w", err)
	}
	prog, err = parse.ExpandIncludes(prog, filepath.Dir(path))
	if err != nil {
		return err
	}
	return w.runner.ExecProgram(prog)
}

func (w *World) findSceneFile(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	cwd, _ := os.Getwd()
	cands := []string{w.resolve(rel)}
	for _, root := range []string{w.base, cwd} {
		p := root
		for i := 0; i < 6; i++ {
			cands = append(cands, filepath.Join(p, rel), filepath.Join(p, "examples", rel))
			next := filepath.Dir(p)
			if next == p {
				break
			}
			p = next
		}
	}
	for _, p := range cands {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return w.resolve(rel)
}
