//go:build !nojolt && ((linux && (amd64 || arm64)) || (darwin && arm64))

package phys3d

import (
	"math"

	"github.com/bbitechnologies/jolt-go/jolt"
)

// jolt-go C wrapper has no two-body constraints. IDs stay 0 on those platforms.

func (w *joltWorld) CreateHingeJoint(a, b int, px, py, pz, ax, ay, az float32) int {
	return w.addSoftJoint(a, b, px, py, pz, ax, ay, az, 0, 0, 0, 1)
}

func (w *joltWorld) CreatePointJoint(a, b int, px, py, pz float32) int {
	return w.addSoftJoint(a, b, px, py, pz, 0, 1, 0, 0, 0, 0, 1)
}

func (w *joltWorld) CreateSliderJoint(a, b int, px, py, pz, ax, ay, az float32) int {
	return w.addSoftJoint(a, b, px, py, pz, ax, ay, az, 0, 0, 0, 2)
}

func (w *joltWorld) CreateSpringJoint(a, b int, px, py, pz, rest, stiff, damp float32) int {
	return w.addSoftJoint(a, b, px, py, pz, 0, 1, 0, rest, stiff, damp, 3)
}

func (w *joltWorld) GetRotation(id int) (float32, float32, float32, float32, bool) {
	if _, ok := w.body[id]; !ok && w.char[id] == nil {
		return 0, 0, 0, 1, false
	}
	q := w.rot[id]
	if q[0] == 0 && q[1] == 0 && q[2] == 0 && q[3] == 0 {
		return 0, 0, 0, 1, true
	}
	return q[0], q[1], q[2], q[3], true
}

func (w *joltWorld) SetRotation(id int, x, y, z, qw float32) {
	x, y, z, qw = quatNormalize(x, y, z, qw)
	w.rot[id] = [4]float32{x, y, z, qw}
}

func (w *joltWorld) SetCCD(int, bool) int { return 0 }

func (w *joltWorld) RemoveJoint(id int) {
	delete(w.soft, id)
}

func (w *joltWorld) PollContacts(int) []ContactEvent { return nil }

func (w *joltWorld) EnableContacts() {}

func (w *joltWorld) LookupBody(v uint32) int { return w.bodyVal[v] }

func (w *joltWorld) ApplyTorque(id int, x, y, z float32) {
	t := w.torque[id]
	w.torque[id] = [3]float32{t[0] + x, t[1] + y, t[2] + z}
}

func (w *joltWorld) ApplyForceAtPosition(id int, fx, fy, fz, px, py, pz float32) {
	w.ApplyForce(id, fx, fy, fz)
	ax, ay, az, ok := w.GetPosition(id)
	if !ok {
		return
	}
	rx, ry, rz := px-ax, py-ay, pz-az
	w.ApplyTorque(id, ry*fz-rz*fy, rz*fx-rx*fz, rx*fy-ry*fx)
}

func (w *joltWorld) ApplyLocalImpulse(id int, lx, ly, lz float32) {
	qx, qy, qz, qw, _ := w.GetRotation(id)
	wx, wy, wz := quatRotateVec(qx, qy, qz, qw, lx, ly, lz)
	w.ApplyImpulse(id, wx, wy, wz)
}

func (w *joltWorld) SetGravityScale(id int, scale float32) { w.gscale[id] = scale }

func (w *joltWorld) SetRestitution(int, float32) {}

func (w *joltWorld) SetLinearDamping(id int, d float32) { w.linDamp[id] = d }

func (w *joltWorld) SetAngularDamping(id int, d float32) { w.angDamp[id] = d }

func (w *joltWorld) SetFriction(int, float32) {}

func (w *joltWorld) SetHingeLimits(int, float32, float32) {}

func (w *joltWorld) SetHingeFriction(int, float32) {}

func (w *joltWorld) SetHingeMotor(int, float32, float32) {}

func (w *joltWorld) DisableBodyCollision(a, b int) {
	if w.nocol == nil {
		w.nocol = map[[2]int]bool{}
	}
	w.nocol[pairKey(a, b)] = true
}

func (w *joltWorld) CreateWheeledVehicle(int, float32, float32, float32) int { return 0 }

func (w *joltWorld) CreateMotorcycleVehicle(int, float32, float32, float32) int { return 0 }

func (w *joltWorld) CreateTrackedVehicle(int, float32, float32, float32) int { return 0 }

func (w *joltWorld) SetVehicleInput(int, float32, float32, float32) {}

func (w *joltWorld) ApplyBuoyancyImpulse(id int, sx, sy, sz, nx, ny, nz, buoyancy, linDrag, angDrag, fvx, fvy, fvz, dt float32) bool {
	if w.locked[id] || (w.body[id] == nil && w.char[id] == nil) {
		return false
	}
	_, y, _, ok := w.GetPosition(id)
	if !ok {
		return false
	}
	depth := sy - y
	if depth <= 0 {
		return false
	}
	if buoyancy < 0.01 {
		buoyancy = 1
	}
	if dt <= 0 {
		dt = 1.0 / 60
	}
	m := w.mass[id]
	if m < 0.001 {
		m = 1
	}
	g := w.gy
	if g > 0 {
		g = -g
	}
	w.ApplyForce(id, nx*buoyancy*m*2, -g*buoyancy*m*depth, nz*buoyancy*m*2)
	if linDrag > 0 {
		cur := w.linDamp[id]
		if linDrag > cur {
			w.linDamp[id] = linDrag
		}
	}
	if angDrag > 0 {
		cur := w.angDamp[id]
		if angDrag > cur {
			w.angDamp[id] = angDrag
		}
	}
	_ = sx
	_ = sz
	_ = fvx
	_ = fvy
	_ = fvz
	return true
}

func (w *joltWorld) OffsetCenterOfMass(id int, ox, oy, oz float32) {
	w.com[id] = [3]float32{ox, oy, oz}
}

func (w *joltWorld) SetSensor(id int, on bool) {
	if w.body[id] == nil {
		return
	}
	x, y, z, ok := w.GetPosition(id)
	if !ok {
		return
	}
	r := w.rad[id]
	if r < 0.1 {
		r = 0.1
	}
	b := w.body[id]
	key := joltHandle(b)
	delete(w.bodyToEnt, key)
	delete(w.bodyVal, uint32(key))
	b.Destroy()
	delete(w.body, id)
	mot := w.motion[id]
	w.addSensor(id, jolt.CreateSphere(r), x, y, z, r, mot, on)
}

func (w *joltWorld) AddMesh(id int, verts [][3]float32, indices []int32, motion int) {
	if len(verts) < 3 || len(indices) < 3 {
		hx, hy, hz := hullHalfExtents(verts)
		w.AddBoxEx(id, 0, 0, 0, hx, hy, hz, motion)
		return
	}
	points := make([]jolt.Vec3, len(verts))
	for i, v := range verts {
		points[i] = jolt.Vec3{X: v[0], Y: v[1], Z: v[2]}
	}
	shape := jolt.CreateMesh(points, indices)
	if shape == nil {
		hx, hy, hz := hullHalfExtents(verts)
		cx, cy, cz := float32(0), float32(0), float32(0)
		for _, v := range verts {
			cx += v[0]
			cy += v[1]
			cz += v[2]
		}
		n := float32(len(verts))
		w.AddBoxEx(id, cx/n, cy/n, cz/n, hx, hy, hz, motion)
		return
	}
	w.add(id, shape, 0, 0, 0, hullRadius(verts), motion)
}

func (w *joltWorld) AddHeightField(id int, samples []float32, n int, ox, oy, oz, sx, sy, sz float32) {
	verts, idx := heightFieldTris(samples, n, ox, oy, oz, sx, sy, sz)
	if len(verts) == 0 {
		w.AddBox(id, ox, oy, oz, 8, 0.25, 8, false)
		return
	}
	w.AddMesh(id, verts, idx, MotionTypeStatic)
}

func (w *joltWorld) AddSensorBox(id int, x, y, z, hx, hy, hz float32, motion int) {
	r := hx
	if hy > r {
		r = hy
	}
	if hz > r {
		r = hz
	}
	w.addSensor(id, jolt.CreateBox(jolt.Vec3{X: hx, Y: hy, Z: hz}), x, y, z, r, motion, true)
}

func (w *joltWorld) collideHits(shape *jolt.Shape, x, y, z float32, max int) []int {
	if shape == nil {
		return nil
	}
	if max <= 0 {
		max = 32
	}
	hits := w.ps.CollideShapeGetHits(shape, jolt.Vec3{X: x, Y: y, Z: z}, max, 0)
	out := make([]int, 0, len(hits))
	for i := 0; i < len(hits); i++ {
		id := w.hitEntity(hits[i].BodyID)
		if id == 0 {
			id = w.entityNear(hits[i].ContactPoint.X, hits[i].ContactPoint.Y, hits[i].ContactPoint.Z)
		}
		if id == 0 {
			continue
		}
		if !w.layersCollide(id, 0) && w.layer[id] != 0 {
			continue
		}
		out = append(out, id)
		if len(out) >= max {
			break
		}
	}
	return out
}

func (w *joltWorld) ShapeCast(hx, hy, hz, x, y, z, dx, dy, dz float32) (int, float32, float32, float32, bool) {
	if hx < 0.05 {
		hx = 0.05
	}
	if hy < 0.05 {
		hy = 0.05
	}
	if hz < 0.05 {
		hz = 0.05
	}
	shape := jolt.CreateBox(jolt.Vec3{X: hx, Y: hy, Z: hz})
	if shape != nil {
		defer shape.Destroy()
	}
	steps := 12
	for i := 1; i <= steps; i++ {
		t := float32(i) / float32(steps)
		px, py, pz := x+dx*t, y+dy*t, z+dz*t
		ids := w.collideHits(shape, px, py, pz, 1)
		if len(ids) > 0 {
			return ids[0], px, py, pz, true
		}
	}
	return w.Raycast(x, y, z, dx, dy, dz)
}

func (w *joltWorld) AddCloth(id int, x, y, z, width, height float32, nx, ny, pinFlags int, thickness, damping, gravityFactor float32) {
	_ = thickness
	_ = gravityFactor
	c := newClothSim(x, y, z, width, height, nx, ny, pinFlags, damping)
	if w.cloths == nil {
		w.cloths = map[int]*clothSim{}
	}
	w.cloths[id] = c
	cx, cy, cz := c.com()
	w.AddBox(id, cx, cy, cz, width*0.5, 0.05, 0.05, true)
}

func (w *joltWorld) stepCloths(dt float32) {
	g := w.gy
	if g == 0 {
		g = -9.81
	}
	for id, c := range w.cloths {
		if c == nil {
			continue
		}
		c.step(dt, g)
		cx, cy, cz := c.com()
		w.SetPosition(id, cx, cy, cz)
	}
}

func (w *joltWorld) ClothVertexCount(id int) int {
	c := w.cloths[id]
	if c == nil {
		return 0
	}
	return len(c.pos)
}

func (w *joltWorld) ClothVertices(id int, dst []float32) int {
	c := w.cloths[id]
	if c == nil {
		return 0
	}
	cx, cy, cz := c.com()
	n := len(c.pos)
	if n*3 > len(dst) {
		n = len(dst) / 3
	}
	for i := 0; i < n; i++ {
		dst[i*3+0] = c.pos[i][0] - cx
		dst[i*3+1] = c.pos[i][1] - cy
		dst[i*3+2] = c.pos[i][2] - cz
	}
	return n
}

func (w *joltWorld) ApplyClothWind(id int, vx, vy, vz float32, start, step uint32) {
	c := w.cloths[id]
	if c == nil {
		return
	}
	if step < 1 {
		step = 1
	}
	phase := float32(start) * 0.11
	for i := 0; i < len(c.pos); i++ {
		if c.inv[i] == 0 {
			continue
		}
		ripple := float32(math.Sin(float64(phase + float32(i)*0.37)))
		s := float32(0.016)
		if step > 1 && (i+int(start))%int(step) != 0 {
			s *= 0.35
		}
		c.pos[i][0] += vx * s
		c.pos[i][1] += vy * s
		c.pos[i][2] += vz*s + ripple*0.02
	}
}

func (w *joltWorld) OverlapSphere(x, y, z, r float32) (int, bool) {
	ids := w.OverlapSphereAll(x, y, z, r, 1)
	if len(ids) == 0 {
		return 0, false
	}
	return ids[0], true
}

func (w *joltWorld) OverlapPoint(x, y, z float32) (int, bool) {
	return w.OverlapSphere(x, y, z, 0.04)
}

func (w *joltWorld) OverlapSphereAll(x, y, z, r float32, max int) []int {
	if r < 0.02 {
		r = 0.02
	}
	shape := jolt.CreateSphere(r)
	if shape != nil {
		defer shape.Destroy()
	}
	ids := w.collideHits(shape, x, y, z, max)
	if len(ids) > 0 {
		return ids
	}
	if max <= 0 {
		max = 32
	}
	out := make([]int, 0, 4)
	for id, b := range w.body {
		p := w.bi.GetPosition(b)
		dx, dy, dz := p.X-x, p.Y-y, p.Z-z
		lim := r + w.rad[id]
		if dx*dx+dy*dy+dz*dz <= lim*lim {
			out = append(out, id)
			if len(out) >= max {
				break
			}
		}
	}
	return out
}

func (w *joltWorld) OptimizeBroadPhase() {}

func (w *joltWorld) CreateGrabJoint(a, b int, px, py, pz, freq, damp float32) int {
	if freq <= 0 {
		freq = 8
	}
	return w.addSoftJoint(a, b, px, py, pz, 0, 1, 0, 0.05, freq*20, damp, 3)
}

func (w *joltWorld) CreateFixedJoint(a, b int, px, py, pz float32) int {
	return w.addSoftJoint(a, b, px, py, pz, 0, 1, 0, 0, 0, 0, 1)
}

func (w *joltWorld) CreateConeJoint(a, b int, px, py, pz, ax, ay, az, halfConeDeg float32) int {
	_ = halfConeDeg
	return w.addSoftJoint(a, b, px, py, pz, ax, ay, az, 0, 0, 0, 1)
}

func (w *joltWorld) CreateSwingTwistJoint(a, b int, px, py, pz, ax, ay, az, swingDeg, twistDeg float32) int {
	_ = swingDeg
	_ = twistDeg
	return w.addSoftJoint(a, b, px, py, pz, ax, ay, az, 0, 0, 0, 1)
}

func (w *joltWorld) AddCompound(id int, parts []CompoundPart, x, y, z float32, motion int) {
	pts := compoundHullPoints(parts)
	if len(pts) >= 3 {
		w.AddConvexHull(id, pts, x, y, z, motion)
		return
	}
	hx, hy, hz := CompoundAABB(parts)
	w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
}

func (w *joltWorld) SetCollisionLayer(id, layer int) {
	w.layer[id] = clampLayer(layer)
}

func (w *joltWorld) SetLayerCollides(a, b int, on bool) {
	a, b = clampLayer(a), clampLayer(b)
	off := !on
	w.layerOff[a][b] = off
	w.layerOff[b][a] = off
}

func (w *joltWorld) layersCollide(a, b int) bool {
	if w.nocol[pairKey(a, b)] {
		return false
	}
	la, lb := clampLayer(w.layer[a]), clampLayer(w.layer[b])
	return !w.layerOff[la][lb]
}

func (w *joltWorld) addSoftJoint(a, b int, px, py, pz, ax, ay, az, rest, stiff, damp float32, kind int) int {
	if a != 0 {
		_, hasA := w.body[a]
		if !hasA && w.char[a] == nil {
			return 0
		}
	}
	if b != 0 {
		_, hasB := w.body[b]
		if !hasB && w.char[b] == nil {
			return 0
		}
	}
	if w.soft == nil {
		w.soft = map[int]*softJoint{}
	}
	if w.nextConstraintID < 1 {
		w.nextConstraintID = 1
	}
	j := &softJoint{a: a, b: b, px: px, py: py, pz: pz, ax: ax, ay: ay, az: az, rest: rest, stiff: stiff, damp: damp, kind: kind}
	if ax, ay, az, ok := w.GetPosition(a); ok {
		j.lax, j.lay, j.laz = px-ax, py-ay, pz-az
	}
	if bx, by, bz, ok := w.GetPosition(b); ok {
		j.lbx, j.lby, j.lbz = px-bx, py-by, pz-bz
	}
	id := w.nextConstraintID
	w.nextConstraintID++
	w.soft[id] = j
	return id
}

func (w *joltWorld) solveSoftJoints() {
	for _, j := range w.soft {
		if j == nil {
			continue
		}
		if w.nocol[pairKey(j.a, j.b)] || !w.layersCollide(j.a, j.b) {
			continue
		}
		wx, wy, wz := j.px, j.py, j.pz
		if ax, ay, az, ok := w.GetPosition(j.a); ok {
			wx, wy, wz = ax+j.lax, ay+j.lay, az+j.laz
		}
		bx, by, bz, ok := w.GetPosition(j.b)
		if !ok {
			continue
		}
		cx, cy, cz := bx+j.lbx, by+j.lby, bz+j.lbz
		dx, dy, dz := wx-cx, wy-cy, wz-cz
		dx, dy, dz = softConstraintDelta(j.kind, j.ax, j.ay, j.az, dx, dy, dz, j.rest)
		if j.kind == 3 && j.stiff > 0 {
			w.ApplyImpulse(j.b, dx*j.stiff*0.02, dy*j.stiff*0.02, dz*j.stiff*0.02)
			continue
		}
		if w.locked[j.b] {
			if j.a != 0 && !w.locked[j.a] {
				if ax, ay, az, aok := w.GetPosition(j.a); aok {
					w.SetPosition(j.a, ax-dx, ay-dy, az-dz)
				}
			}
			continue
		}
		w.SetPosition(j.b, bx+dx, by+dy, bz+dz)
	}
}
