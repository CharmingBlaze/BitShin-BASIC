//go:build !nojolt && windows

package phys3d

import "bitshinbasic/internal/jolt"

func (w *joltWorld) hitEntity(hit jolt.RaycastHit) int {
	if hit.BodyID == nil {
		return 0
	}
	if ent, found := w.bodyToEnt[hit.BodyID.Key()]; found {
		return ent
	}
	if v := hit.BodyID.Value(); v != 0 {
		return w.bodyVal[v]
	}
	return 0
}

func (w *joltWorld) ApplyBuoyancyImpulse(id int, sx, sy, sz, nx, ny, nz, buoyancy, linDrag, angDrag, fvx, fvy, fvz, dt float32) bool {
	id = w.resolvePhysID(id)
	b, ok := w.body[id]
	if !ok {
		return false
	}
	ok = w.bi.ApplyBuoyancyImpulse(b,
		jolt.NewVec3(sx, sy, sz), jolt.NewVec3(nx, ny, nz),
		buoyancy, linDrag, angDrag,
		jolt.NewVec3(fvx, fvy, fvz), jolt.NewVec3(w.gx, w.gy, w.gz), dt)
	if ok {
		w.bi.ActivateBody(b)
	}
	return ok
}

func (w *joltWorld) OffsetCenterOfMass(id int, ox, oy, oz float32) {
	id = w.resolvePhysID(id)
	b, ok := w.body[id]
	if !ok {
		return
	}
	inner := w.ps.GetBodyShape(b)
	if inner == nil {
		return
	}
	wrapped := jolt.OffsetCenterOfMass(inner, jolt.NewVec3(ox, oy, oz))
	if wrapped == nil {
		return
	}
	w.bi.SetShape(b, wrapped, true)
	w.bi.ActivateBody(b)
}

func (w *joltWorld) registerBody(id int, b *jolt.BodyID, r float32, motion int) {
	if b == nil {
		return
	}
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

func (w *joltWorld) AddMesh(id int, verts [][3]float32, indices []int32, motion int) {
	if len(verts) < 3 || len(indices) < 3 {
		return
	}
	points := make([]jolt.Vec3, len(verts))
	for i, v := range verts {
		points[i] = jolt.NewVec3(v[0], v[1], v[2])
	}
	shape := jolt.CreateMesh(points, indices)
	b := w.bi.CreateBodyEx(shape, jolt.Vec3{}, joltMotion(motion), false, true)
	w.registerBody(id, b, 1, motion)
}

func (w *joltWorld) AddHeightField(id int, samples []float32, n int, ox, oy, oz, sx, sy, sz float32) {
	shape := jolt.CreateHeightField(samples, n, jolt.NewVec3(ox, oy, oz), jolt.NewVec3(sx, sy, sz))
	if shape == nil {
		return
	}
	b := w.bi.CreateBodyEx(shape, jolt.Vec3{}, jolt.MotionTypeStatic, false, true)
	w.registerBody(id, b, 1, MotionTypeStatic)
}

func (w *joltWorld) AddSensorBox(id int, x, y, z, hx, hy, hz float32, motion int) {
	shape := jolt.CreateBox(jolt.NewVec3(hx, hy, hz))
	b := w.bi.CreateBodyEx(shape, jolt.NewVec3(x, y, z), joltMotion(motion), true, false)
	w.registerBody(id, b, hx, motion)
}

func (w *joltWorld) SetSensor(id int, on bool) {
	id = w.resolvePhysID(id)
	if b, ok := w.body[id]; ok {
		w.ps.SetBodySensor(b, on)
	}
}

func (w *joltWorld) ShapeCast(hx, hy, hz, x, y, z, dx, dy, dz float32) (int, float32, float32, float32, bool) {
	shape := jolt.CreateBox(jolt.NewVec3(hx, hy, hz))
	hit, ok := w.ps.CastShape(shape, jolt.NewVec3(x, y, z), jolt.NewVec3(dx, dy, dz))
	if !ok {
		return 0, 0, 0, 0, false
	}
	return w.hitEntity(hit), hit.HitPoint.X, hit.HitPoint.Y, hit.HitPoint.Z, true
}

func (w *joltWorld) OverlapSphere(x, y, z, r float32) (int, bool) {
	shape := jolt.CreateSphere(r)
	hits := w.ps.CollideShapeGetHits(shape, jolt.NewVec3(x, y, z), 8, 0)
	if len(hits) == 0 {
		return 0, false
	}
	id := 0
	if hits[0].BodyID != nil {
		id = w.hitEntity(jolt.RaycastHit{BodyID: hits[0].BodyID})
	}
	return id, id != 0 || hits[0].BodyID != nil
}

func (w *joltWorld) OverlapPoint(x, y, z float32) (int, bool) {
	hit, ok := w.ps.CollidePoint(jolt.NewVec3(x, y, z))
	if !ok {
		return 0, false
	}
	return w.hitEntity(hit), true
}

func (w *joltWorld) OptimizeBroadPhase() {
	if w.ps == nil {
		return
	}
	w.ps.OptimizeBroadPhase()
	w.pendingAdds = 0
}

func (w *joltWorld) AddCloth(id int, x, y, z, width, height float32, nx, ny, pinFlags int, thickness, damping, gravityFactor float32) {
	b := w.bi.CreateCloth(jolt.NewVec3(x, y, z), width, height, nx, ny, pinFlags, thickness, damping, gravityFactor)
	if b == nil {
		return
	}
	r := width
	if height > r {
		r = height
	}
	w.registerBody(id, b, r, MotionTypeDynamic)
}

func (w *joltWorld) ClothVertexCount(id int) int {
	id = w.resolvePhysID(id)
	b, ok := w.body[id]
	if !ok {
		return 0
	}
	return w.ps.ClothVertexCount(b)
}

func (w *joltWorld) ClothVertices(id int, dst []float32) int {
	id = w.resolvePhysID(id)
	b, ok := w.body[id]
	if !ok {
		return 0
	}
	return w.ps.ClothVertices(b, dst)
}

func (w *joltWorld) ApplyClothWind(id int, vx, vy, vz float32, start, step uint32) {
	id = w.resolvePhysID(id)
	b, ok := w.body[id]
	if !ok {
		return
	}
	w.ps.ApplyClothWind(b, vx, vy, vz, start, step)
}

func (w *joltWorld) OverlapSphereAll(x, y, z, r float32, max int) []int {
	if max <= 0 {
		max = 32
	}
	shape := jolt.CreateSphere(r)
	hits := w.ps.CollideShapeGetHits(shape, jolt.NewVec3(x, y, z), max, 0)
	out := make([]int, 0, len(hits))
	seen := map[int]bool{}
	for _, h := range hits {
		if h.BodyID == nil {
			continue
		}
		id := w.hitEntity(jolt.RaycastHit{BodyID: h.BodyID})
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func (w *joltWorld) AddCompound(id int, parts []CompoundPart, x, y, z float32, motion int) {
	if len(parts) == 0 {
		w.AddBoxEx(id, x, y, z, 0.5, 0.5, 0.5, motion)
		return
	}
	jp := make([]jolt.CompoundPart, len(parts))
	for i, p := range parts {
		jp[i] = jolt.CompoundPart{
			Kind: int32(p.Kind),
			Ox:   p.Ox, Oy: p.Oy, Oz: p.Oz,
			A: p.A, B: p.B, C: p.C,
		}
	}
	shape := jolt.CreateCompound(jp)
	hx, hy, hz := CompoundAABB(parts)
	if shape == nil {
		w.AddBoxEx(id, x, y, z, hx, hy, hz, motion)
		return
	}
	r := hx
	if hy > r {
		r = hy
	}
	if hz > r {
		r = hz
	}
	w.add(id, shape, x, y, z, r, motion)
}

func (w *joltWorld) SetCollisionLayer(id, layer int) {
	b, ok := w.bodyOf(id)
	if !ok {
		return
	}
	jolt.SetBodyCollisionLayer(b, layer)
}

func (w *joltWorld) SetLayerCollides(a, b int, on bool) {
	jolt.SetLayerPairCollides(a, b, on)
}
