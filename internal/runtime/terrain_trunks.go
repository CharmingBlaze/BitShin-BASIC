package runtime

import (
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

// Trees stop a walking player. A physics body for every tree on an island would be
// thousands of bodies; only the trunks next to a character can touch it, so they are
// worked out from the same seeds that planted the trees, a chunk at a time, and the
// character is pushed out of them as a circle against circles.

type trunk struct{ x, z, r float32 }

// trunksIn is one chunk's trunks, kept once worked out.
func (w *World) trunksIn(t *terrain, k chunkKey, cw float32) []trunk {
	if got, ok := t.trunks[k]; ok {
		return got
	}
	if t.trunks == nil {
		t.trunks = map[chunkKey][]trunk{}
	}
	var out []trunk
	x0, z0 := float32(k.X)*cw, float32(k.Z)*cw
	for _, li := range w.meshTreeLayers(t) {
		layer := t.scatters[li]
		parts, minY := w.treeMeshParts(layer.mesh)
		r := partsTop(parts, minY) * 0.03
		for _, p := range w.scatterPoints(t, layer, k, li, x0, z0, cw, 1) {
			out = append(out, trunk{p.x, p.z, r * p.scale})
		}
	}
	t.trunks[k] = out
	return out
}

// pushOutOfTrunks moves a circle out of any trunk it overlaps. It slides: only the
// part of the way that is into the trunk is undone.
func pushOutOfTrunks(x, z, radius float32, trunks []trunk) (float32, float32, bool) {
	moved := false
	for _, c := range trunks {
		dx, dz := x-c.x, z-c.z
		reach := radius + c.r
		d2 := dx*dx + dz*dz
		if d2 >= reach*reach {
			continue
		}
		d := math32.Sqrt(d2)
		if d < 0.001 {
			dx, dz, d = 1, 0, 1
		}
		x = c.x + dx/d*reach
		z = c.z + dz/d*reach
		moved = true
	}
	return x, z, moved
}

// keepOutOfTrees is called for a character each frame it moves.
func (w *World) keepOutOfTrees(id int, radius float32) {
	if w.noTreeCollision || w.phys3 == nil {
		return
	}
	x, y, z, ok := w.phys3.GetPosition(id)
	if !ok {
		return
	}
	for _, t := range w.terrains {
		if t == nil || len(t.treeMesh) == 0 {
			continue
		}
		cw := t.chunkWorld()
		cx, cz := int(math32.Floor(x/cw)), int(math32.Floor(z/cw))
		nx, nz, moved := x, z, false
		for dz := -1; dz <= 1; dz++ {
			for dx := -1; dx <= 1; dx++ {
				var m bool
				nx, nz, m = pushOutOfTrunks(nx, nz, radius, w.trunksIn(t, chunkKey{X: cx + dx, Z: cz + dz}, cw))
				moved = moved || m
			}
		}
		if moved {
			w.phys3.SetPosition(id, nx, y, nz)
			return
		}
	}
}

func (w *World) trunkCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// SetTreeCollision(on): FPS and TPS characters can't walk through 3D trees. On by default.
		"settreecollision": n(func(a []value.Value) (value.Value, error) {
			w.noTreeCollision = argI(a, 0, 1) == 0
			return z()
		}),
	}
}
