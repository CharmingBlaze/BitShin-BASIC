package runtime

import (
	"testing"

	"github.com/g3n/engine/gls"

	"moonbasic/internal/value"
)

func TestMeshBuildQuad(t *testing.T) {
	w := New(".")
	w.ready = true
	call := func(name string, args ...float64) float64 {
		t.Helper()
		v, err := w.Call(name, edNums(args))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return v.Num
	}
	m := int(call("createmesh"))
	f := float64(m)
	for i, p := range [][5]float64{{0, 0, 0, 0, 1}, {1, 0, 0, 1, 1}, {1, 0, 1, 1, 0}, {0, 0, 1, 0, 0}} {
		if v := call("addvertex", f, p[0], p[1], p[2], p[3], p[4]); int(v) != i {
			t.Fatalf("AddVertex returned %v, want %d", v, i)
		}
	}
	call("addtriangle", f, 0, 2, 1)
	if tri := call("addtriangle", f, 0, 3, 2); tri != 1 {
		t.Fatalf("second triangle index %v", tri)
	}
	call("vertexcolor", f, 2, 255, 0, 0)
	call("vertextexcoords", f, 3, 0.25, 0.75, 1)
	call("updatemesh", f)

	g := w.ents[m].mesh.GetGeometry()
	if n := len(g.Indices()); n != 6 {
		t.Fatalf("%d indices", n)
	}
	if !w.ents[m].mesh.Renderable() {
		t.Fatal("built mesh is not renderable")
	}
	for _, at := range []gls.AttribType{gls.VertexPosition, gls.VertexNormal, gls.VertexTexcoord, gls.VertexTexcoord2, gls.VertexColor} {
		if g.VBO(at) == nil {
			t.Fatalf("no VBO for attribute %d", at)
		}
	}
	// Winding 0,2,1 on the XZ plane faces up: smoothed normals point +Y.
	if ny := call("vertexny", f, 0); ny < 0.99 {
		t.Fatalf("normal Y %v", ny)
	}
	if r := call("vertexred", f, 2); r != 255 || call("vertexgreen", f, 2) != 0 {
		t.Fatalf("vertex colour %v", r)
	}
	if u := call("vertexu", f, 3, 1); u != 0.25 {
		t.Fatalf("second UV %v", u)
	}
	if bb := g.BoundingBox(); bb.Max.X != 1 || bb.Max.Z != 1 {
		t.Fatalf("bounds %+v", bb)
	}
	if call("counttriangles", f) != 2 || call("trianglevertex", f, 1, 1) != 3 {
		t.Fatal("triangle readback")
	}
	if _, err := w.Call("addtriangle", edNums([]float64{f, 0, 1, 9})); err == nil {
		t.Fatal("AddTriangle accepted a missing vertex")
	}

	// Moving a vertex reaches the GPU by itself before the next frame.
	call("vertexcoords", f, 2, 3, 0, 1)
	w.tickMeshBuilds()
	if bb := g.BoundingBox(); bb.Max.X != 3 {
		t.Fatalf("bounds after move %+v", bb)
	}
	call("clearmesh", f)
	call("updatemesh", f)
	if call("countvertices", f) != 0 || w.ents[m].mesh.Renderable() {
		t.Fatal("ClearMesh left geometry")
	}
}

func TestMeshBuildEditsPrimitive(t *testing.T) {
	w := New(".")
	w.ready = true
	v, _ := w.Call("createcube", nil)
	c := v.Num
	n, err := w.Call("countvertices", []value.Value{value.Num(c)})
	if err != nil || n.Num < 8 {
		t.Fatalf("cube vertices %v %v", n.Num, err)
	}
	x, _ := w.Call("vertexx", edNums([]float64{c, 0}))
	if _, err := w.Call("vertexcoords", edNums([]float64{c, 0, x.Num * 2, 0, 0})); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Call("updatemesh", []value.Value{value.Num(c)}); err != nil {
		t.Fatal(err)
	}
	if got, _ := w.Call("vertexx", edNums([]float64{c, 0})); got.Num != x.Num*2 {
		t.Fatalf("vertex x %v, want %v", got.Num, x.Num*2)
	}
}
