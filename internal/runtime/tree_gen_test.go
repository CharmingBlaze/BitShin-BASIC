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
	r1 := rand.New(rand.NewSource(1))
	m := oakCards(r1, 20, oakLobes(r1, 20), 40)
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
	r2 := rand.New(rand.NewSource(2))
	m := oakCards(r2, 20, oakLobes(r2, 20), 40)
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
	lobes := oakLobes(r(), 20)
	if near, far := oakCrown(r(), 20, lobes, 8, 0.55, false), oakCrown(r(), 20, lobes, 8, 0.2, true); len(far.idx)*3 > len(near.idx) {
		t.Fatalf("oak: far %d indices, near %d", len(far.idx), len(near.idx))
	}
}

func TestGrassTuftGrowsFromTheGround(t *testing.T) {
	p := grassTuft(0.55, false)
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

func TestCurvedGrassKeepsItsBladesWhole(t *testing.T) {
	p := grassTuft(0.55, true)
	if len(p.verts) != 25 || len(p.idx) != 45 || len(p.shades) != 25 {
		t.Fatalf("got %d verts, %d indices, %d shades", len(p.verts), len(p.idx), len(p.shades))
	}
	for _, i := range p.idx {
		if int(i) >= len(p.verts) {
			t.Fatalf("index %d of %d", i, len(p.verts))
		}
	}
}

func TestOakLobesAreSeparateMasses(t *testing.T) {
	lobes := oakLobes(rand.New(rand.NewSource(5)), 20)
	if len(lobes) < 5 {
		t.Fatalf("%d lobes", len(lobes))
	}
	top := lobes[0].c.Y
	for _, l := range lobes[1:] {
		if l.c.Y >= top {
			t.Fatalf("lobe at %v above the top at %v", l.c.Y, top)
		}
	}
}

func TestWaterNormalTiles(t *testing.T) {
	const n = 256
	img := genWaterNormal(n)
	var worst, inner int
	diff := func(x0, y0, x1, y1 int) int {
		a, b := img.PixOffset(x0, y0), img.PixOffset(x1, y1)
		d := 0
		for c := 0; c < 3; c++ {
			v := int(img.Pix[a+c]) - int(img.Pix[b+c])
			if v < 0 {
				v = -v
			}
			d += v
		}
		return d
	}
	for i := 0; i < n; i++ {
		if d := diff(n-1, i, 0, i); d > worst {
			worst = d
		}
		if d := diff(n/2, i, n/2+1, i); d > inner {
			inner = d
		}
	}
	// Across the wrap should be no rougher than across any other two columns.
	if worst > inner*2+8 {
		t.Fatalf("seam at the wrap: %d, against %d inside the tile", worst, inner)
	}
}

func TestQualityProfilesNameTheTiers(t *testing.T) {
	for name, want := range map[string]int{"classic": qualityLow, "enhanced": qualityMedium, "advanced": qualityHigh, "Advanced ": qualityHigh} {
		if got, auto := parseQuality(name); auto || got != want {
			t.Fatalf("%q: tier %d auto %v, want tier %d", name, got, auto, want)
		}
	}
}

func TestCloudSheetStaysInsideTheFarPlane(t *testing.T) {
	c := &cloudLayer{y: 420}
	if r := c.cloudRadius(4000); r > 4000 {
		t.Fatalf("radius %v past the far plane", r)
	}
	if r := c.cloudRadius(0); r != 9240 {
		t.Fatalf("with no camera the radius should be its own, got %v", r)
	}
}

func TestPineShapeVariesBySeed(t *testing.T) {
	widest := func(seed int64) float32 {
		m := pineCrown(rand.New(rand.NewSource(seed)), 20, 8, 11, true)
		var w float32
		for i := 0; i+2 < len(m.pos); i += 3 {
			if d := m.pos[i]*m.pos[i] + m.pos[i+2]*m.pos[i+2]; d > w {
				w = d
			}
		}
		return w
	}
	if widest(1) == widest(2) {
		t.Fatal("two seeds gave the same width")
	}
}
