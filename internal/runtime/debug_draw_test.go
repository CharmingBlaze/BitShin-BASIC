package runtime

import (
	"strings"
	"testing"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/phys3d"
	"moonbasic/internal/value"
)

func TestDebugShapeLineCounts(t *testing.T) {
	const s = dbgCircleSegs
	o := identityXform(math32.Vector3{})
	c := dbgColDefault
	cases := []struct {
		name string
		want int
		draw func(b *lineBatch)
	}{
		{"box", 12, func(b *lineBatch) { b.wireBox(o, 1, 2, 3, c) }},
		{"sphere", 3 * s, func(b *lineBatch) { b.wireSphere(o, 1, s, c) }},
		{"capsule", 2*s + 4 + 4*(s/2), func(b *lineBatch) { b.wireCapsule(o, 0.5, 0.4, s, c) }},
		{"cylinder", 2*s + 4, func(b *lineBatch) { b.wireCylinder(o, 0.5, 0.4, s, c) }},
		{"cone", 32 + 4, func(b *lineBatch) { b.wireCone(math32.Vector3{}, math32.Vector3{Y: -1}, 5, 0.5, 32, c) }},
		{"arrow", 5, func(b *lineBatch) { b.arrow(math32.Vector3{}, math32.Vector3{X: 1}, c) }},
		{"cross", 3, func(b *lineBatch) { b.cross(math32.Vector3{}, 0.1, c) }},
		{"rect", 4, func(b *lineBatch) { b.rect(math32.Vector3{}, math32.Vector3{X: 1}, math32.Vector3{Y: 1}, c) }},
		{"frustum", 16, func(b *lineBatch) {
			b.wireFrustum(math32.Vector3{}, frustumCorners(o, 60, 1.5, 0.1, 10), c)
		}},
	}
	for _, tc := range cases {
		var b lineBatch
		tc.draw(&b)
		if got := b.lines(); got != tc.want {
			t.Errorf("%s: %d lines, want %d", tc.name, got, tc.want)
		}
		if len(b.v) != tc.want*2*dbgFloatsPerVert {
			t.Errorf("%s: %d floats, want %d", tc.name, len(b.v), tc.want*2*dbgFloatsPerVert)
		}
	}
	var b lineBatch
	b.arrow(math32.Vector3{X: 1}, math32.Vector3{X: 1}, c)
	if b.lines() != 0 {
		t.Fatalf("zero-length arrow drew %d lines", b.lines())
	}
}

func TestDebugCapsuleExtents(t *testing.T) {
	var b lineBatch
	b.wireCapsule(identityXform(math32.Vector3{Y: 2}), 0.5, 0.4, dbgCircleSegs, dbgColDefault)
	minY, maxY, maxR := float32(1e9), float32(-1e9), float32(0)
	for i := 0; i < len(b.v); i += dbgFloatsPerVert {
		x, y, z := b.v[i], b.v[i+1], b.v[i+2]
		minY, maxY = math32.Min(minY, y), math32.Max(maxY, y)
		maxR = math32.Max(maxR, math32.Sqrt(x*x+z*z))
	}
	if math32.Abs(maxY-2.9) > 1e-3 || math32.Abs(minY-1.1) > 1e-3 {
		t.Fatalf("capsule y range %.3f..%.3f, want 1.1..2.9", minY, maxY)
	}
	if math32.Abs(maxR-0.4) > 1e-3 {
		t.Fatalf("capsule radius %.3f, want 0.4", maxR)
	}
}

func TestDebugSphereOnRadius(t *testing.T) {
	var b lineBatch
	c := math32.Vector3{X: 1, Y: 2, Z: 3}
	b.wireSphere(quatXform(c, *math32.NewQuaternion(0.2, 0.3, 0.1, 0.9).Normalize()), 2.5, dbgCircleSegs, dbgColDefault)
	for i := 0; i < len(b.v); i += dbgFloatsPerVert {
		p := math32.Vector3{X: b.v[i], Y: b.v[i+1], Z: b.v[i+2]}
		if d := p.DistanceTo(&c); math32.Abs(d-2.5) > 1e-3 {
			t.Fatalf("sphere vertex %v at distance %.4f, want 2.5", p, d)
		}
	}
}

func TestFrustumCorners(t *testing.T) {
	p := frustumCorners(identityXform(math32.Vector3{}), 90, 2, 1, 10)
	for i, v := range p {
		d, hx := float32(1), float32(2)
		if i&4 != 0 {
			d, hx = 10, 20
		}
		if math32.Abs(v.Z+d) > 1e-4 || math32.Abs(math32.Abs(v.Y)-d) > 1e-3 || math32.Abs(math32.Abs(v.X)-hx) > 1e-3 {
			t.Fatalf("corner %d = %v (near/far %v)", i, v, d)
		}
	}
	// A camera turned to face +X puts the far plane at x = +far.
	q := math32.NewQuaternion(0, 0, 0, 1)
	q.SetFromAxisAngle(&math32.Vector3{Y: 1}, -math32.Pi/2)
	p = frustumCorners(quatXform(math32.Vector3{X: 5}, *q), 60, 1, 0.5, 4)
	for i := 4; i < 8; i++ {
		if math32.Abs(p[i].X-9) > 1e-3 {
			t.Fatalf("turned far corner %d = %v, want x=9", i, p[i])
		}
	}
	o := orthoCorners(identityXform(math32.Vector3{}), 4, 1, 1, 3)
	if math32.Abs(o[7].X-2) > 1e-4 || math32.Abs(o[7].Y-2) > 1e-4 || math32.Abs(o[7].Z+3) > 1e-4 {
		t.Fatalf("ortho corner = %v", o[7])
	}
}

func TestDebugBatchCap(t *testing.T) {
	b := lineBatch{max: 10}
	for i := 0; i < 25; i++ {
		b.line(math32.Vector3{}, math32.Vector3{X: float32(i)}, dbgColDefault)
	}
	if b.lines() != 10 || b.dropped != 15 {
		t.Fatalf("lines %d dropped %d, want 10 and 15", b.lines(), b.dropped)
	}
	b.reset()
	if b.lines() != 0 || b.dropped != 0 {
		t.Fatal("reset left lines behind")
	}
}

func TestDebugOverlayNames(t *testing.T) {
	for _, o := range dbgOverlayNames {
		bit, ok := dbgOverlayBit(strings.ToUpper(o.name))
		if !ok || bit != o.bit {
			t.Errorf("%s -> %d %v", o.name, bit, ok)
		}
	}
	if bit, _ := dbgOverlayBit("all"); bit != dbgAll {
		t.Fatalf("all = %b", bit)
	}
	if _, ok := dbgOverlayBit("nope"); ok {
		t.Fatal("unknown overlay accepted")
	}
}

func debugTestWorld() *World {
	w := New(".")
	w.scene = core.NewNode()
	w.delta = 1.0 / 60.0
	return w
}

func dbgCall(t *testing.T, w *World, name string, args ...value.Value) value.Value {
	t.Helper()
	v, err := w.Call(name, args)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return v
}

func TestDebugDrawCommandWiring(t *testing.T) {
	names := map[string]bool{}
	for _, n := range CommandNames() {
		names[strings.ToLower(n)] = true
	}
	for _, n := range []string{"debugdraw", "debugdrawenabled", "debugdrawontop", "debugdrawnavmesh", "debugraytime",
		"setphysicsdebug", "getphysicsdebug", "debugline", "debugarrow", "debugsphere", "debugbox", "debugclear",
		"debugdrawlines", "enablephysicsdebug"} {
		if !names[n] {
			t.Errorf("%s is not registered", n)
		}
	}
	w := debugTestWorld()
	dbgCall(t, w, "debugdraw", value.Str("lights"), value.Num(1))
	if dbgCall(t, w, "debugdrawenabled", value.Str("lights")).Int() != 1 {
		t.Fatal("lights not enabled")
	}
	if _, err := w.Call("debugdraw", []value.Value{value.Str("bogus"), value.Num(1)}); err == nil {
		t.Fatal("unknown overlay should error")
	}
	dbgCall(t, w, "enablephysicsdebug", value.Num(1))
	if w.dbg.flags&dbgPhysicsMask != dbgPhysicsMask || !w.physDebug {
		t.Fatalf("EnablePhysicsDebug 1 flags %b", w.dbg.flags)
	}
	dbgCall(t, w, "setphysicsdebug", value.Num(5))
	if got := dbgCall(t, w, "getphysicsdebug").Int(); got != 5 {
		t.Fatalf("GetPhysicsDebug = %d, want 5", got)
	}
	if w.dbg.flags&dbgLights == 0 {
		t.Fatal("SetPhysicsDebug cleared a non-physics overlay")
	}
	dbgCall(t, w, "enablephysicsdebug", value.Num(0))
	if w.dbg.flags&dbgPhysicsMask != 0 || w.physDebug {
		t.Fatal("EnablePhysicsDebug 0 left physics bits")
	}
	dbgCall(t, w, "debugdraw", value.Str("all"), value.Num(0))
	if w.dbg.flags != 0 {
		t.Fatalf("all off left %b", w.dbg.flags)
	}
	dbgCall(t, w, "debugdrawontop", value.Num(2))
	if w.dbg.mode != 2 || len(debugPasses(2)) != 2 || len(debugPasses(0)) != 1 {
		t.Fatal("x-ray mode should draw two passes")
	}
}

func TestDebugUserLinesLifetime(t *testing.T) {
	w := debugTestWorld()
	cam := camera.NewPerspective(1, 0.1, 100, 60, camera.Vertical)
	n := func(v ...float64) []value.Value {
		out := make([]value.Value, len(v))
		for i, x := range v {
			out[i] = value.Num(x)
		}
		return out
	}
	// One-frame line, then a 1 s box.
	if _, err := w.Call("debugline", n(0, 0, 0, 1, 2, 3)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("debugbox", n(0, 0, 0, 1, 1, 1, 255, 0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		var b lineBatch
		w.collectDebugLines(&b, cam)
		return b.lines()
	}
	w.tickDebugDraw(0.016)
	if got := count(); got != 13 {
		t.Fatalf("frame 1: %d lines, want 13", got)
	}
	w.dbg.dropOnce()
	if got := count(); got != 12 {
		t.Fatalf("after frame: %d lines, want the 12 box edges", got)
	}
	// Script Z is flipped into G3N space.
	if l := w.dbg.user[0]; l.c != (dbgColor{1, 0, 0, 1}) {
		t.Fatalf("box colour %v", l.c)
	}
	w.tickDebugDraw(0.6)
	w.tickDebugDraw(0.6)
	if got := count(); got != 12 {
		t.Fatalf("box should draw through its last frame, got %d", got)
	}
	w.tickDebugDraw(0.016)
	if got := count(); got != 0 {
		t.Fatalf("expired box still drawn: %d", got)
	}
	dbgCall(t, w, "debugline", n(0, 0, 5, 0, 0, 6)...)
	if l := w.dbg.user[0]; l.a.Z != -5 || l.b.Z != -6 {
		t.Fatalf("DebugLine z not converted: %v %v", l.a, l.b)
	}
	dbgCall(t, w, "debugclear")
	if len(w.dbg.user) != 0 {
		t.Fatal("DebugClear left lines")
	}
}

func TestDebugPhysicsShapesAndColours(t *testing.T) {
	w := debugTestWorld()
	w.setDebugFlags(dbgShapes, true)
	mat := material.NewStandard(&math32.Color{R: 1})
	add := func(kind, body int, rad, bx, by, bz float32) int {
		m := graphic.NewMesh(geometry.NewCube(1), mat)
		e := &Entity{node: m, mesh: m, mat: mat, collKind: kind, bodyType: body, radius: rad, boxX: bx, boxY: by, boxZ: bz}
		return w.addEntity(e, 0)
	}
	add(2, 1, 1, 0, 0, 0)       // sphere
	add(1, 2, 1, 1, 1, 1)       // static box
	add(3, 1, 0.4, 0, 0.5, 0)   // capsule
	add(4, 1, 0, 0.4, 0.6, 0)   // cylinder
	id := add(5, 3, 0, 1, 1, 1) // sensor
	w.ents[id].hitbox = true
	ch := add(0, 3, 0, 0, 0, 0) // character
	w.debugNoteCharacter(ch, 1.8, 0.4)
	var b lineBatch
	w.debugPhysicsShapes(&b)
	s := dbgCircleSegs
	want := 3*s + 12 + (2*s + 4 + 4*(s/2)) + (2*s + 4) + 12 + (2*s + 4 + 4*(s/2))
	if b.lines() != want {
		t.Fatalf("physics shapes %d lines, want %d", b.lines(), want)
	}
	if c := w.debugBodyColor(id, w.ents[id]); c != dbgColSensor {
		t.Fatalf("sensor colour %v", c)
	}
	if c := w.debugBodyColor(ch, w.ents[ch]); c != dbgColChar {
		t.Fatalf("character colour %v", c)
	}
}

func TestDebugLightsAndFrustums(t *testing.T) {
	w := debugTestWorld()
	w.makeLight(2, 0)
	sp := w.makeLight(3, 0)
	w.ents[sp].lgtRange = 6
	cam := camera.NewPerspective(1, 0.1, 100, 60, camera.Vertical)
	var b lineBatch
	w.debugLights(&b, cam)
	if want := 3 + 3*32 + 32 + 4; b.lines() != want {
		t.Fatalf("point+spot %d lines, want %d", b.lines(), want)
	}
	other := camera.NewPerspective(1.5, 0.1, 500, 60, camera.Vertical)
	w.addEntity(&Entity{node: other, cam: other}, 0)
	w.addEntity(&Entity{node: cam, cam: cam}, 0)
	b.reset()
	w.debugFrustums(&b, cam)
	if b.lines() != 16 {
		t.Fatalf("one inactive camera should give 16 frustum lines, got %d", b.lines())
	}
	// Far is capped so a 500 m camera does not draw across the map.
	for i := 0; i < len(b.v); i += dbgFloatsPerVert {
		if b.v[i+2] < -dbgFrustumFar-1e-3 {
			t.Fatalf("frustum reaches z=%.1f past the %d m cap", b.v[i+2], dbgFrustumFar)
		}
	}
}

func TestDebugRaysFromLinePick(t *testing.T) {
	w := debugTestWorld()
	w.setDebugFlags(dbgRays, true)
	mat := material.NewStandard(&math32.Color{R: 1})
	m := graphic.NewMesh(geometry.NewCube(1), mat)
	m.SetPosition(0, 0, -5) // Blitz z = 5
	w.addEntity(&Entity{node: m, mesh: m, mat: mat, radius: 1}, 0)
	w.refreshWorldMatrices()
	if _, err := w.linePick(0, 0, 0, 0, 0, 10, 0); err != nil {
		t.Fatal(err)
	}
	if len(w.dbg.rays) != 1 || !w.dbg.rays[0].didHit {
		t.Fatalf("rays %+v", w.dbg.rays)
	}
	if _, err := w.linePick(0, 10, 0, 0, 0, 10, 0); err != nil {
		t.Fatal(err)
	}
	var b lineBatch
	for _, r := range w.dbg.rays {
		debugRayLines(&b, r)
	}
	// Hit: segment, faint tail, cross (3). Miss: one line.
	if b.lines() != 5+1 {
		t.Fatalf("ray lines %d, want 6", b.lines())
	}
	w.dbg.rayLife = 0.1
	w.tickDebugDraw(1)
	w.tickDebugDraw(1)
	if len(w.dbg.rays) != 0 {
		t.Fatal("rays did not expire")
	}
}

func TestDebugNavMeshEdges(t *testing.T) {
	w := debugTestWorld()
	mat := material.NewStandard(&math32.Color{R: 0.5, G: 0.5, B: 0.5})
	floor := graphic.NewMesh(geometry.NewCube(2), mat)
	floor.SetScale(8, 0.2, 8)
	fid := w.addEntity(&Entity{node: floor, mesh: floor, mat: mat}, 0)
	nm := &navMesh{meshID: fid}
	if err := w.bakeNav(nm); err != nil {
		t.Fatal(err)
	}
	w.navs[1] = nm
	segs := navMeshEdges(nm.nav)
	if len(segs) == 0 || len(segs)%3 != 0 {
		t.Fatalf("nav edges %d", len(segs))
	}
	border := 0
	for i := 2; i < len(segs); i += 3 {
		if segs[i].X == 1 {
			border++
		}
	}
	if border < 4 {
		t.Fatalf("a square floor should have at least 4 border edges, got %d", border)
	}
	w.setDebugFlags(dbgNav, true)
	var b lineBatch
	w.debugNav(&b)
	if b.lines() != len(segs)/3 {
		t.Fatalf("nav overlay %d lines, want %d", b.lines(), len(segs)/3)
	}
	w.debugState().navOnly = 2
	b.reset()
	w.debugNav(&b)
	if b.lines() != 0 {
		t.Fatal("DebugDrawNavMesh 2 should skip nav 1")
	}
}

func TestDebugBoundsAndNormals(t *testing.T) {
	w := debugTestWorld()
	mat := material.NewStandard(&math32.Color{R: 1})
	m := graphic.NewMesh(geometry.NewCube(1), mat)
	w.addEntity(&Entity{node: m, mesh: m, mat: mat}, 0)
	w.refreshWorldMatrices()
	var b lineBatch
	w.debugBounds(&b)
	if b.lines() != 12 {
		t.Fatalf("bounds %d lines", b.lines())
	}
	b.reset()
	w.debugNormals(&b)
	if b.lines() != 24 { // a G3N cube has 24 vertices
		t.Fatalf("normals %d lines, want 24", b.lines())
	}
	b.reset()
	geom := m.GetGeometry()
	var ident math32.Matrix4
	ident.Identity()
	if got := debugMeshNormals(&b, geom.ReadVertices, geom.ReadVertexNormals, &ident, 0.1, 5); got != 5 || b.lines() != 5 {
		t.Fatalf("budget 5 drew %d (%d lines)", got, b.lines())
	}
}

func TestDebugContactsFromEvents(t *testing.T) {
	w := debugTestWorld()
	evs := []phys3d.ContactEvent{
		{Kind: phys3d.ContactAdded, A: 1, B: 2, X: 1, Y: 0, Z: 2, NX: 0, NY: 1, NZ: 0},
		{Kind: phys3d.ContactRemoved, A: 1, B: 3},
	}
	w.debugNoteContacts(evs)
	if w.dbg != nil && len(w.dbg.contacts) != 0 {
		t.Fatal("contacts recorded while the overlay is off")
	}
	w.setDebugFlags(dbgContacts, true)
	w.debugNoteContacts(evs)
	if len(w.dbg.contacts) != 1 {
		t.Fatalf("contacts %d, want 1 (removed events skipped)", len(w.dbg.contacts))
	}
	if c := w.dbg.contacts[0]; c.p.Z != -2 || c.n.Y != 1 {
		t.Fatalf("contact not in G3N space: %+v", c)
	}
	var b lineBatch
	w.collectDebugLines(&b, nil)
	if b.lines() != 3+5 { // cross + normal arrow
		t.Fatalf("contact lines %d, want 8", b.lines())
	}
}

func TestDebugJointAnchorFollowsBody(t *testing.T) {
	w := debugTestWorld()
	w.setDebugFlags(dbgJoints, true)
	mat := material.NewStandard(&math32.Color{R: 1})
	m := graphic.NewMesh(geometry.NewSphere(0.5, 8, 8), mat)
	m.SetPosition(0, 2, 0)
	id := w.addEntity(&Entity{node: m, mesh: m, mat: mat, bodyType: 1, collKind: 2, radius: 0.5}, 0)
	w.rememberJoint(7, "point", 0, id, 0, 4, 0)
	w.refreshWorldMatrices()
	w.tickDebugDraw(0.016)
	m.SetPosition(3, 2, 0)
	w.refreshWorldMatrices()
	var b lineBatch
	w.debugJoints(&b)
	// World-anchored (a = 0): body line, cross at the fixed anchor, and a
	// stretch line because the body slid without turning.
	if b.lines() != 1+3+1 {
		t.Fatalf("joint lines %d, want 5", b.lines())
	}
	if b.v[7] != 3 || b.v[8] != 2 {
		t.Fatalf("joint line should end on the moved body, got %v", b.v[7:10])
	}
	if b.v[14] != -0.12 || b.v[15] != 4 {
		t.Fatalf("anchor cross should stay at the world anchor, got %v", b.v[14:17])
	}
	w.sceneJoints = nil
	w.tickDebugDraw(0.016)
	if len(w.dbg.joints) != 0 {
		t.Fatal("freed joint anchor kept")
	}
}
