// Package phys2d is Chipmunk2D via jakecoffman/cp (pure Go, no CGO).
package phys2d

import "github.com/jakecoffman/cp/v2"

// Space is a Chipmunk2D world via jakecoffman/cp/v2 (pure Go, no CGO).
// go-zero/go-chipmunk is abandoned (2015); ByteArena/box2d is frozen at 2017;
// CGO Box2D bindings add Windows compiler pain. cp is the maintained game-oriented pick.
type Space struct {
	sp     *cp.Space
	body   map[int]*cp.Body
	size   map[int][2]float64
	hits   map[int][]int
	joint  map[int]*cp.Constraint
	nextJ  int
	ready  bool
}

func New() *Space {
	s := cp.NewSpace()
	s.Iterations = 10
	s.SetGravity(cp.Vector{X: 0, Y: -200})
	return &Space{
		sp:    s,
		body:  map[int]*cp.Body{},
		size:  map[int][2]float64{},
		hits:  map[int][]int{},
		joint: map[int]*cp.Constraint{},
		nextJ: 1,
		ready: true,
	}
}

func (s *Space) Ready() bool { return s != nil && s.ready }

func (s *Space) Step(dt float64) {
	if dt <= 0 || dt > 0.1 {
		dt = 1.0 / 60.0
	}
	s.sp.Step(dt)
	s.collectHits()
}

func (s *Space) SetGravity(x, y float64) {
	s.sp.SetGravity(cp.Vector{X: x, Y: y})
}

func (s *Space) AddCircle(id int, x, y, radius, mass float64, dynamic bool) {
	var b *cp.Body
	if !dynamic || mass <= 0 {
		b = s.sp.AddBody(cp.NewStaticBody())
	} else {
		b = s.sp.AddBody(cp.NewBody(mass, cp.MomentForCircle(mass, 0, radius, cp.Vector{})))
	}
	b.SetPosition(cp.Vector{X: x, Y: y})
	b.UserData = id
	sh := cp.NewCircle(b, radius, cp.Vector{})
	sh.SetFriction(0.7)
	sh.SetElasticity(0.4)
	s.sp.AddShape(sh)
	s.body[id] = b
	s.size[id] = [2]float64{radius * 2, radius * 2}
}

func (s *Space) AddBox(id int, x, y, w, h, mass float64, dynamic bool) {
	var b *cp.Body
	if !dynamic || mass <= 0 {
		b = s.sp.AddBody(cp.NewStaticBody())
	} else {
		b = s.sp.AddBody(cp.NewBody(mass, cp.MomentForBox(mass, w, h)))
	}
	b.SetPosition(cp.Vector{X: x, Y: y})
	b.UserData = id
	sh := cp.NewBox(b, w, h, 0)
	sh.SetFriction(0.8)
	sh.SetElasticity(0.2)
	s.sp.AddShape(sh)
	s.body[id] = b
	s.size[id] = [2]float64{w, h}
}

func (s *Space) SetPosition(id int, x, y float64) {
	if b := s.body[id]; b != nil {
		b.SetPosition(cp.Vector{X: x, Y: y})
	}
}

func (s *Space) GetPosition(id int) (x, y float64, ok bool) {
	b := s.body[id]
	if b == nil {
		return 0, 0, false
	}
	p := b.Position()
	return p.X, p.Y, true
}

func (s *Space) SetVelocity(id int, x, y float64) {
	if b := s.body[id]; b != nil {
		b.SetVelocity(x, y)
	}
}

func (s *Space) GetVelocity(id int) (x, y float64, ok bool) {
	b := s.body[id]
	if b == nil {
		return 0, 0, false
	}
	v := b.Velocity()
	return v.X, v.Y, true
}

func (s *Space) ApplyImpulse(id int, x, y float64) {
	if b := s.body[id]; b != nil {
		b.ApplyImpulseAtLocalPoint(cp.Vector{X: x, Y: y}, cp.Vector{})
	}
}

func (s *Space) Angle(id int) float64 {
	if b := s.body[id]; b != nil {
		return b.Angle() * 180 / 3.141592653589793
	}
	return 0
}

func (s *Space) Remove(id int) {
	if b := s.body[id]; b != nil {
		for jid, c := range s.joint {
			if c != nil && (c.A() == b || c.B() == b) {
				s.RemoveJoint(jid)
			}
		}
		s.sp.RemoveBody(b)
		delete(s.body, id)
		delete(s.size, id)
		delete(s.hits, id)
	}
}

func (s *Space) allocJoint() int {
	id := s.nextJ
	s.nextJ++
	return id
}

func (s *Space) AddPin(a, b int, ax, ay, bx, by float64) int {
	ba, bb := s.body[a], s.body[b]
	if ba == nil || bb == nil {
		return 0
	}
	c := cp.NewPinJoint(ba, bb, cp.Vector{X: ax, Y: ay}, cp.Vector{X: bx, Y: by})
	s.sp.AddConstraint(c)
	id := s.allocJoint()
	s.joint[id] = c
	return id
}

func (s *Space) AddSpring(a, b int, rest, stiff, damp, ax, ay, bx, by float64) int {
	ba, bb := s.body[a], s.body[b]
	if ba == nil || bb == nil {
		return 0
	}
	if rest <= 0 {
		pa, pb := ba.Position(), bb.Position()
		dx, dy := pb.X-pa.X, pb.Y-pa.Y
		rest = mathHypot(dx, dy)
		if rest <= 0 {
			rest = 1
		}
	}
	if stiff <= 0 {
		stiff = 100
	}
	if damp < 0 {
		damp = 0
	}
	c := cp.NewDampedSpring(ba, bb, cp.Vector{X: ax, Y: ay}, cp.Vector{X: bx, Y: by}, rest, stiff, damp)
	s.sp.AddConstraint(c)
	id := s.allocJoint()
	s.joint[id] = c
	return id
}

func (s *Space) AddSlide(a, b int, min, max, ax, ay, bx, by float64) int {
	ba, bb := s.body[a], s.body[b]
	if ba == nil || bb == nil {
		return 0
	}
	if max < min {
		min, max = max, min
	}
	c := cp.NewSlideJoint(ba, bb, cp.Vector{X: ax, Y: ay}, cp.Vector{X: bx, Y: by}, min, max)
	s.sp.AddConstraint(c)
	id := s.allocJoint()
	s.joint[id] = c
	return id
}

func (s *Space) RemoveJoint(id int) {
	if c := s.joint[id]; c != nil {
		s.sp.RemoveConstraint(c)
		delete(s.joint, id)
	}
}

func mathHypot(x, y float64) float64 {
	if x < 0 {
		x = -x
	}
	if y < 0 {
		y = -y
	}
	if x < y {
		x, y = y, x
	}
	if x == 0 {
		return 0
	}
	r := y / x
	return x * (1 + r*r*0.5)
}

func (s *Space) SetStatic(id int, stat bool) {
	if b := s.body[id]; b != nil {
		if stat {
			b.SetType(cp.BODY_STATIC)
		} else {
			b.SetType(cp.BODY_DYNAMIC)
		}
	}
}

func (s *Space) SetMass(id int, mass float64) {
	if b := s.body[id]; b != nil && mass > 0 {
		b.SetMass(mass)
	}
}

func (s *Space) ApplyForce(id int, x, y float64) {
	if b := s.body[id]; b != nil {
		b.ApplyForceAtLocalPoint(cp.Vector{X: x, Y: y}, cp.Vector{})
	}
}

func (s *Space) Collides(a, b int) bool {
	for _, id := range s.hits[a] {
		if id == b {
			return true
		}
	}
	return false
}

func (s *Space) CountHits(id int) int { return len(s.hits[id]) }

func (s *Space) Hits(id int) []int { return s.hits[id] }

func (s *Space) AddPoly(id int, x, y float64, verts [][2]float64, mass float64, dynamic bool) {
	if len(verts) < 3 {
		s.AddBox(id, x, y, 1, 1, mass, dynamic)
		return
	}
	cv := make([]cp.Vector, len(verts))
	minX, minY, maxX, maxY := verts[0][0], verts[0][1], verts[0][0], verts[0][1]
	for i := 0; i < len(verts); i++ {
		cv[i] = cp.Vector{X: verts[i][0], Y: verts[i][1]}
		if verts[i][0] < minX {
			minX = verts[i][0]
		}
		if verts[i][0] > maxX {
			maxX = verts[i][0]
		}
		if verts[i][1] < minY {
			minY = verts[i][1]
		}
		if verts[i][1] > maxY {
			maxY = verts[i][1]
		}
	}
	var b *cp.Body
	if !dynamic || mass <= 0 {
		b = s.sp.AddBody(cp.NewStaticBody())
	} else {
		b = s.sp.AddBody(cp.NewBody(mass, cp.MomentForPoly(mass, len(cv), cv, cp.Vector{}, 0)))
	}
	b.SetPosition(cp.Vector{X: x, Y: y})
	b.UserData = id
	sh := cp.NewPolyShape(b, len(cv), cv, cp.NewTransformIdentity(), 0)
	sh.SetFriction(0.7)
	sh.SetElasticity(0.3)
	s.sp.AddShape(sh)
	s.body[id] = b
	s.size[id] = [2]float64{maxX - minX, maxY - minY}
}

func (s *Space) SetCCD(on bool) {
	if on {
		s.sp.Iterations = 20
		s.sp.SetCollisionSlop(0.05)
	} else {
		s.sp.Iterations = 10
		s.sp.SetCollisionSlop(0.1)
	}
}

func (s *Space) Raycast(x1, y1, x2, y2 float64) (id int, x, y float64, ok bool) {
	info := s.sp.SegmentQueryFirst(cp.Vector{X: x1, Y: y1}, cp.Vector{X: x2, Y: y2}, 0, cp.SHAPE_FILTER_ALL)
	if info.Shape == nil {
		return 0, 0, 0, false
	}
	hit := info.Shape.Body()
	for i, b := range s.body {
		if b == hit {
			return i, info.Point.X, info.Point.Y, true
		}
	}
	return 0, info.Point.X, info.Point.Y, true
}

func (s *Space) collectHits() {
	s.hits = map[int][]int{}
	ids := make([]int, 0, len(s.body))
	for id := range s.body {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids); i++ {
		ax, ay, okA := s.GetPosition(ids[i])
		if !okA {
			continue
		}
		aw, ah := s.size[ids[i]][0], s.size[ids[i]][1]
		if aw <= 0 {
			aw, ah = 2, 2
		}
		for j := i + 1; j < len(ids); j++ {
			bx, by, okB := s.GetPosition(ids[j])
			if !okB {
				continue
			}
			bw, bh := s.size[ids[j]][0], s.size[ids[j]][1]
			if bw <= 0 {
				bw, bh = 2, 2
			}
			if ax-aw/2 < bx+bw/2 && ax+aw/2 > bx-bw/2 && ay-ah/2 < by+bh/2 && ay+ah/2 > by-bh/2 {
				s.hits[ids[i]] = append(s.hits[ids[i]], ids[j])
				s.hits[ids[j]] = append(s.hits[ids[j]], ids[i])
			}
		}
	}
}
