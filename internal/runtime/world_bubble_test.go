package runtime

import (
	"math"
	"testing"

	"github.com/g3n/engine/core"

	"bitshinbasic/internal/value"
)

func TestOldStreamCommandsStillWork(t *testing.T) {
	w := New(t.TempDir())
	if _, err := w.Call("createworldstream", []value.Value{value.Num(20), value.Num(2)}); err != nil {
		t.Fatal(err)
	}
	if w.stream == nil || !w.stream.fill || w.stream.size != 20 || w.stream.radius != 2 {
		t.Fatalf("stream %+v", w.stream)
	}
	if _, err := w.Call("loadchunk", []value.Value{value.Num(0), value.Num(1)}); err != nil {
		t.Fatal(err)
	}
	if w.streamChunkCount() != 1 {
		t.Fatalf("LoadChunk should still build the demo cell, count %d", w.streamChunkCount())
	}
	if _, err := w.Call("setstreamfollow", []value.Value{value.Num(7)}); err != nil {
		t.Fatal(err)
	}
	if w.stream.follow != 7 {
		t.Fatalf("follow %d", w.stream.follow)
	}
	if _, err := w.Call("setstreamorigin", []value.Value{value.Num(10), value.Num(0), value.Num(20)}); err != nil {
		t.Fatal(err)
	}
	if w.stream.follow != 0 || w.stream.ox != 10 || w.stream.oz != 20 {
		t.Fatalf("origin should win over follow, got follow %d ox %v oz %v", w.stream.follow, w.stream.ox, w.stream.oz)
	}
}

func TestSetPlayerDoesNotSpawnDemoProps(t *testing.T) {
	w := New(t.TempDir())
	n := core.NewNode()
	w.ents[4] = &Entity{node: n}
	if _, err := w.Call("setplayer", []value.Value{value.Num(4)}); err != nil {
		t.Fatal(err)
	}
	w.tickStream()
	if w.streamChunkCount() != 0 {
		t.Fatalf("SetPlayer should follow without the demo filler, chunks %d", w.streamChunkCount())
	}
	if _, err := w.Call("setstreamorigin", []value.Value{value.Num(12), value.Num(0), value.Num(4)}); err != nil {
		t.Fatal(err)
	}
	w.tickStream()
	if w.stream.follow != 0 || w.stream.ox != 12 || w.stream.oz != 4 {
		t.Fatalf("SetStreamOrigin still owns the center, follow %d ox %v oz %v", w.stream.follow, w.stream.ox, w.stream.oz)
	}
}

func TestSetPlayerArmsTheWorldBubble(t *testing.T) {
	w := New(t.TempDir())
	n := core.NewNode()
	w.ents[4] = &Entity{node: n}
	if _, err := w.Call("setplayer", []value.Value{value.Num(4)}); err != nil {
		t.Fatal(err)
	}
	if w.stream == nil || w.stream.follow != 4 || w.stream.fill {
		t.Fatalf("stream follow=%v fill=%v", w.stream, w.stream != nil && w.stream.fill)
	}
	if w.bubble == nil || !w.bubble.on || w.bubble.simRadius != 1 || !w.bubble.shift {
		t.Fatalf("bubble %+v", w.bubble)
	}
}

func TestWorldBubbleCanStayOff(t *testing.T) {
	w := New(t.TempDir())
	if _, err := w.Call("setworldbubble", []value.Value{value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	w.ents[2] = &Entity{node: core.NewNode()}
	if _, err := w.Call("setplayer", []value.Value{value.Num(2)}); err != nil {
		t.Fatal(err)
	}
	if w.stream != nil {
		t.Fatal("SetWorldBubble(0) should leave streaming to the expert commands")
	}
}

func TestShiftKeepsThePlayerNearZero(t *testing.T) {
	w := New(t.TempDir())
	n := core.NewNode()
	n.SetPosition(toG3N(5000, 2, 30))
	w.ents[4] = &Entity{node: n}
	if _, err := w.Call("setplayer", []value.Value{value.Num(4)}); err != nil {
		t.Fatal(err)
	}
	w.tickBubble()
	x, y, z := fromG3N(n.Position().X, n.Position().Y, n.Position().Z)
	if math.Abs(float64(x)) > 24 || math.Abs(float64(y-2)) > 0.01 || math.Abs(float64(z)) > 24 {
		t.Fatalf("local position %v %v %v", x, y, z)
	}
	ox, oz := w.worldOrigin()
	if math.Abs(float64(ox-(5000-x))) > 0.1 || math.Abs(float64(oz-(30-z))) > 0.1 {
		t.Fatalf("origin %v %v local %v %v", ox, oz, x, z)
	}
	got, err := w.Call("worldoriginx", nil)
	if err != nil || math.Abs(got.Number()-float64(ox)) > 0.1 {
		t.Fatalf("WorldOriginX %v %v", got, err)
	}
}

func TestCoarseHeightMatchesTheNextLattice(t *testing.T) {
	raw := func(x, z float32) float32 { return x * x }
	got := coarseY(raw, 2, 0, 0, 0, 16, 16, 8, 1, 1)
	if math.Abs(float64(got-8)) > 0.05 {
		t.Fatalf("morphed height %v", got)
	}
	if coarseY(raw, 0, 0, 0, 0, 16, 16, 8, 1, 1) != 0 {
		t.Fatal("a lattice point should stay put")
	}
}

func TestLODMorphRisesNearTheBoundary(t *testing.T) {
	tr := &terrain{lod: 1}
	if step, morph := tr.lodBuild(0); step != 1 || morph != 0 {
		t.Fatalf("near %d %v", step, morph)
	}
	step, morph := tr.lodBuild(1.9)
	if step != 1 || morph < 0.5 {
		t.Fatalf("boundary %d %v", step, morph)
	}
	if !(&World{bubble: &worldBubble{on: true, simRadius: 1}}).chunkInSim(chunkKey{}, 32) {
		t.Fatal("the chunk under the origin is inside the sim ring")
	}
	far := &World{bubble: &worldBubble{on: true, simRadius: 1}}
	if far.chunkInSim(chunkKey{X: 3}, 32) {
		t.Fatal("a far chunk should stay visual-only")
	}
}
