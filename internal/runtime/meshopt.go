package runtime

import (
	"fmt"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/meshoptim"
	"moonbasic/internal/meshsimplify"
	"moonbasic/internal/value"
)

// OptimizeMesh reorders triangles for the post-transform vertex cache, then
// for overdraw, and packs vertices in the order they are drawn.
// SimplifyMesh reduces the triangle count with quadric error.
// Both use the copied Go packages in internal/meshoptim and internal/meshsimplify.
// They do not link the meshoptimizer C++ library.
func (w *World) meshCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"optimizemesh": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			n := optimizeEntity(e)
			if n == 0 {
				return value.Value{}, fmt.Errorf("OptimizeMesh: no triangles")
			}
			e.shadowGeomDirty = true
			w.MarkShadowDirty()
			return value.Num(float64(n)), nil
		}),
		"simplifymesh": n(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			ratio := argN(a, 1, 0.5)
			nTris := simplifyEntity(e, ratio)
			if nTris == 0 {
				return value.Value{}, fmt.Errorf("SimplifyMesh: no triangles")
			}
			e.shadowGeomDirty = true
			w.MarkShadowDirty()
			return value.Num(float64(nTris)), nil
		}),
	}
}

func optimizeEntity(e *Entity) int {
	if e == nil || e.node == nil {
		return 0
	}
	total := 0
	var walk func(core.INode)
	walk = func(n core.INode) {
		if n == nil {
			return
		}
		if m, ok := n.(*graphic.Mesh); ok && m != nil {
			total += optimizeGeometry(m.GetGeometry())
		}
		for _, c := range n.GetNode().Children() {
			walk(c)
		}
	}
	walk(e.node)
	return total
}

func optimizeGeometry(g *geometry.Geometry) int {
	if g == nil {
		return 0
	}
	idx := g.Indices()
	if len(idx) < 3 {
		return 0
	}
	posV := g.VBO(gls.VertexPosition)
	if posV == nil || posV.Buffer() == nil {
		return 0
	}
	stride := posV.Stride()
	if stride <= 0 {
		stride = 3
	}
	vertCount := len(*posV.Buffer()) / stride
	if vertCount <= 0 {
		return 0
	}
	src := append([]uint32(nil), idx...)
	positions := packedPositions(*posV.Buffer(), stride, vertCount)
	if g.GroupCount() > 1 {
		opt := make([]uint32, 0, len(src))
		for i := 0; i < g.GroupCount(); i++ {
			grp := g.GroupAt(i)
			if grp == nil {
				continue
			}
			a := grp.Start
			b := a + grp.Count
			if a < 0 || b > len(src) || b < a {
				return len(src) / 3
			}
			part := optimizeTriangles(src[a:b], positions, vertCount)
			if len(part) != b-a {
				part = append([]uint32(nil), src[a:b]...)
			}
			opt = append(opt, part...)
		}
		g.SetIndices(math32.ArrayU32(opt))
		return len(opt) / 3
	}
	cached := optimizeTriangles(src, positions, vertCount)
	if len(cached) != (len(src)/3)*3 {
		cached = append([]uint32(nil), src[:(len(src)/3)*3]...)
	}
	packed, remap, used := meshoptim.OptimizeVertexFetch(cached, vertCount)
	if used > 0 && used <= vertCount && len(packed) == len(cached) {
		reorderVBOs(g, remap, vertCount, used)
		g.SetIndices(math32.ArrayU32(packed))
		return len(packed) / 3
	}
	g.SetIndices(math32.ArrayU32(cached))
	return len(cached) / 3
}

func optimizeTriangles(idx []uint32, positions []float32, vertCount int) []uint32 {
	cached := meshoptim.OptimizeVertexCache(idx, vertCount)
	tris := (len(idx) / 3) * 3
	if len(cached) != tris {
		cached = append([]uint32(nil), idx[:tris]...)
	}
	drawn := meshoptim.OptimizeOverdraw(cached, positions, vertCount, 1.05)
	if len(drawn) == len(cached) {
		return drawn
	}
	return cached
}

func packedPositions(buf []float32, stride, count int) []float32 {
	if stride == 3 && len(buf) >= count*3 {
		return buf
	}
	out := make([]float32, count*3)
	for i := 0; i < count; i++ {
		base := i * stride
		if base+3 > len(buf) {
			break
		}
		copy(out[i*3:(i+1)*3], buf[base:base+3])
	}
	return out
}

func reorderVBOs(g *geometry.Geometry, remap []int32, oldCount, newCount int) {
	for _, vbo := range g.VBOs() {
		if vbo == nil || vbo.Buffer() == nil {
			continue
		}
		stride := vbo.Stride()
		if stride <= 0 {
			continue
		}
		buf := *vbo.Buffer()
		if len(buf)/stride < oldCount {
			continue
		}
		next := meshoptim.ApplyRemapFloat32(buf, stride, remap, newCount)
		vbo.SetBuffer(math32.ArrayF32(next))
	}
}

func simplifyEntity(e *Entity, ratio float64) int {
	if e == nil || e.node == nil {
		return 0
	}
	total := 0
	var walk func(core.INode)
	walk = func(n core.INode) {
		if n == nil {
			return
		}
		if m, ok := n.(*graphic.Mesh); ok && m != nil {
			total += simplifyGeometry(m.GetGeometry(), ratio)
		}
		for _, c := range n.GetNode().Children() {
			walk(c)
		}
	}
	walk(e.node)
	return total
}

func simplifyGeometry(g *geometry.Geometry, ratio float64) int {
	if g == nil {
		return 0
	}
	idx := g.Indices()
	if len(idx) < 3 {
		return 0
	}
	if ratio <= 0 || ratio > 1 {
		ratio = 0.5
	}
	if ratio > 0.99 {
		return len(idx) / 3
	}
	if g.GroupCount() > 1 {
		return simplifyGroups(g, ratio)
	}
	n := applySimplify(g, idx, ratio)
	if n > 0 && g.GroupCount() == 1 {
		grp := g.GroupAt(0)
		grp.Start = 0
		grp.Count = n * 3
	}
	return n
}

func simplifyGroups(g *geometry.Geometry, ratio float64) int {
	src := append([]uint32(nil), g.Indices()...)
	var idxOut []uint32
	type piece struct {
		start, count int
		pos          []float32
		attrs        map[*gls.VBO][]float32
	}
	var pieces []piece
	for i := 0; i < g.GroupCount(); i++ {
		grp := g.GroupAt(i)
		if grp == nil {
			continue
		}
		a, b := grp.Start, grp.Start+grp.Count
		if a < 0 || b > len(src) || b-a < 3 {
			return len(src) / 3
		}
		res := meshsimplify.Simplify(meshsimplify.Mesh{
			Positions: positionSlice(g),
			Indices:   append([]uint32(nil), src[a:b]...),
		}, meshsimplify.Options{TargetRatio: ratio})
		if res.OutputTriangles < 1 || len(res.Sources) == 0 {
			return len(src) / 3
		}
		p := piece{start: len(idxOut), count: len(res.Mesh.Indices)}
		base := uint32(0)
		for _, prev := range pieces {
			base += uint32(len(prev.pos) / 3)
		}
		for _, v := range res.Mesh.Indices {
			idxOut = append(idxOut, v+base)
		}
		p.pos = append([]float32(nil), res.Mesh.Positions...)
		p.attrs = blendAttrs(g, res.Sources)
		pieces = append(pieces, p)
	}
	if len(idxOut) < 3 {
		return 0
	}
	var pos []float32
	merged := map[*gls.VBO][]float32{}
	for _, p := range pieces {
		pos = append(pos, p.pos...)
		for vbo, buf := range p.attrs {
			merged[vbo] = append(merged[vbo], buf...)
		}
	}
	posV := g.VBO(gls.VertexPosition)
	if posV != nil && posV.Stride() == 3 {
		posV.SetBuffer(math32.ArrayF32(pos))
	}
	for vbo, buf := range merged {
		if vbo == posV && vbo.Stride() == 3 {
			continue
		}
		vbo.SetBuffer(math32.ArrayF32(buf))
	}
	g.SetIndices(math32.ArrayU32(idxOut))
	cursor := 0
	for i, p := range pieces {
		grp := g.GroupAt(i)
		if grp == nil {
			continue
		}
		grp.Start = cursor
		grp.Count = p.count
		cursor += p.count
	}
	return len(idxOut) / 3
}

func applySimplify(g *geometry.Geometry, idx []uint32, ratio float64) int {
	pos := positionSlice(g)
	if len(pos) < 9 {
		return len(idx) / 3
	}
	res := meshsimplify.Simplify(meshsimplify.Mesh{
		Positions: pos,
		Indices:   append([]uint32(nil), idx...),
	}, meshsimplify.Options{TargetRatio: ratio})
	if res.OutputTriangles < 1 || len(res.Mesh.Indices) < 3 || len(res.Sources) == 0 {
		return len(idx) / 3
	}
	posV := g.VBO(gls.VertexPosition)
	if posV != nil && posV.Stride() == 3 {
		posV.SetBuffer(math32.ArrayF32(append([]float32(nil), res.Mesh.Positions...)))
	}
	for vbo, buf := range blendAttrs(g, res.Sources) {
		if vbo == posV && posV.Stride() == 3 {
			continue
		}
		vbo.SetBuffer(math32.ArrayF32(buf))
	}
	g.SetIndices(math32.ArrayU32(append([]uint32(nil), res.Mesh.Indices...)))
	return len(res.Mesh.Indices) / 3
}

func positionSlice(g *geometry.Geometry) []float32 {
	posV := g.VBO(gls.VertexPosition)
	if posV == nil || posV.Buffer() == nil {
		return nil
	}
	stride := posV.Stride()
	if stride <= 0 {
		stride = 3
	}
	buf := *posV.Buffer()
	count := len(buf) / stride
	return packedPositions(buf, stride, count)
}

func blendAttrs(g *geometry.Geometry, sources []meshsimplify.VertexSource) map[*gls.VBO][]float32 {
	out := map[*gls.VBO][]float32{}
	for _, vbo := range g.VBOs() {
		if vbo == nil || vbo.Buffer() == nil {
			continue
		}
		stride := vbo.Stride()
		if stride <= 0 {
			continue
		}
		buf := *vbo.Buffer()
		next := make([]float32, len(sources)*stride)
		for i, src := range sources {
			a := int(src.A)
			b := int(src.B)
			t := float32(src.T)
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			for c := 0; c < stride; c++ {
				va := float32(0)
				vb := float32(0)
				if a >= 0 && (a+1)*stride <= len(buf) {
					va = buf[a*stride+c]
				}
				if b >= 0 && (b+1)*stride <= len(buf) {
					vb = buf[b*stride+c]
				} else {
					vb = va
				}
				next[i*stride+c] = va*(1-t) + vb*t
			}
		}
		out[vbo] = next
	}
	return out
}
