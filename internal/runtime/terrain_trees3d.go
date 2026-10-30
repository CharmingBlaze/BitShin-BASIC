package runtime

import (
	"sort"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

// meshPart is one colored piece of a 3D tree template (trunk, canopy).
type meshPart struct {
	tris  [][3]math32.Vector3
	color math32.Color
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
		parts = append(parts, meshPart{tris: tris, color: col})
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
// one ground offset so the trunk stays under the canopy.
func (w *World) plantMeshTrees(mesh int, pts []scatterPt) []int {
	parts, minY := w.treeMeshParts(mesh)
	if len(parts) == 0 || len(pts) == 0 {
		return nil
	}
	var ids []int
	for _, part := range parts {
		id := w.newInstanced(nil, part.tris, part.color, len(pts), true)
		im := w.instances[id]
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
			ids = append(ids, w.plantMeshTrees(t.scatters[li].mesh, pts)...)
		}
		t.treeMesh[b] = ids
	}
}
