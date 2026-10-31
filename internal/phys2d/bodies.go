package phys2d

import (
	"math"
	"sort"

	"github.com/jakecoffman/cp/v2"
)

// massInfo remembers what a body weighed so SetKinematic / SetStatic can hand
// it back: Chipmunk's SetType(DYNAMIC) rebuilds mass from the shapes, and our
// shapes carry none, which would leave the body weightless.
type massInfo struct {
	mass   float64
	moment func(m float64) float64
}

type sensorState struct {
	cur, prev map[int]bool
}

// extras is the state behind the newer 2D features (layers, sensors, one-way
// platforms, characters, casts). Nothing in it is touched until a script uses
// one of those commands, so older scripts step exactly as before.
type extras struct {
	mass      map[int]massInfo
	layer     map[int]int
	mask      map[int]uint
	layerMask [33]uint
	layersOn  bool
	sensors   map[int]*sensorState
	oneWay    map[int]bool
	oneWayOn  bool
	chars     map[int]*Character
	last      Hit
}

func newExtras() extras {
	e := extras{
		mass:    map[int]massInfo{},
		layer:   map[int]int{},
		mask:    map[int]uint{},
		sensors: map[int]*sensorState{},
		oneWay:  map[int]bool{},
		chars:   map[int]*Character{},
	}
	for i := range e.layerMask {
		e.layerMask[i] = cp.ALL_CATEGORIES
	}
	return e
}

// oneWayType is the Chipmunk collision type for one-way platform shapes.
const oneWayType cp.CollisionType = 0x6f6e65

// track records a new body's mass and puts it on its layer if layers are in use.
func (s *Space) track(id int, mass float64, moment func(m float64) float64) {
	if mass <= 0 {
		mass = 1
	}
	s.ext.mass[id] = massInfo{mass, moment}
	if s.ext.layersOn {
		s.applyFilter(id)
	}
}

func (s *Space) forget(id int) {
	delete(s.ext.mass, id)
	delete(s.ext.layer, id)
	delete(s.ext.mask, id)
	delete(s.ext.sensors, id)
	delete(s.ext.oneWay, id)
	delete(s.ext.chars, id)
	for _, st := range s.ext.sensors {
		delete(st.cur, id)
		delete(st.prev, id)
	}
}

func (s *Space) eachShape(id int, f func(*cp.Shape)) {
	if b := s.body[id]; b != nil {
		b.EachShape(f)
	}
}

// Has reports whether id names a 2D body.
func (s *Space) Has(id int) bool { return s.body[id] != nil }

// idOf maps a Chipmunk body back to its handle (0 when it has none).
func idOf(b *cp.Body) int {
	if b == nil {
		return 0
	}
	if id, ok := b.UserData.(int); ok {
		return id
	}
	return 0
}

func (s *Space) makeDynamic(id int, b *cp.Body) {
	b.SetType(cp.BODY_DYNAMIC)
	m, ok := s.ext.mass[id]
	if !ok || m.moment == nil {
		b.SetMass(1)
		b.SetMoment(1)
		return
	}
	b.SetMass(m.mass)
	if _, isChar := s.ext.chars[id]; isChar {
		b.SetMoment(math.Inf(1))
	} else {
		b.SetMoment(m.moment(m.mass))
	}
}

// SetKinematic turns a body kinematic (moved only by its velocity or position,
// pushes dynamic bodies, ignores gravity and forces) or back to dynamic.
func (s *Space) SetKinematic(id int, on bool) bool {
	b := s.body[id]
	if b == nil {
		return false
	}
	if on {
		v := b.Velocity()
		b.SetType(cp.BODY_KINEMATIC)
		b.SetVelocity(v.X, v.Y)
	} else {
		s.makeDynamic(id, b)
	}
	return true
}

// IsKinematic reports whether the body is kinematic.
func (s *Space) IsKinematic(id int) bool {
	b := s.body[id]
	return b != nil && b.GetType() == cp.BODY_KINEMATIC
}

// AddCapsule adds a capsule: a Chipmunk segment with a radius. The long axis
// follows the larger of w and h; w == h gives a circle.
func (s *Space) AddCapsule(id int, x, y, w, h, mass float64, dynamic bool) {
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	r := math.Min(w, h) / 2
	var a, bb cp.Vector
	if h >= w {
		a, bb = cp.Vector{X: 0, Y: -(h/2 - r)}, cp.Vector{X: 0, Y: h/2 - r}
	} else {
		a, bb = cp.Vector{X: -(w/2 - r), Y: 0}, cp.Vector{X: w/2 - r, Y: 0}
	}
	moment := func(m float64) float64 {
		// A segment's moment ignores its radius; add the end caps as a disc.
		return cp.MomentForSegment(m, a, bb, r) + cp.MomentForCircle(m, 0, r, cp.Vector{})
	}
	var b *cp.Body
	if !dynamic || mass <= 0 {
		b = s.sp.AddBody(cp.NewStaticBody())
	} else {
		b = s.sp.AddBody(cp.NewBody(mass, moment(mass)))
	}
	b.SetPosition(cp.Vector{X: x, Y: y})
	b.UserData = id
	var sh *cp.Shape
	if a == bb {
		sh = cp.NewCircle(b, r, cp.Vector{})
	} else {
		sh = cp.NewSegment(b, a, bb, r)
	}
	sh.SetFriction(0.7)
	sh.SetElasticity(0.2)
	s.sp.AddShape(sh)
	s.body[id] = b
	s.size[id] = [2]float64{w, h}
	s.track(id, mass, moment)
}

// ---- Collision layers -------------------------------------------------------

// MaxLayer is the highest 2D collision layer. Layers run 1..MaxLayer; bodies
// start on layer 1.
const MaxLayer = 32

func layerBit(l int) uint { return 1 << uint(l-1) }

func (s *Space) layerOf(id int) int {
	if l := s.ext.layer[id]; l >= 1 {
		return l
	}
	return 1
}

func (s *Space) filterFor(id int) cp.ShapeFilter {
	l := s.layerOf(id)
	mask := s.ext.layerMask[l]
	if m, ok := s.ext.mask[id]; ok {
		mask &= m
	}
	return cp.NewShapeFilter(cp.NO_GROUP, layerBit(l), mask)
}

func (s *Space) applyFilter(id int) {
	f := s.filterFor(id)
	s.eachShape(id, func(sh *cp.Shape) { sh.SetFilter(f) })
}

func (s *Space) enableLayers() {
	if s.ext.layersOn {
		return
	}
	s.ext.layersOn = true
	for id := range s.body {
		s.applyFilter(id)
	}
}

func (s *Space) refreshFilters() {
	for id := range s.body {
		s.applyFilter(id)
	}
}

// layersMeet reports whether the two bodies' layer filters let them touch.
func (s *Space) layersMeet(a, b int) bool {
	return !s.filterFor(a).Reject(s.filterFor(b))
}

// SetCollisionLayer puts a body on layer 1..MaxLayer (0 means 1).
func (s *Space) SetCollisionLayer(id, layer int) bool {
	if s.body[id] == nil || layer < 0 || layer > MaxLayer {
		return false
	}
	if layer == 0 {
		layer = 1
	}
	s.ext.layer[id] = layer
	s.enableLayers()
	s.applyFilter(id)
	return true
}

// CollisionLayer returns the body's layer (1 when never set).
func (s *Space) CollisionLayer(id int) int { return s.layerOf(id) }

// SetLayerCollides turns contact between two layers on or off (both ways).
func (s *Space) SetLayerCollides(a, b int, on bool) bool {
	if a < 1 || a > MaxLayer || b < 1 || b > MaxLayer {
		return false
	}
	if on {
		s.ext.layerMask[a] |= layerBit(b)
		s.ext.layerMask[b] |= layerBit(a)
	} else {
		s.ext.layerMask[a] &^= layerBit(b)
		s.ext.layerMask[b] &^= layerBit(a)
	}
	s.enableLayers()
	s.refreshFilters()
	return true
}

// SetCollisionMask sets which layers this body touches: bit 0 is layer 1.
// The mask narrows the layer table; it cannot re-enable a pair it turned off.
func (s *Space) SetCollisionMask(id int, mask uint) bool {
	if s.body[id] == nil {
		return false
	}
	s.ext.mask[id] = mask
	s.enableLayers()
	s.applyFilter(id)
	return true
}

// ---- Sensors ----------------------------------------------------------------

// SetSensor makes a body's shapes sensors: they report overlaps and push nothing.
func (s *Space) SetSensor(id int, on bool) bool {
	if s.body[id] == nil {
		return false
	}
	s.eachShape(id, func(sh *cp.Shape) { sh.SetSensor(on) })
	if on {
		if s.ext.sensors[id] == nil {
			s.ext.sensors[id] = &sensorState{cur: map[int]bool{}, prev: map[int]bool{}}
		}
	} else {
		delete(s.ext.sensors, id)
	}
	return true
}

// IsSensor reports whether SetSensor turned the body into a sensor.
func (s *Space) IsSensor(id int) bool { return s.ext.sensors[id] != nil }

// updateSensors refreshes each sensor's overlap set after a step. Static
// bodies are skipped, so a zone resting on the floor does not report the floor.
func (s *Space) updateSensors() {
	for id, st := range s.ext.sensors {
		st.prev, st.cur = st.cur, map[int]bool{}
		s.eachShape(id, func(sh *cp.Shape) {
			s.sp.ShapeQuery(sh, func(other *cp.Shape, _ *cp.ContactPointSet) {
				ob := other.Body()
				oid := idOf(ob)
				if oid == id || ob.GetType() == cp.BODY_STATIC || other.Sensor() {
					return
				}
				if s.body[oid] != ob {
					return
				}
				st.cur[oid] = true
			})
		})
	}
}

func sortedIDs(m map[int]bool, skip map[int]bool) []int {
	out := make([]int, 0, len(m))
	for id := range m {
		if skip == nil || !skip[id] {
			out = append(out, id)
		}
	}
	sort.Ints(out)
	return out
}

// SensorBodies lists the bodies inside the sensor after the last step, in id order.
func (s *Space) SensorBodies(id int) []int {
	if st := s.ext.sensors[id]; st != nil {
		return sortedIDs(st.cur, nil)
	}
	return nil
}

// SensorEntered lists bodies that came in during the last step.
func (s *Space) SensorEntered(id int) []int {
	if st := s.ext.sensors[id]; st != nil {
		return sortedIDs(st.cur, st.prev)
	}
	return nil
}

// SensorExited lists bodies that left during the last step.
func (s *Space) SensorExited(id int) []int {
	if st := s.ext.sensors[id]; st != nil {
		return sortedIDs(st.prev, st.cur)
	}
	return nil
}

// SensorHas reports whether body is inside the sensor.
func (s *Space) SensorHas(id, body int) bool {
	st := s.ext.sensors[id]
	return st != nil && st.cur[body]
}

// ---- One-way platforms -------------------------------------------------------

// SetOneWay makes a body a one-way platform: things pass up through it from
// below and land on its top ("up" is against gravity).
func (s *Space) SetOneWay(id int, on bool) bool {
	if s.body[id] == nil {
		return false
	}
	if on && !s.ext.oneWayOn {
		s.ext.oneWayOn = true
		h := s.sp.NewWildcardCollisionHandler(oneWayType)
		h.PreSolveFunc = func(arb *cp.Arbiter, _ *cp.Space, _ any) bool {
			// The handler sees the platform first, so the normal points from
			// the platform to the other shape: it must point up to collide.
			if arb.Normal().Dot(s.up()) < 0.3 {
				return arb.Ignore()
			}
			return true
		}
	}
	t := cp.CollisionType(0)
	if on {
		t = oneWayType
		s.ext.oneWay[id] = true
	} else {
		delete(s.ext.oneWay, id)
	}
	s.eachShape(id, func(sh *cp.Shape) { sh.SetCollisionType(t) })
	return true
}

// up is the direction against gravity; +Y when gravity is zero.
func (s *Space) up() cp.Vector {
	g := s.sp.Gravity()
	if g.LengthSq() < 1e-12 {
		return cp.Vector{X: 0, Y: 1}
	}
	return g.Neg().Normalize()
}
