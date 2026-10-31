package runtime

import (
	"fmt"
	"sort"

	"github.com/g3n/engine/core"

	"moonbasic/internal/value"
)

// Blend trees mix several clips of one animated entity by a parameter: 1D
// thresholds (idle, walk, run by speed) or 2D points (strafe by x/y velocity).
// Crossfades ease from whatever is playing into a clip or tree over a set
// time. Both feed the same pose mixer as SetAnimBlend; layers apply on top.

type blendClip struct {
	seq  int
	x, y float32
	time float32 // clip-local time when the tree is not phase-synced
}

type blendTree struct {
	id      int
	ent     int
	kind    int // 1 = 1D thresholds, 2 = 2D gradient band
	clips   []blendClip
	px, py  float32
	speed   float32
	sync    bool
	phase   float32 // normalized 0..1 cycle shared by every clip when sync
	weights []float32
}

// animSource is what a crossfade fades out of: a clip, a tree, or a frozen
// pose (a fade that starts while another fade is still running).
type animSource struct {
	tree   *blendTree
	seq    int
	time   float32
	speed  float32
	loop   bool
	frozen []localXF
}

type animFade struct {
	from    animSource
	dur     float32
	elapsed float32
}

func (w *World) blendTreeCommands(need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createblendtree": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			e, err := w.ent(id)
			if err != nil {
				return value.Value{}, fmt.Errorf("CreateBlendTree: %w", err)
			}
			if e.anim == nil || len(e.anim.clips) == 0 {
				return value.Value{}, fmt.Errorf("CreateBlendTree: entity %d has no clips (LoadAnimMesh)", id)
			}
			kind := argI(a, 1, 1)
			if kind != 1 && kind != 2 {
				return value.Value{}, fmt.Errorf("CreateBlendTree: kind %d (1 = 1D, 2 = 2D)", kind)
			}
			if w.blendTrees == nil {
				w.blendTrees = map[int]*blendTree{}
			}
			w.nextBlendTree++
			t := &blendTree{id: w.nextBlendTree, ent: id, kind: kind, speed: 1, sync: true}
			w.blendTrees[t.id] = t
			return value.Num(float64(t.id)), nil
		}),
		"addblendclip": need(func(a []value.Value) (value.Value, error) {
			t, st, err := w.blendTreeAnim(a, "AddBlendClip")
			if err != nil {
				return value.Value{}, err
			}
			seq := argI(a, 1, 0)
			if len(a) >= 2 && a[1].Kind == value.KindStr {
				if seq, err = st.findName(argS(a, 1)); err != nil {
					return value.Value{}, fmt.Errorf("AddBlendClip: no clip %q", argS(a, 1))
				}
			}
			if seq < 0 || seq >= len(st.clips) {
				return value.Value{}, fmt.Errorf("AddBlendClip: seq %d out of range (0..%d)", seq, len(st.clips)-1)
			}
			t.clips = append(t.clips, blendClip{seq: seq, x: float32(argN(a, 2, 0)), y: float32(argN(a, 3, 0)), time: st.first[seq]})
			t.computeWeights()
			return value.Num(float64(len(t.clips) - 1)), nil
		}),
		"setblendparam": need(func(a []value.Value) (value.Value, error) {
			t, err := w.blendTree(argI(a, 0, 0), "SetBlendParam")
			if err != nil {
				return value.Value{}, err
			}
			t.px = float32(argN(a, 1, 0))
			if len(a) >= 3 {
				t.py = float32(argN(a, 2, 0))
			}
			t.computeWeights()
			return value.Num(0), nil
		}),
		"blendparam": need(func(a []value.Value) (value.Value, error) {
			t, err := w.blendTree(argI(a, 0, 0), "BlendParam")
			if err != nil {
				return value.Value{}, err
			}
			if argI(a, 1, 0) == 1 {
				return value.Num(float64(t.py)), nil
			}
			return value.Num(float64(t.px)), nil
		}),
		"blendweight": need(func(a []value.Value) (value.Value, error) {
			t, err := w.blendTree(argI(a, 0, 0), "BlendWeight")
			if err != nil {
				return value.Value{}, err
			}
			i := argI(a, 1, 0)
			if i < 0 || i >= len(t.weights) {
				return value.Num(0), nil
			}
			return value.Num(float64(t.weights[i])), nil
		}),
		"blendclipcount": need(func(a []value.Value) (value.Value, error) {
			t, err := w.blendTree(argI(a, 0, 0), "BlendClipCount")
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(len(t.clips))), nil
		}),
		"setblendtreespeed": need(func(a []value.Value) (value.Value, error) {
			t, err := w.blendTree(argI(a, 0, 0), "SetBlendTreeSpeed")
			if err != nil {
				return value.Value{}, err
			}
			t.speed = float32(argN(a, 1, 1))
			if len(a) >= 3 {
				t.sync = argI(a, 2, 1) != 0
			}
			return value.Num(0), nil
		}),
		"blendtreephase": need(func(a []value.Value) (value.Value, error) {
			t, err := w.blendTree(argI(a, 0, 0), "BlendTreePhase")
			if err != nil {
				return value.Value{}, err
			}
			return value.Num(float64(t.phase)), nil
		}),
		"playblendtree": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, fmt.Errorf("PlayBlendTree: %w", err)
			}
			t, err := w.blendTree(argI(a, 1, 0), "PlayBlendTree")
			if err != nil {
				return value.Value{}, err
			}
			if t.ent != argI(a, 0, 0) || e.anim == nil {
				return value.Value{}, fmt.Errorf("PlayBlendTree: tree %d belongs to entity %d", t.id, t.ent)
			}
			if len(t.clips) == 0 {
				return value.Value{}, fmt.Errorf("PlayBlendTree: tree %d has no clips (AddBlendClip)", t.id)
			}
			st := e.anim
			st.startFade(e.node, float32(argN(a, 2, 0)))
			st.tree = t
			st.mode = 1
			t.computeWeights()
			st.applyPose(e.node)
			w.MarkShadowDirty()
			return value.Num(0), nil
		}),
		"crossfadeanim": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, fmt.Errorf("CrossfadeAnim: %w", err)
			}
			if e.anim == nil || len(e.anim.clips) == 0 {
				return value.Value{}, fmt.Errorf("CrossfadeAnim: entity has no clips")
			}
			st := e.anim
			seq := argI(a, 1, 0)
			if len(a) >= 2 && a[1].Kind == value.KindStr {
				if seq, err = st.findName(argS(a, 1)); err != nil {
					return value.Value{}, fmt.Errorf("CrossfadeAnim: no clip %q", argS(a, 1))
				}
			}
			if seq < 0 || seq >= len(st.clips) {
				return value.Value{}, fmt.Errorf("CrossfadeAnim: seq %d out of range (0..%d)", seq, len(st.clips)-1)
			}
			st.startFade(e.node, float32(argN(a, 2, 0.25)))
			st.tree = nil
			st.seq = seq
			st.time = st.first[seq]
			st.mode = argI(a, 3, 1)
			st.dir = 1
			st.prev = -1
			st.blend = 1
			if st.speed == 0 {
				st.speed = 1
			}
			st.applyPose(e.node)
			w.MarkShadowDirty()
			return value.Num(0), nil
		}),
		"crossfadeprogress": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, fmt.Errorf("CrossfadeProgress: %w", err)
			}
			if e.anim == nil || e.anim.fade == nil {
				return value.Num(1), nil
			}
			f := e.anim.fade
			return value.Num(float64(fadeWeight(f.elapsed, f.dur))), nil
		}),
		"freeblendtree": need(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			t := w.blendTrees[id]
			if t == nil {
				return value.Num(0), nil
			}
			if e := w.ents[t.ent]; e != nil && e.anim != nil {
				st := e.anim
				if st.tree == t {
					st.tree = nil
				}
				if st.fade != nil && st.fade.from.tree == t {
					st.fade = nil
				}
			}
			delete(w.blendTrees, id)
			return value.Num(0), nil
		}),
	}
}

func (w *World) blendTree(id int, who string) (*blendTree, error) {
	t := w.blendTrees[id]
	if t == nil {
		return nil, fmt.Errorf("%s: invalid blend tree %d", who, id)
	}
	return t, nil
}

func (w *World) blendTreeAnim(a []value.Value, who string) (*blendTree, *animState, error) {
	t, err := w.blendTree(argI(a, 0, 0), who)
	if err != nil {
		return nil, nil, err
	}
	e := w.ents[t.ent]
	if e == nil || e.anim == nil {
		return nil, nil, fmt.Errorf("%s: entity %d of tree %d is gone", who, t.ent, t.id)
	}
	return t, e.anim, nil
}

// blendWeights1D gives each threshold a weight for x. Outside the range the
// nearest end clip gets 1; inside, the two neighbouring thresholds share it.
func blendWeights1D(xs []float32, x float32) []float32 {
	n := len(xs)
	w := make([]float32, n)
	if n == 0 {
		return w
	}
	o := make([]int, n)
	for i := range o {
		o[i] = i
	}
	sort.SliceStable(o, func(i, j int) bool { return xs[o[i]] < xs[o[j]] })
	if x <= xs[o[0]] {
		w[o[0]] = 1
		return w
	}
	if x >= xs[o[n-1]] {
		w[o[n-1]] = 1
		return w
	}
	for k := 0; k < n-1; k++ {
		lo, hi := xs[o[k]], xs[o[k+1]]
		if x < hi {
			t := (x - lo) / (hi - lo)
			w[o[k]] = 1 - t
			w[o[k+1]] = t
			return w
		}
	}
	w[o[n-1]] = 1
	return w
}

// blendWeights2D is gradient band interpolation (freeform cartesian): each
// point's influence falls to 0 on the way to every other point. The result is
// normalized, and a point exactly on a clip gives that clip 1.
func blendWeights2D(xs, ys []float32, x, y float32) []float32 {
	n := len(xs)
	w := make([]float32, n)
	if n == 0 {
		return w
	}
	var sum float32
	for i := 0; i < n; i++ {
		wi := float32(1)
		px, py := x-xs[i], y-ys[i]
		for j := 0; j < n; j++ {
			if j == i {
				continue
			}
			dx, dy := xs[j]-xs[i], ys[j]-ys[i]
			d := dx*dx + dy*dy
			if d < 1e-12 {
				continue
			}
			h := 1 - (px*dx+py*dy)/d
			if h < 0 {
				h = 0
			}
			if h < wi {
				wi = h
			}
		}
		w[i] = wi
		sum += wi
	}
	if sum < 1e-6 {
		best, bd := 0, float32(-1)
		for i := 0; i < n; i++ {
			dx, dy := x-xs[i], y-ys[i]
			if d := dx*dx + dy*dy; bd < 0 || d < bd {
				best, bd = i, d
			}
		}
		for i := range w {
			w[i] = 0
		}
		w[best] = 1
		return w
	}
	for i := range w {
		w[i] /= sum
	}
	return w
}

func (t *blendTree) computeWeights() {
	xs := make([]float32, len(t.clips))
	ys := make([]float32, len(t.clips))
	for i, c := range t.clips {
		xs[i], ys[i] = c.x, c.y
	}
	if t.kind == 2 {
		t.weights = blendWeights2D(xs, ys, t.px, t.py)
	} else {
		t.weights = blendWeights1D(xs, t.px)
	}
}

// clipRange is a clip's playable span, matching tickAnims.
func (st *animState) clipRange(seq int) (t0, t1 float32) {
	if seq < 0 || seq >= len(st.first) {
		return 0, 0
	}
	t0, t1 = st.first[seq], st.last[seq]
	if t1 <= t0 {
		t1 = t0 + st.length[seq]
	}
	return t0, t1
}

func wrapTime(t, t0, t1 float32) float32 {
	span := t1 - t0
	if span <= 0 {
		return t0
	}
	for t > t1 {
		t -= span
	}
	for t < t0 {
		t += span
	}
	return t
}

// advance moves the tree on by dt. Synced trees share one normalized phase
// whose rate is the weighted average cycle length, so walk and run feet land
// together; unsynced clips each run on their own clock.
func (t *blendTree) advance(st *animState, dt float32) {
	if len(t.weights) != len(t.clips) {
		t.computeWeights()
	}
	if !t.sync {
		for i := range t.clips {
			c := &t.clips[i]
			t0, t1 := st.clipRange(c.seq)
			c.time = wrapTime(c.time+dt*t.speed, t0, t1)
		}
		return
	}
	var cycle float32
	for i, c := range t.clips {
		t0, t1 := st.clipRange(c.seq)
		cycle += t.weights[i] * (t1 - t0)
	}
	if cycle > 1e-6 {
		t.phase += dt * t.speed / cycle
	}
	t.phase -= float32(int(t.phase))
	if t.phase < 0 {
		t.phase++
	}
	t.syncTimes(st)
}

// syncTimes puts every clip at the shared normalized phase.
func (t *blendTree) syncTimes(st *animState) {
	for i := range t.clips {
		c := &t.clips[i]
		t0, t1 := st.clipRange(c.seq)
		c.time = t0 + t.phase*(t1-t0)
	}
}

// fadeWeight is how far a crossfade is into its target, smoothstepped.
func fadeWeight(elapsed, dur float32) float32 {
	if dur <= 0 {
		return 1
	}
	x := elapsed / dur
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	return x * x * (3 - 2*x)
}

// startFade captures the current animation as the crossfade source.
func (st *animState) startFade(root core.INode, dur float32) {
	if dur <= 0 || root == nil {
		st.fade = nil
		return
	}
	var src animSource
	switch {
	case st.fade != nil:
		src.frozen = snapshotLocal(root)
	case st.tree != nil:
		src.tree = st.tree
	case st.seq >= 0 && st.seq < len(st.clips):
		src.seq, src.time = st.seq, st.time
		src.speed = st.speed * st.dir
		src.loop = st.mode == 1 || st.mode == 2
		if st.mode == 0 {
			src.speed = 0
		}
	default:
		src.frozen = snapshotLocal(root)
	}
	st.fade = &animFade{from: src, dur: dur}
}

// tickBlend advances an active tree and crossfade. It reports whether either
// needs a pose this frame.
func (st *animState) tickBlend(dt float32) bool {
	on := false
	if st.tree != nil && st.mode != 0 {
		st.tree.advance(st, dt)
		on = true
	}
	if f := st.fade; f != nil {
		on = true
		src := &f.from
		switch {
		case src.tree != nil:
			if src.tree != st.tree {
				src.tree.advance(st, dt)
			}
		case src.frozen == nil:
			t0, t1 := st.clipRange(src.seq)
			src.time += dt * src.speed
			if src.loop {
				src.time = wrapTime(src.time, t0, t1)
			} else if src.time > t1 {
				src.time = t1
			}
		}
		f.elapsed += dt
		if f.elapsed >= f.dur {
			st.fade = nil
		}
	}
	return on
}

// resetBind puts every node the mesh loaded with back to its bind pose,
// except the entity root, so a clip that skips a node does not inherit the
// last clip's value for it.
func (st *animState) resetBind() {
	if !st.bindOK || len(st.bind) < 2 {
		return
	}
	writeLocals(st.bind[1:])
}

func (st *animState) clipPose(root core.INode, seq int, t float32) []localXF {
	st.resetBind()
	st.evalClip(seq, t)
	return snapshotLocal(root)
}

// treePose samples every weighted clip and mixes them. The result is written
// to the nodes and returned.
func (st *animState) treePose(t *blendTree, root core.INode) []localXF {
	if len(t.weights) != len(t.clips) {
		t.computeWeights()
	}
	var poses [][]localXF
	var ws []float32
	for i, c := range t.clips {
		if t.weights[i] <= 1e-4 {
			continue
		}
		poses = append(poses, st.clipPose(root, c.seq, c.time))
		ws = append(ws, t.weights[i])
	}
	if len(poses) == 0 {
		st.resetBind()
		return snapshotLocal(root)
	}
	if len(poses) == 1 {
		return poses[0]
	}
	out := blendPoses(poses, ws)
	writeLocals(out)
	return out
}

func (st *animState) sourcePose(src *animSource, root core.INode) []localXF {
	switch {
	case src.frozen != nil:
		return src.frozen
	case src.tree != nil:
		return st.treePose(src.tree, root)
	default:
		return st.clipPose(root, src.seq, src.time)
	}
}

// applyBlendPose is applyPose for a playing tree or a running crossfade.
func (st *animState) applyBlendPose(root core.INode) {
	if root == nil {
		return
	}
	var base []localXF
	if st.tree != nil {
		base = st.treePose(st.tree, root)
	} else {
		st.resetBind()
		st.applyClipPose(root)
		base = snapshotLocal(root)
	}
	f := st.fade
	if f == nil {
		return
	}
	from := st.sourcePose(&f.from, root)
	if len(from) != len(base) {
		writeLocals(base)
		return
	}
	writeLocals(mixPoses(from, base, fadeWeight(f.elapsed, f.dur)))
}

// mixPoses lerps a into b by t (nlerp for rotations).
func mixPoses(a, b []localXF, t float32) []localXF {
	out := make([]localXF, len(b))
	u := 1 - t
	for i := range b {
		out[i] = localXF{n: b[i].n}
		out[i].p.X = a[i].p.X*u + b[i].p.X*t
		out[i].p.Y = a[i].p.Y*u + b[i].p.Y*t
		out[i].p.Z = a[i].p.Z*u + b[i].p.Z*t
		out[i].s.X = a[i].s.X*u + b[i].s.X*t
		out[i].s.Y = a[i].s.Y*u + b[i].s.Y*t
		out[i].s.Z = a[i].s.Z*u + b[i].s.Z*t
		out[i].q = nlerpQuat(a[i].q, b[i].q, t)
	}
	return out
}

// blendPoses is the weighted average of several poses of the same node tree.
// Rotations are summed on the first pose's hemisphere and normalized.
func blendPoses(poses [][]localXF, ws []float32) []localXF {
	if len(poses) == 0 {
		return nil
	}
	n := len(poses[0])
	var sum float32
	for k, p := range poses {
		if len(p) == n {
			sum += ws[k]
		}
	}
	out := make([]localXF, n)
	if sum <= 0 {
		copy(out, poses[0])
		return out
	}
	for i := 0; i < n; i++ {
		o := &out[i]
		o.n = poses[0][i].n
		ref := poses[0][i].q
		for k, p := range poses {
			if len(p) != n {
				continue
			}
			w := ws[k] / sum
			x := p[i]
			o.p.X += x.p.X * w
			o.p.Y += x.p.Y * w
			o.p.Z += x.p.Z * w
			o.s.X += x.s.X * w
			o.s.Y += x.s.Y * w
			o.s.Z += x.s.Z * w
			q := x.q
			if ref.X*q.X+ref.Y*q.Y+ref.Z*q.Z+ref.W*q.W < 0 {
				w = -w
			}
			o.q.X += q.X * w
			o.q.Y += q.Y * w
			o.q.Z += q.Z * w
			o.q.W += q.W * w
		}
		if o.q.X == 0 && o.q.Y == 0 && o.q.Z == 0 && o.q.W == 0 {
			o.q.W = 1
		}
		o.q.Normalize()
	}
	return out
}
