package runtime

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	bsaudio "bitshinbasic/internal/audio"
)

// Midpoint-displacement bolts after fogleman/lightning (MIT-style algorithm).
// Visible ribbon + ambient flash. No GL 4.x.

type lightningBolt struct {
	mesh *graphic.Mesh
	life float64
}

func displaceBolt(a, b math32.Vector3, depth int, jag float32) []math32.Vector3 {
	if depth <= 0 {
		return []math32.Vector3{a, b}
	}
	pts := []math32.Vector3{a, b}
	amp := jag
	for d := 0; d < depth; d++ {
		next := make([]math32.Vector3, 0, len(pts)*2)
		for i := 0; i < len(pts)-1; i++ {
			p0, p1 := pts[i], pts[i+1]
			mid := math32.Vector3{
				(p0.X + p1.X) * 0.5,
				(p0.Y + p1.Y) * 0.5,
				(p0.Z + p1.Z) * 0.5,
			}
			dx, dy, dz := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
			// perpendicular in XZ plus a little Y
			mid.X += (rand.Float32()*2 - 1) * amp
			mid.Z += (rand.Float32()*2 - 1) * amp
			mid.Y += (rand.Float32()*2 - 1) * amp * 0.35
			_ = dx
			_ = dy
			_ = dz
			next = append(next, p0, mid)
		}
		next = append(next, pts[len(pts)-1])
		pts = next
		amp *= 0.48
	}
	return pts
}

func boltRibbon(pts []math32.Vector3, width float32) *geometry.Geometry {
	g := geometry.NewGeometry()
	if len(pts) < 2 {
		return g
	}
	nseg := len(pts) - 1
	pos := math32.NewArrayF32(0, nseg*18)
	nor := math32.NewArrayF32(0, nseg*18)
	uvs := math32.NewArrayF32(0, nseg*12)
	up := math32.Vector3{0, 1, 0}
	for i := 0; i < nseg; i++ {
		a, b := pts[i], pts[i+1]
		dir := math32.Vector3{b.X - a.X, b.Y - a.Y, b.Z - a.Z}
		side := math32.Vector3{dir.Y*up.Z - dir.Z*up.Y, dir.Z*up.X - dir.X*up.Z, dir.X*up.Y - dir.Y*up.X}
		if side.Length() < 1e-5 {
			side = math32.Vector3{1, 0, 0}
		} else {
			side.Normalize()
		}
		side.MultiplyScalar(width)
		a0 := math32.Vector3{a.X - side.X, a.Y - side.Y, a.Z - side.Z}
		a1 := math32.Vector3{a.X + side.X, a.Y + side.Y, a.Z + side.Z}
		b0 := math32.Vector3{b.X - side.X, b.Y - side.Y, b.Z - side.Z}
		b1 := math32.Vector3{b.X + side.X, b.Y + side.Y, b.Z + side.Z}
		quad := []math32.Vector3{a0, b0, b1, a0, b1, a1}
		for _, p := range quad {
			pos.Append(p.X, p.Y, p.Z)
			nor.Append(0, 0, 1)
		}
		uvs.Append(0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1)
	}
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

func (w *World) strikeLightning(ax, ay, az, bx, by, bz float64) {
	w.ensureWeather()
	w.wx.flash = 0.95 + rand.Float64()*0.35
	w.queueThunder(bx, by, bz)
	if w.scene == nil {
		return
	}
	a := math32.Vector3{float32(ax), float32(ay), float32(az)}
	b := math32.Vector3{float32(bx), float32(by), float32(bz)}
	pts := displaceBolt(a, b, 5, 3.2)
	// occasional fork
	if rand.Float32() < 0.55 && len(pts) > 6 {
		mid := pts[len(pts)/2]
		end := math32.Vector3{
			mid.X + (rand.Float32()*2-1)*8,
			mid.Y - 6 - rand.Float32()*8,
			mid.Z + (rand.Float32()*2-1)*8,
		}
		fork := displaceBolt(mid, end, 3, 2.0)
		pts = append(pts, fork...)
	}
	geom := boltRibbon(pts, 0.07)
	mat := material.NewStandard(&math32.Color{0.85, 0.92, 1})
	mat.SetUseLights(material.UseLightNone)
	mat.SetEmissiveColor(&math32.Color{0.95, 0.97, 1})
	mat.SetOpacity(0.95)
	mat.SetTransparent(true)
	mat.SetDepthMask(false)
	mesh := graphic.NewMesh(geom, mat)
	mesh.SetRenderOrder(90)
	w.scene.Add(mesh)
	w.wx.bolts = append(w.wx.bolts, lightningBolt{mesh: mesh, life: 0.28})
}

func (w *World) autoStormBolt() {
	var cx, cy, cz float32
	if w.cam != nil {
		var p math32.Vector3
		w.cam.WorldPosition(&p)
		cx, cy, cz = p.X, p.Y, p.Z
	}
	ox := float64(cx) + (rand.Float64()*2-1)*18
	oz := float64(cz) + (rand.Float64()*2-1)*18
	w.strikeLightning(ox, float64(cy)+28, oz, ox+(rand.Float64()*2-1)*4, 0.4, oz+(rand.Float64()*2-1)*4)
}

func (w *World) tickLightning(dt float32) {
	alive := w.wx.bolts[:0]
	for i := range w.wx.bolts {
		b := w.wx.bolts[i]
		b.life -= float64(dt)
		if b.life <= 0 {
			if b.mesh != nil {
				if p := b.mesh.GetNode().Parent(); p != nil {
					p.GetNode().Remove(b.mesh)
				}
				b.mesh.SetVisible(false)
			}
			continue
		}
		if b.mesh != nil {
			if mat, ok := b.mesh.GetMaterial(0).(*material.Standard); ok {
				mat.SetOpacity(float32(math.Max(0, b.life/0.28)))
			}
		}
		alive = append(alive, b)
	}
	w.wx.bolts = alive
}

func (w *World) clearLightning() {
	for i := range w.wx.bolts {
		if w.wx.bolts[i].mesh != nil {
			if p := w.wx.bolts[i].mesh.GetNode().Parent(); p != nil {
				p.GetNode().Remove(w.wx.bolts[i].mesh)
			}
			w.wx.bolts[i].mesh.SetVisible(false)
		}
	}
	w.wx.bolts = nil
}

func (w *World) listenerXYZ() (x, y, z float64) {
	if w.cam != nil {
		var p math32.Vector3
		w.cam.WorldPosition(&p)
		fx, fy, fz := fromG3N(p.X, p.Y, p.Z)
		return float64(fx), float64(fy), float64(fz)
	}
	lx, ly, lz, _ := w.listenerPose()
	return lx, ly, lz
}

func (w *World) queueThunder(x, y, z float64) {
	lx, ly, lz := w.listenerXYZ()
	dx, dy, dz := x-lx, y-ly, z-lz
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	w.wx.thunder = append(w.wx.thunder, thunderWait{wait: dist / 343, dist: dist})
}

func (w *World) tickThunder(dt float64) {
	if len(w.wx.thunder) == 0 {
		return
	}
	alive := w.wx.thunder[:0]
	for _, ev := range w.wx.thunder {
		ev.wait -= dt
		if ev.wait > 0 {
			alive = append(alive, ev)
			continue
		}
		w.playThunder(ev.dist)
	}
	w.wx.thunder = alive
}

func (w *World) ensureThunderClip() *bsaudio.Clip {
	if w.thunderClip != nil {
		return w.thunderClip
	}
	candidates := []string{
		"assets/thunder.ogg",
		"assets/sfx/thunder.ogg",
		"data/thunder.ogg",
		"thunder.ogg",
	}
	for _, rel := range candidates {
		path := w.resolve(rel)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			clip, err := bsaudio.Load(path)
			if err == nil && clip != nil {
				w.thunderClip = clip
				return clip
			}
		}
		if w.pak != nil {
			if extracted, err := w.pak.extract(filepath.ToSlash(rel)); err == nil {
				clip, err := bsaudio.Load(extracted)
				if err == nil && clip != nil {
					w.thunderClip = clip
					return clip
				}
			}
		}
	}
	if !w.wx.thunderOnce {
		fmt.Println("Weather: no thunder.ogg — using generated rumble")
		w.wx.thunderOnce = true
	}
	w.thunderClip = bsaudio.GenerateThunder(1.8)
	return w.thunderClip
}

func (w *World) playThunder(dist float64) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Weather: thunder:", r)
		}
	}()
	clip := w.ensureThunderClip()
	if clip == nil {
		return
	}
	vol := 1 - dist/180
	if vol < 0.12 {
		vol = 0.12
	}
	if vol > 1 {
		vol = 1
	}
	pitch := 1 - math.Min(dist, 140)/140*0.32
	if pitch < 0.68 {
		pitch = 0.68
	}
	if w.thunderVoice != nil {
		w.thunderVoice.Close()
	}
	voice, err := clip.PlayAt(vol, pitch, 0, false)
	if err != nil {
		if !w.wx.thunderOnce {
			fmt.Println("Weather: thunder play failed:", err)
			w.wx.thunderOnce = true
		}
		return
	}
	w.thunderVoice = voice
}
