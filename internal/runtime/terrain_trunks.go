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

// trunksNear calls f with the trunks of the 3x3 chunks around x, z on every terrain
// that has 3D trees. Stops early when f returns false.
func (w *World) trunksNear(x, z float32, f func([]trunk) bool) {
	for _, t := range w.terrains {
		if t == nil || len(w.meshTreeLayers(t)) == 0 {
			continue
		}
		cw := t.chunkWorld()
		cx, cz := int(math32.Floor(x/cw)), int(math32.Floor(z/cw))
		for dz := -1; dz <= 1; dz++ {
			for dx := -1; dx <= 1; dx++ {
				if !f(w.trunksIn(t, chunkKey{X: cx + dx, Z: cz + dz}, cw)) {
					return
				}
			}
		}
	}
}

func (w *World) trunkCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		// TreePush(x, z, radius): for a script's own movement. Pushes a circle out of the
		// trunks of 3D trees and returns 1 if it moved; TreePushX() and TreePushZ() are
		// where it ends up (x and z unchanged when it did not move).
		"treepush": n(func(a []value.Value) (value.Value, error) {
			x, zz, r := float32(argN(a, 0, 0)), float32(argN(a, 1, 0)), float32(argN(a, 2, 0.4))
			moved := false
			w.trunksNear(x, zz, func(ts []trunk) bool {
				var m bool
				x, zz, m = pushOutOfTrunks(x, zz, r, ts)
				moved = moved || m
				return true
			})
			w.treePushX, w.treePushZ = x, zz
			return value.Num(float64(b01(moved))), nil
		}),
		"treepushx": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.treePushX)), nil }),
		"treepushz": n(func(a []value.Value) (value.Value, error) { return value.Num(float64(w.treePushZ)), nil }),
		// TreeHit(x, z [, radius]): 1 if the point, grown by radius, is inside a 3D tree's
		// trunk. For arrows, and for animals looking where to run.
		"treehit": n(func(a []value.Value) (value.Value, error) {
			x, zz, r := float32(argN(a, 0, 0)), float32(argN(a, 1, 0)), float32(argN(a, 2, 0))
			hit := false
			w.trunksNear(x, zz, func(ts []trunk) bool {
				for _, c := range ts {
					dx, dz, reach := x-c.x, zz-c.z, r+c.r
					if dx*dx+dz*dz < reach*reach {
						hit = true
						return false
					}
				}
				return true
			})
			return value.Num(float64(b01(hit))), nil
		}),
		// SetTreeCollision(on): FPS and TPS characters can't walk through 3D trees. On by default.
		"settreecollision": n(func(a []value.Value) (value.Value, error) {
			w.noTreeCollision = argI(a, 0, 1) == 0
			return z()
		}),
	}
}
