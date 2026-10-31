package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arl/go-detour/detour"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/math32"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/phys3d"
	"moonbasic/internal/value"
)

// Debug overlay: one dynamic VBO of coloured GL_LINES per frame, drawn by a
// tiny shader after the scene (and the modern pass) and before post. See
// docs/DEBUG.md.

// Overlay bits. The low four are the physics view; SetPhysicsDebug takes them
// as a mask.
const (
	dbgShapes uint32 = 1 << iota
	dbgContacts
	dbgVelocity
	dbgJoints
	dbgRays
	dbgNav
	dbgLights
	dbgBounds
	dbgNormals
	dbgIDs
	dbgFrustums
	dbgAudio

	dbgPhysicsMask = dbgShapes | dbgContacts | dbgVelocity | dbgJoints
	dbgAll         = dbgAudio<<1 - 1
)

// dbgOverlayNames is DebugDraw's what$. Order is the doc order.
var dbgOverlayNames = []struct {
	name string
	bit  uint32
}{
	{"physics", dbgShapes},
	{"contacts", dbgContacts},
	{"velocity", dbgVelocity},
	{"joints", dbgJoints},
	{"rays", dbgRays},
	{"nav", dbgNav},
	{"lights", dbgLights},
	{"bounds", dbgBounds},
	{"normals", dbgNormals},
	{"ids", dbgIDs},
	{"frustums", dbgFrustums},
	{"audio", dbgAudio},
}

// dbgOverlayBit maps a name (or alias) to its bit; "all" is every overlay.
func dbgOverlayBit(name string) (uint32, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "all", "*":
		return dbgAll, true
	case "shapes", "colliders", "collision":
		return dbgShapes, true
	case "contact":
		return dbgContacts, true
	case "velocities":
		return dbgVelocity, true
	case "joint", "constraints":
		return dbgJoints, true
	case "ray", "raycasts", "picks":
		return dbgRays, true
	case "navmesh", "navigation":
		return dbgNav, true
	case "light":
		return dbgLights, true
	case "bbox", "aabb", "boxes":
		return dbgBounds, true
	case "normal":
		return dbgNormals, true
	case "id", "labels", "names":
		return dbgIDs, true
	case "frustum", "cameras":
		return dbgFrustums, true
	case "sound", "sounds":
		return dbgAudio, true
	}
	for _, o := range dbgOverlayNames {
		if o.name == name {
			return o.bit, true
		}
	}
	return 0, false
}

const (
	dbgMaxLines        = 100000
	dbgMaxUserLines    = 50000
	dbgMaxRays         = 64
	dbgMaxLabels       = 64
	dbgLabelRange      = 80
	dbgMaxNormals      = 20000
	dbgMaxNormalsMesh  = 4000
	dbgMaxMeshTris     = 6000
	dbgFrustumFar      = 30
	dbgDefaultRayLife  = 1.0
	dbgSoundLife       = 1.5
	dbgVelocityScale   = 0.25
	dbgContactNormal   = 0.5
	dbgRayNormalLength = 0.6
	dbgDepthPull       = 0.004
)

type dbgTimedLine struct {
	a, b math32.Vector3
	c    dbgColor
	life float32
	once bool
}

type dbgRay struct {
	o, e, hit, n math32.Vector3
	didHit, hasN bool
	life         float32
}

type dbgContact struct {
	p, n math32.Vector3
}

type dbgSound struct {
	p    math32.Vector3
	r    float32
	life float32
}

// dbgJointAnchor is the joint anchor in each body's local frame, captured
// the first frame the overlay sees the joint.
type dbgJointAnchor struct {
	la, lb math32.Vector3
	world  math32.Vector3
	hasA   bool
	hasB   bool
}

type debugDrawState struct {
	flags   uint32
	mode    int // 0 depth-tested, 1 on top, 2 depth-tested + faint hidden lines
	navOnly int
	rayLife float32

	batch    lineBatch
	user     []dbgTimedLine
	rays     []dbgRay
	contacts []dbgContact
	sounds   []dbgSound
	joints   map[int]*dbgJointAnchor
	chars    map[int][2]float32
	labels   map[int]*gui.Label
	navCache map[*detour.NavMesh][]math32.Vector3

	prog      uint32
	vao, vbo  uint32
	uView     int32
	uProj     int32
	uPull     int32
	uDim      int32
	vboBytes  int
	failed    bool
	lastLines int
	lastDrop  int
}

func (w *World) debugState() *debugDrawState {
	if w.dbg == nil {
		w.dbg = &debugDrawState{rayLife: dbgDefaultRayLife}
	}
	return w.dbg
}

func (s *debugDrawState) on(bit uint32) bool { return s != nil && s.flags&bit != 0 }

func (w *World) setDebugFlags(bits uint32, on bool) {
	s := w.debugState()
	if on {
		s.flags |= bits
	} else {
		s.flags &^= bits
	}
	w.physDebug = s.flags&dbgPhysicsMask != 0
	if s.flags&dbgContacts == 0 {
		s.contacts = s.contacts[:0]
	}
	if s.flags&dbgRays == 0 {
		s.rays = s.rays[:0]
	}
	if s.flags&dbgAudio == 0 {
		s.sounds = s.sounds[:0]
	}
	if s.flags&dbgIDs == 0 {
		w.dropDebugLabels()
	}
}

// setPhysicsDebugMask replaces the four physics bits with mask.
func (w *World) setPhysicsDebugMask(mask uint32) {
	w.setDebugFlags(dbgPhysicsMask, false)
	w.setDebugFlags(mask&dbgPhysicsMask, true)
}

// ---- script-side primitives ----

func dbgArgColor(a []value.Value, i int) dbgColor {
	if len(a) < i+3 {
		return dbgColDefault
	}
	c := rgb(argN(a, i, 1), argN(a, i+1, 1), argN(a, i+2, 1))
	return dbgColor{c.R, c.G, c.B, 1}
}

func dbgBlitz(a []value.Value, i int) math32.Vector3 {
	x, y, z := toG3N(float32(argN(a, i, 0)), float32(argN(a, i+1, 0)), float32(argN(a, i+2, 0)))
	return math32.Vector3{X: x, Y: y, Z: z}
}

// addUserLines queues the lines in b for seconds (0 = this frame only).
func (s *debugDrawState) addUserLines(b *lineBatch, seconds float32) {
	n := b.lines()
	for i := 0; i < n; i++ {
		if len(s.user) >= dbgMaxUserLines {
			return
		}
		o := i * 2 * dbgFloatsPerVert
		v := b.v[o : o+2*dbgFloatsPerVert]
		s.user = append(s.user, dbgTimedLine{
			a:    math32.Vector3{X: v[0], Y: v[1], Z: v[2]},
			b:    math32.Vector3{X: v[7], Y: v[8], Z: v[9]},
			c:    dbgColor{v[3], v[4], v[5], v[6]},
			life: seconds,
			once: seconds <= 0,
		})
	}
}

func (w *World) debugUserShape(seconds float32, build func(b *lineBatch)) {
	var b lineBatch
	build(&b)
	w.debugState().addUserLines(&b, seconds)
}

// ---- hooks called from the rest of the runtime ----

// debugNotePick records a pick ray in Blitz coordinates when the ray
// overlay is on. hitID 0 is a miss.
func (w *World) debugNotePick(ox, oy, oz, dx, dy, dz float32, hitID int, hx, hy, hz float32, nx, ny, nz float32, hasN bool) {
	s := w.dbg
	if !s.on(dbgRays) {
		return
	}
	if hitID != 0 && !hasN && w.phys3 != nil {
		if h, ok := w.phys3.RaycastDetail(ox, oy, oz, dx, dy, dz); ok && h.ID == hitID {
			nx, ny, nz, hasN = h.NX, h.NY, h.NZ, true
		}
	}
	g := func(x, y, z float32) math32.Vector3 {
		a, b, c := toG3N(x, y, z)
		return math32.Vector3{X: a, Y: b, Z: c}
	}
	r := dbgRay{
		o:      g(ox, oy, oz),
		e:      g(ox+dx, oy+dy, oz+dz),
		hit:    g(hx, hy, hz),
		n:      g(nx, ny, nz),
		didHit: hitID != 0,
		hasN:   hasN,
		life:   s.rayLife,
	}
	if len(s.rays) >= dbgMaxRays {
		copy(s.rays, s.rays[1:])
		s.rays = s.rays[:len(s.rays)-1]
	}
	s.rays = append(s.rays, r)
}

// debugNoteContacts keeps the latest step's contact points for the overlay.
func (w *World) debugNoteContacts(evs []phys3d.ContactEvent) {
	s := w.dbg
	if !s.on(dbgContacts) {
		return
	}
	s.contacts = s.contacts[:0]
	for _, ev := range evs {
		if ev.Kind == phys3d.ContactRemoved {
			continue
		}
		px, py, pz := toG3N(ev.X, ev.Y, ev.Z)
		nx, ny, nz := toG3N(ev.NX, ev.NY, ev.NZ)
		s.contacts = append(s.contacts, dbgContact{
			p: math32.Vector3{X: px, Y: py, Z: pz},
			n: math32.Vector3{X: nx, Y: ny, Z: nz},
		})
	}
}

// debugNoteSound marks an EmitSound position (Blitz coords) for a moment.
func (w *World) debugNoteSound(x, y, z, rng float32) {
	s := w.dbg
	if !s.on(dbgAudio) {
		return
	}
	gx, gy, gz := toG3N(x, y, z)
	if len(s.sounds) >= 64 {
		s.sounds = s.sounds[1:]
	}
	s.sounds = append(s.sounds, dbgSound{p: math32.Vector3{X: gx, Y: gy, Z: gz}, r: rng, life: dbgSoundLife})
}

// debugNoteCharacter remembers a CharacterVirtual's capsule size.
func (w *World) debugNoteCharacter(id int, height, radius float32) {
	s := w.debugState()
	if s.chars == nil {
		s.chars = map[int][2]float32{}
	}
	s.chars[id] = [2]float32{height, radius}
}

// tickDebugDraw runs once a frame before render: ages timed lines, captures
// joint anchors and places entity labels.
func (w *World) tickDebugDraw(dt float32) {
	s := w.dbg
	if s == nil {
		return
	}
	keep := s.user[:0]
	for _, l := range s.user {
		if l.once || l.life > 0 {
			if !l.once {
				l.life -= dt
			}
			keep = append(keep, l)
		}
	}
	s.user = keep
	rays := s.rays[:0]
	for _, r := range s.rays {
		if r.life > 0 {
			r.life -= dt
			rays = append(rays, r)
		}
	}
	s.rays = rays
	snd := s.sounds[:0]
	for _, so := range s.sounds {
		if so.life > 0 {
			so.life -= dt
			snd = append(snd, so)
		}
	}
	s.sounds = snd
	if s.on(dbgJoints) {
		w.captureJointAnchors()
	}
	if s.on(dbgIDs) {
		w.placeDebugLabels()
	}
}

func (s *debugDrawState) dropOnce() {
	keep := s.user[:0]
	for _, l := range s.user {
		if !l.once {
			keep = append(keep, l)
		}
	}
	s.user = keep
}

// ---- overlay collection ----

func (w *World) debugCamera() *camera.Camera {
	if w.cam != nil {
		return w.cam
	}
	for _, e := range w.ents {
		if e != nil && e.cam != nil {
			return e.cam
		}
	}
	return nil
}

// collectDebugLines fills b with every enabled overlay and the script's own
// lines.
func (w *World) collectDebugLines(b *lineBatch, cam *camera.Camera) {
	s := w.dbg
	if s.flags != 0 {
		w.refreshWorldMatrices()
	}
	if s.on(dbgShapes) || s.on(dbgVelocity) {
		w.debugPhysicsShapes(b)
	}
	if s.on(dbgContacts) {
		for _, c := range s.contacts {
			b.cross(c.p, 0.08, dbgColContact)
			to := c.n
			to.MultiplyScalar(dbgContactNormal).Add(&c.p)
			b.arrow(c.p, to, dbgColNormal)
		}
	}
	if s.on(dbgJoints) {
		w.debugJoints(b)
	}
	if s.on(dbgRays) {
		for _, r := range s.rays {
			debugRayLines(b, r)
		}
	}
	if s.on(dbgNav) {
		w.debugNav(b)
	}
	if s.on(dbgLights) {
		w.debugLights(b, cam)
	}
	if s.on(dbgBounds) {
		w.debugBounds(b)
	}
	if s.on(dbgNormals) {
		w.debugNormals(b)
	}
	if s.on(dbgFrustums) {
		w.debugFrustums(b, cam)
	}
	if s.on(dbgAudio) {
		for _, so := range s.sounds {
			t := identityXform(so.p)
			b.wireSphere(t, 0.3, 12, dbgColAudio)
			if so.r > 0 {
				faint := dbgColAudio
				faint[3] = 0.35
				b.circle(so.p, t.x, t.z, so.r, 48, faint)
			}
		}
	}
	for _, l := range s.user {
		b.line(l.a, l.b, l.c)
	}
}

func debugRayLines(b *lineBatch, r dbgRay) {
	if !r.didHit {
		b.line(r.o, r.e, dbgColRayMiss)
		return
	}
	b.line(r.o, r.hit, dbgColRayHit)
	faint := dbgColRayMiss
	faint[3] = 0.3
	b.line(r.hit, r.e, faint)
	b.cross(r.hit, 0.12, dbgColRayHit)
	if r.hasN {
		to := r.n
		to.MultiplyScalar(dbgRayNormalLength).Add(&r.hit)
		b.arrow(r.hit, to, dbgColNormal)
	}
}

// entityPose is an entity's world position and rotation without scale:
// collider sizes are absolute.
func entityPose(e *Entity) xform {
	n := e.node.GetNode()
	return quatXform(worldPos(n), worldQuat(n))
}

func (w *World) debugBodyColor(id int, e *Entity) dbgColor {
	switch {
	case e.hitbox || e.collKind == 5:
		return dbgColSensor
	case e.bodyType == 3 && e.collKind == 0:
		return dbgColChar
	case e.bodyType == 2:
		return dbgColStatic
	}
	if w.phys3 != nil && phys3d.IsSleeping(w.phys3, id) {
		return dbgColSleep
	}
	if e.bodyType == 3 {
		return dbgColKinem
	}
	return dbgColDynamic
}

// debugCharacterSize is the capsule height and radius for a character id.
func (w *World) debugCharacterSize(id int) (float32, float32) {
	if s := w.dbg; s != nil {
		if c, ok := s.chars[id]; ok {
			return c[0], c[1]
		}
	}
	if c := w.fpsControllers[id]; c != nil {
		return c.height, c.radius
	}
	if c := w.tpsControllers[id]; c != nil {
		return c.height, c.radius
	}
	if w.topDownControllers[id] != nil {
		return 1.8, 0.45
	}
	return 1.8, 0.4
}

func (w *World) debugPhysicsShapes(b *lineBatch) {
	s := w.dbg
	for id, e := range w.ents {
		if e == nil || e.node == nil || (e.bodyType == 0 && !e.hitbox) {
			continue
		}
		t := entityPose(e)
		if s.on(dbgShapes) {
			w.debugShape(b, id, e, t, w.debugBodyColor(id, e))
		}
		if s.on(dbgVelocity) && w.phys3 != nil && (e.bodyType == 1 || e.bodyType == 3) {
			if vx, vy, vz, ok := w.phys3.GetVelocity(id); ok {
				gx, gy, gz := toG3N(vx, vy, vz)
				v := math32.Vector3{X: gx, Y: gy, Z: gz}
				if v.Length() > 0.05 {
					v.MultiplyScalar(dbgVelocityScale).Add(&t.o)
					b.arrow(t.o, v, dbgColVelocity)
				}
			}
		}
	}
}

// debugShape draws one collider as its real shape.
func (w *World) debugShape(b *lineBatch, id int, e *Entity, t xform, c dbgColor) {
	segs := dbgCircleSegs
	switch {
	case e.bodyType == 3 && e.collKind == 0:
		h, r := w.debugCharacterSize(id)
		half := h*0.5 - r
		if half < 0.02 {
			half = 0.02
		}
		b.wireCapsule(identityXform(t.o), half, r, segs, c)
		return
	}
	switch e.collKind {
	case 2:
		r := e.radius
		if r <= 0 {
			r = 0.5
		}
		b.wireSphere(t, r, segs, c)
	case 3:
		r := e.radius
		if r <= 0 {
			r = 0.4
		}
		b.wireCapsule(t, e.boxY, r, segs, c)
	case 4:
		b.wireCylinder(t, e.boxY, e.boxX, segs, c)
	case 7, 8:
		tris := entityWorldTris(e)
		if len(tris) > dbgMaxMeshTris {
			tris = tris[:dbgMaxMeshTris]
		}
		for _, tr := range tris {
			b.line(tr[0], tr[1], c)
			b.line(tr[1], tr[2], c)
			b.line(tr[2], tr[0], c)
		}
		if len(tris) == 0 {
			hx, hy, hz := debugHalfExtents(e)
			b.wireBox(t, hx, hy, hz, c)
		}
	default:
		hx, hy, hz := debugHalfExtents(e)
		b.wireBox(t, hx, hy, hz, c)
	}
}

// debugHalfExtents is the box size the old AABB overlay used.
func debugHalfExtents(e *Entity) (float32, float32, float32) {
	pick := func(v float32) float32 {
		if v < 0.05 {
			v = e.radius
		}
		if v < 0.05 {
			v = 0.5
		}
		return v
	}
	return pick(e.boxX), pick(e.boxY), pick(e.boxZ)
}

// captureJointAnchors stores each new joint's anchor in its bodies' frames.
func (w *World) captureJointAnchors() {
	s := w.dbg
	if s.joints == nil {
		s.joints = map[int]*dbgJointAnchor{}
	}
	live := map[int]bool{}
	for _, j := range w.sceneJoints {
		live[j.id] = true
		if s.joints[j.id] != nil || len(j.v) < 3 {
			continue
		}
		ax, ay, az := toG3N(j.v[0], j.v[1], j.v[2])
		an := &dbgJointAnchor{world: math32.Vector3{X: ax, Y: ay, Z: az}}
		an.la, an.hasA = w.localToBody(j.a, an.world)
		an.lb, an.hasB = w.localToBody(j.b, an.world)
		s.joints[j.id] = an
	}
	for id := range s.joints {
		if !live[id] {
			delete(s.joints, id)
		}
	}
}

// localToBody expresses world point p in entity id's rigid frame.
func (w *World) localToBody(id int, p math32.Vector3) (math32.Vector3, bool) {
	e := w.ents[id]
	if id == 0 || e == nil || e.node == nil {
		return p, false
	}
	t := entityPose(e)
	d := p
	d.Sub(&t.o)
	return math32.Vector3{X: d.Dot(&t.x), Y: d.Dot(&t.y), Z: d.Dot(&t.z)}, true
}

func (w *World) debugJoints(b *lineBatch) {
	s := w.dbg
	for _, j := range w.sceneJoints {
		an := s.joints[j.id]
		if an == nil {
			continue
		}
		// Each side's view of the anchor. Body 0 is the world, where the
		// anchor stays put. A gap between the two is joint stretch.
		anchorA, anchorB := an.world, an.world
		if e := w.ents[j.a]; an.hasA && e != nil && e.node != nil {
			t := entityPose(e)
			anchorA = t.at(an.la.X, an.la.Y, an.la.Z)
			b.line(t.o, anchorA, dbgColJoint)
		}
		if e := w.ents[j.b]; an.hasB && e != nil && e.node != nil {
			t := entityPose(e)
			anchorB = t.at(an.lb.X, an.lb.Y, an.lb.Z)
			b.line(anchorB, t.o, dbgColJoint)
		}
		b.cross(anchorA, 0.12, dbgColJoint)
		if anchorA.DistanceTo(&anchorB) > 0.02 {
			b.line(anchorA, anchorB, dbgColContact)
		}
	}
}

func (w *World) debugNav(b *lineBatch) {
	s := w.dbg
	// A rebake makes a new *NavMesh; keeping only the live ones drops the old.
	cache := map[*detour.NavMesh][]math32.Vector3{}
	defer func() { s.navCache = cache }()
	for id, nm := range w.navs {
		if nm == nil || !nm.baked || nm.nav == nil || (s.navOnly != 0 && s.navOnly != id) {
			continue
		}
		segs, ok := s.navCache[nm.nav]
		if !ok {
			segs = navMeshEdges(nm.nav)
		}
		cache[nm.nav] = segs
		inner := dbgColNav
		inner[3] = 0.4
		for i := 0; i+2 < len(segs); i += 3 {
			c := dbgColNav
			if segs[i+2].X == 0 {
				c = inner
			}
			b.line(segs[i], segs[i+1], c)
		}
	}
	for _, ag := range w.agents {
		if ag == nil || (s.navOnly != 0 && ag.nav != s.navOnly) {
			continue
		}
		e := w.ents[ag.ent]
		if e == nil || e.node == nil {
			continue
		}
		prev := worldPos(e.node.GetNode())
		b.circle(prev, math32.Vector3{X: 1}, math32.Vector3{Z: 1}, ag.radius, 16, dbgColNavPath)
		if !ag.moving {
			continue
		}
		for i := ag.i; i < len(ag.path); i++ {
			p := ag.path[i]
			x, y, z := toG3N(p[0], p[1]+0.05, p[2])
			cur := math32.Vector3{X: x, Y: y, Z: z}
			b.line(prev, cur, dbgColNavPath)
			b.cross(cur, 0.1, dbgColNavPath)
			prev = cur
		}
	}
}

// navMeshEdges flattens a baked Detour mesh into edge triples: two G3N points
// lifted 5 cm, then a flag vector whose X is 1 for a border edge. Shared
// inner edges appear once.
func navMeshEdges(nav *detour.NavMesh) []math32.Vector3 {
	var out []math32.Vector3
	const lift = 0.05
	for ti := range nav.Tiles {
		tile := &nav.Tiles[ti]
		if tile.Header == nil {
			continue
		}
		for pi := range tile.Polys {
			p := &tile.Polys[pi]
			if p.Type() != 0 {
				continue // off-mesh connection
			}
			n := int(p.VertCount)
			for j := 0; j < n; j++ {
				nei := p.Neis[j]
				border := nei == 0 || nei&0x8000 != 0
				if !border && int(nei-1) < pi {
					continue
				}
				va, vb := int(p.Verts[j])*3, int(p.Verts[(j+1)%n])*3
				if vb+2 >= len(tile.Verts) || va+2 >= len(tile.Verts) {
					continue
				}
				ax, ay, az := toG3N(tile.Verts[va], tile.Verts[va+1]+lift, tile.Verts[va+2])
				bx, by, bz := toG3N(tile.Verts[vb], tile.Verts[vb+1]+lift, tile.Verts[vb+2])
				flag := math32.Vector3{}
				if border {
					flag.X = 1
				}
				out = append(out, math32.Vector3{X: ax, Y: ay, Z: az}, math32.Vector3{X: bx, Y: by, Z: bz}, flag)
			}
		}
	}
	return out
}

func (w *World) debugLights(b *lineBatch, cam *camera.Camera) {
	for _, e := range w.ents {
		if e == nil || e.node == nil || e.lgtKind == 0 {
			continue
		}
		c := dbgColLight
		if e.lgtDisabled || !e.node.GetNode().Visible() {
			c[3] = 0.3
		}
		p := entityWorldPos(e)
		switch e.lgtKind {
		case 1:
			// G3N's directional position points at the sun; light travels
			// the other way. Draw it a few metres ahead of the camera.
			d := p
			if d.Length() < 1e-4 {
				d = math32.Vector3{Y: 1}
			}
			d.Normalize().Negate()
			at := math32.Vector3{}
			if cam != nil {
				// Cameras look down local -Z.
				at = worldPos(cam.GetNode())
				fwd := math32.Vector3{Z: -8}
				q := worldQuat(cam.GetNode())
				fwd.ApplyQuaternion(&q)
				at.Add(&fwd)
			}
			from := d
			from.MultiplyScalar(-1.5).Add(&at)
			to := d
			to.MultiplyScalar(1.5).Add(&at)
			b.arrow(from, to, c)
			u, v := basisFor(d)
			for _, k := range []float32{-0.6, 0.6} {
				off := u
				off.MultiplyScalar(k)
				f2, t2 := from, to
				f2.Add(&off)
				t2.Add(&off)
				b.arrow(f2, t2, c)
				off = v
				off.MultiplyScalar(k)
				f2, t2 = from, to
				f2.Add(&off)
				t2.Add(&off)
				b.arrow(f2, t2, c)
			}
		case 2:
			b.cross(p, 0.15, c)
			b.wireSphere(identityXform(p), entityLightRange(e), 32, c)
		case 3:
			d := entityWorldDir(e)
			ang := math32.Acos(entitySpotCos(e))
			b.wireCone(p, d, entityLightRange(e), ang, 32, c)
		}
	}
	for id, al := range w.areaLights {
		if al == nil || al.e == nil || al.e.node == nil || w.ents[id] != al.e {
			continue
		}
		n := al.e.node.GetNode()
		q := worldQuat(n)
		var sc math32.Vector3
		n.WorldScale(&sc)
		right := math32.Vector3{X: -1}
		up := math32.Vector3{Y: 1}
		right.ApplyQuaternion(&q)
		up.ApplyQuaternion(&q)
		right.MultiplyScalar(al.width * 0.5 * sc.X)
		up.MultiplyScalar(al.height * 0.5 * sc.Y)
		p := worldPos(n)
		c := dbgColLight
		if al.intensity <= 0 || !n.Visible() {
			c[3] = 0.3
		}
		b.rect(p, right, up, c)
		var nrm math32.Vector3
		nrm.CrossVectors(&right, &up).Normalize()
		l := float32(1)
		if al.rng > 0 {
			l = al.rng
		}
		to := nrm
		to.MultiplyScalar(l).Add(&p)
		b.arrow(p, to, c)
		if al.twoSided {
			back := nrm
			back.MultiplyScalar(-l).Add(&p)
			b.arrow(p, back, c)
		}
	}
}

func (w *World) debugBounds(b *lineBatch) {
	for _, e := range w.ents {
		if e == nil || e.mesh == nil || e.sky || e.cam != nil || !e.node.GetNode().Visible() {
			continue
		}
		geom := e.mesh.GetGeometry()
		if geom == nil {
			continue
		}
		bb := geom.BoundingBox()
		m := e.mesh.GetNode().MatrixWorld()
		b.boxCorners(aabbCorners(bb.Min, bb.Max, &m), dbgColBounds)
	}
}

func (w *World) debugNormals(b *lineBatch) {
	total := 0
	for _, e := range w.ents {
		if total >= dbgMaxNormals {
			return
		}
		if e == nil || e.mesh == nil || e.sky || !e.node.GetNode().Visible() {
			continue
		}
		geom := e.mesh.GetGeometry()
		if geom == nil {
			continue
		}
		m := e.mesh.GetNode().MatrixWorld()
		bb := geom.BoundingBox()
		size := bb.Max
		size.Sub(&bb.Min)
		l := size.Length() * 0.06
		if l < 0.05 {
			l = 0.05
		}
		if l > 0.4 {
			l = 0.4
		}
		budget := dbgMaxNormalsMesh
		if left := dbgMaxNormals - total; left < budget {
			budget = left
		}
		total += debugMeshNormals(b, geom.ReadVertices, geom.ReadVertexNormals, &m, l, budget)
	}
}

// debugMeshNormals draws up to budget vertex normals of length l and
// returns how many it drew. read and readN walk positions and normals in
// vertex order.
func debugMeshNormals(b *lineBatch, read, readN func(func(math32.Vector3) bool), m *math32.Matrix4, l float32, budget int) int {
	pos := make([]math32.Vector3, 0, 256)
	read(func(v math32.Vector3) bool {
		pos = append(pos, v)
		return len(pos) >= budget
	})
	i := 0
	readN(func(n math32.Vector3) bool {
		if i >= len(pos) {
			return true
		}
		p := pos[i]
		tip := p
		tip.Add(&n)
		p.ApplyMatrix4(m)
		tip.ApplyMatrix4(m)
		tip.Sub(&p)
		if tip.Length() < 1e-6 {
			i++
			return i >= len(pos)
		}
		tip.Normalize().MultiplyScalar(l).Add(&p)
		b.line(p, tip, dbgColVNormal)
		i++
		return i >= len(pos)
	})
	return i
}

func (w *World) debugFrustums(b *lineBatch, active *camera.Camera) {
	for _, e := range w.ents {
		if e == nil || e.cam == nil || e.cam == active {
			continue
		}
		c := e.cam
		n := c.GetNode()
		t := quatXform(worldPos(n), worldQuat(n))
		far := c.Far()
		if far > dbgFrustumFar {
			far = dbgFrustumFar
		}
		near := c.Near()
		aspect := c.Aspect()
		if e.vpW > 0 && e.vpH > 0 {
			aspect = float32(e.vpW) / float32(e.vpH)
		}
		var p [8]math32.Vector3
		if c.Projection() == camera.Orthographic {
			p = orthoCorners(t, c.Size(), aspect, near, far)
		} else {
			p = frustumCorners(t, c.Fov(), aspect, near, far)
		}
		b.wireFrustum(t.o, p, dbgColFrustum)
	}
}

// ---- labels ----

func (w *World) dropDebugLabels() {
	s := w.dbg
	if s == nil {
		return
	}
	for id, l := range s.labels {
		if p := l.Parent(); p != nil {
			p.GetNode().Remove(l)
		}
		delete(s.labels, id)
	}
}

// debugLabelText is "#id name", or "#id kind" when the entity has no name.
func debugLabelText(id int, e *Entity) string {
	name := e.name
	if name == "" {
		name = e.kind
	}
	if name == "" {
		switch e.lgtKind {
		case 2:
			name = "point light"
		case 3:
			name = "spot light"
		}
	}
	if name == "" {
		return fmt.Sprintf("#%d", id)
	}
	return fmt.Sprintf("#%d %s", id, name)
}

func (w *World) placeDebugLabels() {
	s := w.dbg
	cam := w.debugCamera()
	if cam == nil || w.scene == nil {
		return
	}
	if s.labels == nil {
		s.labels = map[int]*gui.Label{}
	}
	sw, sh := float32(w.scrW), float32(w.scrH)
	if sw < 1 || sh < 1 {
		sw, sh = 800, 600
	}
	w.refreshWorldMatrices()
	eye := worldPos(cam.GetNode())
	type cand struct {
		id     int
		e      *Entity
		d      float32
		sx, sy float32
	}
	var list []cand
	for id, e := range w.ents {
		// A directional light's position is only its direction.
		if e == nil || e.node == nil || e.cam != nil || e.sky || e.lgtKind == 1 || !e.node.GetNode().Visible() {
			continue
		}
		p := worldPos(e.node.GetNode())
		d := p.DistanceTo(&eye)
		if d > dbgLabelRange {
			continue
		}
		q := p
		cam.Project(&q)
		if q.Z < -1 || q.Z > 1 || q.X < -1.1 || q.X > 1.1 || q.Y < -1.1 || q.Y > 1.1 {
			continue
		}
		list = append(list, cand{id, e, d, (q.X + 1) * 0.5 * sw, (1 - q.Y) * 0.5 * sh})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].d < list[j].d })
	if len(list) > dbgMaxLabels {
		list = list[:dbgMaxLabels]
	}
	shown := map[int]bool{}
	for _, c := range list {
		shown[c.id] = true
		txt := debugLabelText(c.id, c.e)
		if c.e.name == "" && w.areaLights[c.id] != nil {
			txt = fmt.Sprintf("#%d area light", c.id)
		}
		l := s.labels[c.id]
		if l == nil {
			l = gui.NewLabel(txt)
			l.SetColor(&math32.Color{R: 1, G: 1, B: 0.55})
			l.SetBgColor4(&math32.Color4{R: 0, G: 0, B: 0, A: 0.55})
			l.SetPaddings(1, 3, 1, 3)
			s.labels[c.id] = l
		} else if l.Text() != txt {
			l.SetText(txt)
		}
		if l.Parent() == nil {
			w.scene.Add(l)
		}
		l.SetPosition(c.sx+6, c.sy-8)
		l.SetVisible(true)
	}
	for id, l := range s.labels {
		if shown[id] {
			continue
		}
		if w.ents[id] == nil {
			if p := l.Parent(); p != nil {
				p.GetNode().Remove(l)
			}
			delete(s.labels, id)
			continue
		}
		l.SetVisible(false)
	}
}

// ---- GL ----

func (s *debugDrawState) ensureProgram() bool {
	if s.prog != 0 {
		return true
	}
	if s.failed {
		return false
	}
	p, err := compileLink(
		struct {
			kind uint32
			src  string
		}{gl.VERTEX_SHADER, mbdebugVertex},
		struct {
			kind uint32
			src  string
		}{gl.FRAGMENT_SHADER, mbdebugFragment},
	)
	if err != nil {
		fmt.Println("DebugDraw:", err)
		s.failed = true
		return false
	}
	s.prog = p
	s.uView = gl.GetUniformLocation(p, gl.Str("View\x00"))
	s.uProj = gl.GetUniformLocation(p, gl.Str("Proj\x00"))
	s.uPull = gl.GetUniformLocation(p, gl.Str("Pull\x00"))
	s.uDim = gl.GetUniformLocation(p, gl.Str("Dim\x00"))
	gl.GenVertexArrays(1, &s.vao)
	gl.GenBuffers(1, &s.vbo)
	gl.BindVertexArray(s.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	stride := int32(dbgFloatsPerVert * 4)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 4, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.BindVertexArray(0)
	drainGL("DebugDraw init")
	return true
}

// debugPasses lists (depth func, dim) for a mode. 0 means no depth test.
func debugPasses(mode int) [][2]float32 {
	switch mode {
	case 1:
		return [][2]float32{{0, 1}}
	case 2:
		return [][2]float32{{gl.GREATER, 0.25}, {gl.LEQUAL, 1}}
	}
	return [][2]float32{{gl.LEQUAL, 1}}
}

// drawDebugOverlay is render()'s hook, called right after the scene and
// modern pass while the scene framebuffer (post FBO or window) and its depth
// are still bound, so lines are depth-tested against the real scene. Post
// then tonemaps them with everything else.
func (w *World) drawDebugOverlay(vw, vh int) {
	s := w.dbg
	if s == nil || w.app == nil {
		return
	}
	defer s.dropOnce()
	if s.flags == 0 && len(s.user) == 0 {
		s.lastLines = 0
		return
	}
	cam := w.debugCamera()
	if cam == nil || vw <= 0 || vh <= 0 {
		return
	}
	b := &s.batch
	b.max = dbgMaxLines
	b.reset()
	w.collectDebugLines(b, cam)
	s.lastLines, s.lastDrop = b.lines(), b.dropped
	if b.lines() == 0 {
		return
	}
	w.detectModernGL()
	if !s.ensureProgram() {
		return
	}
	gs := w.app.Gls()
	restore := saveModernGL(gs)
	defer restore()
	drainGL("before DebugDraw")

	var view, proj math32.Matrix4
	cam.ViewMatrix(&view)
	cam.ProjMatrix(&proj)

	gl.Viewport(0, 0, int32(vw), int32(vh))
	gl.UseProgram(s.prog)
	gl.UniformMatrix4fv(s.uView, 1, false, &view[0])
	gl.UniformMatrix4fv(s.uProj, 1, false, &proj[0])
	gl.Uniform1f(s.uPull, dbgDepthPull)
	gl.BindVertexArray(s.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	bytes := len(b.v) * 4
	if bytes > s.vboBytes {
		s.vboBytes = bytes + bytes/2
	}
	// Orphan, then fill: the driver need not wait on last frame's draw.
	gl.BufferData(gl.ARRAY_BUFFER, s.vboBytes, nil, gl.STREAM_DRAW)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, bytes, gl.Ptr(b.v))

	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.SCISSOR_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.DepthMask(false)
	passes := debugPasses(s.mode)
	for _, p := range passes {
		if p[0] == 0 {
			gl.Disable(gl.DEPTH_TEST)
		} else {
			gl.Enable(gl.DEPTH_TEST)
			gl.DepthFunc(uint32(p[0]))
		}
		gl.Uniform1f(s.uDim, p[1])
		gl.DrawArrays(gl.LINES, 0, int32(b.lines()*2))
	}
	w.debugRestoreGL()
	drainGL("DebugDraw")
}

// debugRestoreGL puts back the state G3N expects and resyncs its cache,
// like guiRestoreSceneGL but leaving the bound framebuffer alone.
func (w *World) debugRestoreGL() {
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthMask(true)
	gl.DepthFunc(gl.LEQUAL)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gs := w.app.Gls()
	if gs == nil {
		return
	}
	gs.Disable(gls.DEPTH_TEST)
	gs.Enable(gls.DEPTH_TEST)
	gs.Disable(gls.CULL_FACE)
	gs.Enable(gls.CULL_FACE)
	gs.Disable(gls.BLEND)
	gs.Enable(gls.BLEND)
	gs.DepthMask(false)
	gs.DepthMask(true)
}
