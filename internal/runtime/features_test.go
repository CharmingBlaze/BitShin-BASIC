package runtime

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/parse"
	"moonbasic/internal/value"
)

func TestDDSAndKTX2UploadAsCompressed(t *testing.T) {
	dir := t.TempDir()
	w := New(dir)
	dds := filepath.Join(dir, "block.dds")
	if err := os.WriteFile(dds, testDDS(4, 4), 0o644); err != nil {
		t.Fatal(err)
	}
	id, err := w.loadTextureFile("block.dds")
	if err != nil || id == 0 {
		t.Fatal(err)
	}
	if !w.texs[id].tex.Compressed() {
		t.Fatal("dds should stay compressed on the GPU texture")
	}
	ktx := filepath.Join(dir, "block.ktx2")
	if err := os.WriteFile(ktx, testKTX2(4, 4), 0o644); err != nil {
		t.Fatal(err)
	}
	id2, err := w.loadTextureFile("block.ktx2")
	if err != nil || id2 == 0 {
		t.Fatal(err)
	}
	if !w.texs[id2].tex.Compressed() {
		t.Fatal("ktx2 should stay compressed")
	}
}

func TestKTX2SupercompressionInflates(t *testing.T) {
	raw := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	for _, scheme := range []uint32{2, 3} {
		var comp []byte
		if scheme == 3 {
			var buf bytes.Buffer
			zw := zlib.NewWriter(&buf)
			if _, err := zw.Write(raw); err != nil {
				t.Fatal(err)
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			comp = buf.Bytes()
		} else {
			enc, err := zstd.NewWriter(nil)
			if err != nil {
				t.Fatal(err)
			}
			comp = enc.EncodeAll(raw, nil)
			enc.Close()
		}
		img, err := parseKTX2(testKTX2Super(4, 4, scheme, comp, len(raw)))
		if err != nil {
			t.Fatal(scheme, err)
		}
		if len(img.data) != len(raw) || img.data[0] != 1 || img.data[7] != 8 {
			t.Fatalf("scheme %d payload %v", scheme, img.data)
		}
	}
	if _, err := parseKTX2(testKTX2Super(4, 4, 1, []byte{0}, 1)); err == nil {
		t.Fatal("BasisLZ should stay rejected")
	}
}

func testDDS(w, h int) []byte {
	b := make([]byte, 128+8)
	copy(b, "DDS ")
	binary.LittleEndian.PutUint32(b[4:], 124)
	binary.LittleEndian.PutUint32(b[12:], uint32(h))
	binary.LittleEndian.PutUint32(b[16:], uint32(w))
	binary.LittleEndian.PutUint32(b[76:], 32)
	binary.LittleEndian.PutUint32(b[80:], 0x4)
	copy(b[84:88], "DXT1")
	return b
}

func testKTX2(w, h int) []byte {
	b := make([]byte, 80+24+8)
	ident := []byte{0xAB, 'K', 'T', 'X', ' ', '2', '0', 0xBB, 0x0D, 0x0A, 0x1A, 0x0A}
	copy(b, ident)
	binary.LittleEndian.PutUint32(b[12:], 137)
	binary.LittleEndian.PutUint32(b[20:], uint32(w))
	binary.LittleEndian.PutUint32(b[24:], uint32(h))
	binary.LittleEndian.PutUint32(b[40:], 1) // levels
	off := 80 + 24
	binary.LittleEndian.PutUint64(b[80:], uint64(off))
	binary.LittleEndian.PutUint64(b[88:], 8)
	binary.LittleEndian.PutUint64(b[96:], 8)
	return b
}

func testKTX2Super(w, h int, scheme uint32, payload []byte, uncomp int) []byte {
	b := make([]byte, 80+24+len(payload))
	ident := []byte{0xAB, 'K', 'T', 'X', ' ', '2', '0', 0xBB, 0x0D, 0x0A, 0x1A, 0x0A}
	copy(b, ident)
	binary.LittleEndian.PutUint32(b[12:], 137)
	binary.LittleEndian.PutUint32(b[20:], uint32(w))
	binary.LittleEndian.PutUint32(b[24:], uint32(h))
	binary.LittleEndian.PutUint32(b[40:], 1)
	binary.LittleEndian.PutUint32(b[44:], scheme)
	off := 80 + 24
	binary.LittleEndian.PutUint64(b[80:], uint64(off))
	binary.LittleEndian.PutUint64(b[88:], uint64(len(payload)))
	binary.LittleEndian.PutUint64(b[96:], uint64(uncomp))
	copy(b[off:], payload)
	return b
}

func TestOptimizeMeshLowersCacheMisses(t *testing.T) {
	g := geometry.NewGeometry()
	const n = 40
	pos := math32.NewArrayF32(0, n*3)
	for i := 0; i < n; i++ {
		pos.Append(float32(i), 0, 0)
	}
	idx := math32.NewArrayU32(0, (n-2)*3)
	for i := n - 3; i >= 0; i-- {
		idx.Append(uint32(i+2), uint32(i), uint32(i+1))
	}
	nor := math32.NewArrayF32(len(pos), len(pos))
	uv := math32.NewArrayF32(0, n*2)
	for i := 0; i < n; i++ {
		uv.Append(0, 0)
	}
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uv).AddAttrib(gls.VertexTexcoord))
	before := cacheMisses(g.Indices(), 32)
	got := optimizeGeometry(g)
	if got != (n - 2) {
		t.Fatalf("triangles %d", got)
	}
	after := cacheMisses(g.Indices(), 32)
	if after > before {
		t.Fatalf("cache misses rose %d -> %d", before, after)
	}
	if len(*g.VBO(gls.VertexPosition).Buffer())/3 != n && len(*g.VBO(gls.VertexPosition).Buffer())/3 > n {
		t.Fatal("vertex fetch grew the mesh")
	}
}

func cacheMisses(idx []uint32, cache int) int {
	var fifo []uint32
	miss := 0
	for _, v := range idx {
		hit := false
		for _, c := range fifo {
			if c == v {
				hit = true
				break
			}
		}
		if !hit {
			miss++
			fifo = append(fifo, v)
			if len(fifo) > cache {
				fifo = fifo[1:]
			}
		}
	}
	return miss
}

func TestTwoBoneIKReachesTarget(t *testing.T) {
	w := New(".")
	upper := core.NewNode()
	upper.SetName("upper")
	lower := core.NewNode()
	lower.SetName("lower")
	lower.SetPosition(0, 1, 0)
	end := core.NewNode()
	end.SetName("end")
	end.SetPosition(0, 1, 0)
	lower.Add(end)
	upper.Add(lower)
	id := w.addEntity(&Entity{node: upper}, 0)
	w.solveIK(&Entity{})
	_, err := w.Call("setik", []value.Value{
		value.Num(float64(id)), value.Str("upper"), value.Str("lower"), value.Str("end"),
		value.Num(1), value.Num(1), value.Num(0), value.Num(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	w.refreshWorldMatrices()
	var p math32.Vector3
	end.WorldPosition(&p)
	dx, dy := p.X-1, p.Y-1
	if math.Hypot(float64(dx), float64(dy)) > 0.2 {
		t.Fatalf("end at %v %v, want near (1,1)", p.X, p.Y)
	}
}

func TestAdditiveLayerOffsetsBind(t *testing.T) {
	n := core.NewNode()
	base := localXF{n: n, p: math32.Vector3{Y: 1}, q: math32.Quaternion{W: 1}, s: math32.Vector3{X: 1, Y: 1, Z: 1}}
	bind := base
	layer := base
	layer.p.Y = 1.5
	applyAdditive(n, base, bind, layer, 1)
	if math.Abs(float64(n.Position().Y-1.5)) > 0.01 {
		t.Fatalf("Y %v", n.Position().Y)
	}
}

func TestBehaviorSelectorFallsThrough(t *testing.T) {
	w := New(".")
	idv, err := w.Call("createbehavior", nil)
	if err != nil {
		t.Fatal(err)
	}
	tree := int(idv.Number())
	tv := value.Num(float64(tree))
	root, _ := w.Call("btselector", []value.Value{tv})
	cond, _ := w.Call("btcondition", []value.Value{tv, value.Str("Far")})
	act, _ := w.Call("btaction", []value.Value{tv, value.Str("Idle")})
	w.Call("btadd", []value.Value{tv, root, cond})
	w.Call("btadd", []value.Value{tv, root, act})
	w.btStub = func(name string, ent int) int {
		if name == "Far" {
			return 0
		}
		return 0
	}
	st := w.evalBT(w.bts[tree], int(root.Number()), 1)
	if st != btOK {
		t.Fatalf("status %d", st)
	}
	w.btStub = func(name string, ent int) int {
		if name == "Idle" {
			return 1
		}
		return 0
	}
	st = w.evalBT(w.bts[tree], int(root.Number()), 1)
	if st != btRun {
		t.Fatalf("running status %d", st)
	}
}

func TestLargeWorldExampleParses(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "examples", "largeworld.mb"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parse.Parse(string(b)); err != nil {
		t.Fatal(err)
	}
}
