package runtime

import (
	"math/rand"

	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

// Grass near the camera: real blades in cells that follow it, thinning with
// distance and dissolving at the edge. Blades point their normals up, so they
// light like the ground they grow from and not like a field of little cards
// (the usual trick; Ghost of Tsushima and Godot's grass demos do the same).
// Grass casts no shadow, and each blade is one triangle: it is the crowd of them
// that reads as grass.

const (
	grassCell   = 12 // metres a side of one batch
	grassReach  = 46 // metres from the camera to where grass has dissolved
	grassNearTo = 24 // cells nearer than this get full density
	grassPerM2  = 6  // tufts on a square metre, at amount 1
)

type grassCellState struct {
	id   int
	near bool
}

type grassField struct {
	amount, height float32
	cells          map[chunkKey]grassCellState
	tuft           meshPart
}

// grassTuft is a few blades in one template, instanced over a cell.
func grassTuft(height float32) meshPart {
	r := rand.New(rand.NewSource(4177))
	p := meshPart{color: math32.Color{R: 0.4, G: 0.46, B: 0.22}}
	for i := 0; i < 5; i++ {
		x, z := (r.Float32()-0.5)*0.75, (r.Float32()-0.5)*0.75
		a := r.Float32() * 2 * math32.Pi
		c, s := math32.Cos(a), math32.Sin(a)
		w := 0.022 + r.Float32()*0.018
		h := height * (0.65 + r.Float32()*0.5)
		lean := h * (0.15 + r.Float32()*0.3)
		base := uint32(len(p.verts))
		up := math32.Vector3{Y: 1}
		p.verts = append(p.verts,
			math32.Vector3{X: x - s*w, Z: z + c*w},
			math32.Vector3{X: x + s*w, Z: z - c*w},
			math32.Vector3{X: x + c*lean, Y: h, Z: z + s*lean})
		p.vnors = append(p.vnors, up, up, up)
		tip := 0.85 + r.Float32()*0.25
		p.shades = append(p.shades, 0.55, 0.55, tip)
		p.idx = append(p.idx, base, base+1, base+2)
	}
	return p
}

func grassKeyDist(k chunkKey, cx, cz float32) float32 {
	dx := (float32(k.X)+0.5)*grassCell - cx
	dz := (float32(k.Z)+0.5)*grassCell - cz
	return math32.Sqrt(dx*dx + dz*dz)
}

// grassPoints plants one cell. The same cell always gets the same tufts, and
// near or far only changes how many of them are kept.
func (w *World) grassPoints(t *terrain, g *grassField, k chunkKey, near bool) []scatterPt {
	r := rand.New(rand.NewSource(int64(k.X)*73856093 ^ int64(k.Z)*19349663))
	n := int(g.amount * grassPerM2 * grassCell * grassCell * w.scatterScale())
	shore := t.waterY + math32.Max(t.blend, 1.2)*2
	pts := make([]scatterPt, 0, n)
	for i := 0; i < n; i++ {
		x := (float32(k.X) + r.Float32()) * grassCell
		z := (float32(k.Z) + r.Float32()) * grassCell
		yaw, sc := r.Float32()*360, 0.8+r.Float32()*0.6
		if !near && i%2 == 0 {
			continue
		}
		h := w.sampleHeight(&t.hf, x, z)
		if h < shore {
			continue
		}
		flat := scatterFlat(t, x, z)
		if !scatterAccept("grass", flat, h, t.waterY, snowLineAt(t, x, z, flat), t.snowOn) {
			continue
		}
		// Thin patches, so a meadow is not a mown lawn.
		if terrainNoiseGo(x*0.09, z*0.09) < 0.15 {
			continue
		}
		if !near {
			sc *= 1.5
		}
		pts = append(pts, scatterPt{x: x, y: h - 0.03, z: z, yaw: yaw, scale: sc})
	}
	return pts
}

func (w *World) buildGrassCell(t *terrain, g *grassField, k chunkKey, near bool) int {
	pts := w.grassPoints(t, g, k, near)
	if len(pts) == 0 {
		return 0
	}
	arrs := buildTreeBatches([]meshPart{g.tuft}, 0, g.height, pts, true)
	if len(arrs) == 0 {
		return 0
	}
	id := w.attachTreeBatch(arrs[0])
	if e := w.ents[id]; e != nil && e.mesh != nil {
		if lm, ok := e.mesh.Materials()[0].IMaterial().(*litMat); ok {
			lm.grassFade = grassReach
			lm.SetSide(material.SideDouble)
		}
		e.meshNoCast = true
	}
	return id
}

// syncGrass keeps cells around the camera, nearest first, and builds at most a
// few a frame so walking never stalls. The first frame fills the whole field.
func (w *World) syncGrass(t *terrain) {
	g := t.grass
	if g == nil || w.cam == nil {
		return
	}
	p := worldPos(w.cam.GetNode())
	cx, _, cz := fromG3N(p.X, p.Y, p.Z)
	for k, c := range g.cells {
		if grassKeyDist(k, cx, cz) > grassReach+grassCell {
			if c.id != 0 {
				w.releaseInstance(c.id)
			}
			delete(g.cells, k)
		}
	}
	budget := 3
	if len(g.cells) == 0 {
		budget = 1 << 30
	}
	n := int(grassReach/grassCell) + 1
	kx, kz := int(math32.Floor(cx/grassCell)), int(math32.Floor(cz/grassCell))
	// Rings outward from the camera, so the nearest cells are built first.
	for ring := 0; ring <= n && budget > 0; ring++ {
		for z := kz - ring; z <= kz+ring && budget > 0; z++ {
			for x := kx - ring; x <= kx+ring && budget > 0; x++ {
				if x != kx-ring && x != kx+ring && z != kz-ring && z != kz+ring {
					continue
				}
				k := chunkKey{X: x, Z: z}
				d := grassKeyDist(k, cx, cz)
				if d > grassReach {
					continue
				}
				near := d < grassNearTo
				c, ok := g.cells[k]
				if ok && c.near == near {
					continue
				}
				if ok && c.id != 0 {
					w.releaseInstance(c.id)
				}
				g.cells[k] = grassCellState{id: w.buildGrassCell(t, g, k, near), near: near}
				budget--
			}
		}
	}
}

// resetGrass drops the cells, to be planted again: the ground or the density changed.
func (w *World) resetGrass(t *terrain) {
	if t == nil || t.grass == nil {
		return
	}
	for k, c := range t.grass.cells {
		if c.id != 0 {
			w.releaseInstance(c.id)
		}
		delete(t.grass.cells, k)
	}
}

// setTerrainGrass turns the field on, or off at amount 0.
func (w *World) setTerrainGrass(t *terrain, amount, height float32) {
	w.resetGrass(t)
	t.grass = nil
	if amount <= 0 {
		return
	}
	t.grass = &grassField{amount: amount, height: height, cells: map[chunkKey]grassCellState{}, tuft: grassTuft(height)}
}
