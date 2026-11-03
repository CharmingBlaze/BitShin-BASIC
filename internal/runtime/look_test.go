package runtime

import (
	"strings"
	"testing"

	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func TestSetLookRejectsUnknownNames(t *testing.T) {
	w := New(".")
	if _, err := w.Call("setlook", []value.Value{value.Str("shiny")}); err == nil || !strings.Contains(err.Error(), "shiny") {
		t.Fatalf("unknown look should name itself in the error, got %v", err)
	}
}

func TestSetLookNeedsAWindow(t *testing.T) {
	w := New(".")
	if _, err := w.Call("setlook", []value.Value{value.Str("modern")}); err == nil || !strings.Contains(err.Error(), "Graphics3D") {
		t.Fatalf("modern with no window should ask for Graphics3D, got %v", err)
	}
}

// Linear light needs the post pass to encode it, and a pass with no post step
// (an extra window) must take the old path.
func TestLinearLightNeedsThePostPass(t *testing.T) {
	w := New(".")
	if w.linearActive() {
		t.Fatal("off by default: old games must keep their look")
	}
	w.linearLight = true
	if w.linearActive() {
		t.Fatal("no post pass, so nothing would encode the result")
	}
	w.post.on = true
	if !w.linearActive() {
		t.Fatal("should be on with the post pass")
	}
	w.linearHold = true
	if w.linearActive() {
		t.Fatal("an extra window has no post step")
	}
}

func TestLinearClearDecodesOnlyWhenActive(t *testing.T) {
	w := New(".")
	c := math32.Color{R: 0.5, G: 0.5, B: 0.5}
	if w.linearClear(c) != c {
		t.Fatal("the old path keeps the color")
	}
	w.linearLight, w.post.on = true, true
	if got := w.linearClear(c); got.R < 0.2 || got.R > 0.23 {
		t.Fatalf("0.5 should decode to about 0.22, got %v", got.R)
	}
}

func TestSetLookOffClearsTheLook(t *testing.T) {
	w := New(".")
	w.iblOn, w.post.on, w.post.ssao, w.fogMode = true, true, true, 1
	if err := w.applyLook("off"); err != nil {
		t.Fatal(err)
	}
	if w.iblOn || w.post.on || w.post.ssao || w.fogMode != 0 {
		t.Fatalf("off should clear the look: ibl %v post %v fog %d", w.iblOn, w.post.on, w.fogMode)
	}
}

// A small hill keeps its snow near the height asked for: the line used to
// wander by fixed metres, which buried an 18 m hill under SetTerrainSnow 16.
func TestSnowLineFollowsSmallHills(t *testing.T) {
	small := &terrain{snowH: 16}
	for _, p := range [][2]float32{{0, 0}, {40, -70}, {300, 120}, {-510, 900}} {
		if got := snowLineAt(small, p[0], p[1], 1); got < 11.5 || got > 20 {
			t.Fatalf("snow line at %v is %.1f for a height of 16", p, got)
		}
	}
	// Tall mountains keep the full wander they had.
	tall := &terrain{snowH: 140}
	want := float32(140-6+3-2.5) - 9
	if got := snowLineAt(tall, 0, 0, 1); got < want || got > want+18 {
		t.Fatalf("tall snow line %.1f is outside %.1f..%.1f", got, want, want+18)
	}
}
