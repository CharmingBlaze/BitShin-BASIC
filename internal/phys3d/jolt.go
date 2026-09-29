//go:build !nojolt && ((linux && (amd64 || arm64)) || (darwin && arm64))

package phys3d

import (
	"unsafe"

	"github.com/bbitechnologies/jolt-go/jolt"
)

func joltHandle(b *jolt.BodyID) uintptr {
	if b == nil {
		return 0
	}
	return uintptr(*(*unsafe.Pointer)(unsafe.Pointer(b)))
}

type kinChar struct {
	x, y, z  float32
	onGround bool
	radius   float32
	height   float32
	virtual  *jolt.CharacterVirtual
}

type joltWorld struct {
	ps               *jolt.PhysicsSystem
	bi               *jolt.BodyInterface
	body             map[int]*jolt.BodyID
	bodyToEnt        map[uintptr]int
	bodyVal          map[uint32]int
	char             map[int]*kinChar
	vel              map[int][3]float32
	ang              map[int][3]float32
	mass             map[int]float32
	force            map[int][3]float32
	rad              map[int]float32
	constraints      map[int]int
	vehicles         map[int]int
	linked           map[int]bool
	planes           map[int]planeAero
	nextConstraintID int
	kick             map[int][3]float32
	soft             map[int]*softJoint
	locked           map[int]bool
	cloths           map[int]*clothSim
	rot              map[int][4]float32
	gscale           map[int]float32
	linDamp          map[int]float32
	angDamp          map[int]float32
	nocol            map[[2]int]bool
	prev             map[int][3]float32
	torque           map[int][3]float32
	com              map[int][3]float32
	motion           map[int]int
	sensor           map[int]bool
	layer            map[int]int
	layerOff         [32][32]bool
	friction         map[int]float32
	ccd              map[int]bool
	gx, gy, gz       float32
}

func New() World {
	if err := jolt.Init(); err != nil {
		return newFallback()
	}
	ps := jolt.NewPhysicsSystem()
	return &joltWorld{
		ps:               ps,
		bi:               ps.GetBodyInterface(),
		body:             map[int]*jolt.BodyID{},
		bodyToEnt:        map[uintptr]int{},
		bodyVal:          map[uint32]int{},
		char:             map[int]*kinChar{},
		vel:              map[int][3]float32{},
		ang:              map[int][3]float32{},
		mass:             map[int]float32{},
		force:            map[int][3]float32{},
		rad:              map[int]float32{},
		constraints:      map[int]int{},
		vehicles:         map[int]int{},
		linked:           map[int]bool{},
		planes:           map[int]planeAero{},
		cloths:           map[int]*clothSim{},
		kick:             map[int][3]float32{},
		soft:             map[int]*softJoint{},
		locked:           map[int]bool{},
		rot:              map[int][4]float32{},
		gscale:           map[int]float32{},
		linDamp:          map[int]float32{},
		angDamp:          map[int]float32{},
		nocol:            map[[2]int]bool{},
		prev:             map[int][3]float32{},
		torque:           map[int][3]float32{},
		com:              map[int][3]float32{},
		motion:           map[int]int{},
		sensor:           map[int]bool{},
		layer:            map[int]int{},
		friction:         map[int]float32{},
		rest:             map[int]float32{},
		ccd:              map[int]bool{},
		nextConstraintID: 1,
		gy:               -9.81,
	}
}

func (w *joltWorld) Backend() string { return BackendJolt }

func (w *joltWorld) CharacterBackend() string { return CharacterJolt }

func (w *joltWorld) Step(dt float32) {
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	g := jolt.Vec3{X: w.gx, Y: w.gy, Z: w.gz}
	for id, s := range w.gscale {
		if w.char[id] != nil || s == 1 || w.locked[id] {
			continue
		}
		m := w.mass[id]
		if m < 0.001 {
			m = 1
		}
		f := w.force[id]
		w.force[id] = [3]float32{f[0] - w.gx*(1-s)*m, f[1] - w.gy*(1-s)*m, f[2] - w.gz*(1-s)*m}
	}
	for id, f := range w.force {
		if w.char[id] != nil {
			continue
		}
		if f[0] == 0 && f[1] == 0 && f[2] == 0 {
			continue
		}
		k := w.kick[id]
		w.kick[id] = [3]float32{k[0] + f[0]*dt, k[1] + f[1]*dt, k[2] + f[2]*dt}
		w.force[id] = [3]float32{}
	}
	for id, k := range w.kick {
		m := w.mass[id]
		if m < 0.001 {
			m = 1
		}
		if b, ok := w.body[id]; ok {
			p := w.bi.GetPosition(b)
			w.bi.SetPosition(b, jolt.Vec3{X: p.X + k[0]/m*dt, Y: p.Y + k[1]/m*dt, Z: p.Z + k[2]/m*dt})
			w.bi.ActivateBody(b)
		}
		delete(w.kick, id)
	}
	w.stepSoftPose(dt)
	w.ps.Update(dt)
	w.sampleVelocities(dt)
	for id := range w.vehicles {
		if b, ok := w.body[id]; ok {
			w.bi.ActivateBody(b)
		}
	}
	for id, kc := range w.char {
		f := w.force[id]
		w.force[id] = [3]float32{}
		m := w.mass[id]
		if m <= 0 {
			m = 70
		}
		if kc.virtual != nil {
			lv := kc.virtual.GetLinearVelocity()
			if kc.virtual.GetGroundState() == jolt.GroundStateOnGround {
				gv := kc.virtual.GetGroundVelocity()
				if lv.Y-gv.Y < 0.1 {
					lv.Y = gv.Y
				}
			}
			lv.X += (g.X + f[0]/m) * dt
			lv.Y += (g.Y + f[1]/m) * dt
			lv.Z += (g.Z + f[2]/m) * dt
			kc.virtual.SetLinearVelocity(lv)
			kc.virtual.ExtendedUpdate(dt, g)
			p := kc.virtual.GetPosition()
			kc.x, kc.y, kc.z = p.X, p.Y, p.Z
			lv = kc.virtual.GetLinearVelocity()
			w.vel[id] = [3]float32{lv.X, lv.Y, lv.Z}
			kc.onGround = kc.virtual.GetGroundState() == jolt.GroundStateOnGround
			if b, ok := w.body[id]; ok {
				w.bi.SetPosition(b, p)
			}
			continue
		}
		v := w.vel[id]
		v[0] += (g.X + f[0]/m) * dt
		v[1] += (g.Y + f[1]/m) * dt
		v[2] += (g.Z + f[2]/m) * dt
		x, y, z := kc.x, kc.y, kc.z
		dx, dy, dz := v[0]*dt, v[1]*dt, v[2]*dt
		kc.onGround = false
		if hid, hx, hy, hz, hit := w.Raycast(x, y, z, dx, dy, dz); hit && hid != id {
			x, y, z = hx, hy, hz
			if dy < 0 {
				v[1] = 0
				kc.onGround = true
			}
		} else {
			x += dx
			y += dy
			z += dz
		}
		floor := kc.radius
		if y < floor {
			y = floor
			if v[1] < 0 {
				v[1] = 0
			}
			kc.onGround = true
		}
		w.vel[id] = v
		kc.x, kc.y, kc.z = x, y, z
	}
	w.stepCloths(dt)
	w.solveSoftJoints(dt)
}

func joltMotion(motion int) jolt.MotionType {
	switch motion {
	case MotionTypeKinematic:
		return jolt.MotionTypeKinematic
	case MotionTypeDynamic:
		return jolt.MotionTypeDynamic
	default:
		return jolt.MotionTypeStatic
	}
}

func (w *joltWorld) add(id int, shape *jolt.Shape, x, y, z, r float32, motion int) {
	w.addSensor(id, shape, x, y, z, r, motion, false)
}

func (w *joltWorld) addSensor(id int, shape *jolt.Shape, x, y, z, r float32, motion int, sensor bool) {
	mt := joltMotion(motion)
	b := w.bi.CreateBody(shape, jolt.Vec3{X: x, Y: y, Z: z}, mt, sensor)
	w.bi.ActivateBody(b)
	w.body[id] = b
	key := joltHandle(b)
	w.bodyToEnt[key] = id
	w.bodyVal[uint32(key)] = id
	if r < 0.1 {
		r = 0.1
	}
	w.rad[id] = r
	if w.locked == nil {
		w.locked = map[int]bool{}
	}
	w.locked[id] = motion == MotionTypeStatic
	w.rot[id] = [4]float32{0, 0, 0, 1}
	w.gscale[id] = 1
	w.motion[id] = motion
	w.sensor[id] = sensor
}

func (w *joltWorld) hitEntity(b *jolt.BodyID) int {
	if b == nil {
		return 0
	}
	if id, ok := w.bodyToEnt[joltHandle(b)]; ok {
		return id
	}
	return 0
}

func (w *joltWorld) AddBox(id int, x, y, z, hx, hy, hz float32, dynamic bool) {
	w.AddBoxEx(id, x, y, z, hx, hy, hz, motionFromDynamic(dynamic))
}

func (w *joltWorld) AddBoxEx(id int, x, y, z, hx, hy, hz float32, motion int) {
	r := hx
	if hy > r {
		r = hy
	}
	if hz > r {
		r = hz
	}
	w.add(id, jolt.CreateBox(jolt.Vec3{X: hx, Y: hy, Z: hz}), x, y, z, r, motion)
}

func (w *joltWorld) AddSphere(id int, x, y, z, r float32, dynamic bool) {
	w.AddSphereEx(id, x, y, z, r, motionFromDynamic(dynamic))
}

func (w *joltWorld) AddSphereEx(id int, x, y, z, r float32, motion int) {
	w.add(id, jolt.CreateSphere(r), x, y, z, r, motion)
}

func (w *joltWorld) AddCapsule(id int, x, y, z, halfH, r float32, dynamic bool) {
	w.add(id, jolt.CreateCapsule(halfH, r), x, y, z, r+halfH, motionFromDynamic(dynamic))
}

func (w *joltWorld) AddCylinder(id int, x, y, z, halfH, r float32, motion int) {
	w.AddConvexHull(id, cylinderHullPoints(halfH, r, 12), x, y, z, motion)
}

func (w *joltWorld) AddConvexHull(id int, points [][3]float32, x, y, z float32, motion int) {
	if len(points) < 3 {
		hx, hy, hz := hullHalfExtents(points)
		w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
		return
	}
	pts := make([]jolt.Vec3, len(points))
	for i, p := range points {
		pts[i] = jolt.Vec3{X: p[0], Y: p[1], Z: p[2]}
	}
	shape := jolt.CreateConvexHull(pts)
	if shape == nil {
		hx, hy, hz := hullHalfExtents(points)
		w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
		return
	}
	w.add(id, shape, x, y, z, hullRadius(points), motion)
}

func (w *joltWorld) AddCharacter(id int, x, y, z, halfH, r float32) {
	w.AddCharacterController(id, x, y, z, halfH*2, r, 50, 100)
}

func (w *joltWorld) CharacterGround(id int) int {
	if w.CharacterGroundState(id) == 0 {
		return 1
	}
	return 0
}

func (w *joltWorld) MoveKinematic(id int, x, y, z, qx, qy, qz, qw, dt float32) {
	w.SetPosition(id, x, y, z)
	w.SetRotation(id, qx, qy, qz, qw)
}

func (w *joltWorld) SetPosition(id int, x, y, z float32) {
	if kc := w.char[id]; kc != nil {
		kc.x, kc.y, kc.z = x, y, z
		if kc.virtual != nil {
			kc.virtual.SetPosition(jolt.Vec3{X: x, Y: y, Z: z})
		}
	}
	if b, ok := w.body[id]; ok {
		w.bi.SetPosition(b, jolt.Vec3{X: x, Y: y, Z: z})
	}
}

func (w *joltWorld) GetPosition(id int) (float32, float32, float32, bool) {
	if kc := w.char[id]; kc != nil {
		return kc.x, kc.y, kc.z, true
	}
	b, ok := w.body[id]
	if !ok {
		return 0, 0, 0, false
	}
	p := w.bi.GetPosition(b)
	return p.X, p.Y, p.Z, true
}

func (w *joltWorld) GetVelocity(id int) (float32, float32, float32, bool) {
	if kc := w.char[id]; kc != nil && kc.virtual != nil {
		lv := kc.virtual.GetLinearVelocity()
		return lv.X, lv.Y, lv.Z, true
	}
	v, ok := w.vel[id]
	return v[0], v[1], v[2], ok
}

func (w *joltWorld) SetVelocity(id int, x, y, z float32) {
	if kc := w.char[id]; kc != nil && kc.virtual != nil {
		kc.virtual.SetLinearVelocity(jolt.Vec3{X: x, Y: y, Z: z})
		lv := kc.virtual.GetLinearVelocity()
		w.vel[id] = [3]float32{lv.X, lv.Y, lv.Z}
		return
	}
	w.vel[id] = [3]float32{x, y, z}
}

func (w *joltWorld) ApplyImpulse(id int, x, y, z float32) {
	if kc := w.char[id]; kc != nil && kc.virtual != nil {
		lv := kc.virtual.GetLinearVelocity()
		kc.virtual.SetLinearVelocity(jolt.Vec3{X: lv.X + x, Y: lv.Y + y, Z: lv.Z + z})
		lv = kc.virtual.GetLinearVelocity()
		w.vel[id] = [3]float32{lv.X, lv.Y, lv.Z}
		return
	}
	if b, ok := w.body[id]; ok {
		k := w.kick[id]
		w.kick[id] = [3]float32{k[0] + x, k[1] + y, k[2] + z}
		w.bi.ActivateBody(b)
	}
}

func (w *joltWorld) Raycast(ox, oy, oz, dx, dy, dz float32) (int, float32, float32, float32, bool) {
	hit, ok := w.ps.CastRay(jolt.Vec3{X: ox, Y: oy, Z: oz}, jolt.Vec3{X: dx, Y: dy, Z: dz})
	if !ok {
		return 0, 0, 0, 0, false
	}
	id := w.hitEntity(hit.BodyID)
	if id == 0 {
		id = w.entityNear(hit.HitPoint.X, hit.HitPoint.Y, hit.HitPoint.Z)
	}
	return id, hit.HitPoint.X, hit.HitPoint.Y, hit.HitPoint.Z, true
}

func (w *joltWorld) RaycastDetail(ox, oy, oz, dx, dy, dz float32) (RayHit, bool) {
	id, x, y, z, ok := w.Raycast(ox, oy, oz, dx, dy, dz)
	if !ok {
		return RayHit{}, false
	}
	h := RayHit{ID: id, X: x, Y: y, Z: z}
	llen := sqrt32(dx*dx + dy*dy + dz*dz)
	if llen > 1e-8 {
		dist := sqrt32((x-ox)*(x-ox) + (y-oy)*(y-oy) + (z-oz)*(z-oz))
		h.Fraction = dist / llen
		if h.Fraction > 1 {
			h.Fraction = 1
		}
	}
	if px, py, pz, okp := w.GetPosition(id); okp {
		h.NX, h.NY, h.NZ, _ = unit3(x-px, y-py, z-pz)
		return h, true
	}
	if llen > 1e-8 {
		h.NX, h.NY, h.NZ = -dx/llen, -dy/llen, -dz/llen
	} else {
		h.NY = 1
	}
	return h, true
}

func (w *joltWorld) RaycastAll(ox, oy, oz, dx, dy, dz float32, max int) []RayHit {
	if max < 0 {
		return nil
	}
	h, ok := w.RaycastDetail(ox, oy, oz, dx, dy, dz)
	if !ok {
		return nil
	}
	return []RayHit{h}
}

func (w *joltWorld) entityNear(x, y, z float32) int {
	best := 0
	bestD := float32(1e12)
	for id, b := range w.body {
		p := w.bi.GetPosition(b)
		dx, dy, dz := p.X-x, p.Y-y, p.Z-z
		d2 := dx*dx + dy*dy + dz*dz
		r := w.rad[id]
		if r < 0.25 {
			r = 0.25
		}
		lim := (r + 0.5) * (r + 0.5)
		if d2 < bestD && d2 <= lim {
			bestD = d2
			best = id
		}
	}
	return best
}

func (w *joltWorld) ApplyForce(id int, x, y, z float32) {
	f := w.force[id]
	w.force[id] = [3]float32{f[0] + x, f[1] + y, f[2] + z}
	if b, ok := w.body[id]; ok {
		w.bi.ActivateBody(b)
	}
}

func (w *joltWorld) SetAngularVelocity(id int, x, y, z float32) {
	w.ang[id] = [3]float32{x, y, z}
}

func (w *joltWorld) GetAngularVelocity(id int) (float32, float32, float32, bool) {
	v, ok := w.ang[id]
	return v[0], v[1], v[2], ok
}

func (w *joltWorld) SetMass(id int, mass float32) { w.mass[id] = mass }

func (w *joltWorld) GetMass(id int) float32 { return w.mass[id] }

func (w *joltWorld) AddGround(id int, y float32) {
	w.AddBox(id, 0, y, 0, 80, 0.25, 80, false)
}

func (w *joltWorld) SetGravity(x, y, z float32) { w.gx, w.gy, w.gz = x, y, z }
func (w *joltWorld) GetGravity() (float32, float32, float32) {
	return w.gx, w.gy, w.gz
}

func (w *joltWorld) Remove(id int) {
	if kc := w.char[id]; kc != nil && kc.virtual != nil {
		kc.virtual.Destroy()
	}
	if b, ok := w.body[id]; ok {
		key := joltHandle(b)
		delete(w.bodyToEnt, key)
		delete(w.bodyVal, uint32(key))
	}
	delete(w.char, id)
	delete(w.body, id)
	delete(w.vel, id)
	delete(w.force, id)
	delete(w.rad, id)
}

func (w *joltWorld) Sleep(id int) {
	if b, ok := w.body[id]; ok {
		w.bi.DeactivateBody(b)
	}
}

func (w *joltWorld) Wake(id int) {
	if b, ok := w.body[id]; ok {
		w.bi.ActivateBody(b)
	}
}

func (w *joltWorld) stepSoftPose(dt float32) {
	for id := range w.body {
		if w.char[id] != nil || w.locked[id] {
			continue
		}
		m := w.mass[id]
		if m < 0.001 {
			m = 1
		}
		inertia := m * 0.4
		t := w.torque[id]
		c := w.com[id]
		if c[0] != 0 || c[1] != 0 || c[2] != 0 {
			q := w.rot[id]
			rx, ry, rz := quatRotateVec(q[0], q[1], q[2], q[3], c[0], c[1], c[2])
			s := w.gscale[id]
			fx, fy, fz := w.gx*s*m, w.gy*s*m, w.gz*s*m
			t[0] += ry*fz - rz*fy
			t[1] += rz*fx - rx*fz
			t[2] += rx*fy - ry*fx
		}
		a := w.ang[id]
		a[0] += t[0] / inertia * dt
		a[1] += t[1] / inertia * dt
		a[2] += t[2] / inertia * dt
		w.torque[id] = [3]float32{}
		d := w.angDamp[id]
		if d > 0 {
			s := 1 - d*dt
			if s < 0.05 {
				s = 0.05
			}
			a[0] *= s
			a[1] *= s
			a[2] *= s
		}
		w.ang[id] = a
		q := w.rot[id]
		if q[0] == 0 && q[1] == 0 && q[2] == 0 && q[3] == 0 {
			q = [4]float32{0, 0, 0, 1}
		}
		w.rot[id] = quatIntegrate(q, a, dt)
	}
}

func (w *joltWorld) sampleVelocities(dt float32) {
	if dt < 1e-6 {
		dt = 1.0 / 60
	}
	for id, b := range w.body {
		if w.char[id] != nil {
			continue
		}
		p := w.bi.GetPosition(b)
		if prev, ok := w.prev[id]; ok {
			w.vel[id] = [3]float32{(p.X - prev[0]) / dt, (p.Y - prev[1]) / dt, (p.Z - prev[2]) / dt}
			d := w.linDamp[id]
			if f := w.friction[id]; f > d {
				d = f
			}
			if d > 0 && !w.locked[id] {
				s := 1 - d*dt
				if s < 0.05 {
					s = 0.05
				}
				v := w.vel[id]
				w.bi.SetPosition(b, jolt.Vec3{
					X: prev[0] + (p.X-prev[0])*s,
					Y: prev[1] + (p.Y-prev[1])*s,
					Z: prev[2] + (p.Z-prev[2])*s,
				})
				p = w.bi.GetPosition(b)
				w.vel[id] = [3]float32{v[0] * s, v[1] * s, v[2] * s}
			}
			v := w.vel[id]
			if w.ccd[id] && !w.locked[id] {
				dx, dy, dz := p.X-prev[0], p.Y-prev[1], p.Z-prev[2]
				if dx*dx+dy*dy+dz*dz > 1e-6 {
					hid, hx, hy, hz, hit := w.Raycast(prev[0], prev[1], prev[2], dx, dy, dz)
					if hit && hid != id {
						ln := sqrt32(dx*dx + dy*dy + dz*dz)
						pad := w.rad[id] * 0.35
						if pad > ln*0.5 {
							pad = ln * 0.5
						}
						p = jolt.Vec3{X: hx - dx/ln*pad, Y: hy - dy/ln*pad, Z: hz - dz/ln*pad}
						w.bi.SetPosition(b, p)
						w.vel[id] = [3]float32{}
						v = w.vel[id]
					}
				}
			}
			if rest := w.rest[id]; rest > 0 && v[1] < -0.4 && !w.locked[id] {
				hid, _, _, _, hit := w.Raycast(p.X, p.Y, p.Z, 0, -(w.rad[id] + 0.3), 0)
				if hit && hid != id {
					m := w.mass[id]
					if m < 0.001 {
						m = 1
					}
					w.ApplyImpulse(id, 0, m*(-v[1]*rest-v[1]), 0)
				}
			}
		}
		w.prev[id] = [3]float32{p.X, p.Y, p.Z}
	}
}

func (w *joltWorld) SetJobThreads(int) {}

func (w *joltWorld) Close() {
	for _, kc := range w.char {
		if kc.virtual != nil {
			kc.virtual.Destroy()
		}
	}
	for _, b := range w.body {
		b.Destroy()
	}
	w.ps.Destroy()
	jolt.Shutdown()
}
