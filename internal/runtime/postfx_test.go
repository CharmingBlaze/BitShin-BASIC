package runtime

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"moonbasic/internal/value"
)

func TestPostAndShaderCommandsRegistered(t *testing.T) {
	m := New(".").commandTable()
	for _, name := range []string{
		"enablepostfx", "setbloom", "setexposure", "settonemap", "setfxaa", "setcolorgrade", "postfx",
		"enablessao", "setssao", "getssao", "setssaoradius", "getssaoradius", "setssaointensity", "getssaointensity",
		"createshader", "loadshader", "setshader", "setshaderuniform", "shaderok",
		"mouselook", "stopanim", "setanimblend", "setsky", "savescene",
	} {
		if m[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}
}

func TestToneCurveNames(t *testing.T) {
	for name, want := range map[string]int{"reinhard": 0, "neutral": 1, "aces": 2, "none": 3, "hill": 4, "AgX": 5, " agx ": 5, "nonsense": 0} {
		if got := tonemapMode(name, 0); got != want {
			t.Errorf("%q: got %d, want %d", name, got, want)
		}
	}
	for n := 1; n <= 5; n++ {
		if got := tonemapMode("", n); got != n {
			t.Errorf("number %d: got %d", n, got)
		}
	}
	if tonemapMode("", 9) != 0 {
		t.Error("an unknown number should fall back")
	}
	for _, f := range []string{"tonemapHill", "tonemapAgX", "uTonemap == 4", "uTonemap == 5"} {
		if !containsStr(mbpostFragment, f) {
			t.Errorf("post shader lacks %s", f)
		}
	}
}

func TestBloomChainIsGLSL330(t *testing.T) {
	if containsVersion45(mbpostFragment) || containsVersion45(mbBloomDownFragment) {
		t.Fatal("post shaders must stay GLSL 330")
	}
	src := mbpostFragment + mbBloomDownFragment
	for _, s := range []string{"uBloomChain", "uBloom0", "uThreshold", "#version 330 core"} {
		if !containsStr(src, s) {
			t.Fatalf("missing %s", s)
		}
	}
}

func TestSSAOCommands(t *testing.T) {
	w := New(".")
	w.ready = true

	v, err := w.Call("enablessao", []value.Value{value.Num(1)})
	if err != nil {
		t.Fatal(err)
	}
	if v.Num != 1 || !w.post.ssao || !w.post.on {
		t.Fatalf("expected SSAO enabled, got %v, post.ssao=%v", v, w.post.ssao)
	}

	_, err = w.Call("setssaoradius", []value.Value{value.Num(1.25)})
	if err != nil {
		t.Fatal(err)
	}
	if w.post.ssaoRadius != 1.25 {
		t.Fatalf("expected radius 1.25, got %f", w.post.ssaoRadius)
	}

	_, err = w.Call("setssaointensity", []value.Value{value.Num(2.5)})
	if err != nil {
		t.Fatal(err)
	}
	if w.post.ssaoIntensity != 2.5 {
		t.Fatalf("expected intensity 2.5, got %f", w.post.ssaoIntensity)
	}

	v, err = w.Call("getssao", nil)
	if err != nil || v.Num != 1 {
		t.Fatalf("getssao expected 1, got %v, err=%v", v, err)
	}

	_, err = w.Call("setssao", []value.Value{value.Num(0)})
	if err != nil {
		t.Fatal(err)
	}
	if w.post.ssao {
		t.Fatal("expected SSAO disabled")
	}
}

// gl.PtrOffset(n) makes a Go pointer of value n. If the stack grows while one is live,
// Go stops with "invalid pointer found on stack": a crash about once in thirty launches of
// examples/island_orbit.mb. Offsets go in as uintptr, through the WithOffset calls.
func TestNoFakePointerOffsets(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	re := regexp.MustCompile(`PtrOffset\(([^)]*)\)`)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(src), -1) {
			if strings.TrimSpace(m[1]) != "0" {
				t.Errorf("%s: gl.PtrOffset(%s) can crash the garbage collector; use a WithOffset call", f, m[1])
			}
		}
	}
}
