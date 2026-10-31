package runtime

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
)

// rawMesh is an indexed triangle list read from STL or PLY. nor, uv, and col
// are per vertex and empty when the file has none.
type rawMesh struct {
	pos, nor, uv, col []float32
	idx               []uint32
}

func (w *World) loadRawMeshAt(path string, parse func([]byte) (rawMesh, error)) (*geometry.Geometry, bool, error) {
	b, err := readFileBytes(path)
	if err != nil {
		return nil, false, fmt.Errorf("LoadMesh: %w", err)
	}
	m, err := parse(b)
	if err != nil {
		return nil, false, fmt.Errorf("LoadMesh: %w", err)
	}
	g, err := m.geometry()
	if err != nil {
		return nil, false, fmt.Errorf("LoadMesh: %w", err)
	}
	return g, len(m.col) > 0, nil
}

func (w *World) loadSTLAt(path string, parent int) (int, error) {
	return w.loadRawEntity(path, parent, parseSTL)
}

func (w *World) loadPLYAt(path string, parent int) (int, error) {
	return w.loadRawEntity(path, parent, parsePLY)
}

func (w *World) loadRawEntity(path string, parent int, parse func([]byte) (rawMesh, error)) (int, error) {
	geom, colored, err := w.loadRawMeshAt(path, parse)
	if err != nil {
		return 0, err
	}
	id := w.meshEnt(geom, parent)
	if e := w.ents[id]; e != nil && e.mesh != nil {
		optimizeGeometry(e.mesh.GetGeometry())
		e.shadowGeomDirty = true
		// Vertex colours draw through the lit shader, as merged level cells do.
		if colored {
			if lm, ok := e.mesh.Materials()[0].IMaterial().(*litMat); ok && e.mat != nil && e.mat.Shader() == "bsshadow" {
				e.mat.SetColor(&math32.Color{R: 1, G: 1, B: 1})
				lm.vertexColor = true
			}
		}
	}
	return id, nil
}

func (m rawMesh) geometry() (*geometry.Geometry, error) {
	nvert := len(m.pos) / 3
	if nvert < 3 || len(m.pos)%3 != 0 || len(m.idx) < 3 {
		return nil, fmt.Errorf("mesh is empty")
	}
	for _, v := range m.idx {
		if int(v) >= nvert {
			return nil, fmt.Errorf("index %d past %d vertices", v, nvert)
		}
	}
	g := geometry.NewGeometry()
	ip := math32.NewArrayU32(0, len(m.idx))
	ip.Append(m.idx...)
	vp := math32.NewArrayF32(0, len(m.pos))
	vp.Append(m.pos...)
	nor := math32.NewArrayF32(0, len(m.pos))
	if len(m.nor) == len(m.pos) {
		nor.Append(m.nor...)
	} else {
		nor.Append(smoothNormals(m.pos, m.idx)...)
	}
	uvs := math32.NewArrayF32(0, nvert*2)
	if len(m.uv) == nvert*2 {
		uvs.Append(m.uv...)
	} else {
		for i := 0; i < nvert; i++ {
			uvs.Append(m.pos[i*3], m.pos[i*3+2])
		}
	}
	g.SetIndices(ip)
	g.AddVBO(gls.NewVBO(vp).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	if len(m.col) == nvert*3 {
		col := math32.NewArrayF32(0, len(m.col))
		col.Append(m.col...)
		g.AddVBO(gls.NewVBO(col).AddAttrib(gls.VertexColor))
	}
	return g, nil
}

// smoothNormals averages the area-weighted normals of counter-clockwise
// faces. (geometry.CalculateNormals assumes the opposite winding.)
func smoothNormals(pos []float32, idx []uint32) []float32 {
	nor := make([]float32, len(pos))
	for t := 0; t+2 < len(idx); t += 3 {
		a, b, c := idx[t]*3, idx[t+1]*3, idx[t+2]*3
		var e1, e2, n math32.Vector3
		e1.Set(pos[b]-pos[a], pos[b+1]-pos[a+1], pos[b+2]-pos[a+2])
		e2.Set(pos[c]-pos[a], pos[c+1]-pos[a+1], pos[c+2]-pos[a+2])
		n.CrossVectors(&e1, &e2)
		for _, v := range [3]uint32{a, b, c} {
			nor[v] += n.X
			nor[v+1] += n.Y
			nor[v+2] += n.Z
		}
	}
	for i := 0; i+2 < len(nor); i += 3 {
		v := math32.Vector3{X: nor[i], Y: nor[i+1], Z: nor[i+2]}
		if v.Length() > 0 {
			v.Normalize()
		} else {
			v.Y = 1
		}
		nor[i], nor[i+1], nor[i+2] = v.X, v.Y, v.Z
	}
	return nor
}

// ---- STL ----

// stlBuilder welds corners that share a position and a facet normal, so flat
// regions share vertices while edges stay sharp.
type stlBuilder struct {
	m    rawMesh
	seen map[[6]float32]uint32
}

func (s *stlBuilder) tri(n [3]float32, v [3][3]float32) {
	if n == ([3]float32{}) {
		var a, b, c math32.Vector3
		a.Set(v[1][0]-v[0][0], v[1][1]-v[0][1], v[1][2]-v[0][2])
		b.Set(v[2][0]-v[0][0], v[2][1]-v[0][1], v[2][2]-v[0][2])
		c.CrossVectors(&a, &b).Normalize()
		n = [3]float32{c.X, c.Y, c.Z}
	}
	if s.seen == nil {
		s.seen = map[[6]float32]uint32{}
	}
	for _, p := range v {
		key := [6]float32{p[0], p[1], p[2], n[0], n[1], n[2]}
		i, ok := s.seen[key]
		if !ok {
			i = uint32(len(s.m.pos) / 3)
			s.seen[key] = i
			s.m.pos = append(s.m.pos, p[0], p[1], p[2])
			s.m.nor = append(s.m.nor, n[0], n[1], n[2])
		}
		s.m.idx = append(s.m.idx, i)
	}
}

func parseSTL(b []byte) (rawMesh, error) {
	if len(b) >= 84 {
		n := int(binary.LittleEndian.Uint32(b[80:84]))
		if 84+50*n == len(b) {
			return parseSTLBinary(b, n)
		}
	}
	if bytes.HasPrefix(bytes.TrimLeft(b, " \t\r\n"), []byte("solid")) {
		return parseSTLASCII(b)
	}
	if len(b) >= 84 {
		n := int(binary.LittleEndian.Uint32(b[80:84]))
		if 84+50*n <= len(b) {
			return parseSTLBinary(b, n)
		}
	}
	return rawMesh{}, fmt.Errorf("stl: not an ASCII or binary STL file")
}

func parseSTLBinary(b []byte, n int) (rawMesh, error) {
	var s stlBuilder
	f := func(o int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b[o:])) }
	for t := 0; t < n; t++ {
		o := 84 + t*50
		nrm := [3]float32{f(o), f(o + 4), f(o + 8)}
		var v [3][3]float32
		for k := 0; k < 3; k++ {
			p := o + 12 + k*12
			v[k] = [3]float32{f(p), f(p + 4), f(p + 8)}
		}
		s.tri(nrm, v)
	}
	if len(s.m.idx) == 0 {
		return rawMesh{}, fmt.Errorf("stl: no triangles")
	}
	return s.m, nil
}

func parseSTLASCII(b []byte) (rawMesh, error) {
	var s stlBuilder
	tok := strings.Fields(string(b))
	num := func(i int) (float32, error) {
		if i >= len(tok) {
			return 0, fmt.Errorf("stl: truncated")
		}
		v, err := strconv.ParseFloat(tok[i], 32)
		return float32(v), err
	}
	vec := func(i int) ([3]float32, error) {
		var out [3]float32
		for k := range out {
			v, err := num(i + k)
			if err != nil {
				return out, fmt.Errorf("stl: bad number %q", tok[min(i+k, len(tok)-1)])
			}
			out[k] = v
		}
		return out, nil
	}
	var nrm [3]float32
	var loop [][3]float32
	for i := 0; i < len(tok); i++ {
		switch strings.ToLower(tok[i]) {
		case "facet":
			nrm = [3]float32{}
			if i+1 < len(tok) && strings.EqualFold(tok[i+1], "normal") {
				v, err := vec(i + 2)
				if err != nil {
					return rawMesh{}, err
				}
				nrm = v
				i += 4
			}
		case "vertex":
			v, err := vec(i + 1)
			if err != nil {
				return rawMesh{}, err
			}
			loop = append(loop, v)
			i += 3
		case "endloop":
			for k := 1; k+1 < len(loop); k++ {
				s.tri(nrm, [3][3]float32{loop[0], loop[k], loop[k+1]})
			}
			loop = loop[:0]
		}
	}
	if len(s.m.idx) == 0 {
		return rawMesh{}, fmt.Errorf("stl: no triangles")
	}
	return s.m, nil
}

// ---- PLY ----

type plyProp struct {
	name      string
	typ       string // scalar type, or the item type of a list
	countType string // set for list properties
}

type plyElement struct {
	name  string
	count int
	props []plyProp
}

func plyTypeSize(t string) int {
	switch t {
	case "char", "int8", "uchar", "uint8":
		return 1
	case "short", "int16", "ushort", "uint16":
		return 2
	case "int", "int32", "uint", "uint32", "float", "float32":
		return 4
	case "double", "float64":
		return 8
	}
	return 0
}

// plyReader yields property values from an ASCII token stream or a
// little-endian binary body.
type plyReader struct {
	bin  []byte
	off  int
	tok  []string
	tpos int
}

func (r *plyReader) value(t string) (float64, error) {
	if r.tok != nil {
		if r.tpos >= len(r.tok) {
			return 0, fmt.Errorf("ply: truncated data")
		}
		s := r.tok[r.tpos]
		r.tpos++
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("ply: bad number %q", s)
		}
		return v, nil
	}
	n := plyTypeSize(t)
	if r.off+n > len(r.bin) {
		return 0, fmt.Errorf("ply: truncated data")
	}
	b := r.bin[r.off : r.off+n]
	r.off += n
	le := binary.LittleEndian
	switch t {
	case "char", "int8":
		return float64(int8(b[0])), nil
	case "uchar", "uint8":
		return float64(b[0]), nil
	case "short", "int16":
		return float64(int16(le.Uint16(b))), nil
	case "ushort", "uint16":
		return float64(le.Uint16(b)), nil
	case "int", "int32":
		return float64(int32(le.Uint32(b))), nil
	case "uint", "uint32":
		return float64(le.Uint32(b)), nil
	case "float", "float32":
		return float64(math.Float32frombits(le.Uint32(b))), nil
	case "double", "float64":
		return math.Float64frombits(le.Uint64(b)), nil
	}
	return 0, fmt.Errorf("ply: unknown type %q", t)
}

func parsePLY(b []byte) (rawMesh, error) {
	if !bytes.HasPrefix(b, []byte("ply")) {
		return rawMesh{}, fmt.Errorf("ply: missing magic")
	}
	end := bytes.Index(b, []byte("end_header"))
	if end < 0 {
		return rawMesh{}, fmt.Errorf("ply: no end_header")
	}
	nl := bytes.IndexByte(b[end:], '\n')
	if nl < 0 {
		return rawMesh{}, fmt.Errorf("ply: no data")
	}
	body := b[end+nl+1:]
	var elems []*plyElement
	format := ""
	for _, line := range strings.Split(string(b[:end]), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case "format":
			if len(f) > 1 {
				format = f[1]
			}
		case "element":
			if len(f) < 3 {
				return rawMesh{}, fmt.Errorf("ply: bad element line %q", line)
			}
			n, err := strconv.Atoi(f[2])
			if err != nil || n < 0 {
				return rawMesh{}, fmt.Errorf("ply: bad element count %q", f[2])
			}
			elems = append(elems, &plyElement{name: f[1], count: n})
		case "property":
			if len(elems) == 0 {
				return rawMesh{}, fmt.Errorf("ply: property before element")
			}
			e := elems[len(elems)-1]
			var p plyProp
			if len(f) >= 5 && f[1] == "list" {
				p = plyProp{countType: f[2], typ: f[3], name: f[4]}
				if plyTypeSize(p.countType) == 0 {
					return rawMesh{}, fmt.Errorf("ply: bad list count type %q", p.countType)
				}
			} else if len(f) >= 3 {
				p = plyProp{typ: f[1], name: f[2]}
			} else {
				return rawMesh{}, fmt.Errorf("ply: bad property line %q", line)
			}
			if plyTypeSize(p.typ) == 0 {
				return rawMesh{}, fmt.Errorf("ply: bad property type %q", p.typ)
			}
			e.props = append(e.props, p)
		}
	}
	r := &plyReader{}
	switch format {
	case "ascii":
		r.tok = strings.Fields(string(body))
	case "binary_little_endian":
		r.bin = body
	case "binary_big_endian":
		return rawMesh{}, fmt.Errorf("ply: big-endian binary is not supported; re-export as little-endian or ASCII")
	default:
		return rawMesh{}, fmt.Errorf("ply: unknown format %q", format)
	}

	var m rawMesh
	var hasNor, hasUV, hasCol bool
	colScale := 1.0
	for _, e := range elems {
		col := map[string]int{}
		for i, p := range e.props {
			col[p.name] = i
		}
		isVert := e.name == "vertex"
		isFace := e.name == "face"
		if isVert {
			_, hasNor = col["nx"]
			hasUV = plyHasAny(col, "u", "s", "texture_u", "texture_s")
			hasCol = plyHasAny(col, "red", "r", "diffuse_red")
			for _, nm := range []string{"red", "r", "diffuse_red"} {
				if i, ok := col[nm]; ok {
					if plyTypeSize(e.props[i].typ) == 1 {
						colScale = 1.0 / 255 // uchar 0–255
					}
					break
				}
			}
		}
		vals := make([]float64, len(e.props))
		var list []float64
		for n := 0; n < e.count; n++ {
			for i, p := range e.props {
				if p.countType == "" {
					v, err := r.value(p.typ)
					if err != nil {
						return rawMesh{}, err
					}
					vals[i] = v
					continue
				}
				c, err := r.value(p.countType)
				if err != nil {
					return rawMesh{}, err
				}
				if c < 0 || c > 1<<16 {
					return rawMesh{}, fmt.Errorf("ply: bad list length %v", c)
				}
				items := list[:0]
				for k := 0; k < int(c); k++ {
					v, err := r.value(p.typ)
					if err != nil {
						return rawMesh{}, err
					}
					items = append(items, v)
				}
				if isFace && (p.name == "vertex_indices" || p.name == "vertex_index") {
					for k := 1; k+1 < len(items); k++ {
						m.idx = append(m.idx, uint32(items[0]), uint32(items[k]), uint32(items[k+1]))
					}
				}
				list = items
			}
			if !isVert {
				continue
			}
			get := func(names ...string) float64 {
				for _, nm := range names {
					if i, ok := col[nm]; ok {
						return vals[i]
					}
				}
				return 0
			}
			m.pos = append(m.pos, float32(get("x")), float32(get("y")), float32(get("z")))
			if hasNor {
				m.nor = append(m.nor, float32(get("nx")), float32(get("ny")), float32(get("nz")))
			}
			if hasUV {
				m.uv = append(m.uv, float32(get("u", "s", "texture_u", "texture_s")), float32(get("v", "t", "texture_v", "texture_t")))
			}
			if hasCol {
				m.col = append(m.col,
					float32(get("red", "r", "diffuse_red")*colScale),
					float32(get("green", "g", "diffuse_green")*colScale),
					float32(get("blue", "b", "diffuse_blue")*colScale))
			}
		}
	}
	if len(m.pos) == 0 {
		return rawMesh{}, fmt.Errorf("ply: no vertices")
	}
	if len(m.idx) == 0 {
		return rawMesh{}, fmt.Errorf("ply: no faces (point clouds are not loaded)")
	}
	return m, nil
}

func plyHasAny(col map[string]int, names ...string) bool {
	for _, n := range names {
		if _, ok := col[n]; ok {
			return true
		}
	}
	return false
}
