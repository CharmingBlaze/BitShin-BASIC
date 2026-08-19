//go:build !nojolt && ((linux && (amd64 || arm64)) || (darwin && arm64))

package phys3d

import "github.com/bbitechnologies/jolt-go/jolt"

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
	cv := w.ps.CreateCharacterVirtual(settings, jolt.Vec3{X: x, Y: y, Z: z})
	inner := jolt.CreateCapsule(height*0.46, radius*0.92)
	w.add(id, inner, x, y, z, radius+height*0.5, MotionTypeKinematic)
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
			if ent := w.hitEntity(c.BodyB); ent != 0 {
				return ent
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
