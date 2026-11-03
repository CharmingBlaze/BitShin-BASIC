package runtime

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// Generated trees: a game gets a forest with no art files. CreateTree builds a
// template for TerrainTrees, one seed giving one shape.
//
// Two things do most of the work, both from production foliage practice. Normals
// are bent outward from the crown, so a lumpy mesh lights as one soft volume and
// not as a pile of facets (the usual canopy trick; ez-tree and Crysis's foliage
// do the same). Shade is baked into the vertices: undersides and the inside of
// the crown are dark, tips are light. That reads as depth without any cost when
// drawing, and it survives instancing because it belongs to the template.
//
// A broadleaf crown is a dark core inside leaf cards: crossed quads the shader
// cuts into clumps of leaves from noise, so a game needs no leaf texture. Cards
// give a leafy outline for fewer triangles than solid lumps. Each kind also gets a
// hand-built far crown, since auto-simplifying leaves nothing recognisable.

// treeFarOnly names a template part that replaces the one before it on far trees.
const treeFarOnly = "tree-far-only"

// treeCards names a template part of leaf cards. Its vertex color holds the card's
// u, v and shade.
const treeCards = "tree-cards"

// treeMesh collects one part's vertices.
type treeMesh struct {
	pos, nor, col []float32
	idx           []uint32
}

func (m *treeMesh) vert(p, n math32.Vector3, shade float32) uint32 {
	i := uint32(len(m.pos) / 3)
	n.Normalize()
	m.pos = append(m.pos, p.X, p.Y, p.Z)
	m.nor = append(m.nor, n.X, n.Y, n.Z)
	m.col = append(m.col, shade, shade, shade)
	return i
}

func (m *treeMesh) tri(a, b, c uint32) { m.idx = append(m.idx, a, b, c) }

func (m *treeMesh) geometry() *geometry.Geometry {
	n := len(m.pos) / 3
	pos, nor, col := math32.NewArrayF32(0, len(m.pos)), math32.NewArrayF32(0, len(m.nor)), math32.NewArrayF32(0, len(m.col))
	pos.Append(m.pos...)
	nor.Append(m.nor...)
	col.Append(m.col...)
	uvs := math32.NewArrayF32(n*2, n*2)
	idx := math32.NewArrayU32(0, len(m.idx))
	idx.Append(m.idx...)
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	g.AddVBO(gls.NewVBO(col).AddAttrib(gls.VertexColor))
	return g
}

// treeTrunk is a tapered, slightly leaning trunk.
func treeTrunk(r *rand.Rand, height, radius float32) *treeMesh {
	m := &treeMesh{}
	const sides, rows = 6, 3
	leanX, leanZ := (r.Float32()-0.5)*height*0.04, (r.Float32()-0.5)*height*0.04
	for j := 0; j <= rows; j++ {
		t := float32(j) / rows
		rad := radius * (1 - 0.72*t)
		if j == 0 {
			rad = radius * 1.35 // the flare where roots begin
		}
		for i := 0; i < sides; i++ {
			a := float32(i) / sides * 2 * math32.Pi
			c, s := math32.Cos(a), math32.Sin(a)
			p := math32.Vector3{X: c*rad + leanX*t*t, Y: height * t, Z: s*rad + leanZ*t*t}
			m.vert(p, math32.Vector3{X: c, Y: 0.15, Z: s}, 0.55+0.5*t)
		}
	}
	for j := 0; j < rows; j++ {
		for i := 0; i < sides; i++ {
			a, b := uint32(j*sides+i), uint32(j*sides+(i+1)%sides)
			c, d := a+sides, b+sides
			m.tri(a, c, b)
			m.tri(b, c, d)
		}
	}
	return m
}

// pineCrown stacks drooping star-edged skirts, each dark underneath, so the
// outline breaks up into branch tips where a cone would be one smooth line.
func pineCrown(r *rand.Rand, height float32, tiers, tips int, under bool) *treeMesh {
	m := &treeMesh{}
	base := height * 0.2
	span := height - base
	maxR := height * 0.21
	for k := 0; k < tiers; k++ {
		t := float32(k) / float32(tiers)
		top := base + span*(t+1.55/float32(tiers))
		if top > height {
			top = height
		}
		low := base + span*t
		rad := maxR * math32.Pow(1-t*0.93, 0.85) * (0.92 + r.Float32()*0.16)
		lit := 0.78 + 0.32*t
		apex := m.vert(math32.Vector3{Y: top}, math32.Vector3{Y: 1}, lit+0.12)
		spin := r.Float32() * 2 * math32.Pi
		rim := make([]uint32, 0, tips*2)
		below := make([]uint32, 0, tips*2)
		for i := 0; i < tips*2; i++ {
			a := spin + float32(i)/float32(tips*2)*2*math32.Pi
			c, s := math32.Cos(a), math32.Sin(a)
			rr, drop, tip := rad*(0.62+r.Float32()*0.08), float32(0), lit*0.74
			if i%2 == 0 { // a branch tip: further out, drooping, catching light
				rr = rad * (0.9 + r.Float32()*0.22)
				drop = rad * 0.18
				tip = lit * 0.98
			}
			p := math32.Vector3{X: c * rr, Y: low - drop, Z: s * rr}
			n := math32.Vector3{X: c * 0.75, Y: 0.8, Z: s * 0.75}
			rim = append(rim, m.vert(p, n, tip))
			if under {
				below = append(below, m.vert(p, math32.Vector3{X: c * 0.4, Y: -0.2, Z: s * 0.4}, lit*0.42))
			}
		}
		centre := m.vert(math32.Vector3{Y: low + span*0.04}, math32.Vector3{Y: -1}, lit*0.3)
		for i := range rim {
			j := (i + 1) % len(rim)
			m.tri(apex, rim[j], rim[i])
			if under {
				m.tri(centre, below[i], below[j])
			}
		}
	}
	return m
}

// lump appends one low-poly ball to a crown, its normals bent toward the crown's
// centre line and its shade set by how far in and how low it sits.
func (m *treeMesh) lump(r *rand.Rand, c math32.Vector3, rad float32, crown math32.Vector3, crownR float32, plain bool) {
	// An octahedron split once: 18 corners, 32 faces, and a lumpy edge. Plain
	// stays unsplit, for far crowns.
	dirs := []math32.Vector3{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}, {Z: 1}, {Z: -1}}
	faces := [][3]int{{0, 2, 4}, {2, 1, 4}, {1, 3, 4}, {3, 0, 4}, {2, 0, 5}, {1, 2, 5}, {3, 1, 5}, {0, 3, 5}}
	mids := map[[2]int]int{}
	mid := func(a, b int) int {
		if a > b {
			a, b = b, a
		}
		if i, ok := mids[[2]int{a, b}]; ok {
			return i
		}
		d := dirs[a]
		d.Add(&dirs[b])
		d.Normalize()
		dirs = append(dirs, d)
		mids[[2]int{a, b}] = len(dirs) - 1
		return len(dirs) - 1
	}
	var all [][3]int
	if plain {
		all = faces
		faces = nil
	}
	for _, f := range faces {
		ab, bc, ca := mid(f[0], f[1]), mid(f[1], f[2]), mid(f[2], f[0])
		all = append(all, [3]int{f[0], ab, ca}, [3]int{ab, f[1], bc}, [3]int{ca, bc, f[2]}, [3]int{ab, bc, ca})
	}
	base := uint32(len(m.pos) / 3)
	for _, d := range dirs {
		p := d
		p.MultiplyScalar(rad * (0.68 + r.Float32()*0.56))
		p.Add(&c)
		out := p
		out.Sub(&crown)
		dist := out.Length() / math32.Max(crownR, 0.01)
		out.Normalize()
		n := d
		n.MultiplyScalar(0.72)
		out.MultiplyScalar(0.28)
		n.Add(&out)
		// Dark toward the crown's middle and its underside.
		height := (p.Y - crown.Y) / math32.Max(crownR, 0.01)
		m.vert(p, n, crownShade(dist, height)+(r.Float32()-0.5)*0.2)
	}
	for _, f := range all {
		m.tri(base+uint32(f[0]), base+uint32(f[1]), base+uint32(f[2]))
	}
}

// crownShade darkens toward a crown's middle and underside, and never so far
// that the mass goes black once shadow falls on it too.
func crownShade(dist, height float32) float32 {
	s := 0.5 + 0.36*math32.Min(dist, 1) + 0.24*height
	return math32.Max(0.36, math32.Min(s, 1.2))
}

// oakCrown is lumps spread over an ellipsoid shell, with a few inside to keep it
// from looking hollow through the gaps.
func oakCrown(r *rand.Rand, height float32, n int, reach float32, plain bool) *treeMesh {
	m := &treeMesh{}
	centre := math32.Vector3{Y: height * 0.64}
	rx, ry := height*0.36, height*0.3
	for i := 0; i < n; i++ {
		// Fibonacci spiral: even cover with no clumping, nudged to look grown.
		k := (float32(i) + 0.5) / float32(n)
		y := 1 - 2*k
		ring := math32.Sqrt(math32.Max(1-y*y, 0))
		a := float32(i) * 2.399963
		shell := (0.72 + r.Float32()*0.28) * reach
		p := math32.Vector3{
			X: centre.X + math32.Cos(a)*ring*rx*shell,
			Y: centre.Y + y*ry*shell,
			Z: centre.Z + math32.Sin(a)*ring*rx*shell,
		}
		rad := height * (0.1 + r.Float32()*0.05)
		if plain {
			rad *= 1.7
		}
		m.lump(r, p, rad, centre, rx, plain)
	}
	if !plain {
		m.lump(r, centre, rx*0.72, centre, rx*2, false) // fills the middle, stays dark
	}
	return m
}

// oakCards spreads leaf cards over the crown's shell, facing mostly outward. Their
// normals come from the crown, so they light as one soft mass, and each card's
// single shade is also the seed that gives it its own pattern of leaves.
func oakCards(r *rand.Rand, height float32, n int) *treeMesh {
	m := &treeMesh{}
	centre := math32.Vector3{Y: height * 0.64}
	rx, ry := height*0.36, height*0.3
	for i := 0; i < n; i++ {
		k := (float32(i) + 0.5) / float32(n)
		y := 1 - 2*k
		ring := math32.Sqrt(math32.Max(1-y*y, 0))
		a := float32(i) * 2.399963
		shell := 0.6 + r.Float32()*0.45
		d := math32.Vector3{X: math32.Cos(a) * ring, Y: y, Z: math32.Sin(a) * ring}
		p := math32.Vector3{X: centre.X + d.X*rx*shell, Y: centre.Y + d.Y*ry*shell, Z: centre.Z + d.Z*rx*shell}
		out := p
		out.Sub(&centre)
		dist := out.Length() / rx
		height01 := (p.Y - centre.Y) / rx
		out.Normalize()
		face := math32.Vector3{X: d.X + (r.Float32()-0.5)*1.1, Y: d.Y + (r.Float32()-0.5)*1.1, Z: d.Z + (r.Float32()-0.5)*1.1}
		face.Normalize()
		up := math32.Vector3{Y: 1}
		if math32.Abs(face.Y) > 0.9 {
			up = math32.Vector3{X: 1}
		}
		var t1, t2 math32.Vector3
		t1.CrossVectors(&face, &up)
		t1.Normalize()
		t2.CrossVectors(&face, &t1)
		s := height * (0.14 + r.Float32()*0.07)
		shade := crownShade(dist, height01) + (r.Float32()-0.5)*0.22
		base := uint32(len(m.pos) / 3)
		for _, c := range [][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
			q := p
			x, y := (c[0]*2-1)*s, (c[1]*2-1)*s
			q.X += t1.X*x + t2.X*y
			q.Y += t1.Y*x + t2.Y*y
			q.Z += t1.Z*x + t2.Z*y
			m.vert(q, out, shade)
			j := len(m.col) - 3
			m.col[j], m.col[j+1] = c[0], c[1]
		}
		m.tri(base, base+1, base+2)
		m.tri(base, base+2, base+3)
	}
	return m
}

// createTree builds a hidden template: the trunk with the crown as its child.
func (w *World) createTree(kind string, seed int, height float32) (int, error) {
	if height <= 0 {
		height = 20
	}
	r := rand.New(rand.NewSource(int64(seed)*7919 + 17))
	var crown, far, cards *treeMesh
	var trunkH, trunkR float32
	// These are sRGB, so they read darker once decoded for lighting.
	bark := math32.Color{R: 0.42, G: 0.33, B: 0.26}
	leaf := math32.Color{R: 0.2, G: 0.34, B: 0.17}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "pine", "conifer", "fir":
		crown = pineCrown(r, height, 7+r.Intn(3), 11, true)
		far = pineCrown(r, height, 4, 6, false)
		trunkH, trunkR = height*0.55, height*0.028
		// Each tree of a kind differs a little in hue.
		leaf.G += (r.Float32() - 0.5) * 0.03
	case "oak", "broadleaf", "tree":
		crown = oakCrown(r, height, 7, 0.7, false)
		far = oakCrown(r, height, 6, 0.7, true)
		cards = oakCards(r, height, 56+r.Intn(16))
		trunkH, trunkR = height*0.56, height*0.038
		bark = math32.Color{R: 0.45, G: 0.37, B: 0.29}
		leaf = math32.Color{R: 0.29, G: 0.41, B: 0.15}
		leaf.R += (r.Float32() - 0.5) * 0.05
	default:
		return 0, fmt.Errorf("CreateTree: want pine or oak, got %q", kind)
	}
	root := w.meshEnt(treeTrunk(r, trunkH, trunkR).geometry(), 0)
	child := w.meshEnt(crown.geometry(), root)
	farID := w.meshEnt(far.geometry(), root)
	if e := w.ents[farID]; e != nil {
		e.name = treeFarOnly
	}
	tint := map[int]math32.Color{root: bark, child: leaf, farID: leaf}
	if cards != nil {
		id := w.meshEnt(cards.geometry(), root)
		tint[id] = leaf
		if e := w.ents[id]; e != nil {
			e.name = treeCards
		}
	}
	for id, c := range tint {
		if e := w.ents[id]; e != nil && e.mat != nil {
			col := c
			e.mat.SetColor(&col)
		}
	}
	if e := w.ents[root]; e != nil {
		e.node.GetNode().SetVisible(false)
		e.name = "tree"
	}
	return root, nil
}

func (w *World) treeGenCommands(need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// CreateTree(kind$ [, seed, height]): pine or oak, ready for TerrainTrees.
		// The template is hidden; each seed gives a different shape.
		"createtree": need(func(a []value.Value) (value.Value, error) {
			id, err := w.createTree(argS(a, 0), argI(a, 1, 1), float32(argN(a, 2, 20)))
			return value.Num(float64(id)), err
		}),
	}
}
