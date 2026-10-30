package runtime

import (
	"math"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// Clouds are a fixed pool of camera-facing cards. One shared quad, no
// noise texture and no raymarch. Coverage decides which cards are live.

const cloudCardCount = 36

type cloudCard struct {
	ent        int
	ox, oz, oy float32
	sx, sy     float32
	seed       float32
	rank       float32
}

type cloudLayer struct {
	ent     int
	cards   []cloudCard
	cover   float32
	density float32
	speed   float32
	time    float32
	y       float32
	span    float32
	color   math32.Color
}

type cloudMat struct {
	*material.Standard
	w    *World
	c    *cloudLayer
	seed float32
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
	var view, proj, vp math32.Matrix4
	if m.w.cam != nil {
		m.w.cam.WorldPosition(&cam)
		m.w.cam.ViewMatrix(&view)
		m.w.cam.ProjMatrix(&proj)
		vp.MultiplyMatrices(&proj, &view)
	}
	setUni3f(gs, "CloudCam", cam.X, cam.Y, cam.Z)
	setUniMat4(gs, "CloudVP", &vp)
	sx, sy, sz := m.w.cloudSunDir()
	setUni3f(gs, "CloudSun", sx, sy, sz)
	setUni1f(gs, "CloudSeed", m.seed)
	setUni1f(gs, "CloudDensity", c.density)
	setUni3f(gs, "CloudColor", c.color.R, c.color.G, c.color.B)
}

func cloudHash(i int) float32 {
	x := uint32(i*374761393 + 668265263)
	x = (x ^ (x >> 13)) * 1274126177
	return float32(x&0xffffff) / float32(0xffffff)
}

func cloudCardGeometry() *geometry.Geometry {
	pos := math32.NewArrayF32(0, 12)
	nor := math32.NewArrayF32(0, 12)
	uvs := math32.NewArrayF32(0, 8)
	idx := math32.NewArrayU32(0, 6)
	pos.Append(-0.5, -0.5, 0, 0.5, -0.5, 0, 0.5, 0.5, 0, -0.5, 0.5, 0)
	nor.Append(0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1)
	uvs.Append(0, 0, 1, 0, 1, 1, 0, 1)
	idx.Append(0, 1, 2, 0, 2, 3)
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

func wrapSpan(v, span float32) float32 {
	if span < 1 {
		return v
	}
	s := span * 2
	v = float32(math.Mod(float64(v+span), float64(s)))
	if v < 0 {
		v += s
	}
	return v - span
}

func (w *World) createVolumetricClouds(y, scale float32) int {
	if y < 16 {
		y = 16
	}
	if scale < 80 {
		scale = 80
	}
	geom := cloudCardGeometry()
	cl := &cloudLayer{
		cover: 0.52, density: 1, speed: 1, y: y, span: scale,
		color: math32.Color{0.96, 0.97, 0.99},
	}
	cl.cards = make([]cloudCard, cloudCardCount)
	for i := 0; i < cloudCardCount; i++ {
		if i > 0 {
			geom.Incref()
		}
		h1 := cloudHash(i + 1)
		h2 := cloudHash(i + 17)
		h3 := cloudHash(i + 29)
		rad := scale * (0.16 + 0.78*h1)
		ang := float64(i) * 2.399963229728653
		sx := scale * (0.14 + 0.12*h2)
		card := cloudCard{
			ox:   float32(math.Cos(ang)) * rad,
			oz:   float32(math.Sin(ang)) * rad,
			oy:   y * (0.16 + 0.42*h3),
			sx:   sx,
			sy:   sx * (0.36 + 0.22*cloudHash(i+41)),
			seed: h2*2 - 1,
			rank: cloudHash(i + 7),
		}
		mat := material.NewStandard(&math32.Color{0.96, 0.97, 0.99})
		mat.SetUseLights(material.UseLightNone)
		mat.SetSide(material.SideDouble)
		mat.SetTransparent(true)
		mat.SetOpacity(1)
		mat.SetDepthMask(false)
		mat.SetDepthTest(true)
		mat.SetShader("mbclouds")
		cm := &cloudMat{Standard: mat, w: w, c: cl, seed: card.seed}
		mesh := graphic.NewMesh(geom, cm)
		mesh.SetCullable(false)
		mesh.SetRenderOrder(85)
		mesh.SetScale(card.sx, card.sy, 1)
		id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat}, 0)
		if e := w.ents[id]; e != nil {
			e.name = "clouds"
		}
		card.ent = id
		cl.cards[i] = card
		if i == 0 {
			cl.ent = id
		}
	}
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
	driftX := float32(w.wx.windX)
	driftZ := float32(-w.wx.windZ)
	if driftX*driftX+driftZ*driftZ < 1e-4 {
		driftX = 1
	}
	for _, c := range w.clouds {
		if c == nil {
			continue
		}
		c.time += dt * c.speed
		cover := c.cover
		if cover < 0 {
			cover = 0
		}
		if cover > 1 {
			cover = 1
		}
		grow := 0.82 + 0.45*cover
		shift := c.time * 18
		for i := range c.cards {
			card := &c.cards[i]
			e := w.ents[card.ent]
			if e == nil || e.node == nil {
				continue
			}
			show := cover > 0.02 && card.rank <= cover
			e.node.GetNode().SetVisible(show)
			if !show || !hasCam {
				continue
			}
			x := wrapSpan(card.ox+driftX*shift, c.span)
			z := wrapSpan(card.oz+driftZ*shift, c.span)
			e.node.GetNode().SetPosition(cam.X+x, cam.Y+card.oy, cam.Z+z)
			e.node.GetNode().SetScale(card.sx*grow, card.sy*grow, 1)
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
