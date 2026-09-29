package runtime

import (
	"math"
	"math/rand"
	"strings"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

type particle struct {
	x, y, z    float64
	vx, vy, vz float64
	life, max  float64
	spin       float64
	spinVel    float64
	wobble     float64
	vis        *partVis
}

type partVis struct {
	mesh *graphic.Mesh
	mat  *material.Standard
	cube bool
}

type emitter struct {
	ent     int
	parent  int
	space2D bool
	weather bool
	follow  bool
	visible bool

	x, y, z             float64
	areaX, areaY, areaZ float64

	rate                float64
	max                 int
	acc                 float64
	life                float64
	speed               float64
	vx, vy, vz          float64
	gravX, gravY, gravZ float64
	windX, windY, windZ float64
	drag                float64
	cone                float64
	size0, size1        float64
	tall                float64
	cr0, cg0, cb0, ca0  float64
	cr1, cg1, cb1, ca1  float64
	tex                 int
	img                 *ebiten.Image
	duration            float64
	elapsed             float64
	looping             bool
	emitting            bool
	alignY              bool
	style               int // 0 default, 1 rain streak, 2 flake
	recycle             bool
	rateBase            float64
	ca0Base, ca1Base    float64
	parts               []particle
}

func defaultEmitter(space2D bool) *emitter {
	e := &emitter{
		visible:  true,
		emitting: true,
		looping:  true,
		rate:     20,
		max:      64,
		life:     1.2,
		speed:    4,
		cone:     180,
		size0:    0.18,
		size1:    0.06,
		tall:     1,
		cr0:      1, cg0: 0.7, cb0: 0.25, ca0: 1,
		cr1: 1, cg1: 0.35, cb1: 0.08, ca1: 0,
		drag: 0.15,
	}
	if space2D {
		e.x, e.y = 400, 240
		e.speed = 80
		e.size0, e.size1 = 6, 2
		e.gravY = 80
		e.vy = -40
		e.cone = 50
	} else {
		e.vy = 1
		e.gravY = -6
	}
	return e
}

func (w *World) newEmitter(parent int, space2D bool) int {
	em := defaultEmitter(space2D)
	em.parent = parent
	em.space2D = space2D
	if space2D || w.mode2D || w.scene == nil {
		em.space2D = space2D || w.mode2D
		id := w.nextEmit
		w.nextEmit++
		w.emitters[id] = em
		return id
	}
	id := w.addEntity(&Entity{node: core.NewNode()}, parent)
	em.ent = id
	w.emitters[id] = em
	return id
}

func (w *World) emitterOf(id int) *emitter {
	return w.emitters[id]
}

func (w *World) releaseEmitterParts(e *emitter) {
	if e == nil {
		return
	}
	for i := range e.parts {
		w.recyclePartVis(e.parts[i].vis)
		e.parts[i].vis = nil
	}
	e.parts = e.parts[:0]
}

func (w *World) freeEmitter(id int) {
	e := w.emitters[id]
	if e == nil {
		return
	}
	w.releaseEmitterParts(e)
	delete(w.emitters, id)
	if e.ent != 0 && w.ents[e.ent] != nil {
		w.freeEntityID(e.ent)
	}
}

func (w *World) particleGeom() *geometry.Geometry {
	if w.partGeom == nil {
		w.partGeom = geometry.NewPlane(1, 1)
	}
	return w.partGeom
}

func (w *World) particleCube() *geometry.Geometry {
	if w.partCube == nil {
		w.partCube = geometry.NewCube(1)
	}
	return w.partCube
}

func (w *World) acquirePartVis(cube bool) *partVis {
	pool := &w.partPool
	if cube {
		pool = &w.cubePool
	}
	if len(*pool) > 0 {
		v := (*pool)[len(*pool)-1]
		*pool = (*pool)[:len(*pool)-1]
		if v.mesh != nil {
			v.mesh.SetVisible(true)
			if w.scene != nil && v.mesh.GetNode().Parent() == nil {
				w.scene.Add(v.mesh)
			}
		}
		return v
	}
	if w.scene == nil {
		return nil
	}
	mat := material.NewStandard(&math32.Color{1, 1, 1})
	var mesh *graphic.Mesh
	if cube {
		mat.SetSide(material.SideFront)
		mat.SetDepthMask(true)
		mesh = graphic.NewMesh(w.particleCube(), mat)
	} else {
		mat.SetShader("mbsoft")
		mat.SetUseLights(material.UseLightNone)
		mat.SetSide(material.SideDouble)
		mat.SetTransparent(true)
		mat.SetDepthMask(false)
		mesh = graphic.NewMesh(w.particleGeom(), mat)
		mesh.SetRenderOrder(80)
	}
	w.scene.Add(mesh)
	return &partVis{mesh: mesh, mat: mat, cube: cube}
}

func (w *World) recyclePartVis(v *partVis) {
	if v == nil || v.mesh == nil {
		return
	}
	v.mesh.SetVisible(false)
	if v.cube {
		w.cubePool = append(w.cubePool, v)
		return
	}
	w.partPool = append(w.partPool, v)
}

func applyEmitterShape(e *emitter, name string) {
	if e == nil {
		return
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "streak", "rain":
		e.style = 1
	case "flake", "snow":
		e.style = 2
	case "cube", "mesh", "3d":
		e.style = 3
	default:
		e.style = 0
	}
}

func (w *World) emitterOrigin(e *emitter) (x, y, z float64) {
	x, y, z = e.x, e.y, e.z
	if e.ent != 0 {
		if ent := w.ents[e.ent]; ent != nil && ent.node != nil {
			p := worldPos(ent.node.GetNode())
			bx, by, bz := fromG3N(p.X, p.Y, p.Z)
			x, y, z = float64(bx)+e.x, float64(by)+e.y, float64(bz)+e.z
			if !ent.node.GetNode().Visible() {
				e.visible = false
			} else {
				e.visible = true
			}
		}
	} else if e.parent != 0 {
		if ent := w.ents[e.parent]; ent != nil && ent.node != nil {
			p := worldPos(ent.node.GetNode())
			bx, by, bz := fromG3N(p.X, p.Y, p.Z)
			x, y, z = float64(bx)+e.x, float64(by)+e.y, float64(bz)+e.z
		} else if s := w.sprites[e.parent]; s != nil {
			x, y = s.x+e.x, s.y+e.y
		}
	}
	if e.follow && w.cam != nil {
		p := worldPos(w.cam.GetNode())
		bx, by, bz := fromG3N(p.X, p.Y, p.Z)
		x, y, z = float64(bx)+e.x, float64(by)+e.y, float64(bz)+e.z
	}
	if e.areaX > 0 {
		x += (rand.Float64()*2 - 1) * e.areaX
	}
	if e.areaY > 0 {
		y += (rand.Float64()*2 - 1) * e.areaY
	}
	if e.areaZ > 0 {
		z += (rand.Float64()*2 - 1) * e.areaZ
	}
	return
}

func coneVelocity(bx, by, bz, speed, coneDeg float64) (vx, vy, vz float64) {
	dx, dy, dz := bx, by, bz
	if dx == 0 && dy == 0 && dz == 0 {
		dy = 1
	}
	inv := 1 / math.Sqrt(dx*dx+dy*dy+dz*dz)
	dx, dy, dz = dx*inv, dy*inv, dz*inv
	half := coneDeg * 0.5 * math.Pi / 180
	if half < 0 {
		half = 0
	}
	if half > math.Pi {
		half = math.Pi
	}
	cosA := math.Cos(rand.Float64() * half)
	phi := rand.Float64() * math.Pi * 2
	// orthonormal basis around (dx,dy,dz)
	ax, ay, az := 0.0, 1.0, 0.0
	if math.Abs(dy) > 0.9 {
		ax, ay, az = 1, 0, 0
	}
	tx := ay*dz - az*dy
	ty := az*dx - ax*dz
	tz := ax*dy - ay*dx
	tl := math.Sqrt(tx*tx + ty*ty + tz*tz)
	if tl < 1e-8 {
		tx, ty, tz = 1, 0, 0
		tl = 1
	}
	tx, ty, tz = tx/tl, ty/tl, tz/tl
	bx2 := dy*tz - dz*ty
	by2 := dz*tx - dx*tz
	bz2 := dx*ty - dy*tx
	sinA := math.Sqrt(math.Max(0, 1-cosA*cosA))
	dirx := dx*cosA + (tx*math.Cos(phi)+bx2*math.Sin(phi))*sinA
	diry := dy*cosA + (ty*math.Cos(phi)+by2*math.Sin(phi))*sinA
	dirz := dz*cosA + (tz*math.Cos(phi)+bz2*math.Sin(phi))*sinA
	if speed == 0 {
		speed = 1
	}
	return dirx * speed, diry * speed, dirz * speed
}

func (w *World) spawnParticle(e *emitter) {
	if e == nil || e.max < 1 {
		return
	}
	if len(e.parts) >= e.max {
		w.recyclePartVis(e.parts[0].vis)
		e.parts[0].vis = nil
		copy(e.parts, e.parts[1:])
		e.parts = e.parts[:len(e.parts)-1]
	}
	x, y, z := w.emitterOrigin(e)
	spd := e.speed
	bx, by, bz := e.vx, e.vy, e.vz
	if bx == 0 && by == 0 && bz == 0 {
		if e.space2D {
			by = -1
		} else {
			by = 1
		}
	}
	vx, vy, vz := coneVelocity(bx, by, bz, spd, e.cone)
	p := particle{
		x: x, y: y, z: z,
		vx: vx, vy: vy, vz: vz,
		life: e.life, max: e.life,
	}
	if e.style == 2 || e.style == 3 {
		p.spin = rand.Float64() * math.Pi * 2
		p.spinVel = (rand.Float64()*2 - 1) * 2.8
		p.wobble = rand.Float64() * math.Pi * 2
	}
	if p.max <= 0 {
		p.max = 0.001
		p.life = 0.001
	}
	if !e.space2D && w.ready && !w.mode2D {
		vis := w.acquirePartVis(e.style == 3)
		if vis != nil {
			if vis.mat != nil && !vis.cube {
				switch e.style {
				case 2:
					vis.mat.SetShader("mbflake")
				case 1:
					vis.mat.SetShader("mbpart")
				default:
					vis.mat.SetShader("mbsoft")
				}
			}
			if e.tex != 0 {
				if t := w.texs[e.tex]; t != nil && t.tex != nil && vis.mat != nil {
					vis.mat.AddTexture(t.tex)
				}
			}
			gx, gy, gz := toG3N(float32(x), float32(y), float32(z))
			vis.mesh.SetPosition(gx, gy, gz)
			p.vis = vis
		}
	}
	e.parts = append(e.parts, p)
}

func (w *World) seedEmitter(e *emitter, n int) {
	if e == nil || n <= 0 {
		return
	}
	if n > e.max {
		n = e.max
	}
	for i := 0; i < n; i++ {
		w.spawnParticle(e)
	}
}

func (w *World) resetWeatherFlake(e *emitter, p *particle) {
	x, y, z := w.emitterOrigin(e)
	bx, by, bz := e.vx, e.vy, e.vz
	if bx == 0 && by == 0 && bz == 0 {
		by = -1
	}
	vx, vy, vz := coneVelocity(bx, by, bz, e.speed, e.cone)
	p.x, p.y, p.z = x, y, z
	p.vx, p.vy, p.vz = vx, vy, vz
	p.life = e.life
	p.max = e.life
	if p.max <= 0 {
		p.max = 0.001
		p.life = 0.001
	}
	p.spin = rand.Float64() * math.Pi * 2
	p.spinVel = (rand.Float64()*2 - 1) * 2.8
	p.wobble = rand.Float64() * math.Pi * 2
}

func (w *World) emitterActive(e *emitter) bool {
	if e == nil || !e.visible {
		return false
	}
	if e.ent != 0 {
		if ent := w.ents[e.ent]; ent != nil && ent.node != nil && !ent.node.GetNode().Visible() {
			return false
		}
	}
	return true
}

func (w *World) tickEmitters(dt float32) {
	d := float64(dt)
	if d <= 0 {
		return
	}
	var camPos math32.Vector3
	hasCam := w.cam != nil
	if hasCam {
		w.cam.WorldPosition(&camPos)
	}
	for _, e := range w.emitters {
		if e.duration > 0 {
			e.elapsed += d
			if e.elapsed >= e.duration {
				if e.looping {
					e.elapsed = 0
				} else {
					e.emitting = false
				}
			}
		}
		if e.emitting && e.rate > 0 && w.emitterActive(e) {
			e.acc += e.rate * d
			for e.acc >= 1 {
				e.acc--
				w.spawnParticle(e)
			}
		}
		alive := e.parts[:0]
		for i := range e.parts {
			p := e.parts[i]
			p.life -= d
			p.vx += (e.gravX + e.windX) * d
			p.vy += (e.gravY + e.windY) * d
			p.vz += (e.gravZ + e.windZ) * d
			if e.drag > 0 {
				damp := math.Max(0, 1-e.drag*d)
				p.vx *= damp
				p.vy *= damp
				p.vz *= damp
			}
			p.x += p.vx * d
			p.y += p.vy * d
			p.z += p.vz * d
			if e.style == 2 || e.style == 3 {
				p.wobble += d
				p.spin += p.spinVel * d
			}
			if e.style == 2 {
				p.x += math.Sin(p.wobble*1.7+p.spin) * 0.62 * d
				p.z += math.Cos(p.wobble*1.25) * 0.48 * d
			}
			hit := e.recycle && (p.life <= 0 || p.y < 0.04)
			hitWater, hitGround := false, false
			if e.weather && !e.space2D {
				ground, water, hasWater := w.weatherHitPlanes(p.x, p.z)
				hitWater = hasWater && p.y <= water
				hitGround = p.y <= ground+0.05
				if hitWater || hitGround {
					hit = true
				}
			}
			if p.life <= 0 || hit {
				if e.weather && (p.life <= 0 || hitWater || hitGround) {
					w.weatherImpact(p.x, p.y, p.z, hitWater)
				}
				if e.recycle {
					w.resetWeatherFlake(e, &p)
				} else {
					w.recyclePartVis(p.vis)
					continue
				}
			}
			if p.vis != nil && p.vis.mesh != nil {
				t := 1 - p.life/p.max
				if t < 0 {
					t = 0
				}
				if t > 1 {
					t = 1
				}
				s := e.size0 + (e.size1-e.size0)*t
				if s < 0.001 {
					s = 0.001
				}
				sy := s * e.tall
				if sy < 0.001 {
					sy = 0.001
				}
				gx, gy, gz := toG3N(float32(p.x), float32(p.y), float32(p.z))
				p.vis.mesh.SetPosition(gx, gy, gz)
				if e.style == 3 {
					p.vis.mesh.SetScale(float32(s), float32(s), float32(s))
					p.vis.mesh.SetRotation(float32(p.spin), float32(p.spin*0.73), float32(p.wobble))
				} else {
					p.vis.mesh.SetScale(float32(s), float32(sy), 1)
				}
				if e.style != 3 && hasCam {
					if e.alignY {
						dx := camPos.X - gx
						dz := camPos.Z - gz
						p.vis.mesh.SetRotation(0, math32.Atan2(dx, dz), 0)
					} else {
						p.vis.mesh.LookAt(&camPos, &math32.Vector3{0, 1, 0})
						if e.style == 2 {
							p.vis.mesh.RotateZ(float32(p.spin))
						}
					}
				}
				if p.vis.mat != nil {
					cr := e.cr0 + (e.cr1-e.cr0)*t
					cg := e.cg0 + (e.cg1-e.cg0)*t
					cb := e.cb0 + (e.cb1-e.cb0)*t
					ca := e.ca0 + (e.ca1-e.ca0)*t
					p.vis.mat.SetColor(&math32.Color{float32(cr), float32(cg), float32(cb)})
					if ca < 0 {
						ca = 0
					}
					if ca > 1 {
						ca = 1
					}
					p.vis.mat.SetOpacity(float32(ca))
				}
			}
			alive = append(alive, p)
		}
		e.parts = alive
	}
}

func (w *World) drawParticles2D(screen *ebiten.Image) {
	if !w.mode2D {
		return
	}
	for _, e := range w.emitters {
		if e == nil || !e.space2D {
			continue
		}
		for _, p := range e.parts {
			t := 1 - p.life/p.max
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			s := e.size0 + (e.size1-e.size0)*t
			if s < 0.5 {
				s = 0.5
			}
			cr := e.cr0 + (e.cr1-e.cr0)*t
			cg := e.cg0 + (e.cg1-e.cg0)*t
			cb := e.cb0 + (e.cb1-e.cb0)*t
			ca := e.ca0 + (e.ca1-e.ca0)*t
			if ca < 0 {
				ca = 0
			}
			if ca > 1 {
				ca = 1
			}
			c := color.RGBA{u8f(cr), u8f(cg), u8f(cb), u8f(ca)}
			if e.img != nil {
				opt := &ebiten.DrawImageOptions{}
				b := e.img.Bounds()
				opt.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
				opt.GeoM.Scale(s/float64(b.Dx()), s*e.tall/float64(b.Dy()))
				opt.GeoM.Translate(p.x, p.y)
				opt.ColorScale.Scale(float32(cr), float32(cg), float32(cb), float32(ca))
				screen.DrawImage(e.img, opt)
				continue
			}
			vector.FillCircle(screen, float32(p.x), float32(p.y), float32(s), c, true)
		}
	}
}

func (w *World) drawFog2D(screen *ebiten.Image) {
	if !w.mode2D || w.fogMode == 0 || screen == nil {
		return
	}
	a := 0.22
	if w.fogFar > w.fogNear && w.fogFar > 0 {
		a = float64(40 / max32(w.fogFar, 8))
	}
	if w.wx.mode == "fog" {
		a += 0.18 * w.wx.intensity
	}
	if a > 0.55 {
		a = 0.55
	}
	c := color.RGBA{u8f(float64(w.fogRGB.R)), u8f(float64(w.fogRGB.G)), u8f(float64(w.fogRGB.B)), u8f(a)}
	b := screen.Bounds()
	vector.FillRect(screen, 0, 0, float32(b.Dx()), float32(b.Dy()), c, true)
}

func u8f(n float64) uint8 {
	if n < 0 {
		return 0
	}
	if n > 1 {
		return 255
	}
	return uint8(n * 255)
}

func (e *emitter) setColor(a []float64) {
	if len(a) < 3 {
		return
	}
	c0 := rgb(a[0], a[1], a[2])
	e.cr0, e.cg0, e.cb0 = float64(c0.R), float64(c0.G), float64(c0.B)
	e.ca0 = 1
	if len(a) >= 4 {
		e.ca0 = a[3]
		if e.ca0 > 1 {
			e.ca0 /= 255
		}
	}
	if len(a) >= 7 {
		c1 := rgb(a[4], a[5], a[6])
		e.cr1, e.cg1, e.cb1 = float64(c1.R), float64(c1.G), float64(c1.B)
		e.ca1 = 0
		if len(a) >= 8 {
			e.ca1 = a[7]
			if e.ca1 > 1 {
				e.ca1 /= 255
			}
		}
	} else {
		e.cr1, e.cg1, e.cb1, e.ca1 = e.cr0, e.cg0, e.cb0, 0
	}
}
