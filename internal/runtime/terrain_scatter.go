package runtime

import (
	"math"
	"math/rand"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"

	"moonbasic/internal/value"
)

// scatterLayer is one repeated mesh on a terrain (grass cards, trees, or props).
// Copies follow chunk load and unload. They are not heightfield, collision, or nav.
type scatterLayer struct {
	kind    string
	mesh    int
	density float32
	spacing float32
}

type scatterPt struct {
	x, y, z float32
	yaw     float32
	scale   float32
}

// treeCard is one batch of tree sprites. face turns them toward the camera.
// crossed adds a second card at 90 degrees.
type treeCard struct {
	ent           int
	pts           []scatterPt
	width, height float32
	crossed       bool
	face          bool
	// Where the cards were last turned toward, and the batch's middle.
	// A far batch barely changes angle, so it is skipped until it would.
	lastX, lastZ float32
	midX, midZ   float32
	rad          float32
	faced        bool
}

// lodStep is the CPU stand-in for a screen-space triangle target.
// detailPx 0 keeps the old thresholds. Smaller pixels keep the fine mesh farther out.
func (t *terrain) lodStep(dist2 float32) int {
	if t == nil || (t.lod <= 0 && t.detailPx <= 0) {
		return 1
	}
	s := lodScale(t)
	step := 1
	if dist2 > 2*s*s {
		step = 2
	}
	far := float32(8)
	if t.lod > 1 {
		far = 4
	}
	if dist2 > far*s*s {
		step = 4
	}
	return step
}

// lodBuild is the step plus a geomorph weight toward the next coarser grid.
// The weight stays 0 until the last third of the band, then rises to 1.
func (t *terrain) lodBuild(dist2 float32) (int, float32) {
	step := t.lodStep(dist2)
	if t == nil || t.morphOff || step >= 4 || (t.lod <= 0 && t.detailPx <= 0) {
		return step, 0
	}
	s := lodScale(t)
	prev, limit := float32(0), 2*s*s
	if step >= 2 {
		prev = 2 * s * s
		far := float32(8)
		if t.lod > 1 {
			far = 4
		}
		limit = far * s * s
	}
	span := limit - prev
	if span <= 0 {
		return step, 0
	}
	u := (dist2 - prev) / span
	if u < 0.65 {
		return step, 0
	}
	m := (u - 0.65) / 0.35
	if m > 1 {
		m = 1
	}
	return step, m
}

func lodScale(t *terrain) float32 {
	if t == nil || t.detailPx <= 0 {
		return 1
	}
	s := 8 / t.detailPx
	if s < 0.2 {
		s = 0.2
	}
	if s > 4 {
		s = 4
	}
	return s
}

// morphLevels is how many rebuilds a chunk takes across its morph band.
// Each one slides the verts an eighth of the way to the coarser lattice.
const morphLevels = 8

func morphLevel(morph float32) float32 {
	if morph <= 0 {
		return 0
	}
	if morph >= 1 {
		return 1
	}
	return float32(int(morph*morphLevels+0.5)) / morphLevels
}

func lodCode(step int, morph float32) int {
	return step + 8*int(morphLevel(morph)*morphLevels+0.5)
}

func scatterBudget(kind string, density, size float32, step int) (n int, spacing float32) {
	if density <= 0 || size <= 0 {
		return 0, 1
	}
	cell := float32(1.7)
	cap := 48
	spacing = 1.15
	switch kind {
	case "tree":
		// A stand, not one trunk: spacing across the chunk, halved on far tiles.
		spacing = 6
		if size > 0 {
			n = int(density * (size / spacing) * (size / spacing))
		}
		if step >= 2 {
			n = n / 2
		}
		if step >= 4 {
			n = n / 2
		}
		// 24 per 32 m chunk. A large map's 96 m chunk gets the same stand
		// density, up to 160.
		cap := int(24 * (size / 32) * (size / 32))
		if cap < 24 {
			cap = 24
		}
		if cap > 160 {
			cap = 160
		}
		if n > cap {
			n = cap
		}
		if n < 1 && density > 0 {
			n = 1
		}
		return n, spacing
	case "prop":
		cell, cap, spacing = 4.5, 12, 2.8
	case "any":
		cell, cap, spacing = 5, 12, 3
	}
	n = int(density * (size / cell) * (size / cell) * 0.22)
	if step >= 2 {
		n = n / 3
	}
	if n > cap {
		n = cap
	}
	if n < 1 && step < 2 {
		n = 1
	}
	return n, spacing
}

// scatterAccept is the slope and height filter. flat is the up-component of the
// ground normal (1 is level). Grass and trees want that flat ground, above the
// shore and below snow. Props sit on anything that is not a cliff or a snowfield.
func scatterAccept(kind string, flat, h, water, snow float32, snowOn bool) bool {
	switch kind {
	case "grass", "tree":
		if flat < 0.72 {
			return false
		}
		if h < water+0.35 {
			return false
		}
		if snowOn && h > snow-0.4 {
			return false
		}
		return true
	case "prop":
		if flat < 0.38 {
			return false
		}
		if h < water+0.1 {
			return false
		}
		if snowOn && h > snow {
			return false
		}
		return true
	default:
		if flat < 0.30 {
			return false
		}
		if h < water {
			return false
		}
		return true
	}
}

func scatterFlat(t *terrain, x, z float32) float32 {
	if t == nil {
		return 1
	}
	const eps = float32(0.45)
	h := t.hf.atWorld
	hl, hr := h(x-eps, z), h(x+eps, z)
	hd, hu := h(x, z-eps), h(x, z+eps)
	nx := hl - hr
	ny := 2 * eps
	nz := -(hd - hu)
	l := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if l < 1e-5 {
		return 1
	}
	return ny / l
}

func (t *terrain) setScatterLayer(kind string, mesh int, density, spacing float32) {
	if t == nil {
		return
	}
	keep := make([]scatterLayer, 0, len(t.scatters))
	for _, s := range t.scatters {
		if s.kind == kind && s.mesh == mesh {
			continue
		}
		keep = append(keep, s)
	}
	if density > 0 {
		if spacing <= 0 {
			_, spacing = scatterBudget(kind, 1, 32, 1)
		}
		keep = append(keep, scatterLayer{kind: kind, mesh: mesh, density: density, spacing: spacing})
	}
	t.scatters = keep
}

// scatterStep is the part of a chunk's LOD its copies depend on. Trees and
// props are the same at every distance; only grass thins on far chunks.
func (t *terrain) scatterStep(step int) int {
	for _, l := range t.scatters {
		if l.kind == "grass" {
			if step >= 2 {
				return 2
			}
			return 1
		}
	}
	return 1
}

func (w *World) freeScatter(t *terrain, key chunkKey) {
	if t == nil {
		return
	}
	delete(t.scatterAt, key)
	if t.scatterIDs == nil {
		return
	}
	w.dropTreeCards(t, t.scatterIDs[key])
	for _, id := range t.scatterIDs[key] {
		w.releaseInstance(id)
	}
	delete(t.scatterIDs, key)
}

func (w *World) refreshScatter(t *terrain) {
	if t == nil {
		return
	}
	w.releaseMeshTrees(t)
	w.treeParts = nil
	cw := t.chunkWorld()
	for k := range t.ents {
		x0 := float32(k.X) * cw
		z0 := float32(k.Z) * cw
		w.placeScatter(t, k, x0, z0, cw, w.scatterStep(t, k))
	}
	if t.horizonN > 0 {
		w.placeHorizonTrees(t, cw, t.horizonAt, t.horizonN)
	}
}

func (w *World) scatterStep(t *terrain, key chunkKey) int {
	ox, oz := float32(0), float32(0)
	if w.cam != nil {
		p := worldPos(w.cam.GetNode())
		ox, _, oz = fromG3N(p.X, p.Y, p.Z)
	}
	c := chunkOf(ox, oz, t.chunkWorld())
	dx := float32(key.X - c.X)
	dz := float32(key.Z - c.Z)
	return t.lodStep(dx*dx + dz*dz)
}

func (w *World) placeScatter(t *terrain, key chunkKey, x0, z0, size float32, step int) {
	if t == nil {
		return
	}
	w.freeScatter(t, key)
	if t.scatterAt == nil {
		t.scatterAt = map[chunkKey]int{}
	}
	t.scatterAt[key] = t.scatterStep(step)
	if len(t.scatters) == 0 {
		return
	}
	if t.scatterIDs == nil {
		t.scatterIDs = map[chunkKey][]int{}
	}
	var ids []int
	for i, layer := range t.scatters {
		if (layer.kind == "tree" || layer.kind == "prop") && layer.mesh == 0 {
			continue
		}
		if w.isMeshTreeLayer(layer) {
			continue // syncMeshTrees owns these by chunk
		}
		// Trees and props use the near set everywhere. The same chunk then
		// keeps the same copies whether it is near, far, or horizon.
		ls := step
		if layer.kind != "grass" {
			ls = 1
		}
		pts := w.scatterPoints(t, layer, key, i, x0, z0, size, ls)
		if len(pts) == 0 {
			continue
		}
		far := ls >= 2
		if layer.kind == "tree" || layer.kind == "grass" || layer.mesh == 0 {
			var tex *texture.Texture2D
			crossed := !far && layer.kind == "grass"
			if layer.kind == "tree" {
				crossed = t.treeCross()
				if src := w.ents[layer.mesh]; src != nil {
					tex = src.albedoTex
				}
			}
			id := w.addCardScatter(t, layer.kind, pts, crossed, nil, tex)
			if id != 0 {
				ids = append(ids, id)
				if layer.kind == "tree" {
					tw, th := t.treeDims(tex)
					t.cards = append(t.cards, treeCard{
						ent: id, pts: append([]scatterPt(nil), pts...),
						width: tw, height: th, crossed: crossed, face: t.treeFace(),
					})
				}
			}
			continue
		}
		// Same planting as 3D trees: the template's scale and child parts
		// come along, and the batch is frustum culled.
		if parts, _ := w.treeMeshParts(layer.mesh); len(parts) > 0 {
			ids = append(ids, w.plantMeshTrees(layer.mesh, pts, false)...)
		} else if id := w.plantInstances(layer.mesh, pts); id != 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) > 0 {
		t.scatterIDs[key] = ids
	}
}

func (w *World) scatterPoints(t *terrain, layer scatterLayer, key chunkKey, salt int, x0, z0, size float32, step int) []scatterPt {
	n, spacing := scatterBudget(layer.kind, layer.density, size, step)
	if layer.spacing > 0 {
		spacing = layer.spacing
	}
	if sc := w.scatterScale(); sc < 1 {
		n = int(float32(n)*sc + 0.5)
	}
	if n <= 0 {
		return nil
	}
	seed := int64(key.X)*73856093 ^ int64(key.Z)*19349663 ^ int64(salt+1)*83492791 ^ int64(layer.mesh)*17
	rng := rand.New(rand.NewSource(seed))
	pts := make([]scatterPt, 0, n)
	space2 := spacing * spacing
	tries := n * 10
	if tries > 400+n*2 {
		tries = 400 + n*2
	}
	// Trees grow in stands with thinner ground between, not an even
	// plantation. The clearings keep a fifth of the trees, so a small map
	// that falls in one is not bare.
	var forest *noise2
	if layer.kind == "tree" {
		forest = newNoise2(7919)
	}
	for i := 0; i < tries && len(pts) < n; i++ {
		x := x0 + rng.Float32()*size
		z := z0 + rng.Float32()*size
		h := w.sampleHeight(&t.hf, x, z)
		if !scatterAccept(layer.kind, scatterFlat(t, x, z), h, t.waterY, t.snowH, t.snowOn) {
			continue
		}
		if forest != nil {
			m := forest.fbm(float64(x)*0.0045, float64(z)*0.0045, 3, 0.5, 2)
			if float64(rng.Float32()) > 0.2+0.8*smoothstep64(-0.15, 0.25, m) {
				continue
			}
		}
		ok := true
		for _, p := range pts {
			dx, dz := p.x-x, p.z-z
			if dx*dx+dz*dz < space2 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		pts = append(pts, scatterPt{
			x: x, y: h, z: z,
			yaw:   rng.Float32() * 360,
			scale: 0.85 + rng.Float32()*0.3,
		})
	}
	if step >= 2 && w.cam != nil && len(pts) > 0 && (layer.kind != "tree" || t.treeFace()) {
		p := worldPos(w.cam.GetNode())
		cx, _, cz := fromG3N(p.X, p.Y, p.Z)
		for i := range pts {
			dx := cx - pts[i].x
			dz := cz - pts[i].z
			pts[i].yaw = float32(math.Atan2(float64(dx), float64(dz)) * 180 / math.Pi)
		}
	}
	return pts
}

// scatterPlantY puts the bottom of a centered mesh on the ground.
// A cone's base sits at -height/2, so planting at the sample height buries it.
func scatterPlantY(ground, minY, scale float32) float32 {
	if scale == 0 {
		scale = 1
	}
	return ground - minY*scale
}

func geomMinY(tris [][3]math32.Vector3) float32 {
	if len(tris) == 0 {
		return 0
	}
	m := tris[0][0].Y
	for _, tri := range tris {
		for _, v := range tri {
			if v.Y < m {
				m = v.Y
			}
		}
	}
	return m
}

func (w *World) freeHorizonTrees(t *terrain) {
	if t == nil {
		return
	}
	w.dropTreeCards(t, t.horizonTrees)
	for _, id := range t.horizonTrees {
		w.releaseInstance(id)
	}
	t.horizonTrees = nil
}

func (w *World) dropTreeCards(t *terrain, ids []int) {
	if t == nil || len(ids) == 0 || len(t.cards) == 0 {
		return
	}
	drop := map[int]bool{}
	for _, id := range ids {
		drop[id] = true
	}
	keep := t.cards[:0]
	for _, c := range t.cards {
		if !drop[c.ent] {
			keep = append(keep, c)
		}
	}
	t.cards = keep
}

func (w *World) releaseInstance(id int) {
	if im := w.instances[id]; im != nil {
		w.deleteInstanceGL(im)
		if im.batch != 0 {
			w.freeEntityID(im.batch)
			im.batch = 0
		}
		delete(w.instances, id)
	}
	w.freeEntityID(id)
}

func (w *World) plantInstances(mesh int, pts []scatterPt) int {
	if mesh == 0 || len(pts) == 0 {
		return 0
	}
	if len(pts) > 8000 {
		pts = pts[:8000]
	}
	id, err := w.createInstanced(mesh, len(pts))
	if err != nil || id == 0 {
		return 0
	}
	im := w.instances[id]
	if im == nil {
		return id
	}
	minY := geomMinY(im.tris)
	for i, p := range pts {
		im.xforms[i] = instXform{
			x: p.x, y: scatterPlantY(p.y, minY, p.scale), z: p.z,
			yaw: p.yaw, sx: p.scale, sy: p.scale, sz: p.scale,
		}
	}
	im.preferCPU = true
	im.gpuOK = false
	im.dirty = true
	w.rebuildInstanceBatch(id)
	if e := w.ents[id]; e != nil {
		e.name = "scatter"
	}
	return id
}

// placeHorizonTrees keeps the same tree picture on the coarse ring, out to the fog.
func (w *World) placeHorizonTrees(t *terrain, cw float32, cam chunkKey, n int) {
	if t == nil {
		return
	}
	w.freeHorizonTrees(t)
	reach := t.radius + 6
	// Trees past the tier's reach would be fogged cards costing CPU every frame.
	if w.quality.reach > 0 && cw > 0 {
		if r := int(w.quality.reach/cw + 0.999); r < reach {
			reach = r
		}
		if reach < t.radius+1 {
			reach = t.radius + 1
		}
	}
	if reach > n {
		reach = n
	}
	for li, layer := range t.scatters {
		if layer.kind != "tree" || layer.density <= 0 || layer.mesh == 0 || w.isMeshTreeLayer(layer) {
			continue
		}
		// The same stand placeScatter plants, so a chunk crossing into the
		// detail window keeps every tree where it was.
		var pts []scatterPt
		for z := cam.Z - reach; z < cam.Z+reach; z++ {
			for x := cam.X - reach; x < cam.X+reach; x++ {
				k := chunkKey{X: x, Z: z}
				if t.detailLoaded(cam, k) {
					continue
				}
				pts = append(pts, w.scatterPoints(t, layer, k, li, float32(x)*cw, float32(z)*cw, cw, 1)...)
			}
		}
		var tex *texture.Texture2D
		if src := w.ents[layer.mesh]; src != nil {
			tex = src.albedoTex
		}
		tw, th := t.treeDims(tex)
		crossed := t.treeCross()
		if id := w.addCardScatter(t, "tree", pts, crossed, nil, tex); id != 0 {
			t.horizonTrees = append(t.horizonTrees, id)
			t.cards = append(t.cards, treeCard{
				ent: id, pts: append([]scatterPt(nil), pts...),
				width: tw, height: th, crossed: crossed, face: t.treeFace(),
			})
		}
	}
}

func chunkHash01(x, z, salt int) float32 {
	h := uint32(x)*374761393 ^ uint32(z)*668265263 ^ uint32(salt+1)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float32(h%10000) / 10000
}

func treeSpriteSize(tex *texture.Texture2D) (width, height float32) {
	if tex != nil {
		return 3.6, 4.4
	}
	return 1.1, 2.4
}

func (t *terrain) treeDims(tex *texture.Texture2D) (width, height float32) {
	width, height = treeSpriteSize(tex)
	if t == nil {
		return width, height
	}
	if t.cardW > 0 {
		width = t.cardW
	}
	if t.cardH > 0 {
		height = t.cardH
	}
	return width, height
}

func (t *terrain) treeCross() bool {
	return t != nil && t.cardCross
}

func (t *terrain) treeFace() bool {
	return t == nil || t.cardFace
}

func (w *World) addCardScatter(t *terrain, kind string, pts []scatterPt, crossed bool, tint *math32.Color, tex *texture.Texture2D) int {
	width, height := float32(0.55), float32(0.9)
	col := &math32.Color{0.22, 0.48, 0.16}
	switch kind {
	case "tree":
		width, height = t.treeDims(tex)
		col = &math32.Color{1, 1, 1}
		if tex == nil {
			col = &math32.Color{0.14, 0.43, 0.19}
		}
	case "prop":
		width, height = 1.1, 1.2
		col = &math32.Color{0.42, 0.40, 0.36}
	}
	if tint != nil {
		col = tint
	}
	g := cardGeometry(pts, crossed, width, height)
	if g == nil {
		return 0
	}
	mat := w.newMat()
	mat.SetColor(col)
	mat.SetAmbientColor(col)
	mat.SetSide(material.SideDouble)
	if tex != nil {
		tex.SetWrapS(gls.CLAMP_TO_EDGE)
		tex.SetWrapT(gls.CLAMP_TO_EDGE)
		// The card's V already puts the image top at the top of the quad.
		// The default flip turns the trunk into the sky.
		tex.SetFlipY(false)
		mat.AddTexture(tex)
	}
	// Lit, so the card sets its own cutout instead of inheriting whatever
	// the previous mesh on this shader left.
	lm := newLitMat(w, mat)
	lm.cutout = 1
	mesh := graphic.NewMesh(g, lm)
	parent := 0
	if t != nil {
		parent = t.root
	}
	id := w.addEntity(&Entity{node: mesh, mesh: mesh, mat: mat, name: "foliage", kind: "scatter", albedoTex: tex}, parent)
	return id
}

// faceTreeCards turns each tree sprite toward the camera so the picture stays front-on.
func (w *World) faceTreeCards(t *terrain) {
	if t == nil || w.cam == nil || len(t.cards) == 0 {
		return
	}
	p := worldPos(w.cam.GetNode())
	cx, _, cz := fromG3N(p.X, p.Y, p.Z)
	keep := t.cards[:0]
	for _, c := range t.cards {
		e := w.ents[c.ent]
		if e == nil || e.mesh == nil {
			continue
		}
		keep = append(keep, c)
		geom := e.mesh.GetGeometry()
		if geom == nil {
			continue
		}
		pos := geom.VBO(gls.VertexPosition)
		nor := geom.VBO(gls.VertexNormal)
		if pos == nil || nor == nil {
			continue
		}
		if !c.face {
			continue
		}
		last := &keep[len(keep)-1]
		if !last.faced {
			for _, p := range c.pts {
				last.midX += p.x
				last.midZ += p.z
			}
			last.midX /= float32(len(c.pts))
			last.midZ /= float32(len(c.pts))
			for _, p := range c.pts {
				dx, dz := p.x-last.midX, p.z-last.midZ
				if r := math32.Sqrt(dx*dx + dz*dz); r > last.rad {
					last.rad = r
				}
			}
		} else {
			// Turn again once the camera has moved about 1 degree as seen
			// from the batch's nearest tree. Near batches turn every frame.
			dx, dz := cx-last.midX, cz-last.midZ
			mx, mz := cx-last.lastX, cz-last.lastZ
			lim := 0.018 * (math32.Sqrt(dx*dx+dz*dz) - last.rad)
			if lim > 0.05 && mx*mx+mz*mz < lim*lim {
				continue
			}
		}
		last.faced = true
		last.lastX, last.lastZ = cx, cz
		writeTreeSprite(pos.Buffer(), nor.Buffer(), c.pts, c.width, c.height, c.crossed, cx, cz)
		pos.Update()
		nor.Update()
	}
	t.cards = keep
}

func writeTreeSprite(pos, nor *math32.ArrayF32, pts []scatterPt, width, height float32, crossed bool, cx, cz float32) {
	if pos == nil || nor == nil {
		return
	}
	pdst, ndst := []float32(*pos), []float32(*nor)
	vi := 0
	put := func(p scatterPt, yaw float32) {
		c := math32.Cos(yaw)
		s := math32.Sin(yaw)
		hw := width * p.scale * 0.5
		hh := height * p.scale
		nx, ny, nz := toG3N(s, 0, c)
		xs := [4]float32{-hw, hw, hw, -hw}
		ys := [4]float32{0, 0, hh, hh}
		for i := 0; i < 4; i++ {
			rx := xs[i] * c
			rz := -xs[i] * s
			gx, gy, gz := toG3N(p.x+rx, p.y+ys[i], p.z+rz)
			if vi+2 < len(pdst) {
				pdst[vi], pdst[vi+1], pdst[vi+2] = gx, gy, gz
			}
			if vi+2 < len(ndst) {
				ndst[vi], ndst[vi+1], ndst[vi+2] = nx, ny, nz
			}
			vi += 3
		}
	}
	for _, p := range pts {
		yaw := float32(math.Atan2(float64(cx-p.x), float64(cz-p.z)))
		put(p, yaw)
		if crossed {
			put(p, yaw+math32.Pi/2)
		}
	}
}

func cardGeometry(pts []scatterPt, crossed bool, width, height float32) *geometry.Geometry {
	if len(pts) == 0 {
		return nil
	}
	pos := math32.NewArrayF32(0, 0)
	nor := math32.NewArrayF32(0, 0)
	uvs := math32.NewArrayF32(0, 0)
	idx := math32.NewArrayU32(0, 0)
	nverts := 0
	addQuad := func(p scatterPt, yawOff, w, h float32) {
		yaw := (p.yaw + yawOff) * math32.Pi / 180
		c := math32.Cos(yaw)
		s := math32.Sin(yaw)
		hw := w * p.scale * 0.5
		hh := h * p.scale
		local := [4][3]float32{{-hw, 0, 0}, {hw, 0, 0}, {hw, hh, 0}, {-hw, hh, 0}}
		base := uint32(nverts)
		nx, ny, nz := toG3N(s, 0, c)
		for i, v := range local {
			rx := v[0]*c + v[2]*s
			rz := -v[0]*s + v[2]*c
			gx, gy, gz := toG3N(p.x+rx, p.y+v[1], p.z+rz)
			pos.Append(gx, gy, gz)
			nor.Append(nx, ny, nz)
			switch i {
			case 0:
				uvs.Append(0, 1)
			case 1:
				uvs.Append(1, 1)
			case 2:
				uvs.Append(1, 0)
			default:
				uvs.Append(0, 0)
			}
			nverts++
		}
		idx.Append(base, base+1, base+2, base, base+2, base+3)
	}
	for _, p := range pts {
		addQuad(p, 0, width, height)
		if crossed {
			addQuad(p, 90, width, height)
		}
	}
	g := geometry.NewGeometry()
	g.SetIndices(idx)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddVBO(gls.NewVBO(nor).AddAttrib(gls.VertexNormal))
	g.AddVBO(gls.NewVBO(uvs).AddAttrib(gls.VertexTexcoord))
	return g
}

func (w *World) scatterCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	set := func(kind string, mesh bool, defDensity float64) func(a []value.Value) (value.Value, error) {
		return func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			meshID := 0
			if mesh {
				meshID = argI(a, off, 0)
				off++
			}
			dens := float32(argN(a, off, defDensity))
			t.setScatterLayer(kind, meshID, dens, 0)
			w.refreshScatter(t)
			return z()
		}
	}
	return map[string]cmd{
		"terrainfoliage": n(set("grass", false, 1)),
		"terraintrees":   n(set("tree", true, 1)),
		"terrainprops":   n(set("prop", true, 1)),
		"terrainscatter": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			meshID := argI(a, off, 0)
			kind := argS(a, off+1)
			if kind == "" {
				kind = "any"
			}
			dens := float32(argN(a, off+2, 1))
			t.setScatterLayer(kind, meshID, dens, 0)
			w.refreshScatter(t)
			return z()
		}),
		"setterrainblendmap": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			t.blendTex = argI(a, off, 0)
			return z()
		}),
		"settreecards": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			if ww := float32(argN(a, off, 0)); ww > 0 {
				t.cardW = ww
			}
			if hh := float32(argN(a, off+1, 0)); hh > 0 {
				t.cardH = hh
			}
			if len(a) > off+2 {
				t.cardCross = argI(a, off+2, 0) != 0
			}
			if len(a) > off+3 {
				t.cardFace = argI(a, off+3, 1) != 0
			}
			w.refreshScatter(t)
			return z()
		}),
		"settreecardcross": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			t.cardCross = argI(a, terrainArgOff(w, a), 0) != 0
			w.refreshScatter(t)
			return z()
		}),
		"settreecardface": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			t.cardFace = argI(a, terrainArgOff(w, a), 1) != 0
			w.refreshScatter(t)
			return z()
		}),
		"setterraindetail": n(func(a []value.Value) (value.Value, error) {
			t := w.pickTerrain(a)
			if t == nil {
				return z()
			}
			off := terrainArgOff(w, a)
			px := float32(argN(a, off, 8))
			if px < 0 {
				px = 0
			}
			t.detailPx = px
			w.dirtyTerrain(t)
			return z()
		}),
	}
}
