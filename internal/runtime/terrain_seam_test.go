package runtime

import (
	"testing"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func outdoorTerrain(t *testing.T) (*World, *terrain) {
	t.Helper()
	w := New(".")
	v, err := w.commandTable()["generateheightmappreset"]([]value.Value{
		value.Str("rolling-hills"), value.Num(96), value.Num(96), value.Num(7),
	})
	if err != nil {
		t.Fatal(err)
	}
	hm := w.hmaps[v.Int()]
	id := w.addTerrain(heightFieldFromMap(hm, 140, 140, 18), 25, 2)
	return w, w.terrains[id]
}

func meshPositions(t *testing.T, w *World, id int) []math32.Vector3 {
	t.Helper()
	e := w.ents[id]
	if e == nil || e.mesh == nil {
		t.Fatalf("entity %d has no mesh", id)
	}
	buf := e.mesh.GetGeometry().VBO(gls.VertexPosition).Buffer()
	var out []math32.Vector3
	for i := 0; i+2 < len(*buf); i += 3 {
		out = append(out, math32.Vector3{X: (*buf)[i], Y: (*buf)[i+1], Z: (*buf)[i+2]})
	}
	return out
}

func TestHorizonInnerEdgeSitsOnChunkBorderVerts(t *testing.T) {
	w, tr := outdoorTerrain(t)
	cw := tr.chunkWorld()
	segs := tr.chunkSegs(cw)
	cam := chunkKey{}
	// A far-LOD, fully morphed chunk on the window edge: its border must
	// still be the full-resolution raw edge the ring meets.
	k := chunkKey{X: tr.radius, Z: 0}
	w.attachTerrainChunk(tr, k, float32(k.X)*cw, 0, cw, segs, 2, 1, lodCode(2, 1))
	chunk := map[[2]float32]float32{}
	for _, p := range meshPositions(t, w, tr.ents[k]) {
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		if prev, ok := chunk[[2]float32{x, z}]; !ok || y > prev {
			chunk[[2]float32{x, z}] = y // the skirt shares x,z below the surface
		}
	}
	w.syncHorizon(tr, cw, cam)
	edgeX := float32(k.X+1) * cw
	shared := 0
	for _, p := range meshPositions(t, w, tr.horizonID) {
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		if math32.Abs(x-edgeX) > 1e-3 || z < 0 || z > cw {
			continue
		}
		var cy float32
		found := false
		for kz, v := range chunk {
			if math32.Abs(kz[0]-x) < 1e-3 && math32.Abs(kz[1]-z) < 1e-3 {
				cy, found = v, true
				break
			}
		}
		if !found {
			t.Fatalf("ring vertex (%v,%v) is not on the chunk's border lattice", x, z)
		}
		if math32.Abs(cy-y) > 1e-3 {
			t.Fatalf("ring height %v vs chunk %v at (%v,%v)", y, cy, x, z)
		}
		shared++
	}
	if shared < 3 {
		t.Fatalf("only %d ring verts on the shared edge", shared)
	}
}

func TestLODRebuildKeepsTreesAndProps(t *testing.T) {
	w, tr := outdoorTerrain(t)
	tree := w.addEntity(&Entity{}, 0)
	tr.setScatterLayer("tree", tree, 0.7, 0)
	w.refreshScatter(tr)
	cw := tr.chunkWorld()
	segs := tr.chunkSegs(cw)
	var k chunkKey
	var before []int
	for z := -tr.radius; z <= tr.radius && len(before) == 0; z++ {
		for x := -tr.radius; x <= tr.radius && len(before) == 0; x++ {
			k = chunkKey{X: x, Z: z}
			w.attachTerrainChunk(tr, k, float32(x)*cw, float32(z)*cw, cw, segs, 1, 0, lodCode(1, 0))
			before = append([]int(nil), tr.scatterIDs[k]...)
		}
	}
	if len(before) == 0 {
		t.Fatal("no trees planted on any chunk")
	}
	ents := make([]*Entity, len(before))
	for i, id := range before {
		ents[i] = w.ents[id]
	}
	x0, z0 := float32(k.X)*cw, float32(k.Z)*cw
	for _, m := range []float32{0.25, 0.5, 1} {
		w.attachTerrainChunk(tr, k, x0, z0, cw, segs, 1, m, lodCode(1, m))
	}
	w.attachTerrainChunk(tr, k, x0, z0, cw, segs, 2, 0, lodCode(2, 0))
	after := tr.scatterIDs[k]
	if len(after) != len(before) {
		t.Fatalf("scatter %v became %v", before, after)
	}
	for i := range before {
		// IDs are recycled, so compare the entity itself.
		if before[i] != after[i] || ents[i] != w.ents[after[i]] {
			t.Fatalf("tree batch %d replanted on a LOD swap", before[i])
		}
	}
}

func TestTerrainEdgeKeepsTheLandGoing(t *testing.T) {
	w, tr := outdoorTerrain(t)
	if h := tr.hf.atWorld(150, 0); h != 0 {
		t.Fatalf("default edge should stay 0 past the map (island), got %v", h)
	}
	if _, err := w.Call("setterrainedge", []value.Value{value.Num(float64(tr.id)), value.Str("mirror")}); err != nil {
		t.Fatal(err)
	}
	tr = w.terrains[tr.id]
	// 70 is the east border; 70+d mirrors 70-d.
	for _, d := range []float32{0.5, 7, 33} {
		in, out := tr.hf.atWorld(70-d, 12), tr.hf.atWorld(70+d, 12)
		if math32.Abs(in-out) > 1e-3 {
			t.Fatalf("mirror at %v: %v vs %v", d, in, out)
		}
	}
	// Continuous across the border, so no cliff.
	if math32.Abs(tr.hf.atWorld(69.99, 12)-tr.hf.atWorld(70.01, 12)) > 0.05 {
		t.Fatal("mirror edge should be continuous")
	}
	w.Call("setterrainedge", []value.Value{value.Num(float64(tr.id)), value.Str("clamp")})
	if a, b := tr.hf.atWorld(70, -3), tr.hf.atWorld(400, -3); math32.Abs(a-b) > 1e-3 {
		t.Fatalf("clamp: border %v, far %v", a, b)
	}
}

func TestWaterIsNoCoarserThanTheGroundUnderIt(t *testing.T) {
	cell, reach := float32(2.9), float32(280)
	xs, _ := waterLODCoordsOver(1800, 48, cell, reach)
	mid := len(xs) / 2
	for i := mid + 1; i < len(xs); i++ {
		if xs[i] > reach {
			break
		}
		if step := xs[i] - xs[i-1]; step > cell+0.05 {
			t.Fatalf("water cell %v at %v is coarser than the %v ground", step, xs[i], cell)
		}
	}
	if xs[len(xs)-1] != 1800 {
		t.Fatal("the water must still reach the horizon")
	}
}

func TestMeshTreesKeepTheirChunkAsTheCameraMoves(t *testing.T) {
	w, tr := outdoorTerrain(t)
	trunk := w.meshEnt(geometry.NewCylinder(0.2, 2, 6, 1, true, true), 0)
	crown := w.meshEnt(geometry.NewCone(1.2, 3, 7, 1, true), trunk)
	w.ents[crown].node.GetNode().SetPosition(0, 2.2, 0)
	parts, _ := w.treeMeshParts(trunk)
	if len(parts) != 2 {
		t.Fatalf("template has %d parts, want trunk + crown", len(parts))
	}
	tr.setScatterLayer("tree", trunk, 0.7, 0)
	w.refreshScatter(tr)
	cw := tr.chunkWorld()
	for i := 0; i < 40; i++ {
		w.syncMeshTrees(tr, cw, chunkKey{})
	}
	planted := 0
	for _, ids := range tr.treeMesh {
		planted += len(ids)
	}
	if planted == 0 {
		t.Fatal("no 3D trees planted")
	}
	before := map[chunkKey]*Entity{}
	for k, ids := range tr.treeMesh {
		if len(ids) > 0 {
			before[k] = w.ents[ids[0]]
		}
	}
	w.syncMeshTrees(tr, cw, chunkKey{X: 1})
	for k, e := range before {
		ids, ok := tr.treeMesh[k]
		if !ok {
			continue // walked out of reach
		}
		if len(ids) == 0 || w.ents[ids[0]] != e {
			t.Fatalf("chunk %v replanted its trees on a one-chunk move", k)
		}
	}
}

func TestTerrainSnowTrueIsNotATerrainHandle(t *testing.T) {
	w, tr := outdoorTerrain(t)
	if tr.id != 1 {
		t.Skipf("terrain handle %d; the clash needs handle 1", tr.id)
	}
	w.Call("setterrainsnow", []value.Value{value.Num(1), value.Num(140)})
	if !tr.snowOn || tr.snowH != 140 {
		t.Fatalf("SetTerrainSnow(True, 140): on %v at %v", tr.snowOn, tr.snowH)
	}
	w.Call("setterrainsnow", []value.Value{value.Num(1), value.Num(1), value.Num(60)})
	if tr.snowH != 60 {
		t.Fatalf("SetTerrainSnow(t, True, 60): at %v", tr.snowH)
	}
}

func TestPlantedCopiesKeepTheirSize(t *testing.T) {
	// instanceMatrix once built its scale on a zero matrix, so every copy
	// collapsed to a point and planted trees were invisible.
	w, tr := outdoorTerrain(t)
	trunk := w.meshEnt(geometry.NewCylinder(0.2, 3, 5, 1, false, false), 0)
	w.ents[trunk].node.GetNode().SetScale(1, 2, 1)
	w.ents[trunk].node.GetNode().SetVisible(false)
	ground := tr.hf.atWorld(10, 12)
	ids := w.plantMeshTrees(trunk, []scatterPt{{x: 10, y: ground, z: 12, scale: 1}})
	if len(ids) != 1 {
		t.Fatalf("%d batches", len(ids))
	}
	buf := w.ents[w.instances[ids[0]].batch].mesh.GetGeometry().VBO(gls.VertexPosition).Buffer()
	lo, hi := float32(1e9), float32(-1e9)
	for i := 1; i < len(*buf); i += 3 {
		lo = math32.Min(lo, (*buf)[i])
		hi = math32.Max(hi, (*buf)[i])
	}
	if math32.Abs(lo-ground) > 1e-3 || math32.Abs(hi-ground-6) > 1e-3 {
		t.Fatalf("trunk spans %v..%v, want %v..%v (3 m scaled x2, base on the ground)", lo, hi, ground, ground+6)
	}
}
