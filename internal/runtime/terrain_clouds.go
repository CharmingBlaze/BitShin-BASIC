package runtime

import (
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// Clouds are one wide sheet above the camera. Its shader draws value-noise
// cumulus that drift with the wind, catch the sun's color, darken where
// they are thick, glow at the edge toward the sun, and fade into the haze
// at the horizon. No texture and no raymarch: one quad, a few noise taps.

type cloudLayer struct {
	ent     int
	cover   float32
	density float32
	speed   float32
	time    float32
	y       float32 // height above the camera
	span    float32 // cloud size scale, meters
	color   math32.Color
	driftX  float32
	driftZ  float32
}

type cloudMat struct {
	*material.Standard
	w *World
	c *cloudLayer
}

func (m *cloudMat) GetMaterial() *material.Material { return m.Standard.GetMaterial() }
func (m *cloudMat) Dispose()                        { m.Standard.Dispose() }

// cloudRadius is how far the sheet reaches: far enough that its fade into
// the haze sits at the horizon from the usual heights.
func (c *cloudLayer) cloudRadius() float32 {
	return math32.Max(2400, c.y*22)
}

func (m *cloudMat) RenderSetup(gs *gls.GLS) {
	m.Standard.RenderSetup(gs)
	if m.w == nil || m.c == nil || gs == nil {
		return
	}
	c, w := m.c, m.w
	cam := math32.Vector3{}
	var view, proj, vp math32.Matrix4
	if w.cam != nil {
		w.cam.WorldPosition(&cam)
		w.cam.ViewMatrix(&view)
		w.cam.ProjMatrix(&proj)
		vp.MultiplyMatrices(&proj, &view)
	}
	setUni3f(gs, "CloudCam", cam.X, cam.Y, cam.Z)
	setUniMat4(gs, "CloudVP", &vp)
	sx, sy, sz := w.cloudSunDir()
	setUni3f(gs, "CloudSun", sx, sy, sz)
	sun := w.cloudSunColor()
	setUni3f(gs, "CloudSunColor", sun.R, sun.G, sun.B)
	haze := w.cloudHaze()
	setUni3f(gs, "CloudHaze", haze.R, haze.G, haze.B)
	cover := math32.Clamp(c.cover, 0, 1)
	setUni1f(gs, "CloudCover", cover)
	setUni1f(gs, "CloudDensity", c.density)
	setUni3f(gs, "CloudColor", c.color.R, c.color.G, c.color.B)
	setUni1f(gs, "CloudScale", 1/math32.Max(c.span*0.55, 40))
	setUni3f(gs, "CloudDrift", c.driftX, 0, c.driftZ)
	setUni1f(gs, "CloudTime", c.time)
	setUni1f(gs, "CloudRadius", c.cloudRadius())
	setUni1f(gs, "CloudNight", w.skyNight())
	cheap := float32(0)
	if w.quality.set && w.quality.tier == qualityLow {
		cheap = 1
	}
	setUni1f(gs, "CloudCheap", cheap)
}

// cloudSunColor is the first sun's hue at a steady brightness: a sunset
// sun is dim but should still paint the clouds orange. Strength only moves
// it a little.
func (w *World) cloudSunColor() math32.Color {
	for _, e := range w.ents {
		if e == nil || e.lgtKind != 1 || e.lgt == nil {
			continue
		}
		col := math32.Color{R: 1, G: 0.96, B: 0.9}
		if cl, ok := e.lgt.(interface{ Color() math32.Color }); ok {
			col = cl.Color()
		}
		k := float32(1)
		if in, ok := e.lgt.(interface{ Intensity() float32 }); ok {
			k = math32.Clamp(in.Intensity()*1.6, 0.7, 1.25)
		}
		if m := math32.Max(col.R, math32.Max(col.G, col.B)); m > 0.01 {
			k /= m
		}
		return math32.Color{R: col.R * k, G: col.G * k, B: col.B * k}
	}
	return math32.Color{R: 1, G: 0.96, B: 0.9}
}

// cloudHaze is what far clouds melt into: the fog, else the sky's base.
func (w *World) cloudHaze() math32.Color {
	if w.fogMode != 0 {
		return w.fogRGB
	}
	if w.skyBot.R+w.skyBot.G+w.skyBot.B > 0 {
		return w.skyBot
	}
	return math32.Color{R: 0.75, G: 0.82, B: 0.9}
}

// skyNight is 0 by day and 1 at full night, from the sky's top color, the
// same rule the procedural skybox uses for its stars.
func (w *World) skyNight() float32 {
	top := (w.skyTop.R + w.skyTop.G + w.skyTop.B) / 3
	if top <= 0 {
		return 0
	}
	return math32.Clamp((0.16-top)/0.06, 0, 1)
}

func cloudSheetGeometry() *geometry.Geometry {
	pos := math32.NewArrayF32(0, 12)
	nor := math32.NewArrayF32(0, 12)
	uvs := math32.NewArrayF32(0, 8)
	idx := math32.NewArrayU32(0, 6)
	// A unit square in XZ; the node's scale sets its reach.
	pos.Append(-1, 0, -1, 1, 0, -1, 1, 0, 1, -1, 0, 1)
	nor.Append(0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0)
	uvs.Append(0, 0, 1, 0, 1, 1, 0, 1)
	idx.Append(0, 1, 2, 0, 2, 3)
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

func (w *World) createVolumetricClouds(y, scale float32) int {
	// A sheet low over the camera is seen edge-on, as streaks. The old
	// cards sat at a fraction of y; 120 m is the lowest that reads as sky.
	if y < 120 {
		y = 120
	}
	if scale < 80 {
		scale = 80
	}
	cl := &cloudLayer{
		cover: 0.52, density: 1, speed: 1, y: y, span: scale,
		color: math32.Color{R: 0.96, G: 0.97, B: 0.99},
	}
	mat := material.NewStandard(&math32.Color{R: 1, G: 1, B: 1})
	mat.SetUseLights(material.UseLightNone)
	mat.SetSide(material.SideDouble)
	mat.SetTransparent(true)
	mat.SetOpacity(1)
	mat.SetDepthMask(false)
	mat.SetDepthTest(true)
	mat.SetShader("mbclouds")
	mesh := graphic.NewMesh(cloudSheetGeometry(), &cloudMat{Standard: mat, w: w, c: cl})
	mesh.SetCullable(false)
	mesh.SetRenderOrder(85)
	id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat}, 0)
	if e := w.ents[id]; e != nil {
		e.name = "clouds"
		e.meshNoCast = true
	}
	cl.ent = id
	if w.clouds == nil {
		w.clouds = map[int]*cloudLayer{}
	}
	cid := w.takeHandle(&w.freeClouds, &w.nextCloud)
	w.clouds[cid] = cl
	w.curCloud = cid
	w.tickClouds(0)
	return cid
}

func (w *World) cloudSunDir() (float32, float32, float32) {
	for _, e := range w.ents {
		if e != nil && e.lgtKind == 1 {
			return dirLightOffset(float64(e.pitch), float64(e.yaw))
		}
	}
	return -0.35, 0.62, 0.70
}

func (w *World) tickClouds(dt float32) {
	var cam math32.Vector3
	hasCam := w.cam != nil
	if hasCam {
		w.cam.WorldPosition(&cam)
	}
	windX := float32(w.wx.windX)
	windZ := float32(-w.wx.windZ)
	if windX*windX+windZ*windZ < 1e-4 {
		windX = 1
	}
	for _, c := range w.clouds {
		if c == nil {
			continue
		}
		c.time += dt * c.speed
		// About 18 m/s at speed 1, as the cards drifted
		c.driftX -= windX * 18 * dt * c.speed
		c.driftZ -= windZ * 18 * dt * c.speed
		e := w.ents[c.ent]
		if e == nil || e.node == nil {
			continue
		}
		n := e.node.GetNode()
		n.SetVisible(c.cover > 0.02)
		if hasCam {
			r := c.cloudRadius()
			n.SetPosition(cam.X, cam.Y+c.y, cam.Z)
			n.SetScale(r, 1, r)
		}
	}
}

func (w *World) cloudCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createvolumetricclouds": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.createVolumetricClouds(float32(argN(a, 0, 110)), float32(argN(a, 1, 280))))), nil
		}),
		"setclouds": n(func(a []value.Value) (value.Value, error) {
			if w.ready && w.scene != nil && len(w.clouds) == 0 {
				w.createVolumetricClouds(110, 280)
			}
			c := w.clouds[w.curCloud]
			if c == nil {
				for _, cl := range w.clouds {
					c = cl
					break
				}
			}
			if c != nil {
				c.cover = float32(argN(a, 0, 0.52))
				if len(a) >= 2 {
					c.density = float32(argN(a, 1, 1.1))
				}
				if len(a) >= 3 {
					c.speed = float32(argN(a, 2, 1))
				}
			}
			return z()
		}),
		"setcloudcoverage": n(func(a []value.Value) (value.Value, error) {
			c := w.clouds[argI(a, 0, w.curCloud)]
			if c == nil {
				c = w.clouds[w.curCloud]
			}
			if c != nil {
				off := 0
				if w.clouds[argI(a, 0, 0)] != nil && len(a) >= 2 {
					off = 1
				}
				c.cover = float32(argN(a, off, 0.52))
			}
			return z()
		}),
		"setcloudspeed": n(func(a []value.Value) (value.Value, error) {
			c := w.clouds[w.curCloud]
			if c != nil {
				c.speed = float32(argN(a, 0, 1))
			}
			return z()
		}),
		"setclouddensity": n(func(a []value.Value) (value.Value, error) {
			c := w.clouds[w.curCloud]
			if c != nil {
				c.density = float32(argN(a, 0, 1))
			}
			return z()
		}),
		"getcloudcoverage": n(func(a []value.Value) (value.Value, error) {
			c := w.clouds[argI(a, 0, w.curCloud)]
			if c == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(c.cover)), nil
		}),
	}
}
