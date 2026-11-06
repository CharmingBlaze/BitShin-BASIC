package runtime

import (
	"fmt"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/math32"
	"image"
	"image/png"
	"math"
	"moonbasic/internal/meshsimplify"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"moonbasic/internal/value"
)

func TestAutoTierKnowsOldGPUs(t *testing.T) {
	for _, c := range []struct {
		renderer string
		want     int
	}{
		{"Intel(R) HD Graphics 520", qualityLow},
		{"Intel(R) HD Graphics 4000", qualityLow},
		{"Mesa Intel(R) UHD Graphics 620 (KBL GT2)", qualityLow},
		{"AMD Radeon(TM) R7 Graphics", qualityMedium},
		{"AMD Radeon RX 580", qualityHigh},
		{"Intel(R) Iris(R) Xe Graphics", qualityMedium},
		{"Intel(R) Arc(TM) A770 Graphics", qualityHigh},
		{"GeForce GT 730/PCIe/SSE2", qualityLow},
		{"NVIDIA GeForce GT 1030/PCIe/SSE2", qualityMedium},
		{"NVIDIA GeForce GTX 750 Ti/PCIe/SSE2", qualityMedium},
		{"NVIDIA GeForce GTX 660/PCIe/SSE2", qualityMedium},
		{"NVIDIA GeForce GTX 970/PCIe/SSE2", qualityHigh},
		{"NVIDIA GeForce GTX 1060 6GB/PCIe/SSE2", qualityHigh},
		{"NVIDIA GeForce RTX 4070/PCIe/SSE2", qualityHigh},
		{"NVIDIA GeForce 940MX/PCIe/SSE2", qualityMedium},
		{"AMD Radeon HD 6450", qualityLow},
		{"AMD Radeon HD 7770", qualityMedium},
		{"AMD Radeon R5 Graphics", qualityLow},
		{"AMD Radeon R7 200 Series", qualityMedium},
		{"AMD Radeon(TM) Vega 8 Graphics", qualityMedium},
		{"AMD Radeon RX 480", qualityHigh},
		{"AMD Radeon 780M Graphics", qualityHigh},
		{"llvmpipe (LLVM 15.0.7, 256 bits)", qualityLow},
		{"", qualityLow},
	} {
		if got := autoTier(c.renderer); got != c.want {
			t.Errorf("%q: got %s, want %s", c.renderer, qualityNames[got], qualityNames[c.want])
		}
	}
}

func TestProfilesStepUp(t *testing.T) {
	want := []struct{ cascades, slices, maxTex int }{
		qualityLow:    {2, 0, 1024},
		qualityMedium: {3, 2, 2048},
		qualityHigh:   {4, 4, 0},
		qualityUltra:  {4, 6, 0},
	}
	for tier, c := range want {
		w := New(".")
		w.setQuality(tier)
		if w.shadow.cascades != c.cascades {
			t.Errorf("%s: cascades %d, want %d", qualityNames[tier], w.shadow.cascades, c.cascades)
		}
		if w.post.ssao != (c.slices > 0) || (c.slices > 0 && w.post.aoSlices != c.slices) {
			t.Errorf("%s: AO %v with %d slices, want %d", qualityNames[tier], w.post.ssao, w.post.aoSlices, c.slices)
		}
		if w.textureLimit() != c.maxTex {
			t.Errorf("%s: texture limit %d, want %d", qualityNames[tier], w.textureLimit(), c.maxTex)
		}
	}
}

func TestSoftShadowsFromHighUp(t *testing.T) {
	for tier, want := range []int{qualityLow: shadowFilterPCF, qualityMedium: shadowFilterPCF, qualityHigh: shadowFilterPCSS, qualityUltra: shadowFilterPCSS} {
		w := New(".")
		w.setQuality(tier)
		if w.shadow.filter != want {
			t.Errorf("%s: filter %d, want %d", qualityNames[tier], w.shadow.filter, want)
		}
	}
}

func TestTextureLimitShrinksOnlyWhatIsTooBig(t *testing.T) {
	big := image.NewRGBA(image.Rect(0, 0, 4096, 2048))
	got := shrinkToLimit(big, 1024)
	if b := got.Bounds(); b.Dx() != 1024 || b.Dy() != 512 {
		t.Fatalf("got %v", b)
	}
	small := image.NewRGBA(image.Rect(0, 0, 512, 512))
	if shrinkToLimit(small, 1024) != small {
		t.Fatal("an image within the limit should be left alone")
	}
	if shrinkToLimit(big, 0) != big {
		t.Fatal("no limit should leave it alone")
	}
}

func TestHalvingAverages(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	copy(src.Pix, []uint8{0, 0, 0, 255, 100, 0, 0, 255, 0, 100, 0, 255, 100, 100, 200, 255})
	d := halveImage(src)
	if d.Pix[0] != 50 || d.Pix[1] != 50 || d.Pix[2] != 50 || d.Pix[3] != 255 {
		t.Fatalf("got %v", d.Pix[:4])
	}
}

func TestScriptTextureLimitWins(t *testing.T) {
	w := New(".")
	w.setQuality(qualityLow)
	if _, err := w.Call("settexturelimit", []value.Value{value.Num(0)}); err != nil {
		t.Fatal(err)
	}
	if w.textureLimit() != 0 {
		t.Fatal("a script's limit should win over the tier")
	}
}

func TestLargeLoneMeshesGetDetailLevels(t *testing.T) {
	big := geometry.NewSphere(1, 64, 32)
	small := geometry.NewBox(1, 1, 1)
	if !wantsOwnLOD(&batchGroup{key: batchKey{geom: big}, members: make([]batchMember, 1)}) {
		t.Fatal("a large mesh on its own should be batched for its detail levels")
	}
	if wantsOwnLOD(&batchGroup{key: batchKey{geom: small}, members: make([]batchMember, 1)}) {
		t.Fatal("a small mesh on its own has nothing to gain")
	}
	if wantsOwnLOD(&batchGroup{key: batchKey{geom: big}}) {
		t.Fatal("no members, no batch")
	}
}

func TestPlantedTreeBlocksKeepTheirOwnDetail(t *testing.T) {
	big := geometry.NewSphere(1, 64, 32)
	g := &batchGroup{key: batchKey{geom: big}, members: []batchMember{{e: &Entity{name: "scatter"}}}}
	if wantsOwnLOD(g) {
		t.Fatal("a planted tree block has its own far versions")
	}
	g.members[0].e.name = "rock"
	if !wantsOwnLOD(g) {
		t.Fatal("any other large mesh should get levels")
	}
}

func TestSimplifyGoesDeepOnASmoothMesh(t *testing.T) {
	g := geometry.NewSphere(1, 128, 64)
	mesh := meshsimplify.Mesh{Positions: positionSlice(g), Indices: append([]uint32(nil), g.Indices()...)}
	out := simplifyForLOD(mesh)
	if len(out) != len(batchLODRatios) {
		t.Fatalf("got %d levels, want %d", len(out), len(batchLODRatios))
	}
	for i := 1; i < len(out); i++ {
		if out[i].OutputTriangles >= out[i-1].OutputTriangles || out[i].MaxCollapseError < out[i-1].MaxCollapseError {
			t.Fatalf("level %d: %d triangles, error %v, after %d, error %v", i, out[i].OutputTriangles, out[i].MaxCollapseError, out[i-1].OutputTriangles, out[i-1].MaxCollapseError)
		}
	}
}

func TestPickLODTakesTheSimplestThatFits(t *testing.T) {
	l := []lodLevel{{err: 0.001}, {err: 0.01}, {err: 0.1}, {err: 1}}
	if got := pickLOD(l, 1, 10000); got != 0 {
		t.Fatalf("close up should be full detail, got %d", got)
	}
	if got := pickLOD(l, 1, 50); got != 2 {
		t.Fatalf("got %d, want level 2", got)
	}
	if got := pickLOD(l, 1, 0.5); got != 4 {
		t.Fatalf("far off should be the simplest, got %d", got)
	}
}

func TestWeldSharesEqualVertices(t *testing.T) {
	// Two triangles of a quad, written out in full, with a material group over them.
	g := geometry.NewGeometry()
	pos := math32.NewArrayF32(0, 18)
	pos.Append(0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 1, 0)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	g.AddGroup(0, 6, 3)
	w := weldGeometry(g)
	if w == nil {
		t.Fatal("four of six vertices are unique: worth welding")
	}
	if got := len(*w.VBOs()[0].Buffer()); got != 12 {
		t.Fatalf("got %d floats, want 4 vertices", got)
	}
	idx := w.Indices()
	if len(idx) != 6 || idx[3] != idx[0] || idx[4] != idx[2] {
		t.Fatalf("indices %v should reuse the shared corners", idx)
	}
	if w.GroupCount() != 1 || w.GroupAt(0).Count != 6 || w.GroupAt(0).Matindex != 3 {
		t.Fatal("the material group should carry over")
	}
	if weldGeometry(w) != nil {
		t.Fatal("already indexed: nothing to do")
	}
}

func TestWeldLeavesHardEdgesAlone(t *testing.T) {
	g := geometry.NewGeometry()
	pos := math32.NewArrayF32(0, 18)
	pos.Append(0, 0, 0, 1, 0, 0, 1, 1, 0, 5, 0, 0, 6, 0, 0, 6, 1, 0)
	g.AddVBO(gls.NewVBO(pos).AddAttrib(gls.VertexPosition))
	if weldGeometry(g) != nil {
		t.Fatal("nothing shared, nothing to gain")
	}
}

// A smooth ball, in the form an OBJ file gives: smooth normals, no shared vertices once loaded.
func writeBallOBJ(t *testing.T, path string, n int) {
	t.Helper()
	var b strings.Builder
	for j := 0; j <= n; j++ {
		th := math.Pi * float64(j) / float64(n)
		for i := 0; i < n; i++ {
			ph := 2 * math.Pi * float64(i) / float64(n)
			x, y, z := math.Sin(th)*math.Cos(ph), math.Cos(th), math.Sin(th)*math.Sin(ph)
			fmt.Fprintf(&b, "v %f %f %f\nvn %f %f %f\n", x, y, z, x, y, z)
		}
	}
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			p := j*n + i + 1
			q := j*n + (i+1)%n + 1
			r, s := p+n, q+n
			fmt.Fprintf(&b, "f %d//%d %d//%d %d//%d\nf %d//%d %d//%d %d//%d\n", p, p, q, q, r, r, q, q, s, s, r, r)
		}
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadedModelPartsGetDetailLevels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ball.obj")
	writeBallOBJ(t, path, 64)
	w := New(filepath.Dir(path))
	w.ready = true
	id, err := w.loadObjAt(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	w.findPartLODs()
	if len(w.batch.parts) != 1 {
		t.Fatalf("found %d parts, want 1", len(w.batch.parts))
	}
	p := w.batch.parts[0]
	// The file has 64*65, and each pole is one point written 64 times.
	if got := len(*p.full.VBO(gls.VertexPosition).Buffer()) / 3; got > 64*65 || got < 64*63 {
		t.Fatalf("got %d vertices, want about the OBJ file's %d, not three per triangle", got, 64*65)
	}
	var l *geomLOD
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) && l == nil {
		w.lodFor(p.full)
		w.collectLODs()
		l = w.lodFor(p.full)
		time.Sleep(5 * time.Millisecond)
	}
	if l == nil || len(l.levels) < 2 {
		t.Fatalf("got %+v, want at least two simpler levels", l)
	}
	last := l.levels[len(l.levels)-1].geom
	if geomTriangles(last) >= geomTriangles(p.full)/4 {
		t.Fatalf("simplest level has %d of %d triangles", geomTriangles(last), geomTriangles(p.full))
	}
	w.restorePartLODs(w.ents[id])
	if len(w.batch.parts) != 0 {
		t.Fatal("freeing should drop the part")
	}
}

func TestSkyCubeKeepsEnoughDetailWhenRough(t *testing.T) {
	for tier, size := range skyCubeSizes {
		if last := size >> (skyCubeMips - 1); last < 2 {
			t.Errorf("%s: last mip is %d texels a face, and a rough ball shows the box", qualityNames[tier], last)
		}
	}
}

func TestSkyPrefilterMatchesCubeFaces(t *testing.T) {
	// The shader's faces must agree with envFaceDir, which the fallback sky is drawn with.
	for _, f := range []string{"vec3(1.0, -q.y, -q.x)", "vec3(-1.0, -q.y, q.x)", "vec3(q.x, 1.0, q.y)", "vec3(q.x, -1.0, -q.y)", "vec3(q.x, -q.y, 1.0)", "vec3(-q.x, -q.y, -1.0)"} {
		if !strings.Contains(mbEnvPrefilter, f) {
			t.Errorf("prefilter lacks face %s", f)
		}
	}
	// Each image pixel lands where sampleSkyImages would read it.
	for face := 0; face < 6; face++ {
		dx, dy, dz := envFaceDir(face, 3, 5, 16)
		got, u, v := skyDirToFace(dx, dy, dz)
		wantU, wantV := 2*(3.5)/16-1, 2*(5.5)/16-1
		if got != face || math.Abs(float64(u)-wantU) > 1e-4 || math.Abs(float64(v)-wantV) > 1e-4 {
			t.Errorf("face %d: went to face %d at %v, %v, want %v, %v", face, got, u, v, wantU, wantV)
		}
	}
}

func TestShadowCacheIsFarCascadesInTurns(t *testing.T) {
	w := New(".")
	w.ready = true
	w.Call("enableshadowcache", []value.Value{value.Num(0)})
	if !w.shadow.turnsOff {
		t.Fatal("off should redraw every cascade each frame")
	}
	w.Call("enableshadowcache", []value.Value{value.Num(1)})
	if w.shadow.turnsOff {
		t.Fatal("on should let far cascades take turns")
	}
}

func TestPushOutOfTrunks(t *testing.T) {
	tr := []trunk{{10, 10, 1}}
	x, z, moved := pushOutOfTrunks(10.5, 10, 0.5, tr)
	if !moved || x != 11.5 || z != 10 {
		t.Fatalf("got %v, %v, %v", x, z, moved)
	}
	if _, _, moved := pushOutOfTrunks(20, 10, 0.5, tr); moved {
		t.Fatal("far from the trunk nothing moves")
	}
	if x, z, moved := pushOutOfTrunks(10, 10, 0.5, tr); !moved || (x-10)*(x-10)+(z-10)*(z-10) < 2.2 {
		t.Fatalf("dead center should still come out: %v, %v", x, z)
	}
}

func TestLineUnderGround(t *testing.T) {
	hill := func(x, z float32) float32 {
		if x > 40 && x < 60 {
			return 20
		}
		return 0
	}
	if !lineUnderGround(hill, 0, 2, 0, 100, 2, 0, 4) {
		t.Fatal("a hill between two low points hides one from the other")
	}
	if lineUnderGround(hill, 0, 50, 0, 100, 50, 0, 4) {
		t.Fatal("a line above the hill is clear")
	}
	if lineUnderGround(hill, 0, 2, 0, 30, 2, 0, 4) {
		t.Fatal("the hill is not between these")
	}
}

func TestBlockBehindGroundNeedsEveryPointHidden(t *testing.T) {
	// A wall that stops short: the far corner of the block can be seen past its end.
	wall := func(x, z float32) float32 {
		if x > 40 && x < 60 && z < 20 {
			return 50
		}
		return 0
	}
	if blockBehindGround(wall, 0, 2, 0, 80, 0, 40, 30, 4) {
		t.Fatal("a block partly seen past the wall must be drawn")
	}
	full := func(x, z float32) float32 {
		if x > 40 && x < 60 {
			return 50
		}
		return 0
	}
	if !blockBehindGround(full, 0, 2, 0, 80, 0, 40, 30, 4) {
		t.Fatal("a block wholly behind the wall can be hidden")
	}
}

func TestShoreDepths(t *testing.T) {
	// Ground rising to the east, crossing the sea (at 10) in the middle.
	h := func(x, z float32) float32 { return x / 10 }
	d := shoreDepths(h, 10, 0, 0, 200, 200, 4)
	if d[0] == 0 || d[3] != 0 {
		t.Fatalf("west should be under water and east dry: %v", d[:4])
	}
	if d[0] <= d[1] {
		t.Fatalf("deeper to the west: %v", d[:4])
	}
	deep := shoreDepths(func(x, z float32) float32 { return -500 }, 10, 0, 0, 10, 10, 2)
	if deep[0] != 255 {
		t.Fatalf("past the map's range should be full, got %d", deep[0])
	}
}

func TestLoadTextureAsync(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	w := New(dir)
	w.ready = true
	w.setQuality(qualityHigh)
	w.Call("settexturelimit", []value.Value{value.Num(32)})
	v, err := w.Call("loadtextureasync", []value.Value{value.Str(path)})
	if err != nil || v.Num == 0 {
		t.Fatalf("got %v, %v", v, err)
	}
	id := value.Value{Kind: v.Kind, Num: v.Num}
	if r, _ := w.Call("textureready", []value.Value{id}); r.Num != 0 {
		t.Fatalf("not ready at once, got %v", r.Num)
	}
	if w.texs[int(v.Num)].tex.Width() != 1 {
		t.Fatal("it should be usable at once, as one pixel")
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(w.assets.pending) > 0 && time.Now().Before(deadline) {
		w.tickAssets()
		time.Sleep(time.Millisecond)
	}
	if r, _ := w.Call("textureready", []value.Value{id}); r.Num != 1 {
		t.Fatalf("should be ready, got %v", r.Num)
	}
	if got := w.texs[int(v.Num)].tex.Width(); got != 32 {
		t.Fatalf("width %d, want the limit 32", got)
	}
	if n, _ := w.Call("texturesloading", nil); n.Num != 0 {
		t.Fatal("nothing should be loading")
	}
}

func TestLoadTextureAsyncReportsAFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.png")
	os.WriteFile(path, []byte("not an image"), 0o644)
	w := New(dir)
	w.ready = true
	v, _ := w.Call("loadtextureasync", []value.Value{value.Str(path)})
	deadline := time.Now().Add(5 * time.Second)
	for len(w.assets.pending) > 0 && time.Now().Before(deadline) {
		w.tickAssets()
		time.Sleep(time.Millisecond)
	}
	if r, _ := w.Call("textureready", []value.Value{v}); r.Num != -1 {
		t.Fatalf("got %v, want -1", r.Num)
	}
}
