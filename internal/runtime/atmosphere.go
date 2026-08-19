package runtime

import (
	"image"
	"image/color"
	"math"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"

	"bitshinbasic/internal/value"
)

// Simplified Bruneton-style atmosphere (GLSL 330).
// Full 4D inscatter tables from ebruneton/precomputed_atmospheric_scattering
// are not shipped (huge offline bake, typically GL 4.x sampling). We precompute
// a 256×64 transmittance LUT in Go and march single scattering in the shader.

const (
	atmoLUTW = 256
	atmoLUTH = 64
	earthRkm = 6360.0
	atmoHkm  = 80.0
	hrKm     = 8.0
	hmKm     = 1.2
)

type atmoDome struct {
	ent       int
	mesh      *graphic.Mesh
	mat       *atmoMat
	tex       *texture.Texture2D
	sunX      float32
	sunY      float32
	sunZ      float32
	rayleigh  float32
	mie       float32
	turbidity float32
	exposure  float32
	ground    math32.Color
}

type atmoMat struct {
	*material.Standard
	w *World
	a *atmoDome
}

func (m *atmoMat) GetMaterial() *material.Material { return m.Standard.GetMaterial() }
func (m *atmoMat) Dispose()                        { m.Standard.Dispose() }

func (m *atmoMat) RenderSetup(gs *gls.GLS) {
	m.Standard.RenderSetup(gs)
	if m.w == nil || m.a == nil || gs == nil {
		return
	}
	a := m.a
	cam := math32.Vector3{}
	if m.w.cam != nil {
		m.w.cam.WorldPosition(&cam)
	}
	setUni3f(gs, "AtmoCam", cam.X, cam.Y, cam.Z)
	setUni3f(gs, "AtmoSun", a.sunX, a.sunY, a.sunZ)
	setUni1f(gs, "AtmoRayleigh", a.rayleigh)
	setUni1f(gs, "AtmoMie", a.mie)
	setUni1f(gs, "AtmoTurbidity", a.turbidity)
	setUni1f(gs, "AtmoExposure", a.exposure)
	setUni3f(gs, "AtmoGround", a.ground.R, a.ground.G, a.ground.B)
	setUni1f(gs, "AtmoFlash", float32(m.w.wx.flash))
	tint := m.w.atmoWeatherTint()
	setUni3f(gs, "AtmoTint", tint.R, tint.G, tint.B)
	if a.tex != nil {
		a.tex.RenderSetup(gs, 5, 5)
		setUni1i(gs, "AtmoT", 5)
		setUni1i(gs, "AtmoHasT", 1)
	} else {
		setUni1i(gs, "AtmoHasT", 0)
	}
}

func (w *World) atmoWeatherTint() math32.Color {
	c := math32.Color{1, 1, 1}
	switch w.wx.mode {
	case "rain":
		c = math32.Color{0.78, 0.82, 0.88}
	case "snow":
		c = math32.Color{0.88, 0.90, 0.96}
	case "fog":
		c = math32.Color{0.82, 0.84, 0.86}
	case "storm":
		c = math32.Color{0.42, 0.46, 0.55}
	}
	k := float32(w.wx.intensity)
	if k < 0 {
		k = 0
	}
	if k > 1 {
		k = 1
	}
	return math32.Color{
		1 - (1-c.R)*k,
		1 - (1-c.G)*k,
		1 - (1-c.B)*k,
	}
}

func generateTransmittanceLUT() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, atmoLUTW, atmoLUTH))
	betaR := [3]float64{5.8e-3, 13.5e-3, 33.1e-3} // per km, scaled
	betaM := 21e-3
	for y := 0; y < atmoLUTH; y++ {
		alt := float64(y) / float64(atmoLUTH-1) // 0 ground … 1 top
		r := earthRkm + alt*atmoHkm
		for x := 0; x < atmoLUTW; x++ {
			mu := float64(x)/float64(atmoLUTW-1)*2.15 - 0.15 // -0.15..2.0 → clamp in march
			if mu > 1 {
				mu = 1
			}
			odR, odM := opticalDepthRay(r, mu)
			tr := math.Exp(-odR * betaR[0])
			tg := math.Exp(-odR * betaR[1])
			tb := math.Exp(-odR * betaR[2])
			tm := math.Exp(-odM * betaM)
			img.SetRGBA(x, y, color.RGBA{u8f(tr), u8f(tg), u8f(tb), u8f(tm)})
		}
	}
	return img
}

func opticalDepthRay(r0, mu float64) (odR, odM float64) {
	const steps = 32
	rt := earthRkm + atmoHkm
	// quadratic for atmosphere sphere hit
	b := 2 * r0 * mu
	c := r0*r0 - rt*rt
	disc := b*b - 4*c
	if disc < 0 {
		return 1e3, 1e3
	}
	tFar := (-b + math.Sqrt(disc)) * 0.5
	if tFar < 0 {
		return 1e3, 1e3
	}
	ds := tFar / float64(steps)
	for i := 0; i < steps; i++ {
		t := (float64(i) + 0.5) * ds
		r := math.Sqrt(r0*r0 + t*t + 2*r0*mu*t)
		h := r - earthRkm
		if h < 0 {
			return 1e3, 1e3
		}
		odR += math.Exp(-h/hrKm) * ds
		odM += math.Exp(-h/hmKm) * ds
	}
	return odR, odM
}

// skyRadianceCPU is a cheap single-scatter sample used to paint cubemap faces
// when a live dome is not created (CreateSkyBox default).
func skyRadianceCPU(dx, dy, dz, sx, sy, sz, rayleigh, mie float64) (r, g, b float64) {
	inv := 1 / math.Sqrt(dx*dx+dy*dy+dz*dz)
	dx, dy, dz = dx*inv, dy*inv, dz*inv
	sinv := 1 / math.Sqrt(sx*sx+sy*sy+sz*sz)
	sx, sy, sz = sx*sinv, sy*sinv, sz*sinv
	mu := dx*sx + dy*sy + dz*sz
	elev := clamp01f(dy*0.5 + 0.5)
	// Rayleigh-ish zenith vs horizon
	zenith := math32.Color{0.12, 0.28, 0.72}
	horiz := math32.Color{0.62, 0.74, 0.88}
	fade := clamp01f(1 - math.Exp(6.5-13*elev))
	r = float64(horiz.R) + (float64(zenith.R)-float64(horiz.R))*fade
	g = float64(horiz.G) + (float64(zenith.G)-float64(horiz.G))*fade
	b = float64(horiz.B) + (float64(zenith.B)-float64(horiz.B))*fade
	// Mie forward glow
	phaseR := 0.75 * (1 + mu*mu)
	gM := 0.76
	phaseM := (1 - gM*gM) / math.Pow(math.Max(1e-4, 1+gM*gM-2*gM*mu), 1.5)
	r += (0.035*phaseR*rayleigh + 0.09*phaseM*mie) * (0.55 + 0.45*elev)
	g += (0.055*phaseR*rayleigh + 0.07*phaseM*mie) * (0.55 + 0.45*elev)
	b += (0.12*phaseR*rayleigh + 0.04*phaseM*mie) * (0.55 + 0.45*elev)
	if dy < 0 {
		ground := -dy
		r = r*(1-ground*0.7) + 0.18*ground
		g = g*(1-ground*0.65) + 0.22*ground
		b = b*(1-ground*0.55) + 0.14*ground
	}
	disk := math.Pow(math.Max(0, mu), 80)
	glow := math.Pow(math.Max(0, mu), 8) * 0.45
	r += disk*1.1 + glow
	g += disk*0.92 + glow*0.75
	b += disk*0.55 + glow*0.35
	return r, g, b
}

func generateAtmosphereCubemap(sx, sy, sz, rayleigh, mie, exposure float64) [6]*texture.Texture2D {
	if exposure <= 0 {
		exposure = 1
	}
	const n = 128
	var out [6]*texture.Texture2D
	for face := 0; face < 6; face++ {
		img := image.NewRGBA(image.Rect(0, 0, n, n))
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				dx, dy, dz := skyFaceDir(face, x, y, n)
				r, g, b := skyRadianceCPU(dx, dy, dz, sx, sy, sz, rayleigh, mie)
				r = 1 - math.Exp(-r*exposure)
				g = 1 - math.Exp(-g*exposure)
				b = 1 - math.Exp(-b*exposure)
				img.SetRGBA(x, y, color.RGBA{u8f(r), u8f(g), u8f(b), 255})
			}
		}
		out[face] = texture.NewTexture2DFromRGBA(img)
	}
	return out
}

func (w *World) ensureAtmosphere() int {
	if w.atmo != nil && w.atmo.mesh != nil {
		return w.atmo.ent
	}
	if w.scene == nil {
		return 0
	}
	lut := texture.NewTexture2DFromRGBA(generateTransmittanceLUT())
	geom := geometry.NewSphere(90, 32, 24)
	std := material.NewStandard(&math32.Color{0.4, 0.6, 0.9})
	std.SetUseLights(material.UseLightNone)
	std.SetSide(material.SideBack)
	std.SetDepthMask(true)
	std.SetDepthTest(true)
	std.SetDepthFunc(gls.LEQUAL)
	std.SetShader("mbatmo")
	a := &atmoDome{
		tex: lut, sunX: -0.35, sunY: 0.62, sunZ: 0.70,
		rayleigh: 1, mie: 1, turbidity: 1.2, exposure: 1.15,
		ground: math32.Color{0.18, 0.22, 0.14},
	}
	mat := &atmoMat{Standard: std, w: w, a: a}
	mesh := graphic.NewMesh(geom, mat)
	mesh.SetCullable(false)
	// G3N opaque pass draws high RenderOrder first. Draw last so terrain
	// depth is in the buffer; LEQUAL + depth mask lets early-Z skip sky
	// fragments behind mountains.
	mesh.SetRenderOrder(-20)
	w.scene.Add(mesh)
	id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: std, sky: true}, 0)
	if e := w.ents[id]; e != nil {
		e.name = "atmosphere"
		e.sky = true
	}
	a.ent = id
	a.mesh = mesh
	a.mat = mat
	w.atmo = a
	return id
}

func (w *World) tickAtmosphere() {
	if w.atmo == nil || w.atmo.mesh == nil || w.cam == nil {
		return
	}
	var p math32.Vector3
	w.cam.WorldPosition(&p)
	w.atmo.mesh.SetPosition(p.X, p.Y, p.Z)
}

func (w *World) clearAtmosphere() {
	if w.atmo == nil {
		return
	}
	if w.atmo.mesh != nil {
		if p := w.atmo.mesh.GetNode().Parent(); p != nil {
			p.GetNode().Remove(w.atmo.mesh)
		}
		w.atmo.mesh.SetVisible(false)
	}
	if w.atmo.ent != 0 {
		delete(w.ents, w.atmo.ent)
	}
	w.atmo = nil
}

func (w *World) setAtmosphereParams(a []value.Value) {
	w.ensureAtmosphere()
	if w.atmo == nil {
		return
	}
	if len(a) == 1 {
		on := argN(a, 0, 1)
		if w.atmo.mesh != nil {
			w.atmo.mesh.SetVisible(on > 0.5)
		}
		return
	}
	if len(a) >= 3 {
		w.atmo.sunX = float32(argN(a, 0, -0.35))
		w.atmo.sunY = float32(argN(a, 1, 0.62))
		w.atmo.sunZ = float32(argN(a, 2, 0.70))
		w.aimDirLightVec(w.atmo.sunX, w.atmo.sunY, w.atmo.sunZ)
		w.skySunOK = false
		w.syncVisualSun()
	}
	if len(a) >= 4 {
		w.atmo.rayleigh = float32(argN(a, 3, 1))
	}
	if len(a) >= 5 {
		w.atmo.mie = float32(argN(a, 4, 1))
	}
	if len(a) >= 6 {
		w.atmo.exposure = float32(argN(a, 5, 1.15))
	}
}

func (w *World) atmoVisible() bool {
	return w.atmo != nil && w.atmo.mesh != nil && w.atmo.mesh.Visible()
}
