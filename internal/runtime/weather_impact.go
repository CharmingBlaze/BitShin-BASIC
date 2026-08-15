package runtime

func (w *World) weatherHitPlanes(x, z float64) (ground, water float64, hasWater bool) {
	ground = float64(w.terrainHeight(float32(x), float32(z)))
	if wb := w.currentWater(); wb != nil {
		return ground, float64(w.waterHeight(float32(x), float32(z))), true
	}
	return ground, -1e9, false
}

func (w *World) weatherImpact(x, y, z float64, onWater bool) {
	w.wx.impactN++
	if w.wx.impactN%8 != 0 {
		return
	}
	if onWater {
		if wb := w.currentWater(); wb != nil {
			w.wakeImpulse(wb, float32(x), float32(z), 0.22)
			w.waterSplash(wb, float32(x), float32(y), float32(z))
			return
		}
	}
	w.groundSplashRing(x, y, z)
}

func (w *World) groundSplashRing(x, y, z float64) {
	if w.wx.ringID == 0 {
		w.wx.ringID = w.newEmitter(0, false)
		if e := w.emitters[w.wx.ringID]; e != nil {
			e.rate = 0
			e.max = 48
			e.life = 0.28
			e.size0, e.size1 = 0.05, 0.14
			e.cr0, e.cg0, e.cb0, e.ca0 = 0.7, 0.78, 0.88, 0.55
			e.cr1, e.cg1, e.cb1, e.ca1 = 0.6, 0.68, 0.8, 0
			e.vy = 1.8
			e.gravY = -12
			e.cone = 80
			e.speed = 2.2
		}
	}
	if e := w.emitters[w.wx.ringID]; e != nil {
		e.x, e.y, e.z = x, y+0.04, z
		for i := 0; i < 5; i++ {
			w.spawnParticle(e)
		}
	}
}
