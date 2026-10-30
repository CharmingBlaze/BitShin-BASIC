package runtime

import (
	"sort"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
)

// meshPart is one colored piece of a 3D tree template (trunk, canopy).
type meshPart struct {
	tris  [][3]math32.Vector3
	nors  [][3]math32.Vector3 // the template's own normals, per tri corner
	color math32.Color
	// Shared-vertex form, for batches with a third of the vertex work.
	verts, vnors []math32.Vector3
	idx          []uint32
}

// treeMeshParts reads a 3D tree template: its own mesh and its direct
// children's, each in the template's space with its own color. The root's
// scale and turn are kept. Its position is not; copies go on the ground.
// A textured template returns nil and stays a picture card.
func (w *World) treeMeshParts(id int) ([]meshPart, float32) {
	if c, ok := w.treeParts[id]; ok {
		return c.parts, c.minY
	}
	parts, minY := w.readTreeParts(id)
	if w.treeParts == nil {
		w.treeParts = map[int]treePartCache{}
	}
	w.treeParts[id] = treePartCache{parts, minY}
	return parts, minY
}

// treePartCache holds a template's parts. TerrainTrees clears it, so a
// template edited before that call is read again.
type treePartCache struct {
	parts []meshPart
	minY  float32
}

func (w *World) readTreeParts(id int) (parts []meshPart, minY float32) {
	root := w.ents[id]
	if root == nil || root.node == nil || root.albedoTex != nil {
		return nil, 0
	}
	base := localMatrix(root.node.GetNode())
	base[12], base[13], base[14] = 0, 0, 0
	add := func(e *Entity, m math32.Matrix4) {
		if e == nil || e.mesh == nil || e.mesh.GetGeometry() == nil {
			return
		}
		tris := geomWorldTris(e.mesh.GetGeometry(), m)
		if len(tris) == 0 {
			return
		}
		col := math32.Color{0.2, 0.45, 0.2}
		if e.mat != nil {
			col = e.mat.AmbientColor()
		}
		part := meshPart{tris: tris, nors: geomTriNormals(e.mesh.GetGeometry(), m), color: col}
		part.verts, part.vnors, part.idx = geomIndexed(e.mesh.GetGeometry(), m)
		parts = append(parts, part)
	}
	add(root, base)
	var kids []int
	for cid, e := range w.ents {
		if e != nil && e.parent == id && e.mesh != nil {
			kids = append(kids, cid)
		}
	}
	sort.Ints(kids)
	for _, cid := range kids {
		cm := localMatrix(w.ents[cid].node.GetNode())
		var m math32.Matrix4
		m.MultiplyMatrices(&base, &cm)
		add(w.ents[cid], m)
	}
	first := true
	for _, p := range parts {
		if y := geomMinY(p.tris); first || y < minY {
			minY, first = y, false
		}
	}
	return parts, minY
}

// geomTriNormals is each triangle corner's normal, in ReadFaces order, under
// m. nil when the geometry has none (the batch then uses face normals).
func geomTriNormals(g *geometry.Geometry, m math32.Matrix4) [][3]math32.Vector3 {
	if g == nil || g.VBO(gls.VertexNormal) == nil {
		return nil
	}
	nor := readAttrib(g, gls.VertexNormal, 3)
	n := len(nor) / 3
	var nm math32.Matrix3
	nm.GetNormalMatrix(&m)
	at := func(i uint32) math32.Vector3 {
		if int(i) >= n {
			return math32.Vector3{Y: 1}
		}
		v := math32.Vector3{X: nor[i*3], Y: nor[i*3+1], Z: nor[i*3+2]}
		v.ApplyMatrix3(&nm)
		v.Normalize()
		return v
	}
	var out [][3]math32.Vector3
	if idx := g.Indices(); len(idx) > 0 {
		for i := 0; i+2 < len(idx); i += 3 {
			out = append(out, [3]math32.Vector3{at(idx[i]), at(idx[i+1]), at(idx[i+2])})
		}
		return out
	}
	for i := uint32(0); int(i)+2 < n; i += 3 {
		out = append(out, [3]math32.Vector3{at(i), at(i + 1), at(i + 2)})
	}
	return out
}

// geomIndexed is the geometry's own vertices, normals, and indices under m.
func geomIndexed(g *geometry.Geometry, m math32.Matrix4) (verts, nors []math32.Vector3, idx []uint32) {
	pos := readAttrib(g, gls.VertexPosition, 3)
	nor := readAttrib(g, gls.VertexNormal, 3)
	n := len(pos) / 3
	if n == 0 {
		return nil, nil, nil
	}
	var nm math32.Matrix3
	nm.GetNormalMatrix(&m)
	verts = make([]math32.Vector3, n)
	nors = make([]math32.Vector3, n)
	for i := 0; i < n; i++ {
		v := math32.Vector3{X: pos[i*3], Y: pos[i*3+1], Z: pos[i*3+2]}
		v.ApplyMatrix4(&m)
		verts[i] = v
		nv := math32.Vector3{Y: 1}
		if len(nor) >= (i+1)*3 {
			nv = math32.Vector3{X: nor[i*3], Y: nor[i*3+1], Z: nor[i*3+2]}
			nv.ApplyMatrix3(&nm)
			nv.Normalize()
		}
		nors[i] = nv
	}
	if gi := g.Indices(); len(gi) > 0 {
		idx = append([]uint32(nil), gi...)
	} else {
		idx = make([]uint32, n)
		for i := range idx {
			idx[i] = uint32(i)
		}
	}
	return verts, nors, idx
}

// treeTint is a stable per-copy shade: a little lighter or darker and a
// touch warmer or cooler, so a stand is not one flat color.
func treeTint(x, z float32) math32.Color {
	h := chunkHash01(int(x*7.31), int(z*5.17), 41)
	k := chunkHash01(int(x*3.11), int(z*9.43), 59)
	b := 0.82 + 0.26*h
	warm := (k - 0.5) * 0.14
	return math32.Color{R: b * (1 + warm), G: b, B: b * (1 - warm)}
}

// localMatrix is built from the node's own position, turn, and scale.
// Node.Matrix() is stale (zero) for a node the renderer has not visited,
// and a hidden template is never visited.
func localMatrix(n *core.Node) math32.Matrix4 {
	var m math32.Matrix4
	pos, q, sc := n.Position(), n.Quaternion(), n.Scale()
	m.Compose(&pos, &q, &sc)
	return m
}

// meshTreeLayers are the tree layers drawn as 3D meshes.
func (w *World) meshTreeLayers(t *terrain) []int {
	var out []int
	for i, l := range t.scatters {
		if l.kind != "tree" || l.mesh == 0 || l.density <= 0 {
			continue
		}
		if parts, _ := w.treeMeshParts(l.mesh); len(parts) > 0 {
			out = append(out, i)
		}
	}
	return out
}

func (w *World) isMeshTreeLayer(l scatterLayer) bool {
	if l.kind != "tree" || l.mesh == 0 {
		return false
	}
	parts, _ := w.treeMeshParts(l.mesh)
	return len(parts) > 0
}

// plantMeshTrees puts one batch per template part on pts. The parts share
// one ground offset so the trunk stays under the canopy. sway is for trees.
func (w *World) plantMeshTrees(mesh int, pts []scatterPt, sway bool) []int {
	parts, minY := w.treeMeshParts(mesh)
	if len(parts) == 0 || len(pts) == 0 {
		return nil
	}
	maxY := minY
	for _, p := range parts {
		for _, t := range p.tris {
			for _, v := range t {
				maxY = math32.Max(maxY, v.Y)
			}
		}
	}
	tint := make([]math32.Color, len(pts))
	for i, p := range pts {
		tint[i] = treeTint(p.x, p.z)
	}
	var ids []int
	for _, part := range parts {
		id := w.newInstanced(nil, part.tris, part.color, len(pts), true)
		im := w.instances[id]
		im.triNor = part.nors
		im.verts, im.vnors, im.vidx = part.verts, part.vnors, part.idx
		im.tint = tint
		im.sway = sway
		im.swayLo, im.swayH = minY, maxY-minY
		for i, p := range pts {
			im.xforms[i] = instXform{
				x: p.x, y: scatterPlantY(p.y, minY, p.scale), z: p.z,
				yaw: p.yaw, sx: p.scale, sy: p.scale, sz: p.scale,
			}
		}
		w.rebuildInstanceBatch(id)
		if b := w.ents[im.batch]; b != nil && b.mesh != nil {
			// A chunk of forest behind the camera costs nothing.
			b.mesh.SetCullable(true)
			b.name = "scatter"
		}
		if e := w.ents[id]; e != nil {
			e.name = "scatter"
		}
		ids = append(ids, id)
	}
	return ids
}

func (w *World) releaseMeshTrees(t *terrain) {
	if t == nil {
		return
	}
	for k, ids := range t.treeMesh {
		for _, id := range ids {
			w.releaseInstance(id)
		}
		delete(t.treeMesh, k)
	}
}

// meshTreeReach is how far 3D trees are kept, in meters.
func (w *World) meshTreeReach() float32 {
	if w.quality.reach > 0 {
		return w.quality.reach
	}
	return 420
}

// treeBlock is how many chunks a side one 3D tree batch covers. One batch
// per part per block keeps the draw count near 100 on a large island; one
// per chunk was over 1000, and G3N pays per mesh, not per triangle.
const treeBlock = 3

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// syncMeshTrees keeps 3D trees on every block within reach of the camera,
// detail or horizon alike, so a chunk keeps its trees as it changes LOD.
// A block is planted from the same per-chunk points, so its trees do not
// depend on which block they fall in. Nearest blocks first, one per frame
// once the camera's own block is up.
func (w *World) syncMeshTrees(t *terrain, cw float32, cam chunkKey) {
	layers := w.meshTreeLayers(t)
	if len(layers) == 0 {
		if len(t.treeMesh) > 0 {
			w.releaseMeshTrees(t)
		}
		return
	}
	if t.treeMesh == nil {
		t.treeMesh = map[chunkKey][]int{}
	}
	r := w.meshTreeReach() / cw
	if r < float32(t.radius) {
		r = float32(t.radius)
	}
	// Distance in chunks from the camera chunk to a block's middle.
	dist := func(b chunkKey) float32 {
		dx := float32(b.X*treeBlock) + float32(treeBlock-1)/2 - float32(cam.X)
		dz := float32(b.Z*treeBlock) + float32(treeBlock-1)/2 - float32(cam.Z)
		return math32.Sqrt(dx*dx + dz*dz)
	}
	half := float32(treeBlock) * 0.71 // a block's corner from its middle
	keep, add := r+half+1, r+half
	for b, ids := range t.treeMesh {
		if dist(b) > keep {
			for _, id := range ids {
				w.releaseInstance(id)
			}
			delete(t.treeMesh, b)
		}
	}
	cb := chunkKey{floorDiv(cam.X, treeBlock), floorDiv(cam.Z, treeBlock)}
	n := int(add/treeBlock) + 1
	var missing []chunkKey
	for z := cb.Z - n; z <= cb.Z+n; z++ {
		for x := cb.X - n; x <= cb.X+n; x++ {
			b := chunkKey{X: x, Z: z}
			if _, ok := t.treeMesh[b]; !ok && dist(b) <= add {
				missing = append(missing, b)
			}
		}
	}
	sort.Slice(missing, func(i, j int) bool { return dist(missing[i]) < dist(missing[j]) })
	for i, b := range missing {
		if i > 0 && b != cb {
			break
		}
		ids := []int{}
		for _, li := range layers {
			var pts []scatterPt
			for z := 0; z < treeBlock; z++ {
				for x := 0; x < treeBlock; x++ {
					k := chunkKey{X: b.X*treeBlock + x, Z: b.Z*treeBlock + z}
					x0, z0 := float32(k.X)*cw, float32(k.Z)*cw
					pts = append(pts, w.scatterPoints(t, t.scatters[li], k, li, x0, z0, cw, 1)...)
				}
			}
			ids = append(ids, w.plantMeshTrees(t.scatters[li].mesh, pts, true)...)
		}
		t.treeMesh[b] = ids
	}
}
