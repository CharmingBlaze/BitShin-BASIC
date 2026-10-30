package runtime

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/g3n/engine/gls"
)

func TestFBXASCIITriangle(t *testing.T) {
	src := `
; FBX 7.4.0 project file
P: "UpAxis", "int", "Integer", "",1
P: "UnitScaleFactor", "double", "Number", "",1
Geometry: 1, "Geometry::Tri", "Mesh" {
    Vertices: *9 {
        a: 0,0,0, 1,0,0, 0,1,0
    }
    PolygonVertexIndex: *3 {
        a: 0,1,-3
    }
}
`
	m, err := parseFBX([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g, err := m.geometry()
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Indices()) != 3 {
		t.Fatalf("tris %d", len(g.Indices())/3)
	}
}

func TestFBXSkinMovesVertex(t *testing.T) {
	src := `
P: "UpAxis", "int", "Integer", "",1
P: "UnitScaleFactor", "double", "Number", "",1
Geometry: 1, "Geometry::Tri", "Mesh" {
    Vertices: *9 { a: 0,0,0, 1,0,0, 0,1,0 }
    PolygonVertexIndex: *3 { a: 0,1,-3 }
}
Deformer: 2, "Cluster::Arm", "Cluster" {
    Indexes: *1 { a: 0 }
    Weights: *1 { a: 1 }
}
Model: 3, "Model::Arm", "LimbNode" {
    P: "Lcl Translation", "Lcl Translation", "", "A",0,2,0
}
`
	dir := t.TempDir()
	w := New(dir)
	path := filepath.Join(dir, "skin.fbx")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	id, err := w.loadFBXAt(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	e := w.ents[id]
	if e.skin == nil || len(e.skin.joints) != 1 || e.skin.joints[0].name != "Arm" {
		t.Fatal("missing Arm joint")
	}
	e.skin.joints[0].node.SetPosition(0, 3, 0)
	w.applySkins()
	pos := *e.mesh.GetGeometry().VBO(gls.VertexPosition).Buffer()
	if math.Abs(float64(pos[1]-1)) > 0.05 {
		t.Fatalf("skinned y %v", pos[1])
	}
}

func TestFBXBinaryZUpScaled(t *testing.T) {
	raw := buildTestFBX(t)
	m, err := parseFBX(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.upAxis != 2 {
		t.Fatalf("up %d", m.upAxis)
	}
	if m.unit != 100 {
		t.Fatalf("unit %v", m.unit)
	}
	g, err := m.geometry()
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Indices()) != 3 {
		t.Fatalf("indices %d", len(g.Indices()))
	}
	// 100 cm along Z-up becomes 1 m on Y.
	buf := g.VBO(gls.VertexPosition).Buffer()
	if buf == nil || len(*buf) < 9 {
		t.Fatal("positions")
	}
	// vertex 2 was (0,0,100) in Z-up cm -> (0, 1, 0) after conversion
	y := (*buf)[7]
	if y < 0.9 || y > 1.1 {
		t.Fatalf("converted Y %v from %v", y, *buf)
	}
}

func TestFBXDeflateArray(t *testing.T) {
	var raw bytes.Buffer
	zw := zlib.NewWriter(&raw)
	var nums [9 * 8]byte
	binary.LittleEndian.PutUint64(nums[0:], 0)
	binary.LittleEndian.PutUint64(nums[8:], 0)
	binary.LittleEndian.PutUint64(nums[16:], 0)
	if _, err := zw.Write(nums[:24]); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	comp := raw.Bytes()
	got, _, err := fbxArrayBytes(comp, 0, 1, len(comp))
	if err != nil {
		t.Fatal(err)
	}
	dec, err := decodeFBXArray('d', 3, got)
	if err != nil {
		t.Fatal(err)
	}
	if len(dec.df) != 3 || dec.df[0] != 0 {
		t.Fatalf("%v", dec.df)
	}
}

type binNode struct {
	name  string
	props [][]byte
	kids  []*binNode
}

func (n *binNode) write(base int) []byte {
	var propBytes []byte
	for _, p := range n.props {
		propBytes = append(propBytes, p...)
	}
	childBase := base + 13 + len(n.name) + len(propBytes)
	var kids []byte
	for _, c := range n.kids {
		kb := c.write(childBase + len(kids))
		kids = append(kids, kb...)
	}
	kids = append(kids, make([]byte, 13)...)
	total := 13 + len(n.name) + len(propBytes) + len(kids)
	hdr := make([]byte, 13)
	binary.LittleEndian.PutUint32(hdr[0:], uint32(base+total))
	binary.LittleEndian.PutUint32(hdr[4:], uint32(len(n.props)))
	binary.LittleEndian.PutUint32(hdr[8:], uint32(len(propBytes)))
	hdr[12] = byte(len(n.name))
	out := append(hdr, []byte(n.name)...)
	out = append(out, propBytes...)
	return append(out, kids...)
}

func buildTestFBX(t *testing.T) []byte {
	t.Helper()
	const ver = 7400
	var buf bytes.Buffer
	buf.WriteString("Kaydara FBX Binary  ")
	buf.WriteByte(0)
	buf.WriteByte(0x1A)
	buf.WriteByte(0)
	var v [4]byte
	binary.LittleEndian.PutUint32(v[:], ver)
	buf.Write(v[:])
	roots := []*binNode{
		{name: "GlobalSettings", kids: []*binNode{
			{name: "P", props: [][]byte{
				fbxStringProp("UpAxis"), fbxStringProp("int"), fbxStringProp("Integer"), fbxStringProp(""), fbxIntProp(2),
			}},
			{name: "P", props: [][]byte{
				fbxStringProp("UnitScaleFactor"), fbxStringProp("double"), fbxStringProp("Number"), fbxStringProp(""), fbxDoubleProp(100),
			}},
		}},
		{name: "Geometry", kids: []*binNode{
			{name: "Vertices", props: [][]byte{fbxF64Array([]float64{0, 0, 0, 100, 0, 0, 0, 0, 100})}},
			{name: "PolygonVertexIndex", props: [][]byte{fbxI32Array([]int32{0, 1, ^int32(2)})}},
		}},
	}
	base := buf.Len()
	for _, n := range roots {
		kb := n.write(base)
		buf.Write(kb)
		base += len(kb)
	}
	buf.Write(make([]byte, 13))
	return buf.Bytes()
}

func fbxStringProp(s string) []byte {
	b := []byte{'S'}
	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(len(s)))
	b = append(b, n[:]...)
	return append(b, []byte(s)...)
}

func fbxIntProp(v int32) []byte {
	b := []byte{'I', 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(b[1:], uint32(v))
	return b
}

func fbxDoubleProp(v float64) []byte {
	b := make([]byte, 9)
	b[0] = 'D'
	binary.LittleEndian.PutUint64(b[1:], math.Float64bits(v))
	return b
}

func fbxF64Array(vs []float64) []byte {
	raw := make([]byte, len(vs)*8)
	for i, v := range vs {
		binary.LittleEndian.PutUint64(raw[i*8:], math.Float64bits(v))
	}
	return fbxArray('d', raw, len(vs))
}

func fbxI32Array(vs []int32) []byte {
	raw := make([]byte, len(vs)*4)
	for i, v := range vs {
		binary.LittleEndian.PutUint32(raw[i*4:], uint32(v))
	}
	return fbxArray('i', raw, len(vs))
}

func fbxArray(kind byte, raw []byte, count int) []byte {
	b := []byte{kind}
	var h [12]byte
	binary.LittleEndian.PutUint32(h[0:], uint32(count))
	binary.LittleEndian.PutUint32(h[4:], 0)
	binary.LittleEndian.PutUint32(h[8:], uint32(len(raw)))
	b = append(b, h[:]...)
	return append(b, raw...)
}
