package runtime

import (
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

// 3.3 stand-in for Terrain-OpenGL compute volumetric clouds:
// a large dome + fragment raymarch through layered 2D noise. No GL 4.3.

type cloudLayer struct {
	ent     int
	tex     int
	cover   float32
	density float32
	speed   float32
	time    float32
	y       float32
	color   math32.Color
}

type cloudMat struct {
	*material.Standard
	w *World
	c *cloudLayer
}

func (m *cloudMat) GetMaterial() *material.Material { return m.Standard.GetMaterial() }
func (m *cloudMat) Dispose()                        { m.Standard.Dispose() }

func (m *cloudMat) RenderSetup(gs *gls.GLS) {
	m.Standard.RenderSetup(gs)
	if m.w == nil || m.c == nil || gs == nil {
		return
	}
	c := m.c
	cam := math32.Vector3{}
	if m.w.cam != nil {
		m.w.cam.WorldPosition(&cam)
	}
	setUni3f(gs, "CloudCam", cam.X, cam.Y, cam.Z)
	setUni3f(gs, "CloudSun", -0.35, 0.55, 0.75)
	setUni1f(gs, "CloudTime", c.time)
	setUni1f(gs, "CloudCover", c.cover)
	setUni1f(gs, "CloudDensity", c.density)
	setUni3f(gs, "CloudColor", c.color.R, c.color.G, c.color.B)
	setUni3f(gs, "CloudSkyBot", 0.82, 0.86, 0.92)
	setUni3f(gs, "CloudWind", float32(m.w.wx.windX), float32(m.w.wx.windZ), float32(m.w.wx.windStr))
	has := 0
	if slot := m.w.texs[c.tex]; slot != nil && slot.tex != nil {
		slot.tex.RenderSetup(gs, 4, 4)
		setUni1i(gs, "CloudNoise", 4)
		has = 1
	}
	setUni1i(gs, "CloudHasNoise", has)
}

func (w *World) createVolumetricClouds(y, scale float32) int {
	if scale < 40 {
		scale = 40
	}
	geom := geometry.NewSphere(float64(scale), 24, 16)
	mat := material.NewStandard(&math32.Color{0.92, 0.94, 0.96})
	mat.SetUseLights(material.UseLightNone)
	mat.SetSide(material.SideBack)
	mat.SetTransparent(true)
	mat.SetOpacity(0.65)
	mat.SetDepthMask(false)
	mat.SetShader("mbclouds")
	cl := &cloudLayer{
		cover: 0.52, density: 1.1, speed: 1, y: y,
		color: math32.Color{0.95, 0.96, 0.98},
	}
	cl.tex = w.addProcTexture("cloud")
	mesh := graphic.NewMesh(geom, &cloudMat{Standard: mat, w: w, c: cl})
	mesh.SetCullable(false)
	mesh.SetRenderOrder(80)
	id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat}, 0)
	if e := w.ents[id]; e != nil {
		e.name = "clouds"
	}
	cl.ent = id
	if w.clouds == nil {
		w.clouds = map[int]*cloudLayer{}
	}
	cid := w.takeHandle(&w.freeClouds, &w.nextCloud)
	w.clouds[cid] = cl
	w.curCloud = cid
	return cid
}

func (w *World) tickClouds(dt float32) {
	for _, c := range w.clouds {
		if c == nil {
			continue
		}
		c.time += dt * c.speed
		if e := w.ents[c.ent]; e != nil && e.node != nil && w.cam != nil {
			p := worldPos(w.cam.GetNode())
			e.node.GetNode().SetPosition(p.X, p.Y+c.y, p.Z)
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
