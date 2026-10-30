package runtime

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/g3n/engine/animation"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/loader/gltf"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
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
	layers []animLayer
	bind   []localXF
	bindOK bool
}

// animLayer is an extra clip on top of the base pose.
// additive subtracts the bind pose before blending. Otherwise the layer overrides.
type animLayer struct {
	seq      int
	time     float32
	weight   float32
	additive bool
	speed    float32
	mode     int // 0 off, 1 loop
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
		"setanimlayer": need(func(a []value.Value) (value.Value, error) {
			return w.setAnimLayer(a, false)
		}),
		"setanimadditive": need(func(a []value.Value) (value.Value, error) {
			return w.setAnimLayer(a, true)
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
	st.evalClip(st.seq, st.time)
	if root == nil || st.prev < 0 || st.prev == st.seq || st.blend >= 0.999 {
		st.applyLayers(root)
		return
	}
	cur := snapshotLocal(root)
	st.evalClip(st.prev, st.prevT)
	prev := snapshotLocal(root)
	if len(cur) != len(prev) {
		st.evalClip(st.seq, st.time)
		st.applyLayers(root)
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
	st.applyLayers(root)
}

func (w *World) setAnimLayer(a []value.Value, additive bool) (value.Value, error) {
	e, err := w.ent(argI(a, 0, 0))
	if err != nil {
		return value.Value{}, err
	}
	if e.anim == nil || len(e.anim.clips) == 0 {
		return value.Value{}, fmt.Errorf("SetAnimLayer: no clips")
	}
	st := e.anim
	slot := argI(a, 1, 0)
	if slot < 0 {
		return value.Value{}, fmt.Errorf("SetAnimLayer: slot %d", slot)
	}
	for len(st.layers) <= slot {
		st.layers = append(st.layers, animLayer{seq: -1, speed: 1})
	}
	L := &st.layers[slot]
	L.weight = float32(argN(a, 2, 1))
	if L.weight < 0 {
		L.weight = 0
	}
	if L.weight > 1 {
		L.weight = 1
	}
	L.additive = additive
	L.speed = 1
	L.mode = 0
	if L.weight > 0 {
		L.mode = 1
	}
	if len(a) >= 4 {
		if a[3].Kind == value.KindStr {
			seq, err := st.findName(argS(a, 3))
			if err != nil {
				return value.Value{}, err
			}
			L.seq = seq
		} else {
			L.seq = argI(a, 3, 0)
		}
		L.time = 0
		if L.seq >= 0 && L.seq < len(st.first) {
			L.time = st.first[L.seq]
		}
	}
	if L.seq < 0 || L.seq >= len(st.clips) {
		return value.Value{}, fmt.Errorf("SetAnimLayer: seq %d missing", L.seq)
	}
	if !st.bindOK && e.node != nil {
		st.bind = snapshotLocal(e.node)
		st.bindOK = true
	}
	st.applyPose(e.node)
	w.MarkShadowDirty()
	return value.Num(float64(L.weight)), nil
}

func (st *animState) findName(name string) (int, error) {
	want := strings.ToLower(name)
	for i, n := range st.names {
		if strings.ToLower(n) == want {
			return i, nil
		}
	}
	return 0, fmt.Errorf("SetAnimLayer: no clip %q", name)
}

func (st *animState) applyLayers(root core.INode) {
	if root == nil || len(st.layers) == 0 {
		return
	}
	if !st.bindOK {
		st.bind = snapshotLocal(root)
		st.bindOK = len(st.bind) > 0
	}
	for i := range st.layers {
		L := &st.layers[i]
		if L.weight <= 0 || L.mode == 0 || L.seq < 0 || L.seq >= len(st.clips) {
			continue
		}
		base := snapshotLocal(root)
		st.evalClip(L.seq, L.time)
		layer := snapshotLocal(root)
		writeLocals(base)
		if len(layer) != len(base) {
			continue
		}
		for k := range base {
			n := base[k].n
			if n == nil {
				continue
			}
			if L.additive {
				applyAdditive(n, base[k], bindAt(st.bind, k), layer[k], L.weight)
			} else {
				u := 1 - L.weight
				t := L.weight
				n.SetPosition(base[k].p.X*u+layer[k].p.X*t, base[k].p.Y*u+layer[k].p.Y*t, base[k].p.Z*u+layer[k].p.Z*t)
				n.SetScale(base[k].s.X*u+layer[k].s.X*t, base[k].s.Y*u+layer[k].s.Y*t, base[k].s.Z*u+layer[k].s.Z*t)
				q := nlerpQuat(base[k].q, layer[k].q, t)
				n.SetQuaternion(q.X, q.Y, q.Z, q.W)
			}
		}
	}
}

func bindAt(bind []localXF, i int) localXF {
	if i >= 0 && i < len(bind) {
		return bind[i]
	}
	return localXF{q: math32.Quaternion{W: 1}, s: math32.Vector3{X: 1, Y: 1, Z: 1}}
}

func applyAdditive(n *core.Node, base, bind, layer localXF, w float32) {
	n.SetPosition(
		base.p.X+(layer.p.X-bind.p.X)*w,
		base.p.Y+(layer.p.Y-bind.p.Y)*w,
		base.p.Z+(layer.p.Z-bind.p.Z)*w,
	)
	n.SetScale(
		base.s.X+(layer.s.X-bind.s.X)*w,
		base.s.Y+(layer.s.Y-bind.s.Y)*w,
		base.s.Z+(layer.s.Z-bind.s.Z)*w,
	)
	bindInv := bind.q
	if bindInv.W == 0 && bindInv.X == 0 && bindInv.Y == 0 && bindInv.Z == 0 {
		bindInv.W = 1
	}
	bindInv.Inverse()
	var delta, blended, out math32.Quaternion
	delta.MultiplyQuaternions(&bindInv, &layer.q)
	blended = math32.Quaternion{W: 1}
	blended.Slerp(&delta, w)
	out.MultiplyQuaternions(&base.q, &blended)
	n.SetQuaternion(out.X, out.Y, out.Z, out.W)
}

func writeLocals(pose []localXF) {
	for _, xf := range pose {
		if xf.n == nil {
			continue
		}
		xf.n.SetPosition(xf.p.X, xf.p.Y, xf.p.Z)
		xf.n.SetScale(xf.s.X, xf.s.Y, xf.s.Z)
		xf.n.SetQuaternion(xf.q.X, xf.q.Y, xf.q.Z, xf.q.W)
	}
}

func (st *animState) evalClip(i int, t float32) {
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

// driveLocomotion plays an idle clip when the character is nearly still and a
// walk or run clip while it moves. A one-shot clip that is neither of those
// keeps playing until it finishes. Names come from the glTF file.
func (w *World) driveLocomotion(id int, hx, hz float32) {
	e := w.locomotionEntity(id)
	if e == nil || e.anim == nil {
		return
	}
	st := e.anim
	if len(st.clips) == 0 {
		return
	}
	idle, walk := locomotionClips(st.names)
	if st.mode == 3 && st.seq != idle && st.seq != walk {
		return
	}
	speed := float32(math.Hypot(float64(hx), float64(hz)))
	seq := idle
	rate := float32(1)
	if speed >= 0.35 {
		seq = walk
		rate = speed / 1.6
		if rate < 0.65 {
			rate = 0.65
		}
		if rate > 2.2 {
			rate = 2.2
		}
	} else if idle == walk {
		st.mode = 0
		st.speed = 1
		return
	}
	if st.seq != seq {
		st.seq = seq
		if seq >= 0 && seq < len(st.first) {
			st.time = st.first[seq]
		}
		st.dir = 1
	}
	st.mode = 1
	st.speed = rate
}

func (w *World) locomotionEntity(id int) *Entity {
	if e := w.ents[id]; e != nil && e.anim != nil && len(e.anim.clips) > 0 {
		return e
	}
	for _, e := range w.ents {
		if e != nil && e.parent == id && e.anim != nil && len(e.anim.clips) > 0 {
			return e
		}
	}
	return nil
}

func locomotionClips(names []string) (idle, walk int) {
	idle = clipNamed(names, "idle", "stand", "breath")
	walk = clipNamed(names, "walk", "run", "jog", "locom")
	switch {
	case idle < 0 && walk < 0:
		idle, walk = 0, 0
		if len(names) > 1 {
			walk = 1
		}
	case idle < 0:
		idle = walk
	case walk < 0:
		walk = idle
	}
	return idle, walk
}

func clipNamed(names []string, keys ...string) int {
	for i, n := range names {
		ln := strings.ToLower(n)
		for _, k := range keys {
			if strings.Contains(ln, k) {
				return i
			}
		}
	}
	return -1
}

func (w *World) tickAnims(dt float32) {
	posed := false
	for _, e := range w.ents {
		st := e.anim
		if st == nil || len(st.clips) == 0 {
			continue
		}
		layerOn := false
		for i := range st.layers {
			L := &st.layers[i]
			if L.mode == 0 || L.weight <= 0 || L.seq < 0 || L.seq >= len(st.clips) {
				continue
			}
			layerOn = true
			t0, t1 := st.first[L.seq], st.last[L.seq]
			if t1 <= t0 {
				t1 = t0 + st.length[L.seq]
			}
			L.time += dt * L.speed
			span := t1 - t0
			if span > 0 {
				for L.time > t1 {
					L.time -= span
				}
				for L.time < t0 {
					L.time += span
				}
			}
		}
		if st.mode == 0 && !layerOn {
			continue
		}
		if st.mode == 0 {
			st.applyPose(e.node)
			posed = true
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
	for _, e := range w.ents {
		if e != nil && e.ik != nil {
			w.solveIK(e)
			posed = true
		}
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
	if w.ents[id].node != nil {
		st.bind = snapshotLocal(w.ents[id].node)
		st.bindOK = len(st.bind) > 0
	}
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
