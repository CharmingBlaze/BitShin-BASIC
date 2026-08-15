package phys3d

// Software rigid bodies used when native Jolt is not linked:
// go build -tags nojolt, or an OS/arch with no prebuilt (not Windows,
// not Linux amd64/arm64, not macOS ARM). PhysicsBackend$() returns "fallback".

type body struct {
	x, y, z    float32
	vx, vy, vz float32
	ax, ay, az float32
	fx, fy, fz float32
	r          float32
	mass       float32
	dynamic    bool
	kinematic  bool
	asleep     bool
	character  bool
	onGround   bool
	gScale     float32
	linDamp     float32
	angDamp     float32
	restitution float32
	friction    float32
}

type softJoint struct {
	a, b                 int
	px, py, pz           float32
	lax, lay, laz        float32
	lbx, lby, lbz        float32
	ax, ay, az           float32
	rest, stiff, damp    float32
	kind                 int
}

type fallback struct {
	bodies     map[int]*body
	joints     map[int]*softJoint
	planes     map[int]planeAero
	posed      map[int]bool
	nextJoint  int
	gx, gy, gz float32
}

func newFallback() World {
	return &fallback{bodies: map[int]*body{}, joints: map[int]*softJoint{}, planes: map[int]planeAero{}, posed: map[int]bool{}, nextJoint: 1, gy: -9.81}
}

func (w *fallback) Backend() string { return BackendFallback }

func (w *fallback) CharacterBackend() string { return CharacterKinematic }

func (w *fallback) Step(dt float32) {
	if dt <= 0 || dt > 0.1 {
		dt = 1.0 / 60.0
	}
	for id, b := range w.bodies {
		if b.asleep {
			continue
		}
		if b.character {
			w.stepCharacter(b, dt)
			continue
		}
		if b.kinematic {
			if w.posed[id] {
				continue
			}
			b.x += b.vx * dt
			b.y += b.vy * dt
			b.z += b.vz * dt
			continue
		}
		if !b.dynamic {
			continue
		}
		m := b.mass
		if m <= 0 {
			m = 1
		}
		gs := b.gScale
		b.vx += (w.gx*gs + b.fx/m) * dt
		b.vy += (w.gy*gs + b.fy/m) * dt
		b.vz += (w.gz*gs + b.fz/m) * dt
		if b.linDamp > 0 {
			damp := 1 - b.linDamp*dt
			if damp < 0.1 {
				damp = 0.1
			}
			b.vx *= damp
			b.vy *= damp
			b.vz *= damp
		}
		if b.angDamp > 0 {
			ad := 1 - b.angDamp*dt
			if ad < 0.1 {
				ad = 0.1
			}
			b.ax *= ad
			b.ay *= ad
			b.az *= ad
		}
		b.fx, b.fy, b.fz = 0, 0, 0
		b.x += b.vx * dt
		b.y += b.vy * dt
		b.z += b.vz * dt
		if b.y < b.r {
			b.y = b.r
			if b.vy < 0 {
				b.vy = -b.vy * 0.35
			}
		}
	}
	ids := make([]int, 0, len(w.bodies))
	for id := range w.bodies {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a, b := w.bodies[ids[i]], w.bodies[ids[j]]
			dx, dy, dz := a.x-b.x, a.y-b.y, a.z-b.z
			d2 := dx*dx + dy*dy + dz*dz
			min := a.r + b.r
			if d2 > 0 && d2 < min*min {
				d := sqrt32(d2)
				nx, ny, nz := dx/d, dy/d, dz/d
				pen := min - d
				if a.dynamic && !a.kinematic {
					a.x += nx * pen * 0.5
					a.y += ny * pen * 0.5
					a.z += nz * pen * 0.5
				}
				if b.dynamic && !b.kinematic {
					b.x -= nx * pen * 0.5
					b.y -= ny * pen * 0.5
					b.z -= nz * pen * 0.5
				}
				if a.character {
					a.x += nx * pen
					a.y += ny * pen
					a.z += nz * pen
				}
				if b.character {
					b.x -= nx * pen
					b.y -= ny * pen
					b.z -= nz * pen
				}
			}
		}
	}
	w.solveJoints()
	clear(w.posed)
}

func (w *fallback) stepCharacter(b *body, dt float32) {
	m := b.mass
	if m <= 0 {
		m = 70
	}
	b.vx += (w.gx + b.fx/m) * dt
	b.vy += (w.gy + b.fy/m) * dt
	b.vz += (w.gz + b.fz/m) * dt
	b.fx, b.fy, b.fz = 0, 0, 0
	b.x += b.vx * dt
	b.y += b.vy * dt
	b.z += b.vz * dt
	b.onGround = false
	if b.y < b.r {
		b.y = b.r
		if b.vy < 0 {
			b.vy = 0
		}
		b.onGround = true
	}
}

func (w *fallback) AddBox(id int, x, y, z, hx, hy, hz float32, dynamic bool) {
	w.AddBoxEx(id, x, y, z, hx, hy, hz, motionFromDynamic(dynamic))
}

func (w *fallback) AddBoxEx(id int, x, y, z, hx, hy, hz float32, motion int) {
	r := hx
	if hy > r {
		r = hy
	}
	if hz > r {
		r = hz
	}
	w.bodies[id] = &body{
		x: x, y: y, z: z, r: r, mass: 1, gScale: 1,
		dynamic:   motion == MotionTypeDynamic,
		kinematic: motion == MotionTypeKinematic,
	}
}

func (w *fallback) AddSphere(id int, x, y, z, r float32, dynamic bool) {
	w.AddSphereEx(id, x, y, z, r, motionFromDynamic(dynamic))
}

func (w *fallback) AddSphereEx(id int, x, y, z, r float32, motion int) {
	w.bodies[id] = &body{
		x: x, y: y, z: z, r: r, mass: 1, gScale: 1,
		dynamic:   motion == MotionTypeDynamic,
		kinematic: motion == MotionTypeKinematic,
	}
}

func (w *fallback) AddCapsule(id int, x, y, z, halfH, r float32, dynamic bool) {
	if r < halfH {
		r = halfH
	}
	w.bodies[id] = &body{x: x, y: y, z: z, r: r + halfH, mass: 1, dynamic: dynamic, gScale: 1}
}

func (w *fallback) AddCharacter(id int, x, y, z, halfH, r float32) {
	w.AddCharacterController(id, x, y, z, halfH*2, r, 50, 100)
}

func (w *fallback) AddCharacterController(id int, x, y, z, height, radius, maxSlopeDeg, maxStrength float32) {
	_ = maxSlopeDeg
	_ = maxStrength
	halfH := height * 0.5
	rad := radius + halfH
	if rad < 0.4 {
		rad = 0.4
	}
	w.bodies[id] = &body{
		x: x, y: y, z: z, r: rad, mass: 70, gScale: 1,
		dynamic: true, kinematic: true, character: true,
	}
}

func (w *fallback) SetCharacterShape(id int, shapeType string, height, radius float32) {
	b := w.bodies[id]
	if b == nil || !b.character {
		return
	}
	halfH := height * 0.5
	rad := radius + halfH
	if shapeType == "box" || shapeType == "Box" || shapeType == "BOX" {
		rad = height*0.5 + radius
	}
	if rad < 0.2 {
		rad = 0.2
	}
	b.r = rad
}

func (w *fallback) CharacterGround(id int) int {
	if b := w.bodies[id]; b != nil && b.onGround {
		return 1
	}
	return 0
}

func (w *fallback) CharacterGroundState(id int) int {
	if b := w.bodies[id]; b != nil && b.onGround {
		return 0
	}
	return 3
}

func (w *fallback) CharacterGroundNormal(id int) (float32, float32, float32) {
	if b := w.bodies[id]; b != nil && b.onGround {
		return 0, 1, 0
	}
	return 0, 0, 0
}

func (w *fallback) CharacterContact(id int) int {
	self := w.bodies[id]
	if self == nil {
		return 0
	}
	best := 0
	bestD := self.r * self.r * 4
	for otherID, b := range w.bodies {
		if otherID == id {
			continue
		}
		dx, dy, dz := self.x-b.x, self.y-b.y, self.z-b.z
		d2 := dx*dx + dy*dy + dz*dz
		min := self.r + b.r
		if d2 < min*min && d2 < bestD {
			bestD = d2
			best = otherID
		}
	}
	return best
}

func (w *fallback) AddGround(id int, y float32) {
	w.bodies[id] = &body{x: 0, y: y, z: 0, r: 0.2, mass: 0, dynamic: false}
}

func (w *fallback) SetPosition(id int, x, y, z float32) {
	if b := w.bodies[id]; b != nil {
		b.x, b.y, b.z = x, y, z
	}
}

func (w *fallback) MoveKinematic(id int, x, y, z, qx, qy, qz, qw, dt float32) {
	b := w.bodies[id]
	if b == nil {
		return
	}
	if dt <= 1e-6 {
		dt = 1e-6
	}
	b.vx = (x - b.x) / dt
	b.vy = (y - b.y) / dt
	b.vz = (z - b.z) / dt
	b.x, b.y, b.z = x, y, z
	w.posed[id] = true
}

func (w *fallback) GetPosition(id int) (float32, float32, float32, bool) {
	b := w.bodies[id]
	if b == nil {
		return 0, 0, 0, false
	}
	return b.x, b.y, b.z, true
}

func (w *fallback) SetVelocity(id int, x, y, z float32) {
	if b := w.bodies[id]; b != nil {
		b.vx, b.vy, b.vz = x, y, z
	}
}

func (w *fallback) GetVelocity(id int) (float32, float32, float32, bool) {
	b := w.bodies[id]
	if b == nil {
		return 0, 0, 0, false
	}
	return b.vx, b.vy, b.vz, true
}

func (w *fallback) ApplyImpulse(id int, x, y, z float32) {
	if b := w.bodies[id]; b != nil && (b.dynamic || b.character) {
		b.vx += x
		b.vy += y
		b.vz += z
		b.asleep = false
	}
}

func (w *fallback) ApplyForce(id int, x, y, z float32) {
	if b := w.bodies[id]; b != nil && (b.dynamic || b.character) {
		b.fx += x
		b.fy += y
		b.fz += z
		b.asleep = false
	}
}

func (w *fallback) SetAngularVelocity(id int, x, y, z float32) {
	if b := w.bodies[id]; b != nil {
		b.ax, b.ay, b.az = x, y, z
	}
}

func (w *fallback) GetAngularVelocity(id int) (float32, float32, float32, bool) {
	b := w.bodies[id]
	if b == nil {
		return 0, 0, 0, false
	}
	return b.ax, b.ay, b.az, true
}

func (w *fallback) SetMass(id int, mass float32) {
	if b := w.bodies[id]; b != nil {
		if mass < 0.001 {
			mass = 0.001
		}
		b.mass = mass
	}
}

func (w *fallback) Raycast(ox, oy, oz, dx, dy, dz float32) (int, float32, float32, float32, bool) {
	best := float32(2)
	hitID := 0
	var hx, hy, hz float32
	ok := false
	for id, b := range w.bodies {
		t, hit := raySphere(ox, oy, oz, dx, dy, dz, b.x, b.y, b.z, b.r)
		if !hit || t >= best {
			continue
		}
		best, hitID = t, id
		hx, hy, hz = ox+dx*t, oy+dy*t, oz+dz*t
		ok = true
	}
	return hitID, hx, hy, hz, ok
}

func raySphere(ox, oy, oz, dx, dy, dz, cx, cy, cz, r float32) (float32, bool) {
	ocx, ocy, ocz := ox-cx, oy-cy, oz-cz
	a := dx*dx + dy*dy + dz*dz
	if a < 1e-12 {
		return 0, false
	}
	b := 2 * (ocx*dx + ocy*dy + ocz*dz)
	c := ocx*ocx + ocy*ocy + ocz*ocz - r*r
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, false
	}
	s := sqrt32(disc)
	t0 := (-b - s) / (2 * a)
	t1 := (-b + s) / (2 * a)
	t := t0
	if t < 0 {
		t = t1
	}
	if t < 0 || t > 1 {
		return 0, false
	}
	return t, true
}

func (w *fallback) SetGravity(x, y, z float32) { w.gx, w.gy, w.gz = x, y, z }
func (w *fallback) GetGravity() (float32, float32, float32) {
	return w.gx, w.gy, w.gz
}
func (w *fallback) Remove(id int) { delete(w.bodies, id) }
func (w *fallback) Close()        {}
func (w *fallback) Sleep(id int) {
	if b := w.bodies[id]; b != nil {
		b.asleep = true
		b.vx, b.vy, b.vz = 0, 0, 0
	}
}
func (w *fallback) Wake(id int) {
	if b := w.bodies[id]; b != nil {
		b.asleep = false
	}
}

func (w *fallback) GetRotation(id int) (float32, float32, float32, float32, bool) {
	if w.bodies[id] == nil {
		return 0, 0, 0, 1, false
	}
	return 0, 0, 0, 1, true
}

func (w *fallback) SetRotation(int, float32, float32, float32, float32) {}

func (w *fallback) SetCCD(int, bool) int { return 0 }

func (w *fallback) RemoveJoint(id int) { delete(w.joints, id) }

func (w *fallback) PollContacts(int) []ContactEvent { return nil }

func (w *fallback) EnableContacts() {}

func (w *fallback) LookupBody(uint32) int { return 0 }

func (w *fallback) ApplyTorque(id int, x, y, z float32) {
	if b := w.bodies[id]; b != nil && (b.dynamic || b.character) {
		b.ax += x * 0.02
		b.ay += y * 0.02
		b.az += z * 0.02
		b.asleep = false
	}
}

func (w *fallback) ApplyForceAtPosition(id int, fx, fy, fz, px, py, pz float32) {
	b := w.bodies[id]
	if b == nil || !(b.dynamic || b.character) {
		return
	}
	b.fx += fx
	b.fy += fy
	b.fz += fz
	rx, ry, rz := px-b.x, py-b.y, pz-b.z
	b.ax += ry*fz - rz*fy
	b.ay += rz*fx - rx*fz
	b.az += rx*fy - ry*fx
	b.asleep = false
}

func (w *fallback) ApplyLocalImpulse(id int, lx, ly, lz float32) {
	w.ApplyImpulse(id, lx, ly, lz)
}

func (w *fallback) SetGravityScale(id int, scale float32) {
	if b := w.bodies[id]; b != nil {
		b.gScale = scale
	}
}

func (w *fallback) SetRestitution(id int, r float32) {
	if b := w.bodies[id]; b != nil {
		b.restitution = r
	}
}

func (w *fallback) SetLinearDamping(id int, d float32) {
	if b := w.bodies[id]; b != nil {
		b.linDamp = d
	}
}

func (w *fallback) SetAngularDamping(id int, d float32) {
	if b := w.bodies[id]; b != nil {
		b.angDamp = d
	}
}

func (w *fallback) SetHingeLimits(int, float32, float32) {}

func (w *fallback) SetHingeFriction(int, float32) {}

func (w *fallback) SetHingeMotor(int, float32, float32) {}

func (w *fallback) DisableBodyCollision(int, int) {}

func (w *fallback) SetFriction(id int, f float32) {
	if b := w.bodies[id]; b != nil {
		b.friction = f
	}
}

func (w *fallback) CreateWheeledVehicle(int, float32, float32, float32) int { return 0 }

func (w *fallback) CreateMotorcycleVehicle(int, float32, float32, float32) int { return 0 }

func (w *fallback) CreateTrackedVehicle(int, float32, float32, float32) int { return 0 }

func (w *fallback) SetVehicleInput(int, float32, float32, float32) {}

func (w *fallback) CreatePlaneController(id int) int {
	if w.planes == nil {
		w.planes = map[int]planeAero{}
	}
	w.planes[id] = planeAero{thrust: 18000, cl: 0.9, cd: 0.08, stall: 12, rho: 1.2, area: 24, mass: 900}
	return id
}

func (w *fallback) UpdatePlane(id int, throttle, pitch, roll, yaw float32) {
	a, ok := w.planes[id]
	if !ok {
		return
	}
	w.ApplyForce(id, 0, a.cl*throttle*a.thrust*0.02, throttle*a.thrust)
	w.ApplyTorque(id, pitch*4000, yaw*4000, roll*4000)
}

func (w *fallback) addSoftJoint(a, b int, px, py, pz, ax, ay, az, rest, stiff, damp float32, kind int) int {
	if w.joints == nil {
		w.joints = map[int]*softJoint{}
	}
	if w.nextJoint < 1 {
		w.nextJoint = 1
	}
	if a != 0 && w.bodies[a] == nil {
		return 0
	}
	if b != 0 && w.bodies[b] == nil {
		return 0
	}
	ba := w.bodies[a]
	bb := w.bodies[b]
	j := &softJoint{a: a, b: b, px: px, py: py, pz: pz, ax: ax, ay: ay, az: az, rest: rest, stiff: stiff, damp: damp, kind: kind}
	if ba != nil {
		j.lax, j.lay, j.laz = px-ba.x, py-ba.y, pz-ba.z
	}
	if bb != nil {
		j.lbx, j.lby, j.lbz = px-bb.x, py-bb.y, pz-bb.z
	}
	id := w.nextJoint
	w.nextJoint++
	w.joints[id] = j
	return id
}

func (w *fallback) CreateHingeJoint(a, b int, px, py, pz, ax, ay, az float32) int {
	return w.addSoftJoint(a, b, px, py, pz, ax, ay, az, 0, 0, 0, 1)
}

func (w *fallback) CreatePointJoint(a, b int, px, py, pz float32) int {
	return w.addSoftJoint(a, b, px, py, pz, 0, 1, 0, 0, 0, 0, 1)
}

func (w *fallback) CreateSliderJoint(a, b int, px, py, pz, ax, ay, az float32) int {
	return w.addSoftJoint(a, b, px, py, pz, ax, ay, az, 0, 0, 0, 2)
}

func (w *fallback) CreateSpringJoint(a, b int, px, py, pz, rest, stiff, damp float32) int {
	return w.addSoftJoint(a, b, px, py, pz, 0, 1, 0, rest, stiff, damp, 3)
}

func (w *fallback) solveJoints() {
	for _, j := range w.joints {
		ba := w.bodies[j.a]
		bb := w.bodies[j.b]
		wx, wy, wz := j.px, j.py, j.pz
		if ba != nil {
			wx, wy, wz = ba.x+j.lax, ba.y+j.lay, ba.z+j.laz
		}
		if bb == nil {
			continue
		}
		cx, cy, cz := bb.x+j.lbx, bb.y+j.lby, bb.z+j.lbz
		dx, dy, dz := wx-cx, wy-cy, wz-cz
		if j.kind == 3 && j.stiff > 0 {
			bb.vx += dx * j.stiff * 0.02
			bb.vy += dy * j.stiff * 0.02
			bb.vz += dz * j.stiff * 0.02
			if j.damp > 0 {
				bb.vx *= 1 - j.damp*0.01
				bb.vy *= 1 - j.damp*0.01
				bb.vz *= 1 - j.damp*0.01
			}
			continue
		}
		if !bb.dynamic {
			continue
		}
		bb.x += dx
		bb.y += dy
		bb.z += dz
	}
}

func sqrt32(v float32) float32 {
	if v <= 0 {
		return 0
	}
	x := v
	for i := 0; i < 6; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}
