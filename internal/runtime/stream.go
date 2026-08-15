package runtime

import (
	"math"

	"bitshinbasic/internal/value"
)

type chunkKey struct{ X, Z int }

type streamChunk struct {
	key     chunkKey
	ents    []int
	loaded  bool
	pending bool
}

type worldStream struct {
	size    float32
	radius  int
	ox, oz  float32
	follow  int
	chunks  map[chunkKey]*streamChunk
	fill    bool // spawn a few props per chunk (demo)
}

func (w *World) ensureStream() *worldStream {
	if w.stream == nil {
		w.stream = &worldStream{
			size:   24,
			radius: 2,
			chunks: map[chunkKey]*streamChunk{},
			fill:   true,
		}
	}
	return w.stream
}

func chunkOf(x, z, size float32) chunkKey {
	if size <= 0 {
		size = 24
	}
	return chunkKey{
		X: int(math.Floor(float64(x / size))),
		Z: int(math.Floor(float64(z / size))),
	}
}

func (s *worldStream) needed(ox, oz float32) map[chunkKey]bool {
	c := chunkOf(ox, oz, s.size)
	r := s.radius
	if r < 0 {
		r = 0
	}
	out := map[chunkKey]bool{}
	for z := c.Z - r; z <= c.Z+r; z++ {
		for x := c.X - r; x <= c.X+r; x++ {
			out[chunkKey{x, z}] = true
		}
	}
	return out
}

func (w *World) tickStream() {
	s := w.stream
	if s == nil {
		return
	}
	if s.follow != 0 {
		if e := w.ents[s.follow]; e != nil && e.node != nil {
			p := worldPos(e.node.GetNode())
			x, _, z := fromG3N(p.X, p.Y, p.Z)
			s.ox, s.oz = x, z
		}
	}
	need := s.needed(s.ox, s.oz)
	for k, ch := range s.chunks {
		if !need[k] && ch.loaded && !ch.pending {
			w.unloadStreamChunk(ch)
		}
	}
	for k := range need {
		ch := s.chunks[k]
		if ch != nil && (ch.loaded || ch.pending) {
			continue
		}
		if ch == nil {
			ch = &streamChunk{key: k}
			s.chunks[k] = ch
		}
		ch.pending = true
		key := k
		w.ensureJobs().submit(func() {
			plan := buildChunkPlan(key, s.size)
			w.ensureJobs().enqueueGL(func() {
				w.applyChunkPlan(key, plan)
			})
		})
	}
}

type chunkItem struct {
	kind    int // 0 cube, 1 cone tree
	x, y, z float32
	sx, sy, sz float32
	r, g, b float64
}

func buildChunkPlan(k chunkKey, size float32) []chunkItem {
	h := uint32(k.X*73856093 ^ k.Z*19349663)
	n := int(1 + h%3)
	cx := (float32(k.X) + 0.5) * size
	cz := (float32(k.Z) + 0.5) * size
	out := make([]chunkItem, 0, n+1)
	out = append(out, chunkItem{
		kind: 0, x: cx, y: 0.15, z: cz,
		sx: size * 0.92, sy: 0.3, sz: size * 0.92,
		r: 40 + float64(h%30), g: 70 + float64((h>>3)%40), b: 50,
	})
	for i := 0; i < n; i++ {
		h = h*1664525 + 1013904223
		ox := (float32(h%1000)/1000 - 0.5) * size * 0.7
		h = h*1664525 + 1013904223
		oz := (float32(h%1000)/1000 - 0.5) * size * 0.7
		out = append(out, chunkItem{
			kind: 1, x: cx + ox, y: 1.2, z: cz + oz,
			sx: 0.45, sy: 2.2, sz: 0.45,
			r: 30, g: 110 + float64(h%80), b: 40,
		})
	}
	return out
}

func (w *World) applyChunkPlan(k chunkKey, plan []chunkItem) {
	s := w.stream
	if s == nil {
		return
	}
	ch := s.chunks[k]
	if ch == nil {
		return
	}
	if !s.needed(s.ox, s.oz)[k] {
		ch.pending = false
		return
	}
	if ch.loaded {
		ch.pending = false
		return
	}
	for _, it := range plan {
		var id int
		if it.kind == 1 {
			id = w.createConeMesh(nil)
		} else {
			id = w.createCubeMesh(nil)
		}
		if e := w.ents[id]; e != nil {
			gx, gy, gz := toG3N(it.x, it.y, it.z)
			e.node.GetNode().SetPosition(gx, gy, gz)
			e.node.GetNode().SetScale(it.sx, it.sy, it.sz)
			if e.mat != nil {
				e.mat.SetColor(rgb(it.r, it.g, it.b))
			}
			e.name = "stream"
		}
		ch.ents = append(ch.ents, id)
	}
	ch.loaded = true
	ch.pending = false
}

func (w *World) unloadStreamChunk(ch *streamChunk) {
	for _, id := range ch.ents {
		w.freeEntityID(id)
	}
	ch.ents = nil
	ch.loaded = false
	ch.pending = false
}

func (w *World) streamChunkCount() int {
	if w.stream == nil {
		return 0
	}
	n := 0
	for _, ch := range w.stream.chunks {
		if ch.loaded {
			n++
		}
	}
	return n
}

func (w *World) streamCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createworldstream": n(func(a []value.Value) (value.Value, error) {
			s := w.ensureStream()
			if len(a) > 0 {
				s.size = float32(argN(a, 0, 24))
			}
			if len(a) > 1 {
				s.radius = argI(a, 1, 2)
			}
			if s.size < 4 {
				s.size = 4
			}
			return value.Num(1), nil
		}),
		"setstreamradius": n(func(a []value.Value) (value.Value, error) {
			s := w.ensureStream()
			s.radius = argI(a, 0, 2)
			if s.radius < 0 {
				s.radius = 0
			}
			return value.Num(float64(s.radius)), nil
		}),
		"setstreamorigin": n(func(a []value.Value) (value.Value, error) {
			s := w.ensureStream()
			s.ox = float32(argN(a, 0, 0))
			s.oz = float32(argN(a, 2, argN(a, 1, 0)))
			if len(a) >= 3 {
				s.oz = float32(argN(a, 2, 0))
			}
			return z()
		}),
		"setstreamfollow": n(func(a []value.Value) (value.Value, error) {
			s := w.ensureStream()
			s.follow = argI(a, 0, 0)
			return z()
		}),
		"loadchunk": n(func(a []value.Value) (value.Value, error) {
			s := w.ensureStream()
			k := chunkKey{argI(a, 0, 0), argI(a, 1, 0)}
			if s.chunks[k] == nil {
				s.chunks[k] = &streamChunk{key: k}
			}
			if !s.chunks[k].loaded && !s.chunks[k].pending {
				s.chunks[k].pending = true
				plan := buildChunkPlan(k, s.size)
				w.applyChunkPlan(k, plan)
			}
			return z()
		}),
		"unloadchunk": n(func(a []value.Value) (value.Value, error) {
			s := w.ensureStream()
			k := chunkKey{argI(a, 0, 0), argI(a, 1, 0)}
			if ch := s.chunks[k]; ch != nil {
				w.unloadStreamChunk(ch)
			}
			return z()
		}),
		"chunkloaded": n(func(a []value.Value) (value.Value, error) {
			s := w.ensureStream()
			k := chunkKey{argI(a, 0, 0), argI(a, 1, 0)}
			if ch := s.chunks[k]; ch != nil && ch.loaded {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"streamchunkcount": n(func(a []value.Value) (value.Value, error) {
			return value.Num(float64(w.streamChunkCount())), nil
		}),
		"streamoriginx": n(func(a []value.Value) (value.Value, error) {
			if w.stream == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.stream.ox)), nil
		}),
		"streamoriginz": n(func(a []value.Value) (value.Value, error) {
			if w.stream == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(w.stream.oz)), nil
		}),
	}
}

func (w *World) visibleMeshCount() int {
	n := 0
	for _, e := range w.ents {
		if e == nil || e.mesh == nil || e.node == nil {
			continue
		}
		if e.node.GetNode().Visible() {
			n++
		}
	}
	return n
}
