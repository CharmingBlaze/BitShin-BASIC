package runtime

import (
	"math/rand"
	"testing"

	"github.com/g3n/engine/math32"
)

func vec3At(a []float32, i int) math32.Vector3 {
	return math32.Vector3{X: a[i], Y: a[i+1], Z: a[i+2]}
}

func TestOakCardsCarryUVsAndShade(t *testing.T) {
	m := oakCards(rand.New(rand.NewSource(1)), 20, 40)
	if len(m.idx) != 40*6 || len(m.pos) != 40*4*3 {
		t.Fatalf("got %d indices, %d position floats", len(m.idx), len(m.pos))
	}
	for i := 0; i < len(m.col); i += 3 {
		u, v := m.col[i], m.col[i+1]
		if (u != 0 && u != 1) || (v != 0 && v != 1) {
			t.Fatalf("vertex %d: uv %v,%v is not a card corner", i/3, u, v)
		}
		if m.col[i+2] <= 0 {
			t.Fatalf("vertex %d: shade %v", i/3, m.col[i+2])
		}
	}
}

func TestFarCardsKeepWholeCards(t *testing.T) {
	m := oakCards(rand.New(rand.NewSource(2)), 20, 40)
	g := m.geometry()
	if g == nil {
		t.Fatal("no geometry")
	}
	p := meshPart{}
	for i := 0; i+2 < len(m.pos); i += 3 {
		var v, n = vec3At(m.pos, i), vec3At(m.nor, i)
		p.verts = append(p.verts, v)
		p.vnors = append(p.vnors, n)
		p.shades = append(p.shades, m.col[i+2])
		p.cardUV = append(p.cardUV, m.col[i], m.col[i+1])
	}
	f := farCards(p, 2, 1.35)
	if len(f.verts) != 20*4 || len(f.idx) != 20*6 || len(f.cardUV) != 20*8 {
		t.Fatalf("far: %d verts, %d indices, %d uv floats", len(f.verts), len(f.idx), len(f.cardUV))
	}
	for i := range f.idx {
		if int(f.idx[i]) >= len(f.verts) {
			t.Fatalf("index %d out of range", f.idx[i])
		}
	}
}

func TestFarCrownsAreCheaper(t *testing.T) {
	r := func() *rand.Rand { return rand.New(rand.NewSource(3)) }
	if near, far := pineCrown(r(), 20, 8, 11, true), pineCrown(r(), 20, 4, 6, false); len(far.idx)*4 > len(near.idx) {
		t.Fatalf("pine: far %d indices, near %d", len(far.idx), len(near.idx))
	}
	if near, far := oakCrown(r(), 20, 7, 0.7, false), oakCrown(r(), 20, 6, 0.7, true); len(far.idx)*4 > len(near.idx) {
		t.Fatalf("oak: far %d indices, near %d", len(far.idx), len(near.idx))
	}
}

func TestGrassTuftGrowsFromTheGround(t *testing.T) {
	p := grassTuft(0.55)
	if len(p.verts) != 15 || len(p.idx) != 15 || len(p.shades) != 15 {
		t.Fatalf("got %d verts, %d indices, %d shades", len(p.verts), len(p.idx), len(p.shades))
	}
	for i := 0; i < len(p.verts); i += 3 {
		if p.verts[i].Y != 0 || p.verts[i+1].Y != 0 || p.verts[i+2].Y <= 0 {
			t.Fatalf("blade %d: %v %v %v", i/3, p.verts[i], p.verts[i+1], p.verts[i+2])
		}
	}
}

func TestMountainIslandIsAnIsland(t *testing.T) {
	const n = 96
	d := make([]float32, n*n)
	fillMountainIsland(d, n, n, 2026)
	var land int
	for i, v := range d {
		if v < 0 || v > 1 || v != v {
			t.Fatalf("height %d is %v", i, v)
		}
		if v > 0.3 {
			land++
		}
	}
	if land < n*n/10 {
		t.Fatalf("only %d of %d cells above the sea", land, n*n)
	}
	for i := 0; i < n; i++ {
		if d[i] > 0.3 || d[(n-1)*n+i] > 0.3 {
			t.Fatalf("land reaches the map edge at column %d", i)
		}
	}
}
