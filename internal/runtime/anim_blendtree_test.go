package runtime

import (
	"math"
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-4 }

func TestBlendWeights1DThresholdsAndClamp(t *testing.T) {
	xs := []float32{0, 1.5, 4} // idle, walk, run
	cases := []struct {
		x    float32
		want []float32
	}{
		{-2, []float32{1, 0, 0}},
		{0, []float32{1, 0, 0}},
		{0.75, []float32{0.5, 0.5, 0}},
		{1.5, []float32{0, 1, 0}},
		{2.75, []float32{0, 0.5, 0.5}},
		{4, []float32{0, 0, 1}},
		{9, []float32{0, 0, 1}},
	}
	for _, c := range cases {
		w := blendWeights1D(xs, c.x)
		for i := range w {
			if !near(w[i], c.want[i]) {
				t.Fatalf("x=%v weights %v want %v", c.x, w, c.want)
			}
		}
	}
	// Thresholds added out of order still blend by value.
	w := blendWeights1D([]float32{4, 0, 1.5}, 0.75)
	if !near(w[1], 0.5) || !near(w[2], 0.5) || w[0] != 0 {
		t.Fatalf("unsorted %v", w)
	}
	if w := blendWeights1D([]float32{2}, -5); w[0] != 1 {
		t.Fatalf("single %v", w)
	}
	if w := blendWeights1D(nil, 1); len(w) != 0 {
		t.Fatal("empty")
	}
}

func TestBlendWeights2DSumToOne(t *testing.T) {
	// idle centre, forward, back, left, right
	xs := []float32{0, 0, 0, -1, 1}
	ys := []float32{0, 1, -1, 0, 0}
	for y := float32(-2); y <= 2; y += 0.25 {
		for x := float32(-2); x <= 2; x += 0.25 {
			w := blendWeights2D(xs, ys, x, y)
			var sum float32
			for _, v := range w {
				if v < 0 {
					t.Fatalf("(%v,%v) negative %v", x, y, w)
				}
				sum += v
			}
			if !near(sum, 1) {
				t.Fatalf("(%v,%v) sum %v (%v)", x, y, sum, w)
			}
		}
	}
	for i := range xs {
		w := blendWeights2D(xs, ys, xs[i], ys[i])
		if !near(w[i], 1) {
			t.Fatalf("on clip %d weights %v", i, w)
		}
	}
	w := blendWeights2D(xs, ys, 0, 0.5)
	if !near(w[0], 0.5) || !near(w[1], 0.5) {
		t.Fatalf("halfway forward %v", w)
	}
}

func testBlendAnim() *animState {
	return &animState{
		names:  []string{"idle", "walk", "run"},
		first:  []float32{0, 0, 0},
		last:   []float32{2, 1, 0.5},
		length: []float32{2, 1, 0.5},
		dir:    1, speed: 1, prev: -1, blend: 1, mode: 1,
	}
}

func TestBlendTreePhaseSync(t *testing.T) {
	st := testBlendAnim()
	tr := &blendTree{kind: 1, speed: 1, sync: true, clips: []blendClip{{seq: 1, x: 1}, {seq: 2, x: 2}}}
	tr.px = 1.5 // half walk (1s), half run (0.5s): cycle 0.75s
	tr.computeWeights()
	tr.advance(st, 0.375)
	if !near(tr.phase, 0.5) {
		t.Fatalf("phase %v want 0.5", tr.phase)
	}
	if !near(tr.clips[0].time, 0.5) || !near(tr.clips[1].time, 0.25) {
		t.Fatalf("clip times %v %v", tr.clips[0].time, tr.clips[1].time)
	}
	tr.advance(st, 0.5) // wraps past 1
	if tr.phase < 0 || tr.phase >= 1 || !near(tr.phase, 0.5+0.5/0.75-1) {
		t.Fatalf("wrapped phase %v", tr.phase)
	}
	// Both clips stay at the same normalized time.
	if !near(tr.clips[0].time/1, tr.clips[1].time/0.5) {
		t.Fatalf("out of phase %v %v", tr.clips[0].time, tr.clips[1].time)
	}
	// Unsynced clips run on their own clocks.
	tr.sync = false
	tr.clips[0].time, tr.clips[1].time = 0, 0
	tr.advance(st, 0.6)
	if !near(tr.clips[0].time, 0.6) || !near(tr.clips[1].time, 0.1) {
		t.Fatalf("free clocks %v %v", tr.clips[0].time, tr.clips[1].time)
	}
}

func TestCrossfadeProgress(t *testing.T) {
	if fadeWeight(0, 0) != 1 || fadeWeight(0, 1) != 0 || fadeWeight(2, 1) != 1 {
		t.Fatal("fade ends")
	}
	if !near(fadeWeight(0.5, 1), 0.5) {
		t.Fatalf("mid %v", fadeWeight(0.5, 1))
	}
	if !(fadeWeight(0.25, 1) < 0.25) || !(fadeWeight(0.75, 1) > 0.75) {
		t.Fatal("smoothstep should ease in and out")
	}
	st := testBlendAnim()
	st.fade = &animFade{from: animSource{seq: 0, speed: 1, loop: true}, dur: 0.4}
	if !st.tickBlend(0.1) || st.fade == nil || !near(st.fade.elapsed, 0.1) || !near(st.fade.from.time, 0.1) {
		t.Fatalf("fade tick %+v", st.fade)
	}
	st.tickBlend(0.2)
	st.tickBlend(0.2)
	if st.fade != nil {
		t.Fatal("fade should end after its duration")
	}
}

func TestBlendPosesWeighted(t *testing.T) {
	q90 := math32.Quaternion{Y: float32(math.Sin(math.Pi / 4)), W: float32(math.Cos(math.Pi / 4))}
	a := []localXF{{p: math32.Vector3{X: 0}, s: math32.Vector3{X: 1, Y: 1, Z: 1}, q: math32.Quaternion{W: 1}}}
	b := []localXF{{p: math32.Vector3{X: 4}, s: math32.Vector3{X: 1, Y: 1, Z: 1}, q: q90}}
	out := blendPoses([][]localXF{a, b}, []float32{3, 1})
	if !near(out[0].p.X, 1) || !near(out[0].s.Y, 1) {
		t.Fatalf("blend %+v", out[0])
	}
	if out[0].q.Y <= 0 || out[0].q.Y >= q90.Y {
		t.Fatalf("quat %+v", out[0].q)
	}
	m := mixPoses(a, b, 0.5)
	if !near(m[0].p.X, 2) {
		t.Fatalf("mix %+v", m[0])
	}
}

func findNodeNamed(n core.INode, name string) *core.Node {
	if n == nil {
		return nil
	}
	node := n.GetNode()
	if node.Name() == name {
		return node
	}
	for _, c := range node.Children() {
		if f := findNodeNamed(c, name); f != nil {
			return f
		}
	}
	return nil
}

func TestBlendTreeDrivesWalkerGLTF(t *testing.T) {
	w := New("../../examples")
	id, err := w.loadAnimMesh("assets/walker.gltf", 0)
	if err != nil {
		t.Skipf("walker.gltf: %v", err)
	}
	st := w.ents[id].anim
	if len(st.clips) < 3 {
		t.Fatalf("clips %v", st.names)
	}
	walk, _ := st.findName("walk")
	run, _ := st.findName("run")
	idle, _ := st.findName("idle")
	tr := &blendTree{id: 1, ent: id, kind: 1, speed: 1, sync: true,
		clips: []blendClip{{seq: idle, x: 0}, {seq: walk, x: 1.5}, {seq: run, x: 4}}}
	w.blendTrees = map[int]*blendTree{1: tr}
	st.tree, st.mode = tr, 1
	tr.px = 1.5
	tr.computeWeights()
	w.tickAnims(0.25) // walk quarter cycle: left leg fully forward
	leg := findNodeNamed(w.ents[id].node, "LegL")
	if leg == nil {
		t.Fatal("LegL")
	}
	walkX := leg.Quaternion().X
	if walkX < 0.15 {
		t.Fatalf("walk leg not swung: %v", walkX)
	}
	tr.phase = 0
	tr.px = 4
	tr.computeWeights()
	w.tickAnims(0.15) // run quarter cycle
	runX := leg.Quaternion().X
	if runX <= walkX {
		t.Fatalf("run leg %v should swing past walk %v", runX, walkX)
	}
	tr.phase = 0
	tr.px = 2.75
	tr.computeWeights()
	w.tickAnims(0.2) // blended cycle (1 + 0.6)/2 = 0.8s, quarter = 0.2s
	midX := leg.Quaternion().X
	if midX <= walkX || midX >= runX {
		t.Fatalf("blend leg %v not between walk %v and run %v", midX, walkX, runX)
	}
	// A crossfade to wave starts from the tree pose and ends on wave.
	wave, _ := st.findName("wave")
	st.startFade(w.ents[id].node, 0.3)
	st.tree, st.seq, st.time = nil, wave, 0
	w.tickAnims(0.15)
	if st.fade == nil || fadeWeight(st.fade.elapsed, st.fade.dur) <= 0 {
		t.Fatal("fade should be running")
	}
	w.tickAnims(0.2)
	if st.fade != nil {
		t.Fatal("fade should be done")
	}
}
