package runtime

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/g3n/engine/animation"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/loader/gltf"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/value"
)

type animState struct {
	clips  []*animation.Animation
	names  []string
	length []float32
	first  []float32
	last   []float32
	seq    int
	prev   int
	mode   int // 0 stop 1 loop 2 pingpong 3 once
	speed  float32
	time   float32
	prevT  float32
	dir    float32
	blend  float32 // 0 = previous clip, 1 = current clip
}

func (w *World) animCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"loadanimmesh": need(func(a []value.Value) (value.Value, error) {
			id, err := w.loadAnimMesh(argS(a, 0), argI(a, 1, 0))
			return value.Num(float64(id)), err
		}),
		"animate": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil || len(e.anim.clips) == 0 {
				return value.Value{}, fmt.Errorf("Animate: entity has no clips (LoadAnimMesh a .gltf/.glb with animations)")
			}
			st := e.anim
			st.mode = argI(a, 1, 1)
			st.speed = float32(argN(a, 2, 1))
			if st.speed == 0 {
				st.speed = 1
			}
			if len(a) >= 4 {
				st.seq = argI(a, 3, 0)
			}
			if st.seq < 0 || st.seq >= len(st.clips) {
				return value.Value{}, fmt.Errorf("Animate: seq %d out of range (0..%d)", st.seq, len(st.clips)-1)
			}
			st.dir = 1
			st.time = st.first[st.seq]
			st.clips[st.seq].Reset()
			st.clips[st.seq].SetPaused(st.mode == 0)
			return z()
		}),
		"setanimtime": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil || len(e.anim.clips) == 0 {
				return value.Value{}, fmt.Errorf("SetAnimTime: no clips")
			}
			e.anim.time = float32(argN(a, 1, 0))
			e.anim.applyPose(e.node)
			w.MarkShadowDirty()
			return z()
		}),
		"animtime": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(e.anim.time)), nil
		}),
		"animlength": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil || len(e.anim.clips) == 0 {
				return value.Num(0), nil
			}
			return value.Num(float64(e.anim.span())), nil
		}),
		"extractanimseq": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil || len(e.anim.clips) == 0 {
				return value.Value{}, fmt.Errorf("ExtractAnimSeq: no clips")
			}
			src := argI(a, 3, e.anim.seq)
			if src < 0 || src >= len(e.anim.clips) {
				return value.Value{}, fmt.Errorf("ExtractAnimSeq: seq %d missing", src)
			}
			st := e.anim
			st.clips = append(st.clips, st.clips[src])
			st.names = append(st.names, fmt.Sprintf("%s_%d_%d", st.names[src], argI(a, 1, 0), argI(a, 2, 0)))
			st.length = append(st.length, st.length[src])
			st.first = append(st.first, float32(argN(a, 1, 0)))
			st.last = append(st.last, float32(argN(a, 2, float64(st.length[src]))))
			return value.Num(float64(len(st.clips) - 1)), nil
		}),
		"setanimseq": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil {
				return value.Value{}, fmt.Errorf("SetAnimSeq: no clips")
			}
			if len(a) >= 2 && a[1].Kind == value.KindStr {
				return value.Num(0), e.anim.setName(a[1].String())
			}
			e.anim.seq = argI(a, 1, 0)
			if e.anim.seq < 0 || e.anim.seq >= len(e.anim.clips) {
				return value.Value{}, fmt.Errorf("SetAnimSeq: %d out of range", e.anim.seq)
			}
			e.anim.time = e.anim.first[e.anim.seq]
			return z()
		}),
		"stopanim": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil {
				return z()
			}
			e.anim.mode = 0
			if e.anim.seq >= 0 && e.anim.seq < len(e.anim.clips) {
				e.anim.clips[e.anim.seq].SetPaused(true)
			}
			return z()
		}),
		"animplaying": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim != nil && e.anim.mode != 0 {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"setanimblend": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil {
				return value.Value{}, fmt.Errorf("SetAnimBlend: no clips")
			}
			st := e.anim
			st.blend = float32(argN(a, 1, 1))
			if st.blend < 0 {
				st.blend = 0
			}
			if st.blend > 1 {
				st.blend = 1
			}
			if len(a) >= 3 {
				old := st.seq
				if a[2].Kind == value.KindStr {
					_ = st.setName(argS(a, 2))
				} else {
					st.seq = argI(a, 2, st.seq)
				}
				if st.seq != old && old >= 0 && old < len(st.clips) {
					st.prev = old
					st.prevT = st.time
					st.time = st.first[st.seq]
				}
			}
			st.applyPose(e.node)
			w.MarkShadowDirty()
			return value.Num(float64(st.blend)), nil
		}),
		"animseqname": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if e.anim == nil || len(e.anim.clips) == 0 {
				return value.Str(""), nil
			}
			if len(a) >= 2 {
				if err := e.anim.setName(argS(a, 1)); err != nil {
					return value.Value{}, err
				}
			}
			return value.Str(e.anim.names[e.anim.seq]), nil
		}),
	}
}

func (st *animState) setName(name string) error {
	want := strings.ToLower(name)
	for i, n := range st.names {
		if strings.ToLower(n) == want {
			st.seq = i
			st.time = st.first[i]
			return nil
		}
	}
	return fmt.Errorf("AnimSeqName: no clip %q", name)
}

func (st *animState) span() float32 {
	if st.seq < 0 || st.seq >= len(st.last) {
		return 0
	}
	return st.last[st.seq] - st.first[st.seq]
}

func (st *animState) applyPose(root core.INode) {
	if st.seq < 0 || st.seq >= len(st.clips) {
		return
	}
	eval := func(i int, t float32) {
		if i < 0 || i >= len(st.clips) {
			return
		}
		cl := st.clips[i]
		cl.SetSpeed(1)
		cl.SetLoop(false)
		cl.SetPaused(false)
		cl.Reset()
		cl.Update(t)
	}
	eval(st.seq, st.time)
	if root == nil || st.prev < 0 || st.prev == st.seq || st.blend >= 0.999 {
		return
	}
	cur := snapshotLocal(root)
	eval(st.prev, st.prevT)
	prev := snapshotLocal(root)
	if len(cur) != len(prev) {
		eval(st.seq, st.time)
		return
	}
	t := st.blend
	u := 1 - t
	for i := range cur {
		n := cur[i].n
		if n == nil {
			continue
		}
		n.SetPosition(prev[i].p.X*u+cur[i].p.X*t, prev[i].p.Y*u+cur[i].p.Y*t, prev[i].p.Z*u+cur[i].p.Z*t)
		n.SetScale(prev[i].s.X*u+cur[i].s.X*t, prev[i].s.Y*u+cur[i].s.Y*t, prev[i].s.Z*u+cur[i].s.Z*t)
		q := nlerpQuat(prev[i].q, cur[i].q, t)
		n.SetQuaternion(q.X, q.Y, q.Z, q.W)
	}
}

type localXF struct {
	n *core.Node
	p math32.Vector3
	s math32.Vector3
	q math32.Quaternion
}

func snapshotLocal(n core.INode) []localXF {
	var out []localXF
	walkLocal(n, &out)
	return out
}

func walkLocal(n core.INode, out *[]localXF) {
	if n == nil {
		return
	}
	node := n.GetNode()
	p := node.Position()
	s := node.Scale()
	q := node.Quaternion()
	*out = append(*out, localXF{n: node, p: p, s: s, q: q})
	for _, c := range node.Children() {
		walkLocal(c, out)
	}
}

func nlerpQuat(a, b math32.Quaternion, t float32) math32.Quaternion {
	if a.X*b.X+a.Y*b.Y+a.Z*b.Z+a.W*b.W < 0 {
		b.X, b.Y, b.Z, b.W = -b.X, -b.Y, -b.Z, -b.W
	}
	q := math32.Quaternion{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
		W: a.W + (b.W-a.W)*t,
	}
	q.Normalize()
	return q
}

func (w *World) tickAnims(dt float32) {
	posed := false
	for _, e := range w.ents {
		st := e.anim
		if st == nil || len(st.clips) == 0 || st.mode == 0 {
			continue
		}
		t0, t1 := st.first[st.seq], st.last[st.seq]
		if t1 <= t0 {
			t1 = t0 + st.length[st.seq]
		}
		st.time += dt * st.speed * st.dir
		switch st.mode {
		case 1:
			span := t1 - t0
			if span > 0 {
				for st.time > t1 {
					st.time -= span
				}
				for st.time < t0 {
					st.time += span
				}
			}
		case 2:
			if st.time >= t1 {
				st.time = t1
				st.dir = -1
			}
			if st.time <= t0 {
				st.time = t0
				st.dir = 1
			}
		case 3:
			if st.time >= t1 {
				st.time = t1
				st.mode = 0
			}
		}
		st.applyPose(e.node)
		posed = true
	}
	w.tickBoneAttaches()
	if posed {
		w.MarkShadowDirty()
	}
}

func (w *World) loadAnimMesh(file string, parent int) (int, error) {
	path, err := w.openPath(file)
	if err != nil {
		return 0, fmt.Errorf("LoadAnimMesh: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".gltf" && ext != ".glb" {
		return 0, fmt.Errorf("LoadAnimMesh: need a .gltf or .glb with clips (got %s)", ext)
	}
	id, err := w.loadGLTF(file, parent, ext == ".glb")
	if err != nil {
		return 0, err
	}
	g, err := parseGLTF(path, ext == ".glb")
	if err != nil {
		return 0, err
	}
	st := &animState{dir: 1, speed: 1, prev: -1, blend: 1}
	for i := range g.Animations {
		clip, err := g.LoadAnimation(i)
		if err != nil {
			return 0, fmt.Errorf("LoadAnimMesh: clip %d: %w", i, err)
		}
		name := g.Animations[i].Name
		if name == "" {
			name = fmt.Sprintf("seq%d", i)
		}
		ln := gltfClipLength(g, i)
		st.clips = append(st.clips, clip)
		st.names = append(st.names, name)
		st.length = append(st.length, ln)
		st.first = append(st.first, 0)
		st.last = append(st.last, ln)
	}
	w.ents[id].anim = st
	if len(st.clips) == 0 {
		return 0, fmt.Errorf("LoadAnimMesh: %s has no animation clips", file)
	}
	return id, nil
}

func parseGLTF(path string, bin bool) (*gltf.GLTF, error) {
	if bin {
		return gltf.ParseBin(path)
	}
	return gltf.ParseJSON(path)
}

func gltfClipLength(g *gltf.GLTF, i int) float32 {
	var mx float32
	for _, s := range g.Animations[i].Samplers {
		if s.Input >= 0 && s.Input < len(g.Accessors) {
			acc := g.Accessors[s.Input]
			if len(acc.Max) > 0 && acc.Max[0] > mx {
				mx = acc.Max[0]
			}
		}
	}
	if mx <= 0 {
		mx = 1
	}
	return mx
}
