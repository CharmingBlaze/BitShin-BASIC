package phys2d

import (
	"math"

	"github.com/jakecoffman/cp/v2"
)

// Character is a 2D platformer controller: a dynamic, non-rotating rounded box
// with zero friction whose velocity is set each step from the input. Ground is
// found with three short rays cast down from the body (centre, left, right),
// so a wall contact never counts as ground and the body slides off walls
// instead of sticking. While grounded gravity is off and the velocity follows
// the ground's tangent plus the ground body's own velocity, so it stands still
// on slopes up to MaxSlope and rides kinematic platforms.
//
// Sizes default from the box height, so they work in pixels or metres.
type Character struct {
	ID           int
	W, H         float64
	Speed        float64 // ground speed at full input, units per second
	JumpHeight   float64 // apex height of a jump, units
	GravityScale float64 // multiplies world gravity while airborne
	AirControl   float64 // 0 = keep momentum in the air, 1 = full control
	Coyote       float64 // seconds after leaving a ledge a jump still works
	JumpBuffer   float64 // seconds a jump press waits for landing
	MaxSlope     float64 // steepest walkable ground, degrees

	body   *cp.Body
	shape  *cp.Shape
	input  float64
	jumpQ  float64 // remaining buffer time of a pending jump press
	lock   float64 // ground ignored for this long after a jump
	since  float64 // seconds since last grounded
	jumped bool    // jumped since last grounded (no double coyote jumps)

	grounded bool
	groundID int
	groundN  cp.Vector
}

// AddCharacter creates a character body of size w x h centred at x, y.
func (s *Space) AddCharacter(id int, x, y, w, h float64) *Character {
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 2
	}
	b := s.sp.AddBody(cp.NewBody(1, math.Inf(1)))
	b.SetPosition(cp.Vector{X: x, Y: y})
	b.UserData = id
	r := math.Min(w, h) * 0.1
	sh := cp.NewBox(b, w-2*r, h-2*r, r)
	sh.SetFriction(0)
	sh.SetElasticity(0)
	s.sp.AddShape(sh)
	s.body[id] = b
	s.size[id] = [2]float64{w, h}
	s.track(id, 1, func(m float64) float64 { return math.Inf(1) })
	c := &Character{
		ID: id, W: w, H: h,
		Speed:        h * 5,
		JumpHeight:   h * 2,
		GravityScale: 1,
		AirControl:   0.8,
		Coyote:       0.1,
		JumpBuffer:   0.12,
		MaxSlope:     50,
		body:         b,
		shape:        sh,
		since:        1,
	}
	// The controller runs as the body's velocity integrator, after Chipmunk
	// has moved bodies and found contacts but before it solves them. Setting
	// the velocity any earlier would move the body before the solver sees it,
	// and a held direction key would push it through a crate pinned on a wall.
	b.SetVelocityUpdateFunc(func(body *cp.Body, g cp.Vector, damping, dt float64) {
		s.stepCharacter(c, dt)
		scale := c.GravityScale
		if c.grounded {
			scale = 0
		}
		cp.BodyUpdateVelocity(body, g.Mult(scale), damping, dt)
	})
	s.ext.chars[id] = c
	return c
}

// Character returns the controller on body id, or nil.
func (s *Space) Character(id int) *Character { return s.ext.chars[id] }

// Move sets this step's horizontal input, -1..1. It is cleared after each step.
func (c *Character) Move(dx float64) {
	c.input = math.Max(-1, math.Min(1, dx))
}

// Jump asks for a jump. It happens on the next step the character can jump
// (grounded, or within Coyote of leaving the ground), as long as that comes
// within JumpBuffer seconds.
func (c *Character) Jump() {
	c.jumpQ = math.Max(c.JumpBuffer, 1e-6)
}

// OnGround reports whether the last step found walkable ground under the feet.
func (c *Character) OnGround() bool { return c.grounded }

// GroundBody is the handle of the body stood on (0 in the air).
func (c *Character) GroundBody() int { return c.groundID }

// GroundNormal is the normal of the ground stood on (0,0 in the air).
func (c *Character) GroundNormal() (float64, float64) { return c.groundN.X, c.groundN.Y }

type groundHit struct {
	ok    bool
	gap   float64 // distance from the box's bottom to the ground; < 0 is inside
	n     cp.Vector
	point cp.Vector
	body  *cp.Body
}

// probeGround casts the three foot rays and returns the closest walkable hit.
func (s *Space) probeGround(c *Character, up, right cp.Vector) groundHit {
	pos := c.body.Position()
	half := c.H / 2
	skin := c.H * 0.1
	minDot := math.Cos(c.MaxSlope * math.Pi / 180)
	filter := c.shape.Filter
	best := groundHit{gap: math.Inf(1)}
	for _, off := range [3]float64{0, -0.45 * c.W, 0.45 * c.W} {
		start := pos.Add(right.Mult(off))
		end := start.Sub(up.Mult(half + skin))
		length := half + skin
		s.sp.SegmentQuery(start, end, 0, filter, func(sh *cp.Shape, p, n cp.Vector, alpha float64, _ any) {
			hb := sh.Body()
			if hb == c.body || sh.Sensor() || n.Dot(up) < minDot {
				return
			}
			gap := alpha*length - half
			// A one-way platform is ground only once the feet are on its top.
			if sh.CollisionType() == oneWayType && gap < -skin*0.5 {
				return
			}
			if gap < best.gap-1e-9 {
				best = groundHit{ok: true, gap: gap, n: n, point: p, body: hb}
			}
		}, nil)
	}
	return best
}

func (s *Space) stepCharacter(c *Character, dt float64) {
	up := s.up()
	right := cp.Vector{X: 1, Y: 0}.Sub(up.Mult(up.X))
	if right.LengthSq() < 1e-9 {
		right = up.ReversePerp()
	}
	right = right.Normalize()
	gravity := s.sp.Gravity().Length() * math.Max(c.GravityScale, 0)
	jumpSpeed := math.Sqrt(2 * gravity * math.Max(c.JumpHeight, 0))

	c.lock -= dt
	vel := c.body.Velocity()
	g := s.probeGround(c, up, right)
	var groundVel cp.Vector
	if g.ok {
		groundVel = g.body.VelocityAtWorldPoint(g.point)
		// Moving away from the surface (a jump, or passing up through a
		// one-way platform) is not standing on it.
		if c.lock > 0 || vel.Sub(groundVel).Dot(g.n) > math.Max(jumpSpeed*0.25, 1e-6) {
			g.ok = false
		}
	}
	c.grounded = g.ok
	if g.ok {
		c.since, c.jumped = 0, false
		c.groundID, c.groundN = idOf(g.body), g.n
	} else {
		c.since += dt
		c.groundID, c.groundN = 0, cp.Vector{}
	}

	target := c.input * c.Speed
	if g.ok {
		// Walk along the ground's tangent so slopes neither slow nor launch.
		t := g.n.Perp()
		if t.Dot(right) < 0 {
			t = t.Neg()
		}
		vel = groundVel.Add(t.Mult(target))
		if g.gap > 0 && dt > 0 {
			// Close the gap to the ground this step: lands without a bounce
			// and keeps the feet down when walking down a slope.
			vel = vel.Sub(up.Mult(g.gap / dt))
		}
	} else {
		h := vel.Dot(right)
		k := math.Max(0, math.Min(1, c.AirControl))
		if k > 0 {
			k = 1 - math.Pow(1-k, dt*60)
			if c.AirControl >= 1 {
				k = 1
			}
			vel = vel.Add(right.Mult((target - h) * k))
		}
	}

	if c.jumpQ > 0 && (g.ok || (c.since <= c.Coyote && !c.jumped)) {
		rise := math.Max(groundVel.Dot(up), 0)
		vel = vel.Sub(up.Mult(vel.Dot(up))).Add(up.Mult(jumpSpeed + rise))
		c.jumped, c.jumpQ = true, 0
		c.lock = 0.1
		c.grounded = false
		c.groundID, c.groundN = 0, cp.Vector{}
	}
	c.body.SetVelocity(vel.X, vel.Y)
	c.input = 0
	c.jumpQ -= dt
}
