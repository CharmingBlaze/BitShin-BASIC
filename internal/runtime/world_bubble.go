package runtime

import (
	"math"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/value"
)

// worldBubble is the open-world window around the player.
// SetPlayer turns it on. Expert commands change one piece at a time.
type worldBubble struct {
	on           bool
	userOff      bool
	player       int
	followLocked bool
	simRadius    int
	shift        bool
	shiftAt      float32
	originX      float32
	originZ      float32
	morphOff     bool
}

func (w *World) ensureBubble() *worldBubble {
	if w.bubble == nil {
		w.bubble = &worldBubble{simRadius: 1, shift: true, shiftAt: 4096}
	}
	return w.bubble
}

func (w *World) worldOrigin() (float32, float32) {
	if w == nil || w.bubble == nil {
		return 0, 0
	}
	return w.bubble.originX, w.bubble.originZ
}

// armWorldBubble makes terrain, the prop stream, and water follow player.
// The prop stream does not grow the demo filler when this call creates it.
func (w *World) armWorldBubble(player int) {
	if w.bubble != nil && w.bubble.userOff {
		return
	}
	b := w.ensureBubble()
	b.on = true
	b.player = player
	b.followLocked = false
	if b.shiftAt == 0 {
		b.shiftAt = 4096
	}
	created := w.stream == nil
	s := w.ensureStream()
	if created {
		s.fill = false
	}
	s.follow = player
	w.applyBubbleWater()
}

func (w *World) applyBubbleWater() {
	b := w.bubble
	if b == nil || !b.on {
		return
	}
	for _, wb := range w.waters {
		if wb != nil && !wb.followSet {
			wb.follow = true
		}
	}
}

func (w *World) tickBubble() {
	b := w.bubble
	if b == nil || !b.on {
		return
	}
	w.applyBubbleWater()
	id := b.player
	if id == 0 && w.play != nil {
		id = w.play.player
	}
	if id == 0 {
		return
	}
	x, _, z, ok := w.scriptPos(id)
	if !ok || !b.shift {
		return
	}
	lim := b.shiftAt
	if lim < 256 {
		lim = 4096
	}
	if x > lim || x < -lim || z > lim || z < -lim {
		w.shiftScene(x, z)
	}
}

func (w *World) shiftQuantum() float32 {
	if t := w.terrains[w.curTerrain]; t != nil {
		if cw := t.chunkWorld(); cw >= 4 {
			return cw
		}
	}
	if w.stream != nil && w.stream.size >= 4 {
		return w.stream.size
	}
	return 32
}

func sceneLevel(w *World, n *core.Node) bool {
	if n == nil {
		return false
	}
	p := n.Parent()
	if p == nil || w.scene == nil {
		return p == nil || w.scene == nil
	}
	return p.GetNode() == w.scene
}

func (w *World) shiftScene(dx, dz float32) {
	q := w.shiftQuantum()
	qx := float32(math.Trunc(float64(dx/q))) * q
	qz := float32(math.Trunc(float64(dz/q))) * q
	if qx == 0 && qz == 0 {
		return
	}
	b := w.ensureBubble()
	b.originX += qx
	b.originZ += qz
	seen := map[*core.Node]bool{}
	shiftNode := func(n *core.Node) {
		if n == nil || seen[n] || !sceneLevel(w, n) {
			return
		}
		seen[n] = true
		p := n.Position()
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		gx, gy, gz := toG3N(x-qx, y, z-qz)
		n.SetPosition(gx, gy, gz)
	}
	for id, e := range w.ents {
		if e == nil || e.node == nil {
			continue
		}
		n := e.node.GetNode()
		if n == nil || seen[n] || !sceneLevel(w, n) {
			continue
		}
		p := n.Position()
		x, y, z := fromG3N(p.X, p.Y, p.Z)
		shiftNode(n)
		if w.phys3 != nil && e.bodyType != 0 {
			w.phys3.SetPosition(id, x-qx, y, z-qz)
		}
	}
	if w.cam != nil {
		shiftNode(w.cam.GetNode())
	}
	w.rekeyTerrain(qx, qz)
	w.rekeyStream(qx, qz)
	if w.stream != nil {
		w.stream.ox -= qx
		w.stream.oz -= qz
	}
}

func rekey[V any](in map[chunkKey]V, dkx, dkz int) map[chunkKey]V {
	if len(in) == 0 || (dkx == 0 && dkz == 0) {
		return in
	}
	out := make(map[chunkKey]V, len(in))
	for k, v := range in {
		out[chunkKey{k.X - dkx, k.Z - dkz}] = v
	}
	return out
}

func (w *World) rekeyTerrain(qx, qz float32) {
	for _, t := range w.terrains {
		if t == nil {
			continue
		}
		cw := t.chunkWorld()
		if cw < 1 {
			continue
		}
		dkx := int(math.Round(float64(qx / cw)))
		dkz := int(math.Round(float64(qz / cw)))
		t.ents = rekey(t.ents, dkx, dkz)
		t.built = rekey(t.built, dkx, dkz)
		t.scatterIDs = rekey(t.scatterIDs, dkx, dkz)
		t.pending = rekey(t.pending, dkx, dkz)
		if w.phys3 != nil {
			for _, id := range t.ents {
				w.phys3.Remove(id)
			}
		}
		t.simOn = map[chunkKey]bool{}
	}
}

func (w *World) rekeyStream(qx, qz float32) {
	s := w.stream
	if s == nil || len(s.chunks) == 0 {
		return
	}
	next := make(map[chunkKey]*streamChunk, len(s.chunks))
	for k, ch := range s.chunks {
		if ch == nil {
			continue
		}
		ox := (float32(k.X)+0.5)*s.size - qx
		oz := (float32(k.Z)+0.5)*s.size - qz
		nk := chunkOf(ox, oz, s.size)
		ch.key = nk
		next[nk] = ch
	}
	s.chunks = next
}

func (w *World) chunkInSim(k chunkKey, cw float32) bool {
	b := w.bubble
	if b == nil || !b.on || b.simRadius < 0 || cw <= 0 {
		return false
	}
	ox, oz := float32(0), float32(0)
	if w.stream != nil {
		ox, oz = w.stream.ox, w.stream.oz
	}
	c := chunkOf(ox, oz, cw)
	dx := k.X - c.X
	if dx < 0 {
		dx = -dx
	}
	dz := k.Z - c.Z
	if dz < 0 {
		dz = -dz
	}
	return dx <= b.simRadius && dz <= b.simRadius
}

func (w *World) releaseTerrainChunk(t *terrain, k chunkKey, id int) {
	if t == nil {
		return
	}
	if w.phys3 != nil && id != 0 {
		w.phys3.Remove(id)
	}
	if id != 0 {
		w.freeEntityID(id)
	}
	w.freeScatter(t, k)
	delete(t.ents, k)
	delete(t.built, k)
	delete(t.simOn, k)
}

func (w *World) syncChunkSim(t *terrain, k chunkKey, id int, x0, z0, size float32) {
	if t == nil || id == 0 {
		return
	}
	if t.simOn == nil {
		t.simOn = map[chunkKey]bool{}
	}
	want := w.chunkInSim(k, size)
	if on, ok := t.simOn[k]; ok && on == want {
		return
	}
	t.simOn[k] = want
	if !want {
		if w.phys3 != nil {
			w.phys3.Remove(id)
		}
		return
	}
	w.ensurePhys3()
	const n = 9
	samples := make([]float32, n*n)
	den := float32(n - 1)
	for z := 0; z < n; z++ {
		for x := 0; x < n; x++ {
			samples[z*n+x] = w.terrainHeight(x0+float32(x)/den*size, z0+float32(z)/den*size)
		}
	}
	w.phys3.Remove(id)
	w.phys3.AddHeightField(id, samples, n, x0, 0, z0, size/den, 1, size/den)
}

// coarseY pulls a height toward the next coarser lattice so a LOD swap does not pop.
func coarseY(raw func(x, z float32) float32, x, z, x0, z0, x1, z1 float32, full, step int, morph float32) float32 {
	y := raw(x, z)
	if raw == nil || morph <= 0 || step >= 4 || full < 2 {
		return y
	}
	next := step * 2
	if next > full {
		next = full
	}
	if next < 1 {
		return y
	}
	spanX := x1 - x0
	spanZ := z1 - z0
	if spanX < 0.001 {
		spanX = 0.001
	}
	if spanZ < 0.001 {
		spanZ = 0.001
	}
	fx := (x - x0) / spanX * float32(full)
	fz := (z - z0) / spanZ * float32(full)
	i0 := int(math.Floor(float64(fx)/float64(next)+1e-4)) * next
	j0 := int(math.Floor(float64(fz)/float64(next)+1e-4)) * next
	if i0 < 0 {
		i0 = 0
	}
	if j0 < 0 {
		j0 = 0
	}
	i1 := i0 + next
	j1 := j0 + next
	if i1 > full {
		i1 = full
	}
	if j1 > full {
		j1 = full
	}
	tx, tz := float32(0), float32(0)
	if i1 != i0 {
		tx = (fx - float32(i0)) / float32(i1-i0)
	}
	if j1 != j0 {
		tz = (fz - float32(j0)) / float32(j1-j0)
	}
	if tx < 0 {
		tx = 0
	} else if tx > 1 {
		tx = 1
	}
	if tz < 0 {
		tz = 0
	} else if tz > 1 {
		tz = 1
	}
	at := func(i, j int) float32 {
		return raw(x0+float32(i)/float32(full)*spanX, z0+float32(j)/float32(full)*spanZ)
	}
	c := (at(i0, j0)*(1-tx)+at(i1, j0)*tx)*(1-tz) + (at(i0, j1)*(1-tx)+at(i1, j1)*tx)*tz
	return y*(1-morph) + c*morph
}

func (w *World) bubbleCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"setworldbubble": n(func(a []value.Value) (value.Value, error) {
			on := argI(a, 0, 1) != 0
			b := w.ensureBubble()
			if !on {
				b.userOff = true
				b.on = false
				if w.stream != nil && !b.followLocked && w.stream.follow == b.player {
					w.stream.follow = 0
				}
				return value.Num(0), nil
			}
			b.userOff = false
			id := argI(a, 1, 0)
			if id == 0 && w.play != nil {
				id = w.play.player
			}
			w.armWorldBubble(id)
			return value.Num(1), nil
		}),
		"setworldsimradius": n(func(a []value.Value) (value.Value, error) {
			b := w.ensureBubble()
			b.simRadius = argI(a, 0, 1)
			if b.simRadius < -1 {
				b.simRadius = -1
			}
			if b.on {
				for _, t := range w.terrains {
					if t != nil {
						t.simOn = nil
					}
				}
			}
			return value.Num(float64(b.simRadius)), nil
		}),
		"getworldsimradius": n(func(a []value.Value) (value.Value, error) {
			if w.bubble == nil {
				return value.Num(1), nil
			}
			return value.Num(float64(w.bubble.simRadius)), nil
		}),
		"setworldshift": n(func(a []value.Value) (value.Value, error) {
			b := w.ensureBubble()
			b.shift = argI(a, 0, 1) != 0
			if len(a) > 1 {
				b.shiftAt = float32(argN(a, 1, 4096))
			}
			if b.shiftAt < 256 {
				b.shiftAt = 256
			}
			return z()
		}),
		"setworldmorph": n(func(a []value.Value) (value.Value, error) {
			on := argI(a, 0, 1) != 0
			b := w.ensureBubble()
			b.morphOff = !on
			for _, t := range w.terrains {
				if t != nil {
					t.morphOff = b.morphOff
					t.built = nil
				}
			}
			return z()
		}),
		"worldoriginx": n(func(a []value.Value) (value.Value, error) {
			x, _ := w.worldOrigin()
			return value.Num(float64(x)), nil
		}),
		"worldoriginz": n(func(a []value.Value) (value.Value, error) {
			_, z := w.worldOrigin()
			return value.Num(float64(z)), nil
		}),
		"getworldoriginx": n(func(a []value.Value) (value.Value, error) {
			x, _ := w.worldOrigin()
			return value.Num(float64(x)), nil
		}),
		"getworldoriginz": n(func(a []value.Value) (value.Value, error) {
			_, z := w.worldOrigin()
			return value.Num(float64(z)), nil
		}),
	}
}
