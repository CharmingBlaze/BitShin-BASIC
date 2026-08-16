package runtime

import (
	"strings"
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

func TestGeomWorldTrisFromCube(t *testing.T) {
	g := geometry.NewCube(2)
	var ident math32.Matrix4
	ident.Identity()
	tris := geomWorldTris(g, ident)
	if len(tris) < 12 {
		t.Fatalf("cube should yield 12 triangles, got %d", len(tris))
	}
	obj := &strings.Builder{}
	vi := 0
	n := writeWorldTrisOBJ(obj, tris, &vi)
	if n != len(tris) {
		t.Fatalf("wrote %d want %d", n, len(tris))
	}
	s := obj.String()
	if !strings.Contains(s, "\nv ") || !strings.Contains(s, "\nf ") {
		t.Fatalf("OBJ missing verts/faces:\n%s", s)
	}
	if vi != n*3 {
		t.Fatalf("vertex index %d want %d", vi, n*3)
	}
}

func TestNavBakeUsesMeshTriangles(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	mat := material.NewStandard(&math32.Color{0.5, 0.5, 0.5})
	floorGeom := geometry.NewCube(1)
	floorMesh := graphic.NewMesh(floorGeom, mat)
	floorMesh.SetScale(10, 0.2, 10)
	floor := &Entity{node: floorMesh, mesh: floorMesh, mat: mat}
	fid := w.addEntity(floor, 0)
	floor.boxX, floor.boxY, floor.boxZ = 10, 0.2, 10

	blockGeom := geometry.NewCube(1)
	blockMesh := graphic.NewMesh(blockGeom, mat)
	blockMesh.SetPosition(0, 1, 2)
	block := &Entity{node: blockMesh, mesh: blockMesh, mat: mat}
	bid := w.addEntity(block, 0)

	nm := &navMesh{meshID: fid, obstacles: []int{bid}}
	obj := &strings.Builder{}
	n, err := writeNavOBJ(obj, floor, nm, w)
	if err != nil {
		t.Fatal(err)
	}
	if n < 12 {
		t.Fatalf("floor mesh tris %d (AABB fallback?)", n)
	}
	// Cube = 12 tris; obstacle cube adds 12 more faces in the OBJ.
	faces := strings.Count(obj.String(), "\nf ")
	if faces < 24 {
		t.Fatalf("expected mesh floor + mesh obstacle faces, got %d\n%s", faces, obj.String())
	}
	if err := w.bakeNav(nm); err != nil {
		t.Fatal(err)
	}
	if !nm.baked || nm.query == nil {
		t.Fatal("nav not baked")
	}
}

func TestCrowdExampleBakeAndAgentsMove(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	w.delta = 1.0 / 60.0
	mat := material.NewStandard(&math32.Color{0.5, 0.5, 0.5})
	floorGeom := geometry.NewCube(2)
	floorMesh := graphic.NewMesh(floorGeom, mat)
	floorMesh.SetScale(14, 0.2, 14)
	floor := &Entity{node: floorMesh, mesh: floorMesh, mat: mat}
	fid := w.addEntity(floor, 0)

	nm := &navMesh{meshID: fid}
	if err := w.bakeNav(nm); err != nil {
		t.Fatalf("crowd.bb BakeNavMesh: %v", err)
	}
	nid := 1
	w.navs[nid] = nm
	w.curNav = nid

	cr := &crowd{sep: 1.3}
	w.crowds[1] = cr
	w.curCrowd = 1

	startZ := make([]float32, 8)
	for i := 0; i < 8; i++ {
		g := geometry.NewCube(2)
		m := graphic.NewMesh(g, mat)
		m.SetScale(0.35, 0.7, 0.35)
		bx, by, bz := toG3N(-6+float32(i)*1.5, 1, -5)
		m.SetPosition(bx, by, bz)
		e := &Entity{node: m, mesh: m, mat: mat}
		id := w.addEntity(e, 0)
		w.agents[id] = &navAgent{ent: id, nav: nid, speed: 4, radius: 0.45}
		cr.agents = append(cr.agents, id)
		startZ[i] = -5
	}

	c := w.crowds[1]
	for _, id := range c.agents {
		ag := w.agents[id]
		pts, err := w.navFindPath(ag, 6, 1, 6)
		if err != nil || len(pts) == 0 {
			pts = [][3]float32{{6, 1, 6}}
		}
		ag.path = pts
		ag.i = 0
		ag.moving = true
	}

	moved := 0
	for step := 0; step < 90; step++ {
		w.navStepped = false
		w.updateNav()
		w.tickCrowds()
	}
	for i, id := range cr.agents {
		e := w.ents[id]
		p := worldPos(e.node.GetNode())
		_, _, z := fromG3N(p.X, p.Y, p.Z)
		if z > startZ[i]+0.4 {
			moved++
		}
	}
	if moved < 4 {
		t.Fatalf("crowd agents did not walk toward destination, moved=%d/8", moved)
	}
}

func TestNavBakeAABBFallback(t *testing.T) {
	w := New(".")
	w.scene = core.NewNode()
	pivot := core.NewNode()
	e := &Entity{node: pivot, boxX: 4, boxY: 0.1, boxZ: 4}
	id := w.addEntity(e, 0)
	_ = id
	nm := &navMesh{meshID: id}
	obj := &strings.Builder{}
	n, err := writeNavOBJ(obj, e, nm, w)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("AABB fallback should write 2 tris, got %d", n)
	}
}
