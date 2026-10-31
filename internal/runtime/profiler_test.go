package runtime

import (
	"math"
	"testing"

	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func TestProfilerFrameStatsAndBars(t *testing.T) {
	var p profiler
	if avg, worst := p.frameStats(); avg != 0 || worst != 0 {
		t.Fatal("empty profiler has stats")
	}
	for i := 0; i < profFrames+10; i++ { // wraps the ring
		p.frames[p.idx] = 10
		if i == profFrames+5 {
			p.frames[p.idx] = 40
		}
		p.idx = (p.idx + 1) % profFrames
		p.filled = min(p.filled+1, profFrames)
	}
	avg, worst := p.frameStats()
	if worst != 40 || math.Abs(avg-(10*float64(profFrames-1)+40)/profFrames) > 1e-9 {
		t.Fatalf("avg %v worst %v", avg, worst)
	}
	for _, c := range []struct {
		ms      float64
		r, g, b uint8
	}{{10, 90, 210, 110}, {25, 235, 200, 70}, {80, 235, 80, 70}} {
		h, r, g, b := profBar(c.ms)
		if r != c.r || g != c.g || b != c.b || h <= 0 || h > 60 {
			t.Errorf("bar for %v ms: h %v rgb %d,%d,%d", c.ms, h, r, g, b)
		}
	}
	if smooth(0, 5) != 5 || smooth(10, 0) != 9 {
		t.Error("smoothing")
	}
}

func TestProfilerCommands(t *testing.T) {
	w := New(".")
	cmds := w.commandTable()
	for _, name := range []string{"showprofiler", "statsframems", "statsworstms", "statsscriptms", "statsphysicsms",
		"statsrenderms", "statsgpums", "statstriangles", "statsmemorymb", "statsgcpausems", "statsallockb"} {
		if cmds[name] == nil {
			t.Fatalf("missing %s", name)
		}
	}
	if _, err := cmds["showprofiler"]([]value.Value{value.Num(1)}); err != nil || !w.prof.show {
		t.Fatalf("ShowProfiler: %v", err)
	}
	if v, _ := cmds["statsmemorymb"](nil); v.Num <= 0 {
		t.Errorf("heap %v MB", v.Num)
	}
}

// Planes, quads, and disks lie flat in their mesh, so the entity's own
// rotation starts at 0: RotateEntity and ScaleEntity mean what they say.
func TestFlatShapesHaveNoHiddenRotation(t *testing.T) {
	w := New(".")
	w.ready = true
	for _, mk := range []func() int{
		func() int { return w.createPlaneMesh([]value.Value{value.Num(4), value.Num(2)}) },
		func() int { return w.createQuadMesh(nil) },
		func() int { return w.createDiskMesh(nil) },
	} {
		e := w.ents[mk()]
		if e.pitch != 0 {
			t.Fatalf("hidden pitch %v", e.pitch)
		}
		var q math32.Quaternion
		e.node.GetNode().WorldQuaternion(&q)
		if q.X != 0 || q.Y != 0 || q.Z != 0 {
			t.Fatalf("node rotation %v", q)
		}
		g := e.mesh.GetGeometry()
		bb := g.BoundingBox()
		if h := bb.Max.Y - bb.Min.Y; h > 1e-4 {
			t.Fatalf("mesh not flat on XZ: height %v", h)
		}
		normal := math32.Vector3{}
		g.OperateOnVertexNormals(func(n *math32.Vector3) bool { normal = *n; return true })
		if normal.Y < 0.999 {
			t.Fatalf("normal %v does not face up", normal)
		}
	}
	// The plane keeps its requested size: 4 wide (x), 2 deep (z).
	e := w.ents[w.createPlaneMesh([]value.Value{value.Num(4), value.Num(2)})]
	bb := e.mesh.GetGeometry().BoundingBox()
	if bb.Max.X-bb.Min.X != 4 || math.Abs(float64(bb.Max.Z-bb.Min.Z)-2) > 1e-5 {
		t.Fatalf("plane size %v", bb)
	}
}
