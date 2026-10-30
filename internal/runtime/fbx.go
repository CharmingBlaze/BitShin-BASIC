package runtime

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
)

// fbxMesh is one triangulated mesh from an FBX file.
// Clusters carry skin indexes and weights. Cameras and animation clips are not read.
type fbxMesh struct {
	verts     []float64
	polys     []int32
	normals   []float64
	uv        []float64
	upAxis    int // 0 X, 1 Y, 2 Z. 1 when the file does not say.
	unit      float64
	gotUp     bool
	gotUnit   bool
	clusters  []fbxCluster
	limbs     []fbxLimb
	pendingIx []int32
}

func (w *World) loadFBXAt(path string, parent int) (int, error) {
	b, err := readFileBytes(path)
	if err != nil {
		return 0, fmt.Errorf("LoadMesh: %w", err)
	}
	mesh, err := parseFBX(b)
	if err != nil {
		return 0, fmt.Errorf("LoadMesh: %w", err)
	}
	geom, err := mesh.geometry()
	if err != nil {
		return 0, fmt.Errorf("LoadMesh: %w", err)
	}
	id := w.meshEnt(geom, parent)
	if e := w.ents[id]; e != nil && e.mesh != nil {
		if len(mesh.clusters) > 0 {
			w.attachFBXSkin(e, mesh)
		} else {
			optimizeGeometry(e.mesh.GetGeometry())
		}
		e.shadowGeomDirty = true
	}
	return id, nil
}

func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func parseFBX(b []byte) (fbxMesh, error) {
	if bytes.HasPrefix(b, []byte("Kaydara FBX Binary")) {
		return parseFBXBinary(b)
	}
	return parseFBXASCII(string(b))
}

func parseFBXASCII(src string) (fbxMesh, error) {
	var m fbxMesh
	m.upAxis = 1
	m.unit = 1
	verts, ok := asciiArray(src, "Vertices")
	if !ok || len(verts) < 9 || len(verts)%3 != 0 {
		return m, fmt.Errorf("fbx: no mesh vertices")
	}
	m.verts = verts
	if polys, ok := asciiArray(src, "PolygonVertexIndex"); ok {
		m.polys = make([]int32, len(polys))
		for i, v := range polys {
			m.polys[i] = int32(v)
		}
	}
	if len(m.polys) < 3 {
		return m, fmt.Errorf("fbx: no polygons")
	}
	if n, ok := asciiArray(src, "Normals"); ok {
		m.normals = n
	}
	if u, ok := asciiArray(src, "UV"); ok {
		m.uv = u
	}
	if axis, ok := asciiProp(src, "UpAxis"); ok {
		m.upAxis = int(axis)
		m.gotUp = true
	}
	if unit, ok := asciiProp(src, "UnitScaleFactor"); ok && unit > 0 {
		m.unit = unit
		m.gotUnit = true
	}
	m.clusters, m.limbs = parseFBXSkinASCII(src)
	return m, nil
}

func asciiArray(src, key string) ([]float64, bool) {
	i := 0
	for {
		j := strings.Index(src[i:], key+":")
		if j < 0 {
			return nil, false
		}
		j += i + len(key) + 1
		rest := src[j:]
		a := strings.Index(rest, "a:")
		if a < 0 || a > 80 {
			i = j
			continue
		}
		body := rest[a+2:]
		end := strings.IndexAny(body, "}\n")
		if end < 0 {
			end = len(body)
		}
		// Arrays may span lines until the closing brace of the property.
		brace := strings.Index(rest[a:], "}")
		if brace > end {
			end = brace
		}
		nums := splitNumbers(body[:end])
		if len(nums) > 0 {
			return nums, true
		}
		i = j
	}
}

func asciiProp(src, key string) (float64, bool) {
	q := `"` + key + `"`
	i := strings.Index(src, q)
	if i < 0 {
		return 0, false
	}
	line := src[i:]
	if n := strings.IndexByte(line, '\n'); n >= 0 {
		line = line[:n]
	}
	nums := splitNumbers(line)
	if len(nums) == 0 {
		return 0, false
	}
	return nums[len(nums)-1], true
}

func splitNumbers(s string) []float64 {
	var out []float64
	s = strings.ReplaceAll(s, ",", " ")
	for _, f := range strings.Fields(s) {
		f = strings.TrimFunc(f, func(r rune) bool {
			return r != '-' && r != '+' && r != '.' && !unicode.IsDigit(r) && r != 'e' && r != 'E'
		})
		if f == "" || f == "-" || f == "+" || f == "." {
			continue
		}
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func parseFBXBinary(b []byte) (fbxMesh, error) {
	if len(b) < 27 {
		return fbxMesh{}, fmt.Errorf("fbx: truncated header")
	}
	ver := binary.LittleEndian.Uint32(b[23:27])
	var m fbxMesh
	m.upAxis = 1
	m.unit = 1
	pos := 27
	for pos+13 <= len(b) {
		next, err := readFBXNode(b, pos, ver, &m)
		if err != nil {
			return m, err
		}
		if next <= pos {
			break
		}
		pos = next
		if m.verts != nil && m.polys != nil && pos >= len(b) {
			break
		}
	}
	if len(m.verts) < 9 || len(m.polys) < 3 {
		return m, fmt.Errorf("fbx: no mesh vertices")
	}
	return m, nil
}

func readFBXNode(b []byte, pos int, ver uint32, m *fbxMesh) (int, error) {
	wide := ver >= 7500
	hdr := 13
	if wide {
		hdr = 25
	}
	if pos+hdr > len(b) {
		return len(b), nil
	}
	var end uint64
	var nprops uint64
	var propLen uint64
	if wide {
		end = binary.LittleEndian.Uint64(b[pos:])
		nprops = binary.LittleEndian.Uint64(b[pos+8:])
		propLen = binary.LittleEndian.Uint64(b[pos+16:])
	} else {
		end = uint64(binary.LittleEndian.Uint32(b[pos:]))
		nprops = uint64(binary.LittleEndian.Uint32(b[pos+4:]))
		propLen = uint64(binary.LittleEndian.Uint32(b[pos+8:]))
	}
	nameLen := int(b[pos+hdr-1])
	if end == 0 {
		return pos + hdr, nil
	}
	if end > uint64(len(b)) || int(end) < pos+hdr {
		return 0, fmt.Errorf("fbx: bad node offset")
	}
	p := pos + hdr
	if p+nameLen > len(b) {
		return 0, fmt.Errorf("fbx: bad node name")
	}
	name := string(b[p : p+nameLen])
	p += nameLen
	props := make([]fbxProp, 0, nprops)
	propEnd := p + int(propLen)
	if propEnd > int(end) || propEnd > len(b) {
		return 0, fmt.Errorf("fbx: bad property list")
	}
	for i := uint64(0); i < nprops; i++ {
		pr, np, err := readFBXProp(b, p)
		if err != nil {
			return 0, err
		}
		props = append(props, pr)
		p = np
	}
	p = propEnd
	noteFBXNode(name, props, m)
	for p < int(end) {
		next, err := readFBXNode(b, p, ver, m)
		if err != nil {
			return 0, err
		}
		if next <= p {
			break
		}
		p = next
	}
	return int(end), nil
}

type fbxProp struct {
	s  string
	i  int32
	d  float64
	df []float64
	ii []int32
}

func readFBXProp(b []byte, p int) (fbxProp, int, error) {
	if p >= len(b) {
		return fbxProp{}, 0, fmt.Errorf("fbx: truncated property")
	}
	kind := b[p]
	p++
	need := func(n int) error {
		if p+n > len(b) {
			return fmt.Errorf("fbx: truncated property")
		}
		return nil
	}
	switch kind {
	case 'Y':
		if err := need(2); err != nil {
			return fbxProp{}, 0, err
		}
		return fbxProp{i: int32(int16(binary.LittleEndian.Uint16(b[p:])))}, p + 2, nil
	case 'C':
		if err := need(1); err != nil {
			return fbxProp{}, 0, err
		}
		return fbxProp{i: int32(b[p])}, p + 1, nil
	case 'I':
		if err := need(4); err != nil {
			return fbxProp{}, 0, err
		}
		return fbxProp{i: int32(binary.LittleEndian.Uint32(b[p:]))}, p + 4, nil
	case 'F':
		if err := need(4); err != nil {
			return fbxProp{}, 0, err
		}
		return fbxProp{d: float64(math.Float32frombits(binary.LittleEndian.Uint32(b[p:])))}, p + 4, nil
	case 'D':
		if err := need(8); err != nil {
			return fbxProp{}, 0, err
		}
		return fbxProp{d: math.Float64frombits(binary.LittleEndian.Uint64(b[p:]))}, p + 8, nil
	case 'L':
		if err := need(8); err != nil {
			return fbxProp{}, 0, err
		}
		return fbxProp{i: int32(binary.LittleEndian.Uint64(b[p:]))}, p + 8, nil
	case 'S', 'R':
		if err := need(4); err != nil {
			return fbxProp{}, 0, err
		}
		n := int(binary.LittleEndian.Uint32(b[p:]))
		p += 4
		if err := need(n); err != nil {
			return fbxProp{}, 0, err
		}
		return fbxProp{s: string(b[p : p+n])}, p + n, nil
	case 'f', 'd', 'i', 'l', 'b':
		if err := need(12); err != nil {
			return fbxProp{}, 0, err
		}
		count := int(binary.LittleEndian.Uint32(b[p:]))
		enc := binary.LittleEndian.Uint32(b[p+4:])
		comp := int(binary.LittleEndian.Uint32(b[p+8:]))
		p += 12
		raw, np, err := fbxArrayBytes(b, p, enc, comp)
		if err != nil {
			return fbxProp{}, 0, err
		}
		pr, err := decodeFBXArray(kind, count, raw)
		return pr, np, err
	default:
		return fbxProp{}, 0, fmt.Errorf("fbx: unknown property %q", kind)
	}
}

func fbxArrayBytes(b []byte, p int, enc uint32, comp int) ([]byte, int, error) {
	if enc == 0 {
		if p+comp > len(b) {
			return nil, 0, fmt.Errorf("fbx: truncated array")
		}
		return b[p : p+comp], p + comp, nil
	}
	if enc != 1 {
		return nil, 0, fmt.Errorf("fbx: unknown array encoding %d", enc)
	}
	if p+comp > len(b) {
		return nil, 0, fmt.Errorf("fbx: truncated compressed array")
	}
	zr, err := zlib.NewReader(bytes.NewReader(b[p : p+comp]))
	if err != nil {
		return nil, 0, fmt.Errorf("fbx: %w", err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, 0, fmt.Errorf("fbx: %w", err)
	}
	return raw, p + comp, nil
}

func decodeFBXArray(kind byte, count int, raw []byte) (fbxProp, error) {
	var pr fbxProp
	switch kind {
	case 'd':
		if len(raw) < count*8 {
			return pr, fmt.Errorf("fbx: short double array")
		}
		pr.df = make([]float64, count)
		for i := 0; i < count; i++ {
			pr.df[i] = math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8:]))
		}
	case 'f':
		if len(raw) < count*4 {
			return pr, fmt.Errorf("fbx: short float array")
		}
		pr.df = make([]float64, count)
		for i := 0; i < count; i++ {
			pr.df[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:])))
		}
	case 'i':
		if len(raw) < count*4 {
			return pr, fmt.Errorf("fbx: short int array")
		}
		pr.ii = make([]int32, count)
		for i := 0; i < count; i++ {
			pr.ii[i] = int32(binary.LittleEndian.Uint32(raw[i*4:]))
		}
	case 'l':
		if len(raw) < count*8 {
			return pr, fmt.Errorf("fbx: short long array")
		}
		pr.ii = make([]int32, count)
		for i := 0; i < count; i++ {
			pr.ii[i] = int32(binary.LittleEndian.Uint64(raw[i*8:]))
		}
	case 'b':
		if len(raw) < count {
			return pr, fmt.Errorf("fbx: short bool array")
		}
	default:
		return pr, fmt.Errorf("fbx: bad array")
	}
	return pr, nil
}

func noteFBXNode(name string, props []fbxProp, m *fbxMesh) {
	switch name {
	case "Vertices":
		if len(m.verts) == 0 && len(props) > 0 && len(props[0].df) >= 9 {
			m.verts = props[0].df
		}
	case "PolygonVertexIndex":
		if len(m.polys) == 0 && len(props) > 0 && len(props[0].ii) >= 3 {
			m.polys = props[0].ii
		}
	case "Normals":
		if len(m.normals) == 0 && len(props) > 0 {
			m.normals = props[0].df
		}
	case "UV":
		if len(m.uv) == 0 && len(props) > 0 {
			m.uv = props[0].df
		}
	case "Indexes":
		if len(props) > 0 && len(props[0].ii) > 0 {
			m.pendingIx = append([]int32(nil), props[0].ii...)
		}
	case "Weights":
		if len(m.pendingIx) > 0 && len(props) > 0 && len(props[0].df) > 0 {
			n := len(m.pendingIx)
			if len(props[0].df) < n {
				n = len(props[0].df)
			}
			c := fbxCluster{indexes: append([]int32(nil), m.pendingIx[:n]...), weights: append([]float64(nil), props[0].df[:n]...)}
			c.name = fmt.Sprintf("joint%d", len(m.clusters))
			m.clusters = append(m.clusters, c)
			m.pendingIx = nil
		}
	case "P":
		if len(props) == 0 {
			return
		}
		key := props[0].s
		switch key {
		case "UpAxis":
			m.upAxis = int(props[len(props)-1].i)
			if props[len(props)-1].d != 0 && props[len(props)-1].i == 0 {
				m.upAxis = int(props[len(props)-1].d)
			}
			m.gotUp = true
		case "UnitScaleFactor":
			u := props[len(props)-1].d
			if u == 0 {
				u = float64(props[len(props)-1].i)
			}
			if u > 0 {
				m.unit = u
				m.gotUnit = true
			}
		}
	}
}

func (m fbxMesh) geometry() (*geometry.Geometry, error) {
	if len(m.verts) < 9 || len(m.verts)%3 != 0 || len(m.polys) < 3 {
		return nil, fmt.Errorf("fbx: mesh is empty")
	}
	scale := 1.0
	if m.unit >= 50 {
		scale = m.unit
	}
	nvert := len(m.verts) / 3
	pos := make([]float32, len(m.verts))
	for i := 0; i < nvert; i++ {
		x := m.verts[i*3] / scale
		y := m.verts[i*3+1] / scale
		z := m.verts[i*3+2] / scale
		if m.upAxis == 2 {
			y, z = z, -y
		}
		pos[i*3] = float32(x)
		pos[i*3+1] = float32(y)
		pos[i*3+2] = float32(z)
	}
	var poly []int
	var idx []uint32
	flush := func() {
		if len(poly) < 3 {
			poly = poly[:0]
			return
		}
		for i := 1; i < len(poly)-1; i++ {
			idx = append(idx, uint32(poly[0]), uint32(poly[i+1]), uint32(poly[i]))
		}
		poly = poly[:0]
	}
	for _, raw := range m.polys {
		if raw < 0 {
			poly = append(poly, int(^raw))
			flush()
			continue
		}
		poly = append(poly, int(raw))
	}
	flush()
	if len(idx) < 3 {
		return nil, fmt.Errorf("fbx: no triangles")
	}
	for _, v := range idx {
		if int(v) >= nvert {
			return nil, fmt.Errorf("fbx: index %d past %d vertices", v, nvert)
		}
	}
	g := geometry.NewGeometry()
	ip := math32.NewArrayU32(0, len(idx))
	ip.Append(idx...)
	vp := math32.NewArrayF32(0, len(pos))
	vp.Append(pos...)
	var nor math32.ArrayF32
	if len(m.normals) == len(m.verts) {
		nor = math32.NewArrayF32(0, len(pos))
		for i := 0; i < nvert; i++ {
			x := float32(m.normals[i*3])
			y := float32(m.normals[i*3+1])
			z := float32(m.normals[i*3+2])
			if m.upAxis == 2 {
				y, z = z, -y
			}
			nor.Append(x, y, z)
		}
	} else {
		nor = math32.NewArrayF32(len(pos), len(pos))
		nor = geometry.CalculateNormals(ip, vp, nor)
	}
	uvs := math32.NewArrayF32(0, nvert*2)
	if len(m.uv) == nvert*2 {
		for i := 0; i < nvert; i++ {
			uvs.Append(float32(m.uv[i*2]), float32(m.uv[i*2+1]))
		}
	} else {
		for i := 0; i < nvert; i++ {
			uvs.Append(pos[i*3], pos[i*3+2])
		}
	}
	g.SetIndices(ip)
	g.AddVBO(gls.NewVBO(vp).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g, nil
}
