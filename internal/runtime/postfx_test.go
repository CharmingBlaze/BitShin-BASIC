package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
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

func TestBloomChainIsGLSL330(t *testing.T) {
	if containsVersion45(mbpostFragment) || containsVersion45(mbBloomDownFragment) {
		t.Fatal("post shaders must stay GLSL 330")
	}
	src := mbpostFragment + mbBloomDownFragment
	for _, s := range []string{"uBloomChain", "uBloom0", "uBloom4", "uThreshold", "#version 330 core"} {
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
