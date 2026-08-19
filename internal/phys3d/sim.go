package phys3d

// Software rigid bodies used when native Jolt is not linked:
// go build -tags nojolt, or an OS/arch with no prebuilt (not Windows,
// not Linux amd64/arm64, not macOS ARM). PhysicsBackend$() returns "fallback".

type body struct {
	x, y, z     float32
	vx, vy, vz  float32
	ax, ay, az  float32
	fx, fy, fz  float32
	r           float32
	hx, hy, hz  float32
	box         bool
	mass        float32
	dynamic     bool
	kinematic   bool
	asleep      bool
	character   bool
	onGround    bool
	gScale      float32
	linDamp     float32
	angDamp     float32
	restitution float32
	friction    float32
	layer       int
}

type softJoint struct {
	a, b              int
	px, py, pz        float32
	lax, lay, laz     float32
	lbx, lby, lbz     float32
	ax, ay, az        float32
	rest, stiff, damp float32
	kind              int
}

type fallback struct {
	bodies     map[int]*body
	joints     map[int]*softJoint
	planes     map[int]planeAero
	posed      map[int]bool
	cloths     map[int]*clothSim
	nextJoint  int
	gx, gy, gz float32
	layerOff   [32][32]bool
}

func newFallback() World {
	return &fallback{bodies: map[int]*body{}, joints: map[int]*softJoint{}, planes: map[int]planeAero{}, posed: map[int]bool{}, cloths: map[int]*clothSim{}, nextJoint: 1, gy: -9.81}
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
		if w.cloths[id] != nil {
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
		if w.cloths[ids[i]] != nil {
			continue
		}
		for j := i + 1; j < len(ids); j++ {
			if w.cloths[ids[j]] != nil {
				continue
			}
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
	w.stepCloths(dt)
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
		x: x, y: y, z: z, r: r, hx: hx, hy: hy, hz: hz, box: true, mass: 1, gScale: 1,
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

func (w *fallback) AddCylinder(id int, x, y, z, halfH, r float32, motion int) {
	w.AddSphereEx(id, x, y, z, r+halfH, motion)
}

func (w *fallback) AddConvexHull(id int, points [][3]float32, x, y, z float32, motion int) {
	hx, hy, hz := hullHalfExtents(points)
	rad := hx
	if hy > rad {
		rad = hy
	}
	if hz > rad {
		rad = hz
	}
	w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
	_ = rad
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
		var t float32
		hit := false
		if b.box {
			t, hit = rayAABB(ox, oy, oz, dx, dy, dz, b.x-b.hx, b.y-b.hy, b.z-b.hz, b.x+b.hx, b.y+b.hy, b.z+b.hz)
		} else {
			t, hit = raySphere(ox, oy, oz, dx, dy, dz, b.x, b.y, b.z, b.r)
		}
		if !hit || t >= best {
			continue
		}
		best, hitID = t, id
		hx, hy, hz = ox+dx*t, oy+dy*t, oz+dz*t
		ok = true
	}
	return hitID, hx, hy, hz, ok
}

func rayAABB(ox, oy, oz, dx, dy, dz, minX, minY, minZ, maxX, maxY, maxZ float32) (float32, bool) {
	tmin := float32(0)
	tmax := float32(1)
	slab := func(o, d, mn, mx float32) bool {
		if d*d < 1e-20 {
			return o >= mn && o <= mx
		}
		inv := 1 / d
		t0 := (mn - o) * inv
		t1 := (mx - o) * inv
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		if t0 > tmin {
			tmin = t0
		}
		if t1 < tmax {
			tmax = t1
		}
		return tmin <= tmax
	}
	if !slab(ox, dx, minX, maxX) || !slab(oy, dy, minY, maxY) || !slab(oz, dz, minZ, maxZ) {
		return 0, false
	}
	if tmin < 0 {
		if tmax < 0 || tmax > 1 {
			return 0, false
		}
		return tmax, true
	}
	if tmin > 1 {
		return 0, false
	}
	return tmin, true
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
func (w *fallback) Remove(id int) {
	delete(w.bodies, id)
	delete(w.cloths, id)
}
func (w *fallback) Close() {}

func (w *fallback) SetJobThreads(int) {}
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

func (w *fallback) ApplyBuoyancyImpulse(id int, sx, sy, sz, nx, ny, nz, buoyancy, linDrag, angDrag, fvx, fvy, fvz, dt float32) bool {
	b := w.bodies[id]
	if b == nil || !b.dynamic {
		return false
	}
	depth := sy - b.y
	if depth <= 0 {
		return false
	}
	if buoyancy < 0.01 {
		buoyancy = 1
	}
	if dt <= 0 {
		dt = 1.0 / 60
	}
	g := w.gy
	if g > 0 {
		g = -g
	}
	w.ApplyForce(id, nx*buoyancy*b.mass*2, -g*buoyancy*b.mass*depth, nz*buoyancy*b.mass*2)
	b.vx *= 1 - linDrag*dt
	b.vy *= 1 - linDrag*dt
	b.vz *= 1 - linDrag*dt
	b.ax *= 1 - angDrag*dt
	b.ay *= 1 - angDrag*dt
	b.az *= 1 - angDrag*dt
	_ = sx
	_ = sz
	_ = fvx
	_ = fvy
	_ = fvz
	return true
}

func (w *fallback) OffsetCenterOfMass(int, float32, float32, float32) {}

func (w *fallback) AddMesh(id int, verts [][3]float32, indices []int32, motion int) {
	if len(verts) == 0 {
		return
	}
	minX, minY, minZ := verts[0][0], verts[0][1], verts[0][2]
	maxX, maxY, maxZ := minX, minY, minZ
	cx, cy, cz := float32(0), float32(0), float32(0)
	for _, v := range verts {
		cx += v[0]
		cy += v[1]
		cz += v[2]
		if v[0] < minX {
			minX = v[0]
		}
		if v[1] < minY {
			minY = v[1]
		}
		if v[2] < minZ {
			minZ = v[2]
		}
		if v[0] > maxX {
			maxX = v[0]
		}
		if v[1] > maxY {
			maxY = v[1]
		}
		if v[2] > maxZ {
			maxZ = v[2]
		}
	}
	n := float32(len(verts))
	w.AddBoxEx(id, cx/n, cy/n, cz/n, (maxX-minX)*0.5, (maxY-minY)*0.5, (maxZ-minZ)*0.5, motion)
	_ = indices
}

func (w *fallback) AddHeightField(id int, samples []float32, n int, ox, oy, oz, sx, sy, sz float32) {
	if n < 2 || len(samples) < n*n {
		w.AddGround(id, oy)
		return
	}
	minH, maxH := samples[0], samples[0]
	for _, h := range samples[:n*n] {
		if h < minH {
			minH = h
		}
		if h > maxH {
			maxH = h
		}
	}
	hy := (maxH - minH) * sy * 0.5
	if hy < 0.25 {
		hy = 0.25
	}
	cx := ox + sx*float32(n-1)*0.5
	cy := oy + (minH+maxH)*sy*0.5
	cz := oz + sz*float32(n-1)*0.5
	w.AddBoxEx(id, cx, cy, cz, sx*float32(n-1)*0.5, hy, sz*float32(n-1)*0.5, MotionTypeStatic)
}

func (w *fallback) AddSensorBox(id int, x, y, z, hx, hy, hz float32, motion int) {
	w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
}

func (w *fallback) SetSensor(int, bool) {}

func (w *fallback) ShapeCast(hx, hy, hz, x, y, z, dx, dy, dz float32) (int, float32, float32, float32, bool) {
	if hx < 0.02 {
		hx = 0.02
	}
	if hy < 0.02 {
		hy = 0.02
	}
	if hz < 0.02 {
		hz = 0.02
	}
	steps := 16
	for i := 1; i <= steps; i++ {
		t := float32(i) / float32(steps)
		px, py, pz := x+dx*t, y+dy*t, z+dz*t
		for id, b := range w.bodies {
			if bodyHitsBox(b, px, py, pz, hx, hy, hz) {
				return id, px, py, pz, true
			}
		}
	}
	return w.Raycast(x, y, z, dx, dy, dz)
}

func bodyHitsSphere(b *body, x, y, z, r float32) bool {
	if b == nil {
		return false
	}
	if b.box {
		cx := clamp32(x, b.x-b.hx, b.x+b.hx)
		cy := clamp32(y, b.y-b.hy, b.y+b.hy)
		cz := clamp32(z, b.z-b.hz, b.z+b.hz)
		dx, dy, dz := x-cx, y-cy, z-cz
		return dx*dx+dy*dy+dz*dz <= r*r
	}
	dx, dy, dz := b.x-x, b.y-y, b.z-z
	lim := r + b.r
	return dx*dx+dy*dy+dz*dz <= lim*lim
}

func bodyHitsBox(b *body, x, y, z, hx, hy, hz float32) bool {
	if b == nil {
		return false
	}
	if b.box {
		return abs32(b.x-x) <= b.hx+hx && abs32(b.y-y) <= b.hy+hy && abs32(b.z-z) <= b.hz+hz
	}
	cx := clamp32(b.x, x-hx, x+hx)
	cy := clamp32(b.y, y-hy, y+hy)
	cz := clamp32(b.z, z-hz, z+hz)
	dx, dy, dz := b.x-cx, b.y-cy, b.z-cz
	r := b.r
	return dx*dx+dy*dy+dz*dz <= r*r
}

func clamp32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (w *fallback) OverlapSphere(x, y, z, r float32) (int, bool) {
	ids := w.OverlapSphereAll(x, y, z, r, 1)
	if len(ids) == 0 {
		return 0, false
	}
	return ids[0], true
}

func (w *fallback) OverlapSphereAll(x, y, z, r float32, max int) []int {
	if max <= 0 {
		max = 32
	}
	out := []int{}
	for id, b := range w.bodies {
		if bodyHitsSphere(b, x, y, z, r) {
			out = append(out, id)
			if len(out) >= max {
				break
			}
		}
	}
	return out
}

func (w *fallback) OverlapPoint(x, y, z float32) (int, bool) {
	return w.OverlapSphere(x, y, z, 0.05)
}

func (w *fallback) OptimizeBroadPhase() {}

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

func (w *fallback) CreateGrabJoint(a, b int, px, py, pz, freq, damp float32) int {
	if freq <= 0 {
		freq = 8
	}
	return w.addSoftJoint(a, b, px, py, pz, 0, 1, 0, 0.05, freq*20, damp, 3)
}

func (w *fallback) CreateFixedJoint(a, b int, px, py, pz float32) int {
	return w.addSoftJoint(a, b, px, py, pz, 0, 1, 0, 0, 0, 0, 1)
}

func (w *fallback) CreateConeJoint(a, b int, px, py, pz, ax, ay, az, halfConeDeg float32) int {
	_ = halfConeDeg
	return w.addSoftJoint(a, b, px, py, pz, ax, ay, az, 0, 0, 0, 1)
}

func (w *fallback) CreateSwingTwistJoint(a, b int, px, py, pz, ax, ay, az, swingDeg, twistDeg float32) int {
	_ = swingDeg
	_ = twistDeg
	return w.addSoftJoint(a, b, px, py, pz, ax, ay, az, 0, 0, 0, 1)
}

func (w *fallback) AddCompound(id int, parts []CompoundPart, x, y, z float32, motion int) {
	hx, hy, hz := CompoundAABB(parts)
	w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
}

func (w *fallback) SetCollisionLayer(id, layer int) {
	b := w.bodies[id]
	if b == nil {
		return
	}
	if layer < 0 {
		layer = 0
	}
	if layer > 31 {
		layer = 31
	}
	b.layer = layer
}

func (w *fallback) SetLayerCollides(a, b int, on bool) {
	if a < 0 {
		a = 0
	}
	if b < 0 {
		b = 0
	}
	if a > 31 {
		a = 31
	}
	if b > 31 {
		b = 31
	}
	off := !on
	w.layerOff[a][b] = off
	w.layerOff[b][a] = off
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
