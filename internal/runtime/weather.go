package runtime

import (
	"math"
	"math/rand"
	"strings"

	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type weatherSnap struct {
	windX, windY, windZ, windStr float64
	fogMode                      int
	fogRGB                       math32.Color
	fogNear, fogFar, fogDen      float32
	hFall, hY                    float32
	cloudC, cloudD, cloudS       float32
}

type thunderWait struct {
	wait float64
	dist float64
}

type weatherState struct {
	mode                string
	intensity           float64
	windX, windY, windZ float64
	windStr             float64
	time                float64
	flash               float64
	nextBolt            float64
	ids                 []int
	fogOwned            bool
	prevMode            int
	prevRGB             math32.Color
	prevNear            float32
	prevFar             float32
	prevDen             float32
	ambSaved            bool
	ambRGB              math32.Color
	wetOwned            bool
	bolts               []lightningBolt

	targetMode                            string
	targetIntensity                       float64
	targetWindX, targetWindY, targetWindZ float64
	targetWindStr                         float64
	transitionTimer                       float64
	transitionDuration                    float64
	isTransitioning                       bool
	oldIds                                []int
	from, to                              weatherSnap
	fromIntensity                         float64
	dryingSpeed                           float64
	dryingInit                            bool
	camRain                               bool
	thunder                               []thunderWait
	thunderOnce                           bool
	impactN                               int
	ringID                                int
}

func (w *World) ensureWeather() {
	if w.wx.mode == "" {
		w.wx.mode = "clear"
		w.wx.intensity = 0.85
		w.wx.targetMode = "clear"
		w.wx.targetIntensity = 0.85
		w.wx.camRain = true
	}
	if !w.wx.dryingInit {
		w.wx.dryingSpeed = 0.01
		w.wx.dryingInit = true
	}
}

func normalizeWeatherMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "clear", "rain", "snow", "fog", "storm":
		return mode
	default:
		return "clear"
	}
}

func smoothstep01(t float64) float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

func lerpF64(a, b, t float64) float64 { return a + (b-a)*t }

func lerpF32(a, b float32, t float64) float32 {
	return float32(float64(a) + (float64(b)-float64(a))*t)
}

func weatherLook(mode string, intensity float64) weatherSnap {
	k := intensity
	if k <= 0 {
		k = 0.01
	}
	s := weatherSnap{}
	switch mode {
	case "rain":
		s.windX, s.windZ, s.windStr = 1.2, 0.4, 1.4*k
		s.fogMode = 1
		s.fogRGB = *rgb(110, 125, 145)
		s.fogNear, s.fogFar = 18, 90
		s.hFall, s.hY = 0.045, 1.5
		s.cloudC, s.cloudD, s.cloudS = 0.72, 1.15, 1.1
	case "snow":
		s.windX, s.windZ, s.windStr = 0.6, 0.2, 0.7*k
		s.fogMode = 1
		s.fogRGB = *rgb(180, 190, 205)
		s.fogNear, s.fogFar = 22, 110
		s.hFall, s.hY = 0.02, 2
		s.cloudC, s.cloudD, s.cloudS = 0.68, 1.05, 0.7
	case "fog":
		s.windX, s.windY, s.windStr = 0.2, 0.05, 0.25
		s.fogMode = 2
		s.fogRGB = *rgb(160, 170, 180)
		s.fogNear, s.fogFar = 2, 22+float32(18*(1-k))
		s.fogDen = float32(0.045 + 0.08*k)
		s.hFall, s.hY = float32(0.08+0.12*k), 0.4
		s.cloudC, s.cloudD, s.cloudS = 0.55, 0.9, 0.35
	case "storm":
		s.windX, s.windZ, s.windStr = 3.5, 1.2, 2.6*k
		s.fogMode = 1
		s.fogRGB = *rgb(70, 80, 95)
		s.fogNear, s.fogFar = 10, 55
		s.hFall, s.hY = 0.035, 1
		s.cloudC, s.cloudD, s.cloudS = 0.9, 1.35, 1.6
	default:
		s.cloudC, s.cloudD, s.cloudS = 0.28, 0.85, 0.55
	}
	return s
}

func (w *World) captureWeatherSnap() weatherSnap {
	s := weatherSnap{
		windX: w.wx.windX, windY: w.wx.windY, windZ: w.wx.windZ, windStr: w.wx.windStr,
		fogMode: w.fogMode, fogRGB: w.fogRGB,
		fogNear: w.fogNear, fogFar: w.fogFar, fogDen: w.fogDensity,
		hFall: w.fogHFall, hY: w.fogHeight,
	}
	if c := w.currentCloud(); c != nil {
		s.cloudC, s.cloudD, s.cloudS = c.cover, c.density, c.speed
	} else {
		s.cloudC, s.cloudD, s.cloudS = 0.28, 0.85, 0.55
	}
	return s
}

func (w *World) currentCloud() *cloudLayer {
	if len(w.clouds) == 0 {
		return nil
	}
	if c := w.clouds[w.curCloud]; c != nil {
		return c
	}
	for _, c := range w.clouds {
		if c != nil {
			return c
		}
	}
	return nil
}

func (w *World) setWeather(mode string) {
	w.ensureWeather()
	mode = normalizeWeatherMode(mode)
	// Reapplying the active weather must not manufacture a transition. In
	// particular, a game may configure its own camera fog and then confirm
	// "clear" weather; fading that fog toward clear-weather zero values creates
	// a short black wall that looks like a giant shadow crossing the camera.
	if !w.wx.isTransitioning && mode == w.wx.mode {
		w.wx.targetMode = mode
		w.wx.targetIntensity = w.wx.intensity
		return
	}
	w.setWeatherTransition(mode, w.wx.intensity, 1.5)
}

func (w *World) setWeatherIntensity(n float64) {
	w.ensureWeather()
	n = clamp01(n)
	if w.wx.isTransitioning {
		w.wx.targetIntensity = n
		w.wx.to = weatherLook(w.wx.targetMode, n)
		w.wx.targetWindX, w.wx.targetWindY, w.wx.targetWindZ = w.wx.to.windX, w.wx.to.windY, w.wx.to.windZ
		w.wx.targetWindStr = w.wx.to.windStr
		w.retargetEmitterRates(n)
		return
	}
	w.wx.intensity = n
	w.wx.targetIntensity = n
	w.retargetEmitterRates(n)
}

func (w *World) retargetEmitterRates(k float64) {
	if k <= 0 {
		k = 0.01
	}
	for _, id := range w.wx.ids {
		e := w.emitters[id]
		if e == nil {
			continue
		}
		base := weatherEmitterRate(e, w.effectiveWeatherMode(), k)
		e.rateBase = base
		if !w.wx.isTransitioning {
			e.rate = base
		}
	}
}

func weatherEmitterRate(e *emitter, mode string, k float64) float64 {
	if e == nil {
		return 0
	}
	switch {
	case e.style == 1 && mode == "storm":
		return 420 * k * 1.6
	case e.style == 1:
		return 420 * k
	case e.recycle:
		return 520 * k
	case e.size0 >= 2:
		if mode == "storm" {
			return 18 * k * 0.4
		}
		return 18 * k
	default:
		return 40 * k
	}
}

func (w *World) effectiveWeatherMode() string {
	w.ensureWeather()
	if w.wx.isTransitioning && w.wx.targetMode != "" {
		return w.wx.targetMode
	}
	return w.wx.mode
}

func (w *World) setWeatherTransition(mode string, intensity, duration float64) {
	w.ensureWeather()
	mode = normalizeWeatherMode(mode)
	intensity = clamp01(intensity)
	if duration < 0 {
		duration = 0
	}
	// Never End / app.Exit — missing audio and GL leftovers are non-fatal.

	w.wx.from = w.captureWeatherSnap()
	w.wx.fromIntensity = w.wx.intensity
	w.wx.to = weatherLook(mode, intensity)
	w.wx.targetMode = mode
	w.wx.targetIntensity = intensity
	w.wx.targetWindX, w.wx.targetWindY, w.wx.targetWindZ = w.wx.to.windX, w.wx.to.windY, w.wx.to.windZ
	w.wx.targetWindStr = w.wx.to.windStr
	w.wx.flash = 0
	w.wx.nextBolt = 1.5 + rand.Float64()*3
	if mode == "storm" {
		w.wx.nextBolt = 0.8 + rand.Float64()*1.6
	}

	if len(w.wx.oldIds) > 0 {
		w.freeEmitterList(w.wx.oldIds)
		w.wx.oldIds = w.wx.oldIds[:0]
	}
	w.wx.oldIds = append(w.wx.oldIds[:0], w.wx.ids...)
	w.wx.ids = w.wx.ids[:0]
	for _, id := range w.wx.oldIds {
		if e := w.emitters[id]; e != nil && e.rateBase <= 0 {
			e.rateBase = e.rate
			e.ca0Base, e.ca1Base = e.ca0, e.ca1
		}
	}

	w.wx.transitionTimer = 0
	w.wx.transitionDuration = duration
	w.wx.isTransitioning = duration > 0.001
	w.prepareTargetEmitters(mode, intensity)
	if w.wx.to.fogMode != 0 {
		w.ownFog(float64(w.wx.from.fogRGB.R)*255, float64(w.wx.from.fogRGB.G)*255, float64(w.wx.from.fogRGB.B)*255, w.wx.from.fogNear, w.wx.from.fogFar)
	}
	if !w.wx.isTransitioning {
		w.finishWeatherTransition()
	}
}

func (w *World) prepareTargetEmitters(mode string, k float64) {
	if k <= 0 {
		k = 0.01
	}
	switch mode {
	case "rain":
		w.addWeatherRain(k, 1)
	case "snow":
		w.addWeatherSnow(k)
	case "fog":
		w.addWeatherWisps(k)
	case "storm":
		w.addWeatherRain(k, 1.6)
		w.addWeatherAsh(k)
		w.addWeatherWisps(k * 0.4)
	}
	for _, id := range w.wx.ids {
		e := w.emitters[id]
		if e == nil {
			continue
		}
		e.rateBase = e.rate
		e.ca0Base, e.ca1Base = e.ca0, e.ca1
		if w.wx.isTransitioning {
			e.rate = e.rateBase * 0.04
			e.ca0 = e.ca0Base * 0.45
			e.ca1 = e.ca1Base * 0.45
		}
	}
}

func (w *World) finishWeatherTransition() {
	w.freeEmitterList(w.wx.oldIds)
	w.wx.oldIds = w.wx.oldIds[:0]
	w.wx.mode = w.wx.targetMode
	w.wx.intensity = w.wx.targetIntensity
	w.wx.windX, w.wx.windY, w.wx.windZ = w.wx.targetWindX, w.wx.targetWindY, w.wx.targetWindZ
	w.wx.windStr = w.wx.targetWindStr
	w.applyWeatherSnap(w.wx.to)
	w.wx.isTransitioning = false
	w.wx.transitionTimer = 0
	for _, id := range w.wx.ids {
		if e := w.emitters[id]; e != nil {
			e.rate = e.rateBase
			e.ca0, e.ca1 = e.ca0Base, e.ca1Base
		}
	}
}

func (w *World) freeEmitterList(ids []int) {
	for _, id := range ids {
		w.freeEmitter(id)
	}
}

func (w *World) clearWeatherEmitters() {
	w.freeEmitterList(w.wx.oldIds)
	w.wx.oldIds = w.wx.oldIds[:0]
	w.freeEmitterList(w.wx.ids)
	w.wx.ids = w.wx.ids[:0]
}

func (w *World) applyWeatherSnap(s weatherSnap) {
	w.wx.windX, w.wx.windY, w.wx.windZ, w.wx.windStr = s.windX, s.windY, s.windZ, s.windStr
	if s.fogMode == 0 {
		w.releaseOwnedFog()
		w.fogHFall = s.hFall
		w.fogHeight = s.hY
	} else {
		w.ownFog(float64(s.fogRGB.R)*255, float64(s.fogRGB.G)*255, float64(s.fogRGB.B)*255, s.fogNear, s.fogFar)
		w.fogMode = s.fogMode
		w.fogDensity = s.fogDen
		w.fogHFall = s.hFall
		w.fogHeight = s.hY
		w.applyFogShader()
	}
	w.syncWeatherClouds(s.cloudC, s.cloudD, s.cloudS)
}

func (w *World) lerpAtmosphere(t float64) {
	a, b := w.wx.from, w.wx.to
	w.wx.windX = lerpF64(a.windX, b.windX, t)
	w.wx.windY = lerpF64(a.windY, b.windY, t)
	w.wx.windZ = lerpF64(a.windZ, b.windZ, t)
	w.wx.windStr = lerpF64(a.windStr, b.windStr, t)
	w.wx.intensity = lerpF64(w.wx.fromIntensity, w.wx.targetIntensity, t)
	if b.fogMode == 0 && t >= 0.999 {
		w.releaseOwnedFog()
		w.fogHFall = lerpF32(a.hFall, b.hFall, t)
		w.fogHeight = lerpF32(a.hY, b.hY, t)
	} else {
		col := math32.Color{
			lerpF32(a.fogRGB.R, b.fogRGB.R, t),
			lerpF32(a.fogRGB.G, b.fogRGB.G, t),
			lerpF32(a.fogRGB.B, b.fogRGB.B, t),
		}
		near := lerpF32(a.fogNear, b.fogNear, t)
		far := lerpF32(a.fogFar, b.fogFar, t)
		if far <= near {
			far = near + 1
		}
		if !w.wx.fogOwned {
			w.wx.prevMode = w.fogMode
			w.wx.prevRGB = w.fogRGB
			w.wx.prevNear, w.wx.prevFar = w.fogNear, w.fogFar
			w.wx.prevDen = w.fogDensity
			w.wx.fogOwned = true
		}
		if b.fogMode != 0 {
			w.fogMode = b.fogMode
		} else if t < 0.85 {
			w.fogMode = a.fogMode
			if w.fogMode == 0 {
				w.fogMode = 1
			}
		}
		w.fogRGB = col
		w.fogNear, w.fogFar = near, far
		w.fogDensity = lerpF32(a.fogDen, b.fogDen, t)
		w.fogHFall = lerpF32(a.hFall, b.hFall, t)
		w.fogHeight = lerpF32(a.hY, b.hY, t)
	}
	w.syncWeatherClouds(
		lerpF32(a.cloudC, b.cloudC, t),
		lerpF32(a.cloudD, b.cloudD, t),
		lerpF32(a.cloudS, b.cloudS, t),
	)
}

func (w *World) crossfadeEmitterRates(t float64) {
	outK := 1 - t
	inK := t
	if inK < 0.04 {
		inK = 0.04
	}
	for _, id := range w.wx.oldIds {
		if e := w.emitters[id]; e != nil {
			base := e.rateBase
			if base <= 0 {
				base = e.rate
			}
			e.rate = base * outK
			if e.ca0Base > 0 {
				e.ca0 = e.ca0Base * (0.35 + 0.65*outK)
				e.ca1 = e.ca1Base * (0.35 + 0.65*outK)
			}
		}
	}
	for _, id := range w.wx.ids {
		if e := w.emitters[id]; e != nil {
			base := e.rateBase
			if base <= 0 {
				base = e.rate
			}
			e.rate = base * inK
			if e.ca0Base > 0 {
				e.ca0 = e.ca0Base * (0.4 + 0.6*t)
				e.ca1 = e.ca1Base * (0.4 + 0.6*t)
			}
		}
	}
}

func (w *World) applyWeatherLook() {
	s := weatherLook(w.wx.mode, w.wx.intensity)
	w.wx.targetMode = w.wx.mode
	w.wx.targetIntensity = w.wx.intensity
	w.wx.targetWindX, w.wx.targetWindY, w.wx.targetWindZ = s.windX, s.windY, s.windZ
	w.wx.targetWindStr = s.windStr
	w.applyWeatherSnap(s)
	k := w.wx.intensity
	if k <= 0 {
		k = 0.01
	}
	switch w.wx.mode {
	case "rain":
		w.addWeatherRain(k, 1)
	case "snow":
		w.addWeatherSnow(k)
	case "fog":
		w.addWeatherWisps(k)
	case "storm":
		w.addWeatherRain(k, 1.6)
		w.addWeatherAsh(k)
		w.addWeatherWisps(k * 0.4)
	}
	for _, id := range w.wx.ids {
		if e := w.emitters[id]; e != nil {
			e.rateBase = e.rate
			e.ca0Base, e.ca1Base = e.ca0, e.ca1
		}
	}
}

func (w *World) ownFog(r, g, b float64, near, far float32) {
	if !w.wx.fogOwned {
		w.wx.prevMode = w.fogMode
		w.wx.prevRGB = w.fogRGB
		w.wx.prevNear, w.wx.prevFar = w.fogNear, w.fogFar
		w.wx.prevDen = w.fogDensity
		w.wx.fogOwned = true
	}
	w.fogMode = 1
	w.fogRGB = *rgb(r, g, b)
	w.fogNear, w.fogFar = near, far
	w.applyFogShader()
}

func (w *World) releaseOwnedFog() {
	if !w.wx.fogOwned {
		return
	}
	w.fogMode = w.wx.prevMode
	w.fogRGB = w.wx.prevRGB
	w.fogNear, w.fogFar = w.wx.prevNear, w.wx.prevFar
	w.fogDensity = w.wx.prevDen
	w.wx.fogOwned = false
	w.applyFogShader()
}

func (w *World) applyFogShader() {
	w.applyLitShaders()
}

func fogFactor(mode int, dist, near, far, density float32) float32 {
	if mode == 0 {
		return 1
	}
	var f float32
	switch mode {
	case 1:
		span := far - near
		if span < 0.0001 {
			span = 0.0001
		}
		f = (far - dist) / span
	case 2:
		f = float32(powE(-float64(density * dist)))
	default:
		d := density * dist
		f = float32(powE(-float64(d * d)))
	}
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func powE(x float64) float64 { return math.Exp(x) }

func (w *World) setCameraFogMode(a []value.Value) {
	mode := argI(a, 0, 0)
	if len(a) >= 2 {
		mode = argI(a, 1, 0)
	}
	if mode < 0 {
		mode = 0
	}
	if mode > 3 {
		mode = 3
	}
	w.fogMode = mode
	w.wx.fogOwned = false
	w.applyLitShaders()
}

func (w *World) setCameraFogColor(a []value.Value) {
	r, g, b := argN(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0)
	if len(a) >= 4 {
		r, g, b = argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
	}
	w.fogRGB = *rgb(r, g, b)
	w.wx.fogOwned = false
}

func (w *World) setCameraFogRange(a []value.Value) {
	near, far := argN(a, 0, 1), argN(a, 1, 100)
	if len(a) >= 3 {
		near, far = argN(a, 1, 1), argN(a, 2, 100)
	}
	if near < 0 {
		near = 0
	}
	if far <= near {
		far = near + 1
	}
	w.fogNear, w.fogFar = float32(near), float32(far)
	w.wx.fogOwned = false
}

func (w *World) setCameraFogDensity(a []value.Value) {
	d := argN(a, 0, 0.025)
	if len(a) >= 2 {
		d = argN(a, 1, 0.025)
	}
	if d < 0 {
		d = 0
	}
	w.fogDensity = float32(d)
	w.wx.fogOwned = false
}

func (w *World) addWeatherRain(k, storm float64) {
	id := w.newEmitter(0, false)
	e := w.emitters[id]
	e.weather = true
	e.follow = true
	e.areaX, e.areaY, e.areaZ = 22, 3, 22
	e.x, e.y, e.z = 0, 16, 0
	e.style = 1 // streak
	e.rate = 420 * k * storm
	e.max = int(520 * k * storm)
	if e.max < 100 {
		e.max = 100
	}
	e.life = 1.45
	e.speed = 18 * storm
	e.vx, e.vy, e.vz = w.wx.targetWindX*0.18, -1, w.wx.targetWindZ*0.18
	e.gravY = -36
	e.windX, e.windY, e.windZ = w.wx.targetWindX, 0, w.wx.targetWindZ
	e.cone = 6
	e.size0, e.size1 = 0.07, 0.05
	e.tall = 22
	e.alignY = true
	e.cr0, e.cg0, e.cb0, e.ca0 = 0.72, 0.82, 0.95, 0.78
	e.cr1, e.cg1, e.cb1, e.ca1 = 0.6, 0.7, 0.88, 0.12
	e.drag = 0
	e.rateBase = e.rate
	e.ca0Base, e.ca1Base = e.ca0, e.ca1
	w.wx.ids = append(w.wx.ids, id)
}

func (w *World) addWeatherSnow(k float64) {
	id := w.newEmitter(0, false)
	e := w.emitters[id]
	e.weather = true
	e.follow = true
	e.recycle = true
	e.style = 2 // flake (mbflake) — not rain streaks
	e.areaX, e.areaY, e.areaZ = 26, 8, 26
	e.x, e.y, e.z = 0, 11, 0
	e.rate = 520 * k
	e.max = int(720 * k)
	if e.max < 280 {
		e.max = 280
	}
	e.life = 9
	e.speed = 0.72
	e.vx, e.vy, e.vz = w.wx.targetWindX*0.35, -1, w.wx.targetWindZ*0.35
	e.gravY = -0.28
	e.windX, e.windY, e.windZ = w.wx.targetWindX, 0.12, w.wx.targetWindZ
	e.cone = 42
	e.size0, e.size1 = 0.58, 0.44
	e.tall = 1
	e.cr0, e.cg0, e.cb0, e.ca0 = 1, 1, 1, 0.98
	e.cr1, e.cg1, e.cb1, e.ca1 = 0.92, 0.95, 1, 0.35
	e.drag = 0.22
	e.rateBase = e.rate
	e.ca0Base, e.ca1Base = e.ca0, e.ca1
	w.wx.ids = append(w.wx.ids, id)
	n := e.max * 2 / 3
	if n < 180 {
		n = 180
	}
	w.seedEmitter(e, n)
}

func (w *World) addWeatherWisps(k float64) {
	id := w.newEmitter(0, false)
	e := w.emitters[id]
	e.weather = true
	e.follow = true
	e.style = 2
	e.areaX, e.areaY, e.areaZ = 16, 4, 16
	e.x, e.y, e.z = 0, 2.2, 0
	e.rate = 18 * k
	e.max = int(56 * k)
	if e.max < 12 {
		e.max = 12
	}
	e.life = 5.5
	e.speed = 0.4
	e.vx, e.vy, e.vz = 1, 0.12, 0.2
	e.gravY = 0.04
	e.windX, e.windY, e.windZ = w.wx.targetWindX, 0.1, w.wx.targetWindZ
	e.cone = 80
	e.size0, e.size1 = 2.4, 3.4
	e.cr0, e.cg0, e.cb0, e.ca0 = 0.78, 0.81, 0.86, 0.2
	e.cr1, e.cg1, e.cb1, e.ca1 = 0.72, 0.76, 0.82, 0
	e.drag = 0.2
	e.rateBase = e.rate
	e.ca0Base, e.ca1Base = e.ca0, e.ca1
	w.wx.ids = append(w.wx.ids, id)
}

func (w *World) addWeatherAsh(k float64) {
	id := w.newEmitter(0, false)
	e := w.emitters[id]
	e.weather = true
	e.follow = true
	e.style = 2
	e.areaX, e.areaY, e.areaZ = 18, 6, 18
	e.x, e.y, e.z = 0, 10, 0
	e.rate = 40 * k
	e.max = int(90 * k)
	if e.max < 16 {
		e.max = 16
	}
	e.life = 4.2
	e.speed = 2.2
	e.vx, e.vy, e.vz = 0.4, -0.35, 0.15
	e.gravY = -1.1
	e.windX, e.windY, e.windZ = w.wx.targetWindX, 0.2, w.wx.targetWindZ
	e.cone = 50
	e.size0, e.size1 = 0.12, 0.06
	e.tall = 1
	e.cr0, e.cg0, e.cb0, e.ca0 = 0.22, 0.22, 0.24, 0.7
	e.cr1, e.cg1, e.cb1, e.ca1 = 0.12, 0.12, 0.14, 0
	e.drag = 0.35
	e.rateBase = e.rate
	e.ca0Base, e.ca1Base = e.ca0, e.ca1
	w.wx.ids = append(w.wx.ids, id)
}

func (w *World) syncWeatherClouds(cover, density, speed float32) {
	if w.scene == nil || !w.ready {
		return
	}
	if len(w.clouds) == 0 {
		w.ensureWeatherClouds()
	}
	c := w.currentCloud()
	if c != nil {
		c.cover = cover * float32(0.55+0.45*w.wx.intensity)
		c.density = density
		c.speed = speed
	}
}

func (w *World) ensureWeatherClouds() {
	if w.scene == nil || !w.ready {
		return
	}
	if len(w.clouds) > 0 {
		return
	}
	w.createVolumetricClouds(110, 280)
}

func (w *World) setFog(r, g, b, near, far float64) {
	w.wx.fogOwned = false
	w.fogMode = 1
	w.fogRGB = *rgb(r, g, b)
	if near <= 0 {
		near = 1
	}
	if far <= near {
		far = near + 10
	}
	w.fogNear, w.fogFar = float32(near), float32(far)
	w.applyFogShader()
}

func (w *World) setWind(dx, dy, dz, strength float64) {
	w.ensureWeather()
	w.wx.windX, w.wx.windY, w.wx.windZ = dx, dy, dz
	w.wx.targetWindX, w.wx.targetWindY, w.wx.targetWindZ = dx, dy, dz
	if strength < 0 {
		strength = math.Sqrt(dx*dx + dy*dy + dz*dz)
	}
	w.wx.windStr = strength
	w.wx.targetWindStr = strength
}

func precipRateFor(mode string, intensity float64) float64 {
	switch mode {
	case "rain":
		return 0.18 * intensity
	case "storm":
		return 0.28 * intensity
	default:
		return 0
	}
}

func (w *World) tickWetness(dt float64) {
	if w.wx.wetOwned {
		return
	}
	mode := w.wx.mode
	inten := w.wx.intensity
	if w.wx.isTransitioning {
		t := 0.0
		if w.wx.transitionDuration > 0 {
			t = smoothstep01(w.wx.transitionTimer / w.wx.transitionDuration)
		}
		p0 := precipRateFor(mode, inten)
		p1 := precipRateFor(w.wx.targetMode, w.wx.targetIntensity)
		rate := lerpF64(p0, p1, t)
		if rate > 0 {
			w.wetness += float32(rate * dt)
		} else {
			w.wetness -= float32(w.wx.dryingSpeed * dt)
		}
	} else if p := precipRateFor(mode, inten); p > 0 {
		w.wetness += float32(p * dt)
	} else {
		w.wetness -= float32(w.wx.dryingSpeed * dt)
	}
	if w.wetness < 0 {
		w.wetness = 0
	}
	if w.wetness > 1 {
		w.wetness = 1
	}
}

func (w *World) tickWeather(dt float32) {
	w.ensureWeather()
	d := float64(dt)
	w.wx.time += d
	if w.wx.isTransitioning {
		w.wx.transitionTimer += d
		t := 1.0
		if w.wx.transitionDuration > 0 {
			t = w.wx.transitionTimer / w.wx.transitionDuration
		}
		if t > 1 {
			t = 1
		}
		s := smoothstep01(t)
		w.lerpAtmosphere(s)
		w.crossfadeEmitterRates(s)
		if t >= 1 {
			w.finishWeatherTransition()
		}
	}
	w.tickWetness(d)
	storming := w.wx.mode == "storm" || (w.wx.isTransitioning && w.wx.targetMode == "storm")
	if storming {
		w.wx.nextBolt -= d
		if w.wx.nextBolt <= 0 {
			w.wx.flash = 0.85 + rand.Float64()*0.4
			w.wx.nextBolt = 2.2 + rand.Float64()*5
			w.autoStormBolt()
		}
	}
	if w.wx.flash > 0 {
		w.wx.flash = math.Max(0, w.wx.flash-d*3.2)
	}
	w.tickLightning(dt)
	w.tickThunder(d)
	w.tickAtmosphere()
	w.tickWindSway(dt)
	w.applyWeatherAmbient()
	for _, id := range w.wx.ids {
		w.syncEmitterWind(id)
	}
	for _, id := range w.wx.oldIds {
		w.syncEmitterWind(id)
	}
}

func (w *World) syncEmitterWind(id int) {
	if e := w.emitters[id]; e != nil {
		e.windX, e.windZ = w.wx.windX, w.wx.windZ
		if e.style != 2 {
			e.windY = w.wx.windY
		}
	}
}

func (w *World) applyWeatherAmbient() {
	if w.ambient == nil {
		return
	}
	if !w.wx.ambSaved {
		c := w.ambient.Color()
		w.wx.ambRGB = c
		w.wx.ambSaved = true
	}
	tint := func(mode string, cr, cg, cb float32) (float32, float32, float32) {
		switch mode {
		case "storm", "rain":
			return cr * 0.75, cg * 0.78, cb * 0.85
		case "fog":
			return cr*0.9 + 0.05, cg*0.9 + 0.05, cb*0.9 + 0.06
		case "snow":
			return cr*0.92 + 0.08, cg*0.93 + 0.08, cb*0.96 + 0.1
		default:
			return cr, cg, cb
		}
	}
	cr, cg, cb := w.wx.ambRGB.R, w.wx.ambRGB.G, w.wx.ambRGB.B
	ar, ag, ab := tint(w.wx.mode, cr, cg, cb)
	if w.wx.isTransitioning {
		br, bg, bb := tint(w.wx.targetMode, cr, cg, cb)
		t := 0.0
		if w.wx.transitionDuration > 0 {
			t = smoothstep01(w.wx.transitionTimer / w.wx.transitionDuration)
		}
		ar = lerpF32(ar, br, t)
		ag = lerpF32(ag, bg, t)
		ab = lerpF32(ab, bb, t)
	}
	if w.wx.flash > 0 {
		ar += float32(w.wx.flash)
		ag += float32(w.wx.flash)
		ab += float32(w.wx.flash * 0.95)
	}
	w.ambient.SetColor(&math32.Color{ar, ag, ab})
}

func (w *World) tickWindSway(dt float32) {
	str := w.wx.windStr
	if str <= 0 {
		str = math.Sqrt(w.wx.windX*w.wx.windX + w.wx.windZ*w.wx.windZ)
	}
	for _, e := range w.ents {
		if e == nil || !e.windSway || e.node == nil {
			continue
		}
		t := w.wx.time + float64(e.windPhase)
		leanP := float32(math.Sin(t*2.1)*4.2+math.Sin(t*3.4)*1.6) * float32(str)
		leanR := float32(math.Cos(t*1.7)*3.1) * float32(str)
		n := e.node.GetNode()
		n.SetRotationX((e.pitch + leanP) * math32.Pi / 180)
		n.SetRotationY(-e.yaw * math32.Pi / 180)
		n.SetRotationZ(-(e.roll + leanR) * math32.Pi / 180)
	}
}

func (w *World) weatherClearMix() (r, g, b float32, ok bool) {
	if w.wx.flash <= 0 {
		return 0, 0, 0, false
	}
	f := float32(w.wx.flash)
	return f, f, f * 0.95, true
}

func (w *World) cameraRainAmount() float32 {
	if !w.wx.camRain {
		return 0
	}
	mode := w.wx.mode
	inten := w.wx.intensity
	if w.wx.isTransitioning {
		t := 0.0
		if w.wx.transitionDuration > 0 {
			t = smoothstep01(w.wx.transitionTimer / w.wx.transitionDuration)
		}
		a := float32(0)
		b := float32(0)
		if mode == "rain" || mode == "storm" {
			a = float32(inten)
		}
		if w.wx.targetMode == "rain" || w.wx.targetMode == "storm" {
			b = float32(w.wx.targetIntensity)
		}
		inten = float64(lerpF32(a, b, t))
	} else if mode != "rain" && mode != "storm" {
		return 0
	}
	if inten <= 0.02 {
		return 0
	}
	wet := w.wetness
	if wet < 0.2 {
		wet = 0.2
	}
	return float32(inten) * wet
}
