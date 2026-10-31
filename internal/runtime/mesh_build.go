package runtime

import (
	"fmt"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// meshBuild holds a mesh's vertices and triangles on the CPU so scripts
// can read and change them. Changes reach the GPU on UpdateMesh, or by
// themselves before the next frame.
type meshBuild struct {
	mesh  *graphic.Mesh
	pos   []float32
	nor   []float32
	uv    []float32
	uv2   []float32
	col   []float32
	idx   []uint32
	hasN  bool // normals were set or came with the mesh; else UpdateMesh smooths them
	hasC  bool
	hasU2 bool
	dirty bool
}

func (b *meshBuild) verts() int { return len(b.pos) / 3 }

// meshBuilder returns the editable data of a mesh entity, reading it back
// from the GPU geometry the first time (LoadMesh, CreateCube, …).
func (w *World) meshBuilder(id int) (*meshBuild, error) {
	e, err := w.ent(id)
	if err != nil {
		return nil, err
	}
	if e.mesh == nil {
		return nil, fmt.Errorf("entity %d has no mesh of its own (a model with several parts? use GetChild)", id)
	}
	if b := w.meshBuilds[id]; b != nil && b.mesh == e.mesh {
		return b, nil
	}
	if w.meshBuilds == nil {
		w.meshBuilds = map[int]*meshBuild{}
	}
	b := readMeshBuild(e.mesh)
	w.meshBuilds[id] = b
	return b, nil
}

func readMeshBuild(m *graphic.Mesh) *meshBuild {
	b := &meshBuild{mesh: m}
	g := m.GetGeometry()
	read := func(at gls.AttribType, size int) []float32 {
		vbo := g.VBO(at)
		if vbo == nil {
			return nil
		}
		buf := *vbo.Buffer()
		stride, off := vbo.Stride(), vbo.AttribOffset(at) // in floats
		if stride == 0 {
			return nil
		}
		out := make([]float32, 0, len(buf)/stride*size)
		for i := off; i+size <= len(buf); i += stride {
			out = append(out, buf[i:i+size]...)
		}
		return out
	}
	b.pos = read(gls.VertexPosition, 3)
	n := b.verts()
	b.nor = read(gls.VertexNormal, 3)
	b.hasN = len(b.nor) == n*3
	if !b.hasN {
		b.nor = make([]float32, n*3)
	}
	if b.uv = read(gls.VertexTexcoord, 2); len(b.uv) != n*2 {
		b.uv = make([]float32, n*2)
	}
	if b.uv2 = read(gls.VertexTexcoord2, 2); len(b.uv2) == n*2 && n > 0 {
		b.hasU2 = true
	} else {
		b.uv2 = make([]float32, n*2)
	}
	if b.col = read(gls.VertexColor, 3); len(b.col) == n*3 && n > 0 {
		b.hasC = true
	} else {
		b.col = make([]float32, n*3)
		for i := range b.col {
			b.col[i] = 1
		}
	}
	if g.Indexed() {
		b.idx = append([]uint32(nil), g.Indices()...)
	} else {
		for i := 0; i < n; i++ {
			b.idx = append(b.idx, uint32(i))
		}
	}
	return b
}

// smoothNormals averages the face normals around each vertex, weighted by
// face area. Vertices that are not shared stay flat.
func (b *meshBuild) smoothNormals() {
	n := b.verts()
	nor := make([]float32, n*3)
	for t := 0; t+2 < len(b.idx); t += 3 {
		i0, i1, i2 := int(b.idx[t]), int(b.idx[t+1]), int(b.idx[t+2])
		if i0 >= n || i1 >= n || i2 >= n {
			continue
		}
		a := math32.Vector3{X: b.pos[i0*3], Y: b.pos[i0*3+1], Z: b.pos[i0*3+2]}
		v1 := math32.Vector3{X: b.pos[i1*3] - a.X, Y: b.pos[i1*3+1] - a.Y, Z: b.pos[i1*3+2] - a.Z}
		v2 := math32.Vector3{X: b.pos[i2*3] - a.X, Y: b.pos[i2*3+1] - a.Y, Z: b.pos[i2*3+2] - a.Z}
		f := *v1.Cross(&v2)
		for _, i := range [3]int{i0, i1, i2} {
			nor[i*3] += f.X
			nor[i*3+1] += f.Y
			nor[i*3+2] += f.Z
		}
	}
	for i := 0; i < n; i++ {
		v := math32.Vector3{X: nor[i*3], Y: nor[i*3+1], Z: nor[i*3+2]}
		if v.Length() > 0 {
			v.Normalize()
		} else {
			v.Y = 1
		}
		nor[i*3], nor[i*3+1], nor[i*3+2] = v.X, v.Y, v.Z
	}
	b.nor = nor
}

// commit uploads the CPU data to the mesh's geometry in place, so the
// entity keeps its material, transform, and handle.
func (w *World) commitMeshBuild(id int, b *meshBuild) {
	if !b.hasN {
		b.smoothNormals()
	}
	g := b.mesh.GetGeometry()
	set := func(at gls.AttribType, data []float32) {
		arr := math32.ArrayF32(append([]float32(nil), data...))
		if vbo := g.VBO(at); vbo != nil && vbo.AttribCount() == 1 {
			vbo.SetBuffer(arr)
			return
		}
		if g.VBO(at) != nil {
			return // interleaved with another attribute; positions etc. rewrite it below
		}
		g.AddVBO(gls.NewVBO(arr).AddAttrib(at))
	}
	if vbo := g.VBO(gls.VertexPosition); vbo != nil && vbo.AttribCount() > 1 {
		// Interleaved geometry (some loaders): split it into one VBO per attribute.
		*g = *geometry.NewGeometry()
	}
	set(gls.VertexPosition, b.pos)
	set(gls.VertexNormal, b.nor)
	set(gls.VertexTexcoord, b.uv)
	if b.hasU2 || g.VBO(gls.VertexTexcoord2) != nil {
		set(gls.VertexTexcoord2, b.uv2)
	}
	if b.hasC || g.VBO(gls.VertexColor) != nil {
		set(gls.VertexColor, b.col)
	}
	if b.hasC {
		w.useVertexColors(id)
	}
	g.SetIndices(math32.ArrayU32(append([]uint32(nil), b.idx...)))
	g.OperateOnVertices(func(*math32.Vector3) bool { return true }) // bounds changed
	b.mesh.SetRenderable(len(b.idx) >= 3)
	b.dirty = false
}

// useVertexColors switches a mesh to the lit shader, which multiplies the
// material colour by the vertex colour.
func (w *World) useVertexColors(id int) {
	e := w.ents[id]
	if e == nil || e.mesh == nil || e.mat == nil {
		return
	}
	if e.mat.Shader() != "bsshadow" {
		e.mat.SetShader("bsshadow")
	}
	for _, gm := range e.mesh.Materials() {
		if lm, ok := gm.IMaterial().(*litMat); ok {
			lm.vertexColor = true
		}
	}
}

func (w *World) tickMeshBuilds() {
	for id, b := range w.meshBuilds {
		if e := w.ents[id]; e == nil || e.mesh != b.mesh {
			delete(w.meshBuilds, id)
		} else if b.dirty {
			w.commitMeshBuild(id, b)
		}
	}
}

func (w *World) createMesh(parent int) int {
	g := geometry.NewGeometry()
	g.AddVBO(gls.NewVBO(math32.NewArrayF32(0, 0)).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(math32.NewArrayF32(0, 0)).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(math32.NewArrayF32(0, 0)).AddAttrib(gls.VertexTexcoord))
	id := w.tagEnt(w.meshEnt(g, parent), "mesh")
	e := w.ents[id]
	e.mesh.SetRenderable(false)
	if w.meshBuilds == nil {
		w.meshBuilds = map[int]*meshBuild{}
	}
	w.meshBuilds[id] = &meshBuild{mesh: e.mesh}
	return id
}

func (w *World) meshBuildCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	// vert finds mesh a[0] and checks vertex a[1].
	vert := func(a []value.Value) (*meshBuild, int, error) {
		b, err := w.meshBuilder(argI(a, 0, 0))
		if err != nil {
			return nil, 0, err
		}
		i := argI(a, 1, 0)
		if i < 0 || i >= b.verts() {
			return nil, 0, fmt.Errorf("vertex %d out of range (mesh has %d)", i, b.verts())
		}
		return b, i, nil
	}
	f32 := func(a []value.Value, i int, d float64) float32 { return float32(argN(a, i, d)) }
	getV := func(pick func(b *meshBuild, i int) float32) cmd {
		return need(func(a []value.Value) (value.Value, error) {
			b, i, err := vert(a)
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(pick(b, i))), nil
		})
	}
	return map[string]cmd{
		"createmesh": need(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.createMesh(argI(a, 0, 0)))), nil
		}),
		"addvertex": need(func(a []value.Value) (value.Value, error) {
			b, err := w.meshBuilder(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			b.pos = append(b.pos, f32(a, 1, 0), f32(a, 2, 0), f32(a, 3, 0))
			b.nor = append(b.nor, 0, 1, 0)
			b.uv = append(b.uv, f32(a, 4, 0), f32(a, 5, 0))
			b.uv2 = append(b.uv2, 0, 0)
			b.col = append(b.col, 1, 1, 1)
			b.dirty = true
			return value.Num(float64(b.verts() - 1)), nil
		}),
		"addtriangle": need(func(a []value.Value) (value.Value, error) {
			b, err := w.meshBuilder(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			var t [3]uint32
			for k := range t {
				v := argI(a, 1+k, 0)
				if v < 0 || v >= b.verts() {
					return value.Value{}, fmt.Errorf("AddTriangle: vertex %d out of range (mesh has %d)", v, b.verts())
				}
				t[k] = uint32(v)
			}
			b.idx = append(b.idx, t[:]...)
			b.dirty = true
			return value.Num(float64(len(b.idx)/3 - 1)), nil
		}),
		"vertexcoords": need(func(a []value.Value) (value.Value, error) {
			b, i, err := vert(a)
			if err != nil {
				return value.Value{}, err
			}
			b.pos[i*3], b.pos[i*3+1], b.pos[i*3+2] = f32(a, 2, 0), f32(a, 3, 0), f32(a, 4, 0)
			b.dirty = true
			return z()
		}),
		"vertexnormal": need(func(a []value.Value) (value.Value, error) {
			b, i, err := vert(a)
			if err != nil {
				return value.Value{}, err
			}
			v := math32.Vector3{X: f32(a, 2, 0), Y: f32(a, 3, 1), Z: f32(a, 4, 0)}
			v.Normalize()
			b.nor[i*3], b.nor[i*3+1], b.nor[i*3+2] = v.X, v.Y, v.Z
			b.hasN, b.dirty = true, true
			return z()
		}),
		"vertexcolor": need(func(a []value.Value) (value.Value, error) {
			b, i, err := vert(a)
			if err != nil {
				return value.Value{}, err
			}
			b.col[i*3], b.col[i*3+1], b.col[i*3+2] = f32(a, 2, 255)/255, f32(a, 3, 255)/255, f32(a, 4, 255)/255
			b.hasC, b.dirty = true, true
			return z()
		}),
		"vertextexcoords": need(func(a []value.Value) (value.Value, error) {
			b, i, err := vert(a)
			if err != nil {
				return value.Value{}, err
			}
			uv := b.uv
			if argI(a, 4, 0) == 1 {
				uv, b.hasU2 = b.uv2, true
			}
			uv[i*2], uv[i*2+1] = f32(a, 2, 0), f32(a, 3, 0)
			b.dirty = true
			return z()
		}),
		"vertexx":     getV(func(b *meshBuild, i int) float32 { return b.pos[i*3] }),
		"vertexy":     getV(func(b *meshBuild, i int) float32 { return b.pos[i*3+1] }),
		"vertexz":     getV(func(b *meshBuild, i int) float32 { return b.pos[i*3+2] }),
		"vertexnx":    getV(func(b *meshBuild, i int) float32 { return b.nor[i*3] }),
		"vertexny":    getV(func(b *meshBuild, i int) float32 { return b.nor[i*3+1] }),
		"vertexnz":    getV(func(b *meshBuild, i int) float32 { return b.nor[i*3+2] }),
		"vertexred":   getV(func(b *meshBuild, i int) float32 { return b.col[i*3] * 255 }),
		"vertexgreen": getV(func(b *meshBuild, i int) float32 { return b.col[i*3+1] * 255 }),
		"vertexblue":  getV(func(b *meshBuild, i int) float32 { return b.col[i*3+2] * 255 }),
		"vertexu": need(func(a []value.Value) (value.Value, error) {
			b, i, err := vert(a)
			if err != nil {
				return value.Value{}, err
			}
			if argI(a, 2, 0) == 1 {
				return value.Num(float64(b.uv2[i*2])), nil
			}
			return value.Num(float64(b.uv[i*2])), nil
		}),
		"vertexv": need(func(a []value.Value) (value.Value, error) {
			b, i, err := vert(a)
			if err != nil {
				return value.Value{}, err
			}
			if argI(a, 2, 0) == 1 {
				return value.Num(float64(b.uv2[i*2+1])), nil
			}
			return value.Num(float64(b.uv[i*2+1])), nil
		}),
		"countvertices": need(func(a []value.Value) (value.Value, error) {
			b, err := w.meshBuilder(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(b.verts())), nil
		}),
		"counttriangles": need(func(a []value.Value) (value.Value, error) {
			b, err := w.meshBuilder(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(len(b.idx) / 3)), nil
		}),
		"trianglevertex": need(func(a []value.Value) (value.Value, error) {
			b, err := w.meshBuilder(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			t, c := argI(a, 1, 0), argI(a, 2, 0)
			if t < 0 || t >= len(b.idx)/3 || c < 0 || c > 2 {
				return value.Value{}, fmt.Errorf("TriangleVertex: triangle %d corner %d out of range (mesh has %d triangles)", t, c, len(b.idx)/3)
			}
			return value.Num(float64(b.idx[t*3+c])), nil
		}),
		"updatemesh": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			b, err := w.meshBuilder(id)
			if err != nil {
				return value.Value{}, err
			}
			w.commitMeshBuild(id, b)
			return z()
		}),
		"updatenormals": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			b, err := w.meshBuilder(id)
			if err != nil {
				return value.Value{}, err
			}
			b.smoothNormals()
			b.hasN, b.dirty = true, true
			return z()
		}),
		"clearmesh": need(func(a []value.Value) (value.Value, error) {
			b, err := w.meshBuilder(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			b.pos, b.nor, b.uv, b.uv2, b.col, b.idx = nil, nil, nil, nil, nil, nil
			b.hasN, b.hasC, b.hasU2, b.dirty = false, false, false, true
			return z()
		}),
	}
}
