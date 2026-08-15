package runtime

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/arl/go-detour/detour"
	"github.com/arl/go-detour/recast"
	"github.com/arl/go-detour/sample/solomesh"
	"github.com/arl/gogeo/f32/d3"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type navMesh struct {
	meshID    int
	walkExtra []int
	obstacles []int
	nav       *detour.NavMesh
	query     *detour.NavMeshQuery
	baked     bool
}

type navAgent struct {
	ent     int
	nav     int
	speed   float32
	radius  float32
	path    [][3]float32
	i       int
	moving  bool
}

func (w *World) navCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createnavmesh": need(func(a []value.Value) (value.Value, error) {
			mesh := argI(a, 0, 0)
			if _, err := w.ent(mesh); err != nil {
				return value.Value{}, err
			}
			id := w.takeHandle(&w.freeNavs, &w.nextNav)
			if w.navs == nil {
				w.navs = map[int]*navMesh{}
			}
			w.navs[id] = &navMesh{meshID: mesh}
			w.curNav = id
			return value.Num(float64(id)), nil
		}),
		"addnavobstacle": n(func(a []value.Value) (value.Value, error) {
			ent := argI(a, 0, 0)
			nm := w.navs[w.curNav]
			if nm == nil {
				return z()
			}
			nm.obstacles = append(nm.obstacles, ent)
			nm.baked = false
			return z()
		}),
		"bakenavmesh": n(func(a []value.Value) (value.Value, error) {
			nm, err := w.navOf(argI(a, 0, w.curNav))
			if err != nil {
				return value.Value{}, err
			}
			if err := w.bakeNav(nm); err != nil {
				return value.Value{}, err
			}
			return value.Num(1), nil
		}),
		"createagent": need(func(a []value.Value) (value.Value, error) {
			ent := argI(a, 0, 0)
			if _, err := w.ent(ent); err != nil {
				return value.Value{}, err
			}
			if w.agents == nil {
				w.agents = map[int]*navAgent{}
			}
			w.agents[ent] = &navAgent{ent: ent, nav: w.curNav, speed: 4, radius: 0.5}
			return value.Num(float64(ent)), nil
		}),
		"setagentspeed": n(func(a []value.Value) (value.Value, error) {
			if ag := w.agents[argI(a, 0, 0)]; ag != nil {
				ag.speed = float32(argN(a, 1, 4))
			}
			return z()
		}),
		"setagentradius": n(func(a []value.Value) (value.Value, error) {
			if ag := w.agents[argI(a, 0, 0)]; ag != nil {
				ag.radius = float32(argN(a, 1, 0.5))
			}
			return z()
		}),
		"setagentdestination": n(func(a []value.Value) (value.Value, error) {
			ag := w.agents[argI(a, 0, 0)]
			if ag == nil {
				return z()
			}
			x, y, zz := float32(argN(a, 1, 0)), float32(argN(a, 2, 0)), float32(argN(a, 3, 0))
			pts, err := w.navFindPath(ag, x, y, zz)
			if err != nil || len(pts) == 0 {
				pts = [][3]float32{{x, y, zz}}
			}
			ag.path = pts
			ag.i = 0
			ag.moving = true
			return z()
		}),
		"getagentpathpointx": n(func(a []value.Value) (value.Value, error) {
			return w.agentPathComp(a, 0), nil
		}),
		"getagentpathpointy": n(func(a []value.Value) (value.Value, error) {
			return w.agentPathComp(a, 1), nil
		}),
		"getagentpathpointz": n(func(a []value.Value) (value.Value, error) {
			return w.agentPathComp(a, 2), nil
		}),
		"agentcountpath": n(func(a []value.Value) (value.Value, error) {
			ag := w.agents[argI(a, 0, 0)]
			if ag == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(len(ag.path))), nil
		}),
		"agentstop": n(func(a []value.Value) (value.Value, error) {
			if ag := w.agents[argI(a, 0, 0)]; ag != nil {
				ag.moving = false
			}
			return z()
		}),
		"updatenav": n(func(a []value.Value) (value.Value, error) {
			w.updateNav()
			return z()
		}),
	}
}

func (w *World) navOf(id int) (*navMesh, error) {
	n := w.navs[id]
	if n == nil {
		return nil, fmt.Errorf("invalid navmesh %d", id)
	}
	w.curNav = id
	return n, nil
}

func (w *World) agentPathComp(a []value.Value, c int) value.Value {
	ag := w.agents[argI(a, 0, 0)]
	i := argI(a, 1, 0)
	if ag == nil || i < 0 || i >= len(ag.path) {
		return value.Num(0)
	}
	return value.Num(float64(ag.path[i][c]))
}

func (w *World) bakeNav(nm *navMesh) error {
	e, err := w.ent(nm.meshID)
	if err != nil {
		return err
	}
	obj := scratchBuilder()
	defer releaseBuilder(obj)
	if _, err := writeNavOBJ(obj, e, nm, w); err != nil {
		return err
	}
	ctx := recast.NewBuildContext(false)
	sm := solomesh.New(ctx)
	if err := sm.LoadGeometry(bytes.NewReader([]byte(obj.String()))); err != nil {
		return fmt.Errorf("BakeNavMesh: %w", err)
	}
	nav, ok := sm.Build()
	if !ok || nav == nil {
		return fmt.Errorf("BakeNavMesh: recast bake failed")
	}
	st, q := detour.NewNavMeshQuery(nav, 2048)
	if detour.StatusFailed(st) {
		return fmt.Errorf("BakeNavMesh: query init failed")
	}
	nm.nav = nav
	nm.query = q
	nm.baked = true
	return nil
}

// writeNavOBJ writes walkable triangles (G3N mesh / collision mesh) plus obstacle
// geometry. AABB floor/box is used only when a mesh has no triangles.
func writeNavOBJ(b *strings.Builder, floor *Entity, nm *navMesh, w *World) (int, error) {
	vi := 0
	minY := float32(0)
	if w != nil && w.navMaxSlope > 0 && w.navMaxSlope < 90 {
		minY = float32(math.Cos(float64(w.navMaxSlope) * math.Pi / 180))
	}
	tris := entityWorldTris(floor)
	n := writeWorldTrisOBJSlope(b, tris, &vi, minY)
	if n == 0 {
		min, max := w.entityAABB(floor)
		n = writeWalkFloorAABB(b, min, max, &vi)
	}
	if n == 0 {
		return 0, fmt.Errorf("BakeNavMesh: no walkable triangles")
	}
	for _, id := range nm.walkExtra {
		if oe, err := w.ent(id); err == nil {
			n += writeWorldTrisOBJSlope(b, entityWorldTris(oe), &vi, minY)
		}
	}
	for _, id := range nm.obstacles {
		oe, err := w.ent(id)
		if err != nil {
			continue
		}
		if writeEntityTrisOBJ(b, oe, &vi) == 0 {
			omin, omax := w.entityAABB(oe)
			writeBoxOBJ(b, omin, omax, &vi)
		}
	}
	return n, nil
}

func writeEntityTrisOBJ(b *strings.Builder, e *Entity, vi *int) int {
	tris := entityWorldTris(e)
	if len(tris) == 0 {
		return 0
	}
	return writeWorldTrisOBJ(b, tris, vi)
}

func entityWorldTris(e *Entity) [][3]math32.Vector3 {
	if e == nil || e.mesh == nil {
		return nil
	}
	geom := e.mesh.GetGeometry()
	if geom == nil {
		return nil
	}
	n := e.mesh.GetNode()
	n.UpdateMatrixWorld()
	mat := n.MatrixWorld()
	return geomWorldTris(geom, mat)
}

func geomWorldTris(geom *geometry.Geometry, mat math32.Matrix4) [][3]math32.Vector3 {
	if geom == nil {
		return nil
	}
	var out [][3]math32.Vector3
	geom.ReadFaces(func(a, b, c math32.Vector3) bool {
		a.ApplyMatrix4(&mat)
		b.ApplyMatrix4(&mat)
		c.ApplyMatrix4(&mat)
		out = append(out, [3]math32.Vector3{a, b, c})
		return false
	})
	return out
}

func writeWorldTrisOBJ(b *strings.Builder, tris [][3]math32.Vector3, vi *int) int {
	return writeWorldTrisOBJSlope(b, tris, vi, 0)
}

func writeWorldTrisOBJSlope(b *strings.Builder, tris [][3]math32.Vector3, vi *int, minY float32) int {
	n := 0
	for _, t := range tris {
		if minY > 0 {
			e1 := t[1]
			e1.Sub(&t[0])
			e2 := t[2]
			e2.Sub(&t[0])
			nr := e1.Cross(&e2)
			lenN := nr.Length()
			if lenN < 1e-6 || math.Abs(float64(nr.Y/lenN)) < float64(minY) {
				continue
			}
		}
		for _, p := range t {
			x, y, z := fromG3N(p.X, p.Y, p.Z)
			fmt.Fprintf(b, "v %f %f %f\n", x, y, z)
		}
		fmt.Fprintf(b, "f %d %d %d\n", *vi+1, *vi+2, *vi+3)
		*vi += 3
		n++
	}
	return n
}

func writeWalkFloorAABB(b *strings.Builder, min, max [3]float32, vi *int) int {
	y := max[1]
	if y-min[1] < 0.2 {
		y = min[1]
	}
	pad := float32(0.5)
	x0, z0 := min[0]-pad, min[2]-pad
	x1, z1 := max[0]+pad, max[2]+pad
	fmt.Fprintf(b, "v %f %f %f\n", x0, y, z0)
	fmt.Fprintf(b, "v %f %f %f\n", x1, y, z0)
	fmt.Fprintf(b, "v %f %f %f\n", x1, y, z1)
	fmt.Fprintf(b, "v %f %f %f\n", x0, y, z1)
	base := *vi
	fmt.Fprintf(b, "f %d %d %d\nf %d %d %d\n", base+1, base+2, base+3, base+1, base+3, base+4)
	*vi += 4
	return 2
}

func writeBoxOBJ(b *strings.Builder, min, max [3]float32, vi *int) {
	x0, y0, z0 := min[0], min[1], min[2]
	x1, y1, z1 := max[0], max[1], max[2]
	pts := [][3]float32{
		{x0, y0, z0}, {x1, y0, z0}, {x1, y1, z0}, {x0, y1, z0},
		{x0, y0, z1}, {x1, y0, z1}, {x1, y1, z1}, {x0, y1, z1},
	}
	for _, p := range pts {
		fmt.Fprintf(b, "v %f %f %f\n", p[0], p[1], p[2])
	}
	base := *vi
	faces := [][3]int{
		{0, 1, 2}, {0, 2, 3},
		{4, 6, 5}, {4, 7, 6},
		{0, 4, 5}, {0, 5, 1},
		{3, 2, 6}, {3, 6, 7},
		{0, 3, 7}, {0, 7, 4},
		{1, 5, 6}, {1, 6, 2},
	}
	for _, f := range faces {
		fmt.Fprintf(b, "f %d %d %d\n", base+f[0]+1, base+f[1]+1, base+f[2]+1)
	}
	*vi += 8
}

func (w *World) entityAABB(e *Entity) (min, max [3]float32) {
	n := e.node.GetNode()
	p := scratchVec()
	defer releaseVec(p)
	n.WorldPosition(p)
	bx, by, bz := fromG3N(p.X, p.Y, p.Z)
	hx, hy, hz := e.boxX, e.boxY, e.boxZ
	if hx < 0.1 {
		hx = 1
	}
	if hy < 0.1 {
		hy = 1
	}
	if hz < 0.1 {
		hz = 1
	}
	if e.mesh != nil {
		bb := e.mesh.GetGeometry().BoundingBox()
		mat := n.MatrixWorld()
		first := true
		for _, c := range [8]math32.Vector3{
			{bb.Min.X, bb.Min.Y, bb.Min.Z},
			{bb.Max.X, bb.Min.Y, bb.Min.Z},
			{bb.Min.X, bb.Max.Y, bb.Min.Z},
			{bb.Max.X, bb.Max.Y, bb.Min.Z},
			{bb.Min.X, bb.Min.Y, bb.Max.Z},
			{bb.Max.X, bb.Min.Y, bb.Max.Z},
			{bb.Min.X, bb.Max.Y, bb.Max.Z},
			{bb.Max.X, bb.Max.Y, bb.Max.Z},
		} {
			c.ApplyMatrix4(&mat)
			x, y, z := fromG3N(c.X, c.Y, c.Z)
			if first {
				min = [3]float32{x, y, z}
				max = min
				first = false
				continue
			}
			if x < min[0] {
				min[0] = x
			}
			if y < min[1] {
				min[1] = y
			}
			if z < min[2] {
				min[2] = z
			}
			if x > max[0] {
				max[0] = x
			}
			if y > max[1] {
				max[1] = y
			}
			if z > max[2] {
				max[2] = z
			}
		}
		return min, max
	}
	return [3]float32{bx - hx, by - hy, bz - hz}, [3]float32{bx + hx, by + hy, bz + hz}
}

func (w *World) navFindPath(ag *navAgent, tx, ty, tz float32) ([][3]float32, error) {
	nm := w.navs[ag.nav]
	e, err := w.ent(ag.ent)
	if err != nil {
		return nil, err
	}
	p := worldPos(e.node.GetNode())
	sx, sy, sz := fromG3N(p.X, p.Y, p.Z)
	if nm == nil || !nm.baked || nm.query == nil {
		return [][3]float32{{sx, sy, sz}, {tx, ty, tz}}, nil
	}
	filter := detour.NewStandardQueryFilter()
	ext := d3.NewVec3XYZ(2, 4, 2)
	org := d3.NewVec3XYZ(sx, sy, sz)
	dst := d3.NewVec3XYZ(tx, ty, tz)
	st, orgRef, orgP := nm.query.FindNearestPoly(org, ext, filter)
	if detour.StatusFailed(st) || orgRef == 0 {
		return [][3]float32{{sx, sy, sz}, {tx, ty, tz}}, nil
	}
	st, dstRef, dstP := nm.query.FindNearestPoly(dst, ext, filter)
	if detour.StatusFailed(st) || dstRef == 0 {
		return [][3]float32{{sx, sy, sz}, {tx, ty, tz}}, nil
	}
	polys := make([]detour.PolyRef, 64)
	npoly, st := nm.query.FindPath(orgRef, dstRef, orgP, dstP, filter, polys)
	if detour.StatusFailed(st) || npoly == 0 {
		return [][3]float32{{sx, sy, sz}, {tx, ty, tz}}, nil
	}
	straight := make([]d3.Vec3, 64)
	for i := range straight {
		straight[i] = d3.NewVec3()
	}
	flags := make([]uint8, 64)
	refs := make([]detour.PolyRef, 64)
	nstr, st := nm.query.FindStraightPath(org, dst, polys[:npoly], straight, flags, refs, 0)
	if detour.StatusFailed(st) || nstr == 0 {
		return [][3]float32{{sx, sy, sz}, {tx, ty, tz}}, nil
	}
	out := make([][3]float32, 0, nstr)
	for i := 0; i < nstr; i++ {
		v := straight[i]
		out = append(out, [3]float32{v[0], v[1], v[2]})
	}
	return out, nil
}

func (w *World) updateNav() {
	if w.navStepped {
		return
	}
	w.navStepped = true
	dt := float32(w.delta)
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	for _, ag := range w.agents {
		if ag == nil || !ag.moving || ag.i >= len(ag.path) {
			continue
		}
		e, err := w.ent(ag.ent)
		if err != nil {
			continue
		}
		n := e.node.GetNode()
		p := worldPos(n)
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		tgt := ag.path[ag.i]
		dx, dy, dz := tgt[0]-x, tgt[1]-y, tgt[2]-z
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		step := ag.speed * dt
		if dist <= 0.12 || dist <= step {
			x, y, z = tgt[0], tgt[1], tgt[2]
			ag.i++
			if ag.i >= len(ag.path) {
				ag.moving = false
			}
		} else {
			s := step / dist
			x += dx * s
			y += dy * s
			z += dz * s
		}
		gx, gy, gz := toG3N(x, y, z)
		n.SetPosition(gx, gy, gz)
	}
}
