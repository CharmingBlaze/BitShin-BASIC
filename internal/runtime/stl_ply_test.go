package runtime

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/g3n/engine/gls"
)

func TestSTLASCIIQuadWelds(t *testing.T) {
	src := `solid quad
facet normal 0 0 1
 outer loop
  vertex 0 0 0
  vertex 1 0 0
  vertex 1 1 0
 endloop
endfacet
facet normal 0 0 1
 outer loop
  vertex 0 0 0
  vertex 1 1 0
  vertex 0 1 0
 endloop
endfacet
endsolid quad
`
	m, err := parseSTL([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.idx) != 6 || len(m.pos) != 4*3 {
		t.Fatalf("%d indices, %d verts; shared corners should weld", len(m.idx), len(m.pos)/3)
	}
	if m.nor[2] != 1 {
		t.Fatalf("normal %v", m.nor[:3])
	}
}

func stlBinary(tris [][3][3]float32) []byte {
	var b bytes.Buffer
	b.Write(make([]byte, 80))
	binary.Write(&b, binary.LittleEndian, uint32(len(tris)))
	for _, tr := range tris {
		binary.Write(&b, binary.LittleEndian, [3]float32{}) // normal left zero
		binary.Write(&b, binary.LittleEndian, tr)
		binary.Write(&b, binary.LittleEndian, uint16(0))
	}
	return b.Bytes()
}

func TestSTLBinaryComputesNormalAndLoads(t *testing.T) {
	// A binary header that starts with "solid" must still read as binary.
	raw := stlBinary([][3][3]float32{{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}})
	copy(raw, "solid exported by a CAD tool")
	m, err := parseSTL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.idx) != 3 || m.nor[2] < 0.99 {
		t.Fatalf("idx %d normal %v", len(m.idx), m.nor[:3])
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tri.stl"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	w := New(dir)
	id, err := w.loadMeshFile("tri.stl", 0)
	if err != nil {
		t.Fatal(err)
	}
	e := w.ents[id]
	if e == nil || e.mesh == nil || e.kind != "mesh" || len(e.mesh.GetGeometry().Indices()) != 3 {
		t.Fatal("STL did not load as a mesh entity")
	}
}

func TestPLYASCIIQuadWithColoursAndUVs(t *testing.T) {
	src := `ply
format ascii 1.0
comment four corners
element vertex 4
property float x
property float y
property float z
property float nx
property float ny
property float nz
property float s
property float t
property uchar red
property uchar green
property uchar blue
element face 1
property list uchar int vertex_indices
end_header
0 0 0 0 0 1 0 0 255 0 0
1 0 0 0 0 1 1 0 0 255 0
1 1 0 0 0 1 1 1 0 0 255
0 1 0 0 0 1 0 1 255 255 255
4 0 1 2 3
`
	m, err := parsePLY([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.idx) != 6 {
		t.Fatalf("quad gave %d indices", len(m.idx))
	}
	if len(m.uv) != 8 || m.uv[4] != 1 || m.uv[5] != 1 {
		t.Fatalf("uv %v", m.uv)
	}
	if len(m.col) != 12 || m.col[0] != 1 || m.col[4] != 1 || m.col[1] != 0 {
		t.Fatalf("colours %v", m.col)
	}
	g, err := m.geometry()
	if err != nil {
		t.Fatal(err)
	}
	if g.VBO(gls.VertexColor) == nil || g.VBO(gls.VertexNormal) == nil {
		t.Fatal("missing colour or normal VBO")
	}
}

func TestPLYBinaryLittleEndianSkipsOtherElements(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("ply\r\nformat binary_little_endian 1.0\r\n" +
		"element vertex 3\r\nproperty float x\r\nproperty float y\r\nproperty float z\r\n" +
		"element face 1\r\nproperty list uchar uint vertex_indices\r\nproperty uchar flags\r\n" +
		"element edge 1\r\nproperty int vertex1\r\nproperty int vertex2\r\n" +
		"end_header\r\n")
	le := binary.LittleEndian
	binary.Write(&b, le, [9]float32{0, 0, 0, 2, 0, 0, 0, 2, 0})
	b.WriteByte(3)
	binary.Write(&b, le, [3]uint32{0, 1, 2})
	b.WriteByte(7)
	binary.Write(&b, le, [2]int32{0, 1})
	m, err := parsePLY(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(m.idx) != 3 || m.pos[3] != 2 || m.pos[7] != 2 {
		t.Fatalf("idx %v pos %v", m.idx, m.pos)
	}
	if len(m.nor) != 0 || len(m.col) != 0 {
		t.Fatal("no normals or colours in the file")
	}
	g, err := m.geometry()
	if err != nil {
		t.Fatal(err)
	}
	nor := *g.VBO(gls.VertexNormal).Buffer()
	if math.Abs(float64(nor[2])-1) > 1e-4 {
		t.Fatalf("computed normal %v", nor[:3])
	}

	if _, err := parsePLY(b.Bytes()[:b.Len()-10]); err == nil {
		t.Fatal("truncated body should fail")
	}
}

func TestPLYRejectsBigEndianAndPointClouds(t *testing.T) {
	if _, err := parsePLY([]byte("ply\nformat binary_big_endian 1.0\nend_header\n")); err == nil {
		t.Fatal("big endian should fail")
	}
	cloud := "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float y\nproperty float z\nend_header\n1 2 3\n"
	if _, err := parsePLY([]byte(cloud)); err == nil {
		t.Fatal("point cloud should fail")
	}
}
