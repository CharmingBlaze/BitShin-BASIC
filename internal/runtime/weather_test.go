package runtime

import (
	"testing"

	"github.com/g3n/engine/math32"
)

func TestTransmittanceLUT(t *testing.T) {
	img := generateTransmittanceLUT()
	if img.Bounds().Dx() != atmoLUTW || img.Bounds().Dy() != atmoLUTH {
		t.Fatalf("lut size %v", img.Bounds())
	}
	// Zenith, mid-atmosphere should not be black.
	c := img.RGBAAt(atmoLUTW/2, atmoLUTH/2)
	if int(c.R)+int(c.G)+int(c.B) < 40 {
		t.Fatalf("lut mid too dark %+v", c)
	}
}

func TestSkyRadianceCPU(t *testing.T) {
	zr, zg, zb := skyRadianceCPU(0, 1, 0, -0.35, 0.8, 0.4, 1, 1)
	hr, hg, hb := skyRadianceCPU(1, 0.05, 0, -0.35, 0.8, 0.4, 1, 1)
	if zr+zg+zb <= 0 || hr+hg+hb <= 0 {
		t.Fatal("sky radiance zero")
	}
	if zb < hb {
		t.Fatalf("zenith should be bluer than horizon: z %v %v %v h %v %v %v", zr, zg, zb, hr, hg, hb)
	}
}

func TestLightningPath(t *testing.T) {
	a := math32.Vector3{0, 20, 0}
	b := math32.Vector3{1, 0, 2}
	pts := displaceBolt(a, b, 4, 2)
	if len(pts) < 8 {
		t.Fatalf("expected displaced bolt, got %d", len(pts))
	}
	g := boltRibbon(pts, 0.08)
	if g == nil {
		t.Fatal("ribbon nil")
	}
}

func TestSetWindAndWetness(t *testing.T) {
	w := New(".")
	w.setWind(2, 0, 1, 3)
	if w.wx.windX != 2 || w.wx.windStr != 3 {
		t.Fatalf("wind %+v str %v", w.wx, w.wx.windStr)
	}
	w.setWeatherTransition("rain", 0.9, 0)
	if w.wx.mode != "rain" {
		t.Fatalf("mode %q", w.wx.mode)
	}
	before := w.wetness
	for i := 0; i < 90; i++ {
		w.tickWeather(1.0 / 30)
	}
	if w.wetness <= before {
		t.Fatalf("rain should accumulate wetness: %v -> %v", before, w.wetness)
	}
	w.strikeLightning(0, 10, 0, 0, 0, 0)
	if w.wx.flash <= 0 {
		t.Fatal("expected flash")
	}
	if len(w.wx.thunder) == 0 {
		t.Fatal("expected delayed thunder")
	}
}

func TestWeatherTransitionRateRamp(t *testing.T) {
	w := New(".")
	w.setWeatherTransition("rain", 0.8, 2)
	if !w.wx.isTransitioning {
		t.Fatal("expected transition")
	}
	if w.wx.targetMode != "rain" {
		t.Fatalf("target %q", w.wx.targetMode)
	}
	if w.wx.mode == "rain" {
		t.Fatal("mode should stay previous until t>=1")
	}
	if len(w.wx.ids) == 0 {
		t.Fatal("incoming emitters should spawn immediately")
	}
	in := w.emitters[w.wx.ids[0]]
	if in == nil || in.rateBase <= 0 || in.rate >= in.rateBase*0.2 {
		t.Fatalf("incoming rate should start low: rate=%v base=%v", in.rate, in.rateBase)
	}
	w.tickWeather(2.1)
	if w.wx.isTransitioning || w.wx.mode != "rain" {
		t.Fatalf("expected finished rain, transitioning=%v mode=%q", w.wx.isTransitioning, w.wx.mode)
	}
	if in = w.emitters[w.wx.ids[0]]; in == nil || in.rate < in.rateBase*0.9 {
		t.Fatalf("incoming should reach full rate: %v / %v", in.rate, in.rateBase)
	}
}

func TestWeatherDrying(t *testing.T) {
	w := New(".")
	w.wetness = 0.8
	w.wx.dryingSpeed = 0.02
	w.wx.dryingInit = true
	w.setWeatherTransition("clear", 1, 0)
	w.tickWeather(1)
	if w.wetness >= 0.8 {
		t.Fatalf("clear should dry: %v", w.wetness)
	}
}
