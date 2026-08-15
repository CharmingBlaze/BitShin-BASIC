package runtime

import (
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

// WaterMaterial is the G3N IMaterial for CreateWater: Standard + mbwater
// uniforms (Time, WaveSpeed, Gerstner trains, FBO/DuDv, softened shadows).
// Update is called from tickWater each Flip.
type WaterMaterial struct {
	*material.Standard
	w         *World
	wb        *waterBody
	Time      float32
	WaveSpeed float32
}

func NewWaterMaterial(w *World, wb *waterBody) *WaterMaterial {
	std := material.NewStandard(&math32.Color{0.02, 0.12, 0.28})
	m := &WaterMaterial{Standard: std, w: w, wb: wb, WaveSpeed: 1}
	m.Configure()
	return m
}

func (m *WaterMaterial) Configure() {
	if m == nil || m.Standard == nil {
		return
	}
	m.SetShader("mbwater")
	m.SetShaderUnique(true)
	m.SetUseLights(material.UseLightNone)
	m.SetSide(material.SideDouble)
	m.SetTransparent(true)
	m.SetOpacity(0.75)
	m.SetBlending(material.BlendNormal)
	m.SetDepthTest(true)
	m.SetDepthMask(false)
	m.SetWireframe(false)
}

func (m *WaterMaterial) GetMaterial() *material.Material {
	if m == nil || m.Standard == nil {
		return nil
	}
	return m.Standard.GetMaterial()
}

func (m *WaterMaterial) Dispose() {
	if m != nil && m.Standard != nil {
		m.Standard.Dispose()
	}
}

// Update advances Time by delta seconds (called each Flip).
func (m *WaterMaterial) Update(delta float32) {
	if m == nil {
		return
	}
	if delta <= 0 {
		delta = 1.0 / 60
	}
	m.Time += delta
	if m.WaveSpeed <= 0 {
		m.WaveSpeed = 1
	}
	if m.wb != nil {
		m.wb.time = m.Time
		spd := m.wb.speed
		if spd <= 0 {
			spd = 0.03
		}
		m.wb.move = float32(mod1(float64(m.Time * spd)))
	}
}

func (m *WaterMaterial) clock() float32 {
	if m == nil {
		return 0
	}
	spd := m.WaveSpeed
	if spd <= 0 {
		spd = 1
	}
	return m.Time * spd
}

func (m *WaterMaterial) RenderSetup(gs *gls.GLS) {
	m.Configure()
	if m.w != nil && m.wb != nil {
		m.w.ensureWaterMaps(m.wb)
	}
	if m.Standard != nil {
		m.Standard.RenderSetup(gs)
	}
	m.Bind(gs)
}

// Bind uploads Time, WaveSpeed, Gerstner trains, maps, and shadow uniforms
// onto the active mbwater program.
func (m *WaterMaterial) Bind(gs *gls.GLS) {
	if m == nil || m.wb == nil || gs == nil {
		return
	}
	wb := m.wb
	setUni1f(gs, "Time", m.Time)
	spd := m.WaveSpeed
	if spd <= 0 {
		spd = 1
	}
	setUni1f(gs, "WaveSpeed", spd)
	setUni1f(gs, "WaveTime", m.clock())
	setUni1i(gs, "WaveCount", wb.nWaves)
	setUni3f(gs, "WaterColor", wb.color.R, wb.color.G, wb.color.B)
	setUni1f(gs, "WaterLevel", wb.y)
	storm := float32(1)
	if m.w != nil && m.w.wx.mode == "storm" {
		storm = 1.7
	}
	setUni1f(gs, "WaveStorm", storm)
	for i := 0; i < 4; i++ {
		wv := wb.waves[i]
		setUni4f(gs, "WaveDir["+waveIdx(i)+"]", wv.dirX, wv.dirZ, wv.steep, wv.amp)
		setUni4f(gs, "WaveLen["+waveIdx(i)+"]", wv.lambda, wv.speed, 0, 0)
	}
	dummy := uint32(0)
	if m.w != nil {
		dummy = m.w.ensureDummyShadowTex()
	}
	refl, refr := wb.reflect.color, wb.refract.color
	if refl == 0 {
		refl = dummy
	}
	if refr == 0 {
		refr = dummy
	}
	dudv, nrm := wb.dudvGL, wb.normGL
	if dudv == 0 {
		dudv = dummy
	}
	if nrm == 0 {
		nrm = dummy
	}
	bindWaterSampler(gs, 6, refr, "WaterRefract")
	bindWaterSampler(gs, 7, refl, "WaterReflect")
	bindWaterSampler(gs, 8, dudv, "WaterDuDv")
	bindWaterSampler(gs, 9, nrm, "WaterNormal")
	setUni1i(gs, "WaterReflectOn", bool01(wb.reflectOn && wb.reflect.color != 0))
	setUni1i(gs, "WaterRefractOn", bool01(wb.refractOn && wb.refract.color != 0))
	setUni1i(gs, "WaterHasDuDv", bool01(wb.dudvGL != 0))
	setUni1i(gs, "WaterHasNormal", bool01(wb.normGL != 0))
	setUni1f(gs, "WaterMove", wb.move)
	setUni1f(gs, "WaterWaveStrength", wb.waveStr)
	setUni1f(gs, "WaterShine", wb.shine)
	setUni1f(gs, "WaterReflectivity", wb.reflectivity)
	setUni4f(gs, "WaterWind", wb.windX, wb.windZ, wb.windStr, 0)
	if m.w != nil && m.w.cam != nil {
		setUni1f(gs, "CamNear", m.w.cam.Near())
		setUni1f(gs, "CamFar", m.w.cam.Far())
		cp := worldPos(m.w.cam.GetNode())
		setUni3f(gs, "CamWorldPos", cp.X, cp.Y, cp.Z)
	} else {
		setUni1f(gs, "CamNear", 0.1)
		setUni1f(gs, "CamFar", 1000)
	}
	depth := wb.refract.depth
	if depth == 0 {
		depth = dummy
	}
	bindWaterSampler(gs, 10, depth, "WaterRefractDepth")
	setUni1i(gs, "WaterHasDepth", bool01(wb.refract.depth != 0 && wb.refractOn))
	sx, sy, sz := float32(0.4), float32(0.8), float32(0.3)
	if m.w != nil {
		sx, sy, sz = m.w.waterSunDir()
	}
	setUni3f(gs, "WaterSunDir", sx, sy, sz)
	setUni3f(gs, "WaterSunColor", 1.0, 0.95, 0.8)
	peak := float32(0)
	for i := 0; i < wb.nWaves && i < 4; i++ {
		peak += wb.waves[i].amp
	}
	peak *= storm
	setUni1f(gs, "WaterPeak", peak)
	if m.w != nil {
		m.w.bindWaterShadows(gs)
		m.bindWakeSSR(gs)
	} else {
		setUni1i(gs, "ShadowEnabled", 0)
		setUni1i(gs, "WaterSSROn", 0)
		setUni1i(gs, "WaterHasWake", 0)
	}
	if m.w != nil {
		m.w.bindFogUniforms(gs)
	}
}

func mod1(v float64) float64 {
	if v < 0 {
		v = -v
	}
	return v - float64(int(v))
}
