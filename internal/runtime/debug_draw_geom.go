package runtime

import (
	"github.com/g3n/engine/math32"
)

// Line geometry for the debug overlay. Everything here is G3N world space
// (right-handed, -Z forward) and pure, so the tests can count lines
// without a GL context.

// dbgColor is a straight RGBA line colour, 0-1.
type dbgColor [4]float32

var (
	dbgColDynamic  = dbgColor{0.25, 0.85, 1, 1}
	dbgColStatic   = dbgColor{0.55, 0.6, 0.7, 1}
	dbgColKinem    = dbgColor{1, 0.6, 0.15, 1}
	dbgColSleep    = dbgColor{0.35, 0.35, 0.9, 1}
	dbgColSensor   = dbgColor{0.2, 1, 0.25, 1}
	dbgColChar     = dbgColor{1, 0.9, 0.2, 1}
	dbgColContact  = dbgColor{1, 0.15, 0.15, 1}
	dbgColNormal   = dbgColor{1, 1, 0.2, 1}
	dbgColVelocity = dbgColor{0.2, 1, 0.5, 1}
	dbgColJoint    = dbgColor{1, 0.35, 1, 1}
	dbgColRayMiss  = dbgColor{0.8, 0.8, 0.8, 0.8}
	dbgColRayHit   = dbgColor{1, 0.5, 0.1, 1}
	dbgColNav      = dbgColor{0.1, 0.8, 0.9, 0.9}
	dbgColNavPath  = dbgColor{1, 0.85, 0.1, 1}
	dbgColLight    = dbgColor{1, 0.95, 0.5, 0.9}
	dbgColBounds   = dbgColor{0.9, 0.9, 0.9, 0.85}
	dbgColVNormal  = dbgColor{0.3, 0.5, 1, 1}
	dbgColFrustum  = dbgColor{1, 0.45, 0.8, 1}
	dbgColAudio    = dbgColor{0.4, 1, 0.9, 0.9}
	dbgColDefault  = dbgColor{1, 1, 1, 1}
)

// dbgCircleSegs is the segment count for every circle and arc.
const dbgCircleSegs = 24

// dbgFloatsPerVert is x,y,z,r,g,b,a.
const dbgFloatsPerVert = 7

// lineBatch collects GL_LINES vertices. max caps the line count so a huge
// scene cannot grow the buffer without bound; extra lines are dropped and
// counted.
type lineBatch struct {
	v       []float32
	max     int
	dropped int
}

func (b *lineBatch) reset() {
	b.v = b.v[:0]
	b.dropped = 0
}

func (b *lineBatch) lines() int { return len(b.v) / (2 * dbgFloatsPerVert) }

func (b *lineBatch) full() bool { return b.max > 0 && b.lines() >= b.max }

func (b *lineBatch) line(p, q math32.Vector3, c dbgColor) {
	if b.full() {
		b.dropped++
		return
	}
	b.v = append(b.v,
		p.X, p.Y, p.Z, c[0], c[1], c[2], c[3],
		q.X, q.Y, q.Z, c[0], c[1], c[2], c[3])
}

// xform is a rigid pose: origin plus orthonormal axes.
type xform struct {
	o, x, y, z math32.Vector3
}

func identityXform(o math32.Vector3) xform {
	return xform{o: o, x: math32.Vector3{X: 1}, y: math32.Vector3{Y: 1}, z: math32.Vector3{Z: 1}}
}

func quatXform(o math32.Vector3, q math32.Quaternion) xform {
	t := identityXform(o)
	t.x.ApplyQuaternion(&q)
	t.y.ApplyQuaternion(&q)
	t.z.ApplyQuaternion(&q)
	return t
}

// at maps a local point to world.
func (t xform) at(x, y, z float32) math32.Vector3 {
	return math32.Vector3{
		X: t.o.X + t.x.X*x + t.y.X*y + t.z.X*z,
		Y: t.o.Y + t.x.Y*x + t.y.Y*y + t.z.Y*z,
		Z: t.o.Z + t.x.Z*x + t.y.Z*y + t.z.Z*z,
	}
}

// wireBox draws 12 edges of a box with half extents h around t.
func (b *lineBatch) wireBox(t xform, hx, hy, hz float32, c dbgColor) {
	var p [8]math32.Vector3
	for i := 0; i < 8; i++ {
		sx, sy, sz := float32(-1), float32(-1), float32(-1)
		if i&1 != 0 {
			sx = 1
		}
		if i&2 != 0 {
			sy = 1
		}
		if i&4 != 0 {
			sz = 1
		}
		p[i] = t.at(sx*hx, sy*hy, sz*hz)
	}
	b.boxCorners(p, c)
}

// boxCorners draws the 12 edges of 8 corners indexed by bit (x=1, y=2, z=4).
func (b *lineBatch) boxCorners(p [8]math32.Vector3, c dbgColor) {
	for i := 0; i < 8; i++ {
		for _, bit := range [3]int{1, 2, 4} {
			if i&bit == 0 {
				b.line(p[i], p[i|bit], c)
			}
		}
	}
}

// circle draws a full circle of radius r in the plane spanned by u and v.
func (b *lineBatch) circle(center, u, v math32.Vector3, r float32, segs int, c dbgColor) {
	b.arc(center, u, v, r, 0, 2*math32.Pi, segs, c)
}

// arc draws segs segments from angle a0 to a1 (radians).
func (b *lineBatch) arc(center, u, v math32.Vector3, r, a0, a1 float32, segs int, c dbgColor) {
	if segs < 1 {
		segs = 1
	}
	pt := func(a float32) math32.Vector3 {
		cs, sn := math32.Cos(a)*r, math32.Sin(a)*r
		return math32.Vector3{
			X: center.X + u.X*cs + v.X*sn,
			Y: center.Y + u.Y*cs + v.Y*sn,
			Z: center.Z + u.Z*cs + v.Z*sn,
		}
	}
	prev := pt(a0)
	for i := 1; i <= segs; i++ {
		cur := pt(a0 + (a1-a0)*float32(i)/float32(segs))
		b.line(prev, cur, c)
		prev = cur
	}
}

// wireSphere draws three great circles: 3*segs lines.
func (b *lineBatch) wireSphere(t xform, r float32, segs int, c dbgColor) {
	b.circle(t.o, t.x, t.y, r, segs, c)
	b.circle(t.o, t.y, t.z, r, segs, c)
	b.circle(t.o, t.z, t.x, r, segs, c)
}

// wireCapsule is a Y-aligned capsule with shaft half-height halfH: two rings,
// four side lines and two half circles per cap in the XY and ZY planes.
// That is 2*segs + 4 + 4*(segs/2) lines.
func (b *lineBatch) wireCapsule(t xform, halfH, r float32, segs int, c dbgColor) {
	top, bot := t.at(0, halfH, 0), t.at(0, -halfH, 0)
	b.circle(top, t.x, t.z, r, segs, c)
	b.circle(bot, t.x, t.z, r, segs, c)
	for _, s := range [4][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		b.line(t.at(s[0]*r, halfH, s[1]*r), t.at(s[0]*r, -halfH, s[1]*r), c)
	}
	half := segs / 2
	neg := t.y
	neg.Negate()
	for _, side := range [2]math32.Vector3{t.x, t.z} {
		b.arc(top, side, t.y, r, 0, math32.Pi, half, c)
		b.arc(bot, side, neg, r, 0, math32.Pi, half, c)
	}
}

// wireCylinder is Y-aligned: two rings and four sides, 2*segs + 4 lines.
func (b *lineBatch) wireCylinder(t xform, halfH, r float32, segs int, c dbgColor) {
	b.circle(t.at(0, halfH, 0), t.x, t.z, r, segs, c)
	b.circle(t.at(0, -halfH, 0), t.x, t.z, r, segs, c)
	for _, s := range [4][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		b.line(t.at(s[0]*r, halfH, s[1]*r), t.at(s[0]*r, -halfH, s[1]*r), c)
	}
}

// basisFor returns two unit vectors perpendicular to d (unit) and each other.
func basisFor(d math32.Vector3) (math32.Vector3, math32.Vector3) {
	ref := math32.Vector3{Y: 1}
	if math32.Abs(d.Y) > 0.9 {
		ref = math32.Vector3{X: 1}
	}
	var u, v math32.Vector3
	u.CrossVectors(&d, &ref).Normalize()
	v.CrossVectors(&d, &u).Normalize()
	return u, v
}

// wireCone draws a cone from apex along dir (unit) to length, half angle in
// radians: the base ring plus four slant lines, segs + 4 lines.
func (b *lineBatch) wireCone(apex, dir math32.Vector3, length, halfAngle float32, segs int, c dbgColor) {
	u, v := basisFor(dir)
	r := length * math32.Tan(halfAngle)
	base := math32.Vector3{X: apex.X + dir.X*length, Y: apex.Y + dir.Y*length, Z: apex.Z + dir.Z*length}
	b.circle(base, u, v, r, segs, c)
	for _, s := range [4][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		p := math32.Vector3{
			X: base.X + (u.X*s[0]+v.X*s[1])*r,
			Y: base.Y + (u.Y*s[0]+v.Y*s[1])*r,
			Z: base.Z + (u.Z*s[0]+v.Z*s[1])*r,
		}
		b.line(apex, p, c)
	}
}

// arrow draws a shaft and a four-line head: 5 lines. A zero-length arrow
// draws nothing.
func (b *lineBatch) arrow(from, to math32.Vector3, c dbgColor) {
	d := to
	d.Sub(&from)
	l := d.Length()
	if l < 1e-5 {
		return
	}
	b.line(from, to, c)
	d.MultiplyScalar(1 / l)
	u, v := basisFor(d)
	head := l * 0.2
	if head > 0.4 {
		head = 0.4
	}
	back := math32.Vector3{X: to.X - d.X*head, Y: to.Y - d.Y*head, Z: to.Z - d.Z*head}
	w := head * 0.45
	for _, s := range [4][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		p := math32.Vector3{
			X: back.X + (u.X*s[0]+v.X*s[1])*w,
			Y: back.Y + (u.Y*s[0]+v.Y*s[1])*w,
			Z: back.Z + (u.Z*s[0]+v.Z*s[1])*w,
		}
		b.line(to, p, c)
	}
}

// cross draws three axis lines of half size s through p.
func (b *lineBatch) cross(p math32.Vector3, s float32, c dbgColor) {
	b.line(math32.Vector3{X: p.X - s, Y: p.Y, Z: p.Z}, math32.Vector3{X: p.X + s, Y: p.Y, Z: p.Z}, c)
	b.line(math32.Vector3{X: p.X, Y: p.Y - s, Z: p.Z}, math32.Vector3{X: p.X, Y: p.Y + s, Z: p.Z}, c)
	b.line(math32.Vector3{X: p.X, Y: p.Y, Z: p.Z - s}, math32.Vector3{X: p.X, Y: p.Y, Z: p.Z + s}, c)
}

// rect draws a rectangle from its centre and half-size edge vectors.
func (b *lineBatch) rect(center, right, up math32.Vector3, c dbgColor) {
	corner := func(sx, sy float32) math32.Vector3 {
		return math32.Vector3{
			X: center.X + right.X*sx + up.X*sy,
			Y: center.Y + right.Y*sx + up.Y*sy,
			Z: center.Z + right.Z*sx + up.Z*sy,
		}
	}
	p0, p1, p2, p3 := corner(-1, -1), corner(1, -1), corner(1, 1), corner(-1, 1)
	b.line(p0, p1, c)
	b.line(p1, p2, c)
	b.line(p2, p3, c)
	b.line(p3, p0, c)
}

// frustumCorners returns a perspective camera's 8 frustum corners in world
// space, near plane first (bit 4 clear), then far. The camera looks down
// its local -Z. fovDeg is vertical.
func frustumCorners(t xform, fovDeg, aspect, near, far float32) [8]math32.Vector3 {
	if aspect <= 0 {
		aspect = 1
	}
	tan := math32.Tan(fovDeg * math32.Pi / 360)
	var p [8]math32.Vector3
	for i := 0; i < 8; i++ {
		d := near
		if i&4 != 0 {
			d = far
		}
		hy := tan * d
		hx := hy * aspect
		sx, sy := float32(-1), float32(-1)
		if i&1 != 0 {
			sx = 1
		}
		if i&2 != 0 {
			sy = 1
		}
		p[i] = t.at(sx*hx, sy*hy, -d)
	}
	return p
}

// orthoCorners is frustumCorners for an orthographic camera of vertical
// view size.
func orthoCorners(t xform, size, aspect, near, far float32) [8]math32.Vector3 {
	if aspect <= 0 {
		aspect = 1
	}
	var p [8]math32.Vector3
	for i := 0; i < 8; i++ {
		d := near
		if i&4 != 0 {
			d = far
		}
		hy := size / 2
		hx := hy * aspect
		sx, sy := float32(-1), float32(-1)
		if i&1 != 0 {
			sx = 1
		}
		if i&2 != 0 {
			sy = 1
		}
		p[i] = t.at(sx*hx, sy*hy, -d)
	}
	return p
}

// wireFrustum draws the 12 frustum edges plus 4 lines from the eye to the
// near corners.
func (b *lineBatch) wireFrustum(eye math32.Vector3, p [8]math32.Vector3, c dbgColor) {
	b.boxCorners(p, c)
	for i := 0; i < 4; i++ {
		b.line(eye, p[i], c)
	}
}

// aabbCorners transforms the 8 corners of a local box by m.
func aabbCorners(min, max math32.Vector3, m *math32.Matrix4) [8]math32.Vector3 {
	var p [8]math32.Vector3
	for i := 0; i < 8; i++ {
		v := min
		if i&1 != 0 {
			v.X = max.X
		}
		if i&2 != 0 {
			v.Y = max.Y
		}
		if i&4 != 0 {
			v.Z = max.Z
		}
		if m != nil {
			v.ApplyMatrix4(m)
		}
		p[i] = v
	}
	return p
}
