//go:build !nojolt && windows

package phys3d

import "bitshinbasic/internal/jolt"

type kinChar struct {
	x, y, z  float32
	onGround bool
	radius   float32
	height   float32
	virtual  *jolt.CharacterVirtual
}

type joltWorld struct {
	ps               *jolt.PhysicsSystem
	tempAllocator    *jolt.TempAllocator
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
	constraints      map[int]*jolt.Constraint
	vehicles         map[int]*jolt.VehicleConstraint
	linked           map[int]bool
	planes           map[int]planeAero
	motion           map[int]int
	kinPose          map[int]bool
	pendingAdds      int
	nextConstraintID int
	gx, gy, gz       float32
}

func New() World {
	if err := jolt.Init(); err != nil {
		return newFallback()
	}
	ps := jolt.NewPhysicsSystem()
	ps.EnableContactListener(true)
	ps.SetGravity(jolt.Vec3{X: 0, Y: -9.81, Z: 0})
	return &joltWorld{
		ps:            ps,
		tempAllocator: jolt.NewTempAllocatorMalloc(),
		bi:            ps.GetBodyInterface(),
		body:      map[int]*jolt.BodyID{},
		bodyToEnt: map[uintptr]int{},
		bodyVal:   map[uint32]int{},
		char:      map[int]*kinChar{},
		vel:   map[int][3]float32{},
		ang:   map[int][3]float32{},
		mass:  map[int]float32{},
		force:            map[int][3]float32{},
		rad:              map[int]float32{},
		constraints:      map[int]*jolt.Constraint{},
		vehicles:         map[int]*jolt.VehicleConstraint{},
		linked:           map[int]bool{},
		planes:           map[int]planeAero{},
		motion:           map[int]int{},
		kinPose:          map[int]bool{},
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
	w.ps.SetGravity(g)
	for id, b := range w.body {
		if w.motion[id] != MotionTypeKinematic || w.kinPose[id] {
			continue
		}
		v := w.vel[id]
		p := w.bi.GetPosition(b)
		q := w.bi.GetRotation(b)
		w.bi.MoveKinematic(b, jolt.Vec3{X: p.X + v[0]*dt, Y: p.Y + v[1]*dt, Z: p.Z + v[2]*dt}, q, dt)
		w.bi.ActivateBody(b)
	}
	for id := range w.linked {
		if b, ok := w.body[id]; ok {
			w.bi.ActivateBody(b)
		}
	}
	if w.pendingAdds >= 8 {
		w.OptimizeBroadPhase()
	}
	w.ps.Update(dt, w.tempAllocator)
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
	for id, b := range w.body {
		if w.char[id] != nil {
			continue
		}
		lv := w.bi.GetLinearVelocity(b)
		w.vel[id] = [3]float32{lv.X, lv.Y, lv.Z}
		av := w.bi.GetAngularVelocity(b)
		w.ang[id] = [3]float32{av.X, av.Y, av.Z}
	}
	clear(w.kinPose)
}

func (w *joltWorld) MoveKinematic(id int, x, y, z, qx, qy, qz, qw, dt float32) {
	b, ok := w.body[id]
	if !ok {
		return
	}
	if dt <= 1e-6 {
		dt = 1e-6
	}
	w.bi.MoveKinematic(b, jolt.Vec3{X: x, Y: y, Z: z}, jolt.Quat{X: qx, Y: qy, Z: qz, W: qw}, dt)
	w.bi.ActivateBody(b)
	w.kinPose[id] = true
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
	mt := joltMotion(motion)
	b := w.bi.CreateBody(shape, jolt.Vec3{X: x, Y: y, Z: z}, mt, false)
	w.bi.ActivateBody(b)
	w.body[id] = b
	w.bodyToEnt[b.Key()] = id
	w.bodyVal[b.Value()] = id
	w.motion[id] = motion
	if r < 0.1 {
		r = 0.1
	}
	w.rad[id] = r
	w.pendingAdds++
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
	shape := jolt.CreateCylinder(halfH, r)
	if shape == nil {
		w.AddCapsule(id, x, y, z, halfH, r, motion == MotionTypeDynamic)
		return
	}
	rad := r
	if halfH > rad {
		rad = halfH
	}
	w.add(id, shape, x, y, z, rad, motion)
}

func (w *joltWorld) AddConvexHull(id int, points [][3]float32, x, y, z float32, motion int) {
	if len(points) < 3 {
		return
	}
	pts := make([]jolt.Vec3, len(points))
	for i, p := range points {
		pts[i] = jolt.NewVec3(p[0], p[1], p[2])
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

func (w *joltWorld) AddCharacterController(id int, x, y, z, height, radius, maxSlopeDeg, maxStrength float32) {
	if height <= 0 {
		height = 1.8
	}
	if radius <= 0 {
		radius = 0.4
	}
	if maxSlopeDeg <= 0 {
		maxSlopeDeg = 50
	}
	if maxStrength <= 0 {
		maxStrength = 100
	}
	shape := jolt.CreateCapsule(height*0.5, radius)
	settings := jolt.NewCharacterVirtualSettings(shape)
	settings.MaxSlopeAngle = jolt.DegreesToRadians(maxSlopeDeg)
	settings.MaxStrength = maxStrength
	settings.EnhancedInternalEdgeRemoval = true
	inner := jolt.CreateCapsule(height*0.5, radius*0.85)
	cv := w.ps.CreateCharacterVirtualWithInner(settings, jolt.Vec3{X: x, Y: y, Z: z}, inner, 1)
	w.char[id] = &kinChar{
		x: x, y: y, z: z,
		radius:  radius + height*0.5,
		height:  height,
		virtual: cv,
	}
	w.rad[id] = radius + height*0.5
	w.mass[id] = 70
}

func (w *joltWorld) SetCharacterShape(id int, shapeType string, height, radius float32) {
	kc := w.char[id]
	if kc == nil || kc.virtual == nil {
		return
	}
	if height <= 0 {
		height = 1.8
	}
	if radius <= 0 {
		radius = 0.4
	}
	newShape := jolt.CreateCapsule(height*0.5, radius)
	if shapeType == "box" || shapeType == "Box" || shapeType == "BOX" {
		newShape = jolt.CreateBox(jolt.Vec3{X: radius, Y: height * 0.5, Z: radius})
	}
	kc.virtual.SetShape(newShape, 0.1)
	kc.height = height
	kc.radius = radius + height*0.5
	w.rad[id] = kc.radius
}

func (w *joltWorld) CharacterGround(id int) int {
	if w.CharacterGroundState(id) == 0 {
		return 1
	}
	return 0
}

func (w *joltWorld) CharacterGroundState(id int) int {
	kc := w.char[id]
	if kc == nil {
		return 3
	}
	if kc.virtual != nil {
		return int(kc.virtual.GetGroundState())
	}
	if kc.onGround {
		return 0
	}
	return 3
}

func (w *joltWorld) CharacterGroundNormal(id int) (float32, float32, float32) {
	kc := w.char[id]
	if kc == nil || kc.virtual == nil {
		if kc != nil && kc.onGround {
			return 0, 1, 0
		}
		return 0, 0, 0
	}
	n := kc.virtual.GetGroundNormal()
	return n.X, n.Y, n.Z
}

func (w *joltWorld) nearestBody(x, y, z float32, skip int) int {
	best := 0
	bestD := float32(1.2 * 1.2)
	for id, b := range w.body {
		if id == skip {
			continue
		}
		p := w.bi.GetPosition(b)
		dx, dy, dz := p.X-x, p.Y-y, p.Z-z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD {
			bestD = d2
			best = id
		}
	}
	return best
}

func (w *joltWorld) CharacterContact(id int) int {
	kc := w.char[id]
	if kc == nil || kc.virtual == nil {
		return 0
	}
	contacts := kc.virtual.GetActiveContacts(16)
	for i := 0; i < len(contacts); i++ {
		c := contacts[i]
		if c.WasDiscarded {
			continue
		}
		if c.BodyB != nil {
			if ent, ok := w.bodyToEnt[c.BodyB.Key()]; ok {
				return ent
			}
			for ent, b := range w.body {
				if b.Equal(c.BodyB) {
					return ent
				}
			}
			p := w.bi.GetPosition(c.BodyB)
			if ent := w.nearestBody(p.X, p.Y, p.Z, id); ent != 0 {
				return ent
			}
		}
		if ent := w.nearestBody(c.Position.X, c.Position.Y, c.Position.Z, id); ent != 0 {
			return ent
		}
	}
	return 0
}

func (w *joltWorld) resolvePhysID(id int) int {
	if w.body[id] != nil || w.char[id] != nil {
		return id
	}
	if id > 0 {
		if ent, ok := w.bodyVal[uint32(id)]; ok {
			return ent
		}
	}
	if ent, ok := w.bodyToEnt[uintptr(id)]; ok {
		return ent
	}
	return id
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

func (w *joltWorld) SetVelocity(id int, x, y, z float32) {
	id = w.resolvePhysID(id)
	if kc := w.char[id]; kc != nil && kc.virtual != nil {
		kc.virtual.SetLinearVelocity(jolt.Vec3{X: x, Y: y, Z: z})
		lv := kc.virtual.GetLinearVelocity()
		w.vel[id] = [3]float32{lv.X, lv.Y, lv.Z}
		return
	}
	w.vel[id] = [3]float32{x, y, z}
	if b, ok := w.body[id]; ok {
		w.bi.SetLinearVelocity(b, jolt.Vec3{X: x, Y: y, Z: z})
		w.bi.ActivateBody(b)
	}
}

func (w *joltWorld) GetVelocity(id int) (float32, float32, float32, bool) {
	id = w.resolvePhysID(id)
	if kc := w.char[id]; kc != nil && kc.virtual != nil {
		lv := kc.virtual.GetLinearVelocity()
		return lv.X, lv.Y, lv.Z, true
	}
	if b, ok := w.body[id]; ok && w.char[id] == nil {
		v := w.bi.GetLinearVelocity(b)
		return v.X, v.Y, v.Z, true
	}
	v, ok := w.vel[id]
	return v[0], v[1], v[2], ok
}

func (w *joltWorld) ApplyImpulse(id int, x, y, z float32) {
	id = w.resolvePhysID(id)
	if kc := w.char[id]; kc != nil && kc.virtual != nil {
		lv := kc.virtual.GetLinearVelocity()
		kc.virtual.SetLinearVelocity(jolt.Vec3{X: lv.X + x, Y: lv.Y + y, Z: lv.Z + z})
		lv = kc.virtual.GetLinearVelocity()
		w.vel[id] = [3]float32{lv.X, lv.Y, lv.Z}
		return
	}
	if b, ok := w.body[id]; ok {
		w.bi.AddImpulse(b, jolt.NewVec3(x, y, z))
		w.bi.ActivateBody(b)
		lv := w.bi.GetLinearVelocity(b)
		w.vel[id] = [3]float32{lv.X, lv.Y, lv.Z}
	}
}

func (w *joltWorld) ApplyForce(id int, x, y, z float32) {
	id = w.resolvePhysID(id)
	f := w.force[id]
	w.force[id] = [3]float32{f[0] + x, f[1] + y, f[2] + z}
	if b, ok := w.body[id]; ok {
		w.bi.AddForce(b, jolt.Vec3{X: x, Y: y, Z: z})
		w.bi.ActivateBody(b)
	}
}

func (w *joltWorld) SetAngularVelocity(id int, x, y, z float32) {
	id = w.resolvePhysID(id)
	w.ang[id] = [3]float32{x, y, z}
	if b, ok := w.body[id]; ok {
		w.bi.SetAngularVelocity(b, jolt.Vec3{X: x, Y: y, Z: z})
	}
}

func (w *joltWorld) GetAngularVelocity(id int) (float32, float32, float32, bool) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		v := w.bi.GetAngularVelocity(b)
		return v.X, v.Y, v.Z, true
	}
	v, ok := w.ang[id]
	return v[0], v[1], v[2], ok
}

func (w *joltWorld) SetMass(id int, mass float32) {
	id = w.resolvePhysID(id)
	w.mass[id] = mass
	if b, ok := w.body[id]; ok {
		w.ps.SetMass(b, mass)
	}
}

func (w *joltWorld) AddGround(id int, y float32) {
	w.AddBox(id, 0, y, 0, 80, 0.25, 80, false)
}

func (w *joltWorld) Raycast(ox, oy, oz, dx, dy, dz float32) (int, float32, float32, float32, bool) {
	hit, ok := w.ps.CastRay(jolt.Vec3{X: ox, Y: oy, Z: oz}, jolt.Vec3{X: dx, Y: dy, Z: dz})
	if !ok || hit.BodyID == nil {
		return 0, 0, 0, 0, false
	}
	id := 0
	if ent, found := w.bodyToEnt[hit.BodyID.Key()]; found {
		id = ent
	} else if v := hit.BodyID.Value(); v != 0 {
		id = w.bodyVal[v]
	} else {
		for ent, b := range w.body {
			if b.Equal(hit.BodyID) {
				id = ent
				break
			}
		}
	}
	return id, hit.HitPoint.X, hit.HitPoint.Y, hit.HitPoint.Z, true
}

func (w *joltWorld) SetGravity(x, y, z float32) {
	w.gx, w.gy, w.gz = x, y, z
	w.ps.SetGravity(jolt.Vec3{X: x, Y: y, Z: z})
}
func (w *joltWorld) GetGravity() (float32, float32, float32) {
	return w.gx, w.gy, w.gz
}

func (w *joltWorld) Remove(id int) {
	if kc := w.char[id]; kc != nil && kc.virtual != nil {
		kc.virtual.Destroy()
	}
	if b, ok := w.body[id]; ok {
		delete(w.bodyToEnt, b.Key())
		delete(w.bodyVal, b.Value())
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
	if w.tempAllocator != nil {
		w.tempAllocator.Destroy()
		w.tempAllocator = nil
	}
	jolt.Shutdown()
}
