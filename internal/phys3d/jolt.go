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
	w.ps.Update(dt)
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
			if f[0] != 0 || f[1] != 0 || f[2] != 0 {
				lv := kc.virtual.GetLinearVelocity()
				kc.virtual.SetLinearVelocity(jolt.Vec3{
					X: lv.X + f[0]/m*dt,
					Y: lv.Y + f[1]/m*dt,
					Z: lv.Z + f[2]/m*dt,
				})
			}
			kc.virtual.ExtendedUpdate(dt, g)
			p := kc.virtual.GetPosition()
			kc.x, kc.y, kc.z = p.X, p.Y, p.Z
			lv := kc.virtual.GetLinearVelocity()
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
	w.solveSoftJoints()
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
	w.add(id, jolt.CreateCapsule(halfH, r), x, y, z, r+halfH, motion)
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
