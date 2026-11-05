package runtime

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"unsafe"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer/shaders"
	"github.com/g3n/engine/texture"
	"github.com/go-gl/gl/v3.3-core/gl"
)

// Automatic batching. G3N draws every mesh on its own, with its own shader
// lookup and uniforms, which cost about 11 us per object on a 780M. Entities
// that share a geometry and have equal material settings are drawn here as one
// instanced draw instead, in the scene pass and in every shadow tile.
//
// A proxy mesh in the scene stands in for each group. G3N sets the program,
// lights, material and geometry for it as for any mesh, then calls its
// RenderSetup, which draws the copies. G3N's own draw that follows has a count
// of one index, so it draws nothing.
//
// Membership is worked out once a frame in updateBatches and does not depend on
// visibility. Each pass checks visibility and culls, because a water pass hides
// entities for just that pass.

const (
	batchMinCopies  = 2
	batchAttrib     = 12 // four vec4 slots, above G3N's fixed 0..3
	batchFloats     = 16
	batchErrChecks  = 240 // draws checked with glGetError before trusting them
	batchInstDefine = "MB_INSTANCED"
)

// batchKey is everything that makes one copy draw differently from another.
// Equal keys may share a draw call.
type batchKey struct {
	geom      *geometry.Geometry
	shader    string
	useLights material.UseLights
	side      material.Side
	blending  material.Blending
	wireframe bool
	depthMask bool
	depthTest bool
	depthFunc uint32
	lineWidth float32
	polyF     float32
	polyU     float32
	defines   int
	tex       [4]*texture.Texture2D
	texN      int
	cutTex    *texture.Texture2D // the entity's albedo, read by the shadow pass
	uni       [16]float32        // the material's uniform block
	pbr       bool
	recv      bool
	// litMat
	vertexColor             bool
	cutout                  int
	specMap, emitMap, nmMap *texture.Texture2D
	// pbrMat
	coat, coatRough, sub, heightScale float32
	subColor                          math32.Color
	heightTex                         *texture.Texture2D
	ao, toonBands, toonRim, toonSpec  float32
	ibl, shading                      int
	envTex                            *texture.Texture2D
	noFog                             bool
}

// batchEntity is one entity's standing with the batcher.
type batchEntity struct {
	key   batchKey
	group *batchGroup
	drawn bool // the group draws it; its own mesh is not renderable
	// For a mesh with no copies: how long it has been still, and its merged mesh.
	mat    math32.Matrix4
	still  int
	merged *mergeGroup
}

type batchMember struct {
	id int
	e  *Entity
}

type batchGroup struct {
	key     batchKey
	members []batchMember
	proxy   *batchProxy
	// Filled once a frame, one entry per member.
	nodes   []*core.Node
	mats    []float32 // batchFloats per member
	spheres []float32 // world x, y, z, radius per member
	shown   []bool    // the node and everything above it are visible
	lod     *geomLOD  // simpler versions, once built
	caster  []bool    // casts a shadow now
	static  []bool
}

type batchState struct {
	off     bool
	envRead bool
	groups  map[batchKey]*batchGroup
	active  []*batchGroup
	vbo     uint32
	scratch []float32
	checks  int
	// Every entity by id. Walking the map visits memory at random, which at
	// 50,000 entities cost more than everything else here together.
	root     *sceneRoot
	merges   map[mergeKey]*mergeGroup
	lods     map[*geometry.Geometry]*geomLOD
	lodDone  chan *lodJob
	lodSlots chan struct{}
	parts    []*partLOD
	// Triangles drawn through batches this frame: the view (with its reflections) and
	// the shadow tiles. Unlike StatsTriangles, this is after culling and detail.
	drawnView, drawnShadow int
	inDepth                bool
	lodBins                [len(batchLODRatios) + 1][]float32
	order                  []batchMember
	orderAt                int
	orderN                 int
	// Above 0, a water pass has hidden entities just for itself, so shown is
	// out of date and each node must be asked.
	passHides int
	// Depth programs: index by moment output, then by whether every attribute
	// is read (a geometry's first draw, see depthAttribsGLSL).
	depth      [2][2]*gls.Program
	depthTried bool
}

// batchProxy is the scene's stand-in for a group.
type batchProxy struct {
	graphic.Mesh
	w       *World
	g       *batchGroup
	uniView gls.Uniform
	uniProj gls.Uniform
}

func newBatchProxy(w *World, g *batchGroup) *batchProxy {
	p := &batchProxy{w: w, g: g}
	p.Mesh.Init(g.key.geom, nil)
	p.SetCullable(false)
	p.ShaderDefines.Set(batchInstDefine, "1")
	p.uniView.Init("BatchView")
	p.uniProj.Init("BatchProj")
	return p
}

// setMaterial points the proxy at one member's material. All of them are equal.
func (p *batchProxy) setMaterial(imat material.IMaterial) {
	mats := p.Materials()
	if len(mats) == 1 && mats[0].IMaterial() == imat {
		return
	}
	p.ClearMaterials()
	p.Graphic.AddMaterial(p, imat, 0, 1)
}

// RenderSetup runs with this pass's program, lights and material current and
// the geometry's vertex array bound.
func (p *batchProxy) RenderSetup(gs *gls.GLS, rinfo *core.RenderInfo) {
	b := &p.w.batch
	var vp math32.Matrix4
	vp.MultiplyMatrices(&rinfo.ProjMatrix, &rinfo.ViewMatrix)
	planes := frustumPlanes(&vp)
	g := p.g
	live := b.passHides > 0
	keep := func(i int) bool {
		if live {
			if !nodeShown(g.nodes[i]) {
				return false
			}
		} else if !g.shown[i] {
			return false
		}
		return sphereInPlanes(&planes, g.spheres[i*4:i*4+4])
	}
	gs.UniformMatrix4fv(p.uniView.Location(gs), 1, false, &rinfo.ViewMatrix[0])
	gs.UniformMatrix4fv(p.uniProj.Location(gs), 1, false, &rinfo.ProjMatrix[0])
	if g.lod != nil && p.binViewLODs(&rinfo.ViewMatrix, &rinfo.ProjMatrix, keep) {
		p.w.drawLODBins(g, gs)
		return
	}
	if n := g.cullToScratch(b, keep); n > 0 {
		p.w.drawBatchInstances(g.key.geom, n)
	}
}

// cullToScratch copies the matrices of the members keep accepts and counts them.
func (g *batchGroup) cullToScratch(b *batchState, keep func(i int) bool) int {
	b.scratch = b.scratch[:0]
	for i := range g.nodes {
		if keep(i) {
			b.scratch = append(b.scratch, g.mats[i*batchFloats:(i+1)*batchFloats]...)
		}
	}
	return len(b.scratch) / batchFloats
}

// drawBatchInstances draws n copies from the scratch matrices. The geometry's
// vertex array must be bound. The instance attributes are switched off again
// because that array also serves plain draws of the same geometry.
func (w *World) drawBatchInstances(geom *geometry.Geometry, n int) {
	b := &w.batch
	if b.vbo == 0 {
		gl.GenBuffers(1, &b.vbo)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, b.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(b.scratch)*4, gl.Ptr(b.scratch), gl.STREAM_DRAW)
	for c := uint32(0); c < 4; c++ {
		gl.EnableVertexAttribArray(batchAttrib + c)
		gl.VertexAttribPointerWithOffset(batchAttrib+c, 4, gl.FLOAT, false, batchFloats*4, uintptr(c*16))
		gl.VertexAttribDivisor(batchAttrib+c, 1)
	}
	if b.inDepth {
		b.drawnShadow += geomTriangles(geom) * n
	} else {
		b.drawnView += geomTriangles(geom) * n
	}
	if idx := geom.Indices(); len(idx) > 0 {
		gl.DrawElementsInstanced(gl.TRIANGLES, int32(len(idx)), gl.UNSIGNED_INT, nil, int32(n))
	} else {
		gl.DrawArraysInstanced(gl.TRIANGLES, 0, int32(geom.Items()), int32(n))
	}
	for c := uint32(0); c < 4; c++ {
		gl.DisableVertexAttribArray(batchAttrib + c)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	if b.checks < batchErrChecks {
		b.checks++
		if err := gl.GetError(); err != gl.NO_ERROR {
			// Hand every mesh back to G3N; the next frame draws them singly.
			fmt.Printf("batching: draw failed (%s); drawing meshes singly\n", glErrName(err))
			b.off = true
		}
	}
}

// entityOrder returns the entities in id order, rebuilt when any were added or freed.
func (w *World) entityOrder() []batchMember {
	b := &w.batch
	if b.order != nil && b.orderAt == w.entsAdded && b.orderN == len(w.ents) {
		return b.order
	}
	b.orderAt, b.orderN = w.entsAdded, len(w.ents)
	// A new slice each time: a caller may still be walking the old one when a
	// script callback adds an entity.
	b.order = make([]batchMember, 0, len(w.ents))
	for id, e := range w.ents {
		if e != nil {
			b.order = append(b.order, batchMember{id, e})
		}
	}
	sort.Slice(b.order, func(i, j int) bool { return b.order[i].id < b.order[j].id })
	return b.order
}

// updateBatches sorts the entities into groups for this frame.
func (w *World) updateBatches() {
	b := &w.batch
	if !b.envRead {
		b.envRead = true
		if v := os.Getenv("MOON_BATCH"); v == "0" || strings.EqualFold(v, "off") {
			b.off = true
		}
	}
	if b.off || !batchLayout.ok {
		w.releaseBatches()
		return
	}
	w.collectLODs()
	w.scene.UpdateMatrixWorld()
	b.drawnView, b.drawnShadow = 0, 0
	w.tickPartLODs()
	if b.groups == nil {
		b.groups = map[batchKey]*batchGroup{}
	}
	for _, g := range b.groups {
		g.members = g.members[:0]
	}
	var key batchKey
	for _, m := range w.entityOrder() {
		e := m.e
		if e.mesh == nil {
			continue
		}
		key = batchKey{}
		ok := fillBatchKey(e, &key)
		st := e.batch
		if !ok {
			if st != nil {
				releaseBatchEntity(e)
			}
			continue
		}
		if st == nil {
			st = &batchEntity{}
			e.batch = st
		}
		if st.group == nil || st.key != key {
			g := b.groups[key]
			if g == nil {
				g = &batchGroup{key: key}
				b.groups[key] = g
			}
			st.key, st.group = key, g
		}
		st.group.members = append(st.group.members, m)
	}
	b.active = b.active[:0]
	for key, g := range b.groups {
		if len(g.members) < batchMinCopies && !wantsOwnLOD(g) {
			g.detach(w)
			for _, m := range g.members {
				if st := m.e.batch; st.drawn && st.merged == nil {
					st.drawn = false
					m.e.mesh.SetRenderable(true)
				}
				w.tickMergeMember(m)
			}
			if len(g.members) == 0 {
				delete(b.groups, key)
			}
			continue
		}
		b.active = append(b.active, g)
	}
	w.applyMerges()
	for _, g := range b.active {
		w.fillBatchGroup(g)
	}
}

func (g *batchGroup) detach(w *World) {
	if g.proxy != nil {
		w.scene.Remove(g.proxy)
		g.proxy = nil
	}
}

func (w *World) fillBatchGroup(g *batchGroup) {
	if g.proxy == nil {
		g.proxy = newBatchProxy(w, g)
	}
	if g.proxy.Parent() == nil {
		w.scene.Add(g.proxy)
	}
	// The lowest id, so the material in use does not change from frame to frame.
	g.proxy.setMaterial(g.members[0].e.mesh.Materials()[0].IMaterial())

	g.lod = w.lodFor(g.key.geom)
	if g.lod != nil {
		// Vertices that change would leave the simpler versions behind.
		for _, m := range g.members {
			if m.e.skin != nil || m.e.shadowGeomDirty || w.meshBuilds[m.id] != nil {
				g.lod = nil
				break
			}
		}
	}
	sph := g.key.geom.BoundingSphere()
	n := len(g.members)
	g.nodes = g.nodes[:0]
	g.shown = g.shown[:0]
	g.caster = g.caster[:0]
	g.static = g.static[:0]
	if cap(g.mats) < n*batchFloats {
		g.mats = make([]float32, n*batchFloats)
		g.spheres = make([]float32, n*4)
	}
	g.mats = g.mats[:n*batchFloats]
	g.spheres = g.spheres[:n*4]
	for i, m := range g.members {
		mergeOutEntity(m.e) // it has copies now: instancing draws it
		m.e.batch.drawn = true
		if m.e.mesh.Renderable() {
			m.e.mesh.SetRenderable(false)
		}
		node := m.e.mesh.GetNode()
		mw := node.MatrixWorld()
		copy(g.mats[i*batchFloats:], mw[:])
		c := sph.Center
		c.ApplyMatrix4(&mw)
		sx := mw[0]*mw[0] + mw[1]*mw[1] + mw[2]*mw[2]
		sy := mw[4]*mw[4] + mw[5]*mw[5] + mw[6]*mw[6]
		sz := mw[8]*mw[8] + mw[9]*mw[9] + mw[10]*mw[10]
		s := g.spheres[i*4 : i*4+4]
		s[0], s[1], s[2] = c.X, c.Y, c.Z
		s[3] = sph.Radius * math32.Sqrt(math32.Max(sx, math32.Max(sy, sz)))
		g.nodes = append(g.nodes, node)
		g.shown = append(g.shown, nodeShown(node))
		g.caster = append(g.caster, node.Visible() && !shadowSkipMesh(m.e))
		g.static = append(g.static, shadowCasterStatic(m.e))
	}
}

// releaseBatches gives every mesh back to G3N.
func (w *World) releaseBatches() {
	b := &w.batch
	if len(b.groups) == 0 {
		return
	}
	for _, g := range b.groups {
		g.detach(w)
	}
	for _, g := range b.merges {
		g.dropMesh(w)
	}
	b.merges = nil
	b.groups = nil
	b.active = b.active[:0]
	for _, e := range w.ents {
		if e != nil && e.batch != nil {
			releaseBatchEntity(e)
		}
	}
}

func releaseBatchEntity(e *Entity) {
	if e.batch.drawn {
		e.mesh.SetRenderable(true)
	}
	e.batch.drawn = false
	e.batch.group = nil
	e.batch.merged = nil
	e.batch.still = 0
}

// batchDrawn reports whether a group draws this entity in the scene pass.
func batchDrawn(e *Entity) bool { return e.batch != nil && e.batch.drawn }

// batchKeyOf returns the entity's key, or false when it must be drawn alone.
func batchKeyOf(e *Entity) (batchKey, bool) {
	var k batchKey
	ok := fillBatchKey(e, &k)
	return k, ok
}

// fillBatchKey writes the key into a zeroed k. It runs for every mesh every
// frame, and the key is too big to pass around by value.
func fillBatchKey(e *Entity, k *batchKey) bool {
	m := e.mesh
	if n, ok := e.node.(*graphic.Mesh); !ok || n != m {
		return false // a rigged mesh, or a mesh held by a wrapper
	}
	if e.sky || (!m.Renderable() && !batchDrawn(e)) || m.RenderOrder() != 0 || len(m.ShaderDefines) != 0 {
		return false
	}
	mats := m.Materials()
	// One material over the whole mesh. A box has groups but draws them all.
	if len(mats) != 1 || *(*int)(unsafe.Add(unsafe.Pointer(&mats[0]), batchLayout.matCount)) != 0 {
		return false
	}
	if _, morph := m.IGeometry().(*geometry.MorphGeometry); morph {
		return false
	}
	geom := m.GetGeometry()
	if geom == nil || len(geom.ShaderDefines) != 0 {
		return false
	}
	if len(geom.Indices()) == 0 && geom.Items() == 0 {
		return false
	}
	k.geom = geom
	var base *material.Material
	switch im := mats[0].IMaterial().(type) {
	case *litMat:
		if im.wind {
			return false
		}
		base = im.GetMaterial()
		k.uni = *(*[16]float32)(unsafe.Add(unsafe.Pointer(im.Standard), batchLayout.stdUni))
		k.recv, k.vertexColor, k.cutout = im.recvShadow, im.vertexColor, im.cutout
		k.specMap, k.emitMap, k.nmMap = im.specMap, im.emitMap, im.normMap
		if len(base.ShaderDefines) != 0 {
			return false
		}
	case *pbrMat:
		base = im.GetMaterial()
		copy(k.uni[:], (*[10]float32)(unsafe.Add(unsafe.Pointer(im.Physical), batchLayout.physUni))[:])
		k.pbr = true
		k.recv, k.noFog, k.envTex = im.recvShadow, im.noFog, im.envTex
		k.ao, k.ibl, k.shading = im.ao, im.ibl, im.shading
		k.toonBands, k.toonRim, k.toonSpec = im.toonBands, im.toonRim, im.toonSpec
		k.coat, k.coatRough, k.sub, k.subColor = im.clearcoat, im.coatRough, im.subsurface, im.subColor
		k.heightTex, k.heightScale = im.heightTex, im.heightScale
	default:
		return false
	}
	k.shader = base.Shader()
	switch k.shader {
	case "standard", "bsshadow", "mbphysical":
	default:
		return false
	}
	if base.Transparent() || base.ShaderUnique() {
		return false
	}
	p := unsafe.Pointer(base)
	k.useLights, k.side, k.wireframe = base.UseLights(), base.Side(), base.Wireframe()
	k.blending = *(*material.Blending)(unsafe.Add(p, batchLayout.blending))
	k.depthMask = *(*bool)(unsafe.Add(p, batchLayout.depthMask))
	k.depthTest = *(*bool)(unsafe.Add(p, batchLayout.depthTest))
	k.depthFunc = *(*uint32)(unsafe.Add(p, batchLayout.depthFunc))
	k.lineWidth = *(*float32)(unsafe.Add(p, batchLayout.lineWidth))
	k.polyF = *(*float32)(unsafe.Add(p, batchLayout.polyF))
	k.polyU = *(*float32)(unsafe.Add(p, batchLayout.polyU))
	texs := *(*[]*texture.Texture2D)(unsafe.Add(p, batchLayout.textures))
	if len(texs) > len(k.tex) {
		return false
	}
	k.texN = copy(k.tex[:], texs)
	k.defines = len(base.ShaderDefines)
	k.cutTex = e.albedoTex
	// A NaN never equals itself: such a key would found a new group each frame.
	if same := *k; same != *k {
		return false
	}
	return true
}

// batchLayout finds the private material fields the key has to compare. G3N
// has no getters for them, and two entities that differ only in, say, shininess
// must not share a draw. The go.mod pins G3N; if a new version moves or retypes
// a field, ok stays false and every mesh is drawn on its own.
var batchLayout = findBatchLayout()

type batchFieldOffsets struct {
	ok                                                                 bool
	blending, depthMask, depthTest, depthFunc, lineWidth, polyF, polyU uintptr
	textures, stdUni, physUni, matCount                                uintptr
}

func findBatchLayout() batchFieldOffsets {
	var l batchFieldOffsets
	mat := reflect.TypeOf(material.Material{})
	field := func(t reflect.Type, name string, want reflect.Type, off *uintptr) bool {
		f, ok := t.FieldByName(name)
		if !ok || (want != nil && f.Type != want) {
			return false
		}
		*off = f.Offset
		return true
	}
	f32 := reflect.TypeOf(float32(0))
	ok := field(mat, "blending", reflect.TypeOf(material.Blending(0)), &l.blending) &&
		field(mat, "depthMask", reflect.TypeOf(false), &l.depthMask) &&
		field(mat, "depthTest", reflect.TypeOf(false), &l.depthTest) &&
		field(mat, "depthFunc", reflect.TypeOf(uint32(0)), &l.depthFunc) &&
		field(mat, "lineWidth", f32, &l.lineWidth) &&
		field(mat, "polyOffsetFactor", f32, &l.polyF) &&
		field(mat, "polyOffsetUnits", f32, &l.polyU) &&
		field(mat, "textures", reflect.TypeOf([]*texture.Texture2D(nil)), &l.textures) &&
		field(reflect.TypeOf(graphic.GraphicMaterial{}), "count", reflect.TypeOf(0), &l.matCount) &&
		field(reflect.TypeOf(material.Standard{}), "udata", nil, &l.stdUni) &&
		field(reflect.TypeOf(material.Physical{}), "udata", nil, &l.physUni)
	if !ok {
		return l
	}
	std, _ := reflect.TypeOf(material.Standard{}).FieldByName("udata")
	phys, _ := reflect.TypeOf(material.Physical{}).FieldByName("udata")
	l.ok = std.Type.Size() == 16*4 && phys.Type.Size() == 10*4
	return l
}

// nodeShown is whether a node and everything above it are visible.
func nodeShown(n *core.Node) bool {
	for {
		if !n.Visible() {
			return false
		}
		p := n.Parent()
		if p == nil {
			return true
		}
		n = p.GetNode()
	}
}

// frustumPlanes returns the six planes of a view-projection matrix, normalized,
// with the inside on the positive side.
func frustumPlanes(m *math32.Matrix4) [6][4]float32 {
	row := func(i int) [4]float32 { return [4]float32{m[i], m[4+i], m[8+i], m[12+i]} }
	r0, r1, r2, r3 := row(0), row(1), row(2), row(3)
	var out [6][4]float32
	for i := 0; i < 4; i++ {
		out[0][i] = r3[i] + r0[i]
		out[1][i] = r3[i] - r0[i]
		out[2][i] = r3[i] + r1[i]
		out[3][i] = r3[i] - r1[i]
		out[4][i] = r3[i] + r2[i]
		out[5][i] = r3[i] - r2[i]
	}
	for i := range out {
		p := &out[i]
		l := math32.Sqrt(p[0]*p[0] + p[1]*p[1] + p[2]*p[2])
		if l > 1e-9 {
			p[0], p[1], p[2], p[3] = p[0]/l, p[1]/l, p[2]/l, p[3]/l
		}
	}
	return out
}

// sphereInPlanes is false only when x, y, z, radius lies wholly outside a plane.
func sphereInPlanes(planes *[6][4]float32, s []float32) bool {
	for i := range planes {
		p := &planes[i]
		if p[0]*s[0]+p[1]*s[1]+p[2]*s[2]+p[3] < -s[3] {
			return false
		}
	}
	return true
}

// sphereInLightBox is casterInLightView for a cached sphere: X and Y only, since
// a caster behind the box can still throw shadow into it.
func sphereInLightBox(l *math32.Matrix4, s []float32) bool {
	return sphereInClip(l, math32.Vector3{X: s[0], Y: s[1], Z: s[2]}, s[3])
}

// batchVertexSource gives a lit vertex shader its instanced variant: with
// MB_INSTANCED defined the matrices that were uniforms come from an attribute.
// The normal matrix is the cofactor of the model-view; the shader normalizes.
func batchVertexSource(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	for _, line := range []string{
		"uniform mat4 ModelViewMatrix;\n", "uniform mat3 NormalMatrix;\n",
		"uniform mat4 MVP;\n", "uniform mat4 ModelMatrix;\n",
	} {
		src = strings.Replace(src, line, "", 1)
	}
	src = strings.Replace(src, "#include <attributes>\n", `#include <attributes>
#ifdef MB_INSTANCED
layout(location = 12) in mat4 BatchModel;
uniform mat4 BatchView;
uniform mat4 BatchProj;
mat4 ModelMatrix;
mat4 ModelViewMatrix;
mat4 MVP;
mat3 NormalMatrix;
#else
uniform mat4 ModelMatrix;
uniform mat4 ModelViewMatrix;
uniform mat4 MVP;
uniform mat3 NormalMatrix;
#endif
`, 1)
	return strings.Replace(src, "void main() {\n", `void main() {
#ifdef MB_INSTANCED
    ModelMatrix = BatchModel;
    ModelViewMatrix = BatchView * BatchModel;
    MVP = BatchProj * ModelViewMatrix;
    mat3 batchMV = mat3(ModelViewMatrix);
    NormalMatrix = mat3(cross(batchMV[1], batchMV[2]), cross(batchMV[2], batchMV[0]), cross(batchMV[0], batchMV[1]));
    if (determinant(batchMV) < 0.0) { NormalMatrix = -NormalMatrix; }
#endif
`, 1)
}

// registerBatchShaders replaces G3N's standard vertex shader with its instanced
// form. Without the define it compiles to what it was.
func registerBatchShaders(add func(name, source string)) {
	add("standard_vertex", batchVertexSource(shaders.ShaderSource("standard_vertex")))
}

const batchDepthVertexLean = `#version 330 core
layout(location = 0) in vec3 VertexPosition;
layout(location = 12) in mat4 BatchModel;
uniform mat4 BatchLightVP;
void main() {
    gl_Position = BatchLightVP * BatchModel * vec4(VertexPosition, 1.0);
}
`

const batchDepthVertexFull = `#version 330 core
` + depthAttribsGLSL + `
layout(location = 12) in mat4 BatchModel;
uniform mat4 BatchLightVP;
void main() {
    gl_Position = BatchLightVP * BatchModel * vec4(depthPosition(), 1.0);
}
`

func (w *World) batchDepthPrograms(gs *gls.GLS) bool {
	b := &w.batch
	if b.depthTried {
		return b.depth[0][0] != nil
	}
	b.depthTried = true
	var p [2][2]*gls.Program
	for moment, fs := range []string{depthEmptyFragmentSrc, depthFragmentSrc} {
		for full, vs := range []string{batchDepthVertexLean, batchDepthVertexFull} {
			prog, err := compileShadowProgram(gs, vs, fs, "batch depth")
			if err != nil {
				fmt.Println("batching:", err)
				return false
			}
			p[moment][full] = prog
		}
	}
	b.depth = p
	return true
}

// batchCutsOut is true when the shadow pass must read this entity's texture for
// holes. Those casters are still drawn one by one, as before.
func (w *World) batchCutsOut(e *Entity) bool {
	s := &w.shadow
	return e.albedoTex != nil && s.depthAlphaG3N != nil && s.depthAlphaG3N.Handle() != 0 &&
		s.filter != shadowFilterEVSM && s.filter != shadowFilterMSM
}

// batchDepthDrawn is whether the shadow pass leaves this entity to its group.
func (w *World) batchDepthDrawn(e *Entity) bool {
	return batchDrawn(e) && !w.batchCutsOut(e) && w.batch.depth[0][0] != nil
}

// drawBatchDepth draws every group into the current shadow tile.
func (w *World) drawBatchDepth(gs *gls.GLS, lightVP *math32.Matrix4, layer int) int {
	b := &w.batch
	if (len(b.active) == 0 && len(b.merges) == 0) || !w.batchDepthPrograms(gs) {
		return 0
	}
	s := &w.shadow
	moment := 0
	if s.filter == shadowFilterEVSM || s.filter == shadowFilterMSM {
		moment = 1
	}
	b.inDepth = true
	defer func() { b.inDepth = false }()
	drew := 0
	for _, g := range b.active {
		if g.key.cutTex != nil && w.batchCutsOut(g.members[0].e) {
			continue
		}
		geom := g.key.geom
		if s.primed == nil || !s.primed[geom] {
			continue
		}
		keep := func(i int) bool {
			if !g.caster[i] {
				return false
			}
			if layer == shadowLayerStatic && !g.static[i] || layer == shadowLayerDynamic && g.static[i] {
				return false
			}
			return sphereInLightBox(lightVP, g.spheres[i*4:i*4+4])
		}
		binned := g.lod != nil && w.binDepthLODs(g, lightVP, keep)
		n := 0
		if binned {
			for i := range b.lodBins {
				n += len(b.lodBins[i])
			}
		} else {
			n = g.cullToScratch(b, keep)
		}
		if n == 0 {
			continue
		}
		if !g.key.pbr && g.key.side == material.SideDouble {
			gs.Disable(gls.CULL_FACE)
		} else {
			gs.Enable(gls.CULL_FACE)
			gs.CullFace(gls.BACK)
		}
		full := 0
		if s.attribsBound == nil {
			s.attribsBound = map[*geometry.Geometry]bool{}
		}
		// A simpler version is new to G3N, so its first draw reads all attributes.
		if !s.attribsBound[geom] || binned {
			full = 1
			s.attribsBound[geom] = true
		}
		prog := b.depth[moment][full]
		gs.UseProgram(prog)
		gs.UniformMatrix4fv(prog.GetUniformLocation("BatchLightVP"), 1, false, &lightVP[0])
		if moment == 1 {
			w.setDepthMomentUniformsFor(gs, prog)
		}
		if binned {
			drew += w.drawLODBins(g, gs)
			continue
		}
		geom.RenderSetup(gs)
		w.drawBatchInstances(geom, n)
		drew++
	}
	if layer != shadowLayerStatic {
		drew += w.drawMergedDepth(gs, lightVP, func(geom *geometry.Geometry) *gls.Program {
			full := 0
			if !s.attribsBound[geom] {
				full = 1
				if s.attribsBound == nil {
					s.attribsBound = map[*geometry.Geometry]bool{}
				}
				s.attribsBound[geom] = true
			}
			return b.depth[moment][full]
		})
	}
	return drew
}

// sceneRoot is what G3N is handed to render while groups exist. Batched meshes
// are leaves with nothing to draw, and their matrices are read once a frame in
// updateBatches, so G3N need not visit them again in every pass.
type sceneRoot struct {
	*core.Node
	w     *World
	kids  []core.INode
	frame int
	n     int
}

func (r *sceneRoot) Children() []core.INode {
	real := r.Node.Children()
	if r.frame == r.w.renderSerial && r.n == len(real) {
		return r.kids
	}
	r.frame, r.n = r.w.renderSerial, len(real)
	r.kids = r.kids[:0]
	for _, c := range real {
		if m, ok := c.(*graphic.Mesh); ok && !m.Renderable() && len(m.Children()) == 0 {
			continue
		}
		r.kids = append(r.kids, c)
	}
	return r.kids
}

func (r *sceneRoot) UpdateMatrixWorld() {
	for _, c := range r.Children() {
		c.UpdateMatrixWorld()
	}
}

// renderRoot starts a render pass and returns the node to give G3N.
func (w *World) renderRoot() core.INode {
	w.pass.serial++
	if len(w.batch.active) == 0 && len(w.batch.merges) == 0 {
		return w.scene
	}
	if w.batch.root == nil || w.batch.root.Node != w.scene {
		w.batch.root = &sceneRoot{Node: w.scene, w: w}
	}
	return w.batch.root
}

// passBinds remembers which programs already hold this pass's plain uniforms.
// A program keeps its uniforms between draws, so binding the shadow and fog
// blocks for each of hundreds of terrain chunks only repeats them. Textures are
// not remembered: units are shared between programs and must be bound per draw.
type passBinds struct {
	serial  uint64
	seen    map[passBindKey]uint64
	terrain map[*gls.Program]passTerrain
}

type passBindKey struct {
	prog *gls.Program
	kind int
}

type passTerrain struct {
	serial uint64
	t      *terrain
}

const (
	passShadow = iota
	passFog
)

var glsProgOffset = func() uintptr {
	f, ok := reflect.TypeOf(gls.GLS{}).FieldByName("prog")
	if !ok || f.Type != reflect.TypeOf((*gls.Program)(nil)) {
		return 0
	}
	return f.Offset
}()

func currentProgram(gs *gls.GLS) *gls.Program {
	if glsProgOffset == 0 {
		return nil
	}
	return *(**gls.Program)(unsafe.Add(unsafe.Pointer(gs), glsProgOffset))
}

// firstInPass is true the first time a kind of uniforms is bound on the current
// program in this pass. Outside a known program it is always true.
func (w *World) firstInPass(gs *gls.GLS, kind int) bool {
	p := currentProgram(gs)
	if p == nil || w.pass.serial == 0 || w.batch.off {
		return true
	}
	if w.pass.seen == nil {
		w.pass.seen = map[passBindKey]uint64{}
	}
	k := passBindKey{p, kind}
	if w.pass.seen[k] == w.pass.serial {
		return false
	}
	w.pass.seen[k] = w.pass.serial
	return true
}

// firstTerrainInPass is firstInPass for one terrain's uniforms.
func (w *World) firstTerrainInPass(gs *gls.GLS, t *terrain) bool {
	p := currentProgram(gs)
	if p == nil || w.pass.serial == 0 || w.batch.off {
		return true
	}
	if w.pass.terrain == nil {
		w.pass.terrain = map[*gls.Program]passTerrain{}
	}
	if last := w.pass.terrain[p]; last.serial == w.pass.serial && last.t == t {
		return false
	}
	w.pass.terrain[p] = passTerrain{w.pass.serial, t}
	return true
}

// batchedCount is how many entities groups draw this frame.
func (w *World) batchedCount() int {
	n := 0
	for _, g := range w.batch.active {
		n += len(g.members)
	}
	return n
}
