package runtime

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/go-gl/gl/v3.3-core/gl"

	"bitshinbasic/internal/value"
)

type gerstnerWave struct {
	dirX, dirZ float32
	steep      float32
	amp        float32
	lambda     float32
	speed      float32
}

type waterFBO struct {
	fbo, color, depth uint32
	w, h              int32
}

type waterBody struct {
	ent         int
	y           float32
	w, d        float32
	color       math32.Color
	reflectOn   bool
	refractOn   bool
	follow      bool
	waves       [4]gerstnerWave
	nWaves      int
	time        float32
	speed       float32
	move        float32
	waveStr     float32
	shine       float32
	reflectivity float32
	normalTex   int
	dudvTex     int
	buoys       map[int]bool
	wasIn       map[int]bool
	splash      int
	reflect     waterFBO
	refract     waterFBO
	dudvGL      uint32
	normGL      uint32
	windX       float32
	windZ       float32
	windStr     float32
	style       string
	underFog    math32.Color
	underDen    float32
	savedFog    bool
	fogMode     int
	fogRGB      math32.Color
	fogDen      float32
	mat         *WaterMaterial
	caustics    bool
	ssr         bool
	causticGL   uint32
	wakeGL      uint32
	wakeH       []float32
	wakeV       []float32
	wakePix     []uint8
	wakeSpan    float32
	ambSnd      int
	ambVol      float64
	segs        int
	lod         bool
	farR        float32
}

func bindWaterMaterial(mat *material.Standard) {
	if mat == nil {
		return
	}
	(&WaterMaterial{Standard: mat, WaveSpeed: 1}).Configure()
}

func (w *World) bindWaterShadows(gs *gls.GLS) {
	if gs == nil {
		return
	}
	dummy := w.ensureDummyShadowTex()
	if !w.shadow.on || !w.shadow.ready || w.shadow.tex == 0 {
		bindWaterSampler(gs, 11, dummy, "ShadowMap")
		setUni1i(gs, "ShadowEnabled", 0)
		setUni1f(gs, "ShadowBias", 0.0025)
		setUni1i(gs, "AtlasCols", 1)
		setUni1i(gs, "AtlasRows", 1)
		return
	}
	depth := w.shadow.depthTex
	if depth == 0 {
		depth = w.shadow.tex
	}
	bindWaterSampler(gs, 11, depth, "ShadowMap")
	setUni1i(gs, "ShadowEnabled", 1)
	bias := w.shadow.bias
	if bias <= 0 {
		bias = 0.0025
	}
	setUni1f(gs, "ShadowBias", bias)
	cols, rows := w.shadow.atlasCols, w.shadow.atlasRows
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	setUni1i(gs, "AtlasCols", cols)
	setUni1i(gs, "AtlasRows", rows)
	for i := 0; i < maxShadowCascades; i++ {
		setUniMat4(gs, "LightVP["+waveIdx(i)+"]", &w.shadow.lightVP[i])
	}
}

func bindWaterSampler(gs *gls.GLS, unit uint32, tex uint32, name string) {
	if gs == nil || tex == 0 {
		return
	}
	gs.ActiveTexture(gls.TEXTURE0 + unit)
	gs.BindTexture(gls.TEXTURE_2D, tex)
	setUni1i(gs, name, int(unit))
	gs.ActiveTexture(gls.TEXTURE0)
}

func setUni4f(gs *gls.GLS, name string, x, y, z, w float32) {
	u := gls.Uniform{}
	u.Init(name)
	if loc := u.Location(gs); loc >= 0 {
		gs.Uniform4f(loc, x, y, z, w)
	}
}

func waveIdx(i int) string {
	return string(rune('0' + i))
}

func defaultWaves() [4]gerstnerWave {
	return [4]gerstnerWave{
		{dirX: 0.85, dirZ: 0.35, steep: 0.34, amp: 1.15, lambda: 52, speed: 0.82},
		{dirX: -0.28, dirZ: 0.96, steep: 0.28, amp: 0.55, lambda: 24, speed: 1.18},
		{dirX: 0.58, dirZ: -0.72, steep: 0.22, amp: 0.22, lambda: 10, speed: 1.75},
		{dirX: 0.18, dirZ: 0.98, steep: 0.16, amp: 0.08, lambda: 3.6, speed: 2.45},
	}
}

func applyWaterWind(dirX, dirZ, amp, windX, windZ, windStr float32) (float32, float32, float32) {
	dl := float32(math.Sqrt(float64(dirX*dirX + dirZ*dirZ)))
	if dl < 1e-5 {
		return dirX, dirZ, amp
	}
	dx, dz := dirX/dl, dirZ/dl
	if windStr <= 0 {
		return dx, dz, amp
	}
	wl := float32(math.Sqrt(float64(windX*windX + windZ*windZ)))
	if wl < 1e-5 {
		return dx, dz, amp
	}
	wx, wz := windX/wl, windZ/wl
	k := windStr * 0.55
	if k > 0.85 {
		k = 0.85
	}
	mx, mz := dx*(1-k)+wx*k, dz*(1-k)+wz*k
	ml := float32(math.Sqrt(float64(mx*mx + mz*mz)))
	if ml > 1e-5 {
		dx, dz = mx/ml, mz/ml
	}
	dot := dx*wx + dz*wz
	if dot < 0 {
		dot = 0
	}
	amp *= 1 + windStr*0.35*dot
	return dx, dz, amp
}

func gerstnerHeight(waves []gerstnerWave, n int, x, z, t, storm float32) float32 {
	return gerstnerHeightWind(waves, n, x, z, t, storm, 0, 0, 0)
}

func gerstnerHeightWind(waves []gerstnerWave, n int, x, z, t, storm, windX, windZ, windStr float32) float32 {
	y := float32(0)
	if n < 1 {
		return 0
	}
	if n > len(waves) {
		n = len(waves)
	}
	for i := 0; i < n; i++ {
		wv := waves[i]
		if wv.lambda < 0.1 {
			continue
		}
		dx, dz, amp := applyWaterWind(wv.dirX, wv.dirZ, wv.amp*storm, windX, windZ, windStr)
		k := float32(2*math.Pi) / wv.lambda
		f := k*(dx*x+dz*z) - wv.speed*t
		y += amp * float32(math.Sin(float64(f)))
	}
	return y
}

func newWaterGrid(width, depth float32, segs int) *geometry.Geometry {
	if segs < 4 {
		segs = 24
	}
	hw, hd := width/2, depth/2
	pos := math32.NewArrayF32(0, (segs+1)*(segs+1)*3)
	nor := math32.NewArrayF32(0, (segs+1)*(segs+1)*3)
	uvs := math32.NewArrayF32(0, (segs+1)*(segs+1)*2)
	idx := math32.NewArrayU32(0, segs*segs*6)
	for iz := 0; iz <= segs; iz++ {
		for ix := 0; ix <= segs; ix++ {
			u := float32(ix) / float32(segs)
			v := float32(iz) / float32(segs)
			x := -hw + u*width
			z := -hd + v*depth
			gx, gy, gz := toG3N(x, 0, z)
			pos.Append(gx, gy, gz)
			nor.Append(0, 1, 0)
			uvs.Append(x*0.04, z*0.04)
		}
	}
	stride := segs + 1
	for iz := 0; iz < segs; iz++ {
		for ix := 0; ix < segs; ix++ {
			a := uint32(iz*stride + ix)
			b := a + 1
			c := a + uint32(stride)
			d := c + 1
			idx.Append(a, b, d, a, d, c)
		}
	}
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

// newWaterLODGrid is a camera-centered sinh-warped mesh: dense near the
// origin (follow puts that under the camera) and coarse toward a far radius
// so the horizon stays filled without a 128-segment cap.
func newWaterLODGrid(radius float32, segs int) *geometry.Geometry {
	if segs < 24 {
		segs = 24
	}
	if segs > 160 {
		segs = 160
	}
	if radius < 16 {
		radius = 16
	}
	k := 3.45
	sk := math.Sinh(k)
	map1 := func(i int) float32 {
		u := float64(i)/float64(segs)*2 - 1
		return float32(math.Sinh(u*k)/sk) * radius
	}
	n := segs + 1
	pos := math32.NewArrayF32(0, n*n*3)
	nor := math32.NewArrayF32(0, n*n*3)
	uvs := math32.NewArrayF32(0, n*n*2)
	idx := math32.NewArrayU32(0, segs*segs*6)
	xs := make([]float32, n)
	zs := make([]float32, n)
	for i := 0; i < n; i++ {
		xs[i] = map1(i)
		zs[i] = map1(i)
	}
	for iz := 0; iz < n; iz++ {
		for ix := 0; ix < n; ix++ {
			x, z := xs[ix], zs[iz]
			gx, gy, gz := toG3N(x, 0, z)
			pos.Append(gx, gy, gz)
			nor.Append(0, 1, 0)
			uvs.Append(x*0.04, z*0.04)
		}
	}
	stride := n
	for iz := 0; iz < segs; iz++ {
		for ix := 0; ix < segs; ix++ {
			a := uint32(iz*stride + ix)
			b := a + 1
			c := a + uint32(stride)
			d := c + 1
			idx.Append(a, b, d, a, d, c)
		}
	}
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

func (w *World) waterFarRadius(wb *waterBody) float32 {
	r := wb.w
	if wb.d > r {
		r = wb.d
	}
	r *= 0.5
	if !wb.follow {
		return r
	}
	far := float32(1800)
	if w != nil && w.cam != nil {
		if cf := w.cam.Far(); cf > 80 {
			far = cf * 0.58
		}
	}
	if wb.farR > far {
		far = wb.farR
	}
	if far < r {
		far = r
	}
	if far < 400 {
		far = 400
	}
	return far
}

func (w *World) buildWaterGeom(wb *waterBody) *geometry.Geometry {
	if wb == nil {
		return newWaterGrid(80, 80, 48)
	}
	if wb.follow {
		return newWaterLODGrid(w.waterFarRadius(wb), wb.segs)
	}
	return newWaterGrid(wb.w, wb.d, wb.segs)
}

func (w *World) replaceWaterGeom(wb *waterBody, g *geometry.Geometry) {
	if wb == nil || g == nil {
		return
	}
	e := w.ents[wb.ent]
	if e == nil || e.mesh == nil {
		return
	}
	old := e.mesh
	n := old.GetNode()
	pos := n.Position()
	mesh := graphic.NewMesh(g, wb.mat)
	mesh.SetCullable(false)
	mesh.SetRenderOrder(100)
	mesh.SetPosition(pos.X, pos.Y, pos.Z)
	parent := n.Parent()
	if parent != nil {
		parent.GetNode().Remove(old)
		parent.GetNode().Add(mesh)
	} else if w.scene != nil {
		w.scene.Remove(old)
		w.scene.Add(mesh)
	}
	e.node = mesh
	e.mesh = mesh
	e.mat = wb.mat.Standard
	e.name = "water"
	e.castShadow = false
}

func (w *World) rebuildWaterMesh(wb *waterBody) {
	if wb == nil {
		return
	}
	w.replaceWaterGeom(wb, w.buildWaterGeom(wb))
}

func (w *World) applyWaterStyle(wb *waterBody, style string) {
	if wb == nil {
		return
	}
	s := strings.ToLower(strings.TrimSpace(style))
	switch s {
	case "scenic", "pretty", "learnopengl", "dudv", "article":
		wb.style = "scenic"
		if wb.nWaves < 1 {
			wb.nWaves = 2
		}
		wb.reflectOn = true
		wb.refractOn = true
		wb.waveStr = 0.05
	case "gerstner", "waves":
		wb.style = "gerstner"
		wb.nWaves = 4
		wb.waves = defaultWaves()
		wb.reflectOn = true
		wb.refractOn = true
		wb.waveStr = 0.035
		wb.shine = 48
		wb.reflectivity = 0.85
	default:
		wb.style = "ocean"
		wb.nWaves = 4
		wb.waves = defaultWaves()
		wb.reflectOn = true
		wb.refractOn = true
		wb.waveStr = 0.04
		wb.shine = 56
		wb.reflectivity = 0.92
		if wb.color.R+wb.color.G+wb.color.B < 0.05 {
			wb.color = math32.Color{0.03, 0.16, 0.24}
		}
	}
}

func (w *World) currentWater() *waterBody {
	if wb := w.waters[w.curWater]; wb != nil {
		return wb
	}
	for _, wb := range w.waters {
		if wb != nil {
			return wb
		}
	}
	return nil
}

func (w *World) waterHeight(x, z float32) float32 {
	wb := w.currentWater()
	if wb == nil {
		return 0
	}
	storm := float32(1)
	if w.wx.mode == "storm" {
		storm = 1.7
	}
	t := wb.time
	if wb.mat != nil {
		t = wb.mat.clock()
	}
	return wb.y + gerstnerHeightWind(wb.waves[:], wb.nWaves, x, z, t, storm, wb.windX, wb.windZ, wb.windStr)
}

func (w *World) tickWater() {
	dt := float32(w.delta)
	if dt <= 0 {
		dt = 1.0 / 60
	}
	for _, wb := range w.waters {
		if wb == nil {
			continue
		}
		if wb.mat != nil {
			wb.mat.Update(dt)
		} else {
			wb.time += dt
			spd := wb.speed
			if spd <= 0 {
				spd = 0.03
			}
			wb.move = float32(math.Mod(float64(wb.move+spd*dt), 1))
		}
		if e := w.ents[wb.ent]; e != nil && e.node != nil {
			if wb.follow && w.cam != nil {
				p := worldPos(w.cam.GetNode())
				cx, _, cz := fromG3N(p.X, p.Y, p.Z)
				gx, gy, gz := toG3N(cx, wb.y, cz)
				e.node.GetNode().SetPosition(gx, gy, gz)
			}
		}
		w.tickBuoys(wb)
		w.tickWake(wb, dt)
		w.tickWaterAmbient(wb)
		w.tickUnderwater(wb)
	}
}

func (w *World) tickBuoys(wb *waterBody) {
	for id := range wb.buoys {
		e := w.ents[id]
		if e == nil || e.node == nil {
			continue
		}
		p := worldPos(e.node.GetNode())
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		wh := w.waterHeight(x, z)
		sit := wh + e.boxY*0.45
		in := y < wh+0.35
		if in && !wb.wasIn[id] {
			w.waterSplash(wb, x, wh, z)
		}
		if in {
			dt := float32(w.delta)
			if dt <= 0 {
				dt = 1.0 / 60
			}
			w.wakeImpulse(wb, x, z, 0.18*dt)
		}
		wb.wasIn[id] = in
		gx, gy, gz := toG3N(x, sit, z)
		e.node.GetNode().SetPosition(gx, gy, gz)
		if w.phys3 != nil {
			w.phys3.SetPosition(id, x, sit, z)
		}
	}
}

func (w *World) waterSplash(wb *waterBody, x, y, z float32) {
	if wb.splash == 0 {
		wb.splash = w.newEmitter(0, false)
		if em := w.emitters[wb.splash]; em != nil {
			em.rate = 0
			em.max = 40
			em.life = 0.55
			em.size0, em.size1 = 0.12, 0.35
			em.cr0, em.cg0, em.cb0 = 0.75, 0.88, 1
			em.vy = 3
		}
	}
	if em := w.emitters[wb.splash]; em != nil {
		em.x, em.y, em.z = float64(x), float64(y), float64(z)
		for i := 0; i < 12; i++ {
			w.spawnParticle(em)
		}
	}
	w.wakeImpulse(wb, x, z, 0.85)
}

func (w *World) tickUnderwater(wb *waterBody) {
	if w.cam == nil {
		return
	}
	p := worldPos(w.cam.GetNode())
	x, y, z := fromG3N(p.X, p.Y, p.Z)
	wh := w.waterHeight(x, z)
	under := y < wh-0.05
	if under && !wb.savedFog {
		wb.fogMode, wb.fogRGB, wb.fogDen = w.fogMode, w.fogRGB, w.fogDensity
		wb.savedFog = true
	}
	if under {
		w.fogMode = 2
		if wb.underDen > 0 {
			w.fogDensity = wb.underDen
		} else {
			w.fogDensity = 0.08
		}
		if wb.underFog.R+wb.underFog.G+wb.underFog.B > 0 {
			w.fogRGB = wb.underFog
		} else {
			w.fogRGB = math32.Color{wb.color.R * 0.25, wb.color.G * 0.45, wb.color.B * 0.55}
		}
	} else if wb.savedFog {
		w.fogMode, w.fogRGB, w.fogDensity = wb.fogMode, wb.fogRGB, wb.fogDen
		wb.savedFog = false
	}
}

func disposeWaterFBO(t *waterFBO) {
	if t.color != 0 {
		gl.DeleteTextures(1, &t.color)
		t.color = 0
	}
	if t.depth != 0 {
		gl.DeleteTextures(1, &t.depth)
		t.depth = 0
	}
	if t.fbo != 0 {
		gl.DeleteFramebuffers(1, &t.fbo)
		t.fbo = 0
	}
	t.w, t.h = 0, 0
}

func ensureWaterFBO(t *waterFBO, ww, hh int) {
	if ww < 64 {
		ww = 256
	}
	if hh < 64 {
		hh = 256
	}
	if t.fbo != 0 && t.w == int32(ww) && t.h == int32(hh) {
		return
	}
	disposeWaterFBO(t)
	var fbo, tex, depth uint32
	gl.GenFramebuffers(1, &fbo)
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(ww), int32(hh), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.GenTextures(1, &depth)
	gl.BindTexture(gl.TEXTURE_2D, depth)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT24, int32(ww), int32(hh), 0, gl.DEPTH_COMPONENT, gl.UNSIGNED_INT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, tex, 0)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, depth, 0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	t.fbo, t.color, t.depth = fbo, tex, depth
	t.w, t.h = int32(ww), int32(hh)
}

func uploadWaterRGBA(img *image.RGBA) uint32 {
	if img == nil {
		return 0
	}
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(img.Bounds().Dx()), int32(img.Bounds().Dy()), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return tex
}

func waterHash2(ix, iy, wrap int) float64 {
	if wrap < 1 {
		wrap = 1
	}
	ix = ((ix % wrap) + wrap) % wrap
	iy = ((iy % wrap) + wrap) % wrap
	n := ix*374761393 + iy*668265263
	n = (n ^ (n >> 13)) * 1274126177
	return float64(n&0xffff) / 65535.0
}

func waterValueNoise(x, y float64, wrap int) float64 {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	tx := x - math.Floor(x)
	ty := y - math.Floor(y)
	tx = tx * tx * (3 - 2*tx)
	ty = ty * ty * (3 - 2*ty)
	a := waterHash2(x0, y0, wrap)
	b := waterHash2(x0+1, y0, wrap)
	c := waterHash2(x0, y0+1, wrap)
	d := waterHash2(x0+1, y0+1, wrap)
	return (a*(1-tx)+b*tx)*(1-ty) + (c*(1-tx)+d*tx)*ty
}

func genWaterDuDv(n int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	cells := 4.0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			u := float64(x) / float64(n)
			v := float64(y) / float64(n)
			ax := u * 2 * math.Pi * cells
			ay := v * 2 * math.Pi * cells
			dx := math.Sin(ax) * math.Cos(ay)
			dy := math.Cos(ax) * math.Sin(ay)
			dx += 0.35 * math.Sin(ax*2+1.7) * math.Cos(ay*2+0.4)
			dy += 0.35 * math.Cos(ax*2+0.9) * math.Sin(ay*2+2.1)
			rx := uint8((dx*0.5 + 0.5) * 255)
			gy := uint8((dy*0.5 + 0.5) * 255)
			img.SetRGBA(x, y, color.RGBA{rx, gy, 0, 255})
		}
	}
	return img
}

func genWaterNormal(n int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	cells := 10
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			fx := float64(x) / float64(n) * float64(cells)
			fy := float64(y) / float64(n) * float64(cells)
			hL := waterValueNoise(fx-0.08, fy, cells)
			hR := waterValueNoise(fx+0.08, fy, cells)
			hD := waterValueNoise(fx, fy-0.08, cells)
			hU := waterValueNoise(fx, fy+0.08, cells)
			nx := (hL - hR) * 2.4
			nz := (hD - hU) * 2.4
			ny := 1.0
			len := math.Sqrt(nx*nx + ny*ny + nz*nz)
			nx, ny, nz = nx/len, ny/len, nz/len
			img.SetRGBA(x, y, color.RGBA{
				uint8((nx*0.5 + 0.5) * 255),
				uint8((nz*0.5 + 0.5) * 255),
				uint8((ny*0.5 + 0.5) * 255),
				255,
			})
		}
	}
	return img
}

func (w *World) replaceWaterGLMap(dst *uint32, texID int) {
	t := w.texs[texID]
	if t == nil || t.tex == nil || t.path == "" {
		return
	}
	path, err := w.openPath(t.path)
	if err != nil {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return
	}
	rgba, ok := img.(*image.RGBA)
	if !ok {
		b := img.Bounds()
		rgba = image.NewRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}
	if *dst != 0 {
		gl.DeleteTextures(1, dst)
	}
	*dst = uploadWaterRGBA(rgba)
}

func (w *World) waterSunDir() (float32, float32, float32) {
	for _, e := range w.ents {
		if e == nil || e.lgtKind != 1 || e.node == nil {
			continue
		}
		var p math32.Vector3
		e.node.GetNode().WorldPosition(&p)
		if p.LengthSq() < 1e-6 {
			e.node.GetNode().WorldDirection(&p)
			p.Negate()
		}
		if p.LengthSq() < 1e-6 {
			continue
		}
		p.Normalize()
		return p.X, p.Y, p.Z
	}
	return 0.4, 0.8, 0.3
}

func (w *World) isTerrainEntity(id int) bool {
	for _, t := range w.terrains {
		if t == nil {
			continue
		}
		for _, eid := range t.ents {
			if eid == id {
				return true
			}
		}
	}
	return false
}

func (w *World) hideForWaterPass(wb *waterBody, keepAbove bool) func() {
	type rec struct {
		n   core.INode
		vis bool
	}
	var saved []rec
	hide := func(n core.INode) {
		if n == nil {
			return
		}
		saved = append(saved, rec{n: n, vis: n.GetNode().Visible()})
		n.GetNode().SetVisible(false)
	}
	if e := w.ents[wb.ent]; e != nil && e.node != nil {
		hide(e.node)
	}
	for id, e := range w.ents {
		if e == nil || e.node == nil || id == wb.ent {
			continue
		}
		if e.lgtKind != 0 || e.cam != nil || e.sky {
			continue
		}
		if w.isTerrainEntity(id) {
			continue
		}
		p := worldPos(e.node.GetNode())
		_, y, _ := fromG3N(p.X, p.Y, p.Z)
		r := e.radius
		if r < 0.4 {
			r = 0.4
		}
		if e.boxY > r {
			r = e.boxY
		}
		if keepAbove && y+r < wb.y-0.08 {
			hide(e.node)
		}
		if !keepAbove && y-r > wb.y+0.08 {
			hide(e.node)
		}
	}
	return func() {
		for i := range saved {
			saved[i].n.GetNode().SetVisible(saved[i].vis)
		}
	}
}

func (w *World) renderWaterFBO(rend *renderer.Renderer, cam *camera.Camera, t *waterFBO, clear math32.Color) {
	if t.fbo == 0 || cam == nil || t.w < 1 || t.h < 1 {
		return
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	gl.Viewport(0, 0, t.w, t.h)
	gl.ClearColor(clear.R, clear.G, clear.B, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	_ = rend.Render(w.scene, cam)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func (w *World) reflectionCamera(cam *camera.Camera, waterY float32) *camera.Camera {
	mirror := camera.New(cam.Aspect())
	mirror.SetFov(cam.Fov())
	mirror.SetNear(cam.Near())
	mirror.SetFar(cam.Far())
	cam.GetNode().UpdateMatrixWorld()
	cp := worldPos(cam.GetNode())
	var dir math32.Vector3
	cam.GetNode().WorldDirection(&dir)
	dir.Negate()
	tgt := math32.Vector3{cp.X + dir.X, cp.Y + dir.Y, cp.Z + dir.Z}
	mirror.SetPosition(cp.X, 2*waterY-cp.Y, cp.Z)
	rt := math32.Vector3{tgt.X, 2*waterY - tgt.Y, tgt.Z}
	up := math32.Vector3{0, 1, 0}
	mirror.LookAt(&rt, &up)
	return mirror
}

func (w *World) renderWaterReflection(rend *renderer.Renderer, cam *camera.Camera) {
	wb := w.currentWater()
	if wb == nil || cam == nil || rend == nil || w.app == nil {
		return
	}
	if !wb.reflectOn && !wb.refractOn {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("EnableWaterReflection:", r)
		}
	}()
	w.ensureWaterMaps(wb)
	if wb.reflectOn {
		ensureWaterFBO(&wb.reflect, 512, 512)
	}
	if wb.refractOn {
		ensureWaterFBO(&wb.refract, 512, 512)
	}
	sky := math32.Color{wb.color.R * 0.45, wb.color.G * 0.55, wb.color.B * 0.7}
	if wb.reflectOn && wb.reflect.fbo != 0 {
		restore := w.hideForWaterPass(wb, true)
		mirror := w.reflectionCamera(cam, wb.y)
		w.renderWaterFBO(rend, mirror, &wb.reflect, sky)
		restore()
	}
	if wb.refractOn && wb.refract.fbo != 0 {
		restore := w.hideForWaterPass(wb, false)
		w.renderWaterFBO(rend, cam, &wb.refract, math32.Color{wb.color.R * 0.2, wb.color.G * 0.35, wb.color.B * 0.4})
		restore()
	}
}

func (w *World) waterCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createwater": need(func(a []value.Value) (value.Value, error) {
			ww := float32(argN(a, 0, 80))
			dd := float32(argN(a, 1, 80))
			if ww < 4 {
				ww = 4
			}
			if dd < 4 {
				dd = 4
			}
			segs := argI(a, 2, 64)
			if segs < 24 {
				segs = 24
			}
			if segs > 160 {
				segs = 160
			}
			wb := &waterBody{
				y: 0, w: ww, d: dd,
				color:        math32.Color{0.02, 0.12, 0.28},
				waves:        defaultWaves(), nWaves: 4,
				speed:        0.035,
				waveStr:      0.04,
				shine:        52,
				reflectivity: 0.88,
				windX:        0.8,
				windZ:        0.3,
				windStr:      0,
				style:        "ocean",
				buoys:        map[int]bool{}, wasIn: map[int]bool{},
				follow:       true,
				segs:         segs,
				lod:          true,
				wakeSpan:     140,
				ambVol:       -1,
			}
			g := w.buildWaterGeom(wb)
			wm := NewWaterMaterial(w, wb)
			wb.mat = wm
			mesh := graphic.NewMesh(g, wm)
			mesh.SetCullable(false)
			mesh.SetRenderOrder(100)
			id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: wm.Standard}, 0)
			if e := w.ents[id]; e != nil {
				e.name = "water"
				e.castShadow = false
			}
			wb.ent = id
			if w.waters == nil {
				w.waters = map[int]*waterBody{}
			}
			wid := w.takeHandle(&w.freeWaters, &w.nextWater)
			w.waters[wid] = wb
			w.curWater = wid
			return value.Num(float64(wid)), nil
		}),
		"setwatercolor": n(func(a []value.Value) (value.Value, error) {
			wb := w.waters[argI(a, 0, w.curWater)]
			if wb == nil {
				wb = w.currentWater()
			}
			if wb == nil {
				return z()
			}
			off := 0
			if w.waters[argI(a, 0, 0)] != nil && len(a) >= 4 {
				off = 1
			}
			wb.color = *rgb(argN(a, off, 20), argN(a, off+1, 70), argN(a, off+2, 95))
			if e := w.ents[wb.ent]; e != nil && e.mat != nil {
				e.mat.SetColor(&wb.color)
			}
			return z()
		}),
		"setwaternormalmap": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb != nil {
				wb.normalTex = argI(a, 0, 0)
				w.ensureWaterMaps(wb)
				w.replaceWaterGLMap(&wb.normGL, wb.normalTex)
			}
			return z()
		}),
		"setwaterdudv": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb != nil {
				wb.dudvTex = argI(a, 0, 0)
				w.ensureWaterMaps(wb)
				w.replaceWaterGLMap(&wb.dudvGL, wb.dudvTex)
			}
			return z()
		}),
		"setwaterspeed": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil {
				wb.speed = float32(argN(a, 0, 0.035))
			}
			return z()
		}),
		"getwaterspeed": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil {
				return value.Num(float64(wb.speed)), nil
			}
			return value.Num(0), nil
		}),
		"setwaterwavespeed": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil && wb.mat != nil {
				spd := float32(argN(a, 0, 1))
				if spd <= 0 {
					spd = 1
				}
				wb.mat.WaveSpeed = spd
			}
			return z()
		}),
		"getwaterwavespeed": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil && wb.mat != nil {
				return value.Num(float64(wb.mat.WaveSpeed)), nil
			}
			return value.Num(1), nil
		}),
		"setwaterwavestrength": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil {
				wb.waveStr = float32(argN(a, 0, 0.045))
			}
			return z()
		}),
		"enablewaterreflection": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb != nil {
				on := argI(a, 0, 1) != 0
				wb.reflectOn = on
				if on {
					wb.refractOn = true
				}
			}
			return z()
		}),
		"enablewaterrefraction": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil {
				wb.refractOn = argI(a, 0, 1) != 0
			}
			return z()
		}),
		"setwaterwaves": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb == nil {
				return z()
			}
			wb.nWaves = argI(a, 0, 4)
			if wb.nWaves < 0 {
				wb.nWaves = 0
			}
			if wb.nWaves > 4 {
				wb.nWaves = 4
			}
			if len(a) > 1 {
				amp := float32(argN(a, 1, 0.25))
				for i := range wb.waves {
					wb.waves[i].amp = amp * (1 - float32(i)*0.2)
				}
			}
			return z()
		}),
		"setwaterwind": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb == nil {
				return z()
			}
			wb.windX = float32(argN(a, 0, 1))
			wb.windZ = float32(argN(a, 1, 0))
			if len(a) > 2 {
				wb.windStr = float32(argN(a, 2, 0.6))
			} else {
				wb.windStr = 0.6
			}
			return z()
		}),
		"getwaterwind": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil {
				return value.Num(float64(wb.windStr)), nil
			}
			return value.Num(0), nil
		}),
		"setwaterstyle": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb == nil {
				return z()
			}
			w.applyWaterStyle(wb, argS(a, 0))
			return z()
		}),
		"setgerstner": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb == nil {
				return z()
			}
			i := argI(a, 0, 0)
			if i < 0 || i > 3 {
				return z()
			}
			wb.waves[i] = gerstnerWave{
				dirX: float32(argN(a, 1, 1)), dirZ: float32(argN(a, 2, 0)),
				steep: float32(argN(a, 3, 0.3)), amp: float32(argN(a, 4, 0.2)),
				lambda: float32(argN(a, 5, 10)), speed: float32(argN(a, 6, 1.5)),
			}
			if i+1 > wb.nWaves {
				wb.nWaves = i + 1
			}
			return z()
		}),
		"waterheight": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.waterHeight(float32(argN(a, 0, 0)), float32(argN(a, 1, 0))))), nil
		}),
		"createbuoy": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb == nil {
				return z()
			}
			id := argI(a, 0, 0)
			wb.buoys[id] = true
			return value.Num(float64(id)), nil
		}),
		"setwaterfollow": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil {
				on := argI(a, 0, 1) != 0
				if wb.follow != on {
					wb.follow = on
					wb.lod = on
					w.rebuildWaterMesh(wb)
				}
			}
			return z()
		}),
		"setwaterlevel": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb == nil {
				return z()
			}
			wb.y = float32(argN(a, 0, 0))
			if e := w.ents[wb.ent]; e != nil {
				p := e.node.GetNode().Position()
				e.node.GetNode().SetPosition(p.X, wb.y, p.Z)
			}
			return z()
		}),
		"getwaterlevel": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil {
				return value.Num(float64(wb.y)), nil
			}
			return value.Num(0), nil
		}),
		"setunderwaterfog": n(func(a []value.Value) (value.Value, error) {
			wb := w.currentWater()
			if wb == nil {
				return z()
			}
			wb.underFog = *rgb(argN(a, 0, 10), argN(a, 1, 40), argN(a, 2, 55))
			wb.underDen = float32(argN(a, 3, 0.08))
			return z()
		}),
		"setwaterstreamradius": n(func(a []value.Value) (value.Value, error) {
			if wb := w.currentWater(); wb != nil {
				wb.follow = true
				wb.lod = true
				n := float32(argN(a, 0, 2))
				if n < 1 {
					n = 1
				}
				wb.farR = n * 900
				w.rebuildWaterMesh(wb)
			}
			return z()
		}),
		"setwatercaustics": n(func(a []value.Value) (value.Value, error) {
			wb := w.waterArg(a)
			if wb != nil {
				wb.caustics = argI(a, waterOnOff(a), 1) != 0
				if wb.caustics {
					w.ensureWaterMaps(wb)
				}
			}
			return z()
		}),
		"getwatercaustics": n(func(a []value.Value) (value.Value, error) {
			if wb := w.waterArg(a); wb != nil && wb.caustics {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"setwaterssr": n(func(a []value.Value) (value.Value, error) {
			wb := w.waterArg(a)
			if wb != nil {
				wb.ssr = argI(a, waterOnOff(a), 1) != 0
				if wb.ssr {
					wb.reflectOn = true
				}
			}
			return z()
		}),
		"getwaterssr": n(func(a []value.Value) (value.Value, error) {
			if wb := w.waterArg(a); wb != nil && wb.ssr {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"setwaterambientsound": n(func(a []value.Value) (value.Value, error) {
			return w.setWaterAmbientSound(a)
		}),
		"getweatherintensity": n(func(a []value.Value) (value.Value, error) {
			w.ensureWeather()
			return value.Num(w.wx.intensity), nil
		}),
	}
}

func waterOnOff(a []value.Value) int {
	if len(a) >= 2 {
		return 1
	}
	return 0
}

func (w *World) waterArg(a []value.Value) *waterBody {
	if len(a) >= 2 {
		if wb := w.waters[argI(a, 0, 0)]; wb != nil {
			return wb
		}
	}
	if len(a) == 1 {
		if wb := w.waters[argI(a, 0, 0)]; wb != nil {
			return wb
		}
	}
	return w.currentWater()
}
