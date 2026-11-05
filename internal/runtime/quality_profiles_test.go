package runtime

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
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
