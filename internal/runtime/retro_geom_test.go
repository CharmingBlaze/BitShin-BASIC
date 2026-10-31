package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/g3n/engine/renderer/shaders"

	"moonbasic/internal/value"
)

func TestRetroGeomCommandsSetState(t *testing.T) {
	w := New(".")
	call := func(name string, args ...float64) {
		t.Helper()
		vs := make([]value.Value, len(args))
		for i, a := range args {
			vs[i] = value.Num(a)
		}
		if _, err := w.Call(name, vs); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if sw, sh, af, fl, fs := w.retroGeomState(); sw != 0 || sh != 0 || af != 0 || fl || fs != 0 {
		t.Fatalf("retro geometry must start off: %v %v %v %v %v", sw, sh, af, fl, fs)
	}
	call("setretrosnap", 320, 240)
	call("setretroaffine", 0.75)
	call("setretroflat", 1)
	call("setretrofogsteps", 6)
	sw, sh, af, fl, fs := w.retroGeomState()
	if sw != 320 || sh != 240 || af != 0.75 || !fl || fs != 6 {
		t.Fatalf("state %v %v %v %v %v", sw, sh, af, fl, fs)
	}
	// One argument: rows follow the window aspect.
	w.scrW, w.scrH = 800, 600
	call("setretrosnap", 160)
	if sw, sh, _, _, _ := w.retroGeomState(); sw != 160 || sh != 0 {
		t.Fatalf("SetRetroSnap 160 stored %v x %v", sw, sh)
	}
	if rw, rh := w.retroSnapRes(); rw != 160 || rh != 120 {
		t.Fatalf("SetRetroSnap 160 on 800x600 snaps to %v x %v, want 160 x 120", rw, rh)
	}
	// Clamps and switches.
	call("setretroaffine", 3)
	call("setretrofogsteps", -2)
	call("setretroflat", 0)
	call("setretrosnap", 0, 0)
	sw, sh, af, fl, fs = w.retroGeomState()
	if sw != 0 || sh != 0 || af != 1 || fl || fs != 0 {
		t.Fatalf("after clamps/off: %v %v %v %v %v", sw, sh, af, fl, fs)
	}
	if rw, rh := w.retroSnapRes(); rw != 0 || rh != 0 {
		t.Fatalf("snap off resolves to %v x %v", rw, rh)
	}
	w.setRetroGeom(256, 224, 0.5, true, 4)
	if sw, sh, af, fl, fs := w.retroGeomState(); sw != 256 || sh != 224 || af != 0.5 || !fl || fs != 4 {
		t.Fatalf("setRetroGeom round trip: %v %v %v %v %v", sw, sh, af, fl, fs)
	}
}

func TestRetroGeomInShaders(t *testing.T) {
	type want struct {
		name, src string
		needs     []string
	}
	for _, c := range []want{
		{"bsshadow vert", mbshadowVertex, []string{"uniform vec2 RetroSnap;", "gl_Position = retroSnapClip(", "noperspective out vec2 FragTexcoordAffine;", "noperspective out vec3 WorldPosAffine;"}},
		{"bsshadow frag", mbshadowFragment, []string{"noperspective in vec2 FragTexcoordAffine;", "retroUV()", "retroFlatNormal(Position.xyz", "retroFlatNormal(WorldPos", "retroFog(clamp(f"}},
		{"mbphysical vert", mbphysicalVertex, []string{"uniform vec2 RetroSnap;", "gl_Position = retroSnapClip(", "noperspective out vec2 FragTexcoordAffine;"}},
		{"mbphysical frag", mbphysicalFragment, []string{"noperspective in vec2 FragTexcoordAffine;", "texture(uBaseColorSampler, baseUV)", "retroFlatNormal(Position", "retroFlatNormal(WorldPos", "retroFog(clamp(f"}},
		{"mbterrain frag", mbterrainFragment, []string{"noperspective in vec3 WorldPosAffine;", "retroTexPos()", "retroFlatNormal(WorldPos", "retroFlatNormal(Position.xyz", "retroFog(clamp(f"}},
		{"mbwater vert", mbwaterVertex, []string{"gl_Position = retroSnapClip(ClipSpace)"}},
		{"mbwater frag", mbwaterFragment, []string{"uniform int RetroFogSteps;", "retroFog(clamp(f"}},
	} {
		if containsVersion45(c.src) {
			t.Errorf("%s must stay GLSL 330", c.name)
		}
		for _, n := range c.needs {
			if !strings.Contains(c.src, n) {
				t.Errorf("%s lacks %q", c.name, n)
			}
		}
	}
	// The shadow-depth pass draws with its own program and is never snapped.
	for _, src := range []string{depthVertexSrc, depthFullVertexSrc, depthAlphaVertexSrc} {
		if strings.Contains(src, "retroSnap") || strings.Contains(src, "RetroSnap") {
			t.Error("the shadow-depth pass must not be snapped")
		}
	}
	// Every effect is gated off at zero.
	for _, gate := range []string{
		"RetroSnap.x < 1.0 || RetroSnap.y < 1.0",
		"if (RetroAffine <= 0.0) { return FragTexcoord; }",
		"if (RetroFlat == 0) { return n; }",
		"if (RetroFogSteps <= 0) { return f; }",
	} {
		if !strings.Contains(retroSnapGLSL+retroFragGLSL, gate) {
			t.Errorf("retro GLSL lacks the off gate %q", gate)
		}
	}
	m := New(".").commandTable()
	for _, name := range []string{"setretrosnap", "setretroaffine", "setretroflat", "setretrofogsteps"} {
		if m[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}
}

// expandG3N runs a lit shader through the same #include expansion G3N's
// shaman does, with a typical light / texture count.
func expandG3N(t *testing.T, src string, defines map[string]string) string {
	t.Helper()
	rex := regexp.MustCompile(`#include\s+<(.*)>\s*(?:\[(.*)]|)`)
	var proc func(s string) string
	proc = func(s string) string {
		for _, m := range rex.FindAllStringSubmatch(s, 100) {
			inc := shaders.IncludeSource(m[1])
			if inc == "" {
				t.Fatalf("include %s not found", m[1])
			}
			inc = "\n" + proc(inc)
			if m[2] != "" {
				q, _ := strconv.Atoi(defines[m[2]])
				rep := ""
				for i := 0; i < q; i++ {
					rep += strings.ReplaceAll(inc, "{i}", strconv.Itoa(i))
				}
				inc = rep
			}
			s = strings.ReplaceAll(s, m[0], inc)
		}
		return s
	}
	pre := "#version 330 core\n"
	for k, v := range defines {
		pre += fmt.Sprintf("#define %s %s\n", k, v)
	}
	return proc(pre + src)
}

// With glslangValidator on PATH, compile the retro-touched programs as
// OpenGL 3.3 (no -V), with and without textures.
func TestRetroGeomShadersCompile(t *testing.T) {
	exe, err := exec.LookPath("glslangValidator")
	if err != nil {
		t.Skip("glslangValidator not on PATH")
	}
	dir := t.TempDir()
	for _, tex := range []string{"0", "1"} {
		defs := map[string]string{"AMB_LIGHTS": "1", "DIR_LIGHTS": "1", "POINT_LIGHTS": "1", "SPOT_LIGHTS": "1", "MAT_TEXTURES": tex}
		pbrDefs := map[string]string{"AMB_LIGHTS": "1", "DIR_LIGHTS": "1", "POINT_LIGHTS": "1", "SPOT_LIGHTS": "1", "MAT_TEXTURES": "0"}
		if tex == "1" {
			pbrDefs["HAS_BASECOLORMAP"] = ""
			pbrDefs["HAS_NORMALMAP"] = ""
		}
		for name, src := range map[string]string{
			"bsshadow.vert": expandG3N(t, mbshadowVertex, defs),
			"bsshadow.frag": expandG3N(t, mbshadowFragment, defs),
			"terrain.frag":  expandG3N(t, mbterrainFragment, defs),
			"physical.vert": expandG3N(t, mbphysicalVertex, pbrDefs),
			"physical.frag": expandG3N(t, mbphysicalFragment, pbrDefs),
			"water.vert":    expandG3N(t, mbwaterVertex, defs),
			"water.frag":    expandG3N(t, mbwaterFragment, defs),
		} {
			path := filepath.Join(dir, "t"+tex+"_"+name)
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(exe, path).CombinedOutput(); err != nil {
				t.Errorf("%s (MAT_TEXTURES %s): %v\n%s", name, tex, err, out)
			}
		}
	}
}
